-- Create agents table
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    api_key_encrypted BYTEA NOT NULL,
    last_seen_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for user's agents lookup
CREATE INDEX idx_agents_user_id ON agents(user_id);

-- Index for status filtering
CREATE INDEX idx_agents_status ON agents(status);
