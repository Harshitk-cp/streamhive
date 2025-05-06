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
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/hub"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Client represents a connected WebSocket client
type Client struct {
	ID        string
	StreamID  string
	SessionID string
	Connected bool
	Hub       *hub.Hub
}

// WebRTCNode represents a WebRTC node
type WebRTCNode struct {
	ID            string
	Address       string
	Capabilities  []string
	MaxStreams    int32
	LastHeartbeat time.Time
	ActiveStreams int32
	Connections   int32
	CPUUsage      float32
	MemoryUsage   float32
	Client        webrtcPb.WebRTCServiceClient
	Conn          *grpc.ClientConn
	// Track registration status separately from connection
	Registered bool
}

// Stream represents a stream that clients can connect to
type Stream struct {
	ID        string
	SessionID string
	NodeID    string
	Active    bool
	Clients   map[string]*Client
}

// Service implements the SignalingService gRPC interface
type Service struct {
	config        *config.Config
	hub           *hub.Hub
	streams       map[string]*Stream
	nodes         map[string]*WebRTCNode
	clients       map[string]*Client
	streamsMu     sync.RWMutex
	nodesMu       sync.RWMutex
	clientsMu     sync.RWMutex
	stopChan      chan struct{}
	cleanupTicker *time.Ticker

	webrtcPb.UnimplementedSignalingServiceServer
}

// New creates a new signaling service
func New(cfg *config.Config) (*Service, error) {
	h := hub.NewHub()
	go h.Run()

	service := &Service{
		config:        cfg,
		hub:           h,
		streams:       make(map[string]*Stream),
		nodes:         make(map[string]*WebRTCNode),
		clients:       make(map[string]*Client),
		stopChan:      make(chan struct{}),
		cleanupTicker: time.NewTicker(30 * time.Second),
	}

	// Start cleanup routine
	go service.cleanup()

	log.Printf("WebSocket signaling service initialized with node ID: %s", cfg.Service.NodeID)
	log.Printf("Listening for gRPC connections on %s", cfg.GRPC.Address)
	log.Printf("Listening for HTTP/WebSocket connections on %s", cfg.HTTP.Address)

	return service, nil
}

// RegisterWebRTCNode registers a WebRTC node with the signaling service
func (s *Service) RegisterWebRTCNode(ctx context.Context, req *webrtcPb.RegisterWebRTCNodeRequest) (*webrtcPb.RegisterWebRTCNodeResponse, error) {
	log.Printf("Received registration request from WebRTC node: %s at %s", req.NodeId, req.Address)

	// Validate request
	if req.NodeId == "" {
		log.Printf("Registration failed: empty node ID")
		return nil, status.Errorf(codes.InvalidArgument, "node ID cannot be empty")
	}

	if req.Address == "" {
		log.Printf("Registration failed: empty address")
		return nil, status.Errorf(codes.InvalidArgument, "node address cannot be empty")
	}

	// Ensure the address is in proper network format (hostname:port)
	address := req.Address
	if strings.HasPrefix(address, ":") {
		log.Printf("Warning: Address %s starts with ':' which is not a valid network address. Attempting to fix...", address)
		// This is a local port reference, not a valid network address
		// We can't connect to this directly, so we'll use the node's ID or try to derive a hostname
		address = fmt.Sprintf("%s%s", req.NodeId, address)
		log.Printf("Corrected address to: %s", address)
	}

	// Create a connection to the WebRTC node with retry logic
	log.Printf("Attempting to connect to WebRTC node at %s", address)

	var conn *grpc.ClientConn
	var client webrtcPb.WebRTCServiceClient
	var connectionError error

	// Retry connection a few times with backoff
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
		conn, connectionError = grpc.DialContext(
			dialCtx,
			address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		dialCancel()

		if connectionError == nil {
			client = webrtcPb.NewWebRTCServiceClient(conn)
			log.Printf("Successfully connected to WebRTC node at %s", address)
			break
		}

		log.Printf("Connection attempt %d/%d to WebRTC node failed: %v. Retrying...",
			retry+1, maxRetries, connectionError)

		if retry < maxRetries-1 {
			time.Sleep(time.Duration(1<<retry) * time.Second) // Exponential backoff
		}
	}

	// If we couldn't connect after all retries, log but continue with registration
	if connectionError != nil {
		log.Printf("Failed to connect to WebRTC node after %d attempts: %v", maxRetries, connectionError)
		log.Printf("Registering node anyway, will try to reconnect during heartbeats")
	}

	// Register node
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	// Check if node already exists and close old connection if it does
	if oldNode, exists := s.nodes[req.NodeId]; exists && oldNode.Conn != nil {
		log.Printf("Node %s already registered, closing old connection", req.NodeId)
		oldNode.Conn.Close()
	}

	node := &WebRTCNode{
		ID:            req.NodeId,
		Address:       address, // Store the corrected address
		Capabilities:  req.Capabilities,
		MaxStreams:    req.MaxStreams,
		LastHeartbeat: time.Now(),
		Client:        client,
		Conn:          conn,
		Registered:    true,
	}

	s.nodes[req.NodeId] = node

	log.Printf("Successfully registered WebRTC node %s at %s", req.NodeId, address)

	return &webrtcPb.RegisterWebRTCNodeResponse{
		Success: true,
	}, nil
}

