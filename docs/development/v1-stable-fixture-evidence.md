# v1 Stable Fixture Evidence

Date: 2026-07-04

## Scope

This evidence covers the v1.0 stable platform fixture smoke. It runs the long `smoke-local.sh`
chain against a temporary AreaMatrix-like project root instead of the real AreaMatrix repository.

The fixture validates:

- Fresh PostgreSQL migration from an empty database.
- Project registration from a temporary `areaflow.yaml`.
- Repeated metadata import so v0.2 import diff history is available.
- Guarded fixture-only `.areaflow/status.json` projection.
- v0.3/v0.4 workflow authoring, gate, approval and cutover readiness paths.
- v0.5 runner preview and run-control safety boundaries.
- v0.6 worker register, lease/run-once and capability denial paths.
- v0.8 worker pool and schedule preview.
- v0.9 local service status contract and desktop service-control / notification / tray-menu gate CLI contracts.
- v1.0 operations readiness, support bundle metadata-only preview and migration ledger readiness CLI contracts.
- v1.0 backup manifest, restore dry-run, audit coverage, permission doctor, artifact integrity,
  adapter/profile/project-config conformance and release preview gate chain.
- Web/Desktop read-only observation can surface release final gate, evidence bundle, package preview,
  distribution preview, publish gate, publish approval preview and rollout plan preview without opening
  package, publish, tag, sign, upload, approval, rollout or release apply.
- Web/Desktop read-only observation can surface operations readiness without opening support export, telemetry
  upload, migration apply, service process control or managed ops.

## Smoke Entry

```bash
bash scripts/smoke-v1-stable-fixture.sh
```

Make targets:

```bash
make smoke-v1-stable-fixture
make smoke-docker-v1-stable-fixture
```

The script requires `AREAFLOW_DATABASE_URL`. It creates a temporary fixture project root and local
artifact store, then points `smoke-local.sh` at that fixture by setting:

```text
AREAFLOW_SMOKE_PROJECT=areamatrix-v1-fixture
AREAFLOW_SMOKE_CONFIG=<temporary fixture>/areaflow.yaml
AREAFLOW_SMOKE_WORKFLOW_VERSION=v1-stable-smoke-<timestamp>
AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1
```

`make smoke-docker-v1-stable-fixture` starts the Compose PostgreSQL service and, when `AREAFLOW_DATABASE_URL` is
not explicitly provided, creates an isolated temporary database for the smoke and drops only that database on exit.

## Safety Facts

The fixture is allowed to write only inside its temporary project root and temporary artifact store.
By default it does not read the real AreaMatrix projection files. Callers that intentionally want an
extra out-of-band guard can set `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`, which fingerprints these
real files before and after the run:

```text
/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
/Users/as/Ai-Project/project/AreaMatrix/workflow/README.md
```

When that opt-in guard is enabled, the run fails if either real AreaMatrix fingerprint changes.
`smoke-local.sh` refuses real AreaMatrix reads before `project add/import/summary/doctor` unless
`AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1` is set, and still refuses status projection apply by default. The
fixture wrapper only enables `AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1` after redirecting the project config to the
temporary root. It does not set `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1` or
`AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1`, so accidentally resolving to the real AreaMatrix root would
fail closed before reading or writing the real project.

The smoke does not open:

- Real AreaMatrix project writes.
- `workflow/README.md` controlled block writes.
- `workflow/versions/**/execution/**` writes.
- Real task execution.
- Codex CLI or engine execution.
- Secret resolution.
- Network access.
- Git checkpoint, tag, push, sign, upload or publish.
- Restore apply.
- Release exception apply.

## Latest Result

Latest run on 2026-07-04 15:16 CST:

```bash
make smoke-docker-v1-stable-fixture
```

Result:

```text
smoke-docker: creating isolated PostgreSQL database areaflow_smoke_20260706220149_15594
applied 000001_v0_1_core.sql
applied 000002_v0_3_command_requests.sql
applied 000003_v0_3_gate_results.sql
applied 000004_v0_3_approval_transition.sql
applied 000005_v0_5_runner_preview.sql
applied 000006_v0_6_worker_registry.sql
applied 000007_v0_8_scheduling_policy.sql
applied 000008_v0_3_workflow_item_links.sql
applied 000009_v1_boundary_foundation.sql
applied 000010_v1_status_projections.sql
applied 000011_v1_migration_ledger.sql
smoke-local: project status-projection-authorization areamatrix-v1-fixture
smoke-local: project status-projection-apply-packet areamatrix-v1-fixture
smoke-local: project status-projection-apply areamatrix-v1-fixture
smoke-local: project compatibility --json
smoke-local: project shim-preview --json
smoke-local: project shim-readiness --json
smoke-local: project shim-authorization --json
smoke-local: project shim-apply-packet before evidence --json
smoke-local: project shim-apply-gate before evidence --json
smoke-local: project shim-readiness-evidence real_areamatrix_readonly_smoke
smoke-local: project shim-readiness-evidence real_areamatrix_status_projection_schema
smoke-local: project shim-readiness-evidence areamatrix_dirty_worktree_review
smoke-local: project shim-readiness after evidence --json
smoke-local: project shim-apply-packet after evidence --json
smoke-local: desktop service-control-gate
smoke-local: desktop notification-gate
smoke-local: desktop tray-menu-gate
smoke-local: ops migration ledger readiness
smoke-local: support bundle preview
smoke-local: ops readiness
smoke-local: adapter/profile conformance
smoke-local: ops smoke proof record
smoke-local: ops readiness after smoke proof
smoke-local: completion audit after ops smoke proof
smoke-local: ok
smoke-v1-stable-fixture: pass areamatrix-v1-fixture fixture=/private/var/folders/.../areaflow-v1-stable.qs3sO0
smoke-docker: ok
smoke-docker: dropping isolated PostgreSQL database areaflow_smoke_20260706220149_15594
```

