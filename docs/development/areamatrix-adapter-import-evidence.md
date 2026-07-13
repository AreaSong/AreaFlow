# AreaMatrix Adapter Import Evidence

## Purpose

本文记录 backlog 任务
[`AF-V01-002 AreaMatrix Adapter Metadata Import`](../../tasks/backlog/0-100-platform-backlog.md#af-v01-002-areamatrix-adapter-metadata-import)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库和真实 AreaMatrix project root，只做只读 metadata import。
脚本会比较真实 AreaMatrix 的 `.areaflow/status.json` 与 `workflow/README.md` 前后指纹，确保没有写入
AreaMatrix。

## Run

Date: 2026-07-01

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Read-only database: af_ro_1782889335_28167
Project key: areamatrix
Project root: /Users/as/Ai-Project/project/AreaMatrix
Project config: examples/areamatrix/areaflow.yaml
```

Latest checkpoint:

```text
Date: 2026-07-06 00:01 CST
Database: temporary areaflow_smoke_20260706000100_84386 database on localhost:54329
Command: make smoke-docker-areamatrix-readonly
Result: pass
Dropped isolated database: yes
```

Observed latest output:

```text
smoke-areamatrix-readonly: migrate up
migrations already up to date
smoke-areamatrix-readonly: project add examples/areamatrix/areaflow.yaml
smoke-areamatrix-readonly: project import areamatrix #1
smoke-areamatrix-readonly: project import areamatrix #2
smoke-areamatrix-readonly: project doctor --json
smoke-areamatrix-readonly: project summary --json
smoke-areamatrix-readonly: project readiness --json
smoke-areamatrix-readonly: project import-diff --json
smoke-areamatrix-readonly: project verify-bundle --json
smoke-areamatrix-readonly: project status-projection-authorization --json
smoke-areamatrix-readonly: project status-projection-apply-packet --json
smoke-areamatrix-readonly: project status-projection-apply-gate --json
smoke-areamatrix-readonly: project shim-preview --json
smoke-areamatrix-readonly: project shim-readiness --json
smoke-areamatrix-readonly: project shim-authorization --json
smoke-areamatrix-readonly: project shim-apply-packet --json
smoke-areamatrix-readonly: project shim-apply-gate --json
smoke-areamatrix-readonly: project shim-authorization text
smoke-areamatrix-readonly: pass areamatrix root=/Users/as/Ai-Project/project/AreaMatrix
```

This checkpoint used an isolated Docker PostgreSQL database and dropped it after completion. It refreshed
the read-only proof that real AreaMatrix metadata import is repeatable, that status projection and shim
authorization previews remain read-only, and that the smoke detected no changes to real AreaMatrix
`.areaflow/status.json` or `workflow/README.md`.

AreaMatrix file-status spot check after the smoke:

```bash
git -C /Users/as/Ai-Project/project/AreaMatrix status --short \
  .areaflow/status.json workflow/README.md scripts/areaflow_shim.py \
  scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py
```

Result:

```text
<no output>
```

Commands:

```bash
go test ./internal/adapter/areamatrix ./internal/importer ./internal/project
go build ./cmd/areaflow
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_ro_1782889335_28167?sslmode=disable \
  ./scripts/smoke-areamatrix-readonly.sh
psql -h localhost -p 54329 -U areaflow -d af_ro_1782889335_28167 -Atc \
  "SELECT count(*) AS external_artifacts, count(*) FILTER (WHERE sha256 IS NOT NULL AND source_path IS NOT NULL AND size_bytes IS NOT NULL) AS complete_artifacts, count(DISTINCT run_id) AS artifact_runs, count(DISTINCT workflow_version_id) AS artifact_versions, count(*) FILTER (WHERE storage_backend <> 'external_project') AS non_external_artifacts FROM artifacts;"
dropdb -h localhost -p 54329 -U areaflow --if-exists af_ro_1782889335_28167
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname LIKE 'af_ro_%';"
```

## Result

Status: pass

Observed smoke output:

```text
applied 000001_v0_1_core.sql
applied 000002_v0_3_command_requests.sql
applied 000003_v0_3_gate_results.sql
applied 000004_v0_3_approval_transition.sql
applied 000005_v0_5_runner_preview.sql
applied 000006_v0_6_worker_registry.sql
applied 000007_v0_8_scheduling_policy.sql
applied 000008_v0_3_workflow_item_links.sql
applied 000009_v1_boundary_foundation.sql
applied 000010_v1_status_projections.sql
smoke-areamatrix-readonly: project add examples/areamatrix/areaflow.yaml
smoke-areamatrix-readonly: project import areamatrix #1
smoke-areamatrix-readonly: project import areamatrix #2
smoke-areamatrix-readonly: project doctor --json
smoke-areamatrix-readonly: project summary --json
smoke-areamatrix-readonly: project readiness --json
smoke-areamatrix-readonly: project import-diff --json
smoke-areamatrix-readonly: project verify-bundle --json
smoke-areamatrix-readonly: project shim-preview --json
smoke-areamatrix-readonly: project shim-readiness --json
smoke-areamatrix-readonly: pass areamatrix root=/Users/as/Ai-Project/project/AreaMatrix
```

Artifact metadata query:

```text
6|6|1|2|0
```

字段含义：

- 6 `external_project` artifacts。
- 6 artifacts 同时具备 `sha256`、`source_path` 和 `size_bytes`。
- artifact metadata 关联 1 个 import run。
- artifact metadata 关联 2 个 workflow versions。
- 0 个 artifact 使用非 `external_project` storage backend。

Cleanup query:

```text
0
```

`af_ro_%` 临时数据库已清理，无残留。

## Evidence

- `go test ./internal/adapter/areamatrix ./internal/importer ./internal/project` passed.
- `go build ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- `project add` registered the real AreaMatrix root through `examples/areamatrix/areaflow.yaml`.
- `project import` ran twice and completed repeatable metadata import.
- Imported artifact rows use `storage_backend = 'external_project'`.
- Imported artifact rows include hash, source path, size, type, project, workflow version and run linkage.
- The import does not copy historical artifact original content into AreaFlow storage.
- `project import-diff --json` reported unchanged source after the repeated import.
- `project doctor`, `summary`, `readiness` and `verify-bundle` returned v0.1/v0.2 read-only evidence.
- `shim-preview` and `shim-readiness` stayed in planning / blocked readiness mode.
- The smoke script compared real AreaMatrix `.areaflow/status.json` before and after and did not detect changes.
- The smoke script compared real AreaMatrix `workflow/README.md` before and after and did not detect changes.
- The temporary PostgreSQL database was dropped after the smoke.

## Boundary

这份证据不证明：

- Real AreaMatrix status mirror write.
- Native AreaMatrix workflow doctor execution.
- AreaMatrix `workflow/README.md` controlled block write.
- Authoring cutover.
- Task-loop or execution replacement.
- Runner / worker execution.
- Web/Desktop behavior.
- Secret, restore, publish or plugin capabilities.

这些仍属于独立 backlog 项和后续 gate。
