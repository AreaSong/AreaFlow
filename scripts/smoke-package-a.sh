#!/usr/bin/env bash
set -euo pipefail

assert_contains() {
  local output="$1"
  local pattern="$2"

  if ! grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-package-a: expected output to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local output="$1"
  local pattern="$2"

  if grep -Fq -- "${pattern}" <<<"${output}"; then
    echo "smoke-package-a: expected output to omit pattern: ${pattern}" >&2
    exit 1
  fi
}

package_a_smoke_source_hash=""
package_a_unbound_test_source_hash="1111111111111111111111111111111111111111111111111111111111111111"
package_a_authorization_phrase="授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"
package_a_project_root="${AREAFLOW_PACKAGE_A_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}"

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

package_a_gate_exports() {
  python3 - <<'PY'
import json
import os
import shlex

payload = json.loads(os.environ["PACKAGE_A_GATE_JSON"])
target = payload.get("target", ".areaflow/status.json")
preimage = payload.get("target_preimage", {})
exports = {
    "gate_target": target,
    "gate_expected_before_exists": "true" if preimage.get("exists") else "false",
    "gate_expected_before_sha256": preimage.get("sha256", ""),
    "gate_expected_before_size": str(preimage.get("size_bytes", "")),
    "gate_source_hash": payload.get("source_hash", ""),
    "gate_schema_uri": "schemas/status-projection.schema.json",
    "gate_validator_preflight": payload.get("validator_preflight", ""),
    "gate_protected_path_check": payload.get("protected_path_check", ""),
    "gate_protected_path_fingerprint_sha256": payload.get("protected_path_fingerprint_sha256", ""),
    "gate_rollback_action": payload.get("rollback_action", ""),
    "gate_accept_preimage_schema": payload.get("accepted_preimage_schema_status", ""),
}
for key, value in exports.items():
    print(f"{key}={shlex.quote(str(value))}")
PY
}

run_package_a_apply_gate() {
  local approval_reason="$1"

  go run ./cmd/areaflow project status-projection-apply-gate areamatrix \
    --json \
    --target "${gate_target}" \
    --expected-before-exists "${gate_expected_before_exists}" \
    --expected-before-sha256 "${gate_expected_before_sha256}" \
    --expected-before-size "${gate_expected_before_size}" \
    --source-hash "${gate_source_hash}" \
    --schema-uri "${gate_schema_uri}" \
    --validator-preflight "${gate_validator_preflight}" \
    --protected-path-check "${gate_protected_path_check}" \
    --protected-path-fingerprint-sha256 "${gate_protected_path_fingerprint_sha256}" \
    --rollback-action "${gate_rollback_action}" \
    --accept-preimage-schema "${gate_accept_preimage_schema}" \
    --explicit-approval \
    --approval-actor smoke-package-a \
    --approval-reason "${approval_reason}"
}

