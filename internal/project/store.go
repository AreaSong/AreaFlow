package project

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Record struct {
	ID              int64
	Key             string
	Name            string
	Kind            string
	Adapter         string
	WorkflowProfile string
	DefaultBranch   string
	RootPath        string
	ArtifactBackend string
	ArtifactRoot    string
}

type Snapshot struct {
	Summary    map[string]any
	SourceHash string
	CreatedAt  time.Time
}

type RecordDoctorReportOptions struct {
	IdempotencyKey string
	Actor          string
	Reason         string
}

type RecordDoctorReportResult struct {
	EventID        int64
	Severity       string
	OverallStatus  string
	IdempotencyKey string
	Created        bool
}

type StatusProjectionRecord struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	TargetKind        string
	TargetURI         string
	SummaryState      string
	Payload           map[string]any
	SourceEventID     int64
	SourceHash        string
	WriteState        string
	GeneratedAt       time.Time
	WrittenAt         *time.Time
	Metadata          map[string]any
}

type ImportInventory struct {
	Versions        int64
	Residuals       int64
	Artifacts       int64
	ImportSnapshots int64
	MirrorExports   int64
}

type EventRecord struct {
	ID                int64
	ProjectID         int64
	RunID             int64
	WorkflowVersionID int64
	Type              string
	Severity          string
	Message           string
	Metadata          map[string]any
	CreatedAt         time.Time
}

type AuditEventRecord struct {
	ID           int64
	ProjectID    int64
	ActorID      int64
	Action       string
	Capability   string
	ResourceType string
	Resource     string
	Decision     string
	Reason       string
	Metadata     map[string]any
	CreatedAt    time.Time
}

type ProjectConfigRecord struct {
	ID              int64
	ProjectID       int64
	ProtocolVersion int
	ConfigPath      string
	ConfigHash      string
	Ownership       map[string]any
	Permissions     map[string]any
	Scheduling      map[string]any
	Engines         map[string]any
	StatusExport    map[string]any
	Migration       map[string]any
	Metadata        map[string]any
	Active          bool
	LoadedAt        time.Time
	LoadedByActorID int64
}

type EventStreamFilter struct {
	ProjectID int64
	RunID     int64
	AfterID   int64
	Limit     int
}

type ResidualRecord struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	ResidualKey       string
	Status            string
	Type              string
	Title             string
	SourcePath        string
	CurrentImpact     string
	ExecutableTask    bool
	PromotionRequired bool
	CloseCondition    string
	Metadata          map[string]any
	Immutable         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ImportedAt        *time.Time
}

type ProjectSummary struct {
	Project             Record
	Config              ProjectConfigRecord
	HasConfig           bool
	Inventory           ImportInventory
	Import              Snapshot
	HasImport           bool
	PreviousImport      Snapshot
	HasPreviousImport   bool
	LatestDoctor        EventRecord
	HasLatestDoctor     bool
	DoctorStatus        string
	DriftStatus         string
	ConfigDriftStatus   string
	StageCoverageStatus string
	NativeDoctorStatus  string
	LatestEventCount    int
}

type ReadinessItem struct {
	Key      string
	Status   string
	Message  string
	Metadata map[string]any
}

type ProjectReadiness struct {
	Project Record
	Status  string
	Items   []ReadinessItem
	Summary ProjectSummary
}

type ProjectImportDiff struct {
	Project       Record
	Status        string
	HasPrevious   bool
	Latest        Snapshot
	Previous      Snapshot
	SourceChanged bool
	Changes       []ImportDiffChange
}

type ProjectVerificationBundle struct {
	Project    Record
	Status     string
	PhaseGate  PhaseGate
	Summary    ProjectSummary
	Readiness  ProjectReadiness
	ImportDiff ProjectImportDiff
	Events     []EventRecord
}

type ProjectCutoverReadiness struct {
	Project       Record
	Version       WorkflowVersion
	Status        string
	Items         []ReadinessItem
	PhaseGate     PhaseGate
	Verification  ProjectVerificationBundle
	Compatibility CompatibilityContract
	Gates         []GateResult
}

type CompatibilityContract struct {
	Project  Record
	Status   string
	Commands []CompatibilityCommand
	Summary  ProjectSummary
}

type ShimPreview struct {
	Project              Record
	Status               string
	Mode                 string
	Contract             CompatibilityContract
	PlannedFiles         []ShimFilePlan
	CommandMappings      []ShimCommandMapping
	DiscoveryOrder       []string
	ForbiddenPaths       []string
	ForbiddenCommands    []string
	VerificationCommands []string
	RollbackSteps        []string
	Notes                []string
}

type ShimReadiness struct {
	Project Record
	Status  string
	Preview ShimPreview
	Items   []ShimReadinessItem
}

type ShimReadinessItem struct {
	Key      string
	Status   string
	Message  string
	Metadata map[string]any
}

type ShimAuthorizationPacket struct {
	Project              Record
	Status               string
	Mode                 string
	Intent               string
	ReadinessStatus      string
	ReadinessItems       []ShimReadinessItem
	AllowedFiles         []ShimFilePlan
	ForbiddenPaths       []string
	ForbiddenActions     []string
	RequiredPreflight    []string
	PostEditVerification []string
	RollbackScope        []string
	SafetyFacts          map[string]bool
	NextRequiredApproval string
}

type RecordShimReadinessEvidenceOptions struct {
	EvidenceKey    string
	Status         string
	Summary        string
	EvidenceURI    string
	IdempotencyKey string
	Actor          string
	Reason         string
	Metadata       map[string]any
}

type RecordShimReadinessEvidenceResult struct {
	Project                 Record
	EvidenceKey             string
	Status                  string
	Decision                string
	Message                 string
	EventID                 int64
	AuditEventID            int64
	IdempotencyKey          string
	Created                 bool
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	Metadata                map[string]any
}

type ShimFilePlan struct {
	Path     string
	Action   string
	Required bool
	Reason   string
	Boundary string
}

type ShimCommandMapping struct {
	Command        string
	Mode           string
	Status         string
	AreaFlowTarget string
	Fallback       string
	BlockedReason  string
	ReadOnly       bool
	RequiresNative bool
	Message        string
}

type CompatibilityCommand struct {
	Command        string
	Mode           string
	Status         string
	Message        string
	AreaFlowTarget string
	Fallback       string
	BlockedReason  string
	Metadata       map[string]any
}

type PhaseGate struct {
	Name             string
	Status           string
	AcceptedWarnings []string
	Blockers         []string
}

type ImportDiffChange struct {
	Key      string
	Status   string
	Previous string
	Latest   string
}

type CommandPermission struct {
	CapabilityAllowed bool
	CommandAllowed    bool
	Denied            bool
	Reason            string
}

type compatibilityCommandSpec struct {
	Command        string
	AreaFlowTarget string
	Fallback       string
	RequiresNative bool
	ReadOnly       bool
	Forbidden      bool
}

var compatibilityCommandSpecs = []compatibilityCommandSpec{
	{
		Command:        "./dev workflow status",
		AreaFlowTarget: "areaflow project summary",
		Fallback:       ".areaflow/status.json",
		ReadOnly:       true,
	},
	{
		Command:        "./dev workflow doctor",
		AreaFlowTarget: "areaflow project doctor",
		Fallback:       ".areaflow/status.json",
		RequiresNative: true,
		ReadOnly:       true,
	},
	{
		Command:        "./dev workflow init --version <version>",
		AreaFlowTarget: "areaflow workflow version create",
		Fallback:       "show AreaFlow unavailable message",
		ReadOnly:       false,
	},
	{
		Command:        "./dev workflow open",
		AreaFlowTarget: "areaflow workflow version list",
		Fallback:       ".areaflow/status.json",
		ReadOnly:       true,
	},
	{
		Command:        "./task-loop status",
		AreaFlowTarget: "areaflow project summary",
		Fallback:       ".areaflow/status.json",
		ReadOnly:       true,
	},
	{
		Command:        "./task-loop run",
		Fallback:       "blocked until v0.5/v0.6 runner model",
		Forbidden:      true,
		RequiresNative: false,
	},
}

type Store struct {
	pool *pgxpool.Pool
}

