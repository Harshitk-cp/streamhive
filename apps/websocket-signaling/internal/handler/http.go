package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	"github.com/gorilla/mux"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	signalingService *service.Service
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(signalingService *service.Service) *HTTPHandler {
	return &HTTPHandler{
		signalingService: signalingService,
	}
}

// SetupRoutes sets up HTTP routes
func (h *HTTPHandler) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/health", h.handleHealth).Methods("GET")
	r.HandleFunc("/status", h.handleStatus).Methods("GET")
	r.HandleFunc("/nodes", h.handleNodes).Methods("GET")
	r.HandleFunc("/streams", h.handleStreams).Methods("GET")
}

// handleHealth handles health check requests
func (h *HTTPHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// handleStatus handles status check requests
func (h *HTTPHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Get service status
	nodes, streams, clients := h.signalingService.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"nodes":     len(nodes),
		"streams":   len(streams),
		"clients":   len(clients),
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// handleNodes handles node listing requests
func (h *HTTPHandler) handleNodes(w http.ResponseWriter, r *http.Request) {
	// Get nodes
	nodes := h.signalingService.GetNodes()

	// Convert nodes to a simpler format for JSON
	nodeList := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, map[string]interface{}{
			"id":               node.ID,
			"address":          node.Address,
			"capabilities":     node.Capabilities,
			"max_streams":      node.MaxStreams,
			"active_streams":   node.ActiveStreams,
			"connections":      node.Connections,
			"cpu_usage":        node.CPUUsage,
			"memory_usage":     node.MemoryUsage,
			"last_heartbeat":   node.LastHeartbeat.UnixNano() / int64(time.Millisecond),
			"heartbeat_age_ms": time.Since(node.LastHeartbeat).Milliseconds(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeList)
}

// handleStreams handles stream listing requests
func (h *HTTPHandler) handleStreams(w http.ResponseWriter, r *http.Request) {
	// Get streams
	streams := h.signalingService.GetStreams()

	// Convert streams to a simpler format for JSON
	streamList := make([]map[string]interface{}, 0, len(streams))
	for _, stream := range streams {
		streamList = append(streamList, map[string]interface{}{
			"id":         stream.ID,
			"session_id": stream.SessionID,
			"node_id":    stream.NodeID,
			"active":     stream.Active,
			"clients":    len(stream.Clients),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(streamList)
}
