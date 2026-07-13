# Web Write Action Gate Evidence

## Scope

本证据对应 `AF-V07-002 Web Write Action Gate`。目标是在 Web 写操作真正打开前，提供只读矩阵，
说明每类写动作需要的 Command API、risk preview、permission preflight、approval、audit 和 evidence。

## Implemented Surface

```text
GET /api/v1/web/write-action-gate
```

Web dashboard consumes this endpoint as a read-only panel:

```text
Web Action Gate
Disabled Writes
```

The same dashboard smoke also consumes the project shim authorization query as a read-only panel:

```text
GET /api/v1/projects/{project}/shim-authorization
Shim Authorization
AreaMatrix Shim
```

The dashboard also consumes shim apply packet/gate queries as read-only review panels:

```text
GET /api/v1/projects/{project}/shim-apply-packet
GET /api/v1/projects/{project}/shim-apply-gate
Shim Apply Review
Packet Gate
```

The dashboard also consumes execution cutover readiness as a read-only observation panel:

```text
GET /api/v1/projects/{project}/execution-cutover-readiness
Execution Cutover
AreaMatrix Readiness
```

The dashboard also consumes Execution Forwarding v1 apply-packet as a read-only review panel:

```text
GET /api/v1/projects/{project}/execution-forwarding-v1-apply-packet
Forwarding v1
Packet Gate
```

The dashboard also consumes the v1.0 release final gate as a read-only observation panel:

```text
GET /api/v1/release/final-gate
Release Final Gate
Release Readiness
```

The dashboard also consumes the v1.0 release preview chain as read-only observation panels:

```text
GET /api/v1/release/evidence-bundle
GET /api/v1/release/package-preview
GET /api/v1/release/distribution-preview
GET /api/v1/release/publish-gate
GET /api/v1/release/publish-approval-preview
GET /api/v1/release/rollout-plan-preview
Release Evidence
Release Package
Release Distribution
Release Publish
Release Approval
Release Rollout
```

当前 gate 覆盖：

- `approval_record`
- `run_drain`
- `run_cancel`
- `artifact_archive_preview`
- `status_projection_apply`
- `generated_write_apply_beta`

所有 action 当前都返回：

```text
status = blocked
default_ui_state = disabled
```

## Safety Facts

该 gate 是 Query API，只读且无副作用。响应必须保持：

```text
db_write_attempted=false
project_write_attempted=false
artifact_write_attempted=false
execution_write_attempted=false
command_created=false
approval_created=false
audit_event_written=false
worker_scheduled=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
```

## Guardrails

Web 不得：

- 默认启用写按钮。
- 直接写数据库、artifact store 或 project files。
- 绕过 AreaFlow API。
- 绕过 permission preflight 或 approval gate。
- 把 SSE 当作状态源。
- 从 Web 直接调度 worker。

## Verification

Focused checks:

```bash
go test ./internal/project ./internal/api
```

Latest result on 2026-07-02:

```text
ok  	github.com/areasong/areaflow/internal/project
ok  	github.com/areasong/areaflow/internal/api
```

Full baseline before milestone closeout:

```bash
go test ./...
go build ./cmd/areaflow
cd web && npm run build
git diff --check -- .
```

Latest Go baseline on 2026-07-02:

```text
go test ./... PASS
go build ./cmd/areaflow PASS
```

Latest Web build on 2026-07-02:

```text
cd web && npm run build PASS
```

Latest Web smoke on 2026-07-02 21:20 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
```

Temporary database: `af_web_shim_auth_20260702212028_62899`; cleanup residual database count was 0.

The first 20:19 run exposed that the dashboard did not render the schedule preview engine blocker
`engine_profile_disabled`. The Web panel now renders engine blockers in Schedule Preview, and the
20:21 rerun passed. The 21:20 run additionally proves the dashboard requests and renders
`GET /api/v1/projects/{project}/shim-authorization` as a blocked read-only panel. The Web smoke checker
expects both read-only gates and still fails on any non-GET `/api/v1` request.

Latest Web smoke on 2026-07-02 21:32 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
```

Temporary database prefix: `af_web_execution_cutover_20260702213221_*`; cleanup residual database count was 0.
The smoke checker now also expects
`GET /api/v1/projects/{project}/execution-cutover-readiness`, visible `Execution Cutover`,
`AreaMatrix Readiness`, `explicit_execution_cutover_approval`, `execution_cutover_apply=false` and
`task_loop_run_forwarded=false`, while still failing on any non-GET `/api/v1` request.

Latest Web smoke on 2026-07-02 21:38 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_web_release_final_20260702213826_27166?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
```

Temporary database cleanup residual database count was 0. The smoke checker now also expects
`GET /api/v1/release/final-gate`, visible `Release Final Gate`, `Release Readiness`,
`read_only_release_final_gate`, `final_gate:release_readiness`, `create_release_package` and
`apply_release`, while still failing on any non-GET `/api/v1` request.

Latest Web smoke on 2026-07-02 21:45 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_web_release_preview_20260702214511_49410?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
```

