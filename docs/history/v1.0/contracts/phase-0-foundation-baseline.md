# ADR 0005: Phase 0 Foundation Baseline

## Status

Superseded as an implementation baseline. 本文保留早期地基决策历史；当前产品事实以
[`docs/README.md`](../../../README.md)、[`architecture/overview.md`](../../../architecture/overview.md) 和
[`concepts/product-model.md`](../../../concepts/product-model.md) 为准。

## Context

AreaFlow 的 0-100% 路线已经确定为独立平台路线：先建立 PostgreSQL 状态源事实和
AreaMatrix dogfood 迁移协议，再逐步进入 authoring cutover、runner/worker、Web、多项目、
Desktop 和 v1.0 稳定化。

本文收束 Phase 0 讨论中的 0-100% 地基决策。后续 milestone、API、schema、Web、Desktop、worker 和
AreaMatrix compatibility shim 都必须以这些决策为边界。

## Decisions

### 1. Workflow Profile

AreaFlow core 使用 profile-driven workflow stage engine。AreaMatrix 是第一个内置 profile，
不是 core 的硬编码流程。

```text
adapter = 怎么连接、读取、可选写入项目
profile = workflow stage / gate / transition 语义
core = 状态转移、权限、审计和执行安全
```

AreaFlow-authored workflow version 必须冻结 `profile_id`、`profile_version` 和 `profile_hash`。
后续 profile 升级只能显式迁移，不能静默改变既有 version 的 gate 或 transition 规则。

### 2. Data Model And State Separation

AreaFlow 不使用一个万能状态字段表达所有事实。综合状态可以由 API/Web 计算，但写入时必须写入对应事实表。

```text
phase_state:
  workflow_versions + workflow_items + workflow profile

gate_state:
  gate_results

approval_state:
  approval_records

runtime_state:
  runs / run_tasks / run_attempts

lease_state:
  leases / worker_heartbeats

projection_state:
  status_projections 或 project_status_snapshots

security_state:
  audit_events / permissions / command_requests
```

`workflow_item` 是 stage 内语义交付物，不是单文件、单命令或单 prompt。文件、prompt、报告、diff、
日志和 evidence 都必须作为 artifact 关联到 item、run、task 或 attempt。

v0 可以继续用 `project_status_snapshots` 承载部分投影历史，但长期必须收敛到
`status_projections`，用于记录 `.areaflow/status.json`、`workflow/README.md`、Web dashboard
和 Desktop 观察面的 target、write_state、payload、source event 和写入结果。

### 3. AreaMatrix Rough Entry

AreaMatrix 最终只保留粗略入口：

```text
workflow/README.md
.areaflow/status.json
```

`.areaflow/status.json` 给工具读，`workflow/README.md` 给人读。它们都是 projection，不是主状态源。
详细 stage、task、run、attempt、artifact、approval、worker、lease 和 audit 状态必须在 AreaFlow 中查询。

阶段边界：

```text
v0.1-v0.3:
  只允许 .areaflow/status.json
  workflow/README.md 只能 preview 或人工维护

v0.4 cutover 后:
  才允许 AreaFlow 写 workflow/README.md 的受控区块
```

### 4. API Surface And Client Boundary

AreaFlow API / service layer 是唯一业务边界。CLI、Web、Desktop 和 worker 都只是 client。

```text
CLI / Web / Desktop / Worker
  -> REST API + SSE
    -> service layer
      -> PostgreSQL
      -> artifact store
      -> workflow engine
      -> project adapters
      -> worker pool
      -> engine adapters
```

API 分三类：

```text
Query API:
  只读查询，不写文件、不执行命令、不解析 secret、不调度 worker。

Command API:
  所有写入、执行、审批、projection、cutover、restore、publish 和 worker 控制入口。

SSE:
  事件观察通道，不是状态源；断线重连后必须通过 Query API 补齐事实。
```

Web v0.7 先做只读 dashboard。Desktop v0.9 先做 local service shell。两者都不得维护第二数据库、
第二套 progress 或绕过 AreaFlow API 的写入通道。

### 5. Command API

所有写动作最终必须进入 Command API。CLI、Web、Desktop 和 worker 不得各自直接改数据库、
项目文件或 worker 状态。

第一批 command 边界为：

```text
workflow.version.create
workflow.approval.record
workflow.cutover.preview
workflow.cutover.apply
runner.preview
```

