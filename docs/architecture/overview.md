# Architecture Overview

## 技术基线

| 层 | 决策 |
|---|---|
| Backend / Scheduler / CLI | Go |
| 主状态 | PostgreSQL |
| API | REST/JSON |
| 实时事件 | SSE |
| Web | React + TypeScript, v0.7 后 |
| Desktop | Tauri, v0.9 后 |
| 配置 | YAML |
| Artifact 原文 | local artifact store 起步，后续对象存储 |
| Queue | PostgreSQL row lock + lease 起步，后续可引入 Redis/NATS |

项目接入协议见 [`project-config.md`](project-config.md)。`areaflow.yaml` 声明 project scope、
adapter/profile、ownership、permission、status export、migration、scheduler 和 engine 引用。
v0.1 import / mirror 最小命令集、读写边界和进入 v0.2 的条件见
[`v0.1-import-mirror-contract.md`](v0.1-import-mirror-contract.md)。
AreaMatrix v0.1 adapter 的只读导入深度、minimum import set 和 artifact metadata 策略见
[`areamatrix-import-scope-contract.md`](areamatrix-import-scope-contract.md)。
v0.2 Shadow Doctor、Drift Check、readiness bundle、import diff 和 native doctor 授权边界见
[`v0.2-shadow-doctor-contract.md`](v0.2-shadow-doctor-contract.md)。
v0.3 New Version Authoring 的 version create、stage skeleton、gate、transition preview 和 approval record
边界见 [`v0.3-version-authoring-contract.md`](v0.3-version-authoring-contract.md)。
v0.4 Workflow Ownership Cutover 的 compatibility、shim readiness、cutover readiness、DB-only cutover apply
和 rollback 边界见 [`v0.4-workflow-ownership-cutover-contract.md`](v0.4-workflow-ownership-cutover-contract.md)。
v0.5 Runner Preview 的 dry-run run/task/attempt/artifact、run control 和 no-execution 边界见
[`v0.5-runner-preview-contract.md`](v0.5-runner-preview-contract.md)。
v0.6 Worker Beta 的 worker lifecycle、lease、dry-run run-once、scoped execution evidence 和 no-cutover
边界见 [`v0.6-worker-beta-contract.md`](v0.6-worker-beta-contract.md)。
v0.7 Web Dashboard 的 `/api/v1`、SSE、read-only panels、write action gate 和 no-second-state 边界见
[`v0.7-web-dashboard-contract.md`](v0.7-web-dashboard-contract.md)。
v0.8 Multi-project Worker Pool 的 worker pool summary、schedule preview、project isolation、
engine/resource/role readiness 和 no-scheduler 边界见
[`v0.8-multi-project-worker-pool-contract.md`](v0.8-multi-project-worker-pool-contract.md)。
v0.9 Desktop Shell 的 local service status、dashboard launcher、service-control gate、notification gate、
tray/menu gate 和 no-second-state 边界见
[`v0.9-desktop-shell-contract.md`](v0.9-desktop-shell-contract.md)。
v1.0 Stable Platform 的 release final gate、completion audit、backup/restore preview、operations readiness、
AreaMatrix dogfood completion 和 high-risk apply closed 边界见
[`v1.0-stable-platform-contract.md`](v1.0-stable-platform-contract.md)。
统一写入口、approval scope、command class、permission 顺序、write-set、rollback 和 safety facts 见
[`command-approval-contract.md`](command-approval-contract.md)。
Auth、team、API token、secret resolve 和 remote worker credential 的高风险开闸边界见
[`auth-team-secret-boundary.md`](auth-team-secret-boundary.md)。
Team Console、远程控制台和多用户控制面的边界见
[`team-remote-control-boundary.md`](team-remote-control-boundary.md)。
Release final gate、exception 和 publish preview 的边界见
[`release-final-gate-contract.md`](release-final-gate-contract.md)。
0-100% 最终完成审计、release packaging 预览和 completion evidence 的边界见
[`completion-audit-contract.md`](completion-audit-contract.md)。
v1.x high-risk real apply 的统一开闸阶梯见
[`high-risk-apply-ladder.md`](high-risk-apply-ladder.md)。
Plugin / marketplace seed 与未来 plugin execution 的边界见
[`plugin-marketplace-boundary.md`](plugin-marketplace-boundary.md)。
Object artifact store、archive copy/upload、retention-aware GC 和 delete apply 的边界见
[`object-artifact-retention-contract.md`](object-artifact-retention-contract.md)。
Budget、quota、rate limit 和 usage metering 的边界见
[`budget-quota-boundary.md`](budget-quota-boundary.md)。
External integrations、webhooks、third-party callbacks 和多 API 接入的边界见
[`integration-webhook-boundary.md`](integration-webhook-boundary.md)。
Operations、deployment、observability、support bundle、telemetry、upgrade 和 rollback 的边界见
[`operations-deployment-observability-boundary.md`](operations-deployment-observability-boundary.md)。

