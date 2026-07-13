#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-completion-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_COMPLETION_PROOF_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-completion-proof.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
evidence_root="docs/history/v1.0/evidence/real-release-candidate-evidence.md"
archive_evidence_uri="${evidence_root}#e4-archive"
shim_evidence_uri="${evidence_root}#e4-shim-retirement"
execution_cutover_evidence_uri="${evidence_root}#e4-execution-cutover"
review_flags=(--review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z)
real_areamatrix_status="/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
real_areamatrix_readme="/Users/as/Ai-Project/project/AreaMatrix/workflow/README.md"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-completion-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-completion-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-proof: expected output to omit pattern: ${pattern}" >&2
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
    echo "smoke-completion-proof: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
}

json_get() {
  local payload="$1"
  local path="$2"

  JSON_PAYLOAD="${payload}" JSON_PATH="${path}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["JSON_PAYLOAD"])
value = data
for part in os.environ["JSON_PATH"].split("."):
    value = value[part]
if isinstance(value, bool):
    print(str(value).lower())
else:
    print(value)
PY
}

assert_positive_value() {
  local label="$1"
  local value="$2"

  if ! [[ "${value}" =~ ^[1-9][0-9]*$ ]]; then
    echo "smoke-completion-proof: expected positive ${label}, got ${value}" >&2
    exit 1
  fi
}

