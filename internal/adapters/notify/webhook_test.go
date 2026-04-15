package notify_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
)

func TestWebhookNotifier_IncidentOpened_Success(t *testing.T) {
	var receivedPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, http.MethodPost, r.Method)

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, "")

	incident := domain.NewIncident(uuid.New())
	monitor := domain.NewMonitor(uuid.New(), "Test HTTP Monitor", domain.MonitorTypeHTTP, "https://example.com")

	err := notifier.NotifyIncidentOpened(context.Background(), incident, monitor)

	require.NoError(t, err)
	assert.Equal(t, "incident.opened", receivedPayload["event"])

	// Verify incident data
	incidentData := receivedPayload["incident"].(map[string]any)
	assert.Equal(t, incident.ID.String(), incidentData["id"])
	assert.Equal(t, incident.MonitorID.String(), incidentData["monitor_id"])

	// Verify monitor data
	monitorData := receivedPayload["monitor"].(map[string]any)
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

	notifier := notify.NewWebhookNotifier(server.URL, "")

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.True(t, notify.IsNotifierError(err))
}

func TestWebhookNotifier_IncidentResolved_Success(t *testing.T) {
	var receivedPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, "")

	incident := domain.NewIncident(uuid.New())
	now := time.Now()
	incident.ResolvedAt = &now
	incident.Status = domain.IncidentStatusResolved

	monitor := domain.NewMonitor(uuid.New(), "Test DNS", domain.MonitorTypeDNS, "example.com")

	err := notifier.NotifyIncidentResolved(context.Background(), incident, monitor)

	require.NoError(t, err)
	assert.Equal(t, "incident.resolved", receivedPayload["event"])

	// Verify resolved_at is included
	incidentData := receivedPayload["incident"].(map[string]any)
	assert.NotNil(t, incidentData["resolved_at"])
}

func TestWebhookNotifier_Status300_Returns_Error(t *testing.T) {
	// Boundary test: 300 is NOT a success status (kills >= 300 vs > 300 mutant)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMultipleChoices) // 300
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, "")

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.True(t, notify.IsNotifierError(err))
}

func TestWebhookNotifier_ConnectionRefused(t *testing.T) {
	notifier := notify.NewWebhookNotifier("http://127.0.0.1:1", "") // Nothing listening on port 1

	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())

	assert.Error(t, err)
	assert.True(t, notify.IsNotifierError(err))
}

func TestWebhookNotifier_Signed_IncidentOpened_HeadersPresent(t *testing.T) {
	secret := "test-secret-123"
	var receivedHeaders http.Header
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, secret)
	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())
	require.NoError(t, err)

	sig := receivedHeaders.Get("X-Watchdog-Signature-256")
	ts := receivedHeaders.Get("X-Watchdog-Timestamp")
	nonce := receivedHeaders.Get("X-Watchdog-Nonce")

	require.NotEmpty(t, sig, "signature header missing")
	require.NotEmpty(t, ts, "timestamp header missing")
	require.NotEmpty(t, nonce, "nonce header missing")

	assert.True(t, strings.HasPrefix(sig, "sha256="), "signature must have sha256= prefix: %s", sig)
	hexPart := strings.TrimPrefix(sig, "sha256=")
	assert.Len(t, hexPart, 64, "sha256 hex is 64 chars")

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	require.NoError(t, err)
	expected := notify.SignWebhookPayload(secret, time.Unix(tsInt, 0), nonce, receivedBody)
	assert.Equal(t, expected, hexPart, "signature must verify with same inputs")
}

func TestWebhookNotifier_Unsigned_NoHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, "")
	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())
	require.NoError(t, err)

	assert.Empty(t, receivedHeaders.Get("X-Watchdog-Signature-256"))
	assert.Empty(t, receivedHeaders.Get("X-Watchdog-Timestamp"))
	assert.Empty(t, receivedHeaders.Get("X-Watchdog-Nonce"))
}

func TestWebhookNotifier_Signed_TimestampIsRecent(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	before := time.Now().Unix()
	notifier := notify.NewWebhookNotifier(server.URL, "s")
	err := notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor())
	require.NoError(t, err)
	after := time.Now().Unix()

	ts, err := strconv.ParseInt(receivedHeaders.Get("X-Watchdog-Timestamp"), 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, ts, before)
	assert.LessOrEqual(t, ts, after)
}

func TestWebhookNotifier_Signed_AgentOffline_HeadersPresent(t *testing.T) {
	secret := "s"
	var sig, ts, nonce string
	var body []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig = r.Header.Get("X-Watchdog-Signature-256")
		ts = r.Header.Get("X-Watchdog-Timestamp")
		nonce = r.Header.Get("X-Watchdog-Nonce")
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, secret)
	agent := &domain.Agent{ID: uuid.New(), Name: "agent-1"}
	err := notifier.NotifyAgentOffline(context.Background(), agent, 3)
	require.NoError(t, err)

	require.NotEmpty(t, sig)
	require.NotEmpty(t, ts)
	require.NotEmpty(t, nonce)

	tsInt, _ := strconv.ParseInt(ts, 10, 64)
	expected := "sha256=" + notify.SignWebhookPayload(secret, time.Unix(tsInt, 0), nonce, body)
	assert.Equal(t, expected, sig)
}

func TestWebhookNotifier_Signed_EachRequestHasFreshNonce(t *testing.T) {
	var nonces []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonces = append(nonces, r.Header.Get("X-Watchdog-Nonce"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := notify.NewWebhookNotifier(server.URL, "s")
	for range 3 {
		require.NoError(t, notifier.NotifyIncidentOpened(context.Background(), testIncident(), testMonitor()))
	}

	require.Len(t, nonces, 3)
	assert.NotEqual(t, nonces[0], nonces[1])
	assert.NotEqual(t, nonces[1], nonces[2])
	assert.NotEqual(t, nonces[0], nonces[2])
}
