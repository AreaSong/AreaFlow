package project

import (
	"testing"
	"time"
)

func TestExecutionForwardingV1ApplyPreviewBlocksWithoutApprovalProofAndRollback(t *testing.T) {
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

	preview := BuildExecutionForwardingV1ApplyPreview(readiness, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC),
	})

	if preview.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", preview.Status)
	}
	if preview.Mode != "read_only_execution_forwarding_v1_apply_preview" {
		t.Fatalf("mode = %q", preview.Mode)
	}
	if !preview.ApprovalRequired || preview.ApprovalStatus != "needs_approval" {
		t.Fatalf("approval = required %t status %q", preview.ApprovalRequired, preview.ApprovalStatus)
	}
	if preview.ApplyOpen {
		t.Fatalf("apply should stay closed")
	}
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:readiness", "blocked")
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:explicit_approval", "blocked")
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:command_api_contract", "pass")
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:target_policy", "pass")
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:proof_facts", "blocked")
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:rollback", "blocked")
	assertExecutionForwardingV1ApplyPreviewItem(t, preview, "forwarding_v1:read_only_preview", "pass")

	for _, key := range []string{
		"apply_open",
		"forwarding_v1_apply_open",
		"task_loop_run_forwarded",
		"legacy_task_loop_started",
		"project_write_attempted",
		"execution_write_attempted",
		"area_flow_command_created",
		"area_flow_run_created",
		"worker_scheduled",
		"engine_call_attempted",
		"commands_run",
		"secrets_resolved",
		"network_used",
		"source_write_open",
		"generated_retained_write_open",
		"repair_apply_open",
		"checkpoint_apply_open",
		"publish_apply_open",
		"restore_apply_open",
	} {
		if preview.SafetyFacts[key] {
			t.Fatalf("safety fact %s should stay false: %+v", key, preview.SafetyFacts)
		}
	}
	if !preview.SafetyFacts["read_only_preview"] {
		t.Fatalf("read_only_preview should be true")
	}
	if !containsString(preview.RequiredProofFacts, "legacy_task_loop_runner_not_started") ||
		!containsString(preview.RequiredProofFacts, "rollback_to_read_only_shim_verified") {
		t.Fatalf("missing required proof facts: %+v", preview.RequiredProofFacts)
	}
	if !containsString(preview.ApplyPacketFields, "readiness_snapshot_hash") ||
		!containsString(preview.ApplyPacketFields, "approval_id") ||
		!containsString(preview.ApplyPacketFields, "forwarded_task_type") ||
		!containsString(preview.ApplyPacketFields, "target_command_type") {
		t.Fatalf("missing apply packet fields: %+v", preview.ApplyPacketFields)
	}
	if !containsString(preview.FailClosedFields, "legacy_task_loop_started") ||
		!containsString(preview.FailClosedFields, "audit_event_id") {
		t.Fatalf("missing fail-closed fields: %+v", preview.FailClosedFields)
	}
	if len(preview.ForwardingTargets) != len(preview.AllowedTaskTypes) {
		t.Fatalf("forwarding targets = %d, want allowed task count %d", len(preview.ForwardingTargets), len(preview.AllowedTaskTypes))
	}
	for _, target := range preview.ForwardingTargets {
		if !containsString(preview.AllowedTaskTypes, target.TaskType) {
			t.Fatalf("target task type %q is not allowed: %+v", target.TaskType, preview.AllowedTaskTypes)
		}
		if target.FailureMode != "fail_closed" || target.ProjectWriteAllowed || target.ExecutionWriteAllowed || target.LegacyFallbackAllowed {
			t.Fatalf("target should remain fail-closed/no-write/no-fallback: %+v", target)
		}
	}
	blockedByTask := map[string]ExecutionForwardingV1BlockedTarget{}
	for _, target := range preview.BlockedTargets {
		blockedByTask[target.TaskType] = target
		if target.FailureMode != "fail_closed" ||
			target.SafetyFacts["legacy_task_loop_started"] ||
			target.SafetyFacts["project_write_attempted"] ||
			target.SafetyFacts["engine_call_attempted"] ||
			target.SafetyFacts["network_used"] {
			t.Fatalf("blocked target should fail closed without side effects: %+v", target)
		}
	}
	if _, ok := blockedByTask["copy_ready_source_write"]; !ok {
		t.Fatalf("source write target should be blocked: %+v", preview.BlockedTargets)
	}
	if _, ok := blockedByTask["engine_execution"]; !ok {
		t.Fatalf("engine execution target should be blocked: %+v", preview.BlockedTargets)
	}
}

func assertExecutionForwardingV1ApplyPreviewItem(t *testing.T, preview ExecutionForwardingV1ApplyPreview, key string, status string) {
	t.Helper()
	for _, item := range preview.Items {
		if item.Key == key {
			if item.Status != status {
				t.Fatalf("item %s status = %q, want %q", key, item.Status, status)
			}
			return
		}
	}
	t.Fatalf("missing apply preview item %q", key)
}
