# AreaMatrix Compatibility Shim Plan

## Purpose

本文定义 AreaMatrix 仓库里 compatibility shim 的最小落地计划。它不是 cutover apply 方案，也不是
task-loop replacement 方案；它只说明在获得明确授权后，AreaMatrix 如何保留轻量入口并转发或降级展示
AreaFlow 状态。

## Scope

第一版 shim 只覆盖以下入口：

```text
./dev workflow status
./dev workflow doctor
./dev workflow init --version <version>
./dev workflow open
./task-loop status
```

明确禁止：

```text
./task-loop run
promotion apply
write execution
git checkpoint
delete/archive historical workflow
```

`./task-loop run` 必须等 v0.5/v0.6 runner / worker 权限模型、approval、audit 和 checkpoint 语义稳定后，
再通过单独设计进入 execution beta。AreaMatrix 执行入口的真实切换边界见
[`areamatrix-execution-cutover-boundary.md`](./areamatrix-execution-cutover-boundary.md)；shim 落地本身不打开
`./task-loop run` 转发。
v0.5 runner preview 边界见
[`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md)；runner
preview 只证明 dry-run evidence，不授权 shim 转发 execution。

第一版 shim 的 lifecycle state 只能是 `read_only_shim`。它不能声明 `execution_forwarding` 或
`retired_thin_entry`。这两个状态必须等 execution cutover / retirement gate 单独通过。

## Source Of Truth

AreaMatrix shim 不维护第二套 workflow 状态。它的判断来源按优先级排列：

```text
1. AreaFlow local API:
   GET /api/v1/projects/{project_key}/compatibility
   GET /api/v1/projects/{project_key}/shim-preview
   GET /api/v1/projects/{project_key}/shim-readiness
   GET /api/v1/projects/{project_key}/shim-authorization
   GET /api/v1/projects/{project_key}/summary
   GET /api/v1/projects/{project_key}/readiness

2. AreaFlow CLI:
   areaflow project compatibility <project>
   areaflow project shim-preview <project>
   areaflow project shim-readiness <project>
   areaflow project shim-authorization <project>
   areaflow project summary <project>
   areaflow project readiness <project>
   areaflow workflow version create <project> <version>

3. Offline fallback:
   .areaflow/status.json
```

`.areaflow/status.json` 是 projection，不是源事实。AreaMatrix 只能读取它用于降级展示，不得把它写回
workflow、execution、progress、logs 或 checkpoint。

## Command Mapping

| AreaMatrix command | AreaFlow target | Default mode | Fallback | Notes |
|---|---|---|---|---|
| `./dev workflow status` | `areaflow project summary areamatrix` | read-only forward | `.areaflow/status.json` | 输出粗略状态和 AreaFlow 入口。 |
| `./dev workflow doctor` | `areaflow project doctor areamatrix` | read-only forward / fallback | `.areaflow/status.json` | 不隐式传 `--allow-native`；native doctor 仍需显式授权。 |
| `./dev workflow init --version <version>` | AreaFlow version preview / create | preview by default | 显示 AreaFlow unavailable | 不带 `--write` 时保持 preview 语义；带 `--write` 才转发 AreaFlow Command API 创建 authored record，且不写 AreaMatrix `workflow/versions/**`。 |
| `./dev workflow open` | AreaFlow dashboard / `workflow-versions` query | read-only forward | `.areaflow/status.json` | 打开或打印 AreaFlow dashboard URL。 |
| `./task-loop status` | `areaflow project summary areamatrix` | read-only forward | `.areaflow/status.json` | 只显示状态，不运行任务。 |
| `./task-loop run` | none | blocked | blocked message | v0.4 禁止自动转发。 |

命令映射中的 read-only forward 只能使用 Query API 或 CLI query。任何会创建 command、run、lease、
attempt、artifact、checkpoint、repair、restore、publish 或 project write 的动作，都不属于第一版 shim。

## Minimal AreaMatrix Files

获得确认后，AreaMatrix 侧预计只需要小范围改动。当前 `dev` 和 `task-loop` 根入口只是薄
Python launcher，但真实分派路径不同：

```text
./dev -> scripts/task_loop/console.py -> scripts/dev_tools/cli.py
./task-loop -> scripts/task_loop/runner.py
```

第一版 shim 不应改根入口，而应改真实分派层：

```text
scripts/task_loop/console.py
scripts/dev_tools/cli.py
scripts/task_loop/runner.py
workflow/README.md
.areaflow/status.json
```

可选新增文件：

```text
scripts/areaflow_shim.py
```

推荐做法是把公共逻辑放在 `scripts/areaflow_shim.py`。`scripts/task_loop/console.py` 保持菜单和
生命周期入口的用户体验，但对 `workflow status`、`workflow doctor`、`workflow init`、`workflow open`
等动作必须转入同一 shim 决策；`scripts/dev_tools/cli.py` 负责非交互 workflow 命令分派；由
`scripts/task_loop/runner.py` 只拦截 `status` 并在 execution cutover 前阻断 `run` / `resume-*` /
`reset-progress` / `clear-stale` 等会写 progress、logs、checkpoint 或执行任务的命令：

```text
resolve AreaFlow endpoint / CLI
load compatibility contract
forward read-only command or allowed command
fallback to .areaflow/status.json
print blocked message for forbidden command
```

不得改动：

```text
workflow/versions/**/execution/**
workflow/versions/**/execution/_shared/progress.json
workflow/versions/v1-mvp/**
tasks/active/**
tasks/done/**
release evidence
source code
user files
```

`workflow/README.md` 第一阶段只允许增加或更新一个清晰的人工说明区块，指向 AreaFlow 和
`.areaflow/status.json`。自动受控区块写入要等 v0.4 cutover approval 后再打开。
`.areaflow/status.json` 只能通过 AreaFlow `project.status_projection.apply` 受保护 Command API 更新，
必须带 schema validation、expected-before preimage、protected path check、rollback action 和显式 approval；
不得由 shim 脚本手写，也不得承载完整 queue、run、approval、logs、checkpoint、secret 或 artifact 原文。

## Environment And Discovery

Shim 按以下顺序发现 AreaFlow：

```text
AREAFLOW_API_URL
AREAFLOW_BIN
PATH 中的 areaflow
.areaflow/status.json
```

本机默认 API 可使用：

```text
http://127.0.0.1:3847/api/v1
```

如果 API 和 CLI 都不可用，shim 只能读取 `.areaflow/status.json`。如果 status projection 也不存在，
命令应返回清晰的 unavailable message，不得尝试恢复旧 workflow 状态。
对于 `./task-loop run`，API / CLI 不可用时也必须继续 blocked，不能回退到旧 runner。

## Safety Gates

Shim 必须满足：

```text
read-only commands only use GET / CLI query
init/version creation only forward to AreaFlow command API / CLI
no direct PostgreSQL access
no direct write to AreaMatrix workflow versions
no execution writes
no task-loop run
no git mutation
no AI engine invocation
```

`./task-loop drain` 第一版不作为常规 shim 命令开放。若需要处理遗留 live runner，只能作为
recovery-only 设计单独进入 gate：必须先证明存在旧 live lock、只写 legacy control file、不创建新的
AreaFlow execution 状态，并留下人工 recovery 证据；否则保持 blocked。

`./task-loop run`、`resume-*`、`reset-progress` 和 `clear-stale` 在 first shim 中必须 fail closed。Blocked
message 应指向 AreaFlow readiness / migration notice，但不得启动旧 prompt pipeline 或写 progress/log/checkpoint。

`./dev workflow doctor` 不得偷偷执行 AreaMatrix native doctor。若用户需要 native doctor，仍应通过
AreaFlow 的 explicit `--allow-native` 或 AreaMatrix 原有验证命令单独运行。

## AreaMatrix Edit Authorization Packet

进入真实 AreaMatrix shim implementation 前，必须先把下面授权包展示给用户并获得明确确认。该授权只覆盖
compatibility shim 最小落地，不覆盖 execution cutover、source write、generated retained apply、
checkpoint、repair、secret、engine 或 publish。

### Intent

目标是让 AreaMatrix 保留轻量入口，并在 AreaFlow 可用时转发只读 workflow/status 命令，在 AreaFlow
不可用时读取 `.areaflow/status.json` 进行降级展示。它不是迁移历史目录，不是执行任务，也不是替代
`./task-loop run`。

### Files Allowed To Change

仅允许修改或新增以下路径：

```text
scripts/task_loop/console.py
scripts/dev_tools/cli.py
scripts/task_loop/runner.py
workflow/README.md
.areaflow/status.json
scripts/areaflow_shim.py
```

其中 `scripts/areaflow_shim.py` 是可选公共 helper；如果现有代码结构可以保持清晰，也可以不新增。
`workflow/README.md` 只允许写人读说明和 AreaFlow 入口，不得承载完整 workflow 主状态。
`.areaflow/status.json` 只允许作为 R1 projection write 由 AreaFlow status projection apply command 生成或更新；
若当前文件仍是 legacy shape，可把 status projection apply 作为同一授权包的第一步，但必须先展示
authorization / packet / gate，并保留 preimage rollback。

### Files And Actions Forbidden

授权包必须继续禁止：

```text
workflow/versions/**
workflow/versions/**/execution/**
workflow/versions/**/execution/_shared/progress.json
workflow/versions/v1-mvp/**
tasks/active/**
tasks/done/**
.codex/runtime/task-loop/**
release evidence
source code
user files
git checkpoint
./task-loop run forwarding
promotion apply
native doctor without explicit --allow-native
```

如果实现过程中发现必须触碰上述任一路径或动作，本次授权自动失效，必须重新说明影响、风险、验证和回滚。

### Required Preflight

请求 AreaMatrix 编辑授权前，AreaFlow 侧必须已经具备：

```text
areaflow project compatibility areamatrix --json
areaflow project shim-preview areamatrix --json
areaflow project shim-readiness areamatrix --json
areaflow project shim-authorization areamatrix --json
areaflow project shim-apply-packet areamatrix --json
areaflow project shim-apply-gate areamatrix --json
areaflow project shim-apply areamatrix --json  # records protected AreaFlow command state; no AreaMatrix project or execution write
areaflow project status-projections areamatrix --json
areaflow project status-projection-authorization areamatrix --json
areaflow project status-projection-apply-packet areamatrix --json
areaflow project status-projection-apply-gate areamatrix --json
python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash
make smoke-docker-shim-authorization-preflight
AREAFLOW_DATABASE_URL=... ./scripts/smoke-areamatrix-readonly.sh
git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json
```

`make smoke-docker-shim-authorization-preflight` 是 AF-V04 授权前推荐入口；它复用
`scripts/smoke-areamatrix-readonly.sh` 作为底层真实 AreaMatrix 只读 smoke。

preflight 必须证明：

```text
compatibility contract pass or accepted warn
shim preview lists only allowed files
shim readiness blocked only on explicit edit approval
shim authorization status is blocked and safety facts show no project/execution/engine/command/network action
shim apply packet/gate is ready only when readiness blockers are limited to explicit_edit_approval and proof ids are complete
shim apply packet/gate remains read-only and reports no command/project/execution/task-loop/engine action
shim apply command records command_request/event/audit state only after gate review; it still reports no AreaMatrix project/execution/task-loop/status write
status projection validates against schemas/status-projection.schema.json
status projection exposes stable_fallback_projection_v1 required fields and excludes broad legacy fields
status projection apply packet/gate is ready or blocked with explicit, reviewable reasons
real AreaMatrix read-only smoke did not change .areaflow/status.json
real AreaMatrix read-only smoke did not change workflow/README.md
dirty worktree reviewed before edit
```

如果 `.areaflow/status.json` 仍是 legacy schema，进入 AreaMatrix shim implementation 前有两种合格路径：

```text
preferred:
  先单独授权并执行 status-projection-apply，把 .areaflow/status.json 更新为 stable_fallback_projection_v1；
  再重新运行 shim-readiness / shim-authorization。

combined:
  在同一个明确授权包中先执行 status-projection-apply，再落地 read_only_shim 文件；
  如果 projection apply 失败，停止后续 shim 文件修改。
```

两种路径都必须保持 `.areaflow/status.json` 的 rollback preimage，并证明 `workflow/versions/**`、
`progress.json`、logs、checkpoint 和 source files 未变化。

### Post-edit Verification

真实落地后至少运行：

```bash
cd /Users/as/Ai-Project/project/AreaMatrix
./dev workflow status
./dev workflow doctor
./dev workflow init --version shim-smoke
./dev workflow open
./task-loop status
./task-loop run
./dev workflow doctor
python3 /Users/as/Ai-Project/project/AreaFlow/scripts/validate-status-projection-schema.py /Users/as/Ai-Project/project/AreaFlow/schemas/status-projection.schema.json .areaflow/status.json
git diff --check -- scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py workflow/README.md scripts/areaflow_shim.py .areaflow/status.json
```

预期结果：

```text
status / doctor / init preview / open / task-loop status return clear forwarded or fallback output
./task-loop run returns blocked and does not start runner
workflow/versions/**/execution/** unchanged
progress.json unchanged
no AreaMatrix source files changed
no git checkpoint created
.areaflow/status.json validates against stable_fallback_projection_v1 if projection apply was authorized
```

`./dev workflow init --version shim-smoke` 在不带 `--write` 时必须保持 preview；如果需要测试
`--write`，必须单独确认它只转发 AreaFlow Command API，并且不写 AreaMatrix `workflow/versions/**`。

### Rollback Scope

如果 shim 落地后验证失败，rollback 只允许处理本授权包列出的 shim 文件。不得删除 AreaFlow
events、audit_events、workflow_versions、runs、attempts 或 artifacts；不得回写 v1 historical execution、
`progress.json`、logs 或 checkpoint。
如果 `.areaflow/status.json` 在同一授权包内通过 projection apply 更新，rollback 只能恢复 captured
preimage bytes，且必须重新运行 schema/protected-path verification。

## Verification

AreaFlow 侧先验证 contract：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-fixture.sh
AREAFLOW_DATABASE_URL=... ./scripts/smoke-areamatrix-readonly.sh
AREAFLOW_DATABASE_URL=... ./scripts/smoke-web.sh
areaflow project compatibility areamatrix --json
areaflow project shim-preview areamatrix --json
areaflow project shim-readiness areamatrix --json
areaflow project shim-authorization areamatrix --json
areaflow project status-projections areamatrix --json
```

`smoke-areamatrix-readonly.sh` 必须保持只读：它可以注册/导入真实 AreaMatrix metadata 到 AreaFlow
数据库、运行 AreaFlow doctor、读取 shim preview/readiness，并校验真实 `.areaflow/status.json` 与
`workflow/README.md` 指纹不变。它不得运行 `project status-projection-apply`、不得传
`--allow-native`、不得改 AreaMatrix shim 文件，也不得执行 `./task-loop run`。

即使真实只读 smoke 通过，`shim-readiness` 在没有 dirty worktree review 与显式编辑授权前仍应保持
`blocked`。这是进入 AreaMatrix 仓库写 shim 文件前的安全门槛，不是功能失败。

AreaMatrix 侧落地后再验证：

```bash
./dev workflow status
./dev workflow doctor
./dev workflow init --version shim-smoke
./dev workflow open
./task-loop status
```

并确认：

```text
./task-loop run is blocked
workflow/versions/**/execution/** unchanged
workflow/versions/**/execution/_shared/progress.json unchanged
.areaflow/status.json unchanged unless explicit export-status was run
workflow/README.md only changed in the approved status block / manual link section
```

## Rollback

Rollback 必须是追加事实，不删除 AreaFlow 历史：

```text
1. Disable shim forwarding by unsetting AREAFLOW_API_URL / AREAFLOW_BIN or toggling the AreaMatrix shim config.
2. Fall back to .areaflow/status.json for read-only status.
3. If the shim itself is faulty, revert only the AreaMatrix shim files after approval.
4. Do not delete AreaFlow events, audit_events, workflow_versions, runs, attempts, or artifacts.
```

Shim retirement 不属于本计划的 implementation 范围。退役旧 runner 前必须按
[`areamatrix-execution-cutover-boundary.md`](./areamatrix-execution-cutover-boundary.md) 证明
`execution_forwarding` 稳定、active execution state 已由 AreaFlow 持有、历史 progress/log/checkpoint 已作为
archive/reference 处理，并保留 rollback 到 `read_only_shim` 的路径。

## Go / No-Go

可以进入 AreaMatrix shim implementation 的条件：

```text
AreaFlow fixture smoke pass
AreaFlow real AreaMatrix read-only smoke pass
AreaFlow web smoke pass
project compatibility contract returns pass or accepted warn
cutover readiness gate behavior remains read-only
AreaMatrix dirty worktree reviewed before edit
explicit user confirmation to edit AreaMatrix
```

不能进入 implementation：

```text
AreaFlow API/CLI contract missing
status projection write boundary unclear
AreaMatrix workflow/execution/progress write risk unresolved
task-loop run expected to execute
no rollback path
no explicit approval to edit AreaMatrix
```
