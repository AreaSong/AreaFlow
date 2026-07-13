# Task Backlog Status Audit

## Purpose

本文按 [`../../tasks/backlog/0-100-platform-backlog.md`](../plans/task-backlog.md)
的推荐推进顺序，把 AreaFlow 0-100% backlog 拆成 task-level 状态审计。它补充
[`implementation-gap-audit.md`](./implementation-gap-audit.md)，用于回答：

```text
每个 backlog task 当前是 implemented、preview_only、planned 还是 deferred
对应证据在哪里
下一步为什么还不能算 100% 完成
```

状态含义：

```text
implemented:
  任务已有代码/API/CLI/测试或 smoke 证据，且行为范围与当前 milestone 边界一致。

implemented_scoped:
  任务已有受限实现和证据，但范围被刻意收窄，例如 fixture/temp project、AreaFlow-owned artifact
  或 read-only gate；不能解释为真实 AreaMatrix apply 或 v1.x 高风险能力已打开。

preview_only:
  只读 preview、readiness、gate 或 dry-run 已实现；真实 apply / execution / publish / restore 尚未打开。

planned:
  仍主要是文档、schema 或 milestone 计划，缺少可运行实现证据。

deferred:
  明确保留到 post-100% v1.x；不能作为 v1.0 必交付打开。
```

完成解释规则：

- `implemented` 表示对应阶段能力已经在当前边界内可用，不代表下游高风险能力自动打开。
- `implemented_scoped` 表示受限实现有效，例如 fixture/temp project、AreaFlow-owned artifact 或
  read-only evidence；不能升级解释为真实 AreaMatrix 写入、execution cutover 或 v1.x apply 已完成。
- `preview_only` 表示 go/no-go 可查询；只有后续受保护 Command API apply 通过 permission、approval、
  rollback 和 audit 后，才算真实动作打开。
- `deferred` 不计入 v1.0 缺口；它是刻意放到 post-100% v1.x 的高风险开闸清单。

## E2 Current Binding Summary

Completion audit 的 E2 proof 绑定以下当前源文件：

- `tasks/backlog/0-100-platform-backlog.md`
- `docs/development/task-backlog-status-audit.md`

E2 `complete` proof 必须由外部 review 提供上述文件的 sha256、source-set hash，以及三个 zero-count：

- `planned_v1_required_task_count=0`
- `missing_evidence_v1_required_task_count=0`
- `blocked_v1_required_task_count=0`

这些 count 只统计 v0-v1.0 required closure，不把 `deferred` v1.x opening ladder 当作 v1.0 缺口。本文不写死自身
hash；record command 接收外部 review 给出的 hash，completion audit 再只读重算当前文件绑定并阻断漂移。

## v0-v1.0 Task Matrix

