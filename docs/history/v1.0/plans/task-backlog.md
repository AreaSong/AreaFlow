# AreaFlow 0-100% Platform Backlog

## 定位

本文把 [`../../docs/product/phase-backlog.md`](./phase-backlog.md) 拆成可执行候选任务。

这些任务仍是 backlog，不是 active task：

- 不代表已经批准执行。
- 不会被 runner 或 worker 领取。
- 不写 AreaMatrix。
- 不打开真实 execution、secret、restore apply 或 publish apply。
- 进入 active 前必须再按 milestone、risk level、permission、gate 和 approval 重新确认。

## 任务字段

```text
ID:
  稳定候选任务编号。

Phase:
  对应 phase backlog 阶段。

Goal:
  任务目标。

Inputs:
  主要源事实。

Deliverables:
  完成后应产生的文件、命令、API、测试或证据。

Gate:
  进入下一步前必须证明的事实。

Risk:
  R0 read_only / R1 projection / R2 managed_write / R3 execution / R4 migration_security。
```

## Phase 0 Foundation

### AF-P0-001 Design Source Alignment

Phase: Phase 0 Foundation

Goal: 证明产品、路线、ADR、架构和 milestone 文档对 0-100% 目标没有互相冲突。

Inputs:

- `docs/product/platform-blueprint.md`
- `docs/product/phase-backlog.md`
- `docs/product/roadmap.md`
- `docs/milestones/README.md`
- `docs/adr/0001-technology-stack.md` 到 `docs/adr/0006-platform-operating-boundary.md`

Deliverables:

- 文档交叉引用完整。
- 关键决策一致：Go + PostgreSQL + REST/JSON + SSE、AreaMatrix dogfood、adapter/profile 分离、v1.0 preview-only release hardening。
- `docs/development/implementation-gap-audit.md` 能解释当前实现状态。

Gate:

- `git diff --check -- docs/product docs/milestones docs/adr docs/development`
- 所有新增链接目标存在。

Risk: R0 read_only。

### AF-P0-002 Directory Boundary Audit

Phase: Phase 0 Foundation

Goal: 对照长期目录边界，确认当前目录结构没有过早抽象，也没有阻断后续 v0.4-v1.0 拆分。

Inputs:

- `docs/product/platform-blueprint.md`
- `docs/architecture/overview.md`
- `docs/architecture/adapter-profile-boundary.md`
- 当前 `cmd/`、`internal/`、`migrations/`、`docs/`、`governance/`、`workflow/`、`examples/`、`web/`、`tasks/`、`scripts/`

Deliverables:

- [`../../docs/development/directory-boundary-audit.md`](../evidence/directory-boundary-audit.md) 目录差距清单。
- v0.4 前暂不创建空目录的理由。
- v0.5-v1.0 拆分 `workflow`、`runner`、`worker`、`permission`、`engine`、`secret`、`integration` 的触发点。

Gate:

- 不创建无意义空目录。
- 当前实现仍能通过 `go test ./...`。

Risk: R0 read_only。

### AF-P0-003 Governance Boundary Audit

Phase: Phase 0 Foundation

Goal: 确认权限、风险等级、Command API、Query API 和 audit 规则已经覆盖后续所有高风险能力。

Inputs:

- `docs/architecture/security-permissions.md`
- `docs/architecture/command-approval-contract.md`
- `docs/architecture/api-surface.md`
- `docs/adr/0005-phase-0-foundation-baseline.md`
- `docs/adr/0006-platform-operating-boundary.md`
- `governance/README.md`

Deliverables:

- [`../../docs/development/governance-boundary-audit.md`](../evidence/governance-boundary-audit.md) governance 边界审计。
- Capability 与 R0-R4 风险等级一致。
- R2-R4 操作都有 preview、gate、approval、rollback 和 audit 要求。
- Query API 无副作用原则被所有文档引用。
- Command API、approval scope、permission preflight、expected version/hash、rollback 和 safety facts 有统一合同。
- Capability resource matrix、command class、precondition snapshot、affected/forbidden resources 和 project write
  write-set 有统一合同。

Gate:

- release、restore、secret、remote worker、plugin、publish 不能被描述为 v1.0 默认可执行能力。
- preview / readiness / gate 不能被描述为 apply。
- capability 不能被描述成全局布尔许可；必须绑定 resource scope、command class 和 approval scope。

Risk: R0 read_only。

### AF-P0-004 Operations Deployment Observability Boundary

Phase: Phase 0 Foundation

Goal: 确认 install、migration、service lifecycle、health/readiness、doctor、logs、support bundle、telemetry、
upgrade 和 rollback 的长期边界，不让 v1.0 本机诊断能力被误解为远程运维或托管升级。

Inputs:

- `docs/architecture/api-surface.md`
- `docs/architecture/release-final-gate-contract.md`
- `docs/milestones/v0.9-desktop-shell.md`
- `docs/milestones/v1.0-stable-platform.md`
- `docs/product/platform-blueprint.md`

Deliverables:

- [`../../docs/architecture/operations-deployment-observability-boundary.md`](../../../architecture/operations-deployment-observability-boundary.md)
  运维、部署和可观测性边界合同。
- v1.0 只允许 local bootstrap、service status、health/readiness/doctor、metadata-only support bundle preview
  和 local-only telemetry。
- v1.x 才打开 remote ops control、managed upgrade、destructive rollback 和 full support bundle export。

Gate:

- Admin API 不得绕过 Command API 改写 workflow 业务状态。
- support bundle preview 不包含 secret、prompt、用户文件、raw artifact 或未脱敏日志。
- telemetry 默认 local-only。
- AreaMatrix protected paths 不能被 ops / diagnostics / rollback 工具触碰。

Risk: R0 read_only。

## v0.1 Import + Status Mirror

### AF-V01-001 PostgreSQL Bootstrap Smoke

Phase: v0.1 Import + Status Mirror

Goal: 证明 AreaFlow 可以从空 PostgreSQL 环境迁移、启动并注册项目。

Inputs:

- `migrations/**`
- `internal/migrate/**`
- `cmd/areaflow/main.go`
- `docs/development/setup.md`
- `docs/architecture/v0.1-import-mirror-contract.md`
- `docs/architecture/data-model-v0.1.md`

Deliverables:

- [`../../docs/development/bootstrap-smoke-evidence.md`](../evidence/bootstrap-smoke-evidence.md) 最近一次 bootstrap smoke 证据。
- migration 命令可重复。
- `project add` 可注册 AreaMatrix-like fixture。
- v0.1 最小闭环中的 migrate / project add / project status 基础命令与合同一致。
- v0.1 core tables、early support tables 和 inactive tables 的边界与 data model 一致。
- `go test ./...` 覆盖 migration/store 基础路径。

Gate:

- `go test ./...`
- `go build ./cmd/areaflow`
- `AREAFLOW_DATABASE_URL=... ./scripts/smoke-fixture.sh`

Risk: R0 read_only。

### AF-V01-002 AreaMatrix Adapter Metadata Import

Phase: v0.1 Import + Status Mirror

Goal: 只读导入 AreaMatrix workflow metadata、residual、progress/task metadata 和 artifact metadata。

Inputs:

- `internal/adapter/areamatrix/**`
- `internal/importer/**`
- `docs/dogfood/areamatrix-contract.md`
- `docs/migration/areamatrix-workflow-migration.md`
- `docs/architecture/v0.1-import-mirror-contract.md`
- `docs/architecture/areamatrix-import-scope-contract.md`
- `docs/architecture/project-config.md`

Deliverables:

- [`../../docs/development/areamatrix-adapter-import-evidence.md`](../evidence/areamatrix-adapter-import-evidence.md) 最近一次真实 AreaMatrix 只读 metadata import 证据。
- import run 可重复。
- project reference artifact metadata 有 type、hash、path/URI、project、version/run 关联。
- 不复制历史 artifact 原文。
- import 产生的 `run` 只代表 import command，不代表 worker execution。
- `areaflow.yaml` scheduling、engine、allowed commands 和 enabled flags 只作为 metadata 导入。
- read envelope、minimum import set、explicit non-imports 和 artifact metadata minimum set 与 AreaMatrix
  import scope 合同一致。

Gate:

- `project import` 可重复。
- import snapshot 足以支撑 v0.2 的 `project import-diff`。
- imported artifact rows 使用 metadata-only backend，例如当前 AreaMatrix importer 的 `external_project`。
- 真实 AreaMatrix 只读 smoke 不修改 `.areaflow/status.json` 或 `workflow/README.md`。

Risk: R0 read_only。

### AF-V01-003 Guarded Status Projection

Phase: v0.1 Import + Status Mirror

Goal: 生成 `.areaflow/status.json` 粗略状态，并确保它只是 projection，不是主状态。

Inputs:

- `internal/status/**`
- `internal/project/status_projection_apply.go`
- `docs/architecture/project-config.md`
- `docs/architecture/v0.1-import-mirror-contract.md`
- `docs/architecture/data-model-v0.1.md`
- `docs/product/phase-backlog.md`

Deliverables:

- [`../../docs/development/status-projection-evidence.md`](../evidence/status-projection-evidence.md) 最近一次 fixture status projection apply 证据。
- projection payload 包含 summary、active versions、open gates、active runs 和 AreaFlow 链接。
- `status_projections` 记录 source event/hash、target、write state。
- 受保护 apply 写 command request、event 和 audit event。
- `.areaflow/status.json` 只保存粗略状态，不保存 execution attempt、logs、checkpoint、secret 或 artifact 原文。
- `export-status` 只是 compatibility alias，长期主语义是 `status-projection-apply`。

Gate:

- 只在 allowlist 路径写 `.areaflow/status.json`。
- `workflow/README.md` 自动写入仍关闭。
- 返回 `execution_write_attempted=false` 和 `engine_call_attempted=false`。

Risk: R1 projection。

## v0.2 Shadow Doctor + Drift Check

### AF-V02-001 Doctor And Readiness Bundle

Phase: v0.2 Shadow Doctor + Drift Check

Goal: 提供只读 doctor、summary、readiness、import-diff 和 verification bundle。

Inputs:

- `docs/architecture/v0.2-shadow-doctor-contract.md`
- `internal/doctor/**`
- `internal/project/workflow.go`
- `docs/architecture/workflow-engine-contract.md`

Deliverables:

- [`../../docs/development/shadow-doctor-readiness-evidence.md`](../evidence/shadow-doctor-readiness-evidence.md) 最近一次 doctor/readiness/import-diff/verify-bundle 证据。
- 稳定 JSON 输出。
- drift、stage coverage、native doctor status 分层。
- skipped/warn/fail 不被压成 pass。
- verification bundle phase gate 的 `blocked` / accepted warnings 不被解释为 cutover readiness。

Gate:

- `project doctor`
- `project summary`
- `project readiness`
- `project import-diff`
- `project verify-bundle` / API `verification-bundle`

Risk: R0 read_only。

### AF-V02-002 Native Doctor Authorization Boundary

Phase: v0.2 Shadow Doctor + Drift Check

Goal: 确认 AreaMatrix native doctor 只能在显式授权和 command allowlist 下运行。

Inputs:

- `docs/architecture/security-permissions.md`
- `examples/areamatrix/areaflow.yaml`
- `docs/migration/areamatrix-workflow-migration.md`

Deliverables:

- [`../../docs/development/native-doctor-authorization-evidence.md`](../evidence/native-doctor-authorization-evidence.md) 最近一次 native doctor 授权边界证据。
- 未传 `--allow-native` 时只返回 skipped/warn。
- 禁止命令包括 `./task-loop run`、`git reset --hard`、`git checkout --`、`rm -rf`。

Gate:

- 未授权不执行命令。
- 授权也受 command allowlist 和 forbidden command 限制。

Risk: R0 read_only。

## v0.3 New Version Authoring

### AF-V03-001 Workflow Version Authoring Model

Phase: v0.3 New Version Authoring

Goal: AreaFlow 能创建 authored workflow version、stage skeleton、workflow item 和 item link。

Inputs:

- `docs/architecture/v0.3-version-authoring-contract.md`
- `internal/project/workflow.go`
- `internal/workflow/profile.go`
- `workflow/profiles/areamatrix/profile.yaml`
- `docs/architecture/workflow-lifecycle.md`

Deliverables:

- [`../../docs/development/workflow-version-authoring-evidence.md`](../evidence/workflow-version-authoring-evidence.md) 最近一次 workflow version authoring fixture 证据。
- workflow version create。
- stage skeleton create。
- workflow item/link create。
- profile binding 写入 `profile_id`、`profile_version`、`profile_hash`、`adapter`。
- skeleton artifact 写入 AreaFlow-owned local artifact store。

Gate:

- profile hash 冻结。
- profile binding drift 可检测。
- 不写被管理项目 workflow 目录。
- placeholder artifact 不让 content gate 自动通过。

Risk: R0 read_only / R2 managed_write only if project export is later enabled。

### AF-V03-002 Gate Transition Approval Records

Phase: v0.3 New Version Authoring

Goal: gate result、transition preview 和 approval record 可审计。

Inputs:

- `docs/architecture/v0.3-version-authoring-contract.md`
- `internal/project/workflow.go`
- `migrations/000003_v0_3_gate_results.sql`
- `migrations/000004_v0_3_approval_transition.sql`
- `docs/architecture/api-surface.md`

Deliverables:

- [`../../docs/development/gate-transition-approval-evidence.md`](../evidence/gate-transition-approval-evidence.md) 最近一次 gate/transition/approval fixture 证据。
- `workflow gate run/list`
- transition preview。
- approval record。
- event/audit/command request。

Gate:

- promotion preview 不 apply。
- approval 不等于 execution。
- live mapping gate 保持独立。
- approved approval record 只能在 ready transition preview 后记录。
- `approval_is_execution=false` 保持可查。

Risk: R0 read_only / R2 managed_write only if approval writes project files, which remains closed。

## v0.4 Workflow Ownership Cutover

### AF-V04-001 Compatibility And Shim Readiness

Phase: v0.4 Workflow Ownership Cutover

Goal: 提供 AreaMatrix compatibility contract、shim preview 和 shim readiness。

Inputs:

- `docs/architecture/v0.4-workflow-ownership-cutover-contract.md`
- `docs/migration/areamatrix-compatibility-shim-plan.md`
- `docs/migration/cutover-rollback-compat.md`
- `internal/project/workflow.go`

Deliverables:

- [`../../docs/development/compatibility-shim-readiness-evidence.md`](../evidence/compatibility-shim-readiness-evidence.md) 最近一次 compatibility / shim readiness fixture 证据。
- compatibility query。
- shim preview。
- shim readiness gate。
- planned files、command mapping、安全禁区和验证命令。
- AreaMatrix edit authorization packet API/CLI：allowed files、forbidden paths/actions、preflight、post-edit verification、rollback scope 和 safety facts。
- shim lifecycle state：first shim 只能进入 `read_only_shim`，不能声明 `execution_forwarding` 或
  `retired_thin_entry`。
- shim authorization packet 只是只读授权包，不等于用户已批准编辑 AreaMatrix。

Gate:

- 真实 AreaMatrix shim 文件修改仍需单独授权。
- `./task-loop run` 在 execution cutover 前 blocked。
- read-only shim 不得转发 `./task-loop run`，也不得启动旧 runner。
- 授权前必须完成 `make smoke-docker-shim-authorization-preflight`、真实 AreaMatrix read-only smoke 和 dirty worktree review。
- 授权包不得包含 `workflow/versions/**/execution/**`、`progress.json`、source code、git checkpoint 或 `./task-loop run` forwarding。

Risk: R0 read_only；真实 shim 写入为 R2 managed_write。

### AF-V04-002 Authoring Cutover Apply

Phase: v0.4 Workflow Ownership Cutover

Goal: 只在 AreaFlow PostgreSQL 内执行 authoring cutover，并留下可回滚事实。

Inputs:

- `docs/architecture/v0.4-workflow-ownership-cutover-contract.md`
- `internal/project/cutover_apply.go`
- `docs/migration/cutover-rollback-compat.md`
- `docs/product/phase-backlog.md`

Deliverables:

- [`../../docs/development/authoring-cutover-apply-evidence.md`](../evidence/authoring-cutover-apply-evidence.md) 最近一次 authoring cutover apply fixture 证据。
- `project.cutover.apply` command request。
- cutover event。
- audit event。
- workflow version authoring cutover 状态。
- command response 中 `area_matrix_write_attempted=false`。

Gate:

- cutover readiness gate pass。
- 只允许 `mode=authoring_cutover`。
- 返回 `project_write_attempted=false`。
- 返回 `execution_write_attempted=false`。
- 返回 `area_matrix_write_attempted=false`。
- rollback 不删除 historical event/audit/artifact。

Risk: R2 managed_write in AreaFlow DB only；project file writes remain closed。

## v0.5 Runner Preview

### AF-V05-001 Runner Preview Evidence

Phase: v0.5 Runner Preview

Goal: 建立 run、run_task、run_attempt、artifact、event 和 audit 的 dry-run 闭环。

Inputs:

- `internal/project/runner.go`
- `docs/architecture/v0.5-runner-preview-contract.md`
- `docs/architecture/execution-model.md`
- `docs/milestones/v0.5-runner-preview.md`

Deliverables:

- [`../../docs/development/runner-preview-evidence.md`](../evidence/runner-preview-evidence.md) 最近一次 runner preview focused smoke 证据。
- runner preview command/API。
- `runner_preview_report` local artifact。
- risk/permission preflight。
- dry-run event/audit。
- completed `runner.preview` command request response。

Gate:

- 不执行 shell。
- 不写被管理项目文件。
- 不调用 engine。
- preview artifact hash/size 可校验。
- command response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、
  `area_matrix_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、
  `secrets_resolved=false`、`network_used=false`。

Risk: R0 read_only plus AreaFlow-owned `write_artifacts` evidence。

### AF-V05-002 Run Control Dry-run Boundary

Phase: v0.5 Runner Preview

Goal: `run.start`、`run.drain`、`run.cancel` 只控制 dry-run run 的 AreaFlow DB 状态。

Inputs:

- `internal/project/run_control.go`
- `docs/architecture/v0.5-runner-preview-contract.md`
- `docs/architecture/api-surface.md`

Deliverables:

- [`../../docs/development/run-control-evidence.md`](../evidence/run-control-evidence.md) 最近一次 run control focused smoke 证据。
- start/drain/cancel command requests。
- event/audit。
- explicit no project write / no engine call response。
- non-dry-run run control denial evidence。

Gate:

- 不领取 task。
- 不启动 worker。
- 不写 project files。
- 不执行 shell。
- 不调用 engine。
- command response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、
  `area_matrix_write_attempted=false`、`task_claimed=false`、`worker_started=false`、
  `commands_run=false`、`secrets_resolved=false`、`network_used=false`。

Risk: R0/R1 for AreaFlow DB state only。

## v0.6 Worker Execution Beta

阶段合同：[`../../docs/architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md)。
v0.6 只打开 worker lifecycle、lease、dry-run run-once 和 scoped execution evidence；fixture、read-only、
artifact-only、fixture/temp rollback drill、readiness 和 beta gate 证据不得累计成真实 AreaMatrix
execution cutover。

### AF-V06-001 Worker Registry Lease Lifecycle

Phase: v0.6 Worker Execution Beta

Goal: 完成 worker register、heartbeat、lease acquire/release/recover 和 capability denial。

Inputs:

- `internal/project/worker.go`
- `docs/architecture/v0.6-worker-beta-contract.md`
- `docs/architecture/execution-model.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/worker-lease-evidence.md`](../evidence/worker-lease-evidence.md) 最近一次 worker lease focused smoke 证据。
- worker register。
- heartbeat。
- lease lifecycle。
- recovery。
- denial event/audit。
- completed `worker.register` / `worker.heartbeat` command responses。
- completed `lease.acquire` / `lease.release` / `lease.recover` command responses。

Gate:

- worker 只领取 run_task。
- lease 有 TTL。
- capability denial 不创建 lease、attempt 或 artifact。
- worker lifecycle command response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`、`lease_created=false`。
- lease command response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`。
- lease lifecycle 不创建 `worker_run_once` attempt 或 `worker_run_once_report` artifact。

Risk: R3 execution when real task execution opens；current dry-run/lease lifecycle is bounded by Command API。

### AF-V06-002 Codex CLI Adapter Preview

Phase: v0.6 Worker Execution Beta

Goal: 先提供受限 Codex CLI adapter preview，再决定是否打开真实 execution beta。

Inputs:

- `docs/architecture/execution-model.md`
- `docs/architecture/v0.6-worker-beta-contract.md`
- `docs/architecture/security-permissions.md`
- `docs/product/phase-backlog.md`

Deliverables:

- [`../../docs/development/codex-cli-adapter-preview-evidence.md`](../evidence/codex-cli-adapter-preview-evidence.md) 最近一次 Codex CLI adapter preview focused smoke 证据。
- engine/profile readiness。
- command preview。
- allowed command / path / capability preflight。
- artifact redaction plan。

Gate:

- 未获 approval 不执行真实 Codex CLI。
- 不解析 secret。
- 不写未授权 project path。
- preview response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、`network_used=false`。

Risk: R3 execution。

### AF-V06-003 Execution Approval Gate

Phase: v0.6 Worker Execution Beta

Goal: 在真实 execution apply 前提供只读 go/no-go 门禁，统一检查 run、approval、gate、engine preview
和 worker capability。

Inputs:

- `internal/project/execution_approval_gate.go`
- `docs/architecture/v0.6-worker-beta-contract.md`
- `docs/architecture/execution-model.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/execution-approval-gate-evidence.md`](../evidence/execution-approval-gate-evidence.md) 最近一次 execution approval gate focused smoke 证据。
- `areaflow run execution-gate <run-id> [--json]`。
- `GET /api/v1/runs/{run_id}/execution-approval-gate`。
- dry-run preview run denial。
- worker capability readiness。
- engine adapter preview readiness。
- read-only safety facts。

Gate:

- dry-run run 必须 blocked。
- 非 queued run 必须 blocked。
- 未通过 workflow approval、`approval_gate`、`live_mapping_gate` 必须 blocked。
- engine adapter preview blocked 时必须 blocked。
- 没有 online worker 或 capability 不满足时必须 blocked。
- gate 不创建 `command_requests`、lease、attempt 或 artifact。
- gate 不启动 worker、不调用 engine、不运行 shell、不解析 secret、不写被管理项目文件。
- response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、
  `engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、
  `network_used=false`、`task_claimed=false`、`worker_started=false`、
  `attempt_created=false`、`artifact_created=false`。

Risk: R3 execution boundary；current gate is read-only Query API。

### AF-V06-004 Fixture Execution Apply

Phase: v0.6 Worker Execution Beta

Goal: 在不触碰真实项目、不调用 engine 的前提下，验证 approval-gated execution apply 的最小状态闭环。

Inputs:

- `internal/project/fixture_execution.go`
- `internal/project/execution_approval_gate.go`
- `docs/architecture/execution-model.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/fixture-execution-apply-evidence.md`](../evidence/fixture-execution-apply-evidence.md) 最近一次 fixture execution apply focused smoke 证据。
- `areaflow run fixture-queue <project> <version> [--json]`。
- `areaflow worker fixture-execute <project> <worker-key> --run-id <id> [--json]`。
- `POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-queue`。
- `POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-execute`。
- `fixture_execution` lease / attempt / report artifact。
- run_task 和 run 推进到 `passed`。
- idempotent replay。

Gate:

- 必须先通过 execution approval gate。
- worker 必须 online 且 capability 满足 `read_project`、`write_artifacts`、`run_commands` 和 `execute_agents`。
- apply 只允许 fixture-only AreaFlow state / artifact store 写入。
- 不调用 Codex CLI，不运行 shell，不解析 secret，不访问网络。
- 不写真实 AreaMatrix，不写被管理项目文件，不写 `workflow/versions/**/execution/**`。
- response 记录 `project_write_attempted=false`、`execution_write_attempted=false`、
  `engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、
  `network_used=false`、`area_flow_execution_state_written=true`。

Risk: R3 execution boundary；current apply is fixture-only and does not open real engine/project execution.

### AF-V06-005 Read-only Verify

Phase: v0.6 Worker Execution Beta

Goal: 在不写被管理项目、不调用 engine 的前提下，验证 approval-gated worker 可以读取 allowlisted
project file，并把 hash/size evidence 写入 AreaFlow。

Inputs:

- `internal/project/read_only_verify.go`
- `internal/project/execution_approval_gate.go`
- `docs/architecture/execution-model.md`
- `docs/architecture/api-surface.md`
- `docs/architecture/security-permissions.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/read-only-verify-evidence.md`](../evidence/read-only-verify-evidence.md) 最近一次 read-only verify focused smoke 证据。
- `areaflow run read-only-verify-queue <project> <version> --target-path <path> [--json]`。
- `areaflow worker read-only-verify <project> <worker-key> --run-id <id> [--json]`。
- `POST /api/v1/projects/{project_key}/workflow-versions/{version}/read-only-verify-queue`。
- `POST /api/v1/projects/{project_key}/workers/{worker_key}/read-only-verify`。
- `read_only_verify` lease / attempt / report artifact。
- 当前兼容实现可能把 run_task 和 run 推进到 `verified`；目标模型应收敛为 `status=passed` +
  outcome `read_only_verify_passed`。
- target file sha256 / size evidence。
- idempotent replay。

Gate:

- 必须先通过 execution approval gate。
- worker 必须 online 且 capability 满足 `read_project` 和 `write_artifacts`。
- target path 必须满足 project permission allowlist 和 project-root 防逃逸检查。
- artifact 只保存 target path、sha256 和 size，不保存 target file 原文。
- 不调用 Codex CLI，不运行 shell，不解析 secret，不访问网络。
- 不写真实 AreaMatrix，不写被管理项目文件，不写 `workflow/versions/**/execution/**`。
- response 记录 `project_read_attempted=true`、`project_read_allowed=true`、
  `project_write_attempted=false`、`execution_write_attempted=false`、
  `engine_call_attempted=false`、`commands_run=false`、`secrets_resolved=false`、
  `network_used=false`、`area_flow_execution_state_written=true`。

Risk: R3 execution boundary；current apply is read-only project file hashing and does not open real engine/project write execution.

### AF-V06-006 Approved Artifact Write

Phase: v0.6 Worker Execution Beta

Goal: 在不读取/写入被管理项目、不调用 engine 的前提下，验证 approval-gated worker 可以写入
AreaFlow-owned artifact store，并把 artifact metadata / evidence 写入 PostgreSQL。

Inputs:

- `internal/project/approved_artifact_write.go`
- `internal/project/execution_approval_gate.go`
- `docs/architecture/execution-model.md`
- `docs/architecture/api-surface.md`
- `docs/architecture/security-permissions.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/approved-artifact-write-evidence.md`](../evidence/approved-artifact-write-evidence.md) 最近一次 approved artifact write focused smoke 证据。
- `areaflow run approved-artifact-write-queue <project> <version> [--artifact-label <label>] [--json]`。
- `areaflow worker approved-artifact-write <project> <worker-key> --run-id <id> [--json]`。
- `POST /api/v1/projects/{project_key}/workflow-versions/{version}/approved-artifact-write-queue`。
- `POST /api/v1/projects/{project_key}/workers/{worker_key}/approved-artifact-write`。
- `approved_artifact_write` lease / attempt / report artifact。
- 当前兼容实现可能把 run_task 和 run 推进到 `artifact_written`；目标模型应收敛为 `status=passed` +
  outcome `artifact_write_passed`。
- idempotent replay。

Gate:

- 必须先通过 execution approval gate。
- worker 必须 online 且 capability 满足 `write_artifacts`。
- project config 必须允许 `write_artifacts`。
- apply 只允许写 AreaFlow-owned artifact store 和 AreaFlow PG metadata/evidence。
- 不读取 project file，不调用 Codex CLI，不运行 shell，不解析 secret，不访问网络。
- 不写真实 AreaMatrix，不写被管理项目文件，不写 `workflow/versions/**/execution/**`。
- response 记录 `project_read_attempted=false`、`project_write_attempted=false`、
  `execution_write_attempted=false`、`area_flow_artifact_written=true`、
  `area_flow_execution_state_written=true`、`engine_call_attempted=false`、`commands_run=false`、
  `secrets_resolved=false` 和 `network_used=false`。

Risk: R3 execution boundary；current apply writes only AreaFlow-owned artifact evidence and does not open real engine/project write execution.

### AF-V06-007 Execution Plan Preview

Phase: v0.6 Worker Execution Beta

Goal: 在真实 copy / verify / repair / checkpoint 打开前，提供只读 execution plan preview，统一展示已打开的
artifact-only step 和仍被 blocker 关闭的高风险 execution step。

Inputs:

- `internal/project/execution_plan.go`
- `internal/project/execution_approval_gate.go`
- `docs/architecture/execution-model.md`
- `docs/architecture/api-surface.md`
- `docs/architecture/security-permissions.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/execution-plan-preview-evidence.md`](../evidence/execution-plan-preview-evidence.md) 最近一次 execution plan preview focused smoke 证据。
- `areaflow run execution-plan <run-id> [--json]`。
- `GET /api/v1/runs/{run_id}/execution-plan`。
- steps 覆盖 `execution_approval_gate`、`copy`、`verify`、`approved_artifact_write`、`checkpoint` 和 `repair`。
- `approved_artifact_write` 在 gate pass 时可为 `ready`。
- `copy`、`checkpoint` 和 `repair` 保持 blocked / waiting，并暴露 blocker。

Gate:

- preview 不创建 `command_requests`、lease、attempt 或 artifact。
- preview 不领取 task、不启动 worker、不调用 Codex CLI、不运行 shell、不解析 secret、不访问网络。
- preview 不读取或写入被管理项目，不写 `workflow/versions/**/execution/**`。
- response 记录 `project_read_attempted=false`、`project_write_attempted=false`、
  `execution_write_attempted=false`、`area_flow_artifact_written=false`、
  `area_flow_execution_state_written=false`、`engine_call_attempted=false`、`commands_run=false`、
  `secrets_resolved=false`、`network_used=false`、`task_claimed=false`、`worker_started=false`、
  `attempt_created=false` 和 `artifact_created=false`。

Risk: R3 execution boundary；current preview is read-only and does not open copy / repair / checkpoint /
engine / project write execution.

### AF-V06-008 Approved Project Write Design Gate

Phase: v0.6 Worker Execution Beta

Goal: 在真实项目写入前，定义 approved project write、copy / verify / repair / checkpoint 和 rollback 的
最小安全合同，并保持本任务只读。

Inputs:

- `docs/architecture/execution-model.md`
- `docs/architecture/security-permissions.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.6-worker-beta.md`
- `docs/migration/areamatrix-workflow-migration.md`

Deliverables:

- `areaflow run project-write-design-gate <run-id> [--json]`。
- `GET /api/v1/runs/{run_id}/project-write-design-gate`。
- write-set contract：operation、target path、expected-before hash、after hash、content/patch artifact、
  verification plan、rollback plan、approval 和 required capabilities。
- copy attempt contract：preimage artifact、applied write-set artifact、post-write hash evidence、
  event 和 audit event。
- verify contract：copy success 不等于 done；verify attempt 和 `verify_acceptance` gate 必须单独通过。
- repair contract：verify failure 先生成 failure summary 和 repair plan；repair apply 复用同一 write-set /
  approval / permission / rollback 边界。
- rollback contract：只允许 rollback AreaFlow 自己 apply 的 write-set；当前 hash 不匹配时 blocked。
- checkpoint contract：verify pass 后单独 gate；checkpoint fail 阻断下一 task。
- attempt state promotion matrix：copy success、verify pass/fail、repair plan/apply、checkpoint preview/apply、
  rollback verified 分别允许推进到什么状态、继续禁止什么。
- execution cutover proof：fixture/read-only/artifact-only/rollback drill 只证明各自 scope，不能累计成
  真实 AreaMatrix execution cutover。
- first apply sequence：fixture approved project write -> fixture verify -> fixture rollback drill
  -> generated-only write -> source write -> checkpoint -> repair。

Gate:

- 本任务不创建 `command_requests`、lease、attempt 或 artifact。
- 本任务不读取或写入被管理项目，不运行 shell，不调用 engine，不解析 secret，不访问网络。
- 第一版明确禁止 delete、move、chmod、binary rewrite、symlink target、project-root 外路径和 glob 批量写入。
- 真实 AreaMatrix 写入、`workflow/versions/**/execution/**`、progress、logs、checkpoint 和源码写入仍保持 blocked。
- 后续实现必须先用 fixture root 证明 approved project write，再单独讨论真实 AreaMatrix。

Risk: R3/R4 design boundary；current task is R0 documentation only and does not open project write execution.

### AF-V06-009 Fixture-only Approved Project Write

Phase: v0.6 Worker Execution Beta

Goal: 在真实 AreaMatrix 写入前，用 fixture project 证明 approved project write 的最小 write/verify/rollback
闭环。

Inputs:

- `internal/project/fixture_project_write.go`
- `internal/project/project_write_design_gate.go`
- `internal/api/server.go`
- `internal/app/app.go`
- `docs/architecture/execution-model.md`
- `docs/architecture/security-permissions.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/fixture-project-write-evidence.md`](../evidence/fixture-project-write-evidence.md)
  最近一次 fixture-only approved project write focused 证据。
- `areaflow run fixture-project-write-queue <project> <version> --target-path <path> --content <text>
  --expected-before-sha256 <hash> --expected-before-size <size> [--json]`。
- `areaflow worker fixture-project-write <project> <worker-key> --run-id <id> [--json]`。
- `POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-project-write-queue`。
- `POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-project-write`。
- `fixture_project_write_set`、preimage artifact、copy/verify/rollback attempts、
  `fixture_project_write_report` artifact，以及兼容 `rollback_verified` run/task 状态；目标模型应收敛为
  `status=passed` + outcome `rollback_verified`。

Gate:

- 只允许 fixture project。
- 只允许修改一个已存在、普通文件、非 symlink、非目录、位于 project root 内的 allowlisted target。
- 必须校验 expected-before hash/size。
- 必须写 preimage artifact，并在 commit 前恢复 preimage hash/size。
- 不调用 engine，不运行 shell，不解析 secret，不访问网络。
- 不写真实 AreaMatrix，不写 `workflow/versions/**/execution/**`。
- 不开放 create/delete/move/chmod/binary rewrite/symlink target/project-root escape/glob 批量写入。

Risk: R3 project write boundary；current implementation opens only fixture write/verify/rollback drill. Next project
write step must be managed-project generated-only write with a separate gate.

### AF-V06-010 Managed Generated Write Gate

Phase: v0.6 Worker Execution Beta

Goal: 在 managed-project generated-only apply 打开前，提供只读门禁，统一展示 allowed generated prefixes、
required write-set fields、unsupported operations、apply sequence 和 blockers。

Inputs:

- `internal/project/managed_generated_write_gate.go`
- `internal/project/project_write_design_gate.go`
- `internal/api/server.go`
- `internal/app/app.go`
- `docs/architecture/execution-model.md`
- `docs/architecture/security-permissions.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/managed-generated-write-gate-evidence.md`](../evidence/managed-generated-write-gate-evidence.md)
  最近一次 managed generated write gate focused 证据。
- `areaflow run managed-generated-write-gate <run-id> [--json]`。
- `GET /api/v1/runs/{run_id}/managed-generated-write-gate`。
- response 暴露 `allowed_generated_prefixes`、`required_write_set_fields`、`unsupported_operations`、
  `apply_sequence`、`generated_only_write_ready` 和 `generated_only_apply_open=false`。
- required capabilities 必须是 `read_project`、`write_artifacts` 和 `write_generated`，不能借用
  `write_code`。

Gate:

- 本任务不创建 `command_requests`、lease、attempt 或 artifact。
- 本任务不读取或写入被管理项目，不运行 shell，不调用 engine，不解析 secret，不访问网络。
- `generated_only_apply_open` 必须保持 false。
- source write、workflow execution write、progress JSON、checkpoint、repair、delete/move/chmod/binary rewrite、
  symlink target、project-root escape 和 glob bulk write 必须保持 blocked。

Risk: R3 project write boundary；current task is read-only gate only. Next implementation step is managed-project
generated-only apply with a separate approval and rollback drill.

### AF-V06-011 Managed Generated Write Apply Core + API CLI Surfacing

Phase: v0.6 Worker Execution Beta

Goal: 提供 managed-project generated-only apply 的核心服务链和受限 API/CLI surfacing，证明
`write_generated` capability、generated prefix、expected-before、copy/verify/rollback、artifact、audit 和
operator 入口合同。

Inputs:

- `internal/project/managed_generated_write_apply.go`
- `internal/project/managed_generated_write_apply_test.go`
- `internal/project/managed_generated_write_gate.go`
- `docs/architecture/security-permissions.md`
- `docs/architecture/execution-model.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/managed-generated-write-apply-evidence.md`](../evidence/managed-generated-write-apply-evidence.md)
  最近一次 managed generated write apply + API/CLI focused 证据。
- Core service: `QueueManagedGeneratedWrite`。
- Core service: `WriteManagedGenerated`。
- REST API: `POST /api/v1/projects/{project_key}/workflow-versions/{version}/managed-generated-write-queue`。
- REST API: `POST /api/v1/projects/{project_key}/workers/{worker_key}/managed-generated-write`。
- CLI: `areaflow run managed-generated-write-queue <project> <version> --target-path <path> --content <text> --expected-before-sha256 <hash> --expected-before-size <size>`。
- CLI: `areaflow worker managed-generated-write <project> <worker-key> --run-id <id>`。
- Smoke: `scripts/smoke-managed-generated-write.sh`。
- command response 暴露 `managed_generated_write`、`generated_only`、`generated_only_apply_open`、
  `fixture_or_temp_project_only`、`real_areamatrix_write_opened=false`、copy/verify/rollback attempt 和 safety facts。

Gate:

- 只允许 fixture/temp project。
- 只允许 `.areaflow/generated/**` 和 `.areamatrix/generated/**`。
- worker/project capability 必须包含 `read_project`、`write_artifacts` 和 `write_generated`。
- target path 必须同时通过 `read_project` 和 `write_generated` path allowlist。
- 只支持已存在普通文件，必须校验 expected-before hash/size。
- 必须写 preimage/report artifact，并在 commit 前 rollback 到 preimage hash/size。
- API/CLI 只允许调用同一条 fixture/temp generated-only rollback drill。
- focused smoke 必须证明 fixture/temp 成功路径和 non-fixture product denial 路径；真实 AreaMatrix
  文件指纹保护栏必须显式设置 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` 才运行。
- `generated_only_apply_open=true` 只能作为 fixture/temp rollback drill scope 内的兼容响应字段，不能解释为
  retained managed-project apply 或真实 AreaMatrix generated apply 已打开。
- 不开放真实 AreaMatrix 写入、保留 generated apply 结果、source write、checkpoint、repair、engine、shell、
  secret、network 或 `workflow/versions/**/execution/**`。

Risk: R3 project write boundary；current implementation opens service/API/CLI only for fixture/temp
generated-only write/verify/rollback drill and has a real PostgreSQL focused smoke. Next step is managed
generated write dogfood readiness, still without real AreaMatrix generated apply.

### AF-V06-012 Generated Write Dogfood Readiness

Phase: v0.6 Worker Execution Beta

Goal: 在真实 AreaMatrix generated-only apply 打开前，提供 project-scoped 只读 readiness，说明项目配置、
permission rows、generated-only path allowlist、dangerous deny、rollback contract 和高风险关闭状态是否已具备
人工审查资格。

Inputs:

- `internal/project/generated_write_readiness.go`
- `internal/project/generated_write_readiness_test.go`
- `internal/api/server.go`
- `internal/app/app.go`
- `docs/architecture/security-permissions.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/generated-write-readiness-evidence.md`](../evidence/generated-write-readiness-evidence.md)
  最近一次 generated write readiness focused 证据。
- Core service: `GeneratedWriteReadiness`。
- REST API: `GET /api/v1/projects/{project_key}/generated-write-readiness`。
- CLI: `areaflow project generated-write-readiness <project> [--json]`。
- response 暴露 `ready_for_review`、`apply_open=false`、`real_areamatrix_write_opened=false`、
  `required_capabilities`、`allowed_generated_prefixes`、`required_write_paths`、`blockers`、
  `review_blockers` 和 read-only safety facts。

Gate:

- 本任务不依赖 run，不创建 `command_requests`、run、task、lease、attempt、artifact、event 或 audit。
- 本任务只读取 AreaFlow PostgreSQL 中的 active project config 和 permission rows。
- 当前真实 AreaMatrix baseline 应保持 `status=blocked`，因为 `write_generated` 和 generated path allowlist
  尚未打开。
- 即使 generated-only preconditions 都满足，`apply_open=false` 时顶层 `status` 仍必须是 `blocked`。
- 不开放真实 AreaMatrix 写入、保留 generated apply 结果、source write、checkpoint、repair、engine、shell、
  secret、network 或 `workflow/versions/**/execution/**`。

Risk: R0 read_only readiness for an R3 project write boundary；it may only describe review readiness and must not
open apply.

### AF-V06-013 Generated Write Apply Beta Approval Gate

Phase: v0.6 Worker Execution Beta

Goal: 在真实 AreaMatrix generated-only apply beta 打开前，提供只读 approval gate，统一展示 readiness、
explicit R3 approval、focused smoke、expected-before、rollback、scope 和 safety facts。

Inputs:

- `internal/project/generated_write_apply_beta_gate.go`
- `internal/project/generated_write_apply_beta_gate_test.go`
- `internal/project/generated_write_readiness.go`
- `internal/api/server.go`
- `internal/app/app.go`
- `docs/architecture/security-permissions.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.6-worker-beta.md`

Deliverables:

- [`../../docs/development/generated-write-apply-beta-gate-evidence.md`](../evidence/generated-write-apply-beta-gate-evidence.md)
  最近一次 generated write apply beta gate focused 证据。
- Core service: `GeneratedWriteApplyBetaGate`。
- REST API: `GET /api/v1/projects/{project_key}/generated-write-apply-beta-gate`。
- CLI: `areaflow project generated-write-apply-beta-gate <project> [--json]`。
- response 嵌套 `generated-write-readiness`，并暴露 `approval_required=true`、
  `approval_status=needs_approval`、`apply_open=false`、`real_areamatrix_write_opened=false`、
  `required_evidence`、`allowed_generated_prefixes`、`forbidden_actions` 和 read-only safety facts。

Gate:

- readiness 未满足时 blocked。
- readiness 满足时仍 blocked 于 explicit R3 approval 缺失。
- 本任务不创建 approval record、`command_requests`、run、task、lease、attempt、artifact、event 或 audit。
- 本任务不读取或写入真实 AreaMatrix 文件。
- 不开放真实 AreaMatrix generated-only apply beta、保留 generated apply 结果、source write、checkpoint、
  repair、engine、shell、secret、network 或 `workflow/versions/**/execution/**`。

Risk: R0 read_only approval gate for an R3 project write boundary；it may only expose approval requirements and
must not open apply.

## v0.7 Web Dashboard

### AF-V07-001 Read-only Dashboard Coverage

Phase: v0.7 Web Dashboard

Goal: Web 覆盖 project、version、stage、run、artifact、residual、approval、worker、audit、worker pool 和 shim authorization。
阶段合同见 [`../../docs/architecture/v0.7-web-dashboard-contract.md`](../contracts/v0.7-web-dashboard-contract.md)。

Inputs:

- `web/src/**`
- `internal/api/server.go`
- `docs/architecture/v0.7-web-dashboard-contract.md`
- `docs/milestones/v0.7-web-dashboard.md`

Deliverables:

- React/TS dashboard。
- API-backed panels。
- `shim-authorization` blocked gate panel。
- browser smoke。

Gate:

- Web 只走 `/api/v1` GET/SSE。
- Web 不直接读 artifact store path。
- Web 不维护第二状态源。
- Web 不提供 AreaMatrix shim 编辑按钮。
- Web 不发送非 GET `/api/v1` 请求。
- SSE 只做观察，不推进状态。
- Global run/artifact route 必须带 `project_key` visibility guard。

Risk: R0 read_only。

### AF-V07-002 Web Write Action Gate

Phase: v0.7 Web Dashboard

Goal: 定义 approval、drain、cancel、archive 等 Web 写操作的开放门槛。

Inputs:

- `docs/architecture/api-surface.md`
- `docs/architecture/security-permissions.md`
- `docs/product/phase-backlog.md`

Deliverables:

- Web 写操作矩阵。
- 每个写操作的 Command API、risk preview、permission preflight、approval 和 audit 要求。
- Disabled/read-only UI state 与 blockers。

Gate:

- 默认不开启写按钮或保持 disabled/read-only。
- 不允许 Web 直接写 DB 或 project files。
- Gate 本身不创建 command、approval、lease、attempt、artifact、event 或 audit。

Risk: R2-R4 depending on action。

## v0.8 Multi-project Worker Pool

### AF-V08-001 Schedule Preview

Phase: v0.8 Multi-project Worker Pool

Goal: 多项目 worker pool summary 和 schedule preview 稳定展示 recommended / blocked。

Inputs:

- `internal/project/worker.go`
- `docs/milestones/v0.8-multi-project-worker.md`
- `docs/architecture/v0.8-multi-project-worker-pool-contract.md`
- `docs/architecture/execution-model.md`
- `docs/architecture/worker-scheduling-contract.md`

Deliverables:

- worker pool summary。
- schedule preview。
- resource readiness。
- engine readiness。
- agent role matching。
- project-scoped slot calculation。
- blocked reasons for missing capability、parallel limit、engine readiness、secret readiness 和 agent role。

Gate:

- preview 不领取 lease。
- preview 不启动 worker。
- preview 不写 event/audit。
- 每个状态都有 project scope。
- schedule preview 不复用真实 acquire lease path。
- `max_parallel_tasks` 必须参与 available slot 计算。
- AreaMatrix 第一阶段真实 execution 并发保持 1，preview 结果不能解释为真实并发 scheduler 已打开。
- `recommended=true`、`available_slots>0` 和 `next_action=worker_run_once_preview` 不能解释为 scheduler
  apply、lease claim、worker dispatch、secret resolve、remote worker credential 或 execution cutover。
- no engine / no secret / no project write safety facts 可解释。

Risk: R0 read_only。

### AF-V08-002 Engine Secret Readiness Boundary

Phase: v0.8 Multi-project Worker Pool

Goal: engine profile 和 secret_ref 只做 readiness，不解析明文。

Inputs:

- `docs/architecture/security-permissions.md`
- `docs/architecture/auth-team-secret-boundary.md`
- `docs/architecture/project-config.md`
- `docs/architecture/data-model-v1.md`

Deliverables:

- `secret_ref_unavailable` / `secret_ready` / `secret_ref=none` 状态。
- blocked reason。
- no plaintext secret guarantee。

Gate:

- 不读取 env。
- 不读取 keychain。
- 不读取 DB secret 明文。
- 不调用需要 secret 的 engine。

Risk: R0 read_only；真实 secret resolve 是 R4，必须按 `docs/architecture/auth-team-secret-boundary.md`
的 scoped binding、redaction、audit 和 rollback 要求另行开闸。

## v0.9 Desktop Shell

### AF-V09-001 Local Service Status Contract

Phase: v0.9 Desktop Shell

Goal: 为 Desktop shell 提供 local service status、dashboard launcher 和 forbidden actions。

Inputs:

- `internal/project/service_status.go`
- `docs/milestones/v0.9-desktop-shell.md`
- `docs/architecture/v0.9-desktop-shell-contract.md`
- `docs/architecture/api-surface.md`

Deliverables:

- service status API/CLI。
- dashboard URL。
- API URL。
- worker pool status。
- capabilities / forbidden actions。

Gate:

- 不启动真实 workflow。
- 不维护第二数据库。
- 不解析 secret。
- service status 不代表 Desktop process control 已打开。

Risk: R0 read_only。

### AF-V09-002 Tauri Shell Scaffold

Phase: v0.9 Desktop Shell

Goal: 创建 Tauri desktop shell，作为 local service status viewer、desktop gate viewer 和 Web launcher。

Inputs:

- `desktop/`
- `docs/product/phase-backlog.md`
- `docs/milestones/v0.9-desktop-shell.md`
- `docs/architecture/v0.9-desktop-shell-contract.md`

Deliverables:

- Tauri scaffold。
- service health view。
- dashboard launcher。
- shim authorization blocked gate view。
- notification/tray plan。

Gate:

- 所有业务状态来自 AreaFlow API。
- Desktop 不直接写 workflow 或 execution 状态。
- Desktop 不执行 AreaMatrix shim 编辑。
- Desktop 不打开真实 process control、OS notification、native tray/menu、secret resolve 或远程 Team Console。

Risk: R0 read_only / local app capability review before service control。

### AF-V09-003 Desktop Service Control Gate

Phase: v0.9 Desktop Shell

Goal: 为 Desktop shell 提供只读 service control gate，展示 start / stop / restart / notification /
tray/menu 等本机控制能力为什么仍禁用，以及后续打开所需的 capability、preflight、approval、audit
和 recovery evidence。

Inputs:

- `internal/project/desktop_service_control_gate.go`
- `internal/api/server.go`
- `desktop/src/main.ts`
- `docs/architecture/v0.9-desktop-shell-contract.md`
- `docs/architecture/api-surface.md`
- `docs/product/phase-backlog.md`

Deliverables:

- `GET /api/v1/desktop/service-control-gate`。
- Desktop shell service control panel。
- `open_dashboard` 保持 enabled link。
- `start_service`、`stop_service`、`restart_service`、`enable_notifications`、`tray_menu` 保持 disabled / blocked。
- `process_control_attempted=false`、`command_created=false`、`worker_scheduled=false` 等安全事实。

Gate:

- 不启动、停止或重启真实 service。
- 不创建 command request。
- 不调度 worker。
- 不写项目文件。
- 不维护第二状态源。
- gate panel 不代表 process supervision 已打开。

Risk: R0 read_only。

### AF-V09-004 Desktop Notification Gate

Phase: v0.9 Desktop Shell

Goal: 为 Desktop shell 提供只读 notification gate，展示 SSE 观察、系统通知、approval needed、
run failure 和 worker recovery 通知能力为什么仍禁用，以及后续打开所需的 event filter、redaction、
dedupe、rate limit、approval、OS permission 和 audit evidence。

Inputs:

- `internal/project/desktop_notification_gate.go`
- `internal/api/server.go`
- `desktop/src/main.ts`
- `docs/architecture/v0.9-desktop-shell-contract.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.9-desktop-shell.md`

Deliverables:

- `GET /api/v1/desktop/notification-gate`。
- Desktop shell notification gate panel。
- `observe_event_stream` 保持 read-only available。
- `enable_system_notifications`、`approval_needed_notifications`、`run_failure_notifications`、
  `worker_recovery_notifications` 保持 disabled / blocked。
- `event_stream_opened=false`、`notification_requested=false`、`command_created=false`、
  `worker_scheduled=false` 等安全事实。

Gate:

- 不建立 SSE 连接。
- 不请求 OS notification permission。
- 不创建 command request。
- 不调度 worker。
- 不把 notification state 维护成第二状态源。
- 不发送远程通知。
- notification gate panel 不代表 OS notification bridge 已打开。

Risk: R0 read_only。

### AF-V09-005 Desktop Tray Menu Gate

Phase: v0.9 Desktop Shell

Goal: 为 Desktop shell 提供只读 tray/menu gate，展示 dashboard、status、recent events、service control、
notification 和 settings 菜单项哪些可读展示、哪些仍禁用，以及后续打开所需的 OS integration、
permission、preflight、approval、audit 和 settings/secret UI contract。

Inputs:

- `internal/project/desktop_tray_menu_gate.go`
- `internal/api/server.go`
- `desktop/src/main.ts`
- `docs/architecture/v0.9-desktop-shell-contract.md`
- `docs/architecture/api-surface.md`
- `docs/milestones/v0.9-desktop-shell.md`

Deliverables:

- `GET /api/v1/desktop/tray-menu-gate`。
- Desktop shell tray menu gate panel。
- `open_dashboard`、`show_service_status`、`show_recent_events` 保持 ready / read-only。
- `start_service`、`stop_service`、`enable_notifications`、`open_settings` 保持 disabled / blocked。
- `tray_menu_created=false`、`os_integration_requested=false`、`command_created=false`、
  `service_control_attempted=false` 等安全事实。

Gate:

- 不创建 native tray/menu。
- 不请求 OS integration。
- 不创建 command request。
- 不执行 service control。
- 不请求系统通知权限。
- 不打开 secret 明文 settings。
- 不调度 worker。
- tray/menu gate panel 不代表 native tray/menu 已创建。

Risk: R0 read_only。

## v1.0 Stable Platform

### AF-V10-001 Backup Restore Integrity Chain

Phase: v1.0 Stable Platform

Goal: backup manifest、artifact integrity 和 restore dry-run 形成可恢复范围证据链。

Inputs:

- `internal/project/backup.go`
- `internal/project/artifact_integrity.go`
- `internal/project/restore_plan.go`
- `docs/architecture/artifact-backup-restore-contract.md`
- `docs/architecture/object-artifact-retention-contract.md`
- `docs/architecture/v1.0-stable-platform-contract.md`
- `docs/milestones/v1.0-stable-platform.md`

Deliverables:

- backup manifest。
- artifact integrity。
- restore dry-run plan。
- project_reference / external_project limitations。
- archive / retention preview remains metadata-only。
- object backend returns skipped / needs_attention until verifier exists。

Gate:

- 不读取外部 backup 包。
- 不写数据库。
- 不覆盖 project files。
- 不执行 restore apply。
- metadata-only history 不能被报告为完整可恢复。
- local artifact hash mismatch / missing blob 必须 blocked。
- object verifier skipped/failed 不能计入完整可恢复内容。
- `project_reference` / `external_project` 不因索引、hash 或 metadata 变成 AreaFlow-owned content。
- archive copy/upload、object storage upload、retention-aware GC、orphan cleanup 和 delete apply 不属于 v1.0。
- 普通 GC/delete 不得触碰 `audit`、`release`、受保护的 `run_evidence`、`external_ref`、`legal_hold`、
  unknown retention、hash mismatch local artifact 或 verifier skipped/failed object artifact。

Risk: R0 read_only；真实 restore apply 是 R4。

### AF-V10-002 Release Final Gate Chain

Phase: v1.0 Stable Platform

Goal: release readiness、remediation、acceptance、exception、final gate、evidence 和 preview 链完整。

Inputs:

- `internal/project/release_*.go`
- `docs/architecture/api-surface.md`
- `docs/architecture/release-final-gate-contract.md`
- `docs/architecture/v1.0-stable-platform-contract.md`
- `docs/milestones/v1.0-stable-platform.md`

Deliverables:

- release final gate / exception contract。
- release readiness。
- remediation plan。
- acceptance preview/gate。
- exception doctor/record/schema/migration/apply preview。
- final gate。
- evidence bundle。
- package/distribution/publish/rollout preview。

Gate:

- 不创建 release package。
- 不创建 migration。
- 不写 exception record。
- 不 tag/push/sign/upload/publish。
- release final gate `pass` 不代表真实发布，只允许进入 evidence / package / distribution / publish / rollout preview。
- release exception 只允许 `metadata_only_history`、`future_only_gap` 和 `archive_exception`。
- permission、adapter/profile、backup、local artifact integrity、secret、project isolation 或 protected path
  proof 失败不能通过 exception 放行。
- blocked / needs_decision 必须有 owner、required evidence 和 rollback path。

Risk: R0 read_only preview；exception apply / publish apply 是 R4。

### AF-V10-003 Adapter Profile Conformance

Phase: v1.0 Stable Platform

Goal: 证明 AreaMatrix adapter、AreaMatrix profile、plugin / marketplace seed 和 AreaFlow core 边界稳定。

Inputs:

- `internal/project/conformance.go`
- `internal/workflow/profile.go`
- `internal/adapter/areamatrix/**`
- `workflow/profiles/areamatrix/profile.yaml`
- `docs/architecture/adapter-profile-boundary.md`
- `docs/architecture/plugin-marketplace-boundary.md`
- `docs/architecture/v1.0-stable-platform-contract.md`

Deliverables:

- conformance API/CLI。
- profile load/hash/validate。
- 16 stage 顺序检查。
- gate contract 检查。
- adapter snapshot inventory read-only 检查。
- plugin / marketplace seed、manifest draft 和 no-execution boundary 文档。

Gate:

- adapter 不定义 workflow 状态机。
- profile 不读磁盘、不执行命令、不处理 secret。
- v1.0 plugin / marketplace 只允许 built-in / seed metadata 和 conformance。
- plugin 不安装、不启用、不执行、不远程拉取、不绕过 permission。

Risk: R0 read_only。

### AF-V10-004 AreaMatrix Execution Cutover Readiness

Phase: v1.0 Stable Platform

Goal: 证明 AreaMatrix dogfood 从 execution cutover readiness 走向 Archive / Shim Retirement 的完整关闭条件。

Inputs:

- `docs/architecture/v1.0-stable-platform-contract.md`
- `docs/migration/areamatrix-workflow-migration.md`
- `docs/migration/cutover-rollback-compat.md`
- `docs/migration/areamatrix-compatibility-shim-plan.md`
- `docs/development/implementation-gap-audit.md`
- `docs/development/execution-cutover-readiness-evidence.md`

Deliverables:

- `areaflow project execution-cutover-readiness <project> [--json]` 只读 readiness。
- `GET /api/v1/projects/{project}/execution-cutover-readiness` 只读 readiness。
- Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover -> Archive ->
  Shim Retirement 证据。
- compatibility command 转发或清晰 blocked。
- rollback 到 read-only / status projection 模式。
- shim lifecycle / retirement plan：`read_only_shim`、`execution_forwarding`、`retired_thin_entry` 的 go/no-go。

Gate:

- `./task-loop run` 在 execution cutover 前 blocked。
- readiness 必须保持 `execution_cutover_apply_open=false`、`task_loop_run_forwarded=false`、
  `project_write_attempted=false` 和 `execution_write_attempted=false`。
- execution cutover 后也必须先输出迁移说明或显式转发。
- shim retirement 必须晚于 execution forwarding 稳定期；retirement 不能删除 workflow/progress/log/evidence。
- AreaMatrix 只保留粗略入口和项目事实。
- readiness / preview 不能被当作 completion audit 的 dogfood complete 证据；必须有 cutover apply、archive
  和 retirement proof。

Risk: R3 execution / R4 if migration or security boundary changes。

### AF-V10-004A Execution Forwarding v1

Phase: v1.0 Stable Platform

Goal: 在不打开自动写代码的前提下，让 AreaMatrix `./task-loop run` 第一版只转发到 AreaFlow 受保护
Command API，并证明 AreaFlow 接管执行入口、run/task/attempt/artifact/audit 主状态。

Inputs:

- `docs/migration/areamatrix-execution-cutover-boundary.md`
- `docs/architecture/execution-opening-strategy.md`
- `docs/architecture/completion-audit-contract.md`
- `docs/architecture/v1.0-stable-platform-contract.md`
- `docs/migration/areamatrix-compatibility-shim-plan.md`
- `docs/development/execution-cutover-readiness-evidence.md`

Deliverables:

- `execution-forwarding-v1` Command API / CLI 设计或实现。
- `execution-forwarding-v1-readiness`、`execution-forwarding-v1-apply-preview` 和
  `execution-forwarding-v1-rollback-preview` 先作为只读 review surfaces 落地，真实 apply 另行开闸。
- `./task-loop run` forwarding target 只允许 read-only verify、doctor/readiness、artifact evidence、
  status/projection validation 和 release/readiness check 类任务。
- command response 暴露 legacy runner、legacy progress/log/checkpoint、project write、engine、secret、network
  和 publish/restore safety facts。
- proof facts 覆盖 command/run/run_task/run_attempt/artifact/audit 主状态由 AreaFlow 拥有。
- smoke 验证 `./task-loop run` 不启动旧 runner、不写旧 progress/log/checkpoint，并可 fail closed。
- rollback preview 能说明 fail-closed steps、reopen conditions 和 required proof facts；真实 rollback apply
  必须另行通过 Command API、approval、protected path proof 和 audit。

Gate:

- 必须已有 `read_only_shim`，且 AreaMatrix shim 文件真实落地需要单独跨仓库授权。
- 必须有 explicit execution cutover approval、Command API response、event、audit 和 rollback path。
- 只允许 read-only / evidence 类任务；copy-ready source write、generated retained write、repair、checkpoint、
  engine、secret、network、publish 和 restore 必须 fail closed。
- 旧 `workflow/versions/**/execution/**`、`progress.json`、logs、checkpoint 和 legacy runner 不得写入或启动。
- AreaMatrix protected path proof 必须 clean 或明确授权。
- Web/Desktop 只能展示 forwarding v1 readiness / proof，不得提供绕过 Command API 的按钮。

Risk: R3 execution cutover boundary；scope limited to read-only / evidence forwarding and does not open project write,
engine, secret, checkpoint, repair, publish or restore apply.

### AF-V10-005 Security Boundary Readiness

Phase: v1.0 Stable Platform

Goal: 证明 auth、team、API token、secret 和 remote worker credential 的长期模型已落地，但 v1.0 仍只做
schema / readiness / doctor / preview，不打开真实 enforcement、secret resolve 或 credential issuance。

Inputs:

- `docs/architecture/auth-team-secret-boundary.md`
- `docs/architecture/team-remote-control-boundary.md`
- `docs/architecture/security-permissions.md`
- `docs/architecture/data-model-v1.md`
- `docs/architecture/worker-scheduling-contract.md`
- `docs/architecture/v1.0-stable-platform-contract.md`
- `docs/milestones/v1.0-stable-platform.md`

Deliverables:

- security boundary readiness matrix。
- actor kind / principal / audit subject 说明。
- team membership 与 project membership 的 scope 说明。
- API token hash / scope / status readiness。
- secret_ref / engine_profile readiness。
- remote worker credential readiness / blocked reason。

Gate:

- 不创建、轮换或撤销真实 API token。
- 不根据 membership 改变 API authorization。
- 不解析 env、keychain、DB secret 或 external secret manager 明文。
- 不发放 remote worker credential。
- 不启动 remote worker、不直连 PostgreSQL。
- readiness 必须返回 `auth_enforcement_open=false`、`team_permission_enforcement_open=false`、
  `api_token_issuance_open=false`、`api_token_enforcement_open=false`、`secret_resolve_open=false`、
  `remote_worker_credentials_open=false` 和 `authorization_changed=false`。
- Team admin、Desktop、Web、CLI、worker 和 AreaMatrix shim 都不能绕过 project config、Command API、
  approval、secret scope 或 audit。

Risk: R0 read_only readiness for R4 auth/team/secret/remote-worker boundary。

### AF-V10-006 Completion Audit

Phase: v1.0 Stable Platform

Goal: 用只读 completion audit 聚合 0-100% 完成证据，证明 release final gate、AreaMatrix dogfood、task
matrix、implementation gap、operations readiness、project isolation 和 protected path proof 都满足 v1.0
范围。

Inputs:

- `docs/architecture/completion-audit-contract.md`
- `docs/architecture/v1.0-stable-platform-contract.md`
- `docs/product/master-plan.md`
- `docs/product/phase-backlog.md`
- `docs/development/implementation-gap-audit.md`
- `docs/development/task-backlog-status-audit.md`
- `docs/migration/areamatrix-execution-cutover-boundary.md`
- `docs/architecture/release-final-gate-contract.md`
- `docs/architecture/operations-deployment-observability-boundary.md`

Deliverables:

- `GET /api/v1/completion-audit` 只读 completion audit report。
- `areaflow completion audit --json` 只读 CLI report。
- phase/task matrix status。
- AreaMatrix dogfood completion status。
- release packaging preview status。
- operations readiness status。
- protected path proof status。
- v1.x deferred capability matrix。

Gate:

- release final gate `pass` 是必要条件，但不能单独声明 100%。
- completion audit 不运行 smoke、不写数据库、不写项目文件、不创建 release package、不 publish、不 restore。
- `preview_only`、`implemented_scoped` 和 `deferred` 必须被清楚解释。
- `execution-cutover-readiness` blocked 时，completion audit 必须 incomplete。
- AreaMatrix protected path proof 缺失或未授权变化时，completion audit 必须 blocked。
- 任一 v0-v1.0 必交付 task 缺 evidence 时，completion audit 必须 incomplete。

Risk: R0 read_only。

## v1.x Deferred Backlog

这些任务保留为 post-100%，不能作为 v1.0 必交付打开：

1. AF-V1X-001 Real Generated-only Rollback Beta：真实 managed project generated/projection 单文件写入演练后恢复 preimage。
2. AF-V1X-002 Real Generated-only Retained Apply：rollback beta 稳定后，允许真实 managed project generated/projection 单文件保留写入。
3. AF-V1X-003 Manual Patch Artifact：AreaFlow 只生成 source patch/diff artifact、write-set preview、expected-before、验证命令和 rollback/remediation plan。
4. AF-V1X-004 Human-applied Source Evidence：人工或现有 Codex 流程 apply 源码变更，AreaFlow 只读取 diff、changed hash 和验证结果。
5. AF-V1X-005 Source Write Beta：allowlist 内源码 `create` / `modify`，带 write-set、copy/verify、checkpoint preview 和 rollback。
6. AF-V1X-006 Checkpoint Apply：source write beta 稳定后单独打开，失败阻断下一 task。
7. AF-V1X-007 Repair Plan / Repair Apply：先生成 failure summary 和 repair plan artifact；repair apply 追加 attempt，不能跳过 verify 或 checkpoint gate。
8. AF-V1X-008 No-secret Engine Execution：`secret_ref=none` engine 执行 approved run_task。
9. AF-V1X-009 Secret Resolve：按 `docs/architecture/auth-team-secret-boundary.md`，提供 OS keychain、env binding、encrypted store 或外部 secret manager 的短期 scoped binding。
10. AF-V1X-010 Remote Worker：按 `docs/architecture/auth-team-secret-boundary.md`，实现远程 worker identity、token rotation、project scope、capability scope、lease scope 和 audit trail。
11. AF-V1X-011 Restore Apply：真实 restore apply，R4 migration_security。
12. AF-V1X-012 Release Exception Real Write：schema preview、migration approval gate、apply preview 和 R4 approval 后写 exception record。
13. AF-V1X-013 Release Publish Apply：tag、push、sign、upload、publish 拆成独立 Command API。
14. AF-V1X-014 Plugin Execution：按 `docs/architecture/plugin-marketplace-boundary.md` 完成第三方 plugin marketplace、manifest、signature、sandbox、disable/revoke 和受控执行。
15. AF-V1X-015 External Integrations And Webhooks：按 `docs/architecture/integration-webhook-boundary.md`
    完成 catalog/readiness、delivery plan preview、fixture outbound/inbound、project-scoped delivery、
    inbound callback beta、external API connector command 和 provider automation；禁止 callback 直接改状态、
    未知 endpoint delivery 和 external API 绕过 Command API。
16. AF-V1X-016 Team Console：按 `docs/architecture/team-remote-control-boundary.md` 完成 read-only team
    preview、local auth console、team permission enforcement、remote read-only console 和 remote command
    console；多用户、团队、权限、API token 和远程控制台都必须保留 project scope、token/session revoke、
    audit 和 Command API preflight。
17. AF-V1X-017 Object Artifact Store：按 `docs/architecture/object-artifact-retention-contract.md`
    逐级打开 object backend schema metadata、verifier preview、write/read fixture、scoped upload、
    restore dry-run integration、retention preview、archive copy/upload command、GC/delete preview 和
    GC/delete apply；第一版 delete 只能处理 AreaFlow-owned `ephemeral` artifacts。
18. AF-V1X-018 Budget And Quota：按 `docs/architecture/budget-quota-boundary.md` 完成
    metadata/readiness、estimate preview、quota policy doctor、fixture reservation/charge、
    project-scoped enforcement、team/actor/provider aggregation 和 provider billing reconciliation；
    engine cost、rate limit、budget policy、quota、override approval 和 audit 都必须稳定，禁止 silent
    throttling、重复扣费和无 project scope 阻断。
19. AF-V1X-019 Managed Ops / Upgrade / Support Export：按
    `docs/architecture/operations-deployment-observability-boundary.md` 逐级打开 remote read-only ops、
    remote ops control、managed upgrade/rollback 和 full support bundle export；必须具备 auth/team
    scope、redaction proof、destination allowlist、backup/preimage、approval、audit、retention 和 revoke path。

v1.x deferred task 的编号顺序就是建议开闸顺序。任一任务进入 active 前，都必须先补齐 task-local
design gate、Command API、capability preflight、approval、rollback / remediation、audit 和 focused
smoke；不能用 v1.0 final gate 或上一个 v1.x task 的证据替代当前任务的 apply 证据。
统一状态词、apply packet、suspension rule、R4 串行原则和 AreaMatrix first policy 见
`docs/architecture/high-risk-apply-ladder.md`；任一 v1.x task 进入 active 前必须引用对应 rung。

## 推荐推进顺序

1. AF-P0-001 Design Source Alignment。
2. AF-P0-002 Directory Boundary Audit。
3. AF-P0-003 Governance Boundary Audit。
4. AF-P0-004 Operations Deployment Observability Boundary。
5. AF-V01-001 PostgreSQL Bootstrap Smoke。
6. AF-V01-002 AreaMatrix Adapter Metadata Import。
7. AF-V01-003 Guarded Status Projection。
8. AF-V02-001 Doctor And Readiness Bundle。
9. AF-V02-002 Native Doctor Authorization Boundary。
10. AF-V03-001 Workflow Version Authoring Model。
11. AF-V03-002 Gate Transition Approval Records。
12. AF-V04-001 Compatibility And Shim Readiness。
13. AF-V04-002 Authoring Cutover Apply。
13. AF-V05-001 Runner Preview Evidence。
14. AF-V05-002 Run Control Dry-run Boundary。
15. AF-V06-001 Worker Registry Lease Lifecycle。
16. AF-V06-002 Codex CLI Adapter Preview。
17. AF-V06-003 Execution Approval Gate。
18. AF-V06-004 Fixture Execution Apply。
19. AF-V06-005 Read-only Verify。
20. AF-V06-006 Approved Artifact Write。
21. AF-V06-007 Execution Plan Preview。
22. AF-V06-008 Approved Project Write Design Gate。
23. AF-V06-009 Fixture-only Approved Project Write。
24. AF-V06-010 Managed Generated Write Gate。
25. AF-V06-011 Managed Generated Write Apply Core + API CLI Surfacing。
26. AF-V06-012 Generated Write Dogfood Readiness。
27. AF-V06-013 Generated Write Apply Beta Approval Gate。
28. AF-V07-001 Read-only Dashboard Coverage。
29. AF-V07-002 Web Write Action Gate。
30. AF-V08-001 Schedule Preview。
31. AF-V08-002 Engine Secret Readiness Boundary。
32. AF-V09-001 Local Service Status Contract。
33. AF-V09-002 Tauri Shell Scaffold。
34. AF-V09-003 Desktop Service Control Gate。
35. AF-V09-004 Desktop Notification Gate。
36. AF-V09-005 Desktop Tray Menu Gate。
37. AF-V10-001 Backup Restore Integrity Chain。
38. AF-V10-002 Release Final Gate Chain。
39. AF-V10-003 Adapter Profile Conformance。
40. AF-V10-004 AreaMatrix Execution Cutover Readiness。
40.5. AF-V10-004A Execution Forwarding v1。

v1.x 任务只能在 v1.0 证据链完成后按顺序进入 active，并且每个任务必须先补设计门禁、approval、
rollback / remediation 和 audit 证据。

后续每推进一个阶段，都应更新 [`../../docs/development/implementation-gap-audit.md`](../evidence/implementation-gap-audit.md)，记录实现证据和剩余缺口。
