package ingest

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/router"
)

// StreamAnalyzer analyzes stream quality metrics
type StreamAnalyzer struct {
	streamStats      map[string]*StreamStats
	statsMutex       sync.RWMutex
	updateInterval   time.Duration
	routerClient     router.Client
	metricsCollector metrics.Collector
	stopChan         chan struct{}
}

// StreamStats holds stream quality statistics
type StreamStats struct {
	StreamID       string
	StartTime      time.Time
	VideoFrames    int64
	AudioFrames    int64
	KeyFrames      int64
	TotalBytes     int64
	VideoBitrate   int64
	AudioBitrate   int64
	LastKeyFrame   time.Time
	LastVideoFrame time.Time
	LastAudioFrame time.Time
	Width          int
	Height         int
	FrameRate      float64
	CodecInfo      map[string]string
	Mutex          sync.Mutex
}

// NewStreamAnalyzer creates a new stream analyzer
func NewStreamAnalyzer(routerClient router.Client, metricsCollector metrics.Collector) *StreamAnalyzer {
	return &StreamAnalyzer{
		streamStats:      make(map[string]*StreamStats),
		updateInterval:   5 * time.Second,
		routerClient:     routerClient,
		metricsCollector: metricsCollector,
		stopChan:         make(chan struct{}),
	}
}

// Start starts the stream analyzer
func (a *StreamAnalyzer) Start() {
	go a.periodicUpdate()
}

// Stop stops the stream analyzer
func (a *StreamAnalyzer) Stop() {
	close(a.stopChan)
}

// RegisterStream registers a new stream for analysis
func (a *StreamAnalyzer) RegisterStream(streamID string) {
	a.statsMutex.Lock()
	defer a.statsMutex.Unlock()

	a.streamStats[streamID] = &StreamStats{
		StreamID:  streamID,
		StartTime: time.Now(),
		CodecInfo: make(map[string]string),
	}

	log.Printf("Registered stream %s for analysis", streamID)
}

// UnregisterStream removes a stream from analysis
func (a *StreamAnalyzer) UnregisterStream(streamID string) {
	a.statsMutex.Lock()
	defer a.statsMutex.Unlock()

	delete(a.streamStats, streamID)

	log.Printf("Unregistered stream %s from analysis", streamID)
}

// ProcessVideoFrame processes a video frame for analysis
func (a *StreamAnalyzer) ProcessVideoFrame(streamID string, isKeyFrame bool, timestamp time.Time, size int, width, height int, codecName string) {
	a.statsMutex.RLock()
	stats, exists := a.streamStats[streamID]
	a.statsMutex.RUnlock()

	if !exists {
		return
	}

	stats.Mutex.Lock()
	defer stats.Mutex.Unlock()

	stats.VideoFrames++
	stats.TotalBytes += int64(size)
	stats.LastVideoFrame = timestamp

	// Update resolution if provided
	if width > 0 && height > 0 {
		stats.Width = width
		stats.Height = height
	}

	// Update codec info if provided
	if codecName != "" {
		stats.CodecInfo["video"] = codecName
	}

	// Update key frame info
	if isKeyFrame {
		stats.KeyFrames++
		stats.LastKeyFrame = timestamp
	}

	// Update bitrate calculation
	elapsedSeconds := time.Since(stats.StartTime).Seconds()
	if elapsedSeconds > 0 {
		stats.VideoBitrate = int64(float64(stats.TotalBytes) * 8 / elapsedSeconds)
	}

	// Update frame rate
	if stats.VideoFrames > 10 {
		stats.FrameRate = float64(stats.VideoFrames) / elapsedSeconds
	}
}

