#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-execution-forwarding-v1-readiness: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_EXECUTION_FORWARDING_V1_PROJECT_KEY:-areamatrix-forwarding-fixture}"
workflow_label="${AREAFLOW_EXECUTION_FORWARDING_V1_WORKFLOW_VERSION:-forwarding-v1-smoke-$(date +%Y%m%d%H%M%S)}"
worker_key="${workflow_label}-worker"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-forwarding-v1.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
evidence_root="docs/development/real-release-candidate-evidence.md"
execution_cutover_evidence_uri="${evidence_root}#e4-execution-cutover-rollback"
review_flags=(--review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z)
real_areamatrix_root="/Users/as/Ai-Project/project/AreaMatrix"
real_areamatrix_status="${real_areamatrix_root}/.areaflow/status.json"
real_areamatrix_readme="${real_areamatrix_root}/workflow/README.md"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

real_areamatrix_protected_paths=(
  "workflow/README.md"
  ".areaflow/status.json"
  "scripts/task_loop/console.py"
  "scripts/dev_tools/cli.py"
  "scripts/task_loop/runner.py"
  "scripts/areaflow_shim.py"
  "workflow/versions"
  "workflow/versions/v1-mvp/execution/_shared/progress.json"
)

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-execution-forwarding-v1-readiness: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-execution-forwarding-v1-readiness: expected output to contain pattern: ${pattern}" >&2
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

real_areamatrix_protected_path_git_status() {
  git -C "${real_areamatrix_root}" status --short -- "${real_areamatrix_protected_paths[@]}"
}

