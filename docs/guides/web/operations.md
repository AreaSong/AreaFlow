# Operations 页面

Operations 面向平台维护者，展示服务、数据库、release 和受控操作状态。

当前功能：

- local service 与 operations readiness。
- migration ledger。
- metadata-only support bundle preview。
- release final gate 和 completion snapshot readiness。
- Web command boundary 及每个动作的 blocker。
- AreaMatrix compatibility、cutover、status projection 和 forwarding 专项诊断入口。

manifest、plan、preview、readiness 和 gate 只证明系统可以评估对应操作，不代表 restore、publish、rollout 或 destructive action 已经执行。
