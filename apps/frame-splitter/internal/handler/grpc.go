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
	"google.golang.org/protobuf/types/known/emptypb"
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
func (h *GRPCHandler) ProcessFrame(ctx context.Context, frame *framepb.Frame) (*framepb.ProcessFrameResponse, error) {
	// Generate a frame ID if not provided
	frameID := frame.FrameId
	if frameID == "" {
		frameID = util.GenerateID()
	}

	// Convert request to internal model
	internalFrame := model.Frame{
		StreamID:   frame.StreamId,
		FrameID:    frameID,
		Timestamp:  frame.Timestamp.AsTime(),
		Type:       model.FrameType(frame.Type.String()),
		Data:       frame.Data,
		Metadata:   frame.Metadata,
		Sequence:   frame.Sequence,
		IsKeyFrame: frame.IsKeyFrame,
		// Duration:   frame.Duration,
	}

	// Process frame
	result, err := h.processor.ProcessFrame(ctx, internalFrame)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process frame: %v", err)
	}

	// Convert result to response
	response := &framepb.ProcessFrameResponse{
		FrameId: result.FrameID,
		Status:  result.Status,
	}

	// Handle error message differently if the field doesn't exist
	if result.Error != "" && result.Status != "accepted" {
		response.Status = "error: " + result.Error
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
			// Duration:   pbFrame.Duration,
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
		frameResponse := &framepb.ProcessFrameResponse{
			FrameId: res.FrameID,
			Status:  res.Status,
		}

		// Handle error message if needed
		if res.Error != "" && res.Status != "accepted" {
			frameResponse.Status = "error: " + res.Error
		}

		results = append(results, frameResponse)
	}

	response := &framepb.ProcessFrameBatchResponse{
		Results: results,
	}

	return response, nil
}

// RequestKeyFrame requests a key frame for a stream
func (h *GRPCHandler) RequestKeyFrame(ctx context.Context, req *framepb.RequestKeyFrameRequest) (*framepb.RequestKeyFrameResponse, error) {
	// Request a key frame from the processor
	streamID := req.StreamId

	// Get stream config to ensure it exists
	_, err := h.processor.GetStreamConfig(streamID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "stream not found: %v", err)
	}

	// In a real implementation, you'd trigger key frame generation
	// This could involve updating a flag in the stream configuration
	// or sending a signal to the encoder

	// Return a simple success response
	return &framepb.RequestKeyFrameResponse{
		Status: "key_frame_requested",
	}, nil
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
		// Duration:   frame.Duration,
	}

	return protoFrame, nil
}

// These are placeholder stubs for required interface methods
// You'll need to implement them properly or they'll return "unimplemented" errors

func (h *GRPCHandler) SubscribeToStream(req *framepb.SubscribeToStreamRequest, stream framepb.FrameSplitterService_SubscribeToStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeToStream not implemented")
}

func (h *GRPCHandler) RegisterConsumer(ctx context.Context, req *framepb.RegisterConsumerRequest) (*framepb.RegisterConsumerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterConsumer not implemented")
}

func (h *GRPCHandler) ListStreams(ctx context.Context, req *emptypb.Empty) (*framepb.ListStreamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStreams not implemented")
}
