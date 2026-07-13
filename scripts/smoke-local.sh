#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-local: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

echo "smoke-local: warning: this script can write status projection to the configured project root"
echo "smoke-local: warning: real AreaMatrix project reads require AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1"
echo "smoke-local: warning: set AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1 only for fixture/safe roots"
echo "smoke-local: warning: real AreaMatrix status apply also requires AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1"
echo "smoke-local: warning: use scripts/smoke-fixture.sh for the M0 smoke that never writes real AreaMatrix"

project_key="${AREAFLOW_SMOKE_PROJECT:-areamatrix}"
config_path="${AREAFLOW_SMOKE_CONFIG:-examples/areamatrix/areaflow.yaml}"
real_areamatrix_root="${AREAFLOW_REAL_AREAMATRIX_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
workflow_label="${AREAFLOW_SMOKE_WORKFLOW_VERSION:-smoke-$(date +%Y%m%d%H%M%S)}"
ready_workflow_label="${AREAFLOW_SMOKE_READY_WORKFLOW_VERSION:-${workflow_label}-ready}"
service_api_host="${AREAFLOW_HOST:-127.0.0.1}"
service_api_port="${AREAFLOW_PORT:-3847}"
service_api_url="http://${service_api_host}:${service_api_port}/api/v1"
service_web_url="${AREAFLOW_SMOKE_WEB_URL:-http://127.0.0.1:5174}"

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-local: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-local: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_release_real_100_guardrail() {
  local output="$1"

  assert_contains "${output}" '"readiness_scope": "areaflow_release_preview_only"'
  assert_contains "${output}" '"claim_scope": "areaflow_release_preview_only"'
  assert_contains "${output}" '"not_real_100": true'
  assert_contains "${output}" '"evidence_only": true'
  assert_contains "${output}" '"status_alone_is_not_completion": true'
  assert_contains "${output}" '"release_candidate_decision": "not_release_candidate_evidence"'
  assert_contains "${output}" '"real_100_status": "blocked"'
  assert_contains "${output}" '"real_100_blockers": ['
  assert_contains "${output}" '"package_a_status_projection_apply_provenance_missing"'
  assert_contains "${output}" '"real_areamatrix_execution_cutover_not_proven"'
  assert_contains "${output}" '"real_areamatrix_archive_not_proven"'
  assert_contains "${output}" '"real_areamatrix_shim_retirement_not_proven"'
}

assert_completion_real_100_guardrail() {
  local output="$1"

  assert_contains "${output}" '"readiness_scope": "completion_audit_evidence_only"'
  assert_contains "${output}" '"claim_scope": "completion_audit_evidence_only"'
  assert_contains "${output}" '"not_real_100": true'
  assert_contains "${output}" '"evidence_only": true'
  assert_contains "${output}" '"status_alone_is_not_completion": true'
  assert_contains "${output}" '"release_candidate_decision": "requires_release_candidate_snapshot"'
  assert_contains "${output}" '"real_100_status": "blocked"'
  assert_contains "${output}" '"real_100_blockers": ['
  assert_contains "${output}" '"package_a_status_projection_not_applied"'
  assert_contains "${output}" '"real_areamatrix_execution_cutover_not_proven"'
  assert_contains "${output}" '"real_areamatrix_archive_not_proven"'
  assert_contains "${output}" '"real_areamatrix_shim_retirement_not_proven"'
  assert_contains "${output}" '"release_candidate_snapshot_not_ready"'
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
    echo "smoke-local: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
}

config_project_root() {
  python3 - "${config_path}" <<'PY'
import os
import sys

config_path = sys.argv[1]
in_project = False
project_indent = -1
root = ""

with open(config_path, encoding="utf-8") as handle:
    for raw_line in handle:
        line = raw_line.split("#", 1)[0].rstrip()
        if not line.strip():
            continue
        indent = len(line) - len(line.lstrip(" "))
        stripped = line.strip()
        if stripped == "project:":
            in_project = True
            project_indent = indent
            continue
        if in_project and indent <= project_indent:
            in_project = False
        if in_project and stripped.startswith("root:"):
            root = stripped[len("root:"):].strip()
            if len(root) >= 2 and root[0] in ("'", '"') and root[-1] == root[0]:
                root = root[1:-1]
            break

if not root:
    print(f"smoke-local: project.root missing in {config_path}", file=sys.stderr)
    sys.exit(1)

print(os.path.expanduser(os.path.expandvars(root)))
PY
}

resolve_physical_path() {
  python3 - "$1" <<'PY'
import os
import sys

print(os.path.realpath(os.path.expanduser(os.path.expandvars(sys.argv[1]))))
PY
}

project_root="$(config_project_root)"
resolved_project_root="$(resolve_physical_path "${project_root}")"
resolved_real_areamatrix_root=""
if [[ -n "${real_areamatrix_root}" ]]; then
  resolved_real_areamatrix_root="$(resolve_physical_path "${real_areamatrix_root}")"
fi

if [[ -n "${resolved_real_areamatrix_root}" && "${resolved_project_root}" == "${resolved_real_areamatrix_root}" && "${AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ:-0}" != "1" ]]; then
  echo "smoke-local: refusing to read real AreaMatrix without AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1" >&2
  echo "smoke-local: configured project root: ${project_root}" >&2
  exit 1
fi

echo "smoke-local: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-local: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-local: project import ${project_key} #1"
go run ./cmd/areaflow project import "${project_key}"

echo "smoke-local: project import ${project_key} #2"
go run ./cmd/areaflow project import "${project_key}"

echo "smoke-local: project summary --json"
summary_json="$(go run ./cmd/areaflow project summary "${project_key}" --json)"
assert_contains "${summary_json}" '"config": {'
assert_contains "${summary_json}" '"config_hash": "'
summary_project_root="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["project"]["root"])' <<<"${summary_json}")"
if [[ "$(resolve_physical_path "${summary_project_root}")" != "${resolved_project_root}" ]]; then
  echo "smoke-local: project root changed after registration" >&2
  echo "smoke-local: config project root: ${project_root}" >&2
  echo "smoke-local: summary project root: ${summary_project_root}" >&2
  exit 1
fi

echo "smoke-local: project doctor --json"
doctor_json="$(go run ./cmd/areaflow project doctor "${project_key}" --json)"
assert_contains "${doctor_json}" '"name": "project_config_drift"'
assert_contains "${doctor_json}" '"status": "pass"'

echo "smoke-local: project summary after doctor --json"
summary_after_doctor="$(go run ./cmd/areaflow project summary "${project_key}" --json)"
assert_contains "${summary_after_doctor}" '"config_drift_status": "pass"'

if [[ "${AREAFLOW_SMOKE_ALLOW_STATUS_APPLY:-0}" != "1" ]]; then
  echo "smoke-local: refusing status projection apply without AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1" >&2
  echo "smoke-local: configured project root: ${project_root}" >&2
  exit 1
fi
if [[ -n "${resolved_real_areamatrix_root}" && "${resolved_project_root}" == "${resolved_real_areamatrix_root}" && "${AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY:-0}" != "1" ]]; then
  echo "smoke-local: refusing status projection apply against real AreaMatrix without AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1" >&2
  echo "smoke-local: configured project root: ${project_root}" >&2
  exit 1
fi

echo "smoke-local: project status-projection-authorization ${project_key}"
status_authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
assert_contains "${status_authorization_json}" '"status": "needs_approval"'
assert_contains "${status_authorization_json}" '"project_write_attempted": false'
assert_contains "${status_authorization_json}" '"execution_write_attempted": false'

echo "smoke-local: project status-projection-apply-packet ${project_key}"
status_apply_packet_json="$(
  go run ./cmd/areaflow project status-projection-apply-packet "${project_key}" --json \
    --explicit-approval \
    --approval-actor "smoke-local" \
    --approval-reason "local smoke status projection apply"
)"
assert_contains "${status_apply_packet_json}" '"status": "ready"'
assert_contains "${status_apply_packet_json}" '"apply_command_eligible": true'
assert_contains "${status_apply_packet_json}" '"blockers": []'
expected_before_exists="$(python3 -c 'import json,sys; print(str(json.load(sys.stdin)["packet"]["expected_before_exists"]).lower())' <<<"${status_apply_packet_json}")"
expected_before_size="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["expected_before_size"])' <<<"${status_apply_packet_json}")"
expected_before_sha256="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"].get("expected_before_sha256", ""))' <<<"${status_apply_packet_json}")"
source_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["source_hash"])' <<<"${status_apply_packet_json}")"
schema_uri="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["schema_uri"])' <<<"${status_apply_packet_json}")"
validator_preflight="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["validator_preflight"])' <<<"${status_apply_packet_json}")"
protected_path_check="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["protected_path_check"])' <<<"${status_apply_packet_json}")"
protected_path_fingerprint_sha256="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["protected_path_fingerprint_sha256"])' <<<"${status_apply_packet_json}")"
rollback_action="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["rollback_action"])' <<<"${status_apply_packet_json}")"
accept_preimage_schema="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["accept_preimage_schema"])' <<<"${status_apply_packet_json}")"
status_apply_args=(
  project status-projection-apply "${project_key}"
  --expected-before-exists "${expected_before_exists}"
  --expected-before-size "${expected_before_size}"
  --source-hash "${source_hash}"
  --schema-uri "${schema_uri}"
  --validator-preflight "${validator_preflight}"
  --protected-path-check "${protected_path_check}"
  --protected-path-fingerprint-sha256 "${protected_path_fingerprint_sha256}"
  --rollback-action "${rollback_action}"
  --accept-preimage-schema "${accept_preimage_schema}"
  --explicit-approval
  --approval-actor "smoke-local"
  --approval-reason "local smoke status projection apply"
)
if [[ "${expected_before_exists}" == "true" ]]; then
  status_apply_args+=(--expected-before-sha256 "${expected_before_sha256}")
fi

echo "smoke-local: project status-projection-apply ${project_key}"
export_output="$(go run ./cmd/areaflow "${status_apply_args[@]}")"
assert_contains "${export_output}" '/.areaflow/status.json'
assert_contains "${export_output}" "apply_gate: status=pass decision=go"

echo "smoke-local: project readiness after status export --json"
readiness_json="$(go run ./cmd/areaflow project readiness "${project_key}" --json)"
assert_contains "${readiness_json}" '"key": "status_mirror"'
assert_contains "${readiness_json}" '"message": "status mirror export has been recorded"'

echo "smoke-local: project compatibility --json"
compat_json="$(go run ./cmd/areaflow project compatibility "${project_key}" --json)"
assert_contains "${compat_json}" '"status": "pass"'
assert_contains "${compat_json}" '"command": "./task-loop run"'
assert_contains "${compat_json}" '"mode": "blocked"'

