package service

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"aipsa-backend/internal/db"
	"aipsa-backend/pkg/response"
)

type UserService struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewUserService(queries *db.Queries, logger *slog.Logger) *UserService {
	return &UserService{
		queries: queries,
		logger:  logger,
	}
}

func (s *UserService) GetMe(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	user, err := s.queries.GetUserByID(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "User not found")
	}

	return response.Success(c, fiber.Map{
		"id":        user.ID,
		"email":     user.Email,
		"full_name": user.FullName,
		"avatar_url": user.AvatarUrl,
		"provider":  user.Provider,
		"created_at": user.CreatedAt,
	})
}

func (s *UserService) UpdateMe(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req struct {
		FullName  string `json:"full_name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	var fullName *string
	var avatarURL *string

	if req.FullName != "" {
		fullName = &req.FullName
	}
	if req.AvatarURL != "" {
		avatarURL = &req.AvatarURL
	}

	user, err := s.queries.UpdateUser(c.Context(), db.UpdateUserParams{
		ID:         userID,
		FullName:   fullName,
		AvatarUrl:  avatarURL,
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update user")
	}

	return response.Success(c, fiber.Map{
		"id":        user.ID,
		"email":     user.Email,
		"full_name": user.FullName,
		"avatar_url": user.AvatarUrl,
	})
}
