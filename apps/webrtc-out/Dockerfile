# apps/webrtc-out/Dockerfile
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

# Now set up the webrtc-out module
WORKDIR /src/apps/webrtc-out
# Copy webrtc-out module files
COPY ./apps/webrtc-out/go.mod .
COPY ./apps/webrtc-out/go.sum .
RUN go mod tidy && go mod download

# Copy source code
COPY ./apps/webrtc-out/ .

# Build the application
RUN go mod tidy &&  CGO_ENABLED=0 GOOS=linux go build -o webrtc-out ./cmd/server/main.go

# Final stage
FROM alpine:3.17

# Set working directory
WORKDIR /src

# Copy binary from builder
COPY --from=builder /src/apps/webrtc-out .

# Copy configuration
COPY --from=builder /src/apps/webrtc-out/config/config.yaml ./config/config.yaml

# Expose ports
EXPOSE 8085 9094

# Command to run the application
CMD ["./webrtc-out", "--config", "./config/config.yaml"]
