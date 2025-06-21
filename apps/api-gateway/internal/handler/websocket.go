package handler

import (
	"log"
	"net/http"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/transport"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	authService *service.AuthService
	upgrader    websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(authService *service.AuthService) *WebSocketHandler {
	return &WebSocketHandler{
		authService: authService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development - restrict in production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get default WebSocket config
	config := transport.DefaultWebSocketConfig()
	
	// Handle the WebSocket connection using the transport layer
	transport.HandleWebSocket(w, r, config, h.handleConnection)
}

// handleConnection manages a single WebSocket connection
func (h *WebSocketHandler) handleConnection(conn *transport.WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in WebSocket handler: %v", r)
		}
		conn.CloseWithCode(websocket.CloseNormalClosure, "Connection closed")
	}()

	// Get user information from context
	ctx := conn.Context()
	userID, _ := ctx.Value("user_id").(string)
	if userID == "" {
		userID = "anonymous"
	}

	log.Printf("WebSocket connection established for user: %s", userID)

	// Handle messages
	for {
		if conn.IsClosed() {
			break
		}

		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process the message
		h.processMessage(conn, message, userID)
	}

	log.Printf("WebSocket connection closed for user: %s", userID)
}

// processMessage processes incoming WebSocket messages
func (h *WebSocketHandler) processMessage(conn *transport.WebSocketConnection, message map[string]interface{}, userID string) {
	messageType, ok := message["type"].(string)
	if !ok {
		h.sendError(conn, "Invalid message format: missing type")
		return
	}

	log.Printf("Received WebSocket message type: %s from user: %s", messageType, userID)

	switch messageType {
	case "ping":
		h.handlePing(conn, message)
	case "subscribe":
		h.handleSubscribe(conn, message, userID)
	case "unsubscribe":
		h.handleUnsubscribe(conn, message, userID)
	default:
		h.sendError(conn, "Unknown message type: "+messageType)
	}
}

// handlePing responds to ping messages
func (h *WebSocketHandler) handlePing(conn *transport.WebSocketConnection, message map[string]interface{}) {
	response := map[string]interface{}{
		"type":      "pong",
		"timestamp": message["timestamp"],
	}
	
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Failed to send pong: %v", err)
	}
}

// handleSubscribe handles subscription requests
func (h *WebSocketHandler) handleSubscribe(conn *transport.WebSocketConnection, message map[string]interface{}, userID string) {
	streamID, ok := message["stream_id"].(string)
	if !ok {
		h.sendError(conn, "Invalid subscribe message: missing stream_id")
		return
	}

	// TODO: Implement actual subscription logic
	response := map[string]interface{}{
		"type":      "subscribed",
		"stream_id": streamID,
		"status":    "success",
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Failed to send subscription response: %v", err)
	}

	log.Printf("User %s subscribed to stream %s", userID, streamID)
}

// handleUnsubscribe handles unsubscription requests
func (h *WebSocketHandler) handleUnsubscribe(conn *transport.WebSocketConnection, message map[string]interface{}, userID string) {
	streamID, ok := message["stream_id"].(string)
	if !ok {
		h.sendError(conn, "Invalid unsubscribe message: missing stream_id")
		return
	}

	// TODO: Implement actual unsubscription logic
	response := map[string]interface{}{
		"type":      "unsubscribed",
		"stream_id": streamID,
		"status":    "success",
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Failed to send unsubscription response: %v", err)
	}

	log.Printf("User %s unsubscribed from stream %s", userID, streamID)
}

// sendError sends an error message to the client
func (h *WebSocketHandler) sendError(conn *transport.WebSocketConnection, errorMsg string) {
	response := map[string]interface{}{
		"type":    "error",
		"message": errorMsg,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}
