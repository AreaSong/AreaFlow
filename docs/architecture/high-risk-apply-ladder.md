# High-risk Apply Ladder

## 定位

本文定义 AreaFlow v1.x 高风险真实 apply 的统一开闸阶梯。它补充
[`command-approval-contract.md`](command-approval-contract.md)、
[`execution-opening-strategy.md`](execution-opening-strategy.md)、
[`auth-team-secret-boundary.md`](auth-team-secret-boundary.md)、
[`artifact-backup-restore-contract.md`](artifact-backup-restore-contract.md)、
[`object-artifact-retention-contract.md`](object-artifact-retention-contract.md) 和
[`release-final-gate-contract.md`](release-final-gate-contract.md)、
[`plugin-marketplace-boundary.md`](plugin-marketplace-boundary.md)、
[`team-remote-control-boundary.md`](team-remote-control-boundary.md) 和
[`budget-quota-boundary.md`](budget-quota-boundary.md) 和
[`integration-webhook-boundary.md`](integration-webhook-boundary.md) 和
[`operations-deployment-observability-boundary.md`](operations-deployment-observability-boundary.md)。

v1.0 的 100% 是稳定平台边界，不是所有高风险自动化都打开。本文只规定 post-100% 能力如何从
`closed` 逐步进入真实 apply；它本身不授权任何 project write、secret resolve、remote worker、
restore apply、release exception write、publish apply、object archive / GC apply、managed ops / upgrade /
support export 或 plugin execution。

## 状态词

高风险能力必须使用稳定状态词，不能只写“done”或“implemented”：

```text
closed:
  能力关闭，只能出现在 roadmap / backlog / blocked reason。

preview_only:
  只读预览、doctor、readiness 或 gate；不创建 command、lease、attempt、artifact、audit 或写入。

fixture_only:
  只在 fixture/temp project 或 isolated environment 中 apply，不触碰真实 managed project。

scoped_rollback:
  可触碰真实 managed project，但必须在同一 command 内恢复 preimage，并证明非目标文件不变。

retained_beta:
  可在严格 allowlist 内保留真实写入结果；仍是单项目、单能力、显式 approval。

production_scoped:
  可在声明 scope 内重复使用；仍受 Command API、permission、approval、rollback 和 audit 约束。

suspended:
  曾经打开但因 failure、rollback fail、secret leak、audit gap 或 project isolation gap 被关闭。
```

`preview_only`、`fixture_only` 和 `scoped_rollback` 不能被累计成 `retained_beta`。每次升级状态都必须有
单独 gate、approval、focused smoke 和 rollback / revocation 证据。

## 全局不变量

- v1.0 release final gate `pass` 不代表任何 v1.x apply 打开。
- 前一个阶梯的 smoke、approval 或 audit 不能替代后一个阶梯的证据。
- R3/R4 能力默认 project-scoped、capability-scoped、resource-scoped、time-scoped。
- R4 能力默认串行开闸；同一 project 上不得同时打开多个新的 R4 apply 面。
- 所有真实 apply 都必须通过 Command API，不允许 CLI、Web、Desktop、Worker、shim 或 plugin 直接写状态。
- 所有真实 project write 必须有 expected-before hash、preimage 或 rollback/remediation plan。
- 所有 secret / token / credential / publish / restore / plugin 能力必须有 revoke 或 disable path。
- Unknown、missing evidence、scope mismatch、hash mismatch、rollback fail 或 audit gap 默认 `blocked`。
- AreaMatrix protected paths 在单独授权前保持关闭：`workflow/versions/**/execution/**`、`progress.json`、
  legacy logs、checkpoint、release evidence 和用户文件安全边界。

## 通用 Apply Packet

任一阶梯从 `closed`、`preview_only` 或 `fixture_only` 进入真实 managed apply 前，必须提交 apply packet：

```text
target rung
target project_key
command_type
actor and human reason
risk_level and risk_policy
required capabilities
affected resources
explicit forbidden resources
gate snapshot
approval scope and expiry
idempotency_key / request_hash policy
expected_version or expected-before hash policy
preimage / rollback / revocation plan
audit event contract
safety facts
focused smoke
failure and suspension rule
AreaMatrix non-touch or protected-path proof when relevant
implementation gap audit update
```

