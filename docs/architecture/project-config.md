# Project Config Contract

## 定位

`areaflow.yaml` 是 AreaFlow 的项目接入协议。它声明被管理项目的身份、连接方式、所有权阶段、
artifact store、权限边界、命令 allowlist、调度策略、engine 引用、状态投影和迁移阶段。

AreaFlow 从该文件注册项目并写入 PostgreSQL。被管理项目仍拥有自己的源码、产品文档和验证命令；
AreaFlow 只在配置允许的范围内读取、投影或后续执行。

## 放置位置

推荐最终放在被管理项目根目录：

```text
<managed-project>/areaflow.yaml
```

AreaFlow 仓库中的 `examples/<project>/areaflow.yaml` 只作为模板和 smoke 输入。AreaMatrix 在
正式接入前可以先使用 `examples/areamatrix/areaflow.yaml` 注册。

## Version

```yaml
version: 1
```

`version: 1` 是当前唯一支持的协议版本。不支持的版本必须拒绝加载。

## v0.1 Interpretation

v0.1 只把 `areaflow.yaml` 当作项目注册、只读导入和 status projection 的配置源。它可以持久化
normalized config snapshot、permissions、artifact store、scheduling metadata 和 engine metadata，但不能
因此启动 worker、运行命令、解析 secret、写 workflow 或切换 execution ownership。

v0.1 必需字段：

```text
version
project.id
project.name
project.root
project.adapter
project.workflow_profile
```

v0.1 会补默认值：

```text
ownership.mode = import
ownership.source_of_truth.product_docs = project
ownership.source_of_truth.source_code = project
ownership.source_of_truth.workflow = project
ownership.source_of_truth.execution = project
ownership.source_of_truth.status_summary = areaflow
ownership.cutover.new_versions_owned_by = project
ownership.cutover.legacy_versions_mode = project_owned
ownership.cutover.execution_owned_by = project
scheduling.priority = 100
scheduling.max_parallel_tasks = 1
scheduling.agent_role = local_worker
scheduling.required_capabilities = [read_project]
engines.profiles[].secret_ref = none when omitted
status_export.path = .areaflow/status.json when enabled
migration.strategy = import_mirror_shadow_cutover_archive
migration.phase = import
```

默认值只是 registration convenience，不是授权提升。尤其是 scheduling 和 engines 在 v0.1 只进入
metadata / readiness，不会创建 worker、lease、attempt、engine run 或 secret binding。

## Project

```yaml
project:
  id: areamatrix
  name: AreaMatrix
  root: /Users/as/Ai-Project/project/AreaMatrix
  kind: product-repo
  adapter: areamatrix
  workflow_profile: areamatrix
  default_branch: main
```

`project.id` 是 AreaFlow 的 stable project scope。所有 workflow、artifact、run、worker、secret 和
audit 都必须显式挂在这个 scope 下。

`project.root` 必须是被管理项目根目录。v0.1 只读取 allowlist 覆盖的 metadata；不能因为 root 可访问就
递归读取用户文件或写项目文件。

## Ownership

```yaml
ownership:
  mode: import
  source_of_truth:
    product_docs: project
    source_code: project
    workflow: project
    execution: project
    status_summary: areaflow
  cutover:
    enabled: false
    new_versions_owned_by: project
    legacy_versions_mode: project_owned
    execution_owned_by: project
```

支持的 `ownership.mode`：

```text
import
mirror
shadow
cutover
archived
```

支持的 owner 值：

```text
project
areaflow
external
```

v0.4 authoring cutover 只把新 workflow version authoring 切到 AreaFlow，不代表 task-loop 或
execution cutover。

`ownership.mode=cutover` 是所有权粗阶段；更细的迁移阶段由 `migration.phase=authoring_cutover` 或
`migration.phase=execution_cutover` 表达。v0.1 必须保持 `ownership.mode=import`。

`execution_owned_by` 只能在 runner / worker / permission / approval / audit 证据完整后切到
`areaflow`。在 authoring cutover 阶段，它必须保持 `project` 或 `external`，不能因
`new_versions_owned_by: areaflow` 自动推断 execution 已迁移。

v0.1 必须满足：

```text
source_of_truth.workflow = project
source_of_truth.execution = project
cutover.enabled = false
cutover.new_versions_owned_by = project
cutover.execution_owned_by = project
```

如果配置提前声明更高阶段 ownership，v0.1 import 可以记录该配置，但 readiness / doctor 必须返回
blocked 或 needs_review，不能执行 cutover。

## Artifact Store

```yaml
artifact_store:
  backend: local
  root: ~/.areaflow/artifacts
```

配置只声明平台级 root。AreaFlow 写入新 artifact 时自动追加 `{project_key}` namespace。
历史 AreaMatrix artifact 默认先作为 `project_reference` 索引，不复制原文。

v0.1 只允许：

```text
backend: local
```

其他 backend 可以作为 future metadata 保存，但在 object verifier、archive copy/upload 和 restore
integration 打开前，不能被当作完整可恢复内容。

## Permissions And Commands

