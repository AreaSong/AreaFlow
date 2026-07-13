# Operations Readiness Evidence

## Scope

This evidence covers the scoped v1.0 operations readiness surface.

Implemented surface:

- `project.BuildSupportBundlePreview`
- `project.BuildMigrationLedgerReadiness`
- `project.BuildOperationsReadiness`
- `Store.SupportBundlePreview`
- `Store.MigrationLedgerReadiness`
- `Store.OperationsReadiness`
- `Store.RecordOperationsSmokeProof`
- `Store.LatestOperationsSmokeProof`
- `GET /api/v1/ops/readiness`
- `GET /api/v1/ops/support-bundle-preview`
- `GET /api/v1/ops/migration-ledger-readiness`
- `areaflow ops readiness`
- `areaflow ops readiness --json`
- `areaflow ops migration-ledger-readiness`
- `areaflow ops migration-ledger-readiness --json`
- `areaflow ops smoke-proof record <project> --key <proof-key>`
- `areaflow ops smoke-proof record <project> --key <proof-key> --json`
- `areaflow support bundle-preview`
- `areaflow support bundle-preview --json`
- `scripts/smoke-operations-proof.sh`
- Web `Operations / Readiness` panel
- Desktop `Operations Readiness` panel

The reports are read-only. They do not run smoke checks, start or stop services, apply migrations, create
`migration_ledger`, export support bundles, upload telemetry, read secrets, copy project files, read prompt text,
read raw artifact content, read user file content, write database rows, write project files, or touch AreaMatrix
protected paths.

`ops smoke-proof record` is the only mutating operation in this scope. It records a proof that an external smoke
already passed by writing AreaFlow `events`, `audit_events` and `command_requests`; it does not run smoke, start
or stop services, apply migrations, export support bundles, upload telemetry, write managed project files or touch
AreaMatrix protected paths.

## Current Result Semantics

`operations readiness` is expected to remain `needs_attention` until fresh install/migrate/start/register smoke
proof and migration ledger phase evidence exist. On a database with `000011_v1_migration_ledger.sql` applied, the
full ledger table and preflight/apply/verify/remediation phase proof can satisfy the migration ledger item.

Current scoped improvements:

- support bundle preview is metadata-only and redacted by design.
- operations readiness now fail-closes if support bundle preview opens export/upload/DB write, includes secret,
  prompt, user-file, raw artifact or unredacted log content, drops required exclusions, or lacks required forbidden
  actions.
- migration ledger readiness reads embedded migrations, `schema_migrations` and `migration_ledger` phases, but
  does not apply migrations.
- operations readiness aggregates service status, support bundle preview, migration ledger readiness, local-only
  telemetry and managed ops deferral.
- completion audit E7 now reads operations readiness instead of only local service status.
- Web and Desktop render the operations readiness summary as read-only observation surfaces.
- `scripts/smoke-local.sh` now exercises `areaflow ops migration-ledger-readiness --json`,
  `areaflow support bundle-preview --json` and `areaflow ops readiness --json` during the v0.1-v1.0 long smoke.
- The same long smoke also exercises `areaflow desktop service-control-gate --json`,
  `areaflow desktop notification-gate --json` and `areaflow desktop tray-menu-gate --json`, proving desktop
  control surfaces remain read-only while operations readiness keeps service process control, OS notification and
  native tray/menu integration closed.
- `scripts/smoke-local.sh` and `scripts/smoke-web.sh` record operations smoke proof only after their checks pass;
  a later `areaflow ops readiness --json` can consume that proof and remove the
  `fresh_local_ops_smoke_missing` blocker. The proof is intentionally time-bounded: operations readiness records
  `latest_smoke_proof_freshness_status`, `latest_smoke_proof_age_seconds` and
  `smoke_proof_max_age_seconds=86400`, and a proof older than 24 hours remains evidence history but no longer
  closes E7.
