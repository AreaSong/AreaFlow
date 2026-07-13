# Command / Approval Contract

## 定位

本文定义 AreaFlow 的统一写入口合同。它补充
[`api-surface.md`](api-surface.md)、[`security-permissions.md`](security-permissions.md)、
[`data-model-v1.md`](data-model-v1.md) 和 [`worker-scheduling-contract.md`](worker-scheduling-contract.md)。

目标是确保 CLI、Web、Desktop、Worker 和未来 plugin 都不能绕过同一套 permission、gate、approval、
audit 和 rollback 语义。Query、preview、readiness 和 gate 可以解释状态；只有受保护 Command API
能改变业务事实或执行 apply。

## 核心不变量

- Query API、SSE、summary、preview、readiness、doctor 和 gate 默认只读。
- 所有业务写动作必须进入 `command_requests`，并具备 idempotency key 和 request hash。
- Command API 是 CLI、Web、Desktop、Worker、shim 和未来 plugin 的唯一业务写入口。
- Approval record 不是 execution，也不是万能许可；它只批准明确 scope、risk、capability 和资源范围。
- Permission pass、gate pass、approval approved 和 command accepted 必须分开记录。
- Capability 是 resource-scoped，不是全局布尔许可；同一 capability 在不同 path、command、secret、
  network、git 或 artifact backend 上必须重新判断。
- API endpoint 存在不代表对应 apply rung 已打开；command accepted 不代表 apply 已发生。
- R2-R4 操作必须有 rollback 或 remediation plan；没有回退说明不能 apply。
- Audit event 必须记录 allowed / denied 的 actor、capability、resource、reason 和 evidence。
- Admin / maintenance command 不能绕过业务 command 去改变 workflow、run、lease、artifact 或 project files。

## 请求生命周期

标准写动作生命周期：

```text
preview / readiness / gate
-> command request
-> idempotency check
-> risk classification
-> permission preflight
-> gate snapshot check
-> approval check
-> expected_version / expected-before check
-> apply
-> event
-> audit_event
-> command response
```

任何步骤失败都必须返回 blocked / denied / failed 的结构化原因。失败不能被压成“未执行”或“按钮不可用”。
如果 apply 已产生外部影响但 event / audit event 写入失败，该 command 不能返回 success；必须进入
failed / suspended，并暴露 remediation 或 rollback path。

## Command Request

Command request 的最小合同：

```text
project_id
actor_id or actor key
command_type
command_class
idempotency_key
request_hash
risk_level
risk_policy
permission_preview
approval_state
expected_version nullable
precondition_snapshot
affected_resources
forbidden_resources
rollback_or_remediation_ref
safety_facts
response
completed_at
```

同一 `project_id + command_type + idempotency_key`：

- request hash 相同：返回同一业务结果。
- request hash 不同：拒绝为 idempotency conflict。
- 已完成 command 的 response 不能被重写成另一种事实。

未传显式 idempotency key 的周期性/观测型 command 可以由服务端生成 key，但它仍必须写入
`command_requests`，并保留可审计 response。

## Command Classification

Command 必须按实际副作用分类，不能只按 endpoint 名称分类：

```text
record_only:
  只写 AreaFlow PostgreSQL 中的 command/event/audit/metadata 事实，不触碰被管理项目或 artifact bytes。

projection_write:
  写轻量 projection，例如 `.areaflow/status.json`，必须有 path allowlist 和 R1 audit。

artifact_write:
  写 AreaFlow-owned local artifact store 或 artifact metadata，不写被管理项目文件。

managed_project_write:
  写被管理项目 allowlist 内路径，必须有 write-set、expected-before、preimage 和 rollback/remediation。

execution_control:
  创建 run/task/lease/attempt 或控制 worker lifecycle；不能自动获得 project write、secret 或 engine 权限。

external_effect:
  运行 shell、git、network、secret resolve、engine、remote worker、restore、publish 或 object upload/delete。

migration_security:
  DB migration、auth/team/token/secret、restore apply、release exception real write 和 publish apply。
```

Command class 可以升级风险，不能降低风险。只要一个 command 同时包含多类副作用，就按最高风险和最严格
gate / approval / audit 处理。拆分 command 优先于把多个副作用塞进一个 command。

## Capability Resource Matrix

Capability 必须绑定 resource type 和具体范围：

