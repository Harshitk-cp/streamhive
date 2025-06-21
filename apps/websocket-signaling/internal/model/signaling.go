// apps/websocket-signaling/internal/model/signaling.go

package model

import (
	"time"
)

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	Type        string      `json:"type"`
	StreamID    string      `json:"stream_id"`
	SenderID    string      `json:"sender_id"`
	RecipientID string      `json:"recipient_id,omitempty"`
	Payload     []byte      `json:"payload"` // Changed to []byte for consistency
	Timestamp   int64       `json:"timestamp,omitempty"`
}

// ClientInfo represents information about a connected client
type ClientInfo struct {
	ClientID    string    `json:"client_id"`
	StreamID    string    `json:"stream_id"`
	IsPublisher bool      `json:"is_publisher"`
	Connected   bool      `json:"connected"`
	ConnectedAt time.Time `json:"connected_at"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

// StreamInfo represents information about an active stream
type StreamInfo struct {
	StreamID      string       `json:"stream_id"`
	ViewerCount   int          `json:"viewer_count"`
	PublisherInfo *ClientInfo  `json:"publisher_info,omitempty"`
	Viewers       []ClientInfo `json:"viewers,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// MessageType constants for WebRTC signaling
const (
	MessageTypeOffer        = "offer"
	MessageTypeAnswer       = "answer"
	MessageTypeICECandidate = "ice_candidate"
	MessageTypePing         = "ping"
	MessageTypePong         = "pong"
	MessageTypeStreamEnded  = "stream_ended"
	MessageTypeError        = "error"
)
