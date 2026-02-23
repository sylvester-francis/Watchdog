package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

// newJSONLoginContext builds an Echo context with the given IP and a JSON body
// containing the email field.
func newJSONLoginContext(ip, email string) (echo.Context, *httptest.ResponseRecorder) {
	payload := map[string]string{"email": email}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":1234"
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestLoginLimiter_AllowsUnderLimit(t *testing.T) {
	ll := middleware.NewLoginLimiter()
	defer ll.Stop()

	handler := ll.MiddlewareJSON()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	ip := "10.0.0.1"
	email := "user@example.com"

	// Record 4 failures (one under the limit of 5).
	for i := 0; i < 4; i++ {
		ll.RecordFailure(ip, email)
	}

	// The next request through the middleware should still be allowed.
	c, rec := newJSONLoginContext(ip, email)
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLoginLimiter_BlocksAfterMaxAttempts(t *testing.T) {
	ll := middleware.NewLoginLimiter()
	defer ll.Stop()

	handler := ll.MiddlewareJSON()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	ip := "10.0.0.2"
	email := "blocked@example.com"

	// Record 5 failures to trigger lockout.
	for i := 0; i < 5; i++ {
		ll.RecordFailure(ip, email)
	}

	// The middleware should now block the request with 429.
	c, rec := newJSONLoginContext(ip, email)
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestLoginLimiter_BlocksByIP(t *testing.T) {
	ll := middleware.NewLoginLimiter()
	defer ll.Stop()

	handler := ll.MiddlewareJSON()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	ip := "10.0.0.3"

	// Record failures using different emails each time so only the IP
	// accumulates enough failures to be blocked.
	for i := 0; i < 5; i++ {
		ll.RecordFailure(ip, "different-user-"+string(rune('a'+i))+"@example.com")
	}

	// A request from the blocked IP with a brand-new email should still be
	// blocked because IP-based tracking is independent.
	c, rec := newJSONLoginContext(ip, "innocent@example.com")
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestLoginLimiter_BlocksByEmail(t *testing.T) {
	ll := middleware.NewLoginLimiter()
	defer ll.Stop()

	handler := ll.MiddlewareJSON()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	email := "target@example.com"

	// Record failures from different IPs targeting the same email so only the
	// email accumulates enough failures to be blocked.
	for i := 0; i < 5; i++ {
		ll.RecordFailure("192.168.1."+string(rune('1'+i)), email)
	}

	// A request from a fresh IP with the blocked email should be rejected.
	c, rec := newJSONLoginContext("172.16.0.1", email)
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestLoginLimiter_RetryAfterHeader(t *testing.T) {
	ll := middleware.NewLoginLimiter()
	defer ll.Stop()

	handler := ll.MiddlewareJSON()(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	ip := "10.0.0.4"
	email := "retry@example.com"

	// Lock out the IP/email.
	for i := 0; i < 5; i++ {
		ll.RecordFailure(ip, email)
	}

	c, rec := newJSONLoginContext(ip, email)
	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	retryAfter := rec.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter, "blocked response must include Retry-After header")

	// The Retry-After value should be a positive integer (seconds).
	assert.Regexp(t, `^[1-9][0-9]*$`, retryAfter, "Retry-After should be a positive integer of seconds")
}

func TestLoginLimiter_IsBlockedReturnsFalse_WhenNotBlocked(t *testing.T) {
	ll := middleware.NewLoginLimiter()
	defer ll.Stop()

	// An IP and email that have never been seen should not be blocked.
	assert.False(t, ll.IsBlocked("203.0.113.1", "unknown@example.com"))

	// Record a few failures but stay under the limit.
	ll.RecordFailure("203.0.113.1", "unknown@example.com")
	ll.RecordFailure("203.0.113.1", "unknown@example.com")
	assert.False(t, ll.IsBlocked("203.0.113.1", "unknown@example.com"))
}
