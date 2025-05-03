# Build stage
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

# Now set up the stream-router module
WORKDIR /src/apps/stream-router
# Copy stream-router module files
COPY ./apps/stream-router/go.mod .
COPY ./apps/stream-router/go.sum .
RUN go mod tidy && go mod download

# Copy stream-router source code
COPY ./apps/stream-router/ .

# Build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o stream-router ./cmd/server

# Final stage
FROM alpine:3.17

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /src

# Copy binary from builder stage
COPY --from=builder /src/apps/stream-router/stream-router .

# Copy configuration files
COPY --from=builder /src/apps/stream-router/config/*.yaml ./config/

# Ensure proper ownership
RUN chown -R appuser:appgroup /src

# Install curl for health checks
RUN apk --no-cache add curl

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8081 9090

# Set environment variables
ENV HTTP_ADDRESS=:8081
ENV GRPC_ADDRESS=:9090

# Command to run the application
CMD ["./stream-router", "--config", "./config/config.yaml"]