echo "smoke-local: project shim-preview --json"
shim_preview_json="$(go run ./cmd/areaflow project shim-preview "${project_key}" --json)"
assert_contains "${shim_preview_json}" '"mode": "read_only_planning"'
assert_contains "${shim_preview_json}" '"path": "scripts/areaflow_shim.py"'
assert_contains "${shim_preview_json}" '"workflow/versions/**/execution/**"'

echo "smoke-local: project shim-readiness --json"
shim_readiness_json="$(go run ./cmd/areaflow project shim-readiness "${project_key}" --json)"
assert_contains "${shim_readiness_json}" '"status": "blocked"'
assert_contains "${shim_readiness_json}" '"key": "explicit_edit_approval"'
assert_contains "${shim_readiness_json}" '"key": "real_areamatrix_readonly_smoke"'
assert_contains "${shim_readiness_json}" '"key": "real_areamatrix_status_projection_schema"'
assert_contains "${shim_readiness_json}" '"schema_contract": "stable_fallback_projection_v1"'

echo "smoke-local: project shim-authorization --json"
shim_authorization_json="$(go run ./cmd/areaflow project shim-authorization "${project_key}" --json)"
assert_contains "${shim_authorization_json}" '"mode": "read_only_authorization_packet"'
assert_contains "${shim_authorization_json}" '"status": "blocked"'
assert_contains "${shim_authorization_json}" '"path": "scripts/task_loop/console.py"'
assert_contains "${shim_authorization_json}" '"task-loop run forwarding"'
assert_contains "${shim_authorization_json}" '"project_write_attempted": false'
assert_contains "${shim_authorization_json}" '"execution_write_attempted": false'

echo "smoke-local: project shim-apply-packet before evidence --json"
shim_apply_packet_before_json="$(go run ./cmd/areaflow project shim-apply-packet "${project_key}" --json)"
assert_contains "${shim_apply_packet_before_json}" '"mode": "shim_apply_packet_preview_v1"'
assert_contains "${shim_apply_packet_before_json}" '"status": "blocked"'
assert_contains "${shim_apply_packet_before_json}" '"decision": "readiness_blocked"'
assert_contains "${shim_apply_packet_before_json}" '"command_type": "project.shim.apply"'
assert_contains "${shim_apply_packet_before_json}" '"shim_readiness_still_blocked"'
assert_contains "${shim_apply_packet_before_json}" '"command_request_created": false'
assert_contains "${shim_apply_packet_before_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_packet_before_json}" '"execution_write_attempted": false'
assert_contains "${shim_apply_packet_before_json}" '"task_loop_run_forwarded": false'
assert_contains "${shim_apply_packet_before_json}" '"status_projection_written": false'
assert_contains "${shim_apply_packet_before_json}" '"area_matrix_files_modified": false'

echo "smoke-local: project shim-apply-gate before evidence --json"
shim_apply_gate_before_json="$(go run ./cmd/areaflow project shim-apply-gate "${project_key}" --json)"
assert_contains "${shim_apply_gate_before_json}" '"mode": "shim_apply_gate_v1"'
assert_contains "${shim_apply_gate_before_json}" '"status": "blocked"'
assert_contains "${shim_apply_gate_before_json}" '"decision": "no_go"'
assert_contains "${shim_apply_gate_before_json}" '"apply_command_eligible": false'
assert_contains "${shim_apply_gate_before_json}" '"explicit_shim_apply_approval_missing"'
assert_contains "${shim_apply_gate_before_json}" '"project_write_attempted": false'
assert_contains "${shim_apply_gate_before_json}" '"area_matrix_files_modified": false'

if [[ -n "${resolved_real_areamatrix_root}" && "${resolved_project_root}" == "${resolved_real_areamatrix_root}" ]]; then
  echo "smoke-local: skipping fixture shim readiness evidence against real AreaMatrix"
else
  echo "smoke-local: project shim-readiness-evidence real_areamatrix_readonly_smoke"
  readonly_evidence_json="$(go run ./cmd/areaflow project shim-readiness-evidence "${project_key}" \
    --key real_areamatrix_readonly_smoke \
    --status pass \
    --summary "smoke-local fixture proves read-only smoke evidence recording" \
    --evidence-uri "scripts/smoke-local.sh" \
    --json)"
  assert_contains "${readonly_evidence_json}" '"evidence_key": "real_areamatrix_readonly_smoke"'
  assert_contains "${readonly_evidence_json}" '"status": "recorded"'
  assert_contains "${readonly_evidence_json}" '"project_write_attempted": false'
  assert_contains "${readonly_evidence_json}" '"execution_write_attempted": false'

  echo "smoke-local: project shim-readiness-evidence real_areamatrix_status_projection_schema"
  status_schema_evidence_json="$(go run ./cmd/areaflow project shim-readiness-evidence "${project_key}" \
    --key real_areamatrix_status_projection_schema \
    --status pass \
    --summary "smoke-local fixture proves status projection schema evidence recording" \
    --evidence-uri "schemas/status-projection.schema.json" \
    --json)"
  assert_contains "${status_schema_evidence_json}" '"evidence_key": "real_areamatrix_status_projection_schema"'
  assert_contains "${status_schema_evidence_json}" '"status": "recorded"'
  assert_contains "${status_schema_evidence_json}" '"project_write_attempted": false'
  assert_contains "${status_schema_evidence_json}" '"execution_write_attempted": false'

  echo "smoke-local: project shim-readiness-evidence areamatrix_dirty_worktree_review"
  dirty_review_evidence_json="$(go run ./cmd/areaflow project shim-readiness-evidence "${project_key}" \
    --key areamatrix_dirty_worktree_review \
    --status pass \
    --summary "smoke-local fixture proves dirty worktree review evidence recording" \
    --evidence-uri "scripts/smoke-local.sh" \
    --json)"
  assert_contains "${dirty_review_evidence_json}" '"evidence_key": "areamatrix_dirty_worktree_review"'
  assert_contains "${dirty_review_evidence_json}" '"status": "recorded"'
  assert_contains "${dirty_review_evidence_json}" '"project_write_attempted": false'
  assert_contains "${dirty_review_evidence_json}" '"execution_write_attempted": false'

  echo "smoke-local: project shim-readiness after evidence --json"
  shim_readiness_after_evidence_json="$(go run ./cmd/areaflow project shim-readiness "${project_key}" --json)"
  assert_contains "${shim_readiness_after_evidence_json}" '"status": "blocked"'
  assert_contains "${shim_readiness_after_evidence_json}" '"evidence_recorded": true'
  assert_contains "${shim_readiness_after_evidence_json}" '"evidence_status": "pass"'
  assert_contains "${shim_readiness_after_evidence_json}" '"key": "explicit_edit_approval"'

  echo "smoke-local: project shim-apply-packet after evidence --json"
  shim_apply_packet_after_json="$(go run ./cmd/areaflow project shim-apply-packet "${project_key}" \
    --explicit-approval \
    --approval-id shim-approval-1 \
    --approval-actor smoke-local \
    --approval-reason "smoke-local shim apply packet review" \
    --status-projection-packet-id "${project_key}:status_projection_apply_packet:status-packet-1" \
    --status-projection-gate-id "${project_key}:status_projection_apply_gate:status-gate-1" \
    --read-only-smoke-evidence-id "${project_key}:real_areamatrix_readonly_smoke:smoke-evidence-1" \
    --dirty-worktree-review-id "${project_key}:areamatrix_dirty_worktree_review:dirty-review-1" \
    --protected-path-fingerprint-id "${project_key}:protected_path_fingerprint:protected-fingerprint-1" \
    --rollback-plan-id "${project_key}:rollback_plan:rollback-plan-1" \
    --json)"
  assert_contains "${shim_apply_packet_after_json}" '"status": "ready"'
  assert_contains "${shim_apply_packet_after_json}" '"decision": "ready_for_future_apply_command"'
  assert_contains "${shim_apply_packet_after_json}" '"apply_command_eligible": true'
  assert_contains "${shim_apply_packet_after_json}" '"would_create_command_request_after_apply_command": true'
  assert_contains "${shim_apply_packet_after_json}" '"command_request_created": false'
  assert_contains "${shim_apply_packet_after_json}" '"project_write_attempted": false'
  assert_contains "${shim_apply_packet_after_json}" '"execution_write_attempted": false'
  assert_contains "${shim_apply_packet_after_json}" '"task_loop_run_forwarded": false'
  assert_contains "${shim_apply_packet_after_json}" '"status_projection_written": false'
  assert_contains "${shim_apply_packet_after_json}" '"area_matrix_files_modified": false'
fi

echo "smoke-local: workflow version create ${workflow_label}"
version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${workflow_label}" --json)"
assert_contains "${version_json}" '"import_mode": "authored"'
assert_contains "${version_json}" '"profile_binding": {'

echo "smoke-local: workflow version ensure-skeleton ${workflow_label}"
skeleton_json="$(go run ./cmd/areaflow workflow version ensure-skeleton "${project_key}" "${workflow_label}" --json)"
assert_contains "${skeleton_json}" '"links": ['
assert_contains "${skeleton_json}" '"relation_type": "derives_from"'

echo "smoke-local: workflow gate profile_binding_drift"
profile_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" profile_binding_drift --json)"
assert_contains "${profile_gate_json}" '"gate_name": "profile_binding_drift"'
assert_contains "${profile_gate_json}" '"status": "pass"'

echo "smoke-local: workflow gate promotion_preview"
promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" promotion_preview --json)"
assert_contains "${promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${promotion_gate_json}" '"status": "fail"'
assert_contains "${promotion_gate_json}" 'placeholder-only'

echo "smoke-local: workflow transition preview ${workflow_label}"
transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${workflow_label}" --json)"
assert_contains "${transition_json}" '"status": "blocked"'
assert_contains "${transition_json}" 'latest promotion_preview gate status is fail'

echo "smoke-local: workflow approval record rejected ${workflow_label}"
approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${workflow_label}" --decision rejected --reason "smoke blocked transition preview" --json)"
assert_contains "${approval_json}" '"decision": "rejected"'
assert_contains "${approval_json}" '"transition_status": "blocked"'

echo "smoke-local: workflow gate approval_gate"
approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" approval_gate --json)"
assert_contains "${approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${approval_gate_json}" '"status": "blocked"'
assert_contains "${approval_gate_json}" 'latest approval decision is rejected'
assert_contains "${approval_gate_json}" 'linked transition preview status is blocked'

