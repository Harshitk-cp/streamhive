package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateID generates a unique ID with a prefix
func GenerateID() string {
	// Use timestamp as a base
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	// Generate 8 random bytes
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to time-based ID if random generation fails
		return fmt.Sprintf("str_%d", timestamp)
	}

	// Combine timestamp and random bytes
	return fmt.Sprintf("str_%d%s", timestamp, hex.EncodeToString(randomBytes))
}

// GenerateStreamKey generates a secure stream key
func GenerateStreamKey() string {
	// Generate 16 random bytes
	keyBytes := make([]byte, 16)
	_, err := rand.Read(keyBytes)
	if err != nil {
		// Fallback to time-based key if random generation fails
		return fmt.Sprintf("sk_%d", time.Now().UnixNano())
	}

	// Convert to hex string
	return fmt.Sprintf("sk_%s", hex.EncodeToString(keyBytes))
}

// GenerateToken generates a secure token for webhook authentication
func GenerateToken(length int) string {
	// Generate random bytes
	tokenBytes := make([]byte, length)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		// Fallback to time-based token if random generation fails
		return fmt.Sprintf("tkn_%d", time.Now().UnixNano())
	}

	// Convert to hex string
	return hex.EncodeToString(tokenBytes)
}
