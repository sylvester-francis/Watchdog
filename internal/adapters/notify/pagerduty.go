package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sylvester-francis/watchdog/core/domain"
)

const pagerdutyDefaultEventsURL = "https://events.pagerduty.com/v2/enqueue"

// PagerDutyNotifier sends notifications via PagerDuty Events API v2.
type PagerDutyNotifier struct {
	routingKey string
	eventsURL  string
	httpClient *http.Client
}

// NewPagerDutyNotifier creates a new PagerDuty notifier.
func NewPagerDutyNotifier(routingKey string) *PagerDutyNotifier {
	return &PagerDutyNotifier{
		routingKey: routingKey,
		eventsURL:  pagerdutyDefaultEventsURL,
		httpClient: NewHTTPClient(10 * time.Second),
	}
}

// SetEventsURL overrides the PagerDuty events URL (useful for testing).
func (p *PagerDutyNotifier) SetEventsURL(url string) {
	p.eventsURL = url
}

// SetHTTPClient overrides the HTTP client (useful for testing).
func (p *PagerDutyNotifier) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// NotifyIncidentOpened sends a trigger event to PagerDuty.
func (p *PagerDutyNotifier) NotifyIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	details := map[string]string{
		"monitor_name": monitor.Name,
		"monitor_type": string(monitor.Type),
		"target":       monitor.Target,
	}

	if ac := incident.AlertContext; ac != nil {
		if ac.ErrorMessage != "" {
			details["error_message"] = ac.ErrorMessage
		}
		if ac.AgentName != "" {
			details["agent_name"] = ac.AgentName
		}
		if ac.Interval > 0 {
			details["interval"] = formatInterval(ac.Interval)
		}
	}

	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "trigger",
		DedupKey:    incident.ID.String(),
		Payload: pagerdutyPayload{
			Summary:       fmt.Sprintf("Monitor %s is DOWN (%s)", monitor.Name, monitor.Target),
			Source:        BrandName,
			Severity:      "critical",
			Timestamp:     incident.StartedAt.Format(time.RFC3339),
			CustomDetails: details,
		},
	}

	return p.send(ctx, payload)
}

// NotifyIncidentResolved sends a resolve event to PagerDuty.
func (p *PagerDutyNotifier) NotifyIncidentResolved(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	details := map[string]string{
		"monitor_name": monitor.Name,
		"duration":     formatDuration(incident.Duration()),
	}

	if ac := incident.AlertContext; ac != nil {
		if ac.AgentName != "" {
			details["agent_name"] = ac.AgentName
		}
	}

	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "resolve",
		DedupKey:    incident.ID.String(),
		Payload: pagerdutyPayload{
			Summary:       fmt.Sprintf("Monitor %s is UP (%s)", monitor.Name, monitor.Target),
			Source:        BrandName,
			Severity:      "info",
			Timestamp:     time.Now().Format(time.RFC3339),
			CustomDetails: details,
		},
	}

	return p.send(ctx, payload)
}

// NotifyAgentOffline sends a trigger event to PagerDuty when an agent goes offline.
func (p *PagerDutyNotifier) NotifyAgentOffline(ctx context.Context, agent *domain.Agent, affectedMonitors int) error {
	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "trigger",
		DedupKey:    fmt.Sprintf("agent-offline-%s", agent.ID.String()),
		Payload: pagerdutyPayload{
			Summary:   fmt.Sprintf("Agent %s is offline (%d monitors affected)", agent.Name, affectedMonitors),
			Source:    BrandName,
			Severity:  "warning",
			Timestamp: time.Now().Format(time.RFC3339),
			CustomDetails: map[string]string{
				"agent_name":        agent.Name,
				"agent_id":          agent.ID.String(),
				"affected_monitors": fmt.Sprintf("%d", affectedMonitors),
			},
		},
	}

	return p.send(ctx, payload)
}

// NotifyAgentOnline sends a resolve event to PagerDuty when an agent comes back online.
func (p *PagerDutyNotifier) NotifyAgentOnline(ctx context.Context, agent *domain.Agent, resolvedIncidents int) error {
	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "resolve",
		DedupKey:    fmt.Sprintf("agent-offline-%s", agent.ID.String()),
		Payload: pagerdutyPayload{
			Summary:   fmt.Sprintf("Agent %s is back online (%d incidents resolved)", agent.Name, resolvedIncidents),
			Source:    BrandName,
			Severity:  "info",
			Timestamp: time.Now().Format(time.RFC3339),
			CustomDetails: map[string]string{
				"agent_name":         agent.Name,
				"agent_id":           agent.ID.String(),
				"resolved_incidents": fmt.Sprintf("%d", resolvedIncidents),
			},
		},
	}

	return p.send(ctx, payload)
}

// NotifyAgentMaintenance sends an info event to PagerDuty when an agent enters maintenance mode.
func (p *PagerDutyNotifier) NotifyAgentMaintenance(ctx context.Context, agent *domain.Agent, windowName string) error {
	payload := pagerdutyEvent{
		RoutingKey:  p.routingKey,
		EventAction: "trigger",
		DedupKey:    fmt.Sprintf("agent-maintenance-%s", agent.ID.String()),
		Payload: pagerdutyPayload{
			Summary:   fmt.Sprintf("Agent %s entered maintenance mode (window: %s)", agent.Name, windowName),
			Source:    BrandName,
			Severity:  "info",
			Timestamp: time.Now().Format(time.RFC3339),
			CustomDetails: map[string]string{
				"agent_name":  agent.Name,
				"agent_id":    agent.ID.String(),
				"window_name": windowName,
			},
		},
	}

	return p.send(ctx, payload)
}

func (p *PagerDutyNotifier) send(ctx context.Context, event pagerdutyEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("marshal event: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.eventsURL, bytes.NewReader(body))
	if err != nil {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("send request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &NotifierError{Notifier: "pagerduty", Err: fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
	}

	return nil
}

type pagerdutyEvent struct {
	RoutingKey  string           `json:"routing_key"`
	EventAction string           `json:"event_action"`
	DedupKey    string           `json:"dedup_key"`
	Payload     pagerdutyPayload `json:"payload"`
}

type pagerdutyPayload struct {
	Summary       string            `json:"summary"`
	Source        string            `json:"source"`
	Severity      string            `json:"severity"`
	Timestamp     string            `json:"timestamp"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}
