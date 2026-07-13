# Object Artifact Store / Retention Contract

## 定位

本文定义 AreaFlow 从 local artifact store 扩展到 object artifact store、archive copy、retention-aware GC
和 delete apply 的边界。它补充
[`artifact-backup-restore-contract.md`](artifact-backup-restore-contract.md)、
[`command-approval-contract.md`](../history/v1.0/contracts/command-approval-contract.md) 和
[`high-risk-apply-ladder.md`](../../proposals/high-risk-apply.md)。

v1.0 只要求 metadata inventory、local artifact integrity、restore dry-run 和 metadata-only archive
preview 可解释。Object artifact store、archive copy / upload、retention-aware GC、orphan cleanup 和
delete apply 都属于 v1.x 能力；其中任何会删除、移动、覆盖、上传或改变恢复范围的动作都必须走
Command API、approval、rollback / revoke 和 audit。

## Storage Backend

Artifact backend 分层：

```text
project_reference / external_project:
  原文仍在 managed project 或外部位置。AreaFlow 只保存 metadata、hash、path、size、type 和关系。

local:
  AreaFlow-owned local artifact store。AreaFlow 可以读取原文并校验 sha256 / size。

object:
  未来对象存储。AreaFlow 必须通过 object verifier 校验 object key、etag / version、sha256、size、
  retention policy 和 access policy 后，才能把它计入完整可恢复内容。
```

`object` backend 在 verifier 落地前必须返回 `skipped` / `needs_attention`，不能被当作 `pass`。
`project_reference` / `external_project` 即使有 hash，也不是 AreaFlow-owned content。

## Retention Class

稳定 retention class：

```text
ephemeral:
  可重建 preview / 临时报告。只能由未来显式 GC command 清理。

run_evidence:
  runner / worker / verify / repair / failure evidence。至少保留到 run closeout 和 release gate。

audit:
  approval、permission、command、write、security evidence。默认长期保留。

release:
  release readiness、evidence bundle、package/distribution/publish preview。默认长期保留。

external_ref:
  原文留在 managed project 或外部系统。AreaFlow 不删除、不修复，只标记限制。

legal_hold:
  法务、合规或人工 hold。任何 archive / GC / delete apply 都必须 blocked。
```

未知 retention class 必须进入 `needs_policy`，不能自动 archive 或 delete。

## v1.0 Archive Preview

`artifact.archive.preview` 只允许：

```text
read artifact metadata
classify retention class
return archive_candidate / retained / metadata_only_reference / needs_policy
write command response
write event / audit event for preview
```

它必须继续禁止：

```text
copy artifact bytes
move artifact bytes
upload artifact bytes
delete artifact bytes
delete artifact metadata
change storage backend
write managed project files
resolve secrets
```

`ephemeral` 只能显示 `eligible_for_future_gc_preview`，不能自动删除。`external_ref` 只能显示
`requires_archive_ownership_decision`，不能复制、删除或重命名 managed project 原文。

## Object Store Opening Ladder

Object artifact store 按以下顺序打开：

```text
object backend schema metadata
-> object verifier preview
-> object write fixture
-> object read integrity fixture
-> object upload scoped beta
-> object restore dry-run integration
-> object retention policy preview
-> object archive copy / upload command
-> object delete / GC preview
-> object delete / GC apply
```

每一步都必须有独立 command 或 preview。Object upload 不等于 delete；archive copy 不等于 GC；
restore dry-run integration 不等于 restore apply。

## Object Verifier

Object verifier 必须证明：

```text
object key / URI belongs to project namespace
object version or etag is stable
sha256 matches metadata
size matches metadata
content type is expected
encryption policy is known
access policy is project-scoped
retention / lifecycle policy is visible
missing object returns fail
unknown provider returns warn
```

Verifier 不能读取 secret 明文到 logs、artifact、event 或 audit。Provider credentials 必须按
[`auth-team-secret-boundary.md`](../../proposals/auth-team-secret.md) 的 scoped secret resolve 规则打开。

