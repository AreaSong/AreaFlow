# AreaFlow 架构总览

AreaFlow 是以 PostgreSQL 为主状态、以 API 为统一控制面的 AI 开发项目管理平台。CLI、Web、Desktop 和 worker 共享同一组领域对象和审计语义。

## 组件

```text
CLI / Web / Desktop
  -> REST API / SSE
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
| Artifact 原文 | local artifact store |
| Worker coordination | PostgreSQL lease |

## 主状态与历史

- 当前资源状态保存在领域主表。
- domain history 写入 `events`。
- 安全决策写入 `audit_events`。
- 大内容保存在 artifact store，数据库保存 metadata、hash、URI 和关联关系。
- events 和 audit events 采用 append-only 思路，历史事实不通过覆盖更新改写。

## Project boundary

`areaflow.yaml` 声明 project root、adapter、profile、ownership、permissions、status export、scheduling 和 engine references。

AreaFlow 默认只读。读取或写入被管理项目时，必须同时服从 project config、capability、路径规则、gate、approval 和 audit。AreaFlow 不拥有项目源码和产品文档的语义。

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

当前 Web 写操作由统一 write action gate 保持关闭。后端存在的写入 endpoint 仍要求 command、confirmation、idempotency、approval 和 audit 契约，不应由前端直接绕过。

## Operations

Operations 提供 service status、migration ledger、support bundle metadata preview、backup manifest、restore plan、release gate 和 completion audit。

Manifest、plan、preview、readiness 和 gate 是诊断与决策事实，不等于真实 restore、publish、rollout 或 destructive action 已执行。

## 安全不变量

- PostgreSQL 是平台主状态源，不提供 SQLite 主状态 fallback。
- deny 和 forbidden path 优先于 allow。
- secret value 不写入 event、audit、artifact metadata 或 support bundle。
- 未启用的 users、teams、tokens、webhooks、secret resolve 和 remote worker 仅为 schema boundary，不是当前产品能力。
- project write、execution write、engine call 和 network access 必须分别授权，不能由单一 gate 隐式打开。

关键决策记录在 [ADR](../adr/)。
