#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-release-packaging-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_RELEASE_PACKAGING_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-release-packaging.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/areamatrix-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"
real_areamatrix_status="/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
real_areamatrix_readme="/Users/as/Ai-Project/project/AreaMatrix/workflow/README.md"
check_real_areamatrix="${AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX:-0}"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-release-packaging-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-release-packaging-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-release-packaging-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-release-packaging-proof: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_completion_real_100_guardrail() {
  local output="$1"

  assert_contains "${output}" '"readiness_scope": "completion_audit_evidence_only"'
  assert_contains "${output}" '"claim_scope": "completion_audit_evidence_only"'
  assert_contains "${output}" '"not_real_100": true'
  assert_contains "${output}" '"evidence_only": true'
  assert_contains "${output}" '"status_alone_is_not_completion": true'
  assert_contains "${output}" '"release_candidate_decision": "requires_release_candidate_snapshot"'
  assert_contains "${output}" '"real_100_status": "blocked"'
  assert_contains "${output}" '"real_100_blockers": ['
  assert_contains "${output}" '"package_a_status_projection_not_applied"'
  assert_contains "${output}" '"real_areamatrix_execution_cutover_not_proven"'
  assert_contains "${output}" '"real_areamatrix_archive_not_proven"'
  assert_contains "${output}" '"real_areamatrix_shim_retirement_not_proven"'
  assert_contains "${output}" '"release_candidate_snapshot_not_ready"'
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
    echo "smoke-release-packaging-proof: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
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
  echo "smoke-release-packaging-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
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
# Release Packaging Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Release Packaging Proof Fixture Workflow
EOF

cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items:
  - id: global-fixture-note
    status: reference-only
    type: fixture
    title: Release packaging proof fixture residual
    source: workflow/residuals/residuals.yaml
    current_impact: proves global residual inventory exists
    executable_task: false
    promotion_required: false
    close_condition: fixture smoke passes
version_residuals:
  - version: v1-mvp
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    status: fixture-reviewed
    summary: Release packaging proof fixture version residuals
EOF

cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Release Packaging Proof Fixture Templates
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
    title: Release packaging proof v1 fixture residual
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
# Release Packaging Proof Fixture Template Version
EOF

cat >"${project_root}/tasks/indexes/residuals.md" <<'EOF'
# Release Packaging Proof Fixture Residual Index
EOF

cat >"${project_root}/tasks/backlog/README.md" <<'EOF'
# Release Packaging Proof Fixture Backlog
EOF

local_release_artifact="${artifact_root}/release-packaging-readiness.json"
cat >"${local_release_artifact}" <<'EOF'
{"kind":"release_packaging_proof_fixture","release_readiness_local_artifact":true}
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Release Packaging Fixture
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

release_packaging_facts=(
  release_final_gate_passed
  release_evidence_bundle_metadata_only
  release_package_preview_created_no_package
  distribution_preview_no_upload_sign_tag_push
  publish_gate_and_approval_preview_created_no_publish_or_approval
  rollout_plan_preview_created_no_rollout_state
  no_release_package_publish_rollout_apply_opened
)

release_packaging_fact_args=()
for fact in "${release_packaging_facts[@]}"; do
  release_packaging_fact_args+=(--fact "${fact}")
done

echo "smoke-release-packaging-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-release-packaging-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-release-packaging-proof: seed release final gate fixture inputs"
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
    'release-packaging-readiness.json',
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

echo "smoke-release-packaging-proof: release evidence bundle is ready"
release_final_gate_json="$(go run ./cmd/areaflow release final-gate --json)"
assert_contains "${release_final_gate_json}" '"status": "pass"'
assert_contains "${release_final_gate_json}" '"readiness_status": "ready"'
assert_contains "${release_final_gate_json}" '"acceptance_gate_status": "pass"'
release_evidence_json="$(go run ./cmd/areaflow release evidence-bundle --json)"
assert_contains "${release_evidence_json}" '"status": "ready"'
assert_contains "${release_evidence_json}" '"mode": "read_only_release_evidence_bundle"'
assert_contains "${release_evidence_json}" '"final_gate_status": "pass"'
assert_contains "${release_evidence_json}" '"backup_status": "ready"'

echo "smoke-release-packaging-proof: release packaging proof rejects incomplete complete status"
assert_fails_contains \
  "complete release packaging proof missing required facts" \
  go run ./cmd/areaflow completion release-packaging-proof record "${project_key}" \
    --status complete \
    --fact release_final_gate_passed \
    --json

echo "smoke-release-packaging-proof: release packaging proof complete"
proof_json="$(go run ./cmd/areaflow completion release-packaging-proof record "${project_key}" \
  --status complete \
  "${release_packaging_fact_args[@]}" \
  --summary "release packaging proof smoke review" \
  --evidence-uri "scripts/smoke-release-packaging-proof.sh#release-packaging" \
  --idempotency-key "release-packaging-proof-smoke:${project_key}" \
  --reason "record release packaging proof smoke evidence" \
  --json)"
