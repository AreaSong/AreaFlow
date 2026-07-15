# AreaFlow API

AreaFlow API 使用 JSON 和 `/api/v1` 路径。`/api` 是当前兼容入口，新客户端应使用 `/api/v1`。兼容入口响应包含 `Deprecation: true`、指向 successor 的 `Link` 和 `Warning: 299`，移除窗口遵循 deprecation policy。

默认地址：`http://127.0.0.1:3847/api/v1`。

## Authentication

```text
GET /auth/status
GET /auth/me
GET /auth/oidc/login
GET /auth/oidc/callback
POST /auth/logout
GET|POST|DELETE /auth/tokens
```

`AREAFLOW_AUTH_MODE=oidc` 使用 server-side cookie session，非 GET/HEAD/OPTIONS 请求还必须携带匹配的 `X-CSRF-Token`。`token` 模式使用 `Authorization: Bearer <token>`。两种模式都绑定 project visibility 和 capability；跨 project 访问返回 `404`。

项目权限入口：

```text
GET|POST|DELETE /projects/{project_key}/role-bindings
```

角色变更要求 `auth.role.manage`、服务端 principal actor、reason 和 audit。Service token 管理要求 `auth.token.manage`；明文 token 只在创建响应中出现一次。

## 健康与服务

```text
GET /api/v1/health
GET /api/v1/ready
GET /api/v1/service/status
```

`/api/v1/health` 是进程存活探针。`/api/v1/ready` 在 2 秒内检查 PostgreSQL、配置的 artifact store，并确认 OIDC provider 初始化状态。响应 `checks` 包含 `database`、`artifact_store` 和 `oidc`；production 关键依赖不可用或超时时返回 `503`。

## 全局资源集合

```text
GET /projects
GET /workflows
GET /runs
GET /workers
GET /artifacts
```

`projects` 当前返回未归档项目列表，不提供 cursor 或写入操作。其余四个集合返回：

```json
{
  "count": 50,
  "next_cursor": "opaque-value",
  "workflows|runs|workers|artifacts": []
}
```

`limit` 默认 50，最大 200。存在下一页时响应包含 `next_cursor`；客户端应把该值原样作为下一次请求的 `cursor`，不得解析、构造或跨集合复用。无效 cursor 返回 `400`。

集合使用稳定的降序键集分页：workflow 按 `updated_at, id`，run 按 `started_at, id`，worker 按 `updated_at, id`，artifact 按 `created_at, id`。

### Workflow 集合

```text
GET /workflows?project_key=&status=&kind=&import_mode=&cursor=&limit=
```

- `status` 匹配 lifecycle status。
- `kind` 匹配 version kind。
- `import_mode` 匹配 imported/authored 等导入模式。

### Run 集合

```text
GET /runs?project_key=&status=&kind=&type=&dry_run=&cursor=&limit=
```

- `kind` 匹配 run kind。
- `type` 匹配 run type。
- `dry_run` 只接受布尔值。

### Worker 集合

```text
GET /workers?project_key=&worker_key=&status=&type=&capability=&cursor=&limit=
```

`worker_key` 使用精确匹配，供稳定详情深链解析；`capability` 要求 worker capabilities 包含指定值。

### Artifact 集合

```text
GET /artifacts?project_key=&type=&storage_backend=&sha256=&run_id=&workflow_version_id=&cursor=&limit=
```

`run_id` 和 `workflow_version_id` 必须是正整数。

所有集合条目显式包含 `project`。Workflow 条目包含 `workflow_version`；Run 条目包含 `run` 和关联的 `workflow_version`；Artifact 通过 `project_id`、`workflow_version_id`、`run_id` 和 `workflow_item_id` 表达关联。

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
GET  /runs/{run_id}/tasks
GET  /runs/{run_id}/tasks/{task_id}
GET  /runs/{run_id}/attempts
GET  /runs/{run_id}/attempts/{attempt_id}
GET  /runs/{run_id}/events
GET  /runs/{run_id}/events/stream
GET  /runs/{run_id}/execution-approval-gate
GET  /runs/{run_id}/execution-plan
POST /runs/{run_id}/start
POST /runs/{run_id}/drain
POST /runs/{run_id}/cancel
```

Run Task 和 Attempt 列表按 Run 隔离，详情路由同时校验 `run_id` 与子资源 ID。响应包含 `project_id`、`workflow_version_id` 和 `run_id`；Attempt 还可通过 `run_task_id` 关联 Task。这些子资源列表当前不支持 cursor pagination。

全局 Run 详情及其子资源可以使用 `project_key` query 作为 visibility guard；项目不匹配时返回 `404`。

run control 更新 AreaFlow 状态并写入 event/audit。它不隐式执行任意项目命令或 AI engine。

## Worker

```text
GET  /workers/{worker_id}?project_key=&limit=
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

Worker 详情返回 `worker`、最近的 `heartbeats` 和最近的 `leases`。`limit` 同时限制两类历史，默认 50、最大 200；两类历史当前没有独立 cursor。可选 `project_key` 用作 visibility guard。

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
GET /audit-events?project_key=&actor_id=&action=&decision=&resource_type=&resource=&from=&to=&cursor=&limit=
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

Audit events 按 `created_at, id` 降序使用 opaque cursor pagination，`limit` 默认 50、最大 200。`actor_id` 必须是正整数；`action`、`decision`、`resource_type` 和 `resource` 使用精确匹配；`from`、`to` 使用 RFC3339，并分别包含边界时间。响应返回 `count` 和可选的 `next_cursor`。

Operations 中的 manifest、plan、preview、readiness 和 gate 不代表外部操作已执行。

## 错误

API 错误使用 RFC 7807 `application/problem+json`，包含 `type`、`title`、`status`、`detail`、兼容 `error` 和可用时的 `request_id`。HTTP 状态包括：

- `400`：参数或请求体无效。
- `404`：资源不存在或不在 project visibility scope。
- `405`：方法不允许。
- `409`：idempotency、状态或授权冲突。
- `500`：平台内部错误。

写操作必须使用 endpoint 定义的 actor、reason、idempotency 和 approval 字段；客户端不得通过重复请求绕过冲突检查。

认证模式下 workflow transition preview 和 approval 的 actor 由服务端 principal 决定，客户端 body 不能冒充 actor。Token approval 必须提供 `Idempotency-Key` header；高风险、critical 或 L4 approval 的 approver 必须不同于 transition requester。

机器可读契约见 [`openapi.yaml`](openapi.yaml)。公开 API 遵循 SemVer；`/api` 兼容 alias 至少保留到 v2，弃用窗口不少于 90 天。
