# Data Model v1

## 定位

本文定义 AreaFlow v1 目标数据模型。v0.1 schema 可以继续保持轻量，但后续
migration、API 和 Web/Desktop 设计应向本文收敛。

AreaFlow 的数据模型按多项目、多用户、多 worker、多端使用设计。AreaMatrix 是第一个
dogfooding 项目，不是 schema 特例。

## 核心原则

- PostgreSQL 是主状态源事实。
- `events` 与 `audit_events` append-only。
- artifact 原文不直接进入数据库。
- 所有业务对象都带 project scope，避免多项目状态污染。
- 业务命令必须具备幂等键。
- 关键状态转移使用 optimistic concurrency，也就是 `expected_version` 检查。
- 历史导入默认 immutable；修正历史使用追加事件，不改写旧事实。

## 对象分层

```text
Identity
  actors
  users
  teams
  memberships
  project_memberships
  api_tokens
  audit_events

Project
  projects
  project_connections
  project_permissions
  adapters
  workflow_profiles

Workflow
  workflow_versions
  workflow_items
  workflow_item_links
  gate_results
  transitions
  residuals

Execution
  runs
  run_tasks
  run_attempts
  workers
  leases
  worker_heartbeats
  checkpoints
  run_control_requests

Artifact
  artifacts
  artifact_locations
  artifact_snapshots
  status_projections

Integration
  engine_profiles
  engine_runs
  secret_refs
  integration_connections
  webhook_deliveries
  inbound_webhook_events
  usage_meters
  budget_policies
  quota_policies
  budget_reservations
  command_requests
  webhooks
```

## Schema Landing

v1 边界不是等到 v1.0 才一次性落表。AreaFlow 采用“先保留平台边界，后打开能力”的方式：

```text
000001-v0.8:
  projects、workflow_versions、workflow_items、gate_results、approval_records、
  runs、run_tasks、run_attempts、workers、leases、workflow_item_links

000009_v1_boundary_foundation:
  users、teams、memberships、project_memberships、adapters、workflow_profiles、project_configs、
  artifact_locations、artifact_snapshots、secret_refs、engine_profiles、api_tokens、webhooks

000010_v1_status_projections:
  status_projections
```

`000009_v1_boundary_foundation` 只声明长期平台实体，不代表已经启用多用户登录、secret 解析、
webhook 调用、远程 API token 或真实 engine execution。能力打开仍必须经过 permission、gate、
approval 和 audit。

这些实体的 auth、team、API token、secret resolve 和 remote worker credential 开闸顺序见
[`auth-team-secret-boundary.md`](../../proposals/auth-team-secret.md)。在对应 R4 rung 通过前，schema row 只能作为
readiness / doctor / preview 证据，不能改变 API authorization 或 secret 解析结果。

`000010_v1_status_projections` 落地长期 projection metadata，不代表已经打开 `workflow/README.md`
受控写入、cutover apply、Desktop 状态写入或 Web 写操作。第一阶段只记录 `.areaflow/status.json`
mirror export 和只读查询。

## Identity

v0 可以只有 `actors`，但 v1 需要保留团队扩展语义。

```text
actors:
  id
  kind: system | local_user | human | user | service | worker | api_token | cli_token | agent | areamatrix_shim
  display_name
  external_key

users:
  id
  email
  display_name
  status

teams:
  id
  name
  status

memberships:
  team_id
  user_id
  role

project_memberships:
  project_id
  user_id nullable
  team_id nullable
  role
  status
  effective_from
  effective_to nullable

api_tokens:
  actor_id
  project_id nullable
  token_key
  token_hash
  scope
  status
  expires_at nullable
  revoked_at nullable
```

本机单用户模式也应创建稳定 actor，例如 `local-user` 和 `system`，避免后续补审计时失去来源。
`memberships` 表达 team membership；`project_memberships` 才表达某个 user/team 在具体 project 上的
角色上限。两者都不能直接授予 project write、secret resolve、publish、restore 或 worker credential；
真实许可仍由 project config、capability、resource allowlist、gate、approval 和 audit 共同决定。
`api_tokens.token_hash` 是唯一可持久化的 token 认证材料；token 明文只能在 issuance response 中出现一次，
并且在 R4-2/R4-3 打开前不得改变 API authorization 结果。

## Project

