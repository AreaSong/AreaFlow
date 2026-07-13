-- AreaFlow v0.3 approval and transition preview baseline.
-- These records are audit facts only; they do not promote, execute, or write managed project files.

CREATE TABLE IF NOT EXISTS workflow_transition_previews (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    from_stage TEXT NOT NULL,
    to_stage TEXT NOT NULL,
    status TEXT NOT NULL,
    required_gate_name TEXT NOT NULL,
    gate_result_id BIGINT REFERENCES gate_results(id) ON DELETE SET NULL,
    blockers JSONB NOT NULL DEFAULT '[]'::jsonb,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS workflow_transition_previews_version_created_idx
    ON workflow_transition_previews (workflow_version_id, created_at DESC);

CREATE INDEX IF NOT EXISTS workflow_transition_previews_project_status_idx
    ON workflow_transition_previews (project_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS approval_records (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    transition_preview_id BIGINT REFERENCES workflow_transition_previews(id) ON DELETE SET NULL,
    approval_kind TEXT NOT NULL,
    decision TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    actor TEXT NOT NULL,
    reason TEXT NOT NULL,
    risk_level TEXT NOT NULL DEFAULT 'normal',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS approval_records_version_created_idx
    ON approval_records (workflow_version_id, created_at DESC);

CREATE INDEX IF NOT EXISTS approval_records_project_decision_idx
    ON approval_records (project_id, decision, created_at DESC);
