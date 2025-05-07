package transport

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
	framepb "github.com/Harshitk-cp/streamhive/libs/proto/frame"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// FrameReceiver receives frames from the frame splitter
type FrameReceiver struct {
	address        string
	webrtcService  *service.WebRTCService
	conn           *grpc.ClientConn
	client         framepb.FrameSplitterServiceClient
	streamContexts map[string]context.CancelFunc
	stopChan       chan struct{}
}

// NewFrameReceiver creates a new frame receiver
func NewFrameReceiver(address string, webrtcService *service.WebRTCService) (*FrameReceiver, error) {
	return &FrameReceiver{
		address:        address,
		webrtcService:  webrtcService,
		streamContexts: make(map[string]context.CancelFunc),
		stopChan:       make(chan struct{}),
	}, nil
}

// Start starts the frame receiver
func (r *FrameReceiver) Start(ctx context.Context) error {
	// Connect to frame splitter service
	var err error
	r.conn, err = grpc.Dial(
		r.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
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

	// Cancel all stream contexts
	for streamID, cancel := range r.streamContexts {
		log.Printf("Cancelling stream context for %s", streamID)
		cancel()
		delete(r.streamContexts, streamID)
	}

	// Close connection
	if r.conn != nil {
		r.conn.Close()
	}

	log.Println("Frame receiver stopped")
}

// registerWithFrameSplitter registers with the frame splitter service
func (r *FrameReceiver) registerWithFrameSplitter(ctx context.Context) {
	// For simplicity, we'll just log the registration
	log.Printf("Registering with frame splitter at %s", r.address)

	// In a real implementation with proper proto definitions, this would call:
	// r.client.RegisterConsumer(ctx, &framepb.RegisterConsumerRequest{...})
}

// subscribeToAllStreams subscribes to all streams
func (r *FrameReceiver) subscribeToAllStreams(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// For now, just log the attempt
			log.Printf("Checking for available streams to subscribe to")

			// In a real implementation with proper proto definitions:
			// resp, err := r.client.ListStreams(ctx, &emptypb.Empty{})
			// if err != nil { log.Printf("Failed to list streams: %v", err); continue }
			// For each stream in resp.Streams...

		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		}
	}
}

// subscribeToStream subscribes to a stream
func (r *FrameReceiver) subscribeToStream(parentCtx context.Context, streamID string) {
	log.Printf("WebRTC-out: Subscribing to stream %s", streamID)
	// Create a cancellable context for this stream
	ctx, cancel := context.WithCancel(parentCtx)
	r.streamContexts[streamID] = cancel

	log.Printf("Subscribing to stream: %s", streamID)

	// In a real implementation with proper proto definitions:
	// req := &framepb.SubscribeToStreamRequest{
	//     StreamId:    streamID,
	//     ConsumerId:  "webrtc-out",
	//     IncludeVideo: true,
	//     IncludeAudio: true,
	//     IncludeMetadata: true,
	// }
	// stream, err := r.client.SubscribeToStream(ctx, req)

	// For demonstration, create dummy data
	go func() {
		// Simulate receiving frames
		ticker := time.NewTicker(33 * time.Millisecond) // ~30fps
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Create dummy frame for testing
				dummyFrame := &model.Frame{
					StreamID:   streamID,
					FrameID:    fmt.Sprintf("frm_%d", time.Now().UnixNano()),
					Type:       model.FrameTypeVideo,
					Data:       []byte("dummy video data"),
					Timestamp:  time.Now(),
					IsKeyFrame: false,
					Sequence:   0,
					Metadata:   map[string]string{"width": "1280", "height": "720"},
				}

				// Push frame to WebRTC service
				if err := r.webrtcService.PushFrame(dummyFrame); err != nil {
					log.Printf("Failed to push frame to WebRTC service: %v", err)
				}

			case <-ctx.Done():
				log.Printf("Stream context cancelled for %s", streamID)
				return
			}
		}
	}()
}
