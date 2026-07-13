# AreaMatrix Dogfood Contract

## 关系

AreaMatrix 是 AreaFlow 的第一个 dogfooding 项目。AreaFlow 最终接管 workflow/task-loop 主能力；AreaMatrix 最终只保留项目内容、产品源事实、代码、验证命令、治理边界和粗略进度入口。

## AreaMatrix 保留所有权

```text
docs/**
source code
validation commands
project-specific governance
release evidence
user-file safety boundary
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

## v0.1 允许

- 只读导入 AreaMatrix workflow、residual、task metadata。
- 生成 AreaFlow 数据库记录。
- 显式命令导出 `.areaflow/status.json`。

只读导入深度以
[`../architecture/areamatrix-import-scope-contract.md`](../architecture/areamatrix-import-scope-contract.md)
为准：v0.1 只索引 metadata、hash、path、size、status summary 和少量机器可解析 ledger，不复制历史
prompt、日志、报告、diff、evidence 原文。

## v0.1 禁止

- 写 AreaMatrix 代码。
- 写 `workflow/versions/**`。
- 写 execution、progress、logs、checkpoint 或 release evidence。
- 启动 `./task-loop run`。

## 最终 AreaMatrix 粗略入口

```text
areaflow.yaml
workflow/README.md
.areaflow/status.json
```

详细 workflow 状态在 AreaFlow 中查看。

## Compatibility And Retirement

AreaMatrix compatibility shim 只能按以下顺序推进：

```text
not_installed
-> read_only_shim
-> execution_forwarding
-> retired_thin_entry
```

`read_only_shim` 只允许 status、doctor、open、init preview 等只读或 authoring 转发；`./task-loop run`
必须 blocked。`execution_forwarding` 只能把 `./task-loop run` 转发到 AreaFlow 受保护 Command API，不能启动
旧 runner。`retired_thin_entry` 只能退役旧主执行能力，不能删除历史 workflow、progress、logs、evidence、
release evidence 或 AreaFlow 审计事实。
v0.4 compatibility、shim readiness、DB-only authoring cutover apply 和 rollback 边界见
[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../history/v1.0/contracts/v0.4-workflow-ownership-cutover-contract.md)；
`read_only_shim` 和 `project.cutover.apply` 都不能被解释为 `execution_forwarding`。
v0.6 worker beta 的 scoped execution 边界见
[`../architecture/v0.6-worker-beta-contract.md`](../history/v1.0/contracts/v0.6-worker-beta-contract.md)；fixture、
read-only、artifact-only、fixture/temp rollback drill、readiness 和 beta gate 证据都不能累计成 AreaMatrix
execution cutover 或 `./task-loop run` forwarding。

`.areaflow/status.json` 只允许保存粗略 projection：

```text
project_id
project_name
area_flow_url
cutover_phase
active_versions[]
active_versions[].display_label
active_versions[].lifecycle_status
active_versions[].rough_progress.percent
active_versions[].rough_progress.label
active_versions[].rough_progress.blocked
last_synced_at
source_snapshot_hash
compatibility.shim_lifecycle_state
compatibility.offline_source
compatibility.blocked_commands[]
```

它不得保存完整 queue、execution attempt、logs、checkpoint、approval payload、secret、worker lease 或
artifact 原文。AreaMatrix compatibility commands 可以读取该文件做离线降级展示，但不能把它当作主状态源。
