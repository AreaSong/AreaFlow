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

type RecordValidationProofOptions struct {
	ProofStatus          string
	Facts                []string
	Summary              string
	EvidenceURI          string
	ValidationCommands   []string
	ValidationResultHash string
	ValidationStartedAt  string
	ValidationFinishedAt string
	ValidationScope      string
	IdempotencyKey       string
	Actor                string
	Reason               string
	Metadata             map[string]any
}

type ValidationProof struct {
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
	EngineCallAttempted             bool
	CommandsRun                     bool
	SmokeRunAttempted               bool
	WebBuildRunByCommand            bool
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

const validationProofCommandType = "completion.validation_proof.record"
const validationProofEventType = "completion.validation_proof.recorded"

var allowedValidationProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredValidationProofFacts = []string{
	"go_test_passed",
	"go_build_passed",
	"web_build_passed",
	"git_diff_check_passed",
	"v1_stable_fixture_smoke_passed",
	"web_smoke_passed",
	"project_isolation_smoke_passed",
	"completion_proof_smoke_passed",
	"validation_did_not_touch_areamatrix_protected_paths",
}

func (s Store) RecordValidationProof(ctx context.Context, record Record, options RecordValidationProofOptions) (ValidationProof, error) {
	options = normalizeRecordValidationProofOptions(options)
	if !allowedValidationProofStatuses[options.ProofStatus] {
		return ValidationProof{}, fmt.Errorf("unsupported validation proof status %q", options.ProofStatus)
	}
	missingFacts := validationProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return ValidationProof{}, fmt.Errorf("complete validation proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("validation", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return ValidationProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := validationProofEvidenceBindingBlockers(options); len(blockers) > 0 {
			return ValidationProof{}, fmt.Errorf("complete validation proof missing validation evidence binding: %s", strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = validationProofIdempotencyKey(record, options)
	}
	requestHash, err := validationProofRequestHash(record, options)
	if err != nil {
		return ValidationProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ValidationProof{}, fmt.Errorf("begin validation proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, validationProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ValidationProof{}, err
	}
	if !created {
		result, err := loadValidationProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ValidationProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ValidationProof{}, fmt.Errorf("commit idempotent validation proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildValidationProof(record, options)
	eventID, err := insertValidationProofEvent(ctx, tx, result, options)
	if err != nil {
		return ValidationProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertValidationProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ValidationProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, validationProofCommandType, options.IdempotencyKey, validationProofCommandResponse(result)); err != nil {
		return ValidationProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ValidationProof{}, fmt.Errorf("commit validation proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestValidationProof(ctx context.Context) (ValidationProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		validationProofEventType,
	)
	if err != nil {
		return ValidationProof{}, fmt.Errorf("load latest validation proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ValidationProof{}, err
	}
	if len(events) == 0 {
		return ValidationProof{}, nil
	}
	return validationProofFromEvent(events[0]), nil
}

func (s Store) LatestValidationProofForProject(ctx context.Context, record Record) (ValidationProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, validationProofEventType)
	if err != nil {
		return ValidationProof{}, fmt.Errorf("load latest project validation proof: %w", err)
	}
	if !ok {
		return ValidationProof{}, nil
	}
	return validationProofFromEvent(event), nil
}

func normalizeRecordValidationProofOptions(options RecordValidationProofOptions) RecordValidationProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeValidationProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.ValidationCommands = normalizeValidationProofCommands(options.ValidationCommands)
	options.ValidationResultHash = strings.TrimSpace(options.ValidationResultHash)
	options.ValidationStartedAt = strings.TrimSpace(options.ValidationStartedAt)
	options.ValidationFinishedAt = strings.TrimSpace(options.ValidationFinishedAt)
	options.ValidationScope = strings.TrimSpace(options.ValidationScope)
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
		options.Reason = "record validation proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func validationProofRequestHash(record Record, options RecordValidationProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":           validationProofCommandType,
		"project_id":             record.ID,
		"project_key":            record.Key,
		"proof_status":           options.ProofStatus,
		"facts":                  normalizeValidationProofFacts(options.Facts),
		"summary":                options.Summary,
		"evidence_uri":           options.EvidenceURI,
		"validation_commands":    normalizeValidationProofCommands(options.ValidationCommands),
		"validation_result_hash": options.ValidationResultHash,
		"validation_started_at":  options.ValidationStartedAt,
		"validation_finished_at": options.ValidationFinishedAt,
		"validation_scope":       options.ValidationScope,
		"actor":                  options.Actor,
		"reason":                 options.Reason,
		"metadata":               options.Metadata,
		"protected":              true,
		"no_project_write":       true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal validation proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func validationProofIdempotencyKey(record Record, options RecordValidationProofOptions) string {
	hash, err := validationProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.validation_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildValidationProof(record Record, options RecordValidationProofOptions) ValidationProof {
	facts := normalizeValidationProofFacts(options.Facts)
	missingFacts := validationProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "validation proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "validation proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "validation proof is incomplete"
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
	metadata["validation_commands"] = options.ValidationCommands
	metadata["validation_command_count"] = len(options.ValidationCommands)
	metadata["validation_result_hash"] = options.ValidationResultHash
	metadata["validation_started_at"] = options.ValidationStartedAt
	metadata["validation_finished_at"] = options.ValidationFinishedAt
	metadata["validation_scope"] = options.ValidationScope
	if options.ProofStatus == "complete" {
		metadata["validation_evidence_binding_status"] = "pass"
	} else {
		metadata["validation_evidence_binding_status"] = "not_required"
	}
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["engine_call_attempted"] = false
	metadata["commands_run"] = false
	metadata["smoke_run_attempted"] = false
	metadata["web_build_run_by_command"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return ValidationProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		CommandsRun:                     false,
		SmokeRunAttempted:               false,
		WebBuildRunByCommand:            false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertValidationProofEvent(ctx context.Context, tx pgx.Tx, result ValidationProof, options RecordValidationProofOptions) (int64, error) {
	metadata, err := json.Marshal(validationProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal validation proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Validation proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		validationProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert validation proof event: %w", err)
	}
	return eventID, nil
}

func insertValidationProofAuditEvent(ctx context.Context, tx pgx.Tx, result ValidationProof, options RecordValidationProofOptions) (int64, error) {
	metadata, err := json.Marshal(validationProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal validation proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'validation_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		validationProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert validation proof audit event: %w", err)
	}
	return auditEventID, nil
}

func validationProofEventMetadata(result ValidationProof, options RecordValidationProofOptions) map[string]any {
	metadata := validationProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func validationProofCommandResponse(result ValidationProof) map[string]any {
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
		"engine_call_attempted":               result.EngineCallAttempted,
		"commands_run":                        result.CommandsRun,
		"smoke_run_attempted":                 result.SmokeRunAttempted,
		"web_build_run_by_command":            result.WebBuildRunByCommand,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"summary":                             metadataString(result.Metadata, "summary"),
		"evidence_uri":                        metadataString(result.Metadata, "evidence_uri"),
		"validation_evidence_binding_status":  metadataString(result.Metadata, "validation_evidence_binding_status"),
		"validation_commands":                 metadataStringSlice(result.Metadata, "validation_commands"),
		"validation_command_count":            metadataInt64(result.Metadata, "validation_command_count"),
		"validation_result_hash":              metadataString(result.Metadata, "validation_result_hash"),
		"validation_started_at":               metadataString(result.Metadata, "validation_started_at"),
		"validation_finished_at":              metadataString(result.Metadata, "validation_finished_at"),
		"validation_scope":                    metadataString(result.Metadata, "validation_scope"),
		"metadata":                            result.Metadata,
	}
}

func loadValidationProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ValidationProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, validationProofCommandType, idempotencyKey)
	if err != nil {
		return ValidationProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return ValidationProof{
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
		EngineCallAttempted:             metadataBool(response, "engine_call_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		SmokeRunAttempted:               metadataBool(response, "smoke_run_attempted"),
		WebBuildRunByCommand:            metadataBool(response, "web_build_run_by_command"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func validationProofFromEvent(event EventRecord) ValidationProof {
	return ValidationProof{
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
		EngineCallAttempted:             metadataBool(event.Metadata, "engine_call_attempted"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		SmokeRunAttempted:               metadataBool(event.Metadata, "smoke_run_attempted"),
		WebBuildRunByCommand:            metadataBool(event.Metadata, "web_build_run_by_command"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        event.Metadata,
	}
}

func normalizeValidationProofFacts(facts []string) []string {
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

func normalizeValidationProofCommands(commands []string) []string {
	seen := map[string]bool{}
	normalized := []string{}
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" || seen[command] {
			continue
		}
		seen[command] = true
		normalized = append(normalized, command)
	}
	return normalized
}

func validationProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredValidationProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func validationProofEvidenceBindingBlockers(options RecordValidationProofOptions) []string {
	blockers := []string{}
	if len(options.ValidationCommands) == 0 {
		blockers = append(blockers, "validation_commands_missing")
	}
	if !isSHA256Hex(options.ValidationResultHash) {
		blockers = append(blockers, "validation_result_hash_invalid")
	}
	startedAt, startedOK := parseValidationProofTimestamp(options.ValidationStartedAt)
	if !startedOK {
		blockers = append(blockers, "validation_started_at_invalid")
	}
	finishedAt, finishedOK := parseValidationProofTimestamp(options.ValidationFinishedAt)
	if !finishedOK {
		blockers = append(blockers, "validation_finished_at_invalid")
	}
	if startedOK && finishedOK && finishedAt.Before(startedAt) {
		blockers = append(blockers, "validation_finished_before_started")
	}
	if options.ValidationScope == "" {
		blockers = append(blockers, "validation_scope_missing")
	}
	return uniqueStrings(blockers)
}

func validationProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "validation_evidence_binding_status") != "pass" {
		blockers = append(blockers, "validation_evidence_binding_status_not_pass")
	}
	if len(metadataStringSlice(metadata, "validation_commands")) == 0 {
		blockers = append(blockers, "validation_commands_missing")
	}
	if !isSHA256Hex(metadataString(metadata, "validation_result_hash")) {
		blockers = append(blockers, "validation_result_hash_invalid")
	}
	startedAt, startedOK := parseValidationProofTimestamp(metadataString(metadata, "validation_started_at"))
	if !startedOK {
		blockers = append(blockers, "validation_started_at_invalid")
	}
	finishedAt, finishedOK := parseValidationProofTimestamp(metadataString(metadata, "validation_finished_at"))
	if !finishedOK {
		blockers = append(blockers, "validation_finished_at_invalid")
	}
	if startedOK && finishedOK && finishedAt.Before(startedAt) {
		blockers = append(blockers, "validation_finished_before_started")
	}
	if metadataString(metadata, "validation_scope") == "" {
		blockers = append(blockers, "validation_scope_missing")
	}
	return uniqueStrings(blockers)
}

func parseValidationProofTimestamp(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	return parsed, err == nil
}

func isSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func validationProofCompletesAudit(proof ValidationProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(validationProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.EngineCallAttempted &&
		!proof.CommandsRun &&
		!proof.SmokeRunAttempted &&
		!proof.WebBuildRunByCommand &&
		!proof.AreaMatrixProtectedPathsTouched
}
