#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_A_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
project_key="${AREAFLOW_PACKAGE_A_PROJECT_KEY:-areamatrix}"
config_path="${AREAFLOW_PACKAGE_A_CONFIG:-examples/areamatrix/areaflow.yaml}"
schema_path="${AREAFLOW_PACKAGE_A_SCHEMA:-schemas/status-projection.schema.json}"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
reviewed_dirty_output_hash="${AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256:-}"
dirty_reviewer="${AREAFLOW_PACKAGE_A_DIRTY_REVIEWER:-}"
output_json=0

if [[ "${1:-}" == "--json" ]]; then
  output_json=1
fi

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

file_fingerprint() {
  areaflow_file_fingerprint "$@"
}

protected_path_git_status() {
  areaflow_protected_path_git_status "${project_root}" "${protected_paths[@]}"
}

protected_path_fingerprint() {
  areaflow_protected_path_fingerprint "${project_root}" ".areaflow/status.json" "${protected_paths[@]}"
}

json_get() {
  local expression="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); print(${expression})"
}

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "package-a-source-hash: blocked missing AREAFLOW_DATABASE_URL" >&2
  exit 1
fi

if [[ ! -d "${project_root}" ]]; then
  echo "package-a-source-hash: blocked missing project root: ${project_root}" >&2
  exit 1
fi

if [[ ! -f "${config_path}" ]]; then
  echo "package-a-source-hash: blocked missing config: ${config_path}" >&2
  exit 1
fi

if [[ ! -f "${schema_path}" ]]; then
  echo "package-a-source-hash: blocked missing schema: ${schema_path}" >&2
  exit 1
fi

if [[ ! -f "${status_path}" ]]; then
  echo "package-a-source-hash: blocked missing status projection: ${status_path}" >&2
  exit 1
fi

