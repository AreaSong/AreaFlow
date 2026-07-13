# Milestones

AreaFlow milestone 文档定义每个版本的边界。版本路线从 v0.1 到 v1.0 逐步推进，不允许前置阶段偷偷扩展到后续能力。
从 0% 到 100% 的完整阶段 backlog 见 [`../product/phase-backlog.md`](../plans/phase-backlog.md)。
v0.1 Import + Status Mirror 最小闭环和读写边界见
[`../architecture/v0.1-import-mirror-contract.md`](../contracts/v0.1-import-mirror-contract.md)。
v0.2 Shadow Doctor + Drift Check 的只读验收和 native doctor 授权边界见
[`../architecture/v0.2-shadow-doctor-contract.md`](../contracts/v0.2-shadow-doctor-contract.md)。
v0.3 New Version Authoring 的 version create、stage skeleton、gate、transition preview 和 approval record
边界见 [`../architecture/v0.3-version-authoring-contract.md`](../contracts/v0.3-version-authoring-contract.md)。
v0.4 Workflow Ownership Cutover 的 compatibility、shim readiness、cutover readiness、DB-only authoring cutover
apply 和 rollback 边界见
[`../architecture/v0.4-workflow-ownership-cutover-contract.md`](../contracts/v0.4-workflow-ownership-cutover-contract.md)。
v0.5 Runner Preview 的 dry-run execution model、runner preview report、run control 和 no-execution
边界见 [`../architecture/v0.5-runner-preview-contract.md`](../contracts/v0.5-runner-preview-contract.md)。
v0.6 Worker Beta 的 worker lifecycle、lease、dry-run run-once、scoped execution evidence 和 no-cutover
边界见 [`../architecture/v0.6-worker-beta-contract.md`](../contracts/v0.6-worker-beta-contract.md)。
v0.7 Web Dashboard 的 `/api/v1`、SSE、read-only panels、write action gate 和 no-second-state 边界见
[`../architecture/v0.7-web-dashboard-contract.md`](../contracts/v0.7-web-dashboard-contract.md)。
v0.8 Multi-project Worker Pool 的 worker pool summary、schedule preview、project isolation 和 no-scheduler
边界见
[`../architecture/v0.8-multi-project-worker-pool-contract.md`](../contracts/v0.8-multi-project-worker-pool-contract.md)。
v0.9 Desktop Shell 的 local service status、dashboard launcher、desktop gates 和 no-second-state 边界见
[`../architecture/v0.9-desktop-shell-contract.md`](../contracts/v0.9-desktop-shell-contract.md)。
v1.0 Stable Platform 的 100% 完成条件、release/completion/ops/dogfood/protected path 总边界见
[`../architecture/v1.0-stable-platform-contract.md`](../contracts/v1.0-stable-platform-contract.md)。
v1.0 release final gate、exception 和 publish preview 的统一语义见
[`../architecture/release-final-gate-contract.md`](../contracts/release-final-gate-contract.md)。
0-100% completion audit 和最终完成证明边界见
[`../architecture/completion-audit-contract.md`](../contracts/completion-audit-contract.md)。
Plugin / marketplace seed 和未知 plugin execution 的边界见
[`../architecture/plugin-marketplace-boundary.md`](../../../../proposals/plugin-marketplace.md)。
Object artifact store、archive copy/upload、retention-aware GC 和 delete apply 的边界见
[`../architecture/object-artifact-retention-contract.md`](../contracts/object-artifact-retention-contract.md)。
Team Console、远程控制台和多用户控制面的边界见
[`../architecture/team-remote-control-boundary.md`](../../../../proposals/team-and-remote-control.md)。
Budget、quota、rate limit 和 usage metering 的边界见
[`../architecture/budget-quota-boundary.md`](../../../../proposals/budget-and-quota.md)。
External integrations、webhooks、third-party callbacks 和多 API 接入的边界见
[`../architecture/integration-webhook-boundary.md`](../../../../proposals/integrations-and-webhooks.md)。
Operations、deployment、observability、support bundle、telemetry、upgrade 和 rollback 的边界见
[`../architecture/operations-deployment-observability-boundary.md`](../contracts/operations-deployment-observability-boundary.md)。

## Gate Summary

```text
v0.1 Import + Status Mirror
v0.2 Shadow Doctor + Drift Check
v0.3 New Version Authoring
v0.4 Workflow Ownership Cutover
v0.5 Runner Preview
v0.6 Worker Execution Beta
v0.7 Web Dashboard
v0.8 Multi-project / Multi-worker
v0.9 Desktop Shell
v1.0 Stable Platform
```

每个 milestone 至少说明：

- Goal：本阶段交付什么。
- Includes：允许进入本阶段的能力。
- Excludes / Constraint：明确不做什么。
- Success：进入下一阶段前必须证明什么。

## Go / No-Go 总控表

