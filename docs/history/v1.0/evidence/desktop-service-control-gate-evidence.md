# Desktop Service Control Gate Evidence

## Scope

本证据对应 `AF-V09-003 Desktop Service Control Gate`。目标是在 Desktop 真正具备 start / stop /
restart local service 能力前，提供只读门禁矩阵，说明每个 service control 需要的 capability、preflight、
approval、audit 和 recovery contract。

## Implemented Surface

```text
GET /api/v1/desktop/service-control-gate
areaflow desktop service-control-gate
areaflow desktop service-control-gate --json
```

Desktop shell consumes this endpoint together with:

```text
GET /api/v1/service/status
```

当前 gate 覆盖：

- `open_dashboard`
- `start_service`
- `stop_service`
- `restart_service`
- `enable_notifications`
- `tray_menu`

只有 `open_dashboard` 返回：

```text
status = ready
default_ui_state = enabled_link
```

其余 service control、notification 和 tray/menu action 当前都返回 disabled / blocked。

## Safety Facts

该 gate 是 Query API，只读且无副作用。响应必须保持：

```text
db_write_attempted=false
project_write_attempted=false
process_control_attempted=false
command_created=false
approval_created=false
audit_event_written=false
worker_scheduled=false
workflow_execution_started=false
secrets_resolved=false
network_used=false
```

## Guardrails

Desktop 不得：

- 直接启动、停止或重启 AreaFlow service。
- 绕过 AreaFlow API。
- 调度 worker。
- 直接执行 workflow。
- 写项目文件。
- 维护第二数据库。
- 解析 secret 明文。

## Verification

Focused checks:

```bash
go test ./internal/app -run 'Test(Help|ServiceStatusFlags|LocalServiceStatusToJSON|DesktopServiceControlGateToJSON)'
go test ./internal/project -run 'TestBuildDesktopServiceControlGateKeepsControlsDisabled'
bash -n scripts/smoke-local.sh
make smoke-docker-v1-stable-fixture
```

Latest result on 2026-07-04:

```text
internal/app focused tests PASS
internal/project focused tests PASS
bash syntax check PASS
v1 stable fixture smoke PASS using areaflow_smoke_20260704142302_79088
```

`scripts/smoke-local.sh` now calls `areaflow desktop service-control-gate --json` inside the v0.1-v1.0 long smoke.
The smoke asserts:

```text
status=blocked
mode=read_only_desktop_service_control_gate
open_dashboard default_ui_state=enabled_link
start_service / stop_service / restart_service are present but blocked
desktop_service_control_not_open
process_supervision_contract_not_defined
service_stop_requires_drain_policy
process_control_attempted=false
command_created=false
approval_created=false
audit_event_written=false
worker_scheduled=false
workflow_execution_started=false
project_write_attempted=false
secrets_resolved=false
```

This proves the CLI surface exposes the same read-only gate as the API/Desktop shell and does not open real process
control.

Full baseline before milestone closeout:

```bash
go test ./...
go build ./cmd/areaflow
cd web && npm run build
cd desktop && npm run build
cargo check --manifest-path desktop/src-tauri/Cargo.toml
git diff --check -- .
```
