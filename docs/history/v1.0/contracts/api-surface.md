# API Surface

## 定位

AreaFlow 的 API Service 是唯一业务边界。CLI、Web、Desktop 和 worker 都应复用同一套 API
和 service layer，不能各自维护状态或绕过 gate。

稳定 API 契约使用 `/api/v1`。当前实现保留 `/api` 作为兼容别名；新 Web、Desktop、worker 和文档
应优先使用 `/api/v1`。

v0.6 Worker Beta 的 worker lifecycle、lease、dry-run run-once、scoped execution evidence 和 no-cutover
边界见 [`v0.6-worker-beta-contract.md`](./v0.6-worker-beta-contract.md)。API 返回的 `pass`、`ready`、
`verified`、`artifact_written` 或 `rollback_verified` 必须结合 v0.6 scope label / safety facts 解读，
不能被 UI、CLI、Desktop、worker 或 compatibility shim 升级成真实 AreaMatrix execution cutover。

## 总结构

```text
CLI / Web / Desktop / Worker
  -> REST API + SSE
    -> Service Layer
      -> PostgreSQL
      -> Artifact Store
      -> Workflow Stage Engine
      -> Project Adapters
      -> Worker Pool
      -> Engine Adapters
```

API 按能力分四类：

```text
Query API:
  只读状态、metadata、gate、artifact metadata、audit 和 readiness。

Command API:
  所有业务写入、approval、projection apply、cutover、run control、worker lease 和受限 execution。

Event API:
  SSE / watch，只观察事件，不承载写动作，不作为恢复主状态。

Admin API:
  migrate、service、doctor、import/export 等受限运维入口；不得绕过 Command API、permission、
  approval 或 audit 改变 workflow 业务状态。
```

## Query API

Query API 只读当前状态、事件、artifact metadata 和审计记录。

```text
GET /api/v1/health
GET /api/v1/service/status
GET /api/v1/desktop/service-control-gate
GET /api/v1/desktop/notification-gate
GET /api/v1/desktop/tray-menu-gate
GET /api/v1/security/boundary-readiness
GET /api/v1/backup/manifest
GET /api/v1/backup/restore-plan
GET /api/v1/release/readiness
GET /api/v1/release/remediation-plan
GET /api/v1/release/acceptance-preview
GET /api/v1/release/acceptance-gate
GET /api/v1/release/exception-doctor
GET /api/v1/release/exception-record-preview
GET /api/v1/release/exception-schema-preview
GET /api/v1/release/exception-migration-approval-gate
GET /api/v1/release/exception-apply-preview
GET /api/v1/release/final-gate
GET /api/v1/release/evidence-bundle
GET /api/v1/release/package-preview
GET /api/v1/release/distribution-preview
GET /api/v1/release/publish-gate
GET /api/v1/release/publish-approval-preview
GET /api/v1/release/rollout-plan-preview
GET /api/v1/audit/coverage
GET /api/v1/audit/coverage?project_key={project_key}
GET /api/v1/permissions/doctor?project_key={project_key}
GET /api/v1/conformance?project_key={project_key}
GET /api/v1/artifacts/integrity?project_key={project_key}
GET /api/v1/projects
GET /api/v1/projects/{project_key}
GET /api/v1/projects/{project_key}/summary
GET /api/v1/projects/{project_key}/readiness
GET /api/v1/projects/{project_key}/generated-write-readiness
GET /api/v1/projects/{project_key}/generated-write-apply-beta-gate
GET /api/v1/projects/{project_key}/import-diff
GET /api/v1/projects/{project_key}/verification-bundle
GET /api/v1/projects/{project_key}/compatibility
GET /api/v1/projects/{project_key}/shim-preview
GET /api/v1/projects/{project_key}/shim-readiness
GET /api/v1/projects/{project_key}/shim-authorization
GET /api/v1/projects/{project_key}/shim-apply-packet
GET /api/v1/projects/{project_key}/shim-apply-gate
POST /api/v1/projects/{project_key}/shim-apply
GET /api/v1/projects/{project_key}/cutover-readiness?version={version}
GET /api/v1/projects/{project_key}/execution-cutover-readiness
GET /api/v1/projects/{project_key}/execution-forwarding-v1-readiness
GET /api/v1/projects/{project_key}/execution-forwarding-v1-apply-preview
GET /api/v1/projects/{project_key}/execution-forwarding-v1-apply-packet
GET /api/v1/projects/{project_key}/execution-forwarding-v1-apply-gate
GET /api/v1/projects/{project_key}/execution-forwarding-v1-command-preview?task_type={task_type}
GET /api/v1/projects/{project_key}/execution-forwarding-v1-rollback-preview
GET /api/v1/projects/{project_key}/status-projections?limit=20
GET /api/v1/projects/{project_key}/status-projections/authorization?target_uri=.areaflow/status.json
GET /api/v1/projects/{project_key}/status-projections/apply-packet?target_uri=.areaflow/status.json
GET /api/v1/projects/{project_key}/status-projections/apply-gate?target_uri=.areaflow/status.json
GET /api/v1/projects/{project_key}/events?limit=20
GET /api/v1/projects/{project_key}/events/stream?after_id={event_id}
GET /api/v1/projects/{project_key}/artifacts?limit=20
GET /api/v1/projects/{project_key}/residuals?limit=20
GET /api/v1/projects/{project_key}/workflow-versions
GET /api/v1/projects/{project_key}/workflow-versions/{version}
GET /api/v1/projects/{project_key}/workflow-versions/{version}/stages
GET /api/v1/projects/{project_key}/workflow-versions/{version}/artifacts?limit=20
GET /api/v1/projects/{project_key}/workflow-versions/{version}/residuals?limit=20
GET /api/v1/projects/{project_key}/workflow-versions/{version}/gates
GET /api/v1/projects/{project_key}/workflow-versions/{version}/transition-previews
GET /api/v1/projects/{project_key}/workflow-versions/{version}/approvals
GET /api/v1/projects/{project_key}/workflow-versions/{version}/runs?limit=20
GET /api/v1/projects/{project_key}/engines/codex-cli/preview
GET /api/v1/projects/{project_key}/workers
GET /api/v1/runs/{run_id}?project_key={project_key}
GET /api/v1/runs/{run_id}/execution-approval-gate?project_key={project_key}
GET /api/v1/runs/{run_id}/execution-plan?project_key={project_key}
GET /api/v1/runs/{run_id}/project-write-design-gate?project_key={project_key}
GET /api/v1/runs/{run_id}/managed-generated-write-gate?project_key={project_key}
GET /api/v1/runs/{run_id}/events?project_key={project_key}
GET /api/v1/runs/{run_id}/events/stream?project_key={project_key}&after_id={event_id}
GET /api/v1/artifacts/{artifact_id}?project_key={project_key}
GET /api/v1/artifacts/{artifact_id}/content?project_key={project_key}
GET /api/v1/audit-events?limit=20
GET /api/v1/audit-events?project_key={project_key}&limit=20
GET /api/v1/worker-pool/summary
GET /api/v1/worker-pool/schedule-preview
GET /api/v1/web/write-action-gate
GET /api/v1/completion-audit
GET /api/v1/completion-audit/snapshot-readiness?project_key={project_key}
GET /api/v1/ops/readiness
GET /api/v1/ops/support-bundle-preview
GET /api/v1/ops/migration-ledger-readiness
```

Query API 不应触发文件写入、命令执行、secret 读取或 worker 调度。

Release readiness、exception、final gate、evidence、package/distribution/publish/rollout preview 的统一
语义见 [`release-final-gate-contract.md`](../../../architecture/release-final-gate-contract.md)。Release Query API 只回答
go/no-go 和预览，不创建 package、不写 exception record、不运行 migration、不 tag/push/sign/upload/publish。
0-100% completion audit 的整体只读聚合语义见
[`completion-audit-contract.md`](../../../architecture/completion-audit-contract.md)；release final gate `pass` 仍不能单独证明
AreaFlow 已达 100%。

全局 ID route 的 `project_key` 是兼容型 visibility guard。当前本机 single-user 模式仍允许不传
`project_key`；多项目、Web、Desktop、worker 和未来团队调用应传入 `project_key`。当传入的
`project_key` 与 run/artifact metadata 的 `project_id` 不匹配时，API 返回 `404`，避免通过全局 ID
探测其他 project 的记录是否存在。这个 guard 不等同于完整 API token、team 或 user 权限模型。
完整 auth、team、API token、secret resolve 和 remote worker credential 边界见
[`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md)，v1.0 前不启用真实 enforcement。
Team Console 和远程控制台的控制面分层见
[`team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md)；Web、Desktop 或远程 UI 的 presence
不能被解释为 command apply、membership write、token issuance、secret resolve 或 remote worker credential
已打开。
Budget / quota / usage metering 的边界见
[`budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md)；v1.0 Query API 只能展示 estimate / readiness /
blocked reason，不执行 quota decrement、reservation、charge 或 provider billing sync。
External integration / webhook 的边界见
[`integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)；v1.0 Query API 只能展示 integration
readiness / delivery plan preview / blocked reason，不投递 webhook、不接受 callback 作为业务事实、不调用
外部 API。
`GET /api/v1/security/boundary-readiness` 返回 v1.0 security boundary 的只读 readiness report，用于
证明 auth、team、API token、secret、remote worker credential、budget/quota、integration/webhook 和
managed ops 的高风险开口仍关闭。该接口不写数据库、不创建 audit event、不登录用户、不发放 token、
不轮换 token、不撤销真实 token、不改变 membership authorization、不解析 secret、不发放 remote worker
credential、不投递 webhook、不调用外部 API、不扣减 quota、不写 usage charge、不打开 remote ops。
响应包含：

```text
status
mode = read_only_security_boundary_readiness
items[]
items[].key
items[].category
items[].status
items[].message
items[].required_evidence[]
items[].blocked_by[]
items[].metadata
capabilities[]
forbidden_actions[]
auth_enforcement_open=false
team_permission_enforcement_open=false
api_token_issuance_open=false
api_token_enforcement_open=false
secret_resolve_open=false
remote_worker_credentials_open=false
budget_enforcement_open=false
quota_decrement_open=false
usage_charge_written=false
webhook_delivery_open=false
inbound_callback_open=false
external_api_call_open=false
authorization_changed=false
secret_plaintext_read=false
remote_worker_direct_pg_allowed=false
team_console_command_open=false
remote_ops_control_open=false
managed_upgrade_open=false
support_bundle_export_open=false
default_remote_telemetry_open=false
generated_at
```

后续即使新增 security boundary doctor、token fixture、secret readiness 或 remote worker credential
preview endpoint，也必须默认返回 readiness / blocked reason；除非对应 R4 rung 已获 explicit approval，
这些 endpoint 仍不能登录用户、发放 token、轮换 token、撤销真实 token、改变 membership authorization、
解析 secret、发放 remote worker credential 或改变 API 可见性。

`GET /api/v1/completion-audit` 返回 v1.0 0-100% 的只读 completion audit report。它聚合 phase/task、
AreaMatrix dogfood、release final gate、release packaging preview、backup/restore、operations readiness、
security/isolation 和 protected path proof 的当前状态，但不运行 smoke、不执行 `git status`、不写数据库、
不写项目文件、不创建 audit event、不创建 release package、不 publish、不 restore、不解析 secret、不启动
worker。响应包含：

```text
status = complete | incomplete | blocked
mode = read_only_completion_audit
scope = v1.0
items[]
items[].key
items[].category
items[].status
items[].message
items[].evidence_refs[]
items[].required_evidence[]
items[].blocked_by[]
items[].next_command
items[].metadata
deferred_v1x[]
capabilities[]
forbidden_actions[]
safety_facts.read_only=true
safety_facts.release_package_created=false
safety_facts.publish_attempted=false
safety_facts.restore_apply_attempted=false
safety_facts.secret_resolved=false
safety_facts.remote_worker_credentials_issued=false
safety_facts.area_matrix_protected_paths_touched=false
release_final_gate_status
area_matrix_dogfood_status
task_matrix_status
implementation_gap_status
protected_path_proof_status
generated_at
```

第一版不会替用户执行 AreaMatrix protected path proof 命令；缺少 proof record 时必须返回 blocked。
受控记录入口是：

```text
areaflow completion source-alignment-proof record <project> --status complete|incomplete|blocked --fact <key> --json
areaflow completion task-matrix-proof record <project> --status complete|incomplete|blocked --fact <key> --source-set-hash <sha256> --backlog-hash <sha256> --task-status-audit-hash <sha256> --planned-v1-required-task-count 0 --missing-evidence-v1-required-task-count 0 --blocked-v1-required-task-count 0 --json
areaflow completion security-closure-proof record <project> --status complete|incomplete|blocked --fact <key> --json
areaflow completion backup-restore-proof record <project> --status complete|incomplete|blocked --fact <key> --json
areaflow completion release-packaging-proof record <project> --status complete|incomplete|blocked --fact <key> --json
areaflow completion validation-proof record <project> --status complete|incomplete|blocked --fact <key> --json
areaflow completion archive-proof record <project> --status complete|incomplete|blocked --fact <key> --summary <text> --evidence-uri <release-candidate-uri> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --archive-scope <scope> --archive-reference-mode <mode> --archive-source-path <path> --archive-forbidden-action <action> --archive-rollback-target <target> --archive-fail-closed --json
areaflow completion shim-retirement-proof record <project> --status complete|incomplete|blocked --fact <key> --summary <text> --evidence-uri <release-candidate-uri> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --shim-retirement-scope <scope> --shim-prerequisite <key> --shim-retired-surface <surface> --shim-rollback-target <target> --shim-fail-closed --shim-reopen-requires-approval --json
areaflow completion execution-cutover-proof record <project> --status complete|incomplete|blocked --fact <key> --summary <text> --evidence-uri <release-candidate-uri> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --execution-cutover-scope <scope> --allowed-task-type <type> --forbidden-action <action> --rollback-target <target> --rollback-mode <mode> --json
areaflow completion protected-path-proof record <project> --status clean|authorized|dirty|blocked --summary <text> --evidence-uri <uri> --json
areaflow completion audit-snapshot record <project> --release-candidate <label> --evidence-class fixture|release_candidate --evidence-uri <release-candidate-uri> --summary <text> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --json
areaflow completion audit-snapshot readiness <project> --json
```

Source Alignment proof 的 `complete` 记录会由 CLI 自动只读采集当前 AreaFlow E1 source binding：
source paths、per-file sha256、source-set hash、file count 和 missing/unreadable counts。调用者不手填
这些 hash；completion audit 会重算 current binding 并拒绝旧 facts-only proof 或 source drift。

E4 的 Archive / Shim Retirement / Execution Cutover `complete` proof 还必须使用 release-candidate 形状的
proof-level evidence URI，并带 `review_decision=approved`、`reviewed_by` 和 RFC3339 `reviewed_at`。`local:`、
fixture/script/smoke 机制证据、`scripts/**` 路径或 fixture/mock/demo/sample/synthetic/testdata/placeholder/dummy/example
marker 不能关闭 E4；即便 proof 形状通过，completion audit 仍要求目标 project 是真实 AreaMatrix identity：
root `/Users/as/Ai-Project/project/AreaMatrix`、kind `product-repo`、adapter/profile `areamatrix`、branch `main`。

