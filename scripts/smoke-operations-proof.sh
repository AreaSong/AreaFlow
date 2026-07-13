#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-operations-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_OPERATIONS_PROOF_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-operations-proof.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
real_areamatrix_status="/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
real_areamatrix_readme="/Users/as/Ai-Project/project/AreaMatrix/workflow/README.md"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-operations-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-operations-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-operations-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-operations-proof: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

db_audit_row_counts() {
  psql "${AREAFLOW_DATABASE_URL}" -At <<'SQL'
SELECT
  (SELECT COUNT(*) FROM command_requests)::text || ':' ||
  (SELECT COUNT(*) FROM events)::text || ':' ||
  (SELECT COUNT(*) FROM audit_events)::text;
SQL
}

assert_equal() {
  local got="$1"
  local want="$2"
  local message="$3"

  if [[ "${got}" != "${want}" ]]; then
    echo "smoke-operations-proof: ${message}: got ${got}, want ${want}" >&2
    exit 1
  fi
}

file_fingerprint() {
  local path="$1"

  if [[ ! -e "${path}" ]]; then
    echo "__missing__"
    return
  fi

  local stat_value
  if stat_value="$(stat -f '%m:%z' "${path}" 2>/dev/null)"; then
    :
  else
    stat_value="$(stat -c '%Y:%s' "${path}")"
  fi

  local hash
  hash="$(shasum -a 256 "${path}" | awk '{print $1}')"
  echo "${stat_value}:${hash}"
}

real_status_before="__skipped__"
real_readme_before="__skipped__"
if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_before="$(file_fingerprint "${real_areamatrix_readme}")"
else
  echo "smoke-operations-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${project_root}/workflow" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Operations Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Operations Proof Fixture Workflow
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Operations Proof Fixture
  root: ${project_root}
  kind: product-repo
  adapter: areamatrix
  workflow_profile: areamatrix
  default_branch: main

ownership:
  mode: import
  source_of_truth:
    product_docs: project
    source_code: project
    workflow: project
    execution: project
    status_summary: areaflow
  cutover:
    enabled: false
    new_versions_owned_by: project
    execution_owned_by: project

artifact_store:
  backend: local
  root: ${artifact_root}

