#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-source-alignment-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_SOURCE_ALIGNMENT_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-source-alignment.XXXXXX")"
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
    echo "smoke-source-alignment-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-source-alignment-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-source-alignment-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-source-alignment-proof: expected output to omit pattern: ${pattern}" >&2
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
    echo "smoke-source-alignment-proof: expected command to fail: $*" >&2
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
  echo "smoke-source-alignment-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${project_root}/workflow" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Source Alignment Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Source Alignment Proof Fixture Workflow
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Source Alignment Fixture
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

source_alignment_facts=(
  zero_to_hundred_phases_aligned
  v1_and_v1x_boundaries_consistent
  preview_only_not_claimed_as_apply
  implemented_scoped_not_claimed_as_real_cutover
  deferred_high_risk_capabilities_have_contracts
  master_plan_roadmap_phase_backlog_gap_audit_cross_references_current
)

source_alignment_fact_args=()
for fact in "${source_alignment_facts[@]}"; do
  source_alignment_fact_args+=(--fact "${fact}")
done

echo "smoke-source-alignment-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-source-alignment-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-source-alignment-proof: source alignment proof rejects incomplete complete status"
assert_fails_contains \
  "complete source alignment proof missing required facts" \
  go run ./cmd/areaflow completion source-alignment-proof record "${project_key}" \
    --status complete \
    --fact zero_to_hundred_phases_aligned \
    --json

echo "smoke-source-alignment-proof: source alignment proof complete"
proof_json="$(go run ./cmd/areaflow completion source-alignment-proof record "${project_key}" \
  --status complete \
  "${source_alignment_fact_args[@]}" \
  --summary "source alignment proof smoke review" \
  --evidence-uri "scripts/smoke-source-alignment-proof.sh#source-alignment" \
  --idempotency-key "source-alignment-proof-smoke:${project_key}" \
  --reason "record source alignment proof smoke evidence" \
  --json)"
assert_contains "${proof_json}" '"proof_status": "complete"'
assert_contains "${proof_json}" '"decision": "allowed"'
assert_contains "${proof_json}" '"missing_facts": []'
assert_contains "${proof_json}" '"created": true'
assert_contains "${proof_json}" '"project_write_attempted": false'
assert_contains "${proof_json}" '"execution_write_attempted": false'
assert_contains "${proof_json}" '"commands_run": false'
assert_contains "${proof_json}" '"docs_written": false'
assert_contains "${proof_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${proof_json}" '"source_alignment_binding_status": "pass"'
assert_contains "${proof_json}" '"source_alignment_binding_blockers": []'
assert_contains "${proof_json}" '"source_alignment_source_set_hash": "'
assert_contains "${proof_json}" '"source_alignment_source_file_count": '
assert_contains "${proof_json}" '"source_alignment_missing_source_count": 0'
assert_contains "${proof_json}" '"source_alignment_unreadable_source_count": 0'

echo "smoke-source-alignment-proof: source alignment proof idempotent replay"
replay_json="$(go run ./cmd/areaflow completion source-alignment-proof record "${project_key}" \
  --status complete \
  "${source_alignment_fact_args[@]}" \
  --summary "source alignment proof smoke review" \
  --evidence-uri "scripts/smoke-source-alignment-proof.sh#source-alignment" \
  --idempotency-key "source-alignment-proof-smoke:${project_key}" \
  --reason "record source alignment proof smoke evidence" \
  --json)"
assert_contains "${replay_json}" '"created": false'

echo "smoke-source-alignment-proof: completion audit consumes source alignment proof but stays incomplete"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
assert_contains "${completion_json}" '"key": "E1_design_source_alignment"'
assert_contains "${completion_json}" '"source_alignment_gate_passed": true'
assert_contains "${completion_json}" '"source_alignment_binding_status": "pass"'
assert_contains "${completion_json}" '"source_alignment_current_binding_bound": true'
assert_contains "${completion_json}" '"source_alignment_source_set_hash": "'
assert_contains "${completion_json}" '"source_alignment_missing_source_count": 0'
assert_contains "${completion_json}" '"source_alignment_unreadable_source_count": 0'
assert_contains "${completion_json}" '"latest_source_alignment_proof_evidence_uri": "scripts/smoke-source-alignment-proof.sh#source-alignment"'
assert_not_contains "${completion_json}" '"source_alignment_proof_missing"'
assert_contains "${completion_json}" '"key": "E4_areamatrix_dogfood_completion"'
assert_contains "${completion_json}" '"status": "blocked"'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-source-alignment-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-source-alignment-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-source-alignment-proof: pass ${project_key} fixture=${fixture_dir}"
