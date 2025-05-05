package processor

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/pkg/util"
)

// FrameProcessor processes frames from streams
type FrameProcessor struct {
	cfg             *config.Config
	metrics         metrics.Collector
	frameQueues     map[string]chan model.Frame
	streamConfigs   map[string]model.StreamConfig
	configMutex     sync.RWMutex
	routes          map[string]FrameRoute
	routesMutex     sync.RWMutex
	backupProcessor *BackupProcessor
	stopChan        chan struct{}
	streamStats     map[string]*model.StreamStats
	statsMutex      sync.RWMutex
}

// FrameRoute represents a route for frames
type FrameRoute interface {
	// Send sends a batch of frames to the route
	Send(ctx context.Context, batch model.FrameBatch) error

	// Close closes the route
	Close() error
}

// NewFrameProcessor creates a new frame processor
func NewFrameProcessor(cfg *config.Config, metricsCollector metrics.Collector) (*FrameProcessor, error) {
	// Create frame processor
	processor := &FrameProcessor{
		cfg:           cfg,
		metrics:       metricsCollector,
		frameQueues:   make(map[string]chan model.Frame),
		streamConfigs: make(map[string]model.StreamConfig),
		routes:        make(map[string]FrameRoute),
		stopChan:      make(chan struct{}),
		streamStats:   make(map[string]*model.StreamStats),
	}

	// Initialize backup processor if enabled
	if cfg.Backup.Enabled {
		var err error
		processor.backupProcessor, err = NewBackupProcessor(cfg, metricsCollector)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize backup processor: %w", err)
		}
	}

	return processor, nil
}

// Start starts the frame processor
func (p *FrameProcessor) Start(ctx context.Context) {
	// Start backup processor if enabled
	if p.backupProcessor != nil {
		go p.backupProcessor.Start(ctx)
	}

	// Start stats collector
	go p.collectStats(ctx)
}

// Stop stops the frame processor
func (p *FrameProcessor) Stop() {
	// Signal all goroutines to stop
	close(p.stopChan)

	// Close all frame queues
	p.configMutex.Lock()
	for _, queue := range p.frameQueues {
		close(queue)
	}
	p.configMutex.Unlock()

	// Close all routes
	p.routesMutex.Lock()
	for _, route := range p.routes {
		route.Close()
	}
	p.routesMutex.Unlock()

	// Stop backup processor if enabled
	if p.backupProcessor != nil {
		p.backupProcessor.Stop()
	}
}

// ProcessFrame processes a single frame
func (p *FrameProcessor) ProcessFrame(ctx context.Context, frame model.Frame) (*model.ProcessingResult, error) {
	startTime := time.Now()

	// Set processed timestamp
	frame.ProcessedAt = startTime

	// Record frame received metric
	p.metrics.FrameReceived(frame.StreamID, string(frame.Type), frame.IsKeyFrame, len(frame.Data))

	// Get or create queue for stream
	queue, err := p.getOrCreateQueueForStream(frame.StreamID)
	if err != nil {
		p.metrics.ErrorOccurred(frame.StreamID, "queue_creation_failed")
		return nil, fmt.Errorf("failed to get or create queue for stream: %w", err)
	}

	// Check if queue is full and drop frame if necessary
	if p.cfg.FrameProcessing.DropFramesWhenFull && len(queue) >= p.cfg.FrameProcessing.MaxQueueSize {
		p.metrics.FrameDropped(frame.StreamID, string(frame.Type), "queue_full")
		return &model.ProcessingResult{
			FrameID: frame.FrameID,
			Status:  "dropped",
			Error:   "queue full",
		}, nil
	}

	// Send frame to queue
	select {
	case queue <- frame:
		// Frame accepted for processing
	case <-ctx.Done():
		p.metrics.FrameDropped(frame.StreamID, string(frame.Type), "context_cancelled")
		return &model.ProcessingResult{
			FrameID: frame.FrameID,
			Status:  "dropped",
			Error:   "context cancelled",
		}, ctx.Err()
	}

	// Backup frame if enabled
	if p.backupProcessor != nil {
		go p.backupProcessor.BackupFrame(frame)
	}

	// Get stream configuration
	config, err := p.getOrCreateStreamConfig(frame.StreamID)
	if err != nil {
		p.metrics.ErrorOccurred(frame.StreamID, "config_retrieval_failed")
		return nil, fmt.Errorf("failed to get stream configuration: %w", err)
	}

	// Update stream statistics
	p.updateStreamStats(frame)

	// Determine routing destinations
	destinations := p.determineDestinations(frame, config)

	// Record processing time
	processingTime := time.Since(startTime)
	p.metrics.FrameProcessed(frame.StreamID, string(frame.Type), processingTime, true)

	// Create processing result
	result := &model.ProcessingResult{
		FrameID:      frame.FrameID,
		Status:       "accepted",
		Destinations: destinations,
		Metadata: map[string]string{
			"processing_time_ms": fmt.Sprintf("%d", processingTime.Milliseconds()),
		},
	}

	return result, nil
}

