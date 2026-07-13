#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_B_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
schema_path="${AREAFLOW_PACKAGE_B_SCHEMA:-schemas/status-projection.schema.json}"
status_path="${project_root}/.areaflow/status.json"
reviewed_protected_hash="${AREAFLOW_PACKAGE_B_REVIEWED_PROTECTED_OUTPUT_SHA256:-}"
reviewed_worktree_hash="${AREAFLOW_PACKAGE_B_REVIEWED_WORKTREE_OUTPUT_SHA256:-}"
dirty_reviewer="${AREAFLOW_PACKAGE_B_DIRTY_REVIEWER:-}"
output_json=0
authorization_phrase="授权执行 Package B，只允许落地 AreaMatrix read-only shim，不允许 ./task-loop run 转发"

if [[ "${1:-}" == "--json" ]]; then
  output_json=1
fi

required_existing_files=(
  "scripts/task_loop/console.py"
  "scripts/dev_tools/cli.py"
  "scripts/task_loop/runner.py"
  "workflow/README.md"
  ".areaflow/status.json"
)

allowed_writes=(
  "scripts/areaflow_shim.py"
  "scripts/task_loop/console.py"
  "scripts/dev_tools/cli.py"
  "scripts/task_loop/runner.py"
  "workflow/README.md"
)

read_only_prerequisites=(
  ".areaflow/status.json"
)

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

forbidden_actions=(
  "write workflow/versions/**"
  "write execution/progress/logs/checkpoints"
  "write AreaMatrix source outside shim files"
  "forward ./task-loop run"
  "run promotion apply"
  "run native doctor implicitly"
  "create git checkpoint"
  "run engine, resolve secret, use network, publish, restore"
)

required_preflight=(
  "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
  "make smoke-docker-shim-authorization-preflight"
  "AREAFLOW_READONLY_REVIEWED_DIRTY_OUTPUT_SHA256=<hash> AREAFLOW_READONLY_DIRTY_REVIEWER=<reviewer> make smoke-docker-areamatrix-readonly"
  "git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"
)

post_edit_verification=(
  "cd /Users/as/Ai-Project/project/AreaMatrix && ./dev workflow status"
  "cd /Users/as/Ai-Project/project/AreaMatrix && ./dev workflow doctor"
  "cd /Users/as/Ai-Project/project/AreaMatrix && ./dev workflow init --version shim-smoke"
  "cd /Users/as/Ai-Project/project/AreaMatrix && ./dev workflow open"
  "cd /Users/as/Ai-Project/project/AreaMatrix && ./task-loop status"
  "cd /Users/as/Ai-Project/project/AreaMatrix && verify ./task-loop run returns blocked and does not start legacy runner or write progress/log/checkpoint"
  "python3 /Users/as/Ai-Project/project/AreaFlow/scripts/validate-status-projection-schema.py /Users/as/Ai-Project/project/AreaFlow/schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
  "git -C /Users/as/Ai-Project/project/AreaMatrix diff --check -- scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py workflow/README.md scripts/areaflow_shim.py .areaflow/status.json"
)

rollback_scope=(
  "restore captured preimage for modified shim files only"
  "delete scripts/areaflow_shim.py only if Package B created it"
  "do not modify workflow/versions/**, progress.json, logs, checkpoints, release evidence, source files, or user files"
  "do not delete AreaFlow command/event/audit/run/artifact rows"
)

review_status() {
  local output="$1"
  local reviewed_hash="$2"
  local actual_hash="$3"
  if [[ -z "${output}" ]]; then
    echo "clean"
  elif [[ -n "${reviewed_hash}" && "${reviewed_hash}" == "${actual_hash}" && -n "${dirty_reviewer}" ]]; then
    echo "accepted"
  elif [[ -n "${reviewed_hash}" && "${reviewed_hash}" != "${actual_hash}" ]]; then
    echo "mismatch"
  elif [[ -n "${reviewed_hash}" && -z "${dirty_reviewer}" ]]; then
    echo "missing_reviewer"
  else
    echo "required"
  fi
}

status_sha256=""
status_size_bytes=0
status_schema_state="missing"
if [[ -f "${status_path}" ]]; then
  status_sha256="$(shasum -a 256 "${status_path}" | awk '{print $1}')"
  status_size_bytes="$(wc -c <"${status_path}" | awk '{print $1}')"
  if python3 scripts/validate-status-projection-schema.py "${schema_path}" "${status_path}" >/dev/null 2>&1; then
    status_schema_state="stable"
  else
    status_schema_state="blocked_needs_package_a"
  fi
fi

missing_required=()
for relative in "${required_existing_files[@]}"; do
  if [[ ! -e "${project_root}/${relative}" ]]; then
    missing_required+=("${relative}")
  fi
