# Completion Audit Evidence

## Scope

This evidence covers AF-V10-006 Completion Audit.

Implemented surface:

- `project.BuildCompletionAudit`
- `Store.CompletionAudit`
- `Store.RecordArchiveProof`
- `Store.LatestArchiveProof`
- `Store.LatestArchiveProofForProject`
- `Store.RecordShimRetirementProof`
- `Store.LatestShimRetirementProof`
- `Store.LatestShimRetirementProofForProject`
- `Store.RecordExecutionCutoverProof`
- `Store.LatestExecutionCutoverProofForProject`
- `Store.RecordValidationProof`
- `Store.LatestValidationProof`
- `Store.LatestValidationProofForProject`
- `Store.RecordSourceAlignmentProof`
- `Store.LatestSourceAlignmentProof`
- `Store.LatestSourceAlignmentProofForProject`
- `Store.RecordTaskMatrixProof`
- `Store.LatestTaskMatrixProof`
- `Store.LatestTaskMatrixProofForProject`
- `Store.RecordSecurityClosureProof`
- `Store.LatestSecurityClosureProof`
- `Store.LatestSecurityClosureProofForProject`
- `Store.RecordBackupRestoreProof`
- `Store.LatestBackupRestoreProof`
- `Store.LatestBackupRestoreProofForProject`
- `Store.RecordReleasePackagingProof`
- `Store.LatestReleasePackagingProof`
- `Store.LatestReleasePackagingProofForProject`
- `Store.RecordOperationsSmokeProof`
- `Store.LatestOperationsSmokeProofForProject`
- `Store.RecordProtectedPathProof`
- `Store.LatestProtectedPathProofForProject`
- `Store.RecordCompletionAuditSnapshot`
- `Store.CompletionAuditSnapshotReadiness`
- `GET /api/v1/completion-audit`
- `GET /api/v1/completion-audit/snapshot-readiness`
- `areaflow completion audit`
- `areaflow completion audit --json`
- `areaflow completion audit-snapshot record <project> --release-candidate <label> --evidence-class fixture|release_candidate`
- `areaflow completion audit-snapshot record <project> --release-candidate <label> --evidence-class release_candidate --evidence-uri <release-candidate-uri> --summary <text> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --json`
- `areaflow completion audit-snapshot readiness <project> --json`
- `areaflow completion archive-proof record <project> --status complete|incomplete|blocked --fact <key> --summary <text> --evidence-uri <release-candidate-uri> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --archive-scope <scope> --archive-reference-mode <mode> --archive-source-path <path> --archive-forbidden-action <action> --archive-rollback-target <target> --archive-fail-closed`
- `areaflow completion archive-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion shim-retirement-proof record <project> --status complete|incomplete|blocked --fact <key> --summary <text> --evidence-uri <release-candidate-uri> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --shim-retirement-scope <scope> --shim-prerequisite <key> --shim-retired-surface <surface> --shim-rollback-target <target> --shim-fail-closed --shim-reopen-requires-approval`
- `areaflow completion shim-retirement-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion execution-cutover-proof record <project> --status complete|incomplete|blocked --fact <key> --summary <text> --evidence-uri <release-candidate-uri> --review-decision approved --reviewed-by <actor> --reviewed-at <RFC3339> --execution-cutover-scope <scope> --allowed-task-type <type> --forbidden-action <action> --rollback-target <target> --rollback-mode <mode>`
- `areaflow completion execution-cutover-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion validation-proof record <project> --status complete|incomplete|blocked --fact <key>`
- `areaflow completion validation-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion source-alignment-proof record <project> --status complete|incomplete|blocked --fact <key>` (complete records automatically bind current AreaFlow E1 source paths and hashes)
- `areaflow completion source-alignment-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion task-matrix-proof record <project> --status complete|incomplete|blocked --fact <key> --source-set-hash <sha256> --backlog-hash <sha256> --task-status-audit-hash <sha256> --planned-v1-required-task-count 0 --missing-evidence-v1-required-task-count 0 --blocked-v1-required-task-count 0`
- `areaflow completion task-matrix-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion security-closure-proof record <project> --status complete|incomplete|blocked --fact <key>` (complete records bind current security/permission/audit state)
- `areaflow completion security-closure-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion backup-restore-proof record <project> --status complete|incomplete|blocked --fact <key>`
- `areaflow completion backup-restore-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion release-packaging-proof record <project> --status complete|incomplete|blocked --fact <key>`
- `areaflow completion release-packaging-proof record <project> --status incomplete --fact <key> --json`
- `areaflow completion protected-path-proof record <project> --status clean|authorized|dirty|blocked`
- `areaflow completion protected-path-proof record <project> --status clean --summary <text> --evidence-uri <uri> --json`
- `scripts/smoke-completion-proof.sh`
- `scripts/smoke-completion-audit-full-proof.sh`
- `scripts/smoke-completion-audit-release-candidate-snapshot.sh`
- `scripts/smoke-execution-cutover-proof.sh`
- `scripts/smoke-validation-proof.sh`
- `scripts/smoke-source-alignment-proof.sh`
- `scripts/smoke-task-matrix-proof.sh`
- `scripts/smoke-security-closure-proof.sh`
- `scripts/smoke-backup-restore-proof.sh`
- `scripts/smoke-release-packaging-proof.sh`
- `scripts/smoke-operations-proof.sh`

The report is read-only. It does not run smoke checks, execute `git status`, write database rows, write project
files, create audit events, create release packages, publish releases, apply restore plans, resolve secrets,
start workers, or issue remote worker credentials.

Completion audit consumes the latest proof record scoped to the target `areamatrix` project. Proofs recorded against
another project key cannot shadow, replace or close E1-E9 for the real AreaMatrix audit.

Completing proof records now require traceable evidence fields at record time. `complete` completion proofs,
operations smoke `pass` proofs, and protected path `clean` / `authorized` proofs must include non-empty `summary`
and `evidence_uri`; otherwise the command fails before opening a database transaction. Missing required facts still
return the existing missing-fact errors first, so focused negative smokes keep their original failure attribution.

Release final gate query failures are not used as completion evidence. If AreaMatrix adapter snapshot files are
missing or unreadable, conformance reports an `adapter_snapshot` fail item with `snapshot_load_failed`; release
readiness and final gate remain blocked instead of turning the completion audit into a query error.

`completion protected-path-proof record` is the separate mutating evidence input. It records an externally checked
AreaMatrix protected path proof in AreaFlow `events`, `audit_events` and `command_requests`; it does not run
`git status`, execute shell commands, write AreaMatrix, read user file contents or touch protected paths.

