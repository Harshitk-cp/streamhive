package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/model"
	"github.com/Harshitk-cp/streamhive/apps/stream-router/internal/repository"
)

// WebhookSubscription represents a webhook subscription
type WebhookSubscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	StreamID  string    `json:"stream_id"` // Empty for all streams
	URL       string    `json:"url"`
	Secret    string    `json:"secret"`
	Events    []string  `json:"events"` // List of events to subscribe to (empty for all)
	CreatedAt time.Time `json:"created_at"`
}

// NotificationService handles notifications and webhooks
type NotificationService struct {
	eventRepo      repository.EventRepository
	streamRepo     repository.StreamRepository
	subscriptions  []WebhookSubscription
	subsMutex      sync.RWMutex
	retryPolicy    []time.Duration
	httpClient     *http.Client
	processingChan chan model.StreamEvent
	stopChan       chan struct{}
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	eventRepo repository.EventRepository,
	streamRepo repository.StreamRepository,
) *NotificationService {
	// Create HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create service
	service := &NotificationService{
		eventRepo:      eventRepo,
		streamRepo:     streamRepo,
		subscriptions:  make([]WebhookSubscription, 0),
		retryPolicy:    []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second},
		httpClient:     httpClient,
		processingChan: make(chan model.StreamEvent, 100),
		stopChan:       make(chan struct{}),
	}

	// Start event processing
	go service.processEvents()

	return service
}

// AddSubscription adds a new webhook subscription
func (s *NotificationService) AddSubscription(sub WebhookSubscription) string {
	s.subsMutex.Lock()
	defer s.subsMutex.Unlock()

	// Set created at if not set
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}

	// Add subscription
	s.subscriptions = append(s.subscriptions, sub)

	return sub.ID
}

// RemoveSubscription removes a webhook subscription
func (s *NotificationService) RemoveSubscription(id string) error {
	s.subsMutex.Lock()
	defer s.subsMutex.Unlock()

	// Find subscription
	for i, sub := range s.subscriptions {
		if sub.ID == id {
			// Remove subscription by replacing it with the last element and truncating the slice
			s.subscriptions[i] = s.subscriptions[len(s.subscriptions)-1]
			s.subscriptions = s.subscriptions[:len(s.subscriptions)-1]
			return nil
		}
	}

	return fmt.Errorf("subscription not found: %s", id)
}

// GetSubscriptions gets webhook subscriptions for a user
func (s *NotificationService) GetSubscriptions(userID string) []WebhookSubscription {
	s.subsMutex.RLock()
	defer s.subsMutex.RUnlock()

	var userSubs []WebhookSubscription

	// Filter subscriptions by user ID
	for _, sub := range s.subscriptions {
		if sub.UserID == userID {
			userSubs = append(userSubs, sub)
		}
	}

	return userSubs
}

// HandleEvent handles a stream event
func (s *NotificationService) HandleEvent(event model.StreamEvent) {
	// Add event to processing channel
	select {
	case s.processingChan <- event:
		// Event added to channel
	default:
		// Channel is full, log error and drop event
		log.Printf("Event processing channel is full, dropping event: %s for stream %s", event.Type, event.StreamID)
	}
}

// processEvents processes events from the channel
func (s *NotificationService) processEvents() {
	for {
		select {
		case event := <-s.processingChan:
			// Process event
			s.processEvent(event)
		case <-s.stopChan:
			// Stop signal received
			return
		}
	}
}

// processEvent processes a single event
func (s *NotificationService) processEvent(event model.StreamEvent) {
	// Get relevant subscriptions
	subs := s.getRelevantSubscriptions(event)

	// Send notifications
	for _, sub := range subs {
		go s.sendWebhook(sub, event)
	}
}

// getRelevantSubscriptions gets subscriptions relevant to an event
func (s *NotificationService) getRelevantSubscriptions(event model.StreamEvent) []WebhookSubscription {
	s.subsMutex.RLock()
	defer s.subsMutex.RUnlock()

	var relevantSubs []WebhookSubscription

	// Filter subscriptions
	for _, sub := range s.subscriptions {
		// Check if subscription applies to this stream
		if sub.StreamID != "" && sub.StreamID != event.StreamID {
			continue
		}

		// Check if subscription applies to this event type
		if len(sub.Events) > 0 {
			eventMatched := false
			for _, eventType := range sub.Events {
				if eventType == event.Type {
					eventMatched = true
					break
				}
			}
			if !eventMatched {
				continue
			}
		}

		// Add to relevant subscriptions
		relevantSubs = append(relevantSubs, sub)
	}

	return relevantSubs
}

