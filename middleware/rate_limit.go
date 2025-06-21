package middleware

import (
	"context"
	"fmt"
	"ftrack/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Redis          *redis.Client
	Requests       int           // Number of requests allowed
	Window         time.Duration // Time window
	KeyPrefix      string        // Redis key prefix
	SkipPaths      []string      // Paths to skip rate limiting
	SkipUserAgents []string      // User agents to skip
	ErrorMessage   string        // Custom error message
}

// RateLimitStrategy defines different rate limiting strategies
type RateLimitStrategy string

const (
	StrategyIP       RateLimitStrategy = "ip"
	StrategyUser     RateLimitStrategy = "user"
	StrategyUserOrIP RateLimitStrategy = "user_or_ip"
	StrategyGlobal   RateLimitStrategy = "global"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	config   RateLimitConfig
	strategy RateLimitStrategy
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig, strategy RateLimitStrategy) *RateLimiter {
	if config.KeyPrefix == "" {
		config.KeyPrefix = "rate_limit"
	}
	if config.ErrorMessage == "" {
		config.ErrorMessage = "Rate limit exceeded"
	}

	return &RateLimiter{
		config:   config,
		strategy: strategy,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Skip rate limiting for certain paths
		if rl.shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Skip rate limiting for certain user agents
		if rl.shouldSkipUserAgent(c.GetHeader("User-Agent")) {
			c.Next()
			return
		}

		// Get rate limit key based on strategy
		key := rl.getKey(c)
		if key == "" {
			c.Next()
			return
		}

		// Check rate limit
		allowed, resetTime, remaining, err := rl.checkRateLimit(key)
		if err != nil {
			logrus.Errorf("Rate limit check failed: %v", err)
			// Allow request to proceed on error
			c.Next()
			return
		}

		// Set rate limit headers
		rl.setRateLimitHeaders(c, remaining, resetTime)

		if !allowed {
			rl.handleRateLimitExceeded(c, resetTime)
			return
		}

		c.Next()
	})
}

// checkRateLimit checks if request is within rate limit
func (rl *RateLimiter) checkRateLimit(key string) (allowed bool, resetTime time.Time, remaining int, err error) {
	ctx := context.Background()
	now := time.Now()
	window := rl.config.Window

	// Use sliding window log algorithm with Redis sorted sets
	pipe := rl.config.Redis.Pipeline()

	// Remove expired entries
	expiredBefore := now.Add(-window).UnixNano()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", expiredBefore))

	// Count current requests
	pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiration for the key
	pipe.Expire(ctx, key, window+time.Minute)

	// Execute pipeline
	results, err := pipe.Exec(ctx)
	if err != nil {
		return false, time.Time{}, 0, err
	}

	// Get current count (before adding the new request)
	currentCount := results[1].(*redis.IntCmd).Val()

	// Calculate remaining requests
	remaining = rl.config.Requests - int(currentCount) - 1
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (next window)
	resetTime = now.Add(window)

	// Check if within limit
	allowed = currentCount < int64(rl.config.Requests)

	// If not allowed, remove the request we just added
	if !allowed {
		rl.config.Redis.ZRem(ctx, key, fmt.Sprintf("%d", now.UnixNano()))
	}

	return allowed, resetTime, remaining, nil
}

// getKey generates rate limit key based on strategy
func (rl *RateLimiter) getKey(c *gin.Context) string {
	prefix := rl.config.KeyPrefix

	switch rl.strategy {
	case StrategyIP:
		return fmt.Sprintf("%s:ip:%s", prefix, rl.getClientIP(c))

	case StrategyUser:
		userID := c.GetString("userID")
		if userID == "" {
			return ""
		}
		return fmt.Sprintf("%s:user:%s", prefix, userID)

	case StrategyUserOrIP:
		userID := c.GetString("userID")
		if userID != "" {
			return fmt.Sprintf("%s:user:%s", prefix, userID)
		}
		return fmt.Sprintf("%s:ip:%s", prefix, rl.getClientIP(c))

	case StrategyGlobal:
		return fmt.Sprintf("%s:global", prefix)

	default:
		return fmt.Sprintf("%s:ip:%s", prefix, rl.getClientIP(c))
	}
}

// getClientIP gets the real client IP
func (rl *RateLimiter) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Use Gin's ClientIP as fallback
	return c.ClientIP()
}

// setRateLimitHeaders sets rate limit related headers
func (rl *RateLimiter) setRateLimitHeaders(c *gin.Context, remaining int, resetTime time.Time) {
	c.Header("X-RateLimit-Limit", strconv.Itoa(rl.config.Requests))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	c.Header("X-RateLimit-Window", rl.config.Window.String())
}

