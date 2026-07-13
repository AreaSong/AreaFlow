#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-completion-audit-full-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="areamatrix"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-completion-audit-full.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
evidence_root="docs/history/v1.0/evidence/real-release-candidate-evidence.md"
archive_evidence_uri="${evidence_root}#e4-archive"
shim_evidence_uri="${evidence_root}#e4-shim-retirement"
execution_cutover_evidence_uri="${evidence_root}#e4-execution-cutover"
review_flags=(--review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z)
real_areamatrix_status="/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
real_areamatrix_readme="/Users/as/Ai-Project/project/AreaMatrix/workflow/README.md"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-completion-audit-full-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-completion-audit-full-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-full-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_line() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fxq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-full-proof: expected output line: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-full-proof: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_completion_items_not_blocked_by() {
  local output="$1"
  local blocker="$2"

  JSON_PAYLOAD="${output}" BLOCKER="${blocker}" python3 - <<'PY'
import json
import os
import sys

data = json.loads(os.environ["JSON_PAYLOAD"])
blocker = os.environ["BLOCKER"]
matches = [
    item.get("key", "<unknown>")
    for item in data.get("items", [])
    if blocker in item.get("blocked_by", [])
]
if matches:
    print(
        "smoke-completion-audit-full-proof: completion items unexpectedly blocked by "
        f"{blocker}: {','.join(matches)}",
        file=sys.stderr,
    )
    sys.exit(1)
PY
}

assert_fails_contains() {
  local pattern="$1"
  shift
  local output

  set +e
  output="$("$@" 2>&1)"
  local rc=$?
  set -e
  if [[ ${rc} -eq 0 ]]; then
    echo "smoke-completion-audit-full-proof: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
}

json_get() {
  local payload="$1"
  local path="$2"

  JSON_PAYLOAD="${payload}" JSON_PATH="${path}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["JSON_PAYLOAD"])
path = os.environ["JSON_PATH"]

if path.startswith("len:"):
    value = data
    subpath = path[4:]
    if subpath:
        for part in subpath.split("."):
            value = value[part]
    print(len(value))
else:
    value = data
    for part in path.split("."):
        value = value[part]
    if isinstance(value, bool):
        print(str(value).lower())
    else:
        print(value)
PY
}

assert_positive_value() {
  local label="$1"
  local value="$2"

  if ! [[ "${value}" =~ ^[1-9][0-9]*$ ]]; then
    echo "smoke-completion-audit-full-proof: expected positive ${label}, got ${value}" >&2
    exit 1
  fi
}

assert_sha256_value() {
  local label="$1"
  local value="$2"

  if ! [[ "${value}" =~ ^[0-9a-f]{64}$ ]]; then
    echo "smoke-completion-audit-full-proof: expected sha256 ${label}, got ${value}" >&2
    exit 1
  fi
}

