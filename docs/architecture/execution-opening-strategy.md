# Execution Opening Strategy

## 定位

本文定义 AreaFlow 从只读 preview 逐步打开真实 execution 的开闸策略。它补充
[`execution-model.md`](execution-model.md)、[`v0.5-runner-preview-contract.md`](v0.5-runner-preview-contract.md)、
[`v0.6-worker-beta-contract.md`](v0.6-worker-beta-contract.md)、
[`security-permissions.md`](security-permissions.md) 和
[`../migration/areamatrix-workflow-migration.md`](../migration/areamatrix-workflow-migration.md)，回答：

```text
哪些 execution 能力已经打开
下一步打开什么
打开前必须证明什么
失败后如何回退
哪些能力即使 readiness 为 ready 也仍然关闭
```

本文不是 apply 授权。任何真实写入、命令执行、agent execution、secret resolve、checkpoint 或 repair
apply 仍必须通过 Command API、permission、gate、approval、focused smoke 和 audit。
post-100% secret、remote worker、restore、publish、plugin 和其他 R4 real apply 的统一顺序见
[`high-risk-apply-ladder.md`](high-risk-apply-ladder.md)。

## 当前开闸基线

当前 v0.6 路线已经允许的 execution 范围是受限、可审计、分阶段的：

v0.6 的每条能力必须保留
[`v0.6-worker-beta-contract.md`](v0.6-worker-beta-contract.md) 定义的 scope label 或等价 safety facts。
`open scoped` 只表示该 scope 的闭环已打开，不表示下一阶梯、真实 AreaMatrix 写入或 execution cutover 打开。

| 能力 | 状态 | 说明 |
|---|---|---|
| runner preview | open | 只写 AreaFlow state / artifact metadata，`dry_run=true`。 |
| worker registry / heartbeat / lease | open | 只管理 worker lifecycle，不代表真实任务执行。 |
| worker run-once dry-run | open | 只领取 dry-run `run_task`，不做 copy/verify。 |
| fixture execution apply | open scoped | 只写 AreaFlow PG state 和 artifact store。 |
| read-only verify | open scoped | 只读取 allowlisted target file 的 path/hash/size evidence。 |
| approved artifact write | open scoped | 只写 AreaFlow-owned artifact store，不写项目文件。 |
| fixture project write rollback drill | open scoped | 只允许 fixture/temp project，写入后必须恢复 preimage。 |
| fixture/temp generated write rollback drill | open scoped | 只允许 fixture/temp project generated path，写入后必须恢复 preimage。 |
| real AreaMatrix generated rollback beta | closed | 需要单独 R3 approval 和 non-target fingerprint evidence。 |
| retained generated apply | closed | 需要 rollback beta 证据稳定后单独开闸。 |
| source write beta | closed | 需要 manual patch / human-applied evidence 缓冲层。 |
| checkpoint apply | closed | 不能和 source write beta 同时隐式打开。 |
| repair apply | closed | 只能在 verify failure evidence 后追加 attempt。 |
| Codex CLI / engine execution | closed | v1.0 前默认 no real engine call；no-secret execution 也需单独 gate。 |
| secret resolve / remote worker / publish / restore apply | closed | v1.x 高风险能力。 |

任何 UI、CLI、API 或文档不能把 `open scoped` 解释成真实 AreaMatrix execution cutover。

## 不变量

Execution 打开必须遵守以下不变量：

- Query API 只读，不创建 command、lease、attempt、artifact、approval 或 audit。
- 所有业务写动作必须进入 Command API，并带 `idempotency_key`、request hash、actor、risk 和 audit。
- `workflow_item` 表示要做什么；worker 只能领取 `run_task`，不能直接领取 `workflow_item`。
- `run_attempt` append-only；copy、verify、repair、checkpoint 分开记录，不能覆盖旧 attempt。
- 命令退出码 0 不等于完成；只有 verify evidence 和 acceptance gate 能推进 done。
- `write_artifacts` 只表示写 AreaFlow-owned artifact store，不表示写被管理项目。
- `write_generated` 只表示 allowlisted generated/projection path，不表示写源码、execution、progress、logs 或 checkpoint。
- `approval` 不等于 execution；`readiness` / `preview` / `gate` 不等于 apply。
- expected-before hash 不匹配时必须 blocked，不能自动覆盖用户或其他 worker 的新改动。
- rollback/remediation plan 缺失时，R2-R4 操作不能放行。

