# Governance Boundary Audit

## Purpose

本文对应 backlog 任务
[`AF-P0-003 Governance Boundary Audit`](../plans/task-backlog.md#af-p0-003-governance-boundary-audit)。

目标是把 AreaFlow 的权限、安全、API、audit 和高风险能力边界整理成可验收审计。本文只记录边界，不创建
active task、不授权执行、不写 AreaMatrix。

## Source Documents

- [`../architecture/security-permissions.md`](../contracts/security-permissions.md)
- [`../architecture/api-surface.md`](../contracts/api-surface.md)
- [`../architecture/auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)
- [`../adr/0005-phase-0-foundation-baseline.md`](../../../adr/0005-phase-0-foundation-baseline.md)
- [`../adr/0006-platform-operating-boundary.md`](../../../adr/0006-platform-operating-boundary.md)
- [`../../governance/README.md`](../../../../governance/README.md)
- [`../product/phase-backlog.md`](../plans/phase-backlog.md)

## Governance Invariants

| Invariant | Status | Evidence |
|---|---|---|
| AreaFlow 默认只读管理项目 | established | `security-permissions.md` 默认策略；phase backlog 总原则。 |
| Query API 无副作用 | established | `api-surface.md` Query API；ADR 0005/0006。 |
| Command API 是唯一业务写入口 | established | `api-surface.md` Command API；ADR 0006。 |
| SSE 不是状态源 | established | ADR 0005；API surface 多端边界。 |
| PostgreSQL 是主状态源 | established | ADR 0002；platform blueprint。 |
| Projection 不是主状态 | established | ADR 0005/0006；project config；phase backlog。 |
| Deny 优先于 allow | established | `security-permissions.md` 路径策略。 |
| R2-R4 必须有 preview/gate/approval/rollback/audit | established | `security-permissions.md` 风险等级；ADR 0005/0006。 |
| Worker 只领取 `run_task` | established | execution model；ADR 0005/0006。 |
| Secret v1.0 前只做 readiness | established | `security-permissions.md`；`auth-team-secret-boundary.md`；phase backlog。 |
| Release v1.0 只做 preview/gate/evidence | established | v1.0 milestone；API surface。 |
| Plugin 不能绕过 core 权限 | established | adapter/profile boundary；phase backlog。 |

## Capability Matrix

| Capability | Scope | Earliest Enabled Use | Must Not Mean |
|---|---|---|---|
| `read_project` | 读取被管理项目 metadata、hash、status、config | v0.1 import / doctor | 不代表可执行命令。 |
| `write_status` | 写轻量 projection，例如 `.areaflow/status.json` | v0.1 guarded projection | 不代表可写 workflow 或 execution。 |
| `write_artifacts` | 写 AreaFlow-owned artifact store 或 artifact metadata evidence | v0.5 runner preview / v0.6 worker evidence | 不代表可写被管理项目文件。 |
| `write_workflow` | 写被管理项目 allowlist 内 workflow/export 文件 | v0.4+ only with explicit config/gate | 不代表可写 source code 或 execution。 |
| `write_generated` | 只写被管理项目 allowlist 内 generated/projection 前缀 | v0.6+ only after generated-only gate | 不代表可写 source code、execution 或 progress JSON。 |
| `write_code` | 修改被管理项目代码 | v0.6+ only for approved execution | 不代表可绕过 verify 或 checkpoint。 |
| `run_commands` | 执行 allowlist 命令 | v0.2 native doctor optional / v0.6 execution | 不代表可执行 `./task-loop run` 或 forbidden command。 |
| `manage_workers` | 注册、heartbeat、lease、recovery | v0.6 worker beta | 不代表允许真实 agent execution。 |
| `manage_git` | git checkpoint/tag/push 等 | post-approval only | 不代表可运行 destructive git command。 |
| `network` | 网络访问 | future engine/integration only | 不代表可访问任意外部目标。 |
| `use_secrets` | 使用 secret reference 解析结果 | v1.x R4 design | v1.0 前不解析明文。 |
| `execute_agents` | 调用 Codex/OpenAI/local/external agent | v0.6+ approved execution / v1.x secret-backed engines | 不代表可绕过 budget、secret、audit、redaction。 |

## Risk Matrix

| Risk | Meaning | Required Controls | Current 0-100% Boundary |
|---|---|---|---|
| R0 `read_only` | 查询、metadata import、hash、preview、doctor | project scope、no write proof | 默认可用于 v0.1-v1.0 查询和 preview。 |
| R1 `projection` | 写 `.areaflow/status.json` 等轻量 projection | `write_status`、path allowlist、audit、projection gate | v0.1 可受控打开；`workflow/README.md` 自动写入仍推迟。 |
| R2 `managed_write` | 写 allowlist 内 workflow/export 文件 | command request、permission preflight、gate、approval、rollback、audit | v0.4 authoring/shim 之后按显式授权打开。 |
| R3 `execution` | 执行任务、修改代码、运行 worker、生成 execution evidence | approved task、run/task/attempt、worker lease、verify、checkpoint、audit | v0.6 beta 只执行 approved task；未批准执行仍 blocked。 |
| R4 `migration_security` | DB migration、secret 解析、权限变更、远程 worker、release exception apply | explicit R4 approval、affected resources、rollback plan、audit evidence | v1.0 默认不打开真实 apply；v1.x 单独设计。 |

## Query API Audit

Query API 必须保持只读。以下行为在 Query API 中禁止：

- 写数据库业务状态。
- 写被管理项目文件。
- 写 artifact store。
- 创建 approval。
- 创建 release exception。
- 创建 migration。
- 执行命令。
- 读取或解析 secret 明文。
- 领取 worker lease。
- 调度 worker。
- 推进 workflow/run 状态。

当前源事实覆盖：

- `GET /api/v1/projects/{project_key}/status-projections` 只读 projection metadata，不写 `.areaflow/status.json`。
- `GET /api/v1/backup/manifest` 不读 artifact 原文、不生成压缩包、不执行 restore。
- `GET /api/v1/backup/restore-plan` 不写数据库、不覆盖 project files、不执行 restore apply。
- `GET /api/v1/release/*` preview/gate 链不创建 package、不发布、不写 exception record。
- `GET /api/v1/audit/coverage` 不创建 audit event。
- `GET /api/v1/permissions/doctor` 不修改 project config、不执行命令、不读取 secret。
- `GET /api/v1/worker-pool/schedule-preview` 不领取 lease、不执行 run-once。

审计结论：Query API 无副作用原则已经在主要架构文档中建立，后续新增 GET 接口必须继续显式列出 forbidden actions。

## Command API Audit

Command API 是唯一业务写入口。所有改变状态的动作必须具备：

```text
actor
project_scope
command_type
reason
idempotency_key
request_hash
expected_version nullable
risk_level
risk_policy
permission_preview
approval_state
status
audit_event_id nullable
metadata
```

R2-R4 command 还必须返回：

```text
permission preflight
risk preview
affected resources
approval/gate result
rollback wording
audit outcome
```

当前已进入 Command API 幂等边界的动作：

```text
project.import
project.cutover.apply
workflow.version.create
workflow.approval.record
runner.preview
run.start
run.drain
run.cancel
project.status_projection.write
project.status_projection.apply
project.doctor.record
lease.acquire
lease.release
lease.recover
artifact.archive.preview
```

仍必须通过 command request 表达、但当前不应默认打开真实 apply 的动作：

```text
approval decision
artifact archive apply
artifact delete / GC
execution apply
release exception record
restore apply
publish apply
permission change
secret resolve
remote worker credential issuance
```

审计结论：Command API 原则已经成立，但真实 archive/GC、execution apply、restore apply、publish apply 和
secret resolve 仍必须保持 blocked / preview-only，直到对应 R3/R4 gate 完成。

## Audit Event Coverage

所有写入、权限判断、命令执行和密钥引用都必须写入 `audit_events`。当前 v1.0 审计覆盖矩阵应至少区分：

| Requirement | v1.0 Expected Status |
|---|---|
| project registration / config upsert | covered when command writes state |
| status projection write | covered |
| workflow authoring | covered |
| approval decision | covered when approval record command writes state |
| runner preview | covered |
| worker registration | covered |
| worker capability denial | covered |
| worker lease lifecycle | covered |
| real command execution | gap until execution opens |
| secret resolution | gap until v1.x secret manager |
| permission change | gap until permission admin opens |
| release exception decision | preview/gap until approved write path |

审计 gap 不能被伪装成 pass。未启用的长期能力必须在 release readiness / audit coverage 中显示为 gap、
needs_attention 或 blocked。

## High-risk Capability Boundaries

| Capability Area | v1.0 Allowed | v1.0 Forbidden | Future Gate |
|---|---|---|---|
| Restore | manifest、artifact integrity、restore dry-run | restore apply、delete/overwrite existing state | R4 restore apply design。 |
| Secret | `secret_ref` readiness、blocked reason | env/keychain/DB secret 明文解析、engine secret injection | `auth-team-secret-boundary.md` R4 ladder。 |
| Engine | readiness、schedule preview、manual/approved execution boundary | unapproved Codex/OpenAI calls, secret-backed engine execution | R3/R4 engine execution gate。 |
| Remote worker | schema/readiness concept | remote credential issuance、direct DB access、unscoped lease | `auth-team-secret-boundary.md` R4 ladder。 |
| Plugin | adapter/profile conformance、seed marketplace metadata | unknown third-party code execution | plugin registry + permission sandbox design。 |
| Release | readiness、evidence、package/distribution/publish/rollout preview | tag、push、sign、upload、publish | R4 publish apply design。 |
| Release exception | doctor、record/schema/apply preview | creating migration, writing exception record, marking accepted | migration approval gate + R4 approval。 |
| AreaMatrix shim | preview/readiness/compatibility | writing shim files without explicit authorization | R2 managed write gate。 |

## Web / Desktop / Worker Boundary

- Web v0.7 starts read-only: GET/SSE dashboard, approval records, runs, artifacts, workers, audit trail.
- Web write operations require Command API, risk preview, permission preflight, approval/gate and audit outcome.
- Desktop v0.9 is a local service shell and dashboard launcher, not a second state store.
- Desktop must not directly execute workflow or parse secret.
- Worker cannot expand its own capabilities.
- Worker cannot claim `workflow_item` directly; it only receives scoped `run_task` lease.
- Worker output must not leak secret into stdout, stderr, artifact, event or audit metadata.

## Findings

1. Capability 与 R0-R4 风险等级已经一致，且 `write_artifacts`、`write_generated`、`manage_workers` 的范围已被限制。
2. Query API 无副作用原则已覆盖主要 read API，新增 GET 接口必须继续声明 forbidden actions。
3. Command API 幂等、risk、permission、approval 和 audit 字段已经定义，且多项现有命令已进入该边界。
4. R2-R4 操作的 preview/gate/approval/rollback/audit 要求已在源事实中建立。
5. v1.0 默认不会打开真实 restore apply、secret resolve、API token enforcement、team permission enforcement、remote worker credential、plugin execution 或 publish apply；auth/team/secret/remote worker 必须按 `auth-team-secret-boundary.md` 逐级开闸。
6. 当前 `governance/` 目录只有 README；后续打开 R2-R4 能力前，应补 `governance/security`、`governance/permissions`、`governance/workflow`、`governance/adapters` 下的具体规则文档。
7. `audit_events` 是治理事实的核心；release readiness 不能把 future-only audit gap 伪装成 pass。

## Gate Evidence

本审计关闭条件：

- 本文记录 capability、risk、Query API、Command API、audit 和 high-risk capability 边界。
- backlog `AF-P0-003` 指向本审计产物。
- implementation gap audit 指向本审计。
- `git diff --check -- docs/development/governance-boundary-audit.md docs/development/implementation-gap-audit.md tasks/backlog/0-100-platform-backlog.md`
- `go test ./...`
