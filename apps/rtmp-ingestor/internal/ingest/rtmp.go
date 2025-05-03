package ingest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/router"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/rtmp"
)

// RTMPIngestor handles RTMP stream ingestion
type RTMPIngestor struct {
	config           RTMPConfig
	routerClient     router.Client
	metricsCollector metrics.Collector
	server           *rtmp.Server
	activeStreams    map[string]*StreamInfo
	streamsMutex     sync.RWMutex
	shutdownCh       chan struct{}
}

// RTMPConfig contains RTMP server configuration
type RTMPConfig struct {
	Address          string
	ChunkSize        int
	BufferSize       int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	GopCacheEnabled  bool
	GopCacheMaxItems int
	KeyFrameOnly     bool
}

// StreamInfo contains information about an active stream
type StreamInfo struct {
	StreamID      string
	StartTime     time.Time
	FrameCount    int64
	BytesReceived int64
	Width         int
	Height        int
	FrameRate     float64
	VideoBitrate  int
	AudioBitrate  int
	Codec         string
	IsBackup      bool
	LastActivity  time.Time
	Mutex         sync.Mutex
	// Simple cache for GOP (Group of Pictures)
	gopCache        []av.Packet
	maxGopItems     int
	gopCacheEnabled bool
}

// NewRTMPIngestor creates a new RTMP ingestor
func NewRTMPIngestor(config RTMPConfig, routerClient router.Client, metricsCollector metrics.Collector) (*RTMPIngestor, error) {
	ingestor := &RTMPIngestor{
		config:           config,
		routerClient:     routerClient,
		metricsCollector: metricsCollector,
		activeStreams:    make(map[string]*StreamInfo),
		shutdownCh:       make(chan struct{}),
	}

	// Create RTMP server
	ingestor.server = &rtmp.Server{
		Addr:          config.Address,
		HandlePublish: ingestor.handlePublish,
	}

	return ingestor, nil
}

// Start starts the RTMP ingestor
func (i *RTMPIngestor) Start(ctx context.Context) error {
	// Start monitoring active streams
	go i.monitorActiveStreams(ctx)

	// Create a goroutine to listen for shutdown
	go func() {
		select {
		case <-i.shutdownCh:
			// Server will be stopped when ListenAndServe returns
			log.Println("Shutdown signal received, stopping RTMP ingestor")
		case <-ctx.Done():
			// Context was cancelled
			log.Println("Context cancelled, stopping RTMP ingestor")
			i.shutdownCh <- struct{}{}
		}
	}()

	// Start RTMP server - this is a blocking call
	log.Printf("Starting RTMP server on %s", i.config.Address)
	if err := i.server.ListenAndServe(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("failed to start RTMP server: %w", err)
	}

	return nil
}

// Stop stops the RTMP ingestor
func (i *RTMPIngestor) Stop() {
	log.Println("Stopping RTMP ingestor")

	// Signal all goroutines to stop
	close(i.shutdownCh)

	// Disconnect all clients
	i.streamsMutex.Lock()
	activeStreams := make(map[string]*StreamInfo)
	for id, info := range i.activeStreams {
		activeStreams[id] = info
	}
	i.streamsMutex.Unlock()

	// Update stream status for all active streams
	for id, _ := range activeStreams {
		// Set stream status to ended
		if err := i.routerClient.UpdateStreamStatus(context.Background(), id, "ended"); err != nil {
			log.Printf("Failed to update stream status for %s during shutdown: %v", id, err)
		}
	}
	// The server will stop when all clients disconnect or when the process exits

	log.Println("RTMP ingestor stopped")
}

