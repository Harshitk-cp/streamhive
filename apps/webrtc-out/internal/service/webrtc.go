package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/metrics"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Stream struct {
	ID              string
	SessionID       string
	VideoTrack      *webrtc.TrackLocalStaticSample
	AudioTrack      *webrtc.TrackLocalStaticSample
	PeerConnections map[string]*PeerConnection
	VideoFrameQueue chan []byte
	AudioFrameQueue chan []byte
	MaxQueueSize    int
	DropWhenFull    bool
	mu              sync.RWMutex
}

// createPeerConnection creates a new peer connection for a viewer
func (s *Service) createPeerConnection(streamID, sessionID, userID string) (*webrtc.PeerConnection, error) {
	s.mu.RLock()
	stream, exists := s.streams[streamID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamID)
	}

	// Create ICE servers config
	iceServers := []webrtc.ICEServer{}
	for _, server := range s.config.WebRTC.ICEServers {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       server.URLs,
			Username:   server.Username,
			Credential: server.Credential,
		})
	}

	// Create WebRTC configuration
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// Create peer connection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Add video track
	videoSender, err := peerConnection.AddTrack(stream.VideoTrack)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add video track: %w", err)
	}

	// Handle RTCP packets for video
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := videoSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Add audio track
	audioSender, err := peerConnection.AddTrack(stream.AudioTrack)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add audio track: %w", err)
	}

	// Handle RTCP packets for audio
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := audioSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Set up event handlers
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE connection state changed: %s (User: %s, Stream: %s)", state.String(), userID, streamID)

		// Update last activity time
		stream.mu.Lock()
		if peer, exists := stream.PeerConnections[userID]; exists {
			peer.LastActivity = time.Now()
		}
		stream.mu.Unlock()

		switch state {
		case webrtc.ICEConnectionStateConnected:
			s.metrics.PeerConnected(streamID, userID)
		case webrtc.ICEConnectionStateDisconnected, webrtc.ICEConnectionStateFailed, webrtc.ICEConnectionStateClosed:
			s.metrics.PeerDisconnected(streamID, userID)

			// Remove peer connection if disconnected or failed
			if state == webrtc.ICEConnectionStateDisconnected || state == webrtc.ICEConnectionStateFailed {
				go s.cleanupPeerConnection(streamID, userID)
			}
		}
	})

	// Create a new peer connection object
	peer := &PeerConnection{
		ID:             fmt.Sprintf("%s-%s", streamID, userID),
		UserID:         userID,
		Connection:     peerConnection,
		VideoSender:    videoSender,
		AudioSender:    audioSender,
		ConnectionTime: time.Now(),
		LastActivity:   time.Now(),
	}

	// Add to stream's peer connections
	stream.mu.Lock()
	stream.PeerConnections[userID] = peer
	stream.mu.Unlock()

	// Log the creation of the peer connection
	log.Printf("Created peer connection for user %s in stream %s with sessionId: %s", userID, streamID, sessionID)

	return peerConnection, nil
}

