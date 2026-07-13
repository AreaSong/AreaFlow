package project

import (
	"context"
	"time"
)

type ExecutionForwardingV1ReadinessOptions struct{}

type ExecutionForwardingV1Readiness struct {
	Project          Record
	Status           string
	Mode             string
	Items            []ExecutionForwardingV1ReadinessItem
	AllowedTaskTypes []string
	CommandEvidence  map[string]int
	Capabilities     []string
	ForbiddenActions []string
	SafetyFacts      map[string]bool
	NextSteps        []ExecutionForwardingV1NextStep
	GeneratedAt      time.Time
}

type ExecutionForwardingV1ReadinessItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ExecutionForwardingV1NextStep struct {
	Key         string
	Owner       string
	Action      string
	RiskLevel   string
	BlockedBy   []string
	NextCommand string
	Metadata    map[string]any
}

var executionForwardingV1AllowedTaskTypes = []string{
	"read_only_verify",
	"doctor_readiness",
	"artifact_evidence",
	"status_projection_validation",
	"release_readiness_check",
}

var executionForwardingV1CommandEvidenceTypes = []string{
	"run.read_only_verify_queue",
	"worker.read_only_verify",
	"run.approved_artifact_write_queue",
	"worker.approved_artifact_write",
	"completion.validation_proof.record",
	"completion.execution_cutover_proof.record",
}

func (s Store) ExecutionForwardingV1Readiness(ctx context.Context, record Record, _ ExecutionForwardingV1ReadinessOptions) (ExecutionForwardingV1Readiness, error) {
	shim, err := s.ShimReadiness(ctx, record)
	if err != nil {
		return ExecutionForwardingV1Readiness{}, err
	}
	commandEvidence, err := s.completedCommandEvidenceCounts(ctx, record.ID)
	if err != nil {
		return ExecutionForwardingV1Readiness{}, err
	}
	protectedPathProof, err := s.LatestProtectedPathProofForProject(ctx, record)
	if err != nil {
		return ExecutionForwardingV1Readiness{}, err
	}
	rollbackProof, err := s.LatestExecutionCutoverProofForProject(ctx, record)
	if err != nil {
		return ExecutionForwardingV1Readiness{}, err
	}
	return ExecutionForwardingV1ReadinessFromPartsWithProofs(record, shim, commandEvidence, protectedPathProof, rollbackProof), nil
}

func ExecutionForwardingV1ReadinessFromParts(record Record, shim ShimReadiness, commandEvidence map[string]int) ExecutionForwardingV1Readiness {
	return buildExecutionForwardingV1Readiness(record, shim, commandEvidence, nil, nil)
}

func ExecutionForwardingV1ReadinessFromPartsWithProtectedPathProof(record Record, shim ShimReadiness, commandEvidence map[string]int, protectedPathProof ProtectedPathProof) ExecutionForwardingV1Readiness {
	if protectedPathProof.EventID == 0 && protectedPathProof.ProofStatus == "" {
		return buildExecutionForwardingV1Readiness(record, shim, commandEvidence, nil, nil)
	}
	return buildExecutionForwardingV1Readiness(record, shim, commandEvidence, &protectedPathProof, nil)
}

func ExecutionForwardingV1ReadinessFromPartsWithProofs(record Record, shim ShimReadiness, commandEvidence map[string]int, protectedPathProof ProtectedPathProof, rollbackProof ExecutionCutoverProof) ExecutionForwardingV1Readiness {
	var protectedPathProofPtr *ProtectedPathProof
	if protectedPathProof.EventID != 0 || protectedPathProof.ProofStatus != "" {
		protectedPathProofPtr = &protectedPathProof
	}
	var rollbackProofPtr *ExecutionCutoverProof
	if rollbackProof.EventID != 0 || rollbackProof.ProofStatus != "" {
		rollbackProofPtr = &rollbackProof
	}
	return buildExecutionForwardingV1Readiness(record, shim, commandEvidence, protectedPathProofPtr, rollbackProofPtr)
}

