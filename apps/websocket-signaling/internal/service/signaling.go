// apps/websocket-signaling/internal/service/signaling.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/metrics"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
)

// StreamInfo contains information about a registered stream
type StreamInfo struct {
	StreamID  string
	SessionID string
	NodeID    string
	Clients   map[string]*Client
	CreatedAt time.Time
	Mu        sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	ID           string
	UserID       string
	StreamID     string
	Send         chan []byte
	LastActivity time.Time
}

// SignalingMessage represents a WebSocket signaling message
type SignalingMessage struct {
	Type      string          `json:"type"`
	StreamID  string          `json:"streamId"`
	UserID    string          `json:"userId,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// SDPMessage represents an SDP message
type SDPMessage struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

// ICECandidateMessage represents an ICE candidate message
type ICECandidateMessage struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdpMid"`
	SDPMLineIndex uint32 `json:"sdpMLineIndex"`
}

// Service implements the signaling service
type Service struct {
	config   *config.Config
	metrics  metrics.Collector
	streams  map[string]*StreamInfo
	clients  map[string]*Client
	streamMu sync.RWMutex
	clientMu sync.RWMutex
}

// New creates a new signaling service
func New(cfg *config.Config, m metrics.Collector) *Service {
	svc := &Service{
		config:  cfg,
		metrics: m,
		streams: make(map[string]*StreamInfo),
		clients: make(map[string]*Client),
	}

	// Start client session cleanup goroutine
	go svc.cleanupSessions()

	return svc
}

// RegisterStream registers a stream with the signaling service
func (s *Service) RegisterStream(ctx context.Context, req *webrtcPb.RegisterStreamRequest) (*webrtcPb.RegisterStreamResponse, error) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()

	// Check if stream already exists
	if _, exists := s.streams[req.StreamId]; exists {
		return &webrtcPb.RegisterStreamResponse{Success: false}, fmt.Errorf("stream already registered: %s", req.StreamId)
	}

	// Check if maximum number of streams is reached
	if len(s.streams) >= s.config.Stream.MaxStreams {
		return &webrtcPb.RegisterStreamResponse{Success: false}, fmt.Errorf("maximum number of streams reached")
	}

	// Create stream
	stream := &StreamInfo{
		StreamID:  req.StreamId,
		SessionID: req.SessionId,
		NodeID:    req.NodeId,
		Clients:   make(map[string]*Client),
		CreatedAt: time.Now(),
	}

	// Store stream
	s.streams[req.StreamId] = stream

	// Record metric
	s.metrics.StreamRegistered(req.StreamId)

	log.Printf("Stream registered: %s (node: %s)", req.StreamId, req.NodeId)

	return &webrtcPb.RegisterStreamResponse{Success: true}, nil
}

// UnregisterStream unregisters a stream from the signaling service
func (s *Service) UnregisterStream(ctx context.Context, req *webrtcPb.UnregisterStreamRequest) (*webrtcPb.UnregisterStreamResponse, error) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()

	// Check if stream exists
	stream, exists := s.streams[req.StreamId]
	if !exists {
		return &webrtcPb.UnregisterStreamResponse{Success: false}, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Check if session ID matches
	if stream.SessionID != req.SessionId {
		return &webrtcPb.UnregisterStreamResponse{Success: false}, fmt.Errorf("session ID mismatch")
	}

	// Disconnect all clients
	stream.Mu.Lock()
	for _, client := range stream.Clients {
		s.DisconnectClient(client)
	}
	stream.Mu.Unlock()

	// Remove stream
	delete(s.streams, req.StreamId)

	// Record metric
	s.metrics.StreamUnregistered(req.StreamId)

	log.Printf("Stream unregistered: %s (node: %s)", req.StreamId, req.NodeId)

	return &webrtcPb.UnregisterStreamResponse{Success: true}, nil
}

// GetStreamInfo returns information about a stream
func (s *Service) GetStreamInfo(streamID string) (*StreamInfo, error) {
	s.streamMu.RLock()
	defer s.streamMu.RUnlock()

	// Check if stream exists
	stream, exists := s.streams[streamID]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamID)
	}

	return stream, nil
}

