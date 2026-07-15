# Artifacts

Artifact 表示 AreaFlow 管理的输入、输出或证据引用。PostgreSQL 保存 metadata、hash、size、URI、retention class 和资源关联；大内容保存在 artifact store 或由受控外部 URI 引用。

## 所有权

- `local_artifact` / `s3`：AreaFlow-owned artifact store 中的内容。
- `project_reference`：被管理项目拥有的文件引用，AreaFlow 默认只保存 metadata。
- `external_reference`：外部系统拥有的引用。

Artifact metadata 不转移项目文件的所有权。读取 `project_reference` 原文或复制到 AreaFlow store 需要显式能力。

## 完整性

AreaFlow 使用 SHA-256 和 size 校验 artifact 内容。local backend 强制路径边界；S3 backend 强制配置 bucket、SHA-256 checksum 和 AES256 server-side encryption，读取只接受该 bucket 的 `s3://` URI。

全局 Artifact 集合支持 project、type、storage backend、SHA-256、Run 和 Workflow Version 过滤，并使用 opaque cursor 分页。每条记录显式返回 `project_id`、`workflow_version_id`、`run_id` 和 `workflow_item_id` 中适用的关联。

## Retention

Retention class 用于解释保留和归档策略。Archive preview 只计算候选项、阻塞原因和策略事实，不执行 copy、upload、delete 或 GC。

S3 object storage 已作为生产 backend 开放。删除、GC、跨 bucket 归档和 retention apply 尚未开放；生命周期与版本恢复由企业 bucket policy 执行并作为外部依赖验证。

## Backend migration

Artifact backend 迁移按 `inventory -> copy -> readback verify -> activate -> observe` 执行。Copy 使用稳定的 project/artifact namespace 写入目标 backend，随后从目标 URI 回读 bytes 并重新校验 SHA-256 与 size；通过后在 `artifact_locations` 保存 `migration_candidate`，同时把原位置登记为不可删除的 `migration_source`。Activate 只允许选择已验证位置，要求 actor、reason 和未来的 observation deadline，并把主记录切换到目标 URI。

观察期读取先尝试主位置；主位置不可用或完整性校验失败时，依次读取已验证 candidate 和保留 source。到达 deadline 后必须显式完成 observation，状态转为 `stable`；稳定态不再回退 local source，S3 故障会直接失败并由 readiness 摘流。迁移命令不删除 local preimage，也不提供 GC。重复 copy 通过 location 唯一约束更新同一记录，重复 inventory 不产生写入。

```bash
areaflow artifact migration inventory <project> --source-backend local --target-backend s3 --json
areaflow artifact migration copy <project> <artifact-id> --target-backend s3 --target-root PREFIX --actor OPERATOR --reason TEXT
areaflow artifact migration activate <project> <artifact-id> --location-id ID --observe-until RFC3339 --actor OPERATOR --reason TEXT
areaflow artifact migration complete-observation <project> <artifact-id> --actor OPERATOR --reason TEXT
```

## 安全

Artifact metadata 和 support bundle 不得包含 secret value。受保护 evidence、audit export 和 release evidence 默认长期保留，不能因普通清理任务被删除。
