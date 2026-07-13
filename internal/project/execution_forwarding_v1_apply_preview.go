package project

import (
	"context"
	"time"
)

type ExecutionForwardingV1ApplyPreviewOptions struct {
	GeneratedAt time.Time
}

type ExecutionForwardingV1ApplyPreview struct {
	Project              Record
	Status               string
	Mode                 string
	Readiness            ExecutionForwardingV1Readiness
	Items                []ExecutionForwardingV1ApplyPreviewItem
	AllowedTaskTypes     []string
	ForwardingTargets    []ExecutionForwardingV1ForwardingTarget
	BlockedTargets       []ExecutionForwardingV1BlockedTarget
	RequiredCapabilities []string
	ApplyPacketFields    []string
	FailClosedFields     []string
	RequiredProofFacts   []string
	RequiredEvidence     []string
	ForbiddenActions     []string
	ApprovalRequired     bool
	ApprovalStatus       string
	ApplyOpen            bool
	RollbackTarget       string
	SafetyFacts          map[string]bool
	GeneratedAt          time.Time
}

type ExecutionForwardingV1ForwardingTarget struct {
	TaskType              string
	TargetCommandType     string
	TargetStatus          string
	RequiredCapabilities  []string
	RequiredPacketFields  []string
	CreatesCommandRequest bool
	CreatesRun            bool
	CreatesRunTask        bool
	CreatesRunAttempt     bool
	CreatesArtifact       bool
	CreatesAuditEvent     bool
	ProjectWriteAllowed   bool
	ExecutionWriteAllowed bool
	LegacyFallbackAllowed bool
	FailureMode           string
}

type ExecutionForwardingV1BlockedTarget struct {
	TaskType        string
	ForbiddenAction string
	Reason          string
	FailureMode     string
	SafetyFacts     map[string]bool
}

