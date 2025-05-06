package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"k
	"strconv"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/model"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// WebRTCService handles WebRTC streaming
type WebRTCService struct {
	cfg           *config.Config
	streams       map[string]*model.Stream
	streamsMutex  sync.RWMutex
	viewers       map[string]*model.Viewer
	viewersMutex  sync.RWMutex
	frameQueues   map[string]chan *model.Frame
	queuesMutex   sync.RWMutex
	stopChan      chan struct{}
	cleanupTicker *time.Ticker
	webrtcConfig  webrtc.Configuration
	apiOptions    []func(*webrtc.API)
	mediaEngine   *webrtc.MediaEngine
	api           *webrtc.API
	opusParams    OpusParams
}

// OpusParams contains Opus codec parameters
type OpusParams struct {
	MinBitrate  int
	MaxBitrate  int
	Complexity  int
	SampleRate  int
	FrameLength int
}

// NewWebRTCService creates a new WebRTC service
func NewWebRTCService(cfg *config.Config) (*WebRTCService, error) {
	// Convert ICE server config
	iceServers := make([]webrtc.ICEServer, 0, len(cfg.WebRTC.ICEServers))
	for _, server := range cfg.WebRTC.ICEServers {
		iceServer := webrtc.ICEServer{
			URLs:       server.URLs,
			Username:   server.Username,
			Credential: server.Credential,
		}
		iceServers = append(iceServers, iceServer)
	}

	// Create WebRTC configuration
	webrtcConfig := webrtc.Configuration{
		ICEServers:   iceServers,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	}

	// Create media engine
	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("failed to register default codecs: %w", err)
	}

	// Set up API options
	apiOptions := []func(*webrtc.API){
		func(api *webrtc.API) {
			api.MediaEngine().RegisterDefaultCodecs()
		},
	}

	// Create WebRTC API
	api := webrtc.NewAPI(apiOptions...)

	// Create service
	service := &WebRTCService{
		cfg:          cfg,
		streams:      make(map[string]*model.Stream),
		viewers:      make(map[string]*model.Viewer),
		frameQueues:  make(map[string]chan *model.Frame),
		stopChan:     make(chan struct{}),
		webrtcConfig: webrtcConfig,
		mediaEngine:  mediaEngine,
		apiOptions:   apiOptions,
		api:          api,
		opusParams: OpusParams{
			MinBitrate:  cfg.WebRTC.OpusMinBitrate,
			MaxBitrate:  cfg.WebRTC.OpusMaxBitrate,
			Complexity:  cfg.WebRTC.OpusComplexity,
			SampleRate:  48000, // Default for WebRTC
			FrameLength: 20,    // Default 20ms
		},
	}

	return service, nil
}

// Start starts the WebRTC service
func (s *WebRTCService) Start(ctx context.Context) {
	log.Println("Starting WebRTC service")

	// Start cleanup goroutine
	s.cleanupTicker = time.NewTicker(5 * time.Minute)
	go s.cleanupInactiveStreams(ctx)
}

// Stop stops the WebRTC service
func (s *WebRTCService) Stop() {
	log.Println("Stopping WebRTC service")

	// Signal all goroutines to stop
	close(s.stopChan)

	// Stop cleanup ticker
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	// Close all peer connections
	s.closeAllConnections()
}

// CreateStream creates a new stream
func (s *WebRTCService) CreateStream(streamID string) (*model.Stream, error) {
	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()

	// Check if stream already exists
	if _, exists := s.streams[streamID]; exists {
		return nil, fmt.Errorf("stream already exists: %s", streamID)
	}

	// Create stream
	stream := &model.Stream{
		ID:                   streamID,
		Status:               model.StreamStatusIdle,
		CreatedAt:            time.Now(),
		Viewers:              make(map[string]*model.Viewer),
		VideoCodec:           model.VideoCodecH264,
		AudioCodec:           model.AudioCodecOpus,
		KeyFrameInterval:     60, // Default to 2 seconds at 30fps
		MaxConcurrentViewers: 0,
		CurrentViewers:       0,
	}

	// Add to streams map
	s.streams[streamID] = stream

	// Create frame queue for this stream
	s.queuesMutex.Lock()
	s.frameQueues[streamID] = make(chan *model.Frame, 1000) // Buffer up to 1000 frames
	s.queuesMutex.Unlock()

	// Start frame processing goroutine
	go s.processFrames(streamID)

	log.Printf("Created stream: %s", streamID)
	return stream, nil
}

