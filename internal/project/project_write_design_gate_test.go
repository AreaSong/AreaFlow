package project

import (
	"testing"
	"time"
)

func TestBuildProjectWriteDesignGateReadyButApplyClosed(t *testing.T) {
	generated := time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)
	gate := ExecutionApprovalGate{
		Project:              Record{ID: 1, Key: "areamatrix"},
		Version:              WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                  RunRecord{ID: 3, RunKind: "execution", Status: "queued"},
		Status:               "pass",
		Mode:                 "read_only_project_write_design_gate_execution_approval",
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_code"},
	}

	result := BuildProjectWriteDesignGate(gate, generated)

	if result.Status != "ready" || result.Mode != "read_only_project_write_design_gate" {
		t.Fatalf("unexpected design gate status: %+v", result)
	}
	if result.ProjectWriteApplyOpen {
		t.Fatal("project write apply must remain closed")
	}
	if result.ProjectReadAttempted || result.ProjectWriteAttempted || result.ExecutionWriteAttempted ||
		result.AreaFlowArtifactWritten || result.AreaFlowExecutionStateWritten || result.EngineCallAttempted ||
		result.CommandsRun || result.SecretsResolved || result.NetworkUsed || result.TaskClaimed ||
		result.WorkerStarted || result.AttemptCreated || result.ArtifactCreated {
		t.Fatalf("design gate should be read-only: %+v", result)
	}
	if !containsString(result.WriteSetFields, "expected_before_sha256") || !containsString(result.WriteSetFields, "rollback_plan_artifact_id") {
		t.Fatalf("write-set contract missing required safety fields: %+v", result.WriteSetFields)
	}
	if !containsString(result.UnsupportedOperations, "delete") || !containsString(result.UnsupportedOperations, "project_root_escape") {
		t.Fatalf("unsupported operations should include destructive/path escape cases: %+v", result.UnsupportedOperations)
	}
	if !containsString(result.ApplySequence, "fixture_rollback_drill") || !containsString(result.ApplySequence, "managed_project_generated_only_write") {
		t.Fatalf("apply sequence should prove fixture rollback before managed project write: %+v", result.ApplySequence)
	}
	approvalItem := projectWriteDesignItemByKey(result.Items, "execution_approval_gate")
	if approvalItem.Status != "pass" {
		t.Fatalf("expected execution approval item to pass: %+v", approvalItem)
	}
	readOnlyItem := projectWriteDesignItemByKey(result.Items, "read_only_design_gate")
	if readOnlyItem.Status != "pass" {
		t.Fatalf("expected read-only item to pass: %+v", readOnlyItem)
	}
}

func TestBuildProjectWriteDesignGatePropagatesGateBlockers(t *testing.T) {
	gate := ExecutionApprovalGate{
		Project:              Record{ID: 1, Key: "areamatrix"},
		Version:              WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                  RunRecord{ID: 3, RunKind: "execution", Status: "queued"},
		Status:               "blocked",
		Mode:                 "read_only_project_write_design_gate_execution_approval",
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_code"},
		Blockers:             []string{"workflow_approval: missing approved workflow approval record"},
	}

	result := BuildProjectWriteDesignGate(gate, time.Date(2026, 7, 2, 8, 10, 0, 0, time.UTC))

	if result.Status != "blocked" {
		t.Fatalf("expected blocked design gate when execution approval is blocked: %+v", result)
	}
	if len(result.Blockers) < 2 || !containsString(result.Blockers, "workflow_approval: missing approved workflow approval record") {
		t.Fatalf("expected execution gate blockers to be exposed: %+v", result.Blockers)
	}
	approvalItem := projectWriteDesignItemByKey(result.Items, "execution_approval_gate")
	if approvalItem.Status != "blocked" {
		t.Fatalf("expected execution approval item to be blocked: %+v", approvalItem)
	}
	if result.ProjectWriteApplyOpen || result.ProjectWriteAttempted || result.AttemptCreated || result.ArtifactCreated {
		t.Fatalf("blocked design gate should remain non-mutating: %+v", result)
	}
}

func projectWriteDesignItemByKey(items []ReadinessItem, key string) ReadinessItem {
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	return ReadinessItem{}
}
