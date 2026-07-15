# AreaFlow Governance

Governance 记录 AreaFlow 必须长期遵守的平台行为边界。产品使用方式见 `docs/**`，关键设计原因见 `docs/adr/**`，本目录只保留不可绕过的治理规则。

AreaFlow 采用 [`ASW-EWF-001@1.0.0`](baselines/ASW-EWF-001-v1.0.0.md)。规范快照不可原地修改；采用状态、责任角色、外部依赖和偏差以 [`asw-ewf-001-adoption.yaml`](asw-ewf-001-adoption.yaml) 为唯一机器可读来源。

- [`security/`](security/)：敏感数据、secret 和外部副作用边界。
- [`permissions/`](permissions/)：capability、路径、命令和审批顺序。
- [`workflow/`](workflow/)：workflow/profile、transition 和 execution 所有权。
- [`adapters/`](adapters/)：adapter 读取、映射和项目隔离规则。

治理规则发生实质变化时，必须同步更新实现、测试、长期文档和 ADR。

## 变更与门禁

- L0：无行为变更，仅要求基础检查。
- L1：局部低风险功能或 Bug，要求需求关联、测试和必要文档。
- L2：跨模块、API 或数据变更，要求影响分析、设计记录、集成验证和发布方案。
- L3：认证、权限、迁移或跨边界写入，要求安全与数据审查、灰度、回滚和值班。
- L4：不可逆、公开稳定契约或生产基础设施，要求风险接受、演练、连续性和正式发布窗口。

G0-G8 依次覆盖需求、立项、需求就绪、设计就绪、开发就绪、合并、发布、生产结果和退役关闭。阶段 gate 是内部检查点；preview、readiness、fixture 或文档声明不能替代真实 apply、发布或生产证据。

## 责任与例外

关键事项只能有一个最终责任角色。例外必须记录 scope、风险接受人、到期日、补偿控制和关闭证据；过期例外会使治理检查失败。外部 OIDC、HA PostgreSQL、S3、TLS/LB、可观测性和生产发布由采用矩阵登记，仓库不得将其标记为已完成。
