package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/pkg/util"
)

// BackupProcessor handles backing up frames to disk
type BackupProcessor struct {
	cfg           *config.Config
	metrics       metrics.Collector
	backupQueue   chan model.Frame
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewBackupProcessor creates a new backup processor
func NewBackupProcessor(cfg *config.Config, metrics metrics.Collector) (*BackupProcessor, error) {
	// Create backup directory if it doesn't exist
	if err := util.EnsureDirectoryExists(cfg.Backup.StoragePath); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &BackupProcessor{
		cfg:         cfg,
		metrics:     metrics,
		backupQueue: make(chan model.Frame, 1000),
		stopChan:    make(chan struct{}),
	}, nil
}

// Start starts the backup processor
func (p *BackupProcessor) Start(ctx context.Context) {
	// Start cleanup routine
	p.cleanupTicker = time.NewTicker(1 * time.Hour)
	go p.cleanup(ctx)

	// Start worker goroutines
	for i := 0; i < 4; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	log.Println("Backup processor started")
}

// Stop stops the backup processor
func (p *BackupProcessor) Stop() {
	// Signal all goroutines to stop
	close(p.stopChan)

	// Stop cleanup ticker
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	// Wait for all goroutines to finish
	p.wg.Wait()

	log.Println("Backup processor stopped")
}

// BackupFrame adds a frame to the backup queue
func (p *BackupProcessor) BackupFrame(frame model.Frame) {
	select {
	case p.backupQueue <- frame:
		// Frame added to queue
	default:
		// Queue is full, drop frame
		p.metrics.FrameDropped(frame.StreamID, string(frame.Type), "backup_queue_full")
		log.Printf("Warning: Backup queue is full, dropping frame for stream %s", frame.StreamID)
	}
}

// worker processes frames from the backup queue
func (p *BackupProcessor) worker() {
	defer p.wg.Done()

	for {
		select {
		case frame := <-p.backupQueue:
			// Process frame
			p.processFrame(frame)
		case <-p.stopChan:
			// Stop signal received
			return
		}
	}
}

// processFrame backs up a frame to disk
func (p *BackupProcessor) processFrame(frame model.Frame) {
	startTime := time.Now()

	// Create directory for stream if it doesn't exist
	streamDir := filepath.Join(p.cfg.Backup.StoragePath, frame.StreamID)
	if err := util.EnsureDirectoryExists(streamDir); err != nil {
		log.Printf("Error: Failed to create directory for stream %s: %v", frame.StreamID, err)
		p.metrics.ErrorOccurred(frame.StreamID, "backup_directory_creation_failed")
		return
	}

	// Create subdirectory for frame type
	typeDir := filepath.Join(streamDir, string(frame.Type))
	if err := util.EnsureDirectoryExists(typeDir); err != nil {
		log.Printf("Error: Failed to create directory for frame type %s: %v", frame.Type, err)
		p.metrics.ErrorOccurred(frame.StreamID, "backup_directory_creation_failed")
		return
	}

	// Create filename
	keyStr := "n"
	if frame.IsKeyFrame {
		keyStr = "k"
	}
	filename := fmt.Sprintf("seq_%d_%s_%d_%s.bin",
		frame.Sequence,
		keyStr,
		frame.Timestamp.UnixNano()/int64(time.Millisecond),
		frame.FrameID)
	filePath := filepath.Join(typeDir, filename)

	// Write frame data to file
	var data []byte
	originalSize := len(frame.Data)

	if p.cfg.Backup.CompressionEnabled && originalSize > 1024 {
		// Use utility function for compression
		var err error
		data, err = util.CompressData(frame.Data, p.cfg.Backup.CompressionLevel)
		if err != nil {
			log.Printf("Error: Failed to compress frame data for stream %s: %v", frame.StreamID, err)
			p.metrics.ErrorOccurred(frame.StreamID, "backup_compression_failed")
			// Fall back to uncompressed data
			data = frame.Data
		}
	} else {
		// Use uncompressed data
		data = frame.Data
	}

	// Write data to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("Error: Failed to write frame data to file for stream %s: %v", frame.StreamID, err)
		p.metrics.ErrorOccurred(frame.StreamID, "backup_write_failed")
		return
	}

	// Calculate backup time
	backupTime := time.Since(startTime)

	// Write metadata file for the frame
	metadataFilePath := filePath + ".meta"
	metadataContent := fmt.Sprintf(`{
        "stream_id": "%s",
        "frame_id": "%s",
        "type": "%s",
        "sequence": %d,
        "is_key_frame": %t,
        "timestamp": "%s",
        "original_size": %d,
        "compressed_size": %d,
        "compression_ratio": %.2f,
        "backup_time_ms": %d
    }`,
		frame.StreamID,
		frame.FrameID,
		frame.Type,
		frame.Sequence,
		frame.IsKeyFrame,
		frame.Timestamp.Format(time.RFC3339Nano),
		originalSize,
		len(data),
		float64(originalSize)/float64(len(data)),
		backupTime.Milliseconds())

	if err := os.WriteFile(metadataFilePath, []byte(metadataContent), 0644); err != nil {
		log.Printf("Warning: Failed to write frame metadata for stream %s: %v", frame.StreamID, err)
		// Continue anyway, this is not critical
	}

	// Update metrics
	p.metrics.FrameBackedUp(frame.StreamID, string(frame.Type), len(data), backupTime)

	// Log backup
	// log.Printf("Backed up frame %s for stream %s to %s (%d bytes, %v)",
	// 	frame.FrameID, frame.StreamID, filePath, len(data), backupTime)
}