// cleanupPeerConnection cleans up resources associated with a peer connection
func (s *Service) cleanupPeerConnection(streamID, userID string) {
	s.mu.RLock()
	stream, exists := s.streams[streamID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	peer, exists := stream.PeerConnections[userID]
	if !exists {
		return
	}

	// Close the peer connection
	if err := peer.Connection.Close(); err != nil {
		log.Printf("Error closing peer connection: %v", err)
	}

	// Remove from peer connections map
	delete(stream.PeerConnections, userID)

	log.Printf("Cleaned up peer connection for user %s in stream %s", userID, streamID)
}

// HandleOffer handles a SDP offer from a viewer and generates an answer
func (s *Service) HandleOffer(ctx context.Context, req *webrtcPb.SDPOfferRequest) (*webrtcPb.SDPAnswerResponse, error) {
	// Create peer connection
	peerConnection, err := s.createPeerConnection(req.StreamId, req.SessionId, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Set remote description
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  req.Sdp,
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		// Clean up the peer connection
		s.cleanupPeerConnection(req.StreamId, req.UserId)
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		// Clean up the peer connection
		s.cleanupPeerConnection(req.StreamId, req.UserId)
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	// Set local description
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		// Clean up the peer connection
		s.cleanupPeerConnection(req.StreamId, req.UserId)
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	// Forward the SDP answer to the signaling service for delivery to client
	forwardReq := &webrtcPb.SignalingForwardRequest{
		StreamId:    req.StreamId,
		SenderId:    s.config.Service.NodeID,
		RecipientId: req.UserId,
		MessageType: "answer",
		Payload:     []byte(answer.SDP),
	}

	_, err = s.signalingClient.ForwardSignalingMessage(ctx, forwardReq)
	if err != nil {
		log.Printf("Warning: Failed to forward SDP answer through signaling service: %v", err)
		// Continue anyway as we'll return the answer directly in this response
	}

	return &webrtcPb.SDPAnswerResponse{
		StreamId:  req.StreamId,
		SessionId: req.SessionId,
		UserId:    req.UserId,
		Sdp:       answer.SDP,
	}, nil
}

// HandleICECandidate handles an ICE candidate from a viewer
func (s *Service) HandleICECandidate(ctx context.Context, req *webrtcPb.ICECandidateRequest) (*webrtcPb.ICECandidateResponse, error) {
	s.mu.RLock()
	stream, exists := s.streams[req.StreamId]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Get peer connection
	stream.mu.RLock()
	peer, exists := stream.PeerConnections[req.UserId]
	stream.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("peer connection not found for user: %s", req.UserId)
	}

	// Parse ICE candidate
	sdpMid := req.SdpMid
	candidate := webrtc.ICECandidateInit{
		Candidate: req.Candidate,
		SDPMid:    &sdpMid,
		SDPMLineIndex: func(v uint32) *uint16 {
			u := uint16(v)
			return &u
		}(req.SdpMLineIndex),
	}

	// Add ICE candidate
	if err := peer.Connection.AddICECandidate(candidate); err != nil {
		return nil, fmt.Errorf("failed to add ICE candidate: %w", err)
	}

	// Update last activity time
	stream.mu.Lock()
	peer.LastActivity = time.Now()
	stream.mu.Unlock()

	return &webrtcPb.ICECandidateResponse{
		Success: true,
	}, nil
}

// RestartICE restarts ICE for a peer connection
func (s *Service) RestartICE(ctx context.Context, req *webrtcPb.RestartICERequest) (*webrtcPb.RestartICEResponse, error) {
	s.mu.RLock()
	stream, exists := s.streams[req.StreamId]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Get peer connection
	stream.mu.RLock()
	peer, exists := stream.PeerConnections[req.UserId]
	stream.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("peer connection not found for user: %s", req.UserId)
	}

	// Create offer with ICE restart
	offer, err := peer.Connection.CreateOffer(&webrtc.OfferOptions{
		ICERestart: true,
	})
	if err != nil {
		return &webrtcPb.RestartICEResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create ICE restart offer: %v", err),
		}, nil
	}

	// Set local description
	if err := peer.Connection.SetLocalDescription(offer); err != nil {
		return &webrtcPb.RestartICEResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to set local description: %v", err),
		}, nil
	}

	// Forward the ICE restart offer to the signaling service for delivery to client
	forwardReq := &webrtcPb.SignalingForwardRequest{
		StreamId:    req.StreamId,
		SenderId:    s.config.Service.NodeID,
		RecipientId: req.UserId,
		MessageType: "ice-restart",
		Payload:     []byte(offer.SDP),
	}

	_, err = s.signalingClient.ForwardSignalingMessage(ctx, forwardReq)
	if err != nil {
		log.Printf("Warning: Failed to forward ICE restart offer through signaling service: %v", err)
		return &webrtcPb.RestartICEResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to forward ICE restart offer: %v", err),
		}, nil
	}

	// Update last activity time
	stream.mu.Lock()
	peer.LastActivity = time.Now()
	stream.mu.Unlock()

	return &webrtcPb.RestartICEResponse{
		Success: true,
	}, nil
}

