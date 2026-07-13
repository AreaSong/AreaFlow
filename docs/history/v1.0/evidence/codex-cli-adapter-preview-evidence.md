# Codex CLI Adapter Preview Evidence

## Purpose

本文记录 backlog 任务
[`AF-V06-002 Codex CLI Adapter Preview`](../plans/task-backlog.md#af-v06-002-codex-cli-adapter-preview)
的最近一次本机验证证据。

该 smoke 使用临时 PostgreSQL 数据库，验证受限 Codex CLI adapter preview 能返回 engine/profile
readiness、command preview、capability/path preflight 和 artifact redaction plan。它不执行 Codex CLI、
不运行 shell、不解析 secret、不写 artifact store、不写被管理项目文件。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/project ./internal/api ./internal/app
go build -o /tmp/areaflow-v06-codex-preview-smoke ./cmd/areaflow
```

Result: pass

Environment:

```text
PostgreSQL: docker compose service postgres, container areaflow-postgres, postgres:16-alpine, localhost:54329
Fixture database: af_v06codex_1782918599_88273
Project key: areamatrix
Workflow label: v06-codex-20260701230959
```

Focused smoke path:

```text
CREATE DATABASE af_v06codex_1782918599_88273
migrate up
project add --config examples/areamatrix/areaflow.yaml
engine codex-preview areamatrix --json
DROP DATABASE af_v06codex_1782918599_88273
```

Cleanup:

```text
residual_connections=0
```

## Result

Status: pass

Observed proof:

```text
status=blocked|mode=read_only_codex_cli_adapter_preview|profile=codex-cli|engine_status=blocked|command=codex exec|command_allowed=False|artifact_redaction=ready|execution_allowed=False|project_write_attempted=False|execution_write_attempted=False|engine_call_attempted=False|commands_run=False|secrets_resolved=False|network_used=False|blockers=missing_capability:execute_agents,missing_capability:run_commands,engine_profile_disabled,command_not_allowed|capability_count=4|path_count=11
db=af_v06codex_1782918599_88273 label=v06-codex-20260701230959
migrations=10 project_add=registered areamatrix /Users/as/Ai-Project/project/AreaMatrix
connections_before_drop=0 connections_after_drop=0
```

## Evidence

- `go test ./internal/project ./internal/api ./internal/app` passed.
- `go build -o /tmp/areaflow-v06-codex-preview-smoke ./cmd/areaflow` passed.
- Migrations applied from an empty temporary PostgreSQL database.
- `project add --config examples/areamatrix/areaflow.yaml` registered AreaMatrix metadata only.
- `engine codex-preview areamatrix --json` returned `status=blocked` and
  `mode=read_only_codex_cli_adapter_preview`.
- Preview identified `codex-cli` profile as blocked because the profile is disabled.
- Preview identified `execute_agents` and `run_commands` as missing capabilities.
- Preview identified the proposed command `codex exec` as not allowed.
- Preview returned artifact redaction plan status `ready`.
- Preview returned `execution_allowed=false`.
- Preview returned `project_write_attempted=false`, `execution_write_attempted=false`,
  `engine_call_attempted=false`, `commands_run=false`, `secrets_resolved=false` and
  `network_used=false`.
- Temporary PostgreSQL database was dropped and residual connection count was `0`.

## Boundary

This proves v0.6 Codex CLI adapter preview only. It does not prove real Codex CLI execution, copy/verify/repair,
checkpoint, secret resolution, shell command execution, artifact store writes, managed project writes or execution
cutover.
