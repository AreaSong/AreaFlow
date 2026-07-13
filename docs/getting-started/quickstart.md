# AreaFlow Quickstart

本指南完成数据库初始化、项目注册、API 启动和 Web 查看。

## 1. 启动依赖

```bash
docker compose up -d postgres
export AREAFLOW_DATABASE_URL='postgres://areaflow:areaflow@127.0.0.1:54329/areaflow?sslmode=disable'
go run ./cmd/areaflow migrate up
```

## 2. 注册项目

仓库包含 AreaMatrix 示例配置：

```bash
go run ./cmd/areaflow project add --config examples/areamatrix/areaflow.yaml
go run ./cmd/areaflow project list
go run ./cmd/areaflow project summary areamatrix
```

接入其他项目时，应复制配置结构并修改 project root、adapter、profile、权限路径和 status export。项目默认只读，不应直接扩大 `write_paths` 或 capability。

## 3. 导入与检查

```bash
go run ./cmd/areaflow project import areamatrix
go run ./cmd/areaflow project doctor areamatrix
go run ./cmd/areaflow project readiness areamatrix
```

这些命令将项目元数据写入 AreaFlow PostgreSQL。除非配置和授权明确允许，否则不会修改被管理项目源码或 execution 路径。

## 4. 启动 API

```bash
go run ./cmd/areaflow server
```

检查服务：

```bash
curl http://127.0.0.1:3847/api/v1/health
curl http://127.0.0.1:3847/api/v1/projects
```

## 5. 启动 Web

```bash
cd web
npm run dev
```

打开 `http://127.0.0.1:5174`，从左侧项目上下文选择项目，并通过 Overview、Projects、Workflows、Runs、Workers、Artifacts、Audit 和 Operations 页面查看平台状态。
