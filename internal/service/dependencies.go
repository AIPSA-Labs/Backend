package service

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"aipsa-backend/internal/config"
	"aipsa-backend/internal/db"
	"aipsa-backend/internal/security"
	"aipsa-backend/pkg/jwt"
)

type Dependencies struct {
	Auth     *AuthService
	Users    *UserService
	Projects *ProjectService
	Database *DatabaseService
	APIKeys  *APIKeyService
	Health   *HealthService
}

func NewDependencies(cfg *config.Config, logger *slog.Logger) (*Dependencies, error) {
	pool, err := pgxpool.New(nil, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return nil, err
	}

	queries := db.New(pool)

	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTExpiration, cfg.RefreshExpiration)
	encryptor := security.NewEncryptor(cfg.EncryptionKey)

	provisioner := NewProvisioner(pool, queries, cfg, encryptor, logger)

	return &Dependencies{
		Auth:     NewAuthService(queries, jwtManager, cfg, logger),
		Users:    NewUserService(queries, logger),
		Projects: NewProjectService(queries, provisioner, logger),
		Database: NewDatabaseService(queries, provisioner),
		APIKeys:  NewAPIKeyService(queries),
		Health:   NewHealthService(),
	}, nil
}
