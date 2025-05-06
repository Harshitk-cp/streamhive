package handler

// import (
// 	"context"
// 	"encoding/json"
// 	"net/http"
// 	"time"

// 	"github.com/Harshitk-cp/streamhive/apps/websocket-signaling/internal/service"
// )

// // HealthHandler handles health check requests
// type HealthHandler struct {
// 	service     *service.Service
// 	healthCheck *health.Checker
// }

// // NewHealthHandler creates a new health handler
// func NewHealthHandler(service *service.Service) *HealthHandler {
// 	h := &HealthHandler{
// 		service:     service,
// 		healthCheck: health.NewChecker(),
// 	}

// 	// Register components to check
// 	h.registerHealthChecks()

// 	// Start health checker
// 	h.healthCheck.Start()

// 	return h
// }

// // registerHealthChecks registers health checks for dependent services
// func (h *HealthHandler) registerHealthChecks() {
// 	// Register WebSocket signaling service itself
// 	h.healthCheck.RegisterComponent("websocket-signaling", func(ctx context.Context) (health.Status, error) {
// 		// Check if the service is initialized and ready
// 		nodes, streams, clients := h.service.GetStatus()

// 		// If no nodes are registered, service is considered down
// 		if len(nodes) == 0 {
// 			return health.StatusDegraded, nil
// 		}

// 		// Service is up but with no streams or clients
// 		if len(streams) == 0 && len(clients) == 0 {
// 			return health.StatusUp, nil
// 		}

// 		// Service is fully operational
// 		return health.StatusUp, nil
// 	})

// 	// Register WebRTC nodes check
// 	h.healthCheck.RegisterComponent("webrtc-nodes", func(ctx context.Context) (health.Status, error) {
// 		// Get all WebRTC nodes
// 		nodes := h.service.GetNodes()

// 		// If no nodes, consider degraded
// 		if len(nodes) == 0 {
// 			return health.StatusDegraded, nil
// 		}

// 		// Check if any nodes have heartbeats older than 1 minute
// 		now := time.Now()
// 		for _, node := range nodes {
// 			if now.Sub(node.LastHeartbeat) > 1*time.Minute {
// 				return health.StatusDegraded, nil
// 			}
// 		}

// 		return health.StatusUp, nil
// 	})

// 	// Register Redis check if used
// 	// h.healthCheck.RegisterComponent("redis", health.SimpleCheck("redis:6379"))
// }

// // ServeHTTP handles health check requests
// func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	// Use the built-in health check handler
// 	h.healthCheck.HTTPHandler().ServeHTTP(w, r)
// }

// // GetDetailedStatus returns detailed status information
// func (h *HealthHandler) GetDetailedStatus(w http.ResponseWriter, r *http.Request) {
// 	// Get status
// 	nodes, streams, clients := h.service.GetStatus()

// 	// Convert nodes to a simpler format for JSON
// 	nodeList := make([]map[string]interface{}, 0, len(nodes))
// 	for _, node := range nodes {
// 		nodeList = append(nodeList, map[string]interface{}{
// 			"id":               node.ID,
// 			"address":          node.Address,
// 			"capabilities":     node.Capabilities,
// 			"max_streams":      node.MaxStreams,
// 			"active_streams":   node.ActiveStreams,
// 			"connections":      node.Connections,
// 			"cpu_usage":        node.CPUUsage,
// 			"memory_usage":     node.MemoryUsage,
// 			"last_heartbeat":   node.LastHeartbeat.UnixNano() / int64(time.Millisecond),
// 			"heartbeat_age_ms": time.Since(node.LastHeartbeat).Milliseconds(),
// 		})
// 	}

// 	// Convert streams to a simpler format for JSON
// 	streamList := make([]map[string]interface{}, 0, len(streams))
// 	for _, stream := range streams {
// 		streamList = append(streamList, map[string]interface{}{
// 			"id":         stream.ID,
// 			"session_id": stream.SessionID,
// 			"node_id":    stream.NodeID,
// 			"active":     stream.Active,
// 			"clients":    len(stream.Clients),
// 		})
// 	}

// 	// Get client count
// 	clientCount := len(clients)

// 	// Return detailed status
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"status": map[string]interface{}{
// 			"nodes":   len(nodes),
// 			"streams": len(streams),
// 			"clients": clientCount,
// 		},
// 		"nodes":        nodeList,
// 		"streams":      streamList,
// 		"client_count": clientCount,
// 		"timestamp":    time.Now().UnixNano() / int64(time.Millisecond),
// 	})
// }

// // Close stops the health checker
// func (h *HealthHandler) Close() {
// 	h.healthCheck.Stop()
// }
