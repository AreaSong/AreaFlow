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
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrWorkerNotFound         = errors.New("worker not found")
	ErrLeaseNotFound          = errors.New("lease not found")
	ErrNoLeaseAvailable       = errors.New("no lease available")
	ErrWorkerCapabilityDenied = errors.New("worker capability denied")
)

var leaseRecoverDefaultKeySequence atomic.Int64
var workerLifecycleDefaultKeySequence atomic.Int64

type WorkerRecord struct {
	ID                       int64
	ProjectID                int64
	ActorID                  int64
	WorkerKey                string
	WorkerType               string
	Status                   string
	Hostname                 string
	PID                      int
	Capabilities             []string
	Metadata                 map[string]any
	RegisteredAt             time.Time
	LastHeartbeatAt          *time.Time
	HeartbeatIntervalSeconds int
	LeaseTimeoutSeconds      int
	UpdatedAt                time.Time
}

type RegisterWorkerOptions struct {
	WorkerKey                string
	WorkerType               string
	Hostname                 string
	PID                      int
	Capabilities             []string
	Metadata                 map[string]any
	HeartbeatIntervalSeconds int
	LeaseTimeoutSeconds      int
	Actor                    string
	Reason                   string
	IdempotencyKey           string
}

type WorkerHeartbeatOptions struct {
	Status         string
	Metadata       map[string]any
	Actor          string
	Reason         string
	IdempotencyKey string
}

type LeaseRecord struct {
	ID                  int64
	ProjectID           int64
	RunID               int64
	RunTaskID           int64
	WorkflowItemID      int64
	WorkerID            int64
	LeaseKind           string
	Status              string
	AcquiredAt          time.Time
	ExpiresAt           time.Time
	HeartbeatAt         *time.Time
	ReleasedAt          *time.Time
	AllowedCapabilities []string
	Scope               map[string]any
	Metadata            map[string]any
}

type AcquireLeaseOptions struct {
	WorkerKey            string
	RunTaskID            int64
	LeaseKind            string
	AllowedCapabilities  []string
	Scope                map[string]any
	Metadata             map[string]any
	LeaseTimeoutSeconds  int
	RecoverExpiredBefore bool
	Actor                string
	Reason               string
	IdempotencyKey       string
}

type ReleaseLeaseOptions struct {
	WorkerKey      string
	LeaseID        int64
	Status         string
	Metadata       map[string]any
	Actor          string
	Reason         string
	IdempotencyKey string
}

type RecoverLeasesOptions struct {
	Limit          int
	Actor          string
	Reason         string
	Metadata       map[string]any
	IdempotencyKey string
}

type WorkerRunOnceOptions struct {
	WorkerKey           string
	RunID               int64
	AllowedCapabilities []string
	LeaseTimeoutSeconds int
	Actor               string
	Reason              string
	Metadata            map[string]any
}

type WorkerRunOnceResult struct {
	Project  Record
	Worker   WorkerRecord
	Lease    LeaseRecord
	Task     RunTaskRecord
	Attempt  RunAttemptRecord
	Artifact ArtifactRecord
	Claimed  bool
}

type WorkerPoolProjectSummary struct {
	Project             Record
	Workers             int64
	OnlineWorkers       int64
	OfflineWorkers      int64
	ActiveLeases        int64
	NeedsRecoveryLeases int64
	QueuedTasks         int64
	NeedsRecoveryTasks  int64
	Capabilities        []string
	WorkerTypes         []string
	Scheduling          SchedulingPolicy
	Role                RoleReadiness
	Engine              EngineReadiness
	Resources           ResourceReadiness
	LastWorkerHeartbeat *time.Time
}

type WorkerPoolSummary struct {
	Projects           []WorkerPoolProjectSummary
	TotalProjects      int64
	TotalWorkers       int64
	TotalOnlineWorkers int64
	TotalActiveLeases  int64
	TotalQueuedTasks   int64
	TotalNeedsRecovery int64
	GeneratedAt        time.Time
}

type WorkerPoolSchedulePreview struct {
	Projects      []WorkerPoolProjectSchedule
	Policy        WorkerPoolSchedulePolicy
	GeneratedAt   time.Time
	Recommended   int64
	Blocked       int64
	QueuedTasks   int64
	AvailableSlot int64
}

type WorkerPoolSchedulePolicy struct {
	Strategy               string
	DefaultProjectPriority int
	SlotStrategy           string
	DryRunOnly             bool
}

type WorkerPoolProjectSchedule struct {
	Project        Record
	Priority       int
	MaxParallel    int
	AgentRole      string
	Role           RoleReadiness
	EngineProfile  string
	Engine         EngineReadiness
	Resources      ResourceReadiness
	QueuedTasks    int64
	ActiveLeases   int64
	OnlineWorkers  int64
	AvailableSlots int64
	NeedsRecovery  int64
	Capabilities   []string
	RequiredCaps   []string
	Recommended    bool
	BlockedReasons []string
	NextAction     string
}

type SchedulingPolicy struct {
	Priority             int
	MaxParallelTasks     int
	AgentRole            string
	RequiredCapabilities []string
	EngineProfile        string
}

type RoleReadiness struct {
	RequiredRole   string
	Matched        bool
	MatchedTypes   []string
	Status         string
	BlockedReasons []string
}

type EngineReadiness struct {
	ProfileID      string
	Provider       string
	Enabled        bool
	SecretRef      string
	SecretRequired bool
	SecretReady    bool
	ResourceLimits map[string]any
	Status         string
	BlockedReasons []string
}

type ResourceReadiness struct {
	MaxActiveLeases int64
	MaxQueuedTasks  int64
	Status          string
	BlockedReasons  []string
}

