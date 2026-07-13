# Authoring Cutover Apply Evidence

## Purpose

本文记录 backlog 任务
[`AF-V04-002 Authoring Cutover Apply`](../../tasks/backlog/0-100-platform-backlog.md#af-v04-002-authoring-cutover-apply)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库、临时 AreaMatrix-like fixture project 和临时 artifact store，
验证 `project.cutover.apply` 只在 AreaFlow PostgreSQL 中执行 authoring cutover，并留下 command
request、event、audit event 和 workflow version 状态。它不修改真实 AreaMatrix。

## Run

Date: 2026-07-01

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v04cut_1782891991_20547
Fixture project key: areamatrix-cutover-fixture
Fixture workflow version: cutover-smoke-20260701154631
Fixture root: temporary /private/var/folders/.../areaflow-cutover-fixture.*
Artifact store: temporary /private/var/folders/.../areaflow-cutover-fixture.*/artifact-store
```

Baseline command:

```bash
go test ./internal/project ./internal/app ./internal/api
```

Result: pass

Smoke steps:

```text
migrate up
project add
project import #1
project import #2
project doctor --json
project status-projection-apply
workflow version create
workflow version mark-ready --stage queue
workflow version mark-ready --stage promotion_preview
workflow gate run promotion_preview
workflow transition preview
workflow approval record --decision approved
workflow gate run approval_gate
workflow gate run live_mapping_gate
project cutover-readiness --version cutover-smoke-20260701154631
workflow gate run cutover_readiness_gate
project cutover-apply --version cutover-smoke-20260701154631 --idempotency-key cutover-smoke-apply-cutover-smoke-20260701154631
project cutover-apply --version cutover-smoke-20260701154631 --idempotency-key cutover-smoke-apply-cutover-smoke-20260701154631
```

Cleanup:

```text
cleanup_db=af_v04cut_1782891991_20547 residual=0
```

## Result

Status: pass

First apply:

```text
applied|allowed|true|false|false|4
```

字段含义：

- `status=applied`
- `decision=allowed`
- `created=true`
- `project_write_attempted=false`
- `execution_write_attempted=false`
- `cutover_readiness_gate_id=4`

Idempotent replay:

```text
applied|allowed|false|cutover-smoke-apply-cutover-smoke-20260701154631
```

字段含义：

- repeated command returned the existing response.
- `created=false`
- idempotency key matched the first command.

Database proof:

```text
authoring_cutover|true|areaflow|project|false|false|1|1|1|1|1|1
```

字段含义：

- workflow version `lifecycle_status=authoring_cutover`.
- `status_summary.authoring_cutover.applied=true`.
- `workflow_owner=areaflow`.
- `execution_owner=project`.
- `status_summary.authoring_cutover.project_write_attempted=false`.
- `status_summary.authoring_cutover.execution_write_attempted=false`.
- 1 completed `project.cutover.apply` command request.
- 1 `project.cutover.apply.completed` event.
- 1 `project.cutover.apply` audit event with `decision=allowed`.
- 1 passing `cutover_readiness_gate`.
- 1 command response with `project_write_attempted=false`.
- 1 command response with `execution_write_attempted=false`.

## Evidence

- `go test ./internal/project ./internal/app ./internal/api` passed.
- Migrations `000001` through `000010` applied from an empty temporary PostgreSQL database.
- Authoring cutover was attempted only after promotion preview, transition preview, approval, approval gate,
  live mapping gate, cutover readiness and `cutover_readiness_gate` passed.
- `project.cutover.apply` updated the AreaFlow workflow version lifecycle to `authoring_cutover`.
- The workflow version summary records `workflow_owner=areaflow` and `execution_owner=project`.
- The command response records `project_write_attempted=false` and `execution_write_attempted=false`.
- Repeating the same idempotency key returned the same business result with `created=false`.
- Event and audit facts were inserted for the apply operation.
- The temporary PostgreSQL database was dropped after the smoke and residual database count was `0`.

## Boundary

这份证据不证明：

- Real AreaMatrix authoring cutover.
- Real AreaMatrix shim file edits.
- `workflow/README.md` controlled block write.
- Execution cutover.
- `./task-loop run` replacement.
- Real project file writes.
- Runner / worker copy, verify, repair or checkpoint execution.
- Secret resolution or AI engine calls.
- Web/Desktop behavior.

`project.cutover.apply` 在 v0.4 中只表示 AreaFlow DB 内的新 workflow version authoring 所有权切换。
真实 AreaMatrix 文件写入和 execution cutover 必须走后续独立 approval / permission / audit。
