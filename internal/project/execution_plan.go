package project

import (
	"context"
	"time"
)

type ExecutionPlanPreview struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Gate                          ExecutionApprovalGate
	Status                        string
	Mode                          string
	Steps                         []ExecutionPlanStep
	Blockers                      []string
	ForbiddenActions              []string
	ProjectReadAttempted          bool
	ProjectWriteAttempted         bool
	ExecutionWriteAttempted       bool
	AreaFlowArtifactWritten       bool
	AreaFlowExecutionStateWritten bool
	EngineCallAttempted           bool
	CommandsRun                   bool
	SecretsResolved               bool
	NetworkUsed                   bool
	TaskClaimed                   bool
	WorkerStarted                 bool
	AttemptCreated                bool
	ArtifactCreated               bool
	GeneratedAt                   time.Time
}

type ExecutionPlanStep struct {
	Key                  string
	AttemptKind          string
	Status               string
	Message              string
	RequiredCapabilities []string
	Prerequisites        []string
	Blockers             []string
	ReadsProject         bool
	WritesProject        bool
	WritesAreaFlow       bool
	UsesEngine           bool
	RunsCommands         bool
	UsesSecrets          bool
	UsesNetwork          bool
	CreatesAttempt       bool
	CreatesArtifact      bool
	Metadata             map[string]any
}

func (s Store) PreviewExecutionPlan(ctx context.Context, runID int64) (ExecutionPlanPreview, error) {
	detail, err := s.GetRun(ctx, runID)
	if err != nil {
		return ExecutionPlanPreview{}, err
	}
	gate, err := s.ExecutionApprovalGate(ctx, runID, ExecutionApprovalGateOptions{
		RequiredCapabilities: []string{"write_artifacts"},
		SkipEnginePreview:    true,
	})
	if err != nil {
		return ExecutionPlanPreview{}, err
	}
	return BuildExecutionPlanPreview(detail, gate, time.Now().UTC()), nil
}

