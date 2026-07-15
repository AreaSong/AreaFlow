# AreaFlow 开发环境

## 环境

- Go 1.25+
- Node.js 与 npm
- PostgreSQL 16+
- Docker Compose，可选

安装和数据库初始化见 [安装指南](../getting-started/installation.md)。

## 后端

```bash
docker compose up -d postgres
export AREAFLOW_DATABASE_URL='postgres://areaflow:areaflow@127.0.0.1:54329/areaflow?sslmode=disable'
go run ./cmd/areaflow migrate up
go run ./cmd/areaflow server
```

API 默认监听 `127.0.0.1:3847`。

## Web

```bash
cd web
npm install
npm run dev
```

Vite 默认监听 `127.0.0.1:5174`，并将 `/api` 代理到 `AREAFLOW_API_URL`，默认是 `http://127.0.0.1:3847`。

## Desktop

```bash
cd desktop
npm install
npm run dev
npm run tauri dev
```

Desktop 是本地服务观察和 Dashboard 启动入口，不维护第二套 workflow 状态。

## 验证

```bash
make check
```

等价核心检查：

```bash
go fmt ./...
go test ./...
go build ./cmd/areaflow
cd web && npm run build
cd desktop && npm run build
```

Web 交互变更还需要使用真实 API 数据验证所有一级路由、控制台错误、桌面与移动视口。

服务生命周期变更运行 `make smoke-docker-graceful-shutdown`，验证 PostgreSQL readiness、`SIGTERM`、退出码和关闭日志。

## 代码边界

- `cmd/areaflow/`：CLI 入口。
- `internal/api/`：HTTP 和 SSE 控制面。
- `internal/project/`：项目、workflow、execution、worker、artifact 和 audit 领域逻辑。
- `internal/adapter/`：项目 adapter。
- `internal/workflow/`：profile 加载与验证。
- `migrations/`：PostgreSQL schema。
- `web/`：React Web 控制台。
- `desktop/`：Tauri Desktop。
- `workflow/`：内置 profiles 和 templates。

开发变更必须保持项目默认只读，不得使用测试或 smoke 绕过真实 permission boundary。

合并前的生产化聚合门禁为 `make production-smoke`，容量门禁为 `make load-check`。两者要求可用的本地 PostgreSQL；S3 smoke 会启动临时 MinIO，所有数据库规模测试使用隔离 schema。
