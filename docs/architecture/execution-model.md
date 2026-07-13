# Execution Model

## 定位

Execution model 定义 AreaFlow 如何从 workflow item 进入可审计执行。它最终替代
AreaMatrix `./task-loop` 的主能力，但不把 `progress.json`、本地 lock 或日志目录原样搬进数据库。

## 核心关系

```text
workflow_item = 要做什么
run = 一次执行会话
run_task = run 内的一个执行单元
attempt = 一次真实尝试
artifact = 输入/输出证据
gate_result = 能不能前进
event/audit_event = 发生过什么、谁允许了什么
worker/lease = 谁正在跑
```

Queue candidate 与 execution run 是一对多关系。`workflow_item` 或 `queue_candidate` 表达“要做什么”，
`run` 表达一次用户可见执行会话；同一个 queue candidate 可以因为失败重跑、repair、重新 verify、
不同 worker 策略或人工 retry 产生多个 run。

```text
queue_candidate / workflow_item
  -> run
    -> run_task
      -> run_attempt copy
      -> run_attempt verify
      -> run_attempt repair
      -> run_attempt checkpoint
```

旧 run、run_task 和 run_attempt 不覆盖、不删除。修正执行结果只能追加新的 run/attempt、event、
audit_event、projection 或 closeout amendment。

## 分层职责

AreaFlow 替代 `./task-loop` 时不把编排、执行、AI 调用和项目语义塞进同一层。

```text
Runner:
  创建 run，拆分 run_task，检查 gate / approval / permission，汇总结果。

Worker:
  领取 scoped lease，执行一个 run_task，提交 heartbeat、attempt、artifact 和 audit。

Engine Adapter:
  封装 Codex CLI、OpenAI API、local model 或其他 agent/provider 的调用方式。

Project Adapter:
  理解被管理项目的目录、doctor、status、验证命令和项目语义。
```

Runner 不直接执行 shell 或写项目文件；Worker 不决定任务是否允许执行；Engine Adapter 不理解
AreaMatrix 业务；Project Adapter 不绕过 permission evaluator、approval 或 audit。

## Run

Run 是用户看到的一次执行。例如“执行 AreaMatrix v2 phase-1 队列”就是一个 run。

```text
run_kind:
  import
  doctor
  promotion
  execution
  verify
  repair
  closeout

run_status:
  queued
  running
  drained
  passed
  failed
  blocked
  cancelled
  needs_recovery
```

Run 记录范围、启动人、风险策略、worker、最终状态和统计摘要。

## Run Task 状态机

`run_task` 是 worker 可以领取的最小执行单元。`workflow_item` 仍是语义交付物，不能被 worker 直接领取。

`run_task.status` 必须保持通用、少量和跨项目稳定。它表达 worker 生命周期和是否还能继续调度，
不表达某个能力的业务结果。fixture execution、read-only verify、artifact write、rollback drill、
generated-only apply 等差异放入 `run_task.outcome`、`run_task.metadata`、`run_attempts`、artifact、
gate result 或 run summary。

目标状态枚举：

```text
queued
pending
-> leased
-> preflight_running
-> running
-> verifying
-> checkpoint_ready
-> passed
```

失败和控制路径：

```text
blocked
failed
repair_needed
retry_scheduled
cancel_requested
cancelled
lease_expired
needs_recovery
superseded
```

推荐的 outcome 示例：

```text
fixture_execution_passed
read_only_verify_passed
artifact_write_passed
rollback_verified
generated_write_rollback_verified
checkpoint_passed
repair_verified
```

这些 outcome 不是 `status` 新枚举。API / Web 可以展示 outcome，但调度、lease、retry 和 recovery
只能依赖通用 `run_task.status`、lease state、attempt evidence 和 gate result。

关键规则：

