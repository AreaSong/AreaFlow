package project

import "testing"

func TestExecutionForwardingV1ReadinessBlocksUntilCommandAndProofExist(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromParts(
		record,
		ShimReadiness{Project: record, Status: "blocked"},
		map[string]int{
			"run.read_only_verify_queue":         1,
			"worker.read_only_verify":            1,
			"run.approved_artifact_write_queue":  1,
			"worker.approved_artifact_write":     1,
			"completion.validation_proof.record": 1,
		},
	)

	if readiness.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", readiness.Status)
	}
	assertExecutionForwardingV1Item(t, readiness, "allowed_task_scope", "pass")
	assertExecutionForwardingV1Item(t, readiness, "forbidden_high_risk_targets", "pass")
	assertExecutionForwardingV1Item(t, readiness, "read_only_verify_evidence", "pass")
	assertExecutionForwardingV1Item(t, readiness, "artifact_evidence", "pass")
	assertExecutionForwardingV1Item(t, readiness, "read_only_shim", "blocked")
	assertExecutionForwardingV1Item(t, readiness, "forwarding_command_api", "pass")
	assertExecutionForwardingV1Item(t, readiness, "legacy_non_write_proof", "blocked")
	assertExecutionForwardingV1Item(t, readiness, "rollback_to_read_only_shim", "blocked")

	if !containsString(readiness.AllowedTaskTypes, "read_only_verify") || containsString(readiness.AllowedTaskTypes, "source_write") {
		t.Fatalf("unexpected allowed task types: %+v", readiness.AllowedTaskTypes)
	}
	for _, key := range []string{
		"forwarding_v1_apply_open",
		"task_loop_run_forwarded",
		"legacy_task_loop_started",
		"project_write_attempted",
		"engine_call_attempted",
		"source_write_open",
		"generated_retained_write_open",
		"repair_apply_open",
		"checkpoint_apply_open",
		"publish_apply_open",
		"restore_apply_open",
	} {
		if readiness.SafetyFacts[key] {
			t.Fatalf("safety fact %s should stay false: %+v", key, readiness.SafetyFacts)
		}
	}
	if !containsString(readiness.ForbiddenActions, "engine_execution") || !containsString(readiness.ForbiddenActions, "restore_apply") {
		t.Fatalf("forbidden actions should include high-risk targets: %+v", readiness.ForbiddenActions)
	}
}

func TestExecutionForwardingV1ReadinessReportsMissingEvidence(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromParts(record, ShimReadiness{Project: record, Status: "pass"}, map[string]int{})

	item := executionForwardingV1Item(readiness, "read_only_verify_evidence")
	if item.Status != "blocked" {
		t.Fatalf("read_only_verify_evidence status = %q, want blocked", item.Status)
	}
	missing, ok := item.Metadata["missing_command_types"].([]string)
	if !ok || !containsString(missing, "run.read_only_verify_queue") || !containsString(missing, "worker.read_only_verify") {
		t.Fatalf("unexpected missing command metadata: %+v", item.Metadata)
	}
	if blockers := readiness.NextSteps[1].BlockedBy; !containsString(blockers, "read_only_verify_evidence_missing") ||
		!containsString(blockers, "artifact_evidence_missing") {
		t.Fatalf("expected command opening blockers to include execution beta evidence gaps: %+v", blockers)
	}
}

func TestExecutionForwardingV1ReadinessPromotesShimAfterSafeApplyRecord(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	shim := ShimReadiness{
		Project: record,
		Status:  "blocked",
		Items: []ShimReadinessItem{
			{Key: "compatibility_contract", Status: "pass"},
			{Key: "explicit_edit_approval", Status: "blocked"},
		},
	}
	event := EventRecord{
		ID:        1986,
		ProjectID: record.ID,
		Type:      shimApplyCommandEventType,
		Metadata: map[string]any{
			"status":                     "recorded",
			"decision":                   "allowed",
			"gate_status":                "pass",
			"gate_decision":              "go",
			"apply_command_eligible":     true,
			"command_request_created":    true,
			"project_write_attempted":    false,
			"execution_write_attempted":  false,
			"engine_call_attempted":      false,
			"task_loop_run_forwarded":    false,
			"status_projection_written":  false,
			"area_matrix_files_modified": false,
			"idempotency_key":            "package-b-closure",
		},
	}

	if !shimApplyEventCompletesForwardingReadiness(event) {
		t.Fatalf("safe shim apply event should complete forwarding shim readiness: %+v", event)
	}
	updated := executionForwardingShimReadinessAfterApply(shim, event)
	readiness := ExecutionForwardingV1ReadinessFromParts(updated.Project, updated, map[string]int{})

	assertExecutionForwardingV1Item(t, readiness, "read_only_shim", "pass")
	if updated.Status != "pass" {
		t.Fatalf("updated shim readiness status = %q, want pass", updated.Status)
	}
	approval := executionForwardingShimReadinessItem(updated, "explicit_edit_approval")
	if approval.Status != "pass" || approval.Metadata["shim_apply_event_id"] != int64(1986) {
		t.Fatalf("approval item was not bound to shim apply event: %+v", approval)
	}
}

