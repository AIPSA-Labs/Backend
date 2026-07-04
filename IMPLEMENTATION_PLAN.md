# AIPSA Backend - Implementation Plan

## Project Overview
A production-ready minimal PostgreSQL Database Platform built with Go (Fiber v3) and PostgreSQL, designed as an internal tool for managing databases within an organization.

## Tech Stack
- **Language**: Go 1.25+
- **Web Framework**: Fiber v3
- **Database**: PostgreSQL 17+
- **Driver**: pgx (no ORM)
- **Code Generation**: sqlc
- **Migrations**: Goose
- **Authentication**: JWT + Google OAuth
- **Logging**: slog (structured logging)
- **Metrics**: Prometheus
- **Containerization**: Docker + Docker Compose

## Architecture Pattern
Clean Architecture with Repository Pattern:
```
Handler (HTTP) → Service (Business Logic) → Repository (Data Access) → sqlc (Generated Code)
```

---

## Phase 1: Project Foundation

### 1.1 Project Structure
```
cmd/
    server/
        main.go                 # Application entry point
internal/
    config/
        config.go              # Configuration loading
    auth/
        handler.go             # HTTP handlers
        service.go             # Business logic
        repository.go          # Data access
        models.go              # Domain models
    users/
        handler.go
        service.go
        repository.go
        models.go
    projects/
        handler.go
        service.go
        repository.go
        models.go
    database/
        handler.go
        service.go             # Provisioning logic
        repository.go
        provisioner.go         # PostgreSQL provisioning
    sql/
        handler.go
        service.go
        repository.go
    schema/
        handler.go
        service.go
    metrics/
        handler.go
        service.go
    apikeys/
        handler.go
        service.go
        repository.go
    middleware/
        auth.go                # JWT middleware
        internal.go            # Internal service auth
        ratelimit.go           # Rate limiting
        cors.go                # CORS
        requestid.go           # Request ID
        logger.go              # Request logging
    logger/
        logger.go              # Structured logging setup
    security/
        crypto.go              # Encryption, hashing
        password.go            # Argon2id hashing
pkg/
    jwt/
        jwt.go                 # JWT generation/validation
    response/
        response.go            # Standard API responses
    pagination/
        pagination.go          # Pagination utilities
    validator/
        validator.go           # Request validation
migrations/
    001_initial_schema.sql
    002_add_extensions.sql
sql/
    queries/
        users.sql
        projects.sql
        databases.sql
        apikeys.sql
        sessions.sql
        sql_history.sql
    schema.sql                 # Full schema for sqlc
sqlc.yaml                     # sqlc configuration
docker-compose.yml
Dockerfile
Makefile
.env.example
```

### 1.2 Configuration Management (Zero-Config Default)

**File**: `internal/config/config.go`

```go
package config

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "os"
    "strconv"
    "time"
)

type Config struct {
    // Server
    ServerPort  string
    ServerHost  string
    Environment string
    
    // Database
    DatabaseURL string
    
    // JWT (auto-generated if not set)
    JWTSecret         string
    JWTExpiration     time.Duration
    RefreshExpiration time.Duration
    
    // Google OAuth (optional - disabled if empty)
    GoogleClientID     string
    GoogleClientSecret string
    GoogleRedirectURL  string
    
    // Internal API (auto-generated if not set)
    InternalAPIKey string
    
    // Security (auto-generated if not set)
    EncryptionKey string
    
    // CORS
    AllowedOrigins []string
}

func Load() *Config {
    cfg := &Config{
        ServerPort:       getEnv("SERVER_PORT", "8080"),
        ServerHost:       getEnv("SERVER_HOST", "0.0.0.0"),
        Environment:      getEnv("ENVIRONMENT", "development"),
        DatabaseURL:      getEnv("DATABASE_URL", "postgres://aipsa:aipsa_secret@localhost:5432/aipsa_platform?sslmode=disable"),
        JWTExpiration:    getDurationEnv("JWT_EXPIRATION", 15*time.Minute),
        RefreshExpiration: getDurationEnv("REFRESH_EXPIRATION", 168*time.Hour),
        GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/google/callback"),
        AllowedOrigins:     getListEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080"),
    }
    
    // Auto-generate secrets if not provided
    cfg.JWTSecret = getEnv("JWT_SECRET", generateRandomKey(32))
    cfg.InternalAPIKey = getEnv("INTERNAL_API_KEY", generateRandomKey(32))
    cfg.EncryptionKey = getEnv("ENCRYPTION_KEY", generateRandomKey(32))
    
    return cfg
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    return defaultValue
}

func getListEnv(key, defaultValue string) []string {
    value := getEnv(key, defaultValue)
    if value == "" {
        return []string{}
    }
    return strings.Split(value, ",")
}

func generateRandomKey(length int) string {
    b := make([]byte, length)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

**Key Design Decisions:**
- **Zero required env vars** - everything has defaults
- **Auto-generated secrets** - JWT, encryption, API key generated on startup
- **Google OAuth optional** - disabled if client ID/secret not set
- **Single DATABASE_URL** - replaces 6 database variables
- **Works out of the box** - `docker compose up` is all you need

### 1.3 Database Schema (Initial Migration)

**File**: `migrations/001_initial_schema.sql`

```sql
-- +goose Up

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email CITEXT NOT NULL UNIQUE,
    password_hash VARCHAR(255), -- Nullable for OAuth users
    full_name VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),
    provider VARCHAR(50) NOT NULL DEFAULT 'email', -- email, google
    provider_id VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organizations (for multi-tenancy)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organization members
CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

-- Projects (each project = one PostgreSQL database)
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, suspended, deleted
    db_name VARCHAR(255) NOT NULL, -- PostgreSQL database name
    db_host VARCHAR(255) NOT NULL,
    db_port INTEGER NOT NULL DEFAULT 5432,
    db_user VARCHAR(255) NOT NULL,
    db_password_encrypted TEXT NOT NULL, -- Encrypted password
    db_ssl_mode VARCHAR(50) DEFAULT 'require',
    region VARCHAR(100) DEFAULT 'default',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, slug)
);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(10) NOT NULL, -- First 8 chars for identification
    permissions JSONB DEFAULT '["read"]',
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sessions (for refresh tokens)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent VARCHAR(500),
    ip_address INET,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- SQL Query History
CREATE TABLE sql_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    query TEXT NOT NULL,
    duration_ms INTEGER,
    rows_affected INTEGER,
    error_message TEXT,
    is_read_only BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Backups metadata (for future implementation)
CREATE TABLE backups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    size_bytes BIGINT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, completed, failed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Audit logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    organization_id UUID REFERENCES organizations(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    metadata JSONB,
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_provider ON users(provider, provider_id);
CREATE INDEX idx_projects_organization ON projects(organization_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(refresh_token_hash);
CREATE INDEX idx_sql_history_project ON sql_history(project_id);
CREATE INDEX idx_sql_history_user ON sql_history(user_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_organization ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS sql_history;
DROP TABLE IF EXISTS backups;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
```

### 1.4 sqlc Configuration

**File**: `sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries/"
    schema: "sql/schema.sql"
    gen:
      go:
        package: "db"
        sql_package: "pgx/v5"
        out: "internal/db"
        sql_driver: "pgx/v5/stdlib"
        emit_json_tags: true
        emit_db_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "jsonb"
            go_type: "json.RawMessage"
```

### 1.5 Makefile

**File**: `Makefile`

```makefile
.PHONY: all build run test migrate sqlc docker-up docker-down

# Variables
APP_NAME=aipsa-backend
DOCKER_COMPOSE=docker compose

# Default target
all: build

# Build the application
build:
	go build -o bin/$(APP_NAME) cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests
test-integration:
	go test -v -tags=integration ./...

# Database migrations
migrate-up:
	goose -dir migrations up

migrate-down:
	goose -dir migrations down

migrate-create:
	goose -dir migrations create $(name) sql

# Generate sqlc code
sqlc:
	sqlc generate

# Docker commands
docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Development
dev:
	air -c .air.toml

# Lint
lint:
	golangci-lint run

# Format
format:
	gofmt -s -w .
	goimports -w .

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Help
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback migrations"
	@echo "  sqlc           - Generate sqlc code"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  dev            - Run with hot reload"
	@echo "  lint           - Run linter"
	@echo "  format         - Format code"
	@echo "  clean          - Clean build artifacts"
```

### 1.6 Docker Compose (Zero-Config)

**File**: `docker-compose.yml`

```yaml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: aipsa-backend
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://aipsa:aipsa_secret@postgres:5432/aipsa_platform?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - aipsa-network

  postgres:
    image: postgres:17-alpine
    container_name: aipsa-postgres
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_USER=aipsa
      - POSTGRES_PASSWORD=aipsa_secret
      - POSTGRES_DB=aipsa_platform
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U aipsa -d aipsa_platform"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - aipsa-network

volumes:
  postgres_data:

networks:
  aipsa-network:
    driver: bridge
```

**That's it!** Just 1 environment variable. Everything else has sensible defaults:
- JWT secrets: auto-generated
- Encryption key: auto-generated
- Internal API key: auto-generated
- Google OAuth: disabled (add `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` to enable)
- CORS: allows localhost:3000 and localhost:8080
- Rate limiting: 100 requests/minute
- JWT expiration: 15 minutes access, 7 days refresh

### 1.8 Dockerfile

**File**: `Dockerfile`

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server cmd/server/main.go

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/server .
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Run the application
CMD ["./server"]
```

### 1.7 Environment Example (Optional Override)

**File**: `.env.example`

```bash
# All values have sensible defaults - only override what you need

# Database (only needed if using external PostgreSQL)
# DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable

# Google OAuth (optional - disabled if not set)
# GOOGLE_CLIENT_ID=your-google-client-id
# GOOGLE_CLIENT_SECRET=your-google-client-secret

# Production overrides (optional)
# ENVIRONMENT=production
# SERVER_PORT=8080
# JWT_EXPIRATION=15m
# REFRESH_EXPIRATION=168h
# ALLOWED_ORIGINS=https://yourdomain.com
```

**Start the platform:**
```bash
# Just run this - no configuration needed
docker compose up -d

# Or with optional pgAdmin
docker compose --profile tools up -d
```

**Enable Google OAuth (optional):**
```bash
# Add to docker-compose.yml environment section:
- GOOGLE_CLIENT_ID=your-client-id
- GOOGLE_CLIENT_SECRET=your-client-secret
```

---

## Phase 2: Core Implementation

### 2.1 Main Entry Point

**File**: `cmd/server/main.go`

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "aipsa-backend/internal/config"
    "aipsa-backend/internal/server"
)

func main() {
    // Load configuration
    cfg := config.Load()
    
    // Setup logger
    logger := setupLogger(cfg.Environment)
    slog.SetDefault(logger)
    
    // Create context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create and run server
    srv, err := server.New(cfg, logger)
    if err != nil {
        slog.Error("failed to create server", "error", err)
        os.Exit(1)
    }
    
    // Handle graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-quit
        slog.Info("shutting down server...")
        cancel()
        
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer shutdownCancel()
        
        if err := srv.Shutdown(shutdownCtx); err != nil {
            slog.Error("server forced to shutdown", "error", err)
        }
    }()
    
    // Start server
    if err := srv.Start(); err != nil {
        slog.Error("server failed to start", "error", err)
        os.Exit(1)
    }
}

func setupLogger(env string) *slog.Logger {
    var level slog.Level
    if env == "production" {
        level = slog.LevelInfo
    } else {
        level = slog.LevelDebug
    }
    
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: level,
    })
    
    return slog.New(handler)
}
```

### 2.2 Server Setup

**File**: `internal/server/server.go`

```go
package server

