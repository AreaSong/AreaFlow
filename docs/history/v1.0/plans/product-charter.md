# AreaFlow Product Charter

## 定位

AreaFlow 是 AI 开发项目管理平台。它把需求讨论、workflow 生命周期、计划生成、任务草稿、队列候选、审批、执行、验证、证据记录和多项目观察连接成一个可审计的平台。

## 首个 dogfooding 项目

AreaMatrix 是 AreaFlow 的第一个被管理项目。AreaMatrix 当前已有成熟的 workflow 原型、task-loop 原型、residual ledger 和发布证据规则。AreaFlow 从这些经验中抽象平台模型，但不把 AreaMatrix 的项目事实混入 AreaFlow 产品源事实。

## 长期产品能力

- 管理多个项目。
- 管理多个 agent / worker。
- 支持多种 AI engine adapter。
- 支持 workflow lifecycle profile。
- 支持 PostgreSQL 主状态和 artifact store。
- 支持 Web Dashboard 和 Desktop Shell。
- 支持审计、权限、密钥引用和团队化部署。

## 非目标

- v0.1 不执行任务。
- v0.1 不接管 AreaMatrix workflow。
- v0.1 不修改 AreaMatrix 代码。
- AreaFlow 不拥有被管理项目的产品文档或源码语义。
- AreaFlow 不替代项目自己的验证命令和安全边界。
