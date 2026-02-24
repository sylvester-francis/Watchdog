package defaults

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sylvester-francis/watchdog/core/domain"
)

func TestWorkflowModule_Name(t *testing.T) {
	m := &workflowModule{}
	assert.Equal(t, "workflow_engine", m.Name())
}

func TestWorkflowModule_GetFailurePolicy(t *testing.T) {
	m := &workflowModule{}

	tests := []struct {
		handler  string
		expected domain.FailurePolicy
	}{
		{"alert.send_discord", domain.FailurePolicySkip},
		{"alert.send_slack", domain.FailurePolicySkip},
		{"alert.send_email", domain.FailurePolicySkip},
		{"alert.send_telegram", domain.FailurePolicySkip},
		{"alert.send_pagerduty", domain.FailurePolicySkip},
		{"alert.send_webhook", domain.FailurePolicySkip},
		{"alert.record_dispatch", domain.FailurePolicySkip},
		{"alert.resolve_channels", domain.FailurePolicyAbort},
		{"some.other.handler", domain.FailurePolicyAbort},
		{"alert.send", domain.FailurePolicyAbort}, // too short for send_ prefix
	}

	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			assert.Equal(t, tt.expected, m.getFailurePolicy(tt.handler))
		})
	}
}

func TestTenantIDFromCtx_Default(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "default", tenantIDFromCtx(ctx))
}

func TestWorkflowStatus_Constants(t *testing.T) {
	assert.Equal(t, domain.WorkflowStatus("pending"), domain.WorkflowStatusPending)
	assert.Equal(t, domain.WorkflowStatus("running"), domain.WorkflowStatusRunning)
	assert.Equal(t, domain.WorkflowStatus("completed"), domain.WorkflowStatusCompleted)
	assert.Equal(t, domain.WorkflowStatus("failed"), domain.WorkflowStatusFailed)
	assert.Equal(t, domain.WorkflowStatus("cancelled"), domain.WorkflowStatusCancelled)
}

func TestStepStatus_Constants(t *testing.T) {
	assert.Equal(t, domain.StepStatus("pending"), domain.StepStatusPending)
	assert.Equal(t, domain.StepStatus("running"), domain.StepStatusRunning)
	assert.Equal(t, domain.StepStatus("completed"), domain.StepStatusCompleted)
	assert.Equal(t, domain.StepStatus("failed"), domain.StepStatusFailed)
	assert.Equal(t, domain.StepStatus("skipped"), domain.StepStatusSkipped)
}

func TestFailurePolicy_Constants(t *testing.T) {
	assert.Equal(t, domain.FailurePolicy("abort"), domain.FailurePolicyAbort)
	assert.Equal(t, domain.FailurePolicy("retry"), domain.FailurePolicyRetry)
	assert.Equal(t, domain.FailurePolicy("skip"), domain.FailurePolicySkip)
}
