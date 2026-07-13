package project

import (
	"testing"
	"time"
)

func TestBuildManagedGeneratedWriteGateReadyButApplyClosed(t *testing.T) {
	generated := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	gate := ExecutionApprovalGate{
		Project:              Record{ID: 1, Key: "areamatrix"},
		Version:              WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                  RunRecord{ID: 3, RunKind: "execution", Status: "queued"},
		Status:               "pass",
		Mode:                 "read_only_managed_generated_write_gate_execution_approval",
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_generated"},
	}

	result := BuildManagedGeneratedWriteGate(gate, generated)

	if result.Status != "ready" || result.Mode != "read_only_managed_generated_write_gate" {
		t.Fatalf("unexpected managed generated write gate status: %+v", result)
	}
	if !result.GeneratedOnlyWriteReady {
		t.Fatalf("generated-only write should be ready for future apply planning: %+v", result)
	}
	if result.GeneratedOnlyApplyOpen {
		t.Fatal("generated-only apply must remain closed in this gate")
	}
	if result.ProjectReadAttempted || result.ProjectWriteAttempted || result.ExecutionWriteAttempted ||
		result.AreaFlowArtifactWritten || result.AreaFlowExecutionStateWritten || result.EngineCallAttempted ||
		result.CommandsRun || result.SecretsResolved || result.NetworkUsed || result.TaskClaimed ||
		result.WorkerStarted || result.LeaseCreated || result.AttemptCreated || result.ArtifactCreated {
		t.Fatalf("managed generated write gate should be read-only: %+v", result)
	}
	if !containsString(result.AllowedGeneratedPrefixes, ".areaflow/generated/") || !containsString(result.AllowedGeneratedPrefixes, ".areamatrix/generated/") {
		t.Fatalf("generated prefix policy missing expected prefixes: %+v", result.AllowedGeneratedPrefixes)
	}
	if !containsString(result.RequiredWriteSetFields, "generated_only") || !containsString(result.RequiredWriteSetFields, "rollback_plan_artifact_id") {
		t.Fatalf("write-set contract missing generated-only safety fields: %+v", result.RequiredWriteSetFields)
	}
	if !containsString(result.UnsupportedOperations, "source_write") || !containsString(result.UnsupportedOperations, "workflow_execution_write") {
		t.Fatalf("unsupported operations should keep source/execution writes blocked: %+v", result.UnsupportedOperations)
	}
	readOnlyItem := managedGeneratedWriteItemByKey(result.Items, "read_only_gate")
	if readOnlyItem.Status != "pass" {
		t.Fatalf("expected read-only gate item to pass: %+v", readOnlyItem)
	}
}

func TestBuildManagedGeneratedWriteGatePropagatesGateBlockers(t *testing.T) {
	gate := ExecutionApprovalGate{
		Project:              Record{ID: 1, Key: "areamatrix"},
		Version:              WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                  RunRecord{ID: 3, RunKind: "execution", Status: "queued"},
		Status:               "blocked",
		Mode:                 "read_only_managed_generated_write_gate_execution_approval",
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_generated"},
		Blockers:             []string{"workflow_approval: missing approved workflow approval record"},
	}

	result := BuildManagedGeneratedWriteGate(gate, time.Date(2026, 7, 2, 9, 10, 0, 0, time.UTC))

	if result.Status != "blocked" {
		t.Fatalf("expected blocked generated write gate when execution approval is blocked: %+v", result)
	}
	if result.GeneratedOnlyWriteReady {
		t.Fatalf("generated-only write readiness should be false when gate blocks: %+v", result)
	}
	if len(result.Blockers) < 2 || !containsString(result.Blockers, "workflow_approval: missing approved workflow approval record") {
		t.Fatalf("expected execution gate blockers to be exposed: %+v", result.Blockers)
	}
	approvalItem := managedGeneratedWriteItemByKey(result.Items, "execution_approval_gate")
	if approvalItem.Status != "blocked" {
		t.Fatalf("expected execution approval item to be blocked: %+v", approvalItem)
	}
	if result.GeneratedOnlyApplyOpen || result.ProjectWriteAttempted || result.LeaseCreated || result.AttemptCreated || result.ArtifactCreated {
		t.Fatalf("blocked generated write gate should remain non-mutating: %+v", result)
	}
}

func managedGeneratedWriteItemByKey(items []ReadinessItem, key string) ReadinessItem {
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	return ReadinessItem{}
}
