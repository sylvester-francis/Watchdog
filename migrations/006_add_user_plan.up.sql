-- Plan tier for billing
ALTER TABLE users ADD COLUMN plan VARCHAR(20) NOT NULL DEFAULT 'free';
ALTER TABLE users ADD CONSTRAINT chk_users_plan CHECK (plan IN ('free', 'pro', 'team'));

-- Admin flag for admin dashboard access
ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT false;

-- Usage events for analytics (limit hits, approaching limits)
CREATE TABLE usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    resource_type VARCHAR(20) NOT NULL,
    current_count INT NOT NULL,
    max_allowed INT NOT NULL,
    plan VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_usage_events_user_id ON usage_events(user_id);
CREATE INDEX idx_usage_events_created_at ON usage_events(created_at);
CREATE INDEX idx_usage_events_event_type ON usage_events(event_type);