task_matrix_source_set_hash() {
  local backlog_hash="$1"
  local status_audit_hash="$2"

  python3 - "${backlog_hash}" "${status_audit_hash}" <<'PY'
import hashlib
import json
import sys

payload = {
    "required_count_contract": "planned/missing_evidence/blocked_v1_required_counts_must_be_zero",
    "source_paths": [
        "docs/history/v1.0/plans/task-backlog.md",
        "docs/history/v1.0/evidence/task-backlog-status-audit.md",
    ],
    "task_backlog_hash": sys.argv[1],
    "task_status_audit_hash": sys.argv[2],
}
print(hashlib.sha256(json.dumps(payload, sort_keys=True, separators=(",", ":")).encode()).hexdigest())
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

real_status_before="__skipped__"
real_readme_before="__skipped__"
if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_before="$(file_fingerprint "${real_areamatrix_readme}")"
else
  echo "smoke-completion-audit-full-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p \
  "${project_root}/docs" \
  "${project_root}/workflow/residuals" \
  "${project_root}/workflow/templates" \
  "${project_root}/workflow/versions/v1-mvp/execution/_shared" \
  "${project_root}/workflow/versions/v1-mvp/residuals" \
  "${project_root}/workflow/versions/v-template" \
  "${project_root}/tasks/indexes" \
  "${project_root}/tasks/backlog" \
  "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Completion Audit Full Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Completion Audit Full Proof Fixture Workflow
EOF

cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items:
  - id: global-fixture-note
    status: reference-only
    type: fixture
    title: Completion audit full proof fixture residual
    source: workflow/residuals/residuals.yaml
    current_impact: proves global residual inventory exists
    executable_task: false
    promotion_required: false
    close_condition: fixture smoke passes
version_residuals:
  - version: v1-mvp
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    status: fixture-reviewed
    summary: Completion audit full proof fixture version residuals
EOF

cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Completion Audit Full Proof Fixture Templates
EOF

cat >"${project_root}/workflow/versions/v1-mvp/version.yaml" <<'EOF'
version: v1-mvp
display_label: v1-mvp
version_kind: workflow_version
project_id: areamatrix
EOF

cat >"${project_root}/workflow/versions/v1-mvp/residuals/residuals.yaml" <<'EOF'
version_status:
  technical_queue: complete
  fixture: true
items:
  - id: v1-fixture-residual
    status: reference-only
    type: fixture
    title: Completion audit full proof v1 fixture residual
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    current_impact: proves version residual inventory exists
    executable_task: false
    promotion_required: false
    close_condition: fixture smoke passes
EOF

cat >"${project_root}/workflow/versions/v1-mvp/execution/_shared/progress.json" <<'EOF'
{
  "tasks": {
    "fixture-001": {"status": "completed"},
    "fixture-002": {"status": "completed"}
  }
}
EOF

cat >"${project_root}/workflow/versions/v-template/README.md" <<'EOF'
# Completion Audit Full Proof Fixture Template Version
EOF

cat >"${project_root}/tasks/indexes/residuals.md" <<'EOF'
# Completion Audit Full Proof Fixture Residual Index
EOF

cat >"${project_root}/tasks/backlog/README.md" <<'EOF'
# Completion Audit Full Proof Fixture Backlog
EOF

local_release_artifact="${artifact_root}/completion-audit-release-readiness.json"
cat >"${local_release_artifact}" <<'EOF'
{"kind":"completion_audit_full_proof_fixture","release_readiness_local_artifact":true}
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Completion Audit Full Proof Fixture
  root: ${project_root}
  kind: product-repo
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
    write_status: true
    write_artifacts: true
    write_workflow: false
    write_generated: false
    write_code: false
    run_commands: false
    manage_workers: false
    manage_git: false
    network: false
    use_secrets: false
    execute_agents: false

  read_paths:
    - docs/**
    - workflow/**
    - tasks/**

  write_paths:
    - .areaflow/status.json

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
      resource_limits:
        max_active_leases: 1
        max_queued_tasks: 20

status_export:
  enabled: true
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
    - v-template
  immutable_imports:
    - v1-mvp
EOF

source_alignment_facts=(
  zero_to_hundred_phases_aligned
  v1_and_v1x_boundaries_consistent
  preview_only_not_claimed_as_apply
  implemented_scoped_not_claimed_as_real_cutover
  deferred_high_risk_capabilities_have_contracts
  master_plan_roadmap_phase_backlog_gap_audit_cross_references_current
)
task_matrix_facts=(
  all_v0_v1_tasks_have_status_evidence_and_boundary
  no_planned_v1_required_task_hidden
  preview_only_items_have_evidence_or_explicit_boundary
  implemented_scoped_items_have_scope_labels
  nearest_open_task_has_next_command_and_required_evidence
  v1x_deferred_tasks_have_contracts
)
validation_facts=(
  go_test_passed
  go_build_passed
  web_build_passed
  git_diff_check_passed
  v1_stable_fixture_smoke_passed
  web_smoke_passed
  project_isolation_smoke_passed
  completion_proof_smoke_passed
  validation_did_not_touch_areamatrix_protected_paths
)
archive_facts=(
  historical_workflow_versions_marked_immutable
  historical_execution_metadata_indexed_in_areaflow
  historical_artifact_refs_have_hash_path_type_project_version_run
  project_reference_restore_limitations_recorded
  old_progress_logs_checkpoints_are_reference_only
  new_run_attempt_artifact_audit_state_owned_by_areaflow
  areamatrix_workflow_readme_summary_contract_reviewed
  areamatrix_status_json_rough_projection_contract_reviewed
  archive_does_not_delete_or_move_historical_files
  archive_does_not_rewrite_progress_json
  rollback_to_execution_forwarding_documented
)
shim_facts=(
  archive_gate_passed
  execution_forwarding_stable_for_declared_window
  no_legacy_task_loop_run_usage_in_active_workflow_versions
  areaflow_run_attempt_artifact_audit_coverage_pass
  compat_commands_mapped_or_deliberately_blocked
  legacy_progress_log_checkpoint_archive_reference_policy_accepted
  rollback_to_read_only_shim_documented
  user_facing_retirement_notice_present
  protected_path_proof_reference_recorded
)
execution_cutover_facts=(
  explicit_execution_cutover_approval_recorded
  execution_cutover_command_response_recorded
  execution_cutover_event_and_audit_recorded
  task_loop_run_forwarding_window_proven
  rollback_plan_and_compatibility_window_proven
  no_unapproved_project_or_execution_write_attempted
)
release_packaging_facts=(
  release_final_gate_passed
  release_evidence_bundle_metadata_only
  release_package_preview_created_no_package
  distribution_preview_no_upload_sign_tag_push
  publish_gate_and_approval_preview_created_no_publish_or_approval
  rollout_plan_preview_created_no_rollout_state
  no_release_package_publish_rollout_apply_opened
)
backup_restore_facts=(
  backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata
  restore_dry_run_identifies_metadata_only_history_and_object_verifier_limits
  artifact_integrity_distinguishes_local_project_reference_external_and_object
  archive_preview_does_not_copy_upload_delete_or_gc_artifact_bytes
  retention_classes_and_accepted_exceptions_are_documented
  no_restore_apply_or_artifact_mutation_opened
)
security_closure_facts=(
  project_key_isolation_covers_workflow_run_lease_artifact_secret_audit
  global_id_route_guard_project_key_visibility_proven
  permission_doctor_default_read_only_deny_first_passed
  audit_coverage_covers_enabled_capabilities
  auth_team_token_secret_remote_worker_remain_readiness_only
  no_forbidden_v1_security_capability_opened
)

source_alignment_fact_args=()
for fact in "${source_alignment_facts[@]}"; do
  source_alignment_fact_args+=(--fact "${fact}")
done
task_matrix_fact_args=()
for fact in "${task_matrix_facts[@]}"; do
  task_matrix_fact_args+=(--fact "${fact}")
done
task_backlog_hash="$(shasum -a 256 docs/history/v1.0/plans/task-backlog.md | awk '{print $1}')"
task_status_audit_hash="$(shasum -a 256 docs/history/v1.0/evidence/task-backlog-status-audit.md | awk '{print $1}')"
task_matrix_source_set_hash_value="$(task_matrix_source_set_hash "${task_backlog_hash}" "${task_status_audit_hash}")"
task_matrix_binding_args=(
  --source-set-hash "${task_matrix_source_set_hash_value}"
  --backlog-hash "${task_backlog_hash}"
  --task-status-audit-hash "${task_status_audit_hash}"
  --planned-v1-required-task-count 0
  --missing-evidence-v1-required-task-count 0
  --blocked-v1-required-task-count 0
)
validation_fact_args=()
for fact in "${validation_facts[@]}"; do
  validation_fact_args+=(--fact "${fact}")
done
validation_commands=(
  "go test ./..."
  "go build ./cmd/areaflow"
  "npm --prefix web run build"
  "npm --prefix desktop run build"
  "git diff --check -- ."
  "make smoke-docker-v1-stable-fixture"
  "make smoke-docker-web"
  "make smoke-docker-project-isolation"
  "make smoke-docker-completion-proof"
)
validation_command_args=()
for command in "${validation_commands[@]}"; do
  validation_command_args+=(--validation-command "${command}")
done
validation_result_hash="$(printf "%s\n" "${validation_commands[@]}" | shasum -a 256 | awk '{print $1}')"
validation_started_at="2026-07-06T10:00:00Z"
validation_finished_at="2026-07-06T10:30:00Z"
validation_scope="fixture_completion_audit_full_review"
archive_fact_args=()
for fact in "${archive_facts[@]}"; do
  archive_fact_args+=(--fact "${fact}")
done

archive_source_paths=(
  ".areaflow/status.json"
  "workflow/README.md"
  "workflow/versions/**/execution/**"
  "workflow/versions/**/execution/_shared/progress.json"
)

