package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	streamErrors "github.com/Harshitk-cp/streamhive/apps/stream-router/internal/errors"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/repository"
)

// MetadataService handles stream metadata operations
type MetadataService struct {
	streamRepo    repository.StreamRepository
	eventRepo     repository.EventRepository
	cacheTTL      time.Duration
	metadataCache map[string]metadataCacheEntry
	cacheMutex    sync.RWMutex
}

// metadataCacheEntry represents a cached metadata entry
type metadataCacheEntry struct {
	metadata  map[string]string
	expiresAt time.Time
}

// NewMetadataService creates a new metadata service
func NewMetadataService(streamRepo repository.StreamRepository, eventRepo repository.EventRepository) *MetadataService {
	return &MetadataService{
		streamRepo:    streamRepo,
		eventRepo:     eventRepo,
		cacheTTL:      5 * time.Minute,
		metadataCache: make(map[string]metadataCacheEntry),
	}
}

// GetMetadata gets metadata for a stream
func (s *MetadataService) GetMetadata(ctx context.Context, streamID string) (map[string]string, error) {
	// Check cache first
	s.cacheMutex.RLock()
	entry, exists := s.metadataCache[streamID]
	s.cacheMutex.RUnlock()

	// If valid cache entry exists, return it
	if exists && time.Now().Before(entry.expiresAt) {
		return entry.metadata, nil
	}

	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return nil, errors.New("stream not found")
		}
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Update cache
	s.cacheMutex.Lock()
	s.metadataCache[streamID] = metadataCacheEntry{
		metadata:  stream.Metadata,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()

	return stream.Metadata, nil
}

// SetMetadata sets metadata for a stream
func (s *MetadataService) SetMetadata(ctx context.Context, userID, streamID string, metadata map[string]string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return errors.New("stream not found")
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return errors.New("unauthorized access to stream")
	}

	// Save previous metadata for event
	previousMetadata := stream.Metadata

	// Update stream metadata
	stream.Metadata = metadata
	stream.UpdatedAt = time.Now()

	// Update the stream
	err = s.streamRepo.Update(ctx, stream)
	if err != nil {
		return fmt.Errorf("failed to update stream: %w", err)
	}

	// Update cache
	s.cacheMutex.Lock()
	s.metadataCache[streamID] = metadataCacheEntry{
		metadata:  metadata,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()

	// Create metadata update event
	event := &model.StreamEvent{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		StreamID:  stream.ID,
		Type:      "metadata_updated",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"previous": previousMetadata,
			"current":  metadata,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create metadata update event: %v", err)
	}

	return nil
}

// UpdateMetadata updates specific metadata fields for a stream
func (s *MetadataService) UpdateMetadata(ctx context.Context, userID, streamID string, updates map[string]string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return errors.New("stream not found")
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return errors.New("unauthorized access to stream")
	}

	// Save previous metadata for event
	previousMetadata := make(map[string]string)
	for k, v := range stream.Metadata {
		previousMetadata[k] = v
	}

	// Initialize metadata map if nil
	if stream.Metadata == nil {
		stream.Metadata = make(map[string]string)
	}

	// Update stream metadata
	for k, v := range updates {
		stream.Metadata[k] = v
	}
	stream.UpdatedAt = time.Now()

	// Update the stream
	err = s.streamRepo.Update(ctx, stream)
	if err != nil {
		return fmt.Errorf("failed to update stream: %w", err)
	}

	// Update cache
	s.cacheMutex.Lock()
	s.metadataCache[streamID] = metadataCacheEntry{
		metadata:  stream.Metadata,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()

	// Create metadata update event
	event := &model.StreamEvent{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		StreamID:  stream.ID,
		Type:      "metadata_updated",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"previous": previousMetadata,
			"current":  stream.Metadata,
			"updated":  updates,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create metadata update event: %v", err)
	}

	return nil
}

// DeleteMetadataField deletes a metadata field for a stream
func (s *MetadataService) DeleteMetadataField(ctx context.Context, userID, streamID, key string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return errors.New("stream not found")
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Check if the user owns the stream
	if stream.UserID != userID {
		return errors.New("unauthorized access to stream")
	}

	// Check if field exists
	if stream.Metadata == nil || stream.Metadata[key] == "" {
		return errors.New("metadata field not found")
	}

	// Save previous metadata for event
	previousMetadata := make(map[string]string)
	for k, v := range stream.Metadata {
		previousMetadata[k] = v
	}

	// Remove the field
	delete(stream.Metadata, key)
	stream.UpdatedAt = time.Now()

	// Update the stream
	err = s.streamRepo.Update(ctx, stream)
	if err != nil {
		return fmt.Errorf("failed to update stream: %w", err)
	}

	// Update cache
	s.cacheMutex.Lock()
	s.metadataCache[streamID] = metadataCacheEntry{
		metadata:  stream.Metadata,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMutex.Unlock()

	// Create metadata update event
	event := &model.StreamEvent{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		StreamID:  stream.ID,
		Type:      "metadata_field_deleted",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"previous": previousMetadata,
			"current":  stream.Metadata,
			"deleted":  key,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create metadata field deletion event: %v", err)
	}

	return nil
}

// UpdateThumbnail updates the thumbnail for a stream
func (s *MetadataService) UpdateThumbnail(ctx context.Context, streamID, thumbnailURL string) error {
	// Get the stream
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		if errors.Is(err, streamErrors.ErrStreamNotFound) {
			return errors.New("stream not found")
		}
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Save previous thumbnail for event
	previousThumbnail := stream.Thumbnail

	// Update stream thumbnail
	stream.Thumbnail = thumbnailURL
	stream.UpdatedAt = time.Now()

	// Update the stream
	err = s.streamRepo.Update(ctx, stream)
	if err != nil {
		return fmt.Errorf("failed to update stream: %w", err)
	}

	// Create thumbnail update event
	event := &model.StreamEvent{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		StreamID:  stream.ID,
		Type:      "thumbnail_updated",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"previous": previousThumbnail,
			"current":  thumbnailURL,
		},
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		log.Printf("Failed to create thumbnail update event: %v", err)
	}

	return nil
}

// InvalidateCache invalidates the cache for a stream
func (s *MetadataService) InvalidateCache(streamID string) {
	s.cacheMutex.Lock()
	delete(s.metadataCache, streamID)
	s.cacheMutex.Unlock()
}

// PeriodicallyCleanCache periodically cleans expired cache entries
func (s *MetadataService) PeriodicallyCleanCache(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.cleanExpiredCache()
		}
	}()
}

// cleanExpiredCache cleans expired cache entries
func (s *MetadataService) cleanExpiredCache() {
	now := time.Now()
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	for streamID, entry := range s.metadataCache {
		if now.After(entry.expiresAt) {
			delete(s.metadataCache, streamID)
		}
	}
}
