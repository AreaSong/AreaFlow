#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-fixture: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_FIXTURE_PROJECT_KEY:-areamatrix-fixture}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-fixture.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
real_areamatrix_status="/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-fixture: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-fixture: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-fixture: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-fixture: expected output not to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_contains_any() {
  local output="$1"
  shift

  local pattern
  for pattern in "$@"; do
    if grep -Fq -- "${pattern}" <<<"${output}"; then
      return
    fi
  done

  echo "smoke-fixture: expected output to contain one of: $*" >&2
  exit 1
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
if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
fi

mkdir -p \
  "${project_root}/docs" \
  "${project_root}/workflow/residuals" \
  "${project_root}/workflow/templates" \
  "${project_root}/workflow/versions/v1-mvp/residuals" \
  "${project_root}/workflow/versions/v1-mvp/execution/_shared" \
  "${project_root}/workflow/versions/v-template" \
  "${project_root}/tasks/active" \
  "${project_root}/tasks/done" \
  "${project_root}/tasks/backlog/prompts/sample-package" \
  "${project_root}/tasks/indexes" \
  "${artifact_root}"

for stage in discussion middle-layer changes plans drafts queue promotion projection closeout; do
  mkdir -p "${project_root}/workflow/versions/v1-mvp/${stage}"
  printf "# %s fixture\n" "${stage}" >"${project_root}/workflow/versions/v1-mvp/${stage}/README.md"
done

cat >"${project_root}/workflow/intake.md" <<'EOF'
# Fixture Intake

This fixture exists only for AreaFlow smoke tests.
EOF

cat >"${project_root}/docs/README.md" <<'EOF'
# Fixture Docs

Minimal source docs coverage for the AreaMatrix profile.
EOF

cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Fixture Templates

Minimal template coverage for the AreaMatrix profile.
EOF

cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items:
  - id: global-fixture-note
    status: reference-only
    type: fixture
    title: Global fixture residual
    source: workflow/residuals/residuals.yaml
    current_impact: proves global residual import
    executable_task: false
    promotion_required: false
    close_condition: fixture smoke passes
version_residuals:
  - version: v1-mvp
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    status: mixed-blocked
    summary: Fixture v1 metadata
EOF

cat >"${project_root}/workflow/versions/v1-mvp/residuals/residuals.yaml" <<'EOF'
version_status:
  technical_queue: complete
  formal_alpha: blocked
  fixture: true
items:
  - id: v1-fixture-residual
    status: blocked-decision
    type: fixture
    title: Version fixture residual
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    current_impact: proves version residual import
    executable_task: false
    promotion_required: true
    close_condition: fixture decision is recorded
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
# Fixture Version Template

Template-only workflow version placeholder.
EOF

cat >"${project_root}/tasks/indexes/residuals.md" <<'EOF'
# Fixture Residual Index
EOF

cat >"${project_root}/tasks/backlog/README.md" <<'EOF'
# Fixture Backlog
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Fixture
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

artifact_store:
  backend: local
  root: ${artifact_root}

permissions:
  capabilities:
    read_project: true
    write_status: true
    write_artifacts: true
    write_workflow: false
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
  allowed:
    - ./dev workflow doctor
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

git -C "${project_root}" init -q
git -C "${project_root}" config user.email "smoke-fixture@example.invalid"
git -C "${project_root}" config user.name "AreaFlow Smoke Fixture"
git -C "${project_root}" add .
git -C "${project_root}" commit -q -m "fixture baseline"

echo "smoke-fixture: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-fixture: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-fixture: project import ${project_key} #1"
import_output_1="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output_1}" "imported ${project_key}"
assert_contains "${import_output_1}" "v1=2/2"

echo "smoke-fixture: project import ${project_key} #2"
import_output_2="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output_2}" "imported ${project_key}"
assert_contains "${import_output_2}" "v1=2/2"

import_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.import'
  AND cr.completed_at IS NOT NULL
  AND cr.response ? 'run_id'
  AND cr.response ? 'status_snapshot';
SQL
)"
if [[ "${import_command_count}" -lt 2 ]]; then
  echo "smoke-fixture: expected two project.import command requests, got ${import_command_count}" >&2
  exit 1
fi

