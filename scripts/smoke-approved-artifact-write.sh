#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-approved-artifact-write: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_ARTIFACT_WRITE_PROJECT_KEY:-areamatrix-artifact-fixture}"
workflow_label="${AREAFLOW_ARTIFACT_WRITE_WORKFLOW_VERSION:-artifact-write-smoke-$(date +%Y%m%d%H%M%S)}"
worker_key="${workflow_label}-worker"
artifact_label="${AREAFLOW_ARTIFACT_WRITE_LABEL:-approval-note}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-artifact-write.XXXXXX")"
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
    echo "smoke-approved-artifact-write: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-approved-artifact-write: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-approved-artifact-write: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_exists() {
  local path="$1"

  if [[ -e "${path}" ]]; then
    echo "smoke-approved-artifact-write: expected path to be absent: ${path}" >&2
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

real_status_before="__skipped__"
real_readme_before="__skipped__"
if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_before="$(file_fingerprint "${real_areamatrix_readme}")"
else
  echo "smoke-approved-artifact-write: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
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

This fixture exists only for AreaFlow approved artifact write smoke tests.
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
  - id: global-artifact-write-fixture-note
    status: reference-only
    type: fixture
    title: Global artifact write fixture residual
    source: workflow/residuals/residuals.yaml
    current_impact: proves global residual import for approved artifact write smoke
    executable_task: false
    promotion_required: false
    close_condition: approved artifact write smoke passes
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
  - id: v1-artifact-write-fixture-residual
    status: blocked-decision
    type: fixture
    title: Version artifact write fixture residual
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    current_impact: proves version residual import for approved artifact write smoke
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
  name: AreaMatrix Approved Artifact Write Fixture
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

echo "smoke-approved-artifact-write: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-approved-artifact-write: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-approved-artifact-write: project import ${project_key}"
import_output="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output}" "imported ${project_key}"

echo "smoke-approved-artifact-write: workflow version create ${workflow_label}"
version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${workflow_label}" --json)"
assert_contains "${version_json}" '"import_mode": "authored"'

echo "smoke-approved-artifact-write: workflow mark queue ready"
queue_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${workflow_label}" --stage queue --item-type queue_candidate --reason "approved artifact write smoke queue" --json)"
assert_contains "${queue_ready_json}" '"status": "ready"'

echo "smoke-approved-artifact-write: workflow mark promotion_preview ready"
promotion_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${workflow_label}" --stage promotion_preview --item-type promotion_preview --reason "approved artifact write smoke promotion" --json)"
assert_contains "${promotion_ready_json}" '"status": "ready"'

echo "smoke-approved-artifact-write: workflow gate promotion_preview"
promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" promotion_preview --json)"
assert_contains "${promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${promotion_gate_json}" '"status": "pass"'

echo "smoke-approved-artifact-write: workflow transition preview"
transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${workflow_label}" --json)"
assert_contains "${transition_json}" '"status": "ready"'

echo "smoke-approved-artifact-write: workflow approval approved"
approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${workflow_label}" --decision approved --reason "approved artifact write smoke approval" --json)"
assert_contains "${approval_json}" '"decision": "approved"'
assert_contains "${approval_json}" '"transition_status": "ready"'

echo "smoke-approved-artifact-write: workflow gate approval_gate"
approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" approval_gate --json)"
assert_contains "${approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${approval_gate_json}" '"status": "pass"'

echo "smoke-approved-artifact-write: workflow gate live_mapping_gate"
live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" live_mapping_gate --json)"
assert_contains "${live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${live_mapping_gate_json}" '"status": "pass"'
assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-approved-artifact-write: worker register"
worker_json="$(go run ./cmd/areaflow worker register "${project_key}" --worker-key "${worker_key}" --capability write_artifacts --json)"
assert_contains "${worker_json}" '"worker_key": "'"${worker_key}"'"'
assert_contains "${worker_json}" '"write_artifacts"'
assert_contains "${worker_json}" '"status": "online"'

echo "smoke-approved-artifact-write: run approved-artifact-write-queue"
queue_json="$(go run ./cmd/areaflow run approved-artifact-write-queue "${project_key}" "${workflow_label}" --artifact-label "${artifact_label}" --idempotency-key "approved-artifact-write-queue-${workflow_label}" --reason "approved artifact write smoke queue" --json)"
assert_contains "${queue_json}" '"run_type": "approved_artifact_write"'
assert_contains "${queue_json}" '"run_kind": "execution"'
assert_contains "${queue_json}" '"status": "queued"'
assert_contains "${queue_json}" '"dry_run": false'
assert_contains "${queue_json}" '"task_kind": "approved_artifact_write_task"'
assert_contains "${queue_json}" '"artifact_label": "'"${artifact_label}"'"'
assert_contains "${queue_json}" '"area_flow_artifact_written": false'
assert_contains "${queue_json}" '"area_flow_execution_state_written": true'
assert_contains "${queue_json}" '"project_read_attempted": false'
assert_contains "${queue_json}" '"project_write_attempted": false'
assert_contains "${queue_json}" '"execution_write_attempted": false'
assert_contains "${queue_json}" '"engine_call_attempted": false'
assert_contains "${queue_json}" '"commands_run": false'
assert_contains "${queue_json}" '"secrets_resolved": false'
assert_contains "${queue_json}" '"network_used": false'

