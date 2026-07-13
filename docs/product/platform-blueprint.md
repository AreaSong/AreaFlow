# AreaFlow Platform Blueprint

## 定位

本文是 AreaFlow 从 0% 到 100% 的平台蓝图。它把产品定位、目录边界、workflow
状态机、多项目模型、worker 执行、权限审计、artifact 策略和 AreaMatrix 迁移节奏收束到同一张路线图。

AreaFlow 是独立平台仓库，不是 AreaMatrix 的子模块。AreaMatrix 是第一个 dogfooding 项目，
AreaFlow 从它的 workflow/task-loop 经验中抽象平台能力，但不把 AreaMatrix 的项目事实写死进 core。
0-100% 总控计划见 [`master-plan.md`](master-plan.md)。
Phase 0 的地基决策以
[`../adr/0005-phase-0-foundation-baseline.md`](../adr/0005-phase-0-foundation-baseline.md)
为准。逐阶段可执行 backlog 见 [`phase-backlog.md`](phase-backlog.md)。
v0.1 Import + Status Mirror 最小闭环和只读边界见
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
v1.x 高风险真实 apply 的统一开闸阶梯见
[`../architecture/high-risk-apply-ladder.md`](../architecture/high-risk-apply-ladder.md)。
运维、部署、可观测性、support bundle、telemetry、upgrade 和 rollback 边界见
[`../architecture/operations-deployment-observability-boundary.md`](../architecture/operations-deployment-observability-boundary.md)。
最终 completion audit、release packaging preview 和 100% evidence 聚合边界见
[`../architecture/completion-audit-contract.md`](../architecture/completion-audit-contract.md)。
Plugin / marketplace seed 与未来 plugin execution 的边界见
[`../architecture/plugin-marketplace-boundary.md`](../architecture/plugin-marketplace-boundary.md)。
Object artifact store、archive copy/upload、retention-aware GC 和 delete apply 的边界见
[`../architecture/object-artifact-retention-contract.md`](../architecture/object-artifact-retention-contract.md)。

## 长期技术栈

| 层 | 决策 |
|---|---|
| Backend / CLI / scheduler / worker | Go |
| 主状态 | PostgreSQL |
| API | REST/JSON |
| 实时事件 | SSE |
| Web | React + TypeScript |
| Desktop | Tauri |
| 配置 | YAML |
| Artifact 原文 | local artifact store 起步，后续对象存储 |
| Queue | PostgreSQL row lock + lease 起步，后续按需扩展 Redis/NATS |

PostgreSQL 是主状态源事实，不提供 SQLite 主状态 fallback。文件只用于 project config、
artifact 原文、轻量 status projection 和审计导出。

## 0-100% 路线

| 范围 | 阶段 | 目标 | 明确不做 |
|---:|---|---|---|
| 0-5% | Phase 0 Foundation | 产品、架构、目录、技术决策、迁移协议和 v0.1 边界 | 不执行任务，不接管 workflow |
| 5-15% | v0.1 Import + Status Mirror | 导入 AreaMatrix metadata，生成粗略状态，建立 PG 源事实 | 不写 `workflow/**`，不调用 AI engine |
| 15-25% | v0.2 Shadow Doctor + Drift Check | doctor 等价校验、hash drift、stage coverage、readiness bundle | 不创建新 version，不 cutover |
| 25-35% | v0.3 New Version Authoring | 在 AreaFlow 创建 workflow version、stage skeleton、gate、preview、approval | promotion preview 不 apply |
| 35-45% | v0.4 Workflow Ownership Cutover | 新 workflow version authoring 源事实切到 AreaFlow，兼容命令 contract 和 rollback 标准 | 不替代 task-loop，不自动改代码 |
| 45-55% | v0.5 Runner Preview | 建 run/task/attempt/permission preflight，只 dry-run | 不真实执行，不写项目文件 |
| 55-65% | v0.6 Worker Execution Beta | 执行已批准任务，Codex CLI adapter，lease/heartbeat，copy/verify/repair | 不重写 AreaMatrix v1 历史 |
| 65-75% | v0.7 Web Dashboard | 多项目 summary、timeline、stage、artifact、blocker、SSE | Web 不绕过 API，不直接写项目 |
| 75-85% | v0.8 Multi-project Worker Pool | 多项目 worker pool summary、schedule preview、agent role、resource readiness 和 engine readiness | 不打开真实 scheduler、secret resolve、remote worker 或 team/auth enforcement |
| 85-92% | v0.9 Desktop Shell | Tauri shell、local service status、desktop gates、通知路径预览、项目切换 | Desktop 不维护第二套状态，不打开 process control |
| 92-100% | v1.0 Stable Platform | completion audit、release final gate、backup/restore dry-run、ops readiness、AreaMatrix dogfood 闭环 | 不把 high-risk apply 或 preview 当作 100% 完成 |

阶段必须按顺序证明门禁，不能为了展示效果提前打开后续能力。

## 总控门禁

每个阶段进入下一阶段前必须有可查询、可复验的 go/no-go 证据。`pass` 表示可以进入下一阶段；
`warn` 必须有明确解释、责任归属和后续处理路径；未知缺口不能作为 `warn` 放行。

| 阶段 | Go / No-Go 证据 |
|---|---|
| v0.1 Import + Status Mirror | 最小闭环符合 v0.1 import / mirror 合同；AreaMatrix 可注册；metadata import 可重复；粗略状态可导出；未写 `workflow/**`、未执行任务。 |
| v0.2 Shadow Doctor + Drift Check | doctor、summary、readiness、import-diff、verify-bundle 返回稳定 JSON；native doctor 未授权时只记录 skipped/warn；phase gate 的 warn/blocked 不被压成 pass。 |
| v0.3 New Version Authoring | AreaFlow 能创建 authored workflow version；stage skeleton、gate、transition preview、approval record 均可审计；approval record 保持 `approval_is_execution=false`；不写被管理项目。 |
| v0.4 Workflow Ownership Cutover | compatibility、approval gate、live mapping gate、cutover readiness gate 和 DB-only cutover apply 可证明；只切 authoring，不切 execution。 |
| v0.5 Runner Preview | run、run_task、run_attempt、artifact、event、audit_event dry-run 证据完整；risk/permission preflight 可阻断；run control 不启动 worker、不执行命令、不写项目。 |
| v0.6 Worker Execution Beta | worker register、heartbeat、lease、run-once、evidence、capability preflight 可审计；scoped execution 仅限 approved task；不打开 AreaMatrix execution cutover。 |
| v0.7 Web Dashboard | Web 只通过 `/api/v1` 展示 project、version、stage、run、artifact、residual、approval、worker、audit；SSE 只做观察。 |
| v0.8 Multi-project Worker Pool | 多项目 worker pool summary 和 schedule preview 稳定；project scope、priority、agent role、resource readiness 可证明；真实 scheduler、remote worker、secret resolve、team/auth enforcement 仍关闭。 |
| v0.9 Desktop Shell | Desktop 能观察 local service、健康、desktop gates 和 Web 入口；不维护第二数据库，不直接执行 workflow，不打开真实 process control。 |
| v1.0 Stable Platform | 符合 v1.0 stable platform 合同；completion audit 证明 release、ops、backup/restore dry-run、isolation、protected path 和 AreaMatrix dogfood 闭环完成。 |

