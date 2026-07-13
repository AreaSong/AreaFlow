# Fixture Execution Apply Evidence

## Purpose

本文记录 backlog 任务
[`AF-V06-004 Fixture Execution Apply`](../../tasks/backlog/0-100-platform-backlog.md#af-v06-004-fixture-execution-apply)
的最近一次本机验证证据。

该证据覆盖 approval-gated fixture execution apply 的完整 PostgreSQL 事实链：

```text
execution gate pass
-> worker claim / lease
-> attempt
-> evidence artifact
-> run_task passed
-> run passed
```

本阶段仍不写真实 AreaMatrix，不运行 shell，不调用 Codex CLI / engine，不解析 secret，不访问网络。

## Run

Date: 2026-07-02

Baseline commands:

```bash
gofmt -w internal/project/fixture_execution.go internal/project/fixture_execution_test.go internal/api/server.go internal/api/server_test.go internal/app/app.go internal/app/app_test.go
go test ./internal/project ./internal/api ./internal/app
go build -o /tmp/areaflow-v06i-fixture-smoke ./cmd/areaflow
```

Result: pass

## PostgreSQL Fixture Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v06i_20260702011537_35806
Fixture project key: areamatrix-fixture-v06i
Fixture version: v06i-011537
Fixture root: temporary /private/var/folders/.../areaflow-v06i-fixture.*
Artifact store: temporary /private/var/folders/.../areaflow-v06i-fixture.*/artifact-store
```

Smoke sequence:

```bash
areaflow migrate up
areaflow project add --config <fixture>/areaflow.yaml
areaflow workflow version create areamatrix-fixture-v06i v06i-011537 --json
areaflow workflow version mark-ready areamatrix-fixture-v06i v06i-011537 --stage queue --item-type queue_candidate --json
areaflow workflow version mark-ready areamatrix-fixture-v06i v06i-011537 --stage promotion_preview --item-type promotion_preview --json
areaflow workflow gate run areamatrix-fixture-v06i v06i-011537 promotion_preview --json
areaflow workflow transition preview areamatrix-fixture-v06i v06i-011537 --json
areaflow workflow approval record areamatrix-fixture-v06i v06i-011537 --decision approved --transition-preview-id <ready_preview_id> --json
areaflow workflow gate run areamatrix-fixture-v06i v06i-011537 approval_gate --json
areaflow workflow gate run areamatrix-fixture-v06i v06i-011537 live_mapping_gate --json
areaflow run fixture-queue areamatrix-fixture-v06i v06i-011537 --idempotency-key <key> --json
areaflow worker register areamatrix-fixture-v06i --worker-key local-1 --capability read_project --capability write_artifacts --capability run_commands --capability execute_agents --json
areaflow run execution-gate <run_id> --json
areaflow worker fixture-execute areamatrix-fixture-v06i local-1 --run-id <run_id> --idempotency-key <key> --json
areaflow worker fixture-execute areamatrix-fixture-v06i local-1 --run-id <run_id> --idempotency-key <same_key> --json
```

The fixture config intentionally enabled `execute_agents=true`, `run_commands=true`, `write_artifacts=true`,
`codex-cli.enabled=true`, and allowed `codex exec` so the execution approval gate could pass. The fixture
executor still did not invoke Codex CLI, run shell commands, resolve secrets, open network access, or write the
managed project.

## Result

Status: pass

Final database summary:

```text
summary|9|3|1|1|1|1|1|1|12
```

字段含义：

- 9 completed command requests.
- 3 gate results: `promotion_preview`, `approval_gate`, `live_mapping_gate`.
- 1 transition preview.
- 1 approval record.
- 1 fixture execution run.
- 1 fixture execution task.
- 1 completed `fixture_execution` lease.
- 1 passed `fixture_execution` attempt.
- 12 artifact files in the temporary AreaFlow artifact store.

The fixture execution apply response proved:

```text
status=passed
decision=allowed
run.status=passed
task.status=passed
lease.status=completed
attempt.attempt_kind=fixture_execution
attempt.status=passed
attempt.dry_run=false
artifact.artifact_type=fixture_execution_report
```

Safety facts:

```text
project_write_attempted=false
execution_write_attempted=false
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=true
lease_created=true
attempt_created=true
artifact_created=true
worker_started=false
```

Idempotent replay:

```text
worker.fixture_execute replay returned created=false, status=passed, decision=allowed.
command_requests for run.fixture_queue + worker.fixture_execute remained 2.
```

Cleanup proof:

```text
residual_connections=0
temporary fixture directory removed
temporary binary /tmp/areaflow-v06i-fixture-smoke removed
```

## Evidence

- `go test ./internal/project ./internal/api ./internal/app` passed.
- Temporary binary build passed.
- PostgreSQL fixture smoke passed on temporary DB `af_v06i_20260702011537_35806`.
- The smoke asserted one passed non-dry-run `fixture_execution` run/task/attempt and one completed
  `fixture_execution` lease.
- The smoke asserted one `fixture_execution_report` artifact file exists under the temporary AreaFlow artifact store.
- The smoke asserted command response safety facts for `worker.fixture_execute`.
- The smoke dropped the temporary database and verified zero residual connections.
