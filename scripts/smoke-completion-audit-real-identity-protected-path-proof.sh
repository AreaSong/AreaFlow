#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

if [[ "${AREAFLOW_SMOKE_DOCKER_ISOLATED_DB_CREATED:-0}" != "1" && "${AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB:-0}" != "1" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: blocked; this smoke writes AreaFlow DB state and requires an isolated Docker smoke DB or AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB=1" >&2
  exit 1
fi

project_key="${AREAFLOW_REAL_IDENTITY_PROJECT_KEY:-areamatrix}"
config_path="${AREAFLOW_REAL_IDENTITY_CONFIG:-examples/areamatrix/areaflow.yaml}"
project_root="${AREAFLOW_REAL_IDENTITY_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
evidence_uri="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_EVIDENCE_URI:-command:git -C ${project_root} status --short -- protected AreaMatrix paths}"
clean_summary="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_CLEAN_SUMMARY:-AreaMatrix protected path git status returned no output for the protected set}"
authorized_summary="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_AUTHORIZED_SUMMARY:-AreaMatrix protected path git status matches reviewed authorized dirty state}"
reviewed_dirty_output_hash="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_REVIEWED_DIRTY_OUTPUT_SHA256:-}"
approval_id="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_APPROVAL_ID:-}"
allowed_paths_csv="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_ALLOWED_PATHS:-}"
reviewer="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_REVIEWER:-}"
rollback_evidence_uri="${AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_ROLLBACK_EVIDENCE_URI:-}"

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

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-real-identity-protected-path-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
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
  areaflow_protected_path_fingerprint "${project_root}" "__no_target_exclusion__" "${protected_paths[@]}"
}

dirty_path_count() {
  local status_output="$1"
  printf "%s\n" "${status_output}" | awk 'NF {count++} END {print count+0}'
}

trim() {
  xargs <<<"$1"
}

require_authorized_dirty_env() {
  local dirty_output_hash="$1"
  local missing=()

  [[ -z "${reviewed_dirty_output_hash}" ]] && missing+=("AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_REVIEWED_DIRTY_OUTPUT_SHA256")
  [[ -z "${approval_id}" ]] && missing+=("AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_APPROVAL_ID")
  [[ -z "${allowed_paths_csv}" ]] && missing+=("AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_ALLOWED_PATHS")
  [[ -z "${reviewer}" ]] && missing+=("AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_REVIEWER")
  [[ -z "${rollback_evidence_uri}" ]] && missing+=("AREAFLOW_REAL_IDENTITY_PROTECTED_PATH_PROOF_ROLLBACK_EVIDENCE_URI")

  if ((${#missing[@]} > 0)); then
    echo "smoke-completion-audit-real-identity-protected-path-proof: blocked; protected paths are dirty and authorized proof env is incomplete" >&2
    printf 'smoke-completion-audit-real-identity-protected-path-proof: missing_env=%s\n' "${missing[@]}" >&2
    echo "smoke-completion-audit-real-identity-protected-path-proof: dirty_output_sha256=${dirty_output_hash}" >&2
    exit 1
  fi

  if [[ "${reviewed_dirty_output_hash}" != "${dirty_output_hash}" ]]; then
    echo "smoke-completion-audit-real-identity-protected-path-proof: blocked; reviewed dirty hash does not match current protected path status" >&2
    echo "smoke-completion-audit-real-identity-protected-path-proof: reviewed_dirty_output_sha256=${reviewed_dirty_output_hash}" >&2
    echo "smoke-completion-audit-real-identity-protected-path-proof: dirty_output_sha256=${dirty_output_hash}" >&2
    exit 1
  fi
}

append_allowed_path_args() {
  local path
  IFS=',' read -ra raw_paths <<<"${allowed_paths_csv}"
  for path in "${raw_paths[@]}"; do
    path="$(trim "${path}")"
    [[ -z "${path}" ]] && continue
    allowed_path_args+=(--allowed-path "${path}")
  done
  if ((${#allowed_path_args[@]} == 0)); then
    echo "smoke-completion-audit-real-identity-protected-path-proof: blocked; no non-empty allowed paths were provided" >&2
    exit 1
  fi
}

assert_proof_json() {
  local proof_json="$1"
  local expected_status="$2"

  PROOF_JSON="${proof_json}" EXPECTED_STATUS="${expected_status}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["PROOF_JSON"])
expected = os.environ["EXPECTED_STATUS"]
errors = []

def expect(condition, message):
    if not condition:
        errors.append(message)

expect(data.get("status") == "recorded", f"status={data.get('status')!r}")
expect(data.get("decision") == "allowed", f"decision={data.get('decision')!r}")
expect(data.get("proof_status") == expected, f"proof_status={data.get('proof_status')!r}")
expect(data.get("event_id", 0) != 0, "event_id missing")
expect(data.get("audit_event_id", 0) != 0, "audit_event_id missing")
expect(data.get("project_write_attempted") is False, "project_write_attempted should be false")
expect(data.get("execution_write_attempted") is False, "execution_write_attempted should be false")
expect(data.get("engine_call_attempted") is False, "engine_call_attempted should be false")
expect(data.get("commands_run") is False, "commands_run should be false")
expect(data.get("git_status_run_by_command") is False, "git_status_run_by_command should be false")
expect(data.get("protected_path_proof_binding_status") == "pass", "binding status should pass")
expect(data.get("protected_path_proof_binding_blockers") == [], "binding blockers should be empty")
expect(isinstance(data.get("protected_path_set_hash"), str) and len(data["protected_path_set_hash"]) == 64, "protected_path_set_hash missing")
expect(data.get("protected_path_set_count") == 7, f"protected_path_set_count={data.get('protected_path_set_count')!r}")
if expected == "clean":
    expect(data.get("git_status_output_empty") is True, "clean proof should have empty git status")
    expect(data.get("git_status_output_lines") == 0, "clean proof should have zero git status lines")
    expect(data.get("area_matrix_protected_paths_touched") is False, "clean proof should not touch protected paths")
else:
    expect(data.get("git_status_output_empty") is False, "authorized proof should include git status output")
    expect(data.get("git_status_output_lines", 0) > 0, "authorized proof should have git status lines")
    expect(data.get("area_matrix_protected_paths_touched") is True, "authorized proof should mark touched paths")
    expect(data.get("authorized_proof_complete") is True, "authorized proof should be complete")
    expect(data.get("authorized_approval_id"), "authorized_approval_id missing")
    expect(data.get("authorized_dirty_output_hash") == data.get("git_status_output_hash"), "dirty hash mismatch")
    expect(data.get("authorized_reviewer"), "authorized_reviewer missing")
    expect(data.get("authorized_rollback_evidence_uri"), "rollback evidence missing")
    expect(data.get("authorized_allowed_paths"), "allowed paths missing")
    expect(data.get("authorized_touched_paths"), "touched paths missing")

if errors:
    print("protected path proof JSON failed checks:", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)
    sys.exit(1)
PY
}

assert_completion_audit_consumes_proof() {
  local audit_json="$1"
  local expected_status="$2"

  AUDIT_JSON="${audit_json}" EXPECTED_STATUS="${expected_status}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["AUDIT_JSON"])
expected = os.environ["EXPECTED_STATUS"]
items = {item.get("key"): item for item in data.get("items", [])}
item = items.get("E9_areamatrix_protected_path_proof")
errors = []

def expect(condition, message):
    if not condition:
        errors.append(message)

expect(data.get("protected_path_proof_status") == "complete", f"protected_path_proof_status={data.get('protected_path_proof_status')!r}")
expect(item is not None, "E9 item missing")
if item:
    metadata = item.get("metadata", {})
    expect(item.get("status") == "complete", f"E9 status={item.get('status')!r}")
    expect(not item.get("blocked_by"), f"E9 blocked_by={item.get('blocked_by')!r}")
    expect(metadata.get("latest_proof_status") == "recorded", f"latest_proof_status={metadata.get('latest_proof_status')!r}")
    expect(metadata.get("latest_proof_decision") == "allowed", f"latest_proof_decision={metadata.get('latest_proof_decision')!r}")
    expect(metadata.get("latest_proof_project_key") == "areamatrix", f"latest_proof_project_key={metadata.get('latest_proof_project_key')!r}")
    expect(metadata.get("latest_proof_traceable_evidence") is True, "traceable evidence missing")
    expect(metadata.get("protected_path_proof_binding_status") == "pass", "binding status should pass")
    expect(metadata.get("protected_path_proof_binding_blockers") == [], "binding blockers should be empty")
    expect(metadata.get("protected_path_set_count") == 7, f"protected_path_set_count={metadata.get('protected_path_set_count')!r}")
    if expected == "authorized":
        expect(metadata.get("authorized_proof_complete") is True, "authorized_proof_complete should be true")
        expect(metadata.get("area_matrix_protected_paths_touched") is True, "authorized proof should mark touched paths")
        expect(metadata.get("authorized_dirty_output_hash") == metadata.get("git_status_output_hash"), "authorized dirty hash should match")
    else:
        expect(metadata.get("git_status_output_empty") is True, "clean proof should have empty git status")
        expect(metadata.get("area_matrix_protected_paths_touched") is False, "clean proof should not mark touched paths")

if errors:
    print("completion audit did not consume protected path proof:", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)
    sys.exit(1)
PY
}

if [[ ! -d "${project_root}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: missing AreaMatrix project root: ${project_root}" >&2
  exit 1
fi

if [[ ! -f "${config_path}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: missing AreaFlow config: ${config_path}" >&2
  exit 1
fi

status_before="$(file_fingerprint "${status_path}")"
workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_before="$(protected_path_fingerprint)"
protected_path_git_status_before="$(protected_path_git_status)"
dirty_output_hash_before="$(protected_path_status_hash "${protected_path_git_status_before}")"
dirty_count_before="$(dirty_path_count "${protected_path_git_status_before}")"

echo "smoke-completion-audit-real-identity-protected-path-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-completion-audit-real-identity-protected-path-proof: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-completion-audit-real-identity-protected-path-proof: project import ${project_key}"
import_output="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output}" "imported ${project_key}"

proof_status="clean"
proof_args=(
  completion protected-path-proof record "${project_key}"
  --status clean
  --summary "${clean_summary}"
  --evidence-uri "${evidence_uri}"
  --actor "real-identity-smoke"
  --reason "record real AreaMatrix protected path proof smoke"
  --json
)

if [[ -n "${protected_path_git_status_before}" ]]; then
  proof_status="authorized"
  require_authorized_dirty_env "${dirty_output_hash_before}"
  allowed_path_args=()
  append_allowed_path_args
  proof_args=(
    completion protected-path-proof record "${project_key}"
    --status authorized
    --summary "${authorized_summary}"
    --evidence-uri "${evidence_uri}"
    --git-status-output "${protected_path_git_status_before}"
    --dirty-output-hash "${dirty_output_hash_before}"
    --approval-id "${approval_id}"
    "${allowed_path_args[@]}"
    --reviewer "${reviewer}"
    --rollback-evidence-uri "${rollback_evidence_uri}"
    --actor "real-identity-smoke"
    --reason "record authorized real AreaMatrix protected path proof smoke"
    --json
  )
fi

echo "smoke-completion-audit-real-identity-protected-path-proof: protected_path_dirty_path_count=${dirty_count_before}"
echo "smoke-completion-audit-real-identity-protected-path-proof: dirty_output_sha256=${dirty_output_hash_before}"
echo "smoke-completion-audit-real-identity-protected-path-proof: record proof status=${proof_status}"
proof_json="$(go run ./cmd/areaflow "${proof_args[@]}")"
assert_proof_json "${proof_json}" "${proof_status}"

echo "smoke-completion-audit-real-identity-protected-path-proof: completion audit --json"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_completion_audit_consumes_proof "${completion_json}" "${proof_status}"

status_after="$(file_fingerprint "${status_path}")"
workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_after="$(protected_path_fingerprint)"
protected_path_git_status_after="$(protected_path_git_status)"
dirty_output_hash_after="$(protected_path_status_hash "${protected_path_git_status_after}")"

if [[ "${status_before}" != "${status_after}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: real AreaMatrix status changed unexpectedly: ${status_path}" >&2
  exit 1
fi

if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: real AreaMatrix workflow README changed unexpectedly: ${workflow_readme}" >&2
  exit 1
fi

if [[ "${protected_path_fingerprint_before}" != "${protected_path_fingerprint_after}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: real AreaMatrix protected path fingerprint changed unexpectedly" >&2
  exit 1
fi

if [[ "${protected_path_git_status_before}" != "${protected_path_git_status_after}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: real AreaMatrix protected path git status changed unexpectedly" >&2
  exit 1
fi

if [[ "${dirty_output_hash_before}" != "${dirty_output_hash_after}" ]]; then
  echo "smoke-completion-audit-real-identity-protected-path-proof: real AreaMatrix protected path dirty hash changed unexpectedly" >&2
  exit 1
fi

echo "smoke-completion-audit-real-identity-protected-path-proof: pass proof_status=${proof_status} project=${project_key}"