assert_package_a_apply_gate_exact_approval() {
  local packet_json="$1"

  eval "$(PACKAGE_A_GATE_JSON="${packet_json}" package_a_gate_exports)"
  local status_before workflow_readme_before status_after workflow_readme_after
  status_before="$(file_fingerprint "${package_a_project_root}/.areaflow/status.json")"
  workflow_readme_before="$(file_fingerprint "${package_a_project_root}/workflow/README.md")"

  echo "smoke-package-a: status-projection-apply-gate --json non-exact approval reason stays blocked"
  non_exact_gate_json="$(run_package_a_apply_gate "approve fixture status projection apply")"
  echo "${non_exact_gate_json}"
  assert_contains "${non_exact_gate_json}" '"key": "areamatrix"'
  assert_contains "${non_exact_gate_json}" '"root": "'"${package_a_project_root}"'"'
  assert_contains "${non_exact_gate_json}" '"target_path": "'"${package_a_project_root}"'/.areaflow/status.json"'
  assert_contains "${non_exact_gate_json}" '"status": "blocked"'
  assert_contains "${non_exact_gate_json}" '"claim_scope": "package_a_status_projection_preflight_only"'
  assert_contains "${non_exact_gate_json}" '"not_real_100": true'
  assert_contains "${non_exact_gate_json}" '"decision": "no_go"'
  assert_contains "${non_exact_gate_json}" '"apply_command_eligible": false'
  assert_contains "${non_exact_gate_json}" '"approval_status": "missing_or_incomplete"'
  assert_contains "${non_exact_gate_json}" '"required_authorization_phrase": "'"${package_a_authorization_phrase}"'"'
  assert_contains "${non_exact_gate_json}" '"key": "approval_reason"'
  assert_contains "${non_exact_gate_json}" '"expected": "'"${package_a_authorization_phrase}"'"'
  assert_contains "${non_exact_gate_json}" '"actual": "approve fixture status projection apply"'
  assert_contains "${non_exact_gate_json}" '"approval_reason_missing_or_mismatch"'
  assert_contains "${non_exact_gate_json}" '"command_request_created": false'
  assert_contains "${non_exact_gate_json}" '"status_projection_written": false'
  assert_contains "${non_exact_gate_json}" '"project_write_attempted": false'
  assert_contains "${non_exact_gate_json}" '"execution_write_attempted": false'
  assert_contains "${non_exact_gate_json}" '"engine_call_attempted": false'
  assert_contains "${non_exact_gate_json}" '"read_only_gate": true'
  assert_contains "${non_exact_gate_json}" '"apply_open": false'
  assert_contains "${non_exact_gate_json}" '"apply_command_eligible_is_not_apply": true'
  assert_contains "${non_exact_gate_json}" '"requires_separate_apply_command": true'
  assert_contains "${non_exact_gate_json}" '"commands_run": false'
  assert_contains "${non_exact_gate_json}" '"network_used": false'

  echo "smoke-package-a: status-projection-apply-gate --json exact authorization phrase passes read-only gate only; does not approve or apply Package A"
  exact_gate_json="$(run_package_a_apply_gate "${package_a_authorization_phrase}")"
  echo "${exact_gate_json}"
  assert_contains "${exact_gate_json}" '"key": "areamatrix"'
  assert_contains "${exact_gate_json}" '"root": "'"${package_a_project_root}"'"'
  assert_contains "${exact_gate_json}" '"target_path": "'"${package_a_project_root}"'/.areaflow/status.json"'
  assert_contains "${exact_gate_json}" '"status": "pass"'
  assert_contains "${exact_gate_json}" '"claim_scope": "package_a_status_projection_preflight_only"'
  assert_contains "${exact_gate_json}" '"not_real_100": true'
  assert_contains "${exact_gate_json}" '"decision": "go"'
  assert_contains "${exact_gate_json}" '"apply_command_eligible": true'
  assert_contains "${exact_gate_json}" '"approval_status": "approved"'
  assert_contains "${exact_gate_json}" '"required_authorization_phrase": "'"${package_a_authorization_phrase}"'"'
  assert_contains "${exact_gate_json}" '"key": "approval_reason"'
  assert_contains "${exact_gate_json}" '"expected": "'"${package_a_authorization_phrase}"'"'
  assert_contains "${exact_gate_json}" '"actual": "'"${package_a_authorization_phrase}"'"'
  assert_not_contains "${exact_gate_json}" '"approval_reason_missing_or_mismatch"'
  assert_contains "${exact_gate_json}" '"command_request_created": false'
  assert_contains "${exact_gate_json}" '"status_projection_written": false'
  assert_contains "${exact_gate_json}" '"project_write_attempted": false'
  assert_contains "${exact_gate_json}" '"execution_write_attempted": false'
  assert_contains "${exact_gate_json}" '"engine_call_attempted": false'
  assert_contains "${exact_gate_json}" '"read_only_gate": true'
  assert_contains "${exact_gate_json}" '"apply_open": false'
  assert_contains "${exact_gate_json}" '"apply_command_eligible_is_not_apply": true'
  assert_contains "${exact_gate_json}" '"requires_separate_apply_command": true'
  assert_contains "${exact_gate_json}" '"commands_run": false'
  assert_contains "${exact_gate_json}" '"network_used": false'

  status_after="$(file_fingerprint "${package_a_project_root}/.areaflow/status.json")"
  workflow_readme_after="$(file_fingerprint "${package_a_project_root}/workflow/README.md")"
  if [[ "${status_before}" != "${status_after}" ]]; then
    echo "smoke-package-a: status-projection-apply-gate changed AreaMatrix status unexpectedly" >&2
    exit 1
  fi
  if [[ "${workflow_readme_before}" != "${workflow_readme_after}" ]]; then
    echo "smoke-package-a: status-projection-apply-gate changed AreaMatrix workflow README unexpectedly" >&2
    exit 1
  fi
}

