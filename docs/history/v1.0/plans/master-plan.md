# AreaFlow 0-100% Master Plan

## 定位

本文是 AreaFlow 从 0% 到 100% 的总控计划。它收束当前已经讨论并锁定的地基决策，作为后续
实现、迁移、验收和 dogfooding 的入口源事实。

本文不替代更细文档：

- [`platform-blueprint.md`](./platform-blueprint.md)：平台蓝图、长期目录边界和核心模型。
- [`phase-backlog.md`](./phase-backlog.md)：逐阶段可执行 backlog。
- [`roadmap.md`](../../../roadmap.md)：当前路线图入口。
- [`../adr/0005-phase-0-foundation-baseline.md`](../../../adr/0005-phase-0-foundation-baseline.md)：Phase 0 地基 ADR。
- [`../adr/0006-platform-operating-boundary.md`](../../../adr/0006-platform-operating-boundary.md)：平台操作边界 ADR。
- [`../architecture/workflow-engine-contract.md`](../contracts/workflow-engine-contract.md)：workflow engine 契约。
- [`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)：v0.1
  Import + Status Mirror 最小 CLI、API、读写边界和进入 v0.2 的条件。
- [`../architecture/areamatrix-import-scope-contract.md`](../contracts/areamatrix-import-scope-contract.md)：
  AreaMatrix v0.1 只读导入深度、minimum import set 和 artifact metadata 策略。
- [`../architecture/v0.2-shadow-doctor-contract.md`](../contracts/v0.2-shadow-doctor-contract.md)：v0.2
  Shadow Doctor、Drift Check、readiness、import-diff、verify-bundle 和 native doctor 授权边界。
- [`../architecture/v0.3-version-authoring-contract.md`](../contracts/v0.3-version-authoring-contract.md)：v0.3
  workflow version create、stage skeleton、gate、transition preview、approval record 和 no-apply 边界。
- [`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../contracts/v0.4-workflow-ownership-cutover-contract.md)：
  v0.4 compatibility、shim readiness、cutover readiness、DB-only authoring cutover apply 和 rollback 边界。
- [`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md)：v0.5
  runner preview、dry-run run control、runner preview report 和 no-execution 边界。
- [`../architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md)：v0.6
  worker lifecycle、lease、dry-run run-once、scoped execution evidence 和 no-cutover 边界。
- [`../architecture/v0.7-web-dashboard-contract.md`](../contracts/v0.7-web-dashboard-contract.md)：v0.7
  Web Dashboard 的 `/api/v1`、SSE、read-only panels、write action gate 和 no-second-state 边界。
- [`../architecture/v0.8-multi-project-worker-pool-contract.md`](../contracts/v0.8-multi-project-worker-pool-contract.md)：v0.8
  worker pool summary、schedule preview、project isolation、readiness 和 no-scheduler 边界。
- [`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)：v0.9
  local service status、dashboard launcher、desktop gates 和 no-second-state 边界。
- [`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)：v1.0
  completion audit、release final gate、AreaMatrix dogfood completion、protected path proof 和 high-risk apply
  closed 边界。
- [`../architecture/data-model-v1.md`](../contracts/data-model-v1.md)：v1 数据模型。
- [`../architecture/command-approval-contract.md`](../contracts/command-approval-contract.md)：Command API、
  approval、permission、risk、audit 和 apply 的统一写入口合同。
- [`../architecture/high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md)：v1.x
  high-risk real apply 从 closed / preview 到 scoped production 的统一开闸阶梯。
- [`../architecture/artifact-backup-restore-contract.md`](../contracts/artifact-backup-restore-contract.md)：
  artifact store、integrity、backup manifest、restore dry-run、archive/retention 和 restore apply 边界。
- [`../architecture/object-artifact-retention-contract.md`](../contracts/object-artifact-retention-contract.md)：
  object artifact store、archive copy/upload、retention-aware GC 和 delete apply 的 v1.x 开闸边界。
- [`../architecture/integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)：
  external integrations、webhooks、third-party callbacks 和多 API 接入边界。
- [`../architecture/budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md)：budget、quota、
  rate limit 和 usage metering 边界。
- [`../architecture/release-final-gate-contract.md`](../contracts/release-final-gate-contract.md)：v1.0
  release readiness、exception、final gate、evidence、publish preview 和 v1.x apply 边界。
- [`../architecture/completion-audit-contract.md`](../contracts/completion-audit-contract.md)：0-100%
  完成审计、release packaging preview、dogfood cutover 和 evidence 聚合边界。
- [`../architecture/operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md)：
  install、migration、service lifecycle、diagnostics、support bundle、telemetry、upgrade 和 rollback 边界。
- [`../architecture/execution-opening-strategy.md`](../../../../proposals/execution-opening.md)：execution
  能力开闸顺序、copy / verify / repair / checkpoint 分离和 AreaMatrix first execution policy。
- [`../architecture/worker-scheduling-contract.md`](../contracts/worker-scheduling-contract.md)：多项目 worker
  调度、lease、并发、recovery 和 preview / scheduler 边界。
- [`../architecture/auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)：auth、team、
  API token、secret resolve 和 remote worker credential 的 R4 开闸边界。
- [`../architecture/plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)：v1.0
  plugin / marketplace seed、manifest、conformance 和 v1.x plugin execution 边界。
- [`../migration/areamatrix-workflow-migration.md`](../migrations/areamatrix-workflow-migration.md)：AreaMatrix 迁移路线。
- [`../migration/areamatrix-execution-cutover-boundary.md`](../migrations/areamatrix-execution-cutover-boundary.md)：
  AreaMatrix `./task-loop` / `./dev workflow` 执行入口切换边界、命令映射和回滚规则。

## 总目标

