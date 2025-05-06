package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/hub"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("Starting WebSocket signaling service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create hub
	h := hub.NewHub()
	go h.Run()

	// Create signaling service
	signalingService, err := service.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create signaling service: %v", err)
	}

	// Create WebSocket handler
	wsHandler := handler.NewWebSocketHandler(h, signalingService)

	// Create HTTP handler
	httpHandler := handler.NewHTTPHandler(signalingService)

	// Create HTTP router
	router := mux.NewRouter()
	router.HandleFunc("/ws", wsHandler.HandleWebSocket)
	router.HandleFunc("/stream", wsHandler.HandleStreamInfo)
	httpHandler.SetupRoutes(router)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	webrtcPb.RegisterSignalingServiceServer(grpcServer, signalingService)
	reflection.Register(grpcServer)

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on %s", cfg.GRPC.Address)
		lis, err := net.Listen("tcp", cfg.GRPC.Address)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", cfg.GRPC.Address, err)
		}
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	// Close signaling service
	if err := signalingService.Close(); err != nil {
		log.Printf("Signaling service close error: %v", err)
	}

	log.Println("Servers shutdown complete")
}
