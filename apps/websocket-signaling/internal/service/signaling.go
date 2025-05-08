// apps/websocket-signaling/internal/service/signaling.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	webrtcpb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
)

// SignalingService handles WebRTC signaling
type SignalingService struct {
	cfg            *config.Config
	streams        map[string]*StreamSession
	clients        map[string]*ClientSession
	streamsMutex   sync.RWMutex
	clientsMutex   sync.RWMutex
	messageChannel chan model.SignalingMessage
	cleanupTicker  *time.Ticker
	stopChan       chan struct{}
}

// StreamSession represents an active stream session
type StreamSession struct {
	StreamID     string
	Clients      map[string]*ClientSession
	CreatedAt    time.Time
	LastActivity time.Time
	Mutex        sync.RWMutex
}

// ClientSession represents a connected client
type ClientSession struct {
	ClientID     string
	StreamID     string
	SendChannel  chan model.SignalingMessage
	Connected    bool
	IsPublisher  bool
	CreatedAt    time.Time
	LastActivity time.Time
}

// NewSignalingService creates a new signaling service
func NewSignalingService(cfg *config.Config) *SignalingService {
	return &SignalingService{
		cfg:            cfg,
		streams:        make(map[string]*StreamSession),
		clients:        make(map[string]*ClientSession),
		messageChannel: make(chan model.SignalingMessage, 1000),
		stopChan:       make(chan struct{}),
	}
}

// Start starts the signaling service
func (s *SignalingService) Start(ctx context.Context) {
	// Start message processing goroutine
	go s.processMessages()

	// Start cleanup goroutine
	s.cleanupTicker = time.NewTicker(5 * time.Minute)
	go s.cleanupInactiveConnections(ctx)

	log.Println("Signaling service started")
}

// Stop stops the signaling service
func (s *SignalingService) Stop() {
	// Signal all goroutines to stop
	close(s.stopChan)

	// Stop cleanup ticker
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	// Close all client channels
	s.clientsMutex.Lock()
	for _, client := range s.clients {
		close(client.SendChannel)
	}
	s.clientsMutex.Unlock()

	log.Println("Signaling service stopped")
}

