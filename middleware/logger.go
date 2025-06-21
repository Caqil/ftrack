package middleware

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Logger             *logrus.Logger
	EnableRequestBody  bool
	EnableResponseBody bool
	MaxBodySize        int64
	SkipPaths          []string
	SkipUserAgents     []string
}

// LoggerMiddleware returns a logger middleware with configuration
func LoggerMiddleware(config LoggerConfig) gin.HandlerFunc {
	if config.Logger == nil {
		config.Logger = logrus.StandardLogger()
	}

	if config.MaxBodySize == 0 {
		config.MaxBodySize = 4096 // 4KB default
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		// Generate request ID if not exists
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Skip logging for certain paths
		if shouldSkipPath(c.Request.URL.Path, config.SkipPaths) {
			c.Next()
			return
		}

		// Skip logging for certain user agents
		userAgent := c.GetHeader("User-Agent")
		if shouldSkipUserAgent(userAgent, config.SkipUserAgents) {
			c.Next()
			return
		}

		// Start timer
		startTime := time.Now()

		// Capture request body if enabled
		var requestBody []byte
		if config.EnableRequestBody && c.Request.Body != nil {
			requestBody = captureRequestBody(c, config.MaxBodySize)
		}

		// Capture response if enabled
		var responseBody *bytes.Buffer
		if config.EnableResponseBody {
			responseBody = &bytes.Buffer{}
			c.Writer = &responseBodyWriter{
				ResponseWriter: c.Writer,
				body:           responseBody,
				maxSize:        config.MaxBodySize,
			}
		}

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(startTime)

		// Create log fields
		fields := createLogFields(c, duration, requestID, requestBody, responseBody)

		// Log the request
		logRequest(config.Logger, c.Writer.Status(), duration, fields)
	})
}

// DefaultLoggerMiddleware returns a logger middleware with default configuration
func DefaultLoggerMiddleware() gin.HandlerFunc {
	return LoggerMiddleware(LoggerConfig{
		Logger:             logrus.StandardLogger(),
		EnableRequestBody:  false,
		EnableResponseBody: false,
		MaxBodySize:        4096,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
		SkipUserAgents: []string{
			"kube-probe",
			"GoogleHC",
		},
	})
}

// DevelopmentLoggerMiddleware returns a verbose logger for development
func DevelopmentLoggerMiddleware() gin.HandlerFunc {
	return LoggerMiddleware(LoggerConfig{
		Logger:             logrus.StandardLogger(),
		EnableRequestBody:  true,
		EnableResponseBody: true,
		MaxBodySize:        8192,
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	})
}

// ProductionLoggerMiddleware returns a production-safe logger
func ProductionLoggerMiddleware() gin.HandlerFunc {
	return LoggerMiddleware(LoggerConfig{
		Logger:             logrus.StandardLogger(),
		EnableRequestBody:  false,
		EnableResponseBody: false,
		MaxBodySize:        1024,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
		SkipUserAgents: []string{
			"kube-probe",
			"GoogleHC",
			"Amazon-Route53-Health-Check-Service",
		},
	})
}

// responseBodyWriter captures response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	maxSize int64
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	// Write to original response
	n, err := w.ResponseWriter.Write(b)

	// Capture for logging if under max size
	if w.body.Len() < int(w.maxSize) {
		remaining := int(w.maxSize) - w.body.Len()
		if len(b) <= remaining {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remaining])
		}
	}

	return n, err
}

// captureRequestBody safely captures request body
func captureRequestBody(c *gin.Context, maxSize int64) []byte {
	if c.Request.Body == nil {
		return nil
	}

	// Read body
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxSize))
	if err != nil {
		return nil
	}

	// Restore body for further processing
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	return body
}