// SendNodeHeartbeat processes a heartbeat from a WebRTC node
func (s *Service) SendNodeHeartbeat(ctx context.Context, req *webrtcPb.NodeHeartbeatRequest) (*webrtcPb.NodeHeartbeatResponse, error) {
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	node, exists := s.nodes[req.NodeId]
	if !exists {
		// Node doesn't exist, so create it on heartbeat
		log.Printf("Heartbeat from unknown node: %s. Re-registering node.", req.NodeId)

		// Create a new node entry since we didn't find it
		// We don't have the address or capabilities, but we'll just use defaults
		// The node will properly re-register on its next startup
		node = &WebRTCNode{
			ID:            req.NodeId,
			Address:       req.NodeId + ":9094", // Default guess, will be corrected on proper registration
			LastHeartbeat: time.Now(),
			Registered:    true,
		}
		s.nodes[req.NodeId] = node
	}

	// Update node status
	node.LastHeartbeat = time.Now()
	node.ActiveStreams = req.ActiveStreams
	node.Connections = req.ActiveConnections
	node.CPUUsage = req.CpuUsage
	node.MemoryUsage = req.MemoryUsage

	// Debug log every 5 minutes (reduce log noise)
	if node.LastHeartbeat.Minute()%5 == 0 && node.LastHeartbeat.Second() < 10 {
		log.Printf("Heartbeat from node %s: %d streams, %d connections",
			req.NodeId, req.ActiveStreams, req.ActiveConnections)
	}

	return &webrtcPb.NodeHeartbeatResponse{
		Success:    true,
		ServerTime: time.Now().UnixNano() / int64(time.Millisecond),
	}, nil
}

// RegisterStream registers a stream with the signaling service
func (s *Service) RegisterStream(ctx context.Context, req *webrtcPb.RegisterStreamRequest) (*webrtcPb.RegisterStreamResponse, error) {
	log.Printf("Registering stream %s on node %s", req.StreamId, req.NodeId)

	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()

	// Check if node exists
	s.nodesMu.RLock()
	_, nodeExists := s.nodes[req.NodeId]
	s.nodesMu.RUnlock()

	if !nodeExists {
		log.Printf("Node %s not found for stream registration. Creating node record.", req.NodeId)
		// Auto-create the node since it's trying to register a stream
		s.nodesMu.Lock()
		s.nodes[req.NodeId] = &WebRTCNode{
			ID:            req.NodeId,
			Address:       req.NodeId + ":9094", // Default guess
			LastHeartbeat: time.Now(),
			Registered:    true,
		}
		s.nodesMu.Unlock()
	}

	// Create or update stream
	stream, exists := s.streams[req.StreamId]
	if !exists {
		stream = &Stream{
			ID:        req.StreamId,
			SessionID: req.SessionId,
			NodeID:    req.NodeId,
			Active:    true,
			Clients:   make(map[string]*Client),
		}
		s.streams[req.StreamId] = stream
		log.Printf("Registered new stream %s on node %s", req.StreamId, req.NodeId)
	} else {
		// Update existing stream
		stream.NodeID = req.NodeId
		stream.SessionID = req.SessionId
		stream.Active = true
		log.Printf("Updated stream %s to node %s", req.StreamId, req.NodeId)
	}

	return &webrtcPb.RegisterStreamResponse{
		Success: true,
	}, nil
}

