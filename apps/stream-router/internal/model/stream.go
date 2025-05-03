package model

import (
	"time"
)

// StreamStatus represents the status of a stream
type StreamStatus string

const (
	// StreamStatusIdle means the stream is registered but not active
	StreamStatusIdle StreamStatus = "idle"

	// StreamStatusLive means the stream is currently live
	StreamStatusLive StreamStatus = "live"

	// StreamStatusEnded means the stream has ended
	StreamStatusEnded StreamStatus = "ended"

	// StreamStatusError means the stream encountered an error
	StreamStatusError StreamStatus = "error"
)

// StreamVisibility represents the visibility of a stream
type StreamVisibility string

const (
	// StreamVisibilityPublic means the stream is visible to everyone
	StreamVisibilityPublic StreamVisibility = "public"

	// StreamVisibilityPrivate means the stream is only visible to authorized users
	StreamVisibilityPrivate StreamVisibility = "private"

	// StreamVisibilityUnlisted means the stream is not listed but accessible with direct link
	StreamVisibilityUnlisted StreamVisibility = "unlisted"
)

// StreamInput represents the incoming RTMP endpoint information
type StreamInput struct {
	ID       string   `json:"id" bson:"id"`
	Protocol string   `json:"protocol" bson:"protocol"` // rtmp, srt, etc.
	URL      string   `json:"url" bson:"url"`
	Backup   bool     `json:"backup" bson:"backup"`
	Servers  []string `json:"servers" bson:"servers"` // List of ingestor servers that can accept this stream
}

// StreamOutput represents how the stream is distributed to viewers
type StreamOutput struct {
	ID         string   `json:"id" bson:"id"`
	Protocol   string   `json:"protocol" bson:"protocol"` // webrtc, hls, dash, etc.
	URL        string   `json:"url" bson:"url"`
	EnabledFor []string `json:"enabled_for" bson:"enabled_for"` // List of user roles that can access this output
	Format     string   `json:"format" bson:"format"`           // video quality, codec info
	DRMEnabled bool     `json:"drm_enabled" bson:"drm_enabled"`
	CDNEnabled bool     `json:"cdn_enabled" bson:"cdn_enabled"`
}

// StreamEnhancement represents a processing filter applied to the stream
type StreamEnhancement struct {
	ID       string            `json:"id" bson:"id"`
	Type     string            `json:"type" bson:"type"` // ai, filter, overlay, etc.
	Enabled  bool              `json:"enabled" bson:"enabled"`
	Priority int               `json:"priority" bson:"priority"` // Order of application
	Settings map[string]string `json:"settings" bson:"settings"` // Enhancement specific settings
}

// StreamMetrics contains various metrics about the stream
type StreamMetrics struct {
	ViewerCount     int       `json:"viewer_count" bson:"viewer_count"`
	PeakViewerCount int       `json:"peak_viewer_count" bson:"peak_viewer_count"`
	StartTime       time.Time `json:"start_time" bson:"start_time"`
	EndTime         time.Time `json:"end_time" bson:"end_time"`
	Duration        int64     `json:"duration" bson:"duration"` // In seconds
	IngestBitrate   int       `json:"ingest_bitrate" bson:"ingest_bitrate"`
	OutputBitrate   int       `json:"output_bitrate" bson:"output_bitrate"`
	FrameRate       float64   `json:"frame_rate" bson:"frame_rate"`
	Resolution      string    `json:"resolution" bson:"resolution"`
}

// Stream represents a live stream in the platform
type Stream struct {
	ID           string              `json:"id" bson:"_id"`
	UserID       string              `json:"user_id" bson:"user_id"`
	Title        string              `json:"title" bson:"title"`
	Description  string              `json:"description" bson:"description"`
	Status       StreamStatus        `json:"status" bson:"status"`
	Visibility   StreamVisibility    `json:"visibility" bson:"visibility"`
	Tags         []string            `json:"tags" bson:"tags"`
	Inputs       []StreamInput       `json:"inputs" bson:"inputs"`
	Outputs      []StreamOutput      `json:"outputs" bson:"outputs"`
	Enhancements []StreamEnhancement `json:"enhancements" bson:"enhancements"`
	Metrics      StreamMetrics       `json:"metrics" bson:"metrics"`
	CreatedAt    time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at" bson:"updated_at"`
	Recording    bool                `json:"recording" bson:"recording"`
	Thumbnail    string              `json:"thumbnail" bson:"thumbnail"`
	Key          string              `json:"-" bson:"key"` // Stream key, not exposed in API
	Region       string              `json:"region" bson:"region"`
	Metadata     map[string]string   `json:"metadata" bson:"metadata"` // Custom metadata
}

// StreamCreateRequest represents a request to create a new stream
type StreamCreateRequest struct {
	Title       string            `json:"title" validate:"required"`
	Description string            `json:"description"`
	Visibility  StreamVisibility  `json:"visibility" validate:"required"`
	Tags        []string          `json:"tags"`
	Recording   bool              `json:"recording"`
	Region      string            `json:"region"`
	Metadata    map[string]string `json:"metadata"`
}

// StreamUpdateRequest represents a request to update an existing stream
type StreamUpdateRequest struct {
	Title       *string           `json:"title"`
	Description *string           `json:"description"`
	Visibility  *StreamVisibility `json:"visibility"`
	Tags        []string          `json:"tags"`
	Recording   *bool             `json:"recording"`
	Region      *string           `json:"region"`
	Metadata    map[string]string `json:"metadata"`
}

// StreamStartRequest represents a request to start a stream
type StreamStartRequest struct {
	StreamID string            `json:"stream_id" validate:"required"`
	Protocol string            `json:"protocol" validate:"required"` // rtmp, srt, etc.
	Settings map[string]string `json:"settings"`
}

// StreamStartResponse represents a response to a start stream request
type StreamStartResponse struct {
	StreamID  string `json:"stream_id"`
	InputURL  string `json:"input_url"` // RTMP URL to publish to
	StreamKey string `json:"stream_key"`
	BackupURL string `json:"backup_url,omitempty"` // Backup RTMP URL for redundancy
	ServerID  string `json:"server_id"`
	BackupID  string `json:"backup_id,omitempty"`
}

// StreamStopRequest represents a request to stop a stream
type StreamStopRequest struct {
	StreamID string `json:"stream_id" validate:"required"`
	Force    bool   `json:"force"`
}

// StreamListOptions contains options for listing streams
type StreamListOptions struct {
	UserID     string
	Status     StreamStatus
	Visibility StreamVisibility
	Tags       []string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// StreamEvent represents an event related to a stream
type StreamEvent struct {
	ID        string      `json:"id"`
	StreamID  string      `json:"stream_id"`
	Type      string      `json:"type"` // created, updated, started, stopped, error
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}
