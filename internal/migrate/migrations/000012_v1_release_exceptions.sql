-- AreaFlow v1 release exception lifecycle.
-- The migration itself is R4-gated through migration_ledger before apply.

CREATE TABLE IF NOT EXISTS release_exceptions (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    exception_key TEXT NOT NULL,
    source_gate_item TEXT NOT NULL,
    source_decision TEXT NOT NULL,
    acceptance_type TEXT NOT NULL,
    status TEXT NOT NULL,
    owner TEXT NOT NULL,
    reason TEXT NOT NULL,
    required_evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    audit_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    rollback_plan TEXT NOT NULL,
    review_required BOOLEAN NOT NULL DEFAULT true,
    review_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    requested_by TEXT NOT NULL,
    approved_by TEXT,
    revoked_by TEXT,
    created_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    approved_by_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
    decision_reason TEXT,
    audit_event_id BIGINT REFERENCES audit_events(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT release_exceptions_status_check
        CHECK (status IN ('requested', 'approved', 'rejected', 'revoked', 'expired')),
    CONSTRAINT release_exceptions_acceptance_type_check
        CHECK (acceptance_type IN ('metadata_only_history', 'future_only_gap', 'archive_exception')),
    UNIQUE (project_id, exception_key)
);

CREATE INDEX IF NOT EXISTS release_exceptions_project_status_idx
    ON release_exceptions (project_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS release_exceptions_acceptance_type_idx
    ON release_exceptions (acceptance_type, status);
