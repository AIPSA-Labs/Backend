# AIPSA Backend

A production-ready minimal PostgreSQL Database Platform built with Go (Fiber v3) and PostgreSQL.

## Quick Start

```bash
# Start with Docker (recommended)
docker compose up -d

# Or run locally
make run
```

That's it! No configuration needed. The platform works with sensible defaults.

**Note:** The server starts on a random port (10000-60000) by default. Check the logs for the actual port:
```bash
docker compose logs app | grep "port"
```

## Features

- **Zero-Config** - Works out of the box with auto-generated secrets
- **REST API** - Full REST API for database management
- **Database Provisioning** - Automatically create PostgreSQL databases per project
- **JWT Authentication** - Secure token-based authentication
- **Google OAuth** - Optional OAuth integration
- **SQL Editor** - Execute queries with history tracking
- **API Keys** - Generate and manage API keys
- **Rate Limiting** - Built-in request throttling
- **Structured Logging** - JSON logs with slog

## REST API

### Authentication

**Register:**
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","full_name":"John Doe"}'
```

Response:
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "full_name": "John Doe"
    }
  }
}
```

**Login:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

**Refresh Token:**
```bash
curl -X POST http://localhost:8080/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"eyJhbGciOiJIUzI1NiIs..."}'
```

### Projects

**List Projects:**
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/v1/projects
```

**Create Project:**
```bash
curl -X POST http://localhost:8080/v1/projects \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"My App","description":"Production database"}'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My App",
    "slug": "my-app-a1b2c3d4",
    "status": "active",
    "db_name": "proj_550e8400",
    "created_at": "2026-07-04T12:00:00Z"
  }
}
```

**Get Project:**
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/v1/projects/<project_id>
```

**Update Project:**
```bash
curl -X PATCH http://localhost:8080/v1/projects/<project_id> \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Name"}'
```

**Delete Project:**
```bash
curl -X DELETE http://localhost:8080/v1/projects/<project_id> \
  -H "Authorization: Bearer <access_token>"
```

### Database Management

**Get Database Info:**
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/v1/projects/<project_id>/database
```

Response:
```json
{
  "success": true,
  "data": {
    "status": "ready",
    "db_name": "proj_550e8400",
    "db_host": "localhost",
    "db_port": 5432,
    "db_user": "user_550e8400",
    "connection_string": "postgres://user_xxx:password@localhost:5432/proj_xxx?sslmode=disable"
  }
}
```

**Reset Database Password:**
```bash
curl -X POST http://localhost:8080/v1/projects/<project_id>/database/reset-password \
  -H "Authorization: Bearer <access_token>"
```

### SQL Editor

**Execute SQL Query:**
```bash
curl -X POST http://localhost:8080/v1/projects/<project_id>/sql/execute \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"query":"SELECT * FROM users LIMIT 10"}'
```

Response:
```json
{
  "success": true,
  "data": {
    "results": [
      {"id": 1, "name": "John", "email": "john@example.com"}
    ],
    "rows_affected": 1,
    "duration_ms": 12
  }
}
```

**Get Query History:**
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/v1/projects/<project_id>/sql/history
```

### API Keys

**List API Keys:**
```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/v1/keys
```

**Create API Key:**
```bash
curl -X POST http://localhost:8080/v1/keys \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"My App Key","permissions":["read","write"]}'
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My App Key",
    "api_key": "ak_a1b2c3d4e5f6...",
    "key_prefix": "ak_a1b2c3d4",
    "permissions": ["read", "write"]
  }
}
```

**Delete API Key:**
```bash
curl -X DELETE http://localhost:8080/v1/keys/<key_id> \
  -H "Authorization: Bearer <access_token>"
```

### Internal API

Internal APIs are protected by API key authentication.

**Provision Database:**
```bash
curl -X POST http://localhost:8080/internal/projects/provision \
  -H "X-Internal-Key: <internal_api_key>" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"550e8400-e29b-41d4-a716-446655440000"}'
```

**Delete Database:**
```bash
curl -X POST http://localhost:8080/internal/projects/delete \
  -H "X-Internal-Key: <internal_api_key>" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"550e8400-e29b-41d4-a716-446655440000"}'
```

## API Endpoints Summary

### Public API (`/v1`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/auth/register` | Register new user |
| POST | `/v1/auth/login` | Login |
| POST | `/v1/auth/refresh` | Refresh token |
| POST | `/v1/auth/logout` | Logout |
| GET | `/v1/me` | Get current user |
| PATCH | `/v1/me` | Update user |
| GET | `/v1/projects` | List projects |
| POST | `/v1/projects` | Create project |
| GET | `/v1/projects/:id` | Get project |
| PATCH | `/v1/projects/:id` | Update project |
| DELETE | `/v1/projects/:id` | Delete project |
| GET | `/v1/projects/:id/database` | Get database info |
| POST | `/v1/projects/:id/database/reset-password` | Reset password |
| POST | `/v1/projects/:id/sql/execute` | Execute SQL |
| GET | `/v1/projects/:id/sql/history` | SQL history |
| GET | `/v1/keys` | List API keys |
| POST | `/v1/keys` | Create API key |
| DELETE | `/v1/keys/:id` | Delete API key |

### Internal API (`/internal`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/internal/projects/provision` | Provision database |
| POST | `/internal/projects/delete` | Delete database |

## Configuration

All values have sensible defaults. Only override what you need:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://aipsa:aipsa_secret@localhost:5432/aipsa_platform?sslmode=disable` | PostgreSQL connection |
| `SERVER_PORT` | random (10000-60000) | HTTP port |
| `SERVER_HOST` | `0.0.0.0` | Bind address |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `JWT_SECRET` | auto-generated | JWT signing secret |
| `JWT_EXPIRATION` | `15m` | Access token expiry |
| `REFRESH_EXPIRATION` | `168h` | Refresh token expiry |
| `GOOGLE_CLIENT_ID` | empty (disabled) | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | empty (disabled) | Google OAuth secret |
| `ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:8080` | CORS origins |

## Development

```bash
# Build
make build

# Run
make run

# Run tests
make test

# Run tests with coverage
make test-coverage

# Generate sqlc code
make sqlc

# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# View Docker logs
make docker-logs

# Stop Docker
make docker-down

# Format code
make format

# Lint code
make lint
```

## Project Structure

```
cmd/server/              - Application entry point
internal/
  config/                - Configuration (zero-config defaults)
  db/                    - Database queries (sqlc)
  middleware/            - HTTP middleware (auth, CORS, logger)
  security/              - Encryption, password hashing
  server/                - Fiber server setup
  service/               - Business logic
    auth.go              - Authentication service
    users.go             - User service
    projects.go          - Project management
    database.go          - Database operations
    provisioner.go       - PostgreSQL provisioning
    apikeys.go           - API key management
    health.go            - Health check
    dependencies.go      - Dependency injection
pkg/
  jwt/                   - JWT token management
  response/              - Standard API responses
  pagination/            - Pagination utilities
migrations/              - SQL migrations (Goose)
sql/queries/             - SQL queries (sqlc)
sqlc.yaml                - sqlc configuration
docker-compose.yml       - Docker deployment
Dockerfile               - Container build
Makefile                 - Build commands
```

## Tech Stack

- **Language**: Go 1.25+
- **Web Framework**: Fiber v3
- **Database**: PostgreSQL 17+
- **Driver**: pgx (no ORM)
- **Code Generation**: sqlc
- **Migrations**: Goose
- **Authentication**: JWT + Google OAuth
- **Logging**: slog (structured)
- **Containerization**: Docker + Docker Compose

## License

MIT
#   B a c k e n d  
 