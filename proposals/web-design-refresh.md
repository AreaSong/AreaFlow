# AreaFlow Web Design Refresh

> Status: Proposed. 本文描述目标 UI 和实施方向，不代表这些设计已经实现，也不开放任何新的写入、执行或高风险能力。

## 定位

AreaFlow Web 已具备 Overview、Projects、Workflows、Runs、Workers、Artifacts、Audit 和 Operations
八个一级页面。当前功能事实以 [`docs/guides/web/**`](../docs/guides/web/README.md)、
[`product-model.md`](../docs/concepts/product-model.md) 和 [`api.md`](../docs/reference/api.md) 为准。

本提案不重新定义产品能力。目标是把现有 Web 从功能展示型后台优化为高密度、任务导向、可追溯的
开发控制台，让用户更快地扫描状态、定位异常、理解资源关系和判断下一步。

## 用户与任务

目标用户按优先级为：

1. 接入项目并维护 Workflow 的项目维护者。
2. 观察 Run、Worker、Artifact 和异常状态的执行与运维人员。
3. 查询 Permission、Approval 和 Audit Event 的治理人员。
4. 维护 AreaFlow、Adapter 和 Profile 的贡献者。

用户进入 Web 后需要快速回答：

```text
当前发生了什么？
哪些资源异常、阻塞或需要关注？
资源属于哪个 Project、Workflow 或 Run？
下一步能做什么，为什么允许或禁止？
相关事件和证据在哪里？
```

## 目标与非目标

目标：

- 提高单位屏幕可扫描的信息量，同时保持清楚层级。
- 让列表适合比较、详情适合追溯、Overview 适合发现问题。
- 统一搜索、过滤、排序、分页、状态、时间和错误处理。
- 保持 Project、资源、过滤、分页和详情状态可刷新、后退和分享。
- 保持当前 API、Project visibility 和 Command/Approval/Audit 边界。

非目标：

- 不开放 Project Web CRUD、通用 AI engine execution、remote worker 或 secret resolve。
- 不打开真实 restore、publish、rollout 或 destructive action。
- 不绕过现有 REST/SSE API 或建立第二状态源。
- 不制作营销首页、插画 Hero、装饰型大卡片或尚不存在的功能入口。

## 页面清单

| 层级 | 页面 | 路由 | 当前用途 |
|---|---|---|---|
| 一级 | Overview | `/` | 当前项目健康、容量和最近事件 |
| 一级 | Projects | `/projects`、`/projects/:projectKey` | 项目配置身份、Inventory 和 Readiness |
| 一级 | Workflows | `/workflows`、`/projects/:projectKey/workflows/:version` | Version、Stage、Item、Approval 和 Residual |
| 一级 | Runs | `/runs`、`/runs/:runId` | Run、Task、Attempt 和执行 Artifact |
| 一级 | Workers | `/workers`、`/workers/:workerKey` | Worker、Heartbeat、Lease、Pool 和调度 |
| 一级 | Artifacts | `/artifacts`、`/artifacts/:artifactId` | Artifact 索引、关联和完整性标识 |
| 一级 | Audit | `/audit` | Audit Event、Domain Event 和 SSE 状态 |
| 一级 | Operations | `/operations` | 服务、迁移、Release 和受控操作状态 |
| 二级 | Compatibility | `/operations/compatibility` | AreaMatrix Projection、Cutover 和 Forwarding 诊断 |

Compatibility 继续作为 Operations 的二级诊断页面，不进入一级导航。

## 全局设计方向

### 任务导向

页面优先展示用户需要判断和处理的事实。产品说明留在文档中，不占用高频工作区。Failed、Blocked、
Needs Attention、Offline、Expired 和 Drift 等状态优先于正常状态。

### 表格优先

Projects、Runs、Workers、Artifacts 和 Audit 以表格为主要表达。卡片只用于少量独立摘要，不替代表格，
不允许卡片嵌套卡片。Workflows 可使用 Lifecycle View 和紧凑分组，不强制卡片式 Board。

