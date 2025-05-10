package transport

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
	framepb "github.com/Harshitk-cp/streamhive/libs/proto/frame"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// FrameReceiver receives frames from the frame splitter
type FrameReceiver struct {
	address        string
	webrtcService  *service.WebRTCService
	conn           *grpc.ClientConn
	client         framepb.FrameSplitterServiceClient
	streamContexts map[string]context.CancelFunc
	streamsMutex   sync.RWMutex // Add mutex for protecting streamContexts map
	stopChan       chan struct{}
	activeStreams  map[string]bool // Track active streams
	streamsMu      sync.Mutex      // Mutex for activeStreams
}

// NewFrameReceiver creates a new frame receiver
func NewFrameReceiver(address string, webrtcService *service.WebRTCService) (*FrameReceiver, error) {
	return &FrameReceiver{
		address:        address,
		webrtcService:  webrtcService,
		streamContexts: make(map[string]context.CancelFunc),
		stopChan:       make(chan struct{}),
		activeStreams:  make(map[string]bool),
	}, nil
}

// Start starts the frame receiver
func (r *FrameReceiver) Start(ctx context.Context) error {
	// Connect to frame splitter service with improved connection parameters
	var err error
	r.conn, err = grpc.Dial(
		r.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024), // 50MB
			grpc.MaxCallSendMsgSize(50*1024*1024), // 50MB
		),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to frame splitter: %w", err)
	}

	// Create client
	r.client = framepb.NewFrameSplitterServiceClient(r.conn)

	// Register with frame splitter as a consumer
	go r.registerWithFrameSplitter(ctx)

	log.Println("Frame receiver started")
	return nil
}

// Stop stops the frame receiver
func (r *FrameReceiver) Stop() {
	close(r.stopChan)

	// Cancel all stream contexts with mutex protection
	r.streamsMutex.Lock()
	for streamID, cancel := range r.streamContexts {
		log.Printf("Cancelling stream context for %s", streamID)
		cancel()
		delete(r.streamContexts, streamID)
	}
	r.streamsMutex.Unlock()

	// Close connection
	if r.conn != nil {
		r.conn.Close()
	}

	log.Println("Frame receiver stopped")
}

// registerWithFrameSplitter registers with the frame splitter service
func (r *FrameReceiver) registerWithFrameSplitter(ctx context.Context) {
	// If RegisterConsumer isn't available or doesn't match what we need, we'll implement a simulation for now
	log.Printf("Registering with frame splitter at %s", r.address)

	// Simulate registration and subscribing to all streams
	go r.subscribeToAllStreams(ctx)
}

// subscribeToAllStreams subscribes to all streams
func (r *FrameReceiver) subscribeToAllStreams(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	testStreams := []string{}

	for {
		select {
		case <-ticker.C:
			// Check for each test stream if we need to start it
			for _, streamID := range testStreams {
				r.streamsMu.Lock()
				active := r.activeStreams[streamID]
				r.streamsMu.Unlock()

				if !active {
					// Start this stream if not already active
					r.streamsMutex.RLock()
					_, exists := r.streamContexts[streamID]
					r.streamsMutex.RUnlock()

					if !exists {
						go r.simulateStream(ctx, streamID)
					}
				}
			}

		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		}
	}
}

// simulateStream simulates receiving frames for a stream
func (r *FrameReceiver) simulateStream(ctx context.Context, streamID string) {
	// Add this stream to active streams map with mutex protection
	r.streamsMu.Lock()
	r.activeStreams[streamID] = true
	r.streamsMu.Unlock()

	// Create a cancellable context for this stream with mutex protection
	streamCtx, cancel := context.WithCancel(ctx)

	r.streamsMutex.Lock()
	r.streamContexts[streamID] = cancel
	r.streamsMutex.Unlock()

	log.Printf("WebRTC-out: Simulating stream %s", streamID)

	// Simulate receiving frames
	go func() {
		// Make sure to remove from active streams when done
		defer func() {
			r.streamsMu.Lock()
			delete(r.activeStreams, streamID)
			r.streamsMu.Unlock()

			r.streamsMutex.Lock()
			delete(r.streamContexts, streamID)
			r.streamsMutex.Unlock()
		}()

		// Create dummy frames at 30fps (33ms intervals)
		ticker := time.NewTicker(33 * time.Millisecond)
		defer ticker.Stop()

		frameCount := int64(0)
		keyFrameInterval := 30 // Every 30 frames (approx. 1 second)

		for {
			select {
			case <-ticker.C:
				frameCount++
				isKeyFrame := frameCount%int64(keyFrameInterval) == 0

				// Create video frame
				videoFrame := &model.Frame{
					StreamID:   streamID,
					FrameID:    fmt.Sprintf("vid_%d", frameCount),
					Type:       model.FrameTypeVideo,
					Data:       generateDummyFrameData(1024, isKeyFrame), // 1KB dummy data
					Timestamp:  time.Now(),
					IsKeyFrame: isKeyFrame,
					Sequence:   frameCount,
					Metadata: map[string]string{
						"width":  "1280",
						"height": "720",
						"codec":  "h264",
					},
				}

				// Push frame to WebRTC service
				if err := r.webrtcService.PushFrame(videoFrame); err != nil {
					log.Printf("Failed to push video frame: %v", err)
				}

				// Every 10 frames, send an audio frame
				if frameCount%10 == 0 {
					audioFrame := &model.Frame{
						StreamID:  streamID,
						FrameID:   fmt.Sprintf("aud_%d", frameCount),
						Type:      model.FrameTypeAudio,
						Data:      generateDummyFrameData(256, false), // 256B dummy data
						Timestamp: time.Now(),
						Sequence:  frameCount,
						Metadata: map[string]string{
							"codec":      "opus",
							"sampleRate": "48000",
							"channels":   "2",
						},
					}

					if err := r.webrtcService.PushFrame(audioFrame); err != nil {
						log.Printf("Failed to push audio frame: %v", err)
					}
				}

			case <-streamCtx.Done():
				log.Printf("Stream context cancelled for %s", streamID)
				return
			}
		}
	}()
}

// generateDummyFrameData generates dummy frame data
func generateDummyFrameData(size int, isKeyFrame bool) []byte {
	data := make([]byte, size)

	// Fill with some pattern
	for i := 0; i < size; i++ {
		if isKeyFrame {
			data[i] = byte(i % 255) // Different pattern for key frames
		} else {
			data[i] = byte(255 - (i % 255))
		}
	}

	return data
}
