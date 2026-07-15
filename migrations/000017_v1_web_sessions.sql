-- Opaque, server-side browser sessions. Raw cookie and CSRF values are never stored.

CREATE TABLE IF NOT EXISTS web_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    session_key TEXT NOT NULL UNIQUE,
    session_hash TEXT NOT NULL,
    csrf_hash TEXT NOT NULL,
    issuer TEXT NOT NULL,
    auth_time TIMESTAMPTZ NOT NULL,
    idle_ttl_seconds BIGINT NOT NULL CHECK (idle_ttl_seconds > 0),
    idle_expires_at TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked', 'expired')),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (idle_expires_at <= absolute_expires_at)
);

CREATE INDEX IF NOT EXISTS web_sessions_user_status_idx
    ON web_sessions (user_id, status, absolute_expires_at);

CREATE INDEX IF NOT EXISTS web_sessions_expiry_idx
    ON web_sessions (status, idle_expires_at, absolute_expires_at);