import (
    "context"
    "log/slog"
    
    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/cors"
    "github.com/gofiber/fiber/v3/middleware/helmet"
    "github.com/gofiber/fiber/v3/middleware/limiter"
    "github.com/gofiber/fiber/v3/middleware/recover"
    "github.com/gofiber/fiber/v3/middleware/requestid"
    
    "aipsa-backend/internal/config"
    "aipsa-backend/internal/handler"
    "aipsa-backend/internal/middleware"
    "aipsa-backend/internal/service"
    "aipsa-backend/pkg/response"
)

type Server struct {
    app    *fiber.App
    cfg    *config.Config
    logger *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
    // Create Fiber app
    app := fiber.New(fiber.Config{
        ErrorHandler: errorHandler,
    })
    
    // Setup middleware
    setupMiddleware(app, cfg)
    
    // Initialize services
    services, err := service.NewDependencies(cfg, logger)
    if err != nil {
        return nil, err
    }
    
    // Setup routes
    setupRoutes(app, services, cfg)
    
    return &Server{
        app:    app,
        cfg:    cfg,
        logger: logger,
    }, nil
}

func setupMiddleware(app *fiber.App, cfg *config.Config) {
    // Recovery middleware
    app.Use(recover.New())
    
    // Request ID
    app.Use(requestid.New())
    
    // CORS
    app.Use(cors.New(cors.Config{
        AllowOrigins: cfg.AllowedOrigins,
        AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Internal-Key"},
    }))
    
    // Security headers
    app.Use(helmet.New())
    
    // Rate limiting
    app.Use(limiter.New(limiter.Config{
        Max:        100,
        Expiration: 1 * time.Minute,
    }))
    
    // Request logging
    app.Use(middleware.Logger())
}

