package project

import (
	"testing"
	"time"
)

func TestBuildExecutionForwardingV1ApplyGateBlocksMissingPacket(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, ExecutionForwardingV1ApplyGateOptions{})

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected blocked apply gate: %+v", gate)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "readiness_snapshot_hash", "readiness_snapshot_hash_missing_or_mismatch") {
		t.Fatalf("expected readiness hash blocker: %+v", gate.Items)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "explicit_approval", "explicit_execution_forwarding_v1_approval_missing") {
		t.Fatalf("expected explicit approval blocker: %+v", gate.Items)
	}
	if gate.CommandRequestCreated || gate.AreaFlowRunCreated || gate.TaskLoopRunForwarded || gate.ProjectWriteAttempted || gate.ExecutionWriteAttempted || gate.EngineCallAttempted {
		t.Fatalf("apply gate must remain read-only: %+v", gate)
	}
}

func TestBuildExecutionForwardingV1ApplyGatePassesCompletePacket(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC)
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, ExecutionForwardingV1ApplyPacketPreviewOptions{
		ExplicitApproval:           true,
		ApprovalID:                 "approval-1",
		ApprovalActor:              "as",
		ApprovalReason:             "approve forwarding v1 fixture gate",
		LegacyNonWriteProofID:      executionForwardingV1ExpectedLegacyNonWriteProofID(applyPreview.Readiness, applyPreview.Project.Key),
		RollbackPlanID:             executionForwardingV1ExpectedRollbackPlanID(applyPreview.Readiness, applyPreview.Project.Key),
		ProtectedPathFingerprintID: executionForwardingV1ExpectedProtectedPathFingerprintID(applyPreview.Readiness, applyPreview.Project.Key),
		GeneratedAt:                generatedAt,
	})

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, executionForwardingV1ApplyGateOptionsFromPacket(packet, generatedAt))

	if gate.Status != "pass" || gate.Decision != "go" || !gate.ApplyCommandEligible {
		t.Fatalf("expected passing apply gate: %+v", gate)
	}
	for _, item := range gate.Items {
		if item.Status != "pass" {
			t.Fatalf("expected all gate items pass, got %+v", item)
		}
	}
	if gate.SafetyFacts["command_request_created"] || gate.SafetyFacts["task_loop_run_forwarded"] || gate.SafetyFacts["project_write_attempted"] {
		t.Fatalf("unexpected safety facts: %+v", gate.SafetyFacts)
	}
}

func TestBuildExecutionForwardingV1ApplyGateBlocksOverallReadinessStatus(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	applyPreview.Readiness.Status = "blocked"
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, executionForwardingV1ApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected overall readiness status to block apply gate: %+v", gate)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "readiness_status", "execution_forwarding_v1_readiness_not_pass") {
		t.Fatalf("expected readiness status blocker: %+v", gate.Items)
	}
}

func TestBuildExecutionForwardingV1ApplyGateBlocksMissingExecutionBetaEvidence(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	applyPreview.Readiness.Status = "blocked"
	executionForwardingV1ApplyGateSetReadinessItemStatus(t, &applyPreview, "read_only_verify_evidence", "blocked")
	executionForwardingV1ApplyGateSetReadinessItemStatus(t, &applyPreview, "artifact_evidence", "blocked")
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, executionForwardingV1ApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected execution beta evidence to block apply gate: %+v", gate)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "read_only_verify_evidence", "read_only_verify_evidence_not_pass") {
		t.Fatalf("expected read-only verify evidence blocker: %+v", gate.Items)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "artifact_evidence", "artifact_evidence_not_pass") {
		t.Fatalf("expected artifact evidence blocker: %+v", gate.Items)
	}
}

func TestBuildExecutionForwardingV1ApplyGateBlocksStaleReadinessHash(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))
	options := executionForwardingV1ApplyGateOptionsFromPacket(packet, time.Time{})
	options.ReadinessSnapshotHash = "stale"

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, options)

	if gate.Status != "blocked" || gate.Decision != "no_go" {
		t.Fatalf("expected stale hash to block: %+v", gate)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "readiness_snapshot_hash", "readiness_snapshot_hash_missing_or_mismatch") {
		t.Fatalf("expected stale readiness hash blocker: %+v", gate.Items)
	}
}

