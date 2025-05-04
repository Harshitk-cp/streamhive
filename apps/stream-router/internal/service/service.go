package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	streamErrors "github.com/Harshitk-cp/streamhive/apps/stream-router/internal/errors"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/repository"

	"github.com/Harshitk-cp/streamhive/apps/stream-router/pkg/util"
)

// StreamService handles stream operations
type StreamService struct {
	streamRepo repository.StreamRepository
	eventRepo  repository.EventRepository
}

// NewStreamService creates a new stream service
func NewStreamService(streamRepo repository.StreamRepository, eventRepo repository.EventRepository) *StreamService {
	return &StreamService{
		streamRepo: streamRepo,
		eventRepo:  eventRepo,
	}
}

// Create creates a new stream
func (s *StreamService) Create(ctx context.Context, userID string, req model.StreamCreateRequest) (*model.Stream, error) {
	// Validate request
	if req.Title == "" {
		return nil, fmt.Errorf("%w: title is required", streamErrors.ErrInvalidInput)
	}

	// Create a new stream with default values
	stream := &model.Stream{
		ID:           util.GenerateID(),
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		Status:       model.StreamStatusIdle,
		Visibility:   req.Visibility,
		Tags:         req.Tags,
		Recording:    req.Recording,
		Region:       req.Region,
		Metadata:     req.Metadata,
		Key:          util.GenerateStreamKey(),
		Inputs:       []model.StreamInput{},
		Outputs:      []model.StreamOutput{},
		Enhancements: []model.StreamEnhancement{},
		Metrics:      model.StreamMetrics{},
	}

	// Save to repository
	err := s.streamRepo.Create(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create stream creation event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "created",
		Timestamp: time.Now(),
		Data:      stream,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create stream creation event: %v", err)
	}

	return stream, nil
}

// GetByID retrieves a stream by ID
func (s *StreamService) GetByID(ctx context.Context, id string) (*model.Stream, error) {
	stream, err := s.streamRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return nil, streamErrors.ErrStreamNotFound
		}
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	return stream, nil
}

// Update updates an existing stream
func (s *StreamService) Update(ctx context.Context, userID, streamID string, req model.StreamUpdateRequest) (*model.Stream, error) {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return nil, streamErrors.ErrStreamNotFound
		}
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return nil, streamErrors.ErrUnauthorized
	}

	// Update fields if provided
	if req.Title != nil {
		stream.Title = *req.Title
	}
	if req.Description != nil {
		stream.Description = *req.Description
	}
	if req.Visibility != nil {
		stream.Visibility = *req.Visibility
	}
	if req.Tags != nil {
		stream.Tags = req.Tags
	}
	if req.Recording != nil {
		stream.Recording = *req.Recording
	}
	if req.Region != nil {
		stream.Region = *req.Region
	}
	if req.Metadata != nil {
		stream.Metadata = req.Metadata
	}

	// Save changes
	err = s.streamRepo.Update(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to update stream: %w", err)
	}

	// Create stream update event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "updated",
		Timestamp: time.Now(),
		Data:      stream,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create stream update event: %v", err)
	}

	return stream, nil
}

// Delete deletes a stream
func (s *StreamService) Delete(ctx context.Context, userID, streamID string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Delete the stream
	err = s.streamRepo.Delete(ctx, streamID)
	if err != nil {
		return fmt.Errorf("failed to delete stream: %w", err)
	}

	// Create stream deletion event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "deleted",
		Timestamp: time.Now(),
		Data:      stream,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create stream deletion event: %v", err)
	}

	return nil
}

// List lists streams based on filters
func (s *StreamService) List(ctx context.Context, opts model.StreamListOptions) ([]*model.Stream, int, error) {
	streams, count, err := s.streamRepo.List(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list streams: %w", err)
	}

	return streams, count, nil
}

// GetUserStreams gets streams for a specific user
func (s *StreamService) GetUserStreams(ctx context.Context, userID string) ([]*model.Stream, error) {
	streams, err := s.streamRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user streams: %w", err)
	}

	return streams, nil
}

