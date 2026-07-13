# AreaFlow 0-100% Phase Backlog

## 定位

本文把 AreaFlow 从当前 0% 规划到 v1.0 100% 稳定平台的路线拆成可执行 backlog。

它不替代：

- [`master-plan.md`](./master-plan.md)：0-100% 总控计划和已锁定地基决策。
- [`platform-blueprint.md`](./platform-blueprint.md)：平台蓝图和长期架构。
- [`roadmap.md`](../../../roadmap.md)：当前路线图入口。
- [`../milestones/README.md`](../milestones/README.md)：milestone 门禁总览。
- [`../development/implementation-gap-audit.md`](../evidence/implementation-gap-audit.md)：当前实现证据和缺口审计。
- [`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)：v0.1
  Import + Status Mirror 最小 CLI、API、读写边界和 v0.2 handoff。
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
- [`../architecture/high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md)：v1.x 高风险
  real apply 的统一开闸阶梯。
- [`../architecture/plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)：v1.0
  plugin / marketplace seed、manifest 和 v1.x plugin execution 边界。
- [`../architecture/release-final-gate-contract.md`](../contracts/release-final-gate-contract.md)：v1.0
  release final gate、exception 和 publish preview 的统一合同。
- [`../architecture/completion-audit-contract.md`](../contracts/completion-audit-contract.md)：0-100%
  完成审计、release packaging preview、dogfood cutover 和 evidence 聚合边界。
- [`../architecture/object-artifact-retention-contract.md`](../contracts/object-artifact-retention-contract.md)：
  object artifact store、archive copy/upload、retention-aware GC 和 delete apply 的 v1.x 边界。
- [`../architecture/integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)：
  external integrations、webhooks、third-party callbacks 和多 API 接入边界。
- [`../architecture/budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md)：budget、quota、
  rate limit 和 usage metering 边界。
- [`../architecture/operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md)：
  install、migration、service lifecycle、diagnostics、support bundle、telemetry、upgrade 和 rollback 边界。

本文回答：

```text
每个阶段做什么
每个阶段明确不做什么
进入下一阶段前必须证明什么
AreaMatrix 在该阶段受到什么影响
哪些能力必须留到 v1.x
```

## 总原则

- AreaFlow 是独立平台仓库，AreaMatrix 是第一个 dogfooding project。
- PostgreSQL 是主状态源事实；文件只用于 project config、artifact 原文、status projection 和审计导出。
- AreaFlow core 不硬编码 AreaMatrix 目录；AreaMatrix 通过 adapter + workflow profile 接入。
- 所有写动作都走 Command API、command class、permission、gate、approval、expected version/hash、write-set、
  rollback/remediation 和 audit。
- Query、preview、readiness 和 gate 只解释状态，不代表 apply；统一写入口合同见
  [`../architecture/command-approval-contract.md`](../contracts/command-approval-contract.md)。
- Query API 只读，不产生写入、执行、secret 读取、worker 调度或状态推进。
- 阶段按顺序推进；没有 gate evidence，不进入下一阶段。
- `warn` 必须有 owner、原因、后续处理和审计证据；未知缺口不能当作 `warn` 放行。
- v1.0 是稳定平台边界，不是高风险自动化边界。

## Self Dogfood 节奏

AreaFlow schema 从 day 1 支持把 AreaFlow 自己注册为 project，但自管理不能早于外部 dogfood
证明。AreaMatrix 仍是第一 dogfooding 主线，AreaFlow self dogfood 是第二主线。

```text
v0.1-v0.2:
  AreaFlow 不依赖自管理；用普通 docs、milestones 和 backlog 管理自身。

v0.3-v0.4:
  AreaFlow 可以作为 read-only self project 注册；只做 metadata、profile、stage 和 readiness 检查。

v0.5-v0.6:
  允许 dry-run、fixture、worker run-once 和 artifact-only dogfood；不写 AreaFlow 源码，不自动 repair。

v0.7-v0.9:
  Web / Desktop 可以展示 AreaFlow self project 状态，但不能把自管理作为唯一开发入口。

v1.0:
  AreaFlow 可以稳定管理自己的新 workflow version；source write、secret、publish 和 plugin execution
  仍按 v1.x 高风险开闸顺序处理。
```

Self dogfood 不能绕过 Command API、permission、gate、approval、rollback 和 audit。AreaFlow 自己的源码
写入不能比 AreaMatrix 更早开闸；AreaFlow release 可以被 AreaFlow preview/gate，但真实 package、
tag、sign、push、upload 和 publish 放到 v1.x。Release final gate、exception 和 publish preview 的具体
边界见 [`../architecture/release-final-gate-contract.md`](../contracts/release-final-gate-contract.md)。