func TestBuildExecutionForwardingV1ApplyGateBlocksMissingRollbackClosure(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	applyPreview.Readiness.Status = "blocked"
	for index := range applyPreview.Readiness.Items {
		if applyPreview.Readiness.Items[index].Key == "rollback_to_read_only_shim" {
			applyPreview.Readiness.Items[index].Status = "blocked"
		}
	}
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, executionForwardingV1ApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected rollback closure to block apply gate: %+v", gate)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(gate, "rollback_to_read_only_shim", "rollback_to_read_only_shim_not_pass") {
		t.Fatalf("expected rollback-to-read-only-shim blocker: %+v", gate.Items)
	}
}

func TestBuildExecutionForwardingV1ApplyGateBlocksUnscopedProofRefs(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	options := executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{})
	options.LegacyNonWriteProofID = "protected-path-proof-1"
	options.RollbackPlanID = "rollback-plan-1"
	options.ProtectedPathFingerprintID = "fingerprint-1"
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, options)

	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, executionForwardingV1ApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected unscoped proof refs to block apply gate: %+v", gate)
	}
	for _, key := range []string{"legacy_non_write_proof_id", "rollback_plan_id", "protected_path_fingerprint_id"} {
		if !executionForwardingV1ApplyGateHasBlockedItem(gate, key, key+"_missing_or_mismatch") {
			t.Fatalf("expected %s blocker: %+v", key, gate.Items)
		}
	}
}

func TestBuildExecutionForwardingV1ApplyGateBlocksStaleProofRefs(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		mutate  func(*ExecutionForwardingV1ApplyPacketPreviewOptions)
		itemKey string
	}{
		{
			name: "legacy proof event id",
			mutate: func(options *ExecutionForwardingV1ApplyPacketPreviewOptions) {
				options.LegacyNonWriteProofID = "areamatrix:legacy_non_write_proof:999"
			},
			itemKey: "legacy_non_write_proof_id",
		},
		{
			name: "rollback proof event id",
			mutate: func(options *ExecutionForwardingV1ApplyPacketPreviewOptions) {
				options.RollbackPlanID = "areamatrix:rollback_to_read_only_shim:999"
			},
			itemKey: "rollback_plan_id",
		},
		{
			name: "protected path fingerprint",
			mutate: func(options *ExecutionForwardingV1ApplyPacketPreviewOptions) {
				options.ProtectedPathFingerprintID = "areamatrix:protected_path_fingerprint:stale"
			},
			itemKey: "protected_path_fingerprint_id",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			applyPreview := executionForwardingV1ApplyGateFixturePreview()
			options := executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{})
			testCase.mutate(&options)
			packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, options)

			gate := BuildExecutionForwardingV1ApplyGate(applyPreview, executionForwardingV1ApplyGateOptionsFromPacket(packet, time.Time{}))

			if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
				t.Fatalf("expected stale proof ref to block apply gate: %+v", gate)
			}
			if !executionForwardingV1ApplyGateHasBlockedItem(gate, testCase.itemKey, testCase.itemKey+"_missing_or_mismatch") {
				t.Fatalf("expected stale proof ref blocker for %s: %+v", testCase.itemKey, gate.Items)
			}
		})
	}
}

func TestExecutionForwardingV1ReadinessSnapshotHashChangesWithProofIdentity(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	first := ExecutionForwardingV1ReadinessSnapshotHash(applyPreview)

	executionForwardingV1ApplyGateSetReadinessItemMetadata(t, &applyPreview, "legacy_non_write_proof", "proof_event_id", int64(202))
	if changed := ExecutionForwardingV1ReadinessSnapshotHash(applyPreview); changed == first {
		t.Fatalf("readiness snapshot hash should change with legacy proof event id")
	}

	applyPreview = executionForwardingV1ApplyGateFixturePreview()
	first = ExecutionForwardingV1ReadinessSnapshotHash(applyPreview)
	executionForwardingV1ApplyGateSetReadinessItemMetadata(t, &applyPreview, "legacy_non_write_proof", "protected_path_set_hash", "changed-protected-path-set-hash")
	if changed := ExecutionForwardingV1ReadinessSnapshotHash(applyPreview); changed == first {
		t.Fatalf("readiness snapshot hash should change with protected path fingerprint")
	}

	applyPreview = executionForwardingV1ApplyGateFixturePreview()
	first = ExecutionForwardingV1ReadinessSnapshotHash(applyPreview)
	executionForwardingV1ApplyGateSetReadinessItemMetadata(t, &applyPreview, "rollback_to_read_only_shim", "proof_event_id", int64(303))
	if changed := ExecutionForwardingV1ReadinessSnapshotHash(applyPreview); changed == first {
		t.Fatalf("readiness snapshot hash should change with rollback proof event id")
	}
}