real_areamatrix_protected_path_fingerprint() {
  python3 - "${real_areamatrix_root}" ".areaflow/status.json" "${real_areamatrix_protected_paths[@]}" <<'PY'
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

json_run_id() {
  JSON_INPUT="$1" python3 - <<'PY'
import json
import os
print(json.loads(os.environ["JSON_INPUT"])["run"]["id"])
PY
}

real_status_before="__skipped__"
real_readme_before="__skipped__"
real_protected_path_fingerprint_before="__skipped__"
real_protected_path_status_before="__skipped__"
if [[ "${check_real_areamatrix}" == "1" ]]; then
  if [[ ! -d "${real_areamatrix_root}" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: missing real AreaMatrix root: ${real_areamatrix_root}" >&2
    exit 1
  fi
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_before="$(file_fingerprint "${real_areamatrix_readme}")"
  real_protected_path_fingerprint_before="$(real_areamatrix_protected_path_fingerprint)"
  real_protected_path_status_before="$(real_areamatrix_protected_path_git_status)"
  if [[ -n "${real_protected_path_status_before}" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: real AreaMatrix protected paths are dirty before smoke:" >&2
    echo "${real_protected_path_status_before}" >&2
    exit 1
  fi
else
  echo "smoke-execution-forwarding-v1-readiness: skipping real AreaMatrix protected path fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p \
  "${project_root}/docs" \
  "${project_root}/workflow/residuals" \
  "${project_root}/workflow/templates" \
  "${project_root}/workflow/versions/v1-mvp/execution/_shared" \
  "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Execution Forwarding v1 Fixture Docs

This fixture exists only for AreaFlow smoke tests.
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Execution Forwarding v1 Fixture Workflow
EOF

cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items: []
version_residuals: []
EOF

cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Execution Forwarding v1 Fixture Templates
EOF

cat >"${project_root}/workflow/versions/v1-mvp/execution/_shared/progress.json" <<'EOF'
{
  "tasks": {}
}
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Execution Forwarding v1 Fixture
  root: ${project_root}
  kind: fixture
  adapter: areamatrix
  workflow_profile: areamatrix
  default_branch: main

ownership:
  mode: import
  source_of_truth:
    product_docs: project
    source_code: project
    workflow: project
    execution: project
    status_summary: areaflow
  cutover:
    enabled: false
    new_versions_owned_by: project
    execution_owned_by: project

artifact_store:
  backend: local
  root: ${artifact_root}

permissions:
  capabilities:
    read_project: true
    write_status: false
    write_artifacts: true
    write_workflow: false
    write_generated: false
    write_code: false
    run_commands: false
    manage_workers: true
    manage_git: false
    network: false
    use_secrets: false
    execute_agents: false

  read_paths:
    - docs/**
    - workflow/**

  write_paths: []

  forbidden_paths:
    - workflow/versions/*/execution/**
    - workflow/versions/*/execution/_shared/progress.json
    - .areamatrix/**
    - "**/*.sqlite"
    - "**/*.db"

commands:
  allowed: []
  forbidden:
    - ./task-loop run
    - git reset --hard
    - git checkout --
    - rm -rf

scheduling:
  priority: 100
  max_parallel_tasks: 1
  agent_role: local_worker
  required_capabilities:
    - read_project
    - write_artifacts
  engine_profile: codex-cli

engines:
  default: codex-cli
  profiles:
    - id: codex-cli
      provider: codex-cli
      secret_ref: none
      enabled: false

status_export:
  enabled: false
  path: .areaflow/status.json
  human_summary:
    enabled: false
    path: workflow/README.md
    block_marker: AREAFLOW_STATUS

migration:
  strategy: import_mirror_shadow_cutover_archive
  phase: import
  imported_versions:
    - v1-mvp
  immutable_imports:
    - v1-mvp
EOF

echo "smoke-execution-forwarding-v1-readiness: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-execution-forwarding-v1-readiness: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-execution-forwarding-v1-readiness: project import ${project_key}"
import_output="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output}" "imported ${project_key}"

echo "smoke-execution-forwarding-v1-readiness: workflow version create ${workflow_label}"
version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${workflow_label}" --json)"
assert_contains "${version_json}" '"import_mode": "authored"'

echo "smoke-execution-forwarding-v1-readiness: workflow mark queue ready"
queue_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${workflow_label}" --stage queue --item-type queue_candidate --reason "forwarding v1 smoke queue" --json)"
assert_contains "${queue_ready_json}" '"status": "ready"'

echo "smoke-execution-forwarding-v1-readiness: workflow mark promotion_preview ready"
promotion_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${workflow_label}" --stage promotion_preview --item-type promotion_preview --reason "forwarding v1 smoke promotion" --json)"
assert_contains "${promotion_ready_json}" '"status": "ready"'

echo "smoke-execution-forwarding-v1-readiness: workflow gate promotion_preview"
promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" promotion_preview --json)"
assert_contains "${promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${promotion_gate_json}" '"status": "pass"'

echo "smoke-execution-forwarding-v1-readiness: workflow transition preview"
transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${workflow_label}" --json)"
assert_contains "${transition_json}" '"status": "ready"'

echo "smoke-execution-forwarding-v1-readiness: workflow approval approved"
approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${workflow_label}" --decision approved --reason "forwarding v1 smoke approval" --json)"
assert_contains "${approval_json}" '"decision": "approved"'
assert_contains "${approval_json}" '"transition_status": "ready"'

echo "smoke-execution-forwarding-v1-readiness: workflow gate approval_gate"
approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" approval_gate --json)"
assert_contains "${approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${approval_gate_json}" '"status": "pass"'

echo "smoke-execution-forwarding-v1-readiness: workflow gate live_mapping_gate"
live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" live_mapping_gate --json)"
assert_contains "${live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${live_mapping_gate_json}" '"status": "pass"'
assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-execution-forwarding-v1-readiness: worker register"
worker_json="$(go run ./cmd/areaflow worker register "${project_key}" --worker-key "${worker_key}" --capability read_project --capability write_artifacts --json)"
assert_contains "${worker_json}" '"worker_key": "'"${worker_key}"'"'
assert_contains "${worker_json}" '"read_project"'
assert_contains "${worker_json}" '"write_artifacts"'

echo "smoke-execution-forwarding-v1-readiness: run read-only-verify-queue"
verify_queue_json="$(go run ./cmd/areaflow run read-only-verify-queue "${project_key}" "${workflow_label}" --target-path docs/README.md --idempotency-key "forwarding-v1-readonly-queue-${workflow_label}" --reason "forwarding v1 smoke read-only evidence" --json)"
assert_contains "${verify_queue_json}" '"run_type": "read_only_verify"'
assert_contains "${verify_queue_json}" '"project_write_attempted": false'
verify_run_id="$(json_run_id "${verify_queue_json}")"

echo "smoke-execution-forwarding-v1-readiness: worker read-only-verify"
verify_apply_json="$(go run ./cmd/areaflow worker read-only-verify "${project_key}" "${worker_key}" --run-id "${verify_run_id}" --capability read_project --idempotency-key "forwarding-v1-readonly-apply-${workflow_label}" --reason "forwarding v1 smoke read-only apply" --json)"
assert_contains "${verify_apply_json}" '"status": "verified"'
assert_contains "${verify_apply_json}" '"project_write_attempted": false'
assert_contains "${verify_apply_json}" '"execution_write_attempted": false'

echo "smoke-execution-forwarding-v1-readiness: run approved-artifact-write-queue"
artifact_queue_json="$(go run ./cmd/areaflow run approved-artifact-write-queue "${project_key}" "${workflow_label}" --artifact-label forwarding-v1-evidence --idempotency-key "forwarding-v1-artifact-queue-${workflow_label}" --reason "forwarding v1 smoke artifact evidence" --json)"
assert_contains "${artifact_queue_json}" '"run_type": "approved_artifact_write"'
assert_contains "${artifact_queue_json}" '"project_write_attempted": false'
artifact_run_id="$(json_run_id "${artifact_queue_json}")"

echo "smoke-execution-forwarding-v1-readiness: worker approved-artifact-write"
artifact_apply_json="$(go run ./cmd/areaflow worker approved-artifact-write "${project_key}" "${worker_key}" --run-id "${artifact_run_id}" --capability write_artifacts --idempotency-key "forwarding-v1-artifact-apply-${workflow_label}" --reason "forwarding v1 smoke artifact apply" --json)"
assert_contains "${artifact_apply_json}" '"status": "artifact_written"'
assert_contains "${artifact_apply_json}" '"area_flow_artifact_written": true'
assert_contains "${artifact_apply_json}" '"project_write_attempted": false'
assert_contains "${artifact_apply_json}" '"execution_write_attempted": false'

echo "smoke-execution-forwarding-v1-readiness: completion protected-path-proof record"
protected_path_json="$(go run ./cmd/areaflow completion protected-path-proof record "${project_key}" --status clean --summary "forwarding v1 smoke protected paths clean" --evidence-uri "local:forwarding-v1-protected-path-${workflow_label}" --idempotency-key "forwarding-v1-protected-path-${workflow_label}" --json)"
assert_contains "${protected_path_json}" '"proof_status": "clean"'
assert_contains "${protected_path_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${protected_path_json}" '"commands_run": false'
assert_contains "${protected_path_json}" '"git_status_output_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"'
assert_contains "${protected_path_json}" '"git_status_output_empty": true'
assert_contains "${protected_path_json}" '"protected_path_set_hash": "'
assert_contains "${protected_path_json}" '"protected_path_set_count": 7'
assert_contains "${protected_path_json}" '"protected_path_proof_binding_status": "pass"'
assert_contains "${protected_path_json}" '"protected_path_proof_binding_blockers": []'

echo "smoke-execution-forwarding-v1-readiness: completion execution-cutover-proof record"
execution_cutover_binding_args=(
  --execution-cutover-scope execution_forwarding_v1_read_only_evidence_only
  --allowed-task-type read_only_verify
  --allowed-task-type doctor_readiness
  --allowed-task-type artifact_evidence
  --allowed-task-type status_projection_validation
  --allowed-task-type release_readiness_check
  --forbidden-action start_legacy_task_loop_runner
  --forbidden-action write_legacy_progress_json
  --forbidden-action write_legacy_logs
  --forbidden-action write_legacy_checkpoint
  --forbidden-action write_areamatrix_source
  --forbidden-action write_areamatrix_execution_directory
  --forbidden-action generated_retained_write
  --forbidden-action repair_apply
  --forbidden-action checkpoint_apply
  --forbidden-action engine_execution
  --forbidden-action secret_resolve
  --forbidden-action network_api_integration
  --forbidden-action publish_apply
  --forbidden-action restore_apply
  --rollback-target read_only_shim
  --rollback-mode fail_closed_to_read_only_shim
  --fail-closed
  --reopen-requires-approval
)
execution_cutover_json="$(go run ./cmd/areaflow completion execution-cutover-proof record "${project_key}" \
  --status complete \
  --fact explicit_execution_cutover_approval_recorded \
  --fact execution_cutover_command_response_recorded \
  --fact execution_cutover_event_and_audit_recorded \
  --fact task_loop_run_forwarding_window_proven \
  --fact rollback_plan_and_compatibility_window_proven \
  --fact no_unapproved_project_or_execution_write_attempted \
  --fact rollback_target_read_only_shim_confirmed \
  --fact forwarding_v1_command_disabled_or_absent \
  --fact task_loop_run_forwarding_disabled \
  --fact legacy_task_loop_runner_not_started_after_rollback \
  --fact legacy_progress_json_not_written_after_rollback \
  --fact legacy_logs_not_written_after_rollback \
  --fact legacy_checkpoint_not_written_after_rollback \
  --fact areaflow_forwarded_state_preserved_as_audit_history \
  --fact protected_path_proof_clean_after_rollback_recorded \
  "${execution_cutover_binding_args[@]}" \
  --summary "forwarding v1 smoke rollback proof facts complete" \
  --evidence-uri "${execution_cutover_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "forwarding-v1-rollback-proof-${workflow_label}" \
  --json)"
assert_contains "${execution_cutover_json}" '"proof_status": "complete"'
assert_contains "${execution_cutover_json}" '"project_write_attempted": false'
assert_contains "${execution_cutover_json}" '"execution_write_attempted": false'
assert_contains "${execution_cutover_json}" '"task_loop_run_forwarded_by_command": false'
assert_contains "${execution_cutover_json}" '"commands_run": false'
assert_contains "${execution_cutover_json}" '"legacy_progress_written": false'
assert_contains "${execution_cutover_json}" '"legacy_logs_written": false'
assert_contains "${execution_cutover_json}" '"legacy_checkpoint_written": false'
assert_contains "${execution_cutover_json}" '"area_matrix_protected_paths_touched": false'

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-readiness --json"
readiness_json="$(go run ./cmd/areaflow project execution-forwarding-v1-readiness "${project_key}" --json)"
READINESS_JSON="${readiness_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["READINESS_JSON"])
items = {item["key"]: item for item in data["items"]}
safety = data["safety_facts"]

expected_item_status = {
    "allowed_task_scope": "pass",
    "forbidden_high_risk_targets": "pass",
    "read_only_verify_evidence": "pass",
    "artifact_evidence": "pass",
    "read_only_shim": "blocked",
    "forwarding_command_api": "pass",
    "legacy_non_write_proof": "pass",
    "rollback_to_read_only_shim": "pass",
}

if data["status"] != "blocked":
    sys.exit(f"status={data['status']!r}, want blocked")
if data["mode"] != "read_only_execution_forwarding_v1_readiness":
    sys.exit(f"mode={data['mode']!r}, want read_only_execution_forwarding_v1_readiness")
for key, want in expected_item_status.items():
    got = items.get(key, {}).get("status")
    if got != want:
        sys.exit(f"item {key} status={got!r}, want {want!r}")

for task_type in [
    "read_only_verify",
    "doctor_readiness",
    "artifact_evidence",
    "status_projection_validation",
    "release_readiness_check",
]:
    if task_type not in data["allowed_task_types"]:
        sys.exit(f"missing allowed task type {task_type}")
for forbidden in ["source_write", "repair", "checkpoint", "engine_execution", "secret_resolve", "restore_apply"]:
    if forbidden in data["allowed_task_types"]:
        sys.exit(f"forbidden task type leaked into allowed list: {forbidden}")

for key in [
    "forwarding_v1_apply_open",
    "task_loop_run_forwarded",
    "legacy_task_loop_started",
    "legacy_progress_written",
    "legacy_logs_written",
    "legacy_checkpoint_written",
    "project_write_attempted",
    "execution_write_attempted",
    "area_flow_command_created",
    "area_flow_run_created",
    "worker_scheduled",
    "engine_call_attempted",
    "commands_run",
    "secrets_resolved",
    "network_used",
    "source_write_open",
    "generated_retained_write_open",
    "repair_apply_open",
    "checkpoint_apply_open",
    "publish_apply_open",
    "restore_apply_open",
    "areamatrix_protected_paths_touched",
]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
if safety.get("read_only") is not True:
    sys.exit("safety fact read_only should be true")

legacy = items["legacy_non_write_proof"]["metadata"]
if legacy.get("proof_status") != "clean":
    sys.exit(f"legacy proof status={legacy.get('proof_status')!r}, want clean")
if legacy.get("areamatrix_protected_paths_touched") is not False:
    sys.exit(f"legacy protected paths touched={legacy.get('areamatrix_protected_paths_touched')!r}, want false")
if legacy.get("protected_path_proof_binding_status") != "pass":
    sys.exit(f"legacy protected proof binding={legacy.get('protected_path_proof_binding_status')!r}, want pass")
if legacy.get("protected_path_proof_binding_blockers") != []:
    sys.exit(f"legacy protected proof blockers={legacy.get('protected_path_proof_binding_blockers')!r}, want []")
if legacy.get("git_status_output_hash") != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855":
    sys.exit(f"legacy git_status_output_hash={legacy.get('git_status_output_hash')!r}, want empty sha256")
if legacy.get("git_status_output_lines") != 0:
    sys.exit(f"legacy git_status_output_lines={legacy.get('git_status_output_lines')!r}, want 0")
if legacy.get("git_status_output_empty") is not True:
    sys.exit(f"legacy git_status_output_empty={legacy.get('git_status_output_empty')!r}, want true")
if not legacy.get("protected_path_set_hash"):
    sys.exit("legacy protected_path_set_hash missing")
if legacy.get("protected_path_set_count") != 7:
    sys.exit(f"legacy protected_path_set_count={legacy.get('protected_path_set_count')!r}, want 7")
rollback = items["rollback_to_read_only_shim"]["metadata"]
if rollback.get("proof_status") != "complete":
    sys.exit(f"rollback proof status={rollback.get('proof_status')!r}, want complete")
if rollback.get("missing_proof_facts") != []:
    sys.exit(f"rollback missing facts={rollback.get('missing_proof_facts')!r}, want []")
if rollback.get("execution_cutover_scope_binding_status") != "pass":
    sys.exit(f"rollback binding status={rollback.get('execution_cutover_scope_binding_status')!r}, want pass")
for key in [
    "project_write_attempted",
    "execution_write_attempted",
    "task_loop_run_forwarded_by_command",
    "commands_run",
    "legacy_progress_written",
    "legacy_logs_written",
    "legacy_checkpoint_written",
    "areamatrix_protected_paths_touched",
]:
    if rollback.get(key) is not False:
        sys.exit(f"rollback metadata {key}={rollback.get(key)!r}, want false")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply-preview --json"
apply_preview_json="$(go run ./cmd/areaflow project execution-forwarding-v1-apply-preview "${project_key}" --json)"
APPLY_PREVIEW_JSON="${apply_preview_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["APPLY_PREVIEW_JSON"])
items = {item["key"]: item for item in data["items"]}
safety = data["safety_facts"]

if data["status"] != "blocked":
    sys.exit(f"status={data['status']!r}, want blocked")
if data["mode"] != "read_only_execution_forwarding_v1_apply_preview":
    sys.exit(f"mode={data['mode']!r}, want read_only_execution_forwarding_v1_apply_preview")
if data["readiness"]["status"] != "blocked":
    sys.exit(f"nested readiness status={data['readiness']['status']!r}, want blocked")
if data["approval_status"] != "needs_approval" or not data["approval_required"]:
    sys.exit(f"unexpected approval fields: {data['approval_required']=} {data['approval_status']=}")
if data["apply_open"] is not False:
    sys.exit("apply_open should stay false")
if data["rollback_target"] != "read_only_shim":
    sys.exit(f"rollback_target={data['rollback_target']!r}, want read_only_shim")

expected_item_status = {
    "forwarding_v1:readiness": "blocked",
    "forwarding_v1:explicit_approval": "blocked",
    "forwarding_v1:command_api_contract": "pass",
    "forwarding_v1:target_policy": "pass",
    "forwarding_v1:proof_facts": "blocked",
    "forwarding_v1:rollback": "pass",
    "forwarding_v1:read_only_preview": "pass",
}
for key, want in expected_item_status.items():
    got = items.get(key, {}).get("status")
    if got != want:
        sys.exit(f"item {key} status={got!r}, want {want!r}")

for field in ["command_type", "forwarded_task_type", "target_command_type", "approval_id", "readiness_snapshot_hash", "rollback_plan_id", "failure_mode"]:
    if field not in data["apply_packet_fields"]:
        sys.exit(f"missing apply packet field {field}")
for field in ["legacy_task_loop_started", "legacy_progress_written", "audit_event_id"]:
    if field not in data["fail_closed_fields"]:
        sys.exit(f"missing fail-closed field {field}")
for fact in ["legacy_task_loop_runner_not_started", "forwarded_task_type_policy_enforced", "blocked_task_types_fail_closed", "rollback_to_read_only_shim_verified"]:
    if fact not in data["required_proof_facts"]:
        sys.exit(f"missing required proof fact {fact}")
targets = {target["task_type"]: target for target in data["forwarding_targets"]}
if set(targets) != set(data["allowed_task_types"]):
    sys.exit(f"forwarding targets do not match allowed tasks: targets={sorted(targets)} allowed={data['allowed_task_types']}")
for task_type, target in targets.items():
    if target["failure_mode"] != "fail_closed":
        sys.exit(f"target {task_type} failure_mode={target['failure_mode']!r}, want fail_closed")
    for key in ["project_write_allowed", "execution_write_allowed", "legacy_fallback_allowed"]:
        if target[key] is not False:
            sys.exit(f"target {task_type} {key}={target[key]!r}, want false")
for task_type in ["read_only_verify", "artifact_evidence"]:
    if not targets[task_type]["creates_command_request"]:
        sys.exit(f"target {task_type} should create command request in approved scope")
blocked_targets = {target["task_type"]: target for target in data["blocked_targets"]}
for task_type in ["copy_ready_source_write", "generated_retained_write", "repair_apply", "checkpoint_apply", "engine_execution", "secret_resolve", "network_api_integration", "publish_apply", "restore_apply"]:
    target = blocked_targets.get(task_type)
    if not target:
        sys.exit(f"missing blocked target {task_type}")
    if target["failure_mode"] != "fail_closed":
        sys.exit(f"blocked target {task_type} failure_mode={target['failure_mode']!r}, want fail_closed")
    facts = target["safety_facts"]
    for key in ["legacy_task_loop_started", "project_write_attempted", "execution_write_attempted", "engine_call_attempted", "commands_run", "secrets_resolved", "network_used"]:
        if facts.get(key) is not False:
            sys.exit(f"blocked target {task_type} safety {key}={facts.get(key)!r}, want false")
for key in [
    "apply_open",
    "forwarding_v1_apply_open",
    "task_loop_run_forwarded",
    "legacy_task_loop_started",
    "legacy_progress_written",
    "legacy_logs_written",
    "legacy_checkpoint_written",
    "project_write_attempted",
    "execution_write_attempted",
    "area_flow_command_created",
    "area_flow_run_created",
    "worker_scheduled",
    "engine_call_attempted",
    "commands_run",
    "secrets_resolved",
    "network_used",
    "source_write_open",
    "generated_retained_write_open",
    "repair_apply_open",
    "checkpoint_apply_open",
    "publish_apply_open",
    "restore_apply_open",
    "areamatrix_protected_paths_touched",
]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
if safety.get("read_only_preview") is not True:
    sys.exit("safety fact read_only_preview should be true")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply-packet missing approval --json"
apply_packet_missing_json="$(go run ./cmd/areaflow project execution-forwarding-v1-apply-packet "${project_key}" --json)"
APPLY_PACKET_JSON="${apply_packet_missing_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["APPLY_PACKET_JSON"])
packet = data["packet"]
gate = data["gate"]
safety = data["safety_facts"]

if data["status"] != "blocked" or data["decision"] != "readiness_blocked":
    sys.exit(f"packet status={data['status']!r} decision={data['decision']!r}, want blocked/readiness_blocked")
if packet["command_type"] != "project.execution_forwarding_v1.apply":
    sys.exit(f"command_type={packet['command_type']!r}")
if packet["approval_scope"] != "execution_forwarding_v1_read_only_evidence_only":
    sys.exit(f"approval_scope={packet['approval_scope']!r}")
if packet["expected_shim_lifecycle_state"] != "read_only_shim":
    sys.exit(f"expected_shim_lifecycle_state={packet['expected_shim_lifecycle_state']!r}")
if packet["failure_mode"] != "fail_closed":
    sys.exit(f"failure_mode={packet['failure_mode']!r}")
if not packet["readiness_snapshot_hash"]:
    sys.exit("missing readiness_snapshot_hash")
if gate["apply_command_eligible"] is not False or gate["decision"] != "no_go":
    sys.exit(f"gate should be no_go, got {gate['decision']=} {gate['apply_command_eligible']=}")
items = {item["key"]: item for item in gate["items"]}
for key in ["read_only_shim", "approval_id", "explicit_approval", "legacy_non_write_proof_id", "rollback_plan_id", "protected_path_fingerprint_id"]:
    if items.get(key, {}).get("status") != "blocked":
        sys.exit(f"gate item {key} status={items.get(key, {}).get('status')!r}, want blocked")
for key in ["command_request_created", "area_flow_run_created", "task_loop_run_forwarded", "project_write_attempted", "execution_write_attempted", "engine_call_attempted"]:
    if safety.get(key) is not False:
        sys.exit(f"packet safety {key}={safety.get(key)!r}, want false")
if safety.get("read_only_preview") is not True:
    sys.exit("packet should be read-only preview")
PY

readiness_snapshot_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["readiness_snapshot_hash"])' <<<"${apply_packet_missing_json}")"
idempotency_key="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["idempotency_key"])' <<<"${apply_packet_missing_json}")"
audit_correlation_id="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["audit_correlation_id"])' <<<"${apply_packet_missing_json}")"
allowed_task_types="$(python3 -c 'import json,sys; print(",".join(json.load(sys.stdin)["packet"]["allowed_task_types"]))' <<<"${apply_packet_missing_json}")"
legacy_non_write_proof_id="$(PROJECT_KEY="${project_key}" READINESS_JSON="${readiness_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["READINESS_JSON"])
project_key = os.environ["PROJECT_KEY"]
items = {item["key"]: item for item in data["items"]}
event_id = items["legacy_non_write_proof"]["metadata"].get("proof_event_id")
if not event_id:
    sys.exit("legacy proof_event_id missing")
print(f"{project_key}:legacy_non_write_proof:{event_id}")
PY
)"
rollback_plan_id="$(PROJECT_KEY="${project_key}" READINESS_JSON="${readiness_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["READINESS_JSON"])
project_key = os.environ["PROJECT_KEY"]
items = {item["key"]: item for item in data["items"]}
event_id = items["rollback_to_read_only_shim"]["metadata"].get("proof_event_id")
if not event_id:
    sys.exit("rollback proof_event_id missing")
print(f"{project_key}:rollback_to_read_only_shim:{event_id}")
PY
)"
protected_path_fingerprint_id="$(PROJECT_KEY="${project_key}" READINESS_JSON="${readiness_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["READINESS_JSON"])
project_key = os.environ["PROJECT_KEY"]
items = {item["key"]: item for item in data["items"]}
protected_path_set_hash = items["legacy_non_write_proof"]["metadata"].get("protected_path_set_hash")
if not protected_path_set_hash:
    sys.exit("protected_path_set_hash missing")
print(f"{project_key}:protected_path_fingerprint:{protected_path_set_hash}")
PY
)"

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply-packet unscoped proof ids block --json"
apply_packet_unscoped_json="$(
  go run ./cmd/areaflow project execution-forwarding-v1-apply-packet "${project_key}" --json \
    --explicit-approval \
    --approval-id "forwarding-v1-approval-${workflow_label}" \
    --approval-actor "smoke-execution-forwarding-v1-readiness" \
    --approval-reason "fixture forwarding v1 packet review" \
    --legacy-non-write-proof-id "protected-path-proof-${workflow_label}" \
    --rollback-plan-id "rollback-plan-${workflow_label}" \
    --protected-path-fingerprint-id "fingerprint-${workflow_label}" \
    --idempotency-key "${idempotency_key}" \
    --audit-correlation-id "${audit_correlation_id}"
)"
APPLY_PACKET_JSON="${apply_packet_unscoped_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["APPLY_PACKET_JSON"])
items = {item["key"]: item for item in data["gate"]["items"]}

