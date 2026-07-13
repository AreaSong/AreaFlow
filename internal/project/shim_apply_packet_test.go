package project

import "testing"

func TestBuildShimApplyPacketPreviewNeedsApproval(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(true)

	preview := BuildShimApplyPacketPreview(authorization, ShimApplyPacketPreviewOptions{})

	if preview.Status != "needs_approval" || preview.Decision != "needs_explicit_approval" {
		t.Fatalf("expected needs approval packet preview: %+v", preview)
	}
	if preview.Packet.CommandType != "project.shim.apply" || preview.Packet.AuthorizationSnapshotHash == "" {
		t.Fatalf("unexpected packet facts: %+v", preview.Packet)
	}
	if preview.Gate.ApplyCommandEligible || preview.WouldCreateCommandRequestAfterApplyCommand || preview.WouldWriteAreaMatrixShimFilesAfterApplyCommand {
		t.Fatalf("packet without approval must not be command eligible: %+v", preview)
	}
	if preview.CommandRequestCreated || preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.TaskLoopRunForwarded || preview.StatusProjectionWritten || preview.AreaMatrixFilesModified || preview.EngineCallAttempted {
		t.Fatalf("packet preview must remain read-only: %+v", preview)
	}
	if !containsString(preview.ApplyGateCommand, "--authorization-snapshot-hash") || !containsString(preview.ApplyGateCommand, preview.Packet.AuthorizationSnapshotHash) {
		t.Fatalf("expected gate command to include authorization hash: %+v", preview.ApplyGateCommand)
	}
}

func TestBuildShimApplyPacketPreviewReadyWithCompletePacket(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(true)

	preview := BuildShimApplyPacketPreview(authorization, ShimApplyPacketPreviewOptions{
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

	if preview.Status != "ready" || preview.Decision != "ready_for_future_apply_command" {
		t.Fatalf("expected ready packet preview: %+v", preview)
	}
	if !preview.Gate.ApplyCommandEligible || !preview.WouldCreateCommandRequestAfterApplyCommand || !preview.WouldWriteAreaMatrixShimFilesAfterApplyCommand {
		t.Fatalf("expected complete packet to describe future command effects: %+v", preview)
	}
	if !preview.Packet.ExplicitApproval || preview.Packet.ApprovalID != "approval-1" || preview.Packet.StatusProjectionGateID == "" {
		t.Fatalf("unexpected approval/proof packet: %+v", preview.Packet)
	}
	if !containsString(preview.ApplyGateCommand, "--explicit-approval") || !containsString(preview.ApplyGateCommand, "--status-projection-gate-id") {
		t.Fatalf("expected gate command to include approval and proof flags: %+v", preview.ApplyGateCommand)
	}
	if preview.CommandRequestCreated || preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.TaskLoopRunForwarded || preview.StatusProjectionWritten || preview.AreaMatrixFilesModified || preview.EngineCallAttempted {
		t.Fatalf("packet preview must remain read-only: %+v", preview)
	}
}

func TestBuildShimApplyPacketPreviewBlocksReadiness(t *testing.T) {
	authorization := shimApplyGateFixtureAuthorization(false)

	preview := BuildShimApplyPacketPreview(authorization, ShimApplyPacketPreviewOptions{
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

	if preview.Status != "blocked" || preview.Decision != "readiness_blocked" || preview.Gate.ApplyCommandEligible {
		t.Fatalf("expected readiness-blocked packet preview: %+v", preview)
	}
	if !shimApplyGateHasBlockedItem(preview.Gate, "readiness_blockers", "shim_readiness_still_blocked") {
		t.Fatalf("expected readiness blocker: %+v", preview.Gate.Items)
	}
}
