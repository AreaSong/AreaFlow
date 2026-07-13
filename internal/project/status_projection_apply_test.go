package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStatusProjectionApplyDefaultsAndKey(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", RootPath: "/tmp/areamatrix"}
	snapshot := Snapshot{
		Summary:    map[string]any{"version_count": float64(2)},
		SourceHash: "hash-a",
	}
	options := normalizeApplyStatusProjectionOptions(ApplyStatusProjectionOptions{})
	if options.TargetURI != ".areaflow/status.json" || options.Actor != "local-user" {
		t.Fatalf("unexpected defaults: %+v", options)
	}
	key := statusProjectionApplyIdempotencyKey(record, options, snapshot, []byte(`{"version_count":2}`))
	rootHash := shortSHA256Hex([]byte("/tmp/areamatrix"))
	if !strings.HasPrefix(key, "project.status_projection.apply:areamatrix:.areaflow/status.json:"+rootHash+":hash-a:") {
		t.Fatalf("unexpected status projection apply key: %s", key)
	}
	gatedOptions := options
	gatedOptions.Gate.SourceHash = "hash-a"
	gatedOptions.Gate.SchemaURI = "schemas/status-projection.schema.json"
	gatedOptions.Gate.ExplicitApproval = true
	gatedOptions.Gate.ApprovalActor = "as"
	gatedOptions.Gate.ApprovalReason = "approve status projection apply"
	gatedKey := statusProjectionApplyIdempotencyKey(record, gatedOptions, snapshot, []byte(`{"version_count":2}`))
	if key == gatedKey {
		t.Fatalf("expected gate packet to affect idempotency key: key=%s", key)
	}
	requestHash, err := statusProjectionApplyRequestHash(record, options, snapshot, []byte(`{"version_count":2}`))
	if err != nil {
		t.Fatalf("request hash failed: %v", err)
	}
	changed, err := statusProjectionApplyRequestHash(record, ApplyStatusProjectionOptions{
		TargetURI: ".areaflow/status.json",
		Actor:     "local-user",
		Reason:    "different",
	}, snapshot, []byte(`{"version_count":2}`))
	if err != nil {
		t.Fatalf("changed request hash failed: %v", err)
	}
	if requestHash == changed {
		t.Fatalf("request hash should include reason")
	}
}

func TestStatusProjectionTargetKind(t *testing.T) {
	if got := statusProjectionTargetKind(".areaflow/status.json"); got != "project_status_json" {
		t.Fatalf("status json target kind = %q", got)
	}
	if got := statusProjectionTargetKind("workflow/README.md"); got != "workflow_readme" {
		t.Fatalf("workflow readme target kind = %q", got)
	}
	if got := statusProjectionTargetKind("other"); got != "unknown" {
		t.Fatalf("unknown target kind = %q", got)
	}
}