| Capability | Resource Scope | 不代表 |
|---|---|---|
| `read_project` | allowlisted project path / metadata source | 读取 secret、读取任意 root 外路径、读取 artifact 原文 |
| `write_status` | `.areaflow/status.json` 或明确 projection path | 写 workflow、generated、source、execution、progress |
| `write_artifacts` | AreaFlow-owned artifact store / artifact metadata | 写被管理项目文件或删除 artifact 原文 |
| `write_workflow` | AreaFlow-owned workflow metadata 或已授权 workflow projection | 写 execution、progress、logs、checkpoint |
| `write_generated` | allowlisted generated/projection prefix | 写源码、workflow execution、任意 `workflow/**` |
| `write_code` | allowlisted source path 和 operation | delete/move/chmod/binary/symlink/glob/root 外路径 |
| `run_commands` | allowlisted command argv / cwd / env | 任意 shell、git、task-loop、network 或 engine |
| `manage_workers` | worker registry / heartbeat / scoped lease | 远程 credential、secret、project write、engine execution |
| `manage_git` | scoped checkpoint / tag operation | `git reset --hard`、unchecked branch overwrite、publish |
| `network` | allowlisted host / method / purpose | secret exfiltration、plugin remote fetch、unknown API calls |
| `use_secrets` | short-lived scoped secret binding | 明文落盘、跨 project/run 复用、worker 长期持有 |
| `execute_agents` | approved engine profile / command / budget | secret-backed engine、unbounded spend、unscoped shell |

Project config 的 capability 是上限，不是最终许可。最终许可必须由 command request 的 affected resources、
deny list、gate snapshot、approval scope、expected version/hash 和 rollback/remediation 一起决定。
Budget / quota 相关 approval、estimate、reservation、charge 和 override 语义见
[`budget-quota-boundary.md`](budget-quota-boundary.md)；budget pass 不能替代 command approval，reservation
也不能替代 human approval。

## Risk And Permission Order

风险等级决定 gate 和 approval 强度：

| Risk | Examples | Minimum Requirement |
|---|---|---|
| R0 | metadata import、doctor、preview、readiness | Query 或 command 均可，但不得写项目文件。 |
| R1 | `.areaflow/status.json` projection | Command API、capability、path allowlist、audit。 |
| R2 | generated/workflow/export managed write | Command API、permission、gate、approval、rollback。 |
| R3 | execution、worker task、source write、checkpoint、repair | R2 + execution approval、attempt evidence、focused smoke。 |
| R4 | migration、secret、remote worker、restore/publish apply | R3 + migration/security approval、revoke/rollback path。 |

R1 projection write 也必须先能生成只读 authorization preview、只读 apply packet preview 和只读 apply gate。Authorization preview
负责列出 target、schema URI、validator preflight、protected path check、write-set、expected-before
preimage、rollback plan、permission 状态和 safety facts。Apply packet preview 负责从当前
authorization/preimage 生成机器可审查 packet、CLI apply command 和 API request。Apply gate 负责校验提交
packet 是否与当前 authorization/preimage 一致，至少包括 source snapshot hash、expected-before
exists/hash/size、schema URI、validator preflight、protected path check、accepted preimage schema status、
rollback action 和显式 approval actor/reason。preview、packet preview 和 gate 都不是 command request，
不创建 audit/event，不写项目文件；
`apply_command_eligible=true` 只表示 packet 可以进入受保护 command。真实写入仍只能由
`project.status_projection.apply` command 产生。该 command 必须重新消费同一 packet 并记录
`apply_gate_status`、`apply_gate_decision`、`apply_gate_approval_status` 和 `apply_command_eligible`；
gate 不通过时 command 可以记录 blocked/denied command、event、audit 和 metadata，但必须保持
`project_write_attempted=false`，不得调用 writer。

Permission 判断顺序：

```text
project scope match
capability allow
path / command / secret / network / git resource allow
deny override check
gate result check
approval scope check
expected_version or expected-before hash check
rollback / remediation availability
```

Deny 永远优先于 allow。Profile 可以声明 gate 和 required artifacts，但不能授予 capability。Adapter 和
worker 也不能扩大 permission scope。

## Approval Contract

Approval record 必须绑定明确事实：

```text
actor
approval_kind
decision
scope_type
scope_id
risk_level
allowed_capabilities
allowed_resources
forbidden_resources
gate_snapshot
transition_preview_id nullable
expires_at nullable
rollback_or_remediation_ref
human_reason
audit_event_id
```

Approval 只能批准它声明的 scope。以下情况必须视为无效或 `needs_approval`：

