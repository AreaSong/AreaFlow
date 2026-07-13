# AreaFlow 产品模型

AreaFlow 管理 AI 软件开发过程中的项目、workflow、执行资源和审计证据。平台不替代项目自身的产品文档、源码语义和验证规则，而是记录这些规则如何被编排、执行和证明。

## 资源关系

```text
Project
  -> Workflow Version
    -> Workflow Item
    -> Gate / Transition Preview / Approval
    -> Run
      -> Run Task
        -> Attempt
      -> Artifact

Worker -> Lease -> Run Task

Project / Workflow / Run / Worker / Artifact
  -> Event
  -> Audit Event
```

## Project

Project 是 AreaFlow 的管理边界。它保存项目路径、adapter、workflow profile、默认分支和配置快照。项目配置声明 AreaFlow 可以读取、写入和执行哪些能力。

## Workflow Version

Workflow Version 是冻结 profile 版本和 hash 的工作流实例。它包含 stage 中的 workflow item，以及 gate、transition preview 和 approval 记录。已标记 immutable 的导入版本不得被静默改写。

## Run、Task 与 Attempt

- Run 表示一次执行会话。
- Run Task 是 worker 可以领取的最小调度单元。
- Attempt 表示对一个 task 的一次实际尝试。
- Lease 绑定 worker 与 task，并提供超时和恢复边界。

run control 可以改变 AreaFlow 中的 run 状态，但只有明确支持的任务类型和授权链才能产生外部副作用。

## Artifact

Artifact 保存执行输入、输出和证据的 metadata、hash、URI 与关联关系。大内容保存在 artifact store，PostgreSQL 保存索引。Artifact 不能替代被管理项目文件的所有权。

## Event 与 Audit Event

Event 记录领域中发生的事实。Audit Event 记录安全判断，包括 actor、capability、resource、decision 和 reason。两者采用 append-only 思路，历史事实不应通过覆盖更新来伪造。

## 权限模型

项目写入至少需要同时满足：

```text
project config
  + capability
  + path allowlist
  + gate result
  + approval record
  + audit event
```

deny 和 forbidden path 优先于 allow。未配置的高风险能力默认关闭。
