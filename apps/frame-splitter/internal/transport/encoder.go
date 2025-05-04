// apps/frame-splitter/internal/transport/encoder.go
package transport

import (
	"context"
	"fmt"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	encoderpb "github.com/Harshitk-cp/streamhive/libs/proto/encoder"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EncoderClient represents a client for the encoder service
type EncoderClient struct {
	*BaseClient
	client encoderpb.EncoderServiceClient
}

// NewEncoderClient creates a new encoder client
func NewEncoderClient(address string) (*EncoderClient, error) {
	baseClient, err := NewBaseClient(address)
	if err != nil {
		return nil, err
	}

	return &EncoderClient{
		BaseClient: baseClient,
		client:     encoderpb.NewEncoderServiceClient(baseClient.GetConnection()),
	}, nil
}

// Send sends a batch of frames to the encoder service
func (c *EncoderClient) Send(ctx context.Context, batch model.FrameBatch) error {
	// Convert frames to protobuf
	frames := make([]*encoderpb.Frame, 0, len(batch.Frames))
	for _, frame := range batch.Frames {
		frames = append(frames, &encoderpb.Frame{
			StreamId:   frame.StreamID,
			FrameId:    frame.FrameID,
			Timestamp:  timestamppb.New(frame.Timestamp),
			Type:       encoderpb.FrameType(encoderpb.FrameType_value[string(frame.Type)]),
			Data:       frame.Data,
			Metadata:   frame.Metadata,
			Sequence:   frame.Sequence,
			IsKeyFrame: frame.IsKeyFrame,
			Duration:   frame.Duration,
		})
	}

	// Create request
	req := &encoderpb.EncodeFramesRequest{
		StreamId: batch.StreamID,
		Frames:   frames,
	}

	// Send request
	_, err := c.client.EncodeFrames(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to encode frames: %w", err)
	}

	return nil
}
