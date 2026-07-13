# Run Control Evidence

## Purpose

本文记录 backlog 任务
[`AF-V05-002 Run Control Dry-run Boundary`](../plans/task-backlog.md#af-v05-002-run-control-dry-run-boundary)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库，验证 `run.start`、`run.drain` 和 `run.cancel` 只控制
dry-run run 的 AreaFlow DB 状态。它不领取 task、不创建 lease、不启动 worker、不执行 shell、
不调用 engine、不写被管理项目文件。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/project ./internal/app ./internal/api
go build -o /tmp/areaflow-v05-run-control-smoke ./cmd/areaflow
```

Result: pass

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v05ctrl_1782897301_54410
Project key: areamatrix
Workflow label: v05-control-20260701171501
```

Focused smoke path:

```text
migrate up
project add --config examples/areamatrix/areaflow.yaml
workflow version create areamatrix v05-control-20260701171501 --json
run preview areamatrix v05-control-20260701171501 --idempotency-key run-control-preview-v05-control-20260701171501 --json
fixture SQL: set the dry-run runner preview run to queued for protected control coverage
run start <run_id> --idempotency-key run-control-start-v05-control-20260701171501 --json
run start <run_id> --idempotency-key run-control-start-v05-control-20260701171501 --json
run drain <run_id> --idempotency-key run-control-drain-v05-control-20260701171501 --json
run cancel <run_id> --idempotency-key run-control-cancel-v05-control-20260701171501 --json
fixture SQL: insert a non-dry-run queued run
run start <non_dry_run_id> --idempotency-key run-control-nondry-v05-control-20260701171501 --json
```

Cleanup:

```text
DROP DATABASE af_v05ctrl_1782897301_54410
residual_connections=0
removed generated local artifact directory:
~/.areaflow/artifacts/areamatrix/v05-control-20260701171501
```

## Result

Status: pass

Observed proof:

```text
db=af_v05ctrl_1782897301_54410 label=v05-control-20260701171501 run_id=1 non_dry_run_id=2
start_created=true start_replay_created=false
start=running|queued|allowed|false|false|false drain=draining|running|allowed cancel=cancelling|draining|allowed
allowed_command_requests=3|true|true|true|true|true|true|true|true|true|true
denied_command_request=run.start|true|denied|false|false|false|false|false|false
run_counts=cancelling|true|1|2|1|0
event_audit_counts=3|3
residual_connections=0
```

`allowed_command_requests` fields are:

```text
count
all_completed
all_decision_allowed
all_project_write_attempted_false
all_execution_write_attempted_false
all_engine_call_attempted_false
all_task_claimed_false
all_worker_started_false
all_commands_run_false
all_secrets_resolved_false
all_network_used_false
```

`run_counts` fields are:

```text
run.status
run.dry_run
run_task_count
run_attempt_count
artifact_count
lease_count
```

## Evidence

- `go test ./internal/project ./internal/app ./internal/api` passed.
- `go build -o /tmp/areaflow-v05-run-control-smoke ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- `run.start` moved a dry-run run from `queued` to `running`.
- Repeating `run.start` with the same idempotency key returned `created=false`.
- `run.drain` moved the dry-run run from `running` to `draining`.
- `run.cancel` moved the dry-run run from `draining` to `cancelling`.
- Three allowed command requests were completed for `run.start`, `run.drain` and `run.cancel`.
- Allowed command responses recorded `project_write_attempted=false`, `execution_write_attempted=false`,
  `area_matrix_write_attempted=false`, `engine_call_attempted=false`, `task_claimed=false`,
  `worker_started=false`, `commands_run=false`, `secrets_resolved=false` and `network_used=false`.
- A non-dry-run queued run was blocked by `run control blocked: protected run control is only enabled for dry-run runs`.
- The denied command request was completed with `decision=denied`, `dry_run=false` and no forbidden action attempts.
- The controlled run retained one run_task, two original runner preview attempts, one original runner preview artifact
  and zero leases.
- Three run control events and three run control audit events were recorded.
- Temporary PostgreSQL database was dropped and residual connection count was `0`.
- Generated local artifact directory for this smoke label was removed after the smoke.

## Boundary

This proves v0.5 protected run control only. It does not prove worker drain/cancel,
real execution cancellation, process interruption, checkpoint, repair, Codex CLI execution,
project file writes or execution cutover.

## Contract Delta

The v0.5 runner preview contract also requires run control responses to expose `area_matrix_write_attempted=false`.
This is covered by focused service/API/CLI tests, and the current PostgreSQL smoke assertions now require it in the
persisted `run.start` / `run.drain` / `run.cancel` `command_requests.response` proof output.