- `leased` 必须有 TTL 和 heartbeat；worker 失联后先进入 recovery 判断，不直接判失败。
- attempt 不可变；重试创建新的 `run_attempt`，不能覆盖旧 attempt。
- `passed` 必须有 verify / acceptance evidence；命令退出码为 0 不足以证明完成。
- `cancel_requested` 是协作式停止请求；已经产生的 artifact、event 和 audit 不删除。
- `repair_needed` 只能由 verify / gate evidence 触发，不能由 worker 自行决定跳过验收。
- 早期 v0.6 scoped implementation 中已经出现的 `verified`、`artifact_written`、`rollback_verified`
  等能力专属 task status 视为兼容窗口内的 legacy/scoped status。后续不再新增同类 status；进入
  schema/API 整理时应迁移为 `status=passed` + outcome / attempt / artifact / gate evidence。

## Attempt

Attempt 是一次真实尝试。copy 一次、verify 一次、repair 一次、checkpoint 一次，都必须保留独立记录。

```text
attempt_kind:
  copy
  verify
  repair
  doctor
  checkpoint
```

Attempt 不覆盖历史。verify 失败后产生 failure summary，后续 repair/copy 使用该摘要作为输入。

Attempt 必须带上可复验输入和输出：

```text
attempt_input:
  command_request_id
  run_task_id
  attempt_kind
  precondition_snapshot
  write_set_artifact_id nullable
  verification_plan_artifact_id nullable
  approval_record_id nullable

attempt_output:
  status
  outcome
  produced_artifact_ids
  observed_hashes / sizes
  safety_facts
  event_id
  audit_event_id
```

状态推进规则：

- Copy attempt 成功只说明变更动作完成，不说明 task 完成。
- Verify attempt pass 只能推进到 acceptance candidate；仍需 checkpoint/acceptance gate 按 scope 判定。
- Verify attempt fail 必须产生 failure summary，并推进到 `repair_needed` 或 `failed`。
- Repair apply 成功必须触发新的 verify attempt，不能直接把 task 置为 `passed`。
- Checkpoint apply 是独立 attempt；checkpoint fail 必须阻断下一 task。
- Rollback attempt 只能对 AreaFlow 自己 apply 的 write-set 做受控恢复；current hash 不匹配时 blocked。

## Runner Preview

Runner preview 是 v0.5 的 dry-run 执行预演。它只证明 execution model 可以表达 run、
run_task、attempt、preflight、artifact、event 和 audit event，不执行项目写入、命令、secret
解析、网络访问或 AI engine 调用。
阶段合同见 [`v0.5-runner-preview-contract.md`](v0.5-runner-preview-contract.md)；该合同是判断
`runner.preview`、dry-run run control 和 v0.6 handoff 的源事实。

v0.5a preview 固定创建：

```text
run_type = runner_preview
run_kind = execution
run.status = passed
run.dry_run = true
run_task.task_kind = workflow_item_preview
run_task.status = queued
attempt_kind = copy
attempt_kind = verify
artifact_type = runner_preview_report
```

`runner_preview_report` 写入 local artifact store，PostgreSQL 只保存 artifact metadata、
URI、sha256、size、content type 和关联关系。run 级 artifact 可以没有 `workflow_item_id`；
读取方必须把空 item 视为 run-level evidence。

