# AreaFlow 架构总览

AreaFlow 是以 PostgreSQL 为主状态、以 API 为统一控制面的 AI 开发项目管理平台。CLI、Web、Desktop 和 worker 共享同一组领域对象和审计语义。

## 组件

```text
Web / Desktop
  -> REST API / SSE
CLI / local worker
  -> in-process domain Command/Query API
    -> Project service
    -> Workflow service
    -> Run and worker service
    -> Artifact service
    -> Permission and audit service
      -> PostgreSQL
      -> Artifact store
      -> Project adapters
      -> Workflow profiles
```

## 技术基线

| 层 | 当前实现 |
|---|---|
| Backend、CLI、worker | Go |
| 主状态 | PostgreSQL |
| API | REST/JSON，`/api/v1` |
| 实时事件 | SSE |
| Web | React、TypeScript、Vite |
| Desktop | Tauri |
| 项目配置 | YAML |
| Artifact 原文 | local 或 S3-compatible object storage |
| Worker coordination | PostgreSQL lease |
| Authentication | OIDC session 或 scoped service token |
| Observability | Prometheus、OpenTelemetry、JSON log |

## 主状态与历史

- 当前资源状态保存在领域主表。
- domain history 写入 `events`。
- 安全决策写入 `audit_events`。
- 大内容保存在 artifact store，数据库保存 metadata、hash、URI 和关联关系。
- events 和 audit events 由 PostgreSQL trigger 强制 append-only；项目删除只允许外键置空，其他字段不得更新或删除。

## Project boundary

`areaflow.yaml` 声明 project root、adapter、profile、ownership、permissions、status export、scheduling 和 engine references。

AreaFlow 默认只读。读取或写入被管理项目时，必须同时服从 project config、capability、路径规则、gate、approval 和 audit。AreaFlow 不拥有项目源码和产品文档的语义。

CLI 与本地 worker 可以和 server 复用同一 Go domain API，但不得在 `internal/app` 中执行 SQL或直接改 artifact/project 文件。Web/Desktop 只能通过 `/api/v1`。两种 transport 最终都进入相同的 project-scoped Command、idempotency、approval 和 audit 实现；migration、离线 backup/restore drill 是明确的运维例外。

详细配置见 [Project Configuration](../reference/configuration.md)。

## Adapter 与 Profile

Adapter 负责读取和映射项目事实；workflow profile 声明 stage、gate、transition 和 artifact policy。Core engine 不硬编码 AreaMatrix 的流程。

AreaMatrix 是首个内置 adapter/profile 组合，其声明式 profile 位于 `workflow/profiles/areamatrix/profile.yaml`。

## Workflow lifecycle

Workflow Version 冻结 profile version/hash，包含 workflow items、gate results、transition previews 和 approvals。导入并标记 immutable 的历史版本不能被后续 profile 变更静默重写。

## Execution

Run 表示执行会话，Run Task 表示 worker 可领取的单元，Attempt 表示一次尝试，Lease 负责 worker 与 task 的限时绑定。

当前实现同时包含：

- dry-run runner 和 worker `run-once`。
- read-only verification。
- 经过审批的 AreaFlow artifact write。
- fixture project write。
- managed generated write。

这些能力都有不同的权限和副作用边界，不能统一解释为任意 AI engine 或项目命令执行。

## API 与 Web

API 暴露 project 内的 workflow、run、worker、artifact、event 和 audit 资源，以及 operations/readiness endpoints。Web 使用 Overview、Projects、Workflows、Runs、Workers、Artifacts、Audit 和 Operations 八个页面按需读取这些资源。

Web 通过 OIDC cookie session 或显式 token mode 访问 API。项目角色管理和 authored workflow approval/rejection 按 capability 开放；后端写入仍要求 command、confirmation、idempotency、approval 和 audit 契约。

## Operations

Operations 提供 service status、migration ledger、support bundle metadata preview、backup manifest、restore plan、release gate 和 completion audit。

Manifest、plan、preview、readiness 和 gate 是诊断与决策事实，不等于真实 restore、publish、rollout 或 destructive action 已执行。

## 安全不变量

- PostgreSQL 是平台主状态源，不提供 SQLite 主状态 fallback。
- deny 和 forbidden path 优先于 allow。
- secret value 不写入 event、audit、artifact metadata 或 support bundle。
- users、OIDC identity、project role binding、session 和 service token 是当前能力；teams、webhooks、secret resolve 和 remote worker 仍只是边界预留。
- project write、execution write、engine call 和 network access 必须分别授权，不能由单一 gate 隐式打开。

关键决策记录在 [ADR](../adr/)。