func setupRoutes(app *fiber.App, services *service.Dependencies, cfg *config.Config) {
    // Health check
    app.Get("/health", services.Health.Handler)
    app.Get("/internal/health", services.Health.Handler)
    
    // Public API v1
    v1 := app.Group("/v1")
    
    // Auth routes
    auth := v1.Group("/auth")
    auth.Post("/login", services.Auth.Login)
    auth.Post("/register", services.Auth.Register)
    auth.Post("/google", services.Auth.GoogleLogin)
    auth.Get("/google/callback", services.Auth.GoogleCallback)
    auth.Post("/logout", services.Auth.Logout)
    auth.Post("/refresh", services.Auth.RefreshToken)
    
    // Protected routes
    protected := v1.Group("", middleware.Auth(cfg))
    
    // User routes
    protected.Get("/me", services.Users.GetMe)
    protected.Patch("/me", services.Users.UpdateMe)
    
    // Projects routes
    projects := protected.Group("/projects")
    projects.Get("", services.Projects.List)
    projects.Post("", services.Projects.Create)
    projects.Get("/:projectId", services.Projects.Get)
    projects.Patch("/:projectId", services.Projects.Update)
    projects.Delete("/:projectId", services.Projects.Delete)
    
    // Database routes
    projects.Get("/:projectId/database", services.Database.GetInfo)
    projects.Post("/:projectId/database/reset-password", services.Database.ResetPassword)
    
    // SQL routes
    projects.Get("/:projectId/sql/history", services.SQL.GetHistory)
    projects.Post("/:projectId/sql/execute", services.SQL.Execute)
    
    // Schema routes
    projects.Get("/:projectId/schema", services.Schema.GetSchema)
    projects.Get("/:projectId/tables", services.Schema.ListTables)
    projects.Post("/:projectId/tables", services.Schema.CreateTable)
    
    // Metrics routes
    projects.Get("/:projectId/metrics", services.Metrics.GetMetrics)
    
    // API Keys routes
    keys := protected.Group("/keys")
    keys.Get("", services.APIKeys.List)
    keys.Post("", services.APIKeys.Create)
    keys.Delete("/:id", services.APIKeys.Delete)
    
    // Internal API
    internal := app.Group("/internal", middleware.InternalAuth(cfg))
    
    // Internal project routes
    internal.Post("/projects/provision", services.Database.Provision)
    internal.Post("/projects/delete", services.Database.DeleteProject)
    
    // Internal database routes
    internal.Post("/database/create", services.Database.CreateDatabase)
    internal.Post("/database/delete", services.Database.DeleteDatabase)
    internal.Post("/database/reset-password", services.Database.ResetPasswordInternal)
    internal.Post("/database/create-user", services.Database.CreateUser)
    internal.Post("/database/delete-user", services.Database.DeleteUser)
    internal.Post("/database/run-sql", services.Database.RunSQL)
    
    // Internal metrics
    internal.Get("/metrics", services.Metrics.GetInternalMetrics)
}

func (s *Server) Start() error {
    return s.app.Listen(s.cfg.ServerHost + ":" + s.cfg.ServerPort)
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.app.ShutdownWithContext(ctx)
}