## Archive Copy / Upload

Archive copy / upload 是写 artifact store 的真实 apply，必须具备：

```text
source artifact id
source backend
target backend
target object key
expected source sha256 / size
copy artifact hash verification
idempotency_key / request_hash
approval scope
rollback / revoke plan
audit event
```

Archive copy 不得删除 source。复制成功后，metadata 可以新增 `artifact_snapshots` 或新的 artifact
location，但不能把 old reference 静默改写成 owned object，除非 copy verification、approval 和 audit
全部通过。

## GC / Delete Apply

GC 和 delete apply 是最高风险 artifact 操作之一。打开前必须先有：

```text
retention policy
legal_hold check
release gate not depending on artifact
backup manifest reference check
restore dry-run impact report
owner approval
pre-delete manifest
delete command idempotency
post-delete verification
audit event
rollback / remediation plan
```

GC / delete 第一版只能处理 AreaFlow-owned `ephemeral` artifacts。以下永远不能被普通 GC 删除：

```text
audit
release
run_evidence before closeout / release gate
external_ref
legal_hold
unknown retention class
local artifact with hash mismatch
object artifact with verifier skipped / failed
```

删除 artifact bytes 不能删除 audit/event 历史。删除 metadata 需要单独设计，默认禁止。

## Backup / Restore Interaction

Backup manifest 默认仍是 metadata inventory，不是 backup package。Object store 打开后，manifest 必须区分：

```text
metadata_only
local_content_verified
object_content_verified
object_verifier_skipped
external_reference
missing_content
```

Restore dry-run 只有在 local 或 object content 已通过 verifier 时，才能把对应 artifact 计入完整可恢复。
`project_reference` / `external_project`、`object_verifier_skipped`、`missing_content` 必须返回
`needs_attention` 或 `blocked`。

Restore apply 仍按 [`artifact-backup-restore-contract.md`](artifact-backup-restore-contract.md) 和
[`high-risk-apply-ladder.md`](../../proposals/high-risk-apply.md) 的 R4 restore apply 规则执行；object verifier
通过不能绕过 restore approval。

## AreaMatrix First Policy

AreaMatrix historical artifact 第一阶段保持 `project_reference` / `external_project`：

- 不复制历史 `workflow/versions/**/execution/**`、`progress.json`、logs、prompt 或 release evidence 原文。
- 不删除 AreaMatrix 历史 artifact 原文。
- 不把 metadata-only history 计入完整 backup package。
- Archive ownership decision 只能解释限制，不能自动获得 copy/delete 权限。
- 如果未来复制历史原文到 AreaFlow-owned local/object store，必须先有 explicit approval、hash verification、
  non-touch proof 和 rollback / revoke plan。

AreaMatrix 真实对象存储试点不得早于 local artifact store、backup manifest、restore dry-run 和 release
exception 链路稳定。

## Suspension Rule

以下情况必须立即 suspend object / archive / GC apply：

```text
object hash mismatch
object size mismatch
object provider unauthorized
object key outside project namespace
unexpected source deletion
delete affected protected retention class
backup manifest lost referenced artifact
restore dry-run impact mismatch
audit event missing
secret leaked in object metadata or logs
legal_hold ignored
```

恢复前必须重新通过 verifier、manifest diff、restore dry-run impact、owner approval、focused smoke 和 audit。

## 关闭条件

v1.x object artifact store task 进入 active 前，至少需要：

```text
object verifier contract
provider credential boundary
project namespace policy
upload command contract
archive copy command contract
GC / delete preview contract
GC / delete apply approval and rollback plan
backup manifest status vocabulary
restore dry-run impact report
AreaMatrix non-touch proof
```

缺少这些证据时，只能继续使用 v1.0 metadata-only archive preview 和 local artifact integrity。