## 0-100% 阶段总表

| 范围 | 阶段 | 核心目标 | 下一阶段门禁 |
|---:|---|---|---|
| 0-5% | Phase 0 Foundation | 定义产品、架构、目录、技术栈、迁移协议和 v0.1 边界 | 设计源事实完整，禁区明确 |
| 5-15% | v0.1 Import + Status Mirror | 只读导入 AreaMatrix metadata，生成粗略状态 | import 可重复，projection 可验证，不写 workflow/execution |
| 15-25% | v0.2 Shadow Doctor + Drift Check | shadow doctor、drift、stage coverage 和 readiness | AreaFlow 能解释与 AreaMatrix 原状态的差异 |
| 25-35% | v0.3 New Version Authoring | 在 AreaFlow 创建新 workflow version 和治理阶段 | version/gate/preview/approval 可审计 |
| 35-45% | v0.4 Workflow Ownership Cutover | 新 version authoring 源事实切到 AreaFlow | cutover readiness pass，只切 authoring |
| 45-55% | v0.5 Runner Preview | 建立 run/task/attempt/artifact dry-run 执行模型 | dry-run 证据完整，risk/permission 可阻断 |
| 55-65% | v0.6 Worker Execution Beta | worker lease、heartbeat、run-once 和 approved task beta | approved execution 有 copy/verify/repair/checkpoint 证据 |
| 65-75% | v0.7 Web Dashboard | Web 观察 project、version、run、artifact、worker、audit | Web 只走 `/api/v1` 和 SSE |
| 75-85% | v0.8 Multi-project Worker Pool | 多项目 worker pool、schedule preview、resource readiness | 所有状态 project-scoped，preview 可解释且不产生调度副作用 |
| 85-92% | v0.9 Desktop Shell | Tauri local service shell、dashboard launcher、desktop gates | Desktop 不维护第二状态源，不打开 process control |
| 92-100% | v1.0 Stable Platform | completion audit、release final gate、backup/restore dry-run、ops readiness、AreaMatrix dogfood 闭环 | completion audit 逐项 complete，preview 不冒充 apply |

## Phase 0 Foundation，0-5%

### 目标

把 AreaFlow 的地基定死，避免后续在技术栈、状态源、目录边界和 AreaMatrix 迁移策略上反复摇摆。

### 做

- 建立独立仓库 `/Users/as/Ai-Project/project/AreaFlow`。
- 固定技术栈：Go + PostgreSQL + REST/JSON + SSE。
- 固定 Web 技术栈：React + TypeScript。
- 固定 Desktop 方向：Tauri。
- 固定 PostgreSQL 为主状态源事实，不提供 SQLite 主状态 fallback。
- 定义长期目录边界：`cmd/`、`internal/`、`migrations/`、`docs/`、`governance/`、`workflow/`、`examples/`、`schemas/`、`web/`、`desktop/`、`tasks/`、`scripts/`。
- 定义 workflow stage engine、数据模型、risk level、capability、artifact 策略、adapter/profile/plugin 边界。
- 定义 v0.1 schema tiers、`areaflow.yaml` day-1 默认值和 persistence mapping。
- 定义 AreaMatrix migration contract。

### 不做

- 不执行任务。
- 不接管 AreaMatrix workflow。
- 不写 AreaMatrix `workflow/versions/**`、execution、progress、logs 或 checkpoint。
- 不打开 Web/Desktop 产品实现。
- 不调用 AI engine。

### 完成门槛

- 产品 charter、platform blueprint、roadmap、milestones 存在并互相引用。
- ADR 覆盖技术栈、PostgreSQL 主状态、workflow stage engine、import/mirror/shadow/cutover 和 operating boundary。
- 架构文档覆盖 data model、workflow lifecycle、API surface、project config、execution model、security/permission、adapter/profile boundary。
- AreaMatrix migration 和 dogfood contract 明确 import、mirror、shadow、cutover、archive、shim retirement 的顺序。

### AreaMatrix 影响

无写入影响。AreaMatrix 只作为第一个被设计支持的 dogfooding project。

## v0.1 Import + Status Mirror，5-15%

### 目标

AreaFlow 能只读理解 AreaMatrix，并生成粗略状态。

### 做

- 注册 AreaMatrix project。
- 按 AreaMatrix import scope 合同读取 workflow/task index metadata、residual ledger、progress summary 和
  artifact metadata。
- 导入 workflow version、stage、residual、artifact metadata 和 project reference。
- 建立 PostgreSQL 中的 AreaMatrix workflow 状态索引。
- 生成 `.areaflow/status.json` 这类轻量 status projection。
- 记录 import event、audit event 和 command request。