echo "smoke-package-a: package-a-readiness"
set +e
readiness_output="$(bash scripts/audit-package-a-readiness.sh 2>&1)"
readiness_rc=$?
set -e
echo "${readiness_output}"

if [[ ${readiness_rc} -ne 0 ]]; then
  assert_contains "${readiness_output}" "package-a-readiness: blocked"
  assert_contains "${readiness_output}" "status_projection_state=legacy_needs_package_a"
  assert_contains "${readiness_output}" "target_preimage_exists=true"
  assert_contains "${readiness_output}" "target_preimage_sha256="
  assert_contains "${readiness_output}" "target_preimage_size_bytes="
  assert_contains "${readiness_output}" "blocked protected paths are not clean"
  assert_contains "${readiness_output}" "protected_path_rule_count="
  assert_contains "${readiness_output}" "dirty_path_count="

  echo "smoke-package-a: package-a-authorization-packet --json fail-closed"
  set +e
  authorization_json="$(bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
  authorization_rc=$?
  set -e
  echo "${authorization_json}"
  if [[ ${authorization_rc} -eq 0 ]]; then
    echo "smoke-package-a: expected authorization packet to stay blocked while readiness is blocked" >&2
    exit 1
  fi
  assert_contains "${authorization_json}" '"status": "blocked_needs_preflight_review"'
  assert_contains "${authorization_json}" '"schema_state": "legacy_needs_package_a"'
  assert_contains "${authorization_json}" '"accepted_preimage_schema_status": "legacy"'
  assert_contains "${authorization_json}" '"target_preimage": {'
  assert_contains "${authorization_json}" '"exists": true'
  assert_contains "${authorization_json}" '"sha256": "'
  assert_contains "${authorization_json}" '"size_bytes": '
  assert_contains "${authorization_json}" '"apply_gate_arguments": ['
  assert_contains "${authorization_json}" '"--target .areaflow/status.json"'
  assert_contains "${authorization_json}" '"--expected-before-exists true"'
  assert_contains "${authorization_json}" '"--expected-before-size '
  assert_contains "${authorization_json}" '"--schema-uri schemas/status-projection.schema.json"'
  assert_contains "${authorization_json}" '"--validator-preflight python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json '
  assert_contains "${authorization_json}" '"--protected-path-check git -C '
  assert_contains "${authorization_json}" '"--protected-path-fingerprint-sha256 '
  assert_contains "${authorization_json}" '"protected_path_fingerprint_sha256": "'
  assert_contains "${authorization_json}" '"protected_path_fingerprint_status": "captured"'
  assert_contains "${authorization_json}" '"--rollback-action restore the captured preimage bytes for .areaflow/status.json"'
  assert_contains "${authorization_json}" '"--expected-before-sha256 '
  assert_contains "${authorization_json}" '"--accept-preimage-schema legacy"'
  assert_contains "${authorization_json}" '"missing_apply_gate_arguments": ['
  assert_contains "${authorization_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'
  assert_contains "${authorization_json}" '"post_authorization_required_arguments": ['
  assert_contains "${authorization_json}" '"--explicit-approval"'
  assert_contains "${authorization_json}" '"source_hash_status": "missing"'
  assert_contains "${authorization_json}" '"status": "not_attempted_preflight_blocked"'
  assert_contains "${authorization_json}" '"protected_path_state": "dirty"'
  assert_contains "${authorization_json}" '"protected_path_rule_count": '
  assert_contains "${authorization_json}" '"dirty_path_count": '
  assert_contains "${authorization_json}" '"dirty_review_status": "required"'
  assert_contains "${authorization_json}" '"allowed_writes": ['
  assert_contains "${authorization_json}" '".areaflow/status.json"'
  assert_contains "${authorization_json}" '"durable_allowed_writes": ['
  assert_contains "${authorization_json}" '"transient_write_paths": ['
  assert_contains "${authorization_json}" '".areaflow/.status.json.tmp-*"'
  assert_contains "${authorization_json}" '".areaflow/.status.json.rollback-*"'
  assert_contains "${authorization_json}" '"transient_write_scope": "same-directory atomic status projection replace and rollback compensation only"'
  assert_contains "${authorization_json}" '"transient_writes_cleanup_required": true'
  assert_contains "${authorization_json}" '"transient_writes_durable_authorization": false'
  assert_contains "${authorization_json}" '"dirty_review_required": true'
  assert_contains "${authorization_json}" '"modifies_areamatrix": false'
  assert_contains "${authorization_json}" '"applies_status_projection": false'
  assert_contains "${authorization_json}" '"allows_transient_status_projection_temp_files": true'
  assert_contains "${authorization_json}" '"allows_extra_durable_files": false'
  assert_contains "${authorization_json}" '"allows_shim_files": false'
  assert_contains "${authorization_json}" '"allows_workflow_versions": false'
  assert_contains "${authorization_json}" '"allows_task_loop_forwarding": false'
  assert_contains "${authorization_json}" '"required_authorization_phrase": "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
  assert_contains "${authorization_json}" '"user_authorization_phrase": "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
  assert_not_contains "${authorization_json}" '"status": "ready_for_narrow_status_projection_authorization"'

  dirty_hash="$(awk -F= '/dirty_output_sha256=/{print $2; exit}' <<<"${readiness_output}")"
  if [[ -n "${dirty_hash}" ]]; then
    echo "smoke-package-a: package-a-authorization-packet --json dirty hash mismatch"
    set +e
    mismatch_json="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=0000000000000000000000000000000000000000000000000000000000000000 AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
    mismatch_rc=$?
    set -e
    echo "${mismatch_json}"
    if [[ ${mismatch_rc} -eq 0 ]]; then
      echo "smoke-package-a: expected dirty hash mismatch to stay blocked" >&2
      exit 1
    fi
    assert_contains "${mismatch_json}" '"status": "blocked_needs_preflight_review"'
    assert_contains "${mismatch_json}" '"dirty_review_status": "mismatch"'
    assert_contains "${mismatch_json}" '"protected_path_rule_count": '
    assert_contains "${mismatch_json}" '"dirty_path_count": '
    assert_contains "${mismatch_json}" '"dirty_review_required": true'
    assert_contains "${mismatch_json}" '"target_preimage": {'
    assert_contains "${mismatch_json}" '"--accept-preimage-schema legacy"'
    assert_contains "${mismatch_json}" '"--rollback-action restore the captured preimage bytes for .areaflow/status.json"'
    assert_contains "${mismatch_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'

    echo "smoke-package-a: package-a-readiness exact dirty hash reviewed"
    reviewed_readiness_output="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${dirty_hash}" AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a bash scripts/audit-package-a-readiness.sh 2>&1)"
    echo "${reviewed_readiness_output}"
    assert_contains "${reviewed_readiness_output}" "dirty_review_status=accepted"
    assert_contains "${reviewed_readiness_output}" "dirty_reviewer=smoke-package-a"
    assert_contains "${reviewed_readiness_output}" "protected_path_rule_count="
    assert_contains "${reviewed_readiness_output}" "dirty_path_count="
    assert_contains "${reviewed_readiness_output}" "target_preimage_exists=true"
    assert_contains "${reviewed_readiness_output}" "target_preimage_sha256="
    assert_contains "${reviewed_readiness_output}" "package-a-readiness: pass"

    echo "smoke-package-a: package-a-authorization-packet --json exact dirty hash reviewed without explicit source hash"
    set +e
    reviewed_authorization_json="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${dirty_hash}" AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
    reviewed_authorization_rc=$?
    set -e
    echo "${reviewed_authorization_json}"
    if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
      if [[ ${reviewed_authorization_rc} -ne 0 ]]; then
        echo "smoke-package-a: expected database-backed source hash binding to make packet ready" >&2
        exit 1
      fi
      assert_contains "${reviewed_authorization_json}" '"status": "ready_for_narrow_status_projection_authorization"'
      assert_contains "${reviewed_authorization_json}" '"source_hash_status": "bound_to_latest_import_snapshot"'
      assert_contains "${reviewed_authorization_json}" '"status": "bound"'
      assert_contains "${reviewed_authorization_json}" '"writes_db": true'
      assert_contains "${reviewed_authorization_json}" '"modifies_areamatrix": false'
      assert_contains "${reviewed_authorization_json}" '"status_before": "'
      assert_contains "${reviewed_authorization_json}" '"status_after": "'
      assert_contains "${reviewed_authorization_json}" '"missing_apply_gate_arguments": []'
      assert_contains "${reviewed_authorization_json}" '"--approval-reason 授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
      assert_contains "${reviewed_authorization_json}" '"--source-hash '
      assert_contains "${reviewed_authorization_json}" '"--protected-path-fingerprint-sha256 '
      assert_contains "${reviewed_authorization_json}" '"protected_path_fingerprint_sha256": "'
      assert_contains "${reviewed_authorization_json}" '"approves_write": false'
      package_a_smoke_source_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["source_hash"])' <<<"${reviewed_authorization_json}")"
    else
      if [[ ${reviewed_authorization_rc} -eq 0 ]]; then
        echo "smoke-package-a: expected missing database-backed source hash to stay blocked" >&2
        exit 1
      fi
      assert_contains "${reviewed_authorization_json}" '"status": "blocked_needs_authoritative_source_hash"'
      assert_contains "${reviewed_authorization_json}" '"source_hash_status": "missing"'
      assert_contains "${reviewed_authorization_json}" '"status": "not_attempted_missing_database_url"'
      assert_contains "${reviewed_authorization_json}" '"missing_apply_gate_arguments": ['
      assert_contains "${reviewed_authorization_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'
      assert_contains "${reviewed_authorization_json}" '"--protected-path-fingerprint-sha256 '
      assert_contains "${reviewed_authorization_json}" '"protected_path_fingerprint_sha256": "'
    fi
    assert_contains "${reviewed_authorization_json}" '"dirty_review_status": "accepted"'
    assert_contains "${reviewed_authorization_json}" '"protected_path_rule_count": '
    assert_contains "${reviewed_authorization_json}" '"dirty_path_count": '
    assert_contains "${reviewed_authorization_json}" '"dirty_review_required": false'
    assert_contains "${reviewed_authorization_json}" '"durable_allowed_writes": ['
    assert_contains "${reviewed_authorization_json}" '"transient_write_paths": ['
    assert_contains "${reviewed_authorization_json}" '"transient_writes_cleanup_required": true'
    assert_contains "${reviewed_authorization_json}" '"transient_writes_durable_authorization": false'

    if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
      echo "smoke-package-a: package-a-source-hash --json exact dirty hash reviewed"
      source_hash_json="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${dirty_hash}" AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a bash scripts/audit-package-a-source-hash.sh --json)"
      echo "${source_hash_json}"
      assert_contains "${source_hash_json}" '"status": "ready_for_package_a_packet_binding"'
      assert_contains "${source_hash_json}" '"source_hash": "'
      assert_contains "${source_hash_json}" '"protected_path_fingerprint_sha256": "'
      assert_contains "${source_hash_json}" '"source_hash_env": "AREAFLOW_PACKAGE_A_SOURCE_HASH='
      assert_contains "${source_hash_json}" '"dirty_review_status": "accepted"'
      assert_contains "${source_hash_json}" '"writes_db": true'
      assert_contains "${source_hash_json}" '"modifies_areamatrix": false'
      assert_contains "${source_hash_json}" '"status_projection_changed": false'
      assert_contains "${source_hash_json}" '"workflow_readme_changed": false'
      assert_contains "${source_hash_json}" '"applies_status_projection": false'
      direct_source_hash="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["source_hash"])' <<<"${source_hash_json}")"
      if [[ "${direct_source_hash}" != "${package_a_smoke_source_hash}" ]]; then
        echo "smoke-package-a: expected direct source hash to match authorization binding" >&2
        exit 1
      fi
    fi

    if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
      echo "smoke-package-a: package-a-authorization-packet --json exact dirty hash reviewed with matching source hash"
      reviewed_authorization_json="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${dirty_hash}" AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a AREAFLOW_PACKAGE_A_SOURCE_HASH="${package_a_smoke_source_hash}" bash scripts/audit-package-a-authorization-packet.sh --json)"
      echo "${reviewed_authorization_json}"
      assert_contains "${reviewed_authorization_json}" '"status": "ready_for_narrow_status_projection_authorization"'
      assert_contains "${reviewed_authorization_json}" '"accepted_preimage_schema_status": "legacy"'
      assert_contains "${reviewed_authorization_json}" '"target_preimage": {'
      assert_contains "${reviewed_authorization_json}" '"--expected-before-exists true"'
      assert_contains "${reviewed_authorization_json}" '"--source-hash '"${package_a_smoke_source_hash}"'"'
      assert_contains "${reviewed_authorization_json}" '"--protected-path-fingerprint-sha256 '
      assert_contains "${reviewed_authorization_json}" '"protected_path_fingerprint_sha256": "'
      assert_contains "${reviewed_authorization_json}" '"provided_source_hash": "'"${package_a_smoke_source_hash}"'"'
      assert_contains "${reviewed_authorization_json}" '"source_hash_status": "bound_to_latest_import_snapshot"'
      assert_contains "${reviewed_authorization_json}" '"missing_apply_gate_arguments": []'
      assert_contains "${reviewed_authorization_json}" '"post_authorization_required_arguments": ['
      assert_contains "${reviewed_authorization_json}" '"--approval-actor <authorized Package A approver>"'
      assert_contains "${reviewed_authorization_json}" '"--approval-reason 授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
      assert_contains "${reviewed_authorization_json}" '"--accept-preimage-schema legacy"'
      assert_contains "${reviewed_authorization_json}" '"dirty_review_status": "accepted"'
      assert_contains "${reviewed_authorization_json}" '"dirty_review_required": false'
      assert_contains "${reviewed_authorization_json}" '"dirty_reviewer": "smoke-package-a"'
      assert_contains "${reviewed_authorization_json}" '"writes_db": true'
      assert_contains "${reviewed_authorization_json}" '"modifies_areamatrix": false'
      assert_contains "${reviewed_authorization_json}" '"applies_status_projection": false'
      assert_contains "${reviewed_authorization_json}" '"approves_write": false'
      assert_package_a_apply_gate_exact_approval "${reviewed_authorization_json}"

      echo "smoke-package-a: package-a-authorization-packet --json source hash mismatch"
      set +e
      stale_authorization_json="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${dirty_hash}" AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a AREAFLOW_PACKAGE_A_SOURCE_HASH="${package_a_unbound_test_source_hash}" bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
      stale_authorization_rc=$?
      set -e
      echo "${stale_authorization_json}"
      if [[ ${stale_authorization_rc} -eq 0 ]]; then
        echo "smoke-package-a: expected stale source hash to stay blocked" >&2
        exit 1
      fi
      assert_contains "${stale_authorization_json}" '"status": "blocked_needs_authoritative_source_hash"'
      assert_contains "${stale_authorization_json}" '"source_hash_status": "mismatch_latest_import_snapshot"'
      assert_contains "${stale_authorization_json}" '"status": "mismatch"'
      assert_contains "${stale_authorization_json}" '"missing_apply_gate_arguments": ['
      assert_contains "${stale_authorization_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'
    else
      echo "smoke-package-a: package-a-authorization-packet --json explicit source hash without database remains blocked"
      set +e
      unbound_authorization_json="$(AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256="${dirty_hash}" AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=smoke-package-a AREAFLOW_PACKAGE_A_SOURCE_HASH="${package_a_unbound_test_source_hash}" bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
      unbound_authorization_rc=$?
      set -e
      echo "${unbound_authorization_json}"
      if [[ ${unbound_authorization_rc} -eq 0 ]]; then
        echo "smoke-package-a: expected unbound source hash to stay blocked without database" >&2
        exit 1
      fi
      assert_contains "${unbound_authorization_json}" '"status": "blocked_needs_authoritative_source_hash"'
      assert_contains "${unbound_authorization_json}" '"provided_source_hash": "'"${package_a_unbound_test_source_hash}"'"'
      assert_contains "${unbound_authorization_json}" '"source_hash_status": "provided_unbound"'
      assert_contains "${unbound_authorization_json}" '"status": "not_attempted_missing_database_url"'
      assert_contains "${unbound_authorization_json}" '"missing_apply_gate_arguments": ['
      assert_contains "${unbound_authorization_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'
    fi
  fi

  echo "smoke-package-a: package-a-dirty-review"
  dirty_review_output="$(bash scripts/audit-package-a-dirty-review.sh)"
  echo "${dirty_review_output}"
  assert_contains "${dirty_review_output}" "package-a-dirty-review: status=dirty"
  assert_contains "${dirty_review_output}" "protected_path_rule_count="
  assert_contains "${dirty_review_output}" "dirty_path_count="
  assert_contains "${dirty_review_output}" "dirty_output_sha256="
  assert_contains "${dirty_review_output}" "reviewed_hash_input=AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<dirty_output_sha256>"
  assert_contains "${dirty_review_output}" "reviewer_input=AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>"

  echo "smoke-package-a: ok fail-closed"
  exit 0
