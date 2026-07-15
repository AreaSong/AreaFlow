# Workers 页面

Workers 展示执行容量和调度状态。

当前功能：

- 从全局 Worker 集合按项目读取 worker registry。
- worker type、hostname、PID、capability 和 heartbeat。
- 最近 heartbeat 和 lease 历史。
- worker pool 总量、online 数、active lease 和 queued task。
- 各项目 schedule preview、available slots 和 blocked reason。

后端 Worker 集合支持 worker key、status、worker type、capability 和 opaque cursor。Web registry 会遍历 cursor 链后支持搜索、排序和客户端分页；Worker 详情按稳定 URL 加载最近 heartbeat 与 lease 历史。两类历史当前共享 history limit，尚无独立 cursor。

worker `run-once` 当前只领取符合受限执行契约的任务。通用 remote worker 和任意 engine execution 不属于当前开放能力。
