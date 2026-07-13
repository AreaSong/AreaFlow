# Security And Permissions

## 默认策略

AreaFlow 对被管理项目默认只读。任何写入、命令执行、网络访问、密钥使用和 git 操作都必须通过显式授权。

## Capability

```text
read_project
write_status
write_artifacts
write_workflow
write_generated
write_code
run_commands
manage_workers
manage_git
network
use_secrets
execute_agents
```

`write_artifacts` 表示写入 AreaFlow-owned artifact store 或 artifact metadata evidence，不等同于写
被管理项目文件；仍必须带 project scope、artifact type、hash 和 audit/event 证据。`write_generated`
表示只写被管理项目内显式 allowlist 的 generated/projection 前缀，例如 `.areaflow/generated/**` 或
`.areamatrix/generated/**`；不代表可写 source code、workflow execution、progress JSON 或任意项目文件。
`manage_workers` 表示注册、心跳、lease 和 recovery 等 worker lifecycle 操作，不等同于允许真实执行任务。
所有 capability 都是 resource-scoped 上限，不是最终许可；最终许可还必须由 Command API 的 affected
resources、deny list、gate snapshot、approval scope、expected version/hash、write-set、rollback 和 audit
共同决定。

v0.1 仅允许 `read_project` 和可选 `write_status`。`write_artifacts` 随 runner/worker evidence
逐步打开；`manage_workers` 随 v0.6 worker beta 打开。`write_workflow`、`write_generated`、
`write_code`、`run_commands`、`manage_git`、`network`、`use_secrets` 和 `execute_agents` 默认关闭。

## 风险等级

AreaFlow 按风险等级决定 gate、approval 和 audit 强度：

```text
R0 read_only:
  只读查询、metadata import、hash、preview、doctor。

R1 projection:
  写轻量 status projection，例如 `.areaflow/status.json`。

R2 managed_write:
  写被管理项目 allowlist 内的 generated/projection/workflow/export 文件。

R3 execution:
  执行任务、修改代码、运行 worker、生成 execution evidence。

R4 migration_security:
  DB migration、secret 解析、权限变更、远程 worker、release exception apply。
```

R0 不应产生项目写入。R1 必须同时满足 `write_status`、路径 allowlist 和 audit。R2-R4 必须经过
明确 gate、approval、preview 和 rollback 说明；任何端都不能把 R2-R4 操作伪装成只读 preview。
Command API、approval、permission、expected version/hash、audit 和 rollback 的统一执行顺序见
[`command-approval-contract.md`](./command-approval-contract.md)。本文定义能力和风险边界；真实 apply
必须按该合同进入受保护 command。
post-100% 高风险 real apply 的状态词、apply packet、串行原则、suspension rule 和 AreaMatrix first
policy 见 [`high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md)。

Auth、team、API token、secret resolve 和 remote worker credential 的具体 R4 开闸顺序见
[`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)。v1.0 前只允许 schema / readiness /
doctor / preview，不启用真实 bearer auth enforcement、token issuance enforcement、team role
enforcement、secret resolve 或 remote worker credential。
Budget、quota、rate limit 和 usage metering 的开闸边界见
[`budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md)。v1.0 前 budget policy 只能作为 readiness /
blocked reason；不能扣减 quota、写 charge、阻断真实 run 或 silent throttle。
External API、webhook、callback 和 notification provider 的开闸边界见
[`integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)。v1.0 前 integration 只能作为
metadata / readiness / preview；不能投递 webhook、接受 callback 作为业务事实或调用外部 API。

## 路径策略

权限判断必须同时满足：

```text
capability allowed
path allowed
not forbidden
command class allowed
approval scope matched
expected version/hash matched when applicable
rollback or remediation available when applicable
```

deny 优先于 allow。

## 命令策略

