package service

import (
	"github.com/gofiber/fiber/v3"

	"aipsa-backend/pkg/response"
)

type HealthService struct{}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) Handler(c fiber.Ctx) error {
	return response.Success(c, fiber.Map{
		"status": "healthy",
		"service": "aipsa-backend",
	})
}
