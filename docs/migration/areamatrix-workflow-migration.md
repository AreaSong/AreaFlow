# AreaMatrix Workflow Migration

## 策略

AreaMatrix workflow 到 AreaFlow 的迁移采用：

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

短名仍可写作 `Import -> Mirror -> Shadow -> Cutover -> Archive`，但设计上必须区分
authoring cutover 和 execution cutover。v0.4 只切新 workflow version authoring，不替代
`./task-loop run`。

## Import

AreaFlow 读取 AreaMatrix workflow、residual、task metadata，写入 PostgreSQL，不改变 AreaMatrix。
具体 read envelope、minimum import set、explicit non-imports 和 artifact metadata 策略见
[`../architecture/areamatrix-import-scope-contract.md`](../architecture/areamatrix-import-scope-contract.md)。

v1 历史只导入 index、hash、path 和 metadata，不搬原始 execution 文件，不重写 `progress.json`。

Import 允许：

```text
读取 workflow/versions/**
读取 workflow/residuals/**
读取 progress.json / logs / evidence metadata
计算 hash / size / source_path
写入 AreaFlow PostgreSQL
写入 AreaFlow artifact metadata
```

Import 禁止：

```text
修改 AreaMatrix 文件
创建 execution
修改 progress.json
重排历史目录
更改 task-loop 行为
```

## Mirror

AreaFlow 可以显式导出粗略状态到：

```text
AreaMatrix/.areaflow/status.json
```

人读摘要可后续写入 `workflow/README.md` 中的受控区块。

Mirror 阶段 `.areaflow/status.json` 是工具读 projection，不是主状态。详细 run、gate、artifact、
approval 和 audit 仍只在 AreaFlow 中查询。写 status projection 必须经过 `write_status` capability、
path allowlist、projection gate 和 audit event。

## Shadow

AreaFlow 模拟 workflow doctor、stage gate、promotion preview 和 drift check。AreaMatrix 仍是源事实。
v0.2 shadow doctor 的最小闭环、状态语义、readiness bundle、import diff、verification bundle 和 native
doctor 授权边界见
[`../architecture/v0.2-shadow-doctor-contract.md`](../architecture/v0.2-shadow-doctor-contract.md)。

Shadow 允许创建 shadow job、shadow run、read-only doctor evidence、promotion preview 和 execution preview。
Shadow 禁止真实写 `execution/**`、禁止执行 copy-ready 改代码、禁止修改 `progress.json`。

## Authoring Cutover

新 workflow version 由 AreaFlow 创建和管理。AreaMatrix 保留兼容命令和粗略入口。
v0.3 New Version Authoring 只创建 AreaFlow-owned authored records、stage skeleton、gate、transition preview 和
approval record；具体 no-apply 边界见
[`../architecture/v0.3-version-authoring-contract.md`](../architecture/v0.3-version-authoring-contract.md)。
Cutover 不是搬目录，而是切换新 workflow version 的源事实所有权；详细 rollback 与兼容命令规则见
[`cutover-rollback-compat.md`](cutover-rollback-compat.md)。v0.4 的 no-execution / no-project-write
合同见
[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../architecture/v0.4-workflow-ownership-cutover-contract.md)。

v0.4 Cutover 必须通过 import coverage、status mirror、hash drift、stage coverage、doctor 等价、
version authoring、transition preview、approval gate、live mapping gate、compatibility shim、
rollback plan、write permission、audit events 和 cutover readiness gate。它只切新 workflow version
authoring 所有权；promotion apply、execution start、task-loop replacement 和真实 execution cutover
必须等 runner/worker 权限模型完成后再单独批准。

`areaflow project cutover-apply <project> --version <label>` 只执行 AreaFlow DB 内 authoring cutover：
将已通过 gate 的新 workflow version 标记为 `authoring_cutover`，写入 command/event/audit 事实，
并明确证明没有写项目文件或 execution 状态。它不是 AreaMatrix shim 修改，也不是 execution cutover。

Authoring cutover 允许：

```text
AreaFlow 创建新 workflow version
AreaFlow 管理 discussion / changes / plans / drafts / queue / promotion_preview / approval
AreaMatrix 只显示粗略进度、链接和兼容入口
```

AreaMatrix 在 authoring cutover 后的保留入口为：

```text
areaflow.yaml:
  项目级 AreaFlow 配置、project id、allowlist、adapter/profile 引用。

.areaflow/status.json:
  工具读粗略状态，只包含 project、cutover_phase、active_versions、rough_progress、dashboard URL、
  last_synced_at、source_snapshot_hash 和 compatibility blocked commands。

workflow/README.md:
  人读入口，说明 workflow 主状态已迁移、当前粗略阶段和 AreaFlow dashboard 链接。
```

这些入口不得重新承载完整 plans、drafts、queue、execution、progress、logs、checkpoint 或 artifact
metadata。详细状态只能查询 AreaFlow PostgreSQL / API。

Authoring cutover 禁止：

```text
转发 ./task-loop run
写 workflow/versions/**/execution/**
修改 progress.json
自动改代码
自动 git checkpoint
```

