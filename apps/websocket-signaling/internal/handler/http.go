// apps/websocket-signaling/internal/handler/http.go
package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	NodeID    string    `json:"node_id"`
	Uptime    string    `json:"uptime"`
}

// StreamResponse represents the stream info response
type StreamResponse struct {
	StreamID         string   `json:"stream_id"`
	ActiveClients    int      `json:"active_clients"`
	RegisteredAt     string   `json:"registered_at"`
	NodeID           string   `json:"node_id"`
	ConnectedClients []string `json:"connected_clients"`
}

// HTTPServer represents the HTTP server
type HTTPServer struct {
	config           *config.Config
	service          *service.Service
	metricsCollector metrics.Collector
	server           *http.Server
	wsHandler        *WebSocketHandler
	startTime        time.Time
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(cfg *config.Config, svc *service.Service, m metrics.Collector) *HTTPServer {
	// Create WebSocket handler
	wsHandler := NewWebSocketHandler(cfg, svc, m)

	return &HTTPServer{
		config:           cfg,
		service:          svc,
		metricsCollector: m,
		wsHandler:        wsHandler,
		startTime:        time.Now(),
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	// Create router
	mux := http.NewServeMux()

	// Register health check endpoint
	mux.HandleFunc("/health", s.healthHandler)

	// Register ready check endpoint
	mux.HandleFunc("/ready", s.readyHandler)

	// Register stream info endpoint
	mux.HandleFunc("/api/streams/", s.streamInfoHandler)

	// Register WebSocket endpoint
	mux.Handle(s.config.HTTP.WebSocketEndpoint+"/", s.wsHandler)

	// Register metrics endpoint
	if s.config.Metrics.Enabled {
		mux.Handle(s.config.Metrics.Path, s.metricsCollector.Handler())
	}

	// Create server
	s.server = &http.Server{
		Addr:              s.config.HTTP.Address,
		Handler:           s.corsMiddleware(mux),
		ReadTimeout:       s.config.HTTP.ReadTimeout,
		WriteTimeout:      s.config.HTTP.WriteTimeout,
		IdleTimeout:       s.config.HTTP.IdleTimeout,
		ReadHeaderTimeout: s.config.HTTP.ReadHeaderTimeout,
	}

	// Start server
	log.Printf("Starting HTTP server on %s", s.config.HTTP.Address)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// corsMiddleware adds CORS headers to responses
func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.HTTP.EnableCORS {
			// Get origin from request
			origin := r.Header.Get("Origin")

			// Set CORS headers
			if origin != "" {
				// Check if origin is allowed
				allowed := false
				for _, allowedOrigin := range s.config.HTTP.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						allowed = true
						break
					}
				}

				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler handles health check requests
func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "UP",
		Timestamp: time.Now(),
		Version:   s.config.Service.Version,
		NodeID:    s.config.Service.NodeID,
		Uptime:    time.Since(s.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// readyHandler handles readiness check requests
func (s *HTTPServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual readiness check logic

	resp := map[string]string{
		"status": "READY",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// streamInfoHandler handles stream info requests
func (s *HTTPServer) streamInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Extract stream ID from URL path
	streamID := r.URL.Path[len("/api/streams/"):]
	if streamID == "" {
		http.Error(w, "Stream ID is required", http.StatusBadRequest)
		return
	}

	// Get stream info
	streamInfo, err := s.service.GetStreamInfo(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Extract client IDs
	streamInfo.Mu.RLock()
	clientIDs := make([]string, 0, len(streamInfo.Clients))
	for clientID := range streamInfo.Clients {
		clientIDs = append(clientIDs, clientID)
	}
	streamInfo.Mu.RUnlock()

	// Create response
	resp := StreamResponse{
		StreamID:         streamInfo.StreamID,
		ActiveClients:    len(clientIDs),
		RegisteredAt:     streamInfo.CreatedAt.Format(time.RFC3339),
		NodeID:           streamInfo.NodeID,
		ConnectedClients: clientIDs,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
