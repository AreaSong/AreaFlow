package project

import (
	"context"
	"strings"
	"time"
)

const shimApplyPacketMode = "shim_apply_packet_preview_v1"

type ShimApplyPacketPreviewOptions struct {
	ExplicitApproval           bool
	ApprovalID                 string
	ApprovalActor              string
	ApprovalReason             string
	StatusProjectionPacketID   string
	StatusProjectionGateID     string
	ReadOnlySmokeEvidenceID    string
	DirtyWorktreeReviewID      string
	ProtectedPathFingerprintID string
	RollbackPlanID             string
	IdempotencyKey             string
	AuditCorrelationID         string
	GeneratedAt                time.Time
}

type ShimApplyPacketPreview struct {
	Project                                        Record
	Status                                         string
	Mode                                           string
	Decision                                       string
	Message                                        string
	Authorization                                  ShimAuthorizationPacket
	Gate                                           ShimApplyGate
	Packet                                         ShimApplyPacket
	ApplyGateCommand                               []string
	FutureApplyCommand                             []string
	RequiredHumanReview                            []string
	ForbiddenActions                               []string
	SafetyFacts                                    map[string]bool
	WouldCreateCommandRequestAfterApplyCommand     bool
	WouldWriteAreaMatrixShimFilesAfterApplyCommand bool
	WouldWriteStatusProjectionAfterApplyCommand    bool
	CommandRequestCreated                          bool
	ProjectWriteAttempted                          bool
	ExecutionWriteAttempted                        bool
	EngineCallAttempted                            bool
	TaskLoopRunForwarded                           bool
	StatusProjectionWritten                        bool
	AreaMatrixFilesModified                        bool
	GeneratedAt                                    time.Time
}

type ShimApplyPacket struct {
	CommandType                string   `json:"command_type"`
	ProjectKey                 string   `json:"project_key"`
	AllowedFiles               []string `json:"allowed_files"`
	ApprovalID                 string   `json:"approval_id"`
	ApprovalScope              string   `json:"approval_scope"`
	AuthorizationSnapshotHash  string   `json:"authorization_snapshot_hash"`
	ExpectedAuthorizationMode  string   `json:"expected_authorization_mode"`
	StatusProjectionPacketID   string   `json:"status_projection_packet_id"`
	StatusProjectionGateID     string   `json:"status_projection_gate_id"`
	ReadOnlySmokeEvidenceID    string   `json:"read_only_smoke_evidence_id"`
	DirtyWorktreeReviewID      string   `json:"dirty_worktree_review_id"`
	ProtectedPathFingerprintID string   `json:"protected_path_fingerprint_id"`
	RollbackPlanID             string   `json:"rollback_plan_id"`
	IdempotencyKey             string   `json:"idempotency_key"`
	AuditCorrelationID         string   `json:"audit_correlation_id"`
	FailureMode                string   `json:"failure_mode"`
	ExplicitApproval           bool     `json:"explicit_approval"`
	ApprovalActor              string   `json:"approval_actor"`
	ApprovalReason             string   `json:"approval_reason"`
}

func (s Store) ShimApplyPacketPreview(ctx context.Context, record Record, options ShimApplyPacketPreviewOptions) (ShimApplyPacketPreview, error) {
	options = normalizeShimApplyPacketPreviewOptions(options)
	authorization, err := s.ShimAuthorizationPacket(ctx, record)
	if err != nil {
		return ShimApplyPacketPreview{}, err
	}
	return BuildShimApplyPacketPreview(authorization, options), nil
}

