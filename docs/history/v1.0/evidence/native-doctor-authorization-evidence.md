# Native Doctor Authorization Evidence

## Purpose

本文记录 backlog 任务
[`AF-V02-002 Native Doctor Authorization Boundary`](../plans/task-backlog.md#af-v02-002-native-doctor-authorization-boundary)
的最近一次本机验证证据。

该证据覆盖 AreaMatrix native workflow doctor 的授权边界：

- 默认不执行 native command，只记录 skipped/warn。
- `--allow-native` 只能作为一次性人工授权覆盖 `run_commands` capability。
- `--allow-native` 不能绕过 command allowlist。
- `--allow-native` 不能绕过 forbidden command deny。
- `./task-loop run`、`git reset --hard`、`git checkout --`、`rm -rf` 必须保持 forbidden。

## Run

Date: 2026-07-01

Baseline commands:

```bash
go test ./internal/doctor
go test ./internal/project
```

Result: pass

New unit evidence:

- `TestAreaMatrixDoctorAllowNativeDoesNotBypassCommandAllowlist`
- `TestAreaMatrixDoctorAllowNativeDoesNotBypassForbiddenCommand`

These tests use a runner that fails the test if the native command is called.

## Real AreaMatrix Read-only Smoke

Environment:

```text
PostgreSQL: docker compose service areaflow-postgres, postgres:16-alpine, localhost:54329
Read-only database: af_v02auth_1782889977_86485
Project key: areamatrix
Project root: /Users/as/Ai-Project/project/AreaMatrix
Project config: examples/areamatrix/areaflow.yaml
```

Commands:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/af_v02auth_1782889977_86485?sslmode=disable \
  ./scripts/smoke-areamatrix-readonly.sh
go run ./cmd/areaflow project doctor areamatrix --json
dropdb -h localhost -p 54329 -U areaflow --if-exists af_v02auth_1782889977_86485
psql -h localhost -p 54329 -U areaflow -d postgres -Atc \
  "SELECT count(*) FROM pg_database WHERE datname LIKE 'af_v02auth_%';"
```

Observed smoke output:

```text
smoke-areamatrix-readonly: project doctor --json
smoke-areamatrix-readonly: project summary --json
smoke-areamatrix-readonly: project readiness --json
smoke-areamatrix-readonly: project import-diff --json
smoke-areamatrix-readonly: project verify-bundle --json
smoke-areamatrix-readonly: pass areamatrix root=/Users/as/Ai-Project/project/AreaMatrix
```

Capability row:

```text
deny
```

`run_commands` capability is denied for the real AreaMatrix config.

Command policy rows:

```text
./dev tasks status=allow
./dev workflow doctor=allow
./task-loop run=deny
git checkout --=deny
git reset --hard=deny
rm -rf=deny
```

Permission probes:

```text
./dev workflow doctor|false|true|false
./dev tasks status|false|true|false
./task-loop run|false|false|true
git reset --hard|false|false|true
git checkout --|false|false|true
rm -rf|false|false|true
./dev workflow init --version v2|false|false|false
```

字段含义：

- `command|capability_allowed|command_allowed|denied`
- `./dev workflow doctor` and `./dev tasks status` are listed commands, but default execution is blocked by
  `run_commands=false`.
- `./task-loop run`、`git reset --hard`、`git checkout --` and `rm -rf` are denied by forbidden command policy.
- `./dev workflow init --version v2` is not in the allowlist and is not executable.

Doctor native check probe:

```text
warn|run_commands capability not allowed|false
```

字段含义：

- native workflow doctor check status: `warn`
- skip reason: `run_commands capability not allowed`
- `allow_native`: `false`

Cleanup query:

```text
0
```

`af_v02auth_%` 临时数据库已清理，无残留。

## Evidence

- Default `project doctor` does not execute native AreaMatrix workflow doctor when `run_commands=false`.
- The native doctor result stays `warn`; it is not collapsed into `pass`.
- The real AreaMatrix project config keeps `run_commands` denied.
- `./dev workflow doctor` is allowlisted for explicit human native verification only.
- `./task-loop run` stays denied before execution cutover.
- Destructive git/shell commands stay denied.
- Unknown commands stay non-executable because they are not allowlisted.
- Unit tests prove `--allow-native` cannot bypass command allowlist or forbidden command deny.
- The real AreaMatrix smoke did not run `--allow-native` and did not modify AreaMatrix.
- The temporary PostgreSQL database was dropped after the smoke.

## Boundary

这份证据不证明：

- Real native AreaMatrix workflow doctor execution.
- Authorization to run `./dev workflow doctor` on real AreaMatrix.
- Any permission to run `./task-loop run`.
- Runner / worker execution.
- Project file writes.
- Web/Desktop behavior.
- Secret, restore, publish or plugin capabilities.

这些仍属于独立 backlog 项和后续 gate。
