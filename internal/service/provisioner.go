package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"

	"aipsa-backend/internal/config"
	"aipsa-backend/internal/db"
	"aipsa-backend/internal/security"
)

type Provisioner struct {
	superuserPool *pgxpool.Pool
	queries       *db.Queries
	cfg           *config.Config
	encryptor     *security.Encryptor
	logger        *slog.Logger
}

func NewProvisioner(superuserPool *pgxpool.Pool, queries *db.Queries, cfg *config.Config, encryptor *security.Encryptor, logger *slog.Logger) *Provisioner {
	return &Provisioner{
		superuserPool: superuserPool,
		queries:       queries,
		cfg:           cfg,
		encryptor:     encryptor,
		logger:        logger,
	}
}

func (p *Provisioner) Provision(ctx context.Context, project *db.Project) error {
	dbName := fmt.Sprintf("proj_%s", project.ID.String()[:8])
	dbUser := fmt.Sprintf("user_%s", project.ID.String()[:8])
	dbPassword := generatePassword()

	p.logger.Info("provisioning database", "project_id", project.ID, "db_name", dbName)

	createDBQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
	if _, err := p.superuserPool.Exec(ctx, createDBQuery); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	createRoleQuery := fmt.Sprintf(`CREATE ROLE "%s" WITH LOGIN PASSWORD '%s'`, dbUser, dbPassword)
	if _, err := p.superuserPool.Exec(ctx, createRoleQuery); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	grantQuery := fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE "%s" TO "%s"`, dbName, dbUser)
	if _, err := p.superuserPool.Exec(ctx, grantQuery); err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"postgres", "postgres",
		"localhost", "5432", dbName)

	newDBPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		p.logger.Warn("failed to connect to new database for schema setup", "error", err)
	} else {
		defer newDBPool.Close()

		grantSchemaQuery := fmt.Sprintf(`GRANT ALL ON SCHEMA public TO "%s"`, dbUser)
		newDBPool.Exec(ctx, grantSchemaQuery)

		extensions := []string{"uuid-ossp", "pgcrypto", "citext", "pg_trgm"}
		for _, ext := range extensions {
			extQuery := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s"`, ext)
			newDBPool.Exec(ctx, extQuery)
		}
	}

	encryptedPassword, err := p.encryptor.Encrypt(dbPassword)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	_, err = p.queries.UpdateProjectDatabase(ctx, db.UpdateProjectDatabaseParams{
		ID:                   project.ID,
		DbName:               dbName,
		DbHost:               "localhost",
		DbPort:               5432,
		DbUser:               dbUser,
		DbPasswordEncrypted: encryptedPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	p.logger.Info("database provisioned successfully", "project_id", project.ID, "db_name", dbName)
	return nil
}

func (p *Provisioner) Deprovision(ctx context.Context, projectID uuid.UUID) error {
	project, err := p.queries.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if project.DbName == "" {
		return nil
	}

	p.logger.Info("deprovisioning database", "project_id", projectID, "db_name", project.DbName)

	terminateQuery := fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'`, project.DbName)
	p.superuserPool.Exec(ctx, terminateQuery)

	dropDBQuery := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, project.DbName)
	if _, err := p.superuserPool.Exec(ctx, dropDBQuery); err != nil {
		p.logger.Error("failed to drop database", "error", err)
	}

	if project.DbUser != "" {
		dropRoleQuery := fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, project.DbUser)
		p.superuserPool.Exec(ctx, dropRoleQuery)
	}

	p.logger.Info("database deprovisioned successfully", "project_id", projectID)
	return nil
}

func (p *Provisioner) GetConnectionString(ctx context.Context, projectID uuid.UUID) (string, error) {
	project, err := p.queries.GetProject(ctx, projectID)
	if err != nil {
		return "", err
	}

	if project.DbName == "" {
		return "", fmt.Errorf("database not provisioned yet")
	}

	password, err := p.encryptor.Decrypt(project.DbPasswordEncrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		project.DbUser, password, project.DbHost, project.DbPort, project.DbName), nil
}

func generatePassword() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