if data["status"] != "blocked" or data["decision"] != "readiness_blocked":
    sys.exit(f"unscoped packet should be blocked/readiness_blocked, got {data['status']=} {data['decision']=}")
for key in ["legacy_non_write_proof_id", "rollback_plan_id", "protected_path_fingerprint_id"]:
    item = items.get(key)
    if not item or item.get("status") != "blocked":
        sys.exit(f"gate item {key} should block unscoped proof ref: {item}")
    if f"{key}_missing_or_mismatch" not in item.get("blocked_by", []):
        sys.exit(f"gate item {key} blocker mismatch: {item}")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply-packet complete proof ids --json"
apply_packet_complete_json="$(
  go run ./cmd/areaflow project execution-forwarding-v1-apply-packet "${project_key}" --json \
    --explicit-approval \
    --approval-id "forwarding-v1-approval-${workflow_label}" \
    --approval-actor "smoke-execution-forwarding-v1-readiness" \
    --approval-reason "fixture forwarding v1 packet review" \
    --legacy-non-write-proof-id "${legacy_non_write_proof_id}" \
    --rollback-plan-id "${rollback_plan_id}" \
    --protected-path-fingerprint-id "${protected_path_fingerprint_id}" \
    --idempotency-key "${idempotency_key}" \
    --audit-correlation-id "${audit_correlation_id}"
)"
APPLY_PACKET_JSON="${apply_packet_complete_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["APPLY_PACKET_JSON"])
packet = data["packet"]
gate = data["gate"]