通用验证基线：

```bash
go test ./...
go build ./cmd/areaflow
cd web && npm run build
git diff --check -- .
```

阶段追加 smoke：

```text
v0.1: project add / import / status-projection-apply / export-status
v0.2: doctor / summary / readiness / import-diff / verify-bundle
v0.3: workflow version create / stages / gate / transition / approval
v0.4: compatibility / cutover-readiness / cutover_readiness_gate
v0.5: run preview / run start-drain-cancel
v0.6: worker register / heartbeat / lease / run-once
v0.7: Web build + API-backed page smoke
v0.8: worker pool summary / schedule-preview
v0.9: desktop service health smoke
v1.0: backup / restore + release smoke
```

硬规则：

```text
没有 gate evidence，不进入下一阶段。
warn 必须可解释、可接受、可追踪，不能当作 pass。
任何写项目、执行命令、调用 AI 或解析 secret 的能力，必须具备 approval、permission 和 audit。
```

## 长期目录边界

目标结构如下。没有对应阶段的目录可以暂缓创建，但模块边界按此收敛：

```text
AreaFlow/
  cmd/
    areaflow/

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
    product/
    architecture/
    adr/
    milestones/
    migration/
    dogfood/
    development/

  governance/
    security/
    permissions/
    workflow/
    adapters/

  workflow/
    templates/
    profiles/

  examples/
  schemas/
  web/
  desktop/
  tasks/
  scripts/
```

模块含义：

| 模块 | 职责 |
|---|---|
| `project` | 多项目通用状态、连接、版本和 summary 边界 |
| `workflow` | stage、item、gate、transition、approval、profile engine |
| `adapter` | 怎么读取、映射、可选写入某类项目 |
| `artifact` | artifact 原文存储与 metadata 索引 |
| `permission` | capability、path、command、secret、network、git 判断 |
| `audit` | append-only 安全和权限审计 |
| `runner` | run/task/attempt/dry-run/执行模型 |
| `worker` | worker 注册、lease、heartbeat、recovery |
| `engine` | Codex CLI、OpenAI 和其他 agent/AI engine adapter |
| `secret` | secret reference 解析，不暴露明文 |
| `integration` | GitHub、webhook、外部 API 和远程平台接入 |
| `api` | REST/JSON 和 SSE 的统一业务入口 |
| `app` | CLI 命令编排，不承载核心业务规则 |

短期可以让 `internal/project` 承载部分 workflow 能力；v0.5 前后再拆出 `internal/workflow`，
避免为了目录漂亮做空转重构。

目录落地策略：

```text
v0.1-v0.3:
  保持当前 cmd/internal/docs/examples/workflow/web/tasks 结构，避免空目录和过早抽象。

v0.4:
  `internal/workflow` 开始承载 profile、stage engine、gate 和 transition 核心。

v0.5:
  从 `internal/project` 拆出稳定的 `runner` 边界。

v0.6:
  从 `internal/project` 拆出稳定的 `worker` 和 `permission` 边界。

v0.7:
  补 `schemas/`，沉淀 API、status、profile 和 artifact JSON schema。

v0.8:
  补 `engine`、`secret`、`integration` 的稳定接口；实现仍可分阶段迁移。

v0.9:
  创建 `desktop/`，Tauri shell 只接入 AreaFlow API。

v1.0:
  `auth`、`audit`、plugin / adapter / profile conformance 边界稳定。
```

## Core 模型

AreaFlow core 只认通用对象：

```text
project
project_config
workflow_version
workflow_item
workflow_item_link
gate_result
transition_preview
approval_record
run
run_task
run_attempt
worker
lease
artifact
artifact_location
event
audit_event
```

数据库 schema 采用“早建边界、晚开能力”的策略。`000009_v1_boundary_foundation` 保留
`users`、`teams`、`project_configs`、`artifact_locations`、`secret_refs`、`engine_profiles`、
`api_tokens` 和 `webhooks` 等 v1 平台实体，但不启用登录、secret 解析、webhook 投递或真实
engine execution。

AreaMatrix 的现有目录结构不进入 core。它被表达为：

```text
adapter: areamatrix
workflow_profile: areamatrix
```

内置 profile 使用声明式 YAML 存放在 `workflow/profiles/<profile_id>/profile.yaml`。AreaFlow 必须能读取、
校验并计算 profile hash；authored workflow version 后续绑定的是 profile version/hash，而不是仅绑定
一个可变名称。当前可用只读检查命令：

```bash
areaflow workflow profile check areamatrix
areaflow workflow profile show areamatrix --json
```

创建 authored workflow version 时，AreaFlow 会把冻结结果写入
`workflow_versions.status_summary.profile_binding`：

```text
profile_id
profile_version
profile_hash
profile_path
```

`workflow_version` 是 workflow 生命周期根对象。它表达一个版本从 intake 到 closeout 的整体位置；
stage 内的语义交付物由 `workflow_item` 表达，执行时再派生为 `run`、`run_task` 和 `run_attempt`。
不能把 `workflow_item` 直接当成 worker lease，也不能把某个 artifact 的存在当成 version 已完成。

### Workflow Item 粒度

`workflow_item` 表示 stage 内的一件语义交付物，不等于单个文件，也不等于一次执行任务。

```text
workflow_item = 阶段内可追踪的语义交付物
artifact = 原文、文件、报告、prompt、diff、日志或 evidence
run_task = 进入执行模型后的执行单元
run_attempt = 一次 copy / verify / repair / checkpoint / worker 尝试
```

