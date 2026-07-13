package project

import "testing"

func TestFixtureExecutionQueueCommandResponseSafetyFacts(t *testing.T) {
	result := FixtureExecutionQueueResult{
		Project:        Record{ID: 1, Key: "areamatrix"},
		Version:        WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:            RunRecord{ID: 3, Status: "queued", DryRun: false},
		Task:           RunTaskRecord{ID: 4, Status: "queued"},
		EventID:        5,
		AuditEventID:   6,
		IdempotencyKey: "fixture-queue",
	}

	response := fixtureExecutionQueueCommandResponse(result)
	if response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("fixture queue response should record no forbidden attempts: %+v", response)
	}
	if response["run_id"] != int64(3) || response["run_task_id"] != int64(4) || response["dry_run"] != false {
		t.Fatalf("unexpected fixture queue facts: %+v", response)
	}
}

func TestFixtureExecutionCommandResponseSafetyFacts(t *testing.T) {
	result := FixtureExecutionResult{
		Project:                       Record{ID: 1, Key: "areamatrix"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "passed", DryRun: false},
		Worker:                        WorkerRecord{ID: 4, WorkerKey: "local-1"},
		Task:                          RunTaskRecord{ID: 5, Status: "passed"},
		Lease:                         LeaseRecord{ID: 6, Status: "completed"},
		Attempt:                       RunAttemptRecord{ID: 7, AttemptKind: "fixture_execution", DryRun: false},
		Artifact:                      ArtifactRecord{ID: 8, ArtifactType: "fixture_execution_report"},
		Gate:                          ExecutionApprovalGate{Status: "pass"},
		Status:                        "passed",
		Decision:                      "allowed",
		Message:                       "fixture execution applied in AreaFlow state only",
		EventID:                       9,
		AuditEventID:                  10,
		AreaFlowExecutionStateWritten: true,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
	}

	response := fixtureExecutionCommandResponse(result)
	if response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("fixture execution response should record no forbidden attempts: %+v", response)
	}
	if response["task_claimed"] != true ||
		response["lease_created"] != true ||
		response["attempt_created"] != true ||
		response["artifact_created"] != true ||
		response["area_flow_execution_state_written"] != true {
		t.Fatalf("fixture execution response should preserve execution facts: %+v", response)
	}
	if response["artifact_type"] != "fixture_execution_report" || response["dry_run"] != false {
		t.Fatalf("unexpected artifact/dry-run facts: %+v", response)
	}
}

func TestNormalizeFixtureExecutionOptions(t *testing.T) {
	options := normalizeFixtureExecutionOptions(FixtureExecutionOptions{
		WorkerKey: " local-1 ",
		RunID:     42,
	})
	if options.WorkerKey != "local-1" || options.RunID != 42 {
		t.Fatalf("unexpected normalized fixture execution options: %+v", options)
	}
	if options.LeaseTimeoutSeconds != 300 {
		t.Fatalf("lease timeout = %d, want 300", options.LeaseTimeoutSeconds)
	}
	if len(options.AllowedCapabilities) != 4 {
		t.Fatalf("required capability count = %d, want 4: %+v", len(options.AllowedCapabilities), options.AllowedCapabilities)
	}
	if options.Actor != "local-user" || options.Reason == "" {
		t.Fatalf("default actor/reason not applied: %+v", options)
	}
}