v0.6 task rows are governed by
[`../architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md): scoped worker /
execution evidence cannot be accumulated into real AreaMatrix execution cutover.

| 顺序 | Task | 状态 | 当前证据 | 仍需补齐 |
|---:|---|---|---|---|
| 1 | AF-P0-001 Design Source Alignment | implemented | `docs/product/master-plan.md`、`platform-blueprint.md`、`phase-backlog.md`、`roadmap.md`、ADR 0001-0006；已覆盖 `project_key` 隔离、workspace/environment 后置、release exception 白名单和 AreaFlow self dogfood 节奏。 | 持续保持路线与实现同步；self dogfood、workspace/environment 一等实体、release exception real write 仍不能越过各自后续 gate。 |
| 2 | AF-P0-002 Directory Boundary Audit | implemented | [`directory-boundary-audit.md`](./directory-boundary-audit.md) | 后续拆分 `runner`、`worker`、`permission`、`engine`、`secret`、`integration` 时更新。 |
| 3 | AF-P0-003 Governance Boundary Audit | implemented | [`governance-boundary-audit.md`](./governance-boundary-audit.md) | 高风险能力打开前继续补 gate / approval / rollback / audit。 |
| 3.5 | AF-P0-004 Operations Deployment Observability Boundary | implemented | [`../architecture/operations-deployment-observability-boundary.md`](../../../architecture/operations-deployment-observability-boundary.md)、`docs/product/master-plan.md`、`docs/product/platform-blueprint.md`、`docs/milestones/v0.9-desktop-shell.md`、`docs/milestones/v1.0-stable-platform.md` | 这是文档边界，不代表 full support bundle export、remote ops control、managed upgrade 或 destructive rollback 已打开；后续实现需要单独 task 和 smoke evidence。 |
| 4 | AF-V01-001 PostgreSQL Bootstrap Smoke | implemented | [`bootstrap-smoke-evidence.md`](./bootstrap-smoke-evidence.md)、[`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)、[`../architecture/data-model-v0.1.md`](../contracts/data-model-v0.1.md) | 继续用真实 PG smoke 复验迁移和 bootstrap；v0.1 最小闭环不包含 worker、engine、secret 或 cutover；后续 support tables 存在不代表能力打开。 |
| 5 | AF-V01-002 AreaMatrix Adapter Metadata Import | implemented | [`areamatrix-adapter-import-evidence.md`](./areamatrix-adapter-import-evidence.md)、[`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)、[`../architecture/areamatrix-import-scope-contract.md`](../../../architecture/areamatrix-import-scope-contract.md)、[`../architecture/project-config.md`](../contracts/project-config.md) | 不复制历史 artifact 原文；import `run` 不能解释为 execution run；`areaflow.yaml` scheduling/engine/allowed commands 只作为 metadata；当前 AreaMatrix artifact rows 是 metadata-only `external_project`；真实 archive/copy 另走后续命令。 |
| 6 | AF-V01-003 Guarded Status Projection | implemented_scoped | [`status-projection-evidence.md`](./status-projection-evidence.md)、[`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)、[`../architecture/data-model-v0.1.md`](../contracts/data-model-v0.1.md)、`schemas/status-projection.schema.json`；2026-07-04 fixture smoke 已按 schema 校验 `.areaflow/status.json`；2026-07-05 `smoke-fixture` 默认不读取真实 AreaMatrix，真实 projection 指纹守护需显式 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` | 真实 AreaMatrix `.areaflow/status.json` 写入仍需单独授权；`workflow/README.md` 写入关闭；projection 不能保存 logs、checkpoint、secret 或 artifact 原文；`export-status` 是 compatibility alias。 |
| 7 | AF-V02-001 Doctor And Readiness Bundle | implemented | [`shadow-doctor-readiness-evidence.md`](./shadow-doctor-readiness-evidence.md)、[`../architecture/v0.2-shadow-doctor-contract.md`](../contracts/v0.2-shadow-doctor-contract.md) | native doctor 继续受授权和 command allowlist 约束；verification bundle phase gate 不代表 cutover readiness。 |
| 8 | AF-V02-002 Native Doctor Authorization Boundary | implemented | [`native-doctor-authorization-evidence.md`](./native-doctor-authorization-evidence.md) | 不把未授权 native doctor skipped/warn 伪装成 pass。 |
| 9 | AF-V03-001 Workflow Version Authoring Model | implemented | [`workflow-version-authoring-evidence.md`](./workflow-version-authoring-evidence.md)、[`../architecture/v0.3-version-authoring-contract.md`](../contracts/v0.3-version-authoring-contract.md) | 只接管 AreaFlow-authored records；不写被管理项目 workflow 目录；placeholder artifact 不让 gate 自动通过。 |
| 10 | AF-V03-002 Gate Transition Approval Records | implemented | [`gate-transition-approval-evidence.md`](./gate-transition-approval-evidence.md)、[`../architecture/v0.3-version-authoring-contract.md`](../contracts/v0.3-version-authoring-contract.md) | promotion preview 仍不等于 approval 或 execution；approved approval record 需要 ready transition preview 且保持 `approval_is_execution=false`。 |
| 11 | AF-V04-001 Compatibility And Shim Readiness | preview_only | [`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)、[`areamatrix-compatibility-shim-plan.md`](../migrations/areamatrix-compatibility-shim-plan.md)、[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../contracts/v0.4-workflow-ownership-cutover-contract.md)；2026-07-04 真实 AreaMatrix 只读 smoke 和 compatibility fixture smoke 已验证 `shim-readiness` 暴露 `stable_fallback_projection_v1` schema contract，新增 `real_areamatrix_status_projection_schema` evidence gate，且 `shim-authorization` JSON 与普通 CLI 文本都包含 status-projections/schema verification preflight、编辑前后保护路径专项 git status、post-edit verification、rollback scope 和安全事实；focused tests、compatibility Docker smoke 和 v1 stable fixture 长链已覆盖只读 `shim-apply-packet` / `shim-apply-gate` API/CLI/project builders，生成 authorization snapshot hash、allowed files、status projection packet/gate proof ids、protected path fingerprint、rollback plan、explicit approval、idempotency key 和 audit correlation id，并在 readiness evidence 缺失时 fail closed；fixture smoke 证明 evidence 记录后完整 packet 可变为 `ready_for_future_apply_command` 但仍不写文件；2026-07-05 compatibility fixture 默认不读取真实 AreaMatrix 指纹，真实 AreaMatrix smoke 仍是单独 read-only 入口；2026-07-06 `make smoke-docker-shim-authorization-preflight` 作为 AF-V04 命名授权前预检通过，复验真实 AreaMatrix status/shim authorization、apply packet/gate 和 no-write safety facts；2026-07-11 `shim-apply` CLI/API 已补为受保护 AreaFlow-only command 入口，fixture packet/gate 通过时记录 `command_requests` / `events` / `audit_events`，真实只读 smoke 的缺 packet 分支仍 blocked，二者都断言 `project_write_attempted=false`、`execution_write_attempted=false`、`status_projection_written=false`、`area_matrix_files_modified=false` | AreaFlow 侧 compatibility / shim preview / shim readiness / shim-readiness-evidence、只读 `shim-authorization`、只读 `shim-apply-packet/gate` API/CLI 和 AreaFlow-only `shim-apply` API/CLI 已有；这些 packet/gate/command 不是编辑授权；当前真实 AreaMatrix `.areaflow/status.json` 已由 Package A 更新为 stable projection，但 AreaMatrix 仓库内 shim 文件落地仍需要显式编辑授权。 |
| 12 | AF-V04-002 Authoring Cutover Apply | implemented_scoped | [`authoring-cutover-apply-evidence.md`](./authoring-cutover-apply-evidence.md)、[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../contracts/v0.4-workflow-ownership-cutover-contract.md) | 只切 AreaFlow DB 内 authoring ownership；只允许 `mode=authoring_cutover`；不切 execution，不写 AreaMatrix。 |
| 13 | AF-V05-001 Runner Preview Evidence | preview_only | [`runner-preview-evidence.md`](./runner-preview-evidence.md)、[`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md)；2026-07-05 PG smoke assertion 已要求 `runner.preview` persisted command response 包含 `area_matrix_write_attempted=false` | 真实 runner execution、engine 调用和项目写入关闭；runner preview 仍只是 dry-run proof。 |
| 14 | AF-V05-002 Run Control Dry-run Boundary | preview_only | [`run-control-evidence.md`](./run-control-evidence.md)、[`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md)；2026-07-05 PG smoke assertion 已要求 `run.start` / `run.drain` / `run.cancel` persisted command response 包含 `area_matrix_write_attempted=false` | 完整 worker 协作式 cancel/drain 仍属 execution beta 后续；run control 只改 dry-run DB 状态。 |
| 15 | AF-V06-001 Worker Registry Lease Lifecycle | implemented_scoped | [`worker-lease-evidence.md`](./worker-lease-evidence.md) | lifecycle / lease 已有；不代表真实 task execution 或远程 worker 凭证管理打开。 |
| 16 | AF-V06-002 Codex CLI Adapter Preview | preview_only | [`codex-cli-adapter-preview-evidence.md`](./codex-cli-adapter-preview-evidence.md) | 不运行 Codex CLI、不解析 secret、不执行 shell。 |
| 17 | AF-V06-003 Execution Approval Gate | preview_only | [`execution-approval-gate-evidence.md`](./execution-approval-gate-evidence.md) | gate pass 只代表可进入受保护 apply 设计，不代表已执行。 |
| 18 | AF-V06-004 Fixture Execution Apply | implemented_scoped | [`fixture-execution-apply-evidence.md`](./fixture-execution-apply-evidence.md) | 只限 fixture execution；不代表真实 AreaMatrix execution cutover。 |
| 19 | AF-V06-005 Read-only Verify | implemented_scoped | [`read-only-verify-evidence.md`](./read-only-verify-evidence.md) | 只读取 allowlisted target hash/size；不保存原文、不写项目。 |
| 20 | AF-V06-006 Approved Artifact Write | implemented_scoped | [`approved-artifact-write-evidence.md`](./approved-artifact-write-evidence.md) | 只写 AreaFlow-owned artifact store；不写被管理项目。 |
| 21 | AF-V06-007 Execution Plan Preview | preview_only | [`execution-plan-preview-evidence.md`](./execution-plan-preview-evidence.md) | copy、checkpoint、repair 仍 blocked / waiting。 |
| 22 | AF-V06-008 Approved Project Write Design Gate | preview_only | [`project-write-design-gate-evidence.md`](./project-write-design-gate-evidence.md) | 设计 gate 不读取或写入被管理项目。 |
| 23 | AF-V06-009 Fixture-only Approved Project Write | implemented_scoped | [`fixture-project-write-evidence.md`](./fixture-project-write-evidence.md) | 只限 fixture project write/verify/rollback drill；真实 AreaMatrix 写入关闭。 |
| 24 | AF-V06-010 Managed Generated Write Gate | preview_only | [`managed-generated-write-gate-evidence.md`](./managed-generated-write-gate-evidence.md) | generated-only apply 前只读门禁；不创建 lease/attempt/artifact。 |
| 25 | AF-V06-011 Managed Generated Write Apply Core + API CLI Surfacing | implemented_scoped | [`managed-generated-write-apply-evidence.md`](./managed-generated-write-apply-evidence.md) | 只允许 fixture/temp project rollback drill；不保留真实 AreaMatrix generated apply。 |
| 26 | AF-V06-012 Generated Write Dogfood Readiness | preview_only | [`generated-write-readiness-evidence.md`](./generated-write-readiness-evidence.md) | `apply_open=false`，真实 AreaMatrix generated-only apply 关闭。 |
| 27 | AF-V06-013 Generated Write Apply Beta Approval Gate | preview_only | [`generated-write-apply-beta-gate-evidence.md`](./generated-write-apply-beta-gate-evidence.md) | `approval_status=needs_approval` 不能被解释为 apply 许可。 |
| 28 | AF-V07-001 Read-only Dashboard Coverage | implemented_scoped | `web/`、`scripts/smoke-web.sh`、`scripts/smoke-web-check.mjs`、[`../architecture/v0.7-web-dashboard-contract.md`](../contracts/v0.7-web-dashboard-contract.md)、[`implementation-gap-audit.md`](./implementation-gap-audit.md)、[`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)、[`execution-cutover-readiness-evidence.md`](./execution-cutover-readiness-evidence.md)、[`execution-forwarding-v1-readiness-evidence.md`](./execution-forwarding-v1-readiness-evidence.md)、[`operations-readiness-evidence.md`](./operations-readiness-evidence.md)、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)；run detail 已使用 `?project_key=` visibility guard，browser smoke 已点击 run timeline 并验证 query scope；Web 已接入 operations readiness panel，smoke checker 已扩展为期待 `/api/v1/ops/readiness`；Web 已接入 shim apply packet/gate、Execution Forwarding v1 readiness / apply-preview / apply-packet Packet Gate / rollback-preview 只读 panel。 | 继续证明 Web 只走 `/api/v1` GET / SSE；shim authorization、shim apply packet/gate、execution cutover readiness、Execution Forwarding v1 readiness/apply-preview/apply-packet Packet Gate/rollback-preview、operations readiness 与 release final/evidence/package/distribution/publish/approval/rollout preview 只读展示已接入；写操作、shim apply、status projection apply、AreaMatrix file edit、execution cutover apply、execution-forwarding apply/rollback、task-loop forwarding、support export、migration apply、telemetry upload、release package、approval、rollout 和 publish/apply 关闭。 |
| 29 | AF-V07-002 Web Write Action Gate | preview_only | [`web-write-action-gate-evidence.md`](./web-write-action-gate-evidence.md)、`docs/architecture/api-surface.md` | Web approval console 写操作尚未打开；gate 只展示 disabled/read-only 写动作要求。 |
| 30 | AF-V08-001 Schedule Preview | preview_only | `internal/project/worker.go`、`internal/api/server_test.go`、`docs/milestones/v0.8-multi-project-worker.md`、[`../architecture/v0.8-multi-project-worker-pool-contract.md`](../contracts/v0.8-multi-project-worker-pool-contract.md)、[`implementation-gap-audit.md`](./implementation-gap-audit.md)、[`multi-project-isolation-evidence.md`](./multi-project-isolation-evidence.md)、[`../architecture/auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)；v0.8 合同明确 `recommended`、`available_slots` 和 `next_action` 不代表 scheduler apply、lease claim、worker dispatch、secret resolve、remote worker credential、team/auth enforcement 或 execution cutover；`TestStoreProjectKeyIsolationWithPostgres` 已提供并通过 PostgreSQL 隔离 fixture，覆盖同名 workflow version、run、artifact、event、audit、worker 和 lease recovery 的 project scope；`TestProjectScopedAPIsUseRouteProjectKey` 已覆盖 versioned API route 对 summary/events/audit/workers/workflow versions/runs/artifacts 的 project scope；`TestGlobalRunEndpointsHonorProjectKeyVisibility` 与 `TestGlobalArtifactEndpointsHonorProjectKeyVisibility` 已覆盖全局 run/artifact ID route 的兼容型 `project_key` visibility guard；`scripts/smoke-project-isolation.sh` 和 `make smoke-docker-project-isolation` 已接入 smoke 并有 pass evidence；auth/team/API token/secret/remote worker R4 ladder 已成文。 | 真实多项目调度、远程 worker 关闭；API token enforcement、team permission enforcement、secret resolve 和 remote worker credential 仍未打开。 |
| 31 | AF-V08-002 Engine Secret Readiness Boundary | preview_only | `internal/project/engine_preview.go`、`docs/milestones/v0.8-multi-project-worker.md`、[`implementation-gap-audit.md`](./implementation-gap-audit.md)、[`../architecture/auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md) | secret 只做 readiness；真实 resolve 是 v1.x R4，必须先通过 scoped binding、redaction、audit 和 rollback 证据。 |
| 32 | AF-V09-001 Local Service Status Contract | implemented_scoped | `internal/project/service_status.go`、`docs/milestones/v0.9-desktop-shell.md`、[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)、[`implementation-gap-audit.md`](./implementation-gap-audit.md) | 只读 service status 已有；完整 Tauri shell 未实现；service status 不代表 process control 已打开。 |
| 33 | AF-V09-002 Tauri Shell Scaffold | implemented_scoped | [`desktop-shell-scaffold-evidence.md`](./desktop-shell-scaffold-evidence.md)、[`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)、[`execution-cutover-readiness-evidence.md`](./execution-cutover-readiness-evidence.md)、[`execution-forwarding-v1-readiness-evidence.md`](./execution-forwarding-v1-readiness-evidence.md)、[`operations-readiness-evidence.md`](./operations-readiness-evidence.md)、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)、[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)、`desktop/` | Tauri shell scaffold 已有并展示 selected project shim authorization blocked gate、shim apply packet/gate read-only review、execution cutover readiness blocked gate、Execution Forwarding v1 readiness/apply-preview/apply-packet Packet Gate/rollback-preview blocked/read-only gates、operations readiness 与 global release rollout preview chain；local process manager、native tray/menu、OS notification、package icon/signing 尚未实现。 |
| 34 | AF-V09-003 Desktop Service Control Gate | implemented_scoped | [`desktop-service-control-gate-evidence.md`](./desktop-service-control-gate-evidence.md)、[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)、`GET /api/v1/desktop/service-control-gate`、`areaflow desktop service-control-gate [--json]`、`desktop/`、`smoke-local: desktop service-control-gate` 和 2026-07-04 `smoke-docker-v1-stable-fixture` 长链 smoke | 只读 service control gate API/CLI/Desktop surface 已有；start/stop/restart、notification、tray/menu 仍 disabled，真实 process control 未打开。 |
| 35 | AF-V09-004 Desktop Notification Gate | implemented_scoped | [`desktop-notification-gate-evidence.md`](./desktop-notification-gate-evidence.md)、[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)、`GET /api/v1/desktop/notification-gate`、`areaflow desktop notification-gate [--json]`、`desktop/`、`smoke-local: desktop notification-gate` 和 2026-07-04 `smoke-docker-v1-stable-fixture` 长链 smoke | 只读 notification gate API/CLI/Desktop surface 已有；OS notification、approval/run/worker 通知仍 disabled，真实 SSE notification bridge 和 native notification bridge 未打开。 |
| 36 | AF-V09-005 Desktop Tray Menu Gate | implemented_scoped | [`desktop-tray-menu-gate-evidence.md`](./desktop-tray-menu-gate-evidence.md)、[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)、`GET /api/v1/desktop/tray-menu-gate`、`areaflow desktop tray-menu-gate [--json]`、`desktop/`、`smoke-local: desktop tray-menu-gate` 和 2026-07-04 `smoke-docker-v1-stable-fixture` 长链 smoke | 只读 tray/menu gate API/CLI/Desktop surface 已有；native tray/menu、service control、notification、settings 仍 disabled。 |
| 37 | AF-V10-001 Backup Restore Integrity Chain | preview_only | `internal/project/backup.go`、`artifact_integrity.go`、`restore_plan.go`、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、`docs/milestones/v1.0-stable-platform.md`、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md) | 只读 manifest / integrity / dry-run；真实 restore apply 是 v1.x R4；metadata-only history 和 skipped object verifier 不能被当成完整可恢复。 |
| 38 | AF-V10-002 Release Final Gate Chain | preview_only | `internal/project/release_*.go`、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、`docs/milestones/v1.0-stable-platform.md`、[`implementation-gap-audit.md`](./implementation-gap-audit.md)、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)、Web/Desktop release final/evidence/package/distribution/publish/approval/rollout preview panels | 只读 release rollout preview chain 和多端观察面已有；release final gate 不是 100% 充分条件；不创建 release package、不 tag/push/sign/upload/publish，不创建 release approval 或 rollout。 |
| 39 | AF-V10-003 Adapter Profile Conformance | preview_only | `internal/project/conformance.go`、`internal/workflow/profile.go`、`workflow/profiles/areamatrix/profile.yaml`、`examples/areamatrix/areaflow.yaml`、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)；`profile_item_state_contract` 已校验 AreaMatrix item state 枚举；`profile_transition_contract` 已校验 AreaMatrix `intake -> ... -> closeout` transition 顺序和 required gate；`profile_hard_rule_contract` 已校验防止 apply、execution、cutover 和 closeout 被误开的 hard rules；`profile_permission_policy_contract` 已校验默认 readonly 以及写入必须经过 capability、path allowlist、gate result、approval record 和 audit event；`profile_artifact_policy_contract` 已校验 PG metadata、artifact store content、managed project source docs 和 AreaFlow generated output 的 ownership/storage policy；`profile_cutover_policy_contract` 已校验 authoring cutover 与 execution cutover 分层；`project_config_policy` 已校验 active `areaflow.yaml` snapshot 的 protocol v1、migration strategy、safe capabilities、`.areaflow/status.json` write path、execution/DB/.areamatrix forbidden paths、dangerous command denylist、single-task scheduling、disabled engine profiles 和 disabled workflow README human summary；2026-07-05 focused tests 已覆盖 `plugin_seed_catalog_contract`、`plugin_manifest_draft_contract` 和 `plugin_no_execution_boundary`，证明 v1.0 plugin / marketplace 只允许 built-in / seed metadata、manifest draft lint 和未知 plugin execution v1.x 延后。 | conformance 只读；读取 workflow profile、project config snapshot 和内置 plugin seed baseline 但不写 DB、不写项目、不执行命令；plugin install/enable/execute/network/secret/project write 仍是 v1.x。 |
| 40 | AF-V10-004 AreaMatrix Execution Cutover Readiness | preview_only | [`execution-cutover-readiness-evidence.md`](./execution-cutover-readiness-evidence.md)、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、[`areamatrix-execution-cutover-boundary.md`](../migrations/areamatrix-execution-cutover-boundary.md)、`GET /api/v1/projects/{project}/execution-cutover-readiness`、`areaflow project execution-cutover-readiness`、Web `Execution Cutover` panel、Desktop `Execution Cutover` panel | 只读 readiness bundle、多端观察面和 cutover 命令边界已有；真实 AreaMatrix execution cutover、Execution Forwarding v1、Archive 和 Shim Retirement 尚未执行，`./task-loop run` 仍不能自动转发。 |
| 40.5 | AF-V10-004A Execution Forwarding v1 | preview_only | [`execution-forwarding-v1-readiness-evidence.md`](./execution-forwarding-v1-readiness-evidence.md)、`GET /api/v1/projects/{project}/execution-forwarding-v1-readiness`、`GET /api/v1/projects/{project}/execution-forwarding-v1-apply-preview`、`GET /api/v1/projects/{project}/execution-forwarding-v1-apply-packet`、`GET /api/v1/projects/{project}/execution-forwarding-v1-apply-gate`、`GET /api/v1/projects/{project}/execution-forwarding-v1-command-preview?task_type=...`、`GET /api/v1/projects/{project}/execution-forwarding-v1-rollback-preview`、`POST /api/v1/projects/{project}/execution-forwarding-v1-apply`、`areaflow project execution-forwarding-v1-readiness`、`areaflow project execution-forwarding-v1-apply-preview`、`areaflow project execution-forwarding-v1-apply-packet`、`areaflow project execution-forwarding-v1-apply-gate`、`areaflow project execution-forwarding-v1-command-preview --task-type ...`、`areaflow project execution-forwarding-v1-rollback-preview`、`areaflow project execution-forwarding-v1-apply`、Web `Forwarding v1` readiness/apply-preview/apply-packet Packet Gate/command-preview/rollback-preview panels、Desktop `Forwarding v1` readiness/apply-preview/apply-packet Packet Gate/command-preview/rollback-preview panels、`internal/project/execution_forwarding_v1_readiness.go`、`internal/project/execution_forwarding_v1_apply_preview.go`、`internal/project/execution_forwarding_v1_apply_packet.go`、`internal/project/execution_forwarding_v1_apply_gate.go`、`internal/project/execution_forwarding_v1_command_preview.go`、`internal/project/execution_forwarding_v1_rollback_preview.go`、`internal/project/execution_forwarding_v1_apply.go`、`internal/project/execution_forwarding_v1_readiness_test.go`、`internal/project/execution_forwarding_v1_apply_preview_test.go`、`internal/project/execution_forwarding_v1_apply_packet_test.go`、`internal/project/execution_forwarding_v1_apply_gate_test.go`、`internal/project/execution_forwarding_v1_command_preview_test.go`、`internal/project/execution_forwarding_v1_rollback_preview_test.go`、`internal/project/execution_forwarding_v1_apply_test.go`、`scripts/smoke-execution-forwarding-v1-readiness.sh`、`make smoke-docker-execution-forwarding-v1-readiness`、[`../architecture/api-surface.md`](../contracts/api-surface.md)、[`../migration/areamatrix-execution-cutover-boundary.md`](../migrations/areamatrix-execution-cutover-boundary.md)、[`../architecture/execution-opening-strategy.md`](../../../../proposals/execution-opening.md)、[`../architecture/completion-audit-contract.md`](../../../architecture/completion-audit-contract.md)、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、[`../../tasks/backlog/0-100-platform-backlog.md`](../plans/task-backlog.md) | 只读 readiness、apply-preview、apply-packet、apply-gate、command-preview 与 rollback-preview API/CLI、多端观察面和 focused PostgreSQL smoke 入口已实现，能展示 allowed task scope、read-only/evidence forwarding target matrix、blocked target matrix、fail-closed response fields，并对 allowed、blocked 和 unknown task type 返回只读 command response preview；apply-packet 生成 readiness snapshot hash、approval scope、proof ids、idempotency key、audit correlation id 和 gate/future apply 命令草稿；Web/Desktop Packet Gate 面板只读展示 readiness hash、canonical proof refs 和 gate blockers；readiness 和 apply-preview 会消费同项目 clean/authorized protected-path proof，以及同项目 complete execution cutover proof 里的 rollback-specific facts；rollback-to-read-only-shim proof 通过后，apply-gate 在 missing packet 或 complete packet but read-only shim blocked 时继续 fail closed，保持 `apply_command_eligible=false`；受保护 apply Command API/CLI 已落地，并在 gate 未通过时记录 blocked/denied command、event 和 audit，同时保持 run/task/attempt/artifact、legacy task-loop、project write、execution write、engine、secret、network 全部关闭；rollback preview 可在 fixture 中把 `rollback_v1:fail_closed` 与 `rollback_v1:proof_facts` 标成 pass，但 `rollback_v1:reopen_conditions` 仍 blocked；Web/Desktop 只展示 GET 结果，不提供 apply/rollback button；真实 forwarding smoke、rollback apply、真实 `./task-loop run` forwarding、真实 AreaMatrix legacy non-write proof、真实 rollback proof 和 AreaMatrix read-only shim 仍未落地，不打开 source write、generated retained write、repair、checkpoint、engine、secret、network、publish 或 restore。 |
| 41 | AF-V10-005 Security Boundary Readiness | implemented_scoped | [`security-boundary-readiness-evidence.md`](./security-boundary-readiness-evidence.md)、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、[`../architecture/auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)、[`../architecture/team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md)、`GET /api/v1/security/boundary-readiness`、`areaflow security boundary-readiness --json` | 只读 readiness API/CLI 已实现并测试覆盖，明确 auth/team/token/secret/remote worker/budget/integration/ops 高风险开口全部保持关闭；真实 auth enforcement、team permission、API token issuance/enforcement、secret resolve、remote worker credential、external API/webhook、budget/quota enforcement 和 managed ops 仍未打开。 |
| 42 | AF-V10-006 Completion Audit | implemented_scoped | [`completion-audit-evidence.md`](./completion-audit-evidence.md)、[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)、[`../architecture/completion-audit-contract.md`](../../../architecture/completion-audit-contract.md)、`GET /api/v1/completion-audit`、`GET /api/v1/completion-audit/snapshot-readiness`、`areaflow completion audit --json`、`areaflow completion audit-snapshot record`、`areaflow completion audit-snapshot readiness`、`areaflow completion source-alignment-proof record`、`areaflow completion task-matrix-proof record`、`areaflow completion validation-proof record`、`areaflow completion archive-proof record`、`areaflow completion shim-retirement-proof record`、`areaflow completion execution-cutover-proof record`、`areaflow completion backup-restore-proof record`、`areaflow completion release-packaging-proof record`、`areaflow completion security-closure-proof record`、`areaflow completion protected-path-proof record`、`scripts/smoke-source-alignment-proof.sh`、`scripts/smoke-task-matrix-proof.sh`、`scripts/smoke-validation-proof.sh`、`scripts/smoke-completion-proof.sh`、`scripts/smoke-completion-audit-full-proof.sh`、`scripts/smoke-completion-audit-release-candidate-snapshot.sh`、`scripts/smoke-execution-cutover-proof.sh`、`scripts/smoke-backup-restore-proof.sh`、`scripts/smoke-release-packaging-proof.sh`、`scripts/smoke-security-closure-proof.sh`、`scripts/smoke-operations-proof.sh`、`docs/product/master-plan.md`、`docs/product/phase-backlog.md`、[`implementation-gap-audit.md`](./implementation-gap-audit.md)、[`task-backlog-status-audit.md`](./task-backlog-status-audit.md) | 只读 completion audit API/CLI 已实现，并会在 E1-E9 缺证据时返回 incomplete/blocked；`areaflow completion audit-snapshot record` 只在当前 audit 顶层 `status=complete` 后记录 sealed snapshot，保存 audit hash、scope、release candidate label、evidence class、evidence URI 和 proof event IDs，且不运行 smoke、不创建 release package、不写 AreaMatrix、不启动 worker；`areaflow completion audit-snapshot readiness` 只读判断最新 snapshot 是否达到 release_candidate class，并会在非目标 project、非真实 AreaMatrix 身份和 fixture-only snapshot 时 fail closed；E1 已有受控 Source Alignment proof record 输入，completion audit 可消费 latest complete proof 并关闭 source alignment blocker，`scripts/smoke-source-alignment-proof.sh` 已覆盖 complete fact 校验、idempotent replay 和 PostgreSQL evidence consumption；E2 已有受控 Task Matrix proof record 输入，completion audit 可消费 latest complete proof 并关闭 task matrix blocker，`scripts/smoke-task-matrix-proof.sh` 已覆盖 complete fact 校验、idempotent replay 和 PostgreSQL evidence consumption；E3 已有受控 Validation proof record 输入，complete proof 必须绑定 reviewed validation command list、sha256 result hash、RFC3339 time window 和 scope，completion audit 可消费 latest complete proof 并关闭 fresh validation blocker，`scripts/smoke-validation-proof.sh` 已覆盖 complete fact 校验、validation-output binding、idempotent replay 和 PostgreSQL evidence consumption；E4 已有受控 Archive、Shim Retirement 和 Execution Cutover proof record 输入，complete proof 必须额外提供 release-candidate 形状 evidence URI、summary、`review_decision=approved`、非空 `reviewed_by` 和 RFC3339 `reviewed_at`，且 completion audit 只有在真实 AreaMatrix identity 通过时才消费这些 proof；`scripts/smoke-completion-proof.sh` 和 `scripts/smoke-execution-cutover-proof.sh` 已覆盖 complete fact 校验、idempotent replay 和 PostgreSQL evidence consumption，但这些 proof 不能代替真实 source review、task closure review、validation execution 或 execution cutover apply，也不会实际修改 AreaMatrix、转发 `./task-loop run` 或退役旧 runner；E5 已有受控 Release Packaging proof record 输入，completion audit 可消费 latest complete proof 并关闭 release final/packaging blockers，`scripts/smoke-release-packaging-proof.sh` 已覆盖 complete fact 校验、idempotent replay 和 PostgreSQL evidence consumption；E6 已有受控 Backup Restore proof record 输入，completion audit 可消费 latest complete proof 并关闭 backup/restore/artifact retention blockers，`scripts/smoke-backup-restore-proof.sh` 已覆盖 complete fact 校验、idempotent replay 和 PostgreSQL evidence consumption；E7 已改为读取 operations readiness，且 operations readiness 可消费 `ops.smoke_proof.recorded` 和 `000011_v1_migration_ledger.sql` 迁移 ledger 证据；`scripts/smoke-operations-proof.sh` 已覆盖 E7 focused PostgreSQL evidence consumption、idempotent replay 和 no-write safety facts；E8 已有受控 Security Closure proof record 输入，completion audit 可消费 latest complete proof 并关闭 project isolation / audit gap blockers，`scripts/smoke-security-closure-proof.sh` 已覆盖 complete fact 校验、idempotent replay 和 PostgreSQL evidence consumption，但 forbidden security opening 仍会保持 blocked；E9 已有受控 protected path proof record 输入，completion audit 可消费 latest clean/authorized proof；`scripts/smoke-completion-audit-full-proof.sh` 已在隔离 `areamatrix` fixture 中记录 E1-E9 proof 并证明 fixture project identity 会保持 completion audit blocked，随后 snapshot record 因当前 audit 不是 complete 而被拒绝；`scripts/smoke-completion-audit-release-candidate-snapshot.sh` 证明 synthetic reviewed URI 的临时 `areamatrix` fixture 不能记录 release_candidate snapshot，也不能进入 ready；当前仍不能声明真实 100%，因为 real release candidate / real AreaMatrix cutover closure evidence 尚未完成。 |
| 43 | AF-V10-007 Operations Readiness | implemented_scoped | [`operations-readiness-evidence.md`](./operations-readiness-evidence.md)、[`../architecture/operations-deployment-observability-boundary.md`](../../../architecture/operations-deployment-observability-boundary.md)、[`../architecture/api-surface.md`](../contracts/api-surface.md)、`GET /api/v1/ops/readiness`、`GET /api/v1/ops/support-bundle-preview`、`GET /api/v1/ops/migration-ledger-readiness`、`areaflow ops readiness --json`、`areaflow ops migration-ledger-readiness --json`、`areaflow ops smoke-proof record`、`areaflow support bundle-preview --json`、`scripts/smoke-operations-proof.sh`、`make smoke-docker-operations-proof`、Web `Operations Readiness` panel、Desktop `Operations Readiness` panel、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md) | 只读 operations readiness、metadata-only support bundle preview、migration ledger readiness、ops smoke proof record 和多端观察面已实现，并由 focused tests、E7 operations proof smoke、v1 stable fixture smoke 和 Web smoke 覆盖；proof record 只写 AreaFlow evidence，不运行 smoke、不导出 support bundle、不上传 telemetry、不应用 migration、不创建 migration ledger、不控制 service process、不写 AreaMatrix protected paths。full support export、remote telemetry 和 managed ops 仍未打开。 |

