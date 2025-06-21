package middleware

import (
	"errors"
	"ftrack/models"
	"ftrack/utils"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	environment string
	logger      *logrus.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(environment string, logger *logrus.Logger) *ErrorHandler {
	return &ErrorHandler{
		environment: environment,
		logger:      logger,
	}
}

// Handle returns the error handling middleware
func (eh *ErrorHandler) Handle() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				eh.handlePanic(c, err)
			}
		}()

		c.Next()

		// Handle errors that were set during request processing
		if len(c.Errors) > 0 {
			eh.handleGinErrors(c)
		}
	})
}

// handlePanic handles panic recovery
func (eh *ErrorHandler) handlePanic(c *gin.Context, err interface{}) {
	// Log the panic with stack trace
	eh.logger.WithFields(logrus.Fields{
		"panic":      err,
		"stack":      string(debug.Stack()),
		"request_id": c.GetString("request_id"),
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"user_id":    c.GetString("userID"),
	}).Error("Panic recovered")

	// Create error response
	response := models.ErrorResponse{
		Error:     "INTERNAL_ERROR",
		Message:   "Internal server error",
		Code:      "PANIC_RECOVERED",
		RequestID: c.GetString("request_id"),
	}

	// Include stack trace in development
	if eh.environment == "development" {
		response.Details = map[string]interface{}{
			"panic": err,
			"stack": string(debug.Stack()),
		}
	}

	c.JSON(http.StatusInternalServerError, response)
	c.Abort()
}

// handleGinErrors handles errors added to gin context
func (eh *ErrorHandler) handleGinErrors(c *gin.Context) {
	// Get the last error (most recent)
	lastError := c.Errors.Last()
	if lastError == nil {
		return
	}

	// Log all errors
	for _, ginErr := range c.Errors {
		eh.logError(c, ginErr.Err)
	}

	// Process the main error
	eh.processError(c, lastError.Err)
}

// logError logs an error with context
func (eh *ErrorHandler) logError(c *gin.Context, err error) {
	fields := logrus.Fields{
		"error":      err.Error(),
		"request_id": c.GetString("request_id"),
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"user_id":    c.GetString("userID"),
		"ip":         c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
	}

	// Determine log level based on error type
	switch {
	case eh.isClientError(err):
		eh.logger.WithFields(fields).Warn("Client error")
	case eh.isServerError(err):
		eh.logger.WithFields(fields).Error("Server error")
	default:
		eh.logger.WithFields(fields).Error("Unknown error")
	}
}

// processError processes an error and sends appropriate response
func (eh *ErrorHandler) processError(c *gin.Context, err error) {
	requestID := c.GetString("request_id")

	// Handle specific error types
	switch {
	case eh.isValidationError(err):
		eh.handleValidationError(c, err, requestID)
	case eh.isMongoError(err):
		eh.handleMongoError(c, err, requestID)
	case eh.isCustomError(err):
		eh.handleCustomError(c, err, requestID)
	default:
		eh.handleGenericError(c, err, requestID)
	}
}

// isValidationError checks if error is a validation error
func (eh *ErrorHandler) isValidationError(err error) bool {
	var validationErr validator.ValidationErrors
	var customValidationErr *utils.ValidationError
	return errors.As(err, &validationErr) || errors.As(err, &customValidationErr)
}

// isMongoError checks if error is a MongoDB error
func (eh *ErrorHandler) isMongoError(err error) bool {
	return mongo.IsDuplicateKeyError(err) ||
		err == mongo.ErrNoDocuments ||
		mongo.IsTimeout(err) ||
		mongo.IsNetworkError(err)
}

// isCustomError checks if error is a custom application error
func (eh *ErrorHandler) isCustomError(err error) bool {
	var appErr *utils.AppError
	return errors.As(err, &appErr)
}

// isClientError checks if error is a client error (4xx)
func (eh *ErrorHandler) isClientError(err error) bool {
	return eh.isValidationError(err) ||
		err == mongo.ErrNoDocuments ||
		eh.isCustomError(err)
}

// isServerError checks if error is a server error (5xx)
func (eh *ErrorHandler) isServerError(err error) bool {
	return mongo.IsTimeout(err) ||
		mongo.IsNetworkError(err) ||
		(!eh.isClientError(err) && !eh.isCustomError(err))
}

// handleValidationError handles validation errors
func (eh *ErrorHandler) handleValidationError(c *gin.Context, err error, requestID string) {
	var validationErr validator.ValidationErrors
	var customValidationErr *utils.ValidationError

	if errors.As(err, &validationErr) {
		// Handle struct validation errors
		response := models.ErrorResponse{
			Error:     "VALIDATION_ERROR",
			Message:   "Validation failed",
			Code:      "VALIDATION_FAILED",
			RequestID: requestID,
			Details:   eh.formatValidationErrors(validationErr),
		}
		c.JSON(http.StatusBadRequest, response)
	} else if errors.As(err, &customValidationErr) {
		// Handle custom validation errors
		response := models.ErrorResponse{
			Error:     "VALIDATION_ERROR",
			Message:   customValidationErr.Message,
			Code:      "VALIDATION_FAILED",
			RequestID: requestID,
		}

		if customValidationErr.Fields != nil {
			response.Details = map[string]interface{}{
				"fields": customValidationErr.Fields,
			}
		}

		c.JSON(http.StatusBadRequest, response)
	}
}

