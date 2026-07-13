package project

import (
	"testing"
	"time"
)

func TestBuildExecutionForwardingV1ApplyPacketPreviewNeedsApproval(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()

	preview := BuildExecutionForwardingV1ApplyPacketPreview(applyPreview, ExecutionForwardingV1ApplyPacketPreviewOptions{})

	if preview.Status != "needs_approval" || preview.Decision != "needs_explicit_approval" {
		t.Fatalf("expected needs approval packet preview: %+v", preview)
	}
	if preview.Packet.CommandType != "project.execution_forwarding_v1.apply" || preview.Packet.ReadinessSnapshotHash == "" {
		t.Fatalf("unexpected packet facts: %+v", preview.Packet)
	}
	if preview.Gate.ApplyCommandEligible || preview.WouldCreateCommandRequestAfterApplyCommand || preview.WouldCreateRunAfterApplyCommand {
		t.Fatalf("packet without approval must not be command eligible: %+v", preview)
	}
	if preview.CommandRequestCreated || preview.AreaFlowRunCreated || preview.TaskLoopRunForwarded || preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.EngineCallAttempted {
		t.Fatalf("packet preview must remain read-only: %+v", preview)
	}
	if !containsString(preview.ApplyGateCommand, "--readiness-snapshot-hash") || !containsString(preview.ApplyGateCommand, preview.Packet.ReadinessSnapshotHash) {
		t.Fatalf("expected gate command to include readiness hash: %+v", preview.ApplyGateCommand)
	}
}

func TestBuildExecutionForwardingV1ApplyPacketPreviewReadyWithCompletePacket(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()

	preview := BuildExecutionForwardingV1ApplyPacketPreview(applyPreview, ExecutionForwardingV1ApplyPacketPreviewOptions{
		ExplicitApproval:           true,
		ApprovalID:                 "approval-1",
		ApprovalActor:              "as",
		ApprovalReason:             "approve forwarding v1 fixture gate",
		LegacyNonWriteProofID:      executionForwardingV1ExpectedLegacyNonWriteProofID(applyPreview.Readiness, applyPreview.Project.Key),
		RollbackPlanID:             executionForwardingV1ExpectedRollbackPlanID(applyPreview.Readiness, applyPreview.Project.Key),
		ProtectedPathFingerprintID: executionForwardingV1ExpectedProtectedPathFingerprintID(applyPreview.Readiness, applyPreview.Project.Key),
	})

	if preview.Status != "ready" || preview.Decision != "ready_for_future_apply_command" {
		t.Fatalf("expected ready packet preview: %+v", preview)
	}
	if !preview.Gate.ApplyCommandEligible || !preview.WouldCreateCommandRequestAfterApplyCommand || !preview.WouldCreateRunAfterApplyCommand || !preview.WouldCreateAuditEventAfterApplyCommand {
		t.Fatalf("expected complete packet to describe future command effects: %+v", preview)
	}
	if !preview.Packet.ExplicitApproval || preview.Packet.ApprovalID != "approval-1" || preview.Packet.LegacyNonWriteProofID == "" {
		t.Fatalf("unexpected approval/proof packet: %+v", preview.Packet)
	}
	if !containsString(preview.ApplyGateCommand, "--explicit-approval") || !containsString(preview.ApplyGateCommand, "--approval-id") {
		t.Fatalf("expected gate command to include approval flags: %+v", preview.ApplyGateCommand)
	}
	if preview.CommandRequestCreated || preview.AreaFlowRunCreated || preview.TaskLoopRunForwarded || preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.EngineCallAttempted {
		t.Fatalf("packet preview must remain read-only: %+v", preview)
	}
}

func TestBuildExecutionForwardingV1ApplyPacketPreviewBlocksReadiness(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	for index := range applyPreview.Readiness.Items {
		if applyPreview.Readiness.Items[index].Key == "read_only_shim" {
			applyPreview.Readiness.Items[index].Status = "blocked"
		}
	}

	preview := BuildExecutionForwardingV1ApplyPacketPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))

	if preview.Status != "blocked" || preview.Decision != "readiness_blocked" || preview.Gate.ApplyCommandEligible {
		t.Fatalf("expected readiness-blocked packet preview: %+v", preview)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(preview.Gate, "read_only_shim", "read_only_shim_not_pass") {
		t.Fatalf("expected read-only shim blocker: %+v", preview.Gate.Items)
	}
}

func TestBuildExecutionForwardingV1ApplyPacketPreviewBlocksOverallReadinessStatus(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	applyPreview.Readiness.Status = "blocked"

	preview := BuildExecutionForwardingV1ApplyPacketPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))

	if preview.Status != "blocked" || preview.Decision != "readiness_blocked" || preview.Gate.ApplyCommandEligible {
		t.Fatalf("expected overall readiness-blocked packet preview: %+v", preview)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(preview.Gate, "readiness_status", "execution_forwarding_v1_readiness_not_pass") {
		t.Fatalf("expected readiness status blocker: %+v", preview.Gate.Items)
	}
}

func TestBuildExecutionForwardingV1ApplyPacketPreviewBlocksExecutionBetaEvidence(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	executionForwardingV1ApplyGateSetReadinessItemStatus(t, &applyPreview, "read_only_verify_evidence", "blocked")

	preview := BuildExecutionForwardingV1ApplyPacketPreview(applyPreview, executionForwardingV1ApplyGateCompletePacketOptions(applyPreview, time.Time{}))

	if preview.Status != "blocked" || preview.Decision != "readiness_blocked" || preview.Gate.ApplyCommandEligible {
		t.Fatalf("expected execution beta evidence to block packet preview: %+v", preview)
	}
	if !executionForwardingV1ApplyGateHasBlockedItem(preview.Gate, "read_only_verify_evidence", "read_only_verify_evidence_not_pass") {
		t.Fatalf("expected read-only verify evidence blocker: %+v", preview.Gate.Items)
	}
}