这些命令只写 AreaFlow evidence / audit，不运行测试、构建、smoke、`git status` 或 shell，不复制
artifact bytes、不删除历史文件、不重写 `progress.json`、不启动旧 runner、不写旧 logs/checkpoints、
不写 AreaMatrix、不触碰 protected paths；`GET /api/v1/completion-audit` 只读消费目标 project-scoped latest
proof，其它 project 的较新 proof 或 evidence URI 不能 shadow `areamatrix` 的 E1-E9 证据项。
`completion audit-snapshot record` 还要求当前 completion audit 顶层 `status=complete`，只记录 audit hash、
scope、release candidate label、evidence class、evidence URI 和 proof event IDs；`fixture` 是默认机制证明，
`release_candidate` 必须提供非 fixture、非本地脚本 / smoke wrapper 的 evidence URI 和 summary；snapshot
evidence URI 和 sealed proof evidence URI path 必须带 `release-candidate` / `release_candidate` 语义，不能把
`completion-audit-evidence.md`、`operations-readiness-evidence.md` 等机制证据文档包装成 RC closure。该 URI
allowlist 本身只是 path shape gate：只检查路径是否包含 `release-candidate` 或 `release_candidate`。除此之外，
`release_candidate` record 还必须通过只读本地 evidence file audit：snapshot evidence URI 和 sealed proof URI
必须解析到 AreaFlow checkout 内的 `docs/**/*.md`，proof URI fragment 必须匹配 Markdown heading anchor，并封存每个
evidence 文件的路径、anchor、sha256 和 size。file audit 证明本地文件绑定未丢失或漂移，但仍不证明文件内容经过真实审核。
它必须封存
完整 E1-E9 proof evidence URI map、proof event ID map 和 required proof provenance map；缺少 required proof URI
key、复用同一个 proof URI、缺少 proof event ID、复用同一个 proof event ID、缺少 required provenance，或 E7
operations proof key 带 fixture 标记时不能记录。`manual_ops_smoke_review` 是 reviewed ops proof key 示例；
`v1_stable_fixture_smoke` 只用于机制 / full-proof smoke，不能封存 release-candidate。它还必须绑定当前 ready
`ReleaseEvidenceBundle.BundleHash`，
并把 `release_evidence_bundle_hash`、bundle status、mode、item count、ready flag、`proof_evidence_uri_map`、
`proof_evidence_uri_count`、`required_proof_evidence_uri_keys`、`proof_event_ids` 和
`required_proof_event_id_keys`、`proof_provenance_map`、`required_proof_provenance_keys`、
`review_decision`、`reviewed_by`、`reviewed_at`、`review_metadata_status`、`evidence_uri_file_audit`、
`evidence_uri_file_audit_count` 和 `evidence_uri_file_audit_status=pass` 写入 snapshot metadata；
最终仍需要外部证据继续证明。它不运行 smoke、不创建 release package、不 publish、不 restore、不解析 secret、
不启动 worker、不写被管理项目。
`completion audit-snapshot readiness` 是只读查询；当最新 snapshot 仍是 `evidence_class=fixture` 时必须返回
blocked，并报告 `completion_audit_snapshot_fixture_only`。当最新 release-candidate snapshot 的
`metadata.proof_evidence_uri_map` 缺少 required proof URI key 时，必须返回
`completion_audit_snapshot_proof_evidence_uri_missing`；复用 URI binding、generic mechanism proof URI、缺少/复用
proof event ID、缺少 E7 proof provenance 或 fixture operations proof key 也必须保持 blocked；当 latest snapshot 的
`audit_hash` 与重新计算的当前
completion audit hash 不一致时，必须返回 `completion_audit_snapshot_audit_hash_mismatch`；当 sealed
proof URI map / URI set / event ID map / provenance map 与当前 completion audit 重新计算出的 proof bindings 不一致时，必须返回
`completion_audit_snapshot_current_proof_binding_mismatch`；当
`metadata.release_evidence_bundle_hash` 和当前 readiness 顶层 `bundle_hash` 不一致，或 bundle metadata / current
bundle readiness 不一致时，必须返回 `completion_audit_snapshot_release_evidence_bundle_mismatch`。
当最新 release-candidate snapshot 缺少通过的 `evidence_uri_file_audit_status`，或 sealed evidence 文件 path、
anchor、sha256、size 与当前本地文件不一致时，必须返回
`completion_audit_snapshot_evidence_uri_file_audit_mismatch`。
`GET /api/v1/completion-audit/snapshot-readiness` 暴露同一只读判断，默认读取 `areamatrix`，也可通过
`project_key` 查询其他 project；响应包含当前 `bundle_hash`、latest snapshot、readiness items 和 safety facts，
readiness item metadata 暴露 `snapshot_audit_hash`、`current_audit_hash`、`audit_hash_match`、
`latest_bundle_hash` / `current_bundle_hash` 以及 proof / provenance / bundle blocker 细节。
即使 release final gate 未来变成 `pass`，
completion audit 仍必须逐项证明 E1-E9 后才能返回 `complete`。Archive proof 只能关闭 E4 中的
Archive gate 证据缺口；Shim Retirement proof 只能关闭 E4 中的 retirement 证据缺口；二者都不能代替
Execution Cutover approval / command / audit proof；Validation proof 只能关闭 E3 的 fresh validation
evidence 缺口；Source Alignment proof 只能在 current source binding 仍匹配时关闭 E1 的 design source alignment 缺口，不能代替
E2/E3/E4/E8/E9；Task Matrix proof 只能关闭 E2 的 phase/task matrix 缺口，不能代替
E1/E3/E4/E8/E9。

