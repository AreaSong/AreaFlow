#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-execution-plan: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_EXECUTION_PLAN_PROJECT_KEY:-areamatrix-execution-plan-fixture}"
workflow_label="${AREAFLOW_EXECUTION_PLAN_WORKFLOW_VERSION:-execution-plan-smoke-$(date +%Y%m%d%H%M%S)}"
worker_key="${workflow_label}-worker"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-execution-plan.XXXXXX")"
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
    echo "smoke-execution-plan: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-execution-plan: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-execution-plan: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_exists() {
  local path="$1"

  if [[ -e "${path}" ]]; then
    echo "smoke-execution-plan: expected path to be absent: ${path}" >&2
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

count_mutation_tables() {
  psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -At <<'SQL'
SELECT concat_ws('|',
  (SELECT COUNT(*) FROM command_requests cr JOIN projects p ON p.id = cr.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM runs r JOIN projects p ON p.id = r.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM run_tasks rt JOIN projects p ON p.id = rt.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM leases l JOIN projects p ON p.id = l.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM run_attempts ra JOIN projects p ON p.id = ra.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM artifacts a JOIN projects p ON p.id = a.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM events e JOIN projects p ON p.id = e.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM audit_events ae JOIN projects p ON p.id = ae.project_id WHERE p.project_key = :'project_key'),
  (SELECT COUNT(*) FROM worker_heartbeats wh JOIN projects p ON p.id = wh.project_id WHERE p.project_key = :'project_key')
);
SQL
}

real_status_before="__skipped__"
real_readme_before="__skipped__"
if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_before="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_before="$(file_fingerprint "${real_areamatrix_readme}")"
else
  echo "smoke-execution-plan: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Fixture Docs

Minimal source docs coverage for execution plan preview smoke.
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Execution Plan Fixture
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
  imported_versions: []
  immutable_imports: []
EOF

echo "smoke-execution-plan: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-execution-plan: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-execution-plan: workflow version create ${workflow_label}"
version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${workflow_label}" --json)"
assert_contains "${version_json}" '"import_mode": "authored"'

echo "smoke-execution-plan: workflow mark queue ready"
queue_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${workflow_label}" --stage queue --item-type queue_candidate --reason "execution plan smoke queue" --json)"
assert_contains "${queue_ready_json}" '"status": "ready"'

echo "smoke-execution-plan: workflow mark promotion_preview ready"
promotion_ready_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${workflow_label}" --stage promotion_preview --item-type promotion_preview --reason "execution plan smoke promotion" --json)"
assert_contains "${promotion_ready_json}" '"status": "ready"'

echo "smoke-execution-plan: workflow gate promotion_preview"
promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" promotion_preview --json)"
assert_contains "${promotion_gate_json}" '"gate_name": "promotion_preview"'
assert_contains "${promotion_gate_json}" '"status": "pass"'

echo "smoke-execution-plan: workflow transition preview"
transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${workflow_label}" --json)"
assert_contains "${transition_json}" '"status": "ready"'

echo "smoke-execution-plan: workflow approval approved"
approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${workflow_label}" --decision approved --reason "execution plan smoke approval" --json)"
assert_contains "${approval_json}" '"decision": "approved"'
assert_contains "${approval_json}" '"transition_status": "ready"'

echo "smoke-execution-plan: workflow gate approval_gate"
approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" approval_gate --json)"
assert_contains "${approval_gate_json}" '"gate_name": "approval_gate"'
assert_contains "${approval_gate_json}" '"status": "pass"'

echo "smoke-execution-plan: workflow gate live_mapping_gate"
live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${workflow_label}" live_mapping_gate --json)"
assert_contains "${live_mapping_gate_json}" '"gate_name": "live_mapping_gate"'
assert_contains "${live_mapping_gate_json}" '"status": "pass"'
assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'

echo "smoke-execution-plan: worker register"
worker_json="$(go run ./cmd/areaflow worker register "${project_key}" --worker-key "${worker_key}" --capability write_artifacts --json)"
assert_contains "${worker_json}" '"worker_key": "'"${worker_key}"'"'
assert_contains "${worker_json}" '"write_artifacts"'
assert_contains "${worker_json}" '"status": "online"'

