package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureProjectWriteQueueCommandResponseSafetyFacts(t *testing.T) {
	result := FixtureProjectWriteQueueResult{
		Project:                       Record{ID: 1, Key: "areamatrix-fixture"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "queued", DryRun: false},
		Task:                          RunTaskRecord{ID: 4, Status: "queued"},
		WriteSetArtifact:              ArtifactRecord{ID: 5, ArtifactType: "fixture_project_write_set"},
		TargetPath:                    "fixtures/input.txt",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		EventID:                       6,
		AuditEventID:                  7,
		IdempotencyKey:                "fixture-project-write-queue",
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
	}

	response := fixtureProjectWriteQueueCommandResponse(result)
	if response["project_read_attempted"] != false ||
		response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["area_flow_artifact_written"] != true ||
		response["area_flow_execution_state_written"] != true ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("fixture project write queue response should preserve queue-only safety facts: %+v", response)
	}
	if response["target_path"] != "fixtures/input.txt" ||
		response["run_id"] != int64(3) ||
		response["run_task_id"] != int64(4) ||
		response["write_set_artifact_id"] != int64(5) {
		t.Fatalf("unexpected fixture project write queue facts: %+v", response)
	}
}

func TestFixtureProjectWriteCommandResponseSafetyFacts(t *testing.T) {
	result := FixtureProjectWriteResult{
		Project:                       Record{ID: 1, Key: "areamatrix-fixture"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "rollback_verified", DryRun: false},
		Worker:                        WorkerRecord{ID: 4, WorkerKey: "local-1"},
		Task:                          RunTaskRecord{ID: 5, Status: "rollback_verified"},
		Lease:                         LeaseRecord{ID: 6, Status: "completed"},
		CopyAttempt:                   RunAttemptRecord{ID: 7, AttemptKind: "copy", Status: "passed", DryRun: false},
		VerifyAttempt:                 RunAttemptRecord{ID: 8, AttemptKind: "verify", Status: "passed", DryRun: false},
		RollbackAttempt:               RunAttemptRecord{ID: 9, AttemptKind: "rollback", Status: "passed", DryRun: false},
		WriteSetArtifact:              ArtifactRecord{ID: 10, ArtifactType: "fixture_project_write_set"},
		PreimageArtifact:              ArtifactRecord{ID: 11, ArtifactType: "fixture_project_write_preimage"},
		Artifact:                      ArtifactRecord{ID: 12, ArtifactType: "fixture_project_write_report"},
		Gate:                          ExecutionApprovalGate{Status: "pass"},
		TargetPath:                    "fixtures/input.txt",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		RestoredSHA256:                "before123",
		RestoredSize:                  12,
		Status:                        "rollback_verified",
		Decision:                      "allowed",
		Message:                       "fixture project write verified and rolled back",
		EventID:                       13,
		AuditEventID:                  14,
		IdempotencyKey:                "fixture-project-write",
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
		ProjectWriteAttempted:         true,
		ProjectWriteAllowed:           true,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		WriteSetPassed:                true,
		VerificationPassed:            true,
		RollbackAttempted:             true,
		RollbackVerified:              true,
	}

	response := fixtureProjectWriteCommandResponse(result)
	if response["project_read_attempted"] != true ||
		response["project_read_allowed"] != true ||
		response["project_write_attempted"] != true ||
		response["project_write_allowed"] != true ||
		response["execution_write_attempted"] != false ||
		response["area_flow_artifact_written"] != true ||
		response["area_flow_execution_state_written"] != true ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("fixture project write response should preserve approved fixture-only safety facts: %+v", response)
	}
	if response["artifact_type"] != "fixture_project_write_report" ||
		response["target_path"] != "fixtures/input.txt" ||
		response["write_set_passed"] != true ||
		response["verification_passed"] != true ||
		response["rollback_verified"] != true {
		t.Fatalf("unexpected fixture project write facts: %+v", response)
	}
}

func TestNormalizeFixtureProjectWriteOptions(t *testing.T) {
	options := normalizeFixtureProjectWriteOptions(FixtureProjectWriteOptions{
		WorkerKey: " local-1 ",
		RunID:     42,
	})
	if options.WorkerKey != "local-1" || options.RunID != 42 {
		t.Fatalf("unexpected normalized fixture project write options: %+v", options)
	}
	if options.LeaseTimeoutSeconds != 300 {
		t.Fatalf("lease timeout = %d, want 300", options.LeaseTimeoutSeconds)
	}
	if len(options.AllowedCapabilities) != 3 ||
		options.AllowedCapabilities[0] != "read_project" ||
		options.AllowedCapabilities[1] != "write_artifacts" ||
		options.AllowedCapabilities[2] != "write_code" {
		t.Fatalf("required capabilities = %+v, want read_project/write_artifacts/write_code", options.AllowedCapabilities)
	}
	if options.Actor != "local-user" || options.Reason == "" || options.Metadata == nil {
		t.Fatalf("default actor/reason/metadata not applied: %+v", options)
	}
}

func TestNormalizeFixtureProjectWriteQueueOptions(t *testing.T) {
	options := normalizeFixtureProjectWriteQueueOptions(
		Record{ID: 1, Key: "areamatrix-fixture"},
		WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		FixtureProjectWriteQueueOptions{
			TargetPath:           " fixtures/../fixtures/input.txt ",
			Content:              "after",
			ExpectedBeforeSHA256: " ABC123 ",
			ExpectedBeforeSize:   6,
		},
	)
	if options.TargetPath != "fixtures/input.txt" {
		t.Fatalf("target path = %q, want fixtures/input.txt", options.TargetPath)
	}
	if options.ExpectedBeforeSHA256 != "abc123" {
		t.Fatalf("expected before sha256 = %q, want abc123", options.ExpectedBeforeSHA256)
	}
	if options.Actor != "local-user" || options.Reason == "" || options.IdempotencyKey == "" {
		t.Fatalf("default queue fields not applied: %+v", options)
	}
}

func TestSafeFixtureProjectWritePath(t *testing.T) {
	root := t.TempDir()
	fixtures := filepath.Join(root, "fixtures")
	if err := os.Mkdir(fixtures, 0o755); err != nil {
		t.Fatalf("create fixtures: %v", err)
	}
	target := filepath.Join(fixtures, "input.txt")
	if err := os.WriteFile(target, []byte("before"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	got, err := safeFixtureProjectWritePath(root, "fixtures/input.txt")
	if err != nil {
		t.Fatalf("safeFixtureProjectWritePath allowed file error: %v", err)
	}
	targetReal, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	if got != targetReal {
		t.Fatalf("safeFixtureProjectWritePath = %q, want %q", got, targetReal)
	}

	for _, targetPath := range []string{"../outside.txt", "/tmp/outside.txt", "", "fixtures/missing.txt"} {
		if got, err := safeFixtureProjectWritePath(root, targetPath); err == nil {
			t.Fatalf("safeFixtureProjectWritePath(%q) = %q, want error", targetPath, got)
		}
	}
	if got, err := safeFixtureProjectWritePath(root, "fixtures"); err == nil {
		t.Fatalf("safeFixtureProjectWritePath(directory) = %q, want error", got)
	}
	symlinkPath := filepath.Join(fixtures, "link.txt")
	if err := os.Symlink(target, symlinkPath); err == nil {
		if got, err := safeFixtureProjectWritePath(root, "fixtures/link.txt"); err == nil {
			t.Fatalf("safeFixtureProjectWritePath(symlink) = %q, want error", got)
		}
	}
}
