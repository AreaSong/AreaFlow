# AreaFlow

AreaFlow 是一个 AI 开发项目管理平台，目标是从项目需求、workflow lifecycle、任务编排、AI agent 执行、证据记录到多项目可视化形成完整闭环。

当前状态：0-100% 地基讨论已经固化到产品、架构、迁移、milestone、backlog 和 development audit 文档中。仓库已有 Go CLI/API、PostgreSQL migrations、AreaMatrix adapter、workflow profile、runner/worker scoped evidence、Web Dashboard、Desktop Shell scaffold、release preview chain、operations readiness 和 completion audit 的受限实现证据。

当前仍不能声明真实 100%：AreaMatrix compatibility shim 真实落地、Execution Forwarding v1 apply、Execution Cutover、Archive、Shim Retirement、真实 release candidate evidence 和跨仓库 protected path proof 仍未完成。`preview_only` 与 `implemented_scoped` 只能证明对应边界可审计或可预演，不能解释为真实 AreaMatrix execution cutover、source write、repair、checkpoint、engine、secret、restore 或 publish 已打开。

## 核心方向

- CLI first，但不是 CLI only。
- PostgreSQL 作为主状态源事实。
- Go 作为 backend、CLI、scheduler 和 worker 的长期技术栈。
- REST/JSON API + SSE event stream 起步。
- React + TypeScript Web Dashboard 后续接入。
- Tauri Desktop Shell 最后接入。
- AreaMatrix 是第一个 dogfooding 项目。

## 0-100% 路线

| 范围 | 阶段 | 目标 |
|---:|---|---|
| 0-5% | Phase 0 | 产品、架构、技术决策、迁移路线和 v0.1 边界 |
| 5-15% | v0.1 | Import + Status Mirror |
| 15-25% | v0.2 | Shadow Doctor + Drift Check |
| 25-35% | v0.3 | New Version Authoring |
| 35-45% | v0.4 | Workflow Ownership Cutover |
| 45-55% | v0.5 | Runner Preview + Execution Model |
| 55-65% | v0.6 | Worker Execution Beta |
| 65-75% | v0.7 | Web Dashboard |
| 75-85% | v0.8 | Multi-project / Multi-worker |
| 85-92% | v0.9 | Desktop Shell |
| 92-100% | v1.0 | Stable Platform |

## 文档入口

- 产品定位：[docs/product/charter.md](docs/product/charter.md)
- 0-100% 总控计划：[docs/product/master-plan.md](docs/product/master-plan.md)
- 0-100% 平台蓝图：[docs/product/platform-blueprint.md](docs/product/platform-blueprint.md)
- 阶段 backlog：[docs/product/phase-backlog.md](docs/product/phase-backlog.md)
- 路线图：[docs/product/roadmap.md](docs/product/roadmap.md)
- 架构总览：[docs/architecture/overview.md](docs/architecture/overview.md)
- 项目接入协议：[docs/architecture/project-config.md](docs/architecture/project-config.md)
- workflow lifecycle：[docs/architecture/workflow-lifecycle.md](docs/architecture/workflow-lifecycle.md)
- workflow engine contract：[docs/architecture/workflow-engine-contract.md](docs/architecture/workflow-engine-contract.md)
- v0.1 import / mirror contract：[docs/architecture/v0.1-import-mirror-contract.md](docs/architecture/v0.1-import-mirror-contract.md)
- v0.2 shadow doctor contract：[docs/architecture/v0.2-shadow-doctor-contract.md](docs/architecture/v0.2-shadow-doctor-contract.md)
- v0.3 version authoring contract：[docs/architecture/v0.3-version-authoring-contract.md](docs/architecture/v0.3-version-authoring-contract.md)
- v0.4 workflow ownership cutover contract：[docs/architecture/v0.4-workflow-ownership-cutover-contract.md](docs/architecture/v0.4-workflow-ownership-cutover-contract.md)
- v0.5 runner preview contract：[docs/architecture/v0.5-runner-preview-contract.md](docs/architecture/v0.5-runner-preview-contract.md)
- v0.6 worker beta contract：[docs/architecture/v0.6-worker-beta-contract.md](docs/architecture/v0.6-worker-beta-contract.md)
- v0.7 web dashboard contract：[docs/architecture/v0.7-web-dashboard-contract.md](docs/architecture/v0.7-web-dashboard-contract.md)
- v0.8 multi-project worker pool contract：[docs/architecture/v0.8-multi-project-worker-pool-contract.md](docs/architecture/v0.8-multi-project-worker-pool-contract.md)
- v0.9 desktop shell contract：[docs/architecture/v0.9-desktop-shell-contract.md](docs/architecture/v0.9-desktop-shell-contract.md)
- v1.0 stable platform contract：[docs/architecture/v1.0-stable-platform-contract.md](docs/architecture/v1.0-stable-platform-contract.md)
- v0.1 数据模型：[docs/architecture/data-model-v0.1.md](docs/architecture/data-model-v0.1.md)
- v1 数据模型：[docs/architecture/data-model-v1.md](docs/architecture/data-model-v1.md)
- execution model：[docs/architecture/execution-model.md](docs/architecture/execution-model.md)
- API surface：[docs/architecture/api-surface.md](docs/architecture/api-surface.md)
- Command / approval contract：[docs/architecture/command-approval-contract.md](docs/architecture/command-approval-contract.md)
- adapter/profile 边界：[docs/architecture/adapter-profile-boundary.md](docs/architecture/adapter-profile-boundary.md)
- 权限与安全：[docs/architecture/security-permissions.md](docs/architecture/security-permissions.md)
- artifact / backup / restore：[docs/architecture/artifact-backup-restore-contract.md](docs/architecture/artifact-backup-restore-contract.md)
- release final gate：[docs/architecture/release-final-gate-contract.md](docs/architecture/release-final-gate-contract.md)
- completion audit：[docs/architecture/completion-audit-contract.md](docs/architecture/completion-audit-contract.md)
- 实现差距审计：[docs/development/implementation-gap-audit.md](docs/development/implementation-gap-audit.md)
- backlog 状态审计：[docs/development/task-backlog-status-audit.md](docs/development/task-backlog-status-audit.md)
- AreaMatrix 契约：[docs/dogfood/areamatrix-contract.md](docs/dogfood/areamatrix-contract.md)
- 迁移计划：[docs/migration/areamatrix-workflow-migration.md](docs/migration/areamatrix-workflow-migration.md)
- execution cutover 边界：[docs/migration/areamatrix-execution-cutover-boundary.md](docs/migration/areamatrix-execution-cutover-boundary.md)
- compatibility shim 计划：[docs/migration/areamatrix-compatibility-shim-plan.md](docs/migration/areamatrix-compatibility-shim-plan.md)
- cutover / rollback / compatibility：[docs/migration/cutover-rollback-compat.md](docs/migration/cutover-rollback-compat.md)