if data["status"] != "blocked" or data["decision"] != "readiness_blocked":
    sys.exit(f"complete packet should still be readiness blocked, got {data['status']=} {data['decision']=}")
if packet["explicit_approval"] is not True or packet["approval_id"] == "":
    sys.exit(f"approval fields missing: {packet}")
if packet["legacy_non_write_proof_id"] == "" or packet["rollback_plan_id"] == "" or packet["protected_path_fingerprint_id"] == "":
    sys.exit(f"proof ids missing: {packet}")
if gate["apply_command_eligible"] is not False or gate["approval_status"] != "missing_or_incomplete":
    sys.exit(f"gate should remain ineligible while read-only shim is blocked: {gate}")
items = {item["key"]: item for item in gate["items"]}
for key in ["approval_id", "explicit_approval", "legacy_non_write_proof_id", "rollback_plan_id", "protected_path_fingerprint_id"]:
    if items.get(key, {}).get("status") != "pass":
        sys.exit(f"gate item {key} status={items.get(key, {}).get('status')!r}, want pass")
if items.get("read_only_shim", {}).get("status") != "blocked":
    sys.exit(f"read_only_shim should stay blocked: {items.get('read_only_shim')}")
if items.get("rollback_to_read_only_shim", {}).get("status") != "pass":
    sys.exit(f"rollback_to_read_only_shim should pass after proof facts: {items.get('rollback_to_read_only_shim')}")
