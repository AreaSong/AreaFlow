# AreaFlow Threat Model

AreaFlow 的生产信任边界是单租户组织控制面。外部负载均衡终止 TLS；AreaFlow 验证企业 OIDC 身份，通过 project RBAC 和 capability 执行授权，并把安全决策追加到 PostgreSQL audit history。

## Assets And Trust Boundaries

- OIDC issuer/subject 是用户身份键，email 只作为可变显示信息。
- 浏览器持有 HttpOnly session cookie 和独立 CSRF cookie；数据库只保存 hash。
- CLI 和受信 worker 使用有到期时间、可撤销、project/capability-scoped service token。
- PostgreSQL 保存平台主状态；S3 保存 AreaFlow-owned artifact content；被管理项目继续拥有自身源码和用户文件。
- OIDC、PostgreSQL、S3、TLS/LB 和 telemetry provider 是企业外部依赖，不因配置存在而视为可信或可用。

## Primary Threats And Controls

| Threat | Control |
|---|---|
| OIDC code interception or login CSRF | Authorization Code、PKCE、state、nonce、短期加密 state cookie |
| Session theft or fixation | 高熵 opaque session、server-side revoke、Secure/HttpOnly/SameSite、idle/absolute expiry |
| Cross-site request forgery | 双提交 CSRF cookie、header 与数据库 hash 三方校验 |
| Mutable identity takeover | `issuer + subject` 唯一绑定，不以 email 识别用户 |
| Cross-project access | 服务端 project visibility、RBAC/capability、越权时返回 404 |
| Privilege escalation | 固定角色、默认拒绝、grant/revoke 审计、高风险申请与审批分离 |
| Token disclosure | 仅创建时返回原文、数据库保存 hash、最长 90 天、支持轮换与撤销 |
| Proxy/header spoofing | production 必须配置可信代理 CIDR，外部入口必须使用 HTTPS |
| Artifact tampering | PostgreSQL metadata、S3 URI、SHA-256、size、S3 server-side encryption 和完整性检查 |
| Audit rewriting | event/audit trigger 拒绝 UPDATE/DELETE，仅允许项目删除时保留快照并置空 FK |
| Secret leakage | 只接收 secret file/reference，日志、event、audit、artifact 和支持包不得出现明文 |
| Dependency compromise | Dependabot、CodeQL、govulncheck、npm audit、Gitleaks、Trivy、SBOM、provenance、Cosign |

## Explicitly Closed

多租户 SaaS、remote worker、通用 engine execution、plugin execution、webhook、secret resolve、生产 restore apply 和发布 apply 不属于 v1 生产信任边界。