// RegisterClient registers a new client
func (s *SignalingService) RegisterClient(clientID, streamID string, sendChan chan model.SignalingMessage, isPublisher bool) error {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	// Check if client already exists - if so, unregister first to clean up properly
	if existingClient, exists := s.clients[clientID]; exists {
		log.Printf("Client %s already exists, cleaning up previous session", clientID)
		// Close existing send channel
		close(existingClient.SendChannel)

		// Remove from previous stream if different
		if existingClient.StreamID != streamID {
			s.streamsMutex.Lock()
			if stream, streamExists := s.streams[existingClient.StreamID]; streamExists {
				stream.Mutex.Lock()
				delete(stream.Clients, clientID)
				stream.Mutex.Unlock()

				// Check if stream is now empty
				if len(stream.Clients) == 0 {
					delete(s.streams, existingClient.StreamID)
					log.Printf("Removed empty stream: %s", existingClient.StreamID)
				}
			}
			s.streamsMutex.Unlock()
		}

		// Remove from clients map
		delete(s.clients, clientID)
	}

	// Create client session
	client := &ClientSession{
		ClientID:     clientID,
		StreamID:     streamID,
		SendChannel:  sendChan,
		Connected:    true,
		IsPublisher:  isPublisher,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Add to clients map
	s.clients[clientID] = client

	// Get or create stream session with proper locking
	s.streamsMutex.Lock()
	stream, exists := s.streams[streamID]
	if !exists {
		stream = &StreamSession{
			StreamID:     streamID,
			Clients:      make(map[string]*ClientSession),
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		s.streams[streamID] = stream
	}
	s.streamsMutex.Unlock()

	// Add client to stream with proper locking
	stream.Mutex.Lock()
	stream.Clients[clientID] = client
	stream.LastActivity = time.Now()
	stream.Mutex.Unlock()

	log.Printf("Client %s registered for stream %s (publisher: %v)", clientID, streamID, isPublisher)
	return nil
}

// UnregisterClient unregisters a client
func (s *SignalingService) UnregisterClient(clientID string) {
	s.clientsMutex.Lock()
	client, exists := s.clients[clientID]
	if !exists {
		s.clientsMutex.Unlock()
		return
	}

	// Get streamID before deleting the client
	streamID := client.StreamID

	// Delete from clients map
	delete(s.clients, clientID)
	s.clientsMutex.Unlock()

	// Remove client from stream with proper locking
	s.streamsMutex.Lock()
	stream, streamExists := s.streams[streamID]
	if streamExists {
		stream.Mutex.Lock()
		delete(stream.Clients, clientID)
		clientCount := len(stream.Clients)
		stream.Mutex.Unlock()

		// If no clients left in stream, remove stream
		if clientCount == 0 {
			delete(s.streams, streamID)
			log.Printf("Removed empty stream: %s", streamID)
		}
	}
	s.streamsMutex.Unlock()

	log.Printf("Client %s unregistered from stream %s", clientID, streamID)
}

// HandleMessage handles a signaling message
func (s *SignalingService) HandleMessage(msg model.SignalingMessage) {
	// Update client activity timestamp
	s.clientsMutex.RLock()
	client, exists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if exists {
		client.LastActivity = time.Now()
	}

	// Update stream activity timestamp
	s.streamsMutex.RLock()
	stream, streamExists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if streamExists {
		stream.LastActivity = time.Now()
	}

	// Send message to processing channel
	s.messageChannel <- msg
}

// processMessages processes signaling messages
func (s *SignalingService) processMessages() {
	for {
		select {
		case msg := <-s.messageChannel:
			s.routeMessage(msg)
		case <-s.stopChan:
			return
		}
	}
}

// routeMessage routes a signaling message to the appropriate recipient
func (s *SignalingService) routeMessage(msg model.SignalingMessage) {
	// If recipient is specified, send directly to recipient
	if msg.RecipientID != "" && msg.RecipientID != "server" {
		s.clientsMutex.RLock()
		recipient, exists := s.clients[msg.RecipientID]
		s.clientsMutex.RUnlock()

		if exists && recipient.Connected {
			select {
			case recipient.SendChannel <- msg:
				// Message sent
			default:
				// Channel full or closed, log error
				log.Printf("Failed to send message to client %s: channel full or closed", msg.RecipientID)
			}
		}
		return
	}

	// Handle messages directed to the server
	if msg.RecipientID == "server" {
		switch msg.Type {
		case "offer":
			// Forward offer to WebRTC Out
			s.handleOffer(msg)
		case "answer":
			// Received answer from a viewer, forward to publisher
			s.handleAnswer(msg)
		case "ice_candidate":
			// Forward ICE candidate
			s.handleICECandidate(msg)
		case "ping":
			// Respond with pong
			s.handlePing(msg)
		}
		return
	}

	// If no specific recipient, broadcast to all clients in the stream
	s.streamsMutex.RLock()
	stream, exists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if !exists {
		log.Printf("Stream %s not found", msg.StreamID)
		return
	}

	// Broadcast to all clients in stream except sender
	stream.Mutex.RLock()
	for clientID, client := range stream.Clients {
		if clientID != msg.SenderID && client.Connected {
			select {
			case client.SendChannel <- msg:
				// Message sent
			default:
				// Channel full or closed, log error
				log.Printf("Failed to broadcast message to client %s: channel full or closed", clientID)
			}
		}
	}
	stream.Mutex.RUnlock()
}

// handleOffer handles SDP offer messages
func (s *SignalingService) handleOffer(msg model.SignalingMessage) {
	log.Printf("Received offer from %s for stream %s", msg.SenderID, msg.StreamID)

	// Parse offer payload
	var offerSDP string

	// The Payload could be either a json.RawMessage or another type
	// Convert the payload to a string for processing
	var payloadStr string

	// Check the type of Payload and convert appropriately
	switch p := msg.Payload.(type) {
	case json.RawMessage:
		payloadStr = string(p)
	case []byte:
		payloadStr = string(p)
	case string:
		payloadStr = p
	default:
		// Convert unknown types to JSON
		payloadBytes, err := json.Marshal(msg.Payload)
		if err != nil {
			log.Printf("Error marshaling payload: %v", err)
			s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Could not parse offer payload")
			return
		}
		payloadStr = string(payloadBytes)
	}

	// First try to unmarshal as JSON object containing SDP
	var offerPayload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadStr), &offerPayload); err == nil {
		// Successfully parsed as JSON object
		if sdp, ok := offerPayload["sdp"].(string); ok {
			offerSDP = sdp
			log.Printf("Successfully extracted SDP from JSON payload")
		} else {
			log.Printf("Invalid offer format: missing sdp field in JSON object")
			s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Offer is missing SDP field")
			return
		}
	} else {
		// Try if payload is simply a string containing the SDP
		var sdpString string
		if err := json.Unmarshal([]byte(payloadStr), &sdpString); err == nil {
			offerSDP = sdpString
			log.Printf("Successfully extracted SDP as direct string")
		} else {
			// If it doesn't parse as JSON at all, try using the payload string directly
			if strings.HasPrefix(strings.TrimSpace(payloadStr), "v=0") {
				offerSDP = payloadStr
				log.Printf("Using payload directly as SDP string")
			} else {
				log.Printf("Error parsing offer: %v", err)
				s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Could not parse offer payload")
				return
			}
		}
	}

	// Ensure the SDP doesn't have surrounding quotes and is properly formatted
	offerSDP = strings.Trim(offerSDP, "\"")

	// Make sure the SDP starts with v=0
	if !strings.HasPrefix(strings.TrimSpace(offerSDP), "v=0") {
		log.Printf("SDP doesn't start with v=0, which is invalid")
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "SDP format is invalid")
		return
	}

	// Log the SDP for debugging (truncated to avoid massive logs)
	sdpPreviewLength := 50
	if len(offerSDP) < sdpPreviewLength {
		sdpPreviewLength = len(offerSDP)
	}
	log.Printf("Sending SDP offer to WebRTC service (first %d chars): %s",
		sdpPreviewLength, offerSDP[:sdpPreviewLength])

	// Connect to WebRTC-out service via gRPC
	webrtcClient, conn, err := s.getWebRTCServiceClient()
	if err != nil {
		log.Printf("Failed to connect to WebRTC-out service: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "service_unavailable", "WebRTC service is currently unavailable")
		return
	}
	defer conn.Close()

	// Forward offer to WebRTC-out service
	req := &webrtcpb.OfferRequest{
		StreamId: msg.StreamID,
		ViewerId: msg.SenderID,
		Offer:    offerSDP, // Plain SDP string without any JSON formatting
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := webrtcClient.HandleOffer(ctx, req)
	if err != nil {
		log.Printf("Failed to handle offer: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "offer_rejected", "Could not process your connection request")
		return
	}

	// Instead of Base64 encoding, create proper JSON
	answerData := map[string]string{
		"sdp": resp.Answer,
	}

	// Marshal to JSON
	answerJSON, err := json.Marshal(answerData)
	if err != nil {
		log.Printf("Failed to marshal answer to JSON: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "internal_error", "Internal server error")
		return
	}

	// Create answer message with proper JSON payload
	answer := model.SignalingMessage{
		Type:        "answer",
		StreamID:    msg.StreamID,
		SenderID:    "server",
		RecipientID: msg.SenderID,
		Payload:     answerJSON, // Use the JSON bytes directly
	}

	s.clientsMutex.RLock()
	clientSession, exists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if exists && clientSession.Connected {
		select {
		case clientSession.SendChannel <- answer:
			log.Printf("Sent answer to %s", msg.SenderID)
		default:
			log.Printf("Failed to send answer to %s: channel full or closed", msg.SenderID)
		}
	}
}

// sendErrorResponse sends an error response to a client
func (s *SignalingService) sendErrorResponse(clientID, streamID, errorCode, errorMessage string) {
	// Create a properly structured error object
	errorData := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    errorCode,
		Message: errorMessage,
	}

	errorPayload, err := json.Marshal(errorData)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return
	}

	errorMsg := model.SignalingMessage{
		Type:        "error",
		StreamID:    streamID,
		SenderID:    "server",
		RecipientID: clientID,
		Payload:     errorPayload,
	}

	s.clientsMutex.RLock()
	clientSession, exists := s.clients[clientID]
	s.clientsMutex.RUnlock()

	if exists && clientSession.Connected {
		select {
		case clientSession.SendChannel <- errorMsg:
			log.Printf("Sent error response to %s: %s", clientID, errorMessage)
		default:
			log.Printf("Failed to send error response to %s: channel full or closed", clientID)
		}
	}
}

// handleAnswer handles SDP answer messages
func (s *SignalingService) handleAnswer(msg model.SignalingMessage) {
	// Find the publisher for this stream
	s.streamsMutex.RLock()
	stream, exists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if !exists {
		log.Printf("Stream %s not found for answer", msg.StreamID)
		return
	}

	// Find the publisher client
	var publisherID string
	stream.Mutex.RLock()
	for clientID, client := range stream.Clients {
		if client.IsPublisher {
			publisherID = clientID
			break
		}
	}
	stream.Mutex.RUnlock()

	if publisherID == "" {
		log.Printf("No publisher found for stream %s", msg.StreamID)
		return
	}

	// Forward answer to publisher
	forwardMsg := model.SignalingMessage{
		Type:        "answer",
		StreamID:    msg.StreamID,
		SenderID:    msg.SenderID,
		RecipientID: publisherID,
		Payload:     msg.Payload,
	}

	s.clientsMutex.RLock()
	publisher, exists := s.clients[publisherID]
	s.clientsMutex.RUnlock()

	if exists && publisher.Connected {
		select {
		case publisher.SendChannel <- forwardMsg:
			log.Printf("Forwarded answer from %s to publisher %s", msg.SenderID, publisherID)
		default:
			log.Printf("Failed to forward answer to publisher %s: channel full or closed", publisherID)
		}
	}
}

// handleICECandidate handles ICE candidate messages
func (s *SignalingService) handleICECandidate(msg model.SignalingMessage) {
	// If recipient is specified, forward directly
	if msg.RecipientID != "" && msg.RecipientID != "server" {
		s.clientsMutex.RLock()
		recipient, exists := s.clients[msg.RecipientID]
		s.clientsMutex.RUnlock()

		if exists && recipient.Connected {
			select {
			case recipient.SendChannel <- msg:
				log.Printf("Forwarded ICE candidate from %s to %s", msg.SenderID, msg.RecipientID)
			default:
				log.Printf("Failed to forward ICE candidate to %s: channel full or closed", msg.RecipientID)
			}
		}
		return
	}

	// If no specific recipient, try to find the publisher/viewer counterpart
	s.streamsMutex.RLock()
	stream, exists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if !exists {
		log.Printf("Stream %s not found for ICE candidate", msg.StreamID)
		return
	}

	// Determine if sender is publisher or viewer
	s.clientsMutex.RLock()
	sender, senderExists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if !senderExists {
		log.Printf("Sender %s not found for ICE candidate", msg.SenderID)
		return
	}

	isPublisher := sender.IsPublisher

	// Forward to opposite role (publisher->viewers or viewer->publisher)
	stream.Mutex.RLock()
	for clientID, client := range stream.Clients {
		if clientID != msg.SenderID && client.IsPublisher != isPublisher {
			// Create message for recipient
			forwardMsg := model.SignalingMessage{
				Type:        "ice_candidate",
				StreamID:    msg.StreamID,
				SenderID:    msg.SenderID,
				RecipientID: clientID,
				Payload:     msg.Payload,
			}

			select {
			case client.SendChannel <- forwardMsg:
				log.Printf("Forwarded ICE candidate from %s to %s", msg.SenderID, clientID)
			default:
				log.Printf("Failed to forward ICE candidate to %s: channel full or closed", clientID)
			}
		}
	}
	stream.Mutex.RUnlock()
}

// handlePing handles ping messages
func (s *SignalingService) handlePing(msg model.SignalingMessage) {
	// Send pong response
	pong := model.SignalingMessage{
		Type:        "pong",
		StreamID:    msg.StreamID,
		SenderID:    "server",
		RecipientID: msg.SenderID,
		Payload:     msg.Payload, // Echo back the timestamp
	}

	s.clientsMutex.RLock()
	client, exists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if exists && client.Connected {
		select {
		case client.SendChannel <- pong:
			// Pong sent
		default:
			log.Printf("Failed to send pong to %s: channel full or closed", msg.SenderID)
		}
	}
}

// getWebRTCServiceClient gets a client for the WebRTC service with retry logic
func (s *SignalingService) getWebRTCServiceClient() (webrtcpb.WebRTCServiceClient, *grpc.ClientConn, error) {
	// Try to connect with exponential backoff
	var conn *grpc.ClientConn
	var err error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying connection to WebRTC service (attempt %d/%d)", attempt+1, maxRetries)
			// Exponential backoff: 100ms, 200ms, 400ms
			time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
		}

		// Connect with modified options for better reliability
		conn, err = grpc.Dial(
			s.cfg.WebRTCOut.Address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                5 * time.Second,
				Timeout:             2 * time.Second,
				PermitWithoutStream: true,
			}),
			grpc.WithTimeout(5*time.Second),
		)

		if err == nil {
			// Successfully connected
			break
		}

		log.Printf("Attempt %d: Failed to connect to WebRTC service: %v", attempt+1, err)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to WebRTC-out service after %d attempts: %w", maxRetries, err)
	}

	// Create client
	client := webrtcpb.NewWebRTCServiceClient(conn)
	return client, conn, nil
}

