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

type RecordSourceAlignmentProofOptions struct {
	ProofStatus            string
	Facts                  []string
	Summary                string
	EvidenceURI            string
	SourceAlignmentBinding map[string]any
	IdempotencyKey         string
	Actor                  string
	Reason                 string
	Metadata               map[string]any
}

type SourceAlignmentProof struct {
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
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

const sourceAlignmentProofCommandType = "completion.source_alignment_proof.record"
const sourceAlignmentProofEventType = "completion.source_alignment_proof.recorded"

var allowedSourceAlignmentProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredSourceAlignmentProofFacts = []string{
	"zero_to_hundred_phases_aligned",
	"v1_and_v1x_boundaries_consistent",
	"preview_only_not_claimed_as_apply",
	"implemented_scoped_not_claimed_as_real_cutover",
	"deferred_high_risk_capabilities_have_contracts",
	"master_plan_roadmap_phase_backlog_gap_audit_cross_references_current",
}

func (s Store) RecordSourceAlignmentProof(ctx context.Context, record Record, options RecordSourceAlignmentProofOptions) (SourceAlignmentProof, error) {
	options = normalizeRecordSourceAlignmentProofOptions(options)
	if !allowedSourceAlignmentProofStatuses[options.ProofStatus] {
		return SourceAlignmentProof{}, fmt.Errorf("unsupported source alignment proof status %q", options.ProofStatus)
	}
	missingFacts := sourceAlignmentProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return SourceAlignmentProof{}, fmt.Errorf("complete source alignment proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("source alignment", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return SourceAlignmentProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := sourceAlignmentProofOptionsBindingBlockers(options); len(blockers) > 0 {
			return SourceAlignmentProof{}, fmt.Errorf("complete source alignment proof missing source alignment binding: %s", strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = sourceAlignmentProofIdempotencyKey(record, options)
	}
	requestHash, err := sourceAlignmentProofRequestHash(record, options)
	if err != nil {
		return SourceAlignmentProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SourceAlignmentProof{}, fmt.Errorf("begin source alignment proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, sourceAlignmentProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return SourceAlignmentProof{}, err
	}
	if !created {
		result, err := loadSourceAlignmentProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return SourceAlignmentProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return SourceAlignmentProof{}, fmt.Errorf("commit idempotent source alignment proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildSourceAlignmentProof(record, options)
	eventID, err := insertSourceAlignmentProofEvent(ctx, tx, result, options)
	if err != nil {
		return SourceAlignmentProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertSourceAlignmentProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return SourceAlignmentProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, sourceAlignmentProofCommandType, options.IdempotencyKey, sourceAlignmentProofCommandResponse(result)); err != nil {
		return SourceAlignmentProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SourceAlignmentProof{}, fmt.Errorf("commit source alignment proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestSourceAlignmentProof(ctx context.Context) (SourceAlignmentProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		sourceAlignmentProofEventType,
	)
	if err != nil {
		return SourceAlignmentProof{}, fmt.Errorf("load latest source alignment proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return SourceAlignmentProof{}, err
	}
	if len(events) == 0 {
		return SourceAlignmentProof{}, nil
	}
	return sourceAlignmentProofFromEvent(events[0]), nil
}

func (s Store) LatestSourceAlignmentProofForProject(ctx context.Context, record Record) (SourceAlignmentProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, sourceAlignmentProofEventType)
	if err != nil {
		return SourceAlignmentProof{}, fmt.Errorf("load latest project source alignment proof: %w", err)
	}
	if !ok {
		return SourceAlignmentProof{}, nil
	}
	return sourceAlignmentProofFromEvent(event), nil
}

func normalizeRecordSourceAlignmentProofOptions(options RecordSourceAlignmentProofOptions) RecordSourceAlignmentProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeSourceAlignmentProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
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
		options.Reason = "record source alignment proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.SourceAlignmentBinding == nil {
		options.SourceAlignmentBinding = map[string]any{}
	}
	return options
}

func sourceAlignmentProofRequestHash(record Record, options RecordSourceAlignmentProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     sourceAlignmentProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeSourceAlignmentProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"binding":          options.SourceAlignmentBinding,
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal source alignment proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func sourceAlignmentProofIdempotencyKey(record Record, options RecordSourceAlignmentProofOptions) string {
	hash, err := sourceAlignmentProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.source_alignment_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildSourceAlignmentProof(record Record, options RecordSourceAlignmentProofOptions) SourceAlignmentProof {
	facts := normalizeSourceAlignmentProofFacts(options.Facts)
	missingFacts := sourceAlignmentProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "source alignment proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "source alignment proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "source alignment proof is incomplete"
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
	addSourceAlignmentProofBindingMetadata(metadata, options)
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["commands_run"] = false
	metadata["docs_written"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return SourceAlignmentProof{
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
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertSourceAlignmentProofEvent(ctx context.Context, tx pgx.Tx, result SourceAlignmentProof, options RecordSourceAlignmentProofOptions) (int64, error) {
	metadata, err := json.Marshal(sourceAlignmentProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal source alignment proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Source alignment proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		sourceAlignmentProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert source alignment proof event: %w", err)
	}
	return eventID, nil
}

func insertSourceAlignmentProofAuditEvent(ctx context.Context, tx pgx.Tx, result SourceAlignmentProof, options RecordSourceAlignmentProofOptions) (int64, error) {
	metadata, err := json.Marshal(sourceAlignmentProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal source alignment proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'source_alignment_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		sourceAlignmentProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert source alignment proof audit event: %w", err)
	}
	return auditEventID, nil
}

func sourceAlignmentProofEventMetadata(result SourceAlignmentProof, options RecordSourceAlignmentProofOptions) map[string]any {
	metadata := sourceAlignmentProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func sourceAlignmentProofCommandResponse(result SourceAlignmentProof) map[string]any {
	response := map[string]any{
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
		"commands_run":                        result.CommandsRun,
		"docs_written":                        result.DocsWritten,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"source_alignment_binding_status":     metadataString(result.Metadata, "source_alignment_binding_status"),
		"source_alignment_binding_blockers":   metadataStringSlice(result.Metadata, "source_alignment_binding_blockers"),
		"source_alignment_source_paths":       metadataStringSlice(result.Metadata, "source_alignment_source_paths"),
		"source_alignment_source_hashes":      sourceAlignmentMetadataStringMap(result.Metadata, "source_alignment_source_hashes"),
		"source_alignment_source_set_hash":    metadataString(result.Metadata, "source_alignment_source_set_hash"),
		"summary":                             metadataString(result.Metadata, "summary"),
		"evidence_uri":                        metadataString(result.Metadata, "evidence_uri"),
		"metadata":                            result.Metadata,
	}
	for _, key := range sourceAlignmentBindingComparisonKeys {
		if value, ok := result.Metadata[key]; ok {
			response[key] = value
		}
	}
	return response
}

func loadSourceAlignmentProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (SourceAlignmentProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, sourceAlignmentProofCommandType, idempotencyKey)
	if err != nil {
		return SourceAlignmentProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return SourceAlignmentProof{
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
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func sourceAlignmentProofFromEvent(event EventRecord) SourceAlignmentProof {
	return SourceAlignmentProof{
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
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        event.Metadata,
	}
}

func normalizeSourceAlignmentProofFacts(facts []string) []string {
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

func sourceAlignmentProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredSourceAlignmentProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func sourceAlignmentProofCompletesAudit(proof SourceAlignmentProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(sourceAlignmentProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.CommandsRun &&
		!proof.DocsWritten &&
		!proof.AreaMatrixProtectedPathsTouched
}
