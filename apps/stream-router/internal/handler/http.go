package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	streamErrors "github.com/Harshitk-cp/streamhive/apps/stream-router/internal/errors"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/service"

	"github.com/gorilla/mux"
)

// HTTPHandler handles HTTP requests
type HTTPHandler struct {
	streamService       *service.StreamService
	metadataService     *service.MetadataService
	notificationService *service.NotificationService
	router              *mux.Router
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(
	streamService *service.StreamService,
	metadataService *service.MetadataService,
	notificationService *service.NotificationService,
) *HTTPHandler {
	h := &HTTPHandler{
		streamService:       streamService,
		metadataService:     metadataService,
		notificationService: notificationService,
		router:              mux.NewRouter(),
	}

	// Set up routes
	h.setupRoutes()

	return h
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// setupRoutes sets up the HTTP routes
func (h *HTTPHandler) setupRoutes() {
	// API routes
	api := h.router.PathPrefix("/api/v1").Subrouter()

	// Stream routes
	streams := api.PathPrefix("/streams").Subrouter()
	streams.HandleFunc("", h.listStreams).Methods("GET")
	streams.HandleFunc("", h.createStream).Methods("POST")
	streams.HandleFunc("/{id}", h.getStream).Methods("GET")
	streams.HandleFunc("/{id}", h.updateStream).Methods("PUT")
	streams.HandleFunc("/{id}", h.deleteStream).Methods("DELETE")
	streams.HandleFunc("/{id}/start", h.startStream).Methods("POST")
	streams.HandleFunc("/{id}/stop", h.stopStream).Methods("POST")
	streams.HandleFunc("/{id}/events", h.getStreamEvents).Methods("GET")

	// Stream metadata routes
	streams.HandleFunc("/{id}/metadata", h.getMetadata).Methods("GET")
	streams.HandleFunc("/{id}/metadata", h.setMetadata).Methods("PUT")
	streams.HandleFunc("/{id}/metadata", h.updateMetadata).Methods("PATCH")
	streams.HandleFunc("/{id}/metadata/{key}", h.deleteMetadataField).Methods("DELETE")

	// Stream outputs routes
	streams.HandleFunc("/{id}/outputs", h.addOutput).Methods("POST")
	streams.HandleFunc("/{id}/outputs/{outputId}", h.removeOutput).Methods("DELETE")

	// Stream enhancements routes
	streams.HandleFunc("/{id}/enhancements", h.addEnhancement).Methods("POST")
	streams.HandleFunc("/{id}/enhancements/{enhancementId}", h.updateEnhancement).Methods("PUT")
	streams.HandleFunc("/{id}/enhancements/{enhancementId}", h.removeEnhancement).Methods("DELETE")

	// Webhook routes
	webhooks := api.PathPrefix("/webhooks").Subrouter()
	webhooks.HandleFunc("", h.listWebhooks).Methods("GET")
	webhooks.HandleFunc("", h.registerWebhook).Methods("POST")
	webhooks.HandleFunc("/{id}", h.unregisterWebhook).Methods("DELETE")
	webhooks.HandleFunc("/{id}/test", h.testWebhook).Methods("POST")

	// Health check
	h.router.HandleFunc("/health", h.healthCheck).Methods("GET")
}

// healthCheck handles the health check endpoint
func (h *HTTPHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	}

	respondWithJSON(w, http.StatusOK, response)
}

// listStreams handles listing streams
func (h *HTTPHandler) listStreams(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Create list options
	opts := model.StreamListOptions{
		UserID: userID,
	}

	// Parse optional filters
	if status := query.Get("status"); status != "" {
		opts.Status = model.StreamStatus(status)
	}
	if visibility := query.Get("visibility"); visibility != "" {
		opts.Visibility = model.StreamVisibility(visibility)
	}
	if tags := query.Get("tags"); tags != "" {
		opts.Tags = []string{tags}
	}

	// Parse pagination parameters
	if limit := query.Get("limit"); limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil {
			opts.Limit = limitInt
		}
	}
	if offset := query.Get("offset"); offset != "" {
		if offsetInt, err := strconv.Atoi(offset); err == nil {
			opts.Offset = offsetInt
		}
	}

	// Parse sorting parameters
	if sortBy := query.Get("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}
	if sortOrder := query.Get("sort_order"); sortOrder != "" {
		opts.SortOrder = sortOrder
	}

	// Get streams
	streams, count, err := h.streamService.List(r.Context(), opts)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list streams: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"streams": streams,
		"count":   count,
		"limit":   opts.Limit,
		"offset":  opts.Offset,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// createStream handles creating a new stream
// createStream handles creating a new stream
func (h *HTTPHandler) createStream(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body - only read it once
	var req model.StreamCreateRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Create the stream
	stream, err := h.streamService.Create(r.Context(), userID, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create stream: %v", err))
		return
	}

	respondWithJSON(w, http.StatusCreated, stream)
}

// getStream handles retrieving a stream
func (h *HTTPHandler) getStream(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Get the stream
	stream, err := h.streamService.GetByID(r.Context(), streamID)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stream: %v", err))
		return
	}

	// Check if user has access to the stream
	if stream.UserID != userID && stream.Visibility != model.StreamVisibilityPublic {
		respondWithError(w, http.StatusForbidden, "access denied")
		return
	}

	respondWithJSON(w, http.StatusOK, stream)
}

// updateStream handles updating a stream
func (h *HTTPHandler) updateStream(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var req model.StreamUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Update the stream
	stream, err := h.streamService.Update(r.Context(), userID, streamID, req)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update stream: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, stream)
}

