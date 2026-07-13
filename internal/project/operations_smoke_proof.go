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

type RecordOperationsSmokeProofOptions struct {
	ProofKey       string
	EvidenceStatus string
	Summary        string
	EvidenceURI    string
	IdempotencyKey string
	Actor          string
	Reason         string
	Metadata       map[string]any
}

type OperationsSmokeProof struct {
	Project                         Record
	ProofKey                        string
	Status                          string
	EvidenceStatus                  string
	Decision                        string
	Message                         string
	EventID                         int64
	AuditEventID                    int64
	IdempotencyKey                  string
	Created                         bool
	CreatedAt                       time.Time
	ProjectWriteAttempted           bool
	ExecutionWriteAttempted         bool
	EngineCallAttempted             bool
	ServiceProcessControlAttempted  bool
	SupportBundleExported           bool
	MigrationApplyAttempted         bool
	RemoteTelemetryEnabled          bool
	AreaMatrixProtectedPathsTouched bool
	RecordCommandRunsSmoke          bool
	Metadata                        map[string]any
}

const operationsSmokeProofCommandType = "ops.smoke_proof.record"
const operationsSmokeProofEventType = "ops.smoke_proof.recorded"

var allowedOperationsSmokeProofKeys = map[string]bool{
	"local_ops_smoke":             true,
	"v1_stable_fixture_smoke":     true,
	"web_dashboard_ops_smoke":     true,
	"manual_ops_smoke_review":     true,
	"install_migrate_start_smoke": true,
}

