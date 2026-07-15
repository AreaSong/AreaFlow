#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-append-only: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

go run ./cmd/areaflow migrate up >/dev/null

project_id="$(psql "${AREAFLOW_DATABASE_URL}" -qAtc "INSERT INTO projects (project_key, name, kind, adapter, workflow_profile) VALUES ('append-only-' || txid_current(), 'Append-only smoke', 'fixture', 'fixture', 'fixture') RETURNING id")"
event_id="$(psql "${AREAFLOW_DATABASE_URL}" -qAtc "INSERT INTO events (project_id, event_type, severity, message) VALUES (${project_id}, 'smoke', 'info', 'immutable event') RETURNING id")"
audit_id="$(psql "${AREAFLOW_DATABASE_URL}" -qAtc "INSERT INTO audit_events (project_id, action, decision, reason) VALUES (${project_id}, 'smoke.append-only', 'allowed', 'immutable audit') RETURNING id")"

assert_rejected() {
  local statement="$1"
  if psql "${AREAFLOW_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "${statement}" >/dev/null 2>&1; then
    echo "smoke-append-only: mutation unexpectedly succeeded: ${statement}" >&2
    exit 1
  fi
}

assert_rejected "UPDATE events SET message = 'changed' WHERE id = ${event_id}"
assert_rejected "DELETE FROM events WHERE id = ${event_id}"
assert_rejected "UPDATE audit_events SET reason = 'changed' WHERE id = ${audit_id}"
assert_rejected "DELETE FROM audit_events WHERE id = ${audit_id}"

psql "${AREAFLOW_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "DELETE FROM projects WHERE id = ${project_id}" >/dev/null
remaining="$(psql "${AREAFLOW_DATABASE_URL}" -Atc "SELECT (SELECT project_id IS NULL FROM events WHERE id = ${event_id}) AND (SELECT project_id IS NULL FROM audit_events WHERE id = ${audit_id})")"
if [[ "${remaining}" != "t" ]]; then
  echo "smoke-append-only: project deletion did not preserve append-only history" >&2
  exit 1
fi

echo "smoke-append-only: ok"
