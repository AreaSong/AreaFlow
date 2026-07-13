#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_B_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"

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

if [[ ! -d "${project_root}" ]]; then
  echo "package-b-dirty-review: blocked missing project root: ${project_root}" >&2
  exit 1
fi

protected_status="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
worktree_status="$(git -C "${project_root}" status --short)"
protected_hash="$(printf "%s" "${protected_status}" | shasum -a 256 | awk '{print $1}')"
worktree_hash="$(printf "%s" "${worktree_status}" | shasum -a 256 | awk '{print $1}')"
protected_count="$(printf "%s\n" "${protected_status}" | awk 'NF {count++} END {print count+0}')"
worktree_count="$(printf "%s\n" "${worktree_status}" | awk 'NF {count++} END {print count+0}')"

echo "package-b-dirty-review: project_root=${project_root}"
echo "package-b-dirty-review: protected_path_count=${#protected_paths[@]}"
echo "package-b-dirty-review: protected_dirty_path_count=${protected_count}"
echo "package-b-dirty-review: protected_dirty_output_sha256=${protected_hash}"
echo "package-b-dirty-review: worktree_dirty_path_count=${worktree_count}"
echo "package-b-dirty-review: worktree_dirty_output_sha256=${worktree_hash}"
if [[ -n "${protected_status}" ]]; then
  while IFS= read -r line; do
    echo "package-b-dirty-review: protected-path: ${line}"
  done <<<"${protected_status}"
fi
if [[ -n "${worktree_status}" ]]; then
  while IFS= read -r line; do
    echo "package-b-dirty-review: worktree-path: ${line}"
  done <<<"${worktree_status}"
fi
echo "package-b-dirty-review: reviewed_protected_hash_input=AREAFLOW_PACKAGE_B_REVIEWED_PROTECTED_OUTPUT_SHA256=<protected_dirty_output_sha256>"
echo "package-b-dirty-review: reviewed_worktree_hash_input=AREAFLOW_PACKAGE_B_REVIEWED_WORKTREE_OUTPUT_SHA256=<worktree_dirty_output_sha256>"
echo "package-b-dirty-review: reviewer_input=AREAFLOW_PACKAGE_B_DIRTY_REVIEWER=<reviewer>"
echo "package-b-dirty-review: note=this script does not approve, record, or mutate AreaMatrix"
