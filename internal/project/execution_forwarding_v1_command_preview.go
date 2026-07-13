package project

import (
	"context"
	"strings"
	"time"
)

type ExecutionForwardingV1CommandPreviewOptions struct {
	TaskType    string
	GeneratedAt time.Time
}

type ExecutionForwardingV1CommandPreview struct {
	Project                                Record
	Status                                 string
	Mode                                   string
	Decision                               string
	Message                                string
	TaskType                               string
	TargetCommandType                      string
	TargetStatus                           string
	FailureMode                            string
	AllowedTaskType                        bool
	BlockedTaskType                        bool
	ApplyOpen                              bool
	WouldCreateCommandRequestAfterApproval bool
	WouldCreateRunAfterApproval            bool
	WouldCreateRunTaskAfterApproval        bool
	WouldCreateRunAttemptAfterApproval     bool
	WouldCreateArtifactAfterApproval       bool
	WouldCreateAuditEventAfterApproval     bool
	ProjectWriteAllowed                    bool
	ExecutionWriteAllowed                  bool
	LegacyFallbackAllowed                  bool
	RequiredPacketFields                   []string
	RequiredCapabilities                   []string
	FailClosedFields                       []string
	BlockedBy                              []string
	AllowedTaskTypes                       []string
	ForbiddenActions                       []string
	SafetyFacts                            map[string]bool
	GeneratedAt                            time.Time
}

func (s Store) ExecutionForwardingV1CommandPreview(ctx context.Context, record Record, options ExecutionForwardingV1CommandPreviewOptions) (ExecutionForwardingV1CommandPreview, error) {
	options = normalizeExecutionForwardingV1CommandPreviewOptions(options)
	applyPreview, err := s.ExecutionForwardingV1ApplyPreview(ctx, record, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: options.GeneratedAt,
	})
	if err != nil {
		return ExecutionForwardingV1CommandPreview{}, err
	}
	return BuildExecutionForwardingV1CommandPreview(applyPreview, options), nil
}

func normalizeExecutionForwardingV1CommandPreviewOptions(options ExecutionForwardingV1CommandPreviewOptions) ExecutionForwardingV1CommandPreviewOptions {
	options.TaskType = strings.TrimSpace(options.TaskType)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildExecutionForwardingV1CommandPreview(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1CommandPreviewOptions) ExecutionForwardingV1CommandPreview {
	options = normalizeExecutionForwardingV1CommandPreviewOptions(options)
	preview := ExecutionForwardingV1CommandPreview{
		Project:          applyPreview.Project,
		Status:           "blocked",
		Mode:             "read_only_execution_forwarding_v1_command_preview",
		Decision:         "unknown_task_type_fail_closed",
		Message:          "forwarding v1 command preview fails closed for unknown task type",
		TaskType:         options.TaskType,
		FailureMode:      "fail_closed",
		ApplyOpen:        false,
		FailClosedFields: append([]string{}, applyPreview.FailClosedFields...),
		BlockedBy: []string{
			"execution_forwarding_v1_apply_open=false",
			"explicit_execution_forwarding_v1_approval_missing",
		},
		AllowedTaskTypes: append([]string{}, applyPreview.AllowedTaskTypes...),
		ForbiddenActions: append([]string{}, applyPreview.ForbiddenActions...),
		SafetyFacts:      executionForwardingV1CommandPreviewSafetyFacts(),
		GeneratedAt:      options.GeneratedAt,
	}
	for _, target := range applyPreview.ForwardingTargets {
		if target.TaskType != options.TaskType {
			continue
		}
		preview.Decision = "would_forward_after_approval"
		preview.Message = "task type is in the forwarding v1 target matrix, but apply remains closed"
		preview.TargetCommandType = target.TargetCommandType
		preview.TargetStatus = target.TargetStatus
		preview.AllowedTaskType = true
		preview.WouldCreateCommandRequestAfterApproval = target.CreatesCommandRequest
		preview.WouldCreateRunAfterApproval = target.CreatesRun
		preview.WouldCreateRunTaskAfterApproval = target.CreatesRunTask
		preview.WouldCreateRunAttemptAfterApproval = target.CreatesRunAttempt
		preview.WouldCreateArtifactAfterApproval = target.CreatesArtifact
		preview.WouldCreateAuditEventAfterApproval = target.CreatesAuditEvent
		preview.ProjectWriteAllowed = target.ProjectWriteAllowed
		preview.ExecutionWriteAllowed = target.ExecutionWriteAllowed
		preview.LegacyFallbackAllowed = target.LegacyFallbackAllowed
		preview.RequiredPacketFields = append([]string{}, target.RequiredPacketFields...)
		preview.RequiredCapabilities = append([]string{}, target.RequiredCapabilities...)
		if target.TargetStatus == "preview_only" {
			preview.BlockedBy = append(preview.BlockedBy, "target_command_type_preview_only")
		}
		if applyPreview.Readiness.Status != "pass" {
			preview.BlockedBy = append(preview.BlockedBy, "execution_forwarding_v1_readiness_not_pass")
		}
		return preview
	}
	for _, target := range applyPreview.BlockedTargets {
		if target.TaskType != options.TaskType {
			continue
		}
		preview.Decision = "blocked_task_type_fail_closed"
		preview.Message = target.Reason
		preview.TargetCommandType = ""
		preview.TargetStatus = "blocked"
		preview.BlockedTaskType = true
		preview.BlockedBy = append(preview.BlockedBy, target.ForbiddenAction)
		return preview
	}
	preview.BlockedBy = append(preview.BlockedBy, "task_type_not_in_forwarding_v1_policy")
	return preview
}

func executionForwardingV1CommandPreviewSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_preview":                  true,
		"command_preview":                    true,
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
	}
}
