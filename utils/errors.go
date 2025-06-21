package utils

import (
	"fmt"
	"net/http"
)

// ServiceError represents a service-level error with context
type ServiceError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode,omitempty"`
	Details    string `json:"details,omitempty"`
	Cause      error  `json:"-"` // Original error, not exposed in JSON
}

func (e ServiceError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e ServiceError) Unwrap() error {
	return e.Cause
}

// NewServiceError creates a new service error
func NewServiceError(code, message string) error {
	return ServiceError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

// NewServiceErrorWithStatus creates a service error with specific HTTP status
func NewServiceErrorWithStatus(code, message string, statusCode int) error {
	return ServiceError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewServiceErrorWithDetails creates a service error with additional details
func NewServiceErrorWithDetails(code, message, details string) error {
	return ServiceError{
		Code:       code,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusInternalServerError,
	}
}

// NewServiceErrorWithCause creates a service error that wraps another error
func NewServiceErrorWithCause(code, message string, cause error) error {
	return ServiceError{
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
	}
}

// IsServiceError checks if an error is a service error
func IsServiceError(err error) bool {
	_, ok := err.(ServiceError)
	return ok
}

// GetServiceError extracts a ServiceError from an error
func GetServiceError(err error) (ServiceError, bool) {
	if serviceErr, ok := err.(ServiceError); ok {
		return serviceErr, true
	}
	return ServiceError{}, false
}

// Common service error constructors
func NewUnauthorizedError(message string) error {
	return ServiceError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewForbiddenError(message string) error {
	return ServiceError{
		Code:       "FORBIDDEN",
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

func NewNotFoundError(resource string) error {
	return ServiceError{
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
	}
}

func NewBadRequestError(message string) error {
	return ServiceError{
		Code:       "BAD_REQUEST",
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NewConflictError(message string) error {
	return ServiceError{
		Code:       "CONFLICT",
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

func NewInternalError(message string) error {
	return ServiceError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewDatabaseError(operation string, cause error) error {
	return ServiceError{
		Code:       "DATABASE_ERROR",
		Message:    fmt.Sprintf("Database operation failed: %s", operation),
		Cause:      cause,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewNetworkError(message string, cause error) error {
	return ServiceError{
		Code:       "NETWORK_ERROR",
		Message:    message,
		Cause:      cause,
		StatusCode: http.StatusServiceUnavailable,
	}
}

func NewRateLimitError(message string) error {
	return ServiceError{
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
	}
}

// Business logic specific errors
func NewCircleNotFoundError() error {
	return NewNotFoundError("Circle")
}

func NewUserNotFoundError() error {
	return NewNotFoundError("User")
}

func NewEmergencyNotFoundError() error {
	return NewNotFoundError("Emergency")
}

func NewMessageNotFoundError() error {
	return NewNotFoundError("Message")
}

func NewPlaceNotFoundError() error {
	return NewNotFoundError("Place")
}

func NewInvalidCredentialsError() error {
	return NewUnauthorizedError("Invalid credentials")
}

func NewTokenExpiredError() error {
	return NewUnauthorizedError("Token has expired")
}

func NewInsufficientPermissionsError() error {
	return NewForbiddenError("Insufficient permissions")
}

func NewCircleFullError() error {
	return NewConflictError("Circle has reached maximum member limit")
}

func NewAlreadyMemberError() error {
	return NewConflictError("User is already a member of this circle")
}

func NewLocationServiceError(message string) error {
	return NewServiceError("LOCATION_SERVICE_ERROR", message)
}

func NewNotificationServiceError(message string) error {
	return NewServiceError("NOTIFICATION_SERVICE_ERROR", message)
}

func NewWebSocketError(message string) error {
	return NewServiceError("WEBSOCKET_ERROR", message)
}

// Error handling helpers
func WrapError(err error, code, message string) error {
	return ServiceError{
		Code:       code,
		Message:    message,
		Cause:      err,
		StatusCode: http.StatusInternalServerError,
	}
}


func WrapDatabaseError(err error, operation string) error {
	return NewDatabaseError(operation, err)
}

// Error code constants
const (
	ErrCodeValidation          = "VALIDATION_ERROR"
	ErrCodeAuthentication      = "AUTHENTICATION_ERROR"
	ErrCodeAuthorization       = "AUTHORIZATION_ERROR"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeRateLimit           = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternal            = "INTERNAL_ERROR"
	ErrCodeDatabase            = "DATABASE_ERROR"
	ErrCodeNetwork             = "NETWORK_ERROR"
	ErrCodeWebSocket           = "WEBSOCKET_ERROR"
	ErrCodeLocationService     = "LOCATION_SERVICE_ERROR"
	ErrCodeNotificationService = "NOTIFICATION_SERVICE_ERROR"
)

// Common error instances
var (
	ErrServiceUnavailable = NewServiceError("SERVICE_UNAVAILABLE", "Service is temporarily unavailable")
	ErrMaintenanceMode    = NewServiceError("MAINTENANCE_MODE", "Service is in maintenance mode")
	ErrInvalidRequest     = NewBadRequestError("Invalid request")
	ErrAccessDenied       = NewForbiddenError("Access denied")
	ErrResourceNotFound   = NewNotFoundError("Resource")
)
