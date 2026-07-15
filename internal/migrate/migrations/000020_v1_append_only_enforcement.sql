-- Enforce append-only history at the database boundary.
-- Project deletion may null the foreign key while preserving every other historical field.

CREATE OR REPLACE FUNCTION reject_append_only_history_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE'
       AND OLD.project_id IS NOT NULL
       AND NEW.project_id IS NULL
       AND (to_jsonb(NEW) - 'project_id') = (to_jsonb(OLD) - 'project_id') THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION '% is append-only; % is not allowed', TG_TABLE_NAME, TG_OP
        USING ERRCODE = '55000';
END;
$$;

DROP TRIGGER IF EXISTS events_append_only_trigger ON events;
CREATE TRIGGER events_append_only_trigger
BEFORE UPDATE OR DELETE ON events
FOR EACH ROW EXECUTE FUNCTION reject_append_only_history_mutation();

DROP TRIGGER IF EXISTS audit_events_append_only_trigger ON audit_events;
CREATE TRIGGER audit_events_append_only_trigger
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_append_only_history_mutation();
