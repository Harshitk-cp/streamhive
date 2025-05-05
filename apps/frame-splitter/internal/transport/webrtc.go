package transport

import (
	"context"
	"fmt"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	webrtcpb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
)

// WebRTCClient represents a client for the WebRTC service
type WebRTCClient struct {
	*BaseClient
	client webrtcpb.WebRTCServiceClient
}

// NewWebRTCClient creates a new WebRTC client
func NewWebRTCClient(address string) (*WebRTCClient, error) {
	baseClient, err := NewBaseClient(address)
	if err != nil {
		return nil, err
	}

	return &WebRTCClient{
		BaseClient: baseClient,
		client:     webrtcpb.NewWebRTCServiceClient(baseClient.GetConnection()),
	}, nil
}

// Send sends a batch of frames to the WebRTC service
func (c *WebRTCClient) Send(ctx context.Context, batch model.FrameBatch) error {
	// Process each frame individually
	for _, frame := range batch.Frames {
		// Determine if it's a video or audio frame
		if string(frame.Type) == "VIDEO" {
			// Send video frame
			req := &webrtcpb.AddVideoFrameRequest{
				StreamId:   frame.StreamID,
				FrameData:  frame.Data,
				IsKeyFrame: frame.IsKeyFrame,
				Timestamp:  frame.Timestamp.UnixNano() / int64(1000000), // Convert to milliseconds
			}

			// Send request
			resp, err := c.client.AddVideoFrame(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to send video frame to WebRTC: %w", err)
			}

			if !resp.Success && !resp.Dropped {
				return fmt.Errorf("failed to send video frame to WebRTC: frame rejected")
			}
		} else if string(frame.Type) == "AUDIO" {
			// Send audio frame
			req := &webrtcpb.AddAudioFrameRequest{
				StreamId:  frame.StreamID,
				FrameData: frame.Data,
				Timestamp: frame.Timestamp.UnixNano() / int64(1000000), // Convert to milliseconds
			}

			// Send request
			resp, err := c.client.AddAudioFrame(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to send audio frame to WebRTC: %w", err)
			}

			if !resp.Success && !resp.Dropped {
				return fmt.Errorf("failed to send audio frame to WebRTC: frame rejected")
			}
		} else {
			// Skip other frame types like METADATA
			continue
		}
	}

	return nil
}
