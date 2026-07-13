package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	shimApplyCommandMode = "shim_apply_command_v1"
	shimApplyCommandType = "project.shim.apply"
)

type ApplyShimCommandOptions struct {
	Gate           ShimApplyGateOptions
	IdempotencyKey string
	Actor          string
	Reason         string
	GeneratedAt    time.Time
}

type ApplyShimCommandResult struct {
	Project                 Record
	Status                  string
	Mode                    string
	Decision                string
	Message                 string
	Gate                    ShimApplyGate
	Blockers                []string
	EventID                 int64
	AuditEventID            int64
	IdempotencyKey          string
	Created                 bool
	RequiredPreflight       []string
	ForbiddenActions        []string
	SafetyFacts             map[string]bool
	ApplyOpen               bool
	AreaFlowCommandCreated  bool
	CommandRequestCreated   bool
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	TaskLoopRunForwarded    bool
	StatusProjectionWritten bool
	AreaMatrixFilesModified bool
	GeneratedAt             time.Time
}

func (s Store) ApplyShimCommand(ctx context.Context, record Record, options ApplyShimCommandOptions) (ApplyShimCommandResult, error) {
	options = normalizeApplyShimCommandOptions(options)
	requestHash, err := shimApplyCommandRequestHash(record, options)
	if err != nil {
		return ApplyShimCommandResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApplyShimCommandResult{}, fmt.Errorf("begin shim apply command: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, shimApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ApplyShimCommandResult{}, err
	}
	if !created {
		result, err := loadShimApplyCommandByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ApplyShimCommandResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApplyShimCommandResult{}, fmt.Errorf("commit idempotent shim apply command: %w", err)
		}
		result.Created = false
		return result, nil
	}

	gate, err := s.ShimApplyGate(ctx, record, options.Gate)
	if err != nil {
		return ApplyShimCommandResult{}, err
	}
	result := BuildApplyShimCommandResult(gate, options)
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	eventID, err := insertShimApplyCommandEvent(ctx, tx, result, options)
	if err != nil {
		return ApplyShimCommandResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertShimApplyCommandAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ApplyShimCommandResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, shimApplyCommandType, options.IdempotencyKey, shimApplyCommandResponse(result)); err != nil {
		return ApplyShimCommandResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApplyShimCommandResult{}, fmt.Errorf("commit shim apply command: %w", err)
	}
	return result, nil
}

func BuildApplyShimCommandResult(gate ShimApplyGate, options ApplyShimCommandOptions) ApplyShimCommandResult {
	options = normalizeApplyShimCommandOptions(options)
	blockers := []string{}
	if gate.Status != "pass" {
		blockers = append(blockers, "shim_apply_gate_not_pass")
	}
	if options.Gate.FailureMode != "fail_closed" {
		blockers = append(blockers, "failure_mode_must_be_fail_closed")
	}
	result := ApplyShimCommandResult{
		Project:                gate.Project,
		Status:                 "recorded",
		Mode:                   shimApplyCommandMode,
		Decision:               "allowed",
		Message:                "shim apply command recorded in AreaFlow command state; AreaMatrix shim file writes remain separately controlled",
		Gate:                   gate,
		Blockers:               blockers,
		RequiredPreflight:      append([]string{}, gate.RequiredPreflight...),
		ForbiddenActions:       append([]string{}, gate.ForbiddenActions...),
		ApplyOpen:              true,
		AreaFlowCommandCreated: true,
		CommandRequestCreated:  true,
		GeneratedAt:            options.GeneratedAt,
	}
	if len(blockers) > 0 || gate.Decision != "go" || !gate.ApplyCommandEligible {
		result.Status = "blocked"
		result.Decision = "denied"
		result.Message = "shim apply command blocked by gate requirements"
		result.ApplyOpen = false
		result.Blockers = uniqueStrings(append([]string{"shim_apply_gate_blocked"}, blockers...))
	}
	result.SafetyFacts = applyShimCommandSafetyFacts(result)
	return result
}

