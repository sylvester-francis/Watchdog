package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// WSHandler handles WebSocket connections from agents.
type WSHandler struct {
	agentAuthSvc   ports.AgentAuthService
	monitorSvc     ports.MonitorService
	agentRepo      ports.AgentRepository
	hub            *realtime.Hub
	logger         *slog.Logger
	allowedOrigins []string
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(
	agentAuthSvc ports.AgentAuthService,
	monitorSvc ports.MonitorService,
	agentRepo ports.AgentRepository,
	hub *realtime.Hub,
	logger *slog.Logger,
	allowedOrigins []string,
) *WSHandler {
	return &WSHandler{
		agentAuthSvc:   agentAuthSvc,
		monitorSvc:     monitorSvc,
		agentRepo:      agentRepo,
		hub:            hub,
		logger:         logger,
		allowedOrigins: allowedOrigins,
	}
}

// checkOrigin validates the Origin header on WebSocket upgrade requests.
// Native clients (Go agents) don't send an Origin header, so requests
// without an Origin are allowed. Browser-originated requests must match
// the allowed origins list or the request's own Host header.
func (h *WSHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Native clients (agents) don't send Origin
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Allow if origin host matches the request Host header
	if u.Host == r.Host {
		return true
	}

	// Check against explicit allowed origins
	for _, allowed := range h.allowedOrigins {
		if origin == allowed {
			return true
		}
	}

	h.logger.Warn("websocket origin rejected",
		slog.String("origin", origin),
		slog.String("host", r.Host),
	)
	return false
}

// HandleConnection upgrades to WebSocket, authenticates the agent, and manages the connection.
func (h *WSHandler) HandleConnection(c echo.Context) error {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", slog.String("error", err.Error()))
		return err
	}

	// Authenticate: read first message within 10s
	if err := ws.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		ws.Close()
		return nil
	}

	_, data, err := ws.ReadMessage()
	if err != nil {
		ws.Close()
		return nil
	}

	var msg protocol.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		h.sendAuthError(ws, "invalid message format")
		ws.Close()
		return nil
	}

	if msg.Type != protocol.MsgTypeAuth {
		h.sendAuthError(ws, "expected auth message")
		ws.Close()
		return nil
	}

	var authPayload protocol.AuthPayload
	if err := msg.ParsePayload(&authPayload); err != nil {
		h.sendAuthError(ws, "invalid auth payload")
		ws.Close()
		return nil
	}

	// Validate API key
	agent, err := h.agentAuthSvc.ValidateAPIKey(c.Request().Context(), authPayload.APIKey)
	if err != nil {
		h.sendAuthError(ws, "invalid API key")
		ws.Close()
		return nil
	}

	// Send auth acknowledgment
	ackMsg := protocol.NewAuthAckMessage(agent.ID.String(), agent.Name)
	ackData, _ := json.Marshal(ackMsg)
	if err := ws.WriteMessage(websocket.TextMessage, ackData); err != nil {
		ws.Close()
		return nil
	}

	// Mark agent online
	ctx := context.Background()
	if err := h.agentRepo.UpdateStatus(ctx, agent.ID, domain.AgentStatusOnline); err != nil {
		h.logger.Error("failed to update agent status", slog.String("error", err.Error()))
	}
	if err := h.agentRepo.UpdateLastSeen(ctx, agent.ID, time.Now()); err != nil {
		h.logger.Error("failed to update last seen", slog.String("error", err.Error()))
	}

	// Handle agent fingerprinting
	if len(authPayload.Fingerprint) > 0 {
		if agent.Fingerprint == nil {
			// First connection: store the fingerprint
			if err := h.agentRepo.UpdateFingerprint(ctx, agent.ID, authPayload.Fingerprint); err != nil {
				h.logger.Error("failed to store agent fingerprint", slog.String("error", err.Error()))
			} else {
				h.logger.Info("agent fingerprint stored",
					slog.String("agent_id", agent.ID.String()),
				)
			}
		} else {
			// Subsequent connection: check for changes
			changed := false
			for k, v := range authPayload.Fingerprint {
				if agent.Fingerprint[k] != v {
					changed = true
					break
				}
			}
			if changed {
				h.logger.Warn("agent fingerprint changed",
					slog.String("agent_id", agent.ID.String()),
					slog.String("agent_name", agent.Name),
				)
				// Update to the new fingerprint
				if err := h.agentRepo.UpdateFingerprint(ctx, agent.ID, authPayload.Fingerprint); err != nil {
					h.logger.Error("failed to update agent fingerprint", slog.String("error", err.Error()))
				}
			}
		}
	}

	h.logger.Info("agent authenticated",
		slog.String("agent_id", agent.ID.String()),
		slog.String("agent_name", agent.Name),
	)

	// Create client and register with hub
	client := realtime.NewClient(h.hub, ws, agent.ID, agent.Name, h.logger)

	// Wire heartbeat processing: agent heartbeats -> MonitorService.ProcessHeartbeat
	client.SetHeartbeatCallback(func(agentID uuid.UUID, payload *protocol.HeartbeatPayload) {
		monitorID, err := uuid.Parse(payload.MonitorID)
		if err != nil {
			h.logger.Warn("invalid monitor ID in heartbeat", slog.String("monitor_id", payload.MonitorID))
			return
		}

		var heartbeat *domain.Heartbeat
		status := domain.HeartbeatStatus(payload.Status)
		if status.IsSuccess() {
			heartbeat = domain.NewSuccessHeartbeat(monitorID, agentID, payload.LatencyMs)
			// Don't record latency for non-network checks (system metrics, docker)
			if payload.LatencyMs == 0 {
				heartbeat.LatencyMs = nil
			}
			// Preserve ErrorMessage for system monitors (contains metric reading e.g. "cpu usage 23.5%")
			if payload.ErrorMessage != "" {
				heartbeat.ErrorMessage = &payload.ErrorMessage
			}
		} else {
			heartbeat = domain.NewFailureHeartbeat(monitorID, agentID, status, payload.ErrorMessage)
		}

		// Thread TLS certificate data from agent payload
		heartbeat.CertExpiryDays = payload.CertExpiryDays
		if payload.CertIssuer != "" {
			heartbeat.CertIssuer = &payload.CertIssuer
		}

		if err := h.monitorSvc.ProcessHeartbeat(ctx, heartbeat); err != nil {
			h.logger.Error("failed to process heartbeat",
				slog.String("monitor_id", payload.MonitorID),
				slog.String("error", err.Error()),
			)
		}

		// Update agent last seen
		_ = h.agentRepo.UpdateLastSeen(ctx, agentID, time.Now())
	})

	h.hub.Register(client)
	client.Start()

	// Send monitor tasks to the agent
	h.sendTasks(ctx, client, agent.ID)

	// Block until client disconnects (readPump/writePump handle the lifecycle)
	<-client.CloseCh()

	// Only mark agent offline if this client wasn't replaced by a newer connection.
	// When an agent reconnects, the new handler sets status to "online" and replaces
	// this client in the hub. Without this check, this old handler's cleanup would
	// clobber the new connection's "online" status.
	currentClient, connected := h.hub.GetClient(agent.ID)
	if !connected || currentClient == client {
		if err := h.agentRepo.UpdateStatus(ctx, agent.ID, domain.AgentStatusOffline); err != nil {
			h.logger.Error("failed to mark agent offline", slog.String("error", err.Error()))
		}
		h.logger.Info("agent disconnected",
			slog.String("agent_id", agent.ID.String()),
			slog.String("agent_name", agent.Name),
		)
	} else {
		h.logger.Info("agent connection replaced, skipping offline status",
			slog.String("agent_id", agent.ID.String()),
		)
	}

	return nil
}

