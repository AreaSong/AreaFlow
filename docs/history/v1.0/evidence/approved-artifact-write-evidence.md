# Approved Artifact Write Evidence

## Purpose

本文记录 backlog 任务 `AF-V06-006 Approved Artifact Write` 的最近一次本机验证证据。

该证据覆盖 approval-gated worker 在不读取/写入被管理项目、不调用 engine 的前提下，写入
AreaFlow-owned artifact store 的最小 PostgreSQL 事实链：

```text
execution gate pass
-> worker claim / lease
-> AreaFlow-owned artifact write
-> approved_artifact_write attempt
-> run_task artifact_written
-> run artifact_written
```

本阶段只写 AreaFlow PG state、event/audit、command response 和本地 artifact store。它不读取项目文件、
不写真实 AreaMatrix、不运行 shell、不调用 Codex CLI / engine、不解析 secret、不访问网络。

## Run

Date: 2026-07-02

Baseline commands:

```bash
gofmt -w internal/project/approved_artifact_write.go internal/api/server.go internal/api/server_test.go internal/app/app.go internal/app/app_test.go
go test ./internal/project ./internal/api ./internal/app
bash -n scripts/smoke-approved-artifact-write.sh
```

Result: pass

## PostgreSQL Fixture Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: areaflow_smoke_v06k_20260702032213 temporary database
Fixture project key: areamatrix-artifact-fixture
Fixture root: temporary /private/var/folders/.../areaflow-artifact-write.*
Artifact store: temporary /private/var/folders/.../areaflow-artifact-write.*/artifact-store
```

Smoke command:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow_smoke_v06k_20260702032213?sslmode=disable \
  scripts/smoke-approved-artifact-write.sh
```

Makefile smoke entry:

```bash
make smoke-docker-approved-artifact-write
```

Smoke sequence:

```bash
areaflow migrate up
areaflow project add --config <fixture>/areaflow.yaml
areaflow project import areamatrix-artifact-fixture
areaflow workflow version create areamatrix-artifact-fixture <version> --json
areaflow workflow version mark-ready areamatrix-artifact-fixture <version> --stage queue --item-type queue_candidate --json
areaflow workflow version mark-ready areamatrix-artifact-fixture <version> --stage promotion_preview --item-type promotion_preview --json
areaflow workflow gate run areamatrix-artifact-fixture <version> promotion_preview --json
areaflow workflow transition preview areamatrix-artifact-fixture <version> --json
areaflow workflow approval record areamatrix-artifact-fixture <version> --decision approved --json
areaflow workflow gate run areamatrix-artifact-fixture <version> approval_gate --json
areaflow workflow gate run areamatrix-artifact-fixture <version> live_mapping_gate --json
areaflow worker register areamatrix-artifact-fixture --worker-key <worker> --capability write_artifacts --json
areaflow run approved-artifact-write-queue areamatrix-artifact-fixture <version> --artifact-label approval-note --json
areaflow worker approved-artifact-write areamatrix-artifact-fixture <worker> --run-id <run_id> --capability write_artifacts --json
areaflow worker approved-artifact-write areamatrix-artifact-fixture <worker> --run-id <run_id> --capability write_artifacts --idempotency-key <same_key> --json
```

The fixture config enabled `write_artifacts=true` and kept `execute_agents=false`, `run_commands=false`,
`network=false`, and `use_secrets=false`. The worker was registered with only `write_artifacts`.

## Result

Status: pass

Final database summary:

```text
summary|1|1|1|1|1|1|1|1|1
```

字段含义：

- 1 non-dry-run `approved_artifact_write` run with `status=artifact_written`.
- 1 `approved_artifact_write_task` with `status=artifact_written`.
- 1 completed `approved_artifact_write` lease.
- 1 passed non-dry-run `approved_artifact_write` attempt.
- 1 local `approved_artifact_write_report` artifact metadata record.
- 1 completed `run.approved_artifact_write_queue` command response.
- 1 completed `worker.approved_artifact_write` command response.
- 1 `worker.approved_artifact_write.allowed` event.
- 1 `write_artifacts` audit event.

The worker response proved:

```text
status=artifact_written
decision=allowed
run.status=artifact_written
task.status=artifact_written
lease.status=completed
attempt.attempt_kind=approved_artifact_write
attempt.status=passed
attempt.dry_run=false
artifact.artifact_type=approved_artifact_write_report
artifact_label=approval-note
```

Safety facts:

```text
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=true
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
worker_started=false
artifact_write_passed=true
```

Idempotent replay:

```text
run.approved_artifact_write_queue replay returned created=false.
worker.approved_artifact_write replay returned created=false, status=artifact_written, decision=allowed.
```

Cleanup proof:

```text
residual_connections=0
temporary fixture directory removed
temporary database dropped
post-drop database count=0
```

## Evidence

- `go test ./internal/project ./internal/api ./internal/app` passed.
- `bash -n scripts/smoke-approved-artifact-write.sh` passed.
- PostgreSQL fixture smoke passed on a temporary DB.
- 2026-07-06 CST `make smoke-docker-approved-artifact-write` passed on isolated PostgreSQL database
  `areaflow_smoke_20260706022010_46159`; `scripts/smoke-docker.sh` created and dropped the database.
- The smoke asserted one passed non-dry-run `approved_artifact_write` attempt and one completed
  `approved_artifact_write` lease.
- The smoke asserted one `approved_artifact_write_report` artifact metadata record and verified the local
  artifact file stayed under the fixture artifact store.
- The smoke asserted command response safety facts for `run.approved_artifact_write_queue` and
  `worker.approved_artifact_write`.
- The smoke asserted no `workflow/versions/<version>/execution/**` directory was created in the fixture project.
- By default the smoke does not read real AreaMatrix projection files. Set
  `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` only when the caller intentionally wants the extra
  `.areaflow/status.json` / `workflow/README.md` fingerprint guard.
