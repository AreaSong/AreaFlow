#!/usr/bin/env bash
set -euo pipefail

compose=(docker compose)
default_database_url="postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable"
database_url="${AREAFLOW_DATABASE_URL:-${default_database_url}}"
timeout_seconds="${AREAFLOW_SMOKE_DOCKER_TIMEOUT:-60}"
smoke_script="${AREAFLOW_SMOKE_SCRIPT:-scripts/smoke-local.sh}"
isolated_database=""

cleanup() {
  if [[ -z "${isolated_database}" || "${AREAFLOW_KEEP_DOCKER_SMOKE_DB:-0}" == "1" ]]; then
    return
  fi
  echo "smoke-docker: dropping isolated PostgreSQL database ${isolated_database}"
  "${compose[@]}" exec -T postgres dropdb -U areaflow --if-exists "${isolated_database}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if [[ ! -f "${smoke_script}" ]]; then
  echo "smoke-docker: smoke script not found: ${smoke_script}" >&2
  exit 1
fi

echo "smoke-docker: starting PostgreSQL"
"${compose[@]}" up -d postgres

echo "smoke-docker: waiting for PostgreSQL readiness"
deadline=$((SECONDS + timeout_seconds))
while true; do
  if "${compose[@]}" exec -T postgres pg_isready -U areaflow -d areaflow >/dev/null 2>&1; then
    break
  fi
  if (( SECONDS >= deadline )); then
    echo "smoke-docker: PostgreSQL did not become ready within ${timeout_seconds}s" >&2
    "${compose[@]}" ps postgres >&2 || true
    exit 1
  fi
  sleep 1
done

if [[ -z "${AREAFLOW_DATABASE_URL:-}" && "${AREAFLOW_SMOKE_DOCKER_ISOLATED_DB:-1}" == "1" ]]; then
  isolated_database="areaflow_smoke_$(date +%Y%m%d%H%M%S)_$$"
  echo "smoke-docker: creating isolated PostgreSQL database ${isolated_database}"
  "${compose[@]}" exec -T postgres createdb -U areaflow "${isolated_database}"
  database_url="postgres://areaflow:areaflow@localhost:54329/${isolated_database}?sslmode=disable"
fi

echo "smoke-docker: running ${smoke_script}"
AREAFLOW_DATABASE_URL="${database_url}" \
  AREAFLOW_SMOKE_DOCKER_ISOLATED_DB_CREATED="${isolated_database:+1}" \
  AREAFLOW_SMOKE_DOCKER_DATABASE="${isolated_database}" \
  bash "${smoke_script}"

echo "smoke-docker: ok"
