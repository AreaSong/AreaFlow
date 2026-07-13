#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-project-isolation: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

echo "smoke-project-isolation: running PostgreSQL project_key isolation fixture"
go test ./internal/project -run TestStoreProjectKeyIsolationWithPostgres -count=1 -v

echo "smoke-project-isolation: pass"
