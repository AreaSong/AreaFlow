# Workflow Engine Contract

## 定位

Workflow Engine 是 AreaFlow 的核心编排层。它不等同于 AreaMatrix
当前的目录结构，也不等同于一个简单 task runner。

它负责把需求、源事实、讨论、计划、草稿、候选队列、promotion、执行、
投影和收口统一成可查询、可审计、可迁移的状态机。

## 核心原则

- Core engine 只理解 stage、gate、artifact、transition、trace、permission 和 event。
- 项目特有流程通过 workflow profile 表达，不硬编码进 core。
- PostgreSQL 保存当前状态、索引、关系和事件；artifact 原文保存在 artifact store。
- 被管理项目的 `docs/**` 和源码仍由项目自己拥有。
- 默认只读；任何写入、执行和密钥使用都必须通过 capability 与 path allowlist。
- promotion preview 不是 live apply；approval 不是 execution；execution pass 不是 closeout。
- 状态必须来自导入、doctor、gate、runner 或审计事件，不能为了展示手写绿色状态。
- Profile 文件必须可校验并可 hash。创建 authored workflow version 时冻结 profile version/hash，
  后续 profile 升级只能显式迁移。冻结结果写入 `workflow_versions.status_summary.profile_binding`，
  供 API、CLI、Web 和后续 migration/upgrade doctor 查询。

## Core Objects

### Project

Project 表示一个被 AreaFlow 管理的项目。

关键字段：

```text
project_key
name
adapter
workflow_profile
root_uri
default_branch
status
```

### Workflow Version

Workflow Version 表示某个项目的一轮版本化 workflow。

关键字段：

```text
project_id
display_label
version_kind
lifecycle_status
source_path
source_hash
import_mode
immutable
status_summary
```

`display_label` 保留项目自己的语义，例如 `v1-mvp`。AreaFlow 不强迫项目把版本名改成通用编号。

### Workflow Item

Workflow Item 是 stage 内的结构化工作单元。它可以对应 discussion、change、plan、
draft、queue candidate、promotion preview、approval、projection 或 closeout。

关键字段：

```text
project_id
workflow_version_id
stage
item_type
external_key
title
status
source_path
source_hash
metadata
immutable
```

Workflow Item 的粒度是语义交付物。AreaMatrix profile 当前采用的稳定 item type 为：

```text
source_reference
profile_template
discussion_package
middle_layer_ledger
change_ledger
plan
draft_manifest
draft_copy
draft_verify
queue_candidate
promotion_preview
approval_record
live_mapping
execution_package
projection
closeout
```

它不是单个文件。文件、prompt、报告、diff、日志和 evidence 应作为 artifact 挂到 item、run
或 attempt 上。它也不是执行任务；进入执行模型后才创建 `run_task` 和 `run_attempt`。
`prompt_package` 可以作为 API / UI 聚合视图，但在可审计模型里必须拆成 `draft_manifest`、
`draft_copy` 和 `draft_verify`。

### Artifact

Artifact 表示 AreaFlow 能索引、追踪或展示的大内容。原文不直接进入数据库。

关键字段：

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

### Gate Result

Gate Result 表示某个门禁的一次判断结果。v0.1 可以先放在 `events.metadata`，
v0.2 起应提升为可查询模型。

建议字段：

```text
gate_name
scope_type
scope_id
status: pass | warn | fail | blocked | needs_approval | error
checked_at
inputs
failures
warnings
evidence_artifact_ids
metadata
```

`needs_approval` 表示技术前置条件可解释，但继续推进必须等待显式 approval。它不是 `pass`，也不是
`fail`；preview、approval 和 execution 必须继续分开记录。

### Run

Run 表示一次可审计的执行尝试。v0.1 只索引历史 run metadata；v0.5 以后才成为
AreaFlow 自己的 runner model。

建议字段：

```text
project_id
workflow_version_id
run_kind
status
started_at
finished_at
worker_id
metadata
```

### Event And Audit Event

