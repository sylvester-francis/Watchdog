package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// maxWSConnsPerIP limits concurrent WebSocket connections per IP (H-005).
const maxWSConnsPerIP = 10

// WSHandler handles WebSocket connections from agents.
type WSHandler struct {
	agentAuthSvc    ports.AgentAuthService
	monitorSvc      ports.MonitorService
	agentRepo       ports.AgentRepository
	certDetailsRepo ports.CertDetailsRepository
	hub             *realtime.Hub
	logger          *slog.Logger
	allowedOrigins  []string

	// H-005: per-IP concurrent connection tracker.
	connMu    sync.Mutex
	connCount map[string]int
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(
	agentAuthSvc ports.AgentAuthService,
	monitorSvc ports.MonitorService,
	agentRepo ports.AgentRepository,
	certDetailsRepo ports.CertDetailsRepository,
	hub *realtime.Hub,
	logger *slog.Logger,
	allowedOrigins []string,
) *WSHandler {
	return &WSHandler{
		agentAuthSvc:    agentAuthSvc,
		monitorSvc:      monitorSvc,
		agentRepo:       agentRepo,
		certDetailsRepo: certDetailsRepo,
		hub:             hub,
		logger:          logger,
		allowedOrigins:  allowedOrigins,
		connCount:       make(map[string]int),
	}
}

// isBrowserUserAgent returns true if the User-Agent string looks like a
// standard web browser. Browsers always send an Origin header on WebSocket
// upgrades, so a browser UA without Origin indicates a spoofed or
// misconfigured request that should be rejected (H-016).
func isBrowserUserAgent(ua string) bool {
	ua = strings.ToLower(ua)
	browserTokens := []string{"mozilla/", "chrome/", "safari/", "edge/", "opera/", "firefox/"}
	for _, token := range browserTokens {
		if strings.Contains(ua, token) {
			return true
		}
	}
	return false
}

// checkOrigin validates the Origin header on WebSocket upgrade requests.
// Native clients (Go agents) don't send an Origin header, so requests
// without an Origin are allowed — unless the User-Agent indicates a browser.
// Browsers always send Origin on WebSocket upgrades; a missing Origin with
// a browser UA is anomalous and rejected (H-016).
func (h *WSHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// H-016: If the User-Agent looks like a browser, it MUST have an Origin.
		// Only native/programmatic clients (Go agents, curl, scripts) are allowed
		// to connect without Origin.
		ua := r.Header.Get("User-Agent")
		if isBrowserUserAgent(ua) {
			h.logger.Warn("websocket origin missing from browser user-agent",
				slog.String("user_agent", ua),
				slog.String("remote_addr", r.RemoteAddr),
			)
			return false
		}
		return true // Native client (agent) — no Origin expected
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
	clientIP := c.RealIP()

	// H-005: enforce per-IP concurrent connection limit.
	h.connMu.Lock()
	if h.connCount[clientIP] >= maxWSConnsPerIP {
		h.connMu.Unlock()
		h.logger.Warn("websocket connection limit exceeded",
			slog.String("ip", clientIP),
			slog.Int("limit", maxWSConnsPerIP),
		)
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many connections"})
	}
	h.connCount[clientIP]++
	h.connMu.Unlock()

	// Decrement on function exit (connection close).
	defer func() {
		h.connMu.Lock()
		h.connCount[clientIP]--
		if h.connCount[clientIP] <= 0 {
			delete(h.connCount, clientIP)
		}
		h.connMu.Unlock()
	}()

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

	// H-004: Limit message size during auth handshake.
	ws.SetReadLimit(64 * 1024) // 64 KB

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

	// H-023: reject expired agent API keys.
	if agent.IsAPIKeyExpired() {
		h.logger.Warn("agent API key expired",
			slog.String("agent_id", agent.ID.String()),
			slog.String("agent_name", agent.Name),
		)
		h.sendAuthError(ws, "API key expired")
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

		// Upsert extended cert details if agent sent cert metadata
		if h.certDetailsRepo != nil && payload.Metadata["cert_algorithm"] != "" {
			keySize, _ := strconv.Atoi(payload.Metadata["cert_key_size"])
			var sans []string
			if s := payload.Metadata["cert_sans"]; s != "" {
				sans = strings.Split(s, ",")
			}
			cd := &domain.CertDetails{
				MonitorID:    monitorID,
				ExpiryDays:   payload.CertExpiryDays,
				Issuer:       payload.CertIssuer,
				SANs:         sans,
				Algorithm:    payload.Metadata["cert_algorithm"],
				KeySize:      keySize,
				SerialNumber: payload.Metadata["cert_serial"],
				ChainValid:   payload.Metadata["cert_chain_valid"] == "true",
			}
			if err := h.certDetailsRepo.Upsert(ctx, cd); err != nil {
				h.logger.Error("failed to upsert cert details",
					slog.String("monitor_id", payload.MonitorID),
					slog.String("error", err.Error()),
				)
			}
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
		if err := h.monitorSvc.MarkAgentMonitorsDown(ctx, agent.ID); err != nil {
			h.logger.Error("failed to mark monitors down for disconnected agent",
				slog.String("agent_id", agent.ID.String()),
				slog.String("error", err.Error()),
			)
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