// StartStream starts a stream
func (s *StreamService) StartStream(ctx context.Context, userID string, req model.StreamStartRequest) (*model.StreamStartResponse, error) {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, req.StreamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return nil, streamErrors.ErrStreamNotFound
		}
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return nil, streamErrors.ErrUnauthorized
	}

	// Check if the stream is already live
	if stream.Status == model.StreamStatusLive {
		return nil, streamErrors.ErrStreamAlreadyLive
	}

	// Generate stream input
	inputID := util.GenerateID()
	backupID := util.GenerateID()

	primaryURL := fmt.Sprintf("rtmp://rtmp-ingestor:1935/live/%s?key=%s", stream.ID, stream.Key)
	backupURL := fmt.Sprintf("rtmp://rtmp-ingestor:1935/live/%s?key=%s&backup=1", stream.ID, stream.Key)

	// Create primary input
	primaryInput := model.StreamInput{
		ID:       inputID,
		Protocol: req.Protocol,
		URL:      primaryURL,
		Backup:   false,
		Servers:  []string{"ingest-1", "ingest-2"}, // Would come from load balancer in production
	}

	// Create backup input
	backupInput := model.StreamInput{
		ID:       backupID,
		Protocol: req.Protocol,
		URL:      backupURL,
		Backup:   true,
		Servers:  []string{"ingest-3", "ingest-4"}, // Would come from load balancer in production
	}

	// Add inputs to stream
	err = s.streamRepo.AddInput(ctx, stream.ID, primaryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to add primary input: %w", err)
	}

	err = s.streamRepo.AddInput(ctx, stream.ID, backupInput)
	if err != nil {
		return nil, fmt.Errorf("failed to add backup input: %w", err)
	}

	// Update stream status to live
	err = s.streamRepo.UpdateStatus(ctx, stream.ID, model.StreamStatusLive)
	if err != nil {
		return nil, fmt.Errorf("failed to update stream status: %w", err)
	}

	// Create stream start event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "started",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"input_id":  inputID,
			"backup_id": backupID,
			"protocol":  req.Protocol,
			"settings":  req.Settings,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create stream start event: %v", err)
	}

	// Create response
	response := &model.StreamStartResponse{
		StreamID:  stream.ID,
		InputURL:  primaryURL,
		StreamKey: stream.Key,
		BackupURL: backupURL,
		ServerID:  "ingest-1", // Would come from load balancer in production
		BackupID:  "ingest-3", // Would come from load balancer in production
	}

	return response, nil
}

// StopStream stops a stream
func (s *StreamService) StopStream(ctx context.Context, userID string, req model.StreamStopRequest) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, req.StreamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Check if the stream is live
	if stream.Status != model.StreamStatusLive && !req.Force {
		return streamErrors.ErrStreamNotLive
	}

	// Update stream status to ended
	err = s.streamRepo.UpdateStatus(ctx, stream.ID, model.StreamStatusEnded)
	if err != nil {
		return fmt.Errorf("failed to update stream status: %w", err)
	}

	// Create stream stop event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "stopped",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"forced": req.Force,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create stream stop event: %v", err)
	}

	return nil
}

// UpdateStreamMetrics updates a stream's metrics
func (s *StreamService) UpdateStreamMetrics(ctx context.Context, streamID string, metrics model.StreamMetrics) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Update the stream metrics
	err = s.streamRepo.UpdateMetrics(ctx, streamID, metrics)
	if err != nil {
		return fmt.Errorf("failed to update stream metrics: %w", err)
	}

	// Create stream metrics update event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "metrics_updated",
		Timestamp: time.Now(),
		Data:      metrics,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create metrics update event: %v", err)
	}

	return nil
}

// AddStreamOutput adds an output to a stream
func (s *StreamService) AddStreamOutput(ctx context.Context, userID, streamID string, output model.StreamOutput) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Add the output to the stream
	err = s.streamRepo.AddOutput(ctx, streamID, output)
	if err != nil {
		return fmt.Errorf("failed to add stream output: %w", err)
	}

	// Create output added event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "output_added",
		Timestamp: time.Now(),
		Data:      output,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create output added event: %v", err)
	}

	return nil
}

