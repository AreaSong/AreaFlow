# Workers 页面

Workers 展示执行容量和调度状态。

当前功能：

- project worker registry。
- worker type、hostname、PID、capability 和 heartbeat。
- worker pool 总量、online 数、active lease 和 queued task。
- 各项目 schedule preview、available slots 和 blocked reason。

worker `run-once` 当前只领取符合受限执行契约的任务。通用 remote worker 和任意 engine execution 不属于当前开放能力。
