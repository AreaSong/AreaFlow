#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-managed-generated-write: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_MANAGED_GENERATED_PROJECT_KEY:-areamatrix-generated-fixture}"
deny_project_key="${AREAFLOW_MANAGED_GENERATED_DENY_PROJECT_KEY:-areamatrix-product-deny}"
workflow_label="${AREAFLOW_MANAGED_GENERATED_WORKFLOW_VERSION:-managed-generated-smoke-$(date +%Y%m%d%H%M%S)}"
deny_workflow_label="${workflow_label}-deny"
worker_key="${workflow_label}-worker"
deny_worker_key="${deny_workflow_label}-worker"
target_path=".areaflow/generated/status.json"
before_content='{"state":"before","source":"managed-generated-smoke"}'
after_content='{"state":"after","source":"managed-generated-smoke"}'
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-managed-generated.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
deny_project_root="${fixture_dir}/areamatrix-product-root"
artifact_root="${fixture_dir}/artifact-store"
deny_artifact_root="${fixture_dir}/deny-artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
deny_config_path="${fixture_dir}/areaflow-deny.yaml"
real_areamatrix_status="/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
real_areamatrix_readme="/Users/as/Ai-Project/project/AreaMatrix/workflow/README.md"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-managed-generated-write: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-managed-generated-write: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-managed-generated-write: expected output to contain pattern: ${pattern}" >&2
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
    echo "smoke-managed-generated-write: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
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

json_run_id() {
  JSON_INPUT="$1" python3 - <<'PY'
import json
import os
print(json.loads(os.environ["JSON_INPUT"])["run"]["id"])
PY
}

create_config() {
  local path="$1"
  local key="$2"
  local name="$3"
  local root="$4"
  local kind="$5"
  local store_root="$6"

  cat >"${path}" <<EOF
version: 1

project:
  id: ${key}
  name: ${name}
  root: ${root}
  kind: ${kind}
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
  root: ${store_root}

permissions:
  capabilities:
    read_project: true
    write_status: true
    write_artifacts: true
    write_workflow: false
    write_generated: true
    write_code: false
    run_commands: false
    manage_workers: false
    manage_git: false
    network: false
    use_secrets: false
    execute_agents: false

  read_paths:
    - docs/**
    - .areaflow/generated/**

  write_paths:
    - .areaflow/status.json
    - .areaflow/generated/**

  forbidden_paths:
    - workflow/versions/*/execution/**
    - workflow/versions/*/execution/_shared/progress.json
    - .areamatrix/**
    - "**/*.sqlite"
    - "**/*.db"

commands:
  allowed:
    - ./dev workflow doctor
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
    - write_artifacts
    - write_generated
  engine_profile: codex-cli

engines:
  default: codex-cli
  profiles:
    - id: codex-cli
      provider: codex-cli
      secret_ref: none
      enabled: false
      resource_limits:
        max_active_leases: 1
        max_queued_tasks: 20

status_export:
  enabled: true
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
}

prepare_project_root() {
  local root="$1"
  mkdir -p "${root}/docs" "${root}/.areaflow/generated"
  cat >"${root}/docs/README.md" <<'EOF'
# Fixture Docs

Minimal source docs coverage for managed generated write smoke.
EOF
  printf "%s" "${before_content}" >"${root}/${target_path}"
}

ready_version() {
  local key="$1"
  local label="$2"
  local reason="$3"

  version_json="$(go run ./cmd/areaflow workflow version create "${key}" "${label}" --json)"
  assert_contains "${version_json}" '"import_mode": "authored"'

  queue_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${key}" "${label}" --stage queue --item-type queue_candidate --reason "${reason} queue" --json)"
  assert_contains "${queue_ready_json}" '"status": "ready"'

  promotion_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${key}" "${label}" --stage promotion_preview --item-type promotion_preview --reason "${reason} promotion" --json)"
  assert_contains "${promotion_ready_json}" '"status": "ready"'

  promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${key}" "${label}" promotion_preview --json)"
  assert_contains "${promotion_gate_json}" '"gate_name": "promotion_preview"'
  assert_contains "${promotion_gate_json}" '"status": "pass"'

  transition_json="$(go run ./cmd/areaflow workflow transition preview "${key}" "${label}" --json)"
  assert_contains "${transition_json}" '"status": "ready"'

  approval_json="$(go run ./cmd/areaflow workflow approval record "${key}" "${label}" --decision approved --reason "${reason} approval" --json)"
  assert_contains "${approval_json}" '"decision": "approved"'
  assert_contains "${approval_json}" '"transition_status": "ready"'

  approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${key}" "${label}" approval_gate --json)"
  assert_contains "${approval_gate_json}" '"gate_name": "approval_gate"'
  assert_contains "${approval_gate_json}" '"status": "pass"'

  live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${key}" "${label}" live_mapping_gate --json)"
  assert_contains "${live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
  assert_contains "${live_mapping_gate_json}" '"status": "pass"'
  assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'
}

real_status_before="__skipped__"
real_readme_before="__skipped__"
if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_before="$(file_fingerprint "${real_areamatrix_readme}")"
else
  echo "smoke-managed-generated-write: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${artifact_root}" "${deny_artifact_root}"
