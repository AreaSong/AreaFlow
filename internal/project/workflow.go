package project

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/areasong/areaflow/internal/artifact"
	workflowprofile "github.com/areasong/areaflow/internal/workflow"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidWorkflowVersionLabel = errors.New("invalid workflow version label")
	ErrWorkflowVersionExists       = errors.New("workflow version already exists")
	ErrWorkflowVersionNotFound     = errors.New("workflow version not found")
	ErrWorkflowVersionNotAuthored  = errors.New("workflow version is not authored by AreaFlow")
	ErrIdempotencyConflict         = errors.New("idempotency key reused with a different request")
	ErrUnsupportedWorkflowGate     = errors.New("unsupported workflow gate")
	ErrInvalidApprovalDecision     = errors.New("invalid approval decision")
	ErrApprovalPreviewNotReady     = errors.New("approval transition preview is not ready")
)

var workflowVersionLabelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
var workflowGateDefaultKeySequence atomic.Int64
var transitionPreviewDefaultKeySequence atomic.Int64

var workflowGateSpecs = map[string]workflowGateSpec{
	"discussion_gate": {
		GateName:    "discussion_gate",
		Phase:       "v0.3c",
		ScopeType:   "workflow_version",
		TargetStage: "discussion",
		TargetItem:  "discussion_package",
		RequiredItems: []workflowGateRequiredItem{
			{Stage: "discussion", ItemType: "discussion_package", Name: "discussion package"},
			{Stage: "middle_layer", ItemType: "middle_layer_ledger", Name: "middle_layer ledger"},
			{Stage: "changes", ItemType: "change_ledger", Name: "changes ledger"},
		},
		PlaceholderFailure: "discussion artifact is placeholder-only; Exact Docs, open questions, and risk boundaries are not proven yet",
	},
	"plan_doctor": {
		GateName:    "plan_doctor",
		Phase:       "v0.3c",
		ScopeType:   "workflow_version",
		TargetStage: "plans",
		TargetItem:  "plan",
		RequiredItems: []workflowGateRequiredItem{
			{Stage: "middle_layer", ItemType: "middle_layer_ledger", Name: "middle_layer ledger"},
			{Stage: "changes", ItemType: "change_ledger", Name: "changes ledger"},
			{Stage: "plans", ItemType: "plan", Name: "plan draft"},
		},
		PlaceholderFailure: "plan artifact is placeholder-only; dependencies, ordered steps, validation, and rollback wording are not proven yet",
	},
	"draft_doctor": {
		GateName:    "draft_doctor",
		Phase:       "v0.3c",
		ScopeType:   "workflow_version",
		TargetStage: "drafts",
		TargetItem:  "draft_manifest",
		RequiredItems: []workflowGateRequiredItem{
			{Stage: "plans", ItemType: "plan", Name: "plan draft"},
			{Stage: "drafts", ItemType: "draft_manifest", Name: "draft manifest"},
			{Stage: "drafts", ItemType: "draft_copy", Name: "copy-ready draft"},
			{Stage: "drafts", ItemType: "draft_verify", Name: "verify-ready draft"},
		},
		PlaceholderFailure: "draft artifacts are placeholder-only; copy-ready, verify-ready, and manifest boundaries are not proven yet",
	},
	"queue_doctor": {
		GateName:    "queue_doctor",
		Phase:       "v0.3c",
		ScopeType:   "workflow_version",
		TargetStage: "queue",
		TargetItem:  "queue_candidate",
		RequiredItems: []workflowGateRequiredItem{
			{Stage: "drafts", ItemType: "draft_manifest", Name: "draft manifest"},
			{Stage: "drafts", ItemType: "draft_copy", Name: "copy-ready draft"},
			{Stage: "drafts", ItemType: "draft_verify", Name: "verify-ready draft"},
			{Stage: "queue", ItemType: "queue_candidate", Name: "queue candidate"},
		},
		PlaceholderFailure: "queue artifact is placeholder-only; labels, dependencies, readiness, and ordering are not proven yet",
	},
	"promotion_preview": {
		GateName:    "promotion_preview",
		Phase:       "v0.3c",
		ScopeType:   "workflow_version",
		TargetStage: "promotion_preview",
		TargetItem:  "promotion_preview",
		RequiredItems: []workflowGateRequiredItem{
			{Stage: "queue", ItemType: "queue_candidate", Name: "queue candidate"},
			{Stage: "promotion_preview", ItemType: "promotion_preview", Name: "promotion preview"},
		},
		PlaceholderFailure: "promotion preview artifact is placeholder-only; live mapping, collision checks, and scope checks are not proven yet",
	},
	"approval_gate": {
		GateName:    "approval_gate",
		Phase:       "v0.4",
		ScopeType:   "workflow_version",
		TargetStage: "approval",
		TargetItem:  "approval_record",
	},
	"live_mapping_gate": {
		GateName:    "live_mapping_gate",
		Phase:       "v0.4",
		ScopeType:   "workflow_version",
		TargetStage: "approval",
		TargetItem:  "live_mapping",
	},
	"cutover_readiness_gate": {
		GateName:    "cutover_readiness_gate",
		Phase:       "v0.4c",
		ScopeType:   "workflow_version",
		TargetStage: "approval",
		TargetItem:  "cutover_readiness",
	},
	"profile_binding_drift": {
		GateName:    "profile_binding_drift",
		Phase:       "v0.4",
		ScopeType:   "workflow_version",
		TargetStage: "version_init",
		TargetItem:  "profile_binding",
	},
}

var authoredStageSkeleton = []stageSkeletonSpec{
	{
		Stage:        "discussion",
		ItemType:     "discussion_package",
		ArtifactType: "discussion",
		FileName:     "discussion.md",
		Title:        "Discussion package",
		Status:       "draft",
	},
	{
		Stage:        "middle_layer",
		ItemType:     "middle_layer_ledger",
		ArtifactType: "middle_layer",
		FileName:     "middle-layer.yaml",
		Title:        "Middle-layer ledger",
		Status:       "blocked",
	},
	{
		Stage:        "changes",
		ItemType:     "change_ledger",
		ArtifactType: "change",
		FileName:     "changes.yaml",
		Title:        "Change ledger",
		Status:       "blocked",
	},
	{
		Stage:        "plans",
		ItemType:     "plan",
		ArtifactType: "plan",
		FileName:     "plan.md",
		Title:        "Plan draft",
		Status:       "blocked",
	},
	{
		Stage:        "drafts",
		ItemType:     "draft_manifest",
		ArtifactType: "manifest",
		FileName:     "drafts-manifest.yaml",
		Title:        "Draft manifest",
		Status:       "blocked",
	},
	{
		Stage:        "drafts",
		ItemType:     "draft_copy",
		ArtifactType: "draft_copy",
		FileName:     "copy-ready.md",
		Title:        "Copy-ready draft",
		Status:       "blocked",
	},
	{
		Stage:        "drafts",
		ItemType:     "draft_verify",
		ArtifactType: "draft_verify",
		FileName:     "verify-ready.md",
		Title:        "Verify-ready draft",
		Status:       "blocked",
	},
	{
		Stage:        "queue",
		ItemType:     "queue_candidate",
		ArtifactType: "queue",
		FileName:     "queue.yaml",
		Title:        "Queue candidate",
		Status:       "blocked",
	},
	{
		Stage:        "promotion_preview",
		ItemType:     "promotion_preview",
		ArtifactType: "promotion_preview",
		FileName:     "promotion-preview.yaml",
		Title:        "Promotion preview",
		Status:       "blocked",
	},
}

var authoredStageSkeletonLinks = []stageSkeletonLinkSpec{
	{
		FromStage:    "discussion",
		FromItemType: "discussion_package",
		ToStage:      "middle_layer",
		ToItemType:   "middle_layer_ledger",
		RelationType: "derives_from",
	},
	{
		FromStage:    "middle_layer",
		FromItemType: "middle_layer_ledger",
		ToStage:      "changes",
		ToItemType:   "change_ledger",
		RelationType: "derives_from",
	},
	{
		FromStage:    "changes",
		FromItemType: "change_ledger",
		ToStage:      "plans",
		ToItemType:   "plan",
		RelationType: "derives_from",
	},
	{
		FromStage:    "plans",
		FromItemType: "plan",
		ToStage:      "drafts",
		ToItemType:   "draft_manifest",
		RelationType: "derives_from",
	},
	{
		FromStage:    "drafts",
		FromItemType: "draft_manifest",
		ToStage:      "drafts",
		ToItemType:   "draft_copy",
		RelationType: "derives_from",
	},
	{
		FromStage:    "drafts",
		FromItemType: "draft_manifest",
		ToStage:      "drafts",
		ToItemType:   "draft_verify",
		RelationType: "derives_from",
	},
	{
		FromStage:    "drafts",
		FromItemType: "draft_copy",
		ToStage:      "queue",
		ToItemType:   "queue_candidate",
		RelationType: "derives_from",
	},
	{
		FromStage:    "drafts",
		FromItemType: "draft_verify",
		ToStage:      "queue",
		ToItemType:   "queue_candidate",
		RelationType: "derives_from",
	},
	{
		FromStage:    "queue",
		FromItemType: "queue_candidate",
		ToStage:      "promotion_preview",
		ToItemType:   "promotion_preview",
		RelationType: "derives_from",
	},
}

type WorkflowVersion struct {
	ID              int64
	ProjectID       int64
	DisplayLabel    string
	VersionKind     string
	LifecycleStatus string
	SourcePath      string
	SourceHash      string
	ImportMode      string
	Immutable       bool
	StatusSummary   map[string]any
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ImportedAt      *time.Time
}

type WorkflowItem struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	Stage             string
	ItemType          string
	ExternalKey       string
	Title             string
	Status            string
	SourcePath        string
	SourceHash        string
	Metadata          map[string]any
	Immutable         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ImportedAt        *time.Time
}

type WorkflowItemLink struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	FromItemID        int64
	ToItemID          int64
	RelationType      string
	Metadata          map[string]any
	CreatedAt         time.Time
}

type CreateWorkflowVersionOptions struct {
	DisplayLabel   string
	IdempotencyKey string
	Actor          string
	Reason         string
	ProfileBinding ProfileBinding
}

type ProfileBinding struct {
	ProfileID      string
	ProfileVersion int
	ProfileHash    string
	ProfilePath    string
}

type CreateWorkflowVersionResult struct {
	Project        Record
	Version        WorkflowVersion
	InitialItem    WorkflowItem
	StageItems     []WorkflowItem
	Created        bool
	IdempotencyKey string
}

type ArtifactRecord struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	RunID             int64
	WorkflowItemID    int64
	ArtifactType      string
	StorageBackend    string
	URI               string
	SourcePath        string
	SHA256            string
	SizeBytes         int64
	ContentType       string
	Metadata          map[string]any
	CreatedAt         time.Time
}

type GateResult struct {
	ID                  int64
	ProjectID           int64
	WorkflowVersionID   int64
	WorkflowItemID      int64
	GateName            string
	ScopeType           string
	ScopeID             string
	Status              string
	Inputs              map[string]any
	SourceHashes        map[string]any
	Failures            []string
	Warnings            []string
	EvidenceArtifactIDs []int64
	Metadata            map[string]any
	CheckedAt           time.Time
}

type WorkflowTransitionPreview struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	FromStage         string
	ToStage           string
	Status            string
	RequiredGateName  string
	GateResultID      int64
	Blockers          []string
	Warnings          []string
	Metadata          map[string]any
	CreatedAt         time.Time
}

type ApprovalRecord struct {
	ID                  int64
	ProjectID           int64
	WorkflowVersionID   int64
	TransitionPreviewID int64
	ApprovalKind        string
	Decision            string
	ScopeType           string
	ScopeID             string
	Actor               string
	Reason              string
	RiskLevel           string
	Metadata            map[string]any
	CreatedAt           time.Time
}

type RunGateOptions struct {
	GateName       string
	Actor          string
	Reason         string
	IdempotencyKey string
}

type PreviewTransitionOptions struct {
	FromStage      string
	ToStage        string
	Actor          string
	Reason         string
	IdempotencyKey string
}

type CreateApprovalOptions struct {
	Decision            string
	ApprovalKind        string
	Actor               string
	Reason              string
	RiskLevel           string
	IdempotencyKey      string
	TransitionPreviewID int64
	Metadata            map[string]any
}

type EnsureStageSkeletonOptions struct {
	Actor  string
	Reason string
}

type MarkWorkflowItemReadyOptions struct {
	Stage    string
	ItemType string
	Actor    string
	Reason   string
}

type EnsureStageSkeletonResult struct {
	Project   Record
	Version   WorkflowVersion
	Items     []WorkflowItem
	Artifacts []ArtifactRecord
	Links     []WorkflowItemLink
	Created   int
}

