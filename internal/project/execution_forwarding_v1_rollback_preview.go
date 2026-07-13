package project

import (
	"context"
	"time"
)

type ExecutionForwardingV1RollbackPreviewOptions struct {
	GeneratedAt time.Time
}

type ExecutionForwardingV1RollbackPreview struct {
	Project            Record
	Status             string
	Mode               string
	ApplyPreview       ExecutionForwardingV1ApplyPreview
	Items              []ExecutionForwardingV1RollbackPreviewItem
	RollbackTarget     string
	FailClosedSteps    []string
	ReopenConditions   []string
	RequiredProofFacts []string
	RequiredEvidence   []string
	ForbiddenActions   []string
	RollbackApplyOpen  bool
	SafetyFacts        map[string]bool
	GeneratedAt        time.Time
}

type ExecutionForwardingV1RollbackPreviewItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

func (s Store) ExecutionForwardingV1RollbackPreview(ctx context.Context, record Record, options ExecutionForwardingV1RollbackPreviewOptions) (ExecutionForwardingV1RollbackPreview, error) {
	options = normalizeExecutionForwardingV1RollbackPreviewOptions(options)
	applyPreview, err := s.ExecutionForwardingV1ApplyPreview(ctx, record, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: options.GeneratedAt,
	})
	if err != nil {
		return ExecutionForwardingV1RollbackPreview{}, err
	}
	rollbackProof, err := s.LatestExecutionCutoverProofForProject(ctx, record)
	if err != nil {
		return ExecutionForwardingV1RollbackPreview{}, err
	}
	return BuildExecutionForwardingV1RollbackPreviewWithProof(applyPreview, options, rollbackProof), nil
}

func normalizeExecutionForwardingV1RollbackPreviewOptions(options ExecutionForwardingV1RollbackPreviewOptions) ExecutionForwardingV1RollbackPreviewOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildExecutionForwardingV1RollbackPreview(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1RollbackPreviewOptions) ExecutionForwardingV1RollbackPreview {
	return buildExecutionForwardingV1RollbackPreview(applyPreview, options, nil)
}

func BuildExecutionForwardingV1RollbackPreviewWithProof(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1RollbackPreviewOptions, rollbackProof ExecutionCutoverProof) ExecutionForwardingV1RollbackPreview {
	if rollbackProof.EventID == 0 && rollbackProof.ProofStatus == "" {
		return buildExecutionForwardingV1RollbackPreview(applyPreview, options, nil)
	}
	return buildExecutionForwardingV1RollbackPreview(applyPreview, options, &rollbackProof)
}

