#!/usr/bin/env bash
set -euo pipefail

smoke_mode="${AREAFLOW_WEB_SMOKE_MODE:-fixture}"
if [[ "${smoke_mode}" != "fixture" && "${smoke_mode}" != "real-areamatrix" ]]; then
  echo "smoke-web: unsupported AREAFLOW_WEB_SMOKE_MODE=${smoke_mode}" >&2
  exit 1
fi

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  if [[ "${smoke_mode}" == "real-areamatrix" ]]; then
    echo "smoke-web: blocked; AREAFLOW_DATABASE_URL is required for real AreaMatrix smoke" >&2
    exit 1
  fi
  echo "smoke-web: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "smoke-web: skipped; npm is not available"
  exit 0
fi

api_host="${AREAFLOW_WEB_SMOKE_API_HOST:-127.0.0.1}"
api_port="${AREAFLOW_WEB_SMOKE_API_PORT:-3857}"
web_host="${AREAFLOW_WEB_SMOKE_WEB_HOST:-127.0.0.1}"
web_port="${AREAFLOW_WEB_SMOKE_WEB_PORT:-5175}"
smoke_id="$(date +%Y%m%d%H%M%S)"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-smoke-web.XXXXXX")"
default_project_key="areamatrix-web-fixture-${smoke_id}"
default_project_root="${tmp_dir}/areamatrix-root"
default_project_config="${tmp_dir}/areaflow.yaml"
if [[ "${smoke_mode}" == "real-areamatrix" ]]; then
  default_project_key="areamatrix"
  default_project_root="/Users/as/Ai-Project/project/AreaMatrix"
  default_project_config="examples/areamatrix/areaflow.yaml"
fi
project_key="${AREAFLOW_WEB_SMOKE_PROJECT:-${default_project_key}}"
workflow_label="${AREAFLOW_WEB_SMOKE_WORKFLOW_VERSION:-web-smoke-$(date +%Y%m%d%H%M%S)}"
ready_workflow_label="${AREAFLOW_WEB_SMOKE_READY_WORKFLOW_VERSION:-${workflow_label}-ready}"
project_root="${AREAFLOW_WEB_SMOKE_PROJECT_ROOT:-${default_project_root}}"
artifact_root="${tmp_dir}/artifact-store"
project_config="${AREAFLOW_WEB_SMOKE_CONFIG:-${default_project_config}}"
api_log="${tmp_dir}/api.log"
web_log="${tmp_dir}/web.log"
api_bin="${tmp_dir}/areaflow"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
real_readonly_counts_before=""
base_database_url="${AREAFLOW_DATABASE_URL}"
smoke_schema=""

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

api_pid=""
web_pid=""

cleanup() {
  if [[ -n "${web_pid}" ]]; then
    kill "${web_pid}" >/dev/null 2>&1 || true
    wait "${web_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${api_pid}" ]]; then
    kill "${api_pid}" >/dev/null 2>&1 || true
    wait "${api_pid}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${smoke_schema}" ]]; then
    psql "${base_database_url}" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS ${smoke_schema} CASCADE" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

if [[ "${smoke_mode}" == "fixture" ]]; then
  smoke_schema="areaflow_web_smoke_$$_${RANDOM}"
  psql "${base_database_url}" -v ON_ERROR_STOP=1 -c "CREATE SCHEMA ${smoke_schema}" >/dev/null
  if [[ "${base_database_url}" == *\?* ]]; then
    export AREAFLOW_DATABASE_URL="${base_database_url}&options=-csearch_path%3D${smoke_schema}"
  else
    export AREAFLOW_DATABASE_URL="${base_database_url}?options=-csearch_path%3D${smoke_schema}"
  fi
fi

create_fixture() {
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

This fixture exists only for AreaFlow web smoke tests.
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
  - id: global-web-fixture-note
    status: reference-only
    type: fixture
    title: Global web fixture residual
    source: workflow/residuals/residuals.yaml
    current_impact: proves global residual import for the web smoke
    executable_task: false
    promotion_required: false
    close_condition: web smoke passes
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
  - id: v1-web-fixture-residual
    status: blocked-decision
    type: fixture
    title: Version web fixture residual
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    current_impact: proves version residual import for the web smoke
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

  cat >"${project_config}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Web Fixture ${project_key}
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
}

