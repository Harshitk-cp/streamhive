# apps/websocket-signaling/Makefile
.PHONY: build run test clean docker

# Build variables
BINARY_NAME=websocket-signaling
DOCKER_IMAGE=streamhive/websocket-signaling
DOCKER_TAG=latest

# Build the binary
build:
	go build -o $(BINARY_NAME) ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Build Docker image
docker:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Push Docker image
docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
