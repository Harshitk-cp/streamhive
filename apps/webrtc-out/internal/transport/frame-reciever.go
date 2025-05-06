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
	"google.golang.org/protobuf/types/known/emptypb"
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
	// Register as consumer
	req := &framepb.RegisterConsumerRequest{
		ConsumerId:   "webrtc-out",
		ConsumerType: "webrtc",
		Capabilities: []string{"h264", "opus", "aac_to_opus"},
	}

	resp, err := r.client.RegisterConsumer(ctx, req)
	if err != nil {
		log.Printf("Failed to register with frame splitter: %v", err)
		return
	}

	log.Printf("Registered with frame splitter: %s", resp.ConsumerId)

	// Start stream subscription
	go r.subscribeToAllStreams(ctx)
}

// subscribeToAllStreams subscribes to all streams
func (r *FrameReceiver) subscribeToAllStreams(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get available streams
			resp, err := r.client.ListStreams(ctx, &emptypb.Empty{})
			if err != nil {
				log.Printf("Failed to list streams: %v", err)
				continue
			}

			// Subscribe to new streams
			for _, streamInfo := range resp.Streams {
				if _, exists := r.streamContexts[streamInfo.StreamId]; !exists {
					go r.subscribeToStream(ctx, streamInfo.StreamId)
				}
			}

		case <-ctx.Done():
			return

		case <-r.stopChan:
			return
		}
	}
}

// subscribeToStream subscribes to a stream
func (r *FrameReceiver) subscribeToStream(parentCtx context.Context, streamID string) {
	// Create a cancellable context for this stream
	ctx, cancel := context.WithCancel(parentCtx)
	r.streamContexts[streamID] = cancel

	// Subscribe to stream
	req := &framepb.SubscribeToStreamRequest{
		StreamId:        streamID,
		ConsumerId:      "webrtc-out",
		IncludeVideo:    true,
		IncludeAudio:    true,
		IncludeMetadata: true,
	}

	log.Printf("Subscribing to stream: %s", streamID)
	stream, err := r.client.SubscribeToStream(ctx, req)
	if err != nil {
		log.Printf("Failed to subscribe to stream %s: %v", streamID, err)
		delete(r.streamContexts, streamID)
		return
	}

	// Create a stream in the WebRTC service
	_, err = r.webrtcService.CreateStream(streamID)
	if err != nil {
		log.Printf("Failed to create WebRTC stream %s: %v", streamID, err)
		delete(r.streamContexts, streamID)
		return
	}

	// Process frames
	for {
		select {
		case <-ctx.Done():
			log.Printf("Stream context cancelled for %s", streamID)
			return

		case <-r.stopChan:
			log.Printf("Frame receiver stopping, cancelling stream %s", streamID)
			return

		default:
			// Receive frame
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Error receiving frame for stream %s: %v", streamID, err)
				// Remove stream subscription
				delete(r.streamContexts, streamID)
				return
			}

			// Get frame type
			var frameType model.FrameType
			switch resp.Type {
			case framepb.FrameType_VIDEO:
				frameType = model.FrameTypeVideo
			case framepb.FrameType_AUDIO:
				frameType = model.FrameTypeAudio
			case framepb.FrameType_METADATA:
				frameType = model.FrameTypeMetadata
			default:
				continue
			}

			// Convert timestamp
			timestamp := time.Now()
			if resp.Timestamp != nil {
				timestamp = resp.Timestamp.AsTime()
			}

			// Create frame
			frame := &model.Frame{
				StreamID:   resp.StreamId,
				FrameID:    resp.FrameId,
				Type:       frameType,
				Data:       resp.Data,
				Timestamp:  timestamp,
				IsKeyFrame: resp.IsKeyFrame,
				Sequence:   resp.Sequence,
				Metadata:   resp.Metadata,
			}

			// Push frame to WebRTC service
			if err := r.webrtcService.PushFrame(frame); err != nil {
				log.Printf("Failed to push frame to WebRTC service: %v", err)
			}
		}
	}
}
