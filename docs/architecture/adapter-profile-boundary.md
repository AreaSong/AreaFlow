# Adapter / Profile Boundary

## 定位

AreaFlow 支持多项目、多 workflow 生命周期。Adapter 和 workflow profile 必须分离，避免 core
被 AreaMatrix 的目录结构绑死。

```text
Adapter = 怎么连接、读取、写入某类项目
Profile = 这个项目采用什么 workflow 生命周期
Plugin = AreaFlow 能力怎么受控扩展
```

AreaMatrix 当前配置为：

```text
adapter: areamatrix
workflow_profile: areamatrix
```

这只是第一个 dogfooding 组合，不是平台唯一形态。

## Adapter 职责

Adapter 负责把项目特有结构映射成 AreaFlow 通用对象。

```text
load project metadata
scan source references
import historical artifacts
detect drift
export status projection
apply allowed writes
run allowed native commands
map project-specific files to generic AreaFlow objects
```

Adapter 不应该：

- 定义 workflow 状态机。
- 决定 promotion 是否通过。
- 决定 approval 是否有效。
- 直接调 AI engine。
- 绕过 permission evaluator。
- 直接写数据库。
- 私自执行命令。
- 把项目特有状态提升成 core 字段。

## Profile 职责

Profile 定义 workflow 生命周期。

```text
define stages
define allowed transitions
define required artifacts
define gates
define validation commands
define promotion rules
define rollback routes
define closeout rules
```

Profile 不应该：

- 直接读磁盘。
- 执行命令。
- 知道项目本地路径。
- 处理 secret。
- 把默认模式改成可写。
- 省略写入前置条件。

所有 workflow profile 都必须声明安全写入前置：

```text
permissions.default_mode = readonly
permissions.write_requires includes:
  capability
  path_allowlist
  gate_result
  approval_record
  audit_event
```

Profile 可以声明某个 stage 需要哪些 gate 和 artifact，但不能授予 capability、不能扩大 path
allowlist，也不能把 gate pass、approval 或 readiness 解释成 apply。真实写入仍由 project config、
permission evaluator、Command API 和 audit event 共同决定。

## Plugin 职责

Plugin 是后续扩展机制，不是 v0 必需品。Plugin / marketplace 的完整边界见
[`plugin-marketplace-boundary.md`](plugin-marketplace-boundary.md)。长期 plugin 可以提供：

```text
new adapter
new workflow profile
new engine provider
new artifact backend
new notification provider
new integration
new gate checker
```

Plugin 不能绕过 core 安全边界。所有 plugin 动作必须经过：

```text
project scope
permission evaluator
capability check
path allowlist
secret reference policy
audit event
```

Plugin 不是随意执行代码的入口，而是受治理的扩展包。v1.0 只允许 built-in / seed catalog、
manifest draft、profile/template metadata 和 conformance；未知第三方 plugin install / enable /
execution 全部留到 v1.x。

## Core 接口

AreaFlow core 只依赖稳定接口：

```text
ProjectAdapter
WorkflowProfile
ArtifactStore
PermissionEvaluator
CommandRunner
SecretResolver
WorkerRuntime
EngineAdapter
```

## Registry 阶段

```text
v0-v0.4:
  built-in adapters/profiles

v0.5-v0.8:
  registry 接口稳定

v1.0:
  plugin / marketplace seed 边界稳定，不执行未知第三方代码

v1.x:
  第三方 plugin 安装、签名、版本兼容、沙箱、disable/revoke 和受控执行
```

AreaMatrix adapter 可以先内置在：

```text
internal/adapter/areamatrix
workflow/profiles/areamatrix
```

后续再抽 plugin。

## AreaMatrix Profile v0

AreaMatrix profile 映射以下 stage：

```text
intake
source_docs
templates
version_init
discussion
middle_layer
changes
plans
drafts
queue
promotion_preview
approval
execution
run
projection
closeout
```

这些 stage 是 AreaMatrix profile 的生命周期，不是 AreaFlow core 的固定目录。

## 多项目规则

- 所有对象都必须带 `project_key` scope。
- adapter metadata 可以保存项目特有字段。
- API 和 Web 默认展示通用对象。
- profile 决定 stage/gate/transition，不决定权限。
- permission evaluator 是所有 adapter 和 worker 的统一边界。
- conformance 需要证明 adapter、profile、active project config policy、artifact、worker lease、secret
  reference、audit 和 API 查询没有跨 `project_key` 串线；AreaMatrix baseline 还必须证明 workflow
  profile 的 item states、transition、hard rules、artifact ownership/storage policy 和 cutover policy 没有漂移，
  且 `areaflow.yaml` 没有打开 workflow/code/command/worker/git/network/secret/agent 等高风险 capability。

## 示例组合

```text
adapter: areamatrix
workflow_profile: areamatrix

adapter: git-repo
workflow_profile: areaflow-standard

adapter: local-folder
workflow_profile: lightweight-kanban
```

一句话边界：

```text
Adapter 负责 project IO。
Profile 负责 workflow semantics。
Plugin 负责受控扩展。
Core 永远拥有状态转移、权限、审计和执行安全。
```
