package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func realAreaMatrixRecord() Record {
	return Record{
		ID:              1,
		Key:             completionAuditTargetProjectKey,
		Name:            "AreaMatrix",
		Kind:            "product-repo",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix",
		DefaultBranch:   "main",
		RootPath:        completionAuditTargetProjectRoot,
	}
}

func realAreaMatrixRecordPtr() *Record {
	record := realAreaMatrixRecord()
	return &record
}

func e4ReleaseCandidateEvidenceURI(anchor string) string {
	return "docs/development/real-release-candidate-evidence.md#" + anchor
}

func readyReleaseEvidenceBundle() ReleaseEvidenceBundle {
	return BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			Projects: []BackupProjectManifest{
				{
					Project:       realAreaMatrixRecord(),
					Inventory:     ImportInventory{Versions: 1},
					ArtifactCount: 1,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9},
		ReleaseEvidenceBundleOptions{
			GeneratedAt: time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
			ProjectID:   realAreaMatrixRecord().ID,
			ProjectKey:  realAreaMatrixRecord().Key,
		},
	)
}

func readySecurityClosureCurrentBinding(record Record) SecurityClosureCurrentBinding {
	generated := time.Date(2026, 7, 3, 17, 0, 0, 0, time.UTC)
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := BuildPermissionPolicyDoctor(record, testPermissionProjectConfig(generated), true, testPermissionRows(), PermissionPolicyDoctorOptions{GeneratedAt: generated})
	coverage := readyAuditCoverage(record, generated)
	return BuildSecurityClosureCurrentBinding(record, readiness, doctor, coverage, SecurityClosureCurrentBindingOptions{GeneratedAt: generated})
}

func readyAuditCoverage(record Record, generated time.Time) AuditCoverage {
	counts := []auditActionCount{}
	for _, spec := range auditCoverageRequirementSpecs {
		for _, action := range spec.Actions {
			decision := action.Decision
			if decision == "" {
				decision = "allowed"
			}
			counts = append(counts, auditActionCount{
				Action:      action.Action,
				Decision:    decision,
				Count:       1,
				LastAuditAt: generated,
			})
		}
	}
	return BuildAuditCoverage(AuditCoverageOptions{ProjectID: record.ID, ProjectKey: record.Key, GeneratedAt: generated}, int64(len(counts)), counts)
}

func fixtureReleaseEvidenceBundle() ReleaseEvidenceBundle {
	record := realAreaMatrixRecord()
	record.RootPath = "/tmp/areaflow-fixture/areamatrix"
	return BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			Projects: []BackupProjectManifest{
				{
					Project:       record,
					Inventory:     ImportInventory{Versions: 1},
					ArtifactCount: 1,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9},
		ReleaseEvidenceBundleOptions{
			GeneratedAt: time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
			ProjectID:   record.ID,
			ProjectKey:  record.Key,
		},
	)
}

func readyReleaseEvidenceBundleMetadata(bundle ReleaseEvidenceBundle) map[string]any {
	proofEvidenceURIMap := readyProofEvidenceURIMap()
	proofEvidenceURIs := readyProofEvidenceURIs()
	proofProvenanceMap := readyProofProvenanceMap()
	metadata := map[string]any{
		"summary":                          "real release candidate evidence reviewed",
		"review_decision":                  "approved",
		"reviewed_by":                      "release-owner",
		"reviewed_at":                      "2026-07-04T12:00:00Z",
		"review_metadata_status":           "approved",
		"proof_evidence_uris":              proofEvidenceURIs,
		"proof_evidence_uri_map":           proofEvidenceURIMap,
		"proof_evidence_uri_count":         len(proofEvidenceURIs),
		"required_proof_evidence_uri_keys": completionAuditSnapshotRequiredProofEvidenceURIKeys(),
		"proof_provenance_map":             proofProvenanceMap,
		"required_proof_provenance_keys":   completionAuditSnapshotRequiredProofProvenanceKeys(),
	}
	for key, value := range ReleaseEvidenceBundleBindingMetadata(bundle) {
		metadata[key] = value
	}
	return metadata
}

func readyReleaseEvidenceBundleMetadataWithProofURIs(bundle ReleaseEvidenceBundle, proofURIs []string) map[string]any {
	metadata := readyReleaseEvidenceBundleMetadata(bundle)
	metadata["proof_evidence_uris"] = proofURIs
	metadata["proof_evidence_uri_count"] = len(proofURIs)
	return metadata
}

func readyReleaseEvidenceBundleMetadataWithFileAudit(t *testing.T, bundle ReleaseEvidenceBundle, root string) map[string]any {
	t.Helper()
	metadata := readyReleaseEvidenceBundleMetadata(bundle)
	entries, blockers := completionAuditSnapshotAuditEvidenceURIRefs(
		root,
		"docs/development/real-release-candidate-evidence.md",
		readyProofEvidenceURIs(),
	)
	if len(blockers) > 0 {
		t.Fatalf("build release candidate file audit metadata: %+v", blockers)
	}
	metadata["evidence_uri_file_audit"] = entries
	metadata["evidence_uri_file_audit_count"] = len(entries)
	metadata["evidence_uri_file_audit_status"] = "pass"
	return metadata
}

func readyProofEvidenceURIMap() map[string]string {
	return map[string]string{
		"E1_design_source_alignment.latest_source_alignment_proof_evidence_uri":         "docs/development/real-release-candidate-evidence.md#e1-source-alignment",
		"E2_phase_task_matrix.latest_task_matrix_proof_evidence_uri":                    "docs/development/real-release-candidate-evidence.md#e2-task-matrix",
		"E3_command_api_smoke_evidence.latest_validation_proof_evidence_uri":            "docs/development/real-release-candidate-evidence.md#e3-validation",
		"E4_areamatrix_dogfood_completion.latest_archive_proof_evidence_uri":            "docs/development/real-release-candidate-evidence.md#e4-archive",
		"E4_areamatrix_dogfood_completion.latest_shim_retirement_proof_evidence_uri":    "docs/development/real-release-candidate-evidence.md#e4-shim-retirement",
		"E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri":  "docs/development/real-release-candidate-evidence.md#e4-execution-cutover",
		"E5_release_packaging_preview.latest_release_packaging_proof_evidence_uri":      "docs/development/real-release-candidate-evidence.md#e5-release-packaging",
		"E6_backup_restore_artifact_retention.latest_backup_restore_proof_evidence_uri": "docs/development/real-release-candidate-evidence.md#e6-backup-restore",
		"E7_operations_readiness.latest_operations_smoke_proof_evidence_uri":            "docs/development/real-release-candidate-evidence.md#e7-operations",
		"E8_security_permission_isolation.latest_security_closure_proof_evidence_uri":   "docs/development/real-release-candidate-evidence.md#e8-security",
		"E9_areamatrix_protected_path_proof.latest_proof_evidence_uri":                  "docs/development/real-release-candidate-evidence.md#e9-protected-path",
	}
}

func readyProofEvidenceURIs() []string {
	uriMap := readyProofEvidenceURIMap()
	keys := completionAuditSnapshotRequiredProofEvidenceURIKeys()
	uris := make([]string, 0, len(keys))
	for _, key := range keys {
		uris = append(uris, uriMap[key])
	}
	return uris
}

func readyProofEventIDs() map[string]int64 {
	ids := map[string]int64{}
	for index, key := range completionAuditSnapshotRequiredProofEventIDKeys() {
		ids[key] = int64(100 + index)
	}
	return ids
}

func readyProofProvenanceMap() map[string]string {
	return map[string]string{
		"E7_operations_readiness.latest_operations_smoke_proof_key": "manual_ops_smoke_review",
	}
}

func readyCompletionAuditSnapshotCurrentBinding(hash string) completionAuditSnapshotCurrentAuditBinding {
	if hash == "" {
		hash = "rc-hash"
	}
	return completionAuditSnapshotCurrentAuditBinding{
		Status:                   "complete",
		Scope:                    "v1.0",
		Hash:                     hash,
		ProofEvidenceURIs:        readyProofEvidenceURIs(),
		ProofEvidenceURIMap:      readyProofEvidenceURIMap(),
		ProofEventIDs:            readyProofEventIDs(),
		ProofProvenanceMap:       readyProofProvenanceMap(),
		PackageAStatusProjection: readyPackageAStatusProjectionBinding("source-hash-rc"),
	}
}

func readyPackageAStatusProjectionBinding(sourceHash string) completionAuditSnapshotPackageAStatusProjectionBinding {
	if sourceHash == "" {
		sourceHash = "source-hash-rc"
	}
	writtenAt := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	projection := StatusProjectionRecord{
		ID:           44,
		ProjectID:    realAreaMatrixRecord().ID,
		TargetKind:   "project_status_json",
		TargetURI:    ".areaflow/status.json",
		SummaryState: "stable_fallback_projection_v1",
		SourceHash:   sourceHash,
		WriteState:   "written",
		GeneratedAt:  writtenAt,
		WrittenAt:    &writtenAt,
		Metadata: map[string]any{
			"command_type":                statusProjectionApplyCommandType,
			"decision":                    "allowed",
			"write_hash":                  "status-projection-hash",
			"post_write_sha256":           "status-projection-hash",
			"post_write_verified":         true,
			"stable_projection_validated": true,
			"protected_paths_verified":    true,
			"root_contained":              true,
			"apply_gate_status":           "pass",
			"apply_gate_decision":         "go",
			"apply_command_eligible":      true,
			"project_write_attempted":     true,
			"execution_write_attempted":   false,
			"engine_call_attempted":       false,
		},
	}
	return completionAuditSnapshotPackageAStatusProjectionBinding{
		LatestImportSourceHash:  sourceHash,
		HasProjection:           true,
		LatestProjection:        projection,
		HasWrittenProjection:    true,
		LatestWrittenProjection: projection,
		CurrentPreimageCaptured: true,
		CurrentPreimage: StatusProjectionPreimage{
			TargetPath:         "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			Exists:             true,
			Readable:           true,
			SizeBytes:          512,
			SHA256:             "status-projection-hash",
			SchemaStatus:       "stable",
			SourceSnapshotHash: sourceHash,
			Message:            "target matches stable status projection shape",
		},
	}
}

func stablePackageAStatusProjectionWithoutProvenanceBinding(sourceHash string) completionAuditSnapshotPackageAStatusProjectionBinding {
	if sourceHash == "" {
		sourceHash = "source-hash-rc"
	}
	return completionAuditSnapshotPackageAStatusProjectionBinding{
		LatestImportSourceHash:  sourceHash,
		CurrentPreimageCaptured: true,
		CurrentPreimage: StatusProjectionPreimage{
			TargetPath:         "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			Exists:             true,
			Readable:           true,
			SizeBytes:          512,
			SHA256:             "stable-status-hash",
			SchemaStatus:       "stable",
			SourceSnapshotHash: sourceHash,
			Message:            "target matches stable status projection shape",
		},
	}
}

func readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t *testing.T, hash string, root string) completionAuditSnapshotCurrentAuditBinding {
	t.Helper()
	current := readyCompletionAuditSnapshotCurrentBinding(hash)
	current.EvidenceRoot = root
	return current
}

func releaseCandidateEvidenceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeReleaseCandidateEvidenceFile(t, root, []string{
		"release-candidate-reviewed",
		"e1-source-alignment",
		"e2-task-matrix",
		"e3-validation",
		"e4-archive",
		"e4-shim-retirement",
		"e4-execution-cutover",
		"e5-release-packaging",
		"e6-backup-restore",
		"e7-operations",
		"e8-security",
		"e9-protected-path",
	})
	return root
}

func releaseCandidateEvidencePath(root string) string {
	return filepath.Join(root, "docs", "development", "real-release-candidate-evidence.md")
}

func writeReleaseCandidateEvidenceFile(t *testing.T, root string, anchors []string) {
	t.Helper()
	path := releaseCandidateEvidencePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create release candidate evidence dir: %v", err)
	}
	var builder strings.Builder
	builder.WriteString("# Real Release Candidate Evidence\n\n")
	for _, anchor := range anchors {
		builder.WriteString("## ")
		builder.WriteString(anchor)
		builder.WriteString("\n\nReviewed evidence placeholder for mechanism tests.\n\n")
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write release candidate evidence fixture: %v", err)
	}
}

func normalizeReleaseCandidateSnapshotOptions(t *testing.T, options RecordCompletionAuditSnapshotOptions) RecordCompletionAuditSnapshotOptions {
	t.Helper()
	if strings.TrimSpace(options.EvidenceRoot) == "" {
		options.EvidenceRoot = releaseCandidateEvidenceRoot(t)
	}
	if strings.TrimSpace(options.ReviewDecision) == "" {
		options.ReviewDecision = "approved"
	}
	if strings.TrimSpace(options.ReviewedBy) == "" {
		options.ReviewedBy = "release-owner"
	}
	if options.ReviewedAt.IsZero() {
		options.ReviewedAt = time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	}
	return normalizeRecordCompletionAuditSnapshotOptions(options)
}

func completionAuditWithReviewedProofEvidence() CompletionAudit {
	uriMap := readyProofEvidenceURIMap()
	eventIDs := readyProofEventIDs()
	items := make([]CompletionAuditItem, 0, len(completionAuditSnapshotRequiredProofEvidenceItemKeys()))
	for _, itemKey := range completionAuditSnapshotRequiredProofEvidenceItemKeys() {
		metadata := map[string]any{}
		for _, metadataKey := range completionAuditSnapshotRequiredProofEvidenceURIKeysForItem(itemKey) {
			metadata[metadataKey] = uriMap[itemKey+"."+metadataKey]
		}
		for _, metadataKey := range completionAuditSnapshotRequiredProofEventIDKeysForItem(itemKey) {
			metadata[metadataKey] = eventIDs[itemKey+"."+metadataKey]
		}
		for _, metadataKey := range completionAuditSnapshotRequiredProofProvenanceKeysForItem(itemKey) {
			metadata[metadataKey] = readyProofProvenanceMap()[itemKey+"."+metadataKey]
		}
		items = append(items, CompletionAuditItem{
			Key:      itemKey,
			Category: "completion",
			Status:   "complete",
			Metadata: metadata,
		})
	}
	return CompletionAudit{
		Status: "complete",
		Mode:   "read_only_completion_audit",
		Scope:  "v1.0",
		Items:  items,
	}
}

func TestBuildCompletionAuditBlocksWithoutProtectedPathProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseFinalGate:          &ReleaseFinalGate{Status: "blocked", Mode: "read_only_release_final_gate"},
		SecurityBoundaryReadiness: ptrSecurityReadiness(BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})),
		LocalServiceStatus:        &LocalServiceStatus{Status: "ready", Mode: "local_service"},
	})

	if audit.Status != "blocked" || audit.Mode != "read_only_completion_audit" || audit.Scope != "v1.0" {
		t.Fatalf("unexpected completion audit: %+v", audit)
	}
	if !audit.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", audit.GeneratedAt, generated)
	}
	if audit.ReleaseFinalGateStatus != "incomplete" || audit.AreaMatrixDogfoodStatus != "incomplete" ||
		audit.TaskMatrixStatus != "incomplete" || audit.ImplementationGapStatus != "incomplete" ||
		audit.ProtectedPathProofStatus != "blocked" {
		t.Fatalf("unexpected aggregate statuses: %+v", audit)
	}
	if !audit.SafetyFacts["read_only"] ||
		audit.SafetyFacts["release_package_created"] ||
		audit.SafetyFacts["publish_attempted"] ||
		audit.SafetyFacts["restore_apply_attempted"] ||
		audit.SafetyFacts["secret_resolved"] ||
		audit.SafetyFacts["remote_worker_credentials_issued"] ||
		audit.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("unexpected completion audit safety facts: %+v", audit.SafetyFacts)
	}
	assertCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion", "blocked")
	assertCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof", "blocked")
	if !containsString(audit.ForbiddenActions, "run_smoke") ||
		!containsString(audit.ForbiddenActions, "touch_areamatrix_protected_paths") {
		t.Fatalf("missing completion audit forbidden actions: %+v", audit.ForbiddenActions)
	}
}

func TestBuildCompletionAuditRequiresProtectedPathProofRecord(t *testing.T) {
	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{})

	if audit.ProtectedPathProofStatus != "blocked" {
		t.Fatalf("missing protected path proof status = %q, want blocked", audit.ProtectedPathProofStatus)
	}
	item := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "blocked" || !containsString(item.BlockedBy, "protected_path_proof_missing") {
		t.Fatalf("missing protected path proof record must not complete E9: %+v", item)
	}
	if !strings.Contains(item.NextCommand, "--status clean") ||
		!strings.Contains(item.NextCommand, "--status authorized") ||
		!strings.Contains(item.NextCommand, "--dirty-output-hash") {
		t.Fatalf("missing protected path proof should expose clean and authorized next commands: %+v", item.NextCommand)
	}
	if _, ok := item.Metadata["option_protected_path_proof_status"]; ok {
		t.Fatalf("legacy option metadata should not be emitted: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditSnapshotRequiresCompleteAudit(t *testing.T) {
	_, err := buildCompletionAuditSnapshot(
		Record{ID: 1, Key: "areamatrix"},
		CompletionAudit{Status: "incomplete", Scope: "v1.0"},
		normalizeRecordCompletionAuditSnapshotOptions(RecordCompletionAuditSnapshotOptions{ReleaseCandidateLabel: "v1.0-rc1"}),
		ReleaseEvidenceBundle{},
	)
	if err == nil {
		t.Fatal("expected incomplete completion audit snapshot to be rejected")
	}
}

func TestBuildCompletionAuditSnapshotRequiresTargetProject(t *testing.T) {
	_, err := buildCompletionAuditSnapshot(
		Record{ID: 99, Key: "areamatrix-fixture"},
		CompletionAudit{Status: "complete", Scope: "v1.0"},
		normalizeRecordCompletionAuditSnapshotOptions(RecordCompletionAuditSnapshotOptions{ReleaseCandidateLabel: "v1.0-rc1"}),
		ReleaseEvidenceBundle{},
	)
	if err == nil {
		t.Fatal("expected completion audit snapshot to reject non-target project")
	}
}

func TestBuildCompletionAuditSnapshotCapturesHashAndProofIDs(t *testing.T) {
	result, err := buildCompletionAuditSnapshot(
		Record{ID: 1, Key: "areamatrix"},
		CompletionAudit{
			Status: "complete",
			Mode:   "read_only_completion_audit",
			Scope:  "v1.0",
			Items: []CompletionAuditItem{
				{
					Key:    "E1_design_source_alignment",
					Status: "complete",
					Metadata: map[string]any{
						"latest_source_alignment_proof_event_id":     int64(10),
						"latest_source_alignment_proof_evidence_uri": "docs/development/source-alignment-release-candidate-evidence.md",
					},
				},
				{
					Key:    "E9_areamatrix_protected_path_proof",
					Status: "complete",
					Metadata: map[string]any{
						"latest_proof_event_id":     int64(90),
						"latest_proof_evidence_uri": "docs/development/protected-path-release-candidate-evidence.md",
					},
				},
			},
		},
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceURI:           "local:completion-audit",
		}),
		ReleaseEvidenceBundle{},
	)
	if err != nil {
		t.Fatalf("build snapshot failed: %v", err)
	}
	if result.AuditStatus != "complete" || result.AuditScope != "v1.0" || result.AuditHash == "" {
		t.Fatalf("snapshot missing audit identity: %+v", result)
	}
	if result.EvidenceClass != "fixture" || result.Metadata["fixture_snapshot"] != true {
		t.Fatalf("snapshot should default to fixture evidence class: %+v", result)
	}
	if result.ProofEventIDs["E1_design_source_alignment.latest_source_alignment_proof_event_id"] != 10 ||
		result.ProofEventIDs["E9_areamatrix_protected_path_proof.latest_proof_event_id"] != 90 {
		t.Fatalf("snapshot missing proof event ids: %+v", result.ProofEventIDs)
	}
	if !containsString(result.Metadata["proof_evidence_uris"].([]string), "docs/development/source-alignment-release-candidate-evidence.md") ||
		!containsString(result.Metadata["proof_evidence_uris"].([]string), "docs/development/protected-path-release-candidate-evidence.md") {
		t.Fatalf("snapshot missing proof evidence uris: %+v", result.Metadata)
	}
	if result.ProjectWriteAttempted || result.ExecutionWriteAttempted || result.ReleasePackageCreated ||
		result.PublishAttempted || result.RestoreApplyAttempted || result.SecretResolved ||
		result.RemoteWorkerCredentialsIssued || result.AreaMatrixProtectedPathsTouched ||
		result.CommandsRun || result.SmokeRunAttempted || result.WorkerStarted {
		t.Fatalf("snapshot opened unsafe facts: %+v", result)
	}
}

func TestBuildCompletionAuditSnapshotRejectsUnknownEvidenceClass(t *testing.T) {
	_, err := buildCompletionAuditSnapshot(
		Record{ID: 1, Key: "areamatrix"},
		CompletionAudit{Status: "complete", Scope: "v1.0"},
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "realish",
		}),
		ReleaseEvidenceBundle{},
	)
	if err == nil {
		t.Fatal("expected unknown evidence class to be rejected")
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRejectsMechanismProofEvidence(t *testing.T) {
	audit := CompletionAudit{
		Status: "complete",
		Scope:  "v1.0",
		Items: []CompletionAuditItem{
			{
				Key:    "E1_design_source_alignment",
				Status: "complete",
				Metadata: map[string]any{
					"latest_source_alignment_proof_evidence_uri": "scripts/smoke-completion-audit-full-proof.sh#source-alignment",
				},
			},
		},
	}
	_, err := buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		audit,
		normalizeRecordCompletionAuditSnapshotOptions(RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to reject local script proof evidence")
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresRealProjectIdentity(t *testing.T) {
	record := realAreaMatrixRecord()
	record.RootPath = "/tmp/areaflow-completion-audit-rc.fake/areamatrix-root"
	_, err := buildCompletionAuditSnapshot(
		record,
		CompletionAudit{Status: "complete", Scope: "v1.0"},
		normalizeRecordCompletionAuditSnapshotOptions(RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to reject fixture project root")
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresRealEvidenceFields(t *testing.T) {
	record := realAreaMatrixRecord()
	audit := completionAuditWithReviewedProofEvidence()
	base := RecordCompletionAuditSnapshotOptions{
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		Summary:               "real release candidate evidence reviewed",
		ReviewDecision:        "approved",
		ReviewedBy:            "release-owner",
		ReviewedAt:            time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
	}

	for name, options := range map[string]RecordCompletionAuditSnapshotOptions{
		"missing evidence uri": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			Summary:               base.Summary,
		},
		"missing summary": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           base.EvidenceURI,
		},
		"fixture label": {
			ReleaseCandidateLabel: "v1.0-fixture",
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           base.EvidenceURI,
			Summary:               base.Summary,
		},
		"synthetic label": {
			ReleaseCandidateLabel: "v1.0-synthetic",
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           base.EvidenceURI,
			Summary:               base.Summary,
		},
		"fixture evidence uri": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           "scripts/smoke-completion-audit-full-proof.sh#fixture",
			Summary:               base.Summary,
		},
		"mock evidence uri": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           "docs/development/mock-release-candidate-evidence.md",
			Summary:               base.Summary,
		},
		"local smoke script evidence uri": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           "scripts/smoke-completion-audit-full-proof.sh#completion-audit",
			Summary:               base.Summary,
		},
		"local scheme evidence uri": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           "local:completion-audit",
			Summary:               base.Summary,
		},
		"generic mechanism evidence doc": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           "docs/development/completion-audit-evidence.md#release-candidate-review",
			Summary:               base.Summary,
		},
		"fixture summary": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           base.EvidenceURI,
			Summary:               "fixture evidence reviewed",
		},
		"placeholder summary": {
			ReleaseCandidateLabel: base.ReleaseCandidateLabel,
			EvidenceClass:         base.EvidenceClass,
			EvidenceURI:           base.EvidenceURI,
			Summary:               "placeholder evidence reviewed",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildCompletionAuditSnapshot(record, audit, normalizeReleaseCandidateSnapshotOptions(t, options), readyReleaseEvidenceBundle())
			if err == nil {
				t.Fatal("expected release_candidate snapshot to reject weak evidence fields")
			}
		})
	}

	bundle := readyReleaseEvidenceBundle()
	result, err := buildCompletionAuditSnapshot(record, audit, normalizeReleaseCandidateSnapshotOptions(t, base), bundle)
	if err != nil {
		t.Fatalf("valid release_candidate snapshot rejected: %v", err)
	}
	if result.EvidenceClass != "release_candidate" ||
		result.Metadata["release_candidate_snapshot"] != true ||
		result.Metadata["fixture_snapshot"] != false {
		t.Fatalf("release candidate metadata missing: %+v", result)
	}
	if result.Metadata["release_evidence_bundle_hash"] != bundle.BundleHash ||
		result.Metadata["release_evidence_bundle_status"] != "ready" ||
		result.Metadata["release_evidence_bundle_mode"] != "read_only_release_evidence_bundle" ||
		result.Metadata["release_evidence_bundle_item_count"] != len(bundle.Items) ||
		result.Metadata["release_evidence_bundle_ready"] != true {
		t.Fatalf("release candidate snapshot missing release evidence bundle binding: %+v", result.Metadata)
	}
	if result.Metadata["review_decision"] != "approved" ||
		result.Metadata["reviewed_by"] != "release-owner" ||
		result.Metadata["reviewed_at"] != "2026-07-04T12:00:00Z" ||
		result.Metadata["review_metadata_status"] != "approved" {
		t.Fatalf("release candidate snapshot missing approved review metadata: %+v", result.Metadata)
	}
	proofEvidenceURIMap, ok := result.Metadata["proof_evidence_uri_map"].(map[string]string)
	if !ok || len(proofEvidenceURIMap) != len(completionAuditSnapshotRequiredProofEvidenceURIKeys()) {
		t.Fatalf("release candidate snapshot missing proof evidence uri map: %+v", result.Metadata)
	}
	if result.Metadata["proof_evidence_uri_count"] != len(completionAuditSnapshotRequiredProofEvidenceURIKeys()) {
		t.Fatalf("release candidate snapshot missing proof evidence uri count: %+v", result.Metadata)
	}
	if result.Metadata["proof_event_id_count"] != len(completionAuditSnapshotRequiredProofEventIDKeys()) {
		t.Fatalf("release candidate snapshot missing proof event id count: %+v", result.Metadata)
	}
	requiredProofEventIDKeys, ok := result.Metadata["required_proof_event_id_keys"].([]string)
	if !ok || len(requiredProofEventIDKeys) != len(completionAuditSnapshotRequiredProofEventIDKeys()) {
		t.Fatalf("release candidate snapshot missing required proof event id keys: %+v", result.Metadata)
	}
	proofProvenanceMap, ok := result.Metadata["proof_provenance_map"].(map[string]string)
	if !ok || proofProvenanceMap["E7_operations_readiness.latest_operations_smoke_proof_key"] != "manual_ops_smoke_review" {
		t.Fatalf("release candidate snapshot missing proof provenance map: %+v", result.Metadata)
	}
	fileAudit, ok := result.Metadata["evidence_uri_file_audit"].([]map[string]any)
	if !ok || len(fileAudit) != len(completionAuditSnapshotRequiredProofEvidenceURIKeys())+1 {
		t.Fatalf("release candidate snapshot missing evidence URI file audit: %+v", result.Metadata)
	}
	if result.Metadata["evidence_uri_file_audit_status"] != "pass" ||
		result.Metadata["evidence_uri_file_audit_count"] != len(fileAudit) {
		t.Fatalf("release candidate snapshot missing evidence URI file audit status: %+v", result.Metadata)
	}
	for _, entry := range fileAudit {
		if metadataString(entry, "uri") == "" ||
			metadataString(entry, "path") != "docs/development/real-release-candidate-evidence.md" ||
			len(metadataString(entry, "sha256")) != 64 ||
			metadataInt64(entry, "size_bytes") == 0 {
			t.Fatalf("release candidate snapshot file audit entry incomplete: %+v", entry)
		}
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresApprovedReviewMetadata(t *testing.T) {
	base := RecordCompletionAuditSnapshotOptions{
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		Summary:               "real release candidate evidence reviewed",
		ReviewDecision:        "approved",
		ReviewedBy:            "release-owner",
		ReviewedAt:            time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
	}

	for name, tc := range map[string]struct {
		options     RecordCompletionAuditSnapshotOptions
		wantBlocker string
	}{
		"missing decision": {
			options: func() RecordCompletionAuditSnapshotOptions {
				options := base
				options.ReviewDecision = ""
				return options
			}(),
			wantBlocker: "snapshot_review_decision_missing",
		},
		"not approved": {
			options: func() RecordCompletionAuditSnapshotOptions {
				options := base
				options.ReviewDecision = "needs_changes"
				return options
			}(),
			wantBlocker: "snapshot_review_decision_not_approved",
		},
		"missing reviewer": {
			options: func() RecordCompletionAuditSnapshotOptions {
				options := base
				options.ReviewedBy = ""
				return options
			}(),
			wantBlocker: "snapshot_reviewed_by_missing",
		},
		"missing reviewed at": {
			options: func() RecordCompletionAuditSnapshotOptions {
				options := base
				options.ReviewedAt = time.Time{}
				return options
			}(),
			wantBlocker: "snapshot_reviewed_at_missing",
		},
		"invalid reviewed at metadata": {
			options: RecordCompletionAuditSnapshotOptions{
				ReleaseCandidateLabel: "v1.0-rc1",
				EvidenceClass:         "release_candidate",
				EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
				Summary:               "real release candidate evidence reviewed",
				Metadata: map[string]any{
					"review_decision": "approved",
					"reviewed_by":     "release-owner",
					"reviewed_at":     "not-a-time",
				},
			},
			wantBlocker: "snapshot_reviewed_at_invalid",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildCompletionAuditSnapshot(
				realAreaMatrixRecord(),
				completionAuditWithReviewedProofEvidence(),
				normalizeRecordCompletionAuditSnapshotOptions(tc.options),
				readyReleaseEvidenceBundle(),
			)
			if err == nil {
				t.Fatal("expected release_candidate snapshot to require approved review metadata")
			}
			if !strings.Contains(err.Error(), "requires approved review metadata") ||
				!strings.Contains(err.Error(), tc.wantBlocker) {
				t.Fatalf("review metadata blocker %s missing, got %v", tc.wantBlocker, err)
			}
		})
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresEvidenceURIFileAudit(t *testing.T) {
	base := RecordCompletionAuditSnapshotOptions{
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		Summary:               "real release candidate evidence reviewed",
		ReviewDecision:        "approved",
		ReviewedBy:            "release-owner",
		ReviewedAt:            time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
	}

	missingFile := base
	missingFile.EvidenceRoot = t.TempDir()
	_, err := buildCompletionAuditSnapshot(realAreaMatrixRecord(), completionAuditWithReviewedProofEvidence(), normalizeRecordCompletionAuditSnapshotOptions(missingFile), readyReleaseEvidenceBundle())
	if err == nil || !strings.Contains(err.Error(), "requires local evidence URI file audit") ||
		!strings.Contains(err.Error(), "evidence_uri_file_missing:docs/development/real-release-candidate-evidence.md") {
		t.Fatalf("missing evidence file should block release candidate snapshot, got %v", err)
	}

	missingAnchor := base
	missingAnchor.EvidenceRoot = t.TempDir()
	writeReleaseCandidateEvidenceFile(t, missingAnchor.EvidenceRoot, []string{"release-candidate-reviewed", "e1-source-alignment"})
	_, err = buildCompletionAuditSnapshot(realAreaMatrixRecord(), completionAuditWithReviewedProofEvidence(), normalizeRecordCompletionAuditSnapshotOptions(missingAnchor), readyReleaseEvidenceBundle())
	if err == nil || !strings.Contains(err.Error(), "requires local evidence URI file audit") ||
		!strings.Contains(err.Error(), "evidence_uri_anchor_not_found:docs/development/real-release-candidate-evidence.md#e2-task-matrix") {
		t.Fatalf("missing evidence anchor should block release candidate snapshot, got %v", err)
	}
}

func TestBuildCompletionAuditSnapshotReadinessRejectsEvidenceFileDrift(t *testing.T) {
	root := releaseCandidateEvidenceRoot(t)
	options := normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		Summary:               "real release candidate evidence reviewed",
		EvidenceRoot:          root,
	})
	result, err := buildCompletionAuditSnapshot(realAreaMatrixRecord(), completionAuditWithReviewedProofEvidence(), options, readyReleaseEvidenceBundle())
	if err != nil {
		t.Fatalf("build release candidate snapshot failed: %v", err)
	}
	if err := os.WriteFile(releaseCandidateEvidencePath(root), []byte("# Real Release Candidate Evidence\n\n## e1-source-alignment\n\ndrifted\n"), 0o644); err != nil {
		t.Fatalf("drift release candidate evidence fixture: %v", err)
	}

	readiness := buildCompletionAuditSnapshotReadiness(
		realAreaMatrixRecord(),
		result,
		true,
		readyReleaseEvidenceBundle(),
		readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, result.AuditHash, root),
	)
	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_evidence_uri_file_audit_mismatch" {
		t.Fatalf("evidence file drift should block readiness: %+v", readiness)
	}
	wantBlocker := "snapshot_evidence_uri_file_audit_sha256_mismatch:docs/development/real-release-candidate-evidence.md"
	blockers, ok := readiness.Items[0].Metadata["evidence_uri_file_audit_blockers"].([]string)
	if !ok || !containsString(blockers, wantBlocker) {
		t.Fatalf("evidence file drift blocker missing: %+v", readiness.Items[0].Metadata)
	}
	gaps := CompletionAuditSnapshotReadinessGaps(readiness)
	if len(gaps) != 1 || gaps[0].Category != "evidence_file_audit" ||
		!containsString(gaps[0].EvidenceURIFileAuditBlockers, wantBlocker) ||
		!containsString(gaps[0].Blockers, wantBlocker) {
		t.Fatalf("evidence file drift gap missing blocker: %+v", gaps)
	}
	assertCompletionAuditSnapshotClosureGate(t, readiness, "evidence_file_audit", "mismatch", wantBlocker)
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRejectsFixtureOperationsProofKey(t *testing.T) {
	for name, proofKey := range map[string]string{
		"fixture":   "v1_stable_fixture_smoke",
		"synthetic": "synthetic_ops_review",
	} {
		t.Run(name, func(t *testing.T) {
			audit := completionAuditWithReviewedProofEvidence()
			for index := range audit.Items {
				if audit.Items[index].Key == "E7_operations_readiness" {
					audit.Items[index].Metadata["latest_operations_smoke_proof_key"] = proofKey
					break
				}
			}

			_, err := buildCompletionAuditSnapshot(
				realAreaMatrixRecord(),
				audit,
				normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
					ReleaseCandidateLabel: "v1.0-rc1",
					EvidenceClass:         "release_candidate",
					EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
					Summary:               "real release candidate evidence reviewed",
				}),
				readyReleaseEvidenceBundle(),
			)
			if err == nil {
				t.Fatal("expected release_candidate snapshot to reject weak operations proof key")
			}
			if !strings.Contains(err.Error(), "requires release-candidate proof provenance") ||
				!strings.Contains(err.Error(), "snapshot_operations_proof_key_fixture") {
				t.Fatalf("weak operations proof key should be rejected, got %v", err)
			}
		})
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresOperationsProofProvenance(t *testing.T) {
	audit := completionAuditWithReviewedProofEvidence()
	missingKey := "E7_operations_readiness.latest_operations_smoke_proof_key"
	for index := range audit.Items {
		if audit.Items[index].Key == "E7_operations_readiness" {
			delete(audit.Items[index].Metadata, "latest_operations_smoke_proof_key")
			break
		}
	}

	_, err := buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		audit,
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to require operations proof provenance")
	}
	if !strings.Contains(err.Error(), "requires release-candidate proof provenance") ||
		!strings.Contains(err.Error(), "snapshot_proof_provenance_missing:"+missingKey) {
		t.Fatalf("missing operations proof provenance should be rejected, got %v", err)
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresCompleteProofEvidenceURIs(t *testing.T) {
	audit := completionAuditWithReviewedProofEvidence()
	missingKey := "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri"
	for index := range audit.Items {
		if audit.Items[index].Key == "E4_areamatrix_dogfood_completion" {
			delete(audit.Items[index].Metadata, "latest_execution_cutover_proof_evidence_uri")
			break
		}
	}

	_, err := buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		audit,
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to require complete proof evidence URIs")
	}
	if !strings.Contains(err.Error(), "requires complete proof evidence URIs") ||
		!strings.Contains(err.Error(), missingKey) {
		t.Fatalf("missing proof evidence URI error should name %s, got %v", missingKey, err)
	}

	sharedURI := "docs/development/real-release-candidate-evidence.md#shared-proof"
	duplicateAudit := completionAuditWithReviewedProofEvidence()
	for index := range duplicateAudit.Items {
		for _, metadataKey := range completionAuditSnapshotRequiredProofEvidenceURIKeysForItem(duplicateAudit.Items[index].Key) {
			duplicateAudit.Items[index].Metadata[metadataKey] = sharedURI
		}
	}
	_, err = buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		duplicateAudit,
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to require distinct proof evidence URI bindings")
	}
	if !strings.Contains(err.Error(), "requires complete proof evidence URIs") ||
		!strings.Contains(err.Error(), "snapshot_proof_evidence_uri_not_distinct") {
		t.Fatalf("duplicate proof evidence URI error should report weak binding, got %v", err)
	}

	sampleAudit := completionAuditWithReviewedProofEvidence()
	for index := range sampleAudit.Items {
		if sampleAudit.Items[index].Key == "E3_command_api_smoke_evidence" {
			sampleAudit.Items[index].Metadata["latest_validation_proof_evidence_uri"] = "docs/development/sample-release-candidate-evidence.md#e3-validation"
			break
		}
	}
	_, err = buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		sampleAudit,
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to reject sample proof evidence URI")
	}
	if !strings.Contains(err.Error(), "cannot seal local script/smoke proof evidence") ||
		!strings.Contains(err.Error(), "docs/development/sample-release-candidate-evidence.md#e3-validation") {
		t.Fatalf("sample proof evidence URI should fail closed, got %v", err)
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresCompleteProofEventIDs(t *testing.T) {
	audit := completionAuditWithReviewedProofEvidence()
	missingKey := "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id"
	for index := range audit.Items {
		if audit.Items[index].Key == "E4_areamatrix_dogfood_completion" {
			delete(audit.Items[index].Metadata, "latest_execution_cutover_proof_event_id")
			break
		}
	}

	_, err := buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		audit,
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to require complete proof event IDs")
	}
	if !strings.Contains(err.Error(), "requires complete proof event IDs") ||
		!strings.Contains(err.Error(), missingKey) {
		t.Fatalf("missing proof event ID error should name %s, got %v", missingKey, err)
	}

	duplicateAudit := completionAuditWithReviewedProofEvidence()
	for index := range duplicateAudit.Items {
		for _, metadataKey := range completionAuditSnapshotRequiredProofEventIDKeysForItem(duplicateAudit.Items[index].Key) {
			duplicateAudit.Items[index].Metadata[metadataKey] = int64(101)
		}
	}
	_, err = buildCompletionAuditSnapshot(
		realAreaMatrixRecord(),
		duplicateAudit,
		normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceClass:         "release_candidate",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
			Summary:               "real release candidate evidence reviewed",
		}),
		readyReleaseEvidenceBundle(),
	)
	if err == nil {
		t.Fatal("expected release_candidate snapshot to require distinct proof event ID bindings")
	}
	if !strings.Contains(err.Error(), "requires complete proof event IDs") ||
		!strings.Contains(err.Error(), "snapshot_proof_event_id_not_distinct") {
		t.Fatalf("duplicate proof event ID error should report weak binding, got %v", err)
	}
}

func TestBuildCompletionAuditSnapshotReleaseCandidateRequiresReleaseEvidenceBundle(t *testing.T) {
	record := realAreaMatrixRecord()
	audit := CompletionAudit{Status: "complete", Scope: "v1.0"}
	options := normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		Summary:               "real release candidate evidence reviewed",
	})

	driftRecord := realAreaMatrixRecord()
	driftRecord.RootPath = "/tmp/areaflow-completion-audit-rc.fake/areamatrix-root"

	for name, tc := range map[string]struct {
		bundle      ReleaseEvidenceBundle
		wantBlocker string
	}{
		"missing bundle": {
			bundle:      ReleaseEvidenceBundle{},
			wantBlocker: "release_evidence_bundle_hash_missing",
		},
		"blocked bundle": {
			bundle: BuildReleaseEvidenceBundle(
				ReleaseFinalGate{Status: "blocked"},
				BackupManifest{Status: "ready", SchemaVersion: 1, Projects: []BackupProjectManifest{{Project: realAreaMatrixRecord()}}},
				AuditCoverage{Status: "pass", Scope: "platform"},
				ReleaseEvidenceBundleOptions{},
			),
			wantBlocker: "release_evidence_bundle_not_ready",
		},
		"missing project inventory": {
			bundle: BuildReleaseEvidenceBundle(
				ReleaseFinalGate{Status: "pass"},
				BackupManifest{Status: "ready", SchemaVersion: 1},
				AuditCoverage{Status: "pass", Scope: "platform"},
				ReleaseEvidenceBundleOptions{},
			),
			wantBlocker: "release_evidence_bundle_project_inventory_missing",
		},
		"project inventory identity drift": {
			bundle: BuildReleaseEvidenceBundle(
				ReleaseFinalGate{Status: "pass"},
				BackupManifest{Status: "ready", SchemaVersion: 1, Projects: []BackupProjectManifest{{Project: driftRecord}}},
				AuditCoverage{Status: "pass", Scope: "platform"},
				ReleaseEvidenceBundleOptions{},
			),
			wantBlocker: "release_evidence_bundle_project_root_not_real_areamatrix",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildCompletionAuditSnapshot(record, audit, options, tc.bundle)
			if err == nil {
				t.Fatal("expected release_candidate snapshot to require ready release evidence bundle")
			}
			if !strings.Contains(err.Error(), tc.wantBlocker) {
				t.Fatalf("expected release evidence bundle blocker %s, got %v", tc.wantBlocker, err)
			}
		})
	}
}

func TestBuildCompletionAuditSnapshotReadinessDistinguishesFixtureAndReleaseCandidate(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	missing := buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, bundle, completionAuditSnapshotCurrentAuditBinding{Status: "complete", Scope: "v1.0", Hash: "current-audit-hash"})
	if missing.Status != "blocked" || len(missing.Items) != 1 || missing.Items[0].Key != "completion_audit_snapshot_missing" {
		t.Fatalf("missing snapshot readiness should be blocked: %+v", missing)
	}
	missingMetadata := missing.Items[0].Metadata
	requiredProofKeys, ok := missingMetadata["required_proof_evidence_uri_keys"].([]string)
	if !ok || !containsString(requiredProofKeys, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri") {
		t.Fatalf("missing snapshot readiness should expose required proof URI keys: %+v", missingMetadata)
	}
	requiredProofEventIDs, ok := missingMetadata["required_proof_event_id_keys"].([]string)
	if !ok || !containsString(requiredProofEventIDs, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id") {
		t.Fatalf("missing snapshot readiness should expose required proof event ID keys: %+v", missingMetadata)
	}
	currentMissingProofURIs, ok := missingMetadata["current_missing_proof_evidence_uri_keys"].([]string)
	if !ok || !containsString(currentMissingProofURIs, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri") {
		t.Fatalf("missing snapshot readiness should expose current missing proof URI keys: %+v", missingMetadata)
	}
	currentMissingProofEventIDs, ok := missingMetadata["current_missing_proof_event_id_keys"].([]string)
	if !ok || !containsString(currentMissingProofEventIDs, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id") {
		t.Fatalf("missing snapshot readiness should expose current missing proof event ID keys: %+v", missingMetadata)
	}
	currentMissingProofProvenance, ok := missingMetadata["current_missing_proof_provenance_keys"].([]string)
	if !ok || !containsString(currentMissingProofProvenance, "E7_operations_readiness.latest_operations_smoke_proof_key") {
		t.Fatalf("missing snapshot readiness should expose current missing proof provenance keys: %+v", missingMetadata)
	}
	currentProofURIBlockers, ok := missingMetadata["current_proof_evidence_uri_blockers"].([]string)
	if !ok || !containsString(currentProofURIBlockers, "current_proof_evidence_uri_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri") {
		t.Fatalf("missing snapshot readiness should expose current proof URI blockers: %+v", missingMetadata)
	}
	currentProofEventIDBlockers, ok := missingMetadata["current_proof_event_id_blockers"].([]string)
	if !ok || !containsString(currentProofEventIDBlockers, "current_proof_event_id_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id") {
		t.Fatalf("missing snapshot readiness should expose current proof event ID blockers: %+v", missingMetadata)
	}
	currentProofProvenanceBlockers, ok := missingMetadata["current_proof_provenance_blockers"].([]string)
	if !ok || !containsString(currentProofProvenanceBlockers, "current_proof_provenance_missing:E7_operations_readiness.latest_operations_smoke_proof_key") {
		t.Fatalf("missing snapshot readiness should expose current proof provenance blockers: %+v", missingMetadata)
	}
	packageABlockers, ok := missingMetadata["package_a_status_projection_blockers"].([]string)
	if !ok ||
		!containsString(packageABlockers, "package_a_status_projection_not_stable") ||
		!containsString(packageABlockers, "completion_audit_snapshot_package_a_not_applied") ||
		!containsString(packageABlockers, "package_a_status_projection_not_written") {
		t.Fatalf("missing snapshot readiness should expose Package A blockers: %+v", missingMetadata)
	}
	if missingMetadata["current_audit_status"] != "complete" ||
		missingMetadata["current_audit_scope"] != "v1.0" ||
		missingMetadata["current_audit_hash"] != "current-audit-hash" ||
		missingMetadata["current_bundle_hash"] != bundle.BundleHash ||
		missingMetadata["current_bundle_status"] != "ready" ||
		missingMetadata["current_bundle_mode"] != "read_only_release_evidence_bundle" ||
		missingMetadata["current_bundle_item_count"] != len(bundle.Items) {
		t.Fatalf("missing snapshot readiness should expose current audit and bundle binding hints: %+v", missingMetadata)
	}
	missingGaps := CompletionAuditSnapshotReadinessGaps(missing)
	if len(missingGaps) != 1 ||
		missingGaps[0].Key != "completion_audit_snapshot_missing" ||
		missingGaps[0].Category != "snapshot" ||
		!containsString(missingGaps[0].MissingProofEvidenceURIKeys, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri") ||
		!containsString(missingGaps[0].MissingProofEventIDKeys, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id") ||
		!containsString(missingGaps[0].MissingProofProvenanceKeys, "E7_operations_readiness.latest_operations_smoke_proof_key") ||
		!containsString(missingGaps[0].ProofEvidenceURIBlockers, "current_proof_evidence_uri_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri") ||
		!containsString(missingGaps[0].ProofEventIDBlockers, "current_proof_event_id_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id") ||
		!containsString(missingGaps[0].ProofProvenanceBlockers, "current_proof_provenance_missing:E7_operations_readiness.latest_operations_smoke_proof_key") ||
		!containsString(missingGaps[0].PackageAStatusProjectionBlockers, "completion_audit_snapshot_package_a_not_applied") {
		t.Fatalf("missing snapshot readiness gaps should expose current proof blockers: %+v", missingGaps)
	}
	missingClosure := CompletionAuditSnapshotReadinessClosure(missing)
	if missingClosure.Ready ||
		missingClosure.ReadyForReleaseCandidateClosure ||
		missingClosure.Status != "blocked" ||
		missingClosure.SnapshotStatus != "missing" ||
		missingClosure.Snapshot.Status != "missing" ||
		missingClosure.Snapshot.Ready ||
		missingClosure.ProofEvidenceURIStatus != "missing" ||
		missingClosure.ProofEvidenceURIs.Status != "missing" ||
		missingClosure.ProofEventIDStatus != "missing" ||
		missingClosure.ProofEventIDs.Status != "missing" ||
		missingClosure.ProofProvenanceStatus != "missing" ||
		missingClosure.ProofProvenance.Status != "missing" ||
		missingClosure.PackageAStatusProjectionStatus != "missing" ||
		missingClosure.PackageAStatusProjection.Status != "missing" ||
		missingClosure.ReleaseEvidenceBundleStatus != "pass" ||
		!missingClosure.ReleaseEvidenceBundle.Ready ||
		missingClosure.GapCount != 1 ||
		!containsString(missingClosure.GapKeys, "completion_audit_snapshot_missing") ||
		!containsString(missingClosure.Blockers, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id") ||
		!containsString(missingClosure.Blockers, "completion_audit_snapshot_package_a_not_applied") ||
		!containsString(missingClosure.MissingProofProvenanceKeys, "E7_operations_readiness.latest_operations_smoke_proof_key") ||
		missingClosure.CurrentAuditHash != "current-audit-hash" ||
		missingClosure.CurrentBundleHash != bundle.BundleHash ||
		missingClosure.CurrentBundleStatus != "ready" {
		t.Fatalf("missing snapshot closure should expose release-candidate blockers: %+v", missingClosure)
	}

	fixture := buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "fixture-hash",
		ReleaseCandidateLabel: "v1.0-fixture",
		EvidenceClass:         "fixture",
		EvidenceURI:           "scripts/smoke-completion-audit-full-proof.sh",
		EventID:               10,
	}, true, bundle)
	if fixture.Status != "blocked" || fixture.Items[0].Key != "completion_audit_snapshot_fixture_only" {
		t.Fatalf("fixture snapshot readiness should be blocked: %+v", fixture)
	}
	if fixture.Items[0].Metadata["fixture_snapshot"] != true ||
		fixture.Items[0].Metadata["release_candidate_snapshot"] != false {
		t.Fatalf("fixture snapshot metadata missing: %+v", fixture.Items[0].Metadata)
	}
	fixtureClosure := CompletionAuditSnapshotReadinessClosure(fixture)
	if fixtureClosure.Ready ||
		fixtureClosure.ReadyForReleaseCandidateClosure ||
		fixtureClosure.SnapshotStatus != "fixture_only" ||
		fixtureClosure.Snapshot.Status != "fixture_only" ||
		fixtureClosure.Snapshot.Ready ||
		fixtureClosure.ProofEvidenceURIStatus != "pass" ||
		!fixtureClosure.ProofEvidenceURIs.Ready ||
		!containsString(fixtureClosure.Blockers, "completion_audit_snapshot_fixture_only") {
		t.Fatalf("fixture snapshot closure should block as fixture-only: %+v", fixtureClosure)
	}

	evidenceRoot := releaseCandidateEvidenceRoot(t)
	releaseCandidate := buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "rc-hash",
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		EventID:               11,
		ProofEventIDs:         readyProofEventIDs(),
		Metadata:              readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot),
	}, true, bundle, readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, "rc-hash", evidenceRoot))
	if releaseCandidate.Status != "ready" ||
		releaseCandidate.Items[0].Key != "completion_audit_snapshot_release_candidate_present" {
		t.Fatalf("release candidate snapshot readiness should be ready: %+v", releaseCandidate)
	}
	if releaseCandidate.Items[0].Metadata["fixture_snapshot"] != false ||
		releaseCandidate.Items[0].Metadata["release_candidate_snapshot"] != true ||
		releaseCandidate.Items[0].Metadata["latest_summary_present"] != true ||
		releaseCandidate.Items[0].Metadata["current_audit_status"] != "complete" ||
		releaseCandidate.Items[0].Metadata["current_audit_scope"] != "v1.0" ||
		releaseCandidate.Items[0].Metadata["current_audit_hash"] != "rc-hash" ||
		releaseCandidate.Items[0].Metadata["audit_hash_match"] != true ||
		releaseCandidate.Items[0].Metadata["latest_review_decision"] != "approved" ||
		releaseCandidate.Items[0].Metadata["latest_reviewed_by"] != "release-owner" ||
		releaseCandidate.Items[0].Metadata["latest_reviewed_at"] != "2026-07-04T12:00:00Z" ||
		releaseCandidate.Items[0].Metadata["latest_bundle_hash"] != bundle.BundleHash ||
		releaseCandidate.Items[0].Metadata["current_bundle_hash"] != bundle.BundleHash {
		t.Fatalf("release candidate metadata missing: %+v", releaseCandidate.Items[0].Metadata)
	}
	releaseCandidateClosure := CompletionAuditSnapshotReadinessClosure(releaseCandidate)
	if !releaseCandidateClosure.Ready ||
		!releaseCandidateClosure.ReadyForReleaseCandidateClosure ||
		releaseCandidateClosure.Status != "ready" ||
		releaseCandidateClosure.SnapshotStatus != "release_candidate_present" ||
		!releaseCandidateClosure.Snapshot.Ready ||
		releaseCandidateClosure.ProofEvidenceURIStatus != "pass" ||
		!releaseCandidateClosure.ProofEvidenceURIs.Ready ||
		releaseCandidateClosure.ProofEventIDStatus != "pass" ||
		!releaseCandidateClosure.ProofEventIDs.Ready ||
		releaseCandidateClosure.ProofProvenanceStatus != "pass" ||
		!releaseCandidateClosure.ProofProvenance.Ready ||
		releaseCandidateClosure.ReleaseEvidenceBundleStatus != "pass" ||
		!releaseCandidateClosure.ReleaseEvidenceBundle.Ready ||
		releaseCandidateClosure.PackageAStatusProjectionStatus != "pass" ||
		!releaseCandidateClosure.PackageAStatusProjection.Ready ||
		releaseCandidateClosure.ReviewMetadataStatus != "pass" ||
		!releaseCandidateClosure.ReviewMetadata.Ready ||
		releaseCandidateClosure.GapCount != 0 ||
		len(releaseCandidateClosure.Blockers) != 0 {
		t.Fatalf("release candidate closure should be ready: %+v", releaseCandidateClosure)
	}
}

func TestBuildCompletionAuditSnapshotReadinessRejectsFixtureProjectIdentity(t *testing.T) {
	record := realAreaMatrixRecord()
	record.RootPath = "/tmp/areaflow-completion-audit-rc.fake/areamatrix-root"
	readiness := buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, readyReleaseEvidenceBundle())

	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_real_project_identity_missing" {
		t.Fatalf("fixture identity readiness should be blocked: %+v", readiness)
	}
	if readiness.Items[0].Metadata["real_project_identity_ready"] != false {
		t.Fatalf("real identity metadata missing: %+v", readiness.Items[0].Metadata)
	}
	blockers, ok := readiness.Items[0].Metadata["identity_blockers"].([]string)
	if !ok || !containsString(blockers, "project_root_not_real_areamatrix") {
		t.Fatalf("identity blockers metadata missing: %+v", readiness.Items[0].Metadata)
	}
}

func TestBuildCompletionAuditSnapshotReadinessRejectsNonTargetProject(t *testing.T) {
	record := Record{ID: 99, Key: "areamatrix-fixture"}
	readiness := buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, readyReleaseEvidenceBundle())

	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_project_mismatch" {
		t.Fatalf("non-target snapshot readiness should be blocked by project mismatch: %+v", readiness)
	}
	if readiness.Items[0].Metadata["expected_project_key"] != completionAuditTargetProjectKey ||
		readiness.Items[0].Metadata["actual_project_key"] != "areamatrix-fixture" {
		t.Fatalf("project mismatch metadata missing: %+v", readiness.Items[0].Metadata)
	}
}

func TestBuildCompletionAuditSnapshotReadinessRejectsMalformedReleaseCandidate(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	evidenceRoot := releaseCandidateEvidenceRoot(t)
	currentBinding := readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, "rc-hash", evidenceRoot)
	base := CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "rc-hash",
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		EventID:               11,
		ProofEventIDs:         readyProofEventIDs(),
		Metadata:              readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot),
	}

	for name, tc := range map[string]struct {
		snapshot           CompletionAuditSnapshot
		wantKey            string
		wantBlocker        string
		wantClosureGate    string
		wantClosureStatus  string
		wantClosureBlocker string
	}{
		"missing audit identity": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.AuditStatus = ""
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_audit_identity_invalid",
			wantClosureGate:    "audit_binding",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "completion_audit_snapshot_audit_identity_invalid",
		},
		"missing summary": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.Metadata = map[string]any{}
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_summary_missing",
			wantClosureGate:    "snapshot_evidence",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "completion_audit_snapshot_summary_missing",
		},
		"unsafe side effects": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.ProjectWriteAttempted = true
				snapshot.AreaMatrixProtectedPathsTouched = true
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_unsafe_side_effects",
			wantClosureGate:    "safety",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "project_write_attempted",
		},
		"fixture marker": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.EvidenceURI = "docs/development/fixture-release-candidate-evidence.md"
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_fixture_labeled_release_candidate",
			wantClosureGate:    "snapshot_evidence",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "completion_audit_snapshot_fixture_labeled_release_candidate",
		},
		"placeholder marker": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.EvidenceURI = "docs/development/placeholder-release-candidate-evidence.md"
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_fixture_labeled_release_candidate",
			wantClosureGate:    "snapshot_evidence",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "completion_audit_snapshot_fixture_labeled_release_candidate",
		},
		"local smoke script evidence uri": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.EvidenceURI = "scripts/smoke-completion-audit-full-proof.sh#completion-audit"
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_mechanism_evidence_uri",
			wantClosureGate:    "snapshot_evidence",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "completion_audit_snapshot_mechanism_evidence_uri",
		},
		"local smoke proof evidence uri": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.Metadata = readyReleaseEvidenceBundleMetadataWithProofURIs(bundle, []string{"scripts/smoke-completion-audit-full-proof.sh#source-alignment"})
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_mechanism_proof_evidence_uri",
			wantClosureGate:    "proof_evidence_uris",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "scripts/smoke-completion-audit-full-proof.sh#source-alignment",
		},
		"generic mechanism evidence doc": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				snapshot.EvidenceURI = "docs/development/completion-audit-evidence.md#release-candidate-review"
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_generic_evidence_uri",
			wantClosureGate:    "snapshot_evidence",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "completion_audit_snapshot_generic_evidence_uri",
		},
		"missing approved review metadata": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				delete(metadata, "review_decision")
				delete(metadata, "review_metadata_status")
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_review_metadata_missing",
			wantBlocker:        "snapshot_review_decision_missing",
			wantClosureGate:    "review_metadata",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "snapshot_review_decision_missing",
		},
		"generic mechanism proof evidence doc": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadata(bundle)
				uriMap := metadata["proof_evidence_uri_map"].(map[string]string)
				uriMap["E7_operations_readiness.latest_operations_smoke_proof_evidence_uri"] = "docs/development/operations-readiness-evidence.md#release-candidate-review"
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_evidence_uri_missing",
			wantBlocker:        "snapshot_proof_evidence_uri_not_release_candidate:E7_operations_readiness.latest_operations_smoke_proof_evidence_uri",
			wantClosureGate:    "proof_evidence_uris",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "snapshot_proof_evidence_uri_not_release_candidate:E7_operations_readiness.latest_operations_smoke_proof_evidence_uri",
		},
		"missing proof evidence uri map key": {
			snapshot: func() CompletionAuditSnapshot {
				missingKey := "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri"
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadata(bundle)
				uriMap := metadata["proof_evidence_uri_map"].(map[string]string)
				delete(uriMap, missingKey)
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_evidence_uri_missing",
			wantBlocker:        "snapshot_proof_evidence_uri_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri",
			wantClosureGate:    "proof_evidence_uris",
			wantClosureStatus:  "missing",
			wantClosureBlocker: "snapshot_proof_evidence_uri_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri",
		},
		"missing proof event id key": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				eventIDs := readyProofEventIDs()
				delete(eventIDs, "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id")
				snapshot.ProofEventIDs = eventIDs
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_event_id_missing",
			wantBlocker:        "snapshot_proof_event_id_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id",
			wantClosureGate:    "proof_event_ids",
			wantClosureStatus:  "missing",
			wantClosureBlocker: "snapshot_proof_event_id_missing:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id",
		},
		"duplicate proof evidence uri bindings": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadata(bundle)
				uriMap := metadata["proof_evidence_uri_map"].(map[string]string)
				sharedURI := "docs/development/real-release-candidate-evidence.md#shared-proof"
				for key := range uriMap {
					uriMap[key] = sharedURI
				}
				metadata["proof_evidence_uris"] = []string{sharedURI}
				metadata["proof_evidence_uri_count"] = 1
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_evidence_uri_missing",
			wantBlocker:        "snapshot_proof_evidence_uri_not_distinct",
			wantClosureGate:    "proof_evidence_uris",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "snapshot_proof_evidence_uri_not_distinct",
		},
		"duplicate proof event id bindings": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				eventIDs := readyProofEventIDs()
				for key := range eventIDs {
					eventIDs[key] = 101
				}
				snapshot.ProofEventIDs = eventIDs
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_event_id_missing",
			wantBlocker:        "snapshot_proof_event_id_not_distinct",
			wantClosureGate:    "proof_event_ids",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "snapshot_proof_event_id_not_distinct",
		},
		"proof event id metadata mismatch": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadata(bundle)
				eventIDs := readyProofEventIDs()
				eventIDs["E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id"] = 999
				metadata["proof_event_ids"] = eventIDs
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_event_id_missing",
			wantBlocker:        "snapshot_proof_event_id_metadata_mismatch:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id",
			wantClosureGate:    "proof_event_ids",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "snapshot_proof_event_id_metadata_mismatch:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id",
		},
		"fixture operations proof key": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadata(bundle)
				provenanceMap := metadata["proof_provenance_map"].(map[string]string)
				provenanceMap["E7_operations_readiness.latest_operations_smoke_proof_key"] = "v1_stable_fixture_smoke"
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_proof_provenance_missing",
			wantBlocker:        "snapshot_operations_proof_key_fixture",
			wantClosureGate:    "proof_provenance",
			wantClosureStatus:  "blocked",
			wantClosureBlocker: "snapshot_operations_proof_key_fixture",
		},
		"missing evidence file audit metadata": {
			snapshot: func() CompletionAuditSnapshot {
				snapshot := base
				metadata := readyReleaseEvidenceBundleMetadata(bundle)
				snapshot.Metadata = metadata
				return snapshot
			}(),
			wantKey:            "completion_audit_snapshot_evidence_uri_file_audit_mismatch",
			wantBlocker:        "snapshot_evidence_uri_file_audit_status_missing",
			wantClosureGate:    "evidence_file_audit",
			wantClosureStatus:  "mismatch",
			wantClosureBlocker: "snapshot_evidence_uri_file_audit_status_missing",
		},
	} {
		t.Run(name, func(t *testing.T) {
			readiness := buildCompletionAuditSnapshotReadiness(record, tc.snapshot, true, bundle, currentBinding)
			if readiness.Status != "blocked" || len(readiness.Items) != 1 || readiness.Items[0].Key != tc.wantKey {
				t.Fatalf("unexpected malformed release candidate readiness: %+v", readiness)
			}
			if readiness.Items[0].Metadata["release_candidate_snapshot"] != true {
				t.Fatalf("release candidate metadata missing: %+v", readiness.Items[0].Metadata)
			}
			if tc.wantKey == "completion_audit_snapshot_unsafe_side_effects" {
				unsafeFacts, ok := readiness.Items[0].Metadata["unsafe_facts"].([]string)
				if !ok || !containsString(unsafeFacts, "project_write_attempted") || !containsString(unsafeFacts, "area_matrix_protected_paths_touched") {
					t.Fatalf("unsafe facts metadata missing: %+v", readiness.Items[0].Metadata)
				}
			}
			if tc.wantBlocker != "" {
				blockers, ok := readiness.Items[0].Metadata["proof_evidence_uri_blockers"].([]string)
				if !ok {
					blockers, ok = readiness.Items[0].Metadata["proof_event_id_blockers"].([]string)
				}
				if !ok {
					blockers, ok = readiness.Items[0].Metadata["proof_provenance_blockers"].([]string)
				}
				if !ok {
					blockers, ok = readiness.Items[0].Metadata["evidence_uri_file_audit_blockers"].([]string)
				}
				if !ok {
					blockers, ok = readiness.Items[0].Metadata["review_metadata_blockers"].([]string)
				}
				if !ok || !containsString(blockers, tc.wantBlocker) {
					t.Fatalf("proof blocker missing %s: %+v", tc.wantBlocker, readiness.Items[0].Metadata)
				}
			}
			assertCompletionAuditSnapshotClosureGate(t, readiness, tc.wantClosureGate, tc.wantClosureStatus, tc.wantClosureBlocker)
		})
	}
}

func TestBuildCompletionAuditSnapshotReadinessRejectsCurrentProofBindingMismatch(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	evidenceRoot := releaseCandidateEvidenceRoot(t)
	baseBinding := func() completionAuditSnapshotCurrentAuditBinding {
		return readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, "rc-hash", evidenceRoot)
	}
	base := CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "rc-hash",
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		EventID:               11,
		ProofEventIDs:         readyProofEventIDs(),
		Metadata:              readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot),
	}

	for name, tc := range map[string]struct {
		current     completionAuditSnapshotCurrentAuditBinding
		wantBlocker string
	}{
		"proof evidence uri map mismatch": {
			current: func() completionAuditSnapshotCurrentAuditBinding {
				current := baseBinding()
				key := "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri"
				current.ProofEvidenceURIMap[key] = "docs/development/real-release-candidate-evidence.md#changed-execution-cutover"
				return current
			}(),
			wantBlocker: "snapshot_proof_evidence_uri_map_mismatch:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri",
		},
		"proof evidence uri set mismatch": {
			current: func() completionAuditSnapshotCurrentAuditBinding {
				current := baseBinding()
				current.ProofEvidenceURIs = current.ProofEvidenceURIs[:len(current.ProofEvidenceURIs)-1]
				return current
			}(),
			wantBlocker: "snapshot_proof_evidence_uri_set_mismatch:docs/development/real-release-candidate-evidence.md#e9-protected-path",
		},
		"proof event id map mismatch": {
			current: func() completionAuditSnapshotCurrentAuditBinding {
				current := baseBinding()
				key := "E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id"
				current.ProofEventIDs[key] = 999
				return current
			}(),
			wantBlocker: "snapshot_proof_event_id_map_mismatch:E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id",
		},
		"proof provenance map mismatch": {
			current: func() completionAuditSnapshotCurrentAuditBinding {
				current := baseBinding()
				key := "E7_operations_readiness.latest_operations_smoke_proof_key"
				current.ProofProvenanceMap[key] = "manual_ops_smoke_review_v2"
				return current
			}(),
			wantBlocker: "snapshot_proof_provenance_map_mismatch:E7_operations_readiness.latest_operations_smoke_proof_key",
		},
		"proof provenance map missing": {
			current: func() completionAuditSnapshotCurrentAuditBinding {
				current := baseBinding()
				current.ProofProvenanceMap = map[string]string{}
				return current
			}(),
			wantBlocker: "current_proof_provenance_map_missing",
		},
		"proof provenance key missing": {
			current: func() completionAuditSnapshotCurrentAuditBinding {
				current := baseBinding()
				delete(current.ProofProvenanceMap, "E7_operations_readiness.latest_operations_smoke_proof_key")
				return current
			}(),
			wantBlocker: "current_proof_provenance_missing:E7_operations_readiness.latest_operations_smoke_proof_key",
		},
	} {
		t.Run(name, func(t *testing.T) {
			readiness := buildCompletionAuditSnapshotReadiness(record, base, true, bundle, tc.current)
			if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
				readiness.Items[0].Key != "completion_audit_snapshot_current_proof_binding_mismatch" {
				t.Fatalf("current proof binding mismatch should block readiness: %+v", readiness)
			}
			blockers, ok := readiness.Items[0].Metadata["current_proof_binding_blockers"].([]string)
			if !ok || !containsString(blockers, tc.wantBlocker) {
				t.Fatalf("current proof binding blocker missing %s: %+v", tc.wantBlocker, readiness.Items[0].Metadata)
			}
			gaps := CompletionAuditSnapshotReadinessGaps(readiness)
			if len(gaps) != 1 || gaps[0].Category != "current_binding" ||
				!containsString(gaps[0].CurrentProofBindingBlockers, tc.wantBlocker) ||
				!containsString(gaps[0].Blockers, tc.wantBlocker) {
				t.Fatalf("current proof binding gap missing blocker %s: %+v", tc.wantBlocker, gaps)
			}
			assertCompletionAuditSnapshotClosureGate(t, readiness, "current_proof_binding", "mismatch", tc.wantBlocker)
		})
	}
}

func TestBuildCompletionAuditSnapshotReadinessRequiresPackageAStatusProjection(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	evidenceRoot := releaseCandidateEvidenceRoot(t)
	currentBinding := readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, "rc-hash", evidenceRoot)
	currentBinding.PackageAStatusProjection = completionAuditSnapshotPackageAStatusProjectionBinding{
		LatestImportSourceHash:  "source-hash-rc",
		CurrentPreimageCaptured: true,
		CurrentPreimage: StatusProjectionPreimage{
			SchemaStatus: "legacy",
			Exists:       true,
			Readable:     true,
			SHA256:       "legacy-status-hash",
			Message:      "target uses legacy status projection shape",
		},
	}
	snapshot := CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "rc-hash",
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		EventID:               11,
		ProofEventIDs:         readyProofEventIDs(),
		Metadata:              readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot),
	}

	readiness := buildCompletionAuditSnapshotReadiness(record, snapshot, true, bundle, currentBinding)
	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_package_a_not_applied" {
		t.Fatalf("missing Package A status projection should block readiness: %+v", readiness)
	}
	blockers, ok := readiness.Items[0].Metadata["package_a_status_projection_blockers"].([]string)
	if !ok ||
		!containsString(blockers, "package_a_status_projection_not_stable") ||
		!containsString(blockers, "completion_audit_snapshot_package_a_not_applied") ||
		!containsString(blockers, "package_a_status_projection_not_written") ||
		!containsString(blockers, "package_a_current_status_projection_not_stable") {
		t.Fatalf("Package A blockers missing: %+v", readiness.Items[0].Metadata)
	}
	gaps := CompletionAuditSnapshotReadinessGaps(readiness)
	if len(gaps) != 1 || gaps[0].Category != "package_a_status_projection" ||
		!containsString(gaps[0].PackageAStatusProjectionBlockers, "completion_audit_snapshot_package_a_not_applied") {
		t.Fatalf("Package A gap missing blockers: %+v", gaps)
	}
	assertCompletionAuditSnapshotClosureGate(t, readiness, "package_a_status_projection", "missing", "completion_audit_snapshot_package_a_not_applied")
}

func TestBuildCompletionAuditSnapshotReadinessAcceptsStableCurrentPackageAProjectionWithoutNotApplied(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	evidenceRoot := releaseCandidateEvidenceRoot(t)
	currentBinding := readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, "rc-hash", evidenceRoot)
	currentBinding.PackageAStatusProjection = stablePackageAStatusProjectionWithoutProvenanceBinding("source-hash-rc")
	snapshot := CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "rc-hash",
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		EventID:               11,
		ProofEventIDs:         readyProofEventIDs(),
		Metadata:              readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot),
	}

	readiness := buildCompletionAuditSnapshotReadiness(record, snapshot, true, bundle, currentBinding)
	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_package_a_apply_provenance_missing" {
		t.Fatalf("stable Package A status projection should only block on apply provenance: %+v", readiness)
	}
	blockers, ok := readiness.Items[0].Metadata["package_a_status_projection_blockers"].([]string)
	if !ok ||
		!containsString(blockers, "package_a_status_projection_apply_provenance_missing") ||
		containsString(blockers, "completion_audit_snapshot_package_a_not_applied") ||
		containsString(blockers, "package_a_status_projection_not_stable") ||
		containsString(blockers, "package_a_status_projection_not_written") {
		t.Fatalf("stable Package A projection blockers should distinguish apply provenance: %+v", readiness.Items[0].Metadata)
	}
	gaps := CompletionAuditSnapshotReadinessGaps(readiness)
	if len(gaps) != 1 || gaps[0].Category != "package_a_status_projection" ||
		!containsString(gaps[0].PackageAStatusProjectionBlockers, "package_a_status_projection_apply_provenance_missing") {
		t.Fatalf("Package A provenance gap missing blockers: %+v", gaps)
	}
	assertCompletionAuditSnapshotClosureGate(t, readiness, "package_a_status_projection", "blocked", "package_a_status_projection_apply_provenance_missing")
}

func TestBuildCompletionAuditSnapshotReadinessReal100UsesCurrentPackageAClosure(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	currentBinding := readyCompletionAuditSnapshotCurrentBinding("current-audit-hash")
	currentBinding.Real100Guardrail = CompletionAuditReal100GuardrailForItems([]CompletionAuditItem{
		{
			Key:      "E4_areamatrix_dogfood_completion",
			Category: "dogfood",
			Status:   "blocked",
			Message:  "AreaMatrix dogfood still needs real cutover",
			BlockedBy: []string{
				"real_areamatrix_read_only_shim_not_landed",
				"execution_cutover_not_complete",
			},
			Metadata: map[string]any{
				"package_a_status_projection_ready": true,
			},
		},
	})

	readiness := buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, bundle, currentBinding)

	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_missing" {
		t.Fatalf("missing snapshot should still block readiness: %+v", readiness)
	}
	if readiness.Items[0].Metadata["package_a_status_projection_ready"] != true {
		t.Fatalf("readiness should expose current Package A ready metadata: %+v", readiness.Items[0].Metadata)
	}
	blockers := readiness.Real100Guardrail.Real100Blockers
	if containsString(blockers, "package_a_status_projection_apply_provenance_missing") ||
		containsString(blockers, "package_a_status_projection_not_applied") {
		t.Fatalf("Package A ready snapshot readiness should not reintroduce Package A blockers: %+v", readiness.Real100Guardrail)
	}
	if !containsString(blockers, "release_candidate_snapshot_not_ready") ||
		!containsString(blockers, "real_areamatrix_read_only_shim_not_landed") ||
		!containsString(blockers, "real_areamatrix_execution_cutover_not_proven") {
		t.Fatalf("readiness should keep current non-Package-A blockers: %+v", readiness.Real100Guardrail)
	}
}

func TestCompletionAuditSnapshotReadinessReal100CompletesWhenSnapshotIsReady(t *testing.T) {
	readiness := CompletionAuditSnapshotReadiness{
		Status:        "ready",
		HasSnapshot:   true,
		RequiredClass: "release_candidate",
		Latest: CompletionAuditSnapshot{
			EvidenceClass:         "release_candidate",
			ReleaseCandidateLabel: "v1.0-rc1",
			EvidenceURI:           "docs/development/real-release-candidate-evidence.md#release-candidate-closure",
			AuditHash:             "audit-hash",
			EventID:               42,
		},
	}
	current := CompletionAuditReal100Guardrail()
	current.Real100Blockers = []string{"release_candidate_snapshot_not_ready"}

	guardrail := completionAuditSnapshotReadinessReal100Guardrail(readiness, current)
	if guardrail.Real100Status != Real100StatusComplete || len(guardrail.Real100Blockers) != 0 ||
		guardrail.NotReal100 || guardrail.EvidenceOnly || guardrail.StatusAloneIsNotCompletion ||
		guardrail.ReleaseCandidateDecision != "release_candidate_ready" {
		t.Fatalf("ready release candidate should complete real 100 guardrail: %+v", guardrail)
	}
}

func TestBuildCompletionAuditSnapshotReadinessRejectsCompletionAuditHashMismatch(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	snapshot := CompletionAuditSnapshot{
		Project:               record,
		Status:                "recorded",
		AuditStatus:           "complete",
		AuditScope:            "v1.0",
		AuditHash:             "old-audit-hash",
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		EventID:               11,
		ProofEventIDs:         readyProofEventIDs(),
		Metadata:              readyReleaseEvidenceBundleMetadata(bundle),
	}

	readiness := buildCompletionAuditSnapshotReadiness(record, snapshot, true, bundle, completionAuditSnapshotCurrentAuditBinding{Status: "complete", Scope: "v1.0", Hash: "current-audit-hash"})
	if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
		readiness.Items[0].Key != "completion_audit_snapshot_audit_hash_mismatch" {
		t.Fatalf("audit hash mismatch readiness should be blocked: %+v", readiness)
	}
	metadata := readiness.Items[0].Metadata
	if metadata["latest_audit_hash"] != "old-audit-hash" ||
		metadata["snapshot_audit_hash"] != "old-audit-hash" ||
		metadata["current_audit_status"] != "complete" ||
		metadata["current_audit_scope"] != "v1.0" ||
		metadata["current_audit_hash"] != "current-audit-hash" ||
		metadata["audit_hash_match"] != false {
		t.Fatalf("audit hash mismatch metadata missing hashes: %+v", metadata)
	}
	blockers, ok := metadata["audit_hash_blockers"].([]string)
	if !ok || !containsString(blockers, "snapshot_audit_hash_mismatch") {
		t.Fatalf("audit hash mismatch blocker missing: %+v", metadata)
	}
	assertCompletionAuditSnapshotClosureGate(t, readiness, "audit_binding", "mismatch", "completion_audit_snapshot_audit_hash_mismatch")
}

func TestBuildCompletionAuditSnapshotReadinessRejectsReleaseEvidenceBundleMismatch(t *testing.T) {
	record := realAreaMatrixRecord()
	bundle := readyReleaseEvidenceBundle()
	evidenceRoot := releaseCandidateEvidenceRoot(t)
	currentBinding := readyCompletionAuditSnapshotCurrentBindingWithEvidenceRoot(t, "rc-hash", evidenceRoot)
	blockedBundle := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "blocked", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			Projects: []BackupProjectManifest{
				{Project: realAreaMatrixRecord(), Inventory: ImportInventory{Versions: 1}, ArtifactCount: 1, Artifacts: []BackupArtifactSummary{{ID: 1}}},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9},
		ReleaseEvidenceBundleOptions{},
	)
	driftRecord := realAreaMatrixRecord()
	driftRecord.RootPath = "/tmp/areaflow-completion-audit-rc.fake/areamatrix-root"
	identityDriftBundle := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			Projects: []BackupProjectManifest{
				{Project: driftRecord, Inventory: ImportInventory{Versions: 1}, ArtifactCount: 1, Artifacts: []BackupArtifactSummary{{ID: 1}}},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9},
		ReleaseEvidenceBundleOptions{},
	)

	for name, tc := range map[string]struct {
		metadata      map[string]any
		currentBundle ReleaseEvidenceBundle
		wantBlocker   string
	}{
		"missing hash": {
			metadata: func() map[string]any {
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				delete(metadata, "release_evidence_bundle_hash")
				return metadata
			}(),
			currentBundle: bundle,
			wantBlocker:   "snapshot_release_evidence_bundle_hash_missing",
		},
		"hash mismatch": {
			metadata: func() map[string]any {
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				metadata["release_evidence_bundle_hash"] = "old-bundle-hash"
				return metadata
			}(),
			currentBundle: bundle,
			wantBlocker:   "snapshot_release_evidence_bundle_hash_mismatch",
		},
		"status mismatch": {
			metadata: func() map[string]any {
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				metadata["release_evidence_bundle_status"] = "blocked"
				return metadata
			}(),
			currentBundle: bundle,
			wantBlocker:   "snapshot_release_evidence_bundle_status_not_ready",
		},
		"mode mismatch": {
			metadata: func() map[string]any {
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				metadata["release_evidence_bundle_mode"] = "mutable_bundle"
				return metadata
			}(),
			currentBundle: bundle,
			wantBlocker:   "snapshot_release_evidence_bundle_mode_invalid",
		},
		"ready flag false": {
			metadata: func() map[string]any {
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				metadata["release_evidence_bundle_ready"] = false
				return metadata
			}(),
			currentBundle: bundle,
			wantBlocker:   "snapshot_release_evidence_bundle_ready_false",
		},
		"item count mismatch": {
			metadata: func() map[string]any {
				metadata := readyReleaseEvidenceBundleMetadataWithFileAudit(t, bundle, evidenceRoot)
				metadata["release_evidence_bundle_item_count"] = len(bundle.Items) + 1
				return metadata
			}(),
			currentBundle: bundle,
			wantBlocker:   "snapshot_release_evidence_bundle_item_count_mismatch",
		},
		"current bundle not ready": {
			metadata:      readyReleaseEvidenceBundleMetadataWithFileAudit(t, blockedBundle, evidenceRoot),
			currentBundle: blockedBundle,
			wantBlocker:   "release_evidence_bundle_not_ready",
		},
		"current bundle project inventory identity drift": {
			metadata:      readyReleaseEvidenceBundleMetadataWithFileAudit(t, identityDriftBundle, evidenceRoot),
			currentBundle: identityDriftBundle,
			wantBlocker:   "release_evidence_bundle_project_root_not_real_areamatrix",
		},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := CompletionAuditSnapshot{
				Project:               record,
				Status:                "recorded",
				AuditStatus:           "complete",
				AuditScope:            "v1.0",
				AuditHash:             "rc-hash",
				ReleaseCandidateLabel: "v1.0-rc1",
				EvidenceClass:         "release_candidate",
				EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
				EventID:               11,
				ProofEventIDs:         readyProofEventIDs(),
				Metadata:              tc.metadata,
			}

			readiness := buildCompletionAuditSnapshotReadiness(record, snapshot, true, tc.currentBundle, currentBinding)
			if readiness.Status != "blocked" || len(readiness.Items) != 1 ||
				readiness.Items[0].Key != "completion_audit_snapshot_release_evidence_bundle_mismatch" {
				t.Fatalf("bundle mismatch readiness should be blocked: %+v", readiness)
			}
			blockers, ok := readiness.Items[0].Metadata["bundle_blockers"].([]string)
			if !ok || !containsString(blockers, tc.wantBlocker) {
				t.Fatalf("bundle mismatch blockers missing %s: %+v", tc.wantBlocker, readiness.Items[0].Metadata)
			}
			assertCompletionAuditSnapshotClosureGate(t, readiness, "release_evidence_bundle", "mismatch", tc.wantBlocker)
		})
	}
}

func assertCompletionAuditSnapshotClosureGate(t *testing.T, readiness CompletionAuditSnapshotReadiness, gateName string, wantStatus string, wantBlocker string) {
	t.Helper()

	closure := CompletionAuditSnapshotReadinessClosure(readiness)
	if closure.Ready || closure.ReadyForReleaseCandidateClosure {
		t.Fatalf("closure unexpectedly ready for %s: %+v", gateName, closure)
	}
	if closure.Status != "blocked" {
		t.Fatalf("closure status for %s = %q, want blocked: %+v", gateName, closure.Status, closure)
	}
	if !containsString(closure.Blockers, wantBlocker) {
		t.Fatalf("closure blockers for %s missing %s: %+v", gateName, wantBlocker, closure.Blockers)
	}

	gate := completionAuditSnapshotClosureGateByName(t, closure, gateName)
	if gate.Status != wantStatus || gate.Ready {
		t.Fatalf("closure gate %s = status %q ready %t, want status %q and not ready: %+v", gateName, gate.Status, gate.Ready, wantStatus, gate)
	}
	if !containsString(gate.Blockers, wantBlocker) {
		t.Fatalf("closure gate %s blockers missing %s: %+v", gateName, wantBlocker, gate.Blockers)
	}
}

func completionAuditSnapshotClosureGateByName(t *testing.T, closure CompletionAuditSnapshotClosure, gateName string) CompletionAuditSnapshotClosureGate {
	t.Helper()

	switch gateName {
	case "audit_binding":
		return closure.AuditBinding
	case "snapshot_evidence":
		return closure.SnapshotEvidence
	case "proof_evidence_uris":
		return closure.ProofEvidenceURIs
	case "proof_event_ids":
		return closure.ProofEventIDs
	case "proof_provenance":
		return closure.ProofProvenance
	case "current_proof_binding":
		return closure.CurrentProofBinding
	case "release_evidence_bundle":
		return closure.ReleaseEvidenceBundle
	case "evidence_file_audit":
		return closure.EvidenceFileAudit
	case "package_a_status_projection":
		return closure.PackageAStatusProjection
	case "review_metadata":
		return closure.ReviewMetadata
	case "safety":
		return closure.Safety
	default:
		t.Fatalf("unknown closure gate %q", gateName)
		return CompletionAuditSnapshotClosureGate{}
	}
}

func TestCompletionAuditSnapshotRequestHashIncludesProofBindings(t *testing.T) {
	record := realAreaMatrixRecord()
	options := normalizeReleaseCandidateSnapshotOptions(t, RecordCompletionAuditSnapshotOptions{
		ReleaseCandidateLabel: "v1.0-rc1",
		EvidenceClass:         "release_candidate",
		EvidenceURI:           "docs/development/real-release-candidate-evidence.md",
		Summary:               "real release candidate evidence reviewed",
	})
	result, err := buildCompletionAuditSnapshot(record, completionAuditWithReviewedProofEvidence(), options, readyReleaseEvidenceBundle())
	if err != nil {
		t.Fatalf("build release candidate snapshot failed: %v", err)
	}
	base, err := completionAuditSnapshotRequestHash(record, result, options)
	if err != nil {
		t.Fatalf("hash base request failed: %v", err)
	}

	changedURI := result
	changedURI.Metadata = map[string]any{}
	for key, value := range result.Metadata {
		changedURI.Metadata[key] = value
	}
	changedURIMap := readyProofEvidenceURIMap()
	changedURIMap["E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_evidence_uri"] = "docs/development/real-release-candidate-evidence.md#changed-execution-cutover"
	changedURI.Metadata["proof_evidence_uri_map"] = changedURIMap
	changedURIHash, err := completionAuditSnapshotRequestHash(record, changedURI, options)
	if err != nil {
		t.Fatalf("hash changed uri request failed: %v", err)
	}
	if changedURIHash == base {
		t.Fatal("request hash should change when proof evidence URI map changes")
	}

	changedID := result
	changedID.ProofEventIDs = readyProofEventIDs()
	changedID.ProofEventIDs["E4_areamatrix_dogfood_completion.latest_execution_cutover_proof_event_id"] = 999
	changedIDHash, err := completionAuditSnapshotRequestHash(record, changedID, options)
	if err != nil {
		t.Fatalf("hash changed event ID request failed: %v", err)
	}
	if changedIDHash == base {
		t.Fatal("request hash should change when proof event IDs change")
	}

	changedProvenance := result
	changedProvenance.Metadata = map[string]any{}
	for key, value := range result.Metadata {
		changedProvenance.Metadata[key] = value
	}
	changedProvenanceMap := readyProofProvenanceMap()
	changedProvenanceMap["E7_operations_readiness.latest_operations_smoke_proof_key"] = "manual_ops_smoke_review_v2"
	changedProvenance.Metadata["proof_provenance_map"] = changedProvenanceMap
	changedProvenanceHash, err := completionAuditSnapshotRequestHash(record, changedProvenance, options)
	if err != nil {
		t.Fatalf("hash changed provenance request failed: %v", err)
	}
	if changedProvenanceHash == base {
		t.Fatal("request hash should change when proof provenance changes")
	}

	changedReview := result
	changedReview.Metadata = map[string]any{}
	for key, value := range result.Metadata {
		changedReview.Metadata[key] = value
	}
	changedReview.Metadata["reviewed_by"] = "alternate-release-owner"
	changedReviewHash, err := completionAuditSnapshotRequestHash(record, changedReview, options)
	if err != nil {
		t.Fatalf("hash changed review metadata request failed: %v", err)
	}
	if changedReviewHash == base {
		t.Fatal("request hash should change when review metadata changes")
	}
}

func TestCompletionAuditHashIgnoresGeneratedAt(t *testing.T) {
	audit := CompletionAudit{
		Status:      "complete",
		Mode:        "read_only_completion_audit",
		Scope:       "v1.0",
		GeneratedAt: time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC),
		Items: []CompletionAuditItem{
			{
				Key:    "E9_areamatrix_protected_path_proof",
				Status: "complete",
				Metadata: map[string]any{
					"latest_proof_event_id": int64(90),
				},
			},
		},
	}
	first, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash first audit failed: %v", err)
	}
	audit.GeneratedAt = audit.GeneratedAt.Add(time.Hour)
	second, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash second audit failed: %v", err)
	}
	if first != second {
		t.Fatalf("hash changed after generated_at changed: %s != %s", first, second)
	}
	audit.Real100Guardrail = CompletionAuditReal100Guardrail()
	withGuardrail, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash audit with real 100 guardrail failed: %v", err)
	}
	if first != withGuardrail {
		t.Fatalf("hash changed after real 100 guardrail was populated: %s != %s", first, withGuardrail)
	}
	audit.Real100Guardrail.Real100Blockers = []string{"renamed_guardrail_blocker"}
	withChangedGuardrail, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash audit with changed real 100 guardrail failed: %v", err)
	}
	if first != withChangedGuardrail {
		t.Fatalf("hash changed after real 100 guardrail changed: %s != %s", first, withChangedGuardrail)
	}
	audit.Items = append(audit.Items, CompletionAuditItem{
		Key:    "E6_backup_restore_artifact_retention",
		Status: "complete",
		Metadata: map[string]any{
			"backup_restore_current_binding_bound": true,
			"current_backup_manifest_hash":         strings.Repeat("a", 64),
			"current_restore_plan_manifest_hash":   strings.Repeat("a", 64),
			"current_artifact_integrity_status":    "warn",
		},
	})
	withCurrentHash, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash audit with current backup binding failed: %v", err)
	}
	audit.Items[1].Metadata["current_backup_manifest_hash"] = strings.Repeat("b", 64)
	audit.Items[1].Metadata["current_restore_plan_manifest_hash"] = strings.Repeat("b", 64)
	withChangedCurrentHash, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash audit with changed current backup binding failed: %v", err)
	}
	if withCurrentHash != withChangedCurrentHash {
		t.Fatalf("hash changed after append-only current manifest hash changed: %s != %s", withCurrentHash, withChangedCurrentHash)
	}
	audit.Items = audit.Items[:1]
	audit.Items[0].Metadata["latest_proof_event_id"] = int64(91)
	changed, err := completionAuditHash(audit)
	if err != nil {
		t.Fatalf("hash changed audit failed: %v", err)
	}
	if changed == first {
		t.Fatal("hash should change when proof evidence changes")
	}
}

func TestBuildCompletionAuditDetectsForbiddenSecurityOpenings(t *testing.T) {
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{})
	readiness.SecretResolveOpen = true

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		SecurityBoundaryReadiness: &readiness,
	})

	item := findCompletionAuditItem(t, audit, "E8_security_permission_isolation")
	if item.Status != "blocked" {
		t.Fatalf("security item status = %q, want blocked: %+v", item.Status, item)
	}
	if !containsString(item.BlockedBy, "security_boundary_opened_forbidden_capability") {
		t.Fatalf("security item missing forbidden capability blocker: %+v", item)
	}
}

func TestBuildCompletionAuditUsesOperationsReadiness(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 30, 0, 0, time.UTC)
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "backup-hash"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := MigrationLedgerReadiness{
		Status:                       "needs_attention",
		Mode:                         "read_only_migration_ledger_readiness",
		AppliedCount:                 1,
		SchemaMigrationsTablePresent: true,
		FullLedgerTablePresent:       false,
		SafetyFacts:                  map[string]bool{"read_only": true},
		GeneratedAt:                  generated,
	}
	ops := BuildOperationsReadiness(LocalServiceStatus{Status: "ready", Mode: "local_service"}, support, ledger, OperationsReadinessOptions{GeneratedAt: generated})

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		OperationsReadiness: &ops,
	})

	item := findCompletionAuditItem(t, audit, "E7_operations_readiness")
	if item.Status != "incomplete" || item.NextCommand != "areaflow ops readiness --json" {
		t.Fatalf("operations item did not use operations readiness: %+v", item)
	}
	if item.Metadata["support_bundle_status"] != "ready" || item.Metadata["migration_ledger_status"] != "needs_attention" ||
		item.Metadata["telemetry_default"] != "local_only" ||
		item.Metadata["support_bundle_metadata_only"] != true ||
		item.Metadata["support_bundle_export_open"] != false ||
		item.Metadata["support_bundle_prompt_text_included"] != false ||
		item.Metadata["support_bundle_sensitive_exclusion_count"] != 9 {
		t.Fatalf("operations item missing readiness metadata: %+v", item.Metadata)
	}
	if !containsString(item.BlockedBy, "fresh_local_ops_smoke_missing") ||
		!containsString(item.BlockedBy, "full_migration_ledger_missing") {
		t.Fatalf("operations item missing blockers from readiness: %+v", item.BlockedBy)
	}
}

func TestBuildCompletionAuditBlocksUnsafeSupportBundleReadiness(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 35, 0, 0, time.UTC)
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "backup-hash"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	support.SafetyFacts["export_open"] = true
	support.SafetyFacts["user_file_contents_included"] = true
	support.ExcludedSensitiveContent = removeString(support.ExcludedSensitiveContent, "user_file_contents")
	ledger := MigrationLedgerReadiness{
		Status:                               "ready",
		Mode:                                 "read_only_migration_ledger_readiness",
		AppliedCount:                         1,
		SchemaMigrationsTablePresent:         true,
		FullLedgerTablePresent:               true,
		PreflightApplyVerifyRemediationReady: true,
		SafetyFacts:                          map[string]bool{"read_only": true},
		GeneratedAt:                          generated,
	}
	ops := BuildOperationsReadiness(LocalServiceStatus{Status: "ready", Mode: "local_service"}, support, ledger, OperationsReadinessOptions{
		GeneratedAt: generated,
		SmokeProof: OperationsSmokeProof{
			Project:                         Record{ID: 1, Key: "areamatrix"},
			ProofKey:                        "local_ops_smoke",
			Status:                          "recorded",
			EvidenceStatus:                  "pass",
			EventID:                         56,
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
		},
	})

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		OperationsReadiness: &ops,
	})

	item := findCompletionAuditItem(t, audit, "E7_operations_readiness")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "support_bundle_export_open") ||
		!containsString(item.BlockedBy, "support_bundle_user_file_contents_included") ||
		!containsString(item.BlockedBy, "support_bundle_exclusion_missing:user_file_contents") {
		t.Fatalf("unsafe support bundle should block completion audit E7: %+v", item)
	}
	if item.Metadata["support_bundle_export_open"] != true ||
		item.Metadata["support_bundle_user_file_contents_included"] != true {
		t.Fatalf("unsafe support metadata missing from completion audit: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditConsumesOperationsSmokeProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 45, 0, 0, time.UTC)
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "backup-hash"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := MigrationLedgerReadiness{
		Status:                       "needs_attention",
		Mode:                         "read_only_migration_ledger_readiness",
		AppliedCount:                 1,
		SchemaMigrationsTablePresent: true,
		FullLedgerTablePresent:       false,
		SafetyFacts:                  map[string]bool{"read_only": true},
		GeneratedAt:                  generated,
	}
	ops := BuildOperationsReadiness(LocalServiceStatus{Status: "ready", Mode: "local_service"}, support, ledger, OperationsReadinessOptions{
		GeneratedAt: generated,
		SmokeProof: OperationsSmokeProof{
			Project:                         Record{ID: 7, Key: "areamatrix-fixture"},
			ProofKey:                        "v1_stable_fixture_smoke",
			Status:                          "recorded",
			EvidenceStatus:                  "pass",
			EventID:                         55,
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
		},
	})

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		OperationsReadiness: &ops,
	})

	item := findCompletionAuditItem(t, audit, "E7_operations_readiness")
	if containsString(item.BlockedBy, "fresh_local_ops_smoke_missing") {
		t.Fatalf("operations proof should remove fresh smoke blocker: %+v", item.BlockedBy)
	}
	if !containsString(item.BlockedBy, "full_migration_ledger_missing") {
		t.Fatalf("full migration ledger blocker should remain: %+v", item.BlockedBy)
	}
	if item.Metadata["operations_status"] != "needs_attention" {
		t.Fatalf("operations status should still reflect full ledger gap: %+v", item.Metadata)
	}
	if item.Metadata["latest_operations_smoke_proof_key"] != "v1_stable_fixture_smoke" {
		t.Fatalf("operations item should expose latest proof key provenance: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsStaleOperationsSmokeProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 45, 0, 0, time.UTC)
	support := BuildSupportBundlePreview(BackupManifest{Status: "ready", ManifestHash: "backup-hash"}, AuditCoverage{Status: "warn"}, SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := MigrationLedgerReadiness{
		Status:                               "ready",
		Mode:                                 "read_only_migration_ledger_readiness",
		AppliedCount:                         1,
		SchemaMigrationsTablePresent:         true,
		FullLedgerTablePresent:               true,
		PreflightApplyVerifyRemediationReady: true,
		SafetyFacts:                          map[string]bool{"read_only": true},
		GeneratedAt:                          generated,
	}
	ops := BuildOperationsReadiness(LocalServiceStatus{Status: "ready", Mode: "local_service"}, support, ledger, OperationsReadinessOptions{
		GeneratedAt: generated,
		SmokeProof: OperationsSmokeProof{
			Project:                         Record{ID: 1, Key: "areamatrix"},
			ProofKey:                        "local_ops_smoke",
			Status:                          "recorded",
			EvidenceStatus:                  "pass",
			EventID:                         57,
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
			Metadata:                        map[string]any{"summary": "stale ops smoke reviewed", "evidence_uri": "docs/development/operations-readiness-evidence.md"},
		},
	})

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		OperationsReadiness: &ops,
	})

	item := findCompletionAuditItem(t, audit, "E7_operations_readiness")
	if item.Status != "incomplete" ||
		!containsString(item.BlockedBy, "operations_smoke_proof_stale") ||
		item.Metadata["latest_operations_smoke_proof_fresh"] != false ||
		item.Metadata["latest_operations_smoke_proof_freshness_status"] != "stale" {
		t.Fatalf("stale operations proof should not complete E7: %+v", item)
	}
}

func TestBuildCompletionAuditConsumesProtectedPathProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 10, 0, 0, time.UTC)
	proof := buildProtectedPathProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus: "clean",
		Summary:     "protected path review was clean",
		EvidenceURI: "local:protected-path-git-status",
	}))
	proof.EventID = 70
	proof.AuditEventID = 71

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ProtectedPathProof: &proof,
	})

	if audit.ProtectedPathProofStatus != "complete" {
		t.Fatalf("protected path aggregate status = %q, want complete", audit.ProtectedPathProofStatus)
	}
	item := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("protected path proof should complete E9: %+v", item)
	}
	if item.Metadata["latest_proof_event_id"] != int64(70) ||
		item.Metadata["latest_proof_evidence_uri"] != "local:protected-path-git-status" ||
		item.Metadata["protected_path_proof_binding_status"] != "pass" ||
		item.Metadata["git_status_output_empty"] != true ||
		item.Metadata["git_status_output_lines"] != 0 ||
		item.Metadata["protected_path_set_hash"] != protectedPathProofSetHash() ||
		item.Metadata["protected_path_set_count"] != int64(len(protectedPathProofSet())) {
		t.Fatalf("protected path proof metadata missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsLooseCleanProtectedPathProof(t *testing.T) {
	proof := ProtectedPathProof{
		Project:                         Record{ID: 1, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "clean",
		Decision:                        "allowed",
		EventID:                         72,
		AuditEventID:                    73,
		AreaMatrixProtectedPathsTouched: false,
		GitStatusRunByCommand:           false,
		Metadata:                        map[string]any{"summary": "protected path review was clean", "evidence_uri": "local:protected-path-git-status"},
	}

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ProtectedPathProof: &proof})
	if audit.ProtectedPathProofStatus != "blocked" {
		t.Fatalf("loose clean protected path proof should keep aggregate blocked: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "protected_path_proof_binding_incomplete") ||
		item.Metadata["protected_path_proof_binding_status"] != "" {
		t.Fatalf("loose clean protected path proof should be blocked by binding: %+v", item)
	}
	blockers, ok := item.Metadata["protected_path_proof_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "protected_path_proof_binding_status_not_pass") ||
		!containsString(blockers, "protected_path_set_hash_missing_or_mismatch") {
		t.Fatalf("loose proof binding blockers missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsMismatchedProofProject(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 20, 0, 0, time.UTC)
	otherProject := Record{ID: 99, Key: "areamatrix-fixture"}
	sourceBinding := sourceAlignmentProofTestCurrentBinding(t)
	sourceProof := buildSourceAlignmentProof(otherProject, normalizeRecordSourceAlignmentProofOptions(RecordSourceAlignmentProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSourceAlignmentProofFacts,
		Summary:                "wrong project source proof",
		EvidenceURI:            "local:wrong-project-source-proof",
		SourceAlignmentBinding: sourceBinding,
	}))
	archiveProof := buildArchiveProof(otherProject, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "wrong project archive proof",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(otherProject, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "wrong project shim proof",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(otherProject, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "wrong project execution cutover proof",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))
	protectedPathProof := ProtectedPathProof{
		Project:                         otherProject,
		Status:                          "recorded",
		ProofStatus:                     "clean",
		Decision:                        "allowed",
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        map[string]any{"summary": "wrong project protected path proof", "evidence_uri": "local:wrong-project-protected-path-proof"},
	}

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:                 realAreaMatrixRecordPtr(),
		SourceAlignmentProof:          &sourceProof,
		SourceAlignmentCurrentBinding: sourceBinding,
		ArchiveProof:                  &archiveProof,
		ShimRetirementProof:           &shimProof,
		ExecutionCutoverProof:         &executionProof,
		ProtectedPathProof:            &protectedPathProof,
	})

	sourceItem := findCompletionAuditItem(t, audit, "E1_design_source_alignment")
	if sourceItem.Status == "complete" || !containsString(sourceItem.BlockedBy, "source_alignment_proof_project_mismatch") {
		t.Fatalf("source proof from another project must not complete E1: %+v", sourceItem)
	}
	if sourceItem.Metadata["latest_source_alignment_proof_project_key"] != "areamatrix-fixture" ||
		sourceItem.Metadata["expected_project_key"] != completionAuditTargetProjectKey {
		t.Fatalf("source proof mismatch metadata missing: %+v", sourceItem.Metadata)
	}

	dogfoodItem := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if dogfoodItem.Status == "complete" ||
		!containsString(dogfoodItem.BlockedBy, "archive_proof_project_mismatch") ||
		!containsString(dogfoodItem.BlockedBy, "shim_retirement_proof_project_mismatch") ||
		!containsString(dogfoodItem.BlockedBy, "execution_cutover_proof_project_mismatch") {
		t.Fatalf("dogfood proofs from another project must not complete E4: %+v", dogfoodItem)
	}
	if dogfoodItem.Metadata["archive_gate_passed"] == true ||
		dogfoodItem.Metadata["shim_retirement_gate_passed"] == true ||
		dogfoodItem.Metadata["execution_cutover_gate_passed"] == true {
		t.Fatalf("mismatched dogfood proofs must not set gate_passed metadata: %+v", dogfoodItem.Metadata)
	}

	protectedItem := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if protectedItem.Status == "complete" || !containsString(protectedItem.BlockedBy, "protected_path_proof_project_mismatch") {
		t.Fatalf("protected path proof from another project must not complete E9: %+v", protectedItem)
	}
	if audit.Status != "blocked" {
		t.Fatalf("mismatched proof project should keep full audit blocked: %+v", audit)
	}
}

func TestBuildCompletionAuditConsumesArchiveProofWithoutCompletingDogfood(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC)
	proof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	proof.EventID = 90

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject: realAreaMatrixRecordPtr(),
		ArchiveProof:  &proof,
	})

	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status != "incomplete" {
		t.Fatalf("archive proof alone must not complete dogfood: %+v", item)
	}
	if containsString(item.BlockedBy, "real_areamatrix_archive_not_proven") {
		t.Fatalf("archive proof should remove archive blocker: %+v", item.BlockedBy)
	}
	if !containsString(item.BlockedBy, "execution_cutover_not_complete") ||
		!containsString(item.BlockedBy, "real_areamatrix_shim_retirement_not_proven") {
		t.Fatalf("execution and shim blockers should remain: %+v", item.BlockedBy)
	}
	if item.Metadata["archive_gate_passed"] != true ||
		item.Metadata["latest_archive_proof_event_id"] != int64(90) ||
		item.Metadata["latest_archive_proof_evidence_uri"] != e4ReleaseCandidateEvidenceURI("e4-archive") ||
		item.Metadata["archive_scope_binding_status"] != "pass" ||
		item.Metadata["archive_binding_contract"] != archiveProofBindingContract ||
		item.Metadata["archive_scope_binding_hash"] != item.Metadata["current_archive_scope_binding_hash"] ||
		item.Metadata["archive_scope_current_binding_bound"] != true ||
		item.Metadata["archive_scope"] != archiveProofScope ||
		item.Metadata["archive_reference_mode"] != archiveProofReferenceMode ||
		item.Metadata["archive_rollback_target"] != archiveProofRollbackTarget ||
		item.Metadata["archive_fail_closed"] != true {
		t.Fatalf("archive proof metadata missing: %+v", item.Metadata)
	}

	executionReadiness := AreaMatrixExecutionCutoverReadiness{
		Status: "pass",
		SafetyFacts: map[string]bool{
			"execution_cutover_apply_open": true,
			"task_loop_run_forwarded":      true,
		},
	}
	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:     realAreaMatrixRecordPtr(),
		AreaMatrixDogfood: &executionReadiness,
		ArchiveProof:      &proof,
	})
	if audit.AreaMatrixDogfoodStatus != "incomplete" {
		t.Fatalf("dogfood aggregate must stay incomplete until shim retirement proof exists: %+v", audit)
	}
}

func TestBuildCompletionAuditRejectsLooseArchiveProof(t *testing.T) {
	proof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "loose archive gate reviewed",
		EvidenceURI: "local:loose-archive-gate-review",
	}))

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{TargetProject: realAreaMatrixRecordPtr(), ArchiveProof: &proof})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "archive_scope_binding_incomplete") ||
		item.Metadata["archive_gate_passed"] == true {
		t.Fatalf("loose archive proof should be blocked by missing binding: %+v", item)
	}
	blockers, ok := item.Metadata["archive_scope_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "archive_scope_binding_status_not_pass") ||
		!containsString(blockers, "archive_scope_missing_or_mismatch") ||
		!containsString(blockers, "archive_fail_closed_missing") {
		t.Fatalf("loose archive binding blockers missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsTamperedArchiveBindingHash(t *testing.T) {
	proof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	proof.EventID = 90
	proof.Metadata["archive_scope_binding_hash"] = strings.Repeat("0", 64)

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{TargetProject: realAreaMatrixRecordPtr(), ArchiveProof: &proof})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Metadata["archive_gate_passed"] == true ||
		!containsString(item.BlockedBy, "archive_scope_binding_incomplete") ||
		item.Metadata["archive_scope_current_binding_bound"] != false {
		t.Fatalf("tampered archive binding hash should block archive gate: %+v", item)
	}
	blockers, ok := item.Metadata["archive_scope_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "archive_scope_binding_hash_missing_or_mismatch") {
		t.Fatalf("tampered archive binding hash blocker missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsArchiveProofMissingEventID(t *testing.T) {
	proof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ArchiveProof: &proof})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Metadata["archive_gate_passed"] == true ||
		!containsString(item.BlockedBy, "archive_proof_event_id_missing") ||
		!containsString(item.BlockedBy, "real_areamatrix_archive_not_proven") {
		t.Fatalf("archive proof without event id should not pass gate: %+v", item)
	}
	if item.Metadata["archive_scope_current_binding_bound"] != true {
		t.Fatalf("missing event id should not be reported as current binding drift: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditConsumesShimRetirementProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 45, 0, 0, time.UTC)
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:       realAreaMatrixRecordPtr(),
		ArchiveProof:        &archiveProof,
		ShimRetirementProof: &shimProof,
	})

	if audit.AreaMatrixDogfoodStatus != "incomplete" {
		t.Fatalf("dogfood aggregate should stay incomplete until execution cutover proof exists: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status != "incomplete" || !containsString(item.BlockedBy, "execution_cutover_not_complete") {
		t.Fatalf("dogfood item should still require execution cutover proof: %+v", item)
	}
	if item.Metadata["shim_retirement_gate_passed"] != true ||
		item.Metadata["latest_shim_retirement_proof_event_id"] != int64(91) ||
		item.Metadata["latest_shim_retirement_proof_evidence_uri"] != e4ReleaseCandidateEvidenceURI("e4-shim-retirement") ||
		item.Metadata["shim_retirement_scope_binding_status"] != "pass" ||
		item.Metadata["shim_retirement_binding_contract"] != shimRetirementProofBindingContract ||
		item.Metadata["shim_retirement_scope_binding_hash"] != item.Metadata["current_shim_retirement_scope_binding_hash"] ||
		item.Metadata["shim_retirement_scope_current_binding_bound"] != true ||
		item.Metadata["shim_retirement_scope"] != shimRetirementProofScope ||
		item.Metadata["shim_rollback_target"] != shimRetirementProofRollbackTarget ||
		item.Metadata["shim_fail_closed"] != true ||
		item.Metadata["shim_reopen_requires_approval"] != true {
		t.Fatalf("shim proof metadata missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsLooseShimRetirementProof(t *testing.T) {
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	proof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "loose shim retirement reviewed",
		EvidenceURI: "local:loose-shim-retirement-review",
	}))

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		TargetProject:       realAreaMatrixRecordPtr(),
		ArchiveProof:        &archiveProof,
		ShimRetirementProof: &proof,
	})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "shim_retirement_scope_binding_incomplete") ||
		item.Metadata["shim_retirement_gate_passed"] == true {
		t.Fatalf("loose shim retirement proof should be blocked by missing binding: %+v", item)
	}
	blockers, ok := item.Metadata["shim_retirement_scope_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "shim_retirement_scope_binding_status_not_pass") ||
		!containsString(blockers, "shim_retirement_scope_missing_or_mismatch") ||
		!containsString(blockers, "shim_fail_closed_missing") {
		t.Fatalf("loose shim binding blockers missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsShimRetirementProofMissingEventID(t *testing.T) {
	proof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{TargetProject: realAreaMatrixRecordPtr(), ShimRetirementProof: &proof})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Metadata["shim_retirement_gate_passed"] == true ||
		!containsString(item.BlockedBy, "shim_retirement_proof_event_id_missing") ||
		!containsString(item.BlockedBy, "real_areamatrix_shim_retirement_not_proven") {
		t.Fatalf("shim retirement proof without event id should not pass gate: %+v", item)
	}
	if item.Metadata["shim_retirement_scope_current_binding_bound"] != true {
		t.Fatalf("missing event id should not be reported as current binding drift: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsTamperedShimRetirementBindingHash(t *testing.T) {
	proof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	proof.EventID = 91
	proof.Metadata["shim_retirement_scope_binding_hash"] = strings.Repeat("0", 64)

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{TargetProject: realAreaMatrixRecordPtr(), ShimRetirementProof: &proof})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Metadata["shim_retirement_gate_passed"] == true ||
		!containsString(item.BlockedBy, "shim_retirement_scope_binding_incomplete") ||
		item.Metadata["shim_retirement_scope_current_binding_bound"] != false {
		t.Fatalf("tampered shim binding hash should block shim gate: %+v", item)
	}
	blockers, ok := item.Metadata["shim_retirement_scope_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "shim_retirement_scope_binding_hash_missing_or_mismatch") {
		t.Fatalf("tampered shim binding hash blocker missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditConsumesExecutionCutoverProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 55, 0, 0, time.UTC)
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))
	executionProof.EventID = 92

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:            realAreaMatrixRecordPtr(),
		ArchiveProof:             &archiveProof,
		ShimRetirementProof:      &shimProof,
		ExecutionCutoverProof:    &executionProof,
		PackageAStatusProjection: readyPackageAStatusProjectionBinding("source-hash-rc"),
	})

	if audit.AreaMatrixDogfoodStatus != "complete" {
		t.Fatalf("dogfood aggregate should be complete when Package A and archive/shim/execution proofs exist: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("dogfood item should be complete: %+v", item)
	}
	if item.Metadata["execution_cutover_gate_passed"] != true ||
		item.Metadata["execution_cutover_proof_status"] != "complete" ||
		item.Metadata["latest_execution_cutover_proof_event_id"] != int64(92) ||
		item.Metadata["latest_execution_cutover_proof_evidence_uri"] != e4ReleaseCandidateEvidenceURI("e4-execution-cutover") ||
		item.Metadata["archive_scope_binding_status"] != "pass" ||
		item.Metadata["archive_scope_current_binding_bound"] != true ||
		item.Metadata["shim_retirement_scope_binding_status"] != "pass" ||
		item.Metadata["shim_retirement_scope_current_binding_bound"] != true ||
		item.Metadata["execution_cutover_scope_binding_status"] != "pass" ||
		item.Metadata["execution_cutover_scope_current_binding_bound"] != true ||
		item.Metadata["execution_cutover_scope"] != executionCutoverProofScope ||
		item.Metadata["execution_cutover_binding_contract"] != executionCutoverProofBindingContract ||
		item.Metadata["execution_cutover_scope_binding_hash"] != metadataString(executionProof.Metadata, "execution_cutover_scope_binding_hash") ||
		item.Metadata["current_execution_cutover_scope_binding_hash"] != metadataString(executionProof.Metadata, "execution_cutover_scope_binding_hash") ||
		item.Metadata["execution_cutover_rollback_target"] != executionCutoverProofRollbackTarget ||
		item.Metadata["execution_cutover_rollback_mode"] != executionCutoverProofRollbackMode ||
		item.Metadata["execution_cutover_fail_closed"] != true ||
		item.Metadata["execution_cutover_reopen_requires_approval"] != true ||
		item.Metadata["package_a_status_projection_ready"] != true ||
		item.Metadata["package_a_has_written_status_projection"] != true {
		t.Fatalf("execution cutover proof metadata missing: %+v", item.Metadata)
	}
	if item.Metadata["project_write_attempted"] != false ||
		item.Metadata["execution_write_attempted"] != false ||
		item.Metadata["task_loop_run_forwarded_by_command"] != false ||
		item.Metadata["engine_call_attempted"] != false ||
		item.Metadata["commands_run"] != false ||
		item.Metadata["legacy_progress_written"] != false ||
		item.Metadata["legacy_logs_written"] != false ||
		item.Metadata["legacy_checkpoint_written"] != false ||
		item.Metadata["area_matrix_protected_paths_touched"] != false {
		t.Fatalf("execution cutover proof safety metadata missing: %+v", item.Metadata)
	}
}

func TestCompletionProofEventMetadataPreservesReviewFields(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	proofs := map[string]struct {
		response map[string]any
		restore  func(EventRecord) map[string]any
	}{
		"archive": {
			response: archiveProofCommandResponse(buildArchiveProof(record, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
				ProofStatus: "complete",
				Facts:       requiredArchiveProofFacts,
				Summary:     "archive gate reviewed",
				EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
			})))),
			restore: func(event EventRecord) map[string]any {
				return archiveProofFromEvent(event).Metadata
			},
		},
		"shim_retirement": {
			response: shimRetirementProofCommandResponse(buildShimRetirementProof(record, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
				ProofStatus: "complete",
				Facts:       requiredShimRetirementProofFacts,
				Summary:     "shim retirement reviewed",
				EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
			})))),
			restore: func(event EventRecord) map[string]any {
				return shimRetirementProofFromEvent(event).Metadata
			},
		},
		"execution_cutover": {
			response: executionCutoverProofCommandResponse(buildExecutionCutoverProof(record, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
				ProofStatus: "complete",
				Facts:       requiredExecutionCutoverProofFacts,
				Summary:     "execution cutover reviewed",
				EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
			})))),
			restore: func(event EventRecord) map[string]any {
				return executionCutoverProofFromEvent(event).Metadata
			},
		},
	}

	for name, proof := range proofs {
		t.Run(name, func(t *testing.T) {
			if metadataString(proof.response, "review_decision") != "approved" ||
				metadataString(proof.response, "reviewed_by") != "release-owner" ||
				metadataString(proof.response, "reviewed_at") != "2026-07-04T12:00:00Z" ||
				metadataString(proof.response, "review_metadata_status") != "approved" {
				t.Fatalf("response missing top-level review metadata: %+v", proof.response)
			}

			legacyEventMetadata := map[string]any{}
			for key, value := range proof.response {
				legacyEventMetadata[key] = value
			}
			for _, key := range []string{"review_decision", "reviewed_by", "reviewed_at", "review_metadata_status", "review_metadata_blockers"} {
				delete(legacyEventMetadata, key)
			}
			restored := proof.restore(EventRecord{ID: 77, ProjectID: record.ID, Metadata: legacyEventMetadata})
			if metadataString(restored, "review_decision") != "approved" ||
				metadataString(restored, "reviewed_by") != "release-owner" ||
				metadataString(restored, "reviewed_at") != "2026-07-04T12:00:00Z" ||
				metadataString(restored, "review_metadata_status") != "approved" ||
				len(metadataStringSlice(restored, "review_metadata_blockers")) != 0 {
				t.Fatalf("event restore lost nested review metadata: %+v", restored)
			}
		})
	}
}

func TestBuildCompletionAuditBlocksFullProofFixtureIdentity(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 55, 0, 0, time.UTC)
	fixtureProject := realAreaMatrixRecord()
	fixtureProject.Kind = "fixture"
	fixtureProject.RootPath = filepath.Join(os.TempDir(), "areaflow-fixture", "areamatrix-root")
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))
	executionProof.EventID = 92

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:            &fixtureProject,
		ArchiveProof:             &archiveProof,
		ShimRetirementProof:      &shimProof,
		ExecutionCutoverProof:    &executionProof,
		PackageAStatusProjection: readyPackageAStatusProjectionBinding("source-hash-rc"),
	})

	if audit.Status != "blocked" || audit.AreaMatrixDogfoodStatus != "blocked" {
		t.Fatalf("fixture identity full proof should block aggregate completion: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "project_root_not_real_areamatrix") ||
		!containsString(item.BlockedBy, "project_kind_not_product_repo") ||
		!containsString(item.BlockedBy, "execution_cutover_not_complete") ||
		!containsString(item.BlockedBy, "real_areamatrix_archive_not_proven") ||
		!containsString(item.BlockedBy, "real_areamatrix_shim_retirement_not_proven") ||
		containsString(item.BlockedBy, "completion_audit_snapshot_package_a_not_applied") ||
		item.Metadata["package_a_status_projection_ready"] != true ||
		item.Metadata["real_project_identity_status"] != "blocked" {
		t.Fatalf("fixture identity should be the remaining E4 blocker with Package A ready: %+v", item)
	}
	if containsString(audit.Real100Guardrail.Real100Blockers, "package_a_status_projection_not_applied") ||
		containsString(audit.Real100Guardrail.Real100Blockers, "real_areamatrix_read_only_shim_not_landed") ||
		!containsString(audit.Real100Guardrail.Real100Blockers, "real_areamatrix_execution_cutover_not_proven") ||
		!containsString(audit.Real100Guardrail.Real100Blockers, "release_candidate_snapshot_not_ready") {
		t.Fatalf("fixture identity should not reintroduce Package A/read-only shim blockers: %+v", audit.Real100Guardrail)
	}
}

func TestBuildCompletionAuditRequiresPackageAForDogfoodCompletion(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 56, 0, 0, time.UTC)
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))
	executionProof.EventID = 92

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:         realAreaMatrixRecordPtr(),
		ArchiveProof:          &archiveProof,
		ShimRetirementProof:   &shimProof,
		ExecutionCutoverProof: &executionProof,
		PackageAStatusProjection: completionAuditSnapshotPackageAStatusProjectionBinding{
			LatestImportSourceHash:  "source-hash-rc",
			CurrentPreimageCaptured: true,
			CurrentPreimage: StatusProjectionPreimage{
				SchemaStatus: "legacy",
				Exists:       true,
				Readable:     true,
				SHA256:       "legacy-status-hash",
				Message:      "target uses legacy status projection shape",
			},
		},
	})

	if audit.AreaMatrixDogfoodStatus == "complete" {
		t.Fatalf("dogfood aggregate must not complete before Package A apply: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "completion_audit_snapshot_package_a_not_applied") ||
		!containsString(item.BlockedBy, "package_a_status_projection_not_written") ||
		item.Metadata["package_a_status_projection_ready"] != false ||
		item.Metadata["package_a_has_written_status_projection"] != false ||
		item.Metadata["archive_gate_passed"] != true ||
		item.Metadata["shim_retirement_gate_passed"] != true ||
		item.Metadata["execution_cutover_gate_passed"] != true {
		t.Fatalf("Package A should block E4 while preserving other proof pass metadata: %+v", item)
	}
}

func TestBuildCompletionAuditRejectsLooseExecutionCutoverProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 15, 57, 0, 0, time.UTC)
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "loose execution cutover reviewed",
		EvidenceURI: "local:loose-execution-cutover-review",
	}))
	executionProof.EventID = 93

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TargetProject:         realAreaMatrixRecordPtr(),
		ArchiveProof:          &archiveProof,
		ShimRetirementProof:   &shimProof,
		ExecutionCutoverProof: &executionProof,
	})

	if audit.AreaMatrixDogfoodStatus != "incomplete" {
		t.Fatalf("loose execution cutover proof must not complete dogfood aggregate: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status != "incomplete" ||
		!containsString(item.BlockedBy, "execution_cutover_scope_binding_incomplete") ||
		item.Metadata["execution_cutover_gate_passed"] == true {
		t.Fatalf("loose execution proof should be blocked by missing scope binding: %+v", item)
	}
	blockers, ok := item.Metadata["execution_cutover_scope_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "execution_cutover_scope_binding_status_not_pass") ||
		!containsString(blockers, "allowed_task_types_missing_or_mismatch") ||
		!containsString(blockers, "rollback_target_missing_or_mismatch") {
		t.Fatalf("missing scope binding blockers not exposed: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsExecutionCutoverProofMissingEventID(t *testing.T) {
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		TargetProject:            realAreaMatrixRecordPtr(),
		ArchiveProof:             &archiveProof,
		ShimRetirementProof:      &shimProof,
		ExecutionCutoverProof:    &executionProof,
		PackageAStatusProjection: readyPackageAStatusProjectionBinding("source-hash-rc"),
	})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Status == "complete" ||
		item.Metadata["execution_cutover_gate_passed"] == true ||
		!containsString(item.BlockedBy, "execution_cutover_proof_event_id_missing") ||
		!containsString(item.BlockedBy, "execution_cutover_not_complete") {
		t.Fatalf("execution cutover proof without event id should not pass gate: %+v", item)
	}
	if real100BreakdownHasKey(audit.Real100Guardrail.Real100Breakdown.CompletedEvidence, "real_areamatrix_execution_cutover_proof") {
		t.Fatalf("execution cutover proof without event id must not be completed evidence: %+v", audit.Real100Guardrail.Real100Breakdown)
	}
}

func TestBuildCompletionAuditRejectsTamperedExecutionCutoverBindingHash(t *testing.T) {
	archiveProof := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	archiveProof.EventID = 90
	shimProof := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	shimProof.EventID = 91
	executionProof := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))
	executionProof.EventID = 92
	executionProof.Metadata["execution_cutover_scope_binding_hash"] = strings.Repeat("0", 64)

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		TargetProject:            realAreaMatrixRecordPtr(),
		ArchiveProof:             &archiveProof,
		ShimRetirementProof:      &shimProof,
		ExecutionCutoverProof:    &executionProof,
		PackageAStatusProjection: readyPackageAStatusProjectionBinding("source-hash-rc"),
	})
	item := findCompletionAuditItem(t, audit, "E4_areamatrix_dogfood_completion")
	if item.Metadata["execution_cutover_gate_passed"] == true ||
		!containsString(item.BlockedBy, "execution_cutover_scope_binding_incomplete") ||
		item.Metadata["execution_cutover_scope_current_binding_bound"] != false {
		t.Fatalf("tampered execution cutover binding hash should block execution gate: %+v", item)
	}
	blockers, ok := item.Metadata["execution_cutover_scope_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "execution_cutover_scope_binding_hash_missing_or_mismatch") {
		t.Fatalf("tampered execution cutover binding hash blocker missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditConsumesValidationProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 16, 0, 0, 0, time.UTC)
	proof := buildValidationProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordValidationProofOptions(withValidationEvidenceBinding(RecordValidationProofOptions{
		ProofStatus: "complete",
		Facts:       requiredValidationProofFacts,
		Summary:     "fresh validation proof reviewed",
		EvidenceURI: "local:fresh-validation-proof",
	})))
	proof.EventID = 92

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ValidationProof: &proof,
	})
	item := findCompletionAuditItem(t, audit, "E3_command_api_smoke_evidence")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("validation item should be complete: %+v", item)
	}
	if item.Metadata["validation_gate_passed"] != true ||
		item.Metadata["latest_validation_proof_event_id"] != int64(92) ||
		item.Metadata["latest_validation_proof_evidence_uri"] != "local:fresh-validation-proof" ||
		item.Metadata["validation_evidence_binding_status"] != "pass" ||
		item.Metadata["validation_result_hash"] != strings.Repeat("a", 64) ||
		item.Metadata["validation_command_count"] != int64(2) {
		t.Fatalf("validation proof metadata missing: %+v", item.Metadata)
	}
	if audit.Status != "blocked" {
		t.Fatalf("validation proof alone must not complete full audit: %+v", audit)
	}
}

func withValidationEvidenceBinding(options RecordValidationProofOptions) RecordValidationProofOptions {
	options.ValidationCommands = []string{
		"go test ./...",
		"make smoke-docker-validation-proof",
	}
	options.ValidationResultHash = strings.Repeat("a", 64)
	options.ValidationStartedAt = "2026-07-06T10:00:00Z"
	options.ValidationFinishedAt = "2026-07-06T10:10:00Z"
	options.ValidationScope = "fixture_validation_review"
	return options
}

func withExecutionCutoverEvidenceBinding(options RecordExecutionCutoverProofOptions) RecordExecutionCutoverProofOptions {
	options.ExecutionCutoverScope = executionCutoverProofScope
	options.AllowedTaskTypes = append([]string{}, executionForwardingV1AllowedTaskTypes...)
	options.ForbiddenActions = append([]string{}, requiredExecutionCutoverProofForbiddenActions...)
	options.RollbackTarget = executionCutoverProofRollbackTarget
	options.RollbackMode = executionCutoverProofRollbackMode
	options.FailClosed = true
	options.ReopenRequiresApproval = true
	options.ReviewDecision = "approved"
	options.ReviewedBy = "release-owner"
	options.ReviewedAt = time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	return options
}

func withArchiveProofTestBinding(options RecordArchiveProofOptions) RecordArchiveProofOptions {
	options.ArchiveScope = archiveProofScope
	options.ArchiveReferenceMode = archiveProofReferenceMode
	options.ArchiveSourcePaths = append([]string{}, requiredArchiveProofSourcePaths...)
	options.ArchiveForbiddenActions = append([]string{}, requiredArchiveProofForbiddenActions...)
	options.ArchiveRollbackTarget = archiveProofRollbackTarget
	options.ArchiveFailClosed = true
	options.ReviewDecision = "approved"
	options.ReviewedBy = "release-owner"
	options.ReviewedAt = time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	return options
}

func withShimRetirementProofTestBinding(options RecordShimRetirementProofOptions) RecordShimRetirementProofOptions {
	options.ShimRetirementScope = shimRetirementProofScope
	options.ShimRetirementPrerequisites = append([]string{}, requiredShimRetirementProofPrerequisites...)
	options.ShimRetiredSurfaces = append([]string{}, requiredShimRetiredSurfaces...)
	options.ShimRollbackTarget = shimRetirementProofRollbackTarget
	options.ShimFailClosed = true
	options.ShimReopenRequiresApproval = true
	options.ReviewDecision = "approved"
	options.ReviewedBy = "release-owner"
	options.ReviewedAt = time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	return options
}

func withBackupRestoreEvidenceBinding(options RecordBackupRestoreProofOptions) RecordBackupRestoreProofOptions {
	projectCount := int64(1)
	tableCount := int64(35)
	restoreItemCount := int64(5)
	checkedArtifacts := int64(2)
	failedArtifacts := int64(0)
	totalArtifacts := int64(2)
	externalRefs := int64(1)
	needsPolicy := int64(0)
	writeAttempted := false
	hash := strings.Repeat("b", 64)
	options.BackupManifestHash = hash
	options.BackupManifestStatus = "ready"
	options.BackupManifestProjectCount = &projectCount
	options.BackupManifestTableCount = &tableCount
	options.RestorePlanStatus = "needs_attention"
	options.RestorePlanScope = "project"
	options.RestorePlanProjectKey = "areamatrix"
	options.RestorePlanManifestHash = hash
	options.RestorePlanItemCount = &restoreItemCount
	options.ArtifactIntegrityStatus = "warn"
	options.ArtifactIntegrityCheckedCount = &checkedArtifacts
	options.ArtifactIntegrityFailedCount = &failedArtifacts
	options.ArtifactArchivePreviewStatus = "needs_attention"
	options.ArtifactArchivePreviewTotalArtifacts = &totalArtifacts
	options.ArtifactArchivePreviewExternalRefs = &externalRefs
	options.ArtifactArchivePreviewNeedsPolicy = &needsPolicy
	options.ArtifactArchivePreviewProjectWriteAttempted = &writeAttempted
	options.ArtifactArchivePreviewStorageWriteAttempted = &writeAttempted
	options.ArtifactArchivePreviewDeleteAttempted = &writeAttempted
	return options
}

func matchingBackupRestoreCurrentBinding() map[string]any {
	options := withBackupRestoreEvidenceBinding(RecordBackupRestoreProofOptions{})
	metadata := map[string]any{}
	addBackupRestoreProofBindingMetadata(metadata, options)
	currentHash := strings.Repeat("c", 64)
	metadata["backup_manifest_hash"] = currentHash
	metadata["restore_plan_manifest_hash"] = currentHash
	return metadata
}

func driftedBackupRestoreCurrentBinding() map[string]any {
	metadata := copyMap(matchingBackupRestoreCurrentBinding())
	metadata["restore_plan_status"] = "blocked"
	metadata["artifact_integrity_status"] = "fail"
	metadata["artifact_integrity_failed_count"] = int64(1)
	return metadata
}

func sourceAlignmentProofTestCurrentBinding(t *testing.T) map[string]any {
	t.Helper()
	binding, err := SourceAlignmentCurrentBinding()
	if err != nil {
		t.Fatalf("source alignment current binding: %v", err)
	}
	return binding
}

func withSourceAlignmentProofTestBinding(t *testing.T, options RecordSourceAlignmentProofOptions) RecordSourceAlignmentProofOptions {
	t.Helper()
	options.SourceAlignmentBinding = sourceAlignmentProofTestCurrentBinding(t)
	return options
}

func TestBuildCompletionAuditConsumesSourceAlignmentProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 16, 20, 0, 0, time.UTC)
	binding := sourceAlignmentProofTestCurrentBinding(t)
	proof := buildSourceAlignmentProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordSourceAlignmentProofOptions(RecordSourceAlignmentProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSourceAlignmentProofFacts,
		Summary:                "source alignment reviewed",
		EvidenceURI:            "local:source-alignment-review",
		SourceAlignmentBinding: binding,
	}))
	proof.EventID = 93

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		SourceAlignmentProof:          &proof,
		SourceAlignmentCurrentBinding: binding,
	})
	item := findCompletionAuditItem(t, audit, "E1_design_source_alignment")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("source alignment item should be complete: %+v", item)
	}
	if item.Metadata["source_alignment_gate_passed"] != true ||
		item.Metadata["latest_source_alignment_proof_event_id"] != int64(93) ||
		item.Metadata["latest_source_alignment_proof_evidence_uri"] != "local:source-alignment-review" ||
		item.Metadata["source_alignment_binding_status"] != "pass" ||
		item.Metadata["source_alignment_current_binding_bound"] != true ||
		item.Metadata["source_alignment_source_set_hash"] != metadataString(binding, "source_alignment_source_set_hash") ||
		item.Metadata["source_alignment_source_file_count"] != metadataInt64(binding, "source_alignment_source_file_count") {
		t.Fatalf("source alignment proof metadata missing: %+v", item.Metadata)
	}
	if audit.Status != "blocked" {
		t.Fatalf("source alignment proof alone must not complete full audit: %+v", audit)
	}
}

func TestBuildCompletionAuditRejectsLooseSourceAlignmentProof(t *testing.T) {
	proof := SourceAlignmentProof{
		Project:      Record{ID: 1, Key: "areamatrix"},
		Status:       "recorded",
		ProofStatus:  "complete",
		Decision:     "allowed",
		Facts:        requiredSourceAlignmentProofFacts,
		MissingFacts: []string{},
		Metadata:     map[string]any{"summary": "source alignment reviewed", "evidence_uri": "local:source-alignment-review"},
	}

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		SourceAlignmentProof:          &proof,
		SourceAlignmentCurrentBinding: sourceAlignmentProofTestCurrentBinding(t),
	})
	item := findCompletionAuditItem(t, audit, "E1_design_source_alignment")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "source_alignment_proof_incomplete") ||
		item.Metadata["source_alignment_binding_status"] != "" {
		t.Fatalf("loose source alignment proof should be blocked by binding: %+v", item)
	}
	blockers, ok := item.Metadata["source_alignment_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "source_alignment_binding_status_not_pass") ||
		!containsString(blockers, "source_alignment_source_set_hash_missing_or_mismatch") {
		t.Fatalf("loose source alignment binding blockers missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditRejectsDriftedSourceAlignmentBinding(t *testing.T) {
	binding := sourceAlignmentProofTestCurrentBinding(t)
	proof := buildSourceAlignmentProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordSourceAlignmentProofOptions(RecordSourceAlignmentProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSourceAlignmentProofFacts,
		Summary:                "source alignment reviewed",
		EvidenceURI:            "local:source-alignment-review",
		SourceAlignmentBinding: binding,
	}))
	drifted := copyMap(binding)
	paths := metadataStringSlice(drifted, "source_alignment_source_paths")
	if len(paths) == 0 {
		t.Fatalf("source alignment test binding missing paths: %+v", drifted)
	}
	hashes := sourceAlignmentMetadataStringMap(drifted, "source_alignment_source_hashes")
	hashes[paths[0]] = strings.Repeat("a", 64)
	drifted["source_alignment_source_hashes"] = hashes
	drifted["source_alignment_source_set_hash"] = sourceAlignmentSourceSetHash(paths, hashes)

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		SourceAlignmentProof:          &proof,
		SourceAlignmentCurrentBinding: drifted,
	})
	item := findCompletionAuditItem(t, audit, "E1_design_source_alignment")
	if item.Status != "blocked" ||
		item.Metadata["source_alignment_current_binding_bound"] != false ||
		!containsString(item.BlockedBy, "source_alignment_source_set_hash_current_mismatch") {
		t.Fatalf("drifted source alignment binding should block E1: %+v", item)
	}
}

func TestBuildCompletionAuditConsumesTaskMatrixProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 16, 40, 0, 0, time.UTC)
	proof := buildTaskMatrixProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordTaskMatrixProofOptions(withTaskMatrixProofTestBinding(RecordTaskMatrixProofOptions{
		ProofStatus: "complete",
		Facts:       requiredTaskMatrixProofFacts,
		Summary:     "task matrix reviewed",
		EvidenceURI: "local:task-matrix-review",
	})))
	proof.EventID = 94

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		TaskMatrixProof:          &proof,
		TaskMatrixCurrentBinding: taskMatrixProofTestCurrentBinding(),
	})
	item := findCompletionAuditItem(t, audit, "E2_phase_task_matrix")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("task matrix item should be complete: %+v", item)
	}
	if item.Metadata["task_matrix_gate_passed"] != true ||
		item.Metadata["task_matrix_status"] != "complete" ||
		item.Metadata["latest_task_matrix_proof_event_id"] != int64(94) ||
		item.Metadata["latest_task_matrix_proof_evidence_uri"] != "local:task-matrix-review" ||
		item.Metadata["task_matrix_binding_status"] != "pass" ||
		item.Metadata["task_matrix_current_binding_bound"] != true ||
		item.Metadata["planned_v1_required_task_count"] != int64(0) ||
		item.Metadata["missing_evidence_v1_required_task_count"] != int64(0) ||
		item.Metadata["blocked_v1_required_task_count"] != int64(0) {
		t.Fatalf("task matrix proof metadata missing: %+v", item.Metadata)
	}
	if audit.Status != "blocked" {
		t.Fatalf("task matrix proof alone must not complete full audit: %+v", audit)
	}
}

func TestBuildCompletionAuditRejectsLooseTaskMatrixProof(t *testing.T) {
	proof := TaskMatrixProof{
		Project:      Record{ID: 1, Key: "areamatrix"},
		Status:       "recorded",
		ProofStatus:  "complete",
		Decision:     "allowed",
		Facts:        requiredTaskMatrixProofFacts,
		MissingFacts: []string{},
		Metadata:     map[string]any{"summary": "task matrix reviewed", "evidence_uri": "local:task-matrix-review"},
	}

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		TaskMatrixProof:          &proof,
		TaskMatrixCurrentBinding: taskMatrixProofTestCurrentBinding(),
	})
	item := findCompletionAuditItem(t, audit, "E2_phase_task_matrix")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "task_matrix_binding_incomplete") ||
		item.Metadata["task_matrix_binding_status"] != "" {
		t.Fatalf("loose task matrix proof should be blocked by binding: %+v", item)
	}
	blockers, ok := item.Metadata["task_matrix_binding_blockers"].([]string)
	if !ok || !containsString(blockers, "task_matrix_binding_status_not_pass") ||
		!containsString(blockers, "task_matrix_source_set_hash_missing_or_mismatch") {
		t.Fatalf("loose task matrix binding blockers missing: %+v", item.Metadata)
	}
}

func TestBuildCompletionAuditAggregatesTaskAndImplementationGapStatus(t *testing.T) {
	sourceBinding := sourceAlignmentProofTestCurrentBinding(t)
	sourceProof := buildSourceAlignmentProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordSourceAlignmentProofOptions(RecordSourceAlignmentProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSourceAlignmentProofFacts,
		Summary:                "release candidate source alignment reviewed",
		EvidenceURI:            "docs/development/source-alignment-release-candidate-evidence.md",
		SourceAlignmentBinding: sourceBinding,
	}))
	taskProof := buildTaskMatrixProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordTaskMatrixProofOptions(withTaskMatrixProofTestBinding(RecordTaskMatrixProofOptions{
		ProofStatus: "complete",
		Facts:       requiredTaskMatrixProofFacts,
		Summary:     "release candidate task matrix reviewed",
		EvidenceURI: "docs/development/task-matrix-release-candidate-evidence.md",
	})))

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		SourceAlignmentProof:          &sourceProof,
		SourceAlignmentCurrentBinding: sourceBinding,
		TaskMatrixProof:               &taskProof,
		TaskMatrixCurrentBinding:      taskMatrixProofTestCurrentBinding(),
	})
	if audit.TaskMatrixStatus != "complete" || audit.ImplementationGapStatus != "complete" {
		t.Fatalf("E1/E2 complete proofs should complete task and implementation gap aggregates: %+v", audit)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		TaskMatrixProof:          &taskProof,
		TaskMatrixCurrentBinding: taskMatrixProofTestCurrentBinding(),
	})
	if audit.TaskMatrixStatus != "complete" || audit.ImplementationGapStatus != "incomplete" {
		t.Fatalf("implementation gap aggregate should stay incomplete without E1 proof: %+v", audit)
	}
}

func TestBuildCompletionAuditConsumesSecurityClosureProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 17, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	binding := readySecurityClosureCurrentBinding(record)
	proof := buildSecurityClosureProof(record, normalizeRecordSecurityClosureProofOptions(RecordSecurityClosureProofOptions{
		ProofStatus:            "complete",
		Facts:                  requiredSecurityClosureProofFacts,
		Summary:                "security closure reviewed",
		EvidenceURI:            "local:security-closure-review",
		SecurityClosureBinding: binding.Metadata,
	}))
	proof.EventID = 95

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		SecurityBoundaryReadiness:     &binding.SecurityBoundaryReadiness,
		SecurityClosureProof:          &proof,
		SecurityClosureCurrentBinding: binding.Metadata,
	})
	item := findCompletionAuditItem(t, audit, "E8_security_permission_isolation")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("security closure item should be complete: %+v", item)
	}
	if item.Metadata["security_closure_gate_passed"] != true ||
		item.Metadata["security_closure_proof_status"] != "complete" ||
		item.Metadata["latest_security_closure_proof_event_id"] != int64(95) ||
		item.Metadata["latest_security_closure_proof_evidence_uri"] != "local:security-closure-review" ||
		item.Metadata["security_closure_binding_status"] != "pass" ||
		item.Metadata["security_closure_current_binding_bound"] != true {
		t.Fatalf("security closure proof metadata missing: %+v", item.Metadata)
	}
	if item.Metadata["project_write_attempted"] != false ||
		item.Metadata["execution_write_attempted"] != false ||
		item.Metadata["authorization_changed"] != false ||
		item.Metadata["secret_plaintext_read"] != false ||
		item.Metadata["remote_worker_credentials_issued"] != false ||
		item.Metadata["commands_run"] != false ||
		item.Metadata["area_matrix_protected_paths_touched"] != false {
		t.Fatalf("security closure proof safety metadata missing: %+v", item.Metadata)
	}
	if audit.Status != "blocked" {
		t.Fatalf("security closure proof alone must not complete full audit: %+v", audit)
	}

	readiness := binding.SecurityBoundaryReadiness
	readiness.SecretResolveOpen = true
	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		SecurityBoundaryReadiness:     &readiness,
		SecurityClosureProof:          &proof,
		SecurityClosureCurrentBinding: binding.Metadata,
	})
	item = findCompletionAuditItem(t, audit, "E8_security_permission_isolation")
	if item.Status != "blocked" || !containsString(item.BlockedBy, "security_boundary_opened_forbidden_capability") {
		t.Fatalf("forbidden security opening must keep E8 blocked: %+v", item)
	}

	driftedBinding := map[string]any{}
	for key, value := range binding.Metadata {
		driftedBinding[key] = value
	}
	driftedBinding["permission_doctor_warn_count"] = int64(1)
	driftedBinding["security_closure_binding_hash"] = securityClosureBindingHash(driftedBinding)
	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		SecurityBoundaryReadiness:     &binding.SecurityBoundaryReadiness,
		SecurityClosureProof:          &proof,
		SecurityClosureCurrentBinding: driftedBinding,
	})
	item = findCompletionAuditItem(t, audit, "E8_security_permission_isolation")
	if item.Status != "blocked" || !containsString(item.BlockedBy, "current_permission_doctor_warn_count_nonzero") {
		t.Fatalf("current security closure binding drift must block E8: %+v", item)
	}
}

func TestBuildCompletionAuditConsumesBackupRestoreProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 17, 20, 0, 0, time.UTC)
	proof := buildBackupRestoreProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordBackupRestoreProofOptions(withBackupRestoreEvidenceBinding(RecordBackupRestoreProofOptions{
		ProofStatus: "complete",
		Facts:       requiredBackupRestoreProofFacts,
		Summary:     "backup restore reviewed",
		EvidenceURI: "local:backup-restore-review",
	})))
	proof.EventID = 96

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		BackupRestoreProof:          &proof,
		BackupRestoreCurrentBinding: matchingBackupRestoreCurrentBinding(),
	})
	item := findCompletionAuditItem(t, audit, "E6_backup_restore_artifact_retention")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("backup restore item should be complete: %+v", item)
	}
	if item.Metadata["backup_restore_gate_passed"] != true ||
		item.Metadata["backup_restore_proof_status"] != "complete" ||
		item.Metadata["latest_backup_restore_proof_event_id"] != int64(96) ||
		item.Metadata["latest_backup_restore_proof_evidence_uri"] != "local:backup-restore-review" ||
		item.Metadata["backup_restore_evidence_binding_status"] != "pass" ||
		item.Metadata["backup_manifest_hash"] != strings.Repeat("b", 64) ||
		item.Metadata["current_backup_manifest_hash"] != strings.Repeat("c", 64) ||
		item.Metadata["restore_plan_status"] != "needs_attention" ||
		item.Metadata["artifact_integrity_status"] != "warn" ||
		item.Metadata["artifact_archive_preview_status"] != "needs_attention" ||
		item.Metadata["artifact_archive_preview_external_refs"] != int64(1) ||
		item.Metadata["backup_restore_current_binding_bound"] != true {
		t.Fatalf("backup restore proof metadata missing: %+v", item.Metadata)
	}
	if item.Metadata["project_write_attempted"] != false ||
		item.Metadata["execution_write_attempted"] != false ||
		item.Metadata["database_restore_attempted"] != false ||
		item.Metadata["artifact_bytes_copied"] != false ||
		item.Metadata["artifact_bytes_deleted"] != false ||
		item.Metadata["artifact_bytes_uploaded"] != false ||
		item.Metadata["artifact_gc_attempted"] != false ||
		item.Metadata["commands_run"] != false ||
		item.Metadata["area_matrix_protected_paths_touched"] != false {
		t.Fatalf("backup restore proof safety metadata missing: %+v", item.Metadata)
	}
	if audit.Status != "blocked" {
		t.Fatalf("backup restore proof alone must not complete full audit: %+v", audit)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		BackupRestoreProof:          &proof,
		BackupRestoreCurrentBinding: driftedBackupRestoreCurrentBinding(),
	})
	item = findCompletionAuditItem(t, audit, "E6_backup_restore_artifact_retention")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "backup_restore_proof_current_binding_mismatch") ||
		!containsString(item.BlockedBy, "artifact_integrity_status_changed") ||
		!containsString(item.BlockedBy, "current_artifact_integrity_status_not_pass_or_warn") ||
		item.Metadata["backup_restore_current_binding_bound"] != false {
		t.Fatalf("current backup restore binding drift must block E6: %+v", item)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		BackupRestoreProof:               &proof,
		BackupRestoreCurrentBindingError: "read current binding failed",
	})
	item = findCompletionAuditItem(t, audit, "E6_backup_restore_artifact_retention")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "backup_restore_current_binding_query_failed") ||
		item.Metadata["backup_restore_current_binding_bound"] != false {
		t.Fatalf("current backup restore binding query failure must block E6: %+v", item)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		BackupRestoreProof: &proof,
	})
	item = findCompletionAuditItem(t, audit, "E6_backup_restore_artifact_retention")
	if item.Status != "blocked" ||
		!containsString(item.BlockedBy, "backup_restore_current_binding_missing") ||
		item.Metadata["backup_restore_current_binding_bound"] != false {
		t.Fatalf("missing current backup restore binding must block E6: %+v", item)
	}

	looseProof := buildBackupRestoreProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordBackupRestoreProofOptions(RecordBackupRestoreProofOptions{
		ProofStatus: "complete",
		Facts:       requiredBackupRestoreProofFacts,
		Summary:     "legacy backup restore reviewed",
		EvidenceURI: "local:legacy-backup-restore-review",
	}))
	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		BackupRestoreProof: &looseProof,
	})
	item = findCompletionAuditItem(t, audit, "E6_backup_restore_artifact_retention")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "backup_restore_evidence_binding_status_not_pass") {
		t.Fatalf("legacy loose backup restore proof must not complete E6: %+v", item)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseFinalGate: &ReleaseFinalGate{
			Readiness: ReleaseReadiness{
				Status:      "ready",
				Backup:      BackupManifest{Status: "ready"},
				RestorePlan: RestorePlan{Status: "ready"},
			},
		},
	})
	item = findCompletionAuditItem(t, audit, "E6_backup_restore_artifact_retention")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "backup_restore_proof_missing") ||
		item.Metadata["release_readiness_is_sufficient_for_e6"] != false {
		t.Fatalf("release readiness alone must not complete E6 without backup restore proof: %+v", item)
	}
}

func TestBuildCompletionAuditConsumesReleasePackagingProof(t *testing.T) {
	generated := time.Date(2026, 7, 3, 17, 40, 0, 0, time.UTC)
	bundle := readyReleaseEvidenceBundle()
	proof := buildReleasePackagingProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordReleasePackagingProofOptions(RecordReleasePackagingProofOptions{
		ProofStatus: "complete",
		Facts:       requiredReleasePackagingProofFacts,
		Summary:     "release packaging reviewed",
		EvidenceURI: "local:release-packaging-review",
		Metadata:    ReleaseEvidenceBundleBindingMetadata(bundle),
	}))
	proof.EventID = 97

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseEvidenceBundle: &bundle,
		ReleaseFinalGate:      &ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		ReleasePackagingProof: &proof,
	})
	item := findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("release final gate plus packaging proof should complete E5: %+v", item)
	}
	if audit.ReleaseFinalGateStatus != "complete" {
		t.Fatalf("release aggregate should require final gate pass and release packaging proof: %+v", audit)
	}
	if item.Metadata["release_packaging_gate_passed"] != true ||
		item.Metadata["release_packaging_proof_status"] != "complete" ||
		item.Metadata["latest_release_packaging_proof_event_id"] != int64(97) ||
		item.Metadata["latest_release_packaging_proof_evidence_uri"] != "local:release-packaging-review" ||
		item.Metadata["current_release_evidence_bundle_hash"] != bundle.BundleHash ||
		item.Metadata["proof_release_evidence_bundle_hash"] != bundle.BundleHash ||
		item.Metadata["release_packaging_proof_bundle_bound"] != true {
		t.Fatalf("release packaging proof metadata missing: %+v", item.Metadata)
	}
	if item.Metadata["project_write_attempted"] != false ||
		item.Metadata["execution_write_attempted"] != false ||
		item.Metadata["release_package_created"] != false ||
		item.Metadata["release_state_written"] != false ||
		item.Metadata["release_approval_created"] != false ||
		item.Metadata["rollout_state_created"] != false ||
		item.Metadata["migration_apply_attempted"] != false ||
		item.Metadata["tag_created"] != false ||
		item.Metadata["package_signed"] != false ||
		item.Metadata["artifact_uploaded"] != false ||
		item.Metadata["git_push_attempted"] != false ||
		item.Metadata["publish_attempted"] != false ||
		item.Metadata["commands_run"] != false ||
		item.Metadata["area_matrix_protected_paths_touched"] != false {
		t.Fatalf("release packaging proof safety metadata missing: %+v", item.Metadata)
	}
	if audit.Status != "blocked" {
		t.Fatalf("release packaging proof alone must not complete full audit: %+v", audit)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseEvidenceBundle: &bundle,
		ReleasePackagingProof: &proof,
	})
	item = findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "release_final_gate_not_passed") ||
		containsString(item.BlockedBy, "release_packaging_proof_missing") ||
		item.Metadata["release_packaging_proof_recorded"] != true {
		t.Fatalf("release packaging proof alone must not complete E5 without current final gate pass: %+v", item)
	}
	if audit.ReleaseFinalGateStatus != "incomplete" {
		t.Fatalf("release aggregate should stay incomplete without current final gate pass: %+v", audit)
	}

	audit = BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseFinalGate:      &ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		ReleasePackagingProof: &proof,
	})
	item = findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "release_evidence_bundle_missing") {
		t.Fatalf("release packaging proof must bind the current release evidence bundle: %+v", item)
	}
}

func TestBuildCompletionAuditRejectsReleasePackagingProofBundleMismatch(t *testing.T) {
	generated := time.Date(2026, 7, 3, 17, 45, 0, 0, time.UTC)
	bundle := readyReleaseEvidenceBundle()
	metadata := ReleaseEvidenceBundleBindingMetadata(bundle)
	metadata["release_evidence_bundle_hash"] = "old-bundle-hash"
	proof := buildReleasePackagingProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordReleasePackagingProofOptions(RecordReleasePackagingProofOptions{
		ProofStatus: "complete",
		Facts:       requiredReleasePackagingProofFacts,
		Summary:     "release packaging reviewed",
		EvidenceURI: "local:release-packaging-review",
		Metadata:    metadata,
	}))

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseEvidenceBundle: &bundle,
		ReleaseFinalGate:      &ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		ReleasePackagingProof: &proof,
	})
	item := findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "release_packaging_proof_release_evidence_bundle_hash_mismatch") ||
		item.Metadata["release_packaging_proof_bundle_bound"] != false {
		t.Fatalf("stale release evidence bundle binding must not complete E5: %+v", item)
	}
	if audit.ReleaseFinalGateStatus != "incomplete" {
		t.Fatalf("release aggregate should stay incomplete on stale bundle binding: %+v", audit)
	}
}

func TestReleasePackagingProofFromEventPreservesBundleBinding(t *testing.T) {
	generated := time.Date(2026, 7, 3, 17, 50, 0, 0, time.UTC)
	bundle := readyReleaseEvidenceBundle()
	options := normalizeRecordReleasePackagingProofOptions(RecordReleasePackagingProofOptions{
		ProofStatus: "complete",
		Facts:       requiredReleasePackagingProofFacts,
		Summary:     "release packaging reviewed",
		EvidenceURI: "local:release-packaging-review",
		Metadata:    ReleaseEvidenceBundleBindingMetadata(bundle),
	})
	proof := buildReleasePackagingProof(Record{ID: 1, Key: "areamatrix"}, options)
	proof.EventID = 97
	restored := releasePackagingProofFromEvent(EventRecord{
		ID:        97,
		ProjectID: 1,
		Metadata:  releasePackagingProofEventMetadata(proof, options),
	})

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseEvidenceBundle: &bundle,
		ReleaseFinalGate:      &ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		ReleasePackagingProof: &restored,
	})
	item := findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status != "complete" ||
		item.Metadata["proof_release_evidence_bundle_hash"] != bundle.BundleHash ||
		item.Metadata["release_packaging_proof_bundle_bound"] != true {
		t.Fatalf("event-restored release packaging proof should preserve bundle binding: %+v", item)
	}
}

func TestBuildCompletionAuditReleasePackagingBindingDoesNotRequireRealProjectIdentity(t *testing.T) {
	generated := time.Date(2026, 7, 3, 17, 55, 0, 0, time.UTC)
	bundle := fixtureReleaseEvidenceBundle()
	proof := buildReleasePackagingProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordReleasePackagingProofOptions(RecordReleasePackagingProofOptions{
		ProofStatus: "complete",
		Facts:       requiredReleasePackagingProofFacts,
		Summary:     "fixture release packaging reviewed",
		EvidenceURI: "local:fixture-release-packaging-review",
		Metadata:    ReleaseEvidenceBundleBindingMetadata(bundle),
	}))

	audit := BuildCompletionAudit(CompletionAuditOptions{GeneratedAt: generated}, CompletionAuditParts{
		ReleaseEvidenceBundle: &bundle,
		ReleaseFinalGate:      &ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		ReleasePackagingProof: &proof,
	})
	item := findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status != "complete" ||
		containsString(item.BlockedBy, "release_evidence_bundle_project_root_not_real_areamatrix") {
		t.Fatalf("E5 binding should not enforce release-candidate project identity gate: %+v", item)
	}
}

func TestBuildCompletionAuditReleaseFinalGatePassStillNeedsPackagingProof(t *testing.T) {
	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		ReleaseFinalGate: &ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
	})
	if audit.ReleaseFinalGateStatus != "incomplete" {
		t.Fatalf("release aggregate should remain incomplete without release packaging proof: %+v", audit)
	}
	item := findCompletionAuditItem(t, audit, "E5_release_packaging_preview")
	if item.Status != "incomplete" ||
		containsString(item.BlockedBy, "release_final_gate_not_passed") ||
		!containsString(item.BlockedBy, "release_packaging_proof_missing") {
		t.Fatalf("release final gate pass should still require packaging proof: %+v", item)
	}
}

func TestBuildArchiveProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(RecordArchiveProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"historical_workflow_versions_marked_immutable"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete archive proof should list missing facts: %+v", result)
	}
	if archiveProofCompletesAudit(result) {
		t.Fatalf("incomplete archive proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordArchiveProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"historical_workflow_versions_marked_immutable"},
	})
	if err == nil {
		t.Fatal("complete archive proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordArchiveProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: "local:archive-gate-review",
	})
	if err == nil || !strings.Contains(err.Error(), "complete archive proof missing archive scope binding") {
		t.Fatalf("complete archive proof without scope binding should fail before database access: %v", err)
	}

	complete := buildArchiveProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordArchiveProofOptions(withArchiveProofTestBinding(RecordArchiveProofOptions{
		ProofStatus: "complete",
		Facts:       requiredArchiveProofFacts,
		Summary:     "archive gate reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-archive"),
	})))
	if archiveProofCompletesAudit(complete) {
		t.Fatalf("complete archive proof without event id must not complete E4 archive portion: %+v", complete)
	}
	complete.EventID = 90
	if !archiveProofCompletesAudit(complete) {
		t.Fatalf("complete archive proof with scope binding should complete E4 archive portion: %+v", complete)
	}
}

func TestBuildShimRetirementProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(RecordShimRetirementProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"archive_gate_passed"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete shim proof should list missing facts: %+v", result)
	}
	if shimRetirementProofCompletesAudit(result) {
		t.Fatalf("incomplete shim proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordShimRetirementProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"archive_gate_passed"},
	})
	if err == nil {
		t.Fatal("complete shim proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordShimRetirementProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: "local:shim-retirement-review",
	})
	if err == nil || !strings.Contains(err.Error(), "complete shim retirement proof missing shim retirement scope binding") {
		t.Fatalf("complete shim proof without scope binding should fail before database access: %v", err)
	}

	complete := buildShimRetirementProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordShimRetirementProofOptions(withShimRetirementProofTestBinding(RecordShimRetirementProofOptions{
		ProofStatus: "complete",
		Facts:       requiredShimRetirementProofFacts,
		Summary:     "shim retirement reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-shim-retirement"),
	})))
	if shimRetirementProofCompletesAudit(complete) {
		t.Fatalf("complete shim proof without event id must not complete E4 shim portion: %+v", complete)
	}
	complete.EventID = 91
	if !shimRetirementProofCompletesAudit(complete) {
		t.Fatalf("complete shim proof with scope binding should complete E4 shim portion: %+v", complete)
	}
}

func TestBuildExecutionCutoverProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(RecordExecutionCutoverProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"explicit_execution_cutover_approval_recorded"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete execution cutover proof should list missing facts: %+v", result)
	}
	if executionCutoverProofCompletesAudit(result) {
		t.Fatalf("incomplete execution cutover proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordExecutionCutoverProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"explicit_execution_cutover_approval_recorded"},
	})
	if err == nil {
		t.Fatal("complete execution cutover proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordExecutionCutoverProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: "local:execution-cutover-review",
	})
	if err == nil || !strings.Contains(err.Error(), "execution cutover scope binding") ||
		!strings.Contains(err.Error(), "execution_cutover_scope_missing_or_mismatch") ||
		!strings.Contains(err.Error(), "allowed_task_types_missing_or_mismatch") {
		t.Fatalf("complete execution cutover proof without scope binding should fail before database access: %v", err)
	}

	unsafeOptions := withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus:     "complete",
		Facts:           requiredExecutionCutoverProofFacts,
		Summary:         "execution cutover reviewed",
		EvidenceURI:     "local:execution-cutover-review",
		SourceWriteOpen: true,
	})
	_, err = Store{}.RecordExecutionCutoverProof(nil, Record{ID: 1, Key: "areamatrix"}, unsafeOptions)
	if err == nil || !strings.Contains(err.Error(), "source_write_open") {
		t.Fatalf("complete execution cutover proof with source write open should fail before database access: %v", err)
	}

	complete := buildExecutionCutoverProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts:       requiredExecutionCutoverProofFacts,
		Summary:     "execution cutover reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover"),
	})))
	if metadataString(complete.Metadata, "execution_cutover_binding_contract") != executionCutoverProofBindingContract ||
		!looksLikeSHA256(metadataString(complete.Metadata, "allowed_task_types_hash")) ||
		!looksLikeSHA256(metadataString(complete.Metadata, "forbidden_actions_hash")) ||
		!looksLikeSHA256(metadataString(complete.Metadata, "execution_cutover_scope_binding_hash")) {
		t.Fatalf("complete execution cutover proof should expose deterministic binding hashes: %+v", complete.Metadata)
	}
	if executionCutoverProofCompletesAudit(complete) {
		t.Fatalf("complete execution cutover proof without event id must not complete E4 cutover portion: %+v", complete)
	}
	complete.EventID = 92
	if !executionCutoverProofCompletesAudit(complete) {
		t.Fatalf("complete execution cutover proof with scope binding should complete E4 cutover portion: %+v", complete)
	}
}

func TestBuildValidationProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildValidationProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordValidationProofOptions(RecordValidationProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"go_test_passed"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete validation proof should list missing facts: %+v", result)
	}
	if validationProofCompletesAudit(result) {
		t.Fatalf("incomplete validation proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordValidationProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordValidationProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"go_test_passed"},
	})
	if err == nil {
		t.Fatal("complete validation proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordValidationProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordValidationProofOptions{
		ProofStatus: "complete",
		Facts:       requiredValidationProofFacts,
		Summary:     "fresh validation proof reviewed",
		EvidenceURI: "local:fresh-validation-proof",
	})
	if err == nil || !strings.Contains(err.Error(), "complete validation proof missing validation evidence binding") ||
		!strings.Contains(err.Error(), "validation_commands_missing") ||
		!strings.Contains(err.Error(), "validation_result_hash_invalid") {
		t.Fatalf("complete validation proof without binding should fail before database access: %v", err)
	}
}

func TestBuildSourceAlignmentProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildSourceAlignmentProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordSourceAlignmentProofOptions(RecordSourceAlignmentProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"zero_to_hundred_phases_aligned"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete source alignment proof should list missing facts: %+v", result)
	}
	if sourceAlignmentProofCompletesAudit(result) {
		t.Fatalf("incomplete source alignment proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordSourceAlignmentProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordSourceAlignmentProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"zero_to_hundred_phases_aligned"},
	})
	if err == nil {
		t.Fatal("complete source alignment proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordSourceAlignmentProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordSourceAlignmentProofOptions{
		ProofStatus: "complete",
		Facts:       requiredSourceAlignmentProofFacts,
		Summary:     "source alignment reviewed",
		EvidenceURI: "local:source-alignment-review",
	})
	if err == nil || !strings.Contains(err.Error(), "source_alignment_binding_missing") {
		t.Fatalf("complete source alignment proof without binding should fail before database access: %v", err)
	}
}

func TestBuildTaskMatrixProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildTaskMatrixProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordTaskMatrixProofOptions(RecordTaskMatrixProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"all_v0_v1_tasks_have_status_evidence_and_boundary"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete task matrix proof should list missing facts: %+v", result)
	}
	if taskMatrixProofCompletesAudit(result) {
		t.Fatalf("incomplete task matrix proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordTaskMatrixProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordTaskMatrixProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"all_v0_v1_tasks_have_status_evidence_and_boundary"},
	})
	if err == nil {
		t.Fatal("complete task matrix proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordTaskMatrixProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordTaskMatrixProofOptions{
		ProofStatus: "complete",
		Facts:       requiredTaskMatrixProofFacts,
		Summary:     "task matrix reviewed",
		EvidenceURI: "local:task-matrix-review",
	})
	if err == nil || !strings.Contains(err.Error(), "complete task matrix proof missing task matrix binding") {
		t.Fatalf("complete task matrix proof without binding should fail before database access: %v", err)
	}
}

func withTaskMatrixProofTestBinding(options RecordTaskMatrixProofOptions) RecordTaskMatrixProofOptions {
	backlogHash := strings.Repeat("a", 64)
	statusAuditHash := strings.Repeat("b", 64)
	options.TaskBacklogHash = backlogHash
	options.TaskStatusAuditHash = statusAuditHash
	options.TaskMatrixSourceSetHash = taskMatrixProofSourceSetHash(backlogHash, statusAuditHash)
	options.PlannedV1RequiredTaskCount = 0
	options.PlannedV1RequiredTaskCountSet = true
	options.MissingEvidenceV1RequiredTaskCount = 0
	options.MissingEvidenceV1RequiredTaskCountSet = true
	options.BlockedV1RequiredTaskCount = 0
	options.BlockedV1RequiredTaskCountSet = true
	return options
}

func taskMatrixProofTestCurrentBinding() map[string]any {
	return taskMatrixProofBindingMetadata(
		strings.Repeat("a", 64),
		strings.Repeat("b", 64),
		0,
		0,
		0,
		true,
		nil,
	)
}

func withCurrentTaskMatrixProofBinding(t *testing.T, options RecordTaskMatrixProofOptions) RecordTaskMatrixProofOptions {
	t.Helper()
	binding, err := TaskMatrixCurrentBinding()
	if err != nil {
		t.Fatalf("load current task matrix binding: %v", err)
	}
	options.TaskMatrixSourceSetHash = metadataString(binding, "task_matrix_source_set_hash")
	options.TaskBacklogHash = metadataString(binding, "task_backlog_hash")
	options.TaskStatusAuditHash = metadataString(binding, "task_status_audit_hash")
	options.PlannedV1RequiredTaskCount = metadataInt64(binding, "planned_v1_required_task_count")
	options.PlannedV1RequiredTaskCountSet = true
	options.MissingEvidenceV1RequiredTaskCount = metadataInt64(binding, "missing_evidence_v1_required_task_count")
	options.MissingEvidenceV1RequiredTaskCountSet = true
	options.BlockedV1RequiredTaskCount = metadataInt64(binding, "blocked_v1_required_task_count")
	options.BlockedV1RequiredTaskCountSet = true
	return options
}

func TestBuildSecurityClosureProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildSecurityClosureProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordSecurityClosureProofOptions(RecordSecurityClosureProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"project_key_isolation_covers_workflow_run_lease_artifact_secret_audit"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete security closure proof should list missing facts: %+v", result)
	}
	if securityClosureProofCompletesAudit(result) {
		t.Fatalf("incomplete security closure proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordSecurityClosureProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordSecurityClosureProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"project_key_isolation_covers_workflow_run_lease_artifact_secret_audit"},
	})
	if err == nil {
		t.Fatal("complete security closure proof with missing facts should fail before database access")
	}

	loose := buildSecurityClosureProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordSecurityClosureProofOptions(RecordSecurityClosureProofOptions{
		ProofStatus: "complete",
		Facts:       requiredSecurityClosureProofFacts,
		Summary:     "security closure reviewed without binding",
		EvidenceURI: "local:security-closure-review",
	}))
	if securityClosureProofCompletesAudit(loose) {
		t.Fatalf("complete security closure proof without binding must not complete audit: %+v", loose)
	}

	_, err = Store{}.RecordSecurityClosureProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordSecurityClosureProofOptions{
		ProofStatus: "complete",
		Facts:       requiredSecurityClosureProofFacts,
		Summary:     "security closure reviewed without binding",
		EvidenceURI: "local:security-closure-review",
	})
	if err == nil || !strings.Contains(err.Error(), "security_closure_binding_missing") {
		t.Fatalf("complete security closure proof without binding should fail before database access, got %v", err)
	}
}

func TestBuildBackupRestoreProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildBackupRestoreProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordBackupRestoreProofOptions(RecordBackupRestoreProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete backup restore proof should list missing facts: %+v", result)
	}
	if backupRestoreProofCompletesAudit(result) {
		t.Fatalf("incomplete backup restore proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordBackupRestoreProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordBackupRestoreProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata"},
	})
	if err == nil {
		t.Fatal("complete backup restore proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordBackupRestoreProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordBackupRestoreProofOptions{
		ProofStatus: "complete",
		Facts:       requiredBackupRestoreProofFacts,
		Summary:     "backup restore reviewed",
		EvidenceURI: "local:backup-restore-review",
	})
	if err == nil || !strings.Contains(err.Error(), "backup/restore/artifact output binding") {
		t.Fatalf("complete backup restore proof without output binding should fail before database access, err=%v", err)
	}

	complete := buildBackupRestoreProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordBackupRestoreProofOptions(withBackupRestoreEvidenceBinding(RecordBackupRestoreProofOptions{
		ProofStatus: "complete",
		Facts:       requiredBackupRestoreProofFacts,
		Summary:     "backup restore reviewed",
		EvidenceURI: "local:backup-restore-review",
	})))
	if !backupRestoreProofCompletesAudit(complete) {
		t.Fatalf("complete backup restore proof with output binding should complete E6: %+v", complete)
	}
}

func TestBuildReleasePackagingProofRequiresAllFactsForComplete(t *testing.T) {
	result := buildReleasePackagingProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordReleasePackagingProofOptions(RecordReleasePackagingProofOptions{
		ProofStatus: "incomplete",
		Facts:       []string{"release_final_gate_passed"},
	}))
	if result.Status != "recorded" || result.Decision != "needs_attention" || len(result.MissingFacts) == 0 {
		t.Fatalf("incomplete release packaging proof should list missing facts: %+v", result)
	}
	if releasePackagingProofCompletesAudit(result) {
		t.Fatalf("incomplete release packaging proof must not complete audit: %+v", result)
	}

	_, err := Store{}.RecordReleasePackagingProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordReleasePackagingProofOptions{
		ProofStatus: "complete",
		Facts:       []string{"release_final_gate_passed"},
	})
	if err == nil {
		t.Fatal("complete release packaging proof with missing facts should fail before database access")
	}

	_, err = Store{}.RecordReleasePackagingProof(nil, Record{ID: 1, Key: "areamatrix"}, RecordReleasePackagingProofOptions{
		ProofStatus: "complete",
		Facts:       requiredReleasePackagingProofFacts,
		Summary:     "release packaging reviewed",
		EvidenceURI: "local:release-packaging-review",
	})
	if err == nil || !strings.Contains(err.Error(), "release_evidence_bundle_hash_missing") {
		t.Fatalf("complete release packaging proof should require release evidence bundle binding before database access: %v", err)
	}
}

func TestRecordProofRequiresEvidenceForCompletingStatuses(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "archive complete",
			run: func() error {
				_, err := Store{}.RecordArchiveProof(nil, record, RecordArchiveProofOptions{ProofStatus: "complete", Facts: requiredArchiveProofFacts})
				return err
			},
		},
		{
			name: "shim retirement complete",
			run: func() error {
				_, err := Store{}.RecordShimRetirementProof(nil, record, RecordShimRetirementProofOptions{ProofStatus: "complete", Facts: requiredShimRetirementProofFacts})
				return err
			},
		},
		{
			name: "execution cutover complete",
			run: func() error {
				_, err := Store{}.RecordExecutionCutoverProof(nil, record, RecordExecutionCutoverProofOptions{ProofStatus: "complete", Facts: requiredExecutionCutoverProofFacts})
				return err
			},
		},
		{
			name: "validation complete",
			run: func() error {
				_, err := Store{}.RecordValidationProof(nil, record, RecordValidationProofOptions{ProofStatus: "complete", Facts: requiredValidationProofFacts})
				return err
			},
		},
		{
			name: "source alignment complete",
			run: func() error {
				_, err := Store{}.RecordSourceAlignmentProof(nil, record, RecordSourceAlignmentProofOptions{ProofStatus: "complete", Facts: requiredSourceAlignmentProofFacts})
				return err
			},
		},
		{
			name: "task matrix complete",
			run: func() error {
				_, err := Store{}.RecordTaskMatrixProof(nil, record, RecordTaskMatrixProofOptions{ProofStatus: "complete", Facts: requiredTaskMatrixProofFacts})
				return err
			},
		},
		{
			name: "security closure complete",
			run: func() error {
				_, err := Store{}.RecordSecurityClosureProof(nil, record, RecordSecurityClosureProofOptions{ProofStatus: "complete", Facts: requiredSecurityClosureProofFacts})
				return err
			},
		},
		{
			name: "backup restore complete",
			run: func() error {
				_, err := Store{}.RecordBackupRestoreProof(nil, record, RecordBackupRestoreProofOptions{ProofStatus: "complete", Facts: requiredBackupRestoreProofFacts})
				return err
			},
		},
		{
			name: "release packaging complete",
			run: func() error {
				_, err := Store{}.RecordReleasePackagingProof(nil, record, RecordReleasePackagingProofOptions{ProofStatus: "complete", Facts: requiredReleasePackagingProofFacts})
				return err
			},
		},
		{
			name: "protected path clean",
			run: func() error {
				_, err := Store{}.RecordProtectedPathProof(nil, record, RecordProtectedPathProofOptions{ProofStatus: "clean"})
				return err
			},
		},
		{
			name: "protected path authorized",
			run: func() error {
				_, err := Store{}.RecordProtectedPathProof(nil, record, RecordProtectedPathProofOptions{ProofStatus: "authorized"})
				return err
			},
		},
		{
			name: "operations smoke pass",
			run: func() error {
				_, err := Store{}.RecordOperationsSmokeProof(nil, record, RecordOperationsSmokeProofOptions{ProofKey: "local_ops_smoke", EvidenceStatus: "pass"})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("completing proof without summary/evidence URI should fail before database access")
			}
			if !strings.Contains(err.Error(), "missing required evidence fields: summary,evidence_uri") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthorizedProtectedPathProofRequiresStructuredAuthorization(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	base := RecordProtectedPathProofOptions{
		ProofStatus: "authorized",
		Summary:     "protected path changes approved",
		EvidenceURI: "docs/development/protected-path-authorization.md",
	}
	_, err := Store{}.RecordProtectedPathProof(nil, record, base)
	if err == nil || !strings.Contains(err.Error(), "authorized protected path proof missing required fields: approval_id,allowed_paths,dirty_output_hash,reviewer,rollback_evidence_uri") {
		t.Fatalf("authorized proof without structured authorization should fail before database access: %v", err)
	}

	invalid := base
	invalid.AuthorizedApprovalID = "approval-1"
	invalid.AuthorizedAllowedPaths = []string{"workflow/README.md"}
	invalid.AuthorizedDirtyOutputHash = "not-a-sha"
	invalid.AuthorizedReviewer = "release-owner"
	invalid.AuthorizedRollbackEvidence = "docs/development/protected-path-rollback.md"
	if err := validateAuthorizedProtectedPathProofOptions(normalizeRecordProtectedPathProofOptions(invalid)); err == nil ||
		!strings.Contains(err.Error(), "dirty_output_hash must be a sha256 hex digest") {
		t.Fatalf("authorized proof with invalid hash should fail: %v", err)
	}

	mismatch := invalid
	mismatch.AuthorizedDirtyOutputHash = strings.Repeat("a", 64)
	mismatch.GitStatusOutput = " M workflow/README.md"
	if err := validateAuthorizedProtectedPathProofOptions(normalizeRecordProtectedPathProofOptions(mismatch)); err == nil ||
		!strings.Contains(err.Error(), "dirty_output_hash does not match git status output hash") {
		t.Fatalf("authorized proof with mismatched dirty output hash should fail: %v", err)
	}

	noOutput := invalid
	noOutput.AuthorizedDirtyOutputHash = strings.Repeat("a", 64)
	if err := validateAuthorizedProtectedPathProofOptions(normalizeRecordProtectedPathProofOptions(noOutput)); err == nil ||
		!strings.Contains(err.Error(), "requires git status output") {
		t.Fatalf("authorized proof without git status output should fail: %v", err)
	}

	unsafePathOutput := " M /tmp/escape"
	unsafePath := invalid
	unsafePath.AuthorizedAllowedPaths = []string{"/tmp/escape"}
	unsafePath.AuthorizedDirtyOutputHash = protectedPathProofOutputHash(unsafePathOutput)
	unsafePath.GitStatusOutput = unsafePathOutput
	if err := validateAuthorizedProtectedPathProofOptions(normalizeRecordProtectedPathProofOptions(unsafePath)); err == nil ||
		!strings.Contains(err.Error(), "allowed_path must be a safe relative AreaMatrix path") {
		t.Fatalf("authorized proof with unsafe allowed path should fail: %v", err)
	}

	outsideProtectedSetOutput := " M docs/release.md"
	outsideProtectedSet := invalid
	outsideProtectedSet.AuthorizedAllowedPaths = []string{"docs/release.md"}
	outsideProtectedSet.AuthorizedDirtyOutputHash = protectedPathProofOutputHash(outsideProtectedSetOutput)
	outsideProtectedSet.GitStatusOutput = outsideProtectedSetOutput
	if err := validateAuthorizedProtectedPathProofOptions(normalizeRecordProtectedPathProofOptions(outsideProtectedSet)); err == nil ||
		!strings.Contains(err.Error(), "allowed_path is outside the AreaMatrix protected path set") {
		t.Fatalf("authorized proof outside protected path set should fail: %v", err)
	}

	uncoveredPathOutput := " M .areaflow/status.json"
	uncoveredPath := invalid
	uncoveredPath.AuthorizedAllowedPaths = []string{"workflow/README.md"}
	uncoveredPath.AuthorizedDirtyOutputHash = protectedPathProofOutputHash(uncoveredPathOutput)
	uncoveredPath.GitStatusOutput = uncoveredPathOutput
	if err := validateAuthorizedProtectedPathProofOptions(normalizeRecordProtectedPathProofOptions(uncoveredPath)); err == nil ||
		!strings.Contains(err.Error(), "git status path is not covered by allowed_path") {
		t.Fatalf("authorized proof with uncovered git status path should fail: %v", err)
	}
}

func TestBuildCompletionAuditConsumesAuthorizedProtectedPathProof(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	gitStatusOutput := " M workflow/README.md"
	proof := buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus:                "authorized",
		Summary:                    "protected path changes approved by release owner",
		EvidenceURI:                "docs/development/protected-path-authorization.md",
		GitStatusOutput:            gitStatusOutput,
		AuthorizedApprovalID:       "approval-123",
		AuthorizedAllowedPaths:     []string{"workflow/README.md"},
		AuthorizedDirtyOutputHash:  protectedPathProofOutputHash(gitStatusOutput),
		AuthorizedReviewer:         "release-owner",
		AuthorizedRollbackEvidence: "docs/development/protected-path-rollback.md",
	}))
	proof.EventID = 104

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ProtectedPathProof: &proof})
	item := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "complete" || len(item.BlockedBy) != 0 {
		t.Fatalf("authorized protected path proof should complete E9: %+v", item)
	}
	if proof.AreaMatrixProtectedPathsTouched != true ||
		item.Metadata["area_matrix_protected_paths_touched"] != true ||
		item.Metadata["authorized_proof_complete"] != true ||
		item.Metadata["authorized_approval_id"] != "approval-123" ||
		item.Metadata["authorized_dirty_output_hash"] != protectedPathProofOutputHash(gitStatusOutput) ||
		item.Metadata["authorized_reviewer"] != "release-owner" ||
		item.Metadata["authorized_rollback_evidence_uri"] != "docs/development/protected-path-rollback.md" ||
		item.Metadata["protected_path_proof_binding_status"] != "pass" ||
		item.Metadata["git_status_output_empty"] != false ||
		item.Metadata["protected_path_set_hash"] != protectedPathProofSetHash() ||
		item.Metadata["protected_path_set_count"] != int64(len(protectedPathProofSet())) {
		t.Fatalf("authorized proof metadata missing: %+v", item.Metadata)
	}
	allowed, ok := item.Metadata["authorized_allowed_paths"].([]string)
	if !ok || len(allowed) != 1 || allowed[0] != "workflow/README.md" {
		t.Fatalf("authorized allowed paths missing: %+v", item.Metadata)
	}
	touched, ok := item.Metadata["authorized_touched_paths"].([]string)
	if !ok || len(touched) != 1 || touched[0] != "workflow/README.md" {
		t.Fatalf("authorized touched paths missing: %+v", item.Metadata)
	}

	options := normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus:                "authorized",
		Summary:                    "protected path changes approved by release owner",
		EvidenceURI:                "docs/development/protected-path-authorization.md",
		GitStatusOutput:            gitStatusOutput,
		AuthorizedApprovalID:       "approval-123",
		AuthorizedAllowedPaths:     []string{"workflow/README.md"},
		AuthorizedDirtyOutputHash:  protectedPathProofOutputHash(gitStatusOutput),
		AuthorizedReviewer:         "release-owner",
		AuthorizedRollbackEvidence: "docs/development/protected-path-rollback.md",
	})
	eventBackedProof := protectedPathProofFromEvent(EventRecord{
		ID:        105,
		ProjectID: record.ID,
		Type:      protectedPathProofEventType,
		Metadata:  protectedPathProofEventMetadata(proof, options),
		CreatedAt: time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC),
	})
	audit = BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ProtectedPathProof: &eventBackedProof})
	item = findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "complete" || item.Metadata["authorized_proof_complete"] != true ||
		item.Metadata["authorized_dirty_output_hash"] != protectedPathProofOutputHash(gitStatusOutput) ||
		item.Metadata["protected_path_proof_binding_status"] != "pass" {
		t.Fatalf("event-backed authorized protected path proof should complete E9: %+v", item)
	}

	tampered := proof
	tampered.Metadata = map[string]any{}
	for key, value := range proof.Metadata {
		tampered.Metadata[key] = value
	}
	tampered.Metadata["authorized_touched_paths"] = []string{".areaflow/status.json"}
	audit = BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ProtectedPathProof: &tampered})
	item = findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status == "complete" ||
		!containsString(item.BlockedBy, "protected_path_proof_authorization_incomplete") ||
		item.Metadata["authorized_proof_complete"] != false {
		t.Fatalf("tampered authorized proof must not complete E9: %+v", item)
	}
}

func TestCompletionAuditRejectsCompletingProofsWithoutTraceableEvidence(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	validationProof := buildValidationProof(record, normalizeRecordValidationProofOptions(RecordValidationProofOptions{
		ProofStatus: "complete",
		Facts:       requiredValidationProofFacts,
		EvidenceURI: "local:validation-without-summary",
	}))
	validationProof.EventID = 101
	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ValidationProof: &validationProof})
	validationItem := findCompletionAuditItem(t, audit, "E3_command_api_smoke_evidence")
	if validationItem.Status == "complete" || !containsString(validationItem.BlockedBy, "validation_proof_incomplete") {
		t.Fatalf("validation proof without traceable evidence must not complete E3: %+v", validationItem)
	}

	protectedProof := ProtectedPathProof{
		Project:                         record,
		Status:                          "recorded",
		ProofStatus:                     "clean",
		Decision:                        "allowed",
		EventID:                         102,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        map[string]any{"evidence_uri": "local:protected-path-without-summary"},
	}
	audit = BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ProtectedPathProof: &protectedProof})
	protectedItem := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if protectedItem.Status == "complete" ||
		!containsString(protectedItem.BlockedBy, "protected_path_proof_evidence_missing") ||
		protectedItem.Metadata["latest_proof_traceable_evidence"] != false {
		t.Fatalf("protected path proof without traceable evidence must not complete E9: %+v", protectedItem)
	}

	now := time.Date(2026, 7, 3, 14, 45, 0, 0, time.UTC)
	operationsItem := operationsLocalBootstrapSmokeItem(OperationsSmokeProof{
		Project:        record,
		ProofKey:       "local_ops_smoke",
		Status:         "recorded",
		EvidenceStatus: "pass",
		Decision:       "allowed",
		EventID:        103,
		CreatedAt:      now,
		Metadata:       map[string]any{"evidence_uri": "local:operations-without-summary"},
	}, now)
	if operationsItem.Status == "ready" ||
		!containsString(operationsItem.BlockedBy, "operations_smoke_proof_evidence_missing") ||
		operationsItem.Metadata["latest_smoke_proof_traceable_evidence"] != false {
		t.Fatalf("operations pass proof without traceable evidence must not become ready: %+v", operationsItem)
	}

	authorizedProof := buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus: "authorized",
		Summary:     "protected path changes approved",
		EvidenceURI: "docs/development/protected-path-authorization.md",
	}))
	authorizedProof.EventID = 105
	audit = BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{ProtectedPathProof: &authorizedProof})
	authorizedItem := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if authorizedItem.Status == "complete" ||
		!containsString(authorizedItem.BlockedBy, "protected_path_proof_authorization_incomplete") ||
		authorizedItem.Metadata["authorized_proof_complete"] != false {
		t.Fatalf("authorized proof without structured metadata must not complete E9: %+v", authorizedItem)
	}
}

func TestBuildCompletionAuditBlocksDirtyProtectedPathProof(t *testing.T) {
	proof := ProtectedPathProof{
		Project:                         Record{ID: 1, Key: "areamatrix"},
		Status:                          "blocked",
		ProofStatus:                     "dirty",
		Decision:                        "blocked",
		AreaMatrixProtectedPathsTouched: true,
		GitStatusOutputHash:             "hash",
		GitStatusOutputLines:            1,
		Metadata:                        map[string]any{"evidence_uri": "local:protected-path-git-status"},
	}

	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		ProtectedPathProof: &proof,
	})

	if audit.ProtectedPathProofStatus != "blocked" {
		t.Fatalf("protected path aggregate status = %q, want blocked", audit.ProtectedPathProofStatus)
	}
	item := findCompletionAuditItem(t, audit, "E9_areamatrix_protected_path_proof")
	if item.Status != "blocked" || !containsString(item.BlockedBy, "protected_path_proof_not_clean") {
		t.Fatalf("dirty protected path proof should block E9: %+v", item)
	}
	if item.Metadata["git_status_output_lines"] != 1 {
		t.Fatalf("dirty proof metadata missing git status line count: %+v", item.Metadata)
	}
}

func TestBuildProtectedPathProofMarksDirtyOutput(t *testing.T) {
	_, err := protectedPathProofRequestHash(Record{ID: 1, Key: "areamatrix"}, RecordProtectedPathProofOptions{ProofStatus: "clean"})
	if err != nil {
		t.Fatalf("request hash failed: %v", err)
	}
	result := buildProtectedPathProof(Record{ID: 1, Key: "areamatrix"}, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus:     "dirty",
		GitStatusOutput: " M workflow/README.md",
	}))
	if result.Status != "blocked" || !result.AreaMatrixProtectedPathsTouched || result.GitStatusOutputLines != 1 {
		t.Fatalf("dirty proof should be blocked with line count: %+v", result)
	}
}

func assertCompletionAuditItem(t *testing.T, audit CompletionAudit, key string, status string) {
	t.Helper()
	item := findCompletionAuditItem(t, audit, key)
	if item.Status != status {
		t.Fatalf("completion audit item %s status = %q, want %q: %+v", key, item.Status, status, item)
	}
	if len(item.RequiredEvidence) == 0 {
		t.Fatalf("completion audit item %s missing required evidence: %+v", key, item)
	}
}

func findCompletionAuditItem(t *testing.T, audit CompletionAudit, key string) CompletionAuditItem {
	t.Helper()
	for _, item := range audit.Items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("completion audit item %s not found: %+v", key, audit.Items)
	return CompletionAuditItem{}
}

func real100BreakdownHasKey(items []Real100BreakdownItem, key string) bool {
	for _, item := range items {
		if item.Key == key {
			return true
		}
	}
	return false
}

func ptrSecurityReadiness(readiness SecurityBoundaryReadiness) *SecurityBoundaryReadiness {
	return &readiness
}