func errorHandler(c fiber.Ctx, err error) error {
    code := fiber.StatusInternalServerError
    message := "Internal Server Error"
    
    if e, ok := err.(*fiber.Error); ok {
        code = e.Code
        message = e.Message
    }
    
    return c.Status(code).JSON(response.ErrorResponse(message))
}
```

### 2.3 Configuration Loader

**File**: `internal/config/config.go`

```go
package config

import (
    "time"
    
    "github.com/caarlos0/env/v11"
)

type Config struct {
    ServerPort     string        `env:"SERVER_PORT" envDefault:"8080"`
    ServerHost     string        `env:"SERVER_HOST" envDefault:"0.0.0.0"`
    Environment    string        `env:"ENVIRONMENT" envDefault:"development"`
    
    DatabaseURL      string `env:"DATABASE_URL,required"`
    DatabaseHost     string `env:"DATABASE_HOST,required"`
    DatabasePort     string `env:"DATABASE_PORT,required"`
    DatabaseUser     string `env:"DATABASE_USER,required"`
    DatabasePassword string `env:"DATABASE_PASSWORD,required"`
    DatabaseName     string `env:"DATABASE_NAME,required"`
    
    JWTSecret          string        `env:"JWT_SECRET,required"`
    JWTExpiration      time.Duration `env:"JWT_EXPIRATION" envDefault:"15m"`
    RefreshExpiration  time.Duration `env:"REFRESH_EXPIRATION" envDefault:"168h"`
    
    GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
    GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
    GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL"`
    
    InternalAPIKey string `env:"INTERNAL_API_KEY,required"`
    EncryptionKey  string `env:"ENCRYPTION_KEY,required"`
    
    AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
    
    PostgresSuperuser         string `env:"POSTGRES_SUPERUSER" envDefault:"postgres"`
    PostgresSuperuserPassword string `env:"POSTGRES_SUPERUSER_PASSWORD" envDefault:"postgres"`
}

func Load() *Config {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        panic("failed to load config: " + err.Error())
    }
    return cfg
}
```

### 2.4 JWT Package

**File**: `pkg/jwt/jwt.go`

```go
package jwt

import (
    "errors"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrTokenExpired = errors.New("token expired")
)

type Claims struct {
    UserID    uuid.UUID `json:"user_id"`
    Email     string    `json:"email"`
    TokenType string    `json:"token_type"` // access, refresh
    jwt.RegisteredClaims
}

type JWTManager struct {
    secret          []byte
    accessDuration time.Duration
    refreshDuration time.Duration
}

func NewManager(secret string, accessDuration, refreshDuration time.Duration) *JWTManager {
    return &JWTManager{
        secret:          []byte(secret),
        accessDuration: accessDuration,
        refreshDuration: refreshDuration,
    }
}

func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email string) (string, error) {
    return m.generateToken(userID, email, "access", m.accessDuration)
}

func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, email string) (string, error) {
    return m.generateToken(userID, email, "refresh", m.refreshDuration)
}

func (m *JWTManager) generateToken(userID uuid.UUID, email, tokenType string, duration time.Duration) (string, error) {
    claims := Claims{
        UserID:    userID,
        Email:     email,
        TokenType: tokenType,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   userID.String(),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidToken
        }
        return m.secret, nil
    })
    
    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrTokenExpired
        }
        return nil, ErrInvalidToken
    }
    
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    
    return claims, nil
}
```

### 2.5 Response Package

**File**: `pkg/response/response.go`

```go
package response

import (
    "github.com/gofiber/fiber/v3"
)

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

func Success(c fiber.Ctx, data interface{}) error {
    return c.Status(fiber.StatusOK).JSON(Response{
        Success: true,
        Data:    data,
    })
}

func Created(c fiber.Ctx, data interface{}) error {
    return c.Status(fiber.StatusCreated).JSON(Response{
        Success: true,
        Data:    data,
    })
}

func Paginated(c fiber.Ctx, data interface{}, meta *Meta) error {
    return c.Status(fiber.StatusOK).JSON(Response{
        Success: true,
        Data:    data,
        Meta:    meta,
    })
}

func Error(c fiber.Ctx, status int, message string) error {
    return c.Status(status).JSON(Response{
        Success: false,
        Error:   message,
    })
}

