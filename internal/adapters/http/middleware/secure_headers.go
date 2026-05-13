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

			// Prevent MIME type sniffing
			h.Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking. CSP frame-ancestors 'none' below covers
			// modern browsers; X-Frame-Options stays for legacy fallback.
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

			// Nonce-based CSP. Hardening notes:
			// - script-src: 'self' + per-request nonce + jsdelivr for Swagger UI
			//   on /docs. unpkg.com removed — it was a stale allowlist entry,
			//   no source code actually loaded from it.
			// - style-src: 'unsafe-inline' retained (H-018) because SvelteKit
			//   transition styles plus ~20 dynamic style="" attributes (width %,
			//   position %, etc.) in components require it. Style injection can't
			//   execute JS, and connect-src 'self' blocks data exfiltration via
			//   background-url.
			// - frame-ancestors 'none': modern equivalent of X-Frame-Options DENY.
			// - base-uri 'self': prevents <base> tag injection.
			// - form-action 'self': prevents <form action=...> exfiltration.
			// - object-src 'none': blocks plugin embedding.
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'nonce-"+nonce+"' https://cdn.jsdelivr.net; "+
					"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; "+
					"img-src 'self' data: https://validator.swagger.io; "+
					"font-src 'self' https://fonts.gstatic.com; "+
					"connect-src 'self' https://cdn.jsdelivr.net; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'; "+
					"object-src 'none'")

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
