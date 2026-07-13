# Execution Forwarding v1 Readiness Evidence

## Scope

本证据覆盖 `AF-V10-004A Execution Forwarding v1` 的只读 readiness surface。

它证明：

- `areaflow project execution-forwarding-v1-readiness <project> --json` 可以在 PostgreSQL fixture 中运行。
- `areaflow project execution-forwarding-v1-apply-preview <project> --json` 可以在 PostgreSQL fixture 中运行。
- `areaflow project execution-forwarding-v1-apply-packet <project> --json` 可以生成只读 apply packet preview。
- `areaflow project execution-forwarding-v1-apply-gate <project> --json` 可以消费 packet 字段并 fail closed。
- `areaflow project execution-forwarding-v1-apply <project> --json` 可以进入受保护 Command API；当前在 shim /
  proof / approval 未闭合时记录 blocked/denied command、event 和 audit，但不创建 run/task/attempt/artifact，
  不转发 `./task-loop run`，不写 AreaMatrix。
- `areaflow project execution-forwarding-v1-command-preview <project> --task-type <type> --json` 可以在 PostgreSQL
  fixture 中对 allowed、blocked 和 unknown task type 返回只读 command response preview。
- `areaflow project execution-forwarding-v1-rollback-preview <project> --json` 可以在 PostgreSQL fixture 中运行。
- Web dashboard 通过只读 GET surface 展示 readiness、apply-preview、apply-packet / Packet Gate、command-preview
  和 rollback-preview。
- Desktop shell 通过只读 GET surface 展示 readiness、apply-preview、apply-packet / Packet Gate、command-preview
  和 rollback-preview。
- readiness 会消费已完成的 `run.read_only_verify_queue` / `worker.read_only_verify` evidence。
- readiness 会消费已完成的 `run.approved_artifact_write_queue` / `worker.approved_artifact_write` evidence。
- readiness 会消费同项目最新 `completion.protected_path_proof.record` clean / authorized evidence，用于关闭
  legacy non-write proof 前置项。
- apply preview 会列出未来 apply packet fields、required proof facts、explicit approval、rollback target、
  forwarding target matrix、blocked target matrix 和 fail-closed response fields。
- apply packet preview 会输出 readiness snapshot hash、approval scope、allowed task types、target command matrix、
  canonical proof ids、idempotency key、audit correlation id、apply-gate command 和 future apply command，且不创建
  command/run。
- apply gate 会要求 `legacy_non_write_proof_id`、`rollback_plan_id` 与
  `protected_path_fingerprint_id` 精确匹配当前 readiness proof metadata 派生出的 project-scoped ref；裸 ID 或
  stale proof ref 必须 blocked。
- readiness snapshot hash 会包含当前 legacy proof event、rollback proof event 与 protected path fingerprint identity；
  proof 或 fingerprint 被替换时，旧 packet hash 会失效。
- apply gate 会在 missing packet 时 blocked/no_go；在完整 packet 且 rollback proof 已 pass 但
  `read_only_shim` 仍 blocked 时继续 blocked/no_go，并证明 `apply_command_eligible=false`。
- command preview 会对 `read_only_verify` 这类 allowed target 返回 `would_forward_after_approval`，但整体仍
  `status=blocked` / `apply_open=false`；对 `engine_execution` 和 unknown task type 返回 fail-closed
  decision，并保持所有实际 command/run/legacy/project/engine/network safety facts 为 false。
- rollback preview 会列出 fail-closed steps、reopen conditions、required proof facts 和继续禁止的 rollback actions。
- rollback preview 会在 apply 关闭、task-loop forwarding 禁用、legacy non-write proof clean 时，把
  `rollback_v1:fail_closed` 标成 pass。
- rollback preview 会消费同项目最新 `completion.execution_cutover_proof.record` complete evidence；当该 proof
  同时包含 rollback-specific facts 且安全事实保持 closed 时，`rollback_v1:proof_facts` 会在 fixture 中变为
  pass，但 reopen conditions 仍保持 blocked。
- allowed task scope 仅包含 read-only / evidence 类任务。
- `read_only_shim` 仍保持 blocked；rollback-to-read-only-shim 已可由同项目 complete execution cutover
  rollback proof 关闭；apply Command API / CLI 已落地，但在 gate 未通过时必须 fail closed。
