-- Preserve project attribution on append-only history after project deletion.

ALTER TABLE runs ADD COLUMN IF NOT EXISTS project_key_snapshot TEXT;
ALTER TABLE events ADD COLUMN IF NOT EXISTS project_key_snapshot TEXT;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS project_key_snapshot TEXT;

UPDATE runs r
SET project_key_snapshot = p.project_key
FROM projects p
WHERE r.project_id = p.id AND r.project_key_snapshot IS NULL;

UPDATE events e
SET project_key_snapshot = p.project_key
FROM projects p
WHERE e.project_id = p.id AND e.project_key_snapshot IS NULL;

UPDATE audit_events a
SET project_key_snapshot = p.project_key
FROM projects p
WHERE a.project_id = p.id AND a.project_key_snapshot IS NULL;

CREATE OR REPLACE FUNCTION preserve_project_key_snapshot()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.project_key_snapshot IS NULL AND NEW.project_id IS NOT NULL THEN
        SELECT project_key INTO NEW.project_key_snapshot
        FROM projects
        WHERE id = NEW.project_id;
    END IF;
    IF NEW.project_key_snapshot IS NULL AND TG_OP = 'UPDATE' THEN
        NEW.project_key_snapshot := OLD.project_key_snapshot;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS runs_project_key_snapshot_trigger ON runs;
CREATE TRIGGER runs_project_key_snapshot_trigger
BEFORE INSERT OR UPDATE OF project_id ON runs
FOR EACH ROW EXECUTE FUNCTION preserve_project_key_snapshot();

DROP TRIGGER IF EXISTS events_project_key_snapshot_trigger ON events;
CREATE TRIGGER events_project_key_snapshot_trigger
BEFORE INSERT OR UPDATE OF project_id ON events
FOR EACH ROW EXECUTE FUNCTION preserve_project_key_snapshot();

DROP TRIGGER IF EXISTS audit_events_project_key_snapshot_trigger ON audit_events;
CREATE TRIGGER audit_events_project_key_snapshot_trigger
BEFORE INSERT OR UPDATE OF project_id ON audit_events
FOR EACH ROW EXECUTE FUNCTION preserve_project_key_snapshot();

CREATE INDEX IF NOT EXISTS runs_project_key_snapshot_idx
    ON runs (project_key_snapshot, started_at DESC);

CREATE INDEX IF NOT EXISTS events_project_key_snapshot_created_idx
    ON events (project_key_snapshot, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_events_project_key_snapshot_created_idx
    ON audit_events (project_key_snapshot, created_at DESC);