wait_for_http() {
  local url="$1"
  local name="$2"
  local deadline=$((SECONDS + 60))

  while true; do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    if (( SECONDS >= deadline )); then
      echo "smoke-web: ${name} did not become ready: ${url}" >&2
      if [[ -f "${api_log}" ]]; then
        echo "smoke-web: api log" >&2
        tail -80 "${api_log}" >&2 || true
      fi
      if [[ -f "${web_log}" ]]; then
        echo "smoke-web: web log" >&2
        tail -80 "${web_log}" >&2 || true
      fi
      exit 1
    fi
    sleep 1
  done
}

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-web: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-web: output unexpectedly contained pattern: ${pattern}" >&2
    exit 1
  fi
}

file_fingerprint() {
  areaflow_file_fingerprint "$@"
}

protected_path_git_status() {
  areaflow_protected_path_git_status "${project_root}" "${protected_paths[@]}"
}

protected_path_fingerprint() {
  areaflow_protected_path_fingerprint "${project_root}" ".areaflow/status.json" "${protected_paths[@]}"
}

readonly_side_effect_counts() {
  areaflow_readonly_side_effect_counts "${AREAFLOW_DATABASE_URL}" "${project_key}"
}

real_status_before=""
real_workflow_readme_before=""
real_protected_path_fingerprint_before=""
real_protected_paths_status_before=""

capture_real_areamatrix_baseline() {
  if [[ "${project_key}" != "areamatrix" ]]; then
    echo "smoke-web: real AreaMatrix mode requires project key areamatrix" >&2
    exit 1
  fi
  if [[ "${project_root}" != "/Users/as/Ai-Project/project/AreaMatrix" ]]; then
    echo "smoke-web: real AreaMatrix mode requires root /Users/as/Ai-Project/project/AreaMatrix" >&2
    exit 1
  fi
  if [[ ! -d "${project_root}" ]]; then
    echo "smoke-web: missing AreaMatrix project root: ${project_root}" >&2
    exit 1
  fi

  real_status_before="$(file_fingerprint "${status_path}")"
  real_workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
  real_protected_path_fingerprint_before="$(protected_path_fingerprint)"
  real_protected_paths_status_before="$(protected_path_git_status)"
  if [[ -n "${real_protected_paths_status_before}" ]]; then
    echo "smoke-web: AreaMatrix protected paths are dirty before browser smoke:" >&2
    echo "${real_protected_paths_status_before}" >&2
    exit 1
  fi
}

assert_real_areamatrix_unchanged() {
  local status_after
  local workflow_readme_after
  local protected_path_fingerprint_after
  local protected_paths_status_after

  status_after="$(file_fingerprint "${status_path}")"
  workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
  protected_path_fingerprint_after="$(protected_path_fingerprint)"
  protected_paths_status_after="$(protected_path_git_status)"

  if [[ "${real_status_before}" != "${status_after}" ]]; then
    echo "smoke-web: AreaMatrix status export changed unexpectedly: ${status_path}" >&2
    exit 1
  fi
  if [[ "${real_workflow_readme_before}" != "${workflow_readme_after}" ]]; then
    echo "smoke-web: AreaMatrix workflow README changed unexpectedly: ${workflow_readme}" >&2
    exit 1
  fi
  if [[ "${real_protected_path_fingerprint_before}" != "${protected_path_fingerprint_after}" ]]; then
    echo "smoke-web: AreaMatrix protected path fingerprint changed unexpectedly:" >&2
    echo "smoke-web: protected_path_fingerprint_before=${real_protected_path_fingerprint_before}" >&2
    echo "smoke-web: protected_path_fingerprint_after=${protected_path_fingerprint_after}" >&2
    exit 1
  fi
  if [[ "${real_protected_paths_status_before}" != "${protected_paths_status_after}" ]]; then
    echo "smoke-web: AreaMatrix protected paths changed during browser smoke:" >&2
    echo "${protected_paths_status_after}" >&2
    exit 1
  fi
}

