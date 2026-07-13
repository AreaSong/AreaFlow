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

type RecordReleasePackagingProofOptions struct {
	ProofStatus    string
	Facts          []string
	Summary        string
	EvidenceURI    string
	IdempotencyKey string
	Actor          string
	Reason         string
	Metadata       map[string]any
}

type ReleasePackagingProof struct {
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
	ReleasePackageCreated           bool
	ReleaseStateWritten             bool
	ReleaseApprovalCreated          bool
	RolloutStateCreated             bool
	MigrationApplyAttempted         bool
	TagCreated                      bool
	PackageSigned                   bool
	ArtifactUploaded                bool
	GitPushAttempted                bool
	PublishAttempted                bool
	CommandsRun                     bool
	AreaMatrixProtectedPathsTouched bool
	Metadata                        map[string]any
}

const releasePackagingProofCommandType = "completion.release_packaging_proof.record"
const releasePackagingProofEventType = "completion.release_packaging_proof.recorded"

var allowedReleasePackagingProofStatuses = map[string]bool{
	"complete":   true,
	"incomplete": true,
	"blocked":    true,
}

var requiredReleasePackagingProofFacts = []string{
	"release_final_gate_passed",
	"release_evidence_bundle_metadata_only",
	"release_package_preview_created_no_package",
	"distribution_preview_no_upload_sign_tag_push",
	"publish_gate_and_approval_preview_created_no_publish_or_approval",
	"rollout_plan_preview_created_no_rollout_state",
	"no_release_package_publish_rollout_apply_opened",
}

