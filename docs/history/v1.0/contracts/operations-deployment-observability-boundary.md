# Operations, Deployment, And Observability Boundary

## Purpose

本文定义 AreaFlow 从本机 CLI / local service 走向可安装、可诊断、可升级、可支持的平台时的运维边界。
它覆盖 install、bootstrap、migration、service lifecycle、health / readiness、logs、metrics、traces、
support bundle、diagnostics、Admin API、telemetry 和 upgrade / rollback。

v1.0 的目标是证明 AreaFlow 可被本机稳定运行和诊断，不是打开托管升级、远程运维控制或自动导出用户数据。
远程 Team Console / command console 的产品边界见
[`team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md)；auth、API token、secret 和 remote
worker credential 的 R4 边界见 [`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)。

## Current Boundary

v1.0 允许：

- 本机 install / migrate / start / register project smoke。
- `health`、`readiness`、`doctor` 和 `service status` 查询。
- migration ledger、bootstrap evidence 和 release readiness 输入。
- backup manifest、restore dry-run、artifact integrity 和 release evidence preview。
- 本地日志和诊断摘要，默认脱敏。
- support bundle preview / metadata-only export plan。
- Desktop 只读 service status、service-control gate、notification gate 和 tray/menu gate。

v1.0 禁止：

- 自动升级。
- 破坏性 rollback。
- 远程运维控制。
- 默认远程 telemetry。
- support bundle 默认包含 secrets、完整 prompt、用户文件、raw artifact 或未脱敏日志。
- Desktop 绕过 AreaFlow API 直接启动/停止 worker、执行 workflow 或修改项目。
- Admin API 绕过 Command API 改写 workflow、run、lease、worker、artifact、secret、release 或 project file
  业务状态。

## Core Invariants

1. **Admin API 不是第二套业务写入口**。
   Admin API 只做 migrate、bootstrap、diagnose、backup manifest、restore dry-run、service status 和
   import/export 类运维入口。任何改变 workflow business state 的动作必须回到 Command API。

2. **Observability 不携带敏感原文**。
   logs、metrics、traces、events、audit 和 diagnostics 默认不得包含 secret value、API token、完整
   prompt、用户文件内容、raw artifact 内容或未脱敏 stdout / stderr。

3. **Support bundle 先 preview，后 export**。
   v1.0 只能生成 metadata-only bundle plan：scope、hash、path reference、row counts、version、blocked
   reasons 和 redaction summary。真实导出必须显式选择范围、脱敏、hash、审批和 audit。

4. **Migration / upgrade 必须可审计**。
   schema migration、data migration、project import、mirror、cutover 和 release exception migration 都必须有
   preflight、ledger、expected version、rollback / remediation wording 和 verification evidence。

5. **Telemetry 默认留在本机**。
   AreaFlow 可以本地记录 health、diagnostic 和 audit summary；任何远程上报都必须 opt-in，并经过 auth、
   secret redaction、scope、destination allowlist 和 audit。

6. **AreaMatrix protected paths 不能被 ops 工具触碰**。
   运维命令不能借 migrate、support bundle、doctor、cleanup、rollback 或 restore 之名修改 AreaMatrix
   `workflow/README.md`、`.areaflow/status.json`、`scripts/**`、`workflow/versions/**`、execution、
   `progress.json`、logs、checkpoint 或 release evidence。

## Operations Surfaces

AreaFlow 运维面分为五类：

```text
bootstrap:
  install, migrate, server start, project registration, smoke

service lifecycle:
  status, health, readiness, service-control gate, local process manager preview

diagnostics:
  doctor, audit coverage, artifact integrity, permission doctor, support bundle preview

release operations:
  backup manifest, restore dry-run, release readiness, final gate, package/distribution/publish preview

managed operations:
  remote read-only ops, remote control, managed upgrade, destructive rollback, support export apply
```

前四类可以在 v1.0 内以本机、只读或 preview 形态收敛。`managed operations` 全部属于 v1.x，且必须先满足
auth/team/secret/remote control 边界。

## Health And Readiness

`health` 只回答进程是否活着、API 是否可响应、DB 是否可连接。`readiness` 回答当前是否具备执行某个
平台能力的前置条件。`doctor` 回答具体配置、权限、artifact、migration、adapter、worker 或 release 链路
哪里不满足。