### 关系可追溯

Project、Workflow Version、Run、Task、Attempt、Worker、Artifact、Event 和 Audit Event 之间应提供明确链接，
不要求用户复制 ID 后重新搜索。

### 安静视觉

- 中性灰白背景、深色文字和一个品牌强调色。
- Success、Warning、Danger、Info 使用固定语义颜色。
- 正常状态低饱和，异常状态获得更高优先级。
- 圆角不超过 `8px`，不使用渐变、光斑和大面积阴影。
- ID、Hash、URI、命令和路径使用等宽字体。
- 正文默认 `12-14px`，表格行高 `36-44px`。
- 1440px 桌面视口主要列表应看到至少 10 条记录。

## 全局框架

### 导航与上下文

- 左侧保留八个一级页面，桌面宽度建议 `216-224px`。
- 移动端保留可识别的文字标签，不只显示图标。
- Project Context 是全局查询作用域，不能与页面筛选器混淆。
- 顶部上下文栏承载当前 Project、刷新、最近更新时间、连接状态和真实可用的页面操作。
- 当前只读页面不展示无效写按钮。

### 页面结构

列表页统一为：

```text
Compact Header
Query Toolbar
Data Table / Lifecycle View
Pagination
```

详情页统一为：

```text
Resource Header
  identity / status / associations / timestamps
Detail Tabs
  summary / related resources / events or evidence
```

查询工具栏统一考虑 Search、Structured Filters、Sort、Result Count、Refresh 和 Pagination。过滤变化后回到
第一页。客户端搜索只覆盖已加载数据时，不得暗示它覆盖全部服务端数据。

### URL 状态

以下状态尽量写入 URL：

```text
project
search
filters
sort
cursor/page
selected resource
detail tab
view mode
```

无效或无权访问的资源显示 Not Found/No Access，不得静默选择第一条记录。

## 通用状态

| 状态 | 设计要求 |
|---|---|
| Loading | 使用保持布局稳定的 Skeleton 或占位行 |
| Empty | 说明没有哪类数据，只提供真实存在的下一步 |
| Filtered Empty | 区分没有数据和没有匹配结果 |
| Error | 显示失败范围、简短原因和 Retry |
| Partial Error | 一个区域失败时保留其他成功区域 |
| No Permission | 显示缺少的 Capability、Gate 或 Scope，不显示假操作 |
| Disconnected | 保留最后成功数据，显示 SSE 或服务连接状态 |
| Stale | 显示最近更新时间和刷新入口 |
| Large Dataset | 使用稳定分页，不无限渲染记录 |

## 页面改版矩阵

| 页面 | 主要问题 | 目标结构 | 视觉优先级 |
|---|---|---|---|
| Overview | 指标和列表权重接近，缺少待关注入口 | Critical Summary、Needs Attention、Active Runs、Recent Events | Blocker、失败 Run、离线 Worker |
| Projects | 通用列表不利于比较，详情纵向混杂 | Project Table；Overview、Configuration、Workflow、Activity Tabs | Readiness、Config Drift、最近更新 |
| Workflows | Item 权重平均，阶段和阻塞关系不清 | Version Context、Lifecycle View；Items、Approvals、Residuals、Runs Tabs | 当前阶段、阻塞 Item、Approval |
| Runs | 状态、时间和风险难横向比较 | Run Table；Summary、Tasks、Attempts、Artifacts、Events Tabs | Failed、Blocked、Needs Recovery |
| Workers | 身份、健康和调度状态层级不清 | Worker Table；Overview、Heartbeats、Leases、Runs Tabs | Offline、Stale Heartbeat、Expired Lease |
| Artifacts | Path、URI、Hash 形成长文本噪声 | Artifact Table；Metadata、Associations、Integrity、Content Tabs | 关联关系、完整性异常、类型 |
| Audit | 查询和结果不像审计工具 | Audit Query Bar、Audit Table、Domain Timeline | Denied/Blocked、安全判断、时间 |
| Operations | Service、Migration、Release 和 Gate 混杂 | Service、Database、Migration、Release、Audit、Actions、Diagnostics 分组 | 服务不可用、迁移阻塞、Release Blocker |