if "execution-forwarding-v1-apply-gate" not in data["apply_gate_command"]:
    sys.exit(f"missing apply gate command: {data['apply_gate_command']}")
if "execution-forwarding-v1-apply" not in data["future_apply_command"]:
    sys.exit(f"missing future apply command: {data['future_apply_command']}")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply-gate missing packet --json"
apply_gate_missing_json="$(go run ./cmd/areaflow project execution-forwarding-v1-apply-gate "${project_key}" --json)"
assert_contains "${apply_gate_missing_json}" '"status": "blocked"'
assert_contains "${apply_gate_missing_json}" '"decision": "no_go"'
assert_contains "${apply_gate_missing_json}" '"apply_command_eligible": false'
assert_contains "${apply_gate_missing_json}" '"key": "readiness_snapshot_hash"'
assert_contains "${apply_gate_missing_json}" '"key": "explicit_approval"'
assert_contains "${apply_gate_missing_json}" '"command_request_created": false'
assert_contains "${apply_gate_missing_json}" '"task_loop_run_forwarded": false'
assert_contains "${apply_gate_missing_json}" '"project_write_attempted": false'

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply-gate complete packet blocked by shim --json"
apply_gate_complete_json="$(
  go run ./cmd/areaflow project execution-forwarding-v1-apply-gate "${project_key}" --json \
    --allowed-task-types "${allowed_task_types}" \
    --approval-id "forwarding-v1-approval-${workflow_label}" \
    --approval-scope "execution_forwarding_v1_read_only_evidence_only" \
    --readiness-snapshot-hash "${readiness_snapshot_hash}" \
    --expected-shim-lifecycle-state "read_only_shim" \
    --legacy-non-write-proof-id "${legacy_non_write_proof_id}" \
    --rollback-plan-id "${rollback_plan_id}" \
    --protected-path-fingerprint-id "${protected_path_fingerprint_id}" \
    --failure-mode "fail_closed" \
    --idempotency-key "${idempotency_key}" \
    --audit-correlation-id "${audit_correlation_id}" \
    --explicit-approval \
    --approval-actor "smoke-execution-forwarding-v1-readiness" \
    --approval-reason "fixture forwarding v1 packet review"
)"
APPLY_GATE_JSON="${apply_gate_complete_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["APPLY_GATE_JSON"])
items = {item["key"]: item for item in data["items"]}
safety = data["safety_facts"]

