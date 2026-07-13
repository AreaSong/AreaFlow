# Release Final Gate / Exception Contract

## 定位

本文定义 AreaFlow v1.0 的 release readiness、release exception、final gate、evidence bundle 和
publish preview 链路。它补充 [`api.md`](../../../reference/api.md)、
[`commands-and-approvals.md`](../../../concepts/commands-and-approvals.md)、
[`artifact-backup-restore-contract.md`](artifact-backup-restore-contract.md) 和
[`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)、
[`plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)。
0-100% 最终完成审计、release packaging preview 与 completion evidence 的整体门槛见
[`completion-audit-contract.md`](completion-audit-contract.md)。

v1.0 release final gate 是只读 go/no-go 判断，不是发布动作授权。`pass` 只说明发布证据链可以进入
人工 release review；它不创建 release package、不写 exception record、不运行 migration、不 tag、不
push、不 sign、不 upload、不 publish，也不执行 restore apply。
它也不单独证明 AreaFlow 已达 100%；100% 还必须通过 completion audit 聚合 AreaMatrix dogfood、
task matrix、implementation gap、operations readiness、project isolation 和 protected path proof。

## 核心不变量

- 测试通过、构建通过或 smoke 通过不能替代 release final gate。
- Release final gate 通过也不能替代 completion audit。
- Release final gate 必须消费 backup、restore dry-run、artifact integrity、audit、permission、
  adapter/profile conformance、acceptance 和 exception apply preview 证据。
- Query、preview、readiness 和 gate 端点都保持只读，不产生业务写入、worker 调度、secret 解析或命令执行。
- 所有 blocked、needs_attention、needs_decision 和 exception candidate 必须有 owner、reason、
  required evidence、next command 和 rollback / revocation path。
- 未知状态、缺失证据、范围不明或实现未覆盖时默认 `blocked`，不能降级成 `warn`。
- `project_key` 仍是 release evidence、artifact、permission、audit 和 conformance 的隔离边界。
- Release exception 只能接受明确列入白名单的历史或未来缺口，不能接受当前启用能力的安全、完整性或权限失败。

## 证据链

v1.0 release chain 固定为：

```text
backup manifest
-> restore dry-run plan
-> artifact integrity
-> audit coverage
-> permission doctor
-> adapter/profile conformance
-> release readiness
-> remediation plan
-> acceptance preview
-> acceptance gate
-> exception doctor
-> exception record preview
-> exception schema preview
-> exception migration approval gate
-> exception apply preview
-> release final gate
-> release evidence bundle
-> release package preview
-> distribution preview
-> publish gate
-> publish approval preview
-> rollout plan preview
```

链路前半段回答“当前证据是否足够”，中段回答“哪些缺口能不能被明确接受”，后半段只预演未来 package、
distribution、publish 和 rollout。任何一步的 preview / gate 都不能被解释为真实 apply 已经打开。

## Release Readiness 输入

`release readiness` 聚合以下输入：

```text
backup manifest
restore dry-run plan
audit coverage
permission doctor per project
artifact integrity per project
adapter/profile conformance per project
```

默认不带目标项目时，release readiness / final gate / evidence bundle / package preview 仍保持 platform scope，
用于暴露全局诊断和 fixture/stale project 问题。AreaMatrix dogfood release review 必须显式使用
target-scoped scope，例如 `--project areamatrix` 或 `?project=areamatrix`，并只消费目标 AreaMatrix project
的 backup manifest、restore plan、audit coverage、permission doctor、artifact integrity、conformance 和
project inventory。Target scope 不能隐藏 AreaMatrix 自身的 `needs_attention` 或 blocker；它只防止其他
fixture project 的失败污染真实 release-candidate 判断。

状态语义：

- `ready`：输入全部 `ready` / `pass`，且没有需要人工接受的缺口。
- `needs_attention`：存在 `warn`、`skipped`、metadata-only history、future-only gap 或 archive decision。
- `blocked`：存在 `fail`、`blocked`、缺失关键证据、未知状态或不可接受 blocker。

`needs_attention` 不是 release pass；它只能进入 remediation / acceptance 链路。`blocked` 必须先修复，
不能被 exception 静默放行。

## 可接受 Exception

v1.0 只允许三类 release exception candidate：

```text
metadata_only_history:
  历史 AreaMatrix 或其他项目 artifact 原文仍留在被管理项目中，AreaFlow 只持有 metadata/hash/path。

future_only_gap:
  审计或能力缺口对应尚未启用的 v1.x 能力，例如真实 secret resolve、remote worker、restore apply、
  publish apply 或未知 plugin execution。Plugin marketplace 的 v1.0 seed / manifest draft / conformance
  边界见 [`plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)。

archive_exception:
  project_reference / external_project artifact 作为历史归档引用保留，由 archive owner 接受 metadata-only
  恢复限制。
