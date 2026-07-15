# Deprecation And Retirement

AreaFlow 使用 SemVer 管理公开 API、CLI、配置和 artifact backend 契约。

- `/api/v1` 是 v1 规范入口；`/api` 是兼容 alias，最早在 v2 删除。
- 弃用至少跨一个 minor 版本且不少于 90 天。
- 弃用项必须有 owner、公告日期、调用方、替代方案、移除版本和验证方法。
- 功能开关、兼容层、临时表和双写逻辑必须设置到期日。

退役顺序固定为：识别调用方与数据、停止新增、迁移存量、观察无使用、关闭写入、移除契约与代码、清理权限/密钥/告警/资源、按保留策略处置数据、更新源事实并归档证据。

v1 不自动删除 artifact content。S3 归档或删除由企业受控流程执行，必须先检查 legal hold、retention deadline、审批和 hash inventory，再把结果作为审计证明记录。
