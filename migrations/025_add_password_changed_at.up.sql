ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ;
UPDATE users SET password_changed_at = updated_at WHERE password_changed_at IS NULL;