## Overview

用途：发现当前 Project 的主要问题并进入具体资源，不承载全部详情。

当前功能：Workflow Version/Artifact 数量、Queued Task/Online Worker、Readiness、Operations、健康检查、
最近 Domain Event。

调整：

- 只保留 3-4 个关键指标。
- 第一屏优先展示 Blocked、Failed、Offline 和 Needs Attention。
- Active Runs 显示状态、Workflow、开始时间和耗时。
- 最近事件保持紧凑，不复制 Audit 页面。
- 每项摘要链接对应资源页。

保留边界：Readiness 与 Operations 不合并；Domain Event 不代替 Audit Event。

## Projects

用途：查看被管理项目的注册信息、配置身份、Inventory 和 Readiness。

当前功能：列表、搜索排序分页、Project Identity、Adapter/Profile、Config Hash、Inventory、Readiness 和稳定 URL。

调整：

- 使用 Project Table，默认列为 Status、Name/Key、Adapter/Profile、Branch、Readiness、Last Updated。
- Overview Tab 展示身份、Inventory 和关键状态。
- Configuration Tab 展示 Config Identity 和权限摘要，不提供未实现的编辑表单。
- Workflow Tab 链接相关 Workflow Version。
- Activity Tab 提供最近 Event 和 Audit 入口。

保留边界：Project key、Root、Adapter 和 Profile 必须可见；Web 不绕过 CLI、配置和权限边界修改 Project。

## Workflows

用途：理解 Workflow Version 的生命周期、Item、Approval、Residual 和阻塞关系。

当前功能：版本选择、稳定 URL、Stage Board、Item 搜索排序分页、Approval 和 Residual。

调整：

- 顶部显示 Version、Lifecycle Status、Import Mode、Immutable 和 Blocker。
- Lifecycle View 表达阶段顺序、当前阶段和阻塞关系。
- Items 使用紧凑表格或 Stage 分组，不强制全部做卡片。
- Approval 显示 Scope、Decision 和相关 Item。
- Residual 显示 Impact、Status 和 Close Condition。
- Runs Tab 链接使用该版本的 Run。

保留边界：Imported Immutable Version 不得显示成可直接修改；Approval 不代表通用写权限。

## Runs

用途：发现执行异常并追踪 Run、Task、Attempt、Artifact 和 Event。

当前功能：Run Timeline、版本筛选、搜索排序分页、Type/Kind/Status/Risk/Dry-run、Task、Attempt、Artifact 和稳定 URL。

调整：

- Run Table 默认列为 Status、ID、Workflow、Type/Kind、Risk、Started、Duration、Task Progress。
- 默认最新优先，提供 Status、Workflow、Type 和时间过滤。
- Summary 显示关键状态、风险、时间和统计。
- Tasks 显示 Sequence、Kind、Status、Risk 和 Attempt 数量。
- Attempts 显示所属 Task、时间、Dry-run 和结果。
- Artifacts 显示 Type、Size、Hash 和详情链接。
- Events 只显示 Run Domain Event。

保留边界：Run Control 不暗示任意 Engine 已执行；Dry-run、Risk Level、Risk Policy 和 Project visibility 必须保留。

## Workers

用途：判断执行容量、在线状态、Heartbeat、Lease 和调度风险。

当前功能：Worker Registry、身份与 Capability、Heartbeat/Lease 历史、Pool Summary、Schedule Preview 和稳定 URL。

调整：

