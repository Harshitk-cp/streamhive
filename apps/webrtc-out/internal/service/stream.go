package service

import (
	"context"
	"fmt"
	"log"
	"time"

	webrtcPb "github.com/Harshitk-cp/streamhive/libs/proto/webrtc"
)

// RegisterStreamWithSignaling registers a stream with the signaling service
// with retry logic to ensure successful registration
func (s *Service) RegisterStreamWithSignaling(ctx context.Context, streamID, sessionID string) error {
	// Create registration request
	req := &webrtcPb.RegisterStreamRequest{
		StreamId:  streamID,
		SessionId: sessionID,
		NodeId:    s.config.Service.NodeID,
	}

	// Maximum retry attempts
	maxRetries := 5
	var lastError error

	// Try to register with exponential backoff
	for i := 0; i < maxRetries; i++ {
		// Create a timeout context for this attempt
		regCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Attempt to register
		resp, err := s.signalingClient.RegisterStream(regCtx, req)
		if err == nil && resp.Success {
			log.Printf("Successfully registered stream %s with signaling service, attempt #%d", streamID, i+1)
			return nil
		}

		lastError = err
		// If context is done, abort retries
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled while registering stream: %w", ctx.Err())
		}

		// Log the error and retry after delay
		log.Printf("Failed to register stream %s with signaling service (attempt %d/%d): %v",
			streamID, i+1, maxRetries, err)

		// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
		backoff := time.Duration(100*(1<<i)) * time.Millisecond
		time.Sleep(backoff)
	}

	return fmt.Errorf("failed to register stream with signaling service after %d attempts: %w",
		maxRetries, lastError)
}

// UnregisterStreamFromSignaling unregisters a stream from the signaling service
func (s *Service) UnregisterStreamFromSignaling(ctx context.Context, streamID, sessionID string) error {
	// Create unregistration request
	req := &webrtcPb.UnregisterStreamRequest{
		StreamId:  streamID,
		SessionId: sessionID,
		NodeId:    s.config.Service.NodeID,
	}

	// Create a timeout context
	unregCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to unregister
	resp, err := s.signalingClient.UnregisterStream(unregCtx, req)
	if err != nil {
		log.Printf("Failed to unregister stream %s from signaling service: %v", streamID, err)
		return fmt.Errorf("failed to unregister stream from signaling service: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("signaling service returned failure when unregistering stream %s", streamID)
	}

	log.Printf("Successfully unregistered stream %s from signaling service", streamID)
	return nil
}

// VerifyStreamRegistration verifies that a stream is registered with the signaling service
func (s *Service) VerifyStreamRegistration(ctx context.Context, streamID string) (bool, error) {
	// Try to get WebRTC node for stream from signaling service
	// This is a workaround since there's no direct method to check stream registration

	// Create a forward message request to verify stream exists
	// Since we don't actually need to send a message, just check if it would work
	verifyReq := &webrtcPb.SignalingForwardRequest{
		StreamId:    streamID,
		SenderId:    s.config.Service.NodeID,
		RecipientId: "verify-only", // This ID doesn't need to exist
		MessageType: "verification",
		Payload:     []byte("stream verification"),
	}

	// Create a timeout context
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to forward a message to verify stream registration
	_, err := s.signalingClient.ForwardSignalingMessage(verifyCtx, verifyReq)

	// The expected behavior is that the signaling service will return "recipient not found",
	// but not "stream not found" if the stream is registered
	if err != nil {
		// Check if the error message contains "stream not found"
		if err.Error() == "stream not found" || err.Error() == "stream not active" {
			return false, nil
		}

		// Other errors (like "recipient not found") indicate the stream is registered
		// but the recipient doesn't exist, which is expected
		return true, nil
	}

	// If no error, the stream is registered
	return true, nil
}
