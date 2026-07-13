# Auth, Team, And Secret Boundary

## Purpose

本文定义 AreaFlow 从 single-user local service 走向 team / remote / secret-backed platform 前的 R4
安全边界。它只收束设计、验收和开闸顺序，不启用真实登录、API token enforcement、team permission
enforcement、secret resolve 或 remote worker credential issuance。
Team Console 和远程控制台作为产品控制面的分层见
[`team-remote-control-boundary.md`](team-remote-control-boundary.md)；本文中的 team permission
enforcement 通过，不代表 Team Console 或远程 command apply 已经打开。
External integrations、webhooks 和 third-party callbacks 的分层见
[`integration-webhook-boundary.md`](integration-webhook-boundary.md)；本文中的 secret resolve 通过，也不代表
webhook signing secret 或 provider token 可以被任意 connector 复用。

当前事实：

- `000009_v1_boundary_foundation` 已预留 `users`、`teams`、`memberships`、`api_tokens`、
  `secret_refs`、`engine_profiles` 和 `webhooks`；v1 目标模型还保留 project-scoped membership
  语义，用于后续把 team role 映射到具体 project scope。
- v1.0 前这些表只表达长期边界，不代表多用户登录、token auth、secret 解析、webhook 调用或远程 worker
  已打开。
- `project_key` 仍是 workflow、run、artifact、worker、secret、audit 和 API 查询的隔离边界。
- 本机 single-user mode 继续使用稳定 actor，例如 `local-user` 和 `system`，避免后续审计补洞。

## Non-goals Before v1.x

v1.0 之前禁止把以下能力解释为已打开：

- 真实用户登录或 session 管理。
- Bearer token / API token 认证 enforcement。
- API token issuance / rotation / revocation 作为真实权限入口。
- team / role / membership 对业务 API 的强制授权。
- OS keychain、env、encrypted DB secret store 或外部 secret manager 的明文解析。
- secret 注入 Codex CLI、OpenAI API、local model、external agent 或 worker 环境。
- remote worker credential issuance、rotation、revocation 或 lease-scoped token。
- webhook delivery、signing secret 或 third-party callback。

v1.0 可以继续做的是 readiness、doctor、preview、fixture evidence 和 audit coverage gap 标注。

## Boundary Principles

1. **Principal 与 actor 分离**。
   - principal 是请求身份，例如 local service、user、api token、worker 或 system。
   - actor 是审计主体，必须写入 command、event 和 audit 记录。
   - single-user mode 也必须显式使用 `local-user` actor，不能省略来源。

2. **认证不替代 project scope**。
   - token 或 user 只说明“谁在请求”。
   - `project_key` / `project_id` 仍决定“请求属于哪个项目”。
   - global ID route 即使有 token，也必须继续执行 project visibility guard。

3. **membership 不替代 capability**。
   - team membership 只说明用户属于哪个团队。
   - project membership 只说明用户在某个 project 的角色上限。
   - 真实许可仍必须经过 project config、capability、resource、risk、gate、approval 和 audit。

4. **授权是 capability + resource + project + risk 的交集**。
   - capability 允许某类动作。
   - resource allowlist / denylist 限制路径、命令、secret、network、git 或 worker。
   - project scope 限制业务对象。
   - risk level 决定是否需要 gate、approval、rollback 和 audit。

5. **secret_ref 不是 secret**。
   - project config 只能引用 `secret_ref`。
   - `secret_refs` 表只保存 metadata、provider、scope 和 status。
   - 明文 secret 不进入 project config、artifact、event、audit 或长期 worker state。

6. **remote worker 不直连 PostgreSQL**。
   - remote worker 只能通过 API。
   - credential 必须 project-scoped、capability-scoped、lease-scoped、可撤销、可轮换，并有 heartbeat
     和 audit trail。

## Opening Ladder

### R4-0 Current Reserved Boundary

Status: current v1 boundary foundation.

Allowed:

- 创建长期 schema 表。
- 在 docs、readiness、doctor、schedule preview 和 engine readiness 中引用 auth / team / secret gap。
- 保持 single-user local service。

Forbidden:

- 强制 bearer auth。
- 解析 secret。
- 发放 remote worker token。
- 根据 membership 改变 API 可见性。
- 调用 secret-backed engine。

Proof:

- Migration succeeds.
- Existing tests still pass without auth configuration.
- `secret_ref_unavailable` / disabled engine profile 只作为 blocked reason。

