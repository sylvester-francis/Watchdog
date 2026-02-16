package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/adapters/notify"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// IncidentService implements ports.IncidentService for incident lifecycle management.
type IncidentService struct {
	incidentRepo     ports.IncidentRepository
	monitorRepo      ports.MonitorRepository
	agentRepo        ports.AgentRepository
	alertChannelRepo ports.AlertChannelRepository
	notifier         ports.Notifier // global notifier (env-based, server admin fallback)
	transactor       ports.Transactor
	logger           *slog.Logger
}

// NewIncidentService creates a new IncidentService.
func NewIncidentService(
	incidentRepo ports.IncidentRepository,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	alertChannelRepo ports.AlertChannelRepository,
	notifier ports.Notifier,
	transactor ports.Transactor,
	logger *slog.Logger,
) *IncidentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &IncidentService{
		incidentRepo:     incidentRepo,
		monitorRepo:      monitorRepo,
		agentRepo:        agentRepo,
		alertChannelRepo: alertChannelRepo,
		notifier:         notifier,
		transactor:       transactor,
		logger:           logger,
	}
}

// GetIncident retrieves an incident by ID.
func (s *IncidentService) GetIncident(ctx context.Context, id uuid.UUID) (*domain.Incident, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("incidentService.GetIncident: %w", err)
	}
	return incident, nil
}

// GetActiveIncidents retrieves all active (open or acknowledged) incidents.
func (s *IncidentService) GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error) {
	incidents, err := s.incidentRepo.GetActiveIncidents(ctx)
	if err != nil {
		return nil, fmt.Errorf("incidentService.GetActiveIncidents: %w", err)
	}
	return incidents, nil
}

// GetResolvedIncidents retrieves all resolved incidents.
func (s *IncidentService) GetResolvedIncidents(ctx context.Context) ([]*domain.Incident, error) {
	incidents, err := s.incidentRepo.GetResolvedIncidents(ctx)
	if err != nil {
		return nil, fmt.Errorf("incidentService.GetResolvedIncidents: %w", err)
	}
	return incidents, nil
}

// GetAllIncidents retrieves all incidents.
func (s *IncidentService) GetAllIncidents(ctx context.Context) ([]*domain.Incident, error) {
	incidents, err := s.incidentRepo.GetAllIncidents(ctx)
	if err != nil {
		return nil, fmt.Errorf("incidentService.GetAllIncidents: %w", err)
	}
	return incidents, nil
}

// GetIncidentsByMonitor retrieves all incidents for a monitor.
func (s *IncidentService) GetIncidentsByMonitor(ctx context.Context, monitorID uuid.UUID) ([]*domain.Incident, error) {
	incidents, err := s.incidentRepo.GetByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, fmt.Errorf("incidentService.GetIncidentsByMonitor: %w", err)
	}
	return incidents, nil
}

// AcknowledgeIncident marks an incident as acknowledged by a user.
func (s *IncidentService) AcknowledgeIncident(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := s.incidentRepo.Acknowledge(ctx, id, userID); err != nil {
		return fmt.Errorf("incidentService.AcknowledgeIncident: %w", err)
	}
	return nil
}

// ResolveIncident marks an incident as resolved.
func (s *IncidentService) ResolveIncident(ctx context.Context, id uuid.UUID) error {
	// Get the incident first to send notification
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("incidentService.ResolveIncident: get incident: %w", err)
	}
	if incident == nil {
		return fmt.Errorf("incidentService.ResolveIncident: incident not found")
	}

	// Get the monitor for notification context
	monitor, err := s.monitorRepo.GetByID(ctx, incident.MonitorID)
	if err != nil {
		return fmt.Errorf("incidentService.ResolveIncident: get monitor: %w", err)
	}

	// Resolve the incident in a transaction with monitor status update
	err = s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		// Resolve the incident
		if err := s.incidentRepo.Resolve(txCtx, id); err != nil {
			return fmt.Errorf("resolve incident: %w", err)
		}

		// Update monitor status to up
		if err := s.monitorRepo.UpdateStatus(txCtx, incident.MonitorID, domain.MonitorStatusUp); err != nil {
			return fmt.Errorf("update monitor status: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("incidentService.ResolveIncident: %w", err)
	}

	// Refresh the incident to get updated resolved_at and ttr_seconds
	incident, err = s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Warn("failed to refresh incident for notification", "error", err)
	}

	// Send notifications (global + per-user, don't fail the operation)
	if monitor != nil && incident != nil {
		s.notifyAll(ctx, incident, monitor, false)
	}

	return nil
}

