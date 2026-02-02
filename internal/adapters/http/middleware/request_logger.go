package middleware

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID.
	RequestIDKey = "request_id"
)

// RequestID adds a unique request ID to each request.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			c.Set(RequestIDKey, requestID)
			c.Response().Header().Set(RequestIDHeader, requestID)

			return next(c)
		}
	}
}

// RequestLogger logs HTTP requests with structured logging.
func RequestLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			status := c.Response().Status

			requestID := GetRequestID(c)

			logAttrs := []any{
				slog.String("request_id", requestID),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", duration),
				slog.String("ip", c.RealIP()),
				slog.String("user_agent", c.Request().UserAgent()),
			}

			if err != nil {
				logAttrs = append(logAttrs, slog.String("error", err.Error()))
			}

			switch {
			case status >= 500:
				logger.Error("request completed", logAttrs...)
			case status >= 400:
				logger.Warn("request completed", logAttrs...)
			default:
				logger.Info("request completed", logAttrs...)
			}

			return err
		}
	}
}

// GetRequestID returns the request ID from the context.
func GetRequestID(c echo.Context) string {
	if id, ok := c.Get(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