func BuildExecutionPlanPreview(detail RunDetail, gate ExecutionApprovalGate, generatedAt time.Time) ExecutionPlanPreview {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	preview := ExecutionPlanPreview{
		Project:                       gate.Project,
		Version:                       gate.Version,
		Run:                           gate.Run,
		Gate:                          gate,
		Status:                        "blocked",
		Mode:                          "read_only_execution_plan_preview",
		ForbiddenActions:              []string{"claim_task", "start_worker", "create_attempt", "create_artifact", "execute_engine", "run_commands", "resolve_secrets", "use_network", "write_managed_project", "write_workflow_execution", "git_checkpoint"},
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: false,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   false,
		WorkerStarted:                 false,
		AttemptCreated:                false,
		ArtifactCreated:               false,
		GeneratedAt:                   generatedAt,
	}
	gateBlockers := append([]string{}, gate.Blockers...)
	gateStatus := "ready"
	if gate.Status != "pass" {
		gateStatus = "blocked"
	}
	preview.Steps = []ExecutionPlanStep{
		{
			Key:                  "execution_approval_gate",
			AttemptKind:          "gate",
			Status:               gateStatus,
			Message:              "execution approval gate must pass before any worker execution step",
			RequiredCapabilities: gate.RequiredCapabilities,
			Blockers:             gateBlockers,
			Metadata: map[string]any{
				"gate_status": gate.Status,
				"mode":        gate.Mode,
			},
		},
		{
			Key:                  "copy",
			AttemptKind:          "copy",
			Status:               "blocked",
			Message:              "copy attempt remains closed until project write, engine, command, rollback and diff evidence gates are implemented",
			RequiredCapabilities: []string{"execute_agents", "read_project", "run_commands", "write_artifacts", "write_code"},
			Prerequisites:        []string{"execution_approval_gate"},
			Blockers: append(gateBlockers, []string{
				"copy_apply_not_implemented",
				"managed_project_write_not_open",
				"engine_execution_not_open",
				"rollback_plan_not_verified",
			}...),
			ReadsProject:    true,
			WritesProject:   true,
			WritesAreaFlow:  true,
			UsesEngine:      true,
			RunsCommands:    true,
			CreatesAttempt:  true,
			CreatesArtifact: true,
			Metadata: map[string]any{
				"next_design": "copy must produce diff evidence before any managed project write opens",
				"task_count":  len(detail.Tasks),
			},
		},
		{
			Key:                  "verify",
			AttemptKind:          "verify",
			Status:               "waiting",
			Message:              "verify waits for copy output; v0.6j proves only allowlisted read-only file hashing",
			RequiredCapabilities: []string{"read_project", "write_artifacts"},
			Prerequisites:        []string{"copy"},
			Blockers:             []string{"copy_output_missing", "verify_acceptance_policy_not_implemented"},
			ReadsProject:         true,
			WritesAreaFlow:       true,
			CreatesAttempt:       true,
			CreatesArtifact:      true,
			Metadata: map[string]any{
				"existing_proof": "read_only_verify",
			},
		},
		{
			Key:                  "approved_artifact_write",
			AttemptKind:          "approved_artifact_write",
			Status:               gatedStatus(gate.Status),
			Message:              "approved artifact write is the currently opened artifact-store-only execution step",
			RequiredCapabilities: []string{"write_artifacts"},
			Prerequisites:        []string{"execution_approval_gate"},
			Blockers:             blockersWhenGateBlocked(gate),
			WritesAreaFlow:       true,
			CreatesAttempt:       true,
			CreatesArtifact:      true,
			Metadata: map[string]any{
				"existing_proof": "approved_artifact_write",
			},
		},
		{
			Key:                  "checkpoint",
			AttemptKind:          "checkpoint",
			Status:               "blocked",
			Message:              "checkpoint remains closed until manage_git, dirty-state, scope-drift and rollback evidence gates exist",
			RequiredCapabilities: []string{"manage_git", "write_artifacts"},
			Prerequisites:        []string{"verify"},
			Blockers:             []string{"checkpoint_apply_not_implemented", "manage_git_not_open", "scope_drift_gate_not_implemented"},
			WritesAreaFlow:       true,
			RunsCommands:         true,
			CreatesAttempt:       true,
			CreatesArtifact:      true,
		},
		{
			Key:                  "repair",
			AttemptKind:          "repair",
			Status:               "waiting",
			Message:              "repair is entered only after verify failure; it cannot skip verify or checkpoint gates",
			RequiredCapabilities: []string{"execute_agents", "read_project", "run_commands", "write_artifacts", "write_code"},
			Prerequisites:        []string{"verify_failure"},
			Blockers:             []string{"verify_failure_missing", "repair_apply_not_implemented", "engine_execution_not_open"},
			ReadsProject:         true,
			WritesProject:        true,
			WritesAreaFlow:       true,
			UsesEngine:           true,
			RunsCommands:         true,
			CreatesAttempt:       true,
			CreatesArtifact:      true,
		},
	}
	preview.Blockers = collectExecutionPlanBlockers(preview.Steps)
	if len(preview.Blockers) == 0 {
		preview.Status = "ready"
	}
	return preview
}

func gatedStatus(gateStatus string) string {
	if gateStatus == "pass" {
		return "ready"
	}
	return "blocked"
}

func blockersWhenGateBlocked(gate ExecutionApprovalGate) []string {
	if gate.Status == "pass" {
		return []string{}
	}
	return append([]string{}, gate.Blockers...)
}

func collectExecutionPlanBlockers(steps []ExecutionPlanStep) []string {
	blockers := []string{}
	for _, step := range steps {
		if step.Status != "blocked" {
			continue
		}
		if len(step.Blockers) == 0 {
			blockers = append(blockers, step.Key)
			continue
		}
		for _, blocker := range step.Blockers {
			blockers = append(blockers, step.Key+": "+blocker)
		}
	}
	return blockers
}
