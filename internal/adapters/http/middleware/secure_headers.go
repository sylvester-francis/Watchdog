package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/labstack/echo/v4"
)

// NonceContextKey is the Echo context key for the per-request CSP nonce.
const NonceContextKey = "csp_nonce"

// SecureHeaders adds security-related HTTP headers to responses.
// When secureCookies is true, HSTS is also enabled (production with HTTPS).
func SecureHeaders(secureCookies ...bool) echo.MiddlewareFunc {
	enableHSTS := len(secureCookies) > 0 && secureCookies[0]

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()

			// Prevent XSS attacks
			h.Set("X-XSS-Protection", "1; mode=block")

			// Prevent MIME type sniffing
			h.Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			h.Set("X-Frame-Options", "DENY")

			// Control referrer information
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// SvelteKit SPA routes use file-based JS (no inline scripts),
			// so use a relaxed CSP that allows 'self' instead of nonces.
			if strings.HasPrefix(c.Request().RequestURI, "/app") {
				h.Set("Content-Security-Policy",
					"default-src 'self'; "+
						"script-src 'self'; "+
						"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
						"img-src 'self' data:; "+
						"font-src 'self' https://fonts.gstatic.com; "+
						"connect-src 'self'")
			} else {
				// Generate per-request nonce (16 bytes = 128 bits)
				nonceBytes := make([]byte, 16)
				if _, err := rand.Read(nonceBytes); err != nil {
					return err
				}
				nonce := base64.StdEncoding.EncodeToString(nonceBytes)
				c.Set(NonceContextKey, nonce)

				// Nonce-based CSP for Go template pages (Alpine.js CSP build)
				h.Set("Content-Security-Policy",
					"default-src 'self'; "+
						"script-src 'self' 'nonce-"+nonce+"' https://unpkg.com https://cdn.jsdelivr.net; "+
						"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; "+
						"img-src 'self' data: https://validator.swagger.io; "+
						"font-src 'self' https://fonts.gstatic.com; "+
						"connect-src 'self' https://unpkg.com https://cdn.jsdelivr.net")
			}

			// Permissions Policy
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Prevent cross-domain policy loading
			h.Set("X-Permitted-Cross-Domain-Policies", "none")

			// HSTS when running behind HTTPS
			if enableHSTS {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			return next(c)
		}
	}
}
