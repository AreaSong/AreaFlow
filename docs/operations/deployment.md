# Deployment

AreaFlow 支持本机开发部署和单租户 production 部署。production 基线为两个无状态 API 副本、企业 TLS/LB、HA PostgreSQL、S3-compatible artifact store、OIDC 和外部 observability collector。

## 启动关系

```text
PostgreSQL
  -> areaflow migrate up
  -> areaflow server
  -> Web / CLI / Desktop
  -> optional worker
```

安装和首次启动见 [Installation](../getting-started/installation.md) 与 [Quickstart](../getting-started/quickstart.md)。

## 配置

- `AREAFLOW_DATABASE_URL` 指向 PostgreSQL。
- 开发用 Compose 将 PostgreSQL 端口仅绑定到 `127.0.0.1:54329`，不得直接暴露到远程网络。
- 项目通过 `areaflow.yaml` 注册。
- development local artifact root 应位于 AreaFlow-owned 路径，并按 project 隔离。
- production 使用 [`../../deploy/production/compose.yaml`](../../deploy/production/compose.yaml) 的双副本拓扑和 `areaflow.env.example` 配置清单；镜像必须使用签名不可变 digest。
- production 只在 OIDC、HTTPS public URL、可信代理、PostgreSQL TLS、S3 和 OTLP 全部有效时允许远程绑定。TLS 在企业 LB 终止，应用拒绝不可信 forwarded headers。

进程存活探针使用 `GET /api/v1/health`，接流量前的就绪探针使用 `GET /api/v1/ready`。就绪探针在 2 秒内分别检查 PostgreSQL、配置的 artifact store，并确认 OIDC provider 已初始化；任一 production 关键依赖不可用时返回 HTTP 503。响应的 `checks` 显式包含 `database`、`artifact_store` 和 `oidc`。

## 启动检查

1. 确认 PostgreSQL healthcheck 通过。
2. 使用目标版本执行 `areaflow migrate up`。
3. 启动 `areaflow server`，确认 `/api/v1/health` 返回 `ok`。
4. 确认 `/api/v1/ready` 返回 `ready` 后再启动或接入 Web/Desktop。

readiness 返回 `503` 时，先按 `checks` 检查 PostgreSQL、S3 bucket、OIDC 初始化、连接串、网络边界和 migration 状态，不应通过重启循环掩盖依赖故障。

## 变更

部署升级固定顺序为：备份与恢复演练、migration preflight/checksum、应用有序 migration、启动一个 canary 副本、验证 health/readiness/OIDC/权限/S3/metrics，再逐步替换剩余副本。当前不提供破坏性数据库 rollback。

服务进程接收 `SIGINT` 或 `SIGTERM` 后停止接收新请求，并在最多 5 秒内完成 HTTP 优雅关闭。部署器应先将 readiness 摘除，再发送终止信号。

升级失败时先从 LB 摘除 canary 并回退应用镜像或静态 Web。不得对 PostgreSQL 执行破坏性 migration rollback；若旧版本不能读取新 schema，则保持写面关闭，使用隔离验证过的备份恢复或经审批的前向修复 migration。

仓库内 `scripts/smoke-production-ha.sh` 通过单一健康代理地址验证两个 stateless 实例共享 PostgreSQL、单实例退出后客户端无需切换 URL 即继续服务；`scripts/smoke-production-capacity.sh` 在隔离 schema 验证 100 并发、50 RPS 持续、200 RPS 峰值、受控写延迟和大表分页；`scripts/smoke-upgrade-rollback.sh` 验证旧 schema 升级与旧二进制读取。真实 DNS、TLS/LB、HA PostgreSQL、PITR、跨故障域和生产流量观察属于企业外部依赖。