`completion archive-proof record` is the separate E4 Archive gate evidence input. It records externally reviewed
archive facts in AreaFlow `events`, `audit_events` and `command_requests`. A `complete` proof must also bind the
reviewed archive scope to `areamatrix_historical_execution_reference_only`, the metadata-only reference mode, the
required AreaMatrix source/reference path set, the exact forbidden action set, `archive_rollback_target` and
`archive_fail_closed=true`; the command derives deterministic path/action hashes and `archive_scope_binding_hash`.
It must also carry release-candidate proof evidence URI shape plus approved review metadata: no `local:` /
fixture/script/smoke mechanism URI, no `scripts/**` path, no fixture/mock/demo/sample/synthetic/testdata/placeholder/
dummy/example marker, `review_decision=approved`, non-empty `reviewed_by`, and RFC3339 `reviewed_at`. Missing binding
metadata, missing approved review metadata, old loose proof metadata, or binding mismatch fails closed before the
command opens a database transaction. Completion audit recomputes the current binding hash and also requires the latest Archive proof
`event_id > 0`; hash drift or a missing event ID keeps E4 incomplete. The command does not
copy artifact bytes, delete historical files, rewrite `progress.json`, execute shell commands, write AreaMatrix or
touch protected paths.

`completion shim-retirement-proof record` is the separate E4 Shim Retirement gate evidence input. It records
externally reviewed retirement facts in AreaFlow `events`, `audit_events` and `command_requests`. A `complete`
proof must also bind the reviewed retirement scope to `read_only_shim_retirement_after_execution_forwarding_v1`,
the required prerequisite set, the retired legacy surface set, `shim_rollback_target=read_only_shim`,
`shim_fail_closed=true` and `shim_reopen_requires_approval=true`; the command derives deterministic prerequisite/surface
hashes and `shim_retirement_scope_binding_hash`. It must also carry the same release-candidate URI and approved review
metadata requirements as Archive proof. Missing binding metadata, missing review metadata, old loose proof metadata, or
binding mismatch fails closed before the command opens a database transaction. Completion audit recomputes the current
binding hash and also requires the latest Shim Retirement proof `event_id > 0`; hash drift or a missing event ID keeps E4 incomplete.
The command does not edit AreaMatrix commands, start or disable the legacy runner, write
legacy progress/log/checkpoint state, delete historical files, execute shell commands, write AreaMatrix or touch
protected paths.

`completion execution-cutover-proof record` is the separate E4 Execution Cutover gate evidence input. It records
externally reviewed approval, command response, event/audit, forwarding-window and rollback facts in AreaFlow
`events`, `audit_events` and `command_requests`. A `complete` proof must also bind the reviewed cutover scope to
`execution_forwarding_v1_read_only_evidence_only`, the exact allowed read-only/evidence task type set, the exact
forbidden high-risk action set, `rollback_target=read_only_shim`, `rollback_mode=fail_closed_to_read_only_shim`,
`fail_closed=true` and `reopen_requires_approval=true`. It must also carry the same release-candidate URI and approved
review metadata requirements as Archive proof. Missing binding metadata, missing review metadata, old loose proof
metadata, or any open source-write / retained-generated-write / repair / checkpoint / engine / secret / network /
publish / restore flag fails closed before the command opens a database transaction and remains incomplete in
completion audit. The
record command does not forward `./task-loop run`, write `workflow/versions/**/execution/**`, rewrite old
`progress.json`, write legacy logs/checkpoints, call engines, execute shell commands, write AreaMatrix or touch
protected paths.

`completion validation-proof record` is the separate E3 validation evidence input. It records externally reviewed
command/build/smoke facts and a validation-output binding in AreaFlow `events`, `audit_events` and
`command_requests`. A `complete` proof must include the reviewed validation command list, sha256 result hash,
started/finished RFC3339 time window and validation scope; otherwise it fails before opening a database transaction.
The record command validates those metadata fields but does not run tests, builds, browser smoke, PostgreSQL smoke,
shell commands, write AreaMatrix or touch protected paths.

`completion source-alignment-proof record` is the separate E1 design source alignment evidence input. It records
externally reviewed 0-100% source alignment facts plus a current AreaFlow source binding in AreaFlow `events`,
`audit_events` and `command_requests`. A `complete` proof automatically binds the E1 source path set, per-file
sha256 values, the source-set hash, source file count, and zero missing/unreadable counts. The record command reads
and hashes those AreaFlow docs/task files but does not edit docs, run shell commands, write AreaMatrix or touch
protected paths. Completion audit recomputes the current binding and rejects old loose facts-only proof metadata or
post-proof source drift.

`completion task-matrix-proof record` is the separate E2 phase/task matrix evidence input. It records externally
reviewed backlog and task status audit facts plus a current source binding in AreaFlow `events`, `audit_events` and
`command_requests`. A `complete` proof must bind `tasks/backlog/0-100-platform-backlog.md`,
`docs/development/task-backlog-status-audit.md`, their sha256 values, the source-set hash, and zero counts for
planned / missing-evidence / blocked v1-required tasks. The record command validates those metadata fields but does
not scan or edit backlog/status files, run shell commands, write AreaMatrix or touch protected paths. Completion audit
recomputes the current binding and rejects old loose facts-only proof metadata or post-proof source drift.

`completion security-closure-proof record` is the separate E8 security/permission/isolation evidence input. It
records externally reviewed project isolation, permission doctor, audit coverage and forbidden-capability facts in
AreaFlow `events`, `audit_events` and `command_requests`. A `complete` proof must also bind the current
security boundary readiness, permission doctor and project-scoped audit coverage summary. The record command
collects that binding through read-only Store calls; it does not run shell commands, read secrets, change
authorization, issue remote worker credentials, write AreaMatrix or touch protected paths. Completion audit
recomputes the current binding and rejects old loose facts-only proof metadata, permission/audit/security drift,
or any forbidden v1.0 security capability opening.

`completion backup-restore-proof record` is the separate E6 backup/restore/artifact retention evidence input. It
records externally reviewed backup manifest, restore dry-run, artifact integrity, archive preview and retention
facts in AreaFlow `events`, `audit_events` and `command_requests`; it does not run shell commands, apply restore,
copy/delete/upload artifact bytes, run artifact GC, write AreaMatrix or touch protected paths.