// monitorPeerConnections periodically checks peer connections and removes stale ones
func (s *Service) monitorPeerConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			timeout := 2 * time.Minute

			s.mu.RLock()
			for streamID, stream := range s.streams {
				stream.mu.Lock()
				for userID, peer := range stream.PeerConnections {
					if now.Sub(peer.LastActivity) > timeout {
						log.Printf("Removing stale peer connection for user %s in stream %s", userID, streamID)

						// Close and remove peer connection
						if err := peer.Connection.Close(); err != nil {
							log.Printf("Error closing stale peer connection: %v", err)
						}
						delete(stream.PeerConnections, userID)
						s.metrics.PeerDisconnected(streamID, userID)
					}
				}
				stream.mu.Unlock()
			}
			s.mu.RUnlock()
		case <-s.stopChan:
			return
		}
	}
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	// Signal goroutines to stop
	close(s.stopChan)

	// Close all streams and peer connections
	s.mu.Lock()
	defer s.mu.Unlock()

	for streamID, stream := range s.streams {
		// Close all peer connections
		stream.mu.Lock()
		for userID, peer := range stream.PeerConnections {
			if err := peer.Connection.Close(); err != nil {
				log.Printf("Error closing peer connection for user %s: %v", userID, err)
			}
		}
		stream.mu.Unlock()

		// Close frame queues
		close(stream.VideoFrameQueue)
		close(stream.AudioFrameQueue)

		log.Printf("Closed stream %s", streamID)
	}

	// Clear streams map
	s.streams = make(map[string]*Stream)

	// Close signaling connection
	if s.signalingConn != nil {
		err := s.signalingConn.Close()
		if err != nil {
			log.Printf("Error closing signaling connection: %v", err)
		}
	}

	return nil
}

type PeerConnection struct {
	ID             string
	UserID         string
	Connection     *webrtc.PeerConnection
	VideoSender    *webrtc.RTPSender
	AudioSender    *webrtc.RTPSender
	ConnectionTime time.Time
	LastActivity   time.Time
}

// Service represents the WebRTC service
type Service struct {
	config            *config.Config
	metrics           metrics.Collector
	streams           map[string]*Stream
	mu                sync.RWMutex
	signalingConn     *grpc.ClientConn
	signalingClient   webrtcPb.SignalingServiceClient
	stopChan          chan struct{}
	heartbeatInterval time.Duration
}

// New creates a new WebRTC service
func New(cfg *config.Config, m metrics.Collector) (*Service, error) {
	// Connect to the signaling service
	conn, err := grpc.Dial(cfg.WebRTC.SignalingService,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to signaling service: %w", err)
	}

	service := &Service{
		config:            cfg,
		metrics:           m,
		streams:           make(map[string]*Stream),
		signalingConn:     conn,
		signalingClient:   webrtcPb.NewSignalingServiceClient(conn),
		stopChan:          make(chan struct{}),
		heartbeatInterval: 30 * time.Second,
	}

	// Register with signaling service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = service.RegisterWithSignalingService(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to register with signaling service: %w", err)
	}

	// Start connection monitoring
	go service.monitorPeerConnections()

	return service, nil
}

// RegisterWithSignalingService registers this WebRTC node with the signaling service
func (s *Service) RegisterWithSignalingService(ctx context.Context) error {
	// Register as a WebRTC server node
	registerReq := &webrtcPb.RegisterWebRTCNodeRequest{
		NodeId:       s.config.Service.NodeID,
		Address:      s.config.GRPC.Address,
		Capabilities: s.config.WebRTC.CodecPreferences,
		MaxStreams:   int32(100), // Default max streams
	}

	resp, err := s.signalingClient.RegisterWebRTCNode(ctx, registerReq)
	if err != nil {
		return fmt.Errorf("failed to register with signaling service: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration with signaling service failed")
	}

	// Start status update heartbeat
	go s.signalHeartbeat(ctx)

	log.Printf("Successfully registered with signaling service as node %s", s.config.Service.NodeID)
	return nil
}

// signalHeartbeat sends periodic heartbeats to the signaling service
func (s *Service) signalHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Count active streams and connections
			s.mu.RLock()
			activeStreams := len(s.streams)
			activeConnections := 0
			for _, stream := range s.streams {
				stream.mu.RLock()
				activeConnections += len(stream.PeerConnections)
				stream.mu.RUnlock()
			}
			s.mu.RUnlock()

			// Create heartbeat request
			req := &webrtcPb.NodeHeartbeatRequest{
				NodeId:            s.config.Service.NodeID,
				ActiveStreams:     int32(activeStreams),
				ActiveConnections: int32(activeConnections),
				// CPU and memory usage could be added here if available
				CpuUsage:    0.0,
				MemoryUsage: 0.0,
			}

			// Send heartbeat
			beatCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, err := s.signalingClient.SendNodeHeartbeat(beatCtx, req)
			cancel()

			if err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
			}

		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// CreateStream creates a new stream
