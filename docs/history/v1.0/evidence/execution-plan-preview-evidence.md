# Execution Plan Preview Evidence

## Purpose

本文记录 backlog 任务 `AF-V06-007 Execution Plan Preview` 的最近一次本机验证证据。

该证据覆盖真实 copy / verify / repair / checkpoint 打开前的只读执行计划预览：

```text
execution approval gate
-> execution plan preview
-> show approved artifact write as the only currently opened artifact-only step
-> keep copy / checkpoint / repair blocked or waiting
```

本阶段只读取 AreaFlow PostgreSQL 中的 run detail、run_task metadata 和 execution approval gate。它不创建
`command_requests`，不领取 task，不启动 worker，不创建 lease/attempt/artifact，不读取或写入被管理项目，
不调用 engine，不运行 shell，不解析 secret，不访问网络。

## Run

Date: 2026-07-02

Baseline commands:

```bash
gofmt -w internal/project/execution_plan.go internal/project/execution_plan_test.go internal/api/server.go internal/api/server_test.go internal/app/app.go internal/app/app_test.go
go test ./internal/project ./internal/api ./internal/app
bash -n scripts/smoke-execution-plan.sh
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  scripts/smoke-execution-plan.sh
go test ./...
git diff --check -- .
```

Makefile smoke entry:

```bash
make smoke-docker-execution-plan
```

Result: pass

## Covered Paths

- `internal/project/execution_plan.go`
- `internal/project/execution_plan_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `scripts/smoke-execution-plan.sh`

## CLI / API

```text
GET /api/v1/runs/{run_id}/execution-plan
areaflow run execution-plan <run-id>
areaflow run execution-plan <run-id> --json
```

## Expected Result

Execution plan preview returns:

```text
mode=read_only_execution_plan_preview
status=blocked while real copy/checkpoint/repair are unopened
execution_approval_gate ready when gate passes
approved_artifact_write ready when gate passes
copy blocked
checkpoint blocked
verify waiting
repair waiting
```

Safety facts:

```text
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
attempt_created=false
artifact_created=false
```

## Evidence

- `go test ./internal/app` passed after adding CLI support.
- `go test ./internal/project ./internal/api ./internal/app` passed.
- `bash -n scripts/smoke-execution-plan.sh` passed.
- PostgreSQL fixture smoke passed on a temporary database.
- 2026-07-06 CST `make smoke-docker-execution-plan` passed on isolated PostgreSQL database
  `areaflow_smoke_20260706022030_47926`; `scripts/smoke-docker.sh` created and dropped the database.
- By default the fixture smoke does not read real AreaMatrix projection files; callers can opt in to the
  extra `.areaflow/status.json` / `workflow/README.md` fingerprint guard with
  `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`.
- `go test ./...` passed.
- `git diff --check -- .` passed.
- Legacy execution-state safety typo search returned no matches.
- Project-layer tests cover read-only preview construction and gate blocker propagation.
- API-layer tests cover `GET /api/v1/runs/{run_id}/execution-plan`.
- CLI-layer tests cover help output, `--json` flag parsing, JSON conversion and read-only safety facts.

## PostgreSQL Fixture Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: temporary areaflow_smoke_v06l_* database
Fixture project key: areamatrix-execution-plan-fixture
Fixture root: temporary /private/var/folders/.../areaflow-execution-plan.*
Artifact store: temporary /private/var/folders/.../areaflow-execution-plan.*/artifact-store
```

Smoke sequence:

```bash
areaflow migrate up
areaflow project add --config <fixture>/areaflow.yaml
areaflow workflow version create areamatrix-execution-plan-fixture <version> --json
areaflow workflow version mark-ready areamatrix-execution-plan-fixture <version> --stage queue --item-type queue_candidate --json
areaflow workflow version mark-ready areamatrix-execution-plan-fixture <version> --stage promotion_preview --item-type promotion_preview --json
areaflow workflow gate run areamatrix-execution-plan-fixture <version> promotion_preview --json
areaflow workflow transition preview areamatrix-execution-plan-fixture <version> --json
areaflow workflow approval record areamatrix-execution-plan-fixture <version> --decision approved --json
areaflow workflow gate run areamatrix-execution-plan-fixture <version> approval_gate --json
areaflow workflow gate run areamatrix-execution-plan-fixture <version> live_mapping_gate --json
areaflow worker register areamatrix-execution-plan-fixture --worker-key <worker> --capability write_artifacts --json
areaflow run approved-artifact-write-queue areamatrix-execution-plan-fixture <version> --artifact-label plan-preview --json
areaflow run execution-plan <run_id> --json
areaflow run execution-plan <run_id>
```

Smoke asserted:

```text
mode=read_only_execution_plan_preview
status=blocked
gate.status=pass
execution_approval_gate.status=ready
approved_artifact_write.status=ready
copy.status=blocked
checkpoint.status=blocked
repair.status=waiting
copy.writes_project=true
copy.uses_engine=true
copy.runs_commands=true
approved_artifact_write.writes_project=false
approved_artifact_write.writes_areaflow=true
approved_artifact_write.creates_artifact=true
```

Safety facts:

```text
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
attempt_created=false
artifact_created=false
```

Mutation check:

```text
summary|8|1|1|0|0|11|11|12|1|8|1|1|0|0|11|11|12|1
```

The two tuple groups are counts before and after `areaflow run execution-plan`. They matched exactly for
`command_requests`, `runs`, `run_tasks`, `leases`, `run_attempts`, `artifacts`, `events`, `audit_events`
and `worker_heartbeats`.

Cleanup proof:

```text
temporary fixture directory removed
temporary database dropped
post-drop database count=0
```

## Boundary

Execution plan preview is a Query API / CLI read-only surface. It does not open true copy, true verify,
repair, checkpoint, Codex CLI execution, project file writes, secret resolution, network access, git checkpoint
or AreaMatrix execution cutover.