func (s Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

var doctorReportDefaultKeySequence atomic.Int64

const shimReadinessEvidenceCommandType = "project.shim_readiness_evidence.record"

var allowedShimReadinessEvidenceKeys = map[string]bool{
	"real_areamatrix_readonly_smoke":           true,
	"real_areamatrix_status_projection_schema": true,
	"areamatrix_dirty_worktree_review":         true,
}

func (s Store) RecordShimReadinessEvidence(ctx context.Context, record Record, options RecordShimReadinessEvidenceOptions) (RecordShimReadinessEvidenceResult, error) {
	options = normalizeRecordShimReadinessEvidenceOptions(options)
	if !allowedShimReadinessEvidenceKeys[options.EvidenceKey] {
		return RecordShimReadinessEvidenceResult{}, fmt.Errorf("unsupported shim readiness evidence key %q", options.EvidenceKey)
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = shimReadinessEvidenceIdempotencyKey(record, options)
	}
	requestHash, err := shimReadinessEvidenceRequestHash(record, options)
	if err != nil {
		return RecordShimReadinessEvidenceResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RecordShimReadinessEvidenceResult{}, fmt.Errorf("begin shim readiness evidence record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, shimReadinessEvidenceCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return RecordShimReadinessEvidenceResult{}, err
	}
	if !created {
		result, err := loadShimReadinessEvidenceByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return RecordShimReadinessEvidenceResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return RecordShimReadinessEvidenceResult{}, fmt.Errorf("commit idempotent shim readiness evidence record: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := buildShimReadinessEvidenceResult(record, options)
	eventID, err := insertShimReadinessEvidenceEvent(ctx, tx, result, options)
	if err != nil {
		return RecordShimReadinessEvidenceResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertShimReadinessEvidenceAuditEvent(ctx, tx, result, options)
	if err != nil {
		return RecordShimReadinessEvidenceResult{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, shimReadinessEvidenceCommandType, options.IdempotencyKey, shimReadinessEvidenceCommandResponse(result)); err != nil {
		return RecordShimReadinessEvidenceResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RecordShimReadinessEvidenceResult{}, fmt.Errorf("commit shim readiness evidence record: %w", err)
	}
	return result, nil
}

func normalizeRecordShimReadinessEvidenceOptions(options RecordShimReadinessEvidenceOptions) RecordShimReadinessEvidenceOptions {
	options.EvidenceKey = strings.TrimSpace(options.EvidenceKey)
	options.Status = strings.TrimSpace(options.Status)
	options.Summary = strings.TrimSpace(options.Summary)
	options.EvidenceURI = strings.TrimSpace(options.EvidenceURI)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Status == "" {
		options.Status = "pass"
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "record shim readiness evidence"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	return options
}

func shimReadinessEvidenceRequestHash(record Record, options RecordShimReadinessEvidenceOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": shimReadinessEvidenceCommandType,
		"project_id":   record.ID,
		"project_key":  record.Key,
		"evidence_key": options.EvidenceKey,
		"status":       options.Status,
		"summary":      options.Summary,
		"evidence_uri": options.EvidenceURI,
		"actor":        options.Actor,
		"reason":       options.Reason,
		"metadata":     options.Metadata,
		"protected":    true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal shim readiness evidence command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func shimReadinessEvidenceIdempotencyKey(record Record, options RecordShimReadinessEvidenceOptions) string {
	hash, err := shimReadinessEvidenceRequestHash(record, options)
	if err != nil {
		hash = "no-request-hash"
	}
	prefix := hash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("project.shim_readiness_evidence.record:%s:%s:%s", record.Key, options.EvidenceKey, prefix)
}

func buildShimReadinessEvidenceResult(record Record, options RecordShimReadinessEvidenceOptions) RecordShimReadinessEvidenceResult {
	metadata := map[string]any{}
	for key, value := range options.Metadata {
		metadata[key] = value
	}
	metadata["evidence_uri"] = options.EvidenceURI
	metadata["summary"] = options.Summary
	return RecordShimReadinessEvidenceResult{
		Project:                 record,
		EvidenceKey:             options.EvidenceKey,
		Status:                  "recorded",
		Decision:                "allowed",
		Message:                 "shim readiness evidence recorded",
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
		Metadata:                metadata,
	}
}

func insertShimReadinessEvidenceEvent(ctx context.Context, tx pgx.Tx, result RecordShimReadinessEvidenceResult, options RecordShimReadinessEvidenceOptions) (int64, error) {
	metadata, err := json.Marshal(shimReadinessEvidenceEventMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal shim readiness evidence event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, 'project.shim_readiness_evidence.recorded', 'info', 'Shim readiness evidence recorded', $2::jsonb)
RETURNING id`,
		result.Project.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert shim readiness evidence event: %w", err)
	}
	return eventID, nil
}

func insertShimReadinessEvidenceAuditEvent(ctx context.Context, tx pgx.Tx, result RecordShimReadinessEvidenceResult, options RecordShimReadinessEvidenceOptions) (int64, error) {
	metadata, err := json.Marshal(shimReadinessEvidenceCommandResponse(result))
	if err != nil {
		return 0, fmt.Errorf("marshal shim readiness evidence audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'read_project', 'shim_readiness_evidence', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		shimReadinessEvidenceCommandType,
		result.EvidenceKey,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert shim readiness evidence audit event: %w", err)
	}
	return auditEventID, nil
}

func shimReadinessEvidenceEventMetadata(result RecordShimReadinessEvidenceResult, options RecordShimReadinessEvidenceOptions) map[string]any {
	metadata := shimReadinessEvidenceCommandResponse(result)
	metadata["evidence_status"] = options.Status
	metadata["actor"] = options.Actor
	metadata["reason"] = options.Reason
	return metadata
}

func shimReadinessEvidenceCommandResponse(result RecordShimReadinessEvidenceResult) map[string]any {
	return map[string]any{
		"project_key":               result.Project.Key,
		"evidence_key":              result.EvidenceKey,
		"status":                    result.Status,
		"decision":                  result.Decision,
		"message":                   result.Message,
		"event_id":                  result.EventID,
		"audit_event_id":            result.AuditEventID,
		"idempotency_key":           result.IdempotencyKey,
		"project_write_attempted":   result.ProjectWriteAttempted,
		"execution_write_attempted": result.ExecutionWriteAttempted,
		"engine_call_attempted":     result.EngineCallAttempted,
		"summary":                   metadataString(result.Metadata, "summary"),
		"evidence_uri":              metadataString(result.Metadata, "evidence_uri"),
		"metadata":                  result.Metadata,
	}
}

func loadShimReadinessEvidenceByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (RecordShimReadinessEvidenceResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, shimReadinessEvidenceCommandType, idempotencyKey)
	if err != nil {
		return RecordShimReadinessEvidenceResult{}, err
	}
	metadata := map[string]any{}
	if raw, ok := response["metadata"].(map[string]any); ok {
		metadata = raw
	}
	return RecordShimReadinessEvidenceResult{
		Project:                 record,
		EvidenceKey:             metadataString(response, "evidence_key"),
		Status:                  metadataString(response, "status"),
		Decision:                metadataString(response, "decision"),
		Message:                 metadataString(response, "message"),
		EventID:                 metadataInt64(response, "event_id"),
		AuditEventID:            metadataInt64(response, "audit_event_id"),
		IdempotencyKey:          idempotencyKey,
		ProjectWriteAttempted:   metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted: metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:     metadataBool(response, "engine_call_attempted"),
		Metadata:                metadata,
	}, nil
}

func (s Store) ImportInventory(ctx context.Context, projectID int64) (ImportInventory, error) {
	var inventory ImportInventory
	err := s.pool.QueryRow(ctx, `
SELECT
    (SELECT COUNT(*) FROM workflow_versions WHERE project_id = $1 AND import_mode = 'metadata_only'),
    (SELECT COUNT(*) FROM residuals WHERE project_id = $1 AND imported_at IS NOT NULL),
    (SELECT COUNT(*) FROM artifacts WHERE project_id = $1 AND storage_backend = 'external_project'),
    (SELECT COUNT(*) FROM project_status_snapshots WHERE project_id = $1 AND snapshot_kind = 'import'),
    (SELECT COUNT(*) FROM project_status_snapshots WHERE project_id = $1 AND snapshot_kind = 'mirror_export')`,
		projectID,
	).Scan(
		&inventory.Versions,
		&inventory.Residuals,
		&inventory.Artifacts,
		&inventory.ImportSnapshots,
		&inventory.MirrorExports,
	)
	if err != nil {
		return ImportInventory{}, fmt.Errorf("load import inventory: %w", err)
	}
	return inventory, nil
}

func (s Store) LatestImportSnapshot(ctx context.Context, projectID int64) (Snapshot, error) {
	var raw []byte
	var snapshot Snapshot
	err := s.pool.QueryRow(ctx, `
SELECT summary, COALESCE(source_hash, ''), created_at
FROM project_status_snapshots
WHERE project_id = $1 AND snapshot_kind = 'import'
ORDER BY created_at DESC, id DESC
LIMIT 1`,
		projectID,
	).Scan(&raw, &snapshot.SourceHash, &snapshot.CreatedAt)
	if err != nil {
		return Snapshot{}, fmt.Errorf("get latest import snapshot: %w", err)
	}
	if err := json.Unmarshal(raw, &snapshot.Summary); err != nil {
		return Snapshot{}, fmt.Errorf("parse latest import snapshot: %w", err)
	}
	return snapshot, nil
}

func (s Store) RecentImportSnapshots(ctx context.Context, projectID int64, limit int) ([]Snapshot, error) {
	if limit <= 0 {
		limit = 2
	}
	rows, err := s.pool.Query(ctx, `
SELECT summary, COALESCE(source_hash, ''), created_at
FROM project_status_snapshots
WHERE project_id = $1 AND snapshot_kind = 'import'
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		projectID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent import snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := []Snapshot{}
	for rows.Next() {
		var raw []byte
		var snapshot Snapshot
		if err := rows.Scan(&raw, &snapshot.SourceHash, &snapshot.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent import snapshot: %w", err)
		}
		if err := json.Unmarshal(raw, &snapshot.Summary); err != nil {
			return nil, fmt.Errorf("parse recent import snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent import snapshots: %w", err)
	}
	return snapshots, nil
}

func (s Store) ListStatusProjections(ctx context.Context, record Record, limit int) ([]StatusProjectionRecord, error) {
	if record.ID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), target_kind, target_uri,
       summary_state, payload_json, COALESCE(source_event_id, 0), COALESCE(source_hash, ''),
       write_state, generated_at, written_at, metadata
FROM status_projections
WHERE project_id = $1
ORDER BY generated_at DESC, id DESC
LIMIT $2`,
		record.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list status projections: %w", err)
	}
	defer rows.Close()

	projections := []StatusProjectionRecord{}
	for rows.Next() {
		projection, err := scanStatusProjectionRecord(rows)
		if err != nil {
			return nil, err
		}
		projections = append(projections, projection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status projections: %w", err)
	}
	return projections, nil
}

func scanStatusProjectionRecord(row scanner) (StatusProjectionRecord, error) {
	var projection StatusProjectionRecord
	var payloadRaw []byte
	var metadataRaw []byte
	var writtenAt sql.NullTime
	if err := row.Scan(
		&projection.ID,
		&projection.ProjectID,
		&projection.WorkflowVersionID,
		&projection.TargetKind,
		&projection.TargetURI,
		&projection.SummaryState,
		&payloadRaw,
		&projection.SourceEventID,
		&projection.SourceHash,
		&projection.WriteState,
		&projection.GeneratedAt,
		&writtenAt,
		&metadataRaw,
	); err != nil {
		return StatusProjectionRecord{}, fmt.Errorf("scan status projection: %w", err)
	}
	if err := json.Unmarshal(payloadRaw, &projection.Payload); err != nil {
		return StatusProjectionRecord{}, fmt.Errorf("parse status projection payload: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &projection.Metadata); err != nil {
		return StatusProjectionRecord{}, fmt.Errorf("parse status projection metadata: %w", err)
	}
	if writtenAt.Valid {
		projection.WrittenAt = &writtenAt.Time
	}
	return projection, nil
}

func (s Store) ProjectSummary(ctx context.Context, record Record) (ProjectSummary, error) {
	inventory, err := s.ImportInventory(ctx, record.ID)
	if err != nil {
		return ProjectSummary{}, err
	}

	summary := ProjectSummary{
		Project:   record,
		Inventory: inventory,
	}

	projectConfig, ok, err := s.ActiveProjectConfig(ctx, record.ID)
	if err != nil {
		return ProjectSummary{}, err
	}
	if ok {
		summary.Config = projectConfig
		summary.HasConfig = true
	}

	importSnapshot, err := s.LatestImportSnapshot(ctx, record.ID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return ProjectSummary{}, err
		}
	} else {
		summary.Import = importSnapshot
		summary.HasImport = true
	}

	recentImports, err := s.RecentImportSnapshots(ctx, record.ID, 2)
	if err != nil {
		return ProjectSummary{}, err
	}
	if len(recentImports) > 1 {
		summary.PreviousImport = recentImports[1]
		summary.HasPreviousImport = true
	}

	events, err := s.ListEvents(ctx, record.ID, 1)
	if err != nil {
		return ProjectSummary{}, err
	}
	summary.LatestEventCount = len(events)

	doctor, ok, err := s.LatestEventByType(ctx, record.ID, "project.doctor.completed")
	if err != nil {
		return ProjectSummary{}, err
	}
	if ok {
		summary.LatestDoctor = doctor
		summary.HasLatestDoctor = true
		summary.DoctorStatus = metadataString(doctor.Metadata, "overall_status")
		summary.DriftStatus = checkStatus(doctor.Metadata, "hash_drift")
		summary.ConfigDriftStatus = checkStatus(doctor.Metadata, "project_config_drift")
		summary.StageCoverageStatus = checkStatus(doctor.Metadata, "stage_coverage")
		summary.NativeDoctorStatus = checkStatus(doctor.Metadata, "native_workflow_doctor")
	}

	return summary, nil
}

func (s Store) ListProjectResiduals(ctx context.Context, record Record, limit int) ([]ResidualRecord, error) {
	if record.ID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), residual_key, status, type,
       COALESCE(title, ''), COALESCE(source_path, ''), COALESCE(current_impact, ''),
       executable_task, promotion_required, COALESCE(close_condition, ''), metadata,
       immutable, created_at, updated_at, imported_at
FROM residuals
WHERE project_id = $1
ORDER BY
  CASE
    WHEN status IN ('open', 'blocked', 'blocked-decision', 'blocked-external') THEN 0
    WHEN status IN ('accepted-exception', 'deferred') THEN 1
    ELSE 2
  END,
  updated_at DESC,
  id DESC
LIMIT $2`,
		record.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list project residuals: %w", err)
	}
	defer rows.Close()
	residuals := []ResidualRecord{}
	for rows.Next() {
		residual, err := scanResidualRecord(rows)
		if err != nil {
			return nil, err
		}
		residuals = append(residuals, residual)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project residuals: %w", err)
	}
	return residuals, nil
}

func (s Store) ListWorkflowVersionResiduals(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]ResidualRecord, error) {
	if record.ID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if version.ID <= 0 {
		return nil, fmt.Errorf("workflow version id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), residual_key, status, type,
       COALESCE(title, ''), COALESCE(source_path, ''), COALESCE(current_impact, ''),
       executable_task, promotion_required, COALESCE(close_condition, ''), metadata,
       immutable, created_at, updated_at, imported_at
FROM residuals
WHERE project_id = $1
  AND workflow_version_id = $2
ORDER BY
  CASE
    WHEN status IN ('open', 'blocked', 'blocked-decision', 'blocked-external') THEN 0
    WHEN status IN ('accepted-exception', 'deferred') THEN 1
    ELSE 2
  END,
  updated_at DESC,
  id DESC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow version residuals: %w", err)
	}
	defer rows.Close()
	residuals := []ResidualRecord{}
	for rows.Next() {
		residual, err := scanResidualRecord(rows)
		if err != nil {
			return nil, err
		}
		residuals = append(residuals, residual)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow version residuals: %w", err)
	}
	return residuals, nil
}

func ProjectReadinessFromSummary(summary ProjectSummary) ProjectReadiness {
	readiness := ProjectReadiness{
		Project: summary.Project,
		Status:  "pass",
		Summary: summary,
	}
	readiness.add("import_snapshot", statusBool(summary.HasImport), importSnapshotMessage(summary), map[string]any{
		"import_snapshots": summary.Inventory.ImportSnapshots,
		"source_hash":      summary.Import.SourceHash,
	})
	readiness.add("import_history", importHistoryStatus(summary), importHistoryMessage(summary), map[string]any{
		"import_snapshots":       summary.Inventory.ImportSnapshots,
		"previous_source_hash":   summary.PreviousImport.SourceHash,
		"latest_source_hash":     summary.Import.SourceHash,
		"history_ready_for_diff": summary.HasPreviousImport,
	})
	readiness.add("status_mirror", statusBool(summary.Inventory.MirrorExports > 0), statusMirrorMessage(summary), map[string]any{
		"mirror_exports": summary.Inventory.MirrorExports,
	})
	readiness.add("events_timeline", statusBool(summary.LatestEventCount > 0), eventsTimelineMessage(summary), map[string]any{
		"latest_event_count": summary.LatestEventCount,
	})
	readiness.add("summary_api_ready", "pass", "project summary can be built from AreaFlow state", map[string]any{
		"versions":  summary.Inventory.Versions,
		"residuals": summary.Inventory.Residuals,
		"artifacts": summary.Inventory.Artifacts,
	})
	readiness.add("project_config", statusBool(summary.HasConfig), projectConfigMessage(summary), map[string]any{
		"config_path":      summary.Config.ConfigPath,
		"config_hash":      summary.Config.ConfigHash,
		"protocol_version": summary.Config.ProtocolVersion,
		"ownership_mode":   metadataString(summary.Config.Ownership, "mode"),
		"migration_phase":  metadataString(summary.Config.Migration, "phase"),
	})
	readiness.add("doctor_report", statusPresent(summary.DoctorStatus), doctorReportMessage(summary), map[string]any{
		"doctor_status": summary.DoctorStatus,
	})
	readiness.add("drift_check", statusOrFail(summary.DriftStatus), driftCheckMessage(summary), map[string]any{
		"drift_status": summary.DriftStatus,
	})
	readiness.add("project_config_drift", statusOrFail(summary.ConfigDriftStatus), configDriftMessage(summary), map[string]any{
		"config_drift_status": summary.ConfigDriftStatus,
	})
	readiness.add("stage_coverage", statusOrFail(summary.StageCoverageStatus), stageCoverageMessage(summary), map[string]any{
		"stage_coverage_status": summary.StageCoverageStatus,
	})
	readiness.add("native_workflow_doctor", statusOrFail(summary.NativeDoctorStatus), nativeDoctorMessage(summary), map[string]any{
		"native_doctor_status": summary.NativeDoctorStatus,
	})
	return readiness
}

func evaluateCompatibilityCommand(summary ProjectSummary, spec compatibilityCommandSpec, permission CommandPermission) CompatibilityCommand {
	command := CompatibilityCommand{
		Command:        spec.Command,
		Mode:           "forward",
		Status:         "pass",
		AreaFlowTarget: spec.AreaFlowTarget,
		Fallback:       spec.Fallback,
		Metadata: map[string]any{
			"read_only":        spec.ReadOnly,
			"requires_native":  spec.RequiresNative,
			"mirror_exports":   summary.Inventory.MirrorExports,
			"native_allowed":   permission.CapabilityAllowed && permission.CommandAllowed && !permission.Denied,
			"command_allowed":  permission.CommandAllowed,
			"capability_allow": permission.CapabilityAllowed,
		},
	}
	if spec.Forbidden {
		command.Mode = "blocked"
		command.Status = "pass"
		command.Message = "command is intentionally blocked in v0.4 compatibility mode"
		command.BlockedReason = "execution and task-loop replacement are out of v0.4 scope"
		command.AreaFlowTarget = ""
		return command
	}
	if spec.Command == "./dev workflow init --version <version>" {
		command.Mode = "forward"
		command.Message = "create new workflow versions in AreaFlow; do not write managed project workflow directories"
		return command
	}
	if spec.RequiresNative && (!permission.CapabilityAllowed || !permission.CommandAllowed || permission.Denied) {
		command.Mode = "fallback_status"
		command.Status = statusBool(summary.Inventory.MirrorExports > 0)
		command.Message = "native command is not allowed; shim should show AreaFlow status fallback"
		command.BlockedReason = commandPermissionReason(permission)
		return command
	}
	if summary.Inventory.MirrorExports == 0 {
		command.Mode = "fallback_status"
		command.Status = "warn"
		command.Message = "status export is missing; shim can still query AreaFlow but offline fallback is not ready"
		command.BlockedReason = "missing .areaflow/status.json mirror export"
		return command
	}
	command.Message = "command can forward to AreaFlow and fall back to status export"
	return command
}

func commandPermissionReason(permission CommandPermission) string {
	if permission.Denied {
		return permission.Reason
	}
	if !permission.CapabilityAllowed {
		return "run_commands capability not allowed"
	}
	if !permission.CommandAllowed {
		return "command not allowed"
	}
	return "allowed"
}

func CompatibilityContractFromSummary(summary ProjectSummary, permissions map[string]CommandPermission) CompatibilityContract {
	contract := CompatibilityContract{
		Project:  summary.Project,
		Status:   "pass",
		Summary:  summary,
		Commands: make([]CompatibilityCommand, 0, len(compatibilityCommandSpecs)),
	}
	for _, spec := range compatibilityCommandSpecs {
		permission := permissions[spec.Command]
		command := evaluateCompatibilityCommand(summary, spec, permission)
		contract.Commands = append(contract.Commands, command)
		contract.Status = combineStatus(contract.Status, command.Status)
	}
	return contract
}

func (s Store) CompatibilityContract(ctx context.Context, record Record) (CompatibilityContract, error) {
	summary, err := s.ProjectSummary(ctx, record)
	if err != nil {
		return CompatibilityContract{}, err
	}
	permissions := make(map[string]CommandPermission, len(compatibilityCommandSpecs))
	for _, spec := range compatibilityCommandSpecs {
		permission, err := s.CommandPermission(ctx, record.ID, spec.Command)
		if err != nil {
			return CompatibilityContract{}, err
		}
		permissions[spec.Command] = permission
	}
	return CompatibilityContractFromSummary(summary, permissions), nil
}

func ShimPreviewFromCompatibility(contract CompatibilityContract) ShimPreview {
	preview := ShimPreview{
		Project:  contract.Project,
		Status:   contract.Status,
		Mode:     "read_only_planning",
		Contract: contract,
		PlannedFiles: []ShimFilePlan{
			{
				Path:     "scripts/areaflow_shim.py",
				Action:   "add",
				Required: false,
				Reason:   "recommended shared Python compatibility forwarding and offline fallback logic",
				Boundary: "read-only queries and allowed AreaFlow command forwarding only",
			},
			{
				Path:     "scripts/task_loop/console.py",
				Action:   "patch",
				Required: true,
				Reason:   "route interactive workflow menu actions through the shared shim decision path",
				Boundary: "keep non-workflow console behavior unchanged; no task execution or progress writes",
			},
			{
				Path:     "scripts/dev_tools/cli.py",
				Action:   "patch",
				Required: true,
				Reason:   "route workflow status, doctor, init and open through the shared Python shim",
				Boundary: "do not alter non-workflow dev commands or unrelated workflow subcommands",
			},
			{
				Path:     "scripts/task_loop/runner.py",
				Action:   "patch",
				Required: true,
				Reason:   "allow task-loop status fallback and keep task-loop run blocked before execution cutover",
				Boundary: "no task execution, no progress, log or checkpoint writes",
			},
			{
				Path:     "workflow/README.md",
				Action:   "manual_or_controlled_block",
				Required: false,
				Reason:   "human rough entry that links to AreaFlow and .areaflow/status.json",
				Boundary: "only after explicit AreaMatrix edit approval",
			},
			{
				Path:     ".areaflow/status.json",
				Action:   "controlled_projection_apply",
				Required: true,
				Reason:   "stable machine-readable fallback projection for shim offline mode",
				Boundary: "only through AreaFlow status-projection apply Command API with schema validation, expected-before preimage and rollback",
			},
		},
		DiscoveryOrder: []string{
			"AREAFLOW_API_URL",
			"AREAFLOW_BIN",
			"PATH areaflow",
			".areaflow/status.json",
		},
		ForbiddenPaths: []string{
			"workflow/versions/**",
			"workflow/versions/**/execution/**",
			"workflow/versions/**/execution/_shared/progress.json",
			"workflow/versions/v1-mvp/**",
			"tasks/active/**",
			"tasks/done/**",
			".codex/runtime/task-loop/**",
			"release evidence",
			"source code",
			"user files",
		},
		ForbiddenCommands: []string{
			"./task-loop run",
			"promotion apply",
			"write execution",
			"git checkpoint",
			"git reset --hard",
			"git checkout --",
			"rm -rf",
		},
		VerificationCommands: []string{
			"./dev workflow status",
			"./dev workflow doctor",
			"./dev workflow init --version shim-smoke",
			"./dev workflow open",
			"./task-loop status",
		},
		RollbackSteps: []string{
			"disable shim forwarding by unsetting AREAFLOW_API_URL or AREAFLOW_BIN",
			"fall back to .areaflow/status.json for read-only status",
			"revert only approved AreaMatrix shim files if needed",
			"do not delete AreaFlow events, audit_events, workflow_versions, runs, attempts or artifacts",
		},
		Notes: []string{
			"this preview is read-only and does not write AreaMatrix",
			"AreaMatrix shim implementation still requires explicit user approval",
			"execution cutover remains closed; task-loop run stays blocked",
		},
		CommandMappings: make([]ShimCommandMapping, 0, len(contract.Commands)),
	}
	for _, command := range contract.Commands {
		preview.CommandMappings = append(preview.CommandMappings, ShimCommandMapping{
			Command:        command.Command,
			Mode:           command.Mode,
			Status:         command.Status,
			AreaFlowTarget: command.AreaFlowTarget,
			Fallback:       command.Fallback,
			BlockedReason:  command.BlockedReason,
			ReadOnly:       metadataBool(command.Metadata, "read_only"),
			RequiresNative: metadataBool(command.Metadata, "requires_native"),
			Message:        command.Message,
		})
	}
	return preview
}

func (s Store) ShimPreview(ctx context.Context, record Record) (ShimPreview, error) {
	contract, err := s.CompatibilityContract(ctx, record)
	if err != nil {
		return ShimPreview{}, err
	}
	return ShimPreviewFromCompatibility(contract), nil
}

func ShimReadinessFromPreview(preview ShimPreview) ShimReadiness {
	return ShimReadinessFromPreviewWithEvidence(preview, map[string]EventRecord{})
}

func stableStatusProjectionRequiredFields() []string {
	return []string{
		"schema_version",
		"project_id",
		"project_name",
		"area_flow_url",
		"cutover_phase",
		"active_versions[].display_label",
		"active_versions[].lifecycle_status",
		"active_versions[].rough_progress.percent",
		"active_versions[].rough_progress.label",
		"active_versions[].rough_progress.blocked",
		"last_synced_at",
		"source_snapshot_hash",
		"compatibility.shim_lifecycle_state",
		"compatibility.offline_source",
		"compatibility.blocked_commands[]",
	}
}

func stableStatusProjectionForbiddenFields() []string {
	return []string{
		"summary",
		"generated_at",
		"source",
		"source_hash",
		"queue",
		"attempts",
		"logs",
		"checkpoint",
		"approval_payload",
		"secret",
		"worker_lease",
		"artifact_content",
	}
}

func stableStatusProjectionSchemaURI() string {
	return "schemas/status-projection.schema.json"
}

func stableStatusProjectionValidatorPreflight() string {
	return "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json"
}

func ShimReadinessFromPreviewWithEvidence(preview ShimPreview, evidence map[string]EventRecord) ShimReadiness {
	readiness := ShimReadiness{
		Project: preview.Project,
		Status:  "pass",
		Preview: preview,
	}
	readiness.add("compatibility_contract", statusForShimContract(preview.Contract.Status), "compatibility contract must be pass or accepted warn", map[string]any{
		"contract_status": preview.Contract.Status,
	})
	readiness.add("shim_preview", statusBool(len(preview.PlannedFiles) > 0 && len(preview.CommandMappings) > 0), "shim preview must include planned files and command mappings", map[string]any{
		"planned_files":    len(preview.PlannedFiles),
		"command_mappings": len(preview.CommandMappings),
		"mode":             preview.Mode,
	})
	readiness.add("status_projection", statusBool(preview.Contract.Summary.Inventory.MirrorExports > 0), "offline status projection should exist before AreaMatrix shim edit", map[string]any{
		"mirror_exports":         preview.Contract.Summary.Inventory.MirrorExports,
		"schema_contract":        "stable_fallback_projection_v1",
		"target_uri":             ".areaflow/status.json",
		"schema_uri":             stableStatusProjectionSchemaURI(),
		"validator_preflight":    stableStatusProjectionValidatorPreflight(),
		"required_schema_fields": stableStatusProjectionRequiredFields(),
		"forbidden_fields":       stableStatusProjectionForbiddenFields(),
	})
	readiness.add("task_loop_run_blocked", statusBool(shimCommandBlocked(preview, "./task-loop run")), "task-loop run must remain blocked before execution cutover", map[string]any{
		"command": "./task-loop run",
	})
	readiness.add("forbidden_paths_declared", statusBool(len(preview.ForbiddenPaths) > 0 && containsShimString(preview.ForbiddenPaths, "workflow/versions/**/execution/**")), "shim plan must declare execution paths forbidden", map[string]any{
		"forbidden_paths": preview.ForbiddenPaths,
	})
	readiness.add("rollback_plan_declared", statusBool(len(preview.RollbackSteps) > 0), "shim plan must include rollback steps", map[string]any{
		"rollback_steps": len(preview.RollbackSteps),
	})
	readiness.addItem(shimEvidenceReadinessItem("real_areamatrix_readonly_smoke", evidence["real_areamatrix_readonly_smoke"], "real AreaMatrix read-only smoke evidence is required before editing AreaMatrix", map[string]any{
		"required_script": "scripts/smoke-areamatrix-readonly.sh",
	}))
	readiness.addItem(shimEvidenceReadinessItem("real_areamatrix_status_projection_schema", evidence["real_areamatrix_status_projection_schema"], "real AreaMatrix .areaflow/status.json must validate against the stable schema before editing AreaMatrix", map[string]any{
		"schema_uri":          stableStatusProjectionSchemaURI(),
		"validator_preflight": stableStatusProjectionValidatorPreflight(),
		"managed_project":     "AreaMatrix",
	}))
	readiness.addItem(shimEvidenceReadinessItem("areamatrix_dirty_worktree_review", evidence["areamatrix_dirty_worktree_review"], "AreaMatrix dirty worktree must be reviewed before editing shim files", map[string]any{
		"managed_project": "AreaMatrix",
	}))
	readiness.add("explicit_edit_approval", "blocked", "explicit user approval is required before writing AreaMatrix shim files", map[string]any{
		"required_approval": "edit AreaMatrix shim files",
	})
	return readiness
}

func (s Store) ShimReadiness(ctx context.Context, record Record) (ShimReadiness, error) {
	preview, err := s.ShimPreview(ctx, record)
	if err != nil {
		return ShimReadiness{}, err
	}
	evidence, err := s.latestShimReadinessEvidence(ctx, record.ID)
	if err != nil {
		return ShimReadiness{}, err
	}
	return ShimReadinessFromPreviewWithEvidence(preview, evidence), nil
}

func ShimAuthorizationPacketFromReadiness(readiness ShimReadiness) ShimAuthorizationPacket {
	return ShimAuthorizationPacket{
		Project:         readiness.Project,
		Status:          "blocked",
		Mode:            "read_only_authorization_packet",
		Intent:          "authorize the minimal AreaMatrix compatibility shim edit after explicit user approval; no execution cutover, source write, engine call, checkpoint, secret resolution, repair, publish, or task-loop run forwarding is included",
		ReadinessStatus: readiness.Status,
		ReadinessItems:  append([]ShimReadinessItem{}, readiness.Items...),
		AllowedFiles:    shimAuthorizationAllowedFiles(readiness.Preview.PlannedFiles),
		ForbiddenPaths:  append([]string{}, readiness.Preview.ForbiddenPaths...),
		ForbiddenActions: append([]string{
			"workflow/versions/** writes",
			"execution writes",
			"progress.json writes",
			"task-loop run forwarding",
			"promotion apply",
			"git checkpoint",
			"git reset --hard",
			"git checkout --",
			"native doctor without explicit --allow-native",
			"source write",
			"user file write",
			"engine call",
			"secret resolution",
			"remote worker execution",
			"repair automation",
			"publish apply",
		}, readiness.Preview.ForbiddenCommands...),
		RequiredPreflight: []string{
			"areaflow project compatibility areamatrix --json",
			"areaflow project shim-preview areamatrix --json",
			"areaflow project shim-readiness areamatrix --json",
			"areaflow project shim-authorization areamatrix --json",
			"areaflow project status-projections areamatrix --json",
			"areaflow project status-projection-authorization areamatrix --json",
			"areaflow project status-projection-apply-packet areamatrix --json",
			"areaflow project status-projection-apply-gate areamatrix --json",
			stableStatusProjectionValidatorPreflight(),
			"verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash",
			"AREAFLOW_DATABASE_URL=... bash scripts/smoke-areamatrix-readonly.sh",
			"git -C /Users/as/Ai-Project/project/AreaMatrix status --short",
			"git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json",
		},
		PostEditVerification: []string{
			"cd /Users/as/Ai-Project/project/AreaMatrix",
			"./dev workflow status",
			"./dev workflow doctor",
			"./dev workflow init --version shim-smoke",
			"./dev workflow open",
			"./task-loop status",
			"verify ./task-loop run returns blocked and does not start legacy runner or write progress/log/checkpoint",
			"python3 /Users/as/Ai-Project/project/AreaFlow/scripts/validate-status-projection-schema.py /Users/as/Ai-Project/project/AreaFlow/schemas/status-projection.schema.json .areaflow/status.json",
			"git diff --check -- scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py workflow/README.md scripts/areaflow_shim.py .areaflow/status.json",
			"git status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json",
		},
		RollbackScope: []string{
			"disable shim forwarding by unsetting AREAFLOW_API_URL or AREAFLOW_BIN",
			"fall back to .areaflow/status.json for read-only status",
			"restore the captured preimage bytes for .areaflow/status.json if projection apply was part of the approved change",
			"revert only approved AreaMatrix shim files after approval",
			"do not delete AreaFlow events, audit_events, workflow_versions, runs, attempts or artifacts",
			"do not write v1 historical execution, progress.json, logs or checkpoints",
		},
		SafetyFacts: map[string]bool{
			"project_write_attempted":       false,
			"execution_write_attempted":     false,
			"task_loop_run_forwarded":       false,
			"status_projection_write_open":  false,
			"engine_call_attempted":         false,
			"commands_run":                  false,
			"secrets_resolved":              false,
			"network_used":                  false,
			"area_matrix_files_modified":    false,
			"area_matrix_execution_touched": false,
		},
		NextRequiredApproval: "explicit user approval to edit only the listed AreaMatrix shim files",
	}
}

func shimAuthorizationAllowedFiles(files []ShimFilePlan) []ShimFilePlan {
	allowed := []ShimFilePlan{}
	for _, file := range files {
		if file.Path == ".areaflow/status.json" || file.Action == "controlled_projection_apply" {
			continue
		}
		allowed = append(allowed, file)
	}
	return allowed
}

func (s Store) ShimAuthorizationPacket(ctx context.Context, record Record) (ShimAuthorizationPacket, error) {
	readiness, err := s.ShimReadiness(ctx, record)
	if err != nil {
		return ShimAuthorizationPacket{}, err
	}
	return ShimAuthorizationPacketFromReadiness(readiness), nil
}

func shimEvidenceReadinessItem(key string, event EventRecord, blockedMessage string, metadata map[string]any) ShimReadinessItem {
	itemMetadata := map[string]any{}
	for metadataKey, value := range metadata {
		itemMetadata[metadataKey] = value
	}
	if event.ID == 0 {
		itemMetadata["evidence_recorded"] = false
		return ShimReadinessItem{
			Key:      key,
			Status:   "blocked",
			Message:  blockedMessage,
			Metadata: itemMetadata,
		}
	}
	itemMetadata["evidence_recorded"] = true
	itemMetadata["evidence_event_id"] = event.ID
	itemMetadata["evidence_status"] = metadataString(event.Metadata, "evidence_status")
	itemMetadata["evidence_uri"] = metadataString(event.Metadata, "evidence_uri")
	itemMetadata["summary"] = metadataString(event.Metadata, "summary")
	itemMetadata["recorded_at"] = event.CreatedAt
	if metadataString(event.Metadata, "evidence_status") != "pass" {
		return ShimReadinessItem{
			Key:      key,
			Status:   "blocked",
			Message:  blockedMessage,
			Metadata: itemMetadata,
		}
	}
	return ShimReadinessItem{
		Key:      key,
		Status:   "pass",
		Message:  "required shim readiness evidence has been recorded",
		Metadata: itemMetadata,
	}
}

func (s Store) latestShimReadinessEvidence(ctx context.Context, projectID int64) (map[string]EventRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE project_id = $1 AND event_type = 'project.shim_readiness_evidence.recorded'
ORDER BY created_at DESC, id DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list shim readiness evidence events: %w", err)
	}
	defer rows.Close()
	events, err := scanEventRows(rows)
	if err != nil {
		return nil, err
	}
	latest := map[string]EventRecord{}
	for _, event := range events {
		key := metadataString(event.Metadata, "evidence_key")
		if key == "" {
			continue
		}
		if _, exists := latest[key]; !exists {
			latest[key] = event
		}
	}
	return latest, nil
}

func (r *ShimReadiness) add(key string, status string, message string, metadata map[string]any) {
	r.addItem(ShimReadinessItem{Key: key, Status: status, Message: message, Metadata: metadata})
}

func (r *ShimReadiness) addItem(item ShimReadinessItem) {
	r.Items = append(r.Items, ShimReadinessItem{
		Key:      item.Key,
		Status:   item.Status,
		Message:  item.Message,
		Metadata: item.Metadata,
	})
	r.Status = combineShimReadinessStatus(r.Status, item.Status)
}

func combineShimReadinessStatus(current string, next string) string {
	if current == "blocked" || next == "blocked" {
		return "blocked"
	}
	if current == "fail" || next == "fail" {
		return "fail"
	}
	if current == "warn" || next == "warn" {
		return "warn"
	}
	return "pass"
}

func statusForShimContract(status string) string {
	switch status {
	case "pass", "warn":
		return status
	default:
		return "fail"
	}
}

func shimCommandBlocked(preview ShimPreview, command string) bool {
	for _, mapping := range preview.CommandMappings {
		if mapping.Command == command {
			return mapping.Mode == "blocked" && mapping.Status == "pass"
		}
	}
	return false
}

func containsShimString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (s Store) ProjectReadiness(ctx context.Context, record Record) (ProjectReadiness, error) {
	summary, err := s.ProjectSummary(ctx, record)
	if err != nil {
		return ProjectReadiness{}, err
	}
	return ProjectReadinessFromSummary(summary), nil
}

func ProjectImportDiffFromSummary(summary ProjectSummary) ProjectImportDiff {
	diff := ProjectImportDiff{
		Project:       summary.Project,
		Status:        "no_previous",
		HasPrevious:   summary.HasPreviousImport,
		Latest:        summary.Import,
		Previous:      summary.PreviousImport,
		SourceChanged: summary.HasPreviousImport && summary.Import.SourceHash != summary.PreviousImport.SourceHash,
	}
	if !summary.HasImport {
		diff.Status = "no_import"
		return diff
	}
	if !summary.HasPreviousImport {
		return diff
	}

	diff.add("source_hash", summary.PreviousImport.SourceHash, summary.Import.SourceHash)
	diff.add("version_count", snapshotValue(summary.PreviousImport, "version_count"), snapshotValue(summary.Import, "version_count"))
	diff.add("residual_count", snapshotValue(summary.PreviousImport, "residual_count"), snapshotValue(summary.Import, "residual_count"))
	diff.add("tasks.active", nestedSnapshotValue(summary.PreviousImport, "tasks", "active"), nestedSnapshotValue(summary.Import, "tasks", "active"))
	diff.add("tasks.done", nestedSnapshotValue(summary.PreviousImport, "tasks", "done"), nestedSnapshotValue(summary.Import, "tasks", "done"))
	diff.add("tasks.backlog_open", nestedSnapshotValue(summary.PreviousImport, "tasks", "backlog_open"), nestedSnapshotValue(summary.Import, "tasks", "backlog_open"))
	diff.add("tasks.backlog_closed", nestedSnapshotValue(summary.PreviousImport, "tasks", "backlog_closed"), nestedSnapshotValue(summary.Import, "tasks", "backlog_closed"))
	diff.add("v1_execution.done", nestedSnapshotValue(summary.PreviousImport, "v1_execution", "done"), nestedSnapshotValue(summary.Import, "v1_execution", "done"))
	diff.add("v1_execution.total", nestedSnapshotValue(summary.PreviousImport, "v1_execution", "total"), nestedSnapshotValue(summary.Import, "v1_execution", "total"))

	diff.Status = "unchanged"
	for _, change := range diff.Changes {
		if change.Status == "changed" {
			diff.Status = "changed"
			break
		}
	}
	return diff
}

func (s Store) ProjectImportDiff(ctx context.Context, record Record) (ProjectImportDiff, error) {
	summary, err := s.ProjectSummary(ctx, record)
	if err != nil {
		return ProjectImportDiff{}, err
	}
	return ProjectImportDiffFromSummary(summary), nil
}

func ProjectVerificationBundleFromParts(summary ProjectSummary, readiness ProjectReadiness, diff ProjectImportDiff, events []EventRecord) ProjectVerificationBundle {
	status := readiness.Status
	if diff.Status == "changed" {
		status = combineStatus(status, "warn")
	}
	if diff.Status == "no_import" {
		status = combineStatus(status, "fail")
	}
	if len(events) == 0 {
		status = combineStatus(status, "warn")
	}
	phaseGate := EvaluateV02PhaseGate(summary, readiness, diff, events)
	return ProjectVerificationBundle{
		Project:    summary.Project,
		Status:     status,
		PhaseGate:  phaseGate,
		Summary:    summary,
		Readiness:  readiness,
		ImportDiff: diff,
		Events:     events,
	}
}

func EvaluateV02PhaseGate(summary ProjectSummary, readiness ProjectReadiness, diff ProjectImportDiff, events []EventRecord) PhaseGate {
	gate := PhaseGate{
		Name:   "v0.2-shadow-doctor",
		Status: "pass",
	}
	requireGate(&gate, summary.HasImport, "missing import snapshot")
	requireGate(&gate, summary.HasPreviousImport, "missing previous import snapshot for diff history")
	requireGate(&gate, summary.Inventory.MirrorExports > 0, "missing status mirror export")
	requireGate(&gate, len(events) > 0, "missing project event timeline")
	requireGate(&gate, summary.DoctorStatus != "", "missing doctor report")
	requireGate(&gate, summary.DriftStatus == "pass", "hash drift check is not pass")
	if summary.ConfigDriftStatus == "warn" {
		gate.AcceptedWarnings = append(gate.AcceptedWarnings, "project config file drift requires project add refresh")
	} else {
		requireGate(&gate, summary.ConfigDriftStatus == "pass", "project config drift check is not pass or accepted warn")
	}
	requireGate(&gate, summary.StageCoverageStatus == "pass", "stage coverage is not pass")
	requireGate(&gate, diff.Status == "unchanged", "latest import diff is not unchanged")
	if summary.NativeDoctorStatus == "warn" {
		gate.AcceptedWarnings = append(gate.AcceptedWarnings, "native workflow doctor skipped or warned by permission gate")
	} else {
		requireGate(&gate, summary.NativeDoctorStatus == "pass", "native workflow doctor is not pass or accepted warn")
	}
	if len(gate.Blockers) > 0 {
		gate.Status = "blocked"
	}
	return gate
}

func requireGate(gate *PhaseGate, ok bool, blocker string) {
	if ok {
		return
	}
	gate.Blockers = append(gate.Blockers, blocker)
}

func (s Store) ProjectVerificationBundle(ctx context.Context, record Record, eventLimit int) (ProjectVerificationBundle, error) {
	summary, err := s.ProjectSummary(ctx, record)
	if err != nil {
		return ProjectVerificationBundle{}, err
	}
	readiness := ProjectReadinessFromSummary(summary)
	diff := ProjectImportDiffFromSummary(summary)
	events, err := s.ListEvents(ctx, record.ID, eventLimit)
	if err != nil {
		return ProjectVerificationBundle{}, err
	}
	return ProjectVerificationBundleFromParts(summary, readiness, diff, events), nil
}

func ProjectCutoverReadinessFromParts(verification ProjectVerificationBundle, compatibility CompatibilityContract, version WorkflowVersion, gates []GateResult) ProjectCutoverReadiness {
	readiness := ProjectCutoverReadiness{
		Project:       verification.Project,
		Version:       version,
		Status:        "pass",
		Verification:  verification,
		Compatibility: compatibility,
		Gates:         gates,
	}
	readiness.add("verification_bundle", cutoverStatusFromPhaseGate(verification.PhaseGate), cutoverVerificationMessage(verification), map[string]any{
		"bundle_status": verification.Status,
		"phase_gate":    verification.PhaseGate.Name,
		"blockers":      verification.PhaseGate.Blockers,
	})
	readiness.add("status_mirror", statusBool(verification.Summary.Inventory.MirrorExports > 0), statusMirrorMessage(verification.Summary), map[string]any{
		"mirror_exports": verification.Summary.Inventory.MirrorExports,
	})
	readiness.add("compatibility_contract", compatibility.Status, cutoverCompatibilityMessage(compatibility), map[string]any{
		"commands": len(compatibility.Commands),
	})
	readiness.add("workflow_version_authored", statusBool(version.ImportMode == "authored"), cutoverVersionMessage(version), map[string]any{
		"display_label":    version.DisplayLabel,
		"lifecycle_status": version.LifecycleStatus,
		"import_mode":      version.ImportMode,
		"immutable":        version.Immutable,
	})
	readiness.add("approval_gate", cutoverLatestGateStatus(gates, "approval_gate"), cutoverGateMessage(gates, "approval_gate"), cutoverGateMetadata(gates, "approval_gate"))
	readiness.add("live_mapping_gate", cutoverLatestGateStatus(gates, "live_mapping_gate"), cutoverGateMessage(gates, "live_mapping_gate"), cutoverGateMetadata(gates, "live_mapping_gate"))
	readiness.add("rollback_plan", "pass", "rollback plan is documented as append-only soft/hard rollback in migration docs", map[string]any{
		"source": "docs/history/v1.0/migrations/cutover-rollback-compat.md",
	})
	readiness.PhaseGate = EvaluateCutoverPhaseGate(readiness)
	readiness.Status = readiness.PhaseGate.Status
	return readiness
}

func EvaluateCutoverPhaseGate(readiness ProjectCutoverReadiness) PhaseGate {
	gate := PhaseGate{
		Name:   "v0.4-cutover-readiness",
		Status: "pass",
	}
	for _, item := range readiness.Items {
		if item.Status == "pass" {
			continue
		}
		if item.Status == "warn" && item.Key == "compatibility_contract" {
			gate.AcceptedWarnings = append(gate.AcceptedWarnings, "compatibility contract has warnings; shim must fall back to status export where needed")
			continue
		}
		gate.Blockers = append(gate.Blockers, fmt.Sprintf("%s is %s: %s", item.Key, item.Status, item.Message))
	}
	if len(gate.Blockers) > 0 {
		gate.Status = "blocked"
	}
	return gate
}

func (s Store) ProjectCutoverReadiness(ctx context.Context, record Record, label string, eventLimit int) (ProjectCutoverReadiness, error) {
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return ProjectCutoverReadiness{}, err
	}
	verification, err := s.ProjectVerificationBundle(ctx, record, eventLimit)
	if err != nil {
		return ProjectCutoverReadiness{}, err
	}
	compatibility, err := s.CompatibilityContract(ctx, record)
	if err != nil {
		return ProjectCutoverReadiness{}, err
	}
	gates, err := s.ListGateResults(ctx, record, version, 20)
	if err != nil {
		return ProjectCutoverReadiness{}, err
	}
	return ProjectCutoverReadinessFromParts(verification, compatibility, version, gates), nil
}

func (d *ProjectImportDiff) add(key string, previous string, latest string) {
	status := "unchanged"
	if previous != latest {
		status = "changed"
	}
	d.Changes = append(d.Changes, ImportDiffChange{
		Key:      key,
		Status:   status,
		Previous: previous,
		Latest:   latest,
	})
}

func (r *ProjectReadiness) add(key string, status string, message string, metadata map[string]any) {
	r.Items = append(r.Items, ReadinessItem{
		Key:      key,
		Status:   status,
		Message:  message,
		Metadata: metadata,
	})
	r.Status = combineStatus(r.Status, status)
}

func (r *ProjectCutoverReadiness) add(key string, status string, message string, metadata map[string]any) {
	r.Items = append(r.Items, ReadinessItem{
		Key:      key,
		Status:   status,
		Message:  message,
		Metadata: metadata,
	})
	r.Status = combineStatus(r.Status, status)
}

func combineStatus(current string, next string) string {
	if current == "fail" || next == "fail" {
		return "fail"
	}
	if current == "warn" || next == "warn" {
		return "warn"
	}
	return "pass"
}

func statusBool(ok bool) string {
	if ok {
		return "pass"
	}
	return "warn"
}

func statusPresent(status string) string {
	if status == "" {
		return "warn"
	}
	return statusOrFail(status)
}

func statusOrFail(status string) string {
	switch status {
	case "pass", "warn", "fail":
		return status
	case "":
		return "warn"
	default:
		return "warn"
	}
}

func importSnapshotMessage(summary ProjectSummary) string {
	if summary.HasImport {
		return "latest import snapshot is available"
	}
	return "no import snapshot found; run project import first"
}

func importHistoryStatus(summary ProjectSummary) string {
	if summary.HasPreviousImport {
		return "pass"
	}
	if summary.HasImport {
		return "warn"
	}
	return "warn"
}

func importHistoryMessage(summary ProjectSummary) string {
	if summary.HasPreviousImport {
		return "at least two import snapshots are available for future diff checks"
	}
	if summary.HasImport {
		return "only one import snapshot is available; run import again to build diff history"
	}
	return "no import history found"
}

func statusMirrorMessage(summary ProjectSummary) string {
	if summary.Inventory.MirrorExports > 0 {
		return "status mirror export has been recorded"
	}
	return "status mirror export has not been recorded yet"
}

func projectConfigMessage(summary ProjectSummary) string {
	if summary.HasConfig {
		return "active areaflow.yaml config snapshot is persisted"
	}
	return "active areaflow.yaml config snapshot is missing"
}

func eventsTimelineMessage(summary ProjectSummary) string {
	if summary.LatestEventCount > 0 {
		return "project event timeline is available"
	}
	return "project event timeline has no events yet"
}

func doctorReportMessage(summary ProjectSummary) string {
	if summary.DoctorStatus == "" {
		return "no doctor report found; run project doctor first"
	}
	return "latest doctor report is available"
}

func driftCheckMessage(summary ProjectSummary) string {
	if summary.DriftStatus == "" {
		return "drift check has not been recorded yet"
	}
	return "latest doctor report includes drift status"
}

func configDriftMessage(summary ProjectSummary) string {
	if summary.ConfigDriftStatus == "" {
		return "project config drift check has not been recorded yet"
	}
	if summary.ConfigDriftStatus == "warn" {
		return "project config file differs from active AreaFlow config snapshot"
	}
	return "latest doctor report includes project config drift status"
}

func stageCoverageMessage(summary ProjectSummary) string {
	if summary.StageCoverageStatus == "" {
		return "stage coverage has not been recorded yet"
	}
	return "latest doctor report includes stage coverage status"
}

func nativeDoctorMessage(summary ProjectSummary) string {
	if summary.NativeDoctorStatus == "" {
		return "native workflow doctor has not been recorded yet"
	}
	if summary.NativeDoctorStatus == "warn" {
		return "native workflow doctor is warning or skipped; this is acceptable without explicit native authorization"
	}
	return "latest doctor report includes native workflow doctor status"
}

func cutoverStatusFromPhaseGate(gate PhaseGate) string {
	if gate.Status == "pass" {
		return "pass"
	}
	return "blocked"
}

func cutoverVerificationMessage(bundle ProjectVerificationBundle) string {
	if bundle.PhaseGate.Status == "pass" {
		return "v0.2 verification bundle passed"
	}
	if len(bundle.PhaseGate.Blockers) == 0 {
		return "v0.2 verification bundle is not pass"
	}
	return "v0.2 verification bundle has blockers"
}

func cutoverCompatibilityMessage(contract CompatibilityContract) string {
	if contract.Status == "pass" {
		return "compatibility contract is ready"
	}
	if contract.Status == "warn" {
		return "compatibility contract has fallback warnings"
	}
	return "compatibility contract is not ready"
}

func cutoverVersionMessage(version WorkflowVersion) string {
	if version.ImportMode == "authored" {
		return "workflow version is authored by AreaFlow"
	}
	if version.DisplayLabel == "" {
		return "workflow version is missing"
	}
	return "workflow version is not authored by AreaFlow"
}

func cutoverLatestGateStatus(gates []GateResult, gateName string) string {
	gate, ok := latestGateFromList(gates, gateName)
	if !ok {
		return "blocked"
	}
	if gate.Status == "pass" {
		return "pass"
	}
	return "blocked"
}

func cutoverGateMessage(gates []GateResult, gateName string) string {
	gate, ok := latestGateFromList(gates, gateName)
	if !ok {
		return fmt.Sprintf("%s has not been run", gateName)
	}
	if gate.Status == "pass" {
		return fmt.Sprintf("%s passed", gateName)
	}
	return fmt.Sprintf("%s status is %s", gateName, gate.Status)
}

func cutoverGateMetadata(gates []GateResult, gateName string) map[string]any {
	gate, ok := latestGateFromList(gates, gateName)
	if !ok {
		return map[string]any{
			"gate_name": gateName,
			"found":     false,
		}
	}
	return map[string]any{
		"gate_name":  gateName,
		"found":      true,
		"status":     gate.Status,
		"checked_at": gate.CheckedAt,
		"id":         gate.ID,
		"failures":   gate.Failures,
		"warnings":   gate.Warnings,
	}
}

func latestGateFromList(gates []GateResult, gateName string) (GateResult, bool) {
	for _, gate := range gates {
		if gate.GateName == gateName {
			return gate, true
		}
	}
	return GateResult{}, false
}

func (s Store) CanWritePath(ctx context.Context, projectID int64, capability string, path string) (bool, string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT effect, capability, resource_type, pattern
FROM project_permissions
WHERE project_id = $1 AND resource_type IN ('capability', 'path')
ORDER BY id`,
		projectID,
	)
	if err != nil {
		return false, "", fmt.Errorf("load project permissions: %w", err)
	}
	defer rows.Close()

	capabilityAllowed := false
	pathAllowed := false
	for rows.Next() {
		var effect, permissionCapability, resourceType, pattern string
		if err := rows.Scan(&effect, &permissionCapability, &resourceType, &pattern); err != nil {
			return false, "", fmt.Errorf("scan project permission: %w", err)
		}
		if effect == "deny" && resourceType == "path" && globMatch(pattern, path) {
			return false, "path denied by forbidden path", nil
		}
		if resourceType == "capability" && permissionCapability == capability && effect == "allow" {
			capabilityAllowed = true
		}
		if resourceType == "path" && permissionCapability == capability && effect == "allow" && globMatch(pattern, path) {
			pathAllowed = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", fmt.Errorf("iterate project permissions: %w", err)
	}
	if !capabilityAllowed {
		return false, "capability not allowed", nil
	}
	if !pathAllowed {
		return false, "path not allowed", nil
	}
	return true, "allowed", nil
}

func (s Store) CanRunCommand(ctx context.Context, projectID int64, command string) (bool, string, error) {
	permission, err := s.CommandPermission(ctx, projectID, command)
	if err != nil {
		return false, "", err
	}
	if permission.Denied {
		return false, permission.Reason, nil
	}
	if !permission.CapabilityAllowed {
		return false, "run_commands capability not allowed", nil
	}
	if !permission.CommandAllowed {
		return false, "command not allowed", nil
	}
	return true, "allowed", nil
}

func (s Store) CommandPermission(ctx context.Context, projectID int64, command string) (CommandPermission, error) {
	rows, err := s.pool.Query(ctx, `
SELECT effect, capability, resource_type, pattern
FROM project_permissions
WHERE project_id = $1 AND resource_type IN ('capability', 'command')
ORDER BY id`,
		projectID,
	)
	if err != nil {
		return CommandPermission{}, fmt.Errorf("load project command permissions: %w", err)
	}
	defer rows.Close()

	permission := CommandPermission{}
	for rows.Next() {
		var effect, capability, resourceType, pattern string
		if err := rows.Scan(&effect, &capability, &resourceType, &pattern); err != nil {
			return CommandPermission{}, fmt.Errorf("scan project command permission: %w", err)
		}
		if effect == "deny" && resourceType == "command" && globMatch(pattern, command) {
			permission.Denied = true
			permission.Reason = "command denied by forbidden command"
			return permission, nil
		}
		if resourceType == "capability" && capability == "run_commands" && effect == "allow" {
			permission.CapabilityAllowed = true
		}
		if resourceType == "command" && capability == "run_commands" && effect == "allow" && globMatch(pattern, command) {
			permission.CommandAllowed = true
		}
	}
	if err := rows.Err(); err != nil {
		return CommandPermission{}, fmt.Errorf("iterate project command permissions: %w", err)
	}
	return permission, nil
}

func (s Store) RecordStatusExport(ctx context.Context, projectID int64, path string, summary map[string]any, sourceHash string) error {
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshal status export summary: %w", err)
	}
	idempotencyKey := statusProjectionWriteIdempotencyKey(path, sourceHash, summaryJSON)
	requestHash, err := statusProjectionWriteRequestHash(path, summaryJSON, sourceHash)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin status export record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, projectID, "project.status_projection.write", idempotencyKey, requestHash)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	var snapshotID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO project_status_snapshots (project_id, snapshot_kind, summary, source_hash, export_path)
VALUES ($1, 'mirror_export', $2::jsonb, $3, $4)
RETURNING id`,
		projectID,
		string(summaryJSON),
		sourceHash,
		path,
	).Scan(&snapshotID); err != nil {
		return fmt.Errorf("insert mirror export snapshot: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'status.export', 'write_status', 'path', $2, 'allowed', 'explicit status mirror export', $3::jsonb)
RETURNING id`,
		projectID,
		path,
		string(summaryJSON),
	).Scan(&auditEventID); err != nil {
		return fmt.Errorf("insert status export audit event: %w", err)
	}
	var projectionID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO status_projections (
    project_id,
    target_kind,
    target_uri,
    summary_state,
    payload_json,
    source_hash,
    write_state,
    written_at,
    metadata
)
VALUES ($1, 'project_status_json', $2, $3, $4::jsonb, $5, 'written', now(), $6::jsonb)
RETURNING id`,
		projectID,
		path,
		statusProjectionSummaryState(summary),
		string(summaryJSON),
		sourceHash,
		`{"legacy_snapshot_kind":"mirror_export","write_boundary":"status_export","command_type":"project.status_projection.write"}`,
	).Scan(&projectionID); err != nil {
		return fmt.Errorf("insert status projection: %w", err)
	}
	if err := completeCommandRequestResponse(ctx, tx, projectID, "project.status_projection.write", idempotencyKey, map[string]any{
		"snapshot_id":          snapshotID,
		"audit_event_id":       auditEventID,
		"status_projection_id": projectionID,
		"target_kind":          "project_status_json",
		"target_uri":           path,
		"source_hash":          sourceHash,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit status export record: %w", err)
	}
	return nil
}

func statusProjectionWriteIdempotencyKey(path string, sourceHash string, summaryJSON []byte) string {
	path = strings.TrimSpace(path)
	sourceHash = strings.TrimSpace(sourceHash)
	if path == "" {
		path = "unknown-target"
	}
	if sourceHash == "" {
		sourceHash = "no-source-hash"
	}
	return fmt.Sprintf("project.status_projection.write:%s:%s:%s", path, sourceHash, shortSHA256Hex(summaryJSON))
}

func statusProjectionWriteRequestHash(path string, summaryJSON []byte, sourceHash string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": "project.status_projection.write",
		"target_kind":  "project_status_json",
		"target_uri":   strings.TrimSpace(path),
		"source_hash":  strings.TrimSpace(sourceHash),
		"payload":      json.RawMessage(summaryJSON),
	})
	if err != nil {
		return "", fmt.Errorf("marshal status projection command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func shortSHA256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:8])
}

func statusProjectionSummaryState(summary map[string]any) string {
	if summary == nil {
		return "mirroring"
	}
	if status, ok := summary["summary_state"].(string); ok && strings.TrimSpace(status) != "" {
		return strings.TrimSpace(status)
	}
	return "mirroring"
}

func (s Store) RecordDoctorReport(ctx context.Context, projectID int64, summary map[string]any, options RecordDoctorReportOptions) (RecordDoctorReportResult, error) {
	options = normalizeRecordDoctorReportOptions(options)
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return RecordDoctorReportResult{}, fmt.Errorf("marshal doctor report summary: %w", err)
	}
	requestHash, err := doctorReportRequestHash(projectID, summaryJSON, options)
	if err != nil {
		return RecordDoctorReportResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = doctorReportIdempotencyKey(projectID, summaryJSON)
	}

	severity := "info"
	if status, ok := summary["overall_status"].(string); ok {
		switch status {
		case "fail":
			severity = "error"
		case "warn":
			severity = "warning"
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RecordDoctorReportResult{}, fmt.Errorf("begin doctor report record: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, projectID, "project.doctor.record", options.IdempotencyKey, requestHash)
	if err != nil {
		return RecordDoctorReportResult{}, err
	}
	if !created {
		result, err := loadDoctorReportByCommandResponse(ctx, tx, projectID, options.IdempotencyKey)
		if err != nil {
			return RecordDoctorReportResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return RecordDoctorReportResult{}, fmt.Errorf("commit idempotent doctor report record: %w", err)
		}
		result.IdempotencyKey = options.IdempotencyKey
		result.Created = false
		return result, nil
	}

	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, 'project.doctor.completed', $2, 'AreaFlow project doctor completed', $3::jsonb)
RETURNING id`,
		projectID,
		severity,
		string(summaryJSON),
	).Scan(&eventID); err != nil {
		return RecordDoctorReportResult{}, fmt.Errorf("insert doctor report event: %w", err)
	}
	if err := completeCommandRequestResponse(ctx, tx, projectID, "project.doctor.record", options.IdempotencyKey, map[string]any{
		"event_id":       eventID,
		"severity":       severity,
		"overall_status": metadataString(summary, "overall_status"),
	}); err != nil {
		return RecordDoctorReportResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RecordDoctorReportResult{}, fmt.Errorf("commit doctor report record: %w", err)
	}
	return RecordDoctorReportResult{
		EventID:        eventID,
		Severity:       severity,
		OverallStatus:  metadataString(summary, "overall_status"),
		IdempotencyKey: options.IdempotencyKey,
		Created:        true,
	}, nil
}

func normalizeRecordDoctorReportOptions(options RecordDoctorReportOptions) RecordDoctorReportOptions {
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "record project doctor report"
	}
	return options
}

func doctorReportRequestHash(projectID int64, summaryJSON []byte, options RecordDoctorReportOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": "project.doctor.record",
		"project_id":   projectID,
		"summary":      json.RawMessage(summaryJSON),
		"actor":        options.Actor,
		"reason":       options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal doctor report command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func doctorReportIdempotencyKey(projectID int64, summaryJSON []byte) string {
	sequence := doctorReportDefaultKeySequence.Add(1)
	return fmt.Sprintf("project.doctor.record:%d:%d:%d:%s", projectID, time.Now().UTC().UnixNano(), sequence, shortSHA256Hex(summaryJSON))
}

func loadDoctorReportByCommandResponse(ctx context.Context, tx pgx.Tx, projectID int64, idempotencyKey string) (RecordDoctorReportResult, error) {
	response, err := loadCommandResponse(ctx, tx, projectID, "project.doctor.record", idempotencyKey)
	if err != nil {
		return RecordDoctorReportResult{}, err
	}
	return RecordDoctorReportResult{
		EventID:       metadataInt64(response, "event_id"),
		Severity:      metadataString(response, "severity"),
		OverallStatus: metadataString(response, "overall_status"),
	}, nil
}

func loadCommandResponse(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string) (map[string]any, error) {
	var responseRaw []byte
	var completedAt sql.NullTime
	if err := tx.QueryRow(ctx, `
SELECT response, completed_at
FROM command_requests
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		projectID,
		commandType,
		idempotencyKey,
	).Scan(&responseRaw, &completedAt); err != nil {
		return nil, fmt.Errorf("load command response: %w", err)
	}
	if !completedAt.Valid {
		return nil, fmt.Errorf("command request is not complete: %s", idempotencyKey)
	}
	response := map[string]any{}
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		return nil, fmt.Errorf("parse command response: %w", err)
	}
	return response, nil
}

func (s Store) ListEvents(ctx context.Context, projectID int64, limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE project_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		projectID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list project events: %w", err)
	}
	defer rows.Close()

	return scanEventRows(rows)
}

func (s Store) ListAuditEvents(ctx context.Context, projectID int64, limit int) ([]AuditEventRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(actor_id, 0), action,
       COALESCE(capability, ''), COALESCE(resource_type, ''), COALESCE(resource, ''),
       decision, COALESCE(reason, ''), metadata, created_at
FROM audit_events
WHERE ($1::bigint = 0 OR project_id = $1)
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		projectID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := []AuditEventRecord{}
	for rows.Next() {
		var event AuditEventRecord
		var raw []byte
		if err := rows.Scan(
			&event.ID,
			&event.ProjectID,
			&event.ActorID,
			&event.Action,
			&event.Capability,
			&event.ResourceType,
			&event.Resource,
			&event.Decision,
			&event.Reason,
			&raw,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		if err := json.Unmarshal(raw, &event.Metadata); err != nil {
			return nil, fmt.Errorf("parse audit event metadata: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return events, nil
}

func scanEventRows(rows pgx.Rows) ([]EventRecord, error) {
	events := []EventRecord{}
	for rows.Next() {
		var event EventRecord
		var raw []byte
		if err := rows.Scan(
			&event.ID,
			&event.ProjectID,
			&event.RunID,
			&event.WorkflowVersionID,
			&event.Type,
			&event.Severity,
			&event.Message,
			&raw,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project event: %w", err)
		}
		if err := json.Unmarshal(raw, &event.Metadata); err != nil {
			return nil, fmt.Errorf("parse project event metadata: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project events: %w", err)
	}
	return events, nil
}

func (s Store) ListEventStream(ctx context.Context, filter EventStreamFilter) ([]EventRecord, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.AfterID > 0 {
		return s.listEventsAfter(ctx, filter)
	}
	return s.listLatestEventsAscending(ctx, filter)
}

func (s Store) listEventsAfter(ctx context.Context, filter EventStreamFilter) ([]EventRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE ($1::bigint = 0 OR project_id = $1)
  AND ($2::bigint = 0 OR run_id = $2)
  AND id > $3
ORDER BY id ASC
LIMIT $4`,
		filter.ProjectID,
		filter.RunID,
		filter.AfterID,
		filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list event stream after id: %w", err)
	}
	defer rows.Close()
	return scanEventRows(rows)
}

func (s Store) listLatestEventsAscending(ctx context.Context, filter EventStreamFilter) ([]EventRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, run_id, workflow_version_id, event_type, severity, message, metadata, created_at
FROM (
  SELECT id, COALESCE(project_id, 0) AS project_id, COALESCE(run_id, 0) AS run_id,
         COALESCE(workflow_version_id, 0) AS workflow_version_id,
         event_type, severity, message, metadata, created_at
  FROM events
  WHERE ($1::bigint = 0 OR project_id = $1)
    AND ($2::bigint = 0 OR run_id = $2)
  ORDER BY id DESC
  LIMIT $3
) latest
ORDER BY id ASC`,
		filter.ProjectID,
		filter.RunID,
		filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list latest event stream: %w", err)
	}
	defer rows.Close()
	return scanEventRows(rows)
}

func (s Store) LatestEventByType(ctx context.Context, projectID int64, eventType string) (EventRecord, bool, error) {
	events, err := s.listEventsByType(ctx, projectID, eventType, 1)
	if err != nil {
		return EventRecord{}, false, err
	}
	if len(events) == 0 {
		return EventRecord{}, false, nil
	}
	return events[0], true, nil
}

func (s Store) listEventsByType(ctx context.Context, projectID int64, eventType string, limit int) ([]EventRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE project_id = $1 AND event_type = $2
ORDER BY created_at DESC, id DESC
LIMIT $3`,
		projectID,
		eventType,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list project events by type: %w", err)
	}
	defer rows.Close()
	return scanEventRows(rows)
}

func NewStore(pool *pgxpool.Pool) Store {
	return Store{pool: pool}
}

func (s Store) ActiveProjectConfig(ctx context.Context, projectID int64) (ProjectConfigRecord, bool, error) {
	var record ProjectConfigRecord
	var ownershipRaw []byte
	var permissionsRaw []byte
	var schedulingRaw []byte
	var enginesRaw []byte
	var statusExportRaw []byte
	var migrationRaw []byte
	var metadataRaw []byte
	var actorID sql.NullInt64
	err := s.pool.QueryRow(ctx, `
SELECT id, project_id, protocol_version, config_path, config_hash,
       ownership, permissions, scheduling, engines, status_export, migration, metadata,
       active, loaded_at, loaded_by_actor_id
FROM project_configs
WHERE project_id = $1 AND active
ORDER BY loaded_at DESC, id DESC
LIMIT 1`,
		projectID,
	).Scan(
		&record.ID,
		&record.ProjectID,
		&record.ProtocolVersion,
		&record.ConfigPath,
		&record.ConfigHash,
		&ownershipRaw,
		&permissionsRaw,
		&schedulingRaw,
		&enginesRaw,
		&statusExportRaw,
		&migrationRaw,
		&metadataRaw,
		&record.Active,
		&record.LoadedAt,
		&actorID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectConfigRecord{}, false, nil
		}
		return ProjectConfigRecord{}, false, fmt.Errorf("load active project config: %w", err)
	}
	record.LoadedByActorID = actorID.Int64
	if err := unmarshalProjectConfigJSON(ownershipRaw, &record.Ownership, "ownership"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	if err := unmarshalProjectConfigJSON(permissionsRaw, &record.Permissions, "permissions"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	if err := unmarshalProjectConfigJSON(schedulingRaw, &record.Scheduling, "scheduling"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	if err := unmarshalProjectConfigJSON(enginesRaw, &record.Engines, "engines"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	if err := unmarshalProjectConfigJSON(statusExportRaw, &record.StatusExport, "status_export"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	if err := unmarshalProjectConfigJSON(migrationRaw, &record.Migration, "migration"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	if err := unmarshalProjectConfigJSON(metadataRaw, &record.Metadata, "metadata"); err != nil {
		return ProjectConfigRecord{}, false, err
	}
	return record, true, nil
}

func (s Store) UpsertFromConfig(ctx context.Context, cfg Config) (Record, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Record{}, fmt.Errorf("begin project upsert: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
INSERT INTO projects (project_key, name, kind, adapter, workflow_profile, default_branch, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (project_key) DO UPDATE SET
    name = EXCLUDED.name,
    kind = EXCLUDED.kind,
    adapter = EXCLUDED.adapter,
    workflow_profile = EXCLUDED.workflow_profile,
    default_branch = EXCLUDED.default_branch,
    updated_at = now()
RETURNING id`,
		cfg.Project.ID,
		cfg.Project.Name,
		cfg.Project.Kind,
		cfg.Project.Adapter,
		cfg.Project.WorkflowProfile,
		cfg.Project.DefaultBranch,
	).Scan(&id)
	if err != nil {
		return Record{}, fmt.Errorf("upsert project %s: %w", cfg.Project.ID, err)
	}

	if _, err := tx.Exec(ctx, `
DELETE FROM project_connections
WHERE project_id = $1 AND connection_type IN ('local_path', 'artifact_store')`,
		id,
	); err != nil {
		return Record{}, fmt.Errorf("replace project connections: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO project_connections (project_id, connection_type, root_path, current_branch, updated_at)
VALUES ($1, 'local_path', $2, $3, now())`,
		id,
		cfg.Project.Root,
		cfg.Project.DefaultBranch,
	); err != nil {
		return Record{}, fmt.Errorf("insert project connection: %w", err)
	}
	if cfg.ArtifactStore.Backend != "" || cfg.ArtifactStore.Root != "" {
		if _, err := tx.Exec(ctx, `
INSERT INTO project_connections (project_id, connection_type, root_path, remote_url, updated_at)
VALUES ($1, 'artifact_store', $2, $3, now())`,
			id,
			cfg.ArtifactStore.Root,
			cfg.ArtifactStore.Backend,
		); err != nil {
			return Record{}, fmt.Errorf("insert project artifact store connection: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM project_permissions WHERE project_id = $1`, id); err != nil {
		return Record{}, fmt.Errorf("clear project permissions: %w", err)
	}
	if err := insertPermissions(ctx, tx, id, cfg); err != nil {
		return Record{}, err
	}
	if err := upsertSchedulingPolicy(ctx, tx, id, cfg); err != nil {
		return Record{}, err
	}
	if err := insertProjectConfigSnapshot(ctx, tx, id, cfg); err != nil {
		return Record{}, err
	}

	auditMetadata, err := json.Marshal(map[string]any{
		"ownership": map[string]any{
			"mode":                  cfg.Ownership.Mode,
			"source_of_truth":       cfg.Ownership.SourceOfTruth,
			"cutover":               cfg.Ownership.Cutover,
			"new_versions_owned_by": cfg.Ownership.Cutover.NewVersionsOwnedBy,
		},
		"scheduling": map[string]any{
			"priority":              cfg.Scheduling.Priority,
			"max_parallel_tasks":    cfg.Scheduling.MaxParallelTasks,
			"agent_role":            cfg.Scheduling.AgentRole,
			"required_capabilities": cfg.Scheduling.RequiredCapabilities,
			"engine_profile":        cfg.Scheduling.EngineProfile,
		},
		"status_export": cfg.StatusExport,
		"migration":     cfg.Migration,
	})
	if err != nil {
		return Record{}, fmt.Errorf("marshal project upsert audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'project.upsert', 'project', $2, 'allowed', 'local CLI project registration', $3::jsonb)`,
		id,
		cfg.Project.ID,
		string(auditMetadata),
	); err != nil {
		return Record{}, fmt.Errorf("record project upsert audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Record{}, fmt.Errorf("commit project upsert: %w", err)
	}

	return s.GetByKey(ctx, cfg.Project.ID)
}

func insertProjectConfigSnapshot(ctx context.Context, tx pgx.Tx, projectID int64, cfg Config) error {
	ownershipJSON, err := marshalProjectConfigPart(configOwnershipMap(cfg.Ownership), "ownership")
	if err != nil {
		return err
	}
	permissionsJSON, err := marshalProjectConfigPart(configPermissionsMap(cfg.Permissions), "permissions")
	if err != nil {
		return err
	}
	schedulingJSON, err := marshalProjectConfigPart(configSchedulingMap(cfg.Scheduling), "scheduling")
	if err != nil {
		return err
	}
	enginesJSON, err := marshalProjectConfigPart(configEnginesMap(cfg.Engines), "engines")
	if err != nil {
		return err
	}
	statusExportJSON, err := marshalProjectConfigPart(configStatusExportMap(cfg.StatusExport), "status_export")
	if err != nil {
		return err
	}
	migrationJSON, err := marshalProjectConfigPart(configMigrationMap(cfg.Migration), "migration")
	if err != nil {
		return err
	}
	metadataJSON, err := marshalProjectConfigPart(map[string]any{
		"project":        configProjectMap(cfg.Project),
		"artifact_store": configArtifactStoreMap(cfg.ArtifactStore),
		"commands":       configCommandsMap(cfg.Commands),
		"source":         "areaflow.yaml",
	}, "metadata")
	if err != nil {
		return err
	}
	configPath := cfg.SourcePath
	if configPath == "" {
		configPath = "areaflow.yaml"
	}
	configHash := cfg.SourceHash
	if configHash == "" {
		hashSource, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshal project config fallback hash source: %w", err)
		}
		configHash = sha256Hex(hashSource)
	}
	if _, err := tx.Exec(ctx, `
UPDATE project_configs
SET active = false
WHERE project_id = $1 AND active`,
		projectID,
	); err != nil {
		return fmt.Errorf("deactivate previous project configs: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO project_configs (
    project_id, protocol_version, config_path, config_hash,
    ownership, permissions, scheduling, engines, status_export, migration, metadata, active
)
VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, true)`,
		projectID,
		cfg.Version,
		configPath,
		configHash,
		string(ownershipJSON),
		string(permissionsJSON),
		string(schedulingJSON),
		string(enginesJSON),
		string(statusExportJSON),
		string(migrationJSON),
		string(metadataJSON),
	); err != nil {
		return fmt.Errorf("insert project config snapshot: %w", err)
	}
	return nil
}

func configProjectMap(project ProjectConfig) map[string]any {
	return map[string]any{
		"id":               project.ID,
		"name":             project.Name,
		"root":             project.Root,
		"kind":             project.Kind,
		"adapter":          project.Adapter,
		"workflow_profile": project.WorkflowProfile,
		"default_branch":   project.DefaultBranch,
	}
}

func configOwnershipMap(ownership Ownership) map[string]any {
	return map[string]any{
		"mode": ownership.Mode,
		"source_of_truth": map[string]any{
			"product_docs":   ownership.SourceOfTruth.ProductDocs,
			"source_code":    ownership.SourceOfTruth.SourceCode,
			"workflow":       ownership.SourceOfTruth.Workflow,
			"execution":      ownership.SourceOfTruth.Execution,
			"status_summary": ownership.SourceOfTruth.StatusSummary,
		},
		"cutover": map[string]any{
			"enabled":               ownership.Cutover.Enabled,
			"new_versions_owned_by": ownership.Cutover.NewVersionsOwnedBy,
			"legacy_versions_mode":  ownership.Cutover.LegacyVersionsMode,
		},
	}
}

func configArtifactStoreMap(store ArtifactStore) map[string]any {
	return map[string]any{
		"backend": store.Backend,
		"root":    store.Root,
	}
}

func configPermissionsMap(permissions Permissions) map[string]any {
	return map[string]any{
		"capabilities":    permissions.Capabilities,
		"read_paths":      permissions.ReadPaths,
		"write_paths":     permissions.WritePaths,
		"forbidden_paths": permissions.ForbiddenPath,
	}
}

func configCommandsMap(commands Commands) map[string]any {
	return map[string]any{
		"allowed":   commands.Allowed,
		"forbidden": commands.Forbidden,
	}
}

func configSchedulingMap(scheduling Scheduling) map[string]any {
	return map[string]any{
		"priority":              scheduling.Priority,
		"max_parallel_tasks":    scheduling.MaxParallelTasks,
		"agent_role":            scheduling.AgentRole,
		"required_capabilities": scheduling.RequiredCapabilities,
		"engine_profile":        scheduling.EngineProfile,
	}
}

func configEnginesMap(engines Engines) map[string]any {
	profiles := make([]map[string]any, 0, len(engines.Profiles))
	for _, profile := range engines.Profiles {
		profiles = append(profiles, map[string]any{
			"id":              profile.ID,
			"provider":        profile.Provider,
			"secret_ref":      profile.SecretRef,
			"enabled":         profile.Enabled,
			"resource_limits": profile.ResourceLimits,
		})
	}
	return map[string]any{
		"default":  engines.Default,
		"profiles": profiles,
	}
}

func configStatusExportMap(status StatusExport) map[string]any {
	return map[string]any{
		"enabled": status.Enabled,
		"path":    status.Path,
		"human_summary": map[string]any{
			"enabled":      status.HumanSummary.Enabled,
			"path":         status.HumanSummary.Path,
			"block_marker": status.HumanSummary.BlockMarker,
		},
	}
}

func configMigrationMap(migration Migration) map[string]any {
	return map[string]any{
		"strategy":          migration.Strategy,
		"phase":             migration.Phase,
		"imported_versions": migration.ImportedVersions,
		"immutable_imports": migration.ImmutableImports,
	}
}

func marshalProjectConfigPart(value any, name string) ([]byte, error) {
	content, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal project config %s: %w", name, err)
	}
	return content, nil
}

func unmarshalProjectConfigJSON(content []byte, target *map[string]any, name string) error {
	if len(content) == 0 {
		*target = map[string]any{}
		return nil
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("parse project config %s: %w", name, err)
	}
	if *target == nil {
		*target = map[string]any{}
	}
	return nil
}

func upsertSchedulingPolicy(ctx context.Context, tx pgx.Tx, projectID int64, cfg Config) error {
	policy := schedulingPolicyFromConfig(cfg.Scheduling)
	requiredCapabilitiesJSON, err := json.Marshal(policy.RequiredCapabilities)
	if err != nil {
		return fmt.Errorf("marshal project scheduling capabilities: %w", err)
	}
	metadataJSON, err := json.Marshal(map[string]any{
		"source": "project_config",
		"phase":  "v0.8c",
		"engine": engineReadinessFromConfig(cfg),
	})
	if err != nil {
		return fmt.Errorf("marshal project scheduling metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO project_scheduling_policies (
    project_id, priority, max_parallel_tasks, agent_role,
    required_capabilities, engine_profile, metadata, updated_at
)
VALUES ($1, $2, $3, $4, $5::jsonb, NULLIF($6, ''), $7::jsonb, now())
ON CONFLICT (project_id)
DO UPDATE SET
    priority = EXCLUDED.priority,
    max_parallel_tasks = EXCLUDED.max_parallel_tasks,
    agent_role = EXCLUDED.agent_role,
    required_capabilities = EXCLUDED.required_capabilities,
    engine_profile = EXCLUDED.engine_profile,
    metadata = EXCLUDED.metadata,
    updated_at = now()`,
		projectID,
		policy.Priority,
		policy.MaxParallelTasks,
		policy.AgentRole,
		string(requiredCapabilitiesJSON),
		policy.EngineProfile,
		string(metadataJSON),
	); err != nil {
		return fmt.Errorf("upsert project scheduling policy: %w", err)
	}
	return nil
}

func (s Store) GetByKey(ctx context.Context, key string) (Record, error) {
	var record Record
	err := s.pool.QueryRow(ctx, `
SELECT p.id, p.project_key, p.name, p.kind, p.adapter, p.workflow_profile, COALESCE(p.default_branch, ''),
       COALESCE(c.root_path, ''), COALESCE(a.remote_url, ''), COALESCE(a.root_path, '')
FROM projects p
LEFT JOIN LATERAL (
    SELECT root_path
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'local_path'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) c ON true
LEFT JOIN LATERAL (
    SELECT root_path, remote_url
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'artifact_store'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) a ON true
WHERE p.project_key = $1`,
		key,
	).Scan(
		&record.ID,
		&record.Key,
		&record.Name,
		&record.Kind,
		&record.Adapter,
		&record.WorkflowProfile,
		&record.DefaultBranch,
		&record.RootPath,
		&record.ArtifactBackend,
		&record.ArtifactRoot,
	)
	if err != nil {
		return Record{}, fmt.Errorf("get project %s: %w", key, err)
	}
	return record, nil
}

func (s Store) List(ctx context.Context) ([]Record, error) {
	rows, err := s.pool.Query(ctx, `
SELECT p.id, p.project_key, p.name, p.kind, p.adapter, p.workflow_profile, COALESCE(p.default_branch, ''),
       COALESCE(c.root_path, ''), COALESCE(a.remote_url, ''), COALESCE(a.root_path, '')
FROM projects p
LEFT JOIN LATERAL (
    SELECT root_path
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'local_path'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) c ON true
LEFT JOIN LATERAL (
    SELECT root_path, remote_url
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'artifact_store'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) a ON true
WHERE p.archived_at IS NULL
ORDER BY p.project_key`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	records := []Record{}
	for rows.Next() {
		var record Record
		if err := rows.Scan(
			&record.ID,
			&record.Key,
			&record.Name,
			&record.Kind,
			&record.Adapter,
			&record.WorkflowProfile,
			&record.DefaultBranch,
			&record.RootPath,
			&record.ArtifactBackend,
			&record.ArtifactRoot,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return records, nil
}

func insertPermissions(ctx context.Context, tx pgx.Tx, projectID int64, cfg Config) error {
	for capability, allowed := range cfg.Permissions.Capabilities {
		effect := "deny"
		if allowed {
			effect = "allow"
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO project_permissions (project_id, capability, effect, resource_type, pattern)
VALUES ($1, $2, $3, 'capability', $2)`,
			projectID,
			capability,
			effect,
		); err != nil {
			return fmt.Errorf("insert capability permission %s: %w", capability, err)
		}
	}

	for _, path := range cfg.Permissions.ReadPaths {
		if err := insertPathPermission(ctx, tx, projectID, "read_project", "allow", path); err != nil {
			return err
		}
	}
	for _, path := range cfg.Permissions.WritePaths {
		if err := insertPathPermission(ctx, tx, projectID, "write_status", "allow", path); err != nil {
			return err
		}
		if shouldAllowGeneratedWritePath(cfg, path) {
			if err := insertPathPermission(ctx, tx, projectID, "write_generated", "allow", path); err != nil {
				return err
			}
		}
	}
	for _, path := range cfg.Permissions.ForbiddenPath {
		if err := insertPathPermission(ctx, tx, projectID, "*", "deny", path); err != nil {
			return err
		}
	}
	for _, command := range cfg.Commands.Allowed {
		if err := insertCommandPermission(ctx, tx, projectID, "run_commands", "allow", command); err != nil {
			return err
		}
	}
	for _, command := range cfg.Commands.Forbidden {
		if err := insertCommandPermission(ctx, tx, projectID, "run_commands", "deny", command); err != nil {
			return err
		}
	}
	return nil
}

func shouldAllowGeneratedWritePath(cfg Config, path string) bool {
	return cfg.Permissions.Capabilities["write_generated"] && isManagedGeneratedPath(path)
}

func insertPathPermission(ctx context.Context, tx pgx.Tx, projectID int64, capability string, effect string, pattern string) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO project_permissions (project_id, capability, effect, resource_type, pattern)
VALUES ($1, $2, $3, 'path', $4)`,
		projectID,
		capability,
		effect,
		pattern,
	); err != nil {
		return fmt.Errorf("insert path permission %s %s %s: %w", effect, capability, pattern, err)
	}
	return nil
}

func insertCommandPermission(ctx context.Context, tx pgx.Tx, projectID int64, capability string, effect string, pattern string) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO project_permissions (project_id, capability, effect, resource_type, pattern)
VALUES ($1, $2, $3, 'command', $4)`,
		projectID,
		capability,
		effect,
		pattern,
	); err != nil {
		return fmt.Errorf("insert command permission %s %s %s: %w", effect, capability, pattern, err)
	}
	return nil
}

func globMatch(pattern string, path string) bool {
	if pattern == path || pattern == "*" {
		return true
	}
	if len(pattern) >= 3 && pattern[len(pattern)-3:] == "/**" {
		prefix := pattern[:len(pattern)-3]
		return path == prefix || (len(path) > len(prefix) && path[:len(prefix)+1] == prefix+"/")
	}
	return false
}

func snapshotValue(snapshot Snapshot, key string) string {
	return metadataValue(snapshot.Summary, key)
}

func nestedSnapshotValue(snapshot Snapshot, key string, nestedKey string) string {
	value, ok := snapshot.Summary[key].(map[string]any)
	if !ok {
		return ""
	}
	return metadataValue(value, nestedKey)
}

func metadataString(metadata map[string]any, key string) string {
	value, ok := metadata[key].(string)
	if !ok {
		return ""
	}
	return value
}

func metadataValue(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func scanResidualRecord(row scanner) (ResidualRecord, error) {
	var residual ResidualRecord
	var metadataRaw []byte
	var importedAt sql.NullTime
	if err := row.Scan(
		&residual.ID,
		&residual.ProjectID,
		&residual.WorkflowVersionID,
		&residual.ResidualKey,
		&residual.Status,
		&residual.Type,
		&residual.Title,
		&residual.SourcePath,
		&residual.CurrentImpact,
		&residual.ExecutableTask,
		&residual.PromotionRequired,
		&residual.CloseCondition,
		&metadataRaw,
		&residual.Immutable,
		&residual.CreatedAt,
		&residual.UpdatedAt,
		&importedAt,
	); err != nil {
		return ResidualRecord{}, err
	}
	if err := json.Unmarshal(metadataRaw, &residual.Metadata); err != nil {
		return ResidualRecord{}, fmt.Errorf("parse residual metadata: %w", err)
	}
	if importedAt.Valid {
		residual.ImportedAt = &importedAt.Time
	}
	return residual, nil
}

func checkStatus(metadata map[string]any, checkName string) string {
	checks, ok := metadata["checks"].([]any)
	if !ok {
		return ""
	}
	for _, raw := range checks {
		check, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if metadataString(check, "name") == checkName {
			return metadataString(check, "status")
		}
	}
	return ""
}
