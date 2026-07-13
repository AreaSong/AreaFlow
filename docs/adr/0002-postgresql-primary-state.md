# ADR 0002: PostgreSQL Primary State

## Status

Accepted for Phase 0 planning.

## Decision

PostgreSQL 是 AreaFlow 主状态源事实。v0.1 不提供 SQLite fallback。

## Rationale

AreaFlow 长期需要多项目、多 worker、事务、行级锁、审计、JSONB metadata、索引和未来团队部署。PostgreSQL 更适合这些要求。

## Consequences

- 本地开发通过 Docker Compose PostgreSQL 起步。
- 测试覆盖 PostgreSQL，不用 SQLite 模拟主状态。
- 文件只保存配置、artifact 原文和审计导出。
