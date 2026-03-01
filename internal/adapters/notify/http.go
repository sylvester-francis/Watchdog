package notify

import (
	"context"
	"net"
	"net/http"
	"time"
)

// NewHTTPClient creates an HTTP client that forces IPv4 connections.
// This works around broken IPv6 routing on some servers where outbound
// IPv6 connections time out silently.
func NewHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", addr)
			},
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
}
