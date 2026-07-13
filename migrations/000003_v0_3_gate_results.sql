-- AreaFlow v0.3 gate result baseline.
-- Gate results make stage readiness explainable and queryable.

CREATE TABLE IF NOT EXISTS gate_results (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE CASCADE,
    workflow_item_id BIGINT REFERENCES workflow_items(id) ON DELETE SET NULL,
    run_id BIGINT REFERENCES runs(id) ON DELETE SET NULL,
    gate_name TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    status TEXT NOT NULL,
    inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_hashes JSONB NOT NULL DEFAULT '{}'::jsonb,
    failures JSONB NOT NULL DEFAULT '[]'::jsonb,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence_artifact_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS gate_results_project_gate_checked_idx
    ON gate_results (project_id, gate_name, checked_at DESC);

CREATE INDEX IF NOT EXISTS gate_results_workflow_version_idx
    ON gate_results (workflow_version_id, gate_name, checked_at DESC);
