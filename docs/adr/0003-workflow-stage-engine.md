# ADR 0003: Workflow Stage Engine

## Status

Accepted for Phase 0 planning.

## Decision

AreaFlow 使用通用 stage engine + workflow profile。AreaMatrix lifecycle 是第一个内置 profile。

## Rationale

硬编码 AreaMatrix 流程会限制 AreaFlow 支持其他项目类型。通用 stage engine 可以复用 Gate、Artifact、Transition、Permission 和 Validation 模型。

## Consequences

- `internal/workflow/engine` 后续实现通用引擎。
- `workflow/profiles/areamatrix` 保存 AreaMatrix profile 源事实。
- AreaMatrix adapter 将现有文件映射到通用模型。
