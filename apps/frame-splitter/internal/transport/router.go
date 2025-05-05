package transport

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	routerpb "github.com/Harshitk-cp/streamhive/libs/proto/stream"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RouterClient represents a client for the router service
type RouterClient struct {
	*BaseClient
	client routerpb.StreamServiceClient
}

// NewRouterClient creates a new router client
func NewRouterClient(address string) (*RouterClient, error) {
	baseClient, err := NewBaseClient(address)
	if err != nil {
		return nil, err
	}

	return &RouterClient{
		BaseClient: baseClient,
		client:     routerpb.NewStreamServiceClient(baseClient.GetConnection()),
	}, nil
}

// GetStreamConfig gets the configuration for a stream
func (c *RouterClient) GetStreamConfig(ctx context.Context, streamID string) (*model.StreamConfig, error) {
	// Create request
	req := &routerpb.GetStreamRequest{
		Id: streamID,
	}

	// Send request
	stream, err := c.client.GetStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream from router: %w", err)
	}

	// Extract routing rules from stream enhancements
	routingRules := make([]model.RoutingRule, 0, len(stream.Enhancements))
	for _, enhancement := range stream.Enhancements {
		// Only include enabled enhancements
		if enhancement.Enabled {
			// Determine appropriate filter based on enhancement type
			filter := ""
			switch enhancement.Type {
			case "ai":
				filter = "frame.type == VIDEO && frame.is_key_frame == true"
			case "audio":
				filter = "frame.type == AUDIO"
			case "overlay":
				filter = "frame.type == VIDEO"
			case "metadata":
				filter = "frame.type == METADATA"
			default:
				filter = "" // Empty filter matches all frames
			}

			// Create routing rule
			rule := model.RoutingRule{
				Destination: enhancement.Type,
				Filter:      filter,
				Priority:    int(enhancement.Priority),
				Enabled:     enhancement.Enabled,
				BatchSize:   30, // Default batch size
			}

			// Check for custom batch size in settings
			if batchSizeStr, ok := enhancement.Settings["batch_size"]; ok {
				if batchSize, err := strconv.Atoi(batchSizeStr); err == nil {
					rule.BatchSize = batchSize
				}
			}

			routingRules = append(routingRules, rule)
		}
	}

	// If no routing rules were found, create default rules
	if len(routingRules) == 0 {
		routingRules = []model.RoutingRule{
			{
				Destination: "enhancement",
				Filter:      "frame.type == VIDEO && frame.is_key_frame == true",
				Priority:    1,
				Enabled:     true,
				BatchSize:   1,
			},
			{
				Destination: "encoder",
				Filter:      "frame.type == AUDIO",
				Priority:    2,
				Enabled:     true,
				BatchSize:   30,
			},
			{
				Destination: "webrtc",
				Filter:      "frame.type == VIDEO",
				Priority:    3,
				Enabled:     true,
				BatchSize:   1,
			},
		}
	}

	// Create stream config
	config := &model.StreamConfig{
		StreamID:      streamID,
		RoutingRules:  routingRules,
		BackupEnabled: true,
		StoragePath:   fmt.Sprintf("/data/frames/%s", streamID),
		RetentionTime: 24 * time.Hour,
	}

	// Use metadata from stream if available
	if stream.Metadata != nil {
		if backupEnabledStr, ok := stream.Metadata["backup_enabled"]; ok {
			config.BackupEnabled = backupEnabledStr == "true"
		}
		if storagePath, ok := stream.Metadata["storage_path"]; ok {
			config.StoragePath = storagePath
		}
		if retentionTimeStr, ok := stream.Metadata["retention_time"]; ok {
			if retentionHours, err := strconv.Atoi(retentionTimeStr); err == nil {
				config.RetentionTime = time.Duration(retentionHours) * time.Hour
			}
		}
	}

	return config, nil
}

// UpdateStreamMetrics updates metrics for a stream
func (c *RouterClient) UpdateStreamMetrics(ctx context.Context, streamID string, stats model.StreamStats) error {
	// Build stream metrics
	metrics := &routerpb.StreamMetrics{
		IngestBitrate: int32(stats.VideoBitrate + stats.AudioBitrate),
		OutputBitrate: int32(stats.VideoBitrate + stats.AudioBitrate),
		FrameRate:     stats.VideoFrameRate,
		Resolution:    "dynamic", // This would be extracted from frame metadata
		Duration:      0,         // Not applicable for metrics updates
	}

	// Set timestamps if available
	if !stats.LastProcessedAt.IsZero() {
		metrics.StartTime = timestamppb.New(stats.LastProcessedAt)
	}
	if !stats.LastKeyFrameAt.IsZero() {
		// Not a perfect mapping, but we'll use LastKeyFrameAt as a proxy for the most recent important event
		metrics.EndTime = timestamppb.New(stats.LastKeyFrameAt)
	}

	// Create request
	req := &routerpb.UpdateStreamMetricsRequest{
		StreamId: streamID,
		Metrics:  metrics,
	}

	// Send request
	_, err := c.client.UpdateStreamMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update stream metrics: %w", err)
	}

	return nil
}

// Send implements the FrameRoute interface
func (c *RouterClient) Send(ctx context.Context, batch model.FrameBatch) error {
	return fmt.Errorf("direct sending to router not implemented - frames should be routed to specific services")
}
