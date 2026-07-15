# AreaFlow Agent Guide

## 定位

- 本仓库是 AreaFlow：AI 开发项目管理平台。
- AreaFlow 负责 workflow lifecycle、版本规划、任务编排、执行记录、worker 调度、artifact 索引和多项目状态。
- AreaMatrix 是第一个 dogfooding 项目，不是 AreaFlow 的子目录，也不是 AreaFlow 的产品源事实。

## 语言与表达

- 对话、说明、提交说明、设计说明默认使用中文。
- 代码标识符、类型名、函数名、文件名中的技术标识保持英文。
- 技术术语首次出现时，优先附一个简短中文解释。

## 源事实

- 当前产品事实与文档导航：`docs/README.md`、`docs/concepts/**`、`docs/guides/**`、`docs/reference/**`。
- 架构、数据模型和安全不变量：`docs/architecture/**`。
- 关键技术决策：`docs/adr/**`。
- 当前 AreaMatrix adapter 契约：`docs/reference/adapters/areamatrix.md`。
- 尚未实现的方向：`docs/roadmap.md` 与 `proposals/**`。
- 阶段计划、合同、迁移和 evidence：`docs/history/**`，仅用于历史追溯。
- 内置 workflow profile 和模板资料：`workflow/**`。
- 治理、安全、权限和 adapter 边界：`governance/**`。

## 工作原则

- 代码、数据库 migration、API/CLI 契约、页面行为和长期文档必须保持一致；不能用计划或文案替代实现与验证。
- `docs/**` 默认只描述当前真实可用能力；未来设计进入 roadmap/proposals，阶段材料进入 history。
- 不把 AreaMatrix 的历史执行状态、`progress.json`、task-loop logs 或 release evidence 直接搬入 AreaFlow。
- AreaFlow 对被管理项目默认只读；任何写入必须由 project config 显式授权。
- PostgreSQL 是 AreaFlow 的主状态源事实；文件用于配置、artifact 原文和审计导出。
- 大内容不直接塞入数据库；数据库保存 metadata、hash、URI 和关联关系。
- `events` 与 `audit_events` 采用 append-only 思路，历史事实不重写。
- 全项目治理采用 `ASW-EWF-001@1.0.0`；规范快照、适用矩阵、责任角色和外部依赖见 `governance/**`。
- 变更必须先判定 L0-L4，再应用 G0-G8、DoR、DoD、例外、发布观察和退役门禁。
- fixture、preview、readiness、gate 或文档状态不能冒充真实发布、生产运行或外部依赖完成。

## 文档变更规则

- 默认不新增 Markdown；优先更新已经存在的长期源事实，同一事实只能有一个维护位置，其他位置只链接。
- 用户可见页面或功能变化，更新所属 `docs/guides/**`；API、CLI、配置变化分别更新 `docs/reference/api.md`、`docs/reference/cli.md`、`docs/reference/configuration.md`。
- 新增或改变领域概念时更新 `docs/concepts/**`；架构与安全不变量变化时更新 `docs/architecture/**`，关键且长期有效的取舍再新增 ADR。
- 尚未实现且需要独立评审的重大设计进入 `proposals/**`；普通任务拆解、开发步骤和短期排期不创建 proposal。
- 阶段计划、milestone、实施合同、迁移过程和历史 evidence 只允许作为版本归档进入 `docs/history/<release>/**`，不得进入当前产品导航。
- 单个功能不得分别创建 plan、progress、evidence、completion、review 等 Markdown。测试日志、截图、命令输出和运行证据进入 CI、测试系统或 artifact store，不在仓库中生成阶段报告。

功能文档描述当前可用行为，至少回答：

1. 功能解决什么问题，主要使用者是谁。
2. 从哪个页面、命令或 API 进入，以及需要哪些前置条件或权限。
3. 用户可以执行什么操作，会得到什么结果。
4. 空状态、失败、阻塞、无权限和重试时如何表现。
5. 当前限制，以及必要的资源、API、配置、event 或 audit 关联。

完成任务前必须判断用户行为或公开契约是否变化。发生变化时，在同一变更中就地更新对应长期文档；未更新文档时，完成说明必须明确行为与契约没有变化。不得用新增阶段文档替代代码、测试或源事实更新。

## AreaMatrix 边界

- AreaMatrix 目前仍拥有 `docs/**`、源代码、项目治理规则、发布证据和用户文件安全边界。
- AreaFlow 通过 AreaMatrix adapter/profile 管理导入、状态投影、workflow、run、worker 和 artifact metadata；不得把 AreaMatrix 特例硬编码进 core。
- AreaMatrix 的迁移过程已经归档到 `docs/history/v1.0/migrations/**`，不能作为当前产品能力说明。
- `workflow/README.md` 和 `.areaflow/status.json` 是 AreaMatrix 的粗略投影入口，不是 AreaFlow 的主状态源。

## 禁止

- 未经独立设计、安全评审和明确批准就开放通用 AI engine execution、remote worker 或高风险 apply。
- 未经授权写入被管理项目代码、`workflow/versions/**/execution/**`、`progress.json`、checkpoint、logs 或用户文件。
- 把 AreaMatrix 当前 workflow 原样复制为 AreaFlow 的硬编码流程。
- 引入 SQLite 主状态 fallback，或让 Web/Desktop/worker 绕过 AreaFlow API 和 Command/approval/audit 边界。
- 把 users、teams、tokens、webhooks、secret resolve、remote workers、plugin execution 等预留数据结构写成已开放能力。
