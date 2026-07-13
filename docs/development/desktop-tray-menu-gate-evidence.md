# Desktop Tray Menu Gate Evidence

## Scope

本证据对应 `AF-V09-005 Desktop Tray Menu Gate`。目标是在 Desktop 真正创建原生 menu bar / tray
前，提供只读门禁矩阵，说明 dashboard、status、recent events、service control、notification 和 settings
菜单项需要的 API、permission、preflight、approval、audit 和 OS integration contract。

## Implemented Surface

```text
GET /api/v1/desktop/tray-menu-gate
areaflow desktop tray-menu-gate
areaflow desktop tray-menu-gate --json
```

Desktop shell consumes this endpoint together with:

```text
GET /api/v1/service/status
GET /api/v1/desktop/service-control-gate
GET /api/v1/desktop/notification-gate
```

当前 gate 覆盖：

- `open_dashboard`
- `show_service_status`
- `show_recent_events`
- `start_service`
- `stop_service`
- `enable_notifications`
- `open_settings`

以下 action 返回 ready / read-only：

```text
open_dashboard
show_service_status
show_recent_events
```

其余 service control、notification 和 settings action 当前都返回 disabled / blocked。

## Safety Facts

该 gate 是 Query API，只读且无副作用。响应必须保持：

```text
db_write_attempted=false
project_write_attempted=false
tray_menu_created=false
os_integration_requested=false
command_created=false
approval_created=false
audit_event_written=false
service_control_attempted=false
notification_requested=false
worker_scheduled=false
workflow_execution_started=false
secrets_resolved=false
network_used=false
```

## Guardrails

Desktop 不得：

- 从 gate 接口创建原生 tray/menu。
- 从 gate 接口请求 OS integration。
- 从 tray action 直接启动/停止 service。
- 从 tray action 请求系统通知权限。
- 打开包含 secret 明文的 settings。
- 从 tray action 调度 worker 或执行 workflow。
- 把 tray/menu state 维护成第二状态源。

## Verification

Focused checks:

```bash
go test ./internal/app -run 'Test(Help|DesktopTrayMenuGateToJSON)'
go test ./internal/project -run 'TestBuildDesktopTrayMenuGateKeepsControlActionsDisabled'
go test ./internal/api -run 'TestDesktopTrayMenuGateEndpoint'
bash -n scripts/smoke-local.sh
make smoke-docker-v1-stable-fixture
```

Latest result on 2026-07-04:

```text
internal/app focused tests PASS
internal/project focused tests PASS
internal/api focused tests PASS
bash syntax check PASS
v1 stable fixture smoke PASS using areaflow_smoke_20260704142302_79088
```

`scripts/smoke-local.sh` now calls `areaflow desktop tray-menu-gate --json` inside the v0.1-v1.0 long smoke.
The smoke asserts:

```text
status=blocked
mode=read_only_desktop_tray_menu_gate
open_dashboard default_ui_state=enabled_link
show_service_status / show_recent_events are present as read-only actions
start_service / enable_notifications are present but blocked
service_control_gate_blocked
tray_service_control_not_open
notification_gate_blocked
tray_notification_action_not_open
tray_menu_created=false
os_integration_requested=false
service_control_attempted=false
notification_requested=false
command_created=false
approval_created=false
audit_event_written=false
worker_scheduled=false
workflow_execution_started=false
project_write_attempted=false
secrets_resolved=false
```

This proves the CLI surface exposes the same read-only gate as the API/Desktop shell and does not create native
tray/menu integration, service control, notification permission or workflow actions.

Full baseline before milestone closeout:

```bash
go test ./...
go build ./cmd/areaflow
cd web && npm run build
cd desktop && npm run build
cargo check --manifest-path desktop/src-tauri/Cargo.toml
git diff --check -- .
```

Latest full baseline result on 2026-07-02 19:14 CST:

```text
go test ./... PASS
go build ./cmd/areaflow PASS
web npm run build PASS
desktop npm run build PASS
cargo check --manifest-path desktop/src-tauri/Cargo.toml PASS
node --check scripts/smoke-web-check.mjs PASS
git diff --check -- . PASS
```

Generated build outputs remain ignored:

```text
desktop/dist
desktop/node_modules
desktop/src-tauri/target
web/dist
web/node_modules
```
