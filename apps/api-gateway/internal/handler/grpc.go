package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/Harshitk-cp/streamhive/libs/proto/signaling"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCSignalingHandler handles gRPC signaling requests
type GRPCSignalingHandler struct {
	signaling.UnimplementedSignalingServiceServer
}

// NewGRPCSignalingHandler creates a new gRPC signaling handler
func NewGRPCSignalingHandler() *GRPCSignalingHandler {
	return &GRPCSignalingHandler{}
}

// SendSignal processes WebRTC signaling requests
func (h *GRPCSignalingHandler) SendSignal(ctx context.Context, req *signaling.SignalRequest) (*signaling.SignalResponse, error) {
	// Validate request
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	if req.Payload == "" {
		return nil, status.Error(codes.InvalidArgument, "payload is required")
	}

	// Log the signal for debugging
	log.Printf("Received signal request: SessionID=%s, Payload length=%d", req.SessionId, len(req.Payload))

	// TODO: Forward the signal to the websocket-signaling service
	// This would involve making a gRPC call to the websocket-signaling service
	// For now, we'll just acknowledge receipt

	return &signaling.SignalResponse{
		Status: "received",
	}, nil
}

func (h *GRPCSignalingHandler) WebRTCConnect(ctx context.Context, req *signaling.WebRTCConnectRequest) (*signaling.WebRTCConnectResponse, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}

	// Log the connection request
	log.Printf("Received WebRTC connect request: StreamID=%s, UserID=%s", req.StreamId, req.UserId)

	// Forward to the webrtc-out service
	// This would involve making a gRPC call to the webrtc-out service
	// For now, we'll just return a mock response

	return &signaling.WebRTCConnectResponse{
		SessionId: fmt.Sprintf("session_%s_%s", req.StreamId, req.UserId),
		Status:    "connected",
		IceServers: []*signaling.ICEServer{
			{
				Urls:       []string{"stun:stun.l.google.com:19302"},
				Username:   "",
				Credential: "",
			},
		},
	}, nil
}

func (h *GRPCSignalingHandler) WebRTCDisconnect(ctx context.Context, req *signaling.WebRTCDisconnectRequest) (*signaling.WebRTCDisconnectResponse, error) {
	// Validate request
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	// Log the disconnection request
	log.Printf("Received WebRTC disconnect request: SessionID=%s", req.SessionId)

	// Forward to the webrtc-out service
	// This would involve making a gRPC call to the webrtc-out service
	// For now, we'll just acknowledge the disconnection

	return &signaling.WebRTCDisconnectResponse{
		Status: "disconnected",
	}, nil
}
