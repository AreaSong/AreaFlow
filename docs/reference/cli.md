# AreaFlow CLI

CLI 与 Web/API 使用相同的项目、workflow、run、worker、artifact 和 audit 模型。CLI 是同一 Go binary 的受控本地管理面，通过 in-process domain Command/Query API 访问 Store，不嵌入 SQL；Web/Desktop 使用 `/api/v1` REST API。两条路径共享幂等、权限、审批和审计边界。

```bash
areaflow help
areaflow version
areaflow version --json
areaflow health
```

## 命令组

| 命令组 | 用途 |
|---|---|
| `migrate` | PostgreSQL schema、checksum 状态和 legacy attestation |
| `server`、`service` | API 服务和本地状态 |
| `project` | 项目注册、导入、doctor、readiness 和兼容检查 |
| `workflow` | profile、版本、stage、gate、transition 和 approval |
| `run` | run 创建、计划、gate 和状态控制 |
| `worker` | worker、heartbeat、lease 和受限任务执行 |
| `artifact` | integrity、backend migration 和 archive preview |
| `audit`、`permissions` | 审计覆盖和权限诊断 |
| `ops`、`backup`、`support` | 运维、真实备份包、隔离恢复演练和 support metadata |
| `auth` | scoped service token 创建、列表、轮换链和撤销 |
| `release`、`completion` | release gate、exception 和完成审计 |
| `desktop`、`security` | Desktop 与安全边界诊断 |

## 输出

多数查询和命令支持 `--json`。自动化和 Web adapter 应优先消费 JSON，不解析面向人的文本输出。

## 写操作

写操作通常要求：

```text
--actor
--reason
--idempotency-key
```

部分高风险操作还要求 approval、expected preimage、capability、path 和 rollback 信息。CLI 参数存在不代表操作已开放；最终结果以 permission、gate 和 audit response 为准。

完整实时命令列表以 `areaflow help` 为准。长期应从命令定义生成结构化 reference，避免手工复制帮助文本。

## Token

```bash
areaflow auth token create --actor operator --reason "web access" \
  --project areamatrix --capability read --capability workflow.approval.record
areaflow auth token list --json
areaflow auth token revoke <token-key> --actor operator --reason "session retired"
```

明文 token 只在创建时返回一次；数据库只保存 SHA-256。不要把 create 的 JSON 输出写入 CI 日志。
未指定到期时间时默认 30 天；服务端拒绝超过 90 天的 token。Token 应在轮换成功并验证新 token 后撤销旧 token。

## Migration checksum

`migrate status` 同时显示 `verified`、`legacy_unverified`、`mismatch` 或 `pending`。新 migration 在同一事务写入 SHA-256；历史行只能通过显式 attestation 绑定：

```bash
digest="$(areaflow migrate set-digest)"
areaflow migrate attest-legacy-hashes --actor migration-owner \
  --reason "reviewed embedded migration set" --expected-set-digest "$digest"
```

## Backup 与 Drill

停止 AreaFlow 写入进程后创建包：

```bash
areaflow backup create --destination .areaflow/backups/<backup-id> --quiesced
areaflow backup drill --package .areaflow/backups/<backup-id> \
  --actor backup-operator --reason "isolated restore verification"
```

Drill 只恢复到新建的隔离数据库和 artifact root，不覆盖当前数据库或 artifact store。

## Artifact migration

```bash
areaflow artifact migration inventory areamatrix --source-backend local --target-backend s3 --json
areaflow artifact migration copy areamatrix 42 --target-backend s3 --target-root production-migration \
  --actor data-operator --reason "copy and verify"
areaflow artifact migration activate areamatrix 42 --location-id 77 \
  --observe-until 2026-08-01T00:00:00Z --actor data-operator --reason "switch verified primary"
areaflow artifact migration complete-observation areamatrix 42 \
  --actor data-operator --reason "observation window passed"
```

`copy`、`activate` 和 `complete-observation` 写入 append-only audit event。`activate` 只接受已验证 location；完成观察后稳定态读取不再回退 local source。迁移期间不存在删除源位置的 CLI。
