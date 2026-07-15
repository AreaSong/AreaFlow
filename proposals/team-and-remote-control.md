# Team And Remote Control Boundary

> Status: Proposed. 当前产品已开放 OIDC 登录和 project-scoped Access 页面，但未开放 team lifecycle、邀请、远程 worker 或通用远程控制面。

## Purpose

本文定义 AreaFlow Team Console、远程控制台和多用户控制面的边界。它补充
[`auth-team-secret-boundary.md`](./auth-team-secret.md)、
[`security.md`](../docs/architecture/security.md)、
[`api.md`](../docs/reference/api.md) 和
[`high-risk-apply-ladder.md`](./high-risk-apply.md)。

Team Console 是控制面，不是能力放大器。它可以把已有 Query API、Command API、approval、audit、
worker 和 release / restore preview 组合成多人可用界面；它不能绕过 project config、Command API、
permission、gate、approval、secret scope、worker lease、restore gate、publish gate 或 audit。

当前 project role binding 只覆盖已认证用户的项目角色管理，不等于 Team Console。团队生命周期、邀请、
OIDC group 自动映射、remote worker credential、secret resolve 和通用远程 command apply 仍属于后续 R4 能力。

## Non-goals Before v1.x

以下能力仍不得解释为已打开：

- Team invitation、membership lifecycle 或 OIDC group 自动映射。
- Team role 对 project capability 之外能力的授权。
- 远程 Web 控制台发起 Command API 写操作。
- Desktop 变成远程团队控制台。
- 远程 worker credential、远程 worker dispatch 或跨机器 execution。
- 在 UI 中读取、显示、缓存或注入 secret 明文。
- 用 team admin 角色绕过 R3/R4 approval、restore、publish、plugin、secret 或 worker gate。

当前 Access 页面只管理 project role binding，不开放上述能力。

## Control Surface Classes

Team Console 能力必须按控制面分类，不允许用一个 `admin=true` 打开所有功能：

```text
observe:
  Query API、SSE、dashboard、audit、release readiness、execution cutover readiness。

approve:
  approval record、approval preview、scope / expiry / risk / affected resources。

operate:
  run start/drain/cancel、worker lease operation、status projection apply、archive preview。

administer:
  project membership、team membership、api token metadata、project config metadata。

security:
  token issuance / revoke、secret_ref metadata、secret resolve、worker credential。

release:
  release exception、package、publish、rollout。

restore:
  restore package、dry-run、apply。
```

每一类都必须绑定 project scope、actor、role ceiling、capability、resource、risk level 和 audit event。
`observe` 不是 `operate`；`approve` 不是 `apply`；`administer` 不是 `security`；`release` 和 `restore`
不能被普通 team admin 自动继承。

## Role Ceiling

推荐角色只表达上限，不表达最终许可：

```text
owner:
  可以管理团队和项目成员上限；不能绕过 R3/R4 gates。

admin:
  可以管理 project settings metadata 和低风险控制面；不能自动获得 secret / publish / restore。

operator:
  可以请求或执行已批准的 operation command；不能批准自己的高风险操作，除非 policy 另行允许。

approver:
  可以按 approval scope 批准指定 command；不能扩大 affected resources。

viewer:
  只能观察允许 project scope 内的 Query API。

auditor:
  只能观察 audit / evidence / release readiness，不发起业务写入。
```

Role 是 ceiling。最终许可仍必须同时满足：

```text
authenticated actor
project membership scope
project config capability
resource allowlist / denylist
command class
risk policy
gate result
approval scope and expiry
precondition snapshot
audit contract
```

## Opening Ladder

### T0 Reserved Schema And Readiness

Status: current v1.0 boundary.

Allowed:

- `users`、`teams`、`memberships`、`project_memberships`、`api_tokens` schema / metadata。
- Security boundary readiness、role matrix preview、blocked reason。
- Local single-user Web / Desktop observation。

Forbidden:

- 登录、session、remote access。
- membership 改变 API authorization。
- token issuance enforcement。
- remote command apply。

### T1 Read-only Team Console Preview

Status: future preview-only.

Allowed:

- 展示 project list、role matrix、membership metadata、audit trail、approval queue preview。
- 展示每个 action 的 required capability、approval、risk 和 blocker。
- SSE 只观察同一 auth / project scope 下的事件。

Forbidden:

- 创建或更新 membership。
- 创建 token。
- 写 command request。
- 调度 worker。
- 解析 secret。

Required response facts:

```text
mode = read_only_team_console_preview
team_console_open=false
remote_control_open=false
membership_write_open=false
token_issuance_open=false
command_apply_open=false
secret_resolve_open=false
worker_credential_open=false
authorization_changed=false
```

### T2 Local Auth-Enforced Console

Status: v1.x after `auth-team-secret-boundary.md` R4-3.

