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

const executionForwardingV1ApplyCommandType = "project.execution_forwarding_v1.apply"

type ApplyExecutionForwardingV1Options struct {
	IdempotencyKey string
	Actor          string
	Reason         string
	Gate           ExecutionForwardingV1ApplyGateOptions
}

type ApplyExecutionForwardingV1Result struct {
	Project                         Record
	Status                          string
	Decision                        string
	Message                         string
	Blockers                        []string
	Gate                            ExecutionForwardingV1ApplyGate
	EventID                         int64
	AuditEventID                    int64
	IdempotencyKey                  string
	Created                         bool
	SafetyFacts                     map[string]bool
	CommandRequestCreated           bool
	AreaFlowCommandCreated          bool
	AreaFlowRunCreated              bool
	AreaFlowRunTaskCreated          bool
	AreaFlowRunAttemptCreated       bool
	AreaFlowArtifactCreated         bool
	AreaFlowAuditEventCreated       bool
	TaskLoopRunForwarded            bool
	LegacyTaskLoopStarted           bool
	LegacyProgressWritten           bool
	LegacyLogsWritten               bool
	LegacyCheckpointWritten         bool
	ProjectWriteAttempted           bool
	ExecutionWriteAttempted         bool
	EngineCallAttempted             bool
	CommandsRun                     bool
	SecretsResolved                 bool
	NetworkUsed                     bool
	AreaMatrixProtectedPathsTouched bool
	GeneratedAt                     time.Time
}

func (s Store) ApplyExecutionForwardingV1(ctx context.Context, record Record, options ApplyExecutionForwardingV1Options) (ApplyExecutionForwardingV1Result, error) {
	options = normalizeApplyExecutionForwardingV1Options(record, options)
	requestHash, err := executionForwardingV1ApplyRequestHash(record, options)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, fmt.Errorf("begin execution forwarding v1 apply: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, executionForwardingV1ApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}
	if !created {
		result, err := loadExecutionForwardingV1ApplyByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ApplyExecutionForwardingV1Result{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApplyExecutionForwardingV1Result{}, fmt.Errorf("commit idempotent execution forwarding v1 apply: %w", err)
		}
		result.Created = false
		return result, nil
	}

	gate, err := s.ExecutionForwardingV1ApplyGate(ctx, record, options.Gate)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}
	result := evaluateExecutionForwardingV1Apply(record, gate, options)
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	eventID, err := insertExecutionForwardingV1ApplyEvent(ctx, tx, result, options)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertExecutionForwardingV1ApplyAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}
	result.AuditEventID = auditEventID
	result.AreaFlowAuditEventCreated = true
	result.SafetyFacts = executionForwardingV1ApplySafetyFacts(result)
	if err := completeCommandRequestResponse(ctx, tx, record.ID, executionForwardingV1ApplyCommandType, options.IdempotencyKey, executionForwardingV1ApplyCommandResponse(result)); err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApplyExecutionForwardingV1Result{}, fmt.Errorf("commit execution forwarding v1 apply: %w", err)
	}
	return result, nil
}

func normalizeApplyExecutionForwardingV1Options(record Record, options ApplyExecutionForwardingV1Options) ApplyExecutionForwardingV1Options {
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.Gate = normalizeExecutionForwardingV1ApplyGateOptions(options.Gate)
	if options.Gate.IdempotencyKey == "" {
		options.Gate.IdempotencyKey = options.IdempotencyKey
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = executionForwardingV1ApplyIdempotencyKey(record, options.Gate)
		options.Gate.IdempotencyKey = options.IdempotencyKey
	}
	if options.Gate.IdempotencyKey == "" {
		options.Gate.IdempotencyKey = options.IdempotencyKey
	}
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
		options.Reason = "apply execution forwarding v1"
	}
	return options
}

