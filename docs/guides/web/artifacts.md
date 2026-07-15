# Artifacts 页面

Artifacts 展示平台保存的 artifact 索引。

当前功能：

- 从全局 Artifact 集合按项目读取索引。
- artifact type、storage backend、URI 和 source path。
- content type、size 和 SHA-256。
- workflow version 与 workflow item 关联。
- artifact 创建时间。
- 按创建时间、类型或大小排序，并通过稳定详情 URL 直接加载 artifact。
- 对 AreaFlow-owned、文本类型且不超过 512 KiB 的 local artifact 按需预览内容；读取仍经过 project scope、size 和 SHA-256 完整性校验。

后端 Artifact 集合支持 type、storage backend、SHA-256、Run、Workflow Version 和 opaque cursor。记录显式包含 project/run/workflow 关联；Web 会遍历 cursor 链后按项目搜索、排序和分页，不再静默截断首批记录。

artifact 内容可以通过 API 按 ID 读取。完整性检查和 archive preview 属于 Operations 范围；archive preview 不删除、移动或上传 artifact bytes。