func buildExecutionForwardingV1RollbackPreview(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1RollbackPreviewOptions, rollbackProof *ExecutionCutoverProof) ExecutionForwardingV1RollbackPreview {
	options = normalizeExecutionForwardingV1RollbackPreviewOptions(options)
	requiredProofFacts := executionForwardingV1RollbackProofFacts()
	requiredEvidence := []string{
		"areaflow project execution-forwarding-v1-apply-preview " + applyPreview.Project.Key + " --json",
		"read-only shim status after rollback",
		"legacy non-write proof after rollback",
		"protected path proof after rollback",
		"audit history for any suspended forwarding command",
	}
	preview := ExecutionForwardingV1RollbackPreview{
		Project:        applyPreview.Project,
		Status:         "blocked",
		Mode:           "read_only_execution_forwarding_v1_rollback_preview",
		ApplyPreview:   applyPreview,
		RollbackTarget: "read_only_shim",
		FailClosedSteps: []string{
			"disable or withhold project.execution_forwarding_v1.apply command",
			"keep ./task-loop run blocked or read-only according to current shim lifecycle",
			"preserve AreaFlow command/run/task/attempt/artifact/audit history as immutable evidence",
			"verify legacy progress, logs and checkpoint paths were not written",
			"record protected path proof before any future reopening review",
		},
		ReopenConditions: []string{
			"read_only_shim pass",
			"explicit R3 execution forwarding approval",
			"legacy non-write proof pass",
			"rollback proof pass",
			"protected path proof clean or explicitly authorized",
			"focused forwarding v1 smoke pass in approved scope",
		},
		RequiredProofFacts: requiredProofFacts,
		RequiredEvidence:   requiredEvidence,
		ForbiddenActions: []string{
			"create_rollback_command",
			"start_legacy_task_loop_runner",
			"forward_task_loop_run",
			"write_legacy_progress_json",
			"write_legacy_logs",
			"write_legacy_checkpoint",
			"write_areamatrix_source",
			"write_areamatrix_execution_directory",
			"delete_forwarding_history",
			"generated_retained_write",
			"repair_apply",
			"checkpoint_apply",
			"engine_execution",
			"secret_resolve",
			"network_api_integration",
			"publish_apply",
			"restore_apply",
		},
		RollbackApplyOpen: false,
		SafetyFacts: map[string]bool{
			"read_only_preview":                  true,
			"rollback_apply_open":                false,
			"apply_open":                         false,
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
		GeneratedAt: options.GeneratedAt,
	}
	preview.addItem(executionForwardingV1RollbackPreviewApplyItem(applyPreview))
	preview.addItem(executionForwardingV1RollbackPreviewFailClosedItem(applyPreview))
	preview.addItem(executionForwardingV1RollbackPreviewProofItem(applyPreview, requiredProofFacts, rollbackProof))
	preview.addItem(executionForwardingV1RollbackPreviewReopenItem(applyPreview))
	preview.addItem(executionForwardingV1RollbackPreviewSafetyItem())
	return preview
}

func (p *ExecutionForwardingV1RollbackPreview) addItem(item ExecutionForwardingV1RollbackPreviewItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	p.Items = append(p.Items, item)
	if item.Status == "fail" {
		p.Status = "blocked"
	}
}

func executionForwardingV1RollbackProofFacts() []string {
	return []string{
		"rollback_target_read_only_shim_confirmed",
		"forwarding_v1_command_disabled_or_absent",
		"task_loop_run_forwarding_disabled",
		"legacy_task_loop_runner_not_started_after_rollback",
		"legacy_progress_json_not_written_after_rollback",
		"legacy_logs_not_written_after_rollback",
		"legacy_checkpoint_not_written_after_rollback",
		"areaflow_forwarded_state_preserved_as_audit_history",
		"protected_path_proof_clean_after_rollback_recorded",
	}
}

func executionForwardingV1RollbackPreviewApplyItem(applyPreview ExecutionForwardingV1ApplyPreview) ExecutionForwardingV1RollbackPreviewItem {
	return ExecutionForwardingV1RollbackPreviewItem{
		Key:              "rollback_v1:apply_preview",
		Category:         "precondition",
		Status:           "pass",
		Message:          "rollback preview consumes the current apply preview without opening apply",
		Owner:            "execution_owner",
		RequiredEvidence: []string{"areaflow project execution-forwarding-v1-apply-preview " + applyPreview.Project.Key + " --json"},
		NextCommand:      "areaflow project execution-forwarding-v1-apply-preview " + applyPreview.Project.Key + " --json",
		Metadata: map[string]any{
			"apply_preview_status": applyPreview.Status,
			"apply_open":           false,
		},
	}
}

func executionForwardingV1RollbackPreviewFailClosedItem(applyPreview ExecutionForwardingV1ApplyPreview) ExecutionForwardingV1RollbackPreviewItem {
	status := "blocked"
	message := "rollback remains a preview until fail-closed and legacy non-write proof are recorded"
	legacyProofStatus := executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "legacy_non_write_proof")
	if !applyPreview.ApplyOpen &&
		!applyPreview.SafetyFacts["forwarding_v1_apply_open"] &&
		!applyPreview.SafetyFacts["task_loop_run_forwarded"] &&
		legacyProofStatus == "pass" {
		status = "pass"
		message = "apply is closed, task-loop forwarding is disabled, and legacy non-write proof is clean"
	}
	return ExecutionForwardingV1RollbackPreviewItem{
		Key:      "rollback_v1:fail_closed",
		Category: "rollback",
		Status:   status,
		Message:  message,
		Owner:    "execution_owner",
		RequiredEvidence: []string{
			"forwarding command disabled or absent",
			"task-loop run remains blocked",
			"legacy non-write proof",
		},
		NextCommand: "areaflow completion execution-cutover-proof record " + applyPreview.Project.Key + " --json",
		Metadata: map[string]any{
			"rollback_target":            "read_only_shim",
			"rollback_apply_open":        false,
			"apply_open":                 applyPreview.ApplyOpen,
			"forwarding_v1_apply_open":   applyPreview.SafetyFacts["forwarding_v1_apply_open"],
			"task_loop_run_forwarded":    applyPreview.SafetyFacts["task_loop_run_forwarded"],
			"legacy_non_write_proof":     legacyProofStatus,
			"fail_closed_preview_proven": status == "pass",
		},
	}
}