// CreateIncidentIfNeeded creates a new incident for a monitor if there isn't already an open one.
// Returns the existing incident if one is already open, or the newly created incident.
func (s *IncidentService) CreateIncidentIfNeeded(ctx context.Context, monitorID uuid.UUID) (*domain.Incident, error) {
	// Check if there's already an open incident
	existing, err := s.incidentRepo.GetOpenByMonitorID(ctx, monitorID)
	if err != nil {
		return nil, fmt.Errorf("incidentService.CreateIncidentIfNeeded: check existing: %w", err)
	}
	if existing != nil {
		// Return existing open incident
		return existing, nil
	}

	// Get the monitor for notification context
	monitor, err := s.monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, fmt.Errorf("incidentService.CreateIncidentIfNeeded: get monitor: %w", err)
	}
	if monitor == nil {
		return nil, fmt.Errorf("incidentService.CreateIncidentIfNeeded: monitor not found")
	}

	// Create new incident in a transaction with monitor status update
	incident := domain.NewIncident(monitorID)

	err = s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create the incident
		if err := s.incidentRepo.Create(txCtx, incident); err != nil {
			return fmt.Errorf("create incident: %w", err)
		}

		// Update monitor status to down
		if err := s.monitorRepo.UpdateStatus(txCtx, monitorID, domain.MonitorStatusDown); err != nil {
			return fmt.Errorf("update monitor status: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("incidentService.CreateIncidentIfNeeded: %w", err)
	}

	// Send notifications (global + per-user, don't fail the operation)
	s.notifyAll(ctx, incident, monitor, true)

	return incident, nil
}

// notifyAll sends notifications via the global notifier and all per-user alert channels.
func (s *IncidentService) notifyAll(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor, opened bool) {
	// 1. Global notifier (env-based, server admin)
	var globalErr error
	if opened {
		globalErr = s.notifier.NotifyIncidentOpened(ctx, incident, monitor)
	} else {
		globalErr = s.notifier.NotifyIncidentResolved(ctx, incident, monitor)
	}
	if globalErr != nil {
		s.logger.Error("global notification failed",
			"incident_id", incident.ID,
			"monitor_id", monitor.ID,
			"error", globalErr,
		)
	}

	// 2. Per-user notifications: monitor → agent → user → channels
	agent, err := s.agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil || agent == nil {
		s.logger.Error("failed to get agent for per-user notifications",
			"agent_id", monitor.AgentID,
			"error", err,
		)
		return
	}

	channels, err := s.alertChannelRepo.GetEnabledByUserID(ctx, agent.UserID)
	if err != nil {
		s.logger.Error("failed to get alert channels",
			"user_id", agent.UserID,
			"error", err,
		)
		return
	}

	for _, ch := range channels {
		notifier, err := notify.BuildFromChannel(ch)
		if err != nil {
			s.logger.Error("failed to build notifier from channel",
				"channel_id", ch.ID,
				"channel_type", ch.Type,
				"error", err,
			)
			continue
		}

		var notifyErr error
		if opened {
			notifyErr = notifier.NotifyIncidentOpened(ctx, incident, monitor)
		} else {
			notifyErr = notifier.NotifyIncidentResolved(ctx, incident, monitor)
		}
		if notifyErr != nil {
			s.logger.Error("per-user notification failed",
				"channel_id", ch.ID,
				"channel_name", ch.Name,
				"channel_type", ch.Type,
				"error", notifyErr,
			)
		}
	}
}