- `scripts/smoke-operations-proof.sh` is the focused PostgreSQL smoke for this E7 path. It creates a temporary
  AreaMatrix-like project, proves readiness starts with `fresh_local_ops_smoke_missing`, records a
  `local_ops_smoke` proof, and then checks both `ops readiness` and completion audit E7 consume the proof while
  no project write, service control, support export, migration apply, telemetry upload or AreaMatrix protected
  path touch is attempted. The focused smoke also asserts the consumed proof is fresh and bound by the 24-hour
  max-age guard.

Still not complete for v1.0:

- operations readiness now has durable smoke proof and migration ledger proof inputs, but completion audit still
  remains incomplete while other E1-E9 requirements are missing.
- operations smoke proof is not permanent completion evidence. It must be refreshed within the 24-hour freshness
  window before E7 can complete.
- `000011_v1_migration_ledger.sql` creates the preflight/apply/verify/remediation ledger. Older databases that
  have not applied it still correctly report `full_migration_ledger_missing`.
- support bundle export remains v1.x.
- remote telemetry remains disabled.
- remote ops, managed upgrade and destructive rollback remain v1.x.

## Required Closed Facts

Support bundle preview exposes these facts:

```text
read_only=true
metadata_only=true
export_open=false
secret_values_included=false
api_token_values_included=false
prompt_text_included=false
user_file_contents_included=false
raw_artifact_contents_included=false
unredacted_logs_included=false
managed_project_files_copied=false
area_matrix_protected_paths_touched=false
remote_upload_attempted=false
database_write_attempted=false
```

Migration ledger readiness exposes these facts:

```text
read_only=true
migration_apply_attempted=false
database_write_attempted=false
destructive_rollback_attempted=false
project_write_attempted=false
area_matrix_protected_paths_touched=false
```

Operations readiness exposes these facts:

```text
read_only=true
support_bundle_exported=false
support_bundle_metadata_only=true
remote_telemetry_enabled=false
managed_upgrade_attempted=false
destructive_rollback_attempted=false
service_process_control_attempted=false
database_write_attempted=false
project_write_attempted=false
area_matrix_protected_paths_touched=false
```

Operations smoke proof records expose these facts:

```text
record_command_runs_smoke=false
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
service_process_control_attempted=false
support_bundle_exported=false
migration_apply_attempted=false
remote_telemetry_enabled=false
area_matrix_protected_paths_touched=false
```

## Validation

Focused validation:

```bash
go test ./internal/project
go test ./internal/api
go test ./internal/app
bash -n scripts/smoke-docker.sh scripts/smoke-local.sh scripts/smoke-v1-stable-fixture.sh scripts/smoke-web.sh scripts/smoke-operations-proof.sh
make smoke-docker-operations-proof
make smoke-docker-v1-stable-fixture
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-web.sh
cd web && npm run build
cd desktop && npm run build
node --check scripts/smoke-web-check.mjs
```

Result on 2026-07-04 CST:

```text
PASS
```

Covered tests:

- `TestBuildSupportBundlePreviewIsMetadataOnly`
- `TestBuildOperationsReadinessBlocksUnsafeSupportBundlePreview`
- `TestBuildMigrationLedgerReadinessNeedsFullLedger`
- `TestBuildOperationsReadinessAggregatesScopedOps`
- `TestBuildOperationsReadinessConsumesSmokeProof`
- `TestBuildCompletionAuditUsesOperationsReadiness`
- `TestBuildCompletionAuditBlocksUnsafeSupportBundleReadiness`
- `TestBuildCompletionAuditConsumesOperationsSmokeProof`
- `TestSupportBundlePreviewEndpoint`
- `TestMigrationLedgerReadinessEndpoint`
- `TestOperationsReadinessEndpoint`
- `TestSupportBundlePreviewToJSON`
- `TestMigrationLedgerReadinessToJSON`
- `TestOperationsReadinessToJSON`
- `TestOperationsSmokeProofToJSON`

Focused smoke coverage:

