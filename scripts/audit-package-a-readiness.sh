#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_A_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
schema_path="${AREAFLOW_PACKAGE_A_SCHEMA:-schemas/status-projection.schema.json}"
status_path="${project_root}/.areaflow/status.json"
reviewed_dirty_output_hash="${AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256:-}"
dirty_reviewer="${AREAFLOW_PACKAGE_A_DIRTY_REVIEWER:-}"

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

echo "package-a-readiness: checking AreaMatrix Package A prerequisites"
echo "package-a-readiness: project_root=${project_root}"
echo "package-a-readiness: target=${status_path}"

if [[ ! -d "${project_root}" ]]; then
  echo "package-a-readiness: blocked missing project root: ${project_root}" >&2
  exit 1
fi

if [[ ! -f "${schema_path}" ]]; then
  echo "package-a-readiness: blocked missing schema: ${schema_path}" >&2
  exit 1
fi

if [[ ! -f "${status_path}" ]]; then
  echo "package-a-readiness: blocked missing status projection: ${status_path}" >&2
  blocked=1
else
  target_preimage_sha256="$(shasum -a 256 "${status_path}" | awk '{print $1}')"
  target_preimage_size_bytes="$(wc -c <"${status_path}" | awk '{print $1}')"
  echo "package-a-readiness: target_preimage_exists=true"
  echo "package-a-readiness: target_preimage_sha256=${target_preimage_sha256}"
  echo "package-a-readiness: target_preimage_size_bytes=${target_preimage_size_bytes}"
  echo "package-a-readiness: checking status projection schema state"
  if validation_output="$(python3 scripts/validate-status-projection-schema.py "${schema_path}" "${status_path}" 2>&1)"; then
    echo "${validation_output}"
    echo "package-a-readiness: status_projection_state=stable_already"
  else
    echo "package-a-readiness: status_projection_state=legacy_needs_package_a"
    while IFS= read -r line; do
      echo "package-a-readiness: schema: ${line}"
    done <<<"${validation_output}"
  fi
fi

echo "package-a-readiness: checking protected AreaMatrix paths"
git_status_output="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
if [[ -n "${git_status_output}" ]]; then
  dirty_output_hash="$(printf "%s" "${git_status_output}" | shasum -a 256 | awk '{print $1}')"
  dirty_path_count="$(printf "%s\n" "${git_status_output}" | awk 'NF {count++} END {print count+0}')"
  dirty_review_status="required"
  if [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" == "${dirty_output_hash}" && -n "${dirty_reviewer}" ]]; then
    dirty_review_status="accepted"
  elif [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" != "${dirty_output_hash}" ]]; then
    dirty_review_status="mismatch"
  elif [[ -n "${reviewed_dirty_output_hash}" && -z "${dirty_reviewer}" ]]; then
    dirty_review_status="missing_reviewer"
  fi
  if [[ "${dirty_review_status}" != "accepted" ]]; then
    blocked=1
  else
    :
  fi
  if [[ "${dirty_review_status}" == "accepted" ]]; then
    echo "package-a-readiness: protected paths dirty but exact dirty hash was reviewed"
    echo "package-a-readiness: protected_path_rule_count=${#protected_paths[@]}"
    echo "package-a-readiness: dirty_path_count=${dirty_path_count}"
    echo "package-a-readiness: dirty_review_status=accepted"
    echo "package-a-readiness: dirty_reviewer=${dirty_reviewer}"
  else
    echo "package-a-readiness: blocked protected paths are not clean" >&2
    echo "package-a-readiness: protected_path_rule_count=${#protected_paths[@]}" >&2
    echo "package-a-readiness: dirty_path_count=${dirty_path_count}" >&2
    echo "package-a-readiness: dirty_review_status=${dirty_review_status}" >&2
    if [[ "${dirty_review_status}" == "mismatch" ]]; then
      echo "package-a-readiness: reviewed_dirty_output_sha256=${reviewed_dirty_output_hash}" >&2
    fi
  fi
  echo "package-a-readiness: dirty_output_sha256=${dirty_output_hash}" >&2
  while IFS= read -r line; do
    echo "package-a-readiness: protected-path: ${line}" >&2
  done <<<"${git_status_output}"
  if [[ "${dirty_review_status}" != "accepted" ]]; then
    echo "package-a-readiness: dirty_review_command=bash scripts/audit-package-a-dirty-review.sh" >&2
    echo "package-a-readiness: dirty_review_input=AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<dirty_output_sha256> AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>" >&2
  fi
else
  echo "package-a-readiness: protected paths clean"
  echo "package-a-readiness: protected_path_rule_count=${#protected_paths[@]}"
  echo "package-a-readiness: dirty_path_count=0"
fi

if (( blocked )); then
  echo "package-a-readiness: blocked"
  echo "package-a-readiness: next=settle or explicitly review the protected-path dirty state, then rerun this audit before Package A"
  exit 1
fi

echo "package-a-readiness: pass"
echo "package-a-readiness: next=Package A may still require explicit narrow write approval before status-projection-apply"
