package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	// Requests per second allowed
	Rate float64
	// Maximum burst size
	Burst int
	// Cleanup interval for expired entries
	CleanupInterval time.Duration
}

// DefaultRateLimiterConfig returns sensible defaults for the general API rate
// limiter (H-019).
//
// Rationale:
//   - Rate=10 req/s: accommodates normal dashboard usage (page load + XHR
//     polling) without allowing sustained abuse. A user loading the dashboard
//     triggers ~5-8 requests; 10/s gives headroom for rapid navigation.
//   - Burst=20: allows short spikes (e.g. initial page load fetching multiple
//     resources simultaneously) without triggering 429.
//   - CleanupInterval=5m: keeps the visitor map lean by evicting IPs that
//     haven't made a request in 5 minutes.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Rate:            10,
		Burst:           20,
		CleanupInterval: 5 * time.Minute,
	}
}

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	config   RateLimiterConfig
	visitors map[string]*visitor
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewRateLimiter creates a new rate limiter with the given config.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		config:   config,
		visitors: make(map[string]*visitor),
		stopCh:   make(chan struct{}),
	}

	go rl.cleanup()

	return rl
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// Middleware returns an Echo middleware function.
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Skip rate limiting for static assets — a single SvelteKit
			// page load can trigger 20+ chunk requests.
			if strings.HasPrefix(path, "/_app/") || strings.HasPrefix(path, "/static/") {
				return next(c)
			}

			// H-019: Skip rate limiting for long-lived connections (SSE and
			// WebSocket upgrades). These endpoints hold a single connection
			// open for extended periods; counting each keep-alive or message
			// against the token bucket would starve them or force premature
			// disconnects. They have their own guards (auth, per-IP conn
			// limits for WS).
			if strings.HasPrefix(path, "/sse/") || strings.HasPrefix(path, "/ws/") {
				return next(c)
			}

			ip := c.RealIP()

			if !rl.allow(ip) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "rate limit exceeded",
				})
			}

			return next(c)
		}
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{
			tokens:   float64(rl.config.Burst) - 1,
			lastSeen: now,
		}
		return true
	}

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * rl.config.Rate
	if v.tokens > float64(rl.config.Burst) {
		v.tokens = float64(rl.config.Burst)
	}
	v.lastSeen = now

	if v.tokens < 1 {
		return false
	}

	v.tokens--
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			threshold := time.Now().Add(-rl.config.CleanupInterval)
			for key, v := range rl.visitors {
				if v.lastSeen.Before(threshold) {
					delete(rl.visitors, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}
