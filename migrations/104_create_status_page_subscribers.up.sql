CREATE TABLE IF NOT EXISTS status_page_subscribers (
    id                        UUID         PRIMARY KEY,
    status_page_id            UUID         NOT NULL REFERENCES status_pages(id) ON DELETE CASCADE,
    email                     TEXT         NOT NULL,
    token_hash                VARCHAR(64)  NOT NULL UNIQUE,
    confirmed_at              TIMESTAMPTZ,
    unsubscribed_at           TIMESTAMPTZ,
    last_confirmation_sent_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (status_page_id, email)
);

CREATE INDEX IF NOT EXISTS status_page_subscribers_active_idx
    ON status_page_subscribers (status_page_id)
    WHERE confirmed_at IS NOT NULL AND unsubscribed_at IS NULL;
