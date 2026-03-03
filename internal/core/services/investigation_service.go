package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// InvestigationService aggregates existing data to produce incident investigations.
// All operations are read-only — no mutations.
type InvestigationService struct {
	incidentRepo    ports.IncidentRepository
	monitorRepo     ports.MonitorRepository
	agentRepo       ports.AgentRepository
	heartbeatRepo   ports.HeartbeatRepository
	certDetailsRepo ports.CertDetailsRepository
	logger          *slog.Logger
}

// NewInvestigationService creates a new InvestigationService.
func NewInvestigationService(
	incidentRepo ports.IncidentRepository,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	heartbeatRepo ports.HeartbeatRepository,
	certDetailsRepo ports.CertDetailsRepository,
	logger *slog.Logger,
) *InvestigationService {
	return &InvestigationService{
		incidentRepo:    incidentRepo,
		monitorRepo:     monitorRepo,
		agentRepo:       agentRepo,
		heartbeatRepo:   heartbeatRepo,
		certDetailsRepo: certDetailsRepo,
		logger:          logger,
	}
}

// Investigate builds an IncidentInvestigation by aggregating data from existing repos.
func (s *InvestigationService) Investigate(ctx context.Context, incidentID uuid.UUID) (*domain.IncidentInvestigation, error) {
	// 1. Get incident
	incident, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}
	if incident == nil {
		return nil, nil
	}

	// 2. Get monitor
	monitor, err := s.monitorRepo.GetByID(ctx, incident.MonitorID)
	if err != nil {
		return nil, fmt.Errorf("get monitor: %w", err)
	}
	if monitor == nil {
		return nil, fmt.Errorf("monitor %s not found for incident %s", incident.MonitorID, incidentID)
	}

	// 3. Get agent
	agent, err := s.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	// 4. Get heartbeats in window: incident.StartedAt -5min to +10min (or resolved_at +2min)
	windowStart := incident.StartedAt.Add(-5 * time.Minute)
	windowEnd := incident.StartedAt.Add(10 * time.Minute)
	if incident.ResolvedAt != nil {
		windowEnd = incident.ResolvedAt.Add(2 * time.Minute)
	}
	heartbeats, err := s.heartbeatRepo.GetByMonitorIDInRange(ctx, incident.MonitorID, windowStart, windowEnd)
	if err != nil {
		s.logger.Error("failed to get heartbeats for investigation",
			slog.String("incident_id", incidentID.String()),
			slog.String("error", err.Error()),
		)
		heartbeats = nil
	}

	// 5. Get sibling monitors on the same agent
	var siblings []domain.MonitorWithStatus
	if agent != nil {
		agentMonitors, err := s.monitorRepo.GetByAgentID(ctx, monitor.AgentID)
		if err == nil {
			for _, m := range agentMonitors {
				if m.ID == monitor.ID {
					continue
				}
				hasIncident := false
				activeInc, _ := s.incidentRepo.GetActiveByMonitorID(ctx, m.ID)
				if activeInc != nil {
					hasIncident = true
				}
				siblings = append(siblings, domain.MonitorWithStatus{
					ID:          m.ID,
					Name:        m.Name,
					Type:        m.Type,
					Target:      m.Target,
					Status:      m.Status,
					HasIncident: hasIncident,
				})
			}
		}
	}

	// 6. Get previous incidents on this monitor (limit 10)
	allIncidents, err := s.incidentRepo.GetByMonitorID(ctx, incident.MonitorID)
	if err != nil {
		s.logger.Error("failed to get previous incidents",
			slog.String("error", err.Error()),
		)
		allIncidents = nil
	}
	var previousIncidents []*domain.Incident
	for _, inc := range allIncidents {
		if inc.ID == incident.ID {
			continue
		}
		previousIncidents = append(previousIncidents, inc)
		if len(previousIncidents) >= 10 {
			break
		}
	}

	// 7. Detect recurrence pattern
	pattern := detectRecurrencePattern(len(previousIncidents))

	// 8. Calculate MTTR from resolved previous incidents
	mttr := calculateMTTR(previousIncidents)

	// 9. Get system metrics: find system monitors on the same agent
	var systemMetrics []domain.SystemMetricSnapshot
	if agent != nil {
		for _, m := range siblings {
			if m.Type != domain.MonitorTypeSystem {
				continue
			}
			sysHeartbeats, err := s.heartbeatRepo.GetByMonitorIDInRange(ctx, m.ID, windowStart, windowEnd)
			if err != nil || len(sysHeartbeats) == 0 {
				continue
			}
			// Take the heartbeat closest to the incident start
			closest := sysHeartbeats[len(sysHeartbeats)-1]
			value := ""
			if closest.ErrorMessage != nil {
				value = *closest.ErrorMessage
			}
			systemMetrics = append(systemMetrics, domain.SystemMetricSnapshot{
				MonitorName: m.Name,
				Target:      m.Target,
				Value:       value,
				Status:      string(closest.Status),
				Time:        closest.Time,
			})
		}
	}

	// 10. Get cert details if TLS monitor
	var certDetails *domain.CertDetails
	if monitor.Type == domain.MonitorTypeTLS && s.certDetailsRepo != nil {
		certDetails, _ = s.certDetailsRepo.GetByMonitorID(ctx, monitor.ID)
	}

	// 11. Build timeline
	timeline := buildTimeline(incident, heartbeats)

	// Build agent summary
	var agentSummary domain.AgentSummary
	if agent != nil {
		agentSummary = domain.AgentSummary{
			ID:     agent.ID,
			Name:   agent.Name,
			Status: agent.Status,
		}
	}

	return &domain.IncidentInvestigation{
		Incident:          incident,
		Monitor:           monitor,
		Agent:             agent,
		AgentSummary:      agentSummary,
		Heartbeats:        heartbeats,
		SiblingMonitors:   siblings,
		PreviousIncidents: previousIncidents,
		RecurrencePattern: pattern,
		MTTRSeconds:       mttr,
		SystemMetrics:     systemMetrics,
		CertDetails:       certDetails,
		Timeline:          timeline,
	}, nil
}