```text
projects:
  project_key
  name
  kind
  adapter
  workflow_profile
  status
  default_branch
  version

project_connections:
  project_id
  connection_type: local_path | git_remote | api
  root_path
  remote_url
  current_branch
  current_commit

project_permissions:
  project_id
  capability
  effect: allow | deny
  resource_type: path | command | secret | network | git | worker
  pattern

project_scheduling_policies:
  project_id
  priority
  max_parallel_tasks
  agent_role
  required_capabilities
  engine_profile
  metadata

project_configs:
  project_id
  protocol_version
  config_path
  config_hash
  ownership
  permissions
  scheduling
  engines
  status_export
  migration
  active
```

`project_key` 是查询、事件、artifact、worker lease 和权限判断的项目边界。所有跨项目 API 都必须显式带 scope。

### Workspace And Environment

v1.0 前不单独引入复杂 `workspaces` 或 `environments` 表。原因是 AreaFlow 当前最重要的不变量是
project scope 隔离，而不是提前定义团队 UI 或部署环境层级。

短期映射规则：

```text
workspace:
  由 team、project grouping、UI filter 或 metadata 表达。

environment:
  由 project_connections、worker kind、engine_profiles、secret_refs、scheduling policy 和 metadata 表达。
```

当远程团队控制台、多租户部署、跨环境 promotion 或真实远程 worker 成为 v1.x 能力时，再把
workspace / environment 提升为一等实体。提升时仍不得破坏 `project_key` 对 workflow、run、artifact、
secret、worker lease 和 audit 的隔离语义。

## Workflow

Workflow 不按 AreaMatrix 文件夹建表。统一模型为：

```text
workflow_versions
  -> workflow_items
    -> workflow_item_links
    -> gate_results
    -> artifacts
    -> events
    -> runs
```

```text
workflow_versions:
  project_id
  display_label
  version_kind
  lifecycle_status
  source_path
  source_hash
  import_mode
  immutable
  status_summary
  version

workflow_items:
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
  version
```

`workflow_items` 的粒度是 stage 内语义交付物，不是单文件、单命令或单 prompt。单个 item
可以挂多个 artifacts；真实执行进入 `runs -> run_tasks -> run_attempts`。

AreaMatrix profile 当前采用的稳定 `workflow_items.item_type` 为：

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

`prompt_package` 只能作为 API / UI 聚合视图，不作为单个可执行或可审批 item。manifest、copy-ready
和 verify-ready 必须分别落在 `draft_manifest`、`draft_copy` 和 `draft_verify`，否则 copy / verify
边界和后续 evidence trace 会被隐藏。

`workflow_versions.lifecycle_status` 只表达版本所处的粗粒度生命周期位置，建议枚举为：

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

这些值不能替代 `gate_results`、`approval_records`、`runs`、`leases` 或 `status_projections` 的事实。
例如 `lifecycle_status=approved` 只表示 authoring/approval 阶段完成，不表示 worker 已执行或项目文件已写入。

```text
workflow_item_links:
  project_id
  workflow_version_id
  from_item_id
  to_item_id
  relation_type: derives_from | implements | verifies | promotes_to | depends_on | blocks | supersedes | closes
  metadata
  created_at
```

`workflow_item_links` 应作为早期正式模型，而不是长期放在 item metadata。v0.3-v0.4 可以先只写入
`derives_from`、`implements`、`verifies` 和 `promotes_to` 等最小关系；v0.5-v1.0 再扩展
`depends_on`、`blocks`、`supersedes` 和 `closes`，支撑 Web trace、projection 和 closeout。

`import_mode` 目前至少包含：

```text
metadata_only
authored
```

`metadata_only` 表示从被管理项目导入的 immutable metadata index。`authored` 表示由 AreaFlow
创建并拥有的 workflow version candidate。后续 import 刷新只能替换 imported metadata，不能删除
`authored` workflow versions。

标准 item 状态保持少量稳定枚举：

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

更细的原因放入 `metadata`、`events`、`gate_results`，不扩大状态枚举。

`stage=queue` 且 `item_type=queue_candidate` 的 workflow item 表示“准备执行什么”，不是某次执行本身。
同一个 queue candidate 可以产生多个 `runs`，用于表达失败重跑、repair 后重验、不同 worker 策略或
manual retry。旧 run 和 attempt 不覆盖、不删除，只通过 `superseded`、`failed`、`cancelled` 或后续
closeout 追加事实表达。

## 状态分层

AreaFlow 不把 phase、gate、approval、runtime、lease 和 projection 压进同一个字段。`workflow_items.status`
是 item 的粗粒度状态；其他判断来自各自表：

