#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_A_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"

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
  echo "package-a-dirty-review: blocked missing project root: ${project_root}" >&2
  exit 1
fi

git_status_output="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
dirty_output_hash="$(printf "%s" "${git_status_output}" | shasum -a 256 | awk '{print $1}')"
dirty_path_count="$(printf "%s\n" "${git_status_output}" | awk 'NF {count++} END {print count+0}')"

echo "package-a-dirty-review: project_root=${project_root}"
echo "package-a-dirty-review: protected_path_count=${#protected_paths[@]}"
echo "package-a-dirty-review: protected_path_rule_count=${#protected_paths[@]}"
echo "package-a-dirty-review: dirty_path_count=${dirty_path_count}"
echo "package-a-dirty-review: dirty_output_sha256=${dirty_output_hash}"

if [[ -z "${git_status_output}" ]]; then
  echo "package-a-dirty-review: status=clean"
  exit 0
fi

echo "package-a-dirty-review: status=dirty"
while IFS= read -r line; do
  [[ -z "${line}" ]] && continue
  path="${line:3}"
  echo "package-a-dirty-review: protected-path: ${line}"
  echo "package-a-dirty-review: touched-path: ${path}"
done <<<"${git_status_output}"

cat <<'MSG'
package-a-dirty-review: authorization_input=dirty_output_sha256 plus the exact protected-path lines above
package-a-dirty-review: reviewed_hash_input=AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<dirty_output_sha256>
package-a-dirty-review: reviewer_input=AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>
package-a-dirty-review: note=this script does not approve or record the dirty state; it only produces review input
MSG
