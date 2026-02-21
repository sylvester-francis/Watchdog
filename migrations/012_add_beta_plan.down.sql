-- Revert to original plan constraint (requires no 'beta' rows)
ALTER TABLE users DROP CONSTRAINT chk_users_plan;
ALTER TABLE users ADD CONSTRAINT chk_users_plan CHECK (plan IN ('free', 'pro', 'team'));
