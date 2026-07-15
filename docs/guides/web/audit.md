# Audit 页面

Audit 将安全决策和领域事件分开呈现。

## Audit events

记录 actor ID、action、capability、resource、decision、reason 和时间，用于回答谁对什么资源做出了什么安全判断。页面可按 actor ID、action、resource、decision 和时间范围组合过滤。

## Domain events

记录 project、workflow、run、worker 和 artifact 生命周期中已经发生的事实。

Audit 页面当前按项目读取记录。服务端支持组合过滤和 opaque cursor；Web 使用服务端过滤并遍历 cursor 链，再提供客户端搜索、排序和分页。审计导出仍需独立契约。
