# AreaFlow Agent Guide

## 定位

- 本仓库是 AreaFlow：AI 开发项目管理平台。
- AreaFlow 最终负责 workflow lifecycle、版本规划、任务编排、执行记录、worker 调度、artifact 索引和多项目状态。
- AreaMatrix 是第一个 dogfooding 项目，不是 AreaFlow 的子目录，也不是 AreaFlow 的产品源事实。

## 语言与表达

- 对话、说明、提交说明、设计说明默认使用中文。
- 代码标识符、类型名、函数名、文件名中的技术标识保持英文。
- 技术术语首次出现时，优先附一个简短中文解释。

## 源事实

- 产品定位与路线：`docs/product/**`。
- 架构、数据模型、生命周期：`docs/architecture/**`。
- 关键技术决策：`docs/adr/**`。
- AreaMatrix dogfooding 契约：`docs/dogfood/areamatrix-contract.md`。
- AreaMatrix workflow 迁移路线：`docs/migration/areamatrix-workflow-migration.md`。
- 内置 workflow profile 和模板资料：`workflow/**`。
- 治理、安全、权限和 adapter 边界：`governance/**`。

## 工作原则

- Phase 0 是设计基线阶段：先文档、后代码。
- 不把 AreaMatrix 的历史执行状态、`progress.json`、task-loop logs 或 release evidence 直接搬入 AreaFlow。
- AreaFlow 对被管理项目默认只读；任何写入必须由 project config 显式授权。
- PostgreSQL 是 AreaFlow 的主状态源事实；文件用于配置、artifact 原文和审计导出。
- 大内容不直接塞入数据库；数据库保存 metadata、hash、URI 和关联关系。
- `events` 与 `audit_events` 采用 append-only 思路，历史事实不重写。

## AreaMatrix 边界

- AreaMatrix 目前仍拥有 `docs/**`、源代码、项目治理规则、发布证据和用户文件安全边界。
- AreaFlow 最终接管 workflow/task-loop 主能力；迁移顺序是 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover -> Archive -> Shim Retirement。
- v0.1 只做 Import + Status Mirror，不执行任务、不写代码、不接管 workflow。
- AreaMatrix 最终只保留粗略进度入口，例如 `workflow/README.md` 和 `.areaflow/status.json`。

## 禁止

- 未经设计文档确认就创建执行 runner、worker 或 AI engine 调用。
- 未经授权写入被管理项目代码、`workflow/versions/**/execution/**`、`progress.json`、checkpoint、logs 或用户文件。
- 把 AreaMatrix 当前 workflow 原样复制为 AreaFlow 的硬编码流程。
- 在 v0.1 引入 SQLite 主状态 fallback。
- 在 v0.1 引入 Web/Desktop 实现或真实 task execution。
