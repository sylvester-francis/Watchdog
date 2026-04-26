-- TimescaleDB hypertable for OTLP log records received at /v1/logs.
-- Mirrors the spans layout (PR #126): chunked by timestamp, compressed
-- after 1 day, retention applied at runtime from system_settings.

CREATE TABLE log_records (
    timestamp TIMESTAMPTZ NOT NULL,
    observed_timestamp TIMESTAMPTZ NOT NULL,
    trace_id BYTEA,
    span_id BYTEA,
    trace_flags INT NOT NULL DEFAULT 0,
    severity_number SMALLINT NOT NULL DEFAULT 0,
    severity_text TEXT,
    body TEXT,
    service_name TEXT NOT NULL,
    resource JSONB,
    attributes JSONB,
    dropped_attributes_count INT NOT NULL DEFAULT 0,
    flags INT NOT NULL DEFAULT 0
);

SELECT create_hypertable('log_records', 'timestamp', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_log_records_service_ts ON log_records(service_name, timestamp DESC);
CREATE INDEX idx_log_records_trace_id ON log_records(trace_id, timestamp) WHERE trace_id IS NOT NULL;

ALTER TABLE log_records SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'service_name'
);

SELECT add_compression_policy('log_records', INTERVAL '1 day');

INSERT INTO system_settings (key, value) VALUES ('log_retention_days', '7'::jsonb);
