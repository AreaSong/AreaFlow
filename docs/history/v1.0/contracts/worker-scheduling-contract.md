# Worker Scheduling Contract

## 定位

本文定义 AreaFlow 多项目、多 worker 的调度合同。它补充
[`execution-model.md`](execution-model.md)、[`data-model-v1.md`](data-model-v1.md) 和
[`security-permissions.md`](../history/v1.0/contracts/security-permissions.md)，目标是让 runner、worker、API、Web、Desktop
和未来 scheduler 对同一套 lease / concurrency / recovery 语义达成一致。

本文不是自动 scheduler 授权。v0.8 的 worker pool summary 和 schedule preview 仍然只读，不创建 lease、
不启动 worker、不写 event/audit、不调用 engine、不读取 secret，也不写被管理项目。
v0.8 当前只打开的 summary / schedule preview / readiness 边界见
[`v0.8-multi-project-worker-pool-contract.md`](../history/v1.0/contracts/v0.8-multi-project-worker-pool-contract.md)。

## 核心不变量

- PostgreSQL 是 worker scheduling 的主状态源事实；v1.0 前不引入 Redis / NATS 等第二队列。
- `project_key` / `project_id` 是 run、run_task、worker、lease、artifact、secret 和 audit 的隔离边界。
- Worker 只能领取 `run_task`，不能直接领取 `workflow_item`。
- 同一个 `run_task` 同时最多一个 `active` lease；数据库通过 partial unique index 兜底。
- Lease 过期不等于 task failed，只能进入 `needs_recovery`。
- Capability denial 不得创建 lease、attempt 或 artifact。
- Schedule preview 不得复用真实 acquire path，也不得产生 side effect。
- AreaMatrix 第一阶段真实 execution 并发固定为 1；放宽必须单独 gate、approval 和 evidence。

## 调度输入

调度器或 preview 至少读取以下事实：

```text
projects
project_scheduling_policies
runs
run_tasks
workers
leases
approval_records
gate_results
project_permissions
engine_profiles
secret_refs
events / audit_events
```

`project_scheduling_policies` 的最小输入：

```text
priority
max_parallel_tasks
agent_role
required_capabilities
engine_profile
metadata
```

`max_parallel_tasks` 默认是 1。它是项目级并发上限，不代表 worker 可以越过 approval、gate、
permission 或 risk policy。

## 候选任务选择

真实 acquire lease 时，候选 `run_task` 必须同时满足：

```text
same project scope
run is runnable for the requested mode
task status in queued / pending / needs_recovery
no active lease for the same run_task
required gate and approval pass
worker is online
worker type matches agent_role
worker capabilities cover required_capabilities
project permissions allow required capabilities
engine readiness is acceptable for the mode
secret readiness is acceptable for the mode
project and worker slots are available
```

选择候选时使用 PostgreSQL row lock：

```sql
SELECT ...
FOR UPDATE SKIP LOCKED
```

排序应稳定、可解释：

```text
project priority desc
run/task priority desc
created_at asc
id asc
```

如果某个能力还处于 preview / readiness 阶段，候选选择只能返回 `recommended=false` 或
`next_action=worker_run_once_preview`，不能创建真实 lease。

## Slot 计算

Preview 和真实 scheduler 必须使用同一套 slot 语义。区别是 preview 只解释，不写入。

```text
worker_slots = online_matching_workers - active_leases_for_matching_workers
project_slots = max_parallel_tasks - active_leases_for_project
available_slots = min(worker_slots, project_slots)
```

下限为 0。任何 slot 为 0 时都必须返回 blocker：

```text
no_online_workers
missing_agent_role:<role>
missing_required_capability:<capability>
no_available_worker_slots
project_parallel_limit_reached
resource_limit:<name>
engine_profile_disabled
secret_ref_unavailable
```

高风险任务默认串行。即使 `max_parallel_tasks > 1`，R3/R4 任务也必须由 approval 明确允许并发，并记录
risk policy snapshot。

## Lease 生命周期

当前稳定 lease status：

```text
active
completed
released
needs_recovery
```

状态含义：

