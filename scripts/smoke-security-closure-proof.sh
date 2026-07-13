#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-security-closure-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_SECURITY_CLOSURE_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-security-closure.XXXXXX")"
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
    echo "smoke-security-closure-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-security-closure-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-security-closure-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-security-closure-proof: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
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
    echo "smoke-security-closure-proof: expected command to fail: $*" >&2
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
  echo "smoke-security-closure-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${project_root}/workflow" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Security Closure Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Security Closure Proof Fixture Workflow
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Security Closure Fixture
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
  imported_versions: []
  immutable_imports: []
EOF

security_closure_facts=(
  project_key_isolation_covers_workflow_run_lease_artifact_secret_audit
  global_id_route_guard_project_key_visibility_proven
  permission_doctor_default_read_only_deny_first_passed
  audit_coverage_covers_enabled_capabilities
  auth_team_token_secret_remote_worker_remain_readiness_only
  no_forbidden_v1_security_capability_opened
)

security_closure_fact_args=()
for fact in "${security_closure_facts[@]}"; do
  security_closure_fact_args+=(--fact "${fact}")
done

echo "smoke-security-closure-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-security-closure-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-security-closure-proof: seed security audit coverage"
psql "${AREAFLOW_DATABASE_URL}" \
  -v "project_key=${project_key}" >/dev/null <<'SQL'
WITH project_scope AS (
  SELECT id
  FROM projects
  WHERE project_key = :'project_key'
),
audit_seed(action, capability, resource_type, resource, decision, reason) AS (
  VALUES
    ('project.upsert', 'project_config', 'project', :'project_key', 'allowed', 'fixture project registration audit coverage evidence'),
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

echo "smoke-security-closure-proof: security closure proof rejects incomplete complete status"
assert_fails_contains \
  "complete security closure proof missing required facts" \
  go run ./cmd/areaflow completion security-closure-proof record "${project_key}" \
    --status complete \
    --fact project_key_isolation_covers_workflow_run_lease_artifact_secret_audit \
    --json

echo "smoke-security-closure-proof: security closure proof complete"
proof_json="$(go run ./cmd/areaflow completion security-closure-proof record "${project_key}" \
  --status complete \
  "${security_closure_fact_args[@]}" \
  --summary "security closure proof smoke review" \
  --evidence-uri "scripts/smoke-security-closure-proof.sh#security-closure" \
  --idempotency-key "security-closure-proof-smoke:${project_key}" \
  --reason "record security closure proof smoke evidence" \
  --json)"
assert_contains "${proof_json}" '"proof_status": "complete"'
assert_contains "${proof_json}" '"decision": "allowed"'
assert_contains "${proof_json}" '"missing_facts": []'
assert_contains "${proof_json}" '"created": true'
assert_contains "${proof_json}" '"project_write_attempted": false'
assert_contains "${proof_json}" '"execution_write_attempted": false'
assert_contains "${proof_json}" '"authorization_changed": false'
assert_contains "${proof_json}" '"secret_plaintext_read": false'
assert_contains "${proof_json}" '"remote_worker_credentials_issued": false'
assert_contains "${proof_json}" '"commands_run": false'
assert_contains "${proof_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${proof_json}" '"security_closure_binding_status": "pass"'
assert_contains "${proof_json}" '"security_closure_binding_blockers": []'
assert_contains "${proof_json}" '"security_boundary_status": "ready"'
assert_contains "${proof_json}" '"permission_doctor_status": "pass"'
assert_contains "${proof_json}" '"audit_coverage_status": "pass"'

echo "smoke-security-closure-proof: security closure proof idempotent replay"
replay_json="$(go run ./cmd/areaflow completion security-closure-proof record "${project_key}" \
  --status complete \
  "${security_closure_fact_args[@]}" \
  --summary "security closure proof smoke review" \
  --evidence-uri "scripts/smoke-security-closure-proof.sh#security-closure" \
  --idempotency-key "security-closure-proof-smoke:${project_key}" \
  --reason "record security closure proof smoke evidence" \
  --json)"
assert_contains "${replay_json}" '"created": false'

echo "smoke-security-closure-proof: completion audit consumes security closure proof but stays incomplete"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_contains "${completion_json}" '"key": "E8_security_permission_isolation"'
assert_contains "${completion_json}" '"status": "complete"'
assert_contains "${completion_json}" '"security_closure_gate_passed": true'
assert_contains "${completion_json}" '"security_closure_proof_status": "complete"'
assert_contains "${completion_json}" '"latest_security_closure_proof_evidence_uri": "scripts/smoke-security-closure-proof.sh#security-closure"'
assert_not_contains "${completion_json}" '"project_isolation_smoke_missing"'
assert_not_contains "${completion_json}" '"audit_gap_closure_missing"'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"execution_write_attempted": false'
assert_contains "${completion_json}" '"authorization_changed": false'
assert_contains "${completion_json}" '"secret_plaintext_read": false'
assert_contains "${completion_json}" '"remote_worker_credentials_issued": false'
assert_contains "${completion_json}" '"commands_run": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${completion_json}" '"security_closure_binding_status": "pass"'
assert_contains "${completion_json}" '"security_closure_current_binding_bound": true'
assert_contains "${completion_json}" '"permission_doctor_status": "pass"'
assert_contains "${completion_json}" '"audit_coverage_status": "pass"'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-security-closure-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-security-closure-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-security-closure-proof: pass ${project_key} fixture=${fixture_dir}"
