// apps/websocket-signaling/internal/handler/grpc.go
package handler

import (
	"context"
	"log"
	"net"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// GRPCServer represents the gRPC server
type GRPCServer struct {
	webrtcPb.UnimplementedSignalingServiceServer
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
	webrtcPb.RegisterSignalingServiceServer(server, grpcServer)

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

// RegisterStream registers a stream with the signaling service
func (s *GRPCServer) RegisterStream(ctx context.Context, req *webrtcPb.RegisterStreamRequest) (*webrtcPb.RegisterStreamResponse, error) {
	return s.service.RegisterStream(ctx, req)
}

// UnregisterStream unregisters a stream from the signaling service
func (s *GRPCServer) UnregisterStream(ctx context.Context, req *webrtcPb.UnregisterStreamRequest) (*webrtcPb.UnregisterStreamResponse, error) {
	return s.service.UnregisterStream(ctx, req)
}