`runner.preview` 必须完成 command request response。Response 至少记录 run/task/attempt/artifact
ID、event ID、audit event ID、preflight status、artifact hash/size，以及
`project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、
`commands_run=false`、`secrets_resolved=false` 和 `network_used=false`。

Runner preview 的 `run.status = passed` 表示预演本身已完成；`run_task.status = queued`
表示该 dry-run task 可被 worker beta 领取。Worker `run-once` 完成后才把 task 更新为 `passed`。

## Copy / Verify 分离

AreaFlow 将 AreaMatrix 的 copy-ready / verify-ready 治理规则平台化：

- `copy` attempt 可以在授权路径内写入或生成 artifact，但必须有 approved write-set。
- `verify` attempt 语义只读，不能修复、checkpoint 或扩大验证范围。
- verify 不通过，不能 mark done。
- verify 失败生成 failure summary。
- 只有 `verify_acceptance` gate pass，workflow item 才能进入 `done`。
- `verify_acceptance` 只能接受 verify plan 覆盖过的文件、artifact 和命令；覆盖范围不足时必须
  `needs_attention`。

Process sandbox 不是唯一只读边界。只读性由 prompt、permission evaluator、attempt policy 和 gate 共同保证。

## Codex CLI Adapter Preview

v0.6 先提供受限 Codex CLI adapter preview，不打开真实 Codex CLI execution。该 preview 只读取
project config、permission rows 和 engine profile metadata，输出：

- engine/profile readiness。
- proposed command preview。
- capability preflight。
- read / forbidden path preflight。
- artifact redaction plan。

preview 必须保持 `engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、
`network_used=false`、`project_write_attempted=false` 和 `execution_write_attempted=false`。默认
AreaMatrix `codex-cli` profile disabled，且 `run_commands` / `execute_agents` capability disabled，
因此 preview 应返回 `blocked`。即使 engine、command、capability 和 path preflight 全部通过，真实
Codex CLI 调用仍需要 explicit execution approval、secret policy、budget policy、worker scope 和
artifact redaction policy 同时通过；否则状态只能进入 `needs_approval`，不能执行。
Budget / quota policy 的 estimate、reservation、charge 和 enforcement 语义见
[`budget-quota-boundary.md`](budget-quota-boundary.md)；v1.0 前 budget readiness 不能阻断真实 run，也不能
被解释为已打开 spend control。

## Execution Approval Gate

Execution approval gate 是真实 execution apply 前的只读 go/no-go 判断。它组合 run、run_task、
workflow approval、`approval_gate`、`live_mapping_gate`、engine adapter preview、worker registry 和
capability readiness。该 gate 不创建 command request、不领取 task、不启动 worker、不创建 attempt /
artifact、不解析 secret、不调用 engine、不运行 shell、不写被管理项目。

最小 pass 条件：

- run 是 `run_kind=execution`。
- run 是非 dry-run，且 `status=queued`。
- workflow version 是 AreaFlow-authored。
- 至少一个 run_task 是 `queued` 或 `pending`。
- 最新 approval record 是 `approved`。
- `approval_gate` 和 `live_mapping_gate` 均为 `pass`。
- Codex CLI adapter preview 不是 `blocked`。
- 至少一个 online worker 满足执行所需 capabilities。

当前 runner preview 产生的 dry-run run 必须被该 gate 阻断。通过该 gate 只代表“可以进入后续受保护
execution apply 设计”，不代表已经执行 copy/verify、启动 worker 或写项目文件。

受限 apply mode 可以进一步收窄 gate 要求：

- `fixture_execution` 用 fixture-only apply 证明 lease / attempt / artifact / run 状态闭环，不代表真实
  engine execution。
- `read_only_verify` 在 gate 通过后只读取 allowlisted target file，并只保存 path、sha256 和 size
  evidence。
- `approved_artifact_write` 在 gate 通过后只写 AreaFlow-owned artifact store 和 PG metadata/evidence；
  它要求 worker 与 project 都具备 `write_artifacts`，但不读取项目文件、不调用 engine、不运行 shell。

这些受限 mode 仍必须记录 `command_requests`、events、audit events 和 safety facts。它们推进受限
run/task 状态，不等于打开真实 copy/verify/repair/checkpoint 或项目文件写入。

## Execution Plan Preview

Execution plan preview 是真实 copy / verify / repair / checkpoint 打开前的只读执行计划视图。它组合
run detail 和 execution approval gate，告诉 CLI、API、Web 和 Desktop：

```text
execution_approval_gate
copy
verify
approved_artifact_write
checkpoint
repair
```

每个 step 必须暴露 status、required capabilities、prerequisites、blockers 和安全属性：

```text
reads_project
writes_project
writes_areaflow
uses_engine
runs_commands
uses_secrets
uses_network
creates_attempt
creates_artifact
```

当前 `approved_artifact_write` 是唯一已经打开的 artifact-store-only execution step；即使 execution
approval gate pass，`copy`、`checkpoint` 和 `repair` 仍必须保持 blocked / waiting，并明确暴露
`managed_project_write_not_open`、`engine_execution_not_open`、`checkpoint_apply_not_implemented`
等 blocker。

