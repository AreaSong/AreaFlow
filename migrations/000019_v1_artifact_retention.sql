-- Retention metadata is declarative. This migration does not archive or delete content.

ALTER TABLE artifacts ADD COLUMN IF NOT EXISTS retention_class TEXT NOT NULL DEFAULT 'standard';
ALTER TABLE artifacts ADD COLUMN IF NOT EXISTS retention_until TIMESTAMPTZ;
ALTER TABLE artifacts ADD COLUMN IF NOT EXISTS legal_hold BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE artifacts ADD COLUMN IF NOT EXISTS archive_status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE artifacts ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE artifacts DROP CONSTRAINT IF EXISTS artifacts_retention_class_check;
ALTER TABLE artifacts ADD CONSTRAINT artifacts_retention_class_check
    CHECK (retention_class IN ('ephemeral', 'standard', 'audit', 'legal_hold'));

ALTER TABLE artifacts DROP CONSTRAINT IF EXISTS artifacts_archive_status_check;
ALTER TABLE artifacts ADD CONSTRAINT artifacts_archive_status_check
    CHECK (archive_status IN ('active', 'archive_eligible', 'archived', 'delete_requested'));

CREATE INDEX IF NOT EXISTS artifacts_retention_scan_idx
    ON artifacts (archive_status, retention_until, created_at)
    WHERE legal_hold = false;

CREATE INDEX IF NOT EXISTS artifacts_project_page_idx
    ON artifacts (project_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS events_project_page_idx
    ON events (project_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS audit_events_project_page_idx
    ON audit_events (project_id, created_at DESC, id DESC);
