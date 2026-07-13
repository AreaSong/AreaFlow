package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/areasong/areaflow/internal/migrate"
)

func TestStoreProjectKeyIsolationWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL project isolation integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("project-isolation-%d", time.Now().UnixNano())
	projectAKey := fixtureKey + "-a"
	projectBKey := fixtureKey + "-b"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, projectAKey, projectBKey)
	})

	projectA := insertIsolationProject(ctx, t, pool, projectAKey)
	projectB := insertIsolationProject(ctx, t, pool, projectBKey)
	versionA := insertIsolationWorkflowVersion(ctx, t, pool, projectA.ID, fixtureKey, "shared-v1")
	versionB := insertIsolationWorkflowVersion(ctx, t, pool, projectB.ID, fixtureKey, "shared-v1")
	runA, taskA := insertIsolationRunAndTask(ctx, t, pool, projectA.ID, versionA.ID, fixtureKey)
	runB, taskB := insertIsolationRunAndTask(ctx, t, pool, projectB.ID, versionB.ID, fixtureKey)
	workerA, leaseA := insertIsolationWorkerAndLease(ctx, t, pool, projectA.ID, runA, taskA, fixtureKey)
	workerB, leaseB := insertIsolationWorkerAndLease(ctx, t, pool, projectB.ID, runB, taskB, fixtureKey)
	insertIsolationArtifact(ctx, t, pool, projectA.ID, versionA.ID, runA, fixtureKey, "artifact-a")
	insertIsolationArtifact(ctx, t, pool, projectB.ID, versionB.ID, runB, fixtureKey, "artifact-b")
	insertIsolationEventAndAudit(ctx, t, pool, projectA.ID, versionA.ID, runA, fixtureKey, "a")
	insertIsolationEventAndAudit(ctx, t, pool, projectB.ID, versionB.ID, runB, fixtureKey, "b")

	assertWorkflowVersionsIsolated(ctx, t, store, projectA, versionA.ID)
	assertWorkflowVersionsIsolated(ctx, t, store, projectB, versionB.ID)
	assertRunsIsolated(ctx, t, store, projectA, versionA, runA)
	assertRunsIsolated(ctx, t, store, projectB, versionB, runB)
	assertArtifactsIsolated(ctx, t, store, projectA, versionA, "artifact-a")
	assertArtifactsIsolated(ctx, t, store, projectB, versionB, "artifact-b")
	assertEventsAndAuditIsolated(ctx, t, store, projectA.ID)
	assertEventsAndAuditIsolated(ctx, t, store, projectB.ID)
	assertWorkersIsolated(ctx, t, store, projectA, workerA.ID)
	assertWorkersIsolated(ctx, t, store, projectB, workerB.ID)

	recovered, err := store.RecoverExpiredLeases(ctx, projectA, RecoverLeasesOptions{
		Limit:          10,
		Actor:          "integration-test",
		Reason:         "prove project_key lease isolation",
		Metadata:       map[string]any{"fixture_key": fixtureKey, "isolation_check": true},
		IdempotencyKey: "lease.recover:" + fixtureKey + ":a",
	})
	if err != nil {
		t.Fatalf("recover expired leases for project A: %v", err)
	}
	if len(recovered) != 1 || recovered[0].ID != leaseA || recovered[0].ProjectID != projectA.ID {
		t.Fatalf("recovered leases leaked or missed project A lease: %+v, want lease %d project %d", recovered, leaseA, projectA.ID)
	}
	assertLeaseStatus(ctx, t, pool, leaseA, "needs_recovery")
	assertLeaseStatus(ctx, t, pool, leaseB, "active")
}

func TestApplyStatusProjectionBlocksNonExactApprovalReasonBeforeWriterWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL status projection apply integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("status-projection-non-exact-reason-%d", time.Now().UnixNano())
	sourceHash := "source-" + fixtureKey
	idempotencyKey := "project.status_projection.apply:" + fixtureKey
	applyReason := "integration " + fixtureKey + " non-exact approval reason blocks before writer"

	record, createdProject := ensureIsolationProject(ctx, t, pool, completionAuditTargetProjectKey)
	record.RootPath = completionAuditTargetProjectRoot
	if createdProject {
		insertStatusProjectionApplyPermissionFixture(ctx, t, pool, record.ID)
	}
	insertStatusProjectionApplyImportSnapshotFixture(ctx, t, pool, record.ID, sourceHash, fixtureKey)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if createdProject {
			cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, completionAuditTargetProjectKey)
			return
		}
		cleanupStatusProjectionApplyIntegrationFixture(cleanupCtx, t, pool, record.ID, sourceHash, idempotencyKey)
	})

	preview, err := store.StatusProjectionAuthorizationPreview(ctx, record, StatusProjectionAuthorizationPreviewOptions{
		TargetURI: ".areaflow/status.json",
	})
	if err != nil {
		t.Fatalf("build status projection authorization preview: %v", err)
	}
	expectedExists := preview.Preimage.Exists
	expectedSize := preview.Preimage.SizeBytes
	writerCalled := false

	result, err := store.ApplyStatusProjection(ctx, record, ApplyStatusProjectionOptions{
		TargetURI:      ".areaflow/status.json",
		IdempotencyKey: idempotencyKey,
		Actor:          "integration-test",
		Reason:         applyReason,
		Writer: func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
			writerCalled = true
			return StatusProjectionWriteResult{}, errors.New("writer must not be called for non-exact Package A approval reason")
		},
		Gate: StatusProjectionApplyGateOptions{
			TargetURI:                      ".areaflow/status.json",
			ExpectedBeforeExists:           &expectedExists,
			ExpectedBeforeSHA256:           preview.Preimage.SHA256,
			ExpectedBeforeSizeBytes:        &expectedSize,
			SourceHash:                     sourceHash,
			SchemaURI:                      preview.SchemaURI,
			ValidatorPreflight:             preview.ValidatorPreflight,
			ProtectedPathCheck:             statusProjectionProtectedPathCheck(record),
			ProtectedPathFingerprintSHA256: preview.ProtectedPathFingerprintSHA256,
			RollbackAction:                 statusProjectionRollbackAction(preview.Preimage),
			AcceptedPreimageSchemaStatus:   preview.Preimage.SchemaStatus,
			ExplicitApproval:               true,
			ApprovalActor:                  "integration-test",
			ApprovalReason:                 "approve status projection apply",
		},
	})
	if err != nil {
		t.Fatalf("apply status projection: %v", err)
	}
	if writerCalled {
		t.Fatalf("writer must not be called when Package A approval reason is non-exact")
	}
	if result.Status != "blocked" || result.Decision != "denied" || result.ProjectWriteAttempted || result.ApplyCommandEligible {
		t.Fatalf("unexpected blocked apply result: %+v", result)
	}
	if result.ApplyGateStatus != "blocked" || result.ApplyGateDecision != "no_go" {
		t.Fatalf("expected apply gate to block before writer: %+v", result)
	}
	if !containsString(result.Blockers, "approval_reason_missing_or_mismatch") {
		t.Fatalf("expected exact approval reason blocker: %+v", result.Blockers)
	}

	var responseDecision string
	var responseProjectWriteAttempted bool
	if err := pool.QueryRow(ctx, `
SELECT response->>'decision',
       COALESCE((response->>'project_write_attempted')::boolean, false)
FROM command_requests
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		record.ID,
		statusProjectionApplyCommandType,
		idempotencyKey,
	).Scan(&responseDecision, &responseProjectWriteAttempted); err != nil {
		t.Fatalf("load status projection command response: %v", err)
	}
	if responseDecision != "denied" || responseProjectWriteAttempted {
		t.Fatalf("unexpected command response decision=%q project_write_attempted=%t", responseDecision, responseProjectWriteAttempted)
	}
}

func TestCompletionAuditUsesProjectScopedProtectedPathProofWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL completion audit project proof integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("completion-audit-project-proof-%d", time.Now().UnixNano())
	targetProofKey := fmt.Sprintf("completion.audit.project-scoped:%s:areamatrix", fixtureKey)
	otherProofKey := fmt.Sprintf("completion.audit.project-scoped:%s:other", fixtureKey)
	otherProjectKey := fixtureKey + "-other"

	target, createdTarget := ensureIsolationProject(ctx, t, pool, completionAuditTargetProjectKey)
	other := insertIsolationProject(ctx, t, pool, otherProjectKey)
	if createdTarget {
		target, err = store.UpsertFromConfig(ctx, securityClosureBindingFixtureConfig(completionAuditTargetProjectKey, fixtureKey))
		if err != nil {
			t.Fatalf("seed areamatrix security closure config: %v", err)
		}
		seedSecurityClosureAuditCoverageFixture(ctx, t, pool, target.ID, fixtureKey)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupCompletionAuditProtectedPathProofFixture(cleanupCtx, t, pool, fixtureKey, targetProofKey, otherProofKey)
		cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, otherProjectKey)
		if createdTarget {
			cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, completionAuditTargetProjectKey)
		}
	})

	targetEvidenceURI := "local:" + fixtureKey + ":areamatrix-clean"
	if _, err := store.RecordProtectedPathProof(ctx, target, RecordProtectedPathProofOptions{
		ProofStatus:    "clean",
		Summary:        "AreaMatrix protected paths clean for completion audit",
		EvidenceURI:    targetEvidenceURI,
		IdempotencyKey: targetProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record areamatrix protected path proof: %v", err)
	}
	otherEvidenceURI := "local:" + fixtureKey + ":other-dirty"
	if _, err := store.RecordProtectedPathProof(ctx, other, RecordProtectedPathProofOptions{
		ProofStatus:     "dirty",
		Summary:         "Other project protected path proof must not affect AreaMatrix completion audit",
		EvidenceURI:     otherEvidenceURI,
		GitStatusOutput: " M workflow/README.md",
		IdempotencyKey:  otherProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record other project protected path proof: %v", err)
	}

	globalLatest, err := store.LatestProtectedPathProof(ctx)
	if err != nil {
		t.Fatalf("load global latest protected path proof: %v", err)
	}
	if globalLatest.Project.Key != otherProjectKey || metadataString(globalLatest.Metadata, "evidence_uri") != otherEvidenceURI {
		t.Fatalf("test setup expected other project to be global latest proof: %+v", globalLatest)
	}

	audit, err := store.CompletionAudit(ctx, CompletionAuditOptions{GeneratedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("build completion audit: %v", err)
	}
	if audit.ProtectedPathProofStatus != "complete" {
		t.Fatalf("protected path aggregate status = %q, want complete", audit.ProtectedPathProofStatus)
	}
	item := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("E9 should consume AreaMatrix project proof, not global latest proof: %+v", item)
	}
	if item.Metadata["latest_proof_project_key"] != completionAuditTargetProjectKey ||
		item.Metadata["latest_proof_evidence_uri"] != targetEvidenceURI ||
		item.Metadata["area_matrix_protected_paths_touched"] != false {
		t.Fatalf("E9 metadata did not use AreaMatrix proof: %+v", item.Metadata)
	}
	if containsString(item.BlockedBy, "protected_path_proof_project_mismatch") ||
		containsString(item.BlockedBy, "protected_path_proof_not_clean") {
		t.Fatalf("E9 should not inherit other project blockers: %+v", item)
	}
}

func TestCompletionAuditUsesProjectScopedSourceAlignmentProofWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL completion audit project source alignment proof integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("completion-audit-source-alignment-%d", time.Now().UnixNano())
	targetProofKey := fmt.Sprintf("completion.audit.source-alignment:%s:areamatrix", fixtureKey)
	otherProofKey := fmt.Sprintf("completion.audit.source-alignment:%s:other", fixtureKey)
	otherProjectKey := fixtureKey + "-other"

	target, createdTarget := ensureIsolationProject(ctx, t, pool, completionAuditTargetProjectKey)
	other := insertIsolationProject(ctx, t, pool, otherProjectKey)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, sourceAlignmentProofEventType, sourceAlignmentProofCommandType, targetProofKey, otherProofKey)
		cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, otherProjectKey)
		if createdTarget {
			cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, completionAuditTargetProjectKey)
		}
	})

	targetEvidenceURI := "local:" + fixtureKey + ":areamatrix-source-alignment"
	sourceBinding, err := SourceAlignmentCurrentBinding()
	if err != nil {
		t.Fatalf("source alignment current binding: %v", err)
	}
	if _, err := store.RecordSourceAlignmentProof(ctx, target, RecordSourceAlignmentProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSourceAlignmentProofFacts,
		Summary:                "AreaMatrix source alignment proof for completion audit",
		EvidenceURI:            targetEvidenceURI,
		SourceAlignmentBinding: sourceBinding,
		IdempotencyKey:         targetProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record areamatrix source alignment proof: %v", err)
	}
	otherEvidenceURI := "local:" + fixtureKey + ":other-source-alignment"
	if _, err := store.RecordSourceAlignmentProof(ctx, other, RecordSourceAlignmentProofOptions{
		ProofStatus:    "blocked",
		Summary:        "Other project source alignment proof must not affect AreaMatrix completion audit",
		EvidenceURI:    otherEvidenceURI,
		IdempotencyKey: otherProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record other project source alignment proof: %v", err)
	}

	globalLatest, err := store.LatestSourceAlignmentProof(ctx)
	if err != nil {
		t.Fatalf("load global latest source alignment proof: %v", err)
	}
	if globalLatest.Project.Key != otherProjectKey || metadataString(globalLatest.Metadata, "evidence_uri") != otherEvidenceURI {
		t.Fatalf("test setup expected other project to be global latest source alignment proof: %+v", globalLatest)
	}

	audit, err := store.CompletionAudit(ctx, CompletionAuditOptions{GeneratedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("build completion audit: %v", err)
	}
	item := findCompletionAuditItem(t, audit, "E1_design_source_alignment")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("E1 should consume AreaMatrix project proof, not global latest proof: %+v", item)
	}
	if item.Metadata["latest_source_alignment_proof_project_key"] != completionAuditTargetProjectKey ||
		item.Metadata["latest_source_alignment_proof_evidence_uri"] != targetEvidenceURI ||
		item.Metadata["source_alignment_gate_passed"] != true {
		t.Fatalf("E1 metadata did not use AreaMatrix proof: %+v", item.Metadata)
	}
	if containsString(item.BlockedBy, "source_alignment_proof_project_mismatch") ||
		containsString(item.BlockedBy, "source_alignment_proof_blocked") {
		t.Fatalf("E1 should not inherit other project blockers: %+v", item)
	}
}

func TestCompletionAuditDoesNotUseGlobalOperationsSmokeProofWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL completion audit operations smoke project proof integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("completion-audit-ops-proof-%d", time.Now().UnixNano())
	otherProofKey := fmt.Sprintf("completion.audit.ops:%s:other", fixtureKey)
	otherProjectKey := fixtureKey + "-other"

	_, createdTarget := ensureIsolationProject(ctx, t, pool, completionAuditTargetProjectKey)
	other := insertIsolationProject(ctx, t, pool, otherProjectKey)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, operationsSmokeProofEventType, operationsSmokeProofCommandType, otherProofKey)
		cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, otherProjectKey)
		if createdTarget {
			cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, completionAuditTargetProjectKey)
		}
	})

	otherEvidenceURI := "local:" + fixtureKey + ":other-ops-smoke"
	if _, err := store.RecordOperationsSmokeProof(ctx, other, RecordOperationsSmokeProofOptions{
		ProofKey:       "local_ops_smoke",
		EvidenceStatus: "pass",
		Summary:        "Other project operations smoke proof must not affect AreaMatrix completion audit",
		EvidenceURI:    otherEvidenceURI,
		IdempotencyKey: otherProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record other project operations smoke proof: %v", err)
	}

	globalLatest, err := store.LatestOperationsSmokeProof(ctx)
	if err != nil {
		t.Fatalf("load global latest operations smoke proof: %v", err)
	}
	if globalLatest.Project.Key != otherProjectKey || metadataString(globalLatest.Metadata, "evidence_uri") != otherEvidenceURI {
		t.Fatalf("test setup expected other project to be global latest operations smoke proof: %+v", globalLatest)
	}

	globalReadiness, err := store.OperationsReadiness(ctx, OperationsReadinessOptions{GeneratedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("build global operations readiness: %v", err)
	}
	globalItem := findOperationsItem(t, globalReadiness, "install_migrate_start_register_smoke")
	if globalItem.Metadata["latest_smoke_proof_project_key"] != otherProjectKey ||
		globalItem.Metadata["latest_smoke_proof_uri"] != otherEvidenceURI {
		t.Fatalf("ordinary operations readiness should still consume global latest proof: %+v", globalItem.Metadata)
	}

	audit, err := store.CompletionAudit(ctx, CompletionAuditOptions{GeneratedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("build completion audit: %v", err)
	}
	item := findCompletionAuditItem(t, audit, "E7_operations_readiness")
	if !containsString(item.BlockedBy, "fresh_local_ops_smoke_missing") {
		t.Fatalf("E7 should not consume other project global operations smoke proof: %+v", item)
	}
	if item.Metadata["latest_operations_smoke_proof_evidence_uri"] == otherEvidenceURI ||
		item.Metadata["latest_operations_smoke_proof_event_id"] == globalLatest.EventID {
		t.Fatalf("E7 metadata leaked other project operations smoke proof: %+v", item.Metadata)
	}
}

func TestCompletionAuditDoesNotUseGlobalProofsWhenTargetProjectMissingWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL completion audit missing target project integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}
	if projectExists(ctx, t, pool, completionAuditTargetProjectKey) {
		t.Skip("target areamatrix project already exists; missing-target regression requires an isolated database")
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("completion-audit-missing-target-%d", time.Now().UnixNano())
	sourceProofKey := fmt.Sprintf("completion.audit.missing-target:%s:source", fixtureKey)
	protectedProofKey := fmt.Sprintf("completion.audit.missing-target:%s:protected", fixtureKey)
	opsProofKey := fmt.Sprintf("completion.audit.missing-target:%s:ops", fixtureKey)
	otherProjectKey := fixtureKey + "-other"

	other := insertIsolationProject(ctx, t, pool, otherProjectKey)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, sourceAlignmentProofEventType, sourceAlignmentProofCommandType, sourceProofKey)
		cleanupCompletionAuditProtectedPathProofFixture(cleanupCtx, t, pool, fixtureKey, protectedProofKey)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, operationsSmokeProofEventType, operationsSmokeProofCommandType, opsProofKey)
		cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, otherProjectKey)
	})

	sourceEvidenceURI := "local:" + fixtureKey + ":other-source-alignment"
	sourceBinding, err := SourceAlignmentCurrentBinding()
	if err != nil {
		t.Fatalf("source alignment current binding: %v", err)
	}
	if _, err := store.RecordSourceAlignmentProof(ctx, other, RecordSourceAlignmentProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSourceAlignmentProofFacts,
		Summary:                "Other project source alignment proof must not satisfy missing AreaMatrix target",
		EvidenceURI:            sourceEvidenceURI,
		SourceAlignmentBinding: sourceBinding,
		IdempotencyKey:         sourceProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record other project source alignment proof: %v", err)
	}
	protectedEvidenceURI := "local:" + fixtureKey + ":other-protected-path"
	if _, err := store.RecordProtectedPathProof(ctx, other, RecordProtectedPathProofOptions{
		ProofStatus:    "clean",
		Summary:        "Other project protected path proof must not satisfy missing AreaMatrix target",
		EvidenceURI:    protectedEvidenceURI,
		IdempotencyKey: protectedProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record other project protected path proof: %v", err)
	}
	opsEvidenceURI := "local:" + fixtureKey + ":other-ops-smoke"
	if _, err := store.RecordOperationsSmokeProof(ctx, other, RecordOperationsSmokeProofOptions{
		ProofKey:       "local_ops_smoke",
		EvidenceStatus: "pass",
		Summary:        "Other project operations smoke proof must not satisfy missing AreaMatrix target",
		EvidenceURI:    opsEvidenceURI,
		IdempotencyKey: opsProofKey,
		Metadata: map[string]any{
			"fixture_key": fixtureKey,
			"test_name":   t.Name(),
		},
	}); err != nil {
		t.Fatalf("record other project operations smoke proof: %v", err)
	}

	audit, err := store.CompletionAudit(ctx, CompletionAuditOptions{GeneratedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("build completion audit: %v", err)
	}
	sourceItem := findCompletionAuditItem(t, audit, "E1_design_source_alignment")
	if sourceItem.Status == "complete" ||
		sourceItem.Metadata["latest_source_alignment_proof_evidence_uri"] == sourceEvidenceURI ||
		sourceItem.Metadata["latest_source_alignment_proof_project_key"] == otherProjectKey {
		t.Fatalf("missing target should not consume other project source proof: %+v", sourceItem)
	}
	if !containsString(sourceItem.BlockedBy, "source_alignment_proof_query_failed") {
		t.Fatalf("missing target should report source proof query failure: %+v", sourceItem)
	}
	opsItem := findCompletionAuditItem(t, audit, "E7_operations_readiness")
	if opsItem.Metadata["latest_operations_smoke_proof_evidence_uri"] == opsEvidenceURI {
		t.Fatalf("missing target should not leak other project operations proof: %+v", opsItem.Metadata)
	}
	protectedItem := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if protectedItem.Status == "complete" ||
		protectedItem.Metadata["latest_proof_evidence_uri"] == protectedEvidenceURI ||
		protectedItem.Metadata["latest_proof_project_key"] == otherProjectKey {
		t.Fatalf("missing target should not consume other project protected path proof: %+v", protectedItem)
	}
	if !containsString(protectedItem.BlockedBy, "protected_path_proof_query_failed") {
		t.Fatalf("missing target should report protected path query failure: %+v", protectedItem)
	}
}

func TestCompletionAuditUsesProjectScopedRemainingProofsWithPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AREAFLOW_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set AREAFLOW_DATABASE_URL to run PostgreSQL completion audit remaining project proof integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if _, err := migrate.Up(ctx, pool); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	store := NewStore(pool)
	fixtureKey := fmt.Sprintf("completion-audit-remaining-proof-%d", time.Now().UnixNano())
	otherProjectKey := fixtureKey + "-other"
	target, createdTarget := ensureIsolationProject(ctx, t, pool, completionAuditTargetProjectKey)
	other := insertIsolationProject(ctx, t, pool, otherProjectKey)

	type proofKeys struct {
		target string
		other  string
	}
	keys := map[string]proofKeys{
		"task":             {target: fixtureKey + ":task:areamatrix", other: fixtureKey + ":task:other"},
		"validation":       {target: fixtureKey + ":validation:areamatrix", other: fixtureKey + ":validation:other"},
		"archive":          {target: fixtureKey + ":archive:areamatrix", other: fixtureKey + ":archive:other"},
		"shim":             {target: fixtureKey + ":shim:areamatrix", other: fixtureKey + ":shim:other"},
		"execution":        {target: fixtureKey + ":execution:areamatrix", other: fixtureKey + ":execution:other"},
		"releasePackaging": {target: fixtureKey + ":release-packaging:areamatrix", other: fixtureKey + ":release-packaging:other"},
		"backupRestore":    {target: fixtureKey + ":backup-restore:areamatrix", other: fixtureKey + ":backup-restore:other"},
		"securityClosure":  {target: fixtureKey + ":security-closure:areamatrix", other: fixtureKey + ":security-closure:other"},
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, taskMatrixProofEventType, taskMatrixProofCommandType, keys["task"].target, keys["task"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, validationProofEventType, validationProofCommandType, keys["validation"].target, keys["validation"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, archiveProofEventType, archiveProofCommandType, keys["archive"].target, keys["archive"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, shimRetirementProofEventType, shimRetirementProofCommandType, keys["shim"].target, keys["shim"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, executionCutoverProofEventType, executionCutoverProofCommandType, keys["execution"].target, keys["execution"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, releasePackagingProofEventType, releasePackagingProofCommandType, keys["releasePackaging"].target, keys["releasePackaging"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, backupRestoreProofEventType, backupRestoreProofCommandType, keys["backupRestore"].target, keys["backupRestore"].other)
		cleanupCompletionAuditProofFixture(cleanupCtx, t, pool, fixtureKey, securityClosureProofEventType, securityClosureProofCommandType, keys["securityClosure"].target, keys["securityClosure"].other)
		cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, otherProjectKey)
		if createdTarget {
			cleanupProjectIsolationFixture(cleanupCtx, t, pool, fixtureKey, completionAuditTargetProjectKey)
		}
	})

	metadata := map[string]any{"fixture_key": fixtureKey, "test_name": t.Name()}
	releasePackagingMetadata := ReleaseEvidenceBundleBindingMetadata(readyReleaseEvidenceBundle())
	releasePackagingMetadata["fixture_key"] = fixtureKey
	releasePackagingMetadata["test_name"] = t.Name()
	targetURI := func(key string) string { return "local:" + fixtureKey + ":areamatrix-" + key }
	otherURI := func(key string) string { return "local:" + fixtureKey + ":other-" + key }

	if _, err := store.RecordTaskMatrixProof(ctx, target, withCurrentTaskMatrixProofBinding(t, RecordTaskMatrixProofOptions{ProofStatus: "complete", Facts: requiredTaskMatrixProofFacts, Summary: "AreaMatrix task matrix proof", EvidenceURI: targetURI("task-matrix"), IdempotencyKey: keys["task"].target, Metadata: metadata})); err != nil {
		t.Fatalf("record areamatrix task matrix proof: %v", err)
	}
	if _, err := store.RecordValidationProof(ctx, target, withValidationEvidenceBinding(RecordValidationProofOptions{ProofStatus: "complete", Facts: requiredValidationProofFacts, Summary: "AreaMatrix validation proof", EvidenceURI: targetURI("validation"), IdempotencyKey: keys["validation"].target, Metadata: metadata})); err != nil {
		t.Fatalf("record areamatrix validation proof: %v", err)
	}
	if _, err := store.RecordArchiveProof(ctx, target, withArchiveProofTestBinding(RecordArchiveProofOptions{ProofStatus: "complete", Facts: requiredArchiveProofFacts, Summary: "AreaMatrix archive proof", EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"), IdempotencyKey: keys["archive"].target, Metadata: metadata})); err != nil {
		t.Fatalf("record areamatrix archive proof: %v", err)
	}
	if _, err := store.RecordShimRetirementProof(ctx, target, withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{ProofStatus: "complete", Facts: requiredShimRetirementProofFacts, Summary: "AreaMatrix shim retirement proof", EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"), IdempotencyKey: keys["shim"].target, Metadata: metadata})); err != nil {
		t.Fatalf("record areamatrix shim retirement proof: %v", err)
	}
	if _, err := store.RecordExecutionCutoverProof(ctx, target, withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{ProofStatus: "complete", Facts: requiredExecutionCutoverProofFacts, Summary: "AreaMatrix execution cutover proof", EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"), IdempotencyKey: keys["execution"].target, Metadata: metadata})); err != nil {
		t.Fatalf("record areamatrix execution cutover proof: %v", err)
	}
	if _, err := store.RecordReleasePackagingProof(ctx, target, RecordReleasePackagingProofOptions{ProofStatus: "complete", Facts: requiredReleasePackagingProofFacts, Summary: "AreaMatrix release packaging proof", EvidenceURI: targetURI("release-packaging"), IdempotencyKey: keys["releasePackaging"].target, Metadata: releasePackagingMetadata}); err != nil {
		t.Fatalf("record areamatrix release packaging proof: %v", err)
	}
	if _, err := store.RecordBackupRestoreProof(ctx, target, withBackupRestoreEvidenceBinding(RecordBackupRestoreProofOptions{ProofStatus: "complete", Facts: requiredBackupRestoreProofFacts, Summary: "AreaMatrix backup restore proof", EvidenceURI: targetURI("backup-restore"), IdempotencyKey: keys["backupRestore"].target, Metadata: metadata})); err != nil {
		t.Fatalf("record areamatrix backup restore proof: %v", err)
	}
	if _, err := store.RecordSecurityClosureProof(ctx, target, RecordSecurityClosureProofOptions{ProofStatus: "complete", Facts: requiredSecurityClosureProofFacts, Summary: "AreaMatrix security closure proof", EvidenceURI: targetURI("security-closure"), SecurityClosureBinding: readySecurityClosureCurrentBinding(target).Metadata, IdempotencyKey: keys["securityClosure"].target, Metadata: metadata}); err != nil {
		t.Fatalf("record areamatrix security closure proof: %v", err)
	}

	if _, err := store.RecordTaskMatrixProof(ctx, other, RecordTaskMatrixProofOptions{ProofStatus: "blocked", Summary: "Other project task matrix proof must not shadow AreaMatrix", EvidenceURI: otherURI("task-matrix"), IdempotencyKey: keys["task"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other task matrix proof: %v", err)
	}
	if _, err := store.RecordValidationProof(ctx, other, RecordValidationProofOptions{ProofStatus: "blocked", Summary: "Other project validation proof must not shadow AreaMatrix", EvidenceURI: otherURI("validation"), IdempotencyKey: keys["validation"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other validation proof: %v", err)
	}
	if _, err := store.RecordArchiveProof(ctx, other, RecordArchiveProofOptions{ProofStatus: "blocked", Summary: "Other project archive proof must not shadow AreaMatrix", EvidenceURI: otherURI("archive"), IdempotencyKey: keys["archive"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other archive proof: %v", err)
	}
	if _, err := store.RecordShimRetirementProof(ctx, other, RecordShimRetirementProofOptions{ProofStatus: "blocked", Summary: "Other project shim proof must not shadow AreaMatrix", EvidenceURI: otherURI("shim-retirement"), IdempotencyKey: keys["shim"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other shim retirement proof: %v", err)
	}
	if _, err := store.RecordExecutionCutoverProof(ctx, other, RecordExecutionCutoverProofOptions{ProofStatus: "blocked", Summary: "Other project execution proof must not shadow AreaMatrix", EvidenceURI: otherURI("execution-cutover"), IdempotencyKey: keys["execution"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other execution cutover proof: %v", err)
	}
	if _, err := store.RecordReleasePackagingProof(ctx, other, RecordReleasePackagingProofOptions{ProofStatus: "blocked", Summary: "Other project release packaging proof must not shadow AreaMatrix", EvidenceURI: otherURI("release-packaging"), IdempotencyKey: keys["releasePackaging"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other release packaging proof: %v", err)
	}
	if _, err := store.RecordBackupRestoreProof(ctx, other, RecordBackupRestoreProofOptions{ProofStatus: "blocked", Summary: "Other project backup restore proof must not shadow AreaMatrix", EvidenceURI: otherURI("backup-restore"), IdempotencyKey: keys["backupRestore"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other backup restore proof: %v", err)
	}
	if _, err := store.RecordSecurityClosureProof(ctx, other, RecordSecurityClosureProofOptions{ProofStatus: "blocked", Summary: "Other project security closure proof must not shadow AreaMatrix", EvidenceURI: otherURI("security-closure"), IdempotencyKey: keys["securityClosure"].other, Metadata: metadata}); err != nil {
		t.Fatalf("record other security closure proof: %v", err)
	}

	assertLatestProofProject(ctx, t, store.LatestTaskMatrixProof, otherProjectKey, otherURI("task-matrix"))
	assertLatestProofProject(ctx, t, store.LatestValidationProof, otherProjectKey, otherURI("validation"))
	assertLatestProofProject(ctx, t, store.LatestArchiveProof, otherProjectKey, otherURI("archive"))
	assertLatestProofProject(ctx, t, store.LatestShimRetirementProof, otherProjectKey, otherURI("shim-retirement"))
	assertLatestProofProject(ctx, t, store.LatestExecutionCutoverProof, otherProjectKey, otherURI("execution-cutover"))
	assertLatestProofProject(ctx, t, store.LatestReleasePackagingProof, otherProjectKey, otherURI("release-packaging"))
	assertLatestProofProject(ctx, t, store.LatestBackupRestoreProof, otherProjectKey, otherURI("backup-restore"))
	assertLatestProofProject(ctx, t, store.LatestSecurityClosureProof, otherProjectKey, otherURI("security-closure"))

	audit, err := store.CompletionAudit(ctx, CompletionAuditOptions{GeneratedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("build completion audit: %v", err)
	}
	assertCompletionAuditScopedProofItem(t, audit, "E2_phase_task_matrix", "latest_task_matrix_proof_project_key", "latest_task_matrix_proof_evidence_uri", targetURI("task-matrix"))
	assertCompletionAuditScopedProofItem(t, audit, "E3_command_api_smoke_evidence", "latest_validation_proof_project_key", "latest_validation_proof_evidence_uri", targetURI("validation"))
	dogfood := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if dogfood.Status == "complete" ||
		!containsString(dogfood.BlockedBy, "project_root_not_real_areamatrix") ||
		!containsString(dogfood.BlockedBy, "project_kind_not_product_repo") {
		t.Fatalf("E4 should consume AreaMatrix project proofs but keep fixture identity blocked: %+v", dogfood)
	}
	assertCompletionAuditItemMetadata(t, dogfood, "latest_archive_proof_project_key", completionAuditTargetProjectKey)
	assertCompletionAuditItemMetadata(t, dogfood, "latest_archive_proof_evidence_uri", e4ReleaseCandidateEvidenceURI("e4-archive"))
	assertCompletionAuditItemMetadata(t, dogfood, "latest_shim_retirement_proof_project_key", completionAuditTargetProjectKey)
	assertCompletionAuditItemMetadata(t, dogfood, "latest_shim_retirement_proof_evidence_uri", e4ReleaseCandidateEvidenceURI("e4-shim-retirement"))
	assertCompletionAuditItemMetadata(t, dogfood, "latest_execution_cutover_proof_project_key", completionAuditTargetProjectKey)
	assertCompletionAuditItemMetadata(t, dogfood, "latest_execution_cutover_proof_evidence_uri", e4ReleaseCandidateEvidenceURI("e4-execution-cutover"))
	assertCompletionAuditScopedProofItem(t, audit, "E5_release_packaging_preview", "latest_release_packaging_proof_project_key", "latest_release_packaging_proof_evidence_uri", targetURI("release-packaging"))
	assertCompletionAuditScopedProofItem(t, audit, "E6_backup_restore_artifact_retention", "latest_backup_restore_proof_project_key", "latest_backup_restore_proof_evidence_uri", targetURI("backup-restore"))
	assertCompletionAuditScopedProofItem(t, audit, "E8_security_permission_isolation", "latest_security_closure_proof_project_key", "latest_security_closure_proof_evidence_uri", targetURI("security-closure"))
}

func assertLatestProofProject(ctx context.Context, t *testing.T, latest any, wantProjectKey string, wantEvidenceURI string) {
	t.Helper()

	assert := func(project Record, metadata map[string]any, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("load global latest proof: %v", err)
		}
		if project.Key != wantProjectKey || metadataString(metadata, "evidence_uri") != wantEvidenceURI {
			t.Fatalf("global latest proof = project %q evidence %q, want project %q evidence %q", project.Key, metadataString(metadata, "evidence_uri"), wantProjectKey, wantEvidenceURI)
		}
	}

	switch load := latest.(type) {
	case func(context.Context) (TaskMatrixProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (ValidationProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (ArchiveProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (ShimRetirementProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (ExecutionCutoverProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (ReleasePackagingProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (BackupRestoreProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	case func(context.Context) (SecurityClosureProof, error):
		proof, err := load(ctx)
		assert(proof.Project, proof.Metadata, err)
	default:
		t.Fatalf("unsupported latest proof loader %T", latest)
	}
}

func assertCompletionAuditScopedProofItem(t *testing.T, audit CompletionAudit, itemKey string, projectMetadataKey string, evidenceMetadataKey string, wantEvidenceURI string) {
	t.Helper()
	item := findCompletionAuditItem(t, audit, itemKey)
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("%s should consume AreaMatrix project proof, not global latest proof: %+v", itemKey, item)
	}
	assertCompletionAuditItemMetadata(t, item, projectMetadataKey, completionAuditTargetProjectKey)
	assertCompletionAuditItemMetadata(t, item, evidenceMetadataKey, wantEvidenceURI)
}

func assertCompletionAuditItemMetadata(t *testing.T, item CompletionAuditItem, key string, want any) {
	t.Helper()
	if item.Metadata[key] != want {
		t.Fatalf("%s metadata[%q] = %#v, want %#v; metadata: %+v", item.Key, key, item.Metadata[key], want, item.Metadata)
	}
}

func ensureIsolationProject(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) (Record, bool) {
	t.Helper()
	var record Record
	err := pool.QueryRow(ctx, `
SELECT id, project_key, name, kind, adapter, workflow_profile, COALESCE(default_branch, '')
FROM projects
WHERE project_key = $1`,
		key,
	).Scan(
		&record.ID,
		&record.Key,
		&record.Name,
		&record.Kind,
		&record.Adapter,
		&record.WorkflowProfile,
		&record.DefaultBranch,
	)
	if err == nil {
		return record, false
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("load project %s: %v", key, err)
	}
	return insertIsolationProject(ctx, t, pool, key), true
}

func projectExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE project_key = $1)`, key).Scan(&exists); err != nil {
		t.Fatalf("check project %s exists: %v", key, err)
	}
	return exists
}

func insertIsolationProject(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) Record {
	t.Helper()
	var record Record
	err := pool.QueryRow(ctx, `
INSERT INTO projects (project_key, name, kind, adapter, workflow_profile, default_branch, updated_at)
VALUES ($1, $2, 'integration-fixture', 'areamatrix', 'areamatrix', 'main', now())
RETURNING id, project_key, name, kind, adapter, workflow_profile, COALESCE(default_branch, '')`,
		key,
		"Isolation "+key,
	).Scan(
		&record.ID,
		&record.Key,
		&record.Name,
		&record.Kind,
		&record.Adapter,
		&record.WorkflowProfile,
		&record.DefaultBranch,
	)
	if err != nil {
		t.Fatalf("insert project %s: %v", key, err)
	}
	return record
}

func insertStatusProjectionApplyPermissionFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO project_permissions (project_id, capability, effect, resource_type, pattern)
VALUES
  ($1, 'write_status', 'allow', 'capability', 'write_status'),
  ($1, 'write_status', 'allow', 'path', '.areaflow/status.json')`,
		projectID,
	); err != nil {
		t.Fatalf("insert status projection apply permission fixture: %v", err)
	}
}

func insertStatusProjectionApplyImportSnapshotFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, sourceHash string, fixtureKey string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO project_status_snapshots (project_id, snapshot_kind, summary, source_hash, export_path)
VALUES ($1, 'import', $2::jsonb, $3, '.areaflow/status.json')`,
		projectID,
		fmt.Sprintf(`{"summary_state":"mirroring","fixture_key":%q}`, fixtureKey),
		sourceHash,
	); err != nil {
		t.Fatalf("insert status projection apply import snapshot fixture: %v", err)
	}
}

func securityClosureBindingFixtureConfig(projectKey string, fixtureKey string) Config {
	return Config{
		Version: 1,
		Project: ProjectConfig{
			ID:              projectKey,
			Name:            "Security Closure " + projectKey,
			Root:            "/tmp/" + fixtureKey + "/" + projectKey,
			Kind:            "integration-fixture",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
		},
		Permissions: Permissions{
			Capabilities: map[string]bool{
				"read_project":    true,
				"write_status":    true,
				"write_workflow":  false,
				"write_generated": false,
				"write_code":      false,
				"run_commands":    false,
				"manage_git":      false,
				"network":         false,
				"use_secrets":     false,
				"execute_agents":  false,
			},
			ReadPaths:  []string{"docs/**", "workflow/**", "tasks/**"},
			WritePaths: []string{".areaflow/status.json"},
			ForbiddenPath: []string{
				"workflow/versions/*/execution/**",
				"workflow/versions/*/execution/_shared/progress.json",
				".areamatrix/**",
				"**/*.sqlite",
				"**/*.db",
			},
		},
		Commands: Commands{
			Forbidden: []string{"./task-loop run", "git reset --hard", "git checkout --", "rm -rf"},
		},
		Scheduling: Scheduling{
			Priority:             100,
			MaxParallelTasks:     1,
			AgentRole:            "local_worker",
			RequiredCapabilities: []string{"read_project", "write_artifacts"},
			EngineProfile:        "codex-cli",
		},
		Engines: Engines{
			Default: "codex-cli",
			Profiles: []EngineProfileConfig{
				{ID: "codex-cli", Provider: "codex-cli", SecretRef: "none", Enabled: false},
			},
		},
		StatusExport: StatusExport{
			Path: ".areaflow/status.json",
		},
		Migration: Migration{
			Strategy: "import_mirror_shadow_cutover_archive",
			Phase:    "import",
		},
		SourcePath: "integration:" + fixtureKey + ":areaflow.yaml",
		SourceHash: fixtureKey,
	}
}

func seedSecurityClosureAuditCoverageFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, fixtureKey string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
WITH audit_seed(action, capability, resource_type, resource, decision, reason) AS (
  VALUES
    ('project.upsert', 'project_config', 'project', $2, 'allowed', 'fixture project registration audit coverage evidence'),
    ('status.export', 'write_status', 'path', '.areaflow/status.json', 'allowed', 'fixture status export audit coverage evidence'),
    ('workflow.version.create', 'write_workflow', 'workflow_version', 'fixture-v1', 'allowed', 'fixture workflow version audit coverage evidence'),
    ('workflow.stage_skeleton.create', 'write_workflow', 'workflow_version', 'fixture-v1', 'allowed', 'fixture stage skeleton audit coverage evidence'),
    ('workflow.item.mark_ready', 'write_workflow', 'workflow_item', 'fixture-item', 'allowed', 'fixture item ready audit coverage evidence'),
    ('workflow.approval.record', 'approval', 'workflow_version', 'fixture-v1', 'approved', 'fixture approval audit coverage evidence'),
    ('runner.preview', 'execute_runner', 'workflow_version', 'fixture-v1', 'allowed', 'fixture runner preview audit coverage evidence'),
    ('worker.register', 'manage_workers', 'worker', 'fixture-worker', 'allowed', 'fixture worker register audit coverage evidence'),
    ('worker.run_once', 'execute_worker', 'worker', 'fixture-worker', 'denied', 'fixture worker denial audit coverage evidence'),
    ('lease.acquire', 'manage_workers', 'lease', 'fixture-lease', 'allowed', 'fixture lease acquire audit coverage evidence'),
    ('lease.release', 'manage_workers', 'lease', 'fixture-lease', 'allowed', 'fixture lease release audit coverage evidence'),
    ('lease.recover', 'manage_workers', 'lease', 'fixture-lease', 'allowed', 'fixture lease recover audit coverage evidence'),
    ('command.execute', 'run_commands', 'command', 'fixture-command', 'denied', 'fixture command execution audit coverage evidence'),
    ('secret.resolve', 'use_secrets', 'secret_ref', 'fixture-secret', 'denied', 'fixture secret resolve audit coverage evidence'),
    ('permission.change', 'permission', 'project', $2, 'denied', 'fixture permission change audit coverage evidence')
)
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
SELECT
  $1,
  audit_seed.action,
  audit_seed.capability,
  audit_seed.resource_type,
  audit_seed.resource,
  audit_seed.decision,
  audit_seed.reason,
  jsonb_build_object('fixture_key', $2, 'security_closure_binding', true)
FROM audit_seed`,
		projectID,
		fixtureKey,
	); err != nil {
		t.Fatalf("seed security closure audit coverage fixture: %v", err)
	}
}

func insertIsolationWorkflowVersion(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, fixtureKey string, label string) WorkflowVersion {
	t.Helper()
	statusSummary := fmt.Sprintf(`{"fixture_key":%q,"profile_binding":{"profile_id":"areamatrix"}}`, fixtureKey)
	version, err := scanWorkflowVersion(pool.QueryRow(ctx, `
INSERT INTO workflow_versions (
    project_id, display_label, version_kind, lifecycle_status,
    source_path, source_hash, import_mode, immutable, status_summary
)
VALUES ($1, $2, 'workflow_version', 'authoring', 'integration', $3, 'authored', false, $4::jsonb)
RETURNING id, project_id, display_label, version_kind, lifecycle_status,
          COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
          immutable, status_summary, created_at, updated_at, imported_at`,
		projectID,
		label,
		fixtureKey,
		statusSummary,
	))
	if err != nil {
		t.Fatalf("insert workflow version for project %d: %v", projectID, err)
	}
	return version
}

func insertIsolationRunAndTask(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, versionID int64, fixtureKey string) (int64, int64) {
	t.Helper()
	var runID int64
	err := pool.QueryRow(ctx, `
INSERT INTO runs (
    project_id, workflow_version_id, run_type, run_kind, status,
    risk_level, risk_policy, dry_run, summary, metadata
)
VALUES ($1, $2, 'execution', 'integration_isolation', 'queued',
        'low', 'pause', true, $3::jsonb, $3::jsonb)
RETURNING id`,
		projectID,
		versionID,
		fmt.Sprintf(`{"fixture_key":%q}`, fixtureKey),
	).Scan(&runID)
	if err != nil {
		t.Fatalf("insert run for project %d: %v", projectID, err)
	}
	var taskID int64
	err = pool.QueryRow(ctx, `
INSERT INTO run_tasks (
    project_id, workflow_version_id, run_id, task_key, task_kind,
    status, risk_level, sequence, metadata
)
VALUES ($1, $2, $3, 'task-01', 'integration_isolation', 'queued', 'low', 1, $4::jsonb)
RETURNING id`,
		projectID,
		versionID,
		runID,
		fmt.Sprintf(`{"fixture_key":%q}`, fixtureKey),
	).Scan(&taskID)
	if err != nil {
		t.Fatalf("insert run task for project %d: %v", projectID, err)
	}
	return runID, taskID
}

func insertIsolationWorkerAndLease(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, runID int64, taskID int64, fixtureKey string) (WorkerRecord, int64) {
	t.Helper()
	worker, err := scanWorker(pool.QueryRow(ctx, `
INSERT INTO workers (
    project_id, worker_key, worker_type, status, capabilities, metadata,
    heartbeat_interval_seconds, lease_timeout_seconds, updated_at
)
VALUES ($1, $2, 'local_host', 'online', '["read_project","write_artifacts"]'::jsonb,
        $3::jsonb, 30, 300, now())
RETURNING id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
          COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
          registered_at, last_heartbeat_at, heartbeat_interval_seconds,
          lease_timeout_seconds, updated_at`,
		projectID,
		fmt.Sprintf("worker-%d", projectID),
		fmt.Sprintf(`{"fixture_key":%q}`, fixtureKey),
	))
	if err != nil {
		t.Fatalf("insert worker for project %d: %v", projectID, err)
	}
	var leaseID int64
	err = pool.QueryRow(ctx, `
INSERT INTO leases (
    project_id, run_id, run_task_id, worker_id, lease_kind, status,
    expires_at, allowed_capabilities, scope, metadata
)
VALUES ($1, $2, $3, $4, 'run_task', 'active', now() - interval '1 minute',
        '["read_project"]'::jsonb, $5::jsonb, $6::jsonb)
RETURNING id`,
		projectID,
		runID,
		taskID,
		worker.ID,
		fmt.Sprintf(`{"run_task_id":%d}`, taskID),
		fmt.Sprintf(`{"fixture_key":%q}`, fixtureKey),
	).Scan(&leaseID)
	if err != nil {
		t.Fatalf("insert lease for project %d: %v", projectID, err)
	}
	return worker, leaseID
}

func insertIsolationArtifact(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, versionID int64, runID int64, fixtureKey string, label string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
INSERT INTO artifacts (
    project_id, workflow_version_id, run_id, artifact_type, storage_backend,
    uri, source_path, sha256, size_bytes, content_type, metadata
)
VALUES ($1, $2, $3, $4, 'project_reference', $5, $6, $7, 12, 'application/json', $8::jsonb)`,
		projectID,
		versionID,
		runID,
		"integration_"+label,
		"project://"+label,
		"workflow/"+label+".json",
		"sha-"+label,
		fmt.Sprintf(`{"fixture_key":%q,"label":%q}`, fixtureKey, label),
	)
	if err != nil {
		t.Fatalf("insert artifact %s for project %d: %v", label, projectID, err)
	}
}

func insertIsolationEventAndAudit(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, versionID int64, runID int64, fixtureKey string, suffix string) {
	t.Helper()
	metadata := fmt.Sprintf(`{"fixture_key":%q,"suffix":%q}`, fixtureKey, suffix)
	if _, err := pool.Exec(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, 'integration.isolation', 'info', $4, $5::jsonb)`,
		projectID,
		runID,
		versionID,
		"isolation event "+suffix,
		metadata,
	); err != nil {
		t.Fatalf("insert event for project %d: %v", projectID, err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'integration.isolation', 'read_project', 'project', $2, 'allowed', 'isolation fixture', $3::jsonb)`,
		projectID,
		suffix,
		metadata,
	); err != nil {
		t.Fatalf("insert audit event for project %d: %v", projectID, err)
	}
}

func assertWorkflowVersionsIsolated(ctx context.Context, t *testing.T, store Store, record Record, wantVersionID int64) {
	t.Helper()
	versions, err := store.ListWorkflowVersions(ctx, record)
	if err != nil {
		t.Fatalf("list workflow versions for %s: %v", record.Key, err)
	}
	if len(versions) != 1 || versions[0].ID != wantVersionID || versions[0].ProjectID != record.ID {
		t.Fatalf("workflow versions leaked for %s: %+v, want version %d project %d", record.Key, versions, wantVersionID, record.ID)
	}
	version, err := store.GetWorkflowVersion(ctx, record, "shared-v1")
	if err != nil {
		t.Fatalf("get shared workflow version for %s: %v", record.Key, err)
	}
	if version.ID != wantVersionID || version.ProjectID != record.ID {
		t.Fatalf("shared label resolved across project for %s: %+v, want version %d project %d", record.Key, version, wantVersionID, record.ID)
	}
}

func assertRunsIsolated(ctx context.Context, t *testing.T, store Store, record Record, version WorkflowVersion, wantRunID int64) {
	t.Helper()
	runs, err := store.ListWorkflowVersionRuns(ctx, record, version, 10)
	if err != nil {
		t.Fatalf("list runs for %s: %v", record.Key, err)
	}
	if len(runs) != 1 || runs[0].ID != wantRunID || runs[0].ProjectID != record.ID || runs[0].WorkflowVersionID != version.ID {
		t.Fatalf("runs leaked for %s: %+v, want run %d project %d version %d", record.Key, runs, wantRunID, record.ID, version.ID)
	}
}

func assertArtifactsIsolated(ctx context.Context, t *testing.T, store Store, record Record, version WorkflowVersion, wantLabel string) {
	t.Helper()
	artifacts, err := store.ListProjectArtifacts(ctx, record, 10)
	if err != nil {
		t.Fatalf("list project artifacts for %s: %v", record.Key, err)
	}
	if len(artifacts) != 1 || artifacts[0].ProjectID != record.ID || artifacts[0].Metadata["label"] != wantLabel {
		t.Fatalf("project artifacts leaked for %s: %+v, want label %s", record.Key, artifacts, wantLabel)
	}
	versionArtifacts, err := store.ListWorkflowVersionArtifacts(ctx, record, version, 10)
	if err != nil {
		t.Fatalf("list version artifacts for %s: %v", record.Key, err)
	}
	if len(versionArtifacts) != 1 || versionArtifacts[0].ProjectID != record.ID || versionArtifacts[0].WorkflowVersionID != version.ID || versionArtifacts[0].Metadata["label"] != wantLabel {
		t.Fatalf("workflow version artifacts leaked for %s: %+v, want label %s", record.Key, versionArtifacts, wantLabel)
	}
}

func assertEventsAndAuditIsolated(ctx context.Context, t *testing.T, store Store, projectID int64) {
	t.Helper()
	events, err := store.ListEvents(ctx, projectID, 10)
	if err != nil {
		t.Fatalf("list events for project %d: %v", projectID, err)
	}
	if len(events) != 1 || events[0].ProjectID != projectID {
		t.Fatalf("events leaked for project %d: %+v", projectID, events)
	}
	auditEvents, err := store.ListAuditEvents(ctx, projectID, 10)
	if err != nil {
		t.Fatalf("list audit events for project %d: %v", projectID, err)
	}
	if len(auditEvents) != 1 || auditEvents[0].ProjectID != projectID {
		t.Fatalf("audit events leaked for project %d: %+v", projectID, auditEvents)
	}
}

func assertWorkersIsolated(ctx context.Context, t *testing.T, store Store, record Record, wantWorkerID int64) {
	t.Helper()
	workers, err := store.ListWorkers(ctx, record, 10)
	if err != nil {
		t.Fatalf("list workers for %s: %v", record.Key, err)
	}
	if len(workers) != 1 || workers[0].ID != wantWorkerID || workers[0].ProjectID != record.ID {
		t.Fatalf("workers leaked for %s: %+v, want worker %d project %d", record.Key, workers, wantWorkerID, record.ID)
	}
}

func assertLeaseStatus(ctx context.Context, t *testing.T, pool *pgxpool.Pool, leaseID int64, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(ctx, `SELECT status FROM leases WHERE id = $1`, leaseID).Scan(&got); err != nil {
		t.Fatalf("load lease %d status: %v", leaseID, err)
	}
	if got != want {
		t.Fatalf("lease %d status = %q, want %q", leaseID, got, want)
	}
}

func cleanupProjectIsolationFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, fixtureKey string, projectKeys ...string) {
	t.Helper()
	if strings.TrimSpace(fixtureKey) == "" {
		return
	}
	projectIDs := []int64{}
	rows, err := pool.Query(ctx, `SELECT id FROM projects WHERE project_key = ANY($1)`, projectKeys)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			if scanErr := rows.Scan(&id); scanErr == nil {
				projectIDs = append(projectIDs, id)
			}
		}
	}
	if len(projectIDs) > 0 {
		if _, err := pool.Exec(ctx, `DELETE FROM leases WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup leases: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM worker_heartbeats WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup worker heartbeats: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM workers WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup workers: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM run_attempts WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup run attempts: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM run_tasks WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup run tasks: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM artifacts WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup artifacts: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM events WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup events: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM audit_events WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup audit events: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM command_requests WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup command requests: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM runs WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup runs: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM workflow_items WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup workflow items: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM workflow_versions WHERE project_id = ANY($1)`, projectIDs); err != nil {
			t.Logf("cleanup workflow versions: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM projects WHERE project_key = ANY($1)`, projectKeys); err != nil {
		t.Logf("cleanup projects: %v", err)
	}
}

func cleanupStatusProjectionApplyIntegrationFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID int64, sourceHash string, idempotencyKey string) {
	t.Helper()
	if strings.TrimSpace(sourceHash) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM events
WHERE project_id = $1
  AND event_type IN ('project.status_projection.apply.blocked', 'project.status_projection.apply.completed')
  AND metadata->>'idempotency_key' = $2`,
		projectID,
		idempotencyKey,
	); err != nil {
		t.Logf("cleanup status projection apply events: %v", err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM audit_events
WHERE project_id = $1
  AND action = $2
  AND metadata->>'idempotency_key' = $3`,
		projectID,
		statusProjectionApplyCommandType,
		idempotencyKey,
	); err != nil {
		t.Logf("cleanup status projection apply audit events: %v", err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM command_requests
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		projectID,
		statusProjectionApplyCommandType,
		idempotencyKey,
	); err != nil {
		t.Logf("cleanup status projection apply command request: %v", err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM status_projections
WHERE project_id = $1 AND source_hash = $2`,
		projectID,
		sourceHash,
	); err != nil {
		t.Logf("cleanup status projections: %v", err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM project_status_snapshots
WHERE project_id = $1 AND source_hash = $2`,
		projectID,
		sourceHash,
	); err != nil {
		t.Logf("cleanup status projection snapshots: %v", err)
	}
}

func cleanupCompletionAuditProtectedPathProofFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, fixtureKey string, idempotencyKeys ...string) {
	t.Helper()
	if strings.TrimSpace(fixtureKey) == "" {
		return
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM events
WHERE event_type = $1 AND metadata->'metadata'->>'fixture_key' = $2`,
		protectedPathProofEventType,
		fixtureKey,
	); err != nil {
		t.Logf("cleanup protected path proof events: %v", err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM audit_events
WHERE action = $1 AND metadata->'metadata'->>'fixture_key' = $2`,
		protectedPathProofCommandType,
		fixtureKey,
	); err != nil {
		t.Logf("cleanup protected path proof audit events: %v", err)
	}
	if len(idempotencyKeys) > 0 {
		if _, err := pool.Exec(ctx, `
DELETE FROM command_requests
WHERE command_type = $1 AND idempotency_key = ANY($2)`,
			protectedPathProofCommandType,
			idempotencyKeys,
		); err != nil {
			t.Logf("cleanup protected path proof command requests: %v", err)
		}
	}
}

func cleanupCompletionAuditProofFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool, fixtureKey string, eventType string, commandType string, idempotencyKeys ...string) {
	t.Helper()
	if strings.TrimSpace(fixtureKey) == "" {
		return
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM events
WHERE event_type = $1 AND metadata->'metadata'->>'fixture_key' = $2`,
		eventType,
		fixtureKey,
	); err != nil {
		t.Logf("cleanup completion audit proof events: %v", err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM audit_events
WHERE action = $1 AND metadata->'metadata'->>'fixture_key' = $2`,
		commandType,
		fixtureKey,
	); err != nil {
		t.Logf("cleanup completion audit proof audit events: %v", err)
	}
	if len(idempotencyKeys) > 0 {
		if _, err := pool.Exec(ctx, `
DELETE FROM command_requests
WHERE command_type = $1 AND idempotency_key = ANY($2)`,
			commandType,
			idempotencyKeys,
		); err != nil {
			t.Logf("cleanup completion audit proof command requests: %v", err)
		}
	}
}
