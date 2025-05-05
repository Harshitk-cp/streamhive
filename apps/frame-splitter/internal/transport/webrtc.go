package transport

import (
	"context"
	"fmt"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	webrtcpb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	// Convert frames to protobuf
	frames := make([]*webrtcpb.Frame, 0, len(batch.Frames))
	for _, frame := range batch.Frames {
		frames = append(frames, &webrtcpb.Frame{
			StreamId:   frame.StreamID,
			FrameId:    frame.FrameID,
			Timestamp:  timestamppb.New(frame.Timestamp),
			Type:       webrtcpb.FrameType(webrtcpb.FrameType_value[string(frame.Type)]),
			Data:       frame.Data,
			Metadata:   frame.Metadata,
			Sequence:   frame.Sequence,
			IsKeyFrame: frame.IsKeyFrame,
			Duration:   frame.Duration,
		})
	}

	// Create request
	req := &webrtcpb.SendFramesRequest{
		StreamId: batch.StreamID,
		Frames:   frames,
	}

	// Send request
	_, err := c.client.SendFrames(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send frames to WebRTC: %w", err)
	}

	return nil
}
