package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/internal/processor"
	"github.com/Harshitk-cp/streamhive/apps/frame-splitter/pkg/util"
)

// HTTPHandler handles HTTP requests for the frame splitter
type HTTPHandler struct {
	processor   *processor.FrameProcessor
	metrics     metrics.Collector
	rtmpHandler *processor.RTMPHandler
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(
	processor *processor.FrameProcessor,
	metrics metrics.Collector,
	rtmpHandler *processor.RTMPHandler,
) *HTTPHandler {
	return &HTTPHandler{
		processor:   processor,
		metrics:     metrics,
		rtmpHandler: rtmpHandler,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		h.handleHealth(w, r)
	case "/metrics":
		h.metrics.Handler().ServeHTTP(w, r)
	case "/api/v1/frame":
		h.handleFrame(w, r)
	case "/api/v1/batch":
		h.handleBatch(w, r)
	case "/api/v1/config":
		h.handleConfig(w, r)
	case "/api/v1/stats":
		h.handleStats(w, r)
	case "/api/v1/restore":
		h.handleRestore(w, r)
	case "/api/v1/backup/list":
		h.handleBackupList(w, r)
	case "/api/v1/backup/delete":
		h.handleBackupDelete(w, r)
	case "/api/v1/rtmp/frame":
		h.handleRTMPFrame(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleHealth handles health check requests
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleFrame handles frame processing requests
func (h *HTTPHandler) handleFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var frame model.Frame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log incoming frame
	log.Printf("Received frame: ID=%s, Type=%s, IsKeyFrame=%t, Size=%d bytes",
		frame.FrameID,
		util.GetFrameTypeStr(frame.Type),
		frame.IsKeyFrame,
		len(frame.Data))

	result, err := h.processor.ProcessFrame(r.Context(), frame)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleBatch handles batch processing requests
func (h *HTTPHandler) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batch model.FrameBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.processor.ProcessFrameBatch(r.Context(), batch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleConfig handles stream configuration requests
func (h *HTTPHandler) handleConfig(w http.ResponseWriter, r *http.Request) {
	streamID := r.URL.Query().Get("stream_id")
	if streamID == "" {
		http.Error(w, "Missing stream_id parameter", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		config, err := h.processor.GetStreamConfig(streamID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(config)

	case http.MethodPut:
		var config model.StreamConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if config.StreamID != streamID {
			http.Error(w, "Stream ID in URL and body do not match", http.StatusBadRequest)
			return
		}

		if err := h.processor.UpdateStreamConfig(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStats handles stream stats requests
func (h *HTTPHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	streamID := r.URL.Query().Get("stream_id")
	if streamID == "" {
		http.Error(w, "Missing stream_id parameter", http.StatusBadRequest)
		return
	}

	stats, err := h.processor.GetStreamStats(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// handleRestore handles restoring frames from backup
func (h *HTTPHandler) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	streamID := r.URL.Query().Get("stream_id")
	frameID := r.URL.Query().Get("frame_id")

	if streamID == "" || frameID == "" {
		http.Error(w, "stream_id and frame_id are required", http.StatusBadRequest)
		return
	}

	// Get the backup processor from the frame processor
	backupProcessor := h.processor.GetBackupProcessor()
	if backupProcessor == nil {
		http.Error(w, "Backup processor not available", http.StatusInternalServerError)
		return
	}

	// Restore the frame
	frame, err := backupProcessor.RestoreFrame(streamID, frameID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to restore frame: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the restored frame
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(frame)
}

// handleBackupList handles listing backed up frames
func (h *HTTPHandler) handleBackupList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	streamID := r.URL.Query().Get("stream_id")
	frameType := r.URL.Query().Get("type")
	limit := 100 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limitVal, err := strconv.Atoi(limitStr); err == nil && limitVal > 0 {
			limit = limitVal
		}
	}

	if streamID == "" {
		http.Error(w, "stream_id is required", http.StatusBadRequest)
		return
	}

	// Get the backup processor from the frame processor
	backupProcessor := h.processor.GetBackupProcessor()
	if backupProcessor == nil {
		http.Error(w, "Backup processor not available", http.StatusInternalServerError)
		return
	}

	// List backup files
	files, err := backupProcessor.ListBackupFiles(streamID, frameType, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list backup files: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the file list
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stream_id": streamID,
		"type":      frameType,
		"files":     files,
		"count":     len(files),
	})
}

// handleBackupDelete handles deleting backed up frames
func (h *HTTPHandler) handleBackupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	streamID := r.URL.Query().Get("stream_id")
	frameID := r.URL.Query().Get("frame_id")

	if streamID == "" {
		http.Error(w, "stream_id is required", http.StatusBadRequest)
		return
	}

	// Get the backup processor from the frame processor
	backupProcessor := h.processor.GetBackupProcessor()
	if backupProcessor == nil {
		http.Error(w, "Backup processor not available", http.StatusInternalServerError)
		return
	}

	// Delete backup files
	var err error
	if frameID != "" {
		// Delete a specific frame
		err = backupProcessor.DeleteBackupFrame(streamID, frameID)
	} else {
		// Delete all frames for a stream
		err = backupProcessor.DeleteAllBackupFrames(streamID)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete backup files: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// Update NewHTTPHandler constructor
func (h *HTTPHandler) handleRTMPFrame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from URL
	streamID := r.URL.Query().Get("stream_id")
	frameType := r.URL.Query().Get("type")
	frameID := r.URL.Query().Get("frame_id")
	isKeyFrame := r.URL.Query().Get("key_frame") == "true"
	sequenceStr := r.URL.Query().Get("sequence")
	timestampStr := r.URL.Query().Get("timestamp")

	// Validate required parameters
	if streamID == "" || frameType == "" {
		http.Error(w, "Missing required parameters: stream_id, type", http.StatusBadRequest)
		return
	}

	// Parse sequence if provided
	sequence := int64(0)
	if sequenceStr != "" {
		var err error
		sequence, err = strconv.ParseInt(sequenceStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid sequence number", http.StatusBadRequest)
			return
		}
	}

	// Parse timestamp if provided
	timestamp := time.Now()
	if timestampStr != "" {
		msec, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid timestamp", http.StatusBadRequest)
			return
		}
		timestamp = time.Unix(0, msec*int64(time.Millisecond))
	}

	// Read frame data
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse metadata from headers
	metadata := make(map[string]string)
	for name, values := range r.Header {
		if strings.HasPrefix(name, "X-Metadata-") {
			key := strings.TrimPrefix(name, "X-Metadata-")
			if len(values) > 0 {
				metadata[key] = values[0]
			}
		}
	}

	// Process frame based on type
	var result error
	switch frameType {
	case "video":
		result = h.rtmpHandler.HandleVideoFrame(r.Context(), streamID, frameID, data, timestamp, isKeyFrame, sequence, metadata)
	case "audio":
		result = h.rtmpHandler.HandleAudioFrame(r.Context(), streamID, frameID, data, timestamp, sequence, metadata)
	case "metadata":
		result = h.rtmpHandler.HandleMetadataFrame(r.Context(), streamID, frameID, data, timestamp, sequence)
	default:
		http.Error(w, "Invalid frame type", http.StatusBadRequest)
		return
	}

	if result != nil {
		http.Error(w, result.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