A `complete` backup restore proof must also bind to the current backup manifest hash/status/counts, restore plan
status and matching manifest hash, artifact integrity status/counts, and archive preview status/counts/no-write
safety facts. Missing binding data, manifest hash mismatch, artifact integrity failures, retention policy gaps, or
archive preview write/delete attempts fail closed before the proof command opens a transaction; completion audit
also treats older loose proof metadata as incomplete. Completion audit now re-runs the current read-only
manifest/restore/integrity/archive-preview binding and blocks E6 if stable safety fields drift after the proof is
recorded. Current manifest hashes are exposed and checked for internal consistency, but exact equality to the
pre-proof manifest hash is not a freshness condition because the proof command itself appends AreaFlow
command/event/audit rows. The current archive-preview binding is built from all project artifact metadata and does
not reserve command requests or insert event/audit rows.

`completion release-packaging-proof record` is the separate E5 release/packaging preview evidence input. It records
externally reviewed release final gate, evidence bundle, package preview, distribution preview, publish gate and
rollout preview facts in AreaFlow `events`, `audit_events` and `command_requests`; it does not run shell commands,
create release packages, write release state, create approvals or rollout state, tag/sign/upload/push/publish,
apply migrations, write AreaMatrix or touch protected paths. A `complete` release packaging proof must also bind to
the current `ReleaseEvidenceBundle.BundleHash`, bundle status, mode, item count and ready flag. Completion audit
rechecks that binding against the current bundle, so a stale or metadata-only proof cannot keep E5 complete after
release evidence bundle drift.

`completion audit-snapshot record` is the separate release-candidate closure snapshot input. It can record only when
the current completion audit is already `complete`. It stores audit status, scope, hash, release candidate label,
evidence class, evidence URI and proof event ids in AreaFlow `events`, `audit_events` and `command_requests`; it does
not run tests, run smoke, execute `git status`, write AreaMatrix, create a release package, publish, apply restore,
resolve secrets, start workers or issue remote worker credentials. `evidence_class=fixture` is the default and marks
isolated smoke snapshots as mechanism evidence. `evidence_class=release_candidate` is an audited caller declaration
that requires the real `areamatrix` project identity: root `/Users/as/Ai-Project/project/AreaMatrix`, adapter/profile
`areamatrix`, `kind=product-repo`, and no fixture/temp/mock/demo/sample/synthetic/testdata/placeholder/dummy/example
root markers. It also requires a release-candidate label, evidence URI and summary with none of those non-release
markers and no local/script/smoke mechanism URI, and the evidence URI path must carry `release-candidate` /
`release_candidate` semantics rather than a generic mechanism evidence document. The sealed E1-E9 proof evidence
URIs must follow the same release-candidate evidence URI rule. This URI allowlist remains only a path shape gate: it
checks that the URI path carries `release-candidate` or `release_candidate`; it does not prove the referenced content
was genuinely reviewed. `release_candidate` record therefore also requires strong review metadata:
`review_decision=approved`, non-empty `reviewed_by`, and valid RFC3339 `reviewed_at`; readiness rechecks existing
snapshots and blocks with `completion_audit_snapshot_review_metadata_missing` when the sealed review metadata is absent
or not approved. For `release_candidate` snapshots, the record command also performs a read-only local evidence file
audit: the top-level snapshot evidence URI and sealed E1-E9 proof evidence URIs must resolve to `docs/**/*.md`
inside the AreaFlow checkout, proof URI fragments must match Markdown heading anchors, and the snapshot stores each
resolved evidence file path, anchor, sha256 and size. A complete proof evidence URI map, proof event ID map and
required proof provenance map are required; missing required proof URI keys, generic mechanism proof URIs, proof URIs
with non-release markers, reused proof URI bindings, missing proof event ID keys, reused proof event IDs, missing
provenance, missing evidence files, missing anchors, or an E7 operations proof key marked as fixture/mock/demo/sample/
synthetic/testdata/placeholder/dummy/example are rejected before recording. `manual_ops_smoke_review` is
the reviewed operations proof key example for RC provenance; `v1_stable_fixture_smoke` is reserved for mechanism /
full-proof smoke and cannot seal a release-candidate snapshot. It must also bind to the current ready
`ReleaseEvidenceBundle.BundleHash`; recording writes `release_evidence_bundle_hash`, bundle status, mode, item count,
`release_evidence_bundle_ready`, `proof_evidence_uri_map`, `proof_evidence_uri_count`,
`required_proof_evidence_uri_keys`, `proof_event_ids`, `proof_event_id_count`, `required_proof_event_id_keys`,
`proof_provenance_map`, `required_proof_provenance_keys`, `review_decision`, `reviewed_by`, `reviewed_at`,
`review_metadata_status`, `evidence_uri_file_audit`,
`evidence_uri_file_audit_count` and `evidence_uri_file_audit_status` into snapshot metadata.
The bundle hash includes stable project inventory identity fields such as root, kind, adapter, workflow profile and
default branch, while deliberately excluding mutable DB row counts that the snapshot record command itself changes.
The CLI can validate project identity, URI shape, local evidence file presence, Markdown anchors, evidence file
sha256/size, complete proof URI bindings, mechanism evidence markers, closed safety facts and the current release
evidence bundle binding, but the truth of the referenced external evidence remains an audit responsibility outside the
command. A fixture snapshot proves the snapshot mechanism works; it does not prove the real AreaMatrix release
candidate has reached 100%.
`completion audit-snapshot readiness` is the read-only guardrail for this distinction: a temporary fixture with
`project_key=areamatrix` is blocked by `completion_audit_snapshot_real_project_identity_missing`; a real project whose
latest snapshot is still `evidence_class=fixture` is blocked by `completion_audit_snapshot_fixture_only`; only a
release-candidate snapshot with real project identity, complete v1.0 audit identity, reviewed evidence URI,
non-fixture summary, release-candidate proof URI metadata, complete proof URI / event ID / provenance metadata that
matches the current completion-audit-derived proof URI map, URI set, event ID map and proof provenance map, closed
safety facts, matching current completion audit hash and a current release evidence bundle hash can return ready. If
the real project identity is valid but no snapshot has been recorded yet, readiness exposes
`required_proof_evidence_uri_keys`, `required_proof_event_id_keys`, `required_proof_provenance_keys`, the current
completion audit status/scope/hash and the current release evidence bundle hash/status/mode/item count on the
missing-snapshot item so the release-candidate closure checklist is inspectable before recording. The same
missing-snapshot item also exposes the current completion-audit-derived proof URI map, event ID map, provenance map,
`current_missing_proof_evidence_uri_keys`, `current_missing_proof_event_id_keys`,
`current_missing_proof_provenance_keys`, and current proof binding blockers, so real AreaMatrix readiness reports
which E1-E9 proof bindings are still absent before any release-candidate snapshot record attempt. CLI and API JSON
also expose these blockers through a top-level `gaps[]` view with normalized categories and missing-proof fields, so
automation does not need to scrape ad hoc item metadata. They also expose a top-level `closure` summary with
`ready_for_release_candidate_closure`, `required_evidence_class`, `project_identity`, `snapshot`, `audit_binding`,
`snapshot_evidence`, `proof_evidence_uris`, `proof_event_ids`, `proof_provenance`, `current_proof_binding`,
`release_evidence_bundle`, `evidence_file_audit`, `safety`, `gap_keys` and aggregated blockers. `closure` is a
machine-readable guard summary for the snapshot command's evidence identity and drift checks; it still does not prove
the external release-candidate evidence content was genuinely reviewed. Readiness rejects
release-candidate snapshots that do not seal all required proof event IDs with
`completion_audit_snapshot_proof_event_id_missing`. Readiness recomputes the current `CompletionAudit` hash and
exposes `snapshot_audit_hash`, `current_audit_status`, `current_audit_scope`, `current_audit_hash` and
`audit_hash_match`; audit drift is blocked with `completion_audit_snapshot_audit_hash_mismatch`. Readiness also
compares the sealed `proof_evidence_uri_map`, `proof_evidence_uris`, `proof_event_ids` and `proof_provenance_map`
against the current
completion audit bindings; mismatches are blocked with `completion_audit_snapshot_current_proof_binding_mismatch`.
Readiness also exposes the current `bundle_hash` plus `latest_bundle_hash` / `current_bundle_hash` metadata; bundle hash or bundle
metadata drift is blocked with `completion_audit_snapshot_release_evidence_bundle_mismatch`. That ready state means
the snapshot guard accepted the supplied evidence identity; it is not by itself real AreaMatrix release-candidate
closure. Malformed, fixture-labeled, local script / smoke-labeled, incomplete proof-URI-bound, stale audit-bound,
stale bundle-bound, or side-effect-reporting release-candidate snapshots stay blocked.

