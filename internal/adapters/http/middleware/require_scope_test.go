package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

func setScope(c echo.Context, scope string) {
	c.Set("token_scope", scope)
}

func TestRequireScope_AllowsMatchingScope(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			setScope(c, string(domain.TokenScopeTelemetryIngest))
			return next(c)
		}
	})
	e.GET("/x", func(c echo.Context) error { return c.String(http.StatusOK, "ok") },
		middleware.RequireScope(domain.TokenScopeTelemetryIngest))

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireScope_RejectsWrongScope(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			setScope(c, string(domain.TokenScopeReadOnly))
			return next(c)
		}
	})
	e.GET("/x", func(c echo.Context) error { return c.String(http.StatusOK, "ok") },
		middleware.RequireScope(domain.TokenScopeTelemetryIngest))

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireScope_RejectsMissingScope(t *testing.T) {
	e := echo.New()
	e.GET("/x", func(c echo.Context) error { return c.String(http.StatusOK, "ok") },
		middleware.RequireScope(domain.TokenScopeTelemetryIngest))

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusForbidden, rec.Code,
		"missing token_scope should be a hard 403, not allowed by default")
}