- `forwarding_command_api` readiness item 为 `pass`，只证明受保护 Command API 存在；它不打开
  `./task-loop run` forwarding。
- `task_loop_run_forwarded`、legacy progress/log/checkpoint write、project write、execution write、engine、secret、
  network、source write、generated retained write、repair、checkpoint、publish 和 restore safety facts 均保持
  false。
- 默认跳过真实 AreaMatrix protected path fingerprint check；如需额外保护真实 AreaMatrix，设置
  `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。该模式会比较目标 `.areaflow/status.json` 与
  `workflow/README.md` 前后指纹、保护路径 git status，以及覆盖 shim scripts、`workflow/versions` 和 v1
  `progress.json` 的非目标保护路径递归内容指纹。

它不证明：

- `./task-loop run` 已转发。
- Execution Forwarding v1 已经允许真实转发或真实 execution cutover。
- Execution Forwarding v1 apply packet/gate 已经打开真实 apply。
- blocked/denied apply command response 可以被解释为 run/task/attempt/artifact 已创建。
- forwarding target matrix 中的 planned targets 已经可以真实执行。
- command preview 已经创建 AreaFlow command/run/task/attempt/audit。
- apply preview 已经创建 command、run、task、lease、attempt 或 artifact。
- rollback preview 已经创建 rollback command、删除 forwarding history 或执行 rollback。
- AreaMatrix read-only shim 已落地。
- 真实 AreaMatrix legacy non-write proof、read-only shim 或 rollback proof 已完成。
- fixture rollback proof 可以替代真实 AreaMatrix rollback apply 或真实 `./task-loop run` forwarding rollback。
- Web 或 Desktop 已经打开 apply、rollback、project write、service control 或 worker scheduling 操作入口。
- source write、generated retained write、repair、checkpoint、engine、secret、network、publish 或 restore 已打开。

## Focused Smoke

入口：

```bash
bash scripts/smoke-execution-forwarding-v1-readiness.sh
```

Docker PostgreSQL 入口：

```bash
make smoke-docker-execution-forwarding-v1-readiness
```

该 smoke 使用临时 AreaMatrix-like fixture root、临时 local artifact store 和隔离 PostgreSQL 数据库。流程为：

```text
migrate up
-> project add
-> project import
-> workflow version create
-> promotion / approval / live mapping gates
-> worker register
-> read-only verify queue + worker evidence
-> approved artifact write queue + worker evidence
-> protected-path-proof clean evidence
-> execution-cutover-proof complete evidence with scope binding and rollback facts
-> execution-forwarding-v1-readiness --json
-> execution-forwarding-v1-apply-preview --json
-> execution-forwarding-v1-apply-packet missing approval / unscoped proof ids blocked / complete canonical proof ids --json
-> execution-forwarding-v1-apply-gate missing packet / complete packet blocked by read-only shim after rollback proof passes --json
-> execution-forwarding-v1-apply complete packet blocked by read-only shim after rollback proof passes --json
-> forwarding target / blocked target fail-closed assertion
-> execution-forwarding-v1-command-preview allowed / blocked / unknown --json
-> execution-forwarding-v1-rollback-preview --json
-> safety facts assertion
-> optional real AreaMatrix protected path fingerprint check with AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1
```

期望结果：

```text
overall status = blocked
read_only_verify_evidence = pass
artifact_evidence = pass
read_only_shim = blocked
forwarding_command_api = pass
legacy_non_write_proof = pass
rollback_to_read_only_shim = pass
apply_preview.status = blocked
apply_preview.approval_status = needs_approval
apply_preview.apply_open = false
apply_preview.rollback_target = read_only_shim
apply_preview.target_policy = pass
apply_preview.forwarding_targets = allowed read-only / evidence tasks only
apply_preview.blocked_targets = source / generated / repair / checkpoint / engine / secret / network / publish / restore
apply_packet.status = blocked while read_only_shim is blocked
apply_packet.gate.apply_command_eligible = false
apply_gate.missing_packet = blocked/no_go
apply_gate.complete_packet = blocked/no_go while read_only_shim is blocked
apply_command.status = blocked
apply_command.decision = denied
apply_command.command_request_created = true
apply_command.area_flow_command_created = true
apply_command.area_flow_audit_event_created = true
apply_command.area_flow_run_created = false
apply_command.area_flow_run_task_created = false
apply_command.area_flow_attempt_created = false
apply_command.area_flow_artifact_created = false
apply_command.task_loop_run_forwarded = false
apply_command.project_write_attempted = false
apply_command.execution_write_attempted = false
apply_command.engine_call_attempted = false
apply_command.network_used = false
command_preview.allowed.status = blocked
command_preview.allowed.decision = would_forward_after_approval
command_preview.blocked.decision = blocked_task_type_fail_closed
command_preview.unknown.decision = unknown_task_type_fail_closed
rollback_preview.status = blocked
rollback_preview.rollback_apply_open = false
rollback_preview.rollback_target = read_only_shim
rollback_preview.fail_closed = pass
rollback_preview.proof_facts = pass
rollback_preview.reopen_conditions = blocked
```

该 smoke 是 readiness 证据，不是 forwarding apply 证据。

## Recent Result

2026-07-07 17:32 CST 运行：

```bash
go test ./internal/project -run 'TestBuildExecutionForwardingV1ApplyGate|TestBuildExecutionForwardingV1ApplyPacket|TestEvaluateExecutionForwardingV1Apply|TestExecutionForwardingV1ReadinessSnapshotHash' -count=1
go test ./internal/project ./internal/api ./internal/app -count=1
go test ./... -count=1
bash -n scripts/smoke-execution-forwarding-v1-readiness.sh
make smoke-docker-execution-forwarding-v1-readiness
git diff --check -- internal/project/execution_forwarding_v1_apply_gate.go internal/project/execution_forwarding_v1_apply_gate_test.go internal/project/execution_forwarding_v1_apply_packet_test.go internal/project/execution_forwarding_v1_apply_test.go scripts/smoke-execution-forwarding-v1-readiness.sh
git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- .areaflow/status.json workflow/README.md scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json
shasum -a 256 /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json /Users/as/Ai-Project/project/AreaMatrix/workflow/README.md
```

结果：

```text
PASS
isolated database: areaflow_smoke_20260707173225_70141
project: areamatrix-forwarding-fixture
workflow: forwarding-v1-smoke-20260707173225
isolated database dropped after smoke
AreaMatrix protected path git status: empty
0447b1b06ef6d7726d43f912a50aa70670383ad0c6897c938da98ab6137009ca  /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json
ae1d70fb996d3f0a57ed65076d2c8c10987112c5390e559212b551b24f2664eb  /Users/as/Ai-Project/project/AreaMatrix/workflow/README.md
```

新增验证到的关键事实：

- `legacy_non_write_proof_id` 必须等于
  `<project_key>:legacy_non_write_proof:<legacy_non_write_proof.metadata.proof_event_id>`。
- `rollback_plan_id` 必须等于
  `<project_key>:rollback_to_read_only_shim:<rollback_to_read_only_shim.metadata.proof_event_id>`。
- `protected_path_fingerprint_id` 必须等于
  `<project_key>:protected_path_fingerprint:<legacy_non_write_proof.metadata.protected_path_set_hash>`。
- `execution-forwarding-v1-apply-packet` 使用裸 proof ids 时保持 blocked/readiness_blocked，且三个 proof ref
  gate item 都返回 `*_missing_or_mismatch` blocker。
- readiness snapshot hash 会随 legacy proof event、rollback proof event 或 protected path fingerprint 改变。
- complete canonical proof refs 只能关闭 proof ref 字段；`read_only_shim` 未落地时，apply gate 和 apply command
  仍保持 blocked/no_go 或 blocked/denied。
- 本次 smoke 默认未启用真实 AreaMatrix fingerprint mode；随后单独确认真实 AreaMatrix 保护路径 git status 为空，
  且 `.areaflow/status.json` 与 `workflow/README.md` hash 与前次记录一致。

2026-07-07 15:29 CST 运行：

```bash
bash -n scripts/smoke-execution-forwarding-v1-readiness.sh
git diff --check -- internal/project/execution_forwarding_v1_readiness.go internal/project/execution_forwarding_v1_apply_preview.go internal/project/execution_forwarding_v1_apply_packet.go internal/project/execution_forwarding_v1_rollback_preview.go internal/project/execution_forwarding_v1_readiness_test.go scripts/smoke-execution-forwarding-v1-readiness.sh docs/development/execution-forwarding-v1-readiness-evidence.md
go test ./internal/project -run 'Test(ExecutionForwardingV1|BuildExecutionForwardingV1|EvaluateExecutionForwardingV1|ExecutionForwardingV1Apply)' -count=1
go test ./internal/api -run 'TestProjectExecutionForwardingV1' -count=1
AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 make smoke-docker-execution-forwarding-v1-readiness
```

结果：

```text
PASS
isolated database: areaflow_smoke_20260707152952_12396
project: areamatrix-forwarding-fixture
workflow: forwarding-v1-smoke-20260707152952
isolated database dropped after smoke
```

验证到的关键事实：

- `read_only_verify_evidence = pass`
- `artifact_evidence = pass`
- `read_only_shim = blocked`
- `forwarding_command_api = pass`
- `legacy_non_write_proof = pass`
- `rollback_to_read_only_shim = pass`
- `execution-forwarding-v1-apply-preview.status = blocked`
- `execution-forwarding-v1-apply-preview.apply_open = false`
- `execution-forwarding-v1-apply-preview.rollback_target = read_only_shim`
- `execution-forwarding-v1-apply-preview.target_policy = pass`
- `execution-forwarding-v1-apply-preview.forwarding_targets` 与 `allowed_task_types` 一致，并且每个 target 都保持
  `failure_mode=fail_closed`、`project_write_allowed=false`、`execution_write_allowed=false`、
  `legacy_fallback_allowed=false`
- `execution-forwarding-v1-apply-preview.blocked_targets` 覆盖 source write、generated retained write、repair、
  checkpoint、engine、secret、network、publish 和 restore，并断言 fail-closed safety facts 全为 false
- `execution-forwarding-v1-apply-packet` missing approval 返回 blocked/readiness_blocked，生成
  `readiness_snapshot_hash`、`approval_scope=execution_forwarding_v1_read_only_evidence_only`、
  `expected_shim_lifecycle_state=read_only_shim` 和 `failure_mode=fail_closed`，但 `gate.apply_command_eligible=false`
- `execution-forwarding-v1-apply-packet` complete proof ids 让 approval/proof/idempotency/audit packet fields 通过，
  但仍因 `read_only_shim` blocked 而保持 `status=blocked`、`decision=readiness_blocked`
- `execution-forwarding-v1-apply-gate` missing packet 返回 blocked/no_go，并暴露 readiness hash 与 explicit approval blocker
- `execution-forwarding-v1-apply-gate` complete packet 在 allowed task types、readiness hash、approval、proof ids 和
  fail-closed mode 都通过，且 rollback-to-read-only-shim proof 已闭合时，仍因 read-only shim 未落地而 blocked/no_go
- `execution-forwarding-v1-apply-gate` 把 `rollback_to_read_only_shim` 作为硬门禁；只有 complete rollback proof
  facts 通过才能关闭该项，`rollback_plan_id` 字段非空不足以让 `apply_command_eligible=true`
- `execution-forwarding-v1-apply` 在同一 complete packet 下创建受保护 command response、event 和 audit，返回
  `status=blocked` / `decision=denied`
- `execution-forwarding-v1-apply` 保持 `area_flow_command_created=true`、
  `area_flow_audit_event_created=true`，但 `area_flow_run_created=false`、
  `area_flow_run_task_created=false`、`area_flow_attempt_created=false`、`area_flow_artifact_created=false`
- `execution-forwarding-v1-command-preview(read_only_verify).decision = would_forward_after_approval`，但
  `status=blocked`、`apply_open=false`、`area_flow_command_created=false`、`task_loop_run_forwarded=false`
- `execution-forwarding-v1-command-preview(engine_execution).decision = blocked_task_type_fail_closed`
- `execution-forwarding-v1-command-preview(surprise_task).decision = unknown_task_type_fail_closed`
- `execution-forwarding-v1-rollback-preview.status = blocked`
- `execution-forwarding-v1-rollback-preview.rollback_apply_open = false`
- `execution-forwarding-v1-rollback-preview.rollback_target = read_only_shim`
- `execution-forwarding-v1-rollback-preview.fail_closed = pass`
- `execution-forwarding-v1-rollback-preview.proof_facts = pass` after same-project complete execution cutover proof with
  rollback facts
- `execution-forwarding-v1-rollback-preview.reopen_conditions = blocked`
- `task_loop_run_forwarded = false`
- `project_write_attempted = false`
- `execution_write_attempted = false`
- `engine_call_attempted = false`
- `secrets_resolved = false`
- `network_used = false`
- 默认不读取真实 AreaMatrix；显式 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` 时才比较真实
  `.areaflow/status.json` 与 `workflow/README.md` 目标指纹、保护路径 git status，以及非目标保护路径递归内容
  指纹。非目标集合覆盖 shim scripts、`workflow/versions` 和
  `workflow/versions/v1-mvp/execution/_shared/progress.json`，可防止 focused smoke 意外改动真实 AreaMatrix
  保护面。

