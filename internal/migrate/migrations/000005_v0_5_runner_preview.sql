-- AreaFlow v0.5 runner preview baseline.
-- This schema records dry-run execution planning only; it does not execute agents or write managed project files.

ALTER TABLE runs
    ADD COLUMN IF NOT EXISTS workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS run_kind TEXT,
    ADD COLUMN IF NOT EXISTS risk_level TEXT NOT NULL DEFAULT 'low',
    ADD COLUMN IF NOT EXISTS risk_policy TEXT NOT NULL DEFAULT 'pause',
    ADD COLUMN IF NOT EXISTS dry_run BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE runs
SET run_kind = run_type
WHERE run_kind IS NULL;

CREATE INDEX IF NOT EXISTS runs_project_status_idx
    ON runs (project_id, status, started_at DESC);

CREATE INDEX IF NOT EXISTS runs_workflow_version_idx
    ON runs (workflow_version_id, started_at DESC)
    WHERE workflow_version_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS run_tasks (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
    workflow_item_id BIGINT REFERENCES workflow_items(id) ON DELETE SET NULL,
    run_id BIGINT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    task_key TEXT NOT NULL,
    task_kind TEXT NOT NULL,
    status TEXT NOT NULL,
    risk_level TEXT NOT NULL DEFAULT 'low',
    sequence INTEGER NOT NULL DEFAULT 0,
    copy_ready_artifact_id BIGINT REFERENCES artifacts(id) ON DELETE SET NULL,
    verify_ready_artifact_id BIGINT REFERENCES artifacts(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS run_tasks_run_key_idx
    ON run_tasks (run_id, task_key);

CREATE INDEX IF NOT EXISTS run_tasks_project_status_idx
    ON run_tasks (project_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS run_attempts (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
    workflow_item_id BIGINT REFERENCES workflow_items(id) ON DELETE SET NULL,
    run_id BIGINT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    run_task_id BIGINT REFERENCES run_tasks(id) ON DELETE SET NULL,
    attempt_kind TEXT NOT NULL,
    status TEXT NOT NULL,
    dry_run BOOLEAN NOT NULL DEFAULT true,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS run_attempts_run_kind_idx
    ON run_attempts (run_id, attempt_kind, started_at DESC);

CREATE INDEX IF NOT EXISTS run_attempts_project_status_idx
    ON run_attempts (project_id, status, started_at DESC);