// deleteStream handles deleting a stream
func (h *HTTPHandler) deleteStream(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Delete the stream
	err := h.streamService.Delete(r.Context(), userID, streamID)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete stream: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "stream deleted"})
}

// startStream handles starting a stream
func (h *HTTPHandler) startStream(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var req model.StreamStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Set stream ID in the request
	req.StreamID = streamID

	// Start the stream
	resp, err := h.streamService.StartStream(r.Context(), userID, req)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		if err == streamErrors.ErrStreamAlreadyLive {
			respondWithError(w, http.StatusConflict, "stream is already live")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to start stream: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// stopStream handles stopping a stream
func (h *HTTPHandler) stopStream(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var req model.StreamStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to non-force stop if not specified
		req = model.StreamStopRequest{
			StreamID: streamID,
			Force:    false,
		}
	} else {
		// Set stream ID in the request
		req.StreamID = streamID
	}

	// Stop the stream
	err := h.streamService.StopStream(r.Context(), userID, req)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		if err == streamErrors.ErrStreamNotLive {
			respondWithError(w, http.StatusConflict, "stream is not live")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stop stream: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "stream stopped"})
}

// getStreamEvents handles getting events for a stream
func (h *HTTPHandler) getStreamEvents(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Parse pagination parameters
	limit := 10
	offset := 0
	if limitParam := query.Get("limit"); limitParam != "" {
		if limitInt, err := strconv.Atoi(limitParam); err == nil {
			limit = limitInt
		}
	}
	if offsetParam := query.Get("offset"); offsetParam != "" {
		if offsetInt, err := strconv.Atoi(offsetParam); err == nil {
			offset = offsetInt
		}
	}

	// Get events for the stream
	events, err := h.streamService.GetStreamEvents(r.Context(), userID, streamID, limit, offset)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stream events: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"events": events,
		"count":  len(events),
		"limit":  limit,
		"offset": offset,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// getMetadata handles getting metadata for a stream
func (h *HTTPHandler) getMetadata(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Get stream to check ownership
	stream, err := h.streamService.GetByID(r.Context(), streamID)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stream: %v", err))
		return
	}

	// Check if user has access to the stream
	if stream.UserID != userID && stream.Visibility != model.StreamVisibilityPublic {
		respondWithError(w, http.StatusForbidden, "access denied")
		return
	}

	// Get metadata
	metadata, err := h.metadataService.GetMetadata(r.Context(), streamID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get metadata: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}

// setMetadata handles setting metadata for a stream
func (h *HTTPHandler) setMetadata(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var metadata map[string]string
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Set metadata
	err := h.metadataService.SetMetadata(r.Context(), userID, streamID, metadata)
	if err != nil {
		if err.Error() == "stream not found" {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err.Error() == "unauthorized access to stream" {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to set metadata: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}

// updateMetadata handles updating metadata for a stream
func (h *HTTPHandler) updateMetadata(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Update metadata
	err := h.metadataService.UpdateMetadata(r.Context(), userID, streamID, updates)
	if err != nil {
		if err.Error() == "stream not found" {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err.Error() == "unauthorized access to stream" {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update metadata: %v", err))
		return
	}

	// Get the updated metadata
	metadata, err := h.metadataService.GetMetadata(r.Context(), streamID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get updated metadata: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, metadata)
}

// deleteMetadataField handles deleting a metadata field for a stream
func (h *HTTPHandler) deleteMetadataField(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID and key from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	key := vars["key"]
	if streamID == "" || key == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID and key are required")
		return
	}

	// Delete metadata field
	err := h.metadataService.DeleteMetadataField(r.Context(), userID, streamID, key)
	if err != nil {
		if err.Error() == "stream not found" {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err.Error() == "unauthorized access to stream" {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		if err.Error() == "metadata field not found" {
			respondWithError(w, http.StatusNotFound, "metadata field not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete metadata field: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "metadata field deleted"})
}

// addOutput handles adding an output to a stream
func (h *HTTPHandler) addOutput(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var output model.StreamOutput
	if err := json.NewDecoder(r.Body).Decode(&output); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Add output
	err := h.streamService.AddStreamOutput(r.Context(), userID, streamID, output)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add output: %v", err))
		return
	}

	respondWithJSON(w, http.StatusCreated, output)
}

// removeOutput handles removing an output from a stream
func (h *HTTPHandler) removeOutput(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID and output ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	outputID := vars["outputId"]
	if streamID == "" || outputID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID and output ID are required")
		return
	}

	// Remove output
	err := h.streamService.RemoveStreamOutput(r.Context(), userID, streamID, outputID)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to remove output: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "output removed"})
}

// addEnhancement handles adding an enhancement to a stream
func (h *HTTPHandler) addEnhancement(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	if streamID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID is required")
		return
	}

	// Parse request body
	var enhancement model.StreamEnhancement
	if err := json.NewDecoder(r.Body).Decode(&enhancement); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Add enhancement
	err := h.streamService.AddStreamEnhancement(r.Context(), userID, streamID, enhancement)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add enhancement: %v", err))
		return
	}

	respondWithJSON(w, http.StatusCreated, enhancement)
}

// updateEnhancement handles updating an enhancement for a stream
func (h *HTTPHandler) updateEnhancement(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID and enhancement ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	enhancementID := vars["enhancementId"]
	if streamID == "" || enhancementID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID and enhancement ID are required")
		return
	}

	// Parse request body
	var enhancement model.StreamEnhancement
	if err := json.NewDecoder(r.Body).Decode(&enhancement); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Set the enhancement ID
	enhancement.ID = enhancementID

	// Update enhancement
	err := h.streamService.UpdateStreamEnhancement(r.Context(), userID, streamID, enhancement)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update enhancement: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, enhancement)
}

// removeEnhancement handles removing an enhancement from a stream
func (h *HTTPHandler) removeEnhancement(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get stream ID and enhancement ID from the URL
	vars := mux.Vars(r)
	streamID := vars["id"]
	enhancementID := vars["enhancementId"]
	if streamID == "" || enhancementID == "" {
		respondWithError(w, http.StatusBadRequest, "stream ID and enhancement ID are required")
		return
	}

	// Remove enhancement
	err := h.streamService.RemoveStreamEnhancement(r.Context(), userID, streamID, enhancementID)
	if err != nil {
		if err == streamErrors.ErrStreamNotFound {
			respondWithError(w, http.StatusNotFound, "stream not found")
			return
		}
		if err == streamErrors.ErrUnauthorized {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to remove enhancement: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "enhancement removed"})
}

// listWebhooks handles listing webhooks
func (h *HTTPHandler) listWebhooks(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	streamID := query.Get("stream_id")

	// List webhooks
	webhooks, err := h.notificationService.ListWebhooks(r.Context(), userID, streamID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list webhooks: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"webhooks": webhooks,
		"count":    len(webhooks),
	})
}

// registerWebhook handles registering a webhook
func (h *HTTPHandler) registerWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req struct {
		URL      string   `json:"url"`
		Events   []string `json:"events"`
		Secret   string   `json:"secret"`
		StreamID string   `json:"stream_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Validate URL
	if req.URL == "" {
		respondWithError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Register webhook
	webhookID, err := h.notificationService.RegisterWebhook(r.Context(), userID, req.URL, req.Events, req.Secret, req.StreamID)
	if err != nil {
		if err.Error() == "webhook URL is required" {
			respondWithError(w, http.StatusBadRequest, "URL is required")
			return
		}
		if err.Error() == "unauthorized access to stream" {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to register webhook: %v", err))
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"webhook_id": webhookID,
		"url":        req.URL,
		"events":     req.Events,
		"stream_id":  req.StreamID,
	})
}

// unregisterWebhook handles unregistering a webhook
func (h *HTTPHandler) unregisterWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get webhook ID from the URL
	vars := mux.Vars(r)
	webhookID := vars["id"]
	if webhookID == "" {
		respondWithError(w, http.StatusBadRequest, "webhook ID is required")
		return
	}

	// Unregister webhook
	err := h.notificationService.UnregisterWebhook(r.Context(), userID, webhookID)
	if err != nil {
		if err.Error() == "webhook not found" {
			respondWithError(w, http.StatusNotFound, "webhook not found")
			return
		}
		if err.Error() == "unauthorized access to webhook" {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to unregister webhook: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "webhook unregistered"})
}

// testWebhook handles testing a webhook
func (h *HTTPHandler) testWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the request
	userID := getUserIDFromRequest(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get webhook ID from the URL
	vars := mux.Vars(r)
	webhookID := vars["id"]
	if webhookID == "" {
		respondWithError(w, http.StatusBadRequest, "webhook ID is required")
		return
	}

	// Test webhook
	err := h.notificationService.TestWebhook(r.Context(), userID, webhookID)
	if err != nil {
		if err.Error() == "webhook not found" {
			respondWithError(w, http.StatusNotFound, "webhook not found")
			return
		}
		if err.Error() == "unauthorized access to webhook" {
			respondWithError(w, http.StatusForbidden, "access denied")
			return
		}
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to test webhook: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "test webhook sent"})
}

// getUserIDFromRequest extracts the user ID from the request
func getUserIDFromRequest(r *http.Request) string {
	// In a real application, this would extract user ID from the authentication token
	// For simplicity, we'll use a header
	return r.Header.Get("X-User-ID")
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	// Convert payload to JSON
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "failed to marshal response"}`))
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	// Create error response
	response := map[string]string{"error": message}

	// Send response
	respondWithJSON(w, code, response)
}