Event 记录业务状态变化。Audit Event 记录权限、安全、写入、命令执行和密钥引用。

两者都按 append-only 设计：历史事实不回写、不重排、不补造。

## Stage Contract

每个 stage 必须声明以下内容：

```text
name
purpose
required_inputs
required_artifacts
allowed_outputs
allowed_transitions
gate_checks
write_permissions
validation_commands
failure_routes
rollback_policy
```

Stage 不直接表示文件夹。一个 stage 可以来自文件、数据库记录、外部系统或 runner 输出。

## Standard Item States

Workflow item 状态保持少量稳定枚举：

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

状态含义：

| Status | Meaning |
|---|---|
| `draft` | 已创建或导入，但尚未通过本层 gate。 |
| `ready` | 已通过本层 gate，可进入下一 stage。 |
| `blocked` | 存在未解决 blocker，不能继续推进。 |
| `deferred` | 明确延期，不阻塞当前目标。 |
| `promoted` | 已从 planning side 进入 live execution side。 |
| `running` | 正在执行或等待 worker 完成。 |
| `done` | 执行、验收或本层收口已完成。 |
| `superseded` | 被后续版本、变更或决策取代。 |

更细的原因放入 `metadata`、`events` 或 `gate_results`，不继续扩大状态枚举。

## AreaMatrix Profile v0

AreaMatrix 是第一个内置 workflow profile。它用于 dogfooding，不代表所有项目都必须采用同样流程。

源事实文件：

```text
workflow/profiles/areamatrix/profile.yaml
```

只读检查：

```bash
areaflow workflow profile check areamatrix
```

| Order | Stage | AreaMatrix Source | AreaFlow Object |
|---:|---|---|---|
| 1 | `intake` | `workflow/intake.md` 和用户讨论 | workflow item + artifact |
| 2 | `source_docs` | `docs/**` references, baseline docs | artifact refs + source hashes |
| 3 | `templates` | `workflow/templates/**` | profile template artifacts |
| 4 | `version_init` | `workflow/versions/v*/version.yaml` | workflow version |
| 5 | `discussion` | `discussion/docs-discussion.md`, `middle-layer-discussion.md`, `decisions.yaml` | workflow items + gate result |
| 6 | `middle_layer` | `middle-layer/*.yaml` | workflow items |
| 7 | `changes` | `changes/*.yaml` | workflow items |
| 8 | `plans` | `plans/*.md` | workflow items + artifacts |
| 9 | `drafts` | `drafts/**` copy / verify / manifest | workflow items + artifacts |
| 10 | `queue` | `queue/**` | workflow items |
| 11 | `promotion_preview` | `promotion/promotion.*` | workflow item + gate result |
| 12 | `approval` | `promotion/approval.yaml` | approval item + audit event |
| 13 | `execution` | `execution/**` | live execution artifacts |
| 14 | `run` | task-loop progress, logs, checkpoints | runs + events + artifacts |
| 15 | `projection` | `projection/**` | projection items |
| 16 | `closeout` | `closeout/**` | closeout item + gate result |

### AreaMatrix Object Mapping Baseline

AreaMatrix profile 的 stage 是固定的迁移语义，AreaFlow core 的表结构不是固定目录镜像。每个 stage
进入 AreaFlow 时必须落成可查询对象、artifact 引用和 gate 证据：

