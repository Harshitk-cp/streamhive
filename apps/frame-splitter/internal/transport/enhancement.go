// apps/frame-splitter/internal/transport/enhancement.go
package transport

import (
	"context"
	"fmt"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	enhancementpb "github.com/Harshitk-cp/streamhive/libs/proto/enhancement"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EnhancementClient represents a client for the enhancement service
type EnhancementClient struct {
	*BaseClient
	client enhancementpb.EnhancementServiceClient
}

// NewEnhancementClient creates a new enhancement client
func NewEnhancementClient(address string) (*EnhancementClient, error) {
	baseClient, err := NewBaseClient(address)
	if err != nil {
		return nil, err
	}

	return &EnhancementClient{
		BaseClient: baseClient,
		client:     enhancementpb.NewEnhancementServiceClient(baseClient.GetConnection()),
	}, nil
}

// Send sends a batch of frames to the enhancement service
func (c *EnhancementClient) Send(ctx context.Context, batch model.FrameBatch) error {
	// Convert frames to protobuf
	frames := make([]*enhancementpb.Frame, 0, len(batch.Frames))
	for _, frame := range batch.Frames {
		frames = append(frames, &enhancementpb.Frame{
			StreamId:   frame.StreamID,
			FrameId:    frame.FrameID,
			Timestamp:  timestamppb.New(frame.Timestamp),
			Type:       enhancementpb.FrameType(enhancementpb.FrameType_value[string(frame.Type)]),
			Data:       frame.Data,
			Metadata:   frame.Metadata,
			Sequence:   frame.Sequence,
			IsKeyFrame: frame.IsKeyFrame,
			Duration:   frame.Duration,
		})
	}

	// Create request
	req := &enhancementpb.EnhanceFramesRequest{
		StreamId: batch.StreamID,
		Frames:   frames,
	}

	// Send request
	_, err := c.client.EnhanceFrames(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to enhance frames: %w", err)
	}

	return nil
}