assert_contains "${proof_json}" '"proof_status": "complete"'
assert_contains "${proof_json}" '"decision": "allowed"'
assert_contains "${proof_json}" '"missing_facts": []'
assert_contains "${proof_json}" '"created": true'
assert_contains "${proof_json}" '"release_evidence_bundle_status": "ready"'
assert_contains "${proof_json}" '"release_evidence_bundle_ready": true'
assert_contains "${proof_json}" '"project_write_attempted": false'
assert_contains "${proof_json}" '"execution_write_attempted": false'
assert_contains "${proof_json}" '"release_package_created": false'
assert_contains "${proof_json}" '"release_state_written": false'
assert_contains "${proof_json}" '"release_approval_created": false'
assert_contains "${proof_json}" '"rollout_state_created": false'
assert_contains "${proof_json}" '"migration_apply_attempted": false'
assert_contains "${proof_json}" '"tag_created": false'
assert_contains "${proof_json}" '"package_signed": false'
assert_contains "${proof_json}" '"artifact_uploaded": false'
assert_contains "${proof_json}" '"git_push_attempted": false'
assert_contains "${proof_json}" '"publish_attempted": false'
assert_contains "${proof_json}" '"commands_run": false'
assert_contains "${proof_json}" '"area_matrix_protected_paths_touched": false'

echo "smoke-release-packaging-proof: release packaging proof idempotent replay"
replay_json="$(go run ./cmd/areaflow completion release-packaging-proof record "${project_key}" \
  --status complete \
  "${release_packaging_fact_args[@]}" \
  --summary "release packaging proof smoke review" \
  --evidence-uri "scripts/smoke-release-packaging-proof.sh#release-packaging" \
  --idempotency-key "release-packaging-proof-smoke:${project_key}" \
  --reason "record release packaging proof smoke evidence" \
  --json)"
assert_contains "${replay_json}" '"created": false'

echo "smoke-release-packaging-proof: completion audit consumes release packaging proof but stays incomplete"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_completion_real_100_guardrail "${completion_json}"
assert_contains "${completion_json}" '"key": "E5_release_packaging_preview"'
assert_contains "${completion_json}" '"status": "complete"'
assert_contains "${completion_json}" '"release_packaging_gate_passed": true'
assert_contains "${completion_json}" '"release_packaging_proof_bundle_bound": true'
assert_contains "${completion_json}" '"release_packaging_proof_status": "complete"'
assert_contains "${completion_json}" '"latest_release_packaging_proof_evidence_uri": "scripts/smoke-release-packaging-proof.sh#release-packaging"'
assert_not_contains "${completion_json}" '"release_final_gate_not_passed"'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"execution_write_attempted": false'
assert_contains "${completion_json}" '"release_package_created": false'
assert_contains "${completion_json}" '"release_state_written": false'
assert_contains "${completion_json}" '"release_approval_created": false'
assert_contains "${completion_json}" '"rollout_state_created": false'
assert_contains "${completion_json}" '"migration_apply_attempted": false'
assert_contains "${completion_json}" '"tag_created": false'
assert_contains "${completion_json}" '"package_signed": false'
assert_contains "${completion_json}" '"artifact_uploaded": false'
assert_contains "${completion_json}" '"git_push_attempted": false'
assert_contains "${completion_json}" '"publish_attempted": false'
assert_contains "${completion_json}" '"commands_run": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-release-packaging-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-release-packaging-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-release-packaging-proof: pass ${project_key} fixture=${fixture_dir}"
