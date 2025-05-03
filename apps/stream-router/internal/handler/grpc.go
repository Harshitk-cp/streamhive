package handler

import (
	"context"
	"errors"
	"fmt"

	streamErrors "github.com/Harshitk-cp/streamhive/apps/stream-router/internal/errors"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/service"
	routerpb "github.com/Harshitk-cp/streamhive/libs/proto/stream"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler handles gRPC requests
type GRPCHandler struct {
	routerpb.UnimplementedStreamServiceServer
	streamService       *service.StreamService
	metadataService     *service.MetadataService
	notificationService *service.NotificationService
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(
	streamService *service.StreamService,
	metadataService *service.MetadataService,
	notificationService *service.NotificationService,
) *GRPCHandler {
	return &GRPCHandler{
		streamService:       streamService,
		metadataService:     metadataService,
		notificationService: notificationService,
	}
}

// CreateStream handles creating a new stream
func (h *GRPCHandler) CreateStream(ctx context.Context, req *routerpb.CreateStreamRequest) (*routerpb.Stream, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Create stream request
	createReq := model.StreamCreateRequest{
		Title:       req.Title,
		Description: req.Description,
		Visibility:  model.StreamVisibility(req.Visibility),
		Tags:        req.Tags,
		Recording:   req.Recording,
		Region:      req.Region,
		Metadata:    req.Metadata,
	}

	// Create the stream
	stream, err := h.streamService.Create(ctx, userID, createReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create stream: %v", err)
	}

	// Convert to protobuf response
	return convertModelStreamToProto(stream), nil
}

// GetStream handles retrieving a stream
func (h *GRPCHandler) GetStream(ctx context.Context, req *routerpb.GetStreamRequest) (*routerpb.Stream, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Get the stream
	stream, err := h.streamService.GetByID(ctx, req.Id)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get stream: %v", err)
	}

	// Check if user has access to the stream
	if stream.UserID != userID && stream.Visibility != model.StreamVisibilityPublic {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Convert to protobuf response
	return convertModelStreamToProto(stream), nil
}

// UpdateStream handles updating a stream
func (h *GRPCHandler) UpdateStream(ctx context.Context, req *routerpb.UpdateStreamRequest) (*routerpb.Stream, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Create update request
	var title *string
	var description *string
	var visibility *model.StreamVisibility
	var recording *bool
	var region *string

	if req.Title != nil {
		titleStr := req.Title.GetValue()
		title = &titleStr
	}
	if req.Description != nil {
		descStr := req.Description.GetValue()
		description = &descStr
	}
	if req.Visibility != nil {
		visStr := model.StreamVisibility(req.Visibility.GetValue())
		visibility = &visStr
	}
	if req.Recording != nil {
		recBool := req.Recording.GetValue()
		recording = &recBool
	}
	if req.Region != nil {
		regStr := req.Region.GetValue()
		region = &regStr
	}

	updateReq := model.StreamUpdateRequest{
		Title:       title,
		Description: description,
		Visibility:  visibility,
		Tags:        req.Tags,
		Recording:   recording,
		Region:      region,
		Metadata:    req.Metadata,
	}

	// Update the stream
	stream, err := h.streamService.Update(ctx, userID, req.Id, updateReq)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to update stream: %v", err)
	}

	// Convert to protobuf response
	return convertModelStreamToProto(stream), nil
}

// DeleteStream handles deleting a stream
func (h *GRPCHandler) DeleteStream(ctx context.Context, req *routerpb.DeleteStreamRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Delete the stream
	err = h.streamService.Delete(ctx, userID, req.Id)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete stream: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListStreams handles listing streams
func (h *GRPCHandler) ListStreams(ctx context.Context, req *routerpb.ListStreamsRequest) (*routerpb.ListStreamsResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Create list options
	opts := model.StreamListOptions{
		UserID:     userID,
		Status:     model.StreamStatus(req.Status),
		Visibility: model.StreamVisibility(req.Visibility),
		Tags:       req.Tags,
		Limit:      int(req.Limit),
		Offset:     int(req.Offset),
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	// Get streams
	streams, count, err := h.streamService.List(ctx, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list streams: %v", err)
	}

	// Convert streams to protobuf
	protoStreams := make([]*routerpb.Stream, 0, len(streams))
	for _, stream := range streams {
		protoStreams = append(protoStreams, convertModelStreamToProto(stream))
	}

	// Create response
	response := &routerpb.ListStreamsResponse{
		Streams: protoStreams,
		Count:   int32(count),
		Limit:   int32(opts.Limit),
		Offset:  int32(opts.Offset),
	}

	return response, nil
}

// StartStream handles starting a stream
func (h *GRPCHandler) StartStream(ctx context.Context, req *routerpb.StartStreamRequest) (*routerpb.StartStreamResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Create start request
	startReq := model.StreamStartRequest{
		StreamID: req.StreamId,
		Protocol: req.Protocol,
		Settings: req.Settings,
	}

	// Start the stream
	resp, err := h.streamService.StartStream(ctx, userID, startReq)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		if err == streamErrors.ErrStreamAlreadyLive {
			return nil, status.Error(codes.AlreadyExists, "stream is already live")
		}
		return nil, status.Errorf(codes.Internal, "failed to start stream: %v", err)
	}

	// Create response
	response := &routerpb.StartStreamResponse{
		StreamId:  resp.StreamID,
		InputUrl:  resp.InputURL,
		StreamKey: resp.StreamKey,
		BackupUrl: resp.BackupURL,
		ServerId:  resp.ServerID,
		BackupId:  resp.BackupID,
	}

	return response, nil
}

// StopStream handles stopping a stream
func (h *GRPCHandler) StopStream(ctx context.Context, req *routerpb.StopStreamRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Create stop request
	stopReq := model.StreamStopRequest{
		StreamID: req.StreamId,
		Force:    req.Force,
	}

	// Stop the stream
	err = h.streamService.StopStream(ctx, userID, stopReq)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		if err == streamErrors.ErrStreamNotLive {
			return nil, status.Error(codes.FailedPrecondition, "stream is not live")
		}
		return nil, status.Errorf(codes.Internal, "failed to stop stream: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetStreamEvents handles getting events for a stream
func (h *GRPCHandler) GetStreamEvents(ctx context.Context, req *routerpb.GetStreamEventsRequest) (*routerpb.GetStreamEventsResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Get stream events
	events, err := h.streamService.GetStreamEvents(ctx, userID, req.StreamId, int(req.Limit), int(req.Offset))
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to get stream events: %v", err)
	}

	// Convert events to protobuf
	protoEvents := make([]*routerpb.StreamEvent, 0, len(events))
	for _, event := range events {
		protoEvents = append(protoEvents, convertModelEventToProto(event))
	}

	// Create response
	response := &routerpb.GetStreamEventsResponse{
		Events: protoEvents,
		Count:  int32(len(events)),
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	return response, nil
}

// GetStreamMetadata handles getting metadata for a stream
func (h *GRPCHandler) GetStreamMetadata(ctx context.Context, req *routerpb.GetStreamMetadataRequest) (*routerpb.GetStreamMetadataResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Get stream to check ownership
	stream, err := h.streamService.GetByID(ctx, req.StreamId)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get stream: %v", err)
	}

	// Check if user has access to the stream
	if stream.UserID != userID && stream.Visibility != model.StreamVisibilityPublic {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	// Get metadata
	metadata, err := h.metadataService.GetMetadata(ctx, req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get metadata: %v", err)
	}

	// Create response
	response := &routerpb.GetStreamMetadataResponse{
		Metadata: metadata,
	}

	return response, nil
}

// SetStreamMetadata handles setting metadata for a stream
func (h *GRPCHandler) SetStreamMetadata(ctx context.Context, req *routerpb.SetStreamMetadataRequest) (*routerpb.GetStreamMetadataResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Set metadata
	err = h.metadataService.SetMetadata(ctx, userID, req.StreamId, req.Metadata)
	if err != nil {
		if err.Error() == "stream not found" {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err.Error() == "unauthorized access to stream" {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to set metadata: %v", err)
	}

	// Create response
	response := &routerpb.GetStreamMetadataResponse{
		Metadata: req.Metadata,
	}

	return response, nil
}

// UpdateStreamMetadata handles updating metadata for a stream
func (h *GRPCHandler) UpdateStreamMetadata(ctx context.Context, req *routerpb.UpdateStreamMetadataRequest) (*routerpb.GetStreamMetadataResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Update metadata
	err = h.metadataService.UpdateMetadata(ctx, userID, req.StreamId, req.Updates)
	if err != nil {
		if err.Error() == "stream not found" {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err.Error() == "unauthorized access to stream" {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to update metadata: %v", err)
	}

	// Get the updated metadata
	metadata, err := h.metadataService.GetMetadata(ctx, req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated metadata: %v", err)
	}

	// Create response
	response := &routerpb.GetStreamMetadataResponse{
		Metadata: metadata,
	}

	return response, nil
}

// DeleteStreamMetadataField handles deleting a metadata field for a stream
func (h *GRPCHandler) DeleteStreamMetadataField(ctx context.Context, req *routerpb.DeleteStreamMetadataFieldRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Delete metadata field
	err = h.metadataService.DeleteMetadataField(ctx, userID, req.StreamId, req.Key)
	if err != nil {
		if err.Error() == "stream not found" {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err.Error() == "unauthorized access to stream" {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		if err.Error() == "metadata field not found" {
			return nil, status.Error(codes.NotFound, "metadata field not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete metadata field: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// AddStreamOutput handles adding an output to a stream
func (h *GRPCHandler) AddStreamOutput(ctx context.Context, req *routerpb.AddStreamOutputRequest) (*routerpb.StreamOutput, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Convert protobuf output to model
	output := model.StreamOutput{
		ID:         req.Output.Id,
		Protocol:   req.Output.Protocol,
		URL:        req.Output.Url,
		EnabledFor: req.Output.EnabledFor,
		Format:     req.Output.Format,
		DRMEnabled: req.Output.DrmEnabled,
		CDNEnabled: req.Output.CdnEnabled,
	}

	// Add output
	err = h.streamService.AddStreamOutput(ctx, userID, req.StreamId, output)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to add output: %v", err)
	}

	return req.Output, nil
}

// RemoveStreamOutput handles removing an output from a stream
func (h *GRPCHandler) RemoveStreamOutput(ctx context.Context, req *routerpb.RemoveStreamOutputRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Remove output
	err = h.streamService.RemoveStreamOutput(ctx, userID, req.StreamId, req.OutputId)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove output: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// AddStreamEnhancement handles adding an enhancement to a stream
func (h *GRPCHandler) AddStreamEnhancement(ctx context.Context, req *routerpb.AddStreamEnhancementRequest) (*routerpb.StreamEnhancement, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Convert protobuf enhancement to model
	enhancement := model.StreamEnhancement{
		ID:       req.Enhancement.Id,
		Type:     req.Enhancement.Type,
		Enabled:  req.Enhancement.Enabled,
		Priority: int(req.Enhancement.Priority),
		Settings: req.Enhancement.Settings,
	}

	// Add enhancement
	err = h.streamService.AddStreamEnhancement(ctx, userID, req.StreamId, enhancement)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to add enhancement: %v", err)
	}

	return req.Enhancement, nil
}

// UpdateStreamEnhancement handles updating an enhancement for a stream
func (h *GRPCHandler) UpdateStreamEnhancement(ctx context.Context, req *routerpb.UpdateStreamEnhancementRequest) (*routerpb.StreamEnhancement, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Convert protobuf enhancement to model
	enhancement := model.StreamEnhancement{
		ID:       req.Enhancement.Id,
		Type:     req.Enhancement.Type,
		Enabled:  req.Enhancement.Enabled,
		Priority: int(req.Enhancement.Priority),
		Settings: req.Enhancement.Settings,
	}

	// Update enhancement
	err = h.streamService.UpdateStreamEnhancement(ctx, userID, req.StreamId, enhancement)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to update enhancement: %v", err)
	}

	return req.Enhancement, nil
}

// RemoveStreamEnhancement handles removing an enhancement from a stream
func (h *GRPCHandler) RemoveStreamEnhancement(ctx context.Context, req *routerpb.RemoveStreamEnhancementRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Remove enhancement
	err = h.streamService.RemoveStreamEnhancement(ctx, userID, req.StreamId, req.EnhancementId)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		if err == streamErrors.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove enhancement: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// RegisterWebhook handles registering a webhook
func (h *GRPCHandler) RegisterWebhook(ctx context.Context, req *routerpb.RegisterWebhookRequest) (*routerpb.RegisterWebhookResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Register webhook
	webhookID, err := h.notificationService.RegisterWebhook(ctx, userID, req.Url, req.Events, req.Secret, req.StreamId)
	if err != nil {
		if err.Error() == "webhook URL is required" {
			return nil, status.Error(codes.InvalidArgument, "URL is required")
		}
		if err.Error() == "unauthorized access to stream" {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to register webhook: %v", err)
	}

	// Create response
	response := &routerpb.RegisterWebhookResponse{
		WebhookId: webhookID,
		Url:       req.Url,
		Events:    req.Events,
		StreamId:  req.StreamId,
	}

	return response, nil
}

// UnregisterWebhook handles unregistering a webhook
func (h *GRPCHandler) UnregisterWebhook(ctx context.Context, req *routerpb.UnregisterWebhookRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Unregister webhook
	err = h.notificationService.UnregisterWebhook(ctx, userID, req.WebhookId)
	if err != nil {
		if err.Error() == "webhook not found" {
			return nil, status.Error(codes.NotFound, "webhook not found")
		}
		if err.Error() == "unauthorized access to webhook" {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to unregister webhook: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListWebhooks handles listing webhooks
func (h *GRPCHandler) ListWebhooks(ctx context.Context, req *routerpb.ListWebhooksRequest) (*routerpb.ListWebhooksResponse, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// List webhooks
	webhooks, err := h.notificationService.ListWebhooks(ctx, userID, req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks: %v", err)
	}

	// Convert webhooks to protobuf
	protoWebhooks := make([]*routerpb.Webhook, 0, len(webhooks))
	for _, webhook := range webhooks {
		protoWebhook := &routerpb.Webhook{
			Id:        webhook.ID,
			UserId:    webhook.UserID,
			StreamId:  webhook.StreamID,
			Url:       webhook.URL,
			Events:    webhook.Events,
			CreatedAt: timestamppb.New(webhook.CreatedAt),
		}
		protoWebhooks = append(protoWebhooks, protoWebhook)
	}

	// Create response
	response := &routerpb.ListWebhooksResponse{
		Webhooks: protoWebhooks,
		Count:    int32(len(webhooks)),
	}

	return response, nil
}

// TestWebhook handles testing a webhook
func (h *GRPCHandler) TestWebhook(ctx context.Context, req *routerpb.TestWebhookRequest) (*emptypb.Empty, error) {
	// Get user ID from the request metadata
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
	}

	// Test webhook
	err = h.notificationService.TestWebhook(ctx, userID, req.WebhookId)
	if err != nil {
		if err.Error() == "webhook not found" {
			return nil, status.Error(codes.NotFound, "webhook not found")
		}
		if err.Error() == "unauthorized access to webhook" {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		return nil, status.Errorf(codes.Internal, "failed to test webhook: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// UpdateStreamStatus handles updating a stream's status (used by internal services)
func (h *GRPCHandler) UpdateStreamStatus(ctx context.Context, req *routerpb.UpdateStreamStatusRequest) (*emptypb.Empty, error) {
	// This method can only be called by internal services, not by users
	// In a real implementation, we would check for an internal authentication token

	// Convert status
	streamStatus := model.StreamStatus(req.Status)

	// Update stream status
	err := h.streamService.UpdateStreamStatus(ctx, req.StreamId, streamStatus)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update stream status: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// UpdateStreamMetrics handles updating a stream's metrics (used by internal services)
func (h *GRPCHandler) UpdateStreamMetrics(ctx context.Context, req *routerpb.UpdateStreamMetricsRequest) (*emptypb.Empty, error) {
	// This method can only be called by internal services, not by users
	// In a real implementation, we would check for an internal authentication token

	// Convert metrics
	metrics := model.StreamMetrics{
		ViewerCount:     int(req.Metrics.ViewerCount),
		PeakViewerCount: int(req.Metrics.PeakViewerCount),
		IngestBitrate:   int(req.Metrics.IngestBitrate),
		OutputBitrate:   int(req.Metrics.OutputBitrate),
		FrameRate:       req.Metrics.FrameRate,
		Resolution:      req.Metrics.Resolution,
	}

	if req.Metrics.StartTime != nil {
		metrics.StartTime = req.Metrics.StartTime.AsTime()
	}
	if req.Metrics.EndTime != nil {
		metrics.EndTime = req.Metrics.EndTime.AsTime()
	}
	if req.Metrics.Duration != 0 {
		metrics.Duration = req.Metrics.Duration
	}

	// Update stream metrics
	err := h.streamService.UpdateStreamMetrics(ctx, req.StreamId, metrics)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update stream metrics: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ValidateStreamKey validates a stream key (used by ingestor service)
func (h *GRPCHandler) ValidateStreamKey(ctx context.Context, req *routerpb.ValidateStreamKeyRequest) (*routerpb.ValidateStreamKeyResponse, error) {
	// This method can only be called by internal services, not by users
	// In a real implementation, we would check for an internal authentication token

	// Validate stream key
	valid, err := h.streamService.ValidateStreamKey(ctx, req.StreamId, req.StreamKey)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			return nil, status.Error(codes.NotFound, "stream not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to validate stream key: %v", err)
	}

	// Create response
	response := &routerpb.ValidateStreamKeyResponse{
		Valid: valid,
	}

	return response, nil
}

// getUserIDFromContext extracts the user ID from the context
func getUserIDFromContext(ctx context.Context) (string, error) {
	// In a real implementation, this would extract user ID from authentication token
	// For simplicity, we'll use metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("no metadata in context")
	}

	userIDs := md.Get("x-user-id")
	if len(userIDs) == 0 {
		return "", errors.New("no user ID in metadata")
	}

	return userIDs[0], nil
}

// convertModelStreamToProto converts a model.Stream to routerpb.Stream
func convertModelStreamToProto(stream *model.Stream) *routerpb.Stream {
	// Convert inputs
	inputs := make([]*routerpb.StreamInput, 0, len(stream.Inputs))
	for _, input := range stream.Inputs {
		inputs = append(inputs, &routerpb.StreamInput{
			Id:       input.ID,
			Protocol: input.Protocol,
			Url:      input.URL,
			Backup:   input.Backup,
			Servers:  input.Servers,
		})
	}

	// Convert outputs
	outputs := make([]*routerpb.StreamOutput, 0, len(stream.Outputs))
	for _, output := range stream.Outputs {
		outputs = append(outputs, &routerpb.StreamOutput{
			Id:         output.ID,
			Protocol:   output.Protocol,
			Url:        output.URL,
			EnabledFor: output.EnabledFor,
			Format:     output.Format,
			DrmEnabled: output.DRMEnabled,
			CdnEnabled: output.CDNEnabled,
		})
	}

	// Convert enhancements
	enhancements := make([]*routerpb.StreamEnhancement, 0, len(stream.Enhancements))
	for _, enhancement := range stream.Enhancements {
		enhancements = append(enhancements, &routerpb.StreamEnhancement{
			Id:       enhancement.ID,
			Type:     enhancement.Type,
			Enabled:  enhancement.Enabled,
			Priority: int32(enhancement.Priority),
			Settings: enhancement.Settings,
		})
	}

	// Convert metrics
	metrics := &routerpb.StreamMetrics{
		ViewerCount:     int32(stream.Metrics.ViewerCount),
		PeakViewerCount: int32(stream.Metrics.PeakViewerCount),
		IngestBitrate:   int32(stream.Metrics.IngestBitrate),
		OutputBitrate:   int32(stream.Metrics.OutputBitrate),
		FrameRate:       stream.Metrics.FrameRate,
		Resolution:      stream.Metrics.Resolution,
		Duration:        stream.Metrics.Duration,
	}

	if !stream.Metrics.StartTime.IsZero() {
		metrics.StartTime = timestamppb.New(stream.Metrics.StartTime)
	}
	if !stream.Metrics.EndTime.IsZero() {
		metrics.EndTime = timestamppb.New(stream.Metrics.EndTime)
	}

	// Convert stream
	protoStream := &routerpb.Stream{
		Id:           stream.ID,
		UserId:       stream.UserID,
		Title:        stream.Title,
		Description:  stream.Description,
		Status:       string(stream.Status),
		Visibility:   string(stream.Visibility),
		Tags:         stream.Tags,
		Inputs:       inputs,
		Outputs:      outputs,
		Enhancements: enhancements,
		Metrics:      metrics,
		CreatedAt:    timestamppb.New(stream.CreatedAt),
		UpdatedAt:    timestamppb.New(stream.UpdatedAt),
		Recording:    stream.Recording,
		Thumbnail:    stream.Thumbnail,
		Region:       stream.Region,
		Metadata:     stream.Metadata,
	}

	return protoStream
}

// convertModelEventToProto converts a model.StreamEvent to routerpb.StreamEvent
func convertModelEventToProto(event *model.StreamEvent) *routerpb.StreamEvent {
	// Convert event data
	var data []byte
	if event.Data != nil {
		// In a real implementation, we would convert event.Data to protobuf
		// For simplicity, we'll just convert it to a string
		dataStr, ok := event.Data.(string)
		if !ok {
			dataStr = fmt.Sprintf("%v", event.Data)
		}
		data = []byte(dataStr)
	}

	// Convert event
	protoEvent := &routerpb.StreamEvent{
		Id:        event.ID,
		StreamId:  event.StreamID,
		Type:      event.Type,
		Timestamp: timestamppb.New(event.Timestamp),
		Data:      data,
	}

	return protoEvent
}
