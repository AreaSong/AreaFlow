package project

import "testing"

func TestApprovedArtifactWriteQueueCommandResponseSafetyFacts(t *testing.T) {
	result := ApprovedArtifactWriteQueueResult{
		Project:                       Record{ID: 1, Key: "areamatrix"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "queued", DryRun: false},
		Task:                          RunTaskRecord{ID: 4, Status: "queued"},
		ArtifactLabel:                 "approval-note",
		EventID:                       5,
		AuditEventID:                  6,
		IdempotencyKey:                "artifact-write-queue",
		AreaFlowExecutionStateWritten: true,
	}

	response := approvedArtifactWriteQueueCommandResponse(result)
	if response["project_read_attempted"] != false ||
		response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["area_flow_artifact_written"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("approved artifact write queue response should record no forbidden attempts: %+v", response)
	}
	if response["artifact_label"] != "approval-note" || response["run_id"] != int64(3) || response["run_task_id"] != int64(4) {
		t.Fatalf("unexpected approved artifact write queue facts: %+v", response)
	}
}

func TestApprovedArtifactWriteCommandResponseSafetyFacts(t *testing.T) {
	result := ApprovedArtifactWriteResult{
		Project:                       Record{ID: 1, Key: "areamatrix"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "artifact_written", DryRun: false},
		Worker:                        WorkerRecord{ID: 4, WorkerKey: "local-1"},
		Task:                          RunTaskRecord{ID: 5, Status: "artifact_written"},
		Lease:                         LeaseRecord{ID: 6, Status: "completed"},
		Attempt:                       RunAttemptRecord{ID: 7, AttemptKind: "approved_artifact_write", DryRun: false},
		Artifact:                      ArtifactRecord{ID: 8, ArtifactType: "approved_artifact_write_report"},
		Gate:                          ExecutionApprovalGate{Status: "pass"},
		ArtifactLabel:                 "approval-note",
		Status:                        "artifact_written",
		Decision:                      "allowed",
		Message:                       "approved artifact write completed in AreaFlow artifact store only",
		EventID:                       9,
		AuditEventID:                  10,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		ArtifactWritePassed:           true,
	}

	response := approvedArtifactWriteCommandResponse(result)
	if response["project_read_attempted"] != false ||
		response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["area_flow_artifact_written"] != true ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("approved artifact write response should preserve artifact-only safety facts: %+v", response)
	}
	if response["artifact_type"] != "approved_artifact_write_report" ||
		response["artifact_label"] != "approval-note" ||
		response["artifact_write_passed"] != true {
		t.Fatalf("unexpected approved artifact write facts: %+v", response)
	}
}

func TestNormalizeApprovedArtifactWriteOptions(t *testing.T) {
	options := normalizeApprovedArtifactWriteOptions(ApprovedArtifactWriteOptions{
		WorkerKey: " local-1 ",
		RunID:     42,
	})
	if options.WorkerKey != "local-1" || options.RunID != 42 {
		t.Fatalf("unexpected normalized approved artifact write options: %+v", options)
	}
	if options.LeaseTimeoutSeconds != 300 {
		t.Fatalf("lease timeout = %d, want 300", options.LeaseTimeoutSeconds)
	}
	if len(options.AllowedCapabilities) != 1 || options.AllowedCapabilities[0] != "write_artifacts" {
		t.Fatalf("required capabilities = %+v, want write_artifacts", options.AllowedCapabilities)
	}
	if options.Actor != "local-user" || options.Reason == "" {
		t.Fatalf("default actor/reason not applied: %+v", options)
	}
}

func TestNormalizeApprovedArtifactLabel(t *testing.T) {
	cases := map[string]string{
		"":                    "approved-artifact",
		" Approval Note ":     "Approval-Note",
		"../escape":           "escape",
		"notes/final report!": "notes-final-report",
	}
	for input, want := range cases {
		if got := normalizeApprovedArtifactLabel(input); got != want {
			t.Fatalf("normalizeApprovedArtifactLabel(%q) = %q, want %q", input, got, want)
		}
	}
}
