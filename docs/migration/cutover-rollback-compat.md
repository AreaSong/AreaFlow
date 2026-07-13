# Cutover / Rollback / Compatibility

## 定位

Cutover 不是搬目录，而是切换新 workflow version 的源事实所有权。AreaMatrix 历史版本保持
immutable，AreaMatrix 保留项目内容和粗略入口，AreaFlow 接管新 workflow 的主状态。
v0.4 的 compatibility、shim readiness、cutover readiness、DB-only authoring cutover apply 和 rollback
边界见
[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../architecture/v0.4-workflow-ownership-cutover-contract.md)。

迁移阶段保持：

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

## Ownership Mode

项目配置和数据库状态都应表达 ownership mode：

```text
import
mirror
shadow
cutover
archived
```

示例：

```yaml
ownership:
  mode: shadow
  source_of_truth:
    product_docs: project
    source_code: project
    workflow: project
    execution: project
    status_summary: areaflow
  cutover:
    enabled: false
    new_versions_owned_by: project
    legacy_versions_readonly: false
```

Cutover 后：

```yaml
ownership:
  mode: cutover
  source_of_truth:
    product_docs: project
    source_code: project
    workflow: areaflow
    execution: project
    status_summary: areaflow
  cutover:
    enabled: true
    new_versions_owned_by: areaflow
    legacy_versions_readonly: true
```

v0.4 只切新 workflow version 的 authoring 所有权，不替代 task-loop，也不代表 execution cutover。

```text
v0.4 cutover =
  新 workflow version authoring 的源事实从 AreaMatrix 目录切到 AreaFlow PostgreSQL

not:
  task-loop cutover
  execution package apply
  自动写 AreaMatrix 代码
  自动写 workflow/versions/**/execution/**
```

## Cutover Gate

Cutover 前必须满足：

```text
1. import_coverage pass
2. status_mirror pass
3. hash_drift pass
4. stage_coverage pass 或 warn 可解释
5. workflow doctor equivalent pass
6. version_authoring pass
7. transition_preview pass
8. approval_gate pass
9. live_mapping_gate pass
10. compatibility_shim pass
11. rollback_plan pass
12. write_permission pass
13. audit_events pass
14. cutover_readiness_gate pass
```

`warn` 只能是已接受的历史例外，不能是未知缺口。

AreaFlow v0.4c 提供 cutover readiness 源事实：

```text
areaflow project cutover-readiness <project> --version <label>
GET /api/v1/projects/{project_key}/cutover-readiness?version=<label>
areaflow workflow gate run <project> <label> cutover_readiness_gate
areaflow project cutover-apply <project> --version <label>
POST /api/v1/projects/{project_key}/cutover-apply
```

该 bundle/gate 只读聚合 verification、compatibility、approval、live mapping 和 rollback plan；
它不执行 cutover apply，不写 `execution/**`，不替代 task-loop。

`project cutover-apply` 是 v0.4 authoring cutover 的受保护 Command API 入口。它只在
cutover readiness 和 `cutover_readiness_gate` 通过后更新 AreaFlow PostgreSQL 中该
workflow version 的 authoring cutover 状态，记录 `project.cutover.apply.*` event、
`audit_events` 和 `command_requests` response。它必须显式返回：

```text
project_write_attempted=false
execution_write_attempted=false
```

它不写 AreaMatrix `workflow/README.md`、不写 `.areaflow/status.json`、不创建
`workflow/versions/**/execution/**`，也不替代 task-loop。真实 AreaMatrix shim 修改和 execution cutover
仍需后续单独授权。

## Compatibility Commands

AreaMatrix 保留轻量 shim：

```text
./dev workflow status
./dev workflow doctor
./dev workflow init --version <version>
./dev workflow open
./task-loop status
```

Cutover 后这些命令优先调用 AreaFlow CLI/API；失败时读取 `.areaflow/status.json` 显示降级说明。

兼容命令不得维护第二套 workflow 状态。

第一版 shim 只覆盖上述最小入口。`./dev workflow init --version <version>` 不带 `--write` 时必须保持
preview 语义；只有带 `--write` 时才允许转发到 AreaFlow Command API 创建 authored workflow version，
且仍不得写 AreaMatrix `workflow/versions/**`。

`workflow discuss`、`workflow check-template` 和 `./task-loop check` 可以作为 v0.4b 只读扩展入口，
不得混入第一版 shim 的最小落地范围。

兼容命令分三类：

```text
A. 长期保留在 AreaMatrix:
   cargo / xcodebuild / app build
   core tests
   macOS app tests
   docs source-of-truth checks
   AreaMatrix file safety checks

B. 迁移到 AreaFlow，AreaMatrix 保留转发:
   ./dev workflow status
   ./dev workflow doctor
   ./dev workflow init
   ./dev workflow discuss
   ./dev workflow check-template
   ./task-loop status
   ./task-loop check

C. 最终退役:
   直接操作 progress.json 的命令
   直接读写旧 logs / locks / checkpoints 的命令
   旧 prompt pipeline 的 plan / render / status 专用入口
```

`./task-loop run` 不在 v0.4 自动转发范围内，必须等 execution cutover boundary 的
`read_only_shim -> execution_forwarding` gate 通过后，再进入 explicit approval 流程。Execution
forwarding 仍只能转发 AreaFlow Command API，不能恢复旧 runner。

AreaFlow v0.4b 提供 contract 源事实：

```text
areaflow project compatibility <project>
GET /api/v1/projects/{project_key}/compatibility
```

每条命令输出：

```text
command
mode: forward | fallback_status | blocked
status: pass | warn | fail
areaflow_target
fallback
blocked_reason
metadata
```

`.areaflow/status.json` 只保存轻量 compatibility summary，用于 AreaFlow 不可用时的离线降级展示。

## 禁止自动转发

以下命令或能力不得在 v0.4 自动转发执行：

```text
./task-loop run
promotion apply
write execution
git checkpoint
delete/archive historical workflow
```

这些能力必须等 v0.5/v0.6 execution/worker 权限模型完整后，通过 explicit approval 执行。

## Rollback

Rollback 是追加事实，不删除历史。

```text
pre-cutover fail:
  gate 没过，还没有切
  停在 shadow / mirror，不做写入

cutover apply fail:
  approval 后尝试切换失败
  记录 cutover_failed event/audit，保持 AreaMatrix project-owned

post-cutover drift:
  切过去后发现 AreaMatrix / AreaFlow 状态漂移
  冻结新 authoring，跑 recovery doctor，必要时 soft rollback

soft rollback:
  新 workflow version 切回 project-owned 或 read-only frozen
  AreaFlow 保留 event/audit/artifact
  AreaMatrix 继续用兼容入口查看状态

hard rollback:
  只在 cutover 尚未写入项目文件、未开始执行时允许
  标记 cutover attempt failed
  不删除历史 event/audit/artifact
```

已经生成的 AreaFlow events、audit_events 和 artifacts 不应删除。

## AreaMatrix 最终保留

```text
docs/**
source code
validation commands
project governance
release evidence
user-file safety boundary
workflow/README.md 粗略入口
.areaflow/status.json 工具入口
compat commands
```

## AreaFlow 最终拥有

```text
workflow versions
discussion gates
middle-layer
changes
plans
drafts
queue
promotion preview
approval
execution runs
attempts
events
artifact index
worker scheduling
```