// UnregisterStream unregisters a stream from the signaling service
func (s *Service) UnregisterStream(ctx context.Context, req *webrtcPb.UnregisterStreamRequest) (*webrtcPb.UnregisterStreamResponse, error) {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()

	stream, exists := s.streams[req.StreamId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "stream not found: %s", req.StreamId)
	}

	// Check session ID
	if stream.SessionID != req.SessionId {
		return nil, status.Errorf(codes.PermissionDenied, "session ID mismatch")
	}

	// Check node ID
	if stream.NodeID != req.NodeId {
		return nil, status.Errorf(codes.PermissionDenied, "node ID mismatch")
	}

	// Mark stream as inactive
	stream.Active = false

	// Notify all connected clients that the stream has ended
	for _, client := range stream.Clients {
		if client.Connected {
			message := map[string]interface{}{
				"type":      "stream_ended",
				"stream_id": req.StreamId,
			}
			jsonMessage, _ := json.Marshal(message)
			client.Hub.SendToClient(client.ID, jsonMessage)
		}
	}

	// Remove stream
	delete(s.streams, req.StreamId)

	log.Printf("Unregistered stream %s from node %s", req.StreamId, req.NodeId)

	return &webrtcPb.UnregisterStreamResponse{
		Success: true,
	}, nil
}

// ForwardSignalingMessage forwards a signaling message to a client
func (s *Service) ForwardSignalingMessage(ctx context.Context, req *webrtcPb.SignalingForwardRequest) (*webrtcPb.SignalingForwardResponse, error) {
	// Check if stream exists
	s.streamsMu.RLock()
	_, streamExists := s.streams[req.StreamId]
	s.streamsMu.RUnlock()

	if !streamExists {
		log.Printf("Failed to forward message: stream not found: %s", req.StreamId)
		return nil, status.Errorf(codes.NotFound, "stream not found: %s", req.StreamId)
	}

	// Check if client exists
	s.clientsMu.RLock()
	client, exists := s.clients[req.RecipientId]
	s.clientsMu.RUnlock()

	if !exists || !client.Connected {
		log.Printf("Failed to forward message: client not found or not connected: %s", req.RecipientId)
		return nil, status.Errorf(codes.NotFound, "client not found or not connected: %s", req.RecipientId)
	}

	// Create message
	message := map[string]interface{}{
		"type":         req.MessageType,
		"stream_id":    req.StreamId,
		"sender_id":    req.SenderId,
		"recipient_id": req.RecipientId,
		"payload":      string(req.Payload),
	}

	// Convert to JSON
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to marshal message: %v", err)
	}

	// Send message to client
	client.Hub.SendToClient(client.ID, jsonMessage)
	log.Printf("Forwarded message of type %s from %s to %s for stream %s",
		req.MessageType, req.SenderId, req.RecipientId, req.StreamId)

	return &webrtcPb.SignalingForwardResponse{
		Success: true,
	}, nil
}

