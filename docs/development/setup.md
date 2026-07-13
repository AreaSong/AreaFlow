# Development Setup

## Phase 0

当前阶段只建立文档基线，不要求本地运行 AreaFlow。

## Phase 0.1 工程骨架

当前仓库已经有最小 Go binary、local REST health endpoint、PostgreSQL Docker Compose 和 v0.1 core schema migration。

## Requirements

- Go 1.23+。
- Docker 或兼容 Docker Compose 的本地容器环境。

## 规划中的 v0.1 开发形态

```text
Docker Compose PostgreSQL
Go binary: areaflow
REST/JSON local service
SSE events
local artifact store
```

示例项目配置 `examples/areamatrix/areaflow.yaml` 遵循
[`docs/architecture/project-config.md`](../architecture/project-config.md)，并使用平台级 artifact root：

```text
~/.areaflow/artifacts
```

AreaFlow 写入新 artifact 时会自动追加 `{project_key}` 子目录，例如
`~/.areaflow/artifacts/areamatrix/...`。

规划命令：

```bash
make smoke-docker
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  make smoke-web
docker compose up -d
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow migrate up
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project add --config examples/areamatrix/areaflow.yaml
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project status areamatrix
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project import areamatrix
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow version create areamatrix v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow version stages areamatrix v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow version mark-ready areamatrix v2 --stage queue --item-type queue_candidate
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow version mark-ready areamatrix v2 --stage promotion_preview --item-type promotion_preview
go run ./cmd/areaflow workflow profile list --json
go run ./cmd/areaflow workflow profile check areamatrix
go run ./cmd/areaflow workflow profile show areamatrix --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow conformance check areamatrix --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release readiness --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release remediation-plan --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release acceptance-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release acceptance-gate --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-doctor --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-record-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-schema-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-migration-approval-gate --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-apply-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release final-gate --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release evidence-bundle --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release package-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release distribution-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release publish-gate --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release publish-approval-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release rollout-plan-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate run areamatrix v2 profile_binding_drift
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate run areamatrix v2 discussion_gate
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate run areamatrix v2 plan_doctor
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate list areamatrix v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow transition preview areamatrix v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow approval record areamatrix v2 --decision rejected --reason "blocked transition preview"
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow approval list areamatrix v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate run areamatrix v2 approval_gate
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate run areamatrix v2 live_mapping_gate
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow version list areamatrix
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project status-projection-authorization areamatrix --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project status-projection-apply-packet areamatrix --json \
    --explicit-approval \
    --approval-actor local-dev \
    --approval-reason "local status projection apply"
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project compatibility areamatrix
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow project cutover-readiness areamatrix --version v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow workflow gate run areamatrix v2 cutover_readiness_gate
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow run preview areamatrix v2
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker register areamatrix --worker-key local-1
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker heartbeat areamatrix local-1
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker list areamatrix
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker pool-summary
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker schedule-preview
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow service status --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow backup manifest --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow backup restore-plan --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow audit coverage --project areamatrix --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow permissions doctor areamatrix --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow artifact integrity areamatrix --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker lease-acquire areamatrix local-1 --run-task-id 1
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker lease-release areamatrix local-1 --lease-id 1
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker lease-recover areamatrix
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker run-once areamatrix local-1
go run ./cmd/areaflow server
go run ./cmd/areaflow health
```

`make smoke-compatibility-fixture` 是 v0.4 compatibility / cutover-readiness 的安全 fixture smoke。
未设置 `AREAFLOW_DATABASE_URL` 时会明确跳过；设置后会创建临时 AreaMatrix-like root，只写临时
fixture 的 `.areaflow/status.json`，并断言：

```text
project compatibility <fixture> --json
blocked-path cutover readiness status = blocked
blocked-path cutover_readiness_gate.status = blocked
ready-path cutover readiness status = pass
ready-path cutover_readiness_gate.status = pass
cutover_apply_attempted = false
execution_write_attempted = false
./task-loop run remains blocked in compatibility contract
real AreaMatrix .areaflow/status.json unchanged
real AreaMatrix workflow/README.md unchanged
fixture workflow/versions/*/execution/** is not created
```

`make smoke-local` 是可重复的本机 PostgreSQL 主路径 smoke。未设置
`AREAFLOW_DATABASE_URL` 时会明确跳过。该脚本默认使用
`examples/areamatrix/areaflow.yaml`，该配置指向真实 AreaMatrix root；因此真实 root
读写都默认 fail closed：

- 如果 configured project root 解析为真实 AreaMatrix root，`project add/import/summary/doctor`
  之前必须显式设置 `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_READ=1`。
- 任何 status projection apply 都必须显式设置 `AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1`。
- 如果 configured project root 解析为真实 AreaMatrix root，写入 `.areaflow/status.json` 还必须额外设置
  `AREAFLOW_SMOKE_ALLOW_REAL_PROJECT_STATUS_APPLY=1`。
- fixture smoke 只设置 `AREAFLOW_SMOKE_ALLOW_STATUS_APPLY=1`，并把 project root 指向临时目录；
  它不需要真实 AreaMatrix read/write 放行变量。
- 真实 AreaMatrix 本机 apply 只能在单独授权后运行。

放行后会执行：

