-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    public_uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX users_public_uuid_key ON users (public_uuid);
CREATE UNIQUE INDEX users_email_active_key ON users (lower(email)) WHERE deleted_at IS NULL;

-- Deterministic seed: a fixed public_uuid keeps the local admin account
-- reproducible across environments and easy to reference from tests. This
-- account accepts the static development OTP (see internal/auth/otp.go) and
-- must be removed or protected before any untrusted deployment.
INSERT INTO users (public_uuid, email)
VALUES ('00000000-0000-0000-0000-000000000001', 'admin@localhost');

-- +goose Down
DROP TABLE users;