## 常用验证入口

```bash
go test ./...
go build ./cmd/areaflow
make smoke-docker-package-a-fingerprint-parity
make smoke-docker-shim-authorization-preflight
make smoke-docker-web-areamatrix-readonly
make smoke-docker-completion-audit-full-proof
make smoke-docker-completion-audit-release-candidate-snapshot
```

`make smoke-docker-package-a-fingerprint-parity` 使用临时 git fixture 和隔离 PostgreSQL，对比 Go status
projection authorization 与 Package A shell packet 的 protected-path fingerprint，并验证 `.areaflow/status.json`
目标文件变化不会污染非目标 protected path 指纹。

`make smoke-docker-shim-authorization-preflight` 是当前 AF-V04 AreaMatrix compatibility shim 真实落地前的
只读授权预检入口。它读取真实 AreaMatrix root，复验 status projection authorization、apply packet/gate、
shim authorization、shim apply packet/gate、rollback scope 和 no-write safety facts；它不授权、不写入
`.areaflow/status.json`、`workflow/README.md`、shim 文件或 execution/progress 路径。

`make smoke-docker-web-areamatrix-readonly` 是 Web dashboard 的真实 AreaMatrix 只读冒烟入口。它用隔离
PostgreSQL 读取真实 AreaMatrix root，验证 Web 只发 `/api/v1` GET 请求且不会绕到非 v1 `/api` 路由，并断言
status projection 面板展示精确 `required_authorization_phrase`，release / completion 面板仍显示
`real_100=blocked`；它不执行 status projection apply，也不写 AreaMatrix protected paths。

`make smoke-docker-completion-audit-full-proof` 使用隔离 PostgreSQL 数据库和临时 fixture project 证明
completion audit 能消费 E1-E9 机制证据，但 snapshot readiness 仍必须停在 fixture-only blocker，不能冒充真实
release candidate。

`make smoke-docker-completion-audit-release-candidate-snapshot` 是 fail-closed fixture identity / RC snapshot
负例机制验证：它证明 fixture 即使伪装出 release-candidate 路径形态也不能封存真实 RC snapshot。该 smoke
可配合 `AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` 校验真实 AreaMatrix status / workflow README 指纹不变；
它不是真实 AreaMatrix release candidate proof，也不证明外部 RC 证据文件存在或内容已完成真实审核。

## Web Dashboard

```bash
cd web
npm install
npm run dev
```

Vite dev server 默认监听 `127.0.0.1:5174`，并把 `/api` 代理到 `http://127.0.0.1:3847`。
如需指向其他 AreaFlow API，可设置 `AREAFLOW_API_URL`。旧拼写 `AREFLOW_API_URL` 暂时保留兼容。
