package middleware

import (
	"github.com/gofiber/fiber/v3"

	"aipsa-backend/internal/config"
	"aipsa-backend/pkg/response"
)

func InternalAuth(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		apiKey := c.Get("X-Internal-Key")
		if apiKey == "" {
			return response.Error(c, fiber.StatusUnauthorized, "Missing internal API key")
		}

		if apiKey != cfg.InternalAPIKey {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid internal API key")
		}

		return c.Next()
	}
}
