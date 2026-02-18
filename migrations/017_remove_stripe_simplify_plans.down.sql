-- Re-add Stripe column
ALTER TABLE users ADD COLUMN stripe_id VARCHAR(255);
CREATE INDEX idx_users_stripe_id ON users(stripe_id) WHERE stripe_id IS NOT NULL;

-- Restore multi-plan constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_plan;
ALTER TABLE users ADD CONSTRAINT chk_users_plan CHECK (plan IN ('free', 'pro', 'team', 'beta'));
