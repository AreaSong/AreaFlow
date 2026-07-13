#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-backup-restore-proof: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

project_key="${AREAFLOW_BACKUP_RESTORE_PROJECT_KEY:-areamatrix}"
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-backup-restore.XXXXXX")"
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
    echo "smoke-backup-restore-proof: refusing unsafe fixture dir: ${fixture_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  if [[ "${AREAFLOW_KEEP_FIXTURE:-0}" == "1" ]]; then
    echo "smoke-backup-restore-proof: keeping fixture at ${fixture_dir}"
    return
  fi
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-backup-restore-proof: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-backup-restore-proof: expected output to omit pattern: ${pattern}" >&2
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
    echo "smoke-backup-restore-proof: expected command to fail: $*" >&2
    exit 1
  fi
  assert_contains "${output}" "${pattern}"
}

json_get() {
  local payload="$1"
  local path="$2"

  JSON_PAYLOAD="${payload}" JSON_PATH="${path}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["JSON_PAYLOAD"])
path = os.environ["JSON_PATH"]

if path.startswith("len:"):
    value = data
    subpath = path[4:]
    if subpath:
        for part in subpath.split("."):
            value = value[part]
    print(len(value))
else:
    value = data
    for part in path.split("."):
        value = value[part]
    if isinstance(value, bool):
        print(str(value).lower())
    else:
        print(value)
PY
}

db_audit_row_counts() {
  psql "${AREAFLOW_DATABASE_URL}" -At <<'SQL'
SELECT
  (SELECT COUNT(*) FROM command_requests)::text || ':' ||
  (SELECT COUNT(*) FROM events)::text || ':' ||
  (SELECT COUNT(*) FROM audit_events)::text;
SQL
}

