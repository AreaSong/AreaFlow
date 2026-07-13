# Desktop Shell Scaffold Evidence

## Scope

本证据对应 `AF-V09-002 Tauri Shell Scaffold`。目标是在 AreaFlow 仓库内创建 v0.9 desktop shell
最小骨架，让桌面入口只观察本机 AreaFlow service status 和 Web dashboard，不维护第二状态源。

## Implemented Surface

```text
desktop/
  package.json
  vite.config.ts
  src/main.ts
  src/styles.css
  src-tauri/
    Cargo.toml
    tauri.conf.json
    capabilities/default.json
    src/lib.rs
    src/main.rs
```

桌面前端只请求只读 Query API：

```text
GET {api_base}/api/v1/service/status
GET {api_base}/api/v1/desktop/service-control-gate
GET {api_base}/api/v1/desktop/notification-gate
GET {api_base}/api/v1/desktop/tray-menu-gate
GET {api_base}/api/v1/ops/readiness
GET {api_base}/api/v1/projects/{project}/shim-authorization when the selected project exists
GET {api_base}/api/v1/projects/{project}/shim-apply-packet when the selected project exists
GET {api_base}/api/v1/projects/{project}/shim-apply-gate when the selected project exists
GET {api_base}/api/v1/projects/{project}/execution-cutover-readiness when the selected project exists
GET {api_base}/api/v1/projects/{project}/execution-forwarding-v1-apply-packet when the selected project exists
GET {api_base}/api/v1/release/final-gate
GET {api_base}/api/v1/release/evidence-bundle
GET {api_base}/api/v1/release/package-preview
GET {api_base}/api/v1/release/distribution-preview
GET {api_base}/api/v1/release/publish-gate
GET {api_base}/api/v1/release/publish-approval-preview
GET {api_base}/api/v1/release/rollout-plan-preview
```

默认 `api_base` 为：

```text
http://127.0.0.1:3847
```

可用 `?api=http://host:port` 覆盖，并存入 local storage。默认 project 为 `areamatrix`，可用
`?project=<project_key>` 覆盖，并存入 local storage。

## Guardrails

Desktop scaffold 当前保持：

```text
no workflow execution
no project file write
no second database
no direct worker scheduling
no secret parsing
no Command API write surface
no service process control
no AreaMatrix shim editing
no shim apply command
no execution cutover apply
no execution forwarding apply
no execution forwarding rollback
no task-loop run forwarding
no release package creation
no release publish
no release archive/sign/tag/upload
no release approval or rollout creation
no support bundle export
no telemetry upload
no migration apply
no managed upgrade or rollback
```

`src-tauri/capabilities/default.json` 当前不授予任何 JS-side Tauri command permission。`bundle.active=false`
用于避免 v0.9 scaffold 阶段提前进入发布打包；正式 icon、package、signing 和 distribution 仍属于后续
release/package 任务。

## Verification

Latest result on 2026-07-02:

```bash
node -e "JSON.parse(...)"
cargo fmt --manifest-path desktop/src-tauri/Cargo.toml -- --check
npm install --package-lock-only --ignore-scripts
npm install --ignore-scripts
npm run build
cargo check --manifest-path desktop/src-tauri/Cargo.toml
```

Result:

```text
desktop JSON configs PASS
cargo fmt check PASS
npm dependency audit PASS, 0 vulnerabilities
desktop frontend build PASS
desktop Tauri cargo check PASS
```

Latest focused frontend check on 2026-07-02:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

Latest focused frontend check on 2026-07-02 21:32 CST:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

The shell now renders selected-project `Execution Cutover` readiness from
`GET /api/v1/projects/{project}/execution-cutover-readiness` when present. A missing project still returns
`null` for this panel and does not break the service status shell.

Latest focused frontend check on 2026-07-02 21:38 CST:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

The shell now also renders global `Release Final Gate` status from `GET /api/v1/release/final-gate`.
This only displays the read-only final gate, its items and forbidden actions such as
`create_release_package` and `apply_release`.