命令必须通过 allowlist。v0.1 不需要执行任务命令；只读状态命令可以作为可选验证。
v0.2 的 native workflow doctor shadow 对照也必须同时满足 `run_commands`
capability 和 command allowlist；否则只记录 skipped/warn，不执行命令。
`areaflow project doctor <id> --allow-native` 只能作为一次性人工授权覆盖
`run_commands` capability；它仍受 command allowlist 和 forbidden command 限制。

禁止命令示例：

```text
./task-loop run
git reset --hard
git checkout --
rm -rf
```

## 网络与外部集成

`network` capability 只表示允许进入网络 preflight，不表示可以访问任意外部目标。真实 external API、
webhook delivery、provider notification 或 callback processing 还必须满足 provider allowlist、endpoint
allowlist、method allowlist、purpose、secret scope、budget/quota preflight、Command API 和 audit。
完整合同见 [`integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)。

## 密钥

项目配置只写 `secret_ref`，不写明文密钥。v1.0 前 AreaFlow 只做 secret readiness，不解析明文、不把
secret 注入 engine、不把 secret 写入 artifact、event 或 audit metadata。

阶段边界：

```text
v0.1-v0.8:
  只读取项目配置中的 secret_ref，返回 readiness / blocked reason。

v0.9-v1.0:
  Desktop 和 Web 可以展示 secret readiness，但不能显示或解析明文。

v1.x:
  另行设计 OS keychain、env、encrypted secret store 或外部 secret manager，并经过 R4 approval。
```

`secret_ref = none` 表示该 engine/profile 不需要 secret。其他引用在 secret store 能力打开前只能返回
`secret_ref_unavailable`，不能隐式读取本机环境变量或 keychain。

真实 secret resolve 必须按 [`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md) 的 scoped
binding、redaction、audit 和 rollback 要求执行。

## Engine Adapter

Runner 选择 `engine_profile`，worker 调用 `engine_adapter`，密钥解析仍受 AreaFlow policy 控制。Runner
不得硬编码 Codex CLI、OpenAI API 或任何具体 provider。

```text
engine_profile:
  provider
  capabilities
  secret_ref
  readiness_status
  budget_policy

engine_adapter:
  codex_cli
  openai_api
  local_model
  external_agent
  manual_worker
```

v1.0 前 `engine_profile` 只能参与 readiness、schedule preview、risk preview 和 blocked reason；
不能因为 profile 存在就解析 secret 或真实调用 provider。v0.6 Codex CLI adapter preview 也只允许读取
project config、permission rows 和 engine profile metadata，并必须返回 no secret / no command /
no network / no project write safety facts。真实 engine 调用必须同时满足：

v0.6i fixture execution apply 可以在临时 fixture config 中启用 `execute_agents`、`run_commands`、
`codex-cli.enabled=true` 和 `codex exec` allowlist，用于让 execution approval gate 通过。但
`worker.fixture_execute` 本身仍只写 AreaFlow PG state 和 artifact store；它必须返回
`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`
和 `project_write_attempted=false`。该路径不能被解释为真实 engine execution 已经打开。

v0.6j read-only verify 允许在 execution approval gate 通过后读取 project config allowlist 明确允许的
target file。读取前必须满足 `read_project` capability、path allowlist、forbidden path deny 和 project
root 防逃逸检查。`worker.read_only_verify` 只把 target path、sha256 和 size 写入
`read_only_verify_report` artifact，不保存 target file 原文；它必须返回
`project_read_attempted=true`、`project_read_allowed=true`、`project_write_attempted=false`、
`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false` 和 `network_used=false`。该路径不能被解释为项目写入、engine execution
或真实 AreaMatrix execution cutover 已经打开。

v0.6k approved artifact write 允许在 execution approval gate 通过后写 AreaFlow-owned artifact store 和
artifact metadata evidence。它要求 worker 注册 capability 包含 `write_artifacts`，project config 也允许
`write_artifacts`；但它不读取 project file、不写被管理项目、不写 `workflow/versions/**/execution/**`、
不调用 engine、不运行 shell、不解析 secret、不访问网络。`worker.approved_artifact_write` 必须返回
`project_read_attempted=false`、`project_write_attempted=false`、`execution_write_attempted=false`、
`area_flow_artifact_written=true`、`area_flow_execution_state_written=true`、
`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false` 和 `network_used=false`。
该路径不能被解释为 `write_artifacts` 已授权项目文件写入。

