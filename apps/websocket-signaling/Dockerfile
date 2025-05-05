# apps/websocket-signaling/Dockerfile
FROM golang:1.23.2-alpine AS builder

# Set working directory
WORKDIR /src

# First, set up the proto module
WORKDIR /src/libs/proto
# Copy the proto module files
COPY ./libs/proto/go.mod .
COPY ./libs/proto/go.sum .
RUN go mod tidy && go mod download

# Copy proto source files
COPY ./libs/proto/ .

# Now set up the websocket-signaling module
WORKDIR /src/apps/websocket-signaling
# Copy websocket-signaling module files
COPY ./apps/websocket-signaling/go.mod .
COPY ./apps/websocket-signaling/go.sum .
RUN go mod tidy && go mod download

# Copy source code
COPY ./apps/websocket-signaling/ .

# Build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o websocket-signaling ./cmd/server/main.go

# Final stage
FROM alpine:3.17

# Set working directory
WORKDIR /src

# Copy binary from builder
COPY --from=builder /src/apps/websocket-signaling .

# Copy configuration
COPY --from=builder /src/apps/websocket-signaling/config/config.yaml ./config/config.yaml

# Expose ports
EXPOSE 8086 9095

# Command to run the application
CMD ["./websocket-signaling", "--config", "./config/config.yaml"]