```

每个 exception 必须具备：

```text
exception_key
source readiness / gate item
acceptance_type
owner
reason
required_evidence
expires_or_review_at
rollback_or_revocation_plan
release_notes_entry
audit path
```

Exception 只接受“已知且有 owner 的限制”，不能把当前平台的失败改名成风险接受。

## 不可接受 Blocker

以下情况不能通过 release exception 放行：

```text
backup manifest broken
restore dry-run 缺少 guardrail
permission policy fail
adapter/profile conformance fail
profile hash drift 未解释
local artifact missing / unreadable / sha256 mismatch / size mismatch
enabled capability audit gap
secret leak risk
project_key isolation 失败
Command API idempotency / request hash / audit 缺失
真实 project write 缺少 expected-before hash
真实写入、restore apply、publish apply 或 migration 缺少 rollback / revocation path
release exception migration approval gate 未通过
AreaMatrix protected path proof 缺失
未知状态或未知 category
```

这些 blocker 必须先修复或保持 release blocked。特别是 permission、adapter/profile、backup、local
artifact integrity、secret 和 rollback 相关失败，一律不能被 `metadata_only_history`、
`future_only_gap` 或 `archive_exception` 包装。

## Acceptance 与 Exception 顺序

Release exception 不是口头同意，必须按顺序进入：

```text
readiness item
-> remediation action
-> acceptance preview
-> acceptance gate
-> exception doctor
-> exception record preview
-> exception schema preview
-> exception migration approval gate
-> exception apply preview
-> final gate
```

`acceptance preview` 只分类，不写入。`acceptance gate` 遇到 `needs_decision` 必须 blocked。
`exception doctor` 和 `exception record preview` 只证明未来记录字段、审计动作和 rollback 计划完整。
`exception schema preview`、`exception migration approval gate` 和 `exception apply preview` 都属于 R4
边界的预演；没有显式 R4 approval 前，不得创建 migration、运行 migration、写 exception record 或
写 audit event。

## Final Gate 语义

`release final gate` 的最小输入：

```text
release readiness
release acceptance gate
release exception apply preview
```

通过条件：

```text
release readiness = ready
release acceptance gate = pass
release exception apply preview = ready 或不需要 exception apply
```

阻断条件：

```text
release readiness = needs_attention / blocked
release acceptance gate = blocked
release exception apply preview = blocked 且 acceptance gate 仍需要 exception apply
任一输入缺失、未知或范围不匹配
```

Final gate `pass` 不代表真实发布。它只允许生成只读 evidence bundle / package preview / distribution
preview / publish gate / rollout plan preview，并等待后续 v1.x 的 publish apply 设计。

Completion audit 的 E5 / release-candidate snapshot 只接受 AreaMatrix target-scoped
`ReleaseEvidenceBundle`。Release packaging proof 必须绑定 `scope=project`、`project_key=areamatrix` 和
`evidence:project_inventory:areamatrix`，不能用 platform bundle 或 fixture-only bundle 关闭 E5。

## Publish / Restore 边界

v1.0 不打开以下真实动作：

```text
restore_database
write_project_files as restore
delete_existing_state
read or inject real secrets
create release package archive
create git tag
sign artifact
push git
upload artifact
publish release
create rollout state
```

真实 restore apply、release exception real write 和 publish apply 都属于 v1.x R4 能力。它们必须分别
通过 Command API、idempotency、permission、R4 approval、preimage / rollback、focused smoke 和 audit
证据，不能由 v1.0 final gate 隐式启用。

## AreaMatrix First Policy

AreaMatrix dogfood release 证据必须保留历史限制：

- 历史 `workflow/versions/**/execution/**`、`progress.json`、logs、prompt 和 evidence 原文短期仍在
  AreaMatrix 仓库。
- AreaFlow 只导入 index、metadata、hash、path、type、size 和 artifact relation。
- `project_reference` / `external_project` 必须进入 `needs_attention` 或显式 exception。
- `workflow/README.md`、`.areaflow/status.json`、compatibility shim 和 protected paths 的真实写入仍需要
  单独授权与 proof。

Final gate 不能把 AreaMatrix 历史引用说成 AreaFlow-owned 完整备份，也不能把 execution cutover、
restore apply 或 publish apply 伪装成 v1.0 默认能力。

## 关闭条件

进入 v1.0 release review 前，至少需要证明：

```text
release readiness consumes all required inputs
metadata-only history is visible as needs_attention or accepted exception
future-only gaps distinguish disabled capability from enabled audit failure
acceptance gate blocks unresolved needs_decision
exception apply preview remains read-only and R4-gated
final gate blocks until readiness / acceptance / exception chain agree
evidence bundle and package/distribution/publish/rollout previews do not write or publish
blocked / exception items have owner, evidence, next command and rollback / revocation path
```

如果任何一项只能通过口头解释、间接测试或缺失范围的 smoke 支撑，release final gate 仍视为未完成。