// createLogFields creates structured log fields
func createLogFields(c *gin.Context, duration time.Duration, requestID string, requestBody []byte, responseBody *bytes.Buffer) logrus.Fields {
	fields := logrus.Fields{
		"request_id":     requestID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
		"query":          c.Request.URL.RawQuery,
		"status":         c.Writer.Status(),
		"latency":        duration.String(),
		"latency_ms":     float64(duration.Nanoseconds()) / 1000000.0,
		"ip":             c.ClientIP(),
		"user_agent":     c.GetHeader("User-Agent"),
		"content_length": c.Request.ContentLength,
		"response_size":  c.Writer.Size(),
	}

	// Add user information if available
	if userID := c.GetString("userID"); userID != "" {
		fields["user_id"] = userID
	}

	if userEmail := c.GetString("userEmail"); userEmail != "" {
		fields["user_email"] = userEmail
	}

	// Add request headers (selective)
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if shouldLogHeader(key) {
			headers[key] = strings.Join(values, ", ")
		}
	}
	if len(headers) > 0 {
		fields["headers"] = headers
	}

	// Add request body if captured
	if requestBody != nil && len(requestBody) > 0 {
		if isTextContent(c.GetHeader("Content-Type")) {
			fields["request_body"] = string(requestBody)
		} else {
			fields["request_body_size"] = len(requestBody)
		}
	}

	// Add response body if captured
	if responseBody != nil && responseBody.Len() > 0 {
		if isTextContent(c.GetHeader("Content-Type")) {
			fields["response_body"] = responseBody.String()
		} else {
			fields["response_body_size"] = responseBody.Len()
		}
	}

	// Add error information if any
	if len(c.Errors) > 0 {
		errors := make([]string, len(c.Errors))
		for i, err := range c.Errors {
			errors[i] = err.Error()
		}
		fields["errors"] = errors
	}

	return fields
}

// logRequest logs the HTTP request
func logRequest(logger *logrus.Logger, statusCode int, duration time.Duration, fields logrus.Fields) {
	message := fmt.Sprintf("%s %s %d %s",
		fields["method"],
		fields["path"],
		statusCode,
		duration,
	)

	// Determine log level based on status code and duration
	switch {
	case statusCode >= 500:
		logger.WithFields(fields).Error(message)
	case statusCode >= 400:
		logger.WithFields(fields).Warn(message)
	case duration > 5*time.Second:
		logger.WithFields(fields).Warn(message + " (slow request)")
	case statusCode >= 300:
		logger.WithFields(fields).Info(message)
	default:
		logger.WithFields(fields).Info(message)
	}
}

// shouldSkipPath checks if path should be skipped
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// shouldSkipUserAgent checks if user agent should be skipped
func shouldSkipUserAgent(userAgent string, skipUserAgents []string) bool {
	for _, skipUA := range skipUserAgents {
		if strings.Contains(userAgent, skipUA) {
			return true
		}
	}
	return false
}

// shouldLogHeader checks if header should be logged
func shouldLogHeader(headerName string) bool {
	// Headers to exclude for security/noise reasons
	excludeHeaders := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"X-Forwarded-For",
		"X-Real-IP",
		"Connection",
		"Cache-Control",
		"Accept-Encoding",
	}

	headerLower := strings.ToLower(headerName)
	for _, exclude := range excludeHeaders {
		if strings.ToLower(exclude) == headerLower {
			return false
		}
	}

	return true
}

// isTextContent checks if content type is text-based
func isTextContent(contentType string) bool {
	textTypes := []string{
		"application/json",
		"application/xml",
		"text/",
		"application/x-www-form-urlencoded",
	}

	contentTypeLower := strings.ToLower(contentType)
	for _, textType := range textTypes {
		if strings.Contains(contentTypeLower, textType) {
			return true
		}
	}

	return false
}

// RequestIDMiddleware adds request ID to all requests
func RequestIDMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})
}

// ResponseTimeMiddleware adds response time header
func ResponseTimeMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Add response time header
		responseTime := math.Ceil(float64(duration.Nanoseconds()) / 1000000.0) // milliseconds
		c.Header("X-Response-Time", fmt.Sprintf("%.0fms", responseTime))
	})
}

// CombinedLoggerMiddleware combines request ID, response time, and logging
func CombinedLoggerMiddleware(config LoggerConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()

		// Process request with logging
		LoggerMiddleware(config)(c)

		// Add response time
		duration := time.Since(start)
		responseTime := math.Ceil(float64(duration.Nanoseconds()) / 1000000.0)
		c.Header("X-Response-Time", fmt.Sprintf("%.0fms", responseTime))
	})
}