v0.6l execution plan preview 允许在不执行任何 apply 的前提下展示真实 execution 的下一步计划。它只读取
run detail 和 execution approval gate，并返回 copy、verify、approved artifact write、checkpoint 和 repair
step 的 required capabilities、blockers 与安全属性。该 preview 必须返回
`project_read_attempted=false`、`project_write_attempted=false`、`execution_write_attempted=false`、
`area_flow_artifact_written=false`、`area_flow_execution_state_written=false`、
`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、
`task_claimed=false`、`worker_started=false`、`attempt_created=false` 和 `artifact_created=false`。
该路径不能被解释为 copy、repair、checkpoint、engine execution、project write 或 artifact write 已打开。

v0.6n fixture-only approved project write 是第一条真正触碰 project root 的写路径，但它只允许 fixture
project。执行前必须同时满足 execution approval gate、worker capability `read_project` /
`write_artifacts` / `write_code`、project capability `write_artifacts`、target path 对 `read_project`
和 `write_code` 的 allowlist，以及 project root 防逃逸检查。target 必须是已存在的普通文件，不能是目录、
symlink、root 外路径、glob、binary rewrite、create/delete/move/chmod。执行时必须校验 expected-before
hash/size，写 preimage artifact、copy attempt、verify attempt、rollback attempt 和 report artifact，并在
commit 前恢复到 preimage hash/size。该路径必须返回 `project_read_attempted=true`、
`project_read_allowed=true`、`project_write_attempted=true`、`project_write_allowed=true`、
`execution_write_attempted=false`、`area_flow_artifact_written=true`、
`area_flow_execution_state_written=true`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false` 和 `rollback_verified=true`。该路径不能被解释为真实
AreaMatrix 写入、managed-project generated-only write、source write、checkpoint 或 repair 已经打开。