func (s Store) RegisterWorker(ctx context.Context, record Record, options RegisterWorkerOptions) (WorkerRecord, error) {
	options = normalizeRegisterWorkerOptions(record, options)
	requestHash, err := workerRegisterRequestHash(record, options)
	if err != nil {
		return WorkerRecord{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = workerRegisterIdempotencyKey(record, options, requestHash)
	}
	capabilitiesJSON, err := json.Marshal(options.Capabilities)
	if err != nil {
		return WorkerRecord{}, fmt.Errorf("marshal worker capabilities: %w", err)
	}
	metadataJSON, err := json.Marshal(options.Metadata)
	if err != nil {
		return WorkerRecord{}, fmt.Errorf("marshal worker metadata: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkerRecord{}, fmt.Errorf("begin worker register: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, "worker.register", options.IdempotencyKey, requestHash)
	if err != nil {
		return WorkerRecord{}, err
	}
	if !created {
		worker, err := loadWorkerByCommandResponse(ctx, tx, record.ID, "worker.register", options.IdempotencyKey)
		if err != nil {
			return WorkerRecord{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return WorkerRecord{}, fmt.Errorf("commit idempotent worker register: %w", err)
		}
		return worker, nil
	}

	actorID, err := ensureWorkerActor(ctx, tx, record, options.WorkerKey)
	if err != nil {
		return WorkerRecord{}, err
	}

	worker, err := scanWorker(tx.QueryRow(ctx, `
INSERT INTO workers (
    project_id, actor_id, worker_key, worker_type, status, hostname, pid,
    capabilities, metadata, heartbeat_interval_seconds, lease_timeout_seconds,
    last_heartbeat_at
)
VALUES ($1, $2, $3, $4, 'online', $5, $6, $7::jsonb, $8::jsonb, $9, $10, now())
ON CONFLICT (project_id, worker_key)
DO UPDATE SET
    actor_id = EXCLUDED.actor_id,
    worker_type = EXCLUDED.worker_type,
    status = 'online',
    hostname = EXCLUDED.hostname,
    pid = EXCLUDED.pid,
    capabilities = EXCLUDED.capabilities,
    metadata = EXCLUDED.metadata,
    heartbeat_interval_seconds = EXCLUDED.heartbeat_interval_seconds,
    lease_timeout_seconds = EXCLUDED.lease_timeout_seconds,
    last_heartbeat_at = now(),
    updated_at = now()
RETURNING id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
          COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
          registered_at, last_heartbeat_at, heartbeat_interval_seconds,
          lease_timeout_seconds, updated_at`,
		record.ID,
		actorID,
		options.WorkerKey,
		options.WorkerType,
		options.Hostname,
		nullablePID(options.PID),
		string(capabilitiesJSON),
		string(metadataJSON),
		options.HeartbeatIntervalSeconds,
		options.LeaseTimeoutSeconds,
	))
	if err != nil {
		return WorkerRecord{}, fmt.Errorf("register worker: %w", err)
	}

	heartbeatID, err := insertWorkerHeartbeat(ctx, tx, record.ID, worker.ID, worker.Status, options.Metadata)
	if err != nil {
		return WorkerRecord{}, err
	}
	eventID, err := insertWorkerEvent(ctx, tx, record.ID, worker.ID, "worker.registered", "Worker registered", map[string]any{
		"worker_key":   worker.WorkerKey,
		"worker_type":  worker.WorkerType,
		"capabilities": worker.Capabilities,
	})
	if err != nil {
		return WorkerRecord{}, err
	}
	auditEventID, err := insertWorkerAuditEvent(ctx, tx, record.ID, actorID, "worker.register", "manage_workers", worker.WorkerKey, "allowed", options.Reason, map[string]any{
		"worker_id":   worker.ID,
		"worker_type": worker.WorkerType,
	})
	if err != nil {
		return WorkerRecord{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "worker.register", options.IdempotencyKey, workerLifecycleCommandResponse(map[string]any{
		"decision":            "allowed",
		"worker_id":           worker.ID,
		"worker_key":          worker.WorkerKey,
		"worker_type":         worker.WorkerType,
		"worker_status":       worker.Status,
		"heartbeat_id":        heartbeatID,
		"event_id":            eventID,
		"audit_event_id":      auditEventID,
		"worker_registered":   true,
		"heartbeat_recorded":  true,
		"worker_record_write": true,
	})); err != nil {
		return WorkerRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkerRecord{}, fmt.Errorf("commit worker register: %w", err)
	}
	return worker, nil
}

func (s Store) RecordWorkerHeartbeat(ctx context.Context, record Record, workerKey string, options WorkerHeartbeatOptions) (WorkerRecord, error) {
	workerKey = strings.TrimSpace(workerKey)
	if workerKey == "" {
		return WorkerRecord{}, fmt.Errorf("worker key is required")
	}
	options = normalizeWorkerHeartbeatOptions(options)
	requestHash, err := workerHeartbeatRequestHash(record, workerKey, options)
	if err != nil {
		return WorkerRecord{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = workerHeartbeatIdempotencyKey(record, workerKey, options, requestHash)
	}
	metadataJSON, err := json.Marshal(options.Metadata)
	if err != nil {
		return WorkerRecord{}, fmt.Errorf("marshal worker heartbeat metadata: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkerRecord{}, fmt.Errorf("begin worker heartbeat: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, "worker.heartbeat", options.IdempotencyKey, requestHash)
	if err != nil {
		return WorkerRecord{}, err
	}
	if !created {
		worker, err := loadWorkerByCommandResponse(ctx, tx, record.ID, "worker.heartbeat", options.IdempotencyKey)
		if err != nil {
			return WorkerRecord{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return WorkerRecord{}, fmt.Errorf("commit idempotent worker heartbeat: %w", err)
		}
		return worker, nil
	}

	worker, err := scanWorker(tx.QueryRow(ctx, `
UPDATE workers
SET status = $3,
    last_heartbeat_at = now(),
    metadata = metadata || $4::jsonb,
    updated_at = now()
WHERE project_id = $1 AND worker_key = $2
RETURNING id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
          COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
          registered_at, last_heartbeat_at, heartbeat_interval_seconds,
          lease_timeout_seconds, updated_at`,
		record.ID,
		workerKey,
		options.Status,
		string(metadataJSON),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkerRecord{}, ErrWorkerNotFound
		}
		return WorkerRecord{}, fmt.Errorf("record worker heartbeat: %w", err)
	}

	heartbeatID, err := insertWorkerHeartbeat(ctx, tx, record.ID, worker.ID, worker.Status, options.Metadata)
	if err != nil {
		return WorkerRecord{}, err
	}
	eventID, err := insertWorkerEvent(ctx, tx, record.ID, worker.ID, "worker.heartbeat", "Worker heartbeat recorded", map[string]any{
		"worker_key": worker.WorkerKey,
		"status":     worker.Status,
	})
	if err != nil {
		return WorkerRecord{}, err
	}
	auditEventID, err := insertWorkerAuditEvent(ctx, tx, record.ID, worker.ActorID, "worker.heartbeat", "manage_workers", worker.WorkerKey, "allowed", options.Reason, map[string]any{
		"worker_id": worker.ID,
		"status":    worker.Status,
	})
	if err != nil {
		return WorkerRecord{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "worker.heartbeat", options.IdempotencyKey, workerLifecycleCommandResponse(map[string]any{
		"decision":            "allowed",
		"worker_id":           worker.ID,
		"worker_key":          worker.WorkerKey,
		"worker_type":         worker.WorkerType,
		"worker_status":       worker.Status,
		"heartbeat_id":        heartbeatID,
		"event_id":            eventID,
		"audit_event_id":      auditEventID,
		"worker_registered":   false,
		"heartbeat_recorded":  true,
		"worker_record_write": true,
	})); err != nil {
		return WorkerRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkerRecord{}, fmt.Errorf("commit worker heartbeat: %w", err)
	}
	return worker, nil
}

func (s Store) ListWorkers(ctx context.Context, record Record, limit int) ([]WorkerRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
       COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
       registered_at, last_heartbeat_at, heartbeat_interval_seconds,
       lease_timeout_seconds, updated_at
FROM workers
WHERE project_id = $1
ORDER BY updated_at DESC, id DESC
LIMIT $2`,
		record.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()

	workers := []WorkerRecord{}
	for rows.Next() {
		worker, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workers: %w", err)
	}
	return workers, nil
}

func (s Store) WorkerPoolSummary(ctx context.Context) (WorkerPoolSummary, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
    p.id,
    p.project_key,
    p.name,
    p.kind,
    p.adapter,
    p.workflow_profile,
    p.default_branch,
    COALESCE(local_path.root_path, ''),
    COALESCE(artifact_store.remote_url, ''),
    COALESCE(artifact_store.root_path, ''),
    COUNT(DISTINCT w.id) AS workers,
    COUNT(DISTINCT w.id) FILTER (WHERE w.status = 'online') AS online_workers,
    COUNT(DISTINCT w.id) FILTER (WHERE w.status <> 'online') AS offline_workers,
    COUNT(DISTINCT l.id) FILTER (WHERE l.status = 'active') AS active_leases,
    COUNT(DISTINCT l.id) FILTER (WHERE l.status = 'needs_recovery') AS needs_recovery_leases,
    COUNT(DISTINCT rt.id) FILTER (WHERE rt.status = 'queued') AS queued_tasks,
    COUNT(DISTINCT rt.id) FILTER (WHERE rt.status = 'needs_recovery') AS needs_recovery_tasks,
    COALESCE(
        jsonb_agg(DISTINCT capability.value) FILTER (WHERE capability.value IS NOT NULL),
        '[]'::jsonb
    ) AS capabilities,
    COALESCE(
        jsonb_agg(DISTINCT w.worker_type) FILTER (WHERE w.id IS NOT NULL AND w.status = 'online'),
        '[]'::jsonb
    ) AS worker_types,
    COALESCE(policy.priority, 100),
    COALESCE(policy.max_parallel_tasks, 1),
    COALESCE(policy.agent_role, 'local_worker'),
    COALESCE(policy.required_capabilities, '["read_project"]'::jsonb),
    COALESCE(policy.engine_profile, ''),
    COALESCE(policy.metadata, '{}'::jsonb),
    MAX(w.last_heartbeat_at) AS last_worker_heartbeat
FROM projects p
LEFT JOIN LATERAL (
    SELECT root_path
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'local_path'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) local_path ON true
LEFT JOIN LATERAL (
    SELECT root_path, remote_url
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'artifact_store'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) artifact_store ON true
LEFT JOIN workers w ON w.project_id = p.id
LEFT JOIN LATERAL jsonb_array_elements_text(COALESCE(w.capabilities, '[]'::jsonb)) capability(value) ON true
LEFT JOIN leases l ON l.project_id = p.id
LEFT JOIN run_tasks rt ON rt.project_id = p.id
LEFT JOIN project_scheduling_policies policy ON policy.project_id = p.id
WHERE p.archived_at IS NULL
GROUP BY p.id, local_path.root_path, artifact_store.remote_url, artifact_store.root_path,
         policy.priority, policy.max_parallel_tasks, policy.agent_role,
         policy.required_capabilities, policy.engine_profile, policy.metadata
ORDER BY queued_tasks DESC, active_leases DESC, p.project_key`,
	)
	if err != nil {
		return WorkerPoolSummary{}, fmt.Errorf("load worker pool summary: %w", err)
	}
	defer rows.Close()

	summary := WorkerPoolSummary{
		Projects:    []WorkerPoolProjectSummary{},
		GeneratedAt: time.Now().UTC(),
	}
	for rows.Next() {
		var projectSummary WorkerPoolProjectSummary
		var capabilitiesRaw []byte
		var workerTypesRaw []byte
		var requiredCapabilitiesRaw []byte
		var policyMetadataRaw []byte
		var lastHeartbeat sql.NullTime
		if err := rows.Scan(
			&projectSummary.Project.ID,
			&projectSummary.Project.Key,
			&projectSummary.Project.Name,
			&projectSummary.Project.Kind,
			&projectSummary.Project.Adapter,
			&projectSummary.Project.WorkflowProfile,
			&projectSummary.Project.DefaultBranch,
			&projectSummary.Project.RootPath,
			&projectSummary.Project.ArtifactBackend,
			&projectSummary.Project.ArtifactRoot,
			&projectSummary.Workers,
			&projectSummary.OnlineWorkers,
			&projectSummary.OfflineWorkers,
			&projectSummary.ActiveLeases,
			&projectSummary.NeedsRecoveryLeases,
			&projectSummary.QueuedTasks,
			&projectSummary.NeedsRecoveryTasks,
			&capabilitiesRaw,
			&workerTypesRaw,
			&projectSummary.Scheduling.Priority,
			&projectSummary.Scheduling.MaxParallelTasks,
			&projectSummary.Scheduling.AgentRole,
			&requiredCapabilitiesRaw,
			&projectSummary.Scheduling.EngineProfile,
			&policyMetadataRaw,
			&lastHeartbeat,
		); err != nil {
			return WorkerPoolSummary{}, fmt.Errorf("scan worker pool summary: %w", err)
		}
		if err := json.Unmarshal(capabilitiesRaw, &projectSummary.Capabilities); err != nil {
			return WorkerPoolSummary{}, fmt.Errorf("parse worker pool capabilities: %w", err)
		}
		projectSummary.Capabilities = normalizeCapabilityList(projectSummary.Capabilities)
		if err := json.Unmarshal(workerTypesRaw, &projectSummary.WorkerTypes); err != nil {
			return WorkerPoolSummary{}, fmt.Errorf("parse worker pool worker types: %w", err)
		}
		projectSummary.WorkerTypes = normalizeStringList(projectSummary.WorkerTypes)
		if err := json.Unmarshal(requiredCapabilitiesRaw, &projectSummary.Scheduling.RequiredCapabilities); err != nil {
			return WorkerPoolSummary{}, fmt.Errorf("parse worker pool required capabilities: %w", err)
		}
		projectSummary.Scheduling = schedulingPolicyFromConfig(Scheduling{
			Priority:             projectSummary.Scheduling.Priority,
			MaxParallelTasks:     projectSummary.Scheduling.MaxParallelTasks,
			AgentRole:            projectSummary.Scheduling.AgentRole,
			RequiredCapabilities: projectSummary.Scheduling.RequiredCapabilities,
			EngineProfile:        projectSummary.Scheduling.EngineProfile,
		})
		projectSummary.Role = roleReadinessFromWorkerTypes(projectSummary.Scheduling.AgentRole, projectSummary.WorkerTypes)
		projectSummary.Engine = engineReadinessFromSchedulingMetadata(projectSummary.Scheduling.EngineProfile, policyMetadataRaw)
		projectSummary.Resources = resourceReadinessFromLimits(projectSummary.Engine.ResourceLimits, projectSummary.ActiveLeases, projectSummary.QueuedTasks)
		if lastHeartbeat.Valid {
			projectSummary.LastWorkerHeartbeat = &lastHeartbeat.Time
		}
		summary.TotalProjects++
		summary.TotalWorkers += projectSummary.Workers
		summary.TotalOnlineWorkers += projectSummary.OnlineWorkers
		summary.TotalActiveLeases += projectSummary.ActiveLeases
		summary.TotalQueuedTasks += projectSummary.QueuedTasks
		summary.TotalNeedsRecovery += projectSummary.NeedsRecoveryLeases + projectSummary.NeedsRecoveryTasks
		summary.Projects = append(summary.Projects, projectSummary)
	}
	if err := rows.Err(); err != nil {
		return WorkerPoolSummary{}, fmt.Errorf("iterate worker pool summary: %w", err)
	}
	return summary, nil
}

func (s Store) WorkerPoolSchedulePreview(ctx context.Context) (WorkerPoolSchedulePreview, error) {
	summary, err := s.WorkerPoolSummary(ctx)
	if err != nil {
		return WorkerPoolSchedulePreview{}, err
	}
	return BuildWorkerPoolSchedulePreview(summary), nil
}

func BuildWorkerPoolSchedulePreview(summary WorkerPoolSummary) WorkerPoolSchedulePreview {
	preview := WorkerPoolSchedulePreview{
		Projects:    make([]WorkerPoolProjectSchedule, 0, len(summary.Projects)),
		GeneratedAt: summary.GeneratedAt,
		Policy: WorkerPoolSchedulePolicy{
			Strategy:               "default_fifo",
			DefaultProjectPriority: 100,
			SlotStrategy:           "min_online_workers_and_project_parallelism_minus_active_leases",
			DryRunOnly:             true,
		},
	}
	for _, projectSummary := range summary.Projects {
		item := buildWorkerPoolProjectSchedule(projectSummary)
		preview.QueuedTasks += item.QueuedTasks
		preview.AvailableSlot += item.AvailableSlots
		if item.Recommended {
			preview.Recommended++
		} else if item.QueuedTasks > 0 {
			preview.Blocked++
		}
		preview.Projects = append(preview.Projects, item)
	}
	sortWorkerPoolSchedule(preview.Projects)
	return preview
}

func buildWorkerPoolProjectSchedule(summary WorkerPoolProjectSummary) WorkerPoolProjectSchedule {
	policy := normalizeSchedulingPolicy(summary.Scheduling)
	availableSlots := minInt64(
		nonNegativeInt64(summary.OnlineWorkers-summary.ActiveLeases),
		nonNegativeInt64(int64(policy.MaxParallelTasks)-summary.ActiveLeases),
	)
	item := WorkerPoolProjectSchedule{
		Project:        summary.Project,
		Priority:       policy.Priority,
		MaxParallel:    policy.MaxParallelTasks,
		AgentRole:      policy.AgentRole,
		Role:           roleReadinessFromWorkerTypes(policy.AgentRole, summary.WorkerTypes),
		EngineProfile:  policy.EngineProfile,
		Engine:         normalizeEngineReadiness(summary.Engine, policy.EngineProfile),
		Resources:      resourceReadinessFromLimits(summary.Engine.ResourceLimits, summary.ActiveLeases, summary.QueuedTasks),
		QueuedTasks:    summary.QueuedTasks,
		ActiveLeases:   summary.ActiveLeases,
		OnlineWorkers:  summary.OnlineWorkers,
		AvailableSlots: availableSlots,
		NeedsRecovery:  summary.NeedsRecoveryLeases + summary.NeedsRecoveryTasks,
		Capabilities:   summary.Capabilities,
		RequiredCaps:   policy.RequiredCapabilities,
		NextAction:     "idle",
	}
	if item.QueuedTasks == 0 {
		item.BlockedReasons = append(item.BlockedReasons, "no_queued_tasks")
		return item
	}
	if item.OnlineWorkers == 0 {
		item.BlockedReasons = append(item.BlockedReasons, "no_online_workers")
	}
	if item.Role.Status == "blocked" {
		item.BlockedReasons = append(item.BlockedReasons, item.Role.BlockedReasons...)
	}
	if item.AvailableSlots == 0 {
		item.BlockedReasons = append(item.BlockedReasons, "no_available_worker_slots")
	}
	for _, required := range item.RequiredCaps {
		if !capabilityPresent(item.Capabilities, required) {
			item.BlockedReasons = append(item.BlockedReasons, "missing_required_capability:"+required)
		}
	}
	if item.Engine.Status == "blocked" {
		item.BlockedReasons = append(item.BlockedReasons, item.Engine.BlockedReasons...)
	}
	if item.Resources.Status == "blocked" {
		item.BlockedReasons = append(item.BlockedReasons, item.Resources.BlockedReasons...)
	}
	if len(item.BlockedReasons) == 0 {
		item.Recommended = true
		item.NextAction = "worker_run_once_preview"
	}
	return item
}

func schedulingPolicyFromConfig(cfg Scheduling) SchedulingPolicy {
	normalized := NormalizeScheduling(cfg)
	return SchedulingPolicy{
		Priority:             normalized.Priority,
		MaxParallelTasks:     normalized.MaxParallelTasks,
		AgentRole:            normalized.AgentRole,
		RequiredCapabilities: normalized.RequiredCapabilities,
		EngineProfile:        strings.TrimSpace(normalized.EngineProfile),
	}
}

func normalizeSchedulingPolicy(policy SchedulingPolicy) SchedulingPolicy {
	return schedulingPolicyFromConfig(Scheduling{
		Priority:             policy.Priority,
		MaxParallelTasks:     policy.MaxParallelTasks,
		AgentRole:            policy.AgentRole,
		RequiredCapabilities: policy.RequiredCapabilities,
		EngineProfile:        policy.EngineProfile,
	})
}

func roleReadinessFromWorkerTypes(requiredRole string, workerTypes []string) RoleReadiness {
	requiredRole = strings.TrimSpace(requiredRole)
	readiness := RoleReadiness{
		RequiredRole: requiredRole,
		MatchedTypes: []string{},
		Status:       "ready",
	}
	if requiredRole == "" {
		return readiness
	}
	workerTypes = normalizeStringList(workerTypes)
	for _, workerType := range workerTypes {
		if workerTypeMatchesRole(workerType, requiredRole) {
			readiness.Matched = true
			readiness.MatchedTypes = append(readiness.MatchedTypes, workerType)
		}
	}
	if !readiness.Matched {
		readiness.Status = "blocked"
		readiness.BlockedReasons = []string{"missing_agent_role:" + requiredRole}
	}
	return readiness
}

func workerTypeMatchesRole(workerType string, requiredRole string) bool {
	workerType = strings.TrimSpace(workerType)
	requiredRole = strings.TrimSpace(requiredRole)
	if workerType == requiredRole {
		return true
	}
	if requiredRole == "local_worker" && workerType == "local_host" {
		return true
	}
	return false
}

func normalizeStringList(values []string) []string {
	seen := map[string]bool{}
	normalized := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

func resourceReadinessFromLimits(limits map[string]any, activeLeases int64, queuedTasks int64) ResourceReadiness {
	readiness := ResourceReadiness{
		MaxActiveLeases: int64FromLimit(limits, "max_active_leases"),
		MaxQueuedTasks:  int64FromLimit(limits, "max_queued_tasks"),
		Status:          "ready",
	}
	if readiness.MaxActiveLeases > 0 && activeLeases >= readiness.MaxActiveLeases {
		readiness.BlockedReasons = append(readiness.BlockedReasons, "resource_limit:max_active_leases")
	}
	if readiness.MaxQueuedTasks > 0 && queuedTasks > readiness.MaxQueuedTasks {
		readiness.BlockedReasons = append(readiness.BlockedReasons, "resource_limit:max_queued_tasks")
	}
	if len(readiness.BlockedReasons) > 0 {
		readiness.Status = "blocked"
	}
	return readiness
}

func int64FromLimit(limits map[string]any, key string) int64 {
	if limits == nil {
		return 0
	}
	switch value := limits[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func metadataInt64(metadata map[string]any, key string) int64 {
	if metadata == nil {
		return 0
	}
	switch value := metadata[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func int64SliceFromAny(value any) []int64 {
	switch typed := value.(type) {
	case []int64:
		return typed
	case []any:
		out := make([]int64, 0, len(typed))
		for _, item := range typed {
			if parsed := anyInt64(item); parsed != 0 {
				out = append(out, parsed)
			}
		}
		return out
	default:
		return []int64{}
	}
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if parsed, ok := item.(string); ok && strings.TrimSpace(parsed) != "" {
				out = append(out, strings.TrimSpace(parsed))
			}
		}
		return out
	default:
		return []string{}
	}
}

func anyInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

func engineReadinessFromConfig(cfg Config) EngineReadiness {
	profile, ok := engineProfileByID(cfg.Engines, cfg.Scheduling.EngineProfile)
	if !ok {
		profileID := strings.TrimSpace(cfg.Scheduling.EngineProfile)
		if profileID == "" {
			profileID = strings.TrimSpace(cfg.Engines.Default)
		}
		return EngineReadiness{
			ProfileID:      profileID,
			SecretRef:      "none",
			SecretReady:    true,
			ResourceLimits: map[string]any{},
			Status:         "blocked",
			BlockedReasons: []string{"engine_profile_missing"},
		}
	}
	return engineReadinessFromProfile(profile)
}

func engineReadinessFromProfile(profile EngineProfileConfig) EngineReadiness {
	secretRef := strings.TrimSpace(profile.SecretRef)
	secretRequired := secretRef != "" && secretRef != "none"
	secretReady := !secretRequired
	readiness := EngineReadiness{
		ProfileID:      strings.TrimSpace(profile.ID),
		Provider:       strings.TrimSpace(profile.Provider),
		Enabled:        profile.Enabled,
		SecretRef:      secretRef,
		SecretRequired: secretRequired,
		SecretReady:    secretReady,
		ResourceLimits: profile.ResourceLimits,
		Status:         "ready",
	}
	if readiness.SecretRef == "" {
		readiness.SecretRef = "none"
	}
	if readiness.ResourceLimits == nil {
		readiness.ResourceLimits = map[string]any{}
	}
	if !readiness.Enabled {
		readiness.BlockedReasons = append(readiness.BlockedReasons, "engine_profile_disabled")
	}
	if readiness.SecretRequired && !readiness.SecretReady {
		readiness.BlockedReasons = append(readiness.BlockedReasons, "secret_ref_unavailable")
	}
	if len(readiness.BlockedReasons) > 0 {
		readiness.Status = "blocked"
	}
	return readiness
}

func engineReadinessFromSchedulingMetadata(engineProfile string, raw []byte) EngineReadiness {
	var metadata struct {
		Engine EngineReadiness `json:"engine"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &metadata)
	}
	return normalizeEngineReadiness(metadata.Engine, engineProfile)
}

func normalizeEngineReadiness(readiness EngineReadiness, fallbackProfile string) EngineReadiness {
	if readiness.ProfileID == "" {
		readiness.ProfileID = strings.TrimSpace(fallbackProfile)
	}
	if readiness.SecretRef == "" {
		readiness.SecretRef = "none"
	}
	if readiness.ResourceLimits == nil {
		readiness.ResourceLimits = map[string]any{}
	}
	if readiness.Status == "" {
		readiness.Status = "ready"
	}
	if len(readiness.BlockedReasons) > 0 {
		readiness.Status = "blocked"
	}
	return readiness
}

func nonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func minInt64(left int64, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func sortWorkerPoolSchedule(items []WorkerPoolProjectSchedule) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Recommended != items[j].Recommended {
			return items[i].Recommended
		}
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		if items[i].QueuedTasks != items[j].QueuedTasks {
			return items[i].QueuedTasks > items[j].QueuedTasks
		}
		return items[i].Project.Key < items[j].Project.Key
	})
}

func (s Store) AcquireLease(ctx context.Context, record Record, options AcquireLeaseOptions) (LeaseRecord, error) {
	options = normalizeAcquireLeaseOptions(options)
	requestHash, err := leaseAcquireRequestHash(record, options)
	if err != nil {
		return LeaseRecord{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = leaseAcquireIdempotencyKey(record, options, requestHash)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("begin lease acquire: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, "lease.acquire", options.IdempotencyKey, requestHash)
	if err != nil {
		return LeaseRecord{}, err
	}
	if !created {
		lease, err := loadLeaseByCommandResponse(ctx, tx, record.ID, "lease.acquire", options.IdempotencyKey)
		if err != nil {
			return LeaseRecord{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return LeaseRecord{}, fmt.Errorf("commit idempotent lease acquire: %w", err)
		}
		return lease, nil
	}

	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return LeaseRecord{}, err
	}
	if missing := missingWorkerCapabilities(worker.Capabilities, options.AllowedCapabilities); len(missing) > 0 {
		if err := recordWorkerCapabilityDenied(ctx, tx, record.ID, worker, "lease.acquire", fmt.Sprintf("%d", options.RunTaskID), options.Reason, options.AllowedCapabilities, missing, map[string]any{
			"run_task_id": options.RunTaskID,
			"lease_kind":  options.LeaseKind,
		}); err != nil {
			return LeaseRecord{}, err
		}
		if err := completeCommandRequestResponse(ctx, tx, record.ID, "lease.acquire", options.IdempotencyKey, leaseCommandResponse(map[string]any{
			"decision":             "denied",
			"worker_id":            worker.ID,
			"worker_key":           worker.WorkerKey,
			"run_task_id":          options.RunTaskID,
			"lease_kind":           options.LeaseKind,
			"missing_capabilities": missing,
			"lease_created":        false,
		})); err != nil {
			return LeaseRecord{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return LeaseRecord{}, fmt.Errorf("commit lease capability denial: %w", err)
		}
		return LeaseRecord{}, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}
	if options.RecoverExpiredBefore {
		if _, err := recoverExpiredLeases(ctx, tx, record.ID, 50, options.Metadata); err != nil {
			return LeaseRecord{}, err
		}
	}

	task, err := loadRunTaskForLease(ctx, tx, record.ID, options.RunTaskID)
	if err != nil {
		return LeaseRecord{}, err
	}
	lease, err := insertLeaseForTask(ctx, tx, record.ID, worker, task, options.LeaseKind, options.AllowedCapabilities, options.Scope, options.Metadata, options.LeaseTimeoutSeconds)
	if err != nil {
		return LeaseRecord{}, err
	}

	if err := updateRunTaskStatus(ctx, tx, task.ID, "leased"); err != nil {
		return LeaseRecord{}, err
	}
	if _, err := insertWorkerEvent(ctx, tx, record.ID, worker.ID, "lease.acquired", "Worker lease acquired", map[string]any{
		"worker_key":  worker.WorkerKey,
		"lease_id":    lease.ID,
		"run_id":      lease.RunID,
		"run_task_id": lease.RunTaskID,
		"lease_kind":  lease.LeaseKind,
	}); err != nil {
		return LeaseRecord{}, err
	}
	if _, err := insertWorkerAuditEvent(ctx, tx, record.ID, worker.ActorID, "lease.acquire", "manage_workers", fmt.Sprintf("%d", lease.ID), "allowed", options.Reason, map[string]any{
		"worker_id":   worker.ID,
		"worker_key":  worker.WorkerKey,
		"run_task_id": lease.RunTaskID,
		"lease_kind":  lease.LeaseKind,
	}); err != nil {
		return LeaseRecord{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "lease.acquire", options.IdempotencyKey, leaseCommandResponse(map[string]any{
		"decision":      "allowed",
		"lease_id":      lease.ID,
		"run_id":        lease.RunID,
		"run_task_id":   lease.RunTaskID,
		"worker_id":     worker.ID,
		"worker_key":    worker.WorkerKey,
		"lease_kind":    lease.LeaseKind,
		"lease_status":  lease.Status,
		"lease_created": true,
	})); err != nil {
		return LeaseRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LeaseRecord{}, fmt.Errorf("commit lease acquire: %w", err)
	}
	return lease, nil
}

func (s Store) ReleaseLease(ctx context.Context, record Record, options ReleaseLeaseOptions) (LeaseRecord, error) {
	options = normalizeReleaseLeaseOptions(options)
	requestHash, err := leaseReleaseRequestHash(record, options)
	if err != nil {
		return LeaseRecord{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = leaseReleaseIdempotencyKey(record, options, requestHash)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("begin lease release: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, "lease.release", options.IdempotencyKey, requestHash)
	if err != nil {
		return LeaseRecord{}, err
	}
	if !created {
		lease, err := loadLeaseByCommandResponse(ctx, tx, record.ID, "lease.release", options.IdempotencyKey)
		if err != nil {
			return LeaseRecord{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return LeaseRecord{}, fmt.Errorf("commit idempotent lease release: %w", err)
		}
		return lease, nil
	}

	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return LeaseRecord{}, err
	}
	lease, err := releaseLeaseForWorker(ctx, tx, record.ID, worker, options.LeaseID, options.Status, options.Metadata)
	if err != nil {
		return LeaseRecord{}, err
	}
	if lease.RunTaskID != 0 {
		status := "queued"
		if options.Status == "completed" {
			status = "passed"
		}
		if err := updateRunTaskStatus(ctx, tx, lease.RunTaskID, status); err != nil {
			return LeaseRecord{}, err
		}
	}
	if _, err := insertWorkerEvent(ctx, tx, record.ID, worker.ID, "lease.released", "Worker lease released", map[string]any{
		"worker_key":  worker.WorkerKey,
		"lease_id":    lease.ID,
		"run_id":      lease.RunID,
		"run_task_id": lease.RunTaskID,
		"status":      lease.Status,
	}); err != nil {
		return LeaseRecord{}, err
	}
	if _, err := insertWorkerAuditEvent(ctx, tx, record.ID, worker.ActorID, "lease.release", "manage_workers", fmt.Sprintf("%d", lease.ID), "allowed", options.Reason, map[string]any{
		"worker_id":   worker.ID,
		"worker_key":  worker.WorkerKey,
		"run_task_id": lease.RunTaskID,
		"status":      lease.Status,
	}); err != nil {
		return LeaseRecord{}, err
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "lease.release", options.IdempotencyKey, leaseCommandResponse(map[string]any{
		"decision":      "allowed",
		"lease_id":      lease.ID,
		"run_id":        lease.RunID,
		"run_task_id":   lease.RunTaskID,
		"worker_id":     worker.ID,
		"worker_key":    worker.WorkerKey,
		"lease_kind":    lease.LeaseKind,
		"lease_status":  lease.Status,
		"lease_created": false,
	})); err != nil {
		return LeaseRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LeaseRecord{}, fmt.Errorf("commit lease release: %w", err)
	}
	return lease, nil
}

func (s Store) RecoverExpiredLeases(ctx context.Context, record Record, options RecoverLeasesOptions) ([]LeaseRecord, error) {
	options = normalizeRecoverLeasesOptions(options)
	requestHash, err := leaseRecoverRequestHash(record, options)
	if err != nil {
		return nil, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = leaseRecoverIdempotencyKey(record, options, requestHash)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin lease recovery: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, "lease.recover", options.IdempotencyKey, requestHash)
	if err != nil {
		return nil, err
	}
	if !created {
		leases, err := loadLeasesByCommandResponse(ctx, tx, record.ID, "lease.recover", options.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit idempotent lease recovery: %w", err)
		}
		return leases, nil
	}

	leases, err := recoverExpiredLeases(ctx, tx, record.ID, options.Limit, options.Metadata)
	if err != nil {
		return nil, err
	}
	leaseIDs := make([]int64, 0, len(leases))
	for _, lease := range leases {
		leaseIDs = append(leaseIDs, lease.ID)
		if lease.RunTaskID != 0 {
			if err := updateRunTaskStatus(ctx, tx, lease.RunTaskID, "needs_recovery"); err != nil {
				return nil, err
			}
		}
		if _, err := insertWorkerEvent(ctx, tx, record.ID, lease.WorkerID, "lease.recovered", "Expired lease marked for recovery", map[string]any{
			"lease_id":    lease.ID,
			"run_id":      lease.RunID,
			"run_task_id": lease.RunTaskID,
			"status":      lease.Status,
		}); err != nil {
			return nil, err
		}
		if _, err := insertWorkerAuditEvent(ctx, tx, record.ID, 0, "lease.recover", "manage_workers", fmt.Sprintf("%d", lease.ID), "allowed", options.Reason, map[string]any{
			"run_task_id": lease.RunTaskID,
			"status":      lease.Status,
		}); err != nil {
			return nil, err
		}
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, "lease.recover", options.IdempotencyKey, leaseCommandResponse(map[string]any{
		"decision":      "allowed",
		"lease_ids":     leaseIDs,
		"count":         len(leaseIDs),
		"lease_created": false,
	})); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit lease recovery: %w", err)
	}
	return leases, nil
}

func (s Store) RunWorkerOnce(ctx context.Context, record Record, options WorkerRunOnceOptions) (WorkerRunOnceResult, error) {
	options = normalizeWorkerRunOnceOptions(options)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkerRunOnceResult{}, fmt.Errorf("begin worker run once: %w", err)
	}
	defer tx.Rollback(ctx)

	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return WorkerRunOnceResult{}, err
	}
	if missing := missingWorkerCapabilities(worker.Capabilities, options.AllowedCapabilities); len(missing) > 0 {
		if err := recordWorkerCapabilityDenied(ctx, tx, record.ID, worker, "worker.run_once", worker.WorkerKey, options.Reason, options.AllowedCapabilities, missing, map[string]any{
			"run_once": true,
			"run_id":   options.RunID,
		}); err != nil {
			return WorkerRunOnceResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return WorkerRunOnceResult{}, fmt.Errorf("commit worker run-once capability denial: %w", err)
		}
		return WorkerRunOnceResult{}, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}
	if _, err := recoverExpiredLeases(ctx, tx, record.ID, 50, map[string]any{"trigger": "worker_run_once"}); err != nil {
		return WorkerRunOnceResult{}, err
	}
	task, ok, err := nextDryRunTaskForLease(ctx, tx, record.ID, options.RunID)
	if err != nil {
		return WorkerRunOnceResult{}, err
	}
	if !ok {
		if _, err := insertWorkerEvent(ctx, tx, record.ID, worker.ID, "worker.run_once.idle", "Worker run-once found no dry-run task", map[string]any{
			"worker_key": worker.WorkerKey,
			"run_id":     options.RunID,
		}); err != nil {
			return WorkerRunOnceResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return WorkerRunOnceResult{}, fmt.Errorf("commit worker run once idle: %w", err)
		}
		return WorkerRunOnceResult{
			Project: record,
			Worker:  worker,
			Claimed: false,
		}, nil
	}

	lease, err := insertLeaseForTask(ctx, tx, record.ID, worker, task, "run_task", options.AllowedCapabilities, map[string]any{
		"run_task_id": task.ID,
		"run_id":      task.RunID,
		"task_key":    task.TaskKey,
		"task_kind":   task.TaskKind,
		"run_once":    true,
	}, options.Metadata, options.LeaseTimeoutSeconds)
	if err != nil {
		return WorkerRunOnceResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "leased"); err != nil {
		return WorkerRunOnceResult{}, err
	}
	report, err := writeAndInsertWorkerRunOnceArtifact(ctx, tx, record, worker, task, lease, options)
	if err != nil {
		return WorkerRunOnceResult{}, err
	}
	attempt, err := insertWorkerRunOnceAttempt(ctx, tx, record, task, lease, report, options)
	if err != nil {
		return WorkerRunOnceResult{}, err
	}
	released, err := releaseLeaseForWorker(ctx, tx, record.ID, worker, lease.ID, "completed", map[string]any{
		"run_once":    true,
		"dry_run":     true,
		"attempt_id":  attempt.ID,
		"artifact_id": report.ID,
	})
	if err != nil {
		return WorkerRunOnceResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "passed"); err != nil {
		return WorkerRunOnceResult{}, err
	}
	task.Status = "passed"
	if _, err := insertWorkerEvent(ctx, tx, record.ID, worker.ID, "worker.run_once.completed", "Worker run-once dry-run completed", map[string]any{
		"worker_key":  worker.WorkerKey,
		"lease_id":    released.ID,
		"run_id":      released.RunID,
		"run_task_id": released.RunTaskID,
		"attempt_id":  attempt.ID,
		"artifact_id": report.ID,
	}); err != nil {
		return WorkerRunOnceResult{}, err
	}
	if _, err := insertWorkerAuditEvent(ctx, tx, record.ID, worker.ActorID, "worker.run_once", "manage_workers", worker.WorkerKey, "allowed", options.Reason, map[string]any{
		"lease_id":    released.ID,
		"run_id":      released.RunID,
		"run_task_id": released.RunTaskID,
		"attempt_id":  attempt.ID,
		"artifact_id": report.ID,
		"dry_run":     true,
	}); err != nil {
		return WorkerRunOnceResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkerRunOnceResult{}, fmt.Errorf("commit worker run once: %w", err)
	}
	return WorkerRunOnceResult{
		Project:  record,
		Worker:   worker,
		Lease:    released,
		Task:     task,
		Attempt:  attempt,
		Artifact: report,
		Claimed:  true,
	}, nil
}

func normalizeRegisterWorkerOptions(record Record, options RegisterWorkerOptions) RegisterWorkerOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.WorkerType = strings.TrimSpace(options.WorkerType)
	options.Hostname = strings.TrimSpace(options.Hostname)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.WorkerKey == "" {
		options.WorkerKey = fmt.Sprintf("%s-local", record.Key)
	}
	if options.WorkerType == "" {
		options.WorkerType = "local_host"
	}
	if options.Hostname == "" {
		if hostname, err := os.Hostname(); err == nil {
			options.Hostname = hostname
		}
	}
	if len(options.Capabilities) == 0 {
		options.Capabilities = []string{"read_project", "write_artifacts", "execute_agents"}
	}
	options.Capabilities = normalizeCapabilityList(options.Capabilities)
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.HeartbeatIntervalSeconds <= 0 {
		options.HeartbeatIntervalSeconds = 30
	}
	if options.LeaseTimeoutSeconds <= 0 {
		options.LeaseTimeoutSeconds = 300
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "worker registry update"
	}
	return options
}

func normalizeWorkerHeartbeatOptions(options WorkerHeartbeatOptions) WorkerHeartbeatOptions {
	options.Status = strings.TrimSpace(options.Status)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.Status == "" {
		options.Status = "online"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "worker heartbeat"
	}
	return options
}

func normalizeAcquireLeaseOptions(options AcquireLeaseOptions) AcquireLeaseOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.LeaseKind = strings.TrimSpace(options.LeaseKind)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.LeaseKind == "" {
		options.LeaseKind = "run_task"
	}
	if len(options.AllowedCapabilities) == 0 {
		options.AllowedCapabilities = []string{"read_project", "write_artifacts", "execute_agents"}
	}
	options.AllowedCapabilities = normalizeCapabilityList(options.AllowedCapabilities)
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.Scope == nil {
		options.Scope = map[string]any{}
	}
	if options.LeaseTimeoutSeconds <= 0 {
		options.LeaseTimeoutSeconds = 300
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "lease acquire"
	}
	return options
}

func normalizeReleaseLeaseOptions(options ReleaseLeaseOptions) ReleaseLeaseOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.Status = strings.TrimSpace(options.Status)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.Status == "" {
		options.Status = "released"
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "lease release"
	}
	return options
}

func normalizeRecoverLeasesOptions(options RecoverLeasesOptions) RecoverLeasesOptions {
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.Limit <= 0 {
		options.Limit = 20
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "lease recovery"
	}
	return options
}

func leaseAcquireRequestHash(record Record, options AcquireLeaseOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":           "lease.acquire",
		"project_key":            record.Key,
		"worker_key":             options.WorkerKey,
		"run_task_id":            options.RunTaskID,
		"lease_kind":             options.LeaseKind,
		"allowed_capabilities":   options.AllowedCapabilities,
		"scope":                  options.Scope,
		"metadata":               options.Metadata,
		"lease_timeout_seconds":  options.LeaseTimeoutSeconds,
		"recover_expired_before": options.RecoverExpiredBefore,
		"actor":                  options.Actor,
		"reason":                 options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal lease acquire request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func leaseReleaseRequestHash(record Record, options ReleaseLeaseOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": "lease.release",
		"project_key":  record.Key,
		"worker_key":   options.WorkerKey,
		"lease_id":     options.LeaseID,
		"status":       options.Status,
		"metadata":     options.Metadata,
		"actor":        options.Actor,
		"reason":       options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal lease release request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func leaseRecoverRequestHash(record Record, options RecoverLeasesOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": "lease.recover",
		"project_key":  record.Key,
		"limit":        options.Limit,
		"metadata":     options.Metadata,
		"actor":        options.Actor,
		"reason":       options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal lease recover request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func workerRegisterRequestHash(record Record, options RegisterWorkerOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":               "worker.register",
		"project_key":                record.Key,
		"worker_key":                 options.WorkerKey,
		"worker_type":                options.WorkerType,
		"hostname":                   options.Hostname,
		"pid":                        options.PID,
		"capabilities":               options.Capabilities,
		"metadata":                   options.Metadata,
		"heartbeat_interval_seconds": options.HeartbeatIntervalSeconds,
		"lease_timeout_seconds":      options.LeaseTimeoutSeconds,
		"actor":                      options.Actor,
		"reason":                     options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal worker register request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func workerHeartbeatRequestHash(record Record, workerKey string, options WorkerHeartbeatOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": "worker.heartbeat",
		"project_key":  record.Key,
		"worker_key":   strings.TrimSpace(workerKey),
		"status":       options.Status,
		"metadata":     options.Metadata,
		"actor":        options.Actor,
		"reason":       options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal worker heartbeat request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func leaseAcquireIdempotencyKey(record Record, options AcquireLeaseOptions, requestHash string) string {
	return fmt.Sprintf("lease.acquire:%s:%s:%d:%s", record.Key, options.WorkerKey, options.RunTaskID, commandHashPrefix(requestHash))
}

func leaseReleaseIdempotencyKey(record Record, options ReleaseLeaseOptions, requestHash string) string {
	return fmt.Sprintf("lease.release:%s:%s:%d:%s:%s", record.Key, options.WorkerKey, options.LeaseID, options.Status, commandHashPrefix(requestHash))
}

func leaseRecoverIdempotencyKey(record Record, options RecoverLeasesOptions, requestHash string) string {
	sequence := leaseRecoverDefaultKeySequence.Add(1)
	return fmt.Sprintf("lease.recover:%s:%d:%d:%d:%s", record.Key, options.Limit, time.Now().UTC().UnixNano(), sequence, commandHashPrefix(requestHash))
}

func workerRegisterIdempotencyKey(record Record, options RegisterWorkerOptions, requestHash string) string {
	sequence := workerLifecycleDefaultKeySequence.Add(1)
	return fmt.Sprintf("worker.register:%s:%s:%d:%d:%s", record.Key, options.WorkerKey, time.Now().UTC().UnixNano(), sequence, commandHashPrefix(requestHash))
}

func workerHeartbeatIdempotencyKey(record Record, workerKey string, _ WorkerHeartbeatOptions, requestHash string) string {
	sequence := workerLifecycleDefaultKeySequence.Add(1)
	return fmt.Sprintf("worker.heartbeat:%s:%s:%d:%d:%s", record.Key, strings.TrimSpace(workerKey), time.Now().UTC().UnixNano(), sequence, commandHashPrefix(requestHash))
}

func commandHashPrefix(requestHash string) string {
	requestHash = strings.TrimSpace(requestHash)
	if len(requestHash) > 16 {
		return requestHash[:16]
	}
	if requestHash == "" {
		return "no-request-hash"
	}
	return requestHash
}

func leaseCommandResponse(response map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range response {
		out[key] = value
	}
	if _, exists := out["lease_created"]; !exists {
		out["lease_created"] = false
	}
	out["project_write_attempted"] = false
	out["execution_write_attempted"] = false
	out["engine_call_attempted"] = false
	out["commands_run"] = false
	out["secrets_resolved"] = false
	out["network_used"] = false
	out["attempt_created"] = false
	out["artifact_created"] = false
	out["worker_run_once"] = false
	return out
}

func workerLifecycleCommandResponse(response map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range response {
		out[key] = value
	}
	out["project_write_attempted"] = false
	out["execution_write_attempted"] = false
	out["engine_call_attempted"] = false
	out["commands_run"] = false
	out["secrets_resolved"] = false
	out["network_used"] = false
	out["lease_created"] = false
	out["attempt_created"] = false
	out["artifact_created"] = false
	out["worker_run_once"] = false
	return out
}

func normalizeWorkerRunOnceOptions(options WorkerRunOnceOptions) WorkerRunOnceOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.RunID < 0 {
		options.RunID = 0
	}
	if len(options.AllowedCapabilities) == 0 {
		options.AllowedCapabilities = []string{"read_project", "write_artifacts", "execute_agents"}
	}
	options.AllowedCapabilities = normalizeCapabilityList(options.AllowedCapabilities)
	if options.LeaseTimeoutSeconds <= 0 {
		options.LeaseTimeoutSeconds = 300
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "worker run once"
	}
	return options
}

func loadWorkerForUpdate(ctx context.Context, tx pgx.Tx, projectID int64, workerKey string) (WorkerRecord, error) {
	if strings.TrimSpace(workerKey) == "" {
		return WorkerRecord{}, fmt.Errorf("worker key is required")
	}
	worker, err := scanWorker(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
       COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
       registered_at, last_heartbeat_at, heartbeat_interval_seconds,
       lease_timeout_seconds, updated_at
FROM workers
WHERE project_id = $1 AND worker_key = $2
FOR UPDATE`,
		projectID,
		workerKey,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkerRecord{}, ErrWorkerNotFound
		}
		return WorkerRecord{}, fmt.Errorf("load worker: %w", err)
	}
	return worker, nil
}

func loadRunTaskForLease(ctx context.Context, tx pgx.Tx, projectID int64, runTaskID int64) (RunTaskRecord, error) {
	if runTaskID <= 0 {
		return RunTaskRecord{}, fmt.Errorf("run task id is required")
	}
	task, err := scanRunTask(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       run_id, task_key, task_kind, status, risk_level, sequence, metadata,
       created_at, updated_at
FROM run_tasks
WHERE project_id = $1 AND id = $2
FOR UPDATE`,
		projectID,
		runTaskID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunTaskRecord{}, ErrNoLeaseAvailable
		}
		return RunTaskRecord{}, fmt.Errorf("load run task for lease: %w", err)
	}
	return task, nil
}

func nextDryRunTaskForLease(ctx context.Context, tx pgx.Tx, projectID int64, runID int64) (RunTaskRecord, bool, error) {
	task, err := scanRunTask(tx.QueryRow(ctx, `
SELECT rt.id, rt.project_id, COALESCE(rt.workflow_version_id, 0), COALESCE(rt.workflow_item_id, 0),
       rt.run_id, rt.task_key, rt.task_kind, rt.status, rt.risk_level, rt.sequence, rt.metadata,
       rt.created_at, rt.updated_at
FROM run_tasks rt
JOIN runs r ON r.id = rt.run_id
WHERE rt.project_id = $1
  AND ($2 = 0 OR rt.run_id = $2)
  AND r.dry_run = true
  AND rt.status IN ('queued', 'needs_recovery')
  AND NOT EXISTS (
      SELECT 1
      FROM leases l
      WHERE l.run_task_id = rt.id
        AND l.status = 'active'
  )
ORDER BY rt.created_at ASC, rt.id ASC
LIMIT 1
FOR UPDATE SKIP LOCKED`,
		projectID,
		runID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunTaskRecord{}, false, nil
		}
		return RunTaskRecord{}, false, fmt.Errorf("load next dry-run task: %w", err)
	}
	return task, true, nil
}

func updateRunTaskStatus(ctx context.Context, tx pgx.Tx, runTaskID int64, status string) error {
	if runTaskID == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE run_tasks
SET status = $2,
    updated_at = now()
WHERE id = $1`,
		runTaskID,
		status,
	); err != nil {
		return fmt.Errorf("update run task status: %w", err)
	}
	return nil
}

func loadLeaseByCommandResponse(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string) (LeaseRecord, error) {
	response, err := loadCommandResponse(ctx, tx, projectID, commandType, idempotencyKey)
	if err != nil {
		return LeaseRecord{}, err
	}
	if decision := metadataString(response, "decision"); decision == "denied" {
		missing := stringSliceFromAny(response["missing_capabilities"])
		return LeaseRecord{}, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}
	leaseID := metadataInt64(response, "lease_id")
	if leaseID == 0 {
		return LeaseRecord{}, fmt.Errorf("lease id missing from command response: %s", idempotencyKey)
	}
	return loadLeaseByID(ctx, tx, projectID, leaseID)
}

func loadLeasesByCommandResponse(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string) ([]LeaseRecord, error) {
	response, err := loadCommandResponse(ctx, tx, projectID, commandType, idempotencyKey)
	if err != nil {
		return nil, err
	}
	leaseIDs := int64SliceFromAny(response["lease_ids"])
	if len(leaseIDs) == 0 {
		return []LeaseRecord{}, nil
	}
	leases := make([]LeaseRecord, 0, len(leaseIDs))
	for _, leaseID := range leaseIDs {
		lease, err := loadLeaseByID(ctx, tx, projectID, leaseID)
		if err != nil {
			return nil, err
		}
		leases = append(leases, lease)
	}
	return leases, nil
}

func loadWorkerByCommandResponse(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string) (WorkerRecord, error) {
	response, err := loadCommandResponse(ctx, tx, projectID, commandType, idempotencyKey)
	if err != nil {
		return WorkerRecord{}, err
	}
	workerID := metadataInt64(response, "worker_id")
	if workerID == 0 {
		return WorkerRecord{}, fmt.Errorf("worker id missing from command response: %s", idempotencyKey)
	}
	return loadWorkerByID(ctx, tx, projectID, workerID)
}

func loadWorkerByID(ctx context.Context, tx pgx.Tx, projectID int64, workerID int64) (WorkerRecord, error) {
	worker, err := scanWorker(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
       COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
       registered_at, last_heartbeat_at, heartbeat_interval_seconds,
       lease_timeout_seconds, updated_at
FROM workers
WHERE project_id = $1 AND id = $2`,
		projectID,
		workerID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkerRecord{}, ErrWorkerNotFound
		}
		return WorkerRecord{}, fmt.Errorf("load worker by id: %w", err)
	}
	return worker, nil
}

func loadLeaseByID(ctx context.Context, tx pgx.Tx, projectID int64, leaseID int64) (LeaseRecord, error) {
	lease, err := scanLease(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(run_id, 0), COALESCE(run_task_id, 0),
       COALESCE(workflow_item_id, 0), COALESCE(worker_id, 0), lease_kind,
       status, acquired_at, expires_at, heartbeat_at, released_at,
       allowed_capabilities, scope, metadata
FROM leases
WHERE project_id = $1 AND id = $2`,
		projectID,
		leaseID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LeaseRecord{}, ErrLeaseNotFound
		}
		return LeaseRecord{}, fmt.Errorf("load lease by id: %w", err)
	}
	return lease, nil
}

func writeAndInsertWorkerRunOnceArtifact(ctx context.Context, tx pgx.Tx, record Record, worker WorkerRecord, task RunTaskRecord, lease LeaseRecord, options WorkerRunOnceOptions) (ArtifactRecord, error) {
	if record.ArtifactBackend != "" && record.ArtifactBackend != "local" {
		return ArtifactRecord{}, fmt.Errorf("unsupported artifact store backend %q", record.ArtifactBackend)
	}
	content, err := json.MarshalIndent(map[string]any{
		"project":              record.Key,
		"worker_id":            worker.ID,
		"worker_key":           worker.WorkerKey,
		"run_id":               task.RunID,
		"run_task_id":          task.ID,
		"task_key":             task.TaskKey,
		"task_kind":            task.TaskKind,
		"lease_id":             lease.ID,
		"dry_run":              true,
		"writes_attempted":     false,
		"commands_run":         false,
		"secrets_resolved":     false,
		"network_used":         false,
		"executed_attempt":     false,
		"allowed_capabilities": options.AllowedCapabilities,
		"requested_run_id":     options.RunID,
	}, "", "  ")
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal worker run-once report: %w", err)
	}
	relativePath := filepath.Join("workers", worker.WorkerKey, "run-once", fmt.Sprintf("run-task-%d-report.json", task.ID))
	stored, err := writeLocalProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":            "v0.6d",
		"owned_by":         "areaflow",
		"dry_run":          true,
		"worker_id":        worker.ID,
		"worker_key":       worker.WorkerKey,
		"run_id":           task.RunID,
		"requested_run_id": options.RunID,
		"run_task_id":      task.ID,
		"lease_id":         lease.ID,
		"actor":            options.Actor,
		"reason":           options.Reason,
		"run_once":         true,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal worker run-once artifact metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, task.WorkflowVersionID, task.RunID, task.WorkflowItemID, "worker_run_once_report", relativePath, stored, string(metadata))
}

func insertWorkerRunOnceAttempt(ctx context.Context, tx pgx.Tx, record Record, task RunTaskRecord, lease LeaseRecord, report ArtifactRecord, options WorkerRunOnceOptions) (RunAttemptRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"actor":                options.Actor,
		"reason":               options.Reason,
		"dry_run":              true,
		"would_execute":        false,
		"writes_attempted":     false,
		"commands_run":         false,
		"secrets_resolved":     false,
		"network_used":         false,
		"lease_id":             lease.ID,
		"worker_id":            lease.WorkerID,
		"evidence_artifact_id": report.ID,
	})
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("marshal worker run-once attempt metadata: %w", err)
	}
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
INSERT INTO run_attempts (
    project_id, workflow_version_id, workflow_item_id, run_id, run_task_id,
    attempt_kind, status, dry_run, finished_at, metadata
)
VALUES ($1, $2, $3, $4, $5, 'worker_run_once', 'passed', true, now(), $6::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, COALESCE(run_task_id, 0), attempt_kind, status, dry_run,
          metadata, started_at, finished_at`,
		record.ID,
		nullableInt64(task.WorkflowVersionID),
		nullableInt64(task.WorkflowItemID),
		task.RunID,
		task.ID,
		string(metadata),
	))
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("insert worker run-once attempt: %w", err)
	}
	return attempt, nil
}

func insertLeaseForTask(ctx context.Context, tx pgx.Tx, projectID int64, worker WorkerRecord, task RunTaskRecord, leaseKind string, capabilities []string, scope map[string]any, metadata map[string]any, timeoutSeconds int) (LeaseRecord, error) {
	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("marshal lease capabilities: %w", err)
	}
	scopePayload := map[string]any{}
	for key, value := range scope {
		scopePayload[key] = value
	}
	scopePayload["run_task_id"] = task.ID
	scopePayload["task_key"] = task.TaskKey
	scopePayload["task_kind"] = task.TaskKind
	scopeJSON, err := json.Marshal(scopePayload)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("marshal lease scope: %w", err)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("marshal lease metadata: %w", err)
	}
	lease, err := scanLease(tx.QueryRow(ctx, `
INSERT INTO leases (
    project_id, run_id, run_task_id, workflow_item_id, worker_id, lease_kind,
    status, expires_at, heartbeat_at, allowed_capabilities, scope, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, 'active', now() + make_interval(secs => $7),
        now(), $8::jsonb, $9::jsonb, $10::jsonb)
RETURNING id, project_id, COALESCE(run_id, 0), COALESCE(run_task_id, 0),
          COALESCE(workflow_item_id, 0), COALESCE(worker_id, 0), lease_kind,
          status, acquired_at, expires_at, heartbeat_at, released_at,
          allowed_capabilities, scope, metadata`,
		projectID,
		task.RunID,
		task.ID,
		nullableInt64(task.WorkflowItemID),
		worker.ID,
		leaseKind,
		timeoutSeconds,
		string(capabilitiesJSON),
		string(scopeJSON),
		string(metadataJSON),
	))
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("insert lease: %w", err)
	}
	return lease, nil
}

func releaseLeaseForWorker(ctx context.Context, tx pgx.Tx, projectID int64, worker WorkerRecord, leaseID int64, status string, metadata map[string]any) (LeaseRecord, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("marshal lease release metadata: %w", err)
	}
	lease, err := scanLease(tx.QueryRow(ctx, `
UPDATE leases
SET status = $4,
    released_at = now(),
    metadata = metadata || $5::jsonb
WHERE project_id = $1
  AND id = $2
  AND worker_id = $3
  AND status = 'active'
RETURNING id, project_id, COALESCE(run_id, 0), COALESCE(run_task_id, 0),
          COALESCE(workflow_item_id, 0), COALESCE(worker_id, 0), lease_kind,
          status, acquired_at, expires_at, heartbeat_at, released_at,
          allowed_capabilities, scope, metadata`,
		projectID,
		leaseID,
		worker.ID,
		status,
		string(metadataJSON),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LeaseRecord{}, ErrLeaseNotFound
		}
		return LeaseRecord{}, fmt.Errorf("release lease: %w", err)
	}
	return lease, nil
}

func recoverExpiredLeases(ctx context.Context, tx pgx.Tx, projectID int64, limit int, metadata map[string]any) ([]LeaseRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal lease recovery metadata: %w", err)
	}
	rows, err := tx.Query(ctx, `
WITH expired AS (
    SELECT id
    FROM leases
    WHERE project_id = $1
      AND status = 'active'
      AND expires_at < now()
    ORDER BY expires_at ASC, id ASC
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE leases
SET status = 'needs_recovery',
    released_at = now(),
    metadata = metadata || $3::jsonb
WHERE id IN (SELECT id FROM expired)
RETURNING id, project_id, COALESCE(run_id, 0), COALESCE(run_task_id, 0),
          COALESCE(workflow_item_id, 0), COALESCE(worker_id, 0), lease_kind,
          status, acquired_at, expires_at, heartbeat_at, released_at,
          allowed_capabilities, scope, metadata`,
		projectID,
		limit,
		string(metadataJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("recover expired leases: %w", err)
	}
	defer rows.Close()
	leases := []LeaseRecord{}
	for rows.Next() {
		lease, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		leases = append(leases, lease)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recovered leases: %w", err)
	}
	return leases, nil
}

func ensureWorkerActor(ctx context.Context, tx pgx.Tx, record Record, workerKey string) (int64, error) {
	externalKey := fmt.Sprintf("worker:%s:%s", record.Key, workerKey)
	var actorID int64
	err := tx.QueryRow(ctx, `SELECT id FROM actors WHERE external_key = $1`, externalKey).Scan(&actorID)
	if err == nil {
		return actorID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("load worker actor: %w", err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO actors (kind, display_name, external_key)
VALUES ('worker', $1, $2)
RETURNING id`,
		workerKey,
		externalKey,
	).Scan(&actorID); err != nil {
		return 0, fmt.Errorf("insert worker actor: %w", err)
	}
	return actorID, nil
}

func normalizeCapabilityList(capabilities []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" || seen[capability] {
			continue
		}
		seen[capability] = true
		normalized = append(normalized, capability)
	}
	return normalized
}

func missingWorkerCapabilities(workerCapabilities []string, requestedCapabilities []string) []string {
	allowed := map[string]bool{}
	for _, capability := range workerCapabilities {
		capability = strings.TrimSpace(capability)
		if capability != "" {
			allowed[capability] = true
		}
	}
	missing := []string{}
	for _, capability := range normalizeCapabilityList(requestedCapabilities) {
		if !allowed[capability] {
			missing = append(missing, capability)
		}
	}
	return missing
}

func capabilityPresent(capabilities []string, required string) bool {
	required = strings.TrimSpace(required)
	for _, capability := range capabilities {
		if strings.TrimSpace(capability) == required {
			return true
		}
	}
	return false
}

func recordWorkerCapabilityDenied(ctx context.Context, tx pgx.Tx, projectID int64, worker WorkerRecord, action string, resource string, reason string, requested []string, missing []string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["worker_key"] = worker.WorkerKey
	metadata["worker_capabilities"] = worker.Capabilities
	metadata["requested_capabilities"] = requested
	metadata["missing_capabilities"] = missing
	if _, err := insertWorkerEvent(ctx, tx, projectID, worker.ID, action+".denied", "Worker capability denied", metadata); err != nil {
		return err
	}
	_, err := insertWorkerAuditEvent(ctx, tx, projectID, worker.ActorID, action, "manage_workers", resource, "denied", reason, metadata)
	return err
}

func insertWorkerHeartbeat(ctx context.Context, tx pgx.Tx, projectID int64, workerID int64, status string, metadata map[string]any) (int64, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal worker heartbeat metadata: %w", err)
	}
	var heartbeatID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO worker_heartbeats (project_id, worker_id, status, metadata)
VALUES ($1, $2, $3, $4::jsonb)
RETURNING id`,
		projectID,
		workerID,
		status,
		string(metadataJSON),
	).Scan(&heartbeatID); err != nil {
		return 0, fmt.Errorf("insert worker heartbeat: %w", err)
	}
	return heartbeatID, nil
}

func insertWorkerEvent(ctx context.Context, tx pgx.Tx, projectID int64, workerID int64, eventType string, message string, metadata map[string]any) (int64, error) {
	metadata["worker_id"] = workerID
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal worker event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'info', $3, $4::jsonb)
RETURNING id`,
		projectID,
		eventType,
		message,
		string(metadataJSON),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert worker event: %w", err)
	}
	return eventID, nil
}

