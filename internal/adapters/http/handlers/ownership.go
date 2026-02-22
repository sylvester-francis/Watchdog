package handlers

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// verifyMonitorOwnership checks the monitor -> agent -> user ownership chain.
// Returns the monitor if it exists and belongs to the user.
// Returns nil, nil when the resource doesn't exist or doesn't belong to the user (caller should 404).
// Returns nil, err on actual DB errors (caller should 500).
func verifyMonitorOwnership(
	ctx context.Context,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	monitorID uuid.UUID,
	userID uuid.UUID,
) (*domain.Monitor, error) {
	monitor, err := monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	if monitor == nil {
		return nil, nil
	}

	agent, err := agentRepo.GetByID(ctx, monitor.AgentID)
	if err != nil {
		return nil, err
	}
	if agent == nil || agent.UserID != userID {
		return nil, nil
	}

	return monitor, nil
}

// verifyIncidentOwnership checks the incident -> monitor -> agent -> user ownership chain.
// Returns the incident if it exists and belongs to the user.
// Returns nil, nil when the resource doesn't exist or doesn't belong to the user (caller should 404).
// Returns nil, err on actual DB errors (caller should 500).
func verifyIncidentOwnership(
	ctx context.Context,
	incidentSvc ports.IncidentService,
	monitorRepo ports.MonitorRepository,
	agentRepo ports.AgentRepository,
	incidentID uuid.UUID,
	userID uuid.UUID,
) (*domain.Incident, error) {
	incident, err := incidentSvc.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, nil
	}

	monitor, err := verifyMonitorOwnership(ctx, monitorRepo, agentRepo, incident.MonitorID, userID)
	if err != nil {
		return nil, err
	}
	if monitor == nil {
		return nil, nil
	}

	return incident, nil
}