AF-V04-001 / Package A 补充索引：`make smoke-package-a`、`bash scripts/audit-package-a-readiness.sh`、
`bash scripts/audit-package-a-dirty-review.sh` 和 `bash scripts/audit-package-a-authorization-packet.sh --json`
提供真实 AreaMatrix `.areaflow/status.json` 窄写授权前检查。2026-07-10 Package A 已按窄授权写入真实
AreaMatrix `.areaflow/status.json`，当前真实 status projection 是 stable schema；Package A 没有授权
shim files、`workflow/README.md`、`workflow/versions/**`、task-loop forwarding、execution write、source write、
engine、secret、network、publish 或 restore。2026-07-06 起，authorization packet 还会绑定真实 target preimage 的 exists/sha256/size
和 accepted preimage schema status，并输出后续 apply gate 必须消费的 `--target`、`--expected-before-*`、
`--schema-uri`、`--validator-preflight`、`--protected-path-check`、`--protected-path-fingerprint-sha256`、
`--rollback-action`、`--accept-preimage-schema` 参数；`--source-hash` 只能来自当前
AreaFlow DB 最新 import snapshot 绑定。2026-07-10 起，authorization / packet / gate 顶层还暴露
`claim_scope=package_a_status_projection_preflight_only`、`not_real_100=true`、
`apply_command_eligible_is_not_apply=true` 和 `requires_separate_apply_command=true`，防止外部只看
`status=ready/pass` 或 `apply_command_eligible=true` 误判为已 apply。`--protected-path-fingerprint-sha256` 绑定除目标
`.areaflow/status.json` 之外的 protected paths 内容指纹，source-hash 绑定期间、写前或写后发生漂移都必须
fail closed。
若显式提供 `AREAFLOW_PACKAGE_A_SOURCE_HASH=<expected latest AreaFlow import snapshot hash>`，packet
仍会重新取 DB 权威 hash 并要求一致；缺少 DB 绑定、hash 缺失或 hash mismatch 时 packet 保持
`blocked_needs_authoritative_source_hash`。target preimage drift 或 source-hash 采集期间 protected-path
fingerprint drift 必须让后续 gate / write-time recheck fail closed。同日起，packet 支持
`AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<sha256>` +
`AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>` 对当前 dirty output 做精确复核；hash mismatch、缺 reviewer
或缺复核仍 blocked。该复核不是写入授权，审批三件套只在 `post_authorization_required_arguments`
中列出，必须等待窄授权后才可用于真实 apply；它也不允许 shim files、`workflow/README.md`、
`workflow/versions/**`、task-loop forwarding、execution write、source write、engine、secret、network、
publish 或 restore。

