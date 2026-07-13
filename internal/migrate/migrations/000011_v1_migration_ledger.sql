-- AreaFlow v1 migration ledger.
-- Records applied schema migrations with preflight/apply/verify/remediation evidence metadata.

CREATE TABLE IF NOT EXISTS migration_ledger (
    id BIGSERIAL PRIMARY KEY,
    migration_name TEXT NOT NULL,
    phase TEXT NOT NULL CHECK (phase IN ('preflight', 'apply', 'verify', 'remediation')),
    status TEXT NOT NULL CHECK (status IN ('ready', 'pass', 'blocked', 'failed', 'skipped')),
    message TEXT NOT NULL,
    migration_hash TEXT,
    evidence_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    remediation TEXT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (migration_name, phase)
);

CREATE INDEX IF NOT EXISTS migration_ledger_migration_idx
    ON migration_ledger (migration_name);

CREATE INDEX IF NOT EXISTS migration_ledger_phase_status_idx
    ON migration_ledger (phase, status);

WITH migration_names AS (
    SELECT name AS migration_name
    FROM schema_migrations
    UNION
    SELECT '000011_v1_migration_ledger.sql'
),
ledger_phases AS (
    SELECT *
    FROM (VALUES
        ('preflight', 'ready', 'historical migration is recorded in schema_migrations before ledger bootstrap', 'rerun areaflow migrate up after checking schema_migrations and embedded migration list'),
        ('apply', 'ready', 'historical migration apply is represented by schema_migrations', 'restore from backup or apply a targeted remediation migration after approval'),
        ('verify', 'ready', 'historical migration is verifyable through schema_migrations and embedded migration readiness', 'rerun migration ledger readiness after repairing schema state'),
        ('remediation', 'ready', 'rollback/remediation wording is recorded for migration audit readiness', 'prepare approved forward-only remediation migration; destructive rollback is not opened')
    ) AS phase_data(phase, status, message, remediation)
)
INSERT INTO migration_ledger (migration_name, phase, status, message, evidence_json, remediation)
SELECT
    migration_names.migration_name,
    ledger_phases.phase,
    ledger_phases.status,
    ledger_phases.message,
    jsonb_build_object(
        'bootstrap', true,
        'source', '000011_v1_migration_ledger.sql',
        'schema_migrations_recorded', true
    ),
    ledger_phases.remediation
FROM migration_names
CROSS JOIN ledger_phases
ON CONFLICT (migration_name, phase) DO NOTHING;
