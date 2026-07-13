# Multi-project Isolation Evidence

## Purpose

本文记录 AreaFlow `project_key` 多项目隔离的 PostgreSQL fixture 和 API route scope 证据。

PostgreSQL fixture 覆盖：

- 两个 project 使用相同 workflow version label 时，`GetWorkflowVersion` / `ListWorkflowVersions`
  只解析当前 project。
- `ListWorkflowVersionRuns` 只返回当前 project + workflow version 的 run。
- `ListProjectArtifacts` / `ListWorkflowVersionArtifacts` 只返回当前 project 的 artifact metadata。
- `ListEvents` / `ListAuditEvents` 只返回当前 project 的 event / audit event。
- `ListWorkers` 只返回当前 project 的 worker。
- `RecoverExpiredLeases` 只恢复当前 project 的 expired lease，不修改另一个 project 的 active lease。

API route scope 测试覆盖：

- `/api/v1/projects/{project_key}/summary` 使用 route project。
- `/api/v1/projects/{project_key}/events` 使用 route project。
- `/api/v1/audit-events?project_key=<key>` 使用 query project scope。
- `/api/v1/projects/{project_key}/workers` 使用 route project。
- `/api/v1/projects/{project_key}/workflow-versions` 在两个 project 共享 `shared-v1` label 时分别解析。
- `/api/v1/projects/{project_key}/workflow-versions/shared-v1/runs` 返回当前 project 的 version/run。
- `/api/v1/projects/{project_key}/artifacts` 和
  `/api/v1/projects/{project_key}/workflow-versions/shared-v1/artifacts` 返回当前 project 的 artifact metadata。
- 全局 run/artifact ID route 在提供 `project_key` 时执行 visibility guard：
  `/api/v1/runs/{id}?project_key=<key>`、`/api/v1/runs/{id}/events?project_key=<key>`、
  `/api/v1/runs/{id}/events/stream?project_key=<key>`、
  `/api/v1/runs/{id}/execution-approval-gate?project_key=<key>`、
  `/api/v1/runs/{id}/start?project_key=<key>`、`/api/v1/artifacts/{id}?project_key=<key>` 和
  `/api/v1/artifacts/{id}/content?project_key=<key>`。匹配时返回正常响应，不匹配时返回 `404`，避免泄露
  其他 project 的 ID 是否存在。未提供 `project_key` 时保留本机 single-user 兼容行为。
- Web dashboard 的 run detail 调用使用 `GET /api/v1/runs/{id}?project_key=<selected_project>`，
  `scripts/smoke-web-check.mjs` 会点击 run timeline 并验证该 query scope。

## Smoke Entry

```bash
bash scripts/smoke-project-isolation.sh
make smoke-project-isolation
make smoke-docker-project-isolation
```

The script requires `AREAFLOW_DATABASE_URL`. Without it, the smoke exits with a clear skip message.
With PostgreSQL available, it runs:

```bash
go test ./internal/project -run TestStoreProjectKeyIsolationWithPostgres -count=1 -v
```

## Safety Facts

The test creates uniquely named temporary projects:

```text
project-isolation-<timestamp>-a
project-isolation-<timestamp>-b
```

It writes only AreaFlow PostgreSQL metadata rows and removes its own fixture rows during cleanup. It does not:

- read or write AreaMatrix files;
- write managed project files;
- write artifact contents;
- run worker execution;
- call an engine;
- resolve secrets;
- run shell commands in a managed project;
- touch `.areaflow/status.json` or `workflow/README.md`.

## Latest Result

Current local API validation on 2026-07-02:

```bash
go test ./internal/api -run 'TestProjectScopedAPIsUseRouteProjectKey|TestGlobalRunEndpointsHonorProjectKeyVisibility|TestGlobalArtifactEndpointsHonorProjectKeyVisibility|TestProjectWorkerEndpoints|TestWorkerPool' -count=1
```

Result:

```text
ok github.com/areasong/areaflow/internal/api
```

This proves versioned API routes preserve project scope across summary, events, audit query scope, workers,
workflow versions, runs and artifacts, and proves global run/artifact ID routes honor an optional `project_key`
visibility guard, including run SSE stream preflight. It uses an in-memory fake store, so it complements but does not
replace the PostgreSQL fixture below.

Current local Web browser smoke validation on 2026-07-02 23:06 CST:

```bash
PGPASSWORD=areaflow AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_web_project_guard_<timestamp>?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
temporary database cleanup residual=0
```

This proves the Web dashboard clicks a run timeline item and requests
`GET /api/v1/runs/{run_id}?project_key={fixture_project}` rather than a bare global run detail route.

Latest PostgreSQL fixture validation on 2026-07-03 04:26 CST:

```bash
make smoke-docker-project-isolation
```

Result:

```text
smoke-docker: creating isolated PostgreSQL database areaflow_smoke_20260703042608_368
smoke-project-isolation: running PostgreSQL project_key isolation fixture
=== RUN   TestStoreProjectKeyIsolationWithPostgres
    store_isolation_integration_test.go:44: cleanup projects: closed pool
--- PASS: TestStoreProjectKeyIsolationWithPostgres (0.63s)
PASS
ok  	github.com/areasong/areaflow/internal/project	0.638s
smoke-project-isolation: pass
smoke-docker: ok
smoke-docker: dropping isolated PostgreSQL database areaflow_smoke_20260703042608_368
```

This proves the PostgreSQL fixture passes on this machine against an isolated database and cleans up its temporary
project rows. The no-database skip path remains available when `AREAFLOW_DATABASE_URL` is not set.

## PG Evidence Contract

For future regression checks, run one of:

```bash
AREAFLOW_DATABASE_URL=postgres://... make smoke-project-isolation
make smoke-docker-project-isolation
```

Expected pass markers:

```text
smoke-project-isolation: running PostgreSQL project_key isolation fixture
--- PASS: TestStoreProjectKeyIsolationWithPostgres
smoke-project-isolation: pass
```

## Remaining Boundaries

The combined API + PostgreSQL fixtures cover route project scope and store project scope for workflow, run,
artifact, event, audit, worker and lease recovery metadata.

They do not yet cover:

- real secret resolution scope;
- real API token enforcement scope;
- real team / user permission enforcement scope;
- real remote worker credential enforcement scope;
- real multi-project scheduling execution.

Auth、team、API token、secret resolve 和 remote worker credential 的 R4 opening ladder 已在
[`../architecture/auth-team-secret-boundary.md`](../architecture/auth-team-secret-boundary.md) 定义，但真实
enforcement 仍未打开。