func TestExecutionForwardingV1ReadinessRejectsUnsafeShimApplyRecord(t *testing.T) {
	event := EventRecord{
		ID:   1986,
		Type: shimApplyCommandEventType,
		Metadata: map[string]any{
			"status":                     "recorded",
			"decision":                   "allowed",
			"gate_status":                "pass",
			"gate_decision":              "go",
			"apply_command_eligible":     true,
			"command_request_created":    true,
			"area_matrix_files_modified": true,
		},
	}

	if shimApplyEventCompletesForwardingReadiness(event) {
		t.Fatalf("unsafe shim apply event must not complete forwarding readiness: %+v", event)
	}
}

func executionForwardingShimReadinessItem(readiness ShimReadiness, key string) ShimReadinessItem {
	for _, item := range readiness.Items {
		if item.Key == key {
			return item
		}
	}
	return ShimReadinessItem{}
}

func TestExecutionForwardingV1ReadinessConsumesCleanProtectedPathProof(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	proof := buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus: "clean",
		Summary:     "legacy protected path proof reviewed",
		EvidenceURI: "local:protected-path-proof",
	}))
	proof.EventID = 77
	proof.AuditEventID = 78
	readiness := ExecutionForwardingV1ReadinessFromPartsWithProtectedPathProof(
		record,
		ShimReadiness{Project: record, Status: "pass"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		proof,
	)

	item := executionForwardingV1Item(readiness, "legacy_non_write_proof")
	if item.Status != "pass" {
		t.Fatalf("legacy_non_write_proof status = %q, want pass: %+v", item.Status, item)
	}
	if item.Metadata["proof_event_id"] != int64(77) ||
		item.Metadata["proof_evidence_uri"] != "local:protected-path-proof" ||
		item.Metadata["areamatrix_protected_paths_touched"] != false ||
		item.Metadata["protected_path_proof_binding_status"] != "pass" ||
		item.Metadata["git_status_output_hash"] != protectedPathProofEmptyGitStatusOutputHash ||
		item.Metadata["git_status_output_lines"] != 0 ||
		item.Metadata["git_status_output_empty"] != true ||
		item.Metadata["protected_path_set_hash"] != protectedPathProofSetHash() ||
		item.Metadata["protected_path_set_count"] != int64(len(protectedPathProofSet())) {
		t.Fatalf("legacy proof metadata missing: %+v", item.Metadata)
	}
	if readiness.Status != "blocked" {
		t.Fatalf("overall status should still be blocked by rollback gap: %+v", readiness)
	}
}

func TestExecutionForwardingV1ReadinessConsumesCompleteRollbackProof(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	protectedProof := buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus: "clean",
		Summary:     "legacy protected path proof reviewed",
		EvidenceURI: "local:protected-path-proof",
	}))
	protectedProof.EventID = 77
	rollbackProof := executionForwardingV1CompleteRollbackProofFixture(record)

	readiness := ExecutionForwardingV1ReadinessFromPartsWithProofs(
		record,
		ShimReadiness{Project: record, Status: "blocked"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		protectedProof,
		rollbackProof,
	)

	assertExecutionForwardingV1Item(t, readiness, "read_only_shim", "blocked")
	assertExecutionForwardingV1Item(t, readiness, "legacy_non_write_proof", "pass")
	item := executionForwardingV1Item(readiness, "rollback_to_read_only_shim")
	if item.Status != "pass" {
		t.Fatalf("rollback_to_read_only_shim status = %q, want pass: %+v", item.Status, item)
	}
	if item.Metadata["proof_status"] != "complete" ||
		item.Metadata["proof_event_id"] != int64(91) ||
		item.Metadata["proof_evidence_uri"] != e4ReleaseCandidateEvidenceURI("e4-execution-cutover-rollback") ||
		item.Metadata["execution_cutover_scope_binding_status"] != "pass" ||
		item.Metadata["project_write_attempted"] != false ||
		item.Metadata["execution_write_attempted"] != false ||
		item.Metadata["task_loop_run_forwarded_by_command"] != false ||
		item.Metadata["areamatrix_protected_paths_touched"] != false {
		t.Fatalf("rollback proof metadata missing: %+v", item.Metadata)
	}
	missing, ok := item.Metadata["missing_proof_facts"].([]string)
	if !ok || len(missing) != 0 {
		t.Fatalf("expected no missing rollback facts: %+v", item.Metadata)
	}
	if readiness.Status != "blocked" {
		t.Fatalf("overall status should still be blocked by missing read-only shim: %+v", readiness)
	}
	if blockers := readiness.NextSteps[1].BlockedBy; containsString(blockers, "rollback_proof_missing") ||
		!containsString(blockers, "read_only_shim_missing") ||
		!containsString(blockers, "explicit_execution_cutover_approval_missing") {
		t.Fatalf("unexpected forwarding command blockers: %+v", blockers)
	}
}

