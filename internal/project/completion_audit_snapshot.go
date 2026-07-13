package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

type RecordCompletionAuditSnapshotOptions struct {
	ReleaseCandidateLabel string
	EvidenceClass         string
	EvidenceURI           string
	Summary               string
	ReviewDecision        string
	ReviewedBy            string
	ReviewedAt            time.Time
	IdempotencyKey        string
	Actor                 string
	Reason                string
	Metadata              map[string]any
	EvidenceRoot          string
	GeneratedAt           time.Time
}

type CompletionAuditSnapshot struct {
	Real100Guardrail
	Project                         Record
	Status                          string
	Decision                        string
	Message                         string
	AuditStatus                     string
	AuditScope                      string
	AuditHash                       string
	ReleaseCandidateLabel           string
	EvidenceClass                   string
	EvidenceURI                     string
	ProofEventIDs                   map[string]int64
	EventID                         int64
	AuditEventID                    int64
	IdempotencyKey                  string
	Created                         bool
	CreatedAt                       time.Time
	ProjectWriteAttempted           bool
	ExecutionWriteAttempted         bool
	ReleasePackageCreated           bool
	PublishAttempted                bool
	RestoreApplyAttempted           bool
	SecretResolved                  bool
	RemoteWorkerCredentialsIssued   bool
	AreaMatrixProtectedPathsTouched bool
	CommandsRun                     bool
	SmokeRunAttempted               bool
	WorkerStarted                   bool
	Metadata                        map[string]any
}

type CompletionAuditSnapshotReadiness struct {
	Real100Guardrail
	Project       Record
	Status        string
	Message       string
	HasSnapshot   bool
	Latest        CompletionAuditSnapshot
	Items         []ReadinessItem
	SafetyFacts   map[string]bool
	RequiredClass string
	BundleHash    string
}

type CompletionAuditSnapshotGap struct {
	Key                              string         `json:"key"`
	Category                         string         `json:"category"`
	Status                           string         `json:"status"`
	Message                          string         `json:"message"`
	Blockers                         []string       `json:"blockers,omitempty"`
	MissingProofEvidenceURIKeys      []string       `json:"missing_proof_evidence_uri_keys,omitempty"`
	MissingProofEventIDKeys          []string       `json:"missing_proof_event_id_keys,omitempty"`
	MissingProofProvenanceKeys       []string       `json:"missing_proof_provenance_keys,omitempty"`
	IdentityBlockers                 []string       `json:"identity_blockers,omitempty"`
	MechanismProofEvidenceURIs       []string       `json:"mechanism_proof_evidence_uris,omitempty"`
	ProofEvidenceURIBlockers         []string       `json:"proof_evidence_uri_blockers,omitempty"`
	ProofEventIDBlockers             []string       `json:"proof_event_id_blockers,omitempty"`
	ProofProvenanceBlockers          []string       `json:"proof_provenance_blockers,omitempty"`
	CurrentProofBindingBlockers      []string       `json:"current_proof_binding_blockers,omitempty"`
	BundleBlockers                   []string       `json:"bundle_blockers,omitempty"`
	EvidenceURIFileAuditBlockers     []string       `json:"evidence_uri_file_audit_blockers,omitempty"`
	PackageAStatusProjectionBlockers []string       `json:"package_a_status_projection_blockers,omitempty"`
	ReviewMetadataBlockers           []string       `json:"review_metadata_blockers,omitempty"`
	UnsafeFacts                      []string       `json:"unsafe_facts,omitempty"`
	Metadata                         map[string]any `json:"metadata,omitempty"`
}

type CompletionAuditSnapshotClosure struct {
	Status                           string                             `json:"status"`
	Ready                            bool                               `json:"ready"`
	ReadyForReleaseCandidateClosure  bool                               `json:"ready_for_release_candidate_closure"`
	RequiredClass                    string                             `json:"required_class"`
	RequiredEvidenceClass            string                             `json:"required_evidence_class"`
	HasSnapshot                      bool                               `json:"has_snapshot"`
	GapCount                         int                                `json:"gap_count"`
	GapKeys                          []string                           `json:"gap_keys"`
	Blockers                         []string                           `json:"blockers"`
	ProjectIdentity                  CompletionAuditSnapshotClosureGate `json:"project_identity"`
	Snapshot                         CompletionAuditSnapshotClosureGate `json:"snapshot"`
	AuditBinding                     CompletionAuditSnapshotClosureGate `json:"audit_binding"`
	SnapshotEvidence                 CompletionAuditSnapshotClosureGate `json:"snapshot_evidence"`
	ProofEvidenceURIs                CompletionAuditSnapshotClosureGate `json:"proof_evidence_uris"`
	ProofEventIDs                    CompletionAuditSnapshotClosureGate `json:"proof_event_ids"`
	ProofProvenance                  CompletionAuditSnapshotClosureGate `json:"proof_provenance"`
	CurrentProofBinding              CompletionAuditSnapshotClosureGate `json:"current_proof_binding"`
	ReleaseEvidenceBundle            CompletionAuditSnapshotClosureGate `json:"release_evidence_bundle"`
	EvidenceFileAudit                CompletionAuditSnapshotClosureGate `json:"evidence_file_audit"`
	PackageAStatusProjection         CompletionAuditSnapshotClosureGate `json:"package_a_status_projection"`
	ReviewMetadata                   CompletionAuditSnapshotClosureGate `json:"review_metadata"`
	Safety                           CompletionAuditSnapshotClosureGate `json:"safety"`
	IdentityStatus                   string                             `json:"identity_status"`
	SnapshotStatus                   string                             `json:"snapshot_status"`
	AuditIdentityStatus              string                             `json:"audit_identity_status"`
	AuditHashStatus                  string                             `json:"audit_hash_status"`
	SnapshotEvidenceStatus           string                             `json:"snapshot_evidence_status"`
	SafetyStatus                     string                             `json:"safety_status"`
	ProofEvidenceURIStatus           string                             `json:"proof_evidence_uri_status"`
	ProofEventIDStatus               string                             `json:"proof_event_id_status"`
	ProofProvenanceStatus            string                             `json:"proof_provenance_status"`
	CurrentProofBindingStatus        string                             `json:"current_proof_binding_status"`
	EvidenceURIFileAuditStatus       string                             `json:"evidence_uri_file_audit_status"`
	ReleaseEvidenceBundleStatus      string                             `json:"release_evidence_bundle_status"`
	PackageAStatusProjectionStatus   string                             `json:"package_a_status_projection_status"`
	ReviewMetadataStatus             string                             `json:"review_metadata_status"`
	MissingProofEvidenceURIKeys      []string                           `json:"missing_proof_evidence_uri_keys"`
	MissingProofEventIDKeys          []string                           `json:"missing_proof_event_id_keys"`
	MissingProofProvenanceKeys       []string                           `json:"missing_proof_provenance_keys"`
	MechanismProofEvidenceURIs       []string                           `json:"mechanism_proof_evidence_uris"`
	ProofEvidenceURIBlockers         []string                           `json:"proof_evidence_uri_blockers"`
	ProofEventIDBlockers             []string                           `json:"proof_event_id_blockers"`
	ProofProvenanceBlockers          []string                           `json:"proof_provenance_blockers"`
	CurrentProofBindingBlockers      []string                           `json:"current_proof_binding_blockers"`
	BundleBlockers                   []string                           `json:"bundle_blockers"`
	EvidenceURIFileAuditBlockers     []string                           `json:"evidence_uri_file_audit_blockers"`
	PackageAStatusProjectionBlockers []string                           `json:"package_a_status_projection_blockers"`
	ReviewMetadataBlockers           []string                           `json:"review_metadata_blockers"`
	UnsafeFacts                      []string                           `json:"unsafe_facts"`
	LatestEvidenceClass              string                             `json:"latest_evidence_class,omitempty"`
	LatestReleaseCandidate           string                             `json:"latest_release_candidate,omitempty"`
	LatestEvidenceURI                string                             `json:"latest_evidence_uri,omitempty"`
	LatestReviewDecision             string                             `json:"latest_review_decision,omitempty"`
	LatestReviewedBy                 string                             `json:"latest_reviewed_by,omitempty"`
	LatestReviewedAt                 string                             `json:"latest_reviewed_at,omitempty"`
	LatestEventID                    int64                              `json:"latest_event_id,omitempty"`
	SnapshotAuditHash                string                             `json:"snapshot_audit_hash,omitempty"`
	CurrentAuditHash                 string                             `json:"current_audit_hash,omitempty"`
	CurrentAuditStatus               string                             `json:"current_audit_status,omitempty"`
	CurrentAuditScope                string                             `json:"current_audit_scope,omitempty"`
	BundleHash                       string                             `json:"bundle_hash,omitempty"`
	LatestBundleHash                 string                             `json:"latest_bundle_hash,omitempty"`
	CurrentBundleHash                string                             `json:"current_bundle_hash,omitempty"`
	CurrentBundleStatus              string                             `json:"current_bundle_status,omitempty"`
	CurrentBundleMode                string                             `json:"current_bundle_mode,omitempty"`
}