| Stage | Primary Item Type | Artifact Role | Required Gate | First CLI/API Surface |
|---|---|---|---|---|
| `intake` | `discussion_package` | user request, intake note | `import_coverage` | `workflow inspect` / project summary |
| `source_docs` | `source_reference` | docs hash refs | `hash_drift` | import / doctor |
| `templates` | `profile_template` | template refs, schema refs | `stage_coverage` | profile check |
| `version_init` | workflow version root | `version.yaml` source ref | `profile_binding_drift` + `write_permission` | workflow version create/import |
| `discussion` | `discussion_package` | docs discussion, middle-layer discussion, decisions | `discussion_gate` | workflow discuss doctor |
| `middle_layer` | `middle_layer_ledger` | feature mapping ledger | `stage_coverage` | workflow plan preview |
| `changes` | `change_ledger` | docs-change ledger | `plan_doctor` | workflow changes doctor |
| `plans` | `plan` | plan docs, dependency/risk evidence | `plan_doctor` | workflow plan doctor |
| `drafts` | `draft_manifest`, `draft_copy`, `draft_verify` | manifest, copy-ready, verify-ready prompt artifacts | `draft_doctor` | workflow draft doctor |
| `queue` | `queue_candidate` | candidate manifest, local numbering | `queue_doctor` | workflow queue preview |
| `promotion_preview` | `promotion_preview` | dry-run live mapping report | `promotion_preview` | workflow promote preview |
| `approval` | `approval_record`, `live_mapping` | approval packet, risk scope, rollback reference | `approval_gate` + `live_mapping_gate` | command approval / transition approve |
| `execution` | `execution_package` | approved live execution package refs | `runner_gate` | run preview / execution gate |
| `run` | run model, not workflow item | attempt outputs, logs, evidence | `checkpoint_gate` | run / worker APIs |
| `projection` | `projection` | status projection, timeline summary | `projection_gate` | status projection sync |
| `closeout` | `closeout` | closeout report, residual links, archive markers | `closeout_gate` | workflow closeout |

`source_reference` 和 `profile_template` 可以先作为 imported item type 使用；如果后续发现它们只是 artifact
分类而不是用户可见 workflow item，可以通过 migration 收敛到 artifact metadata。当前保留它们，是为了
让 import coverage、profile conformance 和 stage coverage 能按 stage 给出明确证据。

`run` stage 不创建 `workflow_item` 来伪装一次执行。它以 `runs`、`run_tasks`、`run_attempts`、`leases`
和 evidence artifacts 为主。`execution_package` 只描述“批准后可被 runner 消费的包”，不描述执行结果。

### Item Granularity Rules

workflow item 的粒度按“人和系统需要追踪的语义交付物”确定：

- 一个 discussion stage 通常是一个 `discussion_package`，其 artifacts 包含 docs discussion、
  middle-layer discussion 和 decisions ledger。
- 一个 feature 或 change lane 可以产生一个 `middle_layer_ledger`，再派生一个或多个 `change_ledger`
  和 `plan`。
- drafts stage 必须保持 `draft_manifest`、`draft_copy` 和 `draft_verify` 分离；不能合并成一个不可审计
  prompt。
- 一个 `queue_candidate` 表达“准备执行什么”，不能表达“已经执行过什么”。
- 一个 `approval_record` 只批准明确 transition 或 command scope；`live_mapping` 只证明 candidate 到 live
  execution scope 的映射，二者都不能批准整个 project、整个版本或未来未知写入。
- 一个 `closeout` 必须链接实际 run、projection、residual、acceptance evidence 和未关闭风险；不能只保存
  人工总结。

推荐的最小 trace 链为：

```text
discussion_package
  -> middle_layer_ledger
  -> change_ledger
  -> plan
  -> draft_manifest
  -> draft_copy / draft_verify
  -> queue_candidate
  -> promotion_preview
  -> approval_record / live_mapping
  -> execution_package
  -> run_task / run_attempt
  -> projection
  -> closeout
```

关系写入 `workflow_item_links`。缺少上游 link 时，下游 item 可以被导入为 `draft` 或 `blocked`，但不得
自动标记为 `ready`。

`drafts`、`queue`、`promotion_preview` 和 `approval` 必须保持独立 stage，因为它们分别承载
草稿完整性、候选队列、live mapping 预演和显式审批。Gate result 不能替代这些 stage：

```text
draft_doctor 证明 draft item 是否完整
queue_doctor 证明 queue item 是否可进入 promotion preview
promotion_preview 证明映射预演是否安全
approval_gate 证明审批事实是否满足进入下一阶段
```

