# DSPM Makefile

.PHONY: build run test clean deps lint migrate docker-build docker-up docker-down

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOLINT := golangci-lint

# Directories
BUILD_DIR := ./build
CMD_DIR := ./cmd/dspm

# Binary name
BINARY := dspm

## build: Build the DSPM binary
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

## run: Run the DSPM server
run: build
	@echo "Starting DSPM server..."
	$(BUILD_DIR)/$(BINARY) -config config.yaml

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## lint: Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## migrate: Run database migrations
migrate:
	@echo "Running migrations..."
	psql -d dspm -f migrations/001_initial.sql

## migrate-create: Create a new migration file
migrate-create:
	@read -p "Migration name: " name; \
	touch migrations/$$(date +%Y%m%d%H%M%S)_$$name.sql

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t dspm:$(VERSION) .

## docker-up: Start Docker Compose services
docker-up:
	@echo "Starting services..."
	docker-compose up -d

## docker-down: Stop Docker Compose services
docker-down:
	@echo "Stopping services..."
	docker-compose down

## docker-logs: View Docker Compose logs
docker-logs:
	docker-compose logs -f

## setup-db: Create database and run migrations
setup-db:
	@echo "Setting up database..."
	createdb dspm 2>/dev/null || true
	psql -d dspm -f migrations/001_initial.sql

## dev: Run in development mode with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

## generate: Generate code (mocks, etc.)
generate:
	$(GOCMD) generate ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	goimports -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## security: Run security scan
security:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec ./...

## all: Run all checks and build
all: deps fmt vet lint test build

## help: Show this help
help:
	@echo "DSPM - Data Security Posture Management"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

.DEFAULT_GOAL := help
