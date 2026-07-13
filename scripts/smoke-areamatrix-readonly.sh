#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-areamatrix-readonly: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_READONLY_PROJECT:-areamatrix}"
config_path="${AREAFLOW_READONLY_CONFIG:-examples/areamatrix/areaflow.yaml}"
project_root="${AREAFLOW_READONLY_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
schema_path="schemas/status-projection.schema.json"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
reviewed_dirty_output_hash="${AREAFLOW_READONLY_REVIEWED_DIRTY_OUTPUT_SHA256:-${AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256:-}}"
dirty_reviewer="${AREAFLOW_READONLY_DIRTY_REVIEWER:-${AREAFLOW_PACKAGE_A_DIRTY_REVIEWER:-}}"
temp_dir=""

protected_paths=(
  "workflow/README.md"
  ".areaflow/status.json"
  "scripts/task_loop/console.py"
  "scripts/dev_tools/cli.py"
  "scripts/task_loop/runner.py"
  "scripts/areaflow_shim.py"
  "workflow/versions"
  "workflow/versions/v1-mvp/execution/_shared/progress.json"
)

source "scripts/lib/areamatrix-readonly-guards.sh"

cleanup() {
  if [[ -n "${temp_dir}" ]]; then
    rm -rf "${temp_dir}"
  fi
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-areamatrix-readonly: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-areamatrix-readonly: output unexpectedly contained pattern: ${pattern}" >&2
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
    echo "smoke-areamatrix-readonly: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
  printf "%s\n" "${output}"
}

file_fingerprint() {
  areaflow_file_fingerprint "$@"
}

protected_path_git_status() {
  areaflow_protected_path_git_status "${project_root}" "${protected_paths[@]}"
}

protected_path_status_hash() {
  areaflow_protected_path_status_hash "$@"
}

protected_path_fingerprint() {
  areaflow_protected_path_fingerprint "${project_root}" ".areaflow/status.json" "${protected_paths[@]}"
}

readonly_side_effect_counts() {
  areaflow_readonly_side_effect_counts "${AREAFLOW_DATABASE_URL}" "${project_key}"
}

if [[ ! -d "${project_root}" ]]; then
  echo "smoke-areamatrix-readonly: missing AreaMatrix project root: ${project_root}" >&2
  exit 1
fi

if [[ "${project_key}" != "areamatrix" ]]; then
  temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-readonly-config.XXXXXX")"
  temp_dir="$(cd "${temp_dir}" && pwd -P)"
  case "${temp_dir}" in
    /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
    *)
      echo "smoke-areamatrix-readonly: refusing unsafe temp config dir: ${temp_dir}" >&2
      exit 1
      ;;
  esac
  config_path="${temp_dir}/areaflow.yaml"
  awk -v replacement="  id: ${project_key}" '
    !replaced && $0 ~ /^  id: / {
      print replacement
      replaced = 1
      next
    }
    { print }
  ' "${AREAFLOW_READONLY_CONFIG:-examples/areamatrix/areaflow.yaml}" >"${config_path}"
fi

status_before="$(file_fingerprint "${status_path}")"
workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_before="$(protected_path_fingerprint)"
protected_paths_status_before="$(protected_path_git_status)"
protected_paths_status_hash_before="$(protected_path_status_hash "${protected_paths_status_before}")"
status_projection_preimage_schema="legacy"
if python3 scripts/validate-status-projection-schema.py "${schema_path}" "${status_path}" >/dev/null 2>&1; then
  status_projection_preimage_schema="stable"
