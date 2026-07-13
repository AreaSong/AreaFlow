# Plugin / Marketplace Boundary

> Status: Proposed. 当前产品只支持内置 adapter/profile，未开放第三方 plugin execution 或 marketplace 安装。

## 定位

本文定义 AreaFlow 的 adapter、workflow profile、template 和 plugin marketplace 边界。它补充
[`adapter-profile-boundary.md`](../docs/architecture/adapter-profile-boundary.md)、
[`workflow-engine-contract.md`](../docs/architecture/workflow-engine-contract.md)、
[`command-approval-contract.md`](../docs/history/v1.0/contracts/command-approval-contract.md) 和
[`high-risk-apply-ladder.md`](./high-risk-apply.md)。

v1.0 的目标是稳定 plugin / adapter / profile 的治理边界和 seed catalog，不是打开未知第三方代码执行。
任何第三方 plugin execution 都属于 v1.x high-risk rung 14，必须通过 manifest、signature、sandbox、
capability、disable/revoke、audit 和 explicit approval。

## 名词边界

```text
adapter:
  Project IO。读取项目 metadata、snapshot、import、drift、projection 和受控原生命令 preview。

workflow_profile:
  Workflow semantics。声明 stage、transition、gate、required artifact、failure route 和 closeout semantics。

template:
  可复用文档、prompt、stage skeleton、gate checklist 或 profile scaffold；不执行代码。

plugin:
  受治理扩展包。未来可提供 adapter、profile、engine provider、artifact backend、notification provider、
  gate checker 或 integration。

marketplace:
  可发现、可校验、可禁用的 package catalog。v1.0 只允许 built-in / seed metadata，不拉取或执行未知代码。
```

Adapter、profile 和 template 可以是 v1.0 seed catalog 的发布单元；plugin code execution 不能。
即使 manifest 声明 `package_type=integration`，也只能说明 catalog metadata；真实 webhook delivery、
callback processing 或 external API connector 仍必须按
[`integration-webhook-boundary.md`](./integrations-and-webhooks.md) 单独开闸。

## v1.0 Seed Scope

v1.0 允许：

```text
built-in adapter metadata
built-in workflow profile metadata
profile/template seed catalog
manifest schema draft
profile hash and version binding
adapter/profile conformance
docs/examples/schemas for future plugin packages
read-only marketplace listing or documentation
```

v1.0 禁止：

```text
install third-party plugin package
execute third-party plugin code
load dynamic Go plugin / WASM / script hook
fetch remote marketplace package
auto-update adapter/profile/plugin
grant plugin capability
resolve plugin secret
run plugin command
write project files through plugin
write database through plugin
write artifact store through plugin
open network through plugin
```

Seed catalog `ready` 只表示 metadata 可校验，不表示 plugin 可安装或可执行。

## Package Manifest Draft

未来 marketplace package manifest 至少需要：

```text
package_id
package_type: adapter | workflow_profile | template | engine_provider | artifact_backend | notification_provider | gate_checker | integration
display_name
version
publisher
license
source_uri
package_hash
signature
compatibility:
  areaflow_min_version
  areaflow_max_version
  api_contract_version
capabilities_requested[]
resources_requested[]
commands_requested[]
network_access
secret_refs_requested[]
artifact_access
project_write_access
sandbox_policy
install_steps[]
disable_steps[]
revoke_steps[]
migration_steps[]
rollback_steps[]
conformance_checks[]
audit_actions[]
```

v1.0 可以 document / lint 这个 shape，但不能把 manifest presence 当作 execution approval。

## Registry States

Marketplace registry 使用分层状态：

```text
built_in:
  随 AreaFlow repo 发布，代码和 profile 由当前仓库治理。

seed:
  只读 metadata/template/profile catalog，可展示、可 hash、可 conformance check。

candidate:
  未来可安装候选；只允许 manifest lint 和 signature preview，不执行。

verified:
  manifest、signature、compatibility 和 conformance 通过；仍不代表 execution open。

enabled:
  未来显式 approval 后可被某 project 使用；仅在声明 capability scope 内生效。

disabled:
  被 operator 或 policy 禁用，不可新建使用。

suspended:
  因安全、审计、沙箱、hash、secret、project isolation 或 rollback 问题被立即停用。
```

v1.0 只能使用 `built_in` 和 `seed`。`candidate` 以后仍是 preview-only；`enabled` 和 execution 必须走
v1.x high-risk ladder。

## Conformance

v1.0 conformance 至少证明：