### 不做

- 不写 AreaMatrix `workflow/versions/**`。
- 不写 execution、progress、logs、checkpoint。
- 不运行 `./task-loop run`。
- 不调用 AI engine。
- 不读取或复制历史 artifact 大原文。
- 不把 projection 当作主状态源。

### 完成门槛

- `project add`、`project import`、`export-status` 或受保护 projection apply 可重复。
- v0.1 最小闭环符合 [`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)。
- import snapshot 足以支撑 v0.2 的 import diff。
- v0.1 schema tiers 和 `areaflow.yaml` persistence mapping 与
  [`../architecture/data-model-v0.1.md`](../contracts/data-model-v0.1.md)、
  [`../architecture/project-config.md`](../contracts/project-config.md) 一致。
- AreaMatrix import read envelope、minimum import set、explicit non-imports 和 artifact metadata policy 与
  [`../architecture/areamatrix-import-scope-contract.md`](../contracts/areamatrix-import-scope-contract.md)
  一致。
- `.areaflow/status.json` 可在允许路径下生成。
- artifact metadata 有 hash、type、path/URI、project、run/version 关联。
- 能证明未写 AreaMatrix workflow 和 execution。

### AreaMatrix 影响

真实 AreaMatrix 默认只读；只有在明确授权时才允许写 `.areaflow/status.json`。`workflow/README.md` 受控区块写入不在 v0.1 打开。

## v0.2 Shadow Doctor + Drift Check，15-25%

阶段合同见 [`../architecture/v0.2-shadow-doctor-contract.md`](../contracts/v0.2-shadow-doctor-contract.md)。
v0.2 只证明 shadow doctor / drift / readiness 可复验，不证明 cutover 或 execution 已打开。

### 目标

AreaFlow 能与 AreaMatrix 原流程做只读对照，证明自己不是在手写绿色状态。

### 做

- AreaFlow doctor。
- project summary、readiness、import diff、verification bundle。
- hash drift 检测。
- stage coverage 检测。
- native doctor 可选授权对照。

### 不做

- 不创建新 workflow version。
- 不执行 cutover。
- 不自动运行 AreaMatrix 原生命令。
- 不把 native doctor 的未授权状态伪装成 pass。

### 完成门槛

- doctor、summary、readiness、import-diff、verify-bundle 返回稳定 JSON。
- drift、stage coverage 和 native doctor 状态可解释。
- native doctor 未授权时只返回 `skipped` 或 `warn`。
- AreaFlow 能解释自己和 AreaMatrix 原状态之间的差异。
- verification bundle 的 `phase_gate` 保留真实 `pass` / `blocked` 状态；accepted warnings 不能静默升级。

### AreaMatrix 影响

真实 AreaMatrix 仍只读。可运行只读 smoke，必须证明 `.areaflow/status.json` 和 `workflow/README.md` 指纹不变。

## v0.3 New Version Authoring，25-35%

阶段合同见 [`../architecture/v0.3-version-authoring-contract.md`](../contracts/v0.3-version-authoring-contract.md)。
v0.3 只建立 AreaFlow-owned authoring state，不执行 authoring cutover、promotion apply 或 execution。

### 目标

新的 workflow version 可以在 AreaFlow 里创建、规划和治理。

### 做

- 创建 authored workflow version。
- 创建 stage skeleton。
- 记录 workflow item 和 workflow item link。
- 运行或记录 gate result。
- 创建 transition preview。
- 创建 approval record。
- 支持 discussion、middle_layer、changes、plans、drafts、queue、promotion_preview、approval 的 authored 状态。
- 冻结 profile binding：`profile_id`、`profile_version`、`profile_hash`、`adapter`。

### 不做

- promotion preview 不 apply。
- 不写 AreaMatrix execution。
- 不替代 task-loop。
- 不直接改代码。
- 不把 profile 名称当作可变事实；必须绑定 hash。
- 不把 approval record 当作 execution permission。

### 完成门槛

- workflow version、stage skeleton、gate、transition preview、approval record 可审计。
- profile binding drift 可检测。
- authored version 状态来自 AreaFlow，不依赖 AreaMatrix 文件目录作为主状态。
- discussion gate、plan doctor、draft doctor、queue doctor 的失败原因可追踪。
- skeleton artifact 写入 AreaFlow artifact store，而非 AreaMatrix。
- copy-ready / verify-ready 分离。
- approval record 保持 `approval_is_execution=false`，并且只能在 ready transition preview 后批准。

### AreaMatrix 影响

AreaMatrix 仍保留历史 workflow；新 version authoring 可以先在 AreaFlow 中形成候选事实，但尚未切换所有权。

## v0.4 Workflow Ownership Cutover，35-45%

### 目标

新 workflow version 的 authoring 源事实从 AreaMatrix 切到 AreaFlow。

### 做

- compatibility contract。
- shim preview 和 shim readiness。
- shim authorization packet。
- shim lifecycle state：`not_installed` / `read_only_shim`，不进入 `execution_forwarding`。
- approval gate。
- live mapping gate。
- cutover readiness bundle。
- authoring cutover apply。
- compat command 设计。
- rollback plan。

### 不做

- 不切 execution。
- 不替代 `./task-loop run`。
- 不自动写代码。
- 不搬历史 v1 artifact 原文。
- 不让 AreaMatrix 只剩空壳。

### 完成门槛

- compatibility、approval gate、live mapping gate、cutover readiness gate 可证明。
- authoring cutover 只更新 AreaFlow PostgreSQL 状态。
- cutover apply 写入 command request、event 和 audit event。
- 返回事实必须明确 `project_write_attempted=false`、`execution_write_attempted=false`。
- rollback 可回到 project-owned / read-only mirror。
- `./task-loop run` 仍 blocked；read-only shim 不代表 execution cutover。

### AreaMatrix 影响

AreaMatrix 可以出现兼容入口和粗略状态入口，但 execution 仍不由 AreaFlow 替代。`./task-loop run` 在 execution cutover 前必须阻断或清晰说明。

## v0.5 Runner Preview，45-55%

### 目标

AreaFlow 能表达 execution model，但只做 dry-run，不真实执行任务。
阶段合同见 [`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md)。