## 开闸阶梯

Execution 能力按以下顺序打开。后一步不能因为前一步的 preview 或 readiness 返回 `ready` 就自动打开。

| 阶梯 | 能力 | 允许 | 仍禁止 | 必要证据 |
|---:|---|---|---|---|
| 0 | runner preview | 创建 dry-run run/task/attempt/artifact | worker 领取真实 task、项目写入、engine | runner preview report、safety facts |
| 1 | worker lifecycle | register、heartbeat、lease acquire/release/recover | copy/verify、shell、project write | lease TTL、heartbeat、capability denial |
| 2 | fixture execution | fixture-only AreaFlow execution apply | engine、project write、execution files | run/task/attempt/artifact/audit 闭环 |
| 3 | read-only verify | 读取 allowlisted file hash/size | 保存原文、写文件、运行命令 | read-only verify report |
| 4 | approved artifact write | 写 AreaFlow artifact store | 写项目文件、写 execution、engine | artifact hash/URI、audit |
| 5 | fixture project write rollback drill | fixture/temp project 单文件 modify + rollback | 真实 AreaMatrix、create/delete/move | preimage、copy、verify、rollback_verified |
| 6 | fixture/temp generated drill | fixture/temp generated path modify + rollback | retained apply、source write | generated write-set、rollback_verified |
| 7 | real generated rollback beta | 真实 managed project 单文件 generated 写入后恢复 | 保留结果、source write | R3 approval、expected-before、non-target fingerprint |
| 8 | retained generated apply | 真实 generated/projection 单文件保留写入 | source write、checkpoint、repair | rollback plan、focused smoke、audit |
| 9 | manual patch artifact | 生成 patch/diff/write-set artifact | AreaFlow 自动写源码、运行 shell | patch hash、verification plan、rollback plan |
| 10 | human-applied source evidence | 读取人工 apply 后的 diff/hash/验证 | 自动源码写入、checkpoint apply | changed file hash、verify evidence |
| 11 | source write beta | allowlisted text create/modify，copy -> verify -> checkpoint preview | delete/move/chmod/binary/glob/checkpoint apply | write-set gate、expected-before、verify pass |
| 12 | checkpoint apply | 受控 git/checkpoint apply | 未验证 task 进入 checkpoint | dirty state、scope drift、rollback/remediation |
| 13 | repair plan/apply | failure summary -> repair plan -> append attempt | 跳过 verify 或 checkpoint gate | verify failure evidence、repair attempt、re-verify |
| 14 | no-secret engine execution | `secret_ref=none` 的 scoped engine run | secret、remote worker、unbounded network | budget、redaction、command allowlist |
| 15 | secret / remote / restore / publish / plugin | v1.x 高风险能力 | 作为 v1.0 默认能力 | R4 approval、scoped credentials、focused smoke |

第 15 阶梯只表示 execution strategy 把这些能力保持关闭。它们进入 v1.x active 前，必须按
[`high-risk-apply-ladder.md`](high-risk-apply-ladder.md) 的状态词、apply packet、suspension rule 和
AreaMatrix first policy 逐项开闸。

## Copy / Verify / Repair / Checkpoint

### 状态推进矩阵

Execution 状态推进必须按 attempt evidence 决定，不能按 worker 返回码或 UI action 决定：

