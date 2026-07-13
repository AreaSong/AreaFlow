# Real Release Candidate Evidence

本文件是 AreaFlow completion audit 的 release-candidate evidence URI 锚点。
它不替代 `docs/**`、审计事件、proof record 或真实验证输出；最终判定仍以
`areaflow completion audit --json` 和 `areaflow completion audit-snapshot readiness areamatrix --json`
为准。

当前审计对象：`areamatrix`

当前日期：2026-07-13

## Source Alignment Reviewed

E1 用于绑定 AreaFlow v0 到 v1.0 的设计源、阶段边界和 preview/apply 语义。
proof record 必须证明 preview-only、implemented-scoped、deferred v1.x 能力没有被描述为真实 apply。

## Task Matrix Reviewed

E2 用于绑定 `tasks/backlog/0-100-platform-backlog.md` 与
`docs/development/task-backlog-status-audit.md` 的当前哈希，并证明 v0-v1.0 required task
没有隐藏 planned、missing evidence 或 blocked 项。

## Validation Reviewed

E3 用于绑定本轮 fresh validation 命令、结果哈希、开始/结束时间和验证范围。
该 proof 只能记录已实际运行并通过的命令；record 命令本身不运行验证。

## Archive Reviewed

E4 archive gate 目前仍受真实 AreaMatrix execution cutover 授权边界约束。
在未完成真实 cutover 前，不把 archive 记录为 release complete。

## Shim Retirement Reviewed

E4 shim retirement 目前仍受真实 AreaMatrix execution cutover 授权边界约束。
当前 Package B 只允许 read-only shim，不允许 `./task-loop run` 转发。

## Execution Cutover Reviewed

E4 execution cutover 仍未获准执行 run forwarding。
任何 proof 都不得声称 `task_loop_run_forwarding_window_proven`，除非后续用户明确授权并有真实证据。

## Release Packaging Reviewed

E5 release packaging 依赖 release final gate 与完整 E1-E9 proof 证据。
在 E4 未完成前，release final gate 仍不得被记录为 pass。

## Backup Restore Reviewed

E6 用于绑定 backup manifest、restore plan、artifact integrity 和 archive preview 输出。
要求证明没有 restore apply、artifact bytes copy/upload/delete 或 GC 行为。

## Security Closure Reviewed

E8 用于绑定 project-key isolation、默认只读/deny-first 权限、audit coverage 和 v1 范围内
未开放的 auth/team/token/secret/remote-worker 能力。

## Operations Reviewed

E7 用于绑定本地 ops proof、support bundle metadata-only 预览、local-only telemetry 和
migration ledger readiness。

## Protected Path Reviewed

E9 用于绑定 AreaMatrix protected paths 的 git status 输出、授权脏集、allowed paths、reviewer 和 rollback evidence。
当前只允许 Package A `.areaflow/status.json` 与 Package B read-only shim 相关路径，不允许扩大 AreaMatrix 写入范围。

## Release Candidate Reviewed

该锚点只可在 completion audit status 为 complete、proof event IDs 与 release-candidate evidence URIs
全部齐备、release evidence bundle ready 且 approved review metadata 完整后，用于记录 audit snapshot。