echo "smoke-local: workflow gate live_mapping_gate"
live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" live_mapping_gate --json)"
assert_contains "${live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${live_mapping_gate_json}" '"status": "blocked"'
assert_contains "${live_mapping_gate_json}" 'approval_gate has not passed'
assert_contains "${live_mapping_gate_json}" 'transition preview status is blocked'
assert_contains "${live_mapping_gate_json}" 'promotion_preview gate status is fail'
assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-local: project cutover-readiness ${workflow_label}"
cutover_json="$(go run ./cmd/areaflow project cutover-readiness "${project_key}" --version "${workflow_label}" --json)"
assert_contains "${cutover_json}" '"status": "blocked"'
assert_contains "${cutover_json}" '"name": "v0.4-cutover-readiness"'
assert_contains "${cutover_json}" '"key": "status_mirror"'
assert_contains "${cutover_json}" '"message": "status mirror export has been recorded"'
assert_contains "${cutover_json}" 'approval_gate is blocked'
assert_contains "${cutover_json}" 'live_mapping_gate is blocked'

echo "smoke-local: workflow gate cutover_readiness_gate"
cutover_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" cutover_readiness_gate --json)"
assert_contains "${cutover_gate_json}" '"gate_name": "cutover_readiness_gate"'
assert_contains "${cutover_gate_json}" '"status": "blocked"'
assert_contains "${cutover_gate_json}" '"phase_gate_status": "blocked"'
assert_contains "${cutover_gate_json}" '"cutover_apply_attempted": false'
assert_contains "${cutover_gate_json}" '"execution_write_attempted": false'

echo "smoke-local: workflow ready-path version create ${ready_workflow_label}"
ready_version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${ready_workflow_label}" --json)"
assert_contains "${ready_version_json}" '"import_mode": "authored"'

echo "smoke-local: workflow ready-path mark queue ${ready_workflow_label}"
ready_queue_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${ready_workflow_label}" --stage queue --item-type queue_candidate --reason "smoke ready path queue" --json)"
assert_contains "${ready_queue_json}" '"status": "ready"'
assert_contains "${ready_queue_json}" '"artifact_type": "workflow_item_ready_marker"'
assert_contains "${ready_queue_json}" '"source_hash": "'

echo "smoke-local: workflow ready-path mark promotion_preview ${ready_workflow_label}"
ready_promotion_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${ready_workflow_label}" --stage promotion_preview --item-type promotion_preview --reason "smoke ready path promotion" --json)"
assert_contains "${ready_promotion_json}" '"status": "ready"'
assert_contains "${ready_promotion_json}" '"artifact_type": "workflow_item_ready_marker"'
assert_contains "${ready_promotion_json}" '"source_hash": "'

echo "smoke-local: workflow ready-path gate promotion_preview"
ready_promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" promotion_preview --json)"
assert_contains "${ready_promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${ready_promotion_gate_json}" '"status": "pass"'
assert_contains "${ready_promotion_gate_json}" '"placeholder_items": []'

echo "smoke-local: workflow ready-path transition preview ${ready_workflow_label}"
ready_transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${ready_workflow_label}" --json)"
assert_contains "${ready_transition_json}" '"status": "ready"'
assert_contains "${ready_transition_json}" '"required_gate_name": "promotion_preview"'

echo "smoke-local: workflow ready-path approval approved ${ready_workflow_label}"
ready_approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${ready_workflow_label}" --decision approved --reason "smoke ready path approval" --json)"
assert_contains "${ready_approval_json}" '"decision": "approved"'
assert_contains "${ready_approval_json}" '"transition_status": "ready"'

echo "smoke-local: workflow ready-path gate approval_gate"
ready_approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" approval_gate --json)"
assert_contains "${ready_approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${ready_approval_gate_json}" '"status": "pass"'

echo "smoke-local: workflow ready-path gate live_mapping_gate"
ready_live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" live_mapping_gate --json)"
assert_contains "${ready_live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${ready_live_mapping_gate_json}" '"status": "pass"'
assert_contains "${ready_live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-local: project ready-path cutover-readiness ${ready_workflow_label}"
ready_cutover_json="$(go run ./cmd/areaflow project cutover-readiness "${project_key}" --version "${ready_workflow_label}" --json)"
assert_contains "${ready_cutover_json}" '"status": "pass"'
assert_contains "${ready_cutover_json}" '"name": "v0.4-cutover-readiness"'
assert_contains "${ready_cutover_json}" '"key": "verification_bundle"'
assert_contains "${ready_cutover_json}" '"message": "v0.2 verification bundle passed"'
assert_contains "${ready_cutover_json}" '"key": "compatibility_contract"'
assert_contains "${ready_cutover_json}" '"message": "compatibility contract is ready"'
assert_contains "${ready_cutover_json}" '"key": "approval_gate"'
assert_contains "${ready_cutover_json}" '"message": "approval_gate passed"'
assert_contains "${ready_cutover_json}" '"key": "live_mapping_gate"'
assert_contains "${ready_cutover_json}" '"message": "live_mapping_gate passed"'

echo "smoke-local: workflow ready-path gate cutover_readiness_gate"
ready_cutover_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" cutover_readiness_gate --json)"
assert_contains "${ready_cutover_gate_json}" '"gate_name": "cutover_readiness_gate"'
assert_contains "${ready_cutover_gate_json}" '"status": "pass"'
assert_contains "${ready_cutover_gate_json}" '"phase_gate_status": "pass"'
assert_contains "${ready_cutover_gate_json}" '"cutover_apply_attempted": false'
assert_contains "${ready_cutover_gate_json}" '"execution_write_attempted": false'

echo "smoke-local: runner preview ${ready_workflow_label}"
runner_json="$(go run ./cmd/areaflow run preview "${project_key}" "${ready_workflow_label}" --json)"
assert_contains "${runner_json}" '"run_type": "runner_preview"'
assert_contains "${runner_json}" '"status": "passed"'
assert_contains "${runner_json}" '"dry_run": true'
assert_contains "${runner_json}" '"task_kind": "workflow_item_preview"'
assert_contains "${runner_json}" '"attempt_kind": "copy"'
assert_contains "${runner_json}" '"attempt_kind": "verify"'
assert_contains "${runner_json}" '"artifact_type": "runner_preview_report"'
assert_contains "${runner_json}" '"sha256": "'
assert_contains "${runner_json}" '"writes_attempted": false'
assert_contains "${runner_json}" '"commands_run": false'
runner_run_id="$(awk '
  /"run": \{/ { in_run = 1; next }
  in_run && /"id":/ {
    gsub(/[^0-9]/, "", $0)
    print
    exit
  }
  in_run && /^[[:space:]]*\}/ { in_run = 0 }
' <<<"${runner_json}")"
if [[ -z "${runner_run_id}" ]]; then
  echo "smoke-local: expected runner preview JSON to include run.id" >&2
  exit 1
fi
runner_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -v "run_id=${runner_run_id}" -At <<'SQL'
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
  echo "smoke-local: expected runner.preview command response safety facts, got ${runner_command_count}" >&2
  exit 1
fi

echo "smoke-local: runner preview high-risk blocked"
assert_fails_contains "runner preview blocked: risk_gate" go run ./cmd/areaflow run preview "${project_key}" "${ready_workflow_label}" --risk-level high --json

echo "smoke-local: worker register smoke-worker"
worker_json="$(go run ./cmd/areaflow worker register "${project_key}" --worker-key "${ready_workflow_label}-worker" --capability read_project --capability write_artifacts --json)"
assert_contains "${worker_json}" '"worker_key": "'"${ready_workflow_label}"'-worker"'
assert_contains "${worker_json}" '"status": "online"'
assert_contains "${worker_json}" '"read_project"'
assert_contains "${worker_json}" '"write_artifacts"'

echo "smoke-local: worker run-once ${ready_workflow_label}"
worker_run_json="$(go run ./cmd/areaflow worker run-once "${project_key}" "${ready_workflow_label}-worker" --run-id "${runner_run_id}" --capability read_project --capability write_artifacts --json)"
assert_contains "${worker_run_json}" '"claimed": true'
assert_contains "${worker_run_json}" '"status": "completed"'
assert_contains "${worker_run_json}" '"task_kind": "workflow_item_preview"'
assert_contains "${worker_run_json}" '"attempt_kind": "worker_run_once"'
assert_contains "${worker_run_json}" '"dry_run": true'
assert_contains "${worker_run_json}" '"artifact_type": "worker_run_once_report"'
assert_contains "${worker_run_json}" '"writes_attempted": false'
assert_contains "${worker_run_json}" '"commands_run": false'

echo "smoke-local: worker capability denied"
go run ./cmd/areaflow worker register "${project_key}" --worker-key "${ready_workflow_label}-readonly" --capability read_project --json >/dev/null
assert_fails_contains "worker capability denied: missing write_artifacts" go run ./cmd/areaflow worker run-once "${project_key}" "${ready_workflow_label}-readonly" --run-id "${runner_run_id}" --capability read_project --capability write_artifacts --json

echo "smoke-local: worker pool summary"
worker_pool_json="$(go run ./cmd/areaflow worker pool-summary --json)"
assert_contains "${worker_pool_json}" '"total_projects":'
assert_contains "${worker_pool_json}" '"total_workers":'
assert_contains "${worker_pool_json}" '"total_online_workers":'
assert_contains "${worker_pool_json}" '"total_queued_tasks":'
assert_contains "${worker_pool_json}" '"key": "'"${project_key}"'"'
assert_contains "${worker_pool_json}" '"worker_types": ['
assert_contains "${worker_pool_json}" '"local_host"'
assert_contains "${worker_pool_json}" '"capabilities": ['
assert_contains "${worker_pool_json}" '"read_project"'
assert_contains "${worker_pool_json}" '"write_artifacts"'
assert_contains "${worker_pool_json}" '"scheduling": {'
assert_contains "${worker_pool_json}" '"priority": 100'
assert_contains "${worker_pool_json}" '"max_parallel_tasks": 1'
assert_contains "${worker_pool_json}" '"agent_role": "local_worker"'
assert_contains "${worker_pool_json}" '"engine_profile": "codex-cli"'
assert_contains "${worker_pool_json}" '"role": {'
assert_contains "${worker_pool_json}" '"matched": true'
assert_contains "${worker_pool_json}" '"status": "ready"'
assert_contains "${worker_pool_json}" '"engine": {'
assert_contains "${worker_pool_json}" '"profile_id": "codex-cli"'
assert_contains "${worker_pool_json}" '"enabled": false'
assert_contains "${worker_pool_json}" '"secret_ref": "none"'
assert_contains "${worker_pool_json}" '"secret_required": false'
assert_contains "${worker_pool_json}" '"secret_ready": true'
assert_contains "${worker_pool_json}" '"engine_profile_disabled"'
assert_contains "${worker_pool_json}" '"resources": {'
assert_contains "${worker_pool_json}" '"max_active_leases": 1'
assert_contains "${worker_pool_json}" '"max_queued_tasks": 20'