```yaml
permissions:
  capabilities:
    read_project: true
    write_status: true
    write_artifacts: true
    write_workflow: false
    write_generated: false
    write_code: false
    run_commands: false
    manage_workers: false
    manage_git: false
    network: false
    use_secrets: false
    execute_agents: false
  read_paths:
    - docs/**
    - workflow/**
  write_paths:
    - .areaflow/status.json
  forbidden_paths:
    - workflow/versions/*/execution/**

commands:
  allowed:
    - ./dev workflow doctor
  forbidden:
    - ./task-loop run
    - git reset --hard
```

`write_artifacts` 只允许写 AreaFlow artifact store / artifact metadata evidence，不允许写被管理项目
文件。写被管理项目必须使用 `write_status`、`write_generated`、`write_workflow` 或 `write_code` 等更具体
capability；其中 `write_generated` 只能写显式 allowlist 的 generated/projection 前缀。所有项目写入都必须
同时满足 resource allowlist、deny list、gate 和 audit。命令执行必须同时满足 `run_commands` capability 和
command allowlist。
Project config 中的 capability 只是上限，不是最终许可。真实 command apply 还必须通过
`command-approval-contract.md` 定义的 command class、affected resources、approval scope、precondition
snapshot、expected version/hash、write-set、rollback/remediation 和 safety facts 检查。

`write_paths` 默认生成 `write_status` path allow。只有当 `write_generated: true` 且 path 位于
`.areaflow/generated/**` 或 `.areamatrix/generated/**` 这类 generated-only 前缀时，AreaFlow 才会额外生成
`write_generated` path allow。普通 `.areaflow/status.json` 不会因为出现在 `write_paths` 中自动获得
`write_generated` 权限。

v0.1 推荐的 AreaMatrix baseline：

```text
read_project = true
write_status = true only when explicit projection apply is allowed
write_artifacts = true only for AreaFlow-owned artifact store / metadata
write_workflow = false
write_generated = false
write_code = false
run_commands = false
manage_workers = false
manage_git = false
network = false
use_secrets = false
execute_agents = false
```

`commands.allowed` 在 v0.1 只作为未来 native doctor / compatibility 设计输入；只要 `run_commands=false`，
AreaFlow 仍不得执行这些命令。`commands.forbidden` 必须至少覆盖 `./task-loop run` 和破坏性 git / shell
操作。

## Scheduling And Engines

```yaml
scheduling:
  priority: 100
  max_parallel_tasks: 1
  agent_role: local_worker
  required_capabilities:
    - read_project
    - write_artifacts
  engine_profile: codex-cli

engines:
  default: codex-cli
  profiles:
    - id: codex-cli
      provider: codex-cli
      secret_ref: none
      enabled: false
```

`secret_ref` 只能是引用名，不能是明文密钥。v0.8 readiness 只读取配置，不解析 secret 明文、不调用
engine。

v0.1 可以把 scheduling 和 engine profile 写入 `project_configs` / `project_scheduling_policies`，但只能作为
metadata：

- `max_parallel_tasks` 不会创建真实并发 scheduler。
- `required_capabilities` 不会自动授权 worker。
- `engine_profile` 不会调用 engine。
- `secret_ref` 不会解析 env、keychain、DB secret 或外部 secret manager。
- `enabled: true` 在 v0.1 也不能打开 engine execution。

## Status Export

```yaml
status_export:
  enabled: true
  path: .areaflow/status.json
  human_summary:
    enabled: false
    path: workflow/README.md
    block_marker: AREAFLOW_STATUS
```

v0.1 只允许写 `.areaflow/status.json`。人类摘要写入 `workflow/README.md` 的受控区块要等 cutover
边界明确后再打开。

v0.1 `.areaflow/status.json` 只能保存粗略状态；完整 queue、execution attempt、logs、checkpoint、
approval payload、secret、worker lease 或 artifact 原文不得进入 projection。

## Migration

```yaml
migration:
  strategy: import_mirror_shadow_cutover_archive
  phase: import
  imported_versions:
    - v1-mvp
  immutable_imports:
    - v1-mvp
```

支持的 `migration.phase`：

```text
import
mirror
shadow
cutover
authoring_cutover
execution_beta
execution_cutover
archive
shim_retirement
```

历史 import 默认 immutable。修正历史事实应通过追加 event、audit 和 artifact 表达，不回写旧事实。

`cutover` 是 legacy alias；新文档应优先使用 `authoring_cutover` 和 `execution_cutover` 分层表达。
v0.1 只能使用 `import`，最多在 explicit mirror projection 后报告 mirror evidence；不能把
`status_export.enabled=true` 解释成 migration phase 已进入 `mirror`。

## Persistence Mapping

`project add --config <path>` 至少写入：

```text
projects:
  project id/name/kind/adapter/workflow_profile/default_branch。

project_connections:
  local_path -> project.root
  artifact_store -> artifact_store.root/backend

project_permissions:
  capability allow/deny
  read path allow
  write_status path allow for status_export.path / write_paths
  forbidden path deny

audit_events:
  project.upsert
```

如果 `project_configs` 和 `project_scheduling_policies` 表存在，还会写入 normalized snapshots。它们用于
readiness、doctor、future scheduling preview 和 audit，不是 v0.1 execution authorization。
