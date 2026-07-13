# Security Governance

- Secret value 不得写入日志、event、audit event、artifact metadata 或 support bundle。
- Network、secret resolve、engine execution、project write 和 execution write 分别授权。
- 高风险能力默认关闭，不能由 schema 表存在或 readiness 状态推断为已开放。
- 被管理项目的用户文件、源码和 execution 路径遵守项目自身安全边界。
