# Runs 页面

Runs 展示 workflow version 产生的执行会话。

当前功能：

- 从全局 Run 集合按项目读取 run timeline。
- 按 workflow version 查看和筛选 run。
- 按时间或状态排序 run。
- 查看 run type、kind、status、risk 和 dry-run 属性。
- 查看 run tasks 及其状态。
- 查看 attempts 及开始、结束状态。
- 查看与 run 关联的 artifact metadata。

后端 Run 集合支持 status、kind、type、dry-run 和 opaque cursor。Run Task 与 Attempt 还提供独立列表和详情 API。Web 会遍历 Run cursor 链后提供搜索、排序、版本过滤和分页；Task/Attempt 子资源尚无独立 cursor。

run start、drain 和 cancel 会更新 AreaFlow 状态并写入 event/audit；它们不自动代表通用 AI engine 或被管理项目命令已经执行。