AreaMatrix 当前目录映射到 AreaFlow 时，多个文件可以共同挂在一个 item 下。例如
`docs-discussion.md`、`middle-layer-discussion.md` 和 `decisions.yaml` 共同表达一个
`discussion_package`，这些文件本身是 artifacts。一个 plan、一个 prompt draft package、
一个 queue candidate 或一次 promotion preview 可以分别成为不同 stage 的 item。

Queue 与 execution 保持一对多关系：

```text
queue_candidate / workflow_item
  -> run #1
  -> run #2
  -> run #3
```

同一个 queue candidate 可以因为失败、重试、repair、重新 verify 或不同 worker 策略产生多个 run。
run 表达一次用户可见执行会话；run_task 表达该 run 内可被 worker 领取的单元；run_attempt 表达一次
copy、verify、repair、checkpoint 或 doctor 尝试。任何重试都新增 attempt，不能覆盖历史 attempt。

长期需要 `workflow_item_links` 表达 trace：

```text
derives_from
implements
verifies
promotes_to
depends_on
blocks
supersedes
closes
```

`workflow_item_links` 应作为早期正式模型，而不是长期藏在 metadata 中。v0.3-v0.4 可以先只写入
最小关系类型，例如 `derives_from`、`implements`、`verifies` 和 `promotes_to`；v0.5-v1.0 再扩展
`depends_on`、`blocks`、`supersedes` 和 `closes`，供 Web trace、projection 和 closeout 使用。

## Workflow 状态机

AreaMatrix profile v0 使用以下 stage：

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

其中 `source_docs` 仍由被管理项目拥有；AreaFlow 只引用、校验和记录 hash。

版本级 lifecycle status 建议保持为可组合的粗粒度位置，不承载全部 gate/runtime 事实：

```text
intake
discussion
planning
queueing
promotion_preview
approved
execution_ready
executing
projecting
closing
closed
archived
superseded
blocked
```

标准 item 状态保持少量枚举：

```text
draft
ready
blocked
deferred
promoted
running
done
superseded
```

更细的原因放入 `metadata`、`events`、`gate_results` 或 `audit_events`，不扩大状态枚举。

### 状态分层

AreaFlow 不使用单一 `status` 字段表达全部语义。`workflow_items.status` 只表示 item 的粗粒度生命周期；
gate、approval、run、attempt、lease 和 projection 各自保留自己的状态。API 和 Web 可以组合出以下视图，
但不能把它们写回成一个万能状态：

```text
phase_state:
  not_started
  active
  ready
  blocked
  skipped
  done
  superseded

gate_result:
  pass
  warn
  fail
  blocked
  needs_approval

runtime_state:
  queued
  leased
  running
  waiting_approval
  succeeded
  failed
  canceled
  expired

lease_state:
  active
  expired
  released
  stolen
```

例如，一个 workflow version 可以处于 `promotion_preview` 阶段，`approval_gate` 为
`needs_approval`，同时某个 read-only doctor run 已 `succeeded`。这些事实必须分别来自
`workflow_items`、`gate_results`、`approval_records`、`runs`、`run_attempts`、`leases` 和
`events`，不能被压平成一个 `done` 或 `failed`。

Gate 是可重复判断，不是状态本身。AreaMatrix profile 的关键 gate 包括：

```text
import_coverage
write_permission
status_mirror
hash_drift
stage_coverage
profile_binding_drift
discussion_gate
plan_doctor
draft_doctor
queue_doctor
promotion_preview
approval_gate
live_mapping_gate
runner_gate
checkpoint_gate
projection_gate
closeout_gate
```

硬规则：

```text
discussion_gate 未通过，不得进入 changes
plan_doctor 未通过，不得进入 drafts
draft_doctor 未通过，不得进入 queue
queue_doctor 未通过，不得进入 promotion_preview
promotion_preview pass 不等于 apply
approval_gate pass 不等于 execution
live_mapping_gate pass 不等于已经 cutover
runner_gate 未通过，不得真实执行
projection_gate 未通过，不得 closeout
closeout_gate 证据不足，不得 done
```

`drafts`、`queue`、`promotion_preview` 和 `approval` 都是显式治理 stage：

```text
drafts:
  workflow_item: draft_manifest / draft_copy / draft_verify
  artifacts: manifest / copy_ready_prompt / verify_ready_prompt
  gate: draft_doctor

queue:
  workflow_item: queue_candidate
  artifacts: queue_yaml / queue_review
  gate: queue_doctor

promotion_preview:
  workflow_item: promotion_preview
  artifacts: promotion_yaml / promotion_md
  gate: promotion_preview

approval:
  workflow_item: approval_scope
  approval_records: approved / rejected decision
  audit_events: approval-created / approval-rejected / approval-overridden
```

关键边界：

```text
promotion_preview pass != approval
approval approved != execution already happened
live_mapping_gate pass != cutover already happened
```

Approval 的核心事实是 `approval_record + audit_event`，不是普通文件或 metadata。
Approval 还必须保存批准时的 gate snapshot、actor、scope、risk level、允许 capabilities、资源范围、
过期策略和拒绝/撤销路径。缺少这些字段时，approval 只能作为草稿或历史参考，不能打开 execution、
project write、secret、network、git 或 worker 控制。

## Authored Version 边界

AreaFlow 创建新 workflow version 时，主状态写入 PostgreSQL，并把 skeleton、prompt、报告等大内容
写入 artifact store。默认不写被管理项目的 `workflow/versions/**` 目录骨架。

```text
areaflow workflow version create
  -> workflow_versions row
  -> workflow_items rows
  -> gate_results / events / audit_events
  -> skeleton artifacts

not by default:
  -> managed-project/workflow/versions/<version>/**
```

AreaFlow-authored version 不要求在被管理项目内存在同名目录。目录可以作为导入源、兼容投影或显式导出结果，
但不是新版本的主状态。允许写入被管理项目时，默认只导出粗略 projection，例如
`.areaflow/status.json` 和 `workflow/README.md`。

## AreaMatrix 迁移策略

AreaMatrix workflow 迁移节奏固定为：

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

策略：