- `make smoke-docker-operations-proof` passed on 2026-07-06 03:18 CST against isolated PostgreSQL database
  `areaflow_smoke_20260706031856_50924`, created and dropped by `scripts/smoke-docker.sh`.
- The operations proof smoke applied migrations from an empty database, registered a temporary `areamatrix`
  fixture project, verified E7 initially reported `fresh_local_ops_smoke_missing`, recorded `local_ops_smoke`,
  verified idempotent replay returned `created=false`, and confirmed completion audit E7 consumed the proof while
  the full audit stayed blocked on unrelated real 100% gates.
- `make smoke-docker-v1-stable-fixture` passed on 2026-07-04 14:23 CST against isolated PostgreSQL database
  `areaflow_smoke_20260704142302_79088`, created and dropped by `scripts/smoke-docker.sh`.
- The v1 stable fixture smoke applied migrations from an empty database, registered/imported the fixture project,
  ran the long `smoke-local.sh` chain and passed the ops CLI assertions.
- The smoke asserted `areaflow desktop service-control-gate --json` keeps start/stop/restart blocked and keeps
  process control, command creation, approval creation, audit writes, worker scheduling, workflow execution,
  project writes and secret resolution closed.
- The smoke asserted `areaflow desktop notification-gate --json` and `areaflow desktop tray-menu-gate --json`
  keep OS notification permission, event stream opening, native tray/menu integration, service control,
  notification request, command creation, approval creation, audit writes, worker scheduling, workflow execution,
  project writes and secret resolution closed.
- The smoke asserted support bundle preview remains metadata-only, `export_open=false`, sensitive/raw content is
  excluded, and support export / project copy / database write actions remain forbidden.
- The focused operations proof smoke now asserts completion audit does not add `command_requests`, `events` or
  `audit_events` rows while consuming E7 readiness, and exposes support bundle redaction facts such as
  `support_bundle_metadata_only=true`, `support_bundle_export_open=false`, sensitive content flags false, and the
  required sensitive exclusion count.
- The smoke asserted migration ledger readiness reads applied `schema_migrations`, sees
  `full_ledger_table_present=true`, includes `000011_v1_migration_ledger.sql`, and keeps migration apply,
  rollback, database write and project write closed.
- The smoke asserted operations readiness reports local-only telemetry, support export / managed ops deferred,
  no service process control, no remote telemetry and no AreaMatrix protected path touch.
- The smoke records `v1_stable_fixture_smoke` proof after the long chain passes, then reruns operations readiness
  and verifies the `fresh_local_ops_smoke_missing` and `full_migration_ledger_missing` blockers are gone.
- The smoke then runs `areaflow completion audit --json` and verifies completion audit also consumes the proof
  without running smoke itself.

Focused frontend coverage:

- Web build proves the dashboard can compile with `GET /api/v1/ops/readiness` types and panel rendering.
- Desktop build proves the local shell can compile with `GET /api/v1/ops/readiness` panel rendering.
- `scripts/smoke-web-check.mjs` now expects the Web dashboard to request `/api/v1/ops/readiness` and render
  `read_only_operations_readiness`, `install_migrate_start_register_smoke`,
  `metadata_only_support_bundle_preview`, `migration_ledger_readiness`, `support_export=deferred_v1x` and
  `telemetry=local_only`.
- `scripts/smoke-web.sh` passed on 2026-07-03 04:12 CST against a temporary PostgreSQL database. The run started
  the AreaFlow API and Vite dashboard, then the browser checker verified the operations readiness panel.
- The Web smoke records `web_dashboard_ops_smoke` proof after the browser check passes and verifies operations
  readiness can read the latest proof.

## Boundary

This evidence moves operations readiness to `implemented_scoped`: the v1.0 read-only API/CLI exists and is
test-covered and is now included in the long fixture and browser smoke paths, but it is not a claim that AreaFlow
has reached 100%.

It is not evidence for full support export, remote telemetry, managed upgrade, destructive rollback, restore apply,
release publish, AreaMatrix execution cutover, AreaMatrix protected path authorization, or completion audit
`complete`.
