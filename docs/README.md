# AreaFlow 文档

本目录记录 AreaFlow 当前有效、需要长期维护的产品事实。功能、页面、API、CLI 或配置发生变化时，对应文档必须在同一变更中更新。

阶段计划、milestone、版本实施合同、迁移过程和验证 evidence 不属于当前产品说明，统一保存在 `docs/history/**`。尚未实现的能力进入 roadmap 或 proposal，不得写成当前功能。

## 开始使用

- [安装 AreaFlow](getting-started/installation.md)
- [Quickstart](getting-started/quickstart.md)

## 产品概念

- [产品模型](concepts/product-model.md)
- [Adapter 与 Workflow Profile](concepts/adapter-and-profile.md)
- [Workflow lifecycle](concepts/workflow-lifecycle.md)
- [Execution model](concepts/execution-model.md)
- [Worker scheduling](concepts/worker-scheduling.md)
- [Artifacts](concepts/artifacts.md)
- [Command 与 Approval](concepts/commands-and-approvals.md)
- [项目配置](reference/configuration.md)
- [权限与安全](architecture/security.md)
- [数据模型](architecture/data-model.md)
- [API Reference](reference/api.md)
- [CLI Reference](reference/cli.md)

## 使用指南

- [Web 控制台](guides/web/README.md)
- [AreaMatrix workflow profile](../workflow/profiles/areamatrix/README.md)

## 维护与运维

- [架构总览](architecture/overview.md)
- [部署](operations/deployment.md)
- [可观测性](operations/observability.md)
- [Backup 与 Restore](operations/backup-and-restore.md)
- [Support Bundle](operations/support-bundle.md)
- [Release](operations/release.md)
- [Completion Audit](operations/completion-audit.md)
- [开发环境](development/setup.md)
- [品牌素材](../assets/brand/README.md)
- [ADR](adr/)
- [治理边界](../governance/README.md)

## 未来方向（非当前能力）

以下内容用于评审尚未实现的方向，不代表 AreaFlow 已经开放对应能力：

- [路线图](roadmap.md)
- [未来设计 proposals](../proposals/README.md)

## 文档源事实规则

1. 当前行为以代码、数据库 migration、API/CLI 契约和已通过的验证共同证明。
2. `docs/**` 解释当前怎么使用、系统怎么工作以及长期不变量。
3. ADR 解释关键决策为什么成立，不充当用户指南。
4. roadmap 和 proposal 只描述未来，不得被当前功能页引用为可用能力。
5. history 只读保留历史上下文，不参与当前产品导航。
6. 默认更新现有源事实；不得为单个功能创建 plan、progress、evidence、completion 等阶段文档。
7. 功能指南描述用途、入口、前置条件、可执行操作、结果、失败与权限表现、当前限制及必要的 API/audit 关联，不记录实现过程。