Execution plan preview 不创建 `command_requests`，不领取 task，不启动 worker，不创建 lease / attempt /
artifact，不读取或写入被管理项目，不调用 engine，不运行 shell，不解析 secret，不访问网络，也不写
`workflow/versions/**/execution/**`。它只证明下一步真实 execution 打开顺序可解释，不代表已经打开
copy、repair、checkpoint、engine execution 或项目文件写入。

## Approved Project Write Design Gate

Approved project write 是 AreaFlow 从 artifact-only execution 走向真实项目写入的第一条高风险边界。
它必须先通过设计门禁，再进入任何 apply 实现。设计门禁本身只产出只读 contract、planned command、
write-set schema、rollback policy 和 smoke 计划；它不写被管理项目、不运行 engine、不领取 task。

最小打开顺序：

```text
project write design gate
-> fixture approved project write
-> fixture verify
-> fixture rollback drill
-> managed-project generated-only rollback beta
-> managed-project generated-only retained apply
-> manual patch artifact
-> human-applied source evidence
-> controlled source write beta
-> checkpoint preview
-> checkpoint apply
-> repair plan
-> repair apply
```

其中 `managed-project generated-only write` 指只写项目显式允许的生成/投影路径，例如
`.areaflow/generated/**`、`.areamatrix/generated/**` 或项目配置声明的 AreaFlow-owned generated target。
AreaMatrix 的源码、`workflow/versions/**`、
execution、progress、logs、checkpoint 和用户文件路径在单独授权前必须保持 blocked。

`managed-project generated-only rollback beta` 是第一条真实 managed project 写入演练：只允许单个已存在
普通 generated/projection 文件，写入后必须立即恢复到 preimage hash/size。`managed-project
generated-only retained apply` 才允许保留结果，并且需要单独 approval、focused smoke 和非目标文件指纹证据。

`manual patch artifact` 和 `human-applied source evidence` 是进入 source write beta 前的缓冲层。前者只生成
patch / diff artifact、write-set preview、expected-before、验证计划和 rollback/remediation plan；后者只读取
人工或现有 Codex 流程 apply 后的 diff、changed file hash 和验证结果。两者都不能写项目源码、运行 shell、
创建 checkpoint 或执行 repair。

`controlled source write beta` 第一版只允许 allowlist 内小范围 `create` / `modify` 文本源码，并且先停在
copy attempt -> verify attempt -> checkpoint preview。checkpoint apply 和 repair apply 必须在后续独立 gate
中打开，不能和 source write beta 同时隐式启用。

未来 apply 必须从 approved write-set artifact 开始，不能让 engine adapter 直接写项目文件。write-set 至少包含：

```text
operation: create | modify
target_path
target_path_kind
expected_before_sha256
expected_before_size
after_sha256
after_size
content_artifact_id or patch_artifact_id
verification_plan_artifact_id
rollback_plan_artifact_id
permission_capabilities
approval_id
```

第一版不支持 delete、move、chmod、binary rewrite、symlink target、project-root 外路径或 glob 批量写入。
`expected_before_sha256` 不匹配时必须 blocked，不能自动覆盖用户新改动。对新建文件，父目录必须已被
allowlist 明确允许，并且不能通过 `..`、符号链接或大小写绕过 project-root 防逃逸检查。

apply 前必须同时满足：

```text
execution approval gate pass
write_set gate pass
project permission allowlist pass
forbidden path check pass
expected before hash pass
worker capability pass
approval record present
rollback plan present
audit event writable
```

apply 成功时必须创建独立 attempt 和 evidence：

```text
attempt_kind = copy
preimage artifact for modified files
applied write-set artifact
post-write hash evidence
event: project_write_applied
audit_event: project_write_approved_and_applied
```

copy attempt 成功只代表写入完成，不代表 workflow item done。必须经过 verify attempt 和
`verify_acceptance` gate。verify 第一阶段可以只做 hash / file existence / schema / read-only doctor；
运行项目命令属于更高风险 verify command，需要 `run_commands` capability 和单独 command allowlist。

