package transport

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/service"
)

// ResponseWriter wraps http.ResponseWriter to capture status code
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the size of the response
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Capture client IP
		clientIP := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = strings.Split(xff, ",")[0]
		} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			clientIP = xrip
		}

		// Log request start
		log.Printf("HTTP Request: method=%s path=%s client_ip=%s",
			r.Method, r.URL.Path, clientIP)

		// Wrap response writer to capture status code
		rw := NewResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		log.Printf("HTTP Response: method=%s path=%s status=%d size=%d duration=%s client_ip=%s",
			r.Method, r.URL.Path, rw.statusCode, rw.size, duration, clientIP)
	})
}

// RecoveryMiddleware recovers from panics in HTTP handlers
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic in HTTP handler: %v\n%s", err, debug.Stack())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

		// Handle OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware checks authentication for HTTP requests
func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for certain paths
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from request
			token, err := authService.ExtractTokenFromRequest(r)
			if err != nil {
				// Try API key if token extraction failed
				if authService.Config().EnableAPIKeys {
					apiKey, err := authService.ExtractAPIKeyFromRequest(r)
					if err == nil {
						// Validate API key
						key, err := authService.ValidateAPIKey(apiKey)
						if err == nil {
							// Add user ID to context
							ctx := context.WithValue(r.Context(), "user_id", key.UserID)
							ctx = context.WithValue(ctx, "permissions", key.Permissions)

							// Call next handler with updated context
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}

				// If we get here, authentication failed
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
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "role", claims.Role)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimitMiddleware applies rate limiting to HTTP requests
func RateLimitMiddleware(rateLimiter *service.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for certain paths
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract client ID for rate limiting
			clientID := extractClientIDFromHTTP(r)

			// Check rate limit
			err := rateLimiter.Allow(clientID)
			if err != nil {
				// Add rate limit headers
				for key, value := range rateLimiter.GetRateLimitHeaders(clientID) {
					w.Header().Set(key, value)
				}

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Add rate limit headers
			for key, value := range rateLimiter.GetRateLimitHeaders(clientID) {
				w.Header().Set(key, value)
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIDFromHTTP extracts a client identifier for rate limiting from HTTP request
func extractClientIDFromHTTP(r *http.Request) string {
	// Try to get user_id from context
	if userID, ok := r.Context().Value("user_id").(string); ok && userID != "" {
		return userID
	}

	// If no user_id, try to get from client IP
	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = strings.Split(xff, ",")[0]
	} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		clientIP = xrip
	}

	return clientIP
}

// isPublicPath checks if a path is public (doesn't require authentication)
func isPublicPath(path string) bool {
	// List of public paths that don't require authentication
	publicPaths := []string{
		"/health",
		"/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
	}

	// Check if path is in the list
	for _, publicPath := range publicPaths {
		if path == publicPath {
			return true
		}
	}

	return false
}

// WebSocketAuth middleware for WebSocket authentication
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
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "role", claims.Role)

			// Call next handler with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
