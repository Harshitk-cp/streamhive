// apps/websocket-signaling/internal/handler/websocket.go
package handler

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/config"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/metrics"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	config   *config.Config
	service  *service.Service
	metrics  metrics.Collector
	upgrader websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(cfg *config.Config, svc *service.Service, m metrics.Collector) *WebSocketHandler {
	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:    cfg.WebSocket.BufferSize,
		WriteBufferSize:   cfg.WebSocket.BufferSize,
		EnableCompression: cfg.WebSocket.EnableCompression,
		HandshakeTimeout:  cfg.WebSocket.HandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			if !cfg.HTTP.EnableCORS {
				return true
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}

			// Allow all origins if configured
			if len(cfg.HTTP.AllowedOrigins) == 1 && cfg.HTTP.AllowedOrigins[0] == "*" {
				return true
			}

			// Check against allowed origins
			for _, allowed := range cfg.HTTP.AllowedOrigins {
				if allowed == origin {
					return true
				}
			}

			return false
		},
	}

	return &WebSocketHandler{
		config:   cfg,
		service:  svc,
		metrics:  m,
		upgrader: upgrader,
	}
}

// ServeHTTP handles HTTP requests for WebSocket connections
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract stream ID from URL path
	path := strings.TrimPrefix(r.URL.Path, h.config.HTTP.WebSocketEndpoint)
	segments := strings.Split(strings.Trim(path, "/"), "/")

	if len(segments) < 1 {
		http.Error(w, "Invalid request path", http.StatusBadRequest)
		return
	}

	streamID := segments[0]

	// Extract user ID from query parameters
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Register client
	client, err := h.service.RegisterClient(streamID, userID)
	if err != nil {
		log.Printf("Failed to register client: %v", err)
		http.Error(w, "Failed to register client", http.StatusInternalServerError)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		h.service.UnregisterClient(streamID, userID)
		return
	}

	// Set connection parameters
	conn.SetReadLimit(h.config.WebSocket.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(h.config.WebSocket.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(h.config.WebSocket.PongWait))
		return nil
	})

	// Start goroutines for reading and writing
	go h.readPump(conn, client)
	go h.writePump(conn, client)
}

// readPump pumps messages from the WebSocket connection to the service
func (h *WebSocketHandler) readPump(conn *websocket.Conn, client *service.Client) {
	defer func() {
		h.service.UnregisterClient(client.StreamID, client.UserID)
		conn.Close()
	}()

	for {
		// Read message
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
				h.metrics.ClientError(client.StreamID, client.UserID, "unexpected_close")
			}
			break
		}

		// Handle message
		if err := h.service.HandleMessage(client.ID, message); err != nil {
			log.Printf("Failed to handle message: %v", err)
			h.metrics.ClientError(client.StreamID, client.UserID, "handle_error")
		}
	}
}

// writePump pumps messages from the service to the WebSocket connection
func (h *WebSocketHandler) writePump(conn *websocket.Conn, client *service.Client) {
	ticker := time.NewTicker(h.config.WebSocket.PingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(h.config.WebSocket.WriteWait))
			if !ok {
				// Channel was closed
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(h.config.WebSocket.WriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