缺少 apply packet 时，该能力只能保持 `preview_only` 或 `fixture_only`。

## 开闸顺序

v1.x 高风险能力按以下顺序打开。后一步不能因为前一步 readiness 为 `ready` 就自动打开。

| 顺序 | 能力 | 第一可开状态 | 进入条件 | 继续禁止 |
|---:|---|---|---|---|
| 0 | v1.0 stable baseline | preview_only | v1.0 final gate / evidence chain 可复验 | 真实 restore / publish / secret / plugin |
| 1 | real generated-only rollback beta | scoped_rollback | fixture generated drill 通过，R3 approval，expected-before，preimage，non-target fingerprint | 保留结果、source write、execution/progress/log 写入 |
| 2 | real generated-only retained apply | retained_beta | rollback beta 多次稳定，focused smoke，rollback verify，allowlisted generated prefix | source write、checkpoint、repair、engine |
| 3 | manual patch artifact | production_scoped artifact-only | write-set preview、patch hash、verification plan、rollback/remediation plan | AreaFlow 写源码、运行 shell、checkpoint |
| 4 | human-applied source evidence | production_scoped read/evidence | 人工 apply 后 diff、changed hash、validation evidence 可导入 | AreaFlow 自动源码写入、自动 repair |
| 5 | source write beta | retained_beta | allowlisted text create/modify、write-set gate、expected-before、verify pass | delete/move/chmod/binary/symlink/glob/root 外路径 |
| 6 | checkpoint apply | retained_beta | source write beta 稳定，dirty state、scope drift、rollback/remediation、audit | 未 verify 通过时 checkpoint |
| 7 | repair plan / repair apply | retained_beta | verify failure evidence、failure summary、repair write-set、新 attempt、re-verify | 覆盖旧 attempt、跳过 verify/checkpoint gate |
| 8 | no-secret engine execution | retained_beta | `secret_ref=none`、command allowlist、budget、redaction、no-secret/no-network facts | secret 注入、远程 worker、未审计网络 |
| 9 | scoped secret resolve | production_scoped R4 | `auth-team-secret-boundary.md` R4-1 到 R4-6 完成，短期 binding、canary redaction、expiry、revoke、audit | 长期 worker secret、跨 project/run 复用 |
| 10 | remote worker credential | production_scoped R4 | R4-7 完成，API-only worker、project/capability/lease scope、rotation/revoke、heartbeat、audit | 直连 PostgreSQL、全局万能 token |
| 11 | restore apply | production_scoped R4 | restore package、hash/signature、isolated temp DB/store、diff、preimage、R4 approval | 直接覆盖当前状态、静默删除、跳过 dry-run |
| 12 | release exception real write | production_scoped R4 | schema preview、migration approval gate、apply preview、R4 approval、audit/revoke | preview 阶段创建 migration 或 record |
| 13 | publish apply | production_scoped R4 | package hash、evidence bundle、tag/sign/upload/push/publish 拆分 command、rollout/rollback | 一键不可回滚发布、跳过签名或包校验 |
| 14 | third-party plugin execution | production_scoped R4 | manifest、capabilities、signature、sandbox、conformance、disable/revoke、audit | 未知插件绕过 permission 或直接读写项目 |
| 15 | external integrations / webhooks | production_scoped external_effect | `integration-webhook-boundary.md` I0-I6 完成，provider allowlist、secret scope、network allowlist、delivery/callback audit | webhook delivery bypass、callback 直接改状态、未知 endpoint |
| 16 | team console | production_scoped control surface | `team-remote-control-boundary.md` T0-T5 完成，R4-3 / R4-4 完成，team role matrix、audit、revoke、project scope | role 自动获得 project write / secret / publish / restore |
| 17 | object artifact store | production_scoped | `object-artifact-retention-contract.md` verifier、hash/size、namespace、retention、backup manifest、restore dry-run integration | 把 skipped object 当作完整 pass；archive copy/upload、GC/delete 越级打开 |
| 18 | budget / quota enforcement | production_scoped | `budget-quota-boundary.md` B0-B6 完成，engine cost model、rate limit、quota policy、audit、override approval | 无上限 engine spend、silent throttling、重复扣费 |
| 19 | managed ops / upgrade / support export | production_scoped R4 | `operations-deployment-observability-boundary.md` O0-O8 完成，auth/team scope、redaction、destination allowlist、backup/preimage、approval、audit、retention/revoke | 自动升级、无 preimage rollback、默认遥测、support bundle 携带 prompt/secret/user files/raw artifact |