Temporary database cleanup residual database count was 0. The smoke checker now also expects
`GET /api/v1/release/evidence-bundle`, `GET /api/v1/release/package-preview` and
`GET /api/v1/release/publish-gate`, visible `Release Evidence`, `Evidence Bundle`,
`Release Package`, `Package Preview`, `Release Publish`, `Publish Gate`,
`read_only_release_evidence_bundle`, `read_only_release_package_preview`,
`read_only_release_publish_gate`, `evidence:release_final_gate`,
`package:evidence:release_final_gate`, `publish_gate:distribution_preview`,
`compress_artifacts`, `create_git_tag` and `publish_release`, while still failing on any non-GET
`/api/v1` request.

Latest Web smoke on 2026-07-02 21:54 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_web_release_rollout_20260702215419_81144?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
```

Temporary database cleanup residual database count was 0. The smoke checker now also expects
`GET /api/v1/release/distribution-preview`, `GET /api/v1/release/publish-approval-preview` and
`GET /api/v1/release/rollout-plan-preview`, visible `Release Distribution`, `Distribution Preview`,
`Release Approval`, `Publish Approval`, `Release Rollout`, `Rollout Plan`,
`read_only_release_distribution_preview`, `read_only_release_publish_approval_preview`,
`read_only_release_rollout_plan_preview`, `distribution:git_release`,
`publish_approval:publish_gate`, `rollout_plan:publish_approval`, `approve_release` and
`create_rollout`, while still failing on any non-GET `/api/v1` request.

Latest Web smoke on 2026-07-04 13:41 CST:

```bash
AREAFLOW_SMOKE_SCRIPT=scripts/smoke-web.sh bash scripts/smoke-docker.sh
```

Result:

```text
smoke-web: ok
```

Temporary database: `areaflow_smoke_20260704134145_44355`. The smoke checker now also expects
`GET /api/v1/projects/{project}/shim-apply-packet`,
`GET /api/v1/projects/{project}/shim-apply-gate`, visible `Shim Apply Review`, `Packet Gate`,
`project.shim.apply`, `shim_readiness_still_blocked` and `area_matrix_files=false`, while still failing on any
non-GET `/api/v1` request. This proves the Web dashboard only observes the shim apply review packet/gate; it does
not create command requests, write status projections, edit AreaMatrix files, forward task-loop or open a shim apply
button.

Latest real AreaMatrix Web read-only smoke on 2026-07-07 13:34 CST:

```bash
make smoke-docker-web-areamatrix-readonly
```

Result:

```text
smoke-web: ok real-areamatrix
```

Temporary database: `areaflow_smoke_20260707133419_49685`; the isolated database was dropped after the run.
The smoke checker now fails on any non-v1 `/api` request, continues to fail on any non-GET `/api/v1` request,
and verifies the real AreaMatrix dashboard renders the Package A authorization phrase plus `real_100=blocked`
release / completion guardrails. The real read-only script also compares browser-observation side-effect counts
across project/config/permission/workflow/residual/status snapshot/command/event/audit/gate/status projection/
approval/run/task/attempt/artifact/worker/heartbeat/lease/security integration tables and confirms both the
AreaMatrix target fingerprints and the recursive non-target protected path content fingerprint remain unchanged.

Latest Web smoke on 2026-07-07 18:17 CST:

```bash
make smoke-docker-web
```

Result:

```text
smoke-web: ok
```

Temporary database: `areaflow_smoke_20260707181722_27138`; project
`areamatrix-web-fixture-20260707181722`; workflow `web-smoke-20260707181722-ready`. The smoke checker now also
expects `GET /api/v1/projects/{project}/execution-forwarding-v1-apply-packet`, visible `Packet Gate`,
`legacy_ref=` and `fingerprint_ref=`, while still failing on non-v1 `/api` requests and any non-GET `/api/v1`
request. This proves the Web dashboard only observes the Execution Forwarding v1 apply packet / gate review; it
does not expose an apply button, create command requests, create runs or tasks, write AreaMatrix files, or forward
`./task-loop run`.

Latest real AreaMatrix Web read-only smoke on 2026-07-10 11:54 CST:

```bash
make smoke-docker-web-areamatrix-readonly
```

Result:

```text
smoke-web: ok real-areamatrix
```

Temporary database: `areaflow_smoke_20260710115415_8114`. The browser check now also verifies the Status Projection
authorization and Package A Gate panels render `scope=package_a_status_projection_preflight_only`,
`not_real_100=true`, `eligible_is_not_apply=true` and `separate_apply=true`, while still failing on non-v1 `/api`
requests and any non-GET `/api/v1` request. Release / completion panels now also render `claim_scope`,
`not_real_100=true`, `evidence_only=true`, `status_alone_is_not_completion=true` and release-candidate decision text.
This proves the Web surface makes `ready` / `pass` / `complete` / `apply_command_eligible` wording visibly non-apply
and non-real-100.
