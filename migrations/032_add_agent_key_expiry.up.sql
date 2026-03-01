-- H-023: add optional expiry for agent API keys.
-- NULL means the key never expires (backward-compatible with existing agents).
ALTER TABLE agents ADD COLUMN IF NOT EXISTS api_key_expires_at TIMESTAMPTZ;
