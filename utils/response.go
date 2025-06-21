package utils

import (
	"ftrack/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Success responses
func SuccessResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

func SuccessResponseWithMeta(c *gin.Context, message string, data interface{}, meta *models.MetaData) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now(),
	})
}

func CreatedResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, models.APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// Error responses
func ErrorResponse(c *gin.Context, statusCode int, message string, details interface{}) {
	c.JSON(statusCode, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    getErrorCode(statusCode),
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	})
}

func ValidationErrorResponse(c *gin.Context, validationErrors []ValidationError) {
	c.JSON(http.StatusBadRequest, models.APIResponse{
		Success: false,
		Message: "Validation failed",
		Error: &models.APIError{
			Code:    models.ErrCodeValidation,
			Message: "Validation failed",
			Details: validationErrors,
		},
		Timestamp: time.Now(),
	})
}

func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized access"
	}
	c.JSON(http.StatusUnauthorized, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeAuthentication,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Access forbidden"
	}
	c.JSON(http.StatusForbidden, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeAuthorization,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func NotFoundResponse(c *gin.Context, resource string) {
	message := resource + " not found"
	c.JSON(http.StatusNotFound, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeNotFound,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func ConflictResponse(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeConflict,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func RateLimitResponse(c *gin.Context) {
	message := "Rate limit exceeded"
	c.JSON(http.StatusTooManyRequests, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeRateLimit,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func InternalServerErrorResponse(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	c.JSON(http.StatusInternalServerError, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeInternal,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func ServiceUnavailableResponse(c *gin.Context, service string) {
	message := service + " service is currently unavailable"
	c.JSON(http.StatusServiceUnavailable, models.APIResponse{
		Success: false,
		Message: message,
		Error: &models.APIError{
			Code:    models.ErrCodeExternal,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

// WebSocket responses
func WSSuccessResponse(requestID string, data interface{}) models.WSResponse {
	return models.WSResponse{
		Type:      "success",
		Data:      data,
		Success:   true,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
}

func WSErrorResponse(requestID string, errorMsg string) models.WSResponse {
	return models.WSResponse{
		Type:      "error",
		Success:   false,
		Error:     errorMsg,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
}

// Helper functions
func getErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return models.ErrCodeValidation
	case http.StatusUnauthorized:
		return models.ErrCodeAuthentication
	case http.StatusForbidden:
		return models.ErrCodeAuthorization
	case http.StatusNotFound:
		return models.ErrCodeNotFound
	case http.StatusConflict:
		return models.ErrCodeConflict
	case http.StatusTooManyRequests:
		return models.ErrCodeRateLimit
	case http.StatusInternalServerError:
		return models.ErrCodeInternal
	case http.StatusServiceUnavailable:
		return models.ErrCodeExternal
	default:
		return models.ErrCodeInternal
	}
}

func CreatePaginationMeta(page, pageSize int, total int64) *models.MetaData {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.MetaData{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// BadRequestResponse sends a 400 Bad Request response
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message, nil)
}

// TooManyRequestsResponse sends a 429 Too Many Requests response
func TooManyRequestsResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusTooManyRequests, message, nil)
}

// AcceptedResponse sends a 202 Accepted response
func AcceptedResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusAccepted, models.APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// NoContentResponse sends a 204 No Content response
func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// HealthCheckResponse creates a health check response
func HealthCheckResponse(services map[string]string, version, uptime string) models.HealthResponse {
	status := "healthy"
	for _, serviceStatus := range services {
		if serviceStatus != "healthy" {
			status = "unhealthy"
			break
		}
	}

	return models.HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
		Version:   version,
		Uptime:    uptime,
	}
}
