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

type RecordSecurityClosureProofOptions struct {
	ProofStatus            string
	Facts                  []string
	Summary                string
	EvidenceURI            string
	SecurityClosureBinding map[string]any
	IdempotencyKey         string
	Actor                  string
	Reason                 string
	Metadata               map[string]any
}

type SecurityClosureProof struct {
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
	AuthorizationChanged            bool
	SecretPlaintextRead             bool
	RemoteWorkerCredentialsIssued   bool
	CommandsRun                     bool
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

const securityClosureProofCommandType = "completion.security_closure_proof.record"
const securityClosureProofEventType = "completion.security_closure_proof.recorded"

var allowedSecurityClosureProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredSecurityClosureProofFacts = []string{
	"project_key_isolation_covers_workflow_run_lease_artifact_secret_audit",
	"global_id_route_guard_project_key_visibility_proven",
	"permission_doctor_default_read_only_deny_first_passed",
	"audit_coverage_covers_enabled_capabilities",
	"auth_team_token_secret_remote_worker_remain_readiness_only",
	"no_forbidden_v1_security_capability_opened",
}

func (s Store) RecordSecurityClosureProof(ctx context.Context, record Record, options RecordSecurityClosureProofOptions) (SecurityClosureProof, error) {
	options = normalizeRecordSecurityClosureProofOptions(options)
	if !allowedSecurityClosureProofStatuses[options.ProofStatus] {
		return SecurityClosureProof{}, fmt.Errorf("unsupported security closure proof status %q", options.ProofStatus)
	}
	missingFacts := securityClosureProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return SecurityClosureProof{}, fmt.Errorf("complete security closure proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("security closure", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return SecurityClosureProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := securityClosureProofOptionsBindingBlockers(options); len(blockers) > 0 {
			return SecurityClosureProof{}, fmt.Errorf("complete security closure proof missing security closure binding: %s", strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = securityClosureProofIdempotencyKey(record, options)
	}
	requestHash, err := securityClosureProofRequestHash(record, options)
	if err != nil {
		return SecurityClosureProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SecurityClosureProof{}, fmt.Errorf("begin security closure proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, securityClosureProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return SecurityClosureProof{}, err
	}
	if !created {
		result, err := loadSecurityClosureProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return SecurityClosureProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return SecurityClosureProof{}, fmt.Errorf("commit idempotent security closure proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildSecurityClosureProof(record, options)
	eventID, err := insertSecurityClosureProofEvent(ctx, tx, result, options)
	if err != nil {
		return SecurityClosureProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertSecurityClosureProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return SecurityClosureProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, securityClosureProofCommandType, options.IdempotencyKey, securityClosureProofCommandResponse(result)); err != nil {
		return SecurityClosureProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SecurityClosureProof{}, fmt.Errorf("commit security closure proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestSecurityClosureProof(ctx context.Context) (SecurityClosureProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		securityClosureProofEventType,
	)
	if err != nil {
		return SecurityClosureProof{}, fmt.Errorf("load latest security closure proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return SecurityClosureProof{}, err
	}
	if len(events) == 0 {
		return SecurityClosureProof{}, nil
	}
	return securityClosureProofFromEvent(events[0]), nil
}

func (s Store) LatestSecurityClosureProofForProject(ctx context.Context, record Record) (SecurityClosureProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, securityClosureProofEventType)
	if err != nil {
		return SecurityClosureProof{}, fmt.Errorf("load latest project security closure proof: %w", err)
	}
	if !ok {
		return SecurityClosureProof{}, nil
	}
	return securityClosureProofFromEvent(event), nil
}

func normalizeRecordSecurityClosureProofOptions(options RecordSecurityClosureProofOptions) RecordSecurityClosureProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeSecurityClosureProofFacts(options.Facts)
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
		options.Reason = "record security closure proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.SecurityClosureBinding == nil {
		options.SecurityClosureBinding = map[string]any{}
	}
	return options
}

func securityClosureProofRequestHash(record Record, options RecordSecurityClosureProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     securityClosureProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeSecurityClosureProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"binding":          options.SecurityClosureBinding,
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal security closure proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func securityClosureProofIdempotencyKey(record Record, options RecordSecurityClosureProofOptions) string {
	hash, err := securityClosureProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.security_closure_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildSecurityClosureProof(record Record, options RecordSecurityClosureProofOptions) SecurityClosureProof {
	facts := normalizeSecurityClosureProofFacts(options.Facts)
	missingFacts := securityClosureProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "security closure proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "security closure proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "security closure proof is incomplete"
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
	addSecurityClosureProofBindingMetadata(metadata, options)
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["authorization_changed"] = false
	metadata["secret_plaintext_read"] = false
	metadata["remote_worker_credentials_issued"] = false
	metadata["commands_run"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return SecurityClosureProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		AuthorizationChanged:            false,
		SecretPlaintextRead:             false,
		RemoteWorkerCredentialsIssued:   false,
		CommandsRun:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertSecurityClosureProofEvent(ctx context.Context, tx pgx.Tx, result SecurityClosureProof, options RecordSecurityClosureProofOptions) (int64, error) {
	metadata, err := json.Marshal(securityClosureProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal security closure proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Security closure proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		securityClosureProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert security closure proof event: %w", err)
	}
	return eventID, nil
}

func insertSecurityClosureProofAuditEvent(ctx context.Context, tx pgx.Tx, result SecurityClosureProof, options RecordSecurityClosureProofOptions) (int64, error) {
	metadata, err := json.Marshal(securityClosureProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal security closure proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'security_closure_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		securityClosureProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert security closure proof audit event: %w", err)
	}
	return auditEventID, nil
}

func securityClosureProofEventMetadata(result SecurityClosureProof, options RecordSecurityClosureProofOptions) map[string]any {
	metadata := securityClosureProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func securityClosureProofCommandResponse(result SecurityClosureProof) map[string]any {
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
		"authorization_changed":               result.AuthorizationChanged,
		"secret_plaintext_read":               result.SecretPlaintextRead,
		"remote_worker_credentials_issued":    result.RemoteWorkerCredentialsIssued,
		"commands_run":                        result.CommandsRun,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"security_closure_binding_status":     metadataString(result.Metadata, "security_closure_binding_status"),
		"security_closure_binding_blockers":   metadataStringSlice(result.Metadata, "security_closure_binding_blockers"),
		"security_closure_binding_hash":       metadataString(result.Metadata, "security_closure_binding_hash"),
		"security_boundary_status":            metadataString(result.Metadata, "security_boundary_status"),
		"permission_doctor_status":            metadataString(result.Metadata, "permission_doctor_status"),
		"audit_coverage_status":               metadataString(result.Metadata, "audit_coverage_status"),
		"summary":                             metadataString(result.Metadata, "summary"),
		"evidence_uri":                        metadataString(result.Metadata, "evidence_uri"),
		"metadata":                            result.Metadata,
	}
	response["security_closure_binding_blockers"] = metadataStringSlice(result.Metadata, "security_closure_binding_blockers")
	for _, key := range securityClosureBindingComparisonKeys {
		if value, ok := result.Metadata[key]; ok {
			response[key] = value
		}
	}
	return response
}

func loadSecurityClosureProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (SecurityClosureProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, securityClosureProofCommandType, idempotencyKey)
	if err != nil {
		return SecurityClosureProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return SecurityClosureProof{
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
		AuthorizationChanged:            metadataBool(response, "authorization_changed"),
		SecretPlaintextRead:             metadataBool(response, "secret_plaintext_read"),
		RemoteWorkerCredentialsIssued:   metadataBool(response, "remote_worker_credentials_issued"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func securityClosureProofFromEvent(event EventRecord) SecurityClosureProof {
	return SecurityClosureProof{
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
		AuthorizationChanged:            metadataBool(event.Metadata, "authorization_changed"),
		SecretPlaintextRead:             metadataBool(event.Metadata, "secret_plaintext_read"),
		RemoteWorkerCredentialsIssued:   metadataBool(event.Metadata, "remote_worker_credentials_issued"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        event.Metadata,
	}
}

func normalizeSecurityClosureProofFacts(facts []string) []string {
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

func securityClosureProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredSecurityClosureProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func securityClosureProofCompletesAudit(proof SecurityClosureProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(securityClosureProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.AuthorizationChanged &&
		!proof.SecretPlaintextRead &&
		!proof.RemoteWorkerCredentialsIssued &&
		!proof.CommandsRun &&
		!proof.AreaMatrixProtectedPathsTouched
}
