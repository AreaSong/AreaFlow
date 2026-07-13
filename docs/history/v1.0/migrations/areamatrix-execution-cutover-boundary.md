# AreaMatrix Execution Cutover Boundary

## 定位

本文定义 AreaMatrix 从 `./task-loop` 主执行入口切到 AreaFlow execution 主入口的边界。它补充
[`areamatrix-workflow-migration.md`](./areamatrix-workflow-migration.md)、
[`areamatrix-compatibility-shim-plan.md`](./areamatrix-compatibility-shim-plan.md) 和
[`../architecture/execution-opening-strategy.md`](../../../../proposals/execution-opening.md)，并引用
[`../architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md) 的 scoped
evidence / no-cutover 规则。

Execution cutover 不是 v0.4 authoring cutover，不是 compatibility shim 落地，也不是
`execution-cutover-readiness` 返回可查询。只有显式 execution cutover approval、受保护 Command API、
permission、gate、rollback 和 audit 全部通过后，AreaMatrix 的执行入口才能转发到 AreaFlow。

本文不授权修改 AreaMatrix。真实改动 `workflow/README.md`、`.areaflow/status.json`、`scripts/**` 或
`./task-loop` 行为仍需要单独确认。

## 阶段边界

| 阶段 | AreaMatrix 入口行为 | AreaFlow 行为 | `./task-loop run` |
|---|---|---|---|
| Import / Mirror / Shadow | AreaMatrix 仍是 workflow/execution 源事实 | AreaFlow 只读导入、mirror、shadow doctor | blocked / legacy-only，不转发 |
| Authoring Cutover | 新 workflow authoring 源事实切到 AreaFlow | 管理新 version、stage、gate、approval | blocked，不转发 |
| Compatibility Shim | 只读 status/doctor/open/init preview 可转发或 fallback | 提供 compatibility/shim/readiness 查询 | blocked，不转发 |
| Execution Beta | AreaFlow 可运行受限 approved execution | fixture、read-only verify、artifact-only、generated drill 等 scoped apply | blocked，不转发 |
| Execution Cutover | AreaMatrix execution 入口转发到 AreaFlow | AreaFlow 成为 task execution 主入口 | 只转发到受保护 AreaFlow Command API |
| Archive / Shim Retirement | AreaMatrix 保留轻量入口和项目事实 | AreaFlow 持有 run/attempt/artifact/audit 主状态 | 旧 runner 退役 |

因此，Execution Beta 可以在 AreaFlow 内证明 worker/run/attempt 能力，但在 execution cutover approval 前，
AreaMatrix 的 `./task-loop run` 仍必须保持 blocked。
v0.6 scope 内的 `passed`、`verified`、`artifact_written`、`rollback_verified`、`ready_for_review` 或
`needs_approval` 只能证明对应 scoped ability，不能被解释为 forwarding gate 已满足。

## Shim Lifecycle State

Compatibility shim 生命周期必须显式建模，不能只用 installed / done 表示：

```text
not_installed:
  AreaMatrix 没有 shim 文件改动；AreaFlow 只提供 readiness / authorization packet。

read_only_shim:
  只转发 status、doctor、open、init preview 等只读或 authoring command；`./task-loop run` blocked。

execution_forwarding:
  `./task-loop run` 打印 migration notice 后转发 AreaFlow 受保护 Command API；旧 runner 不启动。

retired_thin_entry:
  AreaMatrix 只保留 `workflow/README.md`、`.areaflow/status.json`、`./dev workflow open/status`
  等轻量入口；旧 task-loop 主执行逻辑退役。
```

状态只能单向推进，除 rollback 外不得跳级。`read_only_shim` 不是 execution cutover；
`execution_forwarding` 不是 shim retirement；`retired_thin_entry` 也不能删除历史 evidence。

## Cutover Apply 不是 Readiness

当前已有的 `execution-cutover-readiness` 是只读 go/no-go view：

```text
GET /api/v1/projects/{project_key}/execution-cutover-readiness
areaflow project execution-cutover-readiness <project>
```

它只能聚合证据并返回 blockers，不能：

```text
创建 command request
创建 approval
修改 AreaMatrix
写 workflow/versions/**/execution/**
转发 ./task-loop run
启动 worker
调用 engine
解析 secret
运行 shell
```

未来如果打开 execution cutover apply，必须是新的受保护 Command API，例如：

```text
POST /api/v1/projects/{project_key}/execution-cutover-apply
areaflow project execution-cutover-apply <project>
```

该命令在实现前只能作为设计占位，不得在文档或 UI 中显示为可执行能力。

## 必须关闭的前置缺口

进入 execution cutover apply 前，至少必须关闭以下缺口：

```text
compatibility_shim_landed
real_areamatrix_generated_rollback_beta_passed
retained_generated_apply_policy_passed
manual_patch_artifact_or_human_applied_source_evidence_ready
copy_verify_repair_checkpoint_state_machine_passed
checkpoint_apply_policy_passed
repair_apply_policy_passed
execution_approval_gate_passed
project_permission_allowlist_passed
command_allowlist_passed
rollback_to_read_only_mode_plan_passed
explicit_execution_cutover_approval_recorded
```

如果任一项仍是 preview、fixture-only、readiness、blocked 或 missing，AreaMatrix execution cutover 必须保持
blocked。

## Execution Forwarding Gate

`./task-loop run` 从 blocked 进入 forwarding 前，必须额外满足：

```text
shim_lifecycle_state = read_only_shim
execution_cutover_readiness = pass
explicit_execution_cutover_approval_recorded
AreaFlow execution command target defined
legacy runner bypass proof
legacy progress/log/checkpoint non-write proof
AreaFlow command/event/audit evidence proof
fallback behavior fail_closed proof
rollback_to_blocked_forwarding proof
```

Forwarding 第一版只能转发到 AreaFlow Command API，不得直接调用 `areaflow worker`、shell、engine、
旧 prompt pipeline 或旧 runner helper。如果 AreaFlow API / CLI 不可用，必须 fail closed。

### Execution Forwarding v1 Scope

第一版真实 forwarding 只证明 AreaFlow 接管 AreaMatrix 的执行入口、执行状态和审计链，不同时打开自动写代码。
`./task-loop run` 转发后只能消费受保护 Command API 中明确标记为 read-only / evidence 的任务：

```text
verify-ready task
doctor/readiness task
artifact evidence task
status/projection validation task
release/readiness check task
```

第一版 forwarding 必须继续禁止：

```text
copy-ready source write
repair apply
checkpoint apply
generated retained write
secret-backed engine run
network/API integration task
```

因此，Execution Forwarding v1 的 go 条件不是“所有 execution 能力都打开”，而是：

```text
./task-loop run does not start legacy runner
./task-loop run creates or forwards an AreaFlow Command API request
AreaFlow owns command/run/run_task/run_attempt/artifact/audit state for the forwarded task
allowed task types are limited to read-only verify, doctor/readiness and evidence tasks
legacy progress.json/log/checkpoint files remain unchanged
protected path proof is clean or explicitly authorized
failure path returns blocked / fail-closed behavior
rollback can disable forwarding and return to read_only_shim
```

Generated rollback beta、retained generated apply、manual patch artifact、human-applied source evidence、source write
beta、checkpoint apply、repair apply、no-secret engine execution 和 secret / remote / publish / restore / plugin
能力都必须作为后续独立门禁打开。它们不能因为 Execution Forwarding v1 通过而自动获得 apply 权限。

## AreaMatrix 命令映射

### Cutover 前

| AreaMatrix 命令 | 行为 | 说明 |
|---|---|---|
| `./dev workflow status` | read-only forward / fallback | 读取 AreaFlow summary 或 `.areaflow/status.json`。 |
| `./dev workflow doctor` | read-only forward | 不隐式运行 native doctor；`--allow-native` 另行授权。 |
| `./dev workflow init --version <v>` | preview by default | 不带 `--write` 不创建 AreaMatrix 目录。 |
| `./dev workflow init --version <v> --write` | AreaFlow authoring command | 只创建 AreaFlow-authored version，不写 AreaMatrix `workflow/versions/**`。 |
| `./dev workflow open` | read-only dashboard link | 打开或打印 AreaFlow URL。 |
| `./task-loop status` | read-only forward / fallback | 只显示粗略状态。 |
| `./task-loop check` | read-only forward or blocked | 可作为后续只读扩展，不属于最小 shim。 |
| `./task-loop run` | blocked | 不能转发，不能启动旧 runner。 |
| `./task-loop resume-*` | blocked | 不能读写旧 progress/lock/log。 |
| `./task-loop reset-progress` | blocked | 不能重置旧主状态。 |
| `./task-loop clear-stale` | blocked | 不能清理旧 lock/log/checkpoint。 |

### Cutover 后

| AreaMatrix 命令 | 行为 | 禁止 |
|---|---|---|
| `./dev workflow status` | 转发 AreaFlow summary / dashboard | 维护本地第二状态。 |
| `./dev workflow doctor` | 转发 AreaFlow doctor/readiness | 隐式 native doctor、写 AreaMatrix。 |
| `./dev workflow init --version <v>` | 预览或转发 AreaFlow authoring command | 写 AreaMatrix workflow skeleton。 |
| `./task-loop status` | 转发 AreaFlow project/run summary | 读取旧 progress 作为主状态。 |
| `./task-loop run` | 打印 migration notice 后转发 AreaFlow execution command | 运行旧 prompt pipeline、写旧 progress/log/checkpoint。 |
| `./task-loop resume-*` | 转发 AreaFlow run recovery/drain/cancel 命令，或 blocked | 操作旧 lock/progress。 |
| `./task-loop reset-progress` | retired / blocked | 删除或重写历史 `progress.json`。 |
| `./task-loop clear-stale` | retired / blocked | 清理历史 evidence、logs、checkpoint。 |

Cutover 后的 `./task-loop run` 也不能直接调用 worker binary、shell 或 engine。它只能调用 AreaFlow
Command API，由 AreaFlow 决定 permission、approval、lease、attempt、artifact 和 audit。
第一版 execution forwarding 还必须把转发目标限制在 read-only verify / doctor / readiness / evidence
类任务；任何 copy、repair、checkpoint、generated retained write、source write、engine、secret、network 或
publish/restore 类目标必须 fail closed。

## 转发目标要求

`./task-loop run` 的未来转发目标必须满足：

```text
Command API request
idempotency_key + request_hash
actor + reason
project_key scope
workflow_version scope
queue/run scope
approval_id
risk_level = R3 execution
permission capability preflight
path / command / network / secret policy preflight
execution approval gate pass
rollback / remediation plan
audit event
focused smoke evidence
```

转发响应必须返回 safety facts：

```text
legacy_task_loop_started=false
legacy_progress_written=false
legacy_logs_written=false
legacy_checkpoint_written=false
area_flow_command_created=true
area_flow_run_created=true|false
worker_scheduled=true|false
project_write_attempted=true|false
execution_write_attempted=true|false
engine_call_attempted=true|false
commands_run=true|false
secrets_resolved=true|false
network_used=true|false
```

如果 AreaFlow API 不可用，`./task-loop run` 必须 fail closed，不能回退到旧 runner 自动执行。

## Protected Paths

Execution cutover 前后都不得把以下路径重新当作 AreaMatrix 主执行状态：

```text
workflow/versions/v1-mvp/execution/**
workflow/versions/**/execution/_shared/progress.json
.codex/runtime/task-loop/**
old runner lock files
old task-loop logs
old git checkpoint evidence
```

Cutover 后历史路径可以作为 archive/reference 被读取，但不能被 AreaFlow shim 当作 live state 写回。

新 execution evidence 的主状态必须在 AreaFlow PostgreSQL 和 AreaFlow-owned artifact metadata 中。需要对
AreaMatrix 投影时，只能写受控 projection，例如 `.areaflow/status.json` 或已批准的 `workflow/README.md`
粗略区块。

## Archive Gate

Archive 必须晚于 execution cutover 的基础证据。它表示 AreaMatrix 历史 workflow / execution
主索引不再由 AreaMatrix 继续维护，但历史文件仍作为 archive/reference 保留。

Archive 完成时必须同时证明：

```text
historical_workflow_versions_marked_immutable
historical_execution_metadata_indexed_in_areaflow
historical_artifact_refs_have_hash_path_type_project_version_run
project_reference_restore_limitations_recorded
old_progress_logs_checkpoints_are_reference_only
new_run_attempt_artifact_audit_state_owned_by_areaflow
AreaMatrix_workflow_README_points_to_areaflow_summary
AreaMatrix_status_json_contains_only_rough_projection
archive_does_not_delete_or_move_historical_files
archive_does_not_rewrite_progress_json
rollback_to_execution_forwarding_documented
```

Archive 禁止：

```text
删除 workflow/versions/**
删除 progress.json、logs、evidence、release evidence
把 metadata-only project_reference 伪装成 AreaFlow-owned artifact 原文
把旧 progress/log/checkpoint 重新作为 live state
双写 AreaMatrix 旧 runner 状态和 AreaFlow run 状态
```

Archive 不是 shim retirement。Archive 只处理历史索引、主状态所有权和引用策略；Shim retirement
才处理旧命令和旧 task-loop 主能力是否退役。

## Shim Retirement Gate

Shim retirement 必须晚于 execution forwarding 稳定期。进入 `retired_thin_entry` 前至少需要：

```text
archive_gate_passed
execution_forwarding_stable_for_declared_window
no legacy task-loop run usage in active workflow versions
AreaFlow run/attempt/artifact/audit coverage pass
compat commands mapped or deliberately blocked
legacy progress/log/checkpoint archive/reference policy accepted
rollback_to_read_only_shim documented
user-facing retirement notice present
AreaMatrix protected paths fingerprinted
```

可退役的是旧主执行能力，不是历史文件、项目事实或验证命令。Retirement 禁止：

```text
删除 workflow/versions/** 历史
删除 progress.json / logs / evidence
删除 release evidence
删除 AreaFlow command/event/audit/run/attempt/artifact
让 ./task-loop run 静默成功但不转发 AreaFlow
让旧 runner 与 AreaFlow runner 双写
```

退役后，`./task-loop run` 可以保持 blocked with migration notice，或继续 thin forward 到 AreaFlow；两者必须
在 retirement notice 中明确，且都不得恢复旧 runner。

## Rollback

Execution cutover rollback 只能回到更保守模式，不能删除历史事实：

```text
soft rollback:
  disable task-loop run forwarding
  set execution ownership back to read-only / external
  keep AreaFlow run/attempt/artifact/audit history
  keep AreaMatrix status fallback

recovery rollback:
  mark affected AreaFlow run needs_recovery or cancelled
  keep leases/attempts/artifacts
  require human review before retry

hard rollback:
  only allowed before any project write or worker execution starts
  record cutover_failed event and audit
  do not delete AreaFlow rows
```

Rollback 不得：

```text
重写 v1 historical execution
清空 progress.json
删除 logs/evidence
删除 AreaFlow events/audit_events/runs/attempts/artifacts
自动恢复旧 ./task-loop run 主能力
```

## 验证包

进入真实 AreaMatrix execution cutover apply 前，验证包至少包含：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-areamatrix-readonly.sh
AREAFLOW_DATABASE_URL=... ./scripts/smoke-compatibility-fixture.sh
AREAFLOW_DATABASE_URL=... ./scripts/smoke-v1-stable-fixture.sh
go test ./internal/project -run 'ExecutionCutover|ExecutionPlan|GeneratedWrite|ManagedGenerated|ProjectWrite|Worker|RunControl'
go test ./internal/api -run 'ExecutionCutover|GeneratedWrite|ManagedGenerated'
go test ./internal/app -run 'ExecutionCutover|GeneratedWrite|ManagedGenerated|Help'
git -C /Users/as/Ai-Project/project/AreaMatrix status --short
```

真实 cutover 落地后，AreaMatrix 侧还必须证明：

```text
./task-loop run does not start legacy runner
legacy progress.json unchanged
legacy execution logs unchanged
workflow/versions/**/execution/** unchanged unless an explicit AreaFlow projection write was approved
AreaFlow command/event/audit evidence exists
rollback to blocked run forwarding works
```

## Go / No-Go

Go 条件：

```text
execution-cutover-readiness pass
compatibility shim installed and verified
execution-opening-strategy required steps open with evidence
explicit execution cutover approval present
rollback drill passed
AreaMatrix dirty worktree reviewed
AreaMatrix protected paths fingerprinted before apply
```

Shim retirement Go 条件：

```text
execution forwarding stable
legacy runner disabled or blocked with notice
all active execution state owned by AreaFlow
compatibility commands documented
rollback to read-only shim available
historical files retained as archive/reference
```

No-Go 条件：

```text
readiness is blocked or preview-only
AreaMatrix shim not landed
real generated rollback beta missing
copy / verify / repair / checkpoint evidence incomplete
rollback plan missing
permission or command allowlist fail
expected-before evidence missing
user did not explicitly approve AreaMatrix edits
```

No-Go 时必须保持 `./task-loop run` blocked，并继续只读展示 AreaFlow readiness / blockers。
