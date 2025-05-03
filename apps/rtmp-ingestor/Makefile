.PHONY: build run test lint clean docker-build docker-push

# Variables
SERVICE_NAME := rtmp-ingestor
CMD_DIR := ./cmd/server
BIN_NAME := build/rtmp-ingestor
CONFIG_FILE := ./config/config.yaml
REGISTRY := myregistry.io
VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
DOCKER_IMAGE=streamhive/rtmp-ingestor

# Build the application
build:
	@echo "Building $(SERVICE_NAME)..."
	@go build $(LDFLAGS) -o $(BIN_NAME) $(CMD_DIR)

# Run the application
run: build
	@echo "Running $(SERVICE_NAME)..."
	@./$(BIN_NAME) --config $(CONFIG_FILE)

# Run the application with hot reloading using air
dev:
	@echo "Running $(SERVICE_NAME) in development mode..."
	@air -c .air.toml

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-cov:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Run linting
lint:
	@echo "Running linting..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BIN_NAME)
	@rm -f coverage.out
	@rm -f coverage.html

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(REGISTRY)/$(SERVICE_NAME):$(VERSION) -f Dockerfile ..

docker-build-mac: ## Build Docker image from project root with proper context
	cd ../.. && docker build -t $(DOCKER_IMAGE) -f ./apps/rtmp-ingestor/Dockerfile .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	@docker push $(REGISTRY)/$(SERVICE_NAME):$(VERSION)

# Generate mock files for testing
mocks:
	@echo "Generating mocks..."
	@mockgen -source=internal/router/client.go -destination=internal/router/mocks/client_mock.go -package=mocks

# Run integration tests
integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BIN_NAME)-linux-amd64 $(CMD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BIN_NAME)-darwin-amd64 $(CMD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BIN_NAME)-windows-amd64.exe $(CMD_DIR)

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  dev          - Run the application with hot reloading"
	@echo "  test         - Run tests"
	@echo "  test-cov     - Run tests with coverage"
	@echo "  lint         - Run linting"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image"
	@echo "  mocks        - Generate mock files for testing"
	@echo "  integration  - Run integration tests"
	@echo "  deps         - Install dependencies"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  help         - Show this help message"