func (s *Service) CreateStream(ctx context.Context, req *webrtcPb.CreateStreamRequest) (*webrtcPb.CreateStreamResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if stream already exists
	if _, exists := s.streams[req.StreamId]; exists {
		return nil, fmt.Errorf("stream already exists: %s", req.StreamId)
	}

	// Create video and audio tracks
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/vp8"},
		fmt.Sprintf("video-%s", req.StreamId),
		req.StreamId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "audio/opus"},
		fmt.Sprintf("audio-%s", req.StreamId),
		req.StreamId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	// Create stream
	stream := &Stream{
		ID:              req.StreamId,
		SessionID:       req.SessionId,
		VideoTrack:      videoTrack,
		AudioTrack:      audioTrack,
		PeerConnections: make(map[string]*PeerConnection),
		VideoFrameQueue: make(chan []byte, s.config.FrameProcessing.MaxQueueSize),
		AudioFrameQueue: make(chan []byte, s.config.FrameProcessing.MaxQueueSize),
		MaxQueueSize:    s.config.FrameProcessing.MaxQueueSize,
		DropWhenFull:    s.config.FrameProcessing.DropWhenFull,
	}

	// Start frame processing goroutines
	go s.processVideoFrames(stream)
	go s.processAudioFrames(stream)

	// Store stream
	s.streams[req.StreamId] = stream

	// Register stream with signaling service with retry logic
	err = s.RegisterStreamWithSignaling(ctx, req.StreamId, req.SessionId)
	if err != nil {
		// Close frame queues and clean up resources on failure
		close(stream.VideoFrameQueue)
		close(stream.AudioFrameQueue)
		delete(s.streams, req.StreamId)
		return nil, fmt.Errorf("failed to register stream with signaling service: %w", err)
	}

	// Verify the stream is registered
	registered, verifyErr := s.VerifyStreamRegistration(ctx, req.StreamId)
	if verifyErr != nil {
		log.Printf("Warning: Could not verify stream registration: %v", verifyErr)
	} else if !registered {
		log.Printf("Warning: Stream %s appears to not be registered with signaling service despite success response", req.StreamId)
	}

	log.Printf("Created stream %s with session %s", req.StreamId, req.SessionId)

	return &webrtcPb.CreateStreamResponse{
		StreamId:  req.StreamId,
		SessionId: req.SessionId,
		NodeId:    s.config.Service.NodeID,
	}, nil
}

// DestroyStream destroys a stream
func (s *Service) DestroyStream(ctx context.Context, req *webrtcPb.DestroyStreamRequest) (*webrtcPb.DestroyStreamResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get stream
	stream, exists := s.streams[req.StreamId]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Close all peer connections
	stream.mu.Lock()
	for userID, peer := range stream.PeerConnections {
		if err := peer.Connection.Close(); err != nil {
			log.Printf("Error closing peer connection for user %s: %v", userID, err)
		}
		s.metrics.PeerDisconnected(stream.ID, userID)
	}
	stream.mu.Unlock()

	// Unregister stream from signaling service
	err := s.UnregisterStreamFromSignaling(ctx, req.StreamId, req.SessionId)
	if err != nil {
		log.Printf("Warning: Failed to unregister stream from signaling service: %v", err)
		// Continue with cleanup regardless
	}

	// Close frame queues
	close(stream.VideoFrameQueue)
	close(stream.AudioFrameQueue)

	// Remove stream
	delete(s.streams, req.StreamId)

	log.Printf("Destroyed stream %s", req.StreamId)

	return &webrtcPb.DestroyStreamResponse{
		Success: true,
	}, nil
}