// GetStream gets a stream by ID
func (s *WebRTCService) GetStream(streamID string) (*model.Stream, error) {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()

	// Check if stream exists
	stream, exists := s.streams[streamID]
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", streamID)
	}

	return stream, nil
}

// HandleOffer handles an SDP offer from a viewer
func (s *WebRTCService) HandleOffer(streamID, viewerID string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	// Get or create stream
	stream, err := s.GetStream(streamID)
	if err != nil {
		// Stream doesn't exist, create it
		stream, err = s.CreateStream(streamID)
		if err != nil {
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
	}

	// Create peer connection
	peerConnection, err := s.api.NewPeerConnection(s.webrtcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Create audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio", "streamhive-audio",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	// Create video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video", "streamhive-video",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	// Add tracks to peer connection
	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to add audio track: %w", err)
	}

	videoSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to add video track: %w", err)
	}

	// Listen for RTCP packets to handle PLI/NACK
	go func() {
		for {
			rtcpPackets, err := videoSender.ReadRTCP()
			if err != nil {
				return
			}

			for _, packet := range rtcpPackets {
				switch packet := packet.(type) {
				case *rtcp.PictureLossIndication:
					// Request keyframe
					log.Printf("Received PLI from viewer %s", viewerID)
					// In a real implementation, we would send a keyframe request to the publisher
				case *rtcp.FullIntraRequest:
					log.Printf("Received FIR from viewer %s", viewerID)
					// In a real implementation, we would send a keyframe request to the publisher
				case *rtcp.ReceiverEstimatedMaximumBitrate:
					log.Printf("Received REMB from viewer %s: %d bps", viewerID, packet.Bitrate)
				}
			}
		}
	}()

	// Create data channel for control messages
	dataChannel, err := peerConnection.CreateDataChannel("control", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create data channel: %w", err)
	}

	// Set up data channel handlers
	dataChannel.OnOpen(func() {
		log.Printf("Data channel opened for viewer %s", viewerID)
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received data channel message from viewer %s: %s", viewerID, string(msg.Data))
		// In a real implementation, we would handle control messages
	})

	// Create viewer object
	viewer := &model.Viewer{
		ID:             viewerID,
		PeerConnection: peerConnection,
		DataChannel:    dataChannel,
		Status:         model.StreamStatusConnecting,
		StartTime:      time.Now(),
		LastActivity:   time.Now(),
		StreamID:       streamID,
	}

	// Set event handlers for peer connection
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE connection state changed to %s for viewer %s", state.String(), viewerID)

		s.viewersMutex.Lock()
		defer s.viewersMutex.Unlock()

		// Update viewer
		if v, exists := s.viewers[viewerID]; exists {
			v.LastActivity = time.Now()

			switch state {
			case webrtc.ICEConnectionStateConnected:
				v.Status = model.StreamStatusActive
				// Update stream stats
				s.streamsMutex.Lock()
				if str, exists := s.streams[streamID]; exists {
					str.CurrentViewers++
					if str.CurrentViewers > str.MaxConcurrentViewers {
						str.MaxConcurrentViewers = str.CurrentViewers
					}
					str.TotalViewers++
					str.LastActivity = time.Now()
				}
				s.streamsMutex.Unlock()

			case webrtc.ICEConnectionStateDisconnected,
				webrtc.ICEConnectionStateFailed,
				webrtc.ICEConnectionStateClosed:
				v.Status = model.StreamStatusClosed
				s.removeViewer(viewerID)
			}
		}
	})

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	// Set local SessionDescription
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return nil, fmt.Errorf("failed to set local description: %w", err)
	}

	// Register viewer
	s.viewersMutex.Lock()
	s.viewers[viewerID] = viewer
	s.viewersMutex.Unlock()

	// Add viewer to stream
	s.streamsMutex.Lock()
	stream.Viewers[viewerID] = viewer
	s.streamsMutex.Unlock()

	log.Printf("Created peer connection for viewer %s on stream %s", viewerID, streamID)
	return &answer, nil
}

