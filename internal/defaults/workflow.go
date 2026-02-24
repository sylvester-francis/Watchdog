package defaults

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

const (
	moduleWorkflowEngine = "workflow_engine"
	lockExpiry           = 5 * time.Minute
	pollInterval         = 1 * time.Second
)

var (
	_ registry.Module    = (*workflowModule)(nil)
	_ ports.WorkflowEngine = (*workflowModule)(nil)
)

// workflowModule implements durable workflow execution using PostgreSQL.
type workflowModule struct {
	pool     *pgxpool.Pool
	logger   *slog.Logger
	nodeID   string
	handlers map[string]ports.StepHandler
	mu       sync.RWMutex
	cancel   context.CancelFunc
	done     chan struct{}
}

func newWorkflowModule(pool *pgxpool.Pool, logger *slog.Logger) *workflowModule {
	hostname, _ := os.Hostname()
	return &workflowModule{
		pool:     pool,
		logger:   logger,
		nodeID:   fmt.Sprintf("%s-%d", hostname, os.Getpid()),
		handlers: make(map[string]ports.StepHandler),
		done:     make(chan struct{}),
	}
}

func (m *workflowModule) Name() string { return moduleWorkflowEngine }

func (m *workflowModule) Init(_ context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.pollLoop(ctx)
	return nil
}

func (m *workflowModule) Health(ctx context.Context) error {
	return m.pool.Ping(ctx)
}

func (m *workflowModule) Shutdown(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
		<-m.done
	}
	return nil
}

// RegisterHandler registers a step handler by name.
func (m *workflowModule) RegisterHandler(name string, handler ports.StepHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[name] = handler
}

func (m *workflowModule) getHandler(name string) (ports.StepHandler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.handlers[name]
	return h, ok
}

