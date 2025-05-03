package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// Config holds all configuration for the API Gateway
type Config struct {
	Environment string          `yaml:"environment"`
	HTTP        HTTPConfig      `yaml:"http"`
	GRPC        GRPCConfig      `yaml:"grpc"`
	WebSocket   WebSocketConfig `yaml:"websocket"`
	Auth        AuthConfig      `yaml:"auth"`
	RateLimit   RateLimitConfig `yaml:"rateLimit"`
	Services    ServicesConfig  `yaml:"services"`
	Logging     LoggingConfig   `yaml:"logging"`
	Tracing     TracingConfig   `yaml:"tracing"`
}

// HTTPConfig holds HTTP server configuration
type HTTPConfig struct {
	Address      string        `yaml:"address"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Address          string        `yaml:"address"`
	KeepAliveTime    time.Duration `yaml:"keepAliveTime"`
	KeepAliveTimeout time.Duration `yaml:"keepAliveTimeout"`
	MaxConnectionAge time.Duration `yaml:"maxConnectionAge"`
	TLS              TLSConfig     `yaml:"tls"`
}

// WebSocketConfig holds WebSocket server configuration
type WebSocketConfig struct {
	Address      string        `yaml:"address"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret        string        `yaml:"jwtSecret"`
	JWTExpiration    time.Duration `yaml:"jwtExpiration"`
	EnableAPIKeys    bool          `yaml:"enableApiKeys"`
	APIKeyHeaderName string        `yaml:"apiKeyHeaderName"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled        bool          `yaml:"enabled"`
	RequestsPerMin int           `yaml:"requestsPerMin"`
	BurstSize      int           `yaml:"burstSize"`
	ExpirationTime time.Duration `yaml:"expirationTime"`
}

// ServiceConfig holds a single service configuration
type ServiceConfig struct {
	Name          string        `yaml:"name"`
	HTTPEndpoints []string      `yaml:"httpEndpoints"`
	GRPCEndpoints []string      `yaml:"grpcEndpoints"`
	HealthCheck   string        `yaml:"healthCheck"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryAttempts int           `yaml:"retryAttempts"`
	RetryDelay    time.Duration `yaml:"retryDelay"`
}

// ServicesConfig holds all service configurations
type ServicesConfig struct {
	Discovery       string                   `yaml:"discovery"` // "static", "consul", "etcd", etc.
	ConsulAddress   string                   `yaml:"consulAddress"`
	EtcdEndpoints   []string                 `yaml:"etcdEndpoints"`
	RefreshInterval time.Duration            `yaml:"refreshInterval"`
	Services        map[string]ServiceConfig `yaml:"services"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level"`
	Format   string `yaml:"format"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"filePath"`
}

// TracingConfig holds distributed tracing configuration
type TracingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Endpoint    string `yaml:"endpoint"`
	ServiceName string `yaml:"serviceName"`
}

// Load reads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set defaults for any missing values
	setDefaults(&cfg)

	// Validate the configuration
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets default values for missing configuration
func setDefaults(cfg *Config) {
	// Set default environment
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	// Set default HTTP config
	if cfg.HTTP.Address == "" {
		cfg.HTTP.Address = ":8080"
	}
	if cfg.HTTP.ReadTimeout == 0 {
		cfg.HTTP.ReadTimeout = 10 * time.Second
	}
	if cfg.HTTP.WriteTimeout == 0 {
		cfg.HTTP.WriteTimeout = 10 * time.Second
	}
	if cfg.HTTP.IdleTimeout == 0 {
		cfg.HTTP.IdleTimeout = 120 * time.Second
	}

	// Set default gRPC config
	if cfg.GRPC.Address == "" {
		cfg.GRPC.Address = ":50051"
	}
	if cfg.GRPC.KeepAliveTime == 0 {
		cfg.GRPC.KeepAliveTime = 30 * time.Second
	}
	if cfg.GRPC.KeepAliveTimeout == 0 {
		cfg.GRPC.KeepAliveTimeout = 10 * time.Second
	}
	if cfg.GRPC.MaxConnectionAge == 0 {
		cfg.GRPC.MaxConnectionAge = 30 * time.Minute
	}

	// Set default WebSocket config
	if cfg.WebSocket.Address == "" {
		cfg.WebSocket.Address = ":8082"
	}
	if cfg.WebSocket.ReadTimeout == 0 {
		cfg.WebSocket.ReadTimeout = 60 * time.Second
	}
	if cfg.WebSocket.WriteTimeout == 0 {
		cfg.WebSocket.WriteTimeout = 60 * time.Second
	}
	if cfg.WebSocket.IdleTimeout == 0 {
		cfg.WebSocket.IdleTimeout = 300 * time.Second
	}

	// Set default auth config
	if cfg.Auth.JWTExpiration == 0 {
		cfg.Auth.JWTExpiration = 24 * time.Hour
	}
	if cfg.Auth.APIKeyHeaderName == "" {
		cfg.Auth.APIKeyHeaderName = "X-API-Key"
	}

	// Set default rate limit config
	if cfg.RateLimit.RequestsPerMin == 0 {
		cfg.RateLimit.RequestsPerMin = 60
	}
	if cfg.RateLimit.BurstSize == 0 {
		cfg.RateLimit.BurstSize = 10
	}
	if cfg.RateLimit.ExpirationTime == 0 {
		cfg.RateLimit.ExpirationTime = 1 * time.Hour
	}

	// Set default services config
	if cfg.Services.Discovery == "" {
		cfg.Services.Discovery = "static"
	}
	if cfg.Services.RefreshInterval == 0 {
		cfg.Services.RefreshInterval = 30 * time.Second
	}

	// Set default logging config
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	// Set default tracing config
	if cfg.Tracing.ServiceName == "" {
		cfg.Tracing.ServiceName = "api-gateway"
	}
}

// validate checks configuration for required values and correct formats
func validate(cfg *Config) error {
	// Validate required fields
	if cfg.HTTP.Address == "" {
		return fmt.Errorf("HTTP address is required")
	}
	if cfg.GRPC.Address == "" {
		return fmt.Errorf("gRPC address is required")
	}

	// Validate JWT secret if authentication is enabled
	if cfg.Auth.JWTSecret == "" && cfg.Auth.JWTExpiration > 0 {
		return fmt.Errorf("JWT secret is required when JWT authentication is enabled")
	}

	// Validate TLS configuration if enabled for HTTP
	if cfg.HTTP.TLS.Enabled {
		if cfg.HTTP.TLS.CertFile == "" || cfg.HTTP.TLS.KeyFile == "" {
			return fmt.Errorf("TLS cert file and key file are required when TLS is enabled for HTTP")
		}
	}

	// Validate TLS configuration if enabled for gRPC
	if cfg.GRPC.TLS.Enabled {
		if cfg.GRPC.TLS.CertFile == "" || cfg.GRPC.TLS.KeyFile == "" {
			return fmt.Errorf("TLS cert file and key file are required when TLS is enabled for gRPC")
		}
	}

	// Validate service discovery configuration
	switch cfg.Services.Discovery {
	case "consul":
		if cfg.Services.ConsulAddress == "" {
			return fmt.Errorf("consul address is required when consul service discovery is enabled")
		}
	case "etcd":
		if len(cfg.Services.EtcdEndpoints) == 0 {
			return fmt.Errorf("etcd endpoints are required when etcd service discovery is enabled")
		}
	}

	return nil
}
