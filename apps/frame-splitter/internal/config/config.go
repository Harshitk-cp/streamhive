package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load loads the application configuration from a file
func Load(path string) (*Config, error) {
	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvironmentOverrides(&cfg)

	// Set defaults
	setDefaults(&cfg)

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyEnvironmentOverrides applies environment variable overrides to the configuration
func applyEnvironmentOverrides(cfg *Config) {
	// Apply service environment variable overrides
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Service.Environment = env
	}
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		cfg.Service.NodeID = nodeID
	}

	// Apply HTTP server environment variable overrides
	if addr := os.Getenv("HTTP_ADDRESS"); addr != "" {
		cfg.HTTP.Address = addr
	}

	// Apply gRPC server environment variable overrides
	if addr := os.Getenv("GRPC_ADDRESS"); addr != "" {
		cfg.GRPC.Address = addr
	}

	// Apply router environment variable overrides
	if addr := os.Getenv("ROUTER_ADDRESS"); addr != "" {
		cfg.Router.Address = addr
	}

	// Apply routing environment variable overrides
	if addr := os.Getenv("ENHANCEMENT_SERVICE_ADDRESS"); addr != "" {
		cfg.Routing.EnhancementService.Address = addr
	}
	if addr := os.Getenv("ENCODER_SERVICE_ADDRESS"); addr != "" {
		cfg.Routing.EncoderService.Address = addr
	}
	if addr := os.Getenv("WEBRTC_SERVICE_ADDRESS"); addr != "" {
		cfg.Routing.WebRTCOut.Address = addr
	}

	// Apply backup environment variable overrides
	if path := os.Getenv("BACKUP_STORAGE_PATH"); path != "" {
		cfg.Backup.StoragePath = path
	}
}

