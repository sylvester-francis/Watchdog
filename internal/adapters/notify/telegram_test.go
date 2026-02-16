package notify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
)

func TestTelegramNotifier_IncidentOpened_Success(t *testing.T) {
	var receivedPayload map[string]string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	notifier := notify.NewTelegramNotifier("test-bot-token", "12345")
	notifier.SetBaseURL(server.URL)

	incident := domain.NewIncident(uuid.New())
	monitor := domain.NewMonitor(uuid.New(), "Test HTTP Monitor", domain.MonitorTypeHTTP, "https://example.com")

	err := notifier.NotifyIncidentOpened(context.Background(), incident, monitor)

	require.NoError(t, err)

	// Verify the bot token is used in the URL path.
	assert.Equal(t, "/bottest-bot-token/sendMessage", receivedPath)

	// Verify the payload contains the correct chat_id and parse_mode.
	assert.Equal(t, "12345", receivedPayload["chat_id"])
	assert.Equal(t, "Markdown", receivedPayload["parse_mode"])

	// Verify the text mentions the monitor name and contains "Incident Opened".
	assert.Contains(t, receivedPayload["text"], "Incident Opened")
	assert.Contains(t, receivedPayload["text"], "Test HTTP Monitor")
	assert.Contains(t, receivedPayload["text"], "https://example.com")
}

func TestTelegramNotifier_IncidentResolved_Success(t *testing.T) {
	var receivedPayload map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	notifier := notify.NewTelegramNotifier("test-bot-token", "12345")
	notifier.SetBaseURL(server.URL)

	incident := domain.NewIncident(uuid.New())
	now := time.Now()
	incident.ResolvedAt = &now
	incident.Status = domain.IncidentStatusResolved

	monitor := domain.NewMonitor(uuid.New(), "Test DNS Monitor", domain.MonitorTypeDNS, "example.com")

	err := notifier.NotifyIncidentResolved(context.Background(), incident, monitor)

	require.NoError(t, err)

	// Verify the payload contains the correct chat_id.
	assert.Equal(t, "12345", receivedPayload["chat_id"])
	assert.Equal(t, "Markdown", receivedPayload["parse_mode"])

	// Verify the text mentions resolution and includes the monitor name.
	assert.Contains(t, receivedPayload["text"], "Incident Resolved")
	assert.Contains(t, receivedPayload["text"], "Test DNS Monitor")
	assert.Contains(t, receivedPayload["text"], "example.com")
}

func TestTelegramNotifier_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request"}`))
	}))
	defer server.Close()

	notifier := notify.NewTelegramNotifier("bad-token", "12345")
	notifier.SetBaseURL(server.URL)

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	require.Error(t, err)
	assert.True(t, notify.IsNotifierError(err), "expected NotifierError, got: %T", err)
	assert.Contains(t, err.Error(), "telegram")
	assert.Contains(t, err.Error(), "400")
}