type CompletionAuditSnapshotClosureGate struct {
	Status   string         `json:"status"`
	Ready    bool           `json:"ready"`
	Blockers []string       `json:"blockers,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type completionAuditSnapshotCurrentAuditBinding struct {
	Status                   string
	Scope                    string
	Hash                     string
	Real100Guardrail         Real100Guardrail
	ProofEvidenceURIs        []string
	ProofEvidenceURIMap      map[string]string
	ProofEventIDs            map[string]int64
	ProofProvenanceMap       map[string]string
	EvidenceRoot             string
	PackageAStatusProjection completionAuditSnapshotPackageAStatusProjectionBinding
}

type completionAuditSnapshotPackageAStatusProjectionBinding struct {
	LatestImportSourceHash      string
	LatestImportSourceHashError string
	ProjectionQueryError        string
	HasProjection               bool
	LatestProjection            StatusProjectionRecord
	HasWrittenProjection        bool
	LatestWrittenProjection     StatusProjectionRecord
	CurrentPreimageCaptured     bool
	CurrentPreimage             StatusProjectionPreimage
}

const completionAuditSnapshotCommandType = "completion.audit_snapshot.record"
const completionAuditSnapshotEventType = "completion.audit_snapshot.recorded"
const completionAuditSnapshotEvidenceClassFixture = "fixture"
const completionAuditSnapshotEvidenceClassReleaseCandidate = "release_candidate"
const completionAuditTargetProjectRoot = "/Users/as/Ai-Project/project/AreaMatrix"

func (s Store) CompletionAuditSnapshotReadiness(ctx context.Context, record Record) (CompletionAuditSnapshotReadiness, error) {
	if !completionAuditSnapshotProjectMatches(record) {
		return buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, ReleaseEvidenceBundle{}), nil
	}
	event, ok, err := s.LatestEventByType(ctx, record.ID, completionAuditSnapshotEventType)
	if err != nil {
		return CompletionAuditSnapshotReadiness{}, err
	}
	if len(completionAuditSnapshotRealProjectIdentityBlockers(record)) > 0 {
		if !ok {
			return buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, ReleaseEvidenceBundle{}), nil
		}
		return buildCompletionAuditSnapshotReadiness(record, completionAuditSnapshotFromEvent(record, event), true, ReleaseEvidenceBundle{}), nil
	}
	bundle, err := s.ReleaseEvidenceBundle(ctx, ReleaseEvidenceBundleOptions{ProjectID: record.ID, ProjectKey: record.Key})
	if err != nil {
		return CompletionAuditSnapshotReadiness{}, err
	}
	currentAudit, err := s.CompletionAudit(ctx, CompletionAuditOptions{})
	if err != nil {
		return CompletionAuditSnapshotReadiness{}, err
	}
	currentAuditHash, err := completionAuditHash(currentAudit)
	if err != nil {
		return CompletionAuditSnapshotReadiness{}, err
	}
	currentAuditBinding := completionAuditSnapshotCurrentAuditBinding{
		Status:                   currentAudit.Status,
		Scope:                    currentAudit.Scope,
		Hash:                     currentAuditHash,
		Real100Guardrail:         currentAudit.Real100Guardrail,
		ProofEvidenceURIs:        completionAuditSnapshotProofEvidenceURIs(currentAudit),
		ProofEvidenceURIMap:      completionAuditSnapshotProofEvidenceURIMap(currentAudit),
		ProofEventIDs:            completionAuditProofEventIDs(currentAudit),
		ProofProvenanceMap:       completionAuditSnapshotProofProvenanceMap(currentAudit),
		EvidenceRoot:             completionAuditSnapshotDefaultEvidenceRoot(),
		PackageAStatusProjection: s.completionAuditSnapshotPackageAStatusProjectionBinding(ctx, record),
	}
	if !ok {
		return buildCompletionAuditSnapshotReadiness(record, CompletionAuditSnapshot{}, false, bundle, currentAuditBinding), nil
	}
	return buildCompletionAuditSnapshotReadiness(record, completionAuditSnapshotFromEvent(record, event), true, bundle, currentAuditBinding), nil
}

func (s Store) completionAuditSnapshotPackageAStatusProjectionBinding(ctx context.Context, record Record) completionAuditSnapshotPackageAStatusProjectionBinding {
	binding := completionAuditSnapshotPackageAStatusProjectionBinding{}
	if latestImport, err := s.LatestImportSnapshot(ctx, record.ID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			binding.LatestImportSourceHashError = err.Error()
		} else {
			binding.LatestImportSourceHashError = "latest import snapshot missing"
		}
	} else {
		binding.LatestImportSourceHash = strings.TrimSpace(latestImport.SourceHash)
	}

	if projections, err := s.ListStatusProjections(ctx, record, 50); err != nil {
		binding.ProjectionQueryError = err.Error()
	} else {
		for _, projection := range projections {
			if strings.TrimSpace(projection.TargetURI) != ".areaflow/status.json" ||
				strings.TrimSpace(projection.TargetKind) != "project_status_json" {
				continue
			}
			if !binding.HasProjection {
				binding.HasProjection = true
				binding.LatestProjection = projection
			}
			if strings.TrimSpace(projection.WriteState) == "written" {
				binding.HasWrittenProjection = true
				binding.LatestWrittenProjection = projection
				break
			}
		}
	}

	targetPath, targetPathErr := statusProjectionTargetPath(record, ".areaflow/status.json")
	binding.CurrentPreimage = inspectStatusProjectionPreimage(targetPath, targetPathErr)
	binding.CurrentPreimageCaptured = true
	return binding
}

func (s Store) RecordCompletionAuditSnapshot(ctx context.Context, record Record, options RecordCompletionAuditSnapshotOptions) (CompletionAuditSnapshot, error) {
	options = normalizeRecordCompletionAuditSnapshotOptions(options)
	audit, err := s.CompletionAudit(ctx, CompletionAuditOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	bundle := ReleaseEvidenceBundle{}
	if options.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate {
		bundle, err = s.ReleaseEvidenceBundle(ctx, ReleaseEvidenceBundleOptions{GeneratedAt: options.GeneratedAt, ProjectID: record.ID, ProjectKey: record.Key})
		if err != nil {
			return CompletionAuditSnapshot{}, err
		}
	}
	result, err := buildCompletionAuditSnapshot(record, audit, options, bundle)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = completionAuditSnapshotIdempotencyKey(record, result, options)
	}
	requestHash, err := completionAuditSnapshotRequestHash(record, result, options)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CompletionAuditSnapshot{}, fmt.Errorf("begin completion audit snapshot record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, completionAuditSnapshotCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	if !created {
		result, err := loadCompletionAuditSnapshotByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return CompletionAuditSnapshot{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return CompletionAuditSnapshot{}, fmt.Errorf("commit idempotent completion audit snapshot record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	eventID, err := insertCompletionAuditSnapshotEvent(ctx, tx, result, options)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertCompletionAuditSnapshotAuditEvent(ctx, tx, result, options)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, completionAuditSnapshotCommandType, options.IdempotencyKey, completionAuditSnapshotCommandResponse(result)); err != nil {
		return CompletionAuditSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CompletionAuditSnapshot{}, fmt.Errorf("commit completion audit snapshot record: %w", err)
	}
	return result, nil
}

func normalizeRecordCompletionAuditSnapshotOptions(options RecordCompletionAuditSnapshotOptions) RecordCompletionAuditSnapshotOptions {
	options.ReleaseCandidateLabel = strings.TrimSpace(options.ReleaseCandidateLabel)
	options.EvidenceClass = strings.TrimSpace(options.EvidenceClass)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.Summary = strings.TrimSpace(options.Summary)
	options.ReviewDecision = strings.ToLower(strings.TrimSpace(options.ReviewDecision))
	options.ReviewedBy = strings.TrimSpace(options.ReviewedBy)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.EvidenceRoot = strings.TrimSpace(options.EvidenceRoot)
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.ReviewDecision == "" {
		options.ReviewDecision = strings.ToLower(strings.TrimSpace(metadataString(options.Metadata, "review_decision")))
	}
	if options.ReviewedBy == "" {
		options.ReviewedBy = strings.TrimSpace(metadataString(options.Metadata, "reviewed_by"))
	}
	if options.ReviewedAt.IsZero() {
		options.ReviewedAt = metadataTime(options.Metadata, "reviewed_at")
	}
	if options.ReleaseCandidateLabel == "" {
		options.ReleaseCandidateLabel = "v1.0-candidate"
	}
	if options.EvidenceClass == "" {
		options.EvidenceClass = completionAuditSnapshotEvidenceClassFixture
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "record completion audit snapshot"
	}
	if options.EvidenceRoot == "" {
		options.EvidenceRoot = completionAuditSnapshotDefaultEvidenceRoot()
	}
	return options
}

func buildCompletionAuditSnapshot(record Record, audit CompletionAudit, options RecordCompletionAuditSnapshotOptions, bundle ReleaseEvidenceBundle) (CompletionAuditSnapshot, error) {
	if !completionAuditSnapshotProjectMatches(record) {
		return CompletionAuditSnapshot{}, fmt.Errorf("completion audit snapshot can only be recorded for %s, got %q", completionAuditTargetProjectKey, record.Key)
	}
	if audit.Status != "complete" {
		return CompletionAuditSnapshot{}, fmt.Errorf("completion audit snapshot requires current audit status complete, got %q", audit.Status)
	}
	if !completionAuditSnapshotEvidenceClassAllowed(options.EvidenceClass) {
		return CompletionAuditSnapshot{}, fmt.Errorf("completion audit snapshot evidence class must be fixture or release_candidate, got %q", options.EvidenceClass)
	}
	if err := validateCompletionAuditSnapshotEvidenceClass(options); err != nil {
		return CompletionAuditSnapshot{}, err
	}
	auditHash, err := completionAuditHash(audit)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	proofEventIDs := completionAuditProofEventIDs(audit)
	proofEvidenceURIs := completionAuditSnapshotProofEvidenceURIs(audit)
	proofEvidenceURIMap := completionAuditSnapshotProofEvidenceURIMap(audit)
	proofProvenanceMap := completionAuditSnapshotProofProvenanceMap(audit)
	evidenceURIFileAudit := []map[string]any{}
	if options.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate {
		if blockers := completionAuditSnapshotRealProjectIdentityBlockers(record); len(blockers) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires real AreaMatrix project identity: %s", strings.Join(blockers, ","))
		}
		if blockers := completionAuditSnapshotReleaseEvidenceBundleBlockers(bundle); len(blockers) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires ready release evidence bundle: %s", strings.Join(blockers, ","))
		}
		mechanismURIs := completionAuditSnapshotMechanismEvidenceURIs(audit)
		if len(mechanismURIs) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot cannot seal local script/smoke proof evidence: %s", strings.Join(mechanismURIs, ","))
		}
		missingEvidenceURIs := completionAuditSnapshotMissingProofEvidenceURIs(audit)
		if len(missingEvidenceURIs) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires complete proof evidence URIs: %s", strings.Join(missingEvidenceURIs, ","))
		}
		missingEventIDs := completionAuditSnapshotMissingProofEventIDs(audit)
		if len(missingEventIDs) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires complete proof event IDs: %s", strings.Join(missingEventIDs, ","))
		}
		if blockers := completionAuditSnapshotProofEvidenceURIMapBindingBlockers(proofEvidenceURIs, proofEvidenceURIMap); len(blockers) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires complete proof evidence URIs: %s", strings.Join(blockers, ","))
		}
		if blockers := completionAuditSnapshotProofEventIDMapBindingBlockers(proofEventIDs); len(blockers) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires complete proof event IDs: %s", strings.Join(blockers, ","))
		}
		if blockers := completionAuditSnapshotProofProvenanceBindingBlockers(proofProvenanceMap); len(blockers) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires release-candidate proof provenance: %s", strings.Join(blockers, ","))
		}
		var blockers []string
		evidenceURIFileAudit, blockers = completionAuditSnapshotAuditEvidenceURIRefs(options.EvidenceRoot, options.EvidenceURI, proofEvidenceURIs)
		if len(blockers) > 0 {
			return CompletionAuditSnapshot{}, fmt.Errorf("release_candidate completion audit snapshot requires local evidence URI file audit: %s", strings.Join(blockers, ","))
		}
	}
	metadata := map[string]any{}
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	metadata["project_key"] = record.Key
	metadata["audit_status"] = audit.Status
	metadata["audit_scope"] = audit.Scope
	metadata["audit_mode"] = audit.Mode
	metadata["audit_hash"] = auditHash
	metadata["release_candidate_label"] = options.ReleaseCandidateLabel
	metadata["evidence_class"] = options.EvidenceClass
	metadata["fixture_snapshot"] = options.EvidenceClass == completionAuditSnapshotEvidenceClassFixture
	metadata["release_candidate_snapshot"] = options.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate
	metadata["evidence_uri"] = options.EvidenceURI
	metadata["summary"] = options.Summary
	metadata["proof_event_ids"] = proofEventIDs
	metadata["proof_event_id_count"] = len(proofEventIDs)
	metadata["required_proof_event_id_keys"] = completionAuditSnapshotRequiredProofEventIDKeys()
	metadata["proof_evidence_uris"] = proofEvidenceURIs
	metadata["proof_evidence_uri_map"] = proofEvidenceURIMap
	metadata["proof_evidence_uri_count"] = len(proofEvidenceURIs)
	metadata["required_proof_evidence_uri_keys"] = completionAuditSnapshotRequiredProofEvidenceURIKeys()
	metadata["proof_provenance_map"] = proofProvenanceMap
	metadata["required_proof_provenance_keys"] = completionAuditSnapshotRequiredProofProvenanceKeys()
	if options.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate {
		reviewMetadata := completionAuditSnapshotReviewMetadataFromOptions(options)
		metadata["review_decision"] = reviewMetadata["review_decision"]
		metadata["reviewed_by"] = reviewMetadata["reviewed_by"]
		metadata["reviewed_at"] = reviewMetadata["reviewed_at"]
		metadata["review_metadata_status"] = "approved"
		metadata["release_evidence_bundle_hash"] = bundle.BundleHash
		metadata["release_evidence_bundle_status"] = bundle.Status
		metadata["release_evidence_bundle_mode"] = bundle.Mode
		metadata["release_evidence_bundle_item_count"] = len(bundle.Items)
		metadata["release_evidence_bundle_ready"] = true
		metadata["evidence_uri_file_audit"] = evidenceURIFileAudit
		metadata["evidence_uri_file_audit_count"] = len(evidenceURIFileAudit)
		metadata["evidence_uri_file_audit_status"] = "pass"
	}
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["release_package_created"] = false
	metadata["publish_attempted"] = false
	metadata["restore_apply_attempted"] = false
	metadata["secret_resolved"] = false
	metadata["remote_worker_credentials_issued"] = false
	metadata["area_matrix_protected_paths_touched"] = false
	metadata["commands_run"] = false
	metadata["smoke_run_attempted"] = false
	metadata["worker_started"] = false
	return CompletionAuditSnapshot{
		Real100Guardrail:                CompletionAuditReal100Guardrail(),
		Project:                         record,
		Status:                          "recorded",
		Decision:                        "allowed",
		Message:                         "completion audit snapshot recorded",
		AuditStatus:                     audit.Status,
		AuditScope:                      audit.Scope,
		AuditHash:                       auditHash,
		ReleaseCandidateLabel:           options.ReleaseCandidateLabel,
		EvidenceClass:                   options.EvidenceClass,
		EvidenceURI:                     options.EvidenceURI,
		ProofEventIDs:                   proofEventIDs,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		ReleasePackageCreated:           false,
		PublishAttempted:                false,
		RestoreApplyAttempted:           false,
		SecretResolved:                  false,
		RemoteWorkerCredentialsIssued:   false,
		AreaMatrixProtectedPathsTouched: false,
		CommandsRun:                     false,
		SmokeRunAttempted:               false,
		WorkerStarted:                   false,
		Metadata:                        metadata,
	}, nil
}

func completionAuditSnapshotEvidenceClassAllowed(value string) bool {
	return value == completionAuditSnapshotEvidenceClassFixture || value == completionAuditSnapshotEvidenceClassReleaseCandidate
}

func completionAuditSnapshotProjectMatches(record Record) bool {
	return record.Key == completionAuditTargetProjectKey
}

func completionAuditSnapshotRealProjectIdentityBlockers(record Record) []string {
	blockers := []string{}
	if !completionAuditSnapshotProjectMatches(record) {
		blockers = append(blockers, "project_key_mismatch")
	}
	if filepath.Clean(strings.TrimSpace(record.RootPath)) != completionAuditTargetProjectRoot {
		blockers = append(blockers, "project_root_not_real_areamatrix")
	}
	if strings.TrimSpace(record.Adapter) != "areamatrix" {
		blockers = append(blockers, "adapter_not_areamatrix")
	}
	if strings.TrimSpace(record.WorkflowProfile) != "areamatrix" {
		blockers = append(blockers, "workflow_profile_not_areamatrix")
	}
	if strings.TrimSpace(record.DefaultBranch) != "main" {
		blockers = append(blockers, "default_branch_not_main")
	}
	kind := strings.ToLower(strings.TrimSpace(record.Kind))
	if kind != "product-repo" {
		blockers = append(blockers, "project_kind_not_product_repo")
	}
	if strings.Contains(kind, "fixture") || strings.Contains(kind, "temp") || strings.Contains(kind, "temporary") {
		blockers = append(blockers, "project_kind_marks_fixture_or_temp")
	}
	root := strings.ToLower(filepath.Clean(strings.TrimSpace(record.RootPath)))
	if strings.Contains(root, "fixture") || strings.Contains(root, "tmp") || strings.Contains(root, "temp") {
		blockers = append(blockers, "project_root_marks_fixture_or_temp")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotReleaseEvidenceBundleBlockers(bundle ReleaseEvidenceBundle) []string {
	blockers := []string{}
	if strings.TrimSpace(bundle.BundleHash) == "" {
		blockers = append(blockers, "release_evidence_bundle_hash_missing")
	}
	if bundle.Status != "ready" {
		blockers = append(blockers, "release_evidence_bundle_not_ready")
	}
	if bundle.Mode != "read_only_release_evidence_bundle" {
		blockers = append(blockers, "release_evidence_bundle_mode_invalid")
	}
	if len(bundle.Items) == 0 {
		blockers = append(blockers, "release_evidence_bundle_items_missing")
	}
	for _, key := range []string{
		"evidence:release_final_gate",
		"evidence:backup_manifest",
		"evidence:audit_coverage",
		"evidence:project_inventory:" + completionAuditTargetProjectKey,
	} {
		if !releaseEvidenceBundleHasReadyItem(bundle, key) {
			blockers = append(blockers, "release_evidence_bundle_item_not_ready:"+key)
		}
	}
	blockers = append(blockers, completionAuditSnapshotReleaseEvidenceBundleProjectIdentityBlockers(bundle)...)
	return uniqueStrings(blockers)
}

func completionAuditSnapshotReleaseEvidenceBundleProjectIdentityBlockers(bundle ReleaseEvidenceBundle) []string {
	item, ok := releaseEvidenceBundleItemByKey(bundle, "evidence:project_inventory:"+completionAuditTargetProjectKey)
	if !ok {
		return []string{"release_evidence_bundle_project_inventory_missing"}
	}
	blockers := []string{}
	record := Record{
		Key:             metadataString(item.Metadata, "project_key"),
		Kind:            metadataString(item.Metadata, "project_kind"),
		Adapter:         metadataString(item.Metadata, "adapter"),
		WorkflowProfile: metadataString(item.Metadata, "workflow_profile"),
		DefaultBranch:   metadataString(item.Metadata, "default_branch"),
		RootPath:        metadataString(item.Metadata, "root_path"),
	}
	for _, blocker := range completionAuditSnapshotRealProjectIdentityBlockers(record) {
		blockers = append(blockers, "release_evidence_bundle_"+blocker)
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotReleaseEvidenceBundleBindingBlockers(latest CompletionAuditSnapshot, bundle ReleaseEvidenceBundle) []string {
	blockers := completionAuditSnapshotReleaseEvidenceBundleBlockers(bundle)
	latestHash := metadataString(latest.Metadata, "release_evidence_bundle_hash")
	if latestHash == "" {
		blockers = append(blockers, "snapshot_release_evidence_bundle_hash_missing")
	}
	if bundle.BundleHash != "" && latestHash != "" && latestHash != bundle.BundleHash {
		blockers = append(blockers, "snapshot_release_evidence_bundle_hash_mismatch")
	}
	if metadataString(latest.Metadata, "release_evidence_bundle_status") != "ready" {
		blockers = append(blockers, "snapshot_release_evidence_bundle_status_not_ready")
	}
	if metadataString(latest.Metadata, "release_evidence_bundle_mode") != "read_only_release_evidence_bundle" {
		blockers = append(blockers, "snapshot_release_evidence_bundle_mode_invalid")
	}
	if !metadataBool(latest.Metadata, "release_evidence_bundle_ready") {
		blockers = append(blockers, "snapshot_release_evidence_bundle_ready_false")
	}
	itemCount := metadataInt64(latest.Metadata, "release_evidence_bundle_item_count")
	if itemCount == 0 {
		blockers = append(blockers, "snapshot_release_evidence_bundle_item_count_missing")
	}
	if itemCount != 0 && len(bundle.Items) > 0 && itemCount != int64(len(bundle.Items)) {
		blockers = append(blockers, "snapshot_release_evidence_bundle_item_count_mismatch")
	}
	return uniqueStrings(blockers)
}

func releaseEvidenceBundleHasReadyItem(bundle ReleaseEvidenceBundle, key string) bool {
	item, ok := releaseEvidenceBundleItemByKey(bundle, key)
	return ok && item.Status == "ready"
}

func releaseEvidenceBundleItemByKey(bundle ReleaseEvidenceBundle, key string) (ReleaseEvidenceBundleItem, bool) {
	for _, item := range bundle.Items {
		if item.Key == key {
			return item, true
		}
	}
	return ReleaseEvidenceBundleItem{}, false
}

func validateCompletionAuditSnapshotEvidenceClass(options RecordCompletionAuditSnapshotOptions) error {
	if options.EvidenceClass != completionAuditSnapshotEvidenceClassReleaseCandidate {
		return nil
	}
	if options.EvidenceURI == "" {
		return fmt.Errorf("release_candidate completion audit snapshot requires evidence URI")
	}
	if options.Summary == "" {
		return fmt.Errorf("release_candidate completion audit snapshot requires summary")
	}
	if completionAuditSnapshotContainsFixtureMarker(options.ReleaseCandidateLabel) ||
		completionAuditSnapshotContainsFixtureMarker(options.EvidenceURI) ||
		completionAuditSnapshotContainsFixtureMarker(options.Summary) {
		return fmt.Errorf("release_candidate completion audit snapshot cannot use fixture-labeled evidence")
	}
	if completionAuditSnapshotUsesMechanismEvidenceURI(options.EvidenceURI) {
		return fmt.Errorf("release_candidate completion audit snapshot requires reviewed evidence URI, not local script/smoke evidence")
	}
	if !completionAuditSnapshotUsesReleaseCandidateEvidenceURI(options.EvidenceURI) {
		return fmt.Errorf("release_candidate completion audit snapshot requires release-candidate evidence URI")
	}
	if blockers := completionAuditSnapshotReviewMetadataBlockers(completionAuditSnapshotReviewMetadataFromOptions(options)); len(blockers) > 0 {
		return fmt.Errorf("release_candidate completion audit snapshot requires approved review metadata: %s", strings.Join(blockers, ","))
	}
	return nil
}

func completionAuditSnapshotReviewMetadataFromOptions(options RecordCompletionAuditSnapshotOptions) map[string]any {
	reviewedAt := strings.TrimSpace(metadataString(options.Metadata, "reviewed_at"))
	if !options.ReviewedAt.IsZero() {
		reviewedAt = options.ReviewedAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"review_decision": strings.ToLower(strings.TrimSpace(firstNonEmptyString(options.ReviewDecision, metadataString(options.Metadata, "review_decision")))),
		"reviewed_by":     strings.TrimSpace(firstNonEmptyString(options.ReviewedBy, metadataString(options.Metadata, "reviewed_by"))),
		"reviewed_at":     reviewedAt,
	}
}

func completionAuditSnapshotReviewMetadataBlockers(metadata map[string]any) []string {
	blockers := []string{}
	decision := strings.ToLower(strings.TrimSpace(metadataString(metadata, "review_decision")))
	if decision == "" {
		blockers = append(blockers, "snapshot_review_decision_missing")
	} else if decision != "approved" {
		blockers = append(blockers, "snapshot_review_decision_not_approved")
	}
	if strings.TrimSpace(metadataString(metadata, "reviewed_by")) == "" {
		blockers = append(blockers, "snapshot_reviewed_by_missing")
	}
	reviewedAt := strings.TrimSpace(metadataString(metadata, "reviewed_at"))
	if reviewedAt == "" {
		blockers = append(blockers, "snapshot_reviewed_at_missing")
	} else if parsed, err := time.Parse(time.RFC3339, reviewedAt); err != nil || parsed.IsZero() {
		blockers = append(blockers, "snapshot_reviewed_at_invalid")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotContainsFixtureMarker(value string) bool {
	return completionAuditSnapshotContainsNonReleaseMarker(value)
}

func completionAuditSnapshotContainsNonReleaseMarker(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{
		"fixture",
		"mock",
		"demo",
		"sample",
		"synthetic",
		"testdata",
		"placeholder",
		"dummy",
		"example",
	} {
		if completionAuditSnapshotHasMarkerToken(normalized, marker) {
			return true
		}
	}
	return false
}

func completionAuditSnapshotHasMarkerToken(value string, marker string) bool {
	searchFrom := 0
	for {
		index := strings.Index(value[searchFrom:], marker)
		if index < 0 {
			return false
		}
		index += searchFrom
		after := index + len(marker)
		if completionAuditSnapshotMarkerBoundary(value, index-1) && completionAuditSnapshotMarkerBoundary(value, after) {
			return true
		}
		searchFrom = after
		if searchFrom >= len(value) {
			return false
		}
	}
}

func completionAuditSnapshotMarkerBoundary(value string, index int) bool {
	if index < 0 || index >= len(value) {
		return true
	}
	char := value[index]
	return !(char >= 'a' && char <= 'z') && !(char >= '0' && char <= '9')
}

func completionAuditSnapshotUsesMechanismEvidenceURI(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(normalized, "local:") ||
		strings.HasPrefix(normalized, "fixture:") ||
		strings.HasPrefix(normalized, "script:") ||
		strings.HasPrefix(normalized, "scripts/") ||
		strings.Contains(normalized, "/scripts/") ||
		strings.Contains(normalized, "smoke-")
}

func completionAuditSnapshotUsesReleaseCandidateEvidenceURI(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	path := strings.SplitN(normalized, "#", 2)[0]
	path = strings.SplitN(path, "?", 2)[0]
	return strings.Contains(path, "release-candidate") || strings.Contains(path, "release_candidate")
}

func completionAuditSnapshotDefaultEvidenceRoot() string {
	root, err := os.Getwd()
	if err != nil {
		return "."
	}
	return root
}

func completionAuditSnapshotAuditEvidenceURIRefs(root string, snapshotURI string, proofURIs []string) ([]map[string]any, []string) {
	refs := []completionAuditSnapshotEvidenceURIRef{{URI: snapshotURI}}
	for _, uri := range proofURIs {
		refs = append(refs, completionAuditSnapshotEvidenceURIRef{URI: uri, RequireAnchor: true})
	}
	return completionAuditSnapshotAuditEvidenceURIRefsForRoot(root, refs)
}

type completionAuditSnapshotEvidenceURIRef struct {
	URI           string
	RequireAnchor bool
}

func completionAuditSnapshotAuditEvidenceURIRefsForRoot(root string, refs []completionAuditSnapshotEvidenceURIRef) ([]map[string]any, []string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, []string{"evidence_root_missing"}
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, []string{"evidence_root_invalid:" + root}
	}
	entries := []map[string]any{}
	blockers := []string{}
	seen := map[string]bool{}
	for _, ref := range refs {
		uri := strings.TrimSpace(ref.URI)
		if uri == "" || seen[uri] {
			continue
		}
		seen[uri] = true
		entry, refBlockers := completionAuditSnapshotAuditEvidenceURIRef(absRoot, ref)
		if len(refBlockers) > 0 {
			blockers = append(blockers, refBlockers...)
			continue
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		blockers = append(blockers, "evidence_uri_file_audit_missing")
	}
	return entries, uniqueStrings(blockers)
}

func completionAuditSnapshotAuditEvidenceURIRef(root string, ref completionAuditSnapshotEvidenceURIRef) (map[string]any, []string) {
	uri := strings.TrimSpace(ref.URI)
	pathPart, anchor, hasAnchor := completionAuditSnapshotEvidenceURIPathAndAnchor(uri)
	cleanPath := filepath.Clean(filepath.FromSlash(pathPart))
	blockers := []string{}
	if pathPart == "" || cleanPath == "." {
		blockers = append(blockers, "evidence_uri_path_missing:"+uri)
	}
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		blockers = append(blockers, "evidence_uri_path_escapes_repo:"+uri)
	}
	if !strings.HasPrefix(filepath.ToSlash(cleanPath), "docs/") {
		blockers = append(blockers, "evidence_uri_path_not_docs:"+uri)
	}
	if !strings.HasSuffix(strings.ToLower(cleanPath), ".md") {
		blockers = append(blockers, "evidence_uri_path_not_markdown:"+uri)
	}
	if !completionAuditSnapshotUsesReleaseCandidateEvidenceURI(uri) {
		blockers = append(blockers, "evidence_uri_not_release_candidate:"+uri)
	}
	if ref.RequireAnchor && !hasAnchor {
		blockers = append(blockers, "evidence_uri_anchor_missing:"+uri)
	}
	if len(blockers) > 0 {
		return nil, blockers
	}

	fullPath := filepath.Join(root, cleanPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, []string{"evidence_uri_path_invalid:" + uri}
	}
	if !completionAuditSnapshotPathWithinRoot(root, absPath) {
		return nil, []string{"evidence_uri_path_escapes_repo:" + uri}
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, []string{"evidence_uri_file_missing:" + uri}
	}
	anchor = strings.TrimSpace(anchor)
	if hasAnchor {
		if anchor == "" {
			return nil, []string{"evidence_uri_anchor_missing:" + uri}
		}
		if !completionAuditSnapshotMarkdownHasAnchor(content, anchor) {
			return nil, []string{"evidence_uri_anchor_not_found:" + uri}
		}
	}
	sum := sha256.Sum256(content)
	entry := map[string]any{
		"uri":        uri,
		"path":       filepath.ToSlash(cleanPath),
		"sha256":     hex.EncodeToString(sum[:]),
		"size_bytes": int64(len(content)),
	}
	if hasAnchor {
		entry["anchor"] = strings.TrimPrefix(anchor, "#")
	}
	return entry, nil
}

func completionAuditSnapshotEvidenceURIPathAndAnchor(uri string) (string, string, bool) {
	beforeHash, anchor, hasAnchor := strings.Cut(strings.TrimSpace(uri), "#")
	pathPart, _, _ := strings.Cut(beforeHash, "?")
	return strings.TrimSpace(pathPart), strings.TrimSpace(anchor), hasAnchor
}

func completionAuditSnapshotPathWithinRoot(root string, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "..")
}

func completionAuditSnapshotMarkdownHasAnchor(content []byte, anchor string) bool {
	want := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(anchor)), "#")
	for _, line := range strings.Split(string(content), "\n") {
		got, ok := completionAuditSnapshotMarkdownHeadingAnchor(line)
		if ok && got == want {
			return true
		}
	}
	return false
}

func completionAuditSnapshotMarkdownHeadingAnchor(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	title := strings.TrimLeft(trimmed, "#")
	if title == trimmed {
		return "", false
	}
	title = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(title), "#"))
	if title == "" {
		return "", false
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-':
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-"), builder.Len() > 0
}

func completionAuditSnapshotProofEvidenceURIs(audit CompletionAudit) []string {
	uris := []string{}
	for _, item := range audit.Items {
		for key, value := range item.Metadata {
			if !strings.Contains(strings.ToLower(key), "evidence_uri") {
				continue
			}
			uris = append(uris, completionAuditSnapshotEvidenceURIValues(value)...)
		}
	}
	uris = uniqueStrings(uris)
	sort.Strings(uris)
	return uris
}

func completionAuditSnapshotProofEvidenceURIMap(audit CompletionAudit) map[string]string {
	out := map[string]string{}
	for _, item := range audit.Items {
		for _, key := range completionAuditSnapshotRequiredProofEvidenceURIKeysForItem(item.Key) {
			values := completionAuditSnapshotEvidenceURIValues(item.Metadata[key])
			if len(values) == 0 {
				continue
			}
			out[item.Key+"."+key] = values[0]
		}
	}
	return out
}

func completionAuditSnapshotProofProvenanceMap(audit CompletionAudit) map[string]string {
	out := map[string]string{}
	for _, item := range audit.Items {
		for _, key := range completionAuditSnapshotRequiredProofProvenanceKeysForItem(item.Key) {
			value := strings.TrimSpace(metadataString(item.Metadata, key))
			if value == "" {
				continue
			}
			out[item.Key+"."+key] = value
		}
	}
	return out
}

func completionAuditSnapshotMechanismEvidenceURIs(audit CompletionAudit) []string {
	return completionAuditSnapshotMechanismEvidenceURIValues(completionAuditSnapshotProofEvidenceURIs(audit))
}

func completionAuditSnapshotMissingProofEvidenceURIs(audit CompletionAudit) []string {
	missing := []string{}
	seenItems := map[string]bool{}
	for _, item := range audit.Items {
		requiredKeys := completionAuditSnapshotRequiredProofEvidenceURIKeysForItem(item.Key)
		if len(requiredKeys) == 0 {
			continue
		}
		seenItems[item.Key] = true
		for _, key := range requiredKeys {
			if len(completionAuditSnapshotEvidenceURIValues(item.Metadata[key])) == 0 {
				missing = append(missing, item.Key+"."+key)
			}
		}
	}
	for _, itemKey := range completionAuditSnapshotRequiredProofEvidenceItemKeys() {
		if !seenItems[itemKey] {
			missing = append(missing, "audit_item_missing:"+itemKey)
		}
	}
	return uniqueStrings(missing)
}

func completionAuditSnapshotMissingProofEventIDs(audit CompletionAudit) []string {
	missing := []string{}
	ids := completionAuditProofEventIDs(audit)
	for _, key := range completionAuditSnapshotRequiredProofEventIDKeys() {
		if ids[key] == 0 {
			missing = append(missing, key)
		}
	}
	return uniqueStrings(missing)
}

func completionAuditSnapshotRequiredProofEvidenceItemKeys() []string {
	return []string{
		"E1_design_source_alignment",
		"E2_phase_task_matrix",
		"E3_command_api_smoke_evidence",
		"E4_areamatrix_dogfood_completion",
		"E5_release_packaging_preview",
		"E6_backup_restore_artifact_retention",
		"E7_operations_readiness",
		"E8_security_permission_isolation",
		"E9_areamatrix_protected_path_proof",
	}
}

func completionAuditSnapshotRequiredProofEvidenceURIKeys() []string {
	keys := []string{}
	for _, itemKey := range completionAuditSnapshotRequiredProofEvidenceItemKeys() {
		for _, metadataKey := range completionAuditSnapshotRequiredProofEvidenceURIKeysForItem(itemKey) {
			keys = append(keys, itemKey+"."+metadataKey)
		}
	}
	return keys
}

func completionAuditSnapshotRequiredProofEventIDKeys() []string {
	keys := []string{}
	for _, itemKey := range completionAuditSnapshotRequiredProofEvidenceItemKeys() {
		for _, metadataKey := range completionAuditSnapshotRequiredProofEventIDKeysForItem(itemKey) {
			keys = append(keys, itemKey+"."+metadataKey)
		}
	}
	return keys
}

func completionAuditSnapshotRequiredProofProvenanceKeys() []string {
	keys := []string{}
	for _, itemKey := range completionAuditSnapshotRequiredProofEvidenceItemKeys() {
		for _, metadataKey := range completionAuditSnapshotRequiredProofProvenanceKeysForItem(itemKey) {
			keys = append(keys, itemKey+"."+metadataKey)
		}
	}
	return keys
}

func completionAuditSnapshotRequiredProofEvidenceURIKeysForItem(itemKey string) []string {
	switch itemKey {
	case "E1_design_source_alignment":
		return []string{"latest_source_alignment_proof_evidence_uri"}
	case "E2_phase_task_matrix":
		return []string{"latest_task_matrix_proof_evidence_uri"}
	case "E3_command_api_smoke_evidence":
		return []string{"latest_validation_proof_evidence_uri"}
	case "E4_areamatrix_dogfood_completion":
		return []string{
			"latest_archive_proof_evidence_uri",
			"latest_shim_retirement_proof_evidence_uri",
			"latest_execution_cutover_proof_evidence_uri",
		}
	case "E5_release_packaging_preview":
		return []string{"latest_release_packaging_proof_evidence_uri"}
	case "E6_backup_restore_artifact_retention":
		return []string{"latest_backup_restore_proof_evidence_uri"}
	case "E7_operations_readiness":
		return []string{"latest_operations_smoke_proof_evidence_uri"}
	case "E8_security_permission_isolation":
		return []string{"latest_security_closure_proof_evidence_uri"}
	case "E9_areamatrix_protected_path_proof":
		return []string{"latest_proof_evidence_uri"}
	default:
		return nil
	}
}

func completionAuditSnapshotRequiredProofEventIDKeysForItem(itemKey string) []string {
	switch itemKey {
	case "E1_design_source_alignment":
		return []string{"latest_source_alignment_proof_event_id"}
	case "E2_phase_task_matrix":
		return []string{"latest_task_matrix_proof_event_id"}
	case "E3_command_api_smoke_evidence":
		return []string{"latest_validation_proof_event_id"}
	case "E4_areamatrix_dogfood_completion":
		return []string{
			"latest_archive_proof_event_id",
			"latest_shim_retirement_proof_event_id",
			"latest_execution_cutover_proof_event_id",
		}
	case "E5_release_packaging_preview":
		return []string{"latest_release_packaging_proof_event_id"}
	case "E6_backup_restore_artifact_retention":
		return []string{"latest_backup_restore_proof_event_id"}
	case "E7_operations_readiness":
		return []string{"latest_operations_smoke_proof_event_id"}
	case "E8_security_permission_isolation":
		return []string{"latest_security_closure_proof_event_id"}
	case "E9_areamatrix_protected_path_proof":
		return []string{"latest_proof_event_id"}
	default:
		return nil
	}
}

func completionAuditSnapshotRequiredProofProvenanceKeysForItem(itemKey string) []string {
	switch itemKey {
	case "E7_operations_readiness":
		return []string{"latest_operations_smoke_proof_key"}
	default:
		return nil
	}
}

func completionAuditSnapshotProofEvidenceURIBindingBlockers(latest CompletionAuditSnapshot) []string {
	uriMap := metadataStringMap(latest.Metadata, "proof_evidence_uri_map")
	values := completionAuditSnapshotEvidenceURIValues(latest.Metadata["proof_evidence_uris"])
	blockers := completionAuditSnapshotProofEvidenceURIMapBindingBlockers(values, uriMap)
	if metadataInt64(latest.Metadata, "proof_evidence_uri_count") != int64(len(values)) {
		blockers = append(blockers, "snapshot_proof_evidence_uri_count_mismatch")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotProofEventIDBindingBlockers(latest CompletionAuditSnapshot) []string {
	blockers := completionAuditSnapshotProofEventIDMapBindingBlockers(latest.ProofEventIDs)
	metadataIDs := metadataInt64Map(latest.Metadata, "proof_event_ids")
	if len(metadataIDs) == 0 {
		return blockers
	}
	for _, key := range completionAuditSnapshotRequiredProofEventIDKeys() {
		topLevelValue := latest.ProofEventIDs[key]
		metadataValue := metadataIDs[key]
		if topLevelValue != 0 && metadataValue != 0 && topLevelValue != metadataValue {
			blockers = append(blockers, "snapshot_proof_event_id_metadata_mismatch:"+key)
		}
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotProofProvenanceBindingBlockers(provenanceMap map[string]string) []string {
	blockers := []string{}
	for _, key := range completionAuditSnapshotRequiredProofProvenanceKeys() {
		value := strings.TrimSpace(provenanceMap[key])
		if value == "" {
			blockers = append(blockers, "snapshot_proof_provenance_missing:"+key)
			continue
		}
		if completionAuditSnapshotProofProvenanceMarksFixture(value) {
			blockers = append(blockers, "snapshot_operations_proof_key_fixture")
		}
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotProofProvenanceMarksFixture(value string) bool {
	return completionAuditSnapshotContainsNonReleaseMarker(value)
}

func completionAuditSnapshotEvidenceURIFileAuditBindingBlockers(latest CompletionAuditSnapshot, root string) []string {
	status := metadataString(latest.Metadata, "evidence_uri_file_audit_status")
	blockers := []string{}
	if status == "" {
		blockers = append(blockers, "snapshot_evidence_uri_file_audit_status_missing")
	}
	if status != "pass" {
		blockers = append(blockers, "snapshot_evidence_uri_file_audit_status_not_pass")
	}
	storedEntries := completionAuditSnapshotEvidenceURIFileAuditEntries(latest.Metadata["evidence_uri_file_audit"])
	if len(storedEntries) == 0 {
		blockers = append(blockers, "snapshot_evidence_uri_file_audit_missing")
	}
	if metadataInt64(latest.Metadata, "evidence_uri_file_audit_count") != int64(len(storedEntries)) {
		blockers = append(blockers, "snapshot_evidence_uri_file_audit_count_mismatch")
	}
	currentEntries, currentBlockers := completionAuditSnapshotAuditEvidenceURIRefs(root, latest.EvidenceURI, completionAuditSnapshotEvidenceURIValues(latest.Metadata["proof_evidence_uris"]))
	if len(currentBlockers) > 0 {
		for _, blocker := range currentBlockers {
			blockers = append(blockers, "current_"+blocker)
		}
	}
	storedByURI := completionAuditSnapshotEvidenceURIFileAuditByURI(storedEntries)
	currentByURI := completionAuditSnapshotEvidenceURIFileAuditByURI(currentEntries)
	for uri, stored := range storedByURI {
		current, ok := currentByURI[uri]
		if !ok {
			blockers = append(blockers, "current_evidence_uri_file_audit_missing:"+uri)
			continue
		}
		for _, key := range []string{"path", "anchor", "sha256"} {
			if metadataString(stored, key) != metadataString(current, key) {
				blockers = append(blockers, "snapshot_evidence_uri_file_audit_"+key+"_mismatch:"+uri)
			}
		}
		if metadataInt64(stored, "size_bytes") != metadataInt64(current, "size_bytes") {
			blockers = append(blockers, "snapshot_evidence_uri_file_audit_size_mismatch:"+uri)
		}
	}
	for uri := range currentByURI {
		if _, ok := storedByURI[uri]; !ok {
			blockers = append(blockers, "snapshot_evidence_uri_file_audit_extra_current:"+uri)
		}
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotEvidenceURIFileAuditEntries(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, entry := range typed {
			out = append(out, entry)
		}
		return out
	case []any:
		out := []map[string]any{}
		for _, item := range typed {
			if entry, ok := item.(map[string]any); ok {
				out = append(out, entry)
			}
		}
		return out
	default:
		return nil
	}
}

func completionAuditSnapshotEvidenceURIFileAuditByURI(entries []map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, entry := range entries {
		uri := strings.TrimSpace(metadataString(entry, "uri"))
		if uri != "" {
			out[uri] = entry
		}
	}
	return out
}

func completionAuditSnapshotProofEvidenceURIMapBindingBlockers(values []string, uriMap map[string]string) []string {
	blockers := []string{}
	requiredKeys := completionAuditSnapshotRequiredProofEvidenceURIKeys()
	indexed := map[string]bool{}
	for _, value := range values {
		indexed[value] = true
	}
	requiredValues := map[string]bool{}
	for _, key := range requiredKeys {
		value := strings.TrimSpace(uriMap[key])
		if value == "" {
			blockers = append(blockers, "snapshot_proof_evidence_uri_missing:"+key)
			continue
		}
		if completionAuditSnapshotContainsFixtureMarker(value) || completionAuditSnapshotUsesMechanismEvidenceURI(value) {
			blockers = append(blockers, "snapshot_proof_evidence_uri_mechanism:"+key)
		}
		if !completionAuditSnapshotUsesReleaseCandidateEvidenceURI(value) {
			blockers = append(blockers, "snapshot_proof_evidence_uri_not_release_candidate:"+key)
		}
		if !indexed[value] {
			blockers = append(blockers, "snapshot_proof_evidence_uri_not_indexed:"+key)
		}
		requiredValues[value] = true
	}
	if len(values) == 0 {
		blockers = append(blockers, "snapshot_proof_evidence_uris_missing")
	}
	if len(values) < len(requiredKeys) {
		blockers = append(blockers, "snapshot_proof_evidence_uri_count_too_low")
	}
	if len(requiredValues) < len(requiredKeys) {
		blockers = append(blockers, "snapshot_proof_evidence_uri_not_distinct")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotProofEventIDMapBindingBlockers(ids map[string]int64) []string {
	blockers := []string{}
	requiredKeys := completionAuditSnapshotRequiredProofEventIDKeys()
	distinctIDs := map[int64]bool{}
	for _, key := range requiredKeys {
		value := ids[key]
		if value <= 0 {
			blockers = append(blockers, "snapshot_proof_event_id_missing:"+key)
			continue
		}
		distinctIDs[value] = true
	}
	if len(ids) == 0 {
		blockers = append(blockers, "snapshot_proof_event_ids_missing")
	}
	if len(ids) < len(requiredKeys) {
		blockers = append(blockers, "snapshot_proof_event_id_count_too_low")
	}
	if len(distinctIDs) < len(requiredKeys) {
		blockers = append(blockers, "snapshot_proof_event_id_not_distinct")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotMissingStringBindingKeys(requiredKeys []string, values map[string]string) []string {
	missing := []string{}
	for _, key := range requiredKeys {
		if strings.TrimSpace(values[key]) == "" {
			missing = append(missing, key)
		}
	}
	return uniqueStrings(missing)
}

func completionAuditSnapshotMissingInt64BindingKeys(requiredKeys []string, values map[string]int64) []string {
	missing := []string{}
	for _, key := range requiredKeys {
		if values[key] <= 0 {
			missing = append(missing, key)
		}
	}
	return uniqueStrings(missing)
}

func completionAuditSnapshotCurrentProofEvidenceURIMapBindingBlockers(values []string, uriMap map[string]string) []string {
	blockers := []string{}
	requiredKeys := completionAuditSnapshotRequiredProofEvidenceURIKeys()
	indexed := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			indexed[trimmed] = true
		}
	}
	requiredValues := map[string]bool{}
	for _, key := range requiredKeys {
		value := strings.TrimSpace(uriMap[key])
		if value == "" {
			blockers = append(blockers, "current_proof_evidence_uri_missing:"+key)
			continue
		}
		if completionAuditSnapshotContainsFixtureMarker(value) || completionAuditSnapshotUsesMechanismEvidenceURI(value) {
			blockers = append(blockers, "current_proof_evidence_uri_mechanism:"+key)
		}
		if !completionAuditSnapshotUsesReleaseCandidateEvidenceURI(value) {
			blockers = append(blockers, "current_proof_evidence_uri_not_release_candidate:"+key)
		}
		if !indexed[value] {
			blockers = append(blockers, "current_proof_evidence_uri_not_indexed:"+key)
		}
		requiredValues[value] = true
	}
	if len(values) == 0 {
		blockers = append(blockers, "current_proof_evidence_uris_missing")
	}
	if len(values) < len(requiredKeys) {
		blockers = append(blockers, "current_proof_evidence_uri_count_too_low")
	}
	if len(requiredValues) < len(requiredKeys) {
		blockers = append(blockers, "current_proof_evidence_uri_not_distinct")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotCurrentProofEventIDMapBindingBlockers(ids map[string]int64) []string {
	blockers := []string{}
	requiredKeys := completionAuditSnapshotRequiredProofEventIDKeys()
	distinctIDs := map[int64]bool{}
	for _, key := range requiredKeys {
		value := ids[key]
		if value <= 0 {
			blockers = append(blockers, "current_proof_event_id_missing:"+key)
			continue
		}
		distinctIDs[value] = true
	}
	if len(ids) == 0 {
		blockers = append(blockers, "current_proof_event_ids_missing")
	}
	if len(ids) < len(requiredKeys) {
		blockers = append(blockers, "current_proof_event_id_count_too_low")
	}
	if len(distinctIDs) < len(requiredKeys) {
		blockers = append(blockers, "current_proof_event_id_not_distinct")
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotCurrentProofProvenanceBindingBlockers(provenanceMap map[string]string) []string {
	blockers := []string{}
	for _, key := range completionAuditSnapshotRequiredProofProvenanceKeys() {
		value := strings.TrimSpace(provenanceMap[key])
		if value == "" {
			blockers = append(blockers, "current_proof_provenance_missing:"+key)
			continue
		}
		if completionAuditSnapshotProofProvenanceMarksFixture(value) {
			blockers = append(blockers, "current_operations_proof_key_fixture")
		}
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotCurrentProofBindingBlockers(latest CompletionAuditSnapshot, current completionAuditSnapshotCurrentAuditBinding) []string {
	blockers := []string{}
	requiredURIKeys := completionAuditSnapshotRequiredProofEvidenceURIKeys()
	latestURIMap := metadataStringMap(latest.Metadata, "proof_evidence_uri_map")
	currentURIMap := current.ProofEvidenceURIMap
	latestURIs := completionAuditSnapshotEvidenceURIValues(latest.Metadata["proof_evidence_uris"])
	if len(currentURIMap) == 0 {
		blockers = append(blockers, "current_proof_evidence_uri_map_missing")
	}
	if len(current.ProofEvidenceURIs) == 0 {
		blockers = append(blockers, "current_proof_evidence_uris_missing")
	}
	currentURISet := map[string]bool{}
	for _, value := range current.ProofEvidenceURIs {
		currentURISet[strings.TrimSpace(value)] = true
	}
	if len(currentURISet) > 0 {
		for _, value := range latestURIs {
			if !currentURISet[strings.TrimSpace(value)] {
				blockers = append(blockers, "snapshot_proof_evidence_uri_set_mismatch:"+strings.TrimSpace(value))
			}
		}
		if len(latestURIs) != len(currentURISet) {
			blockers = append(blockers, "snapshot_proof_evidence_uri_set_count_mismatch")
		}
	}
	for _, key := range requiredURIKeys {
		latestValue := strings.TrimSpace(latestURIMap[key])
		currentValue := strings.TrimSpace(currentURIMap[key])
		if latestValue == "" {
			blockers = append(blockers, "snapshot_proof_evidence_uri_missing:"+key)
			continue
		}
		if currentValue == "" {
			blockers = append(blockers, "current_proof_evidence_uri_missing:"+key)
			continue
		}
		if latestValue != currentValue {
			blockers = append(blockers, "snapshot_proof_evidence_uri_map_mismatch:"+key)
		}
	}

	requiredEventIDKeys := completionAuditSnapshotRequiredProofEventIDKeys()
	currentEventIDs := current.ProofEventIDs
	if len(currentEventIDs) == 0 {
		blockers = append(blockers, "current_proof_event_ids_missing")
	}
	for _, key := range requiredEventIDKeys {
		latestValue := latest.ProofEventIDs[key]
		currentValue := currentEventIDs[key]
		if latestValue <= 0 {
			blockers = append(blockers, "snapshot_proof_event_id_missing:"+key)
			continue
		}
		if currentValue <= 0 {
			blockers = append(blockers, "current_proof_event_id_missing:"+key)
			continue
		}
		if latestValue != currentValue {
			blockers = append(blockers, "snapshot_proof_event_id_map_mismatch:"+key)
		}
	}
	requiredProvenanceKeys := completionAuditSnapshotRequiredProofProvenanceKeys()
	latestProvenanceMap := metadataStringMap(latest.Metadata, "proof_provenance_map")
	currentProvenanceMap := current.ProofProvenanceMap
	if len(currentProvenanceMap) == 0 {
		blockers = append(blockers, "current_proof_provenance_map_missing")
	}
	for _, key := range requiredProvenanceKeys {
		latestValue := strings.TrimSpace(latestProvenanceMap[key])
		currentValue := strings.TrimSpace(currentProvenanceMap[key])
		if latestValue == "" {
			blockers = append(blockers, "snapshot_proof_provenance_missing:"+key)
			continue
		}
		if currentValue == "" {
			blockers = append(blockers, "current_proof_provenance_missing:"+key)
			continue
		}
		if latestValue != currentValue {
			blockers = append(blockers, "snapshot_proof_provenance_map_mismatch:"+key)
		}
	}
	return uniqueStrings(blockers)
}

func completionAuditSnapshotMechanismEvidenceURIValues(values []string) []string {
	mechanism := []string{}
	for _, value := range values {
		if completionAuditSnapshotContainsFixtureMarker(value) || completionAuditSnapshotUsesMechanismEvidenceURI(value) {
			mechanism = append(mechanism, value)
		}
	}
	return uniqueStrings(mechanism)
}

func completionAuditSnapshotEvidenceURIValues(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	case []string:
		values := []string{}
		for _, item := range typed {
			if strings.TrimSpace(item) != "" {
				values = append(values, strings.TrimSpace(item))
			}
		}
		return values
	case []any:
		values := []string{}
		for _, item := range typed {
			values = append(values, completionAuditSnapshotEvidenceURIValues(item)...)
		}
		return values
	default:
		return nil
	}
}

func completionAuditHash(audit CompletionAudit) (string, error) {
	stableAudit := audit
	stableAudit.Real100Guardrail = Real100Guardrail{}
	stableAudit.GeneratedAt = time.Time{}
	stableAudit.Items = make([]CompletionAuditItem, len(audit.Items))
	copy(stableAudit.Items, audit.Items)
	for i := range stableAudit.Items {
		stableAudit.Items[i].Metadata = completionAuditHashStableMetadata(stableAudit.Items[i])
	}
	payload, err := json.Marshal(stableAudit)
	if err != nil {
		return "", fmt.Errorf("marshal completion audit snapshot hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func completionAuditHashStableMetadata(item CompletionAuditItem) map[string]any {
	if item.Metadata == nil {
		return nil
	}
	metadata := copyMap(item.Metadata)
	if item.Key == "E6_backup_restore_artifact_retention" {
		delete(metadata, "current_backup_manifest_hash")
		delete(metadata, "current_restore_plan_manifest_hash")
	}
	if item.Key == "E7_operations_readiness" {
		delete(metadata, "latest_operations_smoke_proof_age_seconds")
	}
	return metadata
}

func completionAuditProofEventIDs(audit CompletionAudit) map[string]int64 {
	ids := map[string]int64{}
	for _, item := range audit.Items {
		for _, key := range sortedMetadataKeys(item.Metadata) {
			if !strings.Contains(key, "proof_event_id") {
				continue
			}
			if value := metadataInt64(item.Metadata, key); value != 0 {
				ids[item.Key+"."+key] = value
			}
		}
	}
	return ids
}

func sortedMetadataKeys(metadata map[string]any) []string {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func completionAuditSnapshotRequestHash(record Record, result CompletionAuditSnapshot, options RecordCompletionAuditSnapshotOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":                        completionAuditSnapshotCommandType,
		"project_id":                          record.ID,
		"project_key":                         record.Key,
		"project_root":                        record.RootPath,
		"project_kind":                        record.Kind,
		"adapter":                             record.Adapter,
		"workflow_profile":                    record.WorkflowProfile,
		"default_branch":                      record.DefaultBranch,
		"audit_status":                        result.AuditStatus,
		"audit_scope":                         result.AuditScope,
		"audit_mode":                          result.Metadata["audit_mode"],
		"audit_hash":                          result.AuditHash,
		"release_candidate_label":             options.ReleaseCandidateLabel,
		"evidence_class":                      options.EvidenceClass,
		"evidence_uri":                        options.EvidenceURI,
		"summary":                             options.Summary,
		"review_decision":                     metadataString(result.Metadata, "review_decision"),
		"reviewed_by":                         metadataString(result.Metadata, "reviewed_by"),
		"reviewed_at":                         metadataString(result.Metadata, "reviewed_at"),
		"review_metadata_status":              metadataString(result.Metadata, "review_metadata_status"),
		"release_evidence_bundle_hash":        metadataString(result.Metadata, "release_evidence_bundle_hash"),
		"release_evidence_bundle_status":      metadataString(result.Metadata, "release_evidence_bundle_status"),
		"release_evidence_bundle_mode":        metadataString(result.Metadata, "release_evidence_bundle_mode"),
		"release_evidence_bundle_item_count":  metadataInt64(result.Metadata, "release_evidence_bundle_item_count"),
		"release_evidence_bundle_ready":       metadataBool(result.Metadata, "release_evidence_bundle_ready"),
		"proof_event_ids":                     result.ProofEventIDs,
		"proof_event_id_count":                metadataInt64(result.Metadata, "proof_event_id_count"),
		"required_proof_event_id_keys":        completionAuditSnapshotRequiredProofEventIDKeys(),
		"proof_evidence_uris":                 completionAuditSnapshotEvidenceURIValues(result.Metadata["proof_evidence_uris"]),
		"proof_evidence_uri_map":              metadataStringMap(result.Metadata, "proof_evidence_uri_map"),
		"proof_evidence_uri_count":            metadataInt64(result.Metadata, "proof_evidence_uri_count"),
		"required_proof_evidence_uri_keys":    completionAuditSnapshotRequiredProofEvidenceURIKeys(),
		"evidence_uri_file_audit":             completionAuditSnapshotEvidenceURIFileAuditEntries(result.Metadata["evidence_uri_file_audit"]),
		"evidence_uri_file_audit_count":       metadataInt64(result.Metadata, "evidence_uri_file_audit_count"),
		"evidence_uri_file_audit_status":      metadataString(result.Metadata, "evidence_uri_file_audit_status"),
		"proof_provenance_map":                metadataStringMap(result.Metadata, "proof_provenance_map"),
		"required_proof_provenance_keys":      completionAuditSnapshotRequiredProofProvenanceKeys(),
		"project_write_attempted":             result.ProjectWriteAttempted,
		"execution_write_attempted":           result.ExecutionWriteAttempted,
		"release_package_created":             result.ReleasePackageCreated,
		"publish_attempted":                   result.PublishAttempted,
		"restore_apply_attempted":             result.RestoreApplyAttempted,
		"secret_resolved":                     result.SecretResolved,
		"remote_worker_credentials_issued":    result.RemoteWorkerCredentialsIssued,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"commands_run":                        result.CommandsRun,
		"smoke_run_attempted":                 result.SmokeRunAttempted,
		"worker_started":                      result.WorkerStarted,
		"actor":                               options.Actor,
		"reason":                              options.Reason,
		"metadata":                            options.Metadata,
		"protected":                           true,
		"no_project_write":                    true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal completion audit snapshot request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func completionAuditSnapshotIdempotencyKey(record Record, result CompletionAuditSnapshot, options RecordCompletionAuditSnapshotOptions) string {
	prefix := result.AuditHash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	if options.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate {
		bundlePrefix := metadataString(result.Metadata, "release_evidence_bundle_hash")
		if len(bundlePrefix) > 16 {
			bundlePrefix = bundlePrefix[:16]
		}
		if bundlePrefix != "" {
			prefix = prefix + ":" + bundlePrefix
		}
	}
	return fmt.Sprintf("completion.audit_snapshot.record:%s:%s:%s:%s", record.Key, options.ReleaseCandidateLabel, options.EvidenceClass, prefix)
}

func insertCompletionAuditSnapshotEvent(ctx context.Context, tx pgx.Tx, result CompletionAuditSnapshot, options RecordCompletionAuditSnapshotOptions) (int64, error) {
	metadata, err := json.Marshal(completionAuditSnapshotEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal completion audit snapshot event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', 'Completion audit snapshot recorded', $3::jsonb)
RETURNING id`,
		result.Project.ID,
		completionAuditSnapshotEventType,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert completion audit snapshot event: %w", err)
	}
	return eventID, nil
}