Auth、token、team 和 secret 的细分顺序仍以
[`auth-team-secret-boundary.md`](auth-team-secret-boundary.md) 为准。Team console 是控制面，不是能力放大器；
team admin 不能绕过 project config、Command API、R3/R4 approval、secret scope、restore gate 或 publish gate。
Team permission enforcement 是 R4 auth 子阶梯；Team console 是更晚的产品控制面。前者通过不代表后者已打开。
Team / remote control surface 的观察、审批、操作、管理、安全、release 和 restore 分层见
[`team-remote-control-boundary.md`](team-remote-control-boundary.md)；Team Console 的某个页面或按钮存在，
不代表对应 command rung 已打开。
Budget / quota enforcement 必须按 [`budget-quota-boundary.md`](budget-quota-boundary.md) 逐级打开；
estimate 不是 charge，reservation 不是 approval，quota ready 不是 execution ready。
External integrations、webhooks、third-party callbacks 和 provider automation 必须按
[`integration-webhook-boundary.md`](integration-webhook-boundary.md) 逐级打开；callback verified 不是
approval，delivery planned 不是 delivery sent，network allowed 不是 external API write approved。
Operations、deployment、observability、support bundle、telemetry、upgrade 和 rollback 必须按
[`operations-deployment-observability-boundary.md`](operations-deployment-observability-boundary.md) 逐级打开；
service status 不是 process control，support bundle preview 不是 export，local diagnostics 不是 remote
telemetry opt-in。

## Rung-specific Rules

### Project Write

Generated-only、source write、checkpoint 和 repair 必须从最小写面开始：

```text
fixture rollback
-> real scoped rollback
-> retained generated apply
-> manual patch artifact
-> human-applied evidence
-> source write beta
-> checkpoint
-> repair
```

任何真实 project write 都必须证明：

```text
target path inside project root
target path allowlisted
forbidden path deny checked
expected-before hash matched
preimage captured when modifying existing file
non-target fingerprints unchanged when required
verify evidence exists
rollback or remediation exists
audit event written
```

第一版 source write 只允许 text `create` / `modify`。Delete、move、chmod、binary rewrite、symlink target、
glob 批量写入和 project-root 外路径必须保持 blocked。

### Engine / Secret / Remote Worker

No-secret engine execution 必须先于 secret-backed execution。Secret-backed execution 必须按
[`auth-team-secret-boundary.md`](auth-team-secret-boundary.md) 完成：

```text
security boundary doctor
-> local token fixture
-> optional local auth enforcement
-> team permission enforcement
-> secret store preview
-> scoped secret resolve
-> remote worker credential
```

Secret resolve 只能为一个 command / run / lease 生成短期 binding。Secret value 不得写入 project config、
command request payload、stdout、stderr、artifact、event、audit、backup 或 worker 长期状态。

Remote worker 只能通过 API，不直连 PostgreSQL。Worker credential 不能同时具备任意 project write、
secret resolve、restore apply 或 publish apply。

### Restore

Restore apply 必须从 package 和隔离验证开始：

```text
backup manifest
-> artifact integrity
-> restore dry-run
-> restore package format
-> package hash / signature policy
-> isolated temp database validation
-> isolated temp artifact store validation
-> diff against current state
-> preimage / rollback plan
-> R4 approval
-> scoped apply
```

Restore apply 不能直接覆盖当前 PostgreSQL、artifact store 或 managed project。任何 local artifact missing /
hash mismatch、secret leak risk、broken manifest 或无 rollback 的 restore 都必须 blocked。

### Object Artifact Store

Object artifact store、archive copy/upload、retention-aware GC 和 delete apply 必须按
[`object-artifact-retention-contract.md`](object-artifact-retention-contract.md) 单独开闸：