repair 不能直接修项目。verify 失败后先生成 failure summary 和 repair plan artifact。repair apply 必须走
与 copy 相同的 write-set、approval、permission、expected-hash 和 rollback 边界。

rollback 采用追加事实，不删除历史。只允许对 AreaFlow 自己 apply 的 write-set 做受控反向写入：

```text
rollback precondition:
  current file hash == failed attempt post-write hash
  preimage artifact exists and hash verified
  rollback approval present when required

rollback result:
  new rollback attempt
  restored hash evidence
  event / audit_event
```

如果用户或外部工具已经改动目标文件，rollback 必须 blocked，转人工处理。AreaFlow 不执行 `git reset --hard`、
不删除用户文件、不回滚未由 AreaFlow apply 的改动。

## Worker And Lease

Worker 不拥有状态源事实。Worker 只拿一次 run/task 的 scoped lease。
多项目调度、slot 计算、lease recovery、preview 与真实 scheduler 的完整合同见
[`worker-scheduling-contract.md`](worker-scheduling-contract.md)。
v0.6 worker beta 的 scoped execution 边界见 [`v0.6-worker-beta-contract.md`](v0.6-worker-beta-contract.md)。

规则：

- 同一个 run_task 同时最多一个 active execution lease。
- workflow item 只定义语义工作项；worker 不能直接领取 workflow item。
- lease 过期后可以被 recovery worker 接管。
- heartbeat 超时不等于失败，只表示 `needs_recovery`。
- worker 完成时必须提交 attempt、artifacts、event 和 audit event。
- worker 不得扩大自己的 capability。
- lease / run-once 请求的 `allowed_capabilities` 必须是 worker 注册 capabilities 的子集。
- capability preflight 失败必须写 denied event/audit，并且不得创建 lease、attempt 或 artifact。
- `worker.register`、`worker.heartbeat`、`lease.acquire`、`lease.release` 和 `lease.recover` 必须通过
  `command_requests` 记录幂等结果。
- worker lifecycle response 必须明确 `project_write_attempted=false`、`execution_write_attempted=false`、
  `engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、
  `lease_created=false`、`attempt_created=false`、`artifact_created=false` 和 `worker_run_once=false`。
- lease-only response 必须明确 `project_write_attempted=false`、`execution_write_attempted=false`、
  `engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、
  `attempt_created=false`、`artifact_created=false` 和 `worker_run_once=false`。
- worker register / heartbeat 只证明 registry 和 heartbeat 的 AreaFlow 内部状态写入，不代表真实
  execution、engine 调用、项目文件写入或远程 worker 凭证管理已经打开。

任务领取使用 PostgreSQL row lock：

```sql
SELECT ...
FOR UPDATE SKIP LOCKED
```

## 多项目调度

v0 起步使用 PostgreSQL row lock + lease；只有在 worker 数量、吞吐和远程调度压力明确超过 PG
方案后，才考虑 Redis / NATS 等外部队列。调度器领取任务时必须匹配：

```text
project_scope
required_capabilities
worker_type / agent_role
engine_profile readiness
secret readiness
resource limits
approval / gate state
risk policy
active lease
```

多项目调度字段至少预留：

```text
project_priority
run_priority
task_priority
deadline_at
max_parallel_runs_per_project
max_parallel_tasks_per_project
max_parallel_tasks_per_worker
```

默认策略：

- 同项目内按 run/task priority 和创建时间排序。
- 跨项目按 project priority、等待时间和资源可用性排序。
- 每个项目和 worker 都有并发上限。
- 高风险任务默认串行，除非 approval 明确允许并发。
- AreaMatrix 第一阶段真实 execution 并发固定为 1；提升并发必须有单独 approval、focused smoke 和
  non-target fingerprint evidence。
- v0.8 schedule preview 只能解释 recommended / blocked / available_slots，不创建 lease、不写 event/audit、
  不调用 worker。

运行环境分层：

```text
local_service
local_host
remote_worker
container_worker
host_bound_worker
```