fi

assert_contains "${readiness_output}" "protected paths clean"
assert_contains "${readiness_output}" "protected_path_rule_count="
assert_contains "${readiness_output}" "dirty_path_count=0"

echo "smoke-package-a: package-a-authorization-packet --json ready"
set +e
authorization_json="$(env -u AREAFLOW_DATABASE_URL bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
authorization_rc=$?
set -e
echo "${authorization_json}"
if [[ ${authorization_rc} -eq 0 ]]; then
  echo "smoke-package-a: expected missing source hash to stay blocked before ready packet" >&2
  exit 1
fi
assert_contains "${authorization_json}" '"status": "blocked_needs_authoritative_source_hash"'
assert_contains "${authorization_json}" '"source_hash_status": "missing"'
assert_contains "${authorization_json}" '"missing_apply_gate_arguments": ['
assert_contains "${authorization_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'
assert_contains "${authorization_json}" '"--protected-path-fingerprint-sha256 '
assert_contains "${authorization_json}" '"protected_path_fingerprint_sha256": "'

if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-package-a: package-a-authorization-packet --json ready with database-bound source hash"
  authorization_json="$(bash scripts/audit-package-a-authorization-packet.sh --json)"
  echo "${authorization_json}"
  assert_contains "${authorization_json}" '"status": "ready_for_narrow_status_projection_authorization"'
  assert_contains "${authorization_json}" '"target_preimage": {'
  assert_contains "${authorization_json}" '"allowed_writes": ['
  assert_contains "${authorization_json}" '".areaflow/status.json"'
  assert_contains "${authorization_json}" '"durable_allowed_writes": ['
  assert_contains "${authorization_json}" '"transient_write_paths": ['
  assert_contains "${authorization_json}" '".areaflow/.status.json.tmp-*"'
  assert_contains "${authorization_json}" '".areaflow/.status.json.rollback-*"'
  assert_contains "${authorization_json}" '"transient_writes_cleanup_required": true'
  assert_contains "${authorization_json}" '"transient_writes_durable_authorization": false'
  assert_contains "${authorization_json}" '"--source-hash '
  assert_contains "${authorization_json}" '"--protected-path-fingerprint-sha256 '
  assert_contains "${authorization_json}" '"protected_path_fingerprint_sha256": "'
  assert_contains "${authorization_json}" '"source_hash_status": "bound_to_latest_import_snapshot"'
  assert_contains "${authorization_json}" '"missing_apply_gate_arguments": []'
  assert_contains "${authorization_json}" '"--approval-reason 授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"'
  assert_contains "${authorization_json}" '"dirty_review_status": "clean"'
  assert_contains "${authorization_json}" '"dirty_review_required": false'
  assert_contains "${authorization_json}" '"writes_db": true'
  assert_contains "${authorization_json}" '"modifies_areamatrix": false'
  assert_contains "${authorization_json}" '"applies_status_projection": false'
  assert_contains "${authorization_json}" '"approves_write": false'
  assert_contains "${authorization_json}" '"allows_transient_status_projection_temp_files": true'
  assert_contains "${authorization_json}" '"allows_extra_durable_files": false'
  assert_contains "${authorization_json}" '"allows_shim_files": false'
  assert_package_a_apply_gate_exact_approval "${authorization_json}"
