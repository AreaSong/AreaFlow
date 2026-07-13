# Shadow Doctor Readiness Evidence

## Purpose

本文记录 backlog 任务
[`AF-V02-001 Doctor And Readiness Bundle`](../../tasks/backlog/0-100-platform-backlog.md#af-v02-001-doctor-and-readiness-bundle)
的最近一次本机验证证据。

该证据覆盖 `project doctor`、`project summary`、`project readiness`、`project import-diff` 和
`project verify-bundle`。CLI 命令名是 `verify-bundle`，API 路径名是 `verification-bundle`，两者对应同一
`ProjectVerificationBundle` 能力。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/doctor ./internal/project
go build ./cmd/areaflow
```

Result: pass

## Real AreaMatrix Read-only Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Read-only database: af_v02_1782889705_50502
Project key: areamatrix
Project root: /Users/as/Ai-Project/project/AreaMatrix
Project config: examples/areamatrix/areaflow.yaml
```

Latest checkpoint:

```text
Date: 2026-07-02 16:06 CST
Database: postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable
Command: AREAFLOW_DATABASE_URL=... bash scripts/smoke-areamatrix-readonly.sh
Result: pass
```

Observed latest output:

```text
smoke-areamatrix-readonly: project doctor --json
smoke-areamatrix-readonly: project summary --json
smoke-areamatrix-readonly: project readiness --json
smoke-areamatrix-readonly: project import-diff --json
smoke-areamatrix-readonly: project verify-bundle --json
smoke-areamatrix-readonly: pass areamatrix root=/Users/as/Ai-Project/project/AreaMatrix
```

This checkpoint reused the existing Docker PostgreSQL database and did not run native AreaMatrix commands. It confirms
the v0.2 read-only doctor/readiness/import-diff/verify-bundle chain still passes without modifying real AreaMatrix
projection files.

Commands:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_v02_1782889705_50502?sslmode=disable \
  ./scripts/smoke-areamatrix-readonly.sh
go run ./cmd/areaflow project summary areamatrix --json
go run ./cmd/areaflow project readiness areamatrix --json
go run ./cmd/areaflow project import-diff areamatrix --json
go run ./cmd/areaflow project verify-bundle areamatrix --json
dropdb -h localhost -p 54329 -U areaflow --if-exists af_v02_1782889705_50502
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname LIKE 'af_v02_%';"
```

Observed smoke output:

```text
smoke-areamatrix-readonly: project doctor --json
smoke-areamatrix-readonly: project summary --json
smoke-areamatrix-readonly: project readiness --json
smoke-areamatrix-readonly: project import-diff --json
smoke-areamatrix-readonly: project verify-bundle --json
smoke-areamatrix-readonly: pass areamatrix root=/Users/as/Ai-Project/project/AreaMatrix
```

Summary probe:

```text
warn|pass|pass|pass|warn|true
```

字段含义：

- doctor overall status: `warn`
- hash drift: `pass`
- project config drift: `pass`
- stage coverage: `pass`
- native workflow doctor: `warn`
- import history is ready for diff: `true`

Readiness probe:

```text
warn|pass|pass|warn|warn|pass|pass|pass|warn
```

字段含义：

- readiness overall status: `warn`
- `import_snapshot`: `pass`
- `import_history`: `pass`
- `status_mirror`: `warn`
- `doctor_report`: `warn`
- `drift_check`: `pass`
- `project_config_drift`: `pass`
- `stage_coverage`: `pass`
- `native_workflow_doctor`: `warn`

Import diff probe:

```text
unchanged|true|false|9
```

字段含义：

- import diff status: `unchanged`
- previous import exists: `true`
- source changed: `false`
- compared fields: `9`

Verification bundle probe:

```text
warn|v0.2-shadow-doctor|blocked|true|unchanged|3
```

字段含义：

- bundle status: `warn`
- phase gate name: `v0.2-shadow-doctor`
- phase gate status: `blocked`
- accepted warning includes native doctor permission gate: `true`
- import diff status: `unchanged`
- event count: `3`

The phase gate is expected to stay blocked for the real AreaMatrix read-only smoke because no real
`.areaflow/status.json` mirror export is authorized or written.

Doctor persistence probe:

```text
3|3|3
```

字段含义：

- 3 completed `project.doctor.record` command requests.
- 3 `project.doctor.completed` events.
- 3 doctor events include `overall_status` metadata.

Cleanup query:

```text
0
```

`af_v02_%` 临时数据库已清理，无残留。

## Fixture Smoke With Guarded Projection

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v02fix_1782889742_51978
Fixture project key: areamatrix-fixture
Fixture root: temporary /private/var/folders/.../areaflow-fixture.*
```

Commands:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_v02fix_1782889742_51978?sslmode=disable \
  ./scripts/smoke-fixture.sh
go run ./cmd/areaflow project summary areamatrix-fixture --json
go run ./cmd/areaflow project readiness areamatrix-fixture --json
go run ./cmd/areaflow project import-diff areamatrix-fixture --json
go run ./cmd/areaflow project verify-bundle areamatrix-fixture --json
dropdb -h localhost -p 54329 -U areaflow --if-exists af_v02fix_1782889742_51978
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname LIKE 'af_v02fix_%';"
```

Observed smoke output:

```text
smoke-fixture: project doctor --json
smoke-fixture: project status-projection-apply areamatrix-fixture
smoke-fixture: project status-projections --json
smoke-fixture: project summary --json
smoke-fixture: project readiness --json
smoke-fixture: project import-diff --json
smoke-fixture: project verify-bundle --json
smoke-fixture: pass areamatrix-fixture fixture=/private/var/folders/.../areaflow-fixture.*
```

Summary probe:

```text
warn|pass|pass|pass|warn|1|true
```

字段含义：

- doctor overall status: `warn`
- hash drift: `pass`
- project config drift: `pass`
- stage coverage: `pass`
- native workflow doctor: `warn`
- mirror exports: `1`
- import history is ready for diff: `true`

Readiness probe:

```text
warn|pass|pass|pass|warn|pass|pass|pass|warn
```

字段含义：

- readiness overall status: `warn`
- `import_snapshot`: `pass`
- `import_history`: `pass`
- `status_mirror`: `pass`
- `doctor_report`: `warn`
- `drift_check`: `pass`
- `project_config_drift`: `pass`
- `stage_coverage`: `pass`
- `native_workflow_doctor`: `warn`

Import diff probe:

```text
unchanged|true|false|9
```

Verification bundle probe:

```text
warn|v0.2-shadow-doctor|pass|true|unchanged|4
```

字段含义：

- bundle status: `warn`
- phase gate name: `v0.2-shadow-doctor`
- phase gate status: `pass`
- accepted warning includes native doctor permission gate: `true`
- import diff status: `unchanged`
- event count: `4`

The fixture phase gate passes because `smoke-fixture.sh` writes only the fixture
`.areaflow/status.json` through the guarded status projection command.

Cleanup query:

```text
0
```

`af_v02fix_%` 临时数据库已清理，无残留。

## Evidence

- `go test ./internal/doctor ./internal/project` passed.
- `go build ./cmd/areaflow` passed.
- Real AreaMatrix read-only smoke covered all v0.2 CLI query paths without modifying real AreaMatrix.
- Real AreaMatrix read-only `summary` keeps drift, config drift, stage coverage and native doctor statuses separate.
- Real AreaMatrix read-only `readiness` preserves warn statuses for missing status mirror and skipped native doctor.
- Real AreaMatrix read-only `import-diff` reports unchanged source after repeated imports.
- Real AreaMatrix read-only `verify-bundle` keeps the phase gate blocked while the real mirror export is not authorized.
- Fixture smoke proves the v0.2 phase gate can pass once guarded status projection exists.
- Doctor reports are persisted as command requests and `project.doctor.completed` events with status metadata.
- Native workflow doctor warning is accepted by the v0.2 gate only as an explicit permission-gate warning.
- Temporary PostgreSQL databases were dropped after both smoke runs.

## Boundary

这份证据不证明：

- Real AreaMatrix status mirror write.
- Native AreaMatrix workflow doctor execution.
- `workflow/README.md` controlled block write.
- Authoring cutover.
- Task-loop or execution replacement.
- Runner / worker execution.
- Web/Desktop behavior.
- Secret, restore, publish or plugin capabilities.

这些仍属于独立 backlog 项和后续 gate。
