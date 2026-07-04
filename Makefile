.PHONY: all build run test migrate sqlc docker-up docker-down

APP_NAME=aipsa-backend
DOCKER_COMPOSE=docker compose

all: build

build:
	go build -o bin/$(APP_NAME) cmd/server/main.go

run:
	go run cmd/server/main.go

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

migrate-up:
	goose -dir migrations up

migrate-down:
	goose -dir migrations down

migrate-create:
	goose -dir migrations create $(name) sql

sqlc:
	sqlc generate

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-build:
	$(DOCKER_COMPOSE) build

docker-logs:
	$(DOCKER_COMPOSE) logs -f

lint:
	golangci-lint run

format:
	gofmt -s -w .
	goimports -w .

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback migrations"
	@echo "  sqlc           - Generate sqlc code"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-logs    - View Docker logs"
	@echo "  lint           - Run linter"
	@echo "  format         - Format code"
	@echo "  clean          - Clean build artifacts"