func normalizeApplyShimCommandOptions(options ApplyShimCommandOptions) ApplyShimCommandOptions {
	options.Gate = normalizeShimApplyGateOptions(options.Gate)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	options.Gate.GeneratedAt = options.GeneratedAt
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = options.Gate.IdempotencyKey
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = shimApplyCommandIdempotencyKey(options.Gate)
	}
	options.Gate.IdempotencyKey = options.IdempotencyKey
	if options.Gate.AuditCorrelationID == "" {
		options.Gate.AuditCorrelationID = "audit:" + options.IdempotencyKey
	}
	if options.Actor == "" {
		options.Actor = options.Gate.ApprovalActor
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = options.Gate.ApprovalReason
	}
	if options.Reason == "" {
		options.Reason = "record protected shim apply command"
	}
	return options
}

func shimApplyCommandRequestHash(record Record, options ApplyShimCommandOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": shimApplyCommandType,
		"project_id":   record.ID,
		"project_key":  record.Key,
		"project_root": record.RootPath,
		"actor":        options.Actor,
		"reason":       options.Reason,
		"gate_packet":  shimApplyCommandGateRequestHashPayload(options.Gate),
	})
	if err != nil {
		return "", fmt.Errorf("marshal shim apply command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func shimApplyCommandGateRequestHashPayload(options ShimApplyGateOptions) map[string]any {
	return map[string]any{
		"allowed_files":                 append([]string{}, options.AllowedFiles...),
		"approval_id":                   options.ApprovalID,
		"approval_scope":                options.ApprovalScope,
		"authorization_snapshot_hash":   options.AuthorizationSnapshotHash,
		"expected_authorization_mode":   options.ExpectedAuthorizationMode,
		"status_projection_packet_id":   options.StatusProjectionPacketID,
		"status_projection_gate_id":     options.StatusProjectionGateID,
		"read_only_smoke_evidence_id":   options.ReadOnlySmokeEvidenceID,
		"dirty_worktree_review_id":      options.DirtyWorktreeReviewID,
		"protected_path_fingerprint_id": options.ProtectedPathFingerprintID,
		"rollback_plan_id":              options.RollbackPlanID,
		"idempotency_key":               options.IdempotencyKey,
		"audit_correlation_id":          options.AuditCorrelationID,
		"failure_mode":                  options.FailureMode,
		"explicit_approval":             options.ExplicitApproval,
		"approval_actor":                options.ApprovalActor,
		"approval_reason":               options.ApprovalReason,
	}
}

func shimApplyCommandIdempotencyKey(gate ShimApplyGateOptions) string {
	payload, err := json.Marshal(shimApplyCommandGateRequestHashPayload(gate))
	if err != nil {
		payload = []byte("invalid-shim-apply-gate")
	}
	return fmt.Sprintf("project.shim.apply:%s", shortSHA256Hex(payload))
}

func applyShimCommandSafetyFacts(result ApplyShimCommandResult) map[string]bool {
	return map[string]bool{
		"apply_command_open":                 result.ApplyOpen,
		"apply_command_executed":             true,
		"command_request_created":            result.CommandRequestCreated,
		"area_flow_command_created":          result.AreaFlowCommandCreated,
		"project_write_attempted":            result.ProjectWriteAttempted,
		"execution_write_attempted":          result.ExecutionWriteAttempted,
		"task_loop_run_forwarded":            result.TaskLoopRunForwarded,
		"status_projection_written":          result.StatusProjectionWritten,
		"area_matrix_files_modified":         result.AreaMatrixFilesModified,
		"engine_call_attempted":              result.EngineCallAttempted,
		"commands_run":                       false,
		"worker_scheduled":                   false,
		"secrets_resolved":                   false,
		"network_used":                       false,
		"areamatrix_protected_paths_touched": result.AreaMatrixFilesModified,
		"source_write_open":                  false,
		"generated_retained_write_open":      false,
		"repair_apply_open":                  false,
		"checkpoint_apply_open":              false,
		"publish_apply_open":                 false,
		"restore_apply_open":                 false,
	}
}

func insertShimApplyCommandEvent(ctx context.Context, tx pgx.Tx, result ApplyShimCommandResult, options ApplyShimCommandOptions) (int64, error) {
	metadata, err := json.Marshal(shimApplyCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal shim apply command event metadata: %w", err)
	}
	eventType := "project.shim.apply.recorded"
	severity := "info"
	message := "Shim apply command recorded"
	if result.Decision == "denied" {
		eventType = "project.shim.apply.blocked"
		severity = "warning"
		message = "Shim apply command blocked"
	}
	_ = options
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		eventType,
		severity,
		message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert shim apply command event: %w", err)
	}
	return eventID, nil
}