Allowed:

- 本机 opt-in auth enforcement。
- CLI / Web / Desktop 通过 shared API auth header 访问。
- Query API 按 token / actor / project scope 限制可见性。

Forbidden:

- 远程团队控制台。
- team role enforcement。
- secret resolve。
- remote worker credential。
- 任何未通过 Command API 的写操作。

Required evidence:

- Enforcement off backward compatibility smoke。
- Valid / missing / expired / revoked token tests。
- Global run / artifact ID project guard tests。
- Logs、events、audit、artifact 不包含 token 明文。

### T3 Team Permission Enforcement

Status: v1.x after `auth-team-secret-boundary.md` R4-4.

Allowed:

- `users`、`teams`、`memberships`、`project_memberships` 影响 Query API 和 Command API preflight。
- Membership write 通过 Command API 记录 audit。
- Role matrix tests 覆盖 project x role x capability x resource。

Forbidden:

- role 自动打开 project write、secret、remote worker、restore、publish 或 plugin execution。
- membership change 不写 audit。
- admin 自批高风险 command，除非 policy 明确允许并写 audit。

### T4 Remote Read-only Console

Status: v1.x after T2/T3 and transport security design.

Allowed:

- 远程 read-only dashboard。
- Project-scoped Query API、SSE 和 audit 观察。
- Release / restore / execution readiness preview。

Forbidden:

- 远程 Command API apply。
- 远程 service process control。
- 远程读取本机 artifact path。
- 远程显示 secret 明文。

Required evidence:

- Origin / CORS / CSRF or equivalent browser protection design。
- TLS / reverse proxy / local tunnel threat model。
- Session expiration / revoke tests。
- Cross-project denial and no existence leak policy。

### T5 Remote Command Console

Status: v1.x after T4 and command-specific approval.

Allowed:

- 只对已列入 allowlist 的 command 发起 remote request。
- 每个 command 仍使用 Command API、idempotency key、request hash、permission preflight、approval、
  precondition snapshot、safety facts 和 audit。
- R0 / R1 / low-risk operation 可先开；R2-R4 必须按各自 ladder 单独批准。

Forbidden:

- 远程 console 直接写 PostgreSQL、artifact store 或 managed project。
- 跳过 approval 直接 apply。
- 把 Web button 或 Desktop menu 当作 permission。

### T6 Remote Worker And Security Operations

Status: v1.x after relevant R4 rungs.

Allowed only when corresponding rungs are open:

- Remote worker credential: requires `auth-team-secret-boundary.md` R4-7。
- Secret resolve: requires R4-6。
- Restore apply: requires high-risk ladder rung 11。
- Publish apply: requires high-risk ladder rung 13。
- Plugin execution: requires high-risk ladder rung 14。

Forbidden:

- Team Console 一次性打开所有 security operations。
- 用 team admin 绕过 scoped secret binding、worker credential scope、restore package validation 或 publish approval。

## Desktop Relationship

Desktop v0.9 是本机 local service shell，不是 Team Console。它可以：

- 打开 Web dashboard。
- 展示 service status、notification gate、tray/menu gate。
- 展示 security boundary readiness 和 blocked reason。

Desktop 不得：

- 维护团队 session 或远程控制状态源。
- 直接改 project、workflow、worker、secret 或 release 状态。
- 保存长期 API token 或 secret 明文。
- 绕过 API/service layer 调用本机或远程 worker。

如果未来 Desktop 承载团队登录 UI，它仍只是 Team Console 的 API client；所有权限、command、approval 和
audit 语义仍归 AreaFlow API。

## AreaMatrix Dogfood Policy

AreaMatrix 是第一 dogfood project，但不能作为远程团队控制台的第一试验场打开高风险能力。

AreaMatrix 规则：

- v1.0 不允许远程 Team Console 修改 AreaMatrix。
- Execution cutover 前，Team Console 只能观察 AreaMatrix readiness、status projection、audit 和 blocked reason。
- `workflow/versions/**/execution/**`、`progress.json`、legacy logs、checkpoint、release evidence 和用户文件安全边界仍是 protected path。
- 任何远程 apply 都必须证明 AreaMatrix protected paths 未被触碰，或者已有单独 explicit approval。

## Suspension Rule

Team / remote control surface 一旦命中以下情况，必须降级到 `suspended`：

```text
cross-project data visible
role grants capability outside project scope
membership change without audit
token plaintext persisted
token revoke ignored
session revoke ignored
remote command bypasses Command API
remote console writes PostgreSQL directly
remote console exposes local artifact path as writable state
secret displayed or cached in UI
worker credential issued without scope
admin bypasses R3/R4 approval
AreaMatrix protected path modified without explicit approval
```

恢复前必须补 remediation evidence、revocation / rollback proof、focused regression test 和 explicit approval。
