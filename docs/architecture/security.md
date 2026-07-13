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

Web 当前通过 write action gate 保持读操作优先。后端写 endpoint 的存在不代表浏览器可以直接调用；开放前必须统一 confirmation、idempotency、approval 和 audit 契约。

治理规则见 [`../../governance/security/`](../../governance/security/) 和 [`../../governance/permissions/`](../../governance/permissions/)。
