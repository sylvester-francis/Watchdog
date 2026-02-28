package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus represents the current state of a workflow.
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// StepStatus represents the current state of a workflow step.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// FailurePolicy defines how a step failure is handled.
type FailurePolicy string

const (
	FailurePolicyAbort FailurePolicy = "abort" // Stop workflow on failure
	FailurePolicyRetry FailurePolicy = "retry" // Retry the step
	FailurePolicySkip  FailurePolicy = "skip"  // Skip and continue to next step
)

// Workflow represents a durable workflow execution.
type Workflow struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Status      WorkflowStatus  `json:"status"`
	CurrentStep int             `json:"current_step"`
	Input       json.RawMessage `json:"input,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	MaxRetries  int             `json:"max_retries"`
	RetryCount  int             `json:"retry_count"`
	TimeoutAt   *time.Time      `json:"timeout_at,omitempty"`
	LockedBy    string          `json:"locked_by,omitempty"`
	LockedAt    *time.Time      `json:"locked_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// WorkflowStep represents a single step in a workflow.
type WorkflowStep struct {
	ID          uuid.UUID       `json:"id"`
	WorkflowID  uuid.UUID       `json:"workflow_id"`
	StepIndex   int             `json:"step_index"`
	Name        string          `json:"name"`
	Handler     string          `json:"handler"`
	Status      StepStatus      `json:"status"`
	Input       json.RawMessage `json:"input,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	DurationMs  *int            `json:"duration_ms,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
