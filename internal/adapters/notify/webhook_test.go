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
	"github.com/sylvester-francis/watchdog/core/domain"
)

func TestWebhookNotifier_IncidentOpened_Success(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, http.MethodPost, r.Method)

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL)

	incident := domain.NewIncident(uuid.New())
	monitor := domain.NewMonitor(uuid.New(), "Test HTTP Monitor", domain.MonitorTypeHTTP, "https://example.com")

	err := notifier.NotifyIncidentOpened(context.Background(), incident, monitor)

	require.NoError(t, err)
	assert.Equal(t, "incident.opened", receivedPayload["event"])

	// Verify incident data
	incidentData := receivedPayload["incident"].(map[string]interface{})
	assert.Equal(t, incident.ID.String(), incidentData["id"])
	assert.Equal(t, incident.MonitorID.String(), incidentData["monitor_id"])

	// Verify monitor data
	monitorData := receivedPayload["monitor"].(map[string]interface{})
	assert.Equal(t, monitor.ID.String(), monitorData["id"])
	assert.Equal(t, "Test HTTP Monitor", monitorData["name"])
	assert.Equal(t, "http", monitorData["type"])
	assert.Equal(t, "https://example.com", monitorData["target"])
}

func TestWebhookNotifier_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL)

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.True(t, notify.IsNotifierError(err))
}

func TestWebhookNotifier_IncidentResolved_Success(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL)

	incident := domain.NewIncident(uuid.New())
	now := time.Now()
	incident.ResolvedAt = &now
	incident.Status = domain.IncidentStatusResolved

	monitor := domain.NewMonitor(uuid.New(), "Test DNS", domain.MonitorTypeDNS, "example.com")

	err := notifier.NotifyIncidentResolved(context.Background(), incident, monitor)

	require.NoError(t, err)
	assert.Equal(t, "incident.resolved", receivedPayload["event"])

	// Verify resolved_at is included
	incidentData := receivedPayload["incident"].(map[string]interface{})
	assert.NotNil(t, incidentData["resolved_at"])
}

func TestWebhookNotifier_Status300_Returns_Error(t *testing.T) {
	// Boundary test: 300 is NOT a success status (kills >= 300 vs > 300 mutant)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMultipleChoices) // 300
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL)

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.True(t, notify.IsNotifierError(err))
}

func TestWebhookNotifier_ConnectionRefused(t *testing.T) {
	notifier := notify.NewWebhookNotifier("http://127.0.0.1:1") // Nothing listening on port 1

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.True(t, notify.IsNotifierError(err))
}
