package project

import (
	"context"
	"time"
)

type ManagedGeneratedWriteGate struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Gate                          ExecutionApprovalGate
	Status                        string
	Mode                          string
	Items                         []ReadinessItem
	RequiredCapabilities          []string
	AllowedGeneratedPrefixes      []string
	RequiredWriteSetFields        []string
	UnsupportedOperations         []string
	ApplySequence                 []string
	Blockers                      []string
	ForbiddenActions              []string
	GeneratedOnlyWriteReady       bool
	GeneratedOnlyApplyOpen        bool
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
	LeaseCreated                  bool
	AttemptCreated                bool
	ArtifactCreated               bool
	GeneratedAt                   time.Time
}

func (s Store) PreviewManagedGeneratedWriteGate(ctx context.Context, runID int64) (ManagedGeneratedWriteGate, error) {
	gate, err := s.ExecutionApprovalGate(ctx, runID, ExecutionApprovalGateOptions{
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_generated"},
		SkipEnginePreview:    true,
		Mode:                 "read_only_managed_generated_write_gate_execution_approval",
	})
	if err != nil {
		return ManagedGeneratedWriteGate{}, err
	}
	return BuildManagedGeneratedWriteGate(gate, time.Now().UTC()), nil
}

func BuildManagedGeneratedWriteGate(gate ExecutionApprovalGate, generatedAt time.Time) ManagedGeneratedWriteGate {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	requiredCapabilities := []string{"read_project", "write_artifacts", "write_generated"}
	result := ManagedGeneratedWriteGate{
		Project:              gate.Project,
		Version:              gate.Version,
		Run:                  gate.Run,
		Gate:                 gate,
		Status:               "ready",
		Mode:                 "read_only_managed_generated_write_gate",
		RequiredCapabilities: requiredCapabilities,
		AllowedGeneratedPrefixes: []string{
			".areaflow/generated/",
			".areamatrix/generated/",
		},
		RequiredWriteSetFields: []string{
			"operation",
			"target_path",
			"target_path_kind",
			"expected_before_sha256",
			"expected_before_size",
			"after_sha256",
			"after_size",
			"content_artifact_id",
			"preimage_artifact_id",
			"verification_plan_artifact_id",
			"rollback_plan_artifact_id",
			"permission_capabilities",
			"generated_only",
			"approval_id",
		},
		UnsupportedOperations: []string{
			"source_write",
			"workflow_execution_write",
			"progress_json_write",
			"checkpoint",
			"repair",
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
			"managed_generated_write_gate",
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
			"write_source_file",
			"write_workflow_execution",
			"write_progress_json",
			"git_checkpoint",
			"delete_project_file",
			"move_project_file",
		},
		GeneratedOnlyWriteReady:       true,
		GeneratedOnlyApplyOpen:        false,
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
		LeaseCreated:                  false,
		AttemptCreated:                false,
		ArtifactCreated:               false,
		GeneratedAt:                   generatedAt,
	}
	addManagedGeneratedWriteItem(&result, "execution_approval_gate", gatedProjectWriteDesignStatus(gate.Status), "execution approval gate must pass before generated-only write can be considered", map[string]any{
		"gate_status":           gate.Status,
		"required_capabilities": requiredCapabilities,
	})
	addManagedGeneratedWriteItem(&result, "generated_prefix_policy", "pass", "future apply must stay inside generated-only prefixes declared by project policy", map[string]any{
		"default_generated_prefixes": result.AllowedGeneratedPrefixes,
		"source_write":               "blocked",
	})
	addManagedGeneratedWriteItem(&result, "write_set_contract", "pass", "future generated-only write must start from an approved write-set artifact with rollback evidence", map[string]any{
		"required_fields": result.RequiredWriteSetFields,
	})
	addManagedGeneratedWriteItem(&result, "unsupported_operations", "pass", "generated-only write excludes source, execution, checkpoint and destructive operations", map[string]any{
		"unsupported_operations": result.UnsupportedOperations,
	})
	addManagedGeneratedWriteItem(&result, "fixture_proof_required", "pass", "fixture write, verify and rollback drill must remain the immediate prerequisite", map[string]any{
		"sequence": result.ApplySequence,
	})
	addManagedGeneratedWriteItem(&result, "read_only_gate", "pass", "gate did not claim tasks, create leases, create attempts, create artifacts, run commands or write project files", map[string]any{
		"generated_only_write_ready":        result.GeneratedOnlyWriteReady,
		"generated_only_apply_open":         result.GeneratedOnlyApplyOpen,
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
		"lease_created":                     false,
		"attempt_created":                   false,
		"artifact_created":                  false,
	})
	if gate.Status != "pass" {
		result.Blockers = append(result.Blockers, gate.Blockers...)
		result.GeneratedOnlyWriteReady = false
	}
	if len(result.Blockers) > 0 {
		result.Status = "blocked"
	}
	return result
}

func addManagedGeneratedWriteItem(result *ManagedGeneratedWriteGate, key string, status string, message string, metadata map[string]any) {
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
