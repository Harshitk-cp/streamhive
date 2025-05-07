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

	// HTTP server configuration for REST API and health checks
	HTTP struct {
		Address         string        `yaml:"address"`
		ReadTimeout     time.Duration `yaml:"read_timeout"`
		WriteTimeout    time.Duration `yaml:"write_timeout"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	} `yaml:"http"`

	// WebSocket server configuration
	WebSocket struct {
		Address        string        `yaml:"address"`
		Path           string        `yaml:"path"`
		ReadTimeout    time.Duration `yaml:"read_timeout"`
		WriteTimeout   time.Duration `yaml:"write_timeout"`
		PingInterval   time.Duration `yaml:"ping_interval"`
		PongTimeout    time.Duration `yaml:"pong_timeout"`
		MaxMessageSize int64         `yaml:"max_message_size"`
	} `yaml:"websocket"`

	// gRPC server configuration
	GRPC struct {
		Address              string        `yaml:"address"`
		KeepAliveTime        time.Duration `yaml:"keep_alive_time"`
		KeepAliveTimeout     time.Duration `yaml:"keep_alive_timeout"`
		MaxConcurrentStreams int           `yaml:"max_concurrent_streams"`
	} `yaml:"grpc"`

	// WebRTC Out service address
	WebRTCOut struct {
		Address string `yaml:"address"`
	} `yaml:"webrtc_out"`

	// Router service address
	Router struct {
		Address string `yaml:"address"`
	} `yaml:"router"`

	// Logging configuration
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"logging"`
}

// Load loads the configuration from a file
func Load(path string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the configuration
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment overrides
	applyEnvironmentOverrides(config)

	// Set defaults
	setDefaults(config)

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

// setDefaults sets default values
func setDefaults(config *Config) {
	// Set default HTTP address
	if config.HTTP.Address == "" {
		config.HTTP.Address = ":8086"
	}

	// Set default WebSocket address and path
	if config.WebSocket.Address == "" {
		config.WebSocket.Address = ":8087"
	}
	if config.WebSocket.Path == "" {
		config.WebSocket.Path = "/ws"
	}

	// Set default gRPC address
	if config.GRPC.Address == "" {
		config.GRPC.Address = ":50052"
	}

	// Set default WebRTC Out address
	if config.WebRTCOut.Address == "" {
		config.WebRTCOut.Address = "webrtc-out:50053"
	}

	// Set default Router address
	if config.Router.Address == "" {
		config.Router.Address = "stream-router:9090"
	}

	// Set default HTTP timeouts
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = 10 * time.Second
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = 10 * time.Second
	}
	if config.HTTP.ShutdownTimeout == 0 {
		config.HTTP.ShutdownTimeout = 5 * time.Second
	}

	// Set default WebSocket timeouts
	if config.WebSocket.ReadTimeout == 0 {
		config.WebSocket.ReadTimeout = 60 * time.Second
	}
	if config.WebSocket.WriteTimeout == 0 {
		config.WebSocket.WriteTimeout = 10 * time.Second
	}
	if config.WebSocket.PingInterval == 0 {
		config.WebSocket.PingInterval = 25 * time.Second
	}
	if config.WebSocket.PongTimeout == 0 {
		config.WebSocket.PongTimeout = 60 * time.Second
	}
	if config.WebSocket.MaxMessageSize == 0 {
		config.WebSocket.MaxMessageSize = 1024 * 1024 // 1MB
	}

	// Set default gRPC settings
	if config.GRPC.KeepAliveTime == 0 {
		config.GRPC.KeepAliveTime = 60 * time.Second
	}
	if config.GRPC.KeepAliveTimeout == 0 {
		config.GRPC.KeepAliveTimeout = 20 * time.Second
	}
	if config.GRPC.MaxConcurrentStreams == 0 {
		config.GRPC.MaxConcurrentStreams = 100
	}

	// Set default logging configuration
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
	}
}