// ProcessFrameBatch processes a batch of frames
func (p *FrameProcessor) ProcessFrameBatch(ctx context.Context, batch model.FrameBatch) (*model.BatchProcessingResult, error) {
	startTime := time.Now()

	// Process each frame in the batch
	results := make([]model.ProcessingResult, 0, len(batch.Frames))
	for _, frame := range batch.Frames {
		result, err := p.ProcessFrame(ctx, frame)
		if err != nil {
			// Continue processing other frames
			p.metrics.ErrorOccurred(batch.StreamID, "frame_processing_failed")
			results = append(results, model.ProcessingResult{
				FrameID: frame.FrameID,
				Status:  "error",
				Error:   err.Error(),
			})
			continue
		}
		results = append(results, *result)
	}

	// Log batch processing
	log.Printf("Processed batch of %d frames for stream %s in %v", len(batch.Frames), batch.StreamID, time.Since(startTime))
	p.metrics.FrameBatchProcessed(batch.StreamID, len(batch.Frames), time.Since(startTime))

	return &model.BatchProcessingResult{
		Results: results,
	}, nil
}

// GetStreamConfig gets the configuration for a stream
func (p *FrameProcessor) GetStreamConfig(streamID string) (*model.StreamConfig, error) {
	p.configMutex.RLock()
	defer p.configMutex.RUnlock()

	config, exists := p.streamConfigs[streamID]
	if !exists {
		return nil, fmt.Errorf("stream configuration not found for stream %s", streamID)
	}

	return &config, nil
}

// UpdateStreamConfig updates the configuration for a stream
func (p *FrameProcessor) UpdateStreamConfig(config model.StreamConfig) error {
	p.configMutex.Lock()
	defer p.configMutex.Unlock()

	p.streamConfigs[config.StreamID] = config
	return nil
}

// RegisterRoute registers a frame route
func (p *FrameProcessor) RegisterRoute(name string, route FrameRoute) {
	p.routesMutex.Lock()
	defer p.routesMutex.Unlock()

	p.routes[name] = route
}

// GetStreamStats gets the stats for a stream
func (p *FrameProcessor) GetStreamStats(streamID string) (*model.StreamStats, error) {
	p.statsMutex.RLock()
	defer p.statsMutex.RUnlock()

	stats, exists := p.streamStats[streamID]
	if !exists {
		return nil, fmt.Errorf("stream stats not found for stream %s", streamID)
	}

	return stats, nil
}

// getOrCreateQueueForStream gets or creates a queue for a stream
func (p *FrameProcessor) getOrCreateQueueForStream(streamID string) (chan model.Frame, error) {
	// Check if queue already exists
	p.configMutex.RLock()
	queue, exists := p.frameQueues[streamID]
	p.configMutex.RUnlock()

	if exists {
		return queue, nil
	}

	// Create new queue
	p.configMutex.Lock()
	defer p.configMutex.Unlock()

	// Double check if queue was created in the meantime
	queue, exists = p.frameQueues[streamID]
	if exists {
		return queue, nil
	}

	// Create new queue
	queue = make(chan model.Frame, p.cfg.FrameProcessing.MaxQueueSize)
	p.frameQueues[streamID] = queue

	// Start processing goroutine for queue
	go p.processQueue(streamID, queue)

	return queue, nil
}

