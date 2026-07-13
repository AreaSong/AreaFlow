# Observability

AreaFlow 将 health、readiness、doctor 和 audit 分开表达：

- Health：service 和依赖是否可响应。
- Readiness：特定能力的前置条件是否满足。
- Doctor：配置、数据或边界是否存在可诊断问题。
- Audit：谁对什么资源做了何种安全决策。

`health=live` 不等于某项高风险操作可以执行，doctor pass 也不替代 permission、approval 或 Command API。

## 当前入口

- `GET /health`
- `GET /service/status`
- `GET /api/v1/ops/readiness`
- `GET /api/v1/permissions/doctor`
- `GET /api/v1/audit/coverage`

日志和诊断输出必须避免 secret value。Event 用于领域变化，Audit Event 用于 actor、capability、decision 和 reason。

远程 telemetry、托管 APM 和自动上传诊断不是当前默认能力。
