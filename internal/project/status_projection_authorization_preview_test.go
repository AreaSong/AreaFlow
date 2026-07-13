package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildStatusProjectionAuthorizationPreviewNeedsApproval(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", RootPath: "/Users/as/Ai-Project/project/AreaMatrix"}
	snapshot := Snapshot{
		Summary:    map[string]any{"summary_state": "mirroring"},
		SourceHash: "source-hash",
	}
	preimage := StatusProjectionPreimage{
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
	}
	permission := StatusProjectionAuthorizationPermission{
		Capability:        "write_status",
		ResourceType:      "path",
		TargetURI:         ".areaflow/status.json",
		CapabilityAllowed: true,
		PathAllowed:       true,
		Allowed:           true,
		Reason:            "allowed",
	}

	preview := BuildStatusProjectionAuthorizationPreview(record, snapshot, StatusProjectionAuthorizationPreviewOptions{
		TargetURI:   ".areaflow/status.json",
		GeneratedAt: generatedAt,
	}, preimage, permission)

	if preview.Status != "needs_approval" || preview.Decision != "needs_explicit_approval" || preview.ApplyOpen {
		t.Fatalf("unexpected preview decision: %+v", preview)
	}
	if preview.ClaimScope != statusProjectionClaimScope || !preview.NotReal100 {
		t.Fatalf("expected preflight-only non-real-100 scope: %+v", preview)
	}
	if preview.TargetURI != ".areaflow/status.json" || preview.TargetKind != "project_status_json" {
		t.Fatalf("unexpected target: %+v", preview)
	}
	if preview.SchemaURI != "schemas/status-projection.schema.json" {
		t.Fatalf("unexpected schema URI: %s", preview.SchemaURI)
	}
	if preview.RequiredAuthorizationPhrase != StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("expected Package A authorization phrase: %q", preview.RequiredAuthorizationPhrase)
	}
	if preview.ValidatorPreflight != "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json" {
		t.Fatalf("unexpected validator preflight: %s", preview.ValidatorPreflight)
	}
	if !containsString(preview.ProtectedPaths, ".areaflow/status.json") {
		t.Fatalf("expected protected status path: %+v", preview.ProtectedPaths)
	}
	if !containsString(preview.RequiredPacketFields, "expected_before_sha256") || !containsString(preview.RequiredPacketFields, "rollback_plan") {
		t.Fatalf("expected preimage and rollback fields: %+v", preview.RequiredPacketFields)
	}
	if !containsString(preview.BlockedBy, "explicit_status_projection_apply_approval_missing") {
		t.Fatalf("expected explicit approval blocker: %+v", preview.BlockedBy)
	}
	if !containsString(preview.BlockedBy, "current_target_schema_mismatch_requires_preimage_review") {
		t.Fatalf("expected legacy schema review blocker: %+v", preview.BlockedBy)
	}
	if len(preview.WriteSet) != 1 || preview.WriteSet[0].ExpectedBeforeSHA256 != "before-hash" || !preview.WriteSet[0].RequiresPreimageMatch {
		t.Fatalf("unexpected write set: %+v", preview.WriteSet)
	}
	if !preview.WouldWriteProjectFileAfterApproval || !preview.WouldCreateCommandRequestAfterApproval || !preview.WouldCreateAuditEventAfterApproval {
		t.Fatalf("expected after-approval write facts: %+v", preview)
	}
	if preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.EngineCallAttempted {
		t.Fatalf("preview must not attempt writes or engine calls: %+v", preview)
	}
	if preview.SafetyFacts["project_write_attempted"] || preview.SafetyFacts["execution_write_attempted"] || preview.SafetyFacts["engine_call_attempted"] {
		t.Fatalf("unexpected safety facts: %+v", preview.SafetyFacts)
	}
}

func TestBuildStatusProjectionAuthorizationPreviewBlocksDeniedPermission(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", RootPath: "/tmp/areamatrix"}
	snapshot := Snapshot{Summary: map[string]any{}, SourceHash: "source-hash"}
	preview := BuildStatusProjectionAuthorizationPreview(record, snapshot, StatusProjectionAuthorizationPreviewOptions{}, StatusProjectionPreimage{
		TargetPath:   "/tmp/areamatrix/.areaflow/status.json",
		SchemaStatus: "missing",
		Message:      "target does not exist",
	}, StatusProjectionAuthorizationPermission{
		Capability: "write_status",
		TargetURI:  ".areaflow/status.json",
		Reason:     "path not allowed",
	})

	if preview.Status != "blocked" || preview.Decision != "blocked" {
		t.Fatalf("expected blocked preview: %+v", preview)
	}
	if !containsString(preview.BlockedBy, "path not allowed") {
		t.Fatalf("expected permission blocker: %+v", preview.BlockedBy)
	}
	if preview.WouldWriteProjectFileAfterApproval || preview.WouldCreateCommandRequestAfterApproval {
		t.Fatalf("permission-blocked preview must not advertise apply side effects: %+v", preview)
	}
	if preview.RequiredAuthorizationPhrase != "" {
		t.Fatalf("fixture root must not require the Package A authorization phrase: %q", preview.RequiredAuthorizationPhrase)
	}
}

func TestInspectStatusProjectionPreimageDetectsLegacyShape(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(target, []byte(`{"version":1,"generated_at":"now","project":"AreaMatrix","source_hash":"old","summary":{},"compatibility":{"status":"blocked"}}`), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	preimage := inspectStatusProjectionPreimage(target, nil)
	if !preimage.Exists || !preimage.Readable || preimage.SHA256 == "" {
		t.Fatalf("expected readable preimage: %+v", preimage)
	}
	if preimage.SchemaStatus != "legacy" || !preimage.LegacyShape {
		t.Fatalf("expected legacy schema status: %+v", preimage)
	}
	if !containsString(preimage.MissingRequiredFields, "schema_version") || !containsString(preimage.MissingRequiredFields, "source_snapshot_hash") {
		t.Fatalf("expected stable required field gaps: %+v", preimage.MissingRequiredFields)
	}
	if !containsString(preimage.CompatibilityUnexpected, "status") {
		t.Fatalf("expected compatibility legacy field: %+v", preimage.CompatibilityUnexpected)
	}
}

func TestStatusProjectionTargetPathRejectsEscape(t *testing.T) {
	_, err := statusProjectionTargetPath(Record{RootPath: t.TempDir()}, "../outside/status.json")
	if err == nil {
		t.Fatalf("expected target escape to be rejected")
	}
}