// sendTasks sends all enabled monitor tasks to the newly connected agent.
func (h *WSHandler) sendTasks(ctx context.Context, client *realtime.Client, agentID uuid.UUID) {
	monitors, err := h.monitorSvc.GetMonitorsByAgent(ctx, agentID)
	if err != nil {
		h.logger.Error("failed to get monitors for task distribution",
			slog.String("agent_id", agentID.String()),
			slog.String("error", err.Error()),
		)
		return
	}

	for _, monitor := range monitors {
		if !monitor.Enabled {
			continue
		}

		taskMsg := protocol.NewTaskMessageWithMetadata(
			monitor.ID.String(),
			string(monitor.Type),
			monitor.Target,
			monitor.IntervalSeconds,
			monitor.TimeoutSeconds,
			monitor.Metadata,
		)
		client.Send(taskMsg)
	}

	h.logger.Info("tasks distributed",
		slog.String("agent_id", agentID.String()),
		slog.Int("count", len(monitors)),
	)
}

func (h *WSHandler) sendAuthError(ws *websocket.Conn, msg string) {
	errMsg := protocol.NewAuthErrorMessage(msg)
	data, _ := json.Marshal(errMsg)
	//nolint:errcheck // Best-effort error message before closing
	ws.WriteMessage(websocket.TextMessage, data)
}
