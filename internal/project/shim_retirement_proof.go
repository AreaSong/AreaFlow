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

type RecordShimRetirementProofOptions struct {
	ProofStatus                 string
	Facts                       []string
	Summary                     string
	EvidenceURI                 string
	ShimRetirementScope         string
	ShimRetirementPrerequisites []string
	ShimRetiredSurfaces         []string
	ShimRollbackTarget          string
	ShimFailClosed              bool
	ShimReopenRequiresApproval  bool
	ReviewDecision              string
	ReviewedBy                  string
	ReviewedAt                  time.Time
	IdempotencyKey              string
	Actor                       string
	Reason                      string
	Metadata                    map[string]any
}

type ShimRetirementProof struct {
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
	LegacyRunnerStarted             bool
	LegacyProgressWritten           bool
	LegacyLogsWritten               bool
	LegacyCheckpointWritten         bool
	HistoricalFilesDeleted          bool
	ProgressJSONRewritten           bool
	AreaMatrixProtectedPathsTouched bool
	ShimRetirementScope             string
	ShimRetirementPrerequisites     []string
	ShimRetiredSurfaces             []string
	ShimRollbackTarget              string
	ShimFailClosed                  bool
	ShimReopenRequiresApproval      bool
	Metadata                        map[string]any
}

const shimRetirementProofCommandType = "completion.shim_retirement_proof.record"
const shimRetirementProofEventType = "completion.shim_retirement_proof.recorded"

var allowedShimRetirementProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredShimRetirementProofFacts = []string{
	"archive_gate_passed",
	"execution_forwarding_stable_for_declared_window",
	"no_legacy_task_loop_run_usage_in_active_workflow_versions",
	"areaflow_run_attempt_artifact_audit_coverage_pass",
	"compat_commands_mapped_or_deliberately_blocked",
	"legacy_progress_log_checkpoint_archive_reference_policy_accepted",
	"rollback_to_read_only_shim_documented",
	"user_facing_retirement_notice_present",
	"protected_path_proof_reference_recorded",
}

const shimRetirementProofScope = "read_only_shim_retirement_after_execution_forwarding_v1"
const shimRetirementProofRollbackTarget = "read_only_shim"
const shimRetirementProofBindingContract = "shim_retirement_scope_binding_v1"

var shimRetirementProofCurrentBindingComparisonKeys = []string{
	"shim_retirement_binding_contract",
	"shim_retirement_scope",
	"shim_retirement_prerequisites",
	"shim_retirement_prerequisites_hash",
	"shim_retired_surfaces",
	"shim_retired_surfaces_hash",
	"shim_rollback_target",
	"shim_fail_closed",
	"shim_reopen_requires_approval",
	"shim_retirement_scope_binding_hash",
}

var requiredShimRetirementProofPrerequisites = []string{
	"archive_gate_passed",
	"execution_cutover_gate_passed",
	"protected_path_proof_recorded",
}

var requiredShimRetiredSurfaces = []string{
	"legacy_task_loop_runner",
	"legacy_progress_json_writes",
	"legacy_logs_writes",
	"legacy_checkpoint_writes",
}