func insertCompletionAuditSnapshotAuditEvent(ctx context.Context, tx pgx.Tx, result CompletionAuditSnapshot, options RecordCompletionAuditSnapshotOptions) (int64, error) {
	metadata, err := json.Marshal(completionAuditSnapshotCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal completion audit snapshot audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'completion_audit', 'completion_audit_snapshot', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		completionAuditSnapshotCommandType,
		result.AuditHash,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert completion audit snapshot audit event: %w", err)
	}
	return auditEventID, nil
}

func completionAuditSnapshotEventMetadata(result CompletionAuditSnapshot, options RecordCompletionAuditSnapshotOptions) map[string]any {
	metadata := completionAuditSnapshotCommandResponse(result)
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func completionAuditSnapshotCommandResponse(result CompletionAuditSnapshot) map[string]any {
	return map[string]any{
		"project_key":                         result.Project.Key,
		"status":                              result.Status,
		"decision":                            result.Decision,
		"message":                             result.Message,
		"audit_status":                        result.AuditStatus,
		"audit_scope":                         result.AuditScope,
		"audit_hash":                          result.AuditHash,
		"release_candidate_label":             result.ReleaseCandidateLabel,
		"evidence_class":                      result.EvidenceClass,
		"evidence_uri":                        result.EvidenceURI,
		"proof_event_ids":                     result.ProofEventIDs,
		"event_id":                            result.EventID,
		"audit_event_id":                      result.AuditEventID,
		"idempotency_key":                     result.IdempotencyKey,
		"project_write_attempted":             result.ProjectWriteAttempted,
		"execution_write_attempted":           result.ExecutionWriteAttempted,
		"release_package_created":             result.ReleasePackageCreated,
		"publish_attempted":                   result.PublishAttempted,
		"restore_apply_attempted":             result.RestoreApplyAttempted,
		"secret_resolved":                     result.SecretResolved,
		"remote_worker_credentials_issued":    result.RemoteWorkerCredentialsIssued,
		"area_matrix_protected_paths_touched": result.AreaMatrixProtectedPathsTouched,
		"commands_run":                        result.CommandsRun,
		"smoke_run_attempted":                 result.SmokeRunAttempted,
		"worker_started":                      result.WorkerStarted,
		"summary":                             metadataString(result.Metadata, "summary"),
		"metadata":                            result.Metadata,
	}
}

func loadCompletionAuditSnapshotByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (CompletionAuditSnapshot, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, completionAuditSnapshotCommandType, idempotencyKey)
	if err != nil {
		return CompletionAuditSnapshot{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return CompletionAuditSnapshot{
		Real100Guardrail:                CompletionAuditReal100Guardrail(),
		Project:                         record,
		Status:                          metadataString(response, "status"),
		Decision:                        metadataString(response, "decision"),
		Message:                         metadataString(response, "message"),
		AuditStatus:                     metadataString(response, "audit_status"),
		AuditScope:                      metadataString(response, "audit_scope"),
		AuditHash:                       metadataString(response, "audit_hash"),
		ReleaseCandidateLabel:           metadataString(response, "release_candidate_label"),
		EvidenceClass:                   metadataString(response, "evidence_class"),
		EvidenceURI:                     metadataString(response, "evidence_uri"),
		ProofEventIDs:                   metadataInt64Map(response, "proof_event_ids"),
		EventID:                         metadataInt64(response, "event_id"),
		AuditEventID:                    metadataInt64(response, "audit_event_id"),
		IdempotencyKey:                  idempotencyKey,
		ProjectWriteAttempted:           metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(response, "execution_write_attempted"),
		ReleasePackageCreated:           metadataBool(response, "release_package_created"),
		PublishAttempted:                metadataBool(response, "publish_attempted"),
		RestoreApplyAttempted:           metadataBool(response, "restore_apply_attempted"),
		SecretResolved:                  metadataBool(response, "secret_resolved"),
		RemoteWorkerCredentialsIssued:   metadataBool(response, "remote_worker_credentials_issued"),
		AreaMatrixProtectedPathsTouched: metadataBool(response, "area_matrix_protected_paths_touched"),
		CommandsRun:                     metadataBool(response, "commands_run"),
		SmokeRunAttempted:               metadataBool(response, "smoke_run_attempted"),
		WorkerStarted:                   metadataBool(response, "worker_started"),
		Metadata:                        metadata,
	}, nil
}

func completionAuditSnapshotFromEvent(record Record, event EventRecord) CompletionAuditSnapshot {
	result := completionAuditSnapshotFromMetadata(record, event.Metadata)
	result.EventID = event.ID
	result.CreatedAt = event.CreatedAt
	return result
}

func completionAuditSnapshotPackageAStatusProjectionBlockers(binding completionAuditSnapshotPackageAStatusProjectionBinding) []string {
	blockers := []string{}
	currentStableShape := binding.CurrentPreimageCaptured && binding.CurrentPreimage.SchemaStatus == "stable"
	if binding.LatestImportSourceHashError != "" {
		blockers = append(blockers, "package_a_latest_import_source_hash_missing")
	} else if strings.TrimSpace(binding.LatestImportSourceHash) == "" {
		blockers = append(blockers, "package_a_latest_import_source_hash_missing")
	}
	if binding.ProjectionQueryError != "" {
		blockers = append(blockers, "package_a_status_projection_query_failed")
	}
	if !binding.HasWrittenProjection {
		if currentStableShape {
			blockers = append(blockers, "package_a_status_projection_apply_provenance_missing")
		} else {
			blockers = append(blockers,
				"completion_audit_snapshot_package_a_not_applied",
				"package_a_status_projection_not_written",
			)
		}
	} else {
		projection := binding.LatestWrittenProjection
		if strings.TrimSpace(projection.TargetKind) != "project_status_json" {
			blockers = append(blockers, "package_a_status_projection_target_kind_mismatch")
		}
		if strings.TrimSpace(projection.TargetURI) != ".areaflow/status.json" {
			blockers = append(blockers, "package_a_status_projection_target_uri_mismatch")
		}
		if strings.TrimSpace(projection.WriteState) != "written" {
			blockers = append(blockers, "package_a_status_projection_not_written")
		}
		if strings.TrimSpace(projection.SourceHash) == "" ||
			(strings.TrimSpace(binding.LatestImportSourceHash) != "" && strings.TrimSpace(projection.SourceHash) != strings.TrimSpace(binding.LatestImportSourceHash)) {
			blockers = append(blockers, "package_a_status_projection_source_hash_stale")
		}
		if metadataString(projection.Metadata, "decision") != "allowed" {
			blockers = append(blockers, "package_a_status_projection_decision_not_allowed")
		}
		if metadataString(projection.Metadata, "command_type") != statusProjectionApplyCommandType {
			blockers = append(blockers, "package_a_status_projection_not_stably_bound_to_apply_event")
		}
		if !metadataBool(projection.Metadata, "post_write_verified") {
			blockers = append(blockers, "package_a_status_projection_post_write_not_verified")
		}
		if !metadataBool(projection.Metadata, "stable_projection_validated") {
			blockers = append(blockers, "package_a_status_projection_stable_projection_not_validated")
		}
		if !metadataBool(projection.Metadata, "protected_paths_verified") {
			blockers = append(blockers, "package_a_status_projection_protected_paths_not_verified")
		}
		if !metadataBool(projection.Metadata, "root_contained") {
			blockers = append(blockers, "package_a_status_projection_root_not_contained")
		}
		if metadataString(projection.Metadata, "apply_gate_status") != "pass" ||
			metadataString(projection.Metadata, "apply_gate_decision") != "go" ||
			!metadataBool(projection.Metadata, "apply_command_eligible") {
			blockers = append(blockers, "package_a_status_projection_apply_gate_not_pass")
		}
		if !metadataBool(projection.Metadata, "project_write_attempted") {
			blockers = append(blockers, "package_a_status_projection_project_write_not_recorded")
		}
		if metadataBool(projection.Metadata, "execution_write_attempted") {
			blockers = append(blockers, "package_a_status_projection_execution_write_attempted")
		}
		if metadataBool(projection.Metadata, "engine_call_attempted") {
			blockers = append(blockers, "package_a_status_projection_engine_call_attempted")
		}
	}
	if !binding.CurrentPreimageCaptured {
		blockers = append(blockers, "package_a_current_status_projection_preimage_missing")
	} else {
		if binding.CurrentPreimage.SchemaStatus != "stable" {
			blockers = append(blockers, "package_a_current_status_projection_not_stable")
		} else if strings.TrimSpace(binding.LatestImportSourceHash) != "" {
			currentSourceHash := strings.TrimSpace(binding.CurrentPreimage.SourceSnapshotHash)
			if currentSourceHash == "" {
				blockers = append(blockers, "package_a_current_status_projection_source_hash_missing")
			} else if currentSourceHash != strings.TrimSpace(binding.LatestImportSourceHash) {
				blockers = append(blockers, "package_a_current_status_projection_source_hash_stale")
			}
		}
		if binding.HasWrittenProjection {
			expectedHash := metadataString(binding.LatestWrittenProjection.Metadata, "post_write_sha256")
			if expectedHash == "" {
				expectedHash = metadataString(binding.LatestWrittenProjection.Metadata, "write_hash")
			}
			if expectedHash == "" {
				blockers = append(blockers, "package_a_status_projection_hash_missing")
			} else if binding.CurrentPreimage.SHA256 != "" && binding.CurrentPreimage.SHA256 != expectedHash {
				blockers = append(blockers, "package_a_current_status_projection_hash_mismatch")
			}
		}
	}
	if len(blockers) > 0 && !currentStableShape {
		blockers = append([]string{"package_a_status_projection_not_stable"}, blockers...)
	}
	return uniqueStrings(blockers)
}

func addCompletionAuditSnapshotPackageAStatusProjectionMetadata(metadata map[string]any, binding completionAuditSnapshotPackageAStatusProjectionBinding) {
	blockers := completionAuditSnapshotPackageAStatusProjectionBlockers(binding)
	metadata["package_a_status_projection_blockers"] = blockers
	metadata["package_a_status_projection_ready"] = len(blockers) == 0
	metadata["package_a_latest_import_source_hash"] = binding.LatestImportSourceHash
	metadata["package_a_latest_import_source_hash_error"] = binding.LatestImportSourceHashError
	metadata["package_a_status_projection_query_error"] = binding.ProjectionQueryError
	metadata["package_a_has_status_projection"] = binding.HasProjection
	metadata["package_a_has_written_status_projection"] = binding.HasWrittenProjection
	metadata["package_a_current_status_projection_schema_status"] = binding.CurrentPreimage.SchemaStatus
	metadata["package_a_current_status_projection_sha256"] = binding.CurrentPreimage.SHA256
	metadata["package_a_current_status_projection_source_snapshot_hash"] = binding.CurrentPreimage.SourceSnapshotHash
	metadata["package_a_current_status_projection_exists"] = binding.CurrentPreimage.Exists
	metadata["package_a_current_status_projection_readable"] = binding.CurrentPreimage.Readable
	metadata["package_a_current_status_projection_message"] = binding.CurrentPreimage.Message
	if binding.HasProjection {
		addCompletionAuditSnapshotStatusProjectionRecordMetadata(metadata, "package_a_latest_status_projection_", binding.LatestProjection)
	}
	if binding.HasWrittenProjection {
		addCompletionAuditSnapshotStatusProjectionRecordMetadata(metadata, "package_a_latest_written_status_projection_", binding.LatestWrittenProjection)
	}
}

func addCompletionAuditSnapshotStatusProjectionRecordMetadata(metadata map[string]any, prefix string, projection StatusProjectionRecord) {
	metadata[prefix+"id"] = projection.ID
	metadata[prefix+"target_kind"] = projection.TargetKind
	metadata[prefix+"target_uri"] = projection.TargetURI
	metadata[prefix+"summary_state"] = projection.SummaryState
	metadata[prefix+"source_hash"] = projection.SourceHash
	metadata[prefix+"write_state"] = projection.WriteState
	metadata[prefix+"command_type"] = metadataString(projection.Metadata, "command_type")
	metadata[prefix+"decision"] = metadataString(projection.Metadata, "decision")
	metadata[prefix+"write_hash"] = metadataString(projection.Metadata, "write_hash")
	metadata[prefix+"post_write_sha256"] = metadataString(projection.Metadata, "post_write_sha256")
	metadata[prefix+"post_write_verified"] = metadataBool(projection.Metadata, "post_write_verified")
	metadata[prefix+"stable_projection_validated"] = metadataBool(projection.Metadata, "stable_projection_validated")
	metadata[prefix+"protected_paths_verified"] = metadataBool(projection.Metadata, "protected_paths_verified")
	metadata[prefix+"root_contained"] = metadataBool(projection.Metadata, "root_contained")
	metadata[prefix+"apply_gate_status"] = metadataString(projection.Metadata, "apply_gate_status")
	metadata[prefix+"apply_gate_decision"] = metadataString(projection.Metadata, "apply_gate_decision")
	metadata[prefix+"apply_command_eligible"] = metadataBool(projection.Metadata, "apply_command_eligible")
	metadata[prefix+"project_write_attempted"] = metadataBool(projection.Metadata, "project_write_attempted")
	metadata[prefix+"execution_write_attempted"] = metadataBool(projection.Metadata, "execution_write_attempted")
	metadata[prefix+"engine_call_attempted"] = metadataBool(projection.Metadata, "engine_call_attempted")
}

func buildCompletionAuditSnapshotReadiness(record Record, latest CompletionAuditSnapshot, hasSnapshot bool, bundle ReleaseEvidenceBundle, currentAuditBindingValues ...completionAuditSnapshotCurrentAuditBinding) (readiness CompletionAuditSnapshotReadiness) {
	currentAudit := completionAuditSnapshotCurrentAuditBinding{}
	if len(currentAuditBindingValues) > 0 {
		currentAudit = currentAuditBindingValues[0]
	}
	currentAudit.Status = strings.TrimSpace(currentAudit.Status)
	currentAudit.Scope = strings.TrimSpace(currentAudit.Scope)
	currentAudit.Hash = strings.TrimSpace(currentAudit.Hash)
	currentAudit.EvidenceRoot = strings.TrimSpace(currentAudit.EvidenceRoot)
	if currentAudit.EvidenceRoot == "" {
		currentAudit.EvidenceRoot = completionAuditSnapshotDefaultEvidenceRoot()
	}
	readiness = CompletionAuditSnapshotReadiness{
		Real100Guardrail: CompletionAuditReal100Guardrail(),
		Project:          record,
		Status:           "blocked",
		Message:          "release candidate completion audit snapshot is not ready",
		HasSnapshot:      hasSnapshot,
		Latest:           latest,
		Items:            []ReadinessItem{},
		RequiredClass:    completionAuditSnapshotEvidenceClassReleaseCandidate,
		BundleHash:       bundle.BundleHash,
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"project_write_attempted":             false,
			"execution_write_attempted":           false,
			"release_package_created":             false,
			"publish_attempted":                   false,
			"restore_apply_attempted":             false,
			"secret_resolved":                     false,
			"remote_worker_credentials_issued":    false,
			"area_matrix_protected_paths_touched": false,
			"commands_run":                        false,
			"smoke_run_attempted":                 false,
			"worker_started":                      false,
		},
	}
	defer func() {
		readiness.Real100Guardrail = completionAuditSnapshotReadinessReal100Guardrail(readiness, currentAudit.Real100Guardrail)
	}()
	if !completionAuditSnapshotProjectMatches(record) {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:     "completion_audit_snapshot_project_mismatch",
			Status:  "blocked",
			Message: "Completion audit snapshot readiness is only defined for the target AreaMatrix project",
			Metadata: map[string]any{
				"expected_project_key": completionAuditTargetProjectKey,
				"actual_project_key":   record.Key,
			},
		})
		return readiness
	}
	if blockers := completionAuditSnapshotRealProjectIdentityBlockers(record); len(blockers) > 0 {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:     "completion_audit_snapshot_real_project_identity_missing",
			Status:  "blocked",
			Message: "Release candidate completion audit snapshot requires the real AreaMatrix project identity",
			Metadata: map[string]any{
				"expected_project_key":        completionAuditTargetProjectKey,
				"actual_project_key":          record.Key,
				"expected_project_root":       completionAuditTargetProjectRoot,
				"actual_project_root":         record.RootPath,
				"actual_project_kind":         record.Kind,
				"actual_adapter":              record.Adapter,
				"actual_workflow_profile":     record.WorkflowProfile,
				"real_project_identity_ready": false,
				"identity_blockers":           blockers,
			},
		})
		return readiness
	}
	if !hasSnapshot {
		metadata := map[string]any{
			"required_evidence_class":          completionAuditSnapshotEvidenceClassReleaseCandidate,
			"required_proof_evidence_uri_keys": completionAuditSnapshotRequiredProofEvidenceURIKeys(),
			"required_proof_event_id_keys":     completionAuditSnapshotRequiredProofEventIDKeys(),
			"required_proof_provenance_keys":   completionAuditSnapshotRequiredProofProvenanceKeys(),
			"current_proof_evidence_uris":      currentAudit.ProofEvidenceURIs,
			"current_proof_evidence_uri_map":   currentAudit.ProofEvidenceURIMap,
			"current_proof_evidence_uri_count": len(currentAudit.ProofEvidenceURIs),
			"current_proof_event_ids":          currentAudit.ProofEventIDs,
			"current_proof_event_id_count":     len(currentAudit.ProofEventIDs),
			"current_proof_provenance_map":     currentAudit.ProofProvenanceMap,
			"current_missing_proof_evidence_uri_keys": completionAuditSnapshotMissingStringBindingKeys(
				completionAuditSnapshotRequiredProofEvidenceURIKeys(),
				currentAudit.ProofEvidenceURIMap,
			),
			"current_missing_proof_event_id_keys": completionAuditSnapshotMissingInt64BindingKeys(
				completionAuditSnapshotRequiredProofEventIDKeys(),
				currentAudit.ProofEventIDs,
			),
			"current_missing_proof_provenance_keys": completionAuditSnapshotMissingStringBindingKeys(
				completionAuditSnapshotRequiredProofProvenanceKeys(),
				currentAudit.ProofProvenanceMap,
			),
			"current_mechanism_proof_evidence_uris": completionAuditSnapshotMechanismEvidenceURIValues(currentAudit.ProofEvidenceURIs),
			"current_proof_evidence_uri_blockers":   completionAuditSnapshotCurrentProofEvidenceURIMapBindingBlockers(currentAudit.ProofEvidenceURIs, currentAudit.ProofEvidenceURIMap),
			"current_proof_event_id_blockers":       completionAuditSnapshotCurrentProofEventIDMapBindingBlockers(currentAudit.ProofEventIDs),
			"current_proof_provenance_blockers":     completionAuditSnapshotCurrentProofProvenanceBindingBlockers(currentAudit.ProofProvenanceMap),
			"current_audit_status":                  currentAudit.Status,
			"current_audit_scope":                   currentAudit.Scope,
			"current_audit_hash":                    currentAudit.Hash,
			"current_bundle_hash":                   bundle.BundleHash,
			"current_bundle_status":                 bundle.Status,
			"current_bundle_mode":                   bundle.Mode,
			"current_bundle_item_count":             len(bundle.Items),
		}
		addCompletionAuditSnapshotPackageAStatusProjectionMetadata(metadata, currentAudit.PackageAStatusProjection)
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_missing",
			Status:   "blocked",
			Message:  "No completion audit snapshot has been recorded for this project",
			Metadata: metadata,
		})
		return readiness
	}
	metadata := map[string]any{
		"latest_evidence_class":      latest.EvidenceClass,
		"latest_release_candidate":   latest.ReleaseCandidateLabel,
		"latest_evidence_uri":        latest.EvidenceURI,
		"latest_audit_status":        latest.AuditStatus,
		"latest_audit_scope":         latest.AuditScope,
		"latest_audit_hash":          latest.AuditHash,
		"snapshot_audit_hash":        latest.AuditHash,
		"current_audit_status":       currentAudit.Status,
		"current_audit_scope":        currentAudit.Scope,
		"current_audit_hash":         currentAudit.Hash,
		"audit_hash_match":           currentAudit.Hash == "" || strings.TrimSpace(latest.AuditHash) == currentAudit.Hash,
		"latest_event_id":            latest.EventID,
		"latest_summary_present":     metadataString(latest.Metadata, "summary") != "",
		"latest_review_decision":     metadataString(latest.Metadata, "review_decision"),
		"latest_reviewed_by":         metadataString(latest.Metadata, "reviewed_by"),
		"latest_reviewed_at":         metadataString(latest.Metadata, "reviewed_at"),
		"review_metadata_status":     metadataString(latest.Metadata, "review_metadata_status"),
		"latest_bundle_hash":         metadataString(latest.Metadata, "release_evidence_bundle_hash"),
		"current_bundle_hash":        bundle.BundleHash,
		"current_bundle_status":      bundle.Status,
		"fixture_snapshot":           latest.EvidenceClass == completionAuditSnapshotEvidenceClassFixture,
		"release_candidate_snapshot": latest.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate,
		"project_write_attempted":    latest.ProjectWriteAttempted,
		"execution_write_attempted":  latest.ExecutionWriteAttempted,
		"release_package_created":    latest.ReleasePackageCreated,
		"publish_attempted":          latest.PublishAttempted,
		"restore_apply_attempted":    latest.RestoreApplyAttempted,
		"secret_resolved":            latest.SecretResolved,
		"commands_run":               latest.CommandsRun,
		"smoke_run_attempted":        latest.SmokeRunAttempted,
		"worker_started":             latest.WorkerStarted,
		"protected_paths_touched":    latest.AreaMatrixProtectedPathsTouched,
		"remote_worker_credentials":  latest.RemoteWorkerCredentialsIssued,
	}
	addCompletionAuditSnapshotPackageAStatusProjectionMetadata(metadata, currentAudit.PackageAStatusProjection)
	if latest.EvidenceClass != completionAuditSnapshotEvidenceClassReleaseCandidate {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_fixture_only",
			Status:   "blocked",
			Message:  "Latest completion audit snapshot is fixture evidence, not release_candidate evidence",
			Metadata: metadata,
		})
		return readiness
	}
	if latest.AuditStatus != "complete" || latest.AuditScope != "v1.0" || strings.TrimSpace(latest.AuditHash) == "" {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_audit_identity_invalid",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot does not carry a complete v1.0 audit identity",
			Metadata: metadata,
		})
		return readiness
	}
	if currentAudit.Hash != "" && strings.TrimSpace(latest.AuditHash) != currentAudit.Hash {
		metadata["audit_hash_blockers"] = []string{"snapshot_audit_hash_mismatch"}
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_audit_hash_mismatch",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot is not bound to the current completion audit hash",
			Metadata: metadata,
		})
		return readiness
	}
	unsafeFacts := completionAuditSnapshotUnsafeFacts(latest)
	if len(unsafeFacts) > 0 {
		metadata["unsafe_facts"] = unsafeFacts
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_unsafe_side_effects",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot reports unsafe side effects",
			Metadata: metadata,
		})
		return readiness
	}
	if latest.EvidenceURI == "" {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_evidence_uri_missing",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot is missing evidence URI",
			Metadata: metadata,
		})
		return readiness
	}
	summary := metadataString(latest.Metadata, "summary")
	if summary == "" {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_summary_missing",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot is missing an evidence summary",
			Metadata: metadata,
		})
		return readiness
	}
	if completionAuditSnapshotContainsFixtureMarker(latest.ReleaseCandidateLabel) ||
		completionAuditSnapshotContainsFixtureMarker(latest.EvidenceURI) ||
		completionAuditSnapshotContainsFixtureMarker(summary) {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_fixture_labeled_release_candidate",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot still contains fixture-labeled evidence",
			Metadata: metadata,
		})
		return readiness
	}
	if completionAuditSnapshotUsesMechanismEvidenceURI(latest.EvidenceURI) {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_mechanism_evidence_uri",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot points at local script/smoke mechanism evidence",
			Metadata: metadata,
		})
		return readiness
	}
	if !completionAuditSnapshotUsesReleaseCandidateEvidenceURI(latest.EvidenceURI) {
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_generic_evidence_uri",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot does not point at release-candidate evidence",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotReviewMetadataBlockers(latest.Metadata); len(blockers) > 0 {
		metadata["review_metadata_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_review_metadata_missing",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot is missing approved review metadata",
			Metadata: metadata,
		})
		return readiness
	}
	mechanismProofEvidenceURIs := completionAuditSnapshotMechanismEvidenceURIValues(
		completionAuditSnapshotEvidenceURIValues(latest.Metadata["proof_evidence_uris"]),
	)
	if len(mechanismProofEvidenceURIs) > 0 {
		metadata["mechanism_proof_evidence_uris"] = mechanismProofEvidenceURIs
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_mechanism_proof_evidence_uri",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot sealed local script/smoke proof evidence",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotProofEvidenceURIBindingBlockers(latest); len(blockers) > 0 {
		metadata["proof_evidence_uri_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_proof_evidence_uri_missing",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot does not seal the required E1-E9 proof evidence URIs",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotProofEventIDBindingBlockers(latest); len(blockers) > 0 {
		metadata["proof_event_id_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_proof_event_id_missing",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot does not seal the required E1-E9 proof event IDs",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotProofProvenanceBindingBlockers(metadataStringMap(latest.Metadata, "proof_provenance_map")); len(blockers) > 0 {
		metadata["proof_provenance_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_proof_provenance_missing",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot does not seal the required E1-E9 proof provenance",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotCurrentProofBindingBlockers(latest, currentAudit); len(blockers) > 0 {
		metadata["current_proof_binding_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_current_proof_binding_mismatch",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot proof bindings do not match the current completion audit",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotEvidenceURIFileAuditBindingBlockers(latest, currentAudit.EvidenceRoot); len(blockers) > 0 {
		metadata["evidence_uri_file_audit_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_evidence_uri_file_audit_mismatch",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot evidence URI file audit no longer matches local evidence files",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotReleaseEvidenceBundleBindingBlockers(latest, bundle); len(blockers) > 0 {
		metadata["bundle_blockers"] = blockers
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      "completion_audit_snapshot_release_evidence_bundle_mismatch",
			Status:   "blocked",
			Message:  "Latest release_candidate snapshot is not bound to the current ready release evidence bundle",
			Metadata: metadata,
		})
		return readiness
	}
	if blockers := completionAuditSnapshotPackageAStatusProjectionBlockers(currentAudit.PackageAStatusProjection); len(blockers) > 0 {
		metadata["package_a_status_projection_blockers"] = blockers
		key := "completion_audit_snapshot_package_a_status_projection_unstable"
		message := "Latest release_candidate snapshot is not bound to a stable Package A status projection apply"
		if containsCompletionAuditString(blockers, "completion_audit_snapshot_package_a_not_applied") {
			key = "completion_audit_snapshot_package_a_not_applied"
			message = "Latest release_candidate snapshot does not prove Package A status projection apply"
		} else if containsCompletionAuditString(blockers, "package_a_status_projection_apply_provenance_missing") {
			key = "completion_audit_snapshot_package_a_apply_provenance_missing"
			message = "Latest release_candidate snapshot sees a stable Package A status projection file, but apply provenance is missing"
		}
		readiness.Items = append(readiness.Items, ReadinessItem{
			Key:      key,
			Status:   "blocked",
			Message:  message,
			Metadata: metadata,
		})
		return readiness
	}
	readiness.Status = "ready"
	readiness.Message = "release candidate completion audit snapshot is present"
	readiness.Items = append(readiness.Items, ReadinessItem{
		Key:      "completion_audit_snapshot_release_candidate_present",
		Status:   "ready",
		Message:  "Latest completion audit snapshot is release_candidate evidence",
		Metadata: metadata,
	})
	return readiness
}

func completionAuditSnapshotReadinessReal100Guardrail(readiness CompletionAuditSnapshotReadiness, current Real100Guardrail) Real100Guardrail {
	guardrail := NormalizeReal100Guardrail(current, CompletionAuditReal100Guardrail())
	closure := CompletionAuditSnapshotReadinessClosure(readiness)
	blockers := append([]string{}, guardrail.Real100Blockers...)
	if completionAuditSnapshotReadinessPackageAReady(readiness) {
		blockers = removeString(blockers, "package_a_status_projection_apply_provenance_missing")
		blockers = removeString(blockers, "package_a_status_projection_not_applied")
	}
	switch closure.PackageAStatusProjectionStatus {
	case "missing":
		blockers = append(blockers, "package_a_status_projection_not_applied")
	case "blocked":
		blockers = append(blockers, "package_a_status_projection_apply_provenance_missing")
	}
	if !closure.ReadyForReleaseCandidateClosure {
		blockers = append(blockers, "release_candidate_snapshot_not_ready")
	}
	guardrail.Real100Blockers = real100OrderCompletionBlockers(uniqueStrings(blockers))
	return guardrail
}

func completionAuditSnapshotReadinessPackageAReady(readiness CompletionAuditSnapshotReadiness) bool {
	for _, item := range readiness.Items {
		if metadataBool(item.Metadata, "package_a_status_projection_ready") {
			return true
		}
	}
	return false
}

func CompletionAuditSnapshotReadinessGaps(readiness CompletionAuditSnapshotReadiness) []CompletionAuditSnapshotGap {
	gaps := []CompletionAuditSnapshotGap{}
	for _, item := range readiness.Items {
		if item.Status == "ready" {
			continue
		}
		gap := CompletionAuditSnapshotGap{
			Key:      item.Key,
			Category: completionAuditSnapshotGapCategory(item.Key),
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		}
		gap.IdentityBlockers = metadataStringSlice(item.Metadata, "identity_blockers")
		gap.MissingProofEvidenceURIKeys = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_missing_proof_evidence_uri_keys"),
			metadataStringSlice(item.Metadata, "missing_proof_evidence_uri_keys"),
			completionAuditSnapshotMissingKeysFromBlockers(
				metadataStringSlice(item.Metadata, "proof_evidence_uri_blockers"),
				"snapshot_proof_evidence_uri_missing:",
				"current_proof_evidence_uri_missing:",
			),
		)
		gap.MissingProofEventIDKeys = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_missing_proof_event_id_keys"),
			metadataStringSlice(item.Metadata, "missing_proof_event_id_keys"),
			completionAuditSnapshotMissingKeysFromBlockers(
				metadataStringSlice(item.Metadata, "proof_event_id_blockers"),
				"snapshot_proof_event_id_missing:",
				"current_proof_event_id_missing:",
			),
		)
		gap.MissingProofProvenanceKeys = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_missing_proof_provenance_keys"),
			metadataStringSlice(item.Metadata, "missing_proof_provenance_keys"),
			completionAuditSnapshotMissingKeysFromBlockers(
				metadataStringSlice(item.Metadata, "proof_provenance_blockers"),
				"snapshot_proof_provenance_missing:",
				"current_proof_provenance_missing:",
			),
		)
		gap.MechanismProofEvidenceURIs = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_mechanism_proof_evidence_uris"),
			metadataStringSlice(item.Metadata, "mechanism_proof_evidence_uris"),
		)
		gap.ProofEvidenceURIBlockers = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_proof_evidence_uri_blockers"),
			metadataStringSlice(item.Metadata, "proof_evidence_uri_blockers"),
		)
		gap.ProofEventIDBlockers = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_proof_event_id_blockers"),
			metadataStringSlice(item.Metadata, "proof_event_id_blockers"),
		)
		gap.ProofProvenanceBlockers = firstNonEmptyStringSlice(
			metadataStringSlice(item.Metadata, "current_proof_provenance_blockers"),
			metadataStringSlice(item.Metadata, "proof_provenance_blockers"),
		)
		gap.CurrentProofBindingBlockers = metadataStringSlice(item.Metadata, "current_proof_binding_blockers")
		gap.BundleBlockers = metadataStringSlice(item.Metadata, "bundle_blockers")
		gap.EvidenceURIFileAuditBlockers = metadataStringSlice(item.Metadata, "evidence_uri_file_audit_blockers")
		gap.PackageAStatusProjectionBlockers = metadataStringSlice(item.Metadata, "package_a_status_projection_blockers")
		gap.ReviewMetadataBlockers = metadataStringSlice(item.Metadata, "review_metadata_blockers")
		gap.UnsafeFacts = metadataStringSlice(item.Metadata, "unsafe_facts")
		gap.Blockers = uniqueStrings(append([]string{item.Key}, completionAuditSnapshotGapBlockers(gap)...))
		if len(gap.Blockers) == 0 {
			gap.Blockers = []string{item.Key}
		}
		gaps = append(gaps, gap)
	}
	return gaps
}

