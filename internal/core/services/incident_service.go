package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// IncidentService implements ports.IncidentService for incident lifecycle management.
type IncidentService struct {
	incidentRepo ports.IncidentRepository
	monitorRepo  ports.MonitorRepository
	notifier     ports.Notifier
	transactor   ports.Transactor
	logger       *slog.Logger
}

// NewIncidentService creates a new IncidentService.
func NewIncidentService(
	incidentRepo ports.IncidentRepository,
	monitorRepo ports.MonitorRepository,
	notifier ports.Notifier,
	transactor ports.Transactor,
	logger *slog.Logger,
) *IncidentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &IncidentService{
		incidentRepo: incidentRepo,
		monitorRepo:  monitorRepo,
		notifier:     notifier,
		transactor:   transactor,
		logger:       logger,
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

	// Send notification (don't fail the operation if notification fails)
	if monitor != nil && incident != nil {
		if notifyErr := s.notifier.NotifyIncidentResolved(ctx, incident, monitor); notifyErr != nil {
			s.logger.Error("failed to send incident resolved notification",
				"incident_id", id,
				"monitor_id", incident.MonitorID,
				"error", notifyErr,
			)
		}
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

	// Send notification (don't fail the operation if notification fails)
	if notifyErr := s.notifier.NotifyIncidentOpened(ctx, incident, monitor); notifyErr != nil {
		s.logger.Error("failed to send incident opened notification",
			"incident_id", incident.ID,
			"monitor_id", monitorID,
			"error", notifyErr,
		)
	}

	return incident, nil
}
