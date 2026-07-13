package project

import (
	"context"
	"strings"
	"time"
)

const executionForwardingV1ApplyPacketMode = "execution_forwarding_v1_apply_packet_preview_v1"

type ExecutionForwardingV1ApplyPacketPreviewOptions struct {
	ExplicitApproval           bool
	ApprovalID                 string
	ApprovalActor              string
	ApprovalReason             string
	LegacyNonWriteProofID      string
	RollbackPlanID             string
	ProtectedPathFingerprintID string
	IdempotencyKey             string
	AuditCorrelationID         string
	GeneratedAt                time.Time
}

type ExecutionForwardingV1ApplyPacketPreview struct {
	Project                                    Record
	Status                                     string
	Mode                                       string
	Decision                                   string
	Message                                    string
	ApplyPreview                               ExecutionForwardingV1ApplyPreview
	Gate                                       ExecutionForwardingV1ApplyGate
	Packet                                     ExecutionForwardingV1ApplyPacket
	ApplyGateCommand                           []string
	FutureApplyCommand                         []string
	RequiredHumanReview                        []string
	ForbiddenActions                           []string
	SafetyFacts                                map[string]bool
	WouldCreateCommandRequestAfterApplyCommand bool
	WouldCreateRunAfterApplyCommand            bool
	WouldCreateRunTaskAfterApplyCommand        bool
	WouldCreateAuditEventAfterApplyCommand     bool
	CommandRequestCreated                      bool
	AreaFlowRunCreated                         bool
	TaskLoopRunForwarded                       bool
	ProjectWriteAttempted                      bool
	ExecutionWriteAttempted                    bool
	EngineCallAttempted                        bool
	GeneratedAt                                time.Time
}

type ExecutionForwardingV1ApplyPacket struct {
	CommandType                string   `json:"command_type"`
	ProjectKey                 string   `json:"project_key"`
	AllowedTaskTypes           []string `json:"allowed_task_types"`
	TargetCommandTypes         []string `json:"target_command_types"`
	ApprovalID                 string   `json:"approval_id"`
	ApprovalScope              string   `json:"approval_scope"`
	ReadinessSnapshotHash      string   `json:"readiness_snapshot_hash"`
	ExpectedShimLifecycleState string   `json:"expected_shim_lifecycle_state"`
	LegacyNonWriteProofID      string   `json:"legacy_non_write_proof_id"`
	RollbackPlanID             string   `json:"rollback_plan_id"`
	ProtectedPathFingerprintID string   `json:"protected_path_fingerprint_id"`
	IdempotencyKey             string   `json:"idempotency_key"`
	AuditCorrelationID         string   `json:"audit_correlation_id"`
	FailureMode                string   `json:"failure_mode"`
	ExplicitApproval           bool     `json:"explicit_approval"`
	ApprovalActor              string   `json:"approval_actor"`
	ApprovalReason             string   `json:"approval_reason"`
}

func (s Store) ExecutionForwardingV1ApplyPacketPreview(ctx context.Context, record Record, options ExecutionForwardingV1ApplyPacketPreviewOptions) (ExecutionForwardingV1ApplyPacketPreview, error) {
	options = normalizeExecutionForwardingV1ApplyPacketPreviewOptions(options)
	applyPreview, err := s.ExecutionForwardingV1ApplyPreview(ctx, record, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: options.GeneratedAt,
	})
	if err != nil {
		return ExecutionForwardingV1ApplyPacketPreview{}, err
	}
	return BuildExecutionForwardingV1ApplyPacketPreview(applyPreview, options), nil
}

