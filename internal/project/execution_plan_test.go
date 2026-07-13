package project

import (
	"testing"
	"time"
)

func TestBuildExecutionPlanPreviewKeepsRealExecutionClosed(t *testing.T) {
	generated := time.Date(2026, 7, 2, 5, 0, 0, 0, time.UTC)
	run := RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "approved_artifact_write",
		RunKind:           "execution",
		Status:            "queued",
		DryRun:            false,
		StartedAt:         generated,
	}
	gate := ExecutionApprovalGate{
		Project:              Record{ID: 1, Key: "areamatrix"},
		Version:              WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                  run,
		Status:               "pass",
		Mode:                 "read_only_execution_approval_gate",
		RequiredCapabilities: []string{"write_artifacts"},
	}
	preview := BuildExecutionPlanPreview(RunDetail{
		Run: run,
		Tasks: []RunTaskRecord{{
			ID:       4,
			RunID:    3,
			TaskKind: "approved_artifact_write_task",
			Status:   "queued",
		}},
	}, gate, generated)

	if preview.Status != "blocked" || preview.Mode != "read_only_execution_plan_preview" {
		t.Fatalf("unexpected execution plan status: %+v", preview)
	}
	if preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.EngineCallAttempted ||
		preview.CommandsRun || preview.SecretsResolved || preview.NetworkUsed || preview.TaskClaimed ||
		preview.WorkerStarted || preview.AttemptCreated || preview.ArtifactCreated {
		t.Fatalf("execution plan preview should be read-only: %+v", preview)
	}
	copyStep := executionPlanStepByKey(preview.Steps, "copy")
	if copyStep.Status != "blocked" || !copyStep.WritesProject || !copyStep.UsesEngine || !copyStep.RunsCommands {
		t.Fatalf("copy step should remain blocked and risky: %+v", copyStep)
	}
	artifactStep := executionPlanStepByKey(preview.Steps, "approved_artifact_write")
	if artifactStep.Status != "ready" || artifactStep.WritesProject || !artifactStep.WritesAreaFlow || !artifactStep.CreatesArtifact {
		t.Fatalf("approved artifact write step should be the opened artifact-only step: %+v", artifactStep)
	}
	checkpointStep := executionPlanStepByKey(preview.Steps, "checkpoint")
	if checkpointStep.Status != "blocked" || len(checkpointStep.Blockers) == 0 {
		t.Fatalf("checkpoint step should remain blocked: %+v", checkpointStep)
	}
	if len(preview.Blockers) == 0 {
		t.Fatal("expected execution plan blockers for unopened real execution steps")
	}
}

func TestBuildExecutionPlanPreviewPropagatesGateBlockers(t *testing.T) {
	gate := ExecutionApprovalGate{
		Project:              Record{ID: 1, Key: "areamatrix"},
		Version:              WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                  RunRecord{ID: 3, RunKind: "execution", Status: "queued"},
		Status:               "blocked",
		Mode:                 "read_only_execution_approval_gate",
		RequiredCapabilities: []string{"write_artifacts"},
		Blockers:             []string{"workflow_approval: missing approved workflow approval record"},
	}
	preview := BuildExecutionPlanPreview(RunDetail{Run: gate.Run}, gate, time.Date(2026, 7, 2, 5, 5, 0, 0, time.UTC))

	gateStep := executionPlanStepByKey(preview.Steps, "execution_approval_gate")
	if gateStep.Status != "blocked" || len(gateStep.Blockers) != 1 {
		t.Fatalf("expected gate blockers to be exposed: %+v", gateStep)
	}
	artifactStep := executionPlanStepByKey(preview.Steps, "approved_artifact_write")
	if artifactStep.Status != "blocked" || len(artifactStep.Blockers) != 1 {
		t.Fatalf("expected artifact step to inherit gate blockers: %+v", artifactStep)
	}
}

func executionPlanStepByKey(steps []ExecutionPlanStep, key string) ExecutionPlanStep {
	for _, step := range steps {
		if step.Key == key {
			return step
		}
	}
	return ExecutionPlanStep{}
}