func BuildShimApplyPacketPreview(authorization ShimAuthorizationPacket, options ShimApplyPacketPreviewOptions) ShimApplyPacketPreview {
	options = normalizeShimApplyPacketPreviewOptions(options)
	packet := shimApplyPacketFromAuthorization(authorization, options)
	gateOptions := shimApplyGateOptionsFromPacket(packet, options.GeneratedAt)
	gate := BuildShimApplyGate(authorization, gateOptions)
	preview := ShimApplyPacketPreview{
		Project:            authorization.Project,
		Status:             "needs_approval",
		Mode:               shimApplyPacketMode,
		Decision:           "needs_explicit_approval",
		Message:            "shim apply packet is generated but still needs explicit approval and proof ids",
		Authorization:      authorization,
		Gate:               gate,
		Packet:             packet,
		ApplyGateCommand:   shimApplyGateCommand(packet),
		FutureApplyCommand: shimApplyFutureApplyCommand(packet),
		RequiredHumanReview: []string{
			"review allowed AreaMatrix shim files only",
			"review status projection packet/gate proof ids",
			"review real AreaMatrix read-only smoke and dirty worktree review ids",
			"review protected path fingerprint and rollback plan ids",
			"confirm execution cutover, task-loop run forwarding, source writes and engine calls remain closed",
		},
		ForbiddenActions: append([]string{}, authorization.ForbiddenActions...),
		SafetyFacts:      shimApplyPacketSafetyFacts(),
		GeneratedAt:      options.GeneratedAt,
	}
	if gate.ApplyCommandEligible {
		preview.Status = "ready"
		preview.Decision = "ready_for_future_apply_command"
		preview.Message = "shim apply packet passed the read-only gate; future AreaMatrix shim edit command remains separately controlled"
		preview.WouldCreateCommandRequestAfterApplyCommand = true
		preview.WouldWriteAreaMatrixShimFilesAfterApplyCommand = true
		preview.WouldWriteStatusProjectionAfterApplyCommand = true
	} else if !shimOnlyExplicitApprovalBlocked(shimBlockedReadinessKeys(authorization.ReadinessItems)) {
		preview.Status = "blocked"
		preview.Decision = "readiness_blocked"
		preview.Message = "shim apply packet is blocked until shim readiness proof, status projection schema and AreaMatrix review evidence pass"
	}
	return preview
}

func normalizeShimApplyPacketPreviewOptions(options ShimApplyPacketPreviewOptions) ShimApplyPacketPreviewOptions {
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.ApprovalActor = strings.TrimSpace(options.ApprovalActor)
	options.ApprovalReason = strings.TrimSpace(options.ApprovalReason)
	options.StatusProjectionPacketID = strings.TrimSpace(options.StatusProjectionPacketID)
	options.StatusProjectionGateID = strings.TrimSpace(options.StatusProjectionGateID)
	options.ReadOnlySmokeEvidenceID = strings.TrimSpace(options.ReadOnlySmokeEvidenceID)
	options.DirtyWorktreeReviewID = strings.TrimSpace(options.DirtyWorktreeReviewID)
	options.ProtectedPathFingerprintID = strings.TrimSpace(options.ProtectedPathFingerprintID)
	options.RollbackPlanID = strings.TrimSpace(options.RollbackPlanID)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.AuditCorrelationID = strings.TrimSpace(options.AuditCorrelationID)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func shimApplyPacketFromAuthorization(authorization ShimAuthorizationPacket, options ShimApplyPacketPreviewOptions) ShimApplyPacket {
	authorizationHash := ShimAuthorizationSnapshotHash(authorization)
	idempotencyKey := options.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = "shim-apply:" + authorization.Project.Key + ":" + authorizationHash[:16]
	}
	auditCorrelationID := options.AuditCorrelationID
	if auditCorrelationID == "" {
		auditCorrelationID = "audit:" + idempotencyKey
	}
	return ShimApplyPacket{
		CommandType:                "project.shim.apply",
		ProjectKey:                 authorization.Project.Key,
		AllowedFiles:               shimAllowedFilePaths(authorization.AllowedFiles),
		ApprovalID:                 options.ApprovalID,
		ApprovalScope:              shimApplyApprovalScope(),
		AuthorizationSnapshotHash:  authorizationHash,
		ExpectedAuthorizationMode:  "read_only_authorization_packet",
		StatusProjectionPacketID:   options.StatusProjectionPacketID,
		StatusProjectionGateID:     options.StatusProjectionGateID,
		ReadOnlySmokeEvidenceID:    options.ReadOnlySmokeEvidenceID,
		DirtyWorktreeReviewID:      options.DirtyWorktreeReviewID,
		ProtectedPathFingerprintID: options.ProtectedPathFingerprintID,
		RollbackPlanID:             options.RollbackPlanID,
		IdempotencyKey:             idempotencyKey,
		AuditCorrelationID:         auditCorrelationID,
		FailureMode:                "fail_closed",
		ExplicitApproval:           options.ExplicitApproval,
		ApprovalActor:              options.ApprovalActor,
		ApprovalReason:             options.ApprovalReason,
	}
}