func TestStatusProjectionApplyCommandResponseSafetyFacts(t *testing.T) {
	generatedAt := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	result := ApplyStatusProjectionResult{
		Project:                   Record{ID: 1, Key: "areamatrix"},
		Status:                    "written",
		Decision:                  "allowed",
		Message:                   "status projection written",
		EventID:                   11,
		AuditEventID:              12,
		SnapshotID:                13,
		StatusProjectionID:        14,
		TargetKind:                "project_status_json",
		TargetURI:                 ".areaflow/status.json",
		WrittenTarget:             "/tmp/areamatrix/.areaflow/status.json",
		WriteHash:                 "hash-a",
		WriteSize:                 100,
		PreimageCaptured:          true,
		PreimageExists:            true,
		PreimageSHA256:            "before-hash",
		PreimageSize:              42,
		PostWriteVerified:         true,
		PostWriteSHA256:           "hash-a",
		PostWriteSize:             100,
		ProtectedPathsVerified:    true,
		ProtectedPathBeforeHash:   "protected-hash",
		ProtectedPathAfterHash:    "protected-hash",
		ExpectedProtectedPathHash: "protected-hash",
		RootContained:             true,
		StableProjectionValid:     true,
		AtomicReplaceUsed:         true,
		RollbackCompensation:      true,
		SourceHash:                "source-a",
		SummaryState:              "mirroring",
		ApplyGateStatus:           "pass",
		ApplyGateDecision:         "go",
		ApplyGateApprovalStatus:   "approved",
		ApplyCommandEligible:      true,
		IdempotencyKey:            "projection-key",
		GeneratedAt:               generatedAt,
		ProjectWriteAttempted:     true,
		ExecutionWriteAttempted:   false,
		EngineCallAttempted:       false,
	}
	response := statusProjectionApplyCommandResponse(result)
	if !metadataBool(response, "project_write_attempted") {
		t.Fatalf("expected project write attempted in response: %+v", response)
	}
	if metadataBool(response, "execution_write_attempted") || metadataBool(response, "engine_call_attempted") {
		t.Fatalf("unexpected execution/engine safety facts: %+v", response)
	}
	if metadataInt64(response, "status_projection_id") != 14 || metadataString(response, "target_uri") != ".areaflow/status.json" {
		t.Fatalf("unexpected projection response identity: %+v", response)
	}
	if !metadataBool(response, "preimage_captured") || !metadataBool(response, "preimage_exists") || metadataString(response, "preimage_sha256") != "before-hash" || metadataInt64(response, "preimage_size") != 42 {
		t.Fatalf("unexpected preimage facts: %+v", response)
	}
	if !metadataBool(response, "post_write_verified") || metadataString(response, "post_write_sha256") != "hash-a" || metadataInt64(response, "post_write_size") != 100 {
		t.Fatalf("unexpected post-write facts: %+v", response)
	}
	if !metadataBool(response, "protected_paths_verified") || metadataString(response, "protected_path_before_hash") != "protected-hash" || metadataString(response, "protected_path_after_hash") != "protected-hash" || metadataString(response, "expected_protected_path_hash") != "protected-hash" {
		t.Fatalf("unexpected protected path facts: %+v", response)
	}
	if !metadataBool(response, "root_contained") || !metadataBool(response, "stable_projection_validated") || !metadataBool(response, "atomic_replace_used") || !metadataBool(response, "rollback_compensation_enabled") {
		t.Fatalf("unexpected write safety facts: %+v", response)
	}
	if metadataString(response, "apply_gate_status") != "pass" || metadataString(response, "apply_gate_decision") != "go" || !metadataBool(response, "apply_command_eligible") {
		t.Fatalf("unexpected apply gate response facts: %+v", response)
	}
}

func TestStatusProjectionApplyGateBlockers(t *testing.T) {
	gate := StatusProjectionApplyGate{
		Status:               "blocked",
		Decision:             "no_go",
		ApplyCommandEligible: false,
		Items: []StatusProjectionApplyGateItem{
			{Key: "source_snapshot_hash", Status: "blocked", BlockedBy: []string{"source_snapshot_hash_missing_or_mismatch"}},
			{Key: "explicit_approval", Status: "blocked", BlockedBy: []string{"explicit_status_projection_apply_approval_missing"}},
			{Key: "schema_uri", Status: "pass"},
		},
	}
	blockers := statusProjectionApplyGateBlockers(gate)
	for _, expected := range []string{"status_projection_apply_gate_blocked", "source_snapshot_hash_missing_or_mismatch", "explicit_status_projection_apply_approval_missing"} {
		if !containsString(blockers, expected) {
			t.Fatalf("expected blocker %q in %+v", expected, blockers)
		}
	}
}

