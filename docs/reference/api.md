# AreaFlow API

AreaFlow API 使用 JSON 和 `/api/v1` 路径。`/api` 是当前兼容入口，新客户端应使用 `/api/v1`。

默认地址：`http://127.0.0.1:3847/api/v1`。

## 健康与服务

```text
GET /health
GET /service/status
```

## 全局资源集合

```text
GET /projects
GET /workflows
GET /runs
GET /workers
GET /artifacts
```

`workflows`、`runs`、`workers` 和 `artifacts` 支持：

- `project_key`：限定项目。
- `limit`：正整数，最大 200。

集合条目显式包含 `project`，run 集合还包含 `workflow_version`。

## Project

```text
GET  /projects/{project_key}
GET  /projects/{project_key}/summary
GET  /projects/{project_key}/readiness
GET  /projects/{project_key}/events
GET  /projects/{project_key}/events/stream
GET  /projects/{project_key}/artifacts
GET  /projects/{project_key}/residuals
POST /projects/{project_key}/import
POST /projects/{project_key}/doctor
```

## Workflow

```text
GET  /projects/{project_key}/workflow-versions
POST /projects/{project_key}/workflow-versions
GET  /projects/{project_key}/workflow-versions/{version}
GET  /projects/{project_key}/workflow-versions/{version}/stages
GET  /projects/{project_key}/workflow-versions/{version}/gates
POST /projects/{project_key}/workflow-versions/{version}/gates
GET  /projects/{project_key}/workflow-versions/{version}/transition-previews
POST /projects/{project_key}/workflow-versions/{version}/transition-previews
GET  /projects/{project_key}/workflow-versions/{version}/approvals
POST /projects/{project_key}/workflow-versions/{version}/approvals
GET  /projects/{project_key}/workflow-versions/{version}/runs
GET  /projects/{project_key}/workflow-versions/{version}/artifacts
GET  /projects/{project_key}/workflow-versions/{version}/residuals
```

## Run

```text
GET  /runs/{run_id}
GET  /runs/{run_id}/events
GET  /runs/{run_id}/events/stream
GET  /runs/{run_id}/execution-approval-gate
GET  /runs/{run_id}/execution-plan
POST /runs/{run_id}/start
POST /runs/{run_id}/drain
POST /runs/{run_id}/cancel
```

run control 更新 AreaFlow 状态并写入 event/audit。它不隐式执行任意项目命令或 AI engine。

## Worker

```text
GET  /projects/{project_key}/workers
POST /projects/{project_key}/workers
POST /projects/{project_key}/workers/{worker_key}/heartbeat
POST /projects/{project_key}/workers/{worker_key}/lease-acquire
POST /projects/{project_key}/workers/{worker_key}/lease-release
POST /projects/{project_key}/workers/lease-recover
POST /projects/{project_key}/workers/{worker_key}/run-once
GET  /worker-pool/summary
GET  /worker-pool/schedule-preview
```

## Artifact

```text
GET  /artifacts/{artifact_id}
GET  /artifacts/{artifact_id}/content
GET  /artifacts/integrity?project_key={project_key}
POST /projects/{project_key}/artifacts/archive-preview
```

Archive preview 不执行 artifact copy、delete、upload 或 GC。

## Audit 与 Operations

```text
GET /audit-events
GET /audit/coverage
GET /permissions/doctor
GET /conformance
GET /ops/readiness
GET /ops/support-bundle-preview
GET /ops/migration-ledger-readiness
GET /backup/manifest
GET /backup/restore-plan
GET /release/readiness
GET /release/final-gate
GET /completion-audit
```

Operations 中的 manifest、plan、preview、readiness 和 gate 不代表外部操作已执行。

## 错误

API 使用 HTTP 状态表达请求结果：

- `400`：参数或请求体无效。
- `404`：资源不存在或不在 project visibility scope。
- `405`：方法不允许。
- `409`：idempotency、状态或授权冲突。
- `500`：平台内部错误。

写操作必须使用 endpoint 定义的 actor、reason、idempotency 和 approval 字段；客户端不得通过重复请求绕过冲突检查。