## Current Result Semantics

The current report is expected to remain not complete until the full 0-100% evidence exists.

Important current blockers / incomplete areas:

- E3 command/API/smoke evidence now has a durable validation proof input, but full completion still requires
  current, scope-matching proof records for the required command/build/smoke facts.
- E1 design source alignment now has a durable source alignment proof input, but full completion still requires
  current, scope-matching proof records and does not infer source alignment from docs existing.
- E2 phase/task matrix now has a durable task matrix proof input, but full completion still requires current,
  scope-matching proof records with matching backlog/status-audit binding and does not infer task closure from docs existing.
- E8 security/permission/isolation now has a durable security closure proof input, but full completion still
  requires current, scope-matching proof records with matching security/permission/audit binding and refuses to
  close if forbidden security capabilities are open.
- E6 backup/restore/artifact retention now has a durable backup restore proof input, but full completion still
  requires a current, scope-matching proof record; release readiness / restore plan ready cannot close E6 by itself
  and does not infer restore/apply safety from preview surfaces alone.
- E5 release/packaging preview now has a durable release packaging proof input, but full completion still requires
  current release final gate pass, a scope-matching proof record and a matching current release evidence bundle
  binding; the proof does not open package/publish/rollout apply.
- E4 AreaMatrix dogfood has durable Archive, Shim Retirement and Execution Cutover proof inputs. `complete`
  Archive/Shim/Execution Cutover proofs must now carry structured scope/list/rollback binding metadata,
  release-candidate proof evidence URI shape, approved review metadata, deterministic current binding hashes and positive
  event IDs; completion audit rejects older loose proof records, hash drift, missing proof events, local/fixture/script/
  smoke evidence, missing review metadata, or non-real AreaMatrix project identity.
  These proofs can close the audit evidence blockers only inside AreaFlow; they do not execute cutover, write
  AreaMatrix, forward `./task-loop run`, or retire the legacy runner themselves.
- E7 operations readiness now reads the scoped operations readiness report, including support bundle preview,
  migration ledger readiness and `ops.smoke_proof.recorded` proof. The current migrated database path can satisfy
  the full migration ledger requirement through `000011_v1_migration_ledger.sql`; older databases without that
  table still block correctly.
- E8 security/isolation now has PostgreSQL `project_key` isolation smoke evidence, but still requires current
  project-scoped audit coverage and permission doctor binding for full completion.
- E9 protected path proof is `blocked` unless the latest `areamatrix`-scoped clean or authorized proof record exists.
  The API does not run the git protected path command itself.