func CompletionAuditSnapshotReadinessClosure(readiness CompletionAuditSnapshotReadiness) CompletionAuditSnapshotClosure {
	gaps := CompletionAuditSnapshotReadinessGaps(readiness)
	closure := CompletionAuditSnapshotClosure{
		Status:                          readiness.Status,
		Ready:                           readiness.Status == "ready" && len(gaps) == 0,
		ReadyForReleaseCandidateClosure: readiness.Status == "ready" && len(gaps) == 0,
		RequiredClass:                   readiness.RequiredClass,
		RequiredEvidenceClass:           readiness.RequiredClass,
		HasSnapshot:                     readiness.HasSnapshot,
		GapCount:                        len(gaps),
		GapKeys:                         []string{},
		Blockers:                        []string{},
		IdentityStatus:                  "pass",
		SnapshotStatus:                  "release_candidate_present",
		AuditIdentityStatus:             "pass",
		AuditHashStatus:                 "pass",
		SnapshotEvidenceStatus:          "pass",
		SafetyStatus:                    "pass",
		ProofEvidenceURIStatus:          "pass",
		ProofEventIDStatus:              "pass",
		ProofProvenanceStatus:           "pass",
		CurrentProofBindingStatus:       "pass",
		EvidenceURIFileAuditStatus:      "pass",
		ReleaseEvidenceBundleStatus:     "pass",
		PackageAStatusProjectionStatus:  "pass",
		ReviewMetadataStatus:            "pass",
		BundleHash:                      readiness.BundleHash,
		LatestEvidenceClass:             readiness.Latest.EvidenceClass,
		LatestReleaseCandidate:          readiness.Latest.ReleaseCandidateLabel,
		LatestEvidenceURI:               readiness.Latest.EvidenceURI,
		LatestReviewDecision:            metadataString(readiness.Latest.Metadata, "review_decision"),
		LatestReviewedBy:                metadataString(readiness.Latest.Metadata, "reviewed_by"),
		LatestReviewedAt:                metadataString(readiness.Latest.Metadata, "reviewed_at"),
		LatestEventID:                   readiness.Latest.EventID,
		SnapshotAuditHash:               readiness.Latest.AuditHash,
		LatestBundleHash:                metadataString(readiness.Latest.Metadata, "release_evidence_bundle_hash"),
	}
	if !readiness.HasSnapshot {
		closure.SnapshotStatus = "missing"
	}
	if readiness.Latest.EvidenceClass == completionAuditSnapshotEvidenceClassFixture {
		closure.SnapshotStatus = "fixture_only"
	}
	if readiness.Latest.EvidenceClass == completionAuditSnapshotEvidenceClassReleaseCandidate && readiness.Status != "ready" {
		closure.SnapshotStatus = "blocked"
	}
	if len(gaps) > 0 {
		closure.Ready = false
	}
	for _, gap := range gaps {
		closure.GapKeys = append(closure.GapKeys, gap.Key)
		closure.Blockers = append(closure.Blockers, gap.Blockers...)
		closure.MissingProofEvidenceURIKeys = append(closure.MissingProofEvidenceURIKeys, gap.MissingProofEvidenceURIKeys...)
		closure.MissingProofEventIDKeys = append(closure.MissingProofEventIDKeys, gap.MissingProofEventIDKeys...)
		closure.MissingProofProvenanceKeys = append(closure.MissingProofProvenanceKeys, gap.MissingProofProvenanceKeys...)
		closure.MechanismProofEvidenceURIs = append(closure.MechanismProofEvidenceURIs, gap.MechanismProofEvidenceURIs...)
		closure.ProofEvidenceURIBlockers = append(closure.ProofEvidenceURIBlockers, gap.ProofEvidenceURIBlockers...)
		closure.ProofEventIDBlockers = append(closure.ProofEventIDBlockers, gap.ProofEventIDBlockers...)
		closure.ProofProvenanceBlockers = append(closure.ProofProvenanceBlockers, gap.ProofProvenanceBlockers...)
		closure.CurrentProofBindingBlockers = append(closure.CurrentProofBindingBlockers, gap.CurrentProofBindingBlockers...)
		closure.BundleBlockers = append(closure.BundleBlockers, gap.BundleBlockers...)
		closure.EvidenceURIFileAuditBlockers = append(closure.EvidenceURIFileAuditBlockers, gap.EvidenceURIFileAuditBlockers...)
		closure.PackageAStatusProjectionBlockers = append(closure.PackageAStatusProjectionBlockers, gap.PackageAStatusProjectionBlockers...)
		closure.ReviewMetadataBlockers = append(closure.ReviewMetadataBlockers, gap.ReviewMetadataBlockers...)
		closure.UnsafeFacts = append(closure.UnsafeFacts, gap.UnsafeFacts...)
		closure.mergeMetadata(gap.Metadata)
		switch gap.Key {
		case "completion_audit_snapshot_project_mismatch",
			"completion_audit_snapshot_real_project_identity_missing":
			closure.IdentityStatus = "blocked"
		case "completion_audit_snapshot_missing":
			closure.SnapshotStatus = "missing"
		case "completion_audit_snapshot_fixture_only":
			closure.SnapshotStatus = "fixture_only"
		case "completion_audit_snapshot_audit_identity_invalid":
			closure.AuditIdentityStatus = "blocked"
		case "completion_audit_snapshot_audit_hash_mismatch":
			closure.AuditHashStatus = "mismatch"
		case "completion_audit_snapshot_unsafe_side_effects":
			closure.SafetyStatus = "blocked"
		case "completion_audit_snapshot_evidence_uri_missing",
			"completion_audit_snapshot_summary_missing",
			"completion_audit_snapshot_fixture_labeled_release_candidate",
			"completion_audit_snapshot_mechanism_evidence_uri",
			"completion_audit_snapshot_generic_evidence_uri":
			closure.SnapshotEvidenceStatus = "blocked"
		case "completion_audit_snapshot_mechanism_proof_evidence_uri",
			"completion_audit_snapshot_proof_evidence_uri_missing":
			closure.ProofEvidenceURIStatus = "blocked"
		case "completion_audit_snapshot_proof_event_id_missing":
			closure.ProofEventIDStatus = "blocked"
		case "completion_audit_snapshot_proof_provenance_missing":
			closure.ProofProvenanceStatus = "blocked"
		case "completion_audit_snapshot_current_proof_binding_mismatch":
			closure.CurrentProofBindingStatus = "mismatch"
		case "completion_audit_snapshot_evidence_uri_file_audit_mismatch":
			closure.EvidenceURIFileAuditStatus = "mismatch"
		case "completion_audit_snapshot_release_evidence_bundle_mismatch":
			closure.ReleaseEvidenceBundleStatus = "mismatch"
		case "completion_audit_snapshot_package_a_not_applied":
			closure.PackageAStatusProjectionStatus = "missing"
		case "completion_audit_snapshot_package_a_apply_provenance_missing",
			"completion_audit_snapshot_package_a_status_projection_unstable":
			closure.PackageAStatusProjectionStatus = "blocked"
		case "completion_audit_snapshot_review_metadata_missing":
			closure.ReviewMetadataStatus = "blocked"
		}
	}
	if len(closure.MissingProofEvidenceURIKeys) > 0 {
		closure.ProofEvidenceURIStatus = "missing"
	}
	if len(closure.MissingProofEventIDKeys) > 0 {
		closure.ProofEventIDStatus = "missing"
	}
	if len(closure.MissingProofProvenanceKeys) > 0 {
		closure.ProofProvenanceStatus = "missing"
	}
	if len(closure.PackageAStatusProjectionBlockers) > 0 && closure.PackageAStatusProjectionStatus == "pass" {
		closure.PackageAStatusProjectionStatus = "blocked"
		if containsCompletionAuditString(closure.PackageAStatusProjectionBlockers, "completion_audit_snapshot_package_a_not_applied") {
			closure.PackageAStatusProjectionStatus = "missing"
		}
	}
	closure.Blockers = uniqueStrings(closure.Blockers)
	closure.GapKeys = uniqueStrings(closure.GapKeys)
	closure.MissingProofEvidenceURIKeys = uniqueStrings(closure.MissingProofEvidenceURIKeys)
	closure.MissingProofEventIDKeys = uniqueStrings(closure.MissingProofEventIDKeys)
	closure.MissingProofProvenanceKeys = uniqueStrings(closure.MissingProofProvenanceKeys)
	closure.MechanismProofEvidenceURIs = uniqueStrings(closure.MechanismProofEvidenceURIs)
	closure.ProofEvidenceURIBlockers = uniqueStrings(closure.ProofEvidenceURIBlockers)
	closure.ProofEventIDBlockers = uniqueStrings(closure.ProofEventIDBlockers)
	closure.ProofProvenanceBlockers = uniqueStrings(closure.ProofProvenanceBlockers)
	closure.CurrentProofBindingBlockers = uniqueStrings(closure.CurrentProofBindingBlockers)
	closure.BundleBlockers = uniqueStrings(closure.BundleBlockers)
	closure.EvidenceURIFileAuditBlockers = uniqueStrings(closure.EvidenceURIFileAuditBlockers)
	closure.PackageAStatusProjectionBlockers = uniqueStrings(closure.PackageAStatusProjectionBlockers)
	closure.ReviewMetadataBlockers = uniqueStrings(closure.ReviewMetadataBlockers)
	closure.UnsafeFacts = uniqueStrings(closure.UnsafeFacts)
	closure.populateGates()
	return closure
}