status_before="$(file_fingerprint "${status_path}")"
workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_before="$(protected_path_fingerprint)"
git_status_output="$(protected_path_git_status)"
dirty_output_hash="$(printf "%s" "${git_status_output}" | shasum -a 256 | awk '{print $1}')"
dirty_review_status="clean"
if [[ -n "${git_status_output}" ]]; then
  dirty_review_status="required"
  if [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" == "${dirty_output_hash}" && -n "${dirty_reviewer}" ]]; then
    dirty_review_status="accepted"
  elif [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" != "${dirty_output_hash}" ]]; then
    dirty_review_status="mismatch"
  elif [[ -n "${reviewed_dirty_output_hash}" && -z "${dirty_reviewer}" ]]; then
    dirty_review_status="missing_reviewer"
  fi
fi

if [[ "${dirty_review_status}" == "required" || "${dirty_review_status}" == "mismatch" || "${dirty_review_status}" == "missing_reviewer" ]]; then
  echo "package-a-source-hash: blocked protected paths dirty without exact review" >&2
  echo "package-a-source-hash: dirty_review_status=${dirty_review_status}" >&2
  echo "package-a-source-hash: dirty_output_sha256=${dirty_output_hash}" >&2
  echo "package-a-source-hash: dirty_review_input=AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<dirty_output_sha256> AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>" >&2
  while IFS= read -r line; do
    [[ -z "${line}" ]] && continue
    echo "package-a-source-hash: protected-path: ${line}" >&2
  done <<<"${git_status_output}"
  exit 1
fi

go run ./cmd/areaflow migrate up >/dev/null
go run ./cmd/areaflow project add --config "${config_path}" >/dev/null
go run ./cmd/areaflow project import "${project_key}" >/dev/null
packet_json="$(go run ./cmd/areaflow project status-projection-apply-packet "${project_key}" --json)"

source_hash="$(json_get 'data["packet"]["source_hash"]' <<<"${packet_json}")"
expected_before_exists="$(json_get 'str(data["packet"]["expected_before_exists"]).lower()' <<<"${packet_json}")"
expected_before_sha256="$(json_get 'data["packet"].get("expected_before_sha256", "")' <<<"${packet_json}")"
expected_before_size="$(json_get 'data["packet"]["expected_before_size"]' <<<"${packet_json}")"
schema_uri="$(json_get 'data["packet"]["schema_uri"]' <<<"${packet_json}")"
validator_preflight="$(json_get 'data["packet"]["validator_preflight"]' <<<"${packet_json}")"
protected_path_check="$(json_get 'data["packet"]["protected_path_check"]' <<<"${packet_json}")"
protected_path_fingerprint_sha256="$(json_get 'data["packet"].get("protected_path_fingerprint_sha256", "")' <<<"${packet_json}")"
rollback_action="$(json_get 'data["packet"]["rollback_action"]' <<<"${packet_json}")"
accepted_preimage_schema_status="$(json_get 'data["packet"]["accept_preimage_schema"]' <<<"${packet_json}")"

status_after="$(file_fingerprint "${status_path}")"
workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_after="$(protected_path_fingerprint)"
git_status_after="$(protected_path_git_status)"
dirty_output_hash_after="$(printf "%s" "${git_status_after}" | shasum -a 256 | awk '{print $1}')"

status="ready_for_package_a_packet_binding"
blockers=()
if [[ -z "${source_hash}" ]]; then
  status="blocked"
  blockers+=("source_hash_missing")
fi
if [[ "${expected_before_exists}" != "true" ]]; then
  status="blocked"
  blockers+=("expected_before_exists_not_true")
fi
if [[ -z "${expected_before_sha256}" ]]; then
  status="blocked"
  blockers+=("expected_before_sha256_missing")
fi
if [[ -z "${protected_path_fingerprint_sha256}" ]]; then
  status="blocked"
  blockers+=("protected_path_fingerprint_sha256_missing")
fi
if [[ "${status_before}" != "${status_after}" ]]; then
  status="blocked"
  blockers+=("status_projection_fingerprint_changed")
fi
if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
  status="blocked"
  blockers+=("workflow_readme_fingerprint_changed")
fi
if [[ "${protected_path_fingerprint_before}" != "${protected_path_fingerprint_after}" ]]; then
  status="blocked"
  blockers+=("protected_path_fingerprint_changed")
fi
if [[ "${dirty_output_hash}" != "${dirty_output_hash_after}" ]]; then
  status="blocked"
  blockers+=("protected_path_dirty_hash_changed")
fi

if (( output_json )); then
  export PACKAGE_A_SOURCE_HASH_STATUS="${status}"
  export PACKAGE_A_SOURCE_HASH_BLOCKERS="$(printf "%s\n" "${blockers[@]:-}")"
  export PACKAGE_A_PROJECT_ROOT="${project_root}"
  export PACKAGE_A_PROJECT_KEY="${project_key}"
  export PACKAGE_A_SOURCE_HASH="${source_hash}"
  export PACKAGE_A_DIRTY_REVIEW_STATUS="${dirty_review_status}"
  export PACKAGE_A_DIRTY_OUTPUT_SHA256="${dirty_output_hash}"
  export PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${reviewed_dirty_output_hash}"
  export PACKAGE_A_DIRTY_REVIEWER="${dirty_reviewer}"
  export PACKAGE_A_EXPECTED_BEFORE_EXISTS="${expected_before_exists}"
  export PACKAGE_A_EXPECTED_BEFORE_SHA256="${expected_before_sha256}"
  export PACKAGE_A_EXPECTED_BEFORE_SIZE="${expected_before_size}"
  export PACKAGE_A_SCHEMA_URI="${schema_uri}"
  export PACKAGE_A_VALIDATOR_PREFLIGHT="${validator_preflight}"
  export PACKAGE_A_PROTECTED_PATH_CHECK="${protected_path_check}"
  export PACKAGE_A_PROTECTED_PATH_FINGERPRINT_SHA256="${protected_path_fingerprint_sha256}"
  export PACKAGE_A_ROLLBACK_ACTION="${rollback_action}"
  export PACKAGE_A_ACCEPT_PREIMAGE_SCHEMA="${accepted_preimage_schema_status}"
  export PACKAGE_A_STATUS_BEFORE="${status_before}"
  export PACKAGE_A_STATUS_AFTER="${status_after}"
  export PACKAGE_A_WORKFLOW_README_BEFORE="${workflow_readme_before}"
  export PACKAGE_A_WORKFLOW_README_AFTER="${workflow_readme_after}"
  export PACKAGE_A_PROTECTED_PATH_FINGERPRINT_BEFORE="${protected_path_fingerprint_before}"
  export PACKAGE_A_PROTECTED_PATH_FINGERPRINT_AFTER="${protected_path_fingerprint_after}"
  export PACKAGE_A_DIRTY_OUTPUT_SHA256_AFTER="${dirty_output_hash_after}"
  python3 - <<'PY'
import json
import os

blockers = [line for line in os.environ["PACKAGE_A_SOURCE_HASH_BLOCKERS"].splitlines() if line]
source_hash = os.environ["PACKAGE_A_SOURCE_HASH"]
payload = {
    "status": os.environ["PACKAGE_A_SOURCE_HASH_STATUS"],
    "blockers": blockers,
    "project_root": os.environ["PACKAGE_A_PROJECT_ROOT"],
    "project_key": os.environ["PACKAGE_A_PROJECT_KEY"],
    "source_hash": source_hash,
    "source_hash_env": f"AREAFLOW_PACKAGE_A_SOURCE_HASH={source_hash}" if source_hash else "",
    "dirty_review_status": os.environ["PACKAGE_A_DIRTY_REVIEW_STATUS"],
    "dirty_output_sha256": os.environ["PACKAGE_A_DIRTY_OUTPUT_SHA256"],
    "reviewed_dirty_output_sha256": os.environ["PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256"],
    "dirty_reviewer": os.environ["PACKAGE_A_DIRTY_REVIEWER"],
    "packet": {
        "target_uri": ".areaflow/status.json",
        "expected_before_exists": os.environ["PACKAGE_A_EXPECTED_BEFORE_EXISTS"] == "true",
        "expected_before_sha256": os.environ["PACKAGE_A_EXPECTED_BEFORE_SHA256"],
        "expected_before_size": int(os.environ["PACKAGE_A_EXPECTED_BEFORE_SIZE"]),
        "source_hash": source_hash,
        "schema_uri": os.environ["PACKAGE_A_SCHEMA_URI"],
        "validator_preflight": os.environ["PACKAGE_A_VALIDATOR_PREFLIGHT"],
        "protected_path_check": os.environ["PACKAGE_A_PROTECTED_PATH_CHECK"],
        "protected_path_fingerprint_sha256": os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_SHA256"],
        "rollback_action": os.environ["PACKAGE_A_ROLLBACK_ACTION"],
        "accept_preimage_schema": os.environ["PACKAGE_A_ACCEPT_PREIMAGE_SCHEMA"],
    },
    "fingerprints": {
        "status_before": os.environ["PACKAGE_A_STATUS_BEFORE"],
        "status_after": os.environ["PACKAGE_A_STATUS_AFTER"],
        "workflow_readme_before": os.environ["PACKAGE_A_WORKFLOW_README_BEFORE"],
        "workflow_readme_after": os.environ["PACKAGE_A_WORKFLOW_README_AFTER"],
        "protected_path_fingerprint_before": os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_BEFORE"],
        "protected_path_fingerprint_after": os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_AFTER"],
        "dirty_output_sha256_after": os.environ["PACKAGE_A_DIRTY_OUTPUT_SHA256_AFTER"],
    },
    "safety_facts": {
        "writes_db": True,
        "modifies_areamatrix": False,
        "project_write_attempted": False,
        "workflow_readme_changed": os.environ["PACKAGE_A_WORKFLOW_README_BEFORE"] != os.environ["PACKAGE_A_WORKFLOW_README_AFTER"],
        "status_projection_changed": os.environ["PACKAGE_A_STATUS_BEFORE"] != os.environ["PACKAGE_A_STATUS_AFTER"],
        "protected_path_fingerprint_changed": os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_BEFORE"] != os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_AFTER"],
        "execution_write_attempted": False,
        "engine_call_attempted": False,
        "uses_network": False,
        "approves_write": False,
        "applies_status_projection": False,
    },
}
print(json.dumps(payload, indent=2, ensure_ascii=False, sort_keys=True))
PY
else
  echo "package-a-source-hash: status=${status}"
  for blocker in "${blockers[@]:-}"; do
    [[ -z "${blocker}" ]] && continue
    echo "package-a-source-hash: blocker=${blocker}"
  done
  echo "package-a-source-hash: project_root=${project_root}"
  echo "package-a-source-hash: project_key=${project_key}"
  echo "package-a-source-hash: source_hash=${source_hash}"
  echo "package-a-source-hash: source_hash_env=AREAFLOW_PACKAGE_A_SOURCE_HASH=${source_hash}"
  echo "package-a-source-hash: dirty_review_status=${dirty_review_status}"
  echo "package-a-source-hash: dirty_output_sha256=${dirty_output_hash}"
  echo "package-a-source-hash: expected_before_sha256=${expected_before_sha256}"
  echo "package-a-source-hash: expected_before_size=${expected_before_size}"
  echo "package-a-source-hash: protected_path_fingerprint_sha256=${protected_path_fingerprint_sha256}"
  echo "package-a-source-hash: protected_path_fingerprint_before=${protected_path_fingerprint_before}"
  echo "package-a-source-hash: protected_path_fingerprint_after=${protected_path_fingerprint_after}"
  echo "package-a-source-hash: note=writes AreaFlow DB only; does not modify AreaMatrix or approve/apply Package A"
fi

if [[ "${status}" != "ready_for_package_a_packet_binding" ]]; then
  exit 1
fi