```text
phase_state:
  由 workflow_versions / workflow_items / workflow profile 计算。

gate_result:
  来自 gate_results，表达 pass / warn / fail / blocked / needs_approval。

approval_state:
  来自 approval_records，表达 approved / rejected / pending。

runtime_state:
  来自 runs / run_tasks / run_attempts。

lease_state:
  来自 leases / worker_heartbeats。

projection_state:
  来自 status_projections。
```

API 和 Web 可以组合这些字段展示综合状态，但写入时必须写对应事实表，不能更新一个“万能 status”。
同理，`run_task.status` 只表达通用 worker / runtime 生命周期；能力专属结果必须进入 outcome、
attempt、artifact、gate result 或 summary，不能不断扩展 status 枚举。

## Gate Results

Gate 是一次可重复判断，不是状态本身。

```text
gate_results:
  project_id
  workflow_version_id
  workflow_item_id nullable
  run_id nullable
  gate_name
  scope_type
  scope_id
  status: pass | warn | fail | blocked | needs_approval | error
  inputs jsonb
  source_hashes jsonb
  failures jsonb
  warnings jsonb
  evidence_artifact_ids jsonb
  checked_at
  actor_id
  metadata
```

AreaFlow 必须能回答：“这个 item 为什么可以前进？”答案来自 gate result 和证据 artifact，而不是手写绿色状态。

v0.3c 已落地 `discussion_gate`、`plan_doctor`、`draft_doctor`、`queue_doctor` 和
`promotion_preview` 的 gate result 记录。缺少 required workflow item 时结果为 `blocked`；
skeleton-only artifact 结果为 `fail`，不能自动把下游 item 推进到 `ready`。

## Transition Previews And Approvals

Transition preview 是一次只读推进预演，不是状态转移本身。

```text
workflow_transition_previews:
  project_id
  workflow_version_id
  from_stage
  to_stage
  status: ready | blocked
  required_gate_name
  gate_result_id nullable
  blockers jsonb
  warnings jsonb
  metadata jsonb
  created_at
```

Approval record 是显式人工或系统决策的审计事实，不代表 execution 已经发生。

```text
approval_records:
  project_id
  workflow_version_id
  transition_preview_id nullable
  approval_kind
  decision: approved | rejected
  scope_type
  scope_id
  actor
  reason
  risk_level
  metadata jsonb
  created_at
```

v0.3d 只允许在 transition preview 为 `ready` 时记录 `approved`。blocked preview 可以记录
`rejected`，但不能被记录成可执行批准。

`approval` 作为 workflow stage 保留，但审批事实必须落在 `approval_records` 和 `audit_events`。
`promotion_preview pass`、`approval approved`、`live_mapping_gate pass` 三者不能合并成一个状态。
`approval_records.metadata` 至少应能保存 gate snapshot、allowed capabilities、allowed resources、
forbidden resources、precondition snapshot、command class、expiry、rollback/remediation reference 和 human
reason；缺少这些事实的 approval 只能作为草稿或历史导入记录，不能作为 R2-R4 操作的执行许可。

## Execution

```text
runs:
  project_id
  workflow_version_id
  run_kind
  status
  risk_policy
  started_by_actor_id
  worker_id
  summary
  version

run_tasks:
  run_id
  project_id
  workflow_version_id nullable
  workflow_item_id
  task_key
  task_kind
  status
  outcome nullable
  order_index
  risk_level
  copy_ready_artifact_id nullable
  verify_ready_artifact_id nullable
  retry_policy

run_attempts:
  run_id
  run_task_id
  workflow_item_id
  attempt_no
  attempt_kind: copy | verify | repair | doctor | checkpoint | worker_run_once
  status
  input_artifact_id
  output_artifact_id
  failure_summary
  validation_summary
```

一次用户可见执行是 `run`；copy、verify、repair、checkpoint 都是 `attempt`。Attempt 不覆盖历史。
`worker_run_once` 是 worker loop 的 dry-run 证据 attempt，用于证明 worker 领取、释放和 evidence
写入链路，不代表真实 copy/verify 执行。

`run_tasks.status` 目标枚举应收敛为：

```text
queued
pending
leased
preflight_running
running
verifying
checkpoint_ready
passed
failed
blocked
repair_needed
retry_scheduled
cancel_requested
cancelled
lease_expired
needs_recovery
superseded
```

