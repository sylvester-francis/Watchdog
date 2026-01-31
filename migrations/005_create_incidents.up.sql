-- Create incidents table
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    ttr_seconds INT,
    acknowledged_by UUID REFERENCES users(id),
    acknowledged_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for monitor's incidents lookup
CREATE INDEX idx_incidents_monitor_id ON incidents(monitor_id);

-- Index for open incidents (used in dashboard)
CREATE INDEX idx_incidents_open ON incidents(status, created_at DESC) WHERE status = 'open';

-- Index for user acknowledgments
CREATE INDEX idx_incidents_acknowledged_by ON incidents(acknowledged_by) WHERE acknowledged_by IS NOT NULL;

-- Constraint for valid status values
ALTER TABLE incidents ADD CONSTRAINT chk_incident_status
    CHECK (status IN ('open', 'acknowledged', 'resolved'));

-- Constraint: resolved_at required when status is resolved
ALTER TABLE incidents ADD CONSTRAINT chk_resolved_at
    CHECK (
        (status = 'resolved' AND resolved_at IS NOT NULL) OR
        (status != 'resolved')
    );

-- Constraint: acknowledged_by required when status is acknowledged
ALTER TABLE incidents ADD CONSTRAINT chk_acknowledged
    CHECK (
        (status IN ('acknowledged', 'resolved') AND acknowledged_by IS NOT NULL AND acknowledged_at IS NOT NULL) OR
        (status = 'open')
    );