### 做

- 创建 run。
- 创建 run_task。
- 创建 run_attempt。
- 生成 runner preview report。
- 写 AreaFlow-owned preview artifact。
- 进行 risk preview 和 permission preflight。
- 记录 event、audit event 和 command request。
- 提供 run start、drain、cancel 的 DB-only 控制面。

### 不做

- 不执行 shell。
- 不调用 AI engine。
- 不写被管理项目文件。
- 不领取真实 task。
- 不替代 worker。
- 不将命令退出码等同于完成。

### 完成门槛

- runner preview 能生成完整 run/task/attempt/artifact/event/audit 证据。
- high risk 或缺失 permission 能被阻断。
- preview artifact 可校验 hash、size 和 URI。
- run、run_task、run_attempt 明确 `dry_run` 或 preview 语义。
- command response 明确 `project_write_attempted=false`、`execution_write_attempted=false`、
  `area_matrix_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、
  `secrets_resolved=false` 和 `network_used=false`。

### AreaMatrix 影响

AreaMatrix 仍不被写入。执行模型开始能表达 AreaMatrix task-loop 的主语义，但不接管执行。

## v0.6 Worker Execution Beta，55-65%

### 目标

AreaFlow 开始接管受控任务执行能力，先服务 approved task。
阶段合同见 [`../architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md)；本阶段
所有 execution 证据都必须保留 scope，不得累计成真实 AreaMatrix execution cutover。

### 做

- worker register。
- heartbeat。
- lease acquire、release、recover。
- worker run-once。
- capability preflight。
- copy、verify、repair、checkpoint attempt 分离。
- Codex CLI adapter 初步接入。
- 只执行 approved task。

### 不做

- 不重写 AreaMatrix v1 历史。
- 不跳过 verify。
- 不让 worker 自己决定权限。
- 不给 worker 长期 secret。
- 不执行未批准 task。
- 不把 workflow_item 直接发给 worker。

### 完成门槛

- worker 只领取 run_task。
- lease 有 TTL 和 heartbeat；过期进入 recovery，不直接失败。
- verify pass 才能 mark done。
- checkpoint failure 阻断下一 task。
- repair plan 不等于 repair apply；repair apply 成功后必须重新 verify。
- checkpoint preview 不等于 checkpoint apply；checkpoint apply 必须单独 command、approval 和 audit。
- fixture/read-only/artifact-only/rollback drill 证据不能累计成真实 AreaMatrix execution cutover。
- 每次执行都有 attempt、artifact、event 和 audit。
- capability denial 不创建 lease、attempt 或 artifact。
- fixture/read-only/artifact-only/fixture-temp rollback drill 必须分别标注 scope；`pass` 不能跨 scope 复用。

### AreaMatrix 影响

AreaMatrix 的第一批真实 execution 应优先选择 verify-ready、doctor/readiness 和低风险 artifact evidence。execution cutover 前，`./task-loop run` 仍不能自动转发到 AreaFlow。

## v0.7 Web Dashboard，65-75%

