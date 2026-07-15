# 参与 AreaFlow 开发

提交变更前先阅读 [`AGENTS.md`](AGENTS.md) 的产品边界、文档源事实和安全规则。

## 开发流程

1. 从 `main` 创建短生命周期分支。
2. 保持改动聚焦；用户行为或公开契约变化时，就地更新对应长期文档。
3. 运行 `make check`。涉及 Web 交互时再运行 `make smoke-docker-web`。
4. Pull Request 说明行为变化、验证结果、风险和未验证项。

每项变更必须按 [`governance/README.md`](governance/README.md) 判定 L0-L4，并通过对应 G0-G8 门禁。工作状态、RAID 和事故改进使用 GitHub Issue/Project 维护；长期产品事实仍只写入 `docs/**`、`governance/**`、代码、migration 或机器可读契约。

提交必须使用 Developer Certificate of Origin（DCO）签署：`git commit -s`。贡献者保留版权，并按 Apache-2.0 许可提交贡献。

默认不为单个功能创建 plan、progress、evidence 或 completion Markdown。测试日志与运行证据应由 CI、测试系统或 AreaFlow artifact store 保存。

## 安全边界

认证、授权、secret、remote worker、通用 engine execution、restore apply、publish 或高风险项目写入必须先完成独立设计、安全评审和明确批准。不得通过 Web、Desktop、worker 或测试绕过 Command、approval 和 audit 边界。