func TestRollbackStatusProjectionApplyFileRestoresPreimage(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	before := []byte(`{"version":1}`)
	after := []byte(`{"schema_version":1}`)
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	preimage, err := captureStatusProjectionApplyPreimage(Record{Key: "demo", RootPath: root}, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("capture preimage: %v", err)
	}
	if !preimage.Exists || preimage.SHA256 != sha256Hex(before) {
		t.Fatalf("unexpected preimage: %+v", preimage)
	}
	if err := os.WriteFile(target, after, 0o644); err != nil {
		t.Fatalf("write after: %v", err)
	}

	if err := rollbackStatusProjectionApplyFile(preimage, sha256Hex(after)); err != nil {
		t.Fatalf("rollback status projection: %v", err)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if string(current) != string(before) {
		t.Fatalf("rollback restored %s, want %s", current, before)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".status.json.rollback-*"))
	if err != nil {
		t.Fatalf("glob rollback temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("rollback left temp files: %+v", matches)
	}
}

func TestVerifyStatusProjectionApplyWrittenFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	content := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	verification, err := verifyStatusProjectionApplyWrittenFile(StatusProjectionWriteResult{
		Target: target,
		Hash:   sha256Hex(content),
		Size:   int64(len(content)),
	}, target, "source-a")
	if err != nil {
		t.Fatalf("verify written file: %v", err)
	}
	if !verification.Verified || verification.SHA256 != sha256Hex(content) || verification.Size != int64(len(content)) || !verification.RootContained || !verification.StableProjectionValidated {
		t.Fatalf("unexpected verification result: %+v", verification)
	}
}

func TestVerifyStatusProjectionApplyWrittenFileReportsActualHashForRollback(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	content := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	verification, err := verifyStatusProjectionApplyWrittenFile(StatusProjectionWriteResult{
		Target: target,
		Hash:   "reported-hash",
		Size:   int64(len(content)),
	}, target, "source-a")
	if err == nil {
		t.Fatalf("expected hash mismatch")
	}
	if verification.Verified || verification.SHA256 != sha256Hex(content) || verification.Size != int64(len(content)) {
		t.Fatalf("expected actual file facts for rollback: %+v", verification)
	}
}

func TestVerifyStatusProjectionApplyWrittenFileRejectsUnstableProjection(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	content := []byte(`{"schema_version":1}`)
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	verification, err := verifyStatusProjectionApplyWrittenFile(StatusProjectionWriteResult{
		Target: target,
		Hash:   sha256Hex(content),
		Size:   int64(len(content)),
	}, target, "source-a")
	if err == nil {
		t.Fatalf("expected unstable projection to be rejected")
	}
	if verification.Verified || verification.StableProjectionValidated {
		t.Fatalf("unstable projection should not verify: %+v", verification)
	}
}

func TestVerifyStatusProjectionApplyWrittenFileRejectsSourceHashMismatch(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	content := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	verification, err := verifyStatusProjectionApplyWrittenFile(StatusProjectionWriteResult{
		Target: target,
		Hash:   sha256Hex(content),
		Size:   int64(len(content)),
	}, target, "source-b")
	if err == nil {
		t.Fatalf("expected source hash mismatch")
	}
	if verification.Verified || verification.StableProjectionValidated {
		t.Fatalf("source hash mismatch should not verify: %+v", verification)
	}
}

