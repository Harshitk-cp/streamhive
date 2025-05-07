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

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	signalpb "github.com/Harshitk-cp/streamhive/libs/proto/signaling"

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

	// Initialize signaling service
	signalingService := service.NewSignalingService(cfg)

	// Initialize HTTP handler
	httpHandler := handler.NewHTTPHandler(signalingService)

	// Initialize WebSocket handler
	wsHandler := handler.NewWebSocketHandler(signalingService)

	// Create router for WebSocket
	wsRouter := http.NewServeMux()
	wsRouter.Handle(cfg.WebSocket.Path, wsHandler)

	// Initialize HTTP server for REST API and health checks
	httpServer := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      httpHandler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	// Initialize WebSocket server
	wsServer := &http.Server{
		Addr:         cfg.WebSocket.Address,
		Handler:      wsRouter,
		ReadTimeout:  cfg.WebSocket.ReadTimeout,
		WriteTimeout: cfg.WebSocket.WriteTimeout,
	}

	// Initialize gRPC handler
	grpcHandler := handler.NewGRPCSignalingHandler(signalingService)

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
	signalpb.RegisterSignalingServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start WebSocket server
	go func() {
		log.Printf("Starting WebSocket server on %s%s", cfg.WebSocket.Address, cfg.WebSocket.Path)
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WebSocket server failed: %v", err)
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

	// Start signaling service
	signalingService.Start(ctx)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// Stop signaling service
	signalingService.Stop()

	// Gracefully shut down the HTTP server
	httpCtx, httpCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer httpCancel()
	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}

	// Gracefully shut down the WebSocket server
	wsCtx, wsCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer wsCancel()
	if err := wsServer.Shutdown(wsCtx); err != nil {
		log.Fatalf("WebSocket server shutdown failed: %v", err)
	}

	// Gracefully shut down the gRPC server
	grpcServer.GracefulStop()

	log.Println("Servers successfully shut down")
}