- 先导入索引和 artifact metadata，历史原文短期留在 AreaMatrix。
- 先 mirror，后 authoring cutover；execution cutover 必须等 runner / worker / approval / audit 闭环完成。
- 最终 AreaMatrix 只保留轻量入口和链接。
- AreaMatrix 粗略状态给人看可放 `workflow/README.md`，给工具读放 `.areaflow/status.json`。
- AreaFlow 接管 workflow/task-loop 主能力后，AreaMatrix 仍保留项目文档、源码、验证命令、治理规则、发布证据和用户文件安全边界。
- Shim lifecycle 必须显式区分 `not_installed`、`read_only_shim`、`execution_forwarding` 和
  `retired_thin_entry`；read-only shim 不等于 execution cutover，execution forwarding 不等于 retirement。

v0.4 cutover 的含义只限于：

```text
新 workflow version authoring 源事实切到 AreaFlow
legacy workflow versions 进入 read-only / immutable 口径
AreaMatrix 保留粗略入口和 compatibility shim
```

v0.4 不执行 `./task-loop run`，不写 execution package，不 promotion apply。

Cutover 前必须满足：

```text
import coverage 100%
hash drift check pass
stage coverage pass，或 warn 有明确解释
AreaFlow doctor 与 AreaMatrix doctor 等价
status mirror 稳定
compatibility contract 可查询
write permission allowlist 生效
audit_events 能记录敏感动作
rollback plan 明确
用户显式 approval
approval_gate pass
live_mapping_gate pass
```

Rollback 追加事实，不删除历史：

```text
soft rollback:
  新 workflow version 切回 project-owned 或 frozen
  AreaFlow 保留 events / audit_events / artifacts

hard rollback:
  只在尚未写项目文件、尚未开始执行时允许
  标记 cutover attempt failed
  不删除历史 event / audit / artifact
```

## 多项目与 adapter/profile

所有业务对象都必须显式带 project scope。AreaFlow 不靠文件路径猜项目。

```yaml
project:
  id: areamatrix
  adapter: areamatrix
  workflow_profile: areamatrix
```

后续可扩展：

```yaml
project:
  id: another-product
  adapter: git-repo
  workflow_profile: areaflow-standard
```

Adapter 负责：

```text
load project metadata
scan source references
import historical artifacts
detect drift
export status projection
apply allowed writes
run allowed native commands
map project-specific files to AreaFlow objects
```

Profile 负责：

```text
define stages
define allowed transitions
define required artifacts
define gates
define validation commands
define promotion rules
define rollback routes
define closeout rules
```

Adapter 不定义 workflow 状态机；profile 不读磁盘、不执行命令、不处理 secret。

## Runner 和 Worker

Runner 是执行模型，worker 是实际干活的进程。

```text
workflow_item = 要做什么
run = 一次执行会话
run_task = run 内执行单元
run_attempt = 一次真实尝试
worker = 执行进程
lease = worker 领取任务的租约
artifact = 输入/输出证据
gate_result = 能不能继续
event/audit_event = 发生了什么、谁允许了什么
```

执行链路：

```text
workflow_item ready
-> runner_gate pass
-> run queued
-> run_task queued
-> worker lease acquired for run_task
-> attempt copy
-> attempt verify
-> repair if needed
-> checkpoint if required
-> projection_gate
-> closeout_gate
```

Worker 只拿 scoped lease，不能扩大能力：

```text
project_id
run_id
run_task_id
workflow_item_id
allowed_capabilities
allowed_paths
allowed_commands
secret_refs
expires_at
```

规则：

- 同一个 run_task 同时最多一个 active execution lease。
- workflow_item 只定义语义工作项，worker 不能直接领取 workflow_item。
- lease 过期后进入 `needs_recovery`，不直接判失败。
- worker 完成必须提交 attempt、artifact、event 和 audit_event。
- verify 不通过不能 mark done。
- checkpoint 失败必须阻断下一任务，除非有明确降级 approval。
- drain 跑完当前 task 的 verify/checkpoint 后停止，不跳过验收。

运行环境分层：

```text
local_host      起步支持
local_worker    很快支持
remote_worker   后续支持
container       按需支持，不作为 AreaMatrix 默认路径
```

## 权限、审计和 Secret

AreaFlow 默认只读。任何写入、命令执行、网络访问、密钥使用和 git 操作都必须通过显式授权。

Capability 枚举：

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

`write_artifacts` 只表示写 AreaFlow artifact store 或 artifact metadata evidence，不表示允许写被管理项目。
`write_generated` 只表示写显式 allowlist 的 generated/projection 前缀，不表示允许写 source code、
execution、progress、logs 或 checkpoint。
`manage_workers` 只表示 worker lifecycle、heartbeat、lease 和 recovery 管理，不表示允许真实执行任务。
真实项目写入、命令执行、网络访问、secret 解析和 agent execution 必须使用更具体 capability，并进入
R2-R4 gate / approval / audit。

判断必须同时满足：

```text
capability allowed
path allowed
not forbidden
required gate passed
approval present when needed
expected-before hash matched when writing existing project files
rollback plan present when project write is attempted
audit event written
```

Deny 永远优先于 allow。

### 风险等级

AreaFlow 使用风险等级决定是否需要 preview、approval、额外 audit 和 rollback 说明：

```text
R0 read_only:
  只读查询、metadata import、hash 计算、preview、doctor。

R1 projection:
  写轻量 projection，例如 `.areaflow/status.json`。

R2 managed_write:
  写被管理项目允许路径，例如显式允许的 workflow export。

R3 execution:
  执行任务、修改代码、生成 execution evidence、运行 worker。

R4 migration_security:
  DB migration、secret 解析、权限变更、远程 worker、release exception apply。
```

R0 可以默认运行；R1 需要 project config 允许路径并写 audit；R2-R4 必须有明确 gate 和 approval。
任何阶段不得把 R2-R4 能力伪装成 R0/R1 preview。

即使 v0 是本机单用户，也要创建稳定 actor：

```text
system
local-user
worker
api-token
agent
```

`events` 记录业务发生了什么；`audit_events` 记录谁被允许或拒绝做什么，为什么。

必须审计：

```text
权限判断
写入请求
命令执行请求
secret 引用
approval
cutover attempt
rollback attempt
worker lease
run cancel / drain / pause
API token 使用
```

项目配置只写 secret reference，不写真实密钥：

```yaml
engines:
  profiles:
    - id: openai-main
      provider: openai
      secret_ref: openai/default
```

本机版先使用 env / OS keychain；团队版再引入 encrypted secret store 或外部 secret manager。

## Artifact 策略