// getOrCreateStreamConfig gets or creates a configuration for a stream
func (p *FrameProcessor) getOrCreateStreamConfig(streamID string) (model.StreamConfig, error) {
	// Check if configuration already exists
	p.configMutex.RLock()
	config, exists := p.streamConfigs[streamID]
	p.configMutex.RUnlock()

	if exists {
		return config, nil
	}

	// Create default configuration
	p.configMutex.Lock()
	defer p.configMutex.Unlock()

	// Double check if configuration was created in the meantime
	config, exists = p.streamConfigs[streamID]
	if exists {
		return config, nil
	}

	// Create default configuration
	config = model.StreamConfig{
		StreamID:      streamID,
		BackupEnabled: p.cfg.Backup.Enabled,
		StoragePath:   fmt.Sprintf("%s/%s", p.cfg.Backup.StoragePath, streamID),
		RetentionTime: time.Duration(p.cfg.Backup.RetentionMinutes) * time.Minute,
		RoutingRules:  []model.RoutingRule{},
	}

	// Add default routing rules
	if p.cfg.Routing.EnhancementService.Enabled {
		config.RoutingRules = append(config.RoutingRules, model.RoutingRule{
			Destination: "enhancement",
			Filter:      p.cfg.Routing.EnhancementService.Filter,
			Priority:    p.cfg.Routing.EnhancementService.Priority,
			Enabled:     true,
			BatchSize:   p.cfg.Routing.EnhancementService.BatchSize,
		})
	}

	if p.cfg.Routing.EncoderService.Enabled {
		config.RoutingRules = append(config.RoutingRules, model.RoutingRule{
			Destination: "encoder",
			Filter:      p.cfg.Routing.EncoderService.Filter,
			Priority:    p.cfg.Routing.EncoderService.Priority,
			Enabled:     true,
			BatchSize:   p.cfg.Routing.EncoderService.BatchSize,
		})
	}

	if p.cfg.Routing.WebRTCOut.Enabled {
		config.RoutingRules = append(config.RoutingRules, model.RoutingRule{
			Destination: "webrtc",
			Filter:      p.cfg.Routing.WebRTCOut.Filter,
			Priority:    p.cfg.Routing.WebRTCOut.Priority,
			Enabled:     true,
			BatchSize:   p.cfg.Routing.WebRTCOut.BatchSize,
		})
	}

	p.streamConfigs[streamID] = config

	return config, nil
}

// processQueue processes frames from a queue
func (p *FrameProcessor) processQueue(streamID string, queue chan model.Frame) {
	log.Printf("Starting frame processing for stream %s", streamID)

	// Create batch processing cache
	batchCaches := make(map[string][]model.Frame)
	lastFlushTime := make(map[string]time.Time)

	// Create batch processing ticker
	ticker := time.NewTicker(time.Duration(p.cfg.FrameProcessing.MaxBatchIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case frame, ok := <-queue:
			if !ok {
				// Queue closed, stop processing
				log.Printf("Frame queue closed for stream %s", streamID)
				return
			}

			// Get stream configuration
			config, err := p.getOrCreateStreamConfig(streamID)
			if err != nil {
				log.Printf("Failed to get stream configuration for stream %s: %v", streamID, err)
				continue
			}

			// Determine destinations for frame
			destinations := p.determineDestinations(frame, config)

			// Add frame to batch caches for each destination
			for _, destination := range destinations {
				p.addFrameToBatchCache(destination, frame, batchCaches, lastFlushTime)
			}

			// Check if any batch caches need to be flushed
			p.checkAndFlushBatchCaches(streamID, batchCaches, lastFlushTime, config)

		case <-ticker.C:
			// Check if any batch caches need to be flushed due to time
			config, err := p.getOrCreateStreamConfig(streamID)
			if err != nil {
				log.Printf("Failed to get stream configuration for stream %s: %v", streamID, err)
				continue
			}

			p.checkAndFlushBatchCaches(streamID, batchCaches, lastFlushTime, config)

		case <-p.stopChan:
			// Processor is stopping, flush all batches
			p.flushAllBatchCaches(streamID, batchCaches)
			log.Printf("Stopped frame processing for stream %s", streamID)
			return
		}
	}
}

// addFrameToBatchCache adds a frame to a batch cache
func (p *FrameProcessor) addFrameToBatchCache(destination string, frame model.Frame, batchCaches map[string][]model.Frame, lastFlushTime map[string]time.Time) {
	// Initialize batch cache and last flush time if needed
	if _, exists := batchCaches[destination]; !exists {
		batchCaches[destination] = make([]model.Frame, 0)
		lastFlushTime[destination] = time.Now()
	}

	// Add frame to batch cache
	batchCaches[destination] = append(batchCaches[destination], frame)
}

// checkAndFlushBatchCaches checks if any batch caches need to be flushed
func (p *FrameProcessor) checkAndFlushBatchCaches(streamID string, batchCaches map[string][]model.Frame, lastFlushTime map[string]time.Time, config model.StreamConfig) {
	// Check each destination
	for destination, batch := range batchCaches {
		// Find batch size for destination
		batchSize := p.cfg.FrameProcessing.BatchSize // Default batch size
		for _, rule := range config.RoutingRules {
			if rule.Destination == destination && rule.Enabled {
				batchSize = rule.BatchSize
				break
			}
		}

		// Check if batch is full
		if len(batch) >= batchSize {
			// Flush batch
			p.flushBatchCache(streamID, destination, batch)
			batchCaches[destination] = make([]model.Frame, 0)
			lastFlushTime[destination] = time.Now()
			continue
		}

		// Check if batch has been waiting too long
		if len(batch) > 0 && time.Since(lastFlushTime[destination]) > time.Duration(p.cfg.FrameProcessing.MaxBatchIntervalMs)*time.Millisecond {
			// Flush batch
			p.flushBatchCache(streamID, destination, batch)
			batchCaches[destination] = make([]model.Frame, 0)
			lastFlushTime[destination] = time.Now()
		}
	}
}

