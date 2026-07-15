#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-upgrade-rollback: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi
if ! command -v psql >/dev/null 2>&1; then
  echo "smoke-upgrade-rollback: skipped; psql is not installed"
  exit 0
fi

fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-upgrade.XXXXXX")"
schema="areaflow_upgrade_$$_${RANDOM}"
new_binary="${fixture_dir}/areaflow-new"
old_binary="${fixture_dir}/areaflow-old"

cleanup() {
  psql "${AREAFLOW_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS ${schema} CASCADE" >/dev/null 2>&1 || true
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

psql "${AREAFLOW_DATABASE_URL}" -v ON_ERROR_STOP=1 -c "CREATE SCHEMA ${schema}" >/dev/null
if [[ "${AREAFLOW_DATABASE_URL}" == *\?* ]]; then
  isolated_url="${AREAFLOW_DATABASE_URL}&options=-csearch_path%3D${schema}"
else
  isolated_url="${AREAFLOW_DATABASE_URL}?options=-csearch_path%3D${schema}"
fi

# The historical runner inspected public catalog tables before resolving unqualified names.
# Bootstrap only the isolated fixture ledger tables so that old code never observes or writes
# the caller's public migration ledger.
psql "${isolated_url}" -v ON_ERROR_STOP=1 <<'SQL' >/dev/null
CREATE TABLE schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sha256 TEXT,
    hash_algorithm TEXT,
    hash_recorded_at TIMESTAMPTZ
);
CREATE TABLE migration_ledger (
    id BIGSERIAL PRIMARY KEY,
    migration_name TEXT NOT NULL,
    phase TEXT NOT NULL CHECK (phase IN ('preflight', 'apply', 'verify', 'remediation')),
    status TEXT NOT NULL CHECK (status IN ('ready', 'pass', 'blocked', 'failed', 'skipped')),
    message TEXT NOT NULL,
    migration_hash TEXT,
    evidence_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    remediation TEXT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (migration_name, phase)
);
SQL

addition_commit="$(git log --diff-filter=A --format=%H -- internal/migrate/migrations/000015_v1_oidc_identity.sql | tail -n 1)"
previous_ref="${AREAFLOW_PREVIOUS_REF:-}"
if [[ -z "${previous_ref}" ]]; then
  if [[ -n "${addition_commit}" ]]; then
    previous_ref="${addition_commit}^"
  else
    previous_ref="HEAD"
  fi
fi

mkdir -p "${fixture_dir}/old"
git archive "${previous_ref}" | tar -x -C "${fixture_dir}/old"
(cd "${fixture_dir}/old" && go build -o "${old_binary}" ./cmd/areaflow)
go build -o "${new_binary}" ./cmd/areaflow

export AREAFLOW_DATABASE_URL="${isolated_url}"
"${old_binary}" migrate up >/dev/null
"${old_binary}" release exception-migration-approve --actor upgrade-smoke --reason "approve isolated v14 compatibility fixture" >/dev/null
"${old_binary}" release exception-migration-apply >/dev/null
old_status_before="$("${old_binary}" migrate status)"
if [[ "${old_status_before}" != *"applied 000014_v1_project_history_attribution.sql checksum=verified"* ]]; then
  echo "smoke-upgrade-rollback: legacy schema did not reach 000014" >&2
  echo "${old_status_before}" >&2
  exit 1
fi

"${new_binary}" migrate up >/dev/null
"${new_binary}" migrate up >/dev/null
new_status="$("${new_binary}" migrate status)"
if [[ "${new_status}" != *"applied 000020_v1_append_only_enforcement.sql checksum=verified"* ]]; then
  echo "smoke-upgrade-rollback: upgraded schema did not reach verified 000020" >&2
  echo "${new_status}" >&2
  exit 1
fi

"${new_binary}" project list --json >/dev/null
"${old_binary}" project list --json >/dev/null
old_status_after="$("${old_binary}" migrate status)"
if [[ "${old_status_after}" != *"applied 000014_v1_project_history_attribution.sql checksum=verified"* ]]; then
  echo "smoke-upgrade-rollback: old binary could not read the expanded schema" >&2
  exit 1
fi

if "${new_binary}" migrate down >/dev/null 2>&1; then
  echo "smoke-upgrade-rollback: destructive down migration unexpectedly succeeded" >&2
  exit 1
fi

echo "smoke-upgrade-rollback: ok previous_ref=${previous_ref} schema=${schema} repair=forward-only"
