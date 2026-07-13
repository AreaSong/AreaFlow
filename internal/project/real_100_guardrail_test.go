package project

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReleasePreviewBuildersCarryReal100Guardrail(t *testing.T) {
	created := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	backup := BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		SchemaVersion: 1,
		ManifestHash:  "hash-a",
		Projects:      []BackupProjectManifest{{Project: record}},
		GeneratedAt:   created,
	}
	restore := RestorePlan{
		Status:        "ready",
		Mode:          "read_only_restore_plan",
		SchemaVersion: 1,
		ManifestHash:  "hash-a",
		Projects:      []Record{record},
		GeneratedAt:   created,
	}
	auditCoverage := AuditCoverage{
		Status:      "pass",
		Mode:        "read_only_audit_coverage",
		Scope:       "platform",
		GeneratedAt: created,
	}

	readiness := BuildReleaseReadiness(backup, restore, auditCoverage, nil, ReleaseReadinessOptions{GeneratedAt: created})
	remediation := BuildReleaseRemediationPlan(readiness, ReleaseRemediationOptions{GeneratedAt: created})
	acceptance := BuildReleaseAcceptancePreview(remediation, ReleaseAcceptancePreviewOptions{GeneratedAt: created})
	acceptanceGate := BuildReleaseAcceptanceGate(acceptance, ReleaseAcceptanceGateOptions{GeneratedAt: created})
	exceptionDoctor := BuildReleaseExceptionDoctor(acceptanceGate, ReleaseExceptionDoctorOptions{GeneratedAt: created})
	exceptionRecord := BuildReleaseExceptionRecordPreview(exceptionDoctor, ReleaseExceptionRecordPreviewOptions{GeneratedAt: created})
	exceptionSchema := BuildReleaseExceptionSchemaPreview(exceptionRecord, ReleaseExceptionSchemaPreviewOptions{GeneratedAt: created})
	exceptionMigrationGate := BuildReleaseExceptionMigrationApprovalGate(exceptionSchema, ReleaseExceptionMigrationApprovalGateOptions{GeneratedAt: created})
	exceptionApply := BuildReleaseExceptionApplyPreview(exceptionMigrationGate, ReleaseExceptionApplyPreviewOptions{GeneratedAt: created})
	finalGate := BuildReleaseFinalGate(readiness, acceptanceGate, exceptionApply, ReleaseFinalGateOptions{GeneratedAt: created})
	evidenceBundle := BuildReleaseEvidenceBundle(finalGate, backup, auditCoverage, ReleaseEvidenceBundleOptions{GeneratedAt: created})
	packagePreview := BuildReleasePackagePreview(evidenceBundle, ReleasePackagePreviewOptions{GeneratedAt: created})
	distribution := BuildReleaseDistributionPreview(packagePreview, ReleaseDistributionPreviewOptions{GeneratedAt: created})
	publishGate := BuildReleasePublishGate(distribution, ReleasePublishGateOptions{GeneratedAt: created})
	publishApproval := BuildReleasePublishApprovalPreview(publishGate, ReleasePublishApprovalPreviewOptions{GeneratedAt: created})
	rollout := BuildReleaseRolloutPlanPreview(publishApproval, ReleaseRolloutPlanPreviewOptions{GeneratedAt: created})

	for name, guardrail := range map[string]Real100Guardrail{
		"release_readiness":                         readiness.Real100Guardrail,
		"release_remediation":                       remediation.Real100Guardrail,
		"release_acceptance_preview":                acceptance.Real100Guardrail,
		"release_acceptance_gate":                   acceptanceGate.Real100Guardrail,
		"release_exception_doctor":                  exceptionDoctor.Real100Guardrail,
		"release_exception_record_preview":          exceptionRecord.Real100Guardrail,
		"release_exception_schema_preview":          exceptionSchema.Real100Guardrail,
		"release_exception_migration_approval_gate": exceptionMigrationGate.Real100Guardrail,
		"release_exception_apply_preview":           exceptionApply.Real100Guardrail,
		"release_final_gate":                        finalGate.Real100Guardrail,
		"release_evidence_bundle":                   evidenceBundle.Real100Guardrail,
		"release_package_preview":                   packagePreview.Real100Guardrail,
		"release_distribution_preview":              distribution.Real100Guardrail,
		"release_publish_gate":                      publishGate.Real100Guardrail,
		"release_publish_approval_preview":          publishApproval.Real100Guardrail,
		"release_rollout_plan_preview":              rollout.Real100Guardrail,
	} {
		assertReal100Guardrail(t, name, guardrail, ReleasePreviewReadinessScope, Real100ReleasePreviewBlockers())
		assertReal100BreakdownHasKey(t, name, guardrail.Real100Breakdown.NeedsExactAuthorization, "package_a_exact_authorization")
		assertReal100BreakdownHasKey(t, name, guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "real_areamatrix_execution_cutover")
		assertReal100BreakdownHasKey(t, name, guardrail.Real100Breakdown.AreaFlowOnlyCanContinue, "execution_forwarding_v1_readiness")
	}
}

func TestCompletionAuditBuildersCarryReal100Guardrail(t *testing.T) {
	record := Record{ID: 1, Key: completionAuditTargetProjectKey}
	audit := BuildCompletionAudit(CompletionAuditOptions{}, CompletionAuditParts{
		PackageAStatusProjection: stablePackageAStatusProjectionWithoutProvenanceBinding("source-hash-rc"),
	})
	assertReal100Guardrail(t, "completion_audit", audit.Real100Guardrail, CompletionAuditReadinessScope, Real100CompletionAuditBlockers())
	assertReal100BreakdownHasKey(t, "completion_audit", audit.Real100Guardrail.Real100Breakdown.NeedsExactAuthorization, "package_a_exact_authorization")
	assertReal100BreakdownHasKey(t, "completion_audit", audit.Real100Guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "package_a_status_projection_apply")
	assertReal100BreakdownHasKey(t, "completion_audit", audit.Real100Guardrail.Real100Breakdown.AreaFlowOnlyCanContinue, "E1_design_source_alignment")

	snapshot, err := buildCompletionAuditSnapshot(
		record,
		CompletionAudit{
			Real100Guardrail: CompletionAuditReal100Guardrail(),
			Status:           "complete",
			Mode:             "read_only_completion_audit",
			Scope:            "v1.0",
			SafetyFacts:      map[string]bool{"read_only": true},
		},
		normalizeRecordCompletionAuditSnapshotOptions(RecordCompletionAuditSnapshotOptions{
			EvidenceClass: completionAuditSnapshotEvidenceClassFixture,
		}),
		ReleaseEvidenceBundle{},
	)
	if err != nil {
		t.Fatalf("build completion audit snapshot: %v", err)
	}
	assertReal100Guardrail(t, "completion_audit_snapshot", snapshot.Real100Guardrail, CompletionAuditReadinessScope, Real100CompletionAuditBlockers())

	readiness := buildCompletionAuditSnapshotReadiness(record, snapshot, true, ReleaseEvidenceBundle{})
	assertReal100Guardrail(t, "completion_audit_snapshot_readiness", readiness.Real100Guardrail, CompletionAuditReadinessScope, Real100CompletionAuditBlockers())
}

func TestCompletionAuditReal100BreakdownClassifiesDynamicEvidence(t *testing.T) {
	items := []CompletionAuditItem{
		{
			Key:      "E1_design_source_alignment",
			Category: "design",
			Status:   "complete",
			Message:  "source proof accepted",
		},
		{
			Key:         "E3_command_api_smoke_evidence",
			Category:    "validation",
			Status:      "incomplete",
			Message:     "validation proof missing",
			BlockedBy:   []string{"fresh_validation_proof_missing"},
			NextCommand: "areaflow completion validation-proof record areamatrix --status complete --json",
		},
		{
			Key:      "E4_areamatrix_dogfood_completion",
			Category: "dogfood",
			Status:   "incomplete",
			Message:  "AreaMatrix dogfood still needs real cutover",
			BlockedBy: []string{
				"package_a_status_projection_apply_provenance_missing",
				"real_areamatrix_read_only_shim_not_landed",
				"execution_cutover_not_complete",
				"real_areamatrix_shim_retirement_not_proven",
			},
			Metadata: map[string]any{
				"archive_gate_passed":                  true,
				"latest_archive_proof_event_id":        int64(42),
				"latest_archive_proof_evidence_uri":    "docs/development/completion-audit-evidence.md#archive",
				"package_a_status_projection_ready":    false,
				"execution_cutover_gate_passed":        false,
				"shim_retirement_gate_passed":          false,
				"package_a_status_projection_blockers": []string{"package_a_status_projection_apply_provenance_missing"},
			},
			NextCommand: "areaflow completion execution-cutover-proof record areamatrix --status complete --json",
		},
	}

	guardrail := CompletionAuditReal100GuardrailForItems(items)
	assertReal100Guardrail(t, "dynamic_completion_audit", guardrail, CompletionAuditReadinessScope, []string{
		"package_a_status_projection_apply_provenance_missing",
		"real_areamatrix_read_only_shim_not_landed",
		"real_areamatrix_execution_cutover_not_proven",
		"real_areamatrix_shim_retirement_not_proven",
		"release_candidate_snapshot_not_ready",
	})
	assertReal100GuardrailOmitsBlocker(t, "dynamic_completion_audit", guardrail, "real_areamatrix_archive_not_proven")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsExactAuthorization, "package_a_exact_authorization")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "package_a_status_projection_apply")
	assertReal100BreakdownNextCommand(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsExactAuthorization, "package_a_exact_authorization", "status-projection-apply-packet")
	assertReal100BreakdownNextCommand(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "package_a_status_projection_apply", "status-projection-apply-gate")
	assertReal100BreakdownNextCommand(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "real_areamatrix_read_only_shim", "smoke-package-b-readiness")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "real_areamatrix_execution_cutover")
	assertReal100BreakdownNextCommand(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "real_areamatrix_execution_cutover", "execution-cutover-readiness")
	assertReal100BreakdownNextCommand(t, "dynamic_completion_audit", guardrail.Real100Breakdown.NeedsRealAreaMatrixWrite, "real_areamatrix_shim_retirement", "shim-retirement-proof")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.AreaFlowOnlyCanContinue, "E3_command_api_smoke_evidence")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.AreaFlowOnlyCanContinue, "release_candidate_snapshot_readiness")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.CompletedEvidence, "E1_design_source_alignment")
	assertReal100BreakdownHasKey(t, "dynamic_completion_audit", guardrail.Real100Breakdown.CompletedEvidence, "real_areamatrix_archive_proof")
}

func TestCompletionAuditReal100BlockersShrinkOnlyForCompletedCanonicalEvidence(t *testing.T) {
	items := []CompletionAuditItem{
		{
			Key:      "E4_areamatrix_dogfood_completion",
			Category: "dogfood",
			Status:   "complete",
			Message:  "AreaMatrix dogfood execution cutover, archive and shim retirement are proven",
			Metadata: map[string]any{
				"package_a_status_projection_ready":           true,
				"execution_cutover_gate_passed":               true,
				"archive_gate_passed":                         true,
				"shim_retirement_gate_passed":                 true,
				"latest_execution_cutover_proof_event_id":     int64(91),
				"latest_execution_cutover_proof_evidence_uri": "docs/development/completion-audit-evidence.md#execution-cutover",
				"latest_archive_proof_event_id":               int64(92),
				"latest_archive_proof_evidence_uri":           "docs/development/completion-audit-evidence.md#archive",
				"latest_shim_retirement_proof_event_id":       int64(93),
				"latest_shim_retirement_proof_evidence_uri":   "docs/development/completion-audit-evidence.md#shim-retirement",
			},
		},
	}

	guardrail := CompletionAuditReal100GuardrailForItems(items)

	assertReal100Guardrail(t, "completed_dogfood_still_needs_release_candidate", guardrail, CompletionAuditReadinessScope, []string{"release_candidate_snapshot_not_ready"})
	assertReal100BreakdownHasKey(t, "completed_dogfood_still_needs_release_candidate", guardrail.Real100Breakdown.CompletedEvidence, "E4_areamatrix_dogfood_completion")
	assertReal100BreakdownHasKey(t, "completed_dogfood_still_needs_release_candidate", guardrail.Real100Breakdown.AreaFlowOnlyCanContinue, "release_candidate_snapshot_readiness")
}

func TestNormalizeReal100GuardrailFallsBackAndCopiesBlockers(t *testing.T) {
	fallback := ReleasePreviewReal100Guardrail()
	guardrail := NormalizeReal100Guardrail(Real100Guardrail{}, fallback)
	assertReal100Guardrail(t, "fallback", guardrail, ReleasePreviewReadinessScope, Real100ReleasePreviewBlockers())

	guardrail.Real100Blockers[0] = "mutated"
	if reflect.DeepEqual(guardrail.Real100Blockers, fallback.Real100Blockers) {
		t.Fatalf("expected normalized blockers to be independent copy")
	}
	if fallback.Real100Blockers[0] != "package_a_status_projection_apply_provenance_missing" {
		t.Fatalf("fallback blockers mutated: %+v", fallback.Real100Blockers)
	}
	guardrail.Real100Breakdown.NeedsExactAuthorization[0].Key = "mutated"
	if fallback.Real100Breakdown.NeedsExactAuthorization[0].Key != "package_a_exact_authorization" {
		t.Fatalf("fallback breakdown mutated: %+v", fallback.Real100Breakdown)
	}
}

func assertReal100Guardrail(t *testing.T, name string, guardrail Real100Guardrail, wantScope string, wantBlockers []string) {
	t.Helper()
	if guardrail.ClaimScope != wantScope ||
		!guardrail.NotReal100 ||
		!guardrail.EvidenceOnly ||
		!guardrail.StatusAloneIsNotCompletion ||
		guardrail.ReleaseCandidateDecision == "" ||
		guardrail.ReadinessScope != wantScope ||
		guardrail.Real100Status != Real100StatusBlocked ||
		!reflect.DeepEqual(guardrail.Real100Blockers, wantBlockers) {
		t.Fatalf("%s real 100 guardrail = %+v, want status=%q scope=%q blockers=%v with disambiguation fields", name, guardrail, Real100StatusBlocked, wantScope, wantBlockers)
	}
	if wantScope == ReleasePreviewReadinessScope && guardrail.ReleaseCandidateDecision != "not_release_candidate_evidence" {
		t.Fatalf("%s release candidate decision = %q, want not_release_candidate_evidence", name, guardrail.ReleaseCandidateDecision)
	}
	if wantScope == CompletionAuditReadinessScope && guardrail.ReleaseCandidateDecision != "requires_release_candidate_snapshot" {
		t.Fatalf("%s release candidate decision = %q, want requires_release_candidate_snapshot", name, guardrail.ReleaseCandidateDecision)
	}
}

func assertReal100BreakdownHasKey(t *testing.T, name string, items []Real100BreakdownItem, key string) {
	t.Helper()
	for _, item := range items {
		if item.Key == key {
			return
		}
	}
	t.Fatalf("%s real 100 breakdown missing %q in %+v", name, key, items)
}

func assertReal100BreakdownNextCommand(t *testing.T, name string, items []Real100BreakdownItem, key string, wantFragment string) {
	t.Helper()
	for _, item := range items {
		if item.Key != key {
			continue
		}
		if !strings.Contains(item.NextCommand, wantFragment) {
			t.Fatalf("%s real 100 breakdown %q next_command = %q, want fragment %q", name, key, item.NextCommand, wantFragment)
		}
		return
	}
	t.Fatalf("%s real 100 breakdown missing %q in %+v", name, key, items)
}

func assertReal100GuardrailOmitsBlocker(t *testing.T, name string, guardrail Real100Guardrail, blocker string) {
	t.Helper()
	if containsString(guardrail.Real100Blockers, blocker) {
		t.Fatalf("%s real 100 blockers unexpectedly contain %q: %+v", name, blocker, guardrail.Real100Blockers)
	}
}
