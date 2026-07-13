package project

import (
	"testing"
	"time"
)

func TestBuildShimApplyGatePassesCompletePacket(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(true)
	packet := shimApplyGateFixturePacket(authorization)

	gate := BuildShimApplyGate(authorization, shimApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "pass" || gate.Decision != "go" || !gate.ApplyCommandEligible {
		t.Fatalf("expected passing shim apply gate: %+v", gate)
	}
	if gate.CommandRequestCreated || gate.ProjectWriteAttempted || gate.ExecutionWriteAttempted || gate.TaskLoopRunForwarded || gate.StatusProjectionWritten || gate.AreaMatrixFilesModified || gate.EngineCallAttempted {
		t.Fatalf("shim apply gate must be read-only: %+v", gate)
	}
	if gate.SafetyFacts["command_request_created"] || gate.SafetyFacts["project_write_attempted"] || gate.SafetyFacts["area_matrix_files_modified"] {
		t.Fatalf("unexpected safety facts: %+v", gate.SafetyFacts)
	}
	for _, item := range gate.Items {
		if item.Status != "pass" {
			t.Fatalf("expected all gate items pass, got %+v", item)
		}
	}
}

func TestBuildShimApplyGateBlocksMissingPacket(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(true)

	gate := BuildShimApplyGate(authorization, ShimApplyGateOptions{})

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected blocked shim apply gate: %+v", gate)
	}
	if !shimApplyGateHasBlockedItem(gate, "authorization_snapshot_hash", "authorization_snapshot_hash_missing_or_mismatch") {
		t.Fatalf("expected authorization hash blocker: %+v", gate.Items)
	}
	if !shimApplyGateHasBlockedItem(gate, "explicit_approval", "explicit_shim_apply_approval_missing") {
		t.Fatalf("expected explicit approval blocker: %+v", gate.Items)
	}
	if !shimApplyGateHasBlockedItem(gate, "status_projection_packet_id", "status_projection_packet_id_missing_or_unscoped") {
		t.Fatalf("expected status projection packet proof blocker: %+v", gate.Items)
	}
}

func TestBuildShimApplyGateBlocksUnscopedProofRefs(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(true)
	packet := shimApplyGateFixturePacket(authorization)
	packet.StatusProjectionPacketID = "status-packet-1"

	gate := BuildShimApplyGate(authorization, shimApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected unscoped proof ref to block shim apply gate: %+v", gate)
	}
	if !shimApplyGateHasBlockedItem(gate, "status_projection_packet_id", "status_projection_packet_id_missing_or_unscoped") {
		t.Fatalf("expected scoped proof ref blocker: %+v", gate.Items)
	}
}

func TestBuildShimApplyGateBlocksReadinessEvidence(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(false)
	packet := shimApplyGateFixturePacket(authorization)

	gate := BuildShimApplyGate(authorization, shimApplyGateOptionsFromPacket(packet, time.Time{}))

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected readiness-blocked shim apply gate: %+v", gate)
	}
	if !shimApplyGateHasBlockedItem(gate, "readiness_blockers", "shim_readiness_still_blocked") {
		t.Fatalf("expected readiness blocker: %+v", gate.Items)
	}
}

func shimApplyGateFixtureAuthorization(withEvidence bool) ShimAuthorizationPacket {
	contract := CompatibilityContractFromSummary(ProjectSummary{
		Project: Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", RootPath: "/Users/as/Ai-Project/project/AreaMatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
	}, map[string]CommandPermission{})
	preview := ShimPreviewFromCompatibility(contract)
	evidence := map[string]EventRecord{}
	if withEvidence {
		evidence["real_areamatrix_readonly_smoke"] = shimApplyGateEvidenceEvent(11)
		evidence["real_areamatrix_status_projection_schema"] = shimApplyGateEvidenceEvent(12)
		evidence["areamatrix_dirty_worktree_review"] = shimApplyGateEvidenceEvent(13)
	}
	return ShimAuthorizationPacketFromReadiness(ShimReadinessFromPreviewWithEvidence(preview, evidence))
}

func shimApplyGateEvidenceEvent(id int64) EventRecord {
	return EventRecord{
		ID: id,
		Metadata: map[string]any{
			"evidence_status": "pass",
			"summary":         "fixture evidence passed",
			"evidence_uri":    "fixture://shim-evidence",
		},
	}
}

func shimApplyGateFixturePacket(authorization ShimAuthorizationPacket) ShimApplyPacket {
	return shimApplyPacketFromAuthorization(authorization, ShimApplyPacketPreviewOptions{
		ExplicitApproval:           true,
		ApprovalID:                 "approval-1",
		ApprovalActor:              "as",
		ApprovalReason:             "approve minimal AreaMatrix shim edit fixture",
		StatusProjectionPacketID:   "areamatrix:status_projection_apply_packet:status-packet-1",
		StatusProjectionGateID:     "areamatrix:status_projection_apply_gate:status-gate-1",
		ReadOnlySmokeEvidenceID:    "areamatrix:real_areamatrix_readonly_smoke:smoke-1",
		DirtyWorktreeReviewID:      "areamatrix:areamatrix_dirty_worktree_review:dirty-review-1",
		ProtectedPathFingerprintID: "areamatrix:protected_path_fingerprint:fingerprint-1",
		RollbackPlanID:             "areamatrix:rollback_plan:rollback-1",
	})
}

func shimApplyGateHasBlockedItem(gate ShimApplyGate, key string, blocker string) bool {
	for _, item := range gate.Items {
		if item.Key == key && item.Status == "blocked" && containsString(item.BlockedBy, blocker) {
			return true
		}
	}
	return false
}
