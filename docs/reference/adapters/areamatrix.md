# AreaMatrix Adapter

AreaMatrix 是 AreaFlow 的第一个 dogfooding 项目和内置 adapter/profile 组合。AreaFlow 从 AreaMatrix 读取 workflow、residual、task 和 artifact metadata，但不取得 AreaMatrix 产品文档、源码、用户文件或历史 evidence 原文的所有权。

## Import

Import 建立 metadata index，包括允许文件的 path、hash、size、类型、版本关系和状态摘要。大内容保留在 AreaMatrix，AreaFlow 保存 URI 和 metadata。

默认不导入 prompt、日志、报告、diff、checkpoint、release evidence 原文或用户文件内容。

## Ownership

AreaMatrix 保留：

- `docs/**` 与源码。
- 项目验证命令和治理规则。
- 历史 workflow、execution、progress、logs 和 release evidence。
- 用户文件安全边界。

AreaFlow 管理注册后的 Project、Workflow Version、Run、Worker、Artifact index、Event 和 Audit Event。

## 写入

项目配置默认只读。任何 `.areaflow/status.json`、generated file 或其他项目写入都必须由 project config 显式允许，并通过 capability、path allowlist、gate、approval、Command API 和 audit。

AreaMatrix profile 见 [`workflow/profiles/areamatrix/profile.yaml`](../../../workflow/profiles/areamatrix/profile.yaml)。一次性迁移和 cutover 资料保存在 [`docs/history/v1.0/migrations`](../../history/v1.0/migrations/README.md)。
