#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-project-isolation: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

base_database_url="${AREAFLOW_DATABASE_URL}"
schema="areaflow_project_isolation_$$_${RANDOM}"
cleanup() {
  psql "${base_database_url}" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS ${schema} CASCADE" >/dev/null 2>&1 || true
}
trap cleanup EXIT

psql "${base_database_url}" -v ON_ERROR_STOP=1 -c "CREATE SCHEMA ${schema}" >/dev/null
if [[ "${base_database_url}" == *\?* ]]; then
  export AREAFLOW_DATABASE_URL="${base_database_url}&options=-csearch_path%3D${schema}"
else
  export AREAFLOW_DATABASE_URL="${base_database_url}?options=-csearch_path%3D${schema}"
fi
export AREAFLOW_TEST_ISOLATED_SCHEMA=1

echo "smoke-project-isolation: running PostgreSQL project_key isolation fixture"
go test ./internal/project -run TestStoreProjectKeyIsolationWithPostgres -count=1 -v

echo "smoke-project-isolation: pass"