- E9 `clean` and `authorized` proofs must now carry current protected path binding metadata:
  `protected_path_set_hash`, `protected_path_set_count`, `git_status_output_empty`,
  `git_status_output_hash`, `protected_path_proof_binding_status=pass`, and empty binding blockers. Clean proof uses
  the empty sha256 value `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Completion audit
  rejects older loose proof records with `protected_path_proof_binding_incomplete`.
- E9 `authorized` proof cannot be a free-form assertion. It now requires structured authorization metadata:
  `approval_id`, at least one safe relative protected `allowed_path`, reviewed `git_status_output`, a sha256
  `dirty_output_hash`, `reviewer`, and `rollback_evidence_uri`. The reviewed git status output hash must match
  `dirty_output_hash`, and every parsed touched path must be covered by `allowed_path`.
- Legacy option-only protected path status input has been removed; E9 cannot close without a real proof record.
- Completion audit snapshot recording is available, but it refuses incomplete audits. Snapshot evidence can seal a
  complete audit state; it cannot make an incomplete audit complete.
- Release-candidate snapshot recording and readiness require the current ready release evidence bundle hash. A stale
  `release_evidence_bundle_hash`, missing hash, non-ready bundle, or mismatched bundle metadata keeps readiness
  blocked with `completion_audit_snapshot_release_evidence_bundle_mismatch`.
- Release-candidate snapshot readiness recomputes the current completion audit hash. If the latest snapshot audit hash
  no longer matches, readiness is blocked with `completion_audit_snapshot_audit_hash_mismatch`.
- Release-candidate snapshot readiness rejects local script / smoke wrapper evidence URIs, so a fixture smoke path
  cannot be relabeled into real release-candidate evidence.
- Release-candidate snapshot recording and readiness reject generic mechanism evidence docs as RC closure. The
  snapshot evidence URI and sealed E1-E9 proof URIs must point at release-candidate / real-release-candidate
  evidence paths, not `completion-audit-evidence.md` or `operations-readiness-evidence.md` mechanism notes. This is
  still only a path-shape allowlist for review semantics; the record command separately checks local file existence,
  Markdown anchors, sha256 and size, but that file audit does not prove the referenced evidence was truly reviewed.
- Release-candidate snapshot recording stores `evidence_uri_file_audit` metadata for the top-level snapshot evidence
  URI and sealed E1-E9 proof evidence URIs. Readiness requires release-candidate snapshots to carry a passing
  `evidence_uri_file_audit_status`; missing audit metadata, missing files, missing anchors, sha256 drift or size
  drift stay blocked with
  `completion_audit_snapshot_evidence_uri_file_audit_mismatch`.
- Release-candidate snapshot recording and readiness require complete, distinct E1-E9 proof evidence URI bindings.
  Missing required proof URI keys or reused proof URI bindings keep readiness blocked with
  `completion_audit_snapshot_proof_evidence_uri_missing`.
- Release-candidate snapshot recording and readiness require complete, distinct E1-E9 proof event ID bindings.
  Missing required proof event IDs or reused event IDs keep readiness blocked with
  `completion_audit_snapshot_proof_event_id_missing`, so reviewed URIs cannot replace AreaFlow proof records.
- Release-candidate snapshot recording and readiness require E7 operations proof provenance. `proof_provenance_map`
  must seal `E7_operations_readiness.latest_operations_smoke_proof_key`; `manual_ops_smoke_review` is the reviewed
  ops proof key example, while fixture proof keys such as `v1_stable_fixture_smoke` are only for mechanism /
  full-proof smoke and keep RC closure blocked with `snapshot_operations_proof_key_fixture`.
- Release-candidate snapshot readiness also compares the latest snapshot proof URI map, URI set, event ID map and
  proof provenance map against the current completion audit-derived bindings. If a structurally complete snapshot no
  longer matches the current proof records, readiness is blocked with
  `completion_audit_snapshot_current_proof_binding_mismatch`.
- Release-candidate snapshot recording also rejects completion audits whose sealed E1-E9 proof evidence URIs still
  point at local script, smoke, fixture or `local:` mechanism evidence.
- Release-candidate snapshot recording and readiness require the real AreaMatrix project identity; a temporary
  `project_key=areamatrix` fixture is blocked before release-candidate readiness can become ready.
- Real-identity readiness smoke uses an isolated AreaFlow PostgreSQL database and now compares the full AreaMatrix
  protected path set content fingerprint plus `git status --short` before and after the CLI check. It still does not
  record a snapshot or write AreaMatrix; it only proves that the real project identity path returns the
  `completion_audit_snapshot_missing` gap while protected paths remain unchanged.
- Completing proof records cannot be empty evidence shells: `complete`, operations `pass`, and protected path
  `clean` / `authorized` records require non-empty summary and evidence URI before persistence.

Safety facts stay closed:

```text
read_only=true
release_package_created=false
publish_attempted=false
restore_apply_attempted=false
secret_resolved=false
remote_worker_credentials_issued=false
area_matrix_protected_paths_touched=false
database_write_attempted=false
project_write_attempted=false
smoke_run_attempted=false
worker_started=false
```

## Validation

Focused and full Go validation:

```bash
go test ./...
```

Result on 2026-07-06 CST:

```text
PASS
```

Covered tests:

- `TestBuildCompletionAuditBlocksWithoutProtectedPathProof`
- `TestBuildCompletionAuditRequiresProtectedPathProofRecord`
- `TestBuildCompletionAuditSnapshotRequiresCompleteAudit`
- `TestBuildCompletionAuditSnapshotCapturesHashAndProofIDs`
- `TestBuildCompletionAuditDetectsForbiddenSecurityOpenings`
- `TestBuildCompletionAuditUsesOperationsReadiness`
- `TestBuildCompletionAuditConsumesOperationsSmokeProof`
- `TestBuildCompletionAuditConsumesArchiveProofWithoutCompletingDogfood`
- `TestBuildCompletionAuditRejectsArchiveProofMissingEventID`
- `TestBuildCompletionAuditRejectsTamperedArchiveBindingHash`
- `TestBuildCompletionAuditConsumesShimRetirementProof`
- `TestBuildCompletionAuditRejectsShimRetirementProofMissingEventID`
- `TestBuildCompletionAuditRejectsTamperedShimRetirementBindingHash`
- `TestBuildArchiveProofRequiresAllFactsForComplete`
- `TestBuildShimRetirementProofRequiresAllFactsForComplete`
- `TestBuildCompletionAuditConsumesProtectedPathProof`
- `TestBuildCompletionAuditBlocksDirtyProtectedPathProof`
- `TestBuildProtectedPathProofMarksDirtyOutput`
- `TestCompletionAuditUsesProjectScopedProtectedPathProofWithPostgres`
- `TestCompletionAuditUsesProjectScopedSourceAlignmentProofWithPostgres`
- `TestCompletionAuditUsesProjectScopedRemainingProofsWithPostgres`
- `TestCompletionAuditDoesNotUseGlobalOperationsSmokeProofWithPostgres`
- `TestCompletionAuditDoesNotUseGlobalProofsWhenTargetProjectMissingWithPostgres`
- `TestCompletionAuditEndpoint`
- `TestCompletionAuditToJSON`
- `TestCompletionAuditSnapshotToJSON`
- `TestArchiveProofToJSON`
- `TestShimRetirementProofToJSON`
- `TestProtectedPathProofToJSON`

Focused protected path proof smoke on 2026-07-03 04:23 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  go run ./cmd/areaflow completion protected-path-proof record areamatrix \
    --status clean \
    --summary "AreaMatrix protected path git status returned no output" \
    --evidence-uri local:areamatrix-protected-path-git-status \
    --json

AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  go run ./cmd/areaflow completion audit --json
```

Result:

```text
protected_path_proof_status=complete
E9_areamatrix_protected_path_proof latest_proof_event_id=1
protected_path_proof_binding_status=pass
git_status_output_hash=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
git_status_output_empty=true
project_write_attempted=false
execution_write_attempted=false
git_status_run_by_command=false
area_matrix_protected_paths_touched=false
```

Focused source alignment proof smoke on 2026-07-03 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-source-alignment-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, records complete Source Alignment proof through the
public CLI, verifies the automatically collected current AreaFlow source binding, checks idempotent replay, and then
runs `areaflow completion audit --json`.

Expected result:

```text
source_alignment_proof_status=complete
source_alignment_binding_status=pass
source_alignment_current_binding_bound=true
source_alignment_source_set_hash present
source_alignment_missing_source_count=0
source_alignment_unreadable_source_count=0
source_alignment_gate_passed=true
source_alignment_proof_missing absent
project_write_attempted=false
execution_write_attempted=false
commands_run=false
docs_written=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-source-alignment-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260703100720_60351
applied migrations through 000011_v1_migration_ledger.sql
source alignment proof missing-fact complete request rejected
source alignment proof complete recorded with source binding and idempotent replay returned created=false
completion audit recomputed the current binding, consumed the proof record, and kept the full audit blocked on unrelated incomplete gates
```

The smoke proves the source alignment proof input is durable in PostgreSQL, bound to the current AreaFlow source set,
and consumable by completion audit. It does not itself prove that the current docs have been re-reviewed for a release
candidate.

Focused task matrix proof smoke on 2026-07-03 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-task-matrix-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, records complete Task Matrix proof through the public
CLI, checks idempotent replay, and then runs `areaflow completion audit --json`.

Expected result:

```text
task_matrix_proof_status=complete
task_matrix_gate_passed=true
task_matrix_binding_status=pass
task_matrix_current_binding_bound=true
planned_v1_required_task_count=0
missing_evidence_v1_required_task_count=0
blocked_v1_required_task_count=0
task_matrix_proof_missing absent
task_matrix_status=complete
project_write_attempted=false
execution_write_attempted=false
commands_run=false
docs_written=false
tasks_written=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-task-matrix-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260703102228_83826
applied migrations through 000011_v1_migration_ledger.sql
task matrix proof missing-fact complete request rejected
task matrix proof complete request without binding rejected
task matrix proof complete recorded and idempotent replay returned created=false
completion audit consumed the proof record, matched the current backlog/status-audit binding and kept the full audit
blocked on unrelated incomplete gates
```

The smoke proves the task matrix proof input is durable in PostgreSQL, rejects old facts-only completion metadata, and
is consumable only when completion audit can match the current backlog/status-audit binding. It does not itself prove
that the current backlog and task status audit have been re-reviewed for a release candidate.

Focused security closure proof smoke on 2026-07-03 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-security-closure-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, records complete Security Closure proof through the
public CLI, checks idempotent replay, and then runs `areaflow completion audit --json`.

Expected result:

```text
security_closure_proof_status=complete
security_closure_gate_passed=true
project_isolation_smoke_missing absent
audit_gap_closure_missing absent
project_write_attempted=false
execution_write_attempted=false
authorization_changed=false
secret_plaintext_read=false
remote_worker_credentials_issued=false
commands_run=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-security-closure-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260703104321_26161
applied migrations through 000011_v1_migration_ledger.sql
security closure proof missing-fact complete request rejected
security closure proof complete recorded and idempotent replay returned created=false
completion audit consumed the proof record and kept the full audit blocked on unrelated incomplete gates
```

The smoke proves the security closure proof input is durable in PostgreSQL and consumable by completion audit. It
now also proves `complete` E8 proof metadata is bound to current read-only security boundary readiness, permission
doctor and project-scoped audit coverage. It still does not run shell commands, run project isolation smoke, write
AreaMatrix or by itself establish real release-candidate evidence.

Focused release packaging proof smoke on 2026-07-03 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-release-packaging-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, records complete Release Packaging proof through the
public CLI, checks idempotent replay, and then runs `areaflow completion audit --json`.

Expected result:

```text
release_packaging_proof_status=complete
release_packaging_proof_recorded=true
release_final_gate_not_passed present
release_packaging_gate_passed absent
project_write_attempted=false
execution_write_attempted=false
release_package_created=false
release_state_written=false
release_approval_created=false
rollout_state_created=false
migration_apply_attempted=false
tag_created=false
package_signed=false
artifact_uploaded=false
git_push_attempted=false
publish_attempted=false
commands_run=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-release-packaging-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260706062919_54428
applied migrations through 000011_v1_migration_ledger.sql
release packaging proof missing-fact complete request rejected
release packaging proof complete recorded and idempotent replay returned created=false
completion audit consumed the proof record, set release_packaging_proof_recorded=true, and kept E5 incomplete until
the current release final gate passes
```

The smoke proves the release packaging proof input is durable in PostgreSQL and consumable by completion audit. It
does not itself rerun or satisfy release final gate, evidence bundle, package preview, distribution preview,
publish gate or rollout preview checks for a release candidate.

Focused backup restore proof smoke on 2026-07-06 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-backup-restore-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, seeds local and `project_reference` artifact metadata,
collects `backup manifest`, `backup restore-plan`, `artifact integrity`, and `artifact archive-preview` JSON output,
records complete Backup Restore proof through the public CLI with those output bindings, checks idempotent replay,
and then runs `areaflow completion audit --json`.

Expected result:

```text
backup_restore_proof_status=complete
backup_restore_gate_passed=true
backup_restore_evidence_binding_status=pass
backup_manifest_hash=<sha256>
restore_plan_manifest_hash=<same sha256>
artifact_integrity_failed_count=0
artifact_archive_preview_external_refs=1
artifact_archive_preview_needs_policy=0
restore_dry_run_needs_attention absent
metadata_only_history_not_closed absent
project_write_attempted=false
execution_write_attempted=false
database_restore_attempted=false
artifact_bytes_copied=false
artifact_bytes_deleted=false
artifact_bytes_uploaded=false
artifact_gc_attempted=false
commands_run=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-backup-restore-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260706230712_67141
applied migrations through 000011_v1_migration_ledger.sql
backup restore proof missing-fact complete request rejected
backup restore proof missing-output-binding complete request rejected
backup manifest / restore plan / artifact integrity / archive preview outputs collected and bound
backup restore proof complete recorded and idempotent replay returned created=false
completion audit consumed the proof record and kept the full audit blocked on unrelated incomplete gates
completion audit did not add command_requests/events/audit_events rows while revalidating E6
completion audit blocked E6 after local artifact content drift changed the current integrity binding
```

The smoke proves the backup restore proof input is durable in PostgreSQL, bound to current read-only
backup/restore/artifact preview outputs, consumable by completion audit, and revalidated against current
backup/restore/artifact state after proof recording without adding command/event/audit rows. It still does not
execute restore apply, artifact copy/archive/delete/GC apply, shell commands, or real AreaMatrix writes.