type MarkWorkflowItemReadyResult struct {
	Project  Record
	Version  WorkflowVersion
	Item     WorkflowItem
	Artifact ArtifactRecord
}

type stageSkeletonSpec struct {
	Stage        string
	ItemType     string
	ArtifactType string
	FileName     string
	Title        string
	Status       string
}

type stageSkeletonLinkSpec struct {
	FromStage    string
	FromItemType string
	ToStage      string
	ToItemType   string
	RelationType string
}

type workflowGateSpec struct {
	GateName           string
	Phase              string
	ScopeType          string
	TargetStage        string
	TargetItem         string
	RequiredItems      []workflowGateRequiredItem
	PlaceholderFailure string
}

type workflowGateRequiredItem struct {
	Stage    string
	ItemType string
	Name     string
}

type scanner interface {
	Scan(dest ...any) error
}

func ValidateWorkflowVersionLabel(label string) error {
	if !workflowVersionLabelPattern.MatchString(label) {
		return fmt.Errorf("%w: use 1-64 chars; start with a letter or digit; only letters, digits, dot, underscore, and hyphen are allowed", ErrInvalidWorkflowVersionLabel)
	}
	return nil
}

func (s Store) ListWorkflowVersions(ctx context.Context, record Record) ([]WorkflowVersion, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, display_label, version_kind, lifecycle_status,
       COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
       immutable, status_summary, created_at, updated_at, imported_at
FROM workflow_versions
WHERE project_id = $1
ORDER BY created_at ASC, id ASC`,
		record.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow versions: %w", err)
	}
	defer rows.Close()

	versions := []WorkflowVersion{}
	for rows.Next() {
		version, err := scanWorkflowVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow versions: %w", err)
	}
	return versions, nil
}

func (s Store) GetWorkflowVersion(ctx context.Context, record Record, label string) (WorkflowVersion, error) {
	if err := ValidateWorkflowVersionLabel(label); err != nil {
		return WorkflowVersion{}, err
	}
	version, err := scanWorkflowVersion(s.pool.QueryRow(ctx, `
SELECT id, project_id, display_label, version_kind, lifecycle_status,
       COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
       immutable, status_summary, created_at, updated_at, imported_at
FROM workflow_versions
WHERE project_id = $1 AND display_label = $2`,
		record.ID,
		label,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkflowVersion{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotFound, label)
	}
	if err != nil {
		return WorkflowVersion{}, err
	}
	return version, nil
}

func (s Store) CreateWorkflowVersion(ctx context.Context, record Record, options CreateWorkflowVersionOptions) (CreateWorkflowVersionResult, error) {
	options = normalizeCreateWorkflowVersionOptions(record, options)
	if err := ValidateWorkflowVersionLabel(options.DisplayLabel); err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if options.ProfileBinding.ProfileID == "" {
		binding, err := loadProfileBinding(record.WorkflowProfile)
		if err != nil {
			return CreateWorkflowVersionResult{}, err
		}
		options.ProfileBinding = binding
	}

	requestHash, err := workflowVersionRequestHash(record, options)
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateWorkflowVersionResult{}, fmt.Errorf("begin workflow version create: %w", err)
	}
	defer tx.Rollback(ctx)

	isNewRequest, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.version.create", options.IdempotencyKey, requestHash)
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if !isNewRequest {
		result, err := loadWorkflowVersionCreation(ctx, tx, record, options.DisplayLabel)
		if err != nil {
			return CreateWorkflowVersionResult{}, err
		}
		result.IdempotencyKey = options.IdempotencyKey
		if err := tx.Commit(ctx); err != nil {
			return CreateWorkflowVersionResult{}, fmt.Errorf("commit idempotent workflow version create: %w", err)
		}
		return result, nil
	}

	exists, err := workflowVersionLabelExists(ctx, tx, record.ID, options.DisplayLabel)
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if exists {
		return CreateWorkflowVersionResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionExists, options.DisplayLabel)
	}

	version, err := insertAuthoredWorkflowVersion(ctx, tx, record.ID, options)
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	item, err := insertVersionInitItem(ctx, tx, version.ID, record.ID, options)
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	stageItems, _, _, err := ensureStageSkeletonInTx(ctx, tx, record, version, EnsureStageSkeletonOptions{
		Actor:  options.Actor,
		Reason: options.Reason,
	})
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if err := insertWorkflowVersionCreatedEvent(ctx, tx, version, item, options); err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if err := insertWorkflowVersionAuditEvent(ctx, tx, record, version, item, options); err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if err := recordStageSkeletonEvent(ctx, tx, record, version, len(stageItems), EnsureStageSkeletonOptions{
		Actor:  options.Actor,
		Reason: options.Reason,
	}); err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if err := completeCommandRequest(ctx, tx, record.ID, "workflow.version.create", options.IdempotencyKey, version, item); err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateWorkflowVersionResult{}, fmt.Errorf("commit workflow version create: %w", err)
	}

	return CreateWorkflowVersionResult{
		Project:        record,
		Version:        version,
		InitialItem:    item,
		StageItems:     stageItems,
		Created:        true,
		IdempotencyKey: options.IdempotencyKey,
	}, nil
}

func (s Store) EnsureStageSkeleton(ctx context.Context, record Record, label string, options EnsureStageSkeletonOptions) (EnsureStageSkeletonResult, error) {
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "ensure AreaFlow stage skeleton"
	}
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return EnsureStageSkeletonResult{}, err
	}
	if version.ImportMode != "authored" {
		return EnsureStageSkeletonResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EnsureStageSkeletonResult{}, fmt.Errorf("begin stage skeleton ensure: %w", err)
	}
	defer tx.Rollback(ctx)

	items, artifacts, links, err := ensureStageSkeletonInTx(ctx, tx, record, version, options)
	if err != nil {
		return EnsureStageSkeletonResult{}, err
	}
	if err := recordStageSkeletonEvent(ctx, tx, record, version, len(items), options); err != nil {
		return EnsureStageSkeletonResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EnsureStageSkeletonResult{}, fmt.Errorf("commit stage skeleton ensure: %w", err)
	}
	return EnsureStageSkeletonResult{
		Project:   record,
		Version:   version,
		Items:     items,
		Artifacts: artifacts,
		Links:     links,
		Created:   len(items),
	}, nil
}

func (s Store) MarkWorkflowItemReady(ctx context.Context, record Record, label string, options MarkWorkflowItemReadyOptions) (MarkWorkflowItemReadyResult, error) {
	options = normalizeMarkWorkflowItemReadyOptions(options)
	if options.Stage == "" || options.ItemType == "" {
		return MarkWorkflowItemReadyResult{}, fmt.Errorf("stage and item type are required")
	}
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return MarkWorkflowItemReadyResult{}, err
	}
	if version.ImportMode != "authored" {
		return MarkWorkflowItemReadyResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MarkWorkflowItemReadyResult{}, fmt.Errorf("begin workflow item ready: %w", err)
	}
	defer tx.Rollback(ctx)

	item, found, err := loadWorkflowItemByStageType(ctx, tx, record.ID, version.ID, options.Stage, options.ItemType)
	if err != nil {
		return MarkWorkflowItemReadyResult{}, err
	}
	if !found {
		return MarkWorkflowItemReadyResult{}, fmt.Errorf("workflow item not found: %s/%s", options.Stage, options.ItemType)
	}
	artifactRecord, err := writeAndInsertReadyMarkerArtifact(ctx, tx, record, version, item, options)
	if err != nil {
		return MarkWorkflowItemReadyResult{}, err
	}
	item, err = updateWorkflowItemReady(ctx, tx, item, artifactRecord, options)
	if err != nil {
		return MarkWorkflowItemReadyResult{}, err
	}
	if err := insertWorkflowItemReadyEvent(ctx, tx, record, version, item, artifactRecord, options); err != nil {
		return MarkWorkflowItemReadyResult{}, err
	}
	if err := insertWorkflowItemReadyAuditEvent(ctx, tx, record, version, item, artifactRecord, options); err != nil {
		return MarkWorkflowItemReadyResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MarkWorkflowItemReadyResult{}, fmt.Errorf("commit workflow item ready: %w", err)
	}
	return MarkWorkflowItemReadyResult{
		Project:  record,
		Version:  version,
		Item:     item,
		Artifact: artifactRecord,
	}, nil
}

func (s Store) ListWorkflowItems(ctx context.Context, record Record, version WorkflowVersion) ([]WorkflowItem, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, workflow_version_id, stage, item_type, external_key,
       COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
       COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at
FROM workflow_items
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY created_at ASC, id ASC`,
		record.ID,
		version.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow items: %w", err)
	}
	defer rows.Close()

	items := []WorkflowItem{}
	for rows.Next() {
		item, err := scanWorkflowItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow items: %w", err)
	}
	return items, nil
}

func (s Store) ListWorkflowItemLinks(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]WorkflowItemLink, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, workflow_version_id, from_item_id, to_item_id,
       relation_type, metadata, created_at
FROM workflow_item_links
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY created_at ASC, id ASC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow item links: %w", err)
	}
	defer rows.Close()

	return scanWorkflowItemLinkRows(rows)
}

func (s Store) RunDiscussionGate(ctx context.Context, record Record, label string, options RunGateOptions) (GateResult, error) {
	options.GateName = "discussion_gate"
	return s.RunWorkflowGate(ctx, record, label, options)
}

func (s Store) RunWorkflowGate(ctx context.Context, record Record, label string, options RunGateOptions) (GateResult, error) {
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.GateName = strings.TrimSpace(options.GateName)
	if options.GateName == "" {
		options.GateName = "discussion_gate"
	}
	spec, ok := workflowGateSpecs[options.GateName]
	if !ok {
		return GateResult{}, fmt.Errorf("%w: %s", ErrUnsupportedWorkflowGate, options.GateName)
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "run workflow gate"
	}
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return GateResult{}, err
	}
	if version.ImportMode != "authored" {
		return GateResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = workflowGateIdempotencyKey(record, version, spec, options)
	}
	if spec.GateName == "profile_binding_drift" {
		return s.runProfileBindingDriftGate(ctx, record, version, spec, options)
	}
	if spec.GateName == "approval_gate" || spec.GateName == "live_mapping_gate" {
		return s.runApprovalBackedGate(ctx, record, version, spec, options)
	}
	if spec.GateName == "cutover_readiness_gate" {
		return s.runCutoverReadinessGate(ctx, record, version, spec, options)
	}
	items, err := s.ListWorkflowItems(ctx, record, version)
	if err != nil {
		return GateResult{}, err
	}
	result := evaluateWorkflowGate(record, version, items, spec, options)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GateResult{}, fmt.Errorf("begin workflow gate: %w", err)
	}
	defer tx.Rollback(ctx)
	requestHash, err := workflowGateRequestHash(record, version, result, options)
	if err != nil {
		return GateResult{}, err
	}
	if _, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, requestHash); err != nil {
		return GateResult{}, err
	}
	inserted, err := insertGateResult(ctx, tx, result)
	if err != nil {
		return GateResult{}, err
	}
	eventID, err := insertGateEvent(ctx, tx, record, version, inserted)
	if err != nil {
		return GateResult{}, err
	}
	auditEventID, err := insertGateAuditEvent(ctx, tx, record, version, inserted, options)
	if err != nil {
		return GateResult{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, workflowGateCommandResponse(inserted, eventID, auditEventID)); err != nil {
		return GateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GateResult{}, fmt.Errorf("commit workflow gate: %w", err)
	}
	return inserted, nil
}

func (s Store) runProfileBindingDriftGate(ctx context.Context, record Record, version WorkflowVersion, spec workflowGateSpec, options RunGateOptions) (GateResult, error) {
	binding, bindingFound := workflowVersionProfileBinding(version)
	var currentBinding ProfileBinding
	if bindingFound {
		root, err := workflowProfileRoot()
		if err != nil {
			result := evaluateProfileBindingDriftGate(record, version, binding, bindingFound, currentBinding, false, err.Error(), spec, options)
			return s.insertWorkflowGateResult(ctx, record, version, result, options)
		}
		loaded, err := workflowprofile.LoadBuiltInProfile(root, binding.ProfileID)
		if err != nil {
			currentBinding = ProfileBinding{ProfileID: binding.ProfileID}
			result := evaluateProfileBindingDriftGate(record, version, binding, bindingFound, currentBinding, false, err.Error(), spec, options)
			return s.insertWorkflowGateResult(ctx, record, version, result, options)
		}
		currentBinding = loadedProfileBinding(loaded)
	} else {
		currentBinding = ProfileBinding{ProfileID: record.WorkflowProfile}
	}
	result := evaluateProfileBindingDriftGate(record, version, binding, bindingFound, currentBinding, bindingFound, "", spec, options)
	return s.insertWorkflowGateResult(ctx, record, version, result, options)
}

