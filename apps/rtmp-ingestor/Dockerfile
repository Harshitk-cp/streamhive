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

# Now set up the rtmp-ingestor module
WORKDIR /src/apps/rtmp-ingestor
# Copy rtmp-ingestor module files
COPY ./apps/rtmp-ingestor/go.mod .
COPY ./apps/rtmp-ingestor/go.sum .
RUN go mod tidy && go mod download

# Copy rtmp-ingestor source code
COPY ./apps/rtmp-ingestor/ .

# Build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rtmp-ingestor ./cmd/server

# Final stage
FROM alpine:3.17

# Set working directory
WORKDIR /src

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Copy the binary from the build stage
COPY --from=build /src/apps/rtmp-ingestor .

# Copy configuration files
COPY ./apps/rtmp-ingestor/config/*.yaml ./config/config.yaml

# Expose ports
EXPOSE 8081 1935

# Set environment variables
ENV HTTP_ADDRESS=:8081
ENV RTMP_ADDRESS=:1935
ENV ROUTER_ADDRESS=stream-router:9090

# Run the application
CMD ["./rtmp-ingestor", "--config", "./config/config.yaml"]
