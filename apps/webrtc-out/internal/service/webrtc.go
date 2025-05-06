package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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
	externalAddress   string // The external network address for this service
}

// Stream represents a WebRTC stream
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

// PeerConnection represents a WebRTC peer connection
type PeerConnection struct {
	ID             string
	UserID         string
	Connection     *webrtc.PeerConnection
	VideoSender    *webrtc.RTPSender
	AudioSender    *webrtc.RTPSender
	ConnectionTime time.Time
	LastActivity   time.Time
}

// New creates a new WebRTC service
func New(cfg *config.Config, m metrics.Collector) (*Service, error) {
	// Validate signaling service address
	if cfg.WebRTC.SignalingService == "" {
		return nil, fmt.Errorf("signaling service address is not configured")
	}

	// Get the external address from environment variable or use default
	externalAddress := os.Getenv("EXTERNAL_ADDRESS")
	if externalAddress == "" {
		// If not set in env, construct from hostname and gRPC address setting
		hostname, err := os.Hostname()
		if err != nil {
			log.Printf("Warning: Unable to get hostname: %v, using 'localhost' as fallback", err)
			hostname = "localhost"
		}

		// Remove the leading ":" from gRPC address if present
		grpcPortStr := cfg.GRPC.Address
		if len(grpcPortStr) > 0 && grpcPortStr[0] == ':' {
			grpcPortStr = grpcPortStr[1:]
		}

		// Construct address in hostname:port format
		externalAddress = fmt.Sprintf("%s:%s", hostname, grpcPortStr)
	}

	log.Printf("External address for registration: %s", externalAddress)

	log.Printf("Connecting to signaling service at %s", cfg.WebRTC.SignalingService)

	// Connect to the signaling service with retry logic
	var conn *grpc.ClientConn
	var err error

	// Add retry logic for connecting to signaling service
	maxRetries := 5
	backoffDuration := 2 * time.Second

	for retry := 0; retry < maxRetries; retry++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		// Connect with blocking dial to wait for service to be available
		conn, err = grpc.DialContext(
			ctx,
			cfg.WebRTC.SignalingService,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)

		cancel()

		if err == nil {
			break
		}

		log.Printf("Failed to connect to signaling service (attempt %d/%d): %v. Retrying in %v...",
			retry+1, maxRetries, err, backoffDuration)

		// If this is the last attempt, fail
		if retry == maxRetries-1 {
			return nil, fmt.Errorf("failed to connect to signaling service after %d attempts: %w", maxRetries, err)
		}

		// Wait before retrying
		time.Sleep(backoffDuration)

		// Exponential backoff
		backoffDuration *= 2
	}

	log.Printf("Successfully connected to signaling service")

	service := &Service{
		config:            cfg,
		metrics:           m,
		streams:           make(map[string]*Stream),
		signalingConn:     conn,
		signalingClient:   webrtcPb.NewSignalingServiceClient(conn),
		stopChan:          make(chan struct{}),
		heartbeatInterval: 30 * time.Second,
		externalAddress:   externalAddress,
	}

	// Register with signaling service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Register with retry logic
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
	// Register as a WebRTC server node with retry logic
	maxRetries := 3
	var lastError error

	for i := 0; i < maxRetries; i++ {
		registerReq := &webrtcPb.RegisterWebRTCNodeRequest{
			NodeId:       s.config.Service.NodeID,
			Address:      s.externalAddress, // Use the external addressable hostname:port
			Capabilities: s.config.WebRTC.CodecPreferences,
			MaxStreams:   int32(100), // Default max streams
		}

		log.Printf("Registering WebRTC node with ID: %s, Address: %s",
			registerReq.NodeId, registerReq.Address)

		resp, err := s.signalingClient.RegisterWebRTCNode(ctx, registerReq)
		if err == nil && resp != nil && resp.Success {
			log.Printf("Successfully registered with signaling service as node %s", s.config.Service.NodeID)

			// Start status update heartbeat
			go s.signalHeartbeat(ctx)
			return nil
		}

		lastError = err
		log.Printf("Failed to register with signaling service (attempt %d/%d): %v",
			i+1, maxRetries, err)

		// Wait before retrying
		time.Sleep(time.Duration(2<<i) * time.Second)
	}

	return fmt.Errorf("failed to register with signaling service after %d attempts: %w",
		maxRetries, lastError)
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
			beatCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
