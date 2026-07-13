package project

import (
	"testing"
	"time"
)

func TestBuildStatusProjectionApplyPacketPreviewNeedsApproval(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	authorization := statusProjectionApplyGateFixtureAuthorization()

	preview := BuildStatusProjectionApplyPacketPreview(authorization, StatusProjectionApplyPacketPreviewOptions{
		TargetURI:   ".areaflow/status.json",
		GeneratedAt: generatedAt,
	})

	if preview.Status != "needs_approval" || preview.Decision != "needs_explicit_approval" {
		t.Fatalf("expected needs approval packet preview: %+v", preview)
	}
	if preview.ClaimScope != statusProjectionClaimScope || !preview.NotReal100 || !preview.ApplyCommandEligibleIsNotApply || !preview.RequiresSeparateApplyCommand {
		t.Fatalf("expected top-level non-apply guardrails: %+v", preview)
	}
	if !containsString(preview.Blockers, "explicit_status_projection_apply_approval_missing") ||
		!containsString(preview.Blockers, "approval_actor_missing") ||
		!containsString(preview.Blockers, "approval_reason_missing_or_mismatch") {
		t.Fatalf("expected machine-readable top-level blockers: %+v", preview.Blockers)
	}
	if preview.RequiredAuthorizationPhrase != StatusProjectionApplyRequiredApprovalReason ||
		preview.Packet.RequiredAuthorizationPhrase != StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("expected machine-readable authorization phrase: %+v", preview)
	}
	if preview.Packet.SourceHash != "source-hash" || preview.Packet.ExpectedBeforeSHA256 != "before-hash" {
		t.Fatalf("unexpected packet facts: %+v", preview.Packet)
	}
	if preview.Packet.AcceptedPreimageSchemaStatus != "legacy" || preview.Packet.RollbackAction == "" {
		t.Fatalf("expected preimage schema and rollback facts: %+v", preview.Packet)
	}
	if preview.Packet.ProtectedPathFingerprintSHA256 != "protected-hash" || preview.APIRequest.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("expected protected path fingerprint in packet/API request: %+v", preview)
	}
	if preview.Gate.ApplyCommandEligible || preview.WouldCreateCommandRequestAfterApplyCommand || preview.WouldWriteProjectFileAfterApplyCommand {
		t.Fatalf("packet without approval must not be apply eligible: %+v", preview)
	}
	if preview.CommandRequestCreated || preview.StatusProjectionWritten || preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.EngineCallAttempted {
		t.Fatalf("packet preview must be read-only: %+v", preview)
	}
	if !containsString(preview.ApplyCommand, "--source-hash") || !containsString(preview.ApplyCommand, "source-hash") {
		t.Fatalf("expected apply command to include source hash: %+v", preview.ApplyCommand)
	}
}

func TestBuildStatusProjectionApplyPacketPreviewReadyWithApproval(t *testing.T) {
	authorization := statusProjectionApplyGateFixtureAuthorization()

	preview := BuildStatusProjectionApplyPacketPreview(authorization, StatusProjectionApplyPacketPreviewOptions{
		TargetURI:        ".areaflow/status.json",
		ExplicitApproval: true,
		ApprovalActor:    "as",
		ApprovalReason:   StatusProjectionApplyRequiredApprovalReason,
	})

	if preview.Status != "ready" || preview.Decision != "ready_for_apply_command" {
		t.Fatalf("expected ready packet preview: %+v", preview)
	}
	if preview.ClaimScope != statusProjectionClaimScope || !preview.NotReal100 || !preview.ApplyCommandEligibleIsNotApply || !preview.RequiresSeparateApplyCommand {
		t.Fatalf("expected ready packet to still expose non-apply guardrails: %+v", preview)
	}
	if len(preview.Blockers) != 0 {
		t.Fatalf("ready packet preview must not expose stale blockers: %+v", preview.Blockers)
	}
	if !preview.Gate.ApplyCommandEligible || !preview.WouldCreateCommandRequestAfterApplyCommand || !preview.WouldWriteProjectFileAfterApplyCommand {
		t.Fatalf("expected approved packet to be command eligible: %+v", preview)
	}
	if !preview.Packet.ExplicitApproval || preview.Packet.ApprovalActor != "as" || preview.Packet.ApprovalReason == "" {
		t.Fatalf("unexpected approval packet: %+v", preview.Packet)
	}
	if preview.RequiredAuthorizationPhrase != StatusProjectionApplyRequiredApprovalReason ||
		preview.Packet.RequiredAuthorizationPhrase != StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("expected required authorization phrase to remain visible when ready: %+v", preview)
	}
	if !containsString(preview.ApplyCommand, "--explicit-approval") || !containsString(preview.ApplyCommand, "--approval-actor") {
		t.Fatalf("expected apply command to include approval flags: %+v", preview.ApplyCommand)
	}
	if !preview.APIRequest.ExplicitApproval || preview.APIRequest.ApprovalActor != "as" {
		t.Fatalf("expected API request approval facts: %+v", preview.APIRequest)
	}
	if preview.APIRequest.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("expected API request protected path fingerprint: %+v", preview.APIRequest)
	}
	if preview.CommandRequestCreated || preview.StatusProjectionWritten || preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.EngineCallAttempted {
		t.Fatalf("packet preview must remain read-only: %+v", preview)
	}
}

func TestBuildStatusProjectionApplyPacketPreviewBlocksNonExactApprovalReason(t *testing.T) {
	authorization := statusProjectionApplyGateFixtureAuthorization()

	preview := BuildStatusProjectionApplyPacketPreview(authorization, StatusProjectionApplyPacketPreviewOptions{
		TargetURI:        ".areaflow/status.json",
		ExplicitApproval: true,
		ApprovalActor:    "as",
		ApprovalReason:   "approve fixture status projection apply",
	})

	if preview.Status == "ready" || preview.Decision == "ready_for_apply_command" {
		t.Fatalf("non-exact approval reason must not produce ready packet: %+v", preview)
	}
	if preview.Gate.ApplyCommandEligible || preview.WouldCreateCommandRequestAfterApplyCommand || preview.WouldWriteProjectFileAfterApplyCommand {
		t.Fatalf("non-exact approval reason must not be command eligible: %+v", preview)
	}
	if !statusProjectionGateHasBlockedItem(preview.Gate, "approval_reason", "approval_reason_missing_or_mismatch") {
		t.Fatalf("expected approval reason mismatch blocker: %+v", preview.Gate.Items)
	}
}
