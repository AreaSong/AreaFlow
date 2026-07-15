# Workflows 页面

Workflows 按项目和 workflow version 展示生命周期状态。

当前功能：

- 从全局 workflow version 集合按项目读取版本。
- 选择 workflow version。
- 查看 stage board，并按 workflow sequence、标题或状态排序 workflow items。
- 查看 item 类型、标题和状态。
- 查看 approval records。
- 查看 ready transition preview，并在具备 `workflow.approval.record` capability 时批准或拒绝。
- 查看 residual work。

后端全局集合支持 lifecycle status、version kind、import mode 和 opaque cursor。Web 页面以项目为 scope，遍历 cursor 链后提供搜索、排序和分页；服务端过滤条件尚未全部暴露为页面控件。

Approval 对话框显示 preview 和 stage transition，要求填写长期可审计的原因。Web 每次提交生成 idempotency key；服务端从 token principal 取得 actor，并在成功后刷新 approval 与 audit 数据。

版本创建、gate 和 transition preview 写入仍只通过受控 CLI/API 使用，不在 Web 开放。
