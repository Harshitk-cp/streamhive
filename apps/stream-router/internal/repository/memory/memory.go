package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
)

var (
	// ErrStreamNotFound is returned when a stream is not found
	ErrStreamNotFound = errors.New("stream not found")

	// ErrInputNotFound is returned when a stream input is not found
	ErrInputNotFound = errors.New("stream input not found")

	// ErrOutputNotFound is returned when a stream output is not found
	ErrOutputNotFound = errors.New("stream output not found")

	// ErrEnhancementNotFound is returned when a stream enhancement is not found
	ErrEnhancementNotFound = errors.New("stream enhancement not found")
)

// MemoryStreamRepository implements StreamRepository with in-memory storage
type MemoryStreamRepository struct {
	streams map[string]*model.Stream // Map of streamID -> Stream
	mu      sync.RWMutex
}

// NewMemoryStreamRepository creates a new memory-based stream repository
func NewMemoryStreamRepository() *MemoryStreamRepository {
	return &MemoryStreamRepository{
		streams: make(map[string]*model.Stream),
	}
}

// Create adds a new stream to the repository
func (r *MemoryStreamRepository) Create(ctx context.Context, stream *model.Stream) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set timestamps
	now := time.Now()
	stream.CreatedAt = now
	stream.UpdatedAt = now

	// Store the stream
	r.streams[stream.ID] = stream

	return nil
}

// GetByID retrieves a stream by ID
func (r *MemoryStreamRepository) GetByID(ctx context.Context, id string) (*model.Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stream, exists := r.streams[id]
	if !exists {
		return nil, ErrStreamNotFound
	}

	// Return a copy to prevent concurrent modification
	streamCopy := *stream
	return &streamCopy, nil
}

// GetByUserID retrieves streams for a user
func (r *MemoryStreamRepository) GetByUserID(ctx context.Context, userID string) ([]*model.Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var userStreams []*model.Stream

	for _, stream := range r.streams {
		if stream.UserID == userID {
			// Create a copy to prevent concurrent modification
			streamCopy := *stream
			userStreams = append(userStreams, &streamCopy)
		}
	}

	return userStreams, nil
}

// Update updates an existing stream
func (r *MemoryStreamRepository) Update(ctx context.Context, stream *model.Stream) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.streams[stream.ID]
	if !exists {
		return ErrStreamNotFound
	}

	// Update timestamp
	stream.UpdatedAt = time.Now()

	// Update the stream
	r.streams[stream.ID] = stream

	return nil
}

// Delete deletes a stream
func (r *MemoryStreamRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.streams[id]
	if !exists {
		return ErrStreamNotFound
	}

	// Delete the stream
	delete(r.streams, id)

	return nil
}