// handleRateLimitExceeded handles rate limit exceeded scenarios
func (rl *RateLimiter) handleRateLimitExceeded(c *gin.Context, resetTime time.Time) {
	retryAfter := time.Until(resetTime).Seconds()
	if retryAfter < 0 {
		retryAfter = 0
	}

	c.Header("Retry-After", strconv.Itoa(int(retryAfter)))

	response := models.ErrorResponse{
		Error:     "RATE_LIMIT_EXCEEDED",
		Message:   rl.config.ErrorMessage,
		Code:      "TOO_MANY_REQUESTS",
		RequestID: c.GetString("request_id"),
		Details: map[string]interface{}{
			"retry_after": int(retryAfter),
			"reset_time":  resetTime.Unix(),
		},
	}

	logrus.WithFields(logrus.Fields{
		"client_ip":   rl.getClientIP(c),
		"user_id":     c.GetString("userID"),
		"path":        c.Request.URL.Path,
		"method":      c.Request.Method,
		"retry_after": retryAfter,
	}).Warn("Rate limit exceeded")

	c.JSON(http.StatusTooManyRequests, response)
	c.Abort()
}

// shouldSkipPath checks if path should be skipped
func (rl *RateLimiter) shouldSkipPath(path string) bool {
	for _, skipPath := range rl.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// shouldSkipUserAgent checks if user agent should be skipped
func (rl *RateLimiter) shouldSkipUserAgent(userAgent string) bool {
	for _, skipUA := range rl.config.SkipUserAgents {
		if strings.Contains(userAgent, skipUA) {
			return true
		}
	}
	return false
}

// Predefined rate limiters

// DefaultRateLimit creates a default rate limiter (100 requests per minute per IP)
func DefaultRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     100,
		Window:       time.Minute,
		KeyPrefix:    "rate_limit",
		ErrorMessage: "Too many requests. Please try again later.",
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
		SkipUserAgents: []string{
			"kube-probe",
			"GoogleHC",
		},
	}

	limiter := NewRateLimiter(config, StrategyIP)
	return limiter.Middleware()
}

// AuthRateLimit creates rate limiter for authentication endpoints
func AuthRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     5,
		Window:       time.Minute,
		KeyPrefix:    "auth_rate_limit",
		ErrorMessage: "Too many authentication attempts. Please try again later.",
	}

	limiter := NewRateLimiter(config, StrategyIP)
	return limiter.Middleware()
}

// APIRateLimit creates rate limiter for API endpoints based on user
func APIRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     1000,
		Window:       time.Hour,
		KeyPrefix:    "api_rate_limit",
		ErrorMessage: "API rate limit exceeded. Please try again later.",
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/docs",
		},
	}

	limiter := NewRateLimiter(config, StrategyUserOrIP)
	return limiter.Middleware()
}

// UploadRateLimit creates rate limiter for file uploads
func UploadRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     10,
		Window:       time.Minute,
		KeyPrefix:    "upload_rate_limit",
		ErrorMessage: "Upload rate limit exceeded. Please try again later.",
	}

	limiter := NewRateLimiter(config, StrategyUserOrIP)
	return limiter.Middleware()
}

// MessageRateLimit creates rate limiter for messaging
func MessageRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     60,
		Window:       time.Minute,
		KeyPrefix:    "message_rate_limit",
		ErrorMessage: "Message rate limit exceeded. Please slow down.",
	}

	limiter := NewRateLimiter(config, StrategyUser)
	return limiter.Middleware()
}

// EmergencyRateLimit creates rate limiter for emergency alerts
func EmergencyRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     3,
		Window:       time.Minute,
		KeyPrefix:    "emergency_rate_limit",
		ErrorMessage: "Emergency alert rate limit exceeded.",
	}

	limiter := NewRateLimiter(config, StrategyUser)
	return limiter.Middleware()
}

// AdminRateLimit creates lenient rate limiter for admin operations
func AdminRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     500,
		Window:       time.Minute,
		KeyPrefix:    "admin_rate_limit",
		ErrorMessage: "Admin rate limit exceeded.",
	}

	limiter := NewRateLimiter(config, StrategyUser)
	return limiter.Middleware()
}

// WebSocketRateLimit creates rate limiter for WebSocket connections
func WebSocketRateLimit(redis *redis.Client) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     10,
		Window:       time.Minute,
		KeyPrefix:    "ws_rate_limit",
		ErrorMessage: "WebSocket connection rate limit exceeded.",
	}

	limiter := NewRateLimiter(config, StrategyIP)
	return limiter.Middleware()
}

// CustomRateLimit creates a custom rate limiter
func CustomRateLimit(redis *redis.Client, requests int, window time.Duration, strategy RateLimitStrategy, keyPrefix string) gin.HandlerFunc {
	config := RateLimitConfig{
		Redis:        redis,
		Requests:     requests,
		Window:       window,
		KeyPrefix:    keyPrefix,
		ErrorMessage: "Rate limit exceeded. Please try again later.",
	}

	limiter := NewRateLimiter(config, strategy)
	return limiter.Middleware()
}

// RateLimitMiddleware creates rate limiter based on environment
func RateLimitMiddleware(redis *redis.Client, environment string) gin.HandlerFunc {
	switch environment {
	case "production":
		return APIRateLimit(redis)
	case "development":
		// More lenient for development
		return CustomRateLimit(redis, 10000, time.Hour, StrategyIP, "dev_rate_limit")
	default:
		return DefaultRateLimit(redis)
	}
}