archive_forbidden_actions=(
  copy_artifact_bytes
  delete_artifact_bytes
  delete_historical_files
  move_historical_files
  rewrite_progress_json
  run_commands
  write_areamatrix_protected_paths
)

archive_binding_args=(
  --archive-scope areamatrix_historical_execution_reference_only
  --archive-reference-mode metadata_indexed_reference_only
  --archive-rollback-target execution_forwarding_read_only_shim
  --archive-fail-closed
)
for source_path in "${archive_source_paths[@]}"; do
  archive_binding_args+=(--archive-source-path "${source_path}")
done
for action in "${archive_forbidden_actions[@]}"; do
  archive_binding_args+=(--archive-forbidden-action "${action}")
done

shim_fact_args=()
for fact in "${shim_facts[@]}"; do
  shim_fact_args+=(--fact "${fact}")
done

shim_prerequisites=(
  archive_gate_passed
  execution_cutover_gate_passed
  protected_path_proof_recorded
)

shim_retired_surfaces=(
  legacy_task_loop_runner
  legacy_progress_json_writes
  legacy_logs_writes
  legacy_checkpoint_writes
)

shim_binding_args=(
  --shim-retirement-scope read_only_shim_retirement_after_execution_forwarding_v1
  --shim-rollback-target read_only_shim
  --shim-fail-closed
  --shim-reopen-requires-approval
)
for prerequisite in "${shim_prerequisites[@]}"; do
  shim_binding_args+=(--shim-prerequisite "${prerequisite}")
done
for surface in "${shim_retired_surfaces[@]}"; do
  shim_binding_args+=(--shim-retired-surface "${surface}")
done
execution_cutover_fact_args=()
for fact in "${execution_cutover_facts[@]}"; do
  execution_cutover_fact_args+=(--fact "${fact}")
done
execution_cutover_allowed_task_types=(
  read_only_verify
  doctor_readiness
  artifact_evidence
  status_projection_validation
  release_readiness_check
)
execution_cutover_forbidden_actions=(
  start_legacy_task_loop_runner
  write_legacy_progress_json
  write_legacy_logs
  write_legacy_checkpoint
  write_areamatrix_source
  write_areamatrix_execution_directory
  generated_retained_write
  repair_apply
  checkpoint_apply
  engine_execution
  secret_resolve
  network_api_integration
  publish_apply
  restore_apply
)
execution_cutover_binding_args=(
  --execution-cutover-scope execution_forwarding_v1_read_only_evidence_only
  --rollback-target read_only_shim
  --rollback-mode fail_closed_to_read_only_shim
  --fail-closed
  --reopen-requires-approval
)
for task_type in "${execution_cutover_allowed_task_types[@]}"; do
  execution_cutover_binding_args+=(--allowed-task-type "${task_type}")
done
for action in "${execution_cutover_forbidden_actions[@]}"; do
  execution_cutover_binding_args+=(--forbidden-action "${action}")
done
release_packaging_fact_args=()
for fact in "${release_packaging_facts[@]}"; do
  release_packaging_fact_args+=(--fact "${fact}")
done
backup_restore_fact_args=()
for fact in "${backup_restore_facts[@]}"; do
  backup_restore_fact_args+=(--fact "${fact}")
done
security_closure_fact_args=()
for fact in "${security_closure_facts[@]}"; do
  security_closure_fact_args+=(--fact "${fact}")
done

echo "smoke-completion-audit-full-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-completion-audit-full-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-completion-audit-full-proof: project import ${project_key}"
import_output="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output}" "imported ${project_key}"
assert_contains "${import_output}" "v1=2/2"

echo "smoke-completion-audit-full-proof: localize imported fixture artifacts for release readiness"
localized_artifacts_count="$(psql "${AREAFLOW_DATABASE_URL}" \
  -v "project_key=${project_key}" \
  -v "project_root=${project_root}" \
  -At <<'SQL'
WITH localized AS (
  UPDATE artifacts a
  SET
    storage_backend = 'local',
    uri = :'project_root' || '/' || a.source_path,
    metadata = COALESCE(a.metadata, '{}'::jsonb) || '{"fixture_localized_for_release_readiness":true}'::jsonb
  FROM projects p
  WHERE p.id = a.project_id
    AND p.project_key = :'project_key'
    AND a.storage_backend IN ('external_project', 'project_reference')
    AND a.source_path <> ''
  RETURNING a.id
)
SELECT COUNT(*) FROM localized;
SQL
)"
assert_positive_value "localized fixture artifacts" "${localized_artifacts_count}"

echo "smoke-completion-audit-full-proof: project status-projection-authorization ${project_key}"
authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
fixture_status="${project_root}/.areaflow/status.json"
assert_contains "${authorization_json}" '"status": "needs_approval"'
assert_contains "${authorization_json}" '"decision": "needs_explicit_approval"'
assert_contains "${authorization_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${authorization_json}" '"schema_uri": "schemas/status-projection.schema.json"'
assert_contains "${authorization_json}" '"apply_open": false'
assert_contains "${authorization_json}" '"approval_required": true'
assert_contains "${authorization_json}" '"would_write_project_file_after_approval": true'
assert_contains "${authorization_json}" '"requires_preimage_match": true'
assert_contains "${authorization_json}" '"requires_schema_validation": true'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-completion-audit-full-proof: authorization preview unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi

source_hash="$(json_get "${authorization_json}" "source_hash")"
validator_preflight="$(json_get "${authorization_json}" "validator_preflight")"
protected_path_check="git -C ${project_root} status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"