// performCleanup removes old frame backups
func (p *BackupProcessor) performCleanup() {
	log.Println("Starting backup cleanup")

	// Calculate cutoff time
	cutoff := time.Now().Add(-time.Duration(p.cfg.Backup.RetentionMinutes) * time.Minute)

	// Use utility function to remove old files
	if err := util.RemoveOldFiles(p.cfg.Backup.StoragePath, cutoff); err != nil {
		log.Printf("Error: Failed to remove old backup files: %v", err)
	}

	// Update backup space used metric
	p.updateBackupSpaceUsed()

	log.Println("Backup cleanup completed")
}

// updateBackupSpaceUsed updates the backup space used metric
func (p *BackupProcessor) updateBackupSpaceUsed() {
	// Use utility function to get directory size
	size, err := util.GetDirSize(p.cfg.Backup.StoragePath)
	if err != nil {
		log.Printf("Error: Failed to calculate backup space used: %v", err)
		return
	}

	// Update metric
	p.metrics.BackupSpaceUsed(size)
}

// cleanup removes old frame backups
func (p *BackupProcessor) cleanup(ctx context.Context) {
	for {
		select {
		case <-p.cleanupTicker.C:
			// Perform cleanup
			p.performCleanup()
		case <-ctx.Done():
			// Context cancelled
			return
		case <-p.stopChan:
			// Stop signal received
			return
		}
	}
}