```text
migrate up
project add --config examples/areamatrix/areaflow.yaml
project import areamatrix
project summary areamatrix --json
project doctor areamatrix --json
project summary areamatrix --json
project status-projection-authorization areamatrix --json
project status-projection-apply-packet areamatrix --explicit-approval --json
project status-projection-apply areamatrix <packet-derived preimage/schema/protected-path/rollback args>
project readiness areamatrix --json
workflow version create areamatrix smoke-<timestamp> --json
workflow version ensure-skeleton areamatrix smoke-<timestamp> --json
workflow gate run areamatrix smoke-<timestamp> profile_binding_drift --json
workflow gate run areamatrix smoke-<timestamp> promotion_preview --json
workflow transition preview areamatrix smoke-<timestamp> --json
workflow approval record areamatrix smoke-<timestamp> --decision rejected --json
workflow gate run areamatrix smoke-<timestamp> approval_gate --json
workflow gate run areamatrix smoke-<timestamp> live_mapping_gate --json
project cutover-readiness areamatrix --version smoke-<timestamp> --json
workflow gate run areamatrix smoke-<timestamp> cutover_readiness_gate --json
workflow version create areamatrix smoke-<timestamp>-ready --json
workflow version mark-ready areamatrix smoke-<timestamp>-ready --stage queue --item-type queue_candidate --json
workflow version mark-ready areamatrix smoke-<timestamp>-ready --stage promotion_preview --item-type promotion_preview --json
workflow gate run areamatrix smoke-<timestamp>-ready promotion_preview --json
workflow transition preview areamatrix smoke-<timestamp>-ready --json
workflow approval record areamatrix smoke-<timestamp>-ready --decision approved --json
workflow gate run areamatrix smoke-<timestamp>-ready approval_gate --json
workflow gate run areamatrix smoke-<timestamp>-ready live_mapping_gate --json
project cutover-readiness areamatrix --version smoke-<timestamp>-ready --json
workflow gate run areamatrix smoke-<timestamp>-ready cutover_readiness_gate --json
run preview areamatrix smoke-<timestamp>-ready --json
run preview areamatrix smoke-<timestamp>-ready --risk-level high --json
worker register areamatrix --worker-key smoke-<timestamp>-ready-worker --capability read_project --capability write_artifacts --json
worker run-once areamatrix smoke-<timestamp>-ready-worker --run-id <runner-preview-run-id> --capability read_project --capability write_artifacts --json
worker register areamatrix --worker-key smoke-<timestamp>-ready-readonly --capability read_project --json
worker run-once areamatrix smoke-<timestamp>-ready-readonly --run-id <runner-preview-run-id> --capability read_project --capability write_artifacts --json
worker pool-summary --json
worker schedule-preview --json
service status --web-url http://127.0.0.1:5174 --json
backup manifest --json
backup restore-plan --json
audit coverage --project areamatrix --json
permissions doctor areamatrix --json
artifact integrity areamatrix --json
release readiness --json
release remediation-plan --json
release acceptance-preview --json
release acceptance-gate --json
release exception-doctor --json
release exception-record-preview --json
release exception-schema-preview --json
release exception-migration-approval-gate --json
release exception-apply-preview --json
release final-gate --json
release evidence-bundle --json
release package-preview --json
release distribution-preview --json
release publish-gate --json
release publish-approval-preview --json
release rollout-plan-preview --json
```

默认 workflow label 会使用 `smoke-<timestamp>`，避免重复使用同一 PostgreSQL 数据卷时撞到已存在
version。需要固定 label 时可设置 `AREAFLOW_SMOKE_WORKFLOW_VERSION`。

`make smoke-docker` 会使用本仓库 `docker-compose.yml` 启动 `postgres` 服务，等待
`pg_isready` 后用默认 URL 调用 `make smoke-local` 等价链路：

```text
postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable
```

因为默认 smoke script 是 `scripts/smoke-local.sh`，且默认 config 指向真实 AreaMatrix root，
`make smoke-docker` 同样会在真实 project read 前 fail closed。长链回归优先使用
`make smoke-docker-v1-stable-fixture`，它把 config 重定向到临时 root。

