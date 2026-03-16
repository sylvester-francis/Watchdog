package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
)

// DiscoveryScanInput is the input to the discovery scan workflow.
type DiscoveryScanInput struct {
	ScanID      uuid.UUID `json:"scan_id"`
	AgentID     uuid.UUID `json:"agent_id"`
	UserID      uuid.UUID `json:"user_id"`
	Subnet      string    `json:"subnet"`
	Community   string    `json:"community"`
	SNMPVersion string    `json:"snmp_version"`
}

// DiscoveryScanDef returns the workflow definition for a discovery scan.
func DiscoveryScanDef(scanID uuid.UUID) ports.WorkflowDefinition {
	return ports.WorkflowDefinition{
		Name:       "discovery_scan",
		Timeout:    300, // 5 minutes
		MaxRetries: 1,
		Steps: []ports.StepDefinition{
			{
				Name:           "Dispatch Scan",
				Handler:        "discovery.dispatch",
				OnFailure:      domain.FailurePolicyAbort,
				CorrelationKey: fmt.Sprintf("discovery:%s", scanID),
			},
			{
				Name:       "Process Result",
				Handler:    "discovery.process_result",
				OnFailure:  domain.FailurePolicyRetry,
				MaxRetries: 3,
			},
		},
	}
}

// CorrelationKeyForScan returns the correlation key for a discovery scan workflow.
func CorrelationKeyForScan(scanID uuid.UUID) string {
	return fmt.Sprintf("discovery:%s", scanID)
}

// RegisterDiscoveryHandlers registers discovery scan step handlers with the workflow engine.
func RegisterDiscoveryHandlers(
	engine ports.WorkflowEngine,
	hub *realtime.Hub,
	discoveryRepo ports.DiscoveryRepository,
	logger *slog.Logger,
) {
	engine.RegisterHandler("discovery.dispatch", &discoveryDispatchHandler{
		hub:    hub,
		logger: logger,
	})

	engine.RegisterHandler("discovery.process_result", &discoveryProcessResultHandler{
		discoveryRepo: discoveryRepo,
		logger:        logger,
	})
}

// discoveryDispatchHandler sends the discovery task to the agent via WebSocket
// and returns ErrStepAwaiting to park the workflow until the agent responds.
type discoveryDispatchHandler struct {
	hub    *realtime.Hub
	logger *slog.Logger
}

func (h *discoveryDispatchHandler) Execute(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in DiscoveryScanInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("discovery.dispatch: unmarshal: %w", err)
	}

	if !h.hub.IsConnected(in.AgentID) {
		return nil, fmt.Errorf("discovery.dispatch: agent %s is not connected", in.AgentID)
	}

	taskMsg := protocol.NewDiscoveryTaskMessage(in.ScanID.String(), in.Subnet, in.Community, in.SNMPVersion, 300)
	if !h.hub.SendToAgent(in.AgentID, taskMsg) {
		return nil, fmt.Errorf("discovery.dispatch: failed to send task to agent %s", in.AgentID)
	}

	h.logger.Info("discovery scan dispatched via workflow",
		slog.String("scan_id", in.ScanID.String()),
		slog.String("agent_id", in.AgentID.String()),
		slog.String("subnet", in.Subnet),
	)

	return nil, ports.ErrStepAwaiting
}

// DiscoveryResultOutput wraps the agent's discovery result for the process_result step.
type DiscoveryResultOutput struct {
	DiscoveryScanInput
	Result *protocol.DiscoveryResultPayload `json:"result"`
}

// discoveryProcessResultHandler stores discovered devices and updates the scan record.
type discoveryProcessResultHandler struct {
	discoveryRepo ports.DiscoveryRepository
	logger        *slog.Logger
}

func (h *discoveryProcessResultHandler) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in DiscoveryResultOutput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("discovery.process_result: unmarshal: %w", err)
	}

	if in.Result == nil {
		return nil, fmt.Errorf("discovery.process_result: no result payload")
	}

	h.logger.Info("processing discovery result via workflow",
		slog.String("scan_id", in.ScanID.String()),
		slog.String("status", in.Result.Status),
		slog.Int("devices", len(in.Result.Devices)),
	)

	return input, nil
}