### 目标

Web 成为多项目观察面，不绕过 AreaFlow API。
阶段合同见 [`../architecture/v0.7-web-dashboard-contract.md`](../contracts/v0.7-web-dashboard-contract.md)。

### 做

- project list。
- summary 和 readiness。
- version timeline。
- stage board。
- run timeline。
- artifact metadata。
- residuals。
- approval records。
- worker status。
- audit trail。
- SSE 实时观察。
- shim authorization 只读观察。

### 不做

- Web 不直接写 PostgreSQL。
- Web 不直接读本地 artifact path。
- Web 默认不做 approval、drain、cancel、archive 等写操作。
- Web 不调度 worker。
- Web 不维护第二套状态。
- Web 不把 SSE event 当作主状态。
- Web 不把 disabled write gate 当作写许可。

### 完成门槛

- Web 只走 `/api/v1` 和 SSE。
- 初版请求以 GET/SSE 为主；写操作必须等 Command API 风险面稳定后再打开。
- Browser smoke 必须证明 Web 没有发出非 GET `/api/v1` 请求。
- Run detail、artifact detail 等全局 ID 查询必须带 `project_key` visibility guard。
- Web build 通过。
- browser smoke 证明页面字段、API 调用和只读边界稳定。
- shim authorization 面板只能展示 blocked gate、allowed files 和 safety facts，不打开写动作。

### AreaMatrix 影响

AreaMatrix 通过 AreaFlow Web 查看粗略和详细 workflow 状态；真实项目文件仍不由 Web 直接修改。

## v0.8 Multi-project Worker Pool，75-85%

阶段合同见
[`../architecture/v0.8-multi-project-worker-pool-contract.md`](../contracts/v0.8-multi-project-worker-pool-contract.md)。

### 目标

从 AreaMatrix 单项目扩展到多项目 worker pool 观察面和只读调度预览。

### 做

- worker pool summary。
- schedule preview。
- project priority。
- run/task priority。
- worker type 和 agent role。
- resource limit。
- engine profile readiness。
- secret readiness 只读。
- 多项目隔离。

### 不做

- 不解析 secret 明文。
- 不建立真实远程 worker 凭证体系。
- 不让 schedule preview 直接执行。
- 不让 schedule preview 创建 lease、写 event/audit 或复用真实 acquire path。
- 不打开真实自动 scheduler、远程 worker credential、team/auth enforcement 或多项目 execution apply。
- 不让项目之间共享未授权资源。
- 不牺牲 project scope 隔离。

### 完成门槛

- 每个 run、task、lease、artifact、audit 都有 project scope。
- schedule preview 能解释 recommended、blocked、blocked_reasons、available_slots。
- `max_parallel_tasks` 参与 slot 计算，AreaMatrix 第一阶段真实 execution 并发保持 1。
- resource limit 可见。
- engine/secret 不可用时只返回 blocked reason，不偷偷读取 env、keychain 或 DB secret。

### AreaMatrix 影响

AreaMatrix 不再是唯一项目假设。它仍可作为第一个 host_bound_worker / local_host 场景，因为 macOS app、Xcode 和 GUI 能力不一定适合 container。

## v0.9 Desktop Shell，85-92%

阶段合同见
[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)。

### 目标

Desktop 成为本机 service status shell、desktop gate viewer 和 dashboard launcher。

### 做

- Tauri shell。
- local service status。
- service health。
- service control gate。
- notification gate。
- tray/menu gate。
- shim authorization gate。
- dashboard launcher。
- 项目切换。
- 系统通知路径预览。
- 本机 worker 状态观察。

### 不做

- Desktop 不维护第二套数据库。
- Desktop 不直接执行 workflow。
- Desktop 不绕过 API。
- Desktop 不做远程团队控制台。
- Desktop 不解析 secret 明文。
- process start / stop / restart 在 service-control gate 通过前保持 disabled。
- OS notification permission request、native tray/menu creation 和 settings secret UI 在对应 gate 通过前保持
  disabled。

### 完成门槛

- Desktop 能观察 AreaFlow local service，并通过 service-control gate 展示 start/stop/restart 为什么启用或禁用。
- 所有业务状态来自 `/api/v1`。
- Desktop 可以展示 selected project 的 shim authorization blocked gate，但不能执行 AreaMatrix 编辑。
- service status 明确 `capabilities` 和 `forbidden_actions`。
- service-control gate 明确 `process_control_attempted=false`、`command_created=false` 和 `worker_scheduled=false`。
- notification gate 明确 `event_stream_opened=false`、`notification_requested=false` 和 `worker_scheduled=false`。
- tray/menu gate 明确 `tray_menu_created=false`、`os_integration_requested=false` 和 `service_control_attempted=false`。
- 本机体验能承接 CLI/Web。

