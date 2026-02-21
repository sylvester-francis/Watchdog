-- Add username to users (auto-generate from email local part for existing users)
ALTER TABLE users ADD COLUMN username VARCHAR(50);
UPDATE users SET username = LOWER(SPLIT_PART(email, '@', 1));
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT uq_users_username UNIQUE (username);

-- Change status_pages slug from globally unique to user-scoped
DROP INDEX IF EXISTS idx_status_pages_slug;
ALTER TABLE status_pages DROP CONSTRAINT IF EXISTS status_pages_slug_key;
CREATE UNIQUE INDEX idx_status_pages_user_slug ON status_pages(user_id, slug);