// AddVideoFrame adds a video frame to a stream
func (s *Service) AddVideoFrame(ctx context.Context, req *webrtcPb.AddVideoFrameRequest) (*webrtcPb.AddVideoFrameResponse, error) {
	// Get stream
	s.mu.RLock()
	stream, exists := s.streams[req.StreamId]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Check if frame queue is full
	if len(stream.VideoFrameQueue) >= stream.MaxQueueSize {
		if stream.DropWhenFull {
			// Drop frame if queue is full
			s.metrics.FrameDropped(req.StreamId, "", "video", "queue_full")
			return &webrtcPb.AddVideoFrameResponse{
				Success:    false,
				Dropped:    true,
				QueueDepth: int32(len(stream.VideoFrameQueue)),
			}, nil
		}
		// Block until space is available or context is canceled
		select {
		case stream.VideoFrameQueue <- req.FrameData:
			break
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		// Add frame to queue
		select {
		case stream.VideoFrameQueue <- req.FrameData:
			break
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	s.metrics.FrameReceived(req.StreamId, "video", req.IsKeyFrame, len(req.FrameData))

	return &webrtcPb.AddVideoFrameResponse{
		Success:    true,
		Dropped:    false,
		QueueDepth: int32(len(stream.VideoFrameQueue)),
	}, nil
}

// AddAudioFrame adds an audio frame to a stream
func (s *Service) AddAudioFrame(ctx context.Context, req *webrtcPb.AddAudioFrameRequest) (*webrtcPb.AddAudioFrameResponse, error) {
	// Get stream
	s.mu.RLock()
	stream, exists := s.streams[req.StreamId]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Check if frame queue is full
	if len(stream.AudioFrameQueue) >= stream.MaxQueueSize {
		if stream.DropWhenFull {
			// Drop frame if queue is full
			s.metrics.FrameDropped(req.StreamId, "", "audio", "queue_full")
			return &webrtcPb.AddAudioFrameResponse{
				Success:    false,
				Dropped:    true,
				QueueDepth: int32(len(stream.AudioFrameQueue)),
			}, nil
		}
		// Block until space is available or context is canceled
		select {
		case stream.AudioFrameQueue <- req.FrameData:
			break
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		// Add frame to queue
		select {
		case stream.AudioFrameQueue <- req.FrameData:
			break
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	s.metrics.FrameReceived(req.StreamId, "audio", false, len(req.FrameData))

	return &webrtcPb.AddAudioFrameResponse{
		Success:    true,
		Dropped:    false,
		QueueDepth: int32(len(stream.AudioFrameQueue)),
	}, nil
}

// GetStreamStats gets stats for a stream
func (s *Service) GetStreamStats(ctx context.Context, req *webrtcPb.GetStreamStatsRequest) (*webrtcPb.GetStreamStatsResponse, error) {
	// Get stream
	s.mu.RLock()
	stream, exists := s.streams[req.StreamId]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream not found: %s", req.StreamId)
	}

	// Count active peer connections
	stream.mu.RLock()
	activeConnections := len(stream.PeerConnections)
	stream.mu.RUnlock()

	return &webrtcPb.GetStreamStatsResponse{
		StreamId:          req.StreamId,
		ActiveConnections: int32(activeConnections),
		VideoQueueDepth:   int32(len(stream.VideoFrameQueue)),
		AudioQueueDepth:   int32(len(stream.AudioFrameQueue)),
	}, nil
}

// processVideoFrames processes video frames for a stream
func (s *Service) processVideoFrames(stream *Stream) {
	for frameData := range stream.VideoFrameQueue {
		startTime := time.Now()

		// TODO: Process video frame if needed (e.g., transform, enhance, etc.)

		// Write frame to track
		if err := stream.VideoTrack.WriteSample(media.Sample{
			Data:     frameData,
			Duration: time.Second / time.Duration(s.config.WebRTC.VideoFrameRate),
		}); err != nil {
			if !errors.Is(err, io.ErrClosedPipe) {
				log.Printf("Error writing video sample: %v", err)
				s.metrics.ErrorOccurred(stream.ID, "", "video_write_error")
			}
		}

		processingTime := time.Since(startTime)
		s.metrics.FrameProcessed(stream.ID, "video", processingTime)
	}
}

// processAudioFrames processes audio frames for a stream
func (s *Service) processAudioFrames(stream *Stream) {
	for frameData := range stream.AudioFrameQueue {
		startTime := time.Now()

		// TODO: Process audio frame if needed (e.g., normalize, filter, etc.)

		// Write frame to track
		if err := stream.AudioTrack.WriteSample(media.Sample{
			Data:     frameData,
			Duration: s.config.WebRTC.OpusFrameDuration,
		}); err != nil {
			if !errors.Is(err, io.ErrClosedPipe) {
				log.Printf("Error writing audio sample: %v", err)
				s.metrics.ErrorOccurred(stream.ID, "", "audio_write_error")
			}
		}

		processingTime := time.Since(startTime)
		s.metrics.FrameProcessed(stream.ID, "audio", processingTime)
	}
}
