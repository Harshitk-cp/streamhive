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

	// HTTP server configuration
	HTTP struct {
		Address         string        `yaml:"address"`
		ReadTimeout     time.Duration `yaml:"read_timeout"`
		WriteTimeout    time.Duration `yaml:"write_timeout"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
		CORSAllowOrigin string        `yaml:"cors_allow_origin"`
		APIPrefix       string        `yaml:"api_prefix"`
	} `yaml:"http"`

	// gRPC server configuration
	GRPC struct {
		Address string `yaml:"address"`
	} `yaml:"grpc"`

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

	// Database configuration
	Database struct {
		Type     string `yaml:"type"` // memory, postgres, etc.
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		SSLMode  string `yaml:"ssl_mode"`
		MaxConns int    `yaml:"max_conns"`
	} `yaml:"database"`

	// Redis configuration for caching
	Redis struct {
		Enabled  bool   `yaml:"enabled"`
		Address  string `yaml:"address"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	// Webhook configuration
	Webhook struct {
		MaxRetries   int           `yaml:"max_retries"`
		RetryBackoff time.Duration `yaml:"retry_backoff"`
		Timeout      time.Duration `yaml:"timeout"`
	} `yaml:"webhook"`

	// Service discovery
	Discovery struct {
		Enabled  bool   `yaml:"enabled"`
		Provider string `yaml:"provider"` // consul, etcd, etc.
		Address  string `yaml:"address"`
	} `yaml:"discovery"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret        string        `yaml:"jwtSecret"`
	JWTExpiration    time.Duration `yaml:"jwtExpiration"`
	EnableAPIKeys    bool          `yaml:"enableApiKeys"`
	APIKeyHeaderName string        `yaml:"apiKeyHeaderName"`
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

	// Override database configuration
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		config.Database.Type = dbType
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		config.Database.Username = dbUser
	}
	if dbPass := os.Getenv("DB_PASS"); dbPass != "" {
		config.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		config.Database.Database = dbName
	}

	// Override environment
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Service.Environment = env
	}
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	// Set default HTTP address if not provided
	if config.HTTP.Address == "" {
		config.HTTP.Address = ":8081"
	}

	// Set default gRPC address if not provided
	if config.GRPC.Address == "" {
		config.GRPC.Address = ":9090"
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

	// Set default API prefix if not provided
	if config.HTTP.APIPrefix == "" {
		config.HTTP.APIPrefix = "/api/v1"
	}

	// Set default logging level if not provided
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	// Set default logging format if not provided
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	// Set default webhook retry settings
	if config.Webhook.MaxRetries == 0 {
		config.Webhook.MaxRetries = 3
	}
	if config.Webhook.RetryBackoff == 0 {
		config.Webhook.RetryBackoff = 5 * time.Second
	}
	if config.Webhook.Timeout == 0 {
		config.Webhook.Timeout = 10 * time.Second
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate service name
	if config.Service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Validate logging configuration
	if config.Logging.Level != "debug" && config.Logging.Level != "info" && config.Logging.Level != "warn" && config.Logging.Level != "error" {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	// Validate database configuration if not using memory
	if config.Database.Type != "memory" && config.Database.Type != "" {
		if config.Database.Host == "" {
			return fmt.Errorf("database host is required")
		}
		if config.Database.Username == "" {
			return fmt.Errorf("database username is required")
		}
		if config.Database.Database == "" {
			return fmt.Errorf("database name is required")
		}
	}

	return nil
}
