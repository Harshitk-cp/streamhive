package transport

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConfig represents configuration for WebSocket connections
type WebSocketConfig struct {
	ReadBufferSize    int
	WriteBufferSize   int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	MaxMessageSize    int64
	EnableCompression bool
	AllowedOrigins    []string
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxMessageSize:    65536, // 64 KB
		EnableCompression: true,
		AllowedOrigins:    []string{"*"}, // Allow all origins by default
	}
}

// WebSocketUpgrader creates a websocket upgrader with the provided configuration
func WebSocketUpgrader(cfg WebSocketConfig) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		EnableCompression: cfg.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			// If all origins are allowed
			if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
				return true
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}

			// Check against allowed origins
			for _, allowed := range cfg.AllowedOrigins {
				if allowed == origin {
					return true
				}

				// Support wildcard subdomains
				if strings.HasPrefix(allowed, "*.") {
					domain := strings.TrimPrefix(allowed, "*")
					if strings.HasSuffix(origin, domain) {
						return true
					}
				}
			}

			return false
		},
	}
}

// WebSocketConnection represents a WebSocket connection with context
type WebSocketConnection struct {
	*websocket.Conn
	ctx       context.Context
	config    WebSocketConfig
	closeChan chan struct{}
	closeOnce bool
}

// NewWebSocketConnection creates a new WebSocket connection wrapper
func NewWebSocketConnection(ctx context.Context, conn *websocket.Conn, config WebSocketConfig) *WebSocketConnection {
	return &WebSocketConnection{
		Conn:      conn,
		ctx:       ctx,
		config:    config,
		closeChan: make(chan struct{}),
		closeOnce: false,
	}
}

// Context returns the connection context
func (wsc *WebSocketConnection) Context() context.Context {
	return wsc.ctx
}

// ReadJSON reads a JSON message from the WebSocket connection
func (wsc *WebSocketConnection) ReadJSON(v interface{}) error {
	// Set read deadline if configured
	if wsc.config.ReadTimeout > 0 {
		if err := wsc.Conn.SetReadDeadline(time.Now().Add(wsc.config.ReadTimeout)); err != nil {
			return err
		}
	}

	// Set message size limit
	wsc.Conn.SetReadLimit(wsc.config.MaxMessageSize)

	// Read JSON message
	return wsc.Conn.ReadJSON(v)
}

// WriteJSON writes a JSON message to the WebSocket connection
func (wsc *WebSocketConnection) WriteJSON(v interface{}) error {
	// Set write deadline if configured
	if wsc.config.WriteTimeout > 0 {
		if err := wsc.Conn.SetWriteDeadline(time.Now().Add(wsc.config.WriteTimeout)); err != nil {
			return err
		}
	}

	// Write JSON message
	return wsc.Conn.WriteJSON(v)
}

// CloseWithCode closes the WebSocket connection with the provided close code and reason
func (wsc *WebSocketConnection) CloseWithCode(code int, reason string) error {
	if wsc.closeOnce {
		return nil // Already closed
	}

	wsc.closeOnce = true
	close(wsc.closeChan)

	// Set write deadline
	if wsc.config.WriteTimeout > 0 {
		if err := wsc.Conn.SetWriteDeadline(time.Now().Add(wsc.config.WriteTimeout)); err != nil {
			return err
		}
	}

	// Send close message
	return wsc.Conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason))
}

// IsClosed returns whether the connection is closed
func (wsc *WebSocketConnection) IsClosed() bool {
	select {
	case <-wsc.closeChan:
		return true
	default:
		return false
	}
}

// StartPingPong starts a ping-pong cycle to keep the connection alive
func (wsc *WebSocketConnection) StartPingPong(pingInterval, pongWait time.Duration) {
	// Set pong handler
	wsc.Conn.SetPongHandler(func(string) error {
		if pongWait > 0 {
			wsc.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(pingInterval)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if wsc.IsClosed() {
					return
				}

				if err := wsc.Conn.WriteControl(websocket.PingMessage, []byte{},
					time.Now().Add(wsc.config.WriteTimeout)); err != nil {
					log.Printf("Ping error: %v", err)
					return
				}
			case <-wsc.closeChan:
				return
			case <-wsc.ctx.Done():
				wsc.CloseWithCode(websocket.CloseNormalClosure, "context done")
				return
			}
		}
	}()
}

// UpgradeWebSocket upgrades an HTTP connection to WebSocket
func UpgradeWebSocket(w http.ResponseWriter, r *http.Request, cfg WebSocketConfig) (*WebSocketConnection, error) {
	// Create upgrader
	upgrader := WebSocketUpgrader(cfg)

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// Create WebSocket connection wrapper
	wsConn := NewWebSocketConnection(r.Context(), conn, cfg)

	// Start ping-pong if read timeout is set
	if cfg.ReadTimeout > 0 {
		// Use 80% of read timeout for pong wait and 70% of that for ping interval
		pongWait := cfg.ReadTimeout
		pingInterval := (pongWait * 7) / 10
		wsConn.StartPingPong(pingInterval, pongWait)
	}

	return wsConn, nil
}

// WebSocketHandler is a function type for handling WebSocket connections
type WebSocketHandler func(*WebSocketConnection)

// HandleWebSocket handles a WebSocket connection with the provided handler
func HandleWebSocket(w http.ResponseWriter, r *http.Request, cfg WebSocketConfig, handler WebSocketHandler) {
	// Upgrade connection
	wsConn, err := UpgradeWebSocket(w, r, cfg)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		http.Error(w, "Failed to upgrade WebSocket connection", http.StatusInternalServerError)
		return
	}

	// Handle connection in a new goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in WebSocket handler: %v", r)
				wsConn.CloseWithCode(websocket.CloseInternalServerErr, "Internal server error")
			}
		}()

		// Call handler
		handler(wsConn)
	}()
}

// WebSocketRouter routes WebSocket connections based on path
type WebSocketRouter struct {
	config   WebSocketConfig
	handlers map[string]WebSocketHandler
}

// NewWebSocketRouter creates a new WebSocket router
func NewWebSocketRouter(cfg WebSocketConfig) *WebSocketRouter {
	return &WebSocketRouter{
		config:   cfg,
		handlers: make(map[string]WebSocketHandler),
	}
}

// Handle registers a handler for a specific path
func (r *WebSocketRouter) Handle(path string, handler WebSocketHandler) {
	r.handlers[path] = handler
}

// ServeHTTP implements http.Handler
func (r *WebSocketRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Find handler for path
	handler, ok := r.handlers[req.URL.Path]
	if !ok {
		http.NotFound(w, req)
		return
	}

	// Handle WebSocket connection
	HandleWebSocket(w, req, r.config, handler)
}
