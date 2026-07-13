package project

import (
	"testing"
	"time"
)

func TestBuildExecutionApprovalGateBlocksDryRunPreview(t *testing.T) {
	gate := BuildExecutionApprovalGate(
		Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"},
		WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored"},
		RunDetail{
			Run:   RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunKind: "execution", Status: "passed", DryRun: true},
			Tasks: []RunTaskRecord{{ID: 4, Status: "queued"}},
		},
		ApprovalRecord{ID: 5, Decision: "approved", ApprovalKind: "workflow_transition"},
		true,
		GateResult{ID: 6, Status: "pass"},
		true,
		GateResult{ID: 7, Status: "pass"},
		true,
		CodexCLIAdapterPreview{Status: "needs_approval", Blockers: []string{"execution_approval_required"}},
		[]WorkerRecord{{ID: 8, Status: "online", Capabilities: []string{"read_project", "write_artifacts", "run_commands", "execute_agents"}}},
		ExecutionApprovalGateOptions{GeneratedAt: time.Date(2026, 7, 1, 16, 0, 0, 0, time.UTC)},
	)

	if gate.Status != "blocked" {
		t.Fatalf("gate status = %q, want blocked", gate.Status)
	}
	assertReadinessItem(t, ProjectReadiness{Items: gate.Items}, "dry_run_boundary", "blocked")
	assertReadinessItem(t, ProjectReadiness{Items: gate.Items}, "run_status", "blocked")
	if gate.ProjectWriteAttempted || gate.ExecutionWriteAttempted || gate.EngineCallAttempted ||
		gate.CommandsRun || gate.SecretsResolved || gate.NetworkUsed || gate.TaskClaimed ||
		gate.WorkerStarted || gate.AttemptCreated || gate.ArtifactCreated {
		t.Fatalf("execution approval gate should be read-only: %+v", gate)
	}
}

func TestBuildExecutionApprovalGatePassesWhenPrerequisitesAreReady(t *testing.T) {
	gate := BuildExecutionApprovalGate(
		Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"},
		WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored"},
		RunDetail{
			Run:   RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunKind: "execution", Status: "queued", DryRun: false},
			Tasks: []RunTaskRecord{{ID: 4, Status: "queued"}},
		},
		ApprovalRecord{ID: 5, Decision: "approved", ApprovalKind: "workflow_transition"},
		true,
		GateResult{ID: 6, Status: "pass"},
		true,
		GateResult{ID: 7, Status: "pass"},
		true,
		CodexCLIAdapterPreview{Status: "needs_approval", Blockers: []string{"execution_approval_required"}},
		[]WorkerRecord{{ID: 8, Status: "online", Capabilities: []string{"read_project", "write_artifacts", "run_commands", "execute_agents"}}},
		ExecutionApprovalGateOptions{GeneratedAt: time.Date(2026, 7, 1, 16, 0, 0, 0, time.UTC)},
	)

	if gate.Status != "pass" {
		t.Fatalf("gate status = %q, want pass: %+v", gate.Status, gate.Blockers)
	}
	for _, key := range []string{
		"run_kind",
		"run_status",
		"dry_run_boundary",
		"workflow_version_authored",
		"run_tasks_ready",
		"workflow_approval",
		"approval_gate",
		"live_mapping_gate",
		"engine_adapter_preview",
		"worker_online",
		"worker_capabilities",
		"read_only_boundary",
	} {
		assertReadinessItem(t, ProjectReadiness{Items: gate.Items}, key, "pass")
	}
	if len(gate.RequiredCapabilities) != 4 {
		t.Fatalf("required capability count = %d, want 4: %+v", len(gate.RequiredCapabilities), gate.RequiredCapabilities)
	}
}

func TestBuildExecutionApprovalGateCanSkipEnginePreviewForReadOnlyStep(t *testing.T) {
	gate := BuildExecutionApprovalGate(
		Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"},
		WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored"},
		RunDetail{
			Run:   RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunKind: "execution", Status: "queued", DryRun: false},
			Tasks: []RunTaskRecord{{ID: 4, Status: "queued"}},
		},
		ApprovalRecord{ID: 5, Decision: "approved", ApprovalKind: "workflow_transition"},
		true,
		GateResult{ID: 6, Status: "pass"},
		true,
		GateResult{ID: 7, Status: "pass"},
		true,
		CodexCLIAdapterPreview{Status: "blocked", Blockers: []string{"engine_profile_disabled"}},
		[]WorkerRecord{{ID: 8, Status: "online", Capabilities: []string{"read_project", "write_artifacts"}}},
		ExecutionApprovalGateOptions{
			RequiredCapabilities: []string{"read_project", "write_artifacts"},
			SkipEnginePreview:    true,
			Mode:                 "read_only_verify_gate",
			GeneratedAt:          time.Date(2026, 7, 2, 2, 0, 0, 0, time.UTC),
		},
	)

	if gate.Status != "pass" {
		t.Fatalf("gate status = %q, want pass: %+v", gate.Status, gate.Blockers)
	}
	if gate.Mode != "read_only_verify_gate" {
		t.Fatalf("gate mode = %q", gate.Mode)
	}
	assertReadinessItem(t, ProjectReadiness{Items: gate.Items}, "engine_adapter_preview", "pass")
	if len(gate.RequiredCapabilities) != 2 {
		t.Fatalf("required capability count = %d, want 2: %+v", len(gate.RequiredCapabilities), gate.RequiredCapabilities)
	}
}