// RegisterClient registers a client with the signaling service
func (s *Service) RegisterClient(ctx context.Context, req *webrtcPb.RegisterClientRequest) (*webrtcPb.RegisterClientResponse, error) {
	// Check if stream exists and is active
	s.streamsMu.RLock()
	stream, streamExists := s.streams[req.StreamId]
	streamActive := streamExists && stream.Active
	nodeID := ""
	if streamExists {
		nodeID = stream.NodeID
	}
	s.streamsMu.RUnlock()

	// If stream doesn't exist in our local cache, query all WebRTC nodes
	// to see if any of them have the stream registered
	if !streamExists || !streamActive {
		// Make a copy of nodes to avoid holding the lock during RPC calls
		s.nodesMu.RLock()
		nodesCopy := make(map[string]*WebRTCNode, len(s.nodes))
		for k, v := range s.nodes {
			nodesCopy[k] = v
		}
		s.nodesMu.RUnlock()

		// Check each node for the stream
		var foundNode *WebRTCNode
		for _, node := range nodesCopy {
			// Skip nodes with no client (connection failed)
			if node.Client == nil {
				continue
			}

			// Call the node's GetStreamStats method to check if it has the stream
			statsCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			resp, err := node.Client.GetStreamStats(statsCtx, &webrtcPb.GetStreamStatsRequest{
				StreamId: req.StreamId,
			})
			cancel()

			if err == nil && resp != nil {
				// Stream found on this node
				foundNode = node
				nodeID = node.ID
				break
			}
		}

		if foundNode != nil {
			// Stream exists on a node, register it locally
			s.streamsMu.Lock()
			s.streams[req.StreamId] = &Stream{
				ID:        req.StreamId,
				SessionID: req.SessionId,
				NodeID:    nodeID,
				Active:    true,
				Clients:   make(map[string]*Client),
			}
			s.streamsMu.Unlock()
			log.Printf("Registered stream %s with node %s after discovery", req.StreamId, nodeID)
			streamExists = true
			streamActive = true
		}
	}

	// If stream still doesn't exist or is not active, return an error
	if !streamExists || !streamActive {
		log.Printf("Failed to register client: stream not found: %s", req.StreamId)
		return nil, status.Errorf(codes.NotFound, "stream not found: %s", req.StreamId)
	}

	// Create a new client
	client := &Client{
		ID:        req.ClientId,
		StreamID:  req.StreamId,
		SessionID: req.SessionId,
		Connected: true,
		Hub:       s.hub,
	}

	// Register client
	s.clientsMu.Lock()
	s.clients[req.ClientId] = client
	s.clientsMu.Unlock()

	// Add client to stream
	s.streamsMu.Lock()
	s.streams[req.StreamId].Clients[req.ClientId] = client
	s.streamsMu.Unlock()

	log.Printf("Registered client %s for stream %s on node %s", req.ClientId, req.StreamId, nodeID)

	return &webrtcPb.RegisterClientResponse{
		Success: true,
	}, nil
}

// UnregisterClient unregisters a client from the signaling service
func (s *Service) UnregisterClient(ctx context.Context, req *webrtcPb.UnregisterClientRequest) (*webrtcPb.UnregisterClientResponse, error) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	client, exists := s.clients[req.ClientId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "client not found: %s", req.ClientId)
	}

	// Check stream ID
	if client.StreamID != req.StreamId {
		return nil, status.Errorf(codes.PermissionDenied, "stream ID mismatch")
	}

	// Mark client as disconnected
	client.Connected = false

	// Remove client
	delete(s.clients, req.ClientId)

	// Remove client from stream
	s.streamsMu.Lock()
	if stream, streamExists := s.streams[req.StreamId]; streamExists {
		delete(stream.Clients, req.ClientId)
	}
	s.streamsMu.Unlock()

	log.Printf("Unregistered client %s from stream %s", req.ClientId, req.StreamId)

	return &webrtcPb.UnregisterClientResponse{
		Success: true,
	}, nil
}

