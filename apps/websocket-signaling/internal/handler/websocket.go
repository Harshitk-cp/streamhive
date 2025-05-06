package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	signalingService *service.SignalingService
	upgrader         websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(signalingService *service.SignalingService) http.Handler {
	return &WebSocketHandler{
		signalingService: signalingService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checks
				return true
			},
		},
	}
}

// ServeHTTP implements the http.Handler interface
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	query := r.URL.Query()
	streamID := query.Get("streamId")
	clientID := query.Get("clientId")
	isPublisherStr := query.Get("isPublisher")

	// Validate parameters
	if streamID == "" {
		http.Error(w, "Missing streamId parameter", http.StatusBadRequest)
		return
	}

	if clientID == "" {
		// Generate client ID if not provided
		clientID = generateClientID()
	}

	// Default to viewer if not specified
	isPublisher := isPublisherStr == "true"

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create send channel for client
	sendChan := make(chan model.SignalingMessage, 100)

	// Register client with signaling service
	err = h.signalingService.RegisterClient(clientID, streamID, sendChan, isPublisher)
	if err != nil {
		log.Printf("Failed to register client: %v", err)
		closeMessage := websocket.FormatCloseMessage(
			websocket.ClosePolicyViolation,
			err.Error(),
		)
		conn.WriteMessage(websocket.CloseMessage, closeMessage)
		conn.Close()
		return
	}

	// Ensure client is unregistered when done
	defer h.signalingService.UnregisterClient(clientID)

	// Set WebSocket connection parameters
	conn.SetReadLimit(1024 * 1024) // 1MB
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start reader and writer goroutines
	errChan := make(chan error, 2)
	go h.readPump(conn, clientID, streamID, errChan)
	go h.writePump(conn, sendChan, errChan)

	// Wait for an error or disconnect
	<-errChan
	log.Printf("Connection closed for client %s (stream: %s)", clientID, streamID)

	// Close WebSocket connection
	conn.Close()
}

// readPump reads messages from the WebSocket connection
func (h *WebSocketHandler) readPump(conn *websocket.Conn, clientID, streamID string, errChan chan error) {
	defer func() {
		errChan <- nil // Signal that the reader is done
	}()

	for {
		// Read message
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		// Handle binary frames
		if messageType == websocket.BinaryMessage {
			// Binary frames are not supported yet
			log.Printf("Binary frames are not supported")
			continue
		}

		// Parse message
		var msg model.SignalingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Set sender ID and stream ID
		msg.SenderID = clientID
		msg.StreamID = streamID

		// Process message
		h.signalingService.HandleMessage(msg)
	}
}

// writePump writes messages to the WebSocket connection
func (h *WebSocketHandler) writePump(conn *websocket.Conn, sendChan chan model.SignalingMessage, errChan chan error) {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		errChan <- nil // Signal that the writer is done
	}()

	for {
		select {
		case msg, ok := <-sendChan:
			// Channel closed
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Set write deadline
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// Serialize message
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to serialize message: %v", err)
				return
			}

			// Write message
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			// Send ping
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to write ping: %v", err)
				return
			}
		}
	}
}

// generateClientID generates a random client ID
func generateClientID() string {
	return "client-" + generateRandomString(8)
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(b)
}
