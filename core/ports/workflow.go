package ports

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// ErrStepAwaiting is returned by a step handler to indicate it has dispatched
// an async operation and is waiting for an external event (e.g. agent response).
// The engine will mark the step as "waiting" and park the workflow until
// ResumeStep is called with the matching correlation key.
var ErrStepAwaiting = errors.New("step awaiting external event")

// WorkflowEngine manages durable workflow execution.
type WorkflowEngine interface {
	// Submit creates a new workflow from a definition and input, returning the workflow ID.
	Submit(ctx context.Context, def WorkflowDefinition, input json.RawMessage) (uuid.UUID, error)

	// Status returns the current state of a workflow.
	Status(ctx context.Context, id uuid.UUID) (*domain.Workflow, error)

	// Cancel marks a workflow as cancelled.
	Cancel(ctx context.Context, id uuid.UUID) error

	// Retry resets a failed workflow for re-execution.
	Retry(ctx context.Context, id uuid.UUID) error

	// List returns workflows filtered by status.
	List(ctx context.Context, status *domain.WorkflowStatus, limit int) ([]*domain.Workflow, error)

	// RegisterHandler registers a step handler by name.
	RegisterHandler(name string, handler StepHandler)

	// ResumeStep completes an awaiting step identified by its correlation key.
	// Pass stepErr != nil to fail the step; otherwise it completes with output.
	ResumeStep(ctx context.Context, correlationKey string, output json.RawMessage, stepErr error) error
}

// StepHandler executes a single step in a workflow.
type StepHandler interface {
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

// StepHandlerFunc is an adapter to allow the use of ordinary functions as StepHandlers.
type StepHandlerFunc func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)

// Execute calls f(ctx, input).
func (f StepHandlerFunc) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return f(ctx, input)
}

// WorkflowDefinition describes a workflow to be submitted.
type WorkflowDefinition struct {
	Name       string
	Timeout    int // seconds, 0 = no timeout
	MaxRetries int
	Steps      []StepDefinition
}

// StepDefinition describes a step within a workflow.
type StepDefinition struct {
	Name           string
	Handler        string
	OnFailure      domain.FailurePolicy
	MaxRetries     int
	CorrelationKey string // Optional: set for steps that will return ErrStepAwaiting
}
