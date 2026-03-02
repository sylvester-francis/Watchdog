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

// pagerdutyEventPayload mirrors the PagerDuty event structure for test assertions.
type pagerdutyEventPayload struct {
	RoutingKey  string `json:"routing_key"`
	EventAction string `json:"event_action"`
	DedupKey    string `json:"dedup_key"`
	Payload     struct {
		Summary       string            `json:"summary"`
		Source        string            `json:"source"`
		Severity      string            `json:"severity"`
		Timestamp     string            `json:"timestamp"`
		CustomDetails map[string]string `json:"custom_details"`
	} `json:"payload"`
}

func TestPagerDutyNotifier_TriggerEvent(t *testing.T) {
	var received pagerdutyEventPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&received)
		require.NoError(t, err)

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"success","dedup_key":"test"}`))
	}))
	defer server.Close()

	notifier := notify.NewPagerDutyNotifier("test-routing-key")
	notifier.SetEventsURL(server.URL)

	incident := domain.NewIncident(uuid.New())
	monitor := domain.NewMonitor(uuid.New(), "API Server", domain.MonitorTypeHTTP, "https://api.example.com/health")

	err := notifier.NotifyIncidentOpened(context.Background(), incident, monitor)

	require.NoError(t, err)

	// Verify event_action is "trigger" for opened incidents.
	assert.Equal(t, "trigger", received.EventAction)
	assert.Equal(t, "test-routing-key", received.RoutingKey)
	assert.Equal(t, incident.ID.String(), received.DedupKey)

	// Verify payload contents.
	assert.Equal(t, notify.BrandName, received.Payload.Source)
	assert.Equal(t, "critical", received.Payload.Severity)
	assert.Contains(t, received.Payload.Summary, "API Server")
	assert.Contains(t, received.Payload.Summary, "DOWN")
	assert.Equal(t, incident.StartedAt.Format(time.RFC3339), received.Payload.Timestamp)

	// Verify custom details.
	assert.Equal(t, "API Server", received.Payload.CustomDetails["monitor_name"])
	assert.Equal(t, "http", received.Payload.CustomDetails["monitor_type"])
	assert.Equal(t, "https://api.example.com/health", received.Payload.CustomDetails["target"])
}

func TestPagerDutyNotifier_ResolveEvent(t *testing.T) {
	var received pagerdutyEventPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&received)
		require.NoError(t, err)

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"success","dedup_key":"test"}`))
	}))
	defer server.Close()

	notifier := notify.NewPagerDutyNotifier("test-routing-key")
	notifier.SetEventsURL(server.URL)

	incident := domain.NewIncident(uuid.New())
	now := time.Now()
	incident.ResolvedAt = &now
	incident.Status = domain.IncidentStatusResolved

	monitor := domain.NewMonitor(uuid.New(), "API Server", domain.MonitorTypeHTTP, "https://api.example.com/health")

	err := notifier.NotifyIncidentResolved(context.Background(), incident, monitor)

	require.NoError(t, err)

	// Verify event_action is "resolve" for resolved incidents.
	assert.Equal(t, "resolve", received.EventAction)
	assert.Equal(t, "test-routing-key", received.RoutingKey)
	assert.Equal(t, incident.ID.String(), received.DedupKey)

	// Verify payload contents.
	assert.Equal(t, notify.BrandName, received.Payload.Source)
	assert.Equal(t, "info", received.Payload.Severity)
	assert.Contains(t, received.Payload.Summary, "API Server")
	assert.Contains(t, received.Payload.Summary, "UP")

	// Verify custom details include monitor_name and duration.
	assert.Equal(t, "API Server", received.Payload.CustomDetails["monitor_name"])
	assert.NotEmpty(t, received.Payload.CustomDetails["duration"])
}

func TestPagerDutyNotifier_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"invalid event","message":"routing key is invalid"}`))
	}))
	defer server.Close()

	notifier := notify.NewPagerDutyNotifier("invalid-key")
	notifier.SetEventsURL(server.URL)

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	require.Error(t, err)
	assert.True(t, notify.IsNotifierError(err), "expected NotifierError, got: %T", err)
	assert.Contains(t, err.Error(), "pagerduty")
	assert.Contains(t, err.Error(), "400")
}