func insertWorkerAuditEvent(ctx context.Context, tx pgx.Tx, projectID int64, actorID int64, action string, capability string, resource string, decision string, reason string, metadata map[string]any) (int64, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal worker audit metadata: %w", err)
	}
	var nullableActor any
	if actorID != 0 {
		nullableActor = actorID
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, $3, $4, 'worker', $5, $6, $7, $8::jsonb)
RETURNING id`,
		projectID,
		nullableActor,
		action,
		capability,
		resource,
		decision,
		reason,
		string(metadataJSON),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert worker audit event: %w", err)
	}
	return auditEventID, nil
}

func nullablePID(pid int) any {
	if pid <= 0 {
		return nil
	}
	return pid
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func scanWorker(row scanner) (WorkerRecord, error) {
	var worker WorkerRecord
	var capabilitiesRaw []byte
	var metadataRaw []byte
	var lastHeartbeat sql.NullTime
	if err := row.Scan(
		&worker.ID,
		&worker.ProjectID,
		&worker.ActorID,
		&worker.WorkerKey,
		&worker.WorkerType,
		&worker.Status,
		&worker.Hostname,
		&worker.PID,
		&capabilitiesRaw,
		&metadataRaw,
		&worker.RegisteredAt,
		&lastHeartbeat,
		&worker.HeartbeatIntervalSeconds,
		&worker.LeaseTimeoutSeconds,
		&worker.UpdatedAt,
	); err != nil {
		return WorkerRecord{}, err
	}
	if err := json.Unmarshal(capabilitiesRaw, &worker.Capabilities); err != nil {
		return WorkerRecord{}, fmt.Errorf("parse worker capabilities: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &worker.Metadata); err != nil {
		return WorkerRecord{}, fmt.Errorf("parse worker metadata: %w", err)
	}
	if lastHeartbeat.Valid {
		worker.LastHeartbeatAt = &lastHeartbeat.Time
	}
	return worker, nil
}

func scanLease(row scanner) (LeaseRecord, error) {
	var lease LeaseRecord
	var capabilitiesRaw []byte
	var scopeRaw []byte
	var metadataRaw []byte
	var heartbeatAt sql.NullTime
	var releasedAt sql.NullTime
	if err := row.Scan(
		&lease.ID,
		&lease.ProjectID,
		&lease.RunID,
		&lease.RunTaskID,
		&lease.WorkflowItemID,
		&lease.WorkerID,
		&lease.LeaseKind,
		&lease.Status,
		&lease.AcquiredAt,
		&lease.ExpiresAt,
		&heartbeatAt,
		&releasedAt,
		&capabilitiesRaw,
		&scopeRaw,
		&metadataRaw,
	); err != nil {
		return LeaseRecord{}, err
	}
	if err := json.Unmarshal(capabilitiesRaw, &lease.AllowedCapabilities); err != nil {
		return LeaseRecord{}, fmt.Errorf("parse lease capabilities: %w", err)
	}
	if err := json.Unmarshal(scopeRaw, &lease.Scope); err != nil {
		return LeaseRecord{}, fmt.Errorf("parse lease scope: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &lease.Metadata); err != nil {
		return LeaseRecord{}, fmt.Errorf("parse lease metadata: %w", err)
	}
	if heartbeatAt.Valid {
		lease.HeartbeatAt = &heartbeatAt.Time
	}
	if releasedAt.Valid {
		lease.ReleasedAt = &releasedAt.Time
	}
	return lease, nil
}
