-- AreaFlow v0.8 scheduling policy baseline.
-- This schema records project-scoped scheduling inputs. It does not run workers or acquire leases.

CREATE TABLE IF NOT EXISTS project_scheduling_policies (
    project_id BIGINT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 100,
    max_parallel_tasks INTEGER NOT NULL DEFAULT 1,
    agent_role TEXT NOT NULL DEFAULT 'local_worker',
    required_capabilities JSONB NOT NULL DEFAULT '["read_project"]'::jsonb,
    engine_profile TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS project_scheduling_policies_priority_idx
    ON project_scheduling_policies (priority DESC, updated_at DESC);
