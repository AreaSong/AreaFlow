package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/config"
	"github.com/areasong/areaflow/internal/db"
	"github.com/areasong/areaflow/internal/doctor"
	"github.com/areasong/areaflow/internal/importer"
	"github.com/areasong/areaflow/internal/project"
	statusmirror "github.com/areasong/areaflow/internal/status"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type ProjectStore interface {
	List(ctx context.Context) ([]project.Record, error)
	GetByKey(ctx context.Context, key string) (project.Record, error)
	ListEvents(ctx context.Context, projectID int64, limit int) ([]project.EventRecord, error)
	ListAuditEvents(ctx context.Context, projectID int64, limit int) ([]project.AuditEventRecord, error)
	AuditCoverage(ctx context.Context, options project.AuditCoverageOptions) (project.AuditCoverage, error)
	ListEventStream(ctx context.Context, filter project.EventStreamFilter) ([]project.EventRecord, error)
	ListWorkflowVersions(ctx context.Context, record project.Record) ([]project.WorkflowVersion, error)
	GetWorkflowVersion(ctx context.Context, record project.Record, label string) (project.WorkflowVersion, error)
	CreateWorkflowVersion(ctx context.Context, record project.Record, options project.CreateWorkflowVersionOptions) (project.CreateWorkflowVersionResult, error)
	ListWorkflowItems(ctx context.Context, record project.Record, version project.WorkflowVersion) ([]project.WorkflowItem, error)
	ListWorkflowItemLinks(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.WorkflowItemLink, error)
	EnsureStageSkeleton(ctx context.Context, record project.Record, label string, options project.EnsureStageSkeletonOptions) (project.EnsureStageSkeletonResult, error)
	RunWorkflowGate(ctx context.Context, record project.Record, label string, options project.RunGateOptions) (project.GateResult, error)
	ListGateResults(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.GateResult, error)
	PreviewWorkflowTransition(ctx context.Context, record project.Record, label string, options project.PreviewTransitionOptions) (project.WorkflowTransitionPreview, error)
	ListWorkflowTransitionPreviews(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.WorkflowTransitionPreview, error)
	CreateApprovalRecord(ctx context.Context, record project.Record, label string, options project.CreateApprovalOptions) (project.ApprovalRecord, error)
	ListApprovalRecords(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.ApprovalRecord, error)
	ProjectSummary(ctx context.Context, record project.Record) (project.ProjectSummary, error)
	ProjectReadiness(ctx context.Context, record project.Record) (project.ProjectReadiness, error)
	GeneratedWriteReadiness(ctx context.Context, record project.Record, options project.GeneratedWriteReadinessOptions) (project.GeneratedWriteReadiness, error)
	GeneratedWriteApplyBetaGate(ctx context.Context, record project.Record, options project.GeneratedWriteApplyBetaGateOptions) (project.GeneratedWriteApplyBetaGate, error)
	PermissionPolicyDoctor(ctx context.Context, record project.Record, options project.PermissionPolicyDoctorOptions) (project.PermissionPolicyDoctor, error)
	ConformanceCheck(ctx context.Context, record project.Record, options project.ConformanceOptions) (project.ConformanceReport, error)
	ProjectImportDiff(ctx context.Context, record project.Record) (project.ProjectImportDiff, error)
	ProjectVerificationBundle(ctx context.Context, record project.Record, eventLimit int) (project.ProjectVerificationBundle, error)
	CompatibilityContract(ctx context.Context, record project.Record) (project.CompatibilityContract, error)
	ShimPreview(ctx context.Context, record project.Record) (project.ShimPreview, error)
	ShimReadiness(ctx context.Context, record project.Record) (project.ShimReadiness, error)
	ShimAuthorizationPacket(ctx context.Context, record project.Record) (project.ShimAuthorizationPacket, error)
	ShimApplyPacketPreview(ctx context.Context, record project.Record, options project.ShimApplyPacketPreviewOptions) (project.ShimApplyPacketPreview, error)
	ShimApplyGate(ctx context.Context, record project.Record, options project.ShimApplyGateOptions) (project.ShimApplyGate, error)
	ApplyShimCommand(ctx context.Context, record project.Record, options project.ApplyShimCommandOptions) (project.ApplyShimCommandResult, error)
	RecordShimReadinessEvidence(ctx context.Context, record project.Record, options project.RecordShimReadinessEvidenceOptions) (project.RecordShimReadinessEvidenceResult, error)
	AreaMatrixExecutionCutoverReadiness(ctx context.Context, record project.Record, options project.AreaMatrixExecutionCutoverReadinessOptions) (project.AreaMatrixExecutionCutoverReadiness, error)
	ExecutionForwardingV1Readiness(ctx context.Context, record project.Record, options project.ExecutionForwardingV1ReadinessOptions) (project.ExecutionForwardingV1Readiness, error)
	ExecutionForwardingV1ApplyPreview(ctx context.Context, record project.Record, options project.ExecutionForwardingV1ApplyPreviewOptions) (project.ExecutionForwardingV1ApplyPreview, error)
	ExecutionForwardingV1ApplyPacketPreview(ctx context.Context, record project.Record, options project.ExecutionForwardingV1ApplyPacketPreviewOptions) (project.ExecutionForwardingV1ApplyPacketPreview, error)
	ExecutionForwardingV1ApplyGate(ctx context.Context, record project.Record, options project.ExecutionForwardingV1ApplyGateOptions) (project.ExecutionForwardingV1ApplyGate, error)
	ApplyExecutionForwardingV1(ctx context.Context, record project.Record, options project.ApplyExecutionForwardingV1Options) (project.ApplyExecutionForwardingV1Result, error)
	ExecutionForwardingV1CommandPreview(ctx context.Context, record project.Record, options project.ExecutionForwardingV1CommandPreviewOptions) (project.ExecutionForwardingV1CommandPreview, error)
	ExecutionForwardingV1RollbackPreview(ctx context.Context, record project.Record, options project.ExecutionForwardingV1RollbackPreviewOptions) (project.ExecutionForwardingV1RollbackPreview, error)
	ProjectCutoverReadiness(ctx context.Context, record project.Record, label string, eventLimit int) (project.ProjectCutoverReadiness, error)
	ApplyCutover(ctx context.Context, record project.Record, options project.ApplyCutoverOptions) (project.ApplyCutoverResult, error)
	ListStatusProjections(ctx context.Context, record project.Record, limit int) ([]project.StatusProjectionRecord, error)
	StatusProjectionAuthorizationPreview(ctx context.Context, record project.Record, options project.StatusProjectionAuthorizationPreviewOptions) (project.StatusProjectionAuthorizationPreview, error)
	StatusProjectionApplyPacketPreview(ctx context.Context, record project.Record, options project.StatusProjectionApplyPacketPreviewOptions) (project.StatusProjectionApplyPacketPreview, error)
	StatusProjectionApplyGate(ctx context.Context, record project.Record, options project.StatusProjectionApplyGateOptions) (project.StatusProjectionApplyGate, error)
	ApplyStatusProjection(ctx context.Context, record project.Record, options project.ApplyStatusProjectionOptions) (project.ApplyStatusProjectionResult, error)
	ListProjectResiduals(ctx context.Context, record project.Record, limit int) ([]project.ResidualRecord, error)
	ListWorkflowVersionResiduals(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.ResidualRecord, error)
	PreviewRunner(ctx context.Context, record project.Record, label string, options project.RunnerPreviewOptions) (project.RunnerPreviewResult, error)
	QueueFixtureExecution(ctx context.Context, record project.Record, label string, options project.FixtureExecutionQueueOptions) (project.FixtureExecutionQueueResult, error)
	QueueReadOnlyVerify(ctx context.Context, record project.Record, label string, options project.ReadOnlyVerifyQueueOptions) (project.ReadOnlyVerifyQueueResult, error)
	QueueApprovedArtifactWrite(ctx context.Context, record project.Record, label string, options project.ApprovedArtifactWriteQueueOptions) (project.ApprovedArtifactWriteQueueResult, error)
	QueueFixtureProjectWrite(ctx context.Context, record project.Record, label string, options project.FixtureProjectWriteQueueOptions) (project.FixtureProjectWriteQueueResult, error)
	QueueManagedGeneratedWrite(ctx context.Context, record project.Record, label string, options project.ManagedGeneratedWriteQueueOptions) (project.ManagedGeneratedWriteQueueResult, error)
	ListWorkflowVersionRuns(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.RunRecord, error)
	RegisterWorker(ctx context.Context, record project.Record, options project.RegisterWorkerOptions) (project.WorkerRecord, error)
	RecordWorkerHeartbeat(ctx context.Context, record project.Record, workerKey string, options project.WorkerHeartbeatOptions) (project.WorkerRecord, error)
	ListWorkers(ctx context.Context, record project.Record, limit int) ([]project.WorkerRecord, error)
	WorkerPoolSummary(ctx context.Context) (project.WorkerPoolSummary, error)
	WorkerPoolSchedulePreview(ctx context.Context) (project.WorkerPoolSchedulePreview, error)
	CodexCLIAdapterPreview(ctx context.Context, record project.Record, options project.CodexCLIAdapterPreviewOptions) (project.CodexCLIAdapterPreview, error)
	WebWriteActionGate(ctx context.Context, options project.WebWriteActionGateOptions) (project.WebWriteActionGate, error)
	DesktopServiceControlGate(ctx context.Context, options project.DesktopServiceControlGateOptions) (project.DesktopServiceControlGate, error)
	DesktopNotificationGate(ctx context.Context, options project.DesktopNotificationGateOptions) (project.DesktopNotificationGate, error)
	DesktopTrayMenuGate(ctx context.Context, options project.DesktopTrayMenuGateOptions) (project.DesktopTrayMenuGate, error)
	LocalServiceStatus(ctx context.Context, options project.LocalServiceStatusOptions) (project.LocalServiceStatus, error)
	SecurityBoundaryReadiness(ctx context.Context, options project.SecurityBoundaryReadinessOptions) (project.SecurityBoundaryReadiness, error)
	CompletionAudit(ctx context.Context, options project.CompletionAuditOptions) (project.CompletionAudit, error)
	CompletionAuditSnapshotReadiness(ctx context.Context, record project.Record) (project.CompletionAuditSnapshotReadiness, error)
	SupportBundlePreview(ctx context.Context, options project.SupportBundlePreviewOptions) (project.SupportBundlePreview, error)
	MigrationLedgerReadiness(ctx context.Context, options project.MigrationLedgerReadinessOptions) (project.MigrationLedgerReadiness, error)
	OperationsReadiness(ctx context.Context, options project.OperationsReadinessOptions) (project.OperationsReadiness, error)
	BackupManifest(ctx context.Context, options project.BackupManifestOptions) (project.BackupManifest, error)
	RestorePlan(ctx context.Context, options project.RestorePlanOptions) (project.RestorePlan, error)
	ReleaseReadiness(ctx context.Context, options project.ReleaseReadinessOptions) (project.ReleaseReadiness, error)
	ReleaseRemediationPlan(ctx context.Context, options project.ReleaseRemediationOptions) (project.ReleaseRemediationPlan, error)
	ReleaseAcceptancePreview(ctx context.Context, options project.ReleaseAcceptancePreviewOptions) (project.ReleaseAcceptancePreview, error)
	ReleaseAcceptanceGate(ctx context.Context, options project.ReleaseAcceptanceGateOptions) (project.ReleaseAcceptanceGate, error)
	ReleaseExceptionDoctor(ctx context.Context, options project.ReleaseExceptionDoctorOptions) (project.ReleaseExceptionDoctor, error)
	ReleaseExceptionRecordPreview(ctx context.Context, options project.ReleaseExceptionRecordPreviewOptions) (project.ReleaseExceptionRecordPreview, error)
	ReleaseExceptionSchemaPreview(ctx context.Context, options project.ReleaseExceptionSchemaPreviewOptions) (project.ReleaseExceptionSchemaPreview, error)
	ReleaseExceptionMigrationApprovalGate(ctx context.Context, options project.ReleaseExceptionMigrationApprovalGateOptions) (project.ReleaseExceptionMigrationApprovalGate, error)
	ReleaseExceptionApplyPreview(ctx context.Context, options project.ReleaseExceptionApplyPreviewOptions) (project.ReleaseExceptionApplyPreview, error)
	ReleaseFinalGate(ctx context.Context, options project.ReleaseFinalGateOptions) (project.ReleaseFinalGate, error)
	ReleaseEvidenceBundle(ctx context.Context, options project.ReleaseEvidenceBundleOptions) (project.ReleaseEvidenceBundle, error)
	ReleasePackagePreview(ctx context.Context, options project.ReleasePackagePreviewOptions) (project.ReleasePackagePreview, error)
	ReleaseDistributionPreview(ctx context.Context, options project.ReleaseDistributionPreviewOptions) (project.ReleaseDistributionPreview, error)
	ReleasePublishGate(ctx context.Context, options project.ReleasePublishGateOptions) (project.ReleasePublishGate, error)
	ReleasePublishApprovalPreview(ctx context.Context, options project.ReleasePublishApprovalPreviewOptions) (project.ReleasePublishApprovalPreview, error)
	ReleaseRolloutPlanPreview(ctx context.Context, options project.ReleaseRolloutPlanPreviewOptions) (project.ReleaseRolloutPlanPreview, error)
	ArtifactIntegrity(ctx context.Context, record project.Record, options project.ArtifactIntegrityOptions) (project.ArtifactIntegrityReport, error)
	ArtifactArchivePreview(ctx context.Context, record project.Record, options project.ArtifactArchivePreviewOptions) (project.ArtifactArchivePreviewResult, error)
	AcquireLease(ctx context.Context, record project.Record, options project.AcquireLeaseOptions) (project.LeaseRecord, error)
	ReleaseLease(ctx context.Context, record project.Record, options project.ReleaseLeaseOptions) (project.LeaseRecord, error)
	RecoverExpiredLeases(ctx context.Context, record project.Record, options project.RecoverLeasesOptions) ([]project.LeaseRecord, error)
	RunWorkerOnce(ctx context.Context, record project.Record, options project.WorkerRunOnceOptions) (project.WorkerRunOnceResult, error)
	ExecuteFixture(ctx context.Context, record project.Record, options project.FixtureExecutionOptions) (project.FixtureExecutionResult, error)
	VerifyReadOnly(ctx context.Context, record project.Record, options project.ReadOnlyVerifyOptions) (project.ReadOnlyVerifyResult, error)
	WriteApprovedArtifact(ctx context.Context, record project.Record, options project.ApprovedArtifactWriteOptions) (project.ApprovedArtifactWriteResult, error)
	WriteFixtureProject(ctx context.Context, record project.Record, options project.FixtureProjectWriteOptions) (project.FixtureProjectWriteResult, error)
	WriteManagedGenerated(ctx context.Context, record project.Record, options project.ManagedGeneratedWriteOptions) (project.ManagedGeneratedWriteResult, error)
	GetRun(ctx context.Context, runID int64) (project.RunDetail, error)
	ExecutionApprovalGate(ctx context.Context, runID int64, options project.ExecutionApprovalGateOptions) (project.ExecutionApprovalGate, error)
	PreviewExecutionPlan(ctx context.Context, runID int64) (project.ExecutionPlanPreview, error)
	PreviewProjectWriteDesignGate(ctx context.Context, runID int64) (project.ProjectWriteDesignGate, error)
	PreviewManagedGeneratedWriteGate(ctx context.Context, runID int64) (project.ManagedGeneratedWriteGate, error)
	ControlRun(ctx context.Context, runID int64, options project.RunControlOptions) (project.RunControlResult, error)
	ListRunEvents(ctx context.Context, runID int64, limit int) ([]project.EventRecord, error)
	ListProjectArtifacts(ctx context.Context, record project.Record, limit int) ([]project.ArtifactRecord, error)
	ListWorkflowVersionArtifacts(ctx context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.ArtifactRecord, error)
	GetArtifact(ctx context.Context, artifactID int64) (project.ArtifactRecord, error)
	GetArtifactContent(ctx context.Context, artifactID int64) (project.ArtifactContent, error)
	RecordDoctorReport(ctx context.Context, projectID int64, summary map[string]any, options project.RecordDoctorReportOptions) (project.RecordDoctorReportResult, error)
}

type ProjectDoctorRunner func(ctx context.Context, record project.Record, store ProjectStore, allowNative bool) (doctor.Report, error)
type ProjectImporter func(ctx context.Context, record project.Record, options importer.Options) (importer.Result, error)

type Server struct {
	store        ProjectStore
	doctorRunner ProjectDoctorRunner
	importer     ProjectImporter
}

type projectVisibilityScope struct {
	enabled bool
	record  project.Record
}

func (s Server) projectVisibilityScopeFromQuery(ctx context.Context, r *http.Request) (projectVisibilityScope, error) {
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	if projectKey == "" {
		return projectVisibilityScope{}, nil
	}
	record, err := s.store.GetByKey(ctx, projectKey)
	if err != nil {
		return projectVisibilityScope{}, err
	}
	return projectVisibilityScope{enabled: true, record: record}, nil
}

func (scope projectVisibilityScope) allows(projectID int64) bool {
	return !scope.enabled || scope.record.ID == projectID
}

func (s Server) ensureRunVisibleToProject(ctx context.Context, scope projectVisibilityScope, runID int64) error {
	if !scope.enabled {
		return nil
	}
	detail, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if !scope.allows(detail.Run.ProjectID) {
		return project.ErrRunNotFound
	}
	return nil
}

func (s Server) ensureArtifactVisibleToProject(ctx context.Context, scope projectVisibilityScope, artifactID int64) error {
	if !scope.enabled {
		return nil
	}
	artifact, err := s.store.GetArtifact(ctx, artifactID)
	if err != nil {
		return err
	}
	if !scope.allows(artifact.ProjectID) {
		return project.ErrArtifactNotFound
	}
	return nil
}

type eventResponse struct {
	ID                int64          `json:"id"`
	ProjectID         int64          `json:"project_id,omitempty"`
	RunID             int64          `json:"run_id,omitempty"`
	WorkflowVersionID int64          `json:"workflow_version_id,omitempty"`
	Type              string         `json:"type"`
	Severity          string         `json:"severity"`
	Message           string         `json:"message"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
}

type projectEventsResponse struct {
	Project string          `json:"project"`
	Events  []eventResponse `json:"events"`
}

type auditEventResponse struct {
	ID           int64          `json:"id"`
	ProjectID    int64          `json:"project_id,omitempty"`
	ActorID      int64          `json:"actor_id,omitempty"`
	Action       string         `json:"action"`
	Capability   string         `json:"capability,omitempty"`
	ResourceType string         `json:"resource_type,omitempty"`
	Resource     string         `json:"resource,omitempty"`
	Decision     string         `json:"decision"`
	Reason       string         `json:"reason,omitempty"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    string         `json:"created_at"`
}

type auditEventsResponse struct {
	ProjectKey  string               `json:"project_key,omitempty"`
	AuditEvents []auditEventResponse `json:"audit_events"`
}

type auditCoverageResponse struct {
	Status              string                             `json:"status"`
	Mode                string                             `json:"mode"`
	Scope               string                             `json:"scope"`
	ProjectID           int64                              `json:"project_id,omitempty"`
	ProjectKey          string                             `json:"project_key,omitempty"`
	TotalAuditEvents    int64                              `json:"total_audit_events"`
	CoveredRequirements int                                `json:"covered_requirements"`
	GapRequirements     int                                `json:"gap_requirements"`
	Requirements        []auditCoverageRequirementResponse `json:"requirements"`
	GeneratedAt         string                             `json:"generated_at"`
}

type auditCoverageRequirementResponse struct {
	Key             string                                `json:"key"`
	Category        string                                `json:"category"`
	Description     string                                `json:"description"`
	Status          string                                `json:"status"`
	EvidenceCount   int64                                 `json:"evidence_count"`
	RequiredActions []auditCoverageActionEvidenceResponse `json:"required_actions"`
	MissingActions  []string                              `json:"missing_actions"`
	LastAuditAt     string                                `json:"last_audit_at,omitempty"`
}

type auditCoverageActionEvidenceResponse struct {
	Action      string `json:"action"`
	Decision    string `json:"decision,omitempty"`
	Count       int64  `json:"count"`
	Status      string `json:"status"`
	LastAuditAt string `json:"last_audit_at,omitempty"`
}

type permissionPolicyDoctorResponse struct {
	Status      string                          `json:"status"`
	Mode        string                          `json:"mode"`
	Project     projectRecordResponse           `json:"project"`
	Checks      []permissionPolicyCheckResponse `json:"checks"`
	GeneratedAt string                          `json:"generated_at"`
}

type permissionPolicyCheckResponse struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type artifactIntegrityResponse struct {
	Status           string                           `json:"status"`
	Mode             string                           `json:"mode"`
	Project          projectRecordResponse            `json:"project"`
	CheckedArtifacts int                              `json:"checked_artifacts"`
	PassedArtifacts  int                              `json:"passed_artifacts"`
	WarnArtifacts    int                              `json:"warn_artifacts"`
	FailedArtifacts  int                              `json:"failed_artifacts"`
	SkippedArtifacts int                              `json:"skipped_artifacts"`
	Checks           []artifactIntegrityCheckResponse `json:"checks"`
	GeneratedAt      string                           `json:"generated_at"`
}

type artifactIntegrityCheckResponse struct {
	Artifact artifactResponse `json:"artifact"`
	Status   string           `json:"status"`
	Message  string           `json:"message"`
	Metadata map[string]any   `json:"metadata"`
}

type artifactArchivePreviewResponse struct {
	Project                 projectRecordResponse                 `json:"project"`
	Status                  string                                `json:"status"`
	Mode                    string                                `json:"mode"`
	Summary                 artifactArchivePreviewSummaryResponse `json:"summary"`
	Items                   []artifactArchivePreviewItemResponse  `json:"items"`
	EventID                 int64                                 `json:"event_id,omitempty"`
	AuditEventID            int64                                 `json:"audit_event_id,omitempty"`
	IdempotencyKey          string                                `json:"idempotency_key"`
	Created                 bool                                  `json:"created"`
	GeneratedAt             string                                `json:"generated_at"`
	ProjectWriteAttempted   bool                                  `json:"project_write_attempted"`
	StorageWriteAttempted   bool                                  `json:"storage_write_attempted"`
	ArtifactDeleteAttempted bool                                  `json:"artifact_delete_attempted"`
}

type artifactArchivePreviewSummaryResponse struct {
	TotalArtifacts    int `json:"total_artifacts"`
	ArchiveCandidates int `json:"archive_candidates"`
	RetainedArtifacts int `json:"retained_artifacts"`
	ExternalRefs      int `json:"external_refs"`
	NeedsPolicy       int `json:"needs_policy"`
}

type artifactArchivePreviewItemResponse struct {
	ArtifactID     int64          `json:"artifact_id"`
	ArtifactType   string         `json:"artifact_type"`
	StorageBackend string         `json:"storage_backend"`
	URI            string         `json:"uri"`
	SourcePath     string         `json:"source_path"`
	RetentionClass string         `json:"retention_class"`
	ArchiveState   string         `json:"archive_state"`
	Action         string         `json:"action"`
	Decision       string         `json:"decision"`
	Reason         string         `json:"reason"`
	Metadata       map[string]any `json:"metadata"`
}

type conformanceResponse struct {
	Status      string                     `json:"status"`
	Mode        string                     `json:"mode"`
	Project     projectRecordResponse      `json:"project"`
	ProfileID   string                     `json:"profile_id"`
	Adapter     string                     `json:"adapter"`
	ProfileHash string                     `json:"profile_hash"`
	StageCount  int                        `json:"stage_count"`
	GateCount   int                        `json:"gate_count"`
	Checks      []conformanceCheckResponse `json:"checks"`
	GeneratedAt string                     `json:"generated_at"`
}

type conformanceCheckResponse struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type projectSummaryResponse struct {
	Project   projectRecordResponse    `json:"project"`
	Config    *projectConfigResponse   `json:"config,omitempty"`
	Inventory projectInventoryResponse `json:"inventory"`
	Import    *projectImportResponse   `json:"import,omitempty"`
	Doctor    *projectDoctorResponse   `json:"doctor,omitempty"`
}

type projectConfigResponse struct {
	ProtocolVersion int            `json:"protocol_version"`
	ConfigPath      string         `json:"config_path"`
	ConfigHash      string         `json:"config_hash"`
	Ownership       map[string]any `json:"ownership"`
	StatusExport    map[string]any `json:"status_export"`
	Migration       map[string]any `json:"migration"`
	LoadedAt        string         `json:"loaded_at"`
}

type projectRecordResponse struct {
	Key             string `json:"key"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Adapter         string `json:"adapter"`
	WorkflowProfile string `json:"workflow_profile"`
	DefaultBranch   string `json:"default_branch"`
	Root            string `json:"root"`
}

type projectListResponse struct {
	Projects []projectRecordResponse `json:"projects"`
}

type localServiceStatusResponse struct {
	Status           string                         `json:"status"`
	Mode             string                         `json:"mode"`
	API              localServiceComponentResponse  `json:"api"`
	Database         localServiceComponentResponse  `json:"database"`
	WorkerPool       localServiceWorkerPoolResponse `json:"worker_pool"`
	Dashboard        localServiceDashboardResponse  `json:"dashboard"`
	Capabilities     []string                       `json:"capabilities"`
	ForbiddenActions []string                       `json:"forbidden_actions"`
	GeneratedAt      string                         `json:"generated_at"`
}

type localServiceComponentResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type localServiceWorkerPoolResponse struct {
	Status             string `json:"status"`
	Message            string `json:"message"`
	TotalProjects      int64  `json:"total_projects"`
	TotalWorkers       int64  `json:"total_workers"`
	TotalOnlineWorkers int64  `json:"total_online_workers"`
	TotalActiveLeases  int64  `json:"total_active_leases"`
	TotalQueuedTasks   int64  `json:"total_queued_tasks"`
	TotalNeedsRecovery int64  `json:"total_needs_recovery"`
}

type localServiceDashboardResponse struct {
	URL     string `json:"url"`
	APIURL  string `json:"api_url"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type webWriteActionGateResponse struct {
	Status                  string                   `json:"status"`
	Mode                    string                   `json:"mode"`
	Actions                 []webWriteActionResponse `json:"actions"`
	Capabilities            []string                 `json:"capabilities"`
	ForbiddenActions        []string                 `json:"forbidden_actions"`
	GeneratedAt             string                   `json:"generated_at"`
	DBWriteAttempted        bool                     `json:"db_write_attempted"`
	ProjectWriteAttempted   bool                     `json:"project_write_attempted"`
	ArtifactWriteAttempted  bool                     `json:"artifact_write_attempted"`
	ExecutionWriteAttempted bool                     `json:"execution_write_attempted"`
	CommandCreated          bool                     `json:"command_created"`
	ApprovalCreated         bool                     `json:"approval_created"`
	AuditEventWritten       bool                     `json:"audit_event_written"`
	WorkerScheduled         bool                     `json:"worker_scheduled"`
	EngineCallAttempted     bool                     `json:"engine_call_attempted"`
	CommandsRun             bool                     `json:"commands_run"`
	SecretsResolved         bool                     `json:"secrets_resolved"`
	NetworkUsed             bool                     `json:"network_used"`
}

type webWriteActionResponse struct {
	Key                    string   `json:"key"`
	Label                  string   `json:"label"`
	Category               string   `json:"category"`
	Status                 string   `json:"status"`
	DefaultUIState         string   `json:"default_ui_state"`
	CommandAPI             string   `json:"command_api"`
	RiskLevel              string   `json:"risk_level"`
	RequiredCapabilities   []string `json:"required_capabilities"`
	RequiredPreviews       []string `json:"required_previews"`
	RequiredApprovals      []string `json:"required_approvals"`
	RequiredAuditEvents    []string `json:"required_audit_events"`
	RequiredEvidence       []string `json:"required_evidence"`
	Blockers               []string `json:"blockers"`
	ForbiddenDirectActions []string `json:"forbidden_direct_actions"`
}

type desktopServiceControlGateResponse struct {
	Status                   string                        `json:"status"`
	Mode                     string                        `json:"mode"`
	Actions                  []desktopServiceControlAction `json:"actions"`
	Capabilities             []string                      `json:"capabilities"`
	ForbiddenActions         []string                      `json:"forbidden_actions"`
	GeneratedAt              string                        `json:"generated_at"`
	DBWriteAttempted         bool                          `json:"db_write_attempted"`
	ProjectWriteAttempted    bool                          `json:"project_write_attempted"`
	ProcessControlAttempted  bool                          `json:"process_control_attempted"`
	CommandCreated           bool                          `json:"command_created"`
	ApprovalCreated          bool                          `json:"approval_created"`
	AuditEventWritten        bool                          `json:"audit_event_written"`
	WorkerScheduled          bool                          `json:"worker_scheduled"`
	WorkflowExecutionStarted bool                          `json:"workflow_execution_started"`
	SecretsResolved          bool                          `json:"secrets_resolved"`
	NetworkUsed              bool                          `json:"network_used"`
}

type desktopServiceControlAction struct {
	Key                    string   `json:"key"`
	Label                  string   `json:"label"`
	Category               string   `json:"category"`
	Status                 string   `json:"status"`
	DefaultUIState         string   `json:"default_ui_state"`
	CommandAPI             string   `json:"command_api"`
	RiskLevel              string   `json:"risk_level"`
	RequiredCapabilities   []string `json:"required_capabilities"`
	RequiredPreviews       []string `json:"required_previews"`
	RequiredApprovals      []string `json:"required_approvals"`
	RequiredAuditEvents    []string `json:"required_audit_events"`
	RequiredEvidence       []string `json:"required_evidence"`
	Blockers               []string `json:"blockers"`
	ForbiddenDirectActions []string `json:"forbidden_direct_actions"`
}

type desktopNotificationGateResponse struct {
	Status                   string                      `json:"status"`
	Mode                     string                      `json:"mode"`
	Actions                  []desktopNotificationAction `json:"actions"`
	Capabilities             []string                    `json:"capabilities"`
	ForbiddenActions         []string                    `json:"forbidden_actions"`
	GeneratedAt              string                      `json:"generated_at"`
	DBWriteAttempted         bool                        `json:"db_write_attempted"`
	ProjectWriteAttempted    bool                        `json:"project_write_attempted"`
	EventStreamOpened        bool                        `json:"event_stream_opened"`
	NotificationRequested    bool                        `json:"notification_requested"`
	CommandCreated           bool                        `json:"command_created"`
	ApprovalCreated          bool                        `json:"approval_created"`
	AuditEventWritten        bool                        `json:"audit_event_written"`
	WorkerScheduled          bool                        `json:"worker_scheduled"`
	WorkflowExecutionStarted bool                        `json:"workflow_execution_started"`
	SecretsResolved          bool                        `json:"secrets_resolved"`
	NetworkUsed              bool                        `json:"network_used"`
}

type desktopNotificationAction struct {
	Key                    string   `json:"key"`
	Label                  string   `json:"label"`
	Category               string   `json:"category"`
	Status                 string   `json:"status"`
	DefaultUIState         string   `json:"default_ui_state"`
	RiskLevel              string   `json:"risk_level"`
	RequiredCapabilities   []string `json:"required_capabilities"`
	RequiredPreviews       []string `json:"required_previews"`
	RequiredApprovals      []string `json:"required_approvals"`
	RequiredAuditEvents    []string `json:"required_audit_events"`
	RequiredEvidence       []string `json:"required_evidence"`
	Blockers               []string `json:"blockers"`
	ForbiddenDirectActions []string `json:"forbidden_direct_actions"`
}

type desktopTrayMenuGateResponse struct {
	Status                   string                  `json:"status"`
	Mode                     string                  `json:"mode"`
	Actions                  []desktopTrayMenuAction `json:"actions"`
	Capabilities             []string                `json:"capabilities"`
	ForbiddenActions         []string                `json:"forbidden_actions"`
	GeneratedAt              string                  `json:"generated_at"`
	DBWriteAttempted         bool                    `json:"db_write_attempted"`
	ProjectWriteAttempted    bool                    `json:"project_write_attempted"`
	TrayMenuCreated          bool                    `json:"tray_menu_created"`
	OSIntegrationRequested   bool                    `json:"os_integration_requested"`
	CommandCreated           bool                    `json:"command_created"`
	ApprovalCreated          bool                    `json:"approval_created"`
	AuditEventWritten        bool                    `json:"audit_event_written"`
	ServiceControlAttempted  bool                    `json:"service_control_attempted"`
	NotificationRequested    bool                    `json:"notification_requested"`
	WorkerScheduled          bool                    `json:"worker_scheduled"`
	WorkflowExecutionStarted bool                    `json:"workflow_execution_started"`
	SecretsResolved          bool                    `json:"secrets_resolved"`
	NetworkUsed              bool                    `json:"network_used"`
}

type desktopTrayMenuAction struct {
	Key                    string   `json:"key"`
	Label                  string   `json:"label"`
	Category               string   `json:"category"`
	Status                 string   `json:"status"`
	DefaultUIState         string   `json:"default_ui_state"`
	RiskLevel              string   `json:"risk_level"`
	RequiredCapabilities   []string `json:"required_capabilities"`
	RequiredPreviews       []string `json:"required_previews"`
	RequiredApprovals      []string `json:"required_approvals"`
	RequiredAuditEvents    []string `json:"required_audit_events"`
	RequiredEvidence       []string `json:"required_evidence"`
	Blockers               []string `json:"blockers"`
	ForbiddenDirectActions []string `json:"forbidden_direct_actions"`
}

type securityBoundaryReadinessResponse struct {
	Status                        string                                  `json:"status"`
	Mode                          string                                  `json:"mode"`
	Items                         []securityBoundaryReadinessItemResponse `json:"items"`
	Capabilities                  []string                                `json:"capabilities"`
	ForbiddenActions              []string                                `json:"forbidden_actions"`
	GeneratedAt                   string                                  `json:"generated_at"`
	AuthEnforcementOpen           bool                                    `json:"auth_enforcement_open"`
	TeamPermissionEnforcementOpen bool                                    `json:"team_permission_enforcement_open"`
	APITokenIssuanceOpen          bool                                    `json:"api_token_issuance_open"`
	APITokenEnforcementOpen       bool                                    `json:"api_token_enforcement_open"`
	SecretResolveOpen             bool                                    `json:"secret_resolve_open"`
	RemoteWorkerCredentialsOpen   bool                                    `json:"remote_worker_credentials_open"`
	BudgetEnforcementOpen         bool                                    `json:"budget_enforcement_open"`
	QuotaDecrementOpen            bool                                    `json:"quota_decrement_open"`
	UsageChargeWritten            bool                                    `json:"usage_charge_written"`
	WebhookDeliveryOpen           bool                                    `json:"webhook_delivery_open"`
	InboundCallbackOpen           bool                                    `json:"inbound_callback_open"`
	ExternalAPICallOpen           bool                                    `json:"external_api_call_open"`
	AuthorizationChanged          bool                                    `json:"authorization_changed"`
	SecretPlaintextRead           bool                                    `json:"secret_plaintext_read"`
	RemoteWorkerDirectPGAllowed   bool                                    `json:"remote_worker_direct_pg_allowed"`
	TeamConsoleCommandOpen        bool                                    `json:"team_console_command_open"`
	RemoteOpsControlOpen          bool                                    `json:"remote_ops_control_open"`
	ManagedUpgradeOpen            bool                                    `json:"managed_upgrade_open"`
	SupportBundleExportOpen       bool                                    `json:"support_bundle_export_open"`
	DefaultRemoteTelemetryOpen    bool                                    `json:"default_remote_telemetry_open"`
}

type securityBoundaryReadinessItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	RequiredEvidence []string       `json:"required_evidence"`
	BlockedBy        []string       `json:"blocked_by"`
	Metadata         map[string]any `json:"metadata"`
}

type completionAuditResponse struct {
	Status                     string                        `json:"status"`
	Mode                       string                        `json:"mode"`
	Scope                      string                        `json:"scope"`
	ReadinessScope             string                        `json:"readiness_scope"`
	ClaimScope                 string                        `json:"claim_scope"`
	NotReal100                 bool                          `json:"not_real_100"`
	EvidenceOnly               bool                          `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                          `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                        `json:"release_candidate_decision"`
	Real100Status              string                        `json:"real_100_status"`
	Real100Blockers            []string                      `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown      `json:"real_100_breakdown"`
	Items                      []completionAuditItemResponse `json:"items"`
	DeferredV1x                []string                      `json:"deferred_v1x"`
	Capabilities               []string                      `json:"capabilities"`
	ForbiddenActions           []string                      `json:"forbidden_actions"`
	SafetyFacts                map[string]bool               `json:"safety_facts"`
	ReleaseFinalGateStatus     string                        `json:"release_final_gate_status"`
	AreaMatrixDogfoodStatus    string                        `json:"area_matrix_dogfood_status"`
	TaskMatrixStatus           string                        `json:"task_matrix_status"`
	ImplementationGapStatus    string                        `json:"implementation_gap_status"`
	ProtectedPathProofStatus   string                        `json:"protected_path_proof_status"`
	GeneratedAt                string                        `json:"generated_at"`
}

type completionAuditItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	EvidenceRefs     []string       `json:"evidence_refs"`
	RequiredEvidence []string       `json:"required_evidence"`
	BlockedBy        []string       `json:"blocked_by"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type completionAuditSnapshotReadinessResponse struct {
	Project                    projectRecordResponse                  `json:"project"`
	Status                     string                                 `json:"status"`
	Message                    string                                 `json:"message"`
	ReadinessScope             string                                 `json:"readiness_scope"`
	ClaimScope                 string                                 `json:"claim_scope"`
	NotReal100                 bool                                   `json:"not_real_100"`
	EvidenceOnly               bool                                   `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                   `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                 `json:"release_candidate_decision"`
	Real100Status              string                                 `json:"real_100_status"`
	Real100Blockers            []string                               `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown               `json:"real_100_breakdown"`
	HasSnapshot                bool                                   `json:"has_snapshot"`
	RequiredClass              string                                 `json:"required_class"`
	BundleHash                 string                                 `json:"bundle_hash"`
	Latest                     completionAuditSnapshotResponse        `json:"latest"`
	Items                      []projectReadinessItemResponse         `json:"items"`
	Gaps                       []project.CompletionAuditSnapshotGap   `json:"gaps"`
	Closure                    project.CompletionAuditSnapshotClosure `json:"closure"`
	SafetyFacts                map[string]bool                        `json:"safety_facts"`
}

type completionAuditSnapshotResponse struct {
	Status                     string                   `json:"status"`
	Decision                   string                   `json:"decision"`
	Message                    string                   `json:"message"`
	AuditStatus                string                   `json:"audit_status"`
	AuditScope                 string                   `json:"audit_scope"`
	ReadinessScope             string                   `json:"readiness_scope"`
	ClaimScope                 string                   `json:"claim_scope"`
	NotReal100                 bool                     `json:"not_real_100"`
	EvidenceOnly               bool                     `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                     `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                   `json:"release_candidate_decision"`
	Real100Status              string                   `json:"real_100_status"`
	Real100Blockers            []string                 `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown `json:"real_100_breakdown"`
	AuditHash                  string                   `json:"audit_hash"`
	ReleaseCandidateLabel      string                   `json:"release_candidate_label"`
	EvidenceClass              string                   `json:"evidence_class"`
	EvidenceURI                string                   `json:"evidence_uri"`
	ProofEventIDs              map[string]int64         `json:"proof_event_ids"`
	EventID                    int64                    `json:"event_id,omitempty"`
	AuditEventID               int64                    `json:"audit_event_id,omitempty"`
	IdempotencyKey             string                   `json:"idempotency_key"`
	CreatedAt                  string                   `json:"created_at,omitempty"`
	Metadata                   map[string]any           `json:"metadata"`
}

type supportBundlePreviewResponse struct {
	Status                   string                               `json:"status"`
	Mode                     string                               `json:"mode"`
	BundleID                 string                               `json:"bundle_id"`
	Scope                    string                               `json:"scope"`
	Projects                 []projectRecordResponse              `json:"projects"`
	IncludedMetadata         []string                             `json:"included_metadata"`
	ExcludedSensitiveContent []string                             `json:"excluded_sensitive_content"`
	PathReferences           []supportBundlePathReferenceResponse `json:"path_references"`
	Hashes                   []supportBundleHashReferenceResponse `json:"hashes"`
	Capabilities             []string                             `json:"capabilities"`
	ForbiddenActions         []string                             `json:"forbidden_actions"`
	SafetyFacts              map[string]bool                      `json:"safety_facts"`
	GeneratedAt              string                               `json:"generated_at"`
}

type supportBundlePathReferenceResponse struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"`
	URI         string `json:"uri"`
	ProjectKey  string `json:"project_key,omitempty"`
	Description string `json:"description"`
}

type supportBundleHashReferenceResponse struct {
	Key         string `json:"key"`
	Hash        string `json:"hash"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type migrationLedgerReadinessResponse struct {
	Status                               string                         `json:"status"`
	Mode                                 string                         `json:"mode"`
	Entries                              []migrationLedgerEntryResponse `json:"entries"`
	AppliedCount                         int                            `json:"applied_count"`
	PendingCount                         int                            `json:"pending_count"`
	SchemaMigrationsTablePresent         bool                           `json:"schema_migrations_table_present"`
	FullLedgerTablePresent               bool                           `json:"full_ledger_table_present"`
	PreflightApplyVerifyRemediationReady bool                           `json:"preflight_apply_verify_remediation_ready"`
	Capabilities                         []string                       `json:"capabilities"`
	ForbiddenActions                     []string                       `json:"forbidden_actions"`
	SafetyFacts                          map[string]bool                `json:"safety_facts"`
	GeneratedAt                          string                         `json:"generated_at"`
}

type migrationLedgerEntryResponse struct {
	Name             string                         `json:"name"`
	Applied          bool                           `json:"applied"`
	Status           string                         `json:"status"`
	RequiredEvidence []string                       `json:"required_evidence"`
	Phases           []migrationLedgerPhaseResponse `json:"phases"`
	Metadata         map[string]any                 `json:"metadata"`
}

type migrationLedgerPhaseResponse struct {
	Phase       string         `json:"phase"`
	Status      string         `json:"status"`
	Message     string         `json:"message"`
	Remediation string         `json:"remediation"`
	Metadata    map[string]any `json:"metadata"`
}

type operationsReadinessResponse struct {
	Status              string                            `json:"status"`
	Mode                string                            `json:"mode"`
	Items               []operationsReadinessItemResponse `json:"items"`
	ServiceStatus       localServiceStatusResponse        `json:"service_status"`
	SupportBundle       supportBundlePreviewResponse      `json:"support_bundle"`
	MigrationLedger     migrationLedgerReadinessResponse  `json:"migration_ledger"`
	Capabilities        []string                          `json:"capabilities"`
	ForbiddenActions    []string                          `json:"forbidden_actions"`
	SafetyFacts         map[string]bool                   `json:"safety_facts"`
	TelemetryDefault    string                            `json:"telemetry_default"`
	ManagedOpsStatus    string                            `json:"managed_ops_status"`
	SupportExportStatus string                            `json:"support_export_status"`
	GeneratedAt         string                            `json:"generated_at"`
}

type operationsReadinessItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	EvidenceRefs     []string       `json:"evidence_refs"`
	RequiredEvidence []string       `json:"required_evidence"`
	BlockedBy        []string       `json:"blocked_by"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type backupManifestResponse struct {
	Status           string                          `json:"status"`
	Mode             string                          `json:"mode"`
	Scope            string                          `json:"scope"`
	ProjectKey       string                          `json:"project_key,omitempty"`
	SchemaVersion    int                             `json:"schema_version"`
	GeneratedAt      string                          `json:"generated_at"`
	ManifestHash     string                          `json:"manifest_hash"`
	TableCounts      []backupTableCountResponse      `json:"table_counts"`
	Projects         []backupProjectManifestResponse `json:"projects"`
	Capabilities     []string                        `json:"capabilities"`
	ForbiddenActions []string                        `json:"forbidden_actions"`
}

type backupTableCountResponse struct {
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}

type backupProjectManifestResponse struct {
	Project       projectRecordResponse           `json:"project"`
	Inventory     projectInventoryResponse        `json:"inventory"`
	ArtifactCount int64                           `json:"artifact_count"`
	Artifacts     []backupArtifactSummaryResponse `json:"artifacts"`
}

type backupArtifactSummaryResponse struct {
	ID                int64  `json:"id"`
	ProjectID         int64  `json:"project_id"`
	WorkflowVersionID int64  `json:"workflow_version_id,omitempty"`
	WorkflowItemID    int64  `json:"workflow_item_id,omitempty"`
	ArtifactType      string `json:"artifact_type"`
	StorageBackend    string `json:"storage_backend"`
	URI               string `json:"uri"`
	SourcePath        string `json:"source_path"`
	SHA256            string `json:"sha256"`
	SizeBytes         int64  `json:"size_bytes"`
	ContentType       string `json:"content_type"`
	CreatedAt         string `json:"created_at"`
}

type restorePlanResponse struct {
	Status           string                    `json:"status"`
	Mode             string                    `json:"mode"`
	Scope            string                    `json:"scope"`
	ProjectKey       string                    `json:"project_key,omitempty"`
	SchemaVersion    int                       `json:"schema_version"`
	ManifestHash     string                    `json:"manifest_hash"`
	Projects         []projectRecordResponse   `json:"projects"`
	Items            []restorePlanItemResponse `json:"items"`
	Capabilities     []string                  `json:"capabilities"`
	ForbiddenActions []string                  `json:"forbidden_actions"`
	GeneratedAt      string                    `json:"generated_at"`
}

type restorePlanItemResponse struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type releaseReadinessResponse struct {
	Status                     string                            `json:"status"`
	Mode                       string                            `json:"mode"`
	Scope                      string                            `json:"scope"`
	ProjectKey                 string                            `json:"project_key,omitempty"`
	ReadinessScope             string                            `json:"readiness_scope"`
	ClaimScope                 string                            `json:"claim_scope"`
	NotReal100                 bool                              `json:"not_real_100"`
	EvidenceOnly               bool                              `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                              `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                            `json:"release_candidate_decision"`
	Real100Status              string                            `json:"real_100_status"`
	Real100Blockers            []string                          `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown          `json:"real_100_breakdown"`
	Backup                     backupManifestResponse            `json:"backup"`
	RestorePlan                restorePlanResponse               `json:"restore_plan"`
	AuditCoverage              auditCoverageResponse             `json:"audit_coverage"`
	Projects                   []releaseReadinessProjectResponse `json:"projects"`
	Items                      []releaseReadinessItemResponse    `json:"items"`
	Capabilities               []string                          `json:"capabilities"`
	ForbiddenActions           []string                          `json:"forbidden_actions"`
	GeneratedAt                string                            `json:"generated_at"`
}

type releaseReadinessProjectResponse struct {
	Project             projectRecordResponse          `json:"project"`
	Permission          permissionPolicyDoctorResponse `json:"permission"`
	ArtifactIntegrity   artifactIntegrityResponse      `json:"artifact_integrity"`
	Conformance         conformanceResponse            `json:"conformance"`
	Status              string                         `json:"status"`
	NeedsAttentionItems int                            `json:"needs_attention_items"`
	BlockedItems        int                            `json:"blocked_items"`
}

type releaseReadinessItemResponse struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type releaseRemediationPlanResponse struct {
	Status                     string                             `json:"status"`
	Mode                       string                             `json:"mode"`
	Scope                      string                             `json:"scope"`
	ProjectKey                 string                             `json:"project_key,omitempty"`
	ReadinessScope             string                             `json:"readiness_scope"`
	ClaimScope                 string                             `json:"claim_scope"`
	NotReal100                 bool                               `json:"not_real_100"`
	EvidenceOnly               bool                               `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                               `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                             `json:"release_candidate_decision"`
	Real100Status              string                             `json:"real_100_status"`
	Real100Blockers            []string                           `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown           `json:"real_100_breakdown"`
	Readiness                  releaseReadinessResponse           `json:"readiness"`
	Actions                    []releaseRemediationActionResponse `json:"actions"`
	Capabilities               []string                           `json:"capabilities"`
	ForbiddenActions           []string                           `json:"forbidden_actions"`
	GeneratedAt                string                             `json:"generated_at"`
}

type releaseRemediationActionResponse struct {
	Key               string         `json:"key"`
	Category          string         `json:"category"`
	Status            string         `json:"status"`
	SourceItem        string         `json:"source_item"`
	RecommendedAction string         `json:"recommended_action"`
	Rationale         string         `json:"rationale"`
	Owner             string         `json:"owner"`
	NextCommand       string         `json:"next_command"`
	Acceptance        string         `json:"acceptance"`
	Metadata          map[string]any `json:"metadata"`
}

type releaseAcceptancePreviewResponse struct {
	Status                     string                              `json:"status"`
	Mode                       string                              `json:"mode"`
	Scope                      string                              `json:"scope"`
	ProjectKey                 string                              `json:"project_key,omitempty"`
	ReadinessScope             string                              `json:"readiness_scope"`
	ClaimScope                 string                              `json:"claim_scope"`
	NotReal100                 bool                                `json:"not_real_100"`
	EvidenceOnly               bool                                `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                              `json:"release_candidate_decision"`
	Real100Status              string                              `json:"real_100_status"`
	Real100Blockers            []string                            `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown            `json:"real_100_breakdown"`
	Remediation                releaseRemediationPlanResponse      `json:"remediation"`
	Decisions                  []releaseAcceptanceDecisionResponse `json:"decisions"`
	Capabilities               []string                            `json:"capabilities"`
	ForbiddenActions           []string                            `json:"forbidden_actions"`
	GeneratedAt                string                              `json:"generated_at"`
}

type releaseAcceptanceDecisionResponse struct {
	Key              string         `json:"key"`
	SourceAction     string         `json:"source_action"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	AcceptanceType   string         `json:"acceptance_type"`
	Owner            string         `json:"owner"`
	Reason           string         `json:"reason"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseAcceptanceGateResponse struct {
	Status                     string                              `json:"status"`
	Mode                       string                              `json:"mode"`
	Scope                      string                              `json:"scope"`
	ProjectKey                 string                              `json:"project_key,omitempty"`
	ReadinessScope             string                              `json:"readiness_scope"`
	ClaimScope                 string                              `json:"claim_scope"`
	NotReal100                 bool                                `json:"not_real_100"`
	EvidenceOnly               bool                                `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                              `json:"release_candidate_decision"`
	Real100Status              string                              `json:"real_100_status"`
	Real100Blockers            []string                            `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown            `json:"real_100_breakdown"`
	Preview                    releaseAcceptancePreviewResponse    `json:"preview"`
	Items                      []releaseAcceptanceGateItemResponse `json:"items"`
	Capabilities               []string                            `json:"capabilities"`
	ForbiddenActions           []string                            `json:"forbidden_actions"`
	GeneratedAt                string                              `json:"generated_at"`
}

type releaseAcceptanceGateItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	DecisionStatus   string         `json:"decision_status"`
	AcceptanceType   string         `json:"acceptance_type"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseExceptionDoctorResponse struct {
	Status                     string                                `json:"status"`
	Mode                       string                                `json:"mode"`
	Scope                      string                                `json:"scope"`
	ProjectKey                 string                                `json:"project_key,omitempty"`
	ReadinessScope             string                                `json:"readiness_scope"`
	ClaimScope                 string                                `json:"claim_scope"`
	NotReal100                 bool                                  `json:"not_real_100"`
	EvidenceOnly               bool                                  `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                  `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                `json:"release_candidate_decision"`
	Real100Status              string                                `json:"real_100_status"`
	Real100Blockers            []string                              `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown              `json:"real_100_breakdown"`
	Gate                       releaseAcceptanceGateResponse         `json:"gate"`
	Checks                     []releaseExceptionDoctorCheckResponse `json:"checks"`
	Capabilities               []string                              `json:"capabilities"`
	ForbiddenActions           []string                              `json:"forbidden_actions"`
	GeneratedAt                string                                `json:"generated_at"`
}

type releaseExceptionDoctorCheckResponse struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type releaseExceptionRecordPreviewResponse struct {
	Status                     string                                `json:"status"`
	Mode                       string                                `json:"mode"`
	Scope                      string                                `json:"scope"`
	ProjectKey                 string                                `json:"project_key,omitempty"`
	ReadinessScope             string                                `json:"readiness_scope"`
	ClaimScope                 string                                `json:"claim_scope"`
	NotReal100                 bool                                  `json:"not_real_100"`
	EvidenceOnly               bool                                  `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                  `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                `json:"release_candidate_decision"`
	Real100Status              string                                `json:"real_100_status"`
	Real100Blockers            []string                              `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown              `json:"real_100_breakdown"`
	Doctor                     releaseExceptionDoctorResponse        `json:"doctor"`
	Drafts                     []releaseExceptionRecordDraftResponse `json:"drafts"`
	Capabilities               []string                              `json:"capabilities"`
	ForbiddenActions           []string                              `json:"forbidden_actions"`
	GeneratedAt                string                                `json:"generated_at"`
}

type releaseExceptionRecordDraftResponse struct {
	Key              string         `json:"key"`
	SourceGateItem   string         `json:"source_gate_item"`
	SourceDecision   string         `json:"source_decision"`
	AcceptanceType   string         `json:"acceptance_type"`
	Status           string         `json:"status"`
	Owner            string         `json:"owner"`
	Reason           string         `json:"reason"`
	RequiredEvidence []string       `json:"required_evidence"`
	AuditActions     []string       `json:"audit_actions"`
	RollbackPlan     string         `json:"rollback_plan"`
	ReviewRequired   bool           `json:"review_required"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseExceptionSchemaPreviewResponse struct {
	Status                     string                                  `json:"status"`
	Mode                       string                                  `json:"mode"`
	Scope                      string                                  `json:"scope"`
	ProjectKey                 string                                  `json:"project_key,omitempty"`
	ReadinessScope             string                                  `json:"readiness_scope"`
	ClaimScope                 string                                  `json:"claim_scope"`
	NotReal100                 bool                                    `json:"not_real_100"`
	EvidenceOnly               bool                                    `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                    `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                  `json:"release_candidate_decision"`
	Real100Status              string                                  `json:"real_100_status"`
	Real100Blockers            []string                                `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                `json:"real_100_breakdown"`
	RecordPreview              releaseExceptionRecordPreviewResponse   `json:"record_preview"`
	Tables                     []releaseExceptionSchemaTableResponse   `json:"tables"`
	ApplySteps                 []releaseExceptionMigrationStepResponse `json:"apply_steps"`
	RollbackSteps              []releaseExceptionMigrationStepResponse `json:"rollback_steps"`
	AuditActions               []string                                `json:"audit_actions"`
	Capabilities               []string                                `json:"capabilities"`
	ForbiddenActions           []string                                `json:"forbidden_actions"`
	GeneratedAt                string                                  `json:"generated_at"`
}

type releaseExceptionSchemaTableResponse struct {
	Name        string                                     `json:"name"`
	Purpose     string                                     `json:"purpose"`
	Columns     []releaseExceptionSchemaColumnResponse     `json:"columns"`
	Indexes     []releaseExceptionSchemaIndexResponse      `json:"indexes"`
	ForeignKeys []releaseExceptionSchemaForeignKeyResponse `json:"foreign_keys"`
}

type releaseExceptionSchemaColumnResponse struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Purpose  string `json:"purpose"`
}

type releaseExceptionSchemaIndexResponse struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Purpose string   `json:"purpose"`
}

type releaseExceptionSchemaForeignKeyResponse struct {
	Column           string `json:"column"`
	ReferencesTable  string `json:"references_table"`
	ReferencesColumn string `json:"references_column"`
	OnDelete         string `json:"on_delete"`
}

type releaseExceptionMigrationStepResponse struct {
	Order       int    `json:"order"`
	Action      string `json:"action"`
	Description string `json:"description"`
	SQLPreview  string `json:"sql_preview"`
}

type releaseExceptionMigrationApprovalGateResponse struct {
	Status                     string                                          `json:"status"`
	Mode                       string                                          `json:"mode"`
	Scope                      string                                          `json:"scope"`
	ProjectKey                 string                                          `json:"project_key,omitempty"`
	ReadinessScope             string                                          `json:"readiness_scope"`
	ClaimScope                 string                                          `json:"claim_scope"`
	NotReal100                 bool                                            `json:"not_real_100"`
	EvidenceOnly               bool                                            `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                            `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                          `json:"release_candidate_decision"`
	Real100Status              string                                          `json:"real_100_status"`
	Real100Blockers            []string                                        `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                        `json:"real_100_breakdown"`
	SchemaPreview              releaseExceptionSchemaPreviewResponse           `json:"schema_preview"`
	Items                      []releaseExceptionMigrationApprovalItemResponse `json:"items"`
	Capabilities               []string                                        `json:"capabilities"`
	ForbiddenActions           []string                                        `json:"forbidden_actions"`
	GeneratedAt                string                                          `json:"generated_at"`
}

type releaseExceptionMigrationApprovalItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	ApprovalStatus   string         `json:"approval_status"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseExceptionApplyPreviewResponse struct {
	Status                     string                                        `json:"status"`
	Mode                       string                                        `json:"mode"`
	Scope                      string                                        `json:"scope"`
	ProjectKey                 string                                        `json:"project_key,omitempty"`
	ReadinessScope             string                                        `json:"readiness_scope"`
	ClaimScope                 string                                        `json:"claim_scope"`
	NotReal100                 bool                                          `json:"not_real_100"`
	EvidenceOnly               bool                                          `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                          `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                        `json:"release_candidate_decision"`
	Real100Status              string                                        `json:"real_100_status"`
	Real100Blockers            []string                                      `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                      `json:"real_100_breakdown"`
	MigrationGate              releaseExceptionMigrationApprovalGateResponse `json:"migration_gate"`
	Items                      []releaseExceptionApplyPreviewItemResponse    `json:"items"`
	ApplySteps                 []releaseExceptionApplyPreviewStepResponse    `json:"apply_steps"`
	RollbackSteps              []releaseExceptionApplyPreviewStepResponse    `json:"rollback_steps"`
	Capabilities               []string                                      `json:"capabilities"`
	ForbiddenActions           []string                                      `json:"forbidden_actions"`
	GeneratedAt                string                                        `json:"generated_at"`
}

type releaseExceptionApplyPreviewItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Action           string         `json:"action"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseExceptionApplyPreviewStepResponse struct {
	Order       int      `json:"order"`
	Action      string   `json:"action"`
	Description string   `json:"description"`
	BlockedBy   []string `json:"blocked_by"`
}

type releaseFinalGateResponse struct {
	Status                     string                               `json:"status"`
	Mode                       string                               `json:"mode"`
	Scope                      string                               `json:"scope"`
	ProjectKey                 string                               `json:"project_key,omitempty"`
	ReadinessScope             string                               `json:"readiness_scope"`
	ClaimScope                 string                               `json:"claim_scope"`
	NotReal100                 bool                                 `json:"not_real_100"`
	EvidenceOnly               bool                                 `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                 `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                               `json:"release_candidate_decision"`
	Real100Status              string                               `json:"real_100_status"`
	Real100Blockers            []string                             `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown             `json:"real_100_breakdown"`
	Readiness                  releaseReadinessResponse             `json:"readiness"`
	AcceptanceGate             releaseAcceptanceGateResponse        `json:"acceptance_gate"`
	ExceptionApply             releaseExceptionApplyPreviewResponse `json:"exception_apply"`
	Items                      []releaseFinalGateItemResponse       `json:"items"`
	Capabilities               []string                             `json:"capabilities"`
	ForbiddenActions           []string                             `json:"forbidden_actions"`
	GeneratedAt                string                               `json:"generated_at"`
}

type releaseFinalGateItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseEvidenceBundleResponse struct {
	Status                     string                              `json:"status"`
	Mode                       string                              `json:"mode"`
	Scope                      string                              `json:"scope"`
	ProjectKey                 string                              `json:"project_key,omitempty"`
	ReadinessScope             string                              `json:"readiness_scope"`
	ClaimScope                 string                              `json:"claim_scope"`
	NotReal100                 bool                                `json:"not_real_100"`
	EvidenceOnly               bool                                `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                              `json:"release_candidate_decision"`
	Real100Status              string                              `json:"real_100_status"`
	Real100Blockers            []string                            `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown            `json:"real_100_breakdown"`
	BundleHash                 string                              `json:"bundle_hash"`
	FinalGate                  releaseFinalGateResponse            `json:"final_gate"`
	Backup                     backupManifestResponse              `json:"backup"`
	AuditCoverage              auditCoverageResponse               `json:"audit_coverage"`
	Items                      []releaseEvidenceBundleItemResponse `json:"items"`
	Capabilities               []string                            `json:"capabilities"`
	ForbiddenActions           []string                            `json:"forbidden_actions"`
	GeneratedAt                string                              `json:"generated_at"`
}

type releaseEvidenceBundleItemResponse struct {
	Key         string         `json:"key"`
	Category    string         `json:"category"`
	Status      string         `json:"status"`
	Source      string         `json:"source"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

type releasePackagePreviewResponse struct {
	Status                     string                              `json:"status"`
	Mode                       string                              `json:"mode"`
	Scope                      string                              `json:"scope"`
	ProjectKey                 string                              `json:"project_key,omitempty"`
	ReadinessScope             string                              `json:"readiness_scope"`
	ClaimScope                 string                              `json:"claim_scope"`
	NotReal100                 bool                                `json:"not_real_100"`
	EvidenceOnly               bool                                `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                              `json:"release_candidate_decision"`
	Real100Status              string                              `json:"real_100_status"`
	Real100Blockers            []string                            `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown            `json:"real_100_breakdown"`
	EvidenceBundle             releaseEvidenceBundleResponse       `json:"evidence_bundle"`
	PackageName                string                              `json:"package_name"`
	Items                      []releasePackagePreviewItemResponse `json:"items"`
	Capabilities               []string                            `json:"capabilities"`
	ForbiddenActions           []string                            `json:"forbidden_actions"`
	GeneratedAt                string                              `json:"generated_at"`
}

type releasePackagePreviewItemResponse struct {
	Key         string         `json:"key"`
	Category    string         `json:"category"`
	Status      string         `json:"status"`
	PackagePath string         `json:"package_path"`
	Source      string         `json:"source"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

type releaseDistributionPreviewResponse struct {
	Status                     string                                   `json:"status"`
	Mode                       string                                   `json:"mode"`
	Scope                      string                                   `json:"scope"`
	ProjectKey                 string                                   `json:"project_key,omitempty"`
	ReadinessScope             string                                   `json:"readiness_scope"`
	ClaimScope                 string                                   `json:"claim_scope"`
	NotReal100                 bool                                     `json:"not_real_100"`
	EvidenceOnly               bool                                     `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                     `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                   `json:"release_candidate_decision"`
	Real100Status              string                                   `json:"real_100_status"`
	Real100Blockers            []string                                 `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                 `json:"real_100_breakdown"`
	PackagePreview             releasePackagePreviewResponse            `json:"package_preview"`
	Items                      []releaseDistributionPreviewItemResponse `json:"items"`
	Capabilities               []string                                 `json:"capabilities"`
	ForbiddenActions           []string                                 `json:"forbidden_actions"`
	GeneratedAt                string                                   `json:"generated_at"`
}

type releaseDistributionPreviewItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Channel          string         `json:"channel"`
	Action           string         `json:"action"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releasePublishGateResponse struct {
	Status                     string                             `json:"status"`
	Mode                       string                             `json:"mode"`
	Scope                      string                             `json:"scope"`
	ProjectKey                 string                             `json:"project_key,omitempty"`
	ReadinessScope             string                             `json:"readiness_scope"`
	ClaimScope                 string                             `json:"claim_scope"`
	NotReal100                 bool                               `json:"not_real_100"`
	EvidenceOnly               bool                               `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                               `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                             `json:"release_candidate_decision"`
	Real100Status              string                             `json:"real_100_status"`
	Real100Blockers            []string                           `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown           `json:"real_100_breakdown"`
	DistributionPreview        releaseDistributionPreviewResponse `json:"distribution_preview"`
	Items                      []releasePublishGateItemResponse   `json:"items"`
	Capabilities               []string                           `json:"capabilities"`
	ForbiddenActions           []string                           `json:"forbidden_actions"`
	GeneratedAt                string                             `json:"generated_at"`
}

type releasePublishGateItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Channel          string         `json:"channel"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releasePublishApprovalPreviewResponse struct {
	Status                     string                                      `json:"status"`
	Mode                       string                                      `json:"mode"`
	Scope                      string                                      `json:"scope"`
	ProjectKey                 string                                      `json:"project_key,omitempty"`
	ReadinessScope             string                                      `json:"readiness_scope"`
	ClaimScope                 string                                      `json:"claim_scope"`
	NotReal100                 bool                                        `json:"not_real_100"`
	EvidenceOnly               bool                                        `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                        `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                      `json:"release_candidate_decision"`
	Real100Status              string                                      `json:"real_100_status"`
	Real100Blockers            []string                                    `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                    `json:"real_100_breakdown"`
	PublishGate                releasePublishGateResponse                  `json:"publish_gate"`
	Items                      []releasePublishApprovalPreviewItemResponse `json:"items"`
	Capabilities               []string                                    `json:"capabilities"`
	ForbiddenActions           []string                                    `json:"forbidden_actions"`
	GeneratedAt                string                                      `json:"generated_at"`
}

type releasePublishApprovalPreviewItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	ApprovalStatus   string         `json:"approval_status"`
	Channel          string         `json:"channel"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseRolloutPlanPreviewResponse struct {
	Status                     string                                  `json:"status"`
	Mode                       string                                  `json:"mode"`
	Scope                      string                                  `json:"scope"`
	ProjectKey                 string                                  `json:"project_key,omitempty"`
	ReadinessScope             string                                  `json:"readiness_scope"`
	ClaimScope                 string                                  `json:"claim_scope"`
	NotReal100                 bool                                    `json:"not_real_100"`
	EvidenceOnly               bool                                    `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                    `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                  `json:"release_candidate_decision"`
	Real100Status              string                                  `json:"real_100_status"`
	Real100Blockers            []string                                `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                `json:"real_100_breakdown"`
	PublishApprovalPreview     releasePublishApprovalPreviewResponse   `json:"publish_approval_preview"`
	Items                      []releaseRolloutPlanPreviewItemResponse `json:"items"`
	RolloutSteps               []releaseRolloutPlanPreviewStepResponse `json:"rollout_steps"`
	VerificationCheckpoints    []releaseRolloutPlanPreviewStepResponse `json:"verification_checkpoints"`
	RollbackSteps              []releaseRolloutPlanPreviewStepResponse `json:"rollback_steps"`
	Capabilities               []string                                `json:"capabilities"`
	ForbiddenActions           []string                                `json:"forbidden_actions"`
	GeneratedAt                string                                  `json:"generated_at"`
}

type releaseRolloutPlanPreviewItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Stage            string         `json:"stage"`
	Action           string         `json:"action"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseRolloutPlanPreviewStepResponse struct {
	Order       int      `json:"order"`
	Stage       string   `json:"stage"`
	Action      string   `json:"action"`
	Description string   `json:"description"`
	BlockedBy   []string `json:"blocked_by"`
}

type projectInventoryResponse struct {
	Versions        int64 `json:"versions"`
	Residuals       int64 `json:"residuals"`
	Artifacts       int64 `json:"artifacts"`
	ImportSnapshots int64 `json:"import_snapshots"`
	MirrorExports   int64 `json:"mirror_exports"`
}

type projectImportResponse struct {
	SourceHash             string         `json:"source_hash"`
	CreatedAt              string         `json:"created_at"`
	Summary                map[string]any `json:"summary"`
	HasPrevious            bool           `json:"has_previous"`
	PreviousSourceHash     string         `json:"previous_source_hash,omitempty"`
	PreviousCreatedAt      string         `json:"previous_created_at,omitempty"`
	HistoryReadyForDiff    bool           `json:"history_ready_for_diff"`
	SourceHashChangedSince bool           `json:"source_hash_changed_since_previous"`
}

type projectDoctorResponse struct {
	Status              string         `json:"status"`
	DriftStatus         string         `json:"drift_status"`
	ConfigDriftStatus   string         `json:"config_drift_status"`
	StageCoverageStatus string         `json:"stage_coverage_status"`
	NativeDoctorStatus  string         `json:"native_doctor_status"`
	Severity            string         `json:"severity"`
	CreatedAt           string         `json:"created_at"`
	Metadata            map[string]any `json:"metadata"`
}

type projectDoctorRequest struct {
	AllowNative    bool   `json:"allow_native"`
	IdempotencyKey string `json:"idempotency_key"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
}

type projectImportRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
}

type shimReadinessEvidenceRequest struct {
	EvidenceKey    string         `json:"evidence_key"`
	Status         string         `json:"status"`
	Summary        string         `json:"summary"`
	EvidenceURI    string         `json:"evidence_uri"`
	IdempotencyKey string         `json:"idempotency_key"`
	Actor          string         `json:"actor"`
	Reason         string         `json:"reason"`
	Metadata       map[string]any `json:"metadata"`
}

type shimReadinessEvidenceResponse struct {
	Project                 projectRecordResponse `json:"project"`
	EvidenceKey             string                `json:"evidence_key"`
	Status                  string                `json:"status"`
	Decision                string                `json:"decision"`
	Message                 string                `json:"message"`
	EventID                 int64                 `json:"event_id,omitempty"`
	AuditEventID            int64                 `json:"audit_event_id,omitempty"`
	IdempotencyKey          string                `json:"idempotency_key"`
	Created                 bool                  `json:"created"`
	ProjectWriteAttempted   bool                  `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                  `json:"execution_write_attempted"`
	EngineCallAttempted     bool                  `json:"engine_call_attempted"`
	Metadata                map[string]any        `json:"metadata"`
}

type projectImportRunResponse struct {
	Project        projectRecordResponse `json:"project"`
	Versions       int                   `json:"versions"`
	Residuals      int                   `json:"residuals"`
	Artifacts      int                   `json:"artifacts"`
	ActiveTasks    int                   `json:"active_tasks"`
	V1Done         int                   `json:"v1_done"`
	V1Total        int                   `json:"v1_total"`
	StatusSnapshot string                `json:"status_snapshot"`
	RunID          int64                 `json:"run_id"`
	IdempotencyKey string                `json:"idempotency_key"`
	Created        bool                  `json:"created"`
}

type projectDoctorRunResponse struct {
	Project        projectRecordResponse `json:"project"`
	Report         map[string]any        `json:"report"`
	EventID        int64                 `json:"event_id,omitempty"`
	Severity       string                `json:"severity"`
	OverallStatus  string                `json:"overall_status"`
	IdempotencyKey string                `json:"idempotency_key"`
	Created        bool                  `json:"created"`
}

type projectReadinessResponse struct {
	Project projectRecordResponse          `json:"project"`
	Status  string                         `json:"status"`
	Items   []projectReadinessItemResponse `json:"items"`
	Summary projectSummaryResponse         `json:"summary"`
}

type generatedWriteReadinessResponse struct {
	Project                       projectRecordResponse          `json:"project"`
	Status                        string                         `json:"status"`
	Mode                          string                         `json:"mode"`
	Items                         []projectReadinessItemResponse `json:"items"`
	RequiredCapabilities          []string                       `json:"required_capabilities"`
	AllowedGeneratedPrefixes      []string                       `json:"allowed_generated_prefixes"`
	RequiredWritePaths            []string                       `json:"required_write_paths"`
	ConfiguredWritePaths          []string                       `json:"configured_write_paths"`
	ConfiguredForbiddenPaths      []string                       `json:"configured_forbidden_paths"`
	Blockers                      []string                       `json:"blockers"`
	ReviewBlockers                []string                       `json:"review_blockers"`
	ForbiddenActions              []string                       `json:"forbidden_actions"`
	ReadyForReview                bool                           `json:"ready_for_review"`
	ApplyOpen                     bool                           `json:"apply_open"`
	RealAreaMatrixWriteOpened     bool                           `json:"real_areamatrix_write_opened"`
	GeneratedOnly                 bool                           `json:"generated_only"`
	ProjectConfigRead             bool                           `json:"project_config_read"`
	ProjectReadAttempted          bool                           `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                           `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                           `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                           `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                           `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                           `json:"engine_call_attempted"`
	CommandsRun                   bool                           `json:"commands_run"`
	SecretsResolved               bool                           `json:"secrets_resolved"`
	NetworkUsed                   bool                           `json:"network_used"`
	TaskClaimed                   bool                           `json:"task_claimed"`
	WorkerStarted                 bool                           `json:"worker_started"`
	LeaseCreated                  bool                           `json:"lease_created"`
	AttemptCreated                bool                           `json:"attempt_created"`
	ArtifactCreated               bool                           `json:"artifact_created"`
	GeneratedAt                   string                         `json:"generated_at"`
}

type generatedWriteApplyBetaGateResponse struct {
	Project                       projectRecordResponse                     `json:"project"`
	Status                        string                                    `json:"status"`
	Mode                          string                                    `json:"mode"`
	Readiness                     generatedWriteReadinessResponse           `json:"readiness"`
	Items                         []generatedWriteApplyBetaGateItemResponse `json:"items"`
	RequiredCapabilities          []string                                  `json:"required_capabilities"`
	AllowedGeneratedPrefixes      []string                                  `json:"allowed_generated_prefixes"`
	RequiredEvidence              []string                                  `json:"required_evidence"`
	ForbiddenActions              []string                                  `json:"forbidden_actions"`
	ApprovalRequired              bool                                      `json:"approval_required"`
	ApprovalStatus                string                                    `json:"approval_status"`
	ApplyOpen                     bool                                      `json:"apply_open"`
	RealAreaMatrixWriteOpened     bool                                      `json:"real_areamatrix_write_opened"`
	GeneratedOnly                 bool                                      `json:"generated_only"`
	ProjectReadAttempted          bool                                      `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                                      `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                                      `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                                      `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                                      `json:"engine_call_attempted"`
	CommandsRun                   bool                                      `json:"commands_run"`
	SecretsResolved               bool                                      `json:"secrets_resolved"`
	NetworkUsed                   bool                                      `json:"network_used"`
	TaskClaimed                   bool                                      `json:"task_claimed"`
	WorkerStarted                 bool                                      `json:"worker_started"`
	LeaseCreated                  bool                                      `json:"lease_created"`
	AttemptCreated                bool                                      `json:"attempt_created"`
	ArtifactCreated               bool                                      `json:"artifact_created"`
	GeneratedAt                   string                                    `json:"generated_at"`
}

type generatedWriteApplyBetaGateItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	ApprovalStatus   string         `json:"approval_status,omitempty"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type projectReadinessItemResponse struct {
	Key      string         `json:"key"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type projectImportDiffResponse struct {
	Project       projectRecordResponse       `json:"project"`
	Status        string                      `json:"status"`
	HasPrevious   bool                        `json:"has_previous"`
	SourceChanged bool                        `json:"source_changed"`
	Latest        projectDiffSnapshotResponse `json:"latest"`
	Previous      projectDiffSnapshotResponse `json:"previous,omitempty"`
	Changes       []projectDiffChangeResponse `json:"changes"`
}

type projectDiffSnapshotResponse struct {
	SourceHash string `json:"source_hash"`
	CreatedAt  string `json:"created_at"`
}

type projectDiffChangeResponse struct {
	Key      string `json:"key"`
	Status   string `json:"status"`
	Previous string `json:"previous"`
	Latest   string `json:"latest"`
}

type projectVerificationBundleResponse struct {
	Project    projectRecordResponse     `json:"project"`
	Status     string                    `json:"status"`
	PhaseGate  projectPhaseGateResponse  `json:"phase_gate"`
	Summary    projectSummaryResponse    `json:"summary"`
	Readiness  projectReadinessResponse  `json:"readiness"`
	ImportDiff projectImportDiffResponse `json:"import_diff"`
	Events     []eventResponse           `json:"events"`
}

type projectCutoverReadinessResponse struct {
	Project         projectRecordResponse             `json:"project"`
	WorkflowVersion workflowVersionResponse           `json:"workflow_version"`
	Status          string                            `json:"status"`
	PhaseGate       projectPhaseGateResponse          `json:"phase_gate"`
	Items           []projectReadinessItemResponse    `json:"items"`
	Verification    projectVerificationBundleResponse `json:"verification"`
	Compatibility   compatibilityContractResponse     `json:"compatibility"`
	Gates           []gateResultResponse              `json:"gates"`
}

type projectCutoverApplyRequest struct {
	VersionLabel   string `json:"version"`
	IdempotencyKey string `json:"idempotency_key"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	Mode           string `json:"mode"`
}

type projectCutoverApplyResponse struct {
	Project                  projectRecordResponse   `json:"project"`
	WorkflowVersion          workflowVersionResponse `json:"workflow_version"`
	Status                   string                  `json:"status"`
	Decision                 string                  `json:"decision"`
	Message                  string                  `json:"message"`
	Blockers                 []string                `json:"blockers"`
	Warnings                 []string                `json:"warnings"`
	EventID                  int64                   `json:"event_id,omitempty"`
	AuditEventID             int64                   `json:"audit_event_id,omitempty"`
	IdempotencyKey           string                  `json:"idempotency_key"`
	Created                  bool                    `json:"created"`
	ProjectWriteAttempted    bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted  bool                    `json:"execution_write_attempted"`
	AreaMatrixWriteAttempted bool                    `json:"area_matrix_write_attempted"`
	CutoverReadinessGateID   int64                   `json:"cutover_readiness_gate_id,omitempty"`
}

type executionForwardingV1ApplyRequest struct {
	AllowedTaskTypes           []string `json:"allowed_task_types"`
	ApprovalID                 string   `json:"approval_id"`
	ApprovalScope              string   `json:"approval_scope"`
	ReadinessSnapshotHash      string   `json:"readiness_snapshot_hash"`
	ExpectedShimLifecycleState string   `json:"expected_shim_lifecycle_state"`
	LegacyNonWriteProofID      string   `json:"legacy_non_write_proof_id"`
	RollbackPlanID             string   `json:"rollback_plan_id"`
	ProtectedPathFingerprintID string   `json:"protected_path_fingerprint_id"`
	IdempotencyKey             string   `json:"idempotency_key"`
	AuditCorrelationID         string   `json:"audit_correlation_id"`
	FailureMode                string   `json:"failure_mode"`
	ExplicitApproval           bool     `json:"explicit_approval"`
	ApprovalActor              string   `json:"approval_actor"`
	ApprovalReason             string   `json:"approval_reason"`
	Actor                      string   `json:"actor"`
	Reason                     string   `json:"reason"`
}

type executionForwardingV1ApplyResponse struct {
	Project                         projectRecordResponse                  `json:"project"`
	Status                          string                                 `json:"status"`
	Decision                        string                                 `json:"decision"`
	Message                         string                                 `json:"message"`
	Blockers                        []string                               `json:"blockers"`
	Gate                            executionForwardingV1ApplyGateResponse `json:"gate"`
	EventID                         int64                                  `json:"event_id,omitempty"`
	AuditEventID                    int64                                  `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string                                 `json:"idempotency_key"`
	Created                         bool                                   `json:"created"`
	SafetyFacts                     map[string]bool                        `json:"safety_facts"`
	CommandRequestCreated           bool                                   `json:"command_request_created"`
	AreaFlowCommandCreated          bool                                   `json:"area_flow_command_created"`
	AreaFlowRunCreated              bool                                   `json:"area_flow_run_created"`
	AreaFlowRunTaskCreated          bool                                   `json:"area_flow_run_task_created"`
	AreaFlowRunAttemptCreated       bool                                   `json:"area_flow_run_attempt_created"`
	AreaFlowArtifactCreated         bool                                   `json:"area_flow_artifact_created"`
	AreaFlowAuditEventCreated       bool                                   `json:"area_flow_audit_event_created"`
	TaskLoopRunForwarded            bool                                   `json:"task_loop_run_forwarded"`
	LegacyTaskLoopStarted           bool                                   `json:"legacy_task_loop_started"`
	LegacyProgressWritten           bool                                   `json:"legacy_progress_written"`
	LegacyLogsWritten               bool                                   `json:"legacy_logs_written"`
	LegacyCheckpointWritten         bool                                   `json:"legacy_checkpoint_written"`
	ProjectWriteAttempted           bool                                   `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool                                   `json:"execution_write_attempted"`
	EngineCallAttempted             bool                                   `json:"engine_call_attempted"`
	CommandsRun                     bool                                   `json:"commands_run"`
	SecretsResolved                 bool                                   `json:"secrets_resolved"`
	NetworkUsed                     bool                                   `json:"network_used"`
	AreaMatrixProtectedPathsTouched bool                                   `json:"areamatrix_protected_paths_touched"`
	GeneratedAt                     string                                 `json:"generated_at"`
}

type statusProjectionApplyRequest struct {
	TargetURI                      string `json:"target_uri"`
	Actor                          string `json:"actor"`
	Reason                         string `json:"reason"`
	IdempotencyKey                 string `json:"idempotency_key"`
	ExpectedBeforeExists           *bool  `json:"expected_before_exists"`
	ExpectedBeforeSHA256           string `json:"expected_before_sha256"`
	ExpectedBeforeSizeBytes        *int64 `json:"expected_before_size"`
	SourceHash                     string `json:"source_hash"`
	SchemaURI                      string `json:"schema_uri"`
	ValidatorPreflight             string `json:"validator_preflight"`
	ProtectedPathCheck             string `json:"protected_path_check"`
	ProtectedPathFingerprintSHA256 string `json:"protected_path_fingerprint_sha256"`
	RollbackAction                 string `json:"rollback_action"`
	AcceptedPreimageSchemaStatus   string `json:"accept_preimage_schema"`
	ExplicitApproval               bool   `json:"explicit_approval"`
	ApprovalActor                  string `json:"approval_actor"`
	ApprovalReason                 string `json:"approval_reason"`
}

type statusProjectionApplyResponse struct {
	Project                   projectRecordResponse `json:"project"`
	Status                    string                `json:"status"`
	Decision                  string                `json:"decision"`
	Message                   string                `json:"message"`
	Blockers                  []string              `json:"blockers"`
	EventID                   int64                 `json:"event_id,omitempty"`
	AuditEventID              int64                 `json:"audit_event_id,omitempty"`
	SnapshotID                int64                 `json:"snapshot_id,omitempty"`
	StatusProjectionID        int64                 `json:"status_projection_id,omitempty"`
	TargetKind                string                `json:"target_kind"`
	TargetURI                 string                `json:"target_uri"`
	WrittenTarget             string                `json:"written_target,omitempty"`
	WriteHash                 string                `json:"write_hash,omitempty"`
	WriteSize                 int64                 `json:"write_size,omitempty"`
	PreimageCaptured          bool                  `json:"preimage_captured"`
	PreimageExists            bool                  `json:"preimage_exists"`
	PreimageSHA256            string                `json:"preimage_sha256,omitempty"`
	PreimageSize              int64                 `json:"preimage_size,omitempty"`
	PostWriteVerified         bool                  `json:"post_write_verified"`
	PostWriteSHA256           string                `json:"post_write_sha256,omitempty"`
	PostWriteSize             int64                 `json:"post_write_size,omitempty"`
	ProtectedPathsVerified    bool                  `json:"protected_paths_verified"`
	ProtectedPathBeforeHash   string                `json:"protected_path_before_hash,omitempty"`
	ProtectedPathAfterHash    string                `json:"protected_path_after_hash,omitempty"`
	ExpectedProtectedPathHash string                `json:"expected_protected_path_hash,omitempty"`
	RootContained             bool                  `json:"root_contained"`
	StableProjectionValid     bool                  `json:"stable_projection_validated"`
	AtomicReplaceUsed         bool                  `json:"atomic_replace_used"`
	RollbackCompensation      bool                  `json:"rollback_compensation_enabled"`
	SourceHash                string                `json:"source_hash,omitempty"`
	SummaryState              string                `json:"summary_state"`
	ApplyGateStatus           string                `json:"apply_gate_status"`
	ApplyGateDecision         string                `json:"apply_gate_decision"`
	ApplyGateApprovalStatus   string                `json:"apply_gate_approval_status"`
	ApplyCommandEligible      bool                  `json:"apply_command_eligible"`
	IdempotencyKey            string                `json:"idempotency_key"`
	Created                   bool                  `json:"created"`
	GeneratedAt               string                `json:"generated_at"`
	ProjectWriteAttempted     bool                  `json:"project_write_attempted"`
	ExecutionWriteAttempted   bool                  `json:"execution_write_attempted"`
	EngineCallAttempted       bool                  `json:"engine_call_attempted"`
}

type statusProjectionAuthorizationPreviewResponse struct {
	Project                                       projectRecordResponse                           `json:"project"`
	Status                                        string                                          `json:"status"`
	Mode                                          string                                          `json:"mode"`
	ClaimScope                                    string                                          `json:"claim_scope"`
	NotReal100                                    bool                                            `json:"not_real_100"`
	Decision                                      string                                          `json:"decision"`
	Message                                       string                                          `json:"message"`
	TargetKind                                    string                                          `json:"target_kind"`
	TargetURI                                     string                                          `json:"target_uri"`
	TargetPath                                    string                                          `json:"target_path"`
	SchemaURI                                     string                                          `json:"schema_uri"`
	ValidatorPreflight                            string                                          `json:"validator_preflight"`
	ProtectedPathFingerprintSHA256                string                                          `json:"protected_path_fingerprint_sha256,omitempty"`
	SourceHash                                    string                                          `json:"source_hash,omitempty"`
	SummaryState                                  string                                          `json:"summary_state"`
	RequiredAuthorizationPhrase                   string                                          `json:"required_authorization_phrase,omitempty"`
	Permission                                    statusProjectionAuthorizationPermissionResponse `json:"permission"`
	Preimage                                      statusProjectionPreimageResponse                `json:"preimage"`
	WriteSet                                      []statusProjectionWriteSetEntryResponse         `json:"write_set"`
	RequiredPreflight                             []string                                        `json:"required_preflight"`
	RequiredPacketFields                          []string                                        `json:"required_packet_fields"`
	RequiredCapabilities                          []string                                        `json:"required_capabilities"`
	ProtectedPaths                                []string                                        `json:"protected_paths"`
	RollbackPlan                                  []string                                        `json:"rollback_plan"`
	BlockedBy                                     []string                                        `json:"blocked_by"`
	Warnings                                      []string                                        `json:"warnings"`
	ForbiddenActions                              []string                                        `json:"forbidden_actions"`
	SafetyFacts                                   map[string]bool                                 `json:"safety_facts"`
	ApplyOpen                                     bool                                            `json:"apply_open"`
	ApprovalRequired                              bool                                            `json:"approval_required"`
	ApprovalStatus                                string                                          `json:"approval_status"`
	WouldCreateCommandRequestAfterApproval        bool                                            `json:"would_create_command_request_after_approval"`
	WouldCreateProjectStatusSnapshotAfterApproval bool                                            `json:"would_create_project_status_snapshot_after_approval"`
	WouldCreateStatusProjectionAfterApproval      bool                                            `json:"would_create_status_projection_after_approval"`
	WouldCreateEventAfterApproval                 bool                                            `json:"would_create_event_after_approval"`
	WouldCreateAuditEventAfterApproval            bool                                            `json:"would_create_audit_event_after_approval"`
	WouldWriteProjectFileAfterApproval            bool                                            `json:"would_write_project_file_after_approval"`
	WouldWriteExecutionAfterApproval              bool                                            `json:"would_write_execution_after_approval"`
	WouldRunEngineAfterApproval                   bool                                            `json:"would_run_engine_after_approval"`
	ProjectWriteAttempted                         bool                                            `json:"project_write_attempted"`
	ExecutionWriteAttempted                       bool                                            `json:"execution_write_attempted"`
	EngineCallAttempted                           bool                                            `json:"engine_call_attempted"`
	GeneratedAt                                   string                                          `json:"generated_at"`
}

type statusProjectionApplyGateResponse struct {
	Project                        projectRecordResponse                        `json:"project"`
	Status                         string                                       `json:"status"`
	Mode                           string                                       `json:"mode"`
	ClaimScope                     string                                       `json:"claim_scope"`
	NotReal100                     bool                                         `json:"not_real_100"`
	Decision                       string                                       `json:"decision"`
	Message                        string                                       `json:"message"`
	TargetURI                      string                                       `json:"target_uri"`
	TargetPath                     string                                       `json:"target_path"`
	Authorization                  statusProjectionAuthorizationPreviewResponse `json:"authorization"`
	Items                          []statusProjectionApplyGateItemResponse      `json:"items"`
	RequiredPacketFields           []string                                     `json:"required_packet_fields"`
	RequiredCapabilities           []string                                     `json:"required_capabilities"`
	RequiredAuthorizationPhrase    string                                       `json:"required_authorization_phrase,omitempty"`
	ProtectedPaths                 []string                                     `json:"protected_paths"`
	ForbiddenActions               []string                                     `json:"forbidden_actions"`
	SafetyFacts                    map[string]bool                              `json:"safety_facts"`
	ApplyCommandEligible           bool                                         `json:"apply_command_eligible"`
	ApplyCommandEligibleIsNotApply bool                                         `json:"apply_command_eligible_is_not_apply"`
	RequiresSeparateApplyCommand   bool                                         `json:"requires_separate_apply_command"`
	ApprovalRequired               bool                                         `json:"approval_required"`
	ApprovalStatus                 string                                       `json:"approval_status"`
	ProjectWriteAttempted          bool                                         `json:"project_write_attempted"`
	ExecutionWriteAttempted        bool                                         `json:"execution_write_attempted"`
	EngineCallAttempted            bool                                         `json:"engine_call_attempted"`
	CommandRequestCreated          bool                                         `json:"command_request_created"`
	StatusProjectionWritten        bool                                         `json:"status_projection_written"`
	GeneratedAt                    string                                       `json:"generated_at"`
}

type statusProjectionApplyPacketPreviewResponse struct {
	Project                                      projectRecordResponse                        `json:"project"`
	Status                                       string                                       `json:"status"`
	Mode                                         string                                       `json:"mode"`
	ClaimScope                                   string                                       `json:"claim_scope"`
	NotReal100                                   bool                                         `json:"not_real_100"`
	Decision                                     string                                       `json:"decision"`
	Message                                      string                                       `json:"message"`
	Blockers                                     []string                                     `json:"blockers"`
	RequiredAuthorizationPhrase                  string                                       `json:"required_authorization_phrase,omitempty"`
	Authorization                                statusProjectionAuthorizationPreviewResponse `json:"authorization"`
	Gate                                         statusProjectionApplyGateResponse            `json:"gate"`
	Packet                                       statusProjectionApplyPacketResponse          `json:"packet"`
	ApplyCommand                                 []string                                     `json:"apply_command"`
	APIRequest                                   statusProjectionApplyAPIRequestResponse      `json:"api_request"`
	RequiredHumanReview                          []string                                     `json:"required_human_review"`
	ForbiddenActions                             []string                                     `json:"forbidden_actions"`
	SafetyFacts                                  map[string]bool                              `json:"safety_facts"`
	WouldCreateCommandRequestAfterApplyCommand   bool                                         `json:"would_create_command_request_after_apply_command"`
	WouldCreateStatusProjectionAfterApplyCommand bool                                         `json:"would_create_status_projection_after_apply_command"`
	WouldWriteProjectFileAfterApplyCommand       bool                                         `json:"would_write_project_file_after_apply_command"`
	ApplyCommandEligibleIsNotApply               bool                                         `json:"apply_command_eligible_is_not_apply"`
	RequiresSeparateApplyCommand                 bool                                         `json:"requires_separate_apply_command"`
	ProjectWriteAttempted                        bool                                         `json:"project_write_attempted"`
	ExecutionWriteAttempted                      bool                                         `json:"execution_write_attempted"`
	EngineCallAttempted                          bool                                         `json:"engine_call_attempted"`
	CommandRequestCreated                        bool                                         `json:"command_request_created"`
	StatusProjectionWritten                      bool                                         `json:"status_projection_written"`
	GeneratedAt                                  string                                       `json:"generated_at"`
}

type statusProjectionApplyPacketResponse struct {
	TargetURI                      string `json:"target_uri"`
	ExpectedBeforeExists           bool   `json:"expected_before_exists"`
	ExpectedBeforeSHA256           string `json:"expected_before_sha256"`
	ExpectedBeforeSizeBytes        int64  `json:"expected_before_size"`
	SourceHash                     string `json:"source_hash"`
	SchemaURI                      string `json:"schema_uri"`
	ValidatorPreflight             string `json:"validator_preflight"`
	ProtectedPathCheck             string `json:"protected_path_check"`
	ProtectedPathFingerprintSHA256 string `json:"protected_path_fingerprint_sha256"`
	RollbackAction                 string `json:"rollback_action"`
	AcceptedPreimageSchemaStatus   string `json:"accept_preimage_schema"`
	ExplicitApproval               bool   `json:"explicit_approval"`
	ApprovalActor                  string `json:"approval_actor"`
	ApprovalReason                 string `json:"approval_reason"`
	RequiredAuthorizationPhrase    string `json:"required_authorization_phrase,omitempty"`
}

type statusProjectionApplyAPIRequestResponse statusProjectionApplyPacketResponse

type statusProjectionApplyGateItemResponse struct {
	Key              string   `json:"key"`
	Category         string   `json:"category"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Expected         string   `json:"expected"`
	Actual           string   `json:"actual"`
	RequiredEvidence []string `json:"required_evidence"`
	BlockedBy        []string `json:"blocked_by"`
}

type statusProjectionAuthorizationPermissionResponse struct {
	Capability        string `json:"capability"`
	ResourceType      string `json:"resource_type"`
	TargetURI         string `json:"target_uri"`
	CapabilityAllowed bool   `json:"capability_allowed"`
	PathAllowed       bool   `json:"path_allowed"`
	Allowed           bool   `json:"allowed"`
	Reason            string `json:"reason"`
}

type statusProjectionPreimageResponse struct {
	TargetPath               string   `json:"target_path"`
	Exists                   bool     `json:"exists"`
	Readable                 bool     `json:"readable"`
	SizeBytes                int64    `json:"size_bytes"`
	SHA256                   string   `json:"sha256,omitempty"`
	SchemaStatus             string   `json:"schema_status"`
	LegacyShape              bool     `json:"legacy_shape"`
	MissingRequiredFields    []string `json:"missing_required_fields"`
	UnexpectedTopLevelFields []string `json:"unexpected_top_level_fields"`
	CompatibilityMissing     []string `json:"compatibility_missing"`
	CompatibilityUnexpected  []string `json:"compatibility_unexpected"`
	Message                  string   `json:"message"`
}

type statusProjectionWriteSetEntryResponse struct {
	TargetURI                string `json:"target_uri"`
	TargetPath               string `json:"target_path"`
	Operation                string `json:"operation"`
	Capability               string `json:"capability"`
	ExpectedBeforeExists     bool   `json:"expected_before_exists"`
	ExpectedBeforeSHA256     string `json:"expected_before_sha256,omitempty"`
	ExpectedBeforeSizeBytes  int64  `json:"expected_before_size_bytes"`
	RequiresPreimageMatch    bool   `json:"requires_preimage_match"`
	RequiresSchemaValidation bool   `json:"requires_schema_validation"`
	RollbackAction           string `json:"rollback_action"`
	ProtectedPath            bool   `json:"protected_path"`
}

type compatibilityContractResponse struct {
	Project  projectRecordResponse          `json:"project"`
	Status   string                         `json:"status"`
	Commands []compatibilityCommandResponse `json:"commands"`
}

type compatibilityCommandResponse struct {
	Command        string         `json:"command"`
	Mode           string         `json:"mode"`
	Status         string         `json:"status"`
	Message        string         `json:"message"`
	AreaFlowTarget string         `json:"areaflow_target,omitempty"`
	Fallback       string         `json:"fallback,omitempty"`
	BlockedReason  string         `json:"blocked_reason,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

type shimPreviewResponse struct {
	Project              projectRecordResponse         `json:"project"`
	Status               string                        `json:"status"`
	Mode                 string                        `json:"mode"`
	Contract             compatibilityContractResponse `json:"contract"`
	PlannedFiles         []shimFilePlanResponse        `json:"planned_files"`
	CommandMappings      []shimCommandMappingResponse  `json:"command_mappings"`
	DiscoveryOrder       []string                      `json:"discovery_order"`
	ForbiddenPaths       []string                      `json:"forbidden_paths"`
	ForbiddenCommands    []string                      `json:"forbidden_commands"`
	VerificationCommands []string                      `json:"verification_commands"`
	RollbackSteps        []string                      `json:"rollback_steps"`
	Notes                []string                      `json:"notes"`
}

type shimFilePlanResponse struct {
	Path     string `json:"path"`
	Action   string `json:"action"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
	Boundary string `json:"boundary"`
}

type shimCommandMappingResponse struct {
	Command        string `json:"command"`
	Mode           string `json:"mode"`
	Status         string `json:"status"`
	AreaFlowTarget string `json:"areaflow_target,omitempty"`
	Fallback       string `json:"fallback,omitempty"`
	BlockedReason  string `json:"blocked_reason,omitempty"`
	ReadOnly       bool   `json:"read_only"`
	RequiresNative bool   `json:"requires_native"`
	Message        string `json:"message"`
}

type shimReadinessResponse struct {
	Project projectRecordResponse       `json:"project"`
	Status  string                      `json:"status"`
	Preview shimPreviewResponse         `json:"preview"`
	Items   []shimReadinessItemResponse `json:"items"`
}

type shimAuthorizationPacketResponse struct {
	Project              projectRecordResponse  `json:"project"`
	Status               string                 `json:"status"`
	Mode                 string                 `json:"mode"`
	Intent               string                 `json:"intent"`
	ReadinessStatus      string                 `json:"readiness_status"`
	AllowedFiles         []shimFilePlanResponse `json:"allowed_files"`
	ForbiddenPaths       []string               `json:"forbidden_paths"`
	ForbiddenActions     []string               `json:"forbidden_actions"`
	RequiredPreflight    []string               `json:"required_preflight"`
	PostEditVerification []string               `json:"post_edit_verification"`
	RollbackScope        []string               `json:"rollback_scope"`
	SafetyFacts          map[string]bool        `json:"safety_facts"`
	NextRequiredApproval string                 `json:"next_required_approval"`
}

type shimApplyPacketPreviewResponse struct {
	Project                                        projectRecordResponse           `json:"project"`
	Status                                         string                          `json:"status"`
	Mode                                           string                          `json:"mode"`
	Decision                                       string                          `json:"decision"`
	Message                                        string                          `json:"message"`
	Authorization                                  shimAuthorizationPacketResponse `json:"authorization"`
	Gate                                           shimApplyGateResponse           `json:"gate"`
	Packet                                         shimApplyPacketResponse         `json:"packet"`
	ApplyGateCommand                               []string                        `json:"apply_gate_command"`
	FutureApplyCommand                             []string                        `json:"future_apply_command"`
	RequiredHumanReview                            []string                        `json:"required_human_review"`
	ForbiddenActions                               []string                        `json:"forbidden_actions"`
	SafetyFacts                                    map[string]bool                 `json:"safety_facts"`
	WouldCreateCommandRequestAfterApplyCommand     bool                            `json:"would_create_command_request_after_apply_command"`
	WouldWriteAreaMatrixShimFilesAfterApplyCommand bool                            `json:"would_write_area_matrix_shim_files_after_apply_command"`
	WouldWriteStatusProjectionAfterApplyCommand    bool                            `json:"would_write_status_projection_after_apply_command"`
	CommandRequestCreated                          bool                            `json:"command_request_created"`
	ProjectWriteAttempted                          bool                            `json:"project_write_attempted"`
	ExecutionWriteAttempted                        bool                            `json:"execution_write_attempted"`
	EngineCallAttempted                            bool                            `json:"engine_call_attempted"`
	TaskLoopRunForwarded                           bool                            `json:"task_loop_run_forwarded"`
	StatusProjectionWritten                        bool                            `json:"status_projection_written"`
	AreaMatrixFilesModified                        bool                            `json:"area_matrix_files_modified"`
	GeneratedAt                                    string                          `json:"generated_at"`
}

type shimApplyPacketResponse struct {
	CommandType                string   `json:"command_type"`
	ProjectKey                 string   `json:"project_key"`
	AllowedFiles               []string `json:"allowed_files"`
	ApprovalID                 string   `json:"approval_id"`
	ApprovalScope              string   `json:"approval_scope"`
	AuthorizationSnapshotHash  string   `json:"authorization_snapshot_hash"`
	ExpectedAuthorizationMode  string   `json:"expected_authorization_mode"`
	StatusProjectionPacketID   string   `json:"status_projection_packet_id"`
	StatusProjectionGateID     string   `json:"status_projection_gate_id"`
	ReadOnlySmokeEvidenceID    string   `json:"read_only_smoke_evidence_id"`
	DirtyWorktreeReviewID      string   `json:"dirty_worktree_review_id"`
	ProtectedPathFingerprintID string   `json:"protected_path_fingerprint_id"`
	RollbackPlanID             string   `json:"rollback_plan_id"`
	IdempotencyKey             string   `json:"idempotency_key"`
	AuditCorrelationID         string   `json:"audit_correlation_id"`
	FailureMode                string   `json:"failure_mode"`
	ExplicitApproval           bool     `json:"explicit_approval"`
	ApprovalActor              string   `json:"approval_actor"`
	ApprovalReason             string   `json:"approval_reason"`
}

type shimApplyGateResponse struct {
	Project                 projectRecordResponse       `json:"project"`
	Status                  string                      `json:"status"`
	Mode                    string                      `json:"mode"`
	Decision                string                      `json:"decision"`
	Message                 string                      `json:"message"`
	Items                   []shimApplyGateItemResponse `json:"items"`
	RequiredPacketFields    []string                    `json:"required_packet_fields"`
	RequiredCapabilities    []string                    `json:"required_capabilities"`
	AllowedFiles            []string                    `json:"allowed_files"`
	ForbiddenPaths          []string                    `json:"forbidden_paths"`
	ForbiddenActions        []string                    `json:"forbidden_actions"`
	RequiredPreflight       []string                    `json:"required_preflight"`
	PostEditVerification    []string                    `json:"post_edit_verification"`
	RollbackScope           []string                    `json:"rollback_scope"`
	RequiredProofFacts      []string                    `json:"required_proof_facts"`
	SafetyFacts             map[string]bool             `json:"safety_facts"`
	ApprovalRequired        bool                        `json:"approval_required"`
	ApprovalStatus          string                      `json:"approval_status"`
	ApplyCommandEligible    bool                        `json:"apply_command_eligible"`
	ApplyOpen               bool                        `json:"apply_open"`
	CommandRequestCreated   bool                        `json:"command_request_created"`
	ProjectWriteAttempted   bool                        `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                        `json:"execution_write_attempted"`
	EngineCallAttempted     bool                        `json:"engine_call_attempted"`
	TaskLoopRunForwarded    bool                        `json:"task_loop_run_forwarded"`
	StatusProjectionWritten bool                        `json:"status_projection_written"`
	AreaMatrixFilesModified bool                        `json:"area_matrix_files_modified"`
	GeneratedAt             string                      `json:"generated_at"`
}

type shimApplyCommandRequest struct {
	AllowedFiles               []string `json:"allowed_files"`
	ApprovalID                 string   `json:"approval_id"`
	ApprovalScope              string   `json:"approval_scope"`
	AuthorizationSnapshotHash  string   `json:"authorization_snapshot_hash"`
	ExpectedAuthorizationMode  string   `json:"expected_authorization_mode"`
	StatusProjectionPacketID   string   `json:"status_projection_packet_id"`
	StatusProjectionGateID     string   `json:"status_projection_gate_id"`
	ReadOnlySmokeEvidenceID    string   `json:"read_only_smoke_evidence_id"`
	DirtyWorktreeReviewID      string   `json:"dirty_worktree_review_id"`
	ProtectedPathFingerprintID string   `json:"protected_path_fingerprint_id"`
	RollbackPlanID             string   `json:"rollback_plan_id"`
	IdempotencyKey             string   `json:"idempotency_key"`
	AuditCorrelationID         string   `json:"audit_correlation_id"`
	FailureMode                string   `json:"failure_mode"`
	ExplicitApproval           bool     `json:"explicit_approval"`
	ApprovalActor              string   `json:"approval_actor"`
	ApprovalReason             string   `json:"approval_reason"`
}

type shimApplyCommandResponse struct {
	Project                 projectRecordResponse `json:"project"`
	Status                  string                `json:"status"`
	Mode                    string                `json:"mode"`
	Decision                string                `json:"decision"`
	Message                 string                `json:"message"`
	Gate                    shimApplyGateResponse `json:"gate"`
	Blockers                []string              `json:"blockers"`
	EventID                 int64                 `json:"event_id,omitempty"`
	AuditEventID            int64                 `json:"audit_event_id,omitempty"`
	IdempotencyKey          string                `json:"idempotency_key"`
	Created                 bool                  `json:"created"`
	RequiredPreflight       []string              `json:"required_preflight"`
	ForbiddenActions        []string              `json:"forbidden_actions"`
	SafetyFacts             map[string]bool       `json:"safety_facts"`
	ApplyOpen               bool                  `json:"apply_open"`
	AreaFlowCommandCreated  bool                  `json:"area_flow_command_created"`
	CommandRequestCreated   bool                  `json:"command_request_created"`
	ProjectWriteAttempted   bool                  `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                  `json:"execution_write_attempted"`
	EngineCallAttempted     bool                  `json:"engine_call_attempted"`
	TaskLoopRunForwarded    bool                  `json:"task_loop_run_forwarded"`
	StatusProjectionWritten bool                  `json:"status_projection_written"`
	AreaMatrixFilesModified bool                  `json:"area_matrix_files_modified"`
	GeneratedAt             string                `json:"generated_at"`
}

type shimApplyGateItemResponse struct {
	Key              string   `json:"key"`
	Category         string   `json:"category"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Expected         string   `json:"expected"`
	Actual           string   `json:"actual"`
	RequiredEvidence []string `json:"required_evidence"`
	BlockedBy        []string `json:"blocked_by"`
}

type shimReadinessItemResponse struct {
	Key      string         `json:"key"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type executionCutoverReadinessResponse struct {
	Project          projectRecordResponse                   `json:"project"`
	Status           string                                  `json:"status"`
	Mode             string                                  `json:"mode"`
	Items            []executionCutoverReadinessItemResponse `json:"items"`
	MigrationPath    []string                                `json:"migration_path"`
	CommandEvidence  map[string]int                          `json:"command_evidence"`
	Capabilities     []string                                `json:"capabilities"`
	ForbiddenActions []string                                `json:"forbidden_actions"`
	SafetyFacts      map[string]bool                         `json:"safety_facts"`
	NextSteps        []executionCutoverNextStepResponse      `json:"next_steps"`
	GeneratedAt      string                                  `json:"generated_at"`
}

type executionForwardingV1ReadinessResponse struct {
	Project          projectRecordResponse                   `json:"project"`
	Status           string                                  `json:"status"`
	Mode             string                                  `json:"mode"`
	Items            []executionCutoverReadinessItemResponse `json:"items"`
	AllowedTaskTypes []string                                `json:"allowed_task_types"`
	CommandEvidence  map[string]int                          `json:"command_evidence"`
	Capabilities     []string                                `json:"capabilities"`
	ForbiddenActions []string                                `json:"forbidden_actions"`
	SafetyFacts      map[string]bool                         `json:"safety_facts"`
	NextSteps        []executionCutoverNextStepResponse      `json:"next_steps"`
	GeneratedAt      string                                  `json:"generated_at"`
}

type executionForwardingV1ApplyPreviewResponse struct {
	Project              projectRecordResponse                           `json:"project"`
	Status               string                                          `json:"status"`
	Mode                 string                                          `json:"mode"`
	Readiness            executionForwardingV1ReadinessResponse          `json:"readiness"`
	Items                []executionForwardingV1ApplyPreviewItemResponse `json:"items"`
	AllowedTaskTypes     []string                                        `json:"allowed_task_types"`
	ForwardingTargets    []executionForwardingV1ForwardingTargetResponse `json:"forwarding_targets"`
	BlockedTargets       []executionForwardingV1BlockedTargetResponse    `json:"blocked_targets"`
	RequiredCapabilities []string                                        `json:"required_capabilities"`
	ApplyPacketFields    []string                                        `json:"apply_packet_fields"`
	FailClosedFields     []string                                        `json:"fail_closed_fields"`
	RequiredProofFacts   []string                                        `json:"required_proof_facts"`
	RequiredEvidence     []string                                        `json:"required_evidence"`
	ForbiddenActions     []string                                        `json:"forbidden_actions"`
	ApprovalRequired     bool                                            `json:"approval_required"`
	ApprovalStatus       string                                          `json:"approval_status"`
	ApplyOpen            bool                                            `json:"apply_open"`
	RollbackTarget       string                                          `json:"rollback_target"`
	SafetyFacts          map[string]bool                                 `json:"safety_facts"`
	GeneratedAt          string                                          `json:"generated_at"`
}

type executionForwardingV1ApplyPacketPreviewResponse struct {
	Project                                    projectRecordResponse                     `json:"project"`
	Status                                     string                                    `json:"status"`
	Mode                                       string                                    `json:"mode"`
	Decision                                   string                                    `json:"decision"`
	Message                                    string                                    `json:"message"`
	ApplyPreview                               executionForwardingV1ApplyPreviewResponse `json:"apply_preview"`
	Gate                                       executionForwardingV1ApplyGateResponse    `json:"gate"`
	Packet                                     executionForwardingV1ApplyPacketResponse  `json:"packet"`
	ApplyGateCommand                           []string                                  `json:"apply_gate_command"`
	FutureApplyCommand                         []string                                  `json:"future_apply_command"`
	RequiredHumanReview                        []string                                  `json:"required_human_review"`
	ForbiddenActions                           []string                                  `json:"forbidden_actions"`
	SafetyFacts                                map[string]bool                           `json:"safety_facts"`
	WouldCreateCommandRequestAfterApplyCommand bool                                      `json:"would_create_command_request_after_apply_command"`
	WouldCreateRunAfterApplyCommand            bool                                      `json:"would_create_run_after_apply_command"`
	WouldCreateRunTaskAfterApplyCommand        bool                                      `json:"would_create_run_task_after_apply_command"`
	WouldCreateAuditEventAfterApplyCommand     bool                                      `json:"would_create_audit_event_after_apply_command"`
	CommandRequestCreated                      bool                                      `json:"command_request_created"`
	AreaFlowRunCreated                         bool                                      `json:"area_flow_run_created"`
	TaskLoopRunForwarded                       bool                                      `json:"task_loop_run_forwarded"`
	ProjectWriteAttempted                      bool                                      `json:"project_write_attempted"`
	ExecutionWriteAttempted                    bool                                      `json:"execution_write_attempted"`
	EngineCallAttempted                        bool                                      `json:"engine_call_attempted"`
	GeneratedAt                                string                                    `json:"generated_at"`
}

type executionForwardingV1ApplyPacketResponse struct {
	CommandType                string   `json:"command_type"`
	ProjectKey                 string   `json:"project_key"`
	AllowedTaskTypes           []string `json:"allowed_task_types"`
	TargetCommandTypes         []string `json:"target_command_types"`
	ApprovalID                 string   `json:"approval_id"`
	ApprovalScope              string   `json:"approval_scope"`
	ReadinessSnapshotHash      string   `json:"readiness_snapshot_hash"`
	ExpectedShimLifecycleState string   `json:"expected_shim_lifecycle_state"`
	LegacyNonWriteProofID      string   `json:"legacy_non_write_proof_id"`
	RollbackPlanID             string   `json:"rollback_plan_id"`
	ProtectedPathFingerprintID string   `json:"protected_path_fingerprint_id"`
	IdempotencyKey             string   `json:"idempotency_key"`
	AuditCorrelationID         string   `json:"audit_correlation_id"`
	FailureMode                string   `json:"failure_mode"`
	ExplicitApproval           bool     `json:"explicit_approval"`
	ApprovalActor              string   `json:"approval_actor"`
	ApprovalReason             string   `json:"approval_reason"`
}

type executionForwardingV1ApplyGateResponse struct {
	Project                 projectRecordResponse                        `json:"project"`
	Status                  string                                       `json:"status"`
	Mode                    string                                       `json:"mode"`
	Decision                string                                       `json:"decision"`
	Message                 string                                       `json:"message"`
	Items                   []executionForwardingV1ApplyGateItemResponse `json:"items"`
	RequiredPacketFields    []string                                     `json:"required_packet_fields"`
	RequiredCapabilities    []string                                     `json:"required_capabilities"`
	AllowedTaskTypes        []string                                     `json:"allowed_task_types"`
	TargetCommandTypes      []string                                     `json:"target_command_types"`
	BlockedTaskTypes        []string                                     `json:"blocked_task_types"`
	ForbiddenActions        []string                                     `json:"forbidden_actions"`
	FailClosedFields        []string                                     `json:"fail_closed_fields"`
	RequiredProofFacts      []string                                     `json:"required_proof_facts"`
	SafetyFacts             map[string]bool                              `json:"safety_facts"`
	ApprovalRequired        bool                                         `json:"approval_required"`
	ApprovalStatus          string                                       `json:"approval_status"`
	ApplyCommandEligible    bool                                         `json:"apply_command_eligible"`
	ApplyOpen               bool                                         `json:"apply_open"`
	CommandRequestCreated   bool                                         `json:"command_request_created"`
	AreaFlowRunCreated      bool                                         `json:"area_flow_run_created"`
	TaskLoopRunForwarded    bool                                         `json:"task_loop_run_forwarded"`
	ProjectWriteAttempted   bool                                         `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                                         `json:"execution_write_attempted"`
	EngineCallAttempted     bool                                         `json:"engine_call_attempted"`
	GeneratedAt             string                                       `json:"generated_at"`
}

type executionForwardingV1ApplyGateItemResponse struct {
	Key              string   `json:"key"`
	Category         string   `json:"category"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Expected         string   `json:"expected"`
	Actual           string   `json:"actual"`
	RequiredEvidence []string `json:"required_evidence"`
	BlockedBy        []string `json:"blocked_by"`
}

type executionForwardingV1ForwardingTargetResponse struct {
	TaskType              string   `json:"task_type"`
	TargetCommandType     string   `json:"target_command_type"`
	TargetStatus          string   `json:"target_status"`
	RequiredCapabilities  []string `json:"required_capabilities"`
	RequiredPacketFields  []string `json:"required_packet_fields"`
	CreatesCommandRequest bool     `json:"creates_command_request"`
	CreatesRun            bool     `json:"creates_run"`
	CreatesRunTask        bool     `json:"creates_run_task"`
	CreatesRunAttempt     bool     `json:"creates_run_attempt"`
	CreatesArtifact       bool     `json:"creates_artifact"`
	CreatesAuditEvent     bool     `json:"creates_audit_event"`
	ProjectWriteAllowed   bool     `json:"project_write_allowed"`
	ExecutionWriteAllowed bool     `json:"execution_write_allowed"`
	LegacyFallbackAllowed bool     `json:"legacy_fallback_allowed"`
	FailureMode           string   `json:"failure_mode"`
}

type executionForwardingV1BlockedTargetResponse struct {
	TaskType        string          `json:"task_type"`
	ForbiddenAction string          `json:"forbidden_action"`
	Reason          string          `json:"reason"`
	FailureMode     string          `json:"failure_mode"`
	SafetyFacts     map[string]bool `json:"safety_facts"`
}

type executionForwardingV1ApplyPreviewItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	ApprovalStatus   string         `json:"approval_status,omitempty"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type executionForwardingV1CommandPreviewResponse struct {
	Project                                projectRecordResponse `json:"project"`
	Status                                 string                `json:"status"`
	Mode                                   string                `json:"mode"`
	Decision                               string                `json:"decision"`
	Message                                string                `json:"message"`
	TaskType                               string                `json:"task_type"`
	TargetCommandType                      string                `json:"target_command_type"`
	TargetStatus                           string                `json:"target_status"`
	FailureMode                            string                `json:"failure_mode"`
	AllowedTaskType                        bool                  `json:"allowed_task_type"`
	BlockedTaskType                        bool                  `json:"blocked_task_type"`
	ApplyOpen                              bool                  `json:"apply_open"`
	WouldCreateCommandRequestAfterApproval bool                  `json:"would_create_command_request_after_approval"`
	WouldCreateRunAfterApproval            bool                  `json:"would_create_run_after_approval"`
	WouldCreateRunTaskAfterApproval        bool                  `json:"would_create_run_task_after_approval"`
	WouldCreateRunAttemptAfterApproval     bool                  `json:"would_create_run_attempt_after_approval"`
	WouldCreateArtifactAfterApproval       bool                  `json:"would_create_artifact_after_approval"`
	WouldCreateAuditEventAfterApproval     bool                  `json:"would_create_audit_event_after_approval"`
	ProjectWriteAllowed                    bool                  `json:"project_write_allowed"`
	ExecutionWriteAllowed                  bool                  `json:"execution_write_allowed"`
	LegacyFallbackAllowed                  bool                  `json:"legacy_fallback_allowed"`
	RequiredPacketFields                   []string              `json:"required_packet_fields"`
	RequiredCapabilities                   []string              `json:"required_capabilities"`
	FailClosedFields                       []string              `json:"fail_closed_fields"`
	BlockedBy                              []string              `json:"blocked_by"`
	AllowedTaskTypes                       []string              `json:"allowed_task_types"`
	ForbiddenActions                       []string              `json:"forbidden_actions"`
	SafetyFacts                            map[string]bool       `json:"safety_facts"`
	GeneratedAt                            string                `json:"generated_at"`
}

type executionForwardingV1RollbackPreviewResponse struct {
	Project            projectRecordResponse                              `json:"project"`
	Status             string                                             `json:"status"`
	Mode               string                                             `json:"mode"`
	ApplyPreview       executionForwardingV1ApplyPreviewResponse          `json:"apply_preview"`
	Items              []executionForwardingV1RollbackPreviewItemResponse `json:"items"`
	RollbackTarget     string                                             `json:"rollback_target"`
	FailClosedSteps    []string                                           `json:"fail_closed_steps"`
	ReopenConditions   []string                                           `json:"reopen_conditions"`
	RequiredProofFacts []string                                           `json:"required_proof_facts"`
	RequiredEvidence   []string                                           `json:"required_evidence"`
	ForbiddenActions   []string                                           `json:"forbidden_actions"`
	RollbackApplyOpen  bool                                               `json:"rollback_apply_open"`
	SafetyFacts        map[string]bool                                    `json:"safety_facts"`
	GeneratedAt        string                                             `json:"generated_at"`
}

type executionForwardingV1RollbackPreviewItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type executionCutoverReadinessItemResponse struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type executionCutoverNextStepResponse struct {
	Key         string         `json:"key"`
	Owner       string         `json:"owner"`
	Action      string         `json:"action"`
	RiskLevel   string         `json:"risk_level"`
	BlockedBy   []string       `json:"blocked_by"`
	NextCommand string         `json:"next_command"`
	Metadata    map[string]any `json:"metadata"`
}

type workflowVersionResponse struct {
	ID              int64          `json:"id"`
	DisplayLabel    string         `json:"display_label"`
	VersionKind     string         `json:"version_kind"`
	LifecycleStatus string         `json:"lifecycle_status"`
	SourcePath      string         `json:"source_path,omitempty"`
	SourceHash      string         `json:"source_hash,omitempty"`
	ImportMode      string         `json:"import_mode"`
	Immutable       bool           `json:"immutable"`
	StatusSummary   map[string]any `json:"status_summary"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
	ImportedAt      string         `json:"imported_at,omitempty"`
}

type workflowItemResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	Stage             string         `json:"stage"`
	ItemType          string         `json:"item_type"`
	ExternalKey       string         `json:"external_key"`
	Title             string         `json:"title"`
	Status            string         `json:"status"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	ImportedAt        string         `json:"imported_at,omitempty"`
}

type workflowItemLinkResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	FromItemID        int64          `json:"from_item_id"`
	ToItemID          int64          `json:"to_item_id"`
	RelationType      string         `json:"relation_type"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
}

type workflowVersionListResponse struct {
	Project          projectRecordResponse     `json:"project"`
	WorkflowVersions []workflowVersionResponse `json:"workflow_versions"`
}

type createWorkflowVersionRequest struct {
	DisplayLabel   string `json:"display_label"`
	IdempotencyKey string `json:"idempotency_key"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
}

type workflowVersionCreateResponse struct {
	Project         projectRecordResponse   `json:"project"`
	WorkflowVersion workflowVersionResponse `json:"workflow_version"`
	InitialItem     workflowItemResponse    `json:"initial_item"`
	StageItems      []workflowItemResponse  `json:"stage_items"`
	Created         bool                    `json:"created"`
	IdempotencyKey  string                  `json:"idempotency_key"`
}

type workflowVersionStagesResponse struct {
	Project         projectRecordResponse      `json:"project"`
	WorkflowVersion workflowVersionResponse    `json:"workflow_version"`
	Items           []workflowItemResponse     `json:"items"`
	Links           []workflowItemLinkResponse `json:"links"`
}

type ensureStageSkeletonRequest struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

type ensureStageSkeletonResponse struct {
	Project         projectRecordResponse      `json:"project"`
	WorkflowVersion workflowVersionResponse    `json:"workflow_version"`
	Items           []workflowItemResponse     `json:"items"`
	Links           []workflowItemLinkResponse `json:"links"`
	Created         int                        `json:"created"`
}

type gateResultResponse struct {
	ID                  int64          `json:"id"`
	GateName            string         `json:"gate_name"`
	ScopeType           string         `json:"scope_type"`
	ScopeID             string         `json:"scope_id"`
	Status              string         `json:"status"`
	WorkflowVersionID   int64          `json:"workflow_version_id"`
	WorkflowItemID      int64          `json:"workflow_item_id,omitempty"`
	Inputs              map[string]any `json:"inputs"`
	SourceHashes        map[string]any `json:"source_hashes"`
	Failures            []string       `json:"failures"`
	Warnings            []string       `json:"warnings"`
	EvidenceArtifactIDs []int64        `json:"evidence_artifact_ids"`
	Metadata            map[string]any `json:"metadata"`
	CheckedAt           string         `json:"checked_at"`
}

type gateResultsResponse struct {
	Project         projectRecordResponse   `json:"project"`
	WorkflowVersion workflowVersionResponse `json:"workflow_version"`
	GateResults     []gateResultResponse    `json:"gate_results"`
}

type runGateRequest struct {
	GateName string `json:"gate_name"`
	Actor    string `json:"actor"`
	Reason   string `json:"reason"`
}

type transitionPreviewResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	FromStage         string         `json:"from_stage"`
	ToStage           string         `json:"to_stage"`
	Status            string         `json:"status"`
	RequiredGateName  string         `json:"required_gate_name"`
	GateResultID      int64          `json:"gate_result_id,omitempty"`
	Blockers          []string       `json:"blockers"`
	Warnings          []string       `json:"warnings"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
}

type transitionPreviewsResponse struct {
	Project            projectRecordResponse       `json:"project"`
	WorkflowVersion    workflowVersionResponse     `json:"workflow_version"`
	TransitionPreviews []transitionPreviewResponse `json:"transition_previews"`
}

type previewTransitionRequest struct {
	FromStage string `json:"from_stage"`
	ToStage   string `json:"to_stage"`
	Actor     string `json:"actor"`
	Reason    string `json:"reason"`
}

type approvalRecordResponse struct {
	ID                  int64          `json:"id"`
	WorkflowVersionID   int64          `json:"workflow_version_id"`
	TransitionPreviewID int64          `json:"transition_preview_id,omitempty"`
	ApprovalKind        string         `json:"approval_kind"`
	Decision            string         `json:"decision"`
	ScopeType           string         `json:"scope_type"`
	ScopeID             string         `json:"scope_id"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
	RiskLevel           string         `json:"risk_level"`
	Metadata            map[string]any `json:"metadata"`
	CreatedAt           string         `json:"created_at"`
}

type approvalRecordsResponse struct {
	Project         projectRecordResponse    `json:"project"`
	WorkflowVersion workflowVersionResponse  `json:"workflow_version"`
	ApprovalRecords []approvalRecordResponse `json:"approval_records"`
}

type createApprovalRequest struct {
	Decision            string         `json:"decision"`
	ApprovalKind        string         `json:"approval_kind"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
	RiskLevel           string         `json:"risk_level"`
	IdempotencyKey      string         `json:"idempotency_key"`
	TransitionPreviewID int64          `json:"transition_preview_id"`
	Metadata            map[string]any `json:"metadata"`
}

type runnerPreviewRequest struct {
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	RiskLevel      string `json:"risk_level"`
	RiskPolicy     string `json:"risk_policy"`
	IdempotencyKey string `json:"idempotency_key"`
}

type fixtureExecutionQueueRequest struct {
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type readOnlyVerifyQueueRequest struct {
	TargetPath     string `json:"target_path"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type approvedArtifactWriteQueueRequest struct {
	ArtifactLabel  string `json:"artifact_label"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type fixtureProjectWriteQueueRequest struct {
	TargetPath           string `json:"target_path"`
	Content              string `json:"content"`
	ExpectedBeforeSHA256 string `json:"expected_before_sha256"`
	ExpectedBeforeSize   int64  `json:"expected_before_size"`
	Actor                string `json:"actor"`
	Reason               string `json:"reason"`
	IdempotencyKey       string `json:"idempotency_key"`
}

type managedGeneratedWriteQueueRequest struct {
	TargetPath           string `json:"target_path"`
	Content              string `json:"content"`
	ExpectedBeforeSHA256 string `json:"expected_before_sha256"`
	ExpectedBeforeSize   int64  `json:"expected_before_size"`
	Actor                string `json:"actor"`
	Reason               string `json:"reason"`
	IdempotencyKey       string `json:"idempotency_key"`
}

type runControlRequest struct {
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type artifactArchivePreviewRequest struct {
	RetentionClass string `json:"retention_class"`
	Limit          int    `json:"limit"`
	Actor          string `json:"actor"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type runnerPreviewResponse struct {
	Project         projectRecordResponse   `json:"project"`
	WorkflowVersion workflowVersionResponse `json:"workflow_version"`
	Run             runResponse             `json:"run"`
	Tasks           []runTaskResponse       `json:"tasks"`
	Attempts        []runAttemptResponse    `json:"attempts"`
	Artifacts       []artifactResponse      `json:"artifacts"`
	Preflight       runnerPreflightResponse `json:"preflight"`
	Created         bool                    `json:"created"`
	IdempotencyKey  string                  `json:"idempotency_key"`
}

type fixtureExecutionQueueResponse struct {
	Project                 projectRecordResponse   `json:"project"`
	WorkflowVersion         workflowVersionResponse `json:"workflow_version"`
	Run                     runResponse             `json:"run"`
	Task                    runTaskResponse         `json:"task"`
	Created                 bool                    `json:"created"`
	IdempotencyKey          string                  `json:"idempotency_key"`
	EventID                 int64                   `json:"event_id,omitempty"`
	AuditEventID            int64                   `json:"audit_event_id,omitempty"`
	ProjectWriteAttempted   bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                    `json:"execution_write_attempted"`
	EngineCallAttempted     bool                    `json:"engine_call_attempted"`
	CommandsRun             bool                    `json:"commands_run"`
	SecretsResolved         bool                    `json:"secrets_resolved"`
	NetworkUsed             bool                    `json:"network_used"`
}

type readOnlyVerifyQueueResponse struct {
	Project                 projectRecordResponse   `json:"project"`
	WorkflowVersion         workflowVersionResponse `json:"workflow_version"`
	Run                     runResponse             `json:"run"`
	Task                    runTaskResponse         `json:"task"`
	TargetPath              string                  `json:"target_path"`
	Created                 bool                    `json:"created"`
	IdempotencyKey          string                  `json:"idempotency_key"`
	EventID                 int64                   `json:"event_id,omitempty"`
	AuditEventID            int64                   `json:"audit_event_id,omitempty"`
	ProjectReadAttempted    bool                    `json:"project_read_attempted"`
	ProjectReadAllowed      bool                    `json:"project_read_allowed"`
	ProjectWriteAttempted   bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                    `json:"execution_write_attempted"`
	EngineCallAttempted     bool                    `json:"engine_call_attempted"`
	CommandsRun             bool                    `json:"commands_run"`
	SecretsResolved         bool                    `json:"secrets_resolved"`
	NetworkUsed             bool                    `json:"network_used"`
}

type approvedArtifactWriteQueueResponse struct {
	Project                       projectRecordResponse   `json:"project"`
	WorkflowVersion               workflowVersionResponse `json:"workflow_version"`
	Run                           runResponse             `json:"run"`
	Task                          runTaskResponse         `json:"task"`
	ArtifactLabel                 string                  `json:"artifact_label"`
	Created                       bool                    `json:"created"`
	IdempotencyKey                string                  `json:"idempotency_key"`
	EventID                       int64                   `json:"event_id,omitempty"`
	AuditEventID                  int64                   `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                    `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                    `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                    `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                    `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                    `json:"engine_call_attempted"`
	CommandsRun                   bool                    `json:"commands_run"`
	SecretsResolved               bool                    `json:"secrets_resolved"`
	NetworkUsed                   bool                    `json:"network_used"`
}

type fixtureProjectWriteQueueResponse struct {
	Project                       projectRecordResponse   `json:"project"`
	WorkflowVersion               workflowVersionResponse `json:"workflow_version"`
	Run                           runResponse             `json:"run"`
	Task                          runTaskResponse         `json:"task"`
	WriteSetArtifact              artifactResponse        `json:"write_set_artifact"`
	TargetPath                    string                  `json:"target_path"`
	ExpectedBeforeSHA256          string                  `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64                   `json:"expected_before_size"`
	AfterSHA256                   string                  `json:"after_sha256"`
	AfterSize                     int64                   `json:"after_size"`
	Created                       bool                    `json:"created"`
	IdempotencyKey                string                  `json:"idempotency_key"`
	EventID                       int64                   `json:"event_id,omitempty"`
	AuditEventID                  int64                   `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                    `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                    `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                    `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                    `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                    `json:"engine_call_attempted"`
	CommandsRun                   bool                    `json:"commands_run"`
	SecretsResolved               bool                    `json:"secrets_resolved"`
	NetworkUsed                   bool                    `json:"network_used"`
}

type managedGeneratedWriteQueueResponse struct {
	Project                       projectRecordResponse   `json:"project"`
	WorkflowVersion               workflowVersionResponse `json:"workflow_version"`
	Run                           runResponse             `json:"run"`
	Task                          runTaskResponse         `json:"task"`
	WriteSetArtifact              artifactResponse        `json:"write_set_artifact"`
	TargetPath                    string                  `json:"target_path"`
	ExpectedBeforeSHA256          string                  `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64                   `json:"expected_before_size"`
	AfterSHA256                   string                  `json:"after_sha256"`
	AfterSize                     int64                   `json:"after_size"`
	Created                       bool                    `json:"created"`
	IdempotencyKey                string                  `json:"idempotency_key"`
	EventID                       int64                   `json:"event_id,omitempty"`
	AuditEventID                  int64                   `json:"audit_event_id,omitempty"`
	GeneratedOnly                 bool                    `json:"generated_only"`
	GeneratedOnlyApplyOpen        bool                    `json:"generated_only_apply_open"`
	ProjectReadAttempted          bool                    `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                    `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                    `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                    `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                    `json:"engine_call_attempted"`
	CommandsRun                   bool                    `json:"commands_run"`
	SecretsResolved               bool                    `json:"secrets_resolved"`
	NetworkUsed                   bool                    `json:"network_used"`
}

type runDetailResponse struct {
	Run       runResponse          `json:"run"`
	Tasks     []runTaskResponse    `json:"tasks"`
	Attempts  []runAttemptResponse `json:"attempts"`
	Artifacts []artifactResponse   `json:"artifacts"`
}

type runControlResponse struct {
	Project                  projectRecordResponse `json:"project"`
	Run                      runResponse           `json:"run"`
	PreviousStatus           string                `json:"previous_status"`
	Status                   string                `json:"status"`
	Decision                 string                `json:"decision"`
	Message                  string                `json:"message"`
	Blockers                 []string              `json:"blockers"`
	EventID                  int64                 `json:"event_id,omitempty"`
	AuditEventID             int64                 `json:"audit_event_id,omitempty"`
	IdempotencyKey           string                `json:"idempotency_key"`
	Created                  bool                  `json:"created"`
	ProjectWriteAttempted    bool                  `json:"project_write_attempted"`
	ExecutionWriteAttempted  bool                  `json:"execution_write_attempted"`
	AreaMatrixWriteAttempted bool                  `json:"area_matrix_write_attempted"`
	EngineCallAttempted      bool                  `json:"engine_call_attempted"`
}

type executionApprovalGateResponse struct {
	Project                 projectRecordResponse          `json:"project"`
	WorkflowVersion         workflowVersionResponse        `json:"workflow_version"`
	Run                     runResponse                    `json:"run"`
	Status                  string                         `json:"status"`
	Mode                    string                         `json:"mode"`
	Items                   []projectReadinessItemResponse `json:"items"`
	Blockers                []string                       `json:"blockers"`
	Warnings                []string                       `json:"warnings"`
	RequiredCapabilities    []string                       `json:"required_capabilities"`
	ApprovalFound           bool                           `json:"approval_found"`
	Approval                approvalRecordResponse         `json:"approval,omitempty"`
	ApprovalGateFound       bool                           `json:"approval_gate_found"`
	ApprovalGate            gateResultResponse             `json:"approval_gate,omitempty"`
	LiveMappingGateFound    bool                           `json:"live_mapping_gate_found"`
	LiveMappingGate         gateResultResponse             `json:"live_mapping_gate,omitempty"`
	EnginePreview           codexCLIAdapterPreviewResponse `json:"engine_preview"`
	Workers                 []workerResponse               `json:"workers"`
	ForbiddenActions        []string                       `json:"forbidden_actions"`
	ProjectWriteAttempted   bool                           `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                           `json:"execution_write_attempted"`
	EngineCallAttempted     bool                           `json:"engine_call_attempted"`
	CommandsRun             bool                           `json:"commands_run"`
	SecretsResolved         bool                           `json:"secrets_resolved"`
	NetworkUsed             bool                           `json:"network_used"`
	TaskClaimed             bool                           `json:"task_claimed"`
	WorkerStarted           bool                           `json:"worker_started"`
	AttemptCreated          bool                           `json:"attempt_created"`
	ArtifactCreated         bool                           `json:"artifact_created"`
	GeneratedAt             string                         `json:"generated_at"`
}

type executionPlanPreviewResponse struct {
	Project                       projectRecordResponse         `json:"project"`
	WorkflowVersion               workflowVersionResponse       `json:"workflow_version"`
	Run                           runResponse                   `json:"run"`
	Gate                          executionApprovalGateResponse `json:"gate"`
	Status                        string                        `json:"status"`
	Mode                          string                        `json:"mode"`
	Steps                         []executionPlanStepResponse   `json:"steps"`
	Blockers                      []string                      `json:"blockers"`
	ForbiddenActions              []string                      `json:"forbidden_actions"`
	ProjectReadAttempted          bool                          `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                          `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                          `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                          `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                          `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                          `json:"engine_call_attempted"`
	CommandsRun                   bool                          `json:"commands_run"`
	SecretsResolved               bool                          `json:"secrets_resolved"`
	NetworkUsed                   bool                          `json:"network_used"`
	TaskClaimed                   bool                          `json:"task_claimed"`
	WorkerStarted                 bool                          `json:"worker_started"`
	AttemptCreated                bool                          `json:"attempt_created"`
	ArtifactCreated               bool                          `json:"artifact_created"`
	GeneratedAt                   string                        `json:"generated_at"`
}

type executionPlanStepResponse struct {
	Key                  string         `json:"key"`
	AttemptKind          string         `json:"attempt_kind"`
	Status               string         `json:"status"`
	Message              string         `json:"message"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	Prerequisites        []string       `json:"prerequisites"`
	Blockers             []string       `json:"blockers"`
	ReadsProject         bool           `json:"reads_project"`
	WritesProject        bool           `json:"writes_project"`
	WritesAreaFlow       bool           `json:"writes_area_flow"`
	UsesEngine           bool           `json:"uses_engine"`
	RunsCommands         bool           `json:"runs_commands"`
	UsesSecrets          bool           `json:"uses_secrets"`
	UsesNetwork          bool           `json:"uses_network"`
	CreatesAttempt       bool           `json:"creates_attempt"`
	CreatesArtifact      bool           `json:"creates_artifact"`
	Metadata             map[string]any `json:"metadata"`
}

type projectWriteDesignGateResponse struct {
	Project                       projectRecordResponse          `json:"project"`
	WorkflowVersion               workflowVersionResponse        `json:"workflow_version"`
	Run                           runResponse                    `json:"run"`
	Gate                          executionApprovalGateResponse  `json:"gate"`
	Status                        string                         `json:"status"`
	Mode                          string                         `json:"mode"`
	Items                         []projectReadinessItemResponse `json:"items"`
	RequiredCapabilities          []string                       `json:"required_capabilities"`
	WriteSetFields                []string                       `json:"write_set_fields"`
	UnsupportedOperations         []string                       `json:"unsupported_operations"`
	ApplySequence                 []string                       `json:"apply_sequence"`
	Blockers                      []string                       `json:"blockers"`
	ForbiddenActions              []string                       `json:"forbidden_actions"`
	ProjectWriteApplyOpen         bool                           `json:"project_write_apply_open"`
	ProjectReadAttempted          bool                           `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                           `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                           `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                           `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                           `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                           `json:"engine_call_attempted"`
	CommandsRun                   bool                           `json:"commands_run"`
	SecretsResolved               bool                           `json:"secrets_resolved"`
	NetworkUsed                   bool                           `json:"network_used"`
	TaskClaimed                   bool                           `json:"task_claimed"`
	WorkerStarted                 bool                           `json:"worker_started"`
	AttemptCreated                bool                           `json:"attempt_created"`
	ArtifactCreated               bool                           `json:"artifact_created"`
	GeneratedAt                   string                         `json:"generated_at"`
}

type managedGeneratedWriteGateResponse struct {
	Project                       projectRecordResponse          `json:"project"`
	WorkflowVersion               workflowVersionResponse        `json:"workflow_version"`
	Run                           runResponse                    `json:"run"`
	Gate                          executionApprovalGateResponse  `json:"gate"`
	Status                        string                         `json:"status"`
	Mode                          string                         `json:"mode"`
	Items                         []projectReadinessItemResponse `json:"items"`
	RequiredCapabilities          []string                       `json:"required_capabilities"`
	AllowedGeneratedPrefixes      []string                       `json:"allowed_generated_prefixes"`
	RequiredWriteSetFields        []string                       `json:"required_write_set_fields"`
	UnsupportedOperations         []string                       `json:"unsupported_operations"`
	ApplySequence                 []string                       `json:"apply_sequence"`
	Blockers                      []string                       `json:"blockers"`
	ForbiddenActions              []string                       `json:"forbidden_actions"`
	GeneratedOnlyWriteReady       bool                           `json:"generated_only_write_ready"`
	GeneratedOnlyApplyOpen        bool                           `json:"generated_only_apply_open"`
	ProjectReadAttempted          bool                           `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                           `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                           `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                           `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                           `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                           `json:"engine_call_attempted"`
	CommandsRun                   bool                           `json:"commands_run"`
	SecretsResolved               bool                           `json:"secrets_resolved"`
	NetworkUsed                   bool                           `json:"network_used"`
	TaskClaimed                   bool                           `json:"task_claimed"`
	WorkerStarted                 bool                           `json:"worker_started"`
	LeaseCreated                  bool                           `json:"lease_created"`
	AttemptCreated                bool                           `json:"attempt_created"`
	ArtifactCreated               bool                           `json:"artifact_created"`
	GeneratedAt                   string                         `json:"generated_at"`
}

type workflowVersionRunsResponse struct {
	Project         projectRecordResponse   `json:"project"`
	WorkflowVersion workflowVersionResponse `json:"workflow_version"`
	Runs            []runResponse           `json:"runs"`
}

type runEventsResponse struct {
	RunID  int64           `json:"run_id"`
	Events []eventResponse `json:"events"`
}

type projectArtifactsResponse struct {
	Project   projectRecordResponse `json:"project"`
	Artifacts []artifactResponse    `json:"artifacts"`
}

type workflowVersionArtifactsResponse struct {
	Project         projectRecordResponse   `json:"project"`
	WorkflowVersion workflowVersionResponse `json:"workflow_version"`
	Artifacts       []artifactResponse      `json:"artifacts"`
}

type projectResidualsResponse struct {
	Project   projectRecordResponse `json:"project"`
	Residuals []residualResponse    `json:"residuals"`
}

type statusProjectionResponse struct {
	ID                int64          `json:"id"`
	ProjectID         int64          `json:"project_id"`
	WorkflowVersionID int64          `json:"workflow_version_id,omitempty"`
	TargetKind        string         `json:"target_kind"`
	TargetURI         string         `json:"target_uri"`
	SummaryState      string         `json:"summary_state"`
	Payload           map[string]any `json:"payload"`
	SourceEventID     int64          `json:"source_event_id,omitempty"`
	SourceHash        string         `json:"source_hash,omitempty"`
	WriteState        string         `json:"write_state"`
	GeneratedAt       string         `json:"generated_at"`
	WrittenAt         string         `json:"written_at,omitempty"`
	Metadata          map[string]any `json:"metadata"`
}

type statusProjectionsResponse struct {
	Project     projectRecordResponse      `json:"project"`
	Projections []statusProjectionResponse `json:"projections"`
}

type workflowVersionResidualsResponse struct {
	Project         projectRecordResponse   `json:"project"`
	WorkflowVersion workflowVersionResponse `json:"workflow_version"`
	Residuals       []residualResponse      `json:"residuals"`
}

type residualResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id,omitempty"`
	ResidualKey       string         `json:"residual_key"`
	Status            string         `json:"status"`
	Type              string         `json:"type"`
	Title             string         `json:"title"`
	SourcePath        string         `json:"source_path"`
	CurrentImpact     string         `json:"current_impact"`
	ExecutableTask    bool           `json:"executable_task"`
	PromotionRequired bool           `json:"promotion_required"`
	CloseCondition    string         `json:"close_condition"`
	Metadata          map[string]any `json:"metadata"`
	Immutable         bool           `json:"immutable"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	ImportedAt        string         `json:"imported_at,omitempty"`
}

type registerWorkerRequest struct {
	WorkerKey                string         `json:"worker_key"`
	WorkerType               string         `json:"worker_type"`
	Hostname                 string         `json:"hostname"`
	PID                      int            `json:"pid"`
	Capabilities             []string       `json:"capabilities"`
	Metadata                 map[string]any `json:"metadata"`
	HeartbeatIntervalSeconds int            `json:"heartbeat_interval_seconds"`
	LeaseTimeoutSeconds      int            `json:"lease_timeout_seconds"`
	Actor                    string         `json:"actor"`
	Reason                   string         `json:"reason"`
	IdempotencyKey           string         `json:"idempotency_key"`
}

type workerHeartbeatRequest struct {
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
	Actor          string         `json:"actor"`
	Reason         string         `json:"reason"`
	IdempotencyKey string         `json:"idempotency_key"`
}

type leaseAcquireRequest struct {
	RunTaskID            int64          `json:"run_task_id"`
	LeaseKind            string         `json:"lease_kind"`
	AllowedCapabilities  []string       `json:"allowed_capabilities"`
	Scope                map[string]any `json:"scope"`
	Metadata             map[string]any `json:"metadata"`
	LeaseTimeoutSeconds  int            `json:"lease_timeout_seconds"`
	RecoverExpiredBefore bool           `json:"recover_expired_before"`
	Actor                string         `json:"actor"`
	Reason               string         `json:"reason"`
	IdempotencyKey       string         `json:"idempotency_key"`
}

type leaseReleaseRequest struct {
	LeaseID        int64          `json:"lease_id"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
	Actor          string         `json:"actor"`
	Reason         string         `json:"reason"`
	IdempotencyKey string         `json:"idempotency_key"`
}

type leaseRecoverRequest struct {
	Limit          int            `json:"limit"`
	Metadata       map[string]any `json:"metadata"`
	Actor          string         `json:"actor"`
	Reason         string         `json:"reason"`
	IdempotencyKey string         `json:"idempotency_key"`
}

type workerRunOnceRequest struct {
	RunID               int64          `json:"run_id"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	LeaseTimeoutSeconds int            `json:"lease_timeout_seconds"`
	Metadata            map[string]any `json:"metadata"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
}

type fixtureExecutionRequest struct {
	RunID               int64          `json:"run_id"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	LeaseTimeoutSeconds int            `json:"lease_timeout_seconds"`
	Metadata            map[string]any `json:"metadata"`
	IdempotencyKey      string         `json:"idempotency_key"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
}

type readOnlyVerifyRequest struct {
	RunID               int64          `json:"run_id"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	LeaseTimeoutSeconds int            `json:"lease_timeout_seconds"`
	Metadata            map[string]any `json:"metadata"`
	IdempotencyKey      string         `json:"idempotency_key"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
}

type approvedArtifactWriteRequest struct {
	RunID               int64          `json:"run_id"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	LeaseTimeoutSeconds int            `json:"lease_timeout_seconds"`
	Metadata            map[string]any `json:"metadata"`
	IdempotencyKey      string         `json:"idempotency_key"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
}

type fixtureProjectWriteRequest struct {
	RunID               int64          `json:"run_id"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	LeaseTimeoutSeconds int            `json:"lease_timeout_seconds"`
	Metadata            map[string]any `json:"metadata"`
	IdempotencyKey      string         `json:"idempotency_key"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
}

type managedGeneratedWriteRequest struct {
	RunID               int64          `json:"run_id"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	LeaseTimeoutSeconds int            `json:"lease_timeout_seconds"`
	Metadata            map[string]any `json:"metadata"`
	IdempotencyKey      string         `json:"idempotency_key"`
	Actor               string         `json:"actor"`
	Reason              string         `json:"reason"`
}

type workerRunOnceResponse struct {
	Project  projectRecordResponse `json:"project"`
	Worker   workerResponse        `json:"worker"`
	Lease    *leaseResponse        `json:"lease,omitempty"`
	Task     *runTaskResponse      `json:"task,omitempty"`
	Attempt  *runAttemptResponse   `json:"attempt,omitempty"`
	Artifact *artifactResponse     `json:"artifact,omitempty"`
	Claimed  bool                  `json:"claimed"`
}

type fixtureExecutionResponse struct {
	Project                       projectRecordResponse         `json:"project"`
	WorkflowVersion               workflowVersionResponse       `json:"workflow_version"`
	Run                           runResponse                   `json:"run"`
	Worker                        workerResponse                `json:"worker"`
	Lease                         leaseResponse                 `json:"lease,omitempty"`
	Task                          runTaskResponse               `json:"task,omitempty"`
	Attempt                       runAttemptResponse            `json:"attempt,omitempty"`
	Artifact                      artifactResponse              `json:"artifact,omitempty"`
	Gate                          executionApprovalGateResponse `json:"gate"`
	Status                        string                        `json:"status"`
	Decision                      string                        `json:"decision"`
	Message                       string                        `json:"message"`
	Blockers                      []string                      `json:"blockers"`
	Created                       bool                          `json:"created"`
	IdempotencyKey                string                        `json:"idempotency_key"`
	EventID                       int64                         `json:"event_id,omitempty"`
	AuditEventID                  int64                         `json:"audit_event_id,omitempty"`
	ProjectWriteAttempted         bool                          `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                          `json:"execution_write_attempted"`
	AreaFlowExecutionStateWritten bool                          `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                          `json:"engine_call_attempted"`
	CommandsRun                   bool                          `json:"commands_run"`
	SecretsResolved               bool                          `json:"secrets_resolved"`
	NetworkUsed                   bool                          `json:"network_used"`
	TaskClaimed                   bool                          `json:"task_claimed"`
	WorkerStarted                 bool                          `json:"worker_started"`
	LeaseCreated                  bool                          `json:"lease_created"`
	AttemptCreated                bool                          `json:"attempt_created"`
	ArtifactCreated               bool                          `json:"artifact_created"`
}

type readOnlyVerifyResponse struct {
	Project                       projectRecordResponse         `json:"project"`
	WorkflowVersion               workflowVersionResponse       `json:"workflow_version"`
	Run                           runResponse                   `json:"run"`
	Worker                        workerResponse                `json:"worker"`
	Lease                         leaseResponse                 `json:"lease,omitempty"`
	Task                          runTaskResponse               `json:"task,omitempty"`
	Attempt                       runAttemptResponse            `json:"attempt,omitempty"`
	Artifact                      artifactResponse              `json:"artifact,omitempty"`
	Gate                          executionApprovalGateResponse `json:"gate"`
	TargetPath                    string                        `json:"target_path"`
	TargetSHA256                  string                        `json:"target_sha256"`
	TargetSizeBytes               int64                         `json:"target_size_bytes"`
	Status                        string                        `json:"status"`
	Decision                      string                        `json:"decision"`
	Message                       string                        `json:"message"`
	Blockers                      []string                      `json:"blockers"`
	Created                       bool                          `json:"created"`
	IdempotencyKey                string                        `json:"idempotency_key"`
	EventID                       int64                         `json:"event_id,omitempty"`
	AuditEventID                  int64                         `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                          `json:"project_read_attempted"`
	ProjectReadAllowed            bool                          `json:"project_read_allowed"`
	ProjectWriteAttempted         bool                          `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                          `json:"execution_write_attempted"`
	AreaFlowExecutionStateWritten bool                          `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                          `json:"engine_call_attempted"`
	CommandsRun                   bool                          `json:"commands_run"`
	SecretsResolved               bool                          `json:"secrets_resolved"`
	NetworkUsed                   bool                          `json:"network_used"`
	TaskClaimed                   bool                          `json:"task_claimed"`
	WorkerStarted                 bool                          `json:"worker_started"`
	LeaseCreated                  bool                          `json:"lease_created"`
	AttemptCreated                bool                          `json:"attempt_created"`
	ArtifactCreated               bool                          `json:"artifact_created"`
	VerificationPassed            bool                          `json:"verification_passed"`
}

type approvedArtifactWriteResponse struct {
	Project                       projectRecordResponse         `json:"project"`
	WorkflowVersion               workflowVersionResponse       `json:"workflow_version"`
	Run                           runResponse                   `json:"run"`
	Worker                        workerResponse                `json:"worker"`
	Lease                         leaseResponse                 `json:"lease,omitempty"`
	Task                          runTaskResponse               `json:"task,omitempty"`
	Attempt                       runAttemptResponse            `json:"attempt,omitempty"`
	Artifact                      artifactResponse              `json:"artifact,omitempty"`
	Gate                          executionApprovalGateResponse `json:"gate"`
	ArtifactLabel                 string                        `json:"artifact_label"`
	Status                        string                        `json:"status"`
	Decision                      string                        `json:"decision"`
	Message                       string                        `json:"message"`
	Blockers                      []string                      `json:"blockers"`
	Created                       bool                          `json:"created"`
	IdempotencyKey                string                        `json:"idempotency_key"`
	EventID                       int64                         `json:"event_id,omitempty"`
	AuditEventID                  int64                         `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                          `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                          `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                          `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                          `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                          `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                          `json:"engine_call_attempted"`
	CommandsRun                   bool                          `json:"commands_run"`
	SecretsResolved               bool                          `json:"secrets_resolved"`
	NetworkUsed                   bool                          `json:"network_used"`
	TaskClaimed                   bool                          `json:"task_claimed"`
	WorkerStarted                 bool                          `json:"worker_started"`
	LeaseCreated                  bool                          `json:"lease_created"`
	AttemptCreated                bool                          `json:"attempt_created"`
	ArtifactCreated               bool                          `json:"artifact_created"`
	ArtifactWritePassed           bool                          `json:"artifact_write_passed"`
}

type fixtureProjectWriteResponse struct {
	Project                       projectRecordResponse         `json:"project"`
	WorkflowVersion               workflowVersionResponse       `json:"workflow_version"`
	Run                           runResponse                   `json:"run"`
	Worker                        workerResponse                `json:"worker"`
	Lease                         leaseResponse                 `json:"lease,omitempty"`
	Task                          runTaskResponse               `json:"task,omitempty"`
	CopyAttempt                   runAttemptResponse            `json:"copy_attempt,omitempty"`
	VerifyAttempt                 runAttemptResponse            `json:"verify_attempt,omitempty"`
	RollbackAttempt               runAttemptResponse            `json:"rollback_attempt,omitempty"`
	WriteSetArtifact              artifactResponse              `json:"write_set_artifact,omitempty"`
	PreimageArtifact              artifactResponse              `json:"preimage_artifact,omitempty"`
	Artifact                      artifactResponse              `json:"artifact,omitempty"`
	Gate                          executionApprovalGateResponse `json:"gate"`
	TargetPath                    string                        `json:"target_path"`
	ExpectedBeforeSHA256          string                        `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64                         `json:"expected_before_size"`
	AfterSHA256                   string                        `json:"after_sha256"`
	AfterSize                     int64                         `json:"after_size"`
	RestoredSHA256                string                        `json:"restored_sha256"`
	RestoredSize                  int64                         `json:"restored_size"`
	Status                        string                        `json:"status"`
	Decision                      string                        `json:"decision"`
	Message                       string                        `json:"message"`
	Blockers                      []string                      `json:"blockers"`
	Created                       bool                          `json:"created"`
	IdempotencyKey                string                        `json:"idempotency_key"`
	EventID                       int64                         `json:"event_id,omitempty"`
	AuditEventID                  int64                         `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                          `json:"project_read_attempted"`
	ProjectReadAllowed            bool                          `json:"project_read_allowed"`
	ProjectWriteAttempted         bool                          `json:"project_write_attempted"`
	ProjectWriteAllowed           bool                          `json:"project_write_allowed"`
	ExecutionWriteAttempted       bool                          `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                          `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                          `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                          `json:"engine_call_attempted"`
	CommandsRun                   bool                          `json:"commands_run"`
	SecretsResolved               bool                          `json:"secrets_resolved"`
	NetworkUsed                   bool                          `json:"network_used"`
	TaskClaimed                   bool                          `json:"task_claimed"`
	WorkerStarted                 bool                          `json:"worker_started"`
	LeaseCreated                  bool                          `json:"lease_created"`
	AttemptCreated                bool                          `json:"attempt_created"`
	ArtifactCreated               bool                          `json:"artifact_created"`
	WriteSetPassed                bool                          `json:"write_set_passed"`
	VerificationPassed            bool                          `json:"verification_passed"`
	RollbackAttempted             bool                          `json:"rollback_attempted"`
	RollbackVerified              bool                          `json:"rollback_verified"`
}

type managedGeneratedWriteResponse struct {
	Project                       projectRecordResponse         `json:"project"`
	WorkflowVersion               workflowVersionResponse       `json:"workflow_version"`
	Run                           runResponse                   `json:"run"`
	Worker                        workerResponse                `json:"worker"`
	Lease                         leaseResponse                 `json:"lease,omitempty"`
	Task                          runTaskResponse               `json:"task,omitempty"`
	CopyAttempt                   runAttemptResponse            `json:"copy_attempt,omitempty"`
	VerifyAttempt                 runAttemptResponse            `json:"verify_attempt,omitempty"`
	RollbackAttempt               runAttemptResponse            `json:"rollback_attempt,omitempty"`
	WriteSetArtifact              artifactResponse              `json:"write_set_artifact,omitempty"`
	PreimageArtifact              artifactResponse              `json:"preimage_artifact,omitempty"`
	Artifact                      artifactResponse              `json:"artifact,omitempty"`
	Gate                          executionApprovalGateResponse `json:"gate"`
	TargetPath                    string                        `json:"target_path"`
	ExpectedBeforeSHA256          string                        `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64                         `json:"expected_before_size"`
	AfterSHA256                   string                        `json:"after_sha256"`
	AfterSize                     int64                         `json:"after_size"`
	RestoredSHA256                string                        `json:"restored_sha256"`
	RestoredSize                  int64                         `json:"restored_size"`
	Status                        string                        `json:"status"`
	Decision                      string                        `json:"decision"`
	Message                       string                        `json:"message"`
	Blockers                      []string                      `json:"blockers"`
	Created                       bool                          `json:"created"`
	IdempotencyKey                string                        `json:"idempotency_key"`
	EventID                       int64                         `json:"event_id,omitempty"`
	AuditEventID                  int64                         `json:"audit_event_id,omitempty"`
	GeneratedOnly                 bool                          `json:"generated_only"`
	GeneratedOnlyApplyOpen        bool                          `json:"generated_only_apply_open"`
	ProjectReadAttempted          bool                          `json:"project_read_attempted"`
	ProjectReadAllowed            bool                          `json:"project_read_allowed"`
	ProjectWriteAttempted         bool                          `json:"project_write_attempted"`
	ProjectWriteAllowed           bool                          `json:"project_write_allowed"`
	ExecutionWriteAttempted       bool                          `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                          `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                          `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                          `json:"engine_call_attempted"`
	CommandsRun                   bool                          `json:"commands_run"`
	SecretsResolved               bool                          `json:"secrets_resolved"`
	NetworkUsed                   bool                          `json:"network_used"`
	TaskClaimed                   bool                          `json:"task_claimed"`
	WorkerStarted                 bool                          `json:"worker_started"`
	LeaseCreated                  bool                          `json:"lease_created"`
	AttemptCreated                bool                          `json:"attempt_created"`
	ArtifactCreated               bool                          `json:"artifact_created"`
	WriteSetPassed                bool                          `json:"write_set_passed"`
	VerificationPassed            bool                          `json:"verification_passed"`
	RollbackAttempted             bool                          `json:"rollback_attempted"`
	RollbackVerified              bool                          `json:"rollback_verified"`
}

type leaseResponse struct {
	ID                  int64          `json:"id"`
	ProjectID           int64          `json:"project_id"`
	RunID               int64          `json:"run_id,omitempty"`
	RunTaskID           int64          `json:"run_task_id,omitempty"`
	WorkflowItemID      int64          `json:"workflow_item_id,omitempty"`
	WorkerID            int64          `json:"worker_id,omitempty"`
	LeaseKind           string         `json:"lease_kind"`
	Status              string         `json:"status"`
	AcquiredAt          string         `json:"acquired_at"`
	ExpiresAt           string         `json:"expires_at"`
	HeartbeatAt         string         `json:"heartbeat_at,omitempty"`
	ReleasedAt          string         `json:"released_at,omitempty"`
	AllowedCapabilities []string       `json:"allowed_capabilities"`
	Scope               map[string]any `json:"scope"`
	Metadata            map[string]any `json:"metadata"`
}

type leaseRecoverResponse struct {
	Project projectRecordResponse `json:"project"`
	Leases  []leaseResponse       `json:"leases"`
}

type workerResponse struct {
	ID                       int64          `json:"id"`
	ProjectID                int64          `json:"project_id"`
	ActorID                  int64          `json:"actor_id,omitempty"`
	WorkerKey                string         `json:"worker_key"`
	WorkerType               string         `json:"worker_type"`
	Status                   string         `json:"status"`
	Hostname                 string         `json:"hostname"`
	PID                      int            `json:"pid,omitempty"`
	Capabilities             []string       `json:"capabilities"`
	Metadata                 map[string]any `json:"metadata"`
	RegisteredAt             string         `json:"registered_at"`
	LastHeartbeatAt          string         `json:"last_heartbeat_at,omitempty"`
	HeartbeatIntervalSeconds int            `json:"heartbeat_interval_seconds"`
	LeaseTimeoutSeconds      int            `json:"lease_timeout_seconds"`
	UpdatedAt                string         `json:"updated_at"`
}

type workerListResponse struct {
	Project projectRecordResponse `json:"project"`
	Workers []workerResponse      `json:"workers"`
}

type workerPoolSummaryResponse struct {
	Projects           []workerPoolProjectSummaryResponse `json:"projects"`
	TotalProjects      int64                              `json:"total_projects"`
	TotalWorkers       int64                              `json:"total_workers"`
	TotalOnlineWorkers int64                              `json:"total_online_workers"`
	TotalActiveLeases  int64                              `json:"total_active_leases"`
	TotalQueuedTasks   int64                              `json:"total_queued_tasks"`
	TotalNeedsRecovery int64                              `json:"total_needs_recovery"`
	GeneratedAt        string                             `json:"generated_at"`
}

type workerPoolProjectSummaryResponse struct {
	Project             projectRecordResponse     `json:"project"`
	Workers             int64                     `json:"workers"`
	OnlineWorkers       int64                     `json:"online_workers"`
	OfflineWorkers      int64                     `json:"offline_workers"`
	ActiveLeases        int64                     `json:"active_leases"`
	NeedsRecoveryLeases int64                     `json:"needs_recovery_leases"`
	QueuedTasks         int64                     `json:"queued_tasks"`
	NeedsRecoveryTasks  int64                     `json:"needs_recovery_tasks"`
	Capabilities        []string                  `json:"capabilities"`
	WorkerTypes         []string                  `json:"worker_types"`
	Scheduling          schedulingPolicyResponse  `json:"scheduling"`
	Role                roleReadinessResponse     `json:"role"`
	Engine              engineReadinessResponse   `json:"engine"`
	Resources           resourceReadinessResponse `json:"resources"`
	LastWorkerHeartbeat string                    `json:"last_worker_heartbeat,omitempty"`
}

type schedulingPolicyResponse struct {
	Priority             int      `json:"priority"`
	MaxParallelTasks     int      `json:"max_parallel_tasks"`
	AgentRole            string   `json:"agent_role"`
	RequiredCapabilities []string `json:"required_capabilities"`
	EngineProfile        string   `json:"engine_profile,omitempty"`
}

type roleReadinessResponse struct {
	RequiredRole   string   `json:"required_role"`
	Matched        bool     `json:"matched"`
	MatchedTypes   []string `json:"matched_types"`
	Status         string   `json:"status"`
	BlockedReasons []string `json:"blocked_reasons"`
}

type engineReadinessResponse struct {
	ProfileID      string         `json:"profile_id"`
	Provider       string         `json:"provider,omitempty"`
	Enabled        bool           `json:"enabled"`
	SecretRef      string         `json:"secret_ref"`
	SecretRequired bool           `json:"secret_required"`
	SecretReady    bool           `json:"secret_ready"`
	ResourceLimits map[string]any `json:"resource_limits"`
	Status         string         `json:"status"`
	BlockedReasons []string       `json:"blocked_reasons"`
}

type resourceReadinessResponse struct {
	MaxActiveLeases int64    `json:"max_active_leases"`
	MaxQueuedTasks  int64    `json:"max_queued_tasks"`
	Status          string   `json:"status"`
	BlockedReasons  []string `json:"blocked_reasons"`
}

type workerPoolSchedulePreviewResponse struct {
	Projects      []workerPoolProjectScheduleResponse `json:"projects"`
	Policy        workerPoolSchedulePolicyResponse    `json:"policy"`
	GeneratedAt   string                              `json:"generated_at"`
	Recommended   int64                               `json:"recommended"`
	Blocked       int64                               `json:"blocked"`
	QueuedTasks   int64                               `json:"queued_tasks"`
	AvailableSlot int64                               `json:"available_slots"`
}

type workerPoolSchedulePolicyResponse struct {
	Strategy               string `json:"strategy"`
	DefaultProjectPriority int    `json:"default_project_priority"`
	SlotStrategy           string `json:"slot_strategy"`
	DryRunOnly             bool   `json:"dry_run_only"`
}

type workerPoolProjectScheduleResponse struct {
	Project        projectRecordResponse     `json:"project"`
	Priority       int                       `json:"priority"`
	MaxParallel    int                       `json:"max_parallel"`
	AgentRole      string                    `json:"agent_role"`
	Role           roleReadinessResponse     `json:"role"`
	EngineProfile  string                    `json:"engine_profile,omitempty"`
	Engine         engineReadinessResponse   `json:"engine"`
	Resources      resourceReadinessResponse `json:"resources"`
	QueuedTasks    int64                     `json:"queued_tasks"`
	ActiveLeases   int64                     `json:"active_leases"`
	OnlineWorkers  int64                     `json:"online_workers"`
	AvailableSlots int64                     `json:"available_slots"`
	NeedsRecovery  int64                     `json:"needs_recovery"`
	Capabilities   []string                  `json:"capabilities"`
	RequiredCaps   []string                  `json:"required_capabilities"`
	Recommended    bool                      `json:"recommended"`
	BlockedReasons []string                  `json:"blocked_reasons"`
	NextAction     string                    `json:"next_action"`
}

type codexCLIAdapterPreviewResponse struct {
	Project                 projectRecordResponse               `json:"project"`
	Status                  string                              `json:"status"`
	Mode                    string                              `json:"mode"`
	Engine                  engineReadinessResponse             `json:"engine"`
	Command                 engineCommandPreviewResponse        `json:"command"`
	Capabilities            []engineCapabilityPreflightResponse `json:"capabilities"`
	Paths                   []enginePathPreflightResponse       `json:"paths"`
	ArtifactRedaction       artifactRedactionPlanResponse       `json:"artifact_redaction"`
	ForbiddenActions        []string                            `json:"forbidden_actions"`
	Blockers                []string                            `json:"blockers"`
	ExecutionAllowed        bool                                `json:"execution_allowed"`
	ProjectWriteAttempted   bool                                `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                                `json:"execution_write_attempted"`
	EngineCallAttempted     bool                                `json:"engine_call_attempted"`
	CommandsRun             bool                                `json:"commands_run"`
	SecretsResolved         bool                                `json:"secrets_resolved"`
	NetworkUsed             bool                                `json:"network_used"`
	GeneratedAt             string                              `json:"generated_at"`
}

type engineCommandPreviewResponse struct {
	Command           string `json:"command"`
	Allowed           bool   `json:"allowed"`
	Reason            string `json:"reason"`
	CapabilityAllowed bool   `json:"capability_allowed"`
	CommandAllowed    bool   `json:"command_allowed"`
	Denied            bool   `json:"denied"`
}

type engineCapabilityPreflightResponse struct {
	Capability string `json:"capability"`
	Required   bool   `json:"required"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
}

type enginePathPreflightResponse struct {
	Path       string `json:"path"`
	Capability string `json:"capability"`
	Effect     string `json:"effect"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
}

type artifactRedactionPlanResponse struct {
	Status         string   `json:"status"`
	RetentionClass string   `json:"retention_class"`
	Rules          []string `json:"rules"`
	RedactedFields []string `json:"redacted_fields"`
}

type artifactResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	WorkflowItemID    int64          `json:"workflow_item_id,omitempty"`
	ArtifactType      string         `json:"artifact_type"`
	StorageBackend    string         `json:"storage_backend"`
	URI               string         `json:"uri"`
	SourcePath        string         `json:"source_path"`
	SHA256            string         `json:"sha256"`
	SizeBytes         int64          `json:"size_bytes"`
	ContentType       string         `json:"content_type"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
}

type runnerPreflightResponse struct {
	Status   string                         `json:"status"`
	Checks   []projectReadinessItemResponse `json:"checks"`
	Blockers []string                       `json:"blockers"`
}

type runResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	RunType           string         `json:"run_type"`
	RunKind           string         `json:"run_kind"`
	Status            string         `json:"status"`
	RiskLevel         string         `json:"risk_level"`
	RiskPolicy        string         `json:"risk_policy"`
	DryRun            bool           `json:"dry_run"`
	Summary           map[string]any `json:"summary"`
	Metadata          map[string]any `json:"metadata"`
	StartedAt         string         `json:"started_at"`
	FinishedAt        string         `json:"finished_at,omitempty"`
}

type runTaskResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	WorkflowItemID    int64          `json:"workflow_item_id,omitempty"`
	RunID             int64          `json:"run_id"`
	TaskKey           string         `json:"task_key"`
	TaskKind          string         `json:"task_kind"`
	Status            string         `json:"status"`
	RiskLevel         string         `json:"risk_level"`
	Sequence          int            `json:"sequence"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type runAttemptResponse struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	WorkflowItemID    int64          `json:"workflow_item_id,omitempty"`
	RunID             int64          `json:"run_id"`
	RunTaskID         int64          `json:"run_task_id,omitempty"`
	AttemptKind       string         `json:"attempt_kind"`
	Status            string         `json:"status"`
	DryRun            bool           `json:"dry_run"`
	Metadata          map[string]any `json:"metadata"`
	StartedAt         string         `json:"started_at"`
	FinishedAt        string         `json:"finished_at,omitempty"`
}

type projectPhaseGateResponse struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	AcceptedWarnings []string `json:"accepted_warnings"`
	Blockers         []string `json:"blockers"`
}

// Serve starts the local AreaFlow API service.
func Serve(ctx context.Context, cfg config.ServerConfig) error {
	appCfg := config.FromEnv()
	pool, err := db.Open(ctx, appCfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	return ServeHandler(ctx, cfg, NewHandlerWithImporter(project.NewStore(pool), func(ctx context.Context, record project.Record, options importer.Options) (importer.Result, error) {
		return importer.ImportProject(ctx, pool, record, options)
	}))
}

func ServeHandler(ctx context.Context, cfg config.ServerConfig, handler http.Handler) error {
	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func NewHandler(store ProjectStore) http.Handler {
	server := Server{
		store:        store,
		doctorRunner: runProjectDoctor,
	}
	return server.handler()
}

func NewHandlerWithImporter(store ProjectStore, importer ProjectImporter) http.Handler {
	server := Server{
		store:        store,
		doctorRunner: runProjectDoctor,
		importer:     importer,
	}
	return server.handler()
}

func (server Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/service/status", server.localServiceStatusHandler)
	mux.HandleFunc("/api/backup/manifest", server.backupManifestHandler)
	mux.HandleFunc("/api/backup/restore-plan", server.restorePlanHandler)
	mux.HandleFunc("/api/release/readiness", server.releaseReadinessHandler)
	mux.HandleFunc("/api/release/remediation-plan", server.releaseRemediationPlanHandler)
	mux.HandleFunc("/api/release/acceptance-preview", server.releaseAcceptancePreviewHandler)
	mux.HandleFunc("/api/release/acceptance-gate", server.releaseAcceptanceGateHandler)
	mux.HandleFunc("/api/release/exception-doctor", server.releaseExceptionDoctorHandler)
	mux.HandleFunc("/api/release/exception-record-preview", server.releaseExceptionRecordPreviewHandler)
	mux.HandleFunc("/api/release/exception-schema-preview", server.releaseExceptionSchemaPreviewHandler)
	mux.HandleFunc("/api/release/exception-migration-approval-gate", server.releaseExceptionMigrationApprovalGateHandler)
	mux.HandleFunc("/api/release/exception-apply-preview", server.releaseExceptionApplyPreviewHandler)
	mux.HandleFunc("/api/release/final-gate", server.releaseFinalGateHandler)
	mux.HandleFunc("/api/release/evidence-bundle", server.releaseEvidenceBundleHandler)
	mux.HandleFunc("/api/release/package-preview", server.releasePackagePreviewHandler)
	mux.HandleFunc("/api/release/distribution-preview", server.releaseDistributionPreviewHandler)
	mux.HandleFunc("/api/release/publish-gate", server.releasePublishGateHandler)
	mux.HandleFunc("/api/release/publish-approval-preview", server.releasePublishApprovalPreviewHandler)
	mux.HandleFunc("/api/release/rollout-plan-preview", server.releaseRolloutPlanPreviewHandler)
	mux.HandleFunc("/api/audit/coverage", server.auditCoverageHandler)
	mux.HandleFunc("/api/permissions/doctor", server.permissionDoctorHandler)
	mux.HandleFunc("/api/conformance", server.conformanceHandler)
	mux.HandleFunc("/api/artifacts/integrity", server.artifactIntegrityHandler)
	mux.HandleFunc("/api/audit-events", server.auditEventsHandler)
	mux.HandleFunc("/api/events/stream", server.eventStreamHandler)
	mux.HandleFunc("/api/worker-pool/summary", server.workerPoolSummaryHandler)
	mux.HandleFunc("/api/worker-pool/schedule-preview", server.workerPoolSchedulePreviewHandler)
	mux.HandleFunc("/api/web/write-action-gate", server.webWriteActionGateHandler)
	mux.HandleFunc("/api/desktop/service-control-gate", server.desktopServiceControlGateHandler)
	mux.HandleFunc("/api/desktop/notification-gate", server.desktopNotificationGateHandler)
	mux.HandleFunc("/api/desktop/tray-menu-gate", server.desktopTrayMenuGateHandler)
	mux.HandleFunc("/api/security/boundary-readiness", server.securityBoundaryReadinessHandler)
	mux.HandleFunc("/api/completion-audit", server.completionAuditHandler)
	mux.HandleFunc("/api/completion-audit/snapshot-readiness", server.completionAuditSnapshotReadinessHandler)
	mux.HandleFunc("/api/ops/readiness", server.operationsReadinessHandler)
	mux.HandleFunc("/api/ops/support-bundle-preview", server.supportBundlePreviewHandler)
	mux.HandleFunc("/api/ops/migration-ledger-readiness", server.migrationLedgerReadinessHandler)
	mux.HandleFunc("/api/projects", server.projectsHandler)
	mux.HandleFunc("/api/projects/", server.projectHandler)
	mux.HandleFunc("/api/runs/", server.runsHandler)
	mux.HandleFunc("/api/artifacts/", server.artifactsHandler)
	return apiVersionAlias(mux)
}

func apiVersionAlias(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1" {
			rewritten := cloneRequestWithPath(r, "/api")
			next.ServeHTTP(w, rewritten)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			rewritten := cloneRequestWithPath(r, "/api/"+strings.TrimPrefix(r.URL.Path, "/api/v1/"))
			next.ServeHTTP(w, rewritten)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func cloneRequestWithPath(r *http.Request, path string) *http.Request {
	rewritten := r.Clone(r.Context())
	copiedURL := *r.URL
	copiedURL.Path = path
	copiedURL.RawPath = ""
	rewritten.URL = &copiedURL
	rewritten.RequestURI = requestURI(copiedURL)
	return rewritten
}

func requestURI(u url.URL) string {
	uri := u.RequestURI()
	if uri == "" {
		return u.Path
	}
	return uri
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: "areaflow",
	})
}

func runProjectDoctor(ctx context.Context, record project.Record, store ProjectStore, allowNative bool) (doctor.Report, error) {
	doctorStore, ok := store.(doctor.Store)
	if !ok {
		return doctor.Report{}, fmt.Errorf("project store does not support doctor checks")
	}
	if allowNative {
		return doctor.AreaMatrixWithNative(ctx, record, doctorStore)
	}
	return doctor.AreaMatrix(ctx, record, doctorStore)
}

func (s Server) localServiceStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/service/status" {
		writeError(w, http.StatusNotFound, "service status endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	status, err := s.store.LocalServiceStatus(r.Context(), project.LocalServiceStatusOptions{
		APIBaseURL:      requestAPIBaseURL(r),
		WebDashboardURL: r.URL.Query().Get("web_url"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load service status failed")
		return
	}
	writeJSON(w, http.StatusOK, buildLocalServiceStatusResponse(status))
}

func (s Server) webWriteActionGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/web/write-action-gate" {
		writeError(w, http.StatusNotFound, "web write action gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.WebWriteActionGate(r.Context(), project.WebWriteActionGateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load web write action gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWebWriteActionGateResponse(gate))
}

func (s Server) desktopServiceControlGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/desktop/service-control-gate" {
		writeError(w, http.StatusNotFound, "desktop service control gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.DesktopServiceControlGate(r.Context(), project.DesktopServiceControlGateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load desktop service control gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildDesktopServiceControlGateResponse(gate))
}

func (s Server) desktopNotificationGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/desktop/notification-gate" {
		writeError(w, http.StatusNotFound, "desktop notification gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.DesktopNotificationGate(r.Context(), project.DesktopNotificationGateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load desktop notification gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildDesktopNotificationGateResponse(gate))
}

func (s Server) desktopTrayMenuGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/desktop/tray-menu-gate" {
		writeError(w, http.StatusNotFound, "desktop tray menu gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.DesktopTrayMenuGate(r.Context(), project.DesktopTrayMenuGateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load desktop tray menu gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildDesktopTrayMenuGateResponse(gate))
}

func (s Server) securityBoundaryReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/security/boundary-readiness" {
		writeError(w, http.StatusNotFound, "security boundary readiness endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	readiness, err := s.store.SecurityBoundaryReadiness(r.Context(), project.SecurityBoundaryReadinessOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load security boundary readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildSecurityBoundaryReadinessResponse(readiness))
}

func (s Server) completionAuditHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/completion-audit" {
		writeError(w, http.StatusNotFound, "completion audit endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	audit, err := s.store.CompletionAudit(r.Context(), project.CompletionAuditOptions{
		APIBaseURL:      requestAPIBaseURL(r),
		WebDashboardURL: r.URL.Query().Get("web_url"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load completion audit failed")
		return
	}
	writeJSON(w, http.StatusOK, buildCompletionAuditResponse(audit))
}

func (s Server) completionAuditSnapshotReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/completion-audit/snapshot-readiness" {
		writeError(w, http.StatusNotFound, "completion audit snapshot readiness endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	if projectKey == "" {
		projectKey = "areamatrix"
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	readiness, err := s.store.CompletionAuditSnapshotReadiness(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load completion audit snapshot readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildCompletionAuditSnapshotReadinessResponse(readiness))
}

func (s Server) operationsReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/ops/readiness" {
		writeError(w, http.StatusNotFound, "operations readiness endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	readiness, err := s.store.OperationsReadiness(r.Context(), project.OperationsReadinessOptions{
		APIBaseURL:      requestAPIBaseURL(r),
		WebDashboardURL: r.URL.Query().Get("web_url"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load operations readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildOperationsReadinessResponse(readiness))
}

func (s Server) supportBundlePreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/ops/support-bundle-preview" {
		writeError(w, http.StatusNotFound, "support bundle preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.SupportBundlePreview(r.Context(), project.SupportBundlePreviewOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load support bundle preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildSupportBundlePreviewResponse(preview))
}

func (s Server) migrationLedgerReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/ops/migration-ledger-readiness" {
		writeError(w, http.StatusNotFound, "migration ledger readiness endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	readiness, err := s.store.MigrationLedgerReadiness(r.Context(), project.MigrationLedgerReadinessOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load migration ledger readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildMigrationLedgerReadinessResponse(readiness))
}

func (s Server) backupManifestHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/backup/manifest" {
		writeError(w, http.StatusNotFound, "backup manifest endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	manifest, err := s.store.BackupManifest(r.Context(), project.BackupManifestOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load backup manifest failed")
		return
	}
	writeJSON(w, http.StatusOK, buildBackupManifestResponse(manifest))
}

func (s Server) restorePlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/backup/restore-plan" {
		writeError(w, http.StatusNotFound, "restore plan endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	plan, err := s.store.RestorePlan(r.Context(), project.RestorePlanOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "build restore plan failed")
		return
	}
	writeJSON(w, http.StatusOK, buildRestorePlanResponse(plan))
}

func (s Server) releaseReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/readiness" {
		writeError(w, http.StatusNotFound, "release readiness endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	readiness, err := s.store.ReleaseReadiness(r.Context(), project.ReleaseReadinessOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseReadinessResponse(readiness))
}

func (s Server) releaseRemediationPlanHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/remediation-plan" {
		writeError(w, http.StatusNotFound, "release remediation plan endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	plan, err := s.store.ReleaseRemediationPlan(r.Context(), project.ReleaseRemediationOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release remediation plan failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseRemediationPlanResponse(plan))
}

func (s Server) releaseAcceptancePreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/acceptance-preview" {
		writeError(w, http.StatusNotFound, "release acceptance preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleaseAcceptancePreview(r.Context(), project.ReleaseAcceptancePreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release acceptance preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseAcceptancePreviewResponse(preview))
}

func (s Server) releaseAcceptanceGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/acceptance-gate" {
		writeError(w, http.StatusNotFound, "release acceptance gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.ReleaseAcceptanceGate(r.Context(), project.ReleaseAcceptanceGateOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release acceptance gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseAcceptanceGateResponse(gate))
}

func (s Server) releaseExceptionDoctorHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/exception-doctor" {
		writeError(w, http.StatusNotFound, "release exception doctor endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	doctor, err := s.store.ReleaseExceptionDoctor(r.Context(), project.ReleaseExceptionDoctorOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release exception doctor failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseExceptionDoctorResponse(doctor))
}

func (s Server) releaseExceptionRecordPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/exception-record-preview" {
		writeError(w, http.StatusNotFound, "release exception record preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleaseExceptionRecordPreview(r.Context(), project.ReleaseExceptionRecordPreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release exception record preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseExceptionRecordPreviewResponse(preview))
}

func (s Server) releaseExceptionSchemaPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/exception-schema-preview" {
		writeError(w, http.StatusNotFound, "release exception schema preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleaseExceptionSchemaPreview(r.Context(), project.ReleaseExceptionSchemaPreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release exception schema preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseExceptionSchemaPreviewResponse(preview))
}

func (s Server) releaseExceptionMigrationApprovalGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/exception-migration-approval-gate" {
		writeError(w, http.StatusNotFound, "release exception migration approval gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.ReleaseExceptionMigrationApprovalGate(r.Context(), project.ReleaseExceptionMigrationApprovalGateOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release exception migration approval gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseExceptionMigrationApprovalGateResponse(gate))
}

func (s Server) releaseExceptionApplyPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/exception-apply-preview" {
		writeError(w, http.StatusNotFound, "release exception apply preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleaseExceptionApplyPreview(r.Context(), project.ReleaseExceptionApplyPreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release exception apply preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseExceptionApplyPreviewResponse(preview))
}

func (s Server) releaseFinalGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/final-gate" {
		writeError(w, http.StatusNotFound, "release final gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.ReleaseFinalGate(r.Context(), project.ReleaseFinalGateOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release final gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseFinalGateResponse(gate))
}

func (s Server) releaseEvidenceBundleHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/evidence-bundle" {
		writeError(w, http.StatusNotFound, "release evidence bundle endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	bundle, err := s.store.ReleaseEvidenceBundle(r.Context(), project.ReleaseEvidenceBundleOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release evidence bundle failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseEvidenceBundleResponse(bundle))
}

func (s Server) releasePackagePreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/package-preview" {
		writeError(w, http.StatusNotFound, "release package preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleasePackagePreview(r.Context(), project.ReleasePackagePreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release package preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleasePackagePreviewResponse(preview))
}

func (s Server) releaseDistributionPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/distribution-preview" {
		writeError(w, http.StatusNotFound, "release distribution preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleaseDistributionPreview(r.Context(), project.ReleaseDistributionPreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release distribution preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseDistributionPreviewResponse(preview))
}

func (s Server) releasePublishGateHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/publish-gate" {
		writeError(w, http.StatusNotFound, "release publish gate endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	gate, err := s.store.ReleasePublishGate(r.Context(), project.ReleasePublishGateOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release publish gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleasePublishGateResponse(gate))
}

func (s Server) releasePublishApprovalPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/publish-approval-preview" {
		writeError(w, http.StatusNotFound, "release publish approval preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleasePublishApprovalPreview(r.Context(), project.ReleasePublishApprovalPreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release publish approval preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleasePublishApprovalPreviewResponse(preview))
}

func (s Server) releaseRolloutPlanPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/release/rollout-plan-preview" {
		writeError(w, http.StatusNotFound, "release rollout plan preview endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.ReleaseRolloutPlanPreview(r.Context(), project.ReleaseRolloutPlanPreviewOptions{ProjectKey: releaseProjectKeyFromRequest(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load release rollout plan preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildReleaseRolloutPlanPreviewResponse(preview))
}

func releaseProjectKeyFromRequest(r *http.Request) string {
	projectKey := strings.TrimSpace(r.URL.Query().Get("project"))
	if projectKey != "" {
		return projectKey
	}
	return strings.TrimSpace(r.URL.Query().Get("project_key"))
}

func (s Server) auditCoverageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/audit/coverage" {
		writeError(w, http.StatusNotFound, "audit coverage endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	var projectID int64
	if projectKey != "" {
		record, err := s.store.GetByKey(r.Context(), projectKey)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		projectID = record.ID
		projectKey = record.Key
	}
	coverage, err := s.store.AuditCoverage(r.Context(), project.AuditCoverageOptions{
		ProjectID:  projectID,
		ProjectKey: projectKey,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load audit coverage failed")
		return
	}
	writeJSON(w, http.StatusOK, buildAuditCoverageResponse(coverage))
}

func (s Server) permissionDoctorHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/permissions/doctor" {
		writeError(w, http.StatusNotFound, "permission doctor endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	if projectKey == "" {
		writeError(w, http.StatusBadRequest, "project_key is required")
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	doctor, err := s.store.PermissionPolicyDoctor(r.Context(), record, project.PermissionPolicyDoctorOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load permission policy doctor failed")
		return
	}
	writeJSON(w, http.StatusOK, buildPermissionPolicyDoctorResponse(doctor))
}

func (s Server) conformanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/conformance" {
		writeError(w, http.StatusNotFound, "conformance endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	if projectKey == "" {
		writeError(w, http.StatusBadRequest, "project_key is required")
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	report, err := s.store.ConformanceCheck(r.Context(), record, project.ConformanceOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "check adapter/profile conformance failed")
		return
	}
	writeJSON(w, http.StatusOK, buildConformanceResponse(report))
}

func (s Server) projectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/projects" {
		writeError(w, http.StatusNotFound, "projects endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	records, err := s.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list projects failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectListResponse(records))
}

func requestAPIBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}
	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	if host == "" {
		return "/api/v1"
	}
	return scheme + "://" + host + "/api/v1"
}

func (s Server) projectHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || len(parts) > 4 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "project endpoint not found")
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		record, err := s.store.GetByKey(r.Context(), parts[0])
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", parts[0]))
			return
		}
		writeJSON(w, http.StatusOK, buildProjectRecordResponse(record))
		return
	}
	switch parts[1] {
	case "events":
		if len(parts) == 3 && parts[2] == "stream" && r.Method == http.MethodGet {
			s.projectEventStreamHandler(w, r, parts[0])
			return
		}
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectEventsHandler(w, r, parts[0])
	case "summary":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectSummaryHandler(w, r, parts[0])
	case "doctor":
		if len(parts) != 2 || r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectDoctorHandler(w, r, parts[0])
	case "import":
		if len(parts) != 2 || r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectImportHandler(w, r, parts[0])
	case "readiness":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectReadinessHandler(w, r, parts[0])
	case "generated-write-readiness":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectGeneratedWriteReadinessHandler(w, r, parts[0])
	case "generated-write-apply-beta-gate":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectGeneratedWriteApplyBetaGateHandler(w, r, parts[0])
	case "import-diff":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectImportDiffHandler(w, r, parts[0])
	case "verification-bundle":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectVerificationBundleHandler(w, r, parts[0])
	case "compatibility":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectCompatibilityHandler(w, r, parts[0])
	case "shim-preview":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectShimPreviewHandler(w, r, parts[0])
	case "shim-authorization":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectShimAuthorizationHandler(w, r, parts[0])
	case "shim-apply-packet":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectShimApplyPacketHandler(w, r, parts[0])
	case "shim-apply-gate":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectShimApplyGateHandler(w, r, parts[0])
	case "shim-apply":
		if len(parts) != 2 || r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectShimApplyHandler(w, r, parts[0])
	case "shim-readiness":
		if len(parts) == 3 && parts[2] == "evidence" {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			s.projectShimReadinessEvidenceHandler(w, r, parts[0])
			return
		}
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectShimReadinessHandler(w, r, parts[0])
	case "execution-cutover-readiness":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionCutoverReadinessHandler(w, r, parts[0])
	case "execution-forwarding-v1-readiness":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1ReadinessHandler(w, r, parts[0])
	case "execution-forwarding-v1-apply-preview":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1ApplyPreviewHandler(w, r, parts[0])
	case "execution-forwarding-v1-apply-packet":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1ApplyPacketHandler(w, r, parts[0])
	case "execution-forwarding-v1-apply-gate":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1ApplyGateHandler(w, r, parts[0])
	case "execution-forwarding-v1-apply":
		if len(parts) != 2 || r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1ApplyHandler(w, r, parts[0])
	case "execution-forwarding-v1-command-preview":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1CommandPreviewHandler(w, r, parts[0])
	case "execution-forwarding-v1-rollback-preview":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectExecutionForwardingV1RollbackPreviewHandler(w, r, parts[0])
	case "cutover-readiness":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectCutoverReadinessHandler(w, r, parts[0])
	case "cutover-apply":
		if len(parts) != 2 || r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectCutoverApplyHandler(w, r, parts[0])
	case "status-projections":
		if len(parts) == 3 && parts[2] == "authorization" {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			s.projectStatusProjectionAuthorizationHandler(w, r, parts[0])
			return
		}
		if len(parts) == 3 && parts[2] == "apply-packet" {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			s.projectStatusProjectionApplyPacketHandler(w, r, parts[0])
			return
		}
		if len(parts) == 3 && parts[2] == "apply-gate" {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			s.projectStatusProjectionApplyGateHandler(w, r, parts[0])
			return
		}
		if len(parts) == 3 && parts[2] == "apply" {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			s.projectStatusProjectionApplyHandler(w, r, parts[0])
			return
		}
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectStatusProjectionsHandler(w, r, parts[0])
	case "artifacts":
		if len(parts) == 3 && parts[2] == "archive-preview" {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			s.projectArtifactArchivePreviewHandler(w, r, parts[0])
			return
		}
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectArtifactsHandler(w, r, parts[0])
	case "residuals":
		if len(parts) != 2 || r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.projectResidualsHandler(w, r, parts[0])
	case "workflow-versions":
		s.projectWorkflowVersionsHandler(w, r, parts[0], parts[2:])
	case "workers":
		s.projectWorkersHandler(w, r, parts[0], parts[2:])
	case "engines":
		s.projectEnginesHandler(w, r, parts[0], parts[2:])
	default:
		writeError(w, http.StatusNotFound, "project endpoint not found")
	}
}

func (s Server) runsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || len(parts) > 3 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "run endpoint not found")
		return
	}
	runID, err := parsePositiveInt64(parts[0], "run id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	scope, err := s.projectVisibilityScopeFromQuery(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		detail, err := s.store.GetRun(r.Context(), runID)
		if err != nil {
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "load run failed")
			return
		}
		if !scope.allows(detail.Run.ProjectID) {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, buildRunDetailResponse(detail))
		return
	}
	if err := s.ensureRunVisibleToProject(r.Context(), scope, runID); err != nil {
		if errors.Is(err, project.ErrRunNotFound) {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load run failed")
		return
	}
	if parts[1] == "start" || parts[1] == "drain" || parts[1] == "cancel" {
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "run endpoint not found")
			return
		}
		s.runControlHandler(w, r, runID, parts[1])
		return
	}
	if parts[1] == "execution-approval-gate" {
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "run endpoint not found")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		gate, err := s.store.ExecutionApprovalGate(r.Context(), runID, project.ExecutionApprovalGateOptions{})
		if err != nil {
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "load execution approval gate failed")
			return
		}
		writeJSON(w, http.StatusOK, buildExecutionApprovalGateResponse(gate))
		return
	}
	if parts[1] == "execution-plan" {
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "run endpoint not found")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		preview, err := s.store.PreviewExecutionPlan(r.Context(), runID)
		if err != nil {
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "load execution plan failed")
			return
		}
		writeJSON(w, http.StatusOK, buildExecutionPlanPreviewResponse(preview))
		return
	}
	if parts[1] == "project-write-design-gate" {
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "run endpoint not found")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		gate, err := s.store.PreviewProjectWriteDesignGate(r.Context(), runID)
		if err != nil {
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "load project write design gate failed")
			return
		}
		writeJSON(w, http.StatusOK, buildProjectWriteDesignGateResponse(gate))
		return
	}
	if parts[1] == "managed-generated-write-gate" {
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "run endpoint not found")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		gate, err := s.store.PreviewManagedGeneratedWriteGate(r.Context(), runID)
		if err != nil {
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "load managed generated write gate failed")
			return
		}
		writeJSON(w, http.StatusOK, buildManagedGeneratedWriteGateResponse(gate))
		return
	}
	if parts[1] != "events" {
		writeError(w, http.StatusNotFound, "run endpoint not found")
		return
	}
	if len(parts) == 3 {
		if parts[2] != "stream" {
			writeError(w, http.StatusNotFound, "run endpoint not found")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.runEventStreamHandler(w, r, runID)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	events, err := s.store.ListRunEvents(r.Context(), runID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list run events failed")
		return
	}
	writeJSON(w, http.StatusOK, buildRunEventsResponse(runID, events))
}

func (s Server) runControlHandler(w http.ResponseWriter, r *http.Request, runID int64, action string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request runControlRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid run control request")
		return
	}
	result, err := s.store.ControlRun(r.Context(), runID, project.RunControlOptions{
		Action:         action,
		Actor:          request.Actor,
		Reason:         request.Reason,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		if errors.Is(err, project.ErrRunNotFound) {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		if errors.Is(err, project.ErrRunControlBlocked) || errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "run control failed")
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildRunControlResponse(result))
}

func (s Server) artifactsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/artifacts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 1 && len(parts) != 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "artifact endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifactID, err := parsePositiveInt64(parts[0], "artifact id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	scope, err := s.projectVisibilityScopeFromQuery(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if len(parts) == 2 {
		if parts[1] != "content" {
			writeError(w, http.StatusNotFound, "artifact endpoint not found")
			return
		}
		if err := s.ensureArtifactVisibleToProject(r.Context(), scope, artifactID); err != nil {
			if errors.Is(err, project.ErrArtifactNotFound) {
				writeError(w, http.StatusNotFound, "artifact not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "load artifact failed")
			return
		}
		content, err := s.store.GetArtifactContent(r.Context(), artifactID)
		if err != nil {
			switch {
			case errors.Is(err, project.ErrArtifactNotFound):
				writeError(w, http.StatusNotFound, "artifact not found")
			case errors.Is(err, project.ErrArtifactContentMismatch):
				writeError(w, http.StatusConflict, "artifact content does not match metadata")
			case errors.Is(err, project.ErrArtifactContentUnavailable):
				writeError(w, http.StatusUnprocessableEntity, "artifact content is unavailable")
			default:
				writeError(w, http.StatusInternalServerError, "load artifact content failed")
			}
			return
		}
		if !scope.allows(content.Artifact.ProjectID) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		w.Header().Set("Content-Type", content.ContentType)
		w.Header().Set("X-AreaFlow-Artifact-ID", strconv.FormatInt(content.Artifact.ID, 10))
		if content.Artifact.SHA256 != "" {
			w.Header().Set("X-AreaFlow-Artifact-SHA256", content.Artifact.SHA256)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content.Content)
		return
	}
	artifact, err := s.store.GetArtifact(r.Context(), artifactID)
	if err != nil {
		if errors.Is(err, project.ErrArtifactNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load artifact failed")
		return
	}
	if !scope.allows(artifact.ProjectID) {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}
	writeJSON(w, http.StatusOK, buildArtifactResponse(artifact))
}

func (s Server) artifactIntegrityHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/artifacts/integrity" {
		writeError(w, http.StatusNotFound, "artifact integrity endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	if projectKey == "" {
		writeError(w, http.StatusBadRequest, "project_key is required")
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	report, err := s.store.ArtifactIntegrity(r.Context(), record, project.ArtifactIntegrityOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "check artifact integrity failed")
		return
	}
	writeJSON(w, http.StatusOK, buildArtifactIntegrityResponse(report))
}

func (s Server) projectArtifactArchivePreviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	var request artifactArchivePreviewRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid artifact archive preview request")
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	result, err := s.store.ArtifactArchivePreview(r.Context(), record, project.ArtifactArchivePreviewOptions{
		RetentionClass: request.RetentionClass,
		Limit:          request.Limit,
		Actor:          request.Actor,
		Reason:         request.Reason,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "artifact archive preview failed")
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildArtifactArchivePreviewResponse(result))
}

func (s Server) eventStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.writeEventStream(w, r, project.EventStreamFilter{})
}

func (s Server) projectEventStreamHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	s.writeEventStream(w, r, project.EventStreamFilter{ProjectID: record.ID})
}

func (s Server) runEventStreamHandler(w http.ResponseWriter, r *http.Request, runID int64) {
	s.writeEventStream(w, r, project.EventStreamFilter{RunID: runID})
}

func (s Server) writeEventStream(w http.ResponseWriter, r *http.Request, filter project.EventStreamFilter) {
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	afterID, err := queryOptionalInt64(r, "after_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	filter.Limit = limit
	filter.AfterID = afterID
	once := r.URL.Query().Get("once") == "true"
	interval := time.Second

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fmt.Fprint(w, ": areaflow event stream\n\n")
	flusher.Flush()

	for {
		events, err := s.store.ListEventStream(r.Context(), filter)
		if err != nil {
			_ = writeSSEEvent(w, "error", map[string]string{"error": "list event stream failed"}, 0)
			flusher.Flush()
			return
		}
		for _, event := range events {
			if err := writeSSEEvent(w, event.Type, buildEventResponse(event), event.ID); err != nil {
				return
			}
			filter.AfterID = event.ID
		}
		flusher.Flush()
		if once {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(interval):
		}
	}
}

func (s Server) projectEnginesHandler(w http.ResponseWriter, r *http.Request, projectKey string, rest []string) {
	if len(rest) != 2 || rest[0] != "codex-cli" || rest[1] != "preview" {
		writeError(w, http.StatusNotFound, "engine endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	preview, err := s.store.CodexCLIAdapterPreview(r.Context(), record, project.CodexCLIAdapterPreviewOptions{
		Command: strings.TrimSpace(r.URL.Query().Get("command")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load codex cli adapter preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildCodexCLIAdapterPreviewResponse(preview))
}

func (s Server) projectWorkersHandler(w http.ResponseWriter, r *http.Request, projectKey string, rest []string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			limit, err := queryLimit(r, 20)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			workers, err := s.store.ListWorkers(r.Context(), record, limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "list workers failed")
				return
			}
			writeJSON(w, http.StatusOK, buildWorkerListResponse(record, workers))
			return
		case http.MethodPost:
			var request registerWorkerRequest
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "invalid worker register request")
				return
			}
			worker, err := s.store.RegisterWorker(r.Context(), record, project.RegisterWorkerOptions{
				WorkerKey:                request.WorkerKey,
				WorkerType:               request.WorkerType,
				Hostname:                 request.Hostname,
				PID:                      request.PID,
				Capabilities:             request.Capabilities,
				Metadata:                 request.Metadata,
				HeartbeatIntervalSeconds: request.HeartbeatIntervalSeconds,
				LeaseTimeoutSeconds:      request.LeaseTimeoutSeconds,
				Actor:                    request.Actor,
				Reason:                   request.Reason,
				IdempotencyKey:           request.IdempotencyKey,
			})
			if err != nil {
				if errors.Is(err, project.ErrIdempotencyConflict) {
					writeError(w, http.StatusConflict, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, "register worker failed")
				return
			}
			writeJSON(w, http.StatusCreated, buildWorkerResponse(worker))
			return
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
	}
	if len(rest) == 2 && rest[1] == "heartbeat" && r.Method == http.MethodPost {
		var request workerHeartbeatRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid worker heartbeat request")
			return
		}
		worker, err := s.store.RecordWorkerHeartbeat(r.Context(), record, rest[0], project.WorkerHeartbeatOptions{
			Status:         request.Status,
			Metadata:       request.Metadata,
			Actor:          request.Actor,
			Reason:         request.Reason,
			IdempotencyKey: request.IdempotencyKey,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "record worker heartbeat failed")
			return
		}
		writeJSON(w, http.StatusOK, buildWorkerResponse(worker))
		return
	}
	if len(rest) == 2 && rest[1] == "lease-acquire" && r.Method == http.MethodPost {
		var request leaseAcquireRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid lease acquire request")
			return
		}
		lease, err := s.store.AcquireLease(r.Context(), record, project.AcquireLeaseOptions{
			WorkerKey:            rest[0],
			RunTaskID:            request.RunTaskID,
			LeaseKind:            request.LeaseKind,
			AllowedCapabilities:  request.AllowedCapabilities,
			Scope:                request.Scope,
			Metadata:             request.Metadata,
			LeaseTimeoutSeconds:  request.LeaseTimeoutSeconds,
			RecoverExpiredBefore: request.RecoverExpiredBefore,
			Actor:                request.Actor,
			Reason:               request.Reason,
			IdempotencyKey:       request.IdempotencyKey,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			writeError(w, http.StatusInternalServerError, "acquire lease failed")
			return
		}
		writeJSON(w, http.StatusCreated, buildLeaseResponse(lease))
		return
	}
	if len(rest) == 2 && rest[1] == "lease-release" && r.Method == http.MethodPost {
		var request leaseReleaseRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid lease release request")
			return
		}
		lease, err := s.store.ReleaseLease(r.Context(), record, project.ReleaseLeaseOptions{
			WorkerKey:      rest[0],
			LeaseID:        request.LeaseID,
			Status:         request.Status,
			Metadata:       request.Metadata,
			Actor:          request.Actor,
			Reason:         request.Reason,
			IdempotencyKey: request.IdempotencyKey,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) || errors.Is(err, project.ErrLeaseNotFound) {
				writeError(w, http.StatusNotFound, "lease not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "release lease failed")
			return
		}
		writeJSON(w, http.StatusOK, buildLeaseResponse(lease))
		return
	}
	if len(rest) == 2 && rest[1] == "run-once" && r.Method == http.MethodPost {
		var request workerRunOnceRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid worker run-once request")
			return
		}
		result, err := s.store.RunWorkerOnce(r.Context(), record, project.WorkerRunOnceOptions{
			WorkerKey:           rest[0],
			RunID:               request.RunID,
			AllowedCapabilities: request.AllowedCapabilities,
			LeaseTimeoutSeconds: request.LeaseTimeoutSeconds,
			Metadata:            request.Metadata,
			Actor:               request.Actor,
			Reason:              request.Reason,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			writeError(w, http.StatusInternalServerError, "worker run-once failed")
			return
		}
		writeJSON(w, http.StatusOK, buildWorkerRunOnceResponse(result))
		return
	}
	if len(rest) == 2 && rest[1] == "fixture-execute" && r.Method == http.MethodPost {
		var request fixtureExecutionRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid fixture execution request")
			return
		}
		result, err := s.store.ExecuteFixture(r.Context(), record, project.FixtureExecutionOptions{
			WorkerKey:           rest[0],
			RunID:               request.RunID,
			AllowedCapabilities: request.AllowedCapabilities,
			LeaseTimeoutSeconds: request.LeaseTimeoutSeconds,
			Metadata:            request.Metadata,
			IdempotencyKey:      request.IdempotencyKey,
			Actor:               request.Actor,
			Reason:              request.Reason,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			if errors.Is(err, project.ErrFixtureExecutionBlocked) || errors.Is(err, project.ErrNoLeaseAvailable) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "fixture execution failed")
			return
		}
		status := http.StatusCreated
		if !result.Created {
			status = http.StatusOK
		}
		writeJSON(w, status, buildFixtureExecutionResponse(result))
		return
	}
	if len(rest) == 2 && rest[1] == "read-only-verify" && r.Method == http.MethodPost {
		var request readOnlyVerifyRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid read-only verify request")
			return
		}
		result, err := s.store.VerifyReadOnly(r.Context(), record, project.ReadOnlyVerifyOptions{
			WorkerKey:           rest[0],
			RunID:               request.RunID,
			AllowedCapabilities: request.AllowedCapabilities,
			LeaseTimeoutSeconds: request.LeaseTimeoutSeconds,
			Metadata:            request.Metadata,
			IdempotencyKey:      request.IdempotencyKey,
			Actor:               request.Actor,
			Reason:              request.Reason,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			if errors.Is(err, project.ErrReadOnlyVerifyBlocked) || errors.Is(err, project.ErrNoLeaseAvailable) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "read-only verify failed")
			return
		}
		status := http.StatusCreated
		if !result.Created {
			status = http.StatusOK
		}
		writeJSON(w, status, buildReadOnlyVerifyResponse(result))
		return
	}
	if len(rest) == 2 && rest[1] == "approved-artifact-write" && r.Method == http.MethodPost {
		var request approvedArtifactWriteRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid approved artifact write request")
			return
		}
		result, err := s.store.WriteApprovedArtifact(r.Context(), record, project.ApprovedArtifactWriteOptions{
			WorkerKey:           rest[0],
			RunID:               request.RunID,
			AllowedCapabilities: request.AllowedCapabilities,
			LeaseTimeoutSeconds: request.LeaseTimeoutSeconds,
			Metadata:            request.Metadata,
			IdempotencyKey:      request.IdempotencyKey,
			Actor:               request.Actor,
			Reason:              request.Reason,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			if errors.Is(err, project.ErrApprovedArtifactWriteBlocked) || errors.Is(err, project.ErrNoLeaseAvailable) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "approved artifact write failed")
			return
		}
		status := http.StatusCreated
		if !result.Created {
			status = http.StatusOK
		}
		writeJSON(w, status, buildApprovedArtifactWriteResponse(result))
		return
	}
	if len(rest) == 2 && rest[1] == "fixture-project-write" && r.Method == http.MethodPost {
		var request fixtureProjectWriteRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid fixture project write request")
			return
		}
		result, err := s.store.WriteFixtureProject(r.Context(), record, project.FixtureProjectWriteOptions{
			WorkerKey:           rest[0],
			RunID:               request.RunID,
			AllowedCapabilities: request.AllowedCapabilities,
			LeaseTimeoutSeconds: request.LeaseTimeoutSeconds,
			Metadata:            request.Metadata,
			IdempotencyKey:      request.IdempotencyKey,
			Actor:               request.Actor,
			Reason:              request.Reason,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			if errors.Is(err, project.ErrFixtureProjectWriteBlocked) || errors.Is(err, project.ErrNoLeaseAvailable) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "fixture project write failed")
			return
		}
		status := http.StatusCreated
		if !result.Created {
			status = http.StatusOK
		}
		writeJSON(w, status, buildFixtureProjectWriteResponse(result))
		return
	}
	if len(rest) == 2 && rest[1] == "managed-generated-write" && r.Method == http.MethodPost {
		var request managedGeneratedWriteRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid managed generated write request")
			return
		}
		result, err := s.store.WriteManagedGenerated(r.Context(), record, project.ManagedGeneratedWriteOptions{
			WorkerKey:           rest[0],
			RunID:               request.RunID,
			AllowedCapabilities: request.AllowedCapabilities,
			LeaseTimeoutSeconds: request.LeaseTimeoutSeconds,
			Metadata:            request.Metadata,
			IdempotencyKey:      request.IdempotencyKey,
			Actor:               request.Actor,
			Reason:              request.Reason,
		})
		if err != nil {
			if errors.Is(err, project.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, "worker not found")
				return
			}
			if errors.Is(err, project.ErrRunNotFound) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, project.ErrWorkerCapabilityDenied) {
				writeError(w, http.StatusForbidden, "worker capability denied")
				return
			}
			if errors.Is(err, project.ErrManagedGeneratedWriteBlocked) || errors.Is(err, project.ErrNoLeaseAvailable) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "managed generated write failed")
			return
		}
		status := http.StatusCreated
		if !result.Created {
			status = http.StatusOK
		}
		writeJSON(w, status, buildManagedGeneratedWriteResponse(result))
		return
	}
	if len(rest) == 1 && rest[0] == "lease-recover" && r.Method == http.MethodPost {
		var request leaseRecoverRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid lease recover request")
			return
		}
		leases, err := s.store.RecoverExpiredLeases(r.Context(), record, project.RecoverLeasesOptions{
			Limit:          request.Limit,
			Metadata:       request.Metadata,
			Actor:          request.Actor,
			Reason:         request.Reason,
			IdempotencyKey: request.IdempotencyKey,
		})
		if err != nil {
			if errors.Is(err, project.ErrIdempotencyConflict) {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "recover leases failed")
			return
		}
		writeJSON(w, http.StatusOK, buildLeaseRecoverResponse(record, leases))
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s Server) projectEventsHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	limit, err := queryLimit(r, 10)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	events, err := s.store.ListEvents(r.Context(), record.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list project events failed")
		return
	}

	response := projectEventsResponse{
		Project: record.Key,
		Events:  make([]eventResponse, 0, len(events)),
	}
	for _, event := range events {
		response.Events = append(response.Events, buildEventResponse(event))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s Server) workerPoolSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/worker-pool/summary" {
		writeError(w, http.StatusNotFound, "worker pool endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	summary, err := s.store.WorkerPoolSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load worker pool summary failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWorkerPoolSummaryResponse(summary))
}

func (s Server) workerPoolSchedulePreviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/worker-pool/schedule-preview" {
		writeError(w, http.StatusNotFound, "worker pool endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	preview, err := s.store.WorkerPoolSchedulePreview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load worker pool schedule preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWorkerPoolSchedulePreviewResponse(preview))
}

func (s Server) auditEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/audit-events" {
		writeError(w, http.StatusNotFound, "audit events endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var projectID int64
	projectKey := strings.TrimSpace(r.URL.Query().Get("project_key"))
	if projectKey != "" {
		record, err := s.store.GetByKey(r.Context(), projectKey)
		if err != nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
			return
		}
		projectID = record.ID
		projectKey = record.Key
	}

	events, err := s.store.ListAuditEvents(r.Context(), projectID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list audit events failed")
		return
	}
	writeJSON(w, http.StatusOK, buildAuditEventsResponse(projectKey, events))
}

func (s Server) projectSummaryHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	summary, err := s.store.ProjectSummary(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load project summary failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectSummaryResponse(summary))
}

func (s Server) projectReadinessHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	readiness, err := s.store.ProjectReadiness(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load project readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectReadinessResponse(readiness))
}

func (s Server) projectGeneratedWriteReadinessHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	readiness, err := s.store.GeneratedWriteReadiness(r.Context(), record, project.GeneratedWriteReadinessOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load generated write readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildGeneratedWriteReadinessResponse(readiness))
}

func (s Server) projectGeneratedWriteApplyBetaGateHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	gate, err := s.store.GeneratedWriteApplyBetaGate(r.Context(), record, project.GeneratedWriteApplyBetaGateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load generated write apply beta gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildGeneratedWriteApplyBetaGateResponse(gate))
}

func (s Server) projectDoctorHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request projectDoctorRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid project doctor request")
		return
	}
	runner := s.doctorRunner
	if runner == nil {
		runner = runProjectDoctor
	}
	report, err := runner(r.Context(), record, s.store, request.AllowNative)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "run project doctor failed")
		return
	}
	result, err := s.store.RecordDoctorReport(r.Context(), record.ID, report.Summary(), project.RecordDoctorReportOptions{
		IdempotencyKey: request.IdempotencyKey,
		Actor:          request.Actor,
		Reason:         request.Reason,
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "record project doctor failed")
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, buildProjectDoctorRunResponse(record, report, result))
}

func (s Server) projectImportHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request projectImportRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid project import request")
		return
	}
	if s.importer == nil {
		writeError(w, http.StatusNotImplemented, "project import runner is not configured")
		return
	}
	result, err := s.importer(r.Context(), record, importer.Options{
		IdempotencyKey: request.IdempotencyKey,
		Actor:          request.Actor,
		Reason:         request.Reason,
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "import project failed")
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, buildProjectImportRunResponse(record, result))
}

func (s Server) projectImportDiffHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	diff, err := s.store.ProjectImportDiff(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load project import diff failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectImportDiffResponse(diff))
}

func (s Server) projectVerificationBundleHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	limit, err := queryLimit(r, 10)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	bundle, err := s.store.ProjectVerificationBundle(r.Context(), record, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load project verification bundle failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectVerificationBundleResponse(bundle))
}

func (s Server) projectCompatibilityHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	contract, err := s.store.CompatibilityContract(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load compatibility contract failed")
		return
	}
	writeJSON(w, http.StatusOK, buildCompatibilityContractResponse(contract))
}

func (s Server) projectShimPreviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	preview, err := s.store.ShimPreview(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load shim preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildShimPreviewResponse(preview))
}

func (s Server) projectShimReadinessHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	readiness, err := s.store.ShimReadiness(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load shim readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildShimReadinessResponse(readiness))
}

func (s Server) projectShimAuthorizationHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	packet, err := s.store.ShimAuthorizationPacket(r.Context(), record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load shim authorization packet failed")
		return
	}
	writeJSON(w, http.StatusOK, buildShimAuthorizationPacketResponse(packet))
}

func (s Server) projectShimApplyPacketHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	options, err := shimApplyPacketOptionsFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, err := s.store.ShimApplyPacketPreview(r.Context(), record, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "preview shim apply packet failed")
		return
	}
	writeJSON(w, http.StatusOK, buildShimApplyPacketPreviewResponse(preview))
}

func (s Server) projectShimApplyGateHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	options, err := shimApplyGateOptionsFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	gate, err := s.store.ShimApplyGate(r.Context(), record, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "run shim apply gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildShimApplyGateResponse(gate))
}

func (s Server) projectShimApplyHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request shimApplyCommandRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid shim apply request")
		return
	}
	result, err := s.store.ApplyShimCommand(r.Context(), record, project.ApplyShimCommandOptions{
		IdempotencyKey: request.IdempotencyKey,
		Actor:          request.ApprovalActor,
		Reason:         request.ApprovalReason,
		Gate: project.ShimApplyGateOptions{
			AllowedFiles:               request.AllowedFiles,
			ApprovalID:                 request.ApprovalID,
			ApprovalScope:              request.ApprovalScope,
			AuthorizationSnapshotHash:  request.AuthorizationSnapshotHash,
			ExpectedAuthorizationMode:  request.ExpectedAuthorizationMode,
			StatusProjectionPacketID:   request.StatusProjectionPacketID,
			StatusProjectionGateID:     request.StatusProjectionGateID,
			ReadOnlySmokeEvidenceID:    request.ReadOnlySmokeEvidenceID,
			DirtyWorktreeReviewID:      request.DirtyWorktreeReviewID,
			ProtectedPathFingerprintID: request.ProtectedPathFingerprintID,
			RollbackPlanID:             request.RollbackPlanID,
			IdempotencyKey:             request.IdempotencyKey,
			AuditCorrelationID:         request.AuditCorrelationID,
			FailureMode:                request.FailureMode,
			ExplicitApproval:           request.ExplicitApproval,
			ApprovalActor:              request.ApprovalActor,
			ApprovalReason:             request.ApprovalReason,
		},
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "apply shim failed")
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
		if result.Decision == "denied" {
			status = http.StatusConflict
		}
	}
	writeJSON(w, status, buildShimApplyCommandResponse(result))
}

func (s Server) projectShimReadinessEvidenceHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request shimReadinessEvidenceRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid shim readiness evidence request")
		return
	}
	result, err := s.store.RecordShimReadinessEvidence(r.Context(), record, project.RecordShimReadinessEvidenceOptions{
		EvidenceKey:    request.EvidenceKey,
		Status:         request.Status,
		Summary:        request.Summary,
		EvidenceURI:    request.EvidenceURI,
		IdempotencyKey: request.IdempotencyKey,
		Actor:          request.Actor,
		Reason:         request.Reason,
		Metadata:       request.Metadata,
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	writeJSON(w, status, buildShimReadinessEvidenceResponse(result))
}

func (s Server) projectExecutionCutoverReadinessHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	readiness, err := s.store.AreaMatrixExecutionCutoverReadiness(r.Context(), record, project.AreaMatrixExecutionCutoverReadinessOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load execution cutover readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionCutoverReadinessResponse(readiness))
}

func (s Server) projectExecutionForwardingV1ReadinessHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	readiness, err := s.store.ExecutionForwardingV1Readiness(r.Context(), record, project.ExecutionForwardingV1ReadinessOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load execution forwarding v1 readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionForwardingV1ReadinessResponse(readiness))
}

func (s Server) projectExecutionForwardingV1ApplyPreviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	preview, err := s.store.ExecutionForwardingV1ApplyPreview(r.Context(), record, project.ExecutionForwardingV1ApplyPreviewOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load execution forwarding v1 apply preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionForwardingV1ApplyPreviewResponse(preview))
}

func (s Server) projectExecutionForwardingV1ApplyPacketHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	options, err := executionForwardingV1ApplyPacketOptionsFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, err := s.store.ExecutionForwardingV1ApplyPacketPreview(r.Context(), record, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "preview execution forwarding v1 apply packet failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionForwardingV1ApplyPacketPreviewResponse(preview))
}

func (s Server) projectExecutionForwardingV1ApplyGateHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	options, err := executionForwardingV1ApplyGateOptionsFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	gate, err := s.store.ExecutionForwardingV1ApplyGate(r.Context(), record, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "run execution forwarding v1 apply gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionForwardingV1ApplyGateResponse(gate))
}

func (s Server) projectExecutionForwardingV1CommandPreviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	taskType := strings.TrimSpace(r.URL.Query().Get("task_type"))
	if taskType == "" {
		writeError(w, http.StatusBadRequest, "task_type is required")
		return
	}
	preview, err := s.store.ExecutionForwardingV1CommandPreview(r.Context(), record, project.ExecutionForwardingV1CommandPreviewOptions{
		TaskType: taskType,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load execution forwarding v1 command preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionForwardingV1CommandPreviewResponse(preview))
}

func (s Server) projectExecutionForwardingV1RollbackPreviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	preview, err := s.store.ExecutionForwardingV1RollbackPreview(r.Context(), record, project.ExecutionForwardingV1RollbackPreviewOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load execution forwarding v1 rollback preview failed")
		return
	}
	writeJSON(w, http.StatusOK, buildExecutionForwardingV1RollbackPreviewResponse(preview))
}

func (s Server) projectCutoverReadinessHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	label := strings.TrimSpace(r.URL.Query().Get("version"))
	if label == "" {
		writeError(w, http.StatusBadRequest, "version query parameter is required")
		return
	}
	limit, err := queryLimit(r, 10)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	readiness, err := s.store.ProjectCutoverReadiness(r.Context(), record, label, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load cutover readiness failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectCutoverReadinessResponse(readiness))
}

func (s Server) projectCutoverApplyHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request projectCutoverApplyRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid project cutover apply request")
		return
	}
	result, err := s.store.ApplyCutover(r.Context(), record, project.ApplyCutoverOptions{
		VersionLabel:   request.VersionLabel,
		IdempotencyKey: request.IdempotencyKey,
		Actor:          request.Actor,
		Reason:         request.Reason,
		Mode:           request.Mode,
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "apply project cutover failed")
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
		if result.Decision == "denied" {
			status = http.StatusConflict
		}
	}
	writeJSON(w, status, buildProjectCutoverApplyResponse(result))
}

func (s Server) projectStatusProjectionsHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	projections, err := s.store.ListStatusProjections(r.Context(), record, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list status projections failed")
		return
	}
	writeJSON(w, http.StatusOK, buildStatusProjectionsResponse(record, projections))
}

func (s Server) projectStatusProjectionAuthorizationHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	preview, err := s.store.StatusProjectionAuthorizationPreview(r.Context(), record, project.StatusProjectionAuthorizationPreviewOptions{
		TargetURI: r.URL.Query().Get("target_uri"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "preview status projection authorization failed")
		return
	}
	writeJSON(w, http.StatusOK, buildStatusProjectionAuthorizationPreviewResponse(preview))
}

func statusProjectionApplyGateOptionsFromQuery(r *http.Request) (project.StatusProjectionApplyGateOptions, error) {
	expectedBeforeExists, err := queryOptionalBoolPtr(r, "expected_before_exists")
	if err != nil {
		return project.StatusProjectionApplyGateOptions{}, err
	}
	expectedBeforeSize, err := queryOptionalInt64Ptr(r, "expected_before_size")
	if err != nil {
		return project.StatusProjectionApplyGateOptions{}, err
	}
	explicitApproval, err := queryOptionalBool(r, "explicit_approval")
	if err != nil {
		return project.StatusProjectionApplyGateOptions{}, err
	}
	query := r.URL.Query()
	return project.StatusProjectionApplyGateOptions{
		TargetURI:                      query.Get("target_uri"),
		ExpectedBeforeExists:           expectedBeforeExists,
		ExpectedBeforeSHA256:           query.Get("expected_before_sha256"),
		ExpectedBeforeSizeBytes:        expectedBeforeSize,
		SourceHash:                     query.Get("source_hash"),
		SchemaURI:                      query.Get("schema_uri"),
		ValidatorPreflight:             query.Get("validator_preflight"),
		ProtectedPathCheck:             query.Get("protected_path_check"),
		ProtectedPathFingerprintSHA256: query.Get("protected_path_fingerprint_sha256"),
		RollbackAction:                 query.Get("rollback_action"),
		AcceptedPreimageSchemaStatus:   query.Get("accept_preimage_schema"),
		ExplicitApproval:               explicitApproval,
		ApprovalActor:                  query.Get("approval_actor"),
		ApprovalReason:                 query.Get("approval_reason"),
	}, nil
}

func statusProjectionApplyPacketOptionsFromQuery(r *http.Request) (project.StatusProjectionApplyPacketPreviewOptions, error) {
	explicitApproval, err := queryOptionalBool(r, "explicit_approval")
	if err != nil {
		return project.StatusProjectionApplyPacketPreviewOptions{}, err
	}
	query := r.URL.Query()
	return project.StatusProjectionApplyPacketPreviewOptions{
		TargetURI:        query.Get("target_uri"),
		ExplicitApproval: explicitApproval,
		ApprovalActor:    query.Get("approval_actor"),
		ApprovalReason:   query.Get("approval_reason"),
	}, nil
}

func shimApplyPacketOptionsFromQuery(r *http.Request) (project.ShimApplyPacketPreviewOptions, error) {
	explicitApproval, err := queryOptionalBool(r, "explicit_approval")
	if err != nil {
		return project.ShimApplyPacketPreviewOptions{}, err
	}
	query := r.URL.Query()
	return project.ShimApplyPacketPreviewOptions{
		ExplicitApproval:           explicitApproval,
		ApprovalID:                 query.Get("approval_id"),
		ApprovalActor:              query.Get("approval_actor"),
		ApprovalReason:             query.Get("approval_reason"),
		StatusProjectionPacketID:   query.Get("status_projection_packet_id"),
		StatusProjectionGateID:     query.Get("status_projection_gate_id"),
		ReadOnlySmokeEvidenceID:    query.Get("read_only_smoke_evidence_id"),
		DirtyWorktreeReviewID:      query.Get("dirty_worktree_review_id"),
		ProtectedPathFingerprintID: query.Get("protected_path_fingerprint_id"),
		RollbackPlanID:             query.Get("rollback_plan_id"),
		IdempotencyKey:             query.Get("idempotency_key"),
		AuditCorrelationID:         query.Get("audit_correlation_id"),
	}, nil
}

func shimApplyGateOptionsFromQuery(r *http.Request) (project.ShimApplyGateOptions, error) {
	explicitApproval, err := queryOptionalBool(r, "explicit_approval")
	if err != nil {
		return project.ShimApplyGateOptions{}, err
	}
	query := r.URL.Query()
	return project.ShimApplyGateOptions{
		AllowedFiles:               apiCommaSeparatedList(query.Get("allowed_files")),
		ApprovalID:                 query.Get("approval_id"),
		ApprovalScope:              query.Get("approval_scope"),
		AuthorizationSnapshotHash:  query.Get("authorization_snapshot_hash"),
		ExpectedAuthorizationMode:  query.Get("expected_authorization_mode"),
		StatusProjectionPacketID:   query.Get("status_projection_packet_id"),
		StatusProjectionGateID:     query.Get("status_projection_gate_id"),
		ReadOnlySmokeEvidenceID:    query.Get("read_only_smoke_evidence_id"),
		DirtyWorktreeReviewID:      query.Get("dirty_worktree_review_id"),
		ProtectedPathFingerprintID: query.Get("protected_path_fingerprint_id"),
		RollbackPlanID:             query.Get("rollback_plan_id"),
		IdempotencyKey:             query.Get("idempotency_key"),
		AuditCorrelationID:         query.Get("audit_correlation_id"),
		FailureMode:                query.Get("failure_mode"),
		ExplicitApproval:           explicitApproval,
		ApprovalActor:              query.Get("approval_actor"),
		ApprovalReason:             query.Get("approval_reason"),
	}, nil
}

func executionForwardingV1ApplyPacketOptionsFromQuery(r *http.Request) (project.ExecutionForwardingV1ApplyPacketPreviewOptions, error) {
	explicitApproval, err := queryOptionalBool(r, "explicit_approval")
	if err != nil {
		return project.ExecutionForwardingV1ApplyPacketPreviewOptions{}, err
	}
	query := r.URL.Query()
	return project.ExecutionForwardingV1ApplyPacketPreviewOptions{
		ExplicitApproval:           explicitApproval,
		ApprovalID:                 query.Get("approval_id"),
		ApprovalActor:              query.Get("approval_actor"),
		ApprovalReason:             query.Get("approval_reason"),
		LegacyNonWriteProofID:      query.Get("legacy_non_write_proof_id"),
		RollbackPlanID:             query.Get("rollback_plan_id"),
		ProtectedPathFingerprintID: query.Get("protected_path_fingerprint_id"),
		IdempotencyKey:             query.Get("idempotency_key"),
		AuditCorrelationID:         query.Get("audit_correlation_id"),
	}, nil
}

func executionForwardingV1ApplyGateOptionsFromQuery(r *http.Request) (project.ExecutionForwardingV1ApplyGateOptions, error) {
	explicitApproval, err := queryOptionalBool(r, "explicit_approval")
	if err != nil {
		return project.ExecutionForwardingV1ApplyGateOptions{}, err
	}
	query := r.URL.Query()
	return project.ExecutionForwardingV1ApplyGateOptions{
		AllowedTaskTypes:           apiCommaSeparatedList(query.Get("allowed_task_types")),
		ApprovalID:                 query.Get("approval_id"),
		ApprovalScope:              query.Get("approval_scope"),
		ReadinessSnapshotHash:      query.Get("readiness_snapshot_hash"),
		ExpectedShimLifecycleState: query.Get("expected_shim_lifecycle_state"),
		LegacyNonWriteProofID:      query.Get("legacy_non_write_proof_id"),
		RollbackPlanID:             query.Get("rollback_plan_id"),
		ProtectedPathFingerprintID: query.Get("protected_path_fingerprint_id"),
		IdempotencyKey:             query.Get("idempotency_key"),
		AuditCorrelationID:         query.Get("audit_correlation_id"),
		FailureMode:                query.Get("failure_mode"),
		ExplicitApproval:           explicitApproval,
		ApprovalActor:              query.Get("approval_actor"),
		ApprovalReason:             query.Get("approval_reason"),
	}, nil
}

func apiCommaSeparatedList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (s Server) projectStatusProjectionApplyPacketHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	options, err := statusProjectionApplyPacketOptionsFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, err := s.store.StatusProjectionApplyPacketPreview(r.Context(), record, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "preview status projection apply packet failed")
		return
	}
	writeJSON(w, http.StatusOK, buildStatusProjectionApplyPacketPreviewResponse(preview))
}

func (s Server) projectStatusProjectionApplyGateHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	options, err := statusProjectionApplyGateOptionsFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	gate, err := s.store.StatusProjectionApplyGate(r.Context(), record, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "run status projection apply gate failed")
		return
	}
	writeJSON(w, http.StatusOK, buildStatusProjectionApplyGateResponse(gate))
}

func (s Server) projectExecutionForwardingV1ApplyHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request executionForwardingV1ApplyRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid execution forwarding v1 apply request")
		return
	}
	result, err := s.store.ApplyExecutionForwardingV1(r.Context(), record, project.ApplyExecutionForwardingV1Options{
		IdempotencyKey: request.IdempotencyKey,
		Actor:          request.Actor,
		Reason:         request.Reason,
		Gate: project.ExecutionForwardingV1ApplyGateOptions{
			AllowedTaskTypes:           request.AllowedTaskTypes,
			ApprovalID:                 request.ApprovalID,
			ApprovalScope:              request.ApprovalScope,
			ReadinessSnapshotHash:      request.ReadinessSnapshotHash,
			ExpectedShimLifecycleState: request.ExpectedShimLifecycleState,
			LegacyNonWriteProofID:      request.LegacyNonWriteProofID,
			RollbackPlanID:             request.RollbackPlanID,
			ProtectedPathFingerprintID: request.ProtectedPathFingerprintID,
			IdempotencyKey:             request.IdempotencyKey,
			AuditCorrelationID:         request.AuditCorrelationID,
			FailureMode:                request.FailureMode,
			ExplicitApproval:           request.ExplicitApproval,
			ApprovalActor:              request.ApprovalActor,
			ApprovalReason:             request.ApprovalReason,
		},
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "apply execution forwarding v1 failed")
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
		if result.Decision == "denied" {
			status = http.StatusConflict
		}
	}
	writeJSON(w, status, buildExecutionForwardingV1ApplyResponse(result))
}

func (s Server) projectStatusProjectionApplyHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	var request statusProjectionApplyRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid status projection apply request")
		return
	}
	result, err := s.store.ApplyStatusProjection(r.Context(), record, project.ApplyStatusProjectionOptions{
		TargetURI:      request.TargetURI,
		Actor:          request.Actor,
		Reason:         request.Reason,
		IdempotencyKey: request.IdempotencyKey,
		Writer:         statusProjectionWriter,
		Gate: project.StatusProjectionApplyGateOptions{
			TargetURI:                      request.TargetURI,
			ExpectedBeforeExists:           request.ExpectedBeforeExists,
			ExpectedBeforeSHA256:           request.ExpectedBeforeSHA256,
			ExpectedBeforeSizeBytes:        request.ExpectedBeforeSizeBytes,
			SourceHash:                     request.SourceHash,
			SchemaURI:                      request.SchemaURI,
			ValidatorPreflight:             request.ValidatorPreflight,
			ProtectedPathCheck:             request.ProtectedPathCheck,
			ProtectedPathFingerprintSHA256: request.ProtectedPathFingerprintSHA256,
			RollbackAction:                 request.RollbackAction,
			AcceptedPreimageSchemaStatus:   request.AcceptedPreimageSchemaStatus,
			ExplicitApproval:               request.ExplicitApproval,
			ApprovalActor:                  request.ApprovalActor,
			ApprovalReason:                 request.ApprovalReason,
		},
	})
	if err != nil {
		if errors.Is(err, project.ErrIdempotencyConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "apply status projection failed")
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
		if result.Decision == "denied" {
			status = http.StatusConflict
		}
	}
	writeJSON(w, status, buildStatusProjectionApplyResponse(result))
}

func (s Server) projectArtifactsHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	artifacts, err := s.store.ListProjectArtifacts(r.Context(), record, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list project artifacts failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectArtifactsResponse(record, artifacts))
}

func (s Server) projectResidualsHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}
	residuals, err := s.store.ListProjectResiduals(r.Context(), record, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list project residuals failed")
		return
	}
	writeJSON(w, http.StatusOK, buildProjectResidualsResponse(record, residuals))
}

func (s Server) projectWorkflowVersionsHandler(w http.ResponseWriter, r *http.Request, projectKey string, rest []string) {
	record, err := s.store.GetByKey(r.Context(), projectKey)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("project %s not found", projectKey))
		return
	}

	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			versions, err := s.store.ListWorkflowVersions(r.Context(), record)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "list workflow versions failed")
				return
			}
			writeJSON(w, http.StatusOK, buildWorkflowVersionListResponse(record, versions))
			return
		case http.MethodPost:
			var request createWorkflowVersionRequest
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "invalid workflow version create request")
				return
			}
			result, err := s.store.CreateWorkflowVersion(r.Context(), record, project.CreateWorkflowVersionOptions{
				DisplayLabel:   request.DisplayLabel,
				IdempotencyKey: request.IdempotencyKey,
				Actor:          request.Actor,
				Reason:         request.Reason,
			})
			if err != nil {
				s.writeWorkflowVersionCommandError(w, err)
				return
			}
			status := http.StatusCreated
			if !result.Created {
				status = http.StatusOK
			}
			writeJSON(w, status, buildWorkflowVersionCreateResponse(result))
			return
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
	}

	if len(rest) == 2 && rest[1] == "stages" {
		s.projectWorkflowVersionStagesHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "gates" {
		s.projectWorkflowVersionGatesHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "transition-previews" {
		s.projectWorkflowVersionTransitionPreviewsHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "approvals" {
		s.projectWorkflowVersionApprovalsHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "artifacts" {
		s.projectWorkflowVersionArtifactsHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "residuals" {
		s.projectWorkflowVersionResidualsHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "runner-preview" {
		s.projectWorkflowVersionRunnerPreviewHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "fixture-queue" {
		s.projectWorkflowVersionFixtureQueueHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "read-only-verify-queue" {
		s.projectWorkflowVersionReadOnlyVerifyQueueHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "approved-artifact-write-queue" {
		s.projectWorkflowVersionApprovedArtifactWriteQueueHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "fixture-project-write-queue" {
		s.projectWorkflowVersionFixtureProjectWriteQueueHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "managed-generated-write-queue" {
		s.projectWorkflowVersionManagedGeneratedWriteQueueHandler(w, r, record, rest[0])
		return
	}
	if len(rest) == 2 && rest[1] == "runs" {
		s.projectWorkflowVersionRunsHandler(w, r, record, rest[0])
		return
	}
	if len(rest) != 1 || r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	version, err := s.store.GetWorkflowVersion(r.Context(), record, rest[0])
	if err != nil {
		if errors.Is(err, project.ErrWorkflowVersionNotFound) {
			writeError(w, http.StatusNotFound, "workflow version not found")
			return
		}
		if errors.Is(err, project.ErrInvalidWorkflowVersionLabel) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "load workflow version failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWorkflowVersionResponse(version))
}

func (s Server) projectWorkflowVersionRunnerPreviewHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request runnerPreviewRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid runner preview request")
		return
	}
	result, err := s.store.PreviewRunner(r.Context(), record, label, project.RunnerPreviewOptions{
		Actor:          request.Actor,
		Reason:         request.Reason,
		RiskLevel:      request.RiskLevel,
		RiskPolicy:     request.RiskPolicy,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		s.writeWorkflowVersionCommandError(w, err)
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildRunnerPreviewResponse(result))
}

func (s Server) projectWorkflowVersionFixtureQueueHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request fixtureExecutionQueueRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid fixture execution queue request")
		return
	}
	result, err := s.store.QueueFixtureExecution(r.Context(), record, label, project.FixtureExecutionQueueOptions{
		Actor:          request.Actor,
		Reason:         request.Reason,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		s.writeWorkflowVersionCommandError(w, err)
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildFixtureExecutionQueueResponse(result))
}

func (s Server) projectWorkflowVersionReadOnlyVerifyQueueHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request readOnlyVerifyQueueRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid read-only verify queue request")
		return
	}
	result, err := s.store.QueueReadOnlyVerify(r.Context(), record, label, project.ReadOnlyVerifyQueueOptions{
		TargetPath:     request.TargetPath,
		Actor:          request.Actor,
		Reason:         request.Reason,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		s.writeWorkflowVersionCommandError(w, err)
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildReadOnlyVerifyQueueResponse(result))
}

func (s Server) projectWorkflowVersionApprovedArtifactWriteQueueHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request approvedArtifactWriteQueueRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid approved artifact write queue request")
		return
	}
	result, err := s.store.QueueApprovedArtifactWrite(r.Context(), record, label, project.ApprovedArtifactWriteQueueOptions{
		ArtifactLabel:  request.ArtifactLabel,
		Actor:          request.Actor,
		Reason:         request.Reason,
		IdempotencyKey: request.IdempotencyKey,
	})
	if err != nil {
		s.writeWorkflowVersionCommandError(w, err)
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildApprovedArtifactWriteQueueResponse(result))
}

func (s Server) projectWorkflowVersionFixtureProjectWriteQueueHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request fixtureProjectWriteQueueRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid fixture project write queue request")
		return
	}
	result, err := s.store.QueueFixtureProjectWrite(r.Context(), record, label, project.FixtureProjectWriteQueueOptions{
		TargetPath:           request.TargetPath,
		Content:              request.Content,
		ExpectedBeforeSHA256: request.ExpectedBeforeSHA256,
		ExpectedBeforeSize:   request.ExpectedBeforeSize,
		Actor:                request.Actor,
		Reason:               request.Reason,
		IdempotencyKey:       request.IdempotencyKey,
	})
	if err != nil {
		s.writeWorkflowVersionCommandError(w, err)
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildFixtureProjectWriteQueueResponse(result))
}

func (s Server) projectWorkflowVersionManagedGeneratedWriteQueueHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request managedGeneratedWriteQueueRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid managed generated write queue request")
		return
	}
	result, err := s.store.QueueManagedGeneratedWrite(r.Context(), record, label, project.ManagedGeneratedWriteQueueOptions{
		TargetPath:           request.TargetPath,
		Content:              request.Content,
		ExpectedBeforeSHA256: request.ExpectedBeforeSHA256,
		ExpectedBeforeSize:   request.ExpectedBeforeSize,
		Actor:                request.Actor,
		Reason:               request.Reason,
		IdempotencyKey:       request.IdempotencyKey,
	})
	if err != nil {
		s.writeWorkflowVersionCommandError(w, err)
		return
	}
	status := http.StatusCreated
	if !result.Created {
		status = http.StatusOK
	}
	writeJSON(w, status, buildManagedGeneratedWriteQueueResponse(result))
}

func (s Server) projectWorkflowVersionRunsHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		if errors.Is(err, project.ErrWorkflowVersionNotFound) {
			writeError(w, http.StatusNotFound, "workflow version not found")
			return
		}
		if errors.Is(err, project.ErrInvalidWorkflowVersionLabel) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "load workflow version failed")
		return
	}
	runs, err := s.store.ListWorkflowVersionRuns(r.Context(), record, version, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list workflow version runs failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWorkflowVersionRunsResponse(record, version, runs))
}

func (s Server) projectWorkflowVersionGatesHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		if errors.Is(err, project.ErrWorkflowVersionNotFound) {
			writeError(w, http.StatusNotFound, "workflow version not found")
			return
		}
		if errors.Is(err, project.ErrInvalidWorkflowVersionLabel) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "load workflow version failed")
		return
	}

	switch r.Method {
	case http.MethodGet:
		limit, err := queryLimit(r, 10)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		results, err := s.store.ListGateResults(r.Context(), record, version, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list gate results failed")
			return
		}
		writeJSON(w, http.StatusOK, buildGateResultsResponse(record, version, results))
	case http.MethodPost:
		var request runGateRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid gate request")
			return
		}
		if request.GateName == "" {
			request.GateName = "discussion_gate"
		}
		result, err := s.store.RunWorkflowGate(r.Context(), record, label, project.RunGateOptions{
			GateName: request.GateName,
			Actor:    request.Actor,
			Reason:   request.Reason,
		})
		if err != nil {
			s.writeWorkflowVersionCommandError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildGateResultResponse(result))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) projectWorkflowVersionTransitionPreviewsHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		s.writeWorkflowVersionLookupError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		limit, err := queryLimit(r, 10)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		previews, err := s.store.ListWorkflowTransitionPreviews(r.Context(), record, version, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list transition previews failed")
			return
		}
		writeJSON(w, http.StatusOK, buildTransitionPreviewsResponse(record, version, previews))
	case http.MethodPost:
		var request previewTransitionRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid transition preview request")
			return
		}
		preview, err := s.store.PreviewWorkflowTransition(r.Context(), record, label, project.PreviewTransitionOptions{
			FromStage: request.FromStage,
			ToStage:   request.ToStage,
			Actor:     request.Actor,
			Reason:    request.Reason,
		})
		if err != nil {
			s.writeWorkflowVersionCommandError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildTransitionPreviewResponse(preview))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) projectWorkflowVersionApprovalsHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		s.writeWorkflowVersionLookupError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		limit, err := queryLimit(r, 10)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		approvals, err := s.store.ListApprovalRecords(r.Context(), record, version, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list approval records failed")
			return
		}
		writeJSON(w, http.StatusOK, buildApprovalRecordsResponse(record, version, approvals))
	case http.MethodPost:
		var request createApprovalRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid approval request")
			return
		}
		approval, err := s.store.CreateApprovalRecord(r.Context(), record, label, project.CreateApprovalOptions{
			Decision:            request.Decision,
			ApprovalKind:        request.ApprovalKind,
			Actor:               request.Actor,
			Reason:              request.Reason,
			RiskLevel:           request.RiskLevel,
			IdempotencyKey:      request.IdempotencyKey,
			TransitionPreviewID: request.TransitionPreviewID,
			Metadata:            request.Metadata,
		})
		if err != nil {
			s.writeWorkflowVersionCommandError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildApprovalRecordResponse(approval))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) projectWorkflowVersionArtifactsHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		s.writeWorkflowVersionLookupError(w, err)
		return
	}
	artifacts, err := s.store.ListWorkflowVersionArtifacts(r.Context(), record, version, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list workflow version artifacts failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWorkflowVersionArtifactsResponse(record, version, artifacts))
}

func (s Server) projectWorkflowVersionResidualsHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit, err := queryLimit(r, 20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		s.writeWorkflowVersionLookupError(w, err)
		return
	}
	residuals, err := s.store.ListWorkflowVersionResiduals(r.Context(), record, version, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list workflow version residuals failed")
		return
	}
	writeJSON(w, http.StatusOK, buildWorkflowVersionResidualsResponse(record, version, residuals))
}

func (s Server) projectWorkflowVersionStagesHandler(w http.ResponseWriter, r *http.Request, record project.Record, label string) {
	version, err := s.store.GetWorkflowVersion(r.Context(), record, label)
	if err != nil {
		s.writeWorkflowVersionLookupError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListWorkflowItems(r.Context(), record, version)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list workflow version stages failed")
			return
		}
		links, err := s.store.ListWorkflowItemLinks(r.Context(), record, version, 100)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list workflow item links failed")
			return
		}
		writeJSON(w, http.StatusOK, buildWorkflowVersionStagesResponse(record, version, items, links))
	case http.MethodPost:
		var request ensureStageSkeletonRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if r.Body != nil {
			if err := decoder.Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "invalid stage skeleton request")
				return
			}
		}
		result, err := s.store.EnsureStageSkeleton(r.Context(), record, label, project.EnsureStageSkeletonOptions{
			Actor:  request.Actor,
			Reason: request.Reason,
		})
		if err != nil {
			s.writeWorkflowVersionCommandError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, buildEnsureStageSkeletonResponse(result))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s Server) writeWorkflowVersionLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, project.ErrWorkflowVersionNotFound) {
		writeError(w, http.StatusNotFound, "workflow version not found")
		return
	}
	if errors.Is(err, project.ErrInvalidWorkflowVersionLabel) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "load workflow version failed")
}

func (s Server) writeWorkflowVersionCommandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, project.ErrInvalidWorkflowVersionLabel):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, project.ErrWorkflowVersionExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, project.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, project.ErrWorkflowVersionNotAuthored):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, project.ErrUnsupportedWorkflowGate):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, project.ErrInvalidApprovalDecision):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, project.ErrApprovalPreviewNotReady):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, project.ErrRunnerPreviewBlocked):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "workflow version command failed")
	}
}

func buildWorkflowVersionListResponse(record project.Record, versions []project.WorkflowVersion) workflowVersionListResponse {
	response := workflowVersionListResponse{
		Project:          buildProjectRecordResponse(record),
		WorkflowVersions: make([]workflowVersionResponse, 0, len(versions)),
	}
	for _, version := range versions {
		response.WorkflowVersions = append(response.WorkflowVersions, buildWorkflowVersionResponse(version))
	}
	return response
}

func buildWorkflowVersionCreateResponse(result project.CreateWorkflowVersionResult) workflowVersionCreateResponse {
	response := workflowVersionCreateResponse{
		Project:         buildProjectRecordResponse(result.Project),
		WorkflowVersion: buildWorkflowVersionResponse(result.Version),
		InitialItem:     buildWorkflowItemResponse(result.InitialItem),
		StageItems:      make([]workflowItemResponse, 0, len(result.StageItems)),
		Created:         result.Created,
		IdempotencyKey:  result.IdempotencyKey,
	}
	for _, item := range result.StageItems {
		response.StageItems = append(response.StageItems, buildWorkflowItemResponse(item))
	}
	return response
}

func buildWorkflowVersionStagesResponse(record project.Record, version project.WorkflowVersion, items []project.WorkflowItem, links []project.WorkflowItemLink) workflowVersionStagesResponse {
	response := workflowVersionStagesResponse{
		Project:         buildProjectRecordResponse(record),
		WorkflowVersion: buildWorkflowVersionResponse(version),
		Items:           make([]workflowItemResponse, 0, len(items)),
		Links:           make([]workflowItemLinkResponse, 0, len(links)),
	}
	for _, item := range items {
		response.Items = append(response.Items, buildWorkflowItemResponse(item))
	}
	for _, link := range links {
		response.Links = append(response.Links, buildWorkflowItemLinkResponse(link))
	}
	return response
}

func buildEnsureStageSkeletonResponse(result project.EnsureStageSkeletonResult) ensureStageSkeletonResponse {
	response := ensureStageSkeletonResponse{
		Project:         buildProjectRecordResponse(result.Project),
		WorkflowVersion: buildWorkflowVersionResponse(result.Version),
		Items:           make([]workflowItemResponse, 0, len(result.Items)),
		Links:           make([]workflowItemLinkResponse, 0, len(result.Links)),
		Created:         result.Created,
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, buildWorkflowItemResponse(item))
	}
	for _, link := range result.Links {
		response.Links = append(response.Links, buildWorkflowItemLinkResponse(link))
	}
	return response
}

func buildGateResultsResponse(record project.Record, version project.WorkflowVersion, results []project.GateResult) gateResultsResponse {
	response := gateResultsResponse{
		Project:         buildProjectRecordResponse(record),
		WorkflowVersion: buildWorkflowVersionResponse(version),
		GateResults:     make([]gateResultResponse, 0, len(results)),
	}
	for _, result := range results {
		response.GateResults = append(response.GateResults, buildGateResultResponse(result))
	}
	return response
}

func buildGateResultResponse(result project.GateResult) gateResultResponse {
	return gateResultResponse{
		ID:                  result.ID,
		GateName:            result.GateName,
		ScopeType:           result.ScopeType,
		ScopeID:             result.ScopeID,
		Status:              result.Status,
		WorkflowVersionID:   result.WorkflowVersionID,
		WorkflowItemID:      result.WorkflowItemID,
		Inputs:              result.Inputs,
		SourceHashes:        result.SourceHashes,
		Failures:            result.Failures,
		Warnings:            result.Warnings,
		EvidenceArtifactIDs: result.EvidenceArtifactIDs,
		Metadata:            result.Metadata,
		CheckedAt:           formatTime(result.CheckedAt),
	}
}

func buildTransitionPreviewsResponse(record project.Record, version project.WorkflowVersion, previews []project.WorkflowTransitionPreview) transitionPreviewsResponse {
	response := transitionPreviewsResponse{
		Project:            buildProjectRecordResponse(record),
		WorkflowVersion:    buildWorkflowVersionResponse(version),
		TransitionPreviews: make([]transitionPreviewResponse, 0, len(previews)),
	}
	for _, preview := range previews {
		response.TransitionPreviews = append(response.TransitionPreviews, buildTransitionPreviewResponse(preview))
	}
	return response
}

func buildTransitionPreviewResponse(preview project.WorkflowTransitionPreview) transitionPreviewResponse {
	return transitionPreviewResponse{
		ID:                preview.ID,
		WorkflowVersionID: preview.WorkflowVersionID,
		FromStage:         preview.FromStage,
		ToStage:           preview.ToStage,
		Status:            preview.Status,
		RequiredGateName:  preview.RequiredGateName,
		GateResultID:      preview.GateResultID,
		Blockers:          preview.Blockers,
		Warnings:          preview.Warnings,
		Metadata:          preview.Metadata,
		CreatedAt:         formatTime(preview.CreatedAt),
	}
}

func buildApprovalRecordsResponse(record project.Record, version project.WorkflowVersion, approvals []project.ApprovalRecord) approvalRecordsResponse {
	response := approvalRecordsResponse{
		Project:         buildProjectRecordResponse(record),
		WorkflowVersion: buildWorkflowVersionResponse(version),
		ApprovalRecords: make([]approvalRecordResponse, 0, len(approvals)),
	}
	for _, approval := range approvals {
		response.ApprovalRecords = append(response.ApprovalRecords, buildApprovalRecordResponse(approval))
	}
	return response
}

func buildApprovalRecordResponse(approval project.ApprovalRecord) approvalRecordResponse {
	return approvalRecordResponse{
		ID:                  approval.ID,
		WorkflowVersionID:   approval.WorkflowVersionID,
		TransitionPreviewID: approval.TransitionPreviewID,
		ApprovalKind:        approval.ApprovalKind,
		Decision:            approval.Decision,
		ScopeType:           approval.ScopeType,
		ScopeID:             approval.ScopeID,
		Actor:               approval.Actor,
		Reason:              approval.Reason,
		RiskLevel:           approval.RiskLevel,
		Metadata:            approval.Metadata,
		CreatedAt:           formatTime(approval.CreatedAt),
	}
}

func buildRunnerPreviewResponse(result project.RunnerPreviewResult) runnerPreviewResponse {
	response := runnerPreviewResponse{
		Project:         buildProjectRecordResponse(result.Project),
		WorkflowVersion: buildWorkflowVersionResponse(result.Version),
		Run:             buildRunResponse(result.Run),
		Tasks:           make([]runTaskResponse, 0, len(result.Tasks)),
		Attempts:        make([]runAttemptResponse, 0, len(result.Attempts)),
		Artifacts:       make([]artifactResponse, 0, len(result.Artifacts)),
		Preflight:       buildRunnerPreflightResponse(result.Preflight),
		Created:         result.Created,
		IdempotencyKey:  result.IdempotencyKey,
	}
	for _, task := range result.Tasks {
		response.Tasks = append(response.Tasks, buildRunTaskResponse(task))
	}
	for _, attempt := range result.Attempts {
		response.Attempts = append(response.Attempts, buildRunAttemptResponse(attempt))
	}
	for _, artifact := range result.Artifacts {
		response.Artifacts = append(response.Artifacts, buildArtifactResponse(artifact))
	}
	return response
}

func buildFixtureExecutionQueueResponse(result project.FixtureExecutionQueueResult) fixtureExecutionQueueResponse {
	return fixtureExecutionQueueResponse{
		Project:                 buildProjectRecordResponse(result.Project),
		WorkflowVersion:         buildWorkflowVersionResponse(result.Version),
		Run:                     buildRunResponse(result.Run),
		Task:                    buildRunTaskResponse(result.Task),
		Created:                 result.Created,
		IdempotencyKey:          result.IdempotencyKey,
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		ProjectWriteAttempted:   result.ProjectWriteAttempted,
		ExecutionWriteAttempted: result.ExecutionWriteAttempted,
		EngineCallAttempted:     result.EngineCallAttempted,
		CommandsRun:             result.CommandsRun,
		SecretsResolved:         result.SecretsResolved,
		NetworkUsed:             result.NetworkUsed,
	}
}

func buildReadOnlyVerifyQueueResponse(result project.ReadOnlyVerifyQueueResult) readOnlyVerifyQueueResponse {
	return readOnlyVerifyQueueResponse{
		Project:                 buildProjectRecordResponse(result.Project),
		WorkflowVersion:         buildWorkflowVersionResponse(result.Version),
		Run:                     buildRunResponse(result.Run),
		Task:                    buildRunTaskResponse(result.Task),
		TargetPath:              result.TargetPath,
		Created:                 result.Created,
		IdempotencyKey:          result.IdempotencyKey,
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		ProjectReadAttempted:    result.ProjectReadAttempted,
		ProjectReadAllowed:      result.ProjectReadAllowed,
		ProjectWriteAttempted:   result.ProjectWriteAttempted,
		ExecutionWriteAttempted: result.ExecutionWriteAttempted,
		EngineCallAttempted:     result.EngineCallAttempted,
		CommandsRun:             result.CommandsRun,
		SecretsResolved:         result.SecretsResolved,
		NetworkUsed:             result.NetworkUsed,
	}
}

func buildApprovedArtifactWriteQueueResponse(result project.ApprovedArtifactWriteQueueResult) approvedArtifactWriteQueueResponse {
	return approvedArtifactWriteQueueResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Task:                          buildRunTaskResponse(result.Task),
		ArtifactLabel:                 result.ArtifactLabel,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       result.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
	}
}

func buildFixtureProjectWriteQueueResponse(result project.FixtureProjectWriteQueueResult) fixtureProjectWriteQueueResponse {
	return fixtureProjectWriteQueueResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Task:                          buildRunTaskResponse(result.Task),
		WriteSetArtifact:              buildArtifactResponse(result.WriteSetArtifact),
		TargetPath:                    result.TargetPath,
		ExpectedBeforeSHA256:          result.ExpectedBeforeSHA256,
		ExpectedBeforeSize:            result.ExpectedBeforeSize,
		AfterSHA256:                   result.AfterSHA256,
		AfterSize:                     result.AfterSize,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       result.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
	}
}

func buildManagedGeneratedWriteQueueResponse(result project.ManagedGeneratedWriteQueueResult) managedGeneratedWriteQueueResponse {
	return managedGeneratedWriteQueueResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Task:                          buildRunTaskResponse(result.Task),
		WriteSetArtifact:              buildArtifactResponse(result.WriteSetArtifact),
		TargetPath:                    result.TargetPath,
		ExpectedBeforeSHA256:          result.ExpectedBeforeSHA256,
		ExpectedBeforeSize:            result.ExpectedBeforeSize,
		AfterSHA256:                   result.AfterSHA256,
		AfterSize:                     result.AfterSize,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		GeneratedOnly:                 result.GeneratedOnly,
		GeneratedOnlyApplyOpen:        result.GeneratedOnlyApplyOpen,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       result.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
	}
}

func buildRunDetailResponse(detail project.RunDetail) runDetailResponse {
	response := runDetailResponse{
		Run:       buildRunResponse(detail.Run),
		Tasks:     make([]runTaskResponse, 0, len(detail.Tasks)),
		Attempts:  make([]runAttemptResponse, 0, len(detail.Attempts)),
		Artifacts: make([]artifactResponse, 0, len(detail.Artifacts)),
	}
	for _, task := range detail.Tasks {
		response.Tasks = append(response.Tasks, buildRunTaskResponse(task))
	}
	for _, attempt := range detail.Attempts {
		response.Attempts = append(response.Attempts, buildRunAttemptResponse(attempt))
	}
	for _, artifact := range detail.Artifacts {
		response.Artifacts = append(response.Artifacts, buildArtifactResponse(artifact))
	}
	return response
}

func buildRunControlResponse(result project.RunControlResult) runControlResponse {
	return runControlResponse{
		Project:                  buildProjectRecordResponse(result.Project),
		Run:                      buildRunResponse(result.Run),
		PreviousStatus:           result.PreviousStatus,
		Status:                   result.Status,
		Decision:                 result.Decision,
		Message:                  result.Message,
		Blockers:                 result.Blockers,
		EventID:                  result.EventID,
		AuditEventID:             result.AuditEventID,
		IdempotencyKey:           result.IdempotencyKey,
		Created:                  result.Created,
		ProjectWriteAttempted:    result.ProjectWriteAttempted,
		ExecutionWriteAttempted:  result.ExecutionWriteAttempted,
		AreaMatrixWriteAttempted: result.AreaMatrixWriteAttempted,
		EngineCallAttempted:      result.EngineCallAttempted,
	}
}

func buildExecutionApprovalGateResponse(gate project.ExecutionApprovalGate) executionApprovalGateResponse {
	response := executionApprovalGateResponse{
		Project:                 buildProjectRecordResponse(gate.Project),
		WorkflowVersion:         buildWorkflowVersionResponse(gate.Version),
		Run:                     buildRunResponse(gate.Run),
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Items:                   make([]projectReadinessItemResponse, 0, len(gate.Items)),
		Blockers:                gate.Blockers,
		Warnings:                gate.Warnings,
		RequiredCapabilities:    gate.RequiredCapabilities,
		ApprovalFound:           gate.ApprovalFound,
		ApprovalGateFound:       gate.ApprovalGateFound,
		LiveMappingGateFound:    gate.LiveMappingGateFound,
		EnginePreview:           buildCodexCLIAdapterPreviewResponse(gate.EnginePreview),
		Workers:                 make([]workerResponse, 0, len(gate.Workers)),
		ForbiddenActions:        gate.ForbiddenActions,
		ProjectWriteAttempted:   gate.ProjectWriteAttempted,
		ExecutionWriteAttempted: gate.ExecutionWriteAttempted,
		EngineCallAttempted:     gate.EngineCallAttempted,
		CommandsRun:             gate.CommandsRun,
		SecretsResolved:         gate.SecretsResolved,
		NetworkUsed:             gate.NetworkUsed,
		TaskClaimed:             gate.TaskClaimed,
		WorkerStarted:           gate.WorkerStarted,
		AttemptCreated:          gate.AttemptCreated,
		ArtifactCreated:         gate.ArtifactCreated,
		GeneratedAt:             formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	if gate.ApprovalFound {
		response.Approval = buildApprovalRecordResponse(gate.Approval)
	}
	if gate.ApprovalGateFound {
		response.ApprovalGate = buildGateResultResponse(gate.ApprovalGate)
	}
	if gate.LiveMappingGateFound {
		response.LiveMappingGate = buildGateResultResponse(gate.LiveMappingGate)
	}
	for _, worker := range gate.Workers {
		response.Workers = append(response.Workers, buildWorkerResponse(worker))
	}
	return response
}

func buildExecutionPlanPreviewResponse(preview project.ExecutionPlanPreview) executionPlanPreviewResponse {
	response := executionPlanPreviewResponse{
		Project:                       buildProjectRecordResponse(preview.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(preview.Version),
		Run:                           buildRunResponse(preview.Run),
		Gate:                          buildExecutionApprovalGateResponse(preview.Gate),
		Status:                        preview.Status,
		Mode:                          preview.Mode,
		Steps:                         make([]executionPlanStepResponse, 0, len(preview.Steps)),
		Blockers:                      preview.Blockers,
		ForbiddenActions:              preview.ForbiddenActions,
		ProjectReadAttempted:          preview.ProjectReadAttempted,
		ProjectWriteAttempted:         preview.ProjectWriteAttempted,
		ExecutionWriteAttempted:       preview.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       preview.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: preview.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           preview.EngineCallAttempted,
		CommandsRun:                   preview.CommandsRun,
		SecretsResolved:               preview.SecretsResolved,
		NetworkUsed:                   preview.NetworkUsed,
		TaskClaimed:                   preview.TaskClaimed,
		WorkerStarted:                 preview.WorkerStarted,
		AttemptCreated:                preview.AttemptCreated,
		ArtifactCreated:               preview.ArtifactCreated,
		GeneratedAt:                   formatTime(preview.GeneratedAt),
	}
	for _, step := range preview.Steps {
		response.Steps = append(response.Steps, buildExecutionPlanStepResponse(step))
	}
	return response
}

func buildExecutionPlanStepResponse(step project.ExecutionPlanStep) executionPlanStepResponse {
	return executionPlanStepResponse{
		Key:                  step.Key,
		AttemptKind:          step.AttemptKind,
		Status:               step.Status,
		Message:              step.Message,
		RequiredCapabilities: jsonStringSlice(step.RequiredCapabilities),
		Prerequisites:        jsonStringSlice(step.Prerequisites),
		Blockers:             jsonStringSlice(step.Blockers),
		ReadsProject:         step.ReadsProject,
		WritesProject:        step.WritesProject,
		WritesAreaFlow:       step.WritesAreaFlow,
		UsesEngine:           step.UsesEngine,
		RunsCommands:         step.RunsCommands,
		UsesSecrets:          step.UsesSecrets,
		UsesNetwork:          step.UsesNetwork,
		CreatesAttempt:       step.CreatesAttempt,
		CreatesArtifact:      step.CreatesArtifact,
		Metadata:             jsonObject(step.Metadata),
	}
}

func buildProjectWriteDesignGateResponse(gate project.ProjectWriteDesignGate) projectWriteDesignGateResponse {
	response := projectWriteDesignGateResponse{
		Project:                       buildProjectRecordResponse(gate.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(gate.Version),
		Run:                           buildRunResponse(gate.Run),
		Gate:                          buildExecutionApprovalGateResponse(gate.Gate),
		Status:                        gate.Status,
		Mode:                          gate.Mode,
		Items:                         make([]projectReadinessItemResponse, 0, len(gate.Items)),
		RequiredCapabilities:          jsonStringSlice(gate.RequiredCapabilities),
		WriteSetFields:                jsonStringSlice(gate.WriteSetFields),
		UnsupportedOperations:         jsonStringSlice(gate.UnsupportedOperations),
		ApplySequence:                 jsonStringSlice(gate.ApplySequence),
		Blockers:                      jsonStringSlice(gate.Blockers),
		ForbiddenActions:              jsonStringSlice(gate.ForbiddenActions),
		ProjectWriteApplyOpen:         gate.ProjectWriteApplyOpen,
		ProjectReadAttempted:          gate.ProjectReadAttempted,
		ProjectWriteAttempted:         gate.ProjectWriteAttempted,
		ExecutionWriteAttempted:       gate.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       gate.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: gate.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           gate.EngineCallAttempted,
		CommandsRun:                   gate.CommandsRun,
		SecretsResolved:               gate.SecretsResolved,
		NetworkUsed:                   gate.NetworkUsed,
		TaskClaimed:                   gate.TaskClaimed,
		WorkerStarted:                 gate.WorkerStarted,
		AttemptCreated:                gate.AttemptCreated,
		ArtifactCreated:               gate.ArtifactCreated,
		GeneratedAt:                   formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: jsonObject(item.Metadata),
		})
	}
	return response
}

func buildWorkflowVersionRunsResponse(record project.Record, version project.WorkflowVersion, runs []project.RunRecord) workflowVersionRunsResponse {
	response := workflowVersionRunsResponse{
		Project:         buildProjectRecordResponse(record),
		WorkflowVersion: buildWorkflowVersionResponse(version),
		Runs:            make([]runResponse, 0, len(runs)),
	}
	for _, run := range runs {
		response.Runs = append(response.Runs, buildRunResponse(run))
	}
	return response
}

func buildManagedGeneratedWriteGateResponse(gate project.ManagedGeneratedWriteGate) managedGeneratedWriteGateResponse {
	response := managedGeneratedWriteGateResponse{
		Project:                       buildProjectRecordResponse(gate.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(gate.Version),
		Run:                           buildRunResponse(gate.Run),
		Gate:                          buildExecutionApprovalGateResponse(gate.Gate),
		Status:                        gate.Status,
		Mode:                          gate.Mode,
		Items:                         make([]projectReadinessItemResponse, 0, len(gate.Items)),
		RequiredCapabilities:          jsonStringSlice(gate.RequiredCapabilities),
		AllowedGeneratedPrefixes:      jsonStringSlice(gate.AllowedGeneratedPrefixes),
		RequiredWriteSetFields:        jsonStringSlice(gate.RequiredWriteSetFields),
		UnsupportedOperations:         jsonStringSlice(gate.UnsupportedOperations),
		ApplySequence:                 jsonStringSlice(gate.ApplySequence),
		Blockers:                      jsonStringSlice(gate.Blockers),
		ForbiddenActions:              jsonStringSlice(gate.ForbiddenActions),
		GeneratedOnlyWriteReady:       gate.GeneratedOnlyWriteReady,
		GeneratedOnlyApplyOpen:        gate.GeneratedOnlyApplyOpen,
		ProjectReadAttempted:          gate.ProjectReadAttempted,
		ProjectWriteAttempted:         gate.ProjectWriteAttempted,
		ExecutionWriteAttempted:       gate.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       gate.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: gate.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           gate.EngineCallAttempted,
		CommandsRun:                   gate.CommandsRun,
		SecretsResolved:               gate.SecretsResolved,
		NetworkUsed:                   gate.NetworkUsed,
		TaskClaimed:                   gate.TaskClaimed,
		WorkerStarted:                 gate.WorkerStarted,
		LeaseCreated:                  gate.LeaseCreated,
		AttemptCreated:                gate.AttemptCreated,
		ArtifactCreated:               gate.ArtifactCreated,
		GeneratedAt:                   formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: jsonObject(item.Metadata),
		})
	}
	return response
}

func buildRunEventsResponse(runID int64, events []project.EventRecord) runEventsResponse {
	response := runEventsResponse{
		RunID:  runID,
		Events: make([]eventResponse, 0, len(events)),
	}
	for _, event := range events {
		response.Events = append(response.Events, eventResponse{
			ID:        event.ID,
			Type:      event.Type,
			Severity:  event.Severity,
			Message:   event.Message,
			Metadata:  event.Metadata,
			CreatedAt: formatTime(event.CreatedAt),
		})
	}
	return response
}

func buildAuditEventsResponse(projectKey string, events []project.AuditEventRecord) auditEventsResponse {
	response := auditEventsResponse{
		ProjectKey:  projectKey,
		AuditEvents: make([]auditEventResponse, 0, len(events)),
	}
	for _, event := range events {
		response.AuditEvents = append(response.AuditEvents, buildAuditEventResponse(event))
	}
	return response
}

func buildAuditEventResponse(event project.AuditEventRecord) auditEventResponse {
	return auditEventResponse{
		ID:           event.ID,
		ProjectID:    event.ProjectID,
		ActorID:      event.ActorID,
		Action:       event.Action,
		Capability:   event.Capability,
		ResourceType: event.ResourceType,
		Resource:     event.Resource,
		Decision:     event.Decision,
		Reason:       event.Reason,
		Metadata:     event.Metadata,
		CreatedAt:    formatTime(event.CreatedAt),
	}
}

func buildAuditCoverageResponse(coverage project.AuditCoverage) auditCoverageResponse {
	response := auditCoverageResponse{
		Status:              coverage.Status,
		Mode:                coverage.Mode,
		Scope:               coverage.Scope,
		ProjectID:           coverage.ProjectID,
		ProjectKey:          coverage.ProjectKey,
		TotalAuditEvents:    coverage.TotalAuditEvents,
		CoveredRequirements: coverage.CoveredRequirements,
		GapRequirements:     coverage.GapRequirements,
		Requirements:        make([]auditCoverageRequirementResponse, 0, len(coverage.Requirements)),
		GeneratedAt:         formatTime(coverage.GeneratedAt),
	}
	for _, requirement := range coverage.Requirements {
		item := auditCoverageRequirementResponse{
			Key:             requirement.Key,
			Category:        requirement.Category,
			Description:     requirement.Description,
			Status:          requirement.Status,
			EvidenceCount:   requirement.EvidenceCount,
			RequiredActions: make([]auditCoverageActionEvidenceResponse, 0, len(requirement.RequiredActions)),
			MissingActions:  jsonStringSlice(requirement.MissingActions),
		}
		if requirement.LastAuditAt != nil {
			item.LastAuditAt = formatTime(*requirement.LastAuditAt)
		}
		for _, action := range requirement.RequiredActions {
			actionItem := auditCoverageActionEvidenceResponse{
				Action:   action.Action,
				Decision: action.Decision,
				Count:    action.Count,
				Status:   action.Status,
			}
			if action.LastAuditAt != nil {
				actionItem.LastAuditAt = formatTime(*action.LastAuditAt)
			}
			item.RequiredActions = append(item.RequiredActions, actionItem)
		}
		response.Requirements = append(response.Requirements, item)
	}
	return response
}

func buildPermissionPolicyDoctorResponse(doctor project.PermissionPolicyDoctor) permissionPolicyDoctorResponse {
	response := permissionPolicyDoctorResponse{
		Status:      doctor.Status,
		Mode:        doctor.Mode,
		Project:     buildProjectRecordResponse(doctor.Project),
		Checks:      make([]permissionPolicyCheckResponse, 0, len(doctor.Checks)),
		GeneratedAt: formatTime(doctor.GeneratedAt),
	}
	for _, check := range doctor.Checks {
		response.Checks = append(response.Checks, permissionPolicyCheckResponse{
			Key:      check.Key,
			Category: check.Category,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return response
}

func buildArtifactIntegrityResponse(report project.ArtifactIntegrityReport) artifactIntegrityResponse {
	response := artifactIntegrityResponse{
		Status:           report.Status,
		Mode:             report.Mode,
		Project:          buildProjectRecordResponse(report.Project),
		CheckedArtifacts: report.CheckedArtifacts,
		PassedArtifacts:  report.PassedArtifacts,
		WarnArtifacts:    report.WarnArtifacts,
		FailedArtifacts:  report.FailedArtifacts,
		SkippedArtifacts: report.SkippedArtifacts,
		Checks:           make([]artifactIntegrityCheckResponse, 0, len(report.Checks)),
		GeneratedAt:      formatTime(report.GeneratedAt),
	}
	for _, check := range report.Checks {
		response.Checks = append(response.Checks, artifactIntegrityCheckResponse{
			Artifact: buildArtifactResponse(check.Artifact),
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return response
}

func buildArtifactArchivePreviewResponse(result project.ArtifactArchivePreviewResult) artifactArchivePreviewResponse {
	response := artifactArchivePreviewResponse{
		Project: buildProjectRecordResponse(result.Project),
		Status:  result.Status,
		Mode:    result.Mode,
		Summary: artifactArchivePreviewSummaryResponse{
			TotalArtifacts:    result.Summary.TotalArtifacts,
			ArchiveCandidates: result.Summary.ArchiveCandidates,
			RetainedArtifacts: result.Summary.RetainedArtifacts,
			ExternalRefs:      result.Summary.ExternalRefs,
			NeedsPolicy:       result.Summary.NeedsPolicy,
		},
		Items:                   make([]artifactArchivePreviewItemResponse, 0, len(result.Items)),
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		IdempotencyKey:          result.IdempotencyKey,
		Created:                 result.Created,
		GeneratedAt:             formatTime(result.GeneratedAt),
		ProjectWriteAttempted:   result.ProjectWriteAttempted,
		StorageWriteAttempted:   result.StorageWriteAttempted,
		ArtifactDeleteAttempted: result.ArtifactDeleteAttempted,
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, artifactArchivePreviewItemResponse{
			ArtifactID:     item.ArtifactID,
			ArtifactType:   item.ArtifactType,
			StorageBackend: item.StorageBackend,
			URI:            item.URI,
			SourcePath:     item.SourcePath,
			RetentionClass: item.RetentionClass,
			ArchiveState:   item.ArchiveState,
			Action:         item.Action,
			Decision:       item.Decision,
			Reason:         item.Reason,
			Metadata:       item.Metadata,
		})
	}
	return response
}

func buildConformanceResponse(report project.ConformanceReport) conformanceResponse {
	response := conformanceResponse{
		Status:      report.Status,
		Mode:        report.Mode,
		Project:     buildProjectRecordResponse(report.Project),
		ProfileID:   report.ProfileID,
		Adapter:     report.Adapter,
		ProfileHash: report.ProfileHash,
		StageCount:  report.StageCount,
		GateCount:   report.GateCount,
		Checks:      make([]conformanceCheckResponse, 0, len(report.Checks)),
		GeneratedAt: formatTime(report.GeneratedAt),
	}
	for _, check := range report.Checks {
		response.Checks = append(response.Checks, conformanceCheckResponse{
			Key:      check.Key,
			Category: check.Category,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return response
}

func buildProjectArtifactsResponse(record project.Record, artifacts []project.ArtifactRecord) projectArtifactsResponse {
	response := projectArtifactsResponse{
		Project:   buildProjectRecordResponse(record),
		Artifacts: make([]artifactResponse, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		response.Artifacts = append(response.Artifacts, buildArtifactResponse(artifact))
	}
	return response
}

func buildWorkflowVersionArtifactsResponse(record project.Record, version project.WorkflowVersion, artifacts []project.ArtifactRecord) workflowVersionArtifactsResponse {
	response := workflowVersionArtifactsResponse{
		Project:         buildProjectRecordResponse(record),
		WorkflowVersion: buildWorkflowVersionResponse(version),
		Artifacts:       make([]artifactResponse, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		response.Artifacts = append(response.Artifacts, buildArtifactResponse(artifact))
	}
	return response
}

func buildProjectResidualsResponse(record project.Record, residuals []project.ResidualRecord) projectResidualsResponse {
	response := projectResidualsResponse{
		Project:   buildProjectRecordResponse(record),
		Residuals: make([]residualResponse, 0, len(residuals)),
	}
	for _, residual := range residuals {
		response.Residuals = append(response.Residuals, buildResidualResponse(residual))
	}
	return response
}

func buildWorkflowVersionResidualsResponse(record project.Record, version project.WorkflowVersion, residuals []project.ResidualRecord) workflowVersionResidualsResponse {
	response := workflowVersionResidualsResponse{
		Project:         buildProjectRecordResponse(record),
		WorkflowVersion: buildWorkflowVersionResponse(version),
		Residuals:       make([]residualResponse, 0, len(residuals)),
	}
	for _, residual := range residuals {
		response.Residuals = append(response.Residuals, buildResidualResponse(residual))
	}
	return response
}

func buildStatusProjectionsResponse(record project.Record, projections []project.StatusProjectionRecord) statusProjectionsResponse {
	response := statusProjectionsResponse{
		Project:     buildProjectRecordResponse(record),
		Projections: make([]statusProjectionResponse, 0, len(projections)),
	}
	for _, projection := range projections {
		response.Projections = append(response.Projections, buildStatusProjectionResponse(projection))
	}
	return response
}

func buildStatusProjectionApplyResponse(result project.ApplyStatusProjectionResult) statusProjectionApplyResponse {
	return statusProjectionApplyResponse{
		Project:                   buildProjectRecordResponse(result.Project),
		Status:                    result.Status,
		Decision:                  result.Decision,
		Message:                   result.Message,
		Blockers:                  jsonStringSlice(result.Blockers),
		EventID:                   result.EventID,
		AuditEventID:              result.AuditEventID,
		SnapshotID:                result.SnapshotID,
		StatusProjectionID:        result.StatusProjectionID,
		TargetKind:                result.TargetKind,
		TargetURI:                 result.TargetURI,
		WrittenTarget:             result.WrittenTarget,
		WriteHash:                 result.WriteHash,
		WriteSize:                 result.WriteSize,
		PreimageCaptured:          result.PreimageCaptured,
		PreimageExists:            result.PreimageExists,
		PreimageSHA256:            result.PreimageSHA256,
		PreimageSize:              result.PreimageSize,
		PostWriteVerified:         result.PostWriteVerified,
		PostWriteSHA256:           result.PostWriteSHA256,
		PostWriteSize:             result.PostWriteSize,
		ProtectedPathsVerified:    result.ProtectedPathsVerified,
		ProtectedPathBeforeHash:   result.ProtectedPathBeforeHash,
		ProtectedPathAfterHash:    result.ProtectedPathAfterHash,
		ExpectedProtectedPathHash: result.ExpectedProtectedPathHash,
		RootContained:             result.RootContained,
		StableProjectionValid:     result.StableProjectionValid,
		AtomicReplaceUsed:         result.AtomicReplaceUsed,
		RollbackCompensation:      result.RollbackCompensation,
		SourceHash:                result.SourceHash,
		SummaryState:              result.SummaryState,
		ApplyGateStatus:           result.ApplyGateStatus,
		ApplyGateDecision:         result.ApplyGateDecision,
		ApplyGateApprovalStatus:   result.ApplyGateApprovalStatus,
		ApplyCommandEligible:      result.ApplyCommandEligible,
		IdempotencyKey:            result.IdempotencyKey,
		Created:                   result.Created,
		GeneratedAt:               formatTime(result.GeneratedAt),
		ProjectWriteAttempted:     result.ProjectWriteAttempted,
		ExecutionWriteAttempted:   result.ExecutionWriteAttempted,
		EngineCallAttempted:       result.EngineCallAttempted,
	}
}

func buildStatusProjectionApplyGateResponse(gate project.StatusProjectionApplyGate) statusProjectionApplyGateResponse {
	response := statusProjectionApplyGateResponse{
		Project:                        buildProjectRecordResponse(gate.Project),
		Status:                         gate.Status,
		Mode:                           gate.Mode,
		ClaimScope:                     gate.ClaimScope,
		NotReal100:                     gate.NotReal100,
		Decision:                       gate.Decision,
		Message:                        gate.Message,
		TargetURI:                      gate.TargetURI,
		TargetPath:                     gate.TargetPath,
		Authorization:                  buildStatusProjectionAuthorizationPreviewResponse(gate.Authorization),
		Items:                          make([]statusProjectionApplyGateItemResponse, 0, len(gate.Items)),
		RequiredPacketFields:           jsonStringSlice(gate.RequiredPacketFields),
		RequiredCapabilities:           jsonStringSlice(gate.RequiredCapabilities),
		RequiredAuthorizationPhrase:    gate.RequiredAuthorizationPhrase,
		ProtectedPaths:                 jsonStringSlice(gate.ProtectedPaths),
		ForbiddenActions:               jsonStringSlice(gate.ForbiddenActions),
		SafetyFacts:                    gate.SafetyFacts,
		ApplyCommandEligible:           gate.ApplyCommandEligible,
		ApplyCommandEligibleIsNotApply: gate.ApplyCommandEligibleIsNotApply,
		RequiresSeparateApplyCommand:   gate.RequiresSeparateApplyCommand,
		ApprovalRequired:               gate.ApprovalRequired,
		ApprovalStatus:                 gate.ApprovalStatus,
		ProjectWriteAttempted:          gate.ProjectWriteAttempted,
		ExecutionWriteAttempted:        gate.ExecutionWriteAttempted,
		EngineCallAttempted:            gate.EngineCallAttempted,
		CommandRequestCreated:          gate.CommandRequestCreated,
		StatusProjectionWritten:        gate.StatusProjectionWritten,
		GeneratedAt:                    formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, buildStatusProjectionApplyGateItemResponse(item))
	}
	return response
}

func buildStatusProjectionApplyPacketPreviewResponse(preview project.StatusProjectionApplyPacketPreview) statusProjectionApplyPacketPreviewResponse {
	return statusProjectionApplyPacketPreviewResponse{
		Project:                     buildProjectRecordResponse(preview.Project),
		Status:                      preview.Status,
		Mode:                        preview.Mode,
		ClaimScope:                  preview.ClaimScope,
		NotReal100:                  preview.NotReal100,
		Decision:                    preview.Decision,
		Message:                     preview.Message,
		Blockers:                    jsonStringSlice(preview.Blockers),
		RequiredAuthorizationPhrase: preview.RequiredAuthorizationPhrase,
		Authorization:               buildStatusProjectionAuthorizationPreviewResponse(preview.Authorization),
		Gate:                        buildStatusProjectionApplyGateResponse(preview.Gate),
		Packet:                      buildStatusProjectionApplyPacketResponse(preview.Packet),
		ApplyCommand:                jsonStringSlice(preview.ApplyCommand),
		APIRequest:                  buildStatusProjectionApplyAPIRequestResponse(preview.APIRequest),
		RequiredHumanReview:         jsonStringSlice(preview.RequiredHumanReview),
		ForbiddenActions:            jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:                 preview.SafetyFacts,
		WouldCreateCommandRequestAfterApplyCommand:   preview.WouldCreateCommandRequestAfterApplyCommand,
		WouldCreateStatusProjectionAfterApplyCommand: preview.WouldCreateStatusProjectionAfterApplyCommand,
		WouldWriteProjectFileAfterApplyCommand:       preview.WouldWriteProjectFileAfterApplyCommand,
		ApplyCommandEligibleIsNotApply:               preview.ApplyCommandEligibleIsNotApply,
		RequiresSeparateApplyCommand:                 preview.RequiresSeparateApplyCommand,
		ProjectWriteAttempted:                        preview.ProjectWriteAttempted,
		ExecutionWriteAttempted:                      preview.ExecutionWriteAttempted,
		EngineCallAttempted:                          preview.EngineCallAttempted,
		CommandRequestCreated:                        preview.CommandRequestCreated,
		StatusProjectionWritten:                      preview.StatusProjectionWritten,
		GeneratedAt:                                  formatTime(preview.GeneratedAt),
	}
}

func buildStatusProjectionApplyPacketResponse(packet project.StatusProjectionApplyPacket) statusProjectionApplyPacketResponse {
	return statusProjectionApplyPacketResponse{
		TargetURI:                      packet.TargetURI,
		ExpectedBeforeExists:           packet.ExpectedBeforeExists,
		ExpectedBeforeSHA256:           packet.ExpectedBeforeSHA256,
		ExpectedBeforeSizeBytes:        packet.ExpectedBeforeSizeBytes,
		SourceHash:                     packet.SourceHash,
		SchemaURI:                      packet.SchemaURI,
		ValidatorPreflight:             packet.ValidatorPreflight,
		ProtectedPathCheck:             packet.ProtectedPathCheck,
		ProtectedPathFingerprintSHA256: packet.ProtectedPathFingerprintSHA256,
		RollbackAction:                 packet.RollbackAction,
		AcceptedPreimageSchemaStatus:   packet.AcceptedPreimageSchemaStatus,
		ExplicitApproval:               packet.ExplicitApproval,
		ApprovalActor:                  packet.ApprovalActor,
		ApprovalReason:                 packet.ApprovalReason,
		RequiredAuthorizationPhrase:    packet.RequiredAuthorizationPhrase,
	}
}

func buildStatusProjectionApplyAPIRequestResponse(request project.StatusProjectionApplyAPIRequest) statusProjectionApplyAPIRequestResponse {
	return statusProjectionApplyAPIRequestResponse(buildStatusProjectionApplyPacketResponse(project.StatusProjectionApplyPacket{
		TargetURI:                      request.TargetURI,
		ExpectedBeforeExists:           request.ExpectedBeforeExists,
		ExpectedBeforeSHA256:           request.ExpectedBeforeSHA256,
		ExpectedBeforeSizeBytes:        request.ExpectedBeforeSizeBytes,
		SourceHash:                     request.SourceHash,
		SchemaURI:                      request.SchemaURI,
		ValidatorPreflight:             request.ValidatorPreflight,
		ProtectedPathCheck:             request.ProtectedPathCheck,
		ProtectedPathFingerprintSHA256: request.ProtectedPathFingerprintSHA256,
		RollbackAction:                 request.RollbackAction,
		AcceptedPreimageSchemaStatus:   request.AcceptedPreimageSchemaStatus,
		ExplicitApproval:               request.ExplicitApproval,
		ApprovalActor:                  request.ApprovalActor,
		ApprovalReason:                 request.ApprovalReason,
		RequiredAuthorizationPhrase:    request.RequiredAuthorizationPhrase,
	}))
}

func buildStatusProjectionApplyGateItemResponse(item project.StatusProjectionApplyGateItem) statusProjectionApplyGateItemResponse {
	return statusProjectionApplyGateItemResponse{
		Key:              item.Key,
		Category:         item.Category,
		Status:           item.Status,
		Message:          item.Message,
		Expected:         item.Expected,
		Actual:           item.Actual,
		RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
		BlockedBy:        jsonStringSlice(item.BlockedBy),
	}
}

func buildStatusProjectionAuthorizationPreviewResponse(preview project.StatusProjectionAuthorizationPreview) statusProjectionAuthorizationPreviewResponse {
	response := statusProjectionAuthorizationPreviewResponse{
		Project:                                buildProjectRecordResponse(preview.Project),
		Status:                                 preview.Status,
		Mode:                                   preview.Mode,
		ClaimScope:                             preview.ClaimScope,
		NotReal100:                             preview.NotReal100,
		Decision:                               preview.Decision,
		Message:                                preview.Message,
		TargetKind:                             preview.TargetKind,
		TargetURI:                              preview.TargetURI,
		TargetPath:                             preview.TargetPath,
		SchemaURI:                              preview.SchemaURI,
		ValidatorPreflight:                     preview.ValidatorPreflight,
		ProtectedPathFingerprintSHA256:         preview.ProtectedPathFingerprintSHA256,
		SourceHash:                             preview.SourceHash,
		SummaryState:                           preview.SummaryState,
		RequiredAuthorizationPhrase:            preview.RequiredAuthorizationPhrase,
		Permission:                             buildStatusProjectionAuthorizationPermissionResponse(preview.Permission),
		Preimage:                               buildStatusProjectionPreimageResponse(preview.Preimage),
		WriteSet:                               make([]statusProjectionWriteSetEntryResponse, 0, len(preview.WriteSet)),
		RequiredPreflight:                      jsonStringSlice(preview.RequiredPreflight),
		RequiredPacketFields:                   jsonStringSlice(preview.RequiredPacketFields),
		RequiredCapabilities:                   jsonStringSlice(preview.RequiredCapabilities),
		ProtectedPaths:                         jsonStringSlice(preview.ProtectedPaths),
		RollbackPlan:                           jsonStringSlice(preview.RollbackPlan),
		BlockedBy:                              jsonStringSlice(preview.BlockedBy),
		Warnings:                               jsonStringSlice(preview.Warnings),
		ForbiddenActions:                       jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:                            preview.SafetyFacts,
		ApplyOpen:                              preview.ApplyOpen,
		ApprovalRequired:                       preview.ApprovalRequired,
		ApprovalStatus:                         preview.ApprovalStatus,
		WouldCreateCommandRequestAfterApproval: preview.WouldCreateCommandRequestAfterApproval,
		WouldCreateProjectStatusSnapshotAfterApproval: preview.WouldCreateProjectStatusSnapshotAfterApproval,
		WouldCreateStatusProjectionAfterApproval:      preview.WouldCreateStatusProjectionAfterApproval,
		WouldCreateEventAfterApproval:                 preview.WouldCreateEventAfterApproval,
		WouldCreateAuditEventAfterApproval:            preview.WouldCreateAuditEventAfterApproval,
		WouldWriteProjectFileAfterApproval:            preview.WouldWriteProjectFileAfterApproval,
		WouldWriteExecutionAfterApproval:              preview.WouldWriteExecutionAfterApproval,
		WouldRunEngineAfterApproval:                   preview.WouldRunEngineAfterApproval,
		ProjectWriteAttempted:                         preview.ProjectWriteAttempted,
		ExecutionWriteAttempted:                       preview.ExecutionWriteAttempted,
		EngineCallAttempted:                           preview.EngineCallAttempted,
		GeneratedAt:                                   formatTime(preview.GeneratedAt),
	}
	for _, entry := range preview.WriteSet {
		response.WriteSet = append(response.WriteSet, buildStatusProjectionWriteSetEntryResponse(entry))
	}
	return response
}

func buildStatusProjectionAuthorizationPermissionResponse(permission project.StatusProjectionAuthorizationPermission) statusProjectionAuthorizationPermissionResponse {
	return statusProjectionAuthorizationPermissionResponse{
		Capability:        permission.Capability,
		ResourceType:      permission.ResourceType,
		TargetURI:         permission.TargetURI,
		CapabilityAllowed: permission.CapabilityAllowed,
		PathAllowed:       permission.PathAllowed,
		Allowed:           permission.Allowed,
		Reason:            permission.Reason,
	}
}

func buildStatusProjectionPreimageResponse(preimage project.StatusProjectionPreimage) statusProjectionPreimageResponse {
	return statusProjectionPreimageResponse{
		TargetPath:               preimage.TargetPath,
		Exists:                   preimage.Exists,
		Readable:                 preimage.Readable,
		SizeBytes:                preimage.SizeBytes,
		SHA256:                   preimage.SHA256,
		SchemaStatus:             preimage.SchemaStatus,
		LegacyShape:              preimage.LegacyShape,
		MissingRequiredFields:    jsonStringSlice(preimage.MissingRequiredFields),
		UnexpectedTopLevelFields: jsonStringSlice(preimage.UnexpectedTopLevelFields),
		CompatibilityMissing:     jsonStringSlice(preimage.CompatibilityMissing),
		CompatibilityUnexpected:  jsonStringSlice(preimage.CompatibilityUnexpected),
		Message:                  preimage.Message,
	}
}

func buildStatusProjectionWriteSetEntryResponse(entry project.StatusProjectionWriteSetEntry) statusProjectionWriteSetEntryResponse {
	return statusProjectionWriteSetEntryResponse{
		TargetURI:                entry.TargetURI,
		TargetPath:               entry.TargetPath,
		Operation:                entry.Operation,
		Capability:               entry.Capability,
		ExpectedBeforeExists:     entry.ExpectedBeforeExists,
		ExpectedBeforeSHA256:     entry.ExpectedBeforeSHA256,
		ExpectedBeforeSizeBytes:  entry.ExpectedBeforeSizeBytes,
		RequiresPreimageMatch:    entry.RequiresPreimageMatch,
		RequiresSchemaValidation: entry.RequiresSchemaValidation,
		RollbackAction:           entry.RollbackAction,
		ProtectedPath:            entry.ProtectedPath,
	}
}

func buildStatusProjectionResponse(projection project.StatusProjectionRecord) statusProjectionResponse {
	response := statusProjectionResponse{
		ID:                projection.ID,
		ProjectID:         projection.ProjectID,
		WorkflowVersionID: projection.WorkflowVersionID,
		TargetKind:        projection.TargetKind,
		TargetURI:         projection.TargetURI,
		SummaryState:      projection.SummaryState,
		Payload:           projection.Payload,
		SourceEventID:     projection.SourceEventID,
		SourceHash:        projection.SourceHash,
		WriteState:        projection.WriteState,
		GeneratedAt:       formatTime(projection.GeneratedAt),
		Metadata:          projection.Metadata,
	}
	if projection.WrittenAt != nil {
		response.WrittenAt = formatTime(*projection.WrittenAt)
	}
	return response
}

func statusProjectionWriter(ctx context.Context, record project.Record, snapshot project.Snapshot, targetURI string) (project.StatusProjectionWriteResult, error) {
	_ = ctx
	result, err := statusmirror.WriteWithResult(record, snapshot, targetURI)
	if err != nil {
		return project.StatusProjectionWriteResult{}, err
	}
	return project.StatusProjectionWriteResult{
		Target:                    result.Target,
		Hash:                      result.Hash,
		Size:                      result.Size,
		RootContained:             result.RootContained,
		StableProjectionValidated: result.StableProjectionValidated,
		AtomicReplaceUsed:         result.AtomicReplaceUsed,
	}, nil
}

func buildResidualResponse(residual project.ResidualRecord) residualResponse {
	response := residualResponse{
		ID:                residual.ID,
		WorkflowVersionID: residual.WorkflowVersionID,
		ResidualKey:       residual.ResidualKey,
		Status:            residual.Status,
		Type:              residual.Type,
		Title:             residual.Title,
		SourcePath:        residual.SourcePath,
		CurrentImpact:     residual.CurrentImpact,
		ExecutableTask:    residual.ExecutableTask,
		PromotionRequired: residual.PromotionRequired,
		CloseCondition:    residual.CloseCondition,
		Metadata:          residual.Metadata,
		Immutable:         residual.Immutable,
		CreatedAt:         formatTime(residual.CreatedAt),
		UpdatedAt:         formatTime(residual.UpdatedAt),
	}
	if residual.ImportedAt != nil {
		response.ImportedAt = formatTime(*residual.ImportedAt)
	}
	return response
}

func buildArtifactResponse(artifact project.ArtifactRecord) artifactResponse {
	return artifactResponse{
		ID:                artifact.ID,
		WorkflowVersionID: artifact.WorkflowVersionID,
		WorkflowItemID:    artifact.WorkflowItemID,
		ArtifactType:      artifact.ArtifactType,
		StorageBackend:    artifact.StorageBackend,
		URI:               artifact.URI,
		SourcePath:        artifact.SourcePath,
		SHA256:            artifact.SHA256,
		SizeBytes:         artifact.SizeBytes,
		ContentType:       artifact.ContentType,
		Metadata:          artifact.Metadata,
		CreatedAt:         formatTime(artifact.CreatedAt),
	}
}

func buildRunnerPreflightResponse(preflight project.RunnerPreflight) runnerPreflightResponse {
	response := runnerPreflightResponse{
		Status:   preflight.Status,
		Checks:   make([]projectReadinessItemResponse, 0, len(preflight.Checks)),
		Blockers: preflight.Blockers,
	}
	for _, check := range preflight.Checks {
		response.Checks = append(response.Checks, projectReadinessItemResponse{
			Key:      check.Key,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return response
}

func buildRunResponse(run project.RunRecord) runResponse {
	response := runResponse{
		ID:                run.ID,
		WorkflowVersionID: run.WorkflowVersionID,
		RunType:           run.RunType,
		RunKind:           run.RunKind,
		Status:            run.Status,
		RiskLevel:         run.RiskLevel,
		RiskPolicy:        run.RiskPolicy,
		DryRun:            run.DryRun,
		Summary:           run.Summary,
		Metadata:          run.Metadata,
		StartedAt:         formatTime(run.StartedAt),
	}
	if run.FinishedAt != nil {
		response.FinishedAt = formatTime(*run.FinishedAt)
	}
	return response
}

func buildRunTaskResponse(task project.RunTaskRecord) runTaskResponse {
	return runTaskResponse{
		ID:                task.ID,
		WorkflowVersionID: task.WorkflowVersionID,
		WorkflowItemID:    task.WorkflowItemID,
		RunID:             task.RunID,
		TaskKey:           task.TaskKey,
		TaskKind:          task.TaskKind,
		Status:            task.Status,
		RiskLevel:         task.RiskLevel,
		Sequence:          task.Sequence,
		Metadata:          task.Metadata,
		CreatedAt:         formatTime(task.CreatedAt),
		UpdatedAt:         formatTime(task.UpdatedAt),
	}
}

func buildRunAttemptResponse(attempt project.RunAttemptRecord) runAttemptResponse {
	response := runAttemptResponse{
		ID:                attempt.ID,
		WorkflowVersionID: attempt.WorkflowVersionID,
		WorkflowItemID:    attempt.WorkflowItemID,
		RunID:             attempt.RunID,
		RunTaskID:         attempt.RunTaskID,
		AttemptKind:       attempt.AttemptKind,
		Status:            attempt.Status,
		DryRun:            attempt.DryRun,
		Metadata:          attempt.Metadata,
		StartedAt:         formatTime(attempt.StartedAt),
	}
	if attempt.FinishedAt != nil {
		response.FinishedAt = formatTime(*attempt.FinishedAt)
	}
	return response
}

func buildWorkerListResponse(record project.Record, workers []project.WorkerRecord) workerListResponse {
	response := workerListResponse{
		Project: buildProjectRecordResponse(record),
		Workers: make([]workerResponse, 0, len(workers)),
	}
	for _, worker := range workers {
		response.Workers = append(response.Workers, buildWorkerResponse(worker))
	}
	return response
}

func buildLocalServiceStatusResponse(status project.LocalServiceStatus) localServiceStatusResponse {
	return localServiceStatusResponse{
		Status: status.Status,
		Mode:   status.Mode,
		API: localServiceComponentResponse{
			Status:  status.API.Status,
			Message: status.API.Message,
		},
		Database: localServiceComponentResponse{
			Status:  status.Database.Status,
			Message: status.Database.Message,
		},
		WorkerPool: localServiceWorkerPoolResponse{
			Status:             status.WorkerPool.Status,
			Message:            status.WorkerPool.Message,
			TotalProjects:      status.WorkerPool.TotalProjects,
			TotalWorkers:       status.WorkerPool.TotalWorkers,
			TotalOnlineWorkers: status.WorkerPool.TotalOnlineWorkers,
			TotalActiveLeases:  status.WorkerPool.TotalActiveLeases,
			TotalQueuedTasks:   status.WorkerPool.TotalQueuedTasks,
			TotalNeedsRecovery: status.WorkerPool.TotalNeedsRecovery,
		},
		Dashboard: localServiceDashboardResponse{
			URL:     status.Dashboard.URL,
			APIURL:  status.Dashboard.APIURL,
			Status:  status.Dashboard.Status,
			Message: status.Dashboard.Message,
		},
		Capabilities:     jsonStringSlice(status.Capabilities),
		ForbiddenActions: jsonStringSlice(status.ForbiddenActions),
		GeneratedAt:      formatTime(status.GeneratedAt),
	}
}

func buildWebWriteActionGateResponse(gate project.WebWriteActionGate) webWriteActionGateResponse {
	response := webWriteActionGateResponse{
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Actions:                 make([]webWriteActionResponse, 0, len(gate.Actions)),
		Capabilities:            jsonStringSlice(gate.Capabilities),
		ForbiddenActions:        jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:             formatTime(gate.GeneratedAt),
		DBWriteAttempted:        gate.DBWriteAttempted,
		ProjectWriteAttempted:   gate.ProjectWriteAttempted,
		ArtifactWriteAttempted:  gate.ArtifactWriteAttempted,
		ExecutionWriteAttempted: gate.ExecutionWriteAttempted,
		CommandCreated:          gate.CommandCreated,
		ApprovalCreated:         gate.ApprovalCreated,
		AuditEventWritten:       gate.AuditEventWritten,
		WorkerScheduled:         gate.WorkerScheduled,
		EngineCallAttempted:     gate.EngineCallAttempted,
		CommandsRun:             gate.CommandsRun,
		SecretsResolved:         gate.SecretsResolved,
		NetworkUsed:             gate.NetworkUsed,
	}
	for _, action := range gate.Actions {
		response.Actions = append(response.Actions, webWriteActionResponse{
			Key:                    action.Key,
			Label:                  action.Label,
			Category:               action.Category,
			Status:                 action.Status,
			DefaultUIState:         action.DefaultUIState,
			CommandAPI:             action.CommandAPI,
			RiskLevel:              action.RiskLevel,
			RequiredCapabilities:   jsonStringSlice(action.RequiredCapabilities),
			RequiredPreviews:       jsonStringSlice(action.RequiredPreviews),
			RequiredApprovals:      jsonStringSlice(action.RequiredApprovals),
			RequiredAuditEvents:    jsonStringSlice(action.RequiredAuditEvents),
			RequiredEvidence:       jsonStringSlice(action.RequiredEvidence),
			Blockers:               jsonStringSlice(action.Blockers),
			ForbiddenDirectActions: jsonStringSlice(action.ForbiddenDirectActions),
		})
	}
	return response
}

func buildDesktopServiceControlGateResponse(gate project.DesktopServiceControlGate) desktopServiceControlGateResponse {
	response := desktopServiceControlGateResponse{
		Status:                   gate.Status,
		Mode:                     gate.Mode,
		Actions:                  make([]desktopServiceControlAction, 0, len(gate.Actions)),
		Capabilities:             jsonStringSlice(gate.Capabilities),
		ForbiddenActions:         jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:              formatTime(gate.GeneratedAt),
		DBWriteAttempted:         gate.DBWriteAttempted,
		ProjectWriteAttempted:    gate.ProjectWriteAttempted,
		ProcessControlAttempted:  gate.ProcessControlAttempted,
		CommandCreated:           gate.CommandCreated,
		ApprovalCreated:          gate.ApprovalCreated,
		AuditEventWritten:        gate.AuditEventWritten,
		WorkerScheduled:          gate.WorkerScheduled,
		WorkflowExecutionStarted: gate.WorkflowExecutionStarted,
		SecretsResolved:          gate.SecretsResolved,
		NetworkUsed:              gate.NetworkUsed,
	}
	for _, action := range gate.Actions {
		response.Actions = append(response.Actions, desktopServiceControlAction{
			Key:                    action.Key,
			Label:                  action.Label,
			Category:               action.Category,
			Status:                 action.Status,
			DefaultUIState:         action.DefaultUIState,
			CommandAPI:             action.CommandAPI,
			RiskLevel:              action.RiskLevel,
			RequiredCapabilities:   jsonStringSlice(action.RequiredCapabilities),
			RequiredPreviews:       jsonStringSlice(action.RequiredPreviews),
			RequiredApprovals:      jsonStringSlice(action.RequiredApprovals),
			RequiredAuditEvents:    jsonStringSlice(action.RequiredAuditEvents),
			RequiredEvidence:       jsonStringSlice(action.RequiredEvidence),
			Blockers:               jsonStringSlice(action.Blockers),
			ForbiddenDirectActions: jsonStringSlice(action.ForbiddenDirectActions),
		})
	}
	return response
}

func buildDesktopNotificationGateResponse(gate project.DesktopNotificationGate) desktopNotificationGateResponse {
	response := desktopNotificationGateResponse{
		Status:                   gate.Status,
		Mode:                     gate.Mode,
		Actions:                  make([]desktopNotificationAction, 0, len(gate.Actions)),
		Capabilities:             jsonStringSlice(gate.Capabilities),
		ForbiddenActions:         jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:              formatTime(gate.GeneratedAt),
		DBWriteAttempted:         gate.DBWriteAttempted,
		ProjectWriteAttempted:    gate.ProjectWriteAttempted,
		EventStreamOpened:        gate.EventStreamOpened,
		NotificationRequested:    gate.NotificationRequested,
		CommandCreated:           gate.CommandCreated,
		ApprovalCreated:          gate.ApprovalCreated,
		AuditEventWritten:        gate.AuditEventWritten,
		WorkerScheduled:          gate.WorkerScheduled,
		WorkflowExecutionStarted: gate.WorkflowExecutionStarted,
		SecretsResolved:          gate.SecretsResolved,
		NetworkUsed:              gate.NetworkUsed,
	}
	for _, action := range gate.Actions {
		response.Actions = append(response.Actions, desktopNotificationAction{
			Key:                    action.Key,
			Label:                  action.Label,
			Category:               action.Category,
			Status:                 action.Status,
			DefaultUIState:         action.DefaultUIState,
			RiskLevel:              action.RiskLevel,
			RequiredCapabilities:   jsonStringSlice(action.RequiredCapabilities),
			RequiredPreviews:       jsonStringSlice(action.RequiredPreviews),
			RequiredApprovals:      jsonStringSlice(action.RequiredApprovals),
			RequiredAuditEvents:    jsonStringSlice(action.RequiredAuditEvents),
			RequiredEvidence:       jsonStringSlice(action.RequiredEvidence),
			Blockers:               jsonStringSlice(action.Blockers),
			ForbiddenDirectActions: jsonStringSlice(action.ForbiddenDirectActions),
		})
	}
	return response
}

func buildDesktopTrayMenuGateResponse(gate project.DesktopTrayMenuGate) desktopTrayMenuGateResponse {
	response := desktopTrayMenuGateResponse{
		Status:                   gate.Status,
		Mode:                     gate.Mode,
		Actions:                  make([]desktopTrayMenuAction, 0, len(gate.Actions)),
		Capabilities:             jsonStringSlice(gate.Capabilities),
		ForbiddenActions:         jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:              formatTime(gate.GeneratedAt),
		DBWriteAttempted:         gate.DBWriteAttempted,
		ProjectWriteAttempted:    gate.ProjectWriteAttempted,
		TrayMenuCreated:          gate.TrayMenuCreated,
		OSIntegrationRequested:   gate.OSIntegrationRequested,
		CommandCreated:           gate.CommandCreated,
		ApprovalCreated:          gate.ApprovalCreated,
		AuditEventWritten:        gate.AuditEventWritten,
		ServiceControlAttempted:  gate.ServiceControlAttempted,
		NotificationRequested:    gate.NotificationRequested,
		WorkerScheduled:          gate.WorkerScheduled,
		WorkflowExecutionStarted: gate.WorkflowExecutionStarted,
		SecretsResolved:          gate.SecretsResolved,
		NetworkUsed:              gate.NetworkUsed,
	}
	for _, action := range gate.Actions {
		response.Actions = append(response.Actions, desktopTrayMenuAction{
			Key:                    action.Key,
			Label:                  action.Label,
			Category:               action.Category,
			Status:                 action.Status,
			DefaultUIState:         action.DefaultUIState,
			RiskLevel:              action.RiskLevel,
			RequiredCapabilities:   jsonStringSlice(action.RequiredCapabilities),
			RequiredPreviews:       jsonStringSlice(action.RequiredPreviews),
			RequiredApprovals:      jsonStringSlice(action.RequiredApprovals),
			RequiredAuditEvents:    jsonStringSlice(action.RequiredAuditEvents),
			RequiredEvidence:       jsonStringSlice(action.RequiredEvidence),
			Blockers:               jsonStringSlice(action.Blockers),
			ForbiddenDirectActions: jsonStringSlice(action.ForbiddenDirectActions),
		})
	}
	return response
}

func buildSecurityBoundaryReadinessResponse(readiness project.SecurityBoundaryReadiness) securityBoundaryReadinessResponse {
	response := securityBoundaryReadinessResponse{
		Status:                        readiness.Status,
		Mode:                          readiness.Mode,
		Items:                         make([]securityBoundaryReadinessItemResponse, 0, len(readiness.Items)),
		Capabilities:                  jsonStringSlice(readiness.Capabilities),
		ForbiddenActions:              jsonStringSlice(readiness.ForbiddenActions),
		GeneratedAt:                   formatTime(readiness.GeneratedAt),
		AuthEnforcementOpen:           readiness.AuthEnforcementOpen,
		TeamPermissionEnforcementOpen: readiness.TeamPermissionEnforcementOpen,
		APITokenIssuanceOpen:          readiness.APITokenIssuanceOpen,
		APITokenEnforcementOpen:       readiness.APITokenEnforcementOpen,
		SecretResolveOpen:             readiness.SecretResolveOpen,
		RemoteWorkerCredentialsOpen:   readiness.RemoteWorkerCredentialsOpen,
		BudgetEnforcementOpen:         readiness.BudgetEnforcementOpen,
		QuotaDecrementOpen:            readiness.QuotaDecrementOpen,
		UsageChargeWritten:            readiness.UsageChargeWritten,
		WebhookDeliveryOpen:           readiness.WebhookDeliveryOpen,
		InboundCallbackOpen:           readiness.InboundCallbackOpen,
		ExternalAPICallOpen:           readiness.ExternalAPICallOpen,
		AuthorizationChanged:          readiness.AuthorizationChanged,
		SecretPlaintextRead:           readiness.SecretPlaintextRead,
		RemoteWorkerDirectPGAllowed:   readiness.RemoteWorkerDirectPGAllowed,
		TeamConsoleCommandOpen:        readiness.TeamConsoleCommandOpen,
		RemoteOpsControlOpen:          readiness.RemoteOpsControlOpen,
		ManagedUpgradeOpen:            readiness.ManagedUpgradeOpen,
		SupportBundleExportOpen:       readiness.SupportBundleExportOpen,
		DefaultRemoteTelemetryOpen:    readiness.DefaultRemoteTelemetryOpen,
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, securityBoundaryReadinessItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			BlockedBy:        jsonStringSlice(item.BlockedBy),
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildCompletionAuditResponse(audit project.CompletionAudit) completionAuditResponse {
	guardrail := completionAuditReal100Guardrail(audit.Real100Guardrail)
	response := completionAuditResponse{
		Status:                     audit.Status,
		Mode:                       audit.Mode,
		Scope:                      audit.Scope,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Items:                      make([]completionAuditItemResponse, 0, len(audit.Items)),
		DeferredV1x:                jsonStringSlice(audit.DeferredV1x),
		Capabilities:               jsonStringSlice(audit.Capabilities),
		ForbiddenActions:           jsonStringSlice(audit.ForbiddenActions),
		SafetyFacts:                audit.SafetyFacts,
		ReleaseFinalGateStatus:     audit.ReleaseFinalGateStatus,
		AreaMatrixDogfoodStatus:    audit.AreaMatrixDogfoodStatus,
		TaskMatrixStatus:           audit.TaskMatrixStatus,
		ImplementationGapStatus:    audit.ImplementationGapStatus,
		ProtectedPathProofStatus:   audit.ProtectedPathProofStatus,
		GeneratedAt:                formatTime(audit.GeneratedAt),
	}
	for _, item := range audit.Items {
		response.Items = append(response.Items, completionAuditItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			EvidenceRefs:     jsonStringSlice(item.EvidenceRefs),
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			BlockedBy:        jsonStringSlice(item.BlockedBy),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildCompletionAuditSnapshotReadinessResponse(readiness project.CompletionAuditSnapshotReadiness) completionAuditSnapshotReadinessResponse {
	guardrail := completionAuditReal100Guardrail(readiness.Real100Guardrail)
	response := completionAuditSnapshotReadinessResponse{
		Project:                    buildProjectRecordResponse(readiness.Project),
		Status:                     readiness.Status,
		Message:                    readiness.Message,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		HasSnapshot:                readiness.HasSnapshot,
		RequiredClass:              readiness.RequiredClass,
		BundleHash:                 readiness.BundleHash,
		Latest:                     buildCompletionAuditSnapshotResponse(readiness.Latest),
		Items:                      make([]projectReadinessItemResponse, 0, len(readiness.Items)),
		Gaps:                       project.CompletionAuditSnapshotReadinessGaps(readiness),
		Closure:                    project.CompletionAuditSnapshotReadinessClosure(readiness),
		SafetyFacts:                readiness.SafetyFacts,
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return response
}

func buildCompletionAuditSnapshotResponse(snapshot project.CompletionAuditSnapshot) completionAuditSnapshotResponse {
	guardrail := completionAuditReal100Guardrail(snapshot.Real100Guardrail)
	return completionAuditSnapshotResponse{
		Status:                     snapshot.Status,
		Decision:                   snapshot.Decision,
		Message:                    snapshot.Message,
		AuditStatus:                snapshot.AuditStatus,
		AuditScope:                 snapshot.AuditScope,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		AuditHash:                  snapshot.AuditHash,
		ReleaseCandidateLabel:      snapshot.ReleaseCandidateLabel,
		EvidenceClass:              snapshot.EvidenceClass,
		EvidenceURI:                snapshot.EvidenceURI,
		ProofEventIDs:              snapshot.ProofEventIDs,
		EventID:                    snapshot.EventID,
		AuditEventID:               snapshot.AuditEventID,
		IdempotencyKey:             snapshot.IdempotencyKey,
		CreatedAt:                  formatTime(snapshot.CreatedAt),
		Metadata:                   snapshot.Metadata,
	}
}

func buildSupportBundlePreviewResponse(preview project.SupportBundlePreview) supportBundlePreviewResponse {
	response := supportBundlePreviewResponse{
		Status:                   preview.Status,
		Mode:                     preview.Mode,
		BundleID:                 preview.BundleID,
		Scope:                    preview.Scope,
		Projects:                 make([]projectRecordResponse, 0, len(preview.Projects)),
		IncludedMetadata:         jsonStringSlice(preview.IncludedMetadata),
		ExcludedSensitiveContent: jsonStringSlice(preview.ExcludedSensitiveContent),
		PathReferences:           make([]supportBundlePathReferenceResponse, 0, len(preview.PathReferences)),
		Hashes:                   make([]supportBundleHashReferenceResponse, 0, len(preview.Hashes)),
		Capabilities:             jsonStringSlice(preview.Capabilities),
		ForbiddenActions:         jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:              preview.SafetyFacts,
		GeneratedAt:              formatTime(preview.GeneratedAt),
	}
	for _, record := range preview.Projects {
		response.Projects = append(response.Projects, buildProjectRecordResponse(record))
	}
	for _, ref := range preview.PathReferences {
		response.PathReferences = append(response.PathReferences, supportBundlePathReferenceResponse{
			Key:         ref.Key,
			Kind:        ref.Kind,
			URI:         ref.URI,
			ProjectKey:  ref.ProjectKey,
			Description: ref.Description,
		})
	}
	for _, hash := range preview.Hashes {
		response.Hashes = append(response.Hashes, supportBundleHashReferenceResponse{
			Key:         hash.Key,
			Hash:        hash.Hash,
			Source:      hash.Source,
			Description: hash.Description,
		})
	}
	return response
}

func buildMigrationLedgerReadinessResponse(readiness project.MigrationLedgerReadiness) migrationLedgerReadinessResponse {
	response := migrationLedgerReadinessResponse{
		Status:                               readiness.Status,
		Mode:                                 readiness.Mode,
		Entries:                              make([]migrationLedgerEntryResponse, 0, len(readiness.Entries)),
		AppliedCount:                         readiness.AppliedCount,
		PendingCount:                         readiness.PendingCount,
		SchemaMigrationsTablePresent:         readiness.SchemaMigrationsTablePresent,
		FullLedgerTablePresent:               readiness.FullLedgerTablePresent,
		PreflightApplyVerifyRemediationReady: readiness.PreflightApplyVerifyRemediationReady,
		Capabilities:                         jsonStringSlice(readiness.Capabilities),
		ForbiddenActions:                     jsonStringSlice(readiness.ForbiddenActions),
		SafetyFacts:                          readiness.SafetyFacts,
		GeneratedAt:                          formatTime(readiness.GeneratedAt),
	}
	for _, entry := range readiness.Entries {
		entryResponse := migrationLedgerEntryResponse{
			Name:             entry.Name,
			Applied:          entry.Applied,
			Status:           entry.Status,
			RequiredEvidence: jsonStringSlice(entry.RequiredEvidence),
			Phases:           make([]migrationLedgerPhaseResponse, 0, len(entry.Phases)),
			Metadata:         jsonObject(entry.Metadata),
		}
		for _, phase := range entry.Phases {
			entryResponse.Phases = append(entryResponse.Phases, migrationLedgerPhaseResponse{
				Phase:       phase.Phase,
				Status:      phase.Status,
				Message:     phase.Message,
				Remediation: phase.Remediation,
				Metadata:    jsonObject(phase.Metadata),
			})
		}
		response.Entries = append(response.Entries, entryResponse)
	}
	return response
}

func buildOperationsReadinessResponse(readiness project.OperationsReadiness) operationsReadinessResponse {
	response := operationsReadinessResponse{
		Status:              readiness.Status,
		Mode:                readiness.Mode,
		Items:               make([]operationsReadinessItemResponse, 0, len(readiness.Items)),
		ServiceStatus:       buildLocalServiceStatusResponse(readiness.ServiceStatus),
		SupportBundle:       buildSupportBundlePreviewResponse(readiness.SupportBundle),
		MigrationLedger:     buildMigrationLedgerReadinessResponse(readiness.MigrationLedger),
		Capabilities:        jsonStringSlice(readiness.Capabilities),
		ForbiddenActions:    jsonStringSlice(readiness.ForbiddenActions),
		SafetyFacts:         readiness.SafetyFacts,
		TelemetryDefault:    readiness.TelemetryDefault,
		ManagedOpsStatus:    readiness.ManagedOpsStatus,
		SupportExportStatus: readiness.SupportExportStatus,
		GeneratedAt:         formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, operationsReadinessItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			EvidenceRefs:     jsonStringSlice(item.EvidenceRefs),
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			BlockedBy:        jsonStringSlice(item.BlockedBy),
			NextCommand:      item.NextCommand,
			Metadata:         jsonObject(item.Metadata),
		})
	}
	return response
}

func buildBackupManifestResponse(manifest project.BackupManifest) backupManifestResponse {
	response := backupManifestResponse{
		Status:           manifest.Status,
		Mode:             manifest.Mode,
		Scope:            manifest.Scope,
		ProjectKey:       manifest.ProjectKey,
		SchemaVersion:    manifest.SchemaVersion,
		GeneratedAt:      formatTime(manifest.GeneratedAt),
		ManifestHash:     manifest.ManifestHash,
		TableCounts:      make([]backupTableCountResponse, 0, len(manifest.TableCounts)),
		Projects:         make([]backupProjectManifestResponse, 0, len(manifest.Projects)),
		Capabilities:     jsonStringSlice(manifest.Capabilities),
		ForbiddenActions: jsonStringSlice(manifest.ForbiddenActions),
	}
	for _, table := range manifest.TableCounts {
		response.TableCounts = append(response.TableCounts, backupTableCountResponse{
			Table: table.Table,
			Rows:  table.Rows,
		})
	}
	for _, projectManifest := range manifest.Projects {
		item := backupProjectManifestResponse{
			Project: buildProjectRecordResponse(projectManifest.Project),
			Inventory: projectInventoryResponse{
				Versions:        projectManifest.Inventory.Versions,
				Residuals:       projectManifest.Inventory.Residuals,
				Artifacts:       projectManifest.Inventory.Artifacts,
				ImportSnapshots: projectManifest.Inventory.ImportSnapshots,
				MirrorExports:   projectManifest.Inventory.MirrorExports,
			},
			ArtifactCount: projectManifest.ArtifactCount,
			Artifacts:     make([]backupArtifactSummaryResponse, 0, len(projectManifest.Artifacts)),
		}
		for _, artifact := range projectManifest.Artifacts {
			item.Artifacts = append(item.Artifacts, backupArtifactSummaryResponse{
				ID:                artifact.ID,
				ProjectID:         artifact.ProjectID,
				WorkflowVersionID: artifact.WorkflowVersionID,
				WorkflowItemID:    artifact.WorkflowItemID,
				ArtifactType:      artifact.ArtifactType,
				StorageBackend:    artifact.StorageBackend,
				URI:               artifact.URI,
				SourcePath:        artifact.SourcePath,
				SHA256:            artifact.SHA256,
				SizeBytes:         artifact.SizeBytes,
				ContentType:       artifact.ContentType,
				CreatedAt:         formatTime(artifact.CreatedAt),
			})
		}
		response.Projects = append(response.Projects, item)
	}
	return response
}

func buildRestorePlanResponse(plan project.RestorePlan) restorePlanResponse {
	response := restorePlanResponse{
		Status:           plan.Status,
		Mode:             plan.Mode,
		Scope:            plan.Scope,
		ProjectKey:       plan.ProjectKey,
		SchemaVersion:    plan.SchemaVersion,
		ManifestHash:     plan.ManifestHash,
		Projects:         make([]projectRecordResponse, 0, len(plan.Projects)),
		Items:            make([]restorePlanItemResponse, 0, len(plan.Items)),
		Capabilities:     jsonStringSlice(plan.Capabilities),
		ForbiddenActions: jsonStringSlice(plan.ForbiddenActions),
		GeneratedAt:      formatTime(plan.GeneratedAt),
	}
	for _, record := range plan.Projects {
		response.Projects = append(response.Projects, buildProjectRecordResponse(record))
	}
	for _, item := range plan.Items {
		response.Items = append(response.Items, restorePlanItemResponse{
			Key:      item.Key,
			Category: item.Category,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return response
}

func buildReleaseReadinessResponse(readiness project.ReleaseReadiness) releaseReadinessResponse {
	guardrail := releasePreviewReal100Guardrail(readiness.Real100Guardrail)
	response := releaseReadinessResponse{
		Status:                     readiness.Status,
		Mode:                       readiness.Mode,
		Scope:                      readiness.Scope,
		ProjectKey:                 readiness.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Backup:                     buildBackupManifestResponse(readiness.Backup),
		RestorePlan:                buildRestorePlanResponse(readiness.RestorePlan),
		AuditCoverage:              buildAuditCoverageResponse(readiness.AuditCoverage),
		Projects:                   make([]releaseReadinessProjectResponse, 0, len(readiness.Projects)),
		Items:                      make([]releaseReadinessItemResponse, 0, len(readiness.Items)),
		Capabilities:               jsonStringSlice(readiness.Capabilities),
		ForbiddenActions:           jsonStringSlice(readiness.ForbiddenActions),
		GeneratedAt:                formatTime(readiness.GeneratedAt),
	}
	for _, projectReadiness := range readiness.Projects {
		response.Projects = append(response.Projects, releaseReadinessProjectResponse{
			Project:             buildProjectRecordResponse(projectReadiness.Project),
			Permission:          buildPermissionPolicyDoctorResponse(projectReadiness.Permission),
			ArtifactIntegrity:   buildArtifactIntegrityResponse(projectReadiness.ArtifactIntegrity),
			Conformance:         buildConformanceResponse(projectReadiness.Conformance),
			Status:              projectReadiness.Status,
			NeedsAttentionItems: projectReadiness.NeedsAttentionItems,
			BlockedItems:        projectReadiness.BlockedItems,
		})
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, releaseReadinessItemResponse{
			Key:      item.Key,
			Category: item.Category,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return response
}

func buildReleaseRemediationPlanResponse(plan project.ReleaseRemediationPlan) releaseRemediationPlanResponse {
	guardrail := releasePreviewReal100Guardrail(plan.Real100Guardrail)
	response := releaseRemediationPlanResponse{
		Status:                     plan.Status,
		Mode:                       plan.Mode,
		Scope:                      plan.Scope,
		ProjectKey:                 plan.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Readiness:                  buildReleaseReadinessResponse(plan.Readiness),
		Actions:                    make([]releaseRemediationActionResponse, 0, len(plan.Actions)),
		Capabilities:               jsonStringSlice(plan.Capabilities),
		ForbiddenActions:           jsonStringSlice(plan.ForbiddenActions),
		GeneratedAt:                formatTime(plan.GeneratedAt),
	}
	for _, action := range plan.Actions {
		response.Actions = append(response.Actions, releaseRemediationActionResponse{
			Key:               action.Key,
			Category:          action.Category,
			Status:            action.Status,
			SourceItem:        action.SourceItem,
			RecommendedAction: action.RecommendedAction,
			Rationale:         action.Rationale,
			Owner:             action.Owner,
			NextCommand:       action.NextCommand,
			Acceptance:        action.Acceptance,
			Metadata:          action.Metadata,
		})
	}
	return response
}

func buildReleaseAcceptancePreviewResponse(preview project.ReleaseAcceptancePreview) releaseAcceptancePreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releaseAcceptancePreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Remediation:                buildReleaseRemediationPlanResponse(preview.Remediation),
		Decisions:                  make([]releaseAcceptanceDecisionResponse, 0, len(preview.Decisions)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, decision := range preview.Decisions {
		response.Decisions = append(response.Decisions, releaseAcceptanceDecisionResponse{
			Key:              decision.Key,
			SourceAction:     decision.SourceAction,
			Category:         decision.Category,
			Status:           decision.Status,
			AcceptanceType:   decision.AcceptanceType,
			Owner:            decision.Owner,
			Reason:           decision.Reason,
			RequiredEvidence: jsonStringSlice(decision.RequiredEvidence),
			NextCommand:      decision.NextCommand,
			Metadata:         decision.Metadata,
		})
	}
	return response
}

func buildReleaseAcceptanceGateResponse(gate project.ReleaseAcceptanceGate) releaseAcceptanceGateResponse {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	response := releaseAcceptanceGateResponse{
		Status:                     gate.Status,
		Mode:                       gate.Mode,
		Scope:                      gate.Scope,
		ProjectKey:                 gate.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Preview:                    buildReleaseAcceptancePreviewResponse(gate.Preview),
		Items:                      make([]releaseAcceptanceGateItemResponse, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, releaseAcceptanceGateItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			DecisionStatus:   item.DecisionStatus,
			AcceptanceType:   item.AcceptanceType,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildReleaseExceptionDoctorResponse(doctor project.ReleaseExceptionDoctor) releaseExceptionDoctorResponse {
	guardrail := releasePreviewReal100Guardrail(doctor.Real100Guardrail)
	response := releaseExceptionDoctorResponse{
		Status:                     doctor.Status,
		Mode:                       doctor.Mode,
		Scope:                      doctor.Scope,
		ProjectKey:                 doctor.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Gate:                       buildReleaseAcceptanceGateResponse(doctor.Gate),
		Checks:                     make([]releaseExceptionDoctorCheckResponse, 0, len(doctor.Checks)),
		Capabilities:               jsonStringSlice(doctor.Capabilities),
		ForbiddenActions:           jsonStringSlice(doctor.ForbiddenActions),
		GeneratedAt:                formatTime(doctor.GeneratedAt),
	}
	for _, check := range doctor.Checks {
		response.Checks = append(response.Checks, releaseExceptionDoctorCheckResponse{
			Key:      check.Key,
			Category: check.Category,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return response
}

func buildReleaseExceptionRecordPreviewResponse(preview project.ReleaseExceptionRecordPreview) releaseExceptionRecordPreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releaseExceptionRecordPreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Doctor:                     buildReleaseExceptionDoctorResponse(preview.Doctor),
		Drafts:                     make([]releaseExceptionRecordDraftResponse, 0, len(preview.Drafts)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, draft := range preview.Drafts {
		response.Drafts = append(response.Drafts, releaseExceptionRecordDraftResponse{
			Key:              draft.Key,
			SourceGateItem:   draft.SourceGateItem,
			SourceDecision:   draft.SourceDecision,
			AcceptanceType:   draft.AcceptanceType,
			Status:           draft.Status,
			Owner:            draft.Owner,
			Reason:           draft.Reason,
			RequiredEvidence: jsonStringSlice(draft.RequiredEvidence),
			AuditActions:     jsonStringSlice(draft.AuditActions),
			RollbackPlan:     draft.RollbackPlan,
			ReviewRequired:   draft.ReviewRequired,
			Metadata:         draft.Metadata,
		})
	}
	return response
}

func buildReleaseExceptionSchemaPreviewResponse(preview project.ReleaseExceptionSchemaPreview) releaseExceptionSchemaPreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releaseExceptionSchemaPreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		RecordPreview:              buildReleaseExceptionRecordPreviewResponse(preview.RecordPreview),
		Tables:                     make([]releaseExceptionSchemaTableResponse, 0, len(preview.Tables)),
		ApplySteps:                 make([]releaseExceptionMigrationStepResponse, 0, len(preview.ApplySteps)),
		RollbackSteps:              make([]releaseExceptionMigrationStepResponse, 0, len(preview.RollbackSteps)),
		AuditActions:               jsonStringSlice(preview.AuditActions),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, table := range preview.Tables {
		response.Tables = append(response.Tables, buildReleaseExceptionSchemaTableResponse(table))
	}
	for _, step := range preview.ApplySteps {
		response.ApplySteps = append(response.ApplySteps, buildReleaseExceptionMigrationStepResponse(step))
	}
	for _, step := range preview.RollbackSteps {
		response.RollbackSteps = append(response.RollbackSteps, buildReleaseExceptionMigrationStepResponse(step))
	}
	return response
}

func buildReleaseExceptionSchemaTableResponse(table project.ReleaseExceptionSchemaTable) releaseExceptionSchemaTableResponse {
	response := releaseExceptionSchemaTableResponse{
		Name:        table.Name,
		Purpose:     table.Purpose,
		Columns:     make([]releaseExceptionSchemaColumnResponse, 0, len(table.Columns)),
		Indexes:     make([]releaseExceptionSchemaIndexResponse, 0, len(table.Indexes)),
		ForeignKeys: make([]releaseExceptionSchemaForeignKeyResponse, 0, len(table.ForeignKeys)),
	}
	for _, column := range table.Columns {
		response.Columns = append(response.Columns, releaseExceptionSchemaColumnResponse{
			Name:     column.Name,
			Type:     column.Type,
			Nullable: column.Nullable,
			Purpose:  column.Purpose,
		})
	}
	for _, index := range table.Indexes {
		response.Indexes = append(response.Indexes, releaseExceptionSchemaIndexResponse{
			Name:    index.Name,
			Columns: jsonStringSlice(index.Columns),
			Unique:  index.Unique,
			Purpose: index.Purpose,
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		response.ForeignKeys = append(response.ForeignKeys, releaseExceptionSchemaForeignKeyResponse{
			Column:           foreignKey.Column,
			ReferencesTable:  foreignKey.ReferencesTable,
			ReferencesColumn: foreignKey.ReferencesColumn,
			OnDelete:         foreignKey.OnDelete,
		})
	}
	return response
}

func buildReleaseExceptionMigrationStepResponse(step project.ReleaseExceptionMigrationStep) releaseExceptionMigrationStepResponse {
	return releaseExceptionMigrationStepResponse{
		Order:       step.Order,
		Action:      step.Action,
		Description: step.Description,
		SQLPreview:  step.SQLPreview,
	}
}

func buildReleaseExceptionMigrationApprovalGateResponse(gate project.ReleaseExceptionMigrationApprovalGate) releaseExceptionMigrationApprovalGateResponse {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	response := releaseExceptionMigrationApprovalGateResponse{
		Status:                     gate.Status,
		Mode:                       gate.Mode,
		Scope:                      gate.Scope,
		ProjectKey:                 gate.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		SchemaPreview:              buildReleaseExceptionSchemaPreviewResponse(gate.SchemaPreview),
		Items:                      make([]releaseExceptionMigrationApprovalItemResponse, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, releaseExceptionMigrationApprovalItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			ApprovalStatus:   item.ApprovalStatus,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildReleaseExceptionApplyPreviewResponse(preview project.ReleaseExceptionApplyPreview) releaseExceptionApplyPreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releaseExceptionApplyPreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		MigrationGate:              buildReleaseExceptionMigrationApprovalGateResponse(preview.MigrationGate),
		Items:                      make([]releaseExceptionApplyPreviewItemResponse, 0, len(preview.Items)),
		ApplySteps:                 make([]releaseExceptionApplyPreviewStepResponse, 0, len(preview.ApplySteps)),
		RollbackSteps:              make([]releaseExceptionApplyPreviewStepResponse, 0, len(preview.RollbackSteps)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, releaseExceptionApplyPreviewItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Action:           item.Action,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	for _, step := range preview.ApplySteps {
		response.ApplySteps = append(response.ApplySteps, releaseExceptionApplyPreviewStepResponse{
			Order:       step.Order,
			Action:      step.Action,
			Description: step.Description,
			BlockedBy:   jsonStringSlice(step.BlockedBy),
		})
	}
	for _, step := range preview.RollbackSteps {
		response.RollbackSteps = append(response.RollbackSteps, releaseExceptionApplyPreviewStepResponse{
			Order:       step.Order,
			Action:      step.Action,
			Description: step.Description,
			BlockedBy:   jsonStringSlice(step.BlockedBy),
		})
	}
	return response
}

func buildReleaseFinalGateResponse(gate project.ReleaseFinalGate) releaseFinalGateResponse {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	response := releaseFinalGateResponse{
		Status:                     gate.Status,
		Mode:                       gate.Mode,
		Scope:                      gate.Scope,
		ProjectKey:                 gate.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		Readiness:                  buildReleaseReadinessResponse(gate.Readiness),
		AcceptanceGate:             buildReleaseAcceptanceGateResponse(gate.AcceptanceGate),
		ExceptionApply:             buildReleaseExceptionApplyPreviewResponse(gate.ExceptionApply),
		Items:                      make([]releaseFinalGateItemResponse, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, releaseFinalGateItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildReleaseEvidenceBundleResponse(bundle project.ReleaseEvidenceBundle) releaseEvidenceBundleResponse {
	guardrail := releasePreviewReal100Guardrail(bundle.Real100Guardrail)
	response := releaseEvidenceBundleResponse{
		Status:                     bundle.Status,
		Mode:                       bundle.Mode,
		Scope:                      bundle.Scope,
		ProjectKey:                 bundle.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		BundleHash:                 bundle.BundleHash,
		FinalGate:                  buildReleaseFinalGateResponse(bundle.FinalGate),
		Backup:                     buildBackupManifestResponse(bundle.Backup),
		AuditCoverage:              buildAuditCoverageResponse(bundle.AuditCoverage),
		Items:                      make([]releaseEvidenceBundleItemResponse, 0, len(bundle.Items)),
		Capabilities:               jsonStringSlice(bundle.Capabilities),
		ForbiddenActions:           jsonStringSlice(bundle.ForbiddenActions),
		GeneratedAt:                formatTime(bundle.GeneratedAt),
	}
	for _, item := range bundle.Items {
		response.Items = append(response.Items, releaseEvidenceBundleItemResponse{
			Key:         item.Key,
			Category:    item.Category,
			Status:      item.Status,
			Source:      item.Source,
			Description: item.Description,
			Metadata:    item.Metadata,
		})
	}
	return response
}

func buildReleasePackagePreviewResponse(preview project.ReleasePackagePreview) releasePackagePreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releasePackagePreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		EvidenceBundle:             buildReleaseEvidenceBundleResponse(preview.EvidenceBundle),
		PackageName:                preview.PackageName,
		Items:                      make([]releasePackagePreviewItemResponse, 0, len(preview.Items)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, releasePackagePreviewItemResponse{
			Key:         item.Key,
			Category:    item.Category,
			Status:      item.Status,
			PackagePath: item.PackagePath,
			Source:      item.Source,
			Description: item.Description,
			Metadata:    item.Metadata,
		})
	}
	return response
}

func buildReleaseDistributionPreviewResponse(preview project.ReleaseDistributionPreview) releaseDistributionPreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releaseDistributionPreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		PackagePreview:             buildReleasePackagePreviewResponse(preview.PackagePreview),
		Items:                      make([]releaseDistributionPreviewItemResponse, 0, len(preview.Items)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, releaseDistributionPreviewItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Channel:          item.Channel,
			Action:           item.Action,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildReleasePublishGateResponse(gate project.ReleasePublishGate) releasePublishGateResponse {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	response := releasePublishGateResponse{
		Status:                     gate.Status,
		Mode:                       gate.Mode,
		Scope:                      gate.Scope,
		ProjectKey:                 gate.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		DistributionPreview:        buildReleaseDistributionPreviewResponse(gate.DistributionPreview),
		Items:                      make([]releasePublishGateItemResponse, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, releasePublishGateItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Channel:          item.Channel,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildReleasePublishApprovalPreviewResponse(preview project.ReleasePublishApprovalPreview) releasePublishApprovalPreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releasePublishApprovalPreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		PublishGate:                buildReleasePublishGateResponse(preview.PublishGate),
		Items:                      make([]releasePublishApprovalPreviewItemResponse, 0, len(preview.Items)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, releasePublishApprovalPreviewItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			ApprovalStatus:   item.ApprovalStatus,
			Channel:          item.Channel,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return response
}

func buildReleaseRolloutPlanPreviewResponse(preview project.ReleaseRolloutPlanPreview) releaseRolloutPlanPreviewResponse {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	response := releaseRolloutPlanPreviewResponse{
		Status:                     preview.Status,
		Mode:                       preview.Mode,
		Scope:                      preview.Scope,
		ProjectKey:                 preview.ProjectKey,
		ReadinessScope:             guardrail.ReadinessScope,
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:           guardrail.Real100Breakdown,
		PublishApprovalPreview:     buildReleasePublishApprovalPreviewResponse(preview.PublishApprovalPreview),
		Items:                      make([]releaseRolloutPlanPreviewItemResponse, 0, len(preview.Items)),
		RolloutSteps:               make([]releaseRolloutPlanPreviewStepResponse, 0, len(preview.RolloutSteps)),
		VerificationCheckpoints:    make([]releaseRolloutPlanPreviewStepResponse, 0, len(preview.VerificationCheckpoints)),
		RollbackSteps:              make([]releaseRolloutPlanPreviewStepResponse, 0, len(preview.RollbackSteps)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, releaseRolloutPlanPreviewItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Stage:            item.Stage,
			Action:           item.Action,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	for _, step := range preview.RolloutSteps {
		response.RolloutSteps = append(response.RolloutSteps, buildReleaseRolloutPlanPreviewStepResponse(step))
	}
	for _, checkpoint := range preview.VerificationCheckpoints {
		response.VerificationCheckpoints = append(response.VerificationCheckpoints, buildReleaseRolloutPlanPreviewStepResponse(checkpoint))
	}
	for _, step := range preview.RollbackSteps {
		response.RollbackSteps = append(response.RollbackSteps, buildReleaseRolloutPlanPreviewStepResponse(step))
	}
	return response
}

func buildReleaseRolloutPlanPreviewStepResponse(step project.ReleaseRolloutPlanPreviewStep) releaseRolloutPlanPreviewStepResponse {
	return releaseRolloutPlanPreviewStepResponse{
		Order:       step.Order,
		Stage:       step.Stage,
		Action:      step.Action,
		Description: step.Description,
		BlockedBy:   jsonStringSlice(step.BlockedBy),
	}
}

func buildWorkerPoolSummaryResponse(summary project.WorkerPoolSummary) workerPoolSummaryResponse {
	response := workerPoolSummaryResponse{
		Projects:           make([]workerPoolProjectSummaryResponse, 0, len(summary.Projects)),
		TotalProjects:      summary.TotalProjects,
		TotalWorkers:       summary.TotalWorkers,
		TotalOnlineWorkers: summary.TotalOnlineWorkers,
		TotalActiveLeases:  summary.TotalActiveLeases,
		TotalQueuedTasks:   summary.TotalQueuedTasks,
		TotalNeedsRecovery: summary.TotalNeedsRecovery,
		GeneratedAt:        formatTime(summary.GeneratedAt),
	}
	for _, projectSummary := range summary.Projects {
		response.Projects = append(response.Projects, buildWorkerPoolProjectSummaryResponse(projectSummary))
	}
	return response
}

func buildWorkerPoolProjectSummaryResponse(summary project.WorkerPoolProjectSummary) workerPoolProjectSummaryResponse {
	response := workerPoolProjectSummaryResponse{
		Project:             buildProjectRecordResponse(summary.Project),
		Workers:             summary.Workers,
		OnlineWorkers:       summary.OnlineWorkers,
		OfflineWorkers:      summary.OfflineWorkers,
		ActiveLeases:        summary.ActiveLeases,
		NeedsRecoveryLeases: summary.NeedsRecoveryLeases,
		QueuedTasks:         summary.QueuedTasks,
		NeedsRecoveryTasks:  summary.NeedsRecoveryTasks,
		Capabilities:        jsonStringSlice(summary.Capabilities),
		WorkerTypes:         jsonStringSlice(summary.WorkerTypes),
		Scheduling:          buildSchedulingPolicyResponse(summary.Scheduling),
		Role:                buildRoleReadinessResponse(summary.Role),
		Engine:              buildEngineReadinessResponse(summary.Engine),
		Resources:           buildResourceReadinessResponse(summary.Resources),
	}
	if summary.LastWorkerHeartbeat != nil {
		response.LastWorkerHeartbeat = formatTime(*summary.LastWorkerHeartbeat)
	}
	return response
}

func buildSchedulingPolicyResponse(policy project.SchedulingPolicy) schedulingPolicyResponse {
	return schedulingPolicyResponse{
		Priority:             policy.Priority,
		MaxParallelTasks:     policy.MaxParallelTasks,
		AgentRole:            policy.AgentRole,
		RequiredCapabilities: jsonStringSlice(policy.RequiredCapabilities),
		EngineProfile:        policy.EngineProfile,
	}
}

func buildRoleReadinessResponse(readiness project.RoleReadiness) roleReadinessResponse {
	return roleReadinessResponse{
		RequiredRole:   readiness.RequiredRole,
		Matched:        readiness.Matched,
		MatchedTypes:   jsonStringSlice(readiness.MatchedTypes),
		Status:         readiness.Status,
		BlockedReasons: jsonStringSlice(readiness.BlockedReasons),
	}
}

func buildEngineReadinessResponse(readiness project.EngineReadiness) engineReadinessResponse {
	return engineReadinessResponse{
		ProfileID:      readiness.ProfileID,
		Provider:       readiness.Provider,
		Enabled:        readiness.Enabled,
		SecretRef:      readiness.SecretRef,
		SecretRequired: readiness.SecretRequired,
		SecretReady:    readiness.SecretReady,
		ResourceLimits: jsonObject(readiness.ResourceLimits),
		Status:         readiness.Status,
		BlockedReasons: jsonStringSlice(readiness.BlockedReasons),
	}
}

func buildResourceReadinessResponse(readiness project.ResourceReadiness) resourceReadinessResponse {
	return resourceReadinessResponse{
		MaxActiveLeases: readiness.MaxActiveLeases,
		MaxQueuedTasks:  readiness.MaxQueuedTasks,
		Status:          readiness.Status,
		BlockedReasons:  jsonStringSlice(readiness.BlockedReasons),
	}
}

func buildWorkerPoolSchedulePreviewResponse(preview project.WorkerPoolSchedulePreview) workerPoolSchedulePreviewResponse {
	response := workerPoolSchedulePreviewResponse{
		Projects:      make([]workerPoolProjectScheduleResponse, 0, len(preview.Projects)),
		Policy:        buildWorkerPoolSchedulePolicyResponse(preview.Policy),
		GeneratedAt:   formatTime(preview.GeneratedAt),
		Recommended:   preview.Recommended,
		Blocked:       preview.Blocked,
		QueuedTasks:   preview.QueuedTasks,
		AvailableSlot: preview.AvailableSlot,
	}
	for _, projectSchedule := range preview.Projects {
		response.Projects = append(response.Projects, buildWorkerPoolProjectScheduleResponse(projectSchedule))
	}
	return response
}

func buildWorkerPoolSchedulePolicyResponse(policy project.WorkerPoolSchedulePolicy) workerPoolSchedulePolicyResponse {
	return workerPoolSchedulePolicyResponse{
		Strategy:               policy.Strategy,
		DefaultProjectPriority: policy.DefaultProjectPriority,
		SlotStrategy:           policy.SlotStrategy,
		DryRunOnly:             policy.DryRunOnly,
	}
}

func buildWorkerPoolProjectScheduleResponse(schedule project.WorkerPoolProjectSchedule) workerPoolProjectScheduleResponse {
	return workerPoolProjectScheduleResponse{
		Project:        buildProjectRecordResponse(schedule.Project),
		Priority:       schedule.Priority,
		MaxParallel:    schedule.MaxParallel,
		AgentRole:      schedule.AgentRole,
		Role:           buildRoleReadinessResponse(schedule.Role),
		EngineProfile:  schedule.EngineProfile,
		Engine:         buildEngineReadinessResponse(schedule.Engine),
		Resources:      buildResourceReadinessResponse(schedule.Resources),
		QueuedTasks:    schedule.QueuedTasks,
		ActiveLeases:   schedule.ActiveLeases,
		OnlineWorkers:  schedule.OnlineWorkers,
		AvailableSlots: schedule.AvailableSlots,
		NeedsRecovery:  schedule.NeedsRecovery,
		Capabilities:   jsonStringSlice(schedule.Capabilities),
		RequiredCaps:   jsonStringSlice(schedule.RequiredCaps),
		Recommended:    schedule.Recommended,
		BlockedReasons: jsonStringSlice(schedule.BlockedReasons),
		NextAction:     schedule.NextAction,
	}
}

func buildCodexCLIAdapterPreviewResponse(preview project.CodexCLIAdapterPreview) codexCLIAdapterPreviewResponse {
	response := codexCLIAdapterPreviewResponse{
		Project:                 buildProjectRecordResponse(preview.Project),
		Status:                  preview.Status,
		Mode:                    preview.Mode,
		Engine:                  buildEngineReadinessResponse(preview.Engine),
		Command:                 buildEngineCommandPreviewResponse(preview.Command),
		Capabilities:            make([]engineCapabilityPreflightResponse, 0, len(preview.Capabilities)),
		Paths:                   make([]enginePathPreflightResponse, 0, len(preview.Paths)),
		ArtifactRedaction:       buildArtifactRedactionPlanResponse(preview.ArtifactRedaction),
		ForbiddenActions:        jsonStringSlice(preview.ForbiddenActions),
		Blockers:                jsonStringSlice(preview.Blockers),
		ExecutionAllowed:        preview.ExecutionAllowed,
		ProjectWriteAttempted:   preview.ProjectWriteAttempted,
		ExecutionWriteAttempted: preview.ExecutionWriteAttempted,
		EngineCallAttempted:     preview.EngineCallAttempted,
		CommandsRun:             preview.CommandsRun,
		SecretsResolved:         preview.SecretsResolved,
		NetworkUsed:             preview.NetworkUsed,
		GeneratedAt:             formatTime(preview.GeneratedAt),
	}
	for _, capability := range preview.Capabilities {
		response.Capabilities = append(response.Capabilities, engineCapabilityPreflightResponse{
			Capability: capability.Capability,
			Required:   capability.Required,
			Allowed:    capability.Allowed,
			Reason:     capability.Reason,
		})
	}
	for _, path := range preview.Paths {
		response.Paths = append(response.Paths, enginePathPreflightResponse{
			Path:       path.Path,
			Capability: path.Capability,
			Effect:     path.Effect,
			Allowed:    path.Allowed,
			Reason:     path.Reason,
		})
	}
	return response
}

func buildEngineCommandPreviewResponse(command project.EngineCommandPreview) engineCommandPreviewResponse {
	return engineCommandPreviewResponse{
		Command:           command.Command,
		Allowed:           command.Allowed,
		Reason:            command.Reason,
		CapabilityAllowed: command.CapabilityAllowed,
		CommandAllowed:    command.CommandAllowed,
		Denied:            command.Denied,
	}
}

func buildArtifactRedactionPlanResponse(plan project.ArtifactRedactionPlan) artifactRedactionPlanResponse {
	return artifactRedactionPlanResponse{
		Status:         plan.Status,
		RetentionClass: plan.RetentionClass,
		Rules:          jsonStringSlice(plan.Rules),
		RedactedFields: jsonStringSlice(plan.RedactedFields),
	}
}

func jsonStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func completionAuditReal100Guardrail(guardrail project.Real100Guardrail) project.Real100Guardrail {
	return project.NormalizeReal100Guardrail(guardrail, project.CompletionAuditReal100Guardrail())
}

func releasePreviewReal100Guardrail(guardrail project.Real100Guardrail) project.Real100Guardrail {
	return project.NormalizeReal100Guardrail(guardrail, project.ReleasePreviewReal100Guardrail())
}

func jsonObject(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func buildWorkerResponse(worker project.WorkerRecord) workerResponse {
	response := workerResponse{
		ID:                       worker.ID,
		ProjectID:                worker.ProjectID,
		ActorID:                  worker.ActorID,
		WorkerKey:                worker.WorkerKey,
		WorkerType:               worker.WorkerType,
		Status:                   worker.Status,
		Hostname:                 worker.Hostname,
		PID:                      worker.PID,
		Capabilities:             worker.Capabilities,
		Metadata:                 worker.Metadata,
		RegisteredAt:             formatTime(worker.RegisteredAt),
		HeartbeatIntervalSeconds: worker.HeartbeatIntervalSeconds,
		LeaseTimeoutSeconds:      worker.LeaseTimeoutSeconds,
		UpdatedAt:                formatTime(worker.UpdatedAt),
	}
	if worker.LastHeartbeatAt != nil {
		response.LastHeartbeatAt = formatTime(*worker.LastHeartbeatAt)
	}
	return response
}

func buildLeaseRecoverResponse(record project.Record, leases []project.LeaseRecord) leaseRecoverResponse {
	response := leaseRecoverResponse{
		Project: buildProjectRecordResponse(record),
		Leases:  make([]leaseResponse, 0, len(leases)),
	}
	for _, lease := range leases {
		response.Leases = append(response.Leases, buildLeaseResponse(lease))
	}
	return response
}

func buildWorkerRunOnceResponse(result project.WorkerRunOnceResult) workerRunOnceResponse {
	response := workerRunOnceResponse{
		Project: buildProjectRecordResponse(result.Project),
		Worker:  buildWorkerResponse(result.Worker),
		Claimed: result.Claimed,
	}
	if result.Claimed {
		lease := buildLeaseResponse(result.Lease)
		task := buildRunTaskResponse(result.Task)
		attempt := buildRunAttemptResponse(result.Attempt)
		artifact := buildArtifactResponse(result.Artifact)
		response.Lease = &lease
		response.Task = &task
		response.Attempt = &attempt
		response.Artifact = &artifact
	}
	return response
}

func buildFixtureExecutionResponse(result project.FixtureExecutionResult) fixtureExecutionResponse {
	return fixtureExecutionResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Worker:                        buildWorkerResponse(result.Worker),
		Lease:                         buildLeaseResponse(result.Lease),
		Task:                          buildRunTaskResponse(result.Task),
		Attempt:                       buildRunAttemptResponse(result.Attempt),
		Artifact:                      buildArtifactResponse(result.Artifact),
		Gate:                          buildExecutionApprovalGateResponse(result.Gate),
		Status:                        result.Status,
		Decision:                      result.Decision,
		Message:                       result.Message,
		Blockers:                      result.Blockers,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
		TaskClaimed:                   result.TaskClaimed,
		WorkerStarted:                 result.WorkerStarted,
		LeaseCreated:                  result.LeaseCreated,
		AttemptCreated:                result.AttemptCreated,
		ArtifactCreated:               result.ArtifactCreated,
	}
}

func buildReadOnlyVerifyResponse(result project.ReadOnlyVerifyResult) readOnlyVerifyResponse {
	return readOnlyVerifyResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Worker:                        buildWorkerResponse(result.Worker),
		Lease:                         buildLeaseResponse(result.Lease),
		Task:                          buildRunTaskResponse(result.Task),
		Attempt:                       buildRunAttemptResponse(result.Attempt),
		Artifact:                      buildArtifactResponse(result.Artifact),
		Gate:                          buildExecutionApprovalGateResponse(result.Gate),
		TargetPath:                    result.TargetPath,
		TargetSHA256:                  result.TargetSHA256,
		TargetSizeBytes:               result.TargetSizeBytes,
		Status:                        result.Status,
		Decision:                      result.Decision,
		Message:                       result.Message,
		Blockers:                      result.Blockers,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectReadAllowed:            result.ProjectReadAllowed,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
		TaskClaimed:                   result.TaskClaimed,
		WorkerStarted:                 result.WorkerStarted,
		LeaseCreated:                  result.LeaseCreated,
		AttemptCreated:                result.AttemptCreated,
		ArtifactCreated:               result.ArtifactCreated,
		VerificationPassed:            result.VerificationPassed,
	}
}

func buildApprovedArtifactWriteResponse(result project.ApprovedArtifactWriteResult) approvedArtifactWriteResponse {
	return approvedArtifactWriteResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Worker:                        buildWorkerResponse(result.Worker),
		Lease:                         buildLeaseResponse(result.Lease),
		Task:                          buildRunTaskResponse(result.Task),
		Attempt:                       buildRunAttemptResponse(result.Attempt),
		Artifact:                      buildArtifactResponse(result.Artifact),
		Gate:                          buildExecutionApprovalGateResponse(result.Gate),
		ArtifactLabel:                 result.ArtifactLabel,
		Status:                        result.Status,
		Decision:                      result.Decision,
		Message:                       result.Message,
		Blockers:                      result.Blockers,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       result.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
		TaskClaimed:                   result.TaskClaimed,
		WorkerStarted:                 result.WorkerStarted,
		LeaseCreated:                  result.LeaseCreated,
		AttemptCreated:                result.AttemptCreated,
		ArtifactCreated:               result.ArtifactCreated,
		ArtifactWritePassed:           result.ArtifactWritePassed,
	}
}

func buildFixtureProjectWriteResponse(result project.FixtureProjectWriteResult) fixtureProjectWriteResponse {
	return fixtureProjectWriteResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Worker:                        buildWorkerResponse(result.Worker),
		Lease:                         buildLeaseResponse(result.Lease),
		Task:                          buildRunTaskResponse(result.Task),
		CopyAttempt:                   buildRunAttemptResponse(result.CopyAttempt),
		VerifyAttempt:                 buildRunAttemptResponse(result.VerifyAttempt),
		RollbackAttempt:               buildRunAttemptResponse(result.RollbackAttempt),
		WriteSetArtifact:              buildArtifactResponse(result.WriteSetArtifact),
		PreimageArtifact:              buildArtifactResponse(result.PreimageArtifact),
		Artifact:                      buildArtifactResponse(result.Artifact),
		Gate:                          buildExecutionApprovalGateResponse(result.Gate),
		TargetPath:                    result.TargetPath,
		ExpectedBeforeSHA256:          result.ExpectedBeforeSHA256,
		ExpectedBeforeSize:            result.ExpectedBeforeSize,
		AfterSHA256:                   result.AfterSHA256,
		AfterSize:                     result.AfterSize,
		RestoredSHA256:                result.RestoredSHA256,
		RestoredSize:                  result.RestoredSize,
		Status:                        result.Status,
		Decision:                      result.Decision,
		Message:                       result.Message,
		Blockers:                      result.Blockers,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectReadAllowed:            result.ProjectReadAllowed,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ProjectWriteAllowed:           result.ProjectWriteAllowed,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       result.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
		TaskClaimed:                   result.TaskClaimed,
		WorkerStarted:                 result.WorkerStarted,
		LeaseCreated:                  result.LeaseCreated,
		AttemptCreated:                result.AttemptCreated,
		ArtifactCreated:               result.ArtifactCreated,
		WriteSetPassed:                result.WriteSetPassed,
		VerificationPassed:            result.VerificationPassed,
		RollbackAttempted:             result.RollbackAttempted,
		RollbackVerified:              result.RollbackVerified,
	}
}

func buildManagedGeneratedWriteResponse(result project.ManagedGeneratedWriteResult) managedGeneratedWriteResponse {
	return managedGeneratedWriteResponse{
		Project:                       buildProjectRecordResponse(result.Project),
		WorkflowVersion:               buildWorkflowVersionResponse(result.Version),
		Run:                           buildRunResponse(result.Run),
		Worker:                        buildWorkerResponse(result.Worker),
		Lease:                         buildLeaseResponse(result.Lease),
		Task:                          buildRunTaskResponse(result.Task),
		CopyAttempt:                   buildRunAttemptResponse(result.CopyAttempt),
		VerifyAttempt:                 buildRunAttemptResponse(result.VerifyAttempt),
		RollbackAttempt:               buildRunAttemptResponse(result.RollbackAttempt),
		WriteSetArtifact:              buildArtifactResponse(result.WriteSetArtifact),
		PreimageArtifact:              buildArtifactResponse(result.PreimageArtifact),
		Artifact:                      buildArtifactResponse(result.Artifact),
		Gate:                          buildExecutionApprovalGateResponse(result.Gate),
		TargetPath:                    result.TargetPath,
		ExpectedBeforeSHA256:          result.ExpectedBeforeSHA256,
		ExpectedBeforeSize:            result.ExpectedBeforeSize,
		AfterSHA256:                   result.AfterSHA256,
		AfterSize:                     result.AfterSize,
		RestoredSHA256:                result.RestoredSHA256,
		RestoredSize:                  result.RestoredSize,
		Status:                        result.Status,
		Decision:                      result.Decision,
		Message:                       result.Message,
		Blockers:                      result.Blockers,
		Created:                       result.Created,
		IdempotencyKey:                result.IdempotencyKey,
		EventID:                       result.EventID,
		AuditEventID:                  result.AuditEventID,
		GeneratedOnly:                 result.GeneratedOnly,
		GeneratedOnlyApplyOpen:        result.GeneratedOnlyApplyOpen,
		ProjectReadAttempted:          result.ProjectReadAttempted,
		ProjectReadAllowed:            result.ProjectReadAllowed,
		ProjectWriteAttempted:         result.ProjectWriteAttempted,
		ProjectWriteAllowed:           result.ProjectWriteAllowed,
		ExecutionWriteAttempted:       result.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       result.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: result.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           result.EngineCallAttempted,
		CommandsRun:                   result.CommandsRun,
		SecretsResolved:               result.SecretsResolved,
		NetworkUsed:                   result.NetworkUsed,
		TaskClaimed:                   result.TaskClaimed,
		WorkerStarted:                 result.WorkerStarted,
		LeaseCreated:                  result.LeaseCreated,
		AttemptCreated:                result.AttemptCreated,
		ArtifactCreated:               result.ArtifactCreated,
		WriteSetPassed:                result.WriteSetPassed,
		VerificationPassed:            result.VerificationPassed,
		RollbackAttempted:             result.RollbackAttempted,
		RollbackVerified:              result.RollbackVerified,
	}
}

func buildLeaseResponse(lease project.LeaseRecord) leaseResponse {
	response := leaseResponse{
		ID:                  lease.ID,
		ProjectID:           lease.ProjectID,
		RunID:               lease.RunID,
		RunTaskID:           lease.RunTaskID,
		WorkflowItemID:      lease.WorkflowItemID,
		WorkerID:            lease.WorkerID,
		LeaseKind:           lease.LeaseKind,
		Status:              lease.Status,
		AcquiredAt:          formatTime(lease.AcquiredAt),
		ExpiresAt:           formatTime(lease.ExpiresAt),
		AllowedCapabilities: lease.AllowedCapabilities,
		Scope:               lease.Scope,
		Metadata:            lease.Metadata,
	}
	if lease.HeartbeatAt != nil {
		response.HeartbeatAt = formatTime(*lease.HeartbeatAt)
	}
	if lease.ReleasedAt != nil {
		response.ReleasedAt = formatTime(*lease.ReleasedAt)
	}
	return response
}

func buildWorkflowVersionResponse(version project.WorkflowVersion) workflowVersionResponse {
	response := workflowVersionResponse{
		ID:              version.ID,
		DisplayLabel:    version.DisplayLabel,
		VersionKind:     version.VersionKind,
		LifecycleStatus: version.LifecycleStatus,
		SourcePath:      version.SourcePath,
		SourceHash:      version.SourceHash,
		ImportMode:      version.ImportMode,
		Immutable:       version.Immutable,
		StatusSummary:   version.StatusSummary,
		CreatedAt:       formatTime(version.CreatedAt),
		UpdatedAt:       formatTime(version.UpdatedAt),
	}
	if version.ImportedAt != nil {
		response.ImportedAt = formatTime(*version.ImportedAt)
	}
	return response
}

func buildWorkflowItemResponse(item project.WorkflowItem) workflowItemResponse {
	response := workflowItemResponse{
		ID:                item.ID,
		WorkflowVersionID: item.WorkflowVersionID,
		Stage:             item.Stage,
		ItemType:          item.ItemType,
		ExternalKey:       item.ExternalKey,
		Title:             item.Title,
		Status:            item.Status,
		Metadata:          item.Metadata,
		CreatedAt:         formatTime(item.CreatedAt),
		UpdatedAt:         formatTime(item.UpdatedAt),
	}
	if item.ImportedAt != nil {
		response.ImportedAt = formatTime(*item.ImportedAt)
	}
	return response
}

func buildWorkflowItemLinkResponse(link project.WorkflowItemLink) workflowItemLinkResponse {
	return workflowItemLinkResponse{
		ID:                link.ID,
		WorkflowVersionID: link.WorkflowVersionID,
		FromItemID:        link.FromItemID,
		ToItemID:          link.ToItemID,
		RelationType:      link.RelationType,
		Metadata:          link.Metadata,
		CreatedAt:         formatTime(link.CreatedAt),
	}
}

func buildProjectRecordResponse(record project.Record) projectRecordResponse {
	return projectRecordResponse{
		Key:             record.Key,
		Name:            record.Name,
		Kind:            record.Kind,
		Adapter:         record.Adapter,
		WorkflowProfile: record.WorkflowProfile,
		DefaultBranch:   record.DefaultBranch,
		Root:            record.RootPath,
	}
}

func buildProjectListResponse(records []project.Record) projectListResponse {
	response := projectListResponse{
		Projects: make([]projectRecordResponse, 0, len(records)),
	}
	for _, record := range records {
		response.Projects = append(response.Projects, buildProjectRecordResponse(record))
	}
	return response
}

func buildProjectSummaryResponse(summary project.ProjectSummary) projectSummaryResponse {
	response := projectSummaryResponse{
		Project: buildProjectRecordResponse(summary.Project),
		Inventory: projectInventoryResponse{
			Versions:        summary.Inventory.Versions,
			Residuals:       summary.Inventory.Residuals,
			Artifacts:       summary.Inventory.Artifacts,
			ImportSnapshots: summary.Inventory.ImportSnapshots,
			MirrorExports:   summary.Inventory.MirrorExports,
		},
	}
	if summary.HasConfig {
		response.Config = &projectConfigResponse{
			ProtocolVersion: summary.Config.ProtocolVersion,
			ConfigPath:      summary.Config.ConfigPath,
			ConfigHash:      summary.Config.ConfigHash,
			Ownership:       summary.Config.Ownership,
			StatusExport:    summary.Config.StatusExport,
			Migration:       summary.Config.Migration,
			LoadedAt:        summary.Config.LoadedAt.UTC().Format(time.RFC3339),
		}
	}
	if summary.HasImport {
		response.Import = &projectImportResponse{
			SourceHash:             summary.Import.SourceHash,
			CreatedAt:              summary.Import.CreatedAt.UTC().Format(time.RFC3339),
			Summary:                summary.Import.Summary,
			HasPrevious:            summary.HasPreviousImport,
			HistoryReadyForDiff:    summary.HasPreviousImport,
			SourceHashChangedSince: summary.HasPreviousImport && summary.Import.SourceHash != summary.PreviousImport.SourceHash,
		}
		if summary.HasPreviousImport {
			response.Import.PreviousSourceHash = summary.PreviousImport.SourceHash
			response.Import.PreviousCreatedAt = summary.PreviousImport.CreatedAt.UTC().Format(time.RFC3339)
		}
	}
	if summary.HasLatestDoctor {
		response.Doctor = &projectDoctorResponse{
			Status:              summary.DoctorStatus,
			DriftStatus:         summary.DriftStatus,
			ConfigDriftStatus:   summary.ConfigDriftStatus,
			StageCoverageStatus: summary.StageCoverageStatus,
			NativeDoctorStatus:  summary.NativeDoctorStatus,
			Severity:            summary.LatestDoctor.Severity,
			CreatedAt:           summary.LatestDoctor.CreatedAt.UTC().Format(time.RFC3339),
			Metadata:            summary.LatestDoctor.Metadata,
		}
	}
	return response
}

func buildProjectDoctorRunResponse(record project.Record, report doctor.Report, result project.RecordDoctorReportResult) projectDoctorRunResponse {
	return projectDoctorRunResponse{
		Project:        buildProjectRecordResponse(record),
		Report:         report.Summary(),
		EventID:        result.EventID,
		Severity:       result.Severity,
		OverallStatus:  result.OverallStatus,
		IdempotencyKey: result.IdempotencyKey,
		Created:        result.Created,
	}
}

func buildProjectImportRunResponse(record project.Record, result importer.Result) projectImportRunResponse {
	return projectImportRunResponse{
		Project:        buildProjectRecordResponse(record),
		Versions:       result.Versions,
		Residuals:      result.Residuals,
		Artifacts:      result.Artifacts,
		ActiveTasks:    result.ActiveTasks,
		V1Done:         result.V1Done,
		V1Total:        result.V1Total,
		StatusSnapshot: result.StatusSnapshot,
		RunID:          result.RunID,
		IdempotencyKey: result.IdempotencyKey,
		Created:        result.Created,
	}
}

func buildProjectReadinessResponse(readiness project.ProjectReadiness) projectReadinessResponse {
	response := projectReadinessResponse{
		Project: buildProjectRecordResponse(readiness.Project),
		Status:  readiness.Status,
		Items:   make([]projectReadinessItemResponse, 0, len(readiness.Items)),
		Summary: buildProjectSummaryResponse(readiness.Summary),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return response
}

func buildGeneratedWriteReadinessResponse(readiness project.GeneratedWriteReadiness) generatedWriteReadinessResponse {
	response := generatedWriteReadinessResponse{
		Project:                       buildProjectRecordResponse(readiness.Project),
		Status:                        readiness.Status,
		Mode:                          readiness.Mode,
		Items:                         make([]projectReadinessItemResponse, 0, len(readiness.Items)),
		RequiredCapabilities:          jsonStringSlice(readiness.RequiredCapabilities),
		AllowedGeneratedPrefixes:      jsonStringSlice(readiness.AllowedGeneratedPrefixes),
		RequiredWritePaths:            jsonStringSlice(readiness.RequiredWritePaths),
		ConfiguredWritePaths:          jsonStringSlice(readiness.ConfiguredWritePaths),
		ConfiguredForbiddenPaths:      jsonStringSlice(readiness.ConfiguredForbiddenPaths),
		Blockers:                      jsonStringSlice(readiness.Blockers),
		ReviewBlockers:                jsonStringSlice(readiness.ReviewBlockers),
		ForbiddenActions:              jsonStringSlice(readiness.ForbiddenActions),
		ReadyForReview:                readiness.ReadyForReview,
		ApplyOpen:                     readiness.ApplyOpen,
		RealAreaMatrixWriteOpened:     readiness.RealAreaMatrixWriteOpened,
		GeneratedOnly:                 readiness.GeneratedOnly,
		ProjectConfigRead:             readiness.ProjectConfigRead,
		ProjectReadAttempted:          readiness.ProjectReadAttempted,
		ProjectWriteAttempted:         readiness.ProjectWriteAttempted,
		ExecutionWriteAttempted:       readiness.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       readiness.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: readiness.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           readiness.EngineCallAttempted,
		CommandsRun:                   readiness.CommandsRun,
		SecretsResolved:               readiness.SecretsResolved,
		NetworkUsed:                   readiness.NetworkUsed,
		TaskClaimed:                   readiness.TaskClaimed,
		WorkerStarted:                 readiness.WorkerStarted,
		LeaseCreated:                  readiness.LeaseCreated,
		AttemptCreated:                readiness.AttemptCreated,
		ArtifactCreated:               readiness.ArtifactCreated,
		GeneratedAt:                   formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: jsonObject(item.Metadata),
		})
	}
	return response
}

func buildGeneratedWriteApplyBetaGateResponse(gate project.GeneratedWriteApplyBetaGate) generatedWriteApplyBetaGateResponse {
	response := generatedWriteApplyBetaGateResponse{
		Project:                       buildProjectRecordResponse(gate.Project),
		Status:                        gate.Status,
		Mode:                          gate.Mode,
		Readiness:                     buildGeneratedWriteReadinessResponse(gate.Readiness),
		Items:                         make([]generatedWriteApplyBetaGateItemResponse, 0, len(gate.Items)),
		RequiredCapabilities:          jsonStringSlice(gate.RequiredCapabilities),
		AllowedGeneratedPrefixes:      jsonStringSlice(gate.AllowedGeneratedPrefixes),
		RequiredEvidence:              jsonStringSlice(gate.RequiredEvidence),
		ForbiddenActions:              jsonStringSlice(gate.ForbiddenActions),
		ApprovalRequired:              gate.ApprovalRequired,
		ApprovalStatus:                gate.ApprovalStatus,
		ApplyOpen:                     gate.ApplyOpen,
		RealAreaMatrixWriteOpened:     gate.RealAreaMatrixWriteOpened,
		GeneratedOnly:                 gate.GeneratedOnly,
		ProjectReadAttempted:          gate.ProjectReadAttempted,
		ProjectWriteAttempted:         gate.ProjectWriteAttempted,
		ExecutionWriteAttempted:       gate.ExecutionWriteAttempted,
		AreaFlowArtifactWritten:       gate.AreaFlowArtifactWritten,
		AreaFlowExecutionStateWritten: gate.AreaFlowExecutionStateWritten,
		EngineCallAttempted:           gate.EngineCallAttempted,
		CommandsRun:                   gate.CommandsRun,
		SecretsResolved:               gate.SecretsResolved,
		NetworkUsed:                   gate.NetworkUsed,
		TaskClaimed:                   gate.TaskClaimed,
		WorkerStarted:                 gate.WorkerStarted,
		LeaseCreated:                  gate.LeaseCreated,
		AttemptCreated:                gate.AttemptCreated,
		ArtifactCreated:               gate.ArtifactCreated,
		GeneratedAt:                   formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, generatedWriteApplyBetaGateItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			ApprovalStatus:   item.ApprovalStatus,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         jsonObject(item.Metadata),
		})
	}
	return response
}

func buildProjectVerificationBundleResponse(bundle project.ProjectVerificationBundle) projectVerificationBundleResponse {
	response := projectVerificationBundleResponse{
		Project: buildProjectRecordResponse(bundle.Project),
		Status:  bundle.Status,
		PhaseGate: projectPhaseGateResponse{
			Name:             bundle.PhaseGate.Name,
			Status:           bundle.PhaseGate.Status,
			AcceptedWarnings: bundle.PhaseGate.AcceptedWarnings,
			Blockers:         bundle.PhaseGate.Blockers,
		},
		Summary:    buildProjectSummaryResponse(bundle.Summary),
		Readiness:  buildProjectReadinessResponse(bundle.Readiness),
		ImportDiff: buildProjectImportDiffResponse(bundle.ImportDiff),
		Events:     make([]eventResponse, 0, len(bundle.Events)),
	}
	for _, event := range bundle.Events {
		response.Events = append(response.Events, buildEventResponse(event))
	}
	return response
}

func buildCompatibilityContractResponse(contract project.CompatibilityContract) compatibilityContractResponse {
	response := compatibilityContractResponse{
		Project:  buildProjectRecordResponse(contract.Project),
		Status:   contract.Status,
		Commands: make([]compatibilityCommandResponse, 0, len(contract.Commands)),
	}
	for _, command := range contract.Commands {
		response.Commands = append(response.Commands, compatibilityCommandResponse{
			Command:        command.Command,
			Mode:           command.Mode,
			Status:         command.Status,
			Message:        command.Message,
			AreaFlowTarget: command.AreaFlowTarget,
			Fallback:       command.Fallback,
			BlockedReason:  command.BlockedReason,
			Metadata:       command.Metadata,
		})
	}
	return response
}

func buildShimPreviewResponse(preview project.ShimPreview) shimPreviewResponse {
	response := shimPreviewResponse{
		Project:              buildProjectRecordResponse(preview.Project),
		Status:               preview.Status,
		Mode:                 preview.Mode,
		Contract:             buildCompatibilityContractResponse(preview.Contract),
		PlannedFiles:         make([]shimFilePlanResponse, 0, len(preview.PlannedFiles)),
		CommandMappings:      make([]shimCommandMappingResponse, 0, len(preview.CommandMappings)),
		DiscoveryOrder:       append([]string{}, preview.DiscoveryOrder...),
		ForbiddenPaths:       append([]string{}, preview.ForbiddenPaths...),
		ForbiddenCommands:    append([]string{}, preview.ForbiddenCommands...),
		VerificationCommands: append([]string{}, preview.VerificationCommands...),
		RollbackSteps:        append([]string{}, preview.RollbackSteps...),
		Notes:                append([]string{}, preview.Notes...),
	}
	for _, file := range preview.PlannedFiles {
		response.PlannedFiles = append(response.PlannedFiles, shimFilePlanResponse{
			Path:     file.Path,
			Action:   file.Action,
			Required: file.Required,
			Reason:   file.Reason,
			Boundary: file.Boundary,
		})
	}
	for _, mapping := range preview.CommandMappings {
		response.CommandMappings = append(response.CommandMappings, shimCommandMappingResponse{
			Command:        mapping.Command,
			Mode:           mapping.Mode,
			Status:         mapping.Status,
			AreaFlowTarget: mapping.AreaFlowTarget,
			Fallback:       mapping.Fallback,
			BlockedReason:  mapping.BlockedReason,
			ReadOnly:       mapping.ReadOnly,
			RequiresNative: mapping.RequiresNative,
			Message:        mapping.Message,
		})
	}
	return response
}

func buildShimReadinessResponse(readiness project.ShimReadiness) shimReadinessResponse {
	response := shimReadinessResponse{
		Project: buildProjectRecordResponse(readiness.Project),
		Status:  readiness.Status,
		Preview: buildShimPreviewResponse(readiness.Preview),
		Items:   make([]shimReadinessItemResponse, 0, len(readiness.Items)),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, shimReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return response
}

func buildShimAuthorizationPacketResponse(packet project.ShimAuthorizationPacket) shimAuthorizationPacketResponse {
	response := shimAuthorizationPacketResponse{
		Project:              buildProjectRecordResponse(packet.Project),
		Status:               packet.Status,
		Mode:                 packet.Mode,
		Intent:               packet.Intent,
		ReadinessStatus:      packet.ReadinessStatus,
		AllowedFiles:         make([]shimFilePlanResponse, 0, len(packet.AllowedFiles)),
		ForbiddenPaths:       append([]string{}, packet.ForbiddenPaths...),
		ForbiddenActions:     append([]string{}, packet.ForbiddenActions...),
		RequiredPreflight:    append([]string{}, packet.RequiredPreflight...),
		PostEditVerification: append([]string{}, packet.PostEditVerification...),
		RollbackScope:        append([]string{}, packet.RollbackScope...),
		SafetyFacts:          map[string]bool{},
		NextRequiredApproval: packet.NextRequiredApproval,
	}
	for _, file := range packet.AllowedFiles {
		response.AllowedFiles = append(response.AllowedFiles, shimFilePlanResponse{
			Path:     file.Path,
			Action:   file.Action,
			Required: file.Required,
			Reason:   file.Reason,
			Boundary: file.Boundary,
		})
	}
	for key, value := range packet.SafetyFacts {
		response.SafetyFacts[key] = value
	}
	return response
}

func buildShimApplyPacketPreviewResponse(preview project.ShimApplyPacketPreview) shimApplyPacketPreviewResponse {
	return shimApplyPacketPreviewResponse{
		Project:             buildProjectRecordResponse(preview.Project),
		Status:              preview.Status,
		Mode:                preview.Mode,
		Decision:            preview.Decision,
		Message:             preview.Message,
		Authorization:       buildShimAuthorizationPacketResponse(preview.Authorization),
		Gate:                buildShimApplyGateResponse(preview.Gate),
		Packet:              buildShimApplyPacketResponse(preview.Packet),
		ApplyGateCommand:    jsonStringSlice(preview.ApplyGateCommand),
		FutureApplyCommand:  jsonStringSlice(preview.FutureApplyCommand),
		RequiredHumanReview: jsonStringSlice(preview.RequiredHumanReview),
		ForbiddenActions:    jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:         preview.SafetyFacts,
		WouldCreateCommandRequestAfterApplyCommand:     preview.WouldCreateCommandRequestAfterApplyCommand,
		WouldWriteAreaMatrixShimFilesAfterApplyCommand: preview.WouldWriteAreaMatrixShimFilesAfterApplyCommand,
		WouldWriteStatusProjectionAfterApplyCommand:    preview.WouldWriteStatusProjectionAfterApplyCommand,
		CommandRequestCreated:                          preview.CommandRequestCreated,
		ProjectWriteAttempted:                          preview.ProjectWriteAttempted,
		ExecutionWriteAttempted:                        preview.ExecutionWriteAttempted,
		EngineCallAttempted:                            preview.EngineCallAttempted,
		TaskLoopRunForwarded:                           preview.TaskLoopRunForwarded,
		StatusProjectionWritten:                        preview.StatusProjectionWritten,
		AreaMatrixFilesModified:                        preview.AreaMatrixFilesModified,
		GeneratedAt:                                    formatTime(preview.GeneratedAt),
	}
}

func buildShimApplyPacketResponse(packet project.ShimApplyPacket) shimApplyPacketResponse {
	return shimApplyPacketResponse{
		CommandType:                packet.CommandType,
		ProjectKey:                 packet.ProjectKey,
		AllowedFiles:               jsonStringSlice(packet.AllowedFiles),
		ApprovalID:                 packet.ApprovalID,
		ApprovalScope:              packet.ApprovalScope,
		AuthorizationSnapshotHash:  packet.AuthorizationSnapshotHash,
		ExpectedAuthorizationMode:  packet.ExpectedAuthorizationMode,
		StatusProjectionPacketID:   packet.StatusProjectionPacketID,
		StatusProjectionGateID:     packet.StatusProjectionGateID,
		ReadOnlySmokeEvidenceID:    packet.ReadOnlySmokeEvidenceID,
		DirtyWorktreeReviewID:      packet.DirtyWorktreeReviewID,
		ProtectedPathFingerprintID: packet.ProtectedPathFingerprintID,
		RollbackPlanID:             packet.RollbackPlanID,
		IdempotencyKey:             packet.IdempotencyKey,
		AuditCorrelationID:         packet.AuditCorrelationID,
		FailureMode:                packet.FailureMode,
		ExplicitApproval:           packet.ExplicitApproval,
		ApprovalActor:              packet.ApprovalActor,
		ApprovalReason:             packet.ApprovalReason,
	}
}

func buildShimApplyGateResponse(gate project.ShimApplyGate) shimApplyGateResponse {
	response := shimApplyGateResponse{
		Project:                 buildProjectRecordResponse(gate.Project),
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Decision:                gate.Decision,
		Message:                 gate.Message,
		Items:                   make([]shimApplyGateItemResponse, 0, len(gate.Items)),
		RequiredPacketFields:    jsonStringSlice(gate.RequiredPacketFields),
		RequiredCapabilities:    jsonStringSlice(gate.RequiredCapabilities),
		AllowedFiles:            jsonStringSlice(gate.AllowedFiles),
		ForbiddenPaths:          jsonStringSlice(gate.ForbiddenPaths),
		ForbiddenActions:        jsonStringSlice(gate.ForbiddenActions),
		RequiredPreflight:       jsonStringSlice(gate.RequiredPreflight),
		PostEditVerification:    jsonStringSlice(gate.PostEditVerification),
		RollbackScope:           jsonStringSlice(gate.RollbackScope),
		RequiredProofFacts:      jsonStringSlice(gate.RequiredProofFacts),
		SafetyFacts:             gate.SafetyFacts,
		ApprovalRequired:        gate.ApprovalRequired,
		ApprovalStatus:          gate.ApprovalStatus,
		ApplyCommandEligible:    gate.ApplyCommandEligible,
		ApplyOpen:               gate.ApplyOpen,
		CommandRequestCreated:   gate.CommandRequestCreated,
		ProjectWriteAttempted:   gate.ProjectWriteAttempted,
		ExecutionWriteAttempted: gate.ExecutionWriteAttempted,
		EngineCallAttempted:     gate.EngineCallAttempted,
		TaskLoopRunForwarded:    gate.TaskLoopRunForwarded,
		StatusProjectionWritten: gate.StatusProjectionWritten,
		AreaMatrixFilesModified: gate.AreaMatrixFilesModified,
		GeneratedAt:             formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, shimApplyGateItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Expected:         item.Expected,
			Actual:           item.Actual,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			BlockedBy:        jsonStringSlice(item.BlockedBy),
		})
	}
	return response
}

func buildShimReadinessEvidenceResponse(result project.RecordShimReadinessEvidenceResult) shimReadinessEvidenceResponse {
	return shimReadinessEvidenceResponse{
		Project:                 buildProjectRecordResponse(result.Project),
		EvidenceKey:             result.EvidenceKey,
		Status:                  result.Status,
		Decision:                result.Decision,
		Message:                 result.Message,
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		IdempotencyKey:          result.IdempotencyKey,
		Created:                 result.Created,
		ProjectWriteAttempted:   result.ProjectWriteAttempted,
		ExecutionWriteAttempted: result.ExecutionWriteAttempted,
		EngineCallAttempted:     result.EngineCallAttempted,
		Metadata:                result.Metadata,
	}
}

func buildExecutionCutoverReadinessResponse(readiness project.AreaMatrixExecutionCutoverReadiness) executionCutoverReadinessResponse {
	response := executionCutoverReadinessResponse{
		Project:          buildProjectRecordResponse(readiness.Project),
		Status:           readiness.Status,
		Mode:             readiness.Mode,
		Items:            make([]executionCutoverReadinessItemResponse, 0, len(readiness.Items)),
		MigrationPath:    jsonStringSlice(readiness.MigrationPath),
		CommandEvidence:  readiness.CommandEvidence,
		Capabilities:     jsonStringSlice(readiness.Capabilities),
		ForbiddenActions: jsonStringSlice(readiness.ForbiddenActions),
		SafetyFacts:      readiness.SafetyFacts,
		NextSteps:        make([]executionCutoverNextStepResponse, 0, len(readiness.NextSteps)),
		GeneratedAt:      formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, executionCutoverReadinessItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	for _, step := range readiness.NextSteps {
		response.NextSteps = append(response.NextSteps, executionCutoverNextStepResponse{
			Key:         step.Key,
			Owner:       step.Owner,
			Action:      step.Action,
			RiskLevel:   step.RiskLevel,
			BlockedBy:   jsonStringSlice(step.BlockedBy),
			NextCommand: step.NextCommand,
			Metadata:    step.Metadata,
		})
	}
	return response
}

func buildExecutionForwardingV1ReadinessResponse(readiness project.ExecutionForwardingV1Readiness) executionForwardingV1ReadinessResponse {
	response := executionForwardingV1ReadinessResponse{
		Project:          buildProjectRecordResponse(readiness.Project),
		Status:           readiness.Status,
		Mode:             readiness.Mode,
		Items:            make([]executionCutoverReadinessItemResponse, 0, len(readiness.Items)),
		AllowedTaskTypes: jsonStringSlice(readiness.AllowedTaskTypes),
		CommandEvidence:  readiness.CommandEvidence,
		Capabilities:     jsonStringSlice(readiness.Capabilities),
		ForbiddenActions: jsonStringSlice(readiness.ForbiddenActions),
		SafetyFacts:      readiness.SafetyFacts,
		NextSteps:        make([]executionCutoverNextStepResponse, 0, len(readiness.NextSteps)),
		GeneratedAt:      formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, executionCutoverReadinessItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	for _, step := range readiness.NextSteps {
		response.NextSteps = append(response.NextSteps, executionCutoverNextStepResponse{
			Key:         step.Key,
			Owner:       step.Owner,
			Action:      step.Action,
			RiskLevel:   step.RiskLevel,
			BlockedBy:   jsonStringSlice(step.BlockedBy),
			NextCommand: step.NextCommand,
			Metadata:    step.Metadata,
		})
	}
	return response
}

func buildExecutionForwardingV1ApplyPreviewResponse(preview project.ExecutionForwardingV1ApplyPreview) executionForwardingV1ApplyPreviewResponse {
	response := executionForwardingV1ApplyPreviewResponse{
		Project:              buildProjectRecordResponse(preview.Project),
		Status:               preview.Status,
		Mode:                 preview.Mode,
		Readiness:            buildExecutionForwardingV1ReadinessResponse(preview.Readiness),
		Items:                make([]executionForwardingV1ApplyPreviewItemResponse, 0, len(preview.Items)),
		AllowedTaskTypes:     jsonStringSlice(preview.AllowedTaskTypes),
		ForwardingTargets:    make([]executionForwardingV1ForwardingTargetResponse, 0, len(preview.ForwardingTargets)),
		BlockedTargets:       make([]executionForwardingV1BlockedTargetResponse, 0, len(preview.BlockedTargets)),
		RequiredCapabilities: jsonStringSlice(preview.RequiredCapabilities),
		ApplyPacketFields:    jsonStringSlice(preview.ApplyPacketFields),
		FailClosedFields:     jsonStringSlice(preview.FailClosedFields),
		RequiredProofFacts:   jsonStringSlice(preview.RequiredProofFacts),
		RequiredEvidence:     jsonStringSlice(preview.RequiredEvidence),
		ForbiddenActions:     jsonStringSlice(preview.ForbiddenActions),
		ApprovalRequired:     preview.ApprovalRequired,
		ApprovalStatus:       preview.ApprovalStatus,
		ApplyOpen:            preview.ApplyOpen,
		RollbackTarget:       preview.RollbackTarget,
		SafetyFacts:          preview.SafetyFacts,
		GeneratedAt:          formatTime(preview.GeneratedAt),
	}
	for _, target := range preview.ForwardingTargets {
		response.ForwardingTargets = append(response.ForwardingTargets, executionForwardingV1ForwardingTargetResponse{
			TaskType:              target.TaskType,
			TargetCommandType:     target.TargetCommandType,
			TargetStatus:          target.TargetStatus,
			RequiredCapabilities:  jsonStringSlice(target.RequiredCapabilities),
			RequiredPacketFields:  jsonStringSlice(target.RequiredPacketFields),
			CreatesCommandRequest: target.CreatesCommandRequest,
			CreatesRun:            target.CreatesRun,
			CreatesRunTask:        target.CreatesRunTask,
			CreatesRunAttempt:     target.CreatesRunAttempt,
			CreatesArtifact:       target.CreatesArtifact,
			CreatesAuditEvent:     target.CreatesAuditEvent,
			ProjectWriteAllowed:   target.ProjectWriteAllowed,
			ExecutionWriteAllowed: target.ExecutionWriteAllowed,
			LegacyFallbackAllowed: target.LegacyFallbackAllowed,
			FailureMode:           target.FailureMode,
		})
	}
	for _, target := range preview.BlockedTargets {
		response.BlockedTargets = append(response.BlockedTargets, executionForwardingV1BlockedTargetResponse{
			TaskType:        target.TaskType,
			ForbiddenAction: target.ForbiddenAction,
			Reason:          target.Reason,
			FailureMode:     target.FailureMode,
			SafetyFacts:     target.SafetyFacts,
		})
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, executionForwardingV1ApplyPreviewItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			ApprovalStatus:   item.ApprovalStatus,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         jsonObject(item.Metadata),
		})
	}
	return response
}

func buildExecutionForwardingV1ApplyPacketPreviewResponse(preview project.ExecutionForwardingV1ApplyPacketPreview) executionForwardingV1ApplyPacketPreviewResponse {
	return executionForwardingV1ApplyPacketPreviewResponse{
		Project:             buildProjectRecordResponse(preview.Project),
		Status:              preview.Status,
		Mode:                preview.Mode,
		Decision:            preview.Decision,
		Message:             preview.Message,
		ApplyPreview:        buildExecutionForwardingV1ApplyPreviewResponse(preview.ApplyPreview),
		Gate:                buildExecutionForwardingV1ApplyGateResponse(preview.Gate),
		Packet:              buildExecutionForwardingV1ApplyPacketResponse(preview.Packet),
		ApplyGateCommand:    jsonStringSlice(preview.ApplyGateCommand),
		FutureApplyCommand:  jsonStringSlice(preview.FutureApplyCommand),
		RequiredHumanReview: jsonStringSlice(preview.RequiredHumanReview),
		ForbiddenActions:    jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:         preview.SafetyFacts,
		WouldCreateCommandRequestAfterApplyCommand: preview.WouldCreateCommandRequestAfterApplyCommand,
		WouldCreateRunAfterApplyCommand:            preview.WouldCreateRunAfterApplyCommand,
		WouldCreateRunTaskAfterApplyCommand:        preview.WouldCreateRunTaskAfterApplyCommand,
		WouldCreateAuditEventAfterApplyCommand:     preview.WouldCreateAuditEventAfterApplyCommand,
		CommandRequestCreated:                      preview.CommandRequestCreated,
		AreaFlowRunCreated:                         preview.AreaFlowRunCreated,
		TaskLoopRunForwarded:                       preview.TaskLoopRunForwarded,
		ProjectWriteAttempted:                      preview.ProjectWriteAttempted,
		ExecutionWriteAttempted:                    preview.ExecutionWriteAttempted,
		EngineCallAttempted:                        preview.EngineCallAttempted,
		GeneratedAt:                                formatTime(preview.GeneratedAt),
	}
}

func buildExecutionForwardingV1ApplyPacketResponse(packet project.ExecutionForwardingV1ApplyPacket) executionForwardingV1ApplyPacketResponse {
	return executionForwardingV1ApplyPacketResponse{
		CommandType:                packet.CommandType,
		ProjectKey:                 packet.ProjectKey,
		AllowedTaskTypes:           jsonStringSlice(packet.AllowedTaskTypes),
		TargetCommandTypes:         jsonStringSlice(packet.TargetCommandTypes),
		ApprovalID:                 packet.ApprovalID,
		ApprovalScope:              packet.ApprovalScope,
		ReadinessSnapshotHash:      packet.ReadinessSnapshotHash,
		ExpectedShimLifecycleState: packet.ExpectedShimLifecycleState,
		LegacyNonWriteProofID:      packet.LegacyNonWriteProofID,
		RollbackPlanID:             packet.RollbackPlanID,
		ProtectedPathFingerprintID: packet.ProtectedPathFingerprintID,
		IdempotencyKey:             packet.IdempotencyKey,
		AuditCorrelationID:         packet.AuditCorrelationID,
		FailureMode:                packet.FailureMode,
		ExplicitApproval:           packet.ExplicitApproval,
		ApprovalActor:              packet.ApprovalActor,
		ApprovalReason:             packet.ApprovalReason,
	}
}

func buildExecutionForwardingV1ApplyGateResponse(gate project.ExecutionForwardingV1ApplyGate) executionForwardingV1ApplyGateResponse {
	response := executionForwardingV1ApplyGateResponse{
		Project:                 buildProjectRecordResponse(gate.Project),
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Decision:                gate.Decision,
		Message:                 gate.Message,
		Items:                   make([]executionForwardingV1ApplyGateItemResponse, 0, len(gate.Items)),
		RequiredPacketFields:    jsonStringSlice(gate.RequiredPacketFields),
		RequiredCapabilities:    jsonStringSlice(gate.RequiredCapabilities),
		AllowedTaskTypes:        jsonStringSlice(gate.AllowedTaskTypes),
		TargetCommandTypes:      jsonStringSlice(gate.TargetCommandTypes),
		BlockedTaskTypes:        jsonStringSlice(gate.BlockedTaskTypes),
		ForbiddenActions:        jsonStringSlice(gate.ForbiddenActions),
		FailClosedFields:        jsonStringSlice(gate.FailClosedFields),
		RequiredProofFacts:      jsonStringSlice(gate.RequiredProofFacts),
		SafetyFacts:             gate.SafetyFacts,
		ApprovalRequired:        gate.ApprovalRequired,
		ApprovalStatus:          gate.ApprovalStatus,
		ApplyCommandEligible:    gate.ApplyCommandEligible,
		ApplyOpen:               gate.ApplyOpen,
		CommandRequestCreated:   gate.CommandRequestCreated,
		AreaFlowRunCreated:      gate.AreaFlowRunCreated,
		TaskLoopRunForwarded:    gate.TaskLoopRunForwarded,
		ProjectWriteAttempted:   gate.ProjectWriteAttempted,
		ExecutionWriteAttempted: gate.ExecutionWriteAttempted,
		EngineCallAttempted:     gate.EngineCallAttempted,
		GeneratedAt:             formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		response.Items = append(response.Items, executionForwardingV1ApplyGateItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Expected:         item.Expected,
			Actual:           item.Actual,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			BlockedBy:        jsonStringSlice(item.BlockedBy),
		})
	}
	return response
}

func buildShimApplyCommandResponse(result project.ApplyShimCommandResult) shimApplyCommandResponse {
	return shimApplyCommandResponse{
		Project:                 buildProjectRecordResponse(result.Project),
		Status:                  result.Status,
		Mode:                    result.Mode,
		Decision:                result.Decision,
		Message:                 result.Message,
		Gate:                    buildShimApplyGateResponse(result.Gate),
		Blockers:                jsonStringSlice(result.Blockers),
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		IdempotencyKey:          result.IdempotencyKey,
		Created:                 result.Created,
		RequiredPreflight:       jsonStringSlice(result.RequiredPreflight),
		ForbiddenActions:        jsonStringSlice(result.ForbiddenActions),
		SafetyFacts:             result.SafetyFacts,
		ApplyOpen:               result.ApplyOpen,
		AreaFlowCommandCreated:  result.AreaFlowCommandCreated,
		CommandRequestCreated:   result.CommandRequestCreated,
		ProjectWriteAttempted:   result.ProjectWriteAttempted,
		ExecutionWriteAttempted: result.ExecutionWriteAttempted,
		EngineCallAttempted:     result.EngineCallAttempted,
		TaskLoopRunForwarded:    result.TaskLoopRunForwarded,
		StatusProjectionWritten: result.StatusProjectionWritten,
		AreaMatrixFilesModified: result.AreaMatrixFilesModified,
		GeneratedAt:             formatTime(result.GeneratedAt),
	}
}

func buildExecutionForwardingV1ApplyResponse(result project.ApplyExecutionForwardingV1Result) executionForwardingV1ApplyResponse {
	return executionForwardingV1ApplyResponse{
		Project:                         buildProjectRecordResponse(result.Project),
		Status:                          result.Status,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Blockers:                        jsonStringSlice(result.Blockers),
		Gate:                            buildExecutionForwardingV1ApplyGateResponse(result.Gate),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		SafetyFacts:                     result.SafetyFacts,
		CommandRequestCreated:           result.CommandRequestCreated,
		AreaFlowCommandCreated:          result.AreaFlowCommandCreated,
		AreaFlowRunCreated:              result.AreaFlowRunCreated,
		AreaFlowRunTaskCreated:          result.AreaFlowRunTaskCreated,
		AreaFlowRunAttemptCreated:       result.AreaFlowRunAttemptCreated,
		AreaFlowArtifactCreated:         result.AreaFlowArtifactCreated,
		AreaFlowAuditEventCreated:       result.AreaFlowAuditEventCreated,
		TaskLoopRunForwarded:            result.TaskLoopRunForwarded,
		LegacyTaskLoopStarted:           result.LegacyTaskLoopStarted,
		LegacyProgressWritten:           result.LegacyProgressWritten,
		LegacyLogsWritten:               result.LegacyLogsWritten,
		LegacyCheckpointWritten:         result.LegacyCheckpointWritten,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		EngineCallAttempted:             result.EngineCallAttempted,
		CommandsRun:                     result.CommandsRun,
		SecretsResolved:                 result.SecretsResolved,
		NetworkUsed:                     result.NetworkUsed,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		GeneratedAt:                     formatTime(result.GeneratedAt),
	}
}

func buildExecutionForwardingV1CommandPreviewResponse(preview project.ExecutionForwardingV1CommandPreview) executionForwardingV1CommandPreviewResponse {
	return executionForwardingV1CommandPreviewResponse{
		Project:                                buildProjectRecordResponse(preview.Project),
		Status:                                 preview.Status,
		Mode:                                   preview.Mode,
		Decision:                               preview.Decision,
		Message:                                preview.Message,
		TaskType:                               preview.TaskType,
		TargetCommandType:                      preview.TargetCommandType,
		TargetStatus:                           preview.TargetStatus,
		FailureMode:                            preview.FailureMode,
		AllowedTaskType:                        preview.AllowedTaskType,
		BlockedTaskType:                        preview.BlockedTaskType,
		ApplyOpen:                              preview.ApplyOpen,
		WouldCreateCommandRequestAfterApproval: preview.WouldCreateCommandRequestAfterApproval,
		WouldCreateRunAfterApproval:            preview.WouldCreateRunAfterApproval,
		WouldCreateRunTaskAfterApproval:        preview.WouldCreateRunTaskAfterApproval,
		WouldCreateRunAttemptAfterApproval:     preview.WouldCreateRunAttemptAfterApproval,
		WouldCreateArtifactAfterApproval:       preview.WouldCreateArtifactAfterApproval,
		WouldCreateAuditEventAfterApproval:     preview.WouldCreateAuditEventAfterApproval,
		ProjectWriteAllowed:                    preview.ProjectWriteAllowed,
		ExecutionWriteAllowed:                  preview.ExecutionWriteAllowed,
		LegacyFallbackAllowed:                  preview.LegacyFallbackAllowed,
		RequiredPacketFields:                   jsonStringSlice(preview.RequiredPacketFields),
		RequiredCapabilities:                   jsonStringSlice(preview.RequiredCapabilities),
		FailClosedFields:                       jsonStringSlice(preview.FailClosedFields),
		BlockedBy:                              jsonStringSlice(preview.BlockedBy),
		AllowedTaskTypes:                       jsonStringSlice(preview.AllowedTaskTypes),
		ForbiddenActions:                       jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:                            preview.SafetyFacts,
		GeneratedAt:                            formatTime(preview.GeneratedAt),
	}
}

func buildExecutionForwardingV1RollbackPreviewResponse(preview project.ExecutionForwardingV1RollbackPreview) executionForwardingV1RollbackPreviewResponse {
	response := executionForwardingV1RollbackPreviewResponse{
		Project:            buildProjectRecordResponse(preview.Project),
		Status:             preview.Status,
		Mode:               preview.Mode,
		ApplyPreview:       buildExecutionForwardingV1ApplyPreviewResponse(preview.ApplyPreview),
		Items:              make([]executionForwardingV1RollbackPreviewItemResponse, 0, len(preview.Items)),
		RollbackTarget:     preview.RollbackTarget,
		FailClosedSteps:    jsonStringSlice(preview.FailClosedSteps),
		ReopenConditions:   jsonStringSlice(preview.ReopenConditions),
		RequiredProofFacts: jsonStringSlice(preview.RequiredProofFacts),
		RequiredEvidence:   jsonStringSlice(preview.RequiredEvidence),
		ForbiddenActions:   jsonStringSlice(preview.ForbiddenActions),
		RollbackApplyOpen:  preview.RollbackApplyOpen,
		SafetyFacts:        preview.SafetyFacts,
		GeneratedAt:        formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		response.Items = append(response.Items, executionForwardingV1RollbackPreviewItemResponse{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			NextCommand:      item.NextCommand,
			Metadata:         jsonObject(item.Metadata),
		})
	}
	return response
}

func buildProjectCutoverReadinessResponse(readiness project.ProjectCutoverReadiness) projectCutoverReadinessResponse {
	response := projectCutoverReadinessResponse{
		Project:         buildProjectRecordResponse(readiness.Project),
		WorkflowVersion: buildWorkflowVersionResponse(readiness.Version),
		Status:          readiness.Status,
		PhaseGate: projectPhaseGateResponse{
			Name:             readiness.PhaseGate.Name,
			Status:           readiness.PhaseGate.Status,
			AcceptedWarnings: readiness.PhaseGate.AcceptedWarnings,
			Blockers:         readiness.PhaseGate.Blockers,
		},
		Items:         make([]projectReadinessItemResponse, 0, len(readiness.Items)),
		Verification:  buildProjectVerificationBundleResponse(readiness.Verification),
		Compatibility: buildCompatibilityContractResponse(readiness.Compatibility),
		Gates:         make([]gateResultResponse, 0, len(readiness.Gates)),
	}
	for _, item := range readiness.Items {
		response.Items = append(response.Items, projectReadinessItemResponse{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	for _, gate := range readiness.Gates {
		response.Gates = append(response.Gates, buildGateResultResponse(gate))
	}
	return response
}

func buildProjectCutoverApplyResponse(result project.ApplyCutoverResult) projectCutoverApplyResponse {
	return projectCutoverApplyResponse{
		Project:                  buildProjectRecordResponse(result.Project),
		WorkflowVersion:          buildWorkflowVersionResponse(result.Version),
		Status:                   result.Status,
		Decision:                 result.Decision,
		Message:                  result.Message,
		Blockers:                 result.Blockers,
		Warnings:                 result.Warnings,
		EventID:                  result.EventID,
		AuditEventID:             result.AuditEventID,
		IdempotencyKey:           result.IdempotencyKey,
		Created:                  result.Created,
		ProjectWriteAttempted:    result.ProjectWriteAttempted,
		ExecutionWriteAttempted:  result.ExecutionWriteAttempted,
		AreaMatrixWriteAttempted: result.AreaMatrixWriteAttempted,
		CutoverReadinessGateID:   result.CutoverReadinessGateID,
	}
}

func buildProjectImportDiffResponse(diff project.ProjectImportDiff) projectImportDiffResponse {
	response := projectImportDiffResponse{
		Project:       buildProjectRecordResponse(diff.Project),
		Status:        diff.Status,
		HasPrevious:   diff.HasPrevious,
		SourceChanged: diff.SourceChanged,
		Latest: projectDiffSnapshotResponse{
			SourceHash: diff.Latest.SourceHash,
			CreatedAt:  formatTime(diff.Latest.CreatedAt),
		},
		Changes: make([]projectDiffChangeResponse, 0, len(diff.Changes)),
	}
	if diff.HasPrevious {
		response.Previous = projectDiffSnapshotResponse{
			SourceHash: diff.Previous.SourceHash,
			CreatedAt:  formatTime(diff.Previous.CreatedAt),
		}
	}
	for _, change := range diff.Changes {
		response.Changes = append(response.Changes, projectDiffChangeResponse{
			Key:      change.Key,
			Status:   change.Status,
			Previous: change.Previous,
			Latest:   change.Latest,
		})
	}
	return response
}

func buildEventResponse(event project.EventRecord) eventResponse {
	return eventResponse{
		ID:                event.ID,
		ProjectID:         event.ProjectID,
		RunID:             event.RunID,
		WorkflowVersionID: event.WorkflowVersionID,
		Type:              event.Type,
		Severity:          event.Severity,
		Message:           event.Message,
		Metadata:          event.Metadata,
		CreatedAt:         event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func writeSSEEvent(w http.ResponseWriter, eventType string, value any, eventID int64) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if eventID > 0 {
		if _, err := fmt.Fprintf(w, "id: %d\n", eventID); err != nil {
			return err
		}
	}
	if eventType != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", eventType); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

func queryLimit(r *http.Request, fallback int) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	return limit, nil
}

func queryOptionalInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}

func queryOptionalInt64Ptr(r *http.Request, key string) (*int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return &value, nil
}

func queryOptionalBool(r *http.Request, key string) (bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return value, nil
}

func queryOptionalBoolPtr(r *http.Request, key string) (*bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be true or false", key)
	}
	return &value, nil
}

func parsePositiveInt64(value string, label string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", label)
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

// ListenAddr returns a free local listener address for tests.
func ListenAddr() (string, func(), error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("listen test addr: %w", err)
	}
	addr := listener.Addr().String()
	cleanup := func() {
		_ = listener.Close()
	}
	return addr, cleanup, nil
}
