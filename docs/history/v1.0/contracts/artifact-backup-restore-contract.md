# Artifact / Backup / Restore Contract

## 定位

本文定义 AreaFlow 的 artifact 存储、完整性检查、backup manifest、restore dry-run 和 archive/retention
边界。它补充 [`data-model-v1.md`](data-model-v1.md)、[`api.md`](../../../reference/api.md) 和
[`commands-and-approvals.md`](../../../concepts/commands-and-approvals.md)。Object backend、archive copy/upload、
retention-aware GC 和 delete apply 的长期边界见
[`object-artifact-retention-contract.md`](object-artifact-retention-contract.md)。

v1.0 的目标是证明可恢复范围可解释、证据链可审计、不可恢复缺口不会被伪装成绿色。本文不是 restore
apply 授权；真实 restore apply 属于 v1.x R4 migration/security 能力。

## 核心不变量

- PostgreSQL 保存 artifact metadata、hash、URI、size、type、retention class 和对象关系。
- Prompt、日志、报告、diff、verify evidence、failure summary 等大内容不直接写入 PostgreSQL。
- AreaFlow-owned 原文进入 local artifact store；后续可扩展 object storage。
- 历史 AreaMatrix execution / prompt / log 原文第一阶段只作为 `project_reference` 或 `external_project`
  metadata，不复制、不删除、不当作完整可恢复原文。
- `project_reference` / `external_project` 不会因为被索引或产生 hash 就变成 AreaFlow-owned content。
- `object` backend 在 object verifier 落地前只能返回 `skipped` / `needs_attention`，不能计入完整可恢复内容。
- `project_reference` / `external_project` 只能让 restore plan 返回 `needs_attention`，不能让完整恢复链路
  返回 `ready`。
- Artifact integrity、backup manifest、restore dry-run 和 archive preview 都是只读或 preview；不能执行
  restore、delete、GC、move、copy、secret resolve 或 project write。
- Release final gate 必须诚实暴露 metadata-only history、local hash mismatch、missing blob 和 skipped
  object verifier。

## Artifact 分类

AreaFlow artifact 按 ownership 分三类：

```text
project_reference:
  历史项目内原文。AreaFlow 只保存 path/hash/type/size/metadata，不拥有内容。

local:
  AreaFlow-owned local artifact store 内容。AreaFlow 可读取 hash/size，用于 evidence 和 restore planning。

object:
  未来对象存储内容。v1.0 前只能作为 metadata / skipped verifier，除非 object verifier 单独落地。
```

按 retention class 分：

```text
ephemeral:
  可重建 preview / 临时报告。只能由未来显式 GC command 清理。

run_evidence:
  runner / worker / doctor / verify / repair evidence。至少保留到 run closeout 和 release gate。

audit:
  approval、permission、command、write、security evidence。默认长期保留。

release:
  release readiness / package / distribution / publish preview evidence。默认长期保留。

external_ref:
  原文留在被管理项目或外部系统。AreaFlow 不删除、不修复，只标记限制。

legal_hold:
  法务、合规或人工 hold。任何 archive / GC / delete apply 都必须 blocked。
```

Retention class 决定 archive / GC / backup 行为。普通 archive preview 不能删除 `audit`、`release`、
受保护的 `run_evidence`、`external_ref` 或 `legal_hold`。未知 retention class 必须返回 `needs_policy`，
不能自动 archive、copy、upload 或 delete。

## Local Artifact Store

默认 local artifact store 路径语义：

```text
~/.areaflow/artifacts/{project_key}/{scope}/{category}/{artifact_id}-{sha256-prefix}{extension}
```

路径中的 `project_key` 是稳定 namespace；不能依赖数据库内部 ID 作为唯一语义。Artifact 写入必须：

```text
write bytes
compute sha256 and size
store metadata in PostgreSQL
link to project / workflow_version / workflow_item / run / task / attempt
write event / audit when command-owned
```

Local artifact store 写入不等于写被管理项目。`write_artifacts` 只允许写 AreaFlow-owned artifact store 或
metadata evidence；不能被解释为 `write_generated`、`write_code` 或 `write_workflow`。

## Integrity

Artifact integrity report 必须拆分状态：

```text
pass:
  AreaFlow-owned local artifact 存在，sha256 和 size 与 PG metadata 一致。

warn:
  unknown backend、metadata incomplete、或需要人工策略判断。

skipped:
  project_reference / external_project / object backend 当前不读取原文，或 object verifier 尚未落地。

fail:
  local artifact 缺失、不可读、sha256 mismatch 或 size mismatch。
```