### AreaMatrix 影响

AreaMatrix 用户可以通过 Desktop 打开 AreaFlow dashboard 和本机服务状态，但项目真实状态仍来自 AreaFlow API 和 PostgreSQL。

## v1.0 Stable Platform，92-100%

### 目标

AreaFlow 成为稳定平台，而不是实验工具。
v1.0 总阶段合同见
[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)。

本阶段的 release 语义以
[`../architecture/release-final-gate-contract.md`](../contracts/release-final-gate-contract.md) 为准：
final gate 是只读 go/no-go，不是 package、restore、exception write 或 publish apply 授权。
最终 100% 证明以
[`../architecture/completion-audit-contract.md`](../contracts/completion-audit-contract.md) 为准；release
final gate、package preview、Web/Desktop 展示或 smoke 通过都不能单独声明完成。

### 做

- backup manifest。
- restore dry-run plan。
- artifact integrity。
- health / readiness / doctor / service status。
- metadata-only support bundle preview。
- metadata-only archive preview。
- audit coverage。
- permission doctor。
- adapter/profile conformance。
- release readiness。
- remediation plan。
- acceptance preview。
- acceptance gate。
- exception doctor。
- exception record preview。
- exception schema preview。
- exception migration approval gate。
- exception apply preview。
- release final gate。
- release evidence bundle。
- release package preview。
- distribution preview。
- publish gate。
- publish approval preview。
- rollout plan preview。
- security boundary readiness：auth、team、API token、secret 和 remote worker credential 只做 schema /
  readiness / doctor / preview。
- AreaMatrix dogfood execution cutover 闭环。
- plugin/profile marketplace seed、manifest draft 和 conformance，但不执行未知 plugin 代码。

### 不做

- 不做真实 restore apply。
- 不做 archive copy/upload、object storage upload、retention-aware GC、orphan cleanup 或 delete apply。
- 不自动 tag、push、sign、upload 或 publish。
- 不解析真实 secret。
- 不开放未知第三方 plugin 执行。
- 不安装、启用、执行或远程拉取未知 plugin package。
- 不做远程多用户控制台。
- 不做远程运维控制、自动升级、破坏性 rollback 或完整 support bundle export。
- 不默认远程 telemetry。
- 不做 bearer auth enforcement、API token issuance enforcement、team role enforcement 或 remote worker
  credential issuance。
- 不把 release exception 静默放行。
- 不把测试通过当作 release final gate。
- 不让 permission、adapter/profile、backup、local artifact integrity、secret、project isolation 或
  protected path proof 失败通过 exception 放行。

### 完成门槛

- release readiness、acceptance gate、exception apply preview 和 final gate 证据链可复验。
- release final gate `pass` 只允许进入 evidence bundle / package preview / distribution preview /
  publish gate / rollout plan preview，不代表真实发布。
- install / migrate / start / register smoke、health、readiness、doctor、service status 和 support bundle
  preview 可复验。
- support bundle preview 只能是 metadata-only，且明确排除 prompt、secret、用户文件、raw artifact 和未脱敏日志。
- telemetry 默认 local-only，任何远程上报必须保持关闭或显式 opt-in blocked。
- backup manifest、artifact integrity、restore dry-run 能解释可恢复范围和历史 project reference 限制。
- historical project reference / metadata-only artifact 必须显示为 skipped / needs_attention 或明确 exception，
  不能被累计成完整 restore-ready。
- object backend 在 verifier 落地前必须显示为 skipped / needs_attention，不能计入完整可恢复内容。
- archive preview 保持 metadata-only；`project_reference` / `external_project` 不因索引、hash 或 metadata
  变成 AreaFlow-owned content。
- 普通 GC/delete 不得触碰 `audit`、`release`、受保护的 `run_evidence`、`external_ref`、`legal_hold`、
  未知 retention class、hash mismatch local artifact 或 verifier skipped/failed object artifact。
- release exception 只允许 `metadata_only_history`、`future_only_gap` 和 `archive_exception` 三类。
- audit coverage 覆盖写入、命令执行、approval、worker lease、permission change、secret reference 和 release exception 决策。
- permission doctor 能证明默认只读、deny 优先、allowlist、command deny 和 secret/network/git 边界。
- security boundary readiness 能证明 auth enforcement、team permission enforcement、API token issuance /
  enforcement、secret resolve 和 remote worker credential 全部未打开，且 actor、project membership、
  token hash、secret_ref、credential revoke/audit 的长期模型有落点。
