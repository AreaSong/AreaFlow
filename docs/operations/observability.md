# Observability

AreaFlow 将 health、readiness、doctor 和 audit 分开表达：

- Health：AreaFlow 进程是否可响应，不检查外部依赖。
- Readiness：特定能力的前置条件是否满足。
- Doctor：配置、数据或边界是否存在可诊断问题。
- Audit：谁对什么资源做了何种安全决策。

`health=live` 不等于某项高风险操作可以执行，doctor pass 也不替代 permission、approval 或 Command API。

## 当前入口

- `GET /api/v1/health`
- `GET /api/v1/ready`：检查 PostgreSQL 是否可响应，失败时返回 HTTP 503。
- `GET /api/v1/service/status`
- `GET /api/v1/ops/readiness`
- `GET /api/v1/permissions/doctor`
- `GET /api/v1/audit/coverage`

日志和诊断输出必须避免 secret value。Event 用于领域变化，Audit Event 用于 actor、capability、decision 和 reason。

服务启动与优雅关闭会输出结构化生命周期日志，仅记录监听地址和关闭原因，不记录数据库连接串、请求 query、secret 或 artifact 内容。

远程 telemetry、托管 APM 和自动上传诊断不是当前默认能力。
