-- Revert H-023: remove agent API key expiry column.
ALTER TABLE agents DROP COLUMN IF EXISTS api_key_expires_at;
