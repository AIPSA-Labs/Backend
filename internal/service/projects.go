package service

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"aipsa-backend/internal/db"
	"aipsa-backend/pkg/response"
)

type ProjectService struct {
	queries     *db.Queries
	provisioner *Provisioner
	logger      *slog.Logger
}

func NewProjectService(queries *db.Queries, provisioner *Provisioner, logger *slog.Logger) *ProjectService {
	return &ProjectService{
		queries:     queries,
		provisioner: provisioner,
		logger:      logger,
	}
}

func (s *ProjectService) List(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	projects, err := s.queries.ListProjectsByUser(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch projects")
	}

	return response.Success(c, projects)
}

func (s *ProjectService) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" {
		return response.Error(c, fiber.StatusBadRequest, "Name is required")
	}

	org, err := s.queries.GetOrganizationByUser(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Organization not found. Please register first.")
	}

	slug := generateSlug(req.Name)

	project, err := s.queries.CreateProject(c.Context(), db.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           req.Name,
		Slug:           slug,
		Description:    &req.Description,
		Status:         "active",
	})
	if err != nil {
		s.logger.Error("failed to create project", "error", err)
		return response.Error(c, fiber.StatusInternalServerError, "Failed to create project")
	}

	go func() {
		if err := s.provisioner.Provision(c.Context(), &project); err != nil {
			s.logger.Error("failed to provision database", "project_id", project.ID, "error", err)
		}
	}()

	return response.Created(c, project)
}

func (s *ProjectService) Get(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	project, err := s.queries.GetProject(c.Context(), projectID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Project not found")
	}

	return response.Success(c, project)
}

func (s *ProjectService) Update(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	var name *string
	var description *string

	if req.Name != "" {
		name = &req.Name
	}
	if req.Description != "" {
		description = &req.Description
	}

	project, err := s.queries.UpdateProject(c.Context(), db.UpdateProjectParams{
		ID:          projectID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update project")
	}

	return response.Success(c, project)
}

func (s *ProjectService) Delete(c fiber.Ctx) error {
	projectID, err := uuid.Parse(c.Params("projectId"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid project ID")
	}

	go func() {
		if err := s.provisioner.Deprovision(c.Context(), projectID); err != nil {
			s.logger.Error("failed to deprovision database", "project_id", projectID, "error", err)
		}
	}()

	if err := s.queries.DeleteProject(c.Context(), projectID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to delete project")
	}

	return response.Success(c, fiber.Map{"message": "Project deleted"})
}
