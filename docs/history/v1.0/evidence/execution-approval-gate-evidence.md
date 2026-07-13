# Execution Approval Gate Evidence

## Purpose

本文记录 backlog 任务
[`AF-V06-003 Execution Approval Gate`](../plans/task-backlog.md#af-v06-003-execution-approval-gate)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库，验证真实 execution apply 前的只读 approval gate 能读取 run、
run_task、approval/gate 状态、engine preview 和 worker metadata，并在 dry-run preview run 上保持 blocked。
它不创建 command request、不领取 task、不启动 worker、不创建 attempt/artifact、不调用 engine、不运行 shell、
不解析 secret、不写被管理项目文件。

## Run

Date: 2026-07-02

Baseline commands:

```bash
gofmt -w internal/project/execution_approval_gate.go internal/project/execution_approval_gate_test.go internal/api/server.go internal/api/server_test.go internal/app/app.go internal/app/app_test.go
go test ./internal/project ./internal/api ./internal/app
go build -o /tmp/areaflow-v06-execution-gate-smoke ./cmd/areaflow
```

Result: pass

Environment:

```text
PostgreSQL: docker compose service postgres, container areaflow-postgres, localhost:54329
Fixture database: af_v06gate_20260702001728_18751
Project key: areamatrix
Workflow label: v06-gate-20260702001728
Worker key: v06-gate-worker
Migrations applied: 10
```

Focused smoke path:

```text
CREATE DATABASE af_v06gate_20260702001728_18751
migrate up
project add --config examples/areamatrix/areaflow.yaml
workflow version create areamatrix v06-gate-20260702001728 --json
run preview areamatrix v06-gate-20260702001728 --idempotency-key execution-gate-preview-v06-gate-20260702001728 --json
worker register areamatrix --worker-key v06-gate-worker --capability read_project --capability write_artifacts --capability run_commands --capability execute_agents --json
run execution-gate 1 --json
GET /api/v1/runs/1/execution-approval-gate
DROP DATABASE af_v06gate_20260702001728_18751
```

Cleanup:

```text
connections_before_drop=0
connections_after_drop=0
removed /tmp/areaflow-v06-execution-gate-smoke
removed generated local artifact directory:
~/.areaflow/artifacts/areamatrix/v06-gate-20260702001728
```

## Result

Status: pass

Observed proof:

```text
execution_gate_cli=blocked|dry_run_boundary=blocked|read_only_boundary=pass|project_write_attempted=false|execution_write_attempted=false|engine_call_attempted=false|commands_run=false|secrets_resolved=false|network_used=false|task_claimed=false|worker_started=false|attempt_created=false|artifact_created=false
execution_gate_api=blocked|dry_run_boundary=blocked|read_only_boundary=pass|project_write_attempted=false|task_claimed=false|attempt_created=false|artifact_created=false
counts_before_second_gate=3|1|1|2|10|0|1
counts_after_second_gate=3|1|1|2|10|0|1
counts_after_api_gate=3|1|1|2|10|0|1
residual_connections=0
```

Count fields are:

```text
command_requests
runs
run_tasks
run_attempts
artifacts
leases
worker_heartbeats
```

## Evidence

- `go test ./internal/project ./internal/api ./internal/app` passed.
- `go build -o /tmp/areaflow-v06-execution-gate-smoke ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- `project add --config examples/areamatrix/areaflow.yaml` registered AreaMatrix metadata only.
- `workflow version create` produced an AreaFlow-authored workflow version.
- `run preview` produced a dry-run execution run with queued run task, dry-run attempts and runner preview artifact metadata.
- `worker register` produced one online local worker with `read_project`, `write_artifacts`, `run_commands` and `execute_agents`.
- `run execution-gate 1 --json` returned `status=blocked`.
- The CLI response blocked `dry_run_boundary` and `run_status`, while `read_only_boundary` passed.
- The CLI response returned `project_write_attempted=false`, `execution_write_attempted=false`,
  `engine_call_attempted=false`, `commands_run=false`, `secrets_resolved=false`, `network_used=false`,
  `task_claimed=false`, `worker_started=false`, `attempt_created=false` and `artifact_created=false`.
- Repeating `run execution-gate 1 --json` did not change counts for command requests, runs, run tasks, attempts,
  artifacts, leases or worker heartbeats.
- `GET /api/v1/runs/1/execution-approval-gate` returned the same blocked/read-only boundary over the versioned API alias.
- The API request did not change counts for command requests, runs, run tasks, attempts, artifacts, leases or worker heartbeats.
- Temporary PostgreSQL database was dropped, residual connection count was `0`, the `/tmp` smoke binary was removed,
  and the generated local artifact directory for this workflow label was removed.

## Boundary

This proves v0.6 execution approval gate only. It does not prove real execution apply, worker task claim,
copy/verify/repair/checkpoint execution, Codex CLI execution, secret resolution, shell command execution,
managed project writes, execution cutover or AreaMatrix task-loop replacement.