run_id="$(QUEUE_JSON="${queue_json}" python3 - <<'PY'
import json
import os
print(json.loads(os.environ["QUEUE_JSON"])["run"]["id"])
PY
)"
if [[ -z "${run_id}" ]]; then
  echo "smoke-approved-artifact-write: expected approved artifact write run id" >&2
  exit 1
fi

echo "smoke-approved-artifact-write: run approved-artifact-write-queue replay"
queue_replay_json="$(go run ./cmd/areaflow run approved-artifact-write-queue "${project_key}" "${workflow_label}" --artifact-label "${artifact_label}" --idempotency-key "approved-artifact-write-queue-${workflow_label}" --reason "approved artifact write smoke queue" --json)"
assert_contains "${queue_replay_json}" '"created": false'

echo "smoke-approved-artifact-write: worker approved-artifact-write"
apply_json="$(go run ./cmd/areaflow worker approved-artifact-write "${project_key}" "${worker_key}" --run-id "${run_id}" --capability write_artifacts --idempotency-key "approved-artifact-write-apply-${workflow_label}" --reason "approved artifact write smoke apply" --json)"
assert_contains "${apply_json}" '"status": "artifact_written"'
assert_contains "${apply_json}" '"decision": "allowed"'
assert_contains "${apply_json}" '"lease_kind": "approved_artifact_write"'
assert_contains "${apply_json}" '"attempt_kind": "approved_artifact_write"'
assert_contains "${apply_json}" '"dry_run": false'
assert_contains "${apply_json}" '"artifact_type": "approved_artifact_write_report"'
assert_contains "${apply_json}" '"artifact_label": "'"${artifact_label}"'"'
assert_contains "${apply_json}" '"area_flow_artifact_written": true'
assert_contains "${apply_json}" '"area_flow_execution_state_written": true'
assert_contains "${apply_json}" '"project_read_attempted": false'
assert_contains "${apply_json}" '"project_write_attempted": false'
assert_contains "${apply_json}" '"execution_write_attempted": false'
assert_contains "${apply_json}" '"engine_call_attempted": false'
assert_contains "${apply_json}" '"commands_run": false'
assert_contains "${apply_json}" '"secrets_resolved": false'
assert_contains "${apply_json}" '"network_used": false'
assert_contains "${apply_json}" '"worker_started": false'
assert_contains "${apply_json}" '"task_claimed": true'
assert_contains "${apply_json}" '"lease_created": true'
assert_contains "${apply_json}" '"attempt_created": true'
assert_contains "${apply_json}" '"artifact_created": true'
assert_contains "${apply_json}" '"artifact_write_passed": true'