echo "smoke-execution-plan: run approved-artifact-write-queue"
queue_json="$(go run ./cmd/areaflow run approved-artifact-write-queue "${project_key}" "${workflow_label}" --artifact-label plan-preview --idempotency-key "execution-plan-queue-${workflow_label}" --reason "execution plan smoke queue" --json)"
assert_contains "${queue_json}" '"run_type": "approved_artifact_write"'
assert_contains "${queue_json}" '"run_kind": "execution"'
assert_contains "${queue_json}" '"status": "queued"'
assert_contains "${queue_json}" '"dry_run": false'
assert_contains "${queue_json}" '"task_kind": "approved_artifact_write_task"'

run_id="$(QUEUE_JSON="${queue_json}" python3 - <<'PY'
import json
import os
print(json.loads(os.environ["QUEUE_JSON"])["run"]["id"])
PY
)"
if [[ -z "${run_id}" ]]; then
  echo "smoke-execution-plan: expected approved artifact write run id" >&2
  exit 1
fi

counts_before="$(count_mutation_tables)"

echo "smoke-execution-plan: run execution-plan ${run_id}"
plan_json="$(go run ./cmd/areaflow run execution-plan "${run_id}" --json)"
PLAN_JSON="${plan_json}" python3 - <<'PY'
import json
import os
import sys

plan = json.loads(os.environ["PLAN_JSON"])
steps = {step["key"]: step for step in plan["steps"]}

def require(condition, message):
    if not condition:
        print(f"smoke-execution-plan: {message}", file=sys.stderr)
        sys.exit(1)

require(plan["mode"] == "read_only_execution_plan_preview", "unexpected mode")
require(plan["status"] == "blocked", "plan must remain blocked while copy/checkpoint/repair are unopened")
require(plan["gate"]["status"] == "pass", "artifact-only execution gate should pass")
for key in [
    "project_read_attempted",
    "project_write_attempted",
    "execution_write_attempted",
    "area_flow_artifact_written",
    "area_flow_execution_state_written",
    "engine_call_attempted",
    "commands_run",
    "secrets_resolved",
    "network_used",
    "task_claimed",
    "worker_started",
    "attempt_created",
    "artifact_created",
]:
    require(plan[key] is False, f"{key} should be false")
require("execution_approval_gate" in steps, "missing execution_approval_gate step")
require(steps["execution_approval_gate"]["status"] == "ready", "execution gate step should be ready")
require(steps["approved_artifact_write"]["status"] == "ready", "approved artifact write should be ready")
require(steps["approved_artifact_write"]["writes_project"] is False, "approved artifact write must not write project")
require(steps["approved_artifact_write"]["writes_areaflow"] is True, "approved artifact write should write AreaFlow")
require(steps["copy"]["status"] == "blocked", "copy should remain blocked")
require(steps["copy"]["writes_project"] is True, "copy should disclose project write risk")
require(steps["copy"]["uses_engine"] is True, "copy should disclose engine risk")
require(steps["copy"]["runs_commands"] is True, "copy should disclose command risk")
require(steps["checkpoint"]["status"] == "blocked", "checkpoint should remain blocked")
require(steps["repair"]["status"] == "waiting", "repair should wait for verify failure")
require(any("managed_project_write_not_open" in blocker for blocker in steps["copy"]["blockers"]), "copy blocker missing")
require(any("checkpoint_apply_not_implemented" in blocker for blocker in steps["checkpoint"]["blockers"]), "checkpoint blocker missing")
require("write_managed_project" in plan["forbidden_actions"], "forbidden actions should include managed project write")
PY

human_plan="$(go run ./cmd/areaflow run execution-plan "${run_id}")"
assert_contains "${human_plan}" "execution plan preview: ${project_key}/${workflow_label} run=${run_id} status=blocked"
assert_contains "${human_plan}" "[ready] approved_artifact_write"
assert_contains "${human_plan}" "[blocked] copy"
assert_contains "${human_plan}" "project_write_attempted: false"
assert_contains "${human_plan}" "engine_call_attempted: false"

counts_after="$(count_mutation_tables)"
if [[ "${counts_before}" != "${counts_after}" ]]; then
  echo "smoke-execution-plan: execution-plan mutated database counts: before=${counts_before} after=${counts_after}" >&2
  exit 1
fi
echo "summary|${counts_before}|${counts_after}"

assert_not_exists "${project_root}/workflow/versions/${workflow_label}/execution"
assert_not_exists "${project_root}/workflow/versions/${workflow_label}/execution/_shared/progress.json"

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"
  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-execution-plan: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi
  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-execution-plan: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
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
  echo "smoke-execution-plan: expected no residual database connections, got ${open_connections}" >&2
  exit 1
fi

echo "smoke-execution-plan: pass ${project_key} fixture=${fixture_dir}"
