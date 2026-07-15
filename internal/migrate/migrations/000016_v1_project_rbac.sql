-- Project-scoped RBAC for OIDC users and teams.

CREATE TABLE IF NOT EXISTS project_role_bindings (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN (
        'platform_admin',
        'project_admin',
        'operator',
        'approver',
        'auditor',
        'viewer'
    )),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked', 'expired')),
    reason TEXT NOT NULL,
    assigned_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((user_id IS NOT NULL)::integer + (team_id IS NOT NULL)::integer = 1),
    CHECK ((role = 'platform_admin' AND project_id IS NULL) OR (role <> 'platform_admin' AND project_id IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS project_role_bindings_user_active_idx
    ON project_role_bindings (COALESCE(project_id, 0), user_id, role)
    WHERE user_id IS NOT NULL AND status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS project_role_bindings_team_active_idx
    ON project_role_bindings (COALESCE(project_id, 0), team_id, role)
    WHERE team_id IS NOT NULL AND status = 'active';

CREATE INDEX IF NOT EXISTS project_role_bindings_project_status_idx
    ON project_role_bindings (project_id, status, role);
