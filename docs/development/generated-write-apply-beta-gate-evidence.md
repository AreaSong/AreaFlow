# Generated Write Apply Beta Gate Evidence

Date: 2026-07-02

## Scope

This focused evidence covers v0.6r generated write apply beta gate.

It verifies:

- `BuildGeneratedWriteApplyBetaGate` blocks when generated write readiness is not ready.
- The gate remains blocked with `approval_status=needs_approval` even when readiness is ready.
- The gate nests `generated-write-readiness` so CLI/API/Web/Desktop can show the previous readiness result.
- Required evidence includes focused smoke, explicit R3 approval, target expected-before hash/size and rollback verification.
- REST API exposes `GET /api/v1/projects/{project}/generated-write-apply-beta-gate`.
- CLI exposes `areaflow project generated-write-apply-beta-gate <project> [--json]`.
- API/CLI JSON preserve `approval_required=true`, `approval_status=needs_approval`, `apply_open=false`,
  `real_areamatrix_write_opened=false`, blockers and read-only safety facts.

## Validation

```bash
go test ./internal/project -run 'GeneratedWriteApplyBetaGate|GeneratedWriteReadiness' -count=1
go test ./internal/api -run 'GeneratedWrite' -count=1
go test ./internal/app -run 'GeneratedWrite|Help' -count=1
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
go test ./internal/project ./internal/api ./internal/app passed
go test ./... passed
go build ./cmd/areaflow passed
git diff --check -- . passed
```

## API / CLI Surfacing

REST API:

```text
GET /api/v1/projects/{project}/generated-write-apply-beta-gate
```

CLI:

```text
areaflow project generated-write-apply-beta-gate <project>
areaflow project generated-write-apply-beta-gate <project> --json
```

## Required Evidence

The gate reports these requirements before real AreaMatrix generated-only apply beta can open:

```text
generated-write-readiness ready_for_review=true
make smoke-docker-managed-generated-write passes
make smoke-docker-areamatrix-readonly passes after config review
explicit R3 approval for real AreaMatrix generated-only apply beta
single existing regular generated target path selected
expected-before sha256 and size captured for the target path
preimage artifact and rollback verification plan reviewed
non-target AreaMatrix fingerprints remain unchanged
```

## Safety Facts

The gate is project-scoped and read-only. It must keep:

```text
approval_required=true
approval_status=needs_approval
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

## Not Opened

This evidence does not open:

- Real AreaMatrix writes.
- Generated-only apply beta on real AreaMatrix.
- Approval record creation.
- Command request creation.
- Queue/run/task/lease/attempt/artifact/event/audit creation.
- Persisted generated apply result.
- Source write.
- Checkpoint.
- Repair.
- Engine calls.
- Shell commands.
- Secret resolution.
- Network access.
- `workflow/versions/**/execution/**` writes.
