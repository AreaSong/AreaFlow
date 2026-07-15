# Implementation Gap Audit

## Purpose

本文记录 AreaFlow 0-100% 路线在当前仓库中的实现证据和剩余缺口。它不替代
[`roadmap.md`](../../../roadmap.md) 或
[`../product/platform-blueprint.md`](../plans/platform-blueprint.md)；它只回答：

```text
哪个阶段已有代码/API/测试证据
哪个阶段仍只是 preview 或 dry-run
哪个阶段还需要真实 dogfood、PG smoke 或后续高风险设计
```

规划源事实：

- [`../product/master-plan.md`](../plans/master-plan.md)：0-100% 总控计划、执行前最终确认清单和
  v1.0 / v1.x 完成边界。
- [`../product/phase-backlog.md`](../plans/phase-backlog.md)：0-100% 阶段目标、禁区、门禁和 AreaMatrix 影响。
- [`../architecture/execution-opening-strategy.md`](../../../../proposals/execution-opening.md)：execution
  开闸顺序、受限 apply 状态、copy / verify / repair / checkpoint 证据和 AreaMatrix first execution policy。
- [`../migration/areamatrix-execution-cutover-boundary.md`](../migrations/areamatrix-execution-cutover-boundary.md)：
  AreaMatrix execution cutover 的命令映射、apply 前置缺口、protected paths 和 rollback 规则。