func insertShimApplyCommandAuditEvent(ctx context.Context, tx pgx.Tx, result ApplyShimCommandResult, options ApplyShimCommandOptions) (int64, error) {
	metadata, err := json.Marshal(shimApplyCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal shim apply command audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'project_shim_write', 'shim_apply', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		shimApplyCommandType,
		result.Project.Key,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert shim apply command audit event: %w", err)
	}
	return auditEventID, nil
}

func shimApplyCommandResponse(result ApplyShimCommandResult) map[string]any {
	generatedAt := ""
	if !result.GeneratedAt.IsZero() {
		generatedAt = result.GeneratedAt.Format(time.RFC3339)
	}
	return map[string]any{
		"project_id":                 result.Project.ID,
		"project_key":                result.Project.Key,
		"status":                     result.Status,
		"mode":                       result.Mode,
		"decision":                   result.Decision,
		"message":                    result.Message,
		"blockers":                   result.Blockers,
		"gate_status":                result.Gate.Status,
		"gate_decision":              result.Gate.Decision,
		"gate_approval_status":       result.Gate.ApprovalStatus,
		"apply_command_eligible":     result.Gate.ApplyCommandEligible,
		"event_id":                   result.EventID,
		"audit_event_id":             result.AuditEventID,
		"idempotency_key":            result.IdempotencyKey,
		"created":                    result.Created,
		"required_preflight":         result.RequiredPreflight,
		"forbidden_actions":          result.ForbiddenActions,
		"safety_facts":               result.SafetyFacts,
		"apply_open":                 result.ApplyOpen,
		"area_flow_command_created":  result.AreaFlowCommandCreated,
		"command_request_created":    result.CommandRequestCreated,
		"project_write_attempted":    result.ProjectWriteAttempted,
		"execution_write_attempted":  result.ExecutionWriteAttempted,
		"engine_call_attempted":      result.EngineCallAttempted,
		"task_loop_run_forwarded":    result.TaskLoopRunForwarded,
		"status_projection_written":  result.StatusProjectionWritten,
		"area_matrix_files_modified": result.AreaMatrixFilesModified,
		"generated_at":               generatedAt,
	}
}

func loadShimApplyCommandByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ApplyShimCommandResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, shimApplyCommandType, idempotencyKey)
	if err != nil {
		return ApplyShimCommandResult{}, err
	}
	generatedAt := time.Now().UTC()
	if raw := metadataString(response, "generated_at"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			generatedAt = parsed
		}
	}
	gate := ShimApplyGate{
		Project:              record,
		Status:               metadataString(response, "gate_status"),
		Decision:             metadataString(response, "gate_decision"),
		ApprovalStatus:       metadataString(response, "gate_approval_status"),
		ApplyCommandEligible: metadataBool(response, "apply_command_eligible"),
	}
	result := ApplyShimCommandResult{
		Project:                 record,
		Status:                  metadataString(response, "status"),
		Mode:                    metadataString(response, "mode"),
		Decision:                metadataString(response, "decision"),
		Message:                 metadataString(response, "message"),
		Gate:                    gate,
		Blockers:                metadataStringSlice(response, "blockers"),
		EventID:                 metadataInt64(response, "event_id"),
		AuditEventID:            metadataInt64(response, "audit_event_id"),
		IdempotencyKey:          idempotencyKey,
		RequiredPreflight:       metadataStringSlice(response, "required_preflight"),
		ForbiddenActions:        metadataStringSlice(response, "forbidden_actions"),
		ApplyOpen:               metadataBool(response, "apply_open"),
		AreaFlowCommandCreated:  metadataBool(response, "area_flow_command_created"),
		CommandRequestCreated:   metadataBool(response, "command_request_created"),
		ProjectWriteAttempted:   metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted: metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:     metadataBool(response, "engine_call_attempted"),
		TaskLoopRunForwarded:    metadataBool(response, "task_loop_run_forwarded"),
		StatusProjectionWritten: metadataBool(response, "status_projection_written"),
		AreaMatrixFilesModified: metadataBool(response, "area_matrix_files_modified"),
		GeneratedAt:             generatedAt,
	}
	if raw, ok := response["safety_facts"].(map[string]any); ok {
		result.SafetyFacts = boolMapFromAnyMap(raw)
	}
	if result.SafetyFacts == nil {
		result.SafetyFacts = applyShimCommandSafetyFacts(result)
	}
	return result, nil
}