echo "smoke-completion-audit-full-proof: project status-projection-apply-packet ${project_key}"
apply_packet_ready_json="$(
  go run ./cmd/areaflow project status-projection-apply-packet "${project_key}" --json \
    --explicit-approval \
    --approval-actor "smoke-completion-audit-full-proof" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${apply_packet_ready_json}" '"status": "ready"'
assert_contains "${apply_packet_ready_json}" '"decision": "ready_for_apply_command"'
assert_contains "${apply_packet_ready_json}" '"blockers": []'
assert_contains "${apply_packet_ready_json}" '"apply_command_eligible": true'
assert_contains "${apply_packet_ready_json}" '"explicit_approval": true'
assert_contains "${apply_packet_ready_json}" '"approval_actor": "smoke-completion-audit-full-proof"'
assert_contains "${apply_packet_ready_json}" '"source_hash": "'"${source_hash}"'"'
assert_contains "${apply_packet_ready_json}" '"protected_path_fingerprint_sha256": "'
assert_contains "${apply_packet_ready_json}" '"command_request_created": false'
assert_contains "${apply_packet_ready_json}" '"status_projection_written": false'
assert_contains "${apply_packet_ready_json}" '"project_write_attempted": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-completion-audit-full-proof: apply packet preview unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi
protected_path_fingerprint_sha256="$(json_get "${apply_packet_ready_json}" "packet.protected_path_fingerprint_sha256")"

echo "smoke-completion-audit-full-proof: project status-projection-apply-gate ${project_key}"
apply_gate_ready_json="$(
  go run ./cmd/areaflow project status-projection-apply-gate "${project_key}" --json \
    --expected-before-exists false \
    --expected-before-size 0 \
    --source-hash "${source_hash}" \
    --schema-uri "schemas/status-projection.schema.json" \
    --validator-preflight "${validator_preflight}" \
    --protected-path-check "${protected_path_check}" \
    --protected-path-fingerprint-sha256 "${protected_path_fingerprint_sha256}" \
    --rollback-action "delete .areaflow/status.json if apply created it" \
    --accept-preimage-schema "missing" \
    --explicit-approval \
    --approval-actor "smoke-completion-audit-full-proof" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${apply_gate_ready_json}" '"status": "pass"'
assert_contains "${apply_gate_ready_json}" '"decision": "go"'
assert_contains "${apply_gate_ready_json}" '"apply_command_eligible": true'
assert_contains "${apply_gate_ready_json}" '"approval_status": "approved"'
assert_contains "${apply_gate_ready_json}" '"command_request_created": false'
assert_contains "${apply_gate_ready_json}" '"status_projection_written": false'
assert_contains "${apply_gate_ready_json}" '"project_write_attempted": false'
assert_contains "${apply_gate_ready_json}" '"execution_write_attempted": false'
assert_contains "${apply_gate_ready_json}" '"engine_call_attempted": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-completion-audit-full-proof: apply gate unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi

echo "smoke-completion-audit-full-proof: project status-projection-apply ${project_key}"
status_projection_apply_output="$(
  go run ./cmd/areaflow project status-projection-apply "${project_key}" \
    --expected-before-exists false \
    --expected-before-size 0 \
    --source-hash "${source_hash}" \
    --schema-uri "schemas/status-projection.schema.json" \
    --validator-preflight "${validator_preflight}" \
    --protected-path-check "${protected_path_check}" \
    --protected-path-fingerprint-sha256 "${protected_path_fingerprint_sha256}" \
    --rollback-action "delete .areaflow/status.json if apply created it" \
    --accept-preimage-schema "missing" \
    --explicit-approval \
    --approval-actor "smoke-completion-audit-full-proof" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${status_projection_apply_output}" "${fixture_status}"
assert_contains "${status_projection_apply_output}" "apply_gate: status=pass decision=go"
assert_contains "${status_projection_apply_output}" "post_write_verified=true"
assert_contains "${status_projection_apply_output}" "protected_paths_verified=true"
assert_contains "${status_projection_apply_output}" "stable_projection_validated=true"
assert_contains "${status_projection_apply_output}" "root_contained=true"
if [[ ! -f "${fixture_status}" ]]; then
  echo "smoke-completion-audit-full-proof: expected fixture status export at ${fixture_status}" >&2
  exit 1
fi

echo "smoke-completion-audit-full-proof: validate status projection schema"
python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json "${fixture_status}"

echo "smoke-completion-audit-full-proof: project status-projections --json"
status_projections_json="$(go run ./cmd/areaflow project status-projections "${project_key}" --json)"
assert_contains "${status_projections_json}" '"target_kind": "project_status_json"'
assert_contains "${status_projections_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${status_projections_json}" '"write_state": "written"'
assert_contains "${status_projections_json}" '"source_hash": "'"${source_hash}"'"'
assert_contains "${status_projections_json}" '"command_type": "project.status_projection.apply"'

echo "smoke-completion-audit-full-proof: seed release final gate fixture inputs"
local_release_artifact_sha="$(shasum -a 256 "${local_release_artifact}" | awk '{print $1}')"
local_release_artifact_size="$(wc -c <"${local_release_artifact}" | tr -d ' ')"
psql "${AREAFLOW_DATABASE_URL}" \
  -v "project_key=${project_key}" \
  -v "artifact_uri=${local_release_artifact}" \
  -v "artifact_sha=${local_release_artifact_sha}" \
  -v "artifact_size=${local_release_artifact_size}" >/dev/null <<'SQL'
