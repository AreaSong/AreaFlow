# Permission Governance

项目写入至少需要 capability、path allowlist、gate result、approval record 和 audit event。

- deny 与 forbidden path 优先于 allow。
- 未声明 capability 默认拒绝。
- Preview、readiness 和 gate 不等于 apply。
- Actor、reason、idempotency key 和 expected state 必须进入统一 command contract。