Artifact 原文不直接进入 PostgreSQL。PG 只保存：

```text
project_id
workflow_version_id
workflow_item_id
run_id
artifact_type
storage_backend
uri
source_path
sha256
size_bytes
content_type
metadata
```

本地路径建议：

```text
~/.areaflow/artifacts/{project_key}/{workflow_version_or_global}/{category}/{artifact_id}-{sha256-prefix}{extension}
```

`project_key` 是 artifact namespace 边界，不使用数据库内部数字 id。项目配置只声明平台级 root：

```yaml
artifact_store:
  backend: local
  root: ~/.areaflow/artifacts
```

AreaFlow 写入本地 artifact 时自动追加 `{project_key}` 子目录。`source_path` 保存 artifact 在项目
namespace 内的相对路径；`uri` 保存实际 backend URI。远程 / 团队版 API 默认不直接暴露可写本机路径，
需要原文时通过 artifact API 读取。

Artifact 类型按用途分层：

```text
source_ref
import_snapshot
gate_evidence
run_input
run_output
failure_summary
report
diff
log
```

Artifact retention class 从 day 1 建模：

```text
ephemeral:
  临时 preview、缓存或可重建报告，可被显式 GC。

run_evidence:
  run / run_task / run_attempt 的执行证据，不能早于 run closeout 清理。

audit:
  权限、approval、命令执行、写入和 release 证据，默认长期保留。

release:
  发布、分发、验收和 rollback 证据，不能随普通 project cleanup 删除。

external_ref:
  被管理项目内历史原文引用，AreaFlow 只保存 metadata/hash/path，不删除原文件。

legal_hold:
  法务、合规或人工 hold，任何 archive / GC / delete apply 都必须 blocked。
```

Artifact 生命周期：

- `source_reference` 默认只索引被管理项目内的 path、hash、size 和 metadata，不复制历史原文。
- `managed_copy` 表示 AreaFlow 拥有的新 workflow skeleton、prompt、approval bundle 等原文。
- `generated_output` 表示 runner / worker / doctor 生成的 report、log、diff、evidence。
- artifact 一旦登记到 PG，不覆盖原文；修正使用新的 artifact 和追加事件表达。
- 删除或 archive project 默认不删除 artifact 原文；清理必须是显式命令并写 audit。
- 备份恢复必须覆盖 PostgreSQL dump、artifact inventory、project config、secret ref metadata 和
  artifact integrity 结果。

Artifact doctor 必须能区分以下状态：

```text
metadata_only:
  只有 AreaFlow metadata，例如历史 AreaMatrix project_reference。

local_verified:
  AreaFlow-owned local artifact 存在，sha256 / size 与 PG metadata 一致。

missing_blob:
  PG 有 local artifact metadata，但 artifact store 缺原文。

hash_mismatch:
  原文存在但 sha256 / size 与 PG metadata 不一致。

orphan_blob:
  artifact store 有文件，但 PG 没有 metadata。

external_ref_skipped:
  历史 project_reference 原文不由 AreaFlow 校验或删除，只返回 warn/skipped。

object_verifier_skipped:
  object backend 尚未通过 verifier，只能返回 skipped / needs_attention，不能计入完整可恢复内容。
```

GC 在 v1.0 只做 preview。未来 GC/delete apply 第一版只能处理 AreaFlow-owned `ephemeral` artifacts；
`audit`、`release`、受保护的 `run_evidence`、`external_ref`、`legal_hold`、未知 retention class、
hash mismatch local artifact 和 verifier skipped/failed object artifact 不能被普通 GC 删除。

## Projection 策略

Projection 是外部项目和界面看到的轻量快照，不是主状态源。真实状态仍来自 PostgreSQL 主表、events、
gate results、approval records、runs、leases 和 artifact metadata。

AreaMatrix 的 projection 分两类：

```text
workflow/README.md:
  人读粗略入口，展示当前由 AreaFlow 管理、主要阶段和跳转链接。

.areaflow/status.json:
  工具读粗略 projection，展示 project、cutover_phase、active_versions、rough_progress、AreaFlow 链接、
  last_synced_at 和 source_snapshot_hash。
```

`status_projections` 作为长期模型应记录：

```text
projection_id
project_id
workflow_version_id
target_kind
target_uri
summary_state
payload_json
source_event_id
source_hash
generated_at
write_state
written_at
```

写 projection 也是项目写入。即使只写 `.areaflow/status.json`，也必须满足 `write_status` capability、
path allowlist、projection gate 和 audit event。`workflow/README.md` 的自动更新应等 cutover
gate 之后再打开；cutover 前可生成 preview 或人工维护。

`.areaflow/status.json` 文件本身不能直接暴露完整 import `summary`、queue、attempt、lease、
approval payload、artifact 原文或 release evidence。DB 中的 `status_projections.payload_json` 可以保留
内部粗略 payload 和 source hash；对外文件必须收束为稳定的 fallback projection schema。

## 多端边界

CLI、Web、Desktop、Worker 最终都复用同一套 API 和事件语义：

```text
CLI / Web / Desktop / Worker
  -> AreaFlow API
    -> PostgreSQL
    -> Artifact Store
```

Bootstrap 命令可以直连 PG，例如 `areaflow migrate`。业务能力不应各端各自读取项目文件并维护第二套状态。

Web v0.7 先做 dashboard、timeline、artifact、blocker、approval records、worker status 和 audit trail。
阶段合同见 [`../architecture/v0.7-web-dashboard-contract.md`](../architecture/v0.7-web-dashboard-contract.md)；
v0.8 worker pool preview 边界见
[`../architecture/v0.8-multi-project-worker-pool-contract.md`](../architecture/v0.8-multi-project-worker-pool-contract.md)。
v0.9 Desktop shell 边界见
[`../architecture/v0.9-desktop-shell-contract.md`](../architecture/v0.9-desktop-shell-contract.md)。
v1.0 Stable Platform 关闭边界见
[`../architecture/v1.0-stable-platform-contract.md`](../architecture/v1.0-stable-platform-contract.md)。
Web 初始请求必须保持 `/api/v1` GET / SSE-only，write action gate 只展示 disabled/read-only 动作。
Approval、drain、cancel 等写操作只有在 Command API、risk preview、permission preflight 和 audit
outcome 稳定后才逐步打开。Desktop v0.9 只做 local service status、dashboard launcher、项目切换、
desktop gates 和本机控制台，不维护第二套状态，也不承担团队远程控制台职责。