func (s Store) insertWorkflowGateResult(ctx context.Context, record Record, version WorkflowVersion, result GateResult, options RunGateOptions) (GateResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GateResult{}, fmt.Errorf("begin workflow gate: %w", err)
	}
	defer tx.Rollback(ctx)
	requestHash, err := workflowGateRequestHash(record, version, result, options)
	if err != nil {
		return GateResult{}, err
	}
	if _, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, requestHash); err != nil {
		return GateResult{}, err
	}
	inserted, err := insertGateResult(ctx, tx, result)
	if err != nil {
		return GateResult{}, err
	}
	eventID, err := insertGateEvent(ctx, tx, record, version, inserted)
	if err != nil {
		return GateResult{}, err
	}
	auditEventID, err := insertGateAuditEvent(ctx, tx, record, version, inserted, options)
	if err != nil {
		return GateResult{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, workflowGateCommandResponse(inserted, eventID, auditEventID)); err != nil {
		return GateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GateResult{}, fmt.Errorf("commit workflow gate: %w", err)
	}
	return inserted, nil
}

func (s Store) runCutoverReadinessGate(ctx context.Context, record Record, version WorkflowVersion, spec workflowGateSpec, options RunGateOptions) (GateResult, error) {
	readiness, err := s.ProjectCutoverReadiness(ctx, record, version.DisplayLabel, 10)
	if err != nil {
		return GateResult{}, err
	}
	result := evaluateCutoverReadinessGate(record, version, readiness, spec, options)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GateResult{}, fmt.Errorf("begin workflow gate: %w", err)
	}
	defer tx.Rollback(ctx)
	requestHash, err := workflowGateRequestHash(record, version, result, options)
	if err != nil {
		return GateResult{}, err
	}
	if _, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, requestHash); err != nil {
		return GateResult{}, err
	}
	inserted, err := insertGateResult(ctx, tx, result)
	if err != nil {
		return GateResult{}, err
	}
	eventID, err := insertGateEvent(ctx, tx, record, version, inserted)
	if err != nil {
		return GateResult{}, err
	}
	auditEventID, err := insertGateAuditEvent(ctx, tx, record, version, inserted, options)
	if err != nil {
		return GateResult{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, workflowGateCommandResponse(inserted, eventID, auditEventID)); err != nil {
		return GateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GateResult{}, fmt.Errorf("commit workflow gate: %w", err)
	}
	return inserted, nil
}

func (s Store) ListGateResults(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]GateResult, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       gate_name, scope_type, scope_id, status, inputs, source_hashes,
       failures, warnings, evidence_artifact_ids, metadata, checked_at
FROM gate_results
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY checked_at DESC, id DESC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list gate results: %w", err)
	}
	defer rows.Close()
	return scanGateRows(rows)
}

func (s Store) PreviewWorkflowTransition(ctx context.Context, record Record, label string, options PreviewTransitionOptions) (WorkflowTransitionPreview, error) {
	options = normalizePreviewTransitionOptions(options)
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return WorkflowTransitionPreview{}, err
	}
	if version.ImportMode != "authored" {
		return WorkflowTransitionPreview{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	gate, found, err := s.latestGateResult(ctx, record.ID, version.ID, "promotion_preview")
	if err != nil {
		return WorkflowTransitionPreview{}, err
	}
	preview := evaluateWorkflowTransitionPreview(record, version, gate, found, options)
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = workflowTransitionPreviewIdempotencyKey(record, version, preview, options)
	}
	requestHash, err := workflowTransitionPreviewRequestHash(record, version, preview, options)
	if err != nil {
		return WorkflowTransitionPreview{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("begin transition preview: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.transition.preview", options.IdempotencyKey, requestHash); err != nil {
		return WorkflowTransitionPreview{}, err
	}
	inserted, err := insertWorkflowTransitionPreview(ctx, tx, preview)
	if err != nil {
		return WorkflowTransitionPreview{}, err
	}
	eventID, err := insertTransitionPreviewEvent(ctx, tx, record, version, inserted, options)
	if err != nil {
		return WorkflowTransitionPreview{}, err
	}
	auditEventID, err := insertTransitionPreviewAuditEvent(ctx, tx, record, version, inserted, options)
	if err != nil {
		return WorkflowTransitionPreview{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "workflow.transition.preview", options.IdempotencyKey, transitionPreviewCommandResponse(inserted, eventID, auditEventID)); err != nil {
		return WorkflowTransitionPreview{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("commit transition preview: %w", err)
	}
	return inserted, nil
}

func (s Store) ListWorkflowTransitionPreviews(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]WorkflowTransitionPreview, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, workflow_version_id, from_stage, to_stage, status,
       required_gate_name, COALESCE(gate_result_id, 0), blockers, warnings,
       metadata, created_at
FROM workflow_transition_previews
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY created_at DESC, id DESC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list transition previews: %w", err)
	}
	defer rows.Close()
	return scanTransitionPreviewRows(rows)
}

func (s Store) CreateApprovalRecord(ctx context.Context, record Record, label string, options CreateApprovalOptions) (ApprovalRecord, error) {
	options = normalizeCreateApprovalOptions(options)
	if options.Decision != "approved" && options.Decision != "rejected" {
		return ApprovalRecord{}, fmt.Errorf("%w: %s", ErrInvalidApprovalDecision, options.Decision)
	}
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return ApprovalRecord{}, err
	}
	if version.ImportMode != "authored" {
		return ApprovalRecord{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	preview, found, err := s.resolveApprovalTransitionPreview(ctx, record.ID, version.ID, options.TransitionPreviewID)
	if err != nil {
		return ApprovalRecord{}, err
	}
	if options.TransitionPreviewID != 0 && !found {
		return ApprovalRecord{}, fmt.Errorf("transition preview not found: %d", options.TransitionPreviewID)
	}
	if options.Decision == "approved" && (!found || preview.Status != "ready") {
		return ApprovalRecord{}, fmt.Errorf("%w: create a ready transition preview before approving", ErrApprovalPreviewNotReady)
	}
	if options.Decision == "approved" && highRiskApproval(options.RiskLevel) && sameActor(options.Actor, preview.Metadata["actor"]) {
		return ApprovalRecord{}, fmt.Errorf("high-risk approval requires an approver different from the transition requester")
	}
	approval := buildApprovalRecord(record, version, preview, found, options)
	requestHash, err := approvalRecordRequestHash(record, version, approval)
	if err != nil {
		return ApprovalRecord{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = approvalRecordIdempotencyKey(record, version, approval, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApprovalRecord{}, fmt.Errorf("begin approval record: %w", err)
	}
	defer tx.Rollback(ctx)
	created, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.approval.record", options.IdempotencyKey, requestHash)
	if err != nil {
		return ApprovalRecord{}, err
	}
	if !created {
		return loadApprovalRecordByIdempotency(ctx, tx, record.ID, options.IdempotencyKey)
	}
	inserted, err := insertApprovalRecord(ctx, tx, approval)
	if err != nil {
		return ApprovalRecord{}, err
	}
	eventID, err := insertApprovalEvent(ctx, tx, record, version, inserted)
	if err != nil {
		return ApprovalRecord{}, err
	}
	auditEventID, err := insertApprovalAuditEvent(ctx, tx, record, version, inserted)
	if err != nil {
		return ApprovalRecord{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "workflow.approval.record", options.IdempotencyKey, map[string]any{
		"approval_record_id":    inserted.ID,
		"event_id":              eventID,
		"audit_event_id":        auditEventID,
		"workflow_version_id":   version.ID,
		"display_label":         version.DisplayLabel,
		"transition_preview_id": inserted.TransitionPreviewID,
		"decision":              inserted.Decision,
	}); err != nil {
		return ApprovalRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApprovalRecord{}, fmt.Errorf("commit approval record: %w", err)
	}
	return inserted, nil
}

func (s Store) ListApprovalRecords(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]ApprovalRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, workflow_version_id, COALESCE(transition_preview_id, 0),
       approval_kind, decision, scope_type, scope_id, actor, reason,
       risk_level, metadata, created_at
FROM approval_records
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY created_at DESC, id DESC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list approval records: %w", err)
	}
	defer rows.Close()
	return scanApprovalRows(rows)
}

func (s Store) runApprovalBackedGate(ctx context.Context, record Record, version WorkflowVersion, spec workflowGateSpec, options RunGateOptions) (GateResult, error) {
	approval, approvalFound, err := s.latestApprovalRecord(ctx, record.ID, version.ID)
	if err != nil {
		return GateResult{}, err
	}
	preview, previewFound, err := s.resolveApprovalTransitionPreview(ctx, record.ID, version.ID, approval.TransitionPreviewID)
	if err != nil {
		return GateResult{}, err
	}
	var result GateResult
	switch spec.GateName {
	case "approval_gate":
		result = evaluateApprovalGate(record, version, approval, approvalFound, preview, previewFound, options)
	case "live_mapping_gate":
		gate, gateFound, err := s.latestGateResult(ctx, record.ID, version.ID, "promotion_preview")
		if err != nil {
			return GateResult{}, err
		}
		result = evaluateLiveMappingGate(record, version, approval, approvalFound, preview, previewFound, gate, gateFound, options)
	default:
		return GateResult{}, fmt.Errorf("%w: %s", ErrUnsupportedWorkflowGate, spec.GateName)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GateResult{}, fmt.Errorf("begin workflow gate: %w", err)
	}
	defer tx.Rollback(ctx)
	requestHash, err := workflowGateRequestHash(record, version, result, options)
	if err != nil {
		return GateResult{}, err
	}
	if _, err := reserveCommandRequest(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, requestHash); err != nil {
		return GateResult{}, err
	}
	inserted, err := insertGateResult(ctx, tx, result)
	if err != nil {
		return GateResult{}, err
	}
	eventID, err := insertGateEvent(ctx, tx, record, version, inserted)
	if err != nil {
		return GateResult{}, err
	}
	auditEventID, err := insertGateAuditEvent(ctx, tx, record, version, inserted, options)
	if err != nil {
		return GateResult{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "workflow.gate.run", options.IdempotencyKey, workflowGateCommandResponse(inserted, eventID, auditEventID)); err != nil {
		return GateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GateResult{}, fmt.Errorf("commit workflow gate: %w", err)
	}
	return inserted, nil
}

func normalizeCreateWorkflowVersionOptions(record Record, options CreateWorkflowVersionOptions) CreateWorkflowVersionOptions {
	options.DisplayLabel = strings.TrimSpace(options.DisplayLabel)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "local CLI workflow version authoring"
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = fmt.Sprintf("workflow.version.create:%s:%s", record.Key, options.DisplayLabel)
	}
	return options
}

func workflowVersionRequestHash(record Record, options CreateWorkflowVersionOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":    "workflow.version.create",
		"project_key":     record.Key,
		"display_label":   options.DisplayLabel,
		"version_kind":    "workflow_version",
		"profile_binding": profileBindingMetadata(options.ProfileBinding),
	})
	if err != nil {
		return "", fmt.Errorf("marshal workflow version request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func loadProfileBinding(profileID string) (ProfileBinding, error) {
	root, err := workflowProfileRoot()
	if err != nil {
		return ProfileBinding{}, err
	}
	loaded, err := workflowprofile.LoadBuiltInProfile(root, profileID)
	if err != nil {
		return ProfileBinding{}, err
	}
	return loadedProfileBinding(loaded), nil
}

func loadedProfileBinding(loaded workflowprofile.LoadedProfile) ProfileBinding {
	return ProfileBinding{
		ProfileID:      loaded.Profile.ProfileID,
		ProfileVersion: loaded.Profile.ProfileVersion,
		ProfileHash:    loaded.SHA256,
		ProfilePath:    loaded.Path,
	}
}

func workflowProfileRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "workflow", "profiles")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("workflow profiles root not found")
		}
		dir = parent
	}
}

func profileBindingMetadata(binding ProfileBinding) map[string]any {
	return map[string]any{
		"profile_id":      binding.ProfileID,
		"profile_version": binding.ProfileVersion,
		"profile_hash":    binding.ProfileHash,
		"profile_path":    binding.ProfilePath,
	}
}

func workflowVersionProfileBinding(version WorkflowVersion) (ProfileBinding, bool) {
	raw, ok := version.StatusSummary["profile_binding"]
	if !ok {
		return ProfileBinding{}, false
	}
	metadata, ok := raw.(map[string]any)
	if !ok {
		return ProfileBinding{}, false
	}
	binding := ProfileBinding{
		ProfileID:   metadataString(metadata, "profile_id"),
		ProfileHash: metadataString(metadata, "profile_hash"),
		ProfilePath: metadataString(metadata, "profile_path"),
	}
	switch value := metadata["profile_version"].(type) {
	case int:
		binding.ProfileVersion = value
	case int64:
		binding.ProfileVersion = int(value)
	case float64:
		binding.ProfileVersion = int(value)
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			binding.ProfileVersion = int(parsed)
		}
	}
	if binding.ProfileID == "" || binding.ProfileHash == "" {
		return binding, false
	}
	return binding, true
}

