package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
	"github.com/gorilla/mux"
)

// RESTHandler handles REST API requests
type RESTHandler struct {
	proxyService *service.ProxyService
	authService  *service.AuthService
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(proxyService *service.ProxyService, authService *service.AuthService) *RESTHandler {
	return &RESTHandler{
		proxyService: proxyService,
		authService:  authService,
	}
}

// ListStreams handles GET /api/v1/streams
func (h *RESTHandler) ListStreams(w http.ResponseWriter, r *http.Request) {
	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// GetStream handles GET /api/v1/streams/{streamID}
func (h *RESTHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// CreateStream handles POST /api/v1/streams
func (h *RESTHandler) CreateStream(w http.ResponseWriter, r *http.Request) {

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.handleError(w, nil, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// UpdateStream handles PUT /api/v1/streams/{streamID}
func (h *RESTHandler) UpdateStream(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// DeleteStream handles DELETE /api/v1/streams/{streamID}
func (h *RESTHandler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// StartStream handles POST /api/v1/streams/{streamID}/start
func (h *RESTHandler) StartStream(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// StopStream handles POST /api/v1/streams/{streamID}/stop
func (h *RESTHandler) StopStream(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// GetStreamStats handles GET /api/v1/streams/{streamID}/stats
func (h *RESTHandler) GetStreamStats(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-health-service", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// ListWebhooks handles GET /api/v1/webhooks
func (h *RESTHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("webhook-service", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// UnregisterWebhook handles DELETE /api/v1/webhooks/{webhookID}
func (h *RESTHandler) UnregisterWebhook(w http.ResponseWriter, r *http.Request) {
	// Extract webhookID from URL
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	// Validate webhookID
	if webhookID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Webhook ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("webhook-service", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// TestWebhook handles POST /api/v1/webhooks/{webhookID}/test
func (h *RESTHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	// Extract webhookID from URL
	vars := mux.Vars(r)
	webhookID := vars["webhookID"]

	// Validate webhookID
	if webhookID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Webhook ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("webhook-service", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// GetStreamMetadata handles GET /api/v1/streams/{streamID}/metadata
func (h *RESTHandler) GetStreamMetadata(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// SetStreamMetadata handles PUT /api/v1/streams/{streamID}/metadata
func (h *RESTHandler) SetStreamMetadata(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.handleError(w, nil, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// UpdateStreamMetadata handles PATCH /api/v1/streams/{streamID}/metadata
func (h *RESTHandler) UpdateStreamMetadata(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.handleError(w, nil, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// DeleteMetadataField handles DELETE /api/v1/streams/{streamID}/metadata/{key}
func (h *RESTHandler) DeleteMetadataField(w http.ResponseWriter, r *http.Request) {
	// Extract streamID and key from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]
	key := vars["key"]

	// Validate parameters
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}
	if key == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Metadata key is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// AddOutput handles POST /api/v1/streams/{streamID}/outputs
func (h *RESTHandler) AddOutput(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.handleError(w, nil, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// RemoveOutput handles DELETE /api/v1/streams/{streamID}/outputs/{outputID}
func (h *RESTHandler) RemoveOutput(w http.ResponseWriter, r *http.Request) {
	// Extract streamID and outputID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]
	outputID := vars["outputID"]

	// Validate parameters
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}
	if outputID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Output ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// AddEnhancement handles POST /api/v1/streams/{streamID}/enhancements
func (h *RESTHandler) AddEnhancement(w http.ResponseWriter, r *http.Request) {
	// Extract streamID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]

	// Validate streamID
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.handleError(w, nil, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// UpdateEnhancement handles PUT /api/v1/streams/{streamID}/enhancements/{enhancementID}
func (h *RESTHandler) UpdateEnhancement(w http.ResponseWriter, r *http.Request) {
	// Extract streamID and enhancementID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]
	enhancementID := vars["enhancementID"]

	// Validate parameters
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}
	if enhancementID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Enhancement ID is required")
		return
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.handleError(w, nil, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// RemoveEnhancement handles DELETE /api/v1/streams/{streamID}/enhancements/{enhancementID}
func (h *RESTHandler) RemoveEnhancement(w http.ResponseWriter, r *http.Request) {
	// Extract streamID and enhancementID from URL
	vars := mux.Vars(r)
	streamID := vars["streamID"]
	enhancementID := vars["enhancementID"]

	// Validate parameters
	if streamID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Stream ID is required")
		return
	}
	if enhancementID == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Enhancement ID is required")
		return
	}

	// Proxy the request to the appropriate service
	err := h.proxyService.ProxyHTTP("stream-router", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// RegisterWebhook handles POST /api/v1/webhooks
func (h *RESTHandler) RegisterWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var webhookRequest struct {
		URL      string   `json:"url"`
		Events   []string `json:"events"`
		Secret   string   `json:"secret"`
		StreamID string   `json:"stream_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&webhookRequest)
	if err != nil {
		h.handleError(w, err, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Basic validation
	if webhookRequest.URL == "" {
		h.handleError(w, nil, http.StatusBadRequest, "Webhook URL is required")
		return
	}
	if len(webhookRequest.Events) == 0 {
		h.handleError(w, nil, http.StatusBadRequest, "At least one event is required")
		return
	}

	// Proxy the request to the appropriate service
	err = h.proxyService.ProxyHTTP("webhook-service", w, r)
	if err != nil {
		h.handleError(w, err, http.StatusServiceUnavailable)
		return
	}
}

// handleError handles error responses
func (h *RESTHandler) handleError(w http.ResponseWriter, err error, statusCode int, messages ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	message := "An error occurred"
	if len(messages) > 0 {
		message = messages[0]
	} else if err != nil {
		message = err.Error()
	}

	// Create error response
	errorResponse := struct {
		Error   string `json:"error"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{
		Error:   http.StatusText(statusCode),
		Code:    statusCode,
		Message: message,
	}

	// Write error response as JSON
	json.NewEncoder(w).Encode(errorResponse)
}