func (s Store) RecordShimRetirementProof(ctx context.Context, record Record, options RecordShimRetirementProofOptions) (ShimRetirementProof, error) {
	options = normalizeRecordShimRetirementProofOptions(options)
	if !allowedShimRetirementProofStatuses[options.ProofStatus] {
		return ShimRetirementProof{}, fmt.Errorf("unsupported shim retirement proof status %q", options.ProofStatus)
	}
	missingFacts := shimRetirementProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return ShimRetirementProof{}, fmt.Errorf("complete shim retirement proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("shim retirement", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return ShimRetirementProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := shimRetirementProofOptionsBindingBlockers(options); len(blockers) > 0 {
			return ShimRetirementProof{}, fmt.Errorf("complete shim retirement proof missing shim retirement scope binding: %s", strings.Join(blockers, ","))
		}
	}
	if err := requireCompleteProofReviewEvidence("shim retirement", "shim_retirement_proof", options.ProofStatus, options.EvidenceURI, proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata)); err != nil {
		return ShimRetirementProof{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = shimRetirementProofIdempotencyKey(record, options)
	}
	requestHash, err := shimRetirementProofRequestHash(record, options)
	if err != nil {
		return ShimRetirementProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ShimRetirementProof{}, fmt.Errorf("begin shim retirement proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, shimRetirementProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ShimRetirementProof{}, err
	}
	if !created {
		result, err := loadShimRetirementProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ShimRetirementProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ShimRetirementProof{}, fmt.Errorf("commit idempotent shim retirement proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildShimRetirementProof(record, options)
	eventID, err := insertShimRetirementProofEvent(ctx, tx, result, options)
	if err != nil {
		return ShimRetirementProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertShimRetirementProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ShimRetirementProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, shimRetirementProofCommandType, options.IdempotencyKey, shimRetirementProofCommandResponse(result)); err != nil {
		return ShimRetirementProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ShimRetirementProof{}, fmt.Errorf("commit shim retirement proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestShimRetirementProof(ctx context.Context) (ShimRetirementProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		shimRetirementProofEventType,
	)
	if err != nil {
		return ShimRetirementProof{}, fmt.Errorf("load latest shim retirement proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ShimRetirementProof{}, err
	}
	if len(events) == 0 {
		return ShimRetirementProof{}, nil
	}
	return shimRetirementProofFromEvent(events[0]), nil
}

func (s Store) LatestShimRetirementProofForProject(ctx context.Context, record Record) (ShimRetirementProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, shimRetirementProofEventType)
	if err != nil {
		return ShimRetirementProof{}, fmt.Errorf("load latest project shim retirement proof: %w", err)
	}
	if !ok {
		return ShimRetirementProof{}, nil
	}
	return shimRetirementProofFromEvent(event), nil
}

func normalizeRecordShimRetirementProofOptions(options RecordShimRetirementProofOptions) RecordShimRetirementProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeShimRetirementProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.ShimRetirementScope = strings.TrimSpace(options.ShimRetirementScope)
	options.ShimRetirementPrerequisites = normalizeStringList(options.ShimRetirementPrerequisites)
	options.ShimRetiredSurfaces = normalizeStringList(options.ShimRetiredSurfaces)
	options.ShimRollbackTarget = strings.TrimSpace(options.ShimRollbackTarget)
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
		options.Reason = "record AreaMatrix shim retirement proof"
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

func shimRetirementProofRequestHash(record Record, options RecordShimRetirementProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     shimRetirementProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeShimRetirementProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"binding":          shimRetirementProofOptionsBindingPayload(options),
		"review_metadata":  proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata),
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal shim retirement proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func shimRetirementProofIdempotencyKey(record Record, options RecordShimRetirementProofOptions) string {
	hash, err := shimRetirementProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.shim_retirement_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildShimRetirementProof(record Record, options RecordShimRetirementProofOptions) ShimRetirementProof {
	facts := normalizeShimRetirementProofFacts(options.Facts)
	missingFacts := shimRetirementProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "AreaMatrix shim retirement proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "AreaMatrix shim retirement proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "AreaMatrix shim retirement proof is incomplete"
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
	addProofReviewMetadata(metadata, options.ProofStatus, "shim_retirement_proof", proofReviewMetadataFromFields(options.ReviewDecision, options.ReviewedBy, options.ReviewedAt, options.Metadata))
	addShimRetirementProofBindingMetadata(metadata, options)
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["commands_run"] = false
	metadata["legacy_runner_started"] = false
	metadata["legacy_progress_written"] = false
	metadata["legacy_logs_written"] = false
	metadata["legacy_checkpoint_written"] = false
	metadata["historical_files_deleted"] = false
	metadata["progress_json_rewritten"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return ShimRetirementProof{
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
		LegacyRunnerStarted:             false,
		LegacyProgressWritten:           false,
		LegacyLogsWritten:               false,
		LegacyCheckpointWritten:         false,
		HistoricalFilesDeleted:          false,
		ProgressJSONRewritten:           false,
		AreaMatrixProtectedPathsTouched: false,
		ShimRetirementScope:             options.ShimRetirementScope,
		ShimRetirementPrerequisites:     append([]string{}, options.ShimRetirementPrerequisites...),
		ShimRetiredSurfaces:             append([]string{}, options.ShimRetiredSurfaces...),
		ShimRollbackTarget:              options.ShimRollbackTarget,
		ShimFailClosed:                  options.ShimFailClosed,
		ShimReopenRequiresApproval:      options.ShimReopenRequiresApproval,
		Metadata:                        metadata,
	}
}

func insertShimRetirementProofEvent(ctx context.Context, tx pgx.Tx, result ShimRetirementProof, options RecordShimRetirementProofOptions) (int64, error) {
	metadata, err := json.Marshal(shimRetirementProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal shim retirement proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Shim retirement proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		shimRetirementProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert shim retirement proof event: %w", err)
	}
	return eventID, nil
}

func insertShimRetirementProofAuditEvent(ctx context.Context, tx pgx.Tx, result ShimRetirementProof, options RecordShimRetirementProofOptions) (int64, error) {
	metadata, err := json.Marshal(shimRetirementProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal shim retirement proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'shim_retirement_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		shimRetirementProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert shim retirement proof audit event: %w", err)
	}
	return auditEventID, nil
}

func shimRetirementProofEventMetadata(result ShimRetirementProof, options RecordShimRetirementProofOptions) map[string]any {
	metadata := shimRetirementProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func shimRetirementProofCommandResponse(result ShimRetirementProof) map[string]any {
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
		"project_write_attempted":                result.ProjectWriteAttempted,
		"execution_write_attempted":              result.ExecutionWriteAttempted,
		"commands_run":                           result.CommandsRun,
		"legacy_runner_started":                  result.LegacyRunnerStarted,
		"legacy_progress_written":                result.LegacyProgressWritten,
		"legacy_logs_written":                    result.LegacyLogsWritten,
		"legacy_checkpoint_written":              result.LegacyCheckpointWritten,
		"historical_files_deleted":               result.HistoricalFilesDeleted,
		"progress_json_rewritten":                result.ProgressJSONRewritten,
		"area_matrix_protected_paths_touched":    result.AreaMatrixProtectedPathsTouched,
		"shim_retirement_scope_binding_status":   metadataString(result.Metadata, "shim_retirement_scope_binding_status"),
		"shim_retirement_scope_binding_blockers": metadataStringSlice(result.Metadata, "shim_retirement_scope_binding_blockers"),
		"shim_retirement_binding_contract":       metadataString(result.Metadata, "shim_retirement_binding_contract"),
		"shim_retirement_prerequisites_hash":     metadataString(result.Metadata, "shim_retirement_prerequisites_hash"),
		"shim_retired_surfaces_hash":             metadataString(result.Metadata, "shim_retired_surfaces_hash"),
		"shim_retirement_binding_hash":           metadataString(result.Metadata, "shim_retirement_binding_hash"),
		"shim_retirement_scope_binding_hash":     metadataString(result.Metadata, "shim_retirement_scope_binding_hash"),
		"shim_retirement_scope":                  result.ShimRetirementScope,
		"shim_retirement_prerequisites":          result.ShimRetirementPrerequisites,
		"shim_retired_surfaces":                  result.ShimRetiredSurfaces,
		"shim_rollback_target":                   result.ShimRollbackTarget,
		"shim_fail_closed":                       result.ShimFailClosed,
		"shim_reopen_requires_approval":          result.ShimReopenRequiresApproval,
		"summary":                                metadataString(result.Metadata, "summary"),
		"evidence_uri":                           metadataString(result.Metadata, "evidence_uri"),
		"review_decision":                        metadataString(result.Metadata, "review_decision"),
		"reviewed_by":                            metadataString(result.Metadata, "reviewed_by"),
		"reviewed_at":                            metadataString(result.Metadata, "reviewed_at"),
		"review_metadata_status":                 metadataString(result.Metadata, "review_metadata_status"),
		"review_metadata_blockers":               metadataStringSlice(result.Metadata, "review_metadata_blockers"),
		"metadata":                               result.Metadata,
	}
}

func loadShimRetirementProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ShimRetirementProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, shimRetirementProofCommandType, idempotencyKey)
	if err != nil {
		return ShimRetirementProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return ShimRetirementProof{
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
		LegacyRunnerStarted:             metadataBool(response, "legacy_runner_started"),
		LegacyProgressWritten:           metadataBool(response, "legacy_progress_written"),
		LegacyLogsWritten:               metadataBool(response, "legacy_logs_written"),
		LegacyCheckpointWritten:         metadataBool(response, "legacy_checkpoint_written"),
		HistoricalFilesDeleted:          metadataBool(response, "historical_files_deleted"),
		ProgressJSONRewritten:           metadataBool(response, "progress_json_rewritten"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		ShimRetirementScope:             metadataString(response, "shim_retirement_scope"),
		ShimRetirementPrerequisites:     metadataStringSlice(response, "shim_retirement_prerequisites"),
		ShimRetiredSurfaces:             metadataStringSlice(response, "shim_retired_surfaces"),
		ShimRollbackTarget:              metadataString(response, "shim_rollback_target"),
		ShimFailClosed:                  metadataBool(response, "shim_fail_closed"),
		ShimReopenRequiresApproval:      metadataBool(response, "shim_reopen_requires_approval"),
		Metadata:                        metadata,
	}, nil
}

func shimRetirementProofFromEvent(event EventRecord) ShimRetirementProof {
	metadata := proofMetadataFromEventMetadata(event.Metadata)
	return ShimRetirementProof{
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
		LegacyRunnerStarted:             metadataBool(event.Metadata, "legacy_runner_started"),
		LegacyProgressWritten:           metadataBool(event.Metadata, "legacy_progress_written"),
		LegacyLogsWritten:               metadataBool(event.Metadata, "legacy_logs_written"),
		LegacyCheckpointWritten:         metadataBool(event.Metadata, "legacy_checkpoint_written"),
		HistoricalFilesDeleted:          metadataBool(event.Metadata, "historical_files_deleted"),
		ProgressJSONRewritten:           metadataBool(event.Metadata, "progress_json_rewritten"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		ShimRetirementScope:             metadataString(event.Metadata, "shim_retirement_scope"),
		ShimRetirementPrerequisites:     metadataStringSlice(event.Metadata, "shim_retirement_prerequisites"),
		ShimRetiredSurfaces:             metadataStringSlice(event.Metadata, "shim_retired_surfaces"),
		ShimRollbackTarget:              metadataString(event.Metadata, "shim_rollback_target"),
		ShimFailClosed:                  metadataBool(event.Metadata, "shim_fail_closed"),
		ShimReopenRequiresApproval:      metadataBool(event.Metadata, "shim_reopen_requires_approval"),
		Metadata:                        metadata,
	}
}

func normalizeShimRetirementProofFacts(facts []string) []string {
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

func shimRetirementProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredShimRetirementProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func shimRetirementProofCompletesAudit(proof ShimRetirementProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proof.EventID > 0 &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		proofMetadataHasApprovedReviewEvidence("shim_retirement_proof", proof.Metadata) &&
		len(shimRetirementProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.CommandsRun &&
		!proof.LegacyRunnerStarted &&
		!proof.LegacyProgressWritten &&
		!proof.LegacyLogsWritten &&
		!proof.LegacyCheckpointWritten &&
		!proof.HistoricalFilesDeleted &&
		!proof.ProgressJSONRewritten &&
		!proof.AreaMatrixProtectedPathsTouched
}

func shimRetirementProofOptionsBindingPayload(options RecordShimRetirementProofOptions) map[string]any {
	return shimRetirementProofBindingMetadata(
		options.ShimRetirementScope,
		options.ShimRetirementPrerequisites,
		options.ShimRetiredSurfaces,
		options.ShimRollbackTarget,
		options.ShimFailClosed,
		options.ShimReopenRequiresApproval,
	)
}

func addShimRetirementProofBindingMetadata(metadata map[string]any, options RecordShimRetirementProofOptions) {
	binding := shimRetirementProofOptionsBindingPayload(options)
	for key, value := range binding {
		metadata[key] = value
	}
	blockers := shimRetirementProofOptionsBindingBlockers(options)
	metadata["shim_retirement_scope_binding_blockers"] = blockers
	if options.ProofStatus == "complete" && len(blockers) == 0 {
		metadata["shim_retirement_scope_binding_status"] = "pass"
	} else if len(blockers) > 0 {
		metadata["shim_retirement_scope_binding_status"] = "fail"
	} else {
		metadata["shim_retirement_scope_binding_status"] = "not_required"
	}
}

func shimRetirementProofOptionsBindingBlockers(options RecordShimRetirementProofOptions) []string {
	blockers := []string{}
	if options.ShimRetirementScope != shimRetirementProofScope {
		blockers = append(blockers, "shim_retirement_scope_missing_or_mismatch")
	}
	if !sameNormalizedStrings(options.ShimRetirementPrerequisites, requiredShimRetirementProofPrerequisites) {
		blockers = append(blockers, "shim_retirement_prerequisites_missing_or_mismatch")
	}
	if !sameNormalizedStrings(options.ShimRetiredSurfaces, requiredShimRetiredSurfaces) {
		blockers = append(blockers, "shim_retired_surfaces_missing_or_mismatch")
	}
	if options.ShimRollbackTarget != shimRetirementProofRollbackTarget {
		blockers = append(blockers, "shim_rollback_target_missing_or_mismatch")
	}
	if !options.ShimFailClosed {
		blockers = append(blockers, "shim_fail_closed_missing")
	}
	if !options.ShimReopenRequiresApproval {
		blockers = append(blockers, "shim_reopen_requires_approval_missing")
	}
	return uniqueStrings(blockers)
}

func shimRetirementProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "shim_retirement_scope_binding_status") != "pass" {
		blockers = append(blockers, "shim_retirement_scope_binding_status_not_pass")
	}
	if metadataString(metadata, "shim_retirement_binding_contract") != shimRetirementProofBindingContract {
		blockers = append(blockers, "shim_retirement_binding_contract_missing_or_mismatch")
	}
	if metadataString(metadata, "shim_retirement_scope") != shimRetirementProofScope {
		blockers = append(blockers, "shim_retirement_scope_missing_or_mismatch")
	}
	prerequisites := metadataStringSlice(metadata, "shim_retirement_prerequisites")
	if !sameNormalizedStrings(prerequisites, requiredShimRetirementProofPrerequisites) {
		blockers = append(blockers, "shim_retirement_prerequisites_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "shim_retirement_prerequisites_hash")) ||
		metadataString(metadata, "shim_retirement_prerequisites_hash") != shimRetirementProofStringSetHash("shim_retirement_prerequisites", prerequisites) {
		blockers = append(blockers, "shim_retirement_prerequisites_hash_missing_or_mismatch")
	}
	retiredSurfaces := metadataStringSlice(metadata, "shim_retired_surfaces")
	if !sameNormalizedStrings(retiredSurfaces, requiredShimRetiredSurfaces) {
		blockers = append(blockers, "shim_retired_surfaces_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "shim_retired_surfaces_hash")) ||
		metadataString(metadata, "shim_retired_surfaces_hash") != shimRetirementProofStringSetHash("shim_retired_surfaces", retiredSurfaces) {
		blockers = append(blockers, "shim_retired_surfaces_hash_missing_or_mismatch")
	}
	if metadataString(metadata, "shim_rollback_target") != shimRetirementProofRollbackTarget {
		blockers = append(blockers, "shim_rollback_target_missing_or_mismatch")
	}
	if !metadataBool(metadata, "shim_fail_closed") {
		blockers = append(blockers, "shim_fail_closed_missing")
	}
	if !metadataBool(metadata, "shim_reopen_requires_approval") {
		blockers = append(blockers, "shim_reopen_requires_approval_missing")
	}
	if !looksLikeSHA256(metadataString(metadata, "shim_retirement_scope_binding_hash")) ||
		metadataString(metadata, "shim_retirement_scope_binding_hash") != shimRetirementProofBindingHash(metadata) {
		blockers = append(blockers, "shim_retirement_scope_binding_hash_missing_or_mismatch")
	}
	return uniqueStrings(blockers)
}

func shimRetirementProofCurrentBinding() map[string]any {
	binding := shimRetirementProofBindingMetadata(
		shimRetirementProofScope,
		requiredShimRetirementProofPrerequisites,
		requiredShimRetiredSurfaces,
		shimRetirementProofRollbackTarget,
		true,
		true,
	)
	binding["shim_retirement_scope_binding_status"] = "pass"
	binding["shim_retirement_scope_binding_blockers"] = []string{}
	return binding
}

func shimRetirementProofBindingMetadata(scope string, prerequisites []string, retiredSurfaces []string, rollbackTarget string, failClosed bool, reopenRequiresApproval bool) map[string]any {
	prerequisites = normalizeStringList(prerequisites)
	retiredSurfaces = normalizeStringList(retiredSurfaces)
	metadata := map[string]any{
		"shim_retirement_binding_contract":   shimRetirementProofBindingContract,
		"shim_retirement_scope":              scope,
		"shim_retirement_prerequisites":      prerequisites,
		"shim_retirement_prerequisites_hash": shimRetirementProofStringSetHash("shim_retirement_prerequisites", prerequisites),
		"shim_retired_surfaces":              retiredSurfaces,
		"shim_retired_surfaces_hash":         shimRetirementProofStringSetHash("shim_retired_surfaces", retiredSurfaces),
		"shim_rollback_target":               rollbackTarget,
		"shim_fail_closed":                   failClosed,
		"shim_reopen_requires_approval":      reopenRequiresApproval,
	}
	metadata["shim_retirement_binding_hash"] = shimRetirementProofBindingHash(metadata)
	metadata["shim_retirement_scope_binding_hash"] = metadata["shim_retirement_binding_hash"]
	return metadata
}

func shimRetirementProofCurrentBindingBlockers(proofMetadata map[string]any, currentBinding map[string]any) []string {
	blockers := shimRetirementProofMetadataBindingBlockers(proofMetadata)
	if len(blockers) > 0 {
		return blockers
	}
	currentBlockers := shimRetirementProofMetadataBindingBlockers(currentBinding)
	if len(currentBlockers) > 0 {
		for _, blocker := range currentBlockers {
			blockers = append(blockers, "current_"+blocker)
		}
		return uniqueStrings(append([]string{"shim_retirement_scope_current_binding_mismatch"}, blockers...))
	}
	for _, key := range shimRetirementProofCurrentBindingComparisonKeys {
		if !shimRetirementProofBindingValuesEqual(proofMetadata, currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	if len(blockers) > 0 {
		blockers = append([]string{"shim_retirement_scope_current_binding_mismatch"}, blockers...)
	}
	return uniqueStrings(blockers)
}

func shimRetirementProofBindingValuesEqual(left map[string]any, right map[string]any, key string) bool {
	switch key {
	case "shim_fail_closed", "shim_reopen_requires_approval":
		return metadataBool(left, key) == metadataBool(right, key)
	case "shim_retirement_prerequisites", "shim_retired_surfaces":
		return sameNormalizedStrings(metadataStringSlice(left, key), metadataStringSlice(right, key))
	default:
		return metadataString(left, key) == metadataString(right, key)
	}
}

func shimRetirementProofBindingHash(metadata map[string]any) string {
	payload := map[string]any{
		"shim_retirement_binding_contract":   metadataString(metadata, "shim_retirement_binding_contract"),
		"shim_retirement_scope":              metadataString(metadata, "shim_retirement_scope"),
		"shim_retirement_prerequisites_hash": metadataString(metadata, "shim_retirement_prerequisites_hash"),
		"shim_retired_surfaces_hash":         metadataString(metadata, "shim_retired_surfaces_hash"),
		"shim_rollback_target":               metadataString(metadata, "shim_rollback_target"),
		"shim_fail_closed":                   metadataBool(metadata, "shim_fail_closed"),
		"shim_reopen_requires_approval":      metadataBool(metadata, "shim_reopen_requires_approval"),
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func shimRetirementProofStringSetHash(kind string, values []string) string {
	payload, err := json.Marshal(map[string]any{
		"shim_retirement_binding_contract": shimRetirementProofBindingContract,
		"kind":                             kind,
		"values":                           normalizeStringList(values),
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
