package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/handlers"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/internal/testutil/mocks"
)

// noOpRenderer is a Renderer that writes the template name and data keys for test assertions.
type noOpRenderer struct{}

func (n *noOpRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	_, _ = w.Write([]byte(name))
	return nil
}

func newAuthTestContext(method, path string, form url.Values) (echo.Context, *httptest.ResponseRecorder) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()

	e := echo.New()
	e.Renderer = &noOpRenderer{}
	c := e.NewContext(req, rec)

	// Wire session store
	store := sessions.NewCookieStore([]byte("test-session-secret-at-least-32-bytes!"))
	c.Set("session_store", store)

	return c, rec
}

func TestLogin_EmptyFields_Returns400(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/login", url.Values{
		"email":    {""},
		"password": {""},
	})

	err := h.Login(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_InvalidCredentials_Returns401(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{
		LoginFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return nil, services.ErrInvalidCredentials
		},
	}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/login", url.Values{
		"email":    {"test@example.com"},
		"password": {"wrongpass"},
	})

	err := h.Login(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_Success_Redirects(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{
		LoginFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return domain.NewUser("test@example.com", "hash"), nil
		},
	}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/login", url.Values{
		"email":    {"test@example.com"},
		"password": {"password123"},
	})

	err := h.Login(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/dashboard", rec.Header().Get("Location"))
}

func TestRegister_ShortPassword_Returns400(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/register", url.Values{
		"email":            {"test@example.com"},
		"password":         {"short"},
		"confirm_password": {"short"},
	})

	err := h.Register(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_PasswordExactly8_Succeeds(t *testing.T) {
	// Boundary test: exactly 8 chars should be accepted (kills < vs <= mutant)
	authSvc := &mocks.MockUserAuthService{
		RegisterFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return domain.NewUser("test@example.com", "hash"), nil
		},
	}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/register", url.Values{
		"email":            {"test@example.com"},
		"password":         {"12345678"}, // exactly 8 chars
		"confirm_password": {"12345678"},
	})

	err := h.Register(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code) // redirect = success
}

func TestRegister_PasswordMismatch_Returns400(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/register", url.Values{
		"email":            {"test@example.com"},
		"password":         {"password123"},
		"confirm_password": {"different456"},
	})

	err := h.Register(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_EmailExists_Returns400(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{
		RegisterFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return nil, services.ErrEmailAlreadyExists
		},
	}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/register", url.Values{
		"email":            {"existing@example.com"},
		"password":         {"password123"},
		"confirm_password": {"password123"},
	})

	err := h.Register(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_Success_Redirects(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{
		RegisterFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return domain.NewUser("new@example.com", "hash"), nil
		},
	}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/register", url.Values{
		"email":            {"new@example.com"},
		"password":         {"password123"},
		"confirm_password": {"password123"},
	})

	err := h.Register(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/login")
}

func TestLogin_ServiceError_Returns401(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{
		LoginFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/login", url.Values{
		"email":    {"test@example.com"},
		"password": {"password123"},
	})

	err := h.Login(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogout_Redirects(t *testing.T) {
	authSvc := &mocks.MockUserAuthService{}
	h := handlers.NewAuthHandler(authSvc, &mocks.MockUserRepository{}, nil, middleware.NewLoginLimiter(), nil)

	c, rec := newAuthTestContext(http.MethodPost, "/logout", nil)

	err := h.Logout(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

// --- Unused import guard for middleware ---
var _ = middleware.UserIDKey
