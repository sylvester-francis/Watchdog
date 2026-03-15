DROP INDEX IF EXISTS idx_workflow_steps_correlation_waiting;
ALTER TABLE workflow_steps DROP COLUMN IF EXISTS correlation_key;