type ExecutionForwardingV1ApplyPreviewItem struct {
	Key              string
	Category         string
	Status           string
	ApprovalStatus   string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

func (s Store) ExecutionForwardingV1ApplyPreview(ctx context.Context, record Record, options ExecutionForwardingV1ApplyPreviewOptions) (ExecutionForwardingV1ApplyPreview, error) {
	options = normalizeExecutionForwardingV1ApplyPreviewOptions(options)
	readiness, err := s.ExecutionForwardingV1Readiness(ctx, record, ExecutionForwardingV1ReadinessOptions{})
	if err != nil {
		return ExecutionForwardingV1ApplyPreview{}, err
	}
	return BuildExecutionForwardingV1ApplyPreview(readiness, options), nil
}

func normalizeExecutionForwardingV1ApplyPreviewOptions(options ExecutionForwardingV1ApplyPreviewOptions) ExecutionForwardingV1ApplyPreviewOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildExecutionForwardingV1ApplyPreview(readiness ExecutionForwardingV1Readiness, options ExecutionForwardingV1ApplyPreviewOptions) ExecutionForwardingV1ApplyPreview {
	options = normalizeExecutionForwardingV1ApplyPreviewOptions(options)
	requiredProofFacts := []string{
		"explicit_execution_forwarding_v1_approval_recorded",
		"execution_forwarding_v1_command_response_recorded",
		"execution_forwarding_v1_event_and_audit_recorded",
		"read_only_shim_landed_and_verified",
		"legacy_task_loop_runner_not_started",
		"legacy_progress_json_not_written",
		"legacy_logs_not_written",
		"legacy_checkpoint_not_written",
		"areaflow_run_task_attempt_artifact_audit_owns_forwarded_state",
		"forwarded_task_type_policy_enforced",
		"blocked_task_types_fail_closed",
		"rollback_to_read_only_shim_verified",
		"protected_path_proof_clean_recorded",
	}
	requiredEvidence := []string{
		"areaflow project execution-forwarding-v1-readiness " + readiness.Project.Key + " --json",
		"explicit R3 execution forwarding v1 approval",
		"read-only shim landed after explicit cross-repo approval",
		"legacy non-write protected path proof",
		"rollback-to-read-only-shim proof",
		"focused forwarding v1 smoke in fixture or approved managed scope",
	}
	preview := ExecutionForwardingV1ApplyPreview{
		Project:              readiness.Project,
		Status:               "pass",
		Mode:                 "read_only_execution_forwarding_v1_apply_preview",
		Readiness:            readiness,
		AllowedTaskTypes:     append([]string{}, readiness.AllowedTaskTypes...),
		ForwardingTargets:    executionForwardingV1ForwardingTargets(),
		BlockedTargets:       executionForwardingV1BlockedTargets(),
		RequiredCapabilities: []string{"read_project", "write_artifacts", "manage_workers"},
		ApplyPacketFields: []string{
			"command_type",
			"project_key",
			"forwarded_task_type",
			"target_command_type",
			"allowed_task_types",
			"approval_id",
			"approval_scope",
			"readiness_snapshot_hash",
			"expected_shim_lifecycle_state",
			"legacy_non_write_proof_id",
			"rollback_plan_id",
			"protected_path_fingerprint_id",
			"idempotency_key",
			"audit_correlation_id",
			"failure_mode",
		},
		FailClosedFields: []string{
			"status",
			"decision",
			"failure_mode",
			"blocked_task_type",
			"forbidden_action",
			"legacy_task_loop_started",
			"legacy_progress_written",
			"legacy_logs_written",
			"legacy_checkpoint_written",
			"project_write_attempted",
			"execution_write_attempted",
			"engine_call_attempted",
			"commands_run",
			"secrets_resolved",
			"network_used",
			"event_id",
			"audit_event_id",
		},
		RequiredProofFacts: requiredProofFacts,
		RequiredEvidence:   requiredEvidence,
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
		ApprovalRequired: true,
		ApprovalStatus:   "needs_approval",
		ApplyOpen:        false,
		RollbackTarget:   "read_only_shim",
		SafetyFacts: map[string]bool{
			"read_only_preview":                  true,
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
	preview.addItem(executionForwardingV1ApplyPreviewReadinessItem(readiness))
	preview.addItem(executionForwardingV1ApplyPreviewApprovalItem(readiness, requiredEvidence))
	preview.addItem(executionForwardingV1ApplyPreviewCommandItem(readiness))
	preview.addItem(executionForwardingV1ApplyPreviewTargetPolicyItem(readiness, preview.ForwardingTargets, preview.BlockedTargets))
	preview.addItem(executionForwardingV1ApplyPreviewProofItem(readiness, requiredProofFacts))
	preview.addItem(executionForwardingV1ApplyPreviewRollbackItem(readiness))
	preview.addItem(executionForwardingV1ApplyPreviewSafetyItem())
	return preview
}

func executionForwardingV1ForwardingTargets() []ExecutionForwardingV1ForwardingTarget {
	requiredPacketFields := []string{
		"project_key",
		"forwarded_task_type",
		"target_command_type",
		"approval_id",
		"readiness_snapshot_hash",
		"idempotency_key",
		"audit_correlation_id",
	}
	return []ExecutionForwardingV1ForwardingTarget{
		{
			TaskType:              "read_only_verify",
			TargetCommandType:     "run.read_only_verify_queue",
			TargetStatus:          "available_scoped",
			RequiredCapabilities:  []string{"read_project", "manage_workers"},
			RequiredPacketFields:  append([]string{}, requiredPacketFields...),
			CreatesCommandRequest: true,
			CreatesRun:            true,
			CreatesRunTask:        true,
			CreatesRunAttempt:     false,
			CreatesArtifact:       false,
			CreatesAuditEvent:     true,
			ProjectWriteAllowed:   false,
			ExecutionWriteAllowed: false,
			LegacyFallbackAllowed: false,
			FailureMode:           "fail_closed",
		},
		{
			TaskType:              "doctor_readiness",
			TargetCommandType:     "planned:project.doctor_readiness.forward",
			TargetStatus:          "preview_only",
			RequiredCapabilities:  []string{"read_project"},
			RequiredPacketFields:  append([]string{}, requiredPacketFields...),
			CreatesCommandRequest: true,
			CreatesRun:            false,
			CreatesRunTask:        false,
			CreatesRunAttempt:     false,
			CreatesArtifact:       false,
			CreatesAuditEvent:     true,
			ProjectWriteAllowed:   false,
			ExecutionWriteAllowed: false,
			LegacyFallbackAllowed: false,
			FailureMode:           "fail_closed",
		},
		{
			TaskType:              "artifact_evidence",
			TargetCommandType:     "run.approved_artifact_write_queue",
			TargetStatus:          "available_scoped",
			RequiredCapabilities:  []string{"write_artifacts", "manage_workers"},
			RequiredPacketFields:  append([]string{}, requiredPacketFields...),
			CreatesCommandRequest: true,
			CreatesRun:            true,
			CreatesRunTask:        true,
			CreatesRunAttempt:     false,
			CreatesArtifact:       false,
			CreatesAuditEvent:     true,
			ProjectWriteAllowed:   false,
			ExecutionWriteAllowed: false,
			LegacyFallbackAllowed: false,
			FailureMode:           "fail_closed",
		},
		{
			TaskType:              "status_projection_validation",
			TargetCommandType:     "planned:project.status_projection.validate",
			TargetStatus:          "preview_only",
			RequiredCapabilities:  []string{"read_project"},
			RequiredPacketFields:  append([]string{}, requiredPacketFields...),
			CreatesCommandRequest: true,
			CreatesRun:            false,
			CreatesRunTask:        false,
			CreatesRunAttempt:     false,
			CreatesArtifact:       false,
			CreatesAuditEvent:     true,
			ProjectWriteAllowed:   false,
			ExecutionWriteAllowed: false,
			LegacyFallbackAllowed: false,
			FailureMode:           "fail_closed",
		},
		{
			TaskType:              "release_readiness_check",
			TargetCommandType:     "planned:project.release_readiness.check",
			TargetStatus:          "preview_only",
			RequiredCapabilities:  []string{"read_project"},
			RequiredPacketFields:  append([]string{}, requiredPacketFields...),
			CreatesCommandRequest: true,
			CreatesRun:            false,
			CreatesRunTask:        false,
			CreatesRunAttempt:     false,
			CreatesArtifact:       false,
			CreatesAuditEvent:     true,
			ProjectWriteAllowed:   false,
			ExecutionWriteAllowed: false,
			LegacyFallbackAllowed: false,
			FailureMode:           "fail_closed",
		},
	}
}

func executionForwardingV1BlockedTargets() []ExecutionForwardingV1BlockedTarget {
	blocked := []struct {
		taskType        string
		forbiddenAction string
		reason          string
	}{
		{"copy_ready_source_write", "write_areamatrix_source", "source write is outside Execution Forwarding v1"},
		{"generated_retained_write", "generated_retained_write", "retained generated writes need a later retained apply gate"},
		{"repair_apply", "repair_apply", "repair apply needs verify failure evidence and a separate approval"},
		{"checkpoint_apply", "checkpoint_apply", "checkpoint apply is a separate high-risk command"},
		{"engine_execution", "engine_execution", "engine execution stays closed for forwarding v1"},
		{"secret_resolve", "secret_resolve", "secret resolution is a v1.x high-risk capability"},
		{"network_api_integration", "network_api_integration", "network/API integration stays closed"},
		{"publish_apply", "publish_apply", "publish apply is not part of execution forwarding"},
		{"restore_apply", "restore_apply", "restore apply is a separate R4 capability"},
	}
	targets := make([]ExecutionForwardingV1BlockedTarget, 0, len(blocked))
	for _, item := range blocked {
		targets = append(targets, ExecutionForwardingV1BlockedTarget{
			TaskType:        item.taskType,
			ForbiddenAction: item.forbiddenAction,
			Reason:          item.reason,
			FailureMode:     "fail_closed",
			SafetyFacts: map[string]bool{
				"legacy_task_loop_started":  false,
				"legacy_progress_written":   false,
				"legacy_logs_written":       false,
				"legacy_checkpoint_written": false,
				"project_write_attempted":   false,
				"execution_write_attempted": false,
				"engine_call_attempted":     false,
				"commands_run":              false,
				"secrets_resolved":          false,
				"network_used":              false,
			},
		})
	}
	return targets
}

func (p *ExecutionForwardingV1ApplyPreview) addItem(item ExecutionForwardingV1ApplyPreviewItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	p.Items = append(p.Items, item)
	if item.Status == "blocked" || item.Status == "fail" || item.Status == "needs_approval" {
		p.Status = "blocked"
	}
}

func executionForwardingV1ApplyPreviewReadinessItem(readiness ExecutionForwardingV1Readiness) ExecutionForwardingV1ApplyPreviewItem {
	status := "pass"
	message := "execution forwarding v1 readiness is ready for apply preview review"
	if readiness.Status != "pass" {
		status = "blocked"
		message = "execution forwarding v1 readiness must pass before apply can be considered"
	}
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:              "forwarding_v1:readiness",
		Category:         "readiness",
		Status:           status,
		Message:          message,
		Owner:            "execution_owner",
		RequiredEvidence: []string{"areaflow project execution-forwarding-v1-readiness " + readiness.Project.Key + " --json"},
		NextCommand:      "areaflow project execution-forwarding-v1-readiness " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"readiness_status": readiness.Status,
			"apply_open":       false,
		},
	}
}

func executionForwardingV1ApplyPreviewApprovalItem(readiness ExecutionForwardingV1Readiness, requiredEvidence []string) ExecutionForwardingV1ApplyPreviewItem {
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:              "forwarding_v1:explicit_approval",
		Category:         "approval",
		Status:           "blocked",
		ApprovalStatus:   "needs_approval",
		Message:          "explicit R3 execution forwarding approval is required before apply can open",
		Owner:            "project_owner",
		RequiredEvidence: append([]string{}, requiredEvidence...),
		NextCommand:      "areaflow project execution-forwarding-v1-apply-preview " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"risk_level":     "R3 execution",
			"approval_scope": "execution_forwarding_v1_read_only_evidence_only",
			"apply_open":     false,
		},
	}
}