// Submit creates a new workflow and its steps, returning the workflow ID.
func (m *workflowModule) Submit(ctx context.Context, def ports.WorkflowDefinition, input json.RawMessage) (uuid.UUID, error) {
	wfID := uuid.New()
	now := time.Now()

	var timeoutAt *time.Time
	if def.Timeout > 0 {
		t := now.Add(time.Duration(def.Timeout) * time.Second)
		timeoutAt = &t
	}

	maxRetries := def.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workflow.Submit: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Extract tenant_id from context (same pattern as repository package)
	tenantID := tenantIDFromCtx(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO workflows (id, tenant_id, name, status, current_step, input, max_retries, timeout_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, $5, $6, $7, $8, $8)`,
		wfID, tenantID, def.Name, domain.WorkflowStatusPending, input, maxRetries, timeoutAt, now,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workflow.Submit: insert workflow: %w", err)
	}

	for i, step := range def.Steps {
		stepMaxRetries := step.MaxRetries
		if stepMaxRetries == 0 {
			stepMaxRetries = 3
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO workflow_steps (id, workflow_id, step_index, name, handler, status, max_retries, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
			uuid.New(), wfID, i, step.Name, step.Handler, domain.StepStatusPending, stepMaxRetries, now,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("workflow.Submit: insert step %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("workflow.Submit: commit: %w", err)
	}

	m.logger.Info("workflow submitted",
		slog.String("workflow_id", wfID.String()),
		slog.String("name", def.Name),
		slog.Int("steps", len(def.Steps)),
	)

	return wfID, nil
}

// Status returns the current state of a workflow.
func (m *workflowModule) Status(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	wf, err := m.getWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("workflow.Status: %w", err)
	}
	return wf, nil
}

// Cancel marks a workflow as cancelled.
func (m *workflowModule) Cancel(ctx context.Context, id uuid.UUID) error {
	_, err := m.pool.Exec(ctx, `
		UPDATE workflows SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status IN ($3, $4)`,
		domain.WorkflowStatusCancelled, id, domain.WorkflowStatusPending, domain.WorkflowStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("workflow.Cancel: %w", err)
	}
	return nil
}

// Retry resets a failed workflow for re-execution from its current step.
func (m *workflowModule) Retry(ctx context.Context, id uuid.UUID) error {
	result, err := m.pool.Exec(ctx, `
		UPDATE workflows SET status = $1, error = NULL, retry_count = retry_count + 1, locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE id = $2 AND status = $3`,
		domain.WorkflowStatusPending, id, domain.WorkflowStatusFailed,
	)
	if err != nil {
		return fmt.Errorf("workflow.Retry: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("workflow.Retry: workflow %s not in failed state", id)
	}
	return nil
}

// List returns workflows filtered by status.
func (m *workflowModule) List(ctx context.Context, status *domain.WorkflowStatus, limit int) ([]*domain.Workflow, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows pgx.Rows
	var err error

	if status != nil {
		rows, err = m.pool.Query(ctx, `
			SELECT id, tenant_id, name, status, current_step, input, output, error,
				max_retries, retry_count, timeout_at, locked_by, locked_at, created_at, updated_at
			FROM workflows WHERE status = $1 ORDER BY created_at DESC LIMIT $2`,
			*status, limit,
		)
	} else {
		rows, err = m.pool.Query(ctx, `
			SELECT id, tenant_id, name, status, current_step, input, output, error,
				max_retries, retry_count, timeout_at, locked_by, locked_at, created_at, updated_at
			FROM workflows ORDER BY created_at DESC LIMIT $1`,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("workflow.List: %w", err)
	}
	defer rows.Close()

	var workflows []*domain.Workflow
	for rows.Next() {
		wf := &domain.Workflow{}
		if err := rows.Scan(
			&wf.ID, &wf.TenantID, &wf.Name, &wf.Status, &wf.CurrentStep,
			&wf.Input, &wf.Output, &wf.Error,
			&wf.MaxRetries, &wf.RetryCount, &wf.TimeoutAt,
			&wf.LockedBy, &wf.LockedAt, &wf.CreatedAt, &wf.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("workflow.List: scan: %w", err)
		}
		workflows = append(workflows, wf)
	}
	return workflows, rows.Err()
}

// pollLoop runs the main workflow processing loop.
func (m *workflowModule) pollLoop(ctx context.Context) {
	defer close(m.done)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.recoverStaleLocks(ctx)
			m.checkTimeouts(ctx)
			m.processNext(ctx)
		}
	}
}

// processNext claims and executes the next pending workflow.
func (m *workflowModule) processNext(ctx context.Context) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		m.logger.Error("workflow poll: begin tx", slog.String("error", err.Error()))
		return
	}
	defer tx.Rollback(ctx)

	// Claim a pending workflow using FOR UPDATE SKIP LOCKED
	wf := &domain.Workflow{}
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, name, status, current_step, input, output, error,
			max_retries, retry_count, timeout_at, locked_by, locked_at, created_at, updated_at
		FROM workflows
		WHERE status IN ($1, $2) AND (locked_by IS NULL OR locked_at < $3)
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`,
		domain.WorkflowStatusPending, domain.WorkflowStatusRunning, time.Now().Add(-lockExpiry),
	).Scan(
		&wf.ID, &wf.TenantID, &wf.Name, &wf.Status, &wf.CurrentStep,
		&wf.Input, &wf.Output, &wf.Error,
		&wf.MaxRetries, &wf.RetryCount, &wf.TimeoutAt,
		&wf.LockedBy, &wf.LockedAt, &wf.CreatedAt, &wf.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return // No work to do
		}
		m.logger.Error("workflow poll: claim", slog.String("error", err.Error()))
		return
	}

	// Lock the workflow
	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE workflows SET status = $1, locked_by = $2, locked_at = $3, updated_at = $3
		WHERE id = $4`,
		domain.WorkflowStatusRunning, m.nodeID, now, wf.ID,
	)
	if err != nil {
		m.logger.Error("workflow poll: lock", slog.String("error", err.Error()))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		m.logger.Error("workflow poll: commit lock", slog.String("error", err.Error()))
		return
	}

	// Execute workflow steps outside the claiming transaction
	m.executeWorkflow(ctx, wf)
}

// executeWorkflow runs all remaining steps in a workflow.
func (m *workflowModule) executeWorkflow(ctx context.Context, wf *domain.Workflow) {
	// Get step definitions
	steps, err := m.getSteps(ctx, wf.ID)
	if err != nil {
		m.logger.Error("workflow execute: get steps",
			slog.String("workflow_id", wf.ID.String()),
			slog.String("error", err.Error()),
		)
		m.failWorkflow(ctx, wf.ID, err.Error())
		return
	}

	// Get step definitions from the workflow definition (we need failure policies).
	// Since failure policies aren't stored in DB, we use handler name convention.
	// Steps with OnFailure=Skip will have handler names that are registered as skip-on-fail.

	stepInput := wf.Input
	for i := wf.CurrentStep; i < len(steps); i++ {
		step := steps[i]

		// Update current step
		_, _ = m.pool.Exec(ctx, `
			UPDATE workflows SET current_step = $1, updated_at = NOW() WHERE id = $2`,
			i, wf.ID,
		)

		result, stepErr := m.executeStep(ctx, step, stepInput)

		if stepErr != nil {
			// Determine failure policy from step handler name prefix
			policy := m.getFailurePolicy(step.Handler)

			switch policy {
			case domain.FailurePolicySkip:
				m.logger.Warn("workflow step failed, skipping",
					slog.String("workflow_id", wf.ID.String()),
					slog.String("step", step.Name),
					slog.String("error", stepErr.Error()),
				)
				m.markStepSkipped(ctx, step.ID, stepErr.Error())
				continue
			case domain.FailurePolicyRetry:
				if step.RetryCount < step.MaxRetries {
					m.retryStep(ctx, step.ID)
					// Re-attempt by decrementing i
					steps, _ = m.getSteps(ctx, wf.ID) // refresh
					i--
					continue
				}
				// Exhausted retries, abort
				m.failWorkflow(ctx, wf.ID, fmt.Sprintf("step %q exhausted retries: %s", step.Name, stepErr.Error()))
				return
			default: // abort
				m.failWorkflow(ctx, wf.ID, fmt.Sprintf("step %q failed: %s", step.Name, stepErr.Error()))
				return
			}
		}

		// Pass step output as input to next step
		if result != nil {
			stepInput = result
		}
	}

	// All steps completed
	m.completeWorkflow(ctx, wf.ID, stepInput)
}

// executeStep runs a single step and updates its status in the database.
func (m *workflowModule) executeStep(ctx context.Context, step *domain.WorkflowStep, input json.RawMessage) (json.RawMessage, error) {
	handler, ok := m.getHandler(step.Handler)
	if !ok {
		errMsg := fmt.Sprintf("no handler registered for %q", step.Handler)
		m.markStepFailed(ctx, step.ID, errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Mark step as running
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = $1, input = $2, updated_at = NOW()
		WHERE id = $3`,
		domain.StepStatusRunning, input, step.ID,
	)

	start := time.Now()
	output, err := handler.Execute(ctx, input)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		m.markStepFailed(ctx, step.ID, err.Error())
		_, _ = m.pool.Exec(ctx, `
			UPDATE workflow_steps SET duration_ms = $1, retry_count = retry_count + 1 WHERE id = $2`,
			durationMs, step.ID,
		)
		return nil, err
	}

	// Mark step completed
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = $1, output = $2, duration_ms = $3, updated_at = NOW()
		WHERE id = $4`,
		domain.StepStatusCompleted, output, durationMs, step.ID,
	)

	return output, nil
}

// getFailurePolicy determines the failure policy for a step handler.
// Handlers registered with alert.send_* use Skip policy (don't block other channels).
// All others default to Abort.
func (m *workflowModule) getFailurePolicy(handler string) domain.FailurePolicy {
	// Send handlers skip on failure to not block other channels
	if len(handler) > 11 && handler[:11] == "alert.send_" {
		return domain.FailurePolicySkip
	}
	// Record dispatch is also skip-safe
	if handler == "alert.record_dispatch" {
		return domain.FailurePolicySkip
	}
	return domain.FailurePolicyAbort
}

func (m *workflowModule) markStepFailed(ctx context.Context, stepID uuid.UUID, errMsg string) {
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = $1, error = $2, updated_at = NOW()
		WHERE id = $3`,
		domain.StepStatusFailed, errMsg, stepID,
	)
}

