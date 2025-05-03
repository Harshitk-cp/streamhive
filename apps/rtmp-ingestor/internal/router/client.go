package router

import (
	"context"
	"fmt"
	"time"

	routerpb "github.com/Harshitk-cp/streamhive/libs/proto/stream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client defines the interface for interacting with the stream router service
type Client interface {
	// ValidateStreamKey validates a stream key for a stream
	ValidateStreamKey(ctx context.Context, streamID, streamKey string) (bool, error)

	// UpdateStreamStatus updates the status of a stream
	UpdateStreamStatus(ctx context.Context, streamID, status string) error

	// UpdateStreamMetrics updates metrics for a stream
	UpdateStreamMetrics(ctx context.Context, streamID string, metrics StreamMetrics) error

	// Close closes the client connection
	Close() error
}

// StreamMetrics contains metrics for a stream
type StreamMetrics struct {
	ViewerCount     int
	PeakViewerCount int
	StartTime       time.Time
	EndTime         time.Time
	Duration        int64
	IngestBitrate   int
	OutputBitrate   int
	FrameRate       float64
	Resolution      string
}

// GRPCClient is a gRPC implementation of the Client interface
type GRPCClient struct {
	conn   *grpc.ClientConn
	client routerpb.StreamServiceClient
}

// NewGRPCClient creates a new GRPCClient
func NewGRPCClient(address string) (*GRPCClient, error) {
	// Set up connection parameters with retry
	connParams := grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  100 * time.Millisecond,
			Multiplier: 1.6,
			Jitter:     0.2,
			MaxDelay:   3 * time.Second,
		},
		MinConnectTimeout: 5 * time.Second,
	})

	// Connect to the stream router service
	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		connParams,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stream router: %w", err)
	}

	// Create client
	client := routerpb.NewStreamServiceClient(conn)

	return &GRPCClient{
		conn:   conn,
		client: client,
	}, nil
}

// ValidateStreamKey validates a stream key for a stream
func (c *GRPCClient) ValidateStreamKey(ctx context.Context, streamID, streamKey string) (bool, error) {
	// Create request
	req := &routerpb.ValidateStreamKeyRequest{
		StreamId:  streamID,
		StreamKey: streamKey,
	}

	// Call stream router service
	resp, err := c.client.ValidateStreamKey(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to validate stream key: %w", err)
	}

	return resp.Valid, nil
}

// UpdateStreamStatus updates the status of a stream
func (c *GRPCClient) UpdateStreamStatus(ctx context.Context, streamID, status string) error {
	// Create request
	req := &routerpb.UpdateStreamStatusRequest{
		StreamId: streamID,
		Status:   status,
	}

	// Call stream router service
	_, err := c.client.UpdateStreamStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update stream status: %w", err)
	}

	return nil
}

// UpdateStreamMetrics updates metrics for a stream
func (c *GRPCClient) UpdateStreamMetrics(ctx context.Context, streamID string, metrics StreamMetrics) error {
	// Create request with metrics
	req := &routerpb.UpdateStreamMetricsRequest{
		StreamId: streamID,
		Metrics: &routerpb.StreamMetrics{
			ViewerCount:     int32(metrics.ViewerCount),
			PeakViewerCount: int32(metrics.PeakViewerCount),
			IngestBitrate:   int32(metrics.IngestBitrate),
			OutputBitrate:   int32(metrics.OutputBitrate),
			FrameRate:       metrics.FrameRate,
			Resolution:      metrics.Resolution,
			Duration:        metrics.Duration,
		},
	}

	// Add timestamps if provided
	if !metrics.StartTime.IsZero() {
		req.Metrics.StartTime = timestamppb.New(metrics.StartTime)
	}
	if !metrics.EndTime.IsZero() {
		req.Metrics.EndTime = timestamppb.New(metrics.EndTime)
	}

	// Call stream router service
	_, err := c.client.UpdateStreamMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update stream metrics: %w", err)
	}

	return nil
}

// Close closes the client connection
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// MockClient is a mock implementation of the Client interface for testing
type MockClient struct {
	ValidateStreamKeyFunc   func(ctx context.Context, streamID, streamKey string) (bool, error)
	UpdateStreamStatusFunc  func(ctx context.Context, streamID, status string) error
	UpdateStreamMetricsFunc func(ctx context.Context, streamID string, metrics StreamMetrics) error
}

// ValidateStreamKey validates a stream key for a stream
func (c *MockClient) ValidateStreamKey(ctx context.Context, streamID, streamKey string) (bool, error) {
	if c.ValidateStreamKeyFunc != nil {
		return c.ValidateStreamKeyFunc(ctx, streamID, streamKey)
	}
	return true, nil
}

// UpdateStreamStatus updates the status of a stream
func (c *MockClient) UpdateStreamStatus(ctx context.Context, streamID, status string) error {
	if c.UpdateStreamStatusFunc != nil {
		return c.UpdateStreamStatusFunc(ctx, streamID, status)
	}
	return nil
}

// UpdateStreamMetrics updates metrics for a stream
func (c *MockClient) UpdateStreamMetrics(ctx context.Context, streamID string, metrics StreamMetrics) error {
	if c.UpdateStreamMetricsFunc != nil {
		return c.UpdateStreamMetricsFunc(ctx, streamID, metrics)
	}
	return nil
}

// Close closes the client connection
func (c *MockClient) Close() error {
	return nil
}
