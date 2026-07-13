# Backup 与 Restore

AreaFlow 的可恢复状态由 PostgreSQL、artifact store、项目配置引用和版本信息共同组成。

## 当前能力

- Backup Manifest 汇总需要保护的数据库与 artifact metadata。
- Restore Plan 校验恢复输入、目标和阻塞条件。
- Artifact Integrity 校验本地内容的 hash 与 size。
- Archive Preview 计算归档候选和 retention 阻塞原因。

这些接口提供 manifest、plan、校验和 preview，不执行数据库 restore、artifact copy/upload/delete 或 GC。

## 恢复边界

恢复计划必须绑定明确的 backup identity、schema/migration 版本、artifact root 和 project scope。缺失或不匹配时应 fail closed。

真实 restore apply 属于高风险操作，需要独立审批、审计、回滚演练和验证，不能由 `ready` 状态自动触发。
