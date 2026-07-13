#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-task-matrix-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_TASK_MATRIX_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-task-matrix.XXXXXX")"
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
    echo "smoke-task-matrix-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-task-matrix-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-task-matrix-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-task-matrix-proof: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_fails_contains() {
  local pattern="$1"
  shift
  local output

  set +e
  output="$("$@" 2>&1)"
  local rc=$?
  set -e
  if [[ ${rc} -eq 0 ]]; then
    echo "smoke-task-matrix-proof: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
}

task_matrix_source_set_hash() {
  local backlog_hash="$1"
  local status_audit_hash="$2"

  python3 - "${backlog_hash}" "${status_audit_hash}" <<'PY'
import hashlib
import json
import sys

payload = {
    "required_count_contract": "planned/missing_evidence/blocked_v1_required_counts_must_be_zero",
    "source_paths": [
        "docs/history/v1.0/plans/task-backlog.md",
        "docs/history/v1.0/evidence/task-backlog-status-audit.md",
    ],
    "task_backlog_hash": sys.argv[1],
    "task_status_audit_hash": sys.argv[2],
}
print(hashlib.sha256(json.dumps(payload, sort_keys=True, separators=(",", ":")).encode()).hexdigest())
PY
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
  echo "smoke-task-matrix-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${project_root}/workflow" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Task Matrix Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Task Matrix Proof Fixture Workflow
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Task Matrix Fixture
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

task_matrix_facts=(
  all_v0_v1_tasks_have_status_evidence_and_boundary
  no_planned_v1_required_task_hidden
  preview_only_items_have_evidence_or_explicit_boundary
  implemented_scoped_items_have_scope_labels
  nearest_open_task_has_next_command_and_required_evidence
  v1x_deferred_tasks_have_contracts
)

task_matrix_fact_args=()
for fact in "${task_matrix_facts[@]}"; do
  task_matrix_fact_args+=(--fact "${fact}")
done
task_backlog_hash="$(shasum -a 256 docs/history/v1.0/plans/task-backlog.md | awk '{print $1}')"
task_status_audit_hash="$(shasum -a 256 docs/history/v1.0/evidence/task-backlog-status-audit.md | awk '{print $1}')"
task_matrix_source_set_hash_value="$(task_matrix_source_set_hash "${task_backlog_hash}" "${task_status_audit_hash}")"
task_matrix_binding_args=(
  --source-set-hash "${task_matrix_source_set_hash_value}"
  --backlog-hash "${task_backlog_hash}"
  --task-status-audit-hash "${task_status_audit_hash}"
  --planned-v1-required-task-count 0
  --missing-evidence-v1-required-task-count 0
  --blocked-v1-required-task-count 0
)

echo "smoke-task-matrix-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-task-matrix-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-task-matrix-proof: task matrix proof rejects incomplete complete status"
assert_fails_contains \
  "complete task matrix proof missing required facts" \
  go run ./cmd/areaflow completion task-matrix-proof record "${project_key}" \
    --status complete \
    --fact all_v0_v1_tasks_have_status_evidence_and_boundary \
    --json

echo "smoke-task-matrix-proof: task matrix proof rejects complete status without current binding"
assert_fails_contains \
  "complete task matrix proof missing task matrix binding" \
  go run ./cmd/areaflow completion task-matrix-proof record "${project_key}" \
    --status complete \
    "${task_matrix_fact_args[@]}" \
    --summary "task matrix proof smoke review" \
    --evidence-uri "scripts/smoke-task-matrix-proof.sh#task-matrix" \
    --json

echo "smoke-task-matrix-proof: task matrix proof complete"
proof_json="$(go run ./cmd/areaflow completion task-matrix-proof record "${project_key}" \
  --status complete \
  "${task_matrix_fact_args[@]}" \
  "${task_matrix_binding_args[@]}" \
  --summary "task matrix proof smoke review" \
  --evidence-uri "scripts/smoke-task-matrix-proof.sh#task-matrix" \
  --idempotency-key "task-matrix-proof-smoke:${project_key}" \
  --reason "record task matrix proof smoke evidence" \
  --json)"
assert_contains "${proof_json}" '"proof_status": "complete"'
assert_contains "${proof_json}" '"decision": "allowed"'
assert_contains "${proof_json}" '"missing_facts": []'
assert_contains "${proof_json}" '"created": true'
assert_contains "${proof_json}" '"project_write_attempted": false'
assert_contains "${proof_json}" '"execution_write_attempted": false'
assert_contains "${proof_json}" '"commands_run": false'
assert_contains "${proof_json}" '"docs_written": false'
assert_contains "${proof_json}" '"tasks_written": false'
assert_contains "${proof_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${proof_json}" '"task_matrix_binding_status": "pass"'
assert_contains "${proof_json}" '"task_matrix_binding_blockers": []'
assert_contains "${proof_json}" '"task_matrix_source_set_hash": "'"${task_matrix_source_set_hash_value}"'"'
assert_contains "${proof_json}" '"task_backlog_hash": "'"${task_backlog_hash}"'"'
assert_contains "${proof_json}" '"task_status_audit_hash": "'"${task_status_audit_hash}"'"'
assert_contains "${proof_json}" '"planned_v1_required_task_count": 0'
assert_contains "${proof_json}" '"missing_evidence_v1_required_task_count": 0'
assert_contains "${proof_json}" '"blocked_v1_required_task_count": 0'

echo "smoke-task-matrix-proof: task matrix proof idempotent replay"
replay_json="$(go run ./cmd/areaflow completion task-matrix-proof record "${project_key}" \
  --status complete \
  "${task_matrix_fact_args[@]}" \
  "${task_matrix_binding_args[@]}" \
  --summary "task matrix proof smoke review" \
  --evidence-uri "scripts/smoke-task-matrix-proof.sh#task-matrix" \
  --idempotency-key "task-matrix-proof-smoke:${project_key}" \
  --reason "record task matrix proof smoke evidence" \
  --json)"
assert_contains "${replay_json}" '"created": false'

echo "smoke-task-matrix-proof: completion audit consumes task matrix proof but stays incomplete"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_contains "${completion_json}" '"key": "E2_phase_task_matrix"'
assert_contains "${completion_json}" '"task_matrix_gate_passed": true'
assert_contains "${completion_json}" '"task_matrix_status": "complete"'
assert_contains "${completion_json}" '"latest_task_matrix_proof_evidence_uri": "scripts/smoke-task-matrix-proof.sh#task-matrix"'
assert_contains "${completion_json}" '"task_matrix_binding_status": "pass"'
assert_contains "${completion_json}" '"task_matrix_current_binding_bound": true'
assert_contains "${completion_json}" '"task_matrix_source_set_hash": "'"${task_matrix_source_set_hash_value}"'"'
assert_contains "${completion_json}" '"planned_v1_required_task_count": 0'
assert_contains "${completion_json}" '"missing_evidence_v1_required_task_count": 0'
assert_contains "${completion_json}" '"blocked_v1_required_task_count": 0'
assert_not_contains "${completion_json}" '"task_matrix_proof_missing"'
assert_contains "${completion_json}" '"key": "E4_areamatrix_dogfood_completion"'
assert_contains "${completion_json}" '"status": "blocked"'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-task-matrix-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-task-matrix-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-task-matrix-proof: pass ${project_key} fixture=${fixture_dir}"