func (m *workflowModule) markStepSkipped(ctx context.Context, stepID uuid.UUID, errMsg string) {
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = $1, error = $2, updated_at = NOW()
		WHERE id = $3`,
		domain.StepStatusSkipped, errMsg, stepID,
	)
}

func (m *workflowModule) retryStep(ctx context.Context, stepID uuid.UUID) {
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = $1, retry_count = retry_count + 1, error = NULL, updated_at = NOW()
		WHERE id = $2`,
		domain.StepStatusPending, stepID,
	)
}

func (m *workflowModule) failWorkflow(ctx context.Context, workflowID uuid.UUID, errMsg string) {
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflows SET status = $1, error = $2, locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE id = $3`,
		domain.WorkflowStatusFailed, errMsg, workflowID,
	)
	m.logger.Error("workflow failed",
		slog.String("workflow_id", workflowID.String()),
		slog.String("error", errMsg),
	)
}

func (m *workflowModule) completeWorkflow(ctx context.Context, workflowID uuid.UUID, output json.RawMessage) {
	_, _ = m.pool.Exec(ctx, `
		UPDATE workflows SET status = $1, output = $2, locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE id = $3`,
		domain.WorkflowStatusCompleted, output, workflowID,
	)
	m.logger.Info("workflow completed", slog.String("workflow_id", workflowID.String()))
}

func (m *workflowModule) getWorkflow(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	wf := &domain.Workflow{}
	err := m.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, status, current_step, input, output, error,
			max_retries, retry_count, timeout_at, locked_by, locked_at, created_at, updated_at
		FROM workflows WHERE id = $1`, id,
	).Scan(
		&wf.ID, &wf.TenantID, &wf.Name, &wf.Status, &wf.CurrentStep,
		&wf.Input, &wf.Output, &wf.Error,
		&wf.MaxRetries, &wf.RetryCount, &wf.TimeoutAt,
		&wf.LockedBy, &wf.LockedAt, &wf.CreatedAt, &wf.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow %s not found", id)
		}
		return nil, err
	}
	return wf, nil
}