Focused operations proof smoke on 2026-07-06 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-operations-proof.sh
```

This smoke creates a temporary `areamatrix` fixture project, verifies operations readiness initially reports
`fresh_local_ops_smoke_missing`, records a `local_ops_smoke` proof through the public CLI, checks idempotent
replay, and then runs both `areaflow ops readiness --json` and `areaflow completion audit --json`.

Expected result:

```text
operations_status=ready
latest_smoke_proof_key=local_ops_smoke
fresh_local_ops_smoke_missing absent
full_migration_ledger_missing absent
E7_operations_readiness message = operations readiness evidence is complete for v1.0 scope
record_command_runs_smoke=false
project_write_attempted=false
execution_write_attempted=false
service_process_control_attempted=false
support_bundle_exported=false
migration_apply_attempted=false
remote_telemetry_enabled=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-operations-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260706031856_50924
applied migrations through 000011_v1_migration_ledger.sql
operations readiness initially reported fresh_local_ops_smoke_missing
local_ops_smoke proof recorded and idempotent replay returned created=false
operations readiness returned status=ready and completion audit E7 consumed the proof
completion audit stayed blocked on unrelated real 100% gates
```

The smoke proves the E7 operations proof input is durable in PostgreSQL and consumable by completion audit. It does
not itself run the long local/Web smoke chain or prove a real release-candidate operations review.

Focused E4 archive / shim retirement / execution cutover proof smoke on 2026-07-07 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-completion-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, records complete Archive, Shim Retirement and
Execution Cutover proof through the public CLI using release-candidate-shaped proof evidence URIs and approved review
metadata, checks idempotent replay, and then runs
`areaflow completion audit --json`.

Expected result:

```text
archive_proof_status=complete
shim_retirement_proof_status=complete
execution_cutover_proof_status=complete
execution_cutover_scope_binding_status=pass
execution_cutover_scope=execution_forwarding_v1_read_only_evidence_only
execution_cutover_rollback_target=read_only_shim
execution_cutover_rollback_mode=fail_closed_to_read_only_shim
latest_archive_proof_event_id>0
archive_scope_binding_hash=<sha256>
archive_scope_current_binding_bound=true
latest_shim_retirement_proof_event_id>0
shim_retirement_scope_binding_hash=<sha256>
shim_retirement_scope_current_binding_bound=true
archive_proof_review_metadata_status=approved
shim_retirement_proof_review_metadata_status=approved
execution_cutover_proof_review_metadata_status=approved
status=blocked
project_root_not_real_areamatrix present
execution_cutover_not_complete present
real_areamatrix_archive_not_proven present
real_areamatrix_shim_retirement_not_proven present
project_write_attempted=false
execution_write_attempted=false
task_loop_run_forwarded_by_command=false
commands_run=false
legacy_progress_written=false
legacy_logs_written=false
legacy_checkpoint_written=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-completion-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260707030331_92822
applied migrations through 000011_v1_migration_ledger.sql
archive proof missing-fact complete request rejected
archive proof complete recorded and idempotent replay returned created=false
shim retirement proof missing-fact complete request rejected
shim retirement proof complete recorded and idempotent replay returned created=false
execution cutover proof missing-fact complete request rejected
execution cutover proof loose complete request without scope binding rejected
execution cutover proof complete recorded with execution_forwarding_v1_read_only_evidence_only scope binding and idempotent replay returned created=false
completion audit consumed all three E4 proof records, exposed binding/review metadata, and stayed blocked on real AreaMatrix identity
```

Focused execution cutover proof wrapper also passed:

```text
make smoke-docker-execution-cutover-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260707025932_81292
completion audit consumed archive, shim and execution cutover proof metadata for project key
areamatrix-execution-cutover-proof-fixture
```

The smoke proves the proof inputs are durable in PostgreSQL and consumable by completion audit. It does not prove
AreaMatrix execution cutover apply, does not forward `./task-loop run`, does not land the compatibility shim, and
does not retire the legacy runner. A fixture project root cannot remove the real AreaMatrix dogfood blockers.

Focused full completion audit proof smoke on 2026-07-03 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-completion-audit-full-proof.sh
```

This smoke creates a temporary `areamatrix` project, records complete proof inputs for E1-E9, records an operations
smoke proof, records a clean protected path proof, requires `areaflow completion audit --json` to stay top-level
`status=blocked` because the project root is not the real AreaMatrix identity, and then proves both release-candidate
and fixture snapshot recording fail closed while the audit is not complete.

Latest run:

```text
make smoke-docker-completion-audit-full-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260707030223_89556
applied migrations through 000011_v1_migration_ledger.sql
seeded fixture-only release final gate inputs: local artifact metadata, audit coverage events, safe AreaMatrix-like
conformance files and a safe project config baseline
recorded Source Alignment, Task Matrix, Validation, Archive, Shim Retirement, Execution Cutover with scope binding,
Release Packaging, Backup Restore, Security Closure, Operations Smoke and Protected Path proofs
completion audit returned top-level status=blocked, exposed complete E1/E2/E3/E5/E6/E7/E8/E9 proof bindings, and kept
E4 blocked by real AreaMatrix identity
protected path proof output and completion audit metadata exposed protected_path_proof_binding_status=pass,
git_status_output_hash=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855,
git_status_output_empty=true, protected_path_set_hash, protected_path_set_count=7 and empty binding blockers
completion audit returned task_matrix_status=complete and implementation_gap_status=complete
release_candidate snapshot record was rejected because the temporary fixture did not match the real AreaMatrix
project identity
fixture snapshot record was rejected because completion audit status was blocked rather than complete
completion audit snapshot readiness returned status=blocked with completion_audit_snapshot_real_project_identity_missing
current default skips real AreaMatrix fingerprint reads; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 for that optional guard
the isolated PostgreSQL database was dropped after the smoke
```

Focused project-scope regression also passed:

```text
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  go test ./internal/project -run 'TestCompletionAudit(UsesProjectScoped(ProtectedPath|SourceAlignment|Remaining)Proofs?WithPostgres|DoesNotUseGlobalOperationsSmokeProofWithPostgres|DoesNotUseGlobalProofsWhenTargetProjectMissingWithPostgres)' -count=1
isolated PostgreSQL database: areaflow_it_2f2ed1ae01274dac9913cbad0e449385
an `areamatrix` clean protected path proof was recorded first
an unrelated project dirty protected path proof was recorded later and became global latest
completion audit still consumed the `areamatrix`-scoped proof for E9 and returned protected_path_proof_status=complete
an `areamatrix` complete source alignment proof was recorded first
an unrelated project blocked source alignment proof was recorded later and became global latest
completion audit still consumed the `areamatrix`-scoped proof for E1 and returned E1 complete
complete AreaMatrix Task Matrix, Validation, Archive, Shim Retirement, Execution Cutover, Release Packaging,
Backup Restore and Security Closure proofs were recorded first
blocked proofs for the same proof types were then recorded on an unrelated project and became global latest
completion audit still consumed only the AreaMatrix-scoped E2/E3/E4/E5/E6/E8 proof metadata and left those items complete
an unrelated project passing operations smoke proof became global latest
ordinary operations readiness still consumed the global latest proof for backward compatibility
completion audit E7 still reported `fresh_local_ops_smoke_missing` and did not leak the other project proof URI
when `areamatrix` was absent, unrelated source/protected/operations proofs did not close or leak into E1/E7/E9
```