WITH project_scope AS (
  SELECT id
  FROM projects
  WHERE project_key = :'project_key'
),
artifact_seed AS (
  INSERT INTO artifacts (
    project_id, artifact_type, storage_backend, uri, source_path, sha256, size_bytes, content_type, metadata
  )
  SELECT
    id,
    'release_readiness_fixture',
    'local',
    :'artifact_uri',
    'completion-audit-release-readiness.json',
    :'artifact_sha',
    :'artifact_size'::bigint,
    'application/json',
    '{"fixture_only":true,"purpose":"release_final_gate_readiness"}'::jsonb
  FROM project_scope
  RETURNING id
),
audit_seed(action, capability, resource_type, resource, decision, reason) AS (
  VALUES
    ('project.upsert', 'project_config', 'project', :'project_key', 'allowed', 'fixture project registration evidence'),
    ('status.export', 'write_status', 'path', '.areaflow/status.json', 'allowed', 'fixture status export audit coverage evidence'),
    ('workflow.version.create', 'write_workflow', 'workflow_version', 'fixture-v1', 'allowed', 'fixture workflow version audit coverage evidence'),
    ('workflow.stage_skeleton.create', 'write_workflow', 'workflow_version', 'fixture-v1', 'allowed', 'fixture stage skeleton audit coverage evidence'),
    ('workflow.item.mark_ready', 'write_workflow', 'workflow_item', 'fixture-item', 'allowed', 'fixture item ready audit coverage evidence'),
    ('workflow.approval.record', 'approval', 'workflow_version', 'fixture-v1', 'approved', 'fixture approval audit coverage evidence'),
    ('runner.preview', 'execute_runner', 'workflow_version', 'fixture-v1', 'allowed', 'fixture runner preview audit coverage evidence'),
    ('worker.register', 'manage_workers', 'worker', 'fixture-worker', 'allowed', 'fixture worker register audit coverage evidence'),
    ('worker.run_once', 'execute_worker', 'worker', 'fixture-worker', 'denied', 'fixture worker denial audit coverage evidence'),
    ('lease.acquire', 'manage_workers', 'lease', 'fixture-lease', 'allowed', 'fixture lease acquire audit coverage evidence'),
    ('lease.release', 'manage_workers', 'lease', 'fixture-lease', 'allowed', 'fixture lease release audit coverage evidence'),
    ('lease.recover', 'manage_workers', 'lease', 'fixture-lease', 'allowed', 'fixture lease recover audit coverage evidence'),
    ('command.execute', 'run_commands', 'command', 'fixture-command', 'denied', 'fixture command execution audit coverage evidence'),
    ('secret.resolve', 'use_secrets', 'secret_ref', 'fixture-secret', 'denied', 'fixture secret resolve audit coverage evidence'),
    ('permission.change', 'permission', 'project', :'project_key', 'denied', 'fixture permission change audit coverage evidence')
)
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
SELECT
  project_scope.id,
  audit_seed.action,
  audit_seed.capability,
  audit_seed.resource_type,
  audit_seed.resource,
  audit_seed.decision,
  audit_seed.reason,
  '{"fixture_only":true,"commands_run":false,"writes_real_project":false}'::jsonb
FROM project_scope
CROSS JOIN audit_seed;
SQL

echo "smoke-completion-audit-full-proof: release evidence bundle is ready"
release_final_gate_json="$(go run ./cmd/areaflow release final-gate --json)"
assert_contains "${release_final_gate_json}" '"status": "pass"'
assert_contains "${release_final_gate_json}" '"readiness_status": "ready"'
assert_contains "${release_final_gate_json}" '"acceptance_gate_status": "pass"'
release_evidence_json="$(go run ./cmd/areaflow release evidence-bundle --json)"
assert_contains "${release_evidence_json}" '"status": "ready"'
assert_contains "${release_evidence_json}" '"mode": "read_only_release_evidence_bundle"'
assert_contains "${release_evidence_json}" '"final_gate_status": "pass"'
assert_contains "${release_evidence_json}" '"backup_status": "ready"'

echo "smoke-completion-audit-full-proof: collect E6 output binding"
backup_manifest_json="$(go run ./cmd/areaflow backup manifest --project "${project_key}" --json)"
restore_plan_json="$(go run ./cmd/areaflow backup restore-plan --project "${project_key}" --json)"
artifact_integrity_json="$(go run ./cmd/areaflow artifact integrity "${project_key}" --json)"
archive_preview_json="$(go run ./cmd/areaflow artifact archive-preview "${project_key}" \
  --idempotency-key "completion-audit-full:archive-preview:${project_key}" \
  --reason "collect full completion audit backup restore binding" \
  --json)"

backup_manifest_hash="$(json_get "${backup_manifest_json}" "manifest_hash")"
backup_restore_binding_args=(
  --backup-manifest-hash "${backup_manifest_hash}"
  --backup-manifest-status "$(json_get "${backup_manifest_json}" "status")"
  --backup-manifest-project-count "$(json_get "${backup_manifest_json}" "len:projects")"
  --backup-manifest-table-count "$(json_get "${backup_manifest_json}" "len:table_counts")"
  --restore-plan-status "$(json_get "${restore_plan_json}" "status")"
  --restore-plan-scope "$(json_get "${restore_plan_json}" "scope")"
  --restore-plan-project-key "$(json_get "${restore_plan_json}" "project_key")"
  --restore-plan-manifest-hash "$(json_get "${restore_plan_json}" "manifest_hash")"
  --restore-plan-item-count "$(json_get "${restore_plan_json}" "len:items")"
  --artifact-integrity-status "$(json_get "${artifact_integrity_json}" "status")"
  --artifact-integrity-checked-count "$(json_get "${artifact_integrity_json}" "checked_artifacts")"
  --artifact-integrity-failed-count "$(json_get "${artifact_integrity_json}" "failed_artifacts")"
  --artifact-archive-preview-status "$(json_get "${archive_preview_json}" "status")"
  --artifact-archive-preview-total-artifacts "$(json_get "${archive_preview_json}" "summary.total_artifacts")"
  --artifact-archive-preview-external-refs "$(json_get "${archive_preview_json}" "summary.external_refs")"
  --artifact-archive-preview-needs-policy "$(json_get "${archive_preview_json}" "summary.needs_policy")"
  --artifact-archive-preview-project-write-attempted "$(json_get "${archive_preview_json}" "project_write_attempted")"
  --artifact-archive-preview-storage-write-attempted "$(json_get "${archive_preview_json}" "storage_write_attempted")"
  --artifact-archive-preview-delete-attempted "$(json_get "${archive_preview_json}" "artifact_delete_attempted")"
)

recorded_json="$(go run ./cmd/areaflow completion source-alignment-proof record "${project_key}" \
  --status complete "${source_alignment_fact_args[@]}" \
  --summary "full completion audit source alignment proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#source-alignment" \
  --idempotency-key "completion-audit-full:source-alignment:${project_key}" \
  --reason "record full completion audit source alignment proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'
