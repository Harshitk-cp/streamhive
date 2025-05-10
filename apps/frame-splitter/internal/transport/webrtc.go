package transport

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	webrtcpb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
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
	log.Printf("Frame Splitter: Sending frame batch to WebRTC out: stream=%s, frames=%d",
		batch.StreamID, len(batch.Frames))

	// Process all frames
	wg := sync.WaitGroup{}
	errors := make(chan error, len(batch.Frames))

	for _, frame := range batch.Frames {
		wg.Add(1)
		go func(f model.Frame) {
			defer wg.Done()

			// Determine frame type
			var frameType webrtcpb.FrameType
			switch f.Type {
			case model.FrameTypeVideo:
				frameType = webrtcpb.FrameType_VIDEO
			case model.FrameTypeAudio:
				frameType = webrtcpb.FrameType_AUDIO
			case model.FrameTypeMetadata:
				frameType = webrtcpb.FrameType_METADATA
			default:
				frameType = webrtcpb.FrameType_UNKNOWN
			}

			// Log the frame type and size
			log.Printf("Sending %s frame to WebRTC Out: size=%d bytes, keyframe=%v",
				f.Type, len(f.Data), f.IsKeyFrame)

			// Create frame request
			req := &webrtcpb.PushFrameRequest{
				StreamId:   f.StreamID,
				FrameId:    f.FrameID,
				Type:       frameType,
				Data:       f.Data,
				Timestamp:  timestamppb.New(f.Timestamp),
				IsKeyFrame: f.IsKeyFrame,
				Sequence:   f.Sequence,
				Metadata:   f.Metadata,
			}

			// Add timeout
			reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			resp, err := c.client.PushFrame(reqCtx, req)
			if err != nil {
				log.Printf("Error sending frame to WebRTC: %v", err)
				errors <- fmt.Errorf("failed to send frame to WebRTC: %w", err)
				return
			}

			log.Printf("Successfully sent frame to WebRTC, response: %s", resp.Status)
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