This smoke proves the completion audit can consume a full current evidence set while refusing to treat an isolated
fixture database as real AreaMatrix closure. It is still not proof that real AreaMatrix execution cutover apply,
legacy runner retirement, release publish, restore apply, support export, remote telemetry or managed ops have been opened.

Focused release candidate snapshot smoke on 2026-07-06 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-completion-audit-release-candidate-snapshot.sh
```

This smoke creates a temporary `areamatrix` project, records complete E1-E9 proof inputs using synthetic reviewed
evidence URIs rather than local script / smoke / fixture URIs, verifies completion audit stays `blocked` because the
project root is not the real `/Users/as/Ai-Project/project/AreaMatrix` identity, and then verifies that
`evidence_class=release_candidate` snapshot recording fails closed because the current audit is not complete. Set
`AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1` to additionally fingerprint the real AreaMatrix
`.areaflow/status.json` and `workflow/README.md` before and after the negative smoke.

Latest run:

```text
make smoke-docker-completion-audit-release-candidate-snapshot PASS
isolated PostgreSQL database: areaflow_smoke_20260707030252_90788
applied migrations through 000011_v1_migration_ledger.sql
seeded fixture-only release final gate inputs so proof records were complete, while the current audit stayed blocked by identity
recorded E1-E9 proof inputs with synthetic reviewed docs/development/real-release-candidate-evidence.md evidence URIs
completion audit returned top-level status=blocked without local:, fixture: or scripts/ evidence URI markers
release_candidate snapshot record was rejected because the current audit status was blocked
completion audit snapshot readiness returned status=blocked with completion_audit_snapshot_real_project_identity_missing
default run skipped optional real AreaMatrix fingerprint reads; set AREAFLOW_SMOKE_CHECK_REAL_AREAMATRIX=1 for that guard
the isolated PostgreSQL database was dropped after the smoke
```

This smoke proves fixture evidence cannot be relabeled as a release candidate even when E1-E9 proof URIs look
reviewed and the evidence records are complete in an isolated database. The release-candidate URI allowlist
used here is only a path shape gate; real release-candidate snapshot recording also requires local `docs/**/*.md`
evidence file audit metadata, and readiness blocks when that metadata is missing or drifted. That file audit still
does not prove content review by itself. This smoke is a negative identity gate proof, not real AreaMatrix cutover,
release-candidate closure, release publish, restore apply, support export, remote telemetry or managed ops evidence.

Focused real identity fixture snapshot smoke on 2026-07-07 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-completion-audit-real-identity-fixture-snapshot.sh
```

This smoke registers and imports the real `/Users/as/Ai-Project/project/AreaMatrix` identity into an isolated
AreaFlow DB, seeds only an AreaFlow DB fixture completion-audit snapshot event, and then queries
`completion audit-snapshot readiness --json` through the public CLI. The expected result is `status=blocked`,
`has_snapshot=true`, `completion_audit_snapshot_fixture_only`, `closure.snapshot.status=fixture_only` and
`ready_for_release_candidate_closure=false`. The seeded fixture snapshot also carries machine-readable
`latest.metadata.fixture_only_does_not_prove_release_readiness=true` and `latest.metadata.does_not_prove` values for
`real_100_percent`, `release_candidate_closure`, `release_candidate_readiness`,
`real_identity_does_not_prove_package_a_apply` and `real_identity_does_not_prove_real_100_percent`, and the smoke
asserts those values are visible in readiness JSON. It also fingerprints the real AreaMatrix `.areaflow/status.json`,
`workflow/README.md` and protected path set before and after the smoke.

This proves that a real project identity alone cannot turn fixture snapshot evidence into release-candidate
closure. The smoke writes AreaFlow DB state only; it does not record real release-candidate evidence, does not apply
Package A, does not update `.areaflow/status.json`, and does not prove real 100%.

Focused validation proof smoke on 2026-07-06 CST:

```bash
AREAFLOW_DATABASE_URL=postgres://areaflow:areaflow@localhost:54329/<temp-db>?sslmode=disable \
  bash scripts/smoke-validation-proof.sh
```

This smoke creates a temporary AreaMatrix-like project root, records complete Validation proof through the public
CLI, checks idempotent replay, and then runs `areaflow completion audit --json`.

Expected result:

```text
validation_proof_status=complete
validation_gate_passed=true
validation_evidence_binding_status=pass
validation_result_hash=<sha256>
validation_scope=fixture_validation_review
validation_command_count=9
fresh_validation_proof_missing absent
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
commands_run=false
smoke_run_attempted=false
web_build_run_by_command=false
area_matrix_protected_paths_touched=false
```

Latest run:

```text
make smoke-docker-validation-proof PASS
isolated PostgreSQL database: areaflow_smoke_20260706223007_21791
applied migrations through 000011_v1_migration_ledger.sql
validation proof missing-fact complete request rejected
validation proof complete recorded with command list, sha256 result hash, RFC3339 time window and scope
idempotent replay returned created=false
completion audit consumed the proof record and exposed validation binding metadata while keeping the full audit
blocked on unrelated incomplete gates
```

The smoke proves the validation proof input is durable in PostgreSQL, requires structured validation-output binding
for `complete`, and is consumable by completion audit. It still does not itself prove that the latest full validation
suite has been run for a real release candidate; that remains external review evidence tied to the recorded binding.

## Boundary

This moves AF-V10-006 to `implemented_scoped`: the completion audit API/CLI exists, refuses to claim 100% without
E1-E9 proof, and now has an isolated full-proof smoke proving fixture project identity keeps the audit blocked and
rejects snapshot recording instead of masquerading as real AreaMatrix closure.

The current 0-100% goal remains active until the real release candidate evidence set, not only the isolated fixture
proof set, proves every required AreaFlow platform capability and boundary.