- adapter/profile conformance 能证明 AreaMatrix adapter、AreaMatrix profile 和 core 边界稳定。
- plugin / marketplace seed 范围以
  [`../architecture/plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md) 为准。
- AreaMatrix dogfood 完成 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta ->
  Execution Cutover -> Archive -> Shim Retirement。
- AreaMatrix 只保留粗略入口、项目事实、兼容命令和退役说明。
- AreaFlow 拥有 workflow/task execution 主能力。
- 所有写操作都经过 Command API、permission、gate、approval 和 audit。
- Query API 无副作用。

### AreaMatrix 影响

AreaMatrix 最终保留：

- 产品文档和源码。
- 验证规则、治理规则、发布证据和用户文件安全边界。
- `.areaflow/status.json` 机器粗略状态入口。
- `workflow/README.md` 人类粗略状态入口。
- compatibility commands 和退役说明。
- shim lifecycle 至少解释 `read_only_shim`、`execution_forwarding` 和 `retired_thin_entry` 的差别。

AreaFlow 最终拥有：

- workflow version authoring。
- workflow stage / gate / transition / approval。
- run、run_task、run_attempt。
- artifact metadata。
- worker scheduling。
- audit。
- release readiness 和 evidence chain。

## v1.x Post-100% 扩展

以下能力必须保留到 v1.x，不能塞进第一版 100%：

- 真实 restore apply。
- Secret manager 和真实 secret resolve。
- OpenAI API、Codex CLI、多 engine provider 的真实 secret 注入。
- 远程 worker。
- 团队、多用户、远程控制台。
- API token 和远程权限操作面。
- External API、webhook、GitHub 和 notification provider integration。
- 第三方 plugin marketplace 和 plugin 执行。
- 对象存储 artifact backend。
- 发布自动化：tag、push、sign、upload、publish。
- 远程运维控制、托管升级、破坏性 rollback、完整 support bundle export 和默认远程 telemetry。
- 多组织、多租户。
- 成本、配额、预算控制。

v1.x backlog 拆分顺序：

统一状态词、apply packet、suspension rule、R4 串行原则和 AreaMatrix first policy 见
[`../architecture/high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md)。下表是产品
backlog 顺序，不是自动 apply 授权。

| 顺序 | Backlog 主题 | 关闭门槛 |
|---:|---|---|
| 1 | Real generated-only rollback beta | 真实 managed project generated/projection 单文件写入演练后恢复 preimage；expected-before、preimage、verify、rollback 和非目标文件指纹证据完整。 |
| 2 | Real generated-only retained apply | rollback beta 稳定后才允许 generated/projection 单文件保留写入；expected-before、preimage、rollback verify、非目标文件指纹和 focused smoke 完整。 |
| 3 | Manual patch artifact | AreaFlow 只生成 source patch/diff artifact、write-set preview、expected-before、验证命令和 rollback/remediation plan；不写项目源码。 |
| 4 | Human-applied source evidence | 人工或现有 Codex 流程 apply 源码变更；AreaFlow 只读取 diff、changed hash、验证结果并映射 copy/verify/checkpoint 语义。 |
| 5 | Source write beta | allowlist 内源码小范围 `create` / `modify`；`write_code`、write-set、copy/verify、checkpoint preview、rollback 和 audit 全部可查询。 |
| 6 | Checkpoint apply | verify 稳定后单独打开；scope drift、dirty state、rollback/remediation、audit 和失败阻断下一 task 可证明。 |
| 7 | Repair plan / repair apply | 先生成 failure summary 和 repair plan artifact；repair apply 追加 attempt，且不能跳过 verify 或 checkpoint gate。 |
| 8 | No-secret engine execution | `secret_ref=none` engine 可以执行 approved run_task；manual worker / local no-secret / Codex CLI no-secret 逐步打开，budget、redaction、attempt artifact 和 no-secret/no-network safety facts 可证明。 |
| 9 | Secret resolve | short-lived scoped secret binding 可用；明文不进入 project config、artifact、event、audit 或 worker 长期状态。 |
| 10 | Remote worker | API-only remote worker 具备 project/capability/lease scope、token revoke、heartbeat 和 audit trail。 |
| 11 | Restore apply | restore package、isolated dry-run、diff、R4 approval、preimage、rollback 和 audit 形成闭环。 |
| 12 | Release exception real write | schema preview、migration approval gate、apply preview 和 R4 approval 之后才写 exception record。 |
| 13 | Publish apply | tag、sign、upload、push、publish 拆成独立 Command API；每步都有 evidence hash、approval 和 rollout/remediation plan。 |
| 14 | Third-party plugin execution | signed plugin manifest、capability declaration、sandbox、conformance、disable/revoke 和 audit 完整；v1.0 只允许 seed / manifest draft / conformance。 |
| 15 | External integrations / webhooks | 按 [`../architecture/integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md) 完成 catalog/readiness、delivery plan preview、fixture outbound/inbound、project-scoped delivery、inbound callback beta、external API connector command 和 provider automation；禁止 callback 直接改状态、未知 endpoint delivery 和 external API 绕过 Command API。 |
| 16 | Team console | 按 [`../architecture/team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md) 完成 read-only preview、local auth console、team permission enforcement、remote read-only 和 remote command console 阶梯；role matrix、project scope、token/session revoke 和 audit 完整；不能自动获得 project write / secret / publish / restore。 |
| 17 | Object artifact store | object verifier、namespace、hash/size、retention、restore dry-run integration、archive copy/upload 和 GC/delete preview/apply 按合同逐级打开。 |
| 18 | Budget / quota enforcement | 按 [`../architecture/budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md) 完成 metadata/readiness、estimate preview、quota doctor、fixture reservation/charge、project-scoped enforcement、aggregation 和 provider reconciliation 阶梯；engine cost model、rate limit、quota policy、audit 和 override approval 稳定后才执行真实限制，禁止 silent throttling 和重复扣费。 |
| 19 | Managed ops / upgrade / support export | 按 [`../architecture/operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md) 逐级打开 remote read-only ops、remote ops control、managed upgrade/rollback 和 full support bundle export；必须具备 auth/team scope、redaction、destination allowlist、backup/preimage、approval、audit、retention 和 revoke path。 |

