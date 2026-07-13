# ADR 0004: Import Mirror Shadow Authoring Cutover

## Status

Accepted as the AreaMatrix migration decision. 实施过程已经归档到 `docs/history/v1.0/migrations/**`。

## Decision

AreaMatrix workflow 迁移采用完整顺序：

```text
Import
-> Mirror
-> Shadow
-> Authoring Cutover
-> Execution Beta
-> Execution Cutover
-> Archive
-> Shim Retirement
```

短名仍可写作 `Import -> Mirror -> Shadow -> Cutover -> Archive`，但实现和门禁必须区分
`Authoring Cutover` 与 `Execution Cutover`。

## Rationale

AreaMatrix 当前 workflow 含有真实历史、发布证据、residual ledger 和 v1 execution。直接搬迁风险高。
分阶段迁移可以先验证模型，再切换新版本 authoring 所有权，最后才在 runner / worker / approval /
audit 闭环稳定后进入 execution cutover。

## Consequences

- v0.1 只做 Import + Status Mirror。
- v1 历史默认 immutable。
- v0.4 Authoring Cutover 只影响新 workflow version 的 authoring 源事实。
- Execution Cutover 必须等待 runner、worker、permission、approval、audit 和 rollback 证据完成。
- `./task-loop run` 在 execution cutover 前不得自动转发。
