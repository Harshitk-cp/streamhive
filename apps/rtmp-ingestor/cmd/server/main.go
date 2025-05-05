package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/ingest"
	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/router"
	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/server"
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

	// Initialize metrics collector
	metricsCollector := metrics.NewPrometheusCollector()

	// Initialize router client to communicate with stream-router service
	routerClient, err := router.NewGRPCClient(cfg.Router.Address)
	if err != nil {
		log.Fatalf("Failed to create router client: %v", err)
	}
	defer routerClient.Close()

	// Initialize stream analyzer
	streamAnalyzer := ingest.NewStreamAnalyzer(routerClient, metricsCollector)
	streamAnalyzer.Start()

	// Convert the config struct to the expected RTMPConfig type
	rtmpConfig := ingest.RTMPConfig{
		Address:          cfg.RTMP.Address,
		ChunkSize:        cfg.RTMP.ChunkSize,
		BufferSize:       cfg.RTMP.BufferSize,
		ReadTimeout:      cfg.RTMP.ReadTimeout,
		WriteTimeout:     cfg.RTMP.WriteTimeout,
		GopCacheEnabled:  cfg.RTMP.GopCacheEnabled,
		GopCacheMaxItems: cfg.RTMP.GopCacheMaxItems,
		KeyFrameOnly:     cfg.RTMP.KeyFrameOnly,
	}

	// Initialize RTMP ingestor with the converted config
	frameSplitterAddr := cfg.FrameSplitter.Address
	// Initialize RTMP ingestor with frame forwarder and stream analyzer
	rtmpIngestor, err := ingest.NewRTMPIngestor(
		rtmpConfig,
		routerClient,
		metricsCollector,
		frameSplitterAddr,
		streamAnalyzer,
	)
	if err != nil {
		log.Fatalf("Failed to create RTMP ingestor: %v", err)
	}

	// Initialize HTTP server for metrics and health checks
	httpServer := server.NewHTTPServer(cfg.HTTP.Address, metricsCollector)

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTP.Address)
		if err := httpServer.Start(); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start RTMP ingestor
	go func() {
		log.Printf("Starting RTMP ingestor on %s", cfg.RTMP.Address)
		if err := rtmpIngestor.Start(ctx); err != nil {
			log.Fatalf("RTMP ingestor failed: %v", err)
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

	// Stop RTMP ingestor
	rtmpIngestor.Stop()

	streamAnalyzer.Stop()

	log.Println("Servers successfully shutdown")
}