if data["status"] != "blocked" or data["decision"] != "no_go" or data["apply_command_eligible"] is not False:
    sys.exit(f"gate should stay blocked/no_go until read-only shim lands: {data}")
for key in ["allowed_task_types", "readiness_snapshot_hash", "approval_id", "approval_scope", "explicit_approval", "legacy_non_write_proof_id", "rollback_plan_id", "protected_path_fingerprint_id", "failure_mode"]:
    if items.get(key, {}).get("status") != "pass":
        sys.exit(f"gate item {key} status={items.get(key, {}).get('status')!r}, want pass")
if items.get("read_only_shim", {}).get("status") != "blocked":
    sys.exit(f"read_only_shim should block complete packet: {items.get('read_only_shim')}")
if items.get("rollback_to_read_only_shim", {}).get("status") != "pass":
    sys.exit(f"rollback_to_read_only_shim should pass complete packet: {items.get('rollback_to_read_only_shim')}")
for key in ["command_request_created", "area_flow_run_created", "task_loop_run_forwarded", "project_write_attempted", "execution_write_attempted", "engine_call_attempted"]:
    if safety.get(key) is not False:
        sys.exit(f"gate safety {key}={safety.get(key)!r}, want false")
if safety.get("read_only_gate") is not True:
    sys.exit("gate should be read-only")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-apply complete packet blocked by shim --json"
apply_result_json="$(
  go run ./cmd/areaflow project execution-forwarding-v1-apply "${project_key}" --json \
    --allowed-task-types "${allowed_task_types}" \
    --approval-id "forwarding-v1-approval-${workflow_label}" \
    --approval-scope "execution_forwarding_v1_read_only_evidence_only" \
    --readiness-snapshot-hash "${readiness_snapshot_hash}" \
    --expected-shim-lifecycle-state "read_only_shim" \
    --legacy-non-write-proof-id "${legacy_non_write_proof_id}" \
    --rollback-plan-id "${rollback_plan_id}" \
    --protected-path-fingerprint-id "${protected_path_fingerprint_id}" \
    --failure-mode "fail_closed" \
    --idempotency-key "${idempotency_key}" \
    --audit-correlation-id "${audit_correlation_id}" \
    --explicit-approval \
    --approval-actor "smoke-execution-forwarding-v1-readiness" \
    --approval-reason "fixture forwarding v1 protected apply"
)"
APPLY_RESULT_JSON="${apply_result_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["APPLY_RESULT_JSON"])
safety = data["safety_facts"]