// RegisterClient registers a new client
func (s *Service) RegisterClient(streamID, userID string) (*Client, error) {
	// Check if stream exists
	s.streamMu.RLock()
	stream, exists := s.streams[streamID]
	s.streamMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamID)
	}

	// Check if maximum number of clients is reached
	stream.Mu.RLock()
	clientCount := len(stream.Clients)
	stream.Mu.RUnlock()

	if clientCount >= s.config.Stream.MaxClientsPerStream {
		return nil, fmt.Errorf("maximum number of clients reached for stream: %s", streamID)
	}

	// Create client
	clientID := fmt.Sprintf("%s-%s", streamID, userID)
	client := &Client{
		ID:           clientID,
		UserID:       userID,
		StreamID:     streamID,
		Send:         make(chan []byte, s.config.WebSocket.MessageBufferSize),
		LastActivity: time.Now(),
	}

	// Store client
	s.clientMu.Lock()
	s.clients[clientID] = client
	s.clientMu.Unlock()

	// Add client to stream
	stream.Mu.Lock()
	stream.Clients[userID] = client
	stream.Mu.Unlock()

	// Record metric
	s.metrics.ClientConnected(streamID, userID)

	log.Printf("Client registered: %s (stream: %s)", userID, streamID)

	return client, nil
}

// UnregisterClient unregisters a client
func (s *Service) UnregisterClient(streamID, userID string) {
	clientID := fmt.Sprintf("%s-%s", streamID, userID)

	// Get client
	s.clientMu.RLock()
	client, exists := s.clients[clientID]
	s.clientMu.RUnlock()

	if !exists {
		return
	}

	s.DisconnectClient(client)
}

// DisconnectClient disconnects a client and cleans up resources
func (s *Service) DisconnectClient(client *Client) {
	// Remove client from stream
	s.streamMu.RLock()
	stream, exists := s.streams[client.StreamID]
	s.streamMu.RUnlock()

	if exists {
		stream.Mu.Lock()
		delete(stream.Clients, client.UserID)
		stream.Mu.Unlock()
	}

	// Remove client from clients map
	s.clientMu.Lock()
	delete(s.clients, client.ID)
	s.clientMu.Unlock()

	// Close send channel
	close(client.Send)

	// Record metric
	s.metrics.ClientDisconnected(client.StreamID, client.UserID)

	log.Printf("Client unregistered: %s (stream: %s)", client.UserID, client.StreamID)
}

// HandleMessage handles a WebSocket message
func (s *Service) HandleMessage(clientID string, message []byte) error {
	// Get client
	s.clientMu.RLock()
	client, exists := s.clients[clientID]
	s.clientMu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	// Update client activity
	client.LastActivity = time.Now()

	// Parse message
	var signalingMsg SignalingMessage
	if err := json.Unmarshal(message, &signalingMsg); err != nil {
		s.metrics.SignalingMessageError(client.StreamID, "unknown", "parse_error")
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Validate stream ID matches
	if signalingMsg.StreamID != client.StreamID {
		s.metrics.SignalingMessageError(client.StreamID, signalingMsg.Type, "stream_mismatch")
		return fmt.Errorf("stream ID mismatch")
	}

	// Record metric
	s.metrics.SignalingMessageReceived(client.StreamID, signalingMsg.Type, len(message))

	// Handle message based on type
	switch signalingMsg.Type {
	case "offer":
		return s.handleOfferMessage(client, &signalingMsg)
	case "answer":
		return s.handleAnswerMessage(client, &signalingMsg)
	case "ice-candidate":
		return s.handleICECandidateMessage(client, &signalingMsg)
	case "ping":
		return s.handlePingMessage(client, &signalingMsg)
	default:
		s.metrics.SignalingMessageError(client.StreamID, signalingMsg.Type, "unknown_type")
		return fmt.Errorf("unknown message type: %s", signalingMsg.Type)
	}
}

// handleOfferMessage handles an SDP offer message
func (s *Service) handleOfferMessage(client *Client, msg *SignalingMessage) error {
	// Parse SDP data
	var sdpMsg SDPMessage
	if err := json.Unmarshal(msg.Data, &sdpMsg); err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "parse_error")
		return fmt.Errorf("failed to parse SDP data: %w", err)
	}

	// Get stream info
	s.streamMu.RLock()
	stream, exists := s.streams[client.StreamID]
	s.streamMu.RUnlock()

	if !exists {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "stream_not_found")
		return fmt.Errorf("stream not found: %s", client.StreamID)
	}

	// Create response
	response := SignalingMessage{
		Type:     "offer",
		StreamID: client.StreamID,
		UserID:   client.UserID,
		Data:     msg.Data,
	}

	// Serialize response
	respData, err := json.Marshal(response)
	if err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "marshal_error")
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Broadcast to other clients in the same stream
	stream.Mu.RLock()
	for _, otherClient := range stream.Clients {
		// Skip the sender
		if otherClient.UserID == client.UserID {
			continue
		}

		select {
		case otherClient.Send <- respData:
			s.metrics.SignalingMessageSent(client.StreamID, msg.Type, len(respData))
		default:
			s.metrics.SignalingMessageError(client.StreamID, msg.Type, "send_buffer_full")
			log.Printf("Failed to send message to client: %s (buffer full)", otherClient.UserID)
		}
	}
	stream.Mu.RUnlock()

	return nil
}

