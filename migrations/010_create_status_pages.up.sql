CREATE TABLE IF NOT EXISTS status_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    is_public BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_status_pages_user_id ON status_pages(user_id);
CREATE UNIQUE INDEX idx_status_pages_slug ON status_pages(slug);

CREATE TABLE IF NOT EXISTS status_page_monitors (
    status_page_id UUID NOT NULL REFERENCES status_pages(id) ON DELETE CASCADE,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    sort_order INT DEFAULT 0,
    PRIMARY KEY (status_page_id, monitor_id)
);