- [`../../tasks/backlog/0-100-platform-backlog.md`](../plans/task-backlog.md)：候选任务索引和完整推荐推进顺序；仍是 backlog，不代表 active execution。
- [`../milestones/README.md`](../milestones/README.md)：每个 milestone 的 go/no-go 摘要。
- [`task-backlog-status-audit.md`](./task-backlog-status-audit.md)：按 backlog 推荐顺序整理的 task-level 状态、证据和最靠前缺口。
- [`directory-boundary-audit.md`](./directory-boundary-audit.md)：当前目录结构与长期模块边界的阶段性审计。
- [`governance-boundary-audit.md`](./governance-boundary-audit.md)：权限、风险等级、API、audit 和高风险能力边界审计。
- [`../architecture/operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md)：
  install、migration、service lifecycle、diagnostics、support bundle、telemetry、upgrade 和 rollback 边界。
- [`../architecture/completion-audit-contract.md`](../contracts/completion-audit-contract.md)：0-100%
  完成审计、release packaging preview、dogfood cutover 和 evidence 聚合边界。
- [`bootstrap-smoke-evidence.md`](./bootstrap-smoke-evidence.md)：最近一次 v0.1 PostgreSQL bootstrap fixture smoke 证据。
- [`areamatrix-adapter-import-evidence.md`](./areamatrix-adapter-import-evidence.md)：最近一次真实 AreaMatrix 只读 metadata import 证据。
- [`status-projection-evidence.md`](./status-projection-evidence.md)：最近一次 fixture status projection apply 证据。
- [`shadow-doctor-readiness-evidence.md`](./shadow-doctor-readiness-evidence.md)：最近一次 v0.2 doctor/readiness/import-diff/verify-bundle 证据。
- [`native-doctor-authorization-evidence.md`](./native-doctor-authorization-evidence.md)：最近一次 v0.2 native doctor 授权边界证据。
- [`workflow-version-authoring-evidence.md`](./workflow-version-authoring-evidence.md)：最近一次 v0.3 workflow version authoring fixture 证据。
- [`gate-transition-approval-evidence.md`](./gate-transition-approval-evidence.md)：最近一次 v0.3 gate/transition/approval fixture 证据。
- [`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)：最近一次 v0.4 compatibility/shim readiness fixture 证据。
- [`authoring-cutover-apply-evidence.md`](./authoring-cutover-apply-evidence.md)：最近一次 v0.4 authoring cutover apply fixture 证据。
- [`runner-preview-evidence.md`](./runner-preview-evidence.md)：最近一次 v0.5 runner preview focused smoke 证据。
- [`run-control-evidence.md`](./run-control-evidence.md)：最近一次 v0.5 run control focused smoke 证据。
- [`worker-lease-evidence.md`](./worker-lease-evidence.md)：最近一次 v0.6 worker registry / lease lifecycle focused smoke 证据。
- [`codex-cli-adapter-preview-evidence.md`](./codex-cli-adapter-preview-evidence.md)：最近一次 v0.6 Codex CLI adapter preview focused smoke 证据。
- [`execution-approval-gate-evidence.md`](./execution-approval-gate-evidence.md)：最近一次 v0.6 execution approval gate focused smoke 证据。
- [`fixture-execution-apply-evidence.md`](./fixture-execution-apply-evidence.md)：最近一次 v0.6 fixture execution apply focused smoke 证据。
- [`read-only-verify-evidence.md`](./read-only-verify-evidence.md)：最近一次 v0.6 read-only verify focused smoke 证据。
- [`approved-artifact-write-evidence.md`](./approved-artifact-write-evidence.md)：最近一次 v0.6 approved artifact write focused smoke 证据。
- [`execution-plan-preview-evidence.md`](./execution-plan-preview-evidence.md)：最近一次 v0.6 execution plan preview focused 证据。
- [`project-write-design-gate-evidence.md`](./project-write-design-gate-evidence.md)：最近一次 v0.6 approved project write design gate focused 证据。
- [`fixture-project-write-evidence.md`](./fixture-project-write-evidence.md)：最近一次 v0.6 fixture-only approved project write focused 证据。
- [`managed-generated-write-gate-evidence.md`](./managed-generated-write-gate-evidence.md)：最近一次 v0.6 managed generated write gate focused 证据。
- [`managed-generated-write-apply-evidence.md`](./managed-generated-write-apply-evidence.md)：最近一次 v0.6 managed generated write apply + API/CLI focused 证据。
- [`generated-write-readiness-evidence.md`](./generated-write-readiness-evidence.md)：最近一次 v0.6 generated write dogfood readiness focused 证据。
- [`generated-write-apply-beta-gate-evidence.md`](./generated-write-apply-beta-gate-evidence.md)：最近一次 v0.6 generated write apply beta gate focused 证据。
- [`execution-forwarding-v1-readiness-evidence.md`](./execution-forwarding-v1-readiness-evidence.md)：最近一次 Execution Forwarding v1 readiness / apply-preview focused smoke 入口和只读边界证据。
- [`web-write-action-gate-evidence.md`](./web-write-action-gate-evidence.md)：最近一次 v0.7 Web write action gate focused 证据。
- [`desktop-shell-scaffold-evidence.md`](./desktop-shell-scaffold-evidence.md)：最近一次 v0.9 Tauri desktop shell scaffold 证据。
- [`desktop-service-control-gate-evidence.md`](./desktop-service-control-gate-evidence.md)：最近一次 v0.9 Desktop service control gate focused 证据。
- [`desktop-notification-gate-evidence.md`](./desktop-notification-gate-evidence.md)：最近一次 v0.9 Desktop notification gate focused 证据。
- [`desktop-tray-menu-gate-evidence.md`](./desktop-tray-menu-gate-evidence.md)：最近一次 v0.9 Desktop tray/menu gate focused 证据。
- [`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)：最近一次 v1.0 stable platform fixture smoke 证据。
- [`multi-project-isolation-evidence.md`](./multi-project-isolation-evidence.md)：`project_key` 多项目隔离 PostgreSQL fixture 入口和当前 skip 证据。
- [`security-boundary-readiness-evidence.md`](./security-boundary-readiness-evidence.md)：最近一次 v1.0 security boundary readiness API/CLI focused 证据。
- [`completion-audit-evidence.md`](./completion-audit-evidence.md)：最近一次 v1.0 completion audit API/CLI focused 证据。

## Evidence Levels

```text
implemented:
  有代码路径、测试或 smoke 证据，且能力范围与 milestone 边界一致。

implemented_scoped:
  有受限实现和证据，但只在 fixture/temp project、AreaFlow-owned artifact、只读 evidence、
  generated-only rollback drill 或其他明确 scope 内成立；不能解释为真实 AreaMatrix apply、
  execution cutover 或 v1.x 高风险能力已经打开。

preview_only:
  只读 preview / dry-run / readiness 已实现；真实 apply、execution、publish 或 restore 尚未打开。

needs_pg_smoke:
  单元测试存在，但真实 PostgreSQL、本地 artifact store 或 API/Web 联动仍需 smoke 证明。

planned:
  只有文档和 schema 边界，尚未作为真实能力启用。

deferred:
  明确保留到 post-100% v1.x；不作为 v1.0 必交付能力。
```

## Completion Audit Rule

判断 AreaFlow 是否达到 100% 时，必须从当前仓库状态、命令输出、smoke evidence 和 gate result
逐项证明，而不是根据 roadmap 意图、设计完成度或绿色摘要推断完成。

以下情况一律不能作为 100% 完成证据：

- `preview_only` 被当作真实 apply。
- `implemented_scoped` 被当作真实 AreaMatrix 写入或 execution cutover。
- AreaMatrix dogfood 被 AreaFlow self dogfood 替代；self dogfood 只能作为第二主线证据。
- 多项目 API 返回成功但未证明 `project_key` 隔离覆盖 workflow、run、lease、artifact、secret 和 audit。
- workspace / environment 仍按设计后置时，被误判为缺少 v1.0 必需表；反过来也不能用 metadata 字段替代
  真实多环境执行证据。
- `release final gate preview` 被当作真实发布。
- `release final gate pass` 被当作 100% completion audit。
- `release_candidate` snapshot 机制或 synthetic fixture evidence 被当作真实 AreaMatrix release candidate / cutover
  证据；临时 fixture 必须被真实项目身份门禁拦截。
- release exception preview 被当作真实 exception apply。
- `service status` 被当作 process control、remote ops control 或 managed upgrade。
- `support bundle preview` 被当作真实 support export。
- local diagnostics 被当作远程 telemetry opt-in。
- `restore dry-run` 被当作真实恢复。
- `generated write readiness` 被当作 retained generated apply。
- `execution-cutover-readiness` blocked 时，仍声称 `./task-loop run` 已可转发。
- `secret_ref` readiness 被当作真实 secret resolve。

如果某项证据只能证明 fixture、preview、readiness 或 metadata-only 状态，completion audit 必须继续标记为
incomplete，直到对应 apply / cutover / release / restore gate 有明确通过证据。

## Phase Matrix

| 阶段 | 当前状态 | 当前证据 | 仍需补齐 |
|---|---|---|---|
| Phase 0 Foundation | implemented | ADR 0001-0005、architecture、migration、milestones、profile docs 已建立。 | 持续保持 ADR 与实现同步。 |
| v0.1 Import + Status Mirror | implemented | `project add/import/export-status`、AreaMatrix adapter、core schema、status exporter、`status-projection-authorization`、`status-projection-apply-packet`、`status-projection-apply-gate`、`schemas/status-projection.schema.json`、[`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)、[`../architecture/areamatrix-import-scope-contract.md`](../contracts/areamatrix-import-scope-contract.md)、[`../architecture/data-model-v0.1.md`](../contracts/data-model-v0.1.md)、[`../architecture/project-config.md`](../contracts/project-config.md)、`smoke-fixture` 安全 PG 写入 fixture 并按 schema 校验 `.areaflow/status.json`、`smoke-areamatrix-readonly` 真实 AreaMatrix 只读覆盖；真实只读 smoke 证明 status projection authorization preview 能读取真实 `.areaflow/status.json` preimage、生成 expected-before write-set / schema validator / rollback facts，apply packet preview 能生成 CLI/API packet 且不写入，apply gate 在缺失 packet 时 fail closed，且不创建 status apply command/status_projection rows、不写真实文件；2026-07-10 Package A 已按窄授权把真实 AreaMatrix `.areaflow/status.json` 更新为 `stable_fallback_projection_v1`，当前文件 hash 为 `0ee84e5dbd5a75ae40e7ef78016c631c8c03f7d53264b7aab9dac9d46027a383`；最近证据见 [`areamatrix-adapter-import-evidence.md`](./areamatrix-adapter-import-evidence.md) 和 [`status-projection-evidence.md`](./status-projection-evidence.md)。 | 后续再次写真实 AreaMatrix `.areaflow/status.json` 仍需要单独授权；Package A 不授权 shim 文件、`workflow/README.md`、execution、worker、engine、secret、cutover 或 publish；authorization preview 的 `needs_approval`、apply packet preview 的 `ready_for_apply_command` 和 apply gate 的 `apply_command_eligible=true` 都不能解释为 apply 已发生；AreaMatrix 历史 prompt/log/report/diff/evidence 原文仍未复制。 |
| v0.2 Shadow Doctor + Drift Check | implemented | doctor、summary、readiness、import-diff、verification-bundle、events/API 测试、[`../architecture/v0.2-shadow-doctor-contract.md`](../contracts/v0.2-shadow-doctor-contract.md)、`smoke-fixture` 和 `smoke-areamatrix-readonly` 覆盖；最近证据见 [`shadow-doctor-readiness-evidence.md`](./shadow-doctor-readiness-evidence.md) 和 [`native-doctor-authorization-evidence.md`](./native-doctor-authorization-evidence.md)。 | native doctor 仍必须受 command allowlist 和人工授权约束；真实 AreaMatrix 未授权写 status mirror 时 v0.2 phase gate 应保持 blocked；verification bundle 不能被解释为 authoring cutover、execution cutover 或 worker execution 已打开。 |
| v0.3 New Version Authoring | implemented | workflow version create、authoring stage skeleton、item links、profile binding drift gate、gate result、transition preview、approval record、command request 和 audit 链已有 PG fixture 证据；[`../architecture/v0.3-version-authoring-contract.md`](../contracts/v0.3-version-authoring-contract.md) 已定义 version authoring no-apply 边界。最近证据见 [`workflow-version-authoring-evidence.md`](./workflow-version-authoring-evidence.md) 和 [`gate-transition-approval-evidence.md`](./gate-transition-approval-evidence.md)。 | 只接管 AreaFlow-authored records；不写被管理项目 workflow 目录；approval record 不等于 execution，promotion preview 不 apply。 |
| v0.4 Workflow Ownership Cutover | implemented / preview_only | compatibility contract、`shim-preview`、`shim-readiness`、只读 `shim-authorization`、只读 `shim-apply-packet/gate`、AreaFlow-only `shim-apply`、`shim-readiness-evidence`、approval gate、live mapping gate、cutover readiness bundle/gate 已有 CLI/API/测试；`shim-readiness` 已声明 `.areaflow/status.json` 的 `stable_fallback_projection_v1` required/forbidden schema contract，`shim-authorization` 已要求 `status-projections` 与 stable schema preflight；`shim-apply-packet/gate` 已能生成 authorization snapshot hash、allowed files、status projection packet/gate project-scoped proof references、protected path fingerprint、rollback plan、explicit approval、idempotency key 和 audit correlation id，并在 readiness evidence 缺失、proof reference 缺失或 proof reference 未按 `<project_key>:<evidence_kind>:<id>` 绑定时 fail closed；2026-07-11 `shim-apply` CLI/API 已定义为受保护 AreaFlow-only command 入口，gate 通过时记录 `command_requests` / `events` / `audit_events`，真实只读 smoke 的缺 packet 分支仍 blocked，且两条路径都断言 `project_write_attempted=false`、`execution_write_attempted=false`、`status_projection_written=false`、`area_matrix_files_modified=false`；compatibility fixture 和 v1 stable fixture 长链证明 readiness evidence 记录后完整 packet 可变为 `ready_for_future_apply_command` 但仍不写文件，真实 AreaMatrix read-only smoke 证明仍 blocked/read-only；[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../contracts/v0.4-workflow-ownership-cutover-contract.md) 已定义 compatibility、shim readiness、DB-only cutover apply 和 rollback 边界；`project.cutover.apply` 已进入 Command API 幂等边界，只执行 AreaFlow DB 内 authoring cutover，并记录 event/audit/command response；AreaMatrix shim 最小落地计划已记录；`smoke-compatibility-fixture` 和 `smoke-v1-stable-fixture` 提供安全 fixture 证据。最近证据见 [`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)、[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md) 和 [`authoring-cutover-apply-evidence.md`](./authoring-cutover-apply-evidence.md)。 | `shim-readiness`、`shim-authorization`、`shim-apply-packet/gate` 和 AreaFlow-only `shim-apply` 即使前置 evidence 具备，仍不能被解释为 AreaMatrix 编辑已获授权；`project.cutover.apply` 只切 AreaFlow DB 内 authoring ownership，保持 `project_write_attempted=false`、`execution_write_attempted=false`、`area_matrix_write_attempted=false`；AreaMatrix 仓库内 shim 文件修改、`workflow/README.md` 受控写入、真实项目文件写入和 execution cutover 仍未打开。 |
| v0.5 Runner Preview | preview_only | run、run_task、run_attempt、runner preview report、risk/preflight、artifact metadata、`runner.preview` completed command response、`run.start` / `run.drain` / `run.cancel` protected Command API 已有测试和 safe fixture smoke；[`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md) 已定义 dry-run execution model、runner preview report、run control、safety facts 和 no-execution 边界；2026-07-05 PG smoke assertion 已要求 runner/run-control persisted command response 包含 `area_matrix_write_attempted=false`。最近 focused 证据见 [`runner-preview-evidence.md`](./runner-preview-evidence.md) 和 [`run-control-evidence.md`](./run-control-evidence.md)。 | 真实 runner.run、AI engine 调用、项目文件写入仍未打开；worker 级中断/协作 drain 仍属后续 execution beta；runner/run-control 仍只是 dry-run / protected DB-state proof。 |
| v0.6 Worker Beta | implemented / preview_only | worker register/heartbeat/lease/run-once、capability denial、worker_run_once_report 已有 CLI/API/测试和 smoke-local；[`../architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md) 已定义 worker lifecycle、lease、dry-run run-once、scoped execution evidence、generated rollback drill 和 no-cutover 边界；worker registry / lease lifecycle focused smoke 已证明 `worker.register` / `worker.heartbeat` / `lease.acquire` / `lease.release` / `lease.recover` command response 和 denial 安全边界；Codex CLI adapter preview 已覆盖 engine/profile readiness、command/capability/path preflight 和 artifact redaction plan；execution approval gate 已覆盖真实 execution apply 前的只读 go/no-go、dry-run denial、worker capability readiness 和 API/CLI safety facts；fixture execution apply 已证明 approval-gated `run.fixture_queue` / `worker.fixture_execute` 能闭环到 lease、attempt、artifact、run_task passed 和 run passed；read-only verify 已证明 approval-gated `run.read_only_verify_queue` / `worker.read_only_verify` 能读取 allowlisted project file、生成 hash/size evidence，并闭环到 run_task verified 和 run verified；approved artifact write 已证明 approval-gated `run.approved_artifact_write_queue` / `worker.approved_artifact_write` 能只写 AreaFlow-owned artifact store，并闭环到 run_task/run `artifact_written`；execution plan preview 已能只读展示 execution approval gate、approved artifact write、copy、verify、checkpoint 和 repair 的 status / blockers / safety facts；project write design gate 已能只读展示 approved project write 的 write-set、unsupported operations、apply sequence、rollback contract 和 safety facts；fixture-only approved project write 已有 service/API/CLI focused tests，覆盖 approved write-set、expected-before hash/size、preimage/report artifact、copy/verify/rollback attempts 和 rollback_verified safety facts；managed generated write gate 已有 service/API/CLI focused tests，覆盖 generated-only prefixes、required write-set fields、source/execution/checkpoint/repair blockers 和 read-only safety facts；managed generated write apply 已有 service/API/CLI focused tests 和真实 PostgreSQL smoke，覆盖 `write_generated` 默认能力、generated-only path policy、fixture/temp scope、non-fixture denial、command response safety facts 和 rollback drill contract；真实 AreaMatrix 文件指纹保护栏需显式 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` 才运行；generated write dogfood readiness 已有 service/API/CLI focused tests，覆盖 project-scoped config/permission 前置检查、`ready_for_review`、`apply_open=false`、`real_areamatrix_write_opened=false` 和只读 safety facts；generated write apply beta gate 已有 service/API/CLI focused tests，覆盖 nested readiness、explicit R3 approval blocker、`approval_status=needs_approval`、`apply_open=false` 和 read-only safety facts。最近 focused 证据见 [`worker-lease-evidence.md`](./worker-lease-evidence.md)、[`codex-cli-adapter-preview-evidence.md`](./codex-cli-adapter-preview-evidence.md)、[`execution-approval-gate-evidence.md`](./execution-approval-gate-evidence.md)、[`fixture-execution-apply-evidence.md`](./fixture-execution-apply-evidence.md)、[`read-only-verify-evidence.md`](./read-only-verify-evidence.md)、[`approved-artifact-write-evidence.md`](./approved-artifact-write-evidence.md)、[`execution-plan-preview-evidence.md`](./execution-plan-preview-evidence.md)、[`project-write-design-gate-evidence.md`](./project-write-design-gate-evidence.md)、[`fixture-project-write-evidence.md`](./fixture-project-write-evidence.md)、[`managed-generated-write-gate-evidence.md`](./managed-generated-write-gate-evidence.md)、[`managed-generated-write-apply-evidence.md`](./managed-generated-write-apply-evidence.md)、[`generated-write-readiness-evidence.md`](./generated-write-readiness-evidence.md) 和 [`generated-write-apply-beta-gate-evidence.md`](./generated-write-apply-beta-gate-evidence.md)。 | 已打开的 execution apply 仅限 fixture-only AreaFlow state / artifact store、allowlisted read-only project file hashing、AreaFlow-owned approved artifact write、fixture project 内单个已存在 allowlisted 文件的 write/verify/rollback drill，以及 fixture/temp generated-only service/API/CLI 的 write/verify/rollback drill；真实 AreaMatrix 写入、保留 generated apply 结果、source write、Codex CLI adapter 真实 copy/verify/repair/checkpoint execution、远程 worker 凭证管理和真实 AreaMatrix execution cutover 仍未打开；v0.6 evidence 必须保留 scope label，不能累计成 `./task-loop run` forwarding。 |
| v0.7 Web Dashboard | implemented_scoped | React/TS Web 使用 `/api/v1` GET；[`../architecture/v0.7-web-dashboard-contract.md`](../contracts/v0.7-web-dashboard-contract.md) 已定义 `/api/v1`、SSE、read-only panels、write action gate、project_key visibility guard、no-second-state 和 no-Web-write 边界；project/version/run/artifact/residual/approval/worker/audit 面板、build 和 `smoke-web` browser check 覆盖；run detail 调用已使用 `?project_key=` visibility guard，`smoke-web-check.mjs` 会点击 run timeline 验证 query scope；Web write action gate 已能只读展示 approval、drain、cancel、archive、status projection 和 generated write beta 的 disabled/read-only 打开要求；Schedule Preview 已展示 engine blocker，例如 `engine_profile_disabled`；Shim Authorization 面板已展示 `shim-authorization` blocked gate、allowed files 和 safety facts；Shim Apply Review / Packet Gate 面板已展示 `shim-apply-packet/gate` 的 proof facts、blocked items、future command type 和 no-write safety facts；Execution Cutover 面板已展示 `execution-cutover-readiness` blocked gate、`explicit_execution_cutover_approval` 和 no-write / no-task-loop safety facts；Execution Forwarding v1 面板已展示 readiness、apply-preview、apply-packet / Packet Gate 和 rollback-preview 的只读 blocked/safe 状态，并显示 readiness hash、canonical proof refs 与 gate blockers；Operations Readiness 面板已展示 `GET /api/v1/ops/readiness` 的 service/support/migration/telemetry/managed ops 状态；Release Final Gate、Evidence Bundle、Package Preview、Distribution Preview、Publish Gate、Publish Approval 和 Rollout Plan 面板已展示 release preview 链、关键 item 和 release package / approval / rollout / publish / apply 禁区。最近 focused 证据见 [`web-write-action-gate-evidence.md`](./web-write-action-gate-evidence.md)、[`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)、[`execution-cutover-readiness-evidence.md`](./execution-cutover-readiness-evidence.md)、[`execution-forwarding-v1-readiness-evidence.md`](./execution-forwarding-v1-readiness-evidence.md)、[`operations-readiness-evidence.md`](./operations-readiness-evidence.md) 和 [`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)。 | 需要持续确保 Web 只通过 API/SSE 观察状态；Web 写操作、AreaMatrix shim 编辑、shim apply command、status projection apply、execution cutover apply、Execution Forwarding v1 apply/rollback、`./task-loop run` forwarding、support export、migration apply、telemetry upload、release package、release approval、rollout 和 publish/apply 仍应保持关闭直到 Command API 风险面稳定并获得显式授权。 |
| v0.8 Multi-project Worker Pool | preview_only | worker pool summary、schedule-preview、resource readiness、engine readiness 已有 CLI/API/测试和 smoke-local；阶段合同见 [`../architecture/v0.8-multi-project-worker-pool-contract.md`](../contracts/v0.8-multi-project-worker-pool-contract.md)，明确 summary / schedule preview / readiness 不能解释为 scheduler apply、lease claim、worker dispatch、secret resolve、remote worker credential、team/auth enforcement 或 execution cutover；`TestStoreProjectKeyIsolationWithPostgres` 提供并通过 PostgreSQL project scope fixture，`TestProjectScopedAPIsUseRouteProjectKey` 证明 versioned API route 使用 `{project_key}` 隔离 summary/events/audit/workers/workflow versions/runs/artifacts；`TestGlobalRunEndpointsHonorProjectKeyVisibility` 和 `TestGlobalArtifactEndpointsHonorProjectKeyVisibility` 证明全局 run/artifact ID route 在提供 `project_key` 时执行 visibility guard；auth/team/API token/secret/remote worker R4 ladder 已在 [`../architecture/auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md) 成文。最近证据见 [`multi-project-isolation-evidence.md`](./multi-project-isolation-evidence.md)。 | 多项目真实调度、远程 worker、engine routing 执行仍未打开；API token enforcement、team permission enforcement、secret resolve 和 remote worker credential 仍未打开。 |
| v0.9 Desktop Shell | implemented_scoped / planned | `service status` CLI/API 已实现，返回 dashboard/API/worker pool/capabilities/forbidden actions；阶段合同见 [`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)，明确 Desktop 是 API client，不是第二个 AreaFlow；`GET /api/v1/desktop/service-control-gate` 和 `areaflow desktop service-control-gate [--json]` 已实现只读 service control gate，只允许 dashboard launcher，start/stop/restart/notification/tray 保持 disabled；`GET /api/v1/desktop/notification-gate` 和 `areaflow desktop notification-gate [--json]` 已实现只读 notification gate，只允许 event stream requirement 观察，OS notification、approval/run/worker 通知保持 disabled；`GET /api/v1/desktop/tray-menu-gate` 和 `areaflow desktop tray-menu-gate [--json]` 已实现只读 tray/menu gate，只允许 dashboard/status/events 菜单项作为 read-only/launcher，service control、notification、settings 保持 disabled；三条 Desktop gate CLI 均已进入 `smoke-local.sh` 与 2026-07-04 v1 stable fixture 长链；`desktop/` Tauri scaffold 已创建并读取 service status、service-control gate、notification gate、tray-menu gate、ops readiness、selected project 的 shim authorization blocked gate、shim apply packet/gate read-only review、execution cutover readiness blocked gate、Execution Forwarding v1 readiness/apply-preview/apply-packet Packet Gate/rollback-preview blocked/read-only gates 和 global release rollout preview chain。最近证据见 [`desktop-shell-scaffold-evidence.md`](./desktop-shell-scaffold-evidence.md)、[`compatibility-shim-readiness-evidence.md`](./compatibility-shim-readiness-evidence.md)、[`desktop-service-control-gate-evidence.md`](./desktop-service-control-gate-evidence.md)、[`desktop-notification-gate-evidence.md`](./desktop-notification-gate-evidence.md)、[`desktop-tray-menu-gate-evidence.md`](./desktop-tray-menu-gate-evidence.md)、[`execution-cutover-readiness-evidence.md`](./execution-cutover-readiness-evidence.md)、[`execution-forwarding-v1-readiness-evidence.md`](./execution-forwarding-v1-readiness-evidence.md)、[`operations-readiness-evidence.md`](./operations-readiness-evidence.md) 和 [`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)。 | Local service process manager、真实 OS 通知、native tray/menu、真实 package icon/signing 尚未实现；Desktop 不应提前维护第二状态源、process control、notification state、tray state、AreaMatrix shim editing、shim apply command、status projection apply、execution cutover apply、Execution Forwarding v1 apply/rollback、task-loop forwarding、support export、migration apply、telemetry upload、release package/approval/rollout/publish、secret resolve、remote Team Console 或 worker 调度。 |
| v1.0 Stable Platform | preview_only | backup manifest、restore dry-run、audit coverage、permission doctor、artifact integrity、artifact archive preview、conformance、release readiness/remediation/acceptance/exception/final/package/distribution/publish/rollout preview 链已有测试和 smoke-local / safe fixture smoke；conformance 现在包含 `profile_item_state_contract`、`profile_transition_contract`、`profile_hard_rule_contract`、`profile_permission_policy_contract`、`profile_artifact_policy_contract`、`profile_cutover_policy_contract`、`project_config_policy`、`plugin_seed_catalog_contract`、`plugin_manifest_draft_contract` 和 `plugin_no_execution_boundary`，会只读校验 AreaMatrix item states、transition 顺序、required gate、hard rules、permission write guard、artifact policy、cutover policy、active `areaflow.yaml` snapshot，以及 plugin / marketplace built-in / seed metadata、manifest draft required fields 和未知 plugin execution v1.x 延后边界；security boundary readiness 已提供 `GET /api/v1/security/boundary-readiness` 和 `areaflow security boundary-readiness --json`，并证明 auth/team/token/secret/remote worker/budget/integration/ops 高风险开口保持关闭；operations readiness 已提供 `GET /api/v1/ops/readiness`、`GET /api/v1/ops/support-bundle-preview`、`GET /api/v1/ops/migration-ledger-readiness`、`areaflow ops readiness --json`、`areaflow ops smoke-proof record` 和 `areaflow support bundle-preview --json`，并证明 support bundle metadata-only、telemetry local-only、migration ledger、migration apply/DB write/support export/managed ops 均保持关闭或受控；completion audit 已提供 `GET /api/v1/completion-audit`、`GET /api/v1/completion-audit/snapshot-readiness`、`areaflow completion audit --json`、`areaflow completion audit-snapshot record`、`areaflow completion audit-snapshot readiness`、`areaflow completion source-alignment-proof record`、`areaflow completion task-matrix-proof record`、`areaflow completion validation-proof record`、`areaflow completion archive-proof record`、`areaflow completion shim-retirement-proof record`、`areaflow completion backup-restore-proof record`、`areaflow completion release-packaging-proof record`、`areaflow completion security-closure-proof record` 和 `areaflow completion protected-path-proof record`，并在 E1-E9 缺证据时返回 incomplete/blocked，且 snapshot command 只在 audit complete 后记录 sealed evidence；release_candidate snapshot 必须提供非 fixture 的 evidence URI、summary、`review_decision=approved`、非空 `reviewed_by` 和有效 RFC3339 `reviewed_at`，且不运行 smoke 或写被管理项目；阶段合同见 [`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)，明确 release final gate、package preview、Web/Desktop 展示或 smoke 通过不能单独声明 100%；release final/evidence/package/distribution/publish/approval/rollout preview chain 已在 Web/Desktop 中作为只读观察面展示。最近 safe fixture、Web smoke、security boundary、operations readiness 和 completion audit 证据见 [`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)、[`security-boundary-readiness-evidence.md`](./security-boundary-readiness-evidence.md)、[`operations-readiness-evidence.md`](./operations-readiness-evidence.md) 和 [`completion-audit-evidence.md`](./completion-audit-evidence.md)。 | 真实 restore apply、artifact copy/archive/delete/GC、release package creation、publish/tag/sign/upload、release exception write path、release approval/rollout apply、plugin install/enable/execute/network/secret/project write、auth/team enforcement、secret resolve、remote worker credential、webhook/external API apply、budget/quota enforcement 和 managed ops 均未打开；Source Alignment、Task Matrix、Validation、Archive、Shim Retirement、Backup Restore、Release Packaging、Security Closure proof record 和 completion audit snapshot 只记录外部审查事实或已完成 audit 的 sealed identity，不能代替真实 source review、task closure review、validation execution、execution cutover、旧 runner 退役、backup/restore/artifact preview review、release final/package/publish/rollout preview review、permission doctor、audit coverage 或 project isolation smoke，也不会修改 AreaMatrix；full support export 和真实 release candidate / AreaMatrix cutover closure evidence 仍需补齐。 |

### Phase Matrix Notes

- `execution-cutover-readiness` 已作为只读 readiness bundle 暴露到 CLI/API，并由 `smoke-local.sh` /
  `smoke-v1-stable-fixture.sh` 断言。它聚合 import/mirror/shadow、authoring cutover、compatibility shim、
  worker lease/run control、fixture execution、read-only verify、approved artifact write、fixture project write
  和 fixture/temp generated-only write evidence，但仍保持 `execution_cutover_apply_open=false`、
  `task_loop_run_forwarded=false`、`project_write_attempted=false`、`execution_write_attempted=false`。
  这证明 execution cutover 的 go/no-go 可被查询，不证明真实 AreaMatrix execution cutover 已打开。最近证据见
  [`execution-cutover-readiness-evidence.md`](./execution-cutover-readiness-evidence.md)。
- `Execution Forwarding v1` 已在迁移和 execution 文档中定义为第一版真实 cutover 子任务：只允许
  `./task-loop run` 转发 read-only verify、doctor/readiness、artifact evidence、status/projection validation
  和 release/readiness check 类任务，不打开 source write、generated retained write、repair、checkpoint、
  engine、secret、network、publish 或 restore。当前已实现只读 readiness / apply-preview / rollback-preview API/CLI 和 focused PostgreSQL smoke 入口：
  `GET /api/v1/projects/{project}/execution-forwarding-v1-readiness`、
  `GET /api/v1/projects/{project}/execution-forwarding-v1-apply-preview`、
  `GET /api/v1/projects/{project}/execution-forwarding-v1-apply-packet`、
  `GET /api/v1/projects/{project}/execution-forwarding-v1-apply-gate`、
  `GET /api/v1/projects/{project}/execution-forwarding-v1-rollback-preview`，以及对应 CLI。它只展示 allowed
  scope、消费 read-only verify / AreaFlow-owned artifact evidence / 同项目 clean protected-path proof、列出 future
  apply packet fields、required proof facts、explicit approval、shim 前置、rollback 缺口、fail-closed steps 和
  reopen conditions；受保护 apply Command API/CLI 已实现但当前只会在 gate 未通过时记录 blocked/denied
  command、event 和 audit；真实 forwarding smoke、rollback apply、真实 AreaMatrix legacy non-write proof、
  AreaMatrix read-only shim 和 `./task-loop run` forwarding 尚未实现。

## Current Cross-cutting Proof

当前通用验证基线：

```bash
go test ./...
go build ./cmd/areaflow
cd web && npm run build
git diff --check -- .
```

最近一次通用验证在 2026-07-02 通过：

```text
go test ./... PASS
go build ./cmd/areaflow PASS
cd web && npm run build PASS
git diff --check -- . PASS
```

当前安全 PostgreSQL smoke 入口：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-fixture.sh
```

`scripts/smoke-fixture.sh` 会创建临时 AreaMatrix-like project root、临时 artifact store 和临时
`areaflow.yaml`，只覆盖 M0 / v0.1-v0.2 的 read-only dogfood proof。它只允许带完整 apply gate packet 的
`status-projection-apply` 写 fixture root 下的 `.areaflow/status.json`。默认运行不读取真实
`/Users/as/Ai-Project/project/AreaMatrix`；若需要额外做真实 AreaMatrix projection 指纹守护，必须显式设置
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。

真实 AreaMatrix 只读 smoke 入口：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-areamatrix-readonly.sh
```

`scripts/smoke-areamatrix-readonly.sh` 使用真实
`/Users/as/Ai-Project/project/AreaMatrix` 作为读取对象，只运行
`project add/import/doctor/summary/readiness/import-diff/verify-bundle/status-projection-authorization/status-projection-apply-packet/status-projection-apply-gate`，不运行
`project export-status`、不传 `--allow-native`、不执行 task-loop、runner 或 worker。该脚本会比较真实
`.areaflow/status.json` 与 `workflow/README.md` 的前后指纹，确保只读 dogfood 没有改动 AreaMatrix
projection 文件。

真实 AreaMatrix 只读 smoke 在 2026-07-04 13:42 CST 通过，使用临时
`areaflow_smoke_20260704134250_46786` PostgreSQL 数据库并在结束后清理。运行中
`status-projection-authorization` 识别当时真实 `.areaflow/status.json` 为 legacy schema，`status-projection-apply-packet`
生成待审批 CLI/API packet 但不执行，`status-projection-apply-gate` 在缺失 packet 时返回 `blocked/no_go`，三者都保持
`apply_open=false`、`project_write_attempted=false`、`execution_write_attempted=false`、
`engine_call_attempted=false`、`command_request_created=false` 和 `status_projection_written=false`，并证明
preview/gate 前后 status projection apply command / status_projections 行数不变。运行后对
`.areaflow/status.json`、`workflow/README.md` 和 shim planned files 的 `git status --short` spot check 无输出。

v0.4 compatibility fixture smoke 入口：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-compatibility-fixture.sh
```

`scripts/smoke-compatibility-fixture.sh` 创建临时 AreaMatrix-like project root 和临时 artifact store，
覆盖 compatibility contract、blocked cutover readiness、ready cutover readiness、
`cutover_readiness_gate`、`cutover_apply_attempted=false` 和
`execution_write_attempted=false`。它只允许写 fixture root 下的 `.areaflow/status.json`。默认运行不读取真实
AreaMatrix；若需要额外比较真实 AreaMatrix `.areaflow/status.json` 与 `workflow/README.md` 的前后指纹，
必须显式设置 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。

v0.5 runner preview focused smoke 证据：

```text
临时 PostgreSQL DB + authored workflow version + runner.preview dry-run
```

最近结果见 [`runner-preview-evidence.md`](./runner-preview-evidence.md)。该 focused smoke 证明
`runner.preview` command request completed、idempotent replay、risk gate blocker、run/task/attempt/artifact/event/audit
和 artifact hash/size 校验闭环，并要求 persisted command response 记录 `area_matrix_write_attempted=false`。

v0.5 run control focused smoke 证据：

```text
临时 PostgreSQL DB + dry-run run fixture + run.start / run.drain / run.cancel
```

最近结果见 [`run-control-evidence.md`](./run-control-evidence.md)。该 focused smoke 证明
`run.start` / `run.drain` / `run.cancel` command request completed、idempotent replay、non-dry-run denial、
event/audit、`lease_count=0` 和 no project / execution / engine / task / worker / command / secret / network
attempts，并要求 persisted command response 记录 `area_matrix_write_attempted=false`。

v0.6 worker lease focused smoke 证据：

```text
临时 PostgreSQL DB + worker register / heartbeat + lease acquire / release / recover + capability denial
```

最近结果见 [`worker-lease-evidence.md`](./worker-lease-evidence.md)。该 focused smoke 证明
`worker.register` / `worker.heartbeat` / `lease.acquire` idempotent replay、`lease.release`、expired lease
recovery、capability denied 不创建 lease/attempt/artifact，以及 worker lifecycle / lease command response 中
no project / execution / engine / command / secret / network / attempt / artifact / worker_run_once attempts。

v0.6 execution approval gate focused smoke 证据：

```text
临时 PostgreSQL DB + dry-run runner preview + online worker + CLI/API execution approval gate
```

最近结果见 [`execution-approval-gate-evidence.md`](./execution-approval-gate-evidence.md)。该 focused smoke
证明 dry-run preview run 被 execution approval gate blocked，`read_only_boundary` pass，CLI/API 都返回
no project / execution / engine / command / secret / network / task claim / worker start / attempt / artifact
attempts，并且重复读取 gate 不改变 `command_requests`、runs、run_tasks、run_attempts、artifacts、leases
或 worker_heartbeats 计数。

v0.6 fixture execution apply focused smoke 证据：

```text
临时 PostgreSQL DB + approval-gated fixture execution run + worker fixture-execute
```

最近结果见 [`fixture-execution-apply-evidence.md`](./fixture-execution-apply-evidence.md)。该 focused smoke
证明 `execution gate pass -> worker claim / lease -> attempt -> evidence artifact -> run_task passed -> run passed`
闭环成立，`worker.fixture_execute` idempotent replay 返回 `created=false`，并且 command response 保持
no project / execution / engine / command / secret / network attempts。该 smoke 只使用临时 fixture
project root 和临时 AreaFlow artifact store，不触碰真实 AreaMatrix。

v0.6 read-only verify focused smoke 证据：

```text
临时 PostgreSQL DB + approval-gated read-only verify run + worker read-only-verify
```

最近结果见 [`read-only-verify-evidence.md`](./read-only-verify-evidence.md)。该 focused smoke 证明
`execution gate pass -> worker claim / lease -> allowlisted project file read -> evidence artifact
-> run_task verified -> run verified` 闭环成立，`worker.read_only_verify` idempotent replay 返回
`created=false`，target file sha256 与本地文件一致，并且 command response 保持 no project write /
execution write / engine / command / secret / network attempts。该 smoke 只使用临时 fixture project root
和临时 AreaFlow artifact store，不触碰真实 AreaMatrix。

v0.6 approved artifact write focused smoke 证据：

```text
临时 PostgreSQL DB + approval-gated approved artifact write run + worker approved-artifact-write
```

最近结果见 [`approved-artifact-write-evidence.md`](./approved-artifact-write-evidence.md)。该 focused smoke
证明 `execution gate pass -> worker claim / lease -> AreaFlow-owned artifact write -> attempt
-> run_task artifact_written -> run artifact_written` 闭环成立，`run.approved_artifact_write_queue` 和
`worker.approved_artifact_write` idempotent replay 返回 `created=false`，并且 command response 保持
no project read/write / execution write / engine / command / secret / network attempts。该 smoke 只使用
临时 fixture project root 和临时 AreaFlow artifact store，不触碰真实 AreaMatrix。

v0.6 execution plan preview focused 证据：

```text
project/api/app focused tests + read-only execution plan preview
```

最近结果见 [`execution-plan-preview-evidence.md`](./execution-plan-preview-evidence.md)。该 focused 证据证明
`GET /api/v1/runs/{run_id}/execution-plan` 和 `areaflow run execution-plan <run-id>` 能只读展示
`execution_approval_gate`、`copy`、`verify`、`approved_artifact_write`、`checkpoint` 和 `repair`。
其中 `approved_artifact_write` 可在 gate pass 时为 `ready`；`copy`、`checkpoint` 和 `repair` 仍保持
blocked / waiting，并暴露 `managed_project_write_not_open`、`engine_execution_not_open`、
`checkpoint_apply_not_implemented` 等 blocker。该 preview 保持 no project read/write、no execution write、
no artifact write、no engine、no command、no secret、no network、no task claim、no worker start、
no attempt 和 no artifact creation safety facts。

v0.6 project write design gate focused 证据：

```text
project/api/app focused tests + read-only approved project write design gate
```

最近结果见 [`project-write-design-gate-evidence.md`](./project-write-design-gate-evidence.md)。该 focused 证据证明
`GET /api/v1/runs/{run_id}/project-write-design-gate` 和
`areaflow run project-write-design-gate <run-id>` 能只读展示 approved project write 的 write-set fields、
unsupported operations、apply sequence、copy/verify/repair/checkpoint split、rollback contract 和
safety facts。即使返回 `status=ready`，`project_write_apply_open=false`，并保持 no project read/write、
no execution write、no artifact write、no engine、no command、no secret、no network、no task claim、
no worker start、no attempt 和 no artifact creation safety facts。

真实本机 preview 链入口：

```bash
AREAFLOW_DATABASE_URL=... \
AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1 \
AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1 \
AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1 \
./scripts/smoke-local.sh
```

`scripts/smoke-local.sh` 覆盖 v0.1-v1.0 的主要 CLI path，但只有在提供真实
`AREAFLOW_DATABASE_URL` 时运行。它默认使用 `examples/areamatrix/areaflow.yaml`，该配置指向真实
AreaMatrix root。解析到真实 AreaMatrix root 时，它会在 `project add/import/summary/doctor` 前拒绝读取，
除非显式设置 `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1`。它还会通过
`status-projection-authorization -> status-projection-apply-packet -> status-projection-apply` 写
configured project root 下的 `.areaflow/status.json`，因此必须显式设置
`AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1`；如果 configured project root 解析为真实 AreaMatrix root，写入还
必须额外设置 `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1`。上面的真实 AreaMatrix 命令只能在单独授权
后运行；该脚本不能作为未经授权的只读 dogfood 证据，真实 AreaMatrix smoke 需要单独确认。

安全 v1.0 stable fixture 入口：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-v1-stable-fixture.sh
```

`scripts/smoke-v1-stable-fixture.sh` 创建临时 AreaMatrix-like project root、临时 artifact store 和临时
`areaflow.yaml`，把 `smoke-local.sh` 指向 fixture 配置，覆盖 v0.1-v1.0 的长链 smoke，包括 operations
readiness、support bundle metadata-only preview 和 migration ledger readiness CLI 断言。默认不读取真实
AreaMatrix；如需额外比较真实 `.areaflow/status.json` 与 `workflow/README.md` 的前后指纹，必须显式设置
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。
最近证据见 [`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md)。

completion proof focused Docker 入口：

```bash
make smoke-docker-completion-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-completion-proof.sh`。它使用临时 AreaMatrix-like project root，验证
`completion archive-proof record`、`completion shim-retirement-proof record` 和
`completion execution-cutover-proof record` 的 complete fact 校验、Execution Forwarding v1 scope binding
校验、Archive/Shim structured scope binding 与 deterministic current binding hash 校验、proof EventID>0 门禁、
release-candidate proof URI / approved review metadata、loose proof fail-closed、idempotent replay、
event/audit/command response 持久化，以及 completion audit 消费 proof metadata 后仍被真实 AreaMatrix identity gate
阻断；它不写真实 AreaMatrix、不转发 `./task-loop run`，也不执行 shim retirement。

execution cutover proof focused Docker 入口：

```bash
make smoke-docker-execution-cutover-proof
```

该入口复用 `scripts/smoke-completion-proof.sh` 的 E4 fixture，但使用独立 project key。它验证
受控 `completion execution-cutover-proof record` 可以与 archive/shim proof 一起封存 E4 proof metadata，但 complete
proof 必须绑定 `execution_forwarding_v1_read_only_evidence_only` scope、allowed task types、forbidden actions、
fail-closed rollback、release-candidate proof URI 和 approved review metadata；临时 fixture 仍不能让 completion
audit 移除真实 AreaMatrix E4 blockers，同时保持 project write、execution write、task-loop forwarding、engine call、
legacy progress/log/checkpoint 写入和 AreaMatrix protected path touch 全部为 false。

completion audit full proof Docker 入口：

```bash
make smoke-docker-completion-audit-full-proof
```

该入口在隔离 PostgreSQL 数据库和临时 `areamatrix` fixture project 中记录 E1-E9 全部受控 proof、
operations smoke proof 和 clean protected path proof，然后要求 `areaflow completion audit --json`
因 fixture project identity 返回 blocked，并验证 `areaflow completion audit-snapshot record` 会因当前 audit
不是 complete 而拒绝记录 fixture snapshot。Completion audit 只消费目标
project key 为 `areamatrix` 的 proof；其他 project key 的 proof 会留下 `*_proof_project_mismatch`
blocker。该入口证明 completion audit 能消费当前完整 fixture 证据集并被真实 AreaMatrix identity gate 拦住，而不是记录可被误读的快照；它仍不打开真实 AreaMatrix
execution cutover apply、legacy runner retirement、release publish、restore apply、support export、
remote telemetry 或 managed ops。

completion audit release candidate snapshot focused Docker 入口：

```bash
make smoke-docker-completion-audit-release-candidate-snapshot
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-completion-audit-release-candidate-snapshot.sh`。它使用临时 `areamatrix` fixture project
和 synthetic reviewed evidence URI 记录 E1-E9 proof，随后验证
`evidence_class=release_candidate` snapshot record 因真实 AreaMatrix project identity 缺失而 fail closed，
且 snapshot readiness 返回 `completion_audit_snapshot_real_project_identity_missing`。该入口证明临时
fixture 不能被重标为真实 release candidate；它仍不代表真实 AreaMatrix release-candidate closure、
execution cutover、release publish、restore apply、support export、remote telemetry 或 managed ops 已打开。

operations proof focused Docker 入口：

```bash
make smoke-docker-operations-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-operations-proof.sh`。它使用临时 `areamatrix` fixture project，验证
`ops smoke-proof record --key local_ops_smoke` 的 idempotent replay、event/audit/command response
持久化，以及 `ops readiness` 和 completion audit E7 消费 proof 后关闭
`fresh_local_ops_smoke_missing`；它不运行长链 smoke、不控制 service process、不导出 support bundle、
不应用 migration、不上传 telemetry，也不写真实 AreaMatrix。

validation proof focused Docker 入口：

```bash
make smoke-docker-validation-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-validation-proof.sh`。它使用临时 AreaMatrix-like project root，验证
`completion validation-proof record` 的 complete fact 校验、idempotent replay、event/audit/command
response 持久化，以及 completion audit 消费 proof 后关闭 E3 的 `fresh_validation_proof_missing`；
它不运行测试、构建、browser smoke、PostgreSQL smoke，也不写真实 AreaMatrix。

source alignment proof focused Docker 入口：

```bash
make smoke-docker-source-alignment-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-source-alignment-proof.sh`。它使用临时 AreaMatrix-like project root，验证
`completion source-alignment-proof record` 的 complete fact 校验、idempotent replay、event/audit/command
response 持久化、proof 自动绑定当前 AreaFlow E1 source path/hash/source-set hash，以及 completion audit
重新计算 current binding 后关闭 E1 的 `source_alignment_proof_missing`；它只读并 hash AreaFlow 源文件，
不改写文档、不运行 shell，也不写真实 AreaMatrix。

task matrix proof focused Docker 入口：

```bash
make smoke-docker-task-matrix-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-task-matrix-proof.sh`。它使用临时 AreaMatrix-like project root，验证
`completion task-matrix-proof record` 的 complete fact 校验、current backlog/status-audit binding、
idempotent replay、event/audit/command response 持久化，以及 completion audit 消费 proof 后关闭 E2 的
`task_matrix_proof_missing`；
它不扫描或改写 backlog/status 文件、不运行 shell，也不写真实 AreaMatrix。

backup restore proof focused Docker 入口：

```bash
make smoke-docker-backup-restore-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-backup-restore-proof.sh`。它使用临时 AreaMatrix-like project root，验证
`completion backup-restore-proof record` 的 complete fact 校验、backup manifest / restore plan /
artifact integrity / archive preview 输出绑定、idempotent replay、event/audit/command response 持久化，
以及 completion audit 消费 proof 后关闭 E6 的 `restore_dry_run_needs_attention` 和
`metadata_only_history_not_closed`；缺 manifest hash/status/count、restore plan status/hash/count、
artifact integrity status/count 或 archive preview status/count/no-write safety facts 的 complete proof
会 fail closed。该 smoke 只在临时项目内收集只读/preview 输出用于绑定，不执行 database restore、
不复制/删除/上传 artifact bytes、不运行 GC，也不写真实 AreaMatrix；completion audit 会重新运行当前
manifest/restore/integrity/archive preview 的只读 binding，并在 artifact 内容漂移导致 current integrity
变为 fail 时阻断 E6，同时断言该 audit 过程不新增 command/event/audit 行。archive preview 复核读取全部
项目 artifact metadata。manifest hash 仍作为 metadata 暴露并校验当前自洽，但 proof 记录自身会追加
command/event/audit 行，所以不以 pre-proof hash 等值作为完成条件。

security closure proof focused Docker 入口：

```bash
make smoke-docker-security-closure-proof
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-security-closure-proof.sh`。它使用临时 AreaMatrix-like project root，验证
`completion security-closure-proof record` 的 complete fact 校验、idempotent replay、event/audit/command
response 持久化，以及 completion audit 消费 proof 后关闭 E8 的 `project_isolation_smoke_missing` 和
`audit_gap_closure_missing`。complete proof 会绑定当前只读 security boundary readiness、permission
doctor 和 project-scoped audit coverage，completion audit 会重新计算该 binding 并在漂移时阻断 E8；它
不运行 shell 或 project isolation smoke，不读取 secret、不改 authorization、不发放 remote worker
credential，也不写真实 AreaMatrix。

managed generated write focused Docker 入口：

```bash
make smoke-docker-managed-generated-write
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，但只运行
`scripts/smoke-managed-generated-write.sh`。它使用临时 fixture/temp project，验证 generated-only
write/verify/rollback drill 和 non-fixture denial。默认不读取真实 AreaMatrix projection 文件；若需要额外
比较 `.areaflow/status.json` 与 `workflow/README.md` 的前后指纹，必须显式设置
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。它不运行 `smoke-local.sh`，也不写真实 AreaMatrix。

真实 AreaMatrix 只读 Docker 入口：

```bash
make smoke-docker-areamatrix-readonly
```

该入口复用 `scripts/smoke-docker.sh` 的 PostgreSQL 启动和 readiness 等待，只运行
`scripts/smoke-areamatrix-readonly.sh`。它读取真实 AreaMatrix root，覆盖 import、doctor、summary、
readiness、import-diff、verify-bundle、shim-preview 和 shim-readiness，并校验真实
`.areaflow/status.json` 与 `workflow/README.md` 指纹不变；它不执行 native doctor、不写 status
projection、不落 AreaMatrix shim。

AF-V04 compatibility shim 授权前预检入口：

```bash
make smoke-docker-shim-authorization-preflight
```

该入口复用同一条真实 AreaMatrix read-only smoke，但命名为授权前预检，用于证明 status projection
authorization、apply-packet/gate、shim authorization、shim apply-packet/gate、required preflight、rollback
scope 和 no-write safety facts 当前可复验；它不授权真实 AreaMatrix 编辑，也不写 `.areaflow/status.json`、
shim 文件或 `workflow/README.md`。

Web local-service browser smoke 入口：

```bash
AREAFLOW_DATABASE_URL=... ./scripts/smoke-web.sh
make smoke-docker-web
make smoke-docker-web-areamatrix-readonly
```

`scripts/smoke-web.sh` 默认创建临时 AreaMatrix-like fixture project，再用该 fixture seed
runner preview、worker dry-run、approval、audit、worker pool 和 schedule preview 数据。随后启动本地
AreaFlow API 与 Vite Web，并用 Playwright 打开真实页面。`smoke-web-check.mjs` 会确认 Dashboard
只发 `/api/v1` GET / SSE 请求，且不会绕到非 v1 `/api` 路由；覆盖 project、summary、readiness、version、run、artifact、residual、
approval、worker、audit、worker pool、schedule preview、shim authorization、execution cutover readiness 和
Execution Forwarding v1 readiness/apply-preview/rollback-preview、release final/evidence/package/distribution/
publish/approval/rollout preview 面板。

`scripts/smoke-web-areamatrix-readonly.sh` 复用同一 Web smoke，但切换到真实 `areamatrix` 项目和
`/Users/as/Ai-Project/project/AreaMatrix` root。它只做 PostgreSQL 迁移、project add/import 和浏览器
GET 观察，跳过 fixture 的 status projection apply、workflow seed、worker run 和 ops proof 记录；检查器会
断言 status projection authorization / apply-packet / apply-gate GET 响应和页面都包含
`授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json`，断言 release / completion 面板仍显示
`real_100=blocked` 以及 Package A、read-only shim、execution cutover、archive、shim retirement blockers，且浏览器前后
AreaMatrix protected path 指纹和 AreaFlow command/event/audit/gate/status projection/approval/run/task/attempt/
artifact/worker/heartbeat/lease 计数不变。

最近一次 Web smoke 在 2026-07-02 21:20 CST 通过，使用临时
`af_web_shim_auth_20260702212028_62899` PostgreSQL 数据库并在结束后清理。该 smoke 继续证明
Dashboard 未发出非 GET `/api/v1` 请求；历史上它曾暴露 Schedule Preview 未渲染
`engine_profile_disabled` blocker，修复 Web 展示后已重跑通过；本次也证明 Shim Authorization 面板
会请求并渲染 `GET /api/v1/projects/{project}/shim-authorization`。

最近一次 Web smoke 在 2026-07-02 21:38 CST 通过，使用临时
`af_web_release_final_20260702213826_27166` PostgreSQL 数据库并在结束后清理，residual database count 为 0。
该 smoke 继续证明 Dashboard 未发出非 GET `/api/v1` 请求，并证明 Release Final Gate 面板会请求并渲染
`GET /api/v1/release/final-gate`、`read_only_release_final_gate`、`final_gate:release_readiness`、
`create_release_package` 和 `apply_release`。

最近一次 Web smoke 在 2026-07-02 21:45 CST 通过，使用临时
`af_web_release_preview_20260702214511_49410` PostgreSQL 数据库并在结束后清理，residual database count 为 0。
该 smoke 继续证明 Dashboard 未发出非 GET `/api/v1` 请求，并证明 release preview 面板组会请求并渲染
`GET /api/v1/release/evidence-bundle`、`GET /api/v1/release/package-preview`、
`GET /api/v1/release/publish-gate`、`read_only_release_evidence_bundle`、
`read_only_release_package_preview`、`read_only_release_publish_gate`、`compress_artifacts`、
`create_git_tag` 和 `publish_release`。

最近一次 Web smoke 在 2026-07-02 21:54 CST 通过，使用临时
`af_web_release_rollout_20260702215419_81144` PostgreSQL 数据库并在结束后清理，residual database count 为 0。
该 smoke 继续证明 Dashboard 未发出非 GET `/api/v1` 请求，并证明 release rollout preview 面板组会请求并渲染
`GET /api/v1/release/distribution-preview`、`GET /api/v1/release/publish-approval-preview`、
`GET /api/v1/release/rollout-plan-preview`、`read_only_release_distribution_preview`、
`read_only_release_publish_approval_preview`、`read_only_release_rollout_plan_preview`、
`distribution:git_release`、`publish_approval:publish_gate`、`rollout_plan:publish_approval`、
`approve_release` 和 `create_rollout`。

最近一次 Web smoke 在 2026-07-02 23:06 CST 通过，使用临时
`af_web_project_guard_*` PostgreSQL 数据库并在结束后清理，residual database count 为 0。
该 smoke 继续证明 Dashboard 未发出非 GET `/api/v1` 请求，并新增证明 Web 会点击 run timeline，
随后通过 `GET /api/v1/runs/{run_id}?project_key={fixture_project}` 读取 run detail，实际使用
global run ID route 的 `project_key` visibility guard。

## Hard Gaps Before 100%

- AreaMatrix 真实 dogfood cutover 仍未执行；当前只是 import/mirror/shadow/cutover-readiness 能力。
- AreaFlow 侧 `shim-preview` 已能只读输出 AreaMatrix shim planned files、command mapping、安全禁区和验证命令；
  `shim-readiness` 已能作为 go/no-go 门禁，`shim-authorization` 已能机器可读和人类可读输出 allowed files、
  forbidden paths/actions、preflight、post-edit verification、rollback scope 和 safety facts，`shim-readiness-evidence` 已能记录真实只读 smoke 与 dirty
  worktree review 前置证据；即使这些前置 evidence 已记录，缺少显式编辑授权时仍保持 blocked；
  `smoke-areamatrix-readonly.sh` 已覆盖真实 AreaMatrix 上的 shim-preview / shim-readiness 只读查询，并校验
  `.areaflow/status.json` 与 `workflow/README.md` 指纹不变；2026-07-04 的真实 AreaMatrix 只读 smoke 和
  compatibility fixture smoke 也验证了 `shim-authorization` 普通 CLI 文本包含 required preflight、
  post-edit verification 和 rollback scope；required preflight 已包含 AreaMatrix 保护路径专项
  `git status --short -- ...` 检查，post-edit verification 也包含同一组保护路径复查；
  2026-07-11 Package B readiness / dirty review / authorization packet scripts 已能在 AreaFlow-only 范围内生成
  read-only shim 编辑授权包，要求当前 stable status projection、protected/worktree dirty output hash 精确复核和
  精确授权语句；该预检不写 AreaMatrix，也不授权 `.areaflow/status.json`、`workflow/versions/**` 或
  `./task-loop run` forwarding；
  AreaMatrix 兼容 shim 仍未在 AreaMatrix 仓库落地或验证；最小计划见
  [`../migration/areamatrix-compatibility-shim-plan.md`](../migrations/areamatrix-compatibility-shim-plan.md)。
- Authoring cutover 与 execution cutover 已在迁移文档中分离；`Execution Forwarding v1` 已定义为第一版
  只读/evidence forwarding 子任务，并已有只读 readiness / apply-preview / apply-packet / apply-gate /
  command-preview / rollback-preview API/CLI、受保护 apply Command API/CLI 和 focused PostgreSQL smoke
  入口；但真实 forwarding smoke、rollback、AreaMatrix read-only shim 和 `./task-loop run` forwarding 仍未落地。
- `workflow/README.md` 受控区块写入仍未打开。
- 真实 runner execution、Codex CLI execution、copy/verify/repair/checkpoint 还未启用；execution approval gate
  仍只是只读 go/no-go。v0.6i 已打开 fixture-only execution apply，v0.6j 已打开 allowlisted read-only
  verify，v0.6k 已打开 AreaFlow-owned approved artifact write，v0.6l 已打开只读 execution plan preview；
  这些路径会创建受限 lease/attempt/artifact 或只读展示下一步 execution blockers，但不代表真实 engine
  调用、项目文件写入、copy/repair/checkpoint 或 AreaMatrix execution cutover 已打开。
- Web dashboard 已有真实 local service + browser smoke；Web 写操作和 AreaMatrix shim editing 仍应保持关闭直到 Command API 风险面稳定并获得显式授权。
- Desktop Tauri shell scaffold 和只读 service / notification / tray / shim authorization gate / shim apply review 已实现；local service process
  manager、真实 OS 通知、native tray/menu、package icon/signing 仍未实现。
- 多项目真实调度、远程 worker、team/multi-user auth、API token 使用仍未启用。
- Secret / engine 只做引用和 readiness；真实 secret resolve、engine execution、worker secret context 属于 v1.x R4 能力。
- Restore apply、release exception apply、release publish 均只是 preview/gate，不是可执行能力；artifact archive 目前只有 metadata-only preview command，真实 retention-aware copy/delete/GC 尚未打开。
- Completion audit 只读 API/CLI report 已实现，但当前仍会因 E1-E9 缺证据返回 incomplete/blocked；
  release final gate、package preview、Web/Desktop 展示、绿色测试或 smoke 通过都不能单独声明 0-100% 完成。
- Operations / deployment / observability 边界已成文，且只读 operations readiness、support bundle metadata-only
  preview、migration ledger readiness 已作为 API/CLI 能力打开并进入 v1 stable fixture / Web smoke 路径；completion
  audit 可消费 `ops.smoke_proof.recorded` proof 输入和 `000011_v1_migration_ledger.sql` 的 full ledger phase
  evidence；managed upgrade/rollback 和 full support export 仍未打开。service status 不能解释为 process control、
  remote ops control 或 managed upgrade。
- Release exception 只允许 `metadata_only_history`、`future_only_gap` 和 `archive_exception` 三类显式接受；
  permission policy fail、adapter/profile conformance fail、backup broken、local artifact hash mismatch、
  secret 泄露风险、Command API 幂等/审计缺失和无 rollback 的真实写入不能靠 exception 放行。
- AreaFlow self dogfood 不能替代 AreaMatrix dogfood；v0.1-v0.2 不依赖自管理，v0.3 以后最多只读，
  v0.5 以后最多 dry-run / artifact-only，v1.0 稳定后才可作为平台自身管理证据。
- Workspace / environment 在 v1.0 前是刻意后置实体；当前只能通过 team/project grouping、
  project connections、worker kind、engine profile 和 scheduling metadata 表达，不应新增复杂表来绕过
  `project_key` 隔离审计。
- Plugin marketplace / third-party plugin execution 仍是 v1.x 后续能力。
- v1.x 高风险能力已按 real generated-only rollback beta -> retained apply -> manual patch artifact
  -> human-applied source evidence -> source write beta -> checkpoint apply -> repair plan/apply
  -> no-secret engine execution -> secret resolve -> remote worker -> restore apply
  -> release exception real write -> publish apply -> third-party plugin execution -> external integrations/webhooks
  -> team console -> object artifact store -> budget/quota -> managed ops/upgrade/support export 排序；任一步都不能用
  v1.0 release gate 或上一步 smoke 替代本步 approval、rollback / remediation 和 audit 证据。

## Documented Foundation Decisions Not Yet Fully Implemented

以下内容已进入正式设计文档，但不能当作实现完成证据：

| 议题 | 文档位置 | 当前实现状态 |
|---|---|---|
| Runner / Worker / Engine / Project Adapter 分层 | `docs/architecture/execution-model.md` | Runner preview 和 worker dry-run 已有；真实 engine / project write 未打开。 |
| `run_task` 状态机与 recovery 语义 | `docs/architecture/execution-model.md` | run start/drain/cancel 的 dry-run DB-only 控制面已进入 Command API；worker lease recovery 已有；fixture-only execution apply 已能将 approval-gated run_task/run 推进到 passed 并留下 lease/attempt/artifact evidence；read-only verify 已能将 approval-gated run_task/run 推进到 verified，并保存 allowlisted target file 的 hash/size evidence；approved artifact write 已能将 approval-gated run_task/run 推进到 artifact_written，并写 AreaFlow-owned report artifact；execution plan preview 已能只读展示 copy、verify、approved artifact write、checkpoint 和 repair 的 blockers/safety facts。完整 worker 协作式 cancel/drain、真实 engine execution recovery、copy/verify/repair 和 checkpoint evidence 仍需实现。 |
| 多项目调度匹配条件与远程 worker 边界 | `docs/architecture/execution-model.md`、`docs/architecture/api-surface.md`、`docs/architecture/auth-team-secret-boundary.md` | schedule-preview 已有；auth/team/secret/remote worker R4 opening ladder 已成文；真实调度、远程 worker credential、token enforcement 未打开。 |
| Query API / Command API / SSE 唯一业务边界 | `docs/history/v1.0/contracts/phase-0-foundation-baseline.md`、`docs/adr/0006-platform-operating-boundary.md`、`docs/architecture/api-surface.md` | Web 只读路径已有 `scripts/smoke-web.sh` browser smoke 证明；`project.import`、`project.cutover.apply`、`workflow.version.create`、`workflow.approval.record`、`runner.preview`、`run.fixture_queue`、`worker.fixture_execute`、`run.read_only_verify_queue`、`worker.read_only_verify`、`run.approved_artifact_write_queue`、`worker.approved_artifact_write`、`run.start`、`run.drain`、`run.cancel`、`artifact.archive.preview`、`project.status_projection.write`、`project.status_projection.apply`、`project.doctor.record`、`worker.register`、`worker.heartbeat`、`lease.acquire`、`lease.release`、`lease.recover` 已进入 `command_requests`。`project.import` 覆盖 metadata index 重建、import run、status snapshot 和 audit event；`project.cutover.apply` 覆盖 AreaFlow DB 内 authoring cutover、event/audit 写入，并保持 `project_write_attempted=false`、`execution_write_attempted=false`；`runner.preview` 覆盖 dry-run run/task/attempt/artifact/event/audit，并在 command response 中保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`；execution approval gate 是只读 Query API，不进入 `command_requests`，并保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`task_claimed=false`、`worker_started=false`、`attempt_created=false`、`artifact_created=false`；`run.fixture_queue` 创建非 dry-run fixture execution run/task 但保持 no project / execution / engine / command / secret / network attempts；`worker.fixture_execute` 在 approval gate 通过后创建 lease/attempt/artifact，并保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`area_flow_execution_state_written=true`；`run.read_only_verify_queue` 创建非 dry-run read-only verify run/task，但排队阶段保持 no project read/write / execution / engine / command / secret / network attempts；`worker.read_only_verify` 在 approval gate 通过后读取 allowlisted project file 并保存 hash/size evidence，保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`area_flow_execution_state_written=true`；`run.approved_artifact_write_queue` 创建非 dry-run approved artifact write run/task，但排队阶段保持 no project read/write / execution / engine / command / secret / network attempts；`worker.approved_artifact_write` 在 approval gate 通过后只写 AreaFlow-owned artifact store 和 PG evidence，保持 `project_read_attempted=false`、`project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`area_flow_artifact_written=true`、`area_flow_execution_state_written=true`；`project.status_projection.apply` 只写 `.areaflow/status.json` 并保持 `execution_write_attempted=false`、`engine_call_attempted=false`；run control command 保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`task_claimed=false`、`worker_started=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`；worker lifecycle command 保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`lease_created=false`、`attempt_created=false`、`artifact_created=false`、`worker_run_once=false`；lease command 保持 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`attempt_created=false`、`artifact_created=false`、`worker_run_once=false`；artifact archive preview 保持 `project_write_attempted=false`、`storage_write_attempted=false`、`artifact_delete_attempted=false`。 | 统一 command preview / apply / audit 模型仍需继续收敛到真实 archive/GC apply、真实 engine/project execution apply、远程 worker 凭证管理和 publish/restore apply。 |
| `status_projections` 作为长期投影模型 | `docs/history/v1.0/contracts/phase-0-foundation-baseline.md`、`docs/adr/0006-platform-operating-boundary.md`、`docs/architecture/data-model-v1.md` | `000010_v1_status_projections`、store/API/CLI 查询、`export-status` 兼容入口和 `project.status_projection.apply` 受保护 Command API 已落地；安全 fixture smoke 通过临时项目写 `.areaflow/status.json`，并检查真实 AreaMatrix status 未变化。真实 `workflow/README.md` 写入仍未打开。 |
| Artifact retention class、doctor/GC 状态 | `docs/product/platform-blueprint.md`、`docs/architecture/data-model-v1.md` | artifact integrity preview 和 `artifact.archive.preview` metadata-only command 已有。 | retention-aware copy/archive/delete/GC apply command 未打开。 |
| Artifact content API 边界 | `docs/architecture/api-surface.md`、`docs/architecture/data-model-v1.md` | AreaFlow-owned local artifact content API 已实现并校验 hash/size；project_reference 原文读取仍应保持 unavailable，除非后续显式 archive/copy command。 |
| Secret / engine readiness 与 worker secret context | `docs/architecture/security-permissions.md`、`docs/architecture/data-model-v1.md`、`docs/architecture/auth-team-secret-boundary.md` | readiness / blocked reason 只读路径已有；Codex CLI adapter preview 已证明不解析 secret、不调用 engine、不运行命令、不写项目文件；R4 secret resolve ladder 已定义 scoped binding、redaction、audit 和 rollback 要求。 | 真实 secret resolve、engine 调用、API token enforcement、team permission enforcement、remote worker credential 和 scoped worker secret context 未打开。 |
| Authoring cutover / execution cutover / shim retirement 分层 | `docs/migration/areamatrix-workflow-migration.md`、`docs/migration/areamatrix-execution-cutover-boundary.md`、`docs/architecture/project-config.md` | v0.4 readiness / compatibility preview / shim-preview / shim-readiness 已有；execution cutover 命令映射、protected paths 和 rollback 边界已有文档源事实；`Execution Forwarding v1` 已补为 backlog 子任务并已有只读 readiness / apply-preview / apply-packet / apply-gate API/CLI，限定第一版只转发 read-only / evidence 类任务，并可消费同项目 clean/authorized protected-path proof；apply-preview 已暴露 read-only / evidence forwarding target matrix、blocked target matrix 和 fail-closed response fields；apply-packet 已生成 readiness snapshot hash、approval scope、proof ids、idempotency key 和 audit correlation id；readiness 和 apply-preview 已能消费同项目 complete execution cutover proof 中的 rollback-specific facts 并关闭 rollback-to-read-only-shim 项；apply-gate 已能在 missing packet 或 complete packet but read-only shim blocked 时 fail closed；受保护 apply Command API/CLI 已落地，能记录 blocked/denied command、event 和 audit，并保持 run/task/attempt/artifact、legacy task-loop、project write、execution write、engine、secret、network 全部关闭；command-preview 已能对 allowed、blocked 和 unknown task type 返回只读 response preview，且实际 command/run/legacy/project/engine/network safety facts 保持 false；rollback preview 也可消费同一 proof，在 fixture 中关闭 `rollback_v1:proof_facts`，同时保持 reopen conditions blocked。 | AreaMatrix 仓库内 shim 修改、真实 forwarding smoke、真实 AreaMatrix legacy non-write proof、真实 rollback proof、Archive gate 和 Shim Retirement 未落地。 |
| Release final gate 与 exception 口径 | `docs/history/v1.0/contracts/phase-0-foundation-baseline.md`、`docs/architecture/api-surface.md` | release readiness / remediation / acceptance / final gate preview 链已有；真实 exception apply / package / publish 仍未打开。 |
| Completion audit 边界 | `docs/architecture/completion-audit-contract.md`、`docs/product/master-plan.md`、`docs/product/phase-backlog.md`、[`completion-audit-evidence.md`](./completion-audit-evidence.md) | `GET /api/v1/completion-audit` 和 `areaflow completion audit --json` 已实现，只读 report 会聚合 E1-E9 并在缺 protected path proof、AreaMatrix dogfood cutover 或 smoke evidence 时返回 incomplete/blocked；`real_100_breakdown` 已作为只读解释层透出，将 blockers 拆为精确授权、真实 AreaMatrix 写入/落地、AreaFlow-only 可继续和已完成 evidence 四类，同时 `claim_scope`、`not_real_100=true`、`evidence_only=true`、`status_alone_is_not_completion=true` 和 `release_candidate_decision` 防止外部只看 `status=complete/pass/ready`；这些字段不会降低 `real_100_status=blocked` 或替代真实 cutover；`areaflow completion audit-snapshot record` 已实现，只在当前 audit complete 后记录 sealed snapshot，持久化 audit hash、scope、release candidate label、evidence class、evidence URI 和 proof event IDs；release_candidate snapshot 必须提供无 fixture/mock/demo/sample/synthetic/testdata/placeholder/dummy/example marker 的 evidence URI、summary、`review_decision=approved`、非空 `reviewed_by` 和有效 RFC3339 `reviewed_at`，并保持不运行测试/smoke、不写 AreaMatrix、不创建 release package、不启动 worker；只读 readiness 会在最新 snapshot 仍为 fixture 或带这些 non-release markers 时返回 blocked；`areaflow completion source-alignment-proof record` 已提供受控 E1 proof input，且 completion audit 可消费 latest complete proof 关闭 source alignment blocker；`scripts/smoke-source-alignment-proof.sh` 已提供 focused PostgreSQL smoke，验证 Source Alignment proof 的 complete fact 校验、idempotent replay 和 completion audit consumption；`areaflow completion task-matrix-proof record` 已提供受控 E2 proof input，且 completion audit 可消费 latest complete proof 关闭 task matrix blocker；`scripts/smoke-task-matrix-proof.sh` 已提供 focused PostgreSQL smoke，验证 Task Matrix proof 的 complete fact 校验、idempotent replay 和 completion audit consumption；`areaflow completion validation-proof record` 已提供受控 E3 proof input，complete proof 必须绑定 reviewed validation command list、sha256 result hash、RFC3339 time window 和 scope，且 completion audit 可消费 latest complete proof 关闭 fresh validation blocker；`scripts/smoke-validation-proof.sh` 已提供 focused PostgreSQL smoke，验证 Validation proof 的 complete fact 校验、validation-output binding、idempotent replay 和 completion audit consumption；`areaflow completion archive-proof record`、`areaflow completion shim-retirement-proof record` 和 `areaflow completion execution-cutover-proof record` 已提供受控 E4 proof input，complete proof 必须带 release-candidate URI、summary、approved review metadata 和 scope binding，且 completion audit 只有在真实 AreaMatrix identity 通过时才移除 E4 real blockers；`scripts/smoke-completion-proof.sh` 已提供 focused PostgreSQL smoke，验证 Archive/Shim/Execution proof 的 complete fact 校验、idempotent replay、review evidence 和 completion audit blocked-by-identity behavior；`areaflow ops smoke-proof record` 可作为 E7 operations readiness proof input，且 completion audit 可消费 readiness 中的 latest pass proof；`scripts/smoke-operations-proof.sh` 已提供 focused PostgreSQL smoke，验证 `local_ops_smoke` proof 的 idempotent replay 和 completion audit consumption；`areaflow completion security-closure-proof record` 已提供受控 E8 proof input，且 completion audit 可消费 latest complete proof 关闭 project isolation / audit gap blockers，但 forbidden security opening 仍会保持 blocked；`scripts/smoke-security-closure-proof.sh` 已提供 focused PostgreSQL smoke，验证 Security Closure proof 的 complete fact 校验、idempotent replay 和 completion audit consumption；`areaflow completion backup-restore-proof record` 已提供受控 E6 proof input，complete proof 必须绑定 backup manifest、restore plan、artifact integrity 和 archive preview 输出 metadata，缺绑定或 write/delete/retention/integrity 风险会 fail closed，completion audit 会拒绝旧 loose metadata，并重新运行当前 manifest / restore / integrity / archive preview 的只读 current binding，在当前安全字段漂移、current binding 缺失或查询失败时阻断 E6；`scripts/smoke-backup-restore-proof.sh` 已提供 focused PostgreSQL smoke，验证 Backup Restore proof 的 complete fact 校验、E6 输出绑定、idempotent replay、current binding consumption 和 completion audit consumption；`areaflow completion release-packaging-proof record` 已提供受控 E5 proof input，且 completion audit 可消费 latest complete proof 关闭 release final/packaging blockers；`scripts/smoke-release-packaging-proof.sh` 已提供 focused PostgreSQL smoke，验证 Release Packaging proof 的 complete fact 校验、idempotent replay 和 completion audit consumption；`areaflow completion protected-path-proof record` 已提供受控 E9 proof input，且 completion audit 可消费 latest clean/authorized proof；release final gate 是必要非充分条件。 | fixture completion audit snapshot (evidence_class=fixture) 不是 real 100% evidence；`real_100_breakdown.completed_evidence` 只能说明对应 evidence item 已关闭，不能把静态顶层 `real_100_blockers` 解释为清零；E6 current binding 只是只读 manifest / restore / integrity / archive preview 的安全字段重算和漂移阻断，不是 restore apply、artifact copy/delete/upload、GC 或真实 AreaMatrix 写入；还需要真实 release candidate、AreaMatrix execution cutover 和 full E1-E9 closure。 |
| Operations / deployment / observability 边界 | `docs/architecture/operations-deployment-observability-boundary.md`、`docs/architecture/api-surface.md`、`docs/milestones/v0.9-desktop-shell.md`、`docs/milestones/v1.0-stable-platform.md`、[`operations-readiness-evidence.md`](./operations-readiness-evidence.md) | service status、Desktop gate、只读 operations readiness、metadata-only support bundle preview 和 migration ledger readiness 已有，并由 E7 operations proof smoke、v1 stable fixture / Web smoke 覆盖；telemetry 默认 local-only，support export、migration apply、managed ops、process control、remote telemetry 和 destructive rollback 仍未打开。 |
| 多项目 `project_key` 隔离边界 | `docs/product/master-plan.md`、`docs/architecture/data-model-v1.md`、`docs/architecture/auth-team-secret-boundary.md`、[`multi-project-isolation-evidence.md`](./multi-project-isolation-evidence.md) | schema、API 路径、worker、artifact 和 audit 设计均以 `project_key` 为 scope；`TestStoreProjectKeyIsolationWithPostgres` 已通过真实 PostgreSQL smoke，覆盖同名 workflow version、run、artifact、event、audit、worker 和 lease recovery 隔离；`TestProjectScopedAPIsUseRouteProjectKey` 已覆盖 versioned API route 对 summary/events/audit/workers/workflow versions/runs/artifacts 的 project scope；`TestGlobalRunEndpointsHonorProjectKeyVisibility` 和 `TestGlobalArtifactEndpointsHonorProjectKeyVisibility` 已覆盖全局 run/artifact ID route 的兼容型 `project_key` visibility guard；Web browser smoke 已验证 run detail 使用 `?project_key=`；R4 auth/team/API token/secret/remote worker ladder 已定义；`scripts/smoke-project-isolation.sh` 和 `make smoke-docker-project-isolation` 已提供 smoke 入口；无 `AREAFLOW_DATABASE_URL` 时默认 skip。 | API token enforcement、team permission enforcement、secret resolve 和 remote worker credential 仍未打开，完整权限模型不能只依赖兼容型 query guard。 |
| Workspace / environment 后置实体模型 | `docs/product/master-plan.md`、`docs/architecture/data-model-v1.md` | v1.0 前明确不引入复杂 workspace/environment 表；用 project grouping、connections、worker kind、engine profile 和 scheduling metadata 表达。 | 后续提升为一等实体时仍必须保持 `project_key` 隔离，不得破坏历史 run/artifact/audit scope。 |
| AreaFlow self dogfood 节奏 | `docs/product/master-plan.md`、`docs/product/phase-backlog.md` | schema 支持 self project，但 AreaMatrix dogfood 是第一主线；self dogfood 只能按只读 -> dry-run/artifact-only -> stable self-hosting 逐步打开。 | 不能用 self dogfood 跳过 Command API、permission、gate、approval、rollback、audit 或 release publish 高风险开闸。 |

E1 补充口径：`areaflow completion source-alignment-proof record --status complete` 不再是 facts-only evidence。
CLI 会只读采集当前 AreaFlow E1 source path/hash/source-set hash，completion audit 会重算 current binding
并拒绝旧 loose proof 或 source drift；该流程不改写文档、不运行 shell、不写 AreaMatrix。

E4 补充口径：`areaflow completion archive-proof record --status complete`、
`areaflow completion shim-retirement-proof record --status complete` 和
`areaflow completion execution-cutover-proof record --status complete` 不再是 facts-only evidence。complete
proof 必须携带 archive/shim/execution scope、required path/prerequisite/surface/task/action lists、rollback target、
fail-closed policy binding、release-candidate proof evidence URI、`review_decision=approved`、非空 `reviewed_by`、
RFC3339 `reviewed_at`，还必须携带 deterministic current binding hash，且 latest proof EventID 必须为正；
completion audit 会拒绝旧 loose metadata、hash 漂移、缺少事件 ID、local/fixture/script/smoke 机制证据或缺少
review metadata，并返回对应 binding / proof event / review blocker。即使 proof metadata 通过，只有真实
AreaMatrix identity（root `/Users/as/Ai-Project/project/AreaMatrix`、kind `product-repo`、adapter/profile
`areamatrix`、branch `main`）才能移除真实 E4 blockers。这仍然只是 AreaFlow evidence closure，不代表真实
AreaMatrix archive、execution forwarding、`./task-loop run` forwarding 或 shim retirement 已经执行。

## Next Recommended Closure Order

1. 继续把 `scripts/smoke-v1-stable-fixture.sh` 作为 v0.1-v1.0 长链回归基线；真实 `scripts/smoke-local.sh`
   指向 AreaMatrix 时仍需单独授权，读取需要 `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1`，写真实
   `.areaflow/status.json` 还需要 `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1`。
2. Package A 先运行 `make smoke-package-a` / `bash scripts/audit-package-a-authorization-packet.sh --json`
   作为授权前检查；2026-07-10 Package A 已按窄授权写入真实 AreaMatrix `.areaflow/status.json`，
   且 Package A 的写入范围只允许是 `.areaflow/status.json`。authorization packet 必须绑定当前 target preimage 的 exists、sha256、
   size 和 accepted preimage schema status，并把后续 `status-projection-apply-gate` / apply 所需的
   非审批参数列出：`--target`、`--expected-before-*`、`--schema-uri`、`--validator-preflight`、
   `--protected-path-check`、`--protected-path-fingerprint-sha256`、`--rollback-action`、
   `--accept-preimage-schema`，以及已绑定到当前 AreaFlow DB 最新 import snapshot 的 `--source-hash`。
   `--protected-path-fingerprint-sha256` 是非审批 gate 参数，绑定除目标 `.areaflow/status.json`
   之外的 protected paths 内容指纹；写前重查不一致、写后漂移或 source-hash 绑定期间漂移都必须 fail
   closed。如果显式提供
   `AREAFLOW_PACKAGE_A_SOURCE_HASH=<expected latest AreaFlow import snapshot hash>`，packet 仍必须重新取 DB
   权威 hash 并要求一致。缺少 DB 绑定、hash 缺失或 hash mismatch 时 packet 必须停在
   `blocked_needs_authoritative_source_hash`；如果 target preimage drift，后续 gate / write-time recheck 必须 fail closed。审批三件套 `--explicit-approval`、
   `--approval-actor`、`--approval-reason` 只在 `post_authorization_required_arguments` 中列出；真实
   AreaMatrix Package A 的 Go apply gate 要求 `--approval-reason` 精确等于
   `授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json`，不能由 dirty reviewer 或泛化审批理由代替。
   有 `AREAFLOW_DATABASE_URL` 时，`smoke-package-a` / authorization packet 会写 AreaFlow
   DB 来迁移、注册、导入并绑定最新 import snapshot source hash；这仍不写 AreaMatrix，不等于 Package A
   apply 授权。纯只读 packet 审计应 unset `AREAFLOW_DATABASE_URL`，并期待停在
   `blocked_needs_authoritative_source_hash`。如果 AreaMatrix protected paths 已有既有脏状态，必须先用
   `bash scripts/audit-package-a-dirty-review.sh` 取得 path/status output 的 `dirty_output_sha256`，再通过
   `AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<sha256>` 和
   `AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>` 复核精确 dirty path/status output；内容漂移由
   authorization packet 的 `protected_path_fingerprint_sha256` 单独绑定和写前/写后复查。readiness、dirty review
   和 authorization packet 会同时输出 `protected_path_rule_count` 与 `dirty_path_count`，用于区分受保护规则数量
   和当前实际 dirty path 数量。authorization packet 还必须把持久写入范围与同目录原子写临时路径分开：
   `durable_allowed_writes` 只能是 `.areaflow/status.json`，`transient_write_paths` 只能覆盖
   `.areaflow/.status.json.tmp-*` 和 `.areaflow/.status.json.rollback-*`，并声明临时路径必须清理且不构成额外
   durable authorization。该复核不授权写入，也不允许 shim files、`workflow/README.md`、
   `workflow/versions/**`、`./task-loop run` forwarding、engine、secret、network、publish 或 restore。
3. `make smoke-docker-completion-audit-real-identity-readiness` 可作为真实 AreaMatrix 身份的只读 release-candidate
   snapshot readiness guardrail：它只导入真实项目到隔离临时 AreaFlow DB 并查询 readiness，期望 blocker 是
   `completion_audit_snapshot_missing`，而不是 fixture identity blocker；它不记录 snapshot、不跑 smoke、不生成
   release package，也不写 AreaMatrix。
4. Package B 授权前先运行 `make smoke-package-b-readiness`，复核 protected/worktree dirty output hash，展示
   authorization packet、影响、风险、验证和回滚；只有用户明确授权
   `授权执行 Package B，只允许落地 AreaMatrix read-only shim，不允许 ./task-loop run 转发` 后，才可按
   [`../migration/areamatrix-compatibility-shim-plan.md`](../migrations/areamatrix-compatibility-shim-plan.md)
   在 AreaMatrix 只落转发/降级入口。
5. 基于 v0.6 受限 Codex CLI adapter preview、execution approval gate、fixture execution apply、read-only verify、approved artifact write、managed generated write、generated write readiness 和 generated write apply beta gate，
   决定真实 AreaMatrix generated-only apply beta 以及后续 copy/verify/repair/checkpoint 的最小安全打开顺序。
6. 在真实 dogfood 证据足够后，再讨论 v1.x 的 restore apply、secret resolve、remote worker 和 publish 能力。