Approval 的主要事实是 `approval_record` 和 `audit_event`。`approval_scope` item 只提供 workflow
timeline 上的挂载点。

### AreaMatrix v0.1 Import Rules

v0.1 只读取并索引：

- workflow version metadata。
- residual metadata。
- artifact path、hash、size 和 type。
- v1 execution progress summary。
- active task metadata。

v0.1 不读取或接管：

- task-loop runtime control。
- copy / verify / repair execution。
- progress 写入。
- checkpoint、logs 或 release evidence 的原文迁移。

## Gate Contract

Gate 是可重复执行的判断，不是一次性人工说明。

每个 gate 必须回答：

```text
what is checked
which inputs were used
which source hashes were used
whether it passed
why it failed
where to return on failure
which artifact proves the result
```

### Required Gates For AreaMatrix Profile

| Gate | Purpose | Earliest Phase |
|---|---|---|
| `import_coverage` | 导入覆盖率与索引完整性 | v0.1 |
| `write_permission` | 写入 capability 和 path allowlist 判断 | v0.1 |
| `status_mirror` | 粗略状态导出是否稳定 | v0.1 |
| `hash_drift` | 被管理项目源文件是否与导入快照漂移 | v0.2 |
| `stage_coverage` | 每个 stage 是否有足够 artifact 和状态 | v0.2 |
| `profile_binding_drift` | authored workflow version 冻结的 profile hash 是否仍匹配当前 profile registry | v0.4 |
| `discussion_gate` | Exact Docs、open questions、risk boundaries 是否满足 | v0.3 |
| `plan_doctor` | plan 依赖、验收和 trace 是否完整 | v0.3 |
| `draft_doctor` | copy-ready / verify-ready / manifest 是否可验收 | v0.3 |
| `queue_doctor` | 候选队列 label、依赖和 readiness 是否一致 | v0.3 |
| `promotion_preview` | dry-run 映射、撞名和 scope 检查 | v0.3 |
| `approval_gate` | 显式 approval 和风险确认 | v0.4 |
| `live_mapping_gate` | candidate 到 live execution 的映射可证明 | v0.4 |
| `runner_gate` | execution package 可被 runner 消费 | v0.5 |
| `checkpoint_gate` | scope drift、dirty state、commit evidence 判断 | v0.6 |
| `projection_gate` | run 结果能投影回 change/plan/draft/queue | v0.5 |
| `closeout_gate` | execution、projection、evidence 和风险一致 | v0.5 |

## Transition Rules

标准推进路径：

