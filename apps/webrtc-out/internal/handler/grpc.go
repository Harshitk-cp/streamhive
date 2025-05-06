package handler

import (
	"log"
	"net"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// GRPCServer represents the gRPC server
type GRPCServer struct {
	webrtcPb.UnimplementedWebRTCServiceServer
	config  *config.Config
	service *service.Service
	server  *grpc.Server
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(cfg *config.Config, svc *service.Service) *GRPCServer {
	// Create gRPC server with keepalive options
	server := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     cfg.GRPC.KeepAliveTime,
			MaxConnectionAge:      cfg.GRPC.ConnectionTimeout,
			MaxConnectionAgeGrace: cfg.GRPC.KeepAliveTimeout,
			Time:                  cfg.GRPC.KeepAliveTime,
			Timeout:               cfg.GRPC.KeepAliveTimeout,
		}),
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentStreams)),
	)

	grpcServer := &GRPCServer{
		config:  cfg,
		service: svc,
		server:  server,
	}

	// Register services
	webrtcPb.RegisterWebRTCServiceServer(server, grpcServer)

	return grpcServer
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	// Create listener
	lis, err := net.Listen("tcp", s.config.GRPC.Address)
	if err != nil {
		return err
	}

	// Start server
	log.Printf("Starting gRPC server on %s", s.config.GRPC.Address)
	return s.server.Serve(lis)
}

// Stop stops the gRPC server
func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}

// // CreateStream creates a new stream
// func (s *GRPCServer) CreateStream(ctx context.Context, req *webrtcPb.CreateStreamRequest) (*webrtcPb.CreateStreamResponse, error) {
// 	return s.service.CreateStream(ctx, req)
// }

// // DestroyStream destroys a stream
// func (s *GRPCServer) DestroyStream(ctx context.Context, req *webrtcPb.DestroyStreamRequest) (*webrtcPb.DestroyStreamResponse, error) {
// 	return s.service.DestroyStream(ctx, req)
// }

// // AddVideoFrame adds a video frame to a stream
// func (s *GRPCServer) AddVideoFrame(ctx context.Context, req *webrtcPb.AddVideoFrameRequest) (*webrtcPb.AddVideoFrameResponse, error) {
// 	return s.service.AddVideoFrame(ctx, req)
// }

// // AddAudioFrame adds an audio frame to a stream
// func (s *GRPCServer) AddAudioFrame(ctx context.Context, req *webrtcPb.AddAudioFrameRequest) (*webrtcPb.AddAudioFrameResponse, error) {
// 	return s.service.AddAudioFrame(ctx, req)
// }

// // GetStreamStats gets stats for a stream
// func (s *GRPCServer) GetStreamStats(ctx context.Context, req *webrtcPb.GetStreamStatsRequest) (*webrtcPb.GetStreamStatsResponse, error) {
// 	return s.service.GetStreamStats(ctx, req)
// }

// // HandleOffer handles a SDP offer from a viewer
// func (s *GRPCServer) HandleOffer(ctx context.Context, req *webrtcPb.SDPOfferRequest) (*webrtcPb.SDPAnswerResponse, error) {
// 	return s.service.HandleOffer(ctx, req)
// }

// // HandleICECandidate handles an ICE candidate from a viewer
// func (s *GRPCServer) HandleICECandidate(ctx context.Context, req *webrtcPb.ICECandidateRequest) (*webrtcPb.ICECandidateResponse, error) {
// 	return s.service.HandleICECandidate(ctx, req)
// }