AreaFlow 最终成为独立的 AI 开发项目管理平台，负责 workflow lifecycle、版本规划、任务编排、
执行记录、worker 调度、artifact 索引、多项目状态、审计和发布门禁。

AreaMatrix 是第一个 dogfooding project，不是 AreaFlow 的子模块。AreaFlow 从 AreaMatrix 的
workflow/task-loop 经验中抽象平台能力，但 core 不硬编码 AreaMatrix 的目录结构。

达到 100% 时必须能证明：

- AreaFlow 可从空环境安装、迁移、启动并注册项目。
- AreaMatrix dogfood 完成 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta
  -> Execution Cutover -> Archive -> Shim Retirement 的可验证闭环。
- 新 workflow version 源事实由 AreaFlow 管理。
- Approved task 可由 worker 执行，并留下 copy、verify、repair、checkpoint、artifact、event 和 audit 证据。
- CLI、Web、Desktop、Worker 都复用同一 API 和 service layer。
- 本机 install、migrate、start、doctor、service status 和 support bundle preview 可复验，且不触碰
  AreaMatrix protected paths。
- PostgreSQL metadata、AreaFlow-owned artifact metadata 和 local artifact store 可以生成 backup
  manifest、artifact integrity 和 restore dry-run plan。
- `project_reference` 历史 artifact 的恢复限制被清楚标注。
- Adapter、profile、plugin 边界稳定。
- Release final gate 不依赖口头判断；所有 blocked / exception 都有 owner、evidence、rollback 和审计路径。
- Completion audit 逐项证明 v0-v1.0 task、AreaMatrix dogfood、release packaging preview、operations readiness、
  project isolation 和 protected path proof 均满足 v1.0 范围。

## 讨论关闭账本

当前 0-100% 地基讨论已经关闭以下决策：

