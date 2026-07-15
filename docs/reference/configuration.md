# Project Configuration

AreaFlow 通过 `areaflow.yaml` 接入项目。示例见 [`../../examples/areamatrix/areaflow.yaml`](../../examples/areamatrix/areaflow.yaml)。

## 服务环境变量

- `AREAFLOW_DATABASE_URL`：PostgreSQL 连接串。
- `AREAFLOW_DB_MAX_CONNECTIONS`、`AREAFLOW_DB_MIN_CONNECTIONS`：连接池上限与预热下限；默认 `30`/`5`。
- `AREAFLOW_DB_CONNECT_TIMEOUT`、`AREAFLOW_DB_ACQUIRE_TIMEOUT`、`AREAFLOW_DB_QUERY_TIMEOUT`：连接、获取连接和查询超时；默认 `5s`、`5s`、`10s`。查询超时写入 PostgreSQL `statement_timeout`；普通 API 请求继承 acquire deadline，使连接池耗尽时 fail fast，SSE 长连接不使用该请求 deadline。
- `AREAFLOW_DB_MAX_CONNECTION_IDLE`、`AREAFLOW_DB_MAX_CONNECTION_LIFETIME`：连接最大空闲与生命周期；默认 `5m`、`30m`。
- `AREAFLOW_ENV`：`development` 或 `production`；默认 `development`。
- `AREAFLOW_HOST`：API 监听地址；development 默认只允许 loopback，production 在 OIDC 模式下可绑定远程地址。
- `AREAFLOW_PORT`：API 监听端口；默认 `3847`。
- `AREAFLOW_PUBLIC_BASE_URL`：production 对外 HTTPS 根地址，用于稳定生成 API URL。
- `AREAFLOW_TRUSTED_PROXY_CIDRS`：可信 LB/reverse proxy CIDR 列表；携带 forwarded headers 的请求必须来自该范围。
- `AREAFLOW_AUTH_MODE`：`disabled`、`token` 或 `oidc`；production 强制 `oidc`。
- `AREAFLOW_OIDC_ISSUER_URL`、`AREAFLOW_OIDC_CLIENT_ID`、`AREAFLOW_OIDC_CLIENT_SECRET_FILE`、`AREAFLOW_OIDC_REDIRECT_URL`：OIDC 客户端配置。
- `AREAFLOW_OIDC_GROUPS_CLAIM`、`AREAFLOW_OIDC_BOOTSTRAP_SUBJECTS`：groups claim 名称和首次 platform admin subject allowlist。
- `AREAFLOW_SESSION_SECRET_FILE`、`AREAFLOW_SESSION_COOKIE_NAME`、`AREAFLOW_SESSION_IDLE_TTL`、`AREAFLOW_SESSION_ABSOLUTE_TTL`：Web session 配置。
- `AREAFLOW_TOKEN_MAX_TTL`：service token 最大 TTL，不得超过 90 天。
- `AREAFLOW_ARTIFACT_BACKEND`：`local` 或 `s3`；production 强制 `s3`。
- `AREAFLOW_ARTIFACT_ROOT`：local backend 根目录。
- `AREAFLOW_S3_REGION`、`AREAFLOW_S3_BUCKET`、`AREAFLOW_S3_ENDPOINT`、`AREAFLOW_S3_USE_PATH_STYLE`：S3-compatible backend 配置。
- `AREAFLOW_METRICS_HOST`、`AREAFLOW_METRICS_PORT`：Prometheus listener。
- `OTEL_SERVICE_NAME`、`OTEL_EXPORTER_OTLP_ENDPOINT`：OpenTelemetry service 和 collector；production 强制 OTLP endpoint。
- `AREAFLOW_POSTGRES_CONTAINER`：使用 Docker PostgreSQL 时，backup/restore drill 选用同 major version 工具的容器名；默认 `areaflow-postgres`。

production 还要求 HTTPS public URL、可信代理、启用 TLS 的 PostgreSQL URL、S3 bucket 和 OTLP。应用自身监听 HTTP，由企业 TLS/LB 终止传输安全；任一必需项缺失都会 fail closed。

production 启动还会读取全部 active project；每个 project 的 `artifact_store.backend` 必须是 `s3`/`object`。存在 local 或缺失 backend 的项目时服务拒绝启动，不能用全局 S3 环境变量掩盖项目级配置漂移。

## 顶级结构

```yaml
version: 1
project: {}
ownership: {}
artifact_store: {}
permissions: {}
commands: {}
scheduling: {}
engines: {}
status_export: {}
migration: {}
```

## Project

定义项目 key、名称、root、kind、adapter、workflow profile 和默认分支。Root 是所有路径权限判断的项目边界。

## Ownership

声明产品文档、源码、workflow、execution 和 status summary 的源事实所有者。AreaFlow 不因项目被注册就自动取得 workflow 或 execution 所有权。

## Permissions

`capabilities` 控制读写、命令、worker、network、secret 和 agent execution。`read_paths`、`write_paths` 和 `forbidden_paths` 进一步限制文件范围。

`forbidden_paths` 和关闭的 capability 优先于允许规则。

## Commands

`allowed` 是项目允许命令的候选集合，`forbidden` 是强制拒绝集合。命令存在于 allowlist 不代表可以立即执行，仍需要对应 capability、gate、approval 和 audit。

## Scheduling 与 Engines

Scheduling 声明项目优先级、并行度、worker role 和 required capabilities。Engine profile 保存 provider、secret reference 和 resource limits。

Engine profile `enabled: false` 时不得调用 engine。Secret reference 只是引用，不得在配置外展开为明文日志或 metadata。

## Status Export

声明 `.areaflow/status.json` 等兼容投影。Export 必须经过 schema、preimage、path、authorization、apply packet 和 audit 检查。

## Migration

记录项目接入和所有权迁移策略。Migration 字段用于解释兼容关系，不应被当作自动执行 cutover 的开关。