`skipped` 在顶层应至少提升为 `warn` / `needs_attention`，不能被累计成完整 pass。

## Backup Manifest

Backup manifest 是只读 metadata inventory，不是备份包。

Manifest 必须包含：

```text
schema_version
manifest_hash
table_counts
projects
project inventory
artifact metadata inventory
capabilities
forbidden_actions
generated_at
```

Manifest 允许：

```text
export_postgres_metadata
export_artifact_metadata
verify_manifest_hash
```

Manifest 禁止：

```text
read_artifact_contents
write_project_files
restore_database
delete_existing_state
resolve_secrets
apply_restore
```

`manifest_hash` 证明清单形状稳定，不证明 artifact 原文已打包，也不证明外部 project reference 可恢复。

## Restore Dry-run

Restore dry-run plan 把 backup manifest、project inventory、artifact inventory 和 artifact integrity 串成
恢复前检查链。它只回答：

```text
what metadata exists
which projects are covered
which artifacts are AreaFlow-owned local blobs
which artifacts are metadata-only project references
which integrity checks block restore planning
which actions remain forbidden
```

Restore dry-run 禁止：

```text
read external backup package
write database
write artifact store
write managed project files
delete existing state
resolve secrets
apply restore
```

如果存在 `project_reference` / `external_project` artifact，restore plan 必须返回 `needs_attention`，
除非 release exception 明确接受 metadata-only history，并记录 owner、风险、证据和后续 archive 计划。
如果存在 `object` artifact，只有 object verifier 已证明 key/version、sha256、size、namespace、encryption、
access policy 和 retention policy 后，restore dry-run 才能把它计入完整可恢复内容。

## Archive Preview And GC

`artifact.archive.preview` 是 metadata-only command。它可以写 command response、event 和 audit event，
但不得：

```text
copy artifact bytes
move artifact bytes
delete artifact bytes
delete artifact metadata
write managed project files
resolve secrets
```

真实 archive copy、object storage upload、retention-aware GC、orphan cleanup 和 delete apply 都是后续
Command API 能力。打开前必须满足 [`commands-and-approvals.md`](../../../concepts/commands-and-approvals.md) 的
idempotency、permission、approval、rollback 和 audit 要求，并满足
[`object-artifact-retention-contract.md`](object-artifact-retention-contract.md) 的 verifier、retention、
restore dry-run impact 和 delete suspension 规则。

第一版 GC/delete apply 只能处理 AreaFlow-owned `ephemeral` artifacts。普通 GC 永远不能处理 `audit`、
`release`、受保护的 `run_evidence`、`external_ref`、`legal_hold`、未知 retention class、hash mismatch
local artifact 或 verifier skipped/failed 的 object artifact。

## Restore Apply

真实 restore apply 不属于 v1.0。进入 v1.x 前必须先定义：

```text
restore package format
package hash and signature policy
isolated temp database validation
isolated temp artifact store validation
diff against current state
preimage / rollback plan
R4 approval
audit trail
secret redaction guarantee
managed project write boundary
```

Restore apply 不能直接覆盖当前 PostgreSQL、artifact store 或被管理项目。第一版必须先在隔离环境完成
dry-run、diff 和 rollback rehearsal。

## AreaMatrix First Policy

AreaMatrix 历史 v1 execution、progress、logs 和 evidence 第一阶段保持在 AreaMatrix 仓库。AreaFlow 只导入
index、hash、path、type、size 和 artifact metadata。

因此：

- AreaMatrix 历史 artifact 不因为被索引就变成 AreaFlow-owned backup 内容。
- Archive 前不得删除 AreaMatrix 历史 workflow、execution、progress、logs 或 release evidence。
- Restore plan 对 AreaMatrix historical reference 必须返回 `needs_attention` 或 accepted exception。
- Execution cutover 后的新 artifact 才应优先写入 AreaFlow-owned local artifact store。

## 关闭条件

v1.0 artifact / backup / restore chain 进入 release final gate 前至少需要：

```text
backup manifest returns stable manifest_hash
artifact integrity checks local hash / size
project_reference artifacts are marked skipped / needs_attention
restore dry-run forbids apply actions
archive preview is metadata-only
release readiness consumes restore and integrity status
release exception documents metadata_only_history if accepted
no restore_database / write_project_files / delete_existing_state / resolve_secrets / apply_restore side effects
```

任何 local artifact hash mismatch、missing blob、broken backup manifest、secret leak risk 或无 rollback 的
restore apply 都不能通过 release exception 放行。
