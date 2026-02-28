ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'viewer';

UPDATE users SET role = 'super_admin' WHERE is_admin = true;