// cleanup periodically checks for stale nodes and streams and attempts to reconnect to nodes
func (s *Service) cleanup() {
	for {
		select {
		case <-s.cleanupTicker.C:
			now := time.Now()

			// Check for stale nodes and attempt to reconnect to nodes with failed connections
			s.nodesMu.Lock()
			for id, node := range s.nodes {
				// Check for stale nodes - inactive for over 2 minutes with no heartbeat
				// The 2-minute timeout allows for temporary network disruptions
				if now.Sub(node.LastHeartbeat) > 2*time.Minute && node.Registered {
					log.Printf("Warning: Node %s has not sent a heartbeat for %v",
						id, now.Sub(node.LastHeartbeat))

					// Don't remove the node immediately, just try to reconnect
					// Only after 5 minutes of no activity would we consider removing it
					if now.Sub(node.LastHeartbeat) > 5*time.Minute {
						log.Printf("Removing stale WebRTC node %s after 5 minutes of inactivity", id)
						// Close connection
						if node.Conn != nil {
							node.Conn.Close()
						}
						// Remove node
						delete(s.nodes, id)
						continue
					}
				}

				// Try to reconnect to nodes with failed connections
				if node.Registered && node.Client == nil && node.Conn == nil {
					log.Printf("Attempting to reconnect to WebRTC node %s at %s", id, node.Address)

					// Try to connect with timeout
					dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
					conn, err := grpc.DialContext(
						dialCtx,
						node.Address,
						grpc.WithTransportCredentials(insecure.NewCredentials()),
						grpc.WithBlock(),
					)
					dialCancel()

					if err == nil {
						log.Printf("Successfully reconnected to WebRTC node %s", id)
						node.Conn = conn
						node.Client = webrtcPb.NewWebRTCServiceClient(conn)
						// Reset heartbeat time since we've confirmed it's active
						node.LastHeartbeat = time.Now()
					} else {
						log.Printf("Failed to reconnect to WebRTC node %s: %v", id, err)
					}
				}
			}
			s.nodesMu.Unlock()

		case <-s.stopChan:
			s.cleanupTicker.Stop()
			return
		}
	}
}

// GetNodeForStream returns the WebRTC node for a stream
func (s *Service) GetNodeForStream(streamID string) (*WebRTCNode, error) {
	s.streamsMu.RLock()
	stream, exists := s.streams[streamID]
	s.streamsMu.RUnlock()

	if !exists || !stream.Active {
		return nil, fmt.Errorf("stream not found or not active: %s", streamID)
	}

	s.nodesMu.RLock()
	node, exists := s.nodes[stream.NodeID]
	s.nodesMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("node not found for stream: %s", streamID)
	}

	return node, nil
}

// GetStatus returns the current status of the service
func (s *Service) GetStatus() (map[string]*WebRTCNode, map[string]*Stream, map[string]*Client) {
	s.nodesMu.RLock()
	nodes := make(map[string]*WebRTCNode, len(s.nodes))
	for k, v := range s.nodes {
		nodes[k] = v
	}
	s.nodesMu.RUnlock()

	s.streamsMu.RLock()
	streams := make(map[string]*Stream, len(s.streams))
	for k, v := range s.streams {
		streams[k] = v
	}
	s.streamsMu.RUnlock()

	s.clientsMu.RLock()
	clients := make(map[string]*Client, len(s.clients))
	for k, v := range s.clients {
		clients[k] = v
	}
	s.clientsMu.RUnlock()

	return nodes, streams, clients
}

// GetNodes returns all registered WebRTC nodes
func (s *Service) GetNodes() map[string]*WebRTCNode {
	s.nodesMu.RLock()
	defer s.nodesMu.RUnlock()

	nodes := make(map[string]*WebRTCNode, len(s.nodes))
	for k, v := range s.nodes {
		nodes[k] = v
	}

	return nodes
}

// GetStreams returns all registered streams
func (s *Service) GetStreams() map[string]*Stream {
	s.streamsMu.RLock()
	defer s.streamsMu.RUnlock()

	streams := make(map[string]*Stream, len(s.streams))
	for k, v := range s.streams {
		streams[k] = v
	}

	return streams
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	// Signal goroutines to stop
	close(s.stopChan)

	// Close all node connections
	s.nodesMu.Lock()
	for _, node := range s.nodes {
		if node.Conn != nil {
			node.Conn.Close()
		}
	}
	s.nodesMu.Unlock()

	// Stop WebSocket hub
	s.hub.Close()

	return nil
}