AF-V04-001 / Package B 补充索引：2026-07-11 新增 `make smoke-package-b-readiness`、
`bash scripts/audit-package-b-readiness.sh`、`bash scripts/audit-package-b-dirty-review.sh` 和
`bash scripts/audit-package-b-authorization-packet.sh --json`，用于真实 AreaMatrix read-only shim 落地前的
AreaFlow-only 预检。该预检确认当前 `.areaflow/status.json` 是 stable projection
(`0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383`)，并要求 protected dirty output hash
`43a22da86c19781f42d9319136ba812d1dc6f20f4941d5ec0bca4d929d0ee57c` 与 worktree dirty output hash
`96faa0f64e43b303b715f61e6064cb18a63bf5397582bffafe6d34eb853bd484` 被 reviewer 精确复核。未复核或 hash
mismatch 时保持 blocked；复核通过后只进入 `ready_for_package_b_area_matrix_edit_authorization`，仍必须等待
用户明确说出 `授权执行 Package B，只允许落地 AreaMatrix read-only shim，不允许 ./task-loop run 转发`。Package B
不授权 `.areaflow/status.json` 再写、`workflow/versions/**`、`progress.json`、logs/checkpoints、native doctor
隐式转发、`./task-loop run` forwarding、engine、secret、network、publish 或 restore。

