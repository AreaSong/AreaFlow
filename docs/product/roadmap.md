# AreaFlow Roadmap

## 总原则

AreaFlow 从 0 到 100% 的路线按平台成熟度推进：先建立设计和状态源事实，再导入和镜像真实项目，随后进入 shadow 校验、cutover、执行、Web、多项目、桌面和 1.0 硬化。

完整平台蓝图、目录边界、workflow 状态机、worker、权限审计、artifact 和多端关系见
[`platform-blueprint.md`](platform-blueprint.md)。0-100% 总控计划见
[`master-plan.md`](master-plan.md)。Phase 0 的 0-100% 地基决策见
[`../adr/0005-phase-0-foundation-baseline.md`](../adr/0005-phase-0-foundation-baseline.md)。
运维、部署、可观测性和 support bundle 边界见
[`../architecture/operations-deployment-observability-boundary.md`](../architecture/operations-deployment-observability-boundary.md)。
最终完成审计边界见
[`../architecture/completion-audit-contract.md`](../architecture/completion-audit-contract.md)。
逐阶段 backlog 见 [`phase-backlog.md`](phase-backlog.md)。本文只保留版本路线和阶段门禁摘要。
v0.1 最小闭环和只读 import / mirror 边界见
[`../architecture/v0.1-import-mirror-contract.md`](../architecture/v0.1-import-mirror-contract.md)。
v0.2 Shadow Doctor + Drift Check 的只读验收、import diff、verification bundle 和 native doctor 授权边界见
[`../architecture/v0.2-shadow-doctor-contract.md`](../architecture/v0.2-shadow-doctor-contract.md)。
v0.3 New Version Authoring 的 version create、stage skeleton、gate、transition preview 和 approval record
边界见 [`../architecture/v0.3-version-authoring-contract.md`](../architecture/v0.3-version-authoring-contract.md)。
v0.4 Workflow Ownership Cutover 的 compatibility、shim readiness、cutover readiness、DB-only authoring cutover
apply 和 rollback 边界见
[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../architecture/v0.4-workflow-ownership-cutover-contract.md)。
v0.5 Runner Preview 的 dry-run execution model、runner preview report、run control 和 no-execution
边界见 [`../architecture/v0.5-runner-preview-contract.md`](../architecture/v0.5-runner-preview-contract.md)。
v0.6 Worker Beta 的 worker lifecycle、lease、dry-run run-once、scoped execution evidence 和 no-cutover
边界见 [`../architecture/v0.6-worker-beta-contract.md`](../architecture/v0.6-worker-beta-contract.md)。
v0.7 Web Dashboard 的 `/api/v1`、SSE、read-only panels、write action gate 和 no-second-state
边界见 [`../architecture/v0.7-web-dashboard-contract.md`](../architecture/v0.7-web-dashboard-contract.md)。
v0.8 Multi-project Worker Pool 的 summary、schedule preview、project isolation 和 no-scheduler
边界见
[`../architecture/v0.8-multi-project-worker-pool-contract.md`](../architecture/v0.8-multi-project-worker-pool-contract.md)。
v0.9 Desktop Shell 的 local service status、dashboard launcher、desktop gates 和 no-second-state
边界见 [`../architecture/v0.9-desktop-shell-contract.md`](../architecture/v0.9-desktop-shell-contract.md)。
v1.0 Stable Platform 的 completion audit、release final gate、AreaMatrix dogfood completion 和 high-risk
apply closed 边界见
[`../architecture/v1.0-stable-platform-contract.md`](../architecture/v1.0-stable-platform-contract.md)。

## 版本路线

| 版本 | 阶段 | 核心目标 |
|---|---|---|
| Phase 0 | Foundation | 设计基线、技术决策、迁移协议、v0.1 范围 |
| v0.1 | Import + Status Mirror | 导入 AreaMatrix workflow metadata，生成粗略状态 |
| v0.2 | Shadow Doctor + Drift Check | 等价校验、漂移检测、stage coverage |
| v0.3 | New Version Authoring | 在 AreaFlow 中创建新 workflow version 候选 |
| v0.4 | Workflow Ownership Cutover | 新版本 workflow 源事实切到 AreaFlow |
| v0.5 | Runner Preview | 建 execution/run 模型，但只 dry-run |
| v0.6 | Worker Beta | 执行已批准任务，接入 Codex CLI adapter |
| v0.7 | Web Dashboard | 多项目状态、timeline、artifact、blocker 可视化 |
| v0.8 | Multi-project Worker Pool | 多项目 worker pool summary、schedule preview、agent role 和 resource readiness |
| v0.9 | Desktop Shell | Tauri shell、local service status、dashboard launcher、desktop gates |
| v1.0 | Stable Platform | completion audit、release final gate、backup/restore dry-run、可诊断运维、AreaMatrix dogfood 闭环 |

## 迁移节奏

AreaMatrix workflow 迁移采用完整阶段：

```text
Import
-> Mirror
-> Shadow
-> Authoring Cutover
-> Execution Beta
-> Execution Cutover
-> Archive
-> Shim Retirement
```

旧版本先索引，后归档；新版本先由 AreaFlow 创建和管理。`Authoring Cutover` 只切新 workflow
version 的 authoring 源事实，不代表 `Execution Cutover`，也不替代 `./task-loop run`。

## 阶段门禁

每个阶段必须先满足自己的验收标准，不能偷跑后续能力：

