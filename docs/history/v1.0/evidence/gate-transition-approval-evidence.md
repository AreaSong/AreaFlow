# Gate Transition Approval Evidence

## Purpose

本文记录 backlog 任务
[`AF-V03-002 Gate Transition Approval Records`](../plans/task-backlog.md#af-v03-002-gate-transition-approval-records)
的最近一次本机验证证据。

该证据覆盖 gate result、transition preview 和 approval record 的 PostgreSQL 事实链。gate run、
transition preview 和 approval record 均进入 command request / event / audit 模型。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/project
go build ./cmd/areaflow
```

Result: pass

## Fixture Gate / Transition / Approval Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v03g2_1782890859_48421
Fixture project key: areamatrix-fixture
Fixture root: temporary /private/var/folders/.../areaflow-fixture.*
Artifact store: temporary /private/var/folders/.../areaflow-fixture.*/artifact-store
```

Commands:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_v03g2_1782890859_48421?sslmode=disable \
  AREAFLOW_KEEP_FIXTURE=1 ./scripts/smoke-fixture.sh
go run ./cmd/areaflow workflow version create areamatrix-fixture v2 --json
go run ./cmd/areaflow workflow gate run areamatrix-fixture v2 promotion_preview --json
go run ./cmd/areaflow workflow transition preview areamatrix-fixture v2 --json
go run ./cmd/areaflow workflow approval record areamatrix-fixture v2 --json --decision approved
go run ./cmd/areaflow workflow version mark-ready areamatrix-fixture v2 --stage queue --item-type queue_candidate --json
go run ./cmd/areaflow workflow version mark-ready areamatrix-fixture v2 --stage promotion_preview --item-type promotion_preview --json
go run ./cmd/areaflow workflow gate run areamatrix-fixture v2 promotion_preview --json
go run ./cmd/areaflow workflow transition preview areamatrix-fixture v2 --json
go run ./cmd/areaflow workflow approval record areamatrix-fixture v2 --json --decision approved --transition-preview-id <ready_preview_id>
go run ./cmd/areaflow workflow gate run areamatrix-fixture v2 approval_gate --json
go run ./cmd/areaflow workflow gate run areamatrix-fixture v2 live_mapping_gate --json
dropdb -h localhost -p 54329 -U areaflow --if-exists af_v03g2_1782890859_48421
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname LIKE 'af_v03g2_%';"
```

## Result

Status: pass

Initial blocked promotion preview gate:

```text
promotion_preview|fail|1|1
```

字段含义：

- gate name: `promotion_preview`
- gate status: `fail`
- failure count: `1`
- warning count: `1`

Blocked transition preview:

```text
blocked|promotion_preview|1|1|true
```

字段含义：

- transition preview status: `blocked`
- required gate: `promotion_preview`
- linked gate result id exists: `1`
- blocker count: `1`
- read-only warning is present: `true`

Approval before ready preview:

```text
approval transition preview is not ready: create a ready transition preview before approving
exit status 1
```

Approval cannot be recorded as approved before a ready transition preview exists.

Ready marker probes:

```text
queue|queue_candidate|ready|local|true
promotion_preview|promotion_preview|ready|local|true
```

字段含义：

- queue and promotion preview items were marked ready.
- ready marker artifacts used local AreaFlow artifact storage.
- ready marker artifact hashes exist.

Passing promotion preview gate:

```text
promotion_preview|pass|2|1
```

字段含义：

- gate name: `promotion_preview`
- gate status: `pass`
- source hashes count: `2`
- warning count: `1`

Ready transition preview:

```text
ready|promotion_preview|2|0|true
```

字段含义：

- transition preview status: `ready`
- required gate: `promotion_preview`
- linked gate result id exists: `2`
- blocker count: `0`
- read-only warning is present: `true`

Approval record:

```text
approved|2|false|ready
```

字段含义：

- approval decision: `approved`
- transition preview id: `2`
- approval is execution: `false`
- linked transition status: `ready`

Approval gate:

```text
approval_gate|pass|approved|true
```

字段含义：

- gate name: `approval_gate`
- gate status: `pass`
- approval decision: `approved`
- read-only warning is present: `true`

Live mapping gate:

```text
live_mapping_gate|pass|false|true
```

字段含义：

- gate name: `live_mapping_gate`
- gate status: `pass`
- `execution_write_attempted=false`
- read-only warning is present: `true`

Database proof:

```text
4|2|1|7|9|1|6|4|2
```

字段含义：

- 4 gate results.
- 2 transition previews.
- 1 approval record.
- 7 workflow events for gate / transition / approval.
- 9 audit events for approval, mark-ready, gate run and transition preview actions.
- 1 completed `workflow.approval.record` command request.
- 6 completed command requests for `workflow.gate.run` or `workflow.transition.preview`.
- 4 `workflow.gate.run` command responses record `execution_write_attempted=false`.
- 2 `workflow.transition.preview` command responses record `execution_write_attempted=false`.

Artifact store proof:

```text
artifact_files=11
```

The artifact files are written under the temporary AreaFlow artifact store, not the fixture project root.

Project file boundary:

```text
project file list unchanged before and after gate / transition / approval smoke
```

Cleanup query:

```text
0
```

`af_v03g2_%` 临时数据库已清理，无残留。

## Evidence

- `go test ./internal/project` passed.
- `go build ./cmd/areaflow` passed.
- Promotion preview gate records both failing and passing gate results.
- Transition preview records blocked and ready states, and keeps read-only warnings.
- Approval cannot be approved before a ready transition preview.
- Approval record stores `approval_is_execution=false`.
- Approval gate can pass after approved ready transition preview.
- Live mapping gate remains independent and records `execution_write_attempted=false`.
- Workflow events are written for gate checks, transition previews and approval records.
- Gate run, transition preview and approval record write completed command requests.
- Gate run, transition preview and approval record write audit events.
- The fixture project root file list is unchanged.
- The temporary PostgreSQL database and fixture directory were cleaned up after the smoke.

## Boundary

这份证据不证明：

- Promotion apply.
- Approval as execution.
- Execution materialization.
- Authoring cutover.
- Task-loop or execution replacement.
- Runner / worker execution.
- Real AreaMatrix workflow directory writes.
- Web/Desktop behavior.

这些仍属于独立 backlog 项和后续 gate。
