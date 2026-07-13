-- AreaFlow v0.3 command request baseline.
-- Command requests provide scoped idempotency for mutating API/CLI commands.

CREATE TABLE IF NOT EXISTS command_requests (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    command_type TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS command_requests_project_key_idx
    ON command_requests (project_id, command_type, idempotency_key);
