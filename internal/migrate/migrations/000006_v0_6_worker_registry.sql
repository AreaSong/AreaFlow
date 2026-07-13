-- AreaFlow v0.6 worker registry baseline.
-- This schema records worker identity, heartbeat and scoped lease state. It does not execute tasks.

CREATE TABLE IF NOT EXISTS workers (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    worker_key TEXT NOT NULL,
    worker_type TEXT NOT NULL DEFAULT 'local_host',
    status TEXT NOT NULL DEFAULT 'online',
    hostname TEXT,
    pid INTEGER,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_heartbeat_at TIMESTAMPTZ,
    heartbeat_interval_seconds INTEGER NOT NULL DEFAULT 30,
    lease_timeout_seconds INTEGER NOT NULL DEFAULT 300,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS workers_project_key_idx
    ON workers (project_id, worker_key);

CREATE INDEX IF NOT EXISTS workers_project_status_idx
    ON workers (project_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS worker_heartbeats (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worker_id BIGINT NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS worker_heartbeats_worker_observed_idx
    ON worker_heartbeats (worker_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS leases (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    run_id BIGINT REFERENCES runs(id) ON DELETE CASCADE,
    run_task_id BIGINT REFERENCES run_tasks(id) ON DELETE CASCADE,
    workflow_item_id BIGINT REFERENCES workflow_items(id) ON DELETE SET NULL,
    worker_id BIGINT REFERENCES workers(id) ON DELETE SET NULL,
    lease_kind TEXT NOT NULL,
    status TEXT NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    heartbeat_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    allowed_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    scope JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE UNIQUE INDEX IF NOT EXISTS leases_active_run_task_idx
    ON leases (run_task_id)
    WHERE run_task_id IS NOT NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS leases_project_status_idx
    ON leases (project_id, status, expires_at);

CREATE INDEX IF NOT EXISTS leases_worker_status_idx
    ON leases (worker_id, status, expires_at)
    WHERE worker_id IS NOT NULL;
