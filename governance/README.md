# AreaFlow Governance

Governance 记录 AreaFlow 必须长期遵守的平台行为边界。产品使用方式见 `docs/**`，关键设计原因见 `docs/adr/**`，本目录只保留不可绕过的治理规则。

- [`security/`](security/)：敏感数据、secret 和外部副作用边界。
- [`permissions/`](permissions/)：capability、路径、命令和审批顺序。
- [`workflow/`](workflow/)：workflow/profile、transition 和 execution 所有权。
- [`adapters/`](adapters/)：adapter 读取、映射和项目隔离规则。

治理规则发生实质变化时，必须同步更新实现、测试、长期文档和 ADR。