done

protected_status=""
worktree_status=""
protected_hash=""
worktree_hash=""
protected_count=0
worktree_count=0
protected_review_status="missing_project"
worktree_review_status="missing_project"
if [[ -d "${project_root}" ]]; then
  protected_status="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
  worktree_status="$(git -C "${project_root}" status --short)"
  protected_hash="$(printf "%s" "${protected_status}" | shasum -a 256 | awk '{print $1}')"
  worktree_hash="$(printf "%s" "${worktree_status}" | shasum -a 256 | awk '{print $1}')"
  protected_count="$(printf "%s\n" "${protected_status}" | awk 'NF {count++} END {print count+0}')"
  worktree_count="$(printf "%s\n" "${worktree_status}" | awk 'NF {count++} END {print count+0}')"
  protected_review_status="$(review_status "${protected_status}" "${reviewed_protected_hash}" "${protected_hash}")"
  worktree_review_status="$(review_status "${worktree_status}" "${reviewed_worktree_hash}" "${worktree_hash}")"
fi

packet_status="ready_for_package_b_area_matrix_edit_authorization"
if [[ ! -d "${project_root}" ]]; then
  packet_status="blocked_missing_project_root"
elif [[ "${status_schema_state}" != "stable" ]]; then
  packet_status="blocked_needs_package_a_status_projection"