// ProcessAudioFrame processes an audio frame for analysis
func (a *StreamAnalyzer) ProcessAudioFrame(streamID string, timestamp time.Time, size int, codecName string, sampleRate int, channels int) {
	a.statsMutex.RLock()
	stats, exists := a.streamStats[streamID]
	a.statsMutex.RUnlock()

	if !exists {
		return
	}

	stats.Mutex.Lock()
	defer stats.Mutex.Unlock()

	stats.AudioFrames++
	stats.TotalBytes += int64(size)
	stats.LastAudioFrame = timestamp

	// Update codec info if provided
	if codecName != "" {
		stats.CodecInfo["audio"] = codecName
		if sampleRate > 0 {
			stats.CodecInfo["audio_sample_rate"] = fmt.Sprintf("%d", sampleRate)
		}
		if channels > 0 {
			stats.CodecInfo["audio_channels"] = fmt.Sprintf("%d", channels)
		}
	}

	// Update bitrate calculation
	elapsedSeconds := time.Since(stats.StartTime).Seconds()
	if elapsedSeconds > 0 {
		stats.AudioBitrate = int64(float64(stats.TotalBytes) * 8 / elapsedSeconds)
	}
}

// periodicUpdate periodically updates stream metrics
func (a *StreamAnalyzer) periodicUpdate() {
	ticker := time.NewTicker(a.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.updateAllStreamMetrics()
		case <-a.stopChan:
			return
		}
	}
}

// updateAllStreamMetrics updates metrics for all streams
func (a *StreamAnalyzer) updateAllStreamMetrics() {
	a.statsMutex.RLock()
	streamIDs := make([]string, 0, len(a.streamStats))
	for streamID := range a.streamStats {
		streamIDs = append(streamIDs, streamID)
	}
	a.statsMutex.RUnlock()

	for _, streamID := range streamIDs {
		a.updateStreamMetrics(streamID)
	}
}

// updateStreamMetrics updates metrics for a specific stream
func (a *StreamAnalyzer) updateStreamMetrics(streamID string) {
	a.statsMutex.RLock()
	stats, exists := a.streamStats[streamID]
	a.statsMutex.RUnlock()

	if !exists {
		return
	}

	stats.Mutex.Lock()
	// Create a snapshot of metrics
	metrics := router.StreamMetrics{
		IngestBitrate: int(stats.VideoBitrate + stats.AudioBitrate),
		FrameRate:     stats.FrameRate,
		Resolution:    fmt.Sprintf("%dx%d", stats.Width, stats.Height),
	}

	// Set timestamps
	if !stats.StartTime.IsZero() {
		metrics.StartTime = stats.StartTime
	}
	if !stats.LastKeyFrame.IsZero() {
		metrics.EndTime = stats.LastKeyFrame
	}
	stats.Mutex.Unlock()

	// Update router service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := a.routerClient.UpdateStreamMetrics(ctx, streamID, metrics); err != nil {
		log.Printf("Failed to update stream metrics for %s: %v", streamID, err)
	}

	// Update local metrics
	a.metricsCollector.UpdateStreamMetrics(streamID, metrics)
}

// ExtractH264Resolution extracts resolution from H.264 SPS (Sequence Parameter Set)
func ExtractH264Resolution(sps []byte) (width, height int, err error) {
	// This is a simplified implementation
	// In a real scenario, you would need a proper H.264 bitstream parser

	if len(sps) < 10 {
		return 0, 0, fmt.Errorf("SPS too short")
	}

	// Extract width and height from SPS
	// This is a placeholder - real implementation would parse the SPS properly
	width = int(binary.BigEndian.Uint16(sps[6:8]))
	height = int(binary.BigEndian.Uint16(sps[8:10]))

	return width, height, nil
}

// ExtractAACInfo extracts sample rate and channels from AAC ADTS header
func ExtractAACInfo(adtsHeader []byte) (sampleRate int, channels int, err error) {
	// This is a simplified implementation
	// In a real scenario, you would need a proper AAC ADTS parser

	if len(adtsHeader) < 7 {
		return 0, 0, fmt.Errorf("ADTS header too short")
	}

	// Extract sample rate index
	sampleRateIndex := (adtsHeader[2] & 0x3C) >> 2

	// Sample rate table
	sampleRates := []int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000}
	if int(sampleRateIndex) < len(sampleRates) {
		sampleRate = sampleRates[sampleRateIndex]
	}

	// Extract channel configuration
	channels = int(((adtsHeader[2] & 0x01) << 2) | ((adtsHeader[3] & 0xC0) >> 6))

	return sampleRate, channels, nil
}
