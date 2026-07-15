# Project Configuration

AreaFlow 通过 `areaflow.yaml` 接入项目。示例见 [`../../examples/areamatrix/areaflow.yaml`](../../examples/areamatrix/areaflow.yaml)。

## 服务环境变量

- `AREAFLOW_DATABASE_URL`：PostgreSQL 连接串。
- `AREAFLOW_HOST`：API 监听地址，只接受 `localhost` 或 loopback IP；默认 `127.0.0.1`。
- `AREAFLOW_PORT`：API 监听端口；默认 `3847`。
- `AREAFLOW_AUTH_MODE`：`disabled` 或 `token`；默认 `disabled`。设为 `token` 时，除 health、ready 和 auth status 外，API 都要求 Bearer token。
- `AREAFLOW_POSTGRES_CONTAINER`：使用 Docker PostgreSQL 时，backup/restore drill 选用同 major version 工具的容器名；默认 `areaflow-postgres`。

Token 认证只用于本机 loopback 控制面；远程监听和 TLS 尚未开放，非 loopback 监听会拒绝启动。

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