echo "smoke-local: worker schedule preview"
schedule_json="$(go run ./cmd/areaflow worker schedule-preview --json)"
assert_contains "${schedule_json}" '"policy": {'
assert_contains "${schedule_json}" '"strategy": "default_fifo"'
assert_contains "${schedule_json}" '"slot_strategy": "min_online_workers_and_project_parallelism_minus_active_leases"'
assert_contains "${schedule_json}" '"dry_run_only": true'
assert_contains "${schedule_json}" '"key": "'"${project_key}"'"'
assert_contains "${schedule_json}" '"priority": 100'
assert_contains "${schedule_json}" '"max_parallel": 1'
assert_contains "${schedule_json}" '"agent_role": "local_worker"'
assert_contains "${schedule_json}" '"required_capabilities": ['
assert_contains "${schedule_json}" '"read_project"'
assert_contains "${schedule_json}" '"write_artifacts"'
assert_contains "${schedule_json}" '"engine_profile": "codex-cli"'
assert_contains "${schedule_json}" '"engine_profile_disabled"'
assert_contains "${schedule_json}" '"available_slots":'
assert_contains "${schedule_json}" '"recommended": false'
assert_contains "${schedule_json}" '"next_action": "idle"'

echo "smoke-local: service status"
service_json="$(go run ./cmd/areaflow service status --web-url "${service_web_url}" --json)"
assert_contains "${service_json}" '"status": "ready"'
assert_contains "${service_json}" '"mode": "local_service"'
assert_contains "${service_json}" '"api": {'
assert_contains "${service_json}" '"database": {'
assert_contains "${service_json}" '"worker_pool": {'
assert_contains "${service_json}" '"dashboard": {'
assert_contains "${service_json}" '"api_url": "'"${service_api_url}"'"'
assert_contains "${service_json}" '"url": "'"${service_web_url}"'"'
assert_contains "${service_json}" '"total_projects":'
assert_contains "${service_json}" '"total_workers":'
assert_contains "${service_json}" '"total_queued_tasks":'
assert_contains "${service_json}" '"observe_api"'
assert_contains "${service_json}" '"open_web_dashboard"'
assert_contains "${service_json}" '"maintain_second_database"'
assert_contains "${service_json}" '"run_workflow_directly"'

echo "smoke-local: desktop service-control-gate"
desktop_service_control_json="$(go run ./cmd/areaflow desktop service-control-gate --json)"
assert_contains "${desktop_service_control_json}" '"status": "blocked"'
assert_contains "${desktop_service_control_json}" '"mode": "read_only_desktop_service_control_gate"'
assert_contains "${desktop_service_control_json}" '"key": "open_dashboard"'
assert_contains "${desktop_service_control_json}" '"default_ui_state": "enabled_link"'
assert_contains "${desktop_service_control_json}" '"key": "start_service"'
assert_contains "${desktop_service_control_json}" '"key": "stop_service"'
assert_contains "${desktop_service_control_json}" '"key": "restart_service"'
assert_contains "${desktop_service_control_json}" '"desktop_service_control_not_open"'
assert_contains "${desktop_service_control_json}" '"process_supervision_contract_not_defined"'
assert_contains "${desktop_service_control_json}" '"service_stop_requires_drain_policy"'
assert_contains "${desktop_service_control_json}" '"process_control_attempted": false'
assert_contains "${desktop_service_control_json}" '"command_created": false'
assert_contains "${desktop_service_control_json}" '"approval_created": false'
assert_contains "${desktop_service_control_json}" '"audit_event_written": false'
assert_contains "${desktop_service_control_json}" '"worker_scheduled": false'
assert_contains "${desktop_service_control_json}" '"workflow_execution_started": false'
assert_contains "${desktop_service_control_json}" '"project_write_attempted": false'
assert_contains "${desktop_service_control_json}" '"secrets_resolved": false'

echo "smoke-local: desktop notification-gate"
desktop_notification_json="$(go run ./cmd/areaflow desktop notification-gate --json)"
assert_contains "${desktop_notification_json}" '"status": "blocked"'
assert_contains "${desktop_notification_json}" '"mode": "read_only_desktop_notification_gate"'
assert_contains "${desktop_notification_json}" '"key": "observe_event_stream"'
assert_contains "${desktop_notification_json}" '"default_ui_state": "available_read_only"'
assert_contains "${desktop_notification_json}" '"key": "enable_system_notifications"'
assert_contains "${desktop_notification_json}" '"key": "approval_needed_notifications"'
assert_contains "${desktop_notification_json}" '"key": "run_failure_notifications"'
assert_contains "${desktop_notification_json}" '"notification_permission_flow_not_implemented"'
assert_contains "${desktop_notification_json}" '"notification_redaction_contract_not_defined"'
assert_contains "${desktop_notification_json}" '"system_notifications_not_open"'
assert_contains "${desktop_notification_json}" '"event_stream_opened": false'
assert_contains "${desktop_notification_json}" '"notification_requested": false'
assert_contains "${desktop_notification_json}" '"command_created": false'
assert_contains "${desktop_notification_json}" '"approval_created": false'
assert_contains "${desktop_notification_json}" '"audit_event_written": false'
assert_contains "${desktop_notification_json}" '"worker_scheduled": false'
assert_contains "${desktop_notification_json}" '"workflow_execution_started": false'
assert_contains "${desktop_notification_json}" '"project_write_attempted": false'
assert_contains "${desktop_notification_json}" '"secrets_resolved": false'

echo "smoke-local: desktop tray-menu-gate"
desktop_tray_json="$(go run ./cmd/areaflow desktop tray-menu-gate --json)"
assert_contains "${desktop_tray_json}" '"status": "blocked"'
assert_contains "${desktop_tray_json}" '"mode": "read_only_desktop_tray_menu_gate"'
assert_contains "${desktop_tray_json}" '"key": "open_dashboard"'
assert_contains "${desktop_tray_json}" '"default_ui_state": "enabled_link"'
assert_contains "${desktop_tray_json}" '"key": "show_service_status"'
assert_contains "${desktop_tray_json}" '"key": "show_recent_events"'
assert_contains "${desktop_tray_json}" '"key": "start_service"'
assert_contains "${desktop_tray_json}" '"key": "enable_notifications"'
assert_contains "${desktop_tray_json}" '"service_control_gate_blocked"'
assert_contains "${desktop_tray_json}" '"tray_service_control_not_open"'
assert_contains "${desktop_tray_json}" '"notification_gate_blocked"'
assert_contains "${desktop_tray_json}" '"tray_notification_action_not_open"'
assert_contains "${desktop_tray_json}" '"tray_menu_created": false'
assert_contains "${desktop_tray_json}" '"os_integration_requested": false'
assert_contains "${desktop_tray_json}" '"service_control_attempted": false'
assert_contains "${desktop_tray_json}" '"notification_requested": false'
assert_contains "${desktop_tray_json}" '"command_created": false'
assert_contains "${desktop_tray_json}" '"approval_created": false'
assert_contains "${desktop_tray_json}" '"audit_event_written": false'
assert_contains "${desktop_tray_json}" '"worker_scheduled": false'
assert_contains "${desktop_tray_json}" '"workflow_execution_started": false'
assert_contains "${desktop_tray_json}" '"project_write_attempted": false'
assert_contains "${desktop_tray_json}" '"secrets_resolved": false'

echo "smoke-local: ops migration ledger readiness"
migration_ledger_json="$(go run ./cmd/areaflow ops migration-ledger-readiness --json)"
assert_contains "${migration_ledger_json}" '"status": "ready"'
assert_contains "${migration_ledger_json}" '"mode": "read_only_migration_ledger_readiness"'
assert_contains "${migration_ledger_json}" '"name": "000001_v0_1_core.sql"'
assert_contains "${migration_ledger_json}" '"name": "000011_v1_migration_ledger.sql"'
assert_contains "${migration_ledger_json}" '"applied": true'
assert_contains "${migration_ledger_json}" '"applied_count": 11'
assert_contains "${migration_ledger_json}" '"pending_count": 0'
assert_contains "${migration_ledger_json}" '"schema_migrations_table_present": true'
assert_contains "${migration_ledger_json}" '"full_ledger_table_present": true'
assert_contains "${migration_ledger_json}" '"preflight_apply_verify_remediation_ready": true'
assert_contains "${migration_ledger_json}" '"phase": "preflight"'
assert_contains "${migration_ledger_json}" '"phase": "apply"'
assert_contains "${migration_ledger_json}" '"phase": "verify"'
assert_contains "${migration_ledger_json}" '"phase": "remediation"'
assert_contains "${migration_ledger_json}" '"migration_runner_recorded": true'
assert_contains "${migration_ledger_json}" '"read_only": true'
assert_contains "${migration_ledger_json}" '"migration_apply_attempted": false'
assert_contains "${migration_ledger_json}" '"database_write_attempted": false'
assert_contains "${migration_ledger_json}" '"destructive_rollback_attempted": false'
assert_contains "${migration_ledger_json}" '"project_write_attempted": false'
assert_contains "${migration_ledger_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${migration_ledger_json}" '"apply_migration"'
assert_contains "${migration_ledger_json}" '"rollback_database"'
assert_contains "${migration_ledger_json}" '"write_project_files"'

echo "smoke-local: support bundle preview"
support_bundle_json="$(go run ./cmd/areaflow support bundle-preview --json)"
assert_contains "${support_bundle_json}" '"status": "ready"'
assert_contains "${support_bundle_json}" '"mode": "metadata_only_support_bundle_preview"'
assert_contains "${support_bundle_json}" '"bundle_id": "support-bundle-preview-v1"'
assert_contains "${support_bundle_json}" '"scope": "local_v1_metadata_only"'
assert_contains "${support_bundle_json}" '"key": "'"${project_key}"'"'
assert_contains "${support_bundle_json}" '"kind": "project_root_reference"'
assert_contains "${support_bundle_json}" '"secret_values"'
assert_contains "${support_bundle_json}" '"prompt_text"'
assert_contains "${support_bundle_json}" '"raw_artifact_contents"'
assert_contains "${support_bundle_json}" '"user_file_contents"'
assert_contains "${support_bundle_json}" '"export_support_bundle"'
assert_contains "${support_bundle_json}" '"copy_project_files"'
assert_contains "${support_bundle_json}" '"write_database"'
assert_contains "${support_bundle_json}" '"write_project_files"'
assert_contains "${support_bundle_json}" '"read_only": true'
assert_contains "${support_bundle_json}" '"metadata_only": true'
assert_contains "${support_bundle_json}" '"export_open": false'
assert_contains "${support_bundle_json}" '"secret_values_included": false'
assert_contains "${support_bundle_json}" '"api_token_values_included": false'
assert_contains "${support_bundle_json}" '"prompt_text_included": false'
assert_contains "${support_bundle_json}" '"user_file_contents_included": false'
assert_contains "${support_bundle_json}" '"raw_artifact_contents_included": false'
assert_contains "${support_bundle_json}" '"unredacted_logs_included": false'
assert_contains "${support_bundle_json}" '"managed_project_files_copied": false'
assert_contains "${support_bundle_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${support_bundle_json}" '"remote_upload_attempted": false'
assert_contains "${support_bundle_json}" '"database_write_attempted": false'