// GetClientCount returns the number of clients connected to a stream
func (s *SignalingService) GetClientCount(streamID string) int {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()

	stream, exists := s.streams[streamID]
	if !exists {
		return 0
	}

	stream.Mutex.RLock()
	count := len(stream.Clients)
	stream.Mutex.RUnlock()

	return count
}

// GetTotalClientCount returns the total number of connected clients
func (s *SignalingService) GetTotalClientCount() int {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	return len(s.clients)
}

// cleanupInactiveConnections removes inactive connections
func (s *SignalingService) cleanupInactiveConnections(ctx context.Context) {
	for {
		select {
		case <-s.cleanupTicker.C:
			// Set timeout threshold (10 minutes)
			timeout := time.Now().Add(-10 * time.Minute)

			// Find inactive clients
			var inactiveClients []string

			s.clientsMutex.RLock()
			for clientID, client := range s.clients {
				if client.LastActivity.Before(timeout) {
					inactiveClients = append(inactiveClients, clientID)
				}
			}
			s.clientsMutex.RUnlock()

			// Remove inactive clients
			for _, clientID := range inactiveClients {
				log.Printf("Removing inactive client: %s", clientID)
				s.UnregisterClient(clientID)
			}

			// Find empty streams
			var emptyStreams []string

			s.streamsMutex.RLock()
			for streamID, stream := range s.streams {
				stream.Mutex.RLock()
				clientCount := len(stream.Clients)
				stream.Mutex.RUnlock()

				if clientCount == 0 {
					emptyStreams = append(emptyStreams, streamID)
				}
			}
			s.streamsMutex.RUnlock()

			// Remove empty streams
			if len(emptyStreams) > 0 {
				s.streamsMutex.Lock()
				for _, streamID := range emptyStreams {
					log.Printf("Removing empty stream: %s", streamID)
					delete(s.streams, streamID)
				}
				s.streamsMutex.Unlock()
			}

		case <-ctx.Done():
			return

		case <-s.stopChan:
			return
		}
	}
}
