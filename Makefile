.PHONY: help dev dev-hub build build-hub test test-short test-coverage test-mutation test-mutation-report lint lint-fix fmt sec vuln migrate-up migrate-down migrate-create docker-up docker-down docker-build clean deps install-tools pre-commit-install pre-commit-run

# Default target
help:
	@echo "WatchDog - Development Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Start full development environment (DB + Hub with hot reload)"
	@echo "  make dev-hub      - Run hub with hot reload (Air)"
	@echo ""
	@echo "Build:"
	@echo "  make build        - Build hub binary"
	@echo "  make build-hub    - Build hub binary"
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
	@echo "  make test              - Run all tests with race detection"
	@echo "  make test-short        - Run quick tests"
	@echo "  make test-coverage     - Generate HTML coverage report"
	@echo "  make test-mutation     - Run mutation tests with Gremlins"
	@echo "  make lint              - Run linter"
	@echo "  make lint-fix          - Run linter with auto-fix"
	@echo "  make fmt               - Format code"
	@echo "  make sec               - Run security scan (gosec)"
	@echo "  make vuln              - Check for vulnerabilities"
	@echo ""
	@echo "Setup:"
	@echo "  make deps              - Download Go dependencies"
	@echo "  make install-tools     - Install development tools"
	@echo "  make pre-commit-install - Install pre-commit hooks"
	@echo "  make pre-commit-run    - Run pre-commit on all files"

# Variables
BINARY_DIR := bin
HUB_BINARY := $(BINARY_DIR)/hub
DATABASE_URL ?= postgres://watchdog:watchdog@localhost:5432/watchdog?sslmode=disable

# Development
dev: docker-db dev-hub

dev-hub:
	@command -v air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air -c .air.toml

docker-db:
	docker compose -f deployments/docker-compose.yml up -d postgres

# Build
build: build-hub

build-hub:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(HUB_BINARY) ./cmd/hub

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

test-short:
	go test -short ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-mutation:
	@command -v gremlins > /dev/null || (echo "Installing gremlins..." && go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
	gremlins unleash --config .gremlins.yaml

test-mutation-report:
	@command -v gremlins > /dev/null || (echo "Installing gremlins..." && go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
	gremlins unleash --config .gremlins.yaml --output html

lint:
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

lint-fix:
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run --fix ./...

fmt:
	go fmt ./...
	goimports -w .

sec:
	@command -v gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -quiet ./...

vuln:
	@command -v govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

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
	go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Done! All tools installed."

pre-commit-install:
	@command -v pre-commit > /dev/null || (echo "Please install pre-commit: pip install pre-commit" && exit 1)
	pre-commit install

pre-commit-run:
	@command -v pre-commit > /dev/null || (echo "Please install pre-commit: pip install pre-commit" && exit 1)
	pre-commit run --all-files

# Cleanup
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Generate (for future use)
generate:
	go generate ./...
