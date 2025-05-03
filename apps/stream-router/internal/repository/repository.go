package repository

import (
	"context"

	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
)

// StreamRepository defines the interface for stream data storage
type StreamRepository interface {
	// Create creates a new stream
	Create(ctx context.Context, stream *model.Stream) error

	// GetByID retrieves a stream by ID
	GetByID(ctx context.Context, id string) (*model.Stream, error)

	// GetByUserID retrieves streams for a user
	GetByUserID(ctx context.Context, userID string) ([]*model.Stream, error)

	// Update updates an existing stream
	Update(ctx context.Context, stream *model.Stream) error

	// Delete deletes a stream
	Delete(ctx context.Context, id string) error

	// List retrieves streams based on options
	List(ctx context.Context, opts model.StreamListOptions) ([]*model.Stream, int, error)

	// UpdateStatus updates a stream's status
	UpdateStatus(ctx context.Context, id string, status model.StreamStatus) error

	// UpdateMetrics updates a stream's metrics
	UpdateMetrics(ctx context.Context, id string, metrics model.StreamMetrics) error

	// AddInput adds an input to a stream
	AddInput(ctx context.Context, streamID string, input model.StreamInput) error

	// RemoveInput removes an input from a stream
	RemoveInput(ctx context.Context, streamID string, inputID string) error

	// AddOutput adds an output to a stream
	AddOutput(ctx context.Context, streamID string, output model.StreamOutput) error

	// RemoveOutput removes an output from a stream
	RemoveOutput(ctx context.Context, streamID string, outputID string) error

	// AddEnhancement adds an enhancement to a stream
	AddEnhancement(ctx context.Context, streamID string, enhancement model.StreamEnhancement) error

	// RemoveEnhancement removes an enhancement from a stream
	RemoveEnhancement(ctx context.Context, streamID string, enhancementID string) error

	// UpdateEnhancement updates an enhancement for a stream
	UpdateEnhancement(ctx context.Context, streamID string, enhancement model.StreamEnhancement) error
}

// EventRepository defines the interface for stream event data storage
type EventRepository interface {
	// CreateEvent stores a new stream event
	CreateEvent(ctx context.Context, event *model.StreamEvent) error

	// GetEventsByStreamID retrieves events for a stream
	GetEventsByStreamID(ctx context.Context, streamID string, limit, offset int) ([]*model.StreamEvent, error)

	// GetEventsByType retrieves events of a specific type
	GetEventsByType(ctx context.Context, eventType string, limit, offset int) ([]*model.StreamEvent, error)
}