## Execution Beta

Execution beta 只在 runner / worker / permission / approval / audit 已证明后打开。v0.5 runner preview
边界见 [`../architecture/v0.5-runner-preview-contract.md`](../architecture/v0.5-runner-preview-contract.md)，它只
建立 dry-run run/task/attempt/artifact/audit 证据，不打开真实 execution。v0.6 worker beta 的 worker
lifecycle、lease、dry-run run-once、scoped execution evidence 和 no-cutover 口径见
[`../architecture/v0.6-worker-beta-contract.md`](../architecture/v0.6-worker-beta-contract.md)。建议顺序：

```text
runner preview
-> worker dry-run
-> fixture execution
-> read-only verify on AreaMatrix
-> approved artifact write
-> approved project write
-> checkpoint
-> repair
```

AreaMatrix 第一批真实 execution 应优先选择 `verify-ready`、doctor/readiness 和 artifact evidence，
不应直接打开 copy-ready 自动改代码、repair 自动修复或 git checkpoint。
v0.6 的 fixture、read-only verify、AreaFlow-owned artifact write、fixture/temp rollback drill、readiness 和
beta gate 证据必须按各自 scope 解释；它们不能累计成 `./task-loop run` forwarding 或 AreaMatrix execution
cutover approval。

## Execution Cutover

Execution cutover 表示 AreaFlow 成为 workflow/task-loop execution 主入口。通过标准：

```text
runner/worker lease 可恢复
copy / verify / repair / checkpoint 分离
failure / retry / cancel / drain 有审计
artifact evidence 完整
approval / capability / path allowlist 全部生效
compatibility command 可转发或清晰 blocked
rollback 能回到 read-only / status projection 模式
```

`./task-loop run` 在 execution cutover 前必须 blocked；cutover 后也应先打印迁移说明或转发到
AreaFlow command，不得恢复旧 progress/log/lock 主状态。

详细命令映射、apply 前置缺口、protected paths、rollback 和 go/no-go 规则见
[`areamatrix-execution-cutover-boundary.md`](areamatrix-execution-cutover-boundary.md)。在该边界未通过前，
`execution-cutover-readiness` 只能作为只读观察面，不能解释为 `./task-loop run` 已可转发。
Execution cutover 后的第一阶段状态是 `execution_forwarding`：AreaMatrix 入口只转发受保护 AreaFlow
Command API，旧 runner、旧 progress/log/checkpoint 写入保持关闭。它仍不是 shim retirement。
第一版 `execution_forwarding` 只能转发 read-only verify、doctor/readiness、artifact evidence、
status/projection validation 和 release/readiness check 类任务；copy-ready source write、generated retained
write、repair、checkpoint、engine、secret、network、publish 和 restore 必须继续作为后续独立门禁。

## Archive

历史 v1-mvp 保持 immutable import。长期可以将原始 artifact 副本迁入 AreaFlow artifact store 或对象存储，但不得回填、重写或补造历史。

Archive 保留 AreaMatrix 的项目文档、源码、验证命令、治理规则、发布证据、用户文件安全边界、
`workflow/README.md` 粗略入口、`.areaflow/status.json` 工具入口和 compatibility shim。历史主索引、
workflow versions、task execution、artifact metadata、worker scheduling 和 audit 由 AreaFlow 拥有。

Archive 不等于删除历史。默认策略是：

```text
标记 archived
冻结引用
保留 metadata 和 external project reference
不大规模移动 execution
不清空 progress / logs / evidence
```

## Shim Retirement

Shim retirement 表示 AreaMatrix 只保留轻量入口，不再拥有 workflow/task-loop 主能力。可保留：

```text
.areaflow/status.json
workflow/README.md 的 AreaFlow 入口
./dev workflow open/status 的极薄转发
```

可退役：

```text
AreaMatrix 本地 workflow planning 逻辑
旧 task-loop 主执行逻辑
本地 progress 聚合
本地 queue 管理
直接读写旧 logs / locks / checkpoints 的命令
workflow/versions/** 主副本维护
workflow/versions/**/plans、drafts、queue、projection、closeout 的新写入
progress.json、run summaries、runner lock 的主状态语义
```

不能退役：

```text
AreaMatrix 产品 docs
源码
测试和验证命令
发布证据
治理规则
用户文件安全规则
```

Shim retirement 必须晚于 execution forwarding 稳定期。它只能退役旧主执行能力，不能删除历史
`workflow/versions/**`、`progress.json`、logs、evidence、release evidence 或 AreaFlow events/audit/runs。
退役后 `./task-loop run` 可以保持 blocked with migration notice，或继续 thin forward 到 AreaFlow；两者都
不得恢复旧 runner。详细 gate 见
[`areamatrix-execution-cutover-boundary.md`](areamatrix-execution-cutover-boundary.md)。

## Cutover 门槛

```text
import coverage 100%
hash drift check 可用
AreaFlow doctor 等价通过
粗略状态回写稳定
权限 allowlist 生效
audit_events 完整记录写入
compat command 设计完成
rollback plan 明确
```