// sendWebhook sends a webhook notification with retries
func (s *NotificationService) sendWebhook(sub WebhookSubscription, event model.StreamEvent) {
	// Create payload
	payload := map[string]interface{}{
		"id":         event.ID,
		"stream_id":  event.StreamID,
		"type":       event.Type,
		"timestamp":  event.Timestamp,
		"data":       event.Data,
		"webhook_id": sub.ID,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal webhook payload: %v", err)
		return
	}

	// Create request
	req, err := http.NewRequest("POST", sub.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		log.Printf("Failed to create webhook request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "StreamHive-Webhook/1.0")
	req.Header.Set("X-StreamHive-Event", event.Type)
	req.Header.Set("X-StreamHive-Delivery", event.ID)
	req.Header.Set("X-StreamHive-Timestamp", fmt.Sprintf("%d", event.Timestamp.Unix()))

	// Sign payload if secret is provided
	if sub.Secret != "" {
		signature := s.signPayload(payloadBytes, sub.Secret)
		req.Header.Set("X-StreamHive-Signature", signature)
	}

	// Send request with retries
	s.sendWebhookWithRetries(req, sub, 0)
}

// signPayload signs the payload with the secret
func (s *NotificationService) signPayload(payload []byte, secret string) string {
	// Create HMAC-SHA256 hasher
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)

	// Get signature
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

// sendWebhookWithRetries sends a webhook with retries
func (s *NotificationService) sendWebhookWithRetries(req *http.Request, sub WebhookSubscription, attempt int) {
	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Webhook delivery failed for %s: %v (attempt %d)", sub.ID, err, attempt+1)
	} else {
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			log.Printf("Webhook delivered successfully to %s: %s", sub.URL, resp.Status)
			return
		}

		log.Printf("Webhook delivery failed for %s: %s (attempt %d)", sub.ID, resp.Status, attempt+1)
	}

	// Check if we've reached the maximum number of retries
	if attempt >= len(s.retryPolicy) {
		log.Printf("Webhook delivery to %s failed after %d attempts", sub.URL, attempt+1)
		return
	}

	// Retry after delay
	retryDelay := s.retryPolicy[attempt]
	log.Printf("Retrying webhook delivery to %s in %v", sub.URL, retryDelay)

	// Wait and retry
	time.Sleep(retryDelay)
	s.sendWebhookWithRetries(req, sub, attempt+1)
}

// Stop stops the notification service
func (s *NotificationService) Stop() {
	close(s.stopChan)
}

// RegisterWebhook registers a new webhook
func (s *NotificationService) RegisterWebhook(ctx context.Context, userID string, url string, events []string, secret string, streamID string) (string, error) {
	// Validate URL
	if url == "" {
		return "", fmt.Errorf("webhook URL is required")
	}

	// If streamID is provided, verify it exists and the user has permission
	if streamID != "" {
		stream, err := s.streamRepo.GetByID(ctx, streamID)
		if err != nil {
			return "", fmt.Errorf("failed to get stream: %w", err)
		}

		// Check if the user owns the stream
		if stream.UserID != userID {
			return "", fmt.Errorf("unauthorized access to stream")
		}
	}

	// Create webhook subscription
	sub := WebhookSubscription{
		ID:        fmt.Sprintf("wh_%d", time.Now().UnixNano()),
		UserID:    userID,
		StreamID:  streamID,
		URL:       url,
		Secret:    secret,
		Events:    events,
		CreatedAt: time.Now(),
	}

	// Add subscription
	id := s.AddSubscription(sub)

	return id, nil
}

// UnregisterWebhook unregisters a webhook
func (s *NotificationService) UnregisterWebhook(ctx context.Context, userID string, webhookID string) error {
	// Check if webhook exists and belongs to user
	s.subsMutex.RLock()
	var foundSub *WebhookSubscription
	for _, sub := range s.subscriptions {
		if sub.ID == webhookID {
			foundSub = &sub
			break
		}
	}
	s.subsMutex.RUnlock()

	if foundSub == nil {
		return fmt.Errorf("webhook not found")
	}

	if foundSub.UserID != userID {
		return fmt.Errorf("unauthorized access to webhook")
	}

	// Remove subscription
	return s.RemoveSubscription(webhookID)
}

// TestWebhook sends a test event to a webhook
func (s *NotificationService) TestWebhook(ctx context.Context, userID string, webhookID string) error {
	// Check if webhook exists and belongs to user
	s.subsMutex.RLock()
	var foundSub *WebhookSubscription
	for _, sub := range s.subscriptions {
		if sub.ID == webhookID {
			foundSub = &sub
			break
		}
	}
	s.subsMutex.RUnlock()

	if foundSub == nil {
		return fmt.Errorf("webhook not found")
	}

	if foundSub.UserID != userID {
		return fmt.Errorf("unauthorized access to webhook")
	}

	// Create test event
	testEvent := model.StreamEvent{
		ID:        fmt.Sprintf("evt_test_%d", time.Now().UnixNano()),
		StreamID:  foundSub.StreamID,
		Type:      "test",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "This is a test webhook delivery",
		},
	}

	// Send test webhook
	go s.sendWebhook(*foundSub, testEvent)

	return nil
}

// ListWebhooks lists webhooks for a user
func (s *NotificationService) ListWebhooks(ctx context.Context, userID string, streamID string) ([]WebhookSubscription, error) {
	// Get all user subscriptions
	allSubs := s.GetSubscriptions(userID)

	// Filter by streamID if provided
	if streamID != "" {
		var filteredSubs []WebhookSubscription
		for _, sub := range allSubs {
			if sub.StreamID == streamID || sub.StreamID == "" {
				filteredSubs = append(filteredSubs, sub)
			}
		}
		return filteredSubs, nil
	}

	return allSubs, nil
}