// setDefaults sets default values for the configuration
func setDefaults(cfg *Config) {
	// Set default service values
	if cfg.Service.Name == "" {
		cfg.Service.Name = "frame-splitter"
	}
	if cfg.Service.Version == "" {
		cfg.Service.Version = "0.1.0"
	}
	if cfg.Service.Environment == "" {
		cfg.Service.Environment = "development"
	}
	if cfg.Service.NodeID == "" {
		// Generate a random node ID based on timestamp
		cfg.Service.NodeID = fmt.Sprintf("node-%d", time.Now().UnixNano()%1000)
	}

	// Set default HTTP server values
	if cfg.HTTP.Address == "" {
		cfg.HTTP.Address = ":8084"
	}
	if cfg.HTTP.ReadTimeout == 0 {
		cfg.HTTP.ReadTimeout = 10 * time.Second
	}
	if cfg.HTTP.WriteTimeout == 0 {
		cfg.HTTP.WriteTimeout = 10 * time.Second
	}
	if cfg.HTTP.ShutdownTimeout == 0 {
		cfg.HTTP.ShutdownTimeout = 5 * time.Second
	}

	// Set default gRPC server values
	if cfg.GRPC.Address == "" {
		cfg.GRPC.Address = ":9091"
	}
	if cfg.GRPC.MaxConcurrentStreams == 0 {
		cfg.GRPC.MaxConcurrentStreams = 1000
	}
	if cfg.GRPC.ConnectionTimeout == 0 {
		cfg.GRPC.ConnectionTimeout = 10 * time.Second
	}
	if cfg.GRPC.KeepAliveTime == 0 {
		cfg.GRPC.KeepAliveTime = 30 * time.Second
	}
	if cfg.GRPC.KeepAliveTimeout == 0 {
		cfg.GRPC.KeepAliveTimeout = 10 * time.Second
	}

	// Set default router values
	if cfg.Router.Timeout == 0 {
		cfg.Router.Timeout = 5 * time.Second
	}
	if cfg.Router.MaxRetries == 0 {
		cfg.Router.MaxRetries = 3
	}
	if cfg.Router.RetryBackoff == 0 {
		cfg.Router.RetryBackoff = 1 * time.Second
	}

	// Set default frame processing values
	if cfg.FrameProcessing.BatchSize == 0 {
		cfg.FrameProcessing.BatchSize = 30
	}
	if cfg.FrameProcessing.MaxQueueSize == 0 {
		cfg.FrameProcessing.MaxQueueSize = 10000
	}
	if cfg.FrameProcessing.ProcessingThreads == 0 {
		cfg.FrameProcessing.ProcessingThreads = 8
	}
	if cfg.FrameProcessing.MaxBatchIntervalMs == 0 {
		cfg.FrameProcessing.MaxBatchIntervalMs = 100
	}

	// Set default backup values
	if cfg.Backup.StoragePath == "" {
		cfg.Backup.StoragePath = "./data/frames"
	}
	if cfg.Backup.RetentionMinutes == 0 {
		cfg.Backup.RetentionMinutes = 1440 // 24 hours
	}
	if cfg.Backup.CompressionLevel == 0 {
		cfg.Backup.CompressionLevel = 6
	}

	// Set default monitoring values
	if cfg.Monitoring.MetricsPath == "" {
		cfg.Monitoring.MetricsPath = "/metrics"
	}
	if cfg.Monitoring.TracingService == "" {
		cfg.Monitoring.TracingService = "frame-splitter"
	}
	if cfg.Monitoring.HealthCheckInterval == 0 {
		cfg.Monitoring.HealthCheckInterval = 15 * time.Second
	}

	// Set default logging values
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
	if cfg.Logging.MaxSizeMB == 0 {
		cfg.Logging.MaxSizeMB = 100
	}
	if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 3
	}
	if cfg.Logging.MaxAgeDays == 0 {
		cfg.Logging.MaxAgeDays = 7
	}
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	// Validate service configuration
	if cfg.Service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Validate server configuration
	if cfg.HTTP.Address == "" {
		return fmt.Errorf("HTTP address is required")
	}
	if cfg.GRPC.Address == "" {
		return fmt.Errorf("gRPC address is required")
	}

	// Validate router configuration
	if cfg.Router.Address == "" {
		return fmt.Errorf("router address is required")
	}

	// Validate frame processing configuration
	if cfg.FrameProcessing.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if cfg.FrameProcessing.MaxQueueSize <= 0 {
		return fmt.Errorf("max queue size must be positive")
	}
	if cfg.FrameProcessing.ProcessingThreads <= 0 {
		return fmt.Errorf("processing threads must be positive")
	}

	// Validate backup configuration
	if cfg.Backup.Enabled && cfg.Backup.StoragePath == "" {
		return fmt.Errorf("backup storage path is required when backup is enabled")
	}

	// Validate logging configuration
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if _, ok := validLevels[cfg.Logging.Level]; !ok {
		return fmt.Errorf("invalid logging level: %s", cfg.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if _, ok := validFormats[cfg.Logging.Format]; !ok {
		return fmt.Errorf("invalid logging format: %s", cfg.Logging.Format)
	}

	validOutputs := map[string]bool{"stdout": true, "file": true}
	if _, ok := validOutputs[cfg.Logging.Output]; !ok {
		return fmt.Errorf("invalid logging output: %s", cfg.Logging.Output)
	}

	return nil
}

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

	// HTTP server configuration
	HTTP struct {
		Address         string        `yaml:"address"`
		ReadTimeout     time.Duration `yaml:"read_timeout"`
		WriteTimeout    time.Duration `yaml:"write_timeout"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	} `yaml:"http"`

	// gRPC server configuration
	GRPC struct {
		Address              string        `yaml:"address"`
		MaxConcurrentStreams int           `yaml:"max_concurrent_streams"`
		ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
		KeepAliveTime        time.Duration `yaml:"keep_alive_time"`
		KeepAliveTimeout     time.Duration `yaml:"keep_alive_timeout"`
	} `yaml:"grpc"`

	// Stream router configuration
	Router struct {
		Address      string        `yaml:"address"`
		Timeout      time.Duration `yaml:"timeout"`
		MaxRetries   int           `yaml:"max_retries"`
		RetryBackoff time.Duration `yaml:"retry_backoff"`
	} `yaml:"router"`

	// Frame processing configuration
	FrameProcessing struct {
		BatchSize          int  `yaml:"batch_size"`
		MaxQueueSize       int  `yaml:"max_queue_size"`
		DropFramesWhenFull bool `yaml:"drop_frames_when_full"`
		ProcessingThreads  int  `yaml:"processing_threads"`
		MaxBatchIntervalMs int  `yaml:"max_batch_interval_ms"`
	} `yaml:"frame_processing"`

	// Backup configuration
	Backup struct {
		Enabled            bool   `yaml:"enabled"`
		StoragePath        string `yaml:"storage_path"`
		RetentionMinutes   int    `yaml:"retention_minutes"`
		CompressionEnabled bool   `yaml:"compression_enabled"`
		CompressionLevel   int    `yaml:"compression_level"`
	} `yaml:"backup"`

	// Routing configuration
	Routing struct {
		EnhancementService struct {
			Address   string `yaml:"address"`
			Enabled   bool   `yaml:"enabled"`
			Filter    string `yaml:"filter"`
			BatchSize int    `yaml:"batch_size"`
			Priority  int    `yaml:"priority"`
		} `yaml:"enhancement_service"`

		EncoderService struct {
			Address   string `yaml:"address"`
			Enabled   bool   `yaml:"enabled"`
			Filter    string `yaml:"filter"`
			BatchSize int    `yaml:"batch_size"`
			Priority  int    `yaml:"priority"`
		} `yaml:"encoder_service"`

		WebRTCOut struct {
			Address   string `yaml:"address"`
			Enabled   bool   `yaml:"enabled"`
			Filter    string `yaml:"filter"`
			BatchSize int    `yaml:"batch_size"`
			Priority  int    `yaml:"priority"`
		} `yaml:"webrtc_out"`
	} `yaml:"routing"`

	// Monitoring configuration
	Monitoring struct {
		MetricsEnabled      bool          `yaml:"metrics_enabled"`
		MetricsPath         string        `yaml:"metrics_path"`
		TracingEnabled      bool          `yaml:"tracing_enabled"`
		TracingService      string        `yaml:"tracing_service"`
		HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	} `yaml:"monitoring"`

	// Logging configuration
	Logging struct {
		Level      string `yaml:"level"`
		Format     string `yaml:"format"`
		Output     string `yaml:"output"`
		FilePath   string `yaml:"file_path"`
		MaxSizeMB  int    `yaml:"max_size_mb"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAgeDays int    `yaml:"max_age_days"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"logging"`
}