if data["status"] != "blocked" or data["decision"] != "denied":
    sys.exit(f"apply command should stay blocked/denied until read-only shim lands: {data}")
if "execution_forwarding_v1_apply_gate_blocked" not in data["blockers"]:
    sys.exit(f"missing gate blocker: {data['blockers']}")
if "read_only_shim_not_pass" not in data["blockers"]:
    sys.exit(f"missing read-only shim blocker: {data['blockers']}")
if "rollback_to_read_only_shim_not_pass" in data["blockers"]:
    sys.exit(f"rollback proof should be closed, unexpected blocker: {data['blockers']}")
if data["gate"]["status"] != "blocked" or data["gate"]["decision"] != "no_go":
    sys.exit(f"nested gate should be blocked/no_go: {data['gate']}")
if not data["event_id"] or not data["audit_event_id"]:
    sys.exit(f"apply command should record event and audit ids: {data}")
for key in ["command_request_created", "area_flow_command_created", "area_flow_audit_event_created"]:
    if data.get(key) is not True:
        sys.exit(f"{key}={data.get(key)!r}, want true")
for key in [
    "area_flow_run_created",
    "area_flow_run_task_created",
    "area_flow_run_attempt_created",
    "area_flow_artifact_created",
    "task_loop_run_forwarded",
    "legacy_task_loop_started",
    "legacy_progress_written",
    "legacy_logs_written",
    "legacy_checkpoint_written",
    "project_write_attempted",
    "execution_write_attempted",
    "engine_call_attempted",
    "commands_run",
    "secrets_resolved",
    "network_used",
    "areamatrix_protected_paths_touched",
]:
    if data.get(key) is not False:
        sys.exit(f"{key}={data.get(key)!r}, want false")
if safety.get("apply_command_executed") is not True:
    sys.exit(f"apply command safety fact missing: {safety}")
if safety.get("command_request_created") is not True:
    sys.exit(f"command request safety fact missing: {safety}")
if safety.get("forwarding_v1_apply_open") is not False:
    sys.exit(f"blocked apply must not open forwarding: {safety}")
for key in ["area_flow_run_created", "task_loop_run_forwarded", "project_write_attempted", "execution_write_attempted", "engine_call_attempted", "commands_run", "secrets_resolved", "network_used"]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-command-preview allowed --json"
command_preview_allowed_json="$(go run ./cmd/areaflow project execution-forwarding-v1-command-preview "${project_key}" --task-type read_only_verify --json)"
COMMAND_PREVIEW_JSON="${command_preview_allowed_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["COMMAND_PREVIEW_JSON"])
safety = data["safety_facts"]

if data["status"] != "blocked":
    sys.exit(f"status={data['status']!r}, want blocked while apply is closed")
if data["mode"] != "read_only_execution_forwarding_v1_command_preview":
    sys.exit(f"mode={data['mode']!r}, want command preview")
if data["decision"] != "would_forward_after_approval":
    sys.exit(f"decision={data['decision']!r}, want would_forward_after_approval")
if data["task_type"] != "read_only_verify" or data["target_command_type"] != "run.read_only_verify_queue":
    sys.exit(f"unexpected target mapping: {data}")
if not data["allowed_task_type"] or data["blocked_task_type"]:
    sys.exit(f"unexpected target flags: {data}")
if data["apply_open"] is not False:
    sys.exit("command preview apply_open should stay false")
if not data["would_create_command_request_after_approval"] or not data["would_create_run_after_approval"]:
    sys.exit(f"allowed target should describe post-approval command/run creation: {data}")
for key in ["project_write_allowed", "execution_write_allowed", "legacy_fallback_allowed"]:
    if data[key] is not False:
        sys.exit(f"{key}={data[key]!r}, want false")
for field in ["forwarded_task_type", "target_command_type", "audit_correlation_id"]:
    if field not in data["required_packet_fields"]:
        sys.exit(f"missing required packet field {field}")
for field in ["legacy_task_loop_started", "audit_event_id"]:
    if field not in data["fail_closed_fields"]:
        sys.exit(f"missing fail-closed field {field}")
for key in ["area_flow_command_created", "area_flow_run_created", "task_loop_run_forwarded", "legacy_task_loop_started", "project_write_attempted", "execution_write_attempted", "engine_call_attempted", "commands_run", "secrets_resolved", "network_used"]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
if safety.get("command_preview") is not True or safety.get("read_only_preview") is not True:
    sys.exit(f"preview safety facts should be true: {safety}")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-command-preview blocked --json"
command_preview_blocked_json="$(go run ./cmd/areaflow project execution-forwarding-v1-command-preview "${project_key}" --task-type engine_execution --json)"
COMMAND_PREVIEW_JSON="${command_preview_blocked_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["COMMAND_PREVIEW_JSON"])
safety = data["safety_facts"]

if data["decision"] != "blocked_task_type_fail_closed":
    sys.exit(f"decision={data['decision']!r}, want blocked_task_type_fail_closed")
if data["target_status"] != "blocked" or not data["blocked_task_type"] or data["allowed_task_type"]:
    sys.exit(f"blocked target flags missing: {data}")
if "engine_execution" not in data["blocked_by"]:
    sys.exit(f"blocked_by missing engine_execution: {data['blocked_by']}")
for key in ["area_flow_command_created", "area_flow_run_created", "task_loop_run_forwarded", "legacy_task_loop_started", "project_write_attempted", "execution_write_attempted", "engine_call_attempted", "commands_run", "secrets_resolved", "network_used"]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-command-preview unknown --json"
command_preview_unknown_json="$(go run ./cmd/areaflow project execution-forwarding-v1-command-preview "${project_key}" --task-type surprise_task --json)"
COMMAND_PREVIEW_JSON="${command_preview_unknown_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["COMMAND_PREVIEW_JSON"])
safety = data["safety_facts"]

