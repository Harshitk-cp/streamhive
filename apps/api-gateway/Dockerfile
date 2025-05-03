# Build stage
FROM golang:1.23.2-alpine AS builder

# Set working directory
WORKDIR /src

# Install necessary build tools
RUN apk add --no-cache git gcc musl-dev

# First, set up the proto module
WORKDIR /src/libs/proto
# Copy the proto module files
COPY ../../libs/proto/go.mod .
COPY ../../libs/proto/go.sum .
RUN go mod download

# Copy proto source files
COPY ../../libs/proto/ .

# Now set up the api-gateway module
WORKDIR /src/apps/api-gateway
# Copy API gateway module files
COPY apps/api-gateway/go.mod apps/api-gateway/go.sum ./
RUN go mod download

# Copy API gateway source code
COPY ../../apps/api-gateway/ ./

# Build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/main.go

# Final stage
FROM alpine:3.17

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /src/apps/api-gateway/api-gateway .

# Copy configuration files
COPY --from=builder /src/apps/api-gateway/config/*.yaml ./config/

# Ensure proper ownership
RUN chown -R appuser:appgroup /app

# Install curl for health checks
RUN apk --no-cache add curl

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 50051 8082

# Set environment variables
ENV GIN_MODE=release

# Command to run the application
CMD ["./api-gateway", "--config", "./config/config.yaml"]
