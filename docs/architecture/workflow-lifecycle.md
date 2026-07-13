# Workflow Lifecycle

## 核心决策

AreaFlow 使用通用 stage engine + workflow profile。

- Core engine 只理解 Stage、Gate、Artifact、Transition、Permission 和 Validation。
- AreaMatrix 当前流程作为第一个内置 profile，不硬编码到 core。

详细对象、状态、门禁、失败回路和 trace 要求见
[`workflow-engine-contract.md`](workflow-engine-contract.md)。

## AreaMatrix Profile v0

AreaMatrix 当前目录名和 AreaFlow stage 名不必完全一致。目录路径只作为 source URI 或 artifact
来源；生命周期判断以 profile stage、gate result 和 transition preview 为准。

| 顺序 | Stage | AreaMatrix 目录 / 输入别名 | 含义 |
|---:|---|---|---|
| 1 | `intake` | `workflow/intake.md`、用户讨论 | 需求入口和范围判断 |
| 2 | `source_docs` | `docs/**` | 产品、API、架构源事实引用 |
| 3 | `templates` | `workflow/templates/**` | workflow 模板和 schema 参考 |
| 4 | `version_init` | `workflow/versions/v*/version.yaml` | 创建版本和生命周期 metadata |
| 5 | `discussion` | `discussion/docs-discussion.md`、`middle-layer-discussion.md`、`decisions.yaml` | docs 讨论、中间层讨论、决策账本 |
| 6 | `middle_layer` | `middle-layer/**` | docs 到 changes/plans/drafts 的映射账本 |
| 7 | `changes` | `changes/**` | docs-change ledger |
| 8 | `plans` | `plans/**` | 人读计划、依赖、风险、验证 |
| 9 | `drafts` | `drafts/**` | manifest / copy-ready / verify-ready 草稿 |
| 10 | `queue` | `queue/**` | version-local queue candidate |
| 11 | `promotion_preview` | `promotion/promotion.*` | live mapping 预演 |
| 12 | `approval` | `promotion/approval.yaml`、approval records | 显式 approval 和 live mapping gate |
| 13 | `execution` | `execution/**` | 批准后的 live execution 材料 |
| 14 | `run` | task-loop progress、logs、checkpoints；AreaFlow `runs/**` | copy / verify / repair / checkpoint |
| 15 | `projection` | `projection/**`、`.areaflow/status.json` | 执行结果投影 |
| 16 | `closeout` | `closeout/**` | 收口、审计、归档 |

## Stage Contract

每个 stage 需要定义：

```text
required_inputs
required_artifacts
allowed_outputs
allowed_transitions
gate_checks
write_permissions
validation_commands
rollback_policy
```

## 关键边界

- `source_docs` 仍由被管理项目拥有。
- `execution` 与 `run` 是后期迁移阶段。
- promotion preview 不等于 live apply。
- discussion gate 未通过不得进入 changes。
- approval record 不等于 execution；它只批准明确 scope 和 transition。
- live mapping gate 必须独立存在，不能藏在 promotion preview 或 approval status 内。
- projection 是外部入口快照，不得作为恢复 run、approval、artifact 或 lease 的主状态。

## Stage Gate Chain

AreaMatrix profile 的最小 gate chain 为：

```text
discussion
  -> discussion_gate
middle_layer
  -> middle_layer_consistency_gate
changes
  -> docs_change_gate
plans
  -> plan_readiness_gate
drafts
  -> draft_readiness_gate
queue
  -> queue_readiness_gate
promotion_preview
  -> promotion_preview_gate
approval
  -> approval_gate
  -> live_mapping_gate
execution
  -> execution_permission_gate
run
  -> verify_acceptance_gate
  -> checkpoint_gate
projection
  -> projection_integrity_gate
closeout
  -> closeout_gate
```

Gate pass 只证明当前 transition 的前置条件满足。真实写入、execution、cutover apply、restore apply 或
publish apply 仍必须通过 Command API、permission evaluator、approval 和 audit。
