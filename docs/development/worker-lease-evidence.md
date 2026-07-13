# Worker Lease Evidence

## Purpose

本文记录 backlog 任务
[`AF-V06-001 Worker Registry Lease Lifecycle`](../../tasks/backlog/0-100-platform-backlog.md#af-v06-001-worker-registry-lease-lifecycle)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库，验证 worker register、heartbeat、显式幂等 replay / conflict、
lease acquire、lease release、lease recover 和 capability denial。它不执行 copy/verify、不调用 Codex CLI、
不调用 engine、不写被管理项目文件。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/project ./internal/app ./internal/api
go build -o /tmp/areaflow-v06-worker-lifecycle-smoke ./cmd/areaflow
```

Result: pass

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v06worker_20260701234205_31842
Project key: areamatrix
Workflow label: v06-worker-20260701234205
Worker key: v06-worker-20260701234205-worker
Migrations applied: 10
```

Focused smoke path:

```text
migrate up
project add --config examples/areamatrix/areaflow.yaml
workflow version create areamatrix v06-worker-20260701234205 --json
run preview areamatrix v06-worker-20260701234205 --idempotency-key preview-v06-worker-20260701234205 --json
worker register areamatrix --worker-key v06-worker-20260701234205-worker --capability read_project --capability write_artifacts --idempotency-key worker-register-v06-worker-20260701234205 --json
worker register areamatrix --worker-key v06-worker-20260701234205-worker --capability read_project --capability write_artifacts --idempotency-key worker-register-v06-worker-20260701234205 --json
worker register conflict check: same idempotency key + different worker_type returned non-zero
worker heartbeat areamatrix v06-worker-20260701234205-worker --status online --idempotency-key worker-heartbeat-v06-worker-20260701234205 --json
worker heartbeat areamatrix v06-worker-20260701234205-worker --status online --idempotency-key worker-heartbeat-v06-worker-20260701234205 --json
worker heartbeat conflict check: same idempotency key + different status returned non-zero
worker lease-acquire areamatrix v06-worker-20260701234205-worker --run-task-id 1 --capability read_project --capability write_artifacts --idempotency-key lease-acquire-v06-worker-20260701234205-1 --json
worker lease-acquire areamatrix v06-worker-20260701234205-worker --run-task-id 1 --capability read_project --capability write_artifacts --idempotency-key lease-acquire-v06-worker-20260701234205-1 --json
worker lease-release areamatrix v06-worker-20260701234205-worker --lease-id 1 --status released --idempotency-key lease-release-v06-worker-20260701234205-1 --json
worker lease-acquire areamatrix v06-worker-20260701234205-worker --run-task-id 1 --capability read_project --capability write_artifacts --idempotency-key lease-acquire-v06-worker-20260701234205-2 --json
fixture SQL: expire lease 2
worker lease-recover areamatrix --limit 5 --idempotency-key lease-recover-v06-worker-20260701234205 --json
worker lease-acquire areamatrix v06-worker-20260701234205-worker --run-task-id 1 --capability read_project --capability execute_agents --idempotency-key lease-denied-v06-worker-20260701234205 --json
```

Cleanup:

```text
DROP DATABASE af_v06worker_20260701234205_31842
residual_connections=0
removed generated local artifact directory:
~/.areaflow/artifacts/areamatrix/v06-worker-20260701234205
```

## Result

Status: pass

Observed proof:

```text
db=af_v06worker_20260701234205_31842 label=v06-worker-20260701234205 migrations=10 worker_id=1/1 heartbeat_replay_worker_id=1/1 run_id=1 run_task_id=1
register_conflict_status=1 heartbeat_conflict_status=1
lease_id=1 lease_replay_id=1 second_lease_id=2 recovered_count=1 denied_status=1 run_task_status=needs_recovery
worker_lifecycle_command_counts=2|2|2|2|2|2|2|2|2|2|2|2
worker_state_counts=1|2|2|2
lease_command_counts=5|1|1|5|5|5
worker_run_once_counts=0|0
residual_connections=0
```

`worker_lifecycle_command_counts` fields are:

```text
count
completed_count
project_write_attempted_false_count
execution_write_attempted_false_count
engine_call_attempted_false_count
commands_run_false_count
secrets_resolved_false_count
network_used_false_count
lease_created_false_count
attempt_created_false_count
artifact_created_false_count
worker_run_once_false_count
```

`worker_state_counts` fields are:

```text
worker_count
worker_heartbeat_count
worker_lifecycle_event_count
worker_lifecycle_audit_count
```

`lease_command_counts` fields are:

```text
completed_lease_command_count
denied_lease_acquire_count
lease_created_false_count
attempt_created_false_count
artifact_created_false_count
worker_run_once_false_count
```

`worker_run_once_counts` fields are:

```text
worker_run_once_attempt_count
worker_run_once_artifact_count
```

## Evidence

- `go test ./internal/project ./internal/app ./internal/api` passed.
- `go build -o /tmp/areaflow-v06-worker-lifecycle-smoke ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- Worker register created an online worker and worker actor.
- Repeating `worker.register` with the same explicit idempotency key returned worker ID `1` and did not duplicate worker lifecycle state.
- Reusing the same `worker.register` idempotency key with a different worker type returned non-zero status.
- Worker heartbeat recorded heartbeat state.
- Repeating `worker.heartbeat` with the same explicit idempotency key returned worker ID `1` and did not duplicate worker lifecycle state.
- Reusing the same `worker.heartbeat` idempotency key with a different status returned non-zero status.
- Worker lifecycle command responses recorded `project_write_attempted=false`, `execution_write_attempted=false`,
  `engine_call_attempted=false`, `commands_run=false`, `secrets_resolved=false`, `network_used=false`,
  `lease_created=false`, `attempt_created=false`, `artifact_created=false` and `worker_run_once=false`.
- Worker lifecycle state counts were `1|2|2|2`: one worker, two heartbeat rows, two lifecycle events and two
  lifecycle audit events after replay and conflict checks.
- First `lease-acquire` created active lease `1`; repeating the same idempotency key returned the same lease ID.
- `lease-release` moved lease `1` to `released`.
- Second `lease-acquire` created active lease `2`; fixture SQL expired that lease.
- `lease-recover` moved exactly one expired lease to `needs_recovery`.
- Capability denial for missing `execute_agents` returned `worker capability denied`.
- Denied `lease.acquire` command request completed with `decision=denied`, `lease_created=false`,
  `attempt_created=false`, `artifact_created=false`, `engine_call_attempted=false` and `commands_run=false`.
- Lease denial did not change total lease / run_attempt / artifact counts.
- Allowed lease command responses recorded `project_write_attempted=false`, `execution_write_attempted=false`,
  `engine_call_attempted=false`, `commands_run=false`, `secrets_resolved=false`, `network_used=false`,
  `attempt_created=false`, `artifact_created=false` and `worker_run_once=false`.
- Final run_task status was `needs_recovery`; active lease count was `0`, released lease count was `1`,
  needs-recovery lease count was `1`.
- No `worker_run_once` attempt or `worker_run_once_report` artifact was created in this lease lifecycle smoke.
- Temporary PostgreSQL database was dropped and residual connection count was `0`.
- Generated local artifact directory for this smoke label was removed after the smoke.

## Boundary

This proves v0.6 worker registry, heartbeat and lease lifecycle only. It does not prove worker run-once,
Codex CLI adapter, copy/verify/repair/checkpoint execution, process interruption, managed project writes or
execution cutover.
