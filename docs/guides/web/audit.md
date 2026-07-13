# Audit 页面

Audit 将安全决策和领域事件分开呈现。

## Audit events

记录 action、capability、resource、decision、reason 和时间，用于回答谁对什么资源做出了什么安全判断。

## Domain events

记录 project、workflow、run、worker 和 artifact 生命周期中已经发生的事实。

Audit 页面当前按项目读取最近记录。服务端的多条件过滤、cursor pagination 和导出能力仍需通过正式 API 契约补齐。
