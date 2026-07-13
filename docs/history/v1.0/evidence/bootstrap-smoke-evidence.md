# Bootstrap Smoke Evidence

## Purpose

本文记录 backlog 任务
[`AF-V01-001 PostgreSQL Bootstrap Smoke`](../plans/task-backlog.md#af-v01-001-postgresql-bootstrap-smoke)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库和临时 AreaMatrix-like fixture project，不写真实 AreaMatrix。

## Run

Date: 2026-07-01

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_smoke_1782888949_6064
Fixture project key: areamatrix-fixture
Fixture root: temporary /private/var/folders/.../areaflow-fixture.*
```

Commands:

```bash
go test ./...
go build ./cmd/areaflow
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_smoke_1782888949_6064?sslmode=disable \
  ./scripts/smoke-fixture.sh
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
smoke-fixture: project import areamatrix-fixture #1
smoke-fixture: project import areamatrix-fixture #2
smoke-fixture: project status-projection-apply areamatrix-fixture
smoke-fixture: project verify-bundle --json
smoke-fixture: pass areamatrix-fixture
```

## Evidence

- `go test ./...` passed.
- `go build ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- `project add` registered the fixture project.
- `project import` ran twice and produced completed `project.import` command requests.
- `project doctor` recorded a `project.doctor.record` command request.
- `project status-projection-apply` wrote only the fixture `.areaflow/status.json`.
- `project status-projections --json` returned written projection metadata.
- `project summary`, `readiness`, `import-diff` and `verify-bundle` returned expected v0.1/v0.2 JSON.
- The smoke script compared real AreaMatrix `.areaflow/status.json` before and after and did not detect changes.
- The temporary fixture directory was removed by script cleanup.
- The temporary PostgreSQL database was dropped after the smoke.

## Boundary

这份证据不证明：

- Real AreaMatrix status mirror write.
- Native AreaMatrix doctor execution.
- Workflow authoring cutover.
- Task execution.
- Web/Desktop behavior.
- Secret, restore, publish or plugin capabilities.

这些仍属于独立 backlog 项和后续 gate。