func executionForwardingV1ApplyRequestHash(record Record, options ApplyExecutionForwardingV1Options) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": executionForwardingV1ApplyCommandType,
		"project_id":   record.ID,
		"project_key":  record.Key,
		"project_root": record.RootPath,
		"actor":        options.Actor,
		"reason":       options.Reason,
		"gate_packet":  executionForwardingV1ApplyGateRequestHashPayload(options.Gate),
	})
	if err != nil {
		return "", fmt.Errorf("marshal execution forwarding v1 apply request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func executionForwardingV1ApplyIdempotencyKey(record Record, gate ExecutionForwardingV1ApplyGateOptions) string {
	payload, err := json.Marshal(executionForwardingV1ApplyGateRequestHashPayload(gate))
	if err != nil {
		payload = []byte("invalid-execution-forwarding-v1-gate")
	}
	return fmt.Sprintf("project.execution_forwarding_v1.apply:%s:%s", record.Key, shortSHA256Hex(payload))
}

func executionForwardingV1ApplyGateRequestHashPayload(options ExecutionForwardingV1ApplyGateOptions) map[string]any {
	return map[string]any{
		"allowed_task_types":            append([]string{}, options.AllowedTaskTypes...),
		"approval_id":                   options.ApprovalID,
		"approval_scope":                options.ApprovalScope,
		"readiness_snapshot_hash":       options.ReadinessSnapshotHash,
		"expected_shim_lifecycle_state": options.ExpectedShimLifecycleState,
		"legacy_non_write_proof_id":     options.LegacyNonWriteProofID,
		"rollback_plan_id":              options.RollbackPlanID,
		"protected_path_fingerprint_id": options.ProtectedPathFingerprintID,
		"idempotency_key":               options.IdempotencyKey,
		"audit_correlation_id":          options.AuditCorrelationID,
		"failure_mode":                  options.FailureMode,
		"explicit_approval":             options.ExplicitApproval,
		"approval_actor":                options.ApprovalActor,
		"approval_reason":               options.ApprovalReason,
	}
}

func evaluateExecutionForwardingV1Apply(record Record, gate ExecutionForwardingV1ApplyGate, options ApplyExecutionForwardingV1Options) ApplyExecutionForwardingV1Result {
	result := ApplyExecutionForwardingV1Result{
		Project:                 record,
		Status:                  "applied",
		Decision:                "allowed",
		Message:                 "execution forwarding v1 apply recorded in AreaFlow command state",
		Gate:                    gate,
		CommandRequestCreated:   true,
		AreaFlowCommandCreated:  true,
		GeneratedAt:             time.Now().UTC(),
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	if gate.Status != "pass" || gate.Decision != "go" || !gate.ApplyCommandEligible {
		result.Status = "blocked"
		result.Decision = "denied"
		result.Message = "execution forwarding v1 apply blocked by gate requirements"
		result.Blockers = executionForwardingV1ApplyGateBlockers(gate)
	}
	if len(result.Blockers) == 0 && options.Gate.FailureMode != "fail_closed" {
		result.Status = "blocked"
		result.Decision = "denied"
		result.Message = "execution forwarding v1 apply blocked by protected boundary"
		result.Blockers = []string{"failure_mode_must_be_fail_closed"}
	}
	result.SafetyFacts = executionForwardingV1ApplySafetyFacts(result)
	return result
}

func executionForwardingV1ApplyGateBlockers(gate ExecutionForwardingV1ApplyGate) []string {
	blockers := []string{"execution_forwarding_v1_apply_gate_blocked"}
	for _, item := range gate.Items {
		if item.Status == "pass" {
			continue
		}
		if len(item.BlockedBy) == 0 {
			blockers = append(blockers, item.Key)
			continue
		}
		blockers = append(blockers, item.BlockedBy...)
	}
	return uniqueStrings(blockers)
}

func executionForwardingV1ApplySafetyFacts(result ApplyExecutionForwardingV1Result) map[string]bool {
	return map[string]bool{
		"apply_command_executed":             true,
		"command_request_created":            result.CommandRequestCreated,
		"area_flow_command_created":          result.AreaFlowCommandCreated,
		"area_flow_run_created":              result.AreaFlowRunCreated,
		"area_flow_run_task_created":         result.AreaFlowRunTaskCreated,
		"area_flow_run_attempt_created":      result.AreaFlowRunAttemptCreated,
		"area_flow_artifact_created":         result.AreaFlowArtifactCreated,
		"area_flow_audit_event_created":      result.AreaFlowAuditEventCreated,
		"forwarding_v1_apply_open":           result.Decision == "allowed",
		"task_loop_run_forwarded":            result.TaskLoopRunForwarded,
		"legacy_task_loop_started":           result.LegacyTaskLoopStarted,
		"legacy_progress_written":            result.LegacyProgressWritten,
		"legacy_logs_written":                result.LegacyLogsWritten,
		"legacy_checkpoint_written":          result.LegacyCheckpointWritten,
		"project_write_attempted":            result.ProjectWriteAttempted,
		"execution_write_attempted":          result.ExecutionWriteAttempted,
		"engine_call_attempted":              result.EngineCallAttempted,
		"commands_run":                       result.CommandsRun,
		"secrets_resolved":                   result.SecretsResolved,
		"network_used":                       result.NetworkUsed,
		"areamatrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"source_write_open":                  false,
		"generated_retained_write_open":      false,
		"repair_apply_open":                  false,
		"checkpoint_apply_open":              false,
		"publish_apply_open":                 false,
		"restore_apply_open":                 false,
	}
}

func insertExecutionForwardingV1ApplyEvent(ctx context.Context, tx pgx.Tx, result ApplyExecutionForwardingV1Result, options ApplyExecutionForwardingV1Options) (int64, error) {
	metadata, err := json.Marshal(executionForwardingV1ApplyCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal execution forwarding v1 apply event metadata: %w", err)
	}
	eventType := "project.execution_forwarding_v1.apply.completed"
	severity := "info"
	message := "Execution forwarding v1 apply recorded"
	if result.Decision == "denied" {
		eventType = "project.execution_forwarding_v1.apply.blocked"
		severity = "warning"
		message = "Execution forwarding v1 apply blocked"
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
		return 0, fmt.Errorf("insert execution forwarding v1 apply event: %w", err)
	}
	return eventID, nil
}

func insertExecutionForwardingV1ApplyAuditEvent(ctx context.Context, tx pgx.Tx, result ApplyExecutionForwardingV1Result, options ApplyExecutionForwardingV1Options) (int64, error) {
	metadata, err := json.Marshal(executionForwardingV1ApplyCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal execution forwarding v1 apply audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_execution', 'execution_forwarding_v1', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		executionForwardingV1ApplyCommandType,
		result.Project.Key,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert execution forwarding v1 apply audit event: %w", err)
	}
	return auditEventID, nil
}

func executionForwardingV1ApplyCommandResponse(result ApplyExecutionForwardingV1Result) map[string]any {
	generatedAt := ""
	if !result.GeneratedAt.IsZero() {
		generatedAt = result.GeneratedAt.Format(time.RFC3339)
	}
	return map[string]any{
		"project_id":                         result.Project.ID,
		"project_key":                        result.Project.Key,
		"status":                             result.Status,
		"decision":                           result.Decision,
		"message":                            result.Message,
		"blockers":                           result.Blockers,
		"gate_status":                        result.Gate.Status,
		"gate_decision":                      result.Gate.Decision,
		"gate_approval_status":               result.Gate.ApprovalStatus,
		"apply_command_eligible":             result.Gate.ApplyCommandEligible,
		"allowed_task_types":                 result.Gate.AllowedTaskTypes,
		"target_command_types":               result.Gate.TargetCommandTypes,
		"blocked_task_types":                 result.Gate.BlockedTaskTypes,
		"event_id":                           result.EventID,
		"audit_event_id":                     result.AuditEventID,
		"idempotency_key":                    result.IdempotencyKey,
		"created":                            result.Created,
		"safety_facts":                       result.SafetyFacts,
		"command_request_created":            result.CommandRequestCreated,
		"area_flow_command_created":          result.AreaFlowCommandCreated,
		"area_flow_run_created":              result.AreaFlowRunCreated,
		"area_flow_run_task_created":         result.AreaFlowRunTaskCreated,
		"area_flow_run_attempt_created":      result.AreaFlowRunAttemptCreated,
		"area_flow_artifact_created":         result.AreaFlowArtifactCreated,
		"area_flow_audit_event_created":      result.AreaFlowAuditEventCreated,
		"task_loop_run_forwarded":            result.TaskLoopRunForwarded,
		"legacy_task_loop_started":           result.LegacyTaskLoopStarted,
		"legacy_progress_written":            result.LegacyProgressWritten,
		"legacy_logs_written":                result.LegacyLogsWritten,
		"legacy_checkpoint_written":          result.LegacyCheckpointWritten,
		"project_write_attempted":            result.ProjectWriteAttempted,
		"execution_write_attempted":          result.ExecutionWriteAttempted,
		"engine_call_attempted":              result.EngineCallAttempted,
		"commands_run":                       result.CommandsRun,
		"secrets_resolved":                   result.SecretsResolved,
		"network_used":                       result.NetworkUsed,
		"areamatrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"generated_at":                       generatedAt,
	}
}

func loadExecutionForwardingV1ApplyByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ApplyExecutionForwardingV1Result, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, executionForwardingV1ApplyCommandType, idempotencyKey)
	if err != nil {
		return ApplyExecutionForwardingV1Result{}, err
	}
	generatedAt := time.Now().UTC()
	if raw := metadataString(response, "generated_at"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			generatedAt = parsed
		}
	}
	gate := ExecutionForwardingV1ApplyGate{
		Project:              record,
		Status:               metadataString(response, "gate_status"),
		Decision:             metadataString(response, "gate_decision"),
		ApprovalStatus:       metadataString(response, "gate_approval_status"),
		ApplyCommandEligible: metadataBool(response, "apply_command_eligible"),
		AllowedTaskTypes:     metadataStringSlice(response, "allowed_task_types"),
		TargetCommandTypes:   metadataStringSlice(response, "target_command_types"),
		BlockedTaskTypes:     metadataStringSlice(response, "blocked_task_types"),
	}
	result := ApplyExecutionForwardingV1Result{
		Project:                         record,
		Status:                          metadataString(response, "status"),
		Decision:                        metadataString(response, "decision"),
		Message:                         metadataString(response, "message"),
		Blockers:                        metadataStringSlice(response, "blockers"),
		Gate:                            gate,
		EventID:                         metadataInt64(response, "event_id"),
		AuditEventID:                    metadataInt64(response, "audit_event_id"),
		IdempotencyKey:                  idempotencyKey,
		CommandRequestCreated:           metadataBool(response, "command_request_created"),
		AreaFlowCommandCreated:          metadataBool(response, "area_flow_command_created"),
		AreaFlowRunCreated:              metadataBool(response, "area_flow_run_created"),
		AreaFlowRunTaskCreated:          metadataBool(response, "area_flow_run_task_created"),
		AreaFlowRunAttemptCreated:       metadataBool(response, "area_flow_run_attempt_created"),
		AreaFlowArtifactCreated:         metadataBool(response, "area_flow_artifact_created"),
		AreaFlowAuditEventCreated:       metadataBool(response, "area_flow_audit_event_created"),
		TaskLoopRunForwarded:            metadataBool(response, "task_loop_run_forwarded"),
		LegacyTaskLoopStarted:           metadataBool(response, "legacy_task_loop_started"),
		LegacyProgressWritten:           metadataBool(response, "legacy_progress_written"),
		LegacyLogsWritten:               metadataBool(response, "legacy_logs_written"),
		LegacyCheckpointWritten:         metadataBool(response, "legacy_checkpoint_written"),
		ProjectWriteAttempted:           metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:             metadataBool(response, "engine_call_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		SecretsResolved:                 metadataBool(response, "secrets_resolved"),
		NetworkUsed:                     metadataBool(response, "network_used"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "areamatrix_protected_paths_touched"),
		GeneratedAt:                     generatedAt,
	}
	if raw, ok := response["safety_facts"].(map[string]any); ok {
		result.SafetyFacts = boolMapFromAnyMap(raw)
	}
	if result.SafetyFacts == nil {
		result.SafetyFacts = executionForwardingV1ApplySafetyFacts(result)
	}
	return result, nil
}

func boolMapFromAnyMap(values map[string]any) map[string]bool {
	out := make(map[string]bool, len(values))
	for key, value := range values {
		out[key] = boolFromAny(value)
	}
	return out
}
