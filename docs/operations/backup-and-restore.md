# Backup 与 Restore

AreaFlow 的可恢复状态由 PostgreSQL、artifact store、项目配置引用和版本信息共同组成。

## 当前能力

- Backup Manifest 汇总需要保护的数据库与 artifact metadata。
- Restore Plan 校验恢复输入、目标和阻塞条件。
- Artifact Integrity 校验本地内容的 hash 与 size。
- S3 artifact 读写强制 bucket boundary、SHA-256 checksum 和 AES256 server-side encryption。
- Archive Preview 计算归档候选和 retention 阻塞原因。
- `backup create` 生成 PostgreSQL custom-format dump、可用 local artifact bytes、逐项 hash/size 和 manifest SHA-256。
- `backup drill` 校验整个 package，并恢复到独立 PostgreSQL database 和隔离 artifact root；结果写入 source DB audit。

`external_project` 等引用只记录 metadata 和限制，不宣称已备份原文。缺失 local artifact 会使 package 为 `needs_attention`，并阻断 drill。当前 backup package 不复制 S3 object bytes；production 必须依赖 bucket versioning、生命周期、独立 inventory 和版本恢复演练。

## 恢复边界

恢复计划必须绑定明确的 backup identity、schema/migration 版本、artifact root 和 project scope。缺失或不匹配时应 fail closed。

生产 restore apply 仍未开放。当前 drill 不覆盖生产数据库、不删除现有状态，也不切换 artifact root。生产恢复需要独立审批、preimage backup、隔离验证和切换方案，不能由 drill `pass` 自动触发。

Backup 必须在 AreaFlow writers 停止后使用 `--quiesced` 创建。Docker PostgreSQL 场景会优先使用同容器内的 `pg_dump`/`pg_restore`，避免 client/server major version 不兼容。

production 目标为 RTO 4 小时、RPO 1 小时。仓库 smoke 只能验证隔离 PostgreSQL restore 和对象读写；HA PostgreSQL PITR 与真实 S3 version restore 必须由企业环境提供证据。
