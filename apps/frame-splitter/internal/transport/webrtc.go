package transport

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WebRTCClient represents a client for the WebRTC service
type WebRTCClient struct {
	*BaseClient
	client webrtc.WebRTCServiceClient
}

// NewWebRTCClient creates a new WebRTC client
func NewWebRTCClient(address string) (*WebRTCClient, error) {
	baseClient, err := NewBaseClient(address)
	if err != nil {
		return nil, err
	}

	return &WebRTCClient{
		BaseClient: baseClient,
		client:     webrtc.NewWebRTCServiceClient(baseClient.GetConnection()),
	}, nil
}

// Send sends a batch of frames to the WebRTC service
func (c *WebRTCClient) Send(ctx context.Context, batch model.FrameBatch) error {
	// Process all frames
	wg := sync.WaitGroup{}
	errors := make(chan error, len(batch.Frames))

	log.Printf("Frame Splitter: Sending frame batch to WebRTC out: stream=%s, frames=%d",
		batch.StreamID, len(batch.Frames))

	for _, frame := range batch.Frames {
		wg.Add(1)
		go func(f model.Frame) {
			defer wg.Done()

			// Determine frame type
			var frameType webrtc.FrameType
			switch f.Type {
			case model.FrameTypeVideo:
				frameType = webrtc.FrameType_VIDEO
			case model.FrameTypeAudio:
				frameType = webrtc.FrameType_AUDIO
			case model.FrameTypeMetadata:
				frameType = webrtc.FrameType_METADATA
			default:
				frameType = webrtc.FrameType_UNKNOWN
			}

			// Create frame request
			req := &webrtc.PushFrameRequest{
				StreamId:   f.StreamID,
				FrameId:    f.FrameID,
				Type:       frameType,
				Data:       f.Data,
				Timestamp:  timestamppb.New(f.Timestamp),
				IsKeyFrame: f.IsKeyFrame,
				Sequence:   f.Sequence,
				Metadata:   f.Metadata,
			}

			_, err := c.client.PushFrame(ctx, req)
			if err != nil {
				errors <- fmt.Errorf("failed to send frame to WebRTC: %w", err)
			}
		}(frame)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var errMessages []string
	for err := range errors {
		errMessages = append(errMessages, err.Error())
	}

	if len(errMessages) > 0 {
		return fmt.Errorf("errors sending frames to WebRTC: %s", strings.Join(errMessages, "; "))
	}

	return nil
}
