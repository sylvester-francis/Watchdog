package workflows_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/testutil/mocks"
	"github.com/sylvester-francis/watchdog/internal/workflows"
)

func TestAlertDispatchDef_IncludesAllChannelTypes(t *testing.T) {
	channelTypes := []string{"global", "discord", "slack", "email", "telegram", "pagerduty", "webhook"}
	def := workflows.AlertDispatchDef(channelTypes)

	assert.Equal(t, "alert_dispatch", def.Name)
	assert.Equal(t, 300, def.Timeout)
	// resolve_channels + one per channel type + record_dispatch
	assert.Equal(t, len(channelTypes)+2, len(def.Steps))
	assert.Equal(t, "Resolve Channels", def.Steps[0].Name)
	assert.Equal(t, "alert.resolve_channels", def.Steps[0].Handler)
	assert.Equal(t, "Record Dispatch", def.Steps[len(def.Steps)-1].Name)

	for i, ct := range channelTypes {
		step := def.Steps[i+1]
		assert.Equal(t, "alert.send_"+ct, step.Handler)
		assert.Equal(t, domain.FailurePolicySkip, step.OnFailure)
	}
}

func TestAlertDispatchDef_EmptyChannels(t *testing.T) {
	def := workflows.AlertDispatchDef(nil)
	// resolve_channels + record_dispatch only
	assert.Equal(t, 2, len(def.Steps))
}

func TestResolveChannelsHandler_Success(t *testing.T) {
	incidentID := uuid.New()
	monitorID := uuid.New()
	agentID := uuid.New()
	userID := uuid.New()

	incident := &domain.Incident{ID: incidentID, MonitorID: monitorID}
	monitor := &domain.Monitor{ID: monitorID, AgentID: agentID}
	agent := &domain.Agent{ID: agentID, UserID: userID}
	channels := []*domain.AlertChannel{
		{ID: uuid.New(), Type: domain.AlertChannelDiscord, Enabled: true},
	}

	engine := &mockWorkflowEngine{handlers: make(map[string]ports.StepHandler)}

	workflows.RegisterAlertHandlers(
		engine,
		&mocks.MockNotifier{},
		&mocks.MockNotifierFactory{},
		&mocks.MockAgentRepository{
			GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Agent, error) {
				assert.Equal(t, agentID, id)
				return agent, nil
			},
		},
		&mocks.MockAlertChannelRepository{
			GetEnabledByUserIDFn: func(_ context.Context, id uuid.UUID) ([]*domain.AlertChannel, error) {
				assert.Equal(t, userID, id)
				return channels, nil
			},
		},
		&mocks.MockIncidentRepository{
			GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Incident, error) {
				return incident, nil
			},
		},
		&mocks.MockMonitorRepository{
			GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Monitor, error) {
				return monitor, nil
			},
		},
		slog.Default(),
	)

	handler, ok := engine.handlers["alert.resolve_channels"]
	require.True(t, ok, "resolve_channels handler should be registered")

	input := workflows.AlertDispatchInput{
		IncidentID: incidentID,
		MonitorID:  monitorID,
		AgentID:    agentID,
		Opened:     true,
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	output, err := handler.Execute(context.Background(), inputJSON)
	require.NoError(t, err)
	require.NotNil(t, output)

	// Verify output contains resolved data with channel_ids (not full channel objects)
	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(output, &result))
	assert.Contains(t, string(output), incidentID.String())
	assert.Contains(t, result, "channel_ids", "output should contain channel_ids, not full channel objects")
	assert.NotContains(t, result, "channels", "output must not contain full channel objects with secrets")
}

