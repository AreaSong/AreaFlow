# Security Architecture

AreaFlow 的安全边界围绕 project isolation、capability、path policy、command、approval 和 append-only audit 建立。

## 默认拒绝

- 未声明 capability 默认关闭。
- Project 默认只读。
- Forbidden path 优先于 write path。
- Preview 和 readiness 不授予 apply 权限。

## 副作用分离

以下能力分别判断，不能相互替代：

- AreaFlow database write。
- AreaFlow artifact write。
- managed project file write。
- project execution path write。
- command execution。
- engine call。
- network access。
- secret resolve。

## Project isolation

Project identity 贯穿 workflow、run、worker、lease、artifact、event 和 audit。按 ID 查询资源时，API 必须校验请求的 project visibility scope。

## Authentication 与 Authorization

- production 强制 `AREAFLOW_AUTH_MODE=oidc`；OIDC issuer、client、redirect URI 和 secret file 缺失时拒绝启动。
- Web session 只保存 opaque hash，使用 `Secure`、`HttpOnly`、`SameSite` cookie、滑动 idle timeout 和 absolute timeout；所有非安全方法要求双提交 CSRF token。
- project RBAC 角色为 `project_admin`、`operator`、`approver`、`auditor`、`viewer`，`platform_admin` 仅允许全局 scope。Role 只产生 capability ceiling，project visibility 仍独立校验。
- service token 只在创建时返回明文，数据库保存 SHA-256；token 绑定 project/capability、创建者、到期时间和 rotation chain，最长 90 天。
- 认证模式下 actor 始终来自 principal，客户端 body 不能冒充。高风险、critical 或 L4 workflow approval 禁止申请人自批。
- forwarded headers 只接受 `AREAFLOW_TRUSTED_PROXY_CIDRS` 内来源；production 的外部 URL 固定来自 `AREAFLOW_PUBLIC_BASE_URL`。

## Secret

当前产品只保存 secret reference 边界，不开放真实 secret resolve。未来实现不得把 secret value 写入配置快照、event、audit、artifact metadata、support bundle 或错误响应。

## Web

Web 默认使用 OIDC cookie session，也保留显式 token mode。Token 只保存在当前标签页的 `sessionStorage`，不会进入 URL 或 `localStorage`；`401` 会清除失效 token。

服务端按 endpoint capability 决策，当前 Web 写面包括 authored workflow approval/rejection 和具备 `auth.role.manage` 的项目角色管理。按钮可见性不是授权边界；project write、engine、secret、publish 和 restore apply 不会因认证启用而自动开放。

`runs`、`events` 和 `audit_events` 保存 `project_key_snapshot`。项目被清理后外键可以置空，但 append-only 历史仍保留原项目归属。

治理规则见 [`../../governance/security/`](../../governance/security/) 和 [`../../governance/permissions/`](../../governance/permissions/)。
