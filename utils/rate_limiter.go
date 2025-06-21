package utils

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	rate       int           // requests per period
	period     time.Duration // time period
	tokens     int           // current available tokens
	maxTokens  int           // maximum tokens (burst capacity)
	lastRefill time.Time     // last time tokens were refilled
	mutex      sync.Mutex    // mutex for thread safety
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, period time.Duration) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		period:     period,
		tokens:     rate,
		maxTokens:  rate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Calculate tokens to add based on time passed
	timePassed := now.Sub(rl.lastRefill)
	tokensToAdd := int(timePassed.Nanoseconds() * int64(rl.rate) / rl.period.Nanoseconds())

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Remaining returns the number of remaining tokens
func (rl *RateLimiter) Remaining() int {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.tokens
}

// Reset resets the rate limiter
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.tokens = rl.maxTokens
	rl.lastRefill = time.Now()
}

// SetRate updates the rate limit
func (rl *RateLimiter) SetRate(rate int, period time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.rate = rate
	rl.period = period
	rl.maxTokens = rate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
}

// Simple sliding window rate limiter (alternative implementation)
type SlidingWindowRateLimiter struct {
	requests []time.Time
	limit    int
	window   time.Duration
	mutex    sync.Mutex
}

// NewSlidingWindowRateLimiter creates a sliding window rate limiter
func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request is allowed
func (swrl *SlidingWindowRateLimiter) Allow() bool {
	swrl.mutex.Lock()
	defer swrl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-swrl.window)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, req := range swrl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	swrl.requests = validRequests

	// Check if we're under the limit
	if len(swrl.requests) < swrl.limit {
		swrl.requests = append(swrl.requests, now)
		return true
	}

	return false
}