远程 worker 不直连 PostgreSQL，只通过 API 注册、heartbeat、lease、attempt 和 artifact upload。
remote worker 必须具备 worker identity、token rotation、project scope、capability scope 和 audit trail。
AreaMatrix 这类需要 macOS app / Xcode / 本机 GUI 能力的项目默认使用 `local_host` / `host_bound_worker`，
不强制 container 化。

## Risk Policy

```text
risk_level:
  low
  medium
  high
  mission_critical

risk_policy:
  pause
  allow
  skip
```

命中高风险时，没有 run approval 就进入 `blocked`。`allow` 只能来自明确 approval 或已审计的 run policy。

## Drain / Cancel / Pause

Drain 是一等能力：

- 当前 task 跑完 copy、verify、checkpoint 后停止。
- 不跳过 verify。
- 不跳过 repair retry。
- 不进入下一个 task。

Cancel 用于停止未开始或可安全中断的 run。已经产生的 event、audit 和 artifact 不删除。

当前受保护实现先开放 dry-run run 的 DB-only 控制面：

```text
run.start:
  queued -> running

run.drain:
  running -> draining

run.cancel:
  queued -> cancelled
  running/draining -> cancelling
```

这些命令只写 AreaFlow PostgreSQL 中的 `runs`、`run_tasks`（仅 queued/cancel_requested 标记）、
`events`、`audit_events` 和 `command_requests`。它们不领取 lease、不中断正在跑的 worker、不写被管理项目、
不调用 engine，也不代表真实 execution beta 已打开。真实 worker drain/cancel 需要在后续 execution beta
里继续补 heartbeat 协作、中断策略和恢复 evidence。

Run control command response 必须记录 `project_write_attempted=false`、`execution_write_attempted=false`、
`engine_call_attempted=false`、`task_claimed=false`、`worker_started=false`、`commands_run=false`、
`secrets_resolved=false` 和 `network_used=false`。非 dry-run run 必须被拒绝，并留下 denied command response。

## Checkpoint

Checkpoint 是执行证据，不是完成条件本身。

```text
verify passed + checkpoint failed = blocked_checkpoint
```

不能因为 verify PASS 就进入下一个任务；checkpoint 失败必须先恢复或明确降级。

## 失败恢复

```text
in_progress + active lease -> running
in_progress + expired lease + no pass artifact -> stale / needs_recovery
verify pass + checkpoint fail -> blocked_checkpoint
attempt failed + retries left -> retryable
attempt failed + no retries -> failed
risk gate blocked -> blocked_risk
```

恢复流程要先归因：copy、verify、validation、runner、checkpoint、docs drift、权限、文件安全边界不能混为一类。

## AreaMatrix 迁移口径

- v1 历史 `progress.json` 只作为 import source 和 artifact metadata。
- 新进度源事实在 PostgreSQL。
- AreaMatrix 最终只保留 `.areaflow/status.json` 粗略投影。
- 兼容命令可以转发到 AreaFlow，但不得维护第二套 progress。

## 版本阶段

- v0.5 建 run/attempt/queue model，只 dry-run。
- v0.6 引入 worker、lease、Codex CLI adapter preview、execution approval gate、fixture-only execution
  apply、read-only verify、approved artifact write、fixture-only approved project write 和 fixture/temp
  generated rollback drill；这些只证明各自 scope，默认仍不打开真实 engine、managed project source write
  或 AreaMatrix execution cutover。
- v0.7 才允许在明确 approval、allowlist、audit 和 rollback 边界下试点真实项目 execution。
- v0.8 扩展到多项目、多 worker、resource limit readiness 和 engine readiness；真实 engine routing apply
  仍需后续单独开闸。

真实执行建议按以下顺序打开：

```text
runner preview
-> worker dry-run
-> fixture execution
-> read-only verify on managed project
-> approved artifact write
-> fixture approved project write
-> managed generated write gate
-> fixture/temp generated-only rollback drill
-> real managed generated-only rollback beta
-> real managed generated-only retained apply
-> manual patch artifact
-> human-applied source evidence
-> controlled source write beta
-> checkpoint preview
-> checkpoint apply
-> repair plan
-> repair apply
-> no-secret engine execution
-> secret resolve
-> remote worker
```

