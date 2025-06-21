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

// SignalingService handles WebRTC signaling logic
type SignalingService struct {
	cfg            *config.Config
	streams        map[string]*StreamSession
	streamsMutex   sync.RWMutex
	clients        map[string]*ClientSession
	clientsMutex   sync.RWMutex
	messageChannel chan model.SignalingMessage
	stopChan       chan struct{}
	cleanupTicker  *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
}

// StreamSession represents an active stream session
type StreamSession struct {
	StreamID     string
	Clients      map[string]*ClientSession
	CreatedAt    time.Time
	LastActivity time.Time
	Mutex        sync.RWMutex
	// Legacy fields for compatibility
	IsActive    bool
	ViewerCount int
	Title       string
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

// Stream represents an active stream (for API compatibility)
type Stream struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	IsActive    bool      `json:"is_active"`
	ViewerCount int       `json:"viewer_count"`
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
	s.ctx, s.cancel = context.WithCancel(ctx)
	log.Println("Signaling service started")
	
	// Start background tasks
	go s.processMessages()
	go s.cleanupInactiveConnections(ctx)
}

// Stop stops the signaling service
func (s *SignalingService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	
	close(s.stopChan)

	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	s.clientsMutex.Lock()
	for _, client := range s.clients {
		close(client.SendChannel)
	}
	s.clientsMutex.Unlock()

	log.Println("Signaling service stopped")
}

// GetActiveStreams returns a list of active streams
func (s *SignalingService) GetActiveStreams() []*Stream {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()
	
	var activeStreams []*Stream
	for _, streamSession := range s.streams {
		if streamSession.IsActive {
			stream := &Stream{
				ID:          streamSession.StreamID,
				Title:       streamSession.Title,
				CreatedAt:   streamSession.CreatedAt,
				IsActive:    streamSession.IsActive,
				ViewerCount: streamSession.ViewerCount,
			}
			activeStreams = append(activeStreams, stream)
		}
	}
	
	return activeStreams
}

// CreateStream creates a new stream
func (s *SignalingService) CreateStream(streamID, title string) *Stream {
	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()
	
	streamSession := &StreamSession{
		StreamID:     streamID,
		Title:        title,
		CreatedAt:    time.Now(),
		IsActive:     true,
		ViewerCount:  0,
		Clients:      make(map[string]*ClientSession),
		LastActivity: time.Now(),
	}
	
	s.streams[streamID] = streamSession
	log.Printf("Created stream: %s", streamID)
	
	return &Stream{
		ID:          streamID,
		Title:       title,
		CreatedAt:   streamSession.CreatedAt,
		IsActive:    true,
		ViewerCount: 0,
	}
}

// GetStream returns a stream by ID
func (s *SignalingService) GetStream(streamID string) (*Stream, bool) {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()
	
	streamSession, exists := s.streams[streamID]
	if !exists {
		return nil, false
	}
	
	stream := &Stream{
		ID:          streamSession.StreamID,
		Title:       streamSession.Title,
		CreatedAt:   streamSession.CreatedAt,
		IsActive:    streamSession.IsActive,
		ViewerCount: streamSession.ViewerCount,
	}
	
	return stream, true
}

// UpdateViewerCount updates the viewer count for a stream
func (s *SignalingService) UpdateViewerCount(streamID string, delta int) {
	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()
	
	if stream, exists := s.streams[streamID]; exists {
		stream.ViewerCount += delta
		if stream.ViewerCount < 0 {
			stream.ViewerCount = 0
		}
		log.Printf("Stream %s viewer count: %d", streamID, stream.ViewerCount)
	}
}

// DeactivateStream marks a stream as inactive
func (s *SignalingService) DeactivateStream(streamID string) {
	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()
	
	if stream, exists := s.streams[streamID]; exists {
		stream.IsActive = false
		log.Printf("Deactivated stream: %s", streamID)
	}
}