- Worker Table 默认列为 Status、Key、Type/Host、Capabilities、Last Heartbeat、Active Lease、Current Work。
- 默认突出 Offline、Stale Heartbeat、Expired Lease 和 Needs Recovery。
- Overview 展示 Identity、健康和当前工作。
- Heartbeats 与 Leases 使用紧凑时间序列表格。
- Runs 只在后端关联真实存在时展示。
- Schedule Preview 保持 Pool 级视图。

保留边界：Heartbeat 与 Lease 是不同事实；`run-once` 不得设计成通用远程执行按钮。

## Artifacts

用途：检索执行输入、输出和证据索引，并追溯与 Project、Workflow、Run、Item 的关系。

当前功能：索引、搜索排序分页、Type/Backend/URI/Path、Content Type、Size、SHA-256、资源关联和稳定 URL。

调整：

- Artifact Table 默认列为 Type、Source、Associations、Size、Backend、Created、Integrity。
- Path、URI 和 Hash 默认截断，提供完整值查看与复制。
- Associations 链接 Project、Workflow Version、Run 和 Item。
- Integrity 显示 Hash、Size 和检查结果。
- Content 只在 API 确认内容可用且权限允许时显示。

保留边界：数据库保存 Metadata/Hash/URI，Artifact Store 保存大内容；Archive Preview 不是 Delete/Move/Upload/GC；
Secret Value 不进入 Metadata 或预览。

## Audit

用途：回答谁对什么资源做出了什么安全判断，并区分安全判断和领域事实。

当前功能：Audit/Domain 分段视图、Actor/Action/Resource/Decision/Time 过滤、搜索排序分页和 SSE 状态。

调整：

- Audit Query Bar 集中 Project、Actor、Action、Decision、Resource Type/Resource 和 Time Range。
- Audit Table 作为默认视图，列为 Time、Actor、Action、Capability、Resource、Decision、Reason。
- Domain Timeline 作为辅助视图，不与 Audit Event 合并。
- 过滤状态写入 URL，并提供 Clear Filters。
- 长 Reason/Metadata 在详情中展开，不撑高所有行。
- SSE 断线时保留数据并标记 Stale。

保留边界：Event 与 Audit Event 语义分离；Append-only 历史没有编辑入口；Export 未实现前不显示可用命令。

## Operations 与 Compatibility

Operations 用于观察 Service、Database、Migration、Release、Completion Audit 和受控操作边界。

当前功能：Service/Operations Readiness、Migration Ledger、Support Bundle Preview、Release Final Gate、
Completion Snapshot、Web Command Boundary 和 Compatibility 入口。

调整：

- 顶部只显示真正影响平台使用的 Service 和 Database 状态。
- Migration 展示 Applied/Pending、最近检查和 Blocker。
- Release 区分 Readiness、Gate、Preview 和真实 Apply。
- Controlled Actions 显示 Mode、Blocker 和所需契约，不显示假按钮。
- Diagnostics 提供 Compatibility 入口。
- Compatibility 按 Projection、Cutover、Forwarding 和 Authorization 分组，默认折叠大量通过项。

保留边界：Manifest/Plan/Preview/Readiness/Gate 不代表副作用；Support Bundle Preview 不代表 Export；
Service Status 不代表 Process Control；Restore、Publish 和 Destructive Action 保持关闭。

Compatibility 继续保留 AreaMatrix Protected Path、Required Authorization Phrase 和只读诊断边界。

## 响应式与可访问性

Desktop：

- 目标视口 `1280-1600px`，列表使用完整宽度。
- 关键列固定，次要列可隐藏或进入详情。
- 工具栏和 Detail Tab 尺寸稳定。

Tablet：

- 隐藏低优先级列，Filter Bar 可换行。
- Project 和页面身份始终可见。

Mobile：

- 不压缩完整桌面表格，改为可扫描行摘要或分段详情。
- 一级导航保留文字标签。
- 最长 ID、Path 和 Status 不造成横向溢出。
- Detail Tabs 可滚动，但正文不依赖横向滚动。

可访问性：

