# AreaMatrix Workflow Profile

AreaMatrix profile 是 AreaFlow 的第一个内置 workflow profile。它把 AreaMatrix 当前 workflow lifecycle 映射到通用 stage engine。

声明式 profile 源事实见 [`profile.yaml`](profile.yaml)。Go checker 负责执行 gate、transition
和 artifact 校验；PostgreSQL 记录每个 workflow version 绑定的 profile version/hash。
AreaFlow-authored version 的 profile binding、gate、transition preview 和 approval 语义见
[`../../../docs/concepts/workflow-lifecycle.md`](../../../docs/concepts/workflow-lifecycle.md)。

## Stages

```text
intake
source_docs
templates
version_init
discussion
middle_layer
changes
plans
drafts
queue
promotion_preview
approval
execution
run
projection
closeout
```

## 当前导入边界

Adapter 导入 stage metadata、artifact path/hash、residual、task index 和 status snapshot。Profile 本身只声明 lifecycle，不读取磁盘、不执行 transition，也不授予项目写入能力。

## 版本绑定

AreaFlow-authored workflow version 创建时必须冻结当前 profile hash。后续 profile 升级只能显式迁移，
不能静默改变既有 workflow version 的 gate 或 transition 规则。

## 安全不变量

本 profile 只声明 AreaMatrix workflow 的 stage、gate、transition、artifact policy 和最低写入前置条件。
它不读取磁盘、不运行命令、不解析 secret、不授予 capability，也不决定真实 apply。

profile 加载校验会拒绝不安全的写入策略：

```text
permissions.default_mode = readonly
permissions.write_requires:
  capability
  path_allowlist
  gate_result
  approval_record
  audit_event
```

因此，`promotion_preview`、`approval`、`live_mapping_gate` 或 `runner_gate` 的通过都只能说明下一阶段
前置条件满足；写 AreaMatrix、写 generated projection、执行 `./task-loop` 或调用 engine 仍必须经过
project config、permission evaluator、Command API、approval scope 和 audit event。
