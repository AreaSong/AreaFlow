# Projects 页面

Projects 展示 AreaFlow 管理的项目注册表和项目详情。

项目详情包括：

- project key、名称、类型和 root。
- adapter、workflow profile 和默认分支。
- 当前配置路径、hash 和加载时间。
- workflow version、artifact、residual 和 mirror inventory。
- readiness 检查及其状态和说明。

项目注册或配置变更目前通过 CLI 和 `areaflow.yaml` 完成；Web 页面不会绕过配置验证直接修改项目边界。
