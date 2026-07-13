# Worker Scheduling

AreaFlow 使用 PostgreSQL 中的 Worker、Run Task 和 Lease 协调执行。`project_key` 是调度、artifact 和 audit 的隔离边界。

## 不变量

- Worker 领取 Run Task，不直接领取 Workflow Item。
- 一个 Run Task 同时最多有一个 active Lease。
- capability denial 不创建 Lease、Attempt 或 Artifact。
- Lease 过期进入 `needs_recovery`，由 recovery 流程判断后续状态。
- 项目并发、worker 并发和 task type 必须同时满足。

## 候选判断

调度候选至少考虑 Project、task status、required capability、worker kind、engine/secret readiness、project concurrency 和 worker slot。

Schedule Preview 使用与真实调度相同的候选和 slot 语义，但只返回 recommended/blocked 等解释，不创建 Lease，也不启动 Worker。

## Worker 生命周期

Worker 注册稳定 `worker_key`，通过 heartbeat 更新可用性。Lease acquire/release/recover 由受控 Command API 执行并记录 event/audit。

全局 Worker 集合支持 project、status、worker type 和 capability 过滤，并使用 opaque cursor 分页。Worker 详情按数值 ID 返回 registry 记录及最近 heartbeat/lease 历史；当前两类历史共享同一个 limit，尚无独立 cursor。

Remote worker credential、team permission enforcement 和自动 scheduler 尚未作为当前能力开放，不能从 worker pool summary 或 schedule preview 推导其可用。
