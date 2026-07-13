# Execution Cutover Readiness Evidence

Date: 2026-07-02

## Scope

This focused evidence covers the AreaMatrix execution cutover readiness bundle.

It verifies:

- CLI exposes `areaflow project execution-cutover-readiness <project> [--json]`.
- REST API exposes `GET /api/v1/projects/{project}/execution-cutover-readiness`.
- The readiness bundle is read-only and uses mode `read_only_areamatrix_execution_cutover_readiness`.
- It aggregates evidence for import/mirror/shadow, authoring cutover, compatibility shim, worker lease lifecycle,
  run control, fixture execution, read-only verify, approved artifact write, fixture project write and
  fixture/temp managed generated write.
- It remains blocked until compatibility shim, real AreaMatrix generated apply, copy/repair/checkpoint and
  explicit execution cutover approval are satisfied.
- Safety facts preserve `execution_cutover_apply_open=false`, `project_write_attempted=false`,
  `execution_write_attempted=false`, `task_loop_run_forwarded=false`, `engine_call_attempted=false`,
  `commands_run=false`, `secrets_resolved=false` and `network_used=false`.

## API / CLI Surfacing

REST API:

```text
GET /api/v1/projects/{project}/execution-cutover-readiness
```

CLI:

```text
areaflow project execution-cutover-readiness <project>
areaflow project execution-cutover-readiness <project> --json
```

Web / Desktop observation:

```text
Web dashboard:
  Execution Cutover
  AreaMatrix Readiness

Desktop shell:
  Execution Cutover
```

Both surfaces only read:

```text
GET /api/v1/projects/{project}/execution-cutover-readiness
```

They do not create command requests, approvals, workers, engine calls, project writes, execution writes or
`./task-loop run` forwarding.

## Validation

Focused checks:

```bash
go test ./internal/project -run 'ExecutionCutover|GeneratedWrite|ManagedGenerated|ProjectWriteDesign|ExecutionPlan' -count=1
go test ./internal/api -run 'ExecutionCutover|GeneratedWrite|ManagedGenerated' -count=1
go test ./internal/app -run 'ExecutionCutover|GeneratedWrite|ManagedGenerated|Help' -count=1
bash -n scripts/smoke-local.sh scripts/smoke-v1-stable-fixture.sh scripts/smoke-compatibility-fixture.sh
```

Result:

```text
ok github.com/areasong/areaflow/internal/project
ok github.com/areasong/areaflow/internal/api
ok github.com/areasong/areaflow/internal/app
bash syntax check PASS
```

The long safe fixture path is covered by
[`v1-stable-fixture-evidence.md`](./v1-stable-fixture-evidence.md), where `scripts/smoke-local.sh` asserts
`execution-cutover-readiness` remains blocked and preserves no-write / no-execution safety facts.

Latest Web observation smoke on 2026-07-02 21:32 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-web.sh
```

Result:

```text
smoke-web: ok
```

Temporary database prefix: `af_web_execution_cutover_20260702213221_*`; cleanup residual database count was 0.
The smoke checker asserted the dashboard requested
`/api/v1/projects/{project}/execution-cutover-readiness`, rendered `Execution Cutover`,
`AreaMatrix Readiness`, `explicit_execution_cutover_approval`, `execution_cutover_apply=false` and
`task_loop_run_forwarded=false`, and still failed on any non-GET `/api/v1` request.

## Safety Facts

Execution cutover readiness is a go/no-go view, not an apply command. It must keep:

```text
read_only=true
execution_cutover_apply_open=false
project_write_attempted=false
execution_write_attempted=false
task_loop_run_forwarded=false
engine_call_attempted=false
commands_run=false
secrets_resolved=false
network_used=false
worker_scheduled=false
approval_created=false
retained_generated_apply_open=false
source_write_open=false
checkpoint_apply_open=false
repair_apply_open=false
```

## Not Opened

This evidence does not open:

- Real AreaMatrix compatibility shim file edits.
- Real AreaMatrix generated-only retained apply.
- Source write.
- Copy / repair / checkpoint apply.
- `./task-loop run` forwarding.
- Codex CLI or engine execution.
- Secret resolution.
- Shell command execution.
- Network access.
- Real AreaMatrix execution cutover.