assert_contains "${recorded_json}" '"source_alignment_binding_status": "pass"'
assert_contains "${recorded_json}" '"source_alignment_binding_blockers": []'
source_alignment_source_set_hash_value="$(json_get "${recorded_json}" "source_alignment_source_set_hash")"

recorded_json="$(go run ./cmd/areaflow completion task-matrix-proof record "${project_key}" \
  --status complete "${task_matrix_fact_args[@]}" \
  "${task_matrix_binding_args[@]}" \
  --summary "full completion audit task matrix proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#task-matrix" \
  --idempotency-key "completion-audit-full:task-matrix:${project_key}" \
  --reason "record full completion audit task matrix proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'
assert_contains "${recorded_json}" '"task_matrix_binding_status": "pass"'
assert_contains "${recorded_json}" '"task_matrix_source_set_hash": "'"${task_matrix_source_set_hash_value}"'"'

recorded_json="$(go run ./cmd/areaflow completion validation-proof record "${project_key}" \
  --status complete "${validation_fact_args[@]}" \
  --summary "full completion audit validation proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#validation" \
  "${validation_command_args[@]}" \
  --validation-result-hash "${validation_result_hash}" \
  --validation-started-at "${validation_started_at}" \
  --validation-finished-at "${validation_finished_at}" \
  --validation-scope "${validation_scope}" \
  --idempotency-key "completion-audit-full:validation:${project_key}" \
  --reason "record full completion audit validation proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'
assert_contains "${recorded_json}" '"validation_evidence_binding_status": "pass"'

