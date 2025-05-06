package handler

// import (
// 	"context"
// 	"encoding/json"
// 	"net/http"
// 	"time"
// 	"github.com/Harshitk-cp/streamhive/apps/webrtc-out/internal/service"
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
// 	// Register WebRTC service itself
// 	h.healthCheck.RegisterComponent("webrtc", func(ctx context.Context) (health.Status, error) {
// 		// Check if the service is initialized and ready
// 		isReady := h.service.IsReady()
// 		if !isReady {
// 			return health.StatusDown, nil
// 		}
// 		return health.StatusUp, nil
// 	})

// 	// Register Signaling service check
// 	h.healthCheck.RegisterComponent("signaling", func(ctx context.Context) (health.Status, error) {
// 		// Try to ping the signaling service
// 		err := h.service.PingSignalingService(ctx)
// 		if err != nil {
// 			return health.StatusDown, err
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

// // GetStreamHealth returns the health status of a specific stream
// func (h *HealthHandler) GetStreamHealth(w http.ResponseWriter, r *http.Request) {
// 	streamID := r.URL.Query().Get("stream_id")
// 	if streamID == "" {
// 		http.Error(w, "Missing stream_id parameter", http.StatusBadRequest)
// 		return
// 	}

// 	// Get stream health
// 	stats, err := h.service.GetStreamStat(r.Context(), streamID)
// 	if err != nil {
// 		http.Error(w, "Stream not found: "+err.Error(), http.StatusNotFound)
// 		return
// 	}

// 	// Determine stream health status
// 	status := "healthy"
// 	if stats.GetActiveConnections() == 0 {
// 		status = "no_viewers"
// 	}
// 	if stats.GetVideoQueueDepth() > int32(float64(h.service.GetMaxQueueSize())*0.9) {
// 		status = "queue_near_full"
// 	}

// 	// Return stream health
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"stream_id":          stats.GetStreamId(),
// 		"active_connections": stats.GetActiveConnections(),
// 		"video_queue_depth":  stats.GetVideoQueueDepth(),
// 		"audio_queue_depth":  stats.GetAudioQueueDepth(),
// 		"status":             status,
// 		"timestamp":          time.Now().UnixNano() / int64(time.Millisecond),
// 	})
// }

// // Close stops the health checker
// func (h *HealthHandler) Close() {
// 	h.healthCheck.Stop()
// }
