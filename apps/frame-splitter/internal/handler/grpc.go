package handler

import (
	"context"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/processor"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/pkg/util"
	framepb "github.com/Harshitk-cp/streamhive/libs/proto/frame"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler handles gRPC requests for the frame splitter
type GRPCHandler struct {
	processor *processor.FrameProcessor
	framepb.UnimplementedFrameSplitterServiceServer
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(processor *processor.FrameProcessor) *GRPCHandler {
	return &GRPCHandler{
		processor: processor,
	}
}

// ProcessFrame processes a single frame
func (h *GRPCHandler) ProcessFrame(ctx context.Context, req *framepb.ProcessFrameRequest) (*framepb.ProcessFrameResponse, error) {
	// Generate a frame ID if not provided
	frameID := req.Frame.FrameId
	if frameID == "" {
		frameID = util.GenerateID()
	}

	// Convert request to internal model
	frame := model.Frame{
		StreamID:   req.Frame.StreamId,
		FrameID:    frameID,
		Timestamp:  req.Frame.Timestamp.AsTime(),
		Type:       model.FrameType(req.Frame.Type.String()),
		Data:       req.Frame.Data,
		Metadata:   req.Frame.Metadata,
		Sequence:   req.Frame.Sequence,
		IsKeyFrame: req.Frame.IsKeyFrame,
		Duration:   req.Frame.Duration,
	}

	// Process frame
	result, err := h.processor.ProcessFrame(ctx, frame)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process frame: %v", err)
	}

	// Convert result to response
	response := &framepb.ProcessFrameResponse{
		FrameId:      result.FrameID,
		Status:       result.Status,
		Error:        result.Error,
		Destinations: result.Destinations,
		Metadata:     result.Metadata,
	}

	return response, nil
}

// ProcessFrameBatch processes a batch of frames
func (h *GRPCHandler) ProcessFrameBatch(ctx context.Context, req *framepb.ProcessFrameBatchRequest) (*framepb.ProcessFrameBatchResponse, error) {
	// Convert request to internal model
	frames := make([]model.Frame, 0, len(req.Frames))
	for _, pbFrame := range req.Frames {
		frame := model.Frame{
			StreamID:   pbFrame.StreamId,
			FrameID:    pbFrame.FrameId,
			Timestamp:  pbFrame.Timestamp.AsTime(),
			Type:       model.FrameType(pbFrame.Type.String()),
			Data:       pbFrame.Data,
			Metadata:   pbFrame.Metadata,
			Sequence:   pbFrame.Sequence,
			IsKeyFrame: pbFrame.IsKeyFrame,
			Duration:   pbFrame.Duration,
		}
		frames = append(frames, frame)
	}

	batch := model.FrameBatch{
		StreamID: req.StreamId,
		Frames:   frames,
	}

	// Process batch
	result, err := h.processor.ProcessFrameBatch(ctx, batch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process frame batch: %v", err)
	}

	// Convert result to response
	results := make([]*framepb.ProcessFrameResponse, 0, len(result.Results))
	for _, res := range result.Results {
		results = append(results, &framepb.ProcessFrameResponse{
			FrameId:      res.FrameID,
			Status:       res.Status,
			Error:        res.Error,
			Destinations: res.Destinations,
			Metadata:     res.Metadata,
		})
	}

	response := &framepb.ProcessFrameBatchResponse{
		Results: results,
	}

	return response, nil
}

// GetStreamConfig gets the configuration for a stream
func (h *GRPCHandler) GetStreamConfig(ctx context.Context, req *framepb.GetStreamConfigRequest) (*framepb.StreamConfig, error) {
	// Get stream configuration
	config, err := h.processor.GetStreamConfig(req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "stream configuration not found: %v", err)
	}

	// Convert configuration to response
	routingRules := make([]*framepb.RoutingRule, 0, len(config.RoutingRules))
	for _, rule := range config.RoutingRules {
		routingRules = append(routingRules, &framepb.RoutingRule{
			Destination: rule.Destination,
			Filter:      rule.Filter,
			Priority:    int32(rule.Priority),
			Enabled:     rule.Enabled,
			BatchSize:   int32(rule.BatchSize),
		})
	}

	response := &framepb.StreamConfig{
		StreamId:      config.StreamID,
		RoutingRules:  routingRules,
		BackupEnabled: config.BackupEnabled,
		StoragePath:   config.StoragePath,
		RetentionTime: int64(config.RetentionTime.Seconds()),
	}

	return response, nil
}