func buildExecutionForwardingV1Readiness(record Record, shim ShimReadiness, commandEvidence map[string]int, protectedPathProof *ProtectedPathProof, rollbackProof *ExecutionCutoverProof) ExecutionForwardingV1Readiness {
	readiness := ExecutionForwardingV1Readiness{
		Project:          record,
		Status:           "pass",
		Mode:             "read_only_execution_forwarding_v1_readiness",
		AllowedTaskTypes: append([]string{}, executionForwardingV1AllowedTaskTypes...),
		CommandEvidence: copyCommandEvidence(
			commandEvidence,
			executionForwardingV1CommandEvidenceTypes,
		),
		Capabilities: []string{
			"read_project",
			"write_artifacts",
			"manage_workers",
		},
		ForbiddenActions: []string{
			"start_legacy_task_loop_runner",
			"write_legacy_progress_json",
			"write_legacy_logs",
			"write_legacy_checkpoint",
			"write_areamatrix_source",
			"write_areamatrix_execution_directory",
			"generated_retained_write",
			"repair_apply",
			"checkpoint_apply",
			"engine_execution",
			"secret_resolve",
			"network_api_integration",
			"publish_apply",
			"restore_apply",
		},
		SafetyFacts: map[string]bool{
			"read_only":                          true,
			"forwarding_v1_apply_open":           false,
			"task_loop_run_forwarded":            false,
			"legacy_task_loop_started":           false,
			"legacy_progress_written":            false,
			"legacy_logs_written":                false,
			"legacy_checkpoint_written":          false,
			"project_write_attempted":            false,
			"execution_write_attempted":          false,
			"area_flow_command_created":          false,
			"area_flow_run_created":              false,
			"worker_scheduled":                   false,
			"engine_call_attempted":              false,
			"commands_run":                       false,
			"secrets_resolved":                   false,
			"network_used":                       false,
			"source_write_open":                  false,
			"generated_retained_write_open":      false,
			"repair_apply_open":                  false,
			"checkpoint_apply_open":              false,
			"publish_apply_open":                 false,
			"restore_apply_open":                 false,
			"areamatrix_protected_paths_touched": false,
		},
		GeneratedAt: time.Now().UTC(),
	}

	readiness.add("allowed_task_scope", "scope", "pass", "Execution Forwarding v1 is limited to read-only and evidence task types", []string{"allowed task type list"}, "areaflow project execution-forwarding-v1-readiness "+record.Key+" --json", map[string]any{
		"allowed_task_types": readiness.AllowedTaskTypes,
	})
	readiness.add("forbidden_high_risk_targets", "scope", "pass", "source write, generated retained write, repair, checkpoint, engine, secret, network, publish and restore remain closed", []string{"forbidden action list"}, "areaflow project execution-forwarding-v1-readiness "+record.Key+" --json", map[string]any{
		"forbidden_actions": readiness.ForbiddenActions,
	})
	readiness.add("read_only_shim", "compatibility", statusForExecutionForwardingShim(shim), "AreaMatrix must first reach read_only_shim before task-loop run can forward", []string{"read_only_shim landed", "shim readiness pass"}, "areaflow project shim-readiness "+record.Key+" --json", map[string]any{
		"shim_status": shim.Status,
	})
	readiness.addCommandEvidence("read_only_verify_evidence", "execution_beta", commandEvidence, []string{"run.read_only_verify_queue", "worker.read_only_verify"}, "read-only verify evidence must exist before forwarding v1", "areaflow run read-only-verify-queue "+record.Key+" <version> --json")
	readiness.addCommandEvidence("artifact_evidence", "execution_beta", commandEvidence, []string{"run.approved_artifact_write_queue", "worker.approved_artifact_write"}, "AreaFlow-owned artifact evidence must exist before forwarding v1", "areaflow run approved-artifact-write-queue "+record.Key+" <version> --json")
	readiness.add("forwarding_command_api", "command_api", "pass", "Execution Forwarding v1 apply command exists and stays protected by packet, gate, idempotency and audit", []string{"Command API design", "idempotency key", "approval id", "audit response"}, "areaflow project execution-forwarding-v1-apply "+record.Key+" --json", map[string]any{
		"command_type":             executionForwardingV1ApplyCommandType,
		"apply_open":               false,
		"requires_apply_gate_pass": true,
		"creates_command_response": true,
	})
	readiness.addExecutionForwardingLegacyNonWriteProof(record, protectedPathProof)
	readiness.addExecutionForwardingRollbackProof(record, rollbackProof)
	forwardingCommandBlockers := executionForwardingV1CommandOpeningBlockers(readiness)

	readiness.NextSteps = []ExecutionForwardingV1NextStep{
		{
			Key:         "land_read_only_shim",
			Owner:       "project_owner",
			Action:      "land and verify the AreaMatrix read-only compatibility shim after explicit cross-repo approval",
			RiskLevel:   "R2 managed_write",
			BlockedBy:   []string{"explicit_edit_approval"},
			NextCommand: "areaflow project shim-readiness " + record.Key + " --json",
			Metadata:    map[string]any{"writes_areamatrix": true},
		},
		{
			Key:         "define_forwarding_v1_command",
			Owner:       "execution_owner",
			Action:      "run protected forwarding v1 apply command only after shim, approval and non-write proof pass",
			RiskLevel:   "R3 execution",
			BlockedBy:   forwardingCommandBlockers,
			NextCommand: "areaflow project execution-forwarding-v1-apply " + record.Key + " --json",
			Metadata:    map[string]any{"allowed_task_types": append([]string{}, executionForwardingV1AllowedTaskTypes...)},
		},
		{
			Key:         "prove_legacy_non_write",
			Owner:       "execution_owner",
			Action:      "prove legacy runner, progress, logs and checkpoint stay untouched during forwarding v1",
			RiskLevel:   "R3 execution",
			BlockedBy:   []string{"protected_path_proof_missing", "legacy_non_write_proof_missing"},
			NextCommand: "areaflow completion protected-path-proof record " + record.Key + " --status clean --summary <text> --evidence-uri <uri> --json",
			Metadata:    map[string]any{"legacy_state_read_only": true},
		},
	}

	return readiness
}

