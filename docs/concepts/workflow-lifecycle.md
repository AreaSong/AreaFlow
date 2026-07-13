# Workflow Lifecycle

AreaFlow 使用通用 stage engine 管理 Workflow Version。Core 只理解 Stage、Workflow Item、Gate Result、Transition Preview、Approval、Artifact 和 Event；项目特有语义由 Workflow Profile 提供。

## 对象关系

```text
Workflow Version
  -> Stage
    -> Workflow Item
    -> Gate Result
    -> Transition Preview
    -> Approval
  -> Run
  -> Artifact
```

Workflow Version 在创建时冻结 profile version/hash。导入并标记 immutable 的历史版本不能被覆盖更新。

## 状态与推进

Workflow Item 使用 profile 允许的状态，例如 `draft`、`ready`、`blocked`、`deferred`、`promoted`、`running`、`done` 和 `superseded`。

推进顺序保持显式分层：

```text
item state
  -> gate result
  -> transition preview
  -> approval
  -> command apply
  -> event / audit event
```

Gate 只判断条件，Transition Preview 只解释可能的推进，Approval 只记录授权事实。三者都不等于 apply，也不等于 execution 已完成。

## Trace

Workflow Item、Run、Artifact 和 Event 通过稳定 ID、source URI、hash 和关联记录形成 trace。历史事实追加到 event/audit，不通过覆盖旧记录制造新的完成状态。

## AreaMatrix Profile

AreaMatrix profile 当前声明 `intake`、`source_docs`、`templates`、`version_init`、`discussion`、`middle_layer`、`changes`、`plans`、`drafts`、`queue`、`promotion_preview`、`approval`、`execution`、`run`、`projection` 和 `closeout` 等 stage。

该 stage 集合是 AreaMatrix profile 的事实，不是所有项目必须采用的硬编码流程。