AF-V10-006 补充索引：2026-07-06 新增 `scripts/smoke-completion-audit-release-candidate-snapshot.sh`
和 `make smoke-docker-completion-audit-release-candidate-snapshot`，用于证明临时 `project_key=areamatrix`
fixture 即使使用 synthetic reviewed evidence URI 并记录完整 proof metadata，completion audit 也会保持
blocked，并在 release_candidate snapshot record/readiness 上被真实 AreaMatrix project identity 门禁拦截。它是负向
fail-closed 证据，不代表真实 AreaMatrix cutover、release publish、restore apply、support export、remote
telemetry 或 managed ops 已打开；真实 100% 仍需要 real release candidate / AreaMatrix cutover closure evidence。

AF-V10-006 / real_100 breakdown 补充索引：2026-07-07 起，release / completion guardrail JSON
增加只读 `real_100_breakdown`，把当前 blocker 分成 `needs_exact_authorization`、
`needs_real_areamatrix_write`、`areaflow_only_can_continue` 和 `completed_evidence` 四类。它用于解释
Package A 精确授权、真实 AreaMatrix 写入/落地、AreaFlow-only 可继续工作和已关闭证据之间的差异；
它不改变 `real_100_status=blocked`，也不能把 fixture completion、局部 E4 proof complete 或 release
preview 解释为真实 100%。