| 当前事实 | 可推进到 | 继续禁止 |
|---|---|---|
| copy attempt succeeded | `verifying` 或 `checkpoint_ready` 前置状态 | workflow item done、run passed、checkpoint apply |
| verify attempt passed | `checkpoint_ready` 或 acceptance gate candidate | 自动 checkpoint、自动 next task、自动 closeout |
| verify attempt failed | `repair_needed` 或 `failed` | mark done、checkpoint、覆盖旧 attempt |
| repair plan created | `repair_needed` with plan evidence | repair apply、source write、checkpoint |
| repair apply succeeded | new verify required | 跳过 verify、覆盖 failed attempt |
| checkpoint preview passed | checkpoint approval candidate | git/apply checkpoint、next task |
| checkpoint apply passed | next task candidate | release/cutover 自动通过 |
| checkpoint apply failed | `blocked_checkpoint` / `needs_attention` | 降级为 warn、继续队列 |
| rollback verified | rollback drill passed / evidence retained | retained apply、source write、execution cutover |

`run_task.status=passed` 只能在该 task 的 required verify / acceptance evidence 已满足时使用。早期兼容
状态 `verified`、`artifact_written`、`rollback_verified` 必须解释为 scoped outcome；不能继续扩展成新的
task status。

### Copy

Copy 是写入或产生变更的 attempt。它可以是 artifact-only、generated-only、manual patch artifact、
human-applied evidence 或 source write beta。Copy 成功只表示本次变更动作完成，不表示任务完成。

Copy apply 前必须存在 approved write-set。Worker 或 engine adapter 不能绕过 write-set 直接写项目文件。
Copy attempt 必须记录 command id、write-set artifact、precondition snapshot、affected resources、
expected-before result、post-copy hash 和 safety facts。缺少这些证据时只能进入 `needs_attention`，
不能被 verify 或 checkpoint 消费。

### Verify

Verify 是只读 attempt。它可以读取 allowlisted project files、运行明确允许的只读 doctor 或校验 artifact
metadata，但不能修复、重写或 checkpoint。

Verify 失败时必须产出 failure summary artifact，并把 run_task 推到 `repair_needed` 或 `failed`；
不能把 copy 成功的任务标成 done。
Verify pass 只能证明 verify scope 内的事实成立；如果 verification plan 没覆盖 changed files、generated
artifact、expected outputs 或 release gate 所需证据，状态必须保持 `needs_attention`。

### Repair

Repair 只能由 verify failure evidence 触发。第一阶段只生成 repair plan artifact；repair apply 打开后也必须
追加新的 repair/copy/verify attempt，不能修改旧 attempt。

Repair apply 不得跳过 approval、path allowlist、expected-before、verification plan 和 checkpoint gate。
Repair plan 不是 repair approval。Repair apply 必须重新走 Command API、permission、approval、
expected-before、write-set、rollback/remediation 和 audit。Repair 成功后必须重新 verify；不能把 repair
success 直接解释为 task passed。

### Checkpoint

Checkpoint 是独立 attempt，不是 copy 的副作用。Checkpoint apply 第一版只在 source write beta 多次证明
verify 稳定后打开，并且必须检查：

```text
dirty state
scope drift
expected changed files
non-target fingerprints
rollback/remediation plan
audit event
```

Checkpoint fail 会阻断下一 task。不能为了让队列继续前进而把 checkpoint fail 降级为 warn。
Checkpoint preview 不是 checkpoint apply。Checkpoint apply 不能由 verify pass、source write beta 或
release final gate 隐式打开；它必须具备单独 command、approval、dirty-state check、scope drift check、
rollback/remediation 和 audit。

### Execution Cutover Proof

Execution cutover 不能由局部能力累加推导出来。以下证据只能证明各自 scope：

```text
fixture execution pass:
  证明 AreaFlow PG state / artifact store 闭环。

read-only verify pass:
  证明 allowlisted file hash/size evidence 可生成。

approved artifact write pass:
  证明 AreaFlow-owned artifact store 写入可审计。

fixture/temp rollback drill pass:
  证明 fixture/temp path 的 write/verify/rollback 链路可恢复。

generated write readiness ready:
  证明具备人工审查资格，不打开 apply。

generated write apply beta gate needs_approval:
  证明仍缺 explicit R3 approval，不打开 apply。
```

