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
	ShutdownCh      chan struct{}
	mu              sync.RWMutex
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
	config        *config.Config
	metrics       metrics.Collector
	streams       map[string]*Stream
	mu            sync.RWMutex
	signalingConn *grpc.ClientConn
}

// New creates a new WebRTC service
func New(cfg *config.Config, m metrics.Collector) (*Service, error) {
	// Connect to the signaling service
	conn, err := grpc.Dial(cfg.WebRTC.SignalingService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to signaling service: %w", err)
	}

	return &Service{
		config:        cfg,
		metrics:       m,
		streams:       make(map[string]*Stream),
		signalingConn: conn,
	}, nil
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
		ShutdownCh:      make(chan struct{}),
	}

	// Start frame processing goroutines
	go s.processVideoFrames(stream)
	go s.processAudioFrames(stream)

	// Register stream in signaling service
	client := webrtcPb.NewSignalingServiceClient(s.signalingConn)
	_, err = client.RegisterStream(ctx, &webrtcPb.RegisterStreamRequest{
		StreamId:  req.StreamId,
		SessionId: req.SessionId,
		NodeId:    s.config.Service.NodeID,
	})
	if err != nil {
		close(stream.ShutdownCh)
		return nil, fmt.Errorf("failed to register stream with signaling service: %w", err)
	}

	// Store stream
	s.streams[req.StreamId] = stream

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
	for _, peer := range stream.PeerConnections {
		if err := peer.Connection.Close(); err != nil {
			log.Printf("Error closing peer connection: %v", err)
		}
		s.metrics.PeerDisconnected(stream.ID, peer.UserID)
	}
	stream.mu.Unlock()

	// Signal shutdown to processing goroutines
	close(stream.ShutdownCh)

	// Unregister stream from signaling service
	client := webrtcPb.NewSignalingServiceClient(s.signalingConn)
	_, err := client.UnregisterStream(ctx, &webrtcPb.UnregisterStreamRequest{
		StreamId:  req.StreamId,
		SessionId: req.SessionId,
		NodeId:    s.config.Service.NodeID,
	})
	if err != nil {
		log.Printf("Error unregistering stream: %v", err)
	}

	// Remove stream
	delete(s.streams, req.StreamId)

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
	for {
		select {
		case <-stream.ShutdownCh:
			return
		case frameData := <-stream.VideoFrameQueue:
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
}

// processAudioFrames processes audio frames for a stream
func (s *Service) processAudioFrames(stream *Stream) {
	for {
		select {
		case <-stream.ShutdownCh:
			return
		case frameData := <-stream.AudioFrameQueue:
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
		log.Printf("ICE connection state changed: %s (SessionID: %s)", state.String(), sessionID)

		switch state {
		case webrtc.ICEConnectionStateConnected:
			s.metrics.PeerConnected(streamID, userID)
		case webrtc.ICEConnectionStateDisconnected, webrtc.ICEConnectionStateFailed, webrtc.ICEConnectionStateClosed:
			s.metrics.PeerDisconnected(streamID, userID)

			// Remove peer connection if disconnected
			if state == webrtc.ICEConnectionStateDisconnected || state == webrtc.ICEConnectionStateFailed {
				stream.mu.Lock()
				delete(stream.PeerConnections, userID)
				stream.mu.Unlock()
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

	return peerConnection, nil
}

// HandleOffer handles a SDP offer from a viewer
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
		peerConnection.Close()
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	// Set local description
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to set local description: %w", err)
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
	sdpMid := req.SdpMid // Create a copy to get a pointer to it
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

	return &webrtcPb.ICECandidateResponse{
		Success: true,
	}, nil
}