- 所有交互支持键盘和清晰 Focus。
- Icon-only Button 提供 Tooltip 与 `aria-label`。
- 状态不只依赖颜色。
- 表格 Header、排序、过滤和分页具有可访问名称。
- Loading、Error、Connection Status 使用适当 Live Region。
- 文本和状态颜色满足可读对比度。

## 技术实施方向

保留 React、TypeScript、Vite、React Router、`/api/v1`、SSE 和 Project Context。

建议共享组件：

```text
AppShell
GlobalContextBar
PageToolbar
FilterBar
DataTable
ResourceHeader
DetailTabs
StatusBadge
Timestamp
CopyValue
EmptyState
ErrorState
ConnectionStatus
```

只在多个页面共享真实交互和语义时提取组件。Projects、Runs、Workers、Artifacts 和 Audit 可评估
`@tanstack/react-table`，但必须能与服务端过滤、opaque cursor、URL 状态和移动降级协作。现有规模不需要时，
继续使用小型本地组件。

## 实施阶段

1. Foundation：定义 Token，改造 AppShell、Global Context、Page Toolbar、DataTable、FilterBar、Tabs 和状态组件。
2. Core Resources：改造 Projects、Runs、Workers、Artifacts，验证表格、详情、URL 和移动策略。
3. Lifecycle/Governance：改造 Workflows Lifecycle View、Audit Table 和 Domain Timeline。
4. Operations/Overview：重组 Operations、Compatibility，并用真实资源状态收束 Overview。
5. Verification：真实 AreaMatrix 数据、Desktop/Tablet/Mobile、键盘、状态矩阵和 Browser Smoke。

## 验收标准

产品与密度：

- 八个一级页面职责清晰，不退化成单 Dashboard。
- 用户可在 5 秒内识别主要异常或阻塞。
- 1440px 主要资源表能看到至少 10 条记录。
- 正常状态紧凑，异常状态容易发现。
- 长 Hash、URI、Path、Reason 和 Metadata 不破坏布局。
- 不使用卡片嵌套或重复 Summary Card。

交互与响应式：

- Project、过滤、排序、分页、详情和 Tab 可刷新、后退和分享。
- Loading、Empty、Error、Retry、Disconnected、Stale 和 No Permission 一致。
- Stable Detail URL 不依赖当前列表窗口。
- `390x844` 无非预期横向溢出、遮挡和不可读按钮。
- Desktop、Tablet 和 Mobile 均能识别当前 Project 和页面。

工程：

- `make check` 和 `go vet ./...` 通过。
- Browser Smoke 覆盖八个一级路由和关键深链。
- 当前 API、权限、Project visibility 和只读边界无回归。
- 实现完成后将稳定事实回写 `docs/guides/web/**`，并把本 Proposal 标记为 Implemented 或归档。

## 设计交付要求

设计师至少交付：

- 全局 Shell、导航和 Project Context。
- 八个一级页面的 Desktop 主视图。
- Projects、Runs、Workers、Artifacts 的列表与详情。
- Workflows Lifecycle View。
- Audit Table、Domain Timeline、Operations 分组和 Compatibility 入口。
- Loading、Empty、Error、No Permission、Disconnected 和 Stale 状态。
- Mobile 核心页面和导航。
- Color、Typography、Spacing、Status 和 Table 规范。

设计稿中的新按钮、命令、状态或字段必须标注数据来源和当前可用性。无法映射到当前 API 或长期文档的能力
默认仍是 Proposal，不得直接进入实现。

## 待确认设计决策

- Project Context 位于侧栏还是顶部上下文栏。
- 详情以独立页面为主，还是允许桌面 Drawer Preview。
- Workflow 使用横向生命周期图、紧凑 Board，还是两者切换。
- 是否采用第三方 Headless Table Library。
- 是否提供 Compact/Comfortable 密度切换。
- Operations 使用单页分组还是 Tabs；无论哪种方式都必须保留稳定深链。

这些决策必须通过真实数据原型和响应式验证后批准，不能只根据静态视觉稿决定。
