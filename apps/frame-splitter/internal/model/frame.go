package model

import (
	"time"
)

// FrameType represents the type of a frame
type FrameType string

const (
	// FrameTypeUnknown represents an unknown frame type
	FrameTypeUnknown FrameType = "UNKNOWN"

	// FrameTypeVideo represents a video frame
	FrameTypeVideo FrameType = "VIDEO"

	// FrameTypeAudio represents an audio frame
	FrameTypeAudio FrameType = "AUDIO"

	// FrameTypeMetadata represents a metadata frame
	FrameTypeMetadata FrameType = "METADATA"
)

// Frame represents a video or audio frame
type Frame struct {
	// Frame identification
	StreamID  string    `json:"stream_id"`
	FrameID   string    `json:"frame_id"`
	Timestamp time.Time `json:"timestamp"`

	// Frame type
	Type FrameType `json:"type"`

	// Frame data
	Data []byte `json:"data"`

	// Frame metadata
	Metadata map[string]string `json:"metadata"`

	// Frame sequence number
	Sequence int64 `json:"sequence"`

	// Is key frame
	IsKeyFrame bool `json:"is_key_frame"`

	// Duration in milliseconds
	Duration int32 `json:"duration"`

	// Processing metadata
	ProcessedAt    time.Time     `json:"processed_at"`
	ProcessingTime time.Duration `json:"processing_time"`
	Status         string        `json:"status"`
}

// FrameBatch represents a batch of frames
type FrameBatch struct {
	StreamID string  `json:"stream_id"`
	Frames   []Frame `json:"frames"`
}

// RoutingRule represents a routing rule for frames
type RoutingRule struct {
	Destination string `json:"destination"`
	Filter      string `json:"filter"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	BatchSize   int    `json:"batch_size"`
}

// StreamConfig represents the configuration for a stream
type StreamConfig struct {
	StreamID      string        `json:"stream_id"`
	RoutingRules  []RoutingRule `json:"routing_rules"`
	BackupEnabled bool          `json:"backup_enabled"`
	StoragePath   string        `json:"storage_path"`
	RetentionTime time.Duration `json:"retention_time"`
}

// SubscriptionRequest represents a request to subscribe to frames
type SubscriptionRequest struct {
	StreamID     string    `json:"stream_id"`
	SubscriberID string    `json:"subscriber_id"`
	Type         FrameType `json:"type"`
	RoutingKey   string    `json:"routing_key"`
}

// FrameFilter represents a filter for frames
type FrameFilter struct {
	Type       FrameType `json:"type"`
	IsKeyFrame *bool     `json:"is_key_frame,omitempty"`
	MinSize    *int      `json:"min_size,omitempty"`
	MaxSize    *int      `json:"max_size,omitempty"`
}

// StreamStats represents statistics for a stream
type StreamStats struct {
	StreamID          string        `json:"stream_id"`
	FramesProcessed   int64         `json:"frames_processed"`
	BytesProcessed    int64         `json:"bytes_processed"`
	FramesDropped     int64         `json:"frames_dropped"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	LastProcessedAt   time.Time     `json:"last_processed_at"`
	LastKeyFrameAt    time.Time     `json:"last_key_frame_at"`
	VideoFrameRate    float64       `json:"video_frame_rate"`
	AudioFrameRate    float64       `json:"audio_frame_rate"`
	VideoBitrate      int64         `json:"video_bitrate"`
	AudioBitrate      int64         `json:"audio_bitrate"`
}

// ProcessingResult represents the result of processing a frame
type ProcessingResult struct {
	FrameID      string            `json:"frame_id"`
	Status       string            `json:"status"`
	Error        string            `json:"error,omitempty"`
	Destinations []string          `json:"destinations"`
	Metadata     map[string]string `json:"metadata"`
}

// BatchProcessingResult represents the result of processing a batch of frames
type BatchProcessingResult struct {
	Results []ProcessingResult `json:"results"`
}

// BackupInfo represents information about a backup
type BackupInfo struct {
	FrameID      string    `json:"frame_id"`
	StreamID     string    `json:"stream_id"`
	Type         FrameType `json:"type"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	BackupTime   time.Time `json:"backup_time"`
	Compressed   bool      `json:"compressed"`
	OriginalSize int64     `json:"original_size"`
}
