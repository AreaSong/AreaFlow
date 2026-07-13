# Workflow Version Authoring Evidence

## Purpose

本文记录 backlog 任务
[`AF-V03-001 Workflow Version Authoring Model`](../../tasks/backlog/0-100-platform-backlog.md#af-v03-001-workflow-version-authoring-model)
的最近一次本机验证证据。

该证据覆盖 AreaFlow 在 PostgreSQL 中创建 authored workflow version、冻结 profile binding、创建 authoring
stage skeleton、workflow item links 和 AreaFlow-owned artifact placeholders。它不写被管理项目
workflow 目录。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/workflow ./internal/project
go build ./cmd/areaflow
```

Result: pass

## Fixture Authoring Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v03_1782890188_96844
Fixture project key: areamatrix-fixture
Fixture root: temporary /private/var/folders/.../areaflow-fixture.*
Artifact store: temporary /private/var/folders/.../areaflow-fixture.*/artifact-store
```

Commands:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_v03_1782890188_96844?sslmode=disable \
  AREAFLOW_KEEP_FIXTURE=1 ./scripts/smoke-fixture.sh
go run ./cmd/areaflow workflow version create areamatrix-fixture v2 --json \
  --actor local-user --reason "AF-V03-001 evidence"
go run ./cmd/areaflow workflow version stages areamatrix-fixture v2 --json
go run ./cmd/areaflow workflow version ensure-skeleton areamatrix-fixture v2 --json \
  --actor local-user --reason "AF-V03-001 idempotency"
go run ./cmd/areaflow workflow gate run areamatrix-fixture v2 profile_binding_drift --json
dropdb -h localhost -p 54329 -U areaflow --if-exists af_v03_1782890188_96844
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname LIKE 'af_v03_%';"
```

Observed fixture setup:

```text
smoke-fixture: project import areamatrix-fixture #1
smoke-fixture: project import areamatrix-fixture #2
smoke-fixture: project status-projection-apply areamatrix-fixture
smoke-fixture: pass areamatrix-fixture fixture=/private/var/folders/.../areaflow-fixture.*
```

Workflow version create probe:

```text
v2|authored|draft|areamatrix|0|true|9|true
```

字段含义：

- display label: `v2`
- import mode: `authored`
- lifecycle status: `draft`
- frozen profile id: `areamatrix`
- frozen profile version: `0`
- frozen profile hash exists: `true`
- created stage skeleton items returned by create command: `9`
- command created a new version: `true`

Workflow stages probe:

```text
10|9|false|false|true
```

字段含义：

- total workflow items for `v2`: `10`
- total item links for `v2`: `9`
- `intake` item exists in current authoring skeleton: `false`
- `closeout` item exists in current authoring skeleton: `false`
- all returned items are marked `owned_by = areaflow`: `true`

`10` includes the initial `version_init` item plus 9 authoring skeleton items. The current v0.3 skeleton is an
authoring trace from `discussion` through `promotion_preview`; it does not open execution, run, projection or
closeout skeletons.

Idempotent ensure-skeleton probe:

```text
0|0|9
```

字段含义：

- newly created items on second ensure: `0`
- newly returned item artifacts on second ensure: `0`
- item links still present: `9`

Profile binding drift gate probe:

```text
profile_binding_drift|pass|false|true|true
```

字段含义：

- gate name: `profile_binding_drift`
- gate status: `pass`
- profile migration attempted: `false`
- frozen profile hash exists: `true`
- current profile hash exists: `true`

Database proof:

```text
1|10|9|9|1|2
```

字段含义：

- 1 authored `v2` workflow version.
- 10 workflow items.
- 9 workflow item links.
- 9 local AreaFlow-owned artifact records.
- 1 completed `workflow.version.create` command request.
- 2 audit events for `workflow.version.create` and `workflow.stage_skeleton.create`.

Artifact store proof:

```text
artifact_files=9
```

The 9 artifact files are written under the temporary AreaFlow artifact store, not the fixture project root.

Project file boundary:

```text
project file list unchanged before and after authoring smoke
```

The smoke compared the fixture project root file list before and after `workflow version create`,
`workflow version stages`, `workflow version ensure-skeleton` and `workflow gate run`; no project files changed.

Cleanup query:

```text
0
```

`af_v03_%` 临时数据库已清理，无残留。

## Evidence

- `go test ./internal/workflow ./internal/project` passed.
- `go build ./cmd/areaflow` passed.
- `workflow version create` creates an AreaFlow-authored workflow version in PostgreSQL.
- The workflow version stores `profile_id`、`profile_version`、`profile_hash` and `profile_path` in status summary.
- The command request is completed and audit events are written.
- Stage skeleton placeholders are AreaFlow-owned records and AreaFlow artifact-store files.
- Stage skeleton links connect the authoring trace.
- Re-running `ensure-skeleton` is idempotent for existing items.
- `profile_binding_drift` passes without attempting profile migration.
- The fixture project root file list is unchanged.
- The temporary PostgreSQL database and fixture directory were cleaned up after the smoke.

## Boundary

这份证据不证明：

- Full 16-stage lifecycle skeleton creation.
- Real AreaMatrix workflow directory writes.
- `workflow/README.md` controlled block write.
- Gate transition approval chain.
- Promotion apply.
- Authoring cutover.
- Task-loop or execution replacement.
- Runner / worker execution.
- Web/Desktop behavior.

这些仍属于独立 backlog 项和后续 gate。
