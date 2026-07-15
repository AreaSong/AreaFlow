#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-production-capacity: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi
if ! command -v psql >/dev/null 2>&1; then
  echo "smoke-production-capacity: skipped; psql is not installed"
  exit 0
fi

fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-capacity.XXXXXX")"
schema="areaflow_capacity_$$_${RANDOM}"
binary="${fixture_dir}/areaflow"
server_pid=""

cleanup() {
  [[ -z "${server_pid}" ]] || kill "${server_pid}" >/dev/null 2>&1 || true
  psql "${BASE_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS ${schema} CASCADE" >/dev/null 2>&1 || true
  rm -rf "${fixture_dir}"
}
BASE_DATABASE_URL="${AREAFLOW_DATABASE_URL}"
trap cleanup EXIT

psql "${BASE_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "CREATE SCHEMA ${schema}" >/dev/null
if [[ "${BASE_DATABASE_URL}" == *\?* ]]; then
  export AREAFLOW_DATABASE_URL="${BASE_DATABASE_URL}&options=-csearch_path%3D${schema}"
else
  export AREAFLOW_DATABASE_URL="${BASE_DATABASE_URL}?options=-csearch_path%3D${schema}"
fi

go build -o "${binary}" ./cmd/areaflow
"${binary}" migrate up >/dev/null
project_id="$(psql "${AREAFLOW_DATABASE_URL}" -Atq -v ON_ERROR_STOP=1 -c "
INSERT INTO projects (project_key, name, kind, adapter, workflow_profile, default_branch)
VALUES ('capacity-fixture', 'capacity-fixture', 'fixture', 'fixture', 'areamatrix', 'main') RETURNING id")"
psql "${AREAFLOW_DATABASE_URL}" -v ON_ERROR_STOP=1 <<SQL >/dev/null
INSERT INTO project_connections (project_id, connection_type, root_path, current_branch)
VALUES (${project_id}, 'local_path', '${fixture_dir}/project', 'main'),
       (${project_id}, 'artifact_store', '${fixture_dir}/artifacts', 'local');

INSERT INTO events (project_id, event_type, severity, message, metadata, created_at)
SELECT ${project_id}, 'capacity.event', 'info', 'capacity fixture', jsonb_build_object('sequence', value),
       now() - ((1000000 - value) * interval '1 millisecond')
FROM generate_series(1, 1000000) AS value;

INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata, created_at)
SELECT ${project_id}, 'capacity.audit', 'read', 'fixture', value::text, 'allowed', 'capacity fixture',
       jsonb_build_object('sequence', value), now() - ((1000000 - value) * interval '1 millisecond')
FROM generate_series(1, 1000000) AS value;

INSERT INTO artifacts (project_id, artifact_type, storage_backend, uri, source_path, sha256, size_bytes, content_type, metadata, created_at)
SELECT ${project_id}, 'capacity_metadata', 'external_project', 'capacity/' || value, 'capacity/' || value,
       md5(value::text), 0, 'application/octet-stream', jsonb_build_object('sequence', value),
       now() - ((100000 - value) * interval '1 millisecond')
FROM generate_series(1, 100000) AS value;
ANALYZE events;
ANALYZE audit_events;
ANALYZE artifacts;
SQL

for table_count in "events:1000000" "audit_events:1000000" "artifacts:100000"; do
  table="${table_count%%:*}"
  expected="${table_count##*:}"
  actual="$(psql "${AREAFLOW_DATABASE_URL}" -At -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM ${table} WHERE project_id = ${project_id}")"
  [[ "${actual}" == "${expected}" ]] || { echo "${table} count=${actual}, want ${expected}" >&2; exit 1; }
done

for query in \
  "SELECT id FROM events WHERE project_id = ${project_id} ORDER BY created_at DESC, id DESC LIMIT 50" \
  "SELECT id FROM audit_events WHERE project_id = ${project_id} ORDER BY created_at DESC, id DESC LIMIT 50" \
  "SELECT id FROM artifacts WHERE project_id = ${project_id} ORDER BY created_at DESC, id DESC LIMIT 50"; do
  plan="$(psql "${AREAFLOW_DATABASE_URL}" -At -v ON_ERROR_STOP=1 -c "EXPLAIN ${query}")"
  [[ "${plan}" == *"Index"* ]] || { echo "capacity query did not use an index: ${plan}" >&2; exit 1; }
done

AREAFLOW_ENV=development AREAFLOW_AUTH_MODE=disabled AREAFLOW_HOST=127.0.0.1 AREAFLOW_PORT=3870 \
  AREAFLOW_METRICS_PORT=9120 AREAFLOW_ARTIFACT_ROOT="${fixture_dir}/artifacts" \
  "${binary}" server >"${fixture_dir}/server.log" 2>&1 &
server_pid=$!
deadline=$((SECONDS + 30))
until curl -fsS http://127.0.0.1:3870/api/v1/ready >/dev/null; do
  if (( SECONDS >= deadline )); then cat "${fixture_dir}/server.log" >&2; exit 1; fi
  sleep 1
done

rss_before="$(ps -o rss= -p "${server_pid}" | tr -d ' ')"
AREAFLOW_LOAD_URL=http://127.0.0.1:3870 AREAFLOW_LOAD_WRITE_PROJECT=capacity-fixture node scripts/load-production-baseline.js
curl -fsS "http://127.0.0.1:3870/api/v1/artifacts?project_key=capacity-fixture&limit=50" >/dev/null
curl -fsS "http://127.0.0.1:3870/api/v1/audit-events?project_key=capacity-fixture&limit=50" >/dev/null
curl -fsS "http://127.0.0.1:3870/api/v1/projects/capacity-fixture/events/stream?once=true&limit=50" >/dev/null
curl -fsS http://127.0.0.1:3870/api/v1/ready >/dev/null
rss_after="$(ps -o rss= -p "${server_pid}" | tr -d ' ')"
if (( rss_after - rss_before > 131072 )); then
  echo "server RSS grew more than 128 MiB: before=${rss_before}KiB after=${rss_after}KiB" >&2
  exit 1
fi
if rg -i "pool exhausted|context deadline exceeded|out of memory" "${fixture_dir}/server.log"; then
  echo "capacity smoke detected resource exhaustion" >&2
  exit 1
fi

echo "smoke-production-capacity: ok events=1000000 audit_events=1000000 artifacts=100000"