// handlePublish handles RTMP publish events
func (i *RTMPIngestor) handlePublish(conn *rtmp.Conn) {
	// Extract stream key and stream ID from URL
	// URL format: rtmp://ingest.example.com/live/streamID?key=streamKey
	urlPath := conn.URL.Path
	urlQuery := conn.URL.Query()

	// Parse URL path to extract stream ID
	pathParts := strings.Split(strings.TrimPrefix(urlPath, "/"), "/")
	if len(pathParts) < 2 {
		log.Printf("Invalid URL path format: %s", urlPath)
		conn.Close()
		return
	}

	appName := pathParts[0]  // Usually "live"
	streamID := pathParts[1] // Stream ID
	streamKey := urlQuery.Get("key")

	// Check if this is a backup stream
	isBackup := urlQuery.Get("backup") == "1"

	log.Printf("Received RTMP publish request: app=%s, streamID=%s, isBackup=%v", appName, streamID, isBackup)

	// Validate stream key with router service
	valid, validateErr := i.routerClient.ValidateStreamKey(context.Background(), streamID, streamKey)
	if validateErr != nil {
		log.Printf("Failed to validate stream key: %v", validateErr)
		conn.Close()
		return
	}

	if !valid {
		log.Printf("Invalid stream key for stream %s", streamID)
		conn.Close()
		return
	}

	// Update stream status to live
	if err := i.routerClient.UpdateStreamStatus(context.Background(), streamID, "live"); err != nil {
		log.Printf("Failed to update stream status: %v", err)
		// Continue anyway
	}

	// Create stream info with GOP cache if enabled
	streamInfo := &StreamInfo{
		StreamID:        streamID,
		StartTime:       time.Now(),
		IsBackup:        isBackup,
		LastActivity:    time.Now(),
		gopCacheEnabled: i.config.GopCacheEnabled,
		maxGopItems:     i.config.GopCacheMaxItems,
	}

	if streamInfo.gopCacheEnabled {
		streamInfo.gopCache = make([]av.Packet, 0, streamInfo.maxGopItems)
	}

	// Add to active streams
	i.streamsMutex.Lock()
	i.activeStreams[streamID] = streamInfo
	i.streamsMutex.Unlock()

	// Record stream start in metrics
	i.metricsCollector.StreamStarted(streamID, isBackup)

	// Process stream
	var err error
	demuxer := conn
	muxer := conn

	for {
		pkt, err := demuxer.ReadPacket()
		if err != nil {
			if err == io.EOF {
				log.Printf("Stream %s ended (EOF)", streamID)
				break
			}
			log.Printf("Error reading packet from stream %s: %v", streamID, err)
			break
		}

		// Process the packet
		if packetErr := i.handlePacket(streamInfo)(pkt); packetErr != nil {
			log.Printf("Error processing packet for stream %s: %v", streamID, packetErr)
			break
		}

		// Store in GOP cache if enabled
		if streamInfo.gopCacheEnabled {
			streamInfo.Mutex.Lock()
			// If this is a key frame, clear the cache to start a new GOP
			if pkt.IsKeyFrame {
				streamInfo.gopCache = streamInfo.gopCache[:0]
			}

			// Add packet to cache
			if len(streamInfo.gopCache) < streamInfo.maxGopItems {
				// Make a copy of the packet to avoid data races
				packetCopy := av.Packet{
					IsKeyFrame:      pkt.IsKeyFrame,
					Idx:             pkt.Idx,
					CompositionTime: pkt.CompositionTime,
					Time:            pkt.Time,
				}
				// Copy the data
				packetCopy.Data = make([]byte, len(pkt.Data))
				copy(packetCopy.Data, pkt.Data)

				streamInfo.gopCache = append(streamInfo.gopCache, packetCopy)
			}
			streamInfo.Mutex.Unlock()
		}

		// Write the packet to the muxer
		if writeErr := muxer.WritePacket(pkt); writeErr != nil {
			log.Printf("Error writing packet for stream %s: %v", streamID, writeErr)
			break
		}
	}

	if err != nil && err != io.EOF {
		log.Printf("Stream %s ended with error: %v", streamID, err)
	} else {
		log.Printf("Stream %s ended gracefully", streamID)
	}

	// Remove from active streams
	i.streamsMutex.Lock()
	delete(i.activeStreams, streamID)
	i.streamsMutex.Unlock()

	// Update stream status to ended if this is not a backup stream
	if !isBackup {
		if err := i.routerClient.UpdateStreamStatus(context.Background(), streamID, "ended"); err != nil {
			log.Printf("Failed to update stream status for %s: %v", streamID, err)
		}
	}

	// Update metrics
	i.metricsCollector.StreamEnded(streamID, streamInfo.BytesReceived, streamInfo.FrameCount, time.Since(streamInfo.StartTime))
}