AF-V10-006 / release-completion disambiguation 补充索引：2026-07-10 起，release / completion guardrail
JSON 额外暴露 `claim_scope`、`not_real_100=true`、`evidence_only=true`、
`status_alone_is_not_completion=true` 和 `release_candidate_decision`。这让外部脚本不能只凭
`status=ready/pass/complete` 判定真实 100% 或 release-ready。release_candidate snapshot 也会拒绝
`fixture/mock/demo/sample/synthetic/testdata/placeholder/dummy/example` 出现在 label、summary、snapshot URI、
E1-E9 proof URI 或 E7 proof provenance 中，并要求 `review_decision=approved`、非空 `reviewed_by` 和有效
RFC3339 `reviewed_at`；readiness 会对既有 snapshot 缺失审核 metadata 的情况返回
`completion_audit_snapshot_review_metadata_missing`。

AF-V10-006 / E6 backup-restore 补充索引：2026-07-06 起，`completion backup-restore-proof record`
的 `complete` proof 必须绑定 backup manifest hash/status/count、restore plan status/hash/count、
artifact integrity status/count 和 archive preview status/count/no-write safety facts；缺字段、manifest
hash mismatch、integrity failure、retention policy gap 或 archive preview write/delete attempt 会 fail
closed。`scripts/smoke-backup-restore-proof.sh`、`scripts/smoke-completion-audit-full-proof.sh` 和
`scripts/smoke-completion-audit-release-candidate-snapshot.sh` 会在临时 fixture 中收集这些只读/preview
输出并传入 proof；completion audit 会拒绝旧 loose proof metadata，并会重新运行最新
manifest/restore/integrity/archive preview 的只读 current binding 来阻断 proof 后稳定安全字段漂移；当前
archive preview 复核覆盖全部项目 artifact metadata，且不会新增 command/event/audit 行。当前
manifest hash 会作为 metadata 暴露并校验自洽，但 proof 记录自身会追加 command/event/audit 行，因此不把
pre-proof hash 等值作为 freshness 条件。这不是 restore apply、artifact copy/delete/upload、GC 或真实
AreaMatrix 写入授权。