func executionForwardingV1ReadinessItemStatus(readiness ExecutionForwardingV1Readiness, key string) string {
	for _, item := range readiness.Items {
		if item.Key == key {
			return item.Status
		}
	}
	return ""
}

func executionForwardingV1RollbackPreviewProofItem(applyPreview ExecutionForwardingV1ApplyPreview, requiredProofFacts []string, proof *ExecutionCutoverProof) ExecutionForwardingV1RollbackPreviewItem {
	status := "blocked"
	message := "rollback proof facts must show legacy paths stayed untouched and AreaFlow history was preserved"
	metadata := map[string]any{
		"required_proof_facts": requiredProofFacts,
		"rollback_apply_open":  false,
		"proof_present":        false,
	}
	if proof != nil {
		missingFacts := missingStringFacts(proof.Facts, requiredProofFacts)
		metadata["proof_present"] = true
		metadata["proof_status"] = proof.ProofStatus
		metadata["proof_decision"] = proof.Decision
		metadata["proof_event_id"] = proof.EventID
		metadata["proof_audit_event_id"] = proof.AuditEventID
		metadata["proof_project_key"] = proof.Project.Key
		metadata["proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["missing_proof_facts"] = missingFacts
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["task_loop_run_forwarded_by_command"] = proof.TaskLoopRunForwardedByCommand
		metadata["commands_run"] = proof.CommandsRun
		metadata["legacy_progress_written"] = proof.LegacyProgressWritten
		metadata["legacy_logs_written"] = proof.LegacyLogsWritten
		metadata["legacy_checkpoint_written"] = proof.LegacyCheckpointWritten
		metadata["areamatrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		if len(missingFacts) == 0 &&
			proof.Project.Key == applyPreview.Project.Key &&
			executionCutoverProofCompletesAudit(*proof) {
			status = "pass"
			message = "rollback proof facts are recorded and safety facts remain closed"
		}
	}
	return ExecutionForwardingV1RollbackPreviewItem{
		Key:              "rollback_v1:proof_facts",
		Category:         "proof",
		Status:           status,
		Message:          message,
		Owner:            "execution_owner",
		RequiredEvidence: append([]string{}, requiredProofFacts...),
		NextCommand:      "areaflow completion protected-path-proof record " + applyPreview.Project.Key + " --status clean --summary <text> --evidence-uri <uri> --json",
		Metadata:         metadata,
	}
}

func missingStringFacts(facts []string, required []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, fact := range required {
		if !present[fact] {
			missing = append(missing, fact)
		}
	}
	return missing
}

func executionForwardingV1RollbackPreviewReopenItem(applyPreview ExecutionForwardingV1ApplyPreview) ExecutionForwardingV1RollbackPreviewItem {
	return ExecutionForwardingV1RollbackPreviewItem{
		Key:      "rollback_v1:reopen_conditions",
		Category: "reopen",
		Status:   "blocked",
		Message:  "forwarding can only be reconsidered after approval, smoke, rollback and protected path evidence are refreshed",
		Owner:    "project_owner",
		RequiredEvidence: []string{
			"explicit R3 approval",
			"focused forwarding v1 smoke",
			"rollback proof",
			"protected path proof",
		},
		NextCommand: "areaflow project execution-forwarding-v1-readiness " + applyPreview.Project.Key + " --json",
		Metadata: map[string]any{
			"reopen_requires_new_review": true,
			"rollback_apply_open":        false,
		},
	}
}

func executionForwardingV1RollbackPreviewSafetyItem() ExecutionForwardingV1RollbackPreviewItem {
	return ExecutionForwardingV1RollbackPreviewItem{
		Key:      "rollback_v1:read_only_preview",
		Category: "safety",
		Status:   "pass",
		Message:  "rollback preview did not create commands, runs, tasks, leases, attempts, artifacts or project writes",
		Owner:    "platform_owner",
		Metadata: map[string]any{
			"rollback_apply_open": false,
			"apply_open":          false,
		},
	}
}