func TestSendGlobalHandler_Opened(t *testing.T) {
	notified := false
	engine := &mockWorkflowEngine{handlers: make(map[string]ports.StepHandler)}

	workflows.RegisterAlertHandlers(
		engine,
		&mocks.MockNotifier{
			NotifyIncidentOpenedFn: func(_ context.Context, _ *domain.Incident, _ *domain.Monitor) error {
				notified = true
				return nil
			},
		},
		&mocks.MockNotifierFactory{},
		&mocks.MockAgentRepository{},
		&mocks.MockAlertChannelRepository{},
		&mocks.MockIncidentRepository{},
		&mocks.MockMonitorRepository{},
		slog.Default(),
	)

	handler := engine.handlers["alert.send_global"]
	require.NotNil(t, handler)

	payload := map[string]any{
		"incident_id": uuid.New().String(),
		"monitor_id":  uuid.New().String(),
		"opened":      true,
		"incident":    &domain.Incident{ID: uuid.New()},
		"monitor":     &domain.Monitor{ID: uuid.New()},
		"channel_ids": []string{},
	}
	input, _ := json.Marshal(payload)

	_, err := handler.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, notified)
}

func TestRecordDispatchHandler_LogsCompletion(t *testing.T) {
	engine := &mockWorkflowEngine{handlers: make(map[string]ports.StepHandler)}

	workflows.RegisterAlertHandlers(
		engine,
		&mocks.MockNotifier{},
		&mocks.MockNotifierFactory{},
		&mocks.MockAgentRepository{},
		&mocks.MockAlertChannelRepository{},
		&mocks.MockIncidentRepository{},
		&mocks.MockMonitorRepository{},
		slog.Default(),
	)

	handler := engine.handlers["alert.record_dispatch"]
	require.NotNil(t, handler)

	payload := map[string]any{
		"incident_id": uuid.New().String(),
		"monitor_id":  uuid.New().String(),
		"opened":      true,
		"incident":    &domain.Incident{},
		"monitor":     &domain.Monitor{},
		"channel_ids": []string{},
	}
	input, _ := json.Marshal(payload)

	output, err := handler.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.NotNil(t, output)
}

func TestAllHandlersRegistered(t *testing.T) {
	engine := &mockWorkflowEngine{handlers: make(map[string]ports.StepHandler)}

	workflows.RegisterAlertHandlers(
		engine,
		&mocks.MockNotifier{},
		&mocks.MockNotifierFactory{},
		&mocks.MockAgentRepository{},
		&mocks.MockAlertChannelRepository{},
		&mocks.MockIncidentRepository{},
		&mocks.MockMonitorRepository{},
		slog.Default(),
	)

	expected := []string{
		"alert.resolve_channels",
		"alert.send_global",
		"alert.send_discord",
		"alert.send_slack",
		"alert.send_email",
		"alert.send_telegram",
		"alert.send_pagerduty",
		"alert.send_webhook",
		"alert.record_dispatch",
	}

	for _, name := range expected {
		_, ok := engine.handlers[name]
		assert.True(t, ok, "handler %q should be registered", name)
	}
}

// mockWorkflowEngine captures registered handlers for testing.
type mockWorkflowEngine struct {
	handlers   map[string]ports.StepHandler
	submitFn   func(ctx context.Context, def ports.WorkflowDefinition, input json.RawMessage) (uuid.UUID, error)
}

func (m *mockWorkflowEngine) Submit(ctx context.Context, def ports.WorkflowDefinition, input json.RawMessage) (uuid.UUID, error) {
	if m.submitFn != nil {
		return m.submitFn(ctx, def, input)
	}
	return uuid.New(), nil
}

func (m *mockWorkflowEngine) Status(_ context.Context, _ uuid.UUID) (*domain.Workflow, error) {
	return nil, nil
}

func (m *mockWorkflowEngine) Cancel(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockWorkflowEngine) Retry(_ context.Context, _ uuid.UUID) error  { return nil }

func (m *mockWorkflowEngine) List(_ context.Context, _ *domain.WorkflowStatus, _ int) ([]*domain.Workflow, error) {
	return nil, nil
}

func (m *mockWorkflowEngine) RegisterHandler(name string, handler ports.StepHandler) {
	m.handlers[name] = handler
}
