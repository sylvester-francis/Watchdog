package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSRFMiddleware_SkipsOTLPReceivers exercises the actual CSRF
// middleware against representative paths and asserts that POSTs to the
// OTLP HTTP receivers (/v1/traces, /v1/logs, /v1/logs/raw) are NOT
// rejected for missing the CSRF token. These endpoints are bearer-token
// authenticated and called by collectors that can't participate in the
// double-submit cookie pattern.
func TestCSRFMiddleware_SkipsOTLPReceivers(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"OTLP traces", "/v1/traces"},
		{"OTLP logs", "/v1/logs"},
		{"NDJSON logs", "/v1/logs/raw"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			e.Use(CSRFMiddleware(false))
			e.POST(tc.path, func(c echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			require.NotEqual(t, http.StatusBadRequest, rec.Code,
				"%s POST should not be blocked by CSRF; body=%s",
				tc.path, rec.Body.String())
			assert.Equal(t, http.StatusNoContent, rec.Code,
				"handler should be reached for %s", tc.path)
		})
	}
}

// TestCSRFMiddleware_StillEnforcesOnFormPosts confirms the OTLP exemption
// didn't accidentally relax CSRF on regular browser-facing form POSTs.
func TestCSRFMiddleware_StillEnforcesOnFormPosts(t *testing.T) {
	e := echo.New()
	e.Use(CSRFMiddleware(false))
	e.POST("/login", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"non-OTLP form POSTs without _csrf token must still be rejected")
}