func normalizePreviewTransitionOptions(options PreviewTransitionOptions) PreviewTransitionOptions {
	options.FromStage = strings.TrimSpace(options.FromStage)
	options.ToStage = strings.TrimSpace(options.ToStage)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.FromStage == "" {
		options.FromStage = "promotion_preview"
	}
	if options.ToStage == "" {
		options.ToStage = "approval"
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "preview workflow transition"
	}
	return options
}

func normalizeCreateApprovalOptions(options CreateApprovalOptions) CreateApprovalOptions {
	options.Decision = strings.TrimSpace(options.Decision)
	options.ApprovalKind = strings.TrimSpace(options.ApprovalKind)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.RiskLevel = strings.TrimSpace(options.RiskLevel)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.Decision == "" {
		options.Decision = "approved"
	}
	if options.ApprovalKind == "" {
		options.ApprovalKind = "workflow_transition"
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "explicit workflow approval"
	}
	if options.RiskLevel == "" {
		options.RiskLevel = "normal"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func highRiskApproval(riskLevel string) bool {
	switch strings.ToLower(strings.TrimSpace(riskLevel)) {
	case "high", "critical", "l4":
		return true
	default:
		return false
	}
}

func sameActor(approver string, requester any) bool {
	requesterName, ok := requester.(string)
	return ok && requesterName != "" && strings.EqualFold(strings.TrimSpace(approver), strings.TrimSpace(requesterName))
}

func approvalRecordRequestHash(record Record, version WorkflowVersion, approval ApprovalRecord) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":          "workflow.approval.record",
		"project_key":           record.Key,
		"workflow_version_id":   version.ID,
		"display_label":         version.DisplayLabel,
		"transition_preview_id": approval.TransitionPreviewID,
		"approval_kind":         approval.ApprovalKind,
		"decision":              approval.Decision,
		"scope_type":            approval.ScopeType,
		"scope_id":              approval.ScopeID,
		"actor":                 approval.Actor,
		"reason":                approval.Reason,
		"risk_level":            approval.RiskLevel,
		"metadata":              approval.Metadata,
	})
	if err != nil {
		return "", fmt.Errorf("marshal approval record request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func approvalRecordIdempotencyKey(record Record, version WorkflowVersion, approval ApprovalRecord, requestHash string) string {
	hashPrefix := requestHash
	if len(hashPrefix) > 16 {
		hashPrefix = hashPrefix[:16]
	}
	previewID := "none"
	if approval.TransitionPreviewID != 0 {
		previewID = fmt.Sprintf("%d", approval.TransitionPreviewID)
	}
	return fmt.Sprintf("workflow.approval.record:%s:%s:%s:%s:%s", record.Key, version.DisplayLabel, approval.Decision, previewID, hashPrefix)
}

func workflowGateIdempotencyKey(record Record, version WorkflowVersion, spec workflowGateSpec, options RunGateOptions) string {
	payload := fmt.Sprintf("%s:%s:%s:%s:%s", record.Key, version.DisplayLabel, spec.GateName, options.Actor, options.Reason)
	sequence := workflowGateDefaultKeySequence.Add(1)
	return fmt.Sprintf("workflow.gate.run:%s:%s:%s:%d:%d:%s", record.Key, version.DisplayLabel, spec.GateName, time.Now().UTC().UnixNano(), sequence, shortSHA256Hex([]byte(payload)))
}

func workflowGateRequestHash(record Record, version WorkflowVersion, result GateResult, options RunGateOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":        "workflow.gate.run",
		"project_key":         record.Key,
		"workflow_version_id": version.ID,
		"display_label":       version.DisplayLabel,
		"gate_name":           result.GateName,
		"scope_type":          result.ScopeType,
		"scope_id":            result.ScopeID,
		"status":              result.Status,
		"inputs":              result.Inputs,
		"source_hashes":       result.SourceHashes,
		"failures":            result.Failures,
		"warnings":            result.Warnings,
		"actor":               options.Actor,
		"reason":              options.Reason,
		"read_only":           true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal workflow gate command hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func workflowGateCommandResponse(result GateResult, eventID int64, auditEventID int64) map[string]any {
	return map[string]any{
		"gate_result_id":            result.ID,
		"event_id":                  eventID,
		"audit_event_id":            auditEventID,
		"workflow_version_id":       result.WorkflowVersionID,
		"gate_name":                 result.GateName,
		"status":                    result.Status,
		"scope_type":                result.ScopeType,
		"scope_id":                  result.ScopeID,
		"project_write_attempted":   false,
		"execution_write_attempted": metadataBool(result.Inputs, "execution_write_attempted"),
		"read_only":                 true,
	}
}

func workflowTransitionPreviewIdempotencyKey(record Record, version WorkflowVersion, preview WorkflowTransitionPreview, options PreviewTransitionOptions) string {
	payload := fmt.Sprintf("%s:%s:%s:%s:%s:%s", record.Key, version.DisplayLabel, preview.FromStage, preview.ToStage, options.Actor, options.Reason)
	sequence := transitionPreviewDefaultKeySequence.Add(1)
	return fmt.Sprintf("workflow.transition.preview:%s:%s:%s:%s:%d:%d:%s", record.Key, version.DisplayLabel, preview.FromStage, preview.ToStage, time.Now().UTC().UnixNano(), sequence, shortSHA256Hex([]byte(payload)))
}

func workflowTransitionPreviewRequestHash(record Record, version WorkflowVersion, preview WorkflowTransitionPreview, options PreviewTransitionOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":        "workflow.transition.preview",
		"project_key":         record.Key,
		"workflow_version_id": version.ID,
		"display_label":       version.DisplayLabel,
		"from_stage":          preview.FromStage,
		"to_stage":            preview.ToStage,
		"status":              preview.Status,
		"required_gate_name":  preview.RequiredGateName,
		"gate_result_id":      preview.GateResultID,
		"blockers":            preview.Blockers,
		"warnings":            preview.Warnings,
		"actor":               options.Actor,
		"reason":              options.Reason,
		"read_only":           true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal transition preview command hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func transitionPreviewCommandResponse(preview WorkflowTransitionPreview, eventID int64, auditEventID int64) map[string]any {
	return map[string]any{
		"transition_preview_id":     preview.ID,
		"event_id":                  eventID,
		"audit_event_id":            auditEventID,
		"workflow_version_id":       preview.WorkflowVersionID,
		"from_stage":                preview.FromStage,
		"to_stage":                  preview.ToStage,
		"status":                    preview.Status,
		"required_gate_name":        preview.RequiredGateName,
		"gate_result_id":            preview.GateResultID,
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"read_only":                 true,
	}
}

func normalizeMarkWorkflowItemReadyOptions(options MarkWorkflowItemReadyOptions) MarkWorkflowItemReadyOptions {
	options.Stage = strings.TrimSpace(options.Stage)
	options.ItemType = strings.TrimSpace(options.ItemType)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "mark workflow item ready"
	}
	return options
}

func reserveCommandRequest(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string, requestHash string) (bool, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO command_requests (project_id, command_type, idempotency_key, request_hash)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING id`,
		projectID,
		commandType,
		idempotencyKey,
		requestHash,
	).Scan(&id)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("reserve command request: %w", err)
	}

	var existingHash string
	err = tx.QueryRow(ctx, `
SELECT request_hash
FROM command_requests
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		projectID,
		commandType,
		idempotencyKey,
	).Scan(&existingHash)
	if err != nil {
		return false, fmt.Errorf("load existing command request: %w", err)
	}
	if existingHash != requestHash {
		return false, fmt.Errorf("%w: %s", ErrIdempotencyConflict, idempotencyKey)
	}
	return false, nil
}

func workflowVersionLabelExists(ctx context.Context, tx pgx.Tx, projectID int64, label string) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM workflow_versions
    WHERE project_id = $1 AND display_label = $2
)`,
		projectID,
		label,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check workflow version label: %w", err)
	}
	return exists, nil
}

func insertAuthoredWorkflowVersion(ctx context.Context, tx pgx.Tx, projectID int64, options CreateWorkflowVersionOptions) (WorkflowVersion, error) {
	summary, err := json.Marshal(map[string]any{
		"authoring":       true,
		"phase":           "v0.3a",
		"owned_by":        "areaflow",
		"profile_binding": profileBindingMetadata(options.ProfileBinding),
	})
	if err != nil {
		return WorkflowVersion{}, fmt.Errorf("marshal workflow version status summary: %w", err)
	}

	version, err := scanWorkflowVersion(tx.QueryRow(ctx, `
INSERT INTO workflow_versions (
    project_id, display_label, version_kind, lifecycle_status, source_path,
    source_hash, import_mode, immutable, status_summary
)
VALUES ($1, $2, 'workflow_version', 'draft', NULL, NULL, 'authored', false, $3::jsonb)
RETURNING id, project_id, display_label, version_kind, lifecycle_status,
          COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
          immutable, status_summary, created_at, updated_at, imported_at`,
		projectID,
		options.DisplayLabel,
		string(summary),
	))
	if err != nil {
		return WorkflowVersion{}, fmt.Errorf("insert authored workflow version: %w", err)
	}
	return version, nil
}

func insertVersionInitItem(ctx context.Context, tx pgx.Tx, versionID int64, projectID int64, options CreateWorkflowVersionOptions) (WorkflowItem, error) {
	metadata, err := json.Marshal(map[string]any{
		"phase":           "v0.3a",
		"owned_by":        "areaflow",
		"idempotency_key": options.IdempotencyKey,
		"actor":           options.Actor,
		"reason":          options.Reason,
		"profile_binding": profileBindingMetadata(options.ProfileBinding),
	})
	if err != nil {
		return WorkflowItem{}, fmt.Errorf("marshal version init item metadata: %w", err)
	}

	item, err := scanWorkflowItem(tx.QueryRow(ctx, `
INSERT INTO workflow_items (
    project_id, workflow_version_id, stage, item_type, external_key,
    title, status, source_path, source_hash, metadata, immutable
)
VALUES ($1, $2, 'version_init', 'workflow_version_candidate', $3, $4, 'draft', NULL, NULL, $5::jsonb, false)
RETURNING id, project_id, workflow_version_id, stage, item_type, external_key,
          COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
          COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at`,
		projectID,
		versionID,
		options.DisplayLabel+":version_init",
		options.DisplayLabel+" workflow version candidate",
		string(metadata),
	))
	if err != nil {
		return WorkflowItem{}, fmt.Errorf("insert version init workflow item: %w", err)
	}
	return item, nil
}

func insertWorkflowVersionCreatedEvent(ctx context.Context, tx pgx.Tx, version WorkflowVersion, item WorkflowItem, options CreateWorkflowVersionOptions) error {
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"workflow_item_id":    item.ID,
		"initial_stage":       item.Stage,
		"actor":               options.Actor,
		"reason":              options.Reason,
		"profile_binding":     profileBindingMetadata(options.ProfileBinding),
	})
	if err != nil {
		return fmt.Errorf("marshal workflow version event metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'workflow.version.created', 'info', 'Workflow version candidate created', $3::jsonb)`,
		version.ProjectID,
		version.ID,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert workflow version created event: %w", err)
	}
	return nil
}

func insertWorkflowVersionAuditEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, item WorkflowItem, options CreateWorkflowVersionOptions) error {
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"workflow_item_id":    item.ID,
		"import_mode":         version.ImportMode,
		"actor":               options.Actor,
		"idempotency_key":     options.IdempotencyKey,
		"profile_binding":     profileBindingMetadata(options.ProfileBinding),
	})
	if err != nil {
		return fmt.Errorf("marshal workflow version audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'workflow.version.create', 'workflow_version', $2, 'allowed', $3, $4::jsonb)`,
		record.ID,
		version.DisplayLabel,
		options.Reason,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert workflow version audit event: %w", err)
	}
	return nil
}

func completeCommandRequest(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string, version WorkflowVersion, item WorkflowItem) error {
	return completeCommandRequestResponse(ctx, tx, projectID, commandType, idempotencyKey, map[string]any{
		"workflow_version_id": version.ID,
		"display_label":       version.DisplayLabel,
		"workflow_item_id":    item.ID,
		"initial_stage":       item.Stage,
	})
}