func shimApplyGateOptionsFromPacket(packet ShimApplyPacket, generatedAt time.Time) ShimApplyGateOptions {
	return ShimApplyGateOptions{
		AllowedFiles:               append([]string{}, packet.AllowedFiles...),
		ApprovalID:                 packet.ApprovalID,
		ApprovalScope:              packet.ApprovalScope,
		AuthorizationSnapshotHash:  packet.AuthorizationSnapshotHash,
		ExpectedAuthorizationMode:  packet.ExpectedAuthorizationMode,
		StatusProjectionPacketID:   packet.StatusProjectionPacketID,
		StatusProjectionGateID:     packet.StatusProjectionGateID,
		ReadOnlySmokeEvidenceID:    packet.ReadOnlySmokeEvidenceID,
		DirtyWorktreeReviewID:      packet.DirtyWorktreeReviewID,
		ProtectedPathFingerprintID: packet.ProtectedPathFingerprintID,
		RollbackPlanID:             packet.RollbackPlanID,
		IdempotencyKey:             packet.IdempotencyKey,
		AuditCorrelationID:         packet.AuditCorrelationID,
		FailureMode:                packet.FailureMode,
		ExplicitApproval:           packet.ExplicitApproval,
		ApprovalActor:              packet.ApprovalActor,
		ApprovalReason:             packet.ApprovalReason,
		GeneratedAt:                generatedAt,
	}
}

func shimApplyGateCommand(packet ShimApplyPacket) []string {
	command := []string{
		"areaflow", "project", "shim-apply-gate", packet.ProjectKey,
		"--allowed-files", strings.Join(packet.AllowedFiles, ","),
		"--approval-scope", packet.ApprovalScope,
		"--authorization-snapshot-hash", packet.AuthorizationSnapshotHash,
		"--expected-authorization-mode", packet.ExpectedAuthorizationMode,
		"--failure-mode", packet.FailureMode,
		"--idempotency-key", packet.IdempotencyKey,
		"--audit-correlation-id", packet.AuditCorrelationID,
	}
	if packet.ApprovalID != "" {
		command = append(command, "--approval-id", packet.ApprovalID)
	}
	if packet.StatusProjectionPacketID != "" {
		command = append(command, "--status-projection-packet-id", packet.StatusProjectionPacketID)
	}
	if packet.StatusProjectionGateID != "" {
		command = append(command, "--status-projection-gate-id", packet.StatusProjectionGateID)
	}
	if packet.ReadOnlySmokeEvidenceID != "" {
		command = append(command, "--read-only-smoke-evidence-id", packet.ReadOnlySmokeEvidenceID)
	}
	if packet.DirtyWorktreeReviewID != "" {
		command = append(command, "--dirty-worktree-review-id", packet.DirtyWorktreeReviewID)
	}
	if packet.ProtectedPathFingerprintID != "" {
		command = append(command, "--protected-path-fingerprint-id", packet.ProtectedPathFingerprintID)
	}
	if packet.RollbackPlanID != "" {
		command = append(command, "--rollback-plan-id", packet.RollbackPlanID)
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

func shimApplyFutureApplyCommand(packet ShimApplyPacket) []string {
	command := append([]string{}, shimApplyGateCommand(packet)...)
	command[2] = "shim-apply"
	return command
}

func shimApplyPacketSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_preview":                  true,
		"apply_packet_generated":             true,
		"apply_command_executed":             false,
		"command_request_created":            false,
		"project_write_attempted":            false,
		"execution_write_attempted":          false,
		"task_loop_run_forwarded":            false,
		"status_projection_written":          false,
		"area_matrix_files_modified":         false,
		"engine_call_attempted":              false,
		"commands_run":                       false,
		"worker_scheduled":                   false,
		"secrets_resolved":                   false,
		"network_used":                       false,
		"areamatrix_protected_paths_touched": false,
	}
}
