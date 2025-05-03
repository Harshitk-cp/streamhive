package service

import (
	"errors"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/config"
	"golang.org/x/time/rate"
)

// Errors returned by the rate limiter
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// ClientRateLimiter holds rate limiter for a specific client
type ClientRateLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter handles rate limiting for API requests
type RateLimiter struct {
	enabled         bool
	requestsPerMin  int
	burstSize       int
	expirationTime  time.Duration
	clientLimiters  map[string]*ClientRateLimiter
	mu              sync.RWMutex
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewRateLimiter creates a new rate limiter service
func NewRateLimiter(cfg config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		enabled:         cfg.Enabled,
		requestsPerMin:  cfg.RequestsPerMin,
		burstSize:       cfg.BurstSize,
		expirationTime:  cfg.ExpirationTime,
		clientLimiters:  make(map[string]*ClientRateLimiter),
		cleanupInterval: 10 * time.Minute, // Clean up every 10 minutes
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine to remove expired limiters
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed based on rate limiting rules
func (rl *RateLimiter) Allow(clientID string) error {
	// If rate limiting is disabled, always allow
	if !rl.enabled {
		return nil
	}

	rl.mu.RLock()
	limiter, exists := rl.clientLimiters[clientID]
	rl.mu.RUnlock()

	// If limiter doesn't exist for this client, create one
	if !exists {
		rl.mu.Lock()
		// Double check to avoid race condition
		limiter, exists = rl.clientLimiters[clientID]
		if !exists {
			// Create a new limiter for this client
			// Convert requests per minute to requests per second
			rps := float64(rl.requestsPerMin) / 60.0
			limiter = &ClientRateLimiter{
				limiter:  rate.NewLimiter(rate.Limit(rps), rl.burstSize),
				lastSeen: time.Now(),
			}
			rl.clientLimiters[clientID] = limiter
		}
		rl.mu.Unlock()
	}

	// Update last seen time
	limiter.lastSeen = time.Now()

	// Check if request is allowed
	if !limiter.limiter.Allow() {
		return ErrRateLimitExceeded
	}

	return nil
}

// GetLimit returns the current rate limit for a client
func (rl *RateLimiter) GetLimit(clientID string) (int, int) {
	if !rl.enabled {
		return 0, 0 // No limit
	}

	// Default values for new clients
	limit := rl.requestsPerMin
	burst := rl.burstSize

	// If client has a limiter, use its values
	rl.mu.RLock()
	limiter, exists := rl.clientLimiters[clientID]
	rl.mu.RUnlock()

	if exists {
		// Convert from requests per second to requests per minute
		limit = int(float64(limiter.limiter.Limit()) * 60.0)
		burst = limiter.limiter.Burst()
	}

	return limit, burst
}

// SetLimit sets custom rate limit for a specific client
func (rl *RateLimiter) SetLimit(clientID string, requestsPerMin, burstSize int) {
	// If rate limiting is disabled, do nothing
	if !rl.enabled {
		return
	}

	// Convert requests per minute to requests per second
	rps := float64(requestsPerMin) / 60.0

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// If client has a limiter, update it
	limiter, exists := rl.clientLimiters[clientID]
	if exists {
		limiter.limiter.SetLimit(rate.Limit(rps))
		limiter.limiter.SetBurst(burstSize)
		limiter.lastSeen = time.Now()
	} else {
		// Create a new limiter for this client
		rl.clientLimiters[clientID] = &ClientRateLimiter{
			limiter:  rate.NewLimiter(rate.Limit(rps), burstSize),
			lastSeen: time.Now(),
		}
	}
}

// Reset resets the rate limiter for a specific client
func (rl *RateLimiter) Reset(clientID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.clientLimiters, clientID)
}

// cleanup periodically removes expired limiters
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.removeExpiredLimiters()
		case <-rl.stopCleanup:
			return
		}
	}
}

// removeExpiredLimiters removes limiters that haven't been used for a while
func (rl *RateLimiter) removeExpiredLimiters() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for clientID, limiter := range rl.clientLimiters {
		if now.Sub(limiter.lastSeen) > rl.expirationTime {
			delete(rl.clientLimiters, clientID)
		}
	}
}

// Stop stops the rate limiter and its cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// GetRateLimitHeaders returns headers for rate limit information
func (rl *RateLimiter) GetRateLimitHeaders(clientID string) map[string]string {
	if !rl.enabled {
		return nil
	}

	rl.mu.RLock()
	limiter, exists := rl.clientLimiters[clientID]
	rl.mu.RUnlock()

	headers := make(map[string]string)
	if exists {
		// Get the current rate limit and tokens
		limit := int(float64(limiter.limiter.Limit()) * 60.0) // Convert to per minute
		tokensRemaining := limiter.limiter.Burst() - int(limiter.limiter.Tokens())

		headers["X-RateLimit-Limit"] = itoa(limit)
		headers["X-RateLimit-Remaining"] = itoa(int(tokensRemaining))
		headers["X-RateLimit-Reset"] = itoa(int(time.Now().Add(time.Minute).Unix()))
	}

	return headers
}

// itoa converts int to string without using strconv to avoid import
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	// Handle negative numbers
	var sign string
	if i < 0 {
		sign = "-"
		i = -i
	}

	// Convert to string
	var digits []byte
	for i > 0 {
		digits = append(digits, byte(i%10)+'0')
		i /= 10
	}

	// Reverse digits and add sign
	result := make([]byte, len(digits)+len(sign))
	copy(result, sign)
	for j := 0; j < len(digits); j++ {
		result[len(sign)+j] = digits[len(digits)-1-j]
	}

	return string(result)
}
