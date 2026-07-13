# Managed Generated Write Gate Evidence

Date: 2026-07-02

## Scope

This focused evidence covers v0.6o managed generated write gate.

It verifies:

- `BuildManagedGeneratedWriteGate` ready/blocked behavior.
- Execution approval gate blockers propagate into the generated write gate.
- Generated-only prefixes are exposed.
- Required capabilities are `read_project`, `write_artifacts` and `write_generated`.
- Required write-set fields include `generated_only` and rollback evidence fields.
- Source write, workflow execution write, checkpoint, repair and destructive operations remain blocked.
- REST API response contract.
- CLI help, flags, JSON mapping and text output path compile.

## Validation

```bash
go test ./internal/project ./internal/api ./internal/app
go test ./...
go build ./cmd/areaflow
git diff --check
```

Result:

```text
ok github.com/areasong/areaflow/internal/project
ok github.com/areasong/areaflow/internal/api
ok github.com/areasong/areaflow/internal/app
go test ./... passed
go build ./cmd/areaflow passed
git diff --check passed
```

## Safety Facts

The gate is read-only. It must keep:

```text
generated_only_write_ready=true|false
generated_only_apply_open=false
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

## Not Opened

This evidence does not open:

- Real AreaMatrix writes.
- Managed-project generated-only apply.
- Source-write `write_code` permission for generated-only writes.
- Managed-project source write.
- Checkpoint.
- Repair.
- Engine calls.
- Shell commands.
- Secret resolution.
- Network access.
- `workflow/versions/**/execution/**` writes.
