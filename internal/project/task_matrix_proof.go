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

type RecordTaskMatrixProofOptions struct {
	ProofStatus                           string
	Facts                                 []string
	Summary                               string
	EvidenceURI                           string
	TaskMatrixSourceSetHash               string
	TaskBacklogHash                       string
	TaskStatusAuditHash                   string
	PlannedV1RequiredTaskCount            int64
	PlannedV1RequiredTaskCountSet         bool
	MissingEvidenceV1RequiredTaskCount    int64
	MissingEvidenceV1RequiredTaskCountSet bool
	BlockedV1RequiredTaskCount            int64
	BlockedV1RequiredTaskCountSet         bool
	IdempotencyKey                        string
	Actor                                 string
	Reason                                string
	Metadata                              map[string]any
}

type TaskMatrixProof struct {
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
	ProjectWriteAttempted           bool
	ExecutionWriteAttempted         bool
	CommandsRun                     bool
	DocsWritten                     bool
	TasksWritten                    bool
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

const taskMatrixProofCommandType = "completion.task_matrix_proof.record"
const taskMatrixProofEventType = "completion.task_matrix_proof.recorded"

var allowedTaskMatrixProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredTaskMatrixProofFacts = []string{
	"all_v0_v1_tasks_have_status_evidence_and_boundary",
	"no_planned_v1_required_task_hidden",
	"preview_only_items_have_evidence_or_explicit_boundary",
	"implemented_scoped_items_have_scope_labels",
	"nearest_open_task_has_next_command_and_required_evidence",
	"v1x_deferred_tasks_have_contracts",
}

func (s Store) RecordTaskMatrixProof(ctx context.Context, record Record, options RecordTaskMatrixProofOptions) (TaskMatrixProof, error) {
	options = normalizeRecordTaskMatrixProofOptions(options)
	if !allowedTaskMatrixProofStatuses[options.ProofStatus] {
		return TaskMatrixProof{}, fmt.Errorf("unsupported task matrix proof status %q", options.ProofStatus)
	}
	missingFacts := taskMatrixProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return TaskMatrixProof{}, fmt.Errorf("complete task matrix proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("task matrix", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return TaskMatrixProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := taskMatrixProofOptionsBindingBlockers(options); len(blockers) > 0 {
			return TaskMatrixProof{}, fmt.Errorf("complete task matrix proof missing task matrix binding: %s", strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = taskMatrixProofIdempotencyKey(record, options)
	}
	requestHash, err := taskMatrixProofRequestHash(record, options)
	if err != nil {
		return TaskMatrixProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return TaskMatrixProof{}, fmt.Errorf("begin task matrix proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, taskMatrixProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return TaskMatrixProof{}, err
	}
	if !created {
		result, err := loadTaskMatrixProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return TaskMatrixProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return TaskMatrixProof{}, fmt.Errorf("commit idempotent task matrix proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildTaskMatrixProof(record, options)
	eventID, err := insertTaskMatrixProofEvent(ctx, tx, result, options)
	if err != nil {
		return TaskMatrixProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertTaskMatrixProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return TaskMatrixProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, taskMatrixProofCommandType, options.IdempotencyKey, taskMatrixProofCommandResponse(result)); err != nil {
		return TaskMatrixProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TaskMatrixProof{}, fmt.Errorf("commit task matrix proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestTaskMatrixProof(ctx context.Context) (TaskMatrixProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		taskMatrixProofEventType,
	)
	if err != nil {
		return TaskMatrixProof{}, fmt.Errorf("load latest task matrix proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return TaskMatrixProof{}, err
	}
	if len(events) == 0 {
		return TaskMatrixProof{}, nil
	}
	return taskMatrixProofFromEvent(events[0]), nil
}

func (s Store) LatestTaskMatrixProofForProject(ctx context.Context, record Record) (TaskMatrixProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, taskMatrixProofEventType)
	if err != nil {
		return TaskMatrixProof{}, fmt.Errorf("load latest project task matrix proof: %w", err)
	}
	if !ok {
		return TaskMatrixProof{}, nil
	}
	return taskMatrixProofFromEvent(event), nil
}

func normalizeRecordTaskMatrixProofOptions(options RecordTaskMatrixProofOptions) RecordTaskMatrixProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeTaskMatrixProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.TaskMatrixSourceSetHash = strings.TrimSpace(options.TaskMatrixSourceSetHash)
	options.TaskBacklogHash = strings.TrimSpace(options.TaskBacklogHash)
	options.TaskStatusAuditHash = strings.TrimSpace(options.TaskStatusAuditHash)
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
		options.Reason = "record task matrix proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func taskMatrixProofRequestHash(record Record, options RecordTaskMatrixProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": taskMatrixProofCommandType,
		"project_id":   record.ID,
		"project_key":  record.Key,
		"proof_status": options.ProofStatus,
		"facts":        normalizeTaskMatrixProofFacts(options.Facts),
		"summary":      options.Summary,
		"evidence_uri": options.EvidenceURI,
		"binding": map[string]any{
			"task_backlog_hash":                           options.TaskBacklogHash,
			"task_matrix_source_set_hash":                 options.TaskMatrixSourceSetHash,
			"task_status_audit_hash":                      options.TaskStatusAuditHash,
			"planned_v1_required_task_count":              options.PlannedV1RequiredTaskCount,
			"planned_v1_required_task_count_set":          options.PlannedV1RequiredTaskCountSet,
			"missing_evidence_v1_required_task_count":     options.MissingEvidenceV1RequiredTaskCount,
			"missing_evidence_v1_required_task_count_set": options.MissingEvidenceV1RequiredTaskCountSet,
			"blocked_v1_required_task_count":              options.BlockedV1RequiredTaskCount,
			"blocked_v1_required_task_count_set":          options.BlockedV1RequiredTaskCountSet,
		},
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal task matrix proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func taskMatrixProofIdempotencyKey(record Record, options RecordTaskMatrixProofOptions) string {
	hash, err := taskMatrixProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.task_matrix_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildTaskMatrixProof(record Record, options RecordTaskMatrixProofOptions) TaskMatrixProof {
	facts := normalizeTaskMatrixProofFacts(options.Facts)
	missingFacts := taskMatrixProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "task matrix proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "task matrix proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "task matrix proof is incomplete"
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
	addTaskMatrixProofBindingMetadata(metadata, options)
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["commands_run"] = false
	metadata["docs_written"] = false
	metadata["tasks_written"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return TaskMatrixProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		CommandsRun:                     false,
		DocsWritten:                     false,
		TasksWritten:                    false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertTaskMatrixProofEvent(ctx context.Context, tx pgx.Tx, result TaskMatrixProof, options RecordTaskMatrixProofOptions) (int64, error) {
	metadata, err := json.Marshal(taskMatrixProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal task matrix proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Task matrix proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		taskMatrixProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert task matrix proof event: %w", err)
	}
	return eventID, nil
}

func insertTaskMatrixProofAuditEvent(ctx context.Context, tx pgx.Tx, result TaskMatrixProof, options RecordTaskMatrixProofOptions) (int64, error) {
	metadata, err := json.Marshal(taskMatrixProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal task matrix proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'task_matrix_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		taskMatrixProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert task matrix proof audit event: %w", err)
	}
	return auditEventID, nil
}

func taskMatrixProofEventMetadata(result TaskMatrixProof, options RecordTaskMatrixProofOptions) map[string]any {
	metadata := taskMatrixProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func taskMatrixProofCommandResponse(result TaskMatrixProof) map[string]any {
	return map[string]any{
		"project_key":                             result.Project.Key,
		"status":                                  result.Status,
		"proof_status":                            result.ProofStatus,
		"decision":                                result.Decision,
		"message":                                 result.Message,
		"facts":                                   result.Facts,
		"missing_facts":                           result.MissingFacts,
		"event_id":                                result.EventID,
		"audit_event_id":                          result.AuditEventID,
		"idempotency_key":                         result.IdempotencyKey,
		"project_write_attempted":                 result.ProjectWriteAttempted,
		"execution_write_attempted":               result.ExecutionWriteAttempted,
		"commands_run":                            result.CommandsRun,
		"docs_written":                            result.DocsWritten,
		"tasks_written":                           result.TasksWritten,
		"area_matrix_protected_paths_touched":     result.AreaMatrixProtectedPathsTouched,
		"task_matrix_binding_status":              metadataString(result.Metadata, "task_matrix_binding_status"),
		"task_matrix_binding_blockers":            metadataStringSlice(result.Metadata, "task_matrix_binding_blockers"),
		"task_matrix_source_paths":                metadataStringSlice(result.Metadata, "task_matrix_source_paths"),
		"task_matrix_source_set_hash":             metadataString(result.Metadata, "task_matrix_source_set_hash"),
		"task_backlog_hash":                       metadataString(result.Metadata, "task_backlog_hash"),
		"task_status_audit_hash":                  metadataString(result.Metadata, "task_status_audit_hash"),
		"planned_v1_required_task_count":          metadataInt64(result.Metadata, "planned_v1_required_task_count"),
		"missing_evidence_v1_required_task_count": metadataInt64(result.Metadata, "missing_evidence_v1_required_task_count"),
		"blocked_v1_required_task_count":          metadataInt64(result.Metadata, "blocked_v1_required_task_count"),
		"summary":                                 metadataString(result.Metadata, "summary"),
		"evidence_uri":                            metadataString(result.Metadata, "evidence_uri"),
		"metadata":                                result.Metadata,
	}
}

func loadTaskMatrixProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (TaskMatrixProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, taskMatrixProofCommandType, idempotencyKey)
	if err != nil {
		return TaskMatrixProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return TaskMatrixProof{
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
		ProjectWriteAttempted:           metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(response, "execution_write_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		DocsWritten:                     metadataBool(response, "docs_written"),
		TasksWritten:                    metadataBool(response, "tasks_written"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func taskMatrixProofFromEvent(event EventRecord) TaskMatrixProof {
	return TaskMatrixProof{
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
		ProjectWriteAttempted:           metadataBool(event.Metadata, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(event.Metadata, "execution_write_attempted"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		DocsWritten:                     metadataBool(event.Metadata, "docs_written"),
		TasksWritten:                    metadataBool(event.Metadata, "tasks_written"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        event.Metadata,
	}
}

func normalizeTaskMatrixProofFacts(facts []string) []string {
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

func taskMatrixProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredTaskMatrixProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func taskMatrixProofCompletesAudit(proof TaskMatrixProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(taskMatrixProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.CommandsRun &&
		!proof.DocsWritten &&
		!proof.TasksWritten &&
		!proof.AreaMatrixProtectedPathsTouched
}