// List retrieves streams based on options
func (r *MemoryStreamRepository) List(ctx context.Context, opts model.StreamListOptions) ([]*model.Stream, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filteredStreams []*model.Stream

	// Apply filters
	for _, stream := range r.streams {
		// Filter by userID if specified
		if opts.UserID != "" && stream.UserID != opts.UserID {
			continue
		}

		// Filter by status if specified
		if opts.Status != "" && stream.Status != opts.Status {
			continue
		}

		// Filter by visibility if specified
		if opts.Visibility != "" && stream.Visibility != opts.Visibility {
			continue
		}

		// Filter by tags if specified
		if len(opts.Tags) > 0 {
			matched := false
			for _, tag := range opts.Tags {
				for _, streamTag := range stream.Tags {
					if strings.EqualFold(tag, streamTag) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Create a copy to prevent concurrent modification
		streamCopy := *stream
		filteredStreams = append(filteredStreams, &streamCopy)
	}

	// Get total count before pagination
	totalCount := len(filteredStreams)

	// Sort the streams if sortBy is specified
	if opts.SortBy != "" {
		sort.Slice(filteredStreams, func(i, j int) bool {
			ascending := opts.SortOrder != "desc"

			switch opts.SortBy {
			case "created_at":
				if ascending {
					return filteredStreams[i].CreatedAt.Before(filteredStreams[j].CreatedAt)
				}
				return filteredStreams[i].CreatedAt.After(filteredStreams[j].CreatedAt)
			case "updated_at":
				if ascending {
					return filteredStreams[i].UpdatedAt.Before(filteredStreams[j].UpdatedAt)
				}
				return filteredStreams[i].UpdatedAt.After(filteredStreams[j].UpdatedAt)
			case "title":
				if ascending {
					return filteredStreams[i].Title < filteredStreams[j].Title
				}
				return filteredStreams[i].Title > filteredStreams[j].Title
			default:
				// Default sort by created_at desc
				return filteredStreams[i].CreatedAt.After(filteredStreams[j].CreatedAt)
			}
		})
	} else {
		// Default sort by created_at desc
		sort.Slice(filteredStreams, func(i, j int) bool {
			return filteredStreams[i].CreatedAt.After(filteredStreams[j].CreatedAt)
		})
	}

	// Apply pagination
	if opts.Limit > 0 {
		offset := opts.Offset
		if offset >= len(filteredStreams) {
			return []*model.Stream{}, totalCount, nil
		}

		end := offset + opts.Limit
		if end > len(filteredStreams) {
			end = len(filteredStreams)
		}

		filteredStreams = filteredStreams[offset:end]
	}

	return filteredStreams, totalCount, nil
}

// UpdateStatus updates a stream's status
func (r *MemoryStreamRepository) UpdateStatus(ctx context.Context, id string, status model.StreamStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[id]
	if !exists {
		return ErrStreamNotFound
	}

	// Update status and timestamp
	stream.Status = status
	stream.UpdatedAt = time.Now()

	// Update metrics if the stream is going live or ending
	if status == model.StreamStatusLive && stream.Metrics.StartTime.IsZero() {
		stream.Metrics.StartTime = time.Now()
	} else if (status == model.StreamStatusEnded || status == model.StreamStatusError) && !stream.Metrics.StartTime.IsZero() && stream.Metrics.EndTime.IsZero() {
		stream.Metrics.EndTime = time.Now()
		stream.Metrics.Duration = int64(stream.Metrics.EndTime.Sub(stream.Metrics.StartTime).Seconds())
	}

	return nil
}

// UpdateMetrics updates a stream's metrics
func (r *MemoryStreamRepository) UpdateMetrics(ctx context.Context, id string, metrics model.StreamMetrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[id]
	if !exists {
		return ErrStreamNotFound
	}

	// Update metrics and timestamp
	stream.Metrics = metrics
	stream.UpdatedAt = time.Now()

	return nil
}

// AddInput adds an input to a stream
func (r *MemoryStreamRepository) AddInput(ctx context.Context, streamID string, input model.StreamInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Append the input and update timestamp
	stream.Inputs = append(stream.Inputs, input)
	stream.UpdatedAt = time.Now()

	return nil
}

// RemoveInput removes an input from a stream
func (r *MemoryStreamRepository) RemoveInput(ctx context.Context, streamID string, inputID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Find and remove the input
	for i, input := range stream.Inputs {
		if input.ID == inputID {
			// Remove the input by replacing it with the last element and truncating the slice
			stream.Inputs[i] = stream.Inputs[len(stream.Inputs)-1]
			stream.Inputs = stream.Inputs[:len(stream.Inputs)-1]
			stream.UpdatedAt = time.Now()
			return nil
		}
	}

	return ErrInputNotFound
}

// AddOutput adds an output to a stream
func (r *MemoryStreamRepository) AddOutput(ctx context.Context, streamID string, output model.StreamOutput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Append the output and update timestamp
	stream.Outputs = append(stream.Outputs, output)
	stream.UpdatedAt = time.Now()

	return nil
}

// RemoveOutput removes an output from a stream
func (r *MemoryStreamRepository) RemoveOutput(ctx context.Context, streamID string, outputID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Find and remove the output
	for i, output := range stream.Outputs {
		if output.ID == outputID {
			// Remove the output by replacing it with the last element and truncating the slice
			stream.Outputs[i] = stream.Outputs[len(stream.Outputs)-1]
			stream.Outputs = stream.Outputs[:len(stream.Outputs)-1]
			stream.UpdatedAt = time.Now()
			return nil
		}
	}

	return ErrOutputNotFound
}

// AddEnhancement adds an enhancement to a stream
func (r *MemoryStreamRepository) AddEnhancement(ctx context.Context, streamID string, enhancement model.StreamEnhancement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Append the enhancement and update timestamp
	stream.Enhancements = append(stream.Enhancements, enhancement)
	stream.UpdatedAt = time.Now()

	// Sort enhancements by priority
	sort.Slice(stream.Enhancements, func(i, j int) bool {
		return stream.Enhancements[i].Priority < stream.Enhancements[j].Priority
	})

	return nil
}

// RemoveEnhancement removes an enhancement from a stream
func (r *MemoryStreamRepository) RemoveEnhancement(ctx context.Context, streamID string, enhancementID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Find and remove the enhancement
	for i, enhancement := range stream.Enhancements {
		if enhancement.ID == enhancementID {
			// Remove the enhancement by replacing it with the last element and truncating the slice
			stream.Enhancements[i] = stream.Enhancements[len(stream.Enhancements)-1]
			stream.Enhancements = stream.Enhancements[:len(stream.Enhancements)-1]
			stream.UpdatedAt = time.Now()
			return nil
		}
	}

	return ErrEnhancementNotFound
}

// UpdateEnhancement updates an enhancement for a stream
func (r *MemoryStreamRepository) UpdateEnhancement(ctx context.Context, streamID string, enhancement model.StreamEnhancement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stream, exists := r.streams[streamID]
	if !exists {
		return ErrStreamNotFound
	}

	// Find and update the enhancement
	for i, e := range stream.Enhancements {
		if e.ID == enhancement.ID {
			stream.Enhancements[i] = enhancement
			stream.UpdatedAt = time.Now()

			// Sort enhancements by priority
			sort.Slice(stream.Enhancements, func(i, j int) bool {
				return stream.Enhancements[i].Priority < stream.Enhancements[j].Priority
			})

			return nil
		}
	}

	return ErrEnhancementNotFound
}

// MemoryEventRepository implements EventRepository with in-memory storage
type MemoryEventRepository struct {
	events []*model.StreamEvent
	mu     sync.RWMutex
}

// NewMemoryEventRepository creates a new memory-based event repository
func NewMemoryEventRepository() *MemoryEventRepository {
	return &MemoryEventRepository{
		events: make([]*model.StreamEvent, 0),
	}
}

// CreateEvent stores a new stream event
func (r *MemoryEventRepository) CreateEvent(ctx context.Context, event *model.StreamEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store the event
	r.events = append(r.events, event)

	return nil
}

// GetEventsByStreamID retrieves events for a stream
func (r *MemoryEventRepository) GetEventsByStreamID(ctx context.Context, streamID string, limit, offset int) ([]*model.StreamEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var streamEvents []*model.StreamEvent

	// Filter by streamID
	for _, event := range r.events {
		if event.StreamID == streamID {
			// Create a copy to prevent concurrent modification
			eventCopy := *event
			streamEvents = append(streamEvents, &eventCopy)
		}
	}

	// Sort by timestamp descending
	sort.Slice(streamEvents, func(i, j int) bool {
		return streamEvents[i].Timestamp.After(streamEvents[j].Timestamp)
	})

	// Apply pagination
	if limit > 0 {
		if offset >= len(streamEvents) {
			return []*model.StreamEvent{}, nil
		}

		end := offset + limit
		if end > len(streamEvents) {
			end = len(streamEvents)
		}

		streamEvents = streamEvents[offset:end]
	}

	return streamEvents, nil
}

// GetEventsByType retrieves events of a specific type
func (r *MemoryEventRepository) GetEventsByType(ctx context.Context, eventType string, limit, offset int) ([]*model.StreamEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var typeEvents []*model.StreamEvent

	// Filter by event type
	for _, event := range r.events {
		if event.Type == eventType {
			// Create a copy to prevent concurrent modification
			eventCopy := *event
			typeEvents = append(typeEvents, &eventCopy)
		}
	}

	// Sort by timestamp descending
	sort.Slice(typeEvents, func(i, j int) bool {
		return typeEvents[i].Timestamp.After(typeEvents[j].Timestamp)
	})

	// Apply pagination
	if limit > 0 {
		if offset >= len(typeEvents) {
			return []*model.StreamEvent{}, nil
		}

		end := offset + limit
		if end > len(typeEvents) {
			end = len(typeEvents)
		}

		typeEvents = typeEvents[offset:end]
	}

	return typeEvents, nil
}
