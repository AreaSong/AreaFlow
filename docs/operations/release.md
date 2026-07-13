# Release

AreaFlow 的 release 页面和 API 汇总 readiness、acceptance、exception、packaging、distribution、publish 与 rollout 的决策事实。

## 当前边界

- Release readiness 和 final gate 是评审输入。
- Package、distribution、publish 和 rollout 的 preview 只描述计划。
- Exception 需要明确 scope、reason、actor、审批和 audit。

Gate pass 不是发布副作用。当前系统不因 release status 自动创建 package、tag、signature、upload、push、publish 或 rollout state。

## 完成审计

Completion Audit 聚合源事实对齐、任务矩阵、验证、迁移、release、backup/restore、operations、安全和 protected path proof。单个 smoke、gate 或 evidence 状态不能替代整体审计。

详见 [Completion Audit](completion-audit.md)。