assert_equal() {
  local got="$1"
  local want="$2"
  local message="$3"

  if [[ "${got}" != "${want}" ]]; then
    echo "smoke-backup-restore-proof: ${message}: got ${got}, want ${want}" >&2
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
  echo "smoke-backup-restore-proof: skipping real AreaMatrix fingerprint check; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 to enable"
fi

mkdir -p "${project_root}/docs" "${project_root}/workflow" "${artifact_root}"

cat >"${project_root}/docs/README.md" <<'EOF'
# Backup Restore Proof Fixture Docs
EOF

cat >"${project_root}/workflow/README.md" <<'EOF'
# Backup Restore Proof Fixture Workflow
EOF

cat >"${config_path}" <<EOF
version: 1

project:
  id: ${project_key}
  name: AreaMatrix Backup Restore Fixture
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

backup_restore_facts=(
  backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata
  restore_dry_run_identifies_metadata_only_history_and_object_verifier_limits
  artifact_integrity_distinguishes_local_project_reference_external_and_object
  archive_preview_does_not_copy_upload_delete_or_gc_artifact_bytes
  retention_classes_and_accepted_exceptions_are_documented
  no_restore_apply_or_artifact_mutation_opened
)

backup_restore_fact_args=()
for fact in "${backup_restore_facts[@]}"; do
  backup_restore_fact_args+=(--fact "${fact}")
done

echo "smoke-backup-restore-proof: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-backup-restore-proof: project add ${config_path}"
go run ./cmd/areaflow project add --config "${config_path}"

echo "smoke-backup-restore-proof: seed artifact metadata"
local_artifact="${artifact_root}/backup-restore-local.json"
printf '{"fixture":"backup-restore","kind":"local"}\n' >"${local_artifact}"
local_artifact_sha="$(shasum -a 256 "${local_artifact}" | awk '{print $1}')"
local_artifact_size="$(wc -c <"${local_artifact}" | tr -d ' ')"
reference_sha="$(printf 'backup-restore-project-reference\n' | shasum -a 256 | awk '{print $1}')"
psql "${AREAFLOW_DATABASE_URL}" \
  -v "project_key=${project_key}" \
  -v "local_uri=${local_artifact}" \
  -v "local_sha=${local_artifact_sha}" \
  -v "local_size=${local_artifact_size}" \
  -v "reference_sha=${reference_sha}" >/dev/null <<'SQL'
WITH project_scope AS (
  SELECT id
  FROM projects
  WHERE project_key = :'project_key'
)
INSERT INTO artifacts (
  project_id, artifact_type, storage_backend, uri, source_path, sha256, size_bytes, content_type, metadata
)
SELECT id, 'backup_restore_local_fixture', 'local', :'local_uri', 'artifacts/backup-restore-local.json',
       :'local_sha', :'local_size'::bigint, 'application/json',
       '{"fixture_only":true,"retention_class":"run_evidence"}'::jsonb
FROM project_scope
UNION ALL
SELECT id, 'backup_restore_project_reference_fixture', 'project_reference', 'project://workflow/reference.json',
       'workflow/reference.json', :'reference_sha', 42, 'application/json',
       '{"fixture_only":true,"retention_class":"external_ref"}'::jsonb
FROM project_scope;
SQL

echo "smoke-backup-restore-proof: collect read-only backup/restore/artifact outputs"
backup_manifest_json="$(go run ./cmd/areaflow backup manifest --project "${project_key}" --json)"
restore_plan_json="$(go run ./cmd/areaflow backup restore-plan --project "${project_key}" --json)"
artifact_integrity_json="$(go run ./cmd/areaflow artifact integrity "${project_key}" --json)"
archive_preview_json="$(go run ./cmd/areaflow artifact archive-preview "${project_key}" \
  --idempotency-key "backup-restore-proof-smoke:archive-preview:${project_key}" \
  --reason "collect backup restore proof archive preview binding" \
  --json)"

backup_manifest_hash="$(json_get "${backup_manifest_json}" "manifest_hash")"
backup_manifest_status="$(json_get "${backup_manifest_json}" "status")"
backup_manifest_project_count="$(json_get "${backup_manifest_json}" "len:projects")"
backup_manifest_table_count="$(json_get "${backup_manifest_json}" "len:table_counts")"
restore_plan_status="$(json_get "${restore_plan_json}" "status")"
restore_plan_scope="$(json_get "${restore_plan_json}" "scope")"
restore_plan_project_key="$(json_get "${restore_plan_json}" "project_key")"
restore_plan_manifest_hash="$(json_get "${restore_plan_json}" "manifest_hash")"
restore_plan_item_count="$(json_get "${restore_plan_json}" "len:items")"
artifact_integrity_status="$(json_get "${artifact_integrity_json}" "status")"
artifact_integrity_checked_count="$(json_get "${artifact_integrity_json}" "checked_artifacts")"
artifact_integrity_failed_count="$(json_get "${artifact_integrity_json}" "failed_artifacts")"
archive_preview_status="$(json_get "${archive_preview_json}" "status")"
archive_preview_total_artifacts="$(json_get "${archive_preview_json}" "summary.total_artifacts")"
archive_preview_external_refs="$(json_get "${archive_preview_json}" "summary.external_refs")"
archive_preview_needs_policy="$(json_get "${archive_preview_json}" "summary.needs_policy")"
archive_preview_project_write_attempted="$(json_get "${archive_preview_json}" "project_write_attempted")"
archive_preview_storage_write_attempted="$(json_get "${archive_preview_json}" "storage_write_attempted")"
archive_preview_delete_attempted="$(json_get "${archive_preview_json}" "artifact_delete_attempted")"

assert_contains "${backup_manifest_json}" '"status": "ready"'
assert_contains "${backup_manifest_json}" '"scope": "project"'
assert_contains "${backup_manifest_json}" "\"project_key\": \"${project_key}\""
assert_contains "${backup_manifest_json}" "\"manifest_hash\": \"${backup_manifest_hash}\""
assert_contains "${restore_plan_json}" '"status": "needs_attention"'
assert_contains "${restore_plan_json}" '"scope": "project"'
assert_contains "${restore_plan_json}" "\"project_key\": \"${project_key}\""
assert_contains "${restore_plan_json}" "\"manifest_hash\": \"${backup_manifest_hash}\""
assert_contains "${artifact_integrity_json}" '"status": "warn"'
assert_contains "${artifact_integrity_json}" '"failed_artifacts": 0'
assert_contains "${archive_preview_json}" '"status": "needs_attention"'
assert_contains "${archive_preview_json}" '"external_refs": 1'
assert_contains "${archive_preview_json}" '"needs_policy": 0'
assert_contains "${archive_preview_json}" '"project_write_attempted": false'
assert_contains "${archive_preview_json}" '"storage_write_attempted": false'
assert_contains "${archive_preview_json}" '"artifact_delete_attempted": false'

backup_restore_binding_args=(
  --backup-manifest-hash "${backup_manifest_hash}"
  --backup-manifest-status "${backup_manifest_status}"
  --backup-manifest-project-count "${backup_manifest_project_count}"
  --backup-manifest-table-count "${backup_manifest_table_count}"
  --restore-plan-status "${restore_plan_status}"
  --restore-plan-scope "${restore_plan_scope}"
  --restore-plan-project-key "${restore_plan_project_key}"
  --restore-plan-manifest-hash "${restore_plan_manifest_hash}"
  --restore-plan-item-count "${restore_plan_item_count}"
  --artifact-integrity-status "${artifact_integrity_status}"
  --artifact-integrity-checked-count "${artifact_integrity_checked_count}"
  --artifact-integrity-failed-count "${artifact_integrity_failed_count}"
  --artifact-archive-preview-status "${archive_preview_status}"
  --artifact-archive-preview-total-artifacts "${archive_preview_total_artifacts}"
  --artifact-archive-preview-external-refs "${archive_preview_external_refs}"
  --artifact-archive-preview-needs-policy "${archive_preview_needs_policy}"
  --artifact-archive-preview-project-write-attempted "${archive_preview_project_write_attempted}"
  --artifact-archive-preview-storage-write-attempted "${archive_preview_storage_write_attempted}"
  --artifact-archive-preview-delete-attempted "${archive_preview_delete_attempted}"
)

echo "smoke-backup-restore-proof: backup restore proof rejects incomplete complete status"
assert_fails_contains \
  "complete backup restore proof missing required facts" \
  go run ./cmd/areaflow completion backup-restore-proof record "${project_key}" \
    --status complete \
    --fact backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata \
    --json

echo "smoke-backup-restore-proof: backup restore proof rejects missing output binding"
assert_fails_contains \
  "backup/restore/artifact output binding" \
  go run ./cmd/areaflow completion backup-restore-proof record "${project_key}" \
    --status complete \
    "${backup_restore_fact_args[@]}" \
    --summary "backup restore proof smoke review" \
    --evidence-uri "scripts/smoke-backup-restore-proof.sh#backup-restore" \
    --json

echo "smoke-backup-restore-proof: backup restore proof complete"
proof_json="$(go run ./cmd/areaflow completion backup-restore-proof record "${project_key}" \
  --status complete \
  "${backup_restore_fact_args[@]}" \
  --summary "backup restore proof smoke review" \
  --evidence-uri "scripts/smoke-backup-restore-proof.sh#backup-restore" \
  "${backup_restore_binding_args[@]}" \
  --idempotency-key "backup-restore-proof-smoke:${project_key}" \
  --reason "record backup restore proof smoke evidence" \
  --json)"
