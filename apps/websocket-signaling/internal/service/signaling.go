package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// mustEmbedUnimplementedSignalingServiceServer implements webrtc.SignalingServiceServer.
func (s *Service) mustEmbedUnimplementedSignalingServiceServer() {
	panic("unimplemented")
}

// New creates a new signaling service
func New(cfg *config.Config) (*Service, error) {
	h := hub.NewHub()

	service := &Service{
		config:        cfg,
		hub:           h,
		streams:       make(map[string]*Stream),
		nodes:         make(map[string]*WebRTCNode),
		clients:       make(map[string]*Client),
		stopChan:      make(chan struct{}),
		cleanupTicker: time.NewTicker(30 * time.Second),
	}

	// Start the WebSocket hub
	go h.Run()

	// Start cleanup routine
	go service.cleanup()

	return service, nil
}

// cleanup periodically checks for stale nodes and streams
func (s *Service) cleanup() {
	for {
		select {
		case <-s.cleanupTicker.C:
			now := time.Now()

			// Check for stale nodes
			s.nodesMu.Lock()
			for id, node := range s.nodes {
				if now.Sub(node.LastHeartbeat) > 1*time.Minute {
					log.Printf("Removing stale WebRTC node %s", id)
					// Close connection
					if node.Conn != nil {
						node.Conn.Close()
					}
					// Remove node
					delete(s.nodes, id)
				}
			}
			s.nodesMu.Unlock()

		case <-s.stopChan:
			s.cleanupTicker.Stop()
			return
		}
	}
}

// RegisterStream registers a stream with the signaling service
func (s *Service) RegisterStream(ctx context.Context, req *webrtcPb.RegisterStreamRequest) (*webrtcPb.RegisterStreamResponse, error) {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()

	// Check if node exists
	s.nodesMu.RLock()
	_, nodeExists := s.nodes[req.NodeId]
	s.nodesMu.RUnlock()

	if !nodeExists {
		return nil, status.Errorf(codes.NotFound, "node not found: %s", req.NodeId)
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

// RegisterWebRTCNode registers a WebRTC node with the signaling service
func (s *Service) RegisterWebRTCNode(ctx context.Context, req *webrtcPb.RegisterWebRTCNodeRequest) (*webrtcPb.RegisterWebRTCNodeResponse, error) {
	// Connect to the WebRTC node
	conn, err := grpc.Dial(req.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to connect to WebRTC node: %v", err)
	}

	// Create WebRTC client
	client := webrtcPb.NewWebRTCServiceClient(conn)

	// Register node
	s.nodesMu.Lock()
	defer s.nodesMu.Unlock()

	node := &WebRTCNode{
		ID:            req.NodeId,
		Address:       req.Address,
		Capabilities:  req.Capabilities,
		MaxStreams:    req.MaxStreams,
		LastHeartbeat: time.Now(),
		Client:        client,
		Conn:          conn,
	}

	s.nodes[req.NodeId] = node

	log.Printf("Registered WebRTC node %s at %s", req.NodeId, req.Address)

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
		return nil, status.Errorf(codes.NotFound, "node not found: %s", req.NodeId)
	}

	// Update node status
	node.LastHeartbeat = time.Now()
	node.ActiveStreams = req.ActiveStreams
	node.Connections = req.ActiveConnections
	node.CPUUsage = req.CpuUsage
	node.MemoryUsage = req.MemoryUsage

	return &webrtcPb.NodeHeartbeatResponse{
		Success:    true,
		ServerTime: time.Now().UnixNano() / int64(time.Millisecond),
	}, nil
}

// ForwardSignalingMessage forwards a signaling message to a client
func (s *Service) ForwardSignalingMessage(ctx context.Context, req *webrtcPb.SignalingForwardRequest) (*webrtcPb.SignalingForwardResponse, error) {
	// Check if client exists
	s.clientsMu.RLock()
	client, exists := s.clients[req.RecipientId]
	s.clientsMu.RUnlock()

	if !exists || !client.Connected {
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
		return nil, status.Errorf(codes.Internal, "failed to marshal message: %v", err)
	}

	// Send message to client
	client.Hub.SendToClient(client.ID, jsonMessage)

	return &webrtcPb.SignalingForwardResponse{
		Success: true,
	}, nil
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
