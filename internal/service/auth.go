package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"aipsa-backend/internal/config"
	"aipsa-backend/internal/db"
	"aipsa-backend/internal/security"
	"aipsa-backend/pkg/jwt"
	"aipsa-backend/pkg/response"
)

type AuthService struct {
	queries *db.Queries
	jwt     *jwt.JWTManager
	cfg     *config.Config
	logger  *slog.Logger
}

func NewAuthService(queries *db.Queries, jwtManager *jwt.JWTManager, cfg *config.Config, logger *slog.Logger) *AuthService {
	return &AuthService{
		queries: queries,
		jwt:     jwtManager,
		cfg:     cfg,
		logger:  logger,
	}
}

func (s *AuthService) Register(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return response.Error(c, fiber.StatusBadRequest, "Email, password, and full name are required")
	}

	hashedPassword := security.HashPassword(req.Password, nil)

	user, err := s.queries.CreateUser(c.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: &hashedPassword,
		FullName:     req.FullName,
		Provider:     "email",
	})
	if err != nil {
		s.logger.Error("failed to create user", "error", err)
		return response.Error(c, fiber.StatusConflict, "Email already exists")
	}

	// Create default organization for user
	orgSlug := generateSlug(req.FullName)
	org, err := s.queries.CreateOrganization(c.Context(), db.CreateOrganizationParams{
		Name:    req.FullName + "'s Organization",
		Slug:    orgSlug,
		OwnerID: user.ID,
	})
	if err != nil {
		s.logger.Error("failed to create organization", "error", err)
	} else {
		// Add user as owner
		s.queries.AddOrganizationMember(c.Context(), db.AddOrganizationMemberParams{
			OrganizationID: org.ID,
			UserID:         user.ID,
			Role:           "owner",
		})
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	hashedToken := hashToken(refreshToken)
	userAgent := c.Get("User-Agent")
	ipAddress := c.IP()
	_, err = s.queries.CreateSession(c.Context(), db.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hashedToken,
		UserAgent:        &userAgent,
		IpAddress:        &ipAddress,
		ExpiresAt:        time.Now().Add(s.cfg.RefreshExpiration),
	})
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
	}

	return response.Created(c, fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(s.cfg.JWTExpiration.Seconds()),
		"user": fiber.Map{
			"id":        user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
		},
	})
}

func (s *AuthService) Login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return response.Error(c, fiber.StatusBadRequest, "Email and password are required")
	}

	user, err := s.queries.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	if user.PasswordHash == nil {
		return response.Error(c, fiber.StatusUnauthorized, "Password not set for this account")
	}

	if !security.VerifyPassword(req.Password, *user.PasswordHash) {
		return response.Error(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	hashedToken := hashToken(refreshToken)
	userAgent := c.Get("User-Agent")
	ipAddress := c.IP()
	_, err = s.queries.CreateSession(c.Context(), db.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: hashedToken,
		UserAgent:        &userAgent,
		IpAddress:        &ipAddress,
		ExpiresAt:        time.Now().Add(s.cfg.RefreshExpiration),
	})
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
	}

	return response.Success(c, fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(s.cfg.JWTExpiration.Seconds()),
		"user": fiber.Map{
			"id":        user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
		},
	})
}

func (s *AuthService) RefreshToken(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.RefreshToken == "" {
		return response.Error(c, fiber.StatusBadRequest, "Refresh token is required")
	}

	claims, err := s.jwt.ValidateToken(req.RefreshToken)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "Invalid or expired refresh token")
	}

	if claims.TokenType != "refresh" {
		return response.Error(c, fiber.StatusUnauthorized, "Invalid token type")
	}

	hashedToken := hashToken(req.RefreshToken)
	session, err := s.queries.GetSessionByToken(c.Context(), hashedToken)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "Session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		s.queries.DeleteSession(c.Context(), session.ID)
		return response.Error(c, fiber.StatusUnauthorized, "Session expired")
	}

	accessToken, err := s.jwt.GenerateAccessToken(claims.UserID, claims.Email)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	newRefreshToken, err := s.jwt.GenerateRefreshToken(claims.UserID, claims.Email)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	s.queries.DeleteSession(c.Context(), session.ID)

	newHashedToken := hashToken(newRefreshToken)
	newUserAgent := c.Get("User-Agent")
	newIPAddress := c.IP()
	_, err = s.queries.CreateSession(c.Context(), db.CreateSessionParams{
		UserID:           claims.UserID,
		RefreshTokenHash: newHashedToken,
		UserAgent:        &newUserAgent,
		IpAddress:        &newIPAddress,
		ExpiresAt:        time.Now().Add(s.cfg.RefreshExpiration),
	})
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
	}

	return response.Success(c, fiber.Map{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    int(s.cfg.JWTExpiration.Seconds()),
	})
}

func (s *AuthService) Logout(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if req.RefreshToken != "" {
		hashedToken := hashToken(req.RefreshToken)
		session, err := s.queries.GetSessionByToken(c.Context(), hashedToken)
		if err == nil {
			s.queries.DeleteSession(c.Context(), session.ID)
		}
	}

	return response.Success(c, fiber.Map{"message": "Logged out successfully"})
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32)
		} else if c == ' ' || c == '-' {
			slug += "-"
		}
	}
	if len(slug) > 50 {
		slug = slug[:50]
	}
	b := make([]byte, 4)
	rand.Read(b)
	return slug + "-" + hex.EncodeToString(b)
}
