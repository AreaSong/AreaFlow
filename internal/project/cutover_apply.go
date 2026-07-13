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

type ApplyCutoverOptions struct {
	VersionLabel   string
	IdempotencyKey string
	Actor          string
	Reason         string
	Mode           string
}

type ApplyCutoverResult struct {
	Project                  Record
	Version                  WorkflowVersion
	Status                   string
	Decision                 string
	Message                  string
	Blockers                 []string
	Warnings                 []string
	EventID                  int64
	AuditEventID             int64
	IdempotencyKey           string
	Created                  bool
	ProjectWriteAttempted    bool
	ExecutionWriteAttempted  bool
	AreaMatrixWriteAttempted bool
	CutoverReadinessGateID   int64
}

const cutoverApplyCommandType = "project.cutover.apply"

func (s Store) ApplyCutover(ctx context.Context, record Record, options ApplyCutoverOptions) (ApplyCutoverResult, error) {
	options = normalizeApplyCutoverOptions(record, options)
	version, err := s.GetWorkflowVersion(ctx, record, options.VersionLabel)
	if err != nil {
		return ApplyCutoverResult{}, err
	}
	if version.ImportMode != "authored" {
		return ApplyCutoverResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, options.VersionLabel)
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = cutoverApplyIdempotencyKey(record, version, options)
	}
	requestHash, err := cutoverApplyRequestHash(record, version, options)
	if err != nil {
		return ApplyCutoverResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApplyCutoverResult{}, fmt.Errorf("begin cutover apply: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, cutoverApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ApplyCutoverResult{}, err
	}
	if !created {
		result, err := loadCutoverApplyByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ApplyCutoverResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApplyCutoverResult{}, fmt.Errorf("commit idempotent cutover apply: %w", err)
		}
		result.Created = false
		return result, nil
	}

	readiness, err := s.ProjectCutoverReadiness(ctx, record, version.DisplayLabel, 10)
	if err != nil {
		return ApplyCutoverResult{}, err
	}
	result := evaluateCutoverApply(record, version, readiness, options)
	if result.Decision == "allowed" && result.Status == "applied" {
		version, err = updateWorkflowVersionAuthoringCutover(ctx, tx, version, result)
		if err != nil {
			return ApplyCutoverResult{}, err
		}
		result.Version = version
	}
	eventID, err := insertCutoverApplyEvent(ctx, tx, result)
	if err != nil {
		return ApplyCutoverResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertCutoverApplyAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ApplyCutoverResult{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, cutoverApplyCommandType, options.IdempotencyKey, cutoverApplyCommandResponse(result)); err != nil {
		return ApplyCutoverResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApplyCutoverResult{}, fmt.Errorf("commit cutover apply: %w", err)
	}
	return result, nil
}

func normalizeApplyCutoverOptions(record Record, options ApplyCutoverOptions) ApplyCutoverOptions {
	options.VersionLabel = strings.TrimSpace(options.VersionLabel)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.Mode = strings.TrimSpace(options.Mode)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "apply AreaFlow authoring cutover"
	}
	if options.Mode == "" {
		options.Mode = "authoring_cutover"
	}
	if options.VersionLabel == "" {
		options.VersionLabel = "unknown"
	}
	if record.Key == "" {
		record.Key = "unknown-project"
	}
	return options
}

