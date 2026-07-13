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

type RecordBackupRestoreProofOptions struct {
	ProofStatus                                 string
	Facts                                       []string
	Summary                                     string
	EvidenceURI                                 string
	BackupManifestHash                          string
	BackupManifestStatus                        string
	BackupManifestProjectCount                  *int64
	BackupManifestTableCount                    *int64
	RestorePlanStatus                           string
	RestorePlanScope                            string
	RestorePlanProjectKey                       string
	RestorePlanManifestHash                     string
	RestorePlanItemCount                        *int64
	ArtifactIntegrityStatus                     string
	ArtifactIntegrityCheckedCount               *int64
	ArtifactIntegrityFailedCount                *int64
	ArtifactArchivePreviewStatus                string
	ArtifactArchivePreviewTotalArtifacts        *int64
	ArtifactArchivePreviewExternalRefs          *int64
	ArtifactArchivePreviewNeedsPolicy           *int64
	ArtifactArchivePreviewProjectWriteAttempted *bool
	ArtifactArchivePreviewStorageWriteAttempted *bool
	ArtifactArchivePreviewDeleteAttempted       *bool
	IdempotencyKey                              string
	Actor                                       string
	Reason                                      string
	Metadata                                    map[string]any
}

type BackupRestoreProof struct {
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
	DatabaseRestoreAttempted        bool
	ArtifactBytesCopied             bool
	ArtifactBytesDeleted            bool
	ArtifactBytesUploaded           bool
	ArtifactGCAttempted             bool
	CommandsRun                     bool
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

type BackupRestoreCurrentBindingOptions struct {
	GeneratedAt           time.Time
	ArchivePreviewOptions ArtifactArchivePreviewOptions
}

type BackupRestoreCurrentBinding struct {
	Project                Record
	BackupManifest         BackupManifest
	RestorePlan            RestorePlan
	ArtifactIntegrity      ArtifactIntegrityReport
	ArtifactArchivePreview ArtifactArchivePreviewResult
	Metadata               map[string]any
}

const backupRestoreProofCommandType = "completion.backup_restore_proof.record"
const backupRestoreProofEventType = "completion.backup_restore_proof.recorded"

var backupRestoreProofBindingMetadataKeys = []string{
	"backup_manifest_hash",
	"backup_manifest_status",
	"backup_manifest_project_count",
	"backup_manifest_table_count",
	"restore_plan_status",
	"restore_plan_scope",
	"restore_plan_project_key",
	"restore_plan_manifest_hash",
	"restore_plan_item_count",
	"artifact_integrity_status",
	"artifact_integrity_checked_count",
	"artifact_integrity_failed_count",
	"artifact_archive_preview_status",
	"artifact_archive_preview_total_artifacts",
	"artifact_archive_preview_external_refs",
	"artifact_archive_preview_needs_policy",
	"artifact_archive_preview_project_write_attempted",
	"artifact_archive_preview_storage_write_attempted",
	"artifact_archive_preview_delete_attempted",
}

var backupRestoreProofCurrentBindingComparisonKeys = []string{
	"backup_manifest_status",
	"backup_manifest_project_count",
	"backup_manifest_table_count",
	"restore_plan_status",
	"restore_plan_scope",
	"restore_plan_project_key",
	"restore_plan_item_count",
	"artifact_integrity_status",
	"artifact_integrity_checked_count",
	"artifact_integrity_failed_count",
	"artifact_archive_preview_status",
	"artifact_archive_preview_total_artifacts",
	"artifact_archive_preview_external_refs",
	"artifact_archive_preview_needs_policy",
	"artifact_archive_preview_project_write_attempted",
	"artifact_archive_preview_storage_write_attempted",
	"artifact_archive_preview_delete_attempted",
}

var backupRestoreProofCurrentBindingHashFields = []string{
	"backup_manifest_hash",
	"restore_plan_manifest_hash",
}

var allowedBackupRestoreProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredBackupRestoreProofFacts = []string{
	"backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata",
	"restore_dry_run_identifies_metadata_only_history_and_object_verifier_limits",
	"artifact_integrity_distinguishes_local_project_reference_external_and_object",
	"archive_preview_does_not_copy_upload_delete_or_gc_artifact_bytes",
	"retention_classes_and_accepted_exceptions_are_documented",
	"no_restore_apply_or_artifact_mutation_opened",
}

func (s Store) RecordBackupRestoreProof(ctx context.Context, record Record, options RecordBackupRestoreProofOptions) (BackupRestoreProof, error) {
	options = normalizeRecordBackupRestoreProofOptions(options)
	if !allowedBackupRestoreProofStatuses[options.ProofStatus] {
		return BackupRestoreProof{}, fmt.Errorf("unsupported backup restore proof status %q", options.ProofStatus)
	}
	missingFacts := backupRestoreProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return BackupRestoreProof{}, fmt.Errorf("complete backup restore proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("backup restore", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return BackupRestoreProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := backupRestoreProofEvidenceBindingBlockers(record, options); len(blockers) > 0 {
			return BackupRestoreProof{}, fmt.Errorf("complete backup restore proof missing backup/restore/artifact output binding: %s", strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = backupRestoreProofIdempotencyKey(record, options)
	}
	requestHash, err := backupRestoreProofRequestHash(record, options)
	if err != nil {
		return BackupRestoreProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BackupRestoreProof{}, fmt.Errorf("begin backup restore proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, backupRestoreProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return BackupRestoreProof{}, err
	}
	if !created {
		result, err := loadBackupRestoreProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return BackupRestoreProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return BackupRestoreProof{}, fmt.Errorf("commit idempotent backup restore proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildBackupRestoreProof(record, options)
	eventID, err := insertBackupRestoreProofEvent(ctx, tx, result, options)
	if err != nil {
		return BackupRestoreProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertBackupRestoreProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return BackupRestoreProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, backupRestoreProofCommandType, options.IdempotencyKey, backupRestoreProofCommandResponse(result)); err != nil {
		return BackupRestoreProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BackupRestoreProof{}, fmt.Errorf("commit backup restore proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestBackupRestoreProof(ctx context.Context) (BackupRestoreProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		backupRestoreProofEventType,
	)
	if err != nil {
		return BackupRestoreProof{}, fmt.Errorf("load latest backup restore proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return BackupRestoreProof{}, err
	}
	if len(events) == 0 {
		return BackupRestoreProof{}, nil
	}
	return backupRestoreProofFromEvent(events[0]), nil
}

func (s Store) LatestBackupRestoreProofForProject(ctx context.Context, record Record) (BackupRestoreProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, backupRestoreProofEventType)
	if err != nil {
		return BackupRestoreProof{}, fmt.Errorf("load latest project backup restore proof: %w", err)
	}
	if !ok {
		return BackupRestoreProof{}, nil
	}
	return backupRestoreProofFromEvent(event), nil
}

func (s Store) BackupRestoreCurrentBinding(ctx context.Context, record Record, options BackupRestoreCurrentBindingOptions) (BackupRestoreCurrentBinding, error) {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	manifest, err := s.BackupManifest(ctx, BackupManifestOptions{GeneratedAt: options.GeneratedAt, ProjectID: record.ID, ProjectKey: record.Key})
	if err != nil {
		return BackupRestoreCurrentBinding{}, fmt.Errorf("build current backup manifest binding: %w", err)
	}
	restorePlan, err := s.RestorePlan(ctx, RestorePlanOptions{GeneratedAt: options.GeneratedAt, ProjectID: record.ID, ProjectKey: record.Key})
	if err != nil {
		return BackupRestoreCurrentBinding{}, fmt.Errorf("build current restore plan binding: %w", err)
	}
	integrity, err := s.ArtifactIntegrity(ctx, record, ArtifactIntegrityOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return BackupRestoreCurrentBinding{}, fmt.Errorf("build current artifact integrity binding: %w", err)
	}
	archivePreview, err := s.ArtifactArchivePreviewReadOnly(ctx, record, options.ArchivePreviewOptions)
	if err != nil {
		return BackupRestoreCurrentBinding{}, fmt.Errorf("build current artifact archive preview binding: %w", err)
	}
	return BackupRestoreCurrentBinding{
		Project:                record,
		BackupManifest:         manifest,
		RestorePlan:            restorePlan,
		ArtifactIntegrity:      integrity,
		ArtifactArchivePreview: archivePreview,
		Metadata:               backupRestoreCurrentBindingMetadata(manifest, restorePlan, integrity, archivePreview),
	}, nil
}

func normalizeRecordBackupRestoreProofOptions(options RecordBackupRestoreProofOptions) RecordBackupRestoreProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeBackupRestoreProofFacts(options.Facts)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.BackupManifestHash = strings.TrimSpace(options.BackupManifestHash)
	options.BackupManifestStatus = strings.TrimSpace(options.BackupManifestStatus)
	options.RestorePlanStatus = strings.TrimSpace(options.RestorePlanStatus)
	options.RestorePlanScope = strings.TrimSpace(options.RestorePlanScope)
	options.RestorePlanProjectKey = strings.TrimSpace(options.RestorePlanProjectKey)
	options.RestorePlanManifestHash = strings.TrimSpace(options.RestorePlanManifestHash)
	options.ArtifactIntegrityStatus = strings.TrimSpace(options.ArtifactIntegrityStatus)
	options.ArtifactArchivePreviewStatus = strings.TrimSpace(options.ArtifactArchivePreviewStatus)
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
		options.Reason = "record backup restore proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func backupRestoreProofRequestHash(record Record, options RecordBackupRestoreProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     backupRestoreProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeBackupRestoreProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"binding":          backupRestoreProofOptionsBindingPayload(options),
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal backup restore proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func backupRestoreProofIdempotencyKey(record Record, options RecordBackupRestoreProofOptions) string {
	hash, err := backupRestoreProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.backup_restore_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildBackupRestoreProof(record Record, options RecordBackupRestoreProofOptions) BackupRestoreProof {
	facts := normalizeBackupRestoreProofFacts(options.Facts)
	missingFacts := backupRestoreProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "backup restore proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "backup restore proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "backup restore proof is incomplete"
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
	addBackupRestoreProofBindingMetadata(metadata, options)
	if options.ProofStatus == "complete" && len(backupRestoreProofEvidenceBindingBlockers(record, options)) == 0 {
		metadata["backup_restore_evidence_binding_status"] = "pass"
	} else if options.ProofStatus == "complete" {
		metadata["backup_restore_evidence_binding_status"] = "fail"
	} else {
		metadata["backup_restore_evidence_binding_status"] = "not_required"
	}
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["database_restore_attempted"] = false
	metadata["artifact_bytes_copied"] = false
	metadata["artifact_bytes_deleted"] = false
	metadata["artifact_bytes_uploaded"] = false
	metadata["artifact_gc_attempted"] = false
	metadata["commands_run"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return BackupRestoreProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		DatabaseRestoreAttempted:        false,
		ArtifactBytesCopied:             false,
		ArtifactBytesDeleted:            false,
		ArtifactBytesUploaded:           false,
		ArtifactGCAttempted:             false,
		CommandsRun:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertBackupRestoreProofEvent(ctx context.Context, tx pgx.Tx, result BackupRestoreProof, options RecordBackupRestoreProofOptions) (int64, error) {
	metadata, err := json.Marshal(backupRestoreProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal backup restore proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Backup restore proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		backupRestoreProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert backup restore proof event: %w", err)
	}
	return eventID, nil
}

func insertBackupRestoreProofAuditEvent(ctx context.Context, tx pgx.Tx, result BackupRestoreProof, options RecordBackupRestoreProofOptions) (int64, error) {
	metadata, err := json.Marshal(backupRestoreProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal backup restore proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'backup_restore_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		backupRestoreProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert backup restore proof audit event: %w", err)
	}
	return auditEventID, nil
}

func backupRestoreProofEventMetadata(result BackupRestoreProof, options RecordBackupRestoreProofOptions) map[string]any {
	metadata := backupRestoreProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func backupRestoreProofCommandResponse(result BackupRestoreProof) map[string]any {
	return map[string]any{
		"project_key":                                      result.Project.Key,
		"status":                                           result.Status,
		"proof_status":                                     result.ProofStatus,
		"decision":                                         result.Decision,
		"message":                                          result.Message,
		"facts":                                            result.Facts,
		"missing_facts":                                    result.MissingFacts,
		"event_id":                                         result.EventID,
		"audit_event_id":                                   result.AuditEventID,
		"idempotency_key":                                  result.IdempotencyKey,
		"project_write_attempted":                          result.ProjectWriteAttempted,
		"execution_write_attempted":                        result.ExecutionWriteAttempted,
		"database_restore_attempted":                       result.DatabaseRestoreAttempted,
		"artifact_bytes_copied":                            result.ArtifactBytesCopied,
		"artifact_bytes_deleted":                           result.ArtifactBytesDeleted,
		"artifact_bytes_uploaded":                          result.ArtifactBytesUploaded,
		"artifact_gc_attempted":                            result.ArtifactGCAttempted,
		"commands_run":                                     result.CommandsRun,
		"area_matrix_protected_paths_touched":              result.AreaMatrixProtectedPathsTouched,
		"summary":                                          metadataString(result.Metadata, "summary"),
		"evidence_uri":                                     metadataString(result.Metadata, "evidence_uri"),
		"backup_restore_evidence_binding_status":           metadataString(result.Metadata, "backup_restore_evidence_binding_status"),
		"backup_manifest_hash":                             metadataString(result.Metadata, "backup_manifest_hash"),
		"backup_manifest_status":                           metadataString(result.Metadata, "backup_manifest_status"),
		"backup_manifest_project_count":                    metadataInt64(result.Metadata, "backup_manifest_project_count"),
		"backup_manifest_table_count":                      metadataInt64(result.Metadata, "backup_manifest_table_count"),
		"restore_plan_status":                              metadataString(result.Metadata, "restore_plan_status"),
		"restore_plan_scope":                               metadataString(result.Metadata, "restore_plan_scope"),
		"restore_plan_project_key":                         metadataString(result.Metadata, "restore_plan_project_key"),
		"restore_plan_manifest_hash":                       metadataString(result.Metadata, "restore_plan_manifest_hash"),
		"restore_plan_item_count":                          metadataInt64(result.Metadata, "restore_plan_item_count"),
		"artifact_integrity_status":                        metadataString(result.Metadata, "artifact_integrity_status"),
		"artifact_integrity_checked_count":                 metadataInt64(result.Metadata, "artifact_integrity_checked_count"),
		"artifact_integrity_failed_count":                  metadataInt64(result.Metadata, "artifact_integrity_failed_count"),
		"artifact_archive_preview_status":                  metadataString(result.Metadata, "artifact_archive_preview_status"),
		"artifact_archive_preview_total_artifacts":         metadataInt64(result.Metadata, "artifact_archive_preview_total_artifacts"),
		"artifact_archive_preview_external_refs":           metadataInt64(result.Metadata, "artifact_archive_preview_external_refs"),
		"artifact_archive_preview_needs_policy":            metadataInt64(result.Metadata, "artifact_archive_preview_needs_policy"),
		"artifact_archive_preview_project_write_attempted": metadataBool(result.Metadata, "artifact_archive_preview_project_write_attempted"),
		"artifact_archive_preview_storage_write_attempted": metadataBool(result.Metadata, "artifact_archive_preview_storage_write_attempted"),
		"artifact_archive_preview_delete_attempted":        metadataBool(result.Metadata, "artifact_archive_preview_delete_attempted"),
		"metadata":                                         result.Metadata,
	}
}

func loadBackupRestoreProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (BackupRestoreProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, backupRestoreProofCommandType, idempotencyKey)
	if err != nil {
		return BackupRestoreProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return BackupRestoreProof{
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
		DatabaseRestoreAttempted:        metadataBool(response, "database_restore_attempted"),
		ArtifactBytesCopied:             metadataBool(response, "artifact_bytes_copied"),
		ArtifactBytesDeleted:            metadataBool(response, "artifact_bytes_deleted"),
		ArtifactBytesUploaded:           metadataBool(response, "artifact_bytes_uploaded"),
		ArtifactGCAttempted:             metadataBool(response, "artifact_gc_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func backupRestoreProofFromEvent(event EventRecord) BackupRestoreProof {
	return BackupRestoreProof{
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
		DatabaseRestoreAttempted:        metadataBool(event.Metadata, "database_restore_attempted"),
		ArtifactBytesCopied:             metadataBool(event.Metadata, "artifact_bytes_copied"),
		ArtifactBytesDeleted:            metadataBool(event.Metadata, "artifact_bytes_deleted"),
		ArtifactBytesUploaded:           metadataBool(event.Metadata, "artifact_bytes_uploaded"),
		ArtifactGCAttempted:             metadataBool(event.Metadata, "artifact_gc_attempted"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        event.Metadata,
	}
}

func normalizeBackupRestoreProofFacts(facts []string) []string {
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

func backupRestoreProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredBackupRestoreProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func backupRestoreProofOptionsBindingPayload(options RecordBackupRestoreProofOptions) map[string]any {
	payload := map[string]any{
		"backup_manifest_hash":                             options.BackupManifestHash,
		"backup_manifest_status":                           options.BackupManifestStatus,
		"restore_plan_status":                              options.RestorePlanStatus,
		"restore_plan_scope":                               options.RestorePlanScope,
		"restore_plan_project_key":                         options.RestorePlanProjectKey,
		"restore_plan_manifest_hash":                       options.RestorePlanManifestHash,
		"artifact_integrity_status":                        options.ArtifactIntegrityStatus,
		"artifact_archive_preview_status":                  options.ArtifactArchivePreviewStatus,
		"backup_manifest_project_count":                    options.BackupManifestProjectCount,
		"backup_manifest_table_count":                      options.BackupManifestTableCount,
		"restore_plan_item_count":                          options.RestorePlanItemCount,
		"artifact_integrity_checked_count":                 options.ArtifactIntegrityCheckedCount,
		"artifact_integrity_failed_count":                  options.ArtifactIntegrityFailedCount,
		"artifact_archive_preview_total_artifacts":         options.ArtifactArchivePreviewTotalArtifacts,
		"artifact_archive_preview_external_refs":           options.ArtifactArchivePreviewExternalRefs,
		"artifact_archive_preview_needs_policy":            options.ArtifactArchivePreviewNeedsPolicy,
		"artifact_archive_preview_project_write_attempted": options.ArtifactArchivePreviewProjectWriteAttempted,
		"artifact_archive_preview_storage_write_attempted": options.ArtifactArchivePreviewStorageWriteAttempted,
		"artifact_archive_preview_delete_attempted":        options.ArtifactArchivePreviewDeleteAttempted,
	}
	return payload
}

func addBackupRestoreProofBindingMetadata(metadata map[string]any, options RecordBackupRestoreProofOptions) {
	metadata["backup_manifest_hash"] = options.BackupManifestHash
	metadata["backup_manifest_status"] = options.BackupManifestStatus
	if options.BackupManifestProjectCount != nil {
		metadata["backup_manifest_project_count"] = *options.BackupManifestProjectCount
	}
	if options.BackupManifestTableCount != nil {
		metadata["backup_manifest_table_count"] = *options.BackupManifestTableCount
	}
	metadata["restore_plan_status"] = options.RestorePlanStatus
	metadata["restore_plan_scope"] = options.RestorePlanScope
	metadata["restore_plan_project_key"] = options.RestorePlanProjectKey
	metadata["restore_plan_manifest_hash"] = options.RestorePlanManifestHash
	if options.RestorePlanItemCount != nil {
		metadata["restore_plan_item_count"] = *options.RestorePlanItemCount
	}
	metadata["artifact_integrity_status"] = options.ArtifactIntegrityStatus
	if options.ArtifactIntegrityCheckedCount != nil {
		metadata["artifact_integrity_checked_count"] = *options.ArtifactIntegrityCheckedCount
	}
	if options.ArtifactIntegrityFailedCount != nil {
		metadata["artifact_integrity_failed_count"] = *options.ArtifactIntegrityFailedCount
	}
	metadata["artifact_archive_preview_status"] = options.ArtifactArchivePreviewStatus
	if options.ArtifactArchivePreviewTotalArtifacts != nil {
		metadata["artifact_archive_preview_total_artifacts"] = *options.ArtifactArchivePreviewTotalArtifacts
	}
	if options.ArtifactArchivePreviewExternalRefs != nil {
		metadata["artifact_archive_preview_external_refs"] = *options.ArtifactArchivePreviewExternalRefs
	}
	if options.ArtifactArchivePreviewNeedsPolicy != nil {
		metadata["artifact_archive_preview_needs_policy"] = *options.ArtifactArchivePreviewNeedsPolicy
	}
	if options.ArtifactArchivePreviewProjectWriteAttempted != nil {
		metadata["artifact_archive_preview_project_write_attempted"] = *options.ArtifactArchivePreviewProjectWriteAttempted
	}
	if options.ArtifactArchivePreviewStorageWriteAttempted != nil {
		metadata["artifact_archive_preview_storage_write_attempted"] = *options.ArtifactArchivePreviewStorageWriteAttempted
	}
	if options.ArtifactArchivePreviewDeleteAttempted != nil {
		metadata["artifact_archive_preview_delete_attempted"] = *options.ArtifactArchivePreviewDeleteAttempted
	}
}

func backupRestoreCurrentBindingMetadata(manifest BackupManifest, restorePlan RestorePlan, integrity ArtifactIntegrityReport, archivePreview ArtifactArchivePreviewResult) map[string]any {
	return map[string]any{
		"backup_manifest_hash":                             manifest.ManifestHash,
		"backup_manifest_status":                           manifest.Status,
		"backup_manifest_project_count":                    int64(len(manifest.Projects)),
		"backup_manifest_table_count":                      int64(len(manifest.TableCounts)),
		"restore_plan_status":                              restorePlan.Status,
		"restore_plan_scope":                               restorePlan.Scope,
		"restore_plan_project_key":                         restorePlan.ProjectKey,
		"restore_plan_manifest_hash":                       restorePlan.ManifestHash,
		"restore_plan_item_count":                          int64(len(restorePlan.Items)),
		"artifact_integrity_status":                        integrity.Status,
		"artifact_integrity_checked_count":                 int64(integrity.CheckedArtifacts),
		"artifact_integrity_failed_count":                  int64(integrity.FailedArtifacts),
		"artifact_archive_preview_status":                  archivePreview.Status,
		"artifact_archive_preview_total_artifacts":         int64(archivePreview.Summary.TotalArtifacts),
		"artifact_archive_preview_external_refs":           int64(archivePreview.Summary.ExternalRefs),
		"artifact_archive_preview_needs_policy":            int64(archivePreview.Summary.NeedsPolicy),
		"artifact_archive_preview_project_write_attempted": archivePreview.ProjectWriteAttempted,
		"artifact_archive_preview_storage_write_attempted": archivePreview.StorageWriteAttempted,
		"artifact_archive_preview_delete_attempted":        archivePreview.ArtifactDeleteAttempted,
	}
}

func addBackupRestoreBindingMetadataWithPrefix(metadata map[string]any, prefix string, binding map[string]any) {
	for _, key := range backupRestoreProofBindingMetadataKeys {
		if value, ok := binding[key]; ok {
			metadata[prefix+key] = value
		}
	}
}

func backupRestoreProofEvidenceBindingBlockers(record Record, options RecordBackupRestoreProofOptions) []string {
	blockers := []string{}
	if !isSHA256Hex(options.BackupManifestHash) {
		blockers = append(blockers, "backup_manifest_hash_invalid")
	}
	if options.BackupManifestStatus != "ready" {
		blockers = append(blockers, "backup_manifest_status_not_ready")
	}
	if backupRestoreProofInt64PtrValue(options.BackupManifestProjectCount) <= 0 {
		blockers = append(blockers, "backup_manifest_project_count_missing")
	}
	if backupRestoreProofInt64PtrValue(options.BackupManifestTableCount) <= 0 {
		blockers = append(blockers, "backup_manifest_table_count_missing")
	}
	if !backupRestoreProofAllowedNeedsAttentionStatus(options.RestorePlanStatus) {
		blockers = append(blockers, "restore_plan_status_not_ready_or_needs_attention")
	}
	if options.RestorePlanScope != "project" {
		blockers = append(blockers, "restore_plan_scope_not_project")
	}
	if options.RestorePlanProjectKey != record.Key {
		blockers = append(blockers, "restore_plan_project_key_mismatch")
	}
	if !isSHA256Hex(options.RestorePlanManifestHash) {
		blockers = append(blockers, "restore_plan_manifest_hash_invalid")
	} else if options.BackupManifestHash != "" && options.RestorePlanManifestHash != options.BackupManifestHash {
		blockers = append(blockers, "restore_plan_manifest_hash_mismatch")
	}
	if backupRestoreProofInt64PtrValue(options.RestorePlanItemCount) <= 0 {
		blockers = append(blockers, "restore_plan_item_count_missing")
	}
	if !backupRestoreProofAllowedArtifactIntegrityStatus(options.ArtifactIntegrityStatus) {
		blockers = append(blockers, "artifact_integrity_status_not_pass_or_warn")
	}
	if backupRestoreProofInt64PtrValue(options.ArtifactIntegrityCheckedCount) <= 0 {
		blockers = append(blockers, "artifact_integrity_checked_count_missing")
	}
	if options.ArtifactIntegrityFailedCount == nil {
		blockers = append(blockers, "artifact_integrity_failed_count_missing")
	} else if *options.ArtifactIntegrityFailedCount != 0 {
		blockers = append(blockers, "artifact_integrity_failed_count_nonzero")
	}
	if !backupRestoreProofAllowedNeedsAttentionStatus(options.ArtifactArchivePreviewStatus) {
		blockers = append(blockers, "artifact_archive_preview_status_not_ready_or_needs_attention")
	}
	if backupRestoreProofInt64PtrValue(options.ArtifactArchivePreviewTotalArtifacts) <= 0 {
		blockers = append(blockers, "artifact_archive_preview_total_artifacts_missing")
	}
	if options.ArtifactArchivePreviewExternalRefs == nil {
		blockers = append(blockers, "artifact_archive_preview_external_refs_missing")
	} else if *options.ArtifactArchivePreviewExternalRefs < 0 {
		blockers = append(blockers, "artifact_archive_preview_external_refs_invalid")
	}
	if options.ArtifactArchivePreviewNeedsPolicy == nil {
		blockers = append(blockers, "artifact_archive_preview_needs_policy_missing")
	} else if *options.ArtifactArchivePreviewNeedsPolicy != 0 {
		blockers = append(blockers, "artifact_archive_preview_needs_policy_nonzero")
	}
	if backupRestoreProofBoolPtrValue(options.ArtifactArchivePreviewProjectWriteAttempted, true) {
		blockers = append(blockers, "artifact_archive_preview_project_write_attempted")
	}
	if backupRestoreProofBoolPtrValue(options.ArtifactArchivePreviewStorageWriteAttempted, true) {
		blockers = append(blockers, "artifact_archive_preview_storage_write_attempted")
	}
	if backupRestoreProofBoolPtrValue(options.ArtifactArchivePreviewDeleteAttempted, true) {
		blockers = append(blockers, "artifact_archive_preview_delete_attempted")
	}
	return uniqueStrings(blockers)
}

func backupRestoreProofCurrentBindingBlockers(proofMetadata map[string]any, currentMetadata map[string]any) []string {
	if currentMetadata == nil {
		return []string{"backup_restore_current_binding_missing"}
	}
	blockers := []string{}
	if currentBlockers := backupRestoreCurrentBindingMetadataBlockers(currentMetadata); len(currentBlockers) > 0 {
		blockers = append(blockers, currentBlockers...)
	}
	for _, key := range backupRestoreProofCurrentBindingComparisonKeys {
		if !backupRestoreBindingValuesEqual(proofMetadata, currentMetadata, key) {
			blockers = append(blockers, key+"_changed")
		}
	}
	if len(blockers) > 0 {
		blockers = append([]string{"backup_restore_proof_current_binding_mismatch"}, blockers...)
	}
	return uniqueStrings(blockers)
}

func backupRestoreCurrentBindingMetadataBlockers(metadata map[string]any) []string {
	current := copyMap(metadata)
	current["backup_restore_evidence_binding_status"] = "pass"
	return prefixBackupRestoreBindingBlockers("current_", backupRestoreProofMetadataBindingBlockers(current))
}

func prefixBackupRestoreBindingBlockers(prefix string, blockers []string) []string {
	out := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		out = append(out, prefix+blocker)
	}
	return uniqueStrings(out)
}

func backupRestoreBindingValuesEqual(left map[string]any, right map[string]any, key string) bool {
	switch key {
	case "backup_manifest_hash",
		"backup_manifest_status",
		"restore_plan_status",
		"restore_plan_scope",
		"restore_plan_project_key",
		"restore_plan_manifest_hash",
		"artifact_integrity_status",
		"artifact_archive_preview_status":
		return metadataString(left, key) == metadataString(right, key)
	case "artifact_archive_preview_project_write_attempted",
		"artifact_archive_preview_storage_write_attempted",
		"artifact_archive_preview_delete_attempted":
		return metadataBool(left, key) == metadataBool(right, key)
	default:
		return metadataInt64(left, key) == metadataInt64(right, key)
	}
}

func backupRestoreProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "backup_restore_evidence_binding_status") != "pass" {
		blockers = append(blockers, "backup_restore_evidence_binding_status_not_pass")
	}
	backupHash := metadataString(metadata, "backup_manifest_hash")
	if !isSHA256Hex(backupHash) {
		blockers = append(blockers, "backup_manifest_hash_invalid")
	}
	if metadataString(metadata, "backup_manifest_status") != "ready" {
		blockers = append(blockers, "backup_manifest_status_not_ready")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "backup_manifest_project_count"); !ok || value <= 0 {
		blockers = append(blockers, "backup_manifest_project_count_missing")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "backup_manifest_table_count"); !ok || value <= 0 {
		blockers = append(blockers, "backup_manifest_table_count_missing")
	}
	if !backupRestoreProofAllowedNeedsAttentionStatus(metadataString(metadata, "restore_plan_status")) {
		blockers = append(blockers, "restore_plan_status_not_ready_or_needs_attention")
	}
	if metadataString(metadata, "restore_plan_scope") != "project" {
		blockers = append(blockers, "restore_plan_scope_not_project")
	}
	if metadataString(metadata, "restore_plan_project_key") == "" {
		blockers = append(blockers, "restore_plan_project_key_missing")
	}
	restoreHash := metadataString(metadata, "restore_plan_manifest_hash")
	if !isSHA256Hex(restoreHash) {
		blockers = append(blockers, "restore_plan_manifest_hash_invalid")
	} else if backupHash != "" && restoreHash != backupHash {
		blockers = append(blockers, "restore_plan_manifest_hash_mismatch")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "restore_plan_item_count"); !ok || value <= 0 {
		blockers = append(blockers, "restore_plan_item_count_missing")
	}
	if !backupRestoreProofAllowedArtifactIntegrityStatus(metadataString(metadata, "artifact_integrity_status")) {
		blockers = append(blockers, "artifact_integrity_status_not_pass_or_warn")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "artifact_integrity_checked_count"); !ok || value <= 0 {
		blockers = append(blockers, "artifact_integrity_checked_count_missing")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "artifact_integrity_failed_count"); !ok {
		blockers = append(blockers, "artifact_integrity_failed_count_missing")
	} else if value != 0 {
		blockers = append(blockers, "artifact_integrity_failed_count_nonzero")
	}
	if !backupRestoreProofAllowedNeedsAttentionStatus(metadataString(metadata, "artifact_archive_preview_status")) {
		blockers = append(blockers, "artifact_archive_preview_status_not_ready_or_needs_attention")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "artifact_archive_preview_total_artifacts"); !ok || value <= 0 {
		blockers = append(blockers, "artifact_archive_preview_total_artifacts_missing")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "artifact_archive_preview_external_refs"); !ok {
		blockers = append(blockers, "artifact_archive_preview_external_refs_missing")
	} else if value < 0 {
		blockers = append(blockers, "artifact_archive_preview_external_refs_invalid")
	}
	if value, ok := backupRestoreProofMetadataInt64(metadata, "artifact_archive_preview_needs_policy"); !ok {
		blockers = append(blockers, "artifact_archive_preview_needs_policy_missing")
	} else if value != 0 {
		blockers = append(blockers, "artifact_archive_preview_needs_policy_nonzero")
	}
	if value, ok := backupRestoreProofMetadataBool(metadata, "artifact_archive_preview_project_write_attempted"); !ok || value {
		blockers = append(blockers, "artifact_archive_preview_project_write_attempted")
	}
	if value, ok := backupRestoreProofMetadataBool(metadata, "artifact_archive_preview_storage_write_attempted"); !ok || value {
		blockers = append(blockers, "artifact_archive_preview_storage_write_attempted")
	}
	if value, ok := backupRestoreProofMetadataBool(metadata, "artifact_archive_preview_delete_attempted"); !ok || value {
		blockers = append(blockers, "artifact_archive_preview_delete_attempted")
	}
	return uniqueStrings(blockers)
}

func backupRestoreProofAllowedNeedsAttentionStatus(status string) bool {
	return status == "ready" || status == "needs_attention"
}

func backupRestoreProofAllowedArtifactIntegrityStatus(status string) bool {
	return status == "pass" || status == "warn"
}

func backupRestoreProofInt64PtrValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func backupRestoreProofBoolPtrValue(value *bool, missingValue bool) bool {
	if value == nil {
		return missingValue
	}
	return *value
}

func backupRestoreProofMetadataInt64(metadata map[string]any, key string) (int64, bool) {
	if _, ok := metadata[key]; !ok {
		return 0, false
	}
	return metadataInt64(metadata, key), true
}

func backupRestoreProofMetadataBool(metadata map[string]any, key string) (bool, bool) {
	if _, ok := metadata[key]; !ok {
		return false, false
	}
	return metadataBool(metadata, key), true
}

func backupRestoreProofCompletesAudit(proof BackupRestoreProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(backupRestoreProofMetadataBindingBlockers(proof.Metadata)) == 0 &&
		len(proof.MissingFacts) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.DatabaseRestoreAttempted &&
		!proof.ArtifactBytesCopied &&
		!proof.ArtifactBytesDeleted &&
		!proof.ArtifactBytesUploaded &&
		!proof.ArtifactGCAttempted &&
		!proof.CommandsRun &&
		!proof.AreaMatrixProtectedPathsTouched
}
