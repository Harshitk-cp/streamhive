package transport

import (
	"log"
	"net"

	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/handler"
	routerpb "github.com/Harshitk-cp/streamhive/libs/proto/stream"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer represents a gRPC server
type GRPCServer struct {
	server     *grpc.Server
	handler    *handler.GRPCHandler
	middleware []grpc.UnaryServerInterceptor
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(handler *handler.GRPCHandler) *GRPCServer {
	return &GRPCServer{
		handler:    handler,
		middleware: make([]grpc.UnaryServerInterceptor, 0),
	}
}

// Use adds middleware to the server
func (s *GRPCServer) Use(middleware grpc.UnaryServerInterceptor) {
	s.middleware = append(s.middleware, middleware)
}

// Serve starts the gRPC server
func (s *GRPCServer) Serve(listener net.Listener) error {
	// Add default middleware
	defaultMiddleware := []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_recovery.UnaryServerInterceptor(),
	}

	// Combine middleware
	allMiddleware := append(defaultMiddleware, s.middleware...)

	// Create server with middleware
	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(allMiddleware...)),
	)

	// Register services
	routerpb.RegisterStreamServiceServer(s.server, s.handler)

	// Register reflection service for gRPC CLI and debugging
	reflection.Register(s.server)

	// Serve
	log.Printf("Starting gRPC server on %s", listener.Addr().String())
	return s.server.Serve(listener)
}

// GracefulStop gracefully stops the server
func (s *GRPCServer) GracefulStop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// Stop stops the server immediately
func (s *GRPCServer) Stop() {
	if s.server != nil {
		s.server.Stop()
	}
}