seed_fixture_dashboard_data() {
  echo "smoke-web: create fixture ${project_key}"
  create_fixture

  echo "smoke-web: seed dashboard data ${ready_workflow_label}"
  go run ./cmd/areaflow migrate up
  go run ./cmd/areaflow project add --config "${project_config}" >/dev/null
  go run ./cmd/areaflow project import "${project_key}" >/dev/null
  go run ./cmd/areaflow project import "${project_key}" >/dev/null
  go run ./cmd/areaflow project doctor "${project_key}" --json >/dev/null
  status_authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
  source_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["source_hash"])' <<<"${status_authorization_json}")"
  validator_preflight="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["validator_preflight"])' <<<"${status_authorization_json}")"
  protected_path_fingerprint_sha256="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["protected_path_fingerprint_sha256"])' <<<"${status_authorization_json}")"
  protected_path_check="git -C ${project_root} status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json"
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
    --approval-actor "smoke-web" \
    --approval-reason "fixture status projection apply" >/dev/null

  version_json="$(go run ./cmd/areaflow workflow version create "${project_key}" "${ready_workflow_label}" --json)"
  assert_contains "${version_json}" '"import_mode": "authored"'

  queue_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${ready_workflow_label}" --stage queue --item-type queue_candidate --reason "web smoke ready path queue" --json)"
  assert_contains "${queue_json}" '"status": "ready"'

  promotion_json="$(go run ./cmd/areaflow workflow version mark-ready "${project_key}" "${ready_workflow_label}" --stage promotion_preview --item-type promotion_preview --reason "web smoke ready path promotion" --json)"
  assert_contains "${promotion_json}" '"status": "ready"'

  promotion_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" promotion_preview --json)"
  assert_contains "${promotion_gate_json}" '"status": "pass"'

  transition_json="$(go run ./cmd/areaflow workflow transition preview "${project_key}" "${ready_workflow_label}" --json)"
  assert_contains "${transition_json}" '"status": "ready"'

  approval_json="$(go run ./cmd/areaflow workflow approval record "${project_key}" "${ready_workflow_label}" --decision approved --reason "web smoke ready path approval" --json)"
  assert_contains "${approval_json}" '"decision": "approved"'

  approval_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" approval_gate --json)"
  assert_contains "${approval_gate_json}" '"status": "pass"'

  live_mapping_gate_json="$(go run ./cmd/areaflow workflow gate run "${project_key}" "${ready_workflow_label}" live_mapping_gate --json)"
  assert_contains "${live_mapping_gate_json}" '"status": "pass"'
  assert_contains "${live_mapping_gate_json}" '"execution_write_attempted": false'

  runner_json="$(go run ./cmd/areaflow run preview "${project_key}" "${ready_workflow_label}" --json)"
  assert_contains "${runner_json}" '"run_type": "runner_preview"'
  assert_contains "${runner_json}" '"dry_run": true'
  assert_contains "${runner_json}" '"artifact_type": "runner_preview_report"'
  runner_run_id="$(awk '
    /"run": \{/ { in_run = 1; next }
    in_run && /"id":/ {
      gsub(/[^0-9]/, "", $0)
      print
      exit
    }
    in_run && /^[[:space:]]*\}/ { in_run = 0 }
  ' <<<"${runner_json}")"
  if [[ -z "${runner_run_id}" ]]; then
    echo "smoke-web: expected runner preview JSON to include run.id" >&2
    exit 1
  fi

  worker_key="${ready_workflow_label}-worker"
  worker_json="$(go run ./cmd/areaflow worker register "${project_key}" --worker-key "${worker_key}" --capability read_project --capability write_artifacts --json)"
  assert_contains "${worker_json}" '"status": "online"'

  worker_run_json="$(go run ./cmd/areaflow worker run-once "${project_key}" "${worker_key}" --run-id "${runner_run_id}" --capability read_project --capability write_artifacts --json)"
  assert_contains "${worker_run_json}" '"claimed": true'
  assert_contains "${worker_run_json}" '"artifact_type": "worker_run_once_report"'
  assert_contains "${worker_run_json}" '"commands_run": false'
  assert_contains "${worker_run_json}" '"writes_attempted": false'
}

