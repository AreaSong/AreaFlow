# Read-only Verify Evidence

## Purpose

本文记录 backlog 任务
`AF-V06-005 Read-only Verify` 的最近一次本机验证证据。

该证据覆盖 approval-gated read-only verify 的 PostgreSQL 事实链：

```text
execution gate pass
-> worker claim / lease
-> allowlisted project file read
-> evidence artifact
-> run_task verified
-> run verified
```

本阶段仍不写真实 AreaMatrix，不运行 shell，不调用 Codex CLI / engine，不解析 secret，不访问网络。
read-only verify 只读取 project config allowlist 允许的文件，并在 artifact 中保存 path、sha256 和 size，
不保存被读取文件原文。

## Run

Date: 2026-07-02

Baseline commands:

```bash
gofmt -w internal/project/read_only_verify.go internal/project/read_only_verify_test.go internal/api/server.go internal/api/server_test.go internal/app/app.go internal/app/app_test.go
go test ./internal/project ./internal/api ./internal/app
go build -o /tmp/areaflow-v06j-readonly-smoke ./cmd/areaflow
```

Result: pass

## PostgreSQL Fixture Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v06j_20260702... temporary database
Fixture project key: areamatrix-fixture-v06j
Fixture root: temporary /private/var/folders/.../areaflow-v06j-readonly.*
Artifact store: temporary /private/var/folders/.../areaflow-v06j-readonly.*/artifact-store
Target file: docs/README.md
```

Smoke sequence:

```bash
areaflow migrate up
areaflow project add --config <fixture>/areaflow.yaml
areaflow workflow version create areamatrix-fixture-v06j <version> --json
areaflow workflow version mark-ready areamatrix-fixture-v06j <version> --stage queue --item-type queue_candidate --json
areaflow workflow version mark-ready areamatrix-fixture-v06j <version> --stage promotion_preview --item-type promotion_preview --json
areaflow workflow gate run areamatrix-fixture-v06j <version> promotion_preview --json
areaflow workflow transition preview areamatrix-fixture-v06j <version> --json
areaflow workflow approval record areamatrix-fixture-v06j <version> --decision approved --transition-preview-id <ready_preview_id> --json
areaflow workflow gate run areamatrix-fixture-v06j <version> approval_gate --json
areaflow workflow gate run areamatrix-fixture-v06j <version> live_mapping_gate --json
areaflow worker register areamatrix-fixture-v06j --worker-key local-1 --capability read_project --capability write_artifacts --json
areaflow run read-only-verify-queue areamatrix-fixture-v06j <version> --target-path docs/README.md --idempotency-key <key> --json
areaflow worker read-only-verify areamatrix-fixture-v06j local-1 --run-id <run_id> --idempotency-key <key> --json
areaflow worker read-only-verify areamatrix-fixture-v06j local-1 --run-id <run_id> --idempotency-key <same_key> --json
```

The fixture config enabled `read_project=true` and `write_artifacts=true`, allowed `docs/**`, and kept
`execute_agents=false`, `run_commands=false`, `network=false`, and `use_secrets=false`.

## Result

Status: pass

Final database summary:

```text
summary|9|3|1|1|1|1|1|1|1
```

字段含义：

- 9 completed command requests.
- 3 gate results: `promotion_preview`, `approval_gate`, `live_mapping_gate`.
- 1 transition preview.
- 1 approval record.
- 1 read-only verify run.
- 1 verified `read_only_verify_task`.
- 1 completed `read_only_verify` lease.
- 1 passed `read_only_verify` attempt.
- 1 `read_only_verify_report` artifact metadata record.

The read-only verify response proved:

```text
status=verified
decision=allowed
run.status=verified
task.status=verified
lease.status=completed
attempt.attempt_kind=read_only_verify
attempt.status=passed
attempt.dry_run=false
artifact.artifact_type=read_only_verify_report
target_sha256=74c1b32ac7459d876bed87a593f77d50eb604d80b07483516915511fa492b7dc
```

Safety facts:

```text
project_read_attempted=true
project_read_allowed=true
project_write_attempted=false
execution_write_attempted=false
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
worker_started=false
verification_passed=true
```

Idempotent replay:

```text
worker.read_only_verify replay returned created=false, status=verified, decision=allowed.
```

Cleanup proof:

```text
residual_connections=0
temporary fixture directory removed
temporary binary /tmp/areaflow-v06j-readonly-smoke removed
temporary /tmp/af_v06j_* smoke output files removed
```

## Evidence

- `go test ./internal/project ./internal/api ./internal/app` passed.
- Temporary binary build passed.
- PostgreSQL fixture smoke passed on a temporary DB.
- The smoke asserted one passed non-dry-run `read_only_verify` attempt and one completed
  `read_only_verify` lease.
- The smoke asserted one `read_only_verify_report` artifact metadata record and verified the target file hash.
- The smoke asserted command response safety facts for `worker.read_only_verify`.
- The smoke dropped the temporary database and verified zero residual connections.
