-- Revert H-022: shrink prefix column back to 11 characters.
-- Warning: any tokens created after the up migration with prefixes > 11 chars
-- will be truncated by this rollback.
ALTER TABLE api_tokens ALTER COLUMN prefix TYPE VARCHAR(11);