prepare_project_root "${project_root}"
prepare_project_root "${deny_project_root}"
create_config "${config_path}" "${project_key}" "AreaMatrix Managed Generated Fixture" "${project_root}" "temporary-project" "${artifact_root}"
create_config "${deny_config_path}" "${deny_project_key}" "AreaMatrix Product Deny Fixture" "${deny_project_root}" "product-repo" "${deny_artifact_root}"

before_sha="$(shasum -a 256 "${project_root}/${target_path}" | awk '{print $1}')"
before_size="$(wc -c <"${project_root}/${target_path}" | tr -d ' ')"
deny_before_sha="$(shasum -a 256 "${deny_project_root}/${target_path}" | awk '{print $1}')"
deny_before_size="$(wc -c <"${deny_project_root}/${target_path}" | tr -d ' ')"

echo "smoke-managed-generated-write: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-managed-generated-write: project add fixture"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-managed-generated-write: project add non-fixture deny project"
deny_add_output="$(go run ./cmd/areaflow project add --config "${deny_config_path}")"
assert_contains "${deny_add_output}" "registered ${deny_project_key} ${deny_project_root}"

echo "smoke-managed-generated-write: readiness gates"
ready_version "${project_key}" "${workflow_label}" "managed generated smoke"
ready_version "${deny_project_key}" "${deny_workflow_label}" "managed generated deny smoke"

echo "smoke-managed-generated-write: worker register fixture"
worker_json="$(go run ./cmd/areaflow worker register "${project_key}" --worker-key "${worker_key}" --capability read_project --capability write_artifacts --capability write_generated --json)"
assert_contains "${worker_json}" '"worker_key": "'"${worker_key}"'"'
assert_contains "${worker_json}" '"write_generated"'
assert_contains "${worker_json}" '"status": "online"'

echo "smoke-managed-generated-write: queue fixture generated write"
queue_json="$(go run ./cmd/areaflow run managed-generated-write-queue "${project_key}" "${workflow_label}" --target-path "${target_path}" --content "${after_content}" --expected-before-sha256 "${before_sha}" --expected-before-size "${before_size}" --idempotency-key "managed-generated-write-queue-${workflow_label}" --reason "managed generated smoke queue" --json)"
assert_contains "${queue_json}" '"run_type": "managed_generated_write"'
assert_contains "${queue_json}" '"run_kind": "execution"'
assert_contains "${queue_json}" '"status": "queued"'
assert_contains "${queue_json}" '"dry_run": false'
assert_contains "${queue_json}" '"task_kind": "managed_generated_write_task"'
assert_contains "${queue_json}" '"artifact_type": "managed_generated_write_set"'
assert_contains "${queue_json}" '"generated_only": true'
assert_contains "${queue_json}" '"generated_only_apply_open": true'
assert_contains "${queue_json}" '"area_flow_artifact_written": true'
assert_contains "${queue_json}" '"area_flow_execution_state_written": true'
assert_contains "${queue_json}" '"project_read_attempted": false'
assert_contains "${queue_json}" '"project_write_attempted": false'
assert_contains "${queue_json}" '"execution_write_attempted": false'
assert_contains "${queue_json}" '"engine_call_attempted": false'
assert_contains "${queue_json}" '"commands_run": false'
assert_contains "${queue_json}" '"secrets_resolved": false'
assert_contains "${queue_json}" '"network_used": false'

run_id="$(json_run_id "${queue_json}")"
if [[ -z "${run_id}" ]]; then
  echo "smoke-managed-generated-write: expected managed generated write run id" >&2
  exit 1
fi

echo "smoke-managed-generated-write: managed generated write gate"
gate_json="$(go run ./cmd/areaflow run managed-generated-write-gate "${run_id}" --json)"
assert_contains "${gate_json}" '"status": "ready"'
assert_contains "${gate_json}" '"generated_only_write_ready": true'
assert_contains "${gate_json}" '"generated_only_apply_open": false'
assert_contains "${gate_json}" '"write_generated"'
assert_contains "${gate_json}" '"source_write"'
assert_contains "${gate_json}" '"execution_write_attempted": false'