`outcome` 用于表达 capability-specific result，例如 `read_only_verify_passed`、
`artifact_write_passed` 或 `rollback_verified`。v0.6 早期实现中暴露的 `verified`、
`artifact_written`、`rollback_verified` 等 scoped status 只能作为兼容读取处理；新增能力不得继续扩大
`status` 枚举，后续 schema/API 整理应迁移为 `status=passed` 加 `outcome`、attempt 和 artifact evidence。

## Worker And Lease

Worker、lease 和多项目调度的行为合同见
[`worker-scheduling-contract.md`](worker-scheduling-contract.md)。本节只定义模型落点。

```text
workers:
  id
  worker_key
  worker_kind
  status
  capabilities
  last_seen_at

leases:
  id
  worker_id
  project_id
  run_id
  run_task_id
  workflow_item_id
  lease_kind
  status
  acquired_at
  expires_at
  heartbeat_at

worker_heartbeats:
  worker_id
  status
  metadata
  created_at
```

Worker 只通过 lease 工作，并且领取对象是 `run_task`，不是 `workflow_item`。`workflow_item`
定义要做什么，`run_task` 定义本次 run 中可被 worker 占用和执行的单元。多 worker 抢任务使用
PostgreSQL `FOR UPDATE SKIP LOCKED`，同一 `run_task` 同时最多一个 active lease；capability
denied 时不得创建 lease、attempt 或 artifact。

`leases.status` 当前稳定语义为：

```text
active
completed
released
needs_recovery
```

`active` lease 到期后必须通过 recovery command 进入 `needs_recovery`，不能自动把 task 判为 failed。
`completed` 必须对应 attempt / artifact / event / audit evidence；`released` 只表示释放，不表示成功。

## Artifact

Artifact 存储、integrity、backup manifest、restore dry-run、archive / retention 和 restore apply 边界见
[`artifact-backup-restore-contract.md`](artifact-backup-restore-contract.md)。本节只定义模型落点。

Artifact 分三类：

```text
source_reference
managed_copy
generated_output
```

```text
artifacts:
  project_id
  workflow_version_id nullable
  workflow_item_id nullable
  run_id nullable
  run_task_id nullable
  run_attempt_id nullable
  artifact_type
  storage_backend: project_reference | local | object
  uri
  source_path
  sha256
  size_bytes
  content_type
  retention_class: ephemeral | run_evidence | audit | release | external_ref
  visibility
  metadata
  immutable

artifact_locations:
  artifact_id
  storage_backend
  uri
  location_role
  sha256
  verified_at

artifact_snapshots:
  artifact_id
  location_id nullable
  snapshot_kind
  source_hash
  metadata
```

PG 保存 metadata、hash、URI 和关系。prompt、日志、报告、diff、verify evidence 等大内容保存在 artifact store。

本地 artifact store 默认使用平台级 root：

```text
~/.areaflow/artifacts/{project_key}/{workflow_version_or_global}/{category}/{artifact_id}-{sha256-prefix}{extension}
```

`project_key` 是稳定 namespace，避免数据库内部 id 在迁移/恢复后改变路径语义。

Artifact doctor / integrity report 必须把状态拆开，不能只给一个绿色结果：

```text
local_verified:
  local / object 原文存在，sha256 和 size 与 PG metadata 一致。

metadata_only:
  只有 metadata，例如历史导入索引。

external_ref_skipped:
  project_reference 原文仍在被管理项目，AreaFlow 不读取、不删除，只返回 warn/skipped。

missing_blob:
  PG 有 AreaFlow-owned artifact metadata，但 storage backend 缺原文。

hash_mismatch:
  原文存在但 sha256 或 size 不一致。

orphan_blob:
  storage backend 有文件，但 PG 没有 artifact metadata。
```

`retention_class` 决定 GC 和 backup 行为。`ephemeral` 可显式清理；`run_evidence` 至少保留到 run
closeout；`audit` 和 `release` 默认长期保留；`external_ref` 不由 AreaFlow 删除原文件。

当前 `artifact.archive.preview` command 只根据 artifact metadata 生成 archive / retention 预演结果，
并写入 command response、event 和 audit event。它不新增 artifact location，不复制原文，不移动或删除
artifact store 文件，也不修改被管理项目。真实 archive copy、retention-aware GC、orphan cleanup 和 delete
apply 属于后续高风险 command，必须单独 approval / audit / rollback。

## Status Projections

Projection 是外部项目、Web、Desktop 或兼容命令读取的轻量快照。它不替代主状态表。

