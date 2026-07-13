# AreaFlow Desktop

AreaFlow Desktop 是 Tauri 本地服务观察和 Web 控制台启动入口。它复用 AreaFlow API，不维护第二套 workflow 状态。

## 当前功能

- 读取 `GET /api/v1/service/status`。
- 展示 API、数据库、worker pool 和 Dashboard 状态。
- 打开 Web 控制台。
- 展示 service control、notification 和 tray/menu gate。

## 边界

- Desktop 不直接写项目文件。
- Desktop 不直接运行 workflow task 或调度 worker。
- Desktop 不解析 secret。
- start、stop、restart、系统通知和原生 tray/menu 在相应 gate 开放前保持关闭。
- 所有未来状态变更必须经过 AreaFlow API、permission、approval 和 audit。

## 开发

```bash
npm install
npm run dev
npm run build
npm run tauri dev
```

默认 API base 是 `http://127.0.0.1:3847`，Dashboard URL 由 service status 返回。