## v1.x Deferred Matrix

| 顺序 | Task | 状态 | 打开前必须证明 |
|---:|---|---|---|
| 1 | AF-V1X-001 Real Generated-only Rollback Beta | deferred | 真实 managed project 单文件 generated/projection 写入后立即恢复 preimage；expected-before、preimage、verify、rollback、非目标文件指纹不变、R3 approval。 |
| 2 | AF-V1X-002 Real Generated-only Retained Apply | deferred | rollback beta 稳定后才允许保留 generated/projection 写入结果；expected-before、preimage、rollback verify、focused smoke、非目标文件指纹不变。 |
| 3 | AF-V1X-003 Manual Patch Artifact | deferred | AreaFlow 只生成 source patch/diff artifact、write-set preview、expected-before、验证计划和 rollback/remediation plan；不写项目源码。 |
| 4 | AF-V1X-004 Human-applied Source Evidence | deferred | 人工或现有 Codex 流程 apply 后，AreaFlow 只读取 diff、changed hash、验证结果并映射 copy/verify/checkpoint 语义。 |
| 5 | AF-V1X-005 Source Write Beta | deferred | `write_code`、write-set、copy/verify、checkpoint preview、rollback；禁止 delete/move/chmod/binary/symlink/glob/root 外路径。 |
| 6 | AF-V1X-006 Checkpoint Apply | deferred | source write beta 稳定后单独打开；dirty state、scope drift、checkpoint evidence、rollback/remediation、失败阻断下一 task。 |
| 7 | AF-V1X-007 Repair Plan / Repair Apply | deferred | 先生成 failure summary 和 repair plan artifact；repair apply 追加 attempt，不能跳过 verify 或 checkpoint gate。 |
| 8 | AF-V1X-008 No-secret Engine Execution | deferred | `secret_ref=none` engine execution、budget、redaction、no-secret/no-network 或 network allowlist evidence。 |
| 9 | AF-V1X-009 Secret Resolve | deferred | short-lived scoped binding；明文不进入 project config、artifact、event、audit 或 worker 长期状态。 |
| 10 | AF-V1X-010 Remote Worker | deferred | API-only worker、project/capability/lease scope、token rotation/revoke、heartbeat、audit trail。 |
| 11 | AF-V1X-011 Restore Apply | deferred | restore package、isolated dry-run、diff、preimage、rollback、R4 approval、audit。 |
| 12 | AF-V1X-012 Release Exception Real Write | deferred | schema preview、migration approval gate、apply preview、R4 approval、audit。 |
| 13 | AF-V1X-013 Release Publish Apply | deferred | tag/sign/upload/push/publish 拆分 Command API；package hash、evidence bundle、approval、rollout/remediation。 |
| 14 | AF-V1X-014 Plugin Execution | deferred | signed manifest、capability declaration、sandbox、conformance、disable/revoke、audit。 |
| 15 | AF-V1X-015 External Integrations And Webhooks | deferred | 按 `docs/architecture/integration-webhook-boundary.md` 逐级打开 catalog/readiness、delivery plan preview、fixture outbound/inbound、project-scoped delivery、inbound callback beta、external API connector 和 provider automation；callback / external API 不能绕过 Command API。 |
| 16 | AF-V1X-016 Team Console | deferred | 按 `docs/architecture/team-remote-control-boundary.md` 逐级打开 read-only preview、local auth console、team permission enforcement、remote read-only 和 remote command console；role / UI 不自动获得 project write、secret、publish、restore 或 worker credential。 |
| 17 | AF-V1X-017 Object Artifact Store | deferred | object backend verifier、hash/size integrity、retention policy、restore/archive story。 |
| 18 | AF-V1X-018 Budget And Quota | deferred | 按 `docs/architecture/budget-quota-boundary.md` 逐级打开 estimate、quota doctor、fixture reservation/charge、project-scoped enforcement、aggregation 和 provider reconciliation；禁止 silent throttling、重复扣费和无 project scope 阻断。 |
| 19 | AF-V1X-019 Managed Ops / Upgrade / Support Export | deferred | 按 `docs/architecture/operations-deployment-observability-boundary.md` 逐级打开 remote read-only ops、remote ops control、managed upgrade/rollback 和 full support bundle export；必须证明 auth/team scope、redaction、destination allowlist、backup/preimage、approval、audit、retention 和 revoke path。 |

