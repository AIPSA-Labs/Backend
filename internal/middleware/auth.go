package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"aipsa-backend/internal/config"
	"aipsa-backend/pkg/jwt"
	"aipsa-backend/pkg/response"
)

func Auth(cfg *config.Config) fiber.Handler {
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTExpiration, cfg.RefreshExpiration)

	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Error(c, fiber.StatusUnauthorized, "Missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid authorization format")
		}

		claims, err := jwtManager.ValidateToken(parts[1])
		if err != nil {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid or expired token")
		}

		if claims.TokenType != "access" {
			return response.Error(c, fiber.StatusUnauthorized, "Invalid token type")
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)

		return c.Next()
	}
}

func GetUserID(c fiber.Ctx) (uuid.UUID, bool) {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	return userID, ok
}
