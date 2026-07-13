# AreaFlow Desktop

This directory contains the v0.9 Tauri desktop shell scaffold.

The desktop shell is a local service observer and dashboard launcher. It must not
own workflow state, maintain a second database, write project files, run workflow
tasks directly, resolve secrets, or schedule workers outside the AreaFlow API.

## Contracts

- Source of truth:
  - `GET /api/v1/service/status`
  - `GET /api/v1/desktop/service-control-gate`
  - `GET /api/v1/desktop/notification-gate`
  - `GET /api/v1/desktop/tray-menu-gate`
- Default API base: `http://127.0.0.1:3847`.
- Default dashboard URL: returned by service status.
- Service control: `open_dashboard` may be an enabled link; start/stop/restart,
  notifications, and tray/menu remain disabled until their gate evidence exists.
- Notifications: event stream observation can be described as read-only, but OS
  notification permission and notification delivery remain disabled.
- Tray/menu: dashboard and read-only status actions can be described, but no
  native tray menu is created by this scaffold.
- Write boundary: all future state changes must go through AreaFlow Command API,
  permission gates, approval, and audit.

## Commands

```bash
npm install
npm run dev
npm run build
npm run tauri dev
```

The scaffold is intentionally read-only. Service start/stop, tray/menu,
notifications, and secret source UI are later v0.9 tasks.