func executionForwardingV1ApplyGateFixturePreview() ExecutionForwardingV1ApplyPreview {
	record := Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", RootPath: "/Users/as/Ai-Project/project/AreaMatrix"}
	readiness := ExecutionForwardingV1Readiness{
		Project:          record,
		Status:           "pass",
		Mode:             "read_only_execution_forwarding_v1_readiness",
		AllowedTaskTypes: append([]string{}, executionForwardingV1AllowedTaskTypes...),
		CommandEvidence: map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
		Capabilities:     []string{"read_project", "write_artifacts", "manage_workers"},
		ForbiddenActions: []string{"write_areamatrix_source", "engine_execution", "secret_resolve"},
		SafetyFacts:      map[string]bool{"read_only": true},
		GeneratedAt:      time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
	}
	for _, item := range []ExecutionForwardingV1ReadinessItem{
		{Key: "allowed_task_scope", Category: "scope", Status: "pass"},
		{Key: "forbidden_high_risk_targets", Category: "scope", Status: "pass"},
		{Key: "read_only_shim", Category: "compatibility", Status: "pass"},
		{Key: "read_only_verify_evidence", Category: "execution_beta", Status: "pass"},
		{Key: "artifact_evidence", Category: "execution_beta", Status: "pass"},
		{Key: "legacy_non_write_proof", Category: "protected_paths", Status: "pass", Metadata: map[string]any{
			"proof_event_id":          int64(101),
			"protected_path_set_hash": "fixture-protected-path-set-hash",
		}},
		{Key: "rollback_to_read_only_shim", Category: "rollback", Status: "pass", Metadata: map[string]any{
			"proof_event_id": int64(102),
		}},
	} {
		readiness.Items = append(readiness.Items, item)
	}
	return BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: readiness.GeneratedAt,
	})
}

func executionForwardingV1ApplyGateCompletePacketOptions(applyPreview ExecutionForwardingV1ApplyPreview, generatedAt time.Time) ExecutionForwardingV1ApplyPacketPreviewOptions {
	return ExecutionForwardingV1ApplyPacketPreviewOptions{
		ExplicitApproval:           true,
		ApprovalID:                 "approval-1",
		ApprovalActor:              "as",
		ApprovalReason:             "approve forwarding v1 fixture gate",
		LegacyNonWriteProofID:      executionForwardingV1ExpectedLegacyNonWriteProofID(applyPreview.Readiness, applyPreview.Project.Key),
		RollbackPlanID:             executionForwardingV1ExpectedRollbackPlanID(applyPreview.Readiness, applyPreview.Project.Key),
		ProtectedPathFingerprintID: executionForwardingV1ExpectedProtectedPathFingerprintID(applyPreview.Readiness, applyPreview.Project.Key),
		GeneratedAt:                generatedAt,
	}
}

func executionForwardingV1ApplyGateSetReadinessItemStatus(t *testing.T, applyPreview *ExecutionForwardingV1ApplyPreview, key string, status string) {
	t.Helper()
	for index := range applyPreview.Readiness.Items {
		if applyPreview.Readiness.Items[index].Key == key {
			applyPreview.Readiness.Items[index].Status = status
			return
		}
	}
	t.Fatalf("missing readiness item %q", key)
}

func executionForwardingV1ApplyGateSetReadinessItemMetadata(t *testing.T, applyPreview *ExecutionForwardingV1ApplyPreview, key string, metadataKey string, value any) {
	t.Helper()
	for index := range applyPreview.Readiness.Items {
		if applyPreview.Readiness.Items[index].Key == key {
			if applyPreview.Readiness.Items[index].Metadata == nil {
				applyPreview.Readiness.Items[index].Metadata = map[string]any{}
			}
			applyPreview.Readiness.Items[index].Metadata[metadataKey] = value
			return
		}
	}
	t.Fatalf("missing readiness item %q", key)
}

func executionForwardingV1ApplyGateHasBlockedItem(gate ExecutionForwardingV1ApplyGate, key string, blocker string) bool {
	for _, item := range gate.Items {
		if item.Key == key && item.Status == "blocked" && containsString(item.BlockedBy, blocker) {
			return true
		}
	}
	return false
}