`GET /api/v1/ops/readiness` 返回 v1.0 operations readiness 的只读聚合结果。它聚合 local service status、
metadata-only support bundle preview、migration ledger readiness、local-only telemetry 和 managed ops
deferral。该接口不运行 smoke、不启动或停止 service、不应用 migration、不创建 migration ledger、不导出
support bundle、不上传 telemetry、不读 secret、不复制 project file、不写数据库、不写被管理项目，也不触碰
AreaMatrix protected paths。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_operations_readiness
items[]
items[].key
items[].category
items[].status
items[].message
items[].evidence_refs[]
items[].required_evidence[]
items[].blocked_by[]
items[].next_command
items[].metadata
service_status
support_bundle
migration_ledger
capabilities[]
forbidden_actions[]
safety_facts.read_only=true
safety_facts.support_bundle_exported=false
safety_facts.support_bundle_metadata_only=true
safety_facts.remote_telemetry_enabled=false
safety_facts.managed_upgrade_attempted=false
safety_facts.destructive_rollback_attempted=false
safety_facts.service_process_control_attempted=false
safety_facts.database_write_attempted=false
safety_facts.project_write_attempted=false
safety_facts.area_matrix_protected_paths_touched=false
telemetry_default=local_only
managed_ops_status=deferred_v1x
support_export_status=deferred_v1x
generated_at
```

`areaflow ops smoke-proof record <project> --key <proof-key>` 是该边界内的受控证据写入命令。它只记录
外部 smoke 已通过的证明到 AreaFlow `events`、`audit_events` 和 `command_requests`，供
`GET /api/v1/ops/readiness` 与 completion audit 读取；它不运行 smoke、不控制 service process、不应用
migration、不导出 support bundle、不上传 telemetry、不写被管理项目，也不触碰 AreaMatrix protected paths。
当前允许的 proof key 包括：

```text
local_ops_smoke
v1_stable_fixture_smoke
web_dashboard_ops_smoke
manual_ops_smoke_review
install_migrate_start_smoke
```

其中 `manual_ops_smoke_review` 是 release-candidate 人工复核后的 ops proof key 示例；
`v1_stable_fixture_smoke` 只允许用于机制 / full-proof smoke，不能作为 RC snapshot 的 E7 provenance 封存。

`GET /api/v1/ops/support-bundle-preview` 返回 support bundle 的 metadata-only preview。它只输出 project
reference、metadata list、excluded sensitive content、hash / path reference 和 redaction policy，不复制、
压缩、上传或导出原始内容。响应包含：

```text
status
mode = metadata_only_support_bundle_preview
bundle_id
scope
projects[]
included_metadata[]
excluded_sensitive_content[]
path_references[]
hashes[]
capabilities[]
forbidden_actions[]
safety_facts.read_only=true
safety_facts.metadata_only=true
safety_facts.export_open=false
safety_facts.secret_values_included=false
safety_facts.prompt_text_included=false
safety_facts.user_file_contents_included=false
safety_facts.raw_artifact_contents_included=false
safety_facts.area_matrix_protected_paths_touched=false
generated_at
```

`GET /api/v1/ops/migration-ledger-readiness` 返回 migration ledger 的只读 readiness。它读取 embedded
migrations、`schema_migrations` 和 full ledger table 是否存在，但不应用 migration、不创建 table、不写
ledger、不执行 rollback。响应包含：

```text
status
mode = read_only_migration_ledger_readiness
entries[]
entries[].name
entries[].applied
entries[].status
entries[].required_evidence[]
applied_count
pending_count
schema_migrations_table_present
full_ledger_table_present
preflight_apply_verify_remediation_ready
capabilities[]
forbidden_actions[]
safety_facts.read_only=true
safety_facts.migration_apply_attempted=false
safety_facts.database_write_attempted=false
safety_facts.destructive_rollback_attempted=false
safety_facts.project_write_attempted=false
safety_facts.area_matrix_protected_paths_touched=false
generated_at
```

`GET /api/v1/web/write-action-gate` 返回 Web 写操作开放前的只读门禁矩阵。它为 approval、drain、
cancel、archive preview、status projection 和 generated write beta 等未来 Web 写动作展示 Command API、
risk preview、permission preflight、approval、audit 和 evidence 要求。该接口不创建 `command_requests`、
不写数据库、不写 artifact store、不写被管理项目、不调度 worker、不调用 engine、不运行 shell、不解析
secret，也不访问网络。响应包含：

```text
status
mode = read_only_web_write_action_gate
actions[]
actions[].key
actions[].label
actions[].category
actions[].status
actions[].default_ui_state
actions[].command_api
actions[].risk_level
actions[].required_capabilities[]
actions[].required_previews[]
actions[].required_approvals[]
actions[].required_audit_events[]
actions[].required_evidence[]
actions[].blockers[]
actions[].forbidden_direct_actions[]
capabilities[]
forbidden_actions[]
db_write_attempted
project_write_attempted
artifact_write_attempted
execution_write_attempted
command_created
approval_created
audit_event_written
worker_scheduled
engine_call_attempted
commands_run
secrets_resolved
network_used
generated_at
```

只要 `status=blocked` 或某个 action 的 `default_ui_state=disabled`，Web 必须保持按钮禁用或只读展示。
该 gate 不能被解释为相应 Command API 已对 Web 打开。

`GET /api/v1/projects/{project_key}/generated-write-readiness` 返回真实 managed project
generated-only dogfood 写入前的项目级只读门禁。它不依赖某个 run，不创建 `command_requests`、
run、task、lease、attempt、artifact、event 或 audit event，只读取 PostgreSQL 中的 project config 和
permission rows。响应包含：

```text
status
mode = read_only_generated_write_readiness
ready_for_review
apply_open
real_areamatrix_write_opened
required_capabilities[]
allowed_generated_prefixes[]
required_write_paths[]
configured_write_paths[]
configured_forbidden_paths[]
blockers[]
review_blockers[]
forbidden_actions[]
project_config_read
project_read_attempted
project_write_attempted
execution_write_attempted
area_flow_artifact_written
area_flow_execution_state_written
engine_call_attempted
commands_run
secrets_resolved
network_used
task_claimed
worker_started
lease_created
attempt_created
artifact_created
generated_at
```

`ready_for_review=true` 只表示 project config / permission 前置条件已具备人工审查资格；只要
`apply_open=false`，顶层 `status` 仍必须是 `blocked`。当前真实 AreaMatrix generated-only apply
保持关闭，因此 `real_areamatrix_write_opened=false`。该接口不得被解释为 queue/apply/write 已打开。

`GET /api/v1/projects/{project_key}/generated-write-apply-beta-gate` 返回真实 AreaMatrix
generated-only apply beta 打开前的只读 approval gate。它嵌套 `generated-write-readiness`，并额外暴露
R3 approval、smoke、rollback 和 beta scope 证据要求。该接口不创建 `command_requests`、run、task、
lease、attempt、artifact、event 或 audit event，不读取或写入真实 AreaMatrix 文件。响应包含：

```text
status
mode = read_only_generated_write_apply_beta_gate
readiness
items[]
required_capabilities[]
allowed_generated_prefixes[]
required_evidence[]
forbidden_actions[]
approval_required
approval_status = needs_approval
apply_open = false
real_areamatrix_write_opened = false
generated_only = true
project_read_attempted
project_write_attempted
execution_write_attempted
area_flow_artifact_written
area_flow_execution_state_written
engine_call_attempted
commands_run
secrets_resolved
network_used
task_claimed
worker_started
lease_created
attempt_created
artifact_created
generated_at
```

该 gate 可以在 readiness 已满足时仍返回 `blocked`，因为真实 AreaMatrix generated-only apply beta
必须等待显式 R3 approval。`approval_status=needs_approval` 不能被 CLI、Web、Desktop 或 worker
解释为可执行状态。

`GET /api/v1/projects/{project_key}/shim-authorization` 返回 AreaMatrix compatibility shim 真实落地前的
只读授权包。它把 allowed files、forbidden paths/actions、required preflight、post-edit verification、
rollback scope 和 safety facts 变成机器可读对象，供 CLI、Web、Desktop 或人工审批界面展示。该接口不创建
`command_requests`、不写数据库业务状态、不写 AreaMatrix、不运行 shell、不转发 `./task-loop run`、
不调用 engine、不解析 secret、不访问网络。响应包含：

```text
project
status = blocked
mode = read_only_authorization_packet
intent
readiness_status
allowed_files[]
forbidden_paths[]
forbidden_actions[]
required_preflight[]
post_edit_verification[]
rollback_scope[]
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.task_loop_run_forwarded=false
safety_facts.engine_call_attempted=false
safety_facts.commands_run=false
safety_facts.secrets_resolved=false
safety_facts.network_used=false
next_required_approval
```

该 packet 不能被解释为 AreaMatrix 编辑授权已经获得；它只说明如果用户随后明确授权，允许触碰哪些 shim
文件以及必须如何验证和回滚。
其中 `.areaflow/status.json` 若出现在 `allowed_files`，只表示可以通过受保护
`project.status_projection.apply` Command API 执行 R1 projection write；它不得由 shim 脚本直接写入，
也不得保存完整 queue、run、approval、logs、checkpoint、secret 或 artifact 原文。

`GET /api/v1/projects/{project_key}/shim-apply-packet` 返回 AreaMatrix compatibility shim edit 的只读
apply packet preview。它从当前 `shim-authorization` 生成 allowed files、authorization snapshot hash、
expected authorization mode、approval scope、project-scoped proof reference 字段、idempotency key、audit correlation id、
CLI gate command 和未来 apply command 草稿，并嵌套 `shim-apply-gate` 结果。该接口只生成 packet；
不创建 command request、不写 `.areaflow/status.json`、不写 `workflow/README.md`、不写 shim Python 文件、
不改 AreaMatrix、不转发 `./task-loop run`、不运行 shell、不调用 engine。

```text
project
status = needs_approval | ready | blocked
mode = shim_apply_packet_preview_v1
decision = needs_explicit_approval | ready_for_future_apply_command | readiness_blocked
authorization
gate
packet.command_type = project.shim.apply
packet.allowed_files[]
packet.authorization_snapshot_hash
packet.expected_authorization_mode = read_only_authorization_packet
packet.status_projection_packet_id
packet.status_projection_gate_id
packet.read_only_smoke_evidence_id
packet.dirty_worktree_review_id
packet.protected_path_fingerprint_id
packet.rollback_plan_id
packet.failure_mode = fail_closed
apply_gate_command[]
future_apply_command[]
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.task_loop_run_forwarded=false
safety_facts.area_matrix_files_modified=false
```

`GET /api/v1/projects/{project_key}/shim-apply-gate` 消费上述 packet 字段并返回只读 go/no-go。它要求
allowed files 与 `shim-authorization` 完全一致，authorization snapshot hash 匹配，readiness blocker
仅剩 `explicit_edit_approval`，status projection apply packet/gate proof、真实 AreaMatrix read-only smoke、
dirty worktree review、protected path fingerprint、rollback plan 和显式 approval 字段齐全。proof reference 必须使用
`<project_key>:<evidence_kind>:<id>` 格式，防止把其它项目或其它证据类型的裸 ID 当成 shim apply proof。
`apply_command_eligible=true` 只表示 packet 可以提交给未来受保护 shim apply command；该 gate 自身不创建
command、不写 AreaMatrix、不写 status projection row、不执行 task-loop、不调用 engine。

`POST /api/v1/projects/{project_key}/shim-apply` 是受保护 shim apply 命令入口。它复用
`shim-apply-gate` 的 packet 字段；gate 通过时返回 HTTP 201/200，并只记录 AreaFlow
`command_requests`、`events` 和 `audit_events`。gate 未通过时返回 HTTP 409 和
`decision=denied`，同样只留下受保护 command/audit 证据。该接口不创建 AreaFlow
run/task/attempt、不写 `.areaflow/status.json`、不写 shim 文件、不改 AreaMatrix、不转发
`./task-loop run`、不运行 shell、不调用 engine。真实 AreaMatrix shim 文件落地仍需要单独授权和写入
payload / rollback 证据。

```text
status = recorded | blocked
mode = shim_apply_command_v1
decision = allowed | denied
gate
apply_open=true only when the gate passed
command_request_created=true
area_flow_command_created=true
project_write_attempted=false
execution_write_attempted=false
task_loop_run_forwarded=false
status_projection_written=false
area_matrix_files_modified=false
```

`GET /api/v1/projects/{project_key}/execution-cutover-readiness` 返回 AreaMatrix execution cutover 前的
只读 readiness bundle。它聚合 import/mirror/shadow、authoring cutover、compatibility shim、worker
lease/run control、fixture execution、read-only verify、approved artifact write、fixture project write、
managed generated write 以及显式 execution cutover approval 证据。该接口不创建 command request、不写
AreaMatrix、不写 `workflow/versions/**/execution/**`、不转发 `./task-loop run`、不调用 engine、不运行
shell、不解析 secret、不访问网络。响应包含：

```text
status = pass | blocked
mode = read_only_areamatrix_execution_cutover_readiness
project
migration_path[]
shim_lifecycle_state
execution_forwarding_ready
shim_retirement_ready
items[]
items[].key
items[].category
items[].status
items[].message
items[].required_evidence[]
items[].next_command
items[].metadata
command_evidence
capabilities[]
forbidden_actions[]
safety_facts.read_only=true
safety_facts.execution_cutover_apply_open=false
safety_facts.execution_forwarding_open=false
safety_facts.shim_retirement_open=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.task_loop_run_forwarded=false
safety_facts.legacy_runner_started=false
safety_facts.legacy_progress_written=false
safety_facts.legacy_logs_written=false
safety_facts.legacy_checkpoint_written=false
safety_facts.engine_call_attempted=false
safety_facts.commands_run=false
safety_facts.secrets_resolved=false
safety_facts.network_used=false
safety_facts.retained_generated_apply_open=false
safety_facts.source_write_open=false
safety_facts.checkpoint_apply_open=false
safety_facts.repair_apply_open=false
next_steps[]
generated_at
```

当前真实 AreaMatrix execution cutover 必须保持 blocked，至少受 `compatibility_shim`、
`real_areamatrix_generated_apply`、`copy_repair_checkpoint` 和 `explicit_execution_cutover_approval` 阻断。
该 readiness 只能说明下一步需要什么证据，不能被解释为 task-loop replacement 或 execution cutover apply
已经打开。
Shim lifecycle state 只能按 `not_installed -> read_only_shim -> execution_forwarding -> retired_thin_entry`
推进。`execution_forwarding_ready=true` 仍不代表 `./task-loop run` 已转发；`shim_retirement_ready=true` 也不
代表旧 runner 或历史 evidence 可以删除。

`GET /api/v1/projects/{project_key}/execution-forwarding-v1-readiness` 返回 Execution Forwarding v1 的
只读 readiness。它只说明第一版 `./task-loop run` forwarding 是否具备入口、scope、证据和 rollback
前置条件，不创建 forwarding command、不转发 `./task-loop run`、不启动 worker、不写 AreaMatrix、不写旧
progress/log/checkpoint。响应包含：

```text
status = pass | blocked
mode = read_only_execution_forwarding_v1_readiness
project
allowed_task_types[]
items[]
items[].key
items[].category
items[].status
items[].message
items[].required_evidence[]
items[].next_command
items[].metadata
command_evidence
capabilities[]
forbidden_actions[]
safety_facts.read_only=true
safety_facts.forwarding_v1_apply_open=false
safety_facts.task_loop_run_forwarded=false
safety_facts.legacy_task_loop_started=false
safety_facts.legacy_progress_written=false
safety_facts.legacy_logs_written=false
safety_facts.legacy_checkpoint_written=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.engine_call_attempted=false
safety_facts.secrets_resolved=false
safety_facts.network_used=false
safety_facts.source_write_open=false
safety_facts.generated_retained_write_open=false
safety_facts.repair_apply_open=false
safety_facts.checkpoint_apply_open=false
safety_facts.publish_apply_open=false
safety_facts.restore_apply_open=false
next_steps[]
generated_at
```

第一版 allowed task types 固定为 read-only verify、doctor/readiness、artifact evidence、
status/projection validation 和 release/readiness check。source write、generated retained write、repair、
checkpoint、engine execution、secret resolve、network/API integration、publish 和 restore 必须继续 fail closed。

`GET /api/v1/projects/{project_key}/execution-forwarding-v1-apply-preview` 返回 Execution Forwarding v1 的
只读 apply preview。它说明受保护 apply Command API 必须携带的 packet 字段、approval、proof facts、
rollback target 和 forbidden actions；它不创建 command、不转发 `./task-loop run`、不启动 worker、不写
AreaMatrix、不写旧 progress/log/checkpoint。响应包含：

```text
status = blocked
mode = read_only_execution_forwarding_v1_apply_preview
project
readiness
items[]
allowed_task_types[]
forwarding_targets[]
blocked_targets[]
required_capabilities[]
apply_packet_fields[]
fail_closed_fields[]
required_proof_facts[]
required_evidence[]
forbidden_actions[]
approval_required=true
approval_status=needs_approval
apply_open=false
rollback_target=read_only_shim
safety_facts.read_only_preview=true
safety_facts.forwarding_v1_apply_open=false
safety_facts.task_loop_run_forwarded=false
safety_facts.legacy_task_loop_started=false
safety_facts.legacy_progress_written=false
safety_facts.legacy_logs_written=false
safety_facts.legacy_checkpoint_written=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.area_flow_command_created=false
safety_facts.area_flow_run_created=false
safety_facts.worker_scheduled=false
safety_facts.engine_call_attempted=false
safety_facts.commands_run=false
safety_facts.secrets_resolved=false
safety_facts.network_used=false
safety_facts.source_write_open=false
safety_facts.generated_retained_write_open=false
safety_facts.repair_apply_open=false
safety_facts.checkpoint_apply_open=false
safety_facts.publish_apply_open=false
safety_facts.restore_apply_open=false
generated_at
```

该 preview 是 Execution Forwarding v1 apply 的设计门禁，不是 apply 本身；即使 readiness evidence 通过，
没有 read-only shim、explicit approval、legacy non-write proof、rollback proof 和 protected path proof，
`apply_open` 仍必须保持 `false`。

`GET /api/v1/projects/{project_key}/execution-forwarding-v1-apply-packet` 返回 Execution Forwarding v1 的
只读 apply packet preview。它从当前 apply preview 生成 `readiness_snapshot_hash`、allowed task types、
target command matrix、approval scope、idempotency key、audit correlation id 和 fail-closed mode，并可带入
`approval_id`、`legacy_non_write_proof_id`、`rollback_plan_id` 与 `protected_path_fingerprint_id`。它不创建
command、run、task、attempt、artifact 或 audit，也不转发 `./task-loop run`。响应包含：

```text
status = ready | blocked | needs_approval
mode = execution_forwarding_v1_apply_packet_preview_v1
decision = ready_for_future_apply_command | readiness_blocked | needs_explicit_approval
project
apply_preview
gate
packet
packet.command_type=project.execution_forwarding_v1.apply
packet.allowed_task_types[]
packet.target_command_types[]
packet.approval_scope=execution_forwarding_v1_read_only_evidence_only
packet.readiness_snapshot_hash
packet.expected_shim_lifecycle_state=read_only_shim
packet.failure_mode=fail_closed
apply_gate_command[]
future_apply_command[]
required_human_review[]
forbidden_actions[]
safety_facts.read_only_preview=true
safety_facts.command_request_created=false
safety_facts.area_flow_run_created=false
safety_facts.task_loop_run_forwarded=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.engine_call_attempted=false
generated_at
```

`future_apply_command` 是历史字段名；当前它指向已存在的受保护 apply command。即使 packet preview
返回 `ready_for_future_apply_command`，真实 command 仍会重新执行 gate，并在 read-only shim 或 proof
缺失时记录 blocked/denied，而不是转发 `./task-loop run`。

`GET /api/v1/projects/{project_key}/execution-forwarding-v1-apply-gate` 消费 apply packet 字段并返回只读
go/no-go。它校验 allowed task types、readiness snapshot hash、expected shim lifecycle、legacy non-write
proof id、rollback plan id、rollback-to-read-only-shim readiness、protected path fingerprint id、
approval id/scope/actor/reason、idempotency key、audit correlation id 和 `failure_mode=fail_closed`。它只表示
packet 是否具备进入受保护 apply command 的资格；`apply_command_eligible=true` 也不是 apply。响应包含：

```text
status = pass | blocked
mode = execution_forwarding_v1_apply_gate_v1
decision = go | no_go
project
items[]
required_packet_fields[]
required_capabilities[]
allowed_task_types[]
target_command_types[]
blocked_task_types[]
forbidden_actions[]
fail_closed_fields[]
required_proof_facts[]
approval_required=true
approval_status=approved | missing_or_incomplete
apply_command_eligible
apply_open=false
safety_facts.read_only_gate=true
safety_facts.apply_command_eligible_is_not_apply=true
safety_facts.command_request_created=false
safety_facts.area_flow_run_created=false
safety_facts.task_loop_run_forwarded=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.engine_call_attempted=false
generated_at
```

`GET /api/v1/projects/{project_key}/execution-forwarding-v1-command-preview?task_type={task_type}` 返回未来
`./task-loop run` 转发某个 task type 时的只读 response preview。它只分类 allowed / blocked / unknown
task type，说明 approval 之后会转向哪个受保护 Command API，或当前会怎样 fail closed；它不创建
command request、不创建 run/task/attempt/artifact/audit、不转发 `./task-loop run`、不启动 legacy runner、
不写 AreaMatrix、不写旧 progress/log/checkpoint。响应包含：

```text
status = blocked
mode = read_only_execution_forwarding_v1_command_preview
project
decision = would_forward_after_approval | blocked_task_type_fail_closed | unknown_task_type_fail_closed
message
task_type
target_command_type
target_status
failure_mode=fail_closed
allowed_task_type
blocked_task_type
apply_open=false
would_create_command_request_after_approval
would_create_run_after_approval
would_create_run_task_after_approval
would_create_run_attempt_after_approval
would_create_artifact_after_approval
would_create_audit_event_after_approval
project_write_allowed=false
execution_write_allowed=false
legacy_fallback_allowed=false
required_packet_fields[]
required_capabilities[]
fail_closed_fields[]
blocked_by[]
allowed_task_types[]
forbidden_actions[]
safety_facts.read_only_preview=true
safety_facts.command_preview=true
safety_facts.area_flow_command_created=false
safety_facts.area_flow_run_created=false
safety_facts.task_loop_run_forwarded=false
safety_facts.legacy_task_loop_started=false
safety_facts.legacy_progress_written=false
safety_facts.legacy_logs_written=false
safety_facts.legacy_checkpoint_written=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.engine_call_attempted=false
safety_facts.commands_run=false
safety_facts.secrets_resolved=false
safety_facts.network_used=false
generated_at
```

`GET /api/v1/projects/{project_key}/execution-forwarding-v1-rollback-preview` 返回 Execution Forwarding v1 的
只读 rollback preview。它说明未来 forwarding v1 如果失败或暂停时怎样 fail closed 回到
`read_only_shim`、需要哪些 proof facts、哪些 reopen conditions 必须重新满足，以及哪些动作继续禁止；
它不创建 rollback command、不转发 `./task-loop run`、不启动 legacy runner、不写 AreaMatrix、不删除
AreaFlow forwarding history、不写旧 progress/log/checkpoint。响应包含：

```text
status = blocked
mode = read_only_execution_forwarding_v1_rollback_preview
project
apply_preview
items[]
rollback_target=read_only_shim
fail_closed_steps[]
reopen_conditions[]
required_proof_facts[]
required_evidence[]
forbidden_actions[]
rollback_apply_open=false
safety_facts.read_only_preview=true
safety_facts.rollback_apply_open=false
safety_facts.apply_open=false
safety_facts.forwarding_v1_apply_open=false
safety_facts.task_loop_run_forwarded=false
safety_facts.legacy_task_loop_started=false
safety_facts.legacy_progress_written=false
safety_facts.legacy_logs_written=false
safety_facts.legacy_checkpoint_written=false
safety_facts.project_write_attempted=false
safety_facts.execution_write_attempted=false
safety_facts.area_flow_command_created=false
safety_facts.area_flow_run_created=false
safety_facts.worker_scheduled=false
safety_facts.engine_call_attempted=false
safety_facts.commands_run=false
safety_facts.secrets_resolved=false
safety_facts.network_used=false
safety_facts.source_write_open=false
safety_facts.generated_retained_write_open=false
safety_facts.repair_apply_open=false
safety_facts.checkpoint_apply_open=false
safety_facts.publish_apply_open=false
safety_facts.restore_apply_open=false
generated_at
```

该 preview 只能证明 rollback 方案字段完整、fail-closed 路径可审查；它不执行 rollback，也不能删除或改写
已有 AreaFlow command、run、task、attempt、artifact、event 或 audit history。重新打开 forwarding v1 必须重新
满足 approval、focused smoke、rollback proof 和 protected path proof。

`GET /api/v1/projects/{project_key}/status-projections` 返回只读 status projection 列表，供
AreaMatrix compatibility shim、Web、Desktop 和 CLI 查看 AreaFlow 生成给外部入口的粗略状态。
响应包含：

```text
project
projections[]
projections[].target_kind
projections[].target_uri
projections[].summary_state
projections[].write_state
projections[].source_event_id
projections[].source_hash
projections[].generated_at
projections[].written_at
projections[].payload
```

该接口只读取 PostgreSQL 中的 projection metadata 和 payload，不写 `.areaflow/status.json`、
不更新 `workflow/README.md`、不触发 projection 重新生成、不执行 cutover，也不把 projection payload
当作 workflow、run、approval 或 artifact 的主状态源。真实 projection write 必须走 Command API、
permission preflight、path allowlist、approval（如需要）和 audit。

`GET /api/v1/projects/{project_key}/status-projections/authorization` 返回 status projection apply 前的
只读授权包预览。它读取 project config、latest snapshot、target preimage 和 permission policy，返回 target、
schema URI、validator preflight、protected path check、write-set、required packet fields、rollback plan、
blocked_by 和 safety facts。该接口不创建 `command_requests`、不写 status projection row、不写项目文件、
不运行 shell、不调用 engine。

`GET /api/v1/projects/{project_key}/status-projections/apply-packet` 返回 status projection protected apply
command 前的只读 packet preview。它从 authorization/preimage 自动生成 expected-before、source hash、schema
URI、validator preflight、protected path check、rollback action、accepted preimage schema、CLI apply command
和 API request，并嵌套 apply gate 结果。未传 `explicit_approval=true` 时必须返回 `needs_approval`；即使传入
approval actor/reason，它也只生成 packet，不创建 command、不写 `.areaflow/status.json`、不写 status
projection row、不改 AreaMatrix。

`GET /api/v1/projects/{project_key}/status-projections/apply-gate` 返回 status projection protected apply
command 前的只读 go/no-go。它消费 query packet 字段，包括 expected-before preimage、source hash、schema
URI、validator preflight、protected path check、rollback action、accepted preimage schema 和 approval
actor/reason。`apply_command_eligible=true` 只表示 packet 可提交给受保护 command；该接口自身不创建
command、不写 `.areaflow/status.json`、不写 status projection row、不改 AreaMatrix。

`GET /api/v1/service/status` 返回本机 AreaFlow service 的只读状态，供 Desktop shell、
Web launcher 和 CLI status 使用。响应聚合 API、PostgreSQL、worker pool 和 dashboard 入口状态，
并显式返回 desktop 允许观察的能力与禁止动作：
v0.9 Desktop Shell 的阶段合同见
[`v0.9-desktop-shell-contract.md`](./v0.9-desktop-shell-contract.md)。

```text
status
mode = local_service
api
database
worker_pool
dashboard.url
dashboard.api_url
capabilities[]
forbidden_actions[]
generated_at
```

该接口不启动/停止服务、不读取 secret、不调度 worker、不写被管理项目，也不维护第二状态源。
`web_url` 可作为 query 参数传入，用于 Desktop shell 显示当前 dashboard 入口。

`GET /api/v1/desktop/service-control-gate` 返回 Desktop 未来服务控制能力的只读门禁矩阵。第一版仅允许
`open_dashboard` 作为 enabled link；`start_service`、`stop_service`、`restart_service`、
`enable_notifications` 和 `tray_menu` 必须保持 disabled / blocked，直到 process control、restart
recovery、notification permission、audit 和 rollback contract 都被证明。该接口不启动/停止服务、
不创建 command request、不写数据库、不写项目文件、不调度 worker、不解析 secret、不访问网络，也不维护
Desktop 第二状态源。响应至少包含：

```text
status
mode = read_only_desktop_service_control_gate
actions[]
actions[].key
actions[].label
actions[].category
actions[].status
actions[].default_ui_state
actions[].risk_level
actions[].required_capabilities[]
actions[].required_evidence[]
actions[].blockers[]
actions[].forbidden_direct_actions[]
db_write_attempted
project_write_attempted
process_control_attempted
command_created
approval_created
audit_event_written
worker_scheduled
workflow_execution_started
secrets_resolved
network_used
generated_at
```

`areaflow desktop service-control-gate --json` 是同一语义的 CLI 观察面；它读取 service layer 返回的 gate，
不创建 command、不控制进程、不写 audit、不调度 worker、不写项目。

`GET /api/v1/desktop/notification-gate` 返回 Desktop 未来系统通知和 SSE 订阅能力的只读门禁矩阵。第一版
只允许 `observe_event_stream` 作为 read-only available；`enable_system_notifications`、
`approval_needed_notifications`、`run_failure_notifications` 和 `worker_recovery_notifications`
必须保持 disabled / blocked，直到 OS notification permission、event filter、dedupe、redaction、
rate limit 和 audit contract 都被证明。该接口不打开 SSE 连接、不请求系统通知权限、不创建 command
request、不写数据库、不写项目文件、不调度 worker、不解析 secret、不访问网络，也不维护 Desktop 第二
notification state。响应至少包含：

```text
status
mode = read_only_desktop_notification_gate
actions[]
actions[].key
actions[].label
actions[].category
actions[].status
actions[].default_ui_state
actions[].risk_level
actions[].required_capabilities[]
actions[].required_previews[]
actions[].required_approvals[]
actions[].required_audit_events[]
actions[].required_evidence[]
actions[].blockers[]
actions[].forbidden_direct_actions[]
db_write_attempted
project_write_attempted
event_stream_opened
notification_requested
command_created
approval_created
audit_event_written
worker_scheduled
workflow_execution_started
secrets_resolved
network_used
generated_at
```

`areaflow desktop notification-gate --json` 是同一语义的 CLI 观察面；它只展示 notification gate，不打开
SSE 连接、不请求 OS notification permission、不发送通知、不写 notification state。

`GET /api/v1/desktop/tray-menu-gate` 返回 Desktop 未来菜单栏/托盘能力的只读门禁矩阵。第一版只允许
`open_dashboard`、`show_service_status` 和 `show_recent_events` 作为 launcher / read-only available；
`start_service`、`stop_service`、`enable_notifications` 和 `open_settings` 必须保持 disabled / blocked，
直到 service control gate、notification gate、settings/secret UI contract、OS integration contract 和
audit 证据都被证明。该接口不创建原生 tray menu、不请求 OS integration、不创建 command request、
不写数据库、不写项目文件、不调度 worker、不解析 secret、不访问网络，也不维护 Desktop 第二 tray state。
响应至少包含：

```text
status
mode = read_only_desktop_tray_menu_gate
actions[]
actions[].key
actions[].label
actions[].category
actions[].status
actions[].default_ui_state
actions[].risk_level
actions[].required_capabilities[]
actions[].required_previews[]
actions[].required_approvals[]
actions[].required_audit_events[]
actions[].required_evidence[]
actions[].blockers[]
actions[].forbidden_direct_actions[]
db_write_attempted
project_write_attempted
tray_menu_created
os_integration_requested
command_created
approval_created
audit_event_written
service_control_attempted
notification_requested
worker_scheduled
workflow_execution_started
secrets_resolved
network_used
generated_at
```

`areaflow desktop tray-menu-gate --json` 是同一语义的 CLI 观察面；它不创建 native tray/menu、不请求 OS
integration、不执行 service control、不请求 notification permission、不打开 secret settings。

`GET /api/v1/backup/manifest` 返回只读 backup manifest，供 CLI、Web/Desktop 管理面和 release
smoke 判断当前 PostgreSQL metadata 与 artifact metadata 是否可枚举。响应包含：

```text
status
mode = read_only_manifest
scope = platform | project
project_key
schema_version
manifest_hash
table_counts[]
projects[]
projects[].inventory
projects[].artifact_count
projects[].artifacts[]
capabilities[]
forbidden_actions[]
generated_at
```

该接口只读取 PostgreSQL metadata 和 artifact metadata，不读取 artifact 原文，不生成压缩包，不写项目文件，
不解析 secret，也不执行 restore。`manifest_hash` 只覆盖可恢复清单的稳定形状，不包含
`generated_at`。
Artifact / backup / restore 的完整合同见
[`artifact-backup-restore-contract.md`](../../../architecture/artifact-backup-restore-contract.md)；object backend、archive copy/upload、
GC/delete 的长期合同见
[`object-artifact-retention-contract.md`](../../../architecture/object-artifact-retention-contract.md)。Manifest 不是 restore package。

`GET /api/v1/backup/restore-plan` 返回只读 restore dry-run plan，用于把 backup manifest 与 artifact
integrity 串成恢复前检查链。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_restore_plan
scope = platform | project
project_key
schema_version
manifest_hash
projects[]
items[]
items[].key
items[].category
items[].status
items[].message
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

第一版基于当前 PostgreSQL metadata 生成恢复计划，不读取外部 backup 包，不写数据库，不写 artifact store，
不覆盖项目文件，不解析 secret，也不执行 restore apply。存在历史 `external_project` / `project_reference`
artifact 时应返回 `needs_attention`，因为这些原文仍留在被管理项目中，不能假装可完整恢复。
存在 `object` artifact 时，只有 object verifier 已证明 key/version、sha256、size、namespace、encryption、
access policy 和 retention policy 后，restore plan 才能把它计入完整可恢复内容；verifier skipped/failed
必须返回 `needs_attention` 或 `blocked`。

`GET /api/v1/release/readiness` 返回只读 release readiness report，用于把 backup manifest、restore
dry-run plan、audit coverage、permission policy doctor、artifact integrity 和 adapter/profile
conformance 聚合成一个发布前判断。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_release_readiness
scope = platform | project
project_key
backup
restore_plan
audit_coverage
projects[]
projects[].permission
projects[].artifact_integrity
projects[].conformance
items[]
items[].key
items[].category
items[].status
items[].message
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建发布包，不写数据库，不写项目文件，不写 artifact store，不执行 restore，不执行 cutover，
不解析 secret，也不启动 worker。当前 AreaMatrix baseline 因 restore dry-run 仍有历史 project reference
artifact、audit coverage 仍有未启用长期能力 gap、artifact integrity 仍有 skipped 引用，因此应返回
`needs_attention`，不能伪装成 `ready`。

默认不带 query 时，该接口保持 platform scope，用于全局诊断。传 `?project=areamatrix` 或
`?project_key=areamatrix` 时返回 target-scoped readiness，并把 audit coverage 按目标项目 ID 过滤。
同一 release chain 的只读端点均接受相同 target scope query；CLI release 子命令均接受
`--project areamatrix`，默认不传时保持 platform scope。

`GET /api/v1/release/remediation-plan` 返回只读 release remediation plan，用于把 release readiness 的
`needs_attention` / `blocked` 项转换成可关闭行动。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_release_remediation_plan
scope = platform | project
project_key
readiness
actions[]
actions[].key
actions[].category
actions[].status
actions[].source_item
actions[].recommended_action
actions[].rationale
actions[].owner
actions[].next_command
actions[].acceptance
actions[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不自动接受 gap，不复制 artifact 原文，不写数据库，不写项目文件，不写 artifact store，不解析
secret，不执行命令，也不启动 worker。当前 baseline 应产生 restore、audit 和 artifact 三类 remediation
action，用来分别处理历史 project reference artifact、future-only audit gaps 和 skipped artifact references。

`GET /api/v1/release/acceptance-preview` 返回只读 release acceptance preview，用于预演哪些 remediation
action 可以走显式 release exception，哪些必须先修复。响应包含：

```text
status = ready | needs_decision | not_acceptable
mode = read_only_release_acceptance_preview
scope = platform | project
project_key
remediation
decisions[]
decisions[].key
decisions[].source_action
decisions[].category
decisions[].status
decisions[].acceptance_type
decisions[].owner
decisions[].reason
decisions[].required_evidence
decisions[].next_command
decisions[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不写数据库，不写项目文件，不写 artifact store，不自动接受 gap，不创建 approval，不执行命令，
不启动 worker，也不 apply release。当前 baseline 中 restore 的 metadata-only history、audit 的
future-only gap、artifact 的 skipped project reference 可以预览为 `needs_decision`；permission、
conformance 和 backup blockers 必须保持 `not_acceptable`，不能通过 acceptance preview 静默放行。

`GET /api/v1/release/acceptance-gate` 返回只读 release acceptance gate，用于把 acceptance preview
转换成发布前 gate 判断。响应包含：

```text
status = pass | blocked
mode = read_only_release_acceptance_gate
scope = platform | project
project_key
preview
items[]
items[].key
items[].category
items[].status
items[].decision_status
items[].acceptance_type
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不写数据库，不写项目文件，不写 artifact store，不自动接受 gap，不创建 approval，不执行命令，
不启动 worker，也不 apply release。当前 baseline 应返回 `blocked`，因为 metadata-only history、
future-only gap 和 archive exception 都仍是 `needs_decision`。该 gate 只报告 release exception
缺失的证据，不替代后续真实 acceptance / approval 写入。

`GET /api/v1/release/exception-doctor` 返回只读 release exception doctor，用于在启用真实 exception
record 写入前检查记录字段、审计动作和写入 guardrail。响应包含：

```text
status = pass | warn | fail
mode = read_only_release_exception_doctor
scope = platform | project
project_key
gate
checks[]
checks[].key
checks[].category
checks[].status
checks[].message
checks[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

当前 doctor 只诊断，不创建 exception record，不写 audit event，不 mark accepted，不创建 approval，
不执行命令，不启动 worker，也不 apply release。当前 baseline 应返回 `warn`：exception record schema
和 audit contract 已定义为必需项，但真实写入仍关闭；如果 acceptance gate 出现 `not_acceptable` 项，
doctor 必须返回 `fail`。

`GET /api/v1/release/exception-record-preview` 返回只读 release exception record preview，用于预演
未来会写入的 exception record、audit plan 和 rollback plan。响应包含：

```text
status = ready | draft | blocked
mode = read_only_release_exception_record_preview
scope = platform | project
project_key
doctor
drafts[]
drafts[].key
drafts[].source_gate_item
drafts[].source_decision
drafts[].acceptance_type
drafts[].status
drafts[].owner
drafts[].reason
drafts[].required_evidence
drafts[].audit_actions
drafts[].rollback_plan
drafts[].review_required
drafts[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 migration，不写 exception record，不写 audit event，不 mark accepted，不创建 approval，
不执行命令，不启动 worker，也不 apply release。当前 baseline 应返回 `draft`，并为 restore、
audit、artifact 三类 `needs_decision` gate item 生成只读 record draft；`not_acceptable` 项必须返回
`blocked` draft。

`GET /api/v1/release/exception-schema-preview` 返回只读 release exception schema preview，用于预演未来
真实 migration 的表、字段、索引、外键、apply steps、rollback steps 和 audit actions。响应包含：

```text
status = needs_approval | blocked
mode = read_only_release_exception_schema_preview
scope = platform | project
project_key
record_preview
tables[]
tables[].columns[]
tables[].indexes[]
tables[].foreign_keys[]
apply_steps[]
rollback_steps[]
audit_actions[]
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 migration 文件，不运行 migration，不写数据库，不写 audit event，不创建 release exception，
不 mark accepted，也不 apply release。当前 baseline 应返回 `needs_approval`，因为 schema 可以预览，
但真实 migration 仍需要显式确认。

`GET /api/v1/release/exception-migration-approval-gate` 返回只读 release exception migration approval
gate，用于阻止真实 migration 在缺少显式 approval 时被创建或执行。响应包含：

```text
status = blocked | pass
mode = read_only_release_exception_migration_approval_gate
scope = platform | project
project_key
schema_preview
items[]
items[].key
items[].category
items[].status
items[].approval_status
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 migration 文件，不运行 migration，不写数据库，不写 audit event，不创建 approval，
不 approve migration，不 mark accepted，也不 apply release。当前 baseline 应返回 `blocked`，
因为 `release_exceptions` migration 属于 R4 migration/security 操作，仍缺显式 migration approval。

`GET /api/v1/release/exception-apply-preview` 返回只读 release exception apply preview，用于预演未来
写入 exception records 和重新运行 release acceptance gate 前的 apply plan。响应包含：

```text
status = ready | blocked
mode = read_only_release_exception_apply_preview
scope = platform | project
project_key
migration_gate
items[]
items[].key
items[].category
items[].status
items[].action
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
apply_steps[]
apply_steps[].blocked_by[]
rollback_steps[]
rollback_steps[].blocked_by[]
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 migration 文件，不运行 migration，不写数据库，不写 audit event，不写 exception record，
不创建 approval，不 mark accepted，不执行命令，不启动 worker，也不 apply release。当前 baseline
应返回 `blocked`，因为 release exception migration approval gate 仍为 `blocked`。

`GET /api/v1/release/final-gate` 返回只读 release final gate，用于把 release readiness、
release acceptance gate 和 release exception apply preview 聚合成最终 go/no-go。响应包含：

```text
status = pass | blocked
mode = read_only_release_final_gate
scope = platform | project
project_key
readiness
acceptance_gate
exception_apply
items[]
items[].key
items[].category
items[].status
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 release package，不写数据库，不写项目文件，不写 artifact store，不创建 approval，
不 mark accepted，不运行 migration，不写 exception record，不执行命令，不启动 worker，也不 apply release。
当前 baseline 应返回 `blocked`，因为 release readiness、acceptance gate 和 exception apply preview
仍未同时通过。
完整 final gate / exception 语义见
[`release-final-gate-contract.md`](../../../architecture/release-final-gate-contract.md)；即使该接口未来返回 `pass`，也只允许
进入 evidence bundle、package preview、distribution preview、publish gate 和 rollout plan preview，
不代表真实发布。

`GET /api/v1/release/evidence-bundle` 返回只读 release evidence bundle，用于把 final gate、
backup manifest、audit coverage 和项目 artifact metadata inventory 聚合成发布证据索引。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_release_evidence_bundle
scope = platform | project
project_key
bundle_hash
final_gate
backup
audit_coverage
items[]
items[].key
items[].category
items[].status
items[].source
items[].description
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 release package，不生成压缩包，不读取 artifact 原文，不写数据库，不写项目文件，
不写 artifact store，不创建 approval，不运行 migration，不写 exception record，不执行命令，
不启动 worker，也不 apply release。`bundle_hash` 绑定 release final gate、backup、audit coverage 和项目
artifact metadata inventory 的稳定证据索引，并包含 project inventory 的 root/kind/adapter/profile/branch
identity 字段；它不读取或复制 artifact bytes，也不把会被 snapshot record 自身改变的 DB row counts 当作
release-candidate snapshot 的稳定绑定输入。当前 baseline 应返回 `blocked`，因为 final gate 仍为 `blocked`。
Completion audit 和 release-candidate snapshot 只接受 `scope=project` 且 `project_key=areamatrix` 的
bundle 作为 E5 binding；platform bundle 只能作为全局诊断。

`GET /api/v1/release/package-preview` 返回只读 release package preview，用于预演未来 release package
manifest 和证据文件路径。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_release_package_preview
scope = platform | project
project_key
evidence_bundle
package_name
items[]
items[].key
items[].category
items[].status
items[].package_path
items[].source
items[].description
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 release package，不生成压缩包，不读取 artifact 原文，不写数据库，不写项目文件，
不写 artifact store，不创建 approval，不运行 migration，不写 exception record，不执行命令，
不启动 worker，也不 apply release。当前 baseline 应返回 `blocked`，因为 evidence bundle 仍为 `blocked`。

`GET /api/v1/release/distribution-preview` 返回只读 release distribution preview，用于预演未来分发渠道
和发布动作的门禁。响应包含：

```text
status = ready | needs_attention | blocked
mode = read_only_release_distribution_preview
package_preview
items[]
items[].key
items[].category
items[].status
items[].channel
items[].action
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 release package，不写 release manifest，不上传 artifact，不发布 release，不创建 git tag，
不签名，不 push git，不写数据库，不写项目文件，不写 artifact store，不读取 artifact 原文，不执行命令，
不启动 worker，也不 apply release。当前 baseline 应返回 `blocked`，因为 package preview 仍为 `blocked`。

`GET /api/v1/release/publish-gate` 返回只读 release publish gate，用于阻止未通过 distribution preview
的发布动作进入真实发布。响应包含：

```text
status = pass | blocked
mode = read_only_release_publish_gate
distribution_preview
items[]
items[].key
items[].category
items[].status
items[].channel
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 release package，不写 release manifest，不上传 artifact，不发布 release，不创建 git tag，
不签名，不 push git，不创建 approval，不写数据库，不写项目文件，不写 artifact store，不读取 artifact
原文，不执行命令，不启动 worker，也不 apply release。当前 baseline 应返回 `blocked`，因为 distribution
preview 仍为 `blocked`。

`GET /api/v1/release/publish-approval-preview` 返回只读 release publish approval preview，用于预演未来
发布审批需要的证据和范围。响应包含：

```text
status = needs_approval | blocked
mode = read_only_release_publish_approval_preview
publish_gate
items[]
items[].key
items[].category
items[].status
items[].approval_status
items[].channel
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 approval，不 approve release，不写数据库，不写项目文件，不写 artifact store，不创建
release package，不写 release manifest，不上传 artifact，不发布 release，不创建 git tag，不签名，不
push git，不写 audit event，不执行命令，不启动 worker，也不 apply release。当前 baseline 应返回
`blocked`，因为 publish gate 仍为 `blocked`。

`GET /api/v1/release/rollout-plan-preview` 返回只读 release rollout plan preview，用于预演未来
发布审批后 rollout 的阶段、验证点和回滚步骤。响应包含：

```text
status = ready | needs_approval | blocked
mode = read_only_release_rollout_plan_preview
publish_approval_preview
items[]
items[].key
items[].category
items[].status
items[].stage
items[].action
items[].message
items[].owner
items[].required_evidence
items[].next_command
items[].metadata
rollout_steps[]
verification_checkpoints[]
rollback_steps[]
capabilities[]
forbidden_actions[]
generated_at
```

该接口不创建 rollout，不写 release state，不写数据库，不写项目文件，不写 artifact store，不创建
release package，不写 release manifest，不上传 artifact，不发布 release，不创建 git tag，不签名，
不 push git，不写 audit event，不执行命令，不启动 worker，也不 apply release。当前 baseline 应返回
`blocked`，因为 publish approval preview 仍为 `blocked`。

`GET /api/v1/audit/coverage` 返回只读审计覆盖矩阵。未带 `project_key` 时检查平台范围，
带 `project_key` 时只检查该项目的 `audit_events`。响应包含：

```text
status
mode = read_only_audit_coverage
scope = platform | project
total_audit_events
covered_requirements
gap_requirements
requirements[]
requirements[].required_actions[]
requirements[].missing_actions[]
generated_at
```

该接口不创建新的 audit event。它用于证明当前能力是否留下审计证据，也用于诚实暴露 v1.0 长期能力缺口，
例如真实命令执行、secret 解析和 permission change 尚未启用时应显示为 `gap`。

`GET /api/v1/permissions/doctor?project_key={project_key}` 返回只读 permission policy doctor，用于检查
项目配置、capability、路径 allow/deny、命令 deny、secret/network/git 禁用状态、worker capability
声明和 permission audit readiness。响应包含：

```text
status
mode = read_only_permission_policy_doctor
project
checks[]
checks[].key
checks[].category
checks[].status
checks[].message
checks[].metadata
generated_at
```

该接口不写 `audit_events`，不修改 project config，不执行命令，不读取 secret，也不尝试写入被管理项目。
长期未启用或高风险能力必须返回 `warn` / `fail`，不能为了进入下一阶段伪装成 `pass`。

`GET /api/v1/artifacts/integrity?project_key={project_key}` 返回只读 artifact integrity report，用于检查
AreaFlow-owned local artifact 的文件存在性、sha256 和 size 是否与 PostgreSQL metadata 一致。响应包含：

```text
status
mode = read_only_artifact_integrity
project
checked_artifacts
passed_artifacts
warn_artifacts
failed_artifacts
skipped_artifacts
checks[]
checks[].artifact
checks[].status
checks[].message
checks[].metadata
generated_at
```

该接口不会修复 artifact，不删除文件，不写项目文件，不读取 secret，也不读取 `external_project` /
`project_reference` 历史 artifact 原文。AreaFlow-owned `local` artifact 会读取原文并校验 hash/size；
历史 project reference 只校验 metadata 形状并返回 `skipped`，顶层 status 应为 `warn`，避免把未校验原文
伪装成完整通过。
`project_reference` / `external_project` 只证明 metadata 可索引，不证明 AreaFlow 拥有可恢复原文。
`object` backend 在 object verifier 落地前必须返回 `skipped` / `needs_attention`，不能计入完整可恢复内容。

`GET /api/v1/artifacts/{artifact_id}?project_key={project_key}` 只返回 metadata、hash、URI、关联对象和
retention 信息；`GET /api/v1/artifacts/{artifact_id}/content?project_key={project_key}` 才读取
AreaFlow-owned artifact 原文。Web/Desktop 不应直接读取本地 artifact store 路径，也不应把本地路径作为
长期状态源。大日志或报告后续可通过 range、tail 或 download 语义扩展，但权限、redaction、audit 和
project scope 仍由 API 控制。`project_reference` 历史 artifact 默认不通过 content API 读取原文，除非
另有显式 archive/copy command。

`POST /api/v1/projects/{project_key}/artifacts/archive-preview` 预演 artifact archive / retention 决策，
并通过 `artifact.archive.preview` command request 记录幂等、event 和 audit。响应包含：

```text
project
status = ready | needs_attention
mode = metadata_only_archive_preview
summary
items[]
items[].retention_class
items[].archive_state
items[].action
items[].decision
project_write_attempted = false
storage_write_attempted = false
artifact_delete_attempted = false
```

该接口不复制、不移动、不删除 artifact 原文，不写被管理项目，不写 artifact store，只根据 PG metadata
和 retention 规则生成候选：`ephemeral` 是 future GC candidate，`external_ref` 保持 metadata-only 并要求
archive ownership decision，`run_evidence` / `audit` / `release` 默认 retained。
`legal_hold` 必须返回 blocked/retained；未知 retention class 必须返回 `needs_policy`。真实 archive
copy/upload、object storage upload、retention-aware GC、orphan cleanup 和 delete apply 都属于 v1.x
Command API 能力，不能由 archive-preview 隐式打开。

`GET /api/v1/conformance?project_key={project_key}` 返回只读 adapter/profile conformance report，用于证明
项目 adapter、workflow profile、AreaFlow core 的边界仍然可验证。响应包含：

```text
status
mode = read_only_adapter_profile_conformance
project
profile_id
adapter
profile_hash
stage_count
gate_count
checks[]
checks[].key
checks[].category
checks[].status
checks[].message
checks[].metadata
generated_at
```

第一版针对 AreaMatrix dogfood baseline 检查：project `adapter` / `workflow_profile` 是否匹配内置
profile defaults；profile 是否可加载并拥有稳定 sha256；AreaMatrix item state 枚举、16 个 stage 和
17 个 gate 是否按固定顺序存在；AreaMatrix transition 链是否仍按 `intake -> ... -> closeout` 顺序绑定
required gate；防止 apply、execution、cutover 和 closeout 被误开的 hard rules 是否仍存在；artifact
metadata / content / owner policy 是否仍保持 PG metadata + artifact store content；profile cutover policy
是否仍保持 `v0_4_scope=authoring_source_of_truth_only` 且 execution cutover 不归入 v0.4；AreaMatrix
adapter 是否能只读加载 snapshot inventory；adapter/profile/core 边界是否保持 metadata-only、只读、不执行
命令、不解析 secret、不写数据库；active `areaflow.yaml` snapshot 是否满足当前安全基线，包括 protocol v1、
`import_mirror_shadow_cutover_archive` migration strategy、`read_project` / `write_status` enabled、
workflow/code/command/worker/git/network/secret/agent 高风险 capability disabled、`.areaflow/status.json`
write path、execution/DB/.areamatrix forbidden paths、dangerous command denylist、single-task scheduling、
disabled engine profiles、status export path 和 disabled workflow README human summary。

该接口只读项目 snapshot、workflow profile 和 active project config snapshot；不写项目文件，不写数据库，
不写 `audit_events`，不执行命令，不读取 secret，也不启动 worker。
Plugin / marketplace seed 边界见
[`plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)；该 conformance API 不安装、不启用、
不执行、也不远程拉取未知 plugin package。

`GET /api/v1/audit-events` 返回只读审计记录。未带 `project_key` 时返回平台最近记录；带
`project_key` 时只返回该项目的 `audit_events`。该接口用于 Web/Desktop/CLI 展示权限判断、
写入、approval、worker lease 和命令执行记录，不创建新的审计事件。

`GET /api/v1/worker-pool/summary` 返回跨项目 worker pool 的只读汇总，包括每个项目的 worker
数量、online worker、active lease、queued task、needs_recovery 和 capability 集合。该接口不执行
调度、不领取 lease、不 recovery lease，也不写入 event/audit。v0.8 的 summary / schedule preview /
readiness 边界见 [`v0.8-multi-project-worker-pool-contract.md`](./v0.8-multi-project-worker-pool-contract.md)。

`GET /api/v1/worker-pool/schedule-preview` 基于 worker pool summary 生成只读 dry-run 调度预览。
v0.8b 默认策略为 `default_fifo`；v0.8c 起每个项目读取 `project_scheduling_policies`
中的 `priority`、`max_parallel_tasks`、`agent_role`、`required_capabilities` 和
`engine_profile`。slot 计算使用 `min(online_workers - active_leases, max_parallel_tasks -
active_leases)`，下限为 0。该接口只返回 recommended / blocked、blocked_reasons、
available_slots、required_capabilities、agent_role、engine_profile、engine readiness 和 next_action，
不执行 `run-once`，不创建 lease，不写 event/audit，不复用真实 acquire lease path。调度合同见
[`worker-scheduling-contract.md`](../../../architecture/worker-scheduling-contract.md)；v0.8 阶段合同见
[`v0.8-multi-project-worker-pool-contract.md`](./v0.8-multi-project-worker-pool-contract.md)。
`recommended=true`、`available_slots>0` 和 `next_action=worker_run_once_preview` 都不是 scheduler
apply、lease claim、worker dispatch 或 execution cutover。v0.8d 的 engine readiness 只读取项目配置，
不解析 secret、不调用 engine；disabled profile 或不可用 secret 只体现在 `blocked_reasons` 中。v0.8e 额外解释
`resource_limits.max_active_leases` 与 `resource_limits.max_queued_tasks`，返回 resource readiness；
它仍不执行资源扣减或真实限流。v0.8f 返回 online worker 的 `worker_types` 和 agent role readiness；
`local_host` 可满足 `local_worker`，缺少匹配 worker 时只返回 `missing_agent_role:<role>`。
任何 budget / quota 信息在 v1.0 前也只能作为 blocked reason 或 estimate preview；schedule preview 不能
创建 budget reservation、扣减 quota、写 usage charge 或 silent throttle。

## Command API

Command API 发起会改变状态的动作。所有 command 必须经过 capability、gate、idempotency、risk
preview 和 audit。CLI、Web、Desktop 和 Worker 都必须复用同一 command 语义，不能各自维护状态或
绕过 API/service layer。
统一写入口、approval scope、permission 顺序、expected version/hash、rollback 和 safety facts 合同见
[`command-approval-contract.md`](./command-approval-contract.md)。
v1.x 高风险 real apply command 的状态词、apply packet、suspension rule 和 AreaMatrix first policy 见
[`high-risk-apply-ladder.md`](../../../../proposals/high-risk-apply.md)；API endpoint 存在不代表对应 rung 已打开。
v0.1 Import + Status Mirror 的最小 Command API 和 CLI 闭环见
[`v0.1-import-mirror-contract.md`](./v0.1-import-mirror-contract.md)；`export-status` 是 compatibility
alias，长期主语义是 `status-projections/apply`。

```text
POST /api/v1/projects
POST /api/v1/projects/{project_key}/import
POST /api/v1/projects/{project_key}/export-status
POST /api/v1/projects/{project_key}/status-projections/apply
POST /api/v1/projects/{project_key}/doctor
POST /api/v1/projects/{project_key}/cutover-apply
POST /api/v1/projects/{project_key}/workflow-versions
POST /api/v1/projects/{project_key}/workflow-versions/{version}/stages
POST /api/v1/projects/{project_key}/workflow-versions/{version}/gates
POST /api/v1/projects/{project_key}/workflow-versions/{version}/transition-previews
POST /api/v1/projects/{project_key}/workflow-versions/{version}/approvals
POST /api/v1/projects/{project_key}/workflow-versions/{version}/runner-preview
POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-queue
POST /api/v1/projects/{project_key}/workflow-versions/{version}/read-only-verify-queue
POST /api/v1/projects/{project_key}/workflow-versions/{version}/approved-artifact-write-queue
POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-project-write-queue
POST /api/v1/projects/{project_key}/workflow-versions/{version}/managed-generated-write-queue
POST /api/v1/projects/{project_key}/artifacts/archive-preview
POST /api/v1/projects/{project_key}/execution-forwarding-v1-apply
POST /api/v1/projects/{project_key}/workers
POST /api/v1/projects/{project_key}/workers/{worker_key}/heartbeat
POST /api/v1/projects/{project_key}/workers/{worker_key}/lease-acquire
POST /api/v1/projects/{project_key}/workers/{worker_key}/lease-release
POST /api/v1/projects/{project_key}/workers/{worker_key}/run-once
POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-execute
POST /api/v1/projects/{project_key}/workers/{worker_key}/read-only-verify
POST /api/v1/projects/{project_key}/workers/{worker_key}/approved-artifact-write
POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-project-write
POST /api/v1/projects/{project_key}/workers/{worker_key}/managed-generated-write
POST /api/v1/projects/{project_key}/workers/lease-recover
POST /api/v1/workflow-versions/{id}/promote-preview
POST /api/v1/approvals
POST /api/v1/runs
POST /api/v1/runs/{run_id}/start?project_key={project_key}
POST /api/v1/runs/{run_id}/drain?project_key={project_key}
POST /api/v1/runs/{run_id}/cancel?project_key={project_key}
```

`POST /api/v1/projects/{project_key}/execution-forwarding-v1-apply` 是 Execution Forwarding v1 的受保护
Command API。它消费 apply gate packet、执行幂等检查、写 command response、event 和 audit。当前在
`read_only_shim`、proof、approval 或 rollback 前置未闭合时必须返回 `blocked` / `denied`，并保持
`area_flow_run_created=false`、`area_flow_run_task_created=false`、`area_flow_attempt_created=false`、
`area_flow_artifact_created=false`、`task_loop_run_forwarded=false`、`project_write_attempted=false`、
`execution_write_attempted=false`、`engine_call_attempted=false`、`secrets_resolved=false` 和
`network_used=false`。endpoint 存在只证明受保护写入口已落地，不代表真实 forwarding、legacy task-loop
转发或 AreaMatrix execution cutover 已打开。

Command 请求应包含：

```text
idempotency_key
expected_version nullable
actor
project_scope
command_type
command_class
reason
risk_level
risk_policy
request_hash
permission_preview
approval_state
precondition_snapshot
affected_resources
forbidden_resources
rollback_or_remediation_ref
safety_facts
status
audit_event_id nullable
metadata
```

服务端必须基于标准化 payload 计算或校验 `request_hash`。同一 actor / scope / command_type /
idempotency_key 重复提交同一 request hash 返回同一结果；同一幂等键携带不同 request hash 必须拒绝。
R2-R4 command 还必须返回 permission preflight、risk preview、affected resources、approval/gate
结果、precondition snapshot、rollback/remediation、safety facts 和 audit outcome；不能只返回一个成功布尔值。
`command_class` 必须使用
[`command-approval-contract.md`](./command-approval-contract.md) 中的分类：`record_only`、
`projection_write`、`artifact_write`、`managed_project_write`、`execution_control`、`external_effect`
或 `migration_security`。包含多类副作用时按最高风险处理。

Command API 是唯一写入口。CLI 可以封装命令、Web 可以展示 approval console、Desktop 可以展示本机
控制面、Worker 可以提交 scoped lease 结果，但它们都不能绕过 command request 直接修改 PostgreSQL
业务状态、被管理项目文件、artifact metadata 或 worker lease。Bootstrap / admin 命令例如 migrate
可以有独立入口，但不得承载 workflow 业务状态转移。

当前实现中 `project.import`、`project.cutover.apply`、`workflow.version.create`、`workflow.approval.record`、`runner.preview`、
`run.fixture_queue`、`worker.fixture_execute`、`run.read_only_verify_queue`、`worker.read_only_verify`、
`run.approved_artifact_write_queue`、`worker.approved_artifact_write`、
`run.managed_generated_write_queue`、`worker.managed_generated_write`、`project.status_projection.write`、
`project.status_projection.apply`、`project.doctor.record`、`worker.register`、
`worker.heartbeat`、`lease.acquire`、`lease.release`、`lease.recover`、`run.start`、`run.drain`、`run.cancel`、
`artifact.archive.preview` 已经进入
`command_requests` 幂等边界。`project.import` 用于 AreaMatrix metadata index
重建、import run、status snapshot 和 import audit event；CLI/API 可传显式 `idempotency_key` 来重放同一
import 结果，未传键时每次 import 生成新的 command key，保留 import snapshot/history。
`project.cutover.apply` 当前只执行 v0.4 authoring cutover 的 AreaFlow DB 内状态切换：要求
cutover readiness 和 `cutover_readiness_gate` 通过，记录 workflow event、audit event 和 command
response，并显式返回 `project_write_attempted=false`、`execution_write_attempted=false`。它不写
AreaMatrix 文件、不创建 execution、不替代 task-loop，也不代表 execution cutover。
`runner.preview` 当前只执行 v0.5 dry-run execution preview，创建 run、run_task、copy / verify
run_attempt、`runner_preview_report` artifact、event、audit event 和 completed command response；
阶段合同见 [`v0.5-runner-preview-contract.md`](./v0.5-runner-preview-contract.md)。
response 显式返回 `project_write_attempted=false`、`execution_write_attempted=false`、
`area_matrix_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false` 和 `network_used=false`。
`workflow.approval.record` 用于记录 approval
record、workflow event 和 audit event，并允许 CLI/API 传入显式 `idempotency_key`；重复同一 approval
payload 返回同一审批事实。`project.doctor.record` 用于记录 `project.doctor.completed` event；
CLI/API 可传显式 `idempotency_key` 来重放同一 doctor report 记录，未传键时每次 doctor run 生成新的
command key，保留项目时间线。`project.status_projection.write` 是 legacy mirror export 记录边界；
`project.status_projection.apply` 是当前受保护投影写入边界，用于写 `.areaflow/status.json`，并记录
snapshot、status projection、event、audit event 和 command response。它显式返回
`apply_gate_status`、`apply_gate_decision`、`apply_gate_approval_status` 和 `apply_command_eligible`。
只有 gate pass 且 permission/path allow 后才允许 `project_write_attempted=true`；缺失或过期 packet 必须返回
`decision=denied`、`project_write_attempted=false`、`execution_write_attempted=false`、
`engine_call_attempted=false`；
`workflow/README.md`、`workflow/versions/**`、execution、progress、logs 和 checkpoint 写入仍未打开。
worker lifecycle command 覆盖 worker 注册和心跳；重复的 `worker.register` / `worker.heartbeat`
读取同一 command response 中的 worker、heartbeat、event 和 audit 事实，同一幂等键携带不同 request
hash 必须拒绝。worker lifecycle response 必须显式返回 `project_write_attempted=false`、
`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false`、`lease_created=false`、`attempt_created=false`、
`artifact_created=false` 和 `worker_run_once=false`。它只证明 AreaFlow 内部 worker registry 和
heartbeat 状态写入，不代表真实 execution、engine 调用、项目文件写入或远程 worker 凭证管理已经打开。
这些 worker / lease / scoped execution command 的阶段口径以
[`v0.6-worker-beta-contract.md`](./v0.6-worker-beta-contract.md) 为准；response 中的 scoped pass 不能累计成
`./task-loop run` forwarding 或 execution cutover proof。
worker lease command 覆盖任务领取、
释放和过期恢复；重复的 `lease.acquire` / `lease.release` 读取同一 command response 中的 lease
事实，显式 `lease.recover` 幂等键会读取同一批 recovered leases。周期性 recover 未显式传键时由服务端
生成新的 command key，避免后续扫尾被误判为旧请求。同一幂等键携带不同 request hash 必须拒绝。
lease command response 必须显式返回 `project_write_attempted=false`、
`execution_write_attempted=false`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false`、`attempt_created=false`、`artifact_created=false`
和 `worker_run_once=false`；capability denied 路径还必须返回 `decision=denied` 和
`lease_created=false`，不得创建 lease、attempt 或 artifact。
`run.start`、`run.drain`、`run.cancel` 当前只控制 dry-run run 的 AreaFlow DB 状态：`start`
允许 `queued -> running`，`drain` 允许 `running -> draining`，`cancel` 允许 queued run 变为
`cancelled`、running/draining run 变为 `cancelling`。三者都会写 event、audit event 和 command
response，并显式返回 `project_write_attempted=false`、`execution_write_attempted=false`、
`engine_call_attempted=false`、`task_claimed=false`、`worker_started=false`、`commands_run=false`、
`secrets_resolved=false` 和 `network_used=false`；它们不领取 task、不调用 engine、不写被管理项目、
不替代 task-loop。
`run.read_only_verify_queue` 创建非 dry-run read-only verify run/task，但排队阶段不读取项目文件；
`worker.read_only_verify` 必须先通过 execution approval gate、worker capability preflight 和 project path
allowlist，然后只读取 allowlisted target file，保存 sha256/size evidence，不保存 target file 原文。
response 显式返回 `project_read_attempted=true`、`project_read_allowed=true`、
`project_write_attempted=false`、`execution_write_attempted=false`、`engine_call_attempted=false`、
`commands_run=false`、`secrets_resolved=false` 和 `network_used=false`。
`run.approved_artifact_write_queue` 创建非 dry-run approved artifact write run/task，但排队阶段不读取项目文件、
不写 artifact store；`worker.approved_artifact_write` 必须先通过 execution approval gate、worker capability
preflight 和 project `write_artifacts` capability check，然后只写 AreaFlow-owned artifact store 与 PG
metadata/evidence。response 显式返回 `project_read_attempted=false`、`project_write_attempted=false`、
`execution_write_attempted=false`、`area_flow_artifact_written=true`、
`area_flow_execution_state_written=true`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false` 和 `network_used=false`。
`run.managed_generated_write_queue` 创建非 dry-run managed generated write run/task/write-set artifact，
但排队阶段不读取或写入 project file；`worker.managed_generated_write` 必须先通过 execution approval gate、
worker capability preflight、project permission allowlist、generated-only prefix policy 和 expected-before
hash/size check，然后只在 fixture/temp project 中执行 copy/verify/rollback drill。response 显式返回
`generated_only=true`、`generated_only_apply_open=true`、`project_read_attempted=true`、
`project_read_allowed=true`、`project_write_attempted=true`、`project_write_allowed=true`、
`execution_write_attempted=false`、`area_flow_artifact_written=true`、
`area_flow_execution_state_written=true`、`engine_call_attempted=false`、`commands_run=false`、
`secrets_resolved=false`、`network_used=false`、`write_set_passed=true`、`verification_passed=true`、
`rollback_attempted=true` 和 `rollback_verified=true`。真实 AreaMatrix 写入、保留 generated apply 结果、
source write、checkpoint、repair、engine、shell、secret、network 和 `workflow/versions/**/execution/**`
仍未打开。`generated_only_apply_open=true` 是当前 fixture/temp rollback drill 的兼容响应字段，不能解释为
retained managed-project apply 或真实 AreaMatrix generated apply 已打开。
`artifact.archive.preview` 当前只预演 retention / archive 决策，写 command response、event 和 audit，
并保持 `project_write_attempted=false`、`storage_write_attempted=false`、
`artifact_delete_attempted=false`。真实 copy/archive/upload/delete/GC 仍未打开；`project_reference` /
`external_project` 不因索引、hash 或 metadata 变成 AreaFlow-owned content。

以下能力必须通过 command request 表达，不能各自定义第二套状态机：

```text
approval decision
cutover apply
runner start
worker drain
worker cancel
artifact archive decision
release exception record
managed project write
engine / agent execution
object archive copy / GC apply
future restore apply
future publish apply
```

v0.7 Web 默认只读；approval、drain、cancel、archive、exception、restore 和 publish 等写动作，只有在
Command API、risk preview、permission preflight、approval/gate 和 audit outcome 全部稳定后才逐步打开。

## Runner Preview Response

`POST /api/v1/projects/{project_key}/workflow-versions/{version}/runner-preview` 返回同一套 execution
对象，供 CLI、Web 和 Desktop 共用：

```text
project
workflow_version
run
tasks[]
attempts[]
artifacts[]
preflight
created
idempotency_key
```

v0.5a 中 `artifacts[]` 至少包含一个 `runner_preview_report`，其 JSON 字段包括：

```text
artifact_type
storage_backend
uri
source_path
sha256
size_bytes
content_type
metadata
created_at
```

响应只返回 artifact metadata 和 URI，不把 report 原文塞进 JSON。

## Fixture Execution Responses

### Run Task Status Compatibility

v0.6 scoped execution API 已经暴露了 `passed`、`verified`、`artifact_written`、
`rollback_verified` 等 task/run 状态。长期模型从现在起收敛为：`run_task.status` 只表达通用
worker/runtime 生命周期，能力专属结果通过 response safety facts、`run_task.outcome`、attempt、
artifact、gate result 和 run summary 表达。

因此，现有 `verified`、`artifact_written`、`rollback_verified` 只能作为兼容读取值。新 API 不应继续
新增 capability-specific task status；新增能力应返回通用 `status=passed|failed|blocked|repair_needed`
加 outcome / evidence。

`POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-queue` 创建 approval-gated
fixture execution run。响应包含：

```text
project
workflow_version
run
task
created
idempotency_key
event_id
audit_event_id
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
```

`run.run_type = fixture_execution`，`run.run_kind = execution`，`run.dry_run = false`。该接口只写
AreaFlow PG state，不写被管理项目，不写 `workflow/versions/**/execution/**`。

`POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-execute` 在 execution approval gate
通过后领取 fixture task。响应包含：

```text
project
workflow_version
run
worker
lease
task
attempt
artifact
gate
status
decision
blockers
created
idempotency_key
event_id
audit_event_id
project_write_attempted=false
execution_write_attempted=false
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=true
worker_started=false
lease_created=true
attempt_created=true
artifact_created=true
```

成功路径创建 `leases.lease_kind = fixture_execution`、`run_attempts.attempt_kind = fixture_execution`
和 `fixture_execution_report` artifact，并将 fixture `run_task` 与 run 推进到 `passed`，outcome 语义为
`fixture_execution_passed`。它仍不执行
copy/verify/repair，不调用 Codex CLI，不运行 shell，不解析 secret，不访问网络，不写被管理项目文件。

## Read-only Verify Responses

`POST /api/v1/projects/{project_key}/workflow-versions/{version}/read-only-verify-queue` 创建
approval-gated read-only verify run。请求必须包含 `target_path`。响应包含：

```text
project
workflow_version
run
task
target_path
created
idempotency_key
event_id
audit_event_id
project_read_attempted=false
project_read_allowed=false
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
```

`run.run_type = read_only_verify`，`run.run_kind = execution`，`run.dry_run = false`。排队阶段只写
AreaFlow PG state，不读取 project file，不写被管理项目，不写 `workflow/versions/**/execution/**`。

`POST /api/v1/projects/{project_key}/workers/{worker_key}/read-only-verify` 在 execution approval gate
通过后领取 read-only verify task。响应包含：

```text
project
workflow_version
run
worker
lease
task
attempt
artifact
gate
target_path
target_sha256
target_size_bytes
status
decision
blockers
created
idempotency_key
event_id
audit_event_id
project_read_attempted=true
project_read_allowed=true
project_write_attempted=false
execution_write_attempted=false
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=true
worker_started=false
lease_created=true
attempt_created=true
artifact_created=true
verification_passed=true
```

成功路径创建 `leases.lease_kind = read_only_verify`、`run_attempts.attempt_kind = read_only_verify`
和 `read_only_verify_report` artifact。现有实现可能把 read-only verify `run_task` 与 run 推进到
兼容状态 `verified`；目标语义应解释为 `status=passed`、outcome `read_only_verify_passed`。
该路径只读取 project config allowlist 允许的 target file，并在 report 中保存 path、sha256 和 size；
不保存 target file 原文，不调用 Codex CLI，不运行 shell，不解析 secret，不访问网络，不写被管理项目文件。

## Approved Artifact Write Responses

`POST /api/v1/projects/{project_key}/workflow-versions/{version}/approved-artifact-write-queue` 创建
approval-gated approved artifact write run。请求可选包含 `artifact_label`。响应包含：

```text
project
workflow_version
run
task
artifact_label
created
idempotency_key
event_id
audit_event_id
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
```

`run.run_type = approved_artifact_write`，`run.run_kind = execution`，`run.dry_run = false`。排队阶段只写
AreaFlow PG state，不读取 project file，不写 artifact store，不写被管理项目，不写
`workflow/versions/**/execution/**`。

`POST /api/v1/projects/{project_key}/workers/{worker_key}/approved-artifact-write` 在 execution approval gate
通过后领取 approved artifact write task。响应包含：

```text
project
workflow_version
run
worker
lease
task
attempt
artifact
gate
artifact_label
status
decision
blockers
created
idempotency_key
event_id
audit_event_id
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=true
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=true
worker_started=false
lease_created=true
attempt_created=true
artifact_created=true
artifact_write_passed=true
```

成功路径创建 `leases.lease_kind = approved_artifact_write`、
`run_attempts.attempt_kind = approved_artifact_write` 和 `approved_artifact_write_report` artifact。现有实现
可能把 approved artifact write `run_task` 与 run 推进到兼容状态 `artifact_written`；目标语义应解释为
`status=passed`、outcome `artifact_write_passed`。该路径只写 AreaFlow-owned local
artifact store 和 PostgreSQL metadata/evidence，不读取项目文件、不调用 Codex CLI、不运行 shell、不解析
secret、不访问网络、不写被管理项目文件。

## Managed Generated Write Responses

`POST /api/v1/projects/{project_key}/workflow-versions/{version}/managed-generated-write-queue` 创建
approval-gated managed generated write run。请求必须包含 `target_path`、`content`、
`expected_before_sha256` 和 `expected_before_size`。响应包含：

```text
project
workflow_version
run
task
write_set_artifact
target_path
expected_before_sha256
expected_before_size
after_sha256
after_size
created
idempotency_key
event_id
audit_event_id
generated_only=true
generated_only_apply_open=true
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=true
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
```

`run.run_type = managed_generated_write`，`run.run_kind = execution`，`run.dry_run = false`。排队阶段只写
AreaFlow PG state 和 managed generated write-set artifact，不读取或写入 project file，不调用 engine，
不运行 shell，不解析 secret，不访问网络，不写 `workflow/versions/**/execution/**`。
`generated_only_apply_open=true` 在排队响应中只表示 fixture/temp generated-only drill 已进入当前受控链路，
不表示真实 managed-project retained apply 已打开。

`POST /api/v1/projects/{project_key}/workers/{worker_key}/managed-generated-write` 在 execution approval gate
通过后领取 managed generated write task。响应包含：

```text
project
workflow_version
run
worker
lease
task
copy_attempt
verify_attempt
rollback_attempt
write_set_artifact
preimage_artifact
artifact
gate
target_path
expected_before_sha256
expected_before_size
after_sha256
after_size
restored_sha256
restored_size
status
decision
blockers
created
idempotency_key
event_id
audit_event_id
generated_only=true
generated_only_apply_open=true
project_read_attempted=true
project_read_allowed=true
project_write_attempted=true
project_write_allowed=true
execution_write_attempted=false
area_flow_artifact_written=true
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=true
worker_started=false
lease_created=true
attempt_created=true
artifact_created=true
write_set_passed=true
verification_passed=true
rollback_attempted=true
rollback_verified=true
```

成功路径创建 `leases.lease_kind = managed_generated_write`、copy / verify / rollback attempts、
`managed_generated_write_preimage` artifact 和 `managed_generated_write_report` artifact。现有实现可能把
managed generated write `run_task` 与 run 推进到兼容状态 `rollback_verified`；目标语义应解释为
`status=passed`、outcome `generated_write_rollback_verified`。当前只允许 fixture/temp project
中的已存在普通 generated 文件，target path 必须位于 `.areaflow/generated/**` 或
`.areamatrix/generated/**`，并同时通过 `read_project` 与 `write_generated` allowlist。该路径会在 commit
前恢复 preimage hash/size；不保留 generated apply 结果，不打开真实 AreaMatrix 写入、source write、
checkpoint、repair、engine、shell、secret、network 或 `workflow/versions/**/execution/**`。
响应中的 `generated_only_apply_open=true` 仅限 fixture/temp rollback drill scope。

## Run And Artifact Query Responses

`GET /api/v1/projects/{project_key}/workflow-versions/{version}/runs` 返回 version-scoped run
metadata，供 Web run timeline 从版本进入具体 run。它不读取 task/attempt/artifact 原文、不触发 worker
调度、不改变 run 状态。

`GET /api/v1/runs/{run_id}` 是 Web run detail 的只读基础，返回：

```text
run
tasks[]
attempts[]
artifacts[]
```

它不触发 worker 调度、不读取 artifact 原文、不执行命令。`tasks[]`、`attempts[]` 和
`artifacts[]` 只来自 PostgreSQL metadata。

全局 run detail、run gate、run events、run event stream 和 run control route 都支持
`?project_key={project_key}` visibility guard。多项目、Web、Desktop、worker 和未来团队调用应传入
`project_key`；不匹配时返回 `404`，不进入后续 read 或 command handler。

`GET /api/v1/runs/{run_id}/execution-approval-gate` 返回真实 execution apply 前的只读门禁判断。响应包含：

```text
status = pass | blocked
mode = read_only_execution_approval_gate
items[]
blockers[]
warnings[]
required_capabilities[]
approval_found
approval_gate_found
live_mapping_gate_found
engine_preview
workers[]
forbidden_actions[]
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
attempt_created=false
artifact_created=false
generated_at
```

该 gate 只读取 run、run_task、workflow version、approval records、gate results、engine preview 和 worker
registry metadata。它不创建 `command_requests`，不领取 task，不启动 worker，不创建 attempt/artifact，
不解析 secret，不调用 engine，不运行 shell，不写被管理项目，也不写 `workflow/versions/**/execution/**`。
dry-run runner preview run 必须返回 `blocked`，因为真实 execution apply 只能从非 dry-run queued run
开始。

`GET /api/v1/runs/{run_id}/execution-plan` 返回真实 copy/verify/repair/checkpoint 打开前的只读执行计划。
响应包含：

```text
status = ready | blocked
mode = read_only_execution_plan_preview
gate
steps[]
steps[].key
steps[].attempt_kind
steps[].status
steps[].required_capabilities
steps[].prerequisites
steps[].blockers
steps[].reads_project
steps[].writes_project
steps[].writes_areaflow
steps[].uses_engine
steps[].runs_commands
steps[].uses_secrets
steps[].uses_network
steps[].creates_attempt
steps[].creates_artifact
blockers[]
forbidden_actions[]
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
attempt_created=false
artifact_created=false
generated_at
```

该接口只读取 run detail 和 execution approval gate，不创建 `command_requests`，不领取 task，不启动 worker，
不创建 lease/attempt/artifact，不读取或写入被管理项目，不调用 engine，不运行 shell，不解析 secret，
不访问网络，也不写 `workflow/versions/**/execution/**`。当前 `approved_artifact_write` 是唯一可在 gate
pass 后显示 `ready` 的 artifact-store-only step；`copy`、`checkpoint` 和 `repair` 必须保持 blocked /
waiting，直到后续高风险设计补齐 diff、rollback、engine、project write、git/checkpoint 和 repair gate。

`GET /api/v1/runs/{run_id}/project-write-design-gate` 返回 approved project write 的只读设计门禁。
响应包含：

```text
status = ready | blocked
mode = read_only_project_write_design_gate
gate
items[]
required_capabilities[]
write_set_fields[]
unsupported_operations[]
apply_sequence[]
blockers[]
forbidden_actions[]
project_write_apply_open=false
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
attempt_created=false
artifact_created=false
generated_at
```

该接口只读取 execution approval gate，并返回未来 write-set、copy/verify/repair/checkpoint 和 rollback
合同。它不创建 `command_requests`，不领取 task，不启动 worker，不创建 lease/attempt/artifact，不读取或
写入被管理项目，不调用 engine，不运行 shell，不解析 secret，不访问网络，也不写
`workflow/versions/**/execution/**`。即使 `status=ready`，也只表示设计合同已可查询；
`project_write_apply_open` 必须保持 `false`，真实 apply 仍需后续 fixture approved project write、
fixture verify、fixture rollback drill 和单独 approval。

`GET /api/v1/runs/{run_id}/managed-generated-write-gate` 返回 managed project generated-only write 的只读门禁。
响应包含：

```text
status = ready | blocked
mode = read_only_managed_generated_write_gate
gate
items[]
required_capabilities[]
allowed_generated_prefixes[]
required_write_set_fields[]
unsupported_operations[]
apply_sequence[]
blockers[]
forbidden_actions[]
generated_only_write_ready
generated_only_apply_open=false
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
lease_created=false
attempt_created=false
artifact_created=false
generated_at
```

该接口只读取 execution approval gate，并返回后续 managed-project generated-only write 的目录前缀、
write-set 字段、unsupported operations 和 apply sequence。它不创建 `command_requests`，不领取 task，
不启动 worker，不创建 lease/attempt/artifact，不读取或写入被管理项目，不调用 engine，不运行 shell，
不解析 secret，不访问网络，也不写 `workflow/versions/**/execution/**`。即使
`generated_only_write_ready=true`，`generated_only_apply_open` 也必须保持 `false`，表示当前只是打开
generated-only write apply 前的只读门禁。该 gate 的 `required_capabilities` 必须是
`read_project`、`write_artifacts` 和 `write_generated`，不能借用 source-write 级别的 `write_code`。

`GET /api/v1/runs/{run_id}/events?project_key={project_key}` 返回 run-scoped events，供 timeline/SSE
fallback 使用。

`GET /api/v1/projects/{project_key}/artifacts` 返回 project-scoped artifact metadata 列表，供 Web artifact
browser 使用。它不读取 artifact 原文，不解析 prompt/report 内容，不触发文件写入或命令执行。

`GET /api/v1/projects/{project_key}/residuals` 返回 project-scoped residual metadata 列表，供 Web
residual/blocker view 使用。它只展示导入或 AreaFlow 记录的索引状态，不把 residual 自动提升为
live task，也不触发 promotion、approval 或 execution。

`GET /api/v1/artifacts/{artifact_id}?project_key={project_key}` 返回 artifact metadata：

```text
artifact_type
storage_backend
uri
source_path
sha256
size_bytes
content_type
metadata
created_at
```

全局 artifact metadata 和 content route 同样支持 `?project_key={project_key}` visibility guard。
content route 在读取 AreaFlow-owned artifact 原文前先校验 metadata 的 project scope；不匹配时返回
`404`。v0.7 Web 默认只展示 metadata；artifact 原文读取/导出由后续 `artifact inspect/export`
权限边界处理。

`GET /api/v1/projects/{project_key}/workers` 返回 worker registry 和 heartbeat metadata，供 Web/Desktop
展示 worker health。它不记录 heartbeat、不领取 lease、不触发 `run-once`。

`GET /api/v1/projects/{project_key}/engines/codex-cli/preview` 返回受限 Codex CLI adapter preview，
供 CLI、Web、Desktop 和 worker beta 在打开真实 execution 前查看 engine/profile readiness、命令预览、
capability/path preflight 和 artifact redaction plan。响应包含：

```text
status = blocked | needs_approval | ready
mode = read_only_codex_cli_adapter_preview
engine
command
capabilities[]
paths[]
artifact_redaction
forbidden_actions[]
blockers[]
execution_allowed
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
generated_at
```

该接口只读取 project config、permission rows 和 engine profile metadata，不创建 command request、
不解析 secret、不调用 Codex CLI、不运行 shell、不写 artifact store、不写被管理项目、不写
`workflow/versions/**/execution/**`，也不代表 execution beta 已打开。默认 AreaMatrix config 应返回
`blocked`，因为 `codex-cli` profile disabled、`run_commands=false` 且 `execute_agents=false`。

## SSE

SSE 用于 Web、Desktop 和 CLI watch。

```text
GET /api/v1/events/stream?after_id={event_id}
GET /api/v1/projects/{project_key}/events/stream?after_id={event_id}
GET /api/v1/runs/{run_id}/events/stream?project_key={project_key}&after_id={event_id}
```

SSE 事件来自 `events` 和安全过滤后的状态变更，不是新的状态源事实。`after_id` 是可选的
event cursor；未提供时先返回当前 scope 内最近事件，再继续轮询新事件。SSE data payload 使用
普通 event response shape，只包含 JSON-safe metadata。

## Admin API

Admin API 是受限运维入口，不是第二套业务写入口。允许范围包括：

```text
service status / doctor
migration preview / apply
backup manifest / restore dry-run
project import / export-status
local bootstrap
support bundle preview
```

凡是会改变 workflow authoring、projection apply、project file、run、lease、worker control、secret、
restore、publish 或 release exception 的动作，都必须回到 Command API，并执行 permission、gate、
approval、idempotency 和 audit。Admin API 可以帮助 bootstrap 或诊断，但不能直接改写 workflow 主状态。
Operations / deployment / observability 的完整边界见
[`operations-deployment-observability-boundary.md`](../../../architecture/operations-deployment-observability-boundary.md)；support
bundle 在 v1.0 只能是 metadata-only preview，不能导出 prompt、secret、用户文件、raw artifact 或未脱敏日志。
Auth、team、token、secret 和 remote worker credential 相关 command 进入实现前，还必须先满足
[`auth-team-secret-boundary.md`](../../../../proposals/auth-team-secret.md) 的 R4 opening ladder。Endpoint 或 CLI
命令被列入本文只代表 API shape 被预留，不代表 enforcement、secret resolve 或 credential issuance 已打开。
External API connector、webhook delivery、inbound callback processing 和 provider notification 也必须走
[`integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)；任何 external-effect command 都不能由
Web、Desktop、plugin 或 worker 直接发起网络副作用。

## CLI 边界

CLI 是最先落地的 operator 工具。

```text
areaflow migrate
areaflow server
areaflow health
areaflow readiness
areaflow doctor
areaflow service status
areaflow ops readiness
areaflow ops readiness --json
areaflow ops migration-ledger-readiness
areaflow ops migration-ledger-readiness --json
areaflow support bundle-preview
areaflow support bundle-preview --json
areaflow backup manifest
areaflow backup manifest --project areamatrix
areaflow backup manifest --json
areaflow backup restore-plan
areaflow backup restore-plan --project areamatrix
areaflow backup restore-plan --json
areaflow release readiness
areaflow release readiness --project areamatrix
areaflow release readiness --json
areaflow release remediation-plan
areaflow release remediation-plan --project areamatrix
areaflow release remediation-plan --json
areaflow release acceptance-preview
areaflow release acceptance-preview --project areamatrix
areaflow release acceptance-preview --json
areaflow release acceptance-gate
areaflow release acceptance-gate --project areamatrix
areaflow release acceptance-gate --json
areaflow release exception-doctor
areaflow release exception-doctor --project areamatrix
areaflow release exception-doctor --json
areaflow release exception-record-preview
areaflow release exception-record-preview --project areamatrix
areaflow release exception-record-preview --json
areaflow release exception-schema-preview
areaflow release exception-schema-preview --project areamatrix
areaflow release exception-schema-preview --json
areaflow release exception-migration-approval-gate
areaflow release exception-migration-approval-gate --project areamatrix
areaflow release exception-migration-approval-gate --json
areaflow release exception-apply-preview
areaflow release exception-apply-preview --project areamatrix
areaflow release exception-apply-preview --json
areaflow release final-gate
areaflow release final-gate --project areamatrix
areaflow release final-gate --json
areaflow release evidence-bundle
areaflow release evidence-bundle --project areamatrix
areaflow release evidence-bundle --json
areaflow release package-preview
areaflow release package-preview --project areamatrix
areaflow release package-preview --json
areaflow release distribution-preview
areaflow release distribution-preview --project areamatrix
areaflow release distribution-preview --json
areaflow release publish-gate
areaflow release publish-gate --project areamatrix
areaflow release publish-gate --json
areaflow release publish-approval-preview
areaflow release publish-approval-preview --project areamatrix
areaflow release publish-approval-preview --json
areaflow release rollout-plan-preview
areaflow release rollout-plan-preview --project areamatrix
areaflow release rollout-plan-preview --json
areaflow audit coverage
areaflow audit coverage --project areamatrix --json
areaflow permissions doctor areamatrix
areaflow permissions doctor areamatrix --json
areaflow artifact integrity areamatrix
areaflow artifact integrity areamatrix --json
areaflow conformance check areamatrix
areaflow conformance check areamatrix --json
areaflow worker
areaflow project add/list/status/import/export-status/doctor/summary/readiness
areaflow project import-diff
areaflow project verify-bundle
areaflow project compatibility
areaflow project shim-preview
areaflow project shim-readiness
areaflow project shim-authorization
areaflow project status-projection-authorization
areaflow project status-projection-apply-packet
areaflow project status-projection-apply-gate
areaflow project cutover-readiness --version <label>
areaflow project execution-cutover-readiness
areaflow project execution-forwarding-v1-readiness
areaflow project execution-forwarding-v1-apply-preview
areaflow project execution-forwarding-v1-apply-packet
areaflow project execution-forwarding-v1-apply-gate
areaflow project execution-forwarding-v1-command-preview
areaflow project execution-forwarding-v1-rollback-preview
areaflow project execution-forwarding-v1-apply
areaflow workflow version create/list/show/stages/ensure-skeleton
areaflow workflow gate run/list
areaflow workflow transition preview/list
areaflow workflow approval record/list
areaflow workflow init/doctor/promote-preview
areaflow run preview
areaflow run fixture-queue
areaflow run read-only-verify-queue
areaflow run approved-artifact-write-queue
areaflow run fixture-project-write-queue
areaflow run managed-generated-write-queue
areaflow run execution-gate
areaflow run managed-generated-write-gate
areaflow run start/drain/cancel/status
areaflow engine codex-preview
areaflow worker register/list/heartbeat/lease-acquire/lease-release/lease-recover/run-once/fixture-execute/read-only-verify/approved-artifact-write/fixture-project-write/managed-generated-write
areaflow artifact inspect/export
```

`migrate` 等 bootstrap 命令可以直连 DB；业务命令长期应走 API/service layer。

`status-projection-authorization` / `status-projection-apply-packet` / `status-projection-apply-gate` 的 JSON 契约必须在真实 AreaMatrix
`.areaflow/status.json` 目标上暴露 `required_authorization_phrase`。该字段是 Package A 的机器可读精确授权短语，
用于让 CLI/API/Web 或脚本消费者不用解析 `items[].expected` 即可知道人工授权必须逐字提供什么。

## Web 边界

Web 是 dashboard 和 approval console。
v0.7 Web Dashboard 的详细阶段合同见
[`v0.7-web-dashboard-contract.md`](./v0.7-web-dashboard-contract.md)。本文列出 API shape；合同定义 Web 能否把
这些 API 暴露成只读面板、disabled control 或未来写动作。
v0.8 worker pool 面板只能展示
[`v0.8-multi-project-worker-pool-contract.md`](./v0.8-multi-project-worker-pool-contract.md) 定义的只读
summary / schedule preview / readiness 字段，不能从 preview 触发 scheduler apply、worker run-once、
lease recovery、secret resolve 或 project write。

v0.7 首屏能力：

```text
project list
summary/readiness
version timeline
stage board
run timeline
artifact browser
residual/blocker view
approval page
SSE live events
```

v0.7 初始 Web 以只读 dashboard 为主，只展示 approval records、run timeline、worker status 和 audit
trail。Approval、drain、cancel 等写操作要等对应 Command API、风险提示、影响范围、permission
preflight 和 audit outcome 都稳定后再逐步打开。Web 任何写操作都不能直接写 PG、项目文件、artifact
store 或 worker state。
Web 初始实现必须保持 `/api/v1` GET / SSE-only；`web/write-action-gate` 只展示 disabled/read-only 写动作
矩阵，不能由浏览器创建 command、approval、lease、attempt、artifact 或项目文件写入。
远程 Team Console 不是 v0.7 Web 的默认能力；远程 read-only / command console 必须按
[`team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md) 的 T4/T5 单独打开。

## Desktop 边界

Desktop 是本机服务管理器，不重写业务逻辑。
v0.9 的 local service status、dashboard launcher、service-control gate、notification gate、tray/menu gate
和 no-second-state 阶段合同见 [`v0.9-desktop-shell-contract.md`](./v0.9-desktop-shell-contract.md)。

```text
启动/停止 local service
查看 PG/API/worker 健康
管理本机 secret 来源
系统通知
菜单栏/托盘入口
打开 Web dashboard
显示 worker 状态
```

Desktop 不直接改项目文件，不直接跑 workflow，不维护第二数据库。

v0.9a 先落地 Desktop shell 可复用的只读 local-service status 契约：

```text
GET /api/v1/service/status?web_url=http://127.0.0.1:5174
GET /api/v1/desktop/service-control-gate
GET /api/v1/desktop/notification-gate
GET /api/v1/desktop/tray-menu-gate
areaflow service status --json --web-url http://127.0.0.1:5174
areaflow desktop service-control-gate --json
areaflow desktop notification-gate --json
areaflow desktop tray-menu-gate --json
```

Desktop shell 后续只能把这些状态作为观察面、dashboard launcher 和禁用门禁展示输入。v0.9 Desktop
只面向本机 local service，不承担团队远程控制台职责；远程团队/多用户控制台放到 v1.x 后续阶段。
真正的服务启动/停止、OS notification bridge、native tray/menu、secret 来源和 worker 操作仍必须继续走
AreaFlow API/service layer，并保留 permission、gate 和 audit 证据。
Desktop 可以管理本机 secret 来源的 readiness 和引用名，但在 R4 secret store / resolve 打开前不能读取、
显示、缓存或注入明文 secret。
如果未来 Desktop 承载团队登录 UI，它仍只是 Team Console 的 API client，不拥有第二套 session、
membership、worker、secret 或 command 状态。

## Worker 边界

Worker 通过 API 或 service layer 获取 lease、提交 attempt 和 artifact。Worker 不能直接扩大权限，也不能绕过 gate。

v0.6a 只开放 worker registry 和 heartbeat，不开放真实任务领取：

```text
GET /api/v1/projects/{project_key}/workers
POST /api/v1/projects/{project_key}/workers
POST /api/v1/projects/{project_key}/workers/{worker_key}/heartbeat
```

这些接口只维护 `workers`、`worker_heartbeats`、`events`、`audit_events` 和 completed
`command_requests` response。`leases` schema 已存在。register / heartbeat 只证明 worker registry
和 heartbeat 审计已经具备 command 幂等边界，不证明真实任务领取、engine 调用、项目文件写入或远程
worker 凭证管理已经打开。

v0.6b 开放最小 lease lifecycle，但仍不执行 copy/verify：

```text
POST /api/v1/projects/{project_key}/workers/{worker_key}/lease-acquire
POST /api/v1/projects/{project_key}/workers/{worker_key}/lease-release
POST /api/v1/projects/{project_key}/workers/lease-recover
```

同一 `run_task` 同时最多一个 active lease。过期 lease 进入 `needs_recovery`，不会被直接判失败。
`allowed_capabilities` 必须是 worker 注册 `capabilities` 的子集；否则返回 403。
lease-acquire / lease-release / lease-recover 均写入 completed `command_requests` response。Response
必须记录 no project / execution / engine / command / secret / network attempts，并且在 lease-only
路径保持 `attempt_created=false`、`artifact_created=false` 和 `worker_run_once=false`。

v0.6c 开放 dry-run worker run-once：

```text
POST /api/v1/projects/{project_key}/workers/{worker_key}/run-once
```

run-once 只在 AreaFlow 数据库内领取并释放 dry-run `run_task`，不执行 copy/verify、不调用 agent、
不写被管理项目。没有任务时返回 `claimed=false`。
request 可选携带 `run_id`；未携带时保持项目级队列领取，携带时只领取该 run 下的 eligible dry-run task。

v0.6d 起，run-once 成功领取任务时还会返回 dry-run evidence：

```text
worker_run_once.claimed
worker_run_once.lease
worker_run_once.task
worker_run_once.attempt
worker_run_once.artifact
```

`attempt.attempt_kind = worker_run_once`，`attempt.dry_run = true`。`artifact.artifact_type =
worker_run_once_report`，响应只返回 artifact metadata 和 URI。

v0.6e 起，run-once 同样执行 worker capability preflight。worker 缺少请求 capability 时返回：

```text
HTTP 403
error = worker capability denied
```

拒绝路径写入 event 和 audit event，不创建 lease、attempt 或 artifact。

v0.6f 起，run-once 支持按 run 定向：

```json
{
  "run_id": 3,
  "allowed_capabilities": ["read_project", "write_artifacts"]
}
```

该 scope 只限制本次自动领取；worker 仍不能绕过 project permission、capability preflight 或 lease policy。

v0.6i 开放 approval-gated fixture execution apply：

```text
POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-queue
POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-execute
```

`fixture-queue` 创建非 dry-run execution run 和一个 `fixture_execution_task`。`fixture-execute` 必须先通过
execution approval gate，再由满足 capability 的 online worker 领取 task、创建 completed lease、passed
attempt 和 `fixture_execution_report` artifact。该路径只写 AreaFlow state / artifact store，不调用 engine、
不运行 shell、不解析 secret、不访问网络、不写被管理项目，也不写 `workflow/versions/**/execution/**`。

v0.6j 开放 approval-gated read-only verify：

```text
POST /api/v1/projects/{project_key}/workflow-versions/{version}/read-only-verify-queue
POST /api/v1/projects/{project_key}/workers/{worker_key}/read-only-verify
```

`read-only-verify-queue` 创建非 dry-run execution run 和一个 `read_only_verify_task`。`read-only-verify`
必须先通过 execution approval gate，再由满足 `read_project` / `write_artifacts` capability 的 online
worker 领取 task；随后检查 project path allowlist，读取 target file，创建 completed lease、passed
attempt 和 `read_only_verify_report` artifact。该路径只保存 target path、sha256 和 size evidence，
不保存 target file 原文，不调用 engine、不运行 shell、不解析 secret、不访问网络、不写被管理项目，
也不写 `workflow/versions/**/execution/**`。

v0.6k 开放 approval-gated approved artifact write：

```text
POST /api/v1/projects/{project_key}/workflow-versions/{version}/approved-artifact-write-queue
POST /api/v1/projects/{project_key}/workers/{worker_key}/approved-artifact-write
```

`approved-artifact-write-queue` 创建非 dry-run execution run 和一个
`approved_artifact_write_task`。`approved-artifact-write` 必须先通过 execution approval gate，再由满足
`write_artifacts` capability 的 online worker 领取 task；随后只写 AreaFlow-owned artifact store 和
PostgreSQL metadata/evidence，创建 completed lease、passed attempt 和
`approved_artifact_write_report` artifact。该路径不读取项目文件、不调用 engine、不运行 shell、不解析
secret、不访问网络、不写被管理项目，也不写 `workflow/versions/**/execution/**`。

v0.6l 开放只读 execution plan preview：

```text
GET /api/v1/runs/{run_id}/execution-plan
areaflow run execution-plan <run-id>
```

该 preview 不创建 command、lease、attempt 或 artifact，只展示 execution approval gate、已打开的
approved-artifact-write step，以及仍被 blocker 关闭的 copy、checkpoint 和 repair step。它是 Web/Desktop
后续“下一步执行面”的只读数据源，不是 execution apply。

v0.6m 开放只读 approved project write design gate：

```text
GET /api/v1/runs/{run_id}/project-write-design-gate
areaflow run project-write-design-gate <run-id>
```

该 gate 不创建 command、lease、attempt 或 artifact，只展示 write-set contract、unsupported operations、
copy/verify/repair/checkpoint 分离、rollback contract 和 first apply sequence。即使 gate 返回
`ready`，`project_write_apply_open` 仍必须是 `false`，表示真实项目写入仍未打开。

v0.6n 开放 fixture-only approved project write：

```text
POST /api/v1/projects/{project_key}/workflow-versions/{version}/fixture-project-write-queue
POST /api/v1/projects/{project_key}/workers/{worker_key}/fixture-project-write
areaflow run fixture-project-write-queue <project> <version> --target-path <path> --content <text> --expected-before-sha256 <hash> --expected-before-size <size>
areaflow worker fixture-project-write <project> <worker-key> --run-id <id>
```

`fixture-project-write-queue` 创建非 dry-run execution run、一个 `fixture_project_write_task` 和
`fixture_project_write_set` artifact。`fixture-project-write` 必须先通过 execution approval gate，
再由满足 `read_project` / `write_artifacts` / `write_code` capability 的 online worker 领取 task；
随后只允许 fixture project 写入一个已存在、非 symlink、非目录、位于 project root 内的 allowlisted
target file。worker 必须校验 expected-before hash/size，写入 preimage artifact、copy attempt、verify
attempt、rollback attempt 和 `fixture_project_write_report` artifact，并在 commit 前把 fixture file
恢复到 preimage hash/size。该路径不调用 engine、不运行 shell、不解析 secret、不访问网络、不写真实
AreaMatrix，不写 `workflow/versions/**/execution/**`，也不开放 create/delete/move/chmod/binary/symlink/glob/root
escape。

v0.6o 开放只读 managed generated write gate：

```text
GET /api/v1/runs/{run_id}/managed-generated-write-gate
areaflow run managed-generated-write-gate <run-id>
```

该 gate 不创建 command、lease、attempt 或 artifact，只展示 managed-project generated-only write 的
allowed generated prefixes、required write-set fields、unsupported operations 和 apply sequence。它允许
Web/Desktop/CLI 在真正打开 generated-only apply 前看到 `generated_only_write_ready` 和 blockers，但
`generated_only_apply_open` 必须保持 `false`。该 gate 要求 `read_project` / `write_artifacts` /
`write_generated` capability，避免 generated-only 写入借用 `write_code`。真实 AreaMatrix 写入、source write、
checkpoint、repair、engine、shell、secret、network 和 `workflow/versions/**/execution/**` 仍未打开。

v0.6p 开放 fixture/temp managed generated write API/CLI surfacing：

```text
POST /api/v1/projects/{project_key}/workflow-versions/{version}/managed-generated-write-queue
POST /api/v1/projects/{project_key}/workers/{worker_key}/managed-generated-write
areaflow run managed-generated-write-queue <project> <version> --target-path <path> --content <text> --expected-before-sha256 <hash> --expected-before-size <size>
areaflow worker managed-generated-write <project> <worker-key> --run-id <id>
```

`managed-generated-write-queue` 创建非 dry-run execution run、一个 `managed_generated_write_task` 和
`managed_generated_write_set` artifact。`managed-generated-write` 必须先通过 execution approval gate，
再由满足 `read_project` / `write_artifacts` / `write_generated` capability 的 online worker 领取 task；
随后只允许 fixture/temp project 写入一个已存在、非 symlink、非目录、位于 project root 内的 generated-only
allowlisted target file。worker 必须校验 expected-before hash/size，写入 preimage artifact、copy attempt、
verify attempt、rollback attempt 和 `managed_generated_write_report` artifact，并在 commit 前把 generated
file 恢复到 preimage hash/size。该路径不调用 engine、不运行 shell、不解析 secret、不访问网络、不写真实
AreaMatrix，不保留 generated apply 结果，不写 `workflow/versions/**/execution/**`，也不开放 source write、
checkpoint、repair、create/delete/move/chmod/binary/symlink/glob/root escape。

v0.8a 起，worker pool 增加只读 summary：

```text
GET /api/v1/worker-pool/summary
areaflow worker pool-summary
```

summary 是多项目调度前的观测面。它只聚合 `projects`、`workers`、`leases` 和 `run_tasks`，
用于判断每个 project 的 queued / active / recovery / capability 状态，不改变任何任务状态。

v0.8b 起，worker pool 增加只读 schedule preview：

```text
GET /api/v1/worker-pool/schedule-preview
areaflow worker schedule-preview
```

schedule preview 只提供调度建议：

```text
recommended
blocked_reasons
available_slots
next_action = worker_run_once_preview | idle
policy.dry_run_only = true
```

真实 scheduler、resource limit enforcement、engine routing apply、secret resolve、remote worker credential
和 team permission enforcement 仍属后续显式 command / scheduler / R4 能力。

## 稳定字段

自动化、Web 和 Desktop 依赖稳定 JSON 字段。v0.2 起 `summary` / `doctor` / `readiness`
至少稳定提供：

```text
doctor.status
doctor.drift_status
doctor.stage_coverage_status
doctor.native_doctor_status
import.history_ready_for_diff
import.previous_source_hash
import.source_hash_changed_since_previous
import_diff.status
import_diff.changes[]
verification_bundle.status
verification_bundle.phase_gate
verification_bundle.summary
verification_bundle.readiness
verification_bundle.import_diff
verification_bundle.events[]
cutover_readiness.status
cutover_readiness.phase_gate
cutover_readiness.items[]
cutover_readiness.verification
cutover_readiness.compatibility
cutover_readiness.gates[]
execution_cutover_readiness.status
execution_cutover_readiness.items[]
execution_cutover_readiness.command_evidence
execution_cutover_readiness.safety_facts
workers[].worker_key
workers[].worker_type
workers[].status
workers[].last_heartbeat_at
leases[].id
leases[].status
leases[].run_task_id
leases[].expires_at
worker_run_once.claimed
worker_run_once.attempt
worker_run_once.artifact
approved_artifact_write.artifact_label
approved_artifact_write.artifact_write_passed
service.status
service.mode
service.api.status
service.database.status
service.worker_pool.status
service.worker_pool.total_projects
service.worker_pool.total_workers
service.worker_pool.total_queued_tasks
service.dashboard.url
service.dashboard.api_url
service.capabilities[]
service.forbidden_actions[]
desktop_service_control_gate.status
desktop_service_control_gate.mode
desktop_service_control_gate.actions[]
desktop_service_control_gate.actions[].key
desktop_service_control_gate.actions[].status
desktop_service_control_gate.actions[].default_ui_state
desktop_service_control_gate.actions[].blockers[]
desktop_service_control_gate.db_write_attempted
desktop_service_control_gate.project_write_attempted
desktop_service_control_gate.process_control_attempted
desktop_service_control_gate.command_created
desktop_service_control_gate.worker_scheduled
desktop_service_control_gate.workflow_execution_started
desktop_service_control_gate.secrets_resolved
desktop_service_control_gate.network_used
desktop_notification_gate.status
desktop_notification_gate.mode
desktop_notification_gate.actions[]
desktop_notification_gate.actions[].key
desktop_notification_gate.actions[].status
desktop_notification_gate.actions[].default_ui_state
desktop_notification_gate.actions[].blockers[]
desktop_notification_gate.event_stream_opened
desktop_notification_gate.notification_requested
desktop_notification_gate.command_created
desktop_notification_gate.worker_scheduled
desktop_notification_gate.workflow_execution_started
desktop_notification_gate.secrets_resolved
desktop_notification_gate.network_used
desktop_tray_menu_gate.status
desktop_tray_menu_gate.mode
desktop_tray_menu_gate.actions[]
desktop_tray_menu_gate.actions[].key
desktop_tray_menu_gate.actions[].status
desktop_tray_menu_gate.actions[].default_ui_state
desktop_tray_menu_gate.actions[].blockers[]
desktop_tray_menu_gate.tray_menu_created
desktop_tray_menu_gate.os_integration_requested
desktop_tray_menu_gate.command_created
desktop_tray_menu_gate.service_control_attempted
desktop_tray_menu_gate.notification_requested
desktop_tray_menu_gate.worker_scheduled
desktop_tray_menu_gate.workflow_execution_started
desktop_tray_menu_gate.secrets_resolved
desktop_tray_menu_gate.network_used
backup.status
backup.mode
backup.schema_version
backup.manifest_hash
backup.table_counts[]
backup.projects[]
backup.projects[].inventory
backup.projects[].artifact_count
backup.projects[].artifacts[]
backup.capabilities[]
backup.forbidden_actions[]
restore_plan.status
restore_plan.mode
restore_plan.schema_version
restore_plan.manifest_hash
restore_plan.projects[]
restore_plan.items[]
restore_plan.items[].key
restore_plan.items[].category
restore_plan.items[].status
restore_plan.items[].message
restore_plan.items[].metadata
restore_plan.capabilities[]
restore_plan.forbidden_actions[]
audit_coverage.status
audit_coverage.scope
audit_coverage.total_audit_events
audit_coverage.covered_requirements
audit_coverage.gap_requirements
audit_coverage.requirements[]
audit_coverage.requirements[].required_actions[]
audit_coverage.requirements[].missing_actions[]
permission_doctor.status
permission_doctor.mode
permission_doctor.project
permission_doctor.checks[]
permission_doctor.checks[].key
permission_doctor.checks[].category
permission_doctor.checks[].status
permission_doctor.checks[].message
permission_doctor.checks[].metadata
artifact_integrity.status
artifact_integrity.mode
artifact_integrity.project
artifact_integrity.checked_artifacts
artifact_integrity.passed_artifacts
artifact_integrity.warn_artifacts
artifact_integrity.failed_artifacts
artifact_integrity.skipped_artifacts
artifact_integrity.checks[]
artifact_integrity.checks[].artifact
artifact_integrity.checks[].status
artifact_integrity.checks[].message
artifact_integrity.checks[].metadata
workflow_versions[].display_label
workflow_versions[].lifecycle_status
workflow_versions[].import_mode
workflow_versions[].immutable
readiness.status
readiness.items[]
readiness.items[].key
readiness.items[].status
readiness.items[].message
readiness.items[].metadata
```
