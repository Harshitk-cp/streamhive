package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 65536
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you would want to restrict this to allowed origins
		return true
	},
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client represents a WebSocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	streamID string
	mu       sync.Mutex
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Map of streamID to clients
	streamClients map[string]map[*Client]bool

	// Mutex for protecting streamClients
	mu sync.Mutex
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		broadcast:     make(chan []byte),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		streamClients: make(map[string]map[*Client]bool),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

			// Add to stream clients if streamID is set
			if client.streamID != "" {
				h.mu.Lock()
				if _, ok := h.streamClients[client.streamID]; !ok {
					h.streamClients[client.streamID] = make(map[*Client]bool)
				}
				h.streamClients[client.streamID][client] = true
				h.mu.Unlock()
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from stream clients if streamID is set
				if client.streamID != "" {
					h.mu.Lock()
					if clients, ok := h.streamClients[client.streamID]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.streamClients, client.streamID)
						}
					}
					h.mu.Unlock()
				}
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// BroadcastToStream sends a message to all clients connected to a specific stream
func (h *Hub) BroadcastToStream(streamID string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.streamClients[streamID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.streamClients, streamID)
				}
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Parse the message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing WebSocket message: %v", err)
			continue
		}

		// Handle the message based on its type
		c.handleMessage(msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages based on their type
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case "join_stream":
		// Handle join stream message
		var joinMsg struct {
			StreamID string `json:"stream_id"`
		}
		if err := json.Unmarshal(msg.Payload, &joinMsg); err != nil {
			log.Printf("Error parsing join_stream message: %v", err)
			return
		}

		// Update client's streamID
		c.mu.Lock()
		oldStreamID := c.streamID
		c.streamID = joinMsg.StreamID
		c.mu.Unlock()

		// Update hub's stream clients
		c.hub.mu.Lock()
		// Remove from old stream if exists
		if oldStreamID != "" && oldStreamID != joinMsg.StreamID {
			if clients, ok := c.hub.streamClients[oldStreamID]; ok {
				delete(clients, c)
				if len(clients) == 0 {
					delete(c.hub.streamClients, oldStreamID)
				}
			}
		}
		// Add to new stream
		if _, ok := c.hub.streamClients[joinMsg.StreamID]; !ok {
			c.hub.streamClients[joinMsg.StreamID] = make(map[*Client]bool)
		}
		c.hub.streamClients[joinMsg.StreamID][c] = true
		c.hub.mu.Unlock()

		// Send acknowledgement
		ack, _ := json.Marshal(map[string]interface{}{
			"type": "join_stream_ack",
			"payload": map[string]string{
				"stream_id": joinMsg.StreamID,
				"status":    "joined",
			},
		})
		c.send <- ack

	case "leave_stream":
		// Handle leave stream message
		c.mu.Lock()
		streamID := c.streamID
		c.streamID = ""
		c.mu.Unlock()

		// Update hub's stream clients
		if streamID != "" {
			c.hub.mu.Lock()
			if clients, ok := c.hub.streamClients[streamID]; ok {
				delete(clients, c)
				if len(clients) == 0 {
					delete(c.hub.streamClients, streamID)
				}
			}
			c.hub.mu.Unlock()
		}

		// Send acknowledgement
		ack, _ := json.Marshal(map[string]interface{}{
			"type": "leave_stream_ack",
			"payload": map[string]string{
				"status": "left",
			},
		})
		c.send <- ack

	case "signal":
		// Handle WebRTC signaling message
		var signalMsg struct {
			StreamID string          `json:"stream_id"`
			Payload  json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(msg.Payload, &signalMsg); err != nil {
			log.Printf("Error parsing signal message: %v", err)
			return
		}

		// Forward the signal to the appropriate service
		// In a real implementation, this would forward the signal to the websocket-signaling service
		// For now, we'll just send an acknowledgement
		ack, _ := json.Marshal(map[string]interface{}{
			"type": "signal_ack",
			"payload": map[string]string{
				"status": "received",
			},
		})
		c.send <- ack

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	authService *service.AuthService
	hub         *Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(authService *service.AuthService) *WebSocketHandler {
	hub := NewHub()
	go hub.Run()

	return &WebSocketHandler{
		authService: authService,
		hub:         hub,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID from context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

	// Create new client
	client := &Client{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	// Register client with hub
	client.hub.register <- client

	// Start pumps
	go client.writePump()
	go client.readPump()

	// Send welcome message
	welcome, _ := json.Marshal(map[string]interface{}{
		"type": "welcome",
		"payload": map[string]string{
			"user_id": userID,
			"message": "Connected to streaming platform",
		},
	})
	client.send <- welcome
}

// BroadcastToStream sends a message to all clients connected to a specific stream
func (h *WebSocketHandler) BroadcastToStream(streamID string, message []byte) {
	h.hub.BroadcastToStream(streamID, message)
}