```text
status_projections:
  project_id
  workflow_version_id nullable
  target_kind: project_status_json | workflow_readme | web_dashboard | desktop_shell | api_summary
  target_uri
  summary_state
  payload_json
  source_event_id nullable
  source_hash nullable
  write_state: previewed | approved | written | stale | failed | skipped
  generated_at
  written_at nullable
  metadata jsonb
```

`.areaflow/status.json` 和 `workflow/README.md` 都是 projection target。写入 projection 必须通过
permission、gate 和 audit；projection payload 不能成为恢复 run、approval 或 artifact 的权威来源。

`summary_state` 是给外部入口显示的粗粒度状态，不能承载 gate、attempt、lease 或 approval 的全部细节。
推荐枚举保持稳定：

```text
not_imported
imported
mirroring
shadowing
authoring
awaiting_approval
executing
blocked
ready_for_closeout
closed
archived
```

`write_state` 表示 projection 与目标之间的写入关系。`previewed` 表示只生成预览；`approved` 表示
已允许写入但尚未完成；`written` 表示目标已更新；`stale` 表示源事件或 hash 已变化；`failed` 表示写入失败；
`skipped` 表示当前阶段或策略禁止写入。AreaMatrix v0.1-v0.3 只能把 `.areaflow/status.json` 作为可写
target；`workflow/README.md` 必须等 authoring cutover gate 通过后，才允许写 AreaFlow 受控区块。

## Integration

```text
secret_refs:
  project_id nullable
  secret_name
  provider: env | keychain | encrypted_store | external
  scope
  status: declared | missing | unavailable | ready | disabled | policy_denied

engine_profiles:
  id
  project_id nullable
  provider
  secret_ref
  capabilities
  budget_policy
  status: enabled | disabled | unavailable
  readiness_status: ready | blocked | needs_secret | disabled

usage_meters:
  project_id
  command_request_id nullable
  run_id nullable
  run_task_id nullable
  attempt_id nullable
  engine_profile_id nullable
  actor_id
  provider
  resource_kind
  quantity_estimated
  quantity_observed nullable
  unit
  source: estimate | provider_report | manual_adjustment
  idempotency_key
  audit_event_id

budget_policies:
  project_id
  policy_key
  scope
  resource_kind
  limit_value
  window
  enforcement_state: preview | fixture | enforced | suspended
  override_policy

quota_policies:
  project_id
  policy_key
  resource_kind
  limit_value
  window
  enforcement_state: preview | fixture | enforced | suspended

budget_reservations:
  project_id
  command_request_id
  run_id nullable
  resource_kind
  quantity_reserved
  status: active | released | charged | expired
  expires_at
  audit_event_id

command_requests:
  project_id
  actor_id
  command_type
  command_class
  idempotency_key
  request_hash
  expected_version nullable
  risk_level
  risk_policy
  permission_preview
  approval_state
  precondition_snapshot
  affected_resources
  forbidden_resources
  rollback_or_remediation_ref
  safety_facts
  status
  audit_event_id nullable
  result_ref

webhooks:
  project_id
  webhook_key
  url
  event_types
  secret_ref
  status

integration_connections:
  project_id
  connection_key
  provider
  direction: outbound | inbound | bidirectional
  allowed_event_types
  allowed_command_types
  allowed_network_targets
  secret_ref nullable
  signing_key_ref nullable
  status

webhook_deliveries:
  project_id
  webhook_id
  event_id
  delivery_id
  idempotency_key
  payload_hash
  target_url_hash
  status
  attempt_count
  next_retry_at nullable
  audit_event_id

inbound_webhook_events:
  project_id
  connection_id
  provider
  event_type
  payload_hash
  signature_status
  replay_window_status
  command_preview
  status
  audit_event_id
```

Budget / quota / usage metering 的行为边界见
[`budget-quota-boundary.md`](../../proposals/budget-and-quota.md)。这些表是长期目标模型；在对应 B-rung 通过前，
row 只能作为 readiness / preview / fixture evidence，不能扣减 quota、阻断真实 run、调用 provider
billing API 或写真实 charge。
Integration / webhook 的行为边界见
[`integration-webhook-boundary.md`](../../proposals/integrations-and-webhooks.md)。这些表在对应 I-rung 通过前只能作为
metadata / readiness / preview / fixture evidence；不能投递 webhook、处理 callback、调用外部 API、
启动 delivery worker 或把 provider callback 当作业务事实。

v0.3a 的实际 migration 先按 project scope 落地：

```text
command_requests:
  project_id
  command_type
  idempotency_key
  request_hash
  response
  completed_at
```

