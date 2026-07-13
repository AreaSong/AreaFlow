# Data Model v0.1

## 原则

- PostgreSQL 是主状态源事实。
- `events` 与 `audit_events` append-only。
- artifact 原文不直接进入数据库。
- import 需要幂等。
- 历史导入默认 immutable。
- 表存在不等于能力已打开；v0.1 只能使用 import / mirror 范围内的语义。
- `project_key` 是所有 project-scoped 数据的隔离边界。

## Schema Layers

当前开发数据库可能已经应用 v0.2-v1.0 的后续 migration。v0.1 closure 只允许依赖下列语义层，
不能因为后续表存在就打开 worker、engine、secret、cutover 或 publish。

### Core Import Tables

这些表来自 v0.1 core schema，是 Import + Status Mirror 的最小状态层：

```text
actors
projects
project_connections
project_permissions
workflow_versions
workflow_items
residuals
artifacts
project_status_snapshots
runs
events
audit_events
```

### Early Support Tables

这些表可以被当前实现提前用于幂等、配置快照、projection 历史或调度 metadata，但在 v0.1 只能保存
metadata，不代表对应后续能力已打开：

```text
command_requests:
  只允许记录 project.import、project.status_projection.apply 这类 v0.1 command 的 idempotency、
  request_hash 和 response。不代表通用 Command API 已对 workflow write、execution、restore 或 publish
  开放。

project_configs:
  保存 `areaflow.yaml` 的 normalized snapshot、config_hash、ownership、permissions、scheduling、
  engines、status_export 和 migration。v0.1 只读取和归档配置，不根据配置启动 worker 或解析 secret。

status_projections:
  保存 `.areaflow/status.json` projection 的 payload、source_hash、target_uri 和 write_state。projection
  不是主状态源，不能承载完整 queue、execution、approval、secret、logs 或 artifact 原文。

project_scheduling_policies:
  只保存从 config 派生的 priority、max_parallel_tasks、agent_role、required_capabilities 和
  engine_profile metadata。v0.1 不做 schedule preview，不创建 lease，不启动 worker。
```

### Tables That Must Stay Inactive In v0.1

这些表即使已经存在，也不得由 v0.1 import / mirror 创建业务行或作为完成证据：

```text
run_tasks
run_attempts
workers
worker_heartbeats
leases
gate_results
approvals
transition_previews
engine_runs
secret_refs
users
teams
memberships
api_tokens
artifact_locations object backend rows
release exception apply tables
```

若后续 smoke 为了更高阶段创建了这些行，completion audit 必须按对应阶段证据解释，不能倒灌成
v0.1 能力。

## 关键字段

### projects

```text
project_key
name
kind
adapter
workflow_profile
default_branch
```

语义：

- `project_key` 必须稳定、唯一、不可从 display name 推导。
- `adapter` 和 `workflow_profile` 只是绑定引用；v0.1 不加载未知 plugin。
- `archived_at` 不用于 v0.1 archive cutover，只保留长期状态字段。

### project_connections

```text
project_id
connection_type
root_path
remote_url
current_branch
current_commit
```

v0.1 使用：

- `local_path`：被管理项目 root。
- `artifact_store`：AreaFlow-owned artifact store root / backend。

`project_connections` 不表达 remote worker、team workspace 或 deployment target。

### project_permissions

```text
project_id
capability
effect
resource_type
pattern
```

v0.1 必须遵循 deny-first：

- capability allow 只是上限。
- path allow 必须同时存在。
- forbidden path 命中时直接 deny。
- `write_artifacts` 不允许写被管理项目文件。
- `run_commands=false` 时禁止执行任何 project command，包括 allowlist 中的命令。

### workflow_versions

```text
project_id
display_label
version_kind
lifecycle_status
source_path
source_hash
import_mode
immutable
status_summary jsonb
```

v0.1 语义：

- `display_label` 保留 AreaMatrix 原标签，例如 `v1-mvp`。
- `version_kind` 可表达 `workflow_version`、`template` 或历史导入类型。
- `lifecycle_status` 是粗状态，不塞入 gate / approval / worker runtime。
- `import_mode=metadata_only` 表示只导入索引、hash、path 和 summary。
- `immutable=true` 的历史导入不得被回写修正；只能追加 event / audit / artifact metadata。