```text
intake
-> source_docs
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

强制规则：

- `discussion_gate` 未通过不得进入 `changes`。
- `draft_doctor` 未通过不得进入 `queue`。
- `queue_doctor` 未通过不得进入 `promotion_preview`。
- `promotion_preview` 通过不等于允许写 execution。
- `approval_gate` 与 `live_mapping_gate` 都通过后才允许进入后续 explicit promote 设计；v0.4a
  中二者仍然只是 read-only gate result，不写 execution。
- `runner_gate` 通过后才允许真实 worker 执行。
- `projection_gate` 通过后才允许 closeout 判断。
- `closeout_gate` 证据不足时必须保持 blocked、partial 或 risk-accepted，不能手写 done。

AreaFlow-authored workflow version 的源事实是 PostgreSQL。创建新版本时，engine 写入
`workflow_versions`、`workflow_items`、gate/event/audit 记录，并把 skeleton 大内容写为 artifact。
默认不写被管理项目的 `workflow/versions/**` 目录。目录导出是 explicit projection，不是主状态。

长期 trace 使用 `workflow_item_links`：

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

在关系表落地前，links 可以暂存在 item metadata，但 Web、projection 和 closeout 设计应向显式关系收敛。

## Failure Routing

失败必须回到能修复该问题的最小上游层。

| Failure | Return To |
|---|---|
| source docs missing | `source_docs` |
| unresolved open question | `discussion` |
| discussion risk boundary missing | `discussion` |
| middle-layer trace missing | `middle_layer` |
| change 与 middle-layer 不一致 | `changes` |
| plan dependency broken | `plans` |
| draft expected paths missing | `drafts` |
| copy-ready 与 verify-ready 边界不一致 | `drafts` |
| queue label conflict | `queue` |
| promotion preview collision | `queue` 或 `promotion_preview` |
| approval missing | `approval` |
| write permission denied | `approval` 或 project config |
| execution manifest inconsistent | `execution` |
| runner environment failure | `run` triage |
| task verify failure | current run repair |
| checkpoint scope drift | current run repair 或 checkpoint gate |
| projection mismatch | `projection` |
| closeout evidence missing | `closeout` 或对应上游 stage |

## Trace Requirements

任何 live task 至少必须能反查完整链路：

```text
live task label
-> promotion mapping
-> queue candidate
-> draft id
-> plan item
-> change id
-> middle-layer entry
-> discussion decision
-> source docs
```

缺少 trace 时：

- v0.2 doctor 必须报告 coverage gap。
- v0.3 不允许进入 queue ready。
- v0.4 不允许 cutover。
- v0.5+ 不允许 runner 宣称 closeout complete。

## Permission Contract

任何会改变被管理项目或执行外部动作的操作都必须通过权限判断：

```text
capability allowed
path allowed
not forbidden
audit event written
```

v0.1 AreaMatrix 仅允许：

```text
read_project
write_status -> .areaflow/status.json
```

以下能力在 v0.1 明确禁止：

```text
write_workflow
write_generated
write_code
run_commands
manage_git
network
use_secrets
execute_agents
```

## Artifact Contract

Artifact 原文策略：

- v0.1 对 AreaMatrix 历史 artifact 只索引 path、hash、size、type 和 project relation。
- 新 artifact 默认进入 AreaFlow local artifact store，路径位于
  `~/.areaflow/artifacts/{project_key}/...`。
- 数据库只保存 URI、hash、metadata 和关系。
- 长期可把 local artifact store 替换为对象存储，但 URI 和 hash 语义不变。

Artifact 类型建议：

```text
source_doc
workflow_yaml
workflow_markdown
discussion
middle_layer
change
plan
draft_copy
draft_verify
manifest
queue
promotion_preview
approval
execution_prompt
progress
log
checkpoint
report
evidence
projection
closeout
status_export
```

## API And UI Consequences

CLI、Web 和 Desktop 必须围绕同一批对象工作。

核心页面或命令应能展示：

- projects。
- workflow versions。
- stage coverage。
- gate results。
- queue candidates。
- runs。
- trace graph。
- artifacts。
- residuals。
- audit events。

CLI 不应拥有与 API 不一致的私有状态语义。Web 和 Desktop 后续只是不同入口，不重新定义 workflow。

## Phase Adoption

| Phase | Engine Capability |
|---|---|
| v0.1 | import coverage、status mirror、write permission audit、artifact index |
| v0.2 | shadow doctor、hash drift、stage coverage、residual consistency |
| v0.3 | new version authoring、discussion gate、plan/draft/queue/promotion preview |
| v0.4 | approval、live mapping、cutover、compat command forwarding |
| v0.5 | runner preview、execution model、projection、closeout dry-run |
| v0.6 | worker lease、real execution beta、checkpoint gate、agent adapter |
| v0.7 | Web dashboard over the same engine objects |
| v0.8 | multi-project worker pool、resource limits、team-ready scheduling |
| v0.9 | desktop shell over local service and notification model |
| v1.0 | stable adapter/profile boundary、backup、audit、security hardening |

## Non-Goals

当前契约不定义：

- 具体 Web UI 视觉设计。
- AI engine prompt 内容。
- worker sandbox 的最终实现。
- 团队版 secret store 的具体加密方案。
- 所有项目都必须遵循 AreaMatrix profile。

这些内容分别在 Web、runner、security、multi-user 和 adapter/plugin 阶段继续细化。