v0.7 Web Dashboard 是统一控制台，不是新的状态源。第一版页面围绕：

```text
projects
project summary
version stage timeline
workflow items
gate results
artifacts
runs / run tasks / attempts
workers / leases
approvals
audit
```

第一版 Web 不直接读取 local artifact path，不创建 approval，不执行 drain/cancel，不调度 worker，不编辑
AreaMatrix shim，也不把 SSE event 或 scoped v0.6 evidence 当作主状态。

v0.8 Multi-project / Multi-worker 的核心是 project scope、worker capability、schedule preview 和
engine readiness。
所有任务必须显式带 `project_id`、`workflow_version_id`、`workflow_item_id` 和 `run_id`；
worker 只能领取符合 capability、project scope 和 lease policy 的任务。

v0.9 Desktop Shell 只负责 local service status、项目切换、desktop gates、本机 worker 状态和 Web 入口。
Desktop 不解析 workflow、不维护 progress、不执行 worker、不打开真实 process control，也不保存第二套状态。

v1.0 的重点是稳定化：install / migrate / start / register smoke、health / readiness / doctor、
service status、metadata-only support bundle preview、backup / restore、audit export、permission policy
doctor、adapter/profile conformance test、worker recovery test、artifact integrity check、API compatibility
policy、migration rollback policy 和 release checklist。

v1.0 只要求 restore dry-run plan 和 release hardening 链路可证明；真实 restore apply 放到 v1.x
后续阶段，必须另行经过 R4 migration_security 设计、显式 approval、验证和回滚计划。Release exception
真实写入也不在 preview/gate 阶段默认打开：先完成 schema preview、migration approval gate 和 apply
preview；只有 release exception migration approval gate 通过后，才允许新增真实 migration / write path。

## 后段硬边界

v0.7-v1.x 会从只读观察进入多端控制、worker 执行、secret、plugin、restore 和 release。后段能力必须按
以下硬边界推进，不能为了演示效果提前打开高风险写入。

| 议题 | 决策 | 首次进入阶段 | 禁止提前打开 | 验收证据 |
|---|---|---|---|---|
| Command API | 所有写动作统一进入 command request，不允许 CLI / Web / Desktop / Worker 各自写状态 | v0.5-v0.7 先按语义收敛，v1.0 前稳定 | 绕过 API 直接改 DB、项目文件或 worker 状态 | `command_class`、`idempotency_key`、`request_hash`、`expected_version`、precondition snapshot、risk preview、permission preflight、safety facts、audit outcome 可查询 |
| Secret Store | v1.0 前只声明 `secret_ref` 和 readiness，不解析明文 | v0.8 readiness，v1.x 真实解析 | 提前读取 env、keychain、DB secret 去跑 engine | readiness 明确 `secret_ref_unavailable` / `secret_ready`，不泄露明文 |
| Plugin | v1.0 稳定 adapter/profile contract 和 seed catalog，第三方 plugin 执行放到 v1.x | v1.0 contract | 运行未知插件代码或让 plugin 绕过 permission | conformance check 覆盖 adapter、profile、manifest、schema、docs 和权限声明 |
| Desktop | Desktop 只是 local service shell 和 dashboard launcher | v0.9 | 维护第二数据库、直接执行 workflow、直接写项目、打开真实 process control、OS notification 或 native tray/menu | service status、desktop gates、dashboard URL、capabilities 和 forbidden actions 可验证 |
| Artifact Archive | 历史 artifact 不自动搬；新 artifact 由 AreaFlow artifact store 管 | v1.0 archive decision | cutover 时隐式复制、删除或覆盖历史 artifact 原文 | restore-plan / artifact-integrity 明确 metadata-only、project_reference 或 copied 状态 |
| Object Artifact / GC | object backend 先 metadata/verifier preview，archive copy/upload 和 GC/delete 属于 v1.x | v1.x rung 16 | 把 object metadata 当作完整备份，或删除 audit/release/run evidence/external_ref/legal_hold | object verifier、retention policy、restore impact、pre-delete manifest 和 audit evidence |
| Execution Cutover | workflow authoring cutover 与 execution cutover 分离 | v0.4 authoring，v0.6 execution beta | v0.4 自动替代 `./task-loop run` | run/task/attempt/evidence/audit 闭环后，只执行 approved task |
| Shim Retirement | 只退役旧 task-loop 主执行能力，不删除历史 workflow/progress/log/evidence | execution forwarding 稳定后 | 删除历史 evidence、双写旧 runner 和 AreaFlow runner | retirement notice、rollback to read-only shim、historical archive/reference policy |
| Team / Remote | schema/API 从 day 1 team-ready，v1.0 前仍 local single-user | v1.x 打开远程团队能力 | 把 single-user 假设写死进 actor、project、worker 或 audit | actors、teams、memberships、project_memberships、api_tokens 边界存在但能力关闭；Team Console 按 `team-remote-control-boundary.md` 逐级打开 |
| Release / Publish | v1.0 只做 readiness、evidence 和 preview gate，不自动发布 | v1.0 preview chain | 自动 tag、push、upload、sign、publish | readiness、acceptance gate、package / distribution / publish / rollout preview 全只读 |
| Operations / Deployment | v1.0 只做本机 bootstrap、service status、doctor、support bundle preview 和 local-only telemetry | v1.0 local hardening，v1.x 打开 remote / managed ops | 自动升级、远程控制、破坏性 rollback、完整 support export、默认远程 telemetry | install/migrate/start/register smoke、metadata-only support bundle preview、redaction proof、migration ledger、AreaMatrix protected path proof |

### v1.x 高风险能力打开顺序

v1.x 不是把所有 R3/R4 能力一次性打开，而是沿着最小可回滚面逐步扩大。每一步都必须通过
Command API、capability、allowlist、approval、rollback 和 audit，且前一步的 smoke evidence
不能替代后一步的门禁。
状态词、apply packet、suspension rule 和 AreaMatrix first policy 以
[`../architecture/high-risk-apply-ladder.md`](../architecture/high-risk-apply-ladder.md) 为准。

