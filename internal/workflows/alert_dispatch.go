package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// AlertDispatchInput is the input to the alert dispatch workflow.
type AlertDispatchInput struct {
	IncidentID uuid.UUID `json:"incident_id"`
	MonitorID  uuid.UUID `json:"monitor_id"`
	AgentID    uuid.UUID `json:"agent_id"`
	Opened     bool      `json:"opened"`
}

// AlertDispatchDef returns the workflow definition for alert dispatch.
func AlertDispatchDef(channelTypes []string) ports.WorkflowDefinition {
	steps := []ports.StepDefinition{
		{
			Name:      "Resolve Channels",
			Handler:   "alert.resolve_channels",
			OnFailure: domain.FailurePolicyAbort,
		},
	}

	for _, ct := range channelTypes {
		steps = append(steps, ports.StepDefinition{
			Name:       fmt.Sprintf("Send %s", ct),
			Handler:    fmt.Sprintf("alert.send_%s", ct),
			OnFailure:  domain.FailurePolicySkip,
			MaxRetries: 2,
		})
	}

	steps = append(steps, ports.StepDefinition{
		Name:      "Record Dispatch",
		Handler:   "alert.record_dispatch",
		OnFailure: domain.FailurePolicySkip,
	})

	return ports.WorkflowDefinition{
		Name:       "alert_dispatch",
		Timeout:    300, // 5 minutes
		MaxRetries: 1,
		Steps:      steps,
	}
}

// RegisterAlertHandlers registers all alert dispatch step handlers with the workflow engine.
func RegisterAlertHandlers(
	engine ports.WorkflowEngine,
	notifier ports.Notifier,
	notifierFactory ports.NotifierFactory,
	agentRepo ports.AgentRepository,
	alertChannelRepo ports.AlertChannelRepository,
	incidentRepo ports.IncidentRepository,
	monitorRepo ports.MonitorRepository,
	logger *slog.Logger,
) {
	engine.RegisterHandler("alert.resolve_channels", &resolveChannelsHandler{
		agentRepo:        agentRepo,
		alertChannelRepo: alertChannelRepo,
		incidentRepo:     incidentRepo,
		monitorRepo:      monitorRepo,
		logger:           logger,
	})

	engine.RegisterHandler("alert.send_global", &sendGlobalHandler{
		notifier: notifier,
		logger:   logger,
	})

	engine.RegisterHandler("alert.send_discord", &sendChannelHandler{factory: notifierFactory, alertChannelRepo: alertChannelRepo, channelType: "discord", logger: logger})
	engine.RegisterHandler("alert.send_slack", &sendChannelHandler{factory: notifierFactory, alertChannelRepo: alertChannelRepo, channelType: "slack", logger: logger})
	engine.RegisterHandler("alert.send_email", &sendChannelHandler{factory: notifierFactory, alertChannelRepo: alertChannelRepo, channelType: "email", logger: logger})
	engine.RegisterHandler("alert.send_telegram", &sendChannelHandler{factory: notifierFactory, alertChannelRepo: alertChannelRepo, channelType: "telegram", logger: logger})
	engine.RegisterHandler("alert.send_pagerduty", &sendChannelHandler{factory: notifierFactory, alertChannelRepo: alertChannelRepo, channelType: "pagerduty", logger: logger})
	engine.RegisterHandler("alert.send_webhook", &sendChannelHandler{factory: notifierFactory, alertChannelRepo: alertChannelRepo, channelType: "webhook", logger: logger})

	engine.RegisterHandler("alert.record_dispatch", &recordDispatchHandler{logger: logger})
}

// resolveChannelsPayload is the output of the resolve_channels step.
// Only channel IDs are stored — never full channel objects (which contain
// decrypted secrets). Send handlers re-fetch channels by ID at execution time.
type resolveChannelsPayload struct {
	IncidentID uuid.UUID        `json:"incident_id"`
	MonitorID  uuid.UUID        `json:"monitor_id"`
	Opened     bool             `json:"opened"`
	Incident   *domain.Incident `json:"incident"`
	Monitor    *domain.Monitor  `json:"monitor"`
	ChannelIDs []uuid.UUID      `json:"channel_ids"`
}

// resolveChannelsHandler looks up the incident, monitor, and alert channels.
type resolveChannelsHandler struct {
	agentRepo        ports.AgentRepository
	alertChannelRepo ports.AlertChannelRepository
	incidentRepo     ports.IncidentRepository
	monitorRepo      ports.MonitorRepository
	logger           *slog.Logger
}