`make smoke-docker-completion-audit-release-candidate-snapshot` 使用同一个 Docker PostgreSQL 服务，只运行
`scripts/smoke-completion-audit-release-candidate-snapshot.sh` focused smoke。它创建临时 `areamatrix`
fixture project，使用 reviewed evidence URI 记录 E1-E9 proof，要求 completion audit 到 `complete`，
再尝试记录 `evidence_class=release_candidate` snapshot，并断言 fixture project identity 必须 fail closed；
snapshot readiness 必须保持 `blocked`，且不能出现 recorded snapshot。它证明 release-candidate snapshot
门禁不会被 fixture / mechanism evidence 冒充为真实 RC 证据，但不运行真实 AreaMatrix cutover、不发布
release、不执行 restore apply，也不写真实 AreaMatrix。设置
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` 时，脚本还会比对真实 AreaMatrix `.areaflow/status.json` 和
`workflow/README.md` 指纹，确认 focused smoke 未改动真实项目状态。

`make smoke-docker-completion-audit-real-identity-readiness` 使用隔离 Docker PostgreSQL smoke DB，只运行
`scripts/smoke-completion-audit-real-identity-readiness.sh`。它把真实
`/Users/as/Ai-Project/project/AreaMatrix` identity 注册/导入到临时 AreaFlow DB，查询
`completion audit-snapshot readiness`，并断言 blocker 是 `completion_audit_snapshot_missing`，不是 fixture
identity blocker。该 smoke 不记录 snapshot、不运行 worker、不发布 release、不执行 restore apply，也不写
AreaMatrix 文件；直接用已有 `AREAFLOW_DATABASE_URL` 运行该脚本时，必须显式设置
`AREAFLOW_REAL_IDENTITY_ALLOW_EXISTING_DB=1`，避免误写非隔离 AreaFlow DB。

`make smoke-docker-operations-proof` 使用同一个 Docker PostgreSQL 服务，只运行
`scripts/smoke-operations-proof.sh` focused smoke。它创建临时 `areamatrix` fixture project，验证
`ops smoke-proof record --key local_ops_smoke` 的 idempotent replay、event/audit/command response
持久化，以及 `ops readiness` 和 completion audit E7 消费 proof 后移除
`fresh_local_ops_smoke_missing`。它不运行长链 smoke、不控制 service process、不导出 support bundle、
不应用 migration、不上传 telemetry，也不写真实 AreaMatrix。

`make smoke-docker-managed-generated-write` 使用同一个 Docker PostgreSQL 服务，但只运行
`scripts/smoke-managed-generated-write.sh` focused smoke。它创建临时 fixture/temp project，验证
`write_generated` generated-only apply、non-fixture denial、copy/verify/rollback evidence。默认不读取真实
AreaMatrix；如需额外比较 `.areaflow/status.json` / `workflow/README.md` 指纹，设置
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。它不运行 `smoke-local.sh`。

`make smoke-docker-approved-artifact-write` 使用同一个 Docker PostgreSQL 服务，只运行
`scripts/smoke-approved-artifact-write.sh` focused smoke。它创建临时 fixture project 和临时 artifact store，
验证 approval-gated `run.approved_artifact_write_queue` / `worker.approved_artifact_write` 只写
AreaFlow-owned artifact evidence，并保持 project read/write、execution write、engine、command、secret 和
network 全部关闭。默认不读取真实 AreaMatrix；如需额外比较 `.areaflow/status.json` / `workflow/README.md`
指纹，设置 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。

`make smoke-docker-execution-plan` 使用同一个 Docker PostgreSQL 服务，只运行
`scripts/smoke-execution-plan.sh` focused smoke。它创建临时 fixture project，验证
`areaflow run execution-plan` / `GET /api/v1/runs/{run_id}/execution-plan` 只读展示 execution approval gate、
approved artifact write、copy、verify、checkpoint 和 repair 的状态与 blockers，并确认预览前后
command/run/task/lease/attempt/artifact/event/audit/heartbeat 计数不变。默认不读取真实 AreaMatrix；如需额外比较
`.areaflow/status.json` / `workflow/README.md` 指纹，设置 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1`。

`make smoke-docker-areamatrix-readonly` 使用同一个 Docker PostgreSQL 服务，只运行
`scripts/smoke-areamatrix-readonly.sh`。它读取真实 AreaMatrix root，覆盖 import、doctor、summary、
readiness、import-diff、verify-bundle、shim-preview 和 shim-readiness，并校验真实
`.areaflow/status.json` 与 `workflow/README.md` 指纹不变。默认要求 AreaMatrix protected paths
干净；如果存在已知既有 dirty state，必须提供
`AREAFLOW_READONLY_REVIEWED_DIRTY_OUTPUT_SHA256=<sha256>` 和
`AREAFLOW_READONLY_DIRTY_REVIEWER=<reviewer>`，且脚本会在结束时要求 reviewed dirty output 完全不漂移。

`make smoke-docker-shim-authorization-preflight` 是 AF-V04 compatibility shim 授权前的同义入口。它复用
同一条真实 AreaMatrix read-only smoke，但把用途固定为验证 status projection authorization、
apply-packet/gate、shim authorization、shim apply-packet/gate、required preflight、rollback scope 和 no-write
safety facts；它仍不授权、不写入、不转发 `./task-loop run`。

`make smoke-package-a` 是真实 AreaMatrix Package A 的本机授权前检查入口。它读取真实
`/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json` 和 protected-path git status；无
`AREAFLOW_DATABASE_URL` 时不写 DB，有 DB 时只允许写 AreaFlow DB 来采集并绑定最新 import snapshot
`source_hash`。它不修改 AreaMatrix、不授权 apply。默认在 status projection 仍是 legacy schema、protected paths
未复核或缺少权威 `source_hash` 时 fail closed；authorization packet 会绑定当前 target preimage 的
`target_preimage.exists`、`target_preimage.sha256`、`target_preimage.size_bytes` 和
`accepted_preimage_schema_status`，并输出后续 `status-projection-apply-gate` 必须消费的非审批参数：
`--target`、`--expected-before-*`、`--schema-uri`、`--validator-preflight`、`--protected-path-check`、
`--protected-path-fingerprint-sha256`、`--rollback-action`、`--accept-preimage-schema`，以及已绑定到最新
import snapshot 的 `--source-hash`。其中 `--protected-path-fingerprint-sha256` 绑定非目标 protected paths
的内容指纹；真实 apply 写前会重新捕获并要求匹配，写后也会复核，漂移时 fail closed 并回滚
`.areaflow/status.json`。
若显式提供 `AREAFLOW_PACKAGE_A_SOURCE_HASH=<expected latest AreaFlow import snapshot hash>`，packet
仍必须重新从 DB 取权威 hash 并要求两者一致；缺少 DB 绑定、hash 缺失或 hash mismatch 时 packet 状态为
`blocked_needs_authoritative_source_hash`。审批参数
`--explicit-approval`、`--approval-actor`、`--approval-reason` 只作为
`post_authorization_required_arguments` 输出，必须等用户给出窄授权后才可用于真实 apply。如果 protected paths 已有预期中的既有脏状态，可以用
`bash scripts/audit-package-a-dirty-review.sh` 取得 `dirty_output_sha256`，再用
`AREAFLOW_PACKAGE_A_REVIEWED_DIRTY_OUTPUT_SHA256=<sha256>` 和
`AREAFLOW_PACKAGE_A_DIRTY_REVIEWER=<reviewer>` 复核精确 dirty output。该复核只说明当前脏状态已被审计，
不等于 `授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json`。