func (s Store) RecordOperationsSmokeProof(ctx context.Context, record Record, options RecordOperationsSmokeProofOptions) (OperationsSmokeProof, error) {
	options = normalizeRecordOperationsSmokeProofOptions(options)
	if !allowedOperationsSmokeProofKeys[options.ProofKey] {
		return OperationsSmokeProof{}, fmt.Errorf("unsupported operations smoke proof key %q", options.ProofKey)
	}
	if err := requireProofEvidenceForStatus("operations smoke", options.EvidenceStatus, options.Summary, options.EvidenceURI, "pass"); err != nil {
		return OperationsSmokeProof{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = operationsSmokeProofIdempotencyKey(record, options)
	}
	requestHash, err := operationsSmokeProofRequestHash(record, options)
	if err != nil {
		return OperationsSmokeProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OperationsSmokeProof{}, fmt.Errorf("begin operations smoke proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, operationsSmokeProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return OperationsSmokeProof{}, err
	}
	if !created {
		result, err := loadOperationsSmokeProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return OperationsSmokeProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return OperationsSmokeProof{}, fmt.Errorf("commit idempotent operations smoke proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildOperationsSmokeProof(record, options)
	eventID, err := insertOperationsSmokeProofEvent(ctx, tx, result, options)
	if err != nil {
		return OperationsSmokeProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertOperationsSmokeProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return OperationsSmokeProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, operationsSmokeProofCommandType, options.IdempotencyKey, operationsSmokeProofCommandResponse(result)); err != nil {
		return OperationsSmokeProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OperationsSmokeProof{}, fmt.Errorf("commit operations smoke proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestOperationsSmokeProof(ctx context.Context) (OperationsSmokeProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		operationsSmokeProofEventType,
	)
	if err != nil {
		return OperationsSmokeProof{}, fmt.Errorf("load latest operations smoke proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return OperationsSmokeProof{}, err
	}
	if len(events) == 0 {
		return OperationsSmokeProof{}, nil
	}
	return operationsSmokeProofFromEvent(events[0]), nil
}

func (s Store) LatestOperationsSmokeProofForProject(ctx context.Context, record Record) (OperationsSmokeProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, operationsSmokeProofEventType)
	if err != nil {
		return OperationsSmokeProof{}, fmt.Errorf("load latest project operations smoke proof: %w", err)
	}
	if !ok {
		return OperationsSmokeProof{}, nil
	}
	return operationsSmokeProofFromEvent(event), nil
}

func normalizeRecordOperationsSmokeProofOptions(options RecordOperationsSmokeProofOptions) RecordOperationsSmokeProofOptions {
	options.ProofKey = strings.TrimSpace(options.ProofKey)
	options.EvidenceStatus = strings.TrimSpace(options.EvidenceStatus)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.EvidenceStatus == "" {
		options.EvidenceStatus = "pass"
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "record operations smoke proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func operationsSmokeProofRequestHash(record Record, options RecordOperationsSmokeProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":    operationsSmokeProofCommandType,
		"project_id":      record.ID,
		"project_key":     record.Key,
		"proof_key":       options.ProofKey,
		"evidence_status": options.EvidenceStatus,
		"summary":         options.Summary,
		"evidence_uri":    options.EvidenceURI,
		"actor":           options.Actor,
		"reason":          options.Reason,
		"metadata":        options.Metadata,
		"protected":       true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal operations smoke proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func operationsSmokeProofIdempotencyKey(record Record, options RecordOperationsSmokeProofOptions) string {
	hash, err := operationsSmokeProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("ops.smoke_proof.record:%s:%s:%s", record.Key, options.ProofKey, prefix)
}

func buildOperationsSmokeProof(record Record, options RecordOperationsSmokeProofOptions) OperationsSmokeProof {
	metadata := map[string]any{}
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	metadata["project_key"] = record.Key
	metadata["proof_key"] = options.ProofKey
	metadata["evidence_status"] = options.EvidenceStatus
	metadata["evidence_uri"] = options.EvidenceURI
	metadata["summary"] = options.Summary
	metadata["record_command_runs_smoke"] = false
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["engine_call_attempted"] = false
	metadata["service_process_control_attempted"] = false
	metadata["support_bundle_exported"] = false
	metadata["migration_apply_attempted"] = false
	metadata["remote_telemetry_enabled"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return OperationsSmokeProof{
		Project:                         record,
		ProofKey:                        options.ProofKey,
		Status:                          "recorded",
		EvidenceStatus:                  options.EvidenceStatus,
		Decision:                        "allowed",
		Message:                         "operations smoke proof recorded",
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		ServiceProcessControlAttempted:  false,
		SupportBundleExported:           false,
		MigrationApplyAttempted:         false,
		RemoteTelemetryEnabled:          false,
		AreaMatrixProtectedPathsTouched: false,
		RecordCommandRunsSmoke:          false,
		Metadata:                        metadata,
	}
}

func insertOperationsSmokeProofEvent(ctx context.Context, tx pgx.Tx, result OperationsSmokeProof, options RecordOperationsSmokeProofOptions) (int64, error) {
	metadata, err := json.Marshal(operationsSmokeProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal operations smoke proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Operations smoke proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		operationsSmokeProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert operations smoke proof event: %w", err)
	}
	return eventID, nil
}

func insertOperationsSmokeProofAuditEvent(ctx context.Context, tx pgx.Tx, result OperationsSmokeProof, options RecordOperationsSmokeProofOptions) (int64, error) {
	metadata, err := json.Marshal(operationsSmokeProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal operations smoke proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'read_operations', 'operations_smoke_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		operationsSmokeProofCommandType,
		result.ProofKey,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert operations smoke proof audit event: %w", err)
	}
	return auditEventID, nil
}

func operationsSmokeProofEventMetadata(result OperationsSmokeProof, options RecordOperationsSmokeProofOptions) map[string]any {
	metadata := operationsSmokeProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func operationsSmokeProofCommandResponse(result OperationsSmokeProof) map[string]any {
	return map[string]any{
		"project_key":                         result.Project.Key,
		"proof_key":                           result.ProofKey,
		"status":                              result.Status,
		"evidence_status":                     result.EvidenceStatus,
		"decision":                            result.Decision,
		"message":                             result.Message,
		"event_id":                            result.EventID,
		"audit_event_id":                      result.AuditEventID,
		"idempotency_key":                     result.IdempotencyKey,
		"project_write_attempted":             result.ProjectWriteAttempted,
		"execution_write_attempted":           result.ExecutionWriteAttempted,
		"engine_call_attempted":               result.EngineCallAttempted,
		"service_process_control_attempted":   result.ServiceProcessControlAttempted,
		"support_bundle_exported":             result.SupportBundleExported,
		"migration_apply_attempted":           result.MigrationApplyAttempted,
		"remote_telemetry_enabled":            result.RemoteTelemetryEnabled,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"record_command_runs_smoke":           result.RecordCommandRunsSmoke,
		"summary":                             metadataString(result.Metadata, "summary"),
		"evidence_uri":                        metadataString(result.Metadata, "evidence_uri"),
		"metadata":                            result.Metadata,
	}
}

func loadOperationsSmokeProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (OperationsSmokeProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, operationsSmokeProofCommandType, idempotencyKey)
	if err != nil {
		return OperationsSmokeProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return OperationsSmokeProof{
		Project:                         record,
		ProofKey:                        metadataString(response, "proof_key"),
		Status:                          metadataString(response, "status"),
		EvidenceStatus:                  metadataString(response, "evidence_status"),
		Decision:                        metadataString(response, "decision"),
		Message:                         metadataString(response, "message"),
		EventID:                         metadataInt64(response, "event_id"),
		AuditEventID:                    metadataInt64(response, "audit_event_id"),
		IdempotencyKey:                  idempotencyKey,
		ProjectWriteAttempted:           metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:             metadataBool(response, "engine_call_attempted"),
		ServiceProcessControlAttempted:  metadataBool(response, "service_process_control_attempted"),
		SupportBundleExported:           metadataBool(response, "support_bundle_exported"),
		MigrationApplyAttempted:         metadataBool(response, "migration_apply_attempted"),
		RemoteTelemetryEnabled:          metadataBool(response, "remote_telemetry_enabled"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		RecordCommandRunsSmoke:          metadataBool(response, "record_command_runs_smoke"),
		Metadata:                        metadata,
	}, nil
}

func operationsSmokeProofFromEvent(event EventRecord) OperationsSmokeProof {
	return OperationsSmokeProof{
		Project:                         Record{ID: event.ProjectID, Key: metadataString(event.Metadata, "project_key")},
		ProofKey:                        metadataString(event.Metadata, "proof_key"),
		Status:                          metadataString(event.Metadata, "status"),
		EvidenceStatus:                  metadataString(event.Metadata, "evidence_status"),
		Decision:                        metadataString(event.Metadata, "decision"),
		Message:                         metadataString(event.Metadata, "message"),
		EventID:                         event.ID,
		AuditEventID:                    metadataInt64(event.Metadata, "audit_event_id"),
		IdempotencyKey:                  metadataString(event.Metadata, "idempotency_key"),
		CreatedAt:                       event.CreatedAt,
		ProjectWriteAttempted:           metadataBool(event.Metadata, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(event.Metadata, "execution_write_attempted"),
		EngineCallAttempted:             metadataBool(event.Metadata, "engine_call_attempted"),
		ServiceProcessControlAttempted:  metadataBool(event.Metadata, "service_process_control_attempted"),
		SupportBundleExported:           metadataBool(event.Metadata, "support_bundle_exported"),
		MigrationApplyAttempted:         metadataBool(event.Metadata, "migration_apply_attempted"),
		RemoteTelemetryEnabled:          metadataBool(event.Metadata, "remote_telemetry_enabled"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		RecordCommandRunsSmoke:          metadataBool(event.Metadata, "record_command_runs_smoke"),
		Metadata:                        event.Metadata,
	}
}