### workflow_items

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
metadata jsonb
immutable
```

v0.1 语义：

- `workflow_item` 是 stage 内语义条目，不等于文件，也不等于 worker task。
- AreaMatrix 的 discussion、middle-layer、changes、plans、drafts、queue、execution metadata 都映射成
  item / artifact metadata，不硬编码成 core 表。
- `external_key` 必须稳定，可由 adapter/profile 映射。

### residuals

```text
project_id
workflow_version_id nullable
residual_key
status
type
title
source_path
current_impact
executable_task
promotion_required
close_condition
metadata jsonb
immutable
```

v0.1 语义：

- residual 只做索引和状态解释，不自动转成 task。
- `executable_task=true` 也不代表可执行；必须等后续 promotion / approval / execution gate。
- `promotion_required` 只说明未来进入执行层前需要 promotion，不代表已 promotion。

### artifacts

```text
project_id
workflow_version_id nullable
run_id nullable
workflow_item_id nullable
artifact_type
storage_backend
uri
source_path
sha256
size_bytes
content_type
metadata jsonb
```

v0.1 storage backend 只能按以下语义解释：

```text
local:
  AreaFlow-owned artifact store 内容。

project_reference:
  原文仍留在被管理项目中，AreaFlow 只保存 metadata/hash/path。

external_project:
  AreaFlow 只知道外部路径或 URI，不能声明可恢复。
```

`project_reference` 和 `external_project` 不能被 restore dry-run 计为完整可恢复内容，也不能被 archive / GC
触碰。

### project_status_snapshots

```text
project_id
snapshot_kind
summary jsonb
source_hash
export_path
created_by_actor_id
```

v0.1 snapshot kind：

```text
import:
  只读导入快照。

mirror_export:
  status projection / export-status 快照。
```

snapshot 是 projection 证据，不是 workflow 主状态源。

### runs

```text
project_id
run_type
status
started_at
finished_at
created_by_actor_id
summary jsonb
```

v0.1 只允许：

```text
run_type=import
status=running|succeeded|failed
```

后续实现可能用 command metadata 记录 status projection apply，但 v0.1 的 `runs` 不能表达：

```text
runner_preview
worker_execution
read_only_verify
approved_artifact_write
fixture_project_write
managed_generated_write
repair
checkpoint
```

任何 v0.1 `run` 都不能被 worker lease 领取，也不能关联 `run_task` / `run_attempt`。

### events

```text
project_id
run_id
workflow_version_id
event_type
severity
message
metadata jsonb
actor_id
```

v0.1 event 只记录 import、projection、config registration 和 audit-adjacent timeline。它不能作为
approval、worker heartbeat、secret resolve 或 publish 事实。

### audit_events

```text
project_id
actor_id
action
capability
resource_type
resource
decision
reason
metadata jsonb
```

v0.1 audit 必须能证明：

- project registration 发生过。
- import 是 read-only。
- status projection apply 是否 allowed / denied。
- forbidden path、capability missing 或 unsupported target 被拒绝。
- 没有 workflow / execution / engine / secret / git apply。

## v0.1 不建的表

```text
teams
team_members
api_tokens
workers
leases
heartbeats
attempts
engine_profiles
engine_runs
secret_refs
annotations
```

这些表在后续阶段引入或预留。即使当前开发数据库已经存在其中一部分，v0.1 仍必须把它们视为 inactive。

## `areaflow.yaml` Persistence

`project add --config <path>` 必须持久化三类事实：

```text
projects:
  project id、name、adapter、workflow_profile、default_branch。

project_connections:
  local_path 和 artifact_store。

project_permissions:
  capabilities、read paths、write paths、forbidden paths 和 command-derived denies。
```

如果 `project_configs` 表存在，还必须保存 normalized config snapshot：

```text
protocol_version
config_path
config_hash
ownership
permissions
scheduling
engines
status_export
migration
metadata.project
metadata.artifact_store
metadata.commands
active=true
```

每次重新 `project add` 同一 `project_key` 时，应保留历史 config snapshot，并只把最新一条标记为 active。

## Idempotency And Mutation Boundary

v0.1 幂等边界：

- 重复 `project add` 更新 project metadata、connections、permissions 和 active config snapshot。
- 重复 `project import` 可以创建新的 import snapshot；如果显式传入同一 idempotency key 且 request hash
  相同，应返回同一 command response。
- `project import` 可以刷新当前 metadata index，但不得修改 AreaMatrix 原文件。
- `status-projection-apply` 可以写 `.areaflow/status.json`，但必须记录 source_hash、target_uri、write_state
  和 audit。
- `export-status` 只是 compatibility alias，长期主语义是 `status-projection-apply`。

v0.1 禁止：

- 通过 DB 行直接推进 workflow lifecycle。
- 修改 immutable historical import。
- 创建 worker lease、attempt 或 task execution row。
- 把 config 中的 `enabled=true`、`run_commands` 或 engine profile 解释成可执行授权。
