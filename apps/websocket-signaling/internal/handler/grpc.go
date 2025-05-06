package handler

import (
	"context"
	"log"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	signalpb "github.com/Harshitk-cp/streamhive/libs/proto/signaling"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCSignalingHandler handles gRPC requests for signaling
type GRPCSignalingHandler struct {
	signalpb.UnimplementedSignalingServiceServer
	signalingService *service.SignalingService
}

// NewGRPCSignalingHandler creates a new gRPC handler for signaling
func NewGRPCSignalingHandler(signalingService *service.SignalingService) *GRPCSignalingHandler {
	return &GRPCSignalingHandler{
		signalingService: signalingService,
	}
}

// SendSignalingMessage sends a signaling message
func (h *GRPCSignalingHandler) SendSignalingMessage(ctx context.Context, req *signalpb.SignalingMessage) (*emptypb.Empty, error) {
	// Convert protobuf message to internal model
	msg := model.SignalingMessage{
		Type:        req.Type,
		StreamID:    req.StreamId,
		SenderID:    req.SenderId,
		RecipientID: req.RecipientId,
		Payload:     []byte(req.Payload),
		Timestamp:   req.Timestamp,
	}

	// Process message
	h.signalingService.HandleMessage(msg)

	return &emptypb.Empty{}, nil
}

// GetStreamInfo gets information about a stream
func (h *GRPCSignalingHandler) GetStreamInfo(ctx context.Context, req *signalpb.GetStreamInfoRequest) (*signalpb.StreamInfo, error) {
	// Get client count
	clientCount := h.signalingService.GetClientCount(req.StreamId)

	// Create response
	response := &signalpb.StreamInfo{
		StreamId:    req.StreamId,
		ClientCount: int32(clientCount),
		Active:      clientCount > 0,
		CreatedAt:   timestamppb.Now(),
	}

	return response, nil
}

// NotifyStreamEnded notifies clients that a stream has ended
func (h *GRPCSignalingHandler) NotifyStreamEnded(ctx context.Context, req *signalpb.NotifyStreamEndedRequest) (*emptypb.Empty, error) {
	// Create message
	msg := model.SignalingMessage{
		Type:     model.MessageTypeStreamEnded,
		StreamID: req.StreamId,
		SenderID: "server",
		Payload:  []byte(`{"reason":"` + req.Reason + `"}`),
	}

	// Process message
	h.signalingService.HandleMessage(msg)

	log.Printf("Notified stream ended: %s", req.StreamId)
	return &emptypb.Empty{}, nil
}

// GetStats gets statistics about the signaling service
func (h *GRPCSignalingHandler) GetStats(ctx context.Context, req *emptypb.Empty) (*signalpb.SignalingStats, error) {
	// Get total client count
	totalClients := h.signalingService.GetTotalClientCount()

	// Create response
	response := &signalpb.SignalingStats{
		ConnectedClients: int32(totalClients),
		Timestamp:        timestamppb.Now(),
	}

	return response, nil
}

// StreamSignaling establishes a bidirectional signaling stream
func (h *GRPCSignalingHandler) StreamSignaling(stream signalpb.SignalingService_StreamSignalingServer) error {
	// This method would be used for services that need bidirectional communication
	// For simplicity, we'll just return unimplemented for now
	return status.Errorf(codes.Unimplemented, "method StreamSignaling not implemented")
}
