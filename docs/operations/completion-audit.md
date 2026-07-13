# Completion Audit

Completion Audit 是 AreaFlow 对发布候选进行的只读完整性审计。它聚合多个独立 evidence class，检查当前代码、配置、验证和迁移事实是否共同支持完成声明。

## Evidence Classes

审计覆盖源事实对齐、任务矩阵、fresh validation、AreaMatrix cutover/archive/shim、release packaging、backup/restore、operations、安全闭环和 protected path proof。

每项 evidence 必须绑定 scope、hash、时间、review metadata 和可追溯事件。Fixture、mock、demo 或 synthetic evidence 不能冒充真实 release candidate evidence。

## 结果解释

- `complete`：当前 scope 的全部必要证据已满足。
- `incomplete`：证据缺失、过期或不覆盖当前 scope。
- `blocked`：存在明确不允许放行的事实。

Completion Audit 不运行测试、不写 AreaMatrix、不创建 release package，也不执行 restore、publish 或 rollout。它消费已有证据并 fail closed。

历史 v1.0 完成合同和命令细节保存在 [`docs/history/v1.0/contracts/completion-audit-contract.md`](../history/v1.0/contracts/completion-audit-contract.md)。
