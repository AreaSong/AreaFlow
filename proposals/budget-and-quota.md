# Budget And Quota Boundary

> Status: Proposed. 当前产品未开放 usage metering、budget、quota 或 rate limit enforcement。

## Purpose

本文定义 AreaFlow engine cost、usage metering、budget policy、quota policy 和 rate limit enforcement 的
长期边界。它补充
[`execution-model.md`](../docs/concepts/execution-model.md)、
[`security.md`](../docs/architecture/security.md)、
[`commands-and-approvals.md`](../docs/concepts/commands-and-approvals.md)、
[`auth-team-secret-boundary.md`](./auth-team-secret.md) 和
[`high-risk-apply-ladder.md`](./high-risk-apply.md)。

Budget / quota 是安全和成本控制面，不是 billing system。v1.0 只允许 schema/readiness/preview、
blocked reason 和 audit gap；真实 quota enforcement、rate-limit decrement、spend blocking、provider
billing reconciliation 和 paid billing 都属于 v1.x rung 17。

## Non-goals Before v1.x

v1.0 前禁止把以下能力解释为已打开：

- 真实 spend blocking。
- 真实 quota decrement 或 token bucket 消耗。
- 自动 provider billing reconciliation。
- 团队/用户计费、发票、付款、余额、退款或账务。
- 由于预算不足自动取消 run / task / worker lease。
- silent throttling，也就是不写 audit、不返回 blocker 就悄悄降速或拒绝。
- 用 budget ready 替代 execution approval、secret resolve、network allowlist 或 engine gate。

v1.0 可以做：

- `engine_profiles.budget_policy` metadata。
- schedule / engine / execution preview 中的 `budget_policy_missing`、`quota_policy_missing`、
  `rate_limit_policy_missing` blocked reason。
- 只读 cost estimate preview。
- audit coverage gap。

## Concepts

```text
usage_meter:
  append-only usage event，记录 estimated / observed engine usage metadata。

budget_policy:
  project / team / actor / engine profile 的预算上限和审批要求。

quota_policy:
  tokens、requests、runs、worker minutes、artifact bytes 等资源的使用上限。

rate_limit_policy:
  短时间窗口内的请求/任务/engine 调用限制。

budget_reservation:
  command 执行前的可撤销预占，不等于最终花费。

budget_charge:
  provider 或 engine 返回后的实际用量记录。

override_approval:
  超预算或临时提高额度的显式 approval。
```

Estimate 不是 charge。Reservation 不是 approval。Quota ready 不是 execution ready。

## Scope Model

Budget / quota 必须可以绑定到多个 scope，但执行时必须收敛到具体 project：

```text
project_id / project_key
team_id nullable
actor_id nullable
engine_profile_id nullable
provider
run_id nullable
run_task_id nullable
command_request_id nullable
worker_id nullable
lease_id nullable
time_window
resource_kind
```

第一版真实 enforcement 只允许 project-scoped policy。Team / actor / provider 级汇总可以先做 preview，
不能在没有 project scope 的情况下阻断或放行 run。

## Metering Rules

Usage 记录必须 append-only：

```text
usage_meters:
  project_id
  command_request_id nullable
  run_id nullable
  run_task_id nullable
  attempt_id nullable
  engine_profile_id nullable
  actor_id
  provider
  resource_kind
  quantity_estimated
  quantity_observed nullable
  unit
  source: estimate | provider_report | manual_adjustment
  idempotency_key
  audit_event_id
```

同一 provider report 必须幂等；重复上报不能重复扣费。Manual adjustment 必须有 human reason 和 audit。
Usage record 不能包含 prompt、secret、provider raw response、token 明文或用户文件内容。

## Opening Ladder

### B0 Metadata And Readiness

Status: current v1.0 boundary.

Allowed:

- 在 engine profile / project config 中引用 budget policy 名称。
- 在 preview 中返回 missing / disabled / not_enforced。
- 在 release readiness 和 audit coverage 中暴露 budget gap。

Forbidden:

- 扣减 quota。
- 阻断真实 run。
- 写 usage charge。
- 调用 provider billing API。

Required facts:

```text
budget_enforcement_open=false
quota_decrement_open=false
rate_limit_enforcement_open=false
provider_billing_sync_open=false
usage_charge_written=false
silent_throttling=false
```

### B1 Read-only Estimate Preview

Status: future preview-only.

Allowed:

- 根据 command plan、engine profile、model/provider metadata 和 historical estimate 生成 cost estimate。
- 返回 estimated units、confidence、unknowns、policy blockers。
- 给 Web / Desktop / CLI 展示预算风险。