func (m *workflowModule) getSteps(ctx context.Context, workflowID uuid.UUID) ([]*domain.WorkflowStep, error) {
	rows, err := m.pool.Query(ctx, `
		SELECT id, workflow_id, step_index, name, handler, status, input, output, error,
			retry_count, max_retries, duration_ms, created_at, updated_at
		FROM workflow_steps WHERE workflow_id = $1 ORDER BY step_index ASC`, workflowID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.WorkflowStep
	for rows.Next() {
		s := &domain.WorkflowStep{}
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.StepIndex, &s.Name, &s.Handler, &s.Status,
			&s.Input, &s.Output, &s.Error,
			&s.RetryCount, &s.MaxRetries, &s.DurationMs, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// recoverStaleLocks releases workflows locked by crashed nodes.
func (m *workflowModule) recoverStaleLocks(ctx context.Context) {
	result, err := m.pool.Exec(ctx, `
		UPDATE workflows SET locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE locked_at IS NOT NULL AND locked_at < $1 AND status = $2`,
		time.Now().Add(-lockExpiry), domain.WorkflowStatusRunning,
	)
	if err != nil {
		m.logger.Error("workflow: recover stale locks", slog.String("error", err.Error()))
		return
	}
	if result.RowsAffected() > 0 {
		m.logger.Warn("workflow: recovered stale locks", slog.Int64("count", result.RowsAffected()))
	}
}

// checkTimeouts marks timed-out workflows as failed.
func (m *workflowModule) checkTimeouts(ctx context.Context) {
	result, err := m.pool.Exec(ctx, `
		UPDATE workflows SET status = $1, error = 'workflow timed out', locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE timeout_at IS NOT NULL AND timeout_at < NOW() AND status IN ($2, $3)`,
		domain.WorkflowStatusFailed, domain.WorkflowStatusPending, domain.WorkflowStatusRunning,
	)
	if err != nil {
		m.logger.Error("workflow: check timeouts", slog.String("error", err.Error()))
		return
	}
	if result.RowsAffected() > 0 {
		m.logger.Warn("workflow: timed out workflows", slog.Int64("count", result.RowsAffected()))
	}
}

// tenantIDFromCtx extracts tenant ID from context (mirrors repository.TenantIDFromContext).
func tenantIDFromCtx(ctx context.Context) string {
	type tenantKey struct{}
	if id, ok := ctx.Value(tenantKey{}).(string); ok && id != "" {
		return id
	}
	return "default"
}