## 运行形态

AreaFlow 使用一个 Go binary，多个运行模式：

```text
areaflow migrate
areaflow server
areaflow service status
areaflow worker
areaflow project import
areaflow project status
```

v0.1 中 CLI 默认通过 local service/API 工作；`migrate`、`dev` 等 bootstrap 命令可以直连数据库。

## 组件

```text
CLI / Web / Desktop
  -> AreaFlow API Service
    -> PostgreSQL
    -> Artifact Store
    -> Workflow Stage Engine
    -> Project Adapters
    -> Worker Pool
    -> AI Engine Adapters
```

## 状态原则

- 当前状态在主表。
- 历史过程在 `events`。
- 安全和权限判断在 `audit_events`。
- 大内容在 artifact store，数据库保存 metadata、hash 和 URI。

## API Surface

初始 REST/JSON API 先暴露只读健康检查和项目事件 timeline：

```text
GET /api/v1/health
GET /api/v1/projects/{project_key}/summary
GET /api/v1/projects/{project_key}/events?limit=20
```

CLI、Web 和 Desktop 后续应复用同一事件语义，不各自发明状态历史。
完整 API 分层、CLI/Web/Desktop/Worker 边界见 [`api-surface.md`](api-surface.md)。

## Workflow Engine

AreaFlow 的 workflow 编排层遵循
[`workflow-engine-contract.md`](workflow-engine-contract.md)。Core engine 只理解
stage、gate、artifact、transition、trace、permission 和 event；AreaMatrix 当前
workflow 作为第一个内置 profile，而不是硬编码成唯一流程。

## Data Model

v0.1 轻量 schema 见 [`data-model-v0.1.md`](data-model-v0.1.md)。v1 目标对象模型见
[`data-model-v1.md`](data-model-v1.md)，覆盖 identity、project、workflow、execution、
artifact、integration、幂等和并发边界。

## Execution

AreaFlow 最终以 [`execution-model.md`](execution-model.md) 接管 workflow 执行模型：
`run` 表示一次执行会话，`run_task` 表示 worker 可领取的执行单元，`attempt` 表示一次真实尝试，
worker 只通过指向 `run_task` 的 lease 工作。
v0.5 只按 [`v0.5-runner-preview-contract.md`](v0.5-runner-preview-contract.md) 做 dry-run runner preview
和 DB-only run control。v0.6 按 [`v0.6-worker-beta-contract.md`](v0.6-worker-beta-contract.md) 打开
worker lifecycle、lease、dry-run run-once 和 scoped execution evidence；真实 execution 的开闸顺序、copy /
verify / repair / checkpoint 分离规则和 AreaMatrix first execution policy 见
[`execution-opening-strategy.md`](execution-opening-strategy.md)。
v0.7 Web 只能展示这些状态和 gate，不能把 scoped evidence、readiness 或 SSE event 解释为 execution
cutover 或 Command API apply；详细边界见 [`v0.7-web-dashboard-contract.md`](v0.7-web-dashboard-contract.md)。
v0.8 只能把多项目 worker pool 作为 summary / schedule preview / readiness 展示；`recommended=true`、
`available_slots>0` 或 `next_action=worker_run_once_preview` 都不是 scheduler apply、lease claim、
worker dispatch 或 AreaMatrix execution cutover，详细边界见
[`v0.8-multi-project-worker-pool-contract.md`](v0.8-multi-project-worker-pool-contract.md)。
post-100% 的 secret、remote worker、restore、publish 和 plugin 等 R4 apply 顺序见
[`high-risk-apply-ladder.md`](high-risk-apply-ladder.md)。

## Adapter And Profile

Adapter 和 workflow profile 必须分离，见
[`adapter-profile-boundary.md`](adapter-profile-boundary.md)。Adapter 负责读取/映射项目，
profile 负责声明 stage、gate 和 transition；AreaMatrix 只是第一个内置组合。
v1.0 只稳定 built-in / seed catalog、manifest draft 和 conformance；未知第三方 plugin execution
留到 v1.x，见 [`plugin-marketplace-boundary.md`](plugin-marketplace-boundary.md)。
