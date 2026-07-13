#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-v1-stable-fixture: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_V1_FIXTURE_PROJECT_KEY:-areamatrix-v1-fixture}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-v1-stable.XXXXXX")"
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
    echo "smoke-v1-stable-fixture: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-v1-stable-fixture: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

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
  echo "smoke-v1-stable-fixture: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
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

This fixture exists only for AreaFlow v1.0 stable platform smoke tests.
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
  name: AreaMatrix V1 Fixture
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
    - AGENTS.md
    - docs/**
    - workflow/**
    - tasks/**
    - .ai-governance/**

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
    - ./dev tasks status
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

echo "smoke-v1-stable-fixture: running smoke-local against fixture ${project_key}"
AREAFLOW_SMOKE_PROJECT="${project_key}" \
AREAFLOW_SMOKE_CONFIG="${config_path}" \
AREAFLOW_SMOKE_WORKFLOW_VERSION="v1-stable-smoke-$(date +%Y%m%d%H%M%S)" \
AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1 \
bash scripts/smoke-local.sh

fixture_status="${project_root}/.areaflow/status.json"
if [[ ! -f "${fixture_status}" ]]; then
  echo "smoke-v1-stable-fixture: expected fixture status projection at ${fixture_status}" >&2
  exit 1
fi

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-v1-stable-fixture: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-v1-stable-fixture: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-v1-stable-fixture: pass ${project_key} fixture=${fixture_dir}"