func (closure *CompletionAuditSnapshotClosure) populateGates() {
	closure.ReadyForReleaseCandidateClosure = closure.Ready
	closure.ProjectIdentity = completionAuditSnapshotClosureGate(closure.IdentityStatus, nil, map[string]any{
		"required_project_key":  completionAuditTargetProjectKey,
		"required_project_root": completionAuditTargetProjectRoot,
	})
	closure.Snapshot = completionAuditSnapshotClosureGate(closure.SnapshotStatus, nil, map[string]any{
		"has_snapshot":             closure.HasSnapshot,
		"required_evidence_class":  closure.RequiredEvidenceClass,
		"latest_evidence_class":    closure.LatestEvidenceClass,
		"latest_release_candidate": closure.LatestReleaseCandidate,
		"latest_evidence_uri":      closure.LatestEvidenceURI,
		"latest_event_id":          closure.LatestEventID,
	})
	closure.AuditBinding = completionAuditSnapshotClosureGate(completionAuditSnapshotClosureCombinedStatus(closure.AuditIdentityStatus, closure.AuditHashStatus), nil, map[string]any{
		"audit_identity_status": closure.AuditIdentityStatus,
		"audit_hash_status":     closure.AuditHashStatus,
		"snapshot_audit_hash":   closure.SnapshotAuditHash,
		"current_audit_hash":    closure.CurrentAuditHash,
		"current_audit_status":  closure.CurrentAuditStatus,
		"current_audit_scope":   closure.CurrentAuditScope,
	})
	closure.SnapshotEvidence = completionAuditSnapshotClosureGate(closure.SnapshotEvidenceStatus, nil, map[string]any{
		"latest_evidence_uri": closure.LatestEvidenceURI,
	})
	closure.ProofEvidenceURIs = completionAuditSnapshotClosureGate(closure.ProofEvidenceURIStatus, append(closure.ProofEvidenceURIBlockers, closure.MechanismProofEvidenceURIs...), map[string]any{
		"missing_keys":       closure.MissingProofEvidenceURIKeys,
		"mechanism_uris":     closure.MechanismProofEvidenceURIs,
		"required_key_count": len(completionAuditSnapshotRequiredProofEvidenceURIKeys()),
	})
	closure.ProofEventIDs = completionAuditSnapshotClosureGate(closure.ProofEventIDStatus, closure.ProofEventIDBlockers, map[string]any{
		"missing_keys":       closure.MissingProofEventIDKeys,
		"required_key_count": len(completionAuditSnapshotRequiredProofEventIDKeys()),
	})
	closure.ProofProvenance = completionAuditSnapshotClosureGate(closure.ProofProvenanceStatus, closure.ProofProvenanceBlockers, map[string]any{
		"missing_keys":       closure.MissingProofProvenanceKeys,
		"required_key_count": len(completionAuditSnapshotRequiredProofProvenanceKeys()),
	})
	closure.CurrentProofBinding = completionAuditSnapshotClosureGate(closure.CurrentProofBindingStatus, closure.CurrentProofBindingBlockers, nil)
	closure.ReleaseEvidenceBundle = completionAuditSnapshotClosureGate(closure.ReleaseEvidenceBundleStatus, closure.BundleBlockers, map[string]any{
		"bundle_hash":           closure.BundleHash,
		"latest_bundle_hash":    closure.LatestBundleHash,
		"current_bundle_hash":   closure.CurrentBundleHash,
		"current_bundle_status": closure.CurrentBundleStatus,
		"current_bundle_mode":   closure.CurrentBundleMode,
	})
	closure.EvidenceFileAudit = completionAuditSnapshotClosureGate(closure.EvidenceURIFileAuditStatus, closure.EvidenceURIFileAuditBlockers, nil)
	closure.PackageAStatusProjection = completionAuditSnapshotClosureGate(closure.PackageAStatusProjectionStatus, closure.PackageAStatusProjectionBlockers, nil)
	closure.ReviewMetadata = completionAuditSnapshotClosureGate(closure.ReviewMetadataStatus, closure.ReviewMetadataBlockers, map[string]any{
		"latest_review_decision": closure.LatestReviewDecision,
		"latest_reviewed_by":     closure.LatestReviewedBy,
		"latest_reviewed_at":     closure.LatestReviewedAt,
	})
	closure.Safety = completionAuditSnapshotClosureGate(closure.SafetyStatus, closure.UnsafeFacts, map[string]any{
		"unsafe_facts": closure.UnsafeFacts,
	})
	closure.ProjectIdentity.Blockers = uniqueStrings(append(closure.ProjectIdentity.Blockers, filterBlockersByPrefixes(closure.Blockers, "project_", "adapter_", "workflow_profile_", "default_branch_", "project_kind_", "project_root_")...))
	closure.Snapshot.Blockers = uniqueStrings(append(closure.Snapshot.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_missing", "completion_audit_snapshot_fixture_only")...))
	closure.AuditBinding.Blockers = uniqueStrings(append(closure.AuditBinding.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_audit_identity_invalid", "completion_audit_snapshot_audit_hash_mismatch")...))
	closure.SnapshotEvidence.Blockers = uniqueStrings(append(closure.SnapshotEvidence.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_evidence_uri_missing", "completion_audit_snapshot_summary_missing", "completion_audit_snapshot_fixture_labeled_release_candidate", "completion_audit_snapshot_mechanism_evidence_uri", "completion_audit_snapshot_generic_evidence_uri")...))
	closure.CurrentProofBinding.Blockers = uniqueStrings(append(closure.CurrentProofBinding.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_current_proof_binding_mismatch")...))
	closure.EvidenceFileAudit.Blockers = uniqueStrings(append(closure.EvidenceFileAudit.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_evidence_uri_file_audit_mismatch")...))
	closure.ReleaseEvidenceBundle.Blockers = uniqueStrings(append(closure.ReleaseEvidenceBundle.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_release_evidence_bundle_mismatch")...))
	closure.PackageAStatusProjection.Blockers = uniqueStrings(append(closure.PackageAStatusProjection.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_package_a_not_applied", "completion_audit_snapshot_package_a_apply_provenance_missing", "completion_audit_snapshot_package_a_status_projection_unstable")...))
	closure.ReviewMetadata.Blockers = uniqueStrings(append(closure.ReviewMetadata.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_review_metadata_missing")...))
	closure.Safety.Blockers = uniqueStrings(append(closure.Safety.Blockers, filterBlockersByPrefixes(closure.GapKeys, "completion_audit_snapshot_unsafe_side_effects")...))
}

func completionAuditSnapshotClosureGate(status string, blockers []string, metadata map[string]any) CompletionAuditSnapshotClosureGate {
	return CompletionAuditSnapshotClosureGate{
		Status:   status,
		Ready:    status == "pass" || status == "release_candidate_present",
		Blockers: uniqueStrings(blockers),
		Metadata: metadata,
	}
}

func completionAuditSnapshotClosureCombinedStatus(values ...string) string {
	for _, value := range values {
		if value != "" && value != "pass" {
			return value
		}
	}
	return "pass"
}

func filterBlockersByPrefixes(values []string, prefixes ...string) []string {
	out := []string{}
	for _, value := range values {
		for _, prefix := range prefixes {
			if value == prefix || strings.HasPrefix(value, prefix) {
				out = append(out, value)
				break
			}
		}
	}
	return uniqueStrings(out)
}

func (closure *CompletionAuditSnapshotClosure) mergeMetadata(metadata map[string]any) {
	if closure.CurrentAuditHash == "" {
		closure.CurrentAuditHash = metadataString(metadata, "current_audit_hash")
	}
	if closure.CurrentAuditStatus == "" {
		closure.CurrentAuditStatus = metadataString(metadata, "current_audit_status")
	}
	if closure.CurrentAuditScope == "" {
		closure.CurrentAuditScope = metadataString(metadata, "current_audit_scope")
	}
	if closure.LatestBundleHash == "" {
		closure.LatestBundleHash = metadataString(metadata, "latest_bundle_hash")
	}
	if closure.CurrentBundleHash == "" {
		closure.CurrentBundleHash = metadataString(metadata, "current_bundle_hash")
	}
	if closure.CurrentBundleStatus == "" {
		closure.CurrentBundleStatus = metadataString(metadata, "current_bundle_status")
	}
	if closure.CurrentBundleMode == "" {
		closure.CurrentBundleMode = metadataString(metadata, "current_bundle_mode")
	}
}

func completionAuditSnapshotGapCategory(key string) string {
	switch key {
	case "completion_audit_snapshot_project_mismatch",
		"completion_audit_snapshot_real_project_identity_missing":
		return "identity"
	case "completion_audit_snapshot_missing",
		"completion_audit_snapshot_fixture_only",
		"completion_audit_snapshot_audit_identity_invalid",
		"completion_audit_snapshot_audit_hash_mismatch":
		return "snapshot"
	case "completion_audit_snapshot_unsafe_side_effects":
		return "safety"
	case "completion_audit_snapshot_evidence_uri_missing",
		"completion_audit_snapshot_summary_missing",
		"completion_audit_snapshot_fixture_labeled_release_candidate",
		"completion_audit_snapshot_mechanism_evidence_uri",
		"completion_audit_snapshot_generic_evidence_uri":
		return "snapshot_evidence"
	case "completion_audit_snapshot_mechanism_proof_evidence_uri",
		"completion_audit_snapshot_proof_evidence_uri_missing":
		return "proof_evidence_uri"
	case "completion_audit_snapshot_proof_event_id_missing":
		return "proof_event_id"
	case "completion_audit_snapshot_proof_provenance_missing":
		return "proof_provenance"
	case "completion_audit_snapshot_current_proof_binding_mismatch":
		return "current_binding"
	case "completion_audit_snapshot_evidence_uri_file_audit_mismatch":
		return "evidence_file_audit"
	case "completion_audit_snapshot_release_evidence_bundle_mismatch":
		return "release_evidence_bundle"
	case "completion_audit_snapshot_package_a_not_applied",
		"completion_audit_snapshot_package_a_apply_provenance_missing",
		"completion_audit_snapshot_package_a_status_projection_unstable":
		return "package_a_status_projection"
	case "completion_audit_snapshot_review_metadata_missing":
		return "review_metadata"
	default:
		return "readiness"
	}
}

func completionAuditSnapshotGapBlockers(gap CompletionAuditSnapshotGap) []string {
	blockers := []string{}
	blockers = append(blockers, gap.IdentityBlockers...)
	blockers = append(blockers, gap.MissingProofEvidenceURIKeys...)
	blockers = append(blockers, gap.MissingProofEventIDKeys...)
	blockers = append(blockers, gap.MissingProofProvenanceKeys...)
	blockers = append(blockers, gap.MechanismProofEvidenceURIs...)
	blockers = append(blockers, gap.ProofEvidenceURIBlockers...)
	blockers = append(blockers, gap.ProofEventIDBlockers...)
	blockers = append(blockers, gap.ProofProvenanceBlockers...)
	blockers = append(blockers, gap.CurrentProofBindingBlockers...)
	blockers = append(blockers, gap.BundleBlockers...)
	blockers = append(blockers, gap.EvidenceURIFileAuditBlockers...)
	blockers = append(blockers, gap.PackageAStatusProjectionBlockers...)
	blockers = append(blockers, gap.ReviewMetadataBlockers...)
	blockers = append(blockers, gap.UnsafeFacts...)
	return uniqueStrings(blockers)
}

func firstNonEmptyStringSlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func completionAuditSnapshotMissingKeysFromBlockers(blockers []string, prefixes ...string) []string {
	missing := []string{}
	for _, blocker := range blockers {
		for _, prefix := range prefixes {
			if strings.HasPrefix(blocker, prefix) {
				missing = append(missing, strings.TrimPrefix(blocker, prefix))
				break
			}
		}
	}
	return uniqueStrings(missing)
}

func completionAuditSnapshotUnsafeFacts(snapshot CompletionAuditSnapshot) []string {
	unsafe := []string{}
	if snapshot.ProjectWriteAttempted {
		unsafe = append(unsafe, "project_write_attempted")
	}
	if snapshot.ExecutionWriteAttempted {
		unsafe = append(unsafe, "execution_write_attempted")
	}
	if snapshot.ReleasePackageCreated {
		unsafe = append(unsafe, "release_package_created")
	}
	if snapshot.PublishAttempted {
		unsafe = append(unsafe, "publish_attempted")
	}
	if snapshot.RestoreApplyAttempted {
		unsafe = append(unsafe, "restore_apply_attempted")
	}
	if snapshot.SecretResolved {
		unsafe = append(unsafe, "secret_resolved")
	}
	if snapshot.RemoteWorkerCredentialsIssued {
		unsafe = append(unsafe, "remote_worker_credentials_issued")
	}
	if snapshot.AreaMatrixProtectedPathsTouched {
		unsafe = append(unsafe, "area_matrix_protected_paths_touched")
	}
	if snapshot.CommandsRun {
		unsafe = append(unsafe, "commands_run")
	}
	if snapshot.SmokeRunAttempted {
		unsafe = append(unsafe, "smoke_run_attempted")
	}
	if snapshot.WorkerStarted {
		unsafe = append(unsafe, "worker_started")
	}
	return unsafe
}

func completionAuditSnapshotFromMetadata(record Record, metadata map[string]any) CompletionAuditSnapshot {
	nested := map[string]any{}
	if raw, ok := metadata["metadata"].(map[string]any); ok {
		nested = raw
	}
	return CompletionAuditSnapshot{
		Real100Guardrail:                CompletionAuditReal100Guardrail(),
		Project:                         record,
		Status:                          metadataString(metadata, "status"),
		Decision:                        metadataString(metadata, "decision"),
		Message:                         metadataString(metadata, "message"),
		AuditStatus:                     metadataString(metadata, "audit_status"),
		AuditScope:                      metadataString(metadata, "audit_scope"),
		AuditHash:                       metadataString(metadata, "audit_hash"),
		ReleaseCandidateLabel:           metadataString(metadata, "release_candidate_label"),
		EvidenceClass:                   metadataString(metadata, "evidence_class"),
		EvidenceURI:                     metadataString(metadata, "evidence_uri"),
		ProofEventIDs:                   metadataInt64Map(metadata, "proof_event_ids"),
		EventID:                         metadataInt64(metadata, "event_id"),
		AuditEventID:                    metadataInt64(metadata, "audit_event_id"),
		IdempotencyKey:                  metadataString(metadata, "idempotency_key"),
		ProjectWriteAttempted:           metadataBool(metadata, "project_write_attempted"),
		ExecutionWriteAttempted:         metadataBool(metadata, "execution_write_attempted"),
		ReleasePackageCreated:           metadataBool(metadata, "release_package_created"),
		PublishAttempted:                metadataBool(metadata, "publish_attempted"),
		RestoreApplyAttempted:           metadataBool(metadata, "restore_apply_attempted"),
		SecretResolved:                  metadataBool(metadata, "secret_resolved"),
		RemoteWorkerCredentialsIssued:   metadataBool(metadata, "remote_worker_credentials_issued"),
		AreaMatrixProtectedPathsTouched: metadataBool(metadata, "area_matrix_protected_paths_touched"),
		CommandsRun:                     metadataBool(metadata, "commands_run"),
		SmokeRunAttempted:               metadataBool(metadata, "smoke_run_attempted"),
		WorkerStarted:                   metadataBool(metadata, "worker_started"),
		Metadata:                        nested,
	}
}

func metadataInt64Map(metadata map[string]any, key string) map[string]int64 {
	value, ok := metadata[key]
	if !ok || value == nil {
		return map[string]int64{}
	}
	switch typed := value.(type) {
	case map[string]int64:
		out := map[string]int64{}
		for key, value := range typed {
			out[key] = value
		}
		return out
	case map[string]any:
		out := map[string]int64{}
		for key, value := range typed {
			switch number := value.(type) {
			case float64:
				out[key] = int64(number)
			case int64:
				out[key] = number
			case int:
				out[key] = int64(number)
			}
		}
		return out
	default:
		return map[string]int64{}
	}
}

func metadataStringMap(metadata map[string]any, key string) map[string]string {
	value, ok := metadata[key]
	if !ok || value == nil {
		return map[string]string{}
	}
	switch typed := value.(type) {
	case map[string]string:
		out := map[string]string{}
		for key, value := range typed {
			out[key] = value
		}
		return out
	case map[string]any:
		out := map[string]string{}
		for key, value := range typed {
			if text, ok := value.(string); ok {
				out[key] = text
			}
		}
		return out
	default:
		return map[string]string{}
	}
}
