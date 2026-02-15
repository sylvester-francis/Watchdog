package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

const testSessionSecret = "test-session-secret-at-least-32-bytes!"

// setupEchoWithSession creates an Echo context with session middleware wired.
func setupEchoWithSession(req *http.Request, rec *httptest.ResponseRecorder) (echo.Context, *echo.Echo) {
	e := echo.New()
	c := e.NewContext(req, rec)
	store := sessions.NewCookieStore([]byte(testSessionSecret))
	c.Set("session_store", store)
	return c, e
}

func TestAuthRequired_NoSession_Redirects(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	handlerCalled := false
	handler := middleware.AuthRequired(func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestAuthRequired_ValidSession_PassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	// Set user ID in session
	userID := uuid.New()
	err := middleware.SetUserID(c, userID)
	require.NoError(t, err)

	// Get the cookie from the response and add to new request
	cookies := rec.Result().Cookies()

	req2 := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	rec2 := httptest.NewRecorder()
	c2, _ := setupEchoWithSession(req2, rec2)

	handlerCalled := false
	handler := middleware.AuthRequired(func(c echo.Context) error {
		handlerCalled = true
		return c.String(http.StatusOK, "ok")
	})

	err = handler(c2)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestGetUserID_StringFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	expected := uuid.New()
	c.Set(middleware.UserIDKey, expected.String())

	id, ok := middleware.GetUserID(c)
	assert.True(t, ok)
	assert.Equal(t, expected, id)
}

func TestGetUserID_UUIDFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	expected := uuid.New()
	c.Set(middleware.UserIDKey, expected)

	id, ok := middleware.GetUserID(c)
	assert.True(t, ok)
	assert.Equal(t, expected, id)
}

func TestGetUserID_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	c.Set(middleware.UserIDKey, "not-a-uuid")

	_, ok := middleware.GetUserID(c)
	assert.False(t, ok)
}

func TestGetUserID_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	_, ok := middleware.GetUserID(c)
	assert.False(t, ok)
}

func TestSetUserID_SavesSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	userID := uuid.New()
	err := middleware.SetUserID(c, userID)
	require.NoError(t, err)

	// Verify session cookie was set
	cookies := rec.Result().Cookies()
	assert.NotEmpty(t, cookies)
}

func TestClearSession_ClearsValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	// Set a session value first
	userID := uuid.New()
	err := middleware.SetUserID(c, userID)
	require.NoError(t, err)

	// Clear it
	err = middleware.ClearSession(c)
	require.NoError(t, err)

	// The last session cookie written should have MaxAge -1 (expired).
	// Multiple Set-Cookie headers may exist; the last one for our session name wins.
	cookies := rec.Result().Cookies()
	var lastSessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == middleware.SessionName {
			lastSessionCookie = cookie
		}
	}
	require.NotNil(t, lastSessionCookie, "session cookie should be present")
	assert.True(t, lastSessionCookie.MaxAge < 0, "session cookie MaxAge should be negative, got %d", lastSessionCookie.MaxAge)
}

func TestIsAuthenticated_WithSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c, _ := setupEchoWithSession(req, rec)

	// No session = not authenticated
	assert.False(t, middleware.IsAuthenticated(c))

	// Set session
	userID := uuid.New()
	err := middleware.SetUserID(c, userID)
	require.NoError(t, err)

	// Now read it back with the cookie
	cookies := rec.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}
	rec2 := httptest.NewRecorder()
	c2, _ := setupEchoWithSession(req2, rec2)

	assert.True(t, middleware.IsAuthenticated(c2))
}