assert_contains "${proof_json}" '"proof_status": "complete"'
assert_contains "${proof_json}" '"decision": "allowed"'
assert_contains "${proof_json}" '"missing_facts": []'
assert_contains "${proof_json}" '"created": true'
assert_contains "${proof_json}" '"project_write_attempted": false'
assert_contains "${proof_json}" '"execution_write_attempted": false'
assert_contains "${proof_json}" '"database_restore_attempted": false'
assert_contains "${proof_json}" '"artifact_bytes_copied": false'
assert_contains "${proof_json}" '"artifact_bytes_deleted": false'
assert_contains "${proof_json}" '"artifact_bytes_uploaded": false'
assert_contains "${proof_json}" '"artifact_gc_attempted": false'
assert_contains "${proof_json}" '"commands_run": false'
assert_contains "${proof_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${proof_json}" '"backup_restore_evidence_binding_status": "pass"'
assert_contains "${proof_json}" "\"backup_manifest_hash\": \"${backup_manifest_hash}\""
assert_contains "${proof_json}" "\"restore_plan_status\": \"${restore_plan_status}\""
assert_contains "${proof_json}" "\"artifact_integrity_status\": \"${artifact_integrity_status}\""
assert_contains "${proof_json}" "\"artifact_archive_preview_status\": \"${archive_preview_status}\""

echo "smoke-backup-restore-proof: backup restore proof idempotent replay"
replay_json="$(go run ./cmd/areaflow completion backup-restore-proof record "${project_key}" \
  --status complete \
  "${backup_restore_fact_args[@]}" \
  --summary "backup restore proof smoke review" \
  --evidence-uri "scripts/smoke-backup-restore-proof.sh#backup-restore" \
  "${backup_restore_binding_args[@]}" \
  --idempotency-key "backup-restore-proof-smoke:${project_key}" \
  --reason "record backup restore proof smoke evidence" \
  --json)"
assert_contains "${replay_json}" '"created": false'