permissions:
  capabilities:
    read_project: true
    write_status: false
    write_artifacts: false
    write_workflow: false
    write_generated: false
    write_code: false
    run_commands: false
    manage_workers: false
    manage_git: false
    network: false
    use_secrets: false
    execute_agents: false

  read_paths:
    - docs/**
    - workflow/**

  write_paths: []

  forbidden_paths:
    - workflow/versions/*/execution/**
    - workflow/versions/*/execution/_shared/progress.json
    - .areamatrix/**

commands:
  allowed: []
  forbidden:
    - ./task-loop run
    - git reset --hard
    - git checkout --
    - rm -rf

scheduling:
  priority: 100
  max_parallel_tasks: 1
  agent_role: local_worker
  required_capabilities:
    - read_project
  engine_profile: codex-cli

engines:
  default: codex-cli
  profiles:
    - id: codex-cli
      provider: codex-cli
      secret_ref: none
      enabled: false

status_export:
  enabled: false
  path: .areaflow/status.json
  human_summary:
    enabled: false
    path: workflow/README.md
    block_marker: AREAFLOW_STATUS

migration:
  strategy: import_mirror_shadow_cutover_archive
  phase: import
  imported_versions: []
  immutable_imports: []
EOF

echo "smoke-operations-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-operations-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-operations-proof: operations readiness before proof"
ops_readiness_before_json="$(go run ./cmd/areaflow ops readiness --json)"
assert_contains "${ops_readiness_before_json}" '"status": "needs_attention"'
assert_contains "${ops_readiness_before_json}" '"mode": "read_only_operations_readiness"'
assert_contains "${ops_readiness_before_json}" '"key": "install_migrate_start_register_smoke"'
assert_contains "${ops_readiness_before_json}" '"fresh_local_ops_smoke_missing"'
assert_contains "${ops_readiness_before_json}" '"key": "migration_ledger_readiness"'
assert_contains "${ops_readiness_before_json}" '"full_ledger_table_present": true'
assert_not_contains "${ops_readiness_before_json}" '"full_migration_ledger_missing"'

echo "smoke-operations-proof: completion audit before proof shows E7 missing fresh smoke"
completion_before_counts_before="$(db_audit_row_counts)"
completion_before_json="$(go run ./cmd/areaflow completion audit --json)"
completion_before_counts_after="$(db_audit_row_counts)"
assert_equal "${completion_before_counts_after}" "${completion_before_counts_before}" "completion audit before proof must not write command/event/audit rows"
assert_contains "${completion_before_json}" '"key": "E7_operations_readiness"'
assert_contains "${completion_before_json}" '"fresh_local_ops_smoke_missing"'

echo "smoke-operations-proof: record local ops smoke proof"
ops_smoke_proof_json="$(go run ./cmd/areaflow ops smoke-proof record "${project_key}" \
  --key local_ops_smoke \
  --summary "focused operations proof smoke passed" \
  --evidence-uri "scripts/smoke-operations-proof.sh#operations-proof" \
  --idempotency-key "operations-proof-smoke:${project_key}" \
  --reason "record focused operations proof smoke evidence" \
  --json)"
assert_contains "${ops_smoke_proof_json}" '"proof_key": "local_ops_smoke"'
assert_contains "${ops_smoke_proof_json}" '"status": "recorded"'
assert_contains "${ops_smoke_proof_json}" '"evidence_status": "pass"'
assert_contains "${ops_smoke_proof_json}" '"decision": "allowed"'
assert_contains "${ops_smoke_proof_json}" '"created": true'
assert_contains "${ops_smoke_proof_json}" '"project_write_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"execution_write_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"engine_call_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"service_process_control_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"support_bundle_exported": false'
assert_contains "${ops_smoke_proof_json}" '"migration_apply_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"remote_telemetry_enabled": false'
assert_contains "${ops_smoke_proof_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${ops_smoke_proof_json}" '"record_command_runs_smoke": false'

echo "smoke-operations-proof: operations proof idempotent replay"
ops_smoke_proof_replay_json="$(go run ./cmd/areaflow ops smoke-proof record "${project_key}" \
  --key local_ops_smoke \
  --summary "focused operations proof smoke passed" \
  --evidence-uri "scripts/smoke-operations-proof.sh#operations-proof" \
  --idempotency-key "operations-proof-smoke:${project_key}" \
  --reason "record focused operations proof smoke evidence" \
  --json)"
assert_contains "${ops_smoke_proof_replay_json}" '"created": false'

echo "smoke-operations-proof: operations readiness after proof"
ops_readiness_after_json="$(go run ./cmd/areaflow ops readiness --json)"
assert_contains "${ops_readiness_after_json}" '"status": "ready"'
assert_contains "${ops_readiness_after_json}" '"key": "install_migrate_start_register_smoke"'
assert_contains "${ops_readiness_after_json}" '"evidence_recorded": true'
assert_contains "${ops_readiness_after_json}" '"latest_smoke_proof_key": "local_ops_smoke"'
assert_contains "${ops_readiness_after_json}" '"latest_smoke_proof_fresh": true'
assert_contains "${ops_readiness_after_json}" '"latest_smoke_proof_freshness_status": "fresh"'
assert_contains "${ops_readiness_after_json}" '"smoke_proof_max_age_seconds": 86400'
assert_contains "${ops_readiness_after_json}" '"record_command_runs_smoke": false'
assert_contains "${ops_readiness_after_json}" '"key": "migration_ledger_readiness"'
assert_contains "${ops_readiness_after_json}" '"full_ledger_table_present": true'
assert_not_contains "${ops_readiness_after_json}" '"fresh_local_ops_smoke_missing"'
assert_not_contains "${ops_readiness_after_json}" '"full_migration_ledger_missing"'

echo "smoke-operations-proof: completion audit consumes operations proof"
completion_after_counts_before="$(db_audit_row_counts)"
completion_after_json="$(go run ./cmd/areaflow completion audit --json)"
completion_after_counts_after="$(db_audit_row_counts)"
assert_equal "${completion_after_counts_after}" "${completion_after_counts_before}" "completion audit after proof must not write command/event/audit rows"
assert_contains "${completion_after_json}" '"key": "E7_operations_readiness"'
assert_contains "${completion_after_json}" '"message": "operations readiness evidence is complete for v1.0 scope"'
assert_contains "${completion_after_json}" '"operations_status": "ready"'
assert_contains "${completion_after_json}" '"latest_operations_smoke_proof_fresh": true'
assert_contains "${completion_after_json}" '"latest_operations_smoke_proof_freshness_status": "fresh"'
assert_contains "${completion_after_json}" '"operations_smoke_proof_max_age_seconds": 86400'
assert_contains "${completion_after_json}" '"support_bundle_metadata_only": true'
assert_contains "${completion_after_json}" '"support_bundle_export_open": false'
assert_contains "${completion_after_json}" '"support_bundle_secret_values_included": false'
assert_contains "${completion_after_json}" '"support_bundle_prompt_text_included": false'
assert_contains "${completion_after_json}" '"support_bundle_user_file_contents_included": false'
assert_contains "${completion_after_json}" '"support_bundle_raw_artifact_contents_included": false'
assert_contains "${completion_after_json}" '"support_bundle_unredacted_logs_included": false'
assert_contains "${completion_after_json}" '"support_bundle_sensitive_exclusion_count": 9'
assert_not_contains "${completion_after_json}" '"fresh_local_ops_smoke_missing"'
assert_not_contains "${completion_after_json}" '"full_migration_ledger_missing"'
assert_contains "${completion_after_json}" '"status": "blocked"'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-operations-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-operations-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-operations-proof: pass ${project_key} fixture=${fixture_dir}"
