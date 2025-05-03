package transport

import (
	"context"
	"log"
	"net/http"
	"time"
)

// HTTPServer represents an HTTP server
type HTTPServer struct {
	address     string
	handler     http.Handler
	server      *http.Server
	middlewares []func(http.Handler) http.Handler
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(address string, handler http.Handler) *HTTPServer {
	return &HTTPServer{
		address:     address,
		handler:     handler,
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

// Use adds middleware to the server
func (s *HTTPServer) Use(middleware func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middleware)
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	// Apply middleware
	handler := s.handler
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		handler = s.middlewares[i](handler)
	}

	// Create server
	s.server = &http.Server{
		Addr:         s.address,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	log.Printf("Starting HTTP server on %s", s.address)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
