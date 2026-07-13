package project

import (
	"context"
	"time"
)

type ProjectWriteDesignGate struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Gate                          ExecutionApprovalGate
	Status                        string
	Mode                          string
	Items                         []ReadinessItem
	RequiredCapabilities          []string
	WriteSetFields                []string
	UnsupportedOperations         []string
	ApplySequence                 []string
	Blockers                      []string
	ForbiddenActions              []string
	ProjectWriteApplyOpen         bool
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

func (s Store) PreviewProjectWriteDesignGate(ctx context.Context, runID int64) (ProjectWriteDesignGate, error) {
	gate, err := s.ExecutionApprovalGate(ctx, runID, ExecutionApprovalGateOptions{
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_code"},
		SkipEnginePreview:    true,
		Mode:                 "read_only_project_write_design_gate_execution_approval",
	})
	if err != nil {
		return ProjectWriteDesignGate{}, err
	}
	return BuildProjectWriteDesignGate(gate, time.Now().UTC()), nil
}

func BuildProjectWriteDesignGate(gate ExecutionApprovalGate, generatedAt time.Time) ProjectWriteDesignGate {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	requiredCapabilities := []string{"read_project", "write_artifacts", "write_code"}
	result := ProjectWriteDesignGate{
		Project:              gate.Project,
		Version:              gate.Version,
		Run:                  gate.Run,
		Gate:                 gate,
		Status:               "ready",
		Mode:                 "read_only_project_write_design_gate",
		RequiredCapabilities: requiredCapabilities,
		WriteSetFields: []string{
			"operation",
			"target_path",
			"target_path_kind",
			"expected_before_sha256",
			"expected_before_size",
			"after_sha256",
			"after_size",
			"content_artifact_id",
			"patch_artifact_id",
			"verification_plan_artifact_id",
			"rollback_plan_artifact_id",
			"permission_capabilities",
			"approval_id",
		},
		UnsupportedOperations: []string{
			"delete",
			"move",
			"chmod",
			"binary_rewrite",
			"symlink_target",
			"project_root_escape",
			"glob_bulk_write",
		},
		ApplySequence: []string{
			"project_write_design_gate",
			"fixture_approved_project_write",
			"fixture_verify",
			"fixture_rollback_drill",
			"managed_project_generated_only_write",
			"managed_project_source_write",
			"checkpoint",
			"repair",
		},
		ForbiddenActions: []string{
			"claim_task",
			"start_worker",
			"create_lease",
			"create_attempt",
			"create_artifact",
			"execute_engine",
			"run_commands",
			"resolve_secrets",
			"use_network",
			"write_managed_project",
			"write_workflow_execution",
			"git_checkpoint",
			"delete_project_file",
			"move_project_file",
		},
		ProjectWriteApplyOpen:         false,
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
	addProjectWriteDesignItem(&result, "execution_approval_gate", gatedProjectWriteDesignStatus(gate.Status), "execution approval gate must pass before any future project write apply", map[string]any{
		"gate_status":           gate.Status,
		"required_capabilities": requiredCapabilities,
	})
	addProjectWriteDesignItem(&result, "write_set_contract", "pass", "future project write apply must start from an approved write-set artifact", map[string]any{
		"required_fields": result.WriteSetFields,
	})
	addProjectWriteDesignItem(&result, "unsupported_operations", "pass", "first project write apply excludes destructive or hard-to-rollback operations", map[string]any{
		"unsupported_operations": result.UnsupportedOperations,
	})
	addProjectWriteDesignItem(&result, "copy_verify_repair_split", "pass", "copy success is not done; verify, repair and checkpoint remain separate attempts and gates", map[string]any{
		"copy_attempt_kind":       "copy",
		"verify_attempt_kind":     "verify",
		"repair_attempt_kind":     "repair",
		"checkpoint_attempt_kind": "checkpoint",
	})
	addProjectWriteDesignItem(&result, "rollback_contract", "pass", "rollback only applies AreaFlow-owned write-set preimages when current hash still matches the failed attempt output", map[string]any{
		"current_hash_mismatch": "blocked",
		"destructive_rollback":  "forbidden",
	})
	addProjectWriteDesignItem(&result, "first_apply_sequence", "pass", "future implementation must prove fixture write, fixture verify and fixture rollback before touching managed project generated-only paths", map[string]any{
		"sequence": result.ApplySequence,
	})
	addProjectWriteDesignItem(&result, "read_only_design_gate", "pass", "design gate did not claim tasks, start workers, run commands, create artifacts or write project files", map[string]any{
		"project_write_apply_open":          false,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        false,
		"area_flow_execution_state_written": false,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
		"task_claimed":                      false,
		"worker_started":                    false,
		"attempt_created":                   false,
		"artifact_created":                  false,
	})
	if gate.Status != "pass" {
		result.Blockers = append(result.Blockers, gate.Blockers...)
	}
	if len(result.Blockers) > 0 {
		result.Status = "blocked"
	}
	return result
}

func addProjectWriteDesignItem(result *ProjectWriteDesignGate, key string, status string, message string, metadata map[string]any) {
	result.Items = append(result.Items, ReadinessItem{
		Key:      key,
		Status:   status,
		Message:  message,
		Metadata: metadata,
	})
	if status == "blocked" || status == "fail" {
		result.Blockers = append(result.Blockers, key+": "+message)
	}
}

func gatedProjectWriteDesignStatus(gateStatus string) string {
	if gateStatus == "pass" {
		return "pass"
	}
	return "blocked"
}
