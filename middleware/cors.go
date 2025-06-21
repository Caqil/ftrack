package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowAllOrigins  bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowAllOrigins: false,
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:8080",
			"https://ftrack.app",
			"https://www.ftrack.app",
			"https://api.ftrack.app",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"HEAD",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Cache-Control",
			"Connection",
			"DNT",
			"Host",
			"Pragma",
			"Referer",
			"Sec-Fetch-Dest",
			"Sec-Fetch-Mode",
			"Sec-Fetch-Site",
			"User-Agent",
			"Upgrade-Insecure-Requests",
			"X-Requested-With",
			"X-CSRF-Token",
			"X-Forwarded-For",
			"X-Forwarded-Proto",
			"X-Real-IP",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"Content-Disposition",
			"X-Request-ID",
			"X-Response-Time",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// ProductionCORSConfig returns production-safe CORS configuration
func ProductionCORSConfig() CORSConfig {
	return CORSConfig{
		AllowAllOrigins: false,
		AllowOrigins: []string{
			"https://ftrack.app",
			"https://www.ftrack.app",
			"https://app.ftrack.app",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Cache-Control",
			"X-Requested-With",
			"X-CSRF-Token",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}
}

// CORS returns a CORS middleware with the given configuration
func CORS(config CORSConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		requestMethod := c.Request.Header.Get("Access-Control-Request-Method")
		requestHeaders := c.Request.Header.Get("Access-Control-Request-Headers")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			handlePreflightRequest(c, config, origin, requestMethod, requestHeaders)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Handle actual requests
		handleActualRequest(c, config, origin)

		c.Next()
	})
}

// handlePreflightRequest handles CORS preflight requests
func handlePreflightRequest(c *gin.Context, config CORSConfig, origin, requestMethod, requestHeaders string) {
	// Check if origin is allowed
	if !isOriginAllowed(config, origin) {
		logrus.Warnf("CORS: Origin not allowed: %s", origin)
		return
	}

	// Set Access-Control-Allow-Origin
	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Origin", origin)
	} else if config.AllowAllOrigins {
		c.Header("Access-Control-Allow-Origin", "*")
	} else if origin != "" && isOriginAllowed(config, origin) {
		c.Header("Access-Control-Allow-Origin", origin)
	}

	// Set Access-Control-Allow-Credentials
	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// Set Access-Control-Allow-Methods
	if requestMethod != "" && isMethodAllowed(config, requestMethod) {
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
	}

	// Set Access-Control-Allow-Headers
	if requestHeaders != "" {
		allowedHeaders := filterAllowedHeaders(config, requestHeaders)
		if len(allowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		}
	} else if len(config.AllowHeaders) > 0 {
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
	}

	// Set Access-Control-Max-Age
	if config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", strconv.Itoa(int(config.MaxAge.Seconds())))
	}
}

// handleActualRequest handles CORS for actual requests
func handleActualRequest(c *gin.Context, config CORSConfig, origin string) {
	// Check if origin is allowed
	if !isOriginAllowed(config, origin) {
		logrus.Warnf("CORS: Origin not allowed for actual request: %s", origin)
		return
	}

	// Set Access-Control-Allow-Origin
	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Origin", origin)
	} else if config.AllowAllOrigins {
		c.Header("Access-Control-Allow-Origin", "*")
	} else if origin != "" && isOriginAllowed(config, origin) {
		c.Header("Access-Control-Allow-Origin", origin)
	}

	// Set Access-Control-Allow-Credentials
	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// Set Access-Control-Expose-Headers
	if len(config.ExposeHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
	}

	// Set Vary header to ensure proper caching
	c.Header("Vary", "Origin")
}

// isOriginAllowed checks if the origin is allowed
func isOriginAllowed(config CORSConfig, origin string) bool {
	if config.AllowAllOrigins {
		return true
	}

	if origin == "" {
		return false
	}

	for _, allowedOrigin := range config.AllowOrigins {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := allowedOrigin[2:]
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}

	return false
}

// isMethodAllowed checks if the HTTP method is allowed
func isMethodAllowed(config CORSConfig, method string) bool {
	for _, allowedMethod := range config.AllowMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

// filterAllowedHeaders filters requested headers against allowed headers
func filterAllowedHeaders(config CORSConfig, requestHeaders string) []string {
	headers := strings.Split(requestHeaders, ",")
	var allowedHeaders []string

	for _, header := range headers {
		header = strings.TrimSpace(header)
		if isHeaderAllowed(config, header) {
			allowedHeaders = append(allowedHeaders, header)
		}
	}

	return allowedHeaders
}

// isHeaderAllowed checks if a header is allowed
func isHeaderAllowed(config CORSConfig, header string) bool {
	// Always allow simple headers
	simpleHeaders := []string{
		"Accept",
		"Accept-Language",
		"Content-Language",
		"Content-Type",
	}

	header = strings.ToLower(header)
	for _, simpleHeader := range simpleHeaders {
		if strings.ToLower(simpleHeader) == header {
			return true
		}
	}

	// Check against configured allowed headers
	for _, allowedHeader := range config.AllowHeaders {
		if strings.ToLower(allowedHeader) == header {
			return true
		}
	}

	return false
}

// CORSWithConfig returns CORS middleware with custom configuration
func CORSWithConfig(config CORSConfig) gin.HandlerFunc {
	return CORS(config)
}

// DefaultCORS returns CORS middleware with default configuration
func DefaultCORS() gin.HandlerFunc {
	return CORS(DefaultCORSConfig())
}

// ProductionCORS returns CORS middleware with production configuration
func ProductionCORS() gin.HandlerFunc {
	return CORS(ProductionCORSConfig())
}

// DevelopmentCORS returns permissive CORS for development
func DevelopmentCORS() gin.HandlerFunc {
	config := CORSConfig{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	return CORS(config)
}

// CORSMiddleware is a convenience function that automatically selects
// the appropriate CORS configuration based on environment
func CORSMiddleware(environment string) gin.HandlerFunc {
	switch environment {
	case "production":
		logrus.Info("🔒 Using production CORS configuration")
		return ProductionCORS()
	case "development":
		logrus.Info("🔓 Using development CORS configuration")
		return DevelopmentCORS()
	default:
		logrus.Info("🔧 Using default CORS configuration")
		return DefaultCORS()
	}
}