func TestRunStatusProjectionApplyWriterRollsBackPostWriteHashMismatch(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	before := []byte(`{"version":1}`)
	after := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	record := Record{ID: 1, Key: "demo", RootPath: root}
	snapshot := Snapshot{SourceHash: "source-a"}
	result := ApplyStatusProjectionResult{
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetURI:               ".areaflow/status.json",
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	writer := func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
		if err := os.WriteFile(target, after, 0o644); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		return StatusProjectionWriteResult{
			Target:                    target,
			Hash:                      "reported-hash",
			Size:                      int64(len(after)),
			RootContained:             true,
			StableProjectionValidated: true,
			AtomicReplaceUsed:         true,
		}, nil
	}

	gateOptions := statusProjectionApplyTestGateOptionsForRecord(t, record, true, sha256Hex(before), int64(len(before)))
	preimage, wroteProjectFile, err := runStatusProjectionApplyWriter(context.Background(), record, snapshot, ".areaflow/status.json", gateOptions, writer, &result)
	if err != nil {
		t.Fatalf("apply writer returned unexpected error: %v", err)
	}
	if wroteProjectFile {
		t.Fatalf("writer flow should not report durable project write after rollback")
	}
	if !preimage.Exists || preimage.SHA256 != sha256Hex(before) {
		t.Fatalf("unexpected preimage: %+v", preimage)
	}
	if result.Decision != "denied" || result.Status != "blocked" || result.Message != "status projection post-write verification failed" {
		t.Fatalf("unexpected blocked result: %+v", result)
	}
	if result.PostWriteVerified || result.PostWriteSHA256 != sha256Hex(after) || result.PostWriteSize != int64(len(after)) {
		t.Fatalf("unexpected post-write facts: %+v", result)
	}
	if !containsString(result.Blockers, "status projection post-write hash mismatch: actual "+sha256Hex(after)+" != reported reported-hash") {
		t.Fatalf("expected hash mismatch blocker: %+v", result.Blockers)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if string(current) != string(before) {
		t.Fatalf("post-write mismatch rollback restored %s, want %s", current, before)
	}
}

func TestRunStatusProjectionApplyWriterBlocksPreimageDriftBeforeWrite(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	staleBefore := []byte(`{"version":0}`)
	before := []byte(`{"version":1}`)
	after := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	record := Record{ID: 1, Key: "demo", RootPath: root}
	snapshot := Snapshot{SourceHash: "source-a"}
	result := ApplyStatusProjectionResult{
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetURI:               ".areaflow/status.json",
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	writerCalled := false
	writer := func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
		writerCalled = true
		if err := os.WriteFile(target, after, 0o644); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		return StatusProjectionWriteResult{
			Target:                    target,
			Hash:                      sha256Hex(after),
			Size:                      int64(len(after)),
			RootContained:             true,
			StableProjectionValidated: true,
			AtomicReplaceUsed:         true,
		}, nil
	}

	gateOptions := statusProjectionApplyTestGateOptions(true, sha256Hex(staleBefore), int64(len(staleBefore)))
	preimage, wroteProjectFile, err := runStatusProjectionApplyWriter(context.Background(), record, snapshot, ".areaflow/status.json", gateOptions, writer, &result)
	if err != nil {
		t.Fatalf("apply writer returned unexpected error: %v", err)
	}
	if writerCalled {
		t.Fatalf("writer must not be called after write-time preimage drift")
	}
	if wroteProjectFile {
		t.Fatalf("writer flow should not report project write after preimage drift")
	}
	if !preimage.Exists || preimage.SHA256 != sha256Hex(before) || preimage.SizeBytes != int64(len(before)) {
		t.Fatalf("unexpected current preimage: %+v", preimage)
	}
	if result.Decision != "denied" || result.Status != "blocked" || result.Message != "status projection preimage changed before write" {
		t.Fatalf("unexpected drift result: %+v", result)
	}
	if !result.PreimageCaptured || !result.PreimageExists || result.PreimageSHA256 != sha256Hex(before) || result.PreimageSize != int64(len(before)) {
		t.Fatalf("unexpected captured preimage facts: %+v", result)
	}
	if result.ProjectWriteAttempted {
		t.Fatalf("project write should not be marked attempted when writer is not called")
	}
	if !containsString(result.Blockers, "expected_before_sha256_changed_before_write") {
		t.Fatalf("expected sha drift blocker: %+v", result.Blockers)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if string(current) != string(before) {
		t.Fatalf("preimage drift path should leave target unchanged: %s", current)
	}
}

func TestRunStatusProjectionApplyWriterRollsBackPartialWriterError(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	before := []byte(`{"version":1}`)
	after := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	record := Record{ID: 1, Key: "demo", RootPath: root}
	snapshot := Snapshot{SourceHash: "source-a"}
	result := ApplyStatusProjectionResult{
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetURI:               ".areaflow/status.json",
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	writerErr := errors.New("simulated writer failure")
	writer := func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
		if err := os.WriteFile(target, after, 0o644); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		return StatusProjectionWriteResult{}, writerErr
	}

	gateOptions := statusProjectionApplyTestGateOptionsForRecord(t, record, true, sha256Hex(before), int64(len(before)))
	preimage, wroteProjectFile, err := runStatusProjectionApplyWriter(context.Background(), record, snapshot, ".areaflow/status.json", gateOptions, writer, &result)
	if err != nil {
		t.Fatalf("apply writer returned unexpected error: %v", err)
	}
	if wroteProjectFile {
		t.Fatalf("writer flow should not report durable project write after rollback")
	}
	if !preimage.Exists || preimage.SHA256 != sha256Hex(before) {
		t.Fatalf("unexpected preimage: %+v", preimage)
	}
	if result.Decision != "denied" || result.Status != "blocked" || result.Message != "status projection write failed" {
		t.Fatalf("unexpected blocked result: %+v", result)
	}
	if result.PostWriteVerified || result.PostWriteSHA256 != sha256Hex(after) || result.PostWriteSize != int64(len(after)) {
		t.Fatalf("unexpected partial post-write facts: %+v", result)
	}
	if !result.RollbackCompensation {
		t.Fatalf("expected rollback compensation after partial writer error")
	}
	if !containsString(result.Blockers, writerErr.Error()) {
		t.Fatalf("expected writer error blocker: %+v", result.Blockers)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if string(current) != string(before) {
		t.Fatalf("partial writer error rollback restored %s, want %s", current, before)
	}
}

func TestRunStatusProjectionApplyWriterRemovesCreatedTargetOnWriterError(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	created := stableStatusProjectionTestJSON("source-a")
	record := Record{ID: 1, Key: "demo", RootPath: root}
	snapshot := Snapshot{SourceHash: "source-a"}
	result := ApplyStatusProjectionResult{
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetURI:               ".areaflow/status.json",
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	writerErr := errors.New("simulated create failure")
	writer := func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
		if err := os.WriteFile(target, created, 0o644); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		return StatusProjectionWriteResult{}, writerErr
	}

	gateOptions := statusProjectionApplyTestGateOptionsForRecord(t, record, false, "", 0)
	preimage, wroteProjectFile, err := runStatusProjectionApplyWriter(context.Background(), record, snapshot, ".areaflow/status.json", gateOptions, writer, &result)
	if err != nil {
		t.Fatalf("apply writer returned unexpected error: %v", err)
	}
	if wroteProjectFile {
		t.Fatalf("writer flow should not report durable project write after rollback")
	}
	if preimage.Exists {
		t.Fatalf("preimage should be missing before create failure: %+v", preimage)
	}
	if result.Decision != "denied" || result.Status != "blocked" || result.Message != "status projection write failed" {
		t.Fatalf("unexpected blocked result: %+v", result)
	}
	if result.PostWriteVerified || result.PostWriteSHA256 != sha256Hex(created) || result.PostWriteSize != int64(len(created)) {
		t.Fatalf("unexpected created post-write facts: %+v", result)
	}
	if !result.RollbackCompensation {
		t.Fatalf("expected rollback compensation after created target writer error")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("created target should be removed after rollback: %v", err)
	}
}

func TestRunStatusProjectionApplyWriterRestoresDeletedTargetOnWriterError(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	before := []byte(`{"version":1}`)
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	record := Record{ID: 1, Key: "demo", RootPath: root}
	snapshot := Snapshot{SourceHash: "source-a"}
	result := ApplyStatusProjectionResult{
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetURI:               ".areaflow/status.json",
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	writerErr := errors.New("simulated delete failure")
	writer := func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
		if err := os.Remove(target); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		return StatusProjectionWriteResult{}, writerErr
	}

	gateOptions := statusProjectionApplyTestGateOptionsForRecord(t, record, true, sha256Hex(before), int64(len(before)))
	preimage, wroteProjectFile, err := runStatusProjectionApplyWriter(context.Background(), record, snapshot, ".areaflow/status.json", gateOptions, writer, &result)
	if err != nil {
		t.Fatalf("apply writer returned unexpected error: %v", err)
	}
	if wroteProjectFile {
		t.Fatalf("writer flow should not report durable project write after rollback")
	}
	if !preimage.Exists || preimage.SHA256 != sha256Hex(before) {
		t.Fatalf("unexpected preimage before delete failure: %+v", preimage)
	}
	if result.Decision != "denied" || result.Status != "blocked" || result.Message != "status projection write failed" {
		t.Fatalf("unexpected blocked result: %+v", result)
	}
	if result.PostWriteVerified || result.PostWriteSHA256 != "" || result.PostWriteSize != 0 {
		t.Fatalf("unexpected deleted post-write facts: %+v", result)
	}
	if !result.RollbackCompensation {
		t.Fatalf("expected rollback compensation after deleted target writer error")
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored target: %v", err)
	}
	if string(current) != string(before) {
		t.Fatalf("delete failure rollback restored %s, want %s", current, before)
	}
}

func TestRollbackStatusProjectionApplyFileRemovesCreatedTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	preimage, err := captureStatusProjectionApplyPreimage(Record{Key: "demo", RootPath: root}, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("capture missing preimage: %v", err)
	}
	if preimage.Exists {
		t.Fatalf("preimage should be missing: %+v", preimage)
	}
	after := []byte(`{"schema_version":1}`)
	if err := os.WriteFile(target, after, 0o644); err != nil {
		t.Fatalf("write created target: %v", err)
	}

	if err := rollbackStatusProjectionApplyFile(preimage, sha256Hex(after)); err != nil {
		t.Fatalf("rollback created target: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("created target should be removed: %v", err)
	}
}

func TestRunStatusProjectionApplyWriterRollsBackWhenProtectedPathFingerprintChanges(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	readme := filepath.Join(root, "workflow", "README.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(readme), 0o755); err != nil {
		t.Fatalf("mkdir workflow dir: %v", err)
	}
	before := []byte(`{"version":1}`)
	after := stableStatusProjectionTestJSON("source-a")
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	if err := os.WriteFile(readme, []byte("before\n"), 0o644); err != nil {
		t.Fatalf("write readme before: %v", err)
	}
	record := Record{ID: 1, Key: "demo", RootPath: root}
	snapshot := Snapshot{SourceHash: "source-a"}
	result := ApplyStatusProjectionResult{
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetURI:               ".areaflow/status.json",
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	writer := func(_ context.Context, _ Record, _ Snapshot, _ string) (StatusProjectionWriteResult, error) {
		if err := os.WriteFile(target, after, 0o644); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		if err := os.WriteFile(readme, []byte("after\n"), 0o644); err != nil {
			return StatusProjectionWriteResult{}, err
		}
		return StatusProjectionWriteResult{
			Target:                    target,
			Hash:                      sha256Hex(after),
			Size:                      int64(len(after)),
			RootContained:             true,
			StableProjectionValidated: true,
			AtomicReplaceUsed:         true,
		}, nil
	}

	gateOptions := statusProjectionApplyTestGateOptionsForRecord(t, record, true, sha256Hex(before), int64(len(before)))
	preimage, wroteProjectFile, err := runStatusProjectionApplyWriter(context.Background(), record, snapshot, ".areaflow/status.json", gateOptions, writer, &result)
	if err != nil {
		t.Fatalf("apply writer returned unexpected error: %v", err)
	}
	if wroteProjectFile {
		t.Fatalf("writer flow should not report durable project write after protected path rollback")
	}
	if !preimage.Exists || preimage.SHA256 != sha256Hex(before) {
		t.Fatalf("unexpected preimage: %+v", preimage)
	}
	if result.Decision != "denied" || result.Status != "blocked" || result.Message != "status projection protected paths changed during write" {
		t.Fatalf("unexpected blocked result: %+v", result)
	}
	if !result.PostWriteVerified || result.ProtectedPathsVerified {
		t.Fatalf("expected target verification but protected paths not verified: %+v", result)
	}
	if result.ProtectedPathBeforeHash == "" || result.ProtectedPathAfterHash == "" || result.ProtectedPathBeforeHash == result.ProtectedPathAfterHash {
		t.Fatalf("expected protected path fingerprint drift: %+v", result)
	}
	if !containsString(result.Blockers, "protected_path_fingerprint_changed_after_write") {
		t.Fatalf("expected protected path blocker: %+v", result.Blockers)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if string(current) != string(before) {
		t.Fatalf("status target rollback restored %s, want %s", current, before)
	}
}

func TestStatusProjectionProtectedPathFingerprintMatchesAuthorizationPacketScriptAlgorithm(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	paths := map[string]string{
		".areaflow/status.json":    `{"version":1}`,
		"workflow/README.md":       "workflow entry\n",
		"scripts/dev_tools/cli.py": "print('cli')\n",
		"workflow/versions/v1-mvp/execution/_shared/progress.json":    `{"done":0}`,
		"workflow/versions/v1-mvp/execution/_shared/nested/task.json": `{"task":"read-only"}`,
	}
	for rel, content := range paths {
		absolute := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	record := Record{ID: 1, Key: "demo", RootPath: root}
	goFingerprint, err := captureStatusProjectionProtectedPathFingerprint(record, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("capture protected path fingerprint: %v", err)
	}
	scriptHash, scriptEntries := statusProjectionScriptStyleFingerprintForTest(t, root, ".areaflow/status.json")
	if goFingerprint.Hash != scriptHash || goFingerprint.EntryCount != len(scriptEntries) {
		t.Fatalf("Go fingerprint %+v did not match script hash=%s entries=%d entries=%+v", goFingerprint, scriptHash, len(scriptEntries), scriptEntries)
	}

	if err := os.WriteFile(target, []byte(`{"version":2}`), 0o644); err != nil {
		t.Fatalf("write target drift: %v", err)
	}
	goFingerprintAfterTargetChange, err := captureStatusProjectionProtectedPathFingerprint(record, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("capture protected path fingerprint after target change: %v", err)
	}
	scriptHashAfterTargetChange, _ := statusProjectionScriptStyleFingerprintForTest(t, root, ".areaflow/status.json")
	if goFingerprintAfterTargetChange.Hash != goFingerprint.Hash || scriptHashAfterTargetChange != scriptHash {
		t.Fatalf("target path must be excluded from non-target fingerprint: before go=%s script=%s after go=%s script=%s", goFingerprint.Hash, scriptHash, goFingerprintAfterTargetChange.Hash, scriptHashAfterTargetChange)
	}
}

func TestRollbackStatusProjectionApplyFileRejectsHashDrift(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	before := []byte(`{"version":1}`)
	after := []byte(`{"schema_version":1}`)
	drift := []byte(`{"schema_version":2}`)
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatalf("write before: %v", err)
	}
	preimage, err := captureStatusProjectionApplyPreimage(Record{Key: "demo", RootPath: root}, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("capture preimage: %v", err)
	}
	if err := os.WriteFile(target, drift, 0o644); err != nil {
		t.Fatalf("write drift: %v", err)
	}

	if err := rollbackStatusProjectionApplyFile(preimage, sha256Hex(after)); err == nil {
		t.Fatalf("expected hash drift rollback to fail")
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if string(current) != string(drift) {
		t.Fatalf("rollback should not overwrite drifted file: %s", current)
	}
}

func statusProjectionApplyTestGateOptions(expectedExists bool, expectedSHA256 string, expectedSize int64) StatusProjectionApplyGateOptions {
	return StatusProjectionApplyGateOptions{
		ExpectedBeforeExists:    &expectedExists,
		ExpectedBeforeSHA256:    expectedSHA256,
		ExpectedBeforeSizeBytes: &expectedSize,
	}
}

func statusProjectionApplyTestGateOptionsForRecord(t *testing.T, record Record, expectedExists bool, expectedSHA256 string, expectedSize int64) StatusProjectionApplyGateOptions {
	t.Helper()
	options := statusProjectionApplyTestGateOptions(expectedExists, expectedSHA256, expectedSize)
	fingerprint, err := captureStatusProjectionProtectedPathFingerprint(record, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("capture protected path fingerprint: %v", err)
	}
	options.ProtectedPathFingerprintSHA256 = fingerprint.Hash
	return options
}

func statusProjectionScriptStyleFingerprintForTest(t *testing.T, root string, targetURI string) (string, []string) {
	t.Helper()
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, filepath.FromSlash(targetURI)))
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	entries := []string{}
	for _, protectedPath := range statusProjectionProtectedPathFingerprintPaths() {
		absolute := filepath.Join(rootAbs, filepath.FromSlash(protectedPath))
		absolute, err = filepath.Abs(absolute)
		if err != nil {
			t.Fatalf("resolve protected path %s: %v", protectedPath, err)
		}
		if absolute == targetAbs {
			continue
		}
		entries = append(entries, statusProjectionScriptStyleEntriesForTest(t, rootAbs, absolute)...)
	}
	return sha256Hex([]byte(strings.Join(entries, "\n"))), entries
}

func statusProjectionScriptStyleEntriesForTest(t *testing.T, root string, path string) []string {
	t.Helper()
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			t.Fatalf("rel missing protected path: %v", relErr)
		}
		return []string{filepath.ToSlash(rel) + "\tmissing"}
	}
	if err != nil {
		t.Fatalf("stat protected path %s: %v", path, err)
	}
	entry := statusProjectionScriptStyleEntryForTest(t, root, path, info)
	if !info.IsDir() {
		return []string{entry}
	}
	entries := []string{entry}
	children, err := os.ReadDir(path)
	if err != nil {
		t.Fatalf("read protected path dir %s: %v", path, err)
	}
	for _, child := range children {
		entries = append(entries, statusProjectionScriptStyleEntriesForTest(t, root, filepath.Join(path, child.Name()))...)
	}
	return entries
}

func statusProjectionScriptStyleEntryForTest(t *testing.T, root string, path string, info os.FileInfo) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("rel protected path: %v", err)
	}
	rel = filepath.ToSlash(rel)
	mode := info.Mode()
	if mode.IsRegular() {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read protected path file %s: %v", path, err)
		}
		return rel + "\tfile\t" + strconv.Itoa(len(content)) + "\t" + sha256Hex(content)
	}
	if mode.IsDir() {
		return rel + "\tdir"
	}
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			t.Fatalf("read symlink %s: %v", path, err)
		}
		return rel + "\tsymlink\t" + target
	}
	return rel + "\tother\t" + mode.String() + "\t" + strconv.FormatInt(info.Size(), 10)
}

func stableStatusProjectionTestJSON(sourceHash string) []byte {
	return []byte(`{
  "schema_version": 1,
  "project_id": "demo",
  "project_name": "Demo",
  "area_flow_url": "areaflow://projects/demo",
  "cutover_phase": "mirroring",
  "active_versions": [
    {
      "display_label": "v1",
      "version_kind": "workflow",
      "lifecycle_status": "active",
      "rough_progress": {
        "percent": 50,
        "label": "in progress",
        "blocked": false
      }
    }
  ],
  "last_synced_at": "2026-07-07T00:00:00Z",
  "source_snapshot_hash": "` + sourceHash + `",
  "compatibility": {
    "shim_lifecycle_state": "not_installed",
    "offline_source": ".areaflow/status.json",
    "blocked_commands": [
      "./task-loop run",
      "promotion apply",
      "write execution"
    ]
  }
}
`)
}