func cutoverApplyRequestHash(record Record, version WorkflowVersion, options ApplyCutoverOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":        cutoverApplyCommandType,
		"project_key":         record.Key,
		"project_id":          record.ID,
		"workflow_version_id": version.ID,
		"display_label":       version.DisplayLabel,
		"mode":                options.Mode,
		"actor":               options.Actor,
		"reason":              options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal cutover apply command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func cutoverApplyIdempotencyKey(record Record, version WorkflowVersion, options ApplyCutoverOptions) string {
	hash, err := cutoverApplyRequestHash(record, version, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("project.cutover.apply:%s:%s:%s:%s", record.Key, version.DisplayLabel, options.Mode, prefix)
}

func evaluateCutoverApply(record Record, version WorkflowVersion, readiness ProjectCutoverReadiness, options ApplyCutoverOptions) ApplyCutoverResult {
	result := ApplyCutoverResult{
		Project:                  record,
		Version:                  version,
		Status:                   "applied",
		Decision:                 "allowed",
		Message:                  "authoring cutover applied in AreaFlow state",
		Warnings:                 append([]string{"authoring cutover does not write AreaMatrix project files or execution state"}, readiness.PhaseGate.AcceptedWarnings...),
		ProjectWriteAttempted:    false,
		ExecutionWriteAttempted:  false,
		AreaMatrixWriteAttempted: false,
	}
	if workflowVersionAuthoringCutoverApplied(version) {
		result.Status = "already_applied"
		result.Message = "authoring cutover was already applied in AreaFlow state"
		return result
	}
	blockers := []string{}
	if readiness.PhaseGate.Status != "pass" {
		blockers = append(blockers, readiness.PhaseGate.Blockers...)
		if len(readiness.PhaseGate.Blockers) == 0 {
			blockers = append(blockers, fmt.Sprintf("cutover readiness phase gate is %s", readiness.PhaseGate.Status))
		}
	}
	gate, ok := latestGateFromList(readiness.Gates, "cutover_readiness_gate")
	if !ok {
		blockers = append(blockers, "cutover_readiness_gate has not passed")
	} else {
		result.CutoverReadinessGateID = gate.ID
		if gate.Status != "pass" {
			blockers = append(blockers, fmt.Sprintf("cutover_readiness_gate status is %s", gate.Status))
		}
	}
	if options.Mode != "authoring_cutover" {
		blockers = append(blockers, fmt.Sprintf("cutover mode %q is not enabled; only authoring_cutover is available", options.Mode))
	}
	if len(blockers) > 0 {
		result.Status = "blocked"
		result.Decision = "denied"
		result.Message = "authoring cutover apply blocked by gate requirements"
		result.Blockers = blockers
	}
	return result
}

func workflowVersionAuthoringCutoverApplied(version WorkflowVersion) bool {
	if version.LifecycleStatus == "authoring_cutover" {
		return true
	}
	cutover, ok := version.StatusSummary["authoring_cutover"].(map[string]any)
	if !ok {
		return false
	}
	applied, ok := cutover["applied"].(bool)
	return ok && applied
}

func updateWorkflowVersionAuthoringCutover(ctx context.Context, tx pgx.Tx, version WorkflowVersion, result ApplyCutoverResult) (WorkflowVersion, error) {
	summary := map[string]any{}
	for key, value := range version.StatusSummary {
		summary[key] = value
	}
	summary["authoring_cutover"] = map[string]any{
		"applied":                     true,
		"applied_at":                  time.Now().UTC().Format(time.RFC3339),
		"mode":                        "authoring_cutover",
		"workflow_owner":              "areaflow",
		"execution_owner":             "project",
		"project_write_attempted":     false,
		"execution_write_attempted":   false,
		"cutover_readiness_gate_id":   result.CutoverReadinessGateID,
		"cutover_readiness_gate_name": "cutover_readiness_gate",
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return WorkflowVersion{}, fmt.Errorf("marshal cutover workflow version summary: %w", err)
	}
	updated, err := scanWorkflowVersion(tx.QueryRow(ctx, `
UPDATE workflow_versions
SET lifecycle_status = 'authoring_cutover',
    status_summary = $3::jsonb,
    updated_at = now()
WHERE project_id = $1 AND id = $2
RETURNING id, project_id, display_label, version_kind, lifecycle_status,
          COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
          immutable, status_summary, created_at, updated_at, imported_at`,
		version.ProjectID,
		version.ID,
		string(summaryJSON),
	))
	if err != nil {
		return WorkflowVersion{}, fmt.Errorf("update workflow version authoring cutover: %w", err)
	}
	return updated, nil
}

func insertCutoverApplyEvent(ctx context.Context, tx pgx.Tx, result ApplyCutoverResult) (int64, error) {
	metadata, err := json.Marshal(cutoverApplyCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal cutover apply event metadata: %w", err)
	}
	eventType := "project.cutover.apply.completed"
	severity := "info"
	message := "AreaFlow authoring cutover applied"
	if result.Decision == "denied" {
		eventType = "project.cutover.apply.blocked"
		severity = "warning"
		message = "AreaFlow authoring cutover apply blocked"
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Version.ID,
		eventType,
		severity,
		message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert cutover apply event: %w", err)
	}
	return eventID, nil
}

func insertCutoverApplyAuditEvent(ctx context.Context, tx pgx.Tx, result ApplyCutoverResult, options ApplyCutoverOptions) (int64, error) {
	metadata, err := json.Marshal(cutoverApplyCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal cutover apply audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'workflow_version', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		cutoverApplyCommandType,
		result.Version.DisplayLabel,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert cutover apply audit event: %w", err)
	}
	return auditEventID, nil
}

func cutoverApplyCommandResponse(result ApplyCutoverResult) map[string]any {
	return map[string]any{
		"project_key":                 result.Project.Key,
		"workflow_version_id":         result.Version.ID,
		"display_label":               result.Version.DisplayLabel,
		"lifecycle_status":            result.Version.LifecycleStatus,
		"status":                      result.Status,
		"decision":                    result.Decision,
		"message":                     result.Message,
		"blockers":                    result.Blockers,
		"warnings":                    result.Warnings,
		"event_id":                    result.EventID,
		"audit_event_id":              result.AuditEventID,
		"idempotency_key":             result.IdempotencyKey,
		"project_write_attempted":     result.ProjectWriteAttempted,
		"execution_write_attempted":   result.ExecutionWriteAttempted,
		"cutover_readiness_gate_id":   result.CutoverReadinessGateID,
		"area_matrix_write_attempted": result.AreaMatrixWriteAttempted,
	}
}

func loadCutoverApplyByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ApplyCutoverResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, cutoverApplyCommandType, idempotencyKey)
	if err != nil {
		return ApplyCutoverResult{}, err
	}
	version := WorkflowVersion{
		ID:              metadataInt64(response, "workflow_version_id"),
		ProjectID:       record.ID,
		DisplayLabel:    metadataString(response, "display_label"),
		LifecycleStatus: metadataString(response, "lifecycle_status"),
	}
	return ApplyCutoverResult{
		Project:                  record,
		Version:                  version,
		Status:                   metadataString(response, "status"),
		Decision:                 metadataString(response, "decision"),
		Message:                  metadataString(response, "message"),
		Blockers:                 stringSliceFromAny(response["blockers"]),
		Warnings:                 stringSliceFromAny(response["warnings"]),
		EventID:                  metadataInt64(response, "event_id"),
		AuditEventID:             metadataInt64(response, "audit_event_id"),
		IdempotencyKey:           idempotencyKey,
		ProjectWriteAttempted:    boolFromAny(response["project_write_attempted"]),
		ExecutionWriteAttempted:  boolFromAny(response["execution_write_attempted"]),
		AreaMatrixWriteAttempted: boolFromAny(response["area_matrix_write_attempted"]),
		CutoverReadinessGateID:   metadataInt64(response, "cutover_readiness_gate_id"),
	}, nil
}

func boolFromAny(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}