assert_sha256_value() {
  local label="$1"
  local value="$2"

  if ! [[ "${value}" =~ ^[0-9a-f]{64}$ ]]; then
    echo "smoke-completion-proof: expected sha256 ${label}, got ${value}" >&2
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
  echo "smoke-completion-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p \
  "${project_root}/docs" \
  "${project_root}/workflow/residuals" \
  "${project_root}/workflow/templates" \
  "${project_root}/workflow/versions/v1-mvp/execution/_shared" \
  "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Completion Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Completion Proof Fixture Workflow
EOF

cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items: []
version_residuals: []
EOF

cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Completion Proof Fixture Templates
EOF

cat >"${project_root}/workflow/versions/v1-mvp/execution/_shared/progress.json" <<'EOF'
{
  "tasks": {}
}
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Completion Proof Fixture
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
  imported_versions:
    - v1-mvp
  immutable_imports:
    - v1-mvp
EOF

archive_facts=(
  historical_workflow_versions_marked_immutable
  historical_execution_metadata_indexed_in_areaflow
  historical_artifact_refs_have_hash_path_type_project_version_run
  project_reference_restore_limitations_recorded
  old_progress_logs_checkpoints_are_reference_only
  new_run_attempt_artifact_audit_state_owned_by_areaflow
  areamatrix_workflow_readme_summary_contract_reviewed
  areamatrix_status_json_rough_projection_contract_reviewed
  archive_does_not_delete_or_move_historical_files
  archive_does_not_rewrite_progress_json
  rollback_to_execution_forwarding_documented
)

archive_fact_args=()
for fact in "${archive_facts[@]}"; do
  archive_fact_args+=(--fact "${fact}")
done

archive_source_paths=(
  ".areaflow/status.json"
  "workflow/README.md"
  "workflow/versions/**/execution/**"
  "workflow/versions/**/execution/_shared/progress.json"
)

archive_forbidden_actions=(
  copy_artifact_bytes
  delete_artifact_bytes
  delete_historical_files
  move_historical_files
  rewrite_progress_json
  run_commands
  write_areamatrix_protected_paths
)

archive_binding_args=(
  --archive-scope areamatrix_historical_execution_reference_only
  --archive-reference-mode metadata_indexed_reference_only
  --archive-rollback-target execution_forwarding_read_only_shim
  --archive-fail-closed
)
for source_path in "${archive_source_paths[@]}"; do
  archive_binding_args+=(--archive-source-path "${source_path}")
done
for action in "${archive_forbidden_actions[@]}"; do
  archive_binding_args+=(--archive-forbidden-action "${action}")
done

shim_facts=(
  archive_gate_passed
  execution_forwarding_stable_for_declared_window
  no_legacy_task_loop_run_usage_in_active_workflow_versions
  areaflow_run_attempt_artifact_audit_coverage_pass
  compat_commands_mapped_or_deliberately_blocked
  legacy_progress_log_checkpoint_archive_reference_policy_accepted
  rollback_to_read_only_shim_documented
  user_facing_retirement_notice_present
  protected_path_proof_reference_recorded
)

shim_fact_args=()
for fact in "${shim_facts[@]}"; do
  shim_fact_args+=(--fact "${fact}")
done

shim_prerequisites=(
  archive_gate_passed
  execution_cutover_gate_passed
  protected_path_proof_recorded
)

shim_retired_surfaces=(
  legacy_task_loop_runner
  legacy_progress_json_writes
  legacy_logs_writes
  legacy_checkpoint_writes
)

shim_binding_args=(
  --shim-retirement-scope read_only_shim_retirement_after_execution_forwarding_v1
  --shim-rollback-target read_only_shim
  --shim-fail-closed
  --shim-reopen-requires-approval
)
for prerequisite in "${shim_prerequisites[@]}"; do
  shim_binding_args+=(--shim-prerequisite "${prerequisite}")
done
for surface in "${shim_retired_surfaces[@]}"; do
  shim_binding_args+=(--shim-retired-surface "${surface}")
done

execution_cutover_facts=(
  explicit_execution_cutover_approval_recorded
  execution_cutover_command_response_recorded
  execution_cutover_event_and_audit_recorded
  task_loop_run_forwarding_window_proven
  rollback_plan_and_compatibility_window_proven
  no_unapproved_project_or_execution_write_attempted
)

execution_cutover_fact_args=()
for fact in "${execution_cutover_facts[@]}"; do
  execution_cutover_fact_args+=(--fact "${fact}")
done

execution_cutover_allowed_task_types=(
  read_only_verify
  doctor_readiness
  artifact_evidence
  status_projection_validation
  release_readiness_check
)

execution_cutover_forbidden_actions=(
  start_legacy_task_loop_runner
  write_legacy_progress_json
  write_legacy_logs
  write_legacy_checkpoint
  write_areamatrix_source
  write_areamatrix_execution_directory
  generated_retained_write
  repair_apply
  checkpoint_apply
  engine_execution
  secret_resolve
  network_api_integration
  publish_apply
  restore_apply
)

execution_cutover_binding_args=(
  --execution-cutover-scope execution_forwarding_v1_read_only_evidence_only
  --rollback-target read_only_shim
  --rollback-mode fail_closed_to_read_only_shim
  --fail-closed
  --reopen-requires-approval
)
for task_type in "${execution_cutover_allowed_task_types[@]}"; do
  execution_cutover_binding_args+=(--allowed-task-type "${task_type}")
done
for action in "${execution_cutover_forbidden_actions[@]}"; do
  execution_cutover_binding_args+=(--forbidden-action "${action}")
done

echo "smoke-completion-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-completion-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-completion-proof: archive proof rejects incomplete complete status"
assert_fails_contains \
  "complete archive proof missing required facts" \
  go run ./cmd/areaflow completion archive-proof record "${project_key}" \
    --status complete \
    --fact historical_workflow_versions_marked_immutable \
    --json

echo "smoke-completion-proof: archive proof rejects loose complete status without scope binding"
assert_fails_contains \
  "complete archive proof missing archive scope binding" \
  go run ./cmd/areaflow completion archive-proof record "${project_key}" \
    --status complete \
    "${archive_fact_args[@]}" \
    --summary "completion proof smoke loose archive review" \
    --evidence-uri "scripts/smoke-completion-proof.sh#loose-archive" \
    --json

echo "smoke-completion-proof: archive proof complete"
archive_json="$(go run ./cmd/areaflow completion archive-proof record "${project_key}" \
  --status complete \
  "${archive_fact_args[@]}" \
  "${archive_binding_args[@]}" \
  --summary "completion proof smoke archive review" \
  --evidence-uri "${archive_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-proof-smoke:archive:${project_key}" \
  --reason "record archive proof smoke evidence" \
  --json)"
assert_contains "${archive_json}" '"proof_status": "complete"'
assert_contains "${archive_json}" '"decision": "allowed"'
assert_contains "${archive_json}" '"missing_facts": []'
assert_contains "${archive_json}" '"created": true'
assert_contains "${archive_json}" '"project_write_attempted": false'
assert_contains "${archive_json}" '"execution_write_attempted": false'
assert_contains "${archive_json}" '"artifact_bytes_copied": false'
assert_contains "${archive_json}" '"artifact_bytes_deleted": false'
assert_contains "${archive_json}" '"historical_files_deleted": false'
assert_contains "${archive_json}" '"progress_json_rewritten": false'
assert_contains "${archive_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${archive_json}" '"commands_run": false'
assert_contains "${archive_json}" '"archive_scope_binding_status": "pass"'
assert_contains "${archive_json}" '"archive_scope_binding_blockers": []'
assert_contains "${archive_json}" '"archive_scope": "areamatrix_historical_execution_reference_only"'
assert_contains "${archive_json}" '"archive_reference_mode": "metadata_indexed_reference_only"'
assert_contains "${archive_json}" '"archive_rollback_target": "execution_forwarding_read_only_shim"'
assert_contains "${archive_json}" '"archive_fail_closed": true'
assert_contains "${archive_json}" '"review_metadata_status": "approved"'
assert_contains "${archive_json}" '"review_metadata_blockers": []'
archive_event_id="$(json_get "${archive_json}" "event_id")"
archive_audit_event_id="$(json_get "${archive_json}" "audit_event_id")"
archive_scope_binding_hash="$(json_get "${archive_json}" "archive_scope_binding_hash")"
assert_positive_value "archive event_id" "${archive_event_id}"
assert_positive_value "archive audit_event_id" "${archive_audit_event_id}"
assert_sha256_value "archive_scope_binding_hash" "${archive_scope_binding_hash}"

echo "smoke-completion-proof: archive proof idempotent replay"
archive_replay_json="$(go run ./cmd/areaflow completion archive-proof record "${project_key}" \
  --status complete \
  "${archive_fact_args[@]}" \
  "${archive_binding_args[@]}" \
  --summary "completion proof smoke archive review" \
  --evidence-uri "${archive_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-proof-smoke:archive:${project_key}" \
  --reason "record archive proof smoke evidence" \
  --json)"
assert_contains "${archive_replay_json}" '"created": false'
if [[ "$(json_get "${archive_replay_json}" "event_id")" != "${archive_event_id}" ]]; then
  echo "smoke-completion-proof: archive replay changed event_id" >&2
  exit 1
fi
if [[ "$(json_get "${archive_replay_json}" "archive_scope_binding_hash")" != "${archive_scope_binding_hash}" ]]; then
  echo "smoke-completion-proof: archive replay changed archive_scope_binding_hash" >&2
  exit 1
fi

echo "smoke-completion-proof: shim retirement proof rejects incomplete complete status"
assert_fails_contains \
  "complete shim retirement proof missing required facts" \
  go run ./cmd/areaflow completion shim-retirement-proof record "${project_key}" \
    --status complete \
    --fact archive_gate_passed \
    --json

echo "smoke-completion-proof: shim retirement proof rejects loose complete status without scope binding"
assert_fails_contains \
  "complete shim retirement proof missing shim retirement scope binding" \
  go run ./cmd/areaflow completion shim-retirement-proof record "${project_key}" \
    --status complete \
    "${shim_fact_args[@]}" \
    --summary "completion proof smoke loose shim retirement review" \
    --evidence-uri "scripts/smoke-completion-proof.sh#loose-shim-retirement" \
    --json

echo "smoke-completion-proof: shim retirement proof complete"
shim_json="$(go run ./cmd/areaflow completion shim-retirement-proof record "${project_key}" \
  --status complete \
  "${shim_fact_args[@]}" \
  "${shim_binding_args[@]}" \
  --summary "completion proof smoke shim retirement review" \
  --evidence-uri "${shim_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-proof-smoke:shim:${project_key}" \
  --reason "record shim retirement proof smoke evidence" \
  --json)"
assert_contains "${shim_json}" '"proof_status": "complete"'
assert_contains "${shim_json}" '"decision": "allowed"'
assert_contains "${shim_json}" '"missing_facts": []'
assert_contains "${shim_json}" '"created": true'
assert_contains "${shim_json}" '"project_write_attempted": false'
assert_contains "${shim_json}" '"execution_write_attempted": false'
assert_contains "${shim_json}" '"commands_run": false'
assert_contains "${shim_json}" '"legacy_runner_started": false'
assert_contains "${shim_json}" '"legacy_progress_written": false'
assert_contains "${shim_json}" '"legacy_logs_written": false'
assert_contains "${shim_json}" '"legacy_checkpoint_written": false'
assert_contains "${shim_json}" '"historical_files_deleted": false'
assert_contains "${shim_json}" '"progress_json_rewritten": false'
assert_contains "${shim_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${shim_json}" '"shim_retirement_scope_binding_status": "pass"'
assert_contains "${shim_json}" '"shim_retirement_scope_binding_blockers": []'
assert_contains "${shim_json}" '"shim_retirement_scope": "read_only_shim_retirement_after_execution_forwarding_v1"'
assert_contains "${shim_json}" '"shim_rollback_target": "read_only_shim"'
assert_contains "${shim_json}" '"shim_fail_closed": true'
assert_contains "${shim_json}" '"shim_reopen_requires_approval": true'
assert_contains "${shim_json}" '"review_metadata_status": "approved"'
assert_contains "${shim_json}" '"review_metadata_blockers": []'
shim_event_id="$(json_get "${shim_json}" "event_id")"
shim_audit_event_id="$(json_get "${shim_json}" "audit_event_id")"
shim_retirement_scope_binding_hash="$(json_get "${shim_json}" "shim_retirement_scope_binding_hash")"
assert_positive_value "shim retirement event_id" "${shim_event_id}"
assert_positive_value "shim retirement audit_event_id" "${shim_audit_event_id}"
assert_sha256_value "shim_retirement_scope_binding_hash" "${shim_retirement_scope_binding_hash}"

echo "smoke-completion-proof: shim retirement proof idempotent replay"
shim_replay_json="$(go run ./cmd/areaflow completion shim-retirement-proof record "${project_key}" \
  --status complete \
  "${shim_fact_args[@]}" \
  "${shim_binding_args[@]}" \
  --summary "completion proof smoke shim retirement review" \
  --evidence-uri "${shim_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-proof-smoke:shim:${project_key}" \
  --reason "record shim retirement proof smoke evidence" \
  --json)"
assert_contains "${shim_replay_json}" '"created": false'
if [[ "$(json_get "${shim_replay_json}" "event_id")" != "${shim_event_id}" ]]; then
  echo "smoke-completion-proof: shim replay changed event_id" >&2
  exit 1
fi
if [[ "$(json_get "${shim_replay_json}" "shim_retirement_scope_binding_hash")" != "${shim_retirement_scope_binding_hash}" ]]; then
  echo "smoke-completion-proof: shim replay changed shim_retirement_scope_binding_hash" >&2
  exit 1
fi

echo "smoke-completion-proof: execution cutover proof rejects incomplete complete status"
assert_fails_contains \
  "complete execution cutover proof missing required facts" \
  go run ./cmd/areaflow completion execution-cutover-proof record "${project_key}" \
    --status complete \
    --fact explicit_execution_cutover_approval_recorded \
    --json

echo "smoke-completion-proof: execution cutover proof rejects loose complete status without scope binding"
assert_fails_contains \
  "complete execution cutover proof missing execution cutover scope binding" \
  go run ./cmd/areaflow completion execution-cutover-proof record "${project_key}" \
    --status complete \
    "${execution_cutover_fact_args[@]}" \
    --summary "completion proof smoke loose execution cutover review" \
    --evidence-uri "scripts/smoke-completion-proof.sh#loose-execution-cutover" \
    --json

echo "smoke-completion-proof: execution cutover proof complete"
execution_cutover_json="$(go run ./cmd/areaflow completion execution-cutover-proof record "${project_key}" \
  --status complete \
  "${execution_cutover_fact_args[@]}" \
  "${execution_cutover_binding_args[@]}" \
  --summary "completion proof smoke execution cutover review" \
  --evidence-uri "${execution_cutover_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-proof-smoke:execution-cutover:${project_key}" \
  --reason "record execution cutover proof smoke evidence" \
  --json)"
assert_contains "${execution_cutover_json}" '"proof_status": "complete"'
assert_contains "${execution_cutover_json}" '"decision": "allowed"'
assert_contains "${execution_cutover_json}" '"missing_facts": []'
assert_contains "${execution_cutover_json}" '"created": true'
assert_contains "${execution_cutover_json}" '"execution_cutover_scope": "execution_forwarding_v1_read_only_evidence_only"'
assert_contains "${execution_cutover_json}" '"rollback_target": "read_only_shim"'
assert_contains "${execution_cutover_json}" '"rollback_mode": "fail_closed_to_read_only_shim"'
assert_contains "${execution_cutover_json}" '"fail_closed": true'
assert_contains "${execution_cutover_json}" '"reopen_requires_approval": true'
assert_contains "${execution_cutover_json}" '"source_write_open": false'
assert_contains "${execution_cutover_json}" '"generated_retained_write_open": false'
assert_contains "${execution_cutover_json}" '"repair_apply_open": false'
assert_contains "${execution_cutover_json}" '"checkpoint_apply_open": false'
assert_contains "${execution_cutover_json}" '"engine_execution_open": false'
assert_contains "${execution_cutover_json}" '"secret_resolve_open": false'
assert_contains "${execution_cutover_json}" '"network_api_integration_open": false'
assert_contains "${execution_cutover_json}" '"publish_apply_open": false'
assert_contains "${execution_cutover_json}" '"restore_apply_open": false'
assert_contains "${execution_cutover_json}" '"project_write_attempted": false'
assert_contains "${execution_cutover_json}" '"execution_write_attempted": false'
assert_contains "${execution_cutover_json}" '"task_loop_run_forwarded_by_command": false'
assert_contains "${execution_cutover_json}" '"engine_call_attempted": false'
assert_contains "${execution_cutover_json}" '"commands_run": false'
assert_contains "${execution_cutover_json}" '"legacy_progress_written": false'
assert_contains "${execution_cutover_json}" '"legacy_logs_written": false'
assert_contains "${execution_cutover_json}" '"legacy_checkpoint_written": false'
assert_contains "${execution_cutover_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${execution_cutover_json}" '"review_metadata_status": "approved"'
assert_contains "${execution_cutover_json}" '"review_metadata_blockers": []'
assert_contains "${execution_cutover_json}" '"execution_cutover_scope_binding_status": "pass"'
execution_cutover_event_id="$(json_get "${execution_cutover_json}" "event_id")"
execution_cutover_audit_event_id="$(json_get "${execution_cutover_json}" "audit_event_id")"
execution_cutover_scope_binding_hash="$(json_get "${execution_cutover_json}" "execution_cutover_scope_binding_hash")"
assert_positive_value "execution cutover event_id" "${execution_cutover_event_id}"
assert_positive_value "execution cutover audit_event_id" "${execution_cutover_audit_event_id}"
assert_sha256_value "execution_cutover_scope_binding_hash" "${execution_cutover_scope_binding_hash}"

echo "smoke-completion-proof: execution cutover proof idempotent replay"
execution_cutover_replay_json="$(go run ./cmd/areaflow completion execution-cutover-proof record "${project_key}" \
  --status complete \
  "${execution_cutover_fact_args[@]}" \
  "${execution_cutover_binding_args[@]}" \
  --summary "completion proof smoke execution cutover review" \
  --evidence-uri "${execution_cutover_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-proof-smoke:execution-cutover:${project_key}" \
  --reason "record execution cutover proof smoke evidence" \
  --json)"
assert_contains "${execution_cutover_replay_json}" '"created": false'
if [[ "$(json_get "${execution_cutover_replay_json}" "event_id")" != "${execution_cutover_event_id}" ]]; then
  echo "smoke-completion-proof: execution cutover replay changed event_id" >&2
  exit 1
fi
if [[ "$(json_get "${execution_cutover_replay_json}" "execution_cutover_scope_binding_hash")" != "${execution_cutover_scope_binding_hash}" ]]; then
  echo "smoke-completion-proof: execution cutover replay changed execution_cutover_scope_binding_hash" >&2
  exit 1
fi

echo "smoke-completion-proof: completion audit consumes archive, shim and execution cutover proof"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_contains "${completion_json}" '"key": "E4_areamatrix_dogfood_completion"'
assert_contains "${completion_json}" '"latest_archive_proof_event_id": '"${archive_event_id}"
assert_contains "${completion_json}" '"latest_shim_retirement_proof_event_id": '"${shim_event_id}"
assert_contains "${completion_json}" '"latest_execution_cutover_proof_event_id": '"${execution_cutover_event_id}"
assert_contains "${completion_json}" '"latest_archive_proof_evidence_uri": "'"${archive_evidence_uri}"'"'
assert_contains "${completion_json}" '"latest_shim_retirement_proof_evidence_uri": "'"${shim_evidence_uri}"'"'
assert_contains "${completion_json}" '"latest_execution_cutover_proof_evidence_uri": "'"${execution_cutover_evidence_uri}"'"'
assert_contains "${completion_json}" '"archive_proof_review_metadata_status": "approved"'
assert_contains "${completion_json}" '"shim_retirement_proof_review_metadata_status": "approved"'
assert_contains "${completion_json}" '"execution_cutover_proof_review_metadata_status": "approved"'
assert_contains "${completion_json}" '"archive_scope_binding_status": "pass"'
assert_contains "${completion_json}" '"archive_scope_binding_blockers": []'
assert_contains "${completion_json}" '"archive_scope_binding_hash": "'"${archive_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"current_archive_scope_binding_hash": "'"${archive_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"archive_scope_current_binding_bound": true'
assert_contains "${completion_json}" '"archive_scope": "areamatrix_historical_execution_reference_only"'
assert_contains "${completion_json}" '"archive_reference_mode": "metadata_indexed_reference_only"'
assert_contains "${completion_json}" '"archive_rollback_target": "execution_forwarding_read_only_shim"'
assert_contains "${completion_json}" '"archive_fail_closed": true'
assert_contains "${completion_json}" '"shim_retirement_scope_binding_status": "pass"'
assert_contains "${completion_json}" '"shim_retirement_scope_binding_blockers": []'
assert_contains "${completion_json}" '"shim_retirement_scope_binding_hash": "'"${shim_retirement_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"current_shim_retirement_scope_binding_hash": "'"${shim_retirement_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"shim_retirement_scope_current_binding_bound": true'
assert_contains "${completion_json}" '"shim_retirement_scope": "read_only_shim_retirement_after_execution_forwarding_v1"'
assert_contains "${completion_json}" '"shim_rollback_target": "read_only_shim"'
assert_contains "${completion_json}" '"shim_fail_closed": true'
assert_contains "${completion_json}" '"shim_reopen_requires_approval": true'
assert_contains "${completion_json}" '"execution_cutover_scope_binding_status": "pass"'
assert_contains "${completion_json}" '"execution_cutover_scope_binding_blockers": []'
assert_contains "${completion_json}" '"execution_cutover_scope_binding_hash": "'"${execution_cutover_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"current_execution_cutover_scope_binding_hash": "'"${execution_cutover_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"execution_cutover_scope_current_binding_bound": true'
assert_contains "${completion_json}" '"execution_cutover_scope": "execution_forwarding_v1_read_only_evidence_only"'
assert_contains "${completion_json}" '"execution_cutover_rollback_target": "read_only_shim"'
assert_contains "${completion_json}" '"execution_cutover_rollback_mode": "fail_closed_to_read_only_shim"'
assert_not_contains "${completion_json}" '"archive_scope_binding_incomplete"'
assert_not_contains "${completion_json}" '"shim_retirement_scope_binding_incomplete"'
assert_not_contains "${completion_json}" '"execution_cutover_scope_binding_incomplete"'
assert_contains "${completion_json}" '"project_root_not_real_areamatrix"'
assert_contains "${completion_json}" '"execution_cutover_not_complete"'
assert_contains "${completion_json}" '"real_areamatrix_archive_not_proven"'
assert_contains "${completion_json}" '"real_areamatrix_shim_retirement_not_proven"'
assert_contains "${completion_json}" '"status": "blocked"'
assert_contains "${completion_json}" '"database_write_attempted": false'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"task_loop_run_forwarded_by_command": false'
assert_contains "${completion_json}" '"legacy_progress_written": false'
assert_contains "${completion_json}" '"legacy_logs_written": false'
assert_contains "${completion_json}" '"legacy_checkpoint_written": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-completion-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-completion-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-completion-proof: pass ${project_key} fixture=${fixture_dir}"
