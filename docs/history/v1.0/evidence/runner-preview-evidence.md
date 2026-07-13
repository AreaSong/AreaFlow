# Runner Preview Evidence

## Purpose

本文记录 backlog 任务
[`AF-V05-001 Runner Preview Evidence`](../plans/task-backlog.md#af-v05-001-runner-preview-evidence)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库和 AreaFlow-owned local artifact store，验证 `runner.preview`
Command API 能形成 run、run_task、run_attempt、artifact、event、audit 和 completed command response
的 dry-run 闭环。它不执行 shell、不调用 engine、不写被管理项目文件。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/project ./internal/app ./internal/api
go build ./cmd/areaflow
```

Result: pass

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v05_1782896075_13537
Project key: areamatrix
Workflow label: v05-smoke-20260701165436
Artifact store: ~/.areaflow/artifacts/areamatrix/v05-smoke-20260701165436
```

Focused smoke path:

```text
migrate up
project add --config examples/areamatrix/areaflow.yaml
workflow version create areamatrix v05-smoke-20260701165436 --json
run preview areamatrix v05-smoke-20260701165436 --idempotency-key runner-preview-v05-smoke-20260701165436 --json
run preview areamatrix v05-smoke-20260701165436 --idempotency-key runner-preview-v05-smoke-20260701165436 --json
run preview areamatrix v05-smoke-20260701165436 --risk-level high --json
```

Cleanup:

```text
DROP DATABASE af_v05_1782896075_13537
residual_connections=0
removed generated local artifact directory:
~/.areaflow/artifacts/areamatrix/v05-smoke-20260701165436
```

## Result

Status: pass

Observed proof:

```text
db=af_v05_1782896075_13537 label=v05-smoke-20260701165436 run_id=1 artifact_id=10
created_first=true created_replay=false
command_request=runner.preview|true|1|passed|true|runner_preview_report|true|false|false|false|false|false|false|3|4
run_counts=passed|true|1|2|1
event_count=1 audit_count=1
artifact_sha_match=true artifact_size_match=true
residual_connections=0
```

`command_request` fields are:

```text
command_type
completed_at_is_present
response.run_id
response.run_status
response.dry_run
response.artifact_type
response.artifact_sha256_present
response.project_write_attempted
response.execution_write_attempted
response.engine_call_attempted
response.commands_run
response.secrets_resolved
response.network_used
response.event_id
response.audit_event_id
```

## Evidence

- `go test ./internal/project ./internal/app ./internal/api` passed.
- `go build ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- `workflow version create` produced an AreaFlow-authored workflow version suitable for runner preview.
- First `run preview` returned `created=true`; repeated `run preview` with the same idempotency key returned `created=false`.
- `runner.preview` command request was completed and persisted response evidence.
- Command response included `run_id`, `run_status=passed`, `dry_run=true`, `artifact_type=runner_preview_report`,
  non-empty artifact SHA, event ID and audit event ID.
- Command response explicitly recorded `project_write_attempted=false`, `execution_write_attempted=false`,
  `area_matrix_write_attempted=false`, `engine_call_attempted=false`, `commands_run=false`,
  `secrets_resolved=false` and `network_used=false`.
- Run evidence contained one `run_task`, two `run_attempts` and one `runner_preview_report` artifact.
- `runner_preview_report` local artifact hash and size matched the PostgreSQL artifact metadata before cleanup.
- High-risk runner preview without `risk_policy=allow` was blocked by `runner preview blocked: risk_gate`.
- Temporary PostgreSQL database was dropped and residual connection count was `0`.
- Generated local artifact directory for this smoke label was removed after integrity verification.

## Boundary

This proves v0.5 dry-run runner preview only. It does not prove real `runner.run`,
Codex CLI execution, project file writes, checkpoint, repair, worker execution beta or execution cutover.

## Contract Delta

The v0.5 runner preview contract treats `area_matrix_write_attempted=false` as a required safety fact. Focused tests
cover it, and the current PostgreSQL smoke assertions now require it in the persisted `runner.preview`
`command_requests.response` proof output.