fi
echo "smoke-areamatrix-readonly: status_projection_preimage_schema=${status_projection_preimage_schema}"
protected_paths_dirty_review_status="clean"
if [[ -n "${protected_paths_status_before}" ]]; then
  protected_paths_dirty_review_status="required"
  if [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" == "${protected_paths_status_hash_before}" && -n "${dirty_reviewer}" ]]; then
    protected_paths_dirty_review_status="accepted"
    echo "smoke-areamatrix-readonly: protected paths dirty but exact dirty hash was reviewed"
    echo "smoke-areamatrix-readonly: dirty_review_status=accepted"
    echo "smoke-areamatrix-readonly: dirty_reviewer=${dirty_reviewer}"
    echo "smoke-areamatrix-readonly: dirty_output_sha256=${protected_paths_status_hash_before}"
  else
    echo "smoke-areamatrix-readonly: AreaMatrix protected paths are dirty before read-only chain:" >&2
    echo "smoke-areamatrix-readonly: dirty_review_status=${protected_paths_dirty_review_status}" >&2
    if [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" != "${protected_paths_status_hash_before}" ]]; then
      echo "smoke-areamatrix-readonly: dirty_review_status=mismatch" >&2
      echo "smoke-areamatrix-readonly: reviewed_dirty_output_sha256=${reviewed_dirty_output_hash}" >&2
    elif [[ -n "${reviewed_dirty_output_hash}" && -z "${dirty_reviewer}" ]]; then
      echo "smoke-areamatrix-readonly: dirty_review_status=missing_reviewer" >&2
    fi
    echo "smoke-areamatrix-readonly: dirty_output_sha256=${protected_paths_status_hash_before}" >&2
    echo "${protected_paths_status_before}" >&2
    echo "smoke-areamatrix-readonly: dirty_review_input=AREAFLOW_READONLY_REVIEWED_DIRTY_OUTPUT_SHA256=<dirty_output_sha256> AREAFLOW_READONLY_DIRTY_REVIEWER=<reviewer>" >&2
    exit 1
  fi
fi

echo "smoke-areamatrix-readonly: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-areamatrix-readonly: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-areamatrix-readonly: project import ${project_key} #1"
import_output_1="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output_1}" "imported ${project_key}"

echo "smoke-areamatrix-readonly: project import ${project_key} #2"
import_output_2="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output_2}" "imported ${project_key}"

echo "smoke-areamatrix-readonly: project doctor --json"
doctor_json="$(go run ./cmd/areaflow project doctor "${project_key}" --json)"
assert_contains "${doctor_json}" '"name": "hash_drift"'
assert_contains "${doctor_json}" '"name": "project_config_drift"'
assert_contains "${doctor_json}" '"name": "stage_coverage"'
assert_contains "${doctor_json}" '"name": "native_workflow_doctor"'
assert_contains "${doctor_json}" '"status": "pass"'
assert_contains "${doctor_json}" '"status": "warn"'
assert_contains "${doctor_json}" '"reason": "run_commands capability not allowed"'
assert_contains "${doctor_json}" '"allow_native": "false"'
assert_contains "${doctor_json}" 'native workflow doctor skipped by command permission gate'

echo "smoke-areamatrix-readonly: project summary --json"
summary_json="$(go run ./cmd/areaflow project summary "${project_key}" --json)"
assert_contains "${summary_json}" '"key": "'"${project_key}"'"'
assert_contains "${summary_json}" '"root": "'"${project_root}"'"'
assert_contains "${summary_json}" '"history_ready_for_diff": true'
assert_contains "${summary_json}" '"drift_status": "pass"'
assert_contains "${summary_json}" '"config_drift_status": "pass"'
assert_contains "${summary_json}" '"stage_coverage_status": "pass"'
assert_contains "${summary_json}" '"native_doctor_status": "warn"'

echo "smoke-areamatrix-readonly: project readiness --json"
readiness_json="$(go run ./cmd/areaflow project readiness "${project_key}" --json)"
assert_contains "${readiness_json}" '"key": "import_snapshot"'
assert_contains "${readiness_json}" '"key": "import_history"'
assert_contains "${readiness_json}" '"key": "status_mirror"'
assert_contains "${readiness_json}" '"key": "native_workflow_doctor"'
assert_contains "${readiness_json}" '"native_doctor_status": "warn"'

echo "smoke-areamatrix-readonly: project import-diff --json"
diff_json="$(go run ./cmd/areaflow project import-diff "${project_key}" --json)"
assert_contains "${diff_json}" '"status": "unchanged"'
assert_contains "${diff_json}" '"has_previous": true'
assert_contains "${diff_json}" '"source_changed": false'

echo "smoke-areamatrix-readonly: project verify-bundle --json"
bundle_json="$(go run ./cmd/areaflow project verify-bundle "${project_key}" --json)"
assert_contains "${bundle_json}" '"name": "v0.2-shadow-doctor"'
assert_contains "${bundle_json}" '"status": "unchanged"'
assert_contains "${bundle_json}" 'native workflow doctor skipped or warned by permission gate'
assert_contains "${bundle_json}" '"drift_status": "pass"'
assert_contains "${bundle_json}" '"stage_coverage_status": "pass"'

assert_not_contains "${bundle_json}" 'cutover_apply_attempted'
assert_not_contains "${bundle_json}" 'execution_write_attempted'

readonly_side_effect_counts_before="$(readonly_side_effect_counts)"

echo "smoke-areamatrix-readonly: project status-projection-authorization --json"
status_authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
assert_contains "${status_authorization_json}" '"status": "needs_approval"'
assert_contains "${status_authorization_json}" '"decision": "needs_explicit_approval"'
assert_contains "${status_authorization_json}" '"target_kind": "project_status_json"'
assert_contains "${status_authorization_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${status_authorization_json}" '"target_path": "'"${status_path}"'"'
assert_contains "${status_authorization_json}" '"schema_uri": "schemas/status-projection.schema.json"'
assert_contains "${status_authorization_json}" '"validator_preflight": "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json '"${status_path}"'"'
assert_contains "${status_authorization_json}" '"source_hash": "'
assert_contains "${status_authorization_json}" '"summary_state": "mirroring"'
assert_contains "${status_authorization_json}" '"required_authorization_phrase": "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
assert_contains "${status_authorization_json}" '"capability": "write_status"'
assert_contains "${status_authorization_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${status_authorization_json}" '"allowed": true'
if [[ "${status_projection_preimage_schema}" == "stable" ]]; then
  assert_contains "${status_authorization_json}" '"schema_status": "stable"'
  assert_contains "${status_authorization_json}" '"legacy_shape": false'
  assert_contains "${status_authorization_json}" '"missing_required_fields": []'
  assert_contains "${status_authorization_json}" '"unexpected_top_level_fields": []'
  assert_contains "${status_authorization_json}" '"compatibility_missing": []'
  assert_contains "${status_authorization_json}" '"compatibility_unexpected": []'
  assert_contains "${status_authorization_json}" '"target matches stable status projection shape"'
  assert_not_contains "${status_authorization_json}" '"current_target_schema_mismatch_requires_preimage_review"'
else
  assert_contains "${status_authorization_json}" '"schema_status": "legacy"'
  assert_contains "${status_authorization_json}" '"legacy_shape": true'
  assert_contains "${status_authorization_json}" '"missing_required_fields": ['
  assert_contains "${status_authorization_json}" '"schema_version"'
  assert_contains "${status_authorization_json}" '"project_id"'
  assert_contains "${status_authorization_json}" '"project_name"'
  assert_contains "${status_authorization_json}" '"area_flow_url"'
  assert_contains "${status_authorization_json}" '"cutover_phase"'
  assert_contains "${status_authorization_json}" '"active_versions"'
  assert_contains "${status_authorization_json}" '"last_synced_at"'
  assert_contains "${status_authorization_json}" '"source_snapshot_hash"'
  assert_contains "${status_authorization_json}" '"unexpected_top_level_fields": ['
  assert_contains "${status_authorization_json}" '"summary"'
  assert_contains "${status_authorization_json}" '"version"'
  assert_contains "${status_authorization_json}" '"generated_at"'
  assert_contains "${status_authorization_json}" '"source"'
  assert_contains "${status_authorization_json}" '"source_hash"'
  assert_contains "${status_authorization_json}" '"compatibility_missing": ['
  assert_contains "${status_authorization_json}" '"shim_lifecycle_state"'
  assert_contains "${status_authorization_json}" '"blocked_commands"'
  assert_contains "${status_authorization_json}" '"compatibility_unexpected": ['
  assert_contains "${status_authorization_json}" '"status"'
  assert_contains "${status_authorization_json}" '"blocked"'
fi
assert_contains "${status_authorization_json}" '"claim_scope": "package_a_status_projection_preflight_only"'
assert_contains "${status_authorization_json}" '"not_real_100": true'
assert_contains "${status_authorization_json}" '"apply_open": false'
assert_contains "${status_authorization_json}" '"approval_required": true'
assert_contains "${status_authorization_json}" '"approval_status": "missing"'
assert_contains "${status_authorization_json}" '"would_create_command_request_after_approval": true'
assert_contains "${status_authorization_json}" '"would_write_project_file_after_approval": true'
assert_contains "${status_authorization_json}" '"would_write_execution_after_approval": false'
assert_contains "${status_authorization_json}" '"would_run_engine_after_approval": false'
assert_contains "${status_authorization_json}" '"project_write_attempted": false'
assert_contains "${status_authorization_json}" '"execution_write_attempted": false'
assert_contains "${status_authorization_json}" '"engine_call_attempted": false'
assert_contains "${status_authorization_json}" '"requires_preimage_match": true'
assert_contains "${status_authorization_json}" '"requires_schema_validation": true'
assert_contains "${status_authorization_json}" '"protected_paths": ['
assert_contains "${status_authorization_json}" '".areaflow/status.json"'
assert_contains "${status_authorization_json}" '"workflow/versions/v1-mvp/execution/_shared/progress.json"'
assert_contains "${status_authorization_json}" '"blocked_by": ['
assert_contains "${status_authorization_json}" '"explicit_status_projection_apply_approval_missing"'
if [[ "${status_projection_preimage_schema}" == "legacy" ]]; then
  assert_contains "${status_authorization_json}" '"current_target_schema_mismatch_requires_preimage_review"'
fi
assert_contains "${status_authorization_json}" '"rollback_plan": ['
assert_contains "${status_authorization_json}" '"restore the captured preimage bytes for .areaflow/status.json"'
assert_contains "${status_authorization_json}" '"protected_path_fingerprint_sha256": "'
assert_contains "${status_authorization_json}" '"command_request_created": false'
assert_contains "${status_authorization_json}" '"status_projection_written": false'
assert_contains "${status_authorization_json}" '"areamatrix_protected_paths_touched": false'

echo "smoke-areamatrix-readonly: project status-projection-apply-packet --json"
status_apply_packet_json="$(go run ./cmd/areaflow project status-projection-apply-packet "${project_key}" --json)"
assert_contains "${status_apply_packet_json}" '"status": "needs_approval"'
assert_contains "${status_apply_packet_json}" '"claim_scope": "package_a_status_projection_preflight_only"'
assert_contains "${status_apply_packet_json}" '"not_real_100": true'
assert_contains "${status_apply_packet_json}" '"decision": "needs_explicit_approval"'
assert_contains "${status_apply_packet_json}" '"blockers": ['
assert_contains "${status_apply_packet_json}" '"required_authorization_phrase": "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
assert_contains "${status_apply_packet_json}" '"explicit_status_projection_apply_approval_missing"'
if [[ "${status_projection_preimage_schema}" == "legacy" ]]; then
  assert_contains "${status_apply_packet_json}" '"current_target_schema_mismatch_requires_preimage_review"'
else
  assert_not_contains "${status_apply_packet_json}" '"current_target_schema_mismatch_requires_preimage_review"'
fi
assert_contains "${status_apply_packet_json}" '"approval_actor_missing"'
assert_contains "${status_apply_packet_json}" '"approval_reason_missing_or_mismatch"'
assert_contains "${status_apply_packet_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${status_apply_packet_json}" '"schema_status": "'"${status_projection_preimage_schema}"'"'
assert_contains "${status_apply_packet_json}" '"accept_preimage_schema": "'"${status_projection_preimage_schema}"'"'
assert_contains "${status_apply_packet_json}" '"expected_before_exists": true'
assert_contains "${status_apply_packet_json}" '"expected_before_sha256": "'
assert_contains "${status_apply_packet_json}" '"source_hash": "'
assert_contains "${status_apply_packet_json}" '"protected_path_fingerprint_sha256": "'
assert_contains "${status_apply_packet_json}" '"apply_command": ['
assert_contains "${status_apply_packet_json}" '"status-projection-apply"'
assert_contains "${status_apply_packet_json}" '"apply_command_eligible": false'
assert_contains "${status_apply_packet_json}" '"apply_command_eligible_is_not_apply": true'
assert_contains "${status_apply_packet_json}" '"requires_separate_apply_command": true'
assert_contains "${status_apply_packet_json}" '"command_request_created": false'
assert_contains "${status_apply_packet_json}" '"status_projection_written": false'
assert_contains "${status_apply_packet_json}" '"project_write_attempted": false'
assert_contains "${status_apply_packet_json}" '"execution_write_attempted": false'
assert_contains "${status_apply_packet_json}" '"engine_call_attempted": false'
assert_contains "${status_apply_packet_json}" '"areamatrix_protected_paths_touched": false'

echo "smoke-areamatrix-readonly: project status-projection-apply-gate --json"
status_apply_gate_json="$(go run ./cmd/areaflow project status-projection-apply-gate "${project_key}" --json)"
assert_contains "${status_apply_gate_json}" '"status": "blocked"'
assert_contains "${status_apply_gate_json}" '"claim_scope": "package_a_status_projection_preflight_only"'
assert_contains "${status_apply_gate_json}" '"not_real_100": true'
assert_contains "${status_apply_gate_json}" '"decision": "no_go"'
assert_contains "${status_apply_gate_json}" '"target_kind": "project_status_json"'
assert_contains "${status_apply_gate_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${status_apply_gate_json}" '"schema_status": "'"${status_projection_preimage_schema}"'"'
assert_contains "${status_apply_gate_json}" '"apply_command_eligible": false'
assert_contains "${status_apply_gate_json}" '"apply_command_eligible_is_not_apply": true'
assert_contains "${status_apply_gate_json}" '"requires_separate_apply_command": true'
assert_contains "${status_apply_gate_json}" '"approval_status": "missing_or_incomplete"'
assert_contains "${status_apply_gate_json}" '"required_authorization_phrase": "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
assert_contains "${status_apply_gate_json}" '"key": "explicit_approval"'
assert_contains "${status_apply_gate_json}" '"explicit_status_projection_apply_approval_missing"'
assert_contains "${status_apply_gate_json}" '"key": "source_snapshot_hash"'
assert_contains "${status_apply_gate_json}" '"key": "expected_before_sha256"'
assert_contains "${status_apply_gate_json}" '"key": "protected_path_fingerprint_sha256"'
assert_contains "${status_apply_gate_json}" '"command_request_created": false'
assert_contains "${status_apply_gate_json}" '"status_projection_written": false'
assert_contains "${status_apply_gate_json}" '"project_write_attempted": false'
assert_contains "${status_apply_gate_json}" '"execution_write_attempted": false'
assert_contains "${status_apply_gate_json}" '"engine_call_attempted": false'
assert_contains "${status_apply_gate_json}" '"areamatrix_protected_paths_touched": false'

echo "smoke-areamatrix-readonly: project shim-preview --json"
shim_preview_json="$(go run ./cmd/areaflow project shim-preview "${project_key}" --json)"
assert_contains "${shim_preview_json}" '"mode": "read_only_planning"'
assert_contains "${shim_preview_json}" '"path": "scripts/areaflow_shim.py"'
assert_contains "${shim_preview_json}" '"path": "scripts/task_loop/console.py"'
assert_contains "${shim_preview_json}" '"command": "./task-loop run"'
assert_contains "${shim_preview_json}" '"mode": "blocked"'
assert_contains "${shim_preview_json}" '"workflow/versions/**/execution/**"'

echo "smoke-areamatrix-readonly: project shim-readiness --json"
shim_readiness_json="$(go run ./cmd/areaflow project shim-readiness "${project_key}" --json)"
assert_contains "${shim_readiness_json}" '"status": "blocked"'
assert_contains "${shim_readiness_json}" '"key": "real_areamatrix_readonly_smoke"'
assert_contains "${shim_readiness_json}" '"key": "real_areamatrix_status_projection_schema"'
assert_contains "${shim_readiness_json}" '"key": "areamatrix_dirty_worktree_review"'
assert_contains "${shim_readiness_json}" '"key": "explicit_edit_approval"'
assert_contains "${shim_readiness_json}" '"required_script": "scripts/smoke-areamatrix-readonly.sh"'
assert_contains "${shim_readiness_json}" '"schema_contract": "stable_fallback_projection_v1"'
assert_contains "${shim_readiness_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${shim_readiness_json}" '"schema_uri": "schemas/status-projection.schema.json"'
assert_contains "${shim_readiness_json}" '"validator_preflight": "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"'
assert_contains "${shim_readiness_json}" '"required_schema_fields": ['
assert_contains "${shim_readiness_json}" '"compatibility.blocked_commands[]"'
assert_contains "${shim_readiness_json}" '"forbidden_fields": ['
assert_contains "${shim_readiness_json}" '"artifact_content"'

echo "smoke-areamatrix-readonly: project shim-authorization --json"
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
assert_contains "${shim_authorization_json}" '"area_matrix_files_modified": false'

echo "smoke-areamatrix-readonly: project shim-apply-packet --json"
shim_apply_packet_json="$(go run ./cmd/areaflow project shim-apply-packet "${project_key}" --json)"
assert_contains "${shim_apply_packet_json}" '"mode": "shim_apply_packet_preview_v1"'
assert_contains "${shim_apply_packet_json}" '"status": "blocked"'
assert_contains "${shim_apply_packet_json}" '"decision": "readiness_blocked"'
assert_contains "${shim_apply_packet_json}" '"command_type": "project.shim.apply"'
assert_contains "${shim_apply_packet_json}" '"authorization_snapshot_hash": "'
assert_contains "${shim_apply_packet_json}" '"expected_authorization_mode": "read_only_authorization_packet"'
assert_contains "${shim_apply_packet_json}" '"apply_gate_command": ['
assert_contains "${shim_apply_packet_json}" '"shim-apply-gate"'
assert_contains "${shim_apply_packet_json}" '"key": "readiness_blockers"'
assert_contains "${shim_apply_packet_json}" '"shim_readiness_still_blocked"'
assert_contains "${shim_apply_packet_json}" '"command_request_created": false'
assert_contains "${shim_apply_packet_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_packet_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_packet_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_packet_json}" '"status_projection_written": false'
assert_contains "${shim_apply_packet_json}" '"area_matrix_files_modified": false'
assert_contains "${shim_apply_packet_json}" '"engine_call_attempted": false'

echo "smoke-areamatrix-readonly: project shim-apply-gate --json"
shim_apply_gate_json="$(go run ./cmd/areaflow project shim-apply-gate "${project_key}" --json)"
assert_contains "${shim_apply_gate_json}" '"mode": "shim_apply_gate_v1"'
assert_contains "${shim_apply_gate_json}" '"status": "blocked"'
assert_contains "${shim_apply_gate_json}" '"decision": "no_go"'
assert_contains "${shim_apply_gate_json}" '"apply_command_eligible": false'
assert_contains "${shim_apply_gate_json}" '"key": "readiness_blockers"'
assert_contains "${shim_apply_gate_json}" '"key": "authorization_snapshot_hash"'
assert_contains "${shim_apply_gate_json}" '"key": "explicit_approval"'
assert_contains "${shim_apply_gate_json}" '"explicit_shim_apply_approval_missing"'
assert_contains "${shim_apply_gate_json}" '"command_request_created": false'
assert_contains "${shim_apply_gate_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_gate_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_gate_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_gate_json}" '"status_projection_written": false'
assert_contains "${shim_apply_gate_json}" '"area_matrix_files_modified": false'
assert_contains "${shim_apply_gate_json}" '"engine_call_attempted": false'

readonly_side_effect_counts_after_gate="$(readonly_side_effect_counts)"
if [[ "${readonly_side_effect_counts_before}" != "${readonly_side_effect_counts_after_gate}" ]]; then
  echo "smoke-areamatrix-readonly: read-only status/shim packet-gate chain created or modified DB side effects:" >&2
  echo "smoke-areamatrix-readonly: before=${readonly_side_effect_counts_before}" >&2
  echo "smoke-areamatrix-readonly: after=${readonly_side_effect_counts_after_gate}" >&2
  exit 1
fi

echo "smoke-areamatrix-readonly: project shim-apply --json blocks without packet"
shim_apply_json="$(assert_fails_contains '"mode": "shim_apply_command_v1"' go run ./cmd/areaflow project shim-apply "${project_key}" --json)"
assert_contains "${shim_apply_json}" '"status": "blocked"'
assert_contains "${shim_apply_json}" '"decision": "denied"'
assert_contains "${shim_apply_json}" '"shim_apply_gate_blocked"'
assert_contains "${shim_apply_json}" '"shim_apply_gate_not_pass"'
assert_contains "${shim_apply_json}" '"apply_open": false'
assert_contains "${shim_apply_json}" '"area_flow_command_created": true'
assert_contains "${shim_apply_json}" '"command_request_created": true'
assert_contains "${shim_apply_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_json}" '"status_projection_written": false'
assert_contains "${shim_apply_json}" '"area_matrix_files_modified": false'
assert_contains "${shim_apply_json}" '"engine_call_attempted": false'

echo "smoke-areamatrix-readonly: project shim-authorization text"
shim_authorization_text="$(go run ./cmd/areaflow project shim-authorization "${project_key}")"
assert_contains "${shim_authorization_text}" "required_preflight.count:"
assert_contains "${shim_authorization_text}" "required_preflight: areaflow project shim-authorization areamatrix --json"
assert_contains "${shim_authorization_text}" "required_preflight: areaflow project status-projections areamatrix --json"
assert_contains "${shim_authorization_text}" "required_preflight: python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
assert_contains "${shim_authorization_text}" "required_preflight: verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash"
assert_contains "${shim_authorization_text}" "required_preflight: git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"
assert_contains "${shim_authorization_text}" "required_preflight: AREAFLOW_DATABASE_URL=... bash scripts/smoke-areamatrix-readonly.sh"
assert_contains "${shim_authorization_text}" "post_edit_verification.count:"
assert_contains "${shim_authorization_text}" "post_edit_verification: verify ./task-loop run returns blocked"
assert_contains "${shim_authorization_text}" "post_edit_verification: git status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"
assert_contains "${shim_authorization_text}" "rollback_scope.count:"
assert_contains "${shim_authorization_text}" "rollback_scope: do not write v1 historical execution, progress.json, logs or checkpoints"

status_after="$(file_fingerprint "${status_path}")"
workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_after="$(protected_path_fingerprint)"

if [[ "${status_before}" != "${status_after}" ]]; then
  echo "smoke-areamatrix-readonly: AreaMatrix status export changed unexpectedly: ${status_path}" >&2
  exit 1
fi

if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
  echo "smoke-areamatrix-readonly: AreaMatrix workflow README changed unexpectedly: ${workflow_readme}" >&2
  exit 1
fi

if [[ "${protected_path_fingerprint_before}" != "${protected_path_fingerprint_after}" ]]; then
  echo "smoke-areamatrix-readonly: AreaMatrix protected path fingerprint changed unexpectedly:" >&2
  echo "smoke-areamatrix-readonly: protected_path_fingerprint_before=${protected_path_fingerprint_before}" >&2
  echo "smoke-areamatrix-readonly: protected_path_fingerprint_after=${protected_path_fingerprint_after}" >&2
  exit 1
fi

protected_paths_status_after="$(protected_path_git_status)"
protected_paths_status_hash_after="$(protected_path_status_hash "${protected_paths_status_after}")"
if [[ "${protected_paths_dirty_review_status}" == "accepted" ]]; then
  if [[ "${protected_paths_status_before}" != "${protected_paths_status_after}" || "${protected_paths_status_hash_before}" != "${protected_paths_status_hash_after}" ]]; then
    echo "smoke-areamatrix-readonly: AreaMatrix protected paths changed after reviewed dirty baseline:" >&2
    echo "smoke-areamatrix-readonly: dirty_output_sha256_before=${protected_paths_status_hash_before}" >&2
    echo "smoke-areamatrix-readonly: dirty_output_sha256_after=${protected_paths_status_hash_after}" >&2
    echo "smoke-areamatrix-readonly: protected paths after:" >&2
    echo "${protected_paths_status_after}" >&2
    exit 1
  fi
elif [[ -n "${protected_paths_status_after}" ]]; then
  echo "smoke-areamatrix-readonly: AreaMatrix protected paths changed or are dirty after read-only chain:" >&2
  echo "${protected_paths_status_after}" >&2
  exit 1
fi

echo "smoke-areamatrix-readonly: pass ${project_key} root=${project_root}"