其中 `workflow.cutover.apply`、restore apply、release exception apply、publish apply 均属于后续 R4
能力，Phase 0/v0.4 只能先落 preview、gate、readiness 和 rollback 证据，不得因为 command type
已命名就提前打开真实 apply。

暂不打开：

```text
runner.run
worker.run_once as real execution
task-loop replacement
write execution
write code
secret resolve
git push / tag / release publish
```

长期 command request 必须收敛到：

```text
project_scope
command_type
actor
reason
idempotency_key
request_hash
expected_version
risk_level
risk_policy
permission_preview
approval_state
status
response
audit_event_id
created_at
completed_at
```

同一 `idempotency_key` 携带不同 `request_hash` 必须拒绝。

### 6. Artifact Lifecycle

Artifact 原文不直接进入 PostgreSQL。PostgreSQL 只保存 metadata、URI、hash、size、type 和关系。

Artifact 分三类：

```text
historical reference:
  历史 AreaMatrix 文件，只索引 metadata，不复制原文

managed copy:
  AreaFlow 新生成或拥有的 workflow skeleton / prompt / approval bundle / report

generated evidence:
  runner / worker / doctor 产生的日志、报告、diff、failure summary、verify evidence
```

历史 AreaMatrix artifact 在 v0.1-v0.6 默认为 `external_project` / `project_reference`。不得在
cutover 时隐式复制、删除、覆盖或补造历史 evidence。

AreaFlow 新产物从 v0.3 起写入 local artifact store：

```text
~/.areaflow/artifacts/{project_key}/...
```

备份恢复在 v1.0 前只做 manifest、metadata inventory、integrity check 和 restore dry-run plan。
只要存在历史 project reference artifact，restore plan 必须返回 `needs_attention`，不能伪装成完整可恢复。

### 7. Permission And Write Policy

AreaFlow 默认只读。任何写入、命令执行、网络访问、secret 使用、git 操作或 agent execution 都必须显式授权。

统一判断公式：

```text
capability allowed
+ resource allowlist matched
+ forbidden list not matched
+ required gate passed
+ approval present when needed
+ audit event written
= allowed
```

Deny 永远优先于 allow。

Capability 语义必须保持分离：

```text
write_artifacts != write_project_files
manage_workers != execute_agents
approval != execution
runner_preview != runner_run
```

风险等级：

```text
R0 read_only
R1 projection
R2 managed_write
R3 execution
R4 migration_security
```

R2-R4 必须有明确 gate、approval、preview、affected resources、rollback wording 和 audit。

### 8. Worker Execution Model

Worker 不拥有状态源事实。Worker 领取 `run_task`，不直接领取 `workflow_item`。

```text
workflow_item = 要做什么
run = 一次执行会话
run_task = worker 可领取的执行单元
lease = worker 对 run_task 的临时占用
run_attempt = copy / verify / repair / checkpoint 的一次尝试
artifact = 输入、输出、日志、报告、evidence
```

并发和恢复规则：

```text
同一个 run_task 同时最多一个 active lease
领取任务使用 FOR UPDATE SKIP LOCKED
lease 过期后进入 needs_recovery
heartbeat 超时不等于失败
worker 完成必须提交 attempt + artifact + event + audit
capability denied 时不得创建 lease / attempt / artifact
```

真实执行顺序：

```text
approval passed
-> runner_gate passed
-> run created
-> run_task queued
-> worker acquire lease
-> copy attempt
-> verify attempt
-> repair attempt, if verify failed and retry allowed
-> checkpoint attempt
-> projection
-> closeout
```

`verify fail` 不能 mark done；`checkpoint fail` 不能进入下一 task；repair 只能追加 attempt，
不能覆盖历史 attempt。

### 9. Adapter, Profile And Plugin Boundary

AreaMatrix adapter 和 AreaMatrix profile 可以先内置，但 AreaFlow core 不能依赖 AreaMatrix 目录结构。

```text
Adapter:
  负责 project IO、snapshot、metadata import、drift、projection 和 allowed native command preview。

Profile:
  负责 stage、gate、transition、required artifacts、rollback route 和 closeout semantics。

Plugin:
  后续受控扩展 adapter、profile、engine provider、artifact backend、notification provider 或 gate checker。
```

Adapter 不直接写数据库、不决定 gate/approval、不调用 AI engine、不绕过 permission evaluator。Profile
不读磁盘、不执行命令、不处理 secret。Plugin 不能成为任意代码执行入口，所有 plugin 动作仍必须经过
project scope、capability、allowlist、secret policy、Command API 和 audit。

