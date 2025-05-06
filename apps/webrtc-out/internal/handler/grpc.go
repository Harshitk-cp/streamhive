package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
	webrtcpb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCHandler handles gRPC requests
type GRPCHandler struct {
	webrtcpb.UnimplementedWebRTCServiceServer
	webrtcService *service.WebRTCService
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(webrtcService *service.WebRTCService) *GRPCHandler {
	return &GRPCHandler{
		webrtcService: webrtcService,
	}
}

// HandleOffer handles an SDP offer from a viewer
func (h *GRPCHandler) HandleOffer(ctx context.Context, req *webrtcpb.OfferRequest) (*webrtcpb.AnswerResponse, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream ID is required")
	}
	if req.ViewerId == "" {
		return nil, status.Error(codes.InvalidArgument, "viewer ID is required")
	}
	if req.Offer == "" {
		return nil, status.Error(codes.InvalidArgument, "SDP offer is required")
	}

	// Create offer
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  req.Offer,
	}

	// Handle offer
	answer, err := h.webrtcService.HandleOffer(req.StreamId, req.ViewerId, offer)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle offer: %v", err)
	}

	// Create response
	response := &webrtcpb.AnswerResponse{
		StreamId: req.StreamId,
		ViewerId: req.ViewerId,
		Answer:   answer.SDP,
	}

	return response, nil
}

// HandleICECandidate handles an ICE candidate from a viewer
func (h *GRPCHandler) HandleICECandidate(ctx context.Context, req *webrtcpb.ICECandidateRequest) (*emptypb.Empty, error) {
	// Validate request
	if req.ViewerId == "" {
		return nil, status.Error(codes.InvalidArgument, "viewer ID is required")
	}
	if req.Candidate == "" {
		return nil, status.Error(codes.InvalidArgument, "ICE candidate is required")
	}

	// Create ICE candidate
	candidate := webrtc.ICECandidateInit{
		Candidate:     req.Candidate,
		SDPMid:        req.SdpMid,
		SDPMLineIndex: req.SdpMLineIndex,
	}

	// Handle ICE candidate
	err := h.webrtcService.HandleICECandidate(req.ViewerId, candidate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle ICE candidate: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetStreamInfo gets information about a stream
func (h *GRPCHandler) GetStreamInfo(ctx context.Context, req *webrtcpb.GetStreamInfoRequest) (*webrtcpb.StreamInfo, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream ID is required")
	}

	// Get stream
	stream, err := h.webrtcService.GetStream(req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "stream not found: %v", err)
	}

	// Create response
	response := &webrtcpb.StreamInfo{
		StreamId:       stream.ID,
		Status:         string(stream.Status),
		CurrentViewers: int32(stream.CurrentViewers),
		TotalViewers:   int32(stream.TotalViewers),
		Width:          int32(stream.Width),
		Height:         int32(stream.Height),
		FrameRate:      float32(stream.FrameRate),
		TotalFrames:    stream.TotalFrames,
	}

	return response, nil
}

// RemoveStream removes a stream
func (h *GRPCHandler) RemoveStream(ctx context.Context, req *webrtcpb.RemoveStreamRequest) (*emptypb.Empty, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream ID is required")
	}

	// Remove stream
	err := h.webrtcService.RemoveStream(req.StreamId)
	if err != nil {
		if errors.Is(err, fmt.Errorf("stream not found: %s", req.StreamId)) {
			return nil, status.Errorf(codes.NotFound, "stream not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to remove stream: %v", err)
	}

	return &emptypb.Empty{}, nil
}
