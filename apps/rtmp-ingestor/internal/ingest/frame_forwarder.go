package ingest

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/libs/proto/frame"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FrameForwarder forwards frames to the Frame Splitter service
type FrameForwarder struct {
	frameSplitterAddr string
	client            frame.FrameSplitterServiceClient
	conn              *grpc.ClientConn
	batchSize         int
	frameBatches      map[string][]frame.Frame
	batchMutex        sync.Mutex
	stopChan          chan struct{}
	flushInterval     time.Duration
}

// convertToProtoFrames converts a slice of frame.Frame to a slice of *frame.Frame
func convertToProtoFrames(frames []frame.Frame) []*frame.Frame {
	protoFrames := make([]*frame.Frame, len(frames))
	for i, f := range frames {
		frameCopy := f // Create a copy to avoid referencing the same memory
		protoFrames[i] = &frameCopy
	}
	return protoFrames
}

// NewFrameForwarder creates a new frame forwarder
func NewFrameForwarder(frameSplitterAddr string, batchSize int) (*FrameForwarder, error) {
	// Connect to Frame Splitter service
	conn, err := grpc.Dial(
		frameSplitterAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Frame Splitter: %w", err)
	}

	// Create client
	client := frame.NewFrameSplitterServiceClient(conn)

	// Create forwarder
	forwarder := &FrameForwarder{
		frameSplitterAddr: frameSplitterAddr,
		client:            client,
		conn:              conn,
		batchSize:         batchSize,
		frameBatches:      make(map[string][]frame.Frame),
		stopChan:          make(chan struct{}),
		flushInterval:     500 * time.Millisecond, // Flush every 500ms
	}

	// Start background flush goroutine
	go forwarder.periodicFlush()

	return forwarder, nil
}

// Close closes the frame forwarder
func (f *FrameForwarder) Close() error {
	// Signal background goroutine to stop
	close(f.stopChan)

	// Close connection
	if f.conn != nil {
		return f.conn.Close()
	}
	return nil
}

// ForwardFrame forwards a frame to the Frame Splitter
func (f *FrameForwarder) ForwardFrame(streamID string, isKeyFrame bool, frameType string, data []byte, timestamp time.Time, sequence int64, metadata map[string]string) error {
	// Convert frame type to enum
	var protoFrameType frame.FrameType
	switch frameType {
	case "video":
		protoFrameType = frame.FrameType_VIDEO
	case "audio":
		protoFrameType = frame.FrameType_AUDIO
	default:
		protoFrameType = frame.FrameType_METADATA
	}

	// Create frame
	frameProto := frame.Frame{
		StreamId:   streamID,
		FrameId:    fmt.Sprintf("frm_%d", time.Now().UnixNano()),
		Timestamp:  timestamppb.New(timestamp),
		Type:       protoFrameType,
		Data:       data,
		Metadata:   metadata,
		Sequence:   sequence,
		IsKeyFrame: isKeyFrame,
	}

	// Add to batch
	f.batchMutex.Lock()
	defer f.batchMutex.Unlock()

	// Initialize batch if needed
	if _, exists := f.frameBatches[streamID]; !exists {
		f.frameBatches[streamID] = make([]frame.Frame, 0, f.batchSize)
	}

	// Add frame to batch
	f.frameBatches[streamID] = append(f.frameBatches[streamID], frameProto)

	// Check if batch is full
	if len(f.frameBatches[streamID]) >= f.batchSize {
		return f.flushBatch(streamID)
	}

	return nil
}

// flushBatch sends a batch of frames to the Frame Splitter
func (f *FrameForwarder) flushBatch(streamID string) error {
	// Get batch
	batch := f.frameBatches[streamID]
	if len(batch) == 0 {
		return nil
	}
	// Reset batch
	f.frameBatches[streamID] = make([]frame.Frame, 0, f.batchSize)

	// Release lock while sending
	f.batchMutex.Unlock()
	defer f.batchMutex.Lock()

	// Create request
	req := &frame.ProcessFrameBatchRequest{
		StreamId: streamID,
		Frames:   convertToProtoFrames(batch),
	}

	// Send request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := f.client.ProcessFrameBatch(ctx, req)
	if err != nil {
		log.Printf("Failed to forward frame batch: %v", err)
		return err
	}

	log.Printf("Forwarded batch of %d frames for stream %s", len(batch), streamID)
	return nil
}

// periodicFlush periodically flushes batches
func (f *FrameForwarder) periodicFlush() {
	ticker := time.NewTicker(f.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.flushAllBatches()
		case <-f.stopChan:
			return
		}
	}
}

// flushAllBatches flushes all batches
func (f *FrameForwarder) flushAllBatches() {
	f.batchMutex.Lock()
	defer f.batchMutex.Unlock()

	for streamID := range f.frameBatches {
		if len(f.frameBatches[streamID]) > 0 {
			f.flushBatch(streamID)
		}
	}
}