if data["decision"] != "unknown_task_type_fail_closed":
    sys.exit(f"decision={data['decision']!r}, want unknown_task_type_fail_closed")
if data["allowed_task_type"] or data["blocked_task_type"]:
    sys.exit(f"unknown target should not be marked allowed or known blocked: {data}")
if "task_type_not_in_forwarding_v1_policy" not in data["blocked_by"]:
    sys.exit(f"unknown blocker missing: {data['blocked_by']}")
for key in ["area_flow_command_created", "task_loop_run_forwarded", "legacy_task_loop_started", "project_write_attempted", "execution_write_attempted", "commands_run"]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
PY

echo "smoke-execution-forwarding-v1-readiness: project execution-forwarding-v1-rollback-preview --json"
rollback_preview_json="$(go run ./cmd/areaflow project execution-forwarding-v1-rollback-preview "${project_key}" --json)"
ROLLBACK_PREVIEW_JSON="${rollback_preview_json}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["ROLLBACK_PREVIEW_JSON"])
items = {item["key"]: item for item in data["items"]}
safety = data["safety_facts"]

if data["status"] != "blocked":
    sys.exit(f"status={data['status']!r}, want blocked")
if data["mode"] != "read_only_execution_forwarding_v1_rollback_preview":
    sys.exit(f"mode={data['mode']!r}, want read_only_execution_forwarding_v1_rollback_preview")
if data["apply_preview"]["status"] != "blocked":
    sys.exit(f"nested apply preview status={data['apply_preview']['status']!r}, want blocked")
if data["rollback_target"] != "read_only_shim":
    sys.exit(f"rollback_target={data['rollback_target']!r}, want read_only_shim")
if data["rollback_apply_open"] is not False:
    sys.exit("rollback_apply_open should stay false")

expected_item_status = {
    "rollback_v1:apply_preview": "pass",
    "rollback_v1:fail_closed": "pass",
    "rollback_v1:proof_facts": "pass",
    "rollback_v1:reopen_conditions": "blocked",
    "rollback_v1:read_only_preview": "pass",
}
for key, want in expected_item_status.items():
    got = items.get(key, {}).get("status")
    if got != want:
        sys.exit(f"item {key} status={got!r}, want {want!r}")

for fact in ["task_loop_run_forwarding_disabled", "protected_path_proof_clean_after_rollback_recorded"]:
    if fact not in data["required_proof_facts"]:
        sys.exit(f"missing required rollback proof fact {fact}")
for action in ["create_rollback_command", "delete_forwarding_history", "restore_apply"]:
    if action not in data["forbidden_actions"]:
        sys.exit(f"missing forbidden rollback action {action}")
for step in data["fail_closed_steps"]:
    if "./task-loop run" in step:
        break
else:
    sys.exit("missing task-loop run fail-closed step")
fail_closed = items["rollback_v1:fail_closed"]["metadata"]
if fail_closed.get("fail_closed_preview_proven") is not True:
    sys.exit(f"fail-closed proof metadata missing: {fail_closed}")
if fail_closed.get("legacy_non_write_proof") != "pass":
    sys.exit(f"legacy proof status in rollback metadata={fail_closed.get('legacy_non_write_proof')!r}, want pass")
proof_facts = items["rollback_v1:proof_facts"]["metadata"]
if proof_facts.get("proof_present") is not True:
    sys.exit(f"rollback proof should be present: {proof_facts}")
if proof_facts.get("proof_status") != "complete":
    sys.exit(f"rollback proof status={proof_facts.get('proof_status')!r}, want complete")
if proof_facts.get("missing_proof_facts") not in ([], None):
    sys.exit(f"rollback proof should have no missing facts: {proof_facts}")
for key in [
    "project_write_attempted",
    "execution_write_attempted",
    "task_loop_run_forwarded_by_command",
    "commands_run",
    "legacy_progress_written",
    "legacy_logs_written",
    "legacy_checkpoint_written",
    "areamatrix_protected_paths_touched",
]:
    if proof_facts.get(key) is not False:
        sys.exit(f"rollback proof fact {key}={proof_facts.get(key)!r}, want false")
for key in [
    "rollback_apply_open",
    "apply_open",
    "forwarding_v1_apply_open",
    "task_loop_run_forwarded",
    "legacy_task_loop_started",
    "legacy_progress_written",
    "legacy_logs_written",
    "legacy_checkpoint_written",
    "project_write_attempted",
    "execution_write_attempted",
    "area_flow_command_created",
    "area_flow_run_created",
    "worker_scheduled",
    "engine_call_attempted",
    "commands_run",
    "secrets_resolved",
    "network_used",
    "source_write_open",
    "generated_retained_write_open",
    "repair_apply_open",
    "checkpoint_apply_open",
    "publish_apply_open",
    "restore_apply_open",
    "areamatrix_protected_paths_touched",
]:
    if safety.get(key) is not False:
        sys.exit(f"safety fact {key}={safety.get(key)!r}, want false")
if safety.get("read_only_preview") is not True:
    sys.exit("safety fact read_only_preview should be true")
PY

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"
  real_protected_path_fingerprint_after="$(real_areamatrix_protected_path_fingerprint)"
  real_protected_path_status_after="$(real_areamatrix_protected_path_git_status)"
  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi
  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
  if [[ "${real_protected_path_fingerprint_before}" != "${real_protected_path_fingerprint_after}" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: real AreaMatrix protected path fingerprint changed unexpectedly:" >&2
    echo "smoke-execution-forwarding-v1-readiness: protected_path_fingerprint_before=${real_protected_path_fingerprint_before}" >&2
    echo "smoke-execution-forwarding-v1-readiness: protected_path_fingerprint_after=${real_protected_path_fingerprint_after}" >&2
    exit 1
  fi
  if [[ "${real_protected_path_status_before}" != "${real_protected_path_status_after}" ]]; then
    echo "smoke-execution-forwarding-v1-readiness: real AreaMatrix protected path git status changed unexpectedly:" >&2
    echo "${real_protected_path_status_after}" >&2
    exit 1
  fi
fi

echo "smoke-execution-forwarding-v1-readiness: pass ${project_key} fixture=${fixture_dir}"
