// apps/frame-splitter/cmd/main.go

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

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/processor"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/transport"

	framepb "github.com/Harshitk-cp/streamhive/libs/proto/frame"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {

	// Parse command line flags
	configPath := flag.String("config", "./config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from %s", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize metrics collector
	log.Println("Initializing metrics collector")
	metricsCollector := metrics.NewPrometheusCollector()

	// Initialize frame processor
	log.Println("Initializing frame processor")
	frameProcessor, err := processor.NewFrameProcessor(cfg, metricsCollector)
	if err != nil {
		log.Fatalf("Failed to initialize frame processor: %v", err)
	}

	// Start the frame processor
	log.Println("Starting frame processor")
	go frameProcessor.Start(ctx)

	// Initialize HTTP handler
	log.Println("Initializing HTTP handler")
	httpHandler := handler.NewHTTPHandler(frameProcessor, metricsCollector)

	// Initialize HTTP server
	log.Println("Initializing HTTP server")
	httpServer := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      httpHandler,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Initialize gRPC handler
	log.Println("Initializing gRPC handler")
	grpcHandler := handler.NewGRPCHandler(frameProcessor)

	// Initialize gRPC server
	log.Println("Initializing gRPC server")
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
	log.Println("Registering gRPC services")
	framepb.RegisterFrameSplitterServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

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

	// Setup connections to other services
	log.Println("Setting up connections to other services")

	// Connect to router service
	log.Printf("Connecting to router service at %s", cfg.Router.Address)
	router, err := transport.NewRouterClient(cfg.Router.Address)
	if err != nil {
		log.Printf("Warning: Failed to connect to router service: %v", err)
	} else {
		defer router.Close()
		log.Println("Successfully connected to router service")
	}

	// Connect to enhancement service if enabled
	if cfg.Routing.EnhancementService.Enabled {
		log.Printf("Connecting to enhancement service at %s", cfg.Routing.EnhancementService.Address)
		enhancementClient, err := transport.NewEnhancementClient(cfg.Routing.EnhancementService.Address)
		if err != nil {
			log.Printf("Warning: Failed to connect to enhancement service: %v", err)
		} else {
			defer enhancementClient.Close()
			frameProcessor.RegisterRoute("enhancement", enhancementClient)
			log.Println("Successfully connected to enhancement service")
		}
	}

	// Connect to encoder service if enabled
	if cfg.Routing.EncoderService.Enabled {
		log.Printf("Connecting to encoder service at %s", cfg.Routing.EncoderService.Address)
		encoderClient, err := transport.NewEncoderClient(cfg.Routing.EncoderService.Address)
		if err != nil {
			log.Printf("Warning: Failed to connect to encoder service: %v", err)
		} else {
			defer encoderClient.Close()
			frameProcessor.RegisterRoute("encoder", encoderClient)
			log.Println("Successfully connected to encoder service")
		}
	}

	// Connect to WebRTC service if enabled
	// Connect to WebRTC service if enabled
	if cfg.Routing.WebRTCOut.Enabled {
		log.Printf("Connecting to WebRTC service at %s", cfg.Routing.WebRTCOut.Address)
		webrtcClient, err := transport.NewWebRTCClient(cfg.Routing.WebRTCOut.Address)
		if err != nil {
			log.Printf("Warning: Failed to connect to WebRTC service: %v", err)
		} else {
			defer webrtcClient.Close()
			frameProcessor.RegisterRoute("webrtc", webrtcClient)
			log.Println("Successfully connected to WebRTC service")
		}
	}

	log.Printf("Frame Splitter service started successfully (version: %s, build: %s)", version, commit)

	// Wait for signal to shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutdown signal received")

	// Stop the frame processor
	log.Println("Stopping frame processor")
	frameProcessor.Stop()

	// Gracefully stop the gRPC server
	log.Println("Stopping gRPC server")
	grpcServer.GracefulStop()

	// Gracefully shut down the HTTP server
	log.Println("Stopping HTTP server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}

	log.Println("Servers successfully shut down")
}