func executionForwardingV1CommandOpeningBlockers(readiness ExecutionForwardingV1Readiness) []string {
	blockers := []string{}
	if executionForwardingV1ReadinessItemStatus(readiness, "read_only_shim") != "pass" {
		blockers = append(blockers, "read_only_shim_missing")
	}
	if executionForwardingV1ReadinessItemStatus(readiness, "read_only_verify_evidence") != "pass" {
		blockers = append(blockers, "read_only_verify_evidence_missing")
	}
	if executionForwardingV1ReadinessItemStatus(readiness, "artifact_evidence") != "pass" {
		blockers = append(blockers, "artifact_evidence_missing")
	}
	if executionForwardingV1ReadinessItemStatus(readiness, "legacy_non_write_proof") != "pass" {
		blockers = append(blockers, "legacy_non_write_proof_missing")
	}
	if executionForwardingV1ReadinessItemStatus(readiness, "rollback_to_read_only_shim") != "pass" {
		blockers = append(blockers, "rollback_proof_missing")
	}
	blockers = append(blockers, "explicit_execution_cutover_approval_missing")
	return blockers
}

func (r *ExecutionForwardingV1Readiness) addExecutionForwardingLegacyNonWriteProof(record Record, proof *ProtectedPathProof) {
	status := "blocked"
	message := "legacy runner, progress, logs and checkpoint non-write proof is required"
	metadata := map[string]any{
		"legacy_task_loop_started":  false,
		"legacy_progress_written":   false,
		"legacy_logs_written":       false,
		"legacy_checkpoint_written": false,
		"proof_present":             false,
	}
	if proof != nil {
		metadata["proof_present"] = true
		metadata["proof_status"] = proof.ProofStatus
		metadata["proof_decision"] = proof.Decision
		metadata["proof_event_id"] = proof.EventID
		metadata["proof_audit_event_id"] = proof.AuditEventID
		metadata["proof_project_key"] = proof.Project.Key
		metadata["proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["proof_traceable_evidence"] = proofMetadataHasTraceableEvidence(proof.Metadata)
		metadata["authorized_approval_id"] = metadataString(proof.Metadata, "authorized_approval_id")
		metadata["authorized_allowed_paths"] = metadataStringSlice(proof.Metadata, "authorized_allowed_paths")
		metadata["authorized_dirty_output_hash"] = metadataString(proof.Metadata, "authorized_dirty_output_hash")
		metadata["authorized_reviewer"] = metadataString(proof.Metadata, "authorized_reviewer")
		metadata["authorized_rollback_evidence_uri"] = metadataString(proof.Metadata, "authorized_rollback_evidence_uri")
		metadata["authorized_touched_paths"] = metadataStringSlice(proof.Metadata, "authorized_touched_paths")
		metadata["authorized_proof_complete"] = proof.ProofStatus != "authorized" || protectedPathProofAuthorizedMetadataComplete(proof.Metadata)
		metadata["protected_path_proof_binding_status"] = metadataString(proof.Metadata, "protected_path_proof_binding_status")
		metadata["protected_path_proof_binding_blockers"] = protectedPathProofMetadataBindingBlockers(proof.Metadata)
		metadata["protected_path_set_hash"] = metadataString(proof.Metadata, "protected_path_set_hash")
		metadata["protected_path_set_count"] = metadataInt64(proof.Metadata, "protected_path_set_count")
		metadata["git_status_output_hash"] = proof.GitStatusOutputHash
		metadata["git_status_output_lines"] = proof.GitStatusOutputLines
		metadata["git_status_output_empty"] = metadataBool(proof.Metadata, "git_status_output_empty")
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["engine_call_attempted"] = proof.EngineCallAttempted
		metadata["commands_run"] = proof.CommandsRun
		metadata["git_status_run_by_command"] = proof.GitStatusRunByCommand
		metadata["areamatrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		if protectedPathProofCompletesAudit(*proof) &&
			proof.Project.Key == record.Key &&
			!proof.ProjectWriteAttempted &&
			!proof.ExecutionWriteAttempted &&
			!proof.EngineCallAttempted &&
			!proof.CommandsRun {
			status = "pass"
			message = "legacy runner, progress, logs and checkpoint non-write proof is clean"
		}
	}
	r.add("legacy_non_write_proof", "protected_paths", status, message, []string{"legacy runner bypass proof", "legacy progress/log/checkpoint non-write proof", "protected path proof"}, "areaflow completion protected-path-proof record "+record.Key+" --status clean --summary <text> --evidence-uri <uri> --json", metadata)
}

func (r *ExecutionForwardingV1Readiness) addExecutionForwardingRollbackProof(record Record, proof *ExecutionCutoverProof) {
	requiredFacts := executionForwardingV1RollbackProofFacts()
	status := "blocked"
	message := "forwarding v1 must be able to fail closed and roll back to read_only_shim"
	metadata := map[string]any{
		"rollback_target":      "read_only_shim",
		"required_proof_facts": requiredFacts,
		"proof_present":        false,
	}
	if proof != nil {
		missingFacts := missingStringFacts(proof.Facts, requiredFacts)
		metadata["proof_present"] = true
		metadata["proof_status"] = proof.ProofStatus
		metadata["proof_decision"] = proof.Decision
		metadata["proof_event_id"] = proof.EventID
		metadata["proof_audit_event_id"] = proof.AuditEventID
		metadata["proof_project_key"] = proof.Project.Key
		metadata["proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["missing_proof_facts"] = missingFacts
		metadata["execution_cutover_scope_binding_status"] = metadataString(proof.Metadata, "execution_cutover_scope_binding_status")
		metadata["execution_cutover_scope_binding_blockers"] = executionCutoverProofMetadataBindingBlockers(proof.Metadata)
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["task_loop_run_forwarded_by_command"] = proof.TaskLoopRunForwardedByCommand
		metadata["commands_run"] = proof.CommandsRun
		metadata["legacy_progress_written"] = proof.LegacyProgressWritten
		metadata["legacy_logs_written"] = proof.LegacyLogsWritten
		metadata["legacy_checkpoint_written"] = proof.LegacyCheckpointWritten
		metadata["areamatrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		if len(missingFacts) == 0 &&
			proof.Project.Key == record.Key &&
			executionCutoverProofCompletesAudit(*proof) {
			status = "pass"
			message = "rollback-to-read-only-shim proof facts are recorded and safety facts remain closed"
		}
	}
	r.add("rollback_to_read_only_shim", "rollback", status, message, []string{"fail closed proof", "rollback to read_only_shim proof"}, "areaflow completion execution-cutover-proof record "+record.Key+" --json", metadata)
}

func (r *ExecutionForwardingV1Readiness) add(key string, category string, status string, message string, requiredEvidence []string, nextCommand string, metadata map[string]any) {
	r.Items = append(r.Items, ExecutionForwardingV1ReadinessItem{
		Key:              key,
		Category:         category,
		Status:           status,
		Message:          message,
		RequiredEvidence: append([]string{}, requiredEvidence...),
		NextCommand:      nextCommand,
		Metadata:         metadata,
	})
	r.Status = combineExecutionCutoverStatus(r.Status, status)
}

func (r *ExecutionForwardingV1Readiness) addCommandEvidence(key string, category string, counts map[string]int, required []string, message string, nextCommand string) {
	missing := missingCommandEvidence(counts, required)
	status := "pass"
	if len(missing) > 0 {
		status = "blocked"
	}
	r.add(key, category, status, message, required, nextCommand, map[string]any{
		"required_command_types": required,
		"missing_command_types":  missing,
		"command_counts":         commandEvidenceSubset(counts, required),
	})
}

func statusForExecutionForwardingShim(shim ShimReadiness) string {
	if shim.Status == "pass" {
		return "pass"
	}
	return "blocked"
}
