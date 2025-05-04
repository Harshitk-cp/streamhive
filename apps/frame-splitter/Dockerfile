# Build stage
FROM golang:1.23.2-alpine AS build

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

# Now set up the frame-splitter module
WORKDIR /src/apps/frame-splitter
# Copy frame-splitter module files
COPY ./apps/frame-splitter/go.mod .
COPY ./apps/frame-splitter/go.sum .
RUN go mod tidy && go mod download

# Copy frame-splitter source code
COPY ./apps/frame-splitter/ .

# Build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o frame-splitter ./cmd/server/main.go

# Final stage
FROM alpine:3.17

# Install CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata curl

# Set working directory
WORKDIR /src

# Copy binary from builder stage
COPY --from=build /src/apps/frame-splitter/frame-splitter .

# Copy configuration
COPY --from=build /src/apps/frame-splitter/config/config.yaml ./config/config.yaml

# Create data directory for frame backups
RUN mkdir -p /data/frames && chmod -R 777 /data

# Expose ports
EXPOSE 8084 9091

# Set environment variables
ENV HTTP_ADDRESS=:8084
ENV GRPC_ADDRESS=:9091

# Command to run the application
CMD ["./frame-splitter", "--config", "./config/config.yaml"]