### R4-1 Read-only Security Boundary Doctor

Status: future preview-only.

Allowed:

- 读取 `users`、`teams`、`memberships`、`api_tokens`、`secret_refs` 和 `engine_profiles` metadata。
- 返回 auth / team / token / secret readiness matrix。
- 标注 disabled、missing evidence、unscoped token、expired token、plaintext risk、missing audit coverage。

Forbidden:

- 创建、更新、撤销 token。
- 读取 token secret 或 secret 明文。
- 改变 API authorization result。
- 写 audit event，除非该 doctor 通过明确 Command API 记录 evidence。

Required response facts:

```text
mode = read_only_security_boundary_doctor
auth_enforcement_open=false
team_permission_enforcement_open=false
api_token_enforcement_open=false
secret_resolve_open=false
remote_worker_credentials_open=false
plaintext_secret_read=false
db_secret_plaintext_read=false
env_secret_read=false
keychain_secret_read=false
engine_secret_injected=false
authorization_changed=false
token_issued=false
token_rotated=false
token_revoked=false
```

### R4-2 Local Token Fixture

Status: future fixture-only.

Allowed:

- 在 temporary fixture service 中验证 token hashing、scope parsing、expiration 和 revocation semantics。
- Token value 只在测试进程内出现一次；数据库只保存 hash。
- 验证 token scope 与 `project_key` visibility guard 同时生效。

Forbidden:

- 默认要求 token。
- 在真实 AreaMatrix dogfood 中开启 token enforcement。
- 将 token 明文写入 logs、event、audit、artifact 或 config。

Required evidence:

- Positive and negative API tests.
- Token mismatch returns `401`.
- Scope mismatch returns `403` or `404` according to endpoint leak policy.
- Logs and artifacts do not contain token value.
- Existing no-auth local service mode remains available.

### R4-3 Optional Local Auth Enforcement

Status: v1.x opt-in.

Allowed:

- Local service 启动参数或配置显式开启 token enforcement。
- CLI / Web / Desktop 通过同一 auth header 访问 API。
- Audit 记录 authenticated actor / token key / scope hash，不记录 token 明文。

Forbidden:

- 未配置时破坏 existing local workflows。
- 绕过 Command API。
- 用 token scope 替代 project permission、gate 或 approval。

Required evidence:

- Backward compatibility smoke: enforcement off.
- Enforcement smoke: valid token pass, missing token deny, expired token deny, revoked token deny.
- Project mismatch cannot read global run/artifact IDs.
- Command API still enforces idempotency、permission、approval 和 audit。

### R4-4 Team Permission Enforcement

Status: v1.x after local auth.

Allowed:

- users / teams / memberships 影响 API authorization。
- Role policy 只作为 additive narrowing，不扩大 project config capability。
- Team admin 不能绕过 project forbidden paths、R3/R4 gate 或 approval。

Forbidden:

- role=admin 自动获得 secret resolve、project write、publish 或 restore apply。
- membership change 不写 audit。
- 跨 project 共享未授权 token / worker / secret scope。

Required evidence:

- Matrix tests for role x project x capability x resource.
- Membership change audit.
- Project route scope and global ID guard remain enforced.
- Downgrade / revoke takes effect without service restart, or clearly documents restart requirement.

### R4-5 Secret Store Preview

Status: v1.x preview before resolve.

Allowed:

- 选择 secret provider: OS keychain、env binding、encrypted DB store 或 external secret manager。
- 记录 secret metadata、provider、scope、status、rotation requirement 和 audit readiness。
- 测试 redaction policy against logs、artifacts、events、audit metadata。

Forbidden:

- 读取 secret 明文。
- 注入 worker / engine。
- 将 secret value 存入 PG metadata。

Required evidence:

- Redaction tests with canary values.
- Backup manifest does not include secret values.
- Artifact integrity and restore dry-run do not expose secrets.
- Secret preview cannot be used by engine execution.

### R4-6 Scoped Secret Resolve

Status: v1.x R4 apply.

Allowed only after explicit R4 approval:

- Resolve one secret for one command / run / lease scope.
- Provide short-lived binding to worker or engine adapter.
- Record secret ref, provider, actor, run, lease, expiration and redaction policy in audit.

Required binding scope:

```text
project_id
project_key
actor_id
command_request_id
run_id
run_task_id nullable
lease_id nullable
capability
secret_ref
provider
expires_at
redaction_policy_id
revocation_token_hash
```