| 议题 | 决策 |
|---|---|
| 仓库边界 | AreaFlow 是独立平台仓库；AreaMatrix 是第一个 dogfooding project，不作为子模块。 |
| 技术栈 | Go + PostgreSQL + REST/JSON + SSE；Web 用 React + TypeScript；Desktop 用 Tauri。 |
| 状态源事实 | PostgreSQL 是主状态；文件只用于配置、artifact 原文、projection 和审计导出。 |
| workflow 模型 | Core 使用 stage engine + profile，不硬编码 AreaMatrix 目录。 |
| workflow 根对象 | `workflow_version` 是版本生命周期根对象；stage 内交付物用 `workflow_item` 表达，文件、prompt、日志、报告和 diff 都是 artifact。 |
| workflow 状态分层 | `lifecycle_status`、stage/item status、gate result、approval state、runtime state、lease state 和 projection state 必须分表/分层表达。 |
| lifecycle 粗状态 | `workflow_versions.lifecycle_status` 只表达版本所处粗阶段；AreaMatrix 的 16 个 stage、gate、approval、run、lease 和 projection 不压进同一个字段。 |
| queue 与 execution | `queue_candidate` / `workflow_item` 表示“要做什么”；一个 item 可以产生多个 `run`，一个 `run` 可拆多个 `run_task`，重试只新增 `run_attempt`。 |
| execution 完成语义 | `run_task` 才是 worker lease 单元；attempt append-only；命令退出码 0 不等于完成，必须有 verify evidence；worker 失联先进入 recovery，不直接判失败。 |
| run_task 状态模型 | `run_task.status` 只表达通用 worker/runtime 生命周期；`verified`、`artifact_written`、`rollback_verified` 等能力专属结果应进入 `outcome`、attempt、artifact、gate result 或 summary。 |
| approval 语义 | approval 必须记录 actor、scope、risk、允许能力、资源范围、gate snapshot、有效期和 audit event；不能只是一句“同意”。 |
| AreaMatrix 迁移 | Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover -> Archive -> Shim Retirement。 |
| Artifact 策略 | PG 存 metadata/hash/URI/relation；大内容进 local artifact store，历史 AreaMatrix 原文先保持 project reference。 |
| Backup / restore 策略 | v1.0 只交付 backup manifest、artifact integrity、restore dry-run 和 archive preview；真实 restore apply 是 v1.x R4 能力。 |
| Object / retention / GC 策略 | v1.0 object backend 只能是 skipped/needs_attention；object verifier、archive copy/upload、GC/delete apply 全部留到 v1.x。 |
| 写入策略 | 默认只读；写项目、执行命令、网络、secret、git 和 agent execution 都必须走 capability、allowlist、gate、approval、audit。 |
| 多项目模型 | `project_key` 是 workflow、run、lease、artifact、secret、audit 和 API 查询的隔离边界；AreaMatrix 不是 schema 特例。 |
| workspace / environment | v1.0 前不单独引入复杂 workspace/environment 表；先由 team/project grouping、project connections、worker kind、engine profile 和 scheduling metadata 表达。 |
| worker 调度模型 | PostgreSQL row lock + scoped lease 是 v1.0 前主调度机制；schedule preview 只读，真实 scheduler 另行开闸；AreaMatrix 第一阶段真实 execution 并发固定为 1。 |
| v0.8 多项目 worker pool | v0.8 只证明多项目 worker pool 状态可观察、可解释、可隔离；`recommended`、`available_slots` 和 `next_action` 不能解释为 scheduler apply、lease claim、worker dispatch、secret resolve、remote worker credential、team/auth enforcement 或 AreaMatrix execution cutover。 |
| Command / approval 模型 | Query、preview、readiness 和 gate 只解释状态；业务写动作必须进入 Command API，并按 command class、permission、risk、gate、approval、expected version/hash、write-set、audit 和 rollback 顺序执行。 |
| v1.x high-risk apply | post-100% 能力统一按 `high-risk-apply-ladder.md` 从 `closed`、`preview_only`、`fixture_only`、`scoped_rollback` 到 `retained_beta` / `production_scoped` 逐级打开。 |
| 多端策略 | CLI、Web、Desktop、Worker 都是 API client；API/service layer 是唯一业务边界，Command API 是唯一业务写入口。 |
| API 分类 | Query API 只读，Command API 承载所有写动作，SSE/Event API 只观察，Admin API 只做受限运维入口且不得绕过业务门禁。 |
| 目录结构 | 采用 `cmd/`、`internal/`、`migrations/`、`docs/`、`governance/`、`workflow/`、`examples/`、`web/`、`tasks/`、`scripts/`，后续按阶段补 `schemas/`、`desktop/` 和拆分内部模块。 |
| AreaMatrix 最终入口 | AreaMatrix 最终只保留 `areaflow.yaml`、`workflow/README.md` 粗略入口、`.areaflow/status.json` 工具入口、兼容命令和退役说明。 |
| Compatibility shim v0 | 第一版只做薄转发和强阻断；`./dev workflow init` 不带 `--write` 时保持 preview，带 `--write` 才转 AreaFlow Command API。 |
| Shim lifecycle | `not_installed -> read_only_shim -> execution_forwarding -> retired_thin_entry`；read-only shim 不等于 execution cutover，execution forwarding 不等于 retirement。 |
| 真实写入开闸顺序 | 先 generated-only rollback drill，再 retained generated apply，再 manual patch artifact / human-applied evidence，最后才进入 source write beta。 |
| Engine 打开顺序 | source write beta 后先开 no-secret engine execution；顺序为 manual worker、本机 no-secret command、Codex CLI no-secret，再进入 secret resolve。 |
| Auth / team / secret 顺序 | v1.0 只做 schema/readiness；真实 token enforcement、team permission、secret resolve 和 remote worker credential 按 R4 ladder 进入 v1.x。 |
| 多端控制面 | Web、Desktop、Worker pool 先做观察和 preview；写操作只能通过 Command API、risk preview、permission preflight、approval 和 audit。 |
| v0.9 Desktop Shell | Desktop 是本机 API client 和 dashboard launcher，不是第二个 AreaFlow；v0.9 只显示 service status、service-control gate、notification gate、tray/menu gate 和 selected project readiness，不打开 process control、OS notification、native tray/menu、secret resolve、team console 或 remote ops。 |
| v1.0 Stable Platform | v1.0 只能由 completion audit 证明 100%；release final gate、package preview、Web/Desktop 展示或 smoke 通过都不是充分条件。AreaMatrix dogfood 必须证明到 Archive 和 Shim Retirement，v1.x high-risk apply 必须保持关闭或拥有独立 approved apply packet。 |
| Release 模型 | v1.0 只做 release readiness、exception、final gate、evidence、package/distribution/publish/rollout preview；真实 package、tag、sign、push、upload、publish 放到 v1.x。完整合同见 [`release-final-gate-contract.md`](../contracts/release-final-gate-contract.md)。 |
| Completion audit | 100% 不能靠口头判断、绿色测试或 release final gate 单独证明；必须按 [`completion-audit-contract.md`](../contracts/completion-audit-contract.md) 聚合 phase/task、AreaMatrix dogfood、release、ops、isolation 和 protected path evidence。 |
| 运维 / 部署 / 可观测性 | v1.0 只做本机 install/migrate/start/status/doctor、metadata-only support bundle preview、本地日志脱敏和 telemetry local-only；远程运维控制、托管升级、破坏性 rollback 和完整 support export 放到 v1.x。完整合同见 [`operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md)。 |
| AreaFlow self dogfood | Schema 从 day 1 支持 self project；v0.1-v0.2 不依赖自管理，v0.3 后只读 dogfood，v0.5 后 dry-run / artifact-only，v1.0 才稳定自托管。 |
| Runner Preview | v0.5 只证明 run/task/attempt/artifact/event/audit 可表达；run control 只改 dry-run DB 状态；不启动 worker、不调用 engine、不写项目。 |
| Worker Beta | v0.6 只打开 worker lifecycle、lease、dry-run run-once 和 scoped execution evidence；fixture/read-only/artifact-only/rollback drill 证据不能累计成 AreaMatrix execution cutover。 |

以下议题尚未关闭，必须按阶段形成 gate evidence 后才能进入对应实现：

| 未关闭议题 | 关闭阶段 | 关闭前禁止 |
|---|---|---|
| AreaMatrix 兼容 shim 真实落地 | v0.4 后、单独授权 | 不修改 AreaMatrix 仓库，不写 shim 文件，不转发 `./task-loop run`。 |
| `workflow/README.md` 受控区块写入 | Authoring Cutover gate 通过后 | 不写真实 AreaMatrix `workflow/README.md`。 |
| approved project write 的最小打开方式 | v0.6 后续受限 execution 设计 | 不写被管理项目文件，不打开 copy/repair/checkpoint。 |
| Codex CLI / engine 真实 execution | Execution Beta 后续 gate | 不运行 shell，不调用 engine，不解析 secret。 |
| copy / verify / repair / checkpoint apply 与 rollback | Execution Beta -> Execution Cutover | 不把命令成功当作完成，不跳过 verify，不自动 git checkpoint；早期 scoped task status 需要兼容清理后再作为稳定 API 口径。 |
| AreaMatrix execution cutover | v1.0 dogfood gate | 不替代 AreaMatrix `./task-loop` 主能力，不重写 v1 历史 execution。 |
| Web 写操作 | Command API 风险面稳定后 | Web v0.7 只读；write action gate 只展示 disabled/read-only 动作，不直接写 DB、project files、artifact store 或 worker state。 |
| Desktop service control | v0.9 设计和 smoke 后 | Desktop 不维护第二状态源，不直接执行 workflow。 |
| 多用户、远程 worker、secret manager、第三方 plugin execution | v1.x | v1.0 前不开放团队控制台、远程执行、真实 secret resolve 或未知 plugin 执行。 |
| external integrations / webhook delivery / external API connector | v1.x | v1.0 前不投递 webhook、不处理 callback 为业务事实、不调用外部 API。 |
| Auth / team / secret enforcement | v1.x R4 ladder | 不启用 bearer auth、team role enforcement、secret resolve、remote worker credential 或 secret-backed engine。 |
| object artifact store / archive copy / GC delete apply | v1.x | 不把 object metadata 当作可恢复原文，不上传、不复制、不删除、不清理历史 project reference。 |
| budget / quota enforcement | v1.x | 不扣减 quota、不写 charge、不同步 provider billing、不 silent throttle。 |
| restore apply / publish apply | v1.x | v1.0 前只做 preview/gate，不执行恢复、tag、push、sign、upload 或 publish。 |
| remote ops / managed upgrade / support export | v1.x | 不做远程运维控制、不自动升级、不破坏性 rollback、不导出包含 prompt、secret、用户文件或 raw artifact 的 support bundle。 |

未关闭议题不是路线空白；它们是刻意保留的高风险门禁。对应阶段可以继续做 preview、doctor、
readiness、fixture smoke 和 artifact-only 证据，但不能把预览结果解释为真实 apply 已打开。

## 执行前最终确认清单

进入文档固化或实现推进前，以下事实视为 0-100% 路线的最终确认口径：

1. AreaFlow 是独立平台仓库；AreaMatrix 是第一个 dogfooding project，不是子模块。
2. 技术栈固定为 Go + PostgreSQL + REST/JSON + SSE；Web 是 React + TypeScript；Desktop 是 Tauri。
3. PostgreSQL 是主状态源事实；文件只承载项目配置、artifact 原文、projection 和审计导出。
4. AreaMatrix 迁移顺序固定为 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta
   -> Execution Cutover -> Archive -> Shim Retirement。
5. Authoring cutover 不等于 execution cutover；v0.4 只切新 workflow version authoring 所有权。
6. v1.0 的 100% 是稳定平台边界，不是所有高风险自动化能力全部打开。
7. Preview / readiness / gate 只证明设计、边界和 go/no-go；只有受保护 Command API apply
   通过 permission、approval、rollback 和 audit 后，才代表真实能力打开。
8. AreaMatrix 最终保留项目事实、粗略入口、compatibility commands 和退役说明；AreaFlow 拥有
   workflow、run、attempt、artifact、worker、audit 和 release readiness 主状态。
9. v1.0 前不得把真实 AreaMatrix generated retained apply、source write、repair apply、
   checkpoint apply、secret resolve、remote worker、restore apply、publish apply 或 third-party
   plugin execution 解释为已打开。
10. 任何修改 AreaMatrix shim、`workflow/README.md`、`.areaflow/status.json` 或 `scripts/**`
    的动作，都必须另有明确授权；AreaFlow 规划文档不能替代跨仓库写入批准。
11. AreaMatrix 第一阶段 execution 并发默认固定为 1；多项目 worker pool 先做只读 summary 和 schedule
    preview，真实 scheduler、远程 worker 和自动跨项目调度必须另行开闸。
12. `write_artifacts` 不等于写被管理项目文件；`write_generated` 不等于写源码；`approval` 不等于
    execution；`readiness` / `preview` / `gate` 不等于 apply。
13. release exception 只能接受 `metadata_only_history`、`future_only_gap` 和 `archive_exception`；
    permission fail、adapter/profile fail、backup broken、local artifact hash mismatch、secret 泄露风险和
    无 rollback 的真实写入不能靠 exception 放行。
14. `project_reference` / `external_project` 不会因为被索引或产生 hash 就变成 AreaFlow-owned content；
    object verifier 未通过前，object backend 不能计入完整可恢复内容；普通 GC 永远不能触碰 `audit`、
    `release`、受保护的 `run_evidence`、`external_ref`、`legal_hold` 或未知 retention class。
15. AreaFlow self dogfood 不早于 AreaMatrix 主 dogfood 路线；AreaFlow 自己的源码写入、release publish、
    secret resolve 和 plugin execution 遵循同一套 v1.x 高风险开闸顺序。
16. Telemetry 默认只留本机；support bundle v1.0 只做 metadata-only preview，真实导出、远程上传、
    托管升级和 rollback apply 都必须进入 v1.x 运维开闸。
17. Release final gate 是 100% 必要条件但不是充分条件；只有 completion audit 按当前证据逐项通过后，
    才能声明 AreaFlow 0-100% 完成。

因此，当前实现证据中出现 `implemented_scoped` 或 `preview_only` 时，只能说明对应边界已经可审计或
可预演；它不能被累计成真实 AreaMatrix execution cutover 已完成，也不能绕过 v1.x 高风险开闸顺序。

## 0-100% 阶段

| 范围 | 阶段 | 核心目标 | 明确不做 |
|---:|---|---|---|
| 0-5% | Phase 0 Foundation | 产品、架构、目录、技术决策、迁移协议和 v0.1 边界 | 不执行任务，不接管 workflow |
| 5-15% | v0.1 Import + Status Mirror | 导入 AreaMatrix metadata，生成粗略状态，建立 PG 源事实 | 不写 `workflow/**`，不调用 AI engine |
| 15-25% | v0.2 Shadow Doctor + Drift Check | doctor 等价校验、hash drift、stage coverage、readiness bundle | 不创建新 version，不 cutover |
| 25-35% | v0.3 New Version Authoring | 在 AreaFlow 创建 workflow version、stage skeleton、gate、preview、approval | promotion preview 不 apply |
| 35-45% | v0.4 Workflow Ownership Cutover | 新 workflow version authoring 源事实切到 AreaFlow | 不替代 task-loop，不自动改代码 |
| 45-55% | v0.5 Runner Preview | 建 run/task/attempt/permission preflight，只 dry-run | 不真实执行，不写项目文件 |
| 55-65% | v0.6 Worker Execution Beta | 执行已批准任务，lease/heartbeat，copy/verify/repair | 不重写 AreaMatrix v1 历史 |
| 65-75% | v0.7 Web Dashboard | 多项目 summary、timeline、stage、artifact、blocker、SSE | Web 不绕过 API，不直接写项目，不维护第二状态 |
| 75-85% | v0.8 Multi-project Worker Pool | 多项目 worker pool summary、schedule preview、agent role、resource readiness 和 engine readiness | 不打开真实 scheduler、secret resolve、remote worker 或 team/auth enforcement |
| 85-92% | v0.9 Desktop Shell | Tauri shell、local service status、desktop gates、通知路径预览、项目切换 | Desktop 不维护第二套状态，不打开 process control |
| 92-100% | v1.0 Stable Platform | completion audit、release final gate、backup/restore dry-run、ops readiness、AreaMatrix dogfood 闭环 | 不把 high-risk apply 或 preview 当作 100% 完成 |

阶段必须按顺序证明门禁。没有 gate evidence，不进入下一阶段；未知缺口不能作为 `warn` 放行。

## 地基决策

### 1. Workflow Engine

AreaFlow 使用通用 stage engine + workflow profile。

Core 只理解：

```text
stage
workflow_item
artifact
gate_result
transition_preview
approval_record
run
audit_event
```

Core 不硬编码 AreaMatrix 的目录，例如 `discussion/`、`middle-layer/`、`changes/`、`plans/`、
`drafts/`、`queue/`、`execution/`。这些目录通过 AreaMatrix adapter/profile 映射成通用对象。

AreaMatrix profile 第一版 stage 链路为：

```text
intake
-> source_docs
-> templates
-> version_init
-> discussion
-> middle_layer
-> changes
-> plans
-> drafts
-> queue
-> promotion_preview
-> approval
-> execution
-> run
-> projection
-> closeout
```

`workflow_item` 是 stage 内语义交付物，不是单个文件，也不是执行任务。文件、prompt、报告、diff、
日志和 evidence 都是 artifact。真实执行进入 `run_task` 和 `run_attempt`。

`promotion_preview`、`approval`、`live_mapping_gate` 和 `execution_permission_gate` 必须分开；
preview 不等于 approval，approval 不等于 execution。

Execution 前允许通过 gate 重新预览、回退或重开 planning artifact；进入 execution 后不做破坏性
rewind。修正只能通过新的 run、attempt、projection、closeout amendment 或后续 workflow version 追加表达，
不得覆盖历史 artifact、event、audit event 或 approval 事实。

### 2. Data Model

PostgreSQL 是 AreaFlow 主状态源事实，不提供 SQLite 主状态 fallback。

Day 1 按多项目、多用户、多 worker 的长期模型设计，但能力分阶段打开。v0 真正启用：

```text
actors
projects
project_connections
project_permissions
workflow_versions
workflow_items
workflow_item_links
gate_results
workflow_transition_previews
approval_records
artifacts
runs
events
audit_events
command_requests
status_projections
project_status_snapshots
```

先建边界、晚开能力：

```text
users
teams
memberships
api_tokens
webhooks
secret_refs
engine_profiles
artifact_locations
artifact_snapshots
project_configs
adapters
workflow_profiles
```

状态必须分层表达：

```text
phase_state       -> workflow_versions / workflow_items / profile
gate_state        -> gate_results
approval_state    -> approval_records
runtime_state     -> runs / run_tasks / run_attempts
lease_state       -> leases / worker_heartbeats
projection_state  -> status_projections
security_state    -> audit_events / permissions / command_requests
```

不能用一个万能 status 字段表达全部事实。

所有业务写动作必须进入 `command_requests`，并具备 idempotency key、request hash、risk、permission、
approval、precondition snapshot、affected resources、safety facts 和 audit 边界。同一幂等键携带不同
request hash 必须拒绝。统一写入口和 apply 顺序见
[`../architecture/command-approval-contract.md`](../contracts/command-approval-contract.md)。

### 3. Permission And Write Policy

AreaFlow 对被管理项目默认只读。任何写入、命令执行、网络访问、secret 使用、git 操作或 agent execution
都必须显式授权。

Capability 固定分层：

```text
read_project
write_status
write_artifacts
write_workflow
write_generated
write_code
run_commands
manage_workers
manage_git
network
use_secrets
execute_agents
```

风险等级：

```text
R0 read_only
R1 projection
R2 managed_write
R3 execution
R4 migration_security
```

写入判断公式：

```text
capability allowed
+ resource allowlist matched
+ forbidden list not matched
+ command class allowed
+ required gate passed
+ approval present when needed
+ expected-before hash matched when writing existing project files
+ write-set valid when project write is attempted
+ rollback plan present when project write is attempted
+ audit event written
= allowed
```

Deny 永远优先于 allow。

`write_artifacts` 只允许写 AreaFlow-owned artifact store 和 artifact metadata evidence，不代表可写
被管理项目文件。`write_generated` 只允许写显式 allowlist 的 generated/projection 前缀，不代表可写
source code、execution、progress、logs 或 checkpoint。

AreaMatrix v0.1-v0.3 只允许只读导入、AreaFlow-owned artifact 写入和 `.areaflow/status.json`
projection；不允许写 `workflow/versions/**`、execution、progress、logs、checkpoint 或源码。

### 4. Runner And Worker

AreaFlow 最终替代 `./task-loop run` 主能力，但不迁移 `progress.json` 作为主状态。
真实 execution 的分阶段打开规则见
[`../architecture/execution-opening-strategy.md`](../../../../proposals/execution-opening.md)；
任何 ready / preview / gate 都不能绕过该开闸阶梯。

职责分层：

```text
Runner:
  创建 run，拆分 run_task，检查 gate / approval / permission / risk。

Worker:
  注册、heartbeat、领取 scoped lease，执行一个 run_task，提交 attempt / artifact / event / audit。

Engine Adapter:
  封装 Codex CLI、OpenAI API、local model 或 external agent。

Project Adapter:
  理解被管理项目目录、doctor、验证命令和项目语义。
```

Worker 不能直接领取 `workflow_item`，只能通过 lease 领取 `run_task`。同一个 `run_task` 同时最多一个
active lease。多项目 worker pool、slot 计算、recovery 和真实 scheduler 开闸规则见
[`../architecture/worker-scheduling-contract.md`](../contracts/worker-scheduling-contract.md)。

执行证据必须分开：

```text
copy attempt
verify attempt
repair attempt
checkpoint attempt
```

Copy 成功不代表 done；verify PASS 才能进入验收；checkpoint fail 会阻断下一任务。
Repair plan 不等于 repair apply；repair apply 成功后必须重新 verify。Checkpoint preview 不等于 checkpoint
apply；checkpoint apply 必须单独 command、approval、dirty-state/scope-drift check、rollback/remediation 和 audit。
局部 scoped proof 不能累计成 AreaMatrix execution cutover。

真实执行打开顺序：

```text
runner preview
-> worker dry-run
-> fixture execution
-> read-only verify on managed project
-> approved artifact write
-> approved project write
-> checkpoint
-> repair
```

### 5. API, CLI, Web And Desktop

第一阶段可以 CLI-only 使用，但架构上不能做成 CLI 单体。API / service layer 是唯一业务边界：

```text
CLI / Web / Desktop / Worker
  -> REST API + SSE
    -> Service Layer
      -> PostgreSQL
      -> Artifact Store
      -> Workflow Stage Engine
      -> Project Adapters
      -> Worker Pool
      -> Engine Adapters
```

API 分三类：

```text
Query API:
  只读查询，不写文件、不执行命令、不解析 secret、不调度 worker。

Command API:
  所有写入、执行、审批、projection、cutover、restore、publish 和 worker 控制入口。

SSE:
  事件观察通道，不是状态源；断线后必须通过 Query API 补齐事实。
```

Web v0.7 先做 dashboard 和只读观察；写入动作必须等 Command API、risk preview、permission preflight、
影响范围和 audit outcome 稳定后再打开。

Desktop v0.9 是 local service shell、dashboard launcher、desktop gate 和健康观察面，不维护第二数据库，
不直接执行 workflow，也不打开真实 process control、OS notification、native tray/menu 或远程 Team Console。

### 6. AreaMatrix Migration And Cutover

AreaMatrix workflow 迁移路线固定为：

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

Authoring cutover 和 execution cutover 必须分离。

v0.4 cutover 只表示：

```text
新 workflow version authoring 源事实从 AreaMatrix 目录切到 AreaFlow PostgreSQL + artifact store
```

它不表示：

```text
task-loop cutover
execution cutover
promotion apply
自动写 execution/**
自动改代码
重写 v1 历史 workflow
```

Cutover 前必须证明：

```text
import_coverage pass
status_mirror pass
hash_drift pass
stage_coverage pass 或 accepted warn
workflow doctor equivalent pass
version_authoring pass
transition_preview pass
approval_gate pass
live_mapping_gate pass
compatibility_shim pass
rollback_plan pass
write_permission pass
audit_events pass
cutover_readiness_gate pass
```

`./task-loop run` 在 execution cutover 前必须 blocked。
Shim lifecycle 必须显式记录：`read_only_shim` 只做只读/authoring 转发，`execution_forwarding` 只转发
AreaFlow Command API，`retired_thin_entry` 只退役旧主执行能力，不删除历史 workflow/progress/log/evidence。

### 7. Repository Structure

AreaFlow 使用 Go 项目常见结构，但按平台边界逐阶段演进。长期目标：

```text
AreaFlow/
  cmd/areaflow/
  internal/
    api/
    app/
    auth/
    audit/
    config/
    db/
    migrate/
    project/
    workflow/
    artifact/
    adapter/
    doctor/
    status/
    importer/
    worker/
    runner/
    engine/
    integration/
    secret/
    permission/
  migrations/
  docs/
  governance/
  workflow/templates/
  workflow/profiles/
  examples/
  schemas/
  web/
  desktop/
  tasks/
  scripts/
```

短期不创建空抽象。`internal/project` 可以承载早期聚合逻辑，后续按阶段拆出 `workflow`、`runner`、
`worker`、`permission`、`engine`、`secret` 和 `integration`。

暂不创建 `pkg/`。只有当 Go SDK、plugin SDK、adapter SDK 或外部 client library 稳定后，才引入公共包。

AreaFlow 仓库的 `workflow/` 只保存平台模板和 profile，不保存 AreaMatrix 历史 workflow 主状态。

### 8. Artifact, Backup And Archive

Artifact 原文不直接进入 PostgreSQL。PG 只保存 metadata、URI、hash、size、type、retention class 和关系。
完整 artifact / backup / restore 合同见
[`../architecture/artifact-backup-restore-contract.md`](../contracts/artifact-backup-restore-contract.md)；
object backend、archive copy/upload、GC/delete 的长期合同见
[`../architecture/object-artifact-retention-contract.md`](../contracts/object-artifact-retention-contract.md)。

Artifact 分三类：

```text
historical reference:
  历史 AreaMatrix 文件，只索引 metadata，不复制原文。

managed copy:
  AreaFlow 新生成或拥有的 workflow skeleton / prompt / approval bundle / report。

generated evidence:
  runner / worker / doctor 产生的日志、报告、diff、failure summary、verify evidence。
```

历史 AreaMatrix artifact 第一阶段为 `project_reference` / `external_project`，不能被当作完整可恢复原文。
只要历史原文仍留在 AreaMatrix 或其他被管理项目中，restore plan 和 release readiness 必须返回
`needs_attention`，除非存在明确 release exception、owner、证据、风险说明和后续归档计划。
索引、hash 或 metadata 不等于接管原文，也不赋予复制、上传、移动或删除权限。

AreaFlow 新产物写入 local artifact store：

```text
~/.areaflow/artifacts/{project_key}/...
```

v1.0 前只交付 backup manifest、artifact integrity、restore dry-run plan 和 metadata-only archive preview。
Object artifact store 在 verifier 落地前只能返回 `skipped` / `needs_attention`。真实 archive copy/upload、
retention-aware GC、delete apply 和 restore apply 属于 v1.x R4 能力，必须单独设计 approval、验证、
restore impact、回滚或 revoke。

### 9. Multi-project, Users, Engine, Secret And Plugin

所有核心对象必须带 project scope，避免多项目状态污染。

本机单用户模式也必须使用稳定 actor，例如：

```text
system
local-user
human
service
worker
api-token
cli-token
agent
areamatrix-shim
```

多用户和团队 UI 后续打开，但审计来源从 day 1 保留。

Engine profile 在 v1.0 前只参与 readiness、schedule preview、risk preview 和 blocked reason。项目配置只写
`secret_ref`，不写明文 secret。v1.0 前不解析 secret、不注入 worker、不调用需要 secret 的真实 engine。

Adapter、profile、plugin 分离：

```text
Adapter = project IO
Profile = workflow semantics
Plugin = governed extension
```

AreaMatrix adapter/profile 可以先内置。第三方 plugin 安装、签名、版本兼容和沙箱放到 v1.x。
Plugin / marketplace seed、manifest、install / enable / execute ladder 和 AreaMatrix first policy 见
[`../architecture/plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)。

### 10. v1.0 Release Gate

v1.0 是稳定平台边界，不是高风险自动化边界。v1.0 必须证明状态可恢复、权限可解释、证据可审计、
发布可预演、adapter/profile/plugin 边界稳定。

Release final gate、exception 和 publish preview 的统一语义见
[`../architecture/release-final-gate-contract.md`](../contracts/release-final-gate-contract.md)。该合同是
v1.0 release 链路的源事实；milestone、API 和 backlog 只展开各自阶段的交付项。

v1.0 不默认打开：

```text
真实 restore apply
真实 secret resolve
未知第三方 plugin 执行
自动 tag / push / sign / upload / publish
远程多用户控制台
远程 worker
```

Release final gate 证据链：

```text
backup manifest
-> restore dry-run plan
-> artifact integrity
-> audit coverage
-> permission doctor
-> adapter/profile conformance
-> release readiness
-> remediation plan
-> acceptance preview
-> acceptance gate
-> exception doctor
-> exception record preview
-> exception schema preview
-> exception migration approval gate
-> exception apply preview
-> release final gate
-> release evidence bundle
-> release package preview
-> distribution preview
-> publish gate
-> publish approval preview
-> rollout plan preview
```

可以显式接受的 release exception：

```text
metadata_only_history
future_only_gap
archive_exception
```

不能通过 exception 放行：

```text
backup manifest broken
restore dry-run guardrail missing
permission policy fail
adapter/profile conformance fail
profile hash drift 未解释
local artifact hash mismatch
enabled capability audit gap
secret 泄露风险
project_key isolation 失败
Command API 幂等/审计缺失
真实写入无 rollback
AreaMatrix protected path proof 缺失
未知状态或未知 category
```

## AreaMatrix 最终形态

AreaMatrix 最终保留：

```text
docs/**
source code
validation commands
project governance
release evidence
user-file safety boundary
workflow/README.md 粗略入口
.areaflow/status.json 工具入口
compatibility commands
retirement notes
```

AreaFlow 最终拥有：

```text
workflow versions
discussion gates
middle-layer
changes
plans
drafts
queue
promotion preview
approval
execution runs
attempts
events
artifact index
worker scheduling
audit
release readiness
evidence chain
```

## Post-100% v1.x

以下能力必须留到 v1.x，不能塞进第一版 100%：

- 真实 restore apply。
- Secret manager 和真实 secret resolve。
- OpenAI API、Codex CLI、多 engine provider 的真实 secret 注入。
- 远程 worker。
- 团队、多用户、远程控制台。
- API token 和远程权限操作面。
- 第三方 plugin marketplace 和 plugin 执行。
- External API / webhook / GitHub / notification provider integration。
- 对象存储 artifact backend。
- 发布自动化：tag、push、sign、upload、publish。
- 远程运维控制、托管升级、破坏性 rollback、完整 support bundle export 和默认远程 telemetry。
- 多组织、多租户。
- 成本、配额、预算控制。

v1.x 高风险能力必须按以下顺序逐步打开，不能因为某个 preview、readiness 或 gate 返回 `ready`
就隐式获得 apply 权限：

统一状态词、apply packet、suspension rule 和 AreaMatrix first policy 见
[`../architecture/high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md)。本节只保留总顺序；
具体开闸证据以该合同为准。

1. **真实 generated-only rollback beta**：从 fixture/temp rollback drill 进入真实 managed project 的
   generated/projection 单文件写入演练。只允许已存在普通文件，必须 expected-before、preimage、verify、
   rollback 和非目标文件指纹证据完整；写入结果必须恢复到 preimage。
2. **真实 generated-only retained apply**：在 rollback beta 证据稳定后，才允许真实 managed project
   generated/projection 单文件保留写入。只允许 `.areaflow/generated/**`、`.areamatrix/generated/**`
   或项目配置显式声明的 generated 前缀；必须有 expected-before、preimage、rollback verify、非目标
   文件指纹不变和 focused smoke。
3. **Manual patch artifact**：AreaFlow 只生成 source patch / diff artifact、write-set preview、
   expected-before、验证命令和 rollback/remediation plan，不写项目源码，不运行 shell，不做 checkpoint。
4. **Human-applied source evidence**：由人工或现有 Codex 流程 apply 源码变更，AreaFlow 只读取 diff、
   changed file hash、验证结果和 evidence，把 copy / verify / checkpoint 语义映射回 run/attempt。
5. **真实 source write beta**：只允许 allowlist 内的小范围源码 `create` / `modify`；不支持 delete、move、
   chmod、binary rewrite、symlink target、glob 批量写入或 project-root 外路径。第一版只到
   copy attempt -> verify attempt -> checkpoint preview；checkpoint apply 和 repair apply 必须单独开闸。
6. **Checkpoint apply**：只在 source write beta 多次证明 verify 稳定后打开；checkpoint 必须有 scope drift、
   dirty state、rollback/remediation 和 audit 证据，失败会阻断下一 task。
7. **Repair plan / repair apply**：先只生成 failure summary 和 repair plan artifact；repair apply 必须追加
   attempt，不能跳过 verify 或 checkpoint gate。
8. **无密钥 engine execution**：先运行 `secret_ref=none` 的 engine/profile。推荐顺序是
   manual worker -> local no-secret command -> Codex CLI no-secret dry scoped execution -> Codex CLI
   no-secret copy/verify execution。网络默认关闭，或只允许明确 allowlist；必须有 budget policy、
   redaction policy、attempt artifact 和 audit。
9. **真实 secret resolve**：进入 R4。项目配置只引用 `secret_ref`，真实 secret 按
   [`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md) 管理；worker 最多获得短期
   scoped binding，stdout、stderr、artifact、event 和 audit 都不得记录明文 secret。
10. **远程 worker**：remote worker 只能通过 API，不直连 PostgreSQL；凭证必须 project-scoped、
   capability-scoped、lease-scoped、可撤销、可轮换，并有 heartbeat 和 audit trail。
11. **真实 restore apply**：在 backup manifest、artifact integrity 和 restore dry-run 之后单独打开。
   必须先定义 restore package 格式、隔离 temp DB/project 验证、diff、approval、rollback 和 audit；
   没有 preimage 与人工批准不得覆盖或删除既有状态。
12. **release exception real write**：只有 exception schema preview、migration approval gate、
   apply preview 和 R4 approval 全部通过后，才允许真实写入 exception record 或 migration path。
13. **真实 publish apply**：tag、sign、upload、push、publish 必须拆成独立 Command API 动作；
   每一步都有 package/evidence hash、approval、rollback 或 remediation plan 和 audit。
14. **第三方 plugin execution**：必须先有 manifest、capability declaration、signature、
   sandbox、conformance、version compatibility、disable/revoke 和 audit；具体合同见
   [`plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)。
15. **External integrations / webhooks**：按
   [`integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md) 逐级打开
   catalog/readiness、delivery plan preview、fixture outbound/inbound、project-scoped delivery、inbound
   callback beta、external API connector command 和 provider automation。Integration 不能绕过 Command
   API、secret scope、network allowlist、audit 或 project scope。
16. **Team console**：按
   [`team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md) 逐级打开
   read-only preview、local auth console、team permission enforcement、remote read-only console 和
   remote command console。多用户控制面只授予可审计控制入口，不自动扩大 project write、secret、
   publish 或 restore 权限；team admin 仍受 project config、Command API、R3/R4 approval 和 audit 约束。
17. **Object artifact store**：按
   [`object-artifact-retention-contract.md`](../contracts/object-artifact-retention-contract.md) 逐级打开
   object verifier、scoped upload、restore dry-run integration、archive copy/upload、GC/delete preview
   和 GC/delete apply；skipped object 不能被当作完整 pass。
18. **Budget / quota enforcement**：按
   [`budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md) 逐级打开 metadata/readiness、
   estimate preview、quota policy doctor、fixture reservation/charge、project-scoped enforcement、
   team/actor/provider aggregation 和 provider billing reconciliation。真实 enforcement 必须有 engine
   cost model、rate limit、quota policy、audit 和 override approval，避免无上限 engine spend、silent
   throttling 或重复扣费。
19. **Managed ops / upgrade / support export**：按
   [`operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md)
   逐级打开 remote read-only ops、remote ops control、managed upgrade/rollback 和 full support bundle
   export。必须具备 auth/team scope、redaction proof、destination allowlist、backup/preimage、approval、
   audit、retention 和 revoke path；不能把 service status、support bundle preview 或 local diagnostics
   当成远程控制或导出授权。

所有 v1.x 高风险 apply 都必须同时具备：

```text
Command API request
idempotency_key + request_hash
actor + reason
command_class
risk_level
affected resources
expected_version / expected-before hash
capability preflight
approval record
precondition snapshot
write-set when project write is attempted
rollback / remediation plan
safety facts
audit event
focused smoke evidence
```

AreaMatrix dogfood 在 v1.x 仍保持保守顺序：先 generated-only，再 source write，再 engine，
再 secret / remote / publish；`workflow/versions/**/execution/**`、`progress.json`、旧 logs 和
checkpoint 在单独授权前继续视为 protected path。

## 当前推进顺序

1. 以本文作为总入口，继续维护 `platform-blueprint.md`、`phase-backlog.md`、ADR 和 milestone 文档。
2. 完成 v0.1-v0.4 已落地能力的 evidence 审计，确保每个阶段都有可复验 smoke。
3. 进入 `AF-V04-001 Compatibility And Shim Readiness`，先完成 AreaFlow 侧 contract 和 read-only readiness。
4. 在没有显式授权前，不修改 AreaMatrix 仓库。
5. 每进入下一阶段前，先补齐 gate evidence，再打开更高风险能力。