Forbidden:

- 创建 reservation。
- 创建 charge。
- 改变 scheduler result。
- 阻断 command apply。

### B2 Quota Policy Doctor

Status: future preview-only.

Allowed:

- 检查 policy shape、scope、time window、resource kind、override policy 和 audit coverage。
- 发现 unbounded engine、missing override approval、missing revoke / disable path。

Forbidden:

- 修改 policy。
- 执行 rate limit。
- 回写 provider spend。

### B3 Fixture Reservation And Charge

Status: future fixture-only.

Allowed:

- 在 fixture/temp project 中创建 reservation、release reservation、write estimated charge。
- 验证 idempotency、over-quota denial、rollback / release path。

Forbidden:

- 真实 managed project spend blocking。
- provider billing API。
- 影响 AreaMatrix run / task / worker lease。

### B4 Project-scoped Enforcement Beta

Status: v1.x retained_beta.

Allowed:

- 对单个 non-AreaMatrix project 或 explicit approved project 打开 project-scoped budget enforcement。
- 在 Command API apply 前创建 reservation。
- 成功后写 charge，失败 / cancel / timeout 后 release reservation。
- 返回 explicit `blocked:budget_exceeded` 或 `blocked:quota_exceeded`。

Forbidden:

- Silent throttling。
- Cross-project pooled quota。
- Team-level billing。
- 用 quota override 跳过 R3/R4 approval。

Required evidence:

- Reservation idempotency。
- Charge idempotency。
- Release on cancel / failure / timeout。
- Over-quota denial has audit event。
- Override approval scope and expiry。
- No secret / prompt / provider raw response in usage rows。

### B5 Team / Actor / Provider Aggregation

Status: v1.x after project enforcement.

Allowed:

- 汇总 project usage 到 team / actor / provider view。
- 基于 team / actor quota 进一步缩小 project command allowance。

Forbidden:

- 没有 project scope 的全局阻断。
- Team admin 静默提高预算。
- Usage aggregation 泄露跨项目存在性或敏感 metadata。

### B6 Provider Billing Reconciliation

Status: v1.x after secret and integration design.

Allowed:

- 读取 provider billing report metadata。
- 对账 AreaFlow usage meters 与 provider report。
- 生成 discrepancy report 和 manual adjustment preview。

Forbidden:

- 读取 provider secret without scoped secret resolve。
- 将 provider raw billing payload 写入 unrestricted artifact。
- 自动向用户收费、退款或开票。

## Enforcement Order

Budget / quota 只能作为 Command API preflight 和 engine execution gate 的一个输入：

```text
auth / actor
project scope
permission preflight
gate result
approval scope
budget / quota preflight
secret / network policy
reservation
command apply
usage charge or reservation release
audit event
```

Budget pass 不能替代 permission、approval、secret、network、worker scope 或 project path allowlist。
Budget fail 必须返回机器可读 blocker，并写 audit；不得悄悄排队、降级或丢弃任务。

## Override Policy

Override 是高风险控制面，必须有：

```text
actor
human reason
target project / policy / resource kind
old limit
new limit
expires_at
approval record
audit event
rollback / revoke path
```

Override 不能扩大 project file write scope、secret scope、network scope、publish scope 或 restore scope。

## AreaMatrix Dogfood Policy

AreaMatrix 第一阶段不能作为真实 budget enforcement 的试验对象。

AreaMatrix 允许：

- no-secret engine execution 前的 budget readiness。
- cost estimate preview。
- release readiness 中的 budget gap。

AreaMatrix 禁止：

- budget enforcement 阻断 `./task-loop run` compatibility behavior。
- quota decrement 影响 legacy workflow / progress。
- provider billing reconciliation 读取 AreaMatrix secret 或用户文件。
- 因 budget policy 自动修改 `workflow/versions/**/execution/**`、`progress.json`、logs、checkpoint 或
  release evidence。

## Suspension Rule

Budget / quota 能力命中以下情况必须立即降级到 `suspended`：

```text
silent throttling
quota decrement without command/audit
duplicate provider report double-charged
reservation not released after failure/cancel/timeout
override without expiry
budget policy bypassed by engine execution
cross-project usage visible without scope
usage row contains prompt, secret, raw provider response, or user file content
provider billing sync uses unscoped secret
AreaMatrix run blocked or modified without explicit approval
```

恢复前必须补 remediation evidence、charge correction / reservation release proof、focused regression test 和
explicit approval。
