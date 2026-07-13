#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_B_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
schema_path="${AREAFLOW_PACKAGE_B_SCHEMA:-schemas/status-projection.schema.json}"
status_path="${project_root}/.areaflow/status.json"
reviewed_protected_hash="${AREAFLOW_PACKAGE_B_REVIEWED_PROTECTED_OUTPUT_SHA256:-}"
reviewed_worktree_hash="${AREAFLOW_PACKAGE_B_REVIEWED_WORKTREE_OUTPUT_SHA256:-}"
dirty_reviewer="${AREAFLOW_PACKAGE_B_DIRTY_REVIEWER:-}"

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

blocked=0

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

echo "package-b-readiness: checking AreaMatrix Package B read-only shim prerequisites"
echo "package-b-readiness: project_root=${project_root}"

if [[ ! -d "${project_root}" ]]; then
  echo "package-b-readiness: blocked missing project root: ${project_root}" >&2
  exit 1
fi

if [[ ! -f "${schema_path}" ]]; then
  echo "package-b-readiness: blocked missing schema: ${schema_path}" >&2
  exit 1
fi

echo "package-b-readiness: checking required existing files"
missing_required=()
for relative in "${required_existing_files[@]}"; do
  if [[ ! -e "${project_root}/${relative}" ]]; then
    missing_required+=("${relative}")
  fi
done
if [[ ${#missing_required[@]} -gt 0 ]]; then
  blocked=1
  for relative in "${missing_required[@]}"; do
    echo "package-b-readiness: missing_required_file=${relative}" >&2
  done
else
  echo "package-b-readiness: required_existing_files=present"
fi

echo "package-b-readiness: checking status projection schema"
if validation_output="$(python3 scripts/validate-status-projection-schema.py "${schema_path}" "${status_path}" 2>&1)"; then
  echo "${validation_output}"
  echo "package-b-readiness: status_projection_state=stable"
else
  blocked=1
  echo "package-b-readiness: status_projection_state=blocked_needs_package_a" >&2
  while IFS= read -r line; do
    echo "package-b-readiness: schema: ${line}" >&2
  done <<<"${validation_output}"
fi

protected_status="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
worktree_status="$(git -C "${project_root}" status --short)"
protected_hash="$(printf "%s" "${protected_status}" | shasum -a 256 | awk '{print $1}')"
worktree_hash="$(printf "%s" "${worktree_status}" | shasum -a 256 | awk '{print $1}')"
protected_count="$(printf "%s\n" "${protected_status}" | awk 'NF {count++} END {print count+0}')"
worktree_count="$(printf "%s\n" "${worktree_status}" | awk 'NF {count++} END {print count+0}')"
protected_review_status="$(review_status "${protected_status}" "${reviewed_protected_hash}" "${protected_hash}")"
worktree_review_status="$(review_status "${worktree_status}" "${reviewed_worktree_hash}" "${worktree_hash}")"

echo "package-b-readiness: protected_path_rule_count=${#protected_paths[@]}"
echo "package-b-readiness: protected_dirty_path_count=${protected_count}"
echo "package-b-readiness: protected_dirty_output_sha256=${protected_hash}"
echo "package-b-readiness: protected_dirty_review_status=${protected_review_status}"
echo "package-b-readiness: worktree_dirty_path_count=${worktree_count}"
echo "package-b-readiness: worktree_dirty_output_sha256=${worktree_hash}"
echo "package-b-readiness: worktree_dirty_review_status=${worktree_review_status}"
if [[ "${protected_review_status}" != "clean" && "${protected_review_status}" != "accepted" ]]; then
  blocked=1
fi
if [[ "${worktree_review_status}" != "clean" && "${worktree_review_status}" != "accepted" ]]; then
  blocked=1
fi
if [[ -n "${dirty_reviewer}" ]]; then
  echo "package-b-readiness: dirty_reviewer=${dirty_reviewer}"
fi
if [[ -n "${protected_status}" ]]; then
  while IFS= read -r line; do
    echo "package-b-readiness: protected-path: ${line}" >&2
  done <<<"${protected_status}"
fi
if [[ -n "${worktree_status}" ]]; then
  while IFS= read -r line; do
    echo "package-b-readiness: worktree-path: ${line}" >&2
  done <<<"${worktree_status}"
fi

echo "package-b-readiness: allowed_write_count=${#allowed_writes[@]}"
for relative in "${allowed_writes[@]}"; do
  echo "package-b-readiness: allowed_write=${relative}"
done
echo "package-b-readiness: forbidden=./task-loop run forwarding"
echo "package-b-readiness: forbidden=workflow/versions/**"
echo "package-b-readiness: forbidden=progress/log/checkpoint rewrite"

if (( blocked )); then
  echo "package-b-readiness: blocked"
  echo "package-b-readiness: next=settle or explicitly review dirty state, then request explicit Package B edit authorization"
  exit 1
fi

echo "package-b-readiness: pass"
echo "package-b-readiness: next=show Package B authorization packet and wait for explicit AreaMatrix edit approval"