```text
object backend schema metadata
-> object verifier preview
-> object write fixture
-> object read integrity fixture
-> object upload scoped beta
-> object restore dry-run integration
-> object retention policy preview
-> object archive copy / upload command
-> object delete / GC preview
-> object delete / GC apply
```

`object` verifier 没有通过前，object artifact 只能是 `skipped` / `needs_attention`，不能被 release
final gate、restore dry-run 或 backup manifest 计入完整可恢复内容。Archive copy/upload 不得删除 source；
GC/delete 第一版只能处理 AreaFlow-owned `ephemeral` artifacts，不能触碰 `audit`、`release`、受保护的
`run_evidence`、`external_ref`、`legal_hold`、未知 retention class、hash mismatch local artifact 或
verifier skipped/failed object artifact。

### Release Exception And Publish

Release exception real write 必须等以下链路完成：

```text
acceptance preview
-> acceptance gate
-> exception doctor
-> exception record preview
-> exception schema preview
-> exception migration approval gate
-> exception apply preview
-> explicit R4 approval
```

Publish apply 必须拆成独立 command：

```text
package create
package verify
tag prepare
signature apply
upload artifact
push tag
publish release
rollout observe
```

每一步都必须绑定 evidence hash、actor、approval、rollback/remediation 或 revoke/hide release 计划。
Publish gate 或 final gate 不能替代 publish approval。

### Plugin

第三方 plugin execution 最后打开。完整 install / enable / execute ladder 见
[`plugin-marketplace-boundary.md`](plugin-marketplace-boundary.md)。Plugin 必须声明 capability、
resource access、engine access、network access 和 artifact access。Plugin 只能通过受治理
adapter/profile/command/engine 接口运行，不得直接读写 managed project、PostgreSQL、artifact store
或 secret provider。

## Suspension Rule

任何已打开能力命中以下情况，必须立即降级到 `suspended`：

```text
project isolation failure
permission deny bypass
unexpected project file modification
expected-before mismatch ignored
rollback fail
local artifact hash mismatch
secret or token leak
token scope bypass
token revoke ignored
membership change audit missing
secret revoke ignored
remote worker credential misuse
remote worker credential revoke ignored
restore diff mismatch
publish package hash mismatch
object hash / size mismatch
object key outside project namespace
delete affected protected retention class
plugin sandbox escape
webhook delivery without audit
callback mutates state directly
external API write bypasses Command API
silent throttling
quota double charge
budget override without expiry
audit event missing for successful apply
```

恢复前必须新增 remediation evidence、revocation / rollback proof、focused regression test 和 explicit
approval。不得通过修改历史 event / audit 来“修复”事故。

## AreaMatrix First Policy

AreaMatrix 是第一 dogfood project，但不是高风险能力试验场。AreaMatrix v1.x 顺序必须更保守：

1. 先 generated-only rollback beta，再 retained generated apply。
2. source write beta 前必须经历 manual patch artifact 和 human-applied source evidence。
3. no-secret engine execution 先在 fixture 或非 AreaMatrix project 证明。
4. secret-backed AreaMatrix execution 必须有 R4 approval、redaction evidence、rollback plan 和 protected path proof。
5. remote worker 不得先于 local / host-bound worker 在 AreaMatrix 上打开。
6. restore apply、publish apply 和 third-party plugin execution 不得以 AreaMatrix 为第一试验对象，除非用户单独授权并接受 R4 apply packet。

`workflow/README.md`、`.areaflow/status.json`、compatibility shim 和 AreaMatrix protected paths 的真实写入，
仍需各自 gate 和授权；本文不能替代跨仓库写入批准。

## 关闭条件

某个 v1.x high-risk task 进入 active 前，至少需要：

```text
this ladder references the target rung
task-local design contract exists
Command API endpoint / CLI command defined
permission and risk policy defined
approval and expiry rules defined
expected-before / preimage / rollback / revoke path defined
audit contract defined
focused smoke plan defined
failure suspension rule defined
AreaMatrix non-touch or protected-path proof defined
```

如果上述证据不完整，该 task 只能留在 deferred backlog，不能进入 implementation。
