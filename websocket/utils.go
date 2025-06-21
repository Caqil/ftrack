package websocket

import (
	"ftrack/models"
	"ftrack/utils"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocket upgrader configuration
var DefaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		origin := r.Header.Get("Origin")
		logrus.Debugf("WebSocket connection from origin: %s", origin)
		return true // Allow all origins for now
	},
}

// validateWebSocketMessage validates incoming WebSocket message structure
func validateWebSocketMessage(msg models.WSRequest) error {
	if msg.Type == "" {
		return utils.NewValidationError("Message type is required")
	}

	// Validate specific message types
	switch msg.Type {
	case models.WSRequestLocationUpdate:
		if msg.Data == nil {
			return utils.NewValidationError("Location data is required")
		}
	case models.WSRequestSendMessage:
		if msg.Data == nil {
			return utils.NewValidationError("Message data is required")
		}
	case models.WSRequestEmergencyAlert:
		if msg.Data == nil {
			return utils.NewValidationError("Emergency data is required")
		}
	}

	return nil
}

// createSuccessResponse creates a standardized success response
func createSuccessResponse(message string, data interface{}, requestID string) models.WSMessage {
	responseData := map[string]interface{}{
		"success": true,
		"message": message,
	}

	if data != nil {
		responseData["data"] = data
	}

	if requestID != "" {
		responseData["requestId"] = requestID
	}

	return models.WSMessage{
		Type:      models.WSTypeSuccess,
		Data:      responseData,
		Timestamp: time.Now(),
	}
}

// createErrorResponse creates a standardized error response
func createErrorResponse(code, message string, requestID string) models.WSMessage {
	errorData := models.WSError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}

	responseData := map[string]interface{}{
		"success": false,
		"error":   errorData,
	}

	if requestID != "" {
		responseData["requestId"] = requestID
	}

	return models.WSMessage{
		Type:      models.WSTypeError,
		Data:      responseData,
		Timestamp: time.Now(),
	}
}

// filterMessage checks if a message should be sent to a specific client based on filters
func filterMessage(client *Client, message models.WSMessage, filter MessageFilter) bool {
	if client == nil || !client.isActive {
		return false
	}

	// Check if client is excluded
	for _, userID := range filter.ExcludeUsers {
		if client.userID == userID {
			return false
		}
	}

	// Check if client is in include list (if specified)
	if len(filter.IncludeUsers) > 0 {
		found := false
		for _, userID := range filter.IncludeUsers {
			if client.userID == userID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check event type filter
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if message.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// sanitizeMessageData removes sensitive information from message data
func sanitizeMessageData(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	// Use reflection to recursively sanitize data
	value := reflect.ValueOf(data)
	return sanitizeValue(value)
}

// sanitizeValue recursively sanitizes reflect.Value
func sanitizeValue(value reflect.Value) interface{} {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}
		return sanitizeValue(value.Elem())

	case reflect.Map:
		result := make(map[string]interface{})
		for _, key := range value.MapKeys() {
			keyStr := key.String()

			// Skip sensitive fields
			if isSensitiveField(keyStr) {
				continue
			}

			result[keyStr] = sanitizeValue(value.MapIndex(key))
		}
		return result

	case reflect.Slice, reflect.Array:
		length := value.Len()
		result := make([]interface{}, length)
		for i := 0; i < length; i++ {
			result[i] = sanitizeValue(value.Index(i))
		}
		return result

	case reflect.Struct:
		result := make(map[string]interface{})
		valueType := value.Type()

		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			fieldValue := value.Field(i)

			// Skip unexported fields
			if !fieldValue.CanInterface() {
				continue
			}

			// Skip sensitive fields
			if isSensitiveField(field.Name) {
				continue
			}

			// Get JSON tag name if available
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				// Parse JSON tag (handle omitempty, etc.)
				tagParts := strings.Split(jsonTag, ",")
				if len(tagParts) > 0 && tagParts[0] != "" {
					result[tagParts[0]] = sanitizeValue(fieldValue)
				}
			} else {
				result[field.Name] = sanitizeValue(fieldValue)
			}
		}
		return result

	default:
		return value.Interface()
	}
}

// isSensitiveField checks if a field contains sensitive information
func isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{
		"password", "token", "secret", "key", "auth",
		"private", "confidential", "secure", "credential",
	}

	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

// logWebSocketError logs WebSocket errors with context
func logWebSocketError(client *Client, operation string, err error) {
	logrus.WithFields(logrus.Fields{
		"userID":       client.userID,
		"connectionID": client.connectionID,
		"operation":    operation,
		"error":        err.Error(),
	}).Error("WebSocket operation failed")
}

// logWebSocketEvent logs WebSocket events for debugging
func logWebSocketEvent(client *Client, eventType string, data interface{}) {
	if logrus.GetLevel() == logrus.DebugLevel {
		logrus.WithFields(logrus.Fields{
			"userID":       client.userID,
			"connectionID": client.connectionID,
			"eventType":    eventType,
		}).Debug("WebSocket event processed")
	}
}