func ErrorResponse(message string) Response {
    return Response{
        Success: false,
        Error:   message,
    }
}
```

---

## Phase 3: Feature Implementation

### 3.1 Authentication Service

**File**: `internal/service/auth.go`

```go
package service

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "time"
    
    "github.com/gofiber/fiber/v3"
    "github.com/google/uuid"
    "golang.org/x/crypto/argon2"
    
    "aipsa-backend/internal/config"
    "aipsa-backend/internal/db"
    "aipsa-backend/pkg/jwt"
    "aipsa-backend/pkg/response"
)

type AuthService struct {
    queries   *db.Queries
    jwt       *jwt.JWTManager
    cfg       *config.Config
}

func NewAuthService(queries *db.Queries, jwtManager *jwt.JWTManager, cfg *config.Config) *AuthService {
    return &AuthService{
        queries: queries,
        jwt:     jwtManager,
        cfg:     cfg,
    }
}

func (s *AuthService) Login(c fiber.Ctx) error {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    
    if err := c.Bind().Body(&req); err != nil {
        return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
    }
    
    // Get user by email
    user, err := s.queries.GetUserByEmail(c.Context(), req.Email)
    if err != nil {
        return response.Error(c, fiber.StatusUnauthorized, "Invalid credentials")
    }
    
    // Verify password
    if user.PasswordHash == nil {
        return response.Error(c, fiber.StatusUnauthorized, "Password not set for this account")
    }
    
    if !verifyPassword(req.Password, *user.PasswordHash) {
        return response.Error(c, fiber.StatusUnauthorized, "Invalid credentials")
    }
    
    // Generate tokens
    accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
    }
    
    refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Email)
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
    }
    
    // Store refresh token
    hashedToken := hashToken(refreshToken)
    _, err = s.queries.CreateSession(c.Context(), db.CreateSessionParams{
        UserID:           user.ID,
        RefreshTokenHash: hashedToken,
        UserAgent:        c.Get("User-Agent"),
        IpAddress:        c.IP(),
        ExpiresAt:        time.Now().Add(s.cfg.RefreshExpiration),
    })
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to store session")
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

func (s *AuthService) Register(c fiber.Ctx) error {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
        FullName string `json:"full_name"`
    }
    
    if err := c.Bind().Body(&req); err != nil {
        return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
    }
    
    // Hash password
    hashedPassword := hashPassword(req.Password)
    
    // Create user
    user, err := s.queries.CreateUser(c.Context(), db.CreateUserParams{
        Email:        req.Email,
        PasswordHash: &hashedPassword,
        FullName:     req.FullName,
        Provider:     "email",
    })
    if err != nil {
        return response.Error(c, fiber.StatusConflict, "Email already exists")
    }
    
    // Generate tokens
    accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
    }
    
    refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Email)
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to generate token")
    }
    
    return response.Created(c, fiber.Map{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "user": fiber.Map{
            "id":        user.ID,
            "email":     user.Email,
            "full_name": user.FullName,
        },
    })
}

func hashPassword(password string) string {
    salt := make([]byte, 16)
    rand.Read(salt)
    
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash)
}

func verifyPassword(password, hashedPassword string) bool {
    parts := strings.Split(hashedPassword, ":")
    if len(parts) != 2 {
        return false
    }
    
    salt, _ := hex.DecodeString(parts[0])
    expectedHash, _ := hex.DecodeString(parts[1])
    
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    return bytes.Equal(hash, expectedHash)
}

func hashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
```

### 3.2 Projects Service

**File**: `internal/service/projects.go`

```go
package service

import (
    "context"
    
    "github.com/gofiber/fiber/v3"
    "github.com/google/uuid"
    
    "aipsa-backend/internal/db"
    "aipsa-backend/pkg/response"
)

type ProjectService struct {
    queries      *db.Queries
    provisioner  *Provisioner
}

func NewProjectService(queries *db.Queries, provisioner *Provisioner) *ProjectService {
    return &ProjectService{
        queries:     queries,
        provisioner: provisioner,
    }
}

func (s *ProjectService) List(c fiber.Ctx) error {
    userID := c.Locals("user_id").(uuid.UUID)
    
    projects, err := s.queries.ListProjectsByUser(c.Context(), userID)
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch projects")
    }
    
    return response.Success(c, projects)
}

