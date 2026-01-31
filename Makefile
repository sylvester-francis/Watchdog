.PHONY: help dev dev-hub dev-agent build build-hub build-agent test lint fmt migrate-up migrate-down migrate-create docker-up docker-down docker-build clean deps install-tools

# Default target
help:
	@echo "WatchDog - Development Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Start full development environment (DB + Hub with hot reload)"
	@echo "  make dev-hub      - Run hub with hot reload (Air)"
	@echo "  make dev-agent    - Run agent in development mode"
	@echo ""
	@echo "Build:"
	@echo "  make build        - Build both hub and agent binaries"
	@echo "  make build-hub    - Build hub binary"
	@echo "  make build-agent  - Build agent for all platforms"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up   - Apply all migrations"
	@echo "  make migrate-down - Rollback last migration"
	@echo "  make migrate-create NAME=xxx - Create new migration"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up    - Start all containers"
	@echo "  make docker-down  - Stop all containers"
	@echo "  make docker-build - Build Docker images"
	@echo ""
	@echo "Quality:"
	@echo "  make test         - Run all tests"
	@echo "  make lint         - Run linter"
	@echo "  make fmt          - Format code"
	@echo ""
	@echo "Setup:"
	@echo "  make deps         - Download Go dependencies"
	@echo "  make install-tools - Install development tools"

# Variables
BINARY_DIR := bin
HUB_BINARY := $(BINARY_DIR)/hub
AGENT_BINARY := $(BINARY_DIR)/agent
DATABASE_URL ?= postgres://watchdog:watchdog@localhost:5432/watchdog?sslmode=disable

# Development
dev: docker-db dev-hub

dev-hub:
	@command -v air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air -c .air.toml

dev-agent:
	go run ./cmd/agent/main.go

docker-db:
	docker compose -f deployments/docker-compose.yml up -d postgres

# Build
build: build-hub build-agent

build-hub:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(HUB_BINARY) ./cmd/hub

build-agent:
	@./scripts/build-agent.sh

# Database
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=create_users"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(NAME)

# Docker
docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

docker-build:
	docker compose -f deployments/docker-compose.yml build

docker-logs:
	docker compose -f deployments/docker-compose.yml logs -f

# Quality
test:
	go test -v -race -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -w .

# Setup
deps:
	go mod download
	go mod tidy

install-tools:
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Done! All tools installed."

# Cleanup
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Generate (for future use)
generate:
	go generate ./...
