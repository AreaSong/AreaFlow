# AreaFlow Roadmap

Roadmap 只记录尚未成为当前产品事实的方向。已完成的阶段、milestone 和实施 evidence 保存在 `docs/history/**`，不在本文件重复维护。

## 查询契约补齐

- 为 Run Task、Attempt、Worker heartbeat 和 Lease 等子资源增加独立 cursor，而不是只能返回完整列表或共享 history limit。
- 评估 Projects 集合的过滤和 cursor 契约，保持与其他全局资源一致的错误和分页语义。

## Web command contract

- 统一 actor、reason、idempotency key、expected state 和 confirmation envelope。
- 让 Web write action gate 与实际 HTTP 写入 endpoint 使用同一授权事实。
- 优先开放低风险 AreaFlow DB 操作。
- 项目文件写入继续要求 capability、path allowlist、gate、approval、audit 和 rollback。

## 项目与 Workflow 管理

- 项目 create、update、archive 和配置检查 API。
- workflow item 详情和 mark-ready API。
- workflow transition apply 的状态机契约。
- profile list、show 和 conformance API。

## Operations jobs

- 将 backup、restore、support bundle、publish 和 rollout 建模为可查询 job。
- 区分 plan/preview、approval、execution 和 verification。
- 为长任务提供事件、失败恢复和审计关联。

## 平台扩展

以下能力需要独立 proposal、安全设计和开闸验证：

- team lifecycle、membership invitation 和 OIDC group/team 自动映射。
- external webhooks 与 callback credential。
- secret resolve 和 credential lifecycle。
- remote workers 和通用 AI engine execution。
- 第三方 plugin execution 与 marketplace。
- budget、quota、usage metering 和 managed operations。

OIDC users、project role bindings、Web session 和 scoped service tokens 已进入当前架构、API 与配置文档，不再属于 roadmap。上述剩余方向在实现、测试和验证完成前，不进入当前功能说明。