`make smoke-docker-package-a-fingerprint-parity` 使用隔离 Docker PostgreSQL 和临时 git fixture，不读取或写入
真实 AreaMatrix。它对比 Go `status-projection-authorization` 与
`scripts/audit-package-a-authorization-packet.sh` 实际输出的 `protected_path_fingerprint_sha256`：baseline 必须一致，
改 `.areaflow/status.json` 后必须仍一致且不变，改非目标 protected path 后必须仍一致且变化。Package A 脚本在
该 smoke 中显式取消 `AREAFLOW_DATABASE_URL`，避免进入真实 source-hash authority 分支。

`make smoke-docker-project-isolation` 使用同一个 Docker PostgreSQL 服务，只运行
`scripts/smoke-project-isolation.sh`。它创建两个临时 AreaFlow project metadata fixture，并验证同名
workflow version、run、artifact、event、audit、worker 和 lease recovery 都按 `project_key` 隔离。
该 smoke 不读取或写入 AreaMatrix 文件，不运行 worker execution，不调用 engine，不解析 secret。

`make smoke-web` 是 v0.7 Web Dashboard 的浏览器 smoke。它需要 `AREAFLOW_DATABASE_URL`，
并依赖 `web/` 的 npm devDependencies。它会先调用 `scripts/smoke-local.sh` 播种一轮固定 workflow
数据，再启动 AreaFlow API server 和 Vite dev server，最后用 Playwright 打开真实页面并检查：

`make smoke-docker-web` 使用 Docker PostgreSQL 创建隔离数据库并运行同一套 Web browser smoke，适合
没有手动设置 `AREAFLOW_DATABASE_URL` 时复验 Web 只读 dashboard。

```text
Web 通过 /api/v1/projects 加载 project list
Web 通过 /api/v1/projects/<project>/summary 加载 summary
Web 通过 /api/v1/projects/<project>/workflow-versions 加载 version timeline
Web 通过 /api/v1/projects/<project>/workers 加载 worker status
Web 通过 /api/v1/worker-pool/summary 加载 worker pool
Web 通过 /api/v1/worker-pool/schedule-preview 加载只读 schedule preview
Web 通过 /api/v1/projects/<project>/execution-forwarding-v1-readiness 加载 forwarding readiness
Web 通过 /api/v1/projects/<project>/execution-forwarding-v1-apply-preview 加载 forwarding apply preview
Web 通过 /api/v1/projects/<project>/execution-forwarding-v1-rollback-preview 加载 forwarding rollback preview
页面展示 AreaMatrix、Stage Board、Run Timeline、Version Files、Approval Records、Worker Pool
页面展示 runner_preview、runner_preview_report、worker_run_once_report 和 dry-run
页面展示 codex-cli blocked、engine_profile_disabled 和 idle，不触发真实调度
页面展示 Forwarding v1 Read-only Scope、Apply Preview 和 Rollback Preview，不触发 apply/rollback
```

默认端口为 API `127.0.0.1:3857`、Web `127.0.0.1:5175`，可用
`AREAFLOW_WEB_SMOKE_API_PORT` 和 `AREAFLOW_WEB_SMOKE_WEB_PORT` 覆盖。

该 smoke 预期证明：

