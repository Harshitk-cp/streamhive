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
	} `yaml:"http"`

	// gRPC server configuration
	GRPC struct {
		Address              string        `yaml:"address"`
		KeepAliveTime        time.Duration `yaml:"keep_alive_time"`
		KeepAliveTimeout     time.Duration `yaml:"keep_alive_timeout"`
		MaxConcurrentStreams int           `yaml:"max_concurrent_streams"`
	} `yaml:"grpc"`

	// WebRTC configuration
	WebRTC struct {
		ICEServers        []ICEServer   `yaml:"ice_servers"`
		MaxBitrate        int           `yaml:"max_bitrate"`
		MaxFrameRate      int           `yaml:"max_frame_rate"`
		JitterBuffer      int           `yaml:"jitter_buffer"`
		OpusMinBitrate    int           `yaml:"opus_min_bitrate"`
		OpusMaxBitrate    int           `yaml:"opus_max_bitrate"`
		OpusComplexity    int           `yaml:"opus_complexity"`
		MaxStreamLifetime time.Duration `yaml:"max_stream_lifetime"`
	} `yaml:"webrtc"`

	// Stream Router service configuration
	Router struct {
		Address string `yaml:"address"`
	} `yaml:"router"`

	// Frame Splitter service configuration
	FrameSplitter struct {
		Address string `yaml:"address"`
	} `yaml:"frame_splitter"`

	// WebSocket Signaling service configuration
	Signaling struct {
		Address string `yaml:"address"`
	} `yaml:"signaling"`

	// Logging configuration
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"logging"`
}

// ICEServer represents a WebRTC ICE server configuration
type ICEServer struct {
	URLs       []string `yaml:"urls"`
	Username   string   `yaml:"username"`
	Credential string   `yaml:"credential"`
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

	// gRPC address
	if addr := os.Getenv("GRPC_ADDRESS"); addr != "" {
		config.GRPC.Address = addr
	}

	// Router address
	if addr := os.Getenv("ROUTER_ADDRESS"); addr != "" {
		config.Router.Address = addr
	}

	// Frame Splitter address
	if addr := os.Getenv("FRAME_SPLITTER_ADDRESS"); addr != "" {
		config.FrameSplitter.Address = addr
	}

	// Signaling address
	if addr := os.Getenv("SIGNALING_ADDRESS"); addr != "" {
		config.Signaling.Address = addr
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
		config.HTTP.Address = ":8088"
	}

	// Set default gRPC address
	if config.GRPC.Address == "" {
		config.GRPC.Address = ":50053"
	}

	// Set default Router address
	if config.Router.Address == "" {
		config.Router.Address = "stream-router:9090"
	}

	// Set default Frame Splitter address
	if config.FrameSplitter.Address == "" {
		config.FrameSplitter.Address = "frame-splitter:9091"
	}

	// Set default Signaling address
	if config.Signaling.Address == "" {
		config.Signaling.Address = "websocket-signaling:50052"
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

	// Set default WebRTC configuration
	if len(config.WebRTC.ICEServers) == 0 {
		config.WebRTC.ICEServers = []ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		}
	}
	if config.WebRTC.MaxBitrate == 0 {
		config.WebRTC.MaxBitrate = 2000000 // 2 Mbps
	}
	if config.WebRTC.MaxFrameRate == 0 {
		config.WebRTC.MaxFrameRate = 30
	}
	if config.WebRTC.JitterBuffer == 0 {
		config.WebRTC.JitterBuffer = 50 // ms
	}
	if config.WebRTC.OpusMinBitrate == 0 {
		config.WebRTC.OpusMinBitrate = 6000 // 6 kbps
	}
	if config.WebRTC.OpusMaxBitrate == 0 {
		config.WebRTC.OpusMaxBitrate = 128000 // 128 kbps
	}
	if config.WebRTC.OpusComplexity == 0 {
		config.WebRTC.OpusComplexity = 10
	}
	if config.WebRTC.MaxStreamLifetime == 0 {
		config.WebRTC.MaxStreamLifetime = 8 * time.Hour
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
