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

type RecordArchiveProofOptions struct {
	ProofStatus             string
	Facts                   []string
	Summary                 string
	EvidenceURI             string
	ArchiveScope            string
	ArchiveReferenceMode    string
	ArchiveSourcePaths      []string
	ArchiveForbiddenActions []string
	ArchiveRollbackTarget   string
	ArchiveFailClosed       bool
	ReviewDecision          string
	ReviewedBy              string
	ReviewedAt              time.Time
	IdempotencyKey          string
	Actor                   string
	Reason                  string
	Metadata                map[string]any
}

type ArchiveProof struct {
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
	ArtifactBytesCopied             bool
	ArtifactBytesDeleted            bool
	HistoricalFilesDeleted          bool
	HistoricalFilesMoved            bool
	ProgressJSONRewritten           bool
	AreaMatrixProtectedPathsTouched bool
	CommandsRun                     bool
	ArchiveScope                    string
	ArchiveReferenceMode            string
	ArchiveSourcePaths              []string
	ArchiveForbiddenActions         []string
	ArchiveRollbackTarget           string
	ArchiveFailClosed               bool
	Metadata                        map[string]any
}

const archiveProofCommandType = "completion.archive_proof.record"
const archiveProofEventType = "completion.archive_proof.recorded"

var allowedArchiveProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredArchiveProofFacts = []string{
	"historical_workflow_versions_marked_immutable",
	"historical_execution_metadata_indexed_in_areaflow",
	"historical_artifact_refs_have_hash_path_type_project_version_run",
	"project_reference_restore_limitations_recorded",
	"old_progress_logs_checkpoints_are_reference_only",
	"new_run_attempt_artifact_audit_state_owned_by_areaflow",
	"areamatrix_workflow_readme_summary_contract_reviewed",
	"areamatrix_status_json_rough_projection_contract_reviewed",
	"archive_does_not_delete_or_move_historical_files",
	"archive_does_not_rewrite_progress_json",
	"rollback_to_execution_forwarding_documented",
}

const archiveProofScope = "areamatrix_historical_execution_reference_only"
const archiveProofReferenceMode = "metadata_indexed_reference_only"
const archiveProofRollbackTarget = "execution_forwarding_read_only_shim"
const archiveProofBindingContract = "archive_scope_binding_v1"

var archiveProofCurrentBindingComparisonKeys = []string{
	"archive_binding_contract",
	"archive_scope",
	"archive_reference_mode",
	"archive_source_paths",
	"archive_source_paths_hash",
	"archive_forbidden_actions",
	"archive_forbidden_actions_hash",
	"archive_rollback_target",
	"archive_fail_closed",
	"archive_scope_binding_hash",
}

var requiredArchiveProofSourcePaths = []string{
	".areaflow/status.json",
	"workflow/README.md",
	"workflow/versions/**/execution/**",
	"workflow/versions/**/execution/_shared/progress.json",
}

var requiredArchiveProofForbiddenActions = []string{
	"copy_artifact_bytes",
	"delete_artifact_bytes",
	"delete_historical_files",
	"move_historical_files",
	"rewrite_progress_json",
	"run_commands",
	"write_areamatrix_protected_paths",
}