推荐状态分层：

```text
health:
  live | degraded | down

readiness:
  ready | needs_attention | blocked

doctor item:
  pass | warn | fail | skipped

operation gate:
  pass | needs_approval | blocked
```

`health=live` 不能替代 `readiness=ready`，`doctor=pass` 不能替代 Command API approval，`service
status` 不能替代 release final gate。

## Logs, Metrics, And Traces

v1.0 的 observability 只要求本机可诊断，不要求远程 APM：

- logs 记录 service、API、migration、worker preview、doctor 和 release preview 的摘要。
- metrics 记录 counters / gauges，例如 request count、DB readiness、worker pool totals、queue depth、
  diagnostic duration 和 artifact counts。
- traces 可以先作为 request correlation ID / command request ID / run ID / lease ID，不要求接入外部 tracing。
- 所有 observability record 必须绑定 actor / command / project scope 或明确是 system scope。
- redaction policy 必须先于远程 telemetry、support bundle export 和 secret resolve 真实打开。

禁止记录：

```text
secret values
API token values
full prompt text
user file contents
raw artifact contents
unredacted stdout / stderr
private environment values
provider credentials
```

## Support Bundle

Support bundle v1.0 只做 preview / metadata-only：

```text
bundle_id
scope
projects[]
time_window
included_metadata[]
excluded_sensitive_content[]
hashes[]
path_references[]
redaction_policy
approval_required
audit_plan
export_open=false
```

默认包含：

- AreaFlow version、migration version、schema hash。
- project keys、workflow version labels、run/task/attempt IDs。
- artifact metadata、hash、size、backend 和 relation。
- health/readiness/doctor summary。
- release readiness / final gate summary。
- audit coverage summary。
- logs 的脱敏摘要或 log index，不含 raw log。

默认排除：

- secret values、token values、private env。
- prompt 原文、用户文件内容、raw artifact。
- 未脱敏 stdout / stderr。
- 被管理项目文件副本。
- AreaMatrix 历史 execution logs / progress / evidence 原文。

当前只读实现入口：

```text
GET /api/v1/ops/support-bundle-preview
areaflow support bundle-preview --json
```

真实 support bundle export 是 v1.x 操作：必须显式 scope、redaction proof、hash manifest、approval、audit、
retention 和 revoke/delete story。

## Install, Migration, Upgrade, And Rollback

v1.0 install / migration 目标：

```text
areaflow migrate
areaflow server
areaflow service status
areaflow project add
areaflow project import
areaflow backup manifest
areaflow backup restore-plan
areaflow release readiness
```

Migration 必须记录：

```text
migration_id
schema_version_from
schema_version_to
started_at
finished_at
actor
preflight_result
apply_result
verification_result
rollback_or_remediation
audit_event_id
```

当前只读 readiness 入口：

```text
GET /api/v1/ops/migration-ledger-readiness
areaflow ops migration-ledger-readiness --json
```

v1.0 rollback 只允许 documentation / remediation / restore dry-run 级别，不执行破坏性 DB 回滚、project file
覆盖或 artifact delete。真实 managed upgrade / rollback 属于 v1.x R4 operation，必须有 isolated dry-run、
backup manifest、restore plan、preimage、approval、verification 和 audit。

## Desktop And Local Service

Desktop v0.9 是 local service shell，不是第二套运维平台：

- 可以展示 API、PostgreSQL、worker pool、dashboard URL 和 service status。
- 可以展示 start / stop / restart 为什么 disabled 或需要 approval。
- 可以打开 Web dashboard。
- 可以展示 notification / tray/menu gate。
- 不维护第二数据库。
- 不直接执行 workflow。
- 不读取或缓存 secret 明文。
- 不绕过 AreaFlow API 控制 worker、run、lease 或 project files。

真实 process manager、auto restart、background service install、OS notification bridge 和 native tray/menu 都需要
各自 gate evidence。远程 Team Console 不是 Desktop v0.9 的默认能力。

## Opening Ladder

### O0 Local Bootstrap And Readiness

Allowed:

