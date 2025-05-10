package model

import (
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

// StreamStatus represents the status of a WebRTC stream
type StreamStatus string

const (
	// StreamStatusIdle means the stream is registered but not active
	StreamStatusIdle StreamStatus = "idle"

	// StreamStatusConnecting means the stream is in process of connecting
	StreamStatusConnecting StreamStatus = "connecting"

	// StreamStatusActive means the stream is currently active
	StreamStatusActive StreamStatus = "active"

	// StreamStatusError means the stream encountered an error
	StreamStatusError StreamStatus = "error"

	// StreamStatusClosed means the stream was closed
	StreamStatusClosed StreamStatus = "closed"
)

// FrameType represents the type of media frame
type FrameType string

const (
	// FrameTypeVideo represents a video frame
	FrameTypeVideo FrameType = "video"

	// FrameTypeAudio represents an audio frame
	FrameTypeAudio FrameType = "audio"

	// FrameTypeMetadata represents a metadata frame
	FrameTypeMetadata FrameType = "metadata"
)

// VideoCodec represents a video codec
type VideoCodec string

const (
	// VideoCodecH264 represents the H.264 codec
	VideoCodecH264 VideoCodec = "h264"

	// VideoCodecVP8 represents the VP8 codec
	VideoCodecVP8 VideoCodec = "vp8"

	// VideoCodecVP9 represents the VP9 codec
	VideoCodecVP9 VideoCodec = "vp9"
)

// AudioCodec represents an audio codec
type AudioCodec string

const (
	// AudioCodecOpus represents the Opus codec
	AudioCodecOpus AudioCodec = "opus"

	// AudioCodecAAC represents the AAC codec
	AudioCodecAAC AudioCodec = "aac"
)

// Frame represents a media frame
type Frame struct {
	StreamID   string
	FrameID    string
	Type       FrameType
	Data       []byte
	Timestamp  time.Time
	IsKeyFrame bool
	Sequence   int64
	Metadata   map[string]string
}

// Viewer represents a viewer connection
type Viewer struct {
	ID             string
	PeerConnection *webrtc.PeerConnection
	DataChannel    *webrtc.DataChannel
	Status         StreamStatus
	StartTime      time.Time
	LastActivity   time.Time
	StreamID       string
	TotalBytes     int64
	Mutex          sync.RWMutex
	Stats          ViewerStats
	// track references
	AudioTrack *webrtc.TrackLocalStaticSample
	VideoTrack *webrtc.TrackLocalStaticSample
}

// ViewerStats represents statistics for a viewer connection
type ViewerStats struct {
	VideoPacketsSent   int64
	AudioPacketsSent   int64
	VideoBytesSent     int64
	AudioBytesSent     int64
	NackCount          int64
	PLICount           int64
	FIRCount           int64
	FramesDropped      int64
	RTT                int64 // Round-trip time in ms
	EstimatedBitrate   int64
	QualityLimitReason string
}

// Stream represents a WebRTC stream
type Stream struct {
	ID                   string
	Status               StreamStatus
	CreatedAt            time.Time
	StartedAt            time.Time
	LastActivity         time.Time
	Viewers              map[string]*Viewer
	VideoCodec           VideoCodec
	AudioCodec           AudioCodec
	Width                int
	Height               int
	FrameRate            float64
	KeyFrameInterval     int
	TotalFrames          int64
	TotalViewers         int64
	MaxConcurrentViewers int
	CurrentViewers       int
	Mutex                sync.Mutex
}

// StreamOptions represents options for a WebRTC stream
type StreamOptions struct {
	MaxBitrate         int
	MaxFrameRate       int
	JitterBuffer       int
	OpusMinBitrate     int
	OpusMaxBitrate     int
	OpusComplexity     int
	VideoScalingFactor float64
}
