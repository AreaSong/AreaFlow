-- AreaFlow v1 boundary foundation.
-- This migration reserves long-term platform entities without enabling execution,
-- secret resolution, webhooks or remote integrations.

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    email TEXT,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    external_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx
    ON users (lower(email))
    WHERE email IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_actor_idx
    ON users (actor_id)
    WHERE actor_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    team_key TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS teams_key_idx
    ON teams (team_key);

CREATE TABLE IF NOT EXISTS memberships (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS memberships_team_user_idx
    ON memberships (team_id, user_id);

CREATE TABLE IF NOT EXISTS adapters (
    id BIGSERIAL PRIMARY KEY,
    adapter_key TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS adapters_key_idx
    ON adapters (adapter_key);

CREATE TABLE IF NOT EXISTS workflow_profiles (
    id BIGSERIAL PRIMARY KEY,
    profile_key TEXT NOT NULL,
    profile_version TEXT NOT NULL,
    profile_hash TEXT NOT NULL,
    source_path TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS workflow_profiles_key_version_idx
    ON workflow_profiles (profile_key, profile_version);

CREATE TABLE IF NOT EXISTS project_configs (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    protocol_version INTEGER NOT NULL,
    config_path TEXT NOT NULL,
    config_hash TEXT NOT NULL,
    ownership JSONB NOT NULL DEFAULT '{}'::jsonb,
    permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    scheduling JSONB NOT NULL DEFAULT '{}'::jsonb,
    engines JSONB NOT NULL DEFAULT '{}'::jsonb,
    status_export JSONB NOT NULL DEFAULT '{}'::jsonb,
    migration JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT true,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    loaded_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS project_configs_active_project_idx
    ON project_configs (project_id)
    WHERE active;

CREATE INDEX IF NOT EXISTS project_configs_project_loaded_idx
    ON project_configs (project_id, loaded_at DESC);

CREATE TABLE IF NOT EXISTS artifact_locations (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    artifact_id BIGINT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    location_role TEXT NOT NULL DEFAULT 'primary',
    storage_backend TEXT NOT NULL,
    uri TEXT NOT NULL,
    sha256 TEXT,
    size_bytes BIGINT,
    content_type TEXT,
    verified_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS artifact_locations_artifact_role_uri_idx
    ON artifact_locations (artifact_id, location_role, uri);

CREATE INDEX IF NOT EXISTS artifact_locations_project_backend_idx
    ON artifact_locations (project_id, storage_backend, created_at DESC);

CREATE TABLE IF NOT EXISTS artifact_snapshots (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    artifact_id BIGINT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    location_id BIGINT REFERENCES artifact_locations(id) ON DELETE SET NULL,
    snapshot_kind TEXT NOT NULL,
    source_hash TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS artifact_snapshots_artifact_created_idx
    ON artifact_snapshots (artifact_id, created_at DESC);

CREATE TABLE IF NOT EXISTS secret_refs (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
    secret_name TEXT NOT NULL,
    provider TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'project',
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS secret_refs_project_name_idx
    ON secret_refs (project_id, secret_name)
    WHERE project_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS secret_refs_global_name_idx
    ON secret_refs (secret_name)
    WHERE project_id IS NULL;

CREATE TABLE IF NOT EXISTS engine_profiles (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
    profile_key TEXT NOT NULL,
    provider TEXT NOT NULL,
    secret_ref_id BIGINT REFERENCES secret_refs(id) ON DELETE SET NULL,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    resource_limits JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'disabled',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS engine_profiles_project_key_idx
    ON engine_profiles (project_id, profile_key)
    WHERE project_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS engine_profiles_global_key_idx
    ON engine_profiles (profile_key)
    WHERE project_id IS NULL;

CREATE TABLE IF NOT EXISTS api_tokens (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    token_key TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    scope JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS api_tokens_key_idx
    ON api_tokens (token_key);

CREATE INDEX IF NOT EXISTS api_tokens_project_status_idx
    ON api_tokens (project_id, status)
    WHERE project_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS webhooks (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    webhook_key TEXT NOT NULL,
    url TEXT NOT NULL,
    event_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    secret_ref_id BIGINT REFERENCES secret_refs(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'disabled',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS webhooks_project_key_idx
    ON webhooks (project_id, webhook_key);