## Web / Desktop Observation Surface

2026-07-07 更新：

- Web `web/src/api.ts` 增加以下只读 API helper：
  - `projectExecutionForwardingV1Readiness`
  - `projectExecutionForwardingV1ApplyPreview`
  - `projectExecutionForwardingV1ApplyPacket`
  - `projectExecutionForwardingV1CommandPreview`
  - `projectExecutionForwardingV1RollbackPreview`
- Web `web/src/App.tsx` 增加 `Forwarding v1` 只读 panel，展示 allowed task scope、apply-open、
  rollback-open、forwarding/blocked target counts、required gate、safety facts 和 blocker 状态。
- Web `Forwarding v1 Packet Gate` 面板展示 `readiness_snapshot_hash`、`legacy_ref`、`rollback_ref`、
  `fingerprint_ref` 和关键 gate item 的 expected / actual / blockers，便于人工审阅 canonical proof refs 是否
  绑定当前 readiness metadata。
- Web `Forwarding v1 Command Preview` 面板展示 `read_only_verify` 和 `engine_execution` 两个只读 response
  preview，确认 allowed-after-approval 与 fail-closed blocked 两种 decision。
- Desktop `desktop/src/main.ts` 增加同五条只读 endpoint 的加载和 panel 展示，其中 apply-packet response
  内嵌 gate，并渲染 `Forwarding v1 Packet Gate` 只读审阅面板。
