#!/usr/bin/env bash
set -euo pipefail

project_root="${AREAFLOW_PACKAGE_A_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
schema_path="${AREAFLOW_PACKAGE_A_SCHEMA:-schemas/status-projection.schema.json}"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
reviewed_dirty_output_hash="${AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256:-}"
dirty_reviewer="${AREAFLOW_PACKAGE_A_DIRTY_REVIEWER:-}"
provided_source_hash="${AREAFLOW_PACKAGE_A_SOURCE_HASH:-}"
source_hash=""
output_json=0
authorization_phrase="授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"

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

forbidden_actions=(
  "write workflow/README.md"
  "write workflow/versions/**"
  "write execution/progress/logs/checkpoints"
  "write AreaMatrix scripts or source code"
  "forward ./task-loop run"
  "run promotion apply"
  "run native doctor implicitly"
  "create git checkpoint"
  "run engine, resolve secret, use network, publish, restore"
)

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
  python3 - "${project_root}" ".areaflow/status.json" "${protected_paths[@]}" <<'PY'
import hashlib
import os
import stat
import sys

root = os.path.abspath(sys.argv[1])
target = os.path.abspath(os.path.join(root, sys.argv[2]))
protected_paths = sys.argv[3:]


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
    if absolute == target:
        continue
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

json_get() {
  local expression="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); print(${expression})"
}

if [[ ! -d "${project_root}" ]]; then
  echo "package-a-authorization-packet: blocked missing project root: ${project_root}" >&2
  exit 1
fi

schema_state="missing"
target_preimage_exists="false"
target_preimage_sha256=""
target_preimage_size_bytes="0"
if [[ -f "${status_path}" ]]; then
  target_preimage_exists="true"
  target_preimage_sha256="$(shasum -a 256 "${status_path}" | awk '{print $1}')"
  target_preimage_size_bytes="$(wc -c <"${status_path}" | awk '{print $1}')"
  if python3 scripts/validate-status-projection-schema.py "${schema_path}" "${status_path}" >/dev/null 2>&1; then
    schema_state="stable_already"
  else
    schema_state="legacy_needs_package_a"
  fi
fi

accepted_preimage_schema_status="missing"
if [[ "${schema_state}" == "stable_already" ]]; then
  accepted_preimage_schema_status="stable"
elif [[ "${schema_state}" == "legacy_needs_package_a" ]]; then
  accepted_preimage_schema_status="legacy"
fi

validator_preflight="python3 scripts/validate-status-projection-schema.py ${schema_path} ${status_path}"
protected_path_check="git -C ${project_root} status --short -- ${protected_paths[*]}"
rollback_action="delete .areaflow/status.json if apply created it"
if [[ "${target_preimage_exists}" == "true" ]]; then
  rollback_action="restore the captured preimage bytes for .areaflow/status.json"
fi
durable_allowed_writes=(
  ".areaflow/status.json"
)
transient_write_paths=(
  ".areaflow/.status.json.tmp-*"
  ".areaflow/.status.json.rollback-*"
)
transient_write_scope="same-directory atomic status projection replace and rollback compensation only"

git_status_output="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
dirty_output_hash="$(printf "%s" "${git_status_output}" | shasum -a 256 | awk '{print $1}')"
protected_path_rule_count="${#protected_paths[@]}"
dirty_path_count="$(printf "%s\n" "${git_status_output}" | awk 'NF {count++} END {print count+0}')"
protected_path_fingerprint_sha256=""
protected_path_fingerprint_status="captured"
if ! protected_path_fingerprint_sha256="$(protected_path_fingerprint 2>/dev/null)"; then
  protected_path_fingerprint_sha256=""
  protected_path_fingerprint_status="unavailable"
