package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// EmailNotifier sends notifications via SMTP email.
type EmailNotifier struct {
	host     string
	port     int
	username string
	password string
	from     string
	to       string
}

// EmailConfig holds SMTP configuration.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       string
}

// NewEmailNotifier creates a new email notifier.
func NewEmailNotifier(cfg EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
		to:       cfg.To,
	}
}

// NotifyIncidentOpened sends an email when an incident is opened.
func (e *EmailNotifier) NotifyIncidentOpened(_ context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	subject := fmt.Sprintf("[%s] Incident Opened: %s is DOWN", BrandName, monitor.Name)

	extra := ""
	if ac := incident.AlertContext; ac != nil {
		if ac.ErrorMessage != "" {
			extra += fmt.Sprintf("Error: %s\n", ac.ErrorMessage)
		}
		if ac.AgentName != "" {
			extra += fmt.Sprintf("Agent: %s\n", ac.AgentName)
		}
		if ac.Interval > 0 {
			extra += fmt.Sprintf("Interval: %s\n", formatInterval(ac.Interval))
		}
	}

	body := fmt.Sprintf(
		"Monitor: %s\nType: %s\nTarget: %s\n%sStarted: %s\n\nMonitor %s is currently DOWN.\n\n— %s",
		monitor.Name,
		string(monitor.Type),
		monitor.Target,
		extra,
		incident.StartedAt.Format(time.RFC3339),
		monitor.Name,
		BrandName,
	)

	return e.send(subject, body)
}

// NotifyIncidentResolved sends an email when an incident is resolved.
func (e *EmailNotifier) NotifyIncidentResolved(_ context.Context, incident *domain.Incident, monitor *domain.Monitor) error {
	subject := fmt.Sprintf("[%s] Incident Resolved: %s is UP", BrandName, monitor.Name)

	extra := ""
	if ac := incident.AlertContext; ac != nil {
		if ac.AgentName != "" {
			extra += fmt.Sprintf("Agent: %s\n", ac.AgentName)
		}
	}

	body := fmt.Sprintf(
		"Monitor: %s\nType: %s\nTarget: %s\n%sStarted: %s\nDuration: %s\n\nMonitor %s is back UP.\n\n— %s",
		monitor.Name,
		string(monitor.Type),
		monitor.Target,
		extra,
		incident.StartedAt.Format(time.RFC3339),
		formatDuration(incident.Duration()),
		monitor.Name,
		BrandName,
	)

	return e.send(subject, body)
}

// NotifyAgentOffline sends an email when an agent goes offline.
func (e *EmailNotifier) NotifyAgentOffline(_ context.Context, agent *domain.Agent, affectedMonitors int) error {
	subject := fmt.Sprintf("[%s] Agent Offline: %s", BrandName, agent.Name)
	body := fmt.Sprintf(
		"Agent: %s\nStatus: Offline\nAffected Monitors: %d\n\nAgent %s has disconnected.\n\n— %s",
		agent.Name,
		affectedMonitors,
		agent.Name,
		BrandName,
	)

	return e.send(subject, body)
}

// NotifyAgentOnline sends an email when an agent comes back online.
func (e *EmailNotifier) NotifyAgentOnline(_ context.Context, agent *domain.Agent, resolvedIncidents int) error {
	subject := fmt.Sprintf("[%s] Agent Online: %s", BrandName, agent.Name)
	body := fmt.Sprintf(
		"Agent: %s\nStatus: Online\nResolved Incidents: %d\n\nAgent %s has reconnected.\n\n— %s",
		agent.Name,
		resolvedIncidents,
		agent.Name,
		BrandName,
	)

	return e.send(subject, body)
}

// NotifyAgentMaintenance sends an email when an agent enters maintenance mode.
func (e *EmailNotifier) NotifyAgentMaintenance(_ context.Context, agent *domain.Agent, windowName string) error {
	subject := fmt.Sprintf("[%s] Maintenance Mode: %s", BrandName, agent.Name)
	body := fmt.Sprintf(
		"Agent: %s\nStatus: Maintenance Mode\nWindow: %s\n\nAgent %s entered maintenance mode. Alerts are suppressed until the window expires.\n\n— %s",
		agent.Name,
		windowName,
		agent.Name,
		BrandName,
	)

	return e.send(subject, body)
}

func (e *EmailNotifier) send(subject, body string) error {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		e.from, e.to, subject, body,
	)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)

	if err := smtp.SendMail(addr, auth, e.from, []string{e.to}, []byte(msg)); err != nil {
		return &NotifierError{Notifier: "email", Err: fmt.Errorf("send mail: %w", err)}
	}

	return nil
}
