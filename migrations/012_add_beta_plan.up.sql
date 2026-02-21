-- Add 'beta' to the allowed plan values
ALTER TABLE users DROP CONSTRAINT chk_users_plan;
ALTER TABLE users ADD CONSTRAINT chk_users_plan CHECK (plan IN ('free', 'pro', 'team', 'beta'));