func (h *resolveChannelsHandler) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in AlertDispatchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("resolve_channels: unmarshal input: %w", err)
	}

	incident, err := h.incidentRepo.GetByID(ctx, in.IncidentID)
	if err != nil {
		return nil, fmt.Errorf("resolve_channels: get incident: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("resolve_channels: incident %s not found", in.IncidentID)
	}

	monitor, err := h.monitorRepo.GetByID(ctx, in.MonitorID)
	if err != nil {
		return nil, fmt.Errorf("resolve_channels: get monitor: %w", err)
	}
	if monitor == nil {
		return nil, fmt.Errorf("resolve_channels: monitor %s not found", in.MonitorID)
	}

	agent, err := h.agentRepo.GetByID(ctx, in.AgentID)
	if err != nil || agent == nil {
		return nil, fmt.Errorf("resolve_channels: get agent %s: %w", in.AgentID, err)
	}

	channels, err := h.alertChannelRepo.GetEnabledByUserID(ctx, agent.UserID)
	if err != nil {
		return nil, fmt.Errorf("resolve_channels: get channels: %w", err)
	}

	channelIDs := make([]uuid.UUID, len(channels))
	for i, ch := range channels {
		channelIDs[i] = ch.ID
	}

	payload := resolveChannelsPayload{
		IncidentID: in.IncidentID,
		MonitorID:  in.MonitorID,
		Opened:     in.Opened,
		Incident:   incident,
		Monitor:    monitor,
		ChannelIDs: channelIDs,
	}

	return json.Marshal(payload)
}

// sendGlobalHandler sends via the global (env-based) notifier.
type sendGlobalHandler struct {
	notifier ports.Notifier
	logger   *slog.Logger
}

func (h *sendGlobalHandler) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var payload resolveChannelsPayload
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, fmt.Errorf("send_global: unmarshal: %w", err)
	}

	var err error
	if payload.Opened {
		err = h.notifier.NotifyIncidentOpened(ctx, payload.Incident, payload.Monitor)
	} else {
		err = h.notifier.NotifyIncidentResolved(ctx, payload.Incident, payload.Monitor)
	}
	if err != nil {
		h.logger.Error("global notification failed", slog.String("error", err.Error()))
		return input, fmt.Errorf("send_global: %w", err)
	}

	return input, nil // Pass through for next step
}

// sendChannelHandler sends to a specific channel type using the notifier factory.
// It re-fetches channels by ID at execution time to avoid serializing secrets.
type sendChannelHandler struct {
	factory          ports.NotifierFactory
	alertChannelRepo ports.AlertChannelRepository
	channelType      string
	logger           *slog.Logger
}

func (h *sendChannelHandler) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var payload resolveChannelsPayload
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, fmt.Errorf("send_%s: unmarshal: %w", h.channelType, err)
	}

	sent := 0
	for _, chID := range payload.ChannelIDs {
		ch, err := h.alertChannelRepo.GetByID(ctx, chID)
		if err != nil || ch == nil {
			h.logger.Error("failed to fetch channel",
				slog.String("channel_type", h.channelType),
				slog.String("channel_id", chID.String()),
			)
			continue
		}
		if string(ch.Type) != h.channelType {
			continue
		}

		notifier, err := h.factory.BuildFromChannel(ch)
		if err != nil {
			h.logger.Error("failed to build notifier",
				slog.String("channel_type", h.channelType),
				slog.String("channel_id", chID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		var notifyErr error
		if payload.Opened {
			notifyErr = notifier.NotifyIncidentOpened(ctx, payload.Incident, payload.Monitor)
		} else {
			notifyErr = notifier.NotifyIncidentResolved(ctx, payload.Incident, payload.Monitor)
		}
		if notifyErr != nil {
			h.logger.Error("channel notification failed",
				slog.String("channel_type", h.channelType),
				slog.String("channel_id", chID.String()),
				slog.String("error", notifyErr.Error()),
			)
			continue
		}
		sent++
	}

	return input, nil // Pass through
}

// recordDispatchHandler records that dispatch was completed (logging only for now).
type recordDispatchHandler struct {
	logger *slog.Logger
}

func (h *recordDispatchHandler) Execute(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	var payload resolveChannelsPayload
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, fmt.Errorf("record_dispatch: unmarshal: %w", err)
	}

	h.logger.Info("alert dispatch completed",
		slog.String("incident_id", payload.IncidentID.String()),
		slog.String("monitor_id", payload.MonitorID.String()),
		slog.Bool("opened", payload.Opened),
		slog.Int("channels", len(payload.ChannelIDs)),
	)

	return input, nil
}
