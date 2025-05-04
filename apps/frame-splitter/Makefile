.PHONY: build run test lint clean docker-build docker-push generate-proto

# Variables
SERVICE_NAME := frame-splitter
CMD_DIR := ./cmd/main.go
BIN_NAME := build/frame-splitter
CONFIG_FILE := ./config/config.yaml
REGISTRY := streamhive
VERSION := $(shell git describe --tags --always --dirty || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"
DOCKER_IMAGE=$(REGISTRY)/$(SERVICE_NAME)

# Build the application
build:
	@echo "Building $(SERVICE_NAME)..."
	@mkdir -p build
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
	@docker build -t $(DOCKER_IMAGE):$(VERSION) -f Dockerfile ..

# Build Docker image for local development
docker-build-local:
	@echo "Building Docker image for local development..."
	@cd ../.. && docker build -t $(DOCKER_IMAGE):$(VERSION) -f ./apps/frame-splitter/Dockerfile .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(VERSION)

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8084:8084 -p 9091:9091 -v $(PWD)/data:/data $(DOCKER_IMAGE):$(VERSION)

# Generate protocol buffer files
generate-proto:
	@echo "Generating protocol buffer files..."
	@cd ../../libs/proto && \
		protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		frame/*.proto

# Generate mock files for testing
mocks:
	@echo "Generating mocks..."
	@mockgen -source=internal/processor/processor.go -destination=internal/processor/mocks/processor_mock.go -package=mocks

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Show help
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  dev            - Run with hot reloading"
	@echo "  test           - Run tests"
	@echo "  test-cov       - Run tests with coverage"
	@echo "  lint           - Run linters"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-build-local - Build Docker image for local development"
	@echo "  docker-push    - Push Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  generate-proto - Generate protobuf code"
	@echo "  mocks          - Generate mock files"
	@echo "  deps           - Install dependencies"
	@echo "  help           - Show this help"

.DEFAULT_GOAL := help
