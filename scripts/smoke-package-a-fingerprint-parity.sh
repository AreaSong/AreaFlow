#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-package-a-fingerprint-parity: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="package-a-fingerprint-parity-$RANDOM-$$"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-package-a-fingerprint.XXXXXX")"
fixture_dir="$(cd "${fixture_dir}" && pwd -P)"
project_root="${fixture_dir}/managed-root"
artifact_root="${fixture_dir}/artifact-store"
config_path="${fixture_dir}/areaflow.yaml"

case "${fixture_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-package-a-fingerprint-parity: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FINGERPRINT_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-package-a-fingerprint-parity: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_eq() {
  local got="$1"
  local want="$2"
  local label="$3"

  if [[ "${got}" != "${want}" ]]; then
    echo "smoke-package-a-fingerprint-parity: ${label}: got ${got}, want ${want}" >&2
    exit 1
  fi
}

assert_ne() {
  local left="$1"
  local right="$2"
  local label="$3"

  if [[ "${left}" == "${right}" ]]; then
    echo "smoke-package-a-fingerprint-parity: ${label}: both values were ${left}" >&2
    exit 1
  fi
}

assert_hex_sha256() {
  local value="$1"
  local label="$2"

  if ! [[ "${value}" =~ ^[0-9a-f]{64}$ ]]; then
    echo "smoke-package-a-fingerprint-parity: ${label} is not a sha256: ${value}" >&2
    exit 1
  fi
}

json_get() {
  local expression="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); print(${expression})"
}

go_fingerprint() {
  local output
  output="$(go run ./cmd/areaflow project status-projection-authorization "${project_key}" --json)"
  json_get 'data["protected_path_fingerprint_sha256"]' <<<"${output}"
}

package_a_fingerprint() {
  local output
  set +e
  output="$(env -u AREAFLOW_DATABASE_URL AREAFLOW_PACKAGE_A_PROJECT_ROOT="${project_root}" AREAFLOW_PACKAGE_A_SOURCE_HASH="fixture-source-hash" bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
  local rc=$?
  set -e
  if [[ ${rc} -eq 0 ]]; then
    echo "smoke-package-a-fingerprint-parity: expected package-a packet to stay blocked without DB binding" >&2
    exit 1
  fi
  json_get 'data["protected_path_fingerprint_sha256"]' <<<"${output}"
}

write_fixture_files() {
  mkdir -p \
    "${project_root}/.areaflow" \
    "${project_root}/docs" \
    "${project_root}/scripts/dev_tools" \
    "${project_root}/workflow/residuals" \
    "${project_root}/workflow/templates" \
    "${project_root}/workflow/versions/v1-mvp/residuals" \
    "${project_root}/workflow/versions/v1-mvp/execution/_shared/nested" \
    "${artifact_root}"

  cat >"${project_root}/.areaflow/status.json" <<'JSON'
{"version":1,"summary":{"state":"legacy fixture"}}
JSON
  cat >"${project_root}/workflow/README.md" <<'EOF'
# Fingerprint Fixture Workflow
EOF
  cat >"${project_root}/scripts/dev_tools/cli.py" <<'EOF'
print("fixture cli")
EOF
  cat >"${project_root}/docs/README.md" <<'EOF'
# Fixture Docs
EOF
  cat >"${project_root}/workflow/templates/README.md" <<'EOF'
# Fixture Templates
EOF
  cat >"${project_root}/workflow/residuals/residuals.yaml" <<'EOF'
items:
  - id: global-fingerprint-fixture
    status: reference-only
    type: fixture
    title: Global fingerprint fixture
    source: workflow/residuals/residuals.yaml
    current_impact: proves fingerprint import
    executable_task: false
    promotion_required: false
version_residuals:
  - version: v1-mvp
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    status: mixed-blocked
    summary: Fingerprint fixture v1 metadata
EOF
  cat >"${project_root}/workflow/versions/v1-mvp/residuals/residuals.yaml" <<'EOF'
items:
  - id: v1-fingerprint-fixture
    status: blocked-decision
    type: fixture
    title: Version fingerprint fixture
    source: workflow/versions/v1-mvp/residuals/residuals.yaml
    current_impact: proves version residual import
    executable_task: false
    promotion_required: true
EOF
  cat >"${project_root}/workflow/versions/v1-mvp/execution/_shared/progress.json" <<'EOF'
{"tasks":{"fingerprint-fixture":{"status":"completed"}}}
EOF
  cat >"${project_root}/workflow/versions/v1-mvp/execution/_shared/nested/task.json" <<'EOF'
{"task":"fingerprint parity"}
EOF
}

write_config() {
  cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: Package A Fingerprint Fixture
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
  immutable_imports:
    - v1-mvp
EOF
}

write_fixture_files
write_config

git -C "${project_root}" init -q
git -C "${project_root}" config user.email "smoke-fingerprint@example.invalid"
git -C "${project_root}" config user.name "AreaFlow Fingerprint Smoke"
git -C "${project_root}" add .
git -C "${project_root}" commit -q -m "fingerprint fixture baseline"

echo "smoke-package-a-fingerprint-parity: migrate up"
go run ./cmd/areaflow migrate up >/dev/null

echo "smoke-package-a-fingerprint-parity: project add ${project_key}"
go run ./cmd/areaflow project add --config "${config_path}" >/dev/null

echo "smoke-package-a-fingerprint-parity: project import ${project_key}"
go run ./cmd/areaflow project import "${project_key}" >/dev/null

echo "smoke-package-a-fingerprint-parity: baseline parity"
go_hash_before="$(go_fingerprint)"
package_hash_before="$(package_a_fingerprint)"
assert_hex_sha256 "${go_hash_before}" "go baseline fingerprint"
assert_hex_sha256 "${package_hash_before}" "package-a baseline fingerprint"
assert_eq "${package_hash_before}" "${go_hash_before}" "baseline fingerprint parity"

echo "smoke-package-a-fingerprint-parity: target changes are excluded"
printf '{"version":2,"summary":{"state":"target changed"}}\n' >"${project_root}/.areaflow/status.json"
go_hash_after_target="$(go_fingerprint)"
package_hash_after_target="$(package_a_fingerprint)"
assert_eq "${package_hash_after_target}" "${go_hash_after_target}" "target-change fingerprint parity"
assert_eq "${go_hash_after_target}" "${go_hash_before}" "target-change exclusion"

echo "smoke-package-a-fingerprint-parity: protected path changes are included"
printf '\nprotected change\n' >>"${project_root}/workflow/README.md"
go_hash_after_protected="$(go_fingerprint)"
package_hash_after_protected="$(package_a_fingerprint)"
assert_eq "${package_hash_after_protected}" "${go_hash_after_protected}" "protected-change fingerprint parity"
assert_ne "${go_hash_after_protected}" "${go_hash_before}" "protected-change inclusion"

echo "smoke-package-a-fingerprint-parity: pass project=${project_key} fixture=${fixture_dir}"
