package service

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"aipsa-backend/internal/db"
	"aipsa-backend/pkg/response"
)

type DatabaseService struct {
	queries     *db.Queries
	provisioner *Provisioner
}

func NewDatabaseService(queries *db.Queries, provisioner *Provisioner) *DatabaseService {
	return &DatabaseService{
		queries:     queries,
		provisioner: provisioner,
	}
}

func (s *DatabaseService) GetInfo(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	project, err := s.queries.GetProject(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Project not found")
	}

	if project.DbName == "" {
		return response.Success(c, fiber.Map{
			"status": "provisioning",
			"message": "Database is being provisioned",
		})
	}

	connStr, err := s.provisioner.GetConnectionString(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get connection info")
	}

	return response.Success(c, fiber.Map{
		"status":          "ready",
		"db_name":         project.DbName,
		"db_host":         project.DbHost,
		"db_port":         project.DbPort,
		"db_user":         project.DbUser,
		"db_ssl_mode":     project.DbSslMode,
		"connection_string": connStr,
	})
}

func (s *DatabaseService) ResetPassword(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	project, err := s.queries.GetProject(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Project not found")
	}

	if project.DbName == "" {
		return response.Error(c, fiber.StatusBadRequest, "Database not provisioned yet")
	}

	newPassword := generatePassword()

	connStr := fmt.Sprintf("postgres://postgres:postgres@%s:%d/postgres?sslmode=disable",
		project.DbHost, 5432)

	pool, err := pgxpool.New(c.Context(), connStr)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to connect to database")
	}
	defer pool.Close()

	alterQuery := fmt.Sprintf(`ALTER ROLE "%s" WITH PASSWORD '%s'`, project.DbUser, newPassword)
	if _, err := pool.Exec(c.Context(), alterQuery); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to reset password")
	}

	encryptedPassword, err := s.provisioner.encryptor.Encrypt(newPassword)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to encrypt password")
	}

	_, err = s.queries.UpdateProjectDatabase(c.Context(), db.UpdateProjectDatabaseParams{
		ID:                   project.ID,
		DbName:               project.DbName,
		DbHost:               project.DbHost,
		DbPort:               project.DbPort,
		DbUser:               project.DbUser,
		DbPasswordEncrypted: encryptedPassword,
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update project")
	}

	return response.Success(c, fiber.Map{
		"message": "Password reset successfully",
		"new_password": newPassword,
	})
}

func (s *DatabaseService) ExecuteSQL(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	var req struct {
		Query string `json:"query"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Query == "" {
		return response.Error(c, fiber.StatusBadRequest, "Query is required")
	}

	project, err := s.queries.GetProject(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Project not found")
	}

	if project.DbName == "" {
		return response.Error(c, fiber.StatusBadRequest, "Database not provisioned yet")
	}

	connStr, err := s.provisioner.GetConnectionString(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get connection info")
	}

	pool, err := pgxpool.New(c.Context(), connStr)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to connect to database")
	}
	defer pool.Close()

	start := time.Now()

	rows, err := pool.Query(c.Context(), req.Query)
	if err != nil {
		duration := int32(time.Since(start).Milliseconds())
		userID, _ := c.Locals("user_id").(uuid.UUID)

		s.queries.CreateSQLHistory(c.Context(), db.CreateSQLHistoryParams{
			ProjectID:     projectID,
			UserID:        userID,
			Query:         req.Query,
			DurationMs:    &duration,
			ErrorMessage:  strPtr(err.Error()),
			IsReadOnly:    false,
		})

		return response.Error(c, fiber.StatusBadRequest, fmt.Sprintf("Query error: %s", err.Error()))
	}
	defer rows.Close()

	var results []map[string]interface{}
	columns := rows.FieldDescriptions()
	duration := int32(time.Since(start).Milliseconds())

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col.Name] = values[i]
		}
		results = append(results, row)
	}

	rowsAffected := int64(len(results))
	userID, _ := c.Locals("user_id").(uuid.UUID)

	s.queries.CreateSQLHistory(c.Context(), db.CreateSQLHistoryParams{
		ProjectID:     projectID,
		UserID:        userID,
		Query:         req.Query,
		DurationMs:    &duration,
		RowsAffected:  &rowsAffected,
		IsReadOnly:    true,
	})

	return response.Success(c, fiber.Map{
		"results":      results,
		"rows_affected": rowsAffected,
		"duration_ms":  duration,
	})
}

func (s *DatabaseService) GetSQLHistory(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	page := 1
	perPage := 20

	history, err := s.queries.ListSQLHistoryByProject(c.Context(), db.ListSQLHistoryByProjectParams{
		ProjectID: projectID,
		Limit:     int32(perPage),
		Offset:    int32((page - 1) * perPage),
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch history")
	}

	return response.Success(c, history)
}

func (s *DatabaseService) Provision(c fiber.Ctx) error {
	var req struct {
		ProjectID string `json:"project_id"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	project, err := s.queries.GetProject(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Project not found")
	}

	if err := s.provisioner.Provision(c.Context(), &project); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to provision database")
	}

	return response.Success(c, fiber.Map{"message": "Database provisioned"})
}

func (s *DatabaseService) DeleteProject(c fiber.Ctx) error {
	var req struct {
		ProjectID string `json:"project_id"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	if err := s.provisioner.Deprovision(c.Context(), projectID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to deprovision database")
	}

	return response.Success(c, fiber.Map{"message": "Database deleted"})
}
