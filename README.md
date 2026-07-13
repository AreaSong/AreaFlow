# AreaFlow

AreaFlow 是面向 AI 软件开发的项目管理与执行治理平台。它统一管理项目接入、workflow 生命周期、任务运行、worker 调度、artifact、审批、事件和审计记录，并通过 CLI、REST API、Web 和 Desktop 提供一致的控制面。

AreaMatrix 是 AreaFlow 的第一个被管理项目。被管理项目默认保持只读；任何项目文件写入都必须由项目配置、路径规则、capability、审批和审计共同授权。

## 核心能力

- 多项目注册、配置快照、导入、状态摘要和 readiness 检查。
- workflow profile、版本、stage、item、gate、transition preview 和 approval。
- run、run task、attempt、执行计划和受控运行状态。
- worker 注册、heartbeat、lease、worker pool 和调度预览。
- artifact metadata、hash、内容引用、完整性检查和归档预览。
- domain events、audit events、权限检查和写入证据。
- PostgreSQL 主状态、append-only 事件与本地 artifact store。
- Web 多页面控制台和 Tauri Desktop 本地服务观察入口。

## Web 控制台

Web 控制台按产品资源划分为八个一级页面：

| 页面 | 功能 |
|---|---|
| Overview | 项目健康、执行容量、阻塞项和最近事件 |
| Projects | 项目注册信息、配置身份、inventory 和 readiness |
| Workflows | workflow 版本、stage、item、approval 和 residual |
| Runs | run、task、attempt 和执行证据 |
| Workers | worker、heartbeat、capability、pool 和调度状态 |
| Artifacts | artifact 来源、类型、大小、hash 和关联信息 |
| Audit | 安全决策、授权记录和 domain event timeline |
| Operations | 服务、迁移、release gate、support metadata 和受控操作 |

## 快速开始

环境要求：Go 1.23+、Node.js/npm、Docker 或可访问的 PostgreSQL 16+。

```bash
docker compose up -d postgres

export AREAFLOW_DATABASE_URL='postgres://areaflow:areaflow@127.0.0.1:54329/areaflow?sslmode=disable'

go run ./cmd/areaflow migrate up
go run ./cmd/areaflow project add --config examples/areamatrix/areaflow.yaml
go run ./cmd/areaflow server
```

另一个终端启动 Web：

```bash
cd web
npm install
npm run dev
```

打开 `http://127.0.0.1:5174`。API 默认监听 `http://127.0.0.1:3847`。

详细步骤见 [安装指南](docs/getting-started/installation.md) 和 [Quickstart](docs/getting-started/quickstart.md)。

## 产品边界

- PostgreSQL 是平台主状态源；文件用于项目配置、artifact 原文和审计导出。
- AreaFlow 不拥有被管理项目的产品文档和源码语义。
- Web 当前只开放读操作；已经存在的写入 API 仍受 Command API、approval 和 audit 边界约束。
- run control 和 worker `run-once` 主要管理 AreaFlow 状态或 dry-run 任务，不等同于通用 AI engine 执行。
- users、teams、API tokens、webhooks、secret resolve、remote workers 和第三方 plugin execution 尚未作为当前产品能力开放。
- backup、restore、publish 和 rollout 当前提供 manifest、plan、readiness 或 gate，不应解释为已经执行外部副作用。

## 文档

- [文档总览](docs/README.md)
- [产品模型](docs/concepts/product-model.md)
- [架构总览](docs/architecture/overview.md)
- [项目配置](./docs/history/v1.0/contracts/project-config.md)
- [Web 页面指南](docs/guides/web/README.md)
- [开发环境](docs/development/setup.md)
- [路线图](docs/roadmap.md)
- [未来设计 proposals](proposals/README.md)
- [治理边界](governance/README.md)

## 开发与验证

```bash
make check
```

该命令运行 Go 格式化、后端测试、后端构建和 Web TypeScript/Vite 构建。涉及 Web 交互的变更还应运行对应 browser smoke 和响应式页面验证。

## License

仓库尚未声明开源许可证。公开分发前必须补充明确的 `LICENSE`。
