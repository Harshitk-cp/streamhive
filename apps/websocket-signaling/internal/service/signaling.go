// apps/websocket-signaling/internal/service/signaling.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

	// Extract SDP with improved parsing
	offerSDP, err := extractSDP(msg.Payload)
	if err != nil {
		log.Printf("Error extracting SDP: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Could not parse offer payload")
		return
	}

	// Log SDP length and validate
	log.Printf("SDP offer length: %d bytes", len(offerSDP))

	// Make sure SDP looks valid
	if !isValidSDP(offerSDP) {
		log.Printf("Invalid SDP format")
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Invalid SDP format")
		return
	}

	// Connect to WebRTC-out service via gRPC with improved error handling
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	webrtcClient, conn, err := s.getWebRTCServiceClient()
	if err != nil {
		log.Printf("Failed to connect to WebRTC-out service: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "service_unavailable", "WebRTC service is currently unavailable")
		return
	}
	defer conn.Close()

	// Forward offer to WebRTC-out service
	log.Printf("Sending offer to WebRTC-out service for stream %s (size: %d bytes)", msg.StreamID, len(offerSDP))
	req := &webrtcpb.OfferRequest{
		StreamId: msg.StreamID,
		ViewerId: msg.SenderID,
		Offer:    offerSDP,
	}

	resp, err := webrtcClient.HandleOffer(ctx, req)
	if err != nil {
		log.Printf("Failed to handle offer: %v", err)
		// Enhanced error handling
		if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "message size") {
			log.Printf("Connection error detected - likely an issue with message size. SDP size: %d bytes", len(offerSDP))
			s.sendErrorResponse(msg.SenderID, msg.StreamID, "connection_error", "WebRTC service error. Please try again.")
		} else if strings.Contains(err.Error(), "timeout") {
			s.sendErrorResponse(msg.SenderID, msg.StreamID, "timeout", "Connection timed out. Please try again.")
		} else {
			s.sendErrorResponse(msg.SenderID, msg.StreamID, "offer_rejected", "Could not process your connection request")
		}
		return
	}

	// Verify SDP answer
	if !isValidSDP(resp.Answer) {
		log.Printf("Invalid SDP answer received from WebRTC service")
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "internal_error", "Invalid answer received from WebRTC service")
		return
	}

	log.Printf("Received valid SDP answer from WebRTC service, length: %d bytes", len(resp.Answer))

	// Send answer back with proper JSON structure
	answerData := map[string]string{
		"sdp": resp.Answer,
	}

	answerBytes, err := json.Marshal(answerData)
	if err != nil {
		log.Printf("Failed to marshal answer data: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "internal_error", "Internal server error")
		return
	}

	// Create answer message
	answer := model.SignalingMessage{
		Type:        "answer",
		StreamID:    msg.StreamID,
		SenderID:    "server",
		RecipientID: msg.SenderID,
		Payload:     json.RawMessage(answerBytes),
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

	// Define connection options once
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // Increased from 5s to 10s
			Timeout:             3 * time.Second,  // Increased from 2s to 3s
			PermitWithoutStream: true,
		}),
		grpc.WithTimeout(10 * time.Second), // Increased timeout for connection
		// Handle larger message sizes
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(20*1024*1024), // 20MB max receive
			grpc.MaxCallSendMsgSize(20*1024*1024), // 20MB max send
		),
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying connection to WebRTC service (attempt %d/%d)", attempt+1, maxRetries)
			// Exponential backoff with jitter for more efficient retry strategy
			backoffTime := time.Duration(100*(1<<attempt)) * time.Millisecond
			jitter := time.Duration(rand.Intn(100)) * time.Millisecond
			time.Sleep(backoffTime + jitter)
		}

		// Connect with optimized options
		conn, err = grpc.Dial(s.cfg.WebRTCOut.Address, dialOpts...)

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

// Helper functions for SDP handling
func extractSDP(payload interface{}) (string, error) {
	var payloadStr string

	// Convert payload to string based on type
	switch p := payload.(type) {
	case []byte:
		payloadStr = string(p)
	case string:
		payloadStr = p
	default:
		// Try to marshal to JSON
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("could not marshal payload: %v", err)
		}
		payloadStr = string(payloadBytes)
	}

	// Try different extraction methods

	// 1. Check if it's already a valid SDP
	if strings.HasPrefix(strings.TrimSpace(payloadStr), "v=0") {
		return payloadStr, nil
	}

	// 2. Try JSON object with sdp field
	var offerObject map[string]interface{}
	if err := json.Unmarshal([]byte(payloadStr), &offerObject); err == nil {
		if sdp, ok := offerObject["sdp"].(string); ok {
			return sdp, nil
		}
	}

	// 3. Try JSON string
	var sdpString string
	if err := json.Unmarshal([]byte(payloadStr), &sdpString); err == nil {
		if strings.HasPrefix(strings.TrimSpace(sdpString), "v=0") {
			return sdpString, nil
		}
	}

	return "", fmt.Errorf("could not extract SDP from payload")
}

func isValidSDP(sdp string) bool {
	sdp = strings.TrimSpace(sdp)
	return strings.HasPrefix(sdp, "v=0") &&
		strings.Contains(sdp, "m=") && // At least one media section
		len(sdp) > 50 // Reasonable minimum length
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