// RegisterClient registers a new client
func (s *SignalingService) RegisterClient(clientID, streamID string, sendChan chan model.SignalingMessage, isPublisher bool) error {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	// Check if client already exists
	if existingClient, exists := s.clients[clientID]; exists {
		log.Printf("Client %s already exists, cleaning up previous session", clientID)
		close(existingClient.SendChannel)

		if existingClient.StreamID != streamID {
			s.streamsMutex.Lock()
			if stream, streamExists := s.streams[existingClient.StreamID]; streamExists {
				stream.Mutex.Lock()
				delete(stream.Clients, clientID)
				stream.Mutex.Unlock()

				if len(stream.Clients) == 0 {
					delete(s.streams, existingClient.StreamID)
					log.Printf("Removed empty stream: %s", existingClient.StreamID)
				}
			}
			s.streamsMutex.Unlock()
		}

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

	s.clients[clientID] = client

	// Get or create stream session
	s.streamsMutex.Lock()
	stream, exists := s.streams[streamID]
	if !exists {
		stream = &StreamSession{
			StreamID:     streamID,
			Clients:      make(map[string]*ClientSession),
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			IsActive:     true,
		}
		s.streams[streamID] = stream
	}
	s.streamsMutex.Unlock()

	// Add client to stream
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

	streamID := client.StreamID
	delete(s.clients, clientID)
	s.clientsMutex.Unlock()

	s.streamsMutex.Lock()
	stream, streamExists := s.streams[streamID]
	if streamExists {
		stream.Mutex.Lock()
		delete(stream.Clients, clientID)
		clientCount := len(stream.Clients)
		stream.Mutex.Unlock()

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
	s.clientsMutex.RLock()
	client, exists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if exists {
		client.LastActivity = time.Now()
	}

	s.streamsMutex.RLock()
	stream, streamExists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if streamExists {
		stream.LastActivity = time.Now()
	}

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
	if msg.RecipientID != "" && msg.RecipientID != "server" {
		s.clientsMutex.RLock()
		recipient, exists := s.clients[msg.RecipientID]
		s.clientsMutex.RUnlock()

		if exists && recipient.Connected {
			select {
			case recipient.SendChannel <- msg:
				// Message sent
			default:
				log.Printf("Failed to send message to client %s: channel full or closed", msg.RecipientID)
			}
		}
		return
	}

	if msg.RecipientID == "server" {
		switch msg.Type {
		case "offer":
			s.handleOffer(msg)
		case "answer":
			s.handleAnswer(msg)
		case "ice_candidate":
			s.handleICECandidate(msg)
		case "ping":
			s.handlePing(msg)
		}
		return
	}

	s.streamsMutex.RLock()
	stream, exists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if !exists {
		log.Printf("Stream %s not found", msg.StreamID)
		return
	}

	stream.Mutex.RLock()
	for clientID, client := range stream.Clients {
		if clientID != msg.SenderID && client.Connected {
			select {
			case client.SendChannel <- msg:
				// Message sent
			default:
				log.Printf("Failed to broadcast message to client %s: channel full or closed", clientID)
			}
		}
	}
	stream.Mutex.RUnlock()
}

// handleOffer handles SDP offer messages
func (s *SignalingService) handleOffer(msg model.SignalingMessage) {
	log.Printf("Received offer from %s for stream %s", msg.SenderID, msg.StreamID)

	offerSDP, err := extractSDP(msg.Payload)
	if err != nil {
		log.Printf("Error extracting SDP: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Could not parse offer payload")
		return
	}

	log.Printf("SDP offer length: %d bytes", len(offerSDP))

	if !isValidSDP(offerSDP) {
		log.Printf("Invalid SDP format")
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "invalid_offer", "Invalid SDP format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	webrtcClient, conn, err := s.getWebRTCServiceClient()
	if err != nil {
		log.Printf("Failed to connect to WebRTC-out service: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "service_unavailable", "WebRTC service is currently unavailable")
		return
	}
	defer conn.Close()

	log.Printf("Sending offer to WebRTC-out service for stream %s (size: %d bytes)", msg.StreamID, len(offerSDP))
	req := &webrtcpb.OfferRequest{
		StreamId: msg.StreamID,
		ViewerId: msg.SenderID,
		Offer:    offerSDP,
	}

	resp, err := webrtcClient.HandleOffer(ctx, req)
	if err != nil {
		log.Printf("Failed to handle offer: %v", err)
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

	if !isValidSDP(resp.Answer) {
		log.Printf("Invalid SDP answer received from WebRTC service")
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "internal_error", "Invalid answer received from WebRTC service")
		return
	}

	log.Printf("Received valid SDP answer from WebRTC service, length: %d bytes", len(resp.Answer))

	answerData := map[string]string{
		"sdp": resp.Answer,
	}

	answerBytes, err := json.Marshal(answerData)
	if err != nil {
		log.Printf("Failed to marshal answer data: %v", err)
		s.sendErrorResponse(msg.SenderID, msg.StreamID, "internal_error", "Internal server error")
		return
	}

	answer := model.SignalingMessage{
		Type:        "answer",
		StreamID:    msg.StreamID,
		SenderID:    "server",
		RecipientID: msg.SenderID,
		Payload:     answerBytes,
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
	s.streamsMutex.RLock()
	stream, exists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if !exists {
		log.Printf("Stream %s not found for answer", msg.StreamID)
		return
	}

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

	s.streamsMutex.RLock()
	stream, exists := s.streams[msg.StreamID]
	s.streamsMutex.RUnlock()

	if !exists {
		log.Printf("Stream %s not found for ICE candidate", msg.StreamID)
		return
	}

	s.clientsMutex.RLock()
	sender, senderExists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if !senderExists {
		log.Printf("Sender %s not found for ICE candidate", msg.SenderID)
		return
	}

	isPublisher := sender.IsPublisher

	stream.Mutex.RLock()
	for clientID, client := range stream.Clients {
		if clientID != msg.SenderID && client.IsPublisher != isPublisher {
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
	pong := model.SignalingMessage{
		Type:        "pong",
		StreamID:    msg.StreamID,
		SenderID:    "server",
		RecipientID: msg.SenderID,
		Payload:     msg.Payload,
	}

	s.clientsMutex.RLock()
	client, exists := s.clients[msg.SenderID]
	s.clientsMutex.RUnlock()

	if exists && client.Connected {
		select {
		case client.SendChannel <- pong:
			// Pong sent successfully
		default:
			log.Printf("Failed to send pong to client %s: channel full or closed", msg.SenderID)
		}
	}
}

// getWebRTCServiceClient gets a client for the WebRTC service
func (s *SignalingService) getWebRTCServiceClient() (webrtcpb.WebRTCServiceClient, *grpc.ClientConn, error) {
	var conn *grpc.ClientConn
	var err error
	maxRetries := 3

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithTimeout(10 * time.Second),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(20*1024*1024),
			grpc.MaxCallSendMsgSize(20*1024*1024),
		),
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		conn, err = grpc.Dial(s.cfg.WebRTCOut.Address, dialOpts...)
		if err == nil {
			break
		}

		if attempt < maxRetries-1 {
			waitTime := time.Duration(attempt+1) * time.Second
			log.Printf("Failed to connect to WebRTC service (attempt %d/%d), retrying in %v: %v",
				attempt+1, maxRetries, waitTime, err)
			time.Sleep(waitTime)
		}
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to WebRTC service after %d attempts: %w", maxRetries, err)
	}

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
	defer stream.Mutex.RUnlock()
	return len(stream.Clients)
}

// GetTotalClientCount returns the total number of connected clients
func (s *SignalingService) GetTotalClientCount() int {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	connected := 0
	for _, client := range s.clients {
		if client.Connected {
			connected++
		}
	}
	return connected
}

// cleanupInactiveConnections removes inactive connections
func (s *SignalingService) cleanupInactiveConnections(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performCleanup()
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		}
	}
}

// performCleanup removes inactive connections
func (s *SignalingService) performCleanup() {
	now := time.Now()
	var disconnectedClients []string

	s.clientsMutex.RLock()
	for clientID, client := range s.clients {
		if now.Sub(client.LastActivity) > 10*time.Minute {
			disconnectedClients = append(disconnectedClients, clientID)
		}
	}
	s.clientsMutex.RUnlock()

	for _, clientID := range disconnectedClients {
		log.Printf("Cleaning up inactive client: %s", clientID)
		s.UnregisterClient(clientID)
	}
}

// Helper functions for SDP handling
func extractSDP(payload interface{}) (string, error) {
	var payloadStr string
	switch p := payload.(type) {
	case string:
		payloadStr = p
	case []byte:
		payloadStr = string(p)
	default:
		if jsonBytes, err := json.Marshal(payload); err == nil {
			payloadStr = string(jsonBytes)
		} else {
			return "", fmt.Errorf("unsupported payload type: %T", payload)
		}
	}

	payloadStr = strings.TrimSpace(payloadStr)

	if strings.HasPrefix(payloadStr, "v=0") {
		return payloadStr, nil
	}

	if strings.HasPrefix(payloadStr, "{") {
		var sdpObj map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &sdpObj); err == nil {
			if sdp, ok := sdpObj["sdp"].(string); ok {
				return strings.TrimSpace(sdp), nil
			}
		}
	}

	if strings.HasPrefix(payloadStr, "\"") && strings.HasSuffix(payloadStr, "\"") {
		var sdpStr string
		if err := json.Unmarshal([]byte(payloadStr), &sdpStr); err == nil {
			return strings.TrimSpace(sdpStr), nil
		}
	}

	return payloadStr, nil
}

func isValidSDP(sdp string) bool {
	sdp = strings.TrimSpace(sdp)
	if !strings.HasPrefix(sdp, "v=0") {
		return false
	}

	hasOrigin := strings.Contains(sdp, "o=")
	hasSession := strings.Contains(sdp, "s=")
	hasConnection := strings.Contains(sdp, "c=")
	hasMedia := strings.Contains(sdp, "m=audio") || strings.Contains(sdp, "m=video")

	return hasOrigin && hasSession && hasConnection && hasMedia
}
