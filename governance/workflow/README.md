# Workflow Governance

- Core workflow engine 只理解通用 stage、item、gate、transition、artifact 和 approval。
- 项目生命周期差异通过 profile 表达，不硬编码进 core。
- Workflow Version 创建时冻结 profile version/hash。
- Immutable import 不被后续 profile 变更静默改写。
- Transition、run 和 execution 的状态变化必须产生 event；安全判断必须产生 audit event。
