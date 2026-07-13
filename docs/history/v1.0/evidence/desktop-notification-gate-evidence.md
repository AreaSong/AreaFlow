# Desktop Notification Gate Evidence

## Scope

本证据对应 `AF-V09-004 Desktop Notification Gate`。目标是在 Desktop 真正请求 OS notification permission
或建立 SSE notification bridge 前，提供只读门禁矩阵，说明系统通知、approval 通知、run failure 通知和
worker recovery 通知需要的 event filter、redaction、dedupe、rate limit、approval 和 audit。

## Implemented Surface

```text
GET /api/v1/desktop/notification-gate
areaflow desktop notification-gate
areaflow desktop notification-gate --json
```

Desktop shell consumes this endpoint together with:

```text
GET /api/v1/service/status
GET /api/v1/desktop/service-control-gate
```

当前 gate 覆盖：

- `observe_event_stream`
- `enable_system_notifications`
- `approval_needed_notifications`
- `run_failure_notifications`
- `worker_recovery_notifications`

只有 `observe_event_stream` 返回：

```text
status = ready
default_ui_state = available_read_only
```

其余 notification action 当前都返回 disabled / blocked。

## Safety Facts

该 gate 是 Query API，只读且无副作用。响应必须保持：

```text
db_write_attempted=false
project_write_attempted=false
event_stream_opened=false
notification_requested=false
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

- 从 gate 接口直接打开 SSE 连接。
- 从 gate 接口请求 OS notification permission。
- 未定义 filter / redaction / dedupe / rate limit 时订阅通知。
- 发送远程通知。
- 在通知点击中调度 worker 或执行 workflow。
- 把 notification state 维护成第二状态源。

## Verification

Focused checks:

```bash
go test ./internal/app -run 'Test(Help|DesktopNotificationGateToJSON)'
go test ./internal/project -run 'TestBuildDesktopNotificationGateKeepsNotificationsDisabled'
go test ./internal/api -run 'TestDesktopNotificationGateEndpoint'
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

`scripts/smoke-local.sh` now calls `areaflow desktop notification-gate --json` inside the v0.1-v1.0 long smoke.
The smoke asserts:

```text
status=blocked
mode=read_only_desktop_notification_gate
observe_event_stream default_ui_state=available_read_only
enable_system_notifications / approval_needed_notifications / run_failure_notifications are present but blocked
notification_permission_flow_not_implemented
notification_redaction_contract_not_defined
system_notifications_not_open
event_stream_opened=false
notification_requested=false
command_created=false
approval_created=false
audit_event_written=false
worker_scheduled=false
workflow_execution_started=false
project_write_attempted=false
secrets_resolved=false
```

This proves the CLI surface exposes the same read-only gate as the API/Desktop shell and does not open SSE
connections, OS notification permission, notification delivery or workflow actions.

Full baseline before milestone closeout:

```bash
go test ./...
go build ./cmd/areaflow
cd web && npm run build
cd desktop && npm run build
cargo check --manifest-path desktop/src-tauri/Cargo.toml
git diff --check -- .
```
