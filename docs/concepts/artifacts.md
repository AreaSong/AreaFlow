# Artifacts

Artifact 表示 AreaFlow 管理的输入、输出或证据引用。PostgreSQL 保存 metadata、hash、size、URI、retention class 和资源关联；大内容保存在 artifact store 或由受控外部 URI 引用。

## 所有权

- `local_artifact`：AreaFlow-owned artifact store 中的内容。
- `project_reference`：被管理项目拥有的文件引用，AreaFlow 默认只保存 metadata。
- `external_reference`：外部系统拥有的引用。

Artifact metadata 不转移项目文件的所有权。读取 `project_reference` 原文或复制到 AreaFlow store 需要显式能力。

## 完整性

AreaFlow 使用 hash 和 size 校验本地 artifact 内容。内容 API 只应返回 AreaFlow-owned 且通过路径、hash、size 校验的对象。

全局 Artifact 集合支持 project、type、storage backend、SHA-256、Run 和 Workflow Version 过滤，并使用 opaque cursor 分页。每条记录显式返回 `project_id`、`workflow_version_id`、`run_id` 和 `workflow_item_id` 中适用的关联。

## Retention

Retention class 用于解释保留和归档策略。Archive preview 只计算候选项、阻塞原因和策略事实，不执行 copy、upload、delete 或 GC。

删除、GC、外部 object storage 和 retention apply 尚未作为当前能力开放，相关设计必须进入 proposal。

## 安全

Artifact metadata 和 support bundle 不得包含 secret value。受保护 evidence、audit export 和 release evidence 默认长期保留，不能因普通清理任务被删除。