// handleMongoError handles MongoDB errors
func (eh *ErrorHandler) handleMongoError(c *gin.Context, err error, requestID string) {
	switch {
	case mongo.IsDuplicateKeyError(err):
		response := models.ErrorResponse{
			Error:     "CONFLICT",
			Message:   "Resource already exists",
			Code:      "DUPLICATE_RESOURCE",
			RequestID: requestID,
		}
		c.JSON(http.StatusConflict, response)

	case err == mongo.ErrNoDocuments:
		response := models.ErrorResponse{
			Error:     "NOT_FOUND",
			Message:   "Resource not found",
			Code:      "RESOURCE_NOT_FOUND",
			RequestID: requestID,
		}
		c.JSON(http.StatusNotFound, response)

	case mongo.IsTimeout(err):
		response := models.ErrorResponse{
			Error:     "TIMEOUT",
			Message:   "Database operation timed out",
			Code:      "DATABASE_TIMEOUT",
			RequestID: requestID,
		}
		c.JSON(http.StatusGatewayTimeout, response)

	case mongo.IsNetworkError(err):
		response := models.ErrorResponse{
			Error:     "SERVICE_UNAVAILABLE",
			Message:   "Database connection error",
			Code:      "DATABASE_CONNECTION_ERROR",
			RequestID: requestID,
		}
		c.JSON(http.StatusServiceUnavailable, response)

	default:
		response := models.ErrorResponse{
			Error:     "INTERNAL_ERROR",
			Message:   "Database error",
			Code:      "DATABASE_ERROR",
			RequestID: requestID,
		}
		c.JSON(http.StatusInternalServerError, response)
	}
}

// handleCustomError handles custom application errors
func (eh *ErrorHandler) handleCustomError(c *gin.Context, err error, requestID string) {
	var appErr *utils.AppError
	if errors.As(err, &appErr) {
		response := models.ErrorResponse{
			Error:     appErr.Type,
			Message:   appErr.Message,
			Code:      appErr.Code,
			RequestID: requestID,
		}

		if appErr.Details != nil {
			response.Details = appErr.Details
		}

		c.JSON(appErr.StatusCode, response)
	}
}

// handleGenericError handles unknown errors
func (eh *ErrorHandler) handleGenericError(c *gin.Context, err error, requestID string) {
	response := models.ErrorResponse{
		Error:     "INTERNAL_ERROR",
		Message:   "An unexpected error occurred",
		Code:      "UNKNOWN_ERROR",
		RequestID: requestID,
	}

	// Include error details in development
	if eh.environment == "development" {
		response.Details = map[string]interface{}{
			"original_error": err.Error(),
		}
	}

	c.JSON(http.StatusInternalServerError, response)
}

// formatValidationErrors formats validator.ValidationErrors into a readable format
func (eh *ErrorHandler) formatValidationErrors(validationErrors validator.ValidationErrors) map[string]interface{} {
	errors := make(map[string]interface{})

	for _, err := range validationErrors {
		field := err.Field()
		tag := err.Tag()

		var message string
		switch tag {
		case "required":
			message = "This field is required"
		case "email":
			message = "Must be a valid email address"
		case "min":
			message = "Value is too short"
		case "max":
			message = "Value is too long"
		case "len":
			message = "Invalid length"
		case "oneof":
			message = "Invalid value"
		case "url":
			message = "Must be a valid URL"
		case "uuid":
			message = "Must be a valid UUID"
		default:
			message = "Invalid value"
		}

		errors[field] = map[string]interface{}{
			"message": message,
			"tag":     tag,
			"value":   err.Value(),
		}
	}

	return map[string]interface{}{
		"fields": errors,
	}
}

// AbortWithError aborts the request with an error
func AbortWithError(c *gin.Context, statusCode int, errorType, message, code string) {
	response := models.ErrorResponse{
		Error:     errorType,
		Message:   message,
		Code:      code,
		RequestID: c.GetString("request_id"),
	}
	c.JSON(statusCode, response)
	c.Abort()
}

// AbortWithCustomError aborts the request with a custom error
func AbortWithCustomError(c *gin.Context, err *utils.AppError) {
	response := models.ErrorResponse{
		Error:     err.Type,
		Message:   err.Message,
		Code:      err.Code,
		RequestID: c.GetString("request_id"),
		Details:   err.Details,
	}
	c.JSON(err.StatusCode, response)
	c.Abort()
}

// Helper functions for common errors

// NotFound responds with 404 error
func NotFound(c *gin.Context, message string) {
	AbortWithError(c, http.StatusNotFound, "NOT_FOUND", message, "RESOURCE_NOT_FOUND")
}

// BadRequest responds with 400 error
func BadRequest(c *gin.Context, message string) {
	AbortWithError(c, http.StatusBadRequest, "BAD_REQUEST", message, "INVALID_REQUEST")
}

// Unauthorized responds with 401 error
func Unauthorized(c *gin.Context, message string) {
	AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", message, "AUTHENTICATION_REQUIRED")
}

// Forbidden responds with 403 error
func Forbidden(c *gin.Context, message string) {
	AbortWithError(c, http.StatusForbidden, "FORBIDDEN", message, "INSUFFICIENT_PERMISSIONS")
}

// InternalError responds with 500 error
func InternalError(c *gin.Context, message string) {
	AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, "INTERNAL_SERVER_ERROR")
}
