-- Create heartbeats table (TimescaleDB hypertable)
CREATE TABLE heartbeats (
    time TIMESTAMPTZ NOT NULL,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    latency_ms INT,
    error_message TEXT
);

-- Convert to TimescaleDB hypertable for efficient time-series storage
-- Partitions by 1 day chunks
SELECT create_hypertable('heartbeats', 'time', chunk_time_interval => INTERVAL '1 day');

-- Index for monitor heartbeat lookups (most recent first)
CREATE INDEX idx_heartbeats_monitor_time ON heartbeats(monitor_id, time DESC);

-- Index for agent heartbeat lookups
CREATE INDEX idx_heartbeats_agent_time ON heartbeats(agent_id, time DESC);

-- Constraint for valid status values
ALTER TABLE heartbeats ADD CONSTRAINT chk_heartbeat_status
    CHECK (status IN ('up', 'down', 'timeout', 'error'));

-- Enable compression for older chunks (after 7 days)
ALTER TABLE heartbeats SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'monitor_id'
);

-- Add compression policy - compress chunks older than 7 days
SELECT add_compression_policy('heartbeats', INTERVAL '7 days');

-- Add retention policy - drop chunks older than 90 days
SELECT add_retention_policy('heartbeats', INTERVAL '90 days');