// HandleICECandidate handles an ICE candidate from a viewer
func (s *WebRTCService) HandleICECandidate(viewerID string, candidate webrtc.ICECandidateInit) error {
	s.viewersMutex.RLock()
	defer s.viewersMutex.RUnlock()

	// Check if viewer exists
	viewer, exists := s.viewers[viewerID]
	if !exists {
		return fmt.Errorf("viewer not found: %s", viewerID)
	}

	// Add ICE candidate
	return viewer.PeerConnection.AddICECandidate(candidate)
}

// PushFrame pushes a frame to a stream
func (s *WebRTCService) PushFrame(frame *model.Frame) error {
	s.queuesMutex.RLock()
	frameQueue, exists := s.frameQueues[frame.StreamID]
	s.queuesMutex.RUnlock()

	if !exists {
		// Stream doesn't exist, create it
		_, err := s.CreateStream(frame.StreamID)
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}

		// Get frame queue
		s.queuesMutex.RLock()
		frameQueue = s.frameQueues[frame.StreamID]
		s.queuesMutex.RUnlock()
	}

	// Update stream information based on frame metadata
	s.updateStreamInfo(frame)

	// Send frame to queue (non-blocking)
	select {
	case frameQueue <- frame:
		// Frame added to queue
	default:
		// Queue is full, drop frame
		log.Printf("Frame queue is full for stream %s, dropping frame", frame.StreamID)
	}

	return nil
}

// updateStreamInfo updates stream information based on frame metadata
func (s *WebRTCService) updateStreamInfo(frame *model.Frame) {
	// Only update on video frames
	if frame.Type != model.FrameTypeVideo {
		return
	}

	s.streamsMutex.Lock()
	defer s.streamsMutex.Unlock()

	stream, exists := s.streams[frame.StreamID]
	if !exists {
		return
	}

	// Update total frames
	stream.TotalFrames++

	// Update last activity
	stream.LastActivity = time.Now()

	// If this is the first frame, update start time
	if stream.StartedAt.IsZero() {
		stream.StartedAt = time.Now()
		stream.Status = model.StreamStatusActive
	}

	// Update resolution if available in metadata
	// Update resolution if available in metadata
	if width, exists := frame.Metadata["width"]; exists {
		if widthVal, err := strconv.Atoi(width); err == nil {
			stream.Width = widthVal
		}
	}
	if height, exists := frame.Metadata["height"]; exists {
		if heightVal, err := strconv.Atoi(height); err == nil {
			stream.Height = heightVal
		}
	}

	// Update frame rate
	if frame.IsKeyFrame && stream.TotalFrames > 30 {
		elapsed := time.Since(stream.StartedAt).Seconds()
		if elapsed > 0 {
			stream.FrameRate = float64(stream.TotalFrames) / elapsed
		}
	}

	// Update codec information
	if codec, exists := frame.Metadata["codec"]; exists {
		if codec == "h264" {
			stream.VideoCodec = model.VideoCodecH264
		} else if codec == "vp8" {
			stream.VideoCodec = model.VideoCodecVP8
		} else if codec == "vp9" {
			stream.VideoCodec = model.VideoCodecVP9
		}
	}
}

// removeViewer removes a viewer
func (s *WebRTCService) removeViewer(viewerID string) {
	s.viewersMutex.Lock()
	viewer, exists := s.viewers[viewerID]
	if !exists {
		s.viewersMutex.Unlock()
		return
	}

	streamID := viewer.StreamID
	delete(s.viewers, viewerID)
	s.viewersMutex.Unlock()

	// Close peer connection
	if viewer.PeerConnection != nil {
		viewer.PeerConnection.Close()
	}

	// Remove viewer from stream
	s.streamsMutex.Lock()
	stream, exists := s.streams[streamID]
	if exists {
		delete(stream.Viewers, viewerID)
		stream.CurrentViewers--
	}
	s.streamsMutex.Unlock()

	log.Printf("Removed viewer %s from stream %s", viewerID, streamID)
}

