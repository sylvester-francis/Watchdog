package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// Check result statuses.
const (
	StatusUp      = "up"
	StatusDown    = "down"
	StatusTimeout = "timeout"
	StatusError   = "error"
)

// Task represents a monitoring task running on the agent.
type Task struct {
	payload TaskPayload
	conn    *Connection
	logger  *slog.Logger
	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex
}

// NewTask creates a new monitoring task.
func NewTask(payload TaskPayload, conn *Connection, logger *slog.Logger) *Task {
	return &Task{
		payload: payload,
		conn:    conn,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

// Run starts the monitoring loop.
func (t *Task) Run() {
	ticker := time.NewTicker(time.Duration(t.payload.Interval) * time.Second)
	defer ticker.Stop()

	// Run initial check immediately
	t.runCheck()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.runCheck()
		}
	}
}

// Stop stops the monitoring task.
func (t *Task) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.stopped {
		t.stopped = true
		close(t.stopCh)
	}
}

// IsStopped returns true if the task has been stopped.
func (t *Task) IsStopped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stopped
}

func (t *Task) runCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.payload.Timeout)*time.Second)
	defer cancel()

	var status string
	var latencyMs int
	var errMsg string

	start := time.Now()

	switch t.payload.Type {
	case "http":
		status, errMsg = t.checkHTTP(ctx)
	case "tcp":
		status, errMsg = t.checkTCP(ctx)
	case "ping":
		status, errMsg = t.checkPing(ctx)
	default:
		status = StatusError
		errMsg = fmt.Sprintf("unknown check type: %s", t.payload.Type)
	}

	latencyMs = int(time.Since(start).Milliseconds())

	// Send heartbeat
	if err := t.conn.SendHeartbeat(t.payload.MonitorID, status, latencyMs, errMsg); err != nil {
		t.logger.Error("failed to send heartbeat",
			slog.String("monitor_id", t.payload.MonitorID),
			slog.String("error", err.Error()),
		)
	}

	t.logger.Debug("check completed",
		slog.String("monitor_id", t.payload.MonitorID),
		slog.String("status", status),
		slog.Int("latency_ms", latencyMs),
	)
}

// checkHTTP performs an HTTP check.
func (t *Task) checkHTTP(ctx context.Context) (status, errMsg string) {
	client := &http.Client{
		Timeout: time.Duration(t.payload.Timeout) * time.Second,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.payload.Target, http.NoBody)
	if err != nil {
		return StatusError, fmt.Sprintf("invalid URL: %s", err.Error())
	}

	req.Header.Set("User-Agent", "WatchDog-Agent/1.0")

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return StatusTimeout, "request timed out"
		}
		return StatusDown, err.Error()
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx as success
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return StatusUp, ""
	}

	return StatusDown, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// checkTCP performs a TCP connection check.
func (t *Task) checkTCP(ctx context.Context) (status, errMsg string) {
	dialer := net.Dialer{
		Timeout: time.Duration(t.payload.Timeout) * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", t.payload.Target)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return StatusTimeout, "connection timed out"
		}
		return StatusDown, err.Error()
	}
	conn.Close()

	return StatusUp, ""
}

// checkPing performs a ping check.
// Note: ICMP requires elevated privileges on most systems.
// Falls back to TCP connect on port 80 or 443 for HTTP targets.
func (t *Task) checkPing(ctx context.Context) (status, errMsg string) {
	// For simplicity, use TCP connect as ping alternative
	// Real ICMP ping requires raw sockets and elevated privileges
	target := t.payload.Target

	// Try common ports if just a host is provided
	ports := []string{":80", ":443"}

	for _, port := range ports {
		addr := target + port
		if _, _, err := net.SplitHostPort(target); err == nil {
			addr = target // Already has port
		}

		dialer := net.Dialer{
			Timeout: time.Duration(t.payload.Timeout) * time.Second,
		}

		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			conn.Close()
			return StatusUp, ""
		}
	}

	return StatusDown, "host unreachable"
}

// Checker provides check functions for different monitor types.
type Checker interface {
	Check(ctx context.Context, target string, timeout time.Duration) (status string, latencyMs int, errMsg string)
}

// HTTPChecker performs HTTP checks.
type HTTPChecker struct{}

// Check performs an HTTP GET request.
func (c *HTTPChecker) Check(ctx context.Context, target string, timeout time.Duration) (status string, latencyMs int, errMsg string) {
	start := time.Now()

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return StatusError, 0, err.Error()
	}

	resp, err := client.Do(req)
	latencyMs = int(time.Since(start).Milliseconds())

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return StatusTimeout, latencyMs, "timeout"
		}
		return StatusDown, latencyMs, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return StatusUp, latencyMs, ""
	}

	return StatusDown, latencyMs, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// TCPChecker performs TCP connection checks.
type TCPChecker struct{}

// Check performs a TCP connection attempt.
func (c *TCPChecker) Check(ctx context.Context, target string, timeout time.Duration) (status string, latencyMs int, errMsg string) {
	start := time.Now()

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	latencyMs = int(time.Since(start).Milliseconds())

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return StatusTimeout, latencyMs, "timeout"
		}
		return StatusDown, latencyMs, err.Error()
	}
	defer conn.Close()

	return StatusUp, latencyMs, ""
}