func executionForwardingV1ApplyPreviewCommandItem(readiness ExecutionForwardingV1Readiness) ExecutionForwardingV1ApplyPreviewItem {
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:      "forwarding_v1:command_api_contract",
		Category: "command_api",
		Status:   "pass",
		Message:  "apply is a protected Command API write with idempotency, expected readiness hash and audit response",
		Owner:    "execution_owner",
		RequiredEvidence: []string{
			"command_type=project.execution_forwarding_v1.apply",
			"idempotency key",
			"expected readiness snapshot hash",
			"approval id",
			"event id",
			"audit event id",
		},
		NextCommand: "areaflow project execution-forwarding-v1-apply " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"command_type": "project.execution_forwarding_v1.apply",
			"apply_open":   false,
		},
	}
}

func executionForwardingV1ApplyPreviewTargetPolicyItem(readiness ExecutionForwardingV1Readiness, targets []ExecutionForwardingV1ForwardingTarget, blocked []ExecutionForwardingV1BlockedTarget) ExecutionForwardingV1ApplyPreviewItem {
	allowedTaskTypes := make([]string, 0, len(targets))
	targetCommandTypes := make([]string, 0, len(targets))
	for _, target := range targets {
		allowedTaskTypes = append(allowedTaskTypes, target.TaskType)
		targetCommandTypes = append(targetCommandTypes, target.TargetCommandType)
	}
	blockedTaskTypes := make([]string, 0, len(blocked))
	for _, target := range blocked {
		blockedTaskTypes = append(blockedTaskTypes, target.TaskType)
	}
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:      "forwarding_v1:target_policy",
		Category: "command_api",
		Status:   "pass",
		Message:  "forwarding v1 target policy is defined as read-only/evidence only and forbidden targets fail closed",
		Owner:    "execution_owner",
		RequiredEvidence: []string{
			"forwarding target matrix",
			"blocked target matrix",
			"fail-closed response fields",
		},
		NextCommand: "areaflow project execution-forwarding-v1-apply-preview " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"allowed_task_types":      allowedTaskTypes,
			"target_command_types":    targetCommandTypes,
			"blocked_task_types":      blockedTaskTypes,
			"legacy_fallback_open":    false,
			"project_write_allowed":   false,
			"execution_write_allowed": false,
			"failure_mode":            "fail_closed",
			"apply_open":              false,
		},
	}
}

