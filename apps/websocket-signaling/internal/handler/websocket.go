package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/hub"
	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now, can be restricted in production
		return true
	},
}

type WebSocketHandler struct {
	hub              *hub.Hub
	signalingService *service.Service
}

func NewWebSocketHandler(hub *hub.Hub, signalingService *service.Service) *WebSocketHandler {
	return &WebSocketHandler{
		hub:              hub,
		signalingService: signalingService,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from query string
	query := r.URL.Query()
	streamID := query.Get("streamId")
	sessionID := query.Get("sessionId")
	clientID := query.Get("clientId")

	// Generate client ID if not provided
	if clientID == "" {
		clientID = uuid.New().String()
	}

	// Validate required parameters
	if streamID == "" {
		http.Error(w, "Missing streamId parameter", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Register client with signaling service
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = h.signalingService.RegisterClient(ctx, &webrtcPb.RegisterClientRequest{
		ClientId:  clientID,
		StreamId:  streamID,
		SessionId: sessionID,
	})

	if err != nil {
		// Send error message and close connection
		errorMsg := map[string]interface{}{
			"type":  "error",
			"error": "Failed to register client: " + err.Error(),
		}
		jsonErrorMsg, _ := json.Marshal(errorMsg)
		conn.WriteMessage(websocket.TextMessage, jsonErrorMsg)
		conn.Close()
		log.Printf("Failed to register client: %v", err)
		return
	}

	// After successful registration, send connection info to client
	connInfo := map[string]interface{}{
		"type":       "connected",
		"client_id":  clientID,
		"stream_id":  streamID,
		"session_id": sessionID,
		"timestamp":  time.Now().UnixNano() / int64(time.Millisecond),
	}
	jsonConnInfo, _ := json.Marshal(connInfo)
	conn.WriteMessage(websocket.TextMessage, jsonConnInfo)

	// Register client with hub
	h.hub.RegisterClient(conn, clientID)

	// When the connection is closed, unregister the client
	go func() {
		<-r.Context().Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		h.signalingService.UnregisterClient(ctx, &webrtcPb.UnregisterClientRequest{
			ClientId: clientID,
			StreamId: streamID,
		})
	}()
}

// HandleStreamInfo returns information about a stream
func (h *WebSocketHandler) HandleStreamInfo(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from query string
	query := r.URL.Query()
	streamID := query.Get("streamId")

	// Validate required parameters
	if streamID == "" {
		http.Error(w, "Missing streamId parameter", http.StatusBadRequest)
		return
	}

	// Get node for stream
	node, err := h.signalingService.GetNodeForStream(streamID)
	if err != nil {
		http.Error(w, "Stream not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Return stream info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stream_id": streamID,
		"node_id":   node.ID,
		"address":   node.Address,
		"active":    true,
		"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
	})
}