func (s Store) RecordReleasePackagingProof(ctx context.Context, record Record, options RecordReleasePackagingProofOptions) (ReleasePackagingProof, error) {
	options = normalizeRecordReleasePackagingProofOptions(options)
	if !allowedReleasePackagingProofStatuses[options.ProofStatus] {
		return ReleasePackagingProof{}, fmt.Errorf("unsupported release packaging proof status %q", options.ProofStatus)
	}
	missingFacts := releasePackagingProofMissingFacts(options.Facts)
	if options.ProofStatus == "complete" && len(missingFacts) > 0 {
		return ReleasePackagingProof{}, fmt.Errorf("complete release packaging proof missing required facts: %s", strings.Join(missingFacts, ","))
	}
	if err := requireProofEvidenceForStatus("release packaging", options.ProofStatus, options.Summary, options.EvidenceURI, "complete"); err != nil {
		return ReleasePackagingProof{}, err
	}
	if options.ProofStatus == "complete" {
		if blockers := releasePackagingProofRequiredBundleMetadataBlockersForProject(options.Metadata, record); len(blockers) > 0 {
			return ReleasePackagingProof{}, fmt.Errorf("complete release packaging proof missing release evidence bundle binding: %s", strings.Join(blockers, ","))
		}
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = releasePackagingProofIdempotencyKey(record, options)
	}
	requestHash, err := releasePackagingProofRequestHash(record, options)
	if err != nil {
		return ReleasePackagingProof{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReleasePackagingProof{}, fmt.Errorf("begin release packaging proof record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, releasePackagingProofCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ReleasePackagingProof{}, err
	}
	if !created {
		result, err := loadReleasePackagingProofByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ReleasePackagingProof{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReleasePackagingProof{}, fmt.Errorf("commit idempotent release packaging proof record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildReleasePackagingProof(record, options)
	eventID, err := insertReleasePackagingProofEvent(ctx, tx, result, options)
	if err != nil {
		return ReleasePackagingProof{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertReleasePackagingProofAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ReleasePackagingProof{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, releasePackagingProofCommandType, options.IdempotencyKey, releasePackagingProofCommandResponse(result)); err != nil {
		return ReleasePackagingProof{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReleasePackagingProof{}, fmt.Errorf("commit release packaging proof record: %w", err)
	}
	return result, nil
}

func (s Store) LatestReleasePackagingProof(ctx context.Context) (ReleasePackagingProof, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE event_type = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		releasePackagingProofEventType,
	)
	if err != nil {
		return ReleasePackagingProof{}, fmt.Errorf("load latest release packaging proof: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return ReleasePackagingProof{}, err
	}
	if len(events) == 0 {
		return ReleasePackagingProof{}, nil
	}
	return releasePackagingProofFromEvent(events[0]), nil
}

func (s Store) LatestReleasePackagingProofForProject(ctx context.Context, record Record) (ReleasePackagingProof, error) {
	event, ok, err := s.LatestEventByType(ctx, record.ID, releasePackagingProofEventType)
	if err != nil {
		return ReleasePackagingProof{}, fmt.Errorf("load latest project release packaging proof: %w", err)
	}
	if !ok {
		return ReleasePackagingProof{}, nil
	}
	return releasePackagingProofFromEvent(event), nil
}

func normalizeRecordReleasePackagingProofOptions(options RecordReleasePackagingProofOptions) RecordReleasePackagingProofOptions {
	options.ProofStatus = strings.TrimSpace(options.ProofStatus)
	options.Facts = normalizeReleasePackagingProofFacts(options.Facts)
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
		options.Reason = "record release packaging proof"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func releasePackagingProofRequestHash(record Record, options RecordReleasePackagingProofOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":     releasePackagingProofCommandType,
		"project_id":       record.ID,
		"project_key":      record.Key,
		"proof_status":     options.ProofStatus,
		"facts":            normalizeReleasePackagingProofFacts(options.Facts),
		"summary":          options.Summary,
		"evidence_uri":     options.EvidenceURI,
		"actor":            options.Actor,
		"reason":           options.Reason,
		"metadata":         options.Metadata,
		"protected":        true,
		"no_project_write": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal release packaging proof request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func releasePackagingProofIdempotencyKey(record Record, options RecordReleasePackagingProofOptions) string {
	hash, err := releasePackagingProofRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("completion.release_packaging_proof.record:%s:%s:%s", record.Key, options.ProofStatus, prefix)
}

func buildReleasePackagingProof(record Record, options RecordReleasePackagingProofOptions) ReleasePackagingProof {
	facts := normalizeReleasePackagingProofFacts(options.Facts)
	missingFacts := releasePackagingProofMissingFacts(facts)
	status := "recorded"
	decision := "allowed"
	message := "release packaging proof recorded"
	if options.ProofStatus == "blocked" {
		status = "blocked"
		decision = "blocked"
		message = "release packaging proof is blocked"
	} else if options.ProofStatus == "incomplete" {
		decision = "needs_attention"
		message = "release packaging proof is incomplete"
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
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["release_package_created"] = false
	metadata["release_state_written"] = false
	metadata["release_approval_created"] = false
	metadata["rollout_state_created"] = false
	metadata["migration_apply_attempted"] = false
	metadata["tag_created"] = false
	metadata["package_signed"] = false
	metadata["artifact_uploaded"] = false
	metadata["git_push_attempted"] = false
	metadata["publish_attempted"] = false
	metadata["commands_run"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	return ReleasePackagingProof{
		Project:                         record,
		Status:                          status,
		ProofStatus:                     options.ProofStatus,
		Decision:                        decision,
		Message:                         message,
		Facts:                           facts,
		MissingFacts:                    missingFacts,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		ReleasePackageCreated:           false,
		ReleaseStateWritten:             false,
		ReleaseApprovalCreated:          false,
		RolloutStateCreated:             false,
		MigrationApplyAttempted:         false,
		TagCreated:                      false,
		PackageSigned:                   false,
		ArtifactUploaded:                false,
		GitPushAttempted:                false,
		PublishAttempted:                false,
		CommandsRun:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        metadata,
	}
}

func insertReleasePackagingProofEvent(ctx context.Context, tx pgx.Tx, result ReleasePackagingProof, options RecordReleasePackagingProofOptions) (int64, error) {
	metadata, err := json.Marshal(releasePackagingProofEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal release packaging proof event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Release packaging proof recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		releasePackagingProofEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert release packaging proof event: %w", err)
	}
	return eventID, nil
}

func insertReleasePackagingProofAuditEvent(ctx context.Context, tx pgx.Tx, result ReleasePackagingProof, options RecordReleasePackagingProofOptions) (int64, error) {
	metadata, err := json.Marshal(releasePackagingProofCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal release packaging proof audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'release_packaging_proof', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		releasePackagingProofCommandType,
		result.ProofStatus,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert release packaging proof audit event: %w", err)
	}
	return auditEventID, nil
}

func releasePackagingProofEventMetadata(result ReleasePackagingProof, options RecordReleasePackagingProofOptions) map[string]any {
	metadata := releasePackagingProofCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func releasePackagingProofCommandResponse(result ReleasePackagingProof) map[string]any {
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
		"release_package_created":             result.ReleasePackageCreated,
		"release_state_written":               result.ReleaseStateWritten,
		"release_approval_created":            result.ReleaseApprovalCreated,
		"rollout_state_created":               result.RolloutStateCreated,
		"migration_apply_attempted":           result.MigrationApplyAttempted,
		"tag_created":                         result.TagCreated,
		"package_signed":                      result.PackageSigned,
		"artifact_uploaded":                   result.ArtifactUploaded,
		"git_push_attempted":                  result.GitPushAttempted,
		"publish_attempted":                   result.PublishAttempted,
		"commands_run":                        result.CommandsRun,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"summary":                             metadataString(result.Metadata, "summary"),
		"evidence_uri":                        metadataString(result.Metadata, "evidence_uri"),
		"metadata":                            result.Metadata,
	}
}

func loadReleasePackagingProofByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ReleasePackagingProof, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, releasePackagingProofCommandType, idempotencyKey)
	if err != nil {
		return ReleasePackagingProof{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return ReleasePackagingProof{
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
		ReleasePackageCreated:           metadataBool(response, "release_package_created"),
		ReleaseStateWritten:             metadataBool(response, "release_state_written"),
		ReleaseApprovalCreated:          metadataBool(response, "release_approval_created"),
		RolloutStateCreated:             metadataBool(response, "rollout_state_created"),
		MigrationApplyAttempted:         metadataBool(response, "migration_apply_attempted"),
		TagCreated:                      metadataBool(response, "tag_created"),
		PackageSigned:                   metadataBool(response, "package_signed"),
		ArtifactUploaded:                metadataBool(response, "artifact_uploaded"),
		GitPushAttempted:                metadataBool(response, "git_push_attempted"),
		PublishAttempted:                metadataBool(response, "publish_attempted"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}, nil
}

func releasePackagingProofFromEvent(event EventRecord) ReleasePackagingProof {
	metadata := releasePackagingProofMergedMetadata(event.Metadata)
	return ReleasePackagingProof{
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
		ReleasePackageCreated:           metadataBool(event.Metadata, "release_package_created"),
		ReleaseStateWritten:             metadataBool(event.Metadata, "release_state_written"),
		ReleaseApprovalCreated:          metadataBool(event.Metadata, "release_approval_created"),
		RolloutStateCreated:             metadataBool(event.Metadata, "rollout_state_created"),
		MigrationApplyAttempted:         metadataBool(event.Metadata, "migration_apply_attempted"),
		TagCreated:                      metadataBool(event.Metadata, "tag_created"),
		PackageSigned:                   metadataBool(event.Metadata, "package_signed"),
		ArtifactUploaded:                metadataBool(event.Metadata, "artifact_uploaded"),
		GitPushAttempted:                metadataBool(event.Metadata, "git_push_attempted"),
		PublishAttempted:                metadataBool(event.Metadata, "publish_attempted"),
		CommandsRun:                     metadataBool(event.Metadata, "commands_run"),
		AreaMatrixProtectedPathsTouched: metadataBool(event.Metadata, "area_matrix_protected_paths_touched"),
		Metadata:                        metadata,
	}
}

func releasePackagingProofMergedMetadata(envelope map[string]any) map[string]any {
	metadata := map[string]any{}
	if raw, ok := envelope["metadata"].(map[string]any); ok {
		for key, value := range raw {
			metadata[key] = value
		}
	}
	for key, value := range envelope {
		if key == "metadata" {
			continue
		}
		metadata[key] = value
	}
	return metadata
}

func normalizeReleasePackagingProofFacts(facts []string) []string {
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

func releasePackagingProofMissingFacts(facts []string) []string {
	present := map[string]bool{}
	for _, fact := range facts {
		present[fact] = true
	}
	missing := []string{}
	for _, required := range requiredReleasePackagingProofFacts {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func releasePackagingProofRequiredBundleMetadataBlockers(metadata map[string]any) []string {
	return releasePackagingProofRequiredBundleMetadataBlockersForProject(metadata, Record{})
}

func releasePackagingProofRequiredBundleMetadataBlockersForProject(metadata map[string]any, record Record) []string {
	blockers := []string{}
	if metadataString(metadata, "release_evidence_bundle_hash") == "" {
		blockers = append(blockers, "release_evidence_bundle_hash_missing")
	}
	if metadataString(metadata, "release_evidence_bundle_status") != "ready" {
		blockers = append(blockers, "release_evidence_bundle_status_not_ready")
	}
	if metadataString(metadata, "release_evidence_bundle_mode") != "read_only_release_evidence_bundle" {
		blockers = append(blockers, "release_evidence_bundle_mode_invalid")
	}
	if metadataString(metadata, "release_evidence_bundle_scope") != "project" {
		blockers = append(blockers, "release_evidence_bundle_scope_not_project")
	}
	projectKey := metadataString(metadata, "release_evidence_bundle_project_key")
	if projectKey == "" {
		blockers = append(blockers, "release_evidence_bundle_project_key_missing")
	} else if strings.TrimSpace(record.Key) != "" && projectKey != record.Key {
		blockers = append(blockers, "release_evidence_bundle_project_key_mismatch")
	}
	if metadataInt64(metadata, "release_evidence_bundle_item_count") <= 0 {
		blockers = append(blockers, "release_evidence_bundle_item_count_missing")
	}
	if metadataString(metadata, "release_evidence_bundle_project_inventory_key") == "" {
		blockers = append(blockers, "release_evidence_bundle_project_inventory_key_missing")
	}
	if !metadataBool(metadata, "release_evidence_bundle_project_inventory_present") {
		blockers = append(blockers, "release_evidence_bundle_project_inventory_missing")
	}
	if !metadataBool(metadata, "release_evidence_bundle_project_inventory_ready") {
		blockers = append(blockers, "release_evidence_bundle_project_inventory_not_ready")
	}
	if !metadataBool(metadata, "release_evidence_bundle_ready") {
		blockers = append(blockers, "release_evidence_bundle_ready_false")
	}
	return blockers
}

func releasePackagingProofBundleBindingBlockers(proof ReleasePackagingProof, bundle ReleaseEvidenceBundle) []string {
	blockers := releasePackagingProofCurrentBundleBlockers(bundle)
	proofHash := metadataString(proof.Metadata, "release_evidence_bundle_hash")
	if proofHash == "" {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_hash_missing")
	} else if proofHash != bundle.BundleHash {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_hash_mismatch")
	}
	if metadataString(proof.Metadata, "release_evidence_bundle_status") != "ready" {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_status_not_ready")
	}
	if metadataString(proof.Metadata, "release_evidence_bundle_mode") != "read_only_release_evidence_bundle" {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_mode_invalid")
	}
	if metadataString(proof.Metadata, "release_evidence_bundle_scope") != "project" {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_scope_not_project")
	}
	if metadataString(proof.Metadata, "release_evidence_bundle_project_key") != bundle.ProjectKey {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_project_key_mismatch")
	}
	if !metadataBool(proof.Metadata, "release_evidence_bundle_project_inventory_present") {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_project_inventory_missing")
	}
	if !metadataBool(proof.Metadata, "release_evidence_bundle_project_inventory_ready") {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_project_inventory_not_ready")
	}
	if !metadataBool(proof.Metadata, "release_evidence_bundle_ready") {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_ready_false")
	}
	proofItemCount := metadataInt64(proof.Metadata, "release_evidence_bundle_item_count")
	if proofItemCount <= 0 {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_item_count_missing")
	} else if int(proofItemCount) != len(bundle.Items) {
		blockers = append(blockers, "release_packaging_proof_release_evidence_bundle_item_count_mismatch")
	}
	return uniqueStrings(blockers)
}

func releasePackagingProofCurrentBundleBlockers(bundle ReleaseEvidenceBundle) []string {
	blockers := []string{}
	if bundle.BundleHash == "" {
		blockers = append(blockers, "release_evidence_bundle_hash_missing")
	}
	if bundle.Status != "ready" {
		blockers = append(blockers, "release_evidence_bundle_not_ready")
	}
	if bundle.Mode != "read_only_release_evidence_bundle" {
		blockers = append(blockers, "release_evidence_bundle_mode_invalid")
	}
	if bundle.Scope != "project" {
		blockers = append(blockers, "release_evidence_bundle_scope_not_project")
	}
	if strings.TrimSpace(bundle.ProjectKey) == "" {
		blockers = append(blockers, "release_evidence_bundle_project_key_missing")
	}
	if len(bundle.Items) == 0 {
		blockers = append(blockers, "release_evidence_bundle_items_missing")
	}
	projectInventoryKey := "evidence:project_inventory:" + bundle.ProjectKey
	if projectInventoryKey != "evidence:project_inventory:" && !releaseEvidenceBundleHasReadyItem(bundle, projectInventoryKey) {
		blockers = append(blockers, "release_evidence_bundle_project_inventory_not_ready")
	}
	for _, item := range bundle.Items {
		if item.Status != "ready" {
			key := item.Key
			if key == "" {
				key = "unknown"
			}
			blockers = append(blockers, "release_evidence_bundle_item_not_ready:"+key)
		}
	}
	return blockers
}

func releasePackagingProofCompletesAudit(proof ReleasePackagingProof) bool {
	return proof.Status == "recorded" &&
		proof.ProofStatus == "complete" &&
		proof.Decision == "allowed" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(proof.MissingFacts) == 0 &&
		len(releasePackagingProofRequiredBundleMetadataBlockersForProject(proof.Metadata, proof.Project)) == 0 &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.ReleasePackageCreated &&
		!proof.ReleaseStateWritten &&
		!proof.ReleaseApprovalCreated &&
		!proof.RolloutStateCreated &&
		!proof.MigrationApplyAttempted &&
		!proof.TagCreated &&
		!proof.PackageSigned &&
		!proof.ArtifactUploaded &&
		!proof.GitPushAttempted &&
		!proof.PublishAttempted &&
		!proof.CommandsRun &&
		!proof.AreaMatrixProtectedPathsTouched
}
