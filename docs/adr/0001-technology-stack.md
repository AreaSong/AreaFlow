# ADR 0001: Technology Stack

## Status

Accepted for Phase 0 planning.

## Decision

AreaFlow 使用：

```text
Go backend / CLI / scheduler / worker
PostgreSQL primary state
REST/JSON API
SSE realtime events
React + TypeScript web dashboard later
Tauri desktop shell later
```

## Rationale

Go 适合长期 CLI、server、worker、调度器和单 binary 分发。PostgreSQL 适合多项目、多 worker、高并发状态平台。Web 和 Desktop 后续作为 API client 接入，不拥有核心业务逻辑。