```text
v0.1-v0.2: 只读理解项目
v0.3-v0.4: 接管 workflow authoring
v0.5-v0.6: 接管 execution model 和 worker
v0.7-v0.9: 多端和多项目平台化
v1.0: 安全、审计、发布、兼容稳定
```

Go / No-Go 摘要：

| 阶段 | 放行证据 |
|---|---|
| v0.1 | 最小闭环符合 v0.1 import / mirror 合同；project add/import/status-projection-apply 可重复；`export-status` 作为兼容入口；`.areaflow/status.json` 可生成；无 workflow/execution 写入。 |
| v0.2 | doctor、summary、readiness、import-diff、verify-bundle 稳定；native doctor 未授权时只 warn/skipped；phase gate 的 warn/blocked 不被压成 pass。 |
| v0.3 | workflow version、stage skeleton、gate、transition preview、approval record 可审计；approval 不等于 execution；不 apply promotion。 |
| v0.4 | compatibility、approval gate、live mapping gate、cutover readiness gate 和 DB-only cutover apply 可证明；只切 authoring，不切 execution。 |
| v0.5 | runner preview 产生 run/task/attempt/artifact/event/audit dry-run 证据；run control 只改 dry-run DB 状态；不真实执行。 |
| v0.6 | worker register/heartbeat/lease/run-once/evidence/capability preflight 可审计；scoped execution 只限 approved task；不切 AreaMatrix execution。 |
| v0.7 | Web 只通过 `/api/v1` GET 和 SSE 观察；write action gate 只展示 disabled/read-only 动作，不维护第二套状态。 |
| v0.8 | worker pool summary/schedule-preview 证明多项目状态可观察、可解释、可隔离；真实 scheduler、remote worker、secret resolve、team/auth enforcement 和多项目 execution apply 仍关闭。 |
| v0.9 | Desktop 只观察 local service、健康、worker pool 和 desktop gates；真实 process control、OS notification、native tray/menu、secret resolve 和远程 Team Console 仍关闭。 |
| v1.0 | 符合 v1.0 stable platform 合同；completion audit 证明 release、ops、backup/restore dry-run、isolation、protected path 和 AreaMatrix dogfood 完整闭环。 |

明确禁区：

- v0.1 不执行任务、不写 `workflow/versions/**`、不调用 AI engine。
- v0.2 不创建新 version、不 cutover。
- v0.3 promotion preview 不 apply。
- v0.4 不替代 task-loop、不自动改代码。
- v0.5 不真实执行、不启动 worker、不写项目、不转发 `./task-loop run`。
- v0.6 只执行已批准任务，不重写 AreaMatrix v1 历史。
- v0.8 只做 summary / schedule preview / readiness；`recommended`、`available_slots` 和 `next_action`
  不能解释为 scheduler apply、lease claim、worker dispatch 或 AreaMatrix execution cutover。
- v1.0 不以“测试通过”替代 release final gate；restore apply、secret resolve、publish apply 仍必须走单独 R4 approval。
- v1.0 不以 release final gate 单独替代 completion audit；100% 必须按 completion audit 逐项证明。
- v1.0 不自动升级、不远程运维控制、不导出完整 support bundle、不默认远程 telemetry。
- v1.0 plugin marketplace 只做 seed / manifest draft / conformance；未知第三方 plugin execution 走 v1.x R4。
- v1.0 object artifact backend 只能是 metadata / skipped verifier；archive copy/upload、GC/delete 和 orphan
  cleanup 走 v1.x rung 16，不把 object metadata 当作完整备份。

v1.0 的 100% 是稳定平台边界，不是所有高风险自动化全开。真实项目写入和 engine 打开顺序必须继续保持：

```text
generated-only rollback beta
-> generated-only retained apply
-> manual patch artifact
-> human-applied source evidence
-> source write beta
-> checkpoint apply
-> repair apply
-> no-secret engine execution
-> secret resolve
-> remote worker
-> restore apply
-> release exception real write
-> publish apply
-> third-party plugin execution
-> external integrations / webhooks
-> team console
-> object artifact store
-> budget / quota enforcement
-> managed ops / upgrade / support export
```

这些能力属于 v1.x 逐步开闸范围；每一步都必须重新具备 Command API、capability、allowlist、approval、
rollback / remediation、audit 和 focused smoke 证据。
Team Console 和远程控制台还必须符合
[`../architecture/team-remote-control-boundary.md`](../architecture/team-remote-control-boundary.md)，不能把
角色、页面或按钮当成 project write、secret、publish、restore 或 worker credential 的许可。
Budget / quota enforcement 必须符合
[`../architecture/budget-quota-boundary.md`](../architecture/budget-quota-boundary.md)，不能把 estimate 当成
charge、把 reservation 当成 approval、或用 silent throttling 代替可审计 blocker。
External integrations / webhooks 必须符合
[`../architecture/integration-webhook-boundary.md`](../architecture/integration-webhook-boundary.md)，不能把
callback verified 当成 approval、把 delivery preview 当成 sent，或绕过 Command API 调用外部 API。
Operations / deployment / observability 必须符合
[`../architecture/operations-deployment-observability-boundary.md`](../architecture/operations-deployment-observability-boundary.md)，
不能把 service status 当成 process control、把 support bundle preview 当成 export、把 local diagnostics
当成远程 telemetry opt-in。
统一状态词、apply packet、suspension rule 和 AreaMatrix first policy 见
[`../architecture/high-risk-apply-ladder.md`](../architecture/high-risk-apply-ladder.md)。
