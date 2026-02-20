ALTER TABLE agents ADD COLUMN fingerprint JSONB;
ALTER TABLE agents ADD COLUMN fingerprint_verified_at TIMESTAMPTZ;
