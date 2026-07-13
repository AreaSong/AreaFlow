# Fixture Project Write Evidence

Date: 2026-07-02

## Scope

This focused evidence covers v0.6n fixture-only approved project write implementation.

It verifies:

- `run.fixture_project_write_queue` command response safety facts.
- `worker.fixture_project_write` command response safety facts.
- Fixture write option normalization.
- Fixture write path safety checks for root escape, absolute path, missing file, directory, and symlink.
- REST API queue/apply contracts.
- CLI help, flag parsing, JSON mapping, and focused app contract coverage.

## Validation

```bash
go test ./internal/project ./internal/api ./internal/app
```

Result:

```text
ok github.com/areasong/areaflow/internal/project
ok github.com/areasong/areaflow/internal/api
ok github.com/areasong/areaflow/internal/app
```

## Safety Facts

The fixture project write path is intentionally narrower than managed project write:

- It only allows projects whose key or kind contains `fixture`.
- It requires worker capabilities `read_project`, `write_artifacts`, and `write_code`.
- It requires project `write_artifacts` capability.
- It requires target path allowlist for both `read_project` and `write_code`.
- It only modifies an existing ordinary file under project root.
- It rejects directory, missing file, symlink, absolute path, and `..` root escape targets.
- It records expected-before hash/size, preimage artifact, copy attempt, verify attempt, rollback attempt, and report artifact.
- It restores the target to preimage hash/size before commit.

The focused tests assert:

```text
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
rollback_verified=true
```

## Not Opened

This evidence does not open:

- Real AreaMatrix writes.
- Managed-project generated-only write.
- Managed-project source write.
- Checkpoint.
- Repair.
- Engine calls.
- Shell commands.
- Secret resolution.
- Network access.
- `workflow/versions/**/execution/**` writes.
- create/delete/move/chmod/binary/symlink/glob/root escape writes.