echo "smoke-managed-generated-write: apply fixture generated write"
apply_json="$(go run ./cmd/areaflow worker managed-generated-write "${project_key}" "${worker_key}" --run-id "${run_id}" --capability read_project --capability write_artifacts --capability write_generated --idempotency-key "managed-generated-write-${workflow_label}" --reason "managed generated smoke apply" --json)"
assert_contains "${apply_json}" '"status": "rollback_verified"'
assert_contains "${apply_json}" '"decision": "allowed"'
assert_contains "${apply_json}" '"artifact_type": "managed_generated_write_set"'
assert_contains "${apply_json}" '"artifact_type": "managed_generated_write_preimage"'
assert_contains "${apply_json}" '"artifact_type": "managed_generated_write_report"'
assert_contains "${apply_json}" '"attempt_kind": "copy"'
assert_contains "${apply_json}" '"attempt_kind": "verify"'
assert_contains "${apply_json}" '"attempt_kind": "rollback"'
assert_contains "${apply_json}" '"generated_only": true'
assert_contains "${apply_json}" '"generated_only_apply_open": true'
assert_contains "${apply_json}" '"project_read_attempted": true'
assert_contains "${apply_json}" '"project_read_allowed": true'
assert_contains "${apply_json}" '"project_write_attempted": true'
assert_contains "${apply_json}" '"project_write_allowed": true'
assert_contains "${apply_json}" '"execution_write_attempted": false'
assert_contains "${apply_json}" '"area_flow_artifact_written": true'
assert_contains "${apply_json}" '"area_flow_execution_state_written": true'
assert_contains "${apply_json}" '"engine_call_attempted": false'
assert_contains "${apply_json}" '"commands_run": false'
assert_contains "${apply_json}" '"secrets_resolved": false'
assert_contains "${apply_json}" '"network_used": false'
assert_contains "${apply_json}" '"write_set_passed": true'
assert_contains "${apply_json}" '"verification_passed": true'
assert_contains "${apply_json}" '"rollback_attempted": true'
assert_contains "${apply_json}" '"rollback_verified": true'

after_apply_sha="$(shasum -a 256 "${project_root}/${target_path}" | awk '{print $1}')"
after_apply_size="$(wc -c <"${project_root}/${target_path}" | tr -d ' ')"
if [[ "${after_apply_sha}" != "${before_sha}" || "${after_apply_size}" != "${before_size}" ]]; then
  echo "smoke-managed-generated-write: fixture generated file was not rolled back" >&2
  exit 1
fi

success_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'worker.managed_generated_write'
  AND cr.completed_at IS NOT NULL
  AND cr.response->>'decision' = 'allowed'
  AND cr.response->>'rollback_verified' = 'true'
  AND cr.response->>'real_areamatrix_write_opened' = 'false';
SQL
)"
if [[ "${success_command_count}" -lt 1 ]]; then
  echo "smoke-managed-generated-write: expected allowed worker.managed_generated_write command response" >&2
  exit 1
fi

echo "smoke-managed-generated-write: register non-fixture deny worker"
deny_worker_json="$(go run ./cmd/areaflow worker register "${deny_project_key}" --worker-key "${deny_worker_key}" --capability read_project --capability write_artifacts --capability write_generated --json)"
assert_contains "${deny_worker_json}" '"worker_key": "'"${deny_worker_key}"'"'
assert_contains "${deny_worker_json}" '"write_generated"'

echo "smoke-managed-generated-write: queue non-fixture generated write"
deny_queue_json="$(go run ./cmd/areaflow run managed-generated-write-queue "${deny_project_key}" "${deny_workflow_label}" --target-path "${target_path}" --content "${after_content}" --expected-before-sha256 "${deny_before_sha}" --expected-before-size "${deny_before_size}" --idempotency-key "managed-generated-write-queue-${deny_workflow_label}" --reason "managed generated deny queue" --json)"
assert_contains "${deny_queue_json}" '"run_type": "managed_generated_write"'
assert_contains "${deny_queue_json}" '"generated_only": true'

deny_run_id="$(json_run_id "${deny_queue_json}")"
if [[ -z "${deny_run_id}" ]]; then
  echo "smoke-managed-generated-write: expected denied managed generated write run id" >&2
  exit 1
fi

echo "smoke-managed-generated-write: non-fixture apply denied"
assert_fails_contains "managed generated write blocked: project is not fixture/temp scoped" go run ./cmd/areaflow worker managed-generated-write "${deny_project_key}" "${deny_worker_key}" --run-id "${deny_run_id}" --capability read_project --capability write_artifacts --capability write_generated --idempotency-key "managed-generated-write-${deny_workflow_label}" --reason "managed generated deny apply" --json

deny_after_sha="$(shasum -a 256 "${deny_project_root}/${target_path}" | awk '{print $1}')"
deny_after_size="$(wc -c <"${deny_project_root}/${target_path}" | tr -d ' ')"
if [[ "${deny_after_sha}" != "${deny_before_sha}" || "${deny_after_size}" != "${deny_before_size}" ]]; then
  echo "smoke-managed-generated-write: non-fixture generated file changed unexpectedly" >&2
  exit 1
fi

denied_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${deny_project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'worker.managed_generated_write'
  AND cr.completed_at IS NOT NULL
  AND cr.response->>'decision' = 'denied'
  AND cr.response->>'real_areamatrix_write_opened' = 'false'
  AND cr.response::text LIKE '%project is not fixture/temp scoped%';
SQL
)"
if [[ "${denied_command_count}" -lt 1 ]]; then
  echo "smoke-managed-generated-write: expected denied worker.managed_generated_write command response" >&2
  exit 1
fi

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"
  if [[ "${real_status_after}" != "${real_status_before}" ]]; then
    echo "smoke-managed-generated-write: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi
  if [[ "${real_readme_after}" != "${real_readme_before}" ]]; then
    echo "smoke-managed-generated-write: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-managed-generated-write: PASS"
