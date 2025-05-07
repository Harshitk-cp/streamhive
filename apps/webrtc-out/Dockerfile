# Build stage
FROM golang:1.23.2-alpine AS builder

# Set working directory
WORKDIR /src

# Install build dependencies
RUN apk add --no-cache gcc musl-dev pkgconfig

# First, set up the proto module
WORKDIR /src/libs/proto
# Copy the proto module files
COPY ./libs/proto/go.mod .
COPY ./libs/proto/go.sum .
RUN go mod tidy && go mod download

# Copy proto source files
COPY ./libs/proto/ .

# Now set up the webrtc-out module
WORKDIR /src/apps/webrtc-out
# Copy webrtc-out module files
COPY ./apps/webrtc-out/go.mod .
COPY ./apps/webrtc-out/go.sum .
RUN go mod tidy && go mod download

# Copy webrtc-out source code
COPY ./apps/webrtc-out/ .

# Build the application
RUN go mod tidy && CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o webrtc-out ./cmd/server

# Final stage
FROM alpine:3.17

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /src/apps/webrtc-out/webrtc-out .

# Copy configuration files
COPY --from=builder /src/apps/webrtc-out/config/*.yaml ./config/

# Ensure proper ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8088 50053

# Set environment variables
ENV HTTP_ADDRESS=:8088
ENV GRPC_ADDRESS=:50053

# Command to run the application
CMD ["./webrtc-out", "--config", "./config/config.yaml"]
