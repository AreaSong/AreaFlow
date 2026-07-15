-- Activate provider-neutral OIDC identities without using mutable email as identity.

CREATE TABLE IF NOT EXISTS user_identities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    email_snapshot TEXT,
    claims_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),
    last_authenticated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject)
);

CREATE INDEX IF NOT EXISTS user_identities_user_status_idx
    ON user_identities (user_id, status);

CREATE INDEX IF NOT EXISTS user_identities_last_authenticated_idx
    ON user_identities (last_authenticated_at DESC);