The 22:01 run used an isolated PostgreSQL database created by `scripts/smoke-docker.sh` and cleaned it up after
completion. It also proved the long fixture chain now enters status projection through
`status-projection-authorization -> status-projection-apply-packet -> status-projection-apply`, with
`AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1` enabled only after the wrapper points the project config at the temporary
fixture root. The same long chain now also covers compatibility/shim preview, shim readiness, shim authorization,
shim apply packet/gate before evidence, fixture-only shim readiness evidence recording, and a complete
`shim-apply-packet` review that becomes `ready_for_future_apply_command` while still reporting no command request,
project write, execution write, task-loop forwarding, status projection write or AreaMatrix file modification.
The fixture ops smoke proof now clears the fixture `ops readiness` blocker, while `completion audit` remains scoped
to the canonical `areamatrix` target and therefore does not consume `areamatrix-v1-fixture` proof records as real
AreaMatrix E7 evidence.
It also proves adapter/profile conformance now verifies the AreaMatrix workflow profile item state contract,
transition contract, required hard rules, artifact policy, cutover policy and active project config snapshot. Focused
conformance tests now also cover `plugin_seed_catalog_contract`, `plugin_manifest_draft_contract` and
`plugin_no_execution_boundary`, proving the v1.0 plugin / marketplace surface is limited to built-in / seed metadata,
manifest draft lint and unknown plugin execution deferral. The conformance chain only passes when
`profile_item_state_contract`, `profile_transition_contract`, `profile_hard_rule_contract`,
`profile_permission_policy_contract`, `profile_artifact_policy_contract`, `profile_cutover_policy_contract`,
`project_config_policy`, `plugin_seed_catalog_contract`, `plugin_manifest_draft_contract` and
`plugin_no_execution_boundary` are present, and when `areaflow.yaml` keeps protocol v1, the migration strategy, safe
capabilities, `.areaflow/status.json` write path, execution/DB/.areamatrix forbidden paths, dangerous command
denylist, single-task scheduling, disabled engine profiles and disabled workflow README human summary aligned with the
current AreaMatrix safety baseline.
The same run also proves `areaflow desktop service-control-gate --json` remains in the long fixture chain and keeps
start/stop/restart disabled with no process control, command, approval, audit, worker scheduling, workflow
execution, project write or secret resolution. It also proves `areaflow desktop notification-gate --json` and
`areaflow desktop tray-menu-gate --json` remain in the long fixture chain while keeping event stream opening,
notification requests, tray/menu creation, OS integration, service control, command, approval, audit, worker
scheduling, workflow execution, project write and secret resolution closed. The run also proved the v1.0 operations
readiness CLI smoke
assertions still run inside the long fixture chain,
that the chain records `v1_stable_fixture_smoke` proof only after all checks pass, and that
`areaflow completion audit --json` consumes the proof by removing both `fresh_local_ops_smoke_missing` and
`full_migration_ledger_missing`. An earlier
19:23 run exposed that `scripts/smoke-local.sh` expected a passing v0.2 verification bundle while
only performing one import in a fresh database. The smoke was corrected to perform two imports,
matching the existing fixture and AreaMatrix read-only smoke behavior and satisfying the import diff
history requirement.

An intermediate 03:29 Docker run against the previous persistent default database failed because that database
contained stale fixture project roots from older temporary directories. `scripts/smoke-docker.sh` now avoids that
class of false failure by using an isolated database by default unless `AREAFLOW_DATABASE_URL` is explicitly set.

Guard replay on 2026-07-04 10:50 CST:

```bash
AREAFLOW_SMOKE_SCRIPT=scripts/smoke-local.sh \
AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1 \
bash scripts/smoke-docker.sh
```

This historical run intentionally used the default `examples/areamatrix/areaflow.yaml`, which points at the real
AreaMatrix root, while omitting `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1`. It failed before status
projection write with:

```text
smoke-local: refusing status projection apply against real AreaMatrix without AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1
smoke-docker: dropping isolated PostgreSQL database areaflow_smoke_20260704105008_38646
expected fail-closed guard PASS rc=1
```