// handleAnswerMessage handles an SDP answer message
func (s *Service) handleAnswerMessage(client *Client, msg *SignalingMessage) error {
	// Parse SDP data
	var sdpMsg SDPMessage
	if err := json.Unmarshal(msg.Data, &sdpMsg); err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "parse_error")
		return fmt.Errorf("failed to parse SDP data: %w", err)
	}

	// Get stream info
	s.streamMu.RLock()
	stream, exists := s.streams[client.StreamID]
	s.streamMu.RUnlock()

	if !exists {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "stream_not_found")
		return fmt.Errorf("stream not found: %s", client.StreamID)
	}

	// Create response
	response := SignalingMessage{
		Type:     "answer",
		StreamID: client.StreamID,
		UserID:   client.UserID,
		Data:     msg.Data,
	}

	// Serialize response
	respData, err := json.Marshal(response)
	if err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "marshal_error")
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Forward to the target client
	if msg.UserID != "" {
		stream.Mu.RLock()
		targetClient, exists := stream.Clients[msg.UserID]
		stream.Mu.RUnlock()

		if exists {
			select {
			case targetClient.Send <- respData:
				s.metrics.SignalingMessageSent(client.StreamID, msg.Type, len(respData))
			default:
				s.metrics.SignalingMessageError(client.StreamID, msg.Type, "send_buffer_full")
				log.Printf("Failed to send message to client: %s (buffer full)", targetClient.UserID)
			}
		} else {
			s.metrics.SignalingMessageError(client.StreamID, msg.Type, "target_client_not_found")
			return fmt.Errorf("target client not found: %s", msg.UserID)
		}
	} else {
		// Broadcast to all other clients
		stream.Mu.RLock()
		for _, otherClient := range stream.Clients {
			// Skip the sender
			if otherClient.UserID == client.UserID {
				continue
			}

			select {
			case otherClient.Send <- respData:
				s.metrics.SignalingMessageSent(client.StreamID, msg.Type, len(respData))
			default:
				s.metrics.SignalingMessageError(client.StreamID, msg.Type, "send_buffer_full")
				log.Printf("Failed to send message to client: %s (buffer full)", otherClient.UserID)
			}
		}
		stream.Mu.RUnlock()
	}

	return nil
}

