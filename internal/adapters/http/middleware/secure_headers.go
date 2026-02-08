package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecureHeaders adds security-related HTTP headers to responses.
func SecureHeaders() echo.MiddlewareFunc {
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

			// Content Security Policy
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.tailwindcss.com; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'")

			// Permissions Policy (formerly Feature-Policy)
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			return next(c)
		}
	}
}

// SecureHeadersStrict adds stricter security headers including HSTS.
// Use this for production with HTTPS enabled.
func SecureHeadersStrict() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()

			// All standard headers
			h.Set("X-XSS-Protection", "1; mode=block")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.tailwindcss.com; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// HSTS - enforce HTTPS for 1 year, include subdomains
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			return next(c)
		}
	}
}