archive_recorded_json="$(go run ./cmd/areaflow completion archive-proof record "${project_key}" \
  --status complete "${archive_fact_args[@]}" \
  "${archive_binding_args[@]}" \
  --summary "full completion audit archive proof" \
  --evidence-uri "${archive_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-audit-full:archive:${project_key}" \
  --reason "record full completion audit archive proof" \
  --json)"
assert_contains "${archive_recorded_json}" '"proof_status": "complete"'
assert_contains "${archive_recorded_json}" '"archive_scope_binding_status": "pass"'
assert_contains "${archive_recorded_json}" '"review_metadata_status": "approved"'
archive_event_id="$(json_get "${archive_recorded_json}" "event_id")"
archive_audit_event_id="$(json_get "${archive_recorded_json}" "audit_event_id")"
archive_scope_binding_hash="$(json_get "${archive_recorded_json}" "archive_scope_binding_hash")"
assert_positive_value "archive event_id" "${archive_event_id}"
assert_positive_value "archive audit_event_id" "${archive_audit_event_id}"
assert_sha256_value "archive_scope_binding_hash" "${archive_scope_binding_hash}"

shim_recorded_json="$(go run ./cmd/areaflow completion shim-retirement-proof record "${project_key}" \
  --status complete "${shim_fact_args[@]}" \
  "${shim_binding_args[@]}" \
  --summary "full completion audit shim retirement proof" \
  --evidence-uri "${shim_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-audit-full:shim-retirement:${project_key}" \
  --reason "record full completion audit shim retirement proof" \
  --json)"
assert_contains "${shim_recorded_json}" '"proof_status": "complete"'
assert_contains "${shim_recorded_json}" '"shim_retirement_scope_binding_status": "pass"'
assert_contains "${shim_recorded_json}" '"review_metadata_status": "approved"'
shim_event_id="$(json_get "${shim_recorded_json}" "event_id")"
shim_audit_event_id="$(json_get "${shim_recorded_json}" "audit_event_id")"
shim_retirement_scope_binding_hash="$(json_get "${shim_recorded_json}" "shim_retirement_scope_binding_hash")"
assert_positive_value "shim retirement event_id" "${shim_event_id}"
assert_positive_value "shim retirement audit_event_id" "${shim_audit_event_id}"
assert_sha256_value "shim_retirement_scope_binding_hash" "${shim_retirement_scope_binding_hash}"

recorded_json="$(go run ./cmd/areaflow completion execution-cutover-proof record "${project_key}" \
  --status complete "${execution_cutover_fact_args[@]}" \
  "${execution_cutover_binding_args[@]}" \
  --summary "full completion audit execution cutover proof" \
  --evidence-uri "${execution_cutover_evidence_uri}" \
  "${review_flags[@]}" \
  --idempotency-key "completion-audit-full:execution-cutover:${project_key}" \
  --reason "record full completion audit execution cutover proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'
assert_contains "${recorded_json}" '"execution_cutover_scope_binding_status": "pass"'
assert_contains "${recorded_json}" '"review_metadata_status": "approved"'
execution_cutover_event_id="$(json_get "${recorded_json}" "event_id")"
execution_cutover_audit_event_id="$(json_get "${recorded_json}" "audit_event_id")"
execution_cutover_scope_binding_hash="$(json_get "${recorded_json}" "execution_cutover_scope_binding_hash")"
assert_positive_value "execution cutover event_id" "${execution_cutover_event_id}"
assert_positive_value "execution cutover audit_event_id" "${execution_cutover_audit_event_id}"
assert_sha256_value "execution_cutover_scope_binding_hash" "${execution_cutover_scope_binding_hash}"

recorded_json="$(go run ./cmd/areaflow completion release-packaging-proof record "${project_key}" \
  --status complete "${release_packaging_fact_args[@]}" \
  --summary "full completion audit release packaging proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#release-packaging" \
  --idempotency-key "completion-audit-full:release-packaging:${project_key}" \
  --reason "record full completion audit release packaging proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'

recorded_json="$(go run ./cmd/areaflow completion backup-restore-proof record "${project_key}" \
  --status complete "${backup_restore_fact_args[@]}" \
  --summary "full completion audit backup restore proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#backup-restore" \
  "${backup_restore_binding_args[@]}" \
  --idempotency-key "completion-audit-full:backup-restore:${project_key}" \
  --reason "record full completion audit backup restore proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'
assert_contains "${recorded_json}" '"backup_restore_evidence_binding_status": "pass"'

recorded_json="$(go run ./cmd/areaflow completion security-closure-proof record "${project_key}" \
  --status complete "${security_closure_fact_args[@]}" \
  --summary "full completion audit security closure proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#security-closure" \
  --idempotency-key "completion-audit-full:security-closure:${project_key}" \
  --reason "record full completion audit security closure proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "complete"'
assert_contains "${recorded_json}" '"security_closure_binding_status": "pass"'
assert_contains "${recorded_json}" '"security_closure_binding_blockers": []'
assert_contains "${recorded_json}" '"permission_doctor_status": "pass"'
assert_contains "${recorded_json}" '"audit_coverage_status": "pass"'

recorded_json="$(go run ./cmd/areaflow ops smoke-proof record "${project_key}" \
  --key v1_stable_fixture_smoke \
  --status pass \
  --summary "full completion audit operations smoke proof" \
  --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#operations" \
  --idempotency-key "completion-audit-full:operations:${project_key}" \
  --reason "record full completion audit operations proof" \
  --json)"
assert_contains "${recorded_json}" '"evidence_status": "pass"'
assert_contains "${recorded_json}" '"record_command_runs_smoke": false'

recorded_json="$(go run ./cmd/areaflow completion protected-path-proof record "${project_key}" \
  --status clean \
  --summary "isolated completion audit fixture protected path proof recorded without real AreaMatrix fingerprint check" \
  --evidence-uri "fixture:completion-audit-full-protected-path-proof" \
  --idempotency-key "completion-audit-full:protected-path:${project_key}" \
  --reason "record full completion audit protected path proof" \
  --json)"
assert_contains "${recorded_json}" '"proof_status": "clean"'
assert_contains "${recorded_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${recorded_json}" '"git_status_output_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"'
assert_contains "${recorded_json}" '"git_status_output_empty": true'
assert_contains "${recorded_json}" '"protected_path_set_hash": "'
assert_contains "${recorded_json}" '"protected_path_set_count": 7'
assert_contains "${recorded_json}" '"protected_path_proof_binding_status": "pass"'
assert_contains "${recorded_json}" '"protected_path_proof_binding_blockers": []'

echo "smoke-completion-audit-full-proof: completion audit should stay blocked by fixture AreaMatrix identity"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_line "${completion_json}" '  "status": "blocked",'
assert_contains "${completion_json}" '"task_matrix_status": "complete"'
assert_contains "${completion_json}" '"implementation_gap_status": "complete"'
assert_contains "${completion_json}" '"task_matrix_binding_status": "pass"'
assert_contains "${completion_json}" '"task_matrix_binding_blockers": []'
assert_contains "${completion_json}" '"task_matrix_current_binding_bound": true'
assert_contains "${completion_json}" '"task_matrix_source_set_hash": "'"${task_matrix_source_set_hash_value}"'"'
assert_contains "${completion_json}" '"planned_v1_required_task_count": 0'
assert_contains "${completion_json}" '"missing_evidence_v1_required_task_count": 0'
assert_contains "${completion_json}" '"blocked_v1_required_task_count": 0'
assert_contains "${completion_json}" '"source_alignment_gate_passed": true'
assert_contains "${completion_json}" '"source_alignment_binding_status": "pass"'
assert_contains "${completion_json}" '"source_alignment_current_binding_bound": true'
assert_contains "${completion_json}" '"source_alignment_source_set_hash": "'"${source_alignment_source_set_hash_value}"'"'
assert_contains "${completion_json}" '"source_alignment_missing_source_count": 0'
assert_contains "${completion_json}" '"source_alignment_unreadable_source_count": 0'
assert_contains "${completion_json}" '"task_matrix_gate_passed": true'
assert_contains "${completion_json}" '"validation_gate_passed": true'
assert_contains "${completion_json}" '"latest_archive_proof_event_id": '"${archive_event_id}"
assert_contains "${completion_json}" '"latest_shim_retirement_proof_event_id": '"${shim_event_id}"
assert_contains "${completion_json}" '"latest_execution_cutover_proof_event_id": '"${execution_cutover_event_id}"
assert_contains "${completion_json}" '"latest_archive_proof_evidence_uri": "'"${archive_evidence_uri}"'"'
assert_contains "${completion_json}" '"latest_shim_retirement_proof_evidence_uri": "'"${shim_evidence_uri}"'"'
assert_contains "${completion_json}" '"latest_execution_cutover_proof_evidence_uri": "'"${execution_cutover_evidence_uri}"'"'
assert_contains "${completion_json}" '"archive_proof_review_metadata_status": "approved"'
assert_contains "${completion_json}" '"shim_retirement_proof_review_metadata_status": "approved"'
assert_contains "${completion_json}" '"execution_cutover_proof_review_metadata_status": "approved"'
assert_contains "${completion_json}" '"archive_scope_binding_status": "pass"'
assert_contains "${completion_json}" '"archive_scope_binding_blockers": []'
assert_contains "${completion_json}" '"archive_scope_binding_hash": "'"${archive_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"current_archive_scope_binding_hash": "'"${archive_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"archive_scope_current_binding_bound": true'
assert_contains "${completion_json}" '"archive_scope": "areamatrix_historical_execution_reference_only"'
assert_contains "${completion_json}" '"archive_reference_mode": "metadata_indexed_reference_only"'
assert_contains "${completion_json}" '"archive_rollback_target": "execution_forwarding_read_only_shim"'
assert_contains "${completion_json}" '"archive_fail_closed": true'
assert_contains "${completion_json}" '"shim_retirement_scope_binding_status": "pass"'
assert_contains "${completion_json}" '"shim_retirement_scope_binding_blockers": []'
assert_contains "${completion_json}" '"shim_retirement_scope_binding_hash": "'"${shim_retirement_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"current_shim_retirement_scope_binding_hash": "'"${shim_retirement_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"shim_retirement_scope_current_binding_bound": true'
assert_contains "${completion_json}" '"shim_retirement_scope": "read_only_shim_retirement_after_execution_forwarding_v1"'
assert_contains "${completion_json}" '"shim_rollback_target": "read_only_shim"'
assert_contains "${completion_json}" '"shim_fail_closed": true'
assert_contains "${completion_json}" '"shim_reopen_requires_approval": true'
assert_contains "${completion_json}" '"execution_cutover_scope_binding_status": "pass"'
assert_contains "${completion_json}" '"execution_cutover_scope_binding_blockers": []'
assert_contains "${completion_json}" '"execution_cutover_scope_binding_hash": "'"${execution_cutover_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"current_execution_cutover_scope_binding_hash": "'"${execution_cutover_scope_binding_hash}"'"'
assert_contains "${completion_json}" '"execution_cutover_scope_current_binding_bound": true'
assert_contains "${completion_json}" '"release_packaging_gate_passed": true'
assert_contains "${completion_json}" '"backup_restore_gate_passed": true'
assert_contains "${completion_json}" '"operations_status": "ready"'
assert_contains "${completion_json}" '"security_closure_gate_passed": true'
assert_contains "${completion_json}" '"protected_path_proof_status": "complete"'
assert_contains "${completion_json}" '"protected_path_proof_binding_status": "pass"'
assert_contains "${completion_json}" '"protected_path_proof_binding_blockers": []'
assert_contains "${completion_json}" '"git_status_output_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"'
assert_contains "${completion_json}" '"git_status_output_empty": true'
assert_contains "${completion_json}" '"protected_path_set_hash": "'
assert_contains "${completion_json}" '"protected_path_set_count": 7'
assert_contains "${completion_json}" '"real_100_status": "blocked"'
assert_contains "${completion_json}" '"readiness_scope": "completion_audit_evidence_only"'
assert_contains "${completion_json}" '"claim_scope": "completion_audit_evidence_only"'
assert_contains "${completion_json}" '"not_real_100": true'
assert_contains "${completion_json}" '"evidence_only": true'
assert_contains "${completion_json}" '"status_alone_is_not_completion": true'
assert_contains "${completion_json}" '"release_candidate_decision": "requires_release_candidate_snapshot"'
assert_contains "${completion_json}" '"real_100_blockers": ['
assert_contains "${completion_json}" '"release_candidate_snapshot_not_ready"'
assert_not_contains "${completion_json}" '"package_a_status_projection_not_applied"'
assert_not_contains "${completion_json}" '"real_areamatrix_read_only_shim_not_landed"'
assert_contains "${completion_json}" '"project_root_not_real_areamatrix"'
assert_contains "${completion_json}" '"real_areamatrix_execution_cutover_not_proven"'
assert_contains "${completion_json}" '"real_areamatrix_archive_not_proven"'
assert_contains "${completion_json}" '"real_areamatrix_shim_retirement_not_proven"'
assert_not_contains "${completion_json}" '"source_alignment_proof_missing"'
assert_not_contains "${completion_json}" '"task_matrix_proof_missing"'
assert_not_contains "${completion_json}" '"fresh_validation_proof_missing"'
assert_contains "${completion_json}" '"execution_cutover_not_complete"'
assert_not_contains "${completion_json}" '"archive_scope_binding_incomplete"'
assert_not_contains "${completion_json}" '"shim_retirement_scope_binding_incomplete"'
assert_not_contains "${completion_json}" '"release_final_gate_not_passed"'
assert_not_contains "${completion_json}" '"restore_dry_run_needs_attention"'
assert_not_contains "${completion_json}" '"metadata_only_history_not_closed"'
assert_not_contains "${completion_json}" '"fresh_local_ops_smoke_missing"'
assert_not_contains "${completion_json}" '"full_migration_ledger_missing"'
assert_not_contains "${completion_json}" '"project_isolation_smoke_missing"'
assert_not_contains "${completion_json}" '"audit_gap_closure_missing"'
assert_not_contains "${completion_json}" '"protected_path_proof_missing"'

echo "smoke-completion-audit-full-proof: release candidate snapshot rejects incomplete audit before fixture identity can be sealed"
assert_fails_contains \
  "completion audit snapshot requires current audit status complete" \
  go run ./cmd/areaflow completion audit-snapshot record "${project_key}" \
    --release-candidate "v1.0-rc1" \
	    --evidence-class "release_candidate" \
	    --summary "reviewed release candidate evidence bundle" \
	    --evidence-uri "docs/history/v1.0/evidence/real-release-candidate-evidence.md#release-candidate-review" \
	    --review-decision "approved" \
	    --reviewed-by "release-owner" \
	    --reviewed-at "2026-07-04T12:00:00Z" \
	    --idempotency-key "completion-audit-full:release-candidate-reject:${project_key}" \
	    --reason "prove fixture identity cannot be sealed as release candidate" \
	    --json

echo "smoke-completion-audit-full-proof: fixture snapshot record rejects incomplete audit"
assert_fails_contains \
  "completion audit snapshot requires current audit status complete" \
  go run ./cmd/areaflow completion audit-snapshot record "${project_key}" \
    --release-candidate "v1.0-fixture" \
    --evidence-class "fixture" \
    --summary "full completion audit fixture snapshot" \
    --evidence-uri "scripts/smoke-completion-audit-full-proof.sh#completion-audit" \
    --idempotency-key "completion-audit-full:snapshot:${project_key}" \
    --reason "prove fixture identity cannot be sealed as full completion" \
    --json

snapshot_readiness_json="$(go run ./cmd/areaflow completion audit-snapshot readiness "${project_key}" --json)"
assert_contains "${snapshot_readiness_json}" '"status": "blocked"'
assert_contains "${snapshot_readiness_json}" '"required_class": "release_candidate"'
assert_contains "${snapshot_readiness_json}" '"has_snapshot": false'
assert_contains "${snapshot_readiness_json}" '"completion_audit_snapshot_real_project_identity_missing"'
assert_contains "${snapshot_readiness_json}" '"real_project_identity_ready": false'
assert_contains "${snapshot_readiness_json}" '"project_root_not_real_areamatrix"'
assert_not_contains "${snapshot_readiness_json}" '"completion_audit_snapshot_release_candidate_present"'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-completion-audit-full-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-completion-audit-full-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-completion-audit-full-proof: pass ${project_key} fixture=${fixture_dir}"
