package model

import (
	"encoding/json"
	"fmt"
)

// APIError represents an API error response
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// MarshalJSON implements json.Marshaler
func (e APIError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details,omitempty"`
	}{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	})
}

// Common API errors
var (
	// ErrInvalidRequest is returned when the request is invalid
	ErrInvalidRequest = APIError{
		Status:  400,
		Code:    "invalid_request",
		Message: "The request is invalid",
	}

	// ErrUnauthorized is returned when the user is not authenticated
	ErrUnauthorized = APIError{
		Status:  401,
		Code:    "unauthorized",
		Message: "Authentication is required",
	}

	// ErrForbidden is returned when the user doesn't have permission
	ErrForbidden = APIError{
		Status:  403,
		Code:    "forbidden",
		Message: "You don't have permission to access this resource",
	}

	// ErrNotFound is returned when a resource is not found
	ErrNotFound = APIError{
		Status:  404,
		Code:    "not_found",
		Message: "The requested resource was not found",
	}

	// ErrConflict is returned when a conflict occurs
	ErrConflict = APIError{
		Status:  409,
		Code:    "conflict",
		Message: "A conflict occurred with the current state of the resource",
	}

	// ErrTooManyRequests is returned when rate limit is exceeded
	ErrTooManyRequests = APIError{
		Status:  429,
		Code:    "too_many_requests",
		Message: "Rate limit exceeded",
	}

	// ErrInternalServer is returned when an internal server error occurs
	ErrInternalServer = APIError{
		Status:  500,
		Code:    "internal_server_error",
		Message: "An internal server error occurred",
	}

	// ErrServiceUnavailable is returned when the service is unavailable
	ErrServiceUnavailable = APIError{
		Status:  503,
		Code:    "service_unavailable",
		Message: "The service is currently unavailable",
	}
)

// NewAPIError creates a new API error with the given status, code, and message
func NewAPIError(status int, code, message string) APIError {
	return APIError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to an API error
func (e APIError) WithDetails(details string) APIError {
	e.Details = details
	return e
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	if apiErr, ok := err.(APIError); ok {
		return apiErr.Status == 404
	}
	return false
}

// IsForbidden checks if an error is a forbidden error
func IsForbidden(err error) bool {
	if apiErr, ok := err.(APIError); ok {
		return apiErr.Status == 403
	}
	return false
}

// IsConflict checks if an error is a conflict error
func IsConflict(err error) bool {
	if apiErr, ok := err.(APIError); ok {
		return apiErr.Status == 409
	}
	return false
}
