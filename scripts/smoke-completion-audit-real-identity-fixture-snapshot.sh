#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

if [[ "${AREAFLOW_SMOKE_DOCKER_ISOLATED_DB_CREATED:-0}" != "1" && "${AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB:-0}" != "1" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: blocked; this smoke writes AreaFlow DB state and requires an isolated Docker smoke DB or AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB=1" >&2
  exit 1
fi

project_key="areamatrix"
config_path="${AREAFLOW_REAL_IDENTITY_CONFIG:-examples/areamatrix/areaflow.yaml}"
project_root="${AREAFLOW_REAL_IDENTITY_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"
status_path="${project_root}/.areaflow/status.json"
workflow_readme="${project_root}/workflow/README.md"
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

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-real-identity-fixture-snapshot: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-completion-audit-real-identity-fixture-snapshot: output unexpectedly contained pattern: ${pattern}" >&2
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

protected_path_fingerprint() {
  python3 - "${project_root}" "${protected_paths[@]}" <<'PY'
import hashlib
import os
import stat
import sys

root = os.path.abspath(sys.argv[1])
protected_paths = sys.argv[2:]


def rel(path):
    return os.path.relpath(path, root).replace(os.sep, "/")


def entry(path):
    info = os.lstat(path)
    mode = info.st_mode
    relative = rel(path)
    if stat.S_ISREG(mode):
        with open(path, "rb") as handle:
            content = handle.read()
        return f"{relative}\tfile\t{len(content)}\t{hashlib.sha256(content).hexdigest()}"
    if stat.S_ISDIR(mode):
        return f"{relative}\tdir"
    if stat.S_ISLNK(mode):
        return f"{relative}\tsymlink\t{os.readlink(path)}"
    return f"{relative}\tother\t{stat.filemode(mode)}\t{info.st_size}"


def walk(path, entries):
    entries.append(entry(path))
    if not stat.S_ISDIR(os.lstat(path).st_mode):
        return
    for name in sorted(os.listdir(path)):
        walk(os.path.join(path, name), entries)


entries = []
for protected_path in protected_paths:
    absolute = os.path.abspath(os.path.join(root, protected_path))
    try:
        os.lstat(absolute)
    except FileNotFoundError:
        entries.append(f"{rel(absolute)}\tmissing")
        continue
    walk(absolute, entries)

payload = "\n".join(entries).encode()
print(hashlib.sha256(payload).hexdigest())
PY
}

if [[ ! -d "${project_root}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: missing AreaMatrix project root: ${project_root}" >&2
  exit 1
fi

if [[ ! -f "${config_path}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: missing AreaFlow config: ${config_path}" >&2
  exit 1
fi

status_before="$(file_fingerprint "${status_path}")"
workflow_readme_before="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_before="$(protected_path_fingerprint)"
protected_path_git_status_before="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"

echo "smoke-completion-audit-real-identity-fixture-snapshot: migrate up"
go run ./cmd/areaflow migrate up

echo "smoke-completion-audit-real-identity-fixture-snapshot: project add ${config_path}"
add_output="$(go run ./cmd/areaflow project add --config "${config_path}")"
assert_contains "${add_output}" "registered ${project_key} ${project_root}"

echo "smoke-completion-audit-real-identity-fixture-snapshot: project import ${project_key}"
import_output="$(go run ./cmd/areaflow project import "${project_key}")"
assert_contains "${import_output}" "imported ${project_key}"

echo "smoke-completion-audit-real-identity-fixture-snapshot: seed fixture snapshot event"
inserted_event_id="$(psql "${AREAFLOW_DATABASE_URL}" -v "project_key=${project_key}" -qAt <<'SQL'
WITH target_project AS (
  SELECT id
  FROM projects
  WHERE project_key = :'project_key'
)
INSERT INTO events (project_id, event_type, severity, message, metadata)
SELECT id,
       'completion.audit_snapshot.recorded',
       'info',
       'Fixture completion audit snapshot seeded for real identity readiness guard',
       jsonb_build_object(
         'project_key', :'project_key',
         'status', 'recorded',
         'decision', 'allowed',
         'message', 'completion audit fixture snapshot seeded for real identity readiness guard',
         'audit_status', 'complete',
         'audit_scope', 'v1.0',
         'audit_hash', 'fixture-real-identity-audit-hash',
         'release_candidate_label', 'v1.0-fixture',
         'evidence_class', 'fixture',
         'evidence_uri', 'scripts/smoke-completion-audit-real-identity-fixture-snapshot.sh#fixture-snapshot',
         'proof_event_ids', '{}'::jsonb,
         'event_id', 0,
         'audit_event_id', 0,
         'idempotency_key', 'completion-audit-real-identity-fixture-snapshot:' || :'project_key',
         'project_write_attempted', false,
         'execution_write_attempted', false,
         'release_package_created', false,
         'publish_attempted', false,
         'restore_apply_attempted', false,
         'secret_resolved', false,
         'remote_worker_credentials_issued', false,
         'area_matrix_protected_paths_touched', false,
         'commands_run', false,
         'smoke_run_attempted', false,
         'worker_started', false,
         'metadata', jsonb_build_object(
           'summary', 'fixture snapshot intentionally cannot satisfy release_candidate closure',
           'fixture_snapshot', true,
           'release_candidate_snapshot', false,
           'readiness_guard', 'real_identity_fixture_only',
           'fixture_only_does_not_prove_release_readiness', true,
           'does_not_prove', jsonb_build_array(
             'fixture_only_does_not_prove_release_readiness',
             'real_identity_does_not_prove_package_a_apply',
             'real_identity_does_not_prove_real_100_percent',
             'real_100_percent',
             'release_candidate_closure',
             'release_candidate_readiness',
             'release_publish',
             'restore_apply'
           )
         )
       )
FROM target_project
RETURNING id;
SQL
)"

if [[ -z "${inserted_event_id}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: failed to seed fixture snapshot event" >&2
  exit 1
fi

echo "smoke-completion-audit-real-identity-fixture-snapshot: completion audit-snapshot readiness --json"
readiness_json="$(go run ./cmd/areaflow completion audit-snapshot readiness "${project_key}" --json)"
assert_contains "${readiness_json}" '"project": {'
assert_contains "${readiness_json}" '"key": "areamatrix"'
assert_contains "${readiness_json}" '"root": "'"${project_root}"'"'
assert_contains "${readiness_json}" '"kind": "product-repo"'
assert_contains "${readiness_json}" '"adapter": "areamatrix"'
assert_contains "${readiness_json}" '"workflow_profile": "areamatrix"'
assert_contains "${readiness_json}" '"default_branch": "main"'
assert_contains "${readiness_json}" '"status": "blocked"'
assert_contains "${readiness_json}" '"real_100_status": "blocked"'
assert_contains "${readiness_json}" '"readiness_scope": "completion_audit_evidence_only"'
assert_contains "${readiness_json}" '"claim_scope": "completion_audit_evidence_only"'
assert_contains "${readiness_json}" '"not_real_100": true'
assert_contains "${readiness_json}" '"evidence_only": true'
assert_contains "${readiness_json}" '"status_alone_is_not_completion": true'
assert_contains "${readiness_json}" '"release_candidate_decision": "requires_release_candidate_snapshot"'
assert_contains "${readiness_json}" '"real_100_blockers": ['
assert_contains "${readiness_json}" '"package_a_status_projection_apply_provenance_missing"'
assert_contains "${readiness_json}" '"release_candidate_snapshot_not_ready"'
assert_contains "${readiness_json}" '"has_snapshot": true'
assert_contains "${readiness_json}" '"required_class": "release_candidate"'
assert_contains "${readiness_json}" '"evidence_class": "fixture"'
assert_contains "${readiness_json}" '"release_candidate_label": "v1.0-fixture"'
assert_contains "${readiness_json}" '"key": "completion_audit_snapshot_fixture_only"'
assert_contains "${readiness_json}" '"category": "snapshot"'
assert_contains "${readiness_json}" '"fixture_snapshot": true'
assert_contains "${readiness_json}" '"release_candidate_snapshot": false'
assert_contains "${readiness_json}" '"closure": {'
assert_contains "${readiness_json}" '"ready_for_release_candidate_closure": false'
assert_contains "${readiness_json}" '"required_evidence_class": "release_candidate"'
assert_contains "${readiness_json}" '"snapshot_status": "fixture_only"'
assert_contains "${readiness_json}" '"snapshot": {'
assert_contains "${readiness_json}" '"status": "fixture_only"'
assert_contains "${readiness_json}" '"ready": false'
assert_contains "${readiness_json}" '"completion_audit_snapshot_fixture_only"'
assert_contains "${readiness_json}" '"fixture_only_does_not_prove_release_readiness": true'
assert_contains "${readiness_json}" '"does_not_prove": ['
assert_contains "${readiness_json}" '"fixture_only_does_not_prove_release_readiness"'
assert_contains "${readiness_json}" '"real_identity_does_not_prove_package_a_apply"'
assert_contains "${readiness_json}" '"real_identity_does_not_prove_real_100_percent"'
assert_contains "${readiness_json}" '"real_100_percent"'
assert_contains "${readiness_json}" '"release_candidate_closure"'
assert_contains "${readiness_json}" '"release_candidate_readiness"'
assert_contains "${readiness_json}" '"read_only": true'
assert_contains "${readiness_json}" '"project_write_attempted": false'
assert_contains "${readiness_json}" '"execution_write_attempted": false'
assert_contains "${readiness_json}" '"area_matrix_protected_paths_touched": false'
assert_contains "${readiness_json}" '"commands_run": false'
assert_contains "${readiness_json}" '"smoke_run_attempted": false'
assert_contains "${readiness_json}" '"worker_started": false'
assert_not_contains "${readiness_json}" '"completion_audit_snapshot_real_project_identity_missing"'
assert_not_contains "${readiness_json}" '"project_root_not_real_areamatrix"'
assert_not_contains "${readiness_json}" '"completion_audit_snapshot_release_candidate_present"'
assert_not_contains "${readiness_json}" '"ready_for_release_candidate_closure": true'
assert_not_contains "${readiness_json}" '"status": "ready"'
assert_not_contains "${readiness_json}" '"release_candidate_snapshot": true'
assert_not_contains "${readiness_json}" '"release_package_created": true'
assert_not_contains "${readiness_json}" '"publish_attempted": true'
assert_not_contains "${readiness_json}" '"restore_apply_attempted": true'

status_after="$(file_fingerprint "${status_path}")"
workflow_readme_after="$(file_fingerprint "${workflow_readme}")"
protected_path_fingerprint_after="$(protected_path_fingerprint)"
protected_path_git_status_after="$(git -C "${project_root}" status --short -- "${protected_paths[@]}")"

if [[ "${status_before}" != "${status_after}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: real AreaMatrix status changed unexpectedly: ${status_path}" >&2
  exit 1
fi

if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: real AreaMatrix workflow README changed unexpectedly: ${workflow_readme}" >&2
  exit 1
fi

if [[ "${protected_path_fingerprint_before}" != "${protected_path_fingerprint_after}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: real AreaMatrix protected path fingerprint changed unexpectedly" >&2
  exit 1
fi

if [[ "${protected_path_git_status_before}" != "${protected_path_git_status_after}" ]]; then
  echo "smoke-completion-audit-real-identity-fixture-snapshot: real AreaMatrix protected path git status changed unexpectedly" >&2
  exit 1
fi

echo "smoke-completion-audit-real-identity-fixture-snapshot: pass fixture-only real identity project=${project_key} event=${inserted_event_id} fixture_only_does_not_prove_release_readiness=true does_not_prove=real_100_percent,release_candidate_closure,release_candidate_readiness"
