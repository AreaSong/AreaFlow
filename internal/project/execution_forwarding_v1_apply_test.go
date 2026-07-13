package project

import (
	"strings"
	"testing"
)

func TestEvaluateExecutionForwardingV1ApplyBlocksMissingPacket(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, ExecutionForwardingV1ApplyGateOptions{})

	result := evaluateExecutionForwardingV1Apply(applyPreview.Project, gate, ApplyExecutionForwardingV1Options{})

	if result.Status != "blocked" || result.Decision != "denied" {
		t.Fatalf("expected blocked apply result: %+v", result)
	}
	if !containsString(result.Blockers, "execution_forwarding_v1_apply_gate_blocked") ||
		!containsString(result.Blockers, "explicit_execution_forwarding_v1_approval_missing") {
		t.Fatalf("expected gate blockers: %+v", result.Blockers)
	}
	if !result.CommandRequestCreated || !result.AreaFlowCommandCreated {
		t.Fatalf("expected AreaFlow command evidence flags: %+v", result)
	}
	assertExecutionForwardingV1ApplyNoRuntimeSideEffects(t, result)
	if !result.SafetyFacts["apply_command_executed"] || !result.SafetyFacts["command_request_created"] {
		t.Fatalf("expected command safety facts: %+v", result.SafetyFacts)
	}
	if result.SafetyFacts["forwarding_v1_apply_open"] {
		t.Fatalf("blocked result must not open forwarding: %+v", result.SafetyFacts)
	}
}

func TestEvaluateExecutionForwardingV1ApplyAllowsPassingGateWithoutRuntimeSideEffects(t *testing.T) {
	applyPreview := executionForwardingV1ApplyGateFixturePreview()
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, ExecutionForwardingV1ApplyPacketPreviewOptions{
		ExplicitApproval:           true,
		ApprovalID:                 "approval-1",
		ApprovalActor:              "as",
		ApprovalReason:             "approve forwarding v1",
		LegacyNonWriteProofID:      executionForwardingV1ExpectedLegacyNonWriteProofID(applyPreview.Readiness, applyPreview.Project.Key),
		RollbackPlanID:             executionForwardingV1ExpectedRollbackPlanID(applyPreview.Readiness, applyPreview.Project.Key),
		ProtectedPathFingerprintID: executionForwardingV1ExpectedProtectedPathFingerprintID(applyPreview.Readiness, applyPreview.Project.Key),
		IdempotencyKey:             "forwarding-key",
		AuditCorrelationID:         "audit-forwarding-key",
	})
	gateOptions := executionForwardingV1ApplyGateOptionsFromPacket(packet, applyPreview.GeneratedAt)
	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, gateOptions)

	result := evaluateExecutionForwardingV1Apply(applyPreview.Project, gate, ApplyExecutionForwardingV1Options{Gate: gateOptions})

	if result.Status != "applied" || result.Decision != "allowed" || len(result.Blockers) != 0 {
		t.Fatalf("expected allowed apply result: %+v", result)
	}
	if !result.CommandRequestCreated || !result.AreaFlowCommandCreated {
		t.Fatalf("expected AreaFlow command evidence flags: %+v", result)
	}
	assertExecutionForwardingV1ApplyNoRuntimeSideEffects(t, result)
	if !result.SafetyFacts["forwarding_v1_apply_open"] {
		t.Fatalf("allowed command should record forwarding apply open in AreaFlow state: %+v", result.SafetyFacts)
	}
}

func TestExecutionForwardingV1ApplyRequestHashChangesWithGatePacket(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", RootPath: "/repo"}
	base := ApplyExecutionForwardingV1Options{Gate: ExecutionForwardingV1ApplyGateOptions{
		AllowedTaskTypes:      []string{"read_only_verify"},
		ReadinessSnapshotHash: "hash-a",
		IdempotencyKey:        "key-a",
		AuditCorrelationID:    "audit-a",
		FailureMode:           "fail_closed",
	}}
	first, err := executionForwardingV1ApplyRequestHash(record, base)
	if err != nil {
		t.Fatalf("first request hash failed: %v", err)
	}
	second, err := executionForwardingV1ApplyRequestHash(record, base)
	if err != nil {
		t.Fatalf("second request hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("same request hash differed: %s != %s", first, second)
	}
	base.Gate.ReadinessSnapshotHash = "hash-b"
	changed, err := executionForwardingV1ApplyRequestHash(record, base)
	if err != nil {
		t.Fatalf("changed request hash failed: %v", err)
	}
	if first == changed {
		t.Fatalf("request hash should change with gate packet")
	}

	key := executionForwardingV1ApplyIdempotencyKey(record, base.Gate)
	if !strings.HasPrefix(key, "project.execution_forwarding_v1.apply:areamatrix:") {
		t.Fatalf("unexpected idempotency key: %s", key)
	}
}

func assertExecutionForwardingV1ApplyNoRuntimeSideEffects(t *testing.T, result ApplyExecutionForwardingV1Result) {
	t.Helper()
	if result.AreaFlowRunCreated ||
		result.AreaFlowRunTaskCreated ||
		result.AreaFlowRunAttemptCreated ||
		result.AreaFlowArtifactCreated ||
		result.TaskLoopRunForwarded ||
		result.LegacyTaskLoopStarted ||
		result.LegacyProgressWritten ||
		result.LegacyLogsWritten ||
		result.LegacyCheckpointWritten ||
		result.ProjectWriteAttempted ||
		result.ExecutionWriteAttempted ||
		result.EngineCallAttempted ||
		result.CommandsRun ||
		result.SecretsResolved ||
		result.NetworkUsed ||
		result.AreaMatrixProtectedPathsTouched {
		t.Fatalf("unexpected runtime/project side effects: %+v", result)
	}
}
