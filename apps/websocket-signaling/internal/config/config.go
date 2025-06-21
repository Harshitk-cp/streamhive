package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Service information
	Service struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
		Environment string `yaml:"environment"`
	} `yaml:"service"`

	HTTP      HTTPConfig      `yaml:"http"`
	WebSocket WebSocketConfig `yaml:"websocket"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	Log       LogConfig       `yaml:"log"`
	WebRTCOut WebRTCOutConfig `yaml:"webrtc_out"`
	Router    RouterConfig    `yaml:"router"`
}

// HTTPConfig represents HTTP server configuration
type HTTPConfig struct {
	Address         string        `yaml:"address"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// WebSocketConfig represents WebSocket server configuration
type WebSocketConfig struct {
	Address      string        `yaml:"address"`
	Path         string        `yaml:"path"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// GRPCConfig represents gRPC server configuration
type GRPCConfig struct {
	Address              string        `yaml:"address"`
	KeepAliveTime        time.Duration `yaml:"keepalive_time"`
	KeepAliveTimeout     time.Duration `yaml:"keepalive_timeout"`
	MaxConcurrentStreams int           `yaml:"max_concurrent_streams"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// WebRTCOutConfig represents WebRTC out service configuration
type WebRTCOutConfig struct {
	Address string `yaml:"address"`
}

// RouterConfig represents router service configuration
type RouterConfig struct {
	Address string `yaml:"address"`
}

// Load loads the configuration from a file
func Load(path string) (*Config, error) {
	// Set default configuration
	config := &Config{
		HTTP: HTTPConfig{
			Address:         ":8086",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 15 * time.Second,
		},
		WebSocket: WebSocketConfig{
			Address:      ":8087",
			Path:         "/ws",
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		GRPC: GRPCConfig{
			Address:              ":8088",
			KeepAliveTime:        30 * time.Second,
			KeepAliveTimeout:     10 * time.Second,
			MaxConcurrentStreams: 100,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		WebRTCOut: WebRTCOutConfig{
			Address: "webrtc-out:50053",
		},
		Router: RouterConfig{
			Address: "stream-router:9090",
		},
	}

	// Read the configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the configuration
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment overrides
	applyEnvironmentOverrides(config)

	return config, nil
}

// applyEnvironmentOverrides applies environment overrides
func applyEnvironmentOverrides(config *Config) {
	// HTTP address
	if addr := os.Getenv("HTTP_ADDRESS"); addr != "" {
		config.HTTP.Address = addr
	}

	// WebSocket address
	if addr := os.Getenv("WS_ADDRESS"); addr != "" {
		config.WebSocket.Address = addr
	}

	// gRPC address
	if addr := os.Getenv("GRPC_ADDRESS"); addr != "" {
		config.GRPC.Address = addr
	}

	// WebRTC Out address
	if addr := os.Getenv("WEBRTC_OUT_ADDRESS"); addr != "" {
		config.WebRTCOut.Address = addr
	}

	// Router address
	if addr := os.Getenv("ROUTER_ADDRESS"); addr != "" {
		config.Router.Address = addr
	}

	// Environment
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Service.Environment = env
	}
}
