# ADR 0006: Platform Operating Boundary

## Status

Accepted. Command API、默认只读、显式授权和审计边界继续有效；当前说明见
[`concepts/commands-and-approvals.md`](../concepts/commands-and-approvals.md) 与
[`architecture/security.md`](../architecture/security.md)。

## Context

AreaFlow 的长期目标不是只替代 AreaMatrix 的几个脚本，而是成为多项目、多 worker、多端协作的
workflow platform。Phase 0 已经确定 Go + PostgreSQL + REST/SSE、AreaMatrix dogfood、local
artifact store、Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover
-> Archive -> Shim Retirement，以及 v0 先 CLI-only 的路线。

本文把 0-100% 讨论中的操作边界固化为实现约束。后续 CLI、API、Web、Desktop、Worker、Adapter、
Projection、Cutover、Execution、Artifact 和 Secret 能力都必须遵守这些边界。

## Decisions

### 1. Command API 是唯一业务写入边界

Query API 只读，Command API 写入，SSE 只通知刷新。

```text
CLI / Web / Desktop / Worker
  -> Command API
    -> permission preflight
    -> idempotency
    -> state mutation
    -> events
    -> audit_events
```

CLI、Web、Desktop 和 Worker 不得各自直接改业务状态表、被管理项目文件或 worker 状态。允许的维护例外
只有 migration、只读 doctor、backup/export 和受控 repair；这些能力必须保持 maintenance boundary，
不得伪装成普通业务写入。

每个业务 command 至少需要：

```text
command_type
project_id
actor
reason
idempotency_key
request_hash
risk_level / risk_policy
capability
affected_resources
response
audit evidence
created_at / completed_at
```

同一 `idempotency_key` 携带不同 `request_hash` 必须拒绝。

### 2. Projection 不是状态源

PostgreSQL 是 workflow、gate、run、artifact、approval、worker、lease 和 audit 的状态源。
Projection 只是外部入口快照。

AreaMatrix 的粗略入口固定为：

```text
.areaflow/status.json
workflow/README.md
```

`.areaflow/status.json` 给工具读，可以在 R1 projection 阶段受控写入。`workflow/README.md` 给人读，
cutover 前只能 preview 或人工维护；authoring cutover 后才允许 AreaFlow 写受控区块。

早期 projection apply 只允许 `.areaflow/status.json`。任何 `workflow/README.md`、`workflow/versions/**`、
execution、progress、logs、checkpoint 或 release evidence 写入都必须等后续独立 gate 和 approval。

### 3. Cutover 分层

Cutover 不是搬目录，也不是一刀切打开所有写权限。

```text
L0 mirror_only
L1 status_projection
L2 authoring_cutover
L3 execution_cutover
L4 platform_primary
```

Authoring cutover 只表示新 workflow version 的 authoring 源事实切到 AreaFlow PostgreSQL +
artifact store。Execution cutover 必须单独证明 runner、worker、permission、approval、evidence、
checkpoint 和 rollback 闭环。

Rollback 追加事实，不删除 events、audit_events、artifacts 或 command_requests。

### 4. Workflow 迁移为 Stage Engine，不迁移成目录主状态

AreaMatrix 的历史目录映射为 AreaFlow profile stage：

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

目录、prompt、报告、diff、日志和 evidence 都是 artifact/source URI。生命周期事实来自：

```text
workflow_versions
workflow_items
workflow_item_links
gate_results
approval_records
runs
run_tasks
run_attempts
artifacts
events
audit_events
```

Authoring stage 和真实 run stage 必须分离。`promotion_preview pass` 不等于 approval；
`approval approved` 不等于 execution 已发生；`live_mapping_gate pass` 不等于 cutover 已完成。

### 5. Runner / Worker / Engine / Adapter 分层

```text
Runner:
  创建 run、拆分 run_task、检查 gate / approval / permission / risk。

Worker:
  注册、heartbeat、领取 scoped lease、执行一个 run_task、提交 attempt / artifact / event / audit。

Engine Adapter:
  封装 Codex CLI、OpenAI API、local model 或 external agent。

Project Adapter:
  理解被管理项目的目录、doctor、验证命令和项目语义。
```

Worker 不拥有状态源事实，不能直接领取 workflow_item，只能通过 lease 领取 run_task。worker capability
不能被请求参数扩大。

Worker 类型从第一版预留：

```text
local_service
local_host
host_bound_worker
remote_worker
container_worker
manual_worker
```

AreaMatrix 默认使用 `local_host` / `host_bound_worker`，因为 macOS app、Xcode、SwiftUI、GUI 验证和
证书能力不能默认 container 化。

### 6. 权限和审计默认保守

AreaFlow 对被管理项目默认只读。任何写入、命令执行、网络访问、secret 使用、git 操作或 agent
execution 都必须显式授权。

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

R2-R4 必须具备 preview、gate、approval、affected resources、rollback wording 和 audit evidence。

### 7. Adapter、Profile、Plugin 分离

```text
Adapter = project IO
Profile = workflow semantics
Plugin = governed extension
```

AreaMatrix adapter/profile 可以先内置，但 core 不能依赖 AreaMatrix 目录结构。Plugin 后续可扩展
adapter、profile、engine provider、artifact backend、notification provider 或 gate checker，但仍必须经过
project scope、capability、allowlist、secret policy、Command API 和 audit。

### 8. Artifact 和 Secret 边界

Artifact 原文不直接进入 PostgreSQL。PG 只保存 metadata、URI、hash、size、type、retention class 和关系。

Retention class 至少包括：

```text
ephemeral
external_ref
run_evidence
audit
release
unknown
```

历史 AreaMatrix 文件默认是 `external_ref`，只索引 metadata，不复制、不删除、不补造历史 evidence。

项目配置只写 `secret_ref`，不写明文 secret。v1.0 前 AreaFlow 只做 secret readiness，不解析 secret、
不注入 engine、不把 secret 写入 artifact、event 或 audit metadata。

### 9. 多端不维护第二状态源

CLI、Web、Desktop 和 Worker 都复用同一 API 和事件语义。

Web v0.7 先做 dashboard + approval console；写操作必须走 Command API。Desktop v0.9 只做 local
service manager、项目切换、通知和 Web 入口，不维护第二数据库、不直接执行 workflow、不解析 workflow。

## Consequences

- 早期实现宁可窄，也不能提前打开真实 execution、secret resolve、restore apply、publish apply 或
  `workflow/README.md` 自动写入。
- `project.status_projection.apply` 第一阶段只能写 `.areaflow/status.json`，并且必须记录
  command_requests、events、audit_events 和 status_projections。
- AreaMatrix 真实项目写入只能在用户明确授权时发生；fixture smoke 可以写临时项目目录。
- Web、Desktop、compatibility shim 和 Worker 都不能维护第二套 progress 或 workflow 状态。
- 后续偏离本文的设计必须新增 ADR 或更新本文，并说明 permission、audit、migration 和 rollback 影响。