echo "smoke-local: ops readiness"
ops_readiness_json="$(go run ./cmd/areaflow ops readiness --json)"
assert_contains "${ops_readiness_json}" '"status": "needs_attention"'
assert_contains "${ops_readiness_json}" '"mode": "read_only_operations_readiness"'
assert_contains "${ops_readiness_json}" '"key": "install_migrate_start_register_smoke"'
assert_contains "${ops_readiness_json}" '"key": "local_service_status"'
assert_contains "${ops_readiness_json}" '"key": "metadata_only_support_bundle_preview"'
assert_contains "${ops_readiness_json}" '"key": "migration_ledger_readiness"'
assert_contains "${ops_readiness_json}" '"key": "local_only_telemetry_default"'
assert_contains "${ops_readiness_json}" '"key": "managed_ops_deferred"'
assert_contains "${ops_readiness_json}" '"support_export_status": "deferred_v1x"'
assert_contains "${ops_readiness_json}" '"telemetry_default": "local_only"'
assert_contains "${ops_readiness_json}" '"managed_ops_status": "deferred_v1x"'
assert_contains "${ops_readiness_json}" '"blocked_by": ['
assert_contains "${ops_readiness_json}" '"fresh_local_ops_smoke_missing"'
assert_contains "${ops_readiness_json}" '"read_only": true'
assert_contains "${ops_readiness_json}" '"support_bundle_exported": false'
assert_contains "${ops_readiness_json}" '"support_bundle_metadata_only": true'
assert_contains "${ops_readiness_json}" '"remote_telemetry_enabled": false'
assert_contains "${ops_readiness_json}" '"managed_upgrade_attempted": false'
assert_contains "${ops_readiness_json}" '"destructive_rollback_attempted": false'
assert_contains "${ops_readiness_json}" '"service_process_control_attempted": false'
assert_contains "${ops_readiness_json}" '"database_write_attempted": false'
assert_contains "${ops_readiness_json}" '"project_write_attempted": false'
assert_contains "${ops_readiness_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${ops_readiness_json}" '"export_support_bundle"'
assert_contains "${ops_readiness_json}" '"upload_telemetry"'
assert_contains "${ops_readiness_json}" '"run_managed_upgrade"'
assert_contains "${ops_readiness_json}" '"rollback_database"'

echo "smoke-local: backup manifest"
backup_json="$(go run ./cmd/areaflow backup manifest --json)"
assert_contains "${backup_json}" '"status": "ready"'
assert_contains "${backup_json}" '"mode": "read_only_manifest"'
assert_contains "${backup_json}" '"schema_version": 1'
assert_contains "${backup_json}" '"manifest_hash": "'
assert_contains "${backup_json}" '"table_counts": ['
assert_contains "${backup_json}" '"table": "projects"'
assert_contains "${backup_json}" '"table": "artifacts"'
assert_contains "${backup_json}" '"projects": ['
assert_contains "${backup_json}" '"key": "'"${project_key}"'"'
assert_contains "${backup_json}" '"artifact_count":'
assert_contains "${backup_json}" '"artifacts": ['
assert_contains "${backup_json}" '"export_postgres_metadata"'
assert_contains "${backup_json}" '"export_artifact_metadata"'
assert_contains "${backup_json}" '"restore_database"'
assert_contains "${backup_json}" '"read_artifact_contents"'

echo "smoke-local: restore plan"
restore_plan_json="$(go run ./cmd/areaflow backup restore-plan --json)"
assert_contains "${restore_plan_json}" '"status": "needs_attention"'
assert_contains "${restore_plan_json}" '"mode": "read_only_restore_plan"'
assert_contains "${restore_plan_json}" '"schema_version": 1'
assert_contains "${restore_plan_json}" '"manifest_hash": "'
assert_contains "${restore_plan_json}" '"projects": ['
assert_contains "${restore_plan_json}" '"key": "'"${project_key}"'"'
assert_contains "${restore_plan_json}" '"key": "manifest_shape"'
assert_contains "${restore_plan_json}" '"key": "project_inventory"'
assert_contains "${restore_plan_json}" '"key": "artifact_inventory"'
assert_contains "${restore_plan_json}" '"key": "dry_run_guardrails"'
assert_contains "${restore_plan_json}" '"key": "artifact_integrity:'"${project_key}"'"'
assert_contains "${restore_plan_json}" '"status": "ready"'
assert_contains "${restore_plan_json}" '"status": "needs_attention"'
assert_contains "${restore_plan_json}" '"generate_restore_plan"'
assert_contains "${restore_plan_json}" '"restore_database"'
assert_contains "${restore_plan_json}" '"apply_restore"'
assert_contains "${restore_plan_json}" '"referenced_artifacts":'
assert_contains "${restore_plan_json}" '"skipped_artifacts":'

echo "smoke-local: audit coverage"
audit_json="$(go run ./cmd/areaflow audit coverage --project "${project_key}" --json)"
assert_contains "${audit_json}" '"status": "warn"'
assert_contains "${audit_json}" '"mode": "read_only_audit_coverage"'
assert_contains "${audit_json}" '"scope": "project"'
assert_contains "${audit_json}" '"project_key": "'"${project_key}"'"'
assert_contains "${audit_json}" '"total_audit_events":'
assert_contains "${audit_json}" '"covered_requirements":'
assert_contains "${audit_json}" '"gap_requirements":'
assert_contains "${audit_json}" '"key": "project_registration"'
assert_contains "${audit_json}" '"key": "status_mirror_write"'
assert_contains "${audit_json}" '"key": "workflow_authoring"'
assert_contains "${audit_json}" '"key": "approval_decision"'
assert_contains "${audit_json}" '"key": "runner_preview"'
assert_contains "${audit_json}" '"key": "worker_registration"'
assert_contains "${audit_json}" '"key": "worker_capability_denial"'
assert_contains "${audit_json}" '"key": "secret_resolution"'
assert_contains "${audit_json}" '"key": "permission_change"'
assert_contains "${audit_json}" '"action": "project.upsert"'
assert_contains "${audit_json}" '"action": "worker.run_once"'
assert_contains "${audit_json}" '"missing_actions": ['
assert_contains "${audit_json}" '"secret.resolve"'
assert_contains "${audit_json}" '"permission.change"'

echo "smoke-local: permission policy doctor"
permission_json="$(go run ./cmd/areaflow permissions doctor "${project_key}" --json)"
assert_contains "${permission_json}" '"status": "pass"'
assert_contains "${permission_json}" '"mode": "read_only_permission_policy_doctor"'
assert_contains "${permission_json}" '"key": "'"${project_key}"'"'
assert_contains "${permission_json}" '"key": "project_config"'
assert_contains "${permission_json}" '"key": "default_read_only"'
assert_contains "${permission_json}" '"key": "status_export_write"'
assert_contains "${permission_json}" '"key": "dangerous_write_denies"'
assert_contains "${permission_json}" '"key": "command_policy"'
assert_contains "${permission_json}" '"key": "secret_policy"'
assert_contains "${permission_json}" '"key": "network_policy"'
assert_contains "${permission_json}" '"key": "worker_capability_policy"'
assert_contains "${permission_json}" '"key": "git_policy"'
assert_contains "${permission_json}" '"key": "permission_audit_readiness"'
assert_contains "${permission_json}" '"path": ".areaflow/status.json"'
assert_contains "${permission_json}" '"doctor_writes_audit": false'

echo "smoke-local: artifact integrity"
artifact_integrity_json="$(go run ./cmd/areaflow artifact integrity "${project_key}" --json)"
assert_contains "${artifact_integrity_json}" '"status": "warn"'
assert_contains "${artifact_integrity_json}" '"mode": "read_only_artifact_integrity"'
assert_contains "${artifact_integrity_json}" '"key": "'"${project_key}"'"'
assert_contains "${artifact_integrity_json}" '"checked_artifacts":'
assert_contains "${artifact_integrity_json}" '"passed_artifacts":'
assert_contains "${artifact_integrity_json}" '"skipped_artifacts":'
assert_contains "${artifact_integrity_json}" '"storage_backend": "local"'
assert_contains "${artifact_integrity_json}" '"storage_backend": "external_project"'
assert_contains "${artifact_integrity_json}" '"status": "pass"'
assert_contains "${artifact_integrity_json}" '"status": "skipped"'
assert_contains "${artifact_integrity_json}" '"read_contents": true'
assert_contains "${artifact_integrity_json}" '"read_contents": false'
assert_contains "${artifact_integrity_json}" '"actual_sha256":'
assert_contains "${artifact_integrity_json}" '"reason": "project_reference_metadata_only"'

echo "smoke-local: adapter/profile conformance"
conformance_json="$(go run ./cmd/areaflow conformance check "${project_key}" --json)"
assert_contains "${conformance_json}" '"status": "pass"'
assert_contains "${conformance_json}" '"mode": "read_only_adapter_profile_conformance"'
assert_contains "${conformance_json}" '"key": "'"${project_key}"'"'
assert_contains "${conformance_json}" '"profile_id": "areamatrix"'
assert_contains "${conformance_json}" '"adapter": "areamatrix"'
assert_contains "${conformance_json}" '"profile_hash": "'
assert_contains "${conformance_json}" '"stage_count": 16'
assert_contains "${conformance_json}" '"gate_count": 17'
assert_contains "${conformance_json}" '"key": "project_adapter_profile"'
assert_contains "${conformance_json}" '"key": "profile_load"'
assert_contains "${conformance_json}" '"key": "profile_validate"'
assert_contains "${conformance_json}" '"key": "profile_item_state_contract"'
assert_contains "${conformance_json}" '"key": "profile_stage_contract"'
assert_contains "${conformance_json}" '"key": "profile_gate_contract"'
assert_contains "${conformance_json}" '"key": "profile_transition_contract"'
assert_contains "${conformance_json}" '"key": "profile_hard_rule_contract"'
assert_contains "${conformance_json}" '"key": "profile_permission_policy_contract"'
assert_contains "${conformance_json}" '"key": "profile_artifact_policy_contract"'
assert_contains "${conformance_json}" '"key": "profile_cutover_policy_contract"'
assert_contains "${conformance_json}" '"key": "adapter_snapshot"'
assert_contains "${conformance_json}" '"key": "adapter_profile_boundary"'
assert_contains "${conformance_json}" '"key": "project_config_policy"'
assert_contains "${conformance_json}" '"conformance_writes_project": false'
assert_contains "${conformance_json}" '"conformance_runs_commands": false'
assert_contains "${conformance_json}" '"conformance_resolves_secret": false'

