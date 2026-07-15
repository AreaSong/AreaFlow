# Release

AreaFlow 的 release 页面和 API 汇总 readiness、acceptance、exception、packaging、distribution、publish 与 rollout 的决策事实。

## 当前边界

- Release readiness 和 final gate 是评审输入。
- Package、distribution、publish 和 rollout 的 preview 只描述计划。
- Exception 需要明确 scope、reason、actor、审批和 audit。

Gate pass 不是发布副作用。当前系统不因 release status 自动创建 package、tag、signature、upload、push、publish 或 rollout state。

## Repository Release Pipeline

tag `v*` 触发的 GitHub Actions workflow 会验证版本与许可证契约，构建多平台 CLI、Web、Desktop 和多架构容器，生成 SPDX 与 CycloneDX SBOM，执行 Trivy scan，为全部 release assets 生成 SHA-256 checksums，使用 keyless cosign 签名容器并生成 provenance，最后在 `production-release` environment 后创建 draft release。真实 tag、push、GitHub environment 审批和公开发布必须获得外部授权。

release candidate 必须先通过 `make release-check`、隔离数据库/auth/S3/HA smoke 和回滚演练。灰度顺序固定为 canary 单副本、观察核心 SLI、扩至其余副本；错误预算、认证失败、越权、数据完整性或依赖异常立即停止 rollout。

## 完成审计

Completion Audit 聚合源事实对齐、任务矩阵、验证、迁移、release、backup/restore、operations、安全和 protected path proof。单个 smoke、gate 或 evidence 状态不能替代整体审计。

详见 [Completion Audit](completion-audit.md)。
