-- TimescaleDB hypertable for OTLP spans received at /v1/traces.
-- Span-per-row layout to enable per-trace waterfall reconstruction.

CREATE TABLE spans (
    start_time TIMESTAMPTZ NOT NULL,
    trace_id BYTEA NOT NULL,
    span_id BYTEA NOT NULL,
    parent_span_id BYTEA,
    trace_state TEXT,
    flags INT NOT NULL DEFAULT 0,
    name TEXT NOT NULL,
    kind SMALLINT NOT NULL DEFAULT 0,
    service_name TEXT NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration_ns BIGINT NOT NULL,
    status_code SMALLINT NOT NULL DEFAULT 0,
    status_message TEXT,
    attributes JSONB,
    resource JSONB,
    events JSONB,
    dropped_attributes_count INT NOT NULL DEFAULT 0,
    dropped_events_count INT NOT NULL DEFAULT 0,
    dropped_links_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (start_time, trace_id, span_id)
);

SELECT create_hypertable('spans', 'start_time', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_spans_trace_id ON spans(trace_id, start_time);
CREATE INDEX idx_spans_service_start ON spans(service_name, start_time DESC);

ALTER TABLE spans SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'service_name'
);

SELECT add_compression_policy('spans', INTERVAL '1 day');

-- Retention is intentionally NOT set here; the app reads
-- system_settings.trace_retention_days and applies the policy at runtime.

CREATE TABLE system_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO system_settings (key, value) VALUES ('trace_retention_days', '7'::jsonb);