```text
summary.config.config_hash 非空
doctor.checks 包含 project_config_drift
doctor.project_config_drift = pass
summary.doctor.config_drift_status = pass
status_export writes .areaflow/status.json
readiness.status_mirror = pass
workflow_version.import_mode = authored
ensure-skeleton.links 包含 derives_from
profile_binding_drift.status = pass
promotion_preview.status = fail
transition_preview.status = blocked
approval_record.decision = rejected
approval_gate.status = blocked
live_mapping_gate.status = blocked
live_mapping_gate.inputs.execution_write_attempted = false
cutover_readiness.status = blocked
cutover_readiness.status_mirror = pass
cutover_readiness_gate.status = blocked
cutover_readiness_gate.inputs.cutover_apply_attempted = false
cutover_readiness_gate.inputs.execution_write_attempted = false
ready_path.promotion_preview.status = pass
ready_path.transition_preview.status = ready
ready_path.approval_record.decision = approved
ready_path.approval_gate.status = pass
ready_path.live_mapping_gate.status = pass
ready_path.live_mapping_gate.inputs.execution_write_attempted = false
ready_path.cutover_readiness.status = pass
ready_path.cutover_readiness_gate.status = pass
ready_path.cutover_readiness_gate.inputs.cutover_apply_attempted = false
ready_path.cutover_readiness_gate.inputs.execution_write_attempted = false
runner_preview.run.run_type = runner_preview
runner_preview.run.status = passed
runner_preview.run.dry_run = true
runner_preview.task.task_kind = workflow_item_preview
runner_preview.attempts 包含 copy / verify
runner_preview.artifact.artifact_type = runner_preview_report
runner_preview.artifact.sha256 非空
runner_preview.writes_attempted = false
runner_preview.commands_run = false
runner_preview.high_risk_without_allow = blocked
worker_register.status = online
worker_run_once.claimed = true
worker_run_once.lease.status = completed
worker_run_once.task.task_kind = workflow_item_preview
worker_run_once.attempt.attempt_kind = worker_run_once
worker_run_once.attempt.dry_run = true
worker_run_once.artifact.artifact_type = worker_run_once_report
worker_run_once.writes_attempted = false
worker_run_once.commands_run = false
release_readiness.status = needs_attention
release_remediation_plan.actions 包含 restore / audit / artifact
release_acceptance_preview.status = needs_decision
release_acceptance_preview.decisions 包含 metadata_only_history / future_only_gap / archive_exception
release_acceptance_preview.forbidden_actions 包含 mark_gap_accepted / create_approval / apply_release
release_acceptance_gate.status = blocked
release_acceptance_gate.items 包含 gate:accept:restore_plan / gate:accept:audit_coverage /
  gate:accept:artifact_integrity:<project>
release_acceptance_gate.forbidden_actions 包含 mark_gap_accepted / create_approval / apply_release
release_exception_doctor.status = warn
release_exception_doctor.checks 包含 exception_record_schema / exception_audit_contract /
  exception_write_guardrails
release_exception_doctor.forbidden_actions 包含 mark_gap_accepted / create_approval / apply_release
release_exception_record_preview.status = draft
release_exception_record_preview.drafts 包含 release_exception:restore_plan /
  release_exception:audit_coverage / release_exception:artifact_integrity:<project>
release_exception_record_preview.forbidden_actions 包含 insert_exception_record / insert_audit_event
release_exception_schema_preview.status = needs_approval
release_exception_schema_preview.tables 包含 release_exceptions
release_exception_schema_preview.forbidden_actions 包含 create_migration_file / run_migration
worker_capability_denied = missing write_artifacts
worker_pool.total_projects >= 1
worker_pool.projects[].project.key = areamatrix
worker_pool.projects[].worker_types includes local_host
worker_pool.projects[].capabilities includes read_project/write_artifacts
worker_pool.projects[].scheduling.priority = 100
worker_pool.projects[].scheduling.max_parallel_tasks = 1
worker_pool.projects[].scheduling.agent_role = local_worker
worker_pool.projects[].scheduling.engine_profile = codex-cli
worker_pool.projects[].role.status = ready
worker_pool.projects[].role.matched = true
worker_pool.projects[].engine.profile_id = codex-cli
worker_pool.projects[].engine.status = blocked
worker_pool.projects[].engine.blocked_reasons includes engine_profile_disabled
worker_pool.projects[].resources.max_active_leases = 1
worker_pool.projects[].resources.max_queued_tasks = 20
worker_schedule.policy.dry_run_only = true
worker_schedule.projects[].project.key = areamatrix
worker_schedule.projects[].max_parallel = 1
worker_schedule.projects[].agent_role = local_worker
worker_schedule.projects[].required_capabilities includes read_project/write_artifacts
worker_schedule.projects[].engine.status = blocked
worker_schedule.projects[].blocked_reasons includes engine_profile_disabled
worker_schedule.projects[].next_action = idle
service_status.status = ready
service_status.mode = local_service
service_status.dashboard.api_url = http://<AREAFLOW_HOST>:<AREAFLOW_PORT>/api/v1
service_status.dashboard.url = http://127.0.0.1:5174
service_status.worker_pool.total_projects >= 1
service_status.worker_pool.total_workers >= 1
service_status.worker_pool.total_queued_tasks >= 0
service_status.capabilities includes observe_api/open_web_dashboard
service_status.forbidden_actions includes maintain_second_database/run_workflow_directly
backup_manifest.status = ready
backup_manifest.mode = read_only_manifest
backup_manifest.schema_version = 1
backup_manifest.manifest_hash 非空
backup_manifest.table_counts includes projects/artifacts
backup_manifest.projects[].project.key = areamatrix
backup_manifest.projects[].artifact_count >= 1
backup_manifest.projects[].artifacts[] 只包含 metadata 和 URI，不读取 artifact 原文
backup_manifest.capabilities includes export_postgres_metadata/export_artifact_metadata
backup_manifest.forbidden_actions includes restore_database/read_artifact_contents
restore_plan.status = needs_attention
restore_plan.mode = read_only_restore_plan
restore_plan.schema_version = 1
restore_plan.manifest_hash 非空
restore_plan.projects[].key = areamatrix
restore_plan.items includes manifest_shape/project_inventory/artifact_inventory/dry_run_guardrails
restore_plan.items includes artifact_integrity:areamatrix
restore_plan.capabilities includes generate_restore_plan/check_artifact_integrity
restore_plan.forbidden_actions includes restore_database/apply_restore
restore_plan.artifact_inventory.referenced_artifacts >= 1
audit_coverage.status = warn
audit_coverage.mode = read_only_audit_coverage
audit_coverage.scope = project
audit_coverage.project_key = areamatrix
audit_coverage.total_audit_events >= 1
audit_coverage.requirements includes project_registration/status_mirror_write/workflow_authoring
audit_coverage.requirements includes approval_decision/runner_preview/worker_registration
audit_coverage.requirements includes worker_capability_denial
audit_coverage.missing_actions includes secret.resolve/permission.change
permission_doctor.status = pass
permission_doctor.mode = read_only_permission_policy_doctor
permission_doctor.checks includes project_config/default_read_only/status_export_write
permission_doctor.checks includes dangerous_write_denies/command_policy/secret_policy
permission_doctor.checks includes network_policy/worker_capability_policy/git_policy
permission_doctor.checks includes permission_audit_readiness
permission_doctor.status_export_write.path = .areaflow/status.json
permission_doctor.permission_audit_readiness.doctor_writes_audit = false
artifact_integrity.status = warn
artifact_integrity.mode = read_only_artifact_integrity
artifact_integrity.project.key = areamatrix
artifact_integrity.checked_artifacts >= 1
artifact_integrity.passed_artifacts >= 1
artifact_integrity.skipped_artifacts >= 1
artifact_integrity.checks includes storage_backend local with status pass
artifact_integrity.checks includes storage_backend external_project with status skipped
artifact_integrity.local check includes actual_sha256/read_contents true
artifact_integrity.external_project check includes read_contents false and project_reference_metadata_only
```

