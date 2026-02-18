-- Remove Stripe integration and simplify to single plan
-- Migrate all users to 'beta' plan
UPDATE users SET plan = 'beta' WHERE plan IN ('free', 'pro', 'team');

-- Update plan constraint to only allow 'beta'
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_plan;
ALTER TABLE users ADD CONSTRAINT chk_users_plan CHECK (plan IN ('beta'));

-- Drop Stripe column and index
DROP INDEX IF EXISTS idx_users_stripe_id;
ALTER TABLE users DROP COLUMN IF EXISTS stripe_id;
