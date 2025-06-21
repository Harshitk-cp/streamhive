package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections for signaling
type WebSocketHandler struct {
	service  *service.SignalingService
	upgrader websocket.Upgrader
	clients  map[string]*websocket.Conn
	mutex    sync.RWMutex
}

// SignalingMessage represents a signaling message
type SignalingMessage struct {
	Type     string `json:"type"`
	SenderID string `json:"sender_id"`
	StreamID string `json:"stream_id"`
	Payload  string `json:"payload"`
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(service *service.SignalingService) *WebSocketHandler {
	return &WebSocketHandler{
		service: service,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development - restrict in production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(map[string]*websocket.Conn),
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	log.Printf("WebSocket connection established with %s", r.RemoteAddr)

	// Handle the connection
	h.handleConnection(conn)
}

// handleConnection manages a single WebSocket connection
func (h *WebSocketHandler) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		log.Printf("WebSocket connection closed")
	}()

	var clientID string

	for {
		// Read message from client
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received message: %s", string(messageBytes))

		// Parse the signaling message
		var msg SignalingMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			h.sendError(conn, "Invalid message format")
			continue
		}

		// Register client if not already registered
		if clientID == "" && msg.SenderID != "" {
			clientID = msg.SenderID
			h.mutex.Lock()
			h.clients[clientID] = conn
			h.mutex.Unlock()
			log.Printf("Registered client: %s", clientID)
		}

		// Handle different message types
		switch msg.Type {
		case "offer":
			h.handleOffer(conn, &msg)
		case "answer":
			h.handleAnswer(conn, &msg)
		case "ice_candidate":
			h.handleICECandidate(conn, &msg)
		case "ping":
			h.handlePing(conn, &msg)
		case "request_keyframe":
			h.handleKeyframeRequest(conn, &msg)
		default:
			log.Printf("Unknown message type: %s", msg.Type)
			h.sendError(conn, "Unknown message type")
		}
	}

	// Cleanup client registration
	if clientID != "" {
		h.mutex.Lock()
		delete(h.clients, clientID)
		h.mutex.Unlock()
		log.Printf("Unregistered client: %s", clientID)
	}
}

// handleOffer processes WebRTC offer from client
func (h *WebSocketHandler) handleOffer(conn *websocket.Conn, msg *SignalingMessage) {
	log.Printf("Processing offer from %s for stream %s", msg.SenderID, msg.StreamID)

	// For now, simulate a response - in a real implementation, this would
	// communicate with a media server or streaming service
	response := SignalingMessage{
		Type:     "answer",
		SenderID: "server",
		StreamID: msg.StreamID,
		Payload:  h.generateMockAnswer(msg.Payload),
	}

	h.sendMessage(conn, &response)
}

// handleAnswer processes WebRTC answer (typically from streamer)
func (h *WebSocketHandler) handleAnswer(conn *websocket.Conn, msg *SignalingMessage) {
	log.Printf("Processing answer from %s for stream %s", msg.SenderID, msg.StreamID)
	// Forward to appropriate viewer or handle as needed
}

// handleICECandidate processes ICE candidates
func (h *WebSocketHandler) handleICECandidate(conn *websocket.Conn, msg *SignalingMessage) {
	log.Printf("Processing ICE candidate from %s", msg.SenderID)
	
	// Echo back ICE candidate for testing
	response := SignalingMessage{
		Type:     "ice_candidate",
		SenderID: "server",
		StreamID: msg.StreamID,
		Payload:  msg.Payload,
	}

	h.sendMessage(conn, &response)
}

// handlePing responds to ping messages
func (h *WebSocketHandler) handlePing(conn *websocket.Conn, msg *SignalingMessage) {
	response := SignalingMessage{
		Type:     "pong",
		SenderID: "server",
		StreamID: msg.StreamID,
	}

	h.sendMessage(conn, &response)
}

// handleKeyframeRequest processes keyframe requests
func (h *WebSocketHandler) handleKeyframeRequest(conn *websocket.Conn, msg *SignalingMessage) {
	log.Printf("Keyframe request from %s for stream %s", msg.SenderID, msg.StreamID)
	// In a real implementation, this would trigger a keyframe from the encoder
}

// sendMessage sends a message to the WebSocket connection
func (h *WebSocketHandler) sendMessage(conn *websocket.Conn, msg *SignalingMessage) {
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// sendError sends an error message to the client
func (h *WebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	response := map[string]interface{}{
		"error": errorMsg,
		"type":  "error",
	}

	messageBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal error message: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

// generateMockAnswer creates a mock SDP answer for testing
func (h *WebSocketHandler) generateMockAnswer(offerPayload string) string {
	// This is a simplified mock answer - in production, you'd integrate with
	// a real media server like Pion WebRTC, mediasoup, etc.
	
	mockAnswer := map[string]interface{}{
		"type": "answer",
		"sdp": `v=0
o=- 123456789 123456789 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE 0 1
a=msid-semantic: WMS stream
m=video 9 UDP/TLS/RTP/SAVPF 96
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:test
a=ice-pwd:testpassword
a=ice-options:trickle
a=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00
a=setup:active
a=mid:0
a=recvonly
a=rtcp-mux
a=rtcp-rsize
a=rtpmap:96 H264/90000
a=fmtp:96 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f
m=audio 9 UDP/TLS/RTP/SAVPF 111
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:test
a=ice-pwd:testpassword
a=ice-options:trickle
a=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00
a=setup:active
a=mid:1
a=recvonly
a=rtcp-mux
a=rtpmap:111 opus/48000/2`,
	}

	jsonBytes, _ := json.Marshal(mockAnswer)
	return string(jsonBytes)
}