else
  echo "smoke-package-a: package-a-authorization-packet --json ready with unbound source hash remains blocked"
  set +e
  authorization_json="$(AREAFLOW_PACKAGE_A_SOURCE_HASH="${package_a_unbound_test_source_hash}" bash scripts/audit-package-a-authorization-packet.sh --json 2>&1)"
  authorization_rc=$?
  set -e
  echo "${authorization_json}"
  if [[ ${authorization_rc} -eq 0 ]]; then
    echo "smoke-package-a: expected unbound source hash to stay blocked without database" >&2
    exit 1
  fi
  assert_contains "${authorization_json}" '"status": "blocked_needs_authoritative_source_hash"'
  assert_contains "${authorization_json}" '"provided_source_hash": "'"${package_a_unbound_test_source_hash}"'"'
  assert_contains "${authorization_json}" '"source_hash_status": "provided_unbound"'
  assert_contains "${authorization_json}" '"status": "not_attempted_missing_database_url"'
  assert_contains "${authorization_json}" '"missing_apply_gate_arguments": ['
  assert_contains "${authorization_json}" '"--source-hash <bound latest AreaFlow import snapshot hash>"'
  assert_contains "${authorization_json}" '"--protected-path-fingerprint-sha256 '
  assert_contains "${authorization_json}" '"protected_path_fingerprint_sha256": "'
fi

echo "smoke-package-a: shim authorization preflight"
if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
  bash scripts/smoke-shim-authorization-preflight.sh
else
  set +e
  shim_preflight_output="$(bash scripts/smoke-shim-authorization-preflight.sh 2>&1)"
  shim_preflight_rc=$?
  set -e
  echo "${shim_preflight_output}"
  if [[ ${shim_preflight_rc} -eq 0 ]]; then
    echo "smoke-package-a: expected shim authorization preflight to require AREAFLOW_DATABASE_URL" >&2
    exit 1
  fi
  assert_contains "${shim_preflight_output}" "smoke-shim-authorization-preflight: blocked; AREAFLOW_DATABASE_URL is required"
fi

echo "smoke-package-a: execution forwarding v1 readiness"
bash scripts/smoke-execution-forwarding-v1-readiness.sh

echo "smoke-package-a: ok"
