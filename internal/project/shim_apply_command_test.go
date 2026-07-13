package project

import (
	"testing"
	"time"
)

func TestBuildApplyShimCommandResultRecordsPassingGateWithoutAreaMatrixWrites(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	authorization := ShimAuthorizationPacket{
		Project:         record,
		Mode:            "read_only_authorization_packet",
		ReadinessStatus: "blocked",
		ReadinessItems: []ShimReadinessItem{
			{Key: "explicit_edit_approval", Status: "blocked"},
		},
		AllowedFiles: []ShimFilePlan{
			{Path: ".areaflow/status.json"},
			{Path: "scripts/areaflow_shim.py"},
		},
		ForbiddenPaths: []string{"workflow/versions/**/execution/**"},
		ForbiddenActions: []string{
			"task-loop run forwarding",
		},
		RequiredPreflight: []string{
			"areaflow project status-projection-apply-packet areamatrix --json",
			"areaflow project status-projection-apply-gate areamatrix --json",
		},
		PostEditVerification: []string{"verify ./task-loop run returns blocked"},
		RollbackScope:        []string{"restore the captured preimage bytes for .areaflow/status.json"},
	}
	packet := shimApplyPacketFromAuthorization(authorization, ShimApplyPacketPreviewOptions{
		ExplicitApproval:           true,
		ApprovalID:                 "approval-1",
		ApprovalActor:              "tester",
		ApprovalReason:             "test",
		StatusProjectionPacketID:   "areamatrix:status_projection_apply_packet:status-packet-1",
		StatusProjectionGateID:     "areamatrix:status_projection_apply_gate:status-gate-1",
		ReadOnlySmokeEvidenceID:    "areamatrix:real_areamatrix_readonly_smoke:smoke-1",
		DirtyWorktreeReviewID:      "areamatrix:areamatrix_dirty_worktree_review:dirty-1",
		ProtectedPathFingerprintID: "areamatrix:protected_path_fingerprint:fingerprint-1",
		RollbackPlanID:             "areamatrix:rollback_plan:rollback-1",
	})
	gate := BuildShimApplyGate(authorization, shimApplyGateOptionsFromPacket(packet, packetTimeForTest()))
	if gate.Status != "pass" {
		t.Fatalf("test gate should pass: %+v", gate)
	}

	gateOptions := shimApplyGateOptionsFromPacket(packet, packetTimeForTest())
	result := BuildApplyShimCommandResult(gate, ApplyShimCommandOptions{Gate: gateOptions})

	if result.Status != "recorded" || result.Decision != "allowed" || !result.ApplyOpen {
		t.Fatalf("shim apply command should record passing gate: %+v", result)
	}
	if len(result.Blockers) != 0 || containsString(result.Blockers, "shim_apply_gate_not_pass") {
		t.Fatalf("unexpected blockers for passing gate: %+v", result.Blockers)
	}
	for _, key := range []string{
		"project_write_attempted",
		"execution_write_attempted",
		"task_loop_run_forwarded",
		"status_projection_written",
		"area_matrix_files_modified",
		"engine_call_attempted",
		"commands_run",
		"worker_scheduled",
		"secrets_resolved",
		"network_used",
		"areamatrix_protected_paths_touched",
	} {
		if result.SafetyFacts[key] {
			t.Fatalf("safety fact %s should stay false: %+v", key, result.SafetyFacts)
		}
	}
	for _, key := range []string{
		"apply_command_executed",
		"command_request_created",
		"area_flow_command_created",
	} {
		if !result.SafetyFacts[key] {
			t.Fatalf("safety fact %s should be true: %+v", key, result.SafetyFacts)
		}
	}
}

func TestBuildApplyShimCommandResultBlocksMissingGate(t *testing.T) {
	gate := ShimApplyGate{
		Project: projectRecordForShimApplyCommandTest(),
		Status:  "blocked",
	}

	result := BuildApplyShimCommandResult(gate, ApplyShimCommandOptions{})

	if !containsString(result.Blockers, "shim_apply_gate_not_pass") {
		t.Fatalf("missing gate blocker: %+v", result.Blockers)
	}
	if !containsString(result.Blockers, "shim_apply_gate_blocked") || !containsString(result.Blockers, "failure_mode_must_be_fail_closed") {
		t.Fatalf("missing protected command blockers: %+v", result.Blockers)
	}
	if !result.CommandRequestCreated || !result.AreaFlowCommandCreated {
		t.Fatalf("blocked shim apply should still record command evidence: %+v", result)
	}
	for _, key := range []string{
		"project_write_attempted",
		"execution_write_attempted",
		"task_loop_run_forwarded",
		"status_projection_written",
		"area_matrix_files_modified",
		"engine_call_attempted",
		"commands_run",
		"worker_scheduled",
		"secrets_resolved",
		"network_used",
		"areamatrix_protected_paths_touched",
	} {
		if result.SafetyFacts[key] {
			t.Fatalf("safety fact %s should stay false: %+v", key, result.SafetyFacts)
		}
	}
}

func packetTimeForTest() time.Time {
	return time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
}

func projectRecordForShimApplyCommandTest() Record {
	return Record{ID: 1, Key: "areamatrix"}
}
