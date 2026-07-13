package project

import "testing"

func TestRunnerPreviewRequestHashAndDefaultKey(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	options := normalizeRunnerPreviewOptions(record, version, RunnerPreviewOptions{
		Actor:      " local-user ",
		Reason:     " preview runner ",
		RiskLevel:  " low ",
		RiskPolicy: " pause ",
	})
	if options.Actor != "local-user" || options.Reason != "preview runner" || options.IdempotencyKey != "runner.preview:areamatrix:v2" {
		t.Fatalf("unexpected normalized runner preview options: %+v", options)
	}

	first, err := runnerPreviewRequestHash(record, version, options)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := runnerPreviewRequestHash(record, version, options)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("runner preview hash should be stable: %s != %s", first, second)
	}

	changed := options
	changed.Reason = "different reason"
	changedHash, err := runnerPreviewRequestHash(record, version, changed)
	if err != nil {
		t.Fatalf("changed hash failed: %v", err)
	}
	if first == changedHash {
		t.Fatalf("runner preview hash should include audit reason")
	}
}

func TestRunnerPreviewCommandResponseSafetyFacts(t *testing.T) {
	result := RunnerPreviewResult{
		Project: Record{ID: 1, Key: "areamatrix"},
		Version: WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run: RunRecord{
			ID:        3,
			RunType:   "runner_preview",
			Status:    "passed",
			DryRun:    true,
			ProjectID: 1,
		},
		Tasks:          []RunTaskRecord{{ID: 4, RunID: 3, TaskKind: "workflow_item_preview"}},
		Attempts:       []RunAttemptRecord{{ID: 5, AttemptKind: "copy"}, {ID: 6, AttemptKind: "verify"}},
		Artifacts:      []ArtifactRecord{{ID: 7, ArtifactType: "runner_preview_report", SHA256: "abc123", SizeBytes: 42}},
		Preflight:      RunnerPreflight{Status: "pass"},
		EventID:        8,
		AuditEventID:   9,
		IdempotencyKey: "runner.preview:areamatrix:v2",
	}

	response := runnerPreviewCommandResponse(result)
	if response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["area_matrix_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("runner preview response should record no forbidden attempts: %+v", response)
	}
	if response["run_id"] != int64(3) || response["run_type"] != "runner_preview" || response["run_status"] != "passed" || response["dry_run"] != true {
		t.Fatalf("unexpected run facts in response: %+v", response)
	}
	if response["artifact_type"] != "runner_preview_report" || response["artifact_sha256"] != "abc123" || response["artifact_size_bytes"] != int64(42) {
		t.Fatalf("unexpected artifact facts in response: %+v", response)
	}
	if response["event_id"] != int64(8) || response["audit_event_id"] != int64(9) || response["preflight_status"] != "pass" {
		t.Fatalf("unexpected evidence facts in response: %+v", response)
	}
}
