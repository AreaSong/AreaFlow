#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-validation-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_VALIDATION_PROOF_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-validation-proof.XXXXXX")"
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
    echo "smoke-validation-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-validation-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-validation-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-validation-proof: expected output to omit pattern: ${pattern}" >&2
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
    echo "smoke-validation-proof: expected command to fail: $*" >&2
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
  echo "smoke-validation-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${project_root}/workflow" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Validation Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Validation Proof Fixture Workflow
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Validation Proof Fixture
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
    write_status: false
    write_artifacts: false
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

  write_paths: []

  forbidden_paths:
    - workflow/versions/*/execution/**
    - workflow/versions/*/execution/_shared/progress.json
    - .areamatrix/**

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
validation_scope="fixture_validation_review"

echo "smoke-validation-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-validation-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-validation-proof: validation proof rejects incomplete complete status"
assert_fails_contains \
  "complete validation proof missing required facts" \
  go run ./cmd/areaflow completion validation-proof record "${project_key}" \
    --status complete \
    --fact go_test_passed \
    --json

echo "smoke-validation-proof: validation proof complete"
validation_json="$(go run ./cmd/areaflow completion validation-proof record "${project_key}" \
  --status complete \
  "${validation_fact_args[@]}" \
  --summary "validation proof smoke review" \
  --evidence-uri "scripts/smoke-validation-proof.sh#validation" \
  "${validation_command_args[@]}" \
  --validation-result-hash "${validation_result_hash}" \
  --validation-started-at "${validation_started_at}" \
  --validation-finished-at "${validation_finished_at}" \
  --validation-scope "${validation_scope}" \
  --idempotency-key "validation-proof-smoke:${project_key}" \
  --reason "record validation proof smoke evidence" \
  --json)"
assert_contains "${validation_json}" '"proof_status": "complete"'
assert_contains "${validation_json}" '"decision": "allowed"'
assert_contains "${validation_json}" '"missing_facts": []'
assert_contains "${validation_json}" '"created": true'
assert_contains "${validation_json}" '"project_write_attempted": false'
assert_contains "${validation_json}" '"execution_write_attempted": false'
assert_contains "${validation_json}" '"engine_call_attempted": false'
assert_contains "${validation_json}" '"commands_run": false'
assert_contains "${validation_json}" '"smoke_run_attempted": false'
assert_contains "${validation_json}" '"web_build_run_by_command": false'
assert_contains "${validation_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${validation_json}" '"validation_evidence_binding_status": "pass"'
assert_contains "${validation_json}" "\"validation_result_hash\": \"${validation_result_hash}\""
assert_contains "${validation_json}" "\"validation_scope\": \"${validation_scope}\""
assert_contains "${validation_json}" '"validation_command_count": 9'

echo "smoke-validation-proof: validation proof idempotent replay"
validation_replay_json="$(go run ./cmd/areaflow completion validation-proof record "${project_key}" \
  --status complete \
  "${validation_fact_args[@]}" \
  --summary "validation proof smoke review" \
  --evidence-uri "scripts/smoke-validation-proof.sh#validation" \
  "${validation_command_args[@]}" \
  --validation-result-hash "${validation_result_hash}" \
  --validation-started-at "${validation_started_at}" \
  --validation-finished-at "${validation_finished_at}" \
  --validation-scope "${validation_scope}" \
  --idempotency-key "validation-proof-smoke:${project_key}" \
  --reason "record validation proof smoke evidence" \
  --json)"
assert_contains "${validation_replay_json}" '"created": false'

echo "smoke-validation-proof: completion audit consumes validation proof but stays incomplete"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_contains "${completion_json}" '"key": "E3_command_api_smoke_evidence"'
assert_contains "${completion_json}" '"validation_gate_passed": true'
assert_contains "${completion_json}" '"latest_validation_proof_evidence_uri": "scripts/smoke-validation-proof.sh#validation"'
assert_contains "${completion_json}" '"validation_evidence_binding_status": "pass"'
assert_contains "${completion_json}" "\"validation_result_hash\": \"${validation_result_hash}\""
assert_contains "${completion_json}" "\"validation_scope\": \"${validation_scope}\""
assert_contains "${completion_json}" '"validation_command_count": 9'
assert_not_contains "${completion_json}" '"fresh_validation_proof_missing"'
assert_contains "${completion_json}" '"key": "E4_areamatrix_dogfood_completion"'
assert_contains "${completion_json}" '"status": "blocked"'
assert_contains "${completion_json}" '"smoke_run_attempted": false'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-validation-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-validation-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-validation-proof: pass ${project_key} fixture=${fixture_dir}"