// handlePacket processes a packet from the RTMP stream
func (i *RTMPIngestor) handlePacket(streamInfo *StreamInfo) func(av.Packet) error {
	return func(pkt av.Packet) error {
		// Update stream info
		streamInfo.Mutex.Lock()
		streamInfo.FrameCount++
		streamInfo.BytesReceived += int64(len(pkt.Data))
		streamInfo.LastActivity = time.Now()
		streamInfo.Mutex.Unlock()

		// Update metrics
		i.metricsCollector.PacketReceived(streamInfo.StreamID, len(pkt.Data), pkt.IsKeyFrame)

		// Extract stream metadata if available
		if pkt.IsKeyFrame && streamInfo.Width == 0 && streamInfo.Height == 0 {
			// In a real implementation, we would extract width, height, codec info
			// This is a simplified placeholder
			streamInfo.Mutex.Lock()
			// Example metadata extraction (would be based on actual codec parsing)
			streamInfo.Width = 1280           // Example value
			streamInfo.Height = 720           // Example value
			streamInfo.FrameRate = 30         // Example value
			streamInfo.VideoBitrate = 2500000 // Example value (2.5 Mbps)
			streamInfo.AudioBitrate = 128000  // Example value (128 kbps)
			streamInfo.Codec = "h264"         // Example value
			streamInfo.Mutex.Unlock()
		}

		// Here we would process the packet for further distribution:
		// 1. Send to frame splitter service
		// 2. Store frame for backup if needed
		// 3. Extract thumbnails for preview
		// 4. Calculate metrics (bitrate, fps, etc.)

		return nil
	}
}

// monitorActiveStreams monitors active streams for timeouts
func (i *RTMPIngestor) monitorActiveStreams(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			i.streamsMutex.RLock()
			for id, info := range i.activeStreams {
				info.Mutex.Lock()
				lastActivity := info.LastActivity
				info.Mutex.Unlock()

				// Check for timeout
				if now.Sub(lastActivity) > i.config.ReadTimeout {
					log.Printf("Stream %s timed out", id)
					// Update metrics
					i.metricsCollector.StreamTimeout(id)

					// In a real system, we would:
					// 1. Notify the stream router service
					// 2. Try to reconnect or failover to backup stream
					// 3. Clean up resources

					// For now, we'll just update the stream status
					if err := i.routerClient.UpdateStreamStatus(ctx, id, "error"); err != nil {
						log.Printf("Failed to update stream status for timed out stream %s: %v", id, err)
					}
				}

				// Update stream metrics in router service
				info.Mutex.Lock()
				metrics := router.StreamMetrics{
					ViewerCount:   0, // Would be from a real source
					IngestBitrate: info.VideoBitrate + info.AudioBitrate,
					FrameRate:     info.FrameRate,
					Resolution:    fmt.Sprintf("%dx%d", info.Width, info.Height),
				}
				info.Mutex.Unlock()

				if err := i.routerClient.UpdateStreamMetrics(ctx, id, metrics); err != nil {
					log.Printf("Failed to update stream metrics for %s: %v", id, err)
				}
			}
			i.streamsMutex.RUnlock()
		case <-i.shutdownCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// getGOPCache returns a copy of the GOP cache for a stream
// This could be used to provide fast startup for new viewers
func (i *RTMPIngestor) getGOPCache(streamID string) []av.Packet {
	i.streamsMutex.RLock()
	defer i.streamsMutex.RUnlock()

	streamInfo, exists := i.activeStreams[streamID]
	if !exists || !streamInfo.gopCacheEnabled {
		return nil
	}

	streamInfo.Mutex.Lock()
	defer streamInfo.Mutex.Unlock()

	// Make a deep copy of the GOP cache
	result := make([]av.Packet, len(streamInfo.gopCache))
	for i, pkt := range streamInfo.gopCache {
		// Copy packet attributes
		result[i] = av.Packet{
			IsKeyFrame:      pkt.IsKeyFrame,
			Idx:             pkt.Idx,
			CompositionTime: pkt.CompositionTime,
			Time:            pkt.Time,
		}
		// Deep copy the data
		result[i].Data = make([]byte, len(pkt.Data))
		copy(result[i].Data, pkt.Data)
	}

	return result
}