echo "smoke-local: release readiness"
release_json="$(go run ./cmd/areaflow release readiness --json)"
assert_release_real_100_guardrail "${release_json}"
assert_contains "${release_json}" '"status": "needs_attention"'
assert_contains "${release_json}" '"mode": "read_only_release_readiness"'
assert_contains "${release_json}" '"backup": {'
assert_contains "${release_json}" '"restore_plan": {'
assert_contains "${release_json}" '"audit_coverage": {'
assert_contains "${release_json}" '"projects": ['
assert_contains "${release_json}" '"items": ['
assert_contains "${release_json}" '"key": "backup_manifest"'
assert_contains "${release_json}" '"key": "restore_plan"'
assert_contains "${release_json}" '"key": "audit_coverage"'
assert_contains "${release_json}" '"key": "permission_policy:'"${project_key}"'"'
assert_contains "${release_json}" '"key": "artifact_integrity:'"${project_key}"'"'
assert_contains "${release_json}" '"key": "adapter_profile_conformance:'"${project_key}"'"'
assert_contains "${release_json}" '"restore_status": "needs_attention"'
assert_contains "${release_json}" '"audit_status": "warn"'
assert_contains "${release_json}" '"integrity_status": "warn"'
assert_contains "${release_json}" '"conformance_status": "pass"'
assert_contains "${release_json}" '"generate_release_readiness"'
assert_contains "${release_json}" '"create_release_package"'
assert_contains "${release_json}" '"start_worker"'

echo "smoke-local: release remediation plan"
remediation_json="$(go run ./cmd/areaflow release remediation-plan --json)"
assert_release_real_100_guardrail "${remediation_json}"
assert_contains "${remediation_json}" '"status": "needs_attention"'
assert_contains "${remediation_json}" '"mode": "read_only_release_remediation_plan"'
assert_contains "${remediation_json}" '"readiness": {'
assert_contains "${remediation_json}" '"actions": ['
assert_contains "${remediation_json}" '"key": "remediate:restore_plan"'
assert_contains "${remediation_json}" '"key": "remediate:audit_coverage"'
assert_contains "${remediation_json}" '"key": "remediate:artifact_integrity:'"${project_key}"'"'
assert_contains "${remediation_json}" '"category": "restore"'
assert_contains "${remediation_json}" '"category": "audit"'
assert_contains "${remediation_json}" '"category": "artifact"'
assert_contains "${remediation_json}" '"owner": "release_owner"'
assert_contains "${remediation_json}" '"owner": "platform_owner"'
assert_contains "${remediation_json}" '"owner": "artifact_owner"'
assert_contains "${remediation_json}" '"next_command": "areaflow backup restore-plan --json"'
assert_contains "${remediation_json}" '"next_command": "areaflow audit coverage --json"'
assert_contains "${remediation_json}" '"next_command": "areaflow artifact integrity '"${project_key}"' --json"'
assert_contains "${remediation_json}" '"generate_remediation_plan"'
assert_contains "${remediation_json}" '"mark_gap_accepted"'
assert_contains "${remediation_json}" '"write_artifact_store"'

echo "smoke-local: release acceptance preview"
acceptance_json="$(go run ./cmd/areaflow release acceptance-preview --json)"
assert_release_real_100_guardrail "${acceptance_json}"
assert_contains "${acceptance_json}" '"status": "needs_decision"'
assert_contains "${acceptance_json}" '"mode": "read_only_release_acceptance_preview"'
assert_contains "${acceptance_json}" '"remediation": {'
assert_contains "${acceptance_json}" '"decisions": ['
assert_contains "${acceptance_json}" '"key": "accept:restore_plan"'
assert_contains "${acceptance_json}" '"key": "accept:audit_coverage"'
assert_contains "${acceptance_json}" '"key": "accept:artifact_integrity:'"${project_key}"'"'
assert_contains "${acceptance_json}" '"status": "needs_decision"'
assert_contains "${acceptance_json}" '"acceptance_type": "metadata_only_history"'
assert_contains "${acceptance_json}" '"acceptance_type": "future_only_gap"'
assert_contains "${acceptance_json}" '"acceptance_type": "archive_exception"'
assert_contains "${acceptance_json}" '"owner": "release_owner"'
assert_contains "${acceptance_json}" '"owner": "platform_owner"'
assert_contains "${acceptance_json}" '"owner": "artifact_owner"'
assert_contains "${acceptance_json}" '"required_evidence": ['
assert_contains "${acceptance_json}" '"next_command": "areaflow backup restore-plan --json"'
assert_contains "${acceptance_json}" '"next_command": "areaflow audit coverage --json"'
assert_contains "${acceptance_json}" '"next_command": "areaflow artifact integrity '"${project_key}"' --json"'
assert_contains "${acceptance_json}" '"generate_acceptance_preview"'
assert_contains "${acceptance_json}" '"mark_gap_accepted"'
assert_contains "${acceptance_json}" '"create_approval"'
assert_contains "${acceptance_json}" '"apply_release"'

echo "smoke-local: release acceptance gate"
acceptance_gate_json="$(go run ./cmd/areaflow release acceptance-gate --json)"
assert_release_real_100_guardrail "${acceptance_gate_json}"
assert_contains "${acceptance_gate_json}" '"status": "blocked"'
assert_contains "${acceptance_gate_json}" '"mode": "read_only_release_acceptance_gate"'
assert_contains "${acceptance_gate_json}" '"preview": {'
assert_contains "${acceptance_gate_json}" '"items": ['
assert_contains "${acceptance_gate_json}" '"key": "gate:accept:restore_plan"'
assert_contains "${acceptance_gate_json}" '"key": "gate:accept:audit_coverage"'
assert_contains "${acceptance_gate_json}" '"key": "gate:accept:artifact_integrity:'"${project_key}"'"'
assert_contains "${acceptance_gate_json}" '"decision_status": "needs_decision"'
assert_contains "${acceptance_gate_json}" '"acceptance_type": "metadata_only_history"'
assert_contains "${acceptance_gate_json}" '"acceptance_type": "future_only_gap"'
assert_contains "${acceptance_gate_json}" '"acceptance_type": "archive_exception"'
assert_contains "${acceptance_gate_json}" '"message": "explicit release acceptance evidence is required before this exception can pass"'
assert_contains "${acceptance_gate_json}" '"required_evidence": ['
assert_contains "${acceptance_gate_json}" '"next_command": "areaflow backup restore-plan --json"'
assert_contains "${acceptance_gate_json}" '"next_command": "areaflow audit coverage --json"'
assert_contains "${acceptance_gate_json}" '"next_command": "areaflow artifact integrity '"${project_key}"' --json"'
assert_contains "${acceptance_gate_json}" '"evaluate_release_acceptance_gate"'
assert_contains "${acceptance_gate_json}" '"mark_gap_accepted"'
assert_contains "${acceptance_gate_json}" '"create_approval"'
assert_contains "${acceptance_gate_json}" '"apply_release"'

echo "smoke-local: release exception doctor"
exception_doctor_json="$(go run ./cmd/areaflow release exception-doctor --json)"
assert_release_real_100_guardrail "${exception_doctor_json}"
assert_contains "${exception_doctor_json}" '"status": "warn"'
assert_contains "${exception_doctor_json}" '"mode": "read_only_release_exception_doctor"'
assert_contains "${exception_doctor_json}" '"gate": {'
assert_contains "${exception_doctor_json}" '"checks": ['
assert_contains "${exception_doctor_json}" '"key": "exception_record_schema"'
assert_contains "${exception_doctor_json}" '"key": "exception_audit_contract"'
assert_contains "${exception_doctor_json}" '"key": "exception_write_guardrails"'
assert_contains "${exception_doctor_json}" '"key": "exception:gate:accept:restore_plan"'
assert_contains "${exception_doctor_json}" '"key": "exception:gate:accept:audit_coverage"'
assert_contains "${exception_doctor_json}" '"key": "exception:gate:accept:artifact_integrity:'"${project_key}"'"'
assert_contains "${exception_doctor_json}" '"required_fields": ['
assert_contains "${exception_doctor_json}" '"release.exception.request"'
assert_contains "${exception_doctor_json}" '"release.exception.approve"'
assert_contains "${exception_doctor_json}" '"release.exception.revoke"'
assert_contains "${exception_doctor_json}" '"writes_enabled": false'
assert_contains "${exception_doctor_json}" '"exception_writable": false'
assert_contains "${exception_doctor_json}" '"doctor_marks_gap_accepted": false'
assert_contains "${exception_doctor_json}" '"check_exception_record_requirements"'
assert_contains "${exception_doctor_json}" '"mark_gap_accepted"'
assert_contains "${exception_doctor_json}" '"create_approval"'
assert_contains "${exception_doctor_json}" '"apply_release"'

echo "smoke-local: release exception record preview"
exception_record_json="$(go run ./cmd/areaflow release exception-record-preview --json)"
assert_release_real_100_guardrail "${exception_record_json}"
assert_contains "${exception_record_json}" '"status": "draft"'
assert_contains "${exception_record_json}" '"mode": "read_only_release_exception_record_preview"'
assert_contains "${exception_record_json}" '"doctor": {'
assert_contains "${exception_record_json}" '"drafts": ['
assert_contains "${exception_record_json}" '"key": "release_exception:restore_plan"'
assert_contains "${exception_record_json}" '"key": "release_exception:audit_coverage"'
assert_contains "${exception_record_json}" '"key": "release_exception:artifact_integrity:'"${project_key}"'"'
assert_contains "${exception_record_json}" '"source_decision": "needs_decision"'
assert_contains "${exception_record_json}" '"acceptance_type": "metadata_only_history"'
assert_contains "${exception_record_json}" '"acceptance_type": "future_only_gap"'
assert_contains "${exception_record_json}" '"acceptance_type": "archive_exception"'
assert_contains "${exception_record_json}" '"status": "draft"'
assert_contains "${exception_record_json}" '"review_required": true'
assert_contains "${exception_record_json}" '"audit_actions": ['
assert_contains "${exception_record_json}" '"release.exception.request"'
assert_contains "${exception_record_json}" '"release.exception.approve"'
assert_contains "${exception_record_json}" '"release.exception.revoke"'
assert_contains "${exception_record_json}" '"rollback_plan": "revoke the exception record and rerun release acceptance gate before release apply"'
assert_contains "${exception_record_json}" '"exception_writable": false'
assert_contains "${exception_record_json}" '"preview_exception_records"'
assert_contains "${exception_record_json}" '"insert_exception_record"'
assert_contains "${exception_record_json}" '"insert_audit_event"'