AreaMatrix 不应在 authoring cutover 阶段自动转发 `./task-loop run`；execution cutover 必须等待
run/task/attempt/evidence/audit 闭环被真实 smoke 证明。

v0.6i 的 fixture execution 已证明以下最小闭环：

```text
execution approval gate pass
-> worker fixture-execute
-> completed fixture_execution lease
-> passed fixture_execution attempt
-> fixture_execution_report artifact
-> run_task status passed
-> run_task outcome fixture_execution_passed
-> run status passed
```

该闭环只写 AreaFlow PG state 和 AreaFlow-owned artifact store。它不执行 copy/verify/repair，不调用
Codex CLI 或其他 engine，不运行 shell，不解析 secret，不访问网络，不写被管理项目，也不写
`workflow/versions/**/execution/**`。

v0.6j 的 read-only verify 已证明以下最小闭环：

```text
execution approval gate pass
-> worker read-only-verify
-> allowlisted project file read
-> completed read_only_verify lease
-> passed read_only_verify attempt
-> read_only_verify_report artifact
-> run_task status passed
-> run_task outcome read_only_verify_passed
-> run status passed
```

该闭环会读取 project config allowlist 允许的 target file，但只在 AreaFlow-owned artifact store 中保存
path、sha256 和 size evidence，不保存 target file 原文。它不执行 copy/verify/repair，不调用 Codex CLI
或其他 engine，不运行 shell，不解析 secret，不访问网络，不写被管理项目，也不写
`workflow/versions/**/execution/**`。

v0.6n 的 fixture-only approved project write 已证明以下最小闭环：

```text
fixture project write queue
-> fixture_project_write_set artifact
-> execution approval gate pass
-> worker fixture-project-write
-> expected-before hash/size check
-> preimage artifact
-> copy attempt
-> verify attempt
-> rollback attempt
-> fixture_project_write_report artifact
-> run_task status passed
-> run_task outcome rollback_verified
-> run status passed
```

该闭环只允许 fixture project，且只修改已存在、非 symlink、非目录、位于 project root 内并被
`areaflow.yaml` allowlist 允许的单个 target file。写入后必须立即 rollback 到 preimage hash/size。
它不调用 Codex CLI 或其他 engine，不运行 shell，不解析 secret，不访问网络，不写真实 AreaMatrix，也不写
`workflow/versions/**/execution/**`。真实 managed-project generated-only write、source write、checkpoint
和 repair 仍关闭。

v0.6o 的 managed generated write gate 已证明以下只读门禁：

```text
execution approval gate pass
-> managed-generated-write-gate
-> generated-only prefixes exposed
-> required write-set fields exposed
-> source/execution/checkpoint/repair/destructive operations blocked
-> generated_only_write_ready true
-> generated_only_apply_open false
```

该 gate 不创建 command、lease、attempt 或 artifact，不读取或写入被管理项目，不调用 engine，不运行 shell，
不解析 secret，不访问网络，也不写 `workflow/versions/**/execution/**`。它只为后续 managed-project
generated-only apply 提供统一 preflight 数据；真实 generated-only apply 仍关闭。它的 required
capabilities 使用 `read_project`、`write_artifacts` 和 `write_generated`，不借用 `write_code`。

v0.6p 已增加 managed generated write apply service 和受限 API/CLI surfacing，用于 fixture/temp project
内的 generated-only 写入演练：

```text
queue managed generated write
-> execution approval gate pass
-> worker capability read_project/write_artifacts/write_generated
-> project capability write_artifacts/write_generated
-> read_project + write_generated path allowlist
-> generated prefix check
-> expected-before check
-> copy
-> verify
-> rollback
-> report artifact
```

当前 v0.6p 只支持已存在的普通 generated 文件，并在 commit 前 rollback 到 preimage hash/size。API/CLI 入口
只调用同一条 fixture/temp rollback drill；真实 AreaMatrix generated apply、保留生成结果、source write、
checkpoint、repair、engine、shell、secret、network 和 `workflow/versions/**/execution/**` 仍关闭。