| 顺序 | 能力 | 第一版允许形态 | 必须证明 | 继续禁止 |
|---:|---|---|---|---|
| 1 | Generated-only rollback beta | 真实 managed project 内单个已存在 generated/projection 文件写入后立即恢复 preimage | expected-before、preimage、verify、rollback、非目标文件指纹不变、`write_generated` 审计 | 保留写入结果、source code、execution、progress、logs、checkpoint 写入 |
| 2 | Generated-only retained apply | rollback beta 稳定后，真实 managed project 内单个 generated/projection 文件写入并保留结果 | expected-before、preimage、rollback verify、focused smoke、非目标文件指纹不变 | source code、execution、progress、logs、checkpoint 写入 |
| 3 | Manual patch artifact | 只生成 source patch/diff artifact、write-set preview、验证命令和 rollback/remediation plan | expected-before、affected paths、verification plan、rollback plan、artifact hash | AreaFlow 写项目源码、运行 shell、checkpoint、repair |
| 4 | Human-applied source evidence | 人工或现有 Codex 流程 apply 后，AreaFlow 只读取 diff、changed hash 和验证结果 | changed-file hash、validation evidence、copy/verify/checkpoint 语义映射 | AreaFlow 直接写源码、自动 repair、git checkpoint |
| 5 | Source write beta | allowlist 内小范围源码 `create` / `modify` | `write_code`、write-set、copy/verify、checkpoint preview、rollback、focused smoke | delete、move、chmod、binary、symlink、glob、root 外路径 |
| 6 | Checkpoint apply | source write beta 稳定后单独打开 checkpoint | dirty state、scope drift、checkpoint evidence、rollback/remediation、失败阻断下一 task | verify 未通过时 checkpoint、checkpoint preview 自动变 apply、自动 git mutation |
| 7 | Repair plan / apply | 先生成 failure summary 和 repair plan，再用新 attempt apply | failure summary、repair write-set、verify、checkpoint gate、audit | 跳过 verify、覆盖原 attempt、无 approval repair |
| 8 | No-secret engine execution | `secret_ref=none` 的 Codex CLI / local / manual engine | engine attempt、budget、redaction、no-secret/no-network safety facts、audit | secret 注入、长期 token、未审计网络 |
| 9 | Secret resolve | scoped secret binding 给单次 execution context | secret store / keychain / env policy、短期凭证、脱敏、rotation / revoke path | 明文写入 project config、artifact、event、audit |
| 10 | Remote worker | API-only remote worker with scoped token | worker identity、project/capability/lease scope、heartbeat、token revoke、audit | 直连 PostgreSQL、跨项目领取 task、长期万能 token |
| 11 | Restore apply | restore package -> isolated dry-run -> approved apply | package hash、temp DB/project validation、diff、preimage、rollback、R4 approval | 无 preimage 覆盖、静默删除、跳过 dry-run |
| 12 | Release exception write | exception schema + migration approval + apply preview 后写入 | schema preview、migration approval gate、apply preview、R4 approval、audit | preview 阶段创建 migration 或 record |
| 13 | Publish apply | tag / sign / upload / push / publish 拆分命令 | evidence bundle、package hash、approval、remediation、rollout plan | 一键不可回滚发布、跳过签名/包校验 |
| 14 | Third-party plugin execution | signed governed plugin 执行 | manifest、capabilities、signature、sandbox、conformance、disable/revoke、audit | 未知插件绕过 permission 或直接读写项目 |
| 15 | External integrations / webhooks | catalog/readiness -> delivery plan preview -> fixture outbound/inbound -> project delivery -> inbound callback beta -> external API connector | provider allowlist、endpoint allowlist、secret scope、delivery/callback audit、redaction、disable/revoke | callback 直接改状态、未知 endpoint、external API 绕过 Command API |
| 16 | Team console / remote control | read-only team preview -> local auth console -> team permission enforcement -> remote read-only -> remote command console | role matrix、project scope、token/session revoke、audit、Command API preflight、T4/T5 transport security | role 自动获得 project write / secret / publish / restore，Desktop 变成第二状态源 |
| 17 | Object artifact store | object verifier -> scoped upload -> restore integration -> retention / archive / GC | namespace、hash/size、retention、restore dry-run、pre-delete manifest、audit | 把 object metadata 当作完整备份，GC/delete 越级打开 |
| 18 | Budget / quota enforcement | metadata/readiness -> estimate preview -> quota doctor -> fixture reservation/charge -> project enforcement -> aggregation -> provider reconciliation | cost model、quota policy、rate limit、reservation/charge idempotency、override approval、audit | silent throttling、重复扣费、无 project scope 阻断、无过期 override |
| 19 | Managed ops / upgrade / support export | local readiness -> support bundle preview -> remote read-only ops -> remote control -> managed upgrade/rollback -> support export apply | auth/team scope、redaction、destination allowlist、backup/preimage、approval、audit、retention/revoke | 自动升级、无 preimage rollback、默认遥测、support bundle 携带 prompt/secret/user files/raw artifact |

共同开闸条件：

```text
explicit command type
actor / reason
idempotency_key / request_hash
risk_level and risk_policy
affected project / resources
expected_version or expected-before hash
capability and path / command / network / secret preflight
approval record
rollback or remediation plan
audit event
focused smoke and regression evidence
```

## 完成定义

AreaFlow 达到 100% 时，应能证明：

- 从空环境可安装、迁移、启动、注册项目。
- 本机 health、readiness、doctor、service status 和 support bundle preview 可复验，且 telemetry 默认 local-only。
- AreaMatrix 完成 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover
  -> Archive -> Shim Retirement 的可验证闭环。
- 新 workflow version 源事实由 AreaFlow 管理。
- Approved task 可以由 worker 执行并留下 copy/verify/repair/checkpoint 证据。
- Copy、verify、repair、checkpoint 和 rollback 的状态推进矩阵可解释；局部 scoped proof 不能累计成真实
  AreaMatrix execution cutover。
- 多项目、多 worker、Web、Desktop 均复用同一 API 和事件模型。
- PostgreSQL metadata、AreaFlow-owned artifact metadata 和 local artifact store 可生成 backup manifest、
  artifact integrity report 和 restore dry-run plan；`project_reference` 历史原文的恢复限制被明确标注。
- Object backend 在 verifier 落地前只能作为 skipped / needs_attention；archive copy/upload、GC/delete
  和 orphan cleanup 不属于 v1.0 完成定义。
- audit 覆盖写入、命令执行、approval、worker lease、permission change、secret reference 使用和
  release exception 决策；真实 secret resolve 属于后续 R4 能力时必须另行审批。
