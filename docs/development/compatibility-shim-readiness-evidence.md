# Compatibility Shim Readiness Evidence

## Purpose

本文记录 backlog 任务
[`AF-V04-001 Compatibility And Shim Readiness`](../../tasks/backlog/0-100-platform-backlog.md#af-v04-001-compatibility-and-shim-readiness)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库、临时 AreaMatrix-like fixture project 和临时 artifact store，
验证 AreaFlow 侧 compatibility contract、shim preview、shim readiness、cutover readiness 和受保护
authoring cutover 链路。它也验证 `shim-readiness-evidence` 证据记录命令可以只写 AreaFlow
command/event/audit state，并验证 `shim-authorization` 只读授权包可以机器可读展示 allowed files、
forbidden paths/actions、preflight、post-edit verification、rollback scope 和 safety facts，而不会写真实
AreaMatrix。最新 focused checkpoint 还验证 `shim-apply-packet` / `shim-apply-gate` 可以把未来 AreaMatrix
shim 编辑申请转成只读 packet 和 go/no-go gate，并在真实 readiness 证据不足时保持 blocked。它不修改真实
AreaMatrix。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/project ./internal/app ./internal/api
```

Result: pass

Latest focused implementation checkpoint:

```text
Date: 2026-07-04
Commands:
  go test ./internal/project -run 'Test(ShimAuthorization|ShimReadiness|ShimApply|Compatibility|StatusProjection)'
  go test ./internal/api -run 'TestProject(Shim|StatusProjection|Compatibility)'
  go test ./internal/app -run 'Test(Shim|StatusProjection|Compatibility)'
Result: pass
```

This checkpoint proves:

```text
project shim-apply-packet/gate pure builders exist and remain read-only
GET /api/v1/projects/{project}/shim-apply-packet forwards approval/proof query fields
GET /api/v1/projects/{project}/shim-apply-gate forwards packet query fields
areaflow project shim-apply-packet/gate JSON/text helpers expose blocked items and safety facts
shim apply gate blocks unless readiness blockers are limited to explicit_edit_approval
shim apply gate requires status projection packet/gate proof, real read-only smoke evidence,
dirty worktree review, protected path fingerprint, rollback plan, explicit approval,
idempotency key and audit correlation id
```

Latest real AreaMatrix read-only checkpoint:

```text
Date: 2026-07-07 13:34 CST
Database: temporary areaflow_smoke_20260707133455_53338 database
Command: make smoke-docker-shim-authorization-preflight
Result: pass
Dropped isolated database: yes
```

Observed latest output:

```text
smoke-shim-authorization-preflight: verifying real AreaMatrix read-only shim/status authorization surface
smoke-areamatrix-readonly: project status-projection-authorization --json
smoke-areamatrix-readonly: project status-projection-apply-packet --json
smoke-areamatrix-readonly: project status-projection-apply-gate --json
smoke-areamatrix-readonly: project shim-preview --json
smoke-areamatrix-readonly: project shim-readiness --json
smoke-areamatrix-readonly: project shim-authorization --json
smoke-areamatrix-readonly: project shim-apply-packet --json
smoke-areamatrix-readonly: project shim-apply-gate --json
smoke-areamatrix-readonly: project shim-authorization text
smoke-areamatrix-readonly: pass areamatrix root=/Users/as/Ai-Project/project/AreaMatrix
smoke-shim-authorization-preflight: ok
```

This checkpoint does not authorize shim implementation. It only refreshes the real AreaMatrix read-only prerequisite:
shim preview/readiness remain query-only, `./task-loop run` remains blocked, and the smoke requires the full protected
path `git status` set to be clean or explicitly reviewed by exact dirty-output hash before the status/shim read-only
chain. It captures and compares the full non-target protected path content fingerprint before and after the chain,
while `.areaflow/status.json` remains covered by its own target fingerprint.
The pre-Package A checkpoint proved `project status-projection-authorization areamatrix --json` could inspect the real
`/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json` preimage as legacy schema, produce expected-before
write-set / schema validator / rollback facts, and keep `apply_open=false`, `project_write_attempted=false`,
`execution_write_attempted=false`, `engine_call_attempted=false`, `command_request_created=false` and
`status_projection_written=false`. After Package A, the same chain sees the stable projection preimage. The smoke
compares project-scoped `command_requests`, `events`, `audit_events`, `gate_results`, `status_projections`,
`workflow_transition_previews`, `approval_records`, `runs`, `run_tasks`, `run_attempts`, `artifacts`,
`artifact_locations`, `artifact_snapshots`, `workers`, `worker_heartbeats`, `leases`, `secret_refs`,
`engine_profiles`, `api_tokens`, `webhooks`, and the project/config/permission/workflow/residual/status snapshot tables
before and after the status/shim authorization, packet and gate chain, so those preview/gate surfaces are proven
DB-read-only for this scope as well as project-file-read-only.
The same checkpoint also proves `project status-projection-apply-packet areamatrix --json` generates the real legacy
preimage packet without side effects, and `project status-projection-apply-gate areamatrix --json` returns blocked/no_go
without creating command requests, status projection rows or AreaMatrix file changes when no apply packet is supplied.
It also proves the authorization packet remains `blocked` and reports no project write, execution write, task-loop run
forwarding, engine call or AreaMatrix file modification. Both JSON and human-readable CLI output also include required
preflight, including the `areaflow project shim-authorization areamatrix --json` self-check, post-edit verification and
rollback scope. The required preflight also carries the AreaMatrix protected path `git status --short -- ...` command
covering `workflow/README.md`, `.areaflow/status.json`, shim scripts, `workflow/versions` and v1 `progress.json`.
The post-edit verification list carries the same protected path check from inside the AreaMatrix repository, so the
authorization packet requires before-and-after proof around the same sensitive paths.
This checkpoint also proves shim readiness now exposes the stable fallback projection contract for
`.areaflow/status.json`: `schema_contract=stable_fallback_projection_v1`, required fields including
`schema_version`, `project_id`, `active_versions[].rough_progress`, `source_snapshot_hash` and
`compatibility.blocked_commands[]`, plus forbidden broad fields such as `summary`, `generated_at`,
`source_hash`, secrets and artifact content. The authorization packet requires
`areaflow project status-projections areamatrix --json`,
`areaflow project status-projection-authorization areamatrix --json`,
`areaflow project status-projection-apply-packet areamatrix --json`,
`areaflow project status-projection-apply-gate areamatrix --json`,
`python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json`
and an explicit stable schema verification before any AreaMatrix shim edit can be approved.
The authorization packet now includes `.areaflow/status.json` in allowed files only as a controlled R1 projection write
through AreaFlow `project.status_projection.apply`; it is not a shim-script write surface and it may not carry full
workflow, execution, approval, log, checkpoint, secret or artifact content.
The same smoke now also runs `project shim-apply-packet areamatrix --json` and
`project shim-apply-gate areamatrix --json`. Both remain blocked/read-only on the real AreaMatrix project because
readiness evidence is incomplete; after Package A, the real `.areaflow/status.json` stable-schema preflight is no
longer the blocker. They report no command request, project write, execution write, task-loop forwarding, status
projection write, AreaMatrix file modification or engine call.

Latest Package B authorization-packet readiness checkpoint:

```text
Date: 2026-07-11 CST
Commands:
  bash -n scripts/audit-package-b-authorization-packet.sh scripts/audit-package-b-dirty-review.sh scripts/audit-package-b-readiness.sh scripts/smoke-package-b-readiness.sh
  make smoke-package-b-readiness
Result: pass
Real AreaMatrix status projection hash: 0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383
Protected dirty output hash: 43a22da86c19781f42d9319136ba812d1dc6f20f4941d5ec0bca4d929d0ee57c
Worktree dirty output hash: 96faa0f64e43b303b715f61e6064cb18a63bf5397582bffafe6d34eb853bd484
Required authorization phrase: 授权执行 Package B，只允许落地 AreaMatrix read-only shim，不允许 ./task-loop run 转发
```

The Package B checkpoint is AreaFlow-only and no-write for AreaMatrix. It proves the readiness scripts fail closed
without exact dirty review, reject dirty hash mismatch, and become
`ready_for_package_b_area_matrix_edit_authorization` only after the protected/worktree dirty output hashes are accepted
with a reviewer. The packet allows only the read-only shim scope:
`scripts/areaflow_shim.py`, `scripts/task_loop/console.py`, `scripts/dev_tools/cli.py`,
`scripts/task_loop/runner.py` and `workflow/README.md`. It explicitly does not authorize `.areaflow/status.json`
writes, `workflow/versions/**`, `progress.json`, logs/checkpoints, native doctor forwarding, `./task-loop run`
forwarding, engine, secret, network, publish or restore.

Latest Package B read-only shim landed checkpoint:

```text
Date: 2026-07-13 CST
Scope: AreaMatrix read-only shim only
Authorization phrase: 授权执行 Package B，只允许落地 AreaMatrix read-only shim，不允许 ./task-loop run 转发
AreaMatrix changed shim paths:
  scripts/areaflow_shim.py
  scripts/dev_tools/cli.py
  scripts/task_loop/runner.py
  scripts/task_loop/console.py
  workflow/README.md
Read-only prerequisite:
  .areaflow/status.json
Protected dirty output hash: 0030c9c11e4e6b6dce2f7b6b703c97185549f6fc6fbf8b068555ceff280a691b
Status projection hash: 0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383
Validation:
  PYTHONDONTWRITEBYTECODE=1 python3 -B -m py_compile scripts/areaflow_shim.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/task_loop/console.py
  git diff --check -- scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/task_loop/console.py workflow/README.md scripts/areaflow_shim.py .areaflow/status.json
  python3 /Users/as/Ai-Project/project/AreaFlow/scripts/validate-status-projection-schema.py /Users/as/Ai-Project/project/AreaFlow/schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
  ./dev workflow doctor
  ./dev workflow status
  ./dev workflow open
  ./dev workflow init --version v2
  ./dev workflow init --version v2 --write
  ./dev workflow discuss --version v2 init --write
  ./dev workflow baseline write
  ./dev workflow promote apply --write
  ./dev workflow project write
  ./dev workflow closeout write
  ./dev dry-run
  ./dev drain
  bash scripts/smoke-package-b-readiness.sh
Result: pass for read-only shim; write-mode workflow commands and Dev Console task-loop wrappers return blocked before legacy runner, progress, log, summary, pid, or checkpoint writes.
```

This landed checkpoint proves the AreaMatrix compatibility entry points are present and fail closed for Package B.
It does not change `.areaflow/status.json`, does not write `workflow/versions/**`, does not forward
`./task-loop run`, and does not prove execution cutover, archive, shim retirement, release packaging, backup/restore,
or security closure completion.

Earlier Package A local readiness audit:

```text
Date: 2026-07-06 22:18 CST
Commands:
  python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
  git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json
  make package-a-readiness
  make package-a-dirty-review
  make package-a-authorization-packet
  bash scripts/audit-package-a-authorization-packet.sh --json
  make smoke-package-a
  go test ./internal/project -run 'Test(StatusProjection|Shim|ExecutionForwardingV1|CompletionAudit)'
  go test ./internal/app -run 'Test(StatusProjection|Shim|ExecutionForwardingV1)'
  go test ./internal/api -run 'TestProject(StatusProjection|Shim|ExecutionForwardingV1)'
Result: blocked for real Package A by protected-path dirty state; JSON authorization packet and smoke-package-a
fail closed; focused AreaFlow tests pass
```

Observed blockers:

```text
.areaflow/status.json still needs Package A apply:
  missing schema_version/project_id/project_name/area_flow_url/cutover_phase/active_versions/last_synced_at/source_snapshot_hash
  includes legacy version/generated_at/project/source/source_hash/summary
  compatibility is missing shim_lifecycle_state/blocked_commands and still includes legacy status/blocked

AreaMatrix protected paths are not clean:
  dirty_output_sha256=ddc240de8acdab7f9f44f92e00e1f61dd365197f49d1bc6bdd0bfa2cff17e5e8
  M scripts/dev_tools/cli.py
  M workflow/versions/v1-mvp/closeout/closeout-decision.md
  M workflow/versions/v1-mvp/evidence/alpha-feedback-route.md
  A workflow/versions/v1-mvp/evidence/distribution-signing-notarization.md
  A workflow/versions/v1-mvp/evidence/final-tag-release-evidence.md
  A workflow/versions/v1-mvp/evidence/icloud-placeholder-smoke-evidence.md
  M workflow/versions/v1-mvp/evidence/recovery-scenarios.md
  M workflow/versions/v1-mvp/evidence/release-checklist.md
  A workflow/versions/v1-mvp/evidence/release-gate-review-task05.md
  M workflow/versions/v1-mvp/evidence/release-notes/release-notes-0.1.0.md
  M workflow/versions/v1-mvp/evidence/release-notes/release-notes-v0.1.0-unnotarized-preview.2.md
  M workflow/versions/v1-mvp/residuals/README.md
  M workflow/versions/v1-mvp/residuals/release-evidence.md
  M workflow/versions/v1-mvp/residuals/residuals.yaml
```

At that checkpoint, the legacy `.areaflow/status.json` shape was the expected input for Package A, not a reason by
itself to block the status projection update. The real Package A write gate still had to remain blocked because the
AreaMatrix protected-path set was dirty and needed to be settled or explicitly reviewed in a separate authorization
packet. `make package-a-readiness` recorded this same no-write local audit and intentionally exited non-zero while
protected-path blockers remained.
`make package-a-dirty-review` prints the exact protected-path lines, touched-path list and sha256 hash that an explicit
dirty-state review packet would need to cite; it does not approve, record or mutate anything.
`make package-a-authorization-packet` turns the same facts into a no-write Package A authorization packet. The same
script also supports `--json`; the JSON packet includes `allowed_writes=[".areaflow/status.json"]`,
`dirty_output_sha256`, exact `protected_path_lines`, `touched_paths`, required after-check commands, the required user
authorization phrase and no-write `safety_facts` such as `modifies_areamatrix=false`,
`applies_status_projection=false`, `allows_shim_files=false`, `allows_workflow_versions=false` and
`allows_task_loop_forwarding=false`. In the current workspace it exits non-zero with
`status=blocked_needs_preflight_review`, because the protected-path dirty state must be settled or explicitly reviewed
before the narrow `.areaflow/status.json` write can be authorized.
`make smoke-package-a` and `make smoke-docker-package-a` compose that local audit with the existing shim authorization
preflight and Execution Forwarding v1 readiness smoke when readiness passes. When the current protected-path blockers
are present, they instead assert the Package A authorization packet stays blocked/fail-closed and still carries the
same no-write safety facts; they do not write AreaFlow DB rows or modify AreaMatrix.

Latest Package A status projection apply checkpoint:

```text
Date: 2026-07-10 16:52 CST
AreaMatrix target: /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
Result: written
Final status hash: 0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383
Source snapshot hash: 35f51913af9834e7e21540c866e06f87150eb2cdd109e5f6457e8e659a363a3a
Allowed AreaMatrix write set: .areaflow/status.json only
```

The current real AreaMatrix status projection is `stable_fallback_projection_v1` and includes
`schema_version`, `project_id`, `active_versions`, `rough_progress`, `source_snapshot_hash` and
`compatibility.blocked_commands`. No shim files, `workflow/README.md`, `workflow/versions/**`, task-loop forwarding,
execution write, engine, secret, network, publish or restore authority was opened by Package A.

Current protected shim-apply command checkpoint:

```text
Date: 2026-07-06 20:41 CST
Commands:
  go test ./internal/project -run 'TestBuildApplyShimCommandResult|TestBuildShimApply|TestShim'
  go test ./internal/app -run 'TestProjectStatusProjectionSubcommandHelpDoesNotRequireDatabase|TestHelp|TestShimApply'
  go test ./internal/api -run 'TestProjectShimApply|TestProject.*Shim'
  bash -n scripts/smoke-areamatrix-readonly.sh scripts/smoke-compatibility-fixture.sh
Result: pass
```

`areaflow project shim-apply ...` and `POST /api/v1/projects/<project>/shim-apply` are now explicit protected
AreaFlow-only commands. They reuse the shim apply gate input, record `command_requests` / `events` / `audit_events`
when the gate passes, and still report `project_write_attempted=false`, `execution_write_attempted=false`,
`task_loop_run_forwarded=false`, `status_projection_written=false` and `area_matrix_files_modified=false`. This closes
the previous ambiguity where `shim-apply-packet` emitted a future apply command name while the command itself had no
protected command/audit boundary.
`scripts/smoke-areamatrix-readonly.sh` now calls the real AreaMatrix `shim-apply --json` path and requires it to fail
closed without project, execution, task-loop, status projection, engine or AreaMatrix file side effects.
`scripts/smoke-compatibility-fixture.sh` also calls `shim-apply --json` after a fixture packet/gate becomes eligible,
and requires it to record AreaFlow command state while keeping all AreaMatrix write flags false.

Latest compatibility fixture checkpoint:

```text
Date: 2026-07-04 13:42 CST
Database: temporary areaflow_smoke_20260704134229_45357 database on localhost:54329
Command: make smoke-docker-compatibility-fixture
Result: pass
Dropped isolated database: yes
```

Observed latest output:

```text
smoke-compatibility-fixture: project status-projection-apply areamatrix-compat-fixture
smoke-compatibility-fixture: validate status projection schema
status projection schema validation passed: /private/var/folders/.../areaflow-compat-fixture.*/areamatrix-root/.areaflow/status.json
smoke-compatibility-fixture: project compatibility --json
smoke-compatibility-fixture: project shim-preview --json
smoke-compatibility-fixture: project shim-readiness --json
smoke-compatibility-fixture: project shim-authorization --json
smoke-compatibility-fixture: project shim-authorization text
smoke-compatibility-fixture: project shim-apply-packet before evidence --json
smoke-compatibility-fixture: project shim-apply-gate before evidence --json
smoke-compatibility-fixture: project shim-readiness-evidence real_areamatrix_readonly_smoke
smoke-compatibility-fixture: project shim-readiness-evidence real_areamatrix_status_projection_schema
smoke-compatibility-fixture: project shim-readiness-evidence areamatrix_dirty_worktree_review
smoke-compatibility-fixture: project shim-readiness after evidence --json
smoke-compatibility-fixture: project shim-apply-packet after evidence --json
smoke-compatibility-fixture: project ready-path cutover-apply
smoke-compatibility-fixture: pass areamatrix-compat-fixture fixture=/private/var/folders/.../areaflow-compat-fixture.*
```

This checkpoint proves compatibility and shim readiness remain healthy in a clean temporary database. It also proves
authoring cutover apply remains AreaFlow DB-only and does not write project files or execution state. The fixture smoke
now validates the generated fixture `.areaflow/status.json` against `schemas/status-projection.schema.json` and asserts
the same stable fallback projection contract and schema-verification preflight as the real AreaMatrix read-only smoke,
without writing the real AreaMatrix repository.
By default this fixture smoke also avoids reading real AreaMatrix projection fingerprints; callers that need that extra
guard must opt in with `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`.
It now also proves `shim-apply-packet` / `shim-apply-gate` stay blocked before readiness evidence, then a fixture
packet with explicit approval and proof ids can become `ready_for_future_apply_command` after evidence is recorded while
still creating no command request and writing no project, execution, status projection or AreaMatrix file.
It also proves `real_areamatrix_status_projection_schema` is a first-class shim readiness evidence key. In the real
AreaMatrix path that evidence remains blocked until the actual `.areaflow/status.json` validates against the stable
schema.

Latest Web dashboard preflight checkpoint:

```text
Date: 2026-07-04 13:41 CST
Database: temporary areaflow_smoke_20260704134145_44355 database on localhost:54329
Command: AREAFLOW_SMOKE_SCRIPT=scripts/smoke-web.sh bash scripts/smoke-docker.sh
Result: pass
Cleanup residual database count: 0
```

Observed latest output:

```text
smoke-web: start AreaFlow API 127.0.0.1:3857
smoke-web: start Vite 127.0.0.1:5175
smoke-web: browser check
smoke-web: ops smoke proof record
smoke-web: ok
```

This checkpoint proves the Web dashboard can observe the fixture project through AreaFlow API/SSE during shim preflight.
It does not open Web write actions.

AreaMatrix shim/status file spot check after the smoke:

```text
git status --short .areaflow/status.json workflow/README.md scripts/areaflow_shim.py \
  scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py
```

Result:

```text
<no output>
```

Latest fixture evidence-record checkpoint:

```text
Date: 2026-07-02 19:29 CST
Database: af_v04_1782991754_40166
Command: AREAFLOW_DATABASE_URL=... bash scripts/smoke-compatibility-fixture.sh
Result: pass
Cleanup residual database count: 0
```

This checkpoint proves the AreaFlow-side evidence command path records:

```text
real_areamatrix_readonly_smoke
areamatrix_dirty_worktree_review
```

and then keeps shim readiness `blocked` because `explicit_edit_approval` is still required before writing AreaMatrix
shim files.

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v04_1782991754_40166
Fixture project key: areamatrix-compat-fixture
Fixture root: temporary /private/var/folders/.../areaflow-compat-fixture.*
Artifact store: temporary /private/var/folders/.../areaflow-compat-fixture.*/artifact-store
```

Smoke command:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_v04_1782991754_40166?sslmode=disable \
  ./scripts/smoke-compatibility-fixture.sh
```

Cleanup command:

```bash
dropdb -h localhost -p 54329 -U areaflow --if-exists af_v04_1782991754_40166
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname = 'af_v04_1782991754_40166';"
```

## Result

Status: pass

Observed smoke output:

```text
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
smoke-compatibility-fixture: project add /private/var/folders/.../areaflow.yaml
smoke-compatibility-fixture: project import areamatrix-compat-fixture #1
smoke-compatibility-fixture: project import areamatrix-compat-fixture #2
smoke-compatibility-fixture: project doctor --json
smoke-compatibility-fixture: project status-projection-apply areamatrix-compat-fixture
smoke-compatibility-fixture: project compatibility --json
smoke-compatibility-fixture: project shim-preview --json
smoke-compatibility-fixture: project shim-readiness --json
smoke-compatibility-fixture: project shim-readiness-evidence real_areamatrix_readonly_smoke
smoke-compatibility-fixture: project shim-readiness-evidence areamatrix_dirty_worktree_review
smoke-compatibility-fixture: project shim-readiness after evidence --json
smoke-compatibility-fixture: workflow blocked-path version create compat-smoke-20260702192914
smoke-compatibility-fixture: workflow blocked-path ensure-skeleton compat-smoke-20260702192914
smoke-compatibility-fixture: workflow blocked-path gate promotion_preview
smoke-compatibility-fixture: workflow blocked-path transition preview
smoke-compatibility-fixture: workflow blocked-path approval rejected
smoke-compatibility-fixture: workflow blocked-path approval_gate
smoke-compatibility-fixture: workflow blocked-path live_mapping_gate
smoke-compatibility-fixture: project blocked-path cutover-readiness
smoke-compatibility-fixture: workflow blocked-path cutover_readiness_gate
smoke-compatibility-fixture: workflow ready-path version create compat-smoke-20260702192914-ready
smoke-compatibility-fixture: workflow ready-path mark queue
smoke-compatibility-fixture: workflow ready-path mark promotion_preview
smoke-compatibility-fixture: workflow ready-path gate promotion_preview
smoke-compatibility-fixture: workflow ready-path transition preview
smoke-compatibility-fixture: workflow ready-path approval approved
smoke-compatibility-fixture: workflow ready-path approval_gate
smoke-compatibility-fixture: workflow ready-path live_mapping_gate
smoke-compatibility-fixture: project ready-path cutover-readiness
smoke-compatibility-fixture: workflow ready-path cutover_readiness_gate
smoke-compatibility-fixture: project ready-path cutover-apply
smoke-compatibility-fixture: run ready-path preview
smoke-compatibility-fixture: run ready-path start
smoke-compatibility-fixture: run ready-path drain
smoke-compatibility-fixture: run ready-path cancel
smoke-compatibility-fixture: artifact archive preview
smoke-compatibility-fixture: pass areamatrix-compat-fixture fixture=/private/var/folders/.../areaflow-compat-fixture.*
smoke_db=af_v04_1782991754_40166 residual=0 rc=0
```

## Evidence

- 2026-07-04 02:49 CST focused checks passed:
  `go test ./internal/project ./internal/api ./internal/app -run 'ShimAuthorization|ShimReadiness|Compatibility|StatusProjection'`,
  `bash -n scripts/smoke-shim-authorization-preflight.sh scripts/smoke-areamatrix-readonly.sh scripts/smoke-compatibility-fixture.sh scripts/smoke-fixture.sh`,
  `make smoke-docker-compatibility-fixture` and `make smoke-docker-areamatrix-readonly`.
- 2026-07-04 03:00 CST `make smoke-docker-compatibility-fixture` passed on temporary database
  `areaflow_smoke_20260704030040_28993` and validated fixture `.areaflow/status.json` against
  `schemas/status-projection.schema.json`.
- 2026-07-04 03:04 CST `make smoke-docker-compatibility-fixture` passed on temporary database
  `areaflow_smoke_20260704030446_48609` and asserted the shim authorization packet includes the executable schema
  validator preflight.
- 2026-07-04 03:05 CST `make smoke-docker-areamatrix-readonly` passed on temporary database
  `areaflow_smoke_20260704030508_51268` and asserted the real AreaMatrix read-only authorization packet includes the
  same executable schema validator preflight while leaving real AreaMatrix protected status/readme fingerprints
  unchanged.
- 2026-07-04 03:12 CST `make smoke-docker-compatibility-fixture` passed on temporary database
  `areaflow_smoke_20260704031239_84553` and proved `real_areamatrix_status_projection_schema` can be recorded as a
  no-write shim readiness evidence command.
- 2026-07-04 03:13 CST `make smoke-docker-areamatrix-readonly` passed on temporary database
  `areaflow_smoke_20260704031300_87225` and proved the real AreaMatrix read-only readiness surface exposes the
  `real_areamatrix_status_projection_schema` gate while leaving protected fingerprints unchanged.
- 2026-07-04 04:52 CST `make smoke-docker-compatibility-fixture` passed on temporary database
  `areaflow_smoke_20260704045248_21865`, including status projection apply packet/gate preflight, stable schema
  validation, shim readiness evidence records, DB-only cutover apply and run/artifact preview safety checks.
- 2026-07-04 04:52 CST `make smoke-docker-areamatrix-readonly` passed on temporary database
  `areaflow_smoke_20260704045232_19940` and again proved real AreaMatrix status/readme/shim protected paths stayed
  unchanged while authorization, apply packet, apply gate and shim authorization remained read-only.
- 2026-07-04 04:52 CST `make smoke-docker-web` passed on temporary database
  `areaflow_smoke_20260704045248_21874`, proving the Web dashboard still observes these readiness surfaces through
  API/SSE without opening write actions.
- 2026-07-04 13:41 CST `AREAFLOW_SMOKE_SCRIPT=scripts/smoke-web.sh bash scripts/smoke-docker.sh` passed on temporary
  database `areaflow_smoke_20260704134145_44355`, proving the Web dashboard observes shim apply packet/gate through
  GET-only API calls and still fails closed on write surfaces.
- 2026-07-04 13:42 CST `make smoke-docker-compatibility-fixture` passed on temporary database
  `areaflow_smoke_20260704134229_45357`, including shim apply packet/gate before/after readiness evidence while
  keeping command request, project write, execution write, status projection write and AreaMatrix file modification
  closed.
- 2026-07-04 13:42 CST `make smoke-docker-areamatrix-readonly` passed on temporary database
  `areaflow_smoke_20260704134250_46786` and proved the real AreaMatrix read-only path still keeps status/readme/shim
  protected paths unchanged while shim apply packet/gate remain blocked/read-only.
- 2026-07-06 00:01 CST `make smoke-docker-areamatrix-readonly` passed on temporary database
  `areaflow_smoke_20260706000100_84386` and again proved the real AreaMatrix read-only path keeps status/readme/shim
  protected paths unchanged while status projection and shim apply packet/gate remain blocked/read-only. The isolated
  database was dropped after the smoke.
- 2026-07-06 00:07 CST `make smoke-docker-shim-authorization-preflight` passed on temporary database
  `areaflow_smoke_20260706000723_806`, proving the named AF-V04 authorization preflight wrapper reaches the same
  real AreaMatrix read-only status/shim authorization, apply-packet/gate and no-write safety checks. The isolated
  database was dropped after the smoke.
- 2026-07-06 00:53 CST `make smoke-docker-shim-authorization-preflight` passed on temporary database
  `areaflow_smoke_20260706005304_92636`, adding full protected path `git status` before/after checks and project-level
  DB side-effect counts for `command_requests`, `events`, `audit_events`, `gate_results` and `status_projections`.
  The isolated database was dropped after the smoke.
- 2026-07-07 13:14 CST `make smoke-docker-shim-authorization-preflight` passed on temporary database
  `areaflow_smoke_20260707131456_30958`, adding full non-target protected path content fingerprint before/after checks
  and expanding project-level DB side-effect counts to transition preview, approval, run/task/attempt, artifact, worker,
  heartbeat and lease tables. The isolated database was dropped after the smoke.
- 2026-07-07 13:34 CST `make smoke-docker-shim-authorization-preflight` passed on temporary database
  `areaflow_smoke_20260707133455_53338`, after follow-up hardening expanded the same DB side-effect count set further
  to all current project-scoped mutable tables from the AreaFlow migrations, including project/config/permission/
  workflow/residual/status snapshot, artifact location/snapshot and reserved security/integration tables. Web real
  AreaMatrix mode now captures the same recursive non-target protected path content fingerprint used by the CLI
  read-only smoke. The isolated database was dropped after the smoke.
- The current real AreaMatrix `.areaflow/status.json` is now the stable fallback projection written under Package A.
  Package B readiness treats it as a read-only prerequisite and does not authorize writing it again.
- `go test ./internal/project ./internal/app ./internal/api` passed.
- 2026-07-02 21:13 CST authorization preflight passed `scripts/smoke-compatibility-fixture.sh` and
  `scripts/smoke-areamatrix-readonly.sh` on temporary PostgreSQL database `af_shim_auth_20260702211341_38096`;
  cleanup residual database count was 0.
- 2026-07-02 21:20 CST Web dashboard preflight passed `scripts/smoke-web.sh` on temporary PostgreSQL database
  `af_web_shim_auth_20260702212028_62899`; cleanup residual database count was 0.
- Migrations `000001` through `000010` applied from an empty temporary PostgreSQL database.
- `project import` completed twice, proving import command requests are repeatable for fixture metadata.
- `project status-projection-apply` wrote only the fixture `.areaflow/status.json`.
- `scripts/validate-status-projection-schema.py` validated the compatibility fixture `.areaflow/status.json` against
  `schemas/status-projection.schema.json`.
- `project compatibility --json` returned `status=pass`.
- `./task-loop run` remained `mode=blocked` with blocked reason `execution and task-loop replacement are out of v0.4 scope`.
- `project shim-preview --json` returned read-only planning mode, planned files, command mappings, forbidden paths and verification commands.
- `project shim-readiness --json` now returns `status_projection` metadata with
  `schema_contract=stable_fallback_projection_v1`, `.areaflow/status.json` target URI, required schema fields and
  forbidden broad fields, plus `schema_uri=schemas/status-projection.schema.json` and the executable validator
  preflight command.
- `project shim-readiness --json` now includes `real_areamatrix_status_projection_schema` as a separate blocked evidence
  gate before any real AreaMatrix shim edit.
- `project shim-authorization --json` returned a blocked, read-only authorization packet with allowed files, forbidden
  paths/actions, preflight, post-edit verification, rollback scope and false write/execution/engine/command/network safety facts.
- `project shim-authorization --json` and text output require both
  `areaflow project status-projections areamatrix --json`, the executable schema validator command and explicit
  verification that `.areaflow/status.json` includes the stable fallback fields while excluding legacy broad fields.
- `project shim-readiness --json` returned `status=blocked` because real AreaMatrix shim edits still require external evidence and explicit approval.
- `project shim-readiness-evidence --key real_areamatrix_readonly_smoke --status pass --json` recorded evidence through Command API and returned `project_write_attempted=false`, `execution_write_attempted=false` and `engine_call_attempted=false`.
- `project shim-readiness-evidence --key real_areamatrix_status_projection_schema --status pass --json` recorded fixture evidence through Command API and returned the same no-write safety facts.
- `project shim-readiness-evidence --key areamatrix_dirty_worktree_review --status pass --json` recorded evidence through Command API and returned the same no-write safety facts.
- After all three evidence records, `project shim-readiness --json` showed the evidence items as recorded/pass but still returned `status=blocked` because explicit edit approval is required before writing AreaMatrix shim files.
- The fixture smoke verified at least three completed `project.shim_readiness_evidence.record` command requests with response `event_id` and `audit_event_id`.
- Blocked-path workflow proved cutover readiness remains blocked when promotion, approval and live mapping gates are not pass.
- Ready-path workflow proved promotion preview, transition preview, approval, approval gate, live mapping gate, cutover readiness and `cutover_readiness_gate` can pass.
- `project cutover-apply` completed only AreaFlow DB-side authoring cutover and returned `project_write_attempted=false` and `execution_write_attempted=false`.
- Protected run control commands `run start`, `run drain` and `run cancel` returned `project_write_attempted=false`, `execution_write_attempted=false` and `engine_call_attempted=false`.
- `artifact archive-preview` returned metadata-only archive preview and recorded `project_write_attempted=false`, `storage_write_attempted=false` and `artifact_delete_attempted=false`.
- The compatibility fixture smoke now skips real AreaMatrix `.areaflow/status.json` and `workflow/README.md`
  fingerprint checks by default; set `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` to opt in when that out-of-band guard
  is intentionally needed.
- The temporary PostgreSQL database was dropped after the smoke and residual database count was `0`.
- The latest real AreaMatrix read-only smoke checked both the full AreaMatrix protected path git status set and the
  non-target protected path content fingerprint before and after the run, covering `.areaflow/status.json`,
  `workflow/README.md`, shim scripts, `workflow/versions` and v1 `progress.json`.
- The latest Web smoke started a local AreaFlow API and Vite Web server, then completed the browser check without
  opening Web write actions. Real AreaMatrix Web mode now also checks the recursive non-target protected path content
  fingerprint before and after browser observation.

## Boundary

这份证据不证明：

- Real AreaMatrix shim file edits.
- `workflow/README.md` controlled block write.
- Real AreaMatrix authoring cutover.
- Execution cutover.
- `./task-loop run` replacement.
- Real project file writes.
- Real runner / worker copy, verify, repair or checkpoint execution.
- Secret resolution or AI engine calls.
- Web write actions or Desktop behavior.

这些仍属于独立 backlog 项和后续 gate。AreaMatrix 仓库内 shim 修改仍必须获得明确授权。
