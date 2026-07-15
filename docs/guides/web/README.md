# Web 控制台

AreaFlow Web 使用统一的项目上下文和九个资源页面。页面状态保存在 URL query 或资源路径中，可以刷新、后退和分享。无效资源路径会显示明确的 Not Found 页面，不会静默跳回 Overview；资源详情路径中的项目与全局项目上下文保持一致。

## 页面

- [Overview](overview.md)
- [Projects](projects.md)
- [Workflows](workflows.md)
- [Runs](runs.md)
- [Workers](workers.md)
- [Artifacts](artifacts.md)
- [Audit](audit.md)
- [Operations](operations.md)
- Access：具备 `auth.role.manage` 的用户查看、授予和撤销项目角色；无权限时显示明确空状态。

## 项目上下文

左侧 `Project context` 控制当前页面读取哪个项目。Projects、Workflows、Runs、Workers、Artifacts 和 Audit 都以该项目为默认过滤条件。

页面不会通过切换项目直接执行写操作。当前 Web 操作仍遵循 `web/write-action-gate` 返回的能力和阻塞原因。

production Web 使用 OIDC 登录和 server-side cookie session；token mode 只在显式选择时使用，token 仅保存在当前标签页。写请求自动携带 CSRF header；session 失效、无权限和跨项目访问分别表现为登录、无权限状态或 Not Found。
