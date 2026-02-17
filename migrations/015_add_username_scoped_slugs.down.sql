-- Restore global slug uniqueness
DROP INDEX IF EXISTS idx_status_pages_user_slug;
CREATE UNIQUE INDEX idx_status_pages_slug ON status_pages(slug);

-- Remove username from users
ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS username;