| Status | Meaning |
|---|---|
| `active` | Worker 已领取 scoped lease，TTL 尚未由 recovery command 判定过期。 |
| `completed` | Worker 完成该 lease 范围，并提交了对应 attempt / artifact / event / audit。 |
| `released` | Worker 或 operator 释放 lease，不声明执行成功。 |
| `needs_recovery` | Lease 过期或 recovery command 接管，后续需要重新判断 task 状态。 |

`heartbeat_at` 是 worker 活性证据，`expires_at` 是 lease recovery 的时间边界。Heartbeat timeout 不得直接把
task 标记为 failed。

## Recovery

Recovery 是显式 command，不是后台自动猜测。

```text
active lease expired
-> lease.recover command
-> lease.status = needs_recovery
-> run_task.status = needs_recovery
-> event / audit / command_response
```

Recovery command 不创建新的 attempt、artifact 或 worker run。重新执行必须走新的 acquire / worker command，
并保留旧 lease、旧 attempt 和旧 artifact。

如果 recovery 时发现已有 verify-pass evidence 或 checkpoint evidence，不能直接覆盖 task 状态；必须创建
gate result 或 recovery report，让 runner 决定是 `passed`、`repair_needed`、`blocked` 还是重新排队。

## Preview 与真实 Scheduler

v0.8 summary 和 schedule preview 只允许：

```text
read projects / workers / leases / run_tasks / policies
compute recommended / blocked
compute available_slots
explain blocked_reasons
return next_action
```

v0.8 preview 禁止：

```text
create command_requests
create leases
update run_tasks
start workers
call engine
resolve secrets
write artifacts
write events / audit_events
write managed project files
```

真实 scheduler 只有在后续单独打开 command 后才能做：

```text
schedule.acquire
schedule.dispatch
schedule.recover
schedule.drain
```

这些 command 仍必须满足 idempotency key、request hash、permission preflight、approval/risk policy、
audit event 和 rollback/remediation 说明。

## Worker 类型

Worker type 先按平台能力表达，不按部署幻想提前复杂化：

```text
local_worker
local_host
host_bound_worker
remote_worker
container_worker
```

AreaMatrix 这种需要 macOS app、Xcode 或本机 GUI 能力的项目，默认使用 `local_host` 或
`host_bound_worker`。Container 不是默认要求。

`remote_worker` 在 v1.0 前只作为 readiness / preview / blocked reason。真实 remote worker credential
属于 R4，必须按 [`auth-team-secret-boundary.md`](../../proposals/auth-team-secret.md) 另行打开；remote worker
不得直连 PostgreSQL，只能通过 API 注册、heartbeat、lease、attempt 和 artifact upload。远程 worker
凭证必须至少绑定以下 scope：

```text
project_id / project_key
worker_key
worker_kind
allowed_capabilities
lease_id nullable
command_request_id nullable
expires_at
rotation_generation
revoked_at nullable
```

远程 worker 不持有长期 secret。需要 engine 或 provider secret 时，只能请求
[`auth-team-secret-boundary.md`](../../proposals/auth-team-secret.md) 定义的 short-lived scoped binding；该
binding 不能进入 worker 持久状态、stdout/stderr、artifact、event、audit 或 backup。

## AreaMatrix First Policy

AreaMatrix dogfood 的真实 execution 比 fixture 更保守：

- 第一阶段真实 AreaMatrix execution 并发固定为 1。
- `./task-loop run` forwarding 在 execution cutover approval 前保持 blocked。
- `workflow/versions/**/execution/**`、`progress.json`、legacy logs 和 checkpoint 仍是 protected path。
- 真实 generated-only apply 先做 rollback beta，再做 retained apply。
- source write beta、checkpoint apply、repair apply 和 engine execution 必须单独 gate。

## 关闭条件

真实 scheduler 从 closed 进入 open 之前，至少需要：

```text
schedule command API
project-scoped acquire tests
concurrent lease race tests
capability denial tests
project parallel limit tests
AreaMatrix max_parallel_tasks=1 evidence
recovery evidence
event / audit evidence
no engine / no secret / no project write safety facts
rollback or remediation plan for R3/R4 tasks
```

没有这些证据时，只能标记为 `preview_only` 或 `readiness_only`。
