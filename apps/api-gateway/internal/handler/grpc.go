package handler

import (
	"context"
	"log"

	signalpb "github.com/Harshitk-cp/streamhive/libs/proto/signaling"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCSignalingHandler handles gRPC signaling requests
type GRPCSignalingHandler struct {
	signalpb.UnimplementedSignalingServiceServer
}

// NewGRPCSignalingHandler creates a new gRPC signaling handler
func NewGRPCSignalingHandler() *GRPCSignalingHandler {
	return &GRPCSignalingHandler{}
}

// SendSignalingMessage processes WebRTC signaling messages
func (h *GRPCSignalingHandler) SendSignalingMessage(ctx context.Context, req *signalpb.SignalingMessage) (*emptypb.Empty, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}
	if req.SenderId == "" {
		return nil, status.Error(codes.InvalidArgument, "sender_id is required")
	}

	// Log the message for debugging
	log.Printf("Received signaling message: Type=%s, StreamID=%s, SenderID=%s", req.Type, req.StreamId, req.SenderId)

	// In a real implementation, we would forward this to the appropriate service
	// For now, we'll just acknowledge receipt
	return &emptypb.Empty{}, nil
}

// GetStreamInfo gets information about a stream
func (h *GRPCSignalingHandler) GetStreamInfo(ctx context.Context, req *signalpb.GetStreamInfoRequest) (*signalpb.StreamInfo, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}

	// Log the request
	log.Printf("Received stream info request: StreamID=%s", req.StreamId)

	// In a real implementation, we would get this from the stream-router service
	// For now, we'll just return a mock response
	return &signalpb.StreamInfo{
		StreamId:    req.StreamId,
		ClientCount: 0,
		Active:      false,
	}, nil
}

// NotifyStreamEnded notifies clients that a stream has ended
func (h *GRPCSignalingHandler) NotifyStreamEnded(ctx context.Context, req *signalpb.NotifyStreamEndedRequest) (*emptypb.Empty, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}

	// Log the request
	log.Printf("Received stream ended notification: StreamID=%s, Reason=%s", req.StreamId, req.Reason)

	// In a real implementation, we would notify the signaling service
	// For now, we'll just acknowledge receipt
	return &emptypb.Empty{}, nil
}

// GetStats gets statistics about the signaling service
func (h *GRPCSignalingHandler) GetStats(ctx context.Context, req *emptypb.Empty) (*signalpb.SignalingStats, error) {
	// Log the request
	log.Printf("Received stats request")

	// In a real implementation, we would get this from the signaling service
	// For now, we'll just return a mock response
	return &signalpb.SignalingStats{
		ConnectedClients: 0,
	}, nil
}

// StreamSignaling establishes a bidirectional signaling stream
func (h *GRPCSignalingHandler) StreamSignaling(stream signalpb.SignalingService_StreamSignalingServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamSignaling not implemented")
}
