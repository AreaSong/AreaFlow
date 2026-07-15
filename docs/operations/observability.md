# Observability

AreaFlow 将 health、readiness、doctor 和 audit 分开表达：

- Health：AreaFlow 进程是否可响应，不检查外部依赖。
- Readiness：特定能力的前置条件是否满足。
- Doctor：配置、数据或边界是否存在可诊断问题。
- Audit：谁对什么资源做了何种安全决策。

`health=live` 不等于某项高风险操作可以执行，doctor pass 也不替代 permission、approval 或 Command API。

## 当前入口

- `GET /api/v1/health`
- `GET /api/v1/ready`：检查 PostgreSQL、artifact store 和 OIDC 初始化状态，关键依赖失败时返回 HTTP 503。
- `GET /api/v1/service/status`
- `GET /api/v1/ops/readiness`
- `GET /api/v1/permissions/doctor`
- `GET /api/v1/audit/coverage`
- Prometheus：独立 metrics listener 暴露 HTTP count/latency/inflight、`areaflow_command_requests_total`、`areaflow_oidc_requests_total`、`areaflow_sse_connections`、`areaflow_dependency_ready`、DB/S3 operation count/latency、PostgreSQL pool acquired/idle/total/max 以及 `areaflow_audit_writes_total`。Audit 指标由 API pool 的 pgx tracer 对成功 `audit_events` insert 计数。
- OpenTelemetry：production 启动时初始化 OTLP trace exporter。
- JSON log：请求和生命周期日志关联 `X-Request-ID`，响应同时返回该 ID。

日志和诊断输出必须避免 secret value。Event 用于领域变化，Audit Event 用于 actor、capability、decision 和 reason。

服务启动与优雅关闭会输出结构化 JSON 生命周期日志，仅记录监听地址、请求 ID 和关闭原因，不记录数据库连接串、Authorization、Cookie、请求 query、secret 或 artifact 内容。

仓库提供 [`../../deploy/production/prometheus-rules.yaml`](../../deploy/production/prometheus-rules.yaml) 的 availability burn、read/write p95、依赖不可用、PostgreSQL 连接池饱和和依赖操作错误率规则。依赖错误率只在对应依赖持续有流量时触发，避免空闲期或单次低流量失败造成误报。企业 Prometheus、OTLP、日志平台、告警路由和值班渠道必须外部配置并回读验证。
