# AreaFlow CLI

CLI 与 Web/API 使用相同的项目、workflow、run、worker、artifact 和 audit 模型。

```bash
areaflow help
areaflow version
areaflow health
```

## 命令组

| 命令组 | 用途 |
|---|---|
| `migrate` | PostgreSQL schema 初始化和状态 |
| `server`、`service` | API 服务和本地状态 |
| `project` | 项目注册、导入、doctor、readiness 和兼容检查 |
| `workflow` | profile、版本、stage、gate、transition 和 approval |
| `run` | run 创建、计划、gate 和状态控制 |
| `worker` | worker、heartbeat、lease 和受限任务执行 |
| `artifact` | integrity 和 archive preview |
| `audit`、`permissions` | 审计覆盖和权限诊断 |
| `ops`、`backup`、`support` | 运维、备份清单和 support metadata |
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
