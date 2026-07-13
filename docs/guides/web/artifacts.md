# Artifacts 页面

Artifacts 展示平台保存的 artifact 索引。

当前功能：

- artifact type、storage backend、URI 和 source path。
- content type、size 和 SHA-256。
- workflow version 与 workflow item 关联。
- artifact 创建时间。

artifact 内容可以通过 API 按 ID 读取。完整性检查和 archive preview 属于 Operations 范围；archive preview 不删除、移动或上传 artifact bytes。