`promotion_preview.status = fail`、`transition_preview.status = blocked`、`approval_gate.status = blocked`、
`live_mapping_gate.status = blocked` 和 `cutover_readiness_gate.status = blocked` 是当前 v0.3/v0.4
安全预期：skeleton-only artifact 可以记录显式 rejected approval，但不能自动进入 approved approval、
live mapping、cutover apply 或 execution。

ready-path smoke 使用 `workflow version mark-ready` 把 queue 和 promotion preview 的 AreaFlow-owned
skeleton item 提升为 DB-only ready item。该命令只写 AreaFlow artifact store 和 DB，不写被管理项目的
`workflow/versions/**/execution/**`，也不代表 cutover apply。
`cutover_readiness_gate.status = pass` 只表示 verification、compatibility、approval、live mapping 和
rollback plan 的前置证据满足；它仍然是只读 gate，不会执行 cutover apply。

v0.5 runner preview 已纳入 `make smoke-local` 主路径，也可单独检查 JSON 输出：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow run preview areamatrix v2 --json
```

预期包含：

```text
run.dry_run = true
tasks[0].task_kind = workflow_item_preview
tasks[0].status = queued
attempts[].attempt_kind = copy / verify
artifacts[0].artifact_type = runner_preview_report
artifacts[0].sha256 非空
```

v0.6d worker run-once 已纳入 `make smoke-local` 主路径，也可单独检查 JSON 输出：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker run-once areamatrix local-1 --capability read_project --capability write_artifacts --json
```

预期包含：

```text
claimed = true
lease.status = completed
task.task_kind = workflow_item_preview
task.status = passed
attempt.attempt_kind = worker_run_once
attempt.dry_run = true
artifact.artifact_type = worker_run_once_report
artifact.sha256 非空
```

v1.0 adapter/profile conformance 可单独检查：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow conformance check areamatrix --json
```

预期包含：

```text
status = pass
mode = read_only_adapter_profile_conformance
profile_id = areamatrix
adapter = areamatrix
stage_count = 16
gate_count = 17
checks[].key = project_adapter_profile / profile_load / profile_validate / profile_stage_contract /
  profile_item_state_contract / profile_gate_contract / profile_transition_contract /
  profile_hard_rule_contract / profile_permission_policy_contract /
  profile_artifact_policy_contract / profile_cutover_policy_contract / adapter_snapshot /
  adapter_profile_boundary / project_config_policy
```

v1.0 release readiness 聚合检查：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release readiness --json
```

预期当前 baseline 包含：

```text
status = needs_attention
mode = read_only_release_readiness
backup.status = ready
restore_plan.status = needs_attention
audit_coverage.status = warn
projects[].artifact_integrity.status = warn
projects[].conformance.status = pass
items[].key = backup_manifest / restore_plan / audit_coverage / permission_policy:<project> /
  artifact_integrity:<project> / adapter_profile_conformance:<project>
```

v1.0 release remediation plan 可把 needs_attention 项转为关闭行动：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release remediation-plan --json
```

预期当前 baseline 包含：

```text
status = needs_attention
mode = read_only_release_remediation_plan
actions[].key includes remediate:restore_plan
actions[].key includes remediate:audit_coverage
actions[].key includes remediate:artifact_integrity:areamatrix
actions[].next_command includes areaflow backup restore-plan --json
actions[].next_command includes areaflow audit coverage --json
actions[].next_command includes areaflow artifact integrity areamatrix --json
forbidden_actions includes write_artifact_store / mark_gap_accepted / execute_commands
```

v1.0 release acceptance preview / gate 可区分可显式接受的 exception 和仍阻塞 release 的未决证据：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release acceptance-preview --json
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release acceptance-gate --json
```

预期当前 baseline 包含：

