# Generated Write Readiness Evidence

Date: 2026-07-02

## Scope

This focused evidence covers v0.6q generated write dogfood readiness.

It verifies:

- `BuildGeneratedWriteReadiness` blocks the current AreaMatrix baseline because `write_generated` and
  generated path allowlist are not open.
- A prepared generated-only config can return `ready_for_review=true` while still keeping `apply_open=false`.
- High-risk capabilities such as `run_commands` block generated-only dogfood review.
- REST API exposes `GET /api/v1/projects/{project}/generated-write-readiness`.
- CLI exposes `areaflow project generated-write-readiness <project> [--json]`.
- API/CLI JSON preserve `ready_for_review`, `apply_open=false`, `real_areamatrix_write_opened=false`,
  blockers and read-only safety facts.

## Validation

```bash
go test ./internal/project ./internal/api ./internal/app
go test ./...
go build ./cmd/areaflow
git diff --check -- .
```

Result:

```text
ok github.com/areasong/areaflow/internal/project
ok github.com/areasong/areaflow/internal/api
ok github.com/areasong/areaflow/internal/app
go test ./... passed
go build ./cmd/areaflow passed
git diff --check -- . passed
```

## API / CLI Surfacing

REST API:

```text
GET /api/v1/projects/{project}/generated-write-readiness
```

CLI:

```text
areaflow project generated-write-readiness <project>
areaflow project generated-write-readiness <project> --json
```

## Safety Facts

The readiness is project-scoped and read-only. It must keep:

```text
apply_open=false
real_areamatrix_write_opened=false
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
lease_created=false
attempt_created=false
artifact_created=false
```

`ready_for_review=true` only means project config and permission preconditions are ready for human review.
It does not open apply while `apply_open=false`.

## Not Opened

This evidence does not open:

- Real AreaMatrix writes.
- Generated-only apply on real AreaMatrix.
- Queue/run/task/lease/attempt/artifact creation.
- Persisted generated apply result.
- Source write.
- Checkpoint.
- Repair.
- Engine calls.
- Shell commands.
- Secret resolution.
- Network access.
- `workflow/versions/**/execution/**` writes.