echo "smoke-fixture: project doctor --json"
doctor_json="$(go run ./cmd/areaflow project doctor "${project_key}" --json)"
assert_contains "${doctor_json}" '"name": "hash_drift"'
assert_contains "${doctor_json}" '"name": "project_config_drift"'
assert_contains "${doctor_json}" '"name": "stage_coverage"'
assert_contains "${doctor_json}" '"name": "native_workflow_doctor"'
assert_contains "${doctor_json}" '"status": "pass"'
assert_contains "${doctor_json}" '"status": "warn"'
assert_contains "${doctor_json}" '"native workflow doctor skipped by command permission gate"'
doctor_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.doctor.record'
  AND cr.completed_at IS NOT NULL
  AND cr.response ? 'event_id';
SQL
)"
if [[ "${doctor_command_count}" -lt 1 ]]; then
  echo "smoke-fixture: expected project.doctor.record command request, got ${doctor_command_count}" >&2
  exit 1
fi

echo "smoke-fixture: project status-projection-authorization ${project_key}"
authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
fixture_status="${project_root}/.areaflow/status.json"
assert_contains "${authorization_json}" '"status": "needs_approval"'
assert_contains "${authorization_json}" '"decision": "needs_explicit_approval"'
assert_contains "${authorization_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${authorization_json}" '"schema_uri": "schemas/status-projection.schema.json"'
assert_contains "${authorization_json}" '"validator_preflight": "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json '"${fixture_status}"'"'
assert_contains "${authorization_json}" '"schema_status": "missing"'
assert_contains "${authorization_json}" '"apply_open": false'
assert_contains "${authorization_json}" '"approval_required": true'
assert_contains "${authorization_json}" '"project_write_attempted": false'
assert_contains "${authorization_json}" '"execution_write_attempted": false'
assert_contains "${authorization_json}" '"engine_call_attempted": false'
assert_contains "${authorization_json}" '"would_write_project_file_after_approval": true'
assert_contains "${authorization_json}" '"requires_preimage_match": true'
assert_contains "${authorization_json}" '"requires_schema_validation": true'
assert_contains "${authorization_json}" '"protected_paths": ['
assert_contains "${authorization_json}" '".areaflow/status.json"'
assert_contains "${authorization_json}" '"areaflow project status-projections '"${project_key}"' --json"'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: authorization preview unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi

echo "smoke-fixture: project status-projection-apply-packet ${project_key} missing approval"
apply_packet_needs_approval_json="$(go run ./cmd/areaflow project status-projection-apply-packet "${project_key}" --json)"
assert_contains "${apply_packet_needs_approval_json}" '"status": "needs_approval"'
assert_contains "${apply_packet_needs_approval_json}" '"decision": "needs_explicit_approval"'
assert_contains "${apply_packet_needs_approval_json}" '"blockers": ['
assert_contains "${apply_packet_needs_approval_json}" '"explicit_status_projection_apply_approval_missing"'
assert_contains "${apply_packet_needs_approval_json}" '"approval_actor_missing"'
assert_contains "${apply_packet_needs_approval_json}" '"approval_reason_missing"'
assert_contains "${apply_packet_needs_approval_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${apply_packet_needs_approval_json}" '"source_hash": "'
assert_contains "${apply_packet_needs_approval_json}" '"protected_path_fingerprint_sha256": "'
assert_contains "${apply_packet_needs_approval_json}" '"expected_before_exists": false'
assert_contains "${apply_packet_needs_approval_json}" '"expected_before_size": 0'
assert_contains "${apply_packet_needs_approval_json}" '"accept_preimage_schema": "missing"'
assert_contains "${apply_packet_needs_approval_json}" '"apply_command_eligible": false'
assert_contains "${apply_packet_needs_approval_json}" '"command_request_created": false'
assert_contains "${apply_packet_needs_approval_json}" '"status_projection_written": false'
assert_contains "${apply_packet_needs_approval_json}" '"project_write_attempted": false'
assert_contains "${apply_packet_needs_approval_json}" '"execution_write_attempted": false'
assert_contains "${apply_packet_needs_approval_json}" '"engine_call_attempted": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: apply packet preview unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi
go_packet_fingerprint_sha256="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["protected_path_fingerprint_sha256"])' <<<"${apply_packet_needs_approval_json}")"

echo "smoke-fixture: package-a authorization packet fixture fingerprint matches Go packet"
set +e
package_a_fixture_json="$(AREAFLOW_PACKAGE_A_PROJECT_ROOT="${project_root}" bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
package_a_fixture_rc=$?
set -e
echo "${package_a_fixture_json}"
if [[ ${package_a_fixture_rc} -eq 0 ]]; then
  echo "smoke-fixture: expected fixture package-a packet to stay blocked without target preimage" >&2
  exit 1