// processFrames processes frames for a stream
func (s *WebRTCService) processFrames(streamID string) {
	log.Printf("Starting frame processing for stream %s", streamID)

	s.queuesMutex.RLock()
	frameQueue, exists := s.frameQueues[streamID]
	s.queuesMutex.RUnlock()

	if !exists {
		log.Printf("Frame queue not found for stream %s", streamID)
		return
	}

	// Create sample buffers for video and audio
	videoSamples := make(map[string]*media.Sample)
	audioSamples := make(map[string]*media.Sample)

	for {
		select {
		case frame := <-frameQueue:
			// Process frame based on type
			switch frame.Type {
			case model.FrameTypeVideo:
				// Create video sample and broadcast to all viewers
				sample := &media.Sample{
					Data:     frame.Data,
					Duration: time.Millisecond * 33, // Approximate for 30fps
				}

				s.broadcastVideoSample(streamID, sample)

			case model.FrameTypeAudio:
				// Transcode AAC to Opus if needed
				audioData := frame.Data

				// If codec is AAC, transcode to Opus
				if codec, exists := frame.Metadata["codec"]; exists && codec == "aac" {
					// In a real implementation, we would transcode AAC to Opus
					// For simplicity, we'll just pass through the audio data
					// This would be where you'd call the Opus encoder
				}

				// Create audio sample and broadcast to all viewers
				sample := &media.Sample{
					Data:     audioData,
					Duration: time.Millisecond * 20, // 20ms frames for Opus
				}

				s.broadcastAudioSample(streamID, sample)

			case model.FrameTypeMetadata:
				// Process metadata
				var metadata map[string]interface{}
				if err := json.Unmarshal(frame.Data, &metadata); err == nil {
					// In a real implementation, we would handle metadata
					log.Printf("Received metadata for stream %s: %v", streamID, metadata)
				}
			}

		case <-s.stopChan:
			log.Printf("Stopping frame processing for stream %s", streamID)
			return
		}
	}
}

// broadcastVideoSample broadcasts a video sample to all viewers of a stream
func (s *WebRTCService) broadcastVideoSample(streamID string, sample *media.Sample) {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()

	stream, exists := s.streams[streamID]
	if !exists {
		return
	}

	// Get all active viewers
	for viewerID, viewer := range stream.Viewers {
		if viewer.Status == model.StreamStatusActive {
			// Get peer connection
			pc := viewer.PeerConnection
			if pc == nil {
				continue
			}

			// Get video sender
			senders := pc.GetSenders()
			for _, sender := range senders {
				if sender.Track() != nil && sender.Track().Kind() == webrtc.RTPCodecTypeVideo {
					track := sender.Track().(*webrtc.TrackLocalStaticSample)
					if err := track.WriteSample(*sample); err != nil {
						log.Printf("Failed to write video sample to viewer %s: %v", viewerID, err)
					} else {
						viewer.TotalBytes += int64(len(sample.Data))
						viewer.Stats.VideoBytesSent += int64(len(sample.Data))
						viewer.Stats.VideoPacketsSent++
					}
					break
				}
			}
		}
	}
}

// broadcastAudioSample broadcasts an audio sample to all viewers of a stream
func (s *WebRTCService) broadcastAudioSample(streamID string, sample *media.Sample) {
	s.streamsMutex.RLock()
	defer s.streamsMutex.RUnlock()

	stream, exists := s.streams[streamID]
	if !exists {
		return
	}

	// Get all active viewers
	for viewerID, viewer := range stream.Viewers {
		if viewer.Status == model.StreamStatusActive {
			// Get peer connection
			pc := viewer.PeerConnection
			if pc == nil {
				continue
			}

			// Get audio sender
			senders := pc.GetSenders()
			for _, sender := range senders {
				if sender.Track() != nil && sender.Track().Kind() == webrtc.RTPCodecTypeAudio {
					track := sender.Track().(*webrtc.TrackLocalStaticSample)
					if err := track.WriteSample(*sample); err != nil {
						log.Printf("Failed to write audio sample to viewer %s: %v", viewerID, err)
					} else {
						viewer.TotalBytes += int64(len(sample.Data))
						viewer.Stats.AudioBytesSent += int64(len(sample.Data))
						viewer.Stats.AudioPacketsSent++
					}
					break
				}
			}
		}
	}
}

