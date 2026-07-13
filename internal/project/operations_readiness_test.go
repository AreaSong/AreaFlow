package project

import (
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/migrate"
)

func TestBuildSupportBundlePreviewIsMetadataOnly(t *testing.T) {
	generated := time.Date(2026, 7, 3, 13, 0, 0, 0, time.UTC)
	preview := BuildSupportBundlePreview(BackupManifest{
		Status:       "ready",
		ManifestHash: "abc123",
		Projects: []BackupProjectManifest{
			{Project: Record{Key: "areamatrix", RootPath: "/tmp/areamatrix"}},
		},
	}, AuditCoverage{Status: "warn", TotalAuditEvents: 7}, SupportBundlePreviewOptions{GeneratedAt: generated})

	if preview.Status != "ready" || preview.Mode != "metadata_only_support_bundle_preview" {
		t.Fatalf("unexpected support bundle preview: %+v", preview)
	}
	if !preview.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, generated)
	}
	if !preview.SafetyFacts["read_only"] || !preview.SafetyFacts["metadata_only"] ||
		preview.SafetyFacts["export_open"] ||
		preview.SafetyFacts["secret_values_included"] ||
		preview.SafetyFacts["prompt_text_included"] ||
		preview.SafetyFacts["user_file_contents_included"] ||
		preview.SafetyFacts["raw_artifact_contents_included"] ||
		preview.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("unexpected support bundle safety facts: %+v", preview.SafetyFacts)
	}
	if !containsString(preview.ExcludedSensitiveContent, "secret_values") ||
		!containsString(preview.ForbiddenActions, "export_support_bundle") {
		t.Fatalf("missing support bundle guardrails: %+v", preview)
	}
	if len(preview.PathReferences) != 1 || preview.PathReferences[0].ProjectKey != "areamatrix" {
		t.Fatalf("unexpected path references: %+v", preview.PathReferences)
	}
}

func TestBuildMigrationLedgerReadinessNeedsFullLedger(t *testing.T) {
	generated := time.Date(2026, 7, 3, 13, 30, 0, 0, time.UTC)
	readiness := BuildMigrationLedgerReadiness([]migrate.Migration{
		{Name: "000001_v0_1_core.sql"},
		{Name: "000002_v0_3_command_requests.sql"},
	}, map[string]bool{
		"000001_v0_1_core.sql":             true,
		"000002_v0_3_command_requests.sql": true,
	}, nil, true, false, MigrationLedgerReadinessOptions{GeneratedAt: generated})

	if readiness.Status != "needs_attention" {
		t.Fatalf("migration ledger status = %q, want needs_attention: %+v", readiness.Status, readiness)
	}
	if readiness.AppliedCount != 2 || readiness.PendingCount != 0 || readiness.FullLedgerTablePresent {
		t.Fatalf("unexpected migration ledger counts: %+v", readiness)
	}
	if readiness.PreflightApplyVerifyRemediationReady {
		t.Fatalf("full ledger proof should not be ready without full ledger table: %+v", readiness)
	}
	if !containsString(readiness.ForbiddenActions, "apply_migration") ||
		readiness.SafetyFacts["migration_apply_attempted"] ||
		readiness.SafetyFacts["database_write_attempted"] {
		t.Fatalf("unexpected migration ledger safety facts: %+v", readiness)
	}
}

func TestBuildMigrationLedgerReadinessAcceptsFullLedgerPhases(t *testing.T) {
	generated := time.Date(2026, 7, 3, 13, 45, 0, 0, time.UTC)
	readiness := BuildMigrationLedgerReadiness([]migrate.Migration{
		{Name: "000001_v0_1_core.sql"},
	}, map[string]bool{
		"000001_v0_1_core.sql": true,
	}, map[string]map[string]MigrationLedgerPhase{
		"000001_v0_1_core.sql": readyMigrationLedgerPhases(),
	}, true, true, MigrationLedgerReadinessOptions{GeneratedAt: generated})

	if readiness.Status != "ready" || !readiness.PreflightApplyVerifyRemediationReady {
		t.Fatalf("full migration ledger should be ready: %+v", readiness)
	}
	if len(readiness.Entries) != 1 || len(readiness.Entries[0].Phases) != 4 {
		t.Fatalf("migration ledger phases missing: %+v", readiness.Entries)
	}
}

func TestBuildOperationsReadinessAggregatesScopedOps(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 0, 0, 0, time.UTC)
	service := LocalServiceStatus{Status: "ready", Mode: "local_service"}
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "abc123"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := BuildMigrationLedgerReadiness([]migrate.Migration{{Name: "000001_v0_1_core.sql"}}, map[string]bool{"000001_v0_1_core.sql": true}, nil, true, false, MigrationLedgerReadinessOptions{GeneratedAt: generated})

	readiness := BuildOperationsReadiness(service, support, ledger, OperationsReadinessOptions{GeneratedAt: generated})

	if readiness.Status != "needs_attention" || readiness.Mode != "read_only_operations_readiness" {
		t.Fatalf("unexpected operations readiness: %+v", readiness)
	}
	assertOperationsItem(t, readiness, "install_migrate_start_register_smoke", "needs_attention")
	assertOperationsItem(t, readiness, "metadata_only_support_bundle_preview", "ready")
	assertOperationsItem(t, readiness, "migration_ledger_readiness", "needs_attention")
	assertOperationsItem(t, readiness, "local_only_telemetry_default", "ready")
	assertOperationsItem(t, readiness, "managed_ops_deferred", "deferred")
	if !readiness.SafetyFacts["read_only"] ||
		!readiness.SafetyFacts["support_bundle_metadata_only"] ||
		readiness.SafetyFacts["support_bundle_exported"] ||
		readiness.SafetyFacts["remote_telemetry_enabled"] ||
		readiness.SafetyFacts["managed_upgrade_attempted"] ||
		readiness.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("unexpected operations safety facts: %+v", readiness.SafetyFacts)
	}
}

func TestBuildOperationsReadinessBlocksUnsafeSupportBundlePreview(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 5, 0, 0, time.UTC)
	service := LocalServiceStatus{Status: "ready", Mode: "local_service"}
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "abc123"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	support.SafetyFacts["export_open"] = true
	support.SafetyFacts["prompt_text_included"] = true
	support.SafetyFacts["remote_upload_attempted"] = true
	support.ExcludedSensitiveContent = removeString(support.ExcludedSensitiveContent, "prompt_text")
	support.ForbiddenActions = removeString(support.ForbiddenActions, "upload_bundle")
	support.IncludedMetadata = append(support.IncludedMetadata, "raw_artifact_contents")
	ledger := BuildMigrationLedgerReadiness([]migrate.Migration{{Name: "000001_v0_1_core.sql"}}, map[string]bool{"000001_v0_1_core.sql": true}, map[string]map[string]MigrationLedgerPhase{
		"000001_v0_1_core.sql": readyMigrationLedgerPhases(),
	}, true, true, MigrationLedgerReadinessOptions{GeneratedAt: generated})
	proof := OperationsSmokeProof{
		Project:                         Record{ID: 7, Key: "areamatrix"},
		ProofKey:                        "local_ops_smoke",
		Status:                          "recorded",
		EvidenceStatus:                  "pass",
		EventID:                         43,
		CreatedAt:                       generated,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		ServiceProcessControlAttempted:  false,
		SupportBundleExported:           false,
		MigrationApplyAttempted:         false,
		RemoteTelemetryEnabled:          false,
		AreaMatrixProtectedPathsTouched: false,
		RecordCommandRunsSmoke:          false,
		Metadata:                        map[string]any{"summary": "ops smoke reviewed", "evidence_uri": "docs/development/operations-readiness-evidence.md"},
	}

	readiness := BuildOperationsReadiness(service, support, ledger, OperationsReadinessOptions{GeneratedAt: generated, SmokeProof: proof})

	if readiness.Status != "blocked" {
		t.Fatalf("unsafe support bundle should block operations readiness: %+v", readiness)
	}
	item := findOperationsItem(t, readiness, "metadata_only_support_bundle_preview")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "support_bundle_export_open") ||
		!containsString(item.BlockedBy, "support_bundle_prompt_text_included") ||
		!containsString(item.BlockedBy, "support_bundle_remote_upload_attempted") ||
		!containsString(item.BlockedBy, "support_bundle_exclusion_missing:prompt_text") ||
		!containsString(item.BlockedBy, "support_bundle_forbidden_action_missing:upload_bundle") ||
		!containsString(item.BlockedBy, "support_bundle_sensitive_metadata_included:raw_artifact_contents") {
		t.Fatalf("support bundle blocker details missing: %+v", item)
	}
}

func TestBuildOperationsReadinessConsumesSmokeProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 10, 0, 0, time.UTC)
	service := LocalServiceStatus{Status: "ready", Mode: "local_service"}
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "abc123"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := BuildMigrationLedgerReadiness([]migrate.Migration{{Name: "000001_v0_1_core.sql"}}, map[string]bool{"000001_v0_1_core.sql": true}, nil, true, false, MigrationLedgerReadinessOptions{GeneratedAt: generated})
	proof := OperationsSmokeProof{
		Project:                         Record{ID: 7, Key: "areamatrix-fixture"},
		ProofKey:                        "v1_stable_fixture_smoke",
		Status:                          "recorded",
		EvidenceStatus:                  "pass",
		EventID:                         42,
		CreatedAt:                       generated,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		ServiceProcessControlAttempted:  false,
		SupportBundleExported:           false,
		MigrationApplyAttempted:         false,
		RemoteTelemetryEnabled:          false,
		AreaMatrixProtectedPathsTouched: false,
		RecordCommandRunsSmoke:          false,
		Metadata:                        map[string]any{"summary": "fixture smoke passed", "evidence_uri": "docs/development/v1-stable-fixture-evidence.md"},
	}

	readiness := BuildOperationsReadiness(service, support, ledger, OperationsReadinessOptions{GeneratedAt: generated, SmokeProof: proof})

	item := findOperationsItem(t, readiness, "install_migrate_start_register_smoke")
	if item.Status != "ready" {
		t.Fatalf("smoke item status = %q, want ready: %+v", item.Status, item)
	}
	if item.Metadata["latest_smoke_proof_key"] != "v1_stable_fixture_smoke" ||
		item.Metadata["evidence_recorded"] != true {
		t.Fatalf("smoke proof metadata missing: %+v", item.Metadata)
	}
	if containsString(item.BlockedBy, "fresh_local_ops_smoke_missing") {
		t.Fatalf("smoke proof should remove fresh blocker: %+v", item.BlockedBy)
	}
}

func TestBuildOperationsReadinessRejectsStaleSmokeProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 10, 0, 0, time.UTC)
	service := LocalServiceStatus{Status: "ready", Mode: "local_service"}
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "abc123"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := BuildMigrationLedgerReadiness([]migrate.Migration{{Name: "000001_v0_1_core.sql"}}, map[string]bool{"000001_v0_1_core.sql": true}, map[string]map[string]MigrationLedgerPhase{
		"000001_v0_1_core.sql": readyMigrationLedgerPhases(),
	}, true, true, MigrationLedgerReadinessOptions{GeneratedAt: generated})
	proof := OperationsSmokeProof{
		Project:                         Record{ID: 7, Key: "areamatrix"},
		ProofKey:                        "local_ops_smoke",
		Status:                          "recorded",
		EvidenceStatus:                  "pass",
		EventID:                         42,
		CreatedAt:                       generated.Add(-operationsSmokeProofFreshnessWindow - time.Second),
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		ServiceProcessControlAttempted:  false,
		SupportBundleExported:           false,
		MigrationApplyAttempted:         false,
		RemoteTelemetryEnabled:          false,
		AreaMatrixProtectedPathsTouched: false,
		RecordCommandRunsSmoke:          false,
		Metadata:                        map[string]any{"summary": "operations smoke passed yesterday", "evidence_uri": "docs/development/operations-readiness-evidence.md"},
	}

	readiness := BuildOperationsReadiness(service, support, ledger, OperationsReadinessOptions{GeneratedAt: generated, SmokeProof: proof})

	item := findOperationsItem(t, readiness, "install_migrate_start_register_smoke")
	if item.Status != "needs_attention" ||
		!containsString(item.BlockedBy, "operations_smoke_proof_stale") ||
		item.Metadata["latest_smoke_proof_fresh"] != false ||
		item.Metadata["latest_smoke_proof_freshness_status"] != "stale" {
		t.Fatalf("stale smoke proof should not make readiness ready: %+v", item)
	}
	if item.Metadata["latest_smoke_proof_age_seconds"] == nil ||
		item.Metadata["smoke_proof_max_age_seconds"] != int64(operationsSmokeProofFreshnessWindow.Seconds()) {
		t.Fatalf("stale smoke proof metadata missing freshness facts: %+v", item.Metadata)
	}
}

func assertOperationsItem(t *testing.T, readiness OperationsReadiness, key string, status string) {
	t.Helper()
	item := findOperationsItem(t, readiness, key)
	if item.Status != status {
		t.Fatalf("operations item %s status = %q, want %q: %+v", key, item.Status, status, item)
	}
	if len(item.RequiredEvidence) == 0 {
		t.Fatalf("operations item %s missing required evidence: %+v", key, item)
	}
}

func findOperationsItem(t *testing.T, readiness OperationsReadiness, key string) OperationsReadinessItem {
	t.Helper()
	for _, item := range readiness.Items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("operations item %s not found: %+v", key, readiness.Items)
	return OperationsReadinessItem{}
}

func readyMigrationLedgerPhases() map[string]MigrationLedgerPhase {
	return map[string]MigrationLedgerPhase{
		"preflight":   {Phase: "preflight", Status: "ready"},
		"apply":       {Phase: "apply", Status: "ready"},
		"verify":      {Phase: "verify", Status: "ready"},
		"remediation": {Phase: "remediation", Status: "ready"},
	}
}