func (s *ProjectService) Create(c fiber.Ctx) error {
    userID := c.Locals("user_id").(uuid.UUID)
    
    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
    }
    
    if err := c.Bind().Body(&req); err != nil {
        return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
    }
    
    // Generate slug
    slug := generateSlug(req.Name)
    
    // Get user's organization
    org, err := s.queries.GetOrganizationByUser(c.Context(), userID)
    if err != nil {
        return response.Error(c, fiber.StatusNotFound, "Organization not found")
    }
    
    // Create project
    project, err := s.queries.CreateProject(c.Context(), db.CreateProjectParams{
        OrganizationID: org.ID,
        Name:           req.Name,
        Slug:           slug,
        Description:    &req.Description,
        Status:         "active",
    })
    if err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to create project")
    }
    
    // Provision database
    if err := s.provisioner.Provision(c.Context(), project); err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to provision database")
    }
    
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
    
    project, err := s.queries.UpdateProject(c.Context(), db.UpdateProjectParams{
        ID:          projectID,
        Name:        req.Name,
        Description: &req.Description,
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
    
    // Deprovision database first
    if err := s.provisioner.Deprovision(c.Context(), projectID); err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to deprovision database")
    }
    
    if err := s.queries.DeleteProject(c.Context(), projectID); err != nil {
        return response.Error(c, fiber.StatusInternalServerError, "Failed to delete project")
    }
    
    return response.Success(c, fiber.Map{"message": "Project deleted"})
}
```

### 3.3 Database Provisioner

**File**: `internal/service/provisioner.go`

```go
package service

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    
    "github.com/jackc/pgx/v5"
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
}

func NewProvisioner(superuserPool *pgxpool.Pool, queries *db.Queries, cfg *config.Config, encryptor *security.Encryptor) *Provisioner {
    return &Provisioner{
        superuserPool: superuserPool,
        queries:       queries,
        cfg:           cfg,
        encryptor:     encryptor,
    }
}

func (p *Provisioner) Provision(ctx context.Context, project *db.Project) error {
    dbName := fmt.Sprintf("proj_%s", project.ID.String()[:8])
    dbUser := fmt.Sprintf("user_%s", project.ID.String()[:8])
    dbPassword := generatePassword()
    
    // Create database
    createDBQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
    if _, err := p.superuserPool.Exec(ctx, createDBQuery); err != nil {
        return fmt.Errorf("failed to create database: %w", err)
    }
    
    // Create role
    createRoleQuery := fmt.Sprintf(`CREATE ROLE "%s" WITH LOGIN PASSWORD '%s'`, dbUser, dbPassword)
    if _, err := p.superuserPool.Exec(ctx, createRoleQuery); err != nil {
        return fmt.Errorf("failed to create role: %w", err)
    }
    
    // Grant privileges
    grantQuery := fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE "%s" TO "%s"`, dbName, dbUser)
    if _, err := p.superuserPool.Exec(ctx, grantQuery); err != nil {
        return fmt.Errorf("failed to grant privileges: %w", err)
    }
    
    // Connect to new database and set schema permissions
    connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
        p.cfg.PostgresSuperuser, p.cfg.PostgresSuperuserPassword,
        p.cfg.DatabaseHost, p.cfg.DatabasePort, dbName)
    
    newDBPool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        return fmt.Errorf("failed to connect to new database: %w", err)
    }
    defer newDBPool.Close()
    
    // Grant schema permissions
    grantSchemaQuery := fmt.Sprintf(`GRANT ALL ON SCHEMA public TO "%s"`, dbUser)
    if _, err := newDBPool.Exec(ctx, grantSchemaQuery); err != nil {
        return fmt.Errorf("failed to grant schema privileges: %w", err)
    }
    
    // Enable common extensions
    extensions := []string{"uuid-ossp", "pgcrypto", "citext", "pg_trgm"}
    for _, ext := range extensions {
        extQuery := fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s"`, ext)
        if _, err := newDBPool.Exec(ctx, extQuery); err != nil {
            // Log warning but don't fail
            continue
        }
    }
    
    // Encrypt password
    encryptedPassword, err := p.encryptor.Encrypt(dbPassword)
    if err != nil {
        return fmt.Errorf("failed to encrypt password: %w", err)
    }
    
    // Update project with database details
    _, err = p.queries.UpdateProjectDatabase(ctx, db.UpdateProjectDatabaseParams{
        ID:           project.ID,
        DbName:       dbName,
        DbHost:       p.cfg.DatabaseHost,
        DbPort:       5432,
        DbUser:       dbUser,
        DbPasswordEncrypted: encryptedPassword,
    })
    if err != nil {
        return fmt.Errorf("failed to update project: %w", err)
    }
    
    return nil
}

