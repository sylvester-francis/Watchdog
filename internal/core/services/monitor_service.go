package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// FailureThreshold is the number of consecutive failures required to trigger an incident.
// This implements the "3-strike rule" from the WatchDog requirements.
const FailureThreshold = 3

// MonitorService implements ports.MonitorService for monitor orchestration.
type MonitorService struct {
	monitorRepo    ports.MonitorRepository
	heartbeatRepo  ports.HeartbeatRepository
	incidentRepo   ports.IncidentRepository
	incidentSvc    ports.IncidentService
	userRepo       ports.UserRepository
	usageEventRepo ports.UsageEventRepository
	logger         *slog.Logger
}

// NewMonitorService creates a new MonitorService.
func NewMonitorService(
	monitorRepo ports.MonitorRepository,
	heartbeatRepo ports.HeartbeatRepository,
	incidentRepo ports.IncidentRepository,
	incidentSvc ports.IncidentService,
	userRepo ports.UserRepository,
	usageEventRepo ports.UsageEventRepository,
	logger *slog.Logger,
) *MonitorService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MonitorService{
		monitorRepo:    monitorRepo,
		heartbeatRepo:  heartbeatRepo,
		incidentRepo:   incidentRepo,
		incidentSvc:    incidentSvc,
		userRepo:       userRepo,
		usageEventRepo: usageEventRepo,
		logger:         logger,
	}
}

// CreateMonitor creates a new monitor for an agent, enforcing plan limits.
func (s *MonitorService) CreateMonitor(ctx context.Context, userID uuid.UUID, agentID uuid.UUID, name string, monitorType domain.MonitorType, target string, metadata map[string]string) (*domain.Monitor, error) {
	// Enforce plan limits
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("monitorService.CreateMonitor: get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("monitorService.CreateMonitor: user not found")
	}

	limits := user.Plan.Limits()
	if limits.MaxMonitors != -1 {
		count, err := s.monitorRepo.CountByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("monitorService.CreateMonitor: count monitors: %w", err)
		}
		if count >= limits.MaxMonitors {
			event := domain.NewUsageEvent(userID, domain.EventLimitHit, domain.ResourceMonitor, count, limits.MaxMonitors, user.Plan)
			if err := s.usageEventRepo.Create(ctx, event); err != nil {
				s.logger.Warn("failed to record limit_hit event", "error", err)
			}
			return nil, domain.ErrMonitorLimitReached
		}
		if float64(count) >= float64(limits.MaxMonitors)*0.8 {
			event := domain.NewUsageEvent(userID, domain.EventApproachingLimit, domain.ResourceMonitor, count, limits.MaxMonitors, user.Plan)
			if err := s.usageEventRepo.Create(ctx, event); err != nil {
				s.logger.Warn("failed to record approaching_limit event", "error", err)
			}
		}
	}

	monitor := domain.NewMonitor(agentID, name, monitorType, target)
	if metadata != nil {
		monitor.Metadata = metadata
	}

	if err := s.monitorRepo.Create(ctx, monitor); err != nil {
		return nil, fmt.Errorf("monitorService.CreateMonitor: %w", err)
	}

	return monitor, nil
}

// GetMonitor retrieves a monitor by ID.
func (s *MonitorService) GetMonitor(ctx context.Context, id uuid.UUID) (*domain.Monitor, error) {
	monitor, err := s.monitorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("monitorService.GetMonitor: %w", err)
	}
	return monitor, nil
}

// GetMonitorsByAgent retrieves all monitors for an agent.
func (s *MonitorService) GetMonitorsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	monitors, err := s.monitorRepo.GetByAgentID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("monitorService.GetMonitorsByAgent: %w", err)
	}
	return monitors, nil
}

// UpdateMonitor updates an existing monitor.
func (s *MonitorService) UpdateMonitor(ctx context.Context, monitor *domain.Monitor) error {
	if err := s.monitorRepo.Update(ctx, monitor); err != nil {
		return fmt.Errorf("monitorService.UpdateMonitor: %w", err)
	}
	return nil
}

// DeleteMonitor deletes a monitor.
func (s *MonitorService) DeleteMonitor(ctx context.Context, id uuid.UUID) error {
	if err := s.monitorRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("monitorService.DeleteMonitor: %w", err)
	}
	return nil
}

