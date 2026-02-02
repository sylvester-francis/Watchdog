-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    stripe_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for email lookups (login)
CREATE INDEX idx_users_email ON users(email);

-- Index for Stripe customer lookups
CREATE INDEX idx_users_stripe_id ON users(stripe_id) WHERE stripe_id IS NOT NULL;
