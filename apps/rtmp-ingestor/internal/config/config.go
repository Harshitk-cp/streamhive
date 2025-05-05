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

	// HTTP server configuration for metrics and health checks
	HTTP struct {
		Address         string        `yaml:"address"`
		ReadTimeout     time.Duration `yaml:"read_timeout"`
		WriteTimeout    time.Duration `yaml:"write_timeout"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	} `yaml:"http"`

	// RTMP server configuration
	RTMP struct {
		Address          string        `yaml:"address"`
		ChunkSize        int           `yaml:"chunk_size"`
		BufferSize       int           `yaml:"buffer_size"`
		ReadTimeout      time.Duration `yaml:"read_timeout"`
		WriteTimeout     time.Duration `yaml:"write_timeout"`
		GopCacheEnabled  bool          `yaml:"gop_cache_enabled"`
		GopCacheMaxItems int           `yaml:"gop_cache_max_items"`
		KeyFrameOnly     bool          `yaml:"key_frame_only"`
	} `yaml:"rtmp"`

	// Stream Router service configuration
	Router struct {
		Address      string        `yaml:"address"`
		Timeout      time.Duration `yaml:"timeout"`
		MaxRetries   int           `yaml:"max_retries"`
		RetryBackoff time.Duration `yaml:"retry_backoff"`
	} `yaml:"router"`

	// Frame Splitter service configuration
	FrameSplitter struct {
		Address   string        `yaml:"address"`
		BatchSize int           `yaml:"batch_size"`
		Timeout   time.Duration `yaml:"timeout"`
	} `yaml:"frame_splitter"`

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

	// Local storage configuration for backup frames
	Storage struct {
		Enabled        bool   `yaml:"enabled"`
		Path           string `yaml:"path"`
		MaxSizeGB      int    `yaml:"max_size_gb"`
		RetentionHours int    `yaml:"retention_hours"`
	} `yaml:"storage"`
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

	// Override RTMP address
	if addr := os.Getenv("RTMP_ADDRESS"); addr != "" {
		config.RTMP.Address = addr
	}

	// Override Router address
	if addr := os.Getenv("ROUTER_ADDRESS"); addr != "" {
		config.Router.Address = addr
	}

	// Override Frame Splitter address
	if addr := os.Getenv("FRAME_SPLITTER_ADDRESS"); addr != "" {
		config.FrameSplitter.Address = addr
	}

	// Override node ID for horizontal scaling
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		config.Service.NodeID = nodeID
	}

	// Override environment
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Service.Environment = env
	}

	// Override Frame Splitter configuration
	if addr := os.Getenv("FRAME_SPLITTER_ADDRESS"); addr != "" {
		config.FrameSplitter.Address = addr
	}
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	// Set default HTTP address if not provided
	if config.HTTP.Address == "" {
		config.HTTP.Address = ":8081" // Different from stream-router
	}

	// Set default RTMP address if not provided
	if config.RTMP.Address == "" {
		config.RTMP.Address = ":1935" // Standard RTMP port
	}

	// Set default timeouts if not provided
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = 10 * time.Second
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = 10 * time.Second
	}
	if config.HTTP.ShutdownTimeout == 0 {
		config.HTTP.ShutdownTimeout = 5 * time.Second
	}

	// Set default RTMP configuration
	if config.RTMP.ChunkSize == 0 {
		config.RTMP.ChunkSize = 4096
	}
	if config.RTMP.BufferSize == 0 {
		config.RTMP.BufferSize = 4 * 1024 * 1024 // 4MB
	}
	if config.RTMP.ReadTimeout == 0 {
		config.RTMP.ReadTimeout = 30 * time.Second
	}
	if config.RTMP.WriteTimeout == 0 {
		config.RTMP.WriteTimeout = 30 * time.Second
	}
	if config.RTMP.GopCacheMaxItems == 0 {
		config.RTMP.GopCacheMaxItems = 1024
	}

	// Set default Router configuration
	if config.Router.Timeout == 0 {
		config.Router.Timeout = 5 * time.Second
	}
	if config.Router.MaxRetries == 0 {
		config.Router.MaxRetries = 3
	}
	if config.Router.RetryBackoff == 0 {
		config.Router.RetryBackoff = 1 * time.Second
	}

	// Set default logging level if not provided
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	// Set default logging format if not provided
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	// Set default metrics configuration
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}
	if config.Metrics.CollectPeriod == 0 {
		config.Metrics.CollectPeriod = 15 * time.Second
	}

	// Set default storage configuration
	if config.Storage.Enabled && config.Storage.Path == "" {
		config.Storage.Path = "./data"
	}
	if config.Storage.MaxSizeGB == 0 {
		config.Storage.MaxSizeGB = 10
	}
	if config.Storage.RetentionHours == 0 {
		config.Storage.RetentionHours = 24
	}

	// Generate node ID if not provided
	if config.Service.NodeID == "" {
		config.Service.NodeID = fmt.Sprintf("rtmp-ingestor-%d", time.Now().UnixNano()%1000)
	}

	// Set default Frame Splitter configuration
	if config.FrameSplitter.Address == "" {
		config.FrameSplitter.Address = "frame-splitter:9091"
	}
	if config.FrameSplitter.BatchSize == 0 {
		config.FrameSplitter.BatchSize = 10
	}
	if config.FrameSplitter.Timeout == 0 {
		config.FrameSplitter.Timeout = 5 * time.Second
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate service name
	if config.Service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Validate RTMP address
	if config.RTMP.Address == "" {
		return fmt.Errorf("RTMP address is required")
	}

	// Validate Router address
	if config.Router.Address == "" {
		return fmt.Errorf("router address is required")
	}

	// Validate Frame Splitter address
	if config.FrameSplitter.Address == "" {
		return fmt.Errorf("frame Splitter address is required")
	}

	// Validate logging configuration
	if config.Logging.Level != "debug" && config.Logging.Level != "info" && config.Logging.Level != "warn" && config.Logging.Level != "error" {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	return nil
}
