-- AreaFlow v1 status projection boundary.
-- This table records generated status projections for external project entries,
-- Web, Desktop and compatibility shims without making projection payloads a
-- source of truth.

CREATE TABLE IF NOT EXISTS status_projections (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
    target_kind TEXT NOT NULL,
    target_uri TEXT NOT NULL,
    summary_state TEXT NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_event_id BIGINT REFERENCES events(id) ON DELETE SET NULL,
    source_hash TEXT,
    write_state TEXT NOT NULL DEFAULT 'previewed',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    written_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS status_projections_project_generated_idx
    ON status_projections (project_id, generated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS status_projections_project_target_idx
    ON status_projections (project_id, target_kind, target_uri, generated_at DESC);

CREATE INDEX IF NOT EXISTS status_projections_source_event_idx
    ON status_projections (source_event_id)
    WHERE source_event_id IS NOT NULL;