// RestoreFrame restores a frame from backup
func (p *BackupProcessor) RestoreFrame(streamID, frameID string) (*model.Frame, error) {
	// Search for the frame in the backup storage
	var frameFilePath string
	var foundFrame bool

	streamDir := filepath.Join(p.cfg.Backup.StoragePath, streamID)
	err := filepath.Walk(streamDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".bin") && strings.Contains(path, frameID) {
			frameFilePath = path
			foundFrame = true
			return filepath.SkipDir // Stop walking once found
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search for frame: %w", err)
	}

	if !foundFrame {
		return nil, fmt.Errorf("frame not found in backup storage: %s", frameID)
	}

	// Read frame data
	data, err := os.ReadFile(frameFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read frame data: %w", err)
	}

	// Try to decompress the data
	decompressedData, err := util.DecompressData(data)
	if err != nil {
		// If decompression fails, assume the data is not compressed
		decompressedData = data
	}

	// Parse metadata from filename
	// Format: seq_<sequence>_<key>_<timestamp>_<frameid>.bin
	filename := filepath.Base(frameFilePath)
	parts := strings.Split(strings.TrimSuffix(filename, ".bin"), "_")

	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid frame filename format: %s", filename)
	}

	sequence, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frame sequence: %w", err)
	}

	isKeyFrame := parts[2] == "k"

	timestamp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frame timestamp: %w", err)
	}

	// Determine frame type from directory
	frameType := model.FrameTypeUnknown
	parentDir := filepath.Base(filepath.Dir(frameFilePath))
	switch parentDir {
	case "VIDEO":
		frameType = model.FrameTypeVideo
	case "AUDIO":
		frameType = model.FrameTypeAudio
	case "METADATA":
		frameType = model.FrameTypeMetadata
	}

	// Create frame object
	frame := &model.Frame{
		StreamID:   streamID,
		FrameID:    frameID,
		Timestamp:  time.Unix(0, timestamp*int64(time.Millisecond)),
		Type:       frameType,
		Data:       decompressedData,
		Sequence:   sequence,
		IsKeyFrame: isKeyFrame,
	}

	// Try to read metadata file for additional information
	metadataFilePath := frameFilePath + ".meta"
	if metadataBytes, err := os.ReadFile(metadataFilePath); err == nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err == nil {
			// Extract duration if available
			if durationVal, ok := metadata["duration"]; ok {
				if durationFloat, ok := durationVal.(float64); ok {
					frame.Duration = int32(durationFloat)
				}
			}

			// Extract metadata if available
			if metadataMap, ok := metadata["metadata"].(map[string]interface{}); ok {
				frame.Metadata = make(map[string]string)
				for k, v := range metadataMap {
					frame.Metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	return frame, nil
}

// BackupFileInfo represents information about a backed up frame file
type BackupFileInfo struct {
	FrameID    string    `json:"frame_id"`
	Type       string    `json:"type"`
	Sequence   int64     `json:"sequence"`
	IsKeyFrame bool      `json:"is_key_frame"`
	Timestamp  time.Time `json:"timestamp"`
	Size       int64     `json:"size"`
	Path       string    `json:"path"`
}

// ListBackupFiles lists backup files for a stream
func (p *BackupProcessor) ListBackupFiles(streamID, frameType string, limit int) ([]BackupFileInfo, error) {
	streamDir := filepath.Join(p.cfg.Backup.StoragePath, streamID)
	if !dirExists(streamDir) {
		return nil, fmt.Errorf("no backups found for stream: %s", streamID)
	}

	// Determine which directories to search
	var dirs []string
	if frameType != "" {
		// Search only in the specified type directory
		typeDir := filepath.Join(streamDir, frameType)
		if dirExists(typeDir) {
			dirs = append(dirs, typeDir)
		}
	} else {
		// Search in all type directories
		entries, err := os.ReadDir(streamDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read stream directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, filepath.Join(streamDir, entry.Name()))
			}
		}
	}

	// Find all backup files
	var files []BackupFileInfo
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bin") {
				// Parse filename
				// Format: seq_<sequence>_<key>_<timestamp>_<frameid>.bin
				parts := strings.Split(strings.TrimSuffix(entry.Name(), ".bin"), "_")
				if len(parts) < 5 {
					continue // Invalid filename format
				}

				sequence, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					continue // Invalid sequence
				}

				isKeyFrame := parts[2] == "k"

				timestamp, err := strconv.ParseInt(parts[3], 10, 64)
				if err != nil {
					continue // Invalid timestamp
				}

				frameID := parts[4]

				// Get file info
				info, err := entry.Info()
				if err != nil {
					continue // Cannot get file info
				}

				// Add file info to list
				files = append(files, BackupFileInfo{
					FrameID:    frameID,
					Type:       filepath.Base(dir),
					Sequence:   sequence,
					IsKeyFrame: isKeyFrame,
					Timestamp:  time.Unix(0, timestamp*int64(time.Millisecond)),
					Size:       info.Size(),
					Path:       filepath.Join(dir, entry.Name()),
				})
			}
		}
	}

	// Sort files by timestamp (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Timestamp.After(files[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	return files, nil
}

// DeleteBackupFrame deletes a backed up frame
func (p *BackupProcessor) DeleteBackupFrame(streamID, frameID string) error {
	// Find the frame file
	streamDir := filepath.Join(p.cfg.Backup.StoragePath, streamID)
	var framePath string
	var metadataPath string
	var found bool

	err := filepath.Walk(streamDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".bin") && strings.Contains(path, frameID) {
			framePath = path
			metadataPath = path + ".meta"
			found = true
			return filepath.SkipDir // Stop walking once found
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to search for frame: %w", err)
	}

	if !found {
		return fmt.Errorf("frame not found: %s", frameID)
	}

	// Delete the frame file
	if err := os.Remove(framePath); err != nil {
		return fmt.Errorf("failed to delete frame file: %w", err)
	}

	// Delete the metadata file if it exists
	if _, err := os.Stat(metadataPath); err == nil {
		if err := os.Remove(metadataPath); err != nil {
			log.Printf("Warning: Failed to delete metadata file: %v", err)
			// Continue anyway
		}
	}

	return nil
}

// DeleteAllBackupFrames deletes all backed up frames for a stream
func (p *BackupProcessor) DeleteAllBackupFrames(streamID string) error {
	streamDir := filepath.Join(p.cfg.Backup.StoragePath, streamID)
	if !dirExists(streamDir) {
		return nil // Nothing to delete
	}

	// Remove the entire stream directory
	if err := os.RemoveAll(streamDir); err != nil {
		return fmt.Errorf("failed to delete stream directory: %w", err)
	}

	return nil
}

// Helper function to check if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}
