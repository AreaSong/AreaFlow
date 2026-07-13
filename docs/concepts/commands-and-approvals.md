# Commands And Approvals

AreaFlow 将状态变更建模为可识别、可重放检查、可审计的 command，而不是由 UI 或 worker 直接修改数据库和项目文件。

## Command envelope

写命令至少绑定：

- command type 和目标资源。
- actor 和 reason。
- idempotency key。
- expected state 或 preimage。
- 所需 capability 和 path scope。
- gate、approval 和 audit correlation。

## Approval

Approval 记录 decision、scope、actor、reason 和 risk level。Approval 只对声明的 command scope 生效，不能被另一个项目、路径、run 或 command type 复用。

## Preview、Gate 与 Apply

- Preview 描述计划和可能副作用。
- Gate 判断当前事实是否满足执行条件。
- Apply 才能创建状态变化或外部副作用。

`ready`、`pass`、`eligible` 和 `approved` 均不等于 apply 已发生。

## Idempotency

相同 idempotency key 和相同请求可以安全重放；相同 key 与不同请求必须返回 conflict。客户端不得在失败后随意生成新 key 绕过冲突或审批。

## Audit

每个安全判断记录 audit event。真实状态变化还记录 domain event。任何无法写入所需审计证据的命令必须 fail closed。
