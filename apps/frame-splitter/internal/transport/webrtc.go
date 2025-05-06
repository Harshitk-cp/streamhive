package transport

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
)

// WebRTCClient represents a client for the WebRTC service
type WebRTCClient struct {
	*BaseClient
	client webrtcPb.WebRTCServiceClient
}

// NewWebRTCClient creates a new WebRTC client
func NewWebRTCClient(address string) (*WebRTCClient, error) {
	baseClient, err := NewBaseClient(address)
	if err != nil {
		return nil, err
	}

	return &WebRTCClient{
		BaseClient: baseClient,
		client:     webrtcPb.NewWebRTCServiceClient(baseClient.GetConnection()),
	}, nil
}

// Send sends a batch of frames to the WebRTC service
// Add to apps/frame-splitter/internal/transport/webrtc.go
func (c *WebRTCClient) Send(ctx context.Context, batch model.FrameBatch) error {
	// Group frames by type
	videoFrames := make([]model.Frame, 0)
	audioFrames := make([]model.Frame, 0)

	for _, frame := range batch.Frames {
		switch frame.Type {
		case model.FrameTypeVideo:
			videoFrames = append(videoFrames, frame)
		case model.FrameTypeAudio:
			audioFrames = append(audioFrames, frame)
		}
	}

	// Process video frames
	wg := sync.WaitGroup{}
	errors := make(chan error, len(videoFrames)+len(audioFrames))

	// Send video frames
	for _, frame := range videoFrames {
		wg.Add(1)
		go func(f model.Frame) {
			defer wg.Done()
			req := &webrtcPb.AddVideoFrameRequest{
				StreamId:   f.StreamID,
				FrameData:  f.Data,
				IsKeyFrame: f.IsKeyFrame,
				Timestamp:  f.Timestamp.UnixNano() / int64(time.Millisecond),
			}

			_, err := c.client.AddVideoFrame(ctx, req)
			if err != nil {
				errors <- fmt.Errorf("failed to send video frame: %w", err)
			}
		}(frame)
	}

	// Send audio frames
	for _, frame := range audioFrames {
		wg.Add(1)
		go func(f model.Frame) {
			defer wg.Done()
			req := &webrtcPb.AddAudioFrameRequest{
				StreamId:  f.StreamID,
				FrameData: f.Data,
				Timestamp: f.Timestamp.UnixNano() / int64(time.Millisecond),
			}

			_, err := c.client.AddAudioFrame(ctx, req)
			if err != nil {
				errors <- fmt.Errorf("failed to send audio frame: %w", err)
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
