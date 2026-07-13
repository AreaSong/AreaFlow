# Execution Model

AreaFlow 的执行模型由 Run、Run Task、Attempt、Worker 和 Lease 组成。所有执行状态写入 PostgreSQL，外部副作用必须经过独立授权。

## 资源

- Run：一次执行会话，绑定 Project 和可选 Workflow Version。
- Run Task：worker 可领取的最小调度单元。
- Attempt：对 Run Task 的一次尝试，保留结果和证据。
- Worker：声明 kind、capability、并发和 heartbeat 的执行者。
- Lease：Worker 与 Run Task 的限时绑定。

## 执行链

```text
Run
  -> Run Task queued
  -> approval gate
  -> Worker acquires Lease
  -> Attempt
  -> Artifact / Event / Audit Event
  -> Run Task terminal state
```

Lease 过期进入 recovery 判断，不直接等价于 task failed。旧 Attempt 不覆盖；重试创建新的 Attempt。

## 当前执行类型

当前实现区分 dry-run、read-only verify、AreaFlow-owned artifact write、fixture project write 和 managed generated write。每种类型分别报告 project read/write、execution write、engine call、command、secret 和 network 的安全事实。

这些 scoped execution 不能合并解释为任意命令执行或任意 AI engine 调用。真实项目写入仍要求 project config、capability、path allowlist、gate、approval、idempotency 和 audit 全部满足。

## Run Control

Start、drain 和 cancel 更新 AreaFlow 的控制状态并记录 event/audit。它们不隐式运行 shell、调用 engine 或修改被管理项目。

## 查询模型

全局 Run 集合支持 project、status、kind、type 和 dry-run 过滤，并使用 opaque cursor 分页。Run 详情下的 Task 与 Attempt 提供列表和按 ID 详情；子资源响应保留 `project_id`、`workflow_version_id`、`run_id` 以及可选 `run_task_id` 关联。

Task/Attempt 子资源当前一次返回该 Run 的完整列表，不提供独立 cursor。

未来扩大 engine、secret、remote worker 或项目写入能力时，必须通过对应 proposal 和安全评审。