| 阶段 | 进入下一阶段前必须证明 |
|---|---|
| v0.1 Import + Status Mirror | 最小闭环符合 v0.1 import / mirror 合同；AreaMatrix 可注册；metadata import 可重复；`.areaflow/status.json` 可生成；不写 `workflow/versions/**`、不执行任务。 |
| v0.2 Shadow Doctor + Drift Check | 合同符合 v0.2 shadow doctor 边界；doctor、summary、readiness、import-diff、verify-bundle 字段稳定；native doctor 未授权时只记录 warn/skipped，不越权执行命令；phase gate 的 warn/blocked 不被压成 pass。 |
| v0.3 New Version Authoring | 合同符合 v0.3 version authoring 边界；authored workflow version、stage skeleton、gate、transition preview、approval record 可审计；approval 不等于 execution；promotion preview 不 apply。 |
| v0.4 Workflow Ownership Cutover | 合同符合 v0.4 workflow ownership cutover 边界；compatibility contract、approval gate、live mapping gate、cutover readiness gate 和 DB-only cutover apply 可证明；cutover 只切 authoring，不替代 task-loop、不切 execution。 |
| v0.5 Runner Preview | 合同符合 v0.5 runner preview 边界；run、run_task、run_attempt、artifact、event、audit_event dry-run 证据完整；risk / permission preflight 可阻断；run control 只控制 dry-run DB 状态，不启动 worker、不执行命令、不写项目。 |
| v0.6 Worker Execution Beta | 合同符合 v0.6 worker beta 边界；worker register、heartbeat、lease、run-once、evidence、capability preflight 可审计；fixture/read-only/artifact-only/rollback drill 证据不能累计成真实 AreaMatrix execution cutover。 |
| v0.7 Web Dashboard | 合同符合 v0.7 Web dashboard 边界；Web 只通过 `/api/v1` GET 和 SSE 展示 project、version、stage、run、artifact、residual、approval、worker、audit、cutover/readiness 和 release preview；write action gate 只显示 disabled/read-only 动作，不维护第二套状态。 |
| v0.8 Multi-project / Multi-worker | worker pool summary 和 schedule preview 可跨项目稳定展示；`project_key` isolation fixture、priority、agent role、resource readiness 可证明；真实 scheduler、remote worker、secret resolve、team/auth enforcement 和多项目 execution apply 仍关闭。 |
| v0.9 Desktop Shell | Desktop 能观察 local service、打开 Web、显示健康和 gate；service control、OS notification、native tray/menu、secret resolve 和远程 Team Console 仍关闭。 |
| v1.0 Stable Platform | 符合 v1.0 stable platform 合同；backup/restore dry-run、安全审计、可诊断运维、release final gate、adapter/profile/plugin 边界稳定；completion audit 证明 AreaMatrix dogfood 完成 Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover -> Archive -> Shim Retirement。 |

## 通用验证

每个 milestone 关闭前至少运行：

```bash
go test ./...
go build ./cmd/areaflow
git diff --check -- .
```

涉及 Web 时追加：

```bash
cd web && npm run build
```

阶段 smoke：

```text
v0.1: project add / import / status-projection-apply / export-status
v0.2: doctor / summary / readiness / import-diff / verify-bundle
v0.3: workflow version create / stages / gate / transition / approval
v0.4: compatibility / cutover-readiness / cutover_readiness_gate
v0.5: run preview / run start-drain-cancel
v0.6: worker register / heartbeat / lease / run-once
v0.7: Web build + API-backed page smoke
v0.8: read-only worker pool summary / schedule-preview
v0.9: desktop service health + desktop gate smoke
v1.0: install / migrate / service status / support bundle preview + backup / restore + release final gate + completion audit smoke
```

## 放行规则

- 没有 gate evidence，不进入下一阶段。
- `warn` 必须有解释、责任归属、后续处理路径和审计记录；未知缺口不能作为 `warn` 放行。
- Query API 不得产生写入、执行、secret 读取、worker 调度或状态推进。
- Command API 必须经过 idempotency、permission、gate 和 audit。
- 任何写项目、执行命令、调用 AI、解析 secret 或管理 git 的能力，都必须具备显式 approval、allowlist 和 audit。
- v1.0 发布不得只依赖测试通过；必须有 release readiness、acceptance gate、exception apply preview 和 final gate 证据。
- Release final gate `pass` 不代表真实 package、tag、sign、push、upload 或 publish。
- Release final gate `pass` 也不代表 100%；最终完成必须通过 completion audit 聚合 phase/task、
  AreaMatrix dogfood、release、ops、isolation 和 protected path evidence。
- `project_reference` / `external_project` artifact 不能被当作完整可恢复原文，必须进入 restore/release 的
  `needs_attention` 或显式 exception 路径。
- `object` backend 在 verifier 通过前不能被当作完整可恢复内容；archive copy/upload、GC/delete 和 orphan
  cleanup 不属于 v1.0。
- v1.0 plugin 边界只覆盖 built-in / seed catalog、manifest draft 和 conformance，不覆盖未知第三方
  plugin install / enable / execution。
- v1.0 operations 边界只覆盖 local bootstrap、readiness、doctor、service status、metadata-only support
  bundle preview 和 local-only telemetry；远程运维控制、托管升级、破坏性 rollback 和完整 support export
  属于 v1.x。
