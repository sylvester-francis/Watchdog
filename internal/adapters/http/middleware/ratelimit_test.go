package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            1,
		Burst:           5,
		CleanupInterval: time.Minute,
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First 5 requests should succeed (burst = 5)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i+1)
	}
}

func TestRateLimiter_DeniesOverBurst(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            1,
		Burst:           3,
		CleanupInterval: time.Minute,
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Exhaust burst
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = handler(c)
	}

	// Next request should be denied
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            100, // 100 per second = quick refill
		Burst:           1,
		CleanupInterval: time.Minute,
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request succeeds
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = handler(c)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Wait for token refill
	time.Sleep(50 * time.Millisecond)

	// Should succeed again after refill
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = handler(c)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            1,
		Burst:           1,
		CleanupInterval: time.Minute,
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First IP uses its burst
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = handler(c)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second IP should still have its own burst
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = handler(c)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_Middleware_Returns429(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            0,
		Burst:           1,
		CleanupInterval: time.Minute,
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// First request uses the burst
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = handler(c)

	// Second request should get 429
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiter_Stop(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            1,
		Burst:           1,
		CleanupInterval: 10 * time.Millisecond,
	})

	// Should not panic on Stop
	rl.Stop()
}

func TestRateLimiter_OnRejectCallback_FiresOn429(t *testing.T) {
	var rejectedIPs []string
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            1,
		Burst:           2,
		CleanupInterval: time.Minute,
		OnReject: func(ip string) {
			rejectedIPs = append(rejectedIPs, ip)
		},
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Exhaust burst (2 successful requests).
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.5:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = handler(c)
	}
	require.Empty(t, rejectedIPs, "OnReject must not fire on successful requests")

	// Next request: rejected -> callback fires with the IP.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.5:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = handler(c)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Len(t, rejectedIPs, 1, "OnReject should fire exactly once on the denial")
	assert.Equal(t, "192.0.2.5", rejectedIPs[0], "OnReject should receive the denied client IP")
}

func TestRateLimiter_NilOnReject_DoesNotPanic(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		Rate:            1,
		Burst:           1,
		CleanupInterval: time.Minute,
		// OnReject deliberately left nil — backward-compat path must not panic.
	})
	defer rl.Stop()

	e := echo.New()
	handler := rl.Middleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.6:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = handler(c) // second iteration is the rejection path; must not panic
	}
}
