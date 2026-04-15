package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// WebhookNotifier sends notifications to a generic webhook URL.
// When signingSecret is non-empty, each request includes
// X-Watchdog-Signature-256, X-Watchdog-Timestamp, and X-Watchdog-Nonce
// headers for integrity verification and replay protection.
// See docs/webhooks.md for the verification recipe.
type WebhookNotifier struct {
	url           string
	signingSecret string
	httpClient    *http.Client
}

// NewWebhookNotifier creates a new generic webhook notifier.
// Pass an empty signingSecret to send unsigned webhooks (backward compat).
func NewWebhookNotifier(url, signingSecret string) *WebhookNotifier {
	return &WebhookNotifier{
		url:           url,
		signingSecret: signingSecret,
		httpClient:    NewHTTPClient(10 * time.Second),
	}
}

// NotifyIncidentOpened sends a notification when an incident is opened.
func (w *WebhookNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := webhookPayload{
		Event:     "incident.opened",
		Timestamp: incident.StartedAt,
		Incident: webhookIncident{
			ID:        incident.ID.String(),
			MonitorID: incident.MonitorID.String(),
			Status:    string(incident.Status),
			StartedAt: incident.StartedAt,
		},
		Monitor: webhookMonitor{
			ID:     monitor.ID.String(),
			Name:   monitor.Name,
			Type:   string(monitor.Type),
			Target: monitor.Target,
		},
		Context: buildWebhookContext(incident),
	}

	return w.send(ctx, payload)
}

// NotifyIncidentResolved sends a notification when an incident is resolved.
func (w *WebhookNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	payload := webhookPayload{
		Event:     "incident.resolved",
		Timestamp: time.Now(),
		Incident: webhookIncident{
			ID:         incident.ID.String(),
			MonitorID:  incident.MonitorID.String(),
			Status:     string(incident.Status),
			StartedAt:  incident.StartedAt,
			ResolvedAt: incident.ResolvedAt,
		},
		Monitor: webhookMonitor{
			ID:     monitor.ID.String(),
			Name:   monitor.Name,
			Type:   string(monitor.Type),
			Target: monitor.Target,
		},
		Context: buildWebhookContext(incident),
	}

	return w.send(ctx, payload)
}

// NotifyAgentOffline sends a notification when an agent goes offline.
func (w *WebhookNotifier) NotifyAgentOffline(ctx context.Context, agent *domain.Agent, affectedMonitors int) error {
	payload := webhookAgentPayload{
		Event:            "agent.offline",
		Timestamp:        time.Now(),
		AgentID:          agent.ID.String(),
		AgentName:        agent.Name,
		AffectedMonitors: affectedMonitors,
	}
	return w.sendAgent(ctx, payload)
}

// NotifyAgentOnline sends a notification when an agent comes back online.
func (w *WebhookNotifier) NotifyAgentOnline(ctx context.Context, agent *domain.Agent, resolvedIncidents int) error {
	payload := webhookAgentPayload{
		Event:             "agent.online",
		Timestamp:         time.Now(),
		AgentID:           agent.ID.String(),
		AgentName:         agent.Name,
		ResolvedIncidents: resolvedIncidents,
	}
	return w.sendAgent(ctx, payload)
}

// NotifyAgentMaintenance sends a notification when an agent enters maintenance mode.
func (w *WebhookNotifier) NotifyAgentMaintenance(ctx context.Context, agent *domain.Agent, windowName string) error {
	payload := webhookAgentPayload{
		Event:      "agent.maintenance",
		Timestamp:  time.Now(),
		AgentID:    agent.ID.String(),
		AgentName:  agent.Name,
		WindowName: windowName,
	}
	return w.sendAgent(ctx, payload)
}

func (w *WebhookNotifier) sendAgent(ctx context.Context, payload webhookAgentPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("marshal payload: %w", err)}
	}
	return w.post(ctx, body)
}

func (w *WebhookNotifier) send(ctx context.Context, payload webhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("marshal payload: %w", err)}
	}
	return w.post(ctx, body)
}

// post handles the shared HTTP logic for both incident and agent payloads,
// including signing when a secret is configured.
func (w *WebhookNotifier) post(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	if w.signingSecret != "" {
		ts := time.Now()
		nonce := GenerateNonce()
		sig := SignWebhookPayload(w.signingSecret, ts, nonce, body)
		req.Header.Set("X-Watchdog-Signature-256", "sha256="+sig)
		req.Header.Set("X-Watchdog-Timestamp", strconv.FormatInt(ts.Unix(), 10))
		req.Header.Set("X-Watchdog-Nonce", nonce)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "webhook", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}
	return nil
}

type webhookPayload struct {
	Event     string               `json:"event"`
	Timestamp time.Time            `json:"timestamp"`
	Incident  webhookIncident      `json:"incident"`
	Monitor   webhookMonitor       `json:"monitor"`
	Context   *webhookAlertContext `json:"context,omitempty"`
}

type webhookIncident struct {
	ID         string     `json:"id"`
	MonitorID  string     `json:"monitor_id"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type webhookMonitor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Target string `json:"target"`
}

type webhookAlertContext struct {
	ErrorMessage string `json:"error_message,omitempty"`
	AgentName    string `json:"agent_name,omitempty"`
	Interval     string `json:"interval,omitempty"`
	Threshold    int    `json:"threshold,omitempty"`
}

func buildWebhookContext(incident *domain.Incident) *webhookAlertContext {
	ac := incident.AlertContext
	if ac == nil {
		return nil
	}
	wctx := &webhookAlertContext{
		ErrorMessage: ac.ErrorMessage,
		AgentName:    ac.AgentName,
		Threshold:    ac.Threshold,
	}
	if ac.Interval > 0 {
		wctx.Interval = formatInterval(ac.Interval)
	}
	return wctx
}

type webhookAgentPayload struct {
	Event             string    `json:"event_type"`
	Timestamp         time.Time `json:"timestamp"`
	AgentID           string    `json:"agent_id"`
	AgentName         string    `json:"agent_name"`
	AffectedMonitors  int       `json:"affected_monitors,omitempty"`
	ResolvedIncidents int       `json:"resolved_incidents,omitempty"`
	WindowName        string    `json:"window_name,omitempty"`
}
