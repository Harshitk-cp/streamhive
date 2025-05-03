package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Harshitk-cp/streamhive/apps/api-gateway/internal/config"
	"github.com/golang-jwt/jwt/v4"
)

var (
	// ErrTokenExpired indicates that the token has expired
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidToken indicates that the token is invalid
	ErrInvalidToken = errors.New("invalid token")

	// ErrInvalidAPIKey indicates that the API key is invalid
	ErrInvalidAPIKey = errors.New("invalid API key")

	// ErrAuthRequired indicates that authentication is required
	ErrAuthRequired = errors.New("authentication required")
)

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// APIKey represents an API key with its associated permissions
type APIKey struct {
	Key         string
	UserID      string
	Permissions []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// AuthService handles authentication and authorization
type AuthService struct {
	config       config.AuthConfig
	jwtSecret    []byte
	apiKeys      map[string]APIKey
	apiKeysMutex sync.RWMutex
}

// Config returns the AuthConfig used by the AuthService
func (s *AuthService) Config() config.AuthConfig {
	return s.config
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg config.AuthConfig) *AuthService {
	return &AuthService{
		config:    cfg,
		jwtSecret: []byte(cfg.JWTSecret),
		apiKeys:   make(map[string]APIKey),
	}
}

// GenerateToken generates a new JWT token for a user
func (s *AuthService) GenerateToken(userID, role string) (string, error) {
	expirationTime := time.Now().Add(s.config.JWTExpiration)

	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "streaming-platform-api-gateway",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, ErrTokenExpired
			}
		}
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExtractTokenFromRequest extracts JWT token from HTTP request
func (s *AuthService) ExtractTokenFromRequest(r *http.Request) (string, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Check if it's a Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer "), nil
		}
	}

	// Extract token from cookie
	cookie, err := r.Cookie("jwt")
	if err == nil {
		return cookie.Value, nil
	}

	// Extract token from query parameter
	token := r.URL.Query().Get("token")
	if token != "" {
		return token, nil
	}

	return "", ErrAuthRequired
}

// RegisterAPIKey registers a new API key
func (s *AuthService) RegisterAPIKey(userID string, permissions []string, expiresIn time.Duration) (string, error) {
	if !s.config.EnableAPIKeys {
		return "", errors.New("API keys are not enabled")
	}

	// Generate a random API key
	apiKey := generateRandomString(32)

	// Create API key with expiration
	key := APIKey{
		Key:         apiKey,
		UserID:      userID,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(expiresIn),
	}

	// Store API key
	s.apiKeysMutex.Lock()
	s.apiKeys[apiKey] = key
	s.apiKeysMutex.Unlock()

	return apiKey, nil
}

// ValidateAPIKey validates an API key
func (s *AuthService) ValidateAPIKey(apiKey string) (*APIKey, error) {
	if !s.config.EnableAPIKeys {
		return nil, errors.New("API keys are not enabled")
	}

	s.apiKeysMutex.RLock()
	key, exists := s.apiKeys[apiKey]
	s.apiKeysMutex.RUnlock()

	if !exists {
		return nil, ErrInvalidAPIKey
	}

	// Check if API key is expired
	if time.Now().After(key.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	return &key, nil
}

// ExtractAPIKeyFromRequest extracts API key from HTTP request
func (s *AuthService) ExtractAPIKeyFromRequest(r *http.Request) (string, error) {
	if !s.config.EnableAPIKeys {
		return "", errors.New("API keys are not enabled")
	}

	// Extract API key from header
	apiKey := r.Header.Get(s.config.APIKeyHeaderName)
	if apiKey != "" {
		return apiKey, nil
	}

	// Extract API key from query parameter
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return apiKey, nil
	}

	return "", ErrAuthRequired
}

// RevokeAPIKey revokes an API key
func (s *AuthService) RevokeAPIKey(apiKey string) error {
	if !s.config.EnableAPIKeys {
		return errors.New("API keys are not enabled")
	}

	s.apiKeysMutex.Lock()
	delete(s.apiKeys, apiKey)
	s.apiKeysMutex.Unlock()

	return nil
}

// CheckPermission checks if the user has the required permission
func (s *AuthService) CheckPermission(userID, permission string) bool {
	// In a real implementation, you would check against a database or permission service
	// This is a simplified example

	// TODO: Implement proper permission checking
	return true
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	// In a real implementation, you would use crypto/rand
	// This is a simplified example for brevity
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		// This is not cryptographically secure - use crypto/rand in production
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Add some "randomness"
	}
	return string(result)
}