```text
project adapter/profile binding matches profile defaults
built-in profile loads with stable sha256
profile validate passes without hidden warnings
stage and gate contract is stable
adapter snapshot is read-only
adapter/profile boundary does not write database
adapter/profile boundary does not write project files
adapter/profile boundary does not execute commands
adapter/profile boundary does not resolve secrets
adapter/profile boundary remains project-scoped
```

未来 marketplace conformance 还必须证明：

```text
manifest fields complete
package hash matches source
signature valid
requested capabilities are explicit
requested resources are explicit
secret and network access are explicit
sandbox policy exists
disable/revoke path exists
audit actions declared
version compatibility declared
```

Conformance failure is a release blocker. It cannot be accepted as `metadata_only_history`,
`future_only_gap` or `archive_exception`.

## Install / Enable / Execute Ladder

第三方 plugin 从发现到执行必须分步：

```text
seed catalog entry
-> manifest lint
-> signature preview
-> compatibility preview
-> conformance preview
-> install approval
-> install command
-> enable approval
-> project-scoped enable command
-> execution approval
-> sandboxed execution command
-> audit / closeout
```

每一步都必须是独立 Command API 或 read-only preview。Install approval 不等于 enable；enable 不等于
execution；execution approval 也不能扩大 manifest 未声明的 capability。

## Capability Rules

Plugin capability 永远不能扩大 project config：

```text
effective_capability =
  plugin manifest requested capability
  AND package verified capability
  AND project config allowed capability
  AND permission evaluator allowed resource
  AND gate / approval scope
```

Deny 优先。Plugin 不能获得以下隐式能力：

```text
write_code
run_commands
manage_git
network
use_secrets
execute_agents
restore_database
publish_release
```

这些能力必须逐项声明、逐项 approval，并按 R3/R4 风险处理。

## Sandbox And Secret Rules

未来 plugin execution 必须运行在受控 sandbox 内。Sandbox contract 至少声明：

```text
filesystem view
network policy
environment variables
secret binding policy
artifact input/output paths
stdout/stderr redaction
timeout and resource limits
process isolation
disable/revoke behavior
```

Secret value 不能进入 plugin manifest、package metadata、project config、event、audit、artifact 或 logs。
Plugin 只能接收 short-lived scoped binding，并且必须继承
[`auth-team-secret-boundary.md`](./auth-team-secret.md) 的 redaction、expiry 和 revoke 要求。

## AreaMatrix First Policy

AreaMatrix v1.0 使用 built-in adapter/profile：

```text
internal/adapter/areamatrix
workflow/profiles/areamatrix/profile.yaml
```

AreaMatrix 不依赖第三方 plugin。把 AreaMatrix adapter/profile 抽成 plugin 只能发生在内置边界稳定之后，
并且第一阶段只能是 `seed` / `candidate` metadata，不执行代码。

任何 plugin 对 AreaMatrix 的真实写入都必须额外证明：

```text
project_key scope
protected path deny
workflow/versions/**/execution/** untouched unless explicitly approved
progress.json untouched unless explicitly approved
legacy logs and release evidence untouched
user file safety boundary preserved
AreaMatrix rollback / revoke plan
```

## Release Boundary

v1.0 release 可以通过：

```text
adapter/profile conformance
seed catalog metadata
manifest schema draft
public docs
examples
```

v1.0 release 不能通过：

```text
third-party plugin install
third-party plugin enable
third-party plugin execution
unknown package fetch
plugin secret resolve
plugin project write
plugin network access
plugin publish / restore action
```

Plugin marketplace gap 只能作为 future capability gap 暴露；不能伪装成 plugin execution 已经完成。

## Suspension Rule

未来已启用 plugin 命中以下任一情况，必须立即 `suspended`：

```text
signature mismatch
package hash mismatch
sandbox escape
undeclared capability use
undeclared network use
secret leak
unexpected project file write
project isolation failure
audit event missing
disable/revoke failure
version compatibility mismatch
```

恢复前必须重新通过 manifest、signature、sandbox、conformance、focused smoke、audit 和 explicit approval。

## 关闭条件

v1.0 plugin / marketplace boundary 关闭前至少需要：

```text
adapter/profile/plugin boundary documented
built-in AreaMatrix adapter/profile conformance passes
profile hash/version binding documented
seed catalog scope documented
manifest draft documented
unknown plugin execution explicitly deferred to v1.x rung 14
plugin install/enable/execute ladder documented
AreaMatrix first policy documented
```

如果缺少其中任何一项，v1.0 只能声称 adapter/profile conformance，不能声称 plugin marketplace boundary
稳定。
