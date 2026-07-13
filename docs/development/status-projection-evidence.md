# Status Projection Evidence

## Purpose

本文记录 backlog 任务
[`AF-V01-003 Guarded Status Projection`](../../tasks/backlog/0-100-platform-backlog.md#af-v01-003-guarded-status-projection)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库和临时 AreaMatrix-like fixture project，验证受保护的
`.areaflow/status.json` projection authorization preview、只读 apply gate 和 fixture-only projection
write。真实 AreaMatrix checkpoint 默认只运行 authorization preview / apply gate；只有 Package A 精确授权
路径允许写真实 AreaMatrix 的 `.areaflow/status.json`。

## Run

Date: 2026-07-04 04:52 CST

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: areaflow_smoke_20260704045220_18500
Fixture project key: areamatrix-fixture
Fixture root: temporary /private/var/folders/.../areaflow-fixture.vn2OkV
```

Commands:

```bash
go test ./internal/status ./internal/adapter/areamatrix ./internal/project
go test -count=1 ./internal/project ./internal/app ./internal/api -run 'StatusProjection|Help'
go test -count=1 ./internal/project ./internal/app ./internal/api
go build ./cmd/areaflow
python3 -m py_compile scripts/validate-status-projection-schema.py
bash -n scripts/smoke-fixture.sh scripts/smoke-areamatrix-readonly.sh scripts/smoke-status-projection-schema.sh
make smoke-status-projection-schema
make smoke-docker-fixture
make smoke-docker-areamatrix-readonly
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
applied 000011_v1_migration_ledger.sql
smoke-fixture: project add /private/var/folders/.../areaflow-fixture.*/areaflow.yaml
smoke-fixture: project import areamatrix-fixture #1
smoke-fixture: project import areamatrix-fixture #2
smoke-fixture: project doctor --json
smoke-fixture: project status-projection-authorization areamatrix-fixture
smoke-fixture: project status-projection-apply-packet areamatrix-fixture missing approval
smoke-fixture: project status-projection-apply-gate areamatrix-fixture missing packet
smoke-fixture: project status-projection-apply-packet areamatrix-fixture complete approval
smoke-fixture: project status-projection-apply-gate areamatrix-fixture complete packet
smoke-fixture: project status-projection-apply areamatrix-fixture
smoke-fixture: validate status projection schema
status projection schema validation passed: /private/var/folders/.../areaflow-fixture.*/areamatrix-root/.areaflow/status.json
smoke-fixture: project status-projections --json
smoke-fixture: project summary --json
smoke-fixture: project readiness --json
smoke-fixture: project import-diff --json
smoke-fixture: project verify-bundle --json
smoke-fixture: pass areamatrix-fixture fixture=/private/var/folders/.../areaflow-fixture.*
smoke-docker: ok
smoke-docker: dropping isolated PostgreSQL database areaflow_smoke_20260704045220_18500
```

Schema validator smoke:

```text
smoke-status-projection-schema: valid projection
smoke-status-projection-schema: pass
```

Latest guarded write fixture checkpoint:

```text
Date: 2026-07-04 16:36 CST
Database: temporary areaflow_smoke_20260704163601_70659 database on localhost:54329
Command: make smoke-docker-fixture
Result: pass
Cleanup residual database count: 0
```

This run exercised the protected `project status-projection-apply` path against a temporary fixture project and
validated the generated fixture `.areaflow/status.json` after the writer passed root containment checks, built-in
stable projection validation, same-directory atomic replace, apply-layer post-write hash/size verification and retained
the preimage rollback compensation path in tests. Apply-layer unit coverage now also proves writer-error partial write
compensation for replace, create and delete outcomes: a writer that changes an existing target then errors restores the
captured preimage, a writer that creates a new target then errors removes it, and a writer that deletes an existing
target then errors restores it. The CLI smoke also asserted the apply output exposes `write_safety` facts for preimage
capture, post-write verification, root containment, stable projection validation, atomic replace and rollback
compensation.

Projection safety assertions:

```text
scripts/smoke-fixture.sh first ran project status-projection-authorization and asserted needs_approval,
apply_open=false, approval_required=true, schema URI, validator preflight, preimage match, schema validation,
protected paths, project_write_attempted=false, execution_write_attempted=false and engine_call_attempted=false.
It then ran project status-projection-apply-packet twice: a missing-approval packet preview that returned
needs_approval and a complete approval packet preview that returned ready_for_apply_command. Both packet previews
reported command_request_created=false, status_projection_written=false and project_write_attempted=false. It also ran
project status-projection-apply-gate twice: a missing-packet gate that returned blocked/no_go and a complete fixture
packet that returned pass/go with apply_command_eligible=true while still reporting no side effects.
scripts/smoke-fixture.sh asserted one completed project.status_projection.apply command whose response recorded
apply_gate_status=pass, apply_gate_decision=go and apply_command_eligible=true; one .areaflow/status.json target;
preimage/root/validation/atomic-replace/rollback-compensation write safety facts; no execution write; no engine
call; fixture-only project write; and one written status_projections row.
By default the fixture smoke no longer reads the real AreaMatrix repository for an out-of-band fingerprint check.
Set `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` only when the caller intentionally wants that additional guard.
```

字段含义：

- 1 completed `project.status_projection.apply` command request。
- 1 command response target 是 `.areaflow/status.json`。
- 1 command response 明确 `apply_gate_status=pass`、`apply_gate_decision=go`、
  `apply_command_eligible=true`。
- 1 command response 明确 `execution_write_attempted=false`。
- 1 command response 明确 `engine_call_attempted=false`。
- 1 command response 明确 `project_write_attempted=true`，且该写入位于 fixture root。
- 1 command response 明确 `preimage_captured=true`、`preimage_exists=false`、`root_contained=true`、
  `stable_projection_validated=true`、`atomic_replace_used=true` 和
  `rollback_compensation_enabled=true`。
- 1 `status_projections` row 处于 `write_state='written'`。
- 1 authorization preview 在 apply 前明确 `status=needs_approval`、`apply_open=false`、
  `approval_required=true`、`project_write_attempted=false`、`execution_write_attempted=false`、
  `engine_call_attempted=false`，并且没有创建 fixture `.areaflow/status.json`。
- 1 missing-approval apply packet preview 在 apply 前明确 `status=needs_approval`、
  `decision=needs_explicit_approval`、`apply_command_eligible=false`、`command_request_created=false`、
  `status_projection_written=false`，并且没有创建 fixture `.areaflow/status.json`。
- 1 complete-approval apply packet preview 在 apply 前明确 `status=ready`、
  `decision=ready_for_apply_command`、`apply_command_eligible=true`，但仍保持
  `command_request_created=false`、`status_projection_written=false`、`project_write_attempted=false`。
- 1 missing-packet apply gate 在 apply 前明确 `status=blocked`、`decision=no_go`、
  `apply_command_eligible=false`、`command_request_created=false`、`status_projection_written=false`，
  并且没有创建 fixture `.areaflow/status.json`。
- 1 complete-packet apply gate 在 apply 前明确 `status=pass`、`decision=go`、
  `apply_command_eligible=true`，但仍保持 `command_request_created=false`、
  `status_projection_written=false`、`project_write_attempted=false`。

Cleanup:

```text
smoke-docker dropped areaflow_smoke_20260704045220_18500 after the run.
```

本次 fixture-only smoke 不写真实 AreaMatrix `.areaflow/status.json`。

Latest real AreaMatrix read-only authorization checkpoint:

```text
Date: 2026-07-06 00:01 CST
Database: temporary areaflow_smoke_20260706000100_84386 database on localhost:54329
Command: make smoke-docker-areamatrix-readonly
Result: pass
Dropped isolated database: yes
```

Latest Package A authorization / preflight hardening checkpoint:

```text
Date: 2026-07-10 11:53 CST
Database: temporary areaflow_smoke_20260710115343_3883 database on localhost:54329
Command: make smoke-docker-package-a
Result: pass
Dropped isolated database: yes
```

Latest real Package A apply checkpoint:

```text
Date: 2026-07-10 16:52 CST
AreaMatrix target: /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
Allowed write set: .areaflow/status.json only
Result: written
Final status hash: 0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383
Source snapshot hash: 35f51913af9834e7e21540c866e06f87150eb2cdd109e5f6457e8e659a363a3a
```

The real AreaMatrix status file now validates as the stable fallback projection shape and contains
`schema_version`, `project_id`, `active_versions`, `rough_progress`, `source_snapshot_hash` and
`compatibility.blocked_commands`. The Package A write did not authorize or modify shim files,
`workflow/README.md`, `workflow/versions/**`, task-loop forwarding, execution write, engine, secret, network,
publish or restore paths.

Observed real read-only output:

```text
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
```

Real read-only assertions:

- `project status-projection-authorization areamatrix --json` returned `status=needs_approval`,
  `decision=needs_explicit_approval`, `apply_open=false`, `approval_required=true`, and
  `approval_status=missing`.
- The authorization preview, apply packet preview and apply gate expose `claim_scope =
  package_a_status_projection_preflight_only` and `not_real_100=true`, so consumers do not need to infer that these
  read-only surfaces are not release-completion claims.
- The authorization packet targeted `/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json` and carried
  `schema_uri=schemas/status-projection.schema.json` plus the executable validator preflight.
- The pre-Package A preimage inspection identified the real AreaMatrix status file as `schema_status=legacy` with
  missing stable fields including `schema_version`, `project_id`, `project_name`, `area_flow_url`, `cutover_phase`,
  `active_versions`, `last_synced_at` and `source_snapshot_hash`; legacy broad fields including `version`,
  `generated_at`, `source`, `source_hash` and `summary`; and legacy compatibility shape that lacked
  `shim_lifecycle_state` / `blocked_commands` while still exposing `status` / `blocked`.
- The write-set required `expected-before` preimage matching and schema validation, but only after explicit approval.
- `project status-projection-apply-packet areamatrix --json` generated the pre-apply real legacy preimage packet, CLI
  apply command and API request, but returned `status=needs_approval` and created no command/status projection DB side
  effect and no AreaMatrix file change. It also exposes `apply_command_eligible_is_not_apply=true` and
  `requires_separate_apply_command=true`.
- `project status-projection-apply-gate areamatrix --json` returned `status=blocked`, `decision=no_go`,
  `apply_command_eligible=false` and `approval_status=missing_or_incomplete` when no packet was supplied.
- With a DB-bound source hash and the exact Package A authorization phrase, `project status-projection-apply-gate`
  can return `status=pass`, `decision=go` and `apply_command_eligible=true`, but still reports
  `command_request_created=false`, `status_projection_written=false`, `project_write_attempted=false`,
  `apply_command_eligible_is_not_apply=true` and `requires_separate_apply_command=true`; it is readiness for a
  separate protected apply command, not an apply.
- The real apply gate exposed `explicit_approval`, `source_snapshot_hash` and `expected_before_sha256` blockers while
  carrying the nested legacy authorization/preimage facts.
- The preview safety facts asserted `project_write_attempted=false`, `execution_write_attempted=false`,
  `engine_call_attempted=false`, `command_request_created=false`, `status_projection_written=false` and
  `areamatrix_protected_paths_touched=false`.
- The smoke compared project-scoped `command_requests`, `events`, `audit_events`, `gate_results` and
  `status_projections` rows before/after the status/shim authorization, packet and gate chain, proving no AreaFlow DB
  write-side effect was created by any read-only operation.
- The smoke checked the full real AreaMatrix protected path git status set before/after, covering
  `.areaflow/status.json`, `workflow/README.md`, shim scripts, `workflow/versions` and v1 `progress.json`, and proved
  no real AreaMatrix protected path changed.

## Evidence

- `go test ./internal/status ./internal/adapter/areamatrix ./internal/project` passed.
- `go test -count=1 ./internal/project ./internal/app ./internal/api -run 'StatusProjection|Help'` passed.
- `go test -count=1 ./internal/project ./internal/app ./internal/api` passed.
- `go build ./cmd/areaflow` passed.
- `python3 -m py_compile scripts/validate-status-projection-schema.py` passed.
- `bash -n scripts/smoke-fixture.sh scripts/smoke-areamatrix-readonly.sh scripts/smoke-status-projection-schema.sh` passed.
- `make smoke-status-projection-schema` passed and proved the validator accepts a valid projection while rejecting:
  missing `source_snapshot_hash`, legacy top-level `summary`, `rough_progress.percent > 100` and nested
  `compatibility.artifact_content`.
- `make smoke-docker-areamatrix-readonly` passed before Package A apply and proved the real AreaMatrix authorization
  preview was read-only, detected the then-current legacy `.areaflow/status.json` shape, including missing stable
  compatibility fields, and created no command/status projection DB side effect.
- `make smoke-docker-package-a` passed on 2026-07-10 and proved Package A packet binding, exact authorization phrase
  gating, `claim_scope`, `not_real_100`, `apply_command_eligible_is_not_apply` and
  `requires_separate_apply_command` are visible before any protected apply command runs.
- A later real Package A apply wrote only `/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json`; the current
  file hash is `0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383`, with source snapshot hash
  `35f51913af9834e7e21540c866e06f87150eb2cdd109e5f6457e8e659a363a3a`.
- Migrations applied from an empty temporary PostgreSQL database.
- `project import` created import snapshots before projection apply.
- `project status-projection-authorization` produced a read-only authorization packet with target URI, schema URI,
  validator preflight, protected path check, expected-before preimage, write-set, rollback plan and safety facts.
- `project status-projection-apply-packet` produced missing-approval and complete-approval packet previews with CLI
  apply command / API request output and no side effects.
- `project status-projection-apply-gate` produced a missing-packet blocked/no_go response and a complete-packet
  pass/go response without creating command requests, status projection rows or project writes.
- `project status-projection-apply` consumed the same complete gate packet, recorded pass/go gate facts in the
  command response and wrote only the fixture `.areaflow/status.json`.
- `internal/status` now rejects empty targets, absolute targets, `../` path escapes and symlinked parent escapes
  before writing; it then writes `.areaflow/status.json` through a same-directory temp file and atomic replace, with
  unit coverage proving replacement produces valid `stable_fallback_projection_v1` JSON and leaves no
  `.status.json.tmp-*` residue. Rollback compensation uses the same directory and may transiently create
  `.status.json.rollback-*`, with unit coverage proving restored preimages do not leave rollback temp files behind.
- `internal/status` also performs built-in stable projection validation before writing, with unit coverage rejecting
  missing `source_snapshot_hash`, invalid `rough_progress.percent`, and legacy top-level `summary`.
- `internal/project` now captures the status projection preimage before writer execution and has unit coverage for
  atomic pre-commit rollback: restoring an existing preimage, removing a newly-created target, and refusing rollback
  when the written file hash has drifted.
- `internal/project` also has unit coverage for writer-error partial write compensation: replacing an existing target,
  creating a new target, and deleting an existing target all leave the project file restored or removed according to the
  captured preimage before the blocked result is recorded.
- `schemas/status-projection.schema.json` is now the machine-readable contract for `stable_fallback_projection_v1`.
- `scripts/validate-status-projection-schema.py` validated the generated fixture `.areaflow/status.json` against that schema.
- The fixture `.areaflow/status.json` uses the stable fallback projection schema with `schema_version`,
  `project_id`, `area_flow_url`, `cutover_phase`, `active_versions[].rough_progress`,
  `last_synced_at`, `source_snapshot_hash` and `compatibility.blocked_commands`.
- The fixture `.areaflow/status.json` no longer exposes the legacy broad top-level `summary`, `generated_at`,
  `source` or `source_hash` fields.
- `project status-projections --json` returned `target_kind = project_status_json`, `target_uri = .areaflow/status.json`, `summary_state = mirroring` and `write_state = written`.
- `project.status_projection.apply` command response records `execution_write_attempted=false` and `engine_call_attempted=false`.
- The smoke script verified the real AreaMatrix protected path git status set stayed empty.
- The temporary fixture directory was removed by script cleanup.
- The temporary PostgreSQL database was dropped after the smoke.

## Boundary

这份证据不证明：

- Re-running or widening real AreaMatrix status projection writes beyond the Package A `.areaflow/status.json`
  authorization.
- `workflow/README.md` controlled block write.
- Status projection write outside allowlisted `.areaflow/status.json`.
- Authoring cutover.
- Task-loop or execution replacement.
- Runner / worker execution.
- Web/Desktop behavior.

这些仍属于独立 backlog 项和后续 gate。
