# Workflows 页面

Workflows 按项目和 workflow version 展示生命周期状态。

当前功能：

- 从全局 workflow version 集合按项目读取版本。
- 选择 workflow version。
- 查看 stage board，并按 workflow sequence、标题或状态排序 workflow items。
- 查看 item 类型、标题和状态。
- 查看 approval records。
- 查看 residual work。

后端全局集合支持 lifecycle status、version kind、import mode 和 opaque cursor。当前 Web 页面以项目为 scope，并在已加载数据上提供搜索、排序和分页；服务端过滤条件尚未全部暴露为页面控件。

版本创建、gate、transition preview 和 approval 的写入能力已经存在于 CLI/API，但 Web 是否可操作必须服从统一 command、confirmation、idempotency 和 audit 契约。
