-- AreaFlow v0.3 workflow item trace links.
-- Links model semantic relationships between workflow items for Web trace,
-- projection and closeout without hiding long-term trace state in item metadata.

CREATE TABLE IF NOT EXISTS workflow_item_links (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    workflow_version_id BIGINT NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    from_item_id BIGINT NOT NULL REFERENCES workflow_items(id) ON DELETE CASCADE,
    to_item_id BIGINT NOT NULL REFERENCES workflow_items(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS workflow_item_links_unique_idx
    ON workflow_item_links (project_id, workflow_version_id, from_item_id, to_item_id, relation_type);

CREATE INDEX IF NOT EXISTS workflow_item_links_version_relation_idx
    ON workflow_item_links (workflow_version_id, relation_type, created_at DESC);
