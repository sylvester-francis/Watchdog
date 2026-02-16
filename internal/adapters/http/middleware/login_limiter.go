package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	maxLoginAttempts = 5
	lockoutWindow    = 15 * time.Minute
	lockoutDuration  = 15 * time.Minute
	cleanupTick      = 5 * time.Minute
)

type loginAttempt struct {
	failures int
	firstAt  time.Time
	lockedAt time.Time
}

// LoginLimiter tracks failed login attempts per IP and email,
// locking out after maxLoginAttempts failures within lockoutWindow.
type LoginLimiter struct {
	mu      sync.Mutex
	byIP    map[string]*loginAttempt
	byEmail map[string]*loginAttempt
	stopCh  chan struct{}
}

// NewLoginLimiter creates a LoginLimiter and starts its cleanup goroutine.
func NewLoginLimiter() *LoginLimiter {
	ll := &LoginLimiter{
		byIP:    make(map[string]*loginAttempt),
		byEmail: make(map[string]*loginAttempt),
		stopCh:  make(chan struct{}),
	}
	go ll.cleanup()
	return ll
}

// Stop shuts down the cleanup goroutine.
func (ll *LoginLimiter) Stop() {
	close(ll.stopCh)
}

// RecordFailure records a failed login attempt for both the IP and email.
func (ll *LoginLimiter) RecordFailure(ip, email string) {
	ll.mu.Lock()
	defer ll.mu.Unlock()

	now := time.Now()
	ll.record(ll.byIP, ip, now)
	if email != "" {
		ll.record(ll.byEmail, email, now)
	}
}

// IsBlocked returns true if either the IP or email is currently locked out.
func (ll *LoginLimiter) IsBlocked(ip, email string) bool {
	ll.mu.Lock()
	defer ll.mu.Unlock()

	now := time.Now()
	if ll.blocked(ll.byIP, ip, now) {
		return true
	}
	if email != "" && ll.blocked(ll.byEmail, email, now) {
		return true
	}
	return false
}

// RetryAfter returns the remaining lockout duration for the IP or email,
// whichever is longer. Returns 0 if not locked.
func (ll *LoginLimiter) RetryAfter(ip, email string) time.Duration {
	ll.mu.Lock()
	defer ll.mu.Unlock()

	now := time.Now()
	d1 := ll.remaining(ll.byIP, ip, now)
	d2 := ll.remaining(ll.byEmail, email, now)
	if d1 > d2 {
		return d1
	}
	return d2
}

// Middleware returns an Echo middleware that rejects requests when the
// client IP or submitted email is locked out.
func (ll *LoginLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			email := c.FormValue("email")

			if ll.IsBlocked(ip, email) {
				retry := ll.RetryAfter(ip, email)
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())))
				return c.Render(http.StatusTooManyRequests, "auth.html", map[string]interface{}{
					"Title":   "Login",
					"IsLogin": true,
					"Error":   fmt.Sprintf("Too many failed attempts. Try again in %d minutes.", int(retry.Minutes())+1),
					"Email":   email,
				})
			}

			return next(c)
		}
	}
}

func (ll *LoginLimiter) record(m map[string]*loginAttempt, key string, now time.Time) {
	a, ok := m[key]
	if !ok {
		m[key] = &loginAttempt{failures: 1, firstAt: now}
		return
	}

	// Reset window if it expired
	if now.Sub(a.firstAt) > lockoutWindow && a.lockedAt.IsZero() {
		a.failures = 1
		a.firstAt = now
		a.lockedAt = time.Time{}
		return
	}

	a.failures++
	if a.failures >= maxLoginAttempts {
		a.lockedAt = now
	}
}

func (ll *LoginLimiter) blocked(m map[string]*loginAttempt, key string, now time.Time) bool {
	a, ok := m[key]
	if !ok {
		return false
	}
	if a.lockedAt.IsZero() {
		return false
	}
	if now.Sub(a.lockedAt) > lockoutDuration {
		// Lockout expired — reset
		delete(m, key)
		return false
	}
	return true
}

func (ll *LoginLimiter) remaining(m map[string]*loginAttempt, key string, now time.Time) time.Duration {
	a, ok := m[key]
	if !ok || a.lockedAt.IsZero() {
		return 0
	}
	rem := lockoutDuration - now.Sub(a.lockedAt)
	if rem < 0 {
		return 0
	}
	return rem
}

func (ll *LoginLimiter) cleanup() {
	ticker := time.NewTicker(cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ll.mu.Lock()
			now := time.Now()
			ll.purge(ll.byIP, now)
			ll.purge(ll.byEmail, now)
			ll.mu.Unlock()
		case <-ll.stopCh:
			return
		}
	}
}

func (ll *LoginLimiter) purge(m map[string]*loginAttempt, now time.Time) {
	for key, a := range m {
		if !a.lockedAt.IsZero() && now.Sub(a.lockedAt) > lockoutDuration {
			delete(m, key)
		} else if a.lockedAt.IsZero() && now.Sub(a.firstAt) > lockoutWindow {
			delete(m, key)
		}
	}
}
