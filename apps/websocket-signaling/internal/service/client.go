package service

import (
	"context"
	"fmt"
	"log"
	"time"

	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// GetClientByID gets a client by ID
func (s *Service) GetClientByID(clientID string) (*Client, error) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	return client, nil
}
