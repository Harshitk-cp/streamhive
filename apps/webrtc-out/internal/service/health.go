package service

import (
	"context"
	"time"

	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
)

// IsReady checks if the service is ready to handle requests
func (s *Service) IsReady() bool {
	// Check if signaling connection is active
	if s.signalingConn == nil || s.signalingClient == nil {
		return false
	}

	// Service is ready if it has been initialized successfully
	return true
}

// PingSignalingService checks if the signaling service is reachable
func (s *Service) PingSignalingService(ctx context.Context) error {
	// Create a heartbeat request
	req := &webrtcPb.NodeHeartbeatRequest{
		NodeId:            s.config.Service.NodeID,
		ActiveStreams:     0,
		ActiveConnections: 0,
		CpuUsage:          0.0,
		MemoryUsage:       0.0,
	}

	// Set a short timeout
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Send heartbeat
	_, err := s.signalingClient.SendNodeHeartbeat(pingCtx, req)
	return err
}

// GetStreamStats gets stats for a stream with error handling
func (s *Service) GetStreamStat(ctx context.Context, streamID string) (*webrtcPb.GetStreamStatsResponse, error) {
	// Create request
	req := &webrtcPb.GetStreamStatsRequest{
		StreamId: streamID,
	}

	// Get stream stats
	return s.GetStreamStats(ctx, req)
}

// GetMaxQueueSize returns the maximum queue size for streams
func (s *Service) GetMaxQueueSize() int {
	return s.config.FrameProcessing.MaxQueueSize
}

// GetStreamCount returns the number of active streams
func (s *Service) GetStreamCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.streams)
}

// GetConnectionCount returns the total number of active peer connections
func (s *Service) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, stream := range s.streams {
		stream.mu.RLock()
		count += len(stream.PeerConnections)
		stream.mu.RUnlock()
	}
	return count
}

// GetStreamIDs returns the IDs of all active streams
func (s *Service) GetStreamIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.streams))
	for id := range s.streams {
		ids = append(ids, id)
	}
	return ids
}
