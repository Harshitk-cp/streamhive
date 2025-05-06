package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/transport"
	webrtcpb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize WebRTC service
	webRTCService, err := service.NewWebRTCService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize WebRTC service: %v", err)
	}

	// Initialize frame receiver
	frameReceiver, err := transport.NewFrameReceiver(cfg.FrameSplitter.Address, webRTCService)
	if err != nil {
		log.Fatalf("Failed to initialize frame receiver: %v", err)
	}

	// Initialize HTTP handler
	httpHandler := handler.NewHTTPHandler(webRTCService)

	// Initialize HTTP server
	httpServer := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      httpHandler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	// Initialize gRPC handler
	grpcHandler := handler.NewGRPCHandler(webRTCService)

	// Set up gRPC server
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     cfg.GRPC.KeepAliveTime,
			MaxConnectionAge:      2 * cfg.GRPC.KeepAliveTime,
			MaxConnectionAgeGrace: cfg.GRPC.KeepAliveTimeout,
			Time:                  cfg.GRPC.KeepAliveTime,
			Timeout:               cfg.GRPC.KeepAliveTimeout,
		}),
		grpc.MaxConcurrentStreams(uint32(cfg.GRPC.MaxConcurrentStreams)),
	)

	// Register gRPC services
	webrtcpb.RegisterWebRTCServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", cfg.GRPC.Address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPC.Address, err)
	}

	go func() {
		log.Printf("Starting gRPC server on %s", cfg.GRPC.Address)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start WebRTC service
	webRTCService.Start(ctx)

	// Start frame receiver
	frameReceiver.Start(ctx)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// Stop WebRTC service
	webRTCService.Stop()

	// Stop frame receiver
	frameReceiver.Stop()

	// Gracefully shut down the HTTP server
	httpCtx, httpCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer httpCancel()
	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}

	// Gracefully shut down the gRPC server
	grpcServer.GracefulStop()

	log.Println("Servers successfully shut down")
}
