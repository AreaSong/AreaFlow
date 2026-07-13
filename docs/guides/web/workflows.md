# Workflows 页面

Workflows 按项目和 workflow version 展示生命周期状态。

当前功能：

- 选择 workflow version。
- 查看 stage board 和每个 stage 的 workflow items。
- 查看 item 类型、标题和状态。
- 查看 approval records。
- 查看 residual work。

版本创建、gate、transition preview 和 approval 的写入能力已经存在于 CLI/API，但 Web 是否可操作必须服从统一 command、confirmation、idempotency 和 audit 契约。
