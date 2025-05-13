package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/model"
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
	log.Printf("Received offer request for stream %s from viewer %s", req.StreamId, req.ViewerId)

	// Validate that we have a proper SDP
	if req.Offer == "" {
		return nil, status.Error(codes.InvalidArgument, "offer SDP is empty")
	}

	// Clean and prepare the SDP
	offerSDP := req.Offer

	// Log SDP size
	log.Printf("Processing offer with length: %d bytes", len(offerSDP))

	// Ensure SDP is properly formatted and validate it
	offerSDP = strings.TrimSpace(offerSDP)
	if !strings.HasPrefix(offerSDP, "v=0") {
		// If SDP doesn't start with v=0, try to extract it with improved methods
		log.Printf("SDP doesn't start with v=0, trying to fix it")

		// Try various approaches in sequence
		fixed := false

		// Try to unescape JSON strings
		if !fixed {
			unescaped, err := strconv.Unquote("\"" + offerSDP + "\"")
			if err == nil && strings.HasPrefix(strings.TrimSpace(unescaped), "v=0") {
				offerSDP = unescaped
				fixed = true
				log.Printf("Fixed SDP using JSON unescaping")
			}
		}

		// Try to extract from JSON if it looks like JSON
		if !fixed && strings.HasPrefix(offerSDP, "{") {
			var sdpObj map[string]interface{}
			if err := json.Unmarshal([]byte(offerSDP), &sdpObj); err == nil {
				if sdp, ok := sdpObj["sdp"].(string); ok && strings.HasPrefix(strings.TrimSpace(sdp), "v=0") {
					offerSDP = sdp
					fixed = true
					log.Printf("Extracted SDP from JSON object")
				}
			}
		}

		if !fixed {
			return nil, status.Errorf(codes.InvalidArgument, "failed to process offer: invalid SDP format")
		}
	}

	// Create session description with validated SDP
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	}

	// Double-check that SDP has required sections
	if !strings.Contains(offerSDP, "m=audio") && !strings.Contains(offerSDP, "m=video") {
		log.Printf("SDP missing required media sections")
		return nil, status.Error(codes.InvalidArgument, "SDP missing required media sections")
	}

	log.Printf("Handling WebRTC offer for stream %s from viewer %s", req.StreamId, req.ViewerId)

	// Process the offer using the webrtc service
	answer, err := h.webrtcService.HandleOffer(req.StreamId, req.ViewerId, offer)
	if err != nil {
		log.Printf("Error handling offer: %v", err)
		return nil, fmt.Errorf("failed to handle offer: %w", err)
	}

	// Log success
	log.Printf("Created answer for stream %s, viewer %s, answer length: %d bytes",
		req.StreamId, req.ViewerId, len(answer.SDP))

	// Return response
	return &webrtcpb.AnswerResponse{
		StreamId: req.StreamId,
		ViewerId: req.ViewerId,
		Answer:   answer.SDP,
	}, nil
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
		Candidate: req.Candidate,
		SDPMid:    &req.SdpMid,
		SDPMLineIndex: func(v uint32) *uint16 {
			val := uint16(v)
			return &val
		}(req.SdpMlineIndex),
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

// PushFrame pushes a frame to a stream
func (h *GRPCHandler) PushFrame(ctx context.Context, req *webrtcpb.PushFrameRequest) (*webrtcpb.PushFrameResponse, error) {
	// Validate request
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}
	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "frame data is required")
	}

	// Convert frame type
	var frameType model.FrameType
	switch req.Type {
	case webrtcpb.FrameType_VIDEO:
		frameType = model.FrameTypeVideo
	case webrtcpb.FrameType_AUDIO:
		frameType = model.FrameTypeAudio
	case webrtcpb.FrameType_METADATA:
		frameType = model.FrameTypeMetadata
	default:
		frameType = model.FrameTypeVideo // Default to video
	}

	// Convert timestamp
	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = req.Timestamp.AsTime()
	}

	// Create frame
	frame := &model.Frame{
		StreamID:   req.StreamId,
		FrameID:    req.FrameId,
		Type:       frameType,
		Data:       req.Data,
		Timestamp:  timestamp,
		IsKeyFrame: req.IsKeyFrame,
		Sequence:   req.Sequence,
		Metadata:   req.Metadata,
	}

	// Push frame to WebRTC service
	err := h.webrtcService.PushFrame(frame)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push frame: %v", err)
	}

	return &webrtcpb.PushFrameResponse{
		Status: "success",
	}, nil
}
