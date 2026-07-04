package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"aipsa-backend/internal/db"
	"aipsa-backend/pkg/response"
)

type APIKeyService struct {
	queries *db.Queries
}

func NewAPIKeyService(queries *db.Queries) *APIKeyService {
	return &APIKeyService{queries: queries}
}

func (s *APIKeyService) List(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	keys, err := s.queries.ListAPIKeysByUser(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch API keys")
	}

	return response.Success(c, keys)
}

func (s *APIKeyService) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req struct {
		Name   string   `json:"name"`
		Permissions []string `json:"permissions"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Name == "" {
		return response.Error(c, fiber.StatusBadRequest, "Name is required")
	}

	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read"}
	}

	apiKey, keyHash, keyPrefix, err := generateAPIKey()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate API key")
	}

	key, err := s.queries.CreateAPIKey(c.Context(), db.CreateAPIKeyParams{
		UserID:      userID,
		Name:        req.Name,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Permissions: req.Permissions,
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to create API key")
	}

	return response.Created(c, fiber.Map{
		"id":         key.ID,
		"name":       key.Name,
		"api_key":    apiKey,
		"key_prefix": keyPrefix,
		"permissions": key.Permissions,
		"created_at": key.CreatedAt,
	})
}

func (s *APIKeyService) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	keyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid API key ID")
	}

	if err := s.queries.DeleteAPIKey(c.Context(), db.DeleteAPIKeyParams{
		ID:     keyID,
		UserID: userID,
	}); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to delete API key")
	}

	return response.Success(c, fiber.Map{"message": "API key deleted"})
}

func generateAPIKey() (apiKey string, keyHash string, keyPrefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}

	apiKey = "ak_" + hex.EncodeToString(b)
	
	hash := sha256.Sum256([]byte(apiKey))
	keyHash = hex.EncodeToString(hash[:])
	
	keyPrefix = apiKey[:12]
	
	return apiKey, keyHash, keyPrefix, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func stringsJoin(arr []string, sep string) string {
	return strings.Join(arr, sep)
}
