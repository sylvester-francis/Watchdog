package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// SSEHandler handles Server-Sent Events for real-time updates.
type SSEHandler struct {
	hub         *realtime.Hub
	agentRepo   ports.AgentRepository
	incidentSvc ports.IncidentService
}

// NewSSEHandler creates a new SSEHandler.
func NewSSEHandler(
	hub *realtime.Hub,
	agentRepo ports.AgentRepository,
	incidentSvc ports.IncidentService,
) *SSEHandler {
	return &SSEHandler{
		hub:         hub,
		agentRepo:   agentRepo,
		incidentSvc: incidentSvc,
	}
}

// AgentStatusEvent represents an agent status change event.
type AgentStatusEvent struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	LastSeenAt *string `json:"lastSeenAt,omitempty"`
}

// IncidentEvent represents an incident event.
type IncidentEvent struct {
	ID         string  `json:"id"`
	MonitorID  string  `json:"monitorId"`
	Status     string  `json:"status"`
	StartedAt  string  `json:"startedAt"`
	ResolvedAt *string `json:"resolvedAt,omitempty"`
}

// Events handles the SSE connection for real-time updates.
func (h *SSEHandler) Events(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	// Create a channel for events
	events := make(chan string, 100)
	done := make(chan struct{})

	// Start a goroutine to poll for updates
	go h.pollUpdates(c.Request().Context(), userID, events, done)

	// Send initial keepalive
	fmt.Fprintf(c.Response(), ": keepalive\n\n")
	c.Response().Flush()

	// Send events as they come
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			close(done)
			return nil

		case event := <-events:
			fmt.Fprint(c.Response(), event)
			c.Response().Flush()

		case <-ticker.C:
			// Send keepalive comment
			fmt.Fprintf(c.Response(), ": keepalive\n\n")
			c.Response().Flush()
		}
	}
}

// pollUpdates polls for updates and sends them to the events channel.
func (h *SSEHandler) pollUpdates(ctx context.Context, userID uuid.UUID, events chan<- string, done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Track last known state
	lastAgentStates := make(map[string]string)
	lastIncidentCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			// Check agent status changes
			agents, err := h.agentRepo.GetByUserID(ctx, userID)
			if err == nil {
				for _, agent := range agents {
					currentStatus := string(agent.Status)
					if lastStatus, exists := lastAgentStates[agent.ID.String()]; !exists || lastStatus != currentStatus {
						lastAgentStates[agent.ID.String()] = currentStatus

						event := AgentStatusEvent{
							ID:     agent.ID.String(),
							Name:   agent.Name,
							Status: currentStatus,
						}
						if agent.LastSeenAt != nil {
							t := agent.LastSeenAt.Format(time.RFC3339)
							event.LastSeenAt = &t
						}

						data, _ := json.Marshal(event)
						events <- fmt.Sprintf("event: agent-status\ndata: %s\n\n", data)
					}
				}
			}

			// Check for new incidents
			incidents, err := h.incidentSvc.GetActiveIncidents(ctx)
			if err == nil {
				if len(incidents) != lastIncidentCount {
					lastIncidentCount = len(incidents)

					// Send incident update
					data, _ := json.Marshal(map[string]int{"count": len(incidents)})
					events <- fmt.Sprintf("event: incident-count\ndata: %s\n\n", data)
				}
			}
		}
	}
}
