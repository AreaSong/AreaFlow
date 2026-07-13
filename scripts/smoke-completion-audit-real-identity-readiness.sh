#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

if [[ "${AREAFLOW_SMOKE_DOCKER_ISOLATED_DB_CREATED:-0}" != "1" && "${AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB:-0}" != "1" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: blocked; this smoke writes AreaFlow DB state and requires an isolated Docker smoke DB or AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB=1" >&2
  exit 1
fi

project_key="areamatrix"
config_path="${AREAFLOW_REAL_IDENTITY_CONFIG:-examples/areamatrix/areaflow.yaml}"
project_root="${AREAFLOW_REAL_IDENTITY_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
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

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-real-identity-readiness: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-real-identity-readiness: output unexpectedly contained pattern: ${pattern}" >&2
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

protected_path_fingerprint() {
  python3 - "${project_root}" "${protected_paths[@]}" <<'PY'
import hashlib
import os
import stat
import sys

root = os.path.abspath(sys.argv[1])
protected_paths = sys.argv[2:]


def rel(path):
    return os.path.relpath(path, root).replace(os.sep, "/")


def entry(path):
    info = os.lstat(path)
    mode = info.st_mode
    relative = rel(path)
    if stat.S_ISREG(mode):
        with open(path, "rb") as handle:
            content = handle.read()
        return f"{relative}\tfile\t{len(content)}\t{hashlib.sha256(content).hexdigest()}"
    if stat.S_ISDIR(mode):
        return f"{relative}\tdir"
    if stat.S_ISLNK(mode):
        return f"{relative}\tsymlink\t{os.readlink(path)}"
    return f"{relative}\tother\t{stat.filemode(mode)}\t{info.st_size}"


def walk(path, entries):
    entries.append(entry(path))
    if not stat.S_ISDIR(os.lstat(path).st_mode):
        return
    for name in sorted(os.listdir(path)):
        walk(os.path.join(path, name), entries)


entries = []
for protected_path in protected_paths:
    absolute = os.path.abspath(os.path.join(root, protected_path))
    try:
        os.lstat(absolute)
    except FileNotFoundError:
        entries.append(f"{rel(absolute)}\tmissing")
        continue
    walk(absolute, entries)

payload = "\n".join(entries).encode()
print(hashlib.sha256(payload).hexdigest())
PY
}

if [[ ! -d "${project_root}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: missing AreaMatrix project root: ${project_root}" >&2
  exit 1
fi

if [[ ! -f "${config_path}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: missing AreaFlow config: ${config_path}" >&2
  exit 1
fi

status_before="$(file_fingerprint "${status_path}")"
workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_before="$(protected_path_fingerprint)"
protected_path_git_status_before="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"

echo "smoke-completion-audit-real-identity-readiness: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-completion-audit-real-identity-readiness: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-completion-audit-real-identity-readiness: project import ${project_key}"
import_output="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output}" "imported ${project_key}"

echo "smoke-completion-audit-real-identity-readiness: completion audit --json"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_contains "${completion_json}" '"status": "'
assert_contains "${completion_json}" '"key": "E4_areamatrix_dogfood_completion"'
assert_contains "${completion_json}" '"category": "dogfood"'
assert_contains "${completion_json}" '"package_a_status_projection_blockers":'
assert_contains "${completion_json}" '"package_a_status_projection_apply_provenance_missing"'
assert_not_contains "${completion_json}" '"completion_audit_snapshot_package_a_not_applied"'
assert_not_contains "${completion_json}" '"package_a_status_projection_not_written"'
assert_contains "${completion_json}" '"package_a_has_written_status_projection": false'
assert_contains "${completion_json}" '"package_a_status_projection_ready": false'
assert_not_contains "${completion_json}" '"package_a_status_projection_ready": true'

echo "smoke-completion-audit-real-identity-readiness: completion audit-snapshot readiness --json"
readiness_json="$(go run ./cmd/areaflow completion audit-snapshot readiness "${project_key}" --json)"
assert_contains "${readiness_json}" '"project": {'
assert_contains "${readiness_json}" '"key": "areamatrix"'
assert_contains "${readiness_json}" '"root": "'"${project_root}"'"'
assert_contains "${readiness_json}" '"kind": "product-repo"'
assert_contains "${readiness_json}" '"adapter": "areamatrix"'
assert_contains "${readiness_json}" '"workflow_profile": "areamatrix"'
assert_contains "${readiness_json}" '"default_branch": "main"'
assert_contains "${readiness_json}" '"status": "blocked"'
assert_contains "${readiness_json}" '"real_100_status": "blocked"'
assert_contains "${readiness_json}" '"readiness_scope": "completion_audit_evidence_only"'
assert_contains "${readiness_json}" '"claim_scope": "completion_audit_evidence_only"'
assert_contains "${readiness_json}" '"not_real_100": true'
assert_contains "${readiness_json}" '"evidence_only": true'
assert_contains "${readiness_json}" '"status_alone_is_not_completion": true'
assert_contains "${readiness_json}" '"release_candidate_decision": "requires_release_candidate_snapshot"'
assert_contains "${readiness_json}" '"real_100_blockers": ['
assert_contains "${readiness_json}" '"package_a_status_projection_apply_provenance_missing"'
assert_contains "${readiness_json}" '"release_candidate_snapshot_not_ready"'
assert_contains "${readiness_json}" '"has_snapshot": false'
assert_contains "${readiness_json}" '"required_class": "release_candidate"'
assert_contains "${readiness_json}" '"key": "completion_audit_snapshot_missing"'
assert_contains "${readiness_json}" '"gaps": ['
assert_contains "${readiness_json}" '"category": "snapshot"'
assert_contains "${readiness_json}" '"missing_proof_evidence_uri_keys":'
assert_contains "${readiness_json}" '"missing_proof_event_id_keys":'
assert_contains "${readiness_json}" '"missing_proof_provenance_keys":'
assert_contains "${readiness_json}" '"proof_evidence_uri_blockers":'
assert_contains "${readiness_json}" '"proof_event_id_blockers":'
assert_contains "${readiness_json}" '"proof_provenance_blockers":'
assert_contains "${readiness_json}" '"closure": {'
assert_contains "${readiness_json}" '"ready_for_release_candidate_closure": false'
assert_contains "${readiness_json}" '"required_evidence_class": "release_candidate"'
assert_contains "${readiness_json}" '"snapshot_status": "missing"'
assert_contains "${readiness_json}" '"proof_evidence_uri_status": "missing"'
assert_contains "${readiness_json}" '"proof_event_id_status": "missing"'
assert_contains "${readiness_json}" '"proof_provenance_status": "missing"'
assert_contains "${readiness_json}" '"release_evidence_bundle_status": "pass"'
assert_contains "${readiness_json}" '"snapshot": {'
assert_contains "${readiness_json}" '"proof_event_ids": {'
assert_contains "${readiness_json}" '"proof_provenance": {'
assert_contains "${readiness_json}" '"required_evidence_class": "release_candidate"'
assert_contains "${readiness_json}" '"current_audit_status":'
assert_contains "${readiness_json}" '"current_bundle_hash":'
assert_contains "${readiness_json}" '"current_proof_evidence_uris":'
assert_contains "${readiness_json}" '"current_proof_evidence_uri_map":'
assert_contains "${readiness_json}" '"current_proof_event_ids":'
assert_contains "${readiness_json}" '"current_proof_provenance_map":'
assert_contains "${readiness_json}" '"current_missing_proof_evidence_uri_keys":'
assert_contains "${readiness_json}" '"E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri"'
assert_contains "${readiness_json}" '"current_missing_proof_event_id_keys":'
assert_contains "${readiness_json}" '"E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id"'
assert_contains "${readiness_json}" '"current_missing_proof_provenance_keys":'
assert_contains "${readiness_json}" '"E7_operations_readiness.latest_operations_smoke_proof_key"'
assert_contains "${readiness_json}" '"current_proof_evidence_uri_blockers":'
assert_contains "${readiness_json}" '"current_proof_evidence_uri_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri"'
assert_contains "${readiness_json}" '"current_proof_event_id_blockers":'
assert_contains "${readiness_json}" '"current_proof_event_id_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id"'
assert_contains "${readiness_json}" '"current_proof_provenance_blockers":'
assert_contains "${readiness_json}" '"current_proof_provenance_missing:E7_operations_readiness.latest_operations_smoke_proof_key"'
assert_contains "${readiness_json}" '"package_a_status_projection_blockers":'
assert_contains "${readiness_json}" '"package_a_status_projection_apply_provenance_missing"'
assert_not_contains "${readiness_json}" '"completion_audit_snapshot_package_a_not_applied"'
assert_not_contains "${readiness_json}" '"package_a_status_projection_not_written"'
assert_contains "${readiness_json}" '"package_a_has_written_status_projection": false'
assert_contains "${readiness_json}" '"package_a_status_projection_status": "blocked"'
assert_contains "${readiness_json}" '"package_a_status_projection": {'
assert_contains "${readiness_json}" '"read_only": true'
assert_contains "${readiness_json}" '"project_write_attempted": false'
assert_contains "${readiness_json}" '"execution_write_attempted": false'
assert_contains "${readiness_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${readiness_json}" '"commands_run": false'
assert_contains "${readiness_json}" '"smoke_run_attempted": false'
assert_contains "${readiness_json}" '"worker_started": false'
assert_not_contains "${readiness_json}" '"completion_audit_snapshot_real_project_identity_missing"'
assert_not_contains "${readiness_json}" '"project_root_not_real_areamatrix"'
assert_not_contains "${readiness_json}" '"project_key_mismatch"'
assert_not_contains "${readiness_json}" '"completion_audit_snapshot_project_mismatch"'
assert_not_contains "${readiness_json}" '"completion_audit_snapshot_release_candidate_present"'
assert_not_contains "${readiness_json}" '"ready_for_release_candidate_closure": true'
assert_not_contains "${readiness_json}" '"release_package_created": true'
assert_not_contains "${readiness_json}" '"publish_attempted": true'
assert_not_contains "${readiness_json}" '"restore_apply_attempted": true'

status_after="$(file_fingerprint "${status_path}")"
workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_after="$(protected_path_fingerprint)"
protected_path_git_status_after="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"

if [[ "${status_before}" != "${status_after}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: real AreaMatrix status changed unexpectedly: ${status_path}" >&2
  exit 1
fi

if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: real AreaMatrix workflow README changed unexpectedly: ${workflow_readme}" >&2
  exit 1
fi

if [[ "${protected_path_fingerprint_before}" != "${protected_path_fingerprint_after}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: real AreaMatrix protected path fingerprint changed unexpectedly" >&2
  exit 1
fi

if [[ "${protected_path_git_status_before}" != "${protected_path_git_status_after}" ]]; then
  echo "smoke-completion-audit-real-identity-readiness: real AreaMatrix protected path git status changed unexpectedly" >&2
  exit 1
fi

echo "smoke-completion-audit-real-identity-readiness: pass real identity readonly project=${project_key}"