// handleICECandidateMessage handles an ICE candidate message
func (s *Service) handleICECandidateMessage(client *Client, msg *SignalingMessage) error {
	// Parse ICE candidate data
	var iceMsg ICECandidateMessage
	if err := json.Unmarshal(msg.Data, &iceMsg); err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "parse_error")
		return fmt.Errorf("failed to parse ICE candidate data: %w", err)
	}

	// Get stream info
	s.streamMu.RLock()
	stream, exists := s.streams[client.StreamID]
	s.streamMu.RUnlock()

	if !exists {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "stream_not_found")
		return fmt.Errorf("stream not found: %s", client.StreamID)
	}

	// Create response
	response := SignalingMessage{
		Type:     "ice-candidate",
		StreamID: client.StreamID,
		UserID:   client.UserID,
		Data:     msg.Data,
	}

	// Serialize response
	respData, err := json.Marshal(response)
	if err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "marshal_error")
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Forward to the target client
	if msg.UserID != "" {
		stream.Mu.RLock()
		targetClient, exists := stream.Clients[msg.UserID]
		stream.Mu.RUnlock()

		if exists {
			select {
			case targetClient.Send <- respData:
				s.metrics.SignalingMessageSent(client.StreamID, msg.Type, len(respData))
			default:
				s.metrics.SignalingMessageError(client.StreamID, msg.Type, "send_buffer_full")
				log.Printf("Failed to send message to client: %s (buffer full)", targetClient.UserID)
			}
		} else {
			s.metrics.SignalingMessageError(client.StreamID, msg.Type, "target_client_not_found")
			return fmt.Errorf("target client not found: %s", msg.UserID)
		}
	} else {
		// Broadcast to all other clients
		stream.Mu.RLock()
		for _, otherClient := range stream.Clients {
			// Skip the sender
			if otherClient.UserID == client.UserID {
				continue
			}

			select {
			case otherClient.Send <- respData:
				s.metrics.SignalingMessageSent(client.StreamID, msg.Type, len(respData))
			default:
				s.metrics.SignalingMessageError(client.StreamID, msg.Type, "send_buffer_full")
				log.Printf("Failed to send message to client: %s (buffer full)", otherClient.UserID)
			}
		}
		stream.Mu.RUnlock()
	}

	return nil
}

// handlePingMessage handles a ping message
func (s *Service) handlePingMessage(client *Client, msg *SignalingMessage) error {
	// Create pong response
	response := SignalingMessage{
		Type:     "pong",
		StreamID: client.StreamID,
		UserID:   client.UserID,
	}

	// Serialize response
	respData, err := json.Marshal(response)
	if err != nil {
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "marshal_error")
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Send pong response
	select {
	case client.Send <- respData:
		s.metrics.SignalingMessageSent(client.StreamID, "pong", len(respData))
	default:
		s.metrics.SignalingMessageError(client.StreamID, msg.Type, "send_buffer_full")
		return fmt.Errorf("failed to send pong message: buffer full")
	}

	return nil
}

// cleanupSessions periodically cleans up inactive sessions
func (s *Service) cleanupSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupInactiveClients()
			s.cleanupExpiredStreams()
		}
	}
}

// cleanupInactiveClients cleans up inactive clients
func (s *Service) cleanupInactiveClients() {
	now := time.Now()
	threshold := now.Add(-s.config.WebSocket.ClientSessionTimeout)

	// Get inactive clients
	var inactiveClients []*Client
	s.clientMu.RLock()
	for _, client := range s.clients {
		if client.LastActivity.Before(threshold) {
			inactiveClients = append(inactiveClients, client)
		}
	}
	s.clientMu.RUnlock()

	// Disconnect inactive clients
	for _, client := range inactiveClients {
		log.Printf("Disconnecting inactive client: %s (stream: %s)", client.UserID, client.StreamID)
		s.DisconnectClient(client)
	}
}

// cleanupExpiredStreams cleans up expired streams
func (s *Service) cleanupExpiredStreams() {
	now := time.Now()
	threshold := now.Add(-s.config.Stream.StreamTimeout)

	// Get expired streams
	var expiredStreams []string
	s.streamMu.RLock()
	for streamID, stream := range s.streams {
		if stream.CreatedAt.Before(threshold) {
			expiredStreams = append(expiredStreams, streamID)
		}
	}
	s.streamMu.RUnlock()

	// Remove expired streams
	for _, streamID := range expiredStreams {
		s.streamMu.Lock()
		if stream, exists := s.streams[streamID]; exists {
			// Disconnect all clients
			stream.Mu.Lock()
			for _, client := range stream.Clients {
				s.DisconnectClient(client)
			}
			stream.Mu.Unlock()

			// Remove stream
			delete(s.streams, streamID)
			s.metrics.StreamUnregistered(streamID)
			log.Printf("Stream expired: %s", streamID)
		}
		s.streamMu.Unlock()
	}
}