任一 backlog 主题进入实现前，都必须先补对应设计门禁；不能用上一主题的 smoke 或 v1.0 release gate
替代当前主题的 approval / rollback / audit 证据。

## 最终完成定义

AreaFlow 达到 100% 时，必须能证明：

- 从空环境可安装、迁移、启动、注册项目。
- AreaMatrix dogfood Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover
  -> Archive -> Shim Retirement 迁移闭环已完成，并且不是靠手写状态伪装。
- 新 workflow version 源事实由 AreaFlow 管理。
- approved task 可由 worker 执行，并留下 copy、verify、repair、checkpoint 证据。
- CLI、Web、Desktop、Worker 都复用同一 API 和 service layer。
- PostgreSQL metadata、AreaFlow-owned artifact metadata 和 local artifact store 可以生成 backup manifest、artifact integrity 和 restore dry-run plan。
- `project_reference` 历史 artifact 的恢复限制被清楚标注。
- adapter/profile/plugin 边界稳定。
- release final gate 不依赖口头判断，所有 blocked / exception 都有 owner、evidence、rollback 和审计路径。
- completion audit 能聚合 phase/task、AreaMatrix dogfood、release packaging preview、operations readiness、
  project isolation 和 protected path proof。

100% 的完成证明必须来自当前状态证据，而不是路线意图或手写摘要。至少需要同时检查：

- `docs/development/implementation-gap-audit.md` 没有把真实 cutover / restore / publish / secret /
  plugin 缺口伪装成 pass。
- `docs/development/task-backlog-status-audit.md` 中 v0-v1.0 task 的 `preview_only` 和
  `implemented_scoped` 都有清楚的剩余边界说明。
- `areaflow project execution-cutover-readiness areamatrix --json` 或对应 API 能证明 AreaMatrix
  execution cutover 的 go/no-go 状态；没有真实 pass 前，`./task-loop run` 不得自动转发。
- release readiness、acceptance gate、exception apply preview 和 final gate 都有可复验 evidence；
  测试通过不能替代 release final gate。
- completion audit 按 [`../architecture/completion-audit-contract.md`](../contracts/completion-audit-contract.md)
  逐项返回 complete；release final gate `pass` 不能替代该审计。
- `project_reference` / `external_project` historical artifact 的恢复限制被 restore dry-run 和
  artifact integrity 明确标记为 metadata-only、skipped 或 needs_attention，而不是完整 pass。

100% 明确不要求打开以下能力；这些能力属于 v1.x 高风险开闸：

- 真实 restore apply。
- 真实 secret resolve 和带密钥 engine execution。
- 远程 worker 和远程团队控制台。
- 未知第三方 plugin 执行。
- webhook delivery、inbound callback processing 或 external API connector。
- 自动 tag、push、sign、upload 或 publish。
- 真实 budget / quota enforcement、usage charge 或 provider billing sync。
- 远程运维控制、托管升级、破坏性 rollback、完整 support bundle export 或默认远程 telemetry。
- 未经单独 approval 的 source write、repair apply、checkpoint apply 或 retained generated apply。
