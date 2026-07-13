# Completion Audit Contract

## Purpose

本文定义 AreaFlow 从 0% 走到 v1.0 100% 稳定平台时，怎样证明“真的完成”。它补充
[`v1.0-stable-platform-contract.md`](v1.0-stable-platform-contract.md)、
[`release-final-gate-contract.md`](release-final-gate-contract.md)、
[`operations-deployment-observability-boundary.md`](operations-deployment-observability-boundary.md)、
[`high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md) 和
[`../history/v1.0/evidence/implementation-gap-audit.md`](../evidence/implementation-gap-audit.md)。

Completion audit 是最终完成审计，不是新的 apply 权限。它只聚合当前状态证据、命令输出、smoke 结果、
release preview 链、AreaMatrix dogfood 状态和 backlog 状态。它不创建 release package、不 publish、不
restore、不修改 AreaMatrix、不打开 secret、remote worker、plugin、managed ops 或任何 v1.x 高风险能力。

## Core Rule

`release final gate = pass` 是 v1.0 完成的必要条件，但不是充分条件。

AreaFlow 达到 100% 必须同时证明：

```text
phase evidence complete
AreaMatrix dogfood cutover complete
release final gate pass
release package / distribution / publish / rollout previews remain read-only
operations readiness complete
project_key isolation proven
task backlog status has no v0-v1.0 blocker
implementation gap audit has no hidden v1.0 blocker
v1.x high-risk capabilities remain explicitly deferred
AreaMatrix protected paths proof clean
```

任一项缺失、未知、只由间接证据支撑或范围不匹配时，completion audit 必须返回 `incomplete` 或
`blocked`，不能因为某个 smoke 绿色或某个 preview `ready` 就标记 100%。

Completion audit 只能消费目标 `areamatrix` project-scoped proof。其它 project 的最新 proof、
operations smoke proof 或 evidence URI 不能 shadow、替代或关闭真实 AreaMatrix 的 E1-E9 证据项。

## Status Vocabulary

Completion audit 使用独立状态词：

```text
complete:
  当前证据证明该项在 v1.0 范围内已完成。

incomplete:
  能力方向正确，但仍缺实现、dogfood、smoke、gate、evidence 或 cross-repo proof。

blocked:
  存在不可放行 blocker，或证据显示当前状态与 v1.0 不变量冲突。

deferred:
  明确属于 v1.x，不计入 v1.0 完成缺口，但必须有边界合同和禁用证明。

not_applicable:
  当前 v1.0 范围不要求该项。
```

`deferred` 只能用于已经写入 v1.x high-risk ladder 的能力。不能把 v1.0 必交付项改名成 `deferred`。

## Evidence Classes

Completion audit 必须按证据类别逐项检查。

所有能让 audit 项进入完成态的 proof record 都必须带可追溯证据说明：`complete` proof、operations
`pass` proof、protected path `clean` / `authorized` proof 必须同时提供非空 `--summary` 和
`--evidence-uri`。缺少 required facts 时仍优先报告 fact 缺口；facts 齐全但没有 summary 或 evidence URI
时，record command 必须在写入数据库前拒绝，防止空白审查说明被封存为完成证据。

### E1 Design Source Alignment

必须证明以下源事实互相一致：

```text
docs/product/master-plan.md
docs/product/platform-blueprint.md
docs/product/phase-backlog.md
docs/product/roadmap.md
docs/milestones/README.md
docs/architecture/*
docs/migration/*
tasks/backlog/0-100-platform-backlog.md
docs/development/task-backlog-status-audit.md
docs/development/implementation-gap-audit.md
```

通过标准：

- 0-100% 阶段、v1.0 范围、v1.x deferred 顺序一致。
- release、restore、secret、remote worker、plugin、integration、team console、object artifact、budget/quota、
  managed ops 的边界一致。
- `preview_only`、`implemented_scoped`、`deferred` 没有被描述成真实 apply。

E1 source alignment proof 的受控输入是：

```bash
areaflow completion source-alignment-proof record areamatrix \
  --status complete \
  --fact zero_to_hundred_phases_aligned \
  --fact v1_and_v1x_boundaries_consistent \
  --json
```

真实 `complete` proof 必须包含 E1 列出的全部 required facts，并自动绑定当前 AreaFlow 设计源文件集
（`docs/product/**` 的核心计划文件、`docs/architecture/*.md`、`docs/migration/*.md`、
`docs/milestones/README.md`、task backlog 和 implementation/task audit 文档）的路径、逐文件 sha256、
source-set hash、文件数以及 missing/unreadable 计数。该命令只读这些 AreaFlow 源文件并只写 AreaFlow
`events`、`audit_events` 和 `command_requests`；不改写文档、不执行 shell、不写 AreaMatrix、不触碰
protected paths。Completion audit 会重新计算 current source binding，并拒绝旧 facts-only proof 或
post-proof source drift。Source alignment proof 只能关闭 E1 的设计源事实一致性证据缺口，不能代替 task
matrix、validation、dogfood cutover、release、security/isolation 或 protected path proof。

### E2 Phase And Task Matrix

必须从 task backlog 和 task status audit 证明：

- 所有 v0-v1.0 task 有状态、证据和剩余边界。
- `planned` / `preview_only` / `implemented_scoped` 项没有被隐藏。
- 最靠前未关闭 task 明确列出 owner / next command / required evidence。
- v1.x deferred task 不计入 v1.0 缺口，但不能缺合同。

如果 `task-backlog-status-audit.md` 中任一 v0-v1.0 task 仍是 `planned` 且属于 v1.0 必交付，completion
audit 必须 `incomplete`。

E2 task matrix proof 的受控输入是：

```bash
areaflow completion task-matrix-proof record areamatrix \
  --status complete \
  --fact all_v0_v1_tasks_have_status_evidence_and_boundary \
  --fact no_planned_v1_required_task_hidden \
  --source-set-hash <sha256> \
  --backlog-hash <sha256> \
  --task-status-audit-hash <sha256> \
  --planned-v1-required-task-count 0 \
  --missing-evidence-v1-required-task-count 0 \
  --blocked-v1-required-task-count 0 \
  --json
```

真实 `complete` proof 必须包含 E2 列出的全部 required facts，并绑定当前 task matrix source set：
`tasks/backlog/0-100-platform-backlog.md`、`docs/development/task-backlog-status-audit.md`、两个文件的
sha256、source-set hash，以及 `planned_v1_required_task_count=0`、
`missing_evidence_v1_required_task_count=0`、`blocked_v1_required_task_count=0`。该命令只写 AreaFlow
`events`、`audit_events` 和 `command_requests`；它校验外部 review 传入的 binding metadata 形状，但不扫描或改写
backlog/status audit、不执行 shell、不写 AreaMatrix、不触碰 protected paths。Completion audit 会只读重算当前
backlog/status-audit binding；旧 facts-only proof、binding 缺失、hash 漂移或任一 required count 非 0 都必须保持
E2 blocked。Task matrix proof 只能关闭 E2 的 phase/task matrix 证据缺口，不能代替设计源事实、validation、
dogfood cutover、release、security/isolation 或 protected path proof。

### E3 Command, API, And Smoke Evidence

必须有可复验命令或 smoke 证据：

```text
go test ./...
go build ./cmd/areaflow
cd web && npm run build
git diff --check -- .
AREAFLOW_DATABASE_URL=... ./scripts/smoke-v1-stable-fixture.sh
AREAFLOW_DATABASE_URL=... ./scripts/smoke-web.sh
AREAFLOW_DATABASE_URL=... ./scripts/smoke-project-isolation.sh
```

如果某个 smoke 需要真实 PostgreSQL、browser 或 local service，completion audit 必须记录数据库名、
时间、scope、清理结果和失败时的 blocker。没有运行不能写 pass，只能写 `missing_evidence`。

E3 validation proof 的受控输入是：

```bash
areaflow completion validation-proof record areamatrix \
  --status complete \
  --fact go_test_passed \
  --fact go_build_passed \
  --validation-command "go test ./..." \
  --validation-result-hash <sha256> \
  --validation-started-at <rfc3339> \
  --validation-finished-at <rfc3339> \
  --validation-scope <scope> \
  --json
```

真实 `complete` proof 必须包含 E3 列出的全部 required facts，并绑定 reviewed validation output：实际验证命令清单、
sha256 结果摘要、started/finished RFC3339 时间窗和 validation scope。该命令只写 AreaFlow `events`、
`audit_events` 和 `command_requests`；它校验这些 binding metadata 的形状，但不替用户运行测试、构建、
browser smoke 或 PostgreSQL smoke，不执行 shell、不写 AreaMatrix、不触碰 protected paths。Validation proof
只能关闭 E3 的 validation evidence 缺口，不能代替 release gate、dogfood cutover、protected path proof 或
E1/E2/E8 closure。

### E4 AreaMatrix Dogfood Completion

AreaMatrix 是第一 dogfooding project。100% 必须证明：

```text
Import
-> Mirror
-> Shadow
-> Authoring Cutover
-> Execution Beta
-> Execution Cutover
-> Archive
-> Shim Retirement
```

完成证据必须包含：

- Import coverage 与 drift / doctor 等价证据。
- 新 workflow version authoring 源事实归 AreaFlow。
- Compatibility shim lifecycle 从 `not_installed` 到目标状态的证据。
- Execution beta / execution cutover 不是只读 readiness，而是真实 approval / command / audit 通过。
- Execution Forwarding v1 如果作为 v1.0 的 execution cutover 证据，必须证明 `./task-loop run`
  只转发 read-only verify、doctor/readiness、artifact evidence、status/projection validation 或 release/readiness
  check 类任务，并且 source write、generated retained write、repair、checkpoint、engine、secret、network、
  publish 和 restore 仍保持关闭。
- Archive gate 证明历史 workflow / execution metadata 已由 AreaFlow 索引，旧 progress / logs / checkpoints
  只作为 reference，且没有删除、移动、重写历史文件。
- `./task-loop run` 的转发、阻断或 retirement 行为有命令映射和 rollback 证据。
- AreaMatrix 最终只保留粗略入口、项目事实、compatibility commands 和历史归档说明。

如果缺少受控 Execution Cutover proof，completion audit 必须 `incomplete`。只读
`execution-cutover-readiness` 可以继续保持 `blocked`、`preview_only` 和
`execution_cutover_apply_open=false`，因为它不是 apply 命令；它只能提供 readiness 上下文，不能代替
显式 cutover approval / command response / event / audit / rollback 证据。
如果 Execution Cutover proof 只证明 `Execution Forwarding v1`，completion audit 必须把该 scope
记录为入口/状态/审计接管完成，而不能把它解释为 copy-ready source write、generated retained apply、repair、
checkpoint、engine execution、secret、network、publish 或 restore 已打开。
如果 Archive gate 缺少 immutable historical index、metadata-only artifact reference 限制、protected path
proof，或缺少受控 scope binding，completion audit 也必须 `incomplete`。

E4 三类 `complete` proof 还必须带 proof-level release-candidate review evidence：非空 `--summary`、
路径带 `release-candidate` / `release_candidate` 语义的 `--evidence-uri`、`--review-decision approved`、
非空 `--reviewed-by` 和 RFC3339 `--reviewed-at`。`local:`、`fixture:`、`script:`、`scripts/**`、
`smoke-*` 或带 fixture/mock/demo/sample/synthetic/testdata/placeholder/dummy/example marker 的 URI 不能关闭
E4；completion audit 只有在目标 project 同时具备真实 AreaMatrix identity 后才消费这些 E4 proof。

Archive gate proof 的受控输入是：

```bash
areaflow completion archive-proof record areamatrix \
  --status complete \
  --fact historical_workflow_versions_marked_immutable \
  --fact historical_execution_metadata_indexed_in_areaflow \
  --fact historical_artifact_refs_have_hash_path_type_project_version_run \
  --fact project_reference_restore_limitations_recorded \
  --fact old_progress_logs_checkpoints_are_reference_only \
  --fact new_run_attempt_artifact_audit_state_owned_by_areaflow \
  --fact areamatrix_workflow_readme_summary_contract_reviewed \
  --fact areamatrix_status_json_rough_projection_contract_reviewed \
  --fact archive_does_not_delete_or_move_historical_files \
  --fact archive_does_not_rewrite_progress_json \
  --fact rollback_to_execution_forwarding_documented \
  --summary "real release candidate archive evidence reviewed" \
  --evidence-uri docs/development/real-release-candidate-evidence.md#archive-gate \
  --review-decision approved \
  --reviewed-by release-owner \
  --reviewed-at 2026-07-04T12:00:00Z \
  --archive-scope areamatrix_historical_execution_reference_only \
  --archive-reference-mode metadata_indexed_reference_only \
  --archive-source-path .areaflow/status.json \
  --archive-source-path workflow/README.md \
  --archive-source-path 'workflow/versions/**/execution/**' \
  --archive-source-path 'workflow/versions/**/execution/_shared/progress.json' \
  --archive-forbidden-action copy_artifact_bytes \
  --archive-forbidden-action delete_artifact_bytes \
  --archive-forbidden-action delete_historical_files \
  --archive-forbidden-action move_historical_files \
  --archive-forbidden-action rewrite_progress_json \
  --archive-forbidden-action run_commands \
  --archive-forbidden-action write_areamatrix_protected_paths \
  --archive-rollback-target execution_forwarding_read_only_shim \
  --archive-fail-closed \
  --json
```

真实 `complete` proof 必须包含 Archive gate 合同列出的全部 required facts，并绑定上述 archive scope、
reference mode、source path set、forbidden action set、rollback target 和 fail-closed policy；这些字段必须归一化为
deterministic `archive_scope_binding_hash`，并在 record JSON、event、audit event 与 completion audit metadata 中暴露。
completion audit 必须重算当前 Archive binding，要求 `archive_scope_current_binding_bound=true` 且 latest proof
`event_id > 0` 后才可消费该 proof。缺少 binding 的旧 facts-only / loose metadata proof、hash 缺失/漂移或缺少
event ID 都会在 completion audit 中保持 incomplete。
该命令只写 AreaFlow `events`、`audit_events` 和 `command_requests`；不复制 artifact bytes、不删除历史文件、
不重写 `progress.json`、不执行 shell、不写 AreaMatrix、不触碰 protected paths。Archive proof 不能代替
Execution Cutover 或 Shim Retirement proof。

Shim Retirement gate proof 的受控输入是：

```bash
areaflow completion shim-retirement-proof record areamatrix \
  --status complete \
  --fact archive_gate_passed \
  --fact execution_forwarding_stable_for_declared_window \
  --fact no_legacy_task_loop_run_usage_in_active_workflow_versions \
  --fact areaflow_run_attempt_artifact_audit_coverage_pass \
  --fact compat_commands_mapped_or_deliberately_blocked \
  --fact legacy_progress_log_checkpoint_archive_reference_policy_accepted \
  --fact rollback_to_read_only_shim_documented \
  --fact user_facing_retirement_notice_present \
  --fact protected_path_proof_reference_recorded \
  --summary "real release candidate shim retirement evidence reviewed" \
  --evidence-uri docs/development/real-release-candidate-evidence.md#shim-retirement-gate \
  --review-decision approved \
  --reviewed-by release-owner \
  --reviewed-at 2026-07-04T12:00:00Z \
  --shim-retirement-scope read_only_shim_retirement_after_execution_forwarding_v1 \
  --shim-prerequisite archive_gate_passed \
  --shim-prerequisite execution_cutover_gate_passed \
  --shim-prerequisite protected_path_proof_recorded \
  --shim-retired-surface legacy_task_loop_runner \
  --shim-retired-surface legacy_progress_json_writes \
  --shim-retired-surface legacy_logs_writes \
  --shim-retired-surface legacy_checkpoint_writes \
  --shim-rollback-target read_only_shim \
  --shim-fail-closed \
  --shim-reopen-requires-approval \
  --json
```

真实 `complete` proof 必须包含 Shim Retirement gate 合同列出的全部 required facts，并绑定 retirement
scope、required prerequisites、retired surface set、rollback target、fail-closed policy 和 reopen approval
policy；这些字段必须归一化为 deterministic `shim_retirement_scope_binding_hash`，并在 record JSON、event、
audit event 与 completion audit metadata 中暴露。completion audit 必须重算当前 Shim Retirement binding，
要求 `shim_retirement_scope_current_binding_bound=true` 且 latest proof `event_id > 0` 后才可消费该 proof。
缺少 binding 的旧 facts-only / loose metadata proof、hash 缺失/漂移或缺少 event ID 都会在 completion audit 中
保持 incomplete。该命令只写 AreaFlow `events`、`audit_events` 和 `command_requests`；不编辑 AreaMatrix
commands、不启动或停用旧 runner、不写旧 progress / logs / checkpoint、不删除历史文件、不执行 shell、
不写 AreaMatrix、不触碰 protected paths。Shim Retirement proof 不能代替 Execution Cutover approval /
command / audit proof。

Execution Cutover proof 的受控输入是：

```bash
areaflow completion execution-cutover-proof record areamatrix \
  --status complete \
  --fact explicit_execution_cutover_approval_recorded \
  --fact execution_cutover_command_response_recorded \
  --fact execution_cutover_event_and_audit_recorded \
  --fact task_loop_run_forwarding_window_proven \
  --fact rollback_plan_and_compatibility_window_proven \
  --fact no_unapproved_project_or_execution_write_attempted \
  --summary "real release candidate execution cutover evidence reviewed" \
  --evidence-uri docs/development/real-release-candidate-evidence.md#execution-cutover-gate \
  --review-decision approved \
  --reviewed-by release-owner \
  --reviewed-at 2026-07-04T12:00:00Z \
  --execution-cutover-scope execution_forwarding_v1_read_only_evidence_only \
  --allowed-task-types read_only_verify,doctor_readiness,artifact_evidence,status_projection_validation,release_readiness_check \
  --forbidden-actions start_legacy_task_loop_runner,write_legacy_progress_json,write_legacy_logs,write_legacy_checkpoint,write_areamatrix_source,write_areamatrix_execution_directory,generated_retained_write,repair_apply,checkpoint_apply,engine_execution,secret_resolve,network_api_integration,publish_apply,restore_apply \
  --rollback-target read_only_shim \
  --rollback-mode fail_closed_to_read_only_shim \
  --fail-closed \
  --reopen-requires-approval \
  --json
```

真实 `complete` proof 必须包含 Execution Cutover gate 合同列出的全部 required facts，并绑定
`execution_forwarding_v1_read_only_evidence_only` scope、完整 allowed task type 集合、完整 forbidden action
集合、`read_only_shim` rollback target、`fail_closed_to_read_only_shim` rollback mode、fail-closed 断言和
reopen-requires-approval 断言。缺少这些 binding metadata，或打开 source write、generated retained write、
repair、checkpoint、engine、secret、network、publish、restore 中任一能力时，record command 必须在打开数据库
事务前拒绝；completion audit 也必须把旧的 loose proof metadata 视为 incomplete。该命令只写
AreaFlow `events`、`audit_events` 和 `command_requests`；不转发 `./task-loop run`、不写
`workflow/versions/**/execution/**`、不重写旧 `progress.json`、不写旧 logs / checkpoint、不调用
engine、不执行 shell、不写 AreaMatrix、不触碰 protected paths。Execution Cutover proof 不能代替 Archive
或 Shim Retirement proof。

### E5 Release And Packaging Preview

Release 链路必须证明：

```text
release readiness
release remediation plan
release acceptance preview
release acceptance gate
release exception doctor
release exception record preview
release exception schema preview
release exception migration approval gate
release exception apply preview
release final gate
release evidence bundle
release package preview
release distribution preview
release publish gate
release publish approval preview
release rollout plan preview
```

v1.0 允许这些 preview / gate 作为完成证据，但必须同时证明：

- `release final gate` 不创建 package、不写 exception record、不运行 migration、不 publish。
- `release evidence bundle` 只聚合 evidence index，不读取 artifact 原文。
- `release package preview` 不生成压缩包。
- `distribution preview` 不上传、不签名、不 tag、不 push。
- `publish gate` / `publish approval preview` 不创建 approval、不发布。
- `rollout plan preview` 不创建 rollout state。

真实 package、tag、sign、upload、push、publish 是 v1.x `publish apply`，不能作为 v1.0 完成要求。

E5 release packaging proof 的受控输入是：

```bash
areaflow completion release-packaging-proof record areamatrix \
  --status complete \
  --fact release_final_gate_passed \
  --fact release_package_preview_created_no_package \
  --json
```

真实 `complete` proof 必须包含 E5 列出的全部 required facts。该命令只写 AreaFlow `events`、
`audit_events` 和 `command_requests`；不运行 release final gate、evidence bundle、package preview、
distribution preview、publish gate 或 rollout preview，不创建 release package、不写 release state、不创建
approval / rollout、不 tag/sign/upload/push/publish、不运行 migration、不写 AreaMatrix、不触碰 protected
paths。Release packaging proof 只能关闭 E5 的 release preview 证据缺口，且 E5 仍必须同时看到当前
`release final gate = pass`；proof 不能代替真实 publish/package/apply，也不能单独让 E5 完成。

### E6 Backup, Restore, Artifact, And Retention

必须证明：

- backup manifest 覆盖 PostgreSQL metadata 和 AreaFlow-owned artifact metadata。
- artifact integrity 区分 `local`、`project_reference`、`external_project` 和 `object`。
- restore dry-run 明确哪些内容可恢复，哪些是 metadata-only。
- object verifier skipped / failed 时不能计入完整可恢复内容。
- archive preview 不复制、不删除、不上传 artifact 原文。
- GC/delete 只在 v1.x 单独打开，v1.0 不触碰 protected retention classes。

`project_reference` 或 `external_project` 只要仍是 metadata-only，就必须进入 `needs_attention` 或显式
accepted exception；不能被 report 成完整 restore-ready。

E6 backup restore proof 的受控输入是：

```bash
areaflow completion backup-restore-proof record areamatrix \
  --status complete \
  --fact backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata \
  --fact restore_dry_run_identifies_metadata_only_history_and_object_verifier_limits \
  --backup-manifest-hash <sha256> \
  --backup-manifest-status ready \
  --restore-plan-status ready|needs_attention \
  --restore-plan-scope project \
  --restore-plan-project-key areamatrix \
  --restore-plan-manifest-hash <same-sha256> \
  --artifact-integrity-status pass|warn \
  --artifact-integrity-failed-count 0 \
  --artifact-archive-preview-status ready|needs_attention \
  --artifact-archive-preview-project-write-attempted false \
  --artifact-archive-preview-storage-write-attempted false \
  --artifact-archive-preview-delete-attempted false \
  --json
```

真实 `complete` proof 必须包含 E6 列出的全部 required facts，并绑定目标 release project 的当前
scoped backup manifest、scoped restore plan、artifact integrity 和 metadata-only archive preview 的状态、
hash/count 与 no-write safety facts。全局 backup / restore diagnostics 继续保留，但其他 fixture 或 stale
project 的 artifact failure 不能替代目标 project 的 E6 状态。缺少绑定、
manifest hash 不一致、artifact integrity failed count 非 0、archive preview 需要未定义 retention policy，或
archive preview 试图写项目/写 storage/delete artifact 时，`RecordBackupRestoreProof` 会在进入事务前 fail
closed；completion audit 也会把旧的 loose proof 视为 incomplete。该命令只写 AreaFlow `events`、
`audit_events` 和 `command_requests`；不执行 database restore、不复制/删除/上传 artifact bytes、不运行 GC、不写
AreaMatrix、不触碰 protected paths。Backup restore proof 是关闭 E6 的必要条件；release readiness / restore
plan ready 只能作为上下文，不能单独完成 E6。该 proof 不能代替真实 restore apply、artifact
archive/copy/delete/GC apply 或 release final gate。

### E7 Operations Readiness

必须证明：

- install / migrate / start / register smoke 可复验。
- health、readiness、doctor、service status 分层清楚。
- support bundle preview 是 metadata-only 且 redacted。
- telemetry 默认 local-only。
- migration ledger 记录 preflight / apply / verify / remediation。
- Desktop service-control / notification / tray-menu gate 保持只读或明确 disabled。

如果 support bundle preview 会导出 prompt、secret、用户文件、raw artifact 或未脱敏日志，completion audit
必须 `blocked`。如果 telemetry 默认远程上传，completion audit 必须 `blocked`。

### E8 Security, Permission, And Isolation

必须证明：

- `project_key` 隔离覆盖 workflow、run、lease、artifact、secret、audit 和 global ID route guard。
- permission doctor 证明默认只读、deny 优先、allowlist、command deny 和 secret/network/git 边界。
- audit coverage 覆盖已启用能力。
- auth/team/API token/secret/remote worker credential 只处于 schema/readiness/preview，除非后续 v1.x
  apply packet 明确打开。

Security readiness 不能读取 secret 明文、创建 token、改变 API authorization 或发放 remote worker credential。

E8 security closure proof 的受控输入是：

```bash
areaflow completion security-closure-proof record areamatrix \
  --status complete \
  --fact project_key_isolation_covers_workflow_run_lease_artifact_secret_audit \
  --fact global_id_route_guard_project_key_visibility_proven \
  --json
```

真实 `complete` proof 必须包含 E8 列出的全部 required facts，并绑定当前只读 security boundary
readiness、permission doctor 和 project-scoped audit coverage 摘要。该命令只写 AreaFlow `events`、
`audit_events` 和 `command_requests`；binding 采集只读 Store 状态，不运行 project isolation smoke 或 shell，
不读取 secret、不改 authorization、不发放 remote worker credential、不写 AreaMatrix、不触碰 protected
paths。Completion audit 必须重新计算当前 binding；如果旧 loose proof 缺 binding、permission doctor 或
audit coverage 漂移、或 `security boundary readiness` 显示 secret resolve、remote worker credential、
authorization 等 v1.0 forbidden capability 被打开，E8 必须保持 `blocked`。

### E9 AreaMatrix Protected Path Proof

Completion audit 必须保存 AreaMatrix protected path proof：

```bash
git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- \
  workflow/README.md \
  .areaflow/status.json \
  scripts/task_loop/console.py \
  scripts/dev_tools/cli.py \
  scripts/task_loop/runner.py \
  scripts/areaflow_shim.py \
  workflow/versions \
  workflow/versions/v1-mvp/execution/_shared/progress.json
```

AreaFlow 不替用户或 smoke 自动执行这条 `git status` 命令。命令输出必须由外部检查或人工 review 提供，
再通过受控 proof record 写入 AreaFlow 审计：

```bash
areaflow completion protected-path-proof record areamatrix \
  --status clean \
  --summary "AreaMatrix protected path git status returned no output" \
  --evidence-uri local:areamatrix-protected-path-git-status \
  --json
```

该 record command 只写 AreaFlow `events`、`audit_events` 和 `command_requests`，不运行 `git status`、
不执行 shell、不写 AreaMatrix、不读取用户文件、不启动 worker。`clean` proof 不允许携带非空
git status output，并且必须绑定当前 E9 protected path 集合：proof metadata 必须包含
`protected_path_set`、`protected_path_set_hash`、`protected_path_set_count`、`git_status_output_empty=true`、
`git_status_output_hash=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`、
`protected_path_proof_binding_status=pass` 和空
`protected_path_proof_binding_blockers`。缺少这些当前绑定字段的旧 loose clean proof 不能关闭 E9。
`dirty` / `blocked` proof 必须让 completion audit 保持 blocked。
`authorized` proof 只允许在有明确 cross-repo 授权包时关闭 E9，并且必须同时记录 `approval_id`、
`allowed_paths`、原始 reviewed `git_status_output`、`dirty_output_hash`、`reviewer` 和
`rollback_evidence_uri`。`allowed_paths` 必须是安全相对路径，并且位于 E9 的 AreaMatrix protected path
集合内；原始 git status output 的 sha256 必须等于 `dirty_output_hash`，且 output 中每个 touched path
都必须被 `allowed_paths` 覆盖。`authorized` proof 同样必须带当前 protected path set/hash/count 绑定，
且 `git_status_output_empty=false`、reviewed output hash 与 `dirty_output_hash` 一致，否则不能关闭 E9。示例：

```bash
areaflow completion protected-path-proof record areamatrix \
  --status authorized \
  --summary "AreaMatrix protected path changes approved by release owner" \
  --evidence-uri docs/development/protected-path-authorization.md \
  --git-status-output " M workflow/README.md" \
  --approval-id approval-123 \
  --allowed-path workflow/README.md \
  --dirty-output-hash <sha256-of-reviewed-git-status-output> \
  --reviewer release-owner \
  --rollback-evidence-uri docs/development/protected-path-rollback.md \
  --json
```

Completion audit 只消费目标 `areamatrix` 项目的 latest protected-path proof；其它 project 的更新 proof
不能遮挡、替代或关闭 E9。对于 `clean` / `authorized` proof，completion audit 必须重新检查
`protected_path_proof_binding_status`、`protected_path_set_hash`、`protected_path_set_count`、
`git_status_output_empty`、`git_status_output_hash` 和 `protected_path_proof_binding_blockers`；
binding 缺失、过期或不匹配时返回 `protected_path_proof_binding_incomplete`。
不再保留 option-only protected path status 输入；E9 不能通过 report option 替代 proof record 变成
`complete`。

只有在已进入明确 AreaMatrix cross-repo authorization task 时，允许这些路径变化；否则 completion audit
必须保持无输出。任何历史 workflow、progress、logs、checkpoint、release evidence 或用户文件删除都必须
`blocked`。

## Completion Audit Report Shape

当前提供只读 report：

```text
GET /api/v1/completion-audit
areaflow completion audit --json
```

第一版 report shape：

```text
status = complete | incomplete | blocked
mode = read_only_completion_audit
generated_at
scope = v1.0
items[]
items[].key
items[].category
items[].status
items[].evidence_refs[]
items[].required_evidence[]
items[].blocked_by[]
items[].next_command
release_final_gate_status
area_matrix_dogfood_status
task_matrix_status
implementation_gap_status
protected_path_proof_status
deferred_v1x[]
safety_facts.read_only=true
safety_facts.release_package_created=false
safety_facts.publish_attempted=false
safety_facts.restore_apply_attempted=false
safety_facts.secret_resolved=false
safety_facts.remote_worker_credentials_issued=false
safety_facts.area_matrix_protected_paths_touched=false
```

该 report 不写数据库、不写项目文件、不创建 audit event、不运行 smoke、不启动 worker、不执行命令。

完成审计结果的持久封存使用单独 command：

```bash
areaflow completion audit-snapshot record areamatrix \
  --release-candidate v1.0-rc1 \
  --evidence-class release_candidate \
  --evidence-uri docs/development/real-release-candidate-evidence.md \
  --summary "real release candidate evidence reviewed" \
  --review-decision approved \
  --reviewed-by release-owner \
  --reviewed-at 2026-07-04T12:00:00Z \
  --json
```

`completion audit-snapshot record` 只允许为目标 `areamatrix` project 记录，且只允许在当前 completion audit
`status=complete` 时记录。它保存
audit status、scope、hash、release candidate label、evidence class、evidence URI、proof event id、command
response、event 和 audit event。`evidence_class=fixture` 是默认值，用于隔离 smoke 和机制证明；
`evidence_class=release_candidate` 只能表示调用者声明该 snapshot 绑定真实 release candidate 证据；它必须绑定
真实 AreaMatrix project identity：`project_key=areamatrix`、root 为
`/Users/as/Ai-Project/project/AreaMatrix`、adapter/profile 为 `areamatrix`、kind 为 `product-repo`、
default branch 为 `main`，且 root/kind
不得带 fixture/temp 标记。它还必须提供非 fixture、非本地脚本 / smoke wrapper 的 evidence URI 和 summary，且
sealed E1-E9 proof evidence URI 也不能指向 `local:`、`fixture:`、`scripts/**` 或 smoke 机制证据；snapshot
evidence URI 和 sealed proof URI path 还必须带 `release-candidate` / `release_candidate` 语义，不能把
`completion-audit-evidence.md`、`operations-readiness-evidence.md` 这类机制证据文档包装成 RC closure。该 URI
allowlist 本身只是 path shape gate：只检查路径是否包含 `release-candidate` 或 `release_candidate`。因此
`release_candidate` record 还必须带强审核 metadata：`review_decision=approved`、非空 `reviewed_by` 和有效
RFC3339 `reviewed_at`；record 会封存这些字段，readiness 会对既有 snapshot 重新检查，缺失或非 approved
时返回 `completion_audit_snapshot_review_metadata_missing`。除此之外，`release_candidate` record 还必须通过只读本地
evidence file audit：snapshot evidence URI 和 sealed proof URI
必须解析到 AreaFlow checkout 内的 `docs/**/*.md`，proof URI fragment 必须匹配 Markdown heading anchor，并封存每个
evidence 文件的路径、anchor、sha256 和 size。file audit 证明本地文件绑定未丢失或漂移，但仍不证明文件内容经过真实审核。
同时必须封存
完整 E1-E9 proof evidence URI map、proof event ID map 和 required proof provenance map；每个 required proof key
必须有独立的 reviewed URI binding 和独立 `> 0` proof event ID binding，E4 的 Archive、Shim Retirement 和
Execution Cutover proof event ID 是三个互相独立的门禁。缺少 required proof URI key、复用同一个 URI、
缺少 proof event ID、复用同一个 proof event ID、缺少 required provenance，或 E7 operations proof key 带 fixture
标记时都不能记录；`manual_ops_smoke_review` 是 reviewed ops proof key 示例，`v1_stable_fixture_smoke` 只用于
机制 / full-proof smoke，不能封存 release-candidate。除此之外，
`release_candidate` snapshot 必须绑定当前 ready 且 target-scoped 的 `ReleaseEvidenceBundle.BundleHash`：
record command 会读取 AreaMatrix `scope=project` / `project_key=areamatrix` bundle。Platform bundle 只用于
全局诊断，不能关闭 E5 或 release-candidate snapshot。record command 会读取当前
release evidence bundle，要求 `status=ready`、`mode=read_only_release_evidence_bundle`、必需 evidence items 全部
ready，并把 `release_evidence_bundle_hash`、status、mode、item count、ready flag、`proof_evidence_uri_map`、
`proof_evidence_uri_count`、`required_proof_evidence_uri_keys`、`proof_event_ids`、
`proof_event_id_count`、`required_proof_event_id_keys`、`proof_provenance_map`、
`required_proof_provenance_keys`、`review_decision`、`reviewed_by`、`reviewed_at`、
`review_metadata_status`、`evidence_uri_file_audit`、`evidence_uri_file_audit_count` 和
`evidence_uri_file_audit_status=pass` 写入 command response、event 和 audit event metadata；
bundle hash 必须绑定项目 inventory 的 root/kind/adapter/profile/branch identity 字段，但不得把 snapshot
record 自身会改变的 DB row counts 当作稳定绑定输入；
最终仍必须由外部 evidence URI 和后续审计证明。该命令不运行 completion
audit 之外的检查、不运行 smoke、不执行
`git status`、不写 AreaMatrix、不创建 release package、不 publish、不 restore、不解析 secret、不启动
worker。Fixture snapshot 只能证明封存链路可用，不能代替真实 AreaMatrix release candidate closure evidence。
`areaflow completion audit-snapshot readiness <project> --json` 是只读门禁查询；非目标 project 必须返回
`completion_audit_snapshot_project_mismatch`；目标 key 但 root/adapter/profile/kind 不是真实 AreaMatrix 身份时必须返回
`completion_audit_snapshot_real_project_identity_missing`。只有真实项目身份通过后，最新 snapshot 仍为
`evidence_class=fixture` 时才返回 blocked 并给出 `completion_audit_snapshot_fixture_only`，防止 fixture complete
被误判为真实 release candidate closure。当真实项目身份通过但尚未记录 snapshot 时，readiness 必须在
`completion_audit_snapshot_missing` metadata 中暴露 `required_proof_evidence_uri_keys`、当前 completion audit
status/scope/hash、`required_proof_event_id_keys`、`required_proof_provenance_keys`，以及当前 release evidence
bundle hash/status/mode/item count，供真实 closure 记录前逐项核对。
Readiness JSON 还必须提供机器可消费的 `gaps[]` 和 `closure` 摘要。`gaps[]` 归一化暴露缺失 proof URI key、
proof event ID key、proof provenance key、机制 evidence URI、bundle/file/current-binding blockers 和 unsafe facts。
`closure` 必须至少暴露 `ready_for_release_candidate_closure=false|true`、`required_evidence_class=release_candidate`、
`project_identity`、`snapshot`、`audit_binding`、`snapshot_evidence`、`proof_evidence_uris`、`proof_event_ids`、
`proof_provenance`、`current_proof_binding`、`release_evidence_bundle`、`evidence_file_audit`、`safety`、`gap_keys`
和汇总 blockers，供 UI/automation 判断是哪一道 RC closure gate 未过。`closure.ready_for_release_candidate_closure=true`
只表示 snapshot guard 对证据身份、哈希和漂移检查通过；它仍不能替代外部对 release-candidate evidence 内容的审计。
最新 release-candidate snapshot 的
`metadata.proof_evidence_uri_map` 必须覆盖 required proof URI keys；缺少 key 时，readiness 必须返回
`completion_audit_snapshot_proof_evidence_uri_missing`；如果多个 required proof key 复用同一个 URI binding，或任一
required proof URI 不指向 release-candidate 证据路径，readiness 也必须保持 blocked。最新 release-candidate snapshot 的
`proof_event_ids`
也必须覆盖 required proof event ID keys；缺少 key、event ID 非正数或多个 required proof key 复用同一个 event ID 时，
readiness 必须返回 `completion_audit_snapshot_proof_event_id_missing`。最新 release-candidate snapshot 的 `proof_provenance_map` 必须覆盖
`E7_operations_readiness.latest_operations_smoke_proof_key`；`manual_ops_smoke_review` 可作为 reviewed ops proof key
示例，fixture proof key 如 `v1_stable_fixture_smoke` 必须返回
`completion_audit_snapshot_proof_provenance_missing` 并带 `snapshot_operations_proof_key_fixture` blocker。
Readiness 还必须把 sealed `proof_evidence_uri_map`、`proof_evidence_uris`、`proof_event_ids` 和
`proof_provenance_map` 与当前 completion audit 重新计算出的 proof URI map、URI set、event ID map 和 provenance map
逐项比对；结构完整但不再匹配当前 proof records 时，必须返回
`completion_audit_snapshot_current_proof_binding_mismatch`。Readiness 会重新计算当前 completion audit hash；latest
snapshot 的 audit hash 与当前 hash 不一致时，必须返回 `completion_audit_snapshot_audit_hash_mismatch`，并在
metadata 中暴露 `snapshot_audit_hash`、`current_audit_status`、`current_audit_scope`、`current_audit_hash` 和
`audit_hash_match`。最新 release-candidate snapshot 的 `metadata.release_evidence_bundle_hash` 也必须等于 readiness
查询时的当前 `bundle_hash`；hash 缺失、hash 不一致、bundle 已不 ready、status/mode/ready flag/item count metadata
不一致时，readiness 必须返回
`completion_audit_snapshot_release_evidence_bundle_mismatch`，并在 metadata 中暴露 `latest_bundle_hash`、
`current_bundle_hash` 和 `bundle_blockers`。
最新 release-candidate snapshot 还必须携带通过的 `evidence_uri_file_audit_status`，且 sealed file audit 必须与当前
本地 evidence 文件的 path、anchor、sha256 和 size 一致；metadata 缺失、status 非 pass、文件/anchor 缺失或内容漂移时，
readiness 必须返回 `completion_audit_snapshot_evidence_uri_file_audit_mismatch`，并在 metadata 中暴露
`evidence_uri_file_audit_blockers`。

## Anti-patterns

以下说法必须被 completion audit 拒绝：

- “测试都过了，所以 100%。”
- “release final gate pass，所以 100%。”
- “Web/Desktop 能展示，所以多端完成。”
- “readiness ready，所以 apply 已经打开。”
- “AreaFlow self dogfood 通过，所以 AreaMatrix dogfood 完成。”
- “object metadata 存在，所以 artifact 可完整恢复。”
- “support bundle preview 存在，所以 support export 已完成。”
- “team role 存在，所以远程控制台可以执行命令。”
- “budget estimate 存在，所以 quota enforcement 已打开。”
- “shim installed，所以 execution cutover 完成。”

## Closing Conditions

AreaFlow 才能标记 100%，当且仅当：

```text
all E1-E9 required items complete
release final gate pass
completion audit status complete
no v0-v1.0 task is planned / missing_evidence / blocked
all preview_only and implemented_scoped states are either closed by evidence or explicitly v1.x deferred
AreaMatrix dogfood execution cutover and shim retirement are proven
AreaMatrix protected path proof is clean or explicitly authorized in a completed cross-repo task
release package / publish / restore / secret / remote worker / plugin / managed ops remain closed unless v1.x apply packet exists
```

在这些条件满足前，AreaFlow 可以说“某阶段已实现”“某 gate 已 ready”或“某 preview 可用”，但不能说
“0-100% 已完成”。