AreaMatrix execution cutover 必须额外证明真实 approved task 的 copy、verify、repair/checkpoint gating、
artifact/evidence、audit、compat command behavior、protected-path proof 和 rollback/remediation 全部可复验。
v0.6 的 fixture、read-only、artifact-only、fixture/temp rollback drill、readiness 和 beta gate 证据即使全部
通过，也只能作为后续 execution cutover 的输入材料，不能直接合并成 cutover approval。

Execution cutover 的第一版落地必须更小，称为 `Execution Forwarding v1`。它只允许 AreaMatrix
`./task-loop run` 转发到 AreaFlow 受保护 Command API，并且只消费 read-only / evidence 类任务：

```text
verify-ready task
doctor/readiness task
artifact evidence task
status/projection validation task
release/readiness check task
```

`Execution Forwarding v1` 不能打开：

```text
copy-ready source write
generated retained write
repair apply
checkpoint apply
no-secret engine execution
secret-backed engine execution
network/API integration
publish / restore / plugin apply
```

这意味着第一版 cutover 可以证明“AreaFlow 已经成为执行入口和审计主状态”，但不能证明自动写代码、
自动修复、git checkpoint、engine execution 或高风险能力已经打开。后续每一项都必须沿开闸阶梯单独通过
architecture contract、Command API、permission、approval、focused smoke、rollback/remediation 和 audit。

## Failure And Recovery

Worker 失联、lease 过期或 run cancel 都不能删除历史 evidence：

- active lease 过期后进入 `needs_recovery`，由 recovery command 决定重试、释放或人工介入。
- `cancel_requested` 是协作式停止；已经写出的 artifact、event 和 audit 保留。
- expected-before mismatch 表示外部状态变化，进入 blocked 或 needs_review，不自动 overwrite。
- verification fail 进入 repair_needed；repair 之前必须保留 failure summary。
- rollback fail 是高风险阻断，必须进入 needs_attention，不能继续下一 task。

## AreaMatrix First Execution Policy

AreaMatrix 是第一个 dogfood project，因此真实 AreaMatrix execution 必须比 fixture 更保守：

1. 第一批真实 execution 优先 read-only verify、doctor/readiness 和 artifact evidence。
2. `./task-loop run` 的第一版 forwarding 只转发 read-only / evidence 类任务，并保持旧 runner、旧
   `progress.json`、旧 logs 和旧 checkpoint 非写入。
3. 真实 generated-only 先做 rollback beta，证明写入后能恢复 preimage。
4. retained generated apply 只允许 `.areaflow/generated/**`、`.areamatrix/generated/**` 或 project config
   显式声明的 generated/projection target。
5. source write beta 前必须先经历 manual patch artifact 和 human-applied source evidence。
6. `workflow/versions/**/execution/**`、`progress.json`、旧 logs、checkpoint 和 `./task-loop run`
   forwarding 在 execution cutover approval 前保持 protected path。
7. AreaMatrix 第一阶段 execution 并发默认 1；多项目 worker pool 只能做 schedule preview，真实自动调度另行开闸。

## 多端控制面

CLI、Web、Desktop 和 Worker 必须共用同一 API/service layer：

- Web/Desktop 可以展示 execution plan、approval gate、worker health 和 cutover readiness。
- Web/Desktop 不直接写 PostgreSQL、不直接读本地 artifact path、不直接启动 worker。
- 任何按钮如果会写入、执行、drain、cancel、checkpoint、repair、restore 或 publish，必须映射到 Command API。
- Disabled/blocked action 必须显示 blockers 和 safety facts，不能隐藏成“按钮暂不可用”。
- SSE 只观察 events，不推进状态。

## 关闭条件

某一 execution 阶梯要从 closed 变成 open，至少需要：

```text
architecture contract updated
API / CLI command defined
service tests
API tests when surfaced
focused smoke evidence
permission denial test
safety facts showing forbidden actions remain false
audit event evidence
rollback or remediation evidence when writing
AreaMatrix non-target fingerprint proof when touching real AreaMatrix
implementation gap audit updated
```

如果证据只能覆盖 fixture、preview、readiness 或 rollback drill，就只能标记为 `implemented_scoped` 或
`preview_only`，不能累计成真实 AreaMatrix execution cutover。