## Current Closure Targets

按当前证据，最靠前的未关闭 task 是：

1. **AF-V04-001 Compatibility And Shim Readiness**：AreaFlow 侧 preview/readiness/evidence 记录链、
   只读 `shim-authorization` 和只读 `shim-apply-packet/gate` API/CLI 已有；2026-07-04 smoke 已证明 JSON 与普通 CLI 文本授权包都展示
   status projection schema preflight、编辑前后保护路径专项 git status、post-edit verification 和 rollback scope；
   focused tests、compatibility Docker smoke 和 v1 stable fixture 长链已证明 shim apply packet/gate 会校验 allowed files、authorization snapshot、status projection
   packet/gate proof、real read-only smoke evidence、dirty worktree review、protected path fingerprint、rollback plan 的 project-scoped proof reference
   和 explicit approval，并在 readiness 证据缺失时 fail closed；fixture 证据完整时只生成 ready future packet，
   真实 AreaMatrix 仍 blocked/read-only；
   当前真实 AreaMatrix `.areaflow/status.json` 已由 Package A 更新为 stable projection，`real_areamatrix_status_projection_schema`
   不再是 shim readiness 的当前 blocker；AreaMatrix 仓库内 shim 文件仍没有落地，这是跨仓库写入，需要单独授权。
2. **AF-V10-004 AreaMatrix Execution Cutover Readiness**：只读 preview/readiness 与命令边界文档已有；
   仍需要真实 AreaMatrix shim 和显式 execution cutover approval 证据，当前不应执行 cutover 或转发
   `./task-loop run`。
3. **AF-V10-004A Execution Forwarding v1**：只读 readiness、apply-preview、apply-packet、apply-gate、
   command-preview、rollback-preview 与受保护 apply Command API/CLI 和 focused PostgreSQL
   smoke 入口已实现，可以查询 allowed scope、消费 read-only verify / AreaFlow-owned artifact evidence、列出
   future apply packet fields、required proof facts、explicit approval、read-only shim 前置、同项目 protected-path
   proof 和 rollback 缺口；受保护 apply command 当前只记录 blocked/denied command、event 和 audit，不创建
   run/task/attempt/artifact，也不转发 legacy task-loop；真实 forwarding smoke、rollback、真实
   `./task-loop run` forwarding、真实 AreaMatrix legacy non-write proof 和 AreaMatrix read-only shim 尚未落地。第一版 forwarding 只能承接 read-only / evidence 类任务；real
   generated rollback beta、retained generated apply、manual patch / human-applied source evidence、source write、
   repair、checkpoint、engine、secret、network、publish 和 restore 都是后续独立门禁，不应阻塞 Forwarding v1
   的 readiness 设计，也不能被 Forwarding v1 自动打开。

在没有用户明确授权前，下一步低风险推进可以继续补 Desktop service manager / notification 的只读设计，
或补 Web/Desktop 对只读 gate 的展示细节；不要直接修改 AreaMatrix，也不要打开真实
generated-only retained apply。

当前低风险推进也可以继续留在 AreaFlow-only 文档和实现内，例如继续补 workflow profile conformance、
multi-project isolation fixture 或只读 Web/Desktop 展示。只要涉及 AreaMatrix
`workflow/README.md`、`.areaflow/status.json`、`scripts/**`、shim 文件或 `./task-loop` forwarding，就从
AreaFlow-only 工作升级为跨仓库授权点。

E1 completion audit 口径补充：Source Alignment proof 的 `complete` 记录不再是 facts-only。
CLI 会自动只读绑定当前 AreaFlow E1 source path/hash/source-set hash，completion audit 会重算
current binding，并拒绝旧 loose proof 或 post-proof source drift。该绑定只读 AreaFlow 源文件，
不改写文档、不运行 shell、不写 AreaMatrix。

E4 completion audit 口径补充：Archive、Shim Retirement 和 Execution Cutover proof 的 `complete` 记录不再是
facts-only。Archive proof 必须绑定 `areamatrix_historical_execution_reference_only` scope、metadata-only reference
mode、required source path set、forbidden action set、rollback target 和 fail-closed policy；Shim Retirement proof
必须绑定 retirement scope、required prerequisites、retired legacy surface set、rollback target、fail-closed 和
reopen approval policy；Execution Cutover proof 必须绑定 `execution_forwarding_v1_read_only_evidence_only` scope、
allowed task types、forbidden actions、rollback target/mode、fail-closed 和 reopen approval policy。三类 proof 还必须携带
release-candidate evidence URI、`review_decision=approved`、非空 `reviewed_by`、RFC3339 `reviewed_at`、
deterministic current binding hash，且 latest proof EventID 必须为正。completion audit 会拒绝旧 loose metadata、
hash 漂移、缺少事件 ID、local/fixture/script/smoke 机制证据、缺少 review metadata 或非真实 AreaMatrix identity；
这仍不代表真实 AreaMatrix archive、`./task-loop run` forwarding 或 shim retirement 已执行。

## Implementation Authorization Boundary

从讨论转入真实实施时，必须区分三种授权：

1. **AreaFlow 文档固化**：只修改 AreaFlow `docs/**` 或 `tasks/**`，用于同步 0-100% 源事实。
2. **AreaFlow 平台实现**：修改 AreaFlow 代码、migration、API、CLI、Web 或 Desktop；仍不得触碰
   AreaMatrix 文件，除非任务明确要求跨仓库 dogfood。
3. **AreaMatrix 跨仓库联动**：修改 AreaMatrix `workflow/README.md`、`.areaflow/status.json`、
   `scripts/**` 或任何 shim / task-loop 行为；必须单独确认影响、风险、验证和回滚。

当前最靠前的跨仓库授权点是 AF-V04-001 的 AreaMatrix compatibility shim 真实落地。没有该授权时，
execution cutover readiness 应继续保持 blocked，`./task-loop run` 不能转发到 AreaFlow。授权前的最新
AreaFlow-only 预检入口是 `make smoke-docker-shim-authorization-preflight`；它复用真实 AreaMatrix
read-only smoke，验证 status/shim authorization、apply packet/gate 和 no-write safety facts，但不写
真实 AreaMatrix。