阶段策略：

```text
v0-v0.4:
  built-in adapters/profiles

v0.5-v0.8:
  registry 接口稳定

v1.0:
  plugin 边界稳定

v1.x:
  再考虑第三方 plugin 安装、签名、版本兼容和沙箱
```

### 10. Web And Desktop Boundary

Web 和 Desktop 都只是 AreaFlow client，不是第二套 AreaFlow。

```text
Web / Desktop / CLI / Worker
  -> AreaFlow API + SSE
    -> service layer
      -> PostgreSQL
      -> artifact store
      -> workflow engine
```

Web v0.7 先做只读 dashboard。SSE 只是刷新通道，不是状态源。

Desktop v0.9 只做 local service shell、dashboard launcher、通知和本机健康观察。Desktop 不维护第二数据库、
不直接执行 workflow、不直接写项目、不解析 workflow、不保存第二套 progress。

所有写操作只能通过 Command API，并显示 risk preview、capability、affected resources、approval、
permission preflight 和 audit outcome。

### 11. Cutover And Rollback

Cutover 不是搬目录，而是逐步切换 workflow ownership。AreaMatrix 迁移阶段必须使用完整口径：

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

短名可以写作 Import -> Mirror -> Shadow -> Cutover -> Archive，但设计和实现必须区分
authoring cutover 与 execution cutover。

```text
v0.4 cutover =
  新 workflow version authoring 从 AreaMatrix 目录切到 AreaFlow PostgreSQL + artifact store

not:
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
stage_coverage pass 或 warn 已解释
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

`cutover_readiness_gate = pass` 只表示前置证据满足，不代表 cutover apply、execution write、
task-loop replacement 或 AreaMatrix shim 修改已经发生。

Rollback 是追加事实，不删除历史：

```text
pre-cutover fail:
  gate 没过，不切换，停在 mirror/shadow

cutover apply fail:
  记录 cutover_failed event/audit，保持 AreaMatrix project-owned

post-cutover drift:
  冻结新 authoring，跑 recovery doctor，必要时 soft rollback

soft rollback:
  新 workflow version 切回 project-owned 或 read-only frozen
  AreaFlow 保留 event/audit/artifact

hard rollback:
  只在尚未写项目文件、未开始 execution 时允许
  标记 cutover attempt failed
  不删除 AreaFlow 历史记录
```

### 12. Release, Backup, Restore And Audit Final Gate

AreaFlow v1.0 不能只靠测试通过或 UI 可用来宣称稳定。发布前必须具备可查询、可复验的证据链：

```text
backup manifest
-> restore plan
-> artifact integrity
-> audit coverage
-> permission doctor
-> adapter/profile conformance
-> release readiness
-> remediation plan
-> acceptance preview
-> acceptance gate
-> exception apply preview
-> final gate
-> release package preview
-> distribution preview
-> publish gate
-> publish approval preview
-> rollout plan preview
```

`project_reference` / `external_project` artifact 不能被当作完整可恢复原文。只要历史原文仍留在
AreaMatrix 或其他被管理项目中，restore plan 和 release readiness 必须返回 `needs_attention`，
除非存在明确 release exception、owner、证据、风险说明和后续归档计划。

可以被显式接受的 release exception：

```text
metadata_only_history
future_only_gap
archive_exception
```

不能通过 exception 放行的 blocker：

```text
backup manifest broken
permission policy fail
adapter/profile conformance fail
local artifact hash mismatch
secret 泄露风险
Command API 幂等/审计缺失
真实写入无 rollback
```

Exception apply、restore apply、publish apply、secret resolve 和远程 worker credential 都属于
R4 migration_security，必须经过单独 Command API、approval、audit 和 rollback 设计。

## Consequences

- Phase 0 之后的实现不能用“展示效果”提前打开 execution、secret、restore apply、publish apply 或
  release apply。
- AreaMatrix v1 历史只读 immutable，不回填、不重写、不补造。
- AreaFlow 的新内容必须落入 PostgreSQL + artifact store，并通过 events/audit_events 留痕。
- AreaMatrix 最终保留项目内容和粗略入口；详细 workflow 状态由 AreaFlow 拥有。
- Web、Desktop 和 compatibility shim 都不得维护第二套状态。
- 任何偏离本文的后续设计必须新增 ADR 或更新本文，并说明 migration、permission、audit 和 rollback
  影响。