// cleanupInactiveStreams removes inactive streams
func (s *WebRTCService) cleanupInactiveStreams(ctx context.Context) {
	for {
		select {
		case <-s.cleanupTicker.C:
			// Get current time
			now := time.Now()

			// Find inactive streams
			var inactiveStreams []string
			s.streamsMutex.Lock()
			for streamID, stream := range s.streams {
				// Check if stream is inactive for more than 5 minutes
				if now.Sub(stream.LastActivity) > 5*time.Minute {
					inactiveStreams = append(inactiveStreams, streamID)
				}

				// Check if stream has exceeded maximum lifetime
				if !stream.StartedAt.IsZero() && now.Sub(stream.StartedAt) > s.cfg.WebRTC.MaxStreamLifetime {
					inactiveStreams = append(inactiveStreams, streamID)
				}
			}
			s.streamsMutex.Unlock()

			// Remove inactive streams
			for _, streamID := range inactiveStreams {
				log.Printf("Removing inactive stream: %s", streamID)
				s.RemoveStream(streamID)
			}

		case <-ctx.Done():
			return

		case <-s.stopChan:
			return
		}
	}
}

// RemoveStream removes a stream
func (s *WebRTCService) RemoveStream(streamID string) error {
	// Get stream viewers
	s.streamsMutex.Lock()
	stream, exists := s.streams[streamID]
	if !exists {
		s.streamsMutex.Unlock()
		return fmt.Errorf("stream not found: %s", streamID)
	}

	// Get viewer IDs
	viewerIDs := make([]string, 0, len(stream.Viewers))
	for viewerID := range stream.Viewers {
		viewerIDs = append(viewerIDs, viewerID)
	}

	// Remove stream
	delete(s.streams, streamID)
	s.streamsMutex.Unlock()

	// Remove stream from frame queues
	s.queuesMutex.Lock()
	if queue, exists := s.frameQueues[streamID]; exists {
		close(queue)
		delete(s.frameQueues, streamID)
	}
	s.queuesMutex.Unlock()

	// Remove viewers
	for _, viewerID := range viewerIDs {
		s.removeViewer(viewerID)
	}

	// Notify signaling server that stream has ended
	go s.notifyStreamEnded(streamID)

	log.Printf("Removed stream: %s", streamID)
	return nil
}

// notifyStreamEnded notifies the signaling server that a stream has ended
func (s *WebRTCService) notifyStreamEnded(streamID string) {
	// In a real implementation, we would call the signaling service via gRPC
	// to notify that the stream has ended
	log.Printf("Notifying stream ended: %s", streamID)

	// Connect to signaling service
	conn, err := grpc.Dial(s.cfg.Signaling.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to signaling service: %v", err)
		return
	}
	defer conn.Close()

	// TODO: Create client and call NotifyStreamEnded
}

// closeAllConnections closes all peer connections
func (s *WebRTCService) closeAllConnections() {
	// Get all viewer IDs
	var viewerIDs []string
	s.viewersMutex.RLock()
	for viewerID := range s.viewers {
		viewerIDs = append(viewerIDs, viewerID)
	}
	s.viewersMutex.RUnlock()

	// Remove all viewers
	for _, viewerID := range viewerIDs {
		s.removeViewer(viewerID)
	}

	// Close all frame queues
	s.queuesMutex.Lock()
	for streamID, queue := range s.frameQueues {
		close(queue)
		delete(s.frameQueues, streamID)
	}
	s.queuesMutex.Unlock()

	// Clear streams
	s.streamsMutex.Lock()
	s.streams = make(map[string]*model.Stream)
	s.streamsMutex.Unlock()
}