Forbidden:

- Long-lived worker secret storage.
- Secret value in stdout、stderr、artifact、event、audit、project config or command request payload.
- Secret value in backup manifest、restore package、release evidence、plugin manifest or worker persistent state.
- Secret reuse across project or run scope.

Required evidence:

- Canary secret never appears in persisted outputs.
- Binding expires.
- Revocation prevents reuse.
- Failed resolve leaves no partial credential state.
- Rollback plan for provider outage and token compromise exists.

### R4-7 Remote Worker Credential

Status: v1.x after scoped secret resolve or no-secret remote worker pilot.

Allowed:

- Issue worker credential through API.
- Credential is project-scoped、capability-scoped、lease-scoped where possible.
- Heartbeat, lease acquire/release/recover and attempt submission all use API.
- Credential issue、rotation、revocation 和 heartbeat 都必须写 audit，并可通过 project scope 查询。

Forbidden:

- Direct PostgreSQL access.
- Unscoped global worker token.
- Worker credential that can create arbitrary project writes, secret resolves or publish actions.
- Worker credential that survives revoke / project removal / membership downgrade without a documented grace window.

Required evidence:

- Credential rotation and revocation tests.
- Lost worker recovery.
- Lease timeout recovery.
- Cross-project denial.
- Audit trail from credential issue to attempt closeout.

Suspension triggers:

```text
token plaintext persisted
token hash or scope missing
membership change missing audit
cross-project access allowed
secret canary persisted
secret binding reused after expiry
secret revoke ignored
remote worker credential misuse
remote worker credential revoke ignored
remote worker direct PostgreSQL access
audit missing for successful auth/team/secret/worker action
```

## Data Requirements Before Enforcement

Before any enforcement step leaves preview, prove these invariants:

```text
actors stable
users linked to actors where applicable
api_tokens store token_hash only
token_key is not a secret
token scope includes project constraints
secret_refs do not store plaintext
engine_profiles only reference secret_refs
audit_events record actor/principal/scope/result
events do not include plaintext token or secret
artifacts do not include plaintext token or secret
backups do not include plaintext token or secret
```

## API Requirements Before Enforcement

All API clients must keep using the same service layer:

- CLI, Web, Desktop and Worker send auth through a shared API client path.
- Web and Desktop do not store long-lived secrets in UI state.
- SSE uses the same project and auth scope as REST.
- Command API validates auth before permission preview, but permission preview still determines business allowance.
- Read-only Query API can be globally visible only where explicitly documented; project-scoped records require project scope.

## Audit Requirements

Any future auth / team / secret write must produce audit events for:

```text
api_token.create
api_token.revoke
api_token.rotate
auth.enforcement.enable
auth.enforcement.disable
membership.create
membership.update
membership.revoke
secret_ref.create
secret_ref.update
secret.resolve.preview
secret.resolve.apply
worker_credential.issue
worker_credential.revoke
```

Audit metadata must include stable identifiers and hashes, not secret or token plaintext.

## Rollback Requirements

Every enforcement step needs a rollback path:

- Auth enforcement can be disabled for local recovery without deleting users, teams or audit history.
- Token compromise can revoke token hashes without dropping audit rows.
- Membership mistakes can revert membership state with a new audit event.
- Secret provider outage can disable secret resolve while preserving readiness and no-secret execution.
- Remote worker credential compromise can revoke credentials and recover leases.

Rollback must be additive; do not rewrite events or audit history.

## AreaMatrix Dogfood Policy

AreaMatrix remains conservative:

- v1.0: no auth enforcement, no secret resolve, no remote worker credential.
- v1.x no-secret engine / worker pilots must start on fixture or non-AreaMatrix project first.
- AreaMatrix secret-backed engine execution requires explicit R4 approval, redaction evidence, rollback plan and
  proof that `workflow/versions/**/execution/**`, `progress.json`, user files and release evidence are not modified
  outside approved scope.

## Go / No-go For Implementation

Implementation can start only after a separate explicit approval names the target rung, for example `R4-1
Security Boundary Doctor` or `R4-2 Local Token Fixture`.

Before approval, provide:

- Impacted API and CLI surfaces.
- Database rows touched.
- Files and artifact paths touched.
- Secrets or tokens that might exist in memory.
- Validation commands.
- Rollback procedure.
- AreaMatrix non-touch guarantee or explicit cross-repo authorization.

Without that packet, auth / team / secret work stays documentation and readiness only.
