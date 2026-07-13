package project

import (
	"testing"
	"time"
)

func TestBuildStatusProjectionApplyGatePassesCompletePacket(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	authorization := statusProjectionApplyGateFixtureAuthorization()
	expectedExists := true
	expectedSize := int64(99)

	gate := BuildStatusProjectionApplyGate(authorization, StatusProjectionApplyGateOptions{
		TargetURI:                      ".areaflow/status.json",
		ExpectedBeforeExists:           &expectedExists,
		ExpectedBeforeSHA256:           "before-hash",
		ExpectedBeforeSizeBytes:        &expectedSize,
		SourceHash:                     "source-hash",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		ProtectedPathCheck:             statusProjectionProtectedPathCheck(authorization.Project),
		ProtectedPathFingerprintSHA256: "protected-hash",
		RollbackAction:                 "restore the captured preimage bytes for .areaflow/status.json",
		AcceptedPreimageSchemaStatus:   "legacy",
		ExplicitApproval:               true,
		ApprovalActor:                  "as",
		ApprovalReason:                 StatusProjectionApplyRequiredApprovalReason,
		GeneratedAt:                    generatedAt,
	})

	if gate.Status != "pass" || gate.Decision != "go" || !gate.ApplyCommandEligible {
		t.Fatalf("expected passing apply gate: %+v", gate)
	}
	if gate.ClaimScope != statusProjectionClaimScope || !gate.NotReal100 || !gate.ApplyCommandEligibleIsNotApply || !gate.RequiresSeparateApplyCommand {
		t.Fatalf("expected top-level non-apply guardrails: %+v", gate)
	}
	if gate.CommandRequestCreated || gate.StatusProjectionWritten || gate.ProjectWriteAttempted || gate.ExecutionWriteAttempted || gate.EngineCallAttempted {
		t.Fatalf("apply gate must be read-only: %+v", gate)
	}
	if gate.SafetyFacts["command_request_created"] || gate.SafetyFacts["project_write_attempted"] || gate.SafetyFacts["execution_write_attempted"] {
		t.Fatalf("unexpected safety facts: %+v", gate.SafetyFacts)
	}
	for _, item := range gate.Items {
		if item.Status != "pass" {
			t.Fatalf("expected all gate items pass, got %+v", item)
		}
	}
}

func TestBuildStatusProjectionApplyGateBlocksNonExactApprovalReason(t *testing.T) {
	authorization := statusProjectionApplyGateFixtureAuthorization()
	expectedExists := true
	expectedSize := int64(99)

	gate := BuildStatusProjectionApplyGate(authorization, StatusProjectionApplyGateOptions{
		TargetURI:                      ".areaflow/status.json",
		ExpectedBeforeExists:           &expectedExists,
		ExpectedBeforeSHA256:           "before-hash",
		ExpectedBeforeSizeBytes:        &expectedSize,
		SourceHash:                     "source-hash",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             authorization.ValidatorPreflight,
		ProtectedPathCheck:             statusProjectionProtectedPathCheck(authorization.Project),
		ProtectedPathFingerprintSHA256: "protected-hash",
		RollbackAction:                 "restore the captured preimage bytes for .areaflow/status.json",
		AcceptedPreimageSchemaStatus:   "legacy",
		ExplicitApproval:               true,
		ApprovalActor:                  "as",
		ApprovalReason:                 "approve real status projection schema migration",
	})

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected non-exact approval reason to block: %+v", gate)
	}
	if !statusProjectionGateHasBlockedItem(gate, "approval_reason", "approval_reason_missing_or_mismatch") {
		t.Fatalf("expected approval reason mismatch blocker: %+v", gate.Items)
	}
}

func TestBuildStatusProjectionApplyGateAllowsNonExactApprovalReasonForFixtureRoot(t *testing.T) {
	authorization := statusProjectionApplyGateFixtureAuthorization()
	authorization.Project.RootPath = "/tmp/areamatrix-fixture"
	expectedExists := true
	expectedSize := int64(99)

	gate := BuildStatusProjectionApplyGate(authorization, StatusProjectionApplyGateOptions{
		TargetURI:                      ".areaflow/status.json",
		ExpectedBeforeExists:           &expectedExists,
		ExpectedBeforeSHA256:           "before-hash",
		ExpectedBeforeSizeBytes:        &expectedSize,
		SourceHash:                     "source-hash",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             authorization.ValidatorPreflight,
		ProtectedPathCheck:             statusProjectionProtectedPathCheck(authorization.Project),
		ProtectedPathFingerprintSHA256: "protected-hash",
		RollbackAction:                 "restore the captured preimage bytes for .areaflow/status.json",
		AcceptedPreimageSchemaStatus:   "legacy",
		ExplicitApproval:               true,
		ApprovalActor:                  "as",
		ApprovalReason:                 "approve fixture status projection apply",
	})

	if gate.Status != "pass" || gate.Decision != "go" || !gate.ApplyCommandEligible {
		t.Fatalf("expected fixture root to allow non-exact approval reason: %+v", gate)
	}
}

