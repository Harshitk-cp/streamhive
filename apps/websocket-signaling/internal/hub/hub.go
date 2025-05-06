package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a message sent to/from a client
type Message struct {
	ClientID string
	Data     []byte
	Type     string
}

// Client represents a connected WebSocket client
type wsClient struct {
	hub        *Hub
	conn       *websocket.Conn
	id         string
	send       chan []byte
	lastActive time.Time
	mu         sync.Mutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[string]*wsClient

	// Inbound messages from clients
	broadcast chan Message

	// Register requests from clients
	register chan *wsClient

	// Unregister requests from clients
	unregister chan *wsClient

	// Direct messages to specific clients
	direct chan Message

	// Lock for clients map
	mu sync.RWMutex

	// Stop channel
	stopChan chan struct{}
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan Message),
		register:   make(chan *wsClient),
		unregister: make(chan *wsClient),
		clients:    make(map[string]*wsClient),
		direct:     make(chan Message),
		stopChan:   make(chan struct{}),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.id] = client
			h.mu.Unlock()
			log.Printf("Client registered: %s", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.id]; ok {
				delete(h.clients, client.id)
				close(client.send)
				log.Printf("Client unregistered: %s", client.id)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Broadcast message to all clients
			h.mu.RLock()
			for id, client := range h.clients {
				select {
				case client.send <- message.Data:
				default:
					close(client.send)
					delete(h.clients, id)
				}
			}
			h.mu.RUnlock()

		case message := <-h.direct:
			// Send message to specific client
			h.mu.RLock()
			client, exists := h.clients[message.ClientID]
			h.mu.RUnlock()

			if exists {
				select {
				case client.send <- message.Data:
				default:
					h.mu.Lock()
					close(client.send)
					delete(h.clients, message.ClientID)
					h.mu.Unlock()
				}
			}

		case <-h.stopChan:
			// Close all client connections
			h.mu.Lock()
			for id, client := range h.clients {
				client.conn.Close()
				close(client.send)
				delete(h.clients, id)
			}
			h.mu.Unlock()
			return
		}
	}
}

// RegisterClient registers a new WebSocket client
func (h *Hub) RegisterClient(conn *websocket.Conn, clientID string) {
	client := &wsClient{
		hub:        h,
		conn:       conn,
		id:         clientID,
		send:       make(chan []byte, 256),
		lastActive: time.Now(),
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, message []byte) {
	h.direct <- Message{
		ClientID: clientID,
		Data:     message,
	}
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- Message{
		Data: message,
	}
}

// Close closes the hub
func (h *Hub) Close() {
	close(h.stopChan)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *wsClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096) // Max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.lastActive = time.Now()
		c.mu.Unlock()
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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

		c.mu.Lock()
		c.lastActive = time.Now()
		c.mu.Unlock()

		// Parse message to determine if it's direct or broadcast
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Determine message type and handle accordingly
		msgType, ok := msg["type"].(string)
		if !ok {
			log.Printf("Message missing type field")
			continue
		}

		// Handle different message types
		switch msgType {
		case "ice_candidate":
			// Forward ICE candidate to the appropriate WebRTC service
			recipientID, ok := msg["recipient_id"].(string)
			if !ok {
				continue
			}

			c.hub.direct <- Message{
				ClientID: recipientID,
				Data:     message,
				Type:     msgType,
			}

		case "offer":
			// Forward SDP offer to the appropriate WebRTC service
			recipientID, ok := msg["recipient_id"].(string)
			if !ok {
				continue
			}

			c.hub.direct <- Message{
				ClientID: recipientID,
				Data:     message,
				Type:     msgType,
			}

		case "answer":
			// Forward SDP answer to the appropriate WebRTC service
			recipientID, ok := msg["recipient_id"].(string)
			if !ok {
				continue
			}

			c.hub.direct <- Message{
				ClientID: recipientID,
				Data:     message,
				Type:     msgType,
			}

		case "ping":
			// Respond with pong
			pong := map[string]interface{}{
				"type":      "pong",
				"client_id": c.id,
				"timestamp": time.Now().UnixNano() / int64(time.Millisecond),
			}

			pongBytes, _ := json.Marshal(pong)
			c.send <- pongBytes

		default:
			// Unknown message type, ignore
			log.Printf("Unknown message type: %s", msgType)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

			// Check if client is inactive for too long
			c.mu.Lock()
			inactive := time.Since(c.lastActive) > 2*time.Minute
			c.mu.Unlock()

			if inactive {
				log.Printf("Client %s inactive, closing connection", c.id)
				return
			}
		}
	}
}
