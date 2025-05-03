package middleware

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
)

// Logger is middleware that logs request information
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create wrapper for response writer to capture status code
		ww := NewResponseWriter(w)

		// Process request
		next.ServeHTTP(ww, r)

		// Calculate duration
		duration := time.Since(start)

		// Extract client IP
		clientIP := extractClientIP(r)

		// Log request details
		log.Printf(
			"HTTP %s %s %d %s %s",
			r.Method,
			r.URL.Path,
			ww.Status(),
			duration,
			clientIP,
		)
	})
}

// Recovery is middleware that recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the stack trace
				log.Printf("PANIC: %v\n%s", err, debug.Stack())

				// Return Internal Server Error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Tracing is middleware that adds tracing headers
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID if not present
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			r.Header.Set("X-Request-ID", requestID)
		}

		// Set response header
		w.Header().Set("X-Request-ID", requestID)

		// Add to request context
		ctx := r.Context()
		ctx = AddValue(ctx, "request_id", requestID)

		// Process request with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORS is middleware that handles Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Auth is middleware that checks authentication
func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for public paths
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Try to get JWT token
			token, err := authService.ExtractTokenFromRequest(r)
			if err != nil {
				// Try API key if JWT fails
				if authService.Config().EnableAPIKeys {
					apiKey, err := authService.ExtractAPIKeyFromRequest(r)
					if err == nil {
						key, err := authService.ValidateAPIKey(apiKey)
						if err == nil {
							// Add claims to context
							ctx := r.Context()
							ctx = AddValue(ctx, "user_id", key.UserID)
							ctx = AddValue(ctx, "permissions", key.Permissions)

							// Process request with updated context
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}

				// Both JWT and API key failed
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Validate JWT token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := r.Context()
			ctx = AddValue(ctx, "user_id", claims.UserID)
			ctx = AddValue(ctx, "role", claims.Role)

			// Process request with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimit is middleware that applies rate limiting
func RateLimit(rateLimiter *service.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for public paths
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract client ID for rate limiting
			clientID := extractRateLimitClientID(r)

			// Check rate limit
			err := rateLimiter.Allow(clientID)
			if err != nil {
				// Set rate limit headers
				for k, v := range rateLimiter.GetRateLimitHeaders(clientID) {
					w.Header().Set(k, v)
				}

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Set rate limit headers
			for k, v := range rateLimiter.GetRateLimitHeaders(clientID) {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WebSocketAuth is middleware for WebSocket authentication
func WebSocketAuth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from request
			token, err := authService.ExtractTokenFromRequest(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := r.Context()
			ctx = AddValue(ctx, "user_id", claims.UserID)
			ctx = AddValue(ctx, "role", claims.Role)

			// Process request with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResponseWriter is a wrapper for http.ResponseWriter that captures status code
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}
}

// WriteHeader captures status code
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.written = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures that response was written
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	rw.written = true
	return rw.ResponseWriter.Write(b)
}

// Status returns the HTTP status code
func (rw *ResponseWriter) Status() int {
	return rw.statusCode
}

// isPublicPath checks if a path is public (doesn't require authentication)
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/health",
		"/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
		"/api/v1/public/",
	}

	for _, pp := range publicPaths {
		if pp == path || (strings.HasSuffix(pp, "/") && strings.HasPrefix(path, pp)) {
			return true
		}
	}

	return false
}

// extractClientIP extracts client IP from request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For might contain multiple IPs, use the leftmost (client)
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Fallback to remote address
	return r.RemoteAddr
}

// extractRateLimitClientID extracts client ID for rate limiting
func extractRateLimitClientID(r *http.Request) string {
	// First try to use user ID from context
	if userID, ok := r.Context().Value("user_id").(string); ok && userID != "" {
		return "user:" + userID
	}

	// Otherwise use client IP
	return "ip:" + extractClientIP(r)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// In a real application, use a proper UUID library
	return "req_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of the given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Add some "randomness"
	}
	return string(result)
}

// AddValue adds a value to context (helper function)
func AddValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}