echo "smoke-local: release exception schema preview"
exception_schema_json="$(go run ./cmd/areaflow release exception-schema-preview --json)"
assert_release_real_100_guardrail "${exception_schema_json}"
assert_contains "${exception_schema_json}" '"status": "needs_approval"'
assert_contains "${exception_schema_json}" '"mode": "read_only_release_exception_schema_preview"'
assert_contains "${exception_schema_json}" '"record_preview": {'
assert_contains "${exception_schema_json}" '"tables": ['
assert_contains "${exception_schema_json}" '"name": "release_exceptions"'
assert_contains "${exception_schema_json}" '"columns": ['
assert_contains "${exception_schema_json}" '"name": "exception_key"'
assert_contains "${exception_schema_json}" '"name": "required_evidence"'
assert_contains "${exception_schema_json}" '"name": "rollback_plan"'
assert_contains "${exception_schema_json}" '"indexes": ['
assert_contains "${exception_schema_json}" '"name": "release_exceptions_key_idx"'
assert_contains "${exception_schema_json}" '"foreign_keys": ['
assert_contains "${exception_schema_json}" '"references_table": "projects"'
assert_contains "${exception_schema_json}" '"references_table": "audit_events"'
assert_contains "${exception_schema_json}" '"apply_steps": ['
assert_contains "${exception_schema_json}" '"action": "create_table"'
assert_contains "${exception_schema_json}" '"action": "create_index"'
assert_contains "${exception_schema_json}" '"rollback_steps": ['
assert_contains "${exception_schema_json}" '"action": "drop_table"'
assert_contains "${exception_schema_json}" '"audit_actions": ['
assert_contains "${exception_schema_json}" '"release.exception.request"'
assert_contains "${exception_schema_json}" '"preview_release_exception_schema"'
assert_contains "${exception_schema_json}" '"create_migration_file"'
assert_contains "${exception_schema_json}" '"run_migration"'

echo "smoke-local: release exception migration approval gate"
exception_migration_gate_json="$(go run ./cmd/areaflow release exception-migration-approval-gate --json)"
assert_release_real_100_guardrail "${exception_migration_gate_json}"
assert_contains "${exception_migration_gate_json}" '"status": "blocked"'
assert_contains "${exception_migration_gate_json}" '"mode": "read_only_release_exception_migration_approval_gate"'
assert_contains "${exception_migration_gate_json}" '"schema_preview": {'
assert_contains "${exception_migration_gate_json}" '"status": "needs_approval"'
assert_contains "${exception_migration_gate_json}" '"items": ['
assert_contains "${exception_migration_gate_json}" '"key": "migration_approval:release_exception_schema"'
assert_contains "${exception_migration_gate_json}" '"category": "migration"'
assert_contains "${exception_migration_gate_json}" '"approval_status": "needs_approval"'
assert_contains "${exception_migration_gate_json}" '"risk_level": "R4 migration_security"'
assert_contains "${exception_migration_gate_json}" '"migration_writable": false'
assert_contains "${exception_migration_gate_json}" '"evaluate_release_exception_migration_approval_gate"'
assert_contains "${exception_migration_gate_json}" '"create_migration_file"'
assert_contains "${exception_migration_gate_json}" '"run_migration"'
assert_contains "${exception_migration_gate_json}" '"approve_migration"'

echo "smoke-local: release exception apply preview"
exception_apply_preview_json="$(go run ./cmd/areaflow release exception-apply-preview --json)"
assert_release_real_100_guardrail "${exception_apply_preview_json}"
assert_contains "${exception_apply_preview_json}" '"status": "blocked"'
assert_contains "${exception_apply_preview_json}" '"mode": "read_only_release_exception_apply_preview"'
assert_contains "${exception_apply_preview_json}" '"migration_gate": {'
assert_contains "${exception_apply_preview_json}" '"mode": "read_only_release_exception_migration_approval_gate"'
assert_contains "${exception_apply_preview_json}" '"items": ['
assert_contains "${exception_apply_preview_json}" '"key": "release_exception_apply:migration_approval"'
assert_contains "${exception_apply_preview_json}" '"action": "wait_for_migration_approval"'
assert_contains "${exception_apply_preview_json}" '"risk_level": "R4 migration_security"'
assert_contains "${exception_apply_preview_json}" '"apply_writable": false'
assert_contains "${exception_apply_preview_json}" '"apply_steps": ['
assert_contains "${exception_apply_preview_json}" '"action": "verify_migration_approval"'
assert_contains "${exception_apply_preview_json}" '"action": "apply_release_exception_migration"'
assert_contains "${exception_apply_preview_json}" '"action": "write_exception_records"'
assert_contains "${exception_apply_preview_json}" '"rollback_steps": ['
assert_contains "${exception_apply_preview_json}" '"action": "disable_exception_writes"'
assert_contains "${exception_apply_preview_json}" '"action": "revoke_exception_records"'
assert_contains "${exception_apply_preview_json}" '"preview_release_exception_apply_plan"'
assert_contains "${exception_apply_preview_json}" '"run_migration"'
assert_contains "${exception_apply_preview_json}" '"insert_exception_record"'
assert_contains "${exception_apply_preview_json}" '"apply_release"'

echo "smoke-local: release final gate"
release_final_gate_json="$(go run ./cmd/areaflow release final-gate --json)"
assert_release_real_100_guardrail "${release_final_gate_json}"
assert_contains "${release_final_gate_json}" '"status": "blocked"'
assert_contains "${release_final_gate_json}" '"mode": "read_only_release_final_gate"'
assert_contains "${release_final_gate_json}" '"readiness": {'
assert_contains "${release_final_gate_json}" '"mode": "read_only_release_readiness"'
assert_contains "${release_final_gate_json}" '"acceptance_gate": {'
assert_contains "${release_final_gate_json}" '"mode": "read_only_release_acceptance_gate"'
assert_contains "${release_final_gate_json}" '"exception_apply": {'
assert_contains "${release_final_gate_json}" '"mode": "read_only_release_exception_apply_preview"'
assert_contains "${release_final_gate_json}" '"items": ['
assert_contains "${release_final_gate_json}" '"key": "final_gate:release_readiness"'
assert_contains "${release_final_gate_json}" '"key": "final_gate:release_acceptance"'
assert_contains "${release_final_gate_json}" '"key": "final_gate:release_exception_apply"'
assert_contains "${release_final_gate_json}" '"evaluate_release_final_gate"'
assert_contains "${release_final_gate_json}" '"create_release_package"'
assert_contains "${release_final_gate_json}" '"run_migration"'
assert_contains "${release_final_gate_json}" '"insert_exception_record"'
assert_contains "${release_final_gate_json}" '"apply_release"'

echo "smoke-local: release evidence bundle"
release_evidence_json="$(go run ./cmd/areaflow release evidence-bundle --json)"
assert_release_real_100_guardrail "${release_evidence_json}"
assert_contains "${release_evidence_json}" '"status": "blocked"'
assert_contains "${release_evidence_json}" '"mode": "read_only_release_evidence_bundle"'
assert_contains "${release_evidence_json}" '"final_gate": {'
assert_contains "${release_evidence_json}" '"mode": "read_only_release_final_gate"'
assert_contains "${release_evidence_json}" '"backup": {'
assert_contains "${release_evidence_json}" '"mode": "read_only_manifest"'
assert_contains "${release_evidence_json}" '"audit_coverage": {'
assert_contains "${release_evidence_json}" '"mode": "read_only_audit_coverage"'
assert_contains "${release_evidence_json}" '"items": ['
assert_contains "${release_evidence_json}" '"key": "evidence:release_final_gate"'
assert_contains "${release_evidence_json}" '"key": "evidence:backup_manifest"'
assert_contains "${release_evidence_json}" '"key": "evidence:audit_coverage"'
assert_contains "${release_evidence_json}" '"key": "evidence:project_inventory:'"${project_key}"'"'
assert_contains "${release_evidence_json}" '"assemble_release_evidence_index"'
assert_contains "${release_evidence_json}" '"create_release_package"'
assert_contains "${release_evidence_json}" '"read_artifact_contents"'
assert_contains "${release_evidence_json}" '"apply_release"'

echo "smoke-local: release package preview"
release_package_preview_json="$(go run ./cmd/areaflow release package-preview --json)"
assert_release_real_100_guardrail "${release_package_preview_json}"
assert_contains "${release_package_preview_json}" '"status": "blocked"'
assert_contains "${release_package_preview_json}" '"mode": "read_only_release_package_preview"'
assert_contains "${release_package_preview_json}" '"evidence_bundle": {'
assert_contains "${release_package_preview_json}" '"mode": "read_only_release_evidence_bundle"'
assert_contains "${release_package_preview_json}" '"package_name": "areaflow-v1.0-release-evidence-preview"'
assert_contains "${release_package_preview_json}" '"items": ['
assert_contains "${release_package_preview_json}" '"key": "package:manifest"'
assert_contains "${release_package_preview_json}" '"key": "package:evidence:release_final_gate"'
assert_contains "${release_package_preview_json}" '"package_path": "release/manifest.json"'
assert_contains "${release_package_preview_json}" '"package_writable": false'
assert_contains "${release_package_preview_json}" '"preview_release_package_manifest"'
assert_contains "${release_package_preview_json}" '"create_release_package"'
assert_contains "${release_package_preview_json}" '"read_artifact_contents"'
assert_contains "${release_package_preview_json}" '"compress_artifacts"'
assert_contains "${release_package_preview_json}" '"apply_release"'

