# Deployment

AreaFlow 当前支持本机部署：Go service、PostgreSQL、local artifact store、Web 静态资源和可选 Desktop shell。

## 启动关系

```text
PostgreSQL
  -> areaflow migrate up
  -> areaflow serve
  -> Web / CLI / Desktop
  -> optional worker
```

安装和首次启动见 [Installation](../getting-started/installation.md) 与 [Quickstart](../getting-started/quickstart.md)。

## 配置

- `AREAFLOW_DATABASE_URL` 指向 PostgreSQL。
- 开发用 Compose 将 PostgreSQL 端口仅绑定到 `127.0.0.1:54329`，不得直接暴露到远程网络。
- 项目通过 `areaflow.yaml` 注册。
- artifact root 应位于 AreaFlow-owned 路径，并按 project 隔离。
- API 只允许绑定 `localhost`、`127.0.0.1` 或 `::1`。认证与传输安全尚未开放，因此配置非 loopback 地址会拒绝启动。

进程存活探针使用 `GET /api/v1/health`，接流量前的就绪探针使用 `GET /api/v1/ready`。就绪探针在 2 秒内检查 PostgreSQL，并在依赖不可用时返回 HTTP 503。

## 启动检查

1. 确认 PostgreSQL healthcheck 通过。
2. 使用目标版本执行 `areaflow migrate up`。
3. 启动 `areaflow server`，确认 `/api/v1/health` 返回 `ok`。
4. 确认 `/api/v1/ready` 返回 `ready` 后再启动或接入 Web/Desktop。

readiness 返回 `503` 时，先检查 PostgreSQL 进程、连接串、网络边界和 migration 状态，不应通过重启循环掩盖依赖故障。

## 变更

部署升级先应用有序 migration，再启动新 service。当前不提供托管自动升级、远程 process supervision 或破坏性数据库 rollback。

服务进程接收 `SIGINT` 或 `SIGTERM` 后停止接收新请求，并在最多 5 秒内完成 HTTP 优雅关闭。部署器应先将 readiness 摘除，再发送终止信号。

升级失败时可以回退 AreaFlow 二进制或静态 Web 版本，但不得对 PostgreSQL 执行破坏性 migration rollback。若新 migration 已应用，回退版本必须仍能读取当前 schema；否则保持服务停止并前向修复。
