# SLO And Incident Response

AreaFlow v1 的首个生产基线面向单租户组织内部控制面。

## SLO

- 月度用户可见 API 可用性不低于 99.9%，所有用户可见停机均计入错误预算。
- 普通读取请求 p95 不高于 500 ms；受控写入请求 p95 不高于 1 秒。
- RTO 不超过 4 小时，RPO 不超过 1 小时。
- 验收容量为 100 个并发会话、50 RPS 持续、200 RPS 峰值、100 万 event/audit 和 10 万 artifact metadata。

指标由 `areaflow_http_requests_total`、`areaflow_http_request_duration_seconds` 和依赖健康检查提供。production 必须把 metrics、OTLP trace 和结构化日志接入企业平台；仓库规则不能替代真实采集。

## Incident Levels

| Level | Meaning | Required Response |
|---|---|---|
| Sev1 | 越权、数据破坏、控制面全面不可用 | 立即关闭写面、通知安全与运营 owner、启动恢复与披露评估 |
| Sev2 | 关键项目不可用、持续 SLO 快速消耗 | 停止 rollout、回退应用或隔离故障副本、建立事故时间线 |
| Sev3 | 部分功能退化或单一依赖异常 | 工单跟踪、限时修复、验证监控覆盖 |
| Sev4 | 低影响缺陷或文档问题 | 正常迭代处理 |

事故关闭必须包含影响、时间线、止损、恢复、根因、促成因素、owner、期限和验证证据。复盘结论回写代码、测试、架构或 Runbook，不能只保留在聊天记录。
