package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/util"
	signalpb "github.com/Harshitk-cp/streamhive/libs/proto/signaling"
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
		// Skip TURN servers with empty credentials
		if strings.HasPrefix(server.URLs[0], "turn:") {
			if server.Username == "" || server.Credential == "" {
				log.Printf("Skipping TURN server %s due to missing credentials", server.URLs[0])
				continue
			}
		}

		// Add properly configured server
		iceServer := webrtc.ICEServer{
			URLs:       server.URLs,
			Username:   server.Username,
			Credential: server.Credential,
		}
		iceServers = append(iceServers, iceServer)
	}

	// If no servers were added, add a fallback STUN server
	if len(iceServers) == 0 {
		log.Println("No valid ICE servers configured, using fallback STUN servers")
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"},
		})
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

	// Create setting engine with DTLS role
	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetAnsweringDTLSRole(webrtc.DTLSRoleClient) // Set as active for all connections

	// Create WebRTC API with settings engine
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	// Create service
	service := &WebRTCService{
		cfg:          cfg,
		streams:      make(map[string]*model.Stream),
		viewers:      make(map[string]*model.Viewer),
		frameQueues:  make(map[string]chan *model.Frame),
		stopChan:     make(chan struct{}),
		webrtcConfig: webrtcConfig,
		mediaEngine:  mediaEngine,
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
	if existingStream, exists := s.streams[streamID]; exists {
		log.Printf("Stream %s already exists, returning existing stream", streamID)
		existingStream.LastActivity = time.Now()
		return existingStream, nil
	}

	// Create stream
	stream := &model.Stream{
		ID:                   streamID,
		Status:               model.StreamStatusIdle,
		CreatedAt:            time.Now(),
		LastActivity:         time.Now(),
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
	if _, exists := s.frameQueues[streamID]; !exists {
		s.frameQueues[streamID] = make(chan *model.Frame, 1000) // Buffer up to 1000 frames
		// Start frame processing goroutine
		go s.processFrames(streamID)
	}
	s.queuesMutex.Unlock()

	log.Printf("Created stream: %s", streamID)
	return stream, nil
}

// GetStream gets a stream by ID
func (s *WebRTCService) GetStream(streamID string) (*model.Stream, error) {
	s.streamsMutex.RLock()
	stream, exists := s.streams[streamID]
	s.streamsMutex.RUnlock()

	if !exists {
		// Return error to trigger stream creation
		return nil, fmt.Errorf("stream not found: %s", streamID)
	}

	// Update last activity time
	stream.LastActivity = time.Now()

	return stream, nil
}

// HandleOffer handles an SDP offer from a viewer
func (s *WebRTCService) HandleOffer(streamID, viewerID string, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	log.Printf("Handling WebRTC offer for stream %s from viewer %s", streamID, viewerID)

	// Check if the viewer already exists - if so, clean up first
	s.viewersMutex.RLock()
	existingViewer, viewerExists := s.viewers[viewerID]
	s.viewersMutex.RUnlock()

	if viewerExists {
		log.Printf("Viewer %s already exists with name %s, cleaning up previous connection", viewerID, existingViewer.ID)
		s.removeViewer(viewerID)
		// Short sleep to allow resources to be freed
		time.Sleep(100 * time.Millisecond)
	}

	// Debug log the full SDP to help diagnose issues
	log.Printf("Full SDP offer:\n%s", offer.SDP)

	// Get or create stream
	stream, err := s.GetStream(streamID)
	if err != nil {
		// Stream doesn't exist, create it
		stream, err = s.CreateStream(streamID)
		if err != nil {
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
	}

	// Create peer connection with explicit configuration
	peerConnection, err := s.api.NewPeerConnection(s.webrtcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Create audio track with more specific codec capabilities
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		},
		"audio", "streamhive-audio",
	)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	// Create video track with more specific codec capabilities
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: 90000,
		},
		"video", "streamhive-video",
	)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create video track: %w", err)
	}

	// Add tracks to peer connection with explicit error handling
	audioSender, err := peerConnection.AddTrack(audioTrack)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add audio track: %w", err)
	}

	// Handle RTCP packets for audio
	go func() {
		for {
			rtcpPackets, _, rtcpErr := audioSender.ReadRTCP()
			if rtcpErr != nil {
				if rtcpErr != io.EOF {
					log.Printf("Error reading audio RTCP: %v", rtcpErr)
				}
				return
			}

			for _, packet := range rtcpPackets {
				switch packet.(type) {
				case *rtcp.ReceiverReport:
					log.Printf("Received audio RR from viewer %s", viewerID)
				}
			}
		}
	}()

	// Add video track
	videoSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add video track: %w", err)
	}

	// Listen for RTCP packets to handle PLI/NACK
	go func() {
		for {
			rtcpPackets, _, rtcpErr := videoSender.ReadRTCP()
			if rtcpErr != nil {
				if rtcpErr != io.EOF {
					log.Printf("Error reading video RTCP: %v", rtcpErr)
				}
				return
			}

			for _, packet := range rtcpPackets {
				switch packet := packet.(type) {
				case *rtcp.PictureLossIndication:
					log.Printf("Received PLI from viewer %s", viewerID)
				case *rtcp.FullIntraRequest:
					log.Printf("Received FIR from viewer %s", viewerID)
				case *rtcp.ReceiverEstimatedMaximumBitrate:
					log.Printf("Received REMB from viewer %s: %d bps", viewerID, int(packet.Bitrate))
				}
			}
		}
	}()

	// Create data channel for control messages with specific configuration
	dataChannelConfig := &webrtc.DataChannelInit{
		Ordered:        util.BoolPtr(true),
		MaxRetransmits: util.Uint16Ptr(3),
	}

	dataChannel, err := peerConnection.CreateDataChannel("control", dataChannelConfig)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create data channel: %w", err)
	}

	// Set up data channel handlers
	dataChannel.OnOpen(func() {
		log.Printf("Data channel opened for viewer %s", viewerID)

		// Send initial connection message
		msg := map[string]interface{}{
			"type": "connected",
			"time": time.Now().Unix(),
		}
		msgBytes, err := json.Marshal(msg)
		if err == nil {
			err = dataChannel.Send(msgBytes)
			if err != nil {
				log.Printf("Failed to send initial message to viewer %s: %v", viewerID, err)
			}
		}
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received data channel message from viewer %s: %s", viewerID, string(msg.Data))
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
		Stats:          model.ViewerStats{},
	}

	// Set event handlers for peer connection
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE connection state changed to %s for viewer %s", state.String(), viewerID)

		s.viewersMutex.Lock()
		defer s.viewersMutex.Unlock()

		// Update viewer
		v, exists := s.viewers[viewerID]
		if !exists {
			return
		}

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

			// Send metadata about the stream to the viewer
			if dataChannel.ReadyState() == webrtc.DataChannelStateOpen {
				// Create stream info message
				streamInfo := map[string]interface{}{
					"type":        "stream_info",
					"stream_id":   streamID,
					"resolution":  fmt.Sprintf("%dx%d", stream.Width, stream.Height),
					"frame_rate":  stream.FrameRate,
					"video_codec": string(stream.VideoCodec),
					"audio_codec": string(stream.AudioCodec),
				}

				// Send to data channel
				infoBytes, err := json.Marshal(streamInfo)
				if err == nil {
					err = dataChannel.Send(infoBytes)
					if err != nil {
						log.Printf("Failed to send stream info to viewer %s: %v", viewerID, err)
					}
				}
			}

		case webrtc.ICEConnectionStateDisconnected,
			webrtc.ICEConnectionStateFailed,
			webrtc.ICEConnectionStateClosed:
			v.Status = model.StreamStatusClosed
			s.removeViewer(viewerID)
		}
	})

	// Set the remote SessionDescription - log right before to help debugging
	log.Printf("Setting remote description for viewer %s", viewerID)

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		log.Printf("Error setting remote description: %v", err)
		peerConnection.Close()
		return nil, fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer - note: no VoiceActivityDetection field in AnswerOptions
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create answer: %w", err)
	}

	// Log SDP answer for debugging
	log.Printf("Created SDP answer for viewer %s, type=%s", viewerID, answer.Type.String())
	log.Printf("SDP answer preview: %s", answer.SDP[:util.Min(100, len(answer.SDP))])

	// Set local SessionDescription
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		peerConnection.Close()
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

	log.Printf("Successfully created peer connection for viewer %s on stream %s", viewerID, streamID)
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
	log.Printf("Received frame for stream %s, type=%v, size=%d bytes, keyframe=%v",
		frame.StreamID, frame.Type, len(frame.Data), frame.IsKeyFrame)

	s.queuesMutex.RLock()
	frameQueue, exists := s.frameQueues[frame.StreamID]
	s.queuesMutex.RUnlock()

	if !exists {
		log.Printf("Creating new stream for incoming frame: %s", frame.StreamID)
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
		log.Printf("Frame added to queue for stream %s", frame.StreamID)
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
		log.Printf("Stream %s is now active", frame.StreamID)
	}

	// Update resolution if available in metadata
	if width, exists := frame.Metadata["width"]; exists {
		if widthVal, err := strconv.Atoi(width); err == nil && widthVal > 0 {
			stream.Width = widthVal
		}
	}
	if height, exists := frame.Metadata["height"]; exists {
		if heightVal, err := strconv.Atoi(height); err == nil && heightVal > 0 {
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

	// Remove from viewers map
	delete(s.viewers, viewerID)
	s.viewersMutex.Unlock()

	// Get a reference to the peer connection before cleanup
	peerConnection := viewer.PeerConnection

	// Remove viewer from stream
	s.streamsMutex.Lock()
	stream, exists := s.streams[streamID]
	if exists {
		stream.Mutex.Lock()
		delete(stream.Viewers, viewerID)
		stream.CurrentViewers--
		stream.Mutex.Unlock()
	}
	s.streamsMutex.Unlock()

	// Close peer connection after removing references
	if peerConnection != nil {
		peerConnection.Close()
	}

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

	for {
		select {
		case frame, ok := <-frameQueue:
			if !ok {
				log.Printf("Frame queue closed for stream %s", streamID)
				return
			}

			// Process frame based on type
			switch frame.Type {
			case model.FrameTypeVideo:
				// Create video sample and broadcast to all viewers
				sample := &media.Sample{
					Data:     frame.Data,
					Duration: time.Millisecond * 33, // Approximate for 30fps
				}

				s.broadcastVideoSample(streamID, sample)
				// log.Printf("Broadcast video frame for stream %s, size=%d bytes", streamID, len(frame.Data))

			case model.FrameTypeAudio:
				// Transcode AAC to Opus if needed
				audioData := frame.Data

				// If codec is AAC, transcode to Opus
				if codec, exists := frame.Metadata["codec"]; exists && codec == "aac" {
					// In a real implementation, we would transcode AAC to Opus
					// For simplicity, we'll just pass through the audio data
					log.Printf("AAC audio detected, would be transcoded to Opus in production")
				}

				// Create audio sample and broadcast to all viewers
				sample := &media.Sample{
					Data:     audioData,
					Duration: time.Millisecond * 20, // 20ms frames for Opus
				}

				s.broadcastAudioSample(streamID, sample)
				log.Printf("Broadcast audio frame for stream %s, size=%d bytes", streamID, len(frame.Data))

			case model.FrameTypeMetadata:
				// Process metadata
				var metadata map[string]interface{}
				if err := json.Unmarshal(frame.Data, &metadata); err == nil {
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
					track, ok := sender.Track().(*webrtc.TrackLocalStaticSample)
					if !ok {
						continue
					}

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
					track, ok := sender.Track().(*webrtc.TrackLocalStaticSample)
					if !ok {
						continue
					}

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
	log.Printf("Notifying stream ended: %s", streamID)

	// Connect to signaling service
	conn, err := grpc.Dial(s.cfg.Signaling.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to signaling service: %v", err)
		return
	}
	defer conn.Close()

	// Create client and call NotifyStreamEnded
	client := signalpb.NewSignalingServiceClient(conn)
	_, err = client.NotifyStreamEnded(context.Background(), &signalpb.NotifyStreamEndedRequest{
		StreamId: streamID,
		Reason:   "Stream ended or timed out",
	})

	if err != nil {
		log.Printf("Failed to notify signaling service about stream end: %v", err)
	}
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