fi
dirty_state="clean"
dirty_review_status="clean"
if [[ -n "${git_status_output}" ]]; then
  dirty_state="dirty"
  dirty_review_status="required"
  if [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" == "${dirty_output_hash}" && -n "${dirty_reviewer}" ]]; then
    dirty_review_status="accepted"
  elif [[ -n "${reviewed_dirty_output_hash}" && "${reviewed_dirty_output_hash}" != "${dirty_output_hash}" ]]; then
    dirty_review_status="mismatch"
  elif [[ -n "${reviewed_dirty_output_hash}" && -z "${dirty_reviewer}" ]]; then
    dirty_review_status="missing_reviewer"
  fi
fi

source_hash_status="missing"
source_hash_authority_status="not_attempted"
source_hash_authority_command="bash scripts/audit-package-a-source-hash.sh --json"
source_hash_authority_rc=""
source_hash_authority_output_sha256=""
source_hash_authority_writes_db="false"
source_hash_authority_modifies_areamatrix="false"
source_hash_authority_status_before=""
source_hash_authority_status_after=""
source_hash_authority_workflow_readme_before=""
source_hash_authority_workflow_readme_after=""
source_hash_authority_dirty_output_sha256_before="${dirty_output_hash}"
source_hash_authority_dirty_output_sha256_after="${dirty_output_hash}"
source_hash_authority_protected_path_fingerprint_before="${protected_path_fingerprint_sha256}"
source_hash_authority_protected_path_fingerprint_after="${protected_path_fingerprint_sha256}"

if [[ -n "${provided_source_hash}" ]]; then
  source_hash_status="provided_unbound"
fi

source_hash_preflight_allows_binding="false"
if [[ "${schema_state}" != "missing" && "${protected_path_fingerprint_status}" == "captured" && "${dirty_review_status}" != "required" && "${dirty_review_status}" != "mismatch" && "${dirty_review_status}" != "missing_reviewer" ]]; then
  source_hash_preflight_allows_binding="true"
fi

if [[ "${source_hash_preflight_allows_binding}" == "true" ]]; then
  if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
    source_hash_authority_status="attempted"
    source_hash_authority_writes_db="true"
    source_hash_authority_status_before="$(file_fingerprint "${status_path}")"
    source_hash_authority_workflow_readme_before="$(file_fingerprint "${workflow_readme}")"

    set +e
    source_hash_authority_json="$(bash scripts/audit-package-a-source-hash.sh --json 2>&1)"
    source_hash_authority_rc=$?
    set -e
    source_hash_authority_output_sha256="$(printf "%s" "${source_hash_authority_json}" | shasum -a 256 | awk '{print $1}')"
    source_hash_authority_status_after="$(file_fingerprint "${status_path}")"
    source_hash_authority_workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
    git_status_after_source_hash="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"
    source_hash_authority_dirty_output_sha256_after="$(printf "%s" "${git_status_after_source_hash}" | shasum -a 256 | awk '{print $1}')"
    source_hash_authority_protected_path_fingerprint_after="$(protected_path_fingerprint 2>/dev/null || true)"

    if [[ ${source_hash_authority_rc} -ne 0 ]]; then
      source_hash_status="authority_failed"
      source_hash_authority_status="failed"
    else
      authoritative_source_hash="$(json_get 'data.get("source_hash") or data.get("packet", {}).get("source_hash", "")' <<<"${source_hash_authority_json}")"
      if [[ -z "${authoritative_source_hash}" ]]; then
        source_hash_status="authority_failed"
        source_hash_authority_status="missing_hash"
      elif [[ "${source_hash_authority_status_before}" != "${source_hash_authority_status_after}" ]]; then
        source_hash_status="authority_changed_protected_paths"
        source_hash_authority_status="changed_status_projection"
      elif [[ "${source_hash_authority_workflow_readme_before}" != "${source_hash_authority_workflow_readme_after}" ]]; then
        source_hash_status="authority_changed_protected_paths"
        source_hash_authority_status="changed_workflow_readme"
      elif [[ -z "${source_hash_authority_protected_path_fingerprint_after}" || "${source_hash_authority_protected_path_fingerprint_before}" != "${source_hash_authority_protected_path_fingerprint_after}" ]]; then
        source_hash_status="authority_changed_protected_paths"
        source_hash_authority_status="changed_protected_path_fingerprint"
      elif [[ "${source_hash_authority_dirty_output_sha256_before}" != "${source_hash_authority_dirty_output_sha256_after}" ]]; then
        source_hash_status="authority_changed_protected_paths"
        source_hash_authority_status="changed_protected_path_state"
      elif [[ -n "${provided_source_hash}" && "${provided_source_hash}" != "${authoritative_source_hash}" ]]; then
        source_hash_status="mismatch_latest_import_snapshot"
        source_hash_authority_status="mismatch"
      else
        source_hash="${authoritative_source_hash}"
        source_hash_status="bound_to_latest_import_snapshot"
        source_hash_authority_status="bound"
      fi
    fi
  elif [[ -n "${provided_source_hash}" ]]; then
    source_hash_status="provided_unbound"
    source_hash_authority_status="not_attempted_missing_database_url"
  else
    source_hash_status="missing"
    source_hash_authority_status="not_attempted_missing_database_url"
  fi
else
  source_hash_authority_status="not_attempted_preflight_blocked"
fi

packet_status="ready_for_narrow_status_projection_authorization"
if [[ "${schema_state}" == "missing" || "${protected_path_fingerprint_status}" != "captured" || "${dirty_review_status}" == "required" || "${dirty_review_status}" == "mismatch" || "${dirty_review_status}" == "missing_reviewer" ]]; then
  packet_status="blocked_needs_preflight_review"
elif [[ "${source_hash_status}" != "bound_to_latest_import_snapshot" ]]; then
  packet_status="blocked_needs_authoritative_source_hash"
fi

required_after=(
  "${validator_preflight}"
  "${protected_path_check}"
)

apply_gate_arguments=(
  "--target .areaflow/status.json"
  "--expected-before-exists ${target_preimage_exists}"
  "--expected-before-size ${target_preimage_size_bytes}"
  "--schema-uri ${schema_path}"
  "--validator-preflight ${validator_preflight}"
  "--protected-path-check ${protected_path_check}"
  "--rollback-action ${rollback_action}"
  "--accept-preimage-schema ${accepted_preimage_schema_status}"
)
if [[ -n "${protected_path_fingerprint_sha256}" ]]; then
  apply_gate_arguments+=("--protected-path-fingerprint-sha256 ${protected_path_fingerprint_sha256}")
fi
if [[ -n "${target_preimage_sha256}" ]]; then
  apply_gate_arguments+=("--expected-before-sha256 ${target_preimage_sha256}")
fi
if [[ -n "${source_hash}" ]]; then
  apply_gate_arguments+=("--source-hash ${source_hash}")
fi

missing_apply_gate_arguments=()
if [[ "${source_hash_status}" != "bound_to_latest_import_snapshot" ]]; then
  missing_apply_gate_arguments+=("--source-hash <bound latest AreaFlow import snapshot hash>")
fi
if [[ "${protected_path_fingerprint_status}" != "captured" || -z "${protected_path_fingerprint_sha256}" ]]; then
  missing_apply_gate_arguments+=("--protected-path-fingerprint-sha256 <captured non-target protected path fingerprint>")
fi

post_authorization_required_arguments=(
  "--explicit-approval"
  "--approval-actor <authorized Package A approver>"
  "--approval-reason ${authorization_phrase}"
)

if (( output_json )); then
  export PACKAGE_A_PACKET_STATUS="${packet_status}"
  export PACKAGE_A_PROJECT_ROOT="${project_root}"
  export PACKAGE_A_TARGET=".areaflow/status.json"
  export PACKAGE_A_SCHEMA_STATE="${schema_state}"
  export PACKAGE_A_ACCEPTED_PREIMAGE_SCHEMA_STATUS="${accepted_preimage_schema_status}"
  export PACKAGE_A_TARGET_PREIMAGE_EXISTS="${target_preimage_exists}"
  export PACKAGE_A_TARGET_PREIMAGE_SHA256="${target_preimage_sha256}"
  export PACKAGE_A_TARGET_PREIMAGE_SIZE_BYTES="${target_preimage_size_bytes}"
  export PACKAGE_A_SOURCE_HASH="${source_hash}"
  export PACKAGE_A_PROVIDED_SOURCE_HASH="${provided_source_hash}"
  export PACKAGE_A_SOURCE_HASH_STATUS="${source_hash_status}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_COMMAND="${source_hash_authority_command}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_STATUS="${source_hash_authority_status}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_RC="${source_hash_authority_rc}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_OUTPUT_SHA256="${source_hash_authority_output_sha256}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_WRITES_DB="${source_hash_authority_writes_db}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_MODIFIES_AREAMATRIX="${source_hash_authority_modifies_areamatrix}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_STATUS_BEFORE="${source_hash_authority_status_before}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_STATUS_AFTER="${source_hash_authority_status_after}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_WORKFLOW_README_BEFORE="${source_hash_authority_workflow_readme_before}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_WORKFLOW_README_AFTER="${source_hash_authority_workflow_readme_after}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_DIRTY_OUTPUT_SHA256_BEFORE="${source_hash_authority_dirty_output_sha256_before}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_DIRTY_OUTPUT_SHA256_AFTER="${source_hash_authority_dirty_output_sha256_after}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_PROTECTED_PATH_FINGERPRINT_BEFORE="${source_hash_authority_protected_path_fingerprint_before}"
  export PACKAGE_A_SOURCE_HASH_AUTHORITY_PROTECTED_PATH_FINGERPRINT_AFTER="${source_hash_authority_protected_path_fingerprint_after}"
  export PACKAGE_A_VALIDATOR_PREFLIGHT="${validator_preflight}"
  export PACKAGE_A_PROTECTED_PATH_CHECK="${protected_path_check}"
  export PACKAGE_A_PROTECTED_PATH_FINGERPRINT_SHA256="${protected_path_fingerprint_sha256}"
  export PACKAGE_A_PROTECTED_PATH_FINGERPRINT_STATUS="${protected_path_fingerprint_status}"
  export PACKAGE_A_ROLLBACK_ACTION="${rollback_action}"
  export PACKAGE_A_PROTECTED_PATH_STATE="${dirty_state}"
  export PACKAGE_A_PROTECTED_PATH_RULE_COUNT="${protected_path_rule_count}"
  export PACKAGE_A_DIRTY_PATH_COUNT="${dirty_path_count}"
  export PACKAGE_A_DIRTY_REVIEW_STATUS="${dirty_review_status}"
  export PACKAGE_A_DIRTY_OUTPUT_SHA256="${dirty_output_hash}"
  export PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${reviewed_dirty_output_hash}"
  export PACKAGE_A_DIRTY_REVIEWER="${dirty_reviewer}"
  export PACKAGE_A_SCOPE="package_a_status_projection_only"
  export PACKAGE_A_ALLOWED_WRITE=".areaflow/status.json"
  export PACKAGE_A_DURABLE_ALLOWED_WRITES="$(printf "%s\n" "${durable_allowed_writes[@]}")"
  export PACKAGE_A_TRANSIENT_WRITE_PATHS="$(printf "%s\n" "${transient_write_paths[@]}")"
  export PACKAGE_A_TRANSIENT_WRITE_SCOPE="${transient_write_scope}"
  export PACKAGE_A_REQUIRED_COMMAND="areaflow project status-projection-apply areamatrix <packet-derived args>"
  export PACKAGE_A_REQUIRED_AFTER="$(printf "%s\n" "${required_after[@]}")"
  export PACKAGE_A_APPLY_GATE_ARGUMENTS="$(printf "%s\n" "${apply_gate_arguments[@]}")"
  if ((${#missing_apply_gate_arguments[@]})); then
    export PACKAGE_A_MISSING_APPLY_GATE_ARGUMENTS="$(printf "%s\n" "${missing_apply_gate_arguments[@]}")"
  else
    export PACKAGE_A_MISSING_APPLY_GATE_ARGUMENTS=""
  fi
  export PACKAGE_A_POST_AUTHORIZATION_REQUIRED_ARGUMENTS="$(printf "%s\n" "${post_authorization_required_arguments[@]}")"
  export PACKAGE_A_FORBIDDEN_ACTIONS="$(printf "%s\n" "${forbidden_actions[@]}")"
  export PACKAGE_A_PROTECTED_PATHS="$(printf "%s\n" "${protected_paths[@]}")"
  export PACKAGE_A_GIT_STATUS_OUTPUT="${git_status_output}"
  export PACKAGE_A_AUTHORIZATION_PHRASE="${authorization_phrase}"
  export PACKAGE_A_DIRTY_REVIEW_REQUIRED="false"
  if [[ "${dirty_review_status}" == "required" || "${dirty_review_status}" == "mismatch" || "${dirty_review_status}" == "missing_reviewer" ]]; then
    export PACKAGE_A_DIRTY_REVIEW_REQUIRED="true"
  fi
  python3 - <<'PY'
import json
import os

git_status = os.environ.get("PACKAGE_A_GIT_STATUS_OUTPUT", "")
dirty_lines = [line for line in git_status.splitlines() if line.strip()]
touched_paths = [line[3:] if len(line) > 3 else line for line in dirty_lines]

payload = {
    "status": os.environ["PACKAGE_A_PACKET_STATUS"],
    "project_root": os.environ["PACKAGE_A_PROJECT_ROOT"],
    "target": os.environ["PACKAGE_A_TARGET"],
    "schema_state": os.environ["PACKAGE_A_SCHEMA_STATE"],
    "accepted_preimage_schema_status": os.environ["PACKAGE_A_ACCEPTED_PREIMAGE_SCHEMA_STATUS"],
    "target_preimage": {
        "exists": os.environ["PACKAGE_A_TARGET_PREIMAGE_EXISTS"] == "true",
        "sha256": os.environ["PACKAGE_A_TARGET_PREIMAGE_SHA256"],
        "size_bytes": int(os.environ["PACKAGE_A_TARGET_PREIMAGE_SIZE_BYTES"]),
    },
    "source_hash": os.environ["PACKAGE_A_SOURCE_HASH"],
    "provided_source_hash": os.environ["PACKAGE_A_PROVIDED_SOURCE_HASH"],
    "source_hash_status": os.environ["PACKAGE_A_SOURCE_HASH_STATUS"],
    "source_hash_authority": {
        "command": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_COMMAND"],
        "status": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_STATUS"],
        "rc": int(os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_RC"]) if os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_RC"] else None,
        "output_sha256": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_OUTPUT_SHA256"],
        "writes_db": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_WRITES_DB"] == "true",
        "modifies_areamatrix": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_MODIFIES_AREAMATRIX"] == "true",
        "fingerprints": {
            "status_before": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_STATUS_BEFORE"],
            "status_after": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_STATUS_AFTER"],
            "workflow_readme_before": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_WORKFLOW_README_BEFORE"],
            "workflow_readme_after": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_WORKFLOW_README_AFTER"],
            "dirty_output_sha256_before": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_DIRTY_OUTPUT_SHA256_BEFORE"],
            "dirty_output_sha256_after": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_DIRTY_OUTPUT_SHA256_AFTER"],
            "protected_path_fingerprint_sha256_before": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_PROTECTED_PATH_FINGERPRINT_BEFORE"],
            "protected_path_fingerprint_sha256_after": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_PROTECTED_PATH_FINGERPRINT_AFTER"],
        },
    },
    "validator_preflight": os.environ["PACKAGE_A_VALIDATOR_PREFLIGHT"],
    "protected_path_check": os.environ["PACKAGE_A_PROTECTED_PATH_CHECK"],
    "protected_path_fingerprint_sha256": os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_SHA256"],
    "protected_path_fingerprint_status": os.environ["PACKAGE_A_PROTECTED_PATH_FINGERPRINT_STATUS"],
    "rollback_action": os.environ["PACKAGE_A_ROLLBACK_ACTION"],
    "protected_path_state": os.environ["PACKAGE_A_PROTECTED_PATH_STATE"],
    "protected_path_rule_count": int(os.environ["PACKAGE_A_PROTECTED_PATH_RULE_COUNT"]),
    "dirty_path_count": int(os.environ["PACKAGE_A_DIRTY_PATH_COUNT"]),
    "dirty_review_status": os.environ["PACKAGE_A_DIRTY_REVIEW_STATUS"],
    "dirty_output_sha256": os.environ["PACKAGE_A_DIRTY_OUTPUT_SHA256"],
    "reviewed_dirty_output_sha256": os.environ["PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256"],
    "dirty_reviewer": os.environ["PACKAGE_A_DIRTY_REVIEWER"],
    "scope": os.environ["PACKAGE_A_SCOPE"],
    "allowed_writes": [os.environ["PACKAGE_A_ALLOWED_WRITE"]],
    "durable_allowed_writes": [line for line in os.environ["PACKAGE_A_DURABLE_ALLOWED_WRITES"].splitlines() if line],
    "transient_write_paths": [line for line in os.environ["PACKAGE_A_TRANSIENT_WRITE_PATHS"].splitlines() if line],
    "transient_write_scope": os.environ["PACKAGE_A_TRANSIENT_WRITE_SCOPE"],
    "transient_writes_cleanup_required": True,
    "transient_writes_durable_authorization": False,
    "required_command": os.environ["PACKAGE_A_REQUIRED_COMMAND"],
    "apply_gate_arguments": [line for line in os.environ["PACKAGE_A_APPLY_GATE_ARGUMENTS"].splitlines() if line],
    "missing_apply_gate_arguments": [line for line in os.environ["PACKAGE_A_MISSING_APPLY_GATE_ARGUMENTS"].splitlines() if line],
    "post_authorization_required_arguments": [line for line in os.environ["PACKAGE_A_POST_AUTHORIZATION_REQUIRED_ARGUMENTS"].splitlines() if line],
    "required_after": [line for line in os.environ["PACKAGE_A_REQUIRED_AFTER"].splitlines() if line],
    "forbidden_actions": [line for line in os.environ["PACKAGE_A_FORBIDDEN_ACTIONS"].splitlines() if line],
    "protected_paths": [line for line in os.environ["PACKAGE_A_PROTECTED_PATHS"].splitlines() if line],
    "dirty_review_required": os.environ["PACKAGE_A_DIRTY_REVIEW_REQUIRED"] == "true",
    "dirty_review_command": "bash scripts/audit-package-a-dirty-review.sh",
    "protected_path_lines": dirty_lines,
    "touched_paths": touched_paths,
    "required_authorization_phrase": os.environ["PACKAGE_A_AUTHORIZATION_PHRASE"],
    "user_authorization_phrase": os.environ["PACKAGE_A_AUTHORIZATION_PHRASE"],
    "safety_facts": {
        "approves_write": False,
        "applies_status_projection": False,
        "writes_db": os.environ["PACKAGE_A_SOURCE_HASH_AUTHORITY_WRITES_DB"] == "true",
        "allows_transient_status_projection_temp_files": True,
        "allows_extra_durable_files": False,
        "modifies_areamatrix": False,
        "allows_shim_files": False,
        "allows_workflow_readme": False,
        "allows_workflow_versions": False,
        "allows_task_loop_forwarding": False,
        "allows_execution_write": False,
        "allows_source_write": False,
        "runs_engine": False,
        "resolves_secret": False,
        "uses_network": False,
        "publishes_release": False,
        "applies_restore": False,
    },
}

print(json.dumps(payload, indent=2, ensure_ascii=False, sort_keys=True))
PY
  if [[ "${packet_status}" != "ready_for_narrow_status_projection_authorization" ]]; then
    exit 1
  fi
  exit 0
fi

echo "package-a-authorization-packet: status=${packet_status}"
echo "package-a-authorization-packet: project_root=${project_root}"
echo "package-a-authorization-packet: target=.areaflow/status.json"
echo "package-a-authorization-packet: schema_state=${schema_state}"
echo "package-a-authorization-packet: accepted_preimage_schema_status=${accepted_preimage_schema_status}"
echo "package-a-authorization-packet: target_preimage_exists=${target_preimage_exists}"
if [[ -n "${target_preimage_sha256}" ]]; then
  echo "package-a-authorization-packet: target_preimage_sha256=${target_preimage_sha256}"
fi
echo "package-a-authorization-packet: target_preimage_size_bytes=${target_preimage_size_bytes}"
echo "package-a-authorization-packet: source_hash_status=${source_hash_status}"
if [[ -n "${provided_source_hash}" ]]; then
  echo "package-a-authorization-packet: provided_source_hash=${provided_source_hash}"
fi
echo "package-a-authorization-packet: source_hash_authority_status=${source_hash_authority_status}"
echo "package-a-authorization-packet: source_hash_authority_command=${source_hash_authority_command}"
if [[ -n "${source_hash_authority_rc}" ]]; then
  echo "package-a-authorization-packet: source_hash_authority_rc=${source_hash_authority_rc}"
fi
if [[ -n "${source_hash_authority_output_sha256}" ]]; then
  echo "package-a-authorization-packet: source_hash_authority_output_sha256=${source_hash_authority_output_sha256}"
fi
if [[ -n "${source_hash}" ]]; then
  echo "package-a-authorization-packet: source_hash=${source_hash}"
else
  echo "package-a-authorization-packet: source_hash_input=AREAFLOW_DATABASE_URL=<db> [AREAFLOW_PACKAGE_A_SOURCE_HASH=<expected latest import snapshot hash>]"
fi
echo "package-a-authorization-packet: validator_preflight=${validator_preflight}"
echo "package-a-authorization-packet: protected_path_check=${protected_path_check}"
echo "package-a-authorization-packet: protected_path_fingerprint_status=${protected_path_fingerprint_status}"
if [[ -n "${protected_path_fingerprint_sha256}" ]]; then
  echo "package-a-authorization-packet: protected_path_fingerprint_sha256=${protected_path_fingerprint_sha256}"
fi
echo "package-a-authorization-packet: rollback_action=${rollback_action}"
echo "package-a-authorization-packet: protected_path_state=${dirty_state}"
echo "package-a-authorization-packet: protected_path_rule_count=${protected_path_rule_count}"
echo "package-a-authorization-packet: dirty_path_count=${dirty_path_count}"
echo "package-a-authorization-packet: dirty_review_status=${dirty_review_status}"
echo "package-a-authorization-packet: dirty_output_sha256=${dirty_output_hash}"
if [[ -n "${reviewed_dirty_output_hash}" ]]; then
  echo "package-a-authorization-packet: reviewed_dirty_output_sha256=${reviewed_dirty_output_hash}"
fi
if [[ -n "${dirty_reviewer}" ]]; then
  echo "package-a-authorization-packet: dirty_reviewer=${dirty_reviewer}"
fi
echo "package-a-authorization-packet: scope=package_a_status_projection_only"
echo "package-a-authorization-packet: allowed_write=.areaflow/status.json"
for path in "${durable_allowed_writes[@]}"; do
  echo "package-a-authorization-packet: durable_allowed_write=${path}"
done
echo "package-a-authorization-packet: transient_write_scope=${transient_write_scope}"
for path in "${transient_write_paths[@]}"; do
  echo "package-a-authorization-packet: transient_write_path=${path}"
done
echo "package-a-authorization-packet: transient_writes_cleanup_required=true"
echo "package-a-authorization-packet: transient_writes_durable_authorization=false"
echo "package-a-authorization-packet: required_command=areaflow project status-projection-apply areamatrix <packet-derived args>"
for argument in "${apply_gate_arguments[@]}"; do
  echo "package-a-authorization-packet: apply_gate_argument=${argument}"
done
if ((${#missing_apply_gate_arguments[@]})); then
  for argument in "${missing_apply_gate_arguments[@]}"; do
    echo "package-a-authorization-packet: missing_apply_gate_argument=${argument}"
  done
fi
for argument in "${post_authorization_required_arguments[@]}"; do
  echo "package-a-authorization-packet: post_authorization_required_argument=${argument}"
done
for command in "${required_after[@]}"; do
  echo "package-a-authorization-packet: required_after=${command}"
done

for action in "${forbidden_actions[@]}"; do
  echo "package-a-authorization-packet: forbidden=${action}"
done

if [[ "${dirty_review_status}" == "required" || "${dirty_review_status}" == "mismatch" || "${dirty_review_status}" == "missing_reviewer" ]]; then
  echo "package-a-authorization-packet: dirty_review_required=true"
  echo "package-a-authorization-packet: dirty_review_command=bash scripts/audit-package-a-dirty-review.sh"
  echo "package-a-authorization-packet: dirty_review_input=AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<dirty_output_sha256> AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>"
  while IFS= read -r line; do
    [[ -z "${line}" ]] && continue
    echo "package-a-authorization-packet: protected-path: ${line}"
  done <<<"${git_status_output}"
elif [[ "${dirty_review_status}" == "accepted" ]]; then
  echo "package-a-authorization-packet: dirty_review_required=false"
  echo "package-a-authorization-packet: dirty_review_accepted=true"
else
  echo "package-a-authorization-packet: dirty_review_required=false"
fi

cat <<MSG
package-a-authorization-packet: user_authorization_phrase=${authorization_phrase}
package-a-authorization-packet: required_authorization_phrase=${authorization_phrase}
package-a-authorization-packet: note=authorization must not include shim files, workflow README, workflow versions, task-loop forwarding, execution, source writes, engine, secret, network, publish or restore
package-a-authorization-packet: note=this script does not approve, apply, record evidence, or modify AreaMatrix; with AREAFLOW_DATABASE_URL it may import into the AreaFlow DB to bind source_hash
MSG

if [[ "${packet_status}" != "ready_for_narrow_status_projection_authorization" ]]; then
  exit 1
fi
