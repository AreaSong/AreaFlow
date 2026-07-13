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
- 项目通过 `areaflow.yaml` 注册。
- artifact root 应位于 AreaFlow-owned 路径，并按 project 隔离。
- API 默认绑定本机地址；暴露到远程网络前必须补充认证与传输安全。

## 变更

部署升级先应用有序 migration，再启动新 service。当前不提供托管自动升级、远程 process supervision 或破坏性数据库 rollback。
