package models

import "time"

// ErrorResponse represents a standardized error response structure
type ErrorResponse struct {
	// Error represents the error type or category
	Error string `json:"error" example:"RATE_LIMIT_EXCEEDED"`

	// Message provides a human-readable description of the error
	Message string `json:"message" example:"Rate limit exceeded. Please try again later."`

	// Code represents the HTTP or application-specific error code
	Code string `json:"code" example:"TOO_MANY_REQUESTS"`

	// RequestID is a unique identifier for tracking the request
	RequestID string `json:"request_id" example:"req_123456789"`

	// Details contains additional context-specific information about the error
	Details map[string]interface{} `json:"details,omitempty"`

	// Timestamp indicates when the error occurred
	Timestamp time.Time `json:"timestamp" example:"2023-12-07T10:30:00Z"`
}

// NewErrorResponse creates a new ErrorResponse with the current timestamp
func NewErrorResponse(error, message, code, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Error:     error,
		Message:   message,
		Code:      code,
		RequestID: requestID,
		Details:   make(map[string]interface{}),
		Timestamp: time.Now().UTC(),
	}
}

// WithDetails adds additional details to the error response
func (e *ErrorResponse) WithDetails(key string, value interface{}) *ErrorResponse {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithMultipleDetails adds multiple details to the error response
func (e *ErrorResponse) WithMultipleDetails(details map[string]interface{}) *ErrorResponse {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// Common error types
const (
	ErrorTypeValidation     = "VALIDATION_ERROR"
	ErrorTypeAuthentication = "AUTHENTICATION_ERROR"
	ErrorTypeAuthorization  = "AUTHORIZATION_ERROR"
	ErrorTypeNotFound       = "NOT_FOUND"
	ErrorTypeRateLimit      = "RATE_LIMIT_EXCEEDED"
	ErrorTypeInternal       = "INTERNAL_SERVER_ERROR"
	ErrorTypeBadRequest     = "BAD_REQUEST"
	ErrorTypeConflict       = "CONFLICT"
)

// Common error codes
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeTooManyRequests     = "TOO_MANY_REQUESTS"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeValidationFailed    = "VALIDATION_FAILED"
)
