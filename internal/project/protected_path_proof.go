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

type RecordProtectedPathProofOptions struct {
	ProofStatus                string
	Summary                    string
	EvidenceURI                string
	GitStatusOutput            string
	AuthorizedApprovalID       string
	AuthorizedAllowedPaths     []string
	AuthorizedDirtyOutputHash  string
	AuthorizedReviewer         string
	AuthorizedRollbackEvidence string
	IdempotencyKey             string
	Actor                      string
	Reason                     string
	Metadata                   map[string]any
}

type ProtectedPathProof struct {
	Project                         Record
	Status                          string
	ProofStatus                     string
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
	CommandsRun                     bool
	GitStatusRunByCommand           bool
	AreaMatrixProtectedPathsTouched bool
	GitStatusOutputHash             string
	GitStatusOutputLines            int
	Metadata                        map[string]any
}

const protectedPathProofCommandType = "completion.protected_path_proof.record"
const protectedPathProofEventType = "completion.protected_path_proof.recorded"

var allowedProtectedPathProofStatuses = map[string]bool{
	"clean":      true,
	"authorized": true,
	"dirty":      true,
	"blocked":    true,
}

var areaMatrixProtectedPathProofRoots = []string{
	"workflow/README.md",
	".areaflow/status.json",
	"scripts/task_loop/console.py",
	"scripts/dev_tools/cli.py",
	"scripts/task_loop/runner.py",
	"scripts/areaflow_shim.py",
	"workflow/versions",
}

var protectedPathProofEmptyGitStatusOutputHash = func() string {
	sum := sha256.Sum256(nil)
	return hex.EncodeToString(sum[:])
}()