echo "smoke-local: release distribution preview"
release_distribution_preview_json="$(go run ./cmd/areaflow release distribution-preview --json)"
assert_release_real_100_guardrail "${release_distribution_preview_json}"
assert_contains "${release_distribution_preview_json}" '"status": "blocked"'
assert_contains "${release_distribution_preview_json}" '"mode": "read_only_release_distribution_preview"'
assert_contains "${release_distribution_preview_json}" '"package_preview": {'
assert_contains "${release_distribution_preview_json}" '"mode": "read_only_release_package_preview"'
assert_contains "${release_distribution_preview_json}" '"items": ['
assert_contains "${release_distribution_preview_json}" '"key": "distribution:package_preview"'
assert_contains "${release_distribution_preview_json}" '"key": "distribution:local_archive"'
assert_contains "${release_distribution_preview_json}" '"key": "distribution:git_release"'
assert_contains "${release_distribution_preview_json}" '"key": "distribution:artifact_registry"'
assert_contains "${release_distribution_preview_json}" '"publish_attempted": false'
assert_contains "${release_distribution_preview_json}" '"release_write_allowed": false'
assert_contains "${release_distribution_preview_json}" '"preview_release_distribution_channels"'
assert_contains "${release_distribution_preview_json}" '"upload_release_artifacts"'
assert_contains "${release_distribution_preview_json}" '"publish_release"'
assert_contains "${release_distribution_preview_json}" '"create_git_tag"'
assert_contains "${release_distribution_preview_json}" '"sign_release"'
assert_contains "${release_distribution_preview_json}" '"push_git"'
assert_contains "${release_distribution_preview_json}" '"apply_release"'

echo "smoke-local: release publish gate"
release_publish_gate_json="$(go run ./cmd/areaflow release publish-gate --json)"
assert_release_real_100_guardrail "${release_publish_gate_json}"
assert_contains "${release_publish_gate_json}" '"status": "blocked"'
assert_contains "${release_publish_gate_json}" '"mode": "read_only_release_publish_gate"'
assert_contains "${release_publish_gate_json}" '"distribution_preview": {'
assert_contains "${release_publish_gate_json}" '"mode": "read_only_release_distribution_preview"'
assert_contains "${release_publish_gate_json}" '"items": ['
assert_contains "${release_publish_gate_json}" '"key": "publish_gate:distribution_preview"'
assert_contains "${release_publish_gate_json}" '"key": "publish_gate:local_archive"'
assert_contains "${release_publish_gate_json}" '"key": "publish_gate:git_release"'
assert_contains "${release_publish_gate_json}" '"key": "publish_gate:artifact_registry"'
assert_contains "${release_publish_gate_json}" '"publish_attempted": false'
assert_contains "${release_publish_gate_json}" '"publish_writable": false'
assert_contains "${release_publish_gate_json}" '"evaluate_release_publish_gate"'
assert_contains "${release_publish_gate_json}" '"publish_release"'
assert_contains "${release_publish_gate_json}" '"create_git_tag"'
assert_contains "${release_publish_gate_json}" '"sign_release"'
assert_contains "${release_publish_gate_json}" '"push_git"'
assert_contains "${release_publish_gate_json}" '"apply_release"'

echo "smoke-local: release publish approval preview"
release_publish_approval_preview_json="$(go run ./cmd/areaflow release publish-approval-preview --json)"
assert_release_real_100_guardrail "${release_publish_approval_preview_json}"
assert_contains "${release_publish_approval_preview_json}" '"status": "blocked"'
assert_contains "${release_publish_approval_preview_json}" '"mode": "read_only_release_publish_approval_preview"'
assert_contains "${release_publish_approval_preview_json}" '"publish_gate": {'
assert_contains "${release_publish_approval_preview_json}" '"mode": "read_only_release_publish_gate"'
assert_contains "${release_publish_approval_preview_json}" '"items": ['
assert_contains "${release_publish_approval_preview_json}" '"key": "publish_approval:publish_gate"'
assert_contains "${release_publish_approval_preview_json}" '"approval_status": "blocked"'
assert_contains "${release_publish_approval_preview_json}" '"approval_writable": false'
assert_contains "${release_publish_approval_preview_json}" '"publish_writable": false'
assert_contains "${release_publish_approval_preview_json}" '"preview_release_publish_approval"'
assert_contains "${release_publish_approval_preview_json}" '"create_approval"'
assert_contains "${release_publish_approval_preview_json}" '"approve_release"'
assert_contains "${release_publish_approval_preview_json}" '"publish_release"'
assert_contains "${release_publish_approval_preview_json}" '"create_git_tag"'
assert_contains "${release_publish_approval_preview_json}" '"sign_release"'
assert_contains "${release_publish_approval_preview_json}" '"push_git"'
assert_contains "${release_publish_approval_preview_json}" '"apply_release"'

echo "smoke-local: release rollout plan preview"
release_rollout_plan_preview_json="$(go run ./cmd/areaflow release rollout-plan-preview --json)"
assert_release_real_100_guardrail "${release_rollout_plan_preview_json}"
assert_contains "${release_rollout_plan_preview_json}" '"status": "blocked"'
assert_contains "${release_rollout_plan_preview_json}" '"mode": "read_only_release_rollout_plan_preview"'
assert_contains "${release_rollout_plan_preview_json}" '"publish_approval_preview": {'
assert_contains "${release_rollout_plan_preview_json}" '"mode": "read_only_release_publish_approval_preview"'
assert_contains "${release_rollout_plan_preview_json}" '"items": ['
assert_contains "${release_rollout_plan_preview_json}" '"key": "rollout_plan:publish_approval"'
assert_contains "${release_rollout_plan_preview_json}" '"action": "wait_for_publish_approval_preview"'
assert_contains "${release_rollout_plan_preview_json}" '"rollout_steps": ['
assert_contains "${release_rollout_plan_preview_json}" '"action": "verify_publish_approval"'
assert_contains "${release_rollout_plan_preview_json}" '"verification_checkpoints": ['
assert_contains "${release_rollout_plan_preview_json}" '"action": "publish_approval_recorded"'
assert_contains "${release_rollout_plan_preview_json}" '"rollback_steps": ['
assert_contains "${release_rollout_plan_preview_json}" '"action": "pause_distribution"'
assert_contains "${release_rollout_plan_preview_json}" '"rollout_writable": false'
assert_contains "${release_rollout_plan_preview_json}" '"publish_attempted": false'
assert_contains "${release_rollout_plan_preview_json}" '"preview_release_rollout_plan"'
assert_contains "${release_rollout_plan_preview_json}" '"create_rollout"'
assert_contains "${release_rollout_plan_preview_json}" '"write_release_state"'
assert_contains "${release_rollout_plan_preview_json}" '"publish_release"'
assert_contains "${release_rollout_plan_preview_json}" '"create_git_tag"'
assert_contains "${release_rollout_plan_preview_json}" '"sign_release"'
assert_contains "${release_rollout_plan_preview_json}" '"push_git"'
assert_contains "${release_rollout_plan_preview_json}" '"apply_release"'

echo "smoke-local: execution cutover readiness"
execution_cutover_json="$(go run ./cmd/areaflow project execution-cutover-readiness "${project_key}" --json)"
assert_contains "${execution_cutover_json}" '"status": "blocked"'
assert_contains "${execution_cutover_json}" '"mode": "read_only_areamatrix_execution_cutover_readiness"'
assert_contains "${execution_cutover_json}" '"key": "import_mirror_shadow"'
assert_contains "${execution_cutover_json}" '"key": "compatibility_shim"'
assert_contains "${execution_cutover_json}" '"key": "task_loop_run_policy"'
assert_contains "${execution_cutover_json}" '"key": "explicit_execution_cutover_approval"'
assert_contains "${execution_cutover_json}" '"execution_cutover_apply_open": false'
assert_contains "${execution_cutover_json}" '"project_write_attempted": false'
assert_contains "${execution_cutover_json}" '"execution_write_attempted": false'
assert_contains "${execution_cutover_json}" '"task_loop_run_forwarded": false'
assert_contains "${execution_cutover_json}" '"engine_call_attempted": false'
assert_contains "${execution_cutover_json}" '"commands_run": false'
assert_contains "${execution_cutover_json}" '"secrets_resolved": false'
assert_contains "${execution_cutover_json}" '"forward_task_loop_run"'
assert_contains "${execution_cutover_json}" '"apply_execution_cutover"'

echo "smoke-local: ops smoke proof record"
ops_smoke_proof_json="$(go run ./cmd/areaflow ops smoke-proof record "${project_key}" --key v1_stable_fixture_smoke --summary "smoke-local long fixture chain passed" --evidence-uri "scripts/smoke-local.sh" --idempotency-key "ops-smoke-proof:${project_key}:${ready_workflow_label}" --reason "record long smoke proof after all smoke-local checks passed" --json)"
assert_contains "${ops_smoke_proof_json}" '"proof_key": "v1_stable_fixture_smoke"'
assert_contains "${ops_smoke_proof_json}" '"status": "recorded"'
assert_contains "${ops_smoke_proof_json}" '"evidence_status": "pass"'
assert_contains "${ops_smoke_proof_json}" '"decision": "allowed"'
assert_contains "${ops_smoke_proof_json}" '"project_write_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"execution_write_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"engine_call_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"service_process_control_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"support_bundle_exported": false'
assert_contains "${ops_smoke_proof_json}" '"migration_apply_attempted": false'
assert_contains "${ops_smoke_proof_json}" '"remote_telemetry_enabled": false'
assert_contains "${ops_smoke_proof_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${ops_smoke_proof_json}" '"record_command_runs_smoke": false'

echo "smoke-local: ops readiness after smoke proof"
ops_readiness_after_proof_json="$(go run ./cmd/areaflow ops readiness --json)"
assert_contains "${ops_readiness_after_proof_json}" '"key": "install_migrate_start_register_smoke"'
assert_contains "${ops_readiness_after_proof_json}" '"evidence_recorded": true'
assert_contains "${ops_readiness_after_proof_json}" '"latest_smoke_proof_key": "v1_stable_fixture_smoke"'
assert_contains "${ops_readiness_after_proof_json}" '"latest_smoke_proof_event_id":'
assert_not_contains "${ops_readiness_after_proof_json}" '"fresh_local_ops_smoke_missing"'
assert_not_contains "${ops_readiness_after_proof_json}" '"full_migration_ledger_missing"'

echo "smoke-local: completion audit after ops smoke proof"
completion_after_ops_proof_json="$(go run ./cmd/areaflow completion audit --json)"
assert_completion_real_100_guardrail "${completion_after_ops_proof_json}"
assert_contains "${completion_after_ops_proof_json}" '"key": "E7_operations_readiness"'
if [[ "${project_key}" == "areamatrix" ]]; then
  assert_not_contains "${completion_after_ops_proof_json}" '"fresh_local_ops_smoke_missing"'
else
  assert_contains "${completion_after_ops_proof_json}" '"fresh_local_ops_smoke_missing"'
  assert_not_contains "${completion_after_ops_proof_json}" "\"latest_operations_smoke_proof_project_key\": \"${project_key}\""
fi
assert_not_contains "${completion_after_ops_proof_json}" '"full_migration_ledger_missing"'
assert_contains "${completion_after_ops_proof_json}" '"smoke_run_attempted": false'

echo "smoke-local: ok"
