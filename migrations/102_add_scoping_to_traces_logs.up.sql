-- Migration 102: scope spans and log_records by user_id and tenant_id.
--
-- Hub's OTLP receivers initially persisted to these hypertables without
-- per-user / per-tenant scoping, which would let any tenant on a
-- multi-tenant deployment read every other tenant's traces and logs.
-- This migration adds the scoping columns and indexes that the rest of
-- the schema already uses (see migration 018) so the application-layer
-- WHERE clause has something to filter on.
--
-- Existing rows have no recoverable owner: the receivers wrote them
-- before user_id / tenant_id existed in the schema. Both tables hold
-- short-retention observability data (default 7 days for traces and
-- logs); we drop the unscoped legacy rows rather than backfill them
-- against a sentinel owner that would create a knowable security
-- false-floor. The receivers add the scoping fields in the same
-- release, so post-migration data is correctly tagged from the moment
-- the new binary boots.
--
-- TimescaleDB hypertables don't participate in RLS due to columnstore
-- incompatibility (see migration 045). Tenant isolation on these two
-- tables is enforced by the application layer, mirroring the
-- heartbeats hypertable.

TRUNCATE spans;
TRUNCATE log_records;

ALTER TABLE spans
    ADD COLUMN user_id UUID NOT NULL,
    ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';

ALTER TABLE log_records
    ADD COLUMN user_id UUID NOT NULL,
    ADD COLUMN tenant_id VARCHAR(255) NOT NULL DEFAULT 'default';

CREATE INDEX idx_spans_user_tenant_start
    ON spans (user_id, tenant_id, start_time DESC);

CREATE INDEX idx_log_records_user_tenant_ts
    ON log_records (user_id, tenant_id, timestamp DESC);
