# CasDash Makefile

.PHONY: build build-all clean test lint docker docker-dev docker-prod run dev help

# Variables
BINARY_NAME=casdash
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "development")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Go build flags
GO_BUILD_FLAGS=-a -installsuffix cgo
CGO_ENABLED=1

# Default target
all: build

## Build single binary for current platform
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) main.go

## Build for all supported platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p dist

	# Linux AMD64
	@echo "Building for linux/amd64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 main.go

	# Linux ARM64
	@echo "Building for linux/arm64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 main.go

	# macOS AMD64
	@echo "Building for darwin/amd64..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 main.go

	# macOS ARM64
	@echo "Building for darwin/arm64..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 main.go

	# Windows AMD64
	@echo "Building for windows/amd64..."
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe main.go

## Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@rm -rf data/

## Run tests
test:
	@echo "Running tests..."
	go test -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -cover ./...

## Lint code
lint:
	@echo "Running linter..."
	golangci-lint run

## Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## Run locally
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

## Development mode with hot reload
dev:
	@echo "Starting development mode..."
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml --profile hot-reload up casdash-dev

## Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t casdash:latest .

## Build and run with Docker Compose
docker-dev:
	@echo "Starting development environment..."
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build

## Production Docker build
docker-prod:
	@echo "Building production Docker image..."
	docker build -t casdash:$(VERSION) .
	docker tag casdash:$(VERSION) casdash:latest

## Start with PostgreSQL
start-postgres:
	@echo "Starting with PostgreSQL..."
	docker-compose --profile postgres up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	docker-compose up --build casdash

## Start with MySQL
start-mysql:
	@echo "Starting with MySQL..."
	docker-compose --profile mysql up -d mysql
	@echo "Waiting for MySQL to be ready..."
	@sleep 10
	docker-compose up --build casdash

## Initialize development database
init-db:
	@echo "Initializing development database..."
	@mkdir -p data
	./$(BINARY_NAME) --help || echo "Binary not found, please run 'make build' first"

## Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	go mod download
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## Update dependencies
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

## Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"

## Show help
help:
	@echo "Available commands:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'