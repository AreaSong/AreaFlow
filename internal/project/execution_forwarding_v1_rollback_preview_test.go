package project

import (
	"testing"
	"time"
)

func TestExecutionForwardingV1RollbackPreviewStaysReadOnlyAndBlocked(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromParts(
		record,
		ShimReadiness{Project: record, Status: "blocked"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
	)
	applyPreview := BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: time.Date(2026, 7, 3, 16, 0, 0, 0, time.UTC),
	})

	rollback := BuildExecutionForwardingV1RollbackPreview(applyPreview, ExecutionForwardingV1RollbackPreviewOptions{
		GeneratedAt: time.Date(2026, 7, 3, 16, 5, 0, 0, time.UTC),
	})

	if rollback.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", rollback.Status)
	}
	if rollback.Mode != "read_only_execution_forwarding_v1_rollback_preview" {
		t.Fatalf("mode = %q", rollback.Mode)
	}
	if rollback.RollbackTarget != "read_only_shim" {
		t.Fatalf("rollback target = %q", rollback.RollbackTarget)
	}
	if rollback.RollbackApplyOpen {
		t.Fatalf("rollback apply should stay closed")
	}
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:apply_preview", "pass")
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:fail_closed", "blocked")
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:proof_facts", "blocked")
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:reopen_conditions", "blocked")
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:read_only_preview", "pass")

	for _, key := range []string{
		"rollback_apply_open",
		"apply_open",
		"forwarding_v1_apply_open",
		"task_loop_run_forwarded",
		"legacy_task_loop_started",
		"legacy_progress_written",
		"legacy_logs_written",
		"legacy_checkpoint_written",
		"project_write_attempted",
		"execution_write_attempted",
		"area_flow_command_created",
		"area_flow_run_created",
		"worker_scheduled",
		"engine_call_attempted",
		"commands_run",
		"secrets_resolved",
		"network_used",
		"source_write_open",
		"generated_retained_write_open",
		"repair_apply_open",
		"checkpoint_apply_open",
		"publish_apply_open",
		"restore_apply_open",
	} {
		if rollback.SafetyFacts[key] {
			t.Fatalf("safety fact %s should stay false: %+v", key, rollback.SafetyFacts)
		}
	}
	if !rollback.SafetyFacts["read_only_preview"] {
		t.Fatalf("read_only_preview should be true")
	}
	if !containsString(rollback.RequiredProofFacts, "task_loop_run_forwarding_disabled") ||
		!containsString(rollback.RequiredProofFacts, "protected_path_proof_clean_after_rollback_recorded") {
		t.Fatalf("missing rollback proof facts: %+v", rollback.RequiredProofFacts)
	}
	if !containsString(rollback.FailClosedSteps, "keep ./task-loop run blocked or read-only according to current shim lifecycle") {
		t.Fatalf("missing fail-closed step: %+v", rollback.FailClosedSteps)
	}
	if !containsString(rollback.ForbiddenActions, "delete_forwarding_history") ||
		!containsString(rollback.ForbiddenActions, "restore_apply") {
		t.Fatalf("missing forbidden actions: %+v", rollback.ForbiddenActions)
	}
}

func TestExecutionForwardingV1RollbackPreviewPassesFailClosedWhenApplyIsAbsentAndLegacyProofIsClean(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromPartsWithProtectedPathProof(
		record,
		ShimReadiness{Project: record, Status: "blocked"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		func() ProtectedPathProof {
			proof := buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
				ProofStatus: "clean",
				Summary:     "legacy protected path proof reviewed",
				EvidenceURI: "local:protected-path-proof",
			}))
			proof.EventID = 88
			return proof
		}(),
	)
	applyPreview := BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: time.Date(2026, 7, 3, 16, 10, 0, 0, time.UTC),
	})

	rollback := BuildExecutionForwardingV1RollbackPreview(applyPreview, ExecutionForwardingV1RollbackPreviewOptions{
		GeneratedAt: time.Date(2026, 7, 3, 16, 15, 0, 0, time.UTC),
	})

	item := executionForwardingV1RollbackPreviewItem(rollback, "rollback_v1:fail_closed")
	if item.Status != "pass" {
		t.Fatalf("fail_closed status = %q, want pass: %+v", item.Status, item)
	}
	if item.Metadata["fail_closed_preview_proven"] != true ||
		item.Metadata["legacy_non_write_proof"] != "pass" ||
		item.Metadata["apply_open"] != false {
		t.Fatalf("fail_closed metadata missing proof facts: %+v", item.Metadata)
	}
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:proof_facts", "blocked")
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:reopen_conditions", "blocked")
	if rollback.RollbackApplyOpen {
		t.Fatalf("rollback apply should stay closed")
	}
}

func TestExecutionForwardingV1RollbackPreviewConsumesCompleteRollbackProofFacts(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromPartsWithProtectedPathProof(
		record,
		ShimReadiness{Project: record, Status: "blocked"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
			ProofStatus: "clean",
			Summary:     "legacy protected path proof reviewed",
			EvidenceURI: "local:protected-path-proof",
		})),
	)
	applyPreview := BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{})
	proof := buildExecutionCutoverProof(record, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts: append(
			append([]string{}, requiredExecutionCutoverProofFacts...),
			"rollback_target_read_only_shim_confirmed",
			"forwarding_v1_command_disabled_or_absent",
			"task_loop_run_forwarding_disabled",
			"legacy_task_loop_runner_not_started_after_rollback",
			"legacy_progress_json_not_written_after_rollback",
			"legacy_logs_not_written_after_rollback",
			"legacy_checkpoint_not_written_after_rollback",
			"areaflow_forwarded_state_preserved_as_audit_history",
			"protected_path_proof_clean_after_rollback_recorded",
		),
		Summary:     "rollback proof facts reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover-rollback"),
	})))
	proof.EventID = 91
	rollback := BuildExecutionForwardingV1RollbackPreviewWithProof(
		applyPreview,
		ExecutionForwardingV1RollbackPreviewOptions{},
		proof,
	)

	item := executionForwardingV1RollbackPreviewItem(rollback, "rollback_v1:proof_facts")
	if item.Status != "pass" {
		t.Fatalf("proof_facts status = %q, want pass: %+v", item.Status, item)
	}
	if item.Metadata["proof_event_id"] != int64(91) ||
		item.Metadata["proof_evidence_uri"] != e4ReleaseCandidateEvidenceURI("e4-execution-cutover-rollback") {
		t.Fatalf("proof metadata missing: %+v", item.Metadata)
	}
	missing, ok := item.Metadata["missing_proof_facts"].([]string)
	if !ok || len(missing) != 0 {
		t.Fatalf("expected no missing rollback facts: %+v", item.Metadata)
	}
	assertExecutionForwardingV1RollbackPreviewItem(t, rollback, "rollback_v1:reopen_conditions", "blocked")
	if rollback.Status != "blocked" {
		t.Fatalf("overall rollback preview should remain blocked by reopen conditions: %+v", rollback)
	}
}

func assertExecutionForwardingV1RollbackPreviewItem(t *testing.T, preview ExecutionForwardingV1RollbackPreview, key string, status string) {
	t.Helper()
	item := executionForwardingV1RollbackPreviewItem(preview, key)
	if item.Key == "" {
		t.Fatalf("missing rollback preview item %q", key)
	}
	if item.Status != status {
		t.Fatalf("item %s status = %q, want %q", key, item.Status, status)
	}
}

func executionForwardingV1RollbackPreviewItem(preview ExecutionForwardingV1RollbackPreview, key string) ExecutionForwardingV1RollbackPreviewItem {
	for _, item := range preview.Items {
		if item.Key == key {
			return item
		}
	}
	return ExecutionForwardingV1RollbackPreviewItem{}
}