- 这些 surface 不提供 apply/rollback button，不创建 command，不写 DB，不写 AreaMatrix，不调度 worker，
  不启动 `./task-loop run`。

验证要求：

```bash
cd web && npm run build
cd desktop && npm run build
make smoke-docker-web
```

2026-07-04 04:52 CST 运行：

```bash
make smoke-docker-web
```

结果：

```text
PASS
isolated database: areaflow_smoke_20260704045248_21874
project: areamatrix-web-fixture-20260704045249
workflow: web-smoke-20260704045249-ready
```

该 smoke 通过 Playwright 打开真实 Web dashboard，确认页面只发 `/api/v1` GET / SSE 请求。该历史运行覆盖
Execution Forwarding v1 readiness、apply-preview、command-preview、rollback-preview 四条只读 endpoint 与面板文案；
apply-packet / Packet Gate 覆盖见下方 2026-07-07 18:17 CST 结果。

2026-07-07 18:17 CST 运行：

```bash
npm run build --prefix web
npm run build --prefix desktop
node --check scripts/smoke-web-check.mjs
make smoke-docker-web
```

结果：

```text
PASS
isolated database: areaflow_smoke_20260707181722_27138
project: areamatrix-web-fixture-20260707181722
workflow: web-smoke-20260707181722-ready
```

该 smoke 额外覆盖
`GET /api/v1/projects/{project}/execution-forwarding-v1-apply-packet`、可见 `Packet Gate`、
`legacy_ref=` 和 `fingerprint_ref=`。Web / Desktop 只显示 apply packet / gate review，不调用
`POST /api/v1/projects/{project}/execution-forwarding-v1-apply`，不创建 command/run/task，不写 AreaMatrix，
不转发 `./task-loop run`。
