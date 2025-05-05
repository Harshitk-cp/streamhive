// apps/websocket-signaling/cmd/server/main.go
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/handler"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create metrics collector
	metricsCollector := metrics.NewPrometheusCollector()

	// Create signaling service
	signalingService := service.New(cfg, metricsCollector)

	// Create gRPC server
	grpcServer := handler.NewGRPCServer(cfg, signalingService)

	// Create HTTP server
	httpServer := handler.NewHTTPServer(cfg, signalingService, metricsCollector)

	// Start servers
	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for termination signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh

	// Shutdown servers
	log.Println("Shutting down servers...")
	grpcServer.Stop()
	if err := httpServer.Stop(); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}
}
