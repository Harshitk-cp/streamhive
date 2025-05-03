package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/transport"
	"github.com/Harshitk-cp/streamhive/apps/api-gateway/pkg/middleware"
	"github.com/Harshitk-cp/streamhive/libs/proto/signaling"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config/config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	proxyService := service.NewProxyService(cfg.Services)
	authService := service.NewAuthService(cfg.Auth)
	rateLimiter := service.NewRateLimiter(cfg.RateLimit)

	// Initialize context that listens for termination signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server (in its own goroutine)
	httpServer := setupHTTPServer(cfg, proxyService, authService, rateLimiter)
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start gRPC server (in its own goroutine)
	grpcServer, grpcListener := setupGRPCServer(cfg, authService, rateLimiter)
	go func() {
		log.Printf("Starting gRPC server on %s", cfg.GRPC.Address)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start WebSocket server (in its own goroutine)
	wsServer := setupWebSocketServer(cfg, authService, rateLimiter)
	go func() {
		log.Printf("Starting WebSocket server on %s", cfg.WebSocket.Address)
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WebSocket server failed: %v", err)
		}
	}()

	// Wait for termination signal
	<-signalChan
	log.Println("Received termination signal, initiating graceful shutdown...")

	// Graceful shutdown
	// First, the HTTP server
	httpCtx, httpCancel := context.WithTimeout(ctx, 30*time.Second)
	defer httpCancel()
	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	// Then, the WebSocket server
	wsCtx, wsCancel := context.WithTimeout(ctx, 30*time.Second)
	defer wsCancel()
	if err := wsServer.Shutdown(wsCtx); err != nil {
		log.Printf("WebSocket server forced to shutdown: %v", err)
	}

	// Finally, the gRPC server
	grpcServer.GracefulStop()

	log.Println("Shutdown complete")
}

func setupHTTPServer(
	cfg *config.Config,
	proxyService *service.ProxyService,
	authService *service.AuthService,
	rateLimiter *service.RateLimiter,
) *http.Server {
	// Create router and add middleware
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recovery)
	router.Use(middleware.Tracing)

	// Set up REST handlers
	restHandler := handler.NewRESTHandler(proxyService, authService)
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(middleware.Auth(authService))
	apiRouter.Use(middleware.RateLimit(rateLimiter))

	// Configure API routes
	apiRouter.HandleFunc("/v1/streams", restHandler.ListStreams).Methods("GET")
	apiRouter.HandleFunc("/v1/streams/{streamID}", restHandler.GetStream).Methods("GET")
	apiRouter.HandleFunc("/v1/streams", restHandler.CreateStream).Methods("POST")
	apiRouter.HandleFunc("/v1/streams/{streamID}", restHandler.UpdateStream).Methods("PUT")
	apiRouter.HandleFunc("/v1/streams/{streamID}", restHandler.DeleteStream).Methods("DELETE")

	// Add these new routes for stream operations
	apiRouter.HandleFunc("/v1/streams/{streamID}/start", restHandler.StartStream).Methods("POST")
	apiRouter.HandleFunc("/v1/streams/{streamID}/stop", restHandler.StopStream).Methods("POST")
	apiRouter.HandleFunc("/v1/streams/{streamID}/events", restHandler.GetStreamStats).Methods("GET")

	// Stream metadata routes
	apiRouter.HandleFunc("/v1/streams/{streamID}/metadata", restHandler.GetStreamMetadata).Methods("GET")
	apiRouter.HandleFunc("/v1/streams/{streamID}/metadata", restHandler.SetStreamMetadata).Methods("PUT")
	apiRouter.HandleFunc("/v1/streams/{streamID}/metadata", restHandler.UpdateStreamMetadata).Methods("PATCH")
	apiRouter.HandleFunc("/v1/streams/{streamID}/metadata/{key}", restHandler.DeleteMetadataField).Methods("DELETE")

	// Stream outputs and enhancements
	apiRouter.HandleFunc("/v1/streams/{streamID}/outputs", restHandler.AddOutput).Methods("POST")
	apiRouter.HandleFunc("/v1/streams/{streamID}/outputs/{outputID}", restHandler.RemoveOutput).Methods("DELETE")
	apiRouter.HandleFunc("/v1/streams/{streamID}/enhancements", restHandler.AddEnhancement).Methods("POST")
	apiRouter.HandleFunc("/v1/streams/{streamID}/enhancements/{enhancementID}", restHandler.UpdateEnhancement).Methods("PUT")
	apiRouter.HandleFunc("/v1/streams/{streamID}/enhancements/{enhancementID}", restHandler.RemoveEnhancement).Methods("DELETE")

	// Webhook routes
	apiRouter.HandleFunc("/v1/webhooks", restHandler.ListWebhooks).Methods("GET")
	apiRouter.HandleFunc("/v1/webhooks", restHandler.RegisterWebhook).Methods("POST")
	apiRouter.HandleFunc("/v1/webhooks/{webhookID}", restHandler.UnregisterWebhook).Methods("DELETE")
	apiRouter.HandleFunc("/v1/webhooks/{webhookID}/test", restHandler.TestWebhook).Methods("POST")

	// Health check doesn't need auth
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "API Gateway is healthy")
	}).Methods("GET")

	// Create and configure HTTP server
	return &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}
}

func setupGRPCServer(
	cfg *config.Config,
	authService *service.AuthService,
	rateLimiter *service.RateLimiter,
) (*grpc.Server, net.Listener) {
	// Create listeners
	lis, err := net.Listen("tcp", cfg.GRPC.Address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPC.Address, err)
	}

	// Set up gRPC interceptors (middleware)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(transport.ChainUnaryInterceptors(
			transport.LoggingInterceptor,
			transport.RecoveryInterceptor,
			transport.AuthInterceptor(authService),
			transport.RateLimitInterceptor(rateLimiter),
		)),
		grpc.StreamInterceptor(transport.ChainStreamInterceptors(
			transport.StreamLoggingInterceptor,
			transport.StreamRecoveryInterceptor,
			transport.StreamAuthInterceptor(authService),
			transport.StreamRateLimitInterceptor(rateLimiter),
		)),
	}

	// Create gRPC server with interceptors
	server := grpc.NewServer(opts...)

	// Register services
	signaling.RegisterSignalingServiceServer(server, handler.NewGRPCSignalingHandler())

	// Register reflection service (helpful for debugging)
	reflection.Register(server)

	return server, lis
}

func setupWebSocketServer(
	cfg *config.Config,
	authService *service.AuthService,
	rateLimiter *service.RateLimiter,
) *http.Server {
	// Create router for WebSocket
	router := mux.NewRouter()

	// Apply middleware specifically for WebSocket
	router.Use(middleware.Logger)
	router.Use(middleware.Recovery)
	router.Use(middleware.WebSocketAuth(authService))
	router.Use(middleware.RateLimit(rateLimiter))

	// Create WebSocket handler
	wsHandler := handler.NewWebSocketHandler(authService)
	router.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Create HTTP server for WebSocket
	return &http.Server{
		Addr:         cfg.WebSocket.Address,
		Handler:      router,
		ReadTimeout:  cfg.WebSocket.ReadTimeout,
		WriteTimeout: cfg.WebSocket.WriteTimeout,
		IdleTimeout:  cfg.WebSocket.IdleTimeout,
	}
}
