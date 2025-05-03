package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/rtmp-ingestor/internal/metrics"
)

// HTTPServer represents an HTTP server for metrics and health checks
type HTTPServer struct {
	server           *http.Server
	metricsCollector metrics.Collector
	startTime        time.Time
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(address string, metricsCollector metrics.Collector) *HTTPServer {
	// Create server
	s := &HTTPServer{
		metricsCollector: metricsCollector,
		startTime:        time.Now(),
	}

	// Create router
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/live", s.liveHandler)
	mux.Handle("/metrics", metricsCollector.HTTPHandler())

	// Create HTTP server
	s.server = &http.Server{
		Addr:         address,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// healthHandler handles health check requests
func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Create health check response
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
		"uptime":    time.Since(s.startTime).Seconds(),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// readyHandler handles readiness check requests
func (s *HTTPServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	// Create readiness check response
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding ready response: %v", err)
	}
}

// liveHandler handles liveness check requests
func (s *HTTPServer) liveHandler(w http.ResponseWriter, r *http.Request) {
	// Create liveness check response
	response := map[string]interface{}{
		"status":    "live",
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
		"uptime":    time.Since(s.startTime).Seconds(),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding live response: %v", err)
	}
}