func BuildExecutionForwardingV1ApplyPacketPreview(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1ApplyPacketPreviewOptions) ExecutionForwardingV1ApplyPacketPreview {
	options = normalizeExecutionForwardingV1ApplyPacketPreviewOptions(options)
	packet := executionForwardingV1ApplyPacketFromPreview(applyPreview, options)
	gateOptions := executionForwardingV1ApplyGateOptionsFromPacket(packet, options.GeneratedAt)
	gate := BuildExecutionForwardingV1ApplyGate(applyPreview, gateOptions)
	preview := ExecutionForwardingV1ApplyPacketPreview{
		Project:            applyPreview.Project,
		Status:             "needs_approval",
		Mode:               executionForwardingV1ApplyPacketMode,
		Decision:           "needs_explicit_approval",
		Message:            "execution forwarding v1 apply packet is generated but still needs explicit approval and proof ids",
		ApplyPreview:       applyPreview,
		Gate:               gate,
		Packet:             packet,
		ApplyGateCommand:   executionForwardingV1ApplyGateCommand(packet),
		FutureApplyCommand: executionForwardingV1FutureApplyCommand(packet),
		RequiredHumanReview: []string{
			"review allowed task types and target command matrix",
			"review readiness snapshot hash",
			"review read-only shim and legacy non-write proof ids",
			"review rollback plan and protected path fingerprint ids",
			"confirm forbidden targets fail closed",
		},
		ForbiddenActions: append([]string{}, applyPreview.ForbiddenActions...),
		SafetyFacts:      executionForwardingV1ApplyPacketSafetyFacts(),
		GeneratedAt:      options.GeneratedAt,
	}
	if gate.ApplyCommandEligible {
		preview.Status = "ready"
		preview.Decision = "ready_for_future_apply_command"
		preview.Message = "execution forwarding v1 apply packet passed the read-only gate; future apply command remains separately controlled"
		preview.WouldCreateCommandRequestAfterApplyCommand = true
		preview.WouldCreateRunAfterApplyCommand = true
		preview.WouldCreateRunTaskAfterApplyCommand = true
		preview.WouldCreateAuditEventAfterApplyCommand = true
	} else if executionForwardingV1PacketReadinessBlocked(applyPreview.Readiness) {
		preview.Status = "blocked"
		preview.Decision = "readiness_blocked"
		preview.Message = "execution forwarding v1 apply packet is blocked until readiness proof gates pass"
	}
	return preview
}

func executionForwardingV1PacketReadinessBlocked(readiness ExecutionForwardingV1Readiness) bool {
	if readiness.Status != "pass" {
		return true
	}
	for _, key := range []string{"read_only_shim", "read_only_verify_evidence", "artifact_evidence", "legacy_non_write_proof", "rollback_to_read_only_shim"} {
		if executionForwardingV1ReadinessItemStatus(readiness, key) != "pass" {
			return true
		}
	}
	return false
}

func normalizeExecutionForwardingV1ApplyPacketPreviewOptions(options ExecutionForwardingV1ApplyPacketPreviewOptions) ExecutionForwardingV1ApplyPacketPreviewOptions {
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.ApprovalActor = strings.TrimSpace(options.ApprovalActor)
	options.ApprovalReason = strings.TrimSpace(options.ApprovalReason)
	options.LegacyNonWriteProofID = strings.TrimSpace(options.LegacyNonWriteProofID)
	options.RollbackPlanID = strings.TrimSpace(options.RollbackPlanID)
	options.ProtectedPathFingerprintID = strings.TrimSpace(options.ProtectedPathFingerprintID)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.AuditCorrelationID = strings.TrimSpace(options.AuditCorrelationID)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func executionForwardingV1ApplyPacketFromPreview(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1ApplyPacketPreviewOptions) ExecutionForwardingV1ApplyPacket {
	readinessHash := ExecutionForwardingV1ReadinessSnapshotHash(applyPreview)
	idempotencyKey := options.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = "execution-forwarding-v1:" + applyPreview.Project.Key + ":" + readinessHash[:16]
	}
	auditCorrelationID := options.AuditCorrelationID
	if auditCorrelationID == "" {
		auditCorrelationID = "audit:" + idempotencyKey
	}
	return ExecutionForwardingV1ApplyPacket{
		CommandType:                "project.execution_forwarding_v1.apply",
		ProjectKey:                 applyPreview.Project.Key,
		AllowedTaskTypes:           append([]string{}, applyPreview.AllowedTaskTypes...),
		TargetCommandTypes:         executionForwardingV1TargetCommandTypes(applyPreview.ForwardingTargets),
		ApprovalID:                 options.ApprovalID,
		ApprovalScope:              executionForwardingV1ApprovalScope(),
		ReadinessSnapshotHash:      readinessHash,
		ExpectedShimLifecycleState: "read_only_shim",
		LegacyNonWriteProofID:      options.LegacyNonWriteProofID,
		RollbackPlanID:             options.RollbackPlanID,
		ProtectedPathFingerprintID: options.ProtectedPathFingerprintID,
		IdempotencyKey:             idempotencyKey,
		AuditCorrelationID:         auditCorrelationID,
		FailureMode:                "fail_closed",
		ExplicitApproval:           options.ExplicitApproval,
		ApprovalActor:              options.ApprovalActor,
		ApprovalReason:             options.ApprovalReason,
	}
}

