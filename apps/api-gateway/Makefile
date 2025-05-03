.PHONY: build clean docker run test proto lint help

# Build variables
BINARY_NAME=api-gateway
BUILD_DIR=./build
MAIN_FILE=./cmd/main.go
DOCKER_IMAGE=streamhive/api-gateway

# Go options
GO=go
GOFLAGS=-ldflags="-s -w"

help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the API Gateway binary
	mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

run: ## Run the API Gateway locally
	$(GO) run $(MAIN_FILE) --config ./config/config.yaml

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

test: ## Run tests
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

lint: ## Run linters
	golangci-lint run ./...

proto: ## Generate protobuf code (if needed)
	@echo "Generating protobuf code from proto files..."
	@cd ../../libs/proto && \
		protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		signaling/*.proto

proto-install: ## Install protobuf compiler and plugins
	@echo "Installing protobuf compiler and plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-build-mac: ## Build Docker image from project root with proper context
	cd ../.. && docker build -t $(DOCKER_IMAGE) -f ./apps/api-gateway/Dockerfile .


docker-run: ## Run Docker container
	docker run -p 8080:8080 -p 50051:50051 -p 8082:8082 $(DOCKER_IMAGE)

dev: ## Run in development mode with reload
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air -c .air.toml

deps: ## Download dependencies
	$(GO) mod download

tidy: ## Tidy and verify dependencies
	$(GO) mod tidy
	$(GO) mod verify

.DEFAULT_GOAL := help
