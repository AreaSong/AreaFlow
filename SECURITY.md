# Security Policy

## 支持范围

当前安全边界以 [`docs/architecture/security.md`](docs/architecture/security.md)、[`docs/architecture/threat-model.md`](docs/architecture/threat-model.md) 和 [`governance/security/README.md`](governance/security/README.md) 为准。production 强制企业 OIDC、HTTPS public URL、可信代理、PostgreSQL TLS、S3 和 OTLP；Web session、project RBAC 与 scoped service token 已开放。TLS/LB、OIDC tenant、secret manager、HA PostgreSQL 和监控平台由部署组织负责。Secret resolve、remote workers、webhooks 和 plugin execution 尚未开放。

## 报告漏洞

请使用 GitHub Private Vulnerability Reporting（仓库启用后）或仓库所有者提供的私有联系渠道报告漏洞。不要在公开 Issue、Discussion、日志或 artifact 中提交漏洞细节、凭据、token、secret 或用户数据。

报告应包含受影响版本或 commit、复现条件、影响范围和可用的缓解措施。维护者确认问题前，请避免公开利用细节。
