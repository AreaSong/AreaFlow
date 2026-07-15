# 安装 AreaFlow

## 环境要求

- Go 1.25 或更高版本。
- PostgreSQL 16 或兼容版本。
- Node.js 和 npm，用于 Web 控制台。
- Docker Compose，可选，用于启动仓库自带 PostgreSQL。

## 获取依赖

```bash
go mod download
cd web && npm install
```

## 启动 PostgreSQL

使用仓库配置：

```bash
docker compose up -d postgres
```

默认连接：

```text
postgres://areaflow:areaflow@127.0.0.1:54329/areaflow?sslmode=disable
```

也可以通过 `AREAFLOW_DATABASE_URL` 指向已有 PostgreSQL。

## 初始化数据库

```bash
export AREAFLOW_DATABASE_URL='postgres://areaflow:areaflow@127.0.0.1:54329/areaflow?sslmode=disable'
go run ./cmd/areaflow migrate up
go run ./cmd/areaflow migrate status
```

## 构建

```bash
go build ./cmd/areaflow
cd web && npm run build
```

## 配置项

| 环境变量 | 默认值 | 用途 |
|---|---|---|
| `AREAFLOW_DATABASE_URL` | 本地 `54329` PostgreSQL | 主状态数据库 |
| `AREAFLOW_PORT` | `3847` | API 监听端口 |
| `AREAFLOW_API_URL` | `http://127.0.0.1:3847` | Vite Web 代理目标 |

下一步继续 [Quickstart](quickstart.md)。