```text
acceptance_preview.status = needs_decision
acceptance_preview.decisions[].acceptance_type includes metadata_only_history / future_only_gap / archive_exception
acceptance_gate.status = blocked
acceptance_gate.items[].decision_status includes needs_decision
acceptance_gate.items[].status includes blocked
forbidden_actions includes mark_gap_accepted / create_approval / apply_release
```

v1.0 release exception record preview 可预演未来写入的 exception record、audit plan 和 rollback plan：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-record-preview --json
```

预期当前 baseline 包含：

```text
status = draft
mode = read_only_release_exception_record_preview
drafts[].key includes release_exception:restore_plan
drafts[].key includes release_exception:audit_coverage
drafts[].key includes release_exception:artifact_integrity:areamatrix
drafts[].audit_actions includes release.exception.request / release.exception.approve / release.exception.revoke
drafts[].rollback_plan is non-empty
drafts[].metadata.exception_writable = false
forbidden_actions includes insert_exception_record / insert_audit_event / apply_release
```

v1.0 release exception schema preview 可预演未来 migration，不生成 migration 文件、不写数据库：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-schema-preview --json
```

预期当前 baseline 包含：

```text
status = needs_approval
mode = read_only_release_exception_schema_preview
tables[].name includes release_exceptions
tables[].columns[].name includes exception_key / required_evidence / rollback_plan
tables[].indexes[].name includes release_exceptions_key_idx
tables[].foreign_keys[].references_table includes projects / actors / audit_events
apply_steps[].action includes create_table / create_index / audit_contract
rollback_steps[].action includes disable_writes / export_records / drop_table
forbidden_actions includes create_migration_file / run_migration / write_database
```

v1.0 release exception migration approval gate 可阻止未批准 migration：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-migration-approval-gate --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_exception_migration_approval_gate
schema_preview.status = needs_approval
items[].key includes migration_approval:release_exception_schema
items[].approval_status = needs_approval
items[].metadata.risk_level = R4 migration_security
items[].metadata.migration_writable = false
forbidden_actions includes create_migration_file / run_migration / approve_migration / write_database
```

v1.0 release exception apply preview 可预演未来 exception 写入和 release gate 重跑计划：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-apply-preview --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_exception_apply_preview
migration_gate.status = blocked
items[].key includes release_exception_apply:migration_approval
items[].action = wait_for_migration_approval
items[].metadata.risk_level = R4 migration_security
items[].metadata.apply_writable = false
apply_steps[].action includes verify_migration_approval / apply_release_exception_migration / write_exception_records
rollback_steps[].action includes disable_exception_writes / revoke_exception_records
forbidden_actions includes run_migration / insert_exception_record / apply_release / write_database
```

v1.0 release final gate 可聚合最终 go/no-go：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release final-gate --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_final_gate
readiness.status = needs_attention
acceptance_gate.status = blocked
exception_apply.status = blocked
items[].key includes final_gate:release_readiness
items[].key includes final_gate:release_acceptance
items[].key includes final_gate:release_exception_apply
forbidden_actions includes create_release_package / run_migration / insert_exception_record / apply_release
```

v1.0 release evidence bundle 可聚合发布证据索引：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release evidence-bundle --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_evidence_bundle
final_gate.status = blocked
backup.status = ready
audit_coverage.status = warn
items[].key includes evidence:release_final_gate
items[].key includes evidence:backup_manifest
items[].key includes evidence:audit_coverage
items[].key includes evidence:project_inventory:areamatrix
forbidden_actions includes create_release_package / read_artifact_contents / apply_release
```

v1.0 release package preview 可预演未来发布包 manifest：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release package-preview --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_package_preview
evidence_bundle.status = blocked
package_name = areaflow-v1.0-release-evidence-preview
items[].key includes package:manifest
items[].key includes package:evidence:release_final_gate
items[].package_path includes release/manifest.json
items[].metadata.package_writable = false
forbidden_actions includes create_release_package / read_artifact_contents / compress_artifacts / apply_release
```

v1.0 release distribution preview 可预演未来分发渠道：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release distribution-preview --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_distribution_preview
package_preview.status = blocked
items[].key includes distribution:package_preview
items[].key includes distribution:local_archive
items[].key includes distribution:git_release
items[].key includes distribution:artifact_registry
items[].metadata.publish_attempted = false
items[].metadata.release_write_allowed = false
forbidden_actions includes upload_release_artifacts / publish_release / create_git_tag / sign_release / push_git
```

v1.0 release publish gate 可阻止未准备好的分发进入真实发布：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release publish-gate --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_publish_gate
distribution_preview.status = blocked
items[].key includes publish_gate:distribution_preview
items[].key includes publish_gate:local_archive
items[].key includes publish_gate:git_release
items[].key includes publish_gate:artifact_registry
items[].metadata.publish_attempted = false
items[].metadata.publish_writable = false
forbidden_actions includes publish_release / create_git_tag / sign_release / push_git / apply_release
```

v1.0 release publish approval preview 可预演发布审批证据：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release publish-approval-preview --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_publish_approval_preview
publish_gate.status = blocked
items[].key includes publish_approval:publish_gate
items[].approval_status includes blocked
items[].metadata.approval_writable = false
items[].metadata.publish_writable = false
forbidden_actions includes create_approval / approve_release / publish_release / create_git_tag / sign_release / push_git / apply_release
```

