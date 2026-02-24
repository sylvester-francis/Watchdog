package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	maxRegistrationsPerIP = 3
	registerWindow        = 1 * time.Hour
	registerLockout       = 1 * time.Hour
	registerCleanupTick   = 10 * time.Minute
)

type registerAttempt struct {
	count    int
	firstAt  time.Time
	lockedAt time.Time
}

// RegisterLimiter tracks registration attempts per IP,
// locking out after maxRegistrationsPerIP within registerWindow.
type RegisterLimiter struct {
	mu     sync.Mutex
	byIP   map[string]*registerAttempt
	stopCh chan struct{}
}

// NewRegisterLimiter creates a RegisterLimiter and starts its cleanup goroutine.
func NewRegisterLimiter() *RegisterLimiter {
	rl := &RegisterLimiter{
		byIP:   make(map[string]*registerAttempt),
		stopCh: make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Stop shuts down the cleanup goroutine.
func (rl *RegisterLimiter) Stop() {
	close(rl.stopCh)
}

// Record records a registration attempt from the given IP.
func (rl *RegisterLimiter) Record(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.byIP[ip]
	if !ok {
		rl.byIP[ip] = &registerAttempt{count: 1, firstAt: now}
		return
	}

	// Reset window if it expired and not locked
	if now.Sub(a.firstAt) > registerWindow && a.lockedAt.IsZero() {
		a.count = 1
		a.firstAt = now
		a.lockedAt = time.Time{}
		return
	}

	a.count++
	if a.count >= maxRegistrationsPerIP {
		a.lockedAt = now
	}
}

// IsBlocked returns true if the IP is currently locked out from registering.
func (rl *RegisterLimiter) IsBlocked(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.byIP[ip]
	if !ok {
		return false
	}
	if a.lockedAt.IsZero() {
		return false
	}
	if now.Sub(a.lockedAt) > registerLockout {
		delete(rl.byIP, ip)
		return false
	}
	return true
}

// RetryAfter returns the remaining lockout duration for the IP.
func (rl *RegisterLimiter) RetryAfter(ip string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, ok := rl.byIP[ip]
	if !ok || a.lockedAt.IsZero() {
		return 0
	}
	rem := registerLockout - time.Since(a.lockedAt)
	if rem < 0 {
		return 0
	}
	return rem
}

// Middleware returns an Echo middleware that blocks registration from IPs
// that have exceeded the registration rate limit.
func (rl *RegisterLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			if rl.IsBlocked(ip) {
				retry := rl.RetryAfter(ip)
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())))
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too many registration attempts. Try again later.",
				})
			}

			return next(c)
		}
	}
}

func (rl *RegisterLimiter) cleanup() {
	ticker := time.NewTicker(registerCleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, a := range rl.byIP {
				if !a.lockedAt.IsZero() && now.Sub(a.lockedAt) > registerLockout {
					delete(rl.byIP, ip)
				} else if a.lockedAt.IsZero() && now.Sub(a.firstAt) > registerWindow {
					delete(rl.byIP, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}