fi
assert_contains "${package_a_fixture_json}" '"status": "blocked_needs_preflight_review"'
assert_contains "${package_a_fixture_json}" '"project_root": "'"${project_root}"'"'
assert_contains "${package_a_fixture_json}" '"target": ".areaflow/status.json"'
assert_contains "${package_a_fixture_json}" '"schema_state": "missing"'
assert_contains "${package_a_fixture_json}" '"protected_path_fingerprint_status": "captured"'
assert_contains "${package_a_fixture_json}" '"protected_path_fingerprint_sha256": "'"${go_packet_fingerprint_sha256}"'"'
assert_contains "${package_a_fixture_json}" '"modifies_areamatrix": false'
assert_contains "${package_a_fixture_json}" '"applies_status_projection": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: package-a fixture packet unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi

echo "smoke-fixture: project status-projection-apply-gate ${project_key} missing packet"
apply_gate_blocked_json="$(go run ./cmd/areaflow project status-projection-apply-gate "${project_key}" --json)"
assert_contains "${apply_gate_blocked_json}" '"status": "blocked"'
assert_contains "${apply_gate_blocked_json}" '"decision": "no_go"'
assert_contains "${apply_gate_blocked_json}" '"apply_command_eligible": false'
assert_contains "${apply_gate_blocked_json}" '"approval_status": "missing_or_incomplete"'
assert_contains "${apply_gate_blocked_json}" '"key": "explicit_approval"'
assert_contains "${apply_gate_blocked_json}" '"explicit_status_projection_apply_approval_missing"'
assert_contains "${apply_gate_blocked_json}" '"key": "source_snapshot_hash"'
assert_contains "${apply_gate_blocked_json}" '"command_request_created": false'
assert_contains "${apply_gate_blocked_json}" '"status_projection_written": false'
assert_contains "${apply_gate_blocked_json}" '"project_write_attempted": false'
assert_contains "${apply_gate_blocked_json}" '"execution_write_attempted": false'
assert_contains "${apply_gate_blocked_json}" '"engine_call_attempted": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: apply gate unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi

echo "smoke-fixture: project export-status ${project_key} legacy command fails closed without packet"
set +e
legacy_export_output="$(go run ./cmd/areaflow project export-status "${project_key}" 2>&1)"
legacy_export_rc=$?
set -e
echo "${legacy_export_output}"
if [[ ${legacy_export_rc} -eq 0 ]]; then
  echo "smoke-fixture: expected legacy export-status to fail closed without packet" >&2
  exit 1
fi
assert_contains "${legacy_export_output}" "status export denied for .areaflow/status.json"
assert_contains "${legacy_export_output}" "status_projection_apply_gate_blocked"
assert_contains "${legacy_export_output}" "source_snapshot_hash_missing_or_mismatch"
assert_contains "${legacy_export_output}" "explicit_status_projection_apply_approval_missing"
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: legacy export-status unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi
legacy_export_command_count="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT COUNT(*)
FROM command_requests cr
JOIN projects p ON p.id = cr.project_id
WHERE p.project_key = :'project_key'
  AND cr.command_type = 'project.status_projection.apply'
  AND cr.completed_at IS NOT NULL
  AND cr.response->>'status' = 'blocked'
  AND cr.response->>'decision' = 'denied'
  AND cr.response->>'apply_gate_status' = 'blocked'
  AND cr.response->>'apply_gate_decision' = 'no_go'
  AND cr.response->>'apply_command_eligible' = 'false'
  AND cr.response->>'project_write_attempted' = 'false'
  AND cr.response->>'execution_write_attempted' = 'false'
  AND cr.response->>'engine_call_attempted' = 'false';
SQL
)"
if [[ "${legacy_export_command_count}" -lt 1 ]]; then
  echo "smoke-fixture: expected denied legacy export command response with no project write, got ${legacy_export_command_count}" >&2
  exit 1
fi

source_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["source_hash"])' <<<"${authorization_json}")"
validator_preflight="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["validator_preflight"])' <<<"${authorization_json}")"
protected_path_check="git -C ${project_root} status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"

echo "smoke-fixture: project status-projection-apply-packet ${project_key} complete approval"
apply_packet_ready_json="$(
  go run ./cmd/areaflow project status-projection-apply-packet "${project_key}" --json \
    --explicit-approval \
    --approval-actor "smoke-fixture" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${apply_packet_ready_json}" '"status": "ready"'