v1.0 release rollout plan preview 可预演发布审批后的 rollout 阶段、验证点和回滚步骤：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release rollout-plan-preview --json
```

预期当前 baseline 包含：

```text
status = blocked
mode = read_only_release_rollout_plan_preview
publish_approval_preview.status = blocked
items[].key includes rollout_plan:publish_approval
items[].action includes wait_for_publish_approval_preview
rollout_steps includes verify_publish_approval
verification_checkpoints includes publish_approval_recorded
rollback_steps includes pause_distribution
items[].metadata.rollout_writable = false
items[].metadata.publish_attempted = false
forbidden_actions includes create_rollout / write_release_state / publish_release / create_git_tag / sign_release / push_git / apply_release
```

v1.0 release exception doctor 可在启用真实 exception 写入前检查字段、审计动作和写入 guardrail：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow release exception-doctor --json
```

预期当前 baseline 包含：

```text
status = warn
mode = read_only_release_exception_doctor
checks[].key includes exception_record_schema
checks[].key includes exception_audit_contract
checks[].key includes exception_write_guardrails
checks[].key includes exception:gate:accept:restore_plan
checks[].metadata.writes_enabled = false
checks[].metadata.exception_writable = false
forbidden_actions includes mark_gap_accepted / create_approval / apply_release
```

v0.6f worker run-once 可按 run 定向领取：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker run-once areamatrix local-1 --run-id 3 --capability read_project --capability write_artifacts --json
```

未传 `--run-id` 时保持项目级队列行为；传入后只领取该 run 下的 eligible dry-run task。

v0.6e worker capability preflight 可额外检查拒绝路径：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker register areamatrix --worker-key readonly-1 --capability read_project
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker run-once areamatrix readonly-1 --capability read_project --capability write_artifacts --json
```

第二条预期失败：

```text
worker capability denied
```

失败时不得创建新的 lease、attempt 或 artifact，但应写入 denied event / audit event。

v0.8a worker pool summary 已纳入 `make smoke-local` 主路径，也可单独检查 JSON 输出：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker pool-summary --json
```

预期包含：

```text
total_projects >= 1
total_workers >= 1
projects[].project.key = areamatrix
projects[].queued_tasks / active_leases / capabilities
projects[].scheduling.priority = 100
projects[].scheduling.max_parallel_tasks = 1
projects[].scheduling.agent_role = local_worker
projects[].scheduling.required_capabilities includes read_project/write_artifacts
projects[].worker_types includes local_host
projects[].role.status = ready
projects[].role.matched = true
projects[].engine.profile_id = codex-cli
projects[].engine.status = blocked
projects[].engine.blocked_reasons includes engine_profile_disabled
projects[].resources.max_active_leases = 1
projects[].resources.max_queued_tasks = 20
projects[].resources.status = ready
```

v0.8b worker pool schedule preview 已纳入 `make smoke-local` 主路径，也可单独检查 JSON 输出：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  go run ./cmd/areaflow worker schedule-preview --json
```

预期包含：

```text
policy.dry_run_only = true
projects[].recommended = false
projects[].blocked_reasons
projects[].next_action
projects[].max_parallel = 1
projects[].agent_role = local_worker
projects[].role.status = ready
projects[].role.matched = true
projects[].required_capabilities includes read_project/write_artifacts
projects[].engine.profile_id = codex-cli
projects[].engine.status = blocked
projects[].blocked_reasons includes engine_profile_disabled
projects[].resources.max_active_leases = 1
projects[].resources.max_queued_tasks = 20
projects[].resources.status = ready
```

v0.8g project isolation fixture 可以单独运行：

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  bash scripts/smoke-project-isolation.sh
```

或使用 Docker PostgreSQL：

```bash
make smoke-docker-project-isolation
```

未设置 `AREAFLOW_DATABASE_URL` 时，脚本会明确跳过：

```text
smoke-project-isolation: skipped; AREAFLOW_DATABASE_URL is not set
```

## Migrations

`areaflow migrate up` 按文件名顺序执行 embedded SQL migration，并用 `schema_migrations` 记录已应用文件。命令可重复运行；已应用 migration 会跳过。

当前 SQL 源文件位于：

```text
migrations/000001_v0_1_core.sql
migrations/000002_v0_3_command_requests.sql
migrations/000003_v0_3_gate_results.sql
migrations/000004_v0_3_approval_transition.sql
migrations/000005_v0_5_runner_preview.sql
migrations/000006_v0_6_worker_registry.sql
migrations/000007_v0_8_scheduling_policy.sql
migrations/000008_v0_3_workflow_item_links.sql
migrations/000009_v1_boundary_foundation.sql
internal/migrate/migrations/000001_v0_1_core.sql
internal/migrate/migrations/000002_v0_3_command_requests.sql
internal/migrate/migrations/000003_v0_3_gate_results.sql
internal/migrate/migrations/000004_v0_3_approval_transition.sql
internal/migrate/migrations/000005_v0_5_runner_preview.sql
internal/migrate/migrations/000006_v0_6_worker_registry.sql
internal/migrate/migrations/000007_v0_8_scheduling_policy.sql
internal/migrate/migrations/000008_v0_3_workflow_item_links.sql
internal/migrate/migrations/000009_v1_boundary_foundation.sql
```

测试会校验两份文件内容一致，防止 embed 副本漂移。

## 注意

- v0.1 不提供 SQLite fallback。
- v0.1 不执行任务。
- v0.1 不启动 AI engine。
- v0.1 不写被管理项目代码。
