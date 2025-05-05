// apps/websocket-signaling/internal/config/config.go
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
		NodeID      string `yaml:"node_id"`
	} `yaml:"service"`

	// HTTP server configuration for WebSocket connections and metrics
	HTTP struct {
		Address           string        `yaml:"address"`
		ReadTimeout       time.Duration `yaml:"read_timeout"`
		WriteTimeout      time.Duration `yaml:"write_timeout"`
		IdleTimeout       time.Duration `yaml:"idle_timeout"`
		ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
		ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
		WebSocketEndpoint string        `yaml:"websocket_endpoint"`
		WebSocketPath     string        `yaml:"websocket_path"`
		AllowedOrigins    []string      `yaml:"allowed_origins"`
		EnableCORS        bool          `yaml:"enable_cors"`
		UseCompression    bool          `yaml:"use_compression"`
	} `yaml:"http"`

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
		HandshakeTimeout     time.Duration `yaml:"handshake_timeout"`
		WriteWait            time.Duration `yaml:"write_wait"`
		PongWait             time.Duration `yaml:"pong_wait"`
		PingPeriod           time.Duration `yaml:"ping_period"`
		MaxMessageSize       int64         `yaml:"max_message_size"`
		BufferSize           int           `yaml:"buffer_size"`
		EnableCompression    bool          `yaml:"enable_compression"`
		MessageBufferSize    int           `yaml:"message_buffer_size"`
		ClientSessionTimeout time.Duration `yaml:"client_session_timeout"`
	} `yaml:"websocket"`

	// Stream configuration
	Stream struct {
		MaxClientsPerStream int           `yaml:"max_clients_per_stream"`
		StreamTimeout       time.Duration `yaml:"stream_timeout"`
		MaxStreams          int           `yaml:"max_streams"`
	} `yaml:"stream"`

	// Logging configuration
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"logging"`

	// Tracing configuration
	Tracing struct {
		Enabled    bool    `yaml:"enabled"`
		Endpoint   string  `yaml:"endpoint"`
		ServiceTag string  `yaml:"service_tag"`
		SampleRate float64 `yaml:"sample_rate"`
	} `yaml:"tracing"`

	// Metrics configuration
	Metrics struct {
		Enabled       bool          `yaml:"enabled"`
		Address       string        `yaml:"address"`
		Path          string        `yaml:"path"`
		CollectPeriod time.Duration `yaml:"collect_period"`
	} `yaml:"metrics"`
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

	// Apply environment variable overrides
	applyEnvironmentOverrides(config)

	// Set defaults
	setDefaults(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides applies environment variable overrides
func applyEnvironmentOverrides(config *Config) {
	// Override HTTP address
	if addr := os.Getenv("HTTP_ADDRESS"); addr != "" {
		config.HTTP.Address = addr
	}

	// Override gRPC address
	if addr := os.Getenv("GRPC_ADDRESS"); addr != "" {
		config.GRPC.Address = addr
	}

	// Override environment
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Service.Environment = env
	}

	// Override node ID for horizontal scaling
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		config.Service.NodeID = nodeID
	}

	// Override WebSocket endpoint
	if endpoint := os.Getenv("WEBSOCKET_ENDPOINT"); endpoint != "" {
		config.HTTP.WebSocketEndpoint = endpoint
	}
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	// Set default HTTP address if not provided
	if config.HTTP.Address == "" {
		config.HTTP.Address = ":8086"
	}

	// Set default gRPC address if not provided
	if config.GRPC.Address == "" {
		config.GRPC.Address = ":9095"
	}

	// Set default timeouts if not provided
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = 10 * time.Second
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = 10 * time.Second
	}
	if config.HTTP.IdleTimeout == 0 {
		config.HTTP.IdleTimeout = 60 * time.Second
	}
	if config.HTTP.ReadHeaderTimeout == 0 {
		config.HTTP.ReadHeaderTimeout = 5 * time.Second
	}
	if config.HTTP.ShutdownTimeout == 0 {
		config.HTTP.ShutdownTimeout = 5 * time.Second
	}

	// Set default WebSocket endpoint
	if config.HTTP.WebSocketEndpoint == "" {
		config.HTTP.WebSocketEndpoint = "/ws"
	}

	// Set default WebSocket path
	if config.HTTP.WebSocketPath == "" {
		config.HTTP.WebSocketPath = "/stream/{stream_id}"
	}

	// Set default gRPC configuration
	if config.GRPC.MaxConcurrentStreams == 0 {
		config.GRPC.MaxConcurrentStreams = 1000
	}
	if config.GRPC.ConnectionTimeout == 0 {
		config.GRPC.ConnectionTimeout = 10 * time.Second
	}
	if config.GRPC.KeepAliveTime == 0 {
		config.GRPC.KeepAliveTime = 30 * time.Second
	}
	if config.GRPC.KeepAliveTimeout == 0 {
		config.GRPC.KeepAliveTimeout = 10 * time.Second
	}

	// Set default WebSocket configuration
	if config.WebSocket.HandshakeTimeout == 0 {
		config.WebSocket.HandshakeTimeout = 10 * time.Second
	}
	if config.WebSocket.WriteWait == 0 {
		config.WebSocket.WriteWait = 10 * time.Second
	}
	if config.WebSocket.PongWait == 0 {
		config.WebSocket.PongWait = 60 * time.Second
	}
	if config.WebSocket.PingPeriod == 0 {
		config.WebSocket.PingPeriod = (config.WebSocket.PongWait * 9) / 10
	}
	if config.WebSocket.MaxMessageSize == 0 {
		config.WebSocket.MaxMessageSize = 1024 * 1024 // 1MB
	}
	if config.WebSocket.BufferSize == 0 {
		config.WebSocket.BufferSize = 1024
	}
	if config.WebSocket.MessageBufferSize == 0 {
		config.WebSocket.MessageBufferSize = 256
	}
	if config.WebSocket.ClientSessionTimeout == 0 {
		config.WebSocket.ClientSessionTimeout = 5 * time.Minute
	}

	// Set default stream configuration
	if config.Stream.MaxClientsPerStream == 0 {
		config.Stream.MaxClientsPerStream = 100
	}
	if config.Stream.StreamTimeout == 0 {
		config.Stream.StreamTimeout = 24 * time.Hour
	}
	if config.Stream.MaxStreams == 0 {
		config.Stream.MaxStreams = 1000
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

	// Set default metrics configuration
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}
	if config.Metrics.CollectPeriod == 0 {
		config.Metrics.CollectPeriod = 15 * time.Second
	}

	// Generate node ID if not provided
	if config.Service.NodeID == "" {
		config.Service.NodeID = fmt.Sprintf("signaling-%d", time.Now().UnixNano()%1000)
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate service name
	if config.Service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Validate HTTP address
	if config.HTTP.Address == "" {
		return fmt.Errorf("HTTP address is required")
	}

	// Validate gRPC address
	if config.GRPC.Address == "" {
		return fmt.Errorf("gRPC address is required")
	}

	// Validate logging configuration
	if config.Logging.Level != "debug" && config.Logging.Level != "info" &&
		config.Logging.Level != "warn" && config.Logging.Level != "error" {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	// Ensure ping period is less than pong wait
	if config.WebSocket.PingPeriod >= config.WebSocket.PongWait {
		return fmt.Errorf("ping period must be less than pong wait")
	}

	return nil
}