// detectRecurrencePattern classifies the incident recurrence pattern.
func detectRecurrencePattern(previousCount int) string {
	switch {
	case previousCount == 0:
		return "first_time"
	case previousCount <= 4:
		return "recurring"
	default:
		return "frequent"
	}
}

// calculateMTTR computes mean time to resolve from resolved previous incidents.
func calculateMTTR(incidents []*domain.Incident) *int {
	var totalTTR, count int
	for _, inc := range incidents {
		if inc.TTRSeconds != nil {
			totalTTR += *inc.TTRSeconds
			count++
		}
	}
	if count == 0 {
		return nil
	}
	avg := totalTTR / count
	return &avg
}

// buildTimeline merges heartbeat events and incident lifecycle events chronologically.
func buildTimeline(incident *domain.Incident, heartbeats []*domain.Heartbeat) []domain.TimelineEvent {
	var events []domain.TimelineEvent

	// Add incident lifecycle events
	events = append(events, domain.TimelineEvent{
		Time:        incident.StartedAt,
		Type:        "incident_opened",
		Description: "Incident opened",
		Severity:    "error",
	})

	if incident.AcknowledgedAt != nil {
		events = append(events, domain.TimelineEvent{
			Time:        *incident.AcknowledgedAt,
			Type:        "incident_acknowledged",
			Description: "Incident acknowledged",
			Severity:    "warning",
		})
	}

	if incident.ResolvedAt != nil {
		events = append(events, domain.TimelineEvent{
			Time:        *incident.ResolvedAt,
			Type:        "incident_resolved",
			Description: "Incident resolved",
			Severity:    "info",
		})
	}

	// Add heartbeat events
	for _, hb := range heartbeats {
		severity := "info"
		desc := "Heartbeat: up"
		eventType := "heartbeat_success"

		if hb.Status.IsFailure() {
			severity = "error"
			eventType = "heartbeat_fail"
			desc = fmt.Sprintf("Heartbeat: %s", hb.Status)
			if hb.ErrorMessage != nil && *hb.ErrorMessage != "" {
				desc = fmt.Sprintf("Heartbeat: %s — %s", hb.Status, *hb.ErrorMessage)
			}
		}

		events = append(events, domain.TimelineEvent{
			Time:        hb.Time,
			Type:        eventType,
			Description: desc,
			Severity:    severity,
		})
	}

	// Sort chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})

	return events
}