func (p *Provisioner) Deprovision(ctx context.Context, projectID uuid.UUID) error {
    project, err := p.queries.GetProject(ctx, projectID)
    if err != nil {
        return err
    }
    
    // Terminate connections
    terminateQuery := fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'`, project.DbName)
    if _, err := p.superuserPool.Exec(ctx, terminateQuery); err != nil {
        // Log warning but continue
    }
    
    // Drop database
    dropDBQuery := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, project.DbName)
    if _, err := p.superuserPool.Exec(ctx, dropDBQuery); err != nil {
        return fmt.Errorf("failed to drop database: %w", err)
    }
    
    // Drop role
    dropRoleQuery := fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, project.DbUser)
    if _, err := p.superuserPool.Exec(ctx, dropRoleQuery); err != nil {
        return fmt.Errorf("failed to drop role: %w", err)
    }
    
    return nil
}

func generatePassword() string {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

---

## Phase 4: Implementation Order

### Phase 1: Foundation (Week 1)
1. Project structure setup
2. Configuration management
3. Database connection and migrations
4. Basic Fiber app with middleware
5. Structured logging (slog)
6. Error handling
7. Response utilities

### Phase 2: Authentication (Week 1-2)
1. User model and repository (sqlc)
2. JWT implementation
3. Email/password authentication
4. Google OAuth
5. API Keys
6. Auth middleware
7. Session management

### Phase 3: Projects & Provisioning (Week 2)
1. Project model and repository
2. Organization model
3. Database provisioning service
4. PostgreSQL management
5. Connection string generation

### Phase 4: Core Features (Week 3)
1. SQL Editor
   - Query execution
   - Read-only mode
   - Query history
   - Pagination
2. Schema Explorer
   - Tables
   - Views
   - Functions
   - Indexes
   - Constraints

### Phase 5: Production Ready (Week 3-4)
1. Docker setup
2. Health checks
3. Prometheus metrics
4. Rate limiting
5. CORS
6. Security headers
7. Request ID tracking

### Phase 6: Testing (Week 4)
1. Unit tests for services
2. Integration tests
3. API endpoint tests
4. Test coverage

### Phase 7: Documentation (Week 4)
1. OpenAPI/Swagger documentation
2. README
3. API documentation

---

## Key Implementation Notes

### Security Considerations
- Use Argon2id for password hashing
- Encrypt database credentials with AES-256
- Never expose superuser credentials
- Implement rate limiting
- Use secure headers
- Validate all inputs
- Use parameterized queries (sqlc handles this)

### Performance Considerations
- Use connection pooling (pgxpool)
- Implement caching where appropriate
- Use pagination for list endpoints
- Monitor slow queries

### Testing Strategy
- Unit tests for business logic
- Integration tests for database operations
- API tests for endpoints
- Use testcontainers for database tests

---

## Deliverables Checklist

- [ ] Modular architecture
- [ ] SQL migrations
- [ ] sqlc configuration
- [ ] Docker Compose (1 env var only)
- [ ] Dockerfiles
- [ ] Makefile
- [ ] OpenAPI documentation
- [ ] Authentication (Email + Google OAuth)
- [ ] Internal and Public APIs
- [ ] Provisioning service
- [ ] PostgreSQL management layer
- [ ] Health checks
- [ ] Metrics (Prometheus)
- [ ] Auto-generated secrets (zero-config)
- [ ] CI-ready project structure
- [ ] Comprehensive README

## Quick Start

```bash
# 1. Clone and start
docker compose up -d

# 2. Register first user
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password","full_name":"Admin"}'

# 3. Start using the platform!
```