v0.6o managed generated write gate 是 managed-project generated-only write apply 前的只读门禁。它只读取
execution approval gate，并返回 generated-only 前缀、required write-set fields、unsupported operations、
apply sequence 和 blockers。它的 required capabilities 必须使用 `read_project`、`write_artifacts` 和
`write_generated`，不能借用 `write_code`。它必须返回 `generated_only_write_ready=true|false` 以及
`generated_only_apply_open=false`，并保持 `project_read_attempted=false`、`project_write_attempted=false`、
`execution_write_attempted=false`、`area_flow_artifact_written=false`、
`area_flow_execution_state_written=false`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false`、`task_claimed=false`、`worker_started=false`、
`lease_created=false`、`attempt_created=false` 和 `artifact_created=false`。该路径不能被解释为真实
AreaMatrix 写入、generated-only apply、source write、checkpoint、repair、engine 或 shell execution 已打开。

v0.6p managed generated write apply 是 generated-only apply 的第一条核心服务链，并已暴露受限
API/CLI，但只允许 fixture/temp project。它要求 worker capability 和 project capability 同时包含 `read_project`、
`write_artifacts` 和 `write_generated`，target path 必须位于 `.areaflow/generated/**` 或
`.areamatrix/generated/**`，并且同时通过 `read_project` 和 `write_generated` path allowlist。当前实现只支持
修改已存在的普通 generated 文件，必须校验 expected-before hash/size，写 write-set、preimage、copy、
verify、rollback attempt 和 report artifact，并在 commit 前恢复 preimage hash/size。该路径必须返回
`project_read_attempted=true`、`project_write_attempted=true`、`area_flow_artifact_written=true`、
`area_flow_execution_state_written=true`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false` 和 `rollback_verified=true`。v0.6p 的 API/CLI 入口只调用
同一条 fixture/temp generated-only rollback drill，不开放真实 AreaMatrix 写入、保留生成结果、source write、
checkpoint、repair、engine、shell、secret、network 或 `workflow/versions/**/execution/**` 写入。
如果响应包含 `generated_only_apply_open=true`，该字段只在 fixture/temp rollback drill scope 内成立，
不能解释为真实 AreaMatrix generated apply 或 retained managed-project apply 已打开。

v0.6q generated write readiness 是真实 AreaMatrix generated-only dogfood apply 前的 project-scoped 只读
门禁。它只读取 AreaFlow PostgreSQL 中的 active project config 和 permission rows，检查
`read_project`、`write_artifacts`、`write_generated` capability、`.areaflow/generated/**` /
`.areamatrix/generated/**` path allowlist、dangerous path deny、rollback contract 和 unrelated high-risk
capability 关闭状态。它返回 `ready_for_review=true|false`，但当前必须保持 `apply_open=false` 和
`real_areamatrix_write_opened=false`。该 readiness 必须保持
`project_read_attempted=false`、`project_write_attempted=false`、`execution_write_attempted=false`、
`area_flow_artifact_written=false`、`area_flow_execution_state_written=false`、
`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、
`task_claimed=false`、`worker_started=false`、`lease_created=false`、`attempt_created=false` 和
`artifact_created=false`。它不能被解释为真实 AreaMatrix generated apply、queue、source write、
checkpoint、repair、engine、shell、secret 或 network 已打开。

v0.6r generated write apply beta gate 是真实 AreaMatrix generated-only apply beta 打开前的只读
approval gate。它嵌套 generated write readiness，并额外要求 explicit R3 approval、最新 focused smoke、
单文件 existing generated target、expected-before hash/size、preimage artifact、rollback verification plan
和非目标 AreaMatrix 文件指纹不变。该 gate 当前必须保持 `approval_required=true`、
`approval_status=needs_approval`、`apply_open=false` 和 `real_areamatrix_write_opened=false`。它不创建
approval record、不写 command request、不 queue run、不创建 lease/attempt/artifact/event/audit，也不读取或
写入真实 AreaMatrix 文件。该 gate 必须保持 `project_read_attempted=false`、
`project_write_attempted=false`、`execution_write_attempted=false`、`area_flow_artifact_written=false`、
`area_flow_execution_state_written=false`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false`、`task_claimed=false`、`worker_started=false`、
`lease_created=false`、`attempt_created=false` 和 `artifact_created=false`。`needs_approval` 只能触发人工
审批流程，不能被解释为 apply 许可。

```text
approval valid
use_secrets / execute_agents capability allowed
secret policy allowed
budget / rate limit policy allowed
worker scope allowed
audit event written
artifact redaction policy active
```

`budget / rate limit policy allowed` 只是 execution gate 的一个输入。Budget pass 不能替代 permission、
approval、secret、network、worker scope 或 artifact redaction；budget fail 必须返回机器可读 blocker，
不能静默降级、丢弃或延迟任务。

Worker 不应持有长期密钥。真实执行阶段最多获得一次 scoped execution context：

```text
engine_run_context_id
lease_id
allowed_command
allowed_network_targets
artifact_upload_scope
short_lived_token_or_env_binding
expires_at
```

长期 secret 仍由 AreaFlow secret manager、OS keychain、env binding 或外部 secret manager 管理。worker
stdout、stderr、artifact、event 和 audit metadata 必须避免记录明文 secret。

## 审计

所有写入、权限判断、命令执行和密钥引用都写入 `audit_events`。Team admin、local admin、Desktop、
Web、CLI、worker 和 AreaMatrix shim 都不能绕过 project config、Command API、approval、restore gate、
publish gate、secret scope 或 audit。

最小审计主体包括：

```text
system
local-user
human
service
worker
api-token
cli-token
agent
areamatrix-shim
```

本机单用户模式也要使用稳定 actor，避免后续团队模式无法追溯历史动作。