// flushAllBatchCaches flushes all batch caches
func (p *FrameProcessor) flushAllBatchCaches(streamID string, batchCaches map[string][]model.Frame) {
	for destination, batch := range batchCaches {
		if len(batch) > 0 {
			p.flushBatchCache(streamID, destination, batch)
		}
	}
}

// flushBatchCache flushes a batch cache
func (p *FrameProcessor) flushBatchCache(streamID string, destination string, batch []model.Frame) {
	// Skip if batch is empty
	if len(batch) == 0 {
		return
	}

	// Create frame batch
	frameBatch := model.FrameBatch{
		StreamID: streamID,
		Frames:   batch,
	}

	// Get route
	p.routesMutex.RLock()
	route, exists := p.routes[destination]
	p.routesMutex.RUnlock()

	if !exists {
		log.Printf("Route not found for destination %s", destination)
		p.metrics.ErrorOccurred(streamID, "route_not_found")
		return
	}

	// Send batch to route
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startTime := time.Now()
	err := route.Send(ctx, frameBatch)
	routingTime := time.Since(startTime)

	if err != nil {
		log.Printf("Failed to send frame batch to destination %s: %v", destination, err)
		p.metrics.ErrorOccurred(streamID, "send_batch_failed")
		return
	}

	// Record metrics
	p.metrics.FrameRouted(streamID, destination, len(batch), routingTime)
}

// determineDestinations determines the destinations for a frame
func (p *FrameProcessor) determineDestinations(frame model.Frame, config model.StreamConfig) []string {
	destinations := make([]string, 0)

	// Sort routing rules by priority
	rules := make([]model.RoutingRule, len(config.RoutingRules))
	copy(rules, config.RoutingRules)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	// Apply routing rules
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Match by filter
		if util.MatchesFilter(frame, rule.Filter) {
			destinations = append(destinations, rule.Destination)
		}
	}
	// Add default destinations based on frame type
	if len(destinations) == 0 {
		switch frame.Type {
		case model.FrameTypeVideo:
			// By default, video frames go to encoder and WebRTC
			destinations = append(destinations, "encoder", "webrtc")

			// Key frames also go to enhancement if no specific rule matched
			if frame.IsKeyFrame {
				destinations = append(destinations, "enhancement")
			}
		case model.FrameTypeAudio:
			// Audio frames go to encoder by default
			destinations = append(destinations, "encoder")
		case model.FrameTypeMetadata:
			// Metadata frames go to storage by default
			destinations = append(destinations, "storage")
		}
	}

	// Remove duplicates
	return util.UniqueStrings(destinations)
}

// collectStats collects statistics for all streams
func (p *FrameProcessor) collectStats(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Update queue size metrics
			p.configMutex.RLock()
			for streamID, queue := range p.frameQueues {
				p.metrics.QueueSize(streamID, len(queue))
			}
			p.configMutex.RUnlock()

		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		}
	}
}

// updateStreamStats updates the statistics for a stream
func (p *FrameProcessor) updateStreamStats(frame model.Frame) {
	p.statsMutex.Lock()
	defer p.statsMutex.Unlock()

	// Get or create stream stats
	stats, exists := p.streamStats[frame.StreamID]
	if !exists {
		stats = &model.StreamStats{
			StreamID: frame.StreamID,
		}
		p.streamStats[frame.StreamID] = stats
	}

	// Update stats
	stats.FramesProcessed++
	stats.BytesProcessed += int64(len(frame.Data))
	stats.LastProcessedAt = time.Now()

	// Update key frame timestamp if needed
	if frame.Type == model.FrameTypeVideo && frame.IsKeyFrame {
		stats.LastKeyFrameAt = time.Now()
	}

	// Update frame rates and bitrates
	// This is a simplified implementation
	// In a real application, we would use a more sophisticated approach
	if frame.Type == model.FrameTypeVideo {
		stats.VideoBitrate = int64(float64(len(frame.Data)) * 8 * stats.VideoFrameRate)
		stats.VideoFrameRate = float64(stats.FramesProcessed) / time.Since(stats.LastProcessedAt).Seconds()
	} else if frame.Type == model.FrameTypeAudio {
		stats.AudioBitrate = int64(float64(len(frame.Data)) * 8 * stats.AudioFrameRate)
		stats.AudioFrameRate = float64(stats.FramesProcessed) / time.Since(stats.LastProcessedAt).Seconds()
	}
}

func (p *FrameProcessor) GetBackupProcessor() *BackupProcessor {
	return p.backupProcessor
}
