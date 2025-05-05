// apps/webrtc-out/internal/handler/http.go
package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/metrics"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	NodeID    string    `json:"node_id"`
	Uptime    string    `json:"uptime"`
}

// HTTPServer represents the HTTP server
type HTTPServer struct {
	config           *config.Config
	metricsCollector metrics.Collector
	server           *http.Server
	startTime        time.Time
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(cfg *config.Config, m metrics.Collector) *HTTPServer {
	return &HTTPServer{
		config:           cfg,
		metricsCollector: m,
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

	// Register metrics endpoint
	if s.config.Metrics.Enabled {
		mux.Handle(s.config.Metrics.Path, s.metricsCollector.Handler())
	}

	// Create server
	s.server = &http.Server{
		Addr:         s.config.HTTP.Address,
		Handler:      mux,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
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