elif [[ ${#missing_required[@]} -gt 0 ]]; then
  packet_status="blocked_missing_required_files"
elif [[ "${protected_review_status}" != "clean" && "${protected_review_status}" != "accepted" ]]; then
  packet_status="blocked_needs_protected_path_review"
elif [[ "${worktree_review_status}" != "clean" && "${worktree_review_status}" != "accepted" ]]; then
  packet_status="blocked_needs_worktree_review"
fi

if (( output_json )); then
  export PACKAGE_B_PACKET_STATUS="${packet_status}"
  export PACKAGE_B_PROJECT_ROOT="${project_root}"
  export PACKAGE_B_AUTHORIZATION_PHRASE="${authorization_phrase}"
  export PACKAGE_B_STATUS_SCHEMA_STATE="${status_schema_state}"
  export PACKAGE_B_STATUS_SHA256="${status_sha256}"
  export PACKAGE_B_STATUS_SIZE_BYTES="${status_size_bytes}"
  export PACKAGE_B_PROTECTED_HASH="${protected_hash}"
  export PACKAGE_B_WORKTREE_HASH="${worktree_hash}"
  export PACKAGE_B_PROTECTED_COUNT="${protected_count}"
  export PACKAGE_B_WORKTREE_COUNT="${worktree_count}"
  export PACKAGE_B_PROTECTED_REVIEW_STATUS="${protected_review_status}"
  export PACKAGE_B_WORKTREE_REVIEW_STATUS="${worktree_review_status}"
  export PACKAGE_B_DIRTY_REVIEWER="${dirty_reviewer}"
  export PACKAGE_B_REVIEWED_PROTECTED_HASH="${reviewed_protected_hash}"
  export PACKAGE_B_REVIEWED_WORKTREE_HASH="${reviewed_worktree_hash}"
  export PACKAGE_B_ALLOWED_WRITES="$(printf "%s\n" "${allowed_writes[@]}")"
  export PACKAGE_B_READ_ONLY_PREREQUISITES="$(printf "%s\n" "${read_only_prerequisites[@]}")"
  export PACKAGE_B_PROTECTED_PATHS="$(printf "%s\n" "${protected_paths[@]}")"
  export PACKAGE_B_FORBIDDEN_ACTIONS="$(printf "%s\n" "${forbidden_actions[@]}")"
  export PACKAGE_B_REQUIRED_PREFLIGHT="$(printf "%s\n" "${required_preflight[@]}")"
  export PACKAGE_B_POST_EDIT_VERIFICATION="$(printf "%s\n" "${post_edit_verification[@]}")"
  export PACKAGE_B_ROLLBACK_SCOPE="$(printf "%s\n" "${rollback_scope[@]}")"
  if [[ ${#missing_required[@]} -gt 0 ]]; then
    export PACKAGE_B_MISSING_REQUIRED="$(printf "%s\n" "${missing_required[@]}")"
  else
    export PACKAGE_B_MISSING_REQUIRED=""
  fi
  export PACKAGE_B_PROTECTED_STATUS="${protected_status}"
  export PACKAGE_B_WORKTREE_STATUS="${worktree_status}"
  python3 - <<'PY'
import json
import os


def lines(name):
    value = os.environ.get(name, "")
    return [line for line in value.splitlines() if line]


payload = {
    "status": os.environ["PACKAGE_B_PACKET_STATUS"],
    "scope": "package_b_read_only_shim_only",
    "project_root": os.environ["PACKAGE_B_PROJECT_ROOT"],
    "intent": "land AreaMatrix read-only compatibility shim after explicit approval",
    "required_authorization_phrase": os.environ["PACKAGE_B_AUTHORIZATION_PHRASE"],
    "user_authorization_phrase": os.environ["PACKAGE_B_AUTHORIZATION_PHRASE"],
    "allowed_writes": lines("PACKAGE_B_ALLOWED_WRITES"),
    "read_only_prerequisites": lines("PACKAGE_B_READ_ONLY_PREREQUISITES"),
    "protected_paths": lines("PACKAGE_B_PROTECTED_PATHS"),
    "forbidden_actions": lines("PACKAGE_B_FORBIDDEN_ACTIONS"),
    "required_preflight": lines("PACKAGE_B_REQUIRED_PREFLIGHT"),
    "post_edit_verification": lines("PACKAGE_B_POST_EDIT_VERIFICATION"),
    "rollback_scope": lines("PACKAGE_B_ROLLBACK_SCOPE"),
    "missing_required_files": lines("PACKAGE_B_MISSING_REQUIRED"),
    "status_projection": {
        "target": ".areaflow/status.json",
        "schema_state": os.environ["PACKAGE_B_STATUS_SCHEMA_STATE"],
        "sha256": os.environ["PACKAGE_B_STATUS_SHA256"],
        "size_bytes": int(os.environ["PACKAGE_B_STATUS_SIZE_BYTES"] or "0"),
        "write_authorized_by_package_b": False,
    },
    "dirty_review": {
        "protected_path_state": "dirty" if int(os.environ["PACKAGE_B_PROTECTED_COUNT"]) else "clean",
        "protected_dirty_path_count": int(os.environ["PACKAGE_B_PROTECTED_COUNT"]),
        "protected_dirty_output_sha256": os.environ["PACKAGE_B_PROTECTED_HASH"],
        "protected_dirty_review_status": os.environ["PACKAGE_B_PROTECTED_REVIEW_STATUS"],
        "reviewed_protected_dirty_output_sha256": os.environ["PACKAGE_B_REVIEWED_PROTECTED_HASH"],
        "worktree_state": "dirty" if int(os.environ["PACKAGE_B_WORKTREE_COUNT"]) else "clean",
        "worktree_dirty_path_count": int(os.environ["PACKAGE_B_WORKTREE_COUNT"]),
        "worktree_dirty_output_sha256": os.environ["PACKAGE_B_WORKTREE_HASH"],
        "worktree_dirty_review_status": os.environ["PACKAGE_B_WORKTREE_REVIEW_STATUS"],
        "reviewed_worktree_dirty_output_sha256": os.environ["PACKAGE_B_REVIEWED_WORKTREE_HASH"],
        "dirty_reviewer": os.environ["PACKAGE_B_DIRTY_REVIEWER"],
        "protected_path_lines": lines("PACKAGE_B_PROTECTED_STATUS"),
        "worktree_lines": lines("PACKAGE_B_WORKTREE_STATUS"),
    },
    "safety_facts": {
        "modifies_areamatrix": False,
        "approves_write": False,
        "allows_read_only_shim_files": True,
        "allows_status_projection_write": False,
        "allows_workflow_readme_manual_link": True,
        "allows_workflow_versions": False,
        "allows_execution_write": False,
        "allows_task_loop_run_forwarding": False,
        "allows_native_doctor_without_explicit_allow_native": False,
        "runs_engine": False,
        "resolves_secret": False,
        "uses_network": False,
        "publishes_release": False,
        "creates_git_checkpoint": False,
    },
}
print(json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True))
PY
else
  echo "package-b-authorization-packet: status=${packet_status}"
  echo "package-b-authorization-packet: project_root=${project_root}"
  echo "package-b-authorization-packet: required_authorization_phrase=${authorization_phrase}"
  echo "package-b-authorization-packet: status_projection_state=${status_schema_state}"
  echo "package-b-authorization-packet: protected_dirty_review_status=${protected_review_status}"
  echo "package-b-authorization-packet: worktree_dirty_review_status=${worktree_review_status}"
  for relative in "${allowed_writes[@]}"; do
    echo "package-b-authorization-packet: allowed_write=${relative}"
  done
  for action in "${forbidden_actions[@]}"; do
    echo "package-b-authorization-packet: forbidden=${action}"
  done
fi

if [[ "${packet_status}" != "ready_for_package_b_area_matrix_edit_authorization" ]]; then
  exit 1
fi
