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

## Secret

当前产品只保存 secret reference 边界，不开放真实 secret resolve。未来实现不得把 secret value 写入配置快照、event、audit、artifact metadata、support bundle 或错误响应。

## Web

Web 支持本机 token 登录，token 只保存在当前标签页的 `sessionStorage`，不会进入 URL 或 `localStorage`。`401` 会清除失效 token。

当前 token 写面只开放 authored workflow 的 approval/rejection。服务端强制 project/capability scope、principal actor 和 idempotency；按钮可见性不是授权边界。Run control、project write、worker command、engine、secret、publish 和 restore apply 不会因 token 认证启用而自动开放。

`runs`、`events` 和 `audit_events` 保存 `project_key_snapshot`。项目被清理后外键可以置空，但 append-only 历史仍保留原项目归属。

治理规则见 [`../../governance/security/`](../../governance/security/) 和 [`../../governance/permissions/`](../../governance/permissions/)。