同一 `project_id + command_type + idempotency_key` 只能对应一个 request hash。重复提交同一
request hash 返回同一业务结果；同一 key 携带不同 request hash 必须拒绝。后续多用户阶段再把
actor 维度提升为强约束。
Command request 的行为合同、approval scope、permission 顺序、expected version/hash、audit 和 rollback
要求见 [`command-approval-contract.md`](../history/v1.0/contracts/command-approval-contract.md)。本节只描述数据落点。

当前已接入的 command type 包含 `project.import`、`project.cutover.apply`、`workflow.version.create`、`workflow.approval.record`、
`runner.preview`、`project.status_projection.write`、`project.status_projection.apply`、`project.doctor.record`、`worker.register`、
`worker.heartbeat`、`lease.acquire`、`lease.release` 和 `lease.recover`。其中 `project.import` 覆盖 AreaMatrix metadata index 重建、
import run、status snapshot 和 import audit event 写入；默认未传幂等键时每次生成新的 command key，
以保留 import snapshot/history，显式 `idempotency_key` 则重放同一 import 结果。
`project.cutover.apply` 覆盖 v0.4 authoring cutover 的 AreaFlow DB 内状态切换、workflow event 和
audit event 写入；它要求 cutover readiness 与 `cutover_readiness_gate` 通过，并在 command response
中保留 `project_write_attempted=false` 与 `execution_write_attempted=false`，不写 AreaMatrix 文件、
不创建 execution。
`workflow.approval.record` 覆盖 `approval_records`、workflow event 和 audit event 写入；
`project.status_projection.write` 是 legacy status mirror export 的数据库记录边界；
`project.status_projection.apply` 覆盖 `.areaflow/status.json` 的受保护写入、
数据库侧 `project_status_snapshots`、`events`、`audit_events` 和 `status_projections` 写入，并保持
`execution_write_attempted=false` 与 `engine_call_attempted=false`；
`project.doctor.record` 覆盖 `project.doctor.completed` event 写入；worker lifecycle command 覆盖
`workers`、`worker_heartbeats`、worker event 和 audit event 写入；worker lease command 覆盖 `leases`
状态变化、run task 状态同步、worker event 和 audit event 写入。真实项目文件写入、
`workflow/README.md` 人类可读 projection、AreaMatrix shim 修改和 execution cutover 仍需单独 Command
API / permission / approval 设计。

长期 command request 是所有写动作的统一入口。CLI、Web、Desktop 和 Worker 可以使用不同交互形式，
但不能绕过 command_requests 直接改变 workflow、run、lease、artifact archive、release exception、
restore apply 或 publish apply 状态。

command_requests 至少需要表达：

```text
actor
project_scope
command_type
command_class
idempotency_key
expected_version
risk_level
risk_policy
permission_preview
approval_state
precondition_snapshot
affected_resources
forbidden_resources
rollback_or_remediation_ref
safety_facts
status
audit_event_id
result_ref
```

R0/R1 command 可以直接完成并记录轻量 audit；R2-R4 command 必须先有 risk preview、permission
preflight、affected resources、approval/gate 和 rollback 说明。未知或缺失证据不能降级成 warn 放行。

项目配置只引用 `secret_ref`，不写明文密钥。

v1.0 前 AreaFlow 只计算 secret / engine readiness，不解析明文、不注入 worker、不调用需要 secret 的
真实 engine。`secret_ref = none` 表示该 engine/profile 不需要密钥；其他引用在 secret store 能力打开前
只能返回 `unavailable` 或 `policy_denied`。真实 secret resolve、token rotation、external secret
manager 和远程 worker credential 属于 v1.x R4 能力。

## 并发与幂等

所有 command API 应支持：

```text
idempotency_key
requested_by_actor_id
request_hash
```

唯一约束：

```text
(actor_id, command_type, idempotency_key)
```

关键状态表建议加入：

```text
version integer not null default 1
updated_at timestamptz
```

更新时使用：

```text
WHERE id = $id AND version = $expected_version
```

## v0.1 到 v1 的迁移口径

- v0.1 可以继续使用轻量 schema。
- v0.2/v0.3 引入 `gate_results`、snapshot diff 和 idempotency。
- v0.5 引入 `run_tasks`、`run_attempts`。
- v0.6 引入 `workers`、`leases`、`worker_heartbeats`。
- v0.9/v1.0 前稳定 adapter/profile/secret/integration/backup 边界。