artifact_uri="$(APPLY_JSON="${apply_json}" python3 - <<'PY'
import json
import os
print(json.loads(os.environ["APPLY_JSON"])["artifact"]["uri"])
PY
)"
case "${artifact_uri}" in
  "${artifact_root}"/*) ;;
  *)
    echo "smoke-approved-artifact-write: artifact escaped fixture store: ${artifact_uri}" >&2
    exit 1
    ;;
esac
if [[ ! -f "${artifact_uri}" ]]; then
  echo "smoke-approved-artifact-write: expected artifact file at ${artifact_uri}" >&2
  exit 1
fi
assert_contains "$(cat "${artifact_uri}")" '"approved_artifact_write": true'
assert_contains "$(cat "${artifact_uri}")" '"project_write_attempted": false'
assert_contains "$(cat "${artifact_uri}")" '"engine_call_attempted": false'

echo "smoke-approved-artifact-write: worker approved-artifact-write replay"
apply_replay_json="$(go run ./cmd/areaflow worker approved-artifact-write "${project_key}" "${worker_key}" --run-id "${run_id}" --capability write_artifacts --idempotency-key "approved-artifact-write-apply-${workflow_label}" --reason "approved artifact write smoke apply" --json)"
assert_contains "${apply_replay_json}" '"created": false'
assert_contains "${apply_replay_json}" '"status": "artifact_written"'
assert_contains "${apply_replay_json}" '"artifact_write_passed": true'

db_counts="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -v "run_id=${run_id}" -At <<'SQL'
SELECT
  (SELECT COUNT(*)
   FROM runs r
   JOIN projects p ON p.id = r.project_id
   WHERE p.project_key = :'project_key'
     AND r.id = :'run_id'::bigint
     AND r.run_type = 'approved_artifact_write'
     AND r.run_kind = 'execution'
     AND r.status = 'artifact_written'
     AND r.dry_run = false
     AND r.summary ->> 'area_flow_artifact_written' = 'true'
     AND r.summary ->> 'project_write_attempted' = 'false'),
  (SELECT COUNT(*)
   FROM run_tasks rt
   WHERE rt.run_id = :'run_id'::bigint
     AND rt.task_kind = 'approved_artifact_write_task'
     AND rt.status = 'artifact_written'),
  (SELECT COUNT(*)
   FROM leases l
   WHERE l.run_id = :'run_id'::bigint
     AND l.lease_kind = 'approved_artifact_write'
     AND l.status = 'completed'
     AND l.scope ->> 'approved_artifact_write' = 'true'),
  (SELECT COUNT(*)
   FROM run_attempts ra
   WHERE ra.run_id = :'run_id'::bigint
     AND ra.attempt_kind = 'approved_artifact_write'
     AND ra.status = 'passed'
     AND ra.dry_run = false
     AND ra.metadata ->> 'area_flow_execution_state_written' = 'true'),
  (SELECT COUNT(*)
   FROM artifacts a
   WHERE a.run_id = :'run_id'::bigint
     AND a.artifact_type = 'approved_artifact_write_report'
     AND a.storage_backend = 'local'
     AND a.source_path LIKE 'versions/%/approved-artifact-write/%'),
  (SELECT COUNT(*)
   FROM command_requests cr
   JOIN projects p ON p.id = cr.project_id
   WHERE p.project_key = :'project_key'
     AND cr.command_type = 'run.approved_artifact_write_queue'
     AND cr.completed_at IS NOT NULL
     AND cr.response ->> 'area_flow_artifact_written' = 'false'),
  (SELECT COUNT(*)
   FROM command_requests cr
   JOIN projects p ON p.id = cr.project_id
   WHERE p.project_key = :'project_key'
     AND cr.command_type = 'worker.approved_artifact_write'
     AND cr.completed_at IS NOT NULL
     AND cr.response ->> 'area_flow_artifact_written' = 'true'
     AND cr.response ->> 'project_write_attempted' = 'false'
     AND cr.response ->> 'engine_call_attempted' = 'false'),
  (SELECT COUNT(*)
   FROM events e
   WHERE e.run_id = :'run_id'::bigint
     AND e.event_type = 'worker.approved_artifact_write.allowed'),
  (SELECT COUNT(*)
   FROM audit_events ae
   JOIN projects p ON p.id = ae.project_id
   WHERE p.project_key = :'project_key'
     AND ae.action = 'worker.approved_artifact_write'
     AND ae.capability = 'write_artifacts'
     AND ae.resource_type = 'artifact'
     AND ae.decision = 'allowed');
SQL
)"
IFS='|' read -r run_count task_count lease_count attempt_count artifact_count queue_command_count apply_command_count event_count audit_count <<<"${db_counts}"
if [[ "${run_count}" != "1" || "${task_count}" != "1" || "${lease_count}" != "1" || "${attempt_count}" != "1" || "${artifact_count}" != "1" || "${queue_command_count}" != "1" || "${apply_command_count}" != "1" || "${event_count}" != "1" || "${audit_count}" != "1" ]]; then
  echo "smoke-approved-artifact-write: unexpected DB counts: ${db_counts}" >&2
  exit 1
fi
echo "summary|${run_count}|${task_count}|${lease_count}|${attempt_count}|${artifact_count}|${queue_command_count}|${apply_command_count}|${event_count}|${audit_count}"

assert_not_exists "${project_root}/workflow/versions/${workflow_label}/execution"
assert_not_exists "${project_root}/workflow/versions/${workflow_label}/execution/_shared/progress.json"

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"
  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-approved-artifact-write: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi
  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-approved-artifact-write: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

open_connections="$(psql "${AREAFLOW_DATABASE_URL}" -At <<'SQL'
SELECT COUNT(*)
FROM pg_stat_activity
WHERE datname = current_database()
  AND pid <> pg_backend_pid();
SQL
)"
if [[ "${open_connections}" != "0" ]]; then
  echo "smoke-approved-artifact-write: expected no residual database connections, got ${open_connections}" >&2
  exit 1
fi

echo "smoke-approved-artifact-write: pass ${project_key} fixture=${fixture_dir}"
