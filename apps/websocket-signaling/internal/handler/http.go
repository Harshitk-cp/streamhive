package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	signalingService *service.SignalingService
	mux              *http.ServeMux
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(signalingService *service.SignalingService) http.Handler {
	h := &HTTPHandler{
		signalingService: signalingService,
		mux:              http.NewServeMux(),
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
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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

	totalClients := h.signalingService.GetTotalClientCount()

	response := map[string]interface{}{
		"total_clients": totalClients,
		"timestamp":     time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding metrics response: %v", err)
	}
}

// listStreamsHandler lists active streams
func (h *HTTPHandler) listStreamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement stream listing
	streams := []map[string]interface{}{}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"streams": streams,
		"count":   len(streams),
	}); err != nil {
		log.Printf("Error encoding streams response: %v", err)
	}
}

// streamHandler handles stream-specific requests
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
		clientCount := h.signalingService.GetClientCount(streamID)

		response := map[string]interface{}{
			"stream_id":    streamID,
			"client_count": clientCount,
			"active":       clientCount > 0,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding stream response: %v", err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHealth returns the health status of the service
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"service": "websocket-signaling",
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStreams handles stream-related requests
func (h *HTTPHandler) handleStreams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.getStreams(w, r)
	case "POST":
		h.createStream(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getStreams returns a list of active streams
func (h *HTTPHandler) getStreams(w http.ResponseWriter, r *http.Request) {
	streams := h.signalingService.GetActiveStreams()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"streams": streams,
		"count":   len(streams),
	})
}

// createStream creates a new stream
func (h *HTTPHandler) createStream(w http.ResponseWriter, r *http.Request) {
	var request struct {
		StreamID string `json:"stream_id"`
		Title    string `json:"title,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.StreamID == "" {
		http.Error(w, "stream_id is required", http.StatusBadRequest)
		return
	}

	// Create the stream
	stream := h.signalingService.CreateStream(request.StreamID, request.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(stream)
}