That replay proves the earlier real AreaMatrix write guard failed closed before writing `.areaflow/status.json`.
Current `smoke-local.sh` is stricter: without `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1`, it fails before
project registration or import reads.

## Validation

Focused checks:

```bash
bash -n scripts/smoke-docker.sh scripts/smoke-local.sh scripts/smoke-v1-stable-fixture.sh scripts/smoke-web.sh
make smoke-docker-v1-stable-fixture
AREAFLOW_SMOKE_SCRIPT=scripts/smoke-local.sh AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1 bash scripts/smoke-docker.sh
```

Result:

```text
bash syntax check PASS
v1 stable fixture smoke PASS
isolated Docker smoke database dropped on exit
real AreaMatrix fail-closed guard PASS
```

## Multi-end Observation

Latest Web smoke on 2026-07-03 04:12 CST:

```bash
db="areaflow_web_ops_$(date +%Y%m%d%H%M%S)_$$"
docker compose exec -T postgres createdb -U areaflow "${db}"
AREAFLOW_DATABASE_URL="postgres://areaflow:areaflow@localhost:54329/${db}?sslmode=disable" bash scripts/smoke-web.sh
docker compose exec -T postgres dropdb -U areaflow --if-exists "${db}"
```

Result:

```text
smoke-web: start AreaFlow API 127.0.0.1:3857
smoke-web: start Vite 127.0.0.1:5175
smoke-web: browser check
smoke-web: ops smoke proof record
smoke-web: ok
temporary database dropdb cleanup completed
```

The 04:12 run used fixture project `areamatrix-web-fixture-20260703041229` and proved the Web dashboard
clicks a run timeline item, then requests run detail through the global-ID guard:

```text
GET /api/v1/runs/{run_id}?project_key={fixture_project}
```

The Web dashboard also requests and renders:

```text
GET /api/v1/release/final-gate
GET /api/v1/release/evidence-bundle
GET /api/v1/release/package-preview
GET /api/v1/release/distribution-preview
GET /api/v1/release/publish-gate
GET /api/v1/release/publish-approval-preview
GET /api/v1/release/rollout-plan-preview
Release Final Gate
Release Readiness
Release Evidence
Release Package
Release Distribution
Release Publish
Release Approval
Release Rollout
read_only_release_final_gate
read_only_release_evidence_bundle
read_only_release_package_preview
read_only_release_distribution_preview
read_only_release_publish_gate
read_only_release_publish_approval_preview
read_only_release_rollout_plan_preview
final_gate:release_readiness
evidence:release_final_gate
package:evidence:release_final_gate
distribution:git_release
publish_gate:distribution_preview
publish_approval:publish_gate
rollout_plan:publish_approval
create_release_package
compress_artifacts
create_git_tag
approve_release
create_rollout
publish_release
apply_release
```

The Web smoke checker has since been extended to also expect:

```text
GET /api/v1/ops/readiness
Operations
read_only_operations_readiness
install_migrate_start_register_smoke
metadata_only_support_bundle_preview
migration_ledger_readiness
support_export=deferred_v1x
telemetry=local_only
```

The 04:12 Web smoke run exercised this extended operations readiness expectation against a live API and dashboard.
It records `web_dashboard_ops_smoke` proof after the browser check passes, then verifies operations readiness can
read the latest proof.

The Web smoke checker has since been extended again to also expect:

```text
GET /api/v1/projects/{project}/shim-apply-packet
GET /api/v1/projects/{project}/shim-apply-gate
Shim Apply Review
Packet Gate
project.shim.apply
shim_readiness_still_blocked
area_matrix_files=false
```

The 13:41 Web smoke run on 2026-07-04 exercised this expectation against a live API and dashboard in temporary
database `areaflow_smoke_20260704134145_44355`. It proves the dashboard can show shim apply packet/gate state as a
read-only review surface, not that a shim apply command has been authorized or executed.

The Desktop shell also renders `Release Final Gate`, `Release Evidence`, `Release Package`,
`Release Distribution`, `Release Publish`, `Release Approval`, `Release Rollout` and `Operations Readiness` from
the same read-only API style. It also renders selected-project `Shim Apply Review` from the shim apply packet API.
These observation surfaces do not create release packages, release approvals, release exceptions, rollout state,
tags, pushes, signatures, uploads, publish actions, support exports, telemetry uploads, migration apply operations,
managed upgrades, shim apply commands, status projection writes, AreaMatrix file edits or task-loop forwarding.

## Remaining Boundaries

This evidence proves the v1.0 stable preview/gate chain can run end to end against a safe fixture.
It does not close:

- Real AreaMatrix compatibility shim file edits.
- Real AreaMatrix execution cutover.
- Retained generated-only AreaMatrix apply.
- Source write beta.
- Secret-backed engine execution.
- Remote worker.
- Restore apply.
- Publish apply.
