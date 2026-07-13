package project

import "testing"

func TestExecutionForwardingV1CommandPreviewAllowsKnownTargetOnlyAsClosedPreview(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromParts(
		record,
		ShimReadiness{Project: record, Status: "blocked"},
		map[string]int{
			"run.read_only_verify_queue":        1,
			"worker.read_only_verify":           1,
			"run.approved_artifact_write_queue": 1,
			"worker.approved_artifact_write":    1,
		},
	)
	applyPreview := BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{})
	commandPreview := BuildExecutionForwardingV1CommandPreview(applyPreview, ExecutionForwardingV1CommandPreviewOptions{
		TaskType: "read_only_verify",
	})

	if commandPreview.Status != "blocked" {
		t.Fatalf("status = %q, want blocked while apply is closed", commandPreview.Status)
	}
	if commandPreview.Decision != "would_forward_after_approval" {
		t.Fatalf("decision = %q", commandPreview.Decision)
	}
	if commandPreview.TargetCommandType != "run.read_only_verify_queue" || commandPreview.TargetStatus != "available_scoped" {
		t.Fatalf("unexpected target: %+v", commandPreview)
	}
	if !commandPreview.AllowedTaskType || commandPreview.BlockedTaskType {
		t.Fatalf("unexpected target flags: %+v", commandPreview)
	}
	if !commandPreview.WouldCreateCommandRequestAfterApproval ||
		!commandPreview.WouldCreateRunAfterApproval ||
		!commandPreview.WouldCreateRunTaskAfterApproval ||
		!commandPreview.WouldCreateAuditEventAfterApproval {
		t.Fatalf("missing would-create fields: %+v", commandPreview)
	}
	if commandPreview.ApplyOpen ||
		commandPreview.ProjectWriteAllowed ||
		commandPreview.ExecutionWriteAllowed ||
		commandPreview.LegacyFallbackAllowed {
		t.Fatalf("preview should keep apply/write/fallback closed: %+v", commandPreview)
	}
	if !containsString(commandPreview.RequiredPacketFields, "forwarded_task_type") ||
		!containsString(commandPreview.RequiredCapabilities, "read_project") {
		t.Fatalf("missing packet fields/capabilities: %+v", commandPreview)
	}
	for _, key := range []string{
		"area_flow_command_created",
		"area_flow_run_created",
		"task_loop_run_forwarded",
		"legacy_task_loop_started",
		"project_write_attempted",
		"execution_write_attempted",
		"engine_call_attempted",
		"commands_run",
		"secrets_resolved",
		"network_used",
	} {
		if commandPreview.SafetyFacts[key] {
			t.Fatalf("safety fact %s should stay false: %+v", key, commandPreview.SafetyFacts)
		}
	}
	if !commandPreview.SafetyFacts["read_only_preview"] || !commandPreview.SafetyFacts["command_preview"] {
		t.Fatalf("preview facts should be true: %+v", commandPreview.SafetyFacts)
	}
}

func TestExecutionForwardingV1CommandPreviewFailsClosedForBlockedAndUnknownTargets(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := ExecutionForwardingV1ReadinessFromParts(record, ShimReadiness{Project: record, Status: "blocked"}, map[string]int{})
	applyPreview := BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{})

	blocked := BuildExecutionForwardingV1CommandPreview(applyPreview, ExecutionForwardingV1CommandPreviewOptions{
		TaskType: "engine_execution",
	})
	if blocked.Decision != "blocked_task_type_fail_closed" || !blocked.BlockedTaskType || blocked.AllowedTaskType {
		t.Fatalf("blocked target should fail closed: %+v", blocked)
	}
	if blocked.TargetStatus != "blocked" || !containsString(blocked.BlockedBy, "engine_execution") {
		t.Fatalf("blocked target metadata missing: %+v", blocked)
	}

	unknown := BuildExecutionForwardingV1CommandPreview(applyPreview, ExecutionForwardingV1CommandPreviewOptions{
		TaskType: "surprise_task",
	})
	if unknown.Decision != "unknown_task_type_fail_closed" || unknown.AllowedTaskType || unknown.BlockedTaskType {
		t.Fatalf("unknown target should fail closed: %+v", unknown)
	}
	if !containsString(unknown.BlockedBy, "task_type_not_in_forwarding_v1_policy") {
		t.Fatalf("unknown target blocker missing: %+v", unknown.BlockedBy)
	}
	for _, preview := range []ExecutionForwardingV1CommandPreview{blocked, unknown} {
		if preview.SafetyFacts["legacy_task_loop_started"] ||
			preview.SafetyFacts["project_write_attempted"] ||
			preview.SafetyFacts["execution_write_attempted"] ||
			preview.SafetyFacts["engine_call_attempted"] ||
			preview.SafetyFacts["commands_run"] ||
			preview.SafetyFacts["network_used"] {
			t.Fatalf("fail-closed preview attempted side effects: %+v", preview.SafetyFacts)
		}
	}
}