// ProcessHeartbeat processes a heartbeat from an agent.
// This is the core method that implements the 3-strike rule:
// - If the heartbeat is successful, check if we should resolve an open incident
// - If the heartbeat is a failure, check if we've hit the failure threshold
func (s *MonitorService) ProcessHeartbeat(ctx context.Context, heartbeat *domain.Heartbeat) error {
	// 1. Store the heartbeat
	if err := s.heartbeatRepo.Create(ctx, heartbeat); err != nil {
		return fmt.Errorf("monitorService.ProcessHeartbeat: store heartbeat: %w", err)
	}

	// 2. Handle success or failure
	if heartbeat.IsSuccess() {
		return s.handleRecovery(ctx, heartbeat.MonitorID)
	}

	return s.handleFailure(ctx, heartbeat.MonitorID)
}

// handleRecovery handles a successful heartbeat, potentially resolving an open incident.
func (s *MonitorService) handleRecovery(ctx context.Context, monitorID uuid.UUID) error {
	// Check for an open incident
	incident, err := s.incidentRepo.GetOpenByMonitorID(ctx, monitorID)
	if err != nil {
		return fmt.Errorf("check open incident: %w", err)
	}

	// No open incident, just update status to up
	if incident == nil {
		if err := s.monitorRepo.UpdateStatus(ctx, monitorID, domain.MonitorStatusUp); err != nil {
			s.logger.Warn("failed to update monitor status to up",
				"monitor_id", monitorID,
				"error", err,
			)
		}
		return nil
	}

	// Resolve the incident (this also updates monitor status)
	if err := s.incidentSvc.ResolveIncident(ctx, incident.ID); err != nil {
		return fmt.Errorf("resolve incident: %w", err)
	}

	s.logger.Info("incident resolved due to recovery",
		"incident_id", incident.ID,
		"monitor_id", monitorID,
	)

	return nil
}

// handleFailure handles a failed heartbeat, potentially creating an incident.
// Implements the 3-strike rule: only create an incident after FailureThreshold consecutive failures.
func (s *MonitorService) handleFailure(ctx context.Context, monitorID uuid.UUID) error {
	// Check if there's already an open incident
	existing, err := s.incidentRepo.GetOpenByMonitorID(ctx, monitorID)
	if err != nil {
		return fmt.Errorf("check existing incident: %w", err)
	}

	// If incident already exists, just update status and return
	if existing != nil {
		// Incident already open, nothing more to do
		return nil
	}

	// Check recent heartbeats to see if we've hit the threshold
	// We need to verify we have FailureThreshold consecutive failures
	recentHeartbeats, err := s.heartbeatRepo.GetByMonitorID(ctx, monitorID, FailureThreshold)
	if err != nil {
		return fmt.Errorf("get recent heartbeats: %w", err)
	}

	// Not enough heartbeats yet
	if len(recentHeartbeats) < FailureThreshold {
		s.logger.Debug("not enough heartbeats for threshold",
			"monitor_id", monitorID,
			"count", len(recentHeartbeats),
			"threshold", FailureThreshold,
		)
		return nil
	}

	// Check if all recent heartbeats are failures
	allFailures := true
	for _, hb := range recentHeartbeats {
		if hb.IsSuccess() {
			allFailures = false
			break
		}
	}

	if !allFailures {
		// Not enough consecutive failures yet
		s.logger.Debug("not enough consecutive failures",
			"monitor_id", monitorID,
			"threshold", FailureThreshold,
		)
		return nil
	}

	// We've hit the threshold - create an incident
	incident, err := s.incidentSvc.CreateIncidentIfNeeded(ctx, monitorID)
	if err != nil {
		return fmt.Errorf("create incident: %w", err)
	}

	s.logger.Info("incident created due to consecutive failures",
		"incident_id", incident.ID,
		"monitor_id", monitorID,
		"threshold", FailureThreshold,
	)

	return nil
}

// GetEnabledMonitorsByAgent retrieves all enabled monitors for an agent.
// This is used for task distribution when an agent connects.
func (s *MonitorService) GetEnabledMonitorsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Monitor, error) {
	monitors, err := s.monitorRepo.GetEnabledByAgentID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("monitorService.GetEnabledMonitorsByAgent: %w", err)
	}
	return monitors, nil
}