Latest focused frontend check on 2026-07-02 21:45 CST:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

The shell now also renders global release evidence bundle, package preview and publish gate state from
the read-only release preview endpoints. This only displays items and forbidden actions such as
`compress_artifacts`, `create_git_tag`, `publish_release` and `apply_release`.

Latest focused frontend check on 2026-07-02 21:54 CST:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

The shell now also renders distribution preview, publish approval preview and rollout plan preview from
the read-only release rollout endpoints. This only displays items and forbidden actions such as
`approve_release`, `create_rollout`, `publish_release` and `apply_release`.

Latest focused frontend check on 2026-07-03 CST:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

The shell now also renders `Operations Readiness` from `GET /api/v1/ops/readiness`. This only displays service,
metadata-only support bundle, migration ledger, local-only telemetry, managed ops deferral and safety facts. It
does not export support bundles, apply migrations, upload telemetry, control service processes, write database
rows or write managed project files.

Latest focused frontend check on 2026-07-04 CST:

```bash
npm --prefix desktop run build
```

Result:

```text
desktop frontend build PASS
```

The shell now also renders selected-project `Shim Apply Review` from
`GET /api/v1/projects/{project}/shim-apply-packet` when present. This only displays the packet command type,
gate status, proof facts, blocked items and safety facts such as `project_write=false`,
`status_projection=false` and `area_matrix_files=false`. It does not call a shim apply command, create a command
request, write `.areaflow/status.json`, edit AreaMatrix shim files, forward `./task-loop run` or open execution
cutover apply.

Latest focused Desktop check on 2026-07-06 21:55 CST:

```bash
npm --prefix desktop run build
cargo check --manifest-path desktop/src-tauri/Cargo.toml
```

Result:

```text
desktop frontend build PASS
desktop Tauri cargo check PASS
```

The shell now independently renders selected-project `Shim Apply Gate` from
`GET /api/v1/projects/{project}/shim-apply-gate` when present, in addition to the nested gate visible inside
`shim-apply-packet`. This only displays the gate decision, packet field requirements, capabilities, blocked items
and safety facts such as `project_write=false`, `status_projection=false` and `area_matrix_files=false`. It does
not call `POST /api/v1/projects/{project}/shim-apply`, create a command request, write `.areaflow/status.json`,
edit AreaMatrix shim files, forward `./task-loop run` or open execution cutover apply.

Latest focused Desktop check on 2026-07-07 18:17 CST:

```bash
npm run build --prefix desktop
```

Result:

```text
desktop frontend build PASS
```

The shell now also renders selected-project `Forwarding v1 Packet Gate` from
`GET /api/v1/projects/{project}/execution-forwarding-v1-apply-packet` when present. This only displays the
readiness snapshot hash, canonical legacy / rollback / protected-path proof refs, gate status and blocked items.
It does not call `POST /api/v1/projects/{project}/execution-forwarding-v1-apply`, create command requests, create
runs or tasks, write AreaMatrix files, forward `./task-loop run`, or open rollback/apply actions.

Latest Desktop gate CLI parity check on 2026-07-04 CST:

```bash
go test ./internal/app -run 'Test(Help|Desktop(ServiceControl|Notification|TrayMenu)GateToJSON)'
make smoke-docker-v1-stable-fixture
```

Result:

```text
desktop gate CLI focused tests PASS
v1 stable fixture smoke PASS using areaflow_smoke_20260704142302_79088
```

The long smoke now calls:

```text
areaflow desktop service-control-gate --json
areaflow desktop notification-gate --json
areaflow desktop tray-menu-gate --json
```

Those CLI surfaces expose the same read-only gate state that Desktop renders from the API. They do not create
commands, control service processes, request OS notifications, create native tray/menu integration, schedule workers,
write audit events, write project files or resolve secrets.

## Remaining v0.9 Work

- Local service start / stop / restart manager.
- Tray / menu.
- System notifications from API/SSE.
- Service manager smoke with a real local API process.
- Real package icon and distribution packaging.