func (s Store) RecordArchiveProof(ctx context.Context, record Record, options RecordArchiveProofOptions) (ArchiveProof, error) {
	options = normalizeRecordArchiveProofOptions(options)
	if !allowedArchiveProofStatuses[options.ProofStatus] {
		return ArchiveProof{}, fmt.Errorf("unsupported archive proof status %q", options.ProofStatus)
	}
	missingFacts := archiveProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return ArchiveProof{}, fmt.Errorf("complete archive proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("archive", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return ArchiveProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := archiveProofOptionsBindingBlockers(options); len(blockers) > 0 {
			return ArchiveProof{}, fmt.Errorf("complete archive proof missing archive scope binding: %s", strings.Join(blockers, ","))
		}
	}
	if err := requireCompleteProofReviewEvidence("archive", "archive_proof", options.ProofStatus, options.EvidenceURI, proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata)); err != nil {
		return ArchiveProof{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = archiveProofIdempotencyKey(record, options)
	}
	requestHash, err := archiveProofRequestHash(record, options)
	if err != nil {
		return ArchiveProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ArchiveProof{}, fmt.Errorf("begin archive proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, archiveProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ArchiveProof{}, err
	}
	if !created {
		result, err := loadArchiveProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ArchiveProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ArchiveProof{}, fmt.Errorf("commit idempotent archive proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildArchiveProof(record, options)
	eventID, err := insertArchiveProofEvent(ctx, tx, result, options)
	if err != nil {
		return ArchiveProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertArchiveProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ArchiveProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, archiveProofCommandType, options.IdempotencyKey, archiveProofCommandResponse(result)); err != nil {
		return ArchiveProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArchiveProof{}, fmt.Errorf("commit archive proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestArchiveProof(ctx context.Context) (ArchiveProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		archiveProofEventType,
	)
	if err != nil {
		return ArchiveProof{}, fmt.Errorf("load latest archive proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ArchiveProof{}, err
	}
	if len(events) == 0 {
		return ArchiveProof{}, nil
	}
	return archiveProofFromEvent(events[0]), nil
}

func (s Store) LatestArchiveProofForProject(ctx context.Context, record Record) (ArchiveProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, archiveProofEventType)
	if err != nil {
		return ArchiveProof{}, fmt.Errorf("load latest project archive proof: %w", err)
	}
	if !ok {
		return ArchiveProof{}, nil
	}
	return archiveProofFromEvent(event), nil
}

func normalizeRecordArchiveProofOptions(options RecordArchiveProofOptions) RecordArchiveProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeArchiveProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.ArchiveScope = strings.TrimSpace(options.ArchiveScope)
	options.ArchiveReferenceMode = strings.TrimSpace(options.ArchiveReferenceMode)
	options.ArchiveSourcePaths = normalizeStringList(options.ArchiveSourcePaths)
	options.ArchiveForbiddenActions = normalizeStringList(options.ArchiveForbiddenActions)
	options.ArchiveRollbackTarget = strings.TrimSpace(options.ArchiveRollbackTarget)
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
		options.Reason = "record AreaMatrix archive proof"
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

func archiveProofRequestHash(record Record, options RecordArchiveProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     archiveProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeArchiveProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"binding":          archiveProofOptionsBindingPayload(options),
		"review_metadata":  proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata),
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal archive proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func archiveProofIdempotencyKey(record Record, options RecordArchiveProofOptions) string {
	hash, err := archiveProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.archive_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildArchiveProof(record Record, options RecordArchiveProofOptions) ArchiveProof {
	facts := normalizeArchiveProofFacts(options.Facts)
	missingFacts := archiveProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "AreaMatrix archive proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "AreaMatrix archive proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "AreaMatrix archive proof is incomplete"
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
	addProofReviewMetadata(metadata, options.ProofStatus, "archive_proof", proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata))
	addArchiveProofBindingMetadata(metadata, options)
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["artifact_bytes_copied"] = false
	metadata["artifact_bytes_deleted"] = false
	metadata["historical_files_deleted"] = false
	metadata["historical_files_moved"] = false
	metadata["progress_json_rewritten"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	metadata["commands_run"] = false
	return ArchiveProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		ArtifactBytesCopied:             false,
		ArtifactBytesDeleted:            false,
		HistoricalFilesDeleted:          false,
		HistoricalFilesMoved:            false,
		ProgressJSONRewritten:           false,
		AreaMatrixProtectedPathsTouched: false,
		CommandsRun:                     false,
		ArchiveScope:                    options.ArchiveScope,
		ArchiveReferenceMode:            options.ArchiveReferenceMode,
		ArchiveSourcePaths:              append([]string{}, options.ArchiveSourcePaths...),
		ArchiveForbiddenActions:         append([]string{}, options.ArchiveForbiddenActions...),
		ArchiveRollbackTarget:           options.ArchiveRollbackTarget,
		ArchiveFailClosed:               options.ArchiveFailClosed,
		Metadata:                        metadata,
	}
}

func insertArchiveProofEvent(ctx context.Context, tx pgx.Tx, result ArchiveProof, options RecordArchiveProofOptions) (int64, error) {
	metadata, err := json.Marshal(archiveProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal archive proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Archive proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		archiveProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert archive proof event: %w", err)
	}
	return eventID, nil
}

func insertArchiveProofAuditEvent(ctx context.Context, tx pgx.Tx, result ArchiveProof, options RecordArchiveProofOptions) (int64, error) {
	metadata, err := json.Marshal(archiveProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal archive proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'archive_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		archiveProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert archive proof audit event: %w", err)
	}
	return auditEventID, nil
}

func archiveProofEventMetadata(result ArchiveProof, options RecordArchiveProofOptions) map[string]any {
	metadata := archiveProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func archiveProofCommandResponse(result ArchiveProof) map[string]any {
	return map[string]any{
		"project_key":                         result.Project.Key,
		"status":                              result.Status,
		"proof_status":                        result.ProofStatus,
		"decision":                            result.Decision,
		"message":                             result.Message,
		"facts":                               result.Facts,
		"missing_facts":                       result.MissingFacts,
		"event_id":                            result.EventID,
		"audit_event_id":                      result.AuditEventID,
		"idempotency_key":                     result.IdempotencyKey,
		"project_write_attempted":             result.ProjectWriteAttempted,
		"execution_write_attempted":           result.ExecutionWriteAttempted,
		"artifact_bytes_copied":               result.ArtifactBytesCopied,
		"artifact_bytes_deleted":              result.ArtifactBytesDeleted,
		"historical_files_deleted":            result.HistoricalFilesDeleted,
		"historical_files_moved":              result.HistoricalFilesMoved,
		"progress_json_rewritten":             result.ProgressJSONRewritten,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"commands_run":                        result.CommandsRun,
		"archive_scope_binding_status":        metadataString(result.Metadata, "archive_scope_binding_status"),
		"archive_scope_binding_blockers":      metadataStringSlice(result.Metadata, "archive_scope_binding_blockers"),
		"archive_binding_contract":            metadataString(result.Metadata, "archive_binding_contract"),
		"archive_source_paths_hash":           metadataString(result.Metadata, "archive_source_paths_hash"),
		"archive_forbidden_actions_hash":      metadataString(result.Metadata, "archive_forbidden_actions_hash"),
		"archive_binding_hash":                metadataString(result.Metadata, "archive_binding_hash"),
		"archive_scope_binding_hash":          metadataString(result.Metadata, "archive_scope_binding_hash"),
		"archive_scope":                       result.ArchiveScope,
		"archive_reference_mode":              result.ArchiveReferenceMode,
		"archive_source_paths":                result.ArchiveSourcePaths,
		"archive_forbidden_actions":           result.ArchiveForbiddenActions,
		"archive_rollback_target":             result.ArchiveRollbackTarget,
		"archive_fail_closed":                 result.ArchiveFailClosed,
		"summary":                             metadataString(result.Metadata, "summary"),
		"evidence_uri":                        metadataString(result.Metadata, "evidence_uri"),
		"review_decision":                     metadataString(result.Metadata, "review_decision"),
		"reviewed_by":                         metadataString(result.Metadata, "reviewed_by"),
		"reviewed_at":                         metadataString(result.Metadata, "reviewed_at"),
		"review_metadata_status":              metadataString(result.Metadata, "review_metadata_status"),
		"review_metadata_blockers":            metadataStringSlice(result.Metadata, "review_metadata_blockers"),
		"metadata":                            result.Metadata,
	}
}

func loadArchiveProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ArchiveProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, archiveProofCommandType, idempotencyKey)
	if err != nil {
		return ArchiveProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return ArchiveProof{
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
		ArtifactBytesCopied:             metadataBool(response, "artifact_bytes_copied"),
		ArtifactBytesDeleted:            metadataBool(response, "artifact_bytes_deleted"),
		HistoricalFilesDeleted:          metadataBool(response, "historical_files_deleted"),
		HistoricalFilesMoved:            metadataBool(response, "historical_files_moved"),
		ProgressJSONRewritten:           metadataBool(response, "progress_json_rewritten"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		ArchiveScope:                    metadataString(response, "archive_scope"),
		ArchiveReferenceMode:            metadataString(response, "archive_reference_mode"),
		ArchiveSourcePaths:              metadataStringSlice(response, "archive_source_paths"),
		ArchiveForbiddenActions:         metadataStringSlice(response, "archive_forbidden_actions"),
		ArchiveRollbackTarget:           metadataString(response, "archive_rollback_target"),
		ArchiveFailClosed:               metadataBool(response, "archive_fail_closed"),
		Metadata:                        metadata,
	}, nil
}

func archiveProofFromEvent(event EventRecord) ArchiveProof {
	metadata := proofMetadataFromEventMetadata(event.Metadata)
	return ArchiveProof{
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
		ArtifactBytesCopied:             metadataBool(event.Metadata, "artifact_bytes_copied"),
		ArtifactBytesDeleted:            metadataBool(event.Metadata, "artifact_bytes_deleted"),
		HistoricalFilesDeleted:          metadataBool(event.Metadata, "historical_files_deleted"),
		HistoricalFilesMoved:            metadataBool(event.Metadata, "historical_files_moved"),
		ProgressJSONRewritten:           metadataBool(event.Metadata, "progress_json_rewritten"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		ArchiveScope:                    metadataString(event.Metadata, "archive_scope"),
		ArchiveReferenceMode:            metadataString(event.Metadata, "archive_reference_mode"),
		ArchiveSourcePaths:              metadataStringSlice(event.Metadata, "archive_source_paths"),
		ArchiveForbiddenActions:         metadataStringSlice(event.Metadata, "archive_forbidden_actions"),
		ArchiveRollbackTarget:           metadataString(event.Metadata, "archive_rollback_target"),
		ArchiveFailClosed:               metadataBool(event.Metadata, "archive_fail_closed"),
		Metadata:                        metadata,
	}
}

func normalizeArchiveProofFacts(facts []string) []string {
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

func archiveProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredArchiveProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func archiveProofCompletesAudit(proof ArchiveProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proof.EventID > 0 &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		proofMetadataHasApprovedReviewEvidence("archive_proof", proof.Metadata) &&
		len(archiveProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.ArtifactBytesCopied &&
		!proof.ArtifactBytesDeleted &&
		!proof.HistoricalFilesDeleted &&
		!proof.HistoricalFilesMoved &&
		!proof.ProgressJSONRewritten &&
		!proof.AreaMatrixProtectedPathsTouched &&
		!proof.CommandsRun
}

func archiveProofOptionsBindingPayload(options RecordArchiveProofOptions) map[string]any {
	return archiveProofBindingMetadata(
		options.ArchiveScope,
		options.ArchiveReferenceMode,
		options.ArchiveSourcePaths,
		options.ArchiveForbiddenActions,
		options.ArchiveRollbackTarget,
		options.ArchiveFailClosed,
	)
}

func addArchiveProofBindingMetadata(metadata map[string]any, options RecordArchiveProofOptions) {
	binding := archiveProofOptionsBindingPayload(options)
	for key, value := range binding {
		metadata[key] = value
	}
	blockers := archiveProofOptionsBindingBlockers(options)
	metadata["archive_scope_binding_blockers"] = blockers
	if options.ProofStatus == "complete" && len(blockers) == 0 {
		metadata["archive_scope_binding_status"] = "pass"
	} else if len(blockers) > 0 {
		metadata["archive_scope_binding_status"] = "fail"
	} else {
		metadata["archive_scope_binding_status"] = "not_required"
	}
}

func archiveProofOptionsBindingBlockers(options RecordArchiveProofOptions) []string {
	blockers := []string{}
	if options.ArchiveScope != archiveProofScope {
		blockers = append(blockers, "archive_scope_missing_or_mismatch")
	}
	if options.ArchiveReferenceMode != archiveProofReferenceMode {
		blockers = append(blockers, "archive_reference_mode_missing_or_mismatch")
	}
	if !sameNormalizedStrings(options.ArchiveSourcePaths, requiredArchiveProofSourcePaths) {
		blockers = append(blockers, "archive_source_paths_missing_or_mismatch")
	}
	if !sameNormalizedStrings(options.ArchiveForbiddenActions, requiredArchiveProofForbiddenActions) {
		blockers = append(blockers, "archive_forbidden_actions_missing_or_mismatch")
	}
	if options.ArchiveRollbackTarget != archiveProofRollbackTarget {
		blockers = append(blockers, "archive_rollback_target_missing_or_mismatch")
	}
	if !options.ArchiveFailClosed {
		blockers = append(blockers, "archive_fail_closed_missing")
	}
	return uniqueStrings(blockers)
}

func archiveProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "archive_scope_binding_status") != "pass" {
		blockers = append(blockers, "archive_scope_binding_status_not_pass")
	}
	if metadataString(metadata, "archive_binding_contract") != archiveProofBindingContract {
		blockers = append(blockers, "archive_binding_contract_missing_or_mismatch")
	}
	if metadataString(metadata, "archive_scope") != archiveProofScope {
		blockers = append(blockers, "archive_scope_missing_or_mismatch")
	}
	if metadataString(metadata, "archive_reference_mode") != archiveProofReferenceMode {
		blockers = append(blockers, "archive_reference_mode_missing_or_mismatch")
	}
	sourcePaths := metadataStringSlice(metadata, "archive_source_paths")
	if !sameNormalizedStrings(sourcePaths, requiredArchiveProofSourcePaths) {
		blockers = append(blockers, "archive_source_paths_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "archive_source_paths_hash")) ||
		metadataString(metadata, "archive_source_paths_hash") != archiveProofStringSetHash("archive_source_paths", sourcePaths) {
		blockers = append(blockers, "archive_source_paths_hash_missing_or_mismatch")
	}
	forbiddenActions := metadataStringSlice(metadata, "archive_forbidden_actions")
	if !sameNormalizedStrings(forbiddenActions, requiredArchiveProofForbiddenActions) {
		blockers = append(blockers, "archive_forbidden_actions_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "archive_forbidden_actions_hash")) ||
		metadataString(metadata, "archive_forbidden_actions_hash") != archiveProofStringSetHash("archive_forbidden_actions", forbiddenActions) {
		blockers = append(blockers, "archive_forbidden_actions_hash_missing_or_mismatch")
	}
	if metadataString(metadata, "archive_rollback_target") != archiveProofRollbackTarget {
		blockers = append(blockers, "archive_rollback_target_missing_or_mismatch")
	}
	if !metadataBool(metadata, "archive_fail_closed") {
		blockers = append(blockers, "archive_fail_closed_missing")
	}
	if !looksLikeSHA256(metadataString(metadata, "archive_scope_binding_hash")) ||
		metadataString(metadata, "archive_scope_binding_hash") != archiveProofBindingHash(metadata) {
		blockers = append(blockers, "archive_scope_binding_hash_missing_or_mismatch")
	}
	return uniqueStrings(blockers)
}

func archiveProofCurrentBinding() map[string]any {
	binding := archiveProofBindingMetadata(
		archiveProofScope,
		archiveProofReferenceMode,
		requiredArchiveProofSourcePaths,
		requiredArchiveProofForbiddenActions,
		archiveProofRollbackTarget,
		true,
	)
	binding["archive_scope_binding_status"] = "pass"
	binding["archive_scope_binding_blockers"] = []string{}
	return binding
}

func archiveProofBindingMetadata(scope string, referenceMode string, sourcePaths []string, forbiddenActions []string, rollbackTarget string, failClosed bool) map[string]any {
	sourcePaths = normalizeStringList(sourcePaths)
	forbiddenActions = normalizeStringList(forbiddenActions)
	metadata := map[string]any{
		"archive_binding_contract":       archiveProofBindingContract,
		"archive_scope":                  scope,
		"archive_reference_mode":         referenceMode,
		"archive_source_paths":           sourcePaths,
		"archive_source_paths_hash":      archiveProofStringSetHash("archive_source_paths", sourcePaths),
		"archive_forbidden_actions":      forbiddenActions,
		"archive_forbidden_actions_hash": archiveProofStringSetHash("archive_forbidden_actions", forbiddenActions),
		"archive_rollback_target":        rollbackTarget,
		"archive_fail_closed":            failClosed,
	}
	metadata["archive_binding_hash"] = archiveProofBindingHash(metadata)
	metadata["archive_scope_binding_hash"] = metadata["archive_binding_hash"]
	return metadata
}

func archiveProofCurrentBindingBlockers(proofMetadata map[string]any, currentBinding map[string]any) []string {
	blockers := archiveProofMetadataBindingBlockers(proofMetadata)
	if len(blockers) > 0 {
		return blockers
	}
	currentBlockers := archiveProofMetadataBindingBlockers(currentBinding)
	if len(currentBlockers) > 0 {
		for _, blocker := range currentBlockers {
			blockers = append(blockers, "current_"+blocker)
		}
		return uniqueStrings(append([]string{"archive_scope_current_binding_mismatch"}, blockers...))
	}
	for _, key := range archiveProofCurrentBindingComparisonKeys {
		if !archiveProofBindingValuesEqual(proofMetadata, currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	if len(blockers) > 0 {
		blockers = append([]string{"archive_scope_current_binding_mismatch"}, blockers...)
	}
	return uniqueStrings(blockers)
}

func archiveProofBindingValuesEqual(left map[string]any, right map[string]any, key string) bool {
	switch key {
	case "archive_fail_closed":
		return metadataBool(left, key) == metadataBool(right, key)
	case "archive_source_paths", "archive_forbidden_actions":
		return sameNormalizedStrings(metadataStringSlice(left, key), metadataStringSlice(right, key))
	default:
		return metadataString(left, key) == metadataString(right, key)
	}
}

func archiveProofBindingHash(metadata map[string]any) string {
	payload := map[string]any{
		"archive_binding_contract":       metadataString(metadata, "archive_binding_contract"),
		"archive_scope":                  metadataString(metadata, "archive_scope"),
		"archive_reference_mode":         metadataString(metadata, "archive_reference_mode"),
		"archive_source_paths_hash":      metadataString(metadata, "archive_source_paths_hash"),
		"archive_forbidden_actions_hash": metadataString(metadata, "archive_forbidden_actions_hash"),
		"archive_rollback_target":        metadataString(metadata, "archive_rollback_target"),
		"archive_fail_closed":            metadataBool(metadata, "archive_fail_closed"),
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func archiveProofStringSetHash(kind string, values []string) string {
	payload, err := json.Marshal(map[string]any{
		"archive_binding_contract": archiveProofBindingContract,
		"kind":                     kind,
		"values":                   normalizeStringList(values),
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
