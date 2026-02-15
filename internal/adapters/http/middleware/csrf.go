package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// CSRFMiddleware returns an Echo CSRF middleware configured for HTML form protection.
// It uses a double-submit cookie pattern: a random token is set in a cookie and must
// be included as a hidden form field (_csrf) or header (X-CSRF-Token) on POST requests.
// API endpoints, WebSocket, static files, and health checks are skipped.
func CSRFMiddleware() echo.MiddlewareFunc {
	return echomiddleware.CSRFWithConfig(echomiddleware.CSRFConfig{
		TokenLookup:    "form:_csrf,header:X-CSRF-Token",
		CookieName:     "_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			// Skip CSRF for non-browser endpoints
			if strings.HasPrefix(path, "/ws/") ||
				strings.HasPrefix(path, "/api/") ||
				strings.HasPrefix(path, "/static/") ||
				strings.HasPrefix(path, "/sse/") ||
				path == "/health" {
				return true
			}
			return false
		},
	})
}

// GetCSRFToken extracts the CSRF token from the Echo context.
// Returns empty string if no token is available.
func GetCSRFToken(c echo.Context) string {
	token := c.Get("csrf")
	if token == nil {
		return ""
	}
	if s, ok := token.(string); ok {
		return s
	}
	return ""
}
