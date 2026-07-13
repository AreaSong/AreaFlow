#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-compatibility-fixture: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_COMPAT_FIXTURE_PROJECT_KEY:-areamatrix-compat-fixture}"
workflow_label="${AREAFLOW_COMPAT_FIXTURE_WORKFLOW_VERSION:-compat-smoke-$(date +%Y%m%d%H%M%S)}"
ready_workflow_label="${AREAFLOW_COMPAT_FIXTURE_READY_WORKFLOW_VERSION:-${workflow_label}-ready}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-compat-fixture.XXXXXX")"
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
    echo "smoke-compatibility-fixture: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-compatibility-fixture: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-compatibility-fixture: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-compatibility-fixture: output unexpectedly contained pattern: ${pattern}" >&2
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
    echo "smoke-compatibility-fixture: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
  printf "%s\n" "${output}"
}

assert_not_exists() {
  local path="$1"

  if [[ -e "${path}" ]]; then
    echo "smoke-compatibility-fixture: expected path to be absent: ${path}" >&2
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
fi

mkdir -p \
  "${project_root}/docs" \
  "${project_root}/workflow/residuals" \
  "${project_root}/workflow/templates" \
  "${project_root}/workflow/versions/v1-mvp/residuals" \
  "${project_root}/workflow/versions/v1-mvp/execution/_shared" \
  "${project_root}/workflow/versions/v-template" \
  "${project_root}/tasks/active" \
  "${project_root}/tasks/done" \
  "${project_root}/tasks/backlog/prompts/sample-package" \
  "${project_root}/tasks/indexes" \
  "${artifact_root}"

for stage in discussion middle-layer changes plans drafts queue promotion projection closeout; do
  mkdir -p "${project_root}/workflow/versions/v1-mvp/${stage}"
  printf "# %s fixture\n" "${stage}" >"${project_root}/workflow/versions/v1-mvp/${stage}/README.md"
done

cat >"${project_root}/workflow/intake.md" <<'EOF'
# Fixture Intake

This fixture exists only for AreaFlow compatibility smoke tests.
EOF

cat >"${project_root}/docs/README.md" <<'EOF'
# Fixture Docs

Minimal source docs coverage for the AreaMatrix profile.
EOF

cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Fixture Templates

Minimal template coverage for the AreaMatrix profile.
EOF

cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items:
  - id: global-compat-fixture-note
    status: reference-only
    type: fixture
    title: Global compatibility fixture residual
    source: workflow/residuals/residuals.yaml
    current_impact: proves global residual import for compatibility smoke
    executable_task: false
    promotion_required: false
    close_condition: compatibility smoke passes
version_residuals:
  - version: v1-mvp
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    status: mixed-blocked
    summary: Fixture v1 metadata
EOF

cat >"${project_root}/workflow/versions/v1-mvp/residuals/residuals.yaml" <<'EOF'
version_status:
  technical_queue: complete
  formal_alpha: blocked
  fixture: true
items:
  - id: v1-compat-fixture-residual
    status: blocked-decision
    type: fixture
    title: Version compatibility fixture residual
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    current_impact: proves version residual import for compatibility smoke
    executable_task: false
    promotion_required: true
    close_condition: fixture decision is recorded
EOF

cat >"${project_root}/workflow/versions/v1-mvp/execution/_shared/progress.json" <<'EOF'
{
  "tasks": {
    "fixture-001": {"status": "completed"},
    "fixture-002": {"status": "completed"}
  }
}
EOF

cat >"${project_root}/workflow/versions/v-template/README.md" <<'EOF'
# Fixture Version Template

Template-only workflow version placeholder.
EOF

cat >"${project_root}/tasks/indexes/residuals.md" <<'EOF'
# Fixture Residual Index
EOF

cat >"${project_root}/tasks/backlog/README.md" <<'EOF'
# Fixture Backlog
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Compatibility Fixture
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
    write_status: true
    write_artifacts: true
    write_workflow: false
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
    - tasks/**

  write_paths:
    - .areaflow/status.json

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
  imported_versions:
    - v1-mvp
    - v-template
  immutable_imports:
    - v1-mvp
EOF

echo "smoke-compatibility-fixture: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-compatibility-fixture: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-compatibility-fixture: project import ${project_key} #1"
import_output_1="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output_1}" "imported ${project_key}"

echo "smoke-compatibility-fixture: project import ${project_key} #2"
import_output_2="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output_2}" "imported ${project_key}"

import_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.import'
  AND cr.completed_at IS NOT NULL
  AND cr.response ? 'run_id'
  AND cr.response ? 'status_snapshot';
SQL
)"
if [[ "${import_command_count}" -lt 2 ]]; then
  echo "smoke-compatibility-fixture: expected two project.import command requests, got ${import_command_count}" >&2
  exit 1
fi

echo "smoke-compatibility-fixture: project doctor --json"
doctor_json="$(go run ./cmd/areaflow project doctor "${project_key}" --json)"
assert_contains "${doctor_json}" '"name": "hash_drift"'
assert_contains "${doctor_json}" '"name": "project_config_drift"'
assert_contains "${doctor_json}" '"name": "stage_coverage"'
doctor_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.doctor.record'
  AND cr.completed_at IS NOT NULL
  AND cr.response ? 'event_id';
SQL
)"
if [[ "${doctor_command_count}" -lt 1 ]]; then
  echo "smoke-compatibility-fixture: expected project.doctor.record command request, got ${doctor_command_count}" >&2
  exit 1
fi

fixture_status="${project_root}/.areaflow/status.json"
echo "smoke-compatibility-fixture: project status-projection-authorization ${project_key}"
authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
assert_contains "${authorization_json}" '"status": "needs_approval"'
assert_contains "${authorization_json}" '"schema_status": "missing"'
source_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["source_hash"])' <<<"${authorization_json}")"
validator_preflight="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["validator_preflight"])' <<<"${authorization_json}")"
protected_path_fingerprint_sha256="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["protected_path_fingerprint_sha256"])' <<<"${authorization_json}")"
protected_path_check="git -C ${project_root} status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"

echo "smoke-compatibility-fixture: project status-projection-apply ${project_key}"
export_output="$(
  go run ./cmd/areaflow project status-projection-apply "${project_key}" \
    --expected-before-exists false \
    --expected-before-size 0 \
    --source-hash "${source_hash}" \
    --schema-uri "schemas/status-projection.schema.json" \
    --validator-preflight "${validator_preflight}" \
    --protected-path-check "${protected_path_check}" \
    --protected-path-fingerprint-sha256 "${protected_path_fingerprint_sha256}" \
    --rollback-action "delete .areaflow/status.json if apply created it" \
    --accept-preimage-schema "missing" \
    --explicit-approval \
    --approval-actor "smoke-compatibility-fixture" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${export_output}" "${fixture_status}"
assert_contains "${export_output}" "apply_gate: status=pass decision=go"
if [[ ! -f "${fixture_status}" ]]; then
  echo "smoke-compatibility-fixture: expected fixture status export at ${fixture_status}" >&2
  exit 1
fi

case "${fixture_status}" in
  "${project_root}"/.areaflow/status.json) ;;
  *)
    echo "smoke-compatibility-fixture: status export escaped fixture root: ${fixture_status}" >&2
    exit 1
    ;;
esac

echo "smoke-compatibility-fixture: validate status projection schema"
python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json "${fixture_status}"

echo "smoke-compatibility-fixture: project compatibility --json"
compat_json="$(go run ./cmd/areaflow project compatibility "${project_key}" --json)"
assert_contains "${compat_json}" '"status": "pass"'
assert_contains "${compat_json}" '"command": "./task-loop run"'
assert_contains "${compat_json}" '"mode": "blocked"'
assert_contains "${compat_json}" '"fallback": "blocked until v0.5/v0.6 runner model"'
assert_contains "${compat_json}" '"blocked_reason": "execution and task-loop replacement are out of v0.4 scope"'

echo "smoke-compatibility-fixture: project shim-preview --json"
shim_preview_json="$(go run ./cmd/areaflow project shim-preview "${project_key}" --json)"
assert_contains "${shim_preview_json}" '"mode": "read_only_planning"'
assert_contains "${shim_preview_json}" '"path": "scripts/areaflow_shim.py"'
assert_contains "${shim_preview_json}" '"path": "scripts/task_loop/console.py"'
assert_contains "${shim_preview_json}" '"command": "./task-loop run"'
assert_contains "${shim_preview_json}" '"mode": "blocked"'
assert_contains "${shim_preview_json}" '"./task-loop run"'
assert_contains "${shim_preview_json}" '"workflow/versions/**/execution/**"'

echo "smoke-compatibility-fixture: project shim-readiness --json"
shim_readiness_json="$(go run ./cmd/areaflow project shim-readiness "${project_key}" --json)"
assert_contains "${shim_readiness_json}" '"status": "blocked"'
assert_contains "${shim_readiness_json}" '"key": "task_loop_run_blocked"'
assert_contains "${shim_readiness_json}" '"key": "explicit_edit_approval"'
assert_contains "${shim_readiness_json}" '"key": "real_areamatrix_readonly_smoke"'
assert_contains "${shim_readiness_json}" '"key": "real_areamatrix_status_projection_schema"'
assert_contains "${shim_readiness_json}" '"evidence_recorded": false'
assert_contains "${shim_readiness_json}" '"schema_contract": "stable_fallback_projection_v1"'
assert_contains "${shim_readiness_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${shim_readiness_json}" '"schema_uri": "schemas/status-projection.schema.json"'
assert_contains "${shim_readiness_json}" '"validator_preflight": "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"'
assert_contains "${shim_readiness_json}" '"required_schema_fields": ['
assert_contains "${shim_readiness_json}" '"compatibility.blocked_commands[]"'
assert_contains "${shim_readiness_json}" '"forbidden_fields": ['
assert_contains "${shim_readiness_json}" '"artifact_content"'

echo "smoke-compatibility-fixture: project shim-authorization --json"
shim_authorization_json="$(go run ./cmd/areaflow project shim-authorization "${project_key}" --json)"
assert_contains "${shim_authorization_json}" '"mode": "read_only_authorization_packet"'
assert_contains "${shim_authorization_json}" '"status": "blocked"'
assert_contains "${shim_authorization_json}" '"readiness_status": "blocked"'
assert_contains "${shim_authorization_json}" '"path": "scripts/task_loop/console.py"'
assert_contains "${shim_authorization_json}" '"workflow/versions/**"'
assert_contains "${shim_authorization_json}" '"task-loop run forwarding"'
assert_contains "${shim_authorization_json}" '"areaflow project shim-authorization areamatrix --json"'
assert_contains "${shim_authorization_json}" '"areaflow project status-projections areamatrix --json"'
assert_contains "${shim_authorization_json}" '"python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"'
assert_contains "${shim_authorization_json}" '"verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash"'
assert_contains "${shim_authorization_json}" '"git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"'
assert_contains "${shim_authorization_json}" '"git status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"'
assert_contains "${shim_authorization_json}" '"project_write_attempted": false'
assert_contains "${shim_authorization_json}" '"execution_write_attempted": false'
assert_contains "${shim_authorization_json}" '"task_loop_run_forwarded": false'

echo "smoke-compatibility-fixture: project shim-authorization text"
shim_authorization_text="$(go run ./cmd/areaflow project shim-authorization "${project_key}")"
assert_contains "${shim_authorization_text}" "required_preflight.count:"
assert_contains "${shim_authorization_text}" "required_preflight: areaflow project shim-authorization areamatrix --json"
assert_contains "${shim_authorization_text}" "required_preflight: areaflow project status-projections areamatrix --json"
assert_contains "${shim_authorization_text}" "required_preflight: python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
assert_contains "${shim_authorization_text}" "required_preflight: verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash"
assert_contains "${shim_authorization_text}" "required_preflight: git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json"
assert_contains "${shim_authorization_text}" "required_preflight: AREAFLOW_DATABASE_URL=... bash scripts/smoke-areamatrix-readonly.sh"
assert_contains "${shim_authorization_text}" "post_edit_verification.count:"
assert_contains "${shim_authorization_text}" "post_edit_verification: verify ./task-loop run returns blocked"
assert_contains "${shim_authorization_text}" "post_edit_verification: git status --short -- workflow/README.md .areaflow/status.json"
assert_contains "${shim_authorization_text}" "rollback_scope.count:"
assert_contains "${shim_authorization_text}" "rollback_scope: do not write v1 historical execution, progress.json, logs or checkpoints"

echo "smoke-compatibility-fixture: project shim-apply-packet before evidence --json"
shim_apply_packet_before_json="$(go run ./cmd/areaflow project shim-apply-packet "${project_key}" --json)"
assert_contains "${shim_apply_packet_before_json}" '"mode": "shim_apply_packet_preview_v1"'
assert_contains "${shim_apply_packet_before_json}" '"status": "blocked"'
assert_contains "${shim_apply_packet_before_json}" '"decision": "readiness_blocked"'
assert_contains "${shim_apply_packet_before_json}" '"command_type": "project.shim.apply"'
assert_contains "${shim_apply_packet_before_json}" '"authorization_snapshot_hash": "'
assert_contains "${shim_apply_packet_before_json}" '"expected_authorization_mode": "read_only_authorization_packet"'
assert_contains "${shim_apply_packet_before_json}" '"apply_gate_command": ['
assert_contains "${shim_apply_packet_before_json}" '"shim-apply-gate"'
assert_contains "${shim_apply_packet_before_json}" '"key": "readiness_blockers"'
assert_contains "${shim_apply_packet_before_json}" '"shim_readiness_still_blocked"'
assert_contains "${shim_apply_packet_before_json}" '"command_request_created": false'
assert_contains "${shim_apply_packet_before_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_packet_before_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_packet_before_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_packet_before_json}" '"status_projection_written": false'
assert_contains "${shim_apply_packet_before_json}" '"area_matrix_files_modified": false'

echo "smoke-compatibility-fixture: project shim-apply-gate before evidence --json"
shim_apply_gate_before_json="$(go run ./cmd/areaflow project shim-apply-gate "${project_key}" --json)"
assert_contains "${shim_apply_gate_before_json}" '"mode": "shim_apply_gate_v1"'
assert_contains "${shim_apply_gate_before_json}" '"status": "blocked"'
assert_contains "${shim_apply_gate_before_json}" '"decision": "no_go"'
assert_contains "${shim_apply_gate_before_json}" '"apply_command_eligible": false'
assert_contains "${shim_apply_gate_before_json}" '"key": "readiness_blockers"'
assert_contains "${shim_apply_gate_before_json}" '"key": "authorization_snapshot_hash"'
assert_contains "${shim_apply_gate_before_json}" '"key": "explicit_approval"'
assert_contains "${shim_apply_gate_before_json}" '"explicit_shim_apply_approval_missing"'
assert_contains "${shim_apply_gate_before_json}" '"command_request_created": false'
assert_contains "${shim_apply_gate_before_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_gate_before_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_gate_before_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_gate_before_json}" '"status_projection_written": false'
assert_contains "${shim_apply_gate_before_json}" '"area_matrix_files_modified": false'

echo "smoke-compatibility-fixture: project shim-readiness-evidence real_areamatrix_readonly_smoke"
readonly_evidence_json="$(go run ./cmd/areaflow project shim-readiness-evidence "${project_key}" \
  --key real_areamatrix_readonly_smoke \
  --status pass \
  --summary "compat fixture proves read-only smoke evidence recording" \
  --evidence-uri "docs/development/compatibility-shim-readiness-evidence.md" \
  --json)"
assert_contains "${readonly_evidence_json}" '"evidence_key": "real_areamatrix_readonly_smoke"'
assert_contains "${readonly_evidence_json}" '"status": "recorded"'
assert_contains "${readonly_evidence_json}" '"decision": "allowed"'
assert_contains "${readonly_evidence_json}" '"project_write_attempted": false'
assert_contains "${readonly_evidence_json}" '"execution_write_attempted": false'
assert_contains "${readonly_evidence_json}" '"engine_call_attempted": false'

echo "smoke-compatibility-fixture: project shim-readiness-evidence real_areamatrix_status_projection_schema"
status_schema_evidence_json="$(go run ./cmd/areaflow project shim-readiness-evidence "${project_key}" \
  --key real_areamatrix_status_projection_schema \
  --status pass \
  --summary "compat fixture proves status projection schema evidence recording" \
  --evidence-uri "schemas/status-projection.schema.json" \
  --json)"
assert_contains "${status_schema_evidence_json}" '"evidence_key": "real_areamatrix_status_projection_schema"'
assert_contains "${status_schema_evidence_json}" '"status": "recorded"'
assert_contains "${status_schema_evidence_json}" '"decision": "allowed"'
assert_contains "${status_schema_evidence_json}" '"project_write_attempted": false'
assert_contains "${status_schema_evidence_json}" '"execution_write_attempted": false'
assert_contains "${status_schema_evidence_json}" '"engine_call_attempted": false'

echo "smoke-compatibility-fixture: project shim-readiness-evidence areamatrix_dirty_worktree_review"
dirty_review_evidence_json="$(go run ./cmd/areaflow project shim-readiness-evidence "${project_key}" \
  --key areamatrix_dirty_worktree_review \
  --status pass \
  --summary "compat fixture proves dirty worktree review evidence recording" \
  --evidence-uri "docs/development/compatibility-shim-readiness-evidence.md" \
  --json)"
assert_contains "${dirty_review_evidence_json}" '"evidence_key": "areamatrix_dirty_worktree_review"'
assert_contains "${dirty_review_evidence_json}" '"status": "recorded"'
assert_contains "${dirty_review_evidence_json}" '"decision": "allowed"'
assert_contains "${dirty_review_evidence_json}" '"project_write_attempted": false'
assert_contains "${dirty_review_evidence_json}" '"execution_write_attempted": false'
assert_contains "${dirty_review_evidence_json}" '"engine_call_attempted": false'

shim_evidence_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.shim_readiness_evidence.record'
  AND cr.completed_at IS NOT NULL
  AND cr.response ? 'event_id'
  AND cr.response ? 'audit_event_id'
  AND cr.response ? 'project_write_attempted'
  AND cr.response ? 'execution_write_attempted';
SQL
)"
if [[ "${shim_evidence_command_count}" -lt 3 ]]; then
  echo "smoke-compatibility-fixture: expected three shim readiness evidence command requests, got ${shim_evidence_command_count}" >&2
  exit 1
fi

echo "smoke-compatibility-fixture: project shim-readiness after evidence --json"
shim_readiness_after_evidence_json="$(go run ./cmd/areaflow project shim-readiness "${project_key}" --json)"
assert_contains "${shim_readiness_after_evidence_json}" '"status": "blocked"'
assert_contains "${shim_readiness_after_evidence_json}" '"key": "real_areamatrix_readonly_smoke"'
assert_contains "${shim_readiness_after_evidence_json}" '"key": "real_areamatrix_status_projection_schema"'
assert_contains "${shim_readiness_after_evidence_json}" '"key": "areamatrix_dirty_worktree_review"'
assert_contains "${shim_readiness_after_evidence_json}" '"evidence_recorded": true'
assert_contains "${shim_readiness_after_evidence_json}" '"evidence_status": "pass"'
assert_contains "${shim_readiness_after_evidence_json}" '"key": "explicit_edit_approval"'
assert_contains "${shim_readiness_after_evidence_json}" '"explicit user approval is required before writing AreaMatrix shim files"'

echo "smoke-compatibility-fixture: project shim-apply-packet after evidence --json"
shim_apply_packet_after_json="$(go run ./cmd/areaflow project shim-apply-packet "${project_key}" \
  --explicit-approval \
  --approval-id shim-approval-1 \
  --approval-actor compat-smoke \
  --approval-reason "compat fixture shim apply packet review" \
  --status-projection-packet-id "${project_key}:status_projection_apply_packet:status-packet-1" \
  --status-projection-gate-id "${project_key}:status_projection_apply_gate:status-gate-1" \
  --read-only-smoke-evidence-id "${project_key}:real_areamatrix_readonly_smoke:smoke-evidence-1" \
  --dirty-worktree-review-id "${project_key}:areamatrix_dirty_worktree_review:dirty-review-1" \
  --protected-path-fingerprint-id "${project_key}:protected_path_fingerprint:protected-fingerprint-1" \
  --rollback-plan-id "${project_key}:rollback_plan:rollback-plan-1" \
  --json)"
assert_contains "${shim_apply_packet_after_json}" '"mode": "shim_apply_packet_preview_v1"'
assert_contains "${shim_apply_packet_after_json}" '"status": "ready"'
assert_contains "${shim_apply_packet_after_json}" '"decision": "ready_for_future_apply_command"'
assert_contains "${shim_apply_packet_after_json}" '"apply_command_eligible": true'
assert_contains "${shim_apply_packet_after_json}" '"would_create_command_request_after_apply_command": true'
assert_contains "${shim_apply_packet_after_json}" '"would_write_area_matrix_shim_files_after_apply_command": true'
assert_contains "${shim_apply_packet_after_json}" '"would_write_status_projection_after_apply_command": true'
assert_contains "${shim_apply_packet_after_json}" '"command_request_created": false'
assert_contains "${shim_apply_packet_after_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_packet_after_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_packet_after_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_packet_after_json}" '"status_projection_written": false'
assert_contains "${shim_apply_packet_after_json}" '"area_matrix_files_modified": false'

echo "smoke-compatibility-fixture: project shim-apply after evidence --json records command"
shim_apply_allowed_files="$(python3 -c 'import json,sys; print(",".join(json.load(sys.stdin)["packet"]["allowed_files"]))' <<<"${shim_apply_packet_after_json}")"
shim_apply_authorization_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["authorization_snapshot_hash"])' <<<"${shim_apply_packet_after_json}")"
shim_apply_expected_authorization_mode="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["expected_authorization_mode"])' <<<"${shim_apply_packet_after_json}")"
shim_apply_approval_scope="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["approval_scope"])' <<<"${shim_apply_packet_after_json}")"
shim_apply_failure_mode="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["failure_mode"])' <<<"${shim_apply_packet_after_json}")"
shim_apply_idempotency_key="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["idempotency_key"])' <<<"${shim_apply_packet_after_json}")"
shim_apply_audit_correlation_id="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["audit_correlation_id"])' <<<"${shim_apply_packet_after_json}")"
shim_apply_after_json="$(go run ./cmd/areaflow project shim-apply "${project_key}" \
    --allowed-files "${shim_apply_allowed_files}" \
    --authorization-snapshot-hash "${shim_apply_authorization_hash}" \
    --expected-authorization-mode "${shim_apply_expected_authorization_mode}" \
    --approval-id shim-approval-1 \
    --approval-scope "${shim_apply_approval_scope}" \
    --explicit-approval \
    --approval-actor compat-smoke \
    --approval-reason "compat fixture shim apply packet review" \
    --status-projection-packet-id "${project_key}:status_projection_apply_packet:status-packet-1" \
    --status-projection-gate-id "${project_key}:status_projection_apply_gate:status-gate-1" \
    --read-only-smoke-evidence-id "${project_key}:real_areamatrix_readonly_smoke:smoke-evidence-1" \
    --dirty-worktree-review-id "${project_key}:areamatrix_dirty_worktree_review:dirty-review-1" \
    --protected-path-fingerprint-id "${project_key}:protected_path_fingerprint:protected-fingerprint-1" \
    --rollback-plan-id "${project_key}:rollback_plan:rollback-plan-1" \
    --failure-mode "${shim_apply_failure_mode}" \
    --idempotency-key "${shim_apply_idempotency_key}" \
    --audit-correlation-id "${shim_apply_audit_correlation_id}" \
    --json)"
assert_contains "${shim_apply_after_json}" '"mode": "shim_apply_command_v1"'
assert_contains "${shim_apply_after_json}" '"status": "recorded"'
assert_contains "${shim_apply_after_json}" '"decision": "allowed"'
assert_contains "${shim_apply_after_json}" '"blockers": []'
assert_not_contains "${shim_apply_after_json}" '"shim_apply_gate_not_pass"'
assert_contains "${shim_apply_after_json}" '"apply_command_eligible": true'
assert_contains "${shim_apply_after_json}" '"apply_open": true'
assert_contains "${shim_apply_after_json}" '"area_flow_command_created": true'
assert_contains "${shim_apply_after_json}" '"command_request_created": true'
assert_contains "${shim_apply_after_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_after_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_after_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_after_json}" '"status_projection_written": false'
assert_contains "${shim_apply_after_json}" '"area_matrix_files_modified": false'

echo "smoke-compatibility-fixture: workflow blocked-path version create ${workflow_label}"
version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${workflow_label}" --json)"
assert_contains "${version_json}" '"import_mode": "authored"'
assert_contains "${version_json}" '"profile_binding": {'

echo "smoke-compatibility-fixture: workflow blocked-path ensure-skeleton ${workflow_label}"
skeleton_json="$(go run ./cmd/areaflow workflow version ensure-skeleton "${project_key}" "${workflow_label}" --json)"
assert_contains "${skeleton_json}" '"links": ['

echo "smoke-compatibility-fixture: workflow blocked-path gate promotion_preview"
promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" promotion_preview --json)"
assert_contains "${promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${promotion_gate_json}" '"status": "fail"'
assert_contains "${promotion_gate_json}" 'placeholder-only'

echo "smoke-compatibility-fixture: workflow blocked-path transition preview"
transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${workflow_label}" --json)"
assert_contains "${transition_json}" '"status": "blocked"'

echo "smoke-compatibility-fixture: workflow blocked-path approval rejected"
approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${workflow_label}" --decision rejected --reason "compat smoke blocked transition preview" --json)"
assert_contains "${approval_json}" '"decision": "rejected"'
assert_contains "${approval_json}" '"transition_status": "blocked"'

echo "smoke-compatibility-fixture: workflow blocked-path approval_gate"
approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" approval_gate --json)"
assert_contains "${approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${approval_gate_json}" '"status": "blocked"'

echo "smoke-compatibility-fixture: workflow blocked-path live_mapping_gate"
live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" live_mapping_gate --json)"
assert_contains "${live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${live_mapping_gate_json}" '"status": "blocked"'
assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-compatibility-fixture: project blocked-path cutover-readiness"
cutover_json="$(go run ./cmd/areaflow project cutover-readiness "${project_key}" --version "${workflow_label}" --json)"
assert_contains "${cutover_json}" '"status": "blocked"'
assert_contains "${cutover_json}" '"name": "v0.4-cutover-readiness"'
assert_contains "${cutover_json}" 'approval_gate is blocked'
assert_contains "${cutover_json}" 'live_mapping_gate is blocked'

echo "smoke-compatibility-fixture: workflow blocked-path cutover_readiness_gate"
cutover_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" cutover_readiness_gate --json)"
assert_contains "${cutover_gate_json}" '"gate_name": "cutover_readiness_gate"'
assert_contains "${cutover_gate_json}" '"status": "blocked"'
assert_contains "${cutover_gate_json}" '"cutover_apply_attempted": false'
assert_contains "${cutover_gate_json}" '"execution_write_attempted": false'

echo "smoke-compatibility-fixture: workflow ready-path version create ${ready_workflow_label}"
ready_version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${ready_workflow_label}" --json)"
assert_contains "${ready_version_json}" '"import_mode": "authored"'

echo "smoke-compatibility-fixture: workflow ready-path mark queue"
ready_queue_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${ready_workflow_label}" --stage queue --item-type queue_candidate --reason "compat smoke ready path queue" --json)"
assert_contains "${ready_queue_json}" '"status": "ready"'

echo "smoke-compatibility-fixture: workflow ready-path mark promotion_preview"
ready_promotion_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${ready_workflow_label}" --stage promotion_preview --item-type promotion_preview --reason "compat smoke ready path promotion" --json)"
assert_contains "${ready_promotion_json}" '"status": "ready"'

echo "smoke-compatibility-fixture: workflow ready-path gate promotion_preview"
ready_promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" promotion_preview --json)"
assert_contains "${ready_promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${ready_promotion_gate_json}" '"status": "pass"'
assert_contains "${ready_promotion_gate_json}" '"placeholder_items": []'

echo "smoke-compatibility-fixture: workflow ready-path transition preview"
ready_transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${ready_workflow_label}" --json)"
assert_contains "${ready_transition_json}" '"status": "ready"'

echo "smoke-compatibility-fixture: workflow ready-path approval approved"
ready_approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${ready_workflow_label}" --decision approved --reason "compat smoke ready path approval" --json)"
assert_contains "${ready_approval_json}" '"decision": "approved"'
assert_contains "${ready_approval_json}" '"transition_status": "ready"'

approval_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
JOIN approval_records ar ON ar.id = (cr.response ->> 'approval_record_id')::bigint
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'workflow.approval.record'
  AND cr.completed_at IS NOT NULL
  AND ar.decision IN ('approved', 'rejected');
SQL
)"
if [[ "${approval_command_count}" -lt 2 ]]; then
  echo "smoke-compatibility-fixture: expected approval command requests for blocked and ready paths, got ${approval_command_count}" >&2
  exit 1
fi

echo "smoke-compatibility-fixture: workflow ready-path approval_gate"
ready_approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" approval_gate --json)"
assert_contains "${ready_approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${ready_approval_gate_json}" '"status": "pass"'

echo "smoke-compatibility-fixture: workflow ready-path live_mapping_gate"
ready_live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" live_mapping_gate --json)"
assert_contains "${ready_live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${ready_live_mapping_gate_json}" '"status": "pass"'
assert_contains "${ready_live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-compatibility-fixture: project ready-path cutover-readiness"
ready_cutover_json="$(go run ./cmd/areaflow project cutover-readiness "${project_key}" --version "${ready_workflow_label}" --json)"
assert_contains "${ready_cutover_json}" '"status": "pass"'
assert_contains "${ready_cutover_json}" '"key": "compatibility_contract"'
assert_contains "${ready_cutover_json}" '"message": "compatibility contract is ready"'
assert_contains "${ready_cutover_json}" '"key": "approval_gate"'
assert_contains "${ready_cutover_json}" '"message": "approval_gate passed"'
assert_contains "${ready_cutover_json}" '"key": "live_mapping_gate"'
assert_contains "${ready_cutover_json}" '"message": "live_mapping_gate passed"'

echo "smoke-compatibility-fixture: workflow ready-path cutover_readiness_gate"
ready_cutover_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" cutover_readiness_gate --json)"
assert_contains "${ready_cutover_gate_json}" '"gate_name": "cutover_readiness_gate"'
assert_contains "${ready_cutover_gate_json}" '"status": "pass"'
assert_contains "${ready_cutover_gate_json}" '"phase_gate_status": "pass"'
assert_contains "${ready_cutover_gate_json}" '"cutover_apply_attempted": false'
assert_contains "${ready_cutover_gate_json}" '"execution_write_attempted": false'

echo "smoke-compatibility-fixture: project ready-path cutover-apply"
ready_cutover_apply_json="$(go run ./cmd/areaflow project cutover-apply "${project_key}" --version "${ready_workflow_label}" --json)"
assert_contains "${ready_cutover_apply_json}" '"status": "applied"'
assert_contains "${ready_cutover_apply_json}" '"decision": "allowed"'
assert_contains "${ready_cutover_apply_json}" '"lifecycle_status": "authoring_cutover"'
assert_contains "${ready_cutover_apply_json}" '"project_write_attempted": false'
assert_contains "${ready_cutover_apply_json}" '"execution_write_attempted": false'
assert_contains "${ready_cutover_apply_json}" '"cutover_readiness_gate_id":'

cutover_apply_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.cutover.apply'
  AND cr.completed_at IS NOT NULL
  AND cr.response ? 'workflow_version_id'
  AND cr.response ->> 'decision' = 'allowed'
  AND cr.response ->> 'project_write_attempted' = 'false'
  AND cr.response ->> 'execution_write_attempted' = 'false';
SQL
)"
if [[ "${cutover_apply_command_count}" -lt 1 ]]; then
  echo "smoke-compatibility-fixture: expected project.cutover.apply command request, got ${cutover_apply_command_count}" >&2
  exit 1
fi

echo "smoke-compatibility-fixture: run ready-path preview"
runner_json="$(go run ./cmd/areaflow run preview "${project_key}" "${ready_workflow_label}" --idempotency-key "compat-fixture-run-preview-${ready_workflow_label}" --json)"
assert_contains "${runner_json}" '"dry_run": true'
assert_contains "${runner_json}" '"status": "passed"'
preview_run_id="$(RUNNER_JSON="${runner_json}" python3 - <<'PY'
import json
import os
print(json.loads(os.environ["RUNNER_JSON"])["run"]["id"])
PY
)"
runner_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -v "run_id=${preview_run_id}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'runner.preview'
  AND cr.completed_at IS NOT NULL
  AND (cr.response ->> 'run_id')::bigint = :'run_id'::bigint
  AND cr.response ->> 'run_status' = 'passed'
  AND cr.response ->> 'dry_run' = 'true'
  AND cr.response ->> 'project_write_attempted' = 'false'
  AND cr.response ->> 'execution_write_attempted' = 'false'
  AND cr.response ->> 'area_matrix_write_attempted' = 'false'
  AND cr.response ->> 'engine_call_attempted' = 'false'
  AND cr.response ->> 'commands_run' = 'false'
  AND cr.response ->> 'secrets_resolved' = 'false'
  AND cr.response ->> 'network_used' = 'false';
SQL
)"
if [[ "${runner_command_count}" -lt 1 ]]; then
  echo "smoke-compatibility-fixture: expected runner.preview command response safety facts, got ${runner_command_count}" >&2
  exit 1
fi

psql "${AREAFLOW_DATABASE_URL}" -v "run_id=${preview_run_id}" -At <<'SQL'
UPDATE runs
SET status = 'queued',
    finished_at = NULL,
    summary = jsonb_set(summary, '{smoke_control_fixture}', 'true'::jsonb, true)
WHERE id = :'run_id'::bigint
  AND dry_run = true
  AND run_type = 'runner_preview';
SQL

echo "smoke-compatibility-fixture: run ready-path start"
run_start_json="$(go run ./cmd/areaflow run start "${preview_run_id}" --idempotency-key "compat-fixture-run-start-${preview_run_id}" --reason "compat smoke protected start" --json)"
assert_contains "${run_start_json}" '"previous_status": "queued"'
assert_contains "${run_start_json}" '"status": "running"'
assert_contains "${run_start_json}" '"decision": "allowed"'
assert_contains "${run_start_json}" '"project_write_attempted": false'
assert_contains "${run_start_json}" '"execution_write_attempted": false'
assert_contains "${run_start_json}" '"area_matrix_write_attempted": false'
assert_contains "${run_start_json}" '"engine_call_attempted": false'

echo "smoke-compatibility-fixture: run ready-path drain"
run_drain_json="$(go run ./cmd/areaflow run drain "${preview_run_id}" --idempotency-key "compat-fixture-run-drain-${preview_run_id}" --reason "compat smoke protected drain" --json)"
assert_contains "${run_drain_json}" '"previous_status": "running"'
assert_contains "${run_drain_json}" '"status": "draining"'
assert_contains "${run_drain_json}" '"decision": "allowed"'
assert_contains "${run_drain_json}" '"project_write_attempted": false'
assert_contains "${run_drain_json}" '"execution_write_attempted": false'
assert_contains "${run_drain_json}" '"area_matrix_write_attempted": false'
assert_contains "${run_drain_json}" '"engine_call_attempted": false'

echo "smoke-compatibility-fixture: run ready-path cancel"
run_cancel_json="$(go run ./cmd/areaflow run cancel "${preview_run_id}" --idempotency-key "compat-fixture-run-cancel-${preview_run_id}" --reason "compat smoke protected cancel" --json)"
assert_contains "${run_cancel_json}" '"previous_status": "draining"'
assert_contains "${run_cancel_json}" '"status": "cancelling"'
assert_contains "${run_cancel_json}" '"decision": "allowed"'
assert_contains "${run_cancel_json}" '"project_write_attempted": false'
assert_contains "${run_cancel_json}" '"execution_write_attempted": false'
assert_contains "${run_cancel_json}" '"area_matrix_write_attempted": false'
assert_contains "${run_cancel_json}" '"engine_call_attempted": false'

run_control_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -v "run_id=${preview_run_id}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type IN ('run.start', 'run.drain', 'run.cancel')
  AND cr.completed_at IS NOT NULL
  AND (cr.response ->> 'run_id')::bigint = :'run_id'::bigint
  AND cr.response ->> 'decision' = 'allowed'
  AND cr.response ->> 'project_write_attempted' = 'false'
  AND cr.response ->> 'execution_write_attempted' = 'false'
  AND cr.response ->> 'area_matrix_write_attempted' = 'false'
  AND cr.response ->> 'engine_call_attempted' = 'false';
SQL
)"
if [[ "${run_control_command_count}" -lt 3 ]]; then
  echo "smoke-compatibility-fixture: expected run.start/drain/cancel command requests, got ${run_control_command_count}" >&2
  exit 1
fi

echo "smoke-compatibility-fixture: artifact archive preview"
archive_preview_json="$(go run ./cmd/areaflow artifact archive-preview "${project_key}" --idempotency-key "compat-fixture-artifact-archive-${ready_workflow_label}" --reason "compat smoke artifact archive preview" --json)"
assert_contains "${archive_preview_json}" '"mode": "metadata_only_archive_preview"'
assert_contains "${archive_preview_json}" '"project_write_attempted": false'
assert_contains "${archive_preview_json}" '"storage_write_attempted": false'
assert_contains "${archive_preview_json}" '"artifact_delete_attempted": false'
assert_contains "${archive_preview_json}" '"archive_state": "archive_candidate"'
assert_contains "${archive_preview_json}" '"archive_state": "metadata_only_reference"'

artifact_archive_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'artifact.archive.preview'
  AND cr.completed_at IS NOT NULL
  AND cr.response ->> 'mode' = 'metadata_only_archive_preview'
  AND cr.response ->> 'project_write_attempted' = 'false'
  AND cr.response ->> 'storage_write_attempted' = 'false'
  AND cr.response ->> 'artifact_delete_attempted' = 'false';
SQL
)"
if [[ "${artifact_archive_command_count}" -lt 1 ]]; then
  echo "smoke-compatibility-fixture: expected artifact.archive.preview command request, got ${artifact_archive_command_count}" >&2
  exit 1
fi

assert_not_exists "${project_root}/workflow/versions/${workflow_label}/execution"
assert_not_exists "${project_root}/workflow/versions/${ready_workflow_label}/execution"
assert_not_exists "${project_root}/workflow/versions/${workflow_label}/execution/_shared/progress.json"
assert_not_exists "${project_root}/workflow/versions/${ready_workflow_label}/execution/_shared/progress.json"

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"
  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-compatibility-fixture: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi
  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-compatibility-fixture: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
else
  echo "smoke-compatibility-fixture: skipped real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

echo "smoke-compatibility-fixture: pass ${project_key} fixture=${fixture_dir}"