func executionForwardingV1ApplyPreviewProofItem(readiness ExecutionForwardingV1Readiness, requiredProofFacts []string) ExecutionForwardingV1ApplyPreviewItem {
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:              "forwarding_v1:proof_facts",
		Category:         "proof",
		Status:           "blocked",
		Message:          "proof facts must show AreaFlow owns forwarded state and legacy runner/progress/log/checkpoint stay untouched",
		Owner:            "execution_owner",
		RequiredEvidence: append([]string{}, requiredProofFacts...),
		NextCommand:      "areaflow completion execution-cutover-proof record " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"required_proof_facts": requiredProofFacts,
			"apply_open":           false,
		},
	}
}

func executionForwardingV1ApplyPreviewRollbackItem(readiness ExecutionForwardingV1Readiness) ExecutionForwardingV1ApplyPreviewItem {
	rollbackStatus := executionForwardingV1ReadinessItemStatus(readiness, "rollback_to_read_only_shim")
	status := "blocked"
	message := "forwarding v1 must fail closed and roll back to read_only_shim before apply can open"
	if rollbackStatus == "pass" {
		status = "pass"
		message = "rollback-to-read-only-shim proof is recorded and safety facts remain closed"
	}
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:      "forwarding_v1:rollback",
		Category: "rollback",
		Status:   status,
		Message:  message,
		Owner:    "execution_owner",
		RequiredEvidence: []string{
			"fail closed proof",
			"rollback to read_only_shim proof",
			"post-rollback legacy non-write proof",
		},
		NextCommand: "future: areaflow project execution-forwarding-v1-rollback " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"rollback_target":           "read_only_shim",
			"rollback_readiness_status": rollbackStatus,
			"apply_open":                false,
		},
	}
}

func executionForwardingV1ApplyPreviewSafetyItem() ExecutionForwardingV1ApplyPreviewItem {
	return ExecutionForwardingV1ApplyPreviewItem{
		Key:      "forwarding_v1:read_only_preview",
		Category: "safety",
		Status:   "pass",
		Message:  "apply preview did not create commands, runs, tasks, leases, attempts, artifacts or project writes",
		Owner:    "platform_owner",
		Metadata: map[string]any{
			"apply_open":                    false,
			"task_loop_run_forwarded":       false,
			"project_write_attempted":       false,
			"execution_write_attempted":     false,
			"area_flow_command_created":     false,
			"area_flow_run_created":         false,
			"worker_scheduled":              false,
			"engine_call_attempted":         false,
			"commands_run":                  false,
			"secrets_resolved":              false,
			"network_used":                  false,
			"legacy_task_loop_started":      false,
			"legacy_progress_written":       false,
			"legacy_logs_written":           false,
			"legacy_checkpoint_written":     false,
			"generated_retained_write_open": false,
			"repair_apply_open":             false,
			"checkpoint_apply_open":         false,
			"publish_apply_open":            false,
			"restore_apply_open":            false,
		},
	}
}
