package middleware

import (
	"crypto/rand"
	"encoding/base64"

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

			// Generate per-request nonce (16 bytes = 128 bits)
			nonceBytes := make([]byte, 16)
			if _, err := rand.Read(nonceBytes); err != nil {
				return err
			}
			nonce := base64.StdEncoding.EncodeToString(nonceBytes)
			c.Set(NonceContextKey, nonce)

			// Nonce-based CSP — the SvelteKit SPA has one inline bootstrap
			// script that gets the nonce injected by the router.
			// H-018: 'unsafe-inline' retained in style-src because SvelteKit
			// generates inline styles for transitions and components use
			// inline style= attributes (10 occurrences). Removing it would
			// break the UI. CSS injection risk is low: connect-src 'self'
			// blocks data exfiltration via background-url, and style
			// injection cannot execute JavaScript.
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'nonce-"+nonce+"' https://unpkg.com https://cdn.jsdelivr.net; "+
					"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
					"img-src 'self' data: https://validator.swagger.io; "+
					"font-src 'self' https://fonts.gstatic.com; "+
					"connect-src 'self' https://unpkg.com https://cdn.jsdelivr.net")

			// Permissions Policy
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Prevent cross-domain policy loading
			h.Set("X-Permitted-Cross-Domain-Policies", "none")

			// HSTS when running behind HTTPS (H-012: includeSubDomains + preload)
			if enableHSTS {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			return next(c)
		}
	}
}
