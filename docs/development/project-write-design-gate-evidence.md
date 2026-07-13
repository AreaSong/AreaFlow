# Project Write Design Gate Evidence

## Scope

本证据对应 `v0.6m / AF-V06-008 Approved Project Write Design Gate`。

本轮只实现只读设计门禁：

- `areaflow run project-write-design-gate <run-id> [--json]`
- `GET /api/v1/runs/{run_id}/project-write-design-gate`
- `ProjectWriteDesignGate` service/build contract

该能力只读取 execution approval gate，并返回 write-set、unsupported operations、copy / verify / repair /
checkpoint 分离、rollback contract 和 first apply sequence。它不创建 command request、lease、attempt、
artifact，不读取或写入被管理项目，不调用 engine，不运行 shell，不解析 secret，不访问网络。

## Safety Facts

设计门禁响应必须保持：

```text
project_write_apply_open=false
project_read_attempted=false
project_write_attempted=false
execution_write_attempted=false
area_flow_artifact_written=false
area_flow_execution_state_written=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
task_claimed=false
worker_started=false
attempt_created=false
artifact_created=false
```

即使 `status=ready`，也只表示设计合同已可查询，不表示真实项目写入已经打开。

## Validation

Focused tests:

```bash
go test ./internal/project ./internal/api ./internal/app
```

Result:

```text
ok  	github.com/areasong/areaflow/internal/project
ok  	github.com/areasong/areaflow/internal/api
ok  	github.com/areasong/areaflow/internal/app
```

Coverage highlights:

- Project build test proves ready design gate keeps `project_write_apply_open=false`.
- Project build test proves blocked execution approval gate blockers propagate.
- API test proves `GET /api/v1/runs/{run_id}/project-write-design-gate` returns write-set fields,
  unsupported operations, apply sequence and read-only safety facts.
- App test proves CLI help, flags and JSON conversion preserve the same contract.

## Not Opened

This evidence does not open:

- fixture approved project write
- managed project write
- AreaMatrix write
- `workflow/versions/**/execution/**` write
- copy / repair / checkpoint apply
- Codex CLI or engine execution
- shell commands
- secret resolution
- network access