func completeCommandRequestResponse(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string, responsePayload map[string]any) error {
	response, err := json.Marshal(responsePayload)
	if err != nil {
		return fmt.Errorf("marshal command request response: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE command_requests
SET response = $4::jsonb, completed_at = now()
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		projectID,
		commandType,
		idempotencyKey,
		string(response),
	); err != nil {
		return fmt.Errorf("complete command request: %w", err)
	}
	return nil
}

func insertGateResult(ctx context.Context, tx pgx.Tx, result GateResult) (GateResult, error) {
	inputs, err := json.Marshal(result.Inputs)
	if err != nil {
		return GateResult{}, fmt.Errorf("marshal gate inputs: %w", err)
	}
	sourceHashes, err := json.Marshal(result.SourceHashes)
	if err != nil {
		return GateResult{}, fmt.Errorf("marshal gate source hashes: %w", err)
	}
	failures, err := json.Marshal(result.Failures)
	if err != nil {
		return GateResult{}, fmt.Errorf("marshal gate failures: %w", err)
	}
	warnings, err := json.Marshal(result.Warnings)
	if err != nil {
		return GateResult{}, fmt.Errorf("marshal gate warnings: %w", err)
	}
	evidence, err := json.Marshal(result.EvidenceArtifactIDs)
	if err != nil {
		return GateResult{}, fmt.Errorf("marshal gate evidence ids: %w", err)
	}
	metadata, err := json.Marshal(result.Metadata)
	if err != nil {
		return GateResult{}, fmt.Errorf("marshal gate metadata: %w", err)
	}
	var workflowItemID any
	if result.WorkflowItemID != 0 {
		workflowItemID = result.WorkflowItemID
	}
	inserted, err := scanGateResult(tx.QueryRow(ctx, `
INSERT INTO gate_results (
    project_id, workflow_version_id, workflow_item_id, gate_name, scope_type, scope_id,
    status, inputs, source_hashes, failures, warnings, evidence_artifact_ids, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          gate_name, scope_type, scope_id, status, inputs, source_hashes,
          failures, warnings, evidence_artifact_ids, metadata, checked_at`,
		result.ProjectID,
		result.WorkflowVersionID,
		workflowItemID,
		result.GateName,
		result.ScopeType,
		result.ScopeID,
		result.Status,
		string(inputs),
		string(sourceHashes),
		string(failures),
		string(warnings),
		string(evidence),
		string(metadata),
	))
	if err != nil {
		return GateResult{}, fmt.Errorf("insert gate result: %w", err)
	}
	return inserted, nil
}

func insertGateEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, result GateResult) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"gate_result_id":      result.ID,
		"gate_name":           result.GateName,
		"status":              result.Status,
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"failures":            result.Failures,
		"warnings":            result.Warnings,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal gate event metadata: %w", err)
	}
	severity := "info"
	if result.Status == "fail" {
		severity = "error"
	}
	if result.Status == "warn" {
		severity = "warning"
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'workflow.gate.checked', $3, 'Workflow gate checked', $4::jsonb)
RETURNING id`,
		record.ID,
		version.ID,
		severity,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert gate event: %w", err)
	}
	return eventID, nil
}

func insertGateAuditEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, result GateResult, options RunGateOptions) (int64, error) {
	metadata, err := json.Marshal(workflowGateCommandResponse(result, 0, 0))
	if err != nil {
		return 0, fmt.Errorf("marshal gate audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'workflow.gate.run', 'workflow_gate', $2, 'allowed', $3, $4::jsonb)
RETURNING id`,
		record.ID,
		version.DisplayLabel+":"+result.GateName,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert gate audit event: %w", err)
	}
	return auditEventID, nil
}

func (s Store) latestGateResult(ctx context.Context, projectID int64, versionID int64, gateName string) (GateResult, bool, error) {
	result, err := scanGateResult(s.pool.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       gate_name, scope_type, scope_id, status, inputs, source_hashes,
       failures, warnings, evidence_artifact_ids, metadata, checked_at
FROM gate_results
WHERE project_id = $1 AND workflow_version_id = $2 AND gate_name = $3
ORDER BY checked_at DESC, id DESC
LIMIT 1`,
		projectID,
		versionID,
		gateName,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return GateResult{}, false, nil
	}
	if err != nil {
		return GateResult{}, false, err
	}
	return result, true, nil
}

func evaluateWorkflowTransitionPreview(record Record, version WorkflowVersion, gate GateResult, gateFound bool, options PreviewTransitionOptions) WorkflowTransitionPreview {
	blockers := []string{}
	warnings := []string{"transition preview is read-only; it does not promote or execute"}
	status := "ready"
	var gateResultID int64
	if !gateFound {
		status = "blocked"
		blockers = append(blockers, "missing promotion_preview gate result")
	} else {
		gateResultID = gate.ID
		if gate.Status != "pass" {
			status = "blocked"
			blockers = append(blockers, fmt.Sprintf("latest promotion_preview gate status is %s", gate.Status))
		}
	}
	return WorkflowTransitionPreview{
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		FromStage:         options.FromStage,
		ToStage:           options.ToStage,
		Status:            status,
		RequiredGateName:  "promotion_preview",
		GateResultID:      gateResultID,
		Blockers:          blockers,
		Warnings:          warnings,
		Metadata: map[string]any{
			"actor":         options.Actor,
			"reason":        options.Reason,
			"phase":         "v0.3d",
			"display_label": version.DisplayLabel,
			"gate_found":    gateFound,
		},
	}
}

func insertWorkflowTransitionPreview(ctx context.Context, tx pgx.Tx, preview WorkflowTransitionPreview) (WorkflowTransitionPreview, error) {
	blockers, err := json.Marshal(preview.Blockers)
	if err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("marshal transition preview blockers: %w", err)
	}
	warnings, err := json.Marshal(preview.Warnings)
	if err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("marshal transition preview warnings: %w", err)
	}
	metadata, err := json.Marshal(preview.Metadata)
	if err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("marshal transition preview metadata: %w", err)
	}
	var gateResultID any
	if preview.GateResultID != 0 {
		gateResultID = preview.GateResultID
	}
	inserted, err := scanTransitionPreview(tx.QueryRow(ctx, `
INSERT INTO workflow_transition_previews (
    project_id, workflow_version_id, from_stage, to_stage, status,
    required_gate_name, gate_result_id, blockers, warnings, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb)
RETURNING id, project_id, workflow_version_id, from_stage, to_stage, status,
          required_gate_name, COALESCE(gate_result_id, 0), blockers, warnings,
          metadata, created_at`,
		preview.ProjectID,
		preview.WorkflowVersionID,
		preview.FromStage,
		preview.ToStage,
		preview.Status,
		preview.RequiredGateName,
		gateResultID,
		string(blockers),
		string(warnings),
		string(metadata),
	))
	if err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("insert transition preview: %w", err)
	}
	return inserted, nil
}

func insertTransitionPreviewEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, preview WorkflowTransitionPreview, options PreviewTransitionOptions) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"transition_preview_id": preview.ID,
		"display_label":         version.DisplayLabel,
		"workflow_version_id":   version.ID,
		"from_stage":            preview.FromStage,
		"to_stage":              preview.ToStage,
		"status":                preview.Status,
		"blockers":              preview.Blockers,
		"warnings":              preview.Warnings,
		"actor":                 options.Actor,
		"reason":                options.Reason,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal transition preview event metadata: %w", err)
	}
	severity := "info"
	if preview.Status == "blocked" {
		severity = "warning"
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'workflow.transition.previewed', $3, 'Workflow transition previewed', $4::jsonb)
RETURNING id`,
		record.ID,
		version.ID,
		severity,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert transition preview event: %w", err)
	}
	return eventID, nil
}

func insertTransitionPreviewAuditEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, preview WorkflowTransitionPreview, options PreviewTransitionOptions) (int64, error) {
	metadata, err := json.Marshal(transitionPreviewCommandResponse(preview, 0, 0))
	if err != nil {
		return 0, fmt.Errorf("marshal transition preview audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'workflow.transition.preview', 'workflow_transition', $2, 'allowed', $3, $4::jsonb)
RETURNING id`,
		record.ID,
		version.DisplayLabel+":"+preview.FromStage+"->"+preview.ToStage,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert transition preview audit event: %w", err)
	}
	return auditEventID, nil
}

func (s Store) resolveApprovalTransitionPreview(ctx context.Context, projectID int64, versionID int64, previewID int64) (WorkflowTransitionPreview, bool, error) {
	query := `
SELECT id, project_id, workflow_version_id, from_stage, to_stage, status,
       required_gate_name, COALESCE(gate_result_id, 0), blockers, warnings,
       metadata, created_at
FROM workflow_transition_previews
WHERE project_id = $1 AND workflow_version_id = $2`
	args := []any{projectID, versionID}
	if previewID != 0 {
		query += " AND id = $3"
		args = append(args, previewID)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT 1"
	preview, err := scanTransitionPreview(s.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkflowTransitionPreview{}, false, nil
	}
	if err != nil {
		return WorkflowTransitionPreview{}, false, err
	}
	return preview, true, nil
}

func (s Store) latestApprovalRecord(ctx context.Context, projectID int64, versionID int64) (ApprovalRecord, bool, error) {
	approval, err := scanApprovalRecord(s.pool.QueryRow(ctx, `
SELECT id, project_id, workflow_version_id, COALESCE(transition_preview_id, 0),
       approval_kind, decision, scope_type, scope_id, actor, reason,
       risk_level, metadata, created_at
FROM approval_records
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		projectID,
		versionID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return ApprovalRecord{}, false, nil
	}
	if err != nil {
		return ApprovalRecord{}, false, err
	}
	return approval, true, nil
}

func evaluateApprovalGate(record Record, version WorkflowVersion, approval ApprovalRecord, approvalFound bool, preview WorkflowTransitionPreview, previewFound bool, options RunGateOptions) GateResult {
	failures := []string{}
	warnings := []string{"approval gate is read-only; it does not promote or execute"}
	status := "pass"
	var previewID int64
	if !approvalFound {
		status = "blocked"
		failures = append(failures, "missing approval record")
	} else {
		if approval.Decision != "approved" {
			status = "blocked"
			failures = append(failures, fmt.Sprintf("latest approval decision is %s", approval.Decision))
		}
		if !previewFound {
			status = "blocked"
			failures = append(failures, "approval is not linked to a transition preview")
		} else {
			previewID = preview.ID
			if preview.Status != "ready" {
				status = "blocked"
				failures = append(failures, fmt.Sprintf("linked transition preview status is %s", preview.Status))
			}
		}
	}
	return GateResult{
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		GateName:          "approval_gate",
		ScopeType:         "workflow_version",
		ScopeID:           version.DisplayLabel,
		Status:            status,
		Inputs: map[string]any{
			"workflow_profile":      record.WorkflowProfile,
			"display_label":         version.DisplayLabel,
			"approval_found":        approvalFound,
			"approval_decision":     approval.Decision,
			"approval_record_id":    approval.ID,
			"transition_preview_id": previewID,
			"transition_status":     preview.Status,
		},
		SourceHashes:        map[string]any{},
		Failures:            failures,
		Warnings:            warnings,
		EvidenceArtifactIDs: []int64{},
		Metadata: map[string]any{
			"actor":        options.Actor,
			"reason":       options.Reason,
			"phase":        "v0.4",
			"target_stage": "approval",
			"target_item":  "approval_record",
		},
	}
}

func evaluateProfileBindingDriftGate(record Record, version WorkflowVersion, binding ProfileBinding, bindingFound bool, currentBinding ProfileBinding, currentFound bool, currentError string, spec workflowGateSpec, options RunGateOptions) GateResult {
	failures := []string{}
	warnings := []string{"profile binding drift gate is read-only; it does not migrate or rewrite workflow versions"}
	status := "pass"
	if !bindingFound {
		status = "blocked"
		failures = append(failures, "workflow version has no frozen profile_binding")
	}
	if bindingFound && !currentFound {
		status = "blocked"
		if currentError == "" {
			currentError = "current profile binding could not be loaded"
		}
		failures = append(failures, currentError)
	}
	if bindingFound && currentFound {
		if binding.ProfileID != currentBinding.ProfileID {
			status = "warn"
			warnings = append(warnings, fmt.Sprintf("profile id drift: frozen=%s current=%s", binding.ProfileID, currentBinding.ProfileID))
		}
		if binding.ProfileVersion != currentBinding.ProfileVersion {
			status = "warn"
			warnings = append(warnings, fmt.Sprintf("profile version drift: frozen=%d current=%d", binding.ProfileVersion, currentBinding.ProfileVersion))
		}
		if binding.ProfileHash != currentBinding.ProfileHash {
			status = "warn"
			warnings = append(warnings, "profile hash drift detected; explicit profile migration is required before silently changing gate rules")
		}
	}
	return GateResult{
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		GateName:          spec.GateName,
		ScopeType:         spec.ScopeType,
		ScopeID:           version.DisplayLabel,
		Status:            status,
		Inputs: map[string]any{
			"workflow_profile":        record.WorkflowProfile,
			"display_label":           version.DisplayLabel,
			"binding_found":           bindingFound,
			"current_profile_found":   currentFound,
			"current_profile_error":   currentError,
			"frozen_profile_id":       binding.ProfileID,
			"frozen_profile_version":  binding.ProfileVersion,
			"frozen_profile_hash":     binding.ProfileHash,
			"frozen_profile_path":     binding.ProfilePath,
			"current_profile_id":      currentBinding.ProfileID,
			"current_profile_version": currentBinding.ProfileVersion,
			"current_profile_hash":    currentBinding.ProfileHash,
			"current_profile_path":    currentBinding.ProfilePath,
			"profile_migration_done":  false,
		},
		SourceHashes: map[string]any{
			"frozen_profile_hash":  binding.ProfileHash,
			"current_profile_hash": currentBinding.ProfileHash,
		},
		Failures:            failures,
		Warnings:            warnings,
		EvidenceArtifactIDs: []int64{},
		Metadata: map[string]any{
			"actor":        options.Actor,
			"reason":       options.Reason,
			"phase":        spec.Phase,
			"target_stage": spec.TargetStage,
			"target_item":  spec.TargetItem,
		},
	}
}

func evaluateLiveMappingGate(record Record, version WorkflowVersion, approval ApprovalRecord, approvalFound bool, preview WorkflowTransitionPreview, previewFound bool, promotionGate GateResult, promotionGateFound bool, options RunGateOptions) GateResult {
	failures := []string{}
	warnings := []string{"live mapping gate is read-only in v0.4; it does not write execution"}
	status := "pass"
	if !approvalFound || approval.Decision != "approved" {
		status = "blocked"
		failures = append(failures, "approval_gate has not passed")
	}
	if !previewFound {
		status = "blocked"
		failures = append(failures, "missing transition preview")
	} else if preview.Status != "ready" {
		status = "blocked"
		failures = append(failures, fmt.Sprintf("transition preview status is %s", preview.Status))
	}
	if !promotionGateFound {
		status = "blocked"
		failures = append(failures, "missing promotion_preview gate result")
	} else if promotionGate.Status != "pass" {
		status = "blocked"
		failures = append(failures, fmt.Sprintf("promotion_preview gate status is %s", promotionGate.Status))
	}
	return GateResult{
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		GateName:          "live_mapping_gate",
		ScopeType:         "workflow_version",
		ScopeID:           version.DisplayLabel,
		Status:            status,
		Inputs: map[string]any{
			"workflow_profile":          record.WorkflowProfile,
			"display_label":             version.DisplayLabel,
			"approval_record_id":        approval.ID,
			"approval_decision":         approval.Decision,
			"transition_preview_id":     preview.ID,
			"transition_status":         preview.Status,
			"promotion_gate_result_id":  promotionGate.ID,
			"promotion_gate_status":     promotionGate.Status,
			"execution_write_attempted": false,
		},
		SourceHashes:        map[string]any{},
		Failures:            failures,
		Warnings:            warnings,
		EvidenceArtifactIDs: []int64{},
		Metadata: map[string]any{
			"actor":        options.Actor,
			"reason":       options.Reason,
			"phase":        "v0.4",
			"target_stage": "approval",
			"target_item":  "live_mapping",
		},
	}
}

func evaluateCutoverReadinessGate(record Record, version WorkflowVersion, readiness ProjectCutoverReadiness, spec workflowGateSpec, options RunGateOptions) GateResult {
	failures := []string{}
	warnings := []string{"cutover readiness gate is read-only in v0.4c; it does not apply cutover or write execution"}
	if len(readiness.PhaseGate.AcceptedWarnings) > 0 {
		warnings = append(warnings, readiness.PhaseGate.AcceptedWarnings...)
	}
	failures = append(failures, readiness.PhaseGate.Blockers...)

	status := "pass"
	if readiness.PhaseGate.Status != "pass" {
		status = "blocked"
	}
	itemStatuses := map[string]any{}
	for _, item := range readiness.Items {
		itemStatuses[item.Key] = map[string]any{
			"status":  item.Status,
			"message": item.Message,
		}
	}
	return GateResult{
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		GateName:          spec.GateName,
		ScopeType:         spec.ScopeType,
		ScopeID:           version.DisplayLabel,
		Status:            status,
		Inputs: map[string]any{
			"workflow_profile":          record.WorkflowProfile,
			"display_label":             version.DisplayLabel,
			"verification_status":       readiness.Verification.Status,
			"verification_phase_status": readiness.Verification.PhaseGate.Status,
			"compatibility_status":      readiness.Compatibility.Status,
			"phase_gate":                readiness.PhaseGate.Name,
			"phase_gate_status":         readiness.PhaseGate.Status,
			"item_statuses":             itemStatuses,
			"cutover_apply_attempted":   false,
			"execution_write_attempted": false,
		},
		SourceHashes: map[string]any{
			"latest_import": readiness.Verification.Summary.Import.SourceHash,
		},
		Failures:            failures,
		Warnings:            warnings,
		EvidenceArtifactIDs: []int64{},
		Metadata: map[string]any{
			"actor":        options.Actor,
			"reason":       options.Reason,
			"phase":        spec.Phase,
			"target_stage": spec.TargetStage,
			"target_item":  spec.TargetItem,
		},
	}
}

func buildApprovalRecord(record Record, version WorkflowVersion, preview WorkflowTransitionPreview, previewFound bool, options CreateApprovalOptions) ApprovalRecord {
	scopeID := version.DisplayLabel
	metadata := map[string]any{
		"phase":                 "v0.3d",
		"preview_found":         previewFound,
		"approval_is_execution": false,
	}
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	var previewID int64
	if previewFound {
		previewID = preview.ID
		metadata["transition_status"] = preview.Status
		metadata["from_stage"] = preview.FromStage
		metadata["to_stage"] = preview.ToStage
	}
	return ApprovalRecord{
		ProjectID:           record.ID,
		WorkflowVersionID:   version.ID,
		TransitionPreviewID: previewID,
		ApprovalKind:        options.ApprovalKind,
		Decision:            options.Decision,
		ScopeType:           "workflow_version",
		ScopeID:             scopeID,
		Actor:               options.Actor,
		Reason:              options.Reason,
		RiskLevel:           options.RiskLevel,
		Metadata:            metadata,
	}
}

func insertApprovalRecord(ctx context.Context, tx pgx.Tx, approval ApprovalRecord) (ApprovalRecord, error) {
	metadata, err := json.Marshal(approval.Metadata)
	if err != nil {
		return ApprovalRecord{}, fmt.Errorf("marshal approval metadata: %w", err)
	}
	var previewID any
	if approval.TransitionPreviewID != 0 {
		previewID = approval.TransitionPreviewID
	}
	inserted, err := scanApprovalRecord(tx.QueryRow(ctx, `
INSERT INTO approval_records (
    project_id, workflow_version_id, transition_preview_id, approval_kind,
    decision, scope_type, scope_id, actor, reason, risk_level, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
RETURNING id, project_id, workflow_version_id, COALESCE(transition_preview_id, 0),
          approval_kind, decision, scope_type, scope_id, actor, reason,
          risk_level, metadata, created_at`,
		approval.ProjectID,
		approval.WorkflowVersionID,
		previewID,
		approval.ApprovalKind,
		approval.Decision,
		approval.ScopeType,
		approval.ScopeID,
		approval.Actor,
		approval.Reason,
		approval.RiskLevel,
		string(metadata),
	))
	if err != nil {
		return ApprovalRecord{}, fmt.Errorf("insert approval record: %w", err)
	}
	return inserted, nil
}

func insertApprovalEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, approval ApprovalRecord) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"approval_record_id":    approval.ID,
		"display_label":         version.DisplayLabel,
		"workflow_version_id":   version.ID,
		"transition_preview_id": approval.TransitionPreviewID,
		"approval_kind":         approval.ApprovalKind,
		"decision":              approval.Decision,
		"actor":                 approval.Actor,
		"risk_level":            approval.RiskLevel,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal approval event metadata: %w", err)
	}
	severity := "info"
	if approval.Decision == "rejected" {
		severity = "warning"
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'workflow.approval.recorded', $3, 'Workflow approval recorded', $4::jsonb)
RETURNING id`,
		record.ID,
		version.ID,
		severity,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert approval event: %w", err)
	}
	return eventID, nil
}

func insertApprovalAuditEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, approval ApprovalRecord) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"approval_record_id":    approval.ID,
		"display_label":         version.DisplayLabel,
		"workflow_version_id":   version.ID,
		"transition_preview_id": approval.TransitionPreviewID,
		"approval_kind":         approval.ApprovalKind,
		"decision":              approval.Decision,
		"actor":                 approval.Actor,
		"risk_level":            approval.RiskLevel,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal approval audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'workflow.approval.record', 'workflow_version', $2, $3, $4, $5::jsonb)
RETURNING id`,
		record.ID,
		version.DisplayLabel,
		approval.Decision,
		approval.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert approval audit event: %w", err)
	}
	return auditEventID, nil
}

