# Adapter Governance

- Adapter 负责读取和映射项目事实，不定义 workflow lifecycle。
- Adapter 必须限制在 project config 声明的 root 与 read paths。
- Adapter 不授予 capability，不绕过 permission evaluator。
- 不支持的项目结构应 fail closed，并返回可审计的诊断结果。
- 不同项目的数据必须通过 project identity 隔离。
