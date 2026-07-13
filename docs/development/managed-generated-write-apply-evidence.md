# Managed Generated Write Apply Evidence

Date: 2026-07-02

## Scope

This focused evidence covers v0.6p managed generated write apply core service plus v0.6 API/CLI surfacing.

It verifies:

- `QueueManagedGeneratedWrite` / `WriteManagedGenerated` core service contracts compile.
- REST API exposes managed generated write queue/apply without bypassing the service layer.
- CLI exposes managed generated write queue/apply without bypassing the service layer.
- Default worker capabilities are `read_project`, `write_artifacts` and `write_generated`.
- Generated-only target paths are limited to `.areaflow/generated/**` and `.areamatrix/generated/**`.
- Project config maps generated-only `write_paths` to `write_generated` path allow rules only when
  `write_generated: true`.
- Real AreaMatrix project keys/kinds remain denied by fixture/temp project policy.
- Command responses expose generated-only, rollback and safety facts.
- The core path records write-set/preimage/report artifact intent and copy/verify/rollback attempt intent.
- `scripts/smoke-managed-generated-write.sh` verifies the full CLI/PG path and denial path.

## Validation

```bash
go test ./internal/project
go test ./internal/api
go test ./internal/app
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable \
  bash scripts/smoke-managed-generated-write.sh
go test ./...
go build ./cmd/areaflow
git diff --check
```

Result:

```text
ok github.com/areasong/areaflow/internal/project
ok github.com/areasong/areaflow/internal/api
ok github.com/areasong/areaflow/internal/app
smoke-managed-generated-write: PASS
go test ./... passed
go build ./cmd/areaflow passed
git diff --check passed
```

The smoke uses a temporary fixture project for the allowed path and a non-fixture product project for denial.
By default it does not read real AreaMatrix projection files. Set
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` only when the caller intentionally wants the extra
`.areaflow/status.json` / `workflow/README.md` fingerprint guard.

## API / CLI Surfacing

REST API:

```text
POST /api/v1/projects/{project}/workflow-versions/{version}/managed-generated-write-queue
POST /api/v1/projects/{project}/workers/{worker}/managed-generated-write
```

CLI:

```text
areaflow run managed-generated-write-queue <project> <version> --target-path <path> --content <text> --expected-before-sha256 <hash> --expected-before-size <size>
areaflow worker managed-generated-write <project> <worker-key> --run-id <id>
```

The API and CLI surface only the same fixture/temp generated-only rollback drill. They do not add a second
authorization path, do not bypass execution approval gate, and do not open real AreaMatrix writes.

## Safety Facts

The service/API/CLI path is constrained to fixture/temp project generated-only write/verify/rollback drills:

```text
managed_generated_write=true
generated_only=true
generated_only_apply_open=true
fixture_or_temp_project_only=true
real_areamatrix_write_opened=false
project_read_attempted=true
project_read_allowed=true
project_write_attempted=true
project_write_allowed=true
execution_write_attempted=false
area_flow_artifact_written=true
area_flow_execution_state_written=true
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
write_set_passed=true
verification_passed=true
rollback_attempted=true
rollback_verified=true
```

## Not Opened

This evidence does not open:

- Real AreaMatrix writes.
- Persisted generated apply result.
- Managed-project source write.
- Checkpoint.
- Repair.
- Engine calls.
- Shell commands.
- Secret resolution.
- Network access.
- `workflow/versions/**/execution/**` writes.
