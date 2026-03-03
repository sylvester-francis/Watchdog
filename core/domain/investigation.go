package domain

import (
	"time"

	"github.com/google/uuid"
)

// IncidentInvestigation aggregates existing data into a single investigation view.
// No new DB tables — this is a read-only projection of heartbeats, incidents,
// monitors, agents, and system metrics around the time of an incident.
type IncidentInvestigation struct {
	Incident          *Incident            `json:"incident"`
	Monitor           *Monitor             `json:"monitor"`
	Agent             *Agent               `json:"-"`
	AgentSummary      AgentSummary         `json:"agent"`
	Heartbeats        []*Heartbeat         `json:"heartbeats"`
	SiblingMonitors   []MonitorWithStatus  `json:"sibling_monitors"`
	PreviousIncidents []*Incident          `json:"previous_incidents"`
	RecurrencePattern string               `json:"recurrence_pattern"`
	MTTRSeconds       *int                 `json:"mttr_seconds"`
	SystemMetrics     []SystemMetricSnapshot `json:"system_metrics"`
	CertDetails       *CertDetails         `json:"cert_details,omitempty"`
	Timeline          []TimelineEvent      `json:"timeline"`
}

// AgentSummary is a safe-to-serialize subset of Agent for API responses.
type AgentSummary struct {
	ID     uuid.UUID   `json:"id"`
	Name   string      `json:"name"`
	Status AgentStatus `json:"status"`
}

// MonitorWithStatus pairs a monitor with whether it has an active incident.
type MonitorWithStatus struct {
	ID            uuid.UUID     `json:"id"`
	Name          string        `json:"name"`
	Type          MonitorType   `json:"type"`
	Target        string        `json:"target"`
	Status        MonitorStatus `json:"status"`
	HasIncident   bool          `json:"has_incident"`
}

// SystemMetricSnapshot captures a system metric reading around incident time.
type SystemMetricSnapshot struct {
	MonitorName string    `json:"monitor_name"`
	Target      string    `json:"target"`
	Value       string    `json:"value"`
	Status      string    `json:"status"`
	Time        time.Time `json:"time"`
}

// TimelineEvent represents a chronological event in the investigation timeline.
type TimelineEvent struct {
	Time        time.Time `json:"time"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
}
