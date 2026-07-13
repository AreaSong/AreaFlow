# Runs 页面

Runs 展示 workflow version 产生的执行会话。

当前功能：

- 按 workflow version 查看 run timeline。
- 查看 run type、kind、status、risk 和 dry-run 属性。
- 查看 run tasks 及其状态。
- 查看 attempts 及开始、结束状态。
- 查看与 run 关联的 artifact metadata。

run start、drain 和 cancel 会更新 AreaFlow 状态并写入 event/audit；它们不自动代表通用 AI engine 或被管理项目命令已经执行。
