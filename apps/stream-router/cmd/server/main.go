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
	"time"

	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/repository/memory"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/service"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/transport"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/pkg/middleware"
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

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize repositories
	streamRepo := memory.NewMemoryStreamRepository()
	eventRepo := memory.NewMemoryEventRepository()

	// Initialize services
	streamService := service.NewStreamService(streamRepo, eventRepo)
	metadataService := service.NewMetadataService(streamRepo, eventRepo)
	notificationService := service.NewNotificationService(eventRepo, streamRepo)

	// Start periodic cache cleanup
	metadataService.PeriodicallyCleanCache(5 * time.Minute)

	// Initialize HTTP handlers
	httpHandler := handler.NewHTTPHandler(streamService, metadataService, notificationService)

	// Set up HTTP server
	httpServer := transport.NewHTTPServer(cfg.HTTP.Address, httpHandler)

	// Apply middleware
	httpServer.Use(middleware.Logging)
	httpServer.Use(middleware.Recovery)
	httpServer.Use(middleware.Metrics)

	// Initialize gRPC server
	grpcHandler := handler.NewGRPCHandler(streamService, metadataService, notificationService)
	grpcServer := transport.NewGRPCServer(grpcHandler)

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start gRPC server
	go func() {
		listener, err := net.Listen("tcp", cfg.GRPC.Address)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", cfg.GRPC.Address, err)
		}
		log.Printf("Starting gRPC server on %s", cfg.GRPC.Address)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Stop notification service
	notificationService.Stop()

	log.Println("Servers successfully shutdown")
}