- adapter/profile/plugin 边界稳定并有公开文档。
- AreaMatrix compatibility commands 有明确迁移和退役说明。
- completion audit 按当前证据逐项返回 complete；release final gate pass、绿色测试或 smoke 不能单独替代
  completion audit。

## 已锁定地基决策

1. AreaFlow 是独立平台仓库，不是 AreaMatrix 子模块；AreaMatrix 是第一个 dogfooding project。
2. 技术栈采用 Go + PostgreSQL + REST/JSON + SSE；后续 Web 为 React + TypeScript，Desktop 为 Tauri。
3. PostgreSQL 是主状态源事实，不提供 SQLite 主状态 fallback；文件只用于配置、artifact 原文、status projection 和审计导出。
4. Core 使用 profile-driven workflow stage engine；AreaMatrix profile 声明完整 stage 链路，但 core 不硬编码 AreaMatrix。
5. `workflow_item` 表示语义交付物，artifact 表示原文/报告/日志/evidence，`run_task` 表示执行单元，`run_attempt` 表示一次真实尝试。
6. `drafts`、`queue`、`promotion_preview` 和 `approval` 是独立治理 stage；promotion preview 不等于 approval，approval 不等于 execution。
7. AreaFlow 默认只读管理项目；任何写入必须同时满足 capability、path allowlist、deny 优先、gate 和 audit。
8. AreaMatrix 迁移采用 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover
   -> Archive -> Shim Retirement；v0.4 cutover 只切新 workflow version authoring 源事实，不等于
   task-loop execution cutover。
9. Artifact store 是平台级本地仓库，默认 root 为 `~/.areaflow/artifacts`，按 `{project_key}` 隔离；历史引用用 `project_reference`，新内容用 `local`，后续可扩展 `object`，但 object verifier 未通过前不能计入完整可恢复内容。
10. AreaFlow 从 schema day 1 支持 self project，但 v0.1-v0.2 不依赖自管理；v0.3 后只读 dogfood，v0.5 后 dry-run / 低风险 worker dogfood，v1.0 才稳定自托管。
11. v0.1-v0.4 是 import、shadow doctor、new version authoring、workflow ownership cutover；不真实执行、不自动写项目源码、不替代 task-loop。
12. v0.5-v0.6 建 runner / worker beta：PG 为执行源事实，copy / verify / repair 分离，lease 可恢复，capability preflight 拒绝路径必须无 lease / attempt / artifact。
13. Web v0.7 是 dashboard + approval console，所有写操作走 Command API；SSE 只做刷新，不是状态源事实。
14. v0.8 打开多项目 worker pool；所有 run、task、lease、artifact、secret、audit 都必须 project scoped。
15. v0.9 Desktop Shell 只负责 local service status、desktop gates、secret readiness、通知路径预览、项目切换和
    Web 入口，不维护第二套状态。
16. v1.0 完成门槛是可安装、可恢复、可审计、可多项目运行，并完成 AreaMatrix dogfood cutover 闭环。
17. 状态必须分层表达：phase、gate、approval、runtime、lease 和 projection 不得被压成一个万能状态。
18. 风险等级 R0-R4 从 preview 到 migration/security 分级管理，R2-R4 必须经过 gate、approval 和 audit。
19. Projection 只做外部快照；`.areaflow/status.json` 和 `workflow/README.md` 不能成为主状态源。
20. Command API 标准请求必须包含 actor、reason、command class、idempotency_key、request_hash、
    risk_level、risk_policy、affected resources、safety facts 和可选 expected_version；同一幂等键携带不同
    request hash 必须拒绝。
21. Restore apply 属于 v1.x 高风险能力，v1.0 只交付 backup manifest、artifact integrity 和 restore
    dry-run plan。
22. Object artifact store、archive copy/upload、retention-aware GC、orphan cleanup 和 delete apply 属于
    v1.x 高风险能力；普通 GC 不得触碰 `audit`、`release`、受保护的 `run_evidence`、`external_ref`、
    `legal_hold`、未知 retention class、hash mismatch local artifact 或 verifier skipped/failed object。
23. Release exception 真实写入必须等 schema preview、migration approval gate、apply preview 和 R4
    approval 全部通过；只读 preview/gate 不能创建 migration 或写 exception record。
24. v1.0 前 secret 只做引用和 readiness，不解析明文、不调用需要 secret 的真实 engine。
25. Plugin marketplace 在 v1.0 只允许 template / seed / manifest draft / conformance，不运行未知第三方代码。
26. External integrations / webhooks 在 v1.0 只允许 catalog、readiness、delivery plan preview 和 blocked
    reason，不投递 webhook、不处理 callback 为业务事实、不调用外部 API。
27. Budget / quota 在 v1.0 只允许 estimate、readiness 和 blocked reason，不扣减 quota、不写 charge、
    不同步 provider billing、不 silent throttle。
28. v1.0 release 链路只允许 readiness、evidence 和 preview gate；tag、push、upload、sign、publish
    都属于后续显式 command 能力。
29. v1.0 operations 链路只允许本机 install/migrate/start/status/doctor、metadata-only support bundle
    preview 和 local-only telemetry；远程运维控制、托管升级、破坏性 rollback、完整 support export 属于 v1.x。
30. v1.0 completion audit 是最终完成证明入口；release final gate、package preview、Web/Desktop 展示或
    smoke 通过都不能单独声明 100%。
31. `workflow_version` 是生命周期根对象，`workflow_item` 是阶段语义交付物，artifact 是原文/证据，
    `run_task` 才是 worker 可领取的执行单元。
32. 一个 queue candidate / workflow item 可以产生多个 run；一次 run 内可以有多个 run_task 和
    run_attempt，重试、repair、verify 和 checkpoint 都必须追加事实。
33. Approval 必须绑定 gate snapshot、actor、scope、risk、capability、resource、expiry 和 audit event；
    不能被压缩成普通状态字段。
34. Admin API 只用于 migrate、service、doctor、import/export 等受限运维入口；它不能绕过 Command API、
    permission、approval 或 audit 去改变 workflow 业务状态。
35. AreaMatrix 最终保留 `areaflow.yaml`、`workflow/README.md`、`.areaflow/status.json` 和极薄兼容命令；
    旧 workflow 主副本、progress、logs、checkpoint 和 task-loop 主执行逻辑进入归档或退役，但历史文件和
    release evidence 不因 retirement 被删除。