echo "smoke-backup-restore-proof: completion audit consumes backup restore proof but stays incomplete"
audit_counts_before="$(db_audit_row_counts)"
completion_json="$(go run ./cmd/areaflow completion audit --json)"
audit_counts_after="$(db_audit_row_counts)"
assert_equal "${audit_counts_after}" "${audit_counts_before}" "completion audit must not write command/event/audit rows"
assert_contains "${completion_json}" '"key": "E6_backup_restore_artifact_retention"'
assert_contains "${completion_json}" '"status": "complete"'
assert_contains "${completion_json}" '"backup_restore_gate_passed": true'
assert_contains "${completion_json}" '"backup_restore_proof_status": "complete"'
assert_contains "${completion_json}" '"latest_backup_restore_proof_evidence_uri": "scripts/smoke-backup-restore-proof.sh#backup-restore"'
assert_contains "${completion_json}" '"backup_restore_evidence_binding_status": "pass"'
assert_contains "${completion_json}" '"backup_restore_current_binding_bound": true'
assert_contains "${completion_json}" "\"backup_manifest_hash\": \"${backup_manifest_hash}\""
assert_contains "${completion_json}" "\"restore_plan_manifest_hash\": \"${backup_manifest_hash}\""
assert_contains "${completion_json}" "\"current_artifact_integrity_status\": \"${artifact_integrity_status}\""
assert_contains "${completion_json}" "\"current_artifact_archive_preview_external_refs\": ${archive_preview_external_refs}"
assert_contains "${completion_json}" "\"artifact_archive_preview_external_refs\": ${archive_preview_external_refs}"
assert_not_contains "${completion_json}" '"restore_dry_run_needs_attention"'
assert_not_contains "${completion_json}" '"metadata_only_history_not_closed"'
assert_contains "${completion_json}" '"project_write_attempted": false'
assert_contains "${completion_json}" '"execution_write_attempted": false'
assert_contains "${completion_json}" '"database_restore_attempted": false'
assert_contains "${completion_json}" '"artifact_bytes_copied": false'
assert_contains "${completion_json}" '"artifact_bytes_deleted": false'
assert_contains "${completion_json}" '"artifact_bytes_uploaded": false'
assert_contains "${completion_json}" '"artifact_gc_attempted": false'
assert_contains "${completion_json}" '"commands_run": false'
assert_contains "${completion_json}" '"area_matrix_protected_paths_touched": false'

echo "smoke-backup-restore-proof: completion audit blocks proof after artifact drift"
printf '{"fixture":"backup-restore","kind":"local","drift":true}\n' >"${local_artifact}"
drift_audit_counts_before="$(db_audit_row_counts)"
drift_completion_json="$(go run ./cmd/areaflow completion audit --json)"
drift_audit_counts_after="$(db_audit_row_counts)"
assert_equal "${drift_audit_counts_after}" "${drift_audit_counts_before}" "drifted completion audit must not write command/event/audit rows"
assert_contains "${drift_completion_json}" '"key": "E6_backup_restore_artifact_retention"'
assert_contains "${drift_completion_json}" '"backup_restore_current_binding_bound": false'
assert_contains "${drift_completion_json}" '"backup_restore_proof_current_binding_mismatch"'
assert_contains "${drift_completion_json}" '"artifact_integrity_status_changed"'
assert_contains "${drift_completion_json}" '"current_artifact_integrity_status_not_pass_or_warn"'
assert_contains "${drift_completion_json}" '"current_artifact_integrity_status": "fail"'
assert_contains "${drift_completion_json}" '"current_artifact_integrity_failed_count": 1'
assert_not_contains "${drift_completion_json}" '"backup_restore_gate_passed": true'

if [[ "${check_real_areamatrix}" == "1" ]]; then
  real_status_after="$(file_fingerprint "${real_areamatrix_status}")"
  real_readme_after="$(file_fingerprint "${real_areamatrix_readme}")"

  if [[ "${real_status_before}" != "${real_status_after}" ]]; then
    echo "smoke-backup-restore-proof: real AreaMatrix status changed unexpectedly: ${real_areamatrix_status}" >&2
    exit 1
  fi

  if [[ "${real_readme_before}" != "${real_readme_after}" ]]; then
    echo "smoke-backup-restore-proof: real AreaMatrix workflow README changed unexpectedly: ${real_areamatrix_readme}" >&2
    exit 1
  fi
fi

echo "smoke-backup-restore-proof: pass ${project_key} fixture=${fixture_dir}"