func loadWorkflowVersionCreation(ctx context.Context, tx pgx.Tx, record Record, label string) (CreateWorkflowVersionResult, error) {
	version, err := scanWorkflowVersion(tx.QueryRow(ctx, `
SELECT id, project_id, display_label, version_kind, lifecycle_status,
       COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
       immutable, status_summary, created_at, updated_at, imported_at
FROM workflow_versions
WHERE project_id = $1 AND display_label = $2`,
		record.ID,
		label,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return CreateWorkflowVersionResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotFound, label)
	}
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	item, err := scanWorkflowItem(tx.QueryRow(ctx, `
SELECT id, project_id, workflow_version_id, stage, item_type, external_key,
       COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
       COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at
FROM workflow_items
WHERE project_id = $1 AND workflow_version_id = $2 AND stage = 'version_init'
ORDER BY id ASC
LIMIT 1`,
		record.ID,
		version.ID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return CreateWorkflowVersionResult{}, fmt.Errorf("%w: missing version_init item for %s", ErrWorkflowVersionNotFound, label)
	}
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	stageItems, err := listWorkflowItemsInTx(ctx, tx, record.ID, version.ID, "version_init")
	if err != nil {
		return CreateWorkflowVersionResult{}, err
	}
	return CreateWorkflowVersionResult{
		Project:     record,
		Version:     version,
		InitialItem: item,
		StageItems:  stageItems,
		Created:     false,
	}, nil
}

func loadApprovalRecordByIdempotency(ctx context.Context, tx pgx.Tx, projectID int64, idempotencyKey string) (ApprovalRecord, error) {
	approval, err := scanApprovalRecord(tx.QueryRow(ctx, `
SELECT a.id, a.project_id, a.workflow_version_id, COALESCE(a.transition_preview_id, 0),
       a.approval_kind, a.decision, a.scope_type, a.scope_id, a.actor, a.reason,
       a.risk_level, a.metadata, a.created_at
FROM command_requests cr
JOIN approval_records a
  ON a.id = (cr.response ->> 'approval_record_id')::bigint
WHERE cr.project_id = $1
  AND cr.command_type = 'workflow.approval.record'
  AND cr.idempotency_key = $2`,
		projectID,
		idempotencyKey,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return ApprovalRecord{}, fmt.Errorf("approval record for idempotency key %s not found", idempotencyKey)
	}
	if err != nil {
		return ApprovalRecord{}, fmt.Errorf("load idempotent approval record: %w", err)
	}
	return approval, nil
}

func listWorkflowItemsInTx(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, excludeStage string) ([]WorkflowItem, error) {
	rows, err := tx.Query(ctx, `
SELECT id, project_id, workflow_version_id, stage, item_type, external_key,
       COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
       COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at
FROM workflow_items
WHERE project_id = $1 AND workflow_version_id = $2 AND stage <> $3
ORDER BY created_at ASC, id ASC`,
		projectID,
		versionID,
		excludeStage,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow items in tx: %w", err)
	}
	defer rows.Close()

	items := []WorkflowItem{}
	for rows.Next() {
		item, err := scanWorkflowItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow items in tx: %w", err)
	}
	return items, nil
}

func ensureStageSkeletonInTx(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, options EnsureStageSkeletonOptions) ([]WorkflowItem, []ArtifactRecord, []WorkflowItemLink, error) {
	items := []WorkflowItem{}
	artifacts := []ArtifactRecord{}
	for _, spec := range authoredStageSkeleton {
		externalKey := version.DisplayLabel + ":" + spec.Stage + ":" + spec.ItemType
		_, found, err := loadWorkflowItemByExternalKey(ctx, tx, record.ID, version.ID, externalKey)
		if err != nil {
			return nil, nil, nil, err
		}
		if found {
			continue
		}
		item, err := insertStageSkeletonItem(ctx, tx, record.ID, version, spec, externalKey, options)
		if err != nil {
			return nil, nil, nil, err
		}
		artifactRecord, err := writeAndInsertStageArtifact(ctx, tx, record, version, item, spec, options)
		if err != nil {
			return nil, nil, nil, err
		}
		items = append(items, item)
		artifacts = append(artifacts, artifactRecord)
	}
	links, err := ensureStageSkeletonLinks(ctx, tx, record.ID, version, options)
	if err != nil {
		return nil, nil, nil, err
	}
	return items, artifacts, links, nil
}

func loadWorkflowItemByExternalKey(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, externalKey string) (WorkflowItem, bool, error) {
	item, err := scanWorkflowItem(tx.QueryRow(ctx, `
SELECT id, project_id, workflow_version_id, stage, item_type, external_key,
       COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
       COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at
FROM workflow_items
WHERE project_id = $1 AND workflow_version_id = $2 AND external_key = $3`,
		projectID,
		versionID,
		externalKey,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkflowItem{}, false, nil
	}
	if err != nil {
		return WorkflowItem{}, false, err
	}
	return item, true, nil
}

func loadWorkflowItemByStageType(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, stage string, itemType string) (WorkflowItem, bool, error) {
	item, err := scanWorkflowItem(tx.QueryRow(ctx, `
SELECT id, project_id, workflow_version_id, stage, item_type, external_key,
       COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
       COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at
FROM workflow_items
WHERE project_id = $1 AND workflow_version_id = $2 AND stage = $3 AND item_type = $4`,
		projectID,
		versionID,
		stage,
		itemType,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkflowItem{}, false, nil
	}
	if err != nil {
		return WorkflowItem{}, false, err
	}
	return item, true, nil
}

func insertStageSkeletonItem(ctx context.Context, tx pgx.Tx, projectID int64, version WorkflowVersion, spec stageSkeletonSpec, externalKey string, options EnsureStageSkeletonOptions) (WorkflowItem, error) {
	metadata, err := json.Marshal(map[string]any{
		"phase":         "v0.3b",
		"owned_by":      "areaflow",
		"artifact":      spec.FileName,
		"actor":         options.Actor,
		"reason":        options.Reason,
		"blocked_until": blockedUntil(spec.Stage),
	})
	if err != nil {
		return WorkflowItem{}, fmt.Errorf("marshal stage skeleton metadata: %w", err)
	}
	item, err := scanWorkflowItem(tx.QueryRow(ctx, `
INSERT INTO workflow_items (
    project_id, workflow_version_id, stage, item_type, external_key,
    title, status, source_path, source_hash, metadata, immutable
)
VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, $8::jsonb, false)
RETURNING id, project_id, workflow_version_id, stage, item_type, external_key,
          COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
          COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at`,
		projectID,
		version.ID,
		spec.Stage,
		spec.ItemType,
		externalKey,
		spec.Title,
		spec.Status,
		string(metadata),
	))
	if err != nil {
		return WorkflowItem{}, fmt.Errorf("insert stage skeleton item %s: %w", externalKey, err)
	}
	return item, nil
}

func writeAndInsertStageArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, item WorkflowItem, spec stageSkeletonSpec, options EnsureStageSkeletonOptions) (ArtifactRecord, error) {
	content := stageArtifactContent(record, version, spec, options)
	relativePath := filepath.Join(version.DisplayLabel, spec.Stage, spec.FileName)
	stored, err := writeProjectArtifact(record, relativePath, []byte(content), contentTypeForArtifact(spec.FileName))
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":         "v0.3b",
		"owned_by":      "areaflow",
		"stage":         spec.Stage,
		"item_type":     spec.ItemType,
		"display_label": version.DisplayLabel,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal stage artifact metadata: %w", err)
	}
	return insertArtifactRecord(ctx, tx, record.ID, version.ID, item.ID, spec.ArtifactType, relativePath, stored, string(metadata))
}

func writeAndInsertReadyMarkerArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, item WorkflowItem, options MarkWorkflowItemReadyOptions) (ArtifactRecord, error) {
	content, err := json.MarshalIndent(map[string]any{
		"kind":          "workflow_item_ready_marker",
		"project":       record.Key,
		"display_label": version.DisplayLabel,
		"stage":         item.Stage,
		"item_type":     item.ItemType,
		"external_key":  item.ExternalKey,
		"actor":         options.Actor,
		"reason":        options.Reason,
		"phase":         "v0.4-ready-path",
		"read_only":     true,
	}, "", "  ")
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal ready marker artifact: %w", err)
	}
	content = append(content, '\n')
	fileName := item.ItemType + "-ready.json"
	relativePath := filepath.Join(version.DisplayLabel, item.Stage, fileName)
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":         "v0.4-ready-path",
		"owned_by":      "areaflow",
		"stage":         item.Stage,
		"item_type":     item.ItemType,
		"display_label": version.DisplayLabel,
		"read_only":     true,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal ready marker metadata: %w", err)
	}
	return insertArtifactRecord(ctx, tx, record.ID, version.ID, item.ID, "workflow_item_ready_marker", relativePath, stored, string(metadata))
}

func insertArtifactRecord(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, itemID int64, artifactType string, sourcePath string, stored artifact.Stored, metadata string) (ArtifactRecord, error) {
	record, err := scanArtifactRecord(tx.QueryRow(ctx, `
INSERT INTO artifacts (
    project_id, workflow_version_id, workflow_item_id, artifact_type, storage_backend,
    uri, source_path, sha256, size_bytes, content_type, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
RETURNING id, project_id, workflow_version_id, workflow_item_id, artifact_type,
          storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
          COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at`,
		projectID,
		versionID,
		itemID,
		artifactType,
		stored.Backend,
		stored.URI,
		sourcePath,
		stored.SHA256,
		stored.SizeBytes,
		stored.ContentType,
		metadata,
	))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("insert stage artifact metadata: %w", err)
	}
	return record, nil
}

func updateWorkflowItemReady(ctx context.Context, tx pgx.Tx, item WorkflowItem, artifact ArtifactRecord, options MarkWorkflowItemReadyOptions) (WorkflowItem, error) {
	metadata := map[string]any{}
	for key, value := range item.Metadata {
		metadata[key] = value
	}
	metadata["ready_actor"] = options.Actor
	metadata["ready_reason"] = options.Reason
	metadata["ready_artifact_id"] = artifact.ID
	metadata["ready_artifact_type"] = artifact.ArtifactType
	metadata["ready_phase"] = "v0.4-ready-path"
	metadata["read_only"] = true
	delete(metadata, "blocked_until")
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return WorkflowItem{}, fmt.Errorf("marshal ready workflow item metadata: %w", err)
	}
	updated, err := scanWorkflowItem(tx.QueryRow(ctx, `
UPDATE workflow_items
SET status = 'ready',
    source_path = $2,
    source_hash = $3,
    metadata = $4::jsonb,
    updated_at = now()
WHERE id = $1
RETURNING id, project_id, workflow_version_id, stage, item_type, external_key,
          COALESCE(title, ''), COALESCE(status, ''), COALESCE(source_path, ''),
          COALESCE(source_hash, ''), metadata, immutable, created_at, updated_at, imported_at`,
		item.ID,
		artifact.SourcePath,
		artifact.SHA256,
		string(metadataJSON),
	))
	if err != nil {
		return WorkflowItem{}, fmt.Errorf("update workflow item ready: %w", err)
	}
	return updated, nil
}

func insertWorkflowItemReadyEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, item WorkflowItem, artifact ArtifactRecord, options MarkWorkflowItemReadyOptions) error {
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"workflow_item_id":    item.ID,
		"stage":               item.Stage,
		"item_type":           item.ItemType,
		"artifact_id":         artifact.ID,
		"source_hash":         item.SourceHash,
		"actor":               options.Actor,
		"reason":              options.Reason,
		"read_only":           true,
	})
	if err != nil {
		return fmt.Errorf("marshal workflow item ready event metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'workflow.item.ready_marked', 'info', 'Workflow item marked ready', $3::jsonb)`,
		record.ID,
		version.ID,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert workflow item ready event: %w", err)
	}
	return nil
}

func insertWorkflowItemReadyAuditEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, item WorkflowItem, artifact ArtifactRecord, options MarkWorkflowItemReadyOptions) error {
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"workflow_item_id":    item.ID,
		"stage":               item.Stage,
		"item_type":           item.ItemType,
		"artifact_id":         artifact.ID,
		"source_hash":         item.SourceHash,
		"actor":               options.Actor,
		"read_only":           true,
	})
	if err != nil {
		return fmt.Errorf("marshal workflow item ready audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'workflow.item.mark_ready', 'workflow_item', $2, 'allowed', $3, $4::jsonb)`,
		record.ID,
		item.ExternalKey,
		options.Reason,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert workflow item ready audit event: %w", err)
	}
	return nil
}

func ensureStageSkeletonLinks(ctx context.Context, tx pgx.Tx, projectID int64, version WorkflowVersion, options EnsureStageSkeletonOptions) ([]WorkflowItemLink, error) {
	items, err := loadStageSkeletonItems(ctx, tx, projectID, version)
	if err != nil {
		return nil, err
	}
	links := make([]WorkflowItemLink, 0, len(authoredStageSkeletonLinks))
	for _, spec := range authoredStageSkeletonLinks {
		from, fromFound := items[skeletonItemKey(spec.FromStage, spec.FromItemType)]
		to, toFound := items[skeletonItemKey(spec.ToStage, spec.ToItemType)]
		if !fromFound || !toFound {
			return nil, fmt.Errorf("stage skeleton link missing item %s/%s -> %s/%s", spec.FromStage, spec.FromItemType, spec.ToStage, spec.ToItemType)
		}
		link, err := insertWorkflowItemLink(ctx, tx, projectID, version.ID, from.ID, to.ID, spec.RelationType, map[string]any{
			"phase":       "v0.3-link",
			"owned_by":    "areaflow",
			"source":      "stage_skeleton",
			"actor":       options.Actor,
			"reason":      options.Reason,
			"from_stage":  spec.FromStage,
			"to_stage":    spec.ToStage,
			"from_type":   spec.FromItemType,
			"to_type":     spec.ToItemType,
			"relation_id": skeletonLinkID(spec),
		})
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func loadStageSkeletonItems(ctx context.Context, tx pgx.Tx, projectID int64, version WorkflowVersion) (map[string]WorkflowItem, error) {
	items := make(map[string]WorkflowItem, len(authoredStageSkeleton))
	for _, spec := range authoredStageSkeleton {
		externalKey := version.DisplayLabel + ":" + spec.Stage + ":" + spec.ItemType
		item, found, err := loadWorkflowItemByExternalKey(ctx, tx, projectID, version.ID, externalKey)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("stage skeleton item missing: %s", externalKey)
		}
		items[skeletonItemKey(spec.Stage, spec.ItemType)] = item
	}
	return items, nil
}

func insertWorkflowItemLink(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, fromItemID int64, toItemID int64, relationType string, metadata map[string]any) (WorkflowItemLink, error) {
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return WorkflowItemLink{}, fmt.Errorf("marshal workflow item link metadata: %w", err)
	}
	link, err := scanWorkflowItemLink(tx.QueryRow(ctx, `
INSERT INTO workflow_item_links (
    project_id, workflow_version_id, from_item_id, to_item_id, relation_type, metadata
)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
ON CONFLICT (project_id, workflow_version_id, from_item_id, to_item_id, relation_type)
DO UPDATE SET metadata = workflow_item_links.metadata
RETURNING id, project_id, workflow_version_id, from_item_id, to_item_id,
          relation_type, metadata, created_at`,
		projectID,
		versionID,
		fromItemID,
		toItemID,
		relationType,
		string(metadataRaw),
	))
	if err != nil {
		return WorkflowItemLink{}, fmt.Errorf("insert workflow item link: %w", err)
	}
	return link, nil
}

func listWorkflowItemLinksInTx(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, limit int) ([]WorkflowItemLink, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := tx.Query(ctx, `
SELECT id, project_id, workflow_version_id, from_item_id, to_item_id,
       relation_type, metadata, created_at
FROM workflow_item_links
WHERE project_id = $1 AND workflow_version_id = $2
ORDER BY created_at ASC, id ASC
LIMIT $3`,
		projectID,
		versionID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow item links in tx: %w", err)
	}
	defer rows.Close()

	return scanWorkflowItemLinkRows(rows)
}

func skeletonItemKey(stage string, itemType string) string {
	return stage + ":" + itemType
}

func skeletonLinkID(spec stageSkeletonLinkSpec) string {
	return skeletonItemKey(spec.FromStage, spec.FromItemType) + "->" + skeletonItemKey(spec.ToStage, spec.ToItemType)
}

func recordStageSkeletonEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, created int, options EnsureStageSkeletonOptions) error {
	if created == 0 {
		return nil
	}
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"created_items":       created,
		"actor":               options.Actor,
		"reason":              options.Reason,
	})
	if err != nil {
		return fmt.Errorf("marshal stage skeleton event metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO events (project_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'workflow.stage_skeleton.created', 'info', 'Workflow stage skeleton created', $3::jsonb)`,
		record.ID,
		version.ID,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert stage skeleton event: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'workflow.stage_skeleton.create', 'workflow_version', $2, 'allowed', $3, $4::jsonb)`,
		record.ID,
		version.DisplayLabel,
		options.Reason,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert stage skeleton audit event: %w", err)
	}
	return nil
}

func stageArtifactContent(record Record, version WorkflowVersion, spec stageSkeletonSpec, options EnsureStageSkeletonOptions) string {
	return fmt.Sprintf(`# %s

project: %s
workflow_version: %s
stage: %s
item_type: %s
status: %s
owner: areaflow
phase: v0.3b
actor: %s
reason: %s

This AreaFlow-owned placeholder is stored in the AreaFlow artifact store.
It is not written into the managed project.
`, spec.Title, record.Key, version.DisplayLabel, spec.Stage, spec.ItemType, spec.Status, options.Actor, options.Reason)
}

func blockedUntil(stage string) string {
	if stage == "discussion" {
		return ""
	}
	return "upstream gate pass"
}

func evaluateWorkflowGate(record Record, version WorkflowVersion, items []WorkflowItem, spec workflowGateSpec, options RunGateOptions) GateResult {
	failures := []string{}
	warnings := []string{}
	sourceHashes := map[string]any{}
	missing := []string{}
	placeholders := []string{}
	targetItem, targetFound := findWorkflowItem(items, spec.TargetStage, spec.TargetItem)

	for _, required := range spec.RequiredItems {
		item, found := findWorkflowItem(items, required.Stage, required.ItemType)
		if !found {
			missing = append(missing, required.Name)
			failures = append(failures, fmt.Sprintf("%s workflow item is missing", required.Name))
			continue
		}
		if item.SourceHash != "" {
			sourceHashes[item.ExternalKey] = item.SourceHash
		}
		if isStageSkeletonPlaceholder(item) {
			placeholders = append(placeholders, required.Name)
		}
	}

	status := "pass"
	if len(missing) > 0 {
		status = "blocked"
	}
	if len(missing) == 0 && len(placeholders) > 0 {
		status = "fail"
		failures = append(failures, spec.PlaceholderFailure)
	}
	if len(missing) == 0 {
		warnings = append(warnings, "v0.3c does not mutate workflow item status from gate checks")
	}

	result := GateResult{
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		GateName:          spec.GateName,
		ScopeType:         spec.ScopeType,
		ScopeID:           version.DisplayLabel,
		Status:            status,
		Inputs: map[string]any{
			"workflow_profile":  record.WorkflowProfile,
			"display_label":     version.DisplayLabel,
			"item_count":        len(items),
			"required_items":    requiredGateItemNames(spec.RequiredItems),
			"missing_items":     missing,
			"placeholder_items": placeholders,
		},
		SourceHashes:        sourceHashes,
		Failures:            failures,
		Warnings:            warnings,
		EvidenceArtifactIDs: []int64{},
		Metadata: map[string]any{
			"actor":        options.Actor,
			"reason":       options.Reason,
			"phase":        spec.Phase,
			"target_stage": spec.TargetStage,
			"target_item":  spec.TargetItem,
		},
	}
	if targetFound {
		result.WorkflowItemID = targetItem.ID
	}
	return result
}

func isStageSkeletonPlaceholder(item WorkflowItem) bool {
	if item.SourceHash != "" {
		return false
	}
	if item.Status == "draft" || item.Status == "blocked" {
		return true
	}
	if item.Metadata == nil {
		return false
	}
	return metadataString(item.Metadata, "owned_by") == "areaflow" && strings.HasPrefix(metadataString(item.Metadata, "phase"), "v0.3")
}

func requiredGateItemNames(items []workflowGateRequiredItem) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func contentTypeForArtifact(fileName string) string {
	switch {
	case strings.HasSuffix(fileName, ".yaml"), strings.HasSuffix(fileName, ".yml"):
		return "application/yaml"
	case strings.HasSuffix(fileName, ".json"):
		return "application/json"
	default:
		return "text/markdown"
	}
}

func findWorkflowItem(items []WorkflowItem, stage string, itemType string) (WorkflowItem, bool) {
	for _, item := range items {
		if item.Stage == stage && item.ItemType == itemType {
			return item, true
		}
	}
	return WorkflowItem{}, false
}

func hasWorkflowItem(items []WorkflowItem, stage string, itemType string) bool {
	_, ok := findWorkflowItem(items, stage, itemType)
	return ok
}

func scanWorkflowVersion(row scanner) (WorkflowVersion, error) {
	var version WorkflowVersion
	var raw []byte
	var importedAt sql.NullTime
	if err := row.Scan(
		&version.ID,
		&version.ProjectID,
		&version.DisplayLabel,
		&version.VersionKind,
		&version.LifecycleStatus,
		&version.SourcePath,
		&version.SourceHash,
		&version.ImportMode,
		&version.Immutable,
		&raw,
		&version.CreatedAt,
		&version.UpdatedAt,
		&importedAt,
	); err != nil {
		return WorkflowVersion{}, err
	}
	if err := json.Unmarshal(raw, &version.StatusSummary); err != nil {
		return WorkflowVersion{}, fmt.Errorf("parse workflow version status summary: %w", err)
	}
	if importedAt.Valid {
		version.ImportedAt = &importedAt.Time
	}
	return version, nil
}

func scanGateRows(rows pgx.Rows) ([]GateResult, error) {
	results := []GateResult{}
	for rows.Next() {
		result, err := scanGateResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gate results: %w", err)
	}
	return results, nil
}

func scanGateResult(row scanner) (GateResult, error) {
	var result GateResult
	var inputsRaw []byte
	var sourceHashesRaw []byte
	var failuresRaw []byte
	var warningsRaw []byte
	var evidenceRaw []byte
	var metadataRaw []byte
	if err := row.Scan(
		&result.ID,
		&result.ProjectID,
		&result.WorkflowVersionID,
		&result.WorkflowItemID,
		&result.GateName,
		&result.ScopeType,
		&result.ScopeID,
		&result.Status,
		&inputsRaw,
		&sourceHashesRaw,
		&failuresRaw,
		&warningsRaw,
		&evidenceRaw,
		&metadataRaw,
		&result.CheckedAt,
	); err != nil {
		return GateResult{}, err
	}
	if err := json.Unmarshal(inputsRaw, &result.Inputs); err != nil {
		return GateResult{}, fmt.Errorf("parse gate inputs: %w", err)
	}
	if err := json.Unmarshal(sourceHashesRaw, &result.SourceHashes); err != nil {
		return GateResult{}, fmt.Errorf("parse gate source hashes: %w", err)
	}
	if err := json.Unmarshal(failuresRaw, &result.Failures); err != nil {
		return GateResult{}, fmt.Errorf("parse gate failures: %w", err)
	}
	if err := json.Unmarshal(warningsRaw, &result.Warnings); err != nil {
		return GateResult{}, fmt.Errorf("parse gate warnings: %w", err)
	}
	if err := json.Unmarshal(evidenceRaw, &result.EvidenceArtifactIDs); err != nil {
		return GateResult{}, fmt.Errorf("parse gate evidence ids: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &result.Metadata); err != nil {
		return GateResult{}, fmt.Errorf("parse gate metadata: %w", err)
	}
	return result, nil
}

func scanTransitionPreviewRows(rows pgx.Rows) ([]WorkflowTransitionPreview, error) {
	previews := []WorkflowTransitionPreview{}
	for rows.Next() {
		preview, err := scanTransitionPreview(rows)
		if err != nil {
			return nil, err
		}
		previews = append(previews, preview)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transition previews: %w", err)
	}
	return previews, nil
}

func scanTransitionPreview(row scanner) (WorkflowTransitionPreview, error) {
	var preview WorkflowTransitionPreview
	var blockersRaw []byte
	var warningsRaw []byte
	var metadataRaw []byte
	if err := row.Scan(
		&preview.ID,
		&preview.ProjectID,
		&preview.WorkflowVersionID,
		&preview.FromStage,
		&preview.ToStage,
		&preview.Status,
		&preview.RequiredGateName,
		&preview.GateResultID,
		&blockersRaw,
		&warningsRaw,
		&metadataRaw,
		&preview.CreatedAt,
	); err != nil {
		return WorkflowTransitionPreview{}, err
	}
	if err := json.Unmarshal(blockersRaw, &preview.Blockers); err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("parse transition preview blockers: %w", err)
	}
	if err := json.Unmarshal(warningsRaw, &preview.Warnings); err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("parse transition preview warnings: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &preview.Metadata); err != nil {
		return WorkflowTransitionPreview{}, fmt.Errorf("parse transition preview metadata: %w", err)
	}
	return preview, nil
}

func scanApprovalRows(rows pgx.Rows) ([]ApprovalRecord, error) {
	approvals := []ApprovalRecord{}
	for rows.Next() {
		approval, err := scanApprovalRecord(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate approval records: %w", err)
	}
	return approvals, nil
}

func scanApprovalRecord(row scanner) (ApprovalRecord, error) {
	var approval ApprovalRecord
	var metadataRaw []byte
	if err := row.Scan(
		&approval.ID,
		&approval.ProjectID,
		&approval.WorkflowVersionID,
		&approval.TransitionPreviewID,
		&approval.ApprovalKind,
		&approval.Decision,
		&approval.ScopeType,
		&approval.ScopeID,
		&approval.Actor,
		&approval.Reason,
		&approval.RiskLevel,
		&metadataRaw,
		&approval.CreatedAt,
	); err != nil {
		return ApprovalRecord{}, err
	}
	if err := json.Unmarshal(metadataRaw, &approval.Metadata); err != nil {
		return ApprovalRecord{}, fmt.Errorf("parse approval metadata: %w", err)
	}
	return approval, nil
}

func scanArtifactRecord(row scanner) (ArtifactRecord, error) {
	var record ArtifactRecord
	var raw []byte
	var workflowItemID sql.NullInt64
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.WorkflowVersionID,
		&workflowItemID,
		&record.ArtifactType,
		&record.StorageBackend,
		&record.URI,
		&record.SourcePath,
		&record.SHA256,
		&record.SizeBytes,
		&record.ContentType,
		&raw,
		&record.CreatedAt,
	); err != nil {
		return ArtifactRecord{}, err
	}
	if err := json.Unmarshal(raw, &record.Metadata); err != nil {
		return ArtifactRecord{}, fmt.Errorf("parse artifact metadata: %w", err)
	}
	if workflowItemID.Valid {
		record.WorkflowItemID = workflowItemID.Int64
	}
	return record, nil
}

func scanArtifactRecordWithRun(row scanner) (ArtifactRecord, error) {
	var record ArtifactRecord
	var raw []byte
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.WorkflowVersionID,
		&record.RunID,
		&record.WorkflowItemID,
		&record.ArtifactType,
		&record.StorageBackend,
		&record.URI,
		&record.SourcePath,
		&record.SHA256,
		&record.SizeBytes,
		&record.ContentType,
		&raw,
		&record.CreatedAt,
	); err != nil {
		return ArtifactRecord{}, err
	}
	if err := json.Unmarshal(raw, &record.Metadata); err != nil {
		return ArtifactRecord{}, fmt.Errorf("parse artifact metadata: %w", err)
	}
	return record, nil
}

func scanWorkflowItem(row scanner) (WorkflowItem, error) {
	var item WorkflowItem
	var raw []byte
	var importedAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.ProjectID,
		&item.WorkflowVersionID,
		&item.Stage,
		&item.ItemType,
		&item.ExternalKey,
		&item.Title,
		&item.Status,
		&item.SourcePath,
		&item.SourceHash,
		&raw,
		&item.Immutable,
		&item.CreatedAt,
		&item.UpdatedAt,
		&importedAt,
	); err != nil {
		return WorkflowItem{}, err
	}
	if err := json.Unmarshal(raw, &item.Metadata); err != nil {
		return WorkflowItem{}, fmt.Errorf("parse workflow item metadata: %w", err)
	}
	if importedAt.Valid {
		item.ImportedAt = &importedAt.Time
	}
	return item, nil
}

func scanWorkflowItemLink(row scanner) (WorkflowItemLink, error) {
	var link WorkflowItemLink
	var raw []byte
	if err := row.Scan(
		&link.ID,
		&link.ProjectID,
		&link.WorkflowVersionID,
		&link.FromItemID,
		&link.ToItemID,
		&link.RelationType,
		&raw,
		&link.CreatedAt,
	); err != nil {
		return WorkflowItemLink{}, err
	}
	if err := json.Unmarshal(raw, &link.Metadata); err != nil {
		return WorkflowItemLink{}, fmt.Errorf("parse workflow item link metadata: %w", err)
	}
	return link, nil
}

func scanWorkflowItemLinkRows(rows pgx.Rows) ([]WorkflowItemLink, error) {
	links := []WorkflowItemLink{}
	for rows.Next() {
		link, err := scanWorkflowItemLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow item links: %w", err)
	}
	return links, nil
}
