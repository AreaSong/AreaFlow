-- AreaFlow v0.1 core schema.
-- PostgreSQL is the primary state source; artifacts store content externally.

CREATE TABLE IF NOT EXISTS actors (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    display_name TEXT NOT NULL,
    external_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS actors_external_key_idx
    ON actors (external_key)
    WHERE external_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    project_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    adapter TEXT NOT NULL,
    workflow_profile TEXT NOT NULL,
    default_branch TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS project_connections (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    connection_type TEXT NOT NULL,
    root_path TEXT,
    remote_url TEXT,
    current_branch TEXT,
    current_commit TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS project_permissions (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    capability TEXT NOT NULL,
    effect TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    pattern TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS project_permissions_project_idx
    ON project_permissions (project_id, capability, resource_type);

CREATE TABLE IF NOT EXISTS workflow_versions (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    display_label TEXT NOT NULL,
    version_kind TEXT NOT NULL,
    lifecycle_status TEXT NOT NULL,
    source_path TEXT,
    source_hash TEXT,
    import_mode TEXT NOT NULL,
    immutable BOOLEAN NOT NULL DEFAULT false,
    status_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    imported_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS workflow_versions_project_label_idx
    ON workflow_versions (project_id, display_label);

CREATE TABLE IF NOT EXISTS workflow_items (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE CASCADE,
    stage TEXT NOT NULL,
    item_type TEXT NOT NULL,
    external_key TEXT NOT NULL,
    title TEXT,
    status TEXT,
    source_path TEXT,
    source_hash TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    immutable BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    imported_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS workflow_items_external_key_idx
    ON workflow_items (project_id, workflow_version_id, external_key);

CREATE TABLE IF NOT EXISTS residuals (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE CASCADE,
    residual_key TEXT NOT NULL,
    status TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT,
    source_path TEXT,
    current_impact TEXT,
    executable_task BOOLEAN NOT NULL DEFAULT false,
    promotion_required BOOLEAN NOT NULL DEFAULT false,
    close_condition TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    immutable BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    imported_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS residuals_project_key_idx
    ON residuals (project_id, residual_key);

CREATE TABLE IF NOT EXISTS runs (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    run_type TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    created_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS artifacts (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
    run_id BIGINT REFERENCES runs(id) ON DELETE SET NULL,
    workflow_item_id BIGINT REFERENCES workflow_items(id) ON DELETE SET NULL,
    artifact_type TEXT NOT NULL,
    storage_backend TEXT NOT NULL,
    uri TEXT NOT NULL,
    source_path TEXT,
    sha256 TEXT,
    size_bytes BIGINT,
    content_type TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS artifacts_project_source_hash_idx
    ON artifacts (project_id, source_path, sha256);

CREATE TABLE IF NOT EXISTS project_status_snapshots (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    snapshot_kind TEXT NOT NULL,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_hash TEXT,
    export_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    run_id BIGINT REFERENCES runs(id) ON DELETE SET NULL,
    workflow_version_id BIGINT REFERENCES workflow_versions(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS events_project_created_idx
    ON events (project_id, created_at);

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    capability TEXT,
    resource_type TEXT,
    resource TEXT,
    decision TEXT NOT NULL,
    reason TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_events_project_created_idx
    ON audit_events (project_id, created_at);