assert_contains "${apply_packet_ready_json}" '"decision": "ready_for_apply_command"'
assert_contains "${apply_packet_ready_json}" '"blockers": []'
assert_contains "${apply_packet_ready_json}" '"apply_command_eligible": true'
assert_contains "${apply_packet_ready_json}" '"explicit_approval": true'
assert_contains "${apply_packet_ready_json}" '"approval_actor": "smoke-fixture"'
assert_contains "${apply_packet_ready_json}" '"source_hash": "'"${source_hash}"'"'
assert_contains "${apply_packet_ready_json}" '"protected_path_fingerprint_sha256": "'
assert_contains "${apply_packet_ready_json}" '"apply_command": ['
assert_contains "${apply_packet_ready_json}" '"status-projection-apply"'
assert_contains "${apply_packet_ready_json}" '"command_request_created": false'
assert_contains "${apply_packet_ready_json}" '"status_projection_written": false'
assert_contains "${apply_packet_ready_json}" '"project_write_attempted": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: apply packet preview unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi
protected_path_fingerprint_sha256="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["packet"]["protected_path_fingerprint_sha256"])' <<<"${apply_packet_ready_json}")"

echo "smoke-fixture: project status-projection-apply-gate ${project_key} complete packet"
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
    --approval-actor "smoke-fixture" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${apply_gate_ready_json}" '"status": "pass"'
assert_contains "${apply_gate_ready_json}" '"decision": "go"'
assert_contains "${apply_gate_ready_json}" '"apply_command_eligible": true'
assert_contains "${apply_gate_ready_json}" '"approval_status": "approved"'
assert_contains "${apply_gate_ready_json}" '"expected_before_exists"'
assert_contains "${apply_gate_ready_json}" '"source_snapshot_hash"'
assert_contains "${apply_gate_ready_json}" '"command_request_created": false'
assert_contains "${apply_gate_ready_json}" '"status_projection_written": false'
assert_contains "${apply_gate_ready_json}" '"project_write_attempted": false'
assert_contains "${apply_gate_ready_json}" '"execution_write_attempted": false'
assert_contains "${apply_gate_ready_json}" '"engine_call_attempted": false'
if [[ -f "${fixture_status}" ]]; then
  echo "smoke-fixture: apply gate unexpectedly wrote ${fixture_status}" >&2
  exit 1
fi

echo "smoke-fixture: project status-projection-apply ${project_key}"
export_output="$(
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
    --approval-actor "smoke-fixture" \
    --approval-reason "fixture status projection apply"
)"
assert_contains "${export_output}" "${fixture_status}"
assert_contains "${export_output}" "apply_gate: status=pass decision=go"
assert_contains "${export_output}" "write_safety:"
assert_contains "${export_output}" "preimage_captured=true"
assert_contains "${export_output}" "preimage_exists=false"
assert_contains "${export_output}" "post_write_verified=true"
assert_contains "${export_output}" "protected_paths_verified=true"
assert_contains "${export_output}" "expected_protected_path_hash=${protected_path_fingerprint_sha256}"
assert_contains "${export_output}" "root_contained=true"
assert_contains "${export_output}" "stable_projection_validated=true"
assert_contains "${export_output}" "atomic_replace_used=true"
assert_contains "${export_output}" "rollback_compensation_enabled=true"
if [[ ! -f "${fixture_status}" ]]; then
  echo "smoke-fixture: expected fixture status export at ${fixture_status}" >&2
  exit 1
fi
fixture_status_json="$(cat "${fixture_status}")"
assert_contains "${fixture_status_json}" '"schema_version": 1'
assert_contains "${fixture_status_json}" '"project_id": "'"${project_key}"'"'
assert_contains "${fixture_status_json}" '"project_name": "AreaMatrix Fixture"'
assert_contains "${fixture_status_json}" '"area_flow_url": "http://127.0.0.1:3847/projects/'"${project_key}"'"'
assert_contains "${fixture_status_json}" '"cutover_phase": "import_mirror"'
assert_contains "${fixture_status_json}" '"active_versions":'
assert_contains "${fixture_status_json}" '"display_label": "v1-mvp"'
assert_contains "${fixture_status_json}" '"rough_progress":'
assert_contains "${fixture_status_json}" '"source_snapshot_hash": "'
assert_contains "${fixture_status_json}" '"shim_lifecycle_state": "not_installed"'
assert_contains "${fixture_status_json}" '"blocked_commands":'
assert_not_contains "${fixture_status_json}" '"summary":'
assert_not_contains "${fixture_status_json}" '"generated_at":'
assert_not_contains "${fixture_status_json}" '"source_hash":'

echo "smoke-fixture: validate status projection schema"
python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json "${fixture_status}"

