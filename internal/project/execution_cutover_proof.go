package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type RecordExecutionCutoverProofOptions struct {
	ProofStatus                string
	Facts                      []string
	Summary                    string
	EvidenceURI                string
	ExecutionCutoverScope      string
	AllowedTaskTypes           []string
	ForbiddenActions           []string
	RollbackTarget             string
	RollbackMode               string
	FailClosed                 bool
	ReopenRequiresApproval     bool
	SourceWriteOpen            bool
	GeneratedRetainedWriteOpen bool
	RepairApplyOpen            bool
	CheckpointApplyOpen        bool
	EngineExecutionOpen        bool
	SecretResolveOpen          bool
	NetworkAPIIntegrationOpen  bool
	PublishApplyOpen           bool
	RestoreApplyOpen           bool
	ReviewDecision             string
	ReviewedBy                 string
	ReviewedAt                 time.Time
	IdempotencyKey             string
	Actor                      string
	Reason                     string
	Metadata                   map[string]any
}

type ExecutionCutoverProof struct {
	Project                         Record
	Status                          string
	ProofStatus                     string
	Decision                        string
	Message                         string
	Facts                           []string
	MissingFacts                    []string
	EventID                         int64
	AuditEventID                    int64
	IdempotencyKey                  string
	Created                         bool
	CreatedAt                       time.Time
	ExecutionCutoverScope           string
	AllowedTaskTypes                []string
	ForbiddenActions                []string
	RollbackTarget                  string
	RollbackMode                    string
	FailClosed                      bool
	ReopenRequiresApproval          bool
	SourceWriteOpen                 bool
	GeneratedRetainedWriteOpen      bool
	RepairApplyOpen                 bool
	CheckpointApplyOpen             bool
	EngineExecutionOpen             bool
	SecretResolveOpen               bool
	NetworkAPIIntegrationOpen       bool
	PublishApplyOpen                bool
	RestoreApplyOpen                bool
	ProjectWriteAttempted           bool
	ExecutionWriteAttempted         bool
	TaskLoopRunForwardedByCommand   bool
	EngineCallAttempted             bool
	CommandsRun                     bool
	LegacyProgressWritten           bool
	LegacyLogsWritten               bool
	LegacyCheckpointWritten         bool
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

const executionCutoverProofCommandType = "completion.execution_cutover_proof.record"
const executionCutoverProofEventType = "completion.execution_cutover_proof.recorded"

var allowedExecutionCutoverProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredExecutionCutoverProofFacts = []string{
	"explicit_execution_cutover_approval_recorded",
	"execution_cutover_command_response_recorded",
	"execution_cutover_event_and_audit_recorded",
	"task_loop_run_forwarding_window_proven",
	"rollback_plan_and_compatibility_window_proven",
	"no_unapproved_project_or_execution_write_attempted",
}

const executionCutoverProofScope = "execution_forwarding_v1_read_only_evidence_only"
const executionCutoverProofRollbackTarget = "read_only_shim"
const executionCutoverProofRollbackMode = "fail_closed_to_read_only_shim"
const executionCutoverProofBindingContract = "execution_cutover_scope_binding_v1"

var executionCutoverProofCurrentBindingComparisonKeys = []string{
	"execution_cutover_binding_contract",
	"execution_cutover_scope",
	"allowed_task_types",
	"allowed_task_types_hash",
	"forbidden_actions",
	"forbidden_actions_hash",
	"rollback_target",
	"rollback_mode",
	"fail_closed",
	"reopen_requires_approval",
	"source_write_open",
	"generated_retained_write_open",
	"repair_apply_open",
	"checkpoint_apply_open",
	"engine_execution_open",
	"secret_resolve_open",
	"network_api_integration_open",
	"publish_apply_open",
	"restore_apply_open",
	"execution_cutover_scope_binding_hash",
}

var requiredExecutionCutoverProofForbiddenActions = []string{
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
}

func (s Store) RecordExecutionCutoverProof(ctx context.Context, record Record, options RecordExecutionCutoverProofOptions) (ExecutionCutoverProof, error) {
	options = normalizeRecordExecutionCutoverProofOptions(options)
	if !allowedExecutionCutoverProofStatuses[options.ProofStatus] {
		return ExecutionCutoverProof{}, fmt.Errorf("unsupported execution cutover proof status %q", options.ProofStatus)
	}
	missingFacts := executionCutoverProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return ExecutionCutoverProof{}, fmt.Errorf("complete execution cutover proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("execution cutover", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return ExecutionCutoverProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := executionCutoverProofEvidenceBindingBlockers(options); len(blockers) > 0 {
			return ExecutionCutoverProof{}, fmt.Errorf("complete execution cutover proof missing execution cutover scope binding: %s", strings.Join(blockers, ","))
		}
	}
	if err := requireCompleteProofReviewEvidence("execution cutover", "execution_cutover_proof", options.ProofStatus, options.EvidenceURI, proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata)); err != nil {
		return ExecutionCutoverProof{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = executionCutoverProofIdempotencyKey(record, options)
	}
	requestHash, err := executionCutoverProofRequestHash(record, options)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ExecutionCutoverProof{}, fmt.Errorf("begin execution cutover proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, executionCutoverProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}
	if !created {
		result, err := loadExecutionCutoverProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ExecutionCutoverProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ExecutionCutoverProof{}, fmt.Errorf("commit idempotent execution cutover proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildExecutionCutoverProof(record, options)
	eventID, err := insertExecutionCutoverProofEvent(ctx, tx, result, options)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertExecutionCutoverProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, executionCutoverProofCommandType, options.IdempotencyKey, executionCutoverProofCommandResponse(result)); err != nil {
		return ExecutionCutoverProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExecutionCutoverProof{}, fmt.Errorf("commit execution cutover proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestExecutionCutoverProof(ctx context.Context) (ExecutionCutoverProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		executionCutoverProofEventType,
	)
	if err != nil {
		return ExecutionCutoverProof{}, fmt.Errorf("load latest execution cutover proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}
	if len(events) == 0 {
		return ExecutionCutoverProof{}, nil
	}
	return executionCutoverProofFromEvent(events[0]), nil
}

func (s Store) LatestExecutionCutoverProofForProject(ctx context.Context, record Record) (ExecutionCutoverProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1 AND project_id = $2
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		executionCutoverProofEventType,
		record.ID,
	)
	if err != nil {
		return ExecutionCutoverProof{}, fmt.Errorf("load latest project execution cutover proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}
	if len(events) == 0 {
		return ExecutionCutoverProof{}, nil
	}
	return executionCutoverProofFromEvent(events[0]), nil
}

func normalizeRecordExecutionCutoverProofOptions(options RecordExecutionCutoverProofOptions) RecordExecutionCutoverProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeExecutionCutoverProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.ExecutionCutoverScope = strings.TrimSpace(options.ExecutionCutoverScope)
	options.AllowedTaskTypes = normalizeStringList(options.AllowedTaskTypes)
	options.ForbiddenActions = normalizeStringList(options.ForbiddenActions)
	options.RollbackTarget = strings.TrimSpace(options.RollbackTarget)
	options.RollbackMode = strings.TrimSpace(options.RollbackMode)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.ProofStatus == "" {
		options.ProofStatus = "incomplete"
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "record execution cutover proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	options.ReviewDecision = strings.ToLower(strings.TrimSpace(firstNonEmptyString(options.ReviewDecision, metadataString(options.Metadata, "review_decision"))))
	options.ReviewedBy = strings.TrimSpace(firstNonEmptyString(options.ReviewedBy, metadataString(options.Metadata, "reviewed_by")))
	if options.ReviewedAt.IsZero() {
		options.ReviewedAt = metadataTime(options.Metadata, "reviewed_at")
	}
	return options
}

func executionCutoverProofRequestHash(record Record, options RecordExecutionCutoverProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     executionCutoverProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeExecutionCutoverProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"binding":          executionCutoverProofOptionsBindingPayload(options),
		"review_metadata":  proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata),
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal execution cutover proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func executionCutoverProofIdempotencyKey(record Record, options RecordExecutionCutoverProofOptions) string {
	hash, err := executionCutoverProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.execution_cutover_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildExecutionCutoverProof(record Record, options RecordExecutionCutoverProofOptions) ExecutionCutoverProof {
	facts := normalizeExecutionCutoverProofFacts(options.Facts)
	missingFacts := executionCutoverProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "execution cutover proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "execution cutover proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "execution cutover proof is incomplete"
	}
	metadata := map[string]any{}
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	metadata["project_key"] = record.Key
	metadata["proof_status"] = options.ProofStatus
	metadata["facts"] = facts
	metadata["missing_facts"] = missingFacts
	metadata["summary"] = options.Summary
	metadata["evidence_uri"] = options.EvidenceURI
	addProofReviewMetadata(metadata, options.ProofStatus, "execution_cutover_proof", proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata))
	addExecutionCutoverProofBindingMetadata(metadata, options)
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["task_loop_run_forwarded_by_command"] = false
	metadata["engine_call_attempted"] = false
	metadata["commands_run"] = false
	metadata["legacy_progress_written"] = false
	metadata["legacy_logs_written"] = false
	metadata["legacy_checkpoint_written"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return ExecutionCutoverProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ExecutionCutoverScope:           options.ExecutionCutoverScope,
		AllowedTaskTypes:                append([]string{}, options.AllowedTaskTypes...),
		ForbiddenActions:                append([]string{}, options.ForbiddenActions...),
		RollbackTarget:                  options.RollbackTarget,
		RollbackMode:                    options.RollbackMode,
		FailClosed:                      options.FailClosed,
		ReopenRequiresApproval:          options.ReopenRequiresApproval,
		SourceWriteOpen:                 options.SourceWriteOpen,
		GeneratedRetainedWriteOpen:      options.GeneratedRetainedWriteOpen,
		RepairApplyOpen:                 options.RepairApplyOpen,
		CheckpointApplyOpen:             options.CheckpointApplyOpen,
		EngineExecutionOpen:             options.EngineExecutionOpen,
		SecretResolveOpen:               options.SecretResolveOpen,
		NetworkAPIIntegrationOpen:       options.NetworkAPIIntegrationOpen,
		PublishApplyOpen:                options.PublishApplyOpen,
		RestoreApplyOpen:                options.RestoreApplyOpen,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		TaskLoopRunForwardedByCommand:   false,
		EngineCallAttempted:             false,
		CommandsRun:                     false,
		LegacyProgressWritten:           false,
		LegacyLogsWritten:               false,
		LegacyCheckpointWritten:         false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertExecutionCutoverProofEvent(ctx context.Context, tx pgx.Tx, result ExecutionCutoverProof, options RecordExecutionCutoverProofOptions) (int64, error) {
	metadata, err := json.Marshal(executionCutoverProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal execution cutover proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Execution cutover proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		executionCutoverProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert execution cutover proof event: %w", err)
	}
	return eventID, nil
}

func insertExecutionCutoverProofAuditEvent(ctx context.Context, tx pgx.Tx, result ExecutionCutoverProof, options RecordExecutionCutoverProofOptions) (int64, error) {
	metadata, err := json.Marshal(executionCutoverProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal execution cutover proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'execution_cutover_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		executionCutoverProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert execution cutover proof audit event: %w", err)
	}
	return auditEventID, nil
}

func executionCutoverProofEventMetadata(result ExecutionCutoverProof, options RecordExecutionCutoverProofOptions) map[string]any {
	metadata := executionCutoverProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func executionCutoverProofCommandResponse(result ExecutionCutoverProof) map[string]any {
	return map[string]any{
		"project_key":                            result.Project.Key,
		"status":                                 result.Status,
		"proof_status":                           result.ProofStatus,
		"decision":                               result.Decision,
		"message":                                result.Message,
		"facts":                                  result.Facts,
		"missing_facts":                          result.MissingFacts,
		"event_id":                               result.EventID,
		"audit_event_id":                         result.AuditEventID,
		"idempotency_key":                        result.IdempotencyKey,
		"execution_cutover_scope":                result.ExecutionCutoverScope,
		"execution_cutover_scope_binding_status": metadataString(result.Metadata, "execution_cutover_scope_binding_status"),
		"execution_cutover_scope_binding_blockers": metadataStringSlice(result.Metadata, "execution_cutover_scope_binding_blockers"),
		"execution_cutover_binding_contract":       metadataString(result.Metadata, "execution_cutover_binding_contract"),
		"allowed_task_types_hash":                  metadataString(result.Metadata, "allowed_task_types_hash"),
		"forbidden_actions_hash":                   metadataString(result.Metadata, "forbidden_actions_hash"),
		"execution_cutover_binding_hash":           metadataString(result.Metadata, "execution_cutover_binding_hash"),
		"execution_cutover_scope_binding_hash":     metadataString(result.Metadata, "execution_cutover_scope_binding_hash"),
		"allowed_task_types":                       result.AllowedTaskTypes,
		"forbidden_actions":                        result.ForbiddenActions,
		"rollback_target":                          result.RollbackTarget,
		"rollback_mode":                            result.RollbackMode,
		"fail_closed":                              result.FailClosed,
		"reopen_requires_approval":                 result.ReopenRequiresApproval,
		"source_write_open":                        result.SourceWriteOpen,
		"generated_retained_write_open":            result.GeneratedRetainedWriteOpen,
		"repair_apply_open":                        result.RepairApplyOpen,
		"checkpoint_apply_open":                    result.CheckpointApplyOpen,
		"engine_execution_open":                    result.EngineExecutionOpen,
		"secret_resolve_open":                      result.SecretResolveOpen,
		"network_api_integration_open":             result.NetworkAPIIntegrationOpen,
		"publish_apply_open":                       result.PublishApplyOpen,
		"restore_apply_open":                       result.RestoreApplyOpen,
		"project_write_attempted":                  result.ProjectWriteAttempted,
		"execution_write_attempted":                result.ExecutionWriteAttempted,
		"task_loop_run_forwarded_by_command":       result.TaskLoopRunForwardedByCommand,
		"engine_call_attempted":                    result.EngineCallAttempted,
		"commands_run":                             result.CommandsRun,
		"legacy_progress_written":                  result.LegacyProgressWritten,
		"legacy_logs_written":                      result.LegacyLogsWritten,
		"legacy_checkpoint_written":                result.LegacyCheckpointWritten,
		"area_matrix_protected_paths_touched":      result.AreaMatrixProtectedPathsTouched,
		"summary":                                  metadataString(result.Metadata, "summary"),
		"evidence_uri":                             metadataString(result.Metadata, "evidence_uri"),
		"review_decision":                          metadataString(result.Metadata, "review_decision"),
		"reviewed_by":                              metadataString(result.Metadata, "reviewed_by"),
		"reviewed_at":                              metadataString(result.Metadata, "reviewed_at"),
		"review_metadata_status":                   metadataString(result.Metadata, "review_metadata_status"),
		"review_metadata_blockers":                 metadataStringSlice(result.Metadata, "review_metadata_blockers"),
		"metadata":                                 result.Metadata,
	}
}

func loadExecutionCutoverProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ExecutionCutoverProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, executionCutoverProofCommandType, idempotencyKey)
	if err != nil {
		return ExecutionCutoverProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return ExecutionCutoverProof{
		Project:                         record,
		Status:                          metadataString(response, "status"),
		ProofStatus:                     metadataString(response, "proof_status"),
		Decision:                        metadataString(response, "decision"),
		Message:                         metadataString(response, "message"),
		Facts:                           metadataStringSlice(response, "facts"),
		MissingFacts:                    metadataStringSlice(response, "missing_facts"),
		EventID:                         metadataInt64(response, "event_id"),
		AuditEventID:                    metadataInt64(response, "audit_event_id"),
		IdempotencyKey:                  idempotencyKey,
		ExecutionCutoverScope:           metadataString(response, "execution_cutover_scope"),
		AllowedTaskTypes:                metadataStringSlice(response, "allowed_task_types"),
		ForbiddenActions:                metadataStringSlice(response, "forbidden_actions"),
		RollbackTarget:                  metadataString(response, "rollback_target"),
		RollbackMode:                    metadataString(response, "rollback_mode"),
		FailClosed:                      metadataBool(response, "fail_closed"),
		ReopenRequiresApproval:          metadataBool(response, "reopen_requires_approval"),
		SourceWriteOpen:                 metadataBool(response, "source_write_open"),
		GeneratedRetainedWriteOpen:      metadataBool(response, "generated_retained_write_open"),
		RepairApplyOpen:                 metadataBool(response, "repair_apply_open"),
		CheckpointApplyOpen:             metadataBool(response, "checkpoint_apply_open"),
		EngineExecutionOpen:             metadataBool(response, "engine_execution_open"),
		SecretResolveOpen:               metadataBool(response, "secret_resolve_open"),
		NetworkAPIIntegrationOpen:       metadataBool(response, "network_api_integration_open"),
		PublishApplyOpen:                metadataBool(response, "publish_apply_open"),
		RestoreApplyOpen:                metadataBool(response, "restore_apply_open"),
		ProjectWriteAttempted:           metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(response, "execution_write_attempted"),
		TaskLoopRunForwardedByCommand:   metadataBool(response, "task_loop_run_forwarded_by_command"),
		EngineCallAttempted:             metadataBool(response, "engine_call_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		LegacyProgressWritten:           metadataBool(response, "legacy_progress_written"),
		LegacyLogsWritten:               metadataBool(response, "legacy_logs_written"),
		LegacyCheckpointWritten:         metadataBool(response, "legacy_checkpoint_written"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func executionCutoverProofFromEvent(event EventRecord) ExecutionCutoverProof {
	metadata := proofMetadataFromEventMetadata(event.Metadata)
	return ExecutionCutoverProof{
		Project:                         Record{ID: event.ProjectID, Key: metadataString(event.Metadata, "project_key")},
		Status:                          metadataString(event.Metadata, "status"),
		ProofStatus:                     metadataString(event.Metadata, "proof_status"),
		Decision:                        metadataString(event.Metadata, "decision"),
		Message:                         metadataString(event.Metadata, "message"),
		Facts:                           metadataStringSlice(event.Metadata, "facts"),
		MissingFacts:                    metadataStringSlice(event.Metadata, "missing_facts"),
		EventID:                         event.ID,
		AuditEventID:                    metadataInt64(event.Metadata, "audit_event_id"),
		IdempotencyKey:                  metadataString(event.Metadata, "idempotency_key"),
		CreatedAt:                       event.CreatedAt,
		ExecutionCutoverScope:           metadataString(event.Metadata, "execution_cutover_scope"),
		AllowedTaskTypes:                metadataStringSlice(event.Metadata, "allowed_task_types"),
		ForbiddenActions:                metadataStringSlice(event.Metadata, "forbidden_actions"),
		RollbackTarget:                  metadataString(event.Metadata, "rollback_target"),
		RollbackMode:                    metadataString(event.Metadata, "rollback_mode"),
		FailClosed:                      metadataBool(event.Metadata, "fail_closed"),
		ReopenRequiresApproval:          metadataBool(event.Metadata, "reopen_requires_approval"),
		SourceWriteOpen:                 metadataBool(event.Metadata, "source_write_open"),
		GeneratedRetainedWriteOpen:      metadataBool(event.Metadata, "generated_retained_write_open"),
		RepairApplyOpen:                 metadataBool(event.Metadata, "repair_apply_open"),
		CheckpointApplyOpen:             metadataBool(event.Metadata, "checkpoint_apply_open"),
		EngineExecutionOpen:             metadataBool(event.Metadata, "engine_execution_open"),
		SecretResolveOpen:               metadataBool(event.Metadata, "secret_resolve_open"),
		NetworkAPIIntegrationOpen:       metadataBool(event.Metadata, "network_api_integration_open"),
		PublishApplyOpen:                metadataBool(event.Metadata, "publish_apply_open"),
		RestoreApplyOpen:                metadataBool(event.Metadata, "restore_apply_open"),
		ProjectWriteAttempted:           metadataBool(event.Metadata, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(event.Metadata, "execution_write_attempted"),
		TaskLoopRunForwardedByCommand:   metadataBool(event.Metadata, "task_loop_run_forwarded_by_command"),
		EngineCallAttempted:             metadataBool(event.Metadata, "engine_call_attempted"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		LegacyProgressWritten:           metadataBool(event.Metadata, "legacy_progress_written"),
		LegacyLogsWritten:               metadataBool(event.Metadata, "legacy_logs_written"),
		LegacyCheckpointWritten:         metadataBool(event.Metadata, "legacy_checkpoint_written"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}
}

func normalizeExecutionCutoverProofFacts(facts []string) []string {
	seen := map[string]bool{}
	normalized := []string{}
	for _, fact := range facts {
		fact = strings.TrimSpace(fact)
		if fact == "" || seen[fact] {
			continue
		}
		seen[fact] = true
		normalized = append(normalized, fact)
	}
	sort.Strings(normalized)
	return normalized
}

func executionCutoverProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredExecutionCutoverProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func executionCutoverProofOptionsBindingPayload(options RecordExecutionCutoverProofOptions) map[string]any {
	return executionCutoverProofBindingMetadata(
		options.ExecutionCutoverScope,
		options.AllowedTaskTypes,
		options.ForbiddenActions,
		options.RollbackTarget,
		options.RollbackMode,
		options.FailClosed,
		options.ReopenRequiresApproval,
		options.SourceWriteOpen,
		options.GeneratedRetainedWriteOpen,
		options.RepairApplyOpen,
		options.CheckpointApplyOpen,
		options.EngineExecutionOpen,
		options.SecretResolveOpen,
		options.NetworkAPIIntegrationOpen,
		options.PublishApplyOpen,
		options.RestoreApplyOpen,
	)
}

func addExecutionCutoverProofBindingMetadata(metadata map[string]any, options RecordExecutionCutoverProofOptions) {
	binding := executionCutoverProofOptionsBindingPayload(options)
	for key, value := range binding {
		metadata[key] = value
	}
	blockers := executionCutoverProofEvidenceBindingBlockers(options)
	metadata["execution_cutover_scope_binding_blockers"] = blockers
	if options.ProofStatus == "complete" && len(blockers) == 0 {
		metadata["execution_cutover_scope_binding_status"] = "pass"
	} else if len(blockers) > 0 {
		metadata["execution_cutover_scope_binding_status"] = "fail"
	} else {
		metadata["execution_cutover_scope_binding_status"] = "not_required"
	}
}

func executionCutoverProofEvidenceBindingBlockers(options RecordExecutionCutoverProofOptions) []string {
	blockers := []string{}
	if options.ExecutionCutoverScope != executionCutoverProofScope {
		blockers = append(blockers, "execution_cutover_scope_missing_or_mismatch")
	}
	if !sameNormalizedStrings(options.AllowedTaskTypes, executionForwardingV1AllowedTaskTypes) {
		blockers = append(blockers, "allowed_task_types_missing_or_mismatch")
	}
	if !sameNormalizedStrings(options.ForbiddenActions, requiredExecutionCutoverProofForbiddenActions) {
		blockers = append(blockers, "forbidden_actions_missing_or_mismatch")
	}
	if options.RollbackTarget != executionCutoverProofRollbackTarget {
		blockers = append(blockers, "rollback_target_missing_or_mismatch")
	}
	if options.RollbackMode != executionCutoverProofRollbackMode {
		blockers = append(blockers, "rollback_mode_missing_or_mismatch")
	}
	if !options.FailClosed {
		blockers = append(blockers, "fail_closed_missing")
	}
	if !options.ReopenRequiresApproval {
		blockers = append(blockers, "reopen_requires_approval_missing")
	}
	blockers = append(blockers, executionCutoverProofOpenSafetyBlockers(executionCutoverProofOptionsBindingPayload(options))...)
	return uniqueStrings(blockers)
}

func executionCutoverProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "execution_cutover_scope_binding_status") != "pass" {
		blockers = append(blockers, "execution_cutover_scope_binding_status_not_pass")
	}
	if metadataString(metadata, "execution_cutover_binding_contract") != executionCutoverProofBindingContract {
		blockers = append(blockers, "execution_cutover_binding_contract_missing_or_mismatch")
	}
	if metadataString(metadata, "execution_cutover_scope") != executionCutoverProofScope {
		blockers = append(blockers, "execution_cutover_scope_missing_or_mismatch")
	}
	allowedTaskTypes := metadataStringSlice(metadata, "allowed_task_types")
	if !sameNormalizedStrings(allowedTaskTypes, executionForwardingV1AllowedTaskTypes) {
		blockers = append(blockers, "allowed_task_types_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "allowed_task_types_hash")) ||
		metadataString(metadata, "allowed_task_types_hash") != executionCutoverProofStringSetHash("allowed_task_types", allowedTaskTypes) {
		blockers = append(blockers, "allowed_task_types_hash_missing_or_mismatch")
	}
	forbiddenActions := metadataStringSlice(metadata, "forbidden_actions")
	if !sameNormalizedStrings(forbiddenActions, requiredExecutionCutoverProofForbiddenActions) {
		blockers = append(blockers, "forbidden_actions_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "forbidden_actions_hash")) ||
		metadataString(metadata, "forbidden_actions_hash") != executionCutoverProofStringSetHash("forbidden_actions", forbiddenActions) {
		blockers = append(blockers, "forbidden_actions_hash_missing_or_mismatch")
	}
	if metadataString(metadata, "rollback_target") != executionCutoverProofRollbackTarget {
		blockers = append(blockers, "rollback_target_missing_or_mismatch")
	}
	if metadataString(metadata, "rollback_mode") != executionCutoverProofRollbackMode {
		blockers = append(blockers, "rollback_mode_missing_or_mismatch")
	}
	if !metadataBool(metadata, "fail_closed") {
		blockers = append(blockers, "fail_closed_missing")
	}
	if !metadataBool(metadata, "reopen_requires_approval") {
		blockers = append(blockers, "reopen_requires_approval_missing")
	}
	blockers = append(blockers, executionCutoverProofOpenSafetyBlockers(metadata)...)
	if !looksLikeSHA256(metadataString(metadata, "execution_cutover_scope_binding_hash")) ||
		metadataString(metadata, "execution_cutover_scope_binding_hash") != executionCutoverProofBindingHash(metadata) {
		blockers = append(blockers, "execution_cutover_scope_binding_hash_missing_or_mismatch")
	}
	return uniqueStrings(blockers)
}

func executionCutoverProofCurrentBinding() map[string]any {
	binding := executionCutoverProofBindingMetadata(
		executionCutoverProofScope,
		executionForwardingV1AllowedTaskTypes,
		requiredExecutionCutoverProofForbiddenActions,
		executionCutoverProofRollbackTarget,
		executionCutoverProofRollbackMode,
		true,
		true,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
	)
	binding["execution_cutover_scope_binding_status"] = "pass"
	binding["execution_cutover_scope_binding_blockers"] = []string{}
	return binding
}

func executionCutoverProofBindingMetadata(scope string, allowedTaskTypes []string, forbiddenActions []string, rollbackTarget string, rollbackMode string, failClosed bool, reopenRequiresApproval bool, sourceWriteOpen bool, generatedRetainedWriteOpen bool, repairApplyOpen bool, checkpointApplyOpen bool, engineExecutionOpen bool, secretResolveOpen bool, networkAPIIntegrationOpen bool, publishApplyOpen bool, restoreApplyOpen bool) map[string]any {
	allowedTaskTypes = normalizeStringList(allowedTaskTypes)
	forbiddenActions = normalizeStringList(forbiddenActions)
	metadata := map[string]any{
		"execution_cutover_binding_contract": executionCutoverProofBindingContract,
		"execution_cutover_scope":            scope,
		"allowed_task_types":                 allowedTaskTypes,
		"allowed_task_types_hash":            executionCutoverProofStringSetHash("allowed_task_types", allowedTaskTypes),
		"forbidden_actions":                  forbiddenActions,
		"forbidden_actions_hash":             executionCutoverProofStringSetHash("forbidden_actions", forbiddenActions),
		"rollback_target":                    rollbackTarget,
		"rollback_mode":                      rollbackMode,
		"fail_closed":                        failClosed,
		"reopen_requires_approval":           reopenRequiresApproval,
		"source_write_open":                  sourceWriteOpen,
		"generated_retained_write_open":      generatedRetainedWriteOpen,
		"repair_apply_open":                  repairApplyOpen,
		"checkpoint_apply_open":              checkpointApplyOpen,
		"engine_execution_open":              engineExecutionOpen,
		"secret_resolve_open":                secretResolveOpen,
		"network_api_integration_open":       networkAPIIntegrationOpen,
		"publish_apply_open":                 publishApplyOpen,
		"restore_apply_open":                 restoreApplyOpen,
	}
	metadata["execution_cutover_binding_hash"] = executionCutoverProofBindingHash(metadata)
	metadata["execution_cutover_scope_binding_hash"] = metadata["execution_cutover_binding_hash"]
	return metadata
}

func executionCutoverProofCurrentBindingBlockers(proofMetadata map[string]any, currentBinding map[string]any) []string {
	blockers := executionCutoverProofMetadataBindingBlockers(proofMetadata)
	if len(blockers) > 0 {
		return blockers
	}
	currentBlockers := executionCutoverProofMetadataBindingBlockers(currentBinding)
	if len(currentBlockers) > 0 {
		for _, blocker := range currentBlockers {
			blockers = append(blockers, "current_"+blocker)
		}
		return uniqueStrings(append([]string{"execution_cutover_scope_current_binding_mismatch"}, blockers...))
	}
	for _, key := range executionCutoverProofCurrentBindingComparisonKeys {
		if !executionCutoverProofBindingValuesEqual(proofMetadata, currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	if len(blockers) > 0 {
		blockers = append([]string{"execution_cutover_scope_current_binding_mismatch"}, blockers...)
	}
	return uniqueStrings(blockers)
}

func executionCutoverProofBindingValuesEqual(left map[string]any, right map[string]any, key string) bool {
	switch key {
	case "fail_closed", "reopen_requires_approval", "source_write_open", "generated_retained_write_open",
		"repair_apply_open", "checkpoint_apply_open", "engine_execution_open", "secret_resolve_open",
		"network_api_integration_open", "publish_apply_open", "restore_apply_open":
		return metadataBool(left, key) == metadataBool(right, key)
	case "allowed_task_types", "forbidden_actions":
		return sameNormalizedStrings(metadataStringSlice(left, key), metadataStringSlice(right, key))
	default:
		return metadataString(left, key) == metadataString(right, key)
	}
}

func executionCutoverProofBindingHash(metadata map[string]any) string {
	payload := map[string]any{
		"execution_cutover_binding_contract": metadataString(metadata, "execution_cutover_binding_contract"),
		"execution_cutover_scope":            metadataString(metadata, "execution_cutover_scope"),
		"allowed_task_types_hash":            metadataString(metadata, "allowed_task_types_hash"),
		"forbidden_actions_hash":             metadataString(metadata, "forbidden_actions_hash"),
		"rollback_target":                    metadataString(metadata, "rollback_target"),
		"rollback_mode":                      metadataString(metadata, "rollback_mode"),
		"fail_closed":                        metadataBool(metadata, "fail_closed"),
		"reopen_requires_approval":           metadataBool(metadata, "reopen_requires_approval"),
		"source_write_open":                  metadataBool(metadata, "source_write_open"),
		"generated_retained_write_open":      metadataBool(metadata, "generated_retained_write_open"),
		"repair_apply_open":                  metadataBool(metadata, "repair_apply_open"),
		"checkpoint_apply_open":              metadataBool(metadata, "checkpoint_apply_open"),
		"engine_execution_open":              metadataBool(metadata, "engine_execution_open"),
		"secret_resolve_open":                metadataBool(metadata, "secret_resolve_open"),
		"network_api_integration_open":       metadataBool(metadata, "network_api_integration_open"),
		"publish_apply_open":                 metadataBool(metadata, "publish_apply_open"),
		"restore_apply_open":                 metadataBool(metadata, "restore_apply_open"),
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func executionCutoverProofStringSetHash(kind string, values []string) string {
	payload, err := json.Marshal(map[string]any{
		"execution_cutover_binding_contract": executionCutoverProofBindingContract,
		"kind":                               kind,
		"values":                             normalizeStringList(values),
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func executionCutoverProofOpenSafetyBlockers(metadata map[string]any) []string {
	blockers := []string{}
	for key, blocker := range map[string]string{
		"source_write_open":             "source_write_open",
		"generated_retained_write_open": "generated_retained_write_open",
		"repair_apply_open":             "repair_apply_open",
		"checkpoint_apply_open":         "checkpoint_apply_open",
		"engine_execution_open":         "engine_execution_open",
		"secret_resolve_open":           "secret_resolve_open",
		"network_api_integration_open":  "network_api_integration_open",
		"publish_apply_open":            "publish_apply_open",
		"restore_apply_open":            "restore_apply_open",
	} {
		if metadataBool(metadata, key) {
			blockers = append(blockers, blocker)
		}
	}
	return blockers
}

func sameNormalizedStrings(left []string, right []string) bool {
	left = normalizeStringList(left)
	right = normalizeStringList(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func executionCutoverProofCompletesAudit(proof ExecutionCutoverProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proof.EventID > 0 &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		proofMetadataHasApprovedReviewEvidence("execution_cutover_proof", proof.Metadata) &&
		len(executionCutoverProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.SourceWriteOpen &&
		!proof.GeneratedRetainedWriteOpen &&
		!proof.RepairApplyOpen &&
		!proof.CheckpointApplyOpen &&
		!proof.EngineExecutionOpen &&
		!proof.SecretResolveOpen &&
		!proof.NetworkAPIIntegrationOpen &&
		!proof.PublishApplyOpen &&
		!proof.RestoreApplyOpen &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.TaskLoopRunForwardedByCommand &&
		!proof.EngineCallAttempted &&
		!proof.CommandsRun &&
		!proof.LegacyProgressWritten &&
		!proof.LegacyLogsWritten &&
		!proof.LegacyCheckpointWritten &&
		!proof.AreaMatrixProtectedPathsTouched
}