func executionForwardingV1ApplyGateOptionsFromPacket(packet ExecutionForwardingV1ApplyPacket, generatedAt time.Time) ExecutionForwardingV1ApplyGateOptions {
	return ExecutionForwardingV1ApplyGateOptions{
		AllowedTaskTypes:           append([]string{}, packet.AllowedTaskTypes...),
		ApprovalID:                 packet.ApprovalID,
		ApprovalScope:              packet.ApprovalScope,
		ReadinessSnapshotHash:      packet.ReadinessSnapshotHash,
		ExpectedShimLifecycleState: packet.ExpectedShimLifecycleState,
		LegacyNonWriteProofID:      packet.LegacyNonWriteProofID,
		RollbackPlanID:             packet.RollbackPlanID,
		ProtectedPathFingerprintID: packet.ProtectedPathFingerprintID,
		IdempotencyKey:             packet.IdempotencyKey,
		AuditCorrelationID:         packet.AuditCorrelationID,
		FailureMode:                packet.FailureMode,
		ExplicitApproval:           packet.ExplicitApproval,
		ApprovalActor:              packet.ApprovalActor,
		ApprovalReason:             packet.ApprovalReason,
		GeneratedAt:                generatedAt,
	}
}

func executionForwardingV1ApplyGateCommand(packet ExecutionForwardingV1ApplyPacket) []string {
	command := []string{
		"areaflow", "project", "execution-forwarding-v1-apply-gate", packet.ProjectKey,
		"--allowed-task-types", strings.Join(packet.AllowedTaskTypes, ","),
		"--approval-scope", packet.ApprovalScope,
		"--readiness-snapshot-hash", packet.ReadinessSnapshotHash,
		"--expected-shim-lifecycle-state", packet.ExpectedShimLifecycleState,
		"--failure-mode", packet.FailureMode,
		"--idempotency-key", packet.IdempotencyKey,
		"--audit-correlation-id", packet.AuditCorrelationID,
	}
	if packet.ApprovalID != "" {
		command = append(command, "--approval-id", packet.ApprovalID)
	}
	if packet.LegacyNonWriteProofID != "" {
		command = append(command, "--legacy-non-write-proof-id", packet.LegacyNonWriteProofID)
	}
	if packet.RollbackPlanID != "" {
		command = append(command, "--rollback-plan-id", packet.RollbackPlanID)
	}
	if packet.ProtectedPathFingerprintID != "" {
		command = append(command, "--protected-path-fingerprint-id", packet.ProtectedPathFingerprintID)
	}
	if packet.ExplicitApproval {
		command = append(command, "--explicit-approval")
	}
	if packet.ApprovalActor != "" {
		command = append(command, "--approval-actor", packet.ApprovalActor)
	}
	if packet.ApprovalReason != "" {
		command = append(command, "--approval-reason", packet.ApprovalReason)
	}
	return command
}

func executionForwardingV1FutureApplyCommand(packet ExecutionForwardingV1ApplyPacket) []string {
	command := append([]string{}, executionForwardingV1ApplyGateCommand(packet)...)
	command[2] = "execution-forwarding-v1-apply"
	return command
}

func executionForwardingV1ApplyPacketSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_preview":                  true,
		"apply_packet_generated":             true,
		"apply_command_executed":             false,
		"command_request_created":            false,
		"area_flow_run_created":              false,
		"task_loop_run_forwarded":            false,
		"legacy_task_loop_started":           false,
		"legacy_progress_written":            false,
		"legacy_logs_written":                false,
		"legacy_checkpoint_written":          false,
		"project_write_attempted":            false,
		"execution_write_attempted":          false,
		"engine_call_attempted":              false,
		"commands_run":                       false,
		"worker_scheduled":                   false,
		"secrets_resolved":                   false,
		"network_used":                       false,
		"areamatrix_protected_paths_touched": false,
	}
}