// UpdateStreamConfig updates the configuration for a stream
func (h *GRPCHandler) UpdateStreamConfig(ctx context.Context, req *framepb.StreamConfig) (*framepb.UpdateStreamConfigResponse, error) {
	// Convert request to internal model
	routingRules := make([]model.RoutingRule, 0, len(req.RoutingRules))
	for _, rule := range req.RoutingRules {
		routingRules = append(routingRules, model.RoutingRule{
			Destination: rule.Destination,
			Filter:      rule.Filter,
			Priority:    int(rule.Priority),
			Enabled:     rule.Enabled,
			BatchSize:   int(rule.BatchSize),
		})
	}

	config := model.StreamConfig{
		StreamID:      req.StreamId,
		RoutingRules:  routingRules,
		BackupEnabled: req.BackupEnabled,
		StoragePath:   req.StoragePath,
		RetentionTime: time.Duration(req.RetentionTime) * time.Second,
	}

	// Update stream configuration
	err := h.processor.UpdateStreamConfig(config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update stream configuration: %v", err)
	}

	response := &framepb.UpdateStreamConfigResponse{
		Status: "ok",
	}

	return response, nil
}

// GetStreamStats gets the stats for a stream
func (h *GRPCHandler) GetStreamStats(ctx context.Context, req *framepb.GetStreamStatsRequest) (*framepb.StreamStats, error) {
	// Get stream stats
	stats, err := h.processor.GetStreamStats(req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "stream stats not found: %v", err)
	}

	// Convert stats to response
	response := &framepb.StreamStats{
		StreamId:          stats.StreamID,
		FramesProcessed:   stats.FramesProcessed,
		BytesProcessed:    stats.BytesProcessed,
		FramesDropped:     stats.FramesDropped,
		AvgProcessingTime: float64(stats.AvgProcessingTime.Microseconds()) / 1000,
		LastProcessedAt:   timestamppb.New(stats.LastProcessedAt),
		LastKeyFrameAt:    timestamppb.New(stats.LastKeyFrameAt),
		VideoFrameRate:    stats.VideoFrameRate,
		AudioFrameRate:    stats.AudioFrameRate,
		VideoBitrate:      stats.VideoBitrate,
		AudioBitrate:      stats.AudioBitrate,
	}

	return response, nil
}

// RestoreFrame restores a frame from backup
func (h *GRPCHandler) RestoreFrame(ctx context.Context, req *framepb.RestoreFrameRequest) (*framepb.Frame, error) {
	// Get the backup processor from the frame processor
	backupProcessor := h.processor.GetBackupProcessor()
	if backupProcessor == nil {
		return nil, status.Error(codes.Unavailable, "backup processor not available")
	}

	// Restore the frame
	frame, err := backupProcessor.RestoreFrame(req.StreamId, req.FrameId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to restore frame: %v", err)
	}

	// Convert to proto frame
	protoFrame := &framepb.Frame{
		StreamId:   frame.StreamID,
		FrameId:    frame.FrameID,
		Timestamp:  timestamppb.New(frame.Timestamp),
		Type:       framepb.FrameType(framepb.FrameType_value[string(frame.Type)]),
		Data:       frame.Data,
		Metadata:   frame.Metadata,
		Sequence:   frame.Sequence,
		IsKeyFrame: frame.IsKeyFrame,
		Duration:   frame.Duration,
	}

	return protoFrame, nil
}
