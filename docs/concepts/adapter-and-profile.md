# Adapter 与 Workflow Profile

AreaFlow 使用 Adapter（项目适配器）连接项目，使用 Workflow Profile（工作流配置）描述项目采用的生命周期。两者相互独立，core 不硬编码 AreaMatrix 的目录和流程。

## Adapter

Adapter 负责读取项目配置和允许的项目事实，并把它们映射为 AreaFlow 的 Project、Workflow Version、Artifact、Residual 和状态快照。Adapter 不负责定义 stage、gate 或 transition。

Adapter 的读写范围受以下约束：

- project config 声明的 root、adapter 和 ownership。
- capability 与路径 allowlist。
- forbidden path 和 deny-first 策略。
- Command API、approval 与 audit。

默认操作模式是只读。Adapter 不能仅凭 gate、readiness 或 approval 推导出项目写入权限。

## Workflow Profile

Profile 声明 stage、item state、required artifact、gate、transition、failure route 和最低权限要求。Profile 文件必须可校验、可 hash；创建 AreaFlow-authored Workflow Version 时冻结 profile version/hash。

Profile 更新不会静默改变既有版本。已有版本需要显式迁移后才能绑定新的 profile。

## AreaMatrix

AreaMatrix 是首个内置组合：

```yaml
adapter: areamatrix
workflow_profile: areamatrix
```

声明式 profile 位于 [`workflow/profiles/areamatrix/profile.yaml`](../../workflow/profiles/areamatrix/profile.yaml)，接入规则见 [AreaMatrix Adapter](../reference/adapters/areamatrix.md)。

## 扩展边界

当前 registry 由内置 Adapter 和 Profile 组成。第三方 plugin execution、marketplace 安装和未知代码加载不是当前产品能力，未来设计记录在 [`proposals/plugin-marketplace.md`](../../proposals/plugin-marketplace.md)。
