#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-auth-postgres: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

AREAFLOW_AUTH_DB_SMOKE=1 go test ./internal/api -run TestOIDCSessionAndRBACPostgresSmoke -count=1
echo "smoke-auth-postgres: ok"
