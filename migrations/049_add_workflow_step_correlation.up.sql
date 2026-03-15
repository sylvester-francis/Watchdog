ALTER TABLE workflow_steps ADD COLUMN correlation_key VARCHAR(255);

CREATE UNIQUE INDEX idx_workflow_steps_correlation_waiting
  ON workflow_steps (correlation_key)
  WHERE status = 'waiting' AND correlation_key IS NOT NULL;
