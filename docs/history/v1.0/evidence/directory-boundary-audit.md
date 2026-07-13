# Directory Boundary Audit

## Purpose

本文对应 backlog 任务
[`AF-P0-002 Directory Boundary Audit`](../plans/task-backlog.md#af-p0-002-directory-boundary-audit)。

目标是对照 [`../product/platform-blueprint.md`](../plans/platform-blueprint.md) 的长期目录边界，确认当前
AreaFlow 目录结构：

- 是否支持 0-100% 路线。
- 是否过早创建了空目录或空抽象。
- 哪些长期模块仍应暂存在 `internal/project`。
- 哪些目录应在后续阶段触发创建。

本文是审计记录，不创建 active task，不授权执行，也不写 AreaMatrix。

## Current Directory Snapshot

当前主要目录：

```text
cmd/areaflow
docs/adr
docs/architecture
docs/development
docs/dogfood
docs/migration
docs/milestones
docs/product
desktop
examples/areamatrix
governance
internal/adapter/areamatrix
internal/api
internal/app
internal/artifact
internal/config
internal/db
internal/doctor
internal/importer
internal/migrate
internal/project
internal/status
internal/workflow
migrations
schemas
scripts
tasks/active
tasks/backlog
tasks/done
tasks/templates
web
workflow/profiles/areamatrix
workflow/templates
```

当前未创建的长期目录：

```text
internal/auth/
internal/audit/
internal/engine/
internal/integration/
internal/ops/
internal/permission/
internal/runner/
internal/secret/
internal/worker/
```

这些缺口是有意保留，不是当前阻塞。Phase 0 / v0.1-v0.3 应避免为了目录完整性创建空模块。

## Boundary Assessment

| 目标边界 | 当前落点 | 状态 | 审计结论 |
|---|---|---|---|
| CLI entry | `cmd/areaflow` | present | 合理。CLI 只做命令入口，业务规则应在 service/internal 层。 |
| API service | `internal/api` | present | 合理。REST/SSE 边界存在。 |
| CLI orchestration | `internal/app` | present | 合理。继续避免把业务规则放进 app 层。 |
| Project state | `internal/project` | present / overloaded | 可接受。当前承载 workflow、runner preview、worker preview、release preview 等早期能力。 |
| Workflow profile engine | `internal/workflow` | present | 合理。当前主要承载 profile loader/hash/validation。 |
| Project adapters | `internal/adapter/areamatrix` | present | 合理。AreaMatrix adapter 内置到 v0.x。 |
| Artifact store | `internal/artifact` + `internal/project/*artifact*` | partial | 可接受。store 边界已出现，metadata/integrity/archive 仍在 project。 |
| Status projection | `internal/status` + `internal/project/status_projection_apply.go` | present | 合理。projection 是外部快照，不是主状态。 |
| Doctor | `internal/doctor` + `internal/project/*doctor*` | partial | 可接受。通用 doctor 与 project-specific report 仍可分阶段拆。 |
| Runner | `internal/project/runner.go` | deferred split | v0.5 后拆 `internal/runner`，当前不创建空目录。 |
| Worker | `internal/project/worker.go` | deferred split | v0.6 后拆 `internal/worker`，当前只保留 dry-run/lease lifecycle。 |
| Permission | `internal/project/permission_doctor.go` + config rules | deferred split | v0.6 后拆 `internal/permission`，高风险写入前必须先稳定。 |
| Engine | config/readiness docs only | deferred | v0.8 建接口；v1.0 前不解析 secret、不真实调用需要 secret 的 engine。 |
| Secret | config/readiness docs only | deferred | v1.x 真实 secret manager 前不创建可误用实现。 |
| Integration | schema/docs only | deferred | GitHub/webhook/外部 API 按 `docs/architecture/integration-webhook-boundary.md` 放 v1.x 逐级打开。 |
| Operations | `internal/project/service_status.go` + docs | deferred split | 本机 service status 可先留在 project 层；support bundle、telemetry、managed upgrade 和 remote ops 按 `docs/architecture/operations-deployment-observability-boundary.md` 分阶段打开。 |
| Auth / audit | migrations + `audit_events` usage | deferred split | schema 已有边界；实现目录 v1.0 前按能力打开。 |
| Web | `web` | present | 合理。v0.7 dashboard 已有，必须继续只走 API/SSE。 |
| Desktop | `desktop` | present | v0.9 Tauri shell scaffold 已创建，只接 `/api/v1/service/status`、`/api/v1/desktop/service-control-gate`、`/api/v1/desktop/notification-gate`、`/api/v1/desktop/tray-menu-gate` 和 dashboard launcher。 |
| Schemas | `schemas/status-projection.schema.json` | partial | 已开始沉淀 status projection JSON schema；API/profile/artifact schema 后续按稳定面继续补。 |
| Governance | `governance/README.md` and subdirs | partial | 目录存在，细分文档可随高风险能力补。 |

## Overloaded `internal/project`

当前 `internal/project` 包含多类能力：

```text
project store / config
workflow authoring
cutover apply
runner preview
run control
worker dry-run / lease lifecycle
artifact integrity / archive preview
backup / restore dry-run
permission doctor
conformance
release preview / final gate chain
service status
status projection apply
```

这是早期阶段可接受的集中式实现。原因：

- v0.1-v0.4 需要快速稳定 project scope、Command API、event/audit 和 PG state shape。
- 多数能力仍是 preview / dry-run / readiness，不应过早抽出看似稳定的接口。
- 过早拆 `runner`、`worker`、`permission`、`engine`、`secret` 会让目录像成熟平台，但能力尚未真正打开。

需要监控的风险：

- `internal/project` 文件数量已经较多，后续真实 execution 打开前必须拆出稳定边界。
- release preview chain 已集中在 `internal/project/release_*.go`，v1.0 前应保持只读 preview；真实 apply 不能继续塞进同一堆 helper。
- permission / audit 判断不能散落在 feature 文件里；R2-R4 打开前要有统一 evaluator。

## Stage-triggered Split Plan

| 阶段 | 触发动作 | 目标目录 | 拆分条件 |
|---|---|---|---|
| v0.4 | 稳定 stage/gate/transition authoring | `internal/workflow` | profile、stage engine、gate 和 transition 已有稳定接口。 |
| v0.5 | runner preview 进入可复用 service | `internal/runner` | run/task/attempt/preflight 类型稳定，dry-run 和真实 run 有清晰边界。 |
| v0.6 | worker lease / heartbeat / recovery 稳定 | `internal/worker` | worker registry、lease lifecycle、recovery 和 run-once 不再只是 project helper。 |
| v0.6 | R2-R3 操作准备打开 | `internal/permission` | capability、path、command、risk、approval、audit evaluator 可统一调用。 |
| v0.7 | API/Web schema 稳定 | `schemas/` | status projection schema 已落地；API response、profile、artifact schema 需要跨端固定时继续补。 |
| v0.8 | engine profile readiness 稳定 | `internal/engine` | engine profile、provider readiness、budget/rate limit 进入统一接口；真实 engine routing apply、budget/quota enforcement 仍按对应 architecture boundary 延后。 |
| v0.8 | secret readiness 与 future resolver 分离 | `internal/secret` | 仍不解析明文，但需要稳定 secret_ref policy 接口。 |
| v0.8+ | webhook / GitHub / external API 出现 | `internal/integration` | 外部系统接入不应进入 project 包；真实 delivery/callback/API call 必须按 integration boundary 另行开闸。 |
| v0.9 | Desktop shell 实现 | `desktop/` | Tauri shell 只接 `/api/v1/service/status`、`/api/v1/desktop/service-control-gate`、`/api/v1/desktop/notification-gate`、`/api/v1/desktop/tray-menu-gate` 和 dashboard launcher。 |
| v1.0 | local operations readiness 稳定 | `internal/ops` | service lifecycle、health/readiness、support bundle preview、telemetry redaction 和 migration ledger 从 project helper 变成复用边界；remote ops / managed upgrade 仍不打开。 |
| v1.0 | audit coverage 与 policy 稳定 | `internal/audit` | audit coverage、audit writing contract 和 retention policy 稳定。 |
| v1.0 | local/team identity 边界稳定 | `internal/auth` | users、actors、teams、api_tokens 从 schema 预留进入能力启用前。 |

## Explicit Non-actions

当前不创建：

- `internal/runner/`：runner preview 仍在 `internal/project`，真实 execution 未打开。
- `internal/worker/`：worker beta 仍以 lease/dry-run 为主。
- `internal/permission/`：真实 R2-R4 写入前再抽统一 evaluator。
- `internal/engine/`：v1.0 前不真实调用需要 secret 的 engine。
- `internal/secret/`：v1.0 前只做 readiness，不解析明文。
- `internal/integration/`：远程平台、webhook、GitHub 接入尚未打开；当前只有 schema/docs 边界。
- `internal/ops/`：当前只需要 service status 和文档边界；support bundle export、remote ops、managed upgrade
  和 destructive rollback 都未打开。
- `internal/auth/`：schema 预留存在，团队/远程 auth 是 v1.x。
- `internal/audit/`：audit events 已写入；独立 audit package 等 coverage/policy 稳定后再拆。

## Findings

1. 当前目录结构与 Phase 0 / v0.1-v0.4 目标兼容。
2. `internal/project` 当前偏大，但这是早期边界稳定前的有意集中。
3. 已经存在 `internal/workflow`，符合 v0.4 profile/stage engine 拆分方向。
4. 已经存在 `internal/artifact`，但 artifact metadata、integrity、archive preview 仍在 project 层；v1.0 前可接受。
5. `governance/{security,permissions,workflow,adapters}` 目录存在但大多无文件；后续高风险能力打开前应补治理文档，而不是让目录长期空置。
6. `desktop/` 已按 v0.9 scaffold 创建；`schemas/` 已以 status projection schema 形式开始落地，其他 schema 继续按稳定面补齐。
7. `internal/ops/` 暂不创建；当前运维边界已在文档层固定，service status 仍可留在 `internal/project`。
8. 没有发现需要立即迁移文件的目录阻塞。

## Gate Evidence

本审计关闭条件：

- 本文记录当前目录、长期目标、延迟拆分理由和阶段触发点。
- 不创建空目录。
- 不移动现有文件。
- `git diff --check -- docs/development/directory-boundary-audit.md schemas/README.md schemas/status-projection.schema.json scripts/validate-status-projection-schema.py`
- `go test ./...`
