# Data Model

PostgreSQL 是 AreaFlow 的主状态源事实。当前 schema 由 `migrations/000001` 至 `000020` 顺序建立，应用不得跳过 migration ledger、checksum 校验或用文件状态替代数据库状态。

## 领域分组

| 分组 | 主要实体 |
|---|---|
| Project | projects、project configs、imports、status projections |
| Workflow | workflow versions、stages、items、item links、gate results、transition previews、approvals |
| Execution | runs、run tasks、attempts、workers、heartbeats、leases、scheduling policies |
| Artifact | artifacts、artifact locations、artifact relations、integrity/retention metadata |
| Command | command requests、idempotency、risk/permission/approval facts |
| History | events、audit events |
| Operations | migration ledger、release exceptions、completion/operations proof records |
| Identity | users、OIDC identities、project role bindings、Web sessions、service token lifecycle |

具体列和约束以 SQL migration 为准，API 文档不复制完整 schema。

## 标识与隔离

内部关系使用稳定 ID，外部路由使用 `project_key`、version、worker key 等稳定业务标识。所有 project-scoped 查询和写入必须显式携带 project scope，不能仅依赖客户端过滤。

## 状态与历史

领域主表保存当前状态；Event 和 Audit Event 追加历史事实。数据库 trigger 拒绝 UPDATE/DELETE，只允许项目删除产生的 `project_id -> NULL` 外键动作。Run retry、Attempt、Approval 和 Command Request 创建新记录，不覆盖旧证据。

## 大内容

数据库保存 artifact metadata、hash、size、URI 和关联关系。Prompt、日志、报告等大内容保存在 artifact store；secret value 不进入数据库 metadata、event 或 audit。

`artifacts` 保存当前 primary 位置，`artifact_locations` 保存迁移 source、candidate 和 primary 历史位置。Location 切换只更新当前指针和追加审计，源内容不由迁移命令删除；观察期读取可回退到保留 source。

## Migration

Migration 文件一旦进入已发布共享历史不原地重写。结构变化通过新的有序 migration 完成，并由 migration ledger、SHA-256 checksum 和隔离升级 smoke 证明已应用版本。数据库回退采用备份恢复或经审批的前向修复，不开放破坏性 down migration。