- 本机 migration、server start、project registration、service status。
- health/readiness/doctor。
- migration ledger 和 smoke evidence。

Forbidden:

- auto upgrade。
- destructive rollback。
- remote telemetry。
- support bundle export。

Proof:

- 空 PG -> migrate -> server -> project add/import -> status/doctor 可重复。

### O1 Diagnostic Bundle Preview

Allowed:

- support bundle metadata-only preview。
- redaction policy preview。
- local diagnostic summary。

Forbidden:

- raw log、prompt、artifact、secret 或用户文件导出。
- 自动上传。

Proof:

- preview 列出 included / excluded / hash / approval_required。
- secret canary 不出现在 preview 输出。

### O2 Local Service Control Fixture

Allowed:

- service-control gate。
- fixture-only start/stop/restart plan。
- drain/restart recovery preview。

Forbidden:

- 真实 process control。
- worker kill 或 run cancel apply。

Proof:

- `process_control_attempted=false`。
- no command / lease / attempt created.

### O3 Package, Sign, And Notarize Preview

Allowed:

- release package preview。
- signing/notarization requirement preview。
- distribution preview。

Forbidden:

- create archive、sign、notarize、upload、publish。

Proof:

- package hash plan、missing cert / keychain readiness、publish gate blocked reason。

### O4 Optional Local Process Manager

Allowed:

- 用户显式 opt-in 的 local process manager。
- start/stop/restart with preflight、drain、audit。

Forbidden:

- 默认后台驻留。
- 未经 gate 的 worker 或 workflow 操作。

Proof:

- opt-in config、audit event、recovery plan、manual disable path。

### O5 Remote Ops Read-only

Allowed:

- 远程只读 health/readiness/diagnostic dashboard。

Forbidden:

- command apply。
- secret resolve。
- support export。

Proof:

- auth/token/team scope 生效。
- project visibility guard 通过。
- telemetry redaction 通过。

### O6 Remote Ops Control

Allowed:

- 远程 command console 通过 Command API 发起受保护运维命令。

Forbidden:

- Admin API 直接改业务状态。
- role=admin 越过 permission/gate/approval。

Proof:

- Team Console T5 / auth R4 证据齐全。
- command request、approval、audit 和 rollback/remediation 完整。

### O7 Managed Upgrade And Rollback

Allowed:

- managed upgrade plan -> dry-run -> approved apply。
- rollback / remediation apply。

Forbidden:

- 静默升级。
- 无 preimage / backup / restore plan 的 DB 或 project file 覆盖。

Proof:

- isolated dry-run、backup manifest、schema diff、preimage、approval、verification、audit。

### O8 Support Bundle Export Apply

Allowed:

- 明确 scope 的 support bundle export。

Forbidden:

- 默认包含 prompt、secret、user files、raw artifact 或 unredacted logs。
- 自动上传未知 endpoint。

Proof:

- redaction test、hash manifest、approval、audit、retention / revocation plan。

## AreaMatrix First Policy

AreaMatrix dogfood 阶段，运维命令必须保持更保守：

- `doctor` / `service status` 可以读取 AreaFlow 和 AreaMatrix metadata。
- support bundle preview 只能引用 AreaMatrix 历史路径和 hash，不复制原文。
- migration / rollback 不能修改 AreaMatrix protected paths。
- Desktop 只能打开 AreaFlow dashboard 或展示 blocked gate，不能编辑 AreaMatrix shim。
- `./task-loop run` 转发必须等 execution cutover approval，不属于运维 shortcut。

## v1.0 Closing Conditions

进入 v1.0 release review 前，operations boundary 至少需要证明：

```text
install / migrate / start / register smoke exists
health / readiness / doctor are distinct
service status and Desktop gates are read-only
Admin API cannot mutate workflow business state
support bundle preview is metadata-only and redacted
logs / diagnostics exclude secret, prompt, user file and raw artifact content
migration ledger records preflight, apply, verify and remediation
backup manifest and restore dry-run feed release readiness
telemetry is local-only by default
AreaMatrix protected paths remain untouched
remote ops, managed upgrade and support export apply are deferred to v1.x
```

如果这些事实只能通过口头解释、未覆盖 smoke 或间接测试支撑，operations boundary 仍视为未完成。
