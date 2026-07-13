package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOnlyVerifyQueueCommandResponseSafetyFacts(t *testing.T) {
	result := ReadOnlyVerifyQueueResult{
		Project:        Record{ID: 1, Key: "areamatrix"},
		Version:        WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:            RunRecord{ID: 3, Status: "queued", DryRun: false},
		Task:           RunTaskRecord{ID: 4, Status: "queued"},
		TargetPath:     "docs/README.md",
		EventID:        5,
		AuditEventID:   6,
		IdempotencyKey: "read-only-queue",
	}

	response := readOnlyVerifyQueueCommandResponse(result)
	if response["project_read_attempted"] != false ||
		response["project_read_allowed"] != false ||
		response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("read-only queue response should record no forbidden attempts: %+v", response)
	}
	if response["target_path"] != "docs/README.md" || response["run_id"] != int64(3) || response["run_task_id"] != int64(4) {
		t.Fatalf("unexpected read-only queue facts: %+v", response)
	}
}

func TestReadOnlyVerifyCommandResponseSafetyFacts(t *testing.T) {
	result := ReadOnlyVerifyResult{
		Project:                       Record{ID: 1, Key: "areamatrix"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "verified", DryRun: false},
		Worker:                        WorkerRecord{ID: 4, WorkerKey: "local-1"},
		Task:                          RunTaskRecord{ID: 5, Status: "verified"},
		Lease:                         LeaseRecord{ID: 6, Status: "completed"},
		Attempt:                       RunAttemptRecord{ID: 7, AttemptKind: "read_only_verify", DryRun: false},
		Artifact:                      ArtifactRecord{ID: 8, ArtifactType: "read_only_verify_report"},
		Gate:                          ExecutionApprovalGate{Status: "pass"},
		TargetPath:                    "docs/README.md",
		TargetSHA256:                  "abc123",
		TargetSizeBytes:               64,
		Status:                        "verified",
		Decision:                      "allowed",
		Message:                       "read-only verify completed without managed project writes",
		EventID:                       9,
		AuditEventID:                  10,
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
		AreaFlowExecutionStateWritten: true,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		VerificationPassed:            true,
	}

	response := readOnlyVerifyCommandResponse(result)
	if response["project_read_attempted"] != true ||
		response["project_read_allowed"] != true ||
		response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("read-only verify response should preserve read-only safety facts: %+v", response)
	}
	if response["artifact_type"] != "read_only_verify_report" ||
		response["target_path"] != "docs/README.md" ||
		response["target_sha256"] != "abc123" ||
		response["verification_passed"] != true {
		t.Fatalf("unexpected read-only verify facts: %+v", response)
	}
}

func TestNormalizeReadOnlyVerifyOptions(t *testing.T) {
	options := normalizeReadOnlyVerifyOptions(ReadOnlyVerifyOptions{
		WorkerKey: " local-1 ",
		RunID:     42,
	})
	if options.WorkerKey != "local-1" || options.RunID != 42 {
		t.Fatalf("unexpected normalized read-only verify options: %+v", options)
	}
	if options.LeaseTimeoutSeconds != 300 {
		t.Fatalf("lease timeout = %d, want 300", options.LeaseTimeoutSeconds)
	}
	if len(options.AllowedCapabilities) != 2 ||
		options.AllowedCapabilities[0] != "read_project" ||
		options.AllowedCapabilities[1] != "write_artifacts" {
		t.Fatalf("required capabilities = %+v, want read_project/write_artifacts", options.AllowedCapabilities)
	}
	if options.Actor != "local-user" || options.Reason == "" {
		t.Fatalf("default actor/reason not applied: %+v", options)
	}
}

func TestSafeProjectReadPathRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("create docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if got, err := safeProjectReadPath(root, "docs/README.md"); err != nil || filepath.Base(got) != "README.md" {
		t.Fatalf("safeProjectReadPath allowed file = %q err=%v", got, err)
	}
	for _, target := range []string{"../outside.txt", "/tmp/outside.txt", ""} {
		if got, err := safeProjectReadPath(root, target); err == nil {
			t.Fatalf("safeProjectReadPath(%q) = %q, want error", target, got)
		}
	}
}
