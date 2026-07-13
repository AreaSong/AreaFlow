#!/usr/bin/env bash
set -euo pipefail

schema="schemas/status-projection.schema.json"
validator="scripts/validate-status-projection-schema.py"
temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-status-schema.XXXXXX")"
temp_dir="$(cd "${temp_dir}" && pwd -P)"

case "${temp_dir}" in
  /tmp/*|/private/tmp/*|/var/folders/*|/private/var/folders/*) ;;
  *)
    echo "smoke-status-projection-schema: refusing unsafe temp dir: ${temp_dir}" >&2
    exit 1
    ;;
esac

cleanup() {
  rm -rf "${temp_dir}"
}
trap cleanup EXIT

assert_fails() {
  local path="$1"
  local expected="$2"
  local output

  if output="$(python3 "${validator}" "${schema}" "${path}" 2>&1)"; then
    echo "smoke-status-projection-schema: expected validation to fail for ${path}" >&2
    exit 1
  fi
  if ! grep -Fq -- "${expected}" <<<"${output}"; then
    echo "smoke-status-projection-schema: expected failure to mention ${expected}" >&2
    echo "${output}" >&2
    exit 1
  fi
}

valid="${temp_dir}/valid.json"
cat >"${valid}" <<'JSON'
{
  "schema_version": 1,
  "project_id": "areamatrix",
  "project_name": "AreaMatrix",
  "area_flow_url": "http://127.0.0.1:3847/projects/areamatrix",
  "cutover_phase": "import_mirror",
  "active_versions": [
    {
      "display_label": "v1-mvp",
      "version_kind": "workflow_version",
      "lifecycle_status": "imported",
      "rough_progress": {
        "percent": 100,
        "label": "637/637 v1 execution tasks completed",
        "blocked": false
      }
    }
  ],
  "last_synced_at": "2026-07-04T03:00:00Z",
  "source_snapshot_hash": "hash-fixture",
  "compatibility": {
    "shim_lifecycle_state": "not_installed",
    "offline_source": ".areaflow/status.json",
    "blocked_commands": [
      "./task-loop run",
      "promotion apply",
      "write execution"
    ]
  }
}
JSON

echo "smoke-status-projection-schema: valid projection"
python3 "${validator}" "${schema}" "${valid}" >/dev/null

missing_required="${temp_dir}/missing-required.json"
python3 - "${valid}" "${missing_required}" <<'PY'
import json
import sys
source, target = sys.argv[1], sys.argv[2]
data = json.load(open(source))
del data["source_snapshot_hash"]
json.dump(data, open(target, "w"))
PY
assert_fails "${missing_required}" "missing required property source_snapshot_hash"

extra_top_level="${temp_dir}/extra-top-level.json"
python3 - "${valid}" "${extra_top_level}" <<'PY'
import json
import sys
source, target = sys.argv[1], sys.argv[2]
data = json.load(open(source))
data["summary"] = {"legacy": True}
json.dump(data, open(target, "w"))
PY
assert_fails "${extra_top_level}" "unexpected property summary"

invalid_percent="${temp_dir}/invalid-percent.json"
python3 - "${valid}" "${invalid_percent}" <<'PY'
import json
import sys
source, target = sys.argv[1], sys.argv[2]
data = json.load(open(source))
data["active_versions"][0]["rough_progress"]["percent"] = 101
json.dump(data, open(target, "w"))
PY
assert_fails "${invalid_percent}" "expected <= 100"

extra_nested="${temp_dir}/extra-nested.json"
python3 - "${valid}" "${extra_nested}" <<'PY'
import json
import sys
source, target = sys.argv[1], sys.argv[2]
data = json.load(open(source))
data["compatibility"]["artifact_content"] = "forbidden"
json.dump(data, open(target, "w"))
PY
assert_fails "${extra_nested}" "unexpected property artifact_content"

echo "smoke-status-projection-schema: pass"