func (s Store) RecordProtectedPathProof(ctx context.Context, record Record, options RecordProtectedPathProofOptions) (ProtectedPathProof, error) {
	options = normalizeRecordProtectedPathProofOptions(options)
	if !allowedProtectedPathProofStatuses[options.ProofStatus] {
		return ProtectedPathProof{}, fmt.Errorf("unsupported protected path proof status %q", options.ProofStatus)
	}
	if options.ProofStatus == "clean" && strings.TrimSpace(options.GitStatusOutput) != "" {
		return ProtectedPathProof{}, fmt.Errorf("clean protected path proof cannot include git status output")
	}
	if err := requireProofEvidenceForStatus("protected path", options.ProofStatus, options.Summary, options.EvidenceURI, "clean", "authorized"); err != nil {
		return ProtectedPathProof{}, err
	}
	if err := validateAuthorizedProtectedPathProofOptions(options); err != nil {
		return ProtectedPathProof{}, err
	}
	if options.ProofStatus == "clean" || options.ProofStatus == "authorized" {
		if blockers := protectedPathProofOptionsBindingBlockers(options); len(blockers) > 0 {
			return ProtectedPathProof{}, fmt.Errorf("%s protected path proof missing protected path binding: %s", options.ProofStatus, strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = protectedPathProofIdempotencyKey(record, options)
	}
	requestHash, err := protectedPathProofRequestHash(record, options)
	if err != nil {
		return ProtectedPathProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProtectedPathProof{}, fmt.Errorf("begin protected path proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, protectedPathProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ProtectedPathProof{}, err
	}
	if !created {
		result, err := loadProtectedPathProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ProtectedPathProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ProtectedPathProof{}, fmt.Errorf("commit idempotent protected path proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildProtectedPathProof(record, options)
	eventID, err := insertProtectedPathProofEvent(ctx, tx, result, options)
	if err != nil {
		return ProtectedPathProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertProtectedPathProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ProtectedPathProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, protectedPathProofCommandType, options.IdempotencyKey, protectedPathProofCommandResponse(result)); err != nil {
		return ProtectedPathProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProtectedPathProof{}, fmt.Errorf("commit protected path proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestProtectedPathProof(ctx context.Context) (ProtectedPathProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		protectedPathProofEventType,
	)
	if err != nil {
		return ProtectedPathProof{}, fmt.Errorf("load latest protected path proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ProtectedPathProof{}, err
	}
	if len(events) == 0 {
		return ProtectedPathProof{}, nil
	}
	return protectedPathProofFromEvent(events[0]), nil
}

func (s Store) LatestProtectedPathProofForProject(ctx context.Context, record Record) (ProtectedPathProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1 AND project_id = $2
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		protectedPathProofEventType,
		record.ID,
	)
	if err != nil {
		return ProtectedPathProof{}, fmt.Errorf("load latest project protected path proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ProtectedPathProof{}, err
	}
	if len(events) == 0 {
		return ProtectedPathProof{}, nil
	}
	return protectedPathProofFromEvent(events[0]), nil
}

func normalizeRecordProtectedPathProofOptions(options RecordProtectedPathProofOptions) RecordProtectedPathProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.GitStatusOutput = strings.TrimRight(options.GitStatusOutput, "\r\n")
	options.AuthorizedApprovalID = strings.TrimSpace(options.AuthorizedApprovalID)
	options.AuthorizedAllowedPaths = normalizeProtectedPathProofAllowedPaths(options.AuthorizedAllowedPaths)
	options.AuthorizedDirtyOutputHash = strings.TrimSpace(options.AuthorizedDirtyOutputHash)
	options.AuthorizedReviewer = strings.TrimSpace(options.AuthorizedReviewer)
	options.AuthorizedRollbackEvidence = strings.TrimSpace(options.AuthorizedRollbackEvidence)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.ProofStatus == "" {
		options.ProofStatus = "clean"
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "record protected path proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func validateAuthorizedProtectedPathProofOptions(options RecordProtectedPathProofOptions) error {
	if options.ProofStatus != "authorized" {
		return nil
	}
	missing := []string{}
	if options.AuthorizedApprovalID == "" {
		missing = append(missing, "approval_id")
	}
	if len(options.AuthorizedAllowedPaths) == 0 {
		missing = append(missing, "allowed_paths")
	}
	if options.AuthorizedDirtyOutputHash == "" {
		missing = append(missing, "dirty_output_hash")
	}
	if options.AuthorizedReviewer == "" {
		missing = append(missing, "reviewer")
	}
	if options.AuthorizedRollbackEvidence == "" {
		missing = append(missing, "rollback_evidence_uri")
	}
	if len(missing) > 0 {
		return fmt.Errorf("authorized protected path proof missing required fields: %s", strings.Join(missing, ","))
	}
	if !protectedPathProofLooksLikeSHA256(options.AuthorizedDirtyOutputHash) {
		return fmt.Errorf("authorized protected path proof dirty_output_hash must be a sha256 hex digest")
	}
	for _, path := range options.AuthorizedAllowedPaths {
		if !protectedPathProofPathIsSafeRelative(path) {
			return fmt.Errorf("authorized protected path proof allowed_path must be a safe relative AreaMatrix path: %s", path)
		}
		if !protectedPathProofPathIsKnownProtectedPath(path) {
			return fmt.Errorf("authorized protected path proof allowed_path is outside the AreaMatrix protected path set: %s", path)
		}
	}
	if options.GitStatusOutput == "" {
		return fmt.Errorf("authorized protected path proof requires git status output")
	}
	outputHash := protectedPathProofOutputHash(options.GitStatusOutput)
	if outputHash != options.AuthorizedDirtyOutputHash {
		return fmt.Errorf("authorized protected path proof dirty_output_hash does not match git status output hash")
	}
	touchedPaths := protectedPathProofTouchedPaths(options.GitStatusOutput)
	if len(touchedPaths) == 0 {
		return fmt.Errorf("authorized protected path proof requires parseable git status output paths")
	}
	for _, path := range touchedPaths {
		if !protectedPathProofPathIsSafeRelative(path) {
			return fmt.Errorf("authorized protected path proof git status path must be a safe relative AreaMatrix path: %s", path)
		}
		if !protectedPathProofPathIsKnownProtectedPath(path) {
			return fmt.Errorf("authorized protected path proof git status path is outside the AreaMatrix protected path set: %s", path)
		}
		if !protectedPathProofPathAllowedBy(path, options.AuthorizedAllowedPaths) {
			return fmt.Errorf("authorized protected path proof git status path is not covered by allowed_path: %s", path)
		}
	}
	return nil
}

func protectedPathProofRequestHash(record Record, options RecordProtectedPathProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":                     protectedPathProofCommandType,
		"project_id":                       record.ID,
		"project_key":                      record.Key,
		"proof_status":                     options.ProofStatus,
		"summary":                          options.Summary,
		"evidence_uri":                     options.EvidenceURI,
		"git_status_output_hash":           protectedPathProofOutputHash(options.GitStatusOutput),
		"authorized_approval_id":           options.AuthorizedApprovalID,
		"authorized_allowed_paths":         options.AuthorizedAllowedPaths,
		"authorized_dirty_output_hash":     options.AuthorizedDirtyOutputHash,
		"authorized_reviewer":              options.AuthorizedReviewer,
		"authorized_rollback_evidence_uri": options.AuthorizedRollbackEvidence,
		"actor":                            options.Actor,
		"reason":                           options.Reason,
		"metadata":                         options.Metadata,
		"protected":                        true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal protected path proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func protectedPathProofIdempotencyKey(record Record, options RecordProtectedPathProofOptions) string {
	hash, err := protectedPathProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.protected_path_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildProtectedPathProof(record Record, options RecordProtectedPathProofOptions) ProtectedPathProof {
	status := "recorded"
	decision := "allowed"
	message := "AreaMatrix protected path proof recorded"
	touched := options.ProofStatus == "authorized"
	if options.ProofStatus == "dirty" || options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "AreaMatrix protected path proof is not clean"
		touched = true
	}
	metadata := map[string]any{}
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	outputHash := protectedPathProofOutputHash(options.GitStatusOutput)
	outputLines := protectedPathProofOutputLines(options.GitStatusOutput)
	metadata["project_key"] = record.Key
	metadata["proof_status"] = options.ProofStatus
	metadata["summary"] = options.Summary
	metadata["evidence_uri"] = options.EvidenceURI
	metadata["git_status_output_hash"] = outputHash
	metadata["git_status_output_lines"] = outputLines
	addProtectedPathProofBindingMetadata(metadata, options)
	if options.ProofStatus == "authorized" {
		metadata["authorized_approval_id"] = options.AuthorizedApprovalID
		metadata["authorized_allowed_paths"] = options.AuthorizedAllowedPaths
		metadata["authorized_dirty_output_hash"] = options.AuthorizedDirtyOutputHash
		metadata["authorized_reviewer"] = options.AuthorizedReviewer
		metadata["authorized_rollback_evidence_uri"] = options.AuthorizedRollbackEvidence
		metadata["authorized_touched_paths"] = protectedPathProofTouchedPaths(options.GitStatusOutput)
	}
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["engine_call_attempted"] = false
	metadata["commands_run"] = false
	metadata["git_status_run_by_command"] = false
	metadata["area_matrix_protected_paths_touched"] = touched
	return ProtectedPathProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		CommandsRun:                     false,
		GitStatusRunByCommand:           false,
		AreaMatrixProtectedPathsTouched: touched,
		GitStatusOutputHash:             outputHash,
		GitStatusOutputLines:            outputLines,
		Metadata:                        metadata,
	}
}

func insertProtectedPathProofEvent(ctx context.Context, tx pgx.Tx, result ProtectedPathProof, options RecordProtectedPathProofOptions) (int64, error) {
	metadata, err := json.Marshal(protectedPathProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal protected path proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Protected path proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		protectedPathProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert protected path proof event: %w", err)
	}
	return eventID, nil
}

func insertProtectedPathProofAuditEvent(ctx context.Context, tx pgx.Tx, result ProtectedPathProof, options RecordProtectedPathProofOptions) (int64, error) {
	metadata, err := json.Marshal(protectedPathProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal protected path proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'protected_path_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		protectedPathProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert protected path proof audit event: %w", err)
	}
	return auditEventID, nil
}

func protectedPathProofEventMetadata(result ProtectedPathProof, options RecordProtectedPathProofOptions) map[string]any {
	metadata := protectedPathProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func protectedPathProofCommandResponse(result ProtectedPathProof) map[string]any {
	return map[string]any{
		"project_key":                           result.Project.Key,
		"status":                                result.Status,
		"proof_status":                          result.ProofStatus,
		"decision":                              result.Decision,
		"message":                               result.Message,
		"event_id":                              result.EventID,
		"audit_event_id":                        result.AuditEventID,
		"idempotency_key":                       result.IdempotencyKey,
		"project_write_attempted":               result.ProjectWriteAttempted,
		"execution_write_attempted":             result.ExecutionWriteAttempted,
		"engine_call_attempted":                 result.EngineCallAttempted,
		"commands_run":                          result.CommandsRun,
		"git_status_run_by_command":             result.GitStatusRunByCommand,
		"area_matrix_protected_paths_touched":   result.AreaMatrixProtectedPathsTouched,
		"git_status_output_hash":                result.GitStatusOutputHash,
		"git_status_output_lines":               result.GitStatusOutputLines,
		"git_status_output_empty":               metadataBool(result.Metadata, "git_status_output_empty"),
		"protected_path_set":                    metadataStringSlice(result.Metadata, "protected_path_set"),
		"protected_path_set_hash":               metadataString(result.Metadata, "protected_path_set_hash"),
		"protected_path_set_count":              metadataInt64(result.Metadata, "protected_path_set_count"),
		"protected_path_proof_binding_status":   metadataString(result.Metadata, "protected_path_proof_binding_status"),
		"protected_path_proof_binding_blockers": metadataStringSlice(result.Metadata, "protected_path_proof_binding_blockers"),
		"summary":                               metadataString(result.Metadata, "summary"),
		"evidence_uri":                          metadataString(result.Metadata, "evidence_uri"),
		"metadata":                              result.Metadata,
	}
}

func loadProtectedPathProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ProtectedPathProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, protectedPathProofCommandType, idempotencyKey)
	if err != nil {
		return ProtectedPathProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return ProtectedPathProof{
		Project:                         record,
		Status:                          metadataString(response, "status"),
		ProofStatus:                     metadataString(response, "proof_status"),
		Decision:                        metadataString(response, "decision"),
		Message:                         metadataString(response, "message"),
		EventID:                         metadataInt64(response, "event_id"),
		AuditEventID:                    metadataInt64(response, "audit_event_id"),
		IdempotencyKey:                  idempotencyKey,
		ProjectWriteAttempted:           metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:             metadataBool(response, "engine_call_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		GitStatusRunByCommand:           metadataBool(response, "git_status_run_by_command"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		GitStatusOutputHash:             metadataString(response, "git_status_output_hash"),
		GitStatusOutputLines:            int(metadataInt64(response, "git_status_output_lines")),
		Metadata:                        metadata,
	}, nil
}

func protectedPathProofFromEvent(event EventRecord) ProtectedPathProof {
	metadata := protectedPathProofMetadataFromEvent(event.Metadata)
	return ProtectedPathProof{
		Project:                         Record{ID: event.ProjectID, Key: metadataString(event.Metadata, "project_key")},
		Status:                          metadataString(event.Metadata, "status"),
		ProofStatus:                     metadataString(event.Metadata, "proof_status"),
		Decision:                        metadataString(event.Metadata, "decision"),
		Message:                         metadataString(event.Metadata, "message"),
		EventID:                         event.ID,
		AuditEventID:                    metadataInt64(event.Metadata, "audit_event_id"),
		IdempotencyKey:                  metadataString(event.Metadata, "idempotency_key"),
		CreatedAt:                       event.CreatedAt,
		ProjectWriteAttempted:           metadataBool(event.Metadata, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(event.Metadata, "execution_write_attempted"),
		EngineCallAttempted:             metadataBool(event.Metadata, "engine_call_attempted"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		GitStatusRunByCommand:           metadataBool(event.Metadata, "git_status_run_by_command"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		GitStatusOutputHash:             metadataString(event.Metadata, "git_status_output_hash"),
		GitStatusOutputLines:            int(metadataInt64(event.Metadata, "git_status_output_lines")),
		Metadata:                        metadata,
	}
}

func protectedPathProofMetadataFromEvent(eventMetadata map[string]any) map[string]any {
	metadata := map[string]any{}
	if nested, ok := eventMetadata["metadata"].(map[string]any); ok {
		for key, value := range nested {
			metadata[key] = value
		}
	}
	for key, value := range eventMetadata {
		if key == "metadata" {
			continue
		}
		if _, exists := metadata[key]; !exists {
			metadata[key] = value
		}
	}
	return metadata
}

func protectedPathProofOutputHash(output string) string {
	if strings.TrimSpace(output) == "" {
		return protectedPathProofEmptyGitStatusOutputHash
	}
	sum := sha256.Sum256([]byte(output))
	return hex.EncodeToString(sum[:])
}

func protectedPathProofOutputLines(output string) int {
	output = strings.TrimSpace(output)
	if output == "" {
		return 0
	}
	return len(strings.Split(output, "\n"))
}

func protectedPathProofSet() []string {
	return normalizeProtectedPathProofAllowedPaths(areaMatrixProtectedPathProofRoots)
}

func protectedPathProofSetHash() string {
	payload, err := json.Marshal(map[string]any{
		"git_status_mode": "short",
		"protected_paths": protectedPathProofSet(),
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func addProtectedPathProofBindingMetadata(metadata map[string]any, options RecordProtectedPathProofOptions) {
	protectedSet := protectedPathProofSet()
	metadata["protected_path_set"] = protectedSet
	metadata["protected_path_set_hash"] = protectedPathProofSetHash()
	metadata["protected_path_set_count"] = int64(len(protectedSet))
	metadata["git_status_output_empty"] = strings.TrimSpace(options.GitStatusOutput) == ""
	blockers := protectedPathProofOptionsBindingBlockers(options)
	metadata["protected_path_proof_binding_blockers"] = blockers
	if (options.ProofStatus == "clean" || options.ProofStatus == "authorized") && len(blockers) == 0 {
		metadata["protected_path_proof_binding_status"] = "pass"
	} else if len(blockers) > 0 {
		metadata["protected_path_proof_binding_status"] = "fail"
	} else {
		metadata["protected_path_proof_binding_status"] = "not_required"
	}
}

func protectedPathProofOptionsBindingBlockers(options RecordProtectedPathProofOptions) []string {
	metadata := map[string]any{
		"proof_status":                        options.ProofStatus,
		"protected_path_proof_binding_status": "pass",
		"git_status_output_hash":              protectedPathProofOutputHash(options.GitStatusOutput),
		"git_status_output_lines":             int64(protectedPathProofOutputLines(options.GitStatusOutput)),
		"git_status_output_empty":             strings.TrimSpace(options.GitStatusOutput) == "",
		"protected_path_set":                  protectedPathProofSet(),
		"protected_path_set_hash":             protectedPathProofSetHash(),
		"protected_path_set_count":            int64(len(protectedPathProofSet())),
		"authorized_approval_id":              options.AuthorizedApprovalID,
		"authorized_allowed_paths":            options.AuthorizedAllowedPaths,
		"authorized_dirty_output_hash":        options.AuthorizedDirtyOutputHash,
		"authorized_reviewer":                 options.AuthorizedReviewer,
		"authorized_rollback_evidence_uri":    options.AuthorizedRollbackEvidence,
		"authorized_touched_paths":            protectedPathProofTouchedPaths(options.GitStatusOutput),
	}
	return protectedPathProofMetadataBindingBlockers(metadata)
}

func protectedPathProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "protected_path_proof_binding_status") != "pass" {
		blockers = append(blockers, "protected_path_proof_binding_status_not_pass")
	}
	if !sameNormalizedStrings(metadataStringSlice(metadata, "protected_path_set"), protectedPathProofSet()) {
		blockers = append(blockers, "protected_path_set_missing_or_mismatch")
	}
	if metadataString(metadata, "protected_path_set_hash") != protectedPathProofSetHash() {
		blockers = append(blockers, "protected_path_set_hash_missing_or_mismatch")
	}
	if metadataInt64(metadata, "protected_path_set_count") != int64(len(protectedPathProofSet())) {
		blockers = append(blockers, "protected_path_set_count_missing_or_mismatch")
	}
	proofStatus := metadataString(metadata, "proof_status")
	switch proofStatus {
	case "clean":
		if metadataString(metadata, "git_status_output_hash") != protectedPathProofEmptyGitStatusOutputHash {
			blockers = append(blockers, "clean_git_status_output_hash_missing_or_mismatch")
		}
		if metadataInt64(metadata, "git_status_output_lines") != 0 {
			blockers = append(blockers, "clean_git_status_output_lines_nonzero")
		}
		if !metadataBool(metadata, "git_status_output_empty") {
			blockers = append(blockers, "clean_git_status_output_not_empty")
		}
	case "authorized":
		if !protectedPathProofAuthorizedMetadataComplete(metadata) {
			blockers = append(blockers, "authorized_metadata_incomplete")
		}
		if metadataString(metadata, "git_status_output_hash") == "" ||
			metadataString(metadata, "git_status_output_hash") != metadataString(metadata, "authorized_dirty_output_hash") {
			blockers = append(blockers, "authorized_git_status_output_hash_mismatch")
		}
		if metadataInt64(metadata, "git_status_output_lines") == 0 {
			blockers = append(blockers, "authorized_git_status_output_lines_missing")
		}
		if metadataBool(metadata, "git_status_output_empty") {
			blockers = append(blockers, "authorized_git_status_output_empty")
		}
	}
	return uniqueStrings(blockers)
}

func normalizeProtectedPathProofAllowedPaths(paths []string) []string {
	seen := map[string]bool{}
	normalized := []string{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		path = strings.TrimPrefix(path, "./")
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		normalized = append(normalized, path)
	}
	sort.Strings(normalized)
	return normalized
}

func protectedPathProofLooksLikeSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func protectedPathProofAuthorizedMetadataComplete(metadata map[string]any) bool {
	allowedPaths := metadataStringSlice(metadata, "authorized_allowed_paths")
	touchedPaths := metadataStringSlice(metadata, "authorized_touched_paths")
	if strings.TrimSpace(metadataString(metadata, "authorized_approval_id")) == "" ||
		len(allowedPaths) == 0 ||
		!protectedPathProofLooksLikeSHA256(strings.TrimSpace(metadataString(metadata, "authorized_dirty_output_hash"))) ||
		strings.TrimSpace(metadataString(metadata, "authorized_reviewer")) == "" ||
		strings.TrimSpace(metadataString(metadata, "authorized_rollback_evidence_uri")) == "" ||
		len(touchedPaths) == 0 {
		return false
	}
	for _, path := range allowedPaths {
		if !protectedPathProofPathIsSafeRelative(path) || !protectedPathProofPathIsKnownProtectedPath(path) {
			return false
		}
	}
	for _, path := range touchedPaths {
		if !protectedPathProofPathIsSafeRelative(path) ||
			!protectedPathProofPathIsKnownProtectedPath(path) ||
			!protectedPathProofPathAllowedBy(path, allowedPaths) {
			return false
		}
	}
	return true
}

func protectedPathProofTouchedPaths(output string) []string {
	seen := map[string]bool{}
	paths := []string{}
	for _, line := range strings.Split(strings.TrimRight(output, "\r\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		line = strings.TrimRight(line, " \t\r")
		for _, path := range protectedPathProofTouchedPathsFromLine(line) {
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func protectedPathProofTouchedPathsFromLine(line string) []string {
	if len(line) < 4 {
		return nil
	}
	pathPart := strings.TrimSpace(line[3:])
	if pathPart == "" {
		return nil
	}
	rawPaths := []string{pathPart}
	if strings.Contains(pathPart, " -> ") {
		rawPaths = strings.Split(pathPart, " -> ")
	}
	paths := []string{}
	for _, path := range rawPaths {
		path = strings.TrimSpace(path)
		path = strings.Trim(path, `"`)
		path = strings.TrimPrefix(path, "./")
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func protectedPathProofPathAllowedBy(path string, allowedPaths []string) bool {
	for _, allowed := range allowedPaths {
		if path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

func protectedPathProofPathIsSafeRelative(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~") {
		return false
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func protectedPathProofPathIsKnownProtectedPath(path string) bool {
	for _, root := range areaMatrixProtectedPathProofRoots {
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}