func TestExecutionForwardingV1ReadinessSurfacesAuthorizedProtectedPathProof(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	gitStatusOutput := " M workflow/README.md"
	proof := buildProtectedPathProof(record, normalizeRecordProtectedPathProofOptions(RecordProtectedPathProofOptions{
		ProofStatus:                "authorized",
		Summary:                    "legacy protected path proof reviewed",
		EvidenceURI:                "local:protected-path-proof",
		GitStatusOutput:            gitStatusOutput,
		AuthorizedApprovalID:       "approval-123",
		AuthorizedAllowedPaths:     []string{"workflow/README.md"},
		AuthorizedDirtyOutputHash:  protectedPathProofOutputHash(gitStatusOutput),
		AuthorizedReviewer:         "release-owner",
		AuthorizedRollbackEvidence: "local:rollback-proof",
	}))
	proof.EventID = 88
	proof.AuditEventID = 89

	readiness := ExecutionForwardingV1ReadinessFromPartsWithProtectedPathProof(
		record,
		ShimReadiness{Project: record, Status: "pass"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		proof,
	)

	item := executionForwardingV1Item(readiness, "legacy_non_write_proof")
	if item.Status != "pass" ||
		item.Metadata["authorized_proof_complete"] != true ||
		item.Metadata["authorized_approval_id"] != "approval-123" ||
		item.Metadata["authorized_dirty_output_hash"] != protectedPathProofOutputHash(gitStatusOutput) ||
		item.Metadata["authorized_rollback_evidence_uri"] != "local:rollback-proof" ||
		item.Metadata["protected_path_proof_binding_status"] != "pass" ||
		item.Metadata["git_status_output_hash"] != protectedPathProofOutputHash(gitStatusOutput) ||
		item.Metadata["git_status_output_lines"] != 1 ||
		item.Metadata["git_status_output_empty"] != false ||
		item.Metadata["protected_path_set_hash"] != protectedPathProofSetHash() {
		t.Fatalf("authorized proof metadata missing: %+v", item)
	}
	touched, ok := item.Metadata["authorized_touched_paths"].([]string)
	if !ok || len(touched) != 1 || touched[0] != "workflow/README.md" {
		t.Fatalf("authorized touched paths missing: %+v", item.Metadata)
	}
}

func TestExecutionForwardingV1ReadinessBlocksDirtyProtectedPathProof(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromPartsWithProtectedPathProof(
		record,
		ShimReadiness{Project: record, Status: "pass"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		ProtectedPathProof{
			Project:                         record,
			Status:                          "blocked",
			ProofStatus:                     "dirty",
			Decision:                        "blocked",
			EventID:                         79,
			AreaMatrixProtectedPathsTouched: true,
		},
	)

	item := executionForwardingV1Item(readiness, "legacy_non_write_proof")
	if item.Status != "blocked" || item.Metadata["areamatrix_protected_paths_touched"] != true {
		t.Fatalf("dirty proof should keep legacy proof blocked: %+v", item)
	}
}

func assertExecutionForwardingV1Item(t *testing.T, readiness ExecutionForwardingV1Readiness, key string, status string) {
	t.Helper()
	item := executionForwardingV1Item(readiness, key)
	if item.Key == "" {
		t.Fatalf("missing execution forwarding v1 item %q", key)
	}
	if item.Status != status {
		t.Fatalf("item %s status = %q, want %q", key, item.Status, status)
	}
}

func executionForwardingV1Item(readiness ExecutionForwardingV1Readiness, key string) ExecutionForwardingV1ReadinessItem {
	for _, item := range readiness.Items {
		if item.Key == key {
			return item
		}
	}
	return ExecutionForwardingV1ReadinessItem{}
}

func executionForwardingV1CompleteRollbackProofFixture(record Record) ExecutionCutoverProof {
	proof := buildExecutionCutoverProof(record, normalizeRecordExecutionCutoverProofOptions(withExecutionCutoverEvidenceBinding(RecordExecutionCutoverProofOptions{
		ProofStatus: "complete",
		Facts: append(
			append([]string{}, requiredExecutionCutoverProofFacts...),
			executionForwardingV1RollbackProofFacts()...,
		),
		Summary:     "rollback proof facts reviewed",
		EvidenceURI: e4ReleaseCandidateEvidenceURI("e4-execution-cutover-rollback"),
	})))
	proof.EventID = 91
	proof.AuditEventID = 92
	return proof
}
