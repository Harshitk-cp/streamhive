// apps/webrtc-out/internal/config/config.go
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

	// gRPC server configuration
	GRPC struct {
		Address              string        `yaml:"address"`
		MaxConcurrentStreams int           `yaml:"max_concurrent_streams"`
		ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
		KeepAliveTime        time.Duration `yaml:"keep_alive_time"`
		KeepAliveTimeout     time.Duration `yaml:"keep_alive_timeout"`
	} `yaml:"grpc"`

	// WebRTC configuration
	WebRTC struct {
		ICEServers            []ICEServer   `yaml:"ice_servers"`
		TimeoutDuration       time.Duration `yaml:"timeout_duration"`
		MaxBufferedFrames     int           `yaml:"max_buffered_frames"`
		VideoFrameRate        int           `yaml:"video_frame_rate"`
		VideoKeyFrameInterval int           `yaml:"video_key_frame_interval"`
		AudioSampleRate       int           `yaml:"audio_sample_rate"`
		AudioChannels         int           `yaml:"audio_channels"`
		OpusFrameDuration     time.Duration `yaml:"opus_frame_duration"`
		TrickleICE            bool          `yaml:"trickle_ice"`
		CodecPreferences      []string      `yaml:"codec_preferences"`
		BWEMode               string        `yaml:"bwe_mode"`
		EnableDTX             bool          `yaml:"enable_dtx"`
		SignalingService      string        `yaml:"signaling_service"`
	} `yaml:"webrtc"`

	// Frame processing configuration
	FrameProcessing struct {
		BatchSize         int  `yaml:"batch_size"`
		MaxQueueSize      int  `yaml:"max_queue_size"`
		DropWhenFull      bool `yaml:"drop_when_full"`
		ProcessingThreads int  `yaml:"processing_threads"`
	} `yaml:"frame_processing"`

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

// ICEServer represents a STUN/TURN server configuration
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

	// Override Signaling Service address
	if addr := os.Getenv("SIGNALING_SERVICE"); addr != "" {
		config.WebRTC.SignalingService = addr
	}

	// Override environment
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Service.Environment = env
	}

	// Override node ID for horizontal scaling
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		config.Service.NodeID = nodeID
	}
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	// Set default HTTP address if not provided
	if config.HTTP.Address == "" {
		config.HTTP.Address = ":8085"
	}

	// Set default gRPC address if not provided
	if config.GRPC.Address == "" {
		config.GRPC.Address = ":9094"
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

	// Set default WebRTC configuration
	if len(config.WebRTC.ICEServers) == 0 {
		config.WebRTC.ICEServers = []ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		}
	}
	if config.WebRTC.TimeoutDuration == 0 {
		config.WebRTC.TimeoutDuration = 30 * time.Second
	}
	if config.WebRTC.MaxBufferedFrames == 0 {
		config.WebRTC.MaxBufferedFrames = 120
	}
	if config.WebRTC.VideoFrameRate == 0 {
		config.WebRTC.VideoFrameRate = 30
	}
	if config.WebRTC.VideoKeyFrameInterval == 0 {
		config.WebRTC.VideoKeyFrameInterval = 60
	}
	if config.WebRTC.AudioSampleRate == 0 {
		config.WebRTC.AudioSampleRate = 48000
	}
	if config.WebRTC.AudioChannels == 0 {
		config.WebRTC.AudioChannels = 2
	}
	if config.WebRTC.OpusFrameDuration == 0 {
		config.WebRTC.OpusFrameDuration = 20 * time.Millisecond
	}
	if config.WebRTC.BWEMode == "" {
		config.WebRTC.BWEMode = "dynamic"
	}
	if len(config.WebRTC.CodecPreferences) == 0 {
		config.WebRTC.CodecPreferences = []string{"VP8", "H264", "Opus"}
	}
	if config.WebRTC.SignalingService == "" {
		config.WebRTC.SignalingService = "websocket-signaling:9095"
	}

	// Set default frame processing configuration
	if config.FrameProcessing.BatchSize == 0 {
		config.FrameProcessing.BatchSize = 10
	}
	if config.FrameProcessing.MaxQueueSize == 0 {
		config.FrameProcessing.MaxQueueSize = 1000
	}
	if config.FrameProcessing.ProcessingThreads == 0 {
		config.FrameProcessing.ProcessingThreads = 4
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
		config.Service.NodeID = fmt.Sprintf("webrtc-out-%d", time.Now().UnixNano()%1000)
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

	return nil
}