echo "smoke-fixture: project status-projections --json"
status_projections_json="$(go run ./cmd/areaflow project status-projections "${project_key}" --json)"
assert_contains "${status_projections_json}" '"target_kind": "project_status_json"'
assert_contains "${status_projections_json}" '"target_uri": ".areaflow/status.json"'
assert_contains "${status_projections_json}" '"summary_state": "mirroring"'
assert_contains "${status_projections_json}" '"write_state": "written"'
assert_contains "${status_projections_json}" '"source_hash": "'
assert_contains "${status_projections_json}" '"command_type": "project.status_projection.apply"'

projection_db_counts="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT
  (SELECT COUNT(*)
   FROM project_status_snapshots s
   JOIN projects p ON p.id = s.project_id
   WHERE p.project_key = :'project_key'
     AND s.snapshot_kind = 'mirror_export'),
  (SELECT COUNT(*)
   FROM status_projections sp
   JOIN projects p ON p.id = sp.project_id
   WHERE p.project_key = :'project_key'
     AND sp.target_kind = 'project_status_json'
     AND sp.target_uri = '.areaflow/status.json'
     AND sp.write_state = 'written'),
  (SELECT COUNT(*)
   FROM command_requests cr
   JOIN projects p ON p.id = cr.project_id
   WHERE p.project_key = :'project_key'
     AND cr.command_type = 'project.status_projection.apply'
     AND cr.completed_at IS NOT NULL
     AND cr.response->>'apply_gate_status' = 'pass'
     AND cr.response->>'apply_gate_decision' = 'go'
     AND cr.response->>'apply_command_eligible' = 'true'
     AND cr.response->>'project_write_attempted' = 'true'
     AND cr.response->>'execution_write_attempted' = 'false'
     AND cr.response->>'engine_call_attempted' = 'false');
SQL
)"
snapshot_count="${projection_db_counts%%|*}"
rest_counts="${projection_db_counts#*|}"
projection_count="${rest_counts%%|*}"
command_request_count="${projection_db_counts##*|}"
if [[ "${snapshot_count}" -lt 1 || "${projection_count}" -lt 1 || "${command_request_count}" -lt 1 ]]; then
  echo "smoke-fixture: expected mirror_export snapshot, status projection and gate-passed command request rows, got ${projection_db_counts}" >&2
  exit 1
fi

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-fixture: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi
else
  echo "smoke-fixture: skipped real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

echo "smoke-fixture: project summary --json"
summary_json="$(go run ./cmd/areaflow project summary "${project_key}" --json)"
assert_contains "${summary_json}" '"key": "'"${project_key}"'"'
assert_contains "${summary_json}" '"config_hash": "'
assert_contains "${summary_json}" '"history_ready_for_diff": true'
assert_contains "${summary_json}" '"drift_status": "pass"'
assert_contains "${summary_json}" '"stage_coverage_status": "pass"'
assert_contains "${summary_json}" '"native_doctor_status": "warn"'

echo "smoke-fixture: project readiness --json"
readiness_json="$(go run ./cmd/areaflow project readiness "${project_key}" --json)"
assert_contains "${readiness_json}" '"status": "warn"'
assert_contains "${readiness_json}" '"key": "status_mirror"'
assert_contains "${readiness_json}" '"key": "native_workflow_doctor"'
assert_contains "${readiness_json}" '"native_doctor_status": "warn"'

echo "smoke-fixture: project import-diff --json"
diff_json="$(go run ./cmd/areaflow project import-diff "${project_key}" --json)"
assert_contains "${diff_json}" '"status": "unchanged"'
assert_contains "${diff_json}" '"has_previous": true'
assert_contains "${diff_json}" '"source_changed": false'

echo "smoke-fixture: project verify-bundle --json"
bundle_json="$(go run ./cmd/areaflow project verify-bundle "${project_key}" --json)"
assert_contains "${bundle_json}" '"status": "warn"'
assert_contains "${bundle_json}" '"name": "v0.2-shadow-doctor"'
assert_contains "${bundle_json}" '"status": "pass"'
assert_contains "${bundle_json}" '"accepted_warnings": ['
assert_contains "${bundle_json}" 'native workflow doctor skipped or warned by permission gate'
assert_contains_any "${bundle_json}" '"blockers": null' '"blockers": []'
assert_contains "${bundle_json}" '"status": "unchanged"'

case "${fixture_status}" in
  "${project_root}"/.areaflow/status.json) ;;
  *)
    echo "smoke-fixture: status export escaped fixture root: ${fixture_status}" >&2
    exit 1
    ;;
esac

echo "smoke-fixture: pass ${project_key} fixture=${fixture_dir}"
