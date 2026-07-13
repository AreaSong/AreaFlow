#!/usr/bin/env bash
set -euo pipefail

assert_contains() {
  local output="$1"
  local pattern="$2"
  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-package-b-readiness: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"
  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-package-b-readiness: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_json_list_contains() {
  local json_input="$1"
  local list_key="$2"
  local want="$3"
  JSON_INPUT="${json_input}" JSON_LIST_KEY="${list_key}" JSON_WANT="${want}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["JSON_INPUT"])
list_key = os.environ["JSON_LIST_KEY"]
want = os.environ["JSON_WANT"]
values = data.get(list_key)
if not isinstance(values, list) or want not in values:
    print(f"smoke-package-b-readiness: expected {list_key} to contain {want}: {values}", file=sys.stderr)
    sys.exit(1)
PY
}

assert_json_list_omits() {
  local json_input="$1"
  local list_key="$2"
  local forbidden="$3"
  JSON_INPUT="${json_input}" JSON_LIST_KEY="${list_key}" JSON_FORBIDDEN="${forbidden}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["JSON_INPUT"])
list_key = os.environ["JSON_LIST_KEY"]
forbidden = os.environ["JSON_FORBIDDEN"]
values = data.get(list_key)
if not isinstance(values, list) or forbidden in values:
    print(f"smoke-package-b-readiness: expected {list_key} to omit {forbidden}: {values}", file=sys.stderr)
    sys.exit(1)
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

project_root="${AREAFLOW_PACKAGE_B_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"

status_before="$(file_fingerprint "${status_path}")"
workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
protected_status_before="$(git -C "${project_root}" status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json)"
worktree_status_before="$(git -C "${project_root}" status --short)"

echo "smoke-package-b-readiness: package-b-dirty-review"
dirty_output="$(bash scripts/audit-package-b-dirty-review.sh)"
echo "${dirty_output}"
assert_contains "${dirty_output}" "package-b-dirty-review: protected_dirty_output_sha256="
assert_contains "${dirty_output}" "package-b-dirty-review: worktree_dirty_output_sha256="
protected_hash="$(awk -F= '/protected_dirty_output_sha256=/{print $2; exit}' <<<"${dirty_output}")"
worktree_hash="$(awk -F= '/worktree_dirty_output_sha256=/{print $2; exit}' <<<"${dirty_output}")"

echo "smoke-package-b-readiness: package-b-readiness fail-closed without dirty review"
set +e
readiness_blocked="$(bash scripts/audit-package-b-readiness.sh 2>&1)"
readiness_blocked_rc=$?
set -e
echo "${readiness_blocked}"
if [[ ${readiness_blocked_rc} -eq 0 ]]; then
  echo "smoke-package-b-readiness: expected readiness to block without dirty review" >&2
  exit 1
fi
assert_contains "${readiness_blocked}" "package-b-readiness: status_projection_state=stable"
assert_contains "${readiness_blocked}" "package-b-readiness: blocked"
assert_contains "${readiness_blocked}" "package-b-readiness: allowed_write=scripts/areaflow_shim.py"
assert_contains "${readiness_blocked}" "package-b-readiness: forbidden=./task-loop run forwarding"

echo "smoke-package-b-readiness: package-b-authorization-packet --json fail-closed without dirty review"
set +e
packet_blocked="$(bash scripts/audit-package-b-authorization-packet.sh --json 2>&1)"
packet_blocked_rc=$?
set -e
echo "${packet_blocked}"
if [[ ${packet_blocked_rc} -eq 0 ]]; then
  echo "smoke-package-b-readiness: expected authorization packet to block without dirty review" >&2
  exit 1
fi
assert_contains "${packet_blocked}" '"scope": "package_b_read_only_shim_only"'
assert_contains "${packet_blocked}" '"status": "blocked_needs_protected_path_review"'
assert_contains "${packet_blocked}" '"required_authorization_phrase": "授权执行 Package B，只允许落地 AreaMatrix read-only shim，不允许 ./task-loop run 转发"'
assert_contains "${packet_blocked}" '"write_authorized_by_package_b": false'
assert_contains "${packet_blocked}" '"allows_task_loop_run_forwarding": false'
assert_contains "${packet_blocked}" '"modifies_areamatrix": false'

echo "smoke-package-b-readiness: package-b-authorization-packet --json dirty hash mismatch"
set +e
packet_mismatch="$(AREAFLOW_PACKAGE_B_REVIEWED_PROTECTED_OUTPUT_SHA256=0000000000000000000000000000000000000000000000000000000000000000 AREAFLOW_PACKAGE_B_REVIEWED_WORKTREE_OUTPUT_SHA256="${worktree_hash}" AREAFLOW_PACKAGE_B_DIRTY_REVIEWER=smoke-package-b bash scripts/audit-package-b-authorization-packet.sh --json 2>&1)"
packet_mismatch_rc=$?
set -e
echo "${packet_mismatch}"
if [[ ${packet_mismatch_rc} -eq 0 ]]; then
  echo "smoke-package-b-readiness: expected authorization packet to block on dirty hash mismatch" >&2
  exit 1
fi
assert_contains "${packet_mismatch}" '"protected_dirty_review_status": "mismatch"'
assert_contains "${packet_mismatch}" '"status": "blocked_needs_protected_path_review"'

echo "smoke-package-b-readiness: package-b-readiness accepted dirty review"
readiness_ready="$(AREAFLOW_PACKAGE_B_REVIEWED_PROTECTED_OUTPUT_SHA256="${protected_hash}" AREAFLOW_PACKAGE_B_REVIEWED_WORKTREE_OUTPUT_SHA256="${worktree_hash}" AREAFLOW_PACKAGE_B_DIRTY_REVIEWER=smoke-package-b bash scripts/audit-package-b-readiness.sh)"
echo "${readiness_ready}"
assert_contains "${readiness_ready}" "package-b-readiness: pass"
assert_contains "${readiness_ready}" "package-b-readiness: protected_dirty_review_status=accepted"
assert_contains "${readiness_ready}" "package-b-readiness: worktree_dirty_review_status=accepted"

echo "smoke-package-b-readiness: package-b-authorization-packet --json accepted dirty review"
packet_ready="$(AREAFLOW_PACKAGE_B_REVIEWED_PROTECTED_OUTPUT_SHA256="${protected_hash}" AREAFLOW_PACKAGE_B_REVIEWED_WORKTREE_OUTPUT_SHA256="${worktree_hash}" AREAFLOW_PACKAGE_B_DIRTY_REVIEWER=smoke-package-b bash scripts/audit-package-b-authorization-packet.sh --json)"
echo "${packet_ready}"
assert_contains "${packet_ready}" '"status": "ready_for_package_b_area_matrix_edit_authorization"'
assert_contains "${packet_ready}" '"allowed_writes": ['
assert_contains "${packet_ready}" '"scripts/areaflow_shim.py"'
assert_contains "${packet_ready}" '"scripts/task_loop/console.py"'
assert_contains "${packet_ready}" '"scripts/dev_tools/cli.py"'
assert_contains "${packet_ready}" '"scripts/task_loop/runner.py"'
assert_contains "${packet_ready}" '"workflow/README.md"'
assert_contains "${packet_ready}" '"read_only_prerequisites": ['
assert_contains "${packet_ready}" '".areaflow/status.json"'
assert_json_list_omits "${packet_ready}" "allowed_writes" ".areaflow/status.json"
assert_json_list_contains "${packet_ready}" "read_only_prerequisites" ".areaflow/status.json"
assert_contains "${packet_ready}" '"status_projection": {'
assert_contains "${packet_ready}" '"schema_state": "stable"'
assert_contains "${packet_ready}" '"write_authorized_by_package_b": false'
assert_contains "${packet_ready}" '"protected_dirty_review_status": "accepted"'
assert_contains "${packet_ready}" '"worktree_dirty_review_status": "accepted"'
assert_contains "${packet_ready}" '"approves_write": false'
assert_contains "${packet_ready}" '"allows_read_only_shim_files": true'
assert_contains "${packet_ready}" '"allows_status_projection_write": false'
assert_contains "${packet_ready}" '"allows_workflow_versions": false'
assert_contains "${packet_ready}" '"allows_execution_write": false'
assert_contains "${packet_ready}" '"allows_task_loop_run_forwarding": false'
assert_contains "${packet_ready}" '"runs_engine": false'
assert_contains "${packet_ready}" '"resolves_secret": false'
assert_contains "${packet_ready}" '"uses_network": false'
assert_contains "${packet_ready}" '"creates_git_checkpoint": false'
assert_not_contains "${packet_ready}" '"workflow/versions/**"'

status_after="$(file_fingerprint "${status_path}")"
workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
protected_status_after="$(git -C "${project_root}" status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json)"
worktree_status_after="$(git -C "${project_root}" status --short)"

if [[ "${status_before}" != "${status_after}" ]]; then
  echo "smoke-package-b-readiness: status projection changed unexpectedly" >&2
  exit 1
fi
if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
  echo "smoke-package-b-readiness: workflow README changed unexpectedly" >&2
  exit 1
fi
if [[ "${protected_status_before}" != "${protected_status_after}" ]]; then
  echo "smoke-package-b-readiness: protected status changed unexpectedly" >&2
  exit 1
fi
if [[ "${worktree_status_before}" != "${worktree_status_after}" ]]; then
  echo "smoke-package-b-readiness: worktree status changed unexpectedly" >&2
  exit 1
fi

echo "smoke-package-b-readiness: pass"