seed_real_areamatrix_dashboard_data() {
  echo "smoke-web: capture real AreaMatrix baseline"
  capture_real_areamatrix_baseline

  echo "smoke-web: seed real AreaMatrix dashboard data"
  go run ./cmd/areaflow migrate up
  go run ./cmd/areaflow project add --config "${project_config}" >/dev/null
  go run ./cmd/areaflow project import "${project_key}" >/dev/null
  go run ./cmd/areaflow project import "${project_key}" >/dev/null
  go run ./cmd/areaflow project doctor "${project_key}" --json >/dev/null

  status_authorization_json="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
  assert_contains "${status_authorization_json}" '"required_authorization_phrase": "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
  assert_contains "${status_authorization_json}" '"apply_open": false'
  assert_contains "${status_authorization_json}" '"project_write_attempted": false'
  assert_contains "${status_authorization_json}" '"execution_write_attempted": false'
  assert_contains "${status_authorization_json}" '"engine_call_attempted": false'
  assert_not_contains "${status_authorization_json}" '"required_authorization_phrase": ""'

  real_readonly_counts_before="$(readonly_side_effect_counts)"
}

if [[ "${smoke_mode}" == "real-areamatrix" ]]; then
  seed_real_areamatrix_dashboard_data
else
  seed_fixture_dashboard_data
fi

echo "smoke-web: start AreaFlow API ${api_host}:${api_port}"
go build -o "${api_bin}" ./cmd/areaflow
AREAFLOW_HOST="${api_host}" \
  AREAFLOW_PORT="${api_port}" \
  AREAFLOW_DATABASE_URL="${AREAFLOW_DATABASE_URL}" \
  "${api_bin}" server >"${api_log}" 2>&1 &
api_pid=$!
wait_for_http "http://${api_host}:${api_port}/api/v1/ready" "AreaFlow API readiness"
curl -fsS "http://${api_host}:${api_port}/api/v1/health" >/dev/null

echo "smoke-web: start Vite ${web_host}:${web_port}"
(
  cd web
  AREAFLOW_API_URL="http://${api_host}:${api_port}" \
    ./node_modules/.bin/vite --host "${web_host}" --port "${web_port}" --strictPort
) >"${web_log}" 2>&1 &
web_pid=$!
wait_for_http "http://${web_host}:${web_port}" "Vite"

echo "smoke-web: browser check"
(
  cd web
  node ../scripts/smoke-web-check.mjs \
    "http://${web_host}:${web_port}" \
    "${project_key}" \
    "${ready_workflow_label}" \
    "${smoke_mode}"
)

if [[ "${smoke_mode}" == "real-areamatrix" ]]; then
  real_readonly_counts_after="$(readonly_side_effect_counts)"
  if [[ "${real_readonly_counts_before}" != "${real_readonly_counts_after}" ]]; then
    echo "smoke-web: real AreaMatrix browser check created or modified DB side effects:" >&2
    echo "smoke-web: before=${real_readonly_counts_before}" >&2
    echo "smoke-web: after=${real_readonly_counts_after}" >&2
    exit 1
  fi
  assert_real_areamatrix_unchanged
else
  echo "smoke-web: ops smoke proof record"
  ops_smoke_proof_json="$(go run ./cmd/areaflow ops smoke-proof record "${project_key}" --key web_dashboard_ops_smoke --summary "smoke-web API and dashboard ops readiness check passed" --evidence-uri "scripts/smoke-web.sh" --idempotency-key "ops-web-smoke-proof:${project_key}:${ready_workflow_label}" --reason "record web smoke proof after browser check passed" --json)"
  assert_contains "${ops_smoke_proof_json}" '"proof_key": "web_dashboard_ops_smoke"'
  assert_contains "${ops_smoke_proof_json}" '"status": "recorded"'
  assert_contains "${ops_smoke_proof_json}" '"evidence_status": "pass"'
  assert_contains "${ops_smoke_proof_json}" '"service_process_control_attempted": false'
  assert_contains "${ops_smoke_proof_json}" '"support_bundle_exported": false'
  assert_contains "${ops_smoke_proof_json}" '"migration_apply_attempted": false'
  assert_contains "${ops_smoke_proof_json}" '"remote_telemetry_enabled": false'
  assert_contains "${ops_smoke_proof_json}" '"area_matrix_protected_paths_touched": false'
  assert_contains "${ops_smoke_proof_json}" '"record_command_runs_smoke": false'

  ops_readiness_after_proof_json="$(go run ./cmd/areaflow ops readiness --json)"
  assert_contains "${ops_readiness_after_proof_json}" '"latest_smoke_proof_key": "web_dashboard_ops_smoke"'
  assert_contains "${ops_readiness_after_proof_json}" '"evidence_recorded": true'
fi

echo "smoke-web: ok ${smoke_mode}"
