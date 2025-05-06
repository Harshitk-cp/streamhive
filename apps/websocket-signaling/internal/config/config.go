package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the configuration for the WebSocket signaling service
type Config struct {
	// Service configuration
	Service struct {
		// Name of the service
		Name string
		// Environment (dev, staging, prod)
		Environment string
		// NodeID unique identifier for this node
		NodeID string
	}

	// HTTP server configuration
	HTTP struct {
		// Address to listen on
		Address string
		// CORS configuration
		CORS struct {
			// Allowed origins
			AllowedOrigins []string
			// Allowed methods
			AllowedMethods []string
			// Allowed headers
			AllowedHeaders []string
		}
	}

	// gRPC server configuration
	GRPC struct {
		Address              string        `yaml:"address"`
		MaxConcurrentStreams int           `yaml:"max_concurrent_streams"`
		ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
		KeepAliveTime        time.Duration `yaml:"keep_alive_time"`
		KeepAliveTimeout     time.Duration `yaml:"keep_alive_timeout"`
	} `yaml:"grpc"`

	// WebSocket configuration
	WebSocket struct {
		// Max message size in bytes
		MaxMessageSize int
		// Write wait timeout
		WriteWait int
		// Pong wait timeout
		PongWait int
		// Ping period
		PingPeriod int
	}
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Service configuration
	cfg.Service.Name = getEnvWithDefault("SERVICE_NAME", "websocket-signaling")
	cfg.Service.Environment = getEnvWithDefault("ENVIRONMENT", "dev")
	cfg.Service.NodeID = getEnvWithDefault("NODE_ID", fmt.Sprintf("node-%s", generateRandomString(8)))

	// HTTP server configuration
	cfg.HTTP.Address = getEnvWithDefault("HTTP_ADDRESS", ":8080")

	// CORS configuration
	cfg.HTTP.CORS.AllowedOrigins = getEnvArrayWithDefault("CORS_ALLOWED_ORIGINS", []string{"*"})
	cfg.HTTP.CORS.AllowedMethods = getEnvArrayWithDefault("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	cfg.HTTP.CORS.AllowedHeaders = getEnvArrayWithDefault("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization"})

	// gRPC server configuration
	cfg.GRPC.Address = getEnvWithDefault("GRPC_ADDRESS", ":9090")

	// WebSocket configuration
	cfg.WebSocket.MaxMessageSize = getEnvIntWithDefault("WS_MAX_MESSAGE_SIZE", 4096)
	cfg.WebSocket.WriteWait = getEnvIntWithDefault("WS_WRITE_WAIT", 10)
	cfg.WebSocket.PongWait = getEnvIntWithDefault("WS_PONG_WAIT", 60)
	cfg.WebSocket.PingPeriod = getEnvIntWithDefault("WS_PING_PERIOD", 30)

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Ensure NodeID is set
	if cfg.Service.NodeID == "" {
		return fmt.Errorf("NODE_ID is required")
	}

	return nil
}

// Helper functions

// getEnvWithDefault gets an environment variable or returns a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvArrayWithDefault gets an environment variable as an array or returns a default value
func getEnvArrayWithDefault(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// getEnvIntWithDefault gets an environment variable as an integer or returns a default value
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[int(os.Getpid()+i)%len(charset)]
	}
	return string(result)
}