- gate snapshot 已过期或 source hash 变化。
- command 请求的 capability 超出 approval allowed_capabilities。
- target path / command / secret / network 超出 allowed_resources。
- approval 已过期。
- rollback / remediation reference 缺失。
- risk level 高于 approval 记录的 risk level。
- project config、permission row、profile hash、adapter hash 或 workflow version 发生影响 scope 的变化。
- idempotency request hash、affected resources 或 expected version/hash 与 approval snapshot 不一致。
- command class 从 record/projection/artifact 升级到 project write、execution 或 external effect。
- actor、team、worker、engine profile 或 secret binding 超出 approval 记录的 actor/resource 范围。

Approval 不代表 apply 已发生。Apply 必须由后续 command 独立执行，并写 event / audit / evidence。

## Precondition And Write-set Contract

R2-R4 command 在 apply 前必须冻结 precondition snapshot：

```text
project_key
project_config_hash
permission_policy_hash
adapter_id / adapter_hash
profile_id / profile_hash
workflow_version_id / workflow_version_status
gate_result_ids
approval_record_ids
target resources and expected_version/hash
deny list version
```

涉及被管理项目写入时，command 还必须提供 write-set：

```text
operation: create | modify | delete | move | chmod | binary_write | symlink_write
target_path
resource_capability
expected_before_sha256 nullable
expected_before_size nullable
preimage_artifact_id nullable
post_apply_verify
rollback_or_remediation_ref
non_target_fingerprint_policy
```

第一版真实 source write 只能支持 allowlist 内 text `create` / `modify`。Delete、move、chmod、binary rewrite、
symlink target、glob 批量写入和 project-root 外路径必须保持 blocked，直到对应高风险 rung 单独打开。
对已有文件的修改必须校验 expected-before hash/size；校验不通过时不能 apply。

## Preview / Gate / Apply 边界

以下动作只能解释状态，不能 apply：

```text
summary
doctor
readiness
transition preview
promotion preview
execution approval gate
generated write readiness
generated write apply beta gate
managed generated write gate
worker pool schedule preview
release readiness / final gate / publish gate
restore dry-run plan
```

这些动作不得创建 lease、attempt、artifact、project write、engine call、secret resolve、network call 或
被管理项目文件写入，除非对应文档明确声明它们是受保护 command 且已打开 apply。

## Apply Response Safety Facts

所有 R1-R4 command response 必须包含 safety facts：

```text
project_read_attempted
project_read_allowed
project_write_attempted
project_write_allowed
execution_write_attempted
area_flow_artifact_written
area_flow_execution_state_written
engine_call_attempted
commands_run
secrets_resolved
network_used
task_claimed
worker_started
lease_created
attempt_created
artifact_created
rollback_available
rollback_verified nullable
```

未涉及的字段也应显式返回 `false` 或省略规则明确。高风险路径不能只返回 `ok=true`。

## Multi-client Boundary

客户端职责：

```text
CLI:
  构造 command，显示 preview/gate/response，不直接写 DB。

Web:
  先只读展示；写按钮必须调用 Command API，并显示 blockers 和 safety facts。

Desktop:
  控制 local service / dashboard；不维护第二状态源，不直接执行 workflow。

Worker:
  只在 scoped lease 内提交 attempt / artifact / command response。

Compatibility shim:
  只做薄转发和 blocked message，不维护第二套 workflow 状态。

Plugin:
  只能通过受治理 command / adapter / profile / engine 接口工作。
```

任何 client 发现 API 不可用时，只能降级到只读 projection 或 unavailable message，不能恢复旧 workflow
写路径。

## AreaMatrix First Policy

AreaMatrix dogfood 的写入口更保守：

- Authoring cutover 只切新 workflow version 的 authoring ownership，不切 execution。
- `./task-loop run` forwarding 在 execution cutover approval 前保持 blocked。
- `workflow/versions/**/execution/**`、`progress.json`、legacy logs 和 checkpoint 是 protected path。
- `.areaflow/status.json` 属于 R1 projection；`workflow/README.md` 受控区块必须等 cutover gate。
- Generated-only apply、source write、checkpoint、repair、engine execution、secret resolve 和 remote worker
  都必须按各自 gate 单独开闸。

## 打开条件

任何新 apply command 从 closed 进入 open，至少需要：

```text
architecture contract updated
API / CLI command defined
idempotency conflict test
permission denial test
gate blocked test
approval missing / expired / scope mismatch test
expected_version or expected-before mismatch test
event / audit evidence
safety facts
rollback or remediation plan
focused smoke
AreaMatrix protected path proof when relevant
implementation gap audit updated
```

缺少这些证据时，只能标记为 `preview_only`、`readiness_only` 或 `implemented_scoped`。