func TestBuildStatusProjectionApplyGateBlocksMissingPacket(t *testing.T) {
	authorization := statusProjectionApplyGateFixtureAuthorization()
	gate := BuildStatusProjectionApplyGate(authorization, StatusProjectionApplyGateOptions{
		TargetURI: ".areaflow/status.json",
	})

	if gate.Status != "blocked" || gate.Decision != "no_go" || gate.ApplyCommandEligible {
		t.Fatalf("expected blocked apply gate: %+v", gate)
	}
	if !statusProjectionGateHasBlockedItem(gate, "explicit_approval", "explicit_status_projection_apply_approval_missing") {
		t.Fatalf("expected explicit approval blocker: %+v", gate.Items)
	}
	if !statusProjectionGateHasBlockedItem(gate, "expected_before_sha256", "expected_before_sha256_missing_or_mismatch") {
		t.Fatalf("expected expected-before hash blocker: %+v", gate.Items)
	}
	if !statusProjectionGateHasBlockedItem(gate, "source_snapshot_hash", "source_snapshot_hash_missing_or_mismatch") {
		t.Fatalf("expected source hash blocker: %+v", gate.Items)
	}
	if !statusProjectionGateHasBlockedItem(gate, "protected_path_fingerprint_sha256", "protected_path_fingerprint_sha256_missing_or_mismatch") {
		t.Fatalf("expected protected path fingerprint blocker: %+v", gate.Items)
	}
}

func TestBuildStatusProjectionApplyGateBlocksStalePreimage(t *testing.T) {
	authorization := statusProjectionApplyGateFixtureAuthorization()
	expectedExists := true
	expectedSize := int64(100)

	gate := BuildStatusProjectionApplyGate(authorization, StatusProjectionApplyGateOptions{
		TargetURI:                      ".areaflow/status.json",
		ExpectedBeforeExists:           &expectedExists,
		ExpectedBeforeSHA256:           "stale-hash",
		ExpectedBeforeSizeBytes:        &expectedSize,
		SourceHash:                     "source-hash",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             authorization.ValidatorPreflight,
		ProtectedPathCheck:             statusProjectionProtectedPathCheck(authorization.Project),
		ProtectedPathFingerprintSHA256: "protected-hash",
		RollbackAction:                 "restore the captured preimage bytes for .areaflow/status.json",
		AcceptedPreimageSchemaStatus:   "legacy",
		ExplicitApproval:               true,
		ApprovalActor:                  "as",
		ApprovalReason:                 StatusProjectionApplyRequiredApprovalReason,
	})

	if gate.Status != "blocked" || gate.Decision != "no_go" {
		t.Fatalf("expected stale packet to block: %+v", gate)
	}
	if !statusProjectionGateHasBlockedItem(gate, "expected_before_sha256", "expected_before_sha256_missing_or_mismatch") {
		t.Fatalf("expected stale hash blocker: %+v", gate.Items)
	}
	if !statusProjectionGateHasBlockedItem(gate, "expected_before_size_bytes", "expected_before_size_bytes_missing_or_mismatch") {
		t.Fatalf("expected stale size blocker: %+v", gate.Items)
	}
}

func statusProjectionApplyGateFixtureAuthorization() StatusProjectionAuthorizationPreview {
	preview := BuildStatusProjectionAuthorizationPreview(
		Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", RootPath: "/Users/as/Ai-Project/project/AreaMatrix"},
		Snapshot{Summary: map[string]any{"summary_state": "mirroring"}, SourceHash: "source-hash"},
		StatusProjectionAuthorizationPreviewOptions{TargetURI: ".areaflow/status.json"},
		StatusProjectionPreimage{
			TargetPath:               "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			Exists:                   true,
			Readable:                 true,
			SizeBytes:                99,
			SHA256:                   "before-hash",
			SchemaStatus:             "legacy",
			LegacyShape:              true,
			MissingRequiredFields:    []string{"schema_version", "source_snapshot_hash"},
			UnexpectedTopLevelFields: []string{"summary", "version"},
			Message:                  "target uses legacy status projection shape",
		},
		StatusProjectionAuthorizationPermission{
			Capability:        "write_status",
			ResourceType:      "path",
			TargetURI:         ".areaflow/status.json",
			CapabilityAllowed: true,
			PathAllowed:       true,
			Allowed:           true,
			Reason:            "allowed",
		},
	)
	preview.ProtectedPathFingerprintSHA256 = "protected-hash"
	return preview
}

func statusProjectionGateHasBlockedItem(gate StatusProjectionApplyGate, key string, blocker string) bool {
	for _, item := range gate.Items {
		if item.Key == key && item.Status == "blocked" && containsString(item.BlockedBy, blocker) {
			return true
		}
	}
	return false
}
