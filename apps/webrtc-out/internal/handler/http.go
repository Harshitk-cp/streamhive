package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	webrtcService *service.WebRTCService
	mux           *http.ServeMux
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(webrtcService *service.WebRTCService) http.Handler {
	h := &HTTPHandler{
		webrtcService: webrtcService,
		mux:           http.NewServeMux(),
	}

	// Register routes
	h.mux.HandleFunc("/health", h.healthHandler)
	h.mux.HandleFunc("/ready", h.readyHandler)
	h.mux.HandleFunc("/metrics", h.metricsHandler)
	h.mux.HandleFunc("/streams", h.listStreamsHandler)
	h.mux.HandleFunc("/streams/", h.streamHandler)

	return h
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// healthHandler handles health check requests
func (h *HTTPHandler) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// readyHandler handles readiness check requests
func (h *HTTPHandler) readyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding ready response: %v", err)
	}
}

// metricsHandler handles metrics requests
func (h *HTTPHandler) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get metrics from WebRTC service
	// TODO: Implement metrics

	response := map[string]interface{}{
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding metrics response: %v", err)
	}
}

// listStreamsHandler lists all active streams
func (h *HTTPHandler) listStreamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement stream listing
	streams := []string{}

	response := map[string]interface{}{
		"streams": streams,
		"count":   len(streams),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding streams response: %v", err)
	}
}

// streamHandler handles requests for a specific stream
func (h *HTTPHandler) streamHandler(w http.ResponseWriter, r *http.Request) {
	// Extract stream ID from path
	streamID := r.URL.Path[len("/streams/"):]
	if streamID == "" {
		http.Error(w, "Stream ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get stream info
		stream, err := h.webrtcService.GetStream(streamID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Create response
		response := map[string]interface{}{
			"stream_id":       stream.ID,
			"status":          stream.Status,
			"created_at":      stream.CreatedAt.Unix(),
			"current_viewers": stream.CurrentViewers,
			"total_viewers":   stream.TotalViewers,
			"resolution":      fmt.Sprintf("%dx%d", stream.Width, stream.Height),
			"frame_rate":      stream.FrameRate,
			"total_frames":    stream.TotalFrames,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding stream response: %v", err)
		}

	case http.MethodDelete:
		// Remove stream
		err := h.webrtcService.RemoveStream(streamID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