// RemoveStreamOutput removes an output from a stream
func (s *StreamService) RemoveStreamOutput(ctx context.Context, userID, streamID, outputID string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Remove the output from the stream
	err = s.streamRepo.RemoveOutput(ctx, streamID, outputID)
	if err != nil {
		return fmt.Errorf("failed to remove stream output: %w", err)
	}

	// Create output removed event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "output_removed",
		Timestamp: time.Now(),
		Data: map[string]string{
			"output_id": outputID,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create output removed event: %v", err)
	}

	return nil
}

// AddStreamEnhancement adds an enhancement to a stream
func (s *StreamService) AddStreamEnhancement(ctx context.Context, userID, streamID string, enhancement model.StreamEnhancement) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Add the enhancement to the stream
	err = s.streamRepo.AddEnhancement(ctx, streamID, enhancement)
	if err != nil {
		return fmt.Errorf("failed to add stream enhancement: %w", err)
	}

	// Create enhancement added event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "enhancement_added",
		Timestamp: time.Now(),
		Data:      enhancement,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create enhancement added event: %v", err)
	}

	return nil
}

// UpdateStreamEnhancement updates an enhancement for a stream
func (s *StreamService) UpdateStreamEnhancement(ctx context.Context, userID, streamID string, enhancement model.StreamEnhancement) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Update the enhancement
	err = s.streamRepo.UpdateEnhancement(ctx, streamID, enhancement)
	if err != nil {
		return fmt.Errorf("failed to update stream enhancement: %w", err)
	}

	// Create enhancement updated event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "enhancement_updated",
		Timestamp: time.Now(),
		Data:      enhancement,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create enhancement updated event: %v", err)
	}

	return nil
}

// RemoveStreamEnhancement removes an enhancement from a stream
func (s *StreamService) RemoveStreamEnhancement(ctx context.Context, userID, streamID, enhancementID string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return streamErrors.ErrUnauthorized
	}

	// Remove the enhancement from the stream
	err = s.streamRepo.RemoveEnhancement(ctx, streamID, enhancementID)
	if err != nil {
		return fmt.Errorf("failed to remove stream enhancement: %w", err)
	}

	// Create enhancement removed event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "enhancement_removed",
		Timestamp: time.Now(),
		Data: map[string]string{
			"enhancement_id": enhancementID,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create enhancement removed event: %v", err)
	}

	return nil
}

// GetStreamEvents gets events for a stream
func (s *StreamService) GetStreamEvents(ctx context.Context, userID, streamID string, limit, offset int) ([]*model.StreamEvent, error) {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return nil, streamErrors.ErrStreamNotFound
		}
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return nil, streamErrors.ErrUnauthorized
	}

	// Get events for the stream
	events, err := s.eventRepo.GetEventsByStreamID(ctx, streamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream events: %w", err)
	}

	return events, nil
}

// UpdateStreamStatus updates a stream's status (used by ingestor service)
func (s *StreamService) UpdateStreamStatus(ctx context.Context, streamID string, status model.StreamStatus) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return streamErrors.ErrStreamNotFound
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Update the stream status
	err = s.streamRepo.UpdateStatus(ctx, streamID, status)
	if err != nil {
		return fmt.Errorf("failed to update stream status: %w", err)
	}

	// Create status change event
	event := &model.StreamEvent{
		ID:        util.GenerateID(),
		StreamID:  stream.ID,
		Type:      "status_changed",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"previous_status": stream.Status,
			"new_status":      status,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create status change event: %v", err)
	}

	return nil
}

// ValidateStreamKey validates a stream key for a given stream ID (used by ingestor service)
func (s *StreamService) ValidateStreamKey(ctx context.Context, streamID, streamKey string) (bool, error) {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return false, streamErrors.ErrStreamNotFound
		}
		return false, fmt.Errorf("failed to get stream: %w", err)
	}

	// Validate the stream key
	return stream.Key == streamKey, nil
}
