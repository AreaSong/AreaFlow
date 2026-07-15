package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/auth"
	"github.com/areasong/areaflow/internal/config"
	"github.com/areasong/areaflow/internal/doctor"
	"github.com/areasong/areaflow/internal/importer"
	"github.com/areasong/areaflow/internal/project"
)

type fakeTokenAuthenticator struct {
	principal auth.Principal
	err       error
	rawToken  string
}

type fakeReadinessCheck struct{ err error }

func (f fakeReadinessCheck) Ping(context.Context) error { return f.err }

func (f *fakeTokenAuthenticator) Authenticate(_ context.Context, rawToken string) (auth.Principal, error) {
	f.rawToken = rawToken
	if f.err != nil {
		return auth.Principal{}, f.err
	}
	return f.principal, nil
}

type fakeProjectStore struct {
	record                        project.Record
	records                       []project.Record
	recordsByKey                  map[string]project.Record
	getByKeyHook                  func(string)
	events                        []project.EventRecord
	eventsHook                    func(projectID int64, limit int) []project.EventRecord
	auditEvents                   []project.AuditEventRecord
	auditEventsHook               func(projectID int64, limit int) []project.AuditEventRecord
	auditCover                    project.AuditCoverage
	streamHook                    func(project.EventStreamFilter)
	summary                       project.ProjectSummary
	summaryHook                   func(project.Record) project.ProjectSummary
	doctorRecord                  project.RecordDoctorReportResult
	doctorRecordHook              func(project.RecordDoctorReportOptions)
	readiness                     project.ProjectReadiness
	generatedWriteReadiness       project.GeneratedWriteReadiness
	generatedApplyBetaGate        project.GeneratedWriteApplyBetaGate
	permission                    project.PermissionPolicyDoctor
	conformance                   project.ConformanceReport
	diff                          project.ProjectImportDiff
	bundle                        project.ProjectVerificationBundle
	cutover                       project.ProjectCutoverReadiness
	executionCutover              project.AreaMatrixExecutionCutoverReadiness
	executionForwarding           project.ExecutionForwardingV1Readiness
	executionForwardingApply      project.ExecutionForwardingV1ApplyPreview
	executionForwardingPacket     project.ExecutionForwardingV1ApplyPacketPreview
	executionForwardingPacketHook func(project.ExecutionForwardingV1ApplyPacketPreviewOptions)
	executionForwardingGate       project.ExecutionForwardingV1ApplyGate
	executionForwardingGateHook   func(project.ExecutionForwardingV1ApplyGateOptions)
	executionForwardingApplyCmd   project.ApplyExecutionForwardingV1Result
	executionForwardingApplyHook  func(project.ApplyExecutionForwardingV1Options)
	executionForwardingCmd        project.ExecutionForwardingV1CommandPreview
	executionForwardingRoll       project.ExecutionForwardingV1RollbackPreview
	compat                        project.CompatibilityContract
	shimPreview                   project.ShimPreview
	shimReadiness                 project.ShimReadiness
	shimAuthorization             project.ShimAuthorizationPacket
	shimApplyPacket               project.ShimApplyPacketPreview
	shimApplyPacketHook           func(project.ShimApplyPacketPreviewOptions)
	shimApplyGate                 project.ShimApplyGate
	shimApplyGateHook             func(project.ShimApplyGateOptions)
	shimApplyCommand              project.ApplyShimCommandResult
	shimApplyCommandHook          func(project.ApplyShimCommandOptions)
	shimEvidence                  project.RecordShimReadinessEvidenceResult
	shimEvidenceHook              func(project.RecordShimReadinessEvidenceOptions)
	cutoverApply                  project.ApplyCutoverResult
	cutoverApplyHook              func(project.ApplyCutoverOptions)
	statusProjections             []project.StatusProjectionRecord
	statusAuthorization           project.StatusProjectionAuthorizationPreview
	statusAuthorizationHook       func(project.StatusProjectionAuthorizationPreviewOptions)
	statusApplyPacket             project.StatusProjectionApplyPacketPreview
	statusApplyPacketHook         func(project.StatusProjectionApplyPacketPreviewOptions)
	statusApplyGate               project.StatusProjectionApplyGate
	statusApplyGateHook           func(project.StatusProjectionApplyGateOptions)
	statusApply                   project.ApplyStatusProjectionResult
	statusApplyHook               func(project.ApplyStatusProjectionOptions)
	versions                      []project.WorkflowVersion
	version                       project.WorkflowVersion
	versionsHook                  func(project.Record) []project.WorkflowVersion
	getWorkflowVersionHook        func(project.Record, string) (project.WorkflowVersion, error)
	items                         []project.WorkflowItem
	create                        project.CreateWorkflowVersionResult
	ensure                        project.EnsureStageSkeletonResult
	gate                          project.GateResult
	gates                         []project.GateResult
	preview                       project.WorkflowTransitionPreview
	previewHook                   func(project.PreviewTransitionOptions)
	previews                      []project.WorkflowTransitionPreview
	approval                      project.ApprovalRecord
	approvalHook                  func(project.CreateApprovalOptions)
	approvals                     []project.ApprovalRecord
	runner                        project.RunnerPreviewResult
	workerPool                    project.WorkerPoolSummary
	schedule                      project.WorkerPoolSchedulePreview
	codexPreview                  project.CodexCLIAdapterPreview
	codexPreviewHook              func(project.CodexCLIAdapterPreviewOptions)
	webGate                       project.WebWriteActionGate
	desktopGate                   project.DesktopServiceControlGate
	notificationGate              project.DesktopNotificationGate
	trayMenuGate                  project.DesktopTrayMenuGate
	service                       project.LocalServiceStatus
	security                      project.SecurityBoundaryReadiness
	completion                    project.CompletionAudit
	completionSnapshot            project.CompletionAuditSnapshotReadiness
	supportBundle                 project.SupportBundlePreview
	migrationLedger               project.MigrationLedgerReadiness
	operations                    project.OperationsReadiness
	backup                        project.BackupManifest
	backupHook                    func(project.BackupManifestOptions)
	restore                       project.RestorePlan
	restoreHook                   func(project.RestorePlanOptions)
	release                       project.ReleaseReadiness
	releaseHook                   func(project.ReleaseReadinessOptions)
	remediation                   project.ReleaseRemediationPlan
	remediationHook               func(project.ReleaseRemediationOptions)
	acceptance                    project.ReleaseAcceptancePreview
	acceptanceHook                func(project.ReleaseAcceptancePreviewOptions)
	gateRelease                   project.ReleaseAcceptanceGate
	gateReleaseHook               func(project.ReleaseAcceptanceGateOptions)
	exception                     project.ReleaseExceptionDoctor
	exceptionHook                 func(project.ReleaseExceptionDoctorOptions)
	recordPrev                    project.ReleaseExceptionRecordPreview
	recordPrevHook                func(project.ReleaseExceptionRecordPreviewOptions)
	schemaPrev                    project.ReleaseExceptionSchemaPreview
	schemaPrevHook                func(project.ReleaseExceptionSchemaPreviewOptions)
	migration                     project.ReleaseExceptionMigrationApprovalGate
	migrationHook                 func(project.ReleaseExceptionMigrationApprovalGateOptions)
	applyPrev                     project.ReleaseExceptionApplyPreview
	applyPrevHook                 func(project.ReleaseExceptionApplyPreviewOptions)
	finalGate                     project.ReleaseFinalGate
	finalGateHook                 func(project.ReleaseFinalGateOptions)
	evidence                      project.ReleaseEvidenceBundle
	evidenceHook                  func(project.ReleaseEvidenceBundleOptions)
	packagePrev                   project.ReleasePackagePreview
	packagePrevHook               func(project.ReleasePackagePreviewOptions)
	distPrev                      project.ReleaseDistributionPreview
	distPrevHook                  func(project.ReleaseDistributionPreviewOptions)
	publishGate                   project.ReleasePublishGate
	publishGateHook               func(project.ReleasePublishGateOptions)
	publishAppr                   project.ReleasePublishApprovalPreview
	publishApprHook               func(project.ReleasePublishApprovalPreviewOptions)
	rollout                       project.ReleaseRolloutPlanPreview
	rolloutHook                   func(project.ReleaseRolloutPlanPreviewOptions)
	integrity                     project.ArtifactIntegrityReport
	archivePreview                project.ArtifactArchivePreviewResult
	archivePreviewHook            func(project.ArtifactArchivePreviewOptions)
	worker                        project.WorkerRecord
	workers                       []project.WorkerRecord
	workersHook                   func(project.Record, int) []project.WorkerRecord
	workerRegisterHook            func(project.RegisterWorkerOptions)
	workerHeartbeatHook           func(string, project.WorkerHeartbeatOptions)
	fixtureQueue                  project.FixtureExecutionQueueResult
	fixtureQueueHook              func(project.FixtureExecutionQueueOptions)
	fixtureExecute                project.FixtureExecutionResult
	fixtureExecuteHook            func(project.FixtureExecutionOptions)
	readOnlyQueue                 project.ReadOnlyVerifyQueueResult
	readOnlyQueueHook             func(project.ReadOnlyVerifyQueueOptions)
	readOnlyVerify                project.ReadOnlyVerifyResult
	readOnlyVerifyHook            func(project.ReadOnlyVerifyOptions)
	artifactWriteQueue            project.ApprovedArtifactWriteQueueResult
	artifactWriteQueueHook        func(project.ApprovedArtifactWriteQueueOptions)
	artifactWrite                 project.ApprovedArtifactWriteResult
	artifactWriteHook             func(project.ApprovedArtifactWriteOptions)
	fixtureProjectQueue           project.FixtureProjectWriteQueueResult
	fixtureProjectQueueHook       func(project.FixtureProjectWriteQueueOptions)
	fixtureProjectWrite           project.FixtureProjectWriteResult
	fixtureProjectWriteHook       func(project.FixtureProjectWriteOptions)
	managedGeneratedQueue         project.ManagedGeneratedWriteQueueResult
	managedGeneratedQueueHook     func(project.ManagedGeneratedWriteQueueOptions)
	managedGeneratedWrite         project.ManagedGeneratedWriteResult
	managedGeneratedWriteHook     func(project.ManagedGeneratedWriteOptions)
	lease                         project.LeaseRecord
	leases                        []project.LeaseRecord
	leaseAcquireHook              func(project.AcquireLeaseOptions)
	leaseReleaseHook              func(project.ReleaseLeaseOptions)
	leaseRecoverHook              func(project.RecoverLeasesOptions)
	runOnce                       project.WorkerRunOnceResult
	runOnceHook                   func(project.WorkerRunOnceOptions)
	runDetail                     project.RunDetail
	executionGate                 project.ExecutionApprovalGate
	executionPlan                 project.ExecutionPlanPreview
	projectWriteDesignGate        project.ProjectWriteDesignGate
	managedGeneratedGate          project.ManagedGeneratedWriteGate
	runControl                    project.RunControlResult
	runControlHook                func(project.RunControlOptions)
	runs                          []project.RunRecord
	runsHook                      func(project.Record, project.WorkflowVersion, int) []project.RunRecord
	runEvents                     []project.EventRecord
	artifact                      project.ArtifactRecord
	artifactContent               project.ArtifactContent
	artifacts                     []project.ArtifactRecord
	artifactsHook                 func(project.Record, int) []project.ArtifactRecord
	versionArtifactsHook          func(project.Record, project.WorkflowVersion, int) []project.ArtifactRecord
	residuals                     []project.ResidualRecord
	leaseErr                      error
	workerErr                     error
	runOnceErr                    error
	runErr                        error
	artifactErr                   error
	artifactContentErr            error
	pingErr                       error
	err                           error
}

func (s fakeProjectStore) Ping(context.Context) error {
	return s.pingErr
}

func (s fakeProjectStore) List(context.Context) ([]project.Record, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records, nil
}

func (s fakeProjectStore) GetByKey(_ context.Context, key string) (project.Record, error) {
	if s.getByKeyHook != nil {
		s.getByKeyHook(key)
	}
	if s.err != nil {
		return project.Record{}, s.err
	}
	if s.recordsByKey != nil {
		record, ok := s.recordsByKey[key]
		if !ok {
			return project.Record{}, errors.New("not found")
		}
		return record, nil
	}
	return s.record, nil
}

func (s fakeProjectStore) RecordDoctorReport(_ context.Context, _ int64, _ map[string]any, options project.RecordDoctorReportOptions) (project.RecordDoctorReportResult, error) {
	if s.doctorRecordHook != nil {
		s.doctorRecordHook(options)
	}
	if s.err != nil {
		return project.RecordDoctorReportResult{}, s.err
	}
	return s.doctorRecord, nil
}

func (s fakeProjectStore) ListEvents(_ context.Context, projectID int64, limit int) ([]project.EventRecord, error) {
	if s.eventsHook != nil {
		return s.eventsHook(projectID, limit), nil
	}
	return s.events, nil
}

func (s fakeProjectStore) ListAuditEvents(_ context.Context, projectID int64, limit int) ([]project.AuditEventRecord, error) {
	if s.auditEventsHook != nil {
		return s.auditEventsHook(projectID, limit), nil
	}
	return s.auditEvents, nil
}

func (s fakeProjectStore) AuditCoverage(context.Context, project.AuditCoverageOptions) (project.AuditCoverage, error) {
	if s.err != nil {
		return project.AuditCoverage{}, s.err
	}
	return s.auditCover, nil
}

func (s fakeProjectStore) ListEventStream(_ context.Context, filter project.EventStreamFilter) ([]project.EventRecord, error) {
	if s.streamHook != nil {
		s.streamHook(filter)
	}
	return s.events, nil
}

func (s fakeProjectStore) ListWorkflowVersions(_ context.Context, record project.Record) ([]project.WorkflowVersion, error) {
	if s.versionsHook != nil {
		return s.versionsHook(record), nil
	}
	return s.versions, nil
}

func (s fakeProjectStore) GetWorkflowVersion(_ context.Context, record project.Record, label string) (project.WorkflowVersion, error) {
	if s.getWorkflowVersionHook != nil {
		return s.getWorkflowVersionHook(record, label)
	}
	if s.err != nil {
		return project.WorkflowVersion{}, s.err
	}
	return s.version, nil
}

func (s fakeProjectStore) CreateWorkflowVersion(context.Context, project.Record, project.CreateWorkflowVersionOptions) (project.CreateWorkflowVersionResult, error) {
	if s.err != nil {
		return project.CreateWorkflowVersionResult{}, s.err
	}
	return s.create, nil
}

func (s fakeProjectStore) ListWorkflowItems(context.Context, project.Record, project.WorkflowVersion) ([]project.WorkflowItem, error) {
	return s.items, nil
}

func (s fakeProjectStore) ListWorkflowItemLinks(context.Context, project.Record, project.WorkflowVersion, int) ([]project.WorkflowItemLink, error) {
	return s.ensure.Links, nil
}

func (s fakeProjectStore) EnsureStageSkeleton(context.Context, project.Record, string, project.EnsureStageSkeletonOptions) (project.EnsureStageSkeletonResult, error) {
	if s.err != nil {
		return project.EnsureStageSkeletonResult{}, s.err
	}
	return s.ensure, nil
}

func (s fakeProjectStore) RunWorkflowGate(context.Context, project.Record, string, project.RunGateOptions) (project.GateResult, error) {
	if s.err != nil {
		return project.GateResult{}, s.err
	}
	return s.gate, nil
}

func (s fakeProjectStore) ListGateResults(context.Context, project.Record, project.WorkflowVersion, int) ([]project.GateResult, error) {
	return s.gates, nil
}

func (s fakeProjectStore) PreviewWorkflowTransition(_ context.Context, _ project.Record, _ string, options project.PreviewTransitionOptions) (project.WorkflowTransitionPreview, error) {
	if s.previewHook != nil {
		s.previewHook(options)
	}
	if s.err != nil {
		return project.WorkflowTransitionPreview{}, s.err
	}
	return s.preview, nil
}

func (s fakeProjectStore) ListWorkflowTransitionPreviews(context.Context, project.Record, project.WorkflowVersion, int) ([]project.WorkflowTransitionPreview, error) {
	return s.previews, nil
}

func (s fakeProjectStore) CreateApprovalRecord(_ context.Context, _ project.Record, _ string, options project.CreateApprovalOptions) (project.ApprovalRecord, error) {
	if s.approvalHook != nil {
		s.approvalHook(options)
	}
	if s.err != nil {
		return project.ApprovalRecord{}, s.err
	}
	return s.approval, nil
}

func (s fakeProjectStore) ListApprovalRecords(context.Context, project.Record, project.WorkflowVersion, int) ([]project.ApprovalRecord, error) {
	return s.approvals, nil
}

func (s fakeProjectStore) ProjectSummary(_ context.Context, record project.Record) (project.ProjectSummary, error) {
	if s.summaryHook != nil {
		return s.summaryHook(record), nil
	}
	return s.summary, nil
}

func (s fakeProjectStore) ProjectReadiness(context.Context, project.Record) (project.ProjectReadiness, error) {
	return s.readiness, nil
}

func (s fakeProjectStore) GeneratedWriteReadiness(context.Context, project.Record, project.GeneratedWriteReadinessOptions) (project.GeneratedWriteReadiness, error) {
	return s.generatedWriteReadiness, nil
}

func (s fakeProjectStore) GeneratedWriteApplyBetaGate(context.Context, project.Record, project.GeneratedWriteApplyBetaGateOptions) (project.GeneratedWriteApplyBetaGate, error) {
	return s.generatedApplyBetaGate, nil
}

func (s fakeProjectStore) PermissionPolicyDoctor(context.Context, project.Record, project.PermissionPolicyDoctorOptions) (project.PermissionPolicyDoctor, error) {
	if s.err != nil {
		return project.PermissionPolicyDoctor{}, s.err
	}
	return s.permission, nil
}

func (s fakeProjectStore) ConformanceCheck(context.Context, project.Record, project.ConformanceOptions) (project.ConformanceReport, error) {
	if s.err != nil {
		return project.ConformanceReport{}, s.err
	}
	return s.conformance, nil
}

func (s fakeProjectStore) ArtifactIntegrity(context.Context, project.Record, project.ArtifactIntegrityOptions) (project.ArtifactIntegrityReport, error) {
	if s.err != nil {
		return project.ArtifactIntegrityReport{}, s.err
	}
	return s.integrity, nil
}

func (s fakeProjectStore) ArtifactArchivePreview(_ context.Context, _ project.Record, options project.ArtifactArchivePreviewOptions) (project.ArtifactArchivePreviewResult, error) {
	if s.archivePreviewHook != nil {
		s.archivePreviewHook(options)
	}
	if s.err != nil {
		return project.ArtifactArchivePreviewResult{}, s.err
	}
	return s.archivePreview, nil
}

func (s fakeProjectStore) ProjectImportDiff(context.Context, project.Record) (project.ProjectImportDiff, error) {
	return s.diff, nil
}

func (s fakeProjectStore) ProjectVerificationBundle(context.Context, project.Record, int) (project.ProjectVerificationBundle, error) {
	return s.bundle, nil
}

func (s fakeProjectStore) CompatibilityContract(context.Context, project.Record) (project.CompatibilityContract, error) {
	return s.compat, nil
}

func (s fakeProjectStore) ShimPreview(context.Context, project.Record) (project.ShimPreview, error) {
	return s.shimPreview, nil
}

func (s fakeProjectStore) ShimReadiness(context.Context, project.Record) (project.ShimReadiness, error) {
	return s.shimReadiness, nil
}

func (s fakeProjectStore) ShimAuthorizationPacket(context.Context, project.Record) (project.ShimAuthorizationPacket, error) {
	return s.shimAuthorization, nil
}

func (s fakeProjectStore) ShimApplyPacketPreview(_ context.Context, _ project.Record, options project.ShimApplyPacketPreviewOptions) (project.ShimApplyPacketPreview, error) {
	if s.shimApplyPacketHook != nil {
		s.shimApplyPacketHook(options)
	}
	if s.err != nil {
		return project.ShimApplyPacketPreview{}, s.err
	}
	return s.shimApplyPacket, nil
}

func (s fakeProjectStore) ShimApplyGate(_ context.Context, _ project.Record, options project.ShimApplyGateOptions) (project.ShimApplyGate, error) {
	if s.shimApplyGateHook != nil {
		s.shimApplyGateHook(options)
	}
	if s.err != nil {
		return project.ShimApplyGate{}, s.err
	}
	return s.shimApplyGate, nil
}

func (s fakeProjectStore) ApplyShimCommand(_ context.Context, _ project.Record, options project.ApplyShimCommandOptions) (project.ApplyShimCommandResult, error) {
	if s.shimApplyCommandHook != nil {
		s.shimApplyCommandHook(options)
	}
	if s.err != nil {
		return project.ApplyShimCommandResult{}, s.err
	}
	return s.shimApplyCommand, nil
}

func (s fakeProjectStore) RecordShimReadinessEvidence(_ context.Context, _ project.Record, options project.RecordShimReadinessEvidenceOptions) (project.RecordShimReadinessEvidenceResult, error) {
	if s.shimEvidenceHook != nil {
		s.shimEvidenceHook(options)
	}
	if s.err != nil {
		return project.RecordShimReadinessEvidenceResult{}, s.err
	}
	return s.shimEvidence, nil
}

func (s fakeProjectStore) AreaMatrixExecutionCutoverReadiness(context.Context, project.Record, project.AreaMatrixExecutionCutoverReadinessOptions) (project.AreaMatrixExecutionCutoverReadiness, error) {
	if s.err != nil {
		return project.AreaMatrixExecutionCutoverReadiness{}, s.err
	}
	return s.executionCutover, nil
}

func (s fakeProjectStore) ExecutionForwardingV1Readiness(context.Context, project.Record, project.ExecutionForwardingV1ReadinessOptions) (project.ExecutionForwardingV1Readiness, error) {
	if s.err != nil {
		return project.ExecutionForwardingV1Readiness{}, s.err
	}
	return s.executionForwarding, nil
}

func (s fakeProjectStore) ExecutionForwardingV1ApplyPreview(context.Context, project.Record, project.ExecutionForwardingV1ApplyPreviewOptions) (project.ExecutionForwardingV1ApplyPreview, error) {
	if s.err != nil {
		return project.ExecutionForwardingV1ApplyPreview{}, s.err
	}
	return s.executionForwardingApply, nil
}

func (s fakeProjectStore) ExecutionForwardingV1ApplyPacketPreview(_ context.Context, _ project.Record, options project.ExecutionForwardingV1ApplyPacketPreviewOptions) (project.ExecutionForwardingV1ApplyPacketPreview, error) {
	if s.executionForwardingPacketHook != nil {
		s.executionForwardingPacketHook(options)
	}
	if s.err != nil {
		return project.ExecutionForwardingV1ApplyPacketPreview{}, s.err
	}
	return s.executionForwardingPacket, nil
}

func (s fakeProjectStore) ExecutionForwardingV1ApplyGate(_ context.Context, _ project.Record, options project.ExecutionForwardingV1ApplyGateOptions) (project.ExecutionForwardingV1ApplyGate, error) {
	if s.executionForwardingGateHook != nil {
		s.executionForwardingGateHook(options)
	}
	if s.err != nil {
		return project.ExecutionForwardingV1ApplyGate{}, s.err
	}
	return s.executionForwardingGate, nil
}

func (s fakeProjectStore) ApplyExecutionForwardingV1(_ context.Context, _ project.Record, options project.ApplyExecutionForwardingV1Options) (project.ApplyExecutionForwardingV1Result, error) {
	if s.executionForwardingApplyHook != nil {
		s.executionForwardingApplyHook(options)
	}
	if s.err != nil {
		return project.ApplyExecutionForwardingV1Result{}, s.err
	}
	return s.executionForwardingApplyCmd, nil
}

func (s fakeProjectStore) ExecutionForwardingV1CommandPreview(_ context.Context, _ project.Record, options project.ExecutionForwardingV1CommandPreviewOptions) (project.ExecutionForwardingV1CommandPreview, error) {
	if s.err != nil {
		return project.ExecutionForwardingV1CommandPreview{}, s.err
	}
	result := s.executionForwardingCmd
	if result.TaskType == "" {
		result.TaskType = options.TaskType
	}
	return result, nil
}

func (s fakeProjectStore) ExecutionForwardingV1RollbackPreview(context.Context, project.Record, project.ExecutionForwardingV1RollbackPreviewOptions) (project.ExecutionForwardingV1RollbackPreview, error) {
	if s.err != nil {
		return project.ExecutionForwardingV1RollbackPreview{}, s.err
	}
	return s.executionForwardingRoll, nil
}

func (s fakeProjectStore) ProjectCutoverReadiness(context.Context, project.Record, string, int) (project.ProjectCutoverReadiness, error) {
	return s.cutover, nil
}

func (s fakeProjectStore) ApplyCutover(_ context.Context, _ project.Record, options project.ApplyCutoverOptions) (project.ApplyCutoverResult, error) {
	if s.cutoverApplyHook != nil {
		s.cutoverApplyHook(options)
	}
	if s.err != nil {
		return project.ApplyCutoverResult{}, s.err
	}
	return s.cutoverApply, nil
}

func (s fakeProjectStore) ListStatusProjections(context.Context, project.Record, int) ([]project.StatusProjectionRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.statusProjections, nil
}

func (s fakeProjectStore) StatusProjectionAuthorizationPreview(_ context.Context, _ project.Record, options project.StatusProjectionAuthorizationPreviewOptions) (project.StatusProjectionAuthorizationPreview, error) {
	if s.statusAuthorizationHook != nil {
		s.statusAuthorizationHook(options)
	}
	if s.err != nil {
		return project.StatusProjectionAuthorizationPreview{}, s.err
	}
	return s.statusAuthorization, nil
}

func (s fakeProjectStore) StatusProjectionApplyPacketPreview(_ context.Context, _ project.Record, options project.StatusProjectionApplyPacketPreviewOptions) (project.StatusProjectionApplyPacketPreview, error) {
	if s.statusApplyPacketHook != nil {
		s.statusApplyPacketHook(options)
	}
	if s.err != nil {
		return project.StatusProjectionApplyPacketPreview{}, s.err
	}
	return s.statusApplyPacket, nil
}

func (s fakeProjectStore) StatusProjectionApplyGate(_ context.Context, _ project.Record, options project.StatusProjectionApplyGateOptions) (project.StatusProjectionApplyGate, error) {
	if s.statusApplyGateHook != nil {
		s.statusApplyGateHook(options)
	}
	if s.err != nil {
		return project.StatusProjectionApplyGate{}, s.err
	}
	return s.statusApplyGate, nil
}

func (s fakeProjectStore) ApplyStatusProjection(_ context.Context, _ project.Record, options project.ApplyStatusProjectionOptions) (project.ApplyStatusProjectionResult, error) {
	if s.statusApplyHook != nil {
		s.statusApplyHook(options)
	}
	if s.err != nil {
		return project.ApplyStatusProjectionResult{}, s.err
	}
	return s.statusApply, nil
}

func (s fakeProjectStore) PreviewRunner(context.Context, project.Record, string, project.RunnerPreviewOptions) (project.RunnerPreviewResult, error) {
	if s.err != nil {
		return project.RunnerPreviewResult{}, s.err
	}
	return s.runner, nil
}

func (s fakeProjectStore) QueueFixtureExecution(_ context.Context, _ project.Record, _ string, options project.FixtureExecutionQueueOptions) (project.FixtureExecutionQueueResult, error) {
	if s.fixtureQueueHook != nil {
		s.fixtureQueueHook(options)
	}
	if s.err != nil {
		return project.FixtureExecutionQueueResult{}, s.err
	}
	return s.fixtureQueue, nil
}

func (s fakeProjectStore) QueueReadOnlyVerify(_ context.Context, _ project.Record, _ string, options project.ReadOnlyVerifyQueueOptions) (project.ReadOnlyVerifyQueueResult, error) {
	if s.readOnlyQueueHook != nil {
		s.readOnlyQueueHook(options)
	}
	if s.err != nil {
		return project.ReadOnlyVerifyQueueResult{}, s.err
	}
	return s.readOnlyQueue, nil
}

func (s fakeProjectStore) QueueApprovedArtifactWrite(_ context.Context, _ project.Record, _ string, options project.ApprovedArtifactWriteQueueOptions) (project.ApprovedArtifactWriteQueueResult, error) {
	if s.artifactWriteQueueHook != nil {
		s.artifactWriteQueueHook(options)
	}
	if s.err != nil {
		return project.ApprovedArtifactWriteQueueResult{}, s.err
	}
	return s.artifactWriteQueue, nil
}

func (s fakeProjectStore) QueueFixtureProjectWrite(_ context.Context, _ project.Record, _ string, options project.FixtureProjectWriteQueueOptions) (project.FixtureProjectWriteQueueResult, error) {
	if s.fixtureProjectQueueHook != nil {
		s.fixtureProjectQueueHook(options)
	}
	if s.err != nil {
		return project.FixtureProjectWriteQueueResult{}, s.err
	}
	return s.fixtureProjectQueue, nil
}

func (s fakeProjectStore) QueueManagedGeneratedWrite(_ context.Context, _ project.Record, _ string, options project.ManagedGeneratedWriteQueueOptions) (project.ManagedGeneratedWriteQueueResult, error) {
	if s.managedGeneratedQueueHook != nil {
		s.managedGeneratedQueueHook(options)
	}
	if s.err != nil {
		return project.ManagedGeneratedWriteQueueResult{}, s.err
	}
	return s.managedGeneratedQueue, nil
}

func (s fakeProjectStore) RegisterWorker(_ context.Context, _ project.Record, options project.RegisterWorkerOptions) (project.WorkerRecord, error) {
	if s.workerRegisterHook != nil {
		s.workerRegisterHook(options)
	}
	if s.workerErr != nil {
		return project.WorkerRecord{}, s.workerErr
	}
	if s.err != nil {
		return project.WorkerRecord{}, s.err
	}
	return s.worker, nil
}

func (s fakeProjectStore) RecordWorkerHeartbeat(_ context.Context, _ project.Record, workerKey string, options project.WorkerHeartbeatOptions) (project.WorkerRecord, error) {
	if s.workerHeartbeatHook != nil {
		s.workerHeartbeatHook(workerKey, options)
	}
	if s.workerErr != nil {
		return project.WorkerRecord{}, s.workerErr
	}
	if s.err != nil {
		return project.WorkerRecord{}, s.err
	}
	return s.worker, nil
}

func (s fakeProjectStore) ListWorkers(_ context.Context, record project.Record, limit int) ([]project.WorkerRecord, error) {
	if s.workersHook != nil {
		return s.workersHook(record, limit), nil
	}
	return s.workers, nil
}

func (s fakeProjectStore) WorkerPoolSummary(context.Context) (project.WorkerPoolSummary, error) {
	if s.err != nil {
		return project.WorkerPoolSummary{}, s.err
	}
	return s.workerPool, nil
}

func (s fakeProjectStore) WorkerPoolSchedulePreview(context.Context) (project.WorkerPoolSchedulePreview, error) {
	if s.err != nil {
		return project.WorkerPoolSchedulePreview{}, s.err
	}
	return s.schedule, nil
}

func (s fakeProjectStore) CodexCLIAdapterPreview(_ context.Context, _ project.Record, options project.CodexCLIAdapterPreviewOptions) (project.CodexCLIAdapterPreview, error) {
	if s.codexPreviewHook != nil {
		s.codexPreviewHook(options)
	}
	if s.err != nil {
		return project.CodexCLIAdapterPreview{}, s.err
	}
	return s.codexPreview, nil
}

func (s fakeProjectStore) LocalServiceStatus(context.Context, project.LocalServiceStatusOptions) (project.LocalServiceStatus, error) {
	if s.err != nil {
		return project.LocalServiceStatus{}, s.err
	}
	return s.service, nil
}

func (s fakeProjectStore) RestorePlan(_ context.Context, options project.RestorePlanOptions) (project.RestorePlan, error) {
	if s.restoreHook != nil {
		s.restoreHook(options)
	}
	if s.err != nil {
		return project.RestorePlan{}, s.err
	}
	return s.restore, nil
}

func (s fakeProjectStore) ReleaseReadiness(_ context.Context, options project.ReleaseReadinessOptions) (project.ReleaseReadiness, error) {
	if s.releaseHook != nil {
		s.releaseHook(options)
	}
	if s.err != nil {
		return project.ReleaseReadiness{}, s.err
	}
	return s.release, nil
}

func (s fakeProjectStore) ReleaseRemediationPlan(_ context.Context, options project.ReleaseRemediationOptions) (project.ReleaseRemediationPlan, error) {
	if s.remediationHook != nil {
		s.remediationHook(options)
	}
	if s.err != nil {
		return project.ReleaseRemediationPlan{}, s.err
	}
	return s.remediation, nil
}

func (s fakeProjectStore) ReleaseAcceptancePreview(_ context.Context, options project.ReleaseAcceptancePreviewOptions) (project.ReleaseAcceptancePreview, error) {
	if s.acceptanceHook != nil {
		s.acceptanceHook(options)
	}
	if s.err != nil {
		return project.ReleaseAcceptancePreview{}, s.err
	}
	return s.acceptance, nil
}

func (s fakeProjectStore) ReleaseAcceptanceGate(_ context.Context, options project.ReleaseAcceptanceGateOptions) (project.ReleaseAcceptanceGate, error) {
	if s.gateReleaseHook != nil {
		s.gateReleaseHook(options)
	}
	if s.err != nil {
		return project.ReleaseAcceptanceGate{}, s.err
	}
	return s.gateRelease, nil
}

func (s fakeProjectStore) ReleaseExceptionDoctor(_ context.Context, options project.ReleaseExceptionDoctorOptions) (project.ReleaseExceptionDoctor, error) {
	if s.exceptionHook != nil {
		s.exceptionHook(options)
	}
	if s.err != nil {
		return project.ReleaseExceptionDoctor{}, s.err
	}
	return s.exception, nil
}

func (s fakeProjectStore) ReleaseExceptionRecordPreview(_ context.Context, options project.ReleaseExceptionRecordPreviewOptions) (project.ReleaseExceptionRecordPreview, error) {
	if s.recordPrevHook != nil {
		s.recordPrevHook(options)
	}
	if s.err != nil {
		return project.ReleaseExceptionRecordPreview{}, s.err
	}
	return s.recordPrev, nil
}

func (s fakeProjectStore) ReleaseExceptionSchemaPreview(_ context.Context, options project.ReleaseExceptionSchemaPreviewOptions) (project.ReleaseExceptionSchemaPreview, error) {
	if s.schemaPrevHook != nil {
		s.schemaPrevHook(options)
	}
	if s.err != nil {
		return project.ReleaseExceptionSchemaPreview{}, s.err
	}
	return s.schemaPrev, nil
}

func (s fakeProjectStore) ReleaseExceptionMigrationApprovalGate(_ context.Context, options project.ReleaseExceptionMigrationApprovalGateOptions) (project.ReleaseExceptionMigrationApprovalGate, error) {
	if s.migrationHook != nil {
		s.migrationHook(options)
	}
	if s.err != nil {
		return project.ReleaseExceptionMigrationApprovalGate{}, s.err
	}
	return s.migration, nil
}

func (s fakeProjectStore) ReleaseExceptionApplyPreview(_ context.Context, options project.ReleaseExceptionApplyPreviewOptions) (project.ReleaseExceptionApplyPreview, error) {
	if s.applyPrevHook != nil {
		s.applyPrevHook(options)
	}
	if s.err != nil {
		return project.ReleaseExceptionApplyPreview{}, s.err
	}
	return s.applyPrev, nil
}

func (s fakeProjectStore) ReleaseFinalGate(_ context.Context, options project.ReleaseFinalGateOptions) (project.ReleaseFinalGate, error) {
	if s.finalGateHook != nil {
		s.finalGateHook(options)
	}
	if s.err != nil {
		return project.ReleaseFinalGate{}, s.err
	}
	return s.finalGate, nil
}

func (s fakeProjectStore) ReleaseEvidenceBundle(_ context.Context, options project.ReleaseEvidenceBundleOptions) (project.ReleaseEvidenceBundle, error) {
	if s.evidenceHook != nil {
		s.evidenceHook(options)
	}
	if s.err != nil {
		return project.ReleaseEvidenceBundle{}, s.err
	}
	return s.evidence, nil
}

func (s fakeProjectStore) ReleasePackagePreview(_ context.Context, options project.ReleasePackagePreviewOptions) (project.ReleasePackagePreview, error) {
	if s.packagePrevHook != nil {
		s.packagePrevHook(options)
	}
	if s.err != nil {
		return project.ReleasePackagePreview{}, s.err
	}
	return s.packagePrev, nil
}

func (s fakeProjectStore) ReleaseDistributionPreview(_ context.Context, options project.ReleaseDistributionPreviewOptions) (project.ReleaseDistributionPreview, error) {
	if s.distPrevHook != nil {
		s.distPrevHook(options)
	}
	if s.err != nil {
		return project.ReleaseDistributionPreview{}, s.err
	}
	return s.distPrev, nil
}

func (s fakeProjectStore) ReleasePublishGate(_ context.Context, options project.ReleasePublishGateOptions) (project.ReleasePublishGate, error) {
	if s.publishGateHook != nil {
		s.publishGateHook(options)
	}
	if s.err != nil {
		return project.ReleasePublishGate{}, s.err
	}
	return s.publishGate, nil
}

func (s fakeProjectStore) ReleasePublishApprovalPreview(_ context.Context, options project.ReleasePublishApprovalPreviewOptions) (project.ReleasePublishApprovalPreview, error) {
	if s.publishApprHook != nil {
		s.publishApprHook(options)
	}
	if s.err != nil {
		return project.ReleasePublishApprovalPreview{}, s.err
	}
	return s.publishAppr, nil
}

func (s fakeProjectStore) ReleaseRolloutPlanPreview(_ context.Context, options project.ReleaseRolloutPlanPreviewOptions) (project.ReleaseRolloutPlanPreview, error) {
	if s.rolloutHook != nil {
		s.rolloutHook(options)
	}
	if s.err != nil {
		return project.ReleaseRolloutPlanPreview{}, s.err
	}
	return s.rollout, nil
}

func (s fakeProjectStore) WebWriteActionGate(context.Context, project.WebWriteActionGateOptions) (project.WebWriteActionGate, error) {
	if s.err != nil {
		return project.WebWriteActionGate{}, s.err
	}
	return s.webGate, nil
}

func (s fakeProjectStore) DesktopServiceControlGate(context.Context, project.DesktopServiceControlGateOptions) (project.DesktopServiceControlGate, error) {
	if s.err != nil {
		return project.DesktopServiceControlGate{}, s.err
	}
	return s.desktopGate, nil
}

func (s fakeProjectStore) DesktopNotificationGate(context.Context, project.DesktopNotificationGateOptions) (project.DesktopNotificationGate, error) {
	if s.err != nil {
		return project.DesktopNotificationGate{}, s.err
	}
	return s.notificationGate, nil
}

func (s fakeProjectStore) DesktopTrayMenuGate(context.Context, project.DesktopTrayMenuGateOptions) (project.DesktopTrayMenuGate, error) {
	if s.err != nil {
		return project.DesktopTrayMenuGate{}, s.err
	}
	return s.trayMenuGate, nil
}

func (s fakeProjectStore) SecurityBoundaryReadiness(context.Context, project.SecurityBoundaryReadinessOptions) (project.SecurityBoundaryReadiness, error) {
	if s.err != nil {
		return project.SecurityBoundaryReadiness{}, s.err
	}
	return s.security, nil
}

func (s fakeProjectStore) CompletionAudit(context.Context, project.CompletionAuditOptions) (project.CompletionAudit, error) {
	if s.err != nil {
		return project.CompletionAudit{}, s.err
	}
	return s.completion, nil
}

func (s fakeProjectStore) CompletionAuditSnapshotReadiness(context.Context, project.Record) (project.CompletionAuditSnapshotReadiness, error) {
	if s.err != nil {
		return project.CompletionAuditSnapshotReadiness{}, s.err
	}
	return s.completionSnapshot, nil
}

func (s fakeProjectStore) SupportBundlePreview(context.Context, project.SupportBundlePreviewOptions) (project.SupportBundlePreview, error) {
	if s.err != nil {
		return project.SupportBundlePreview{}, s.err
	}
	return s.supportBundle, nil
}

func (s fakeProjectStore) MigrationLedgerReadiness(context.Context, project.MigrationLedgerReadinessOptions) (project.MigrationLedgerReadiness, error) {
	if s.err != nil {
		return project.MigrationLedgerReadiness{}, s.err
	}
	return s.migrationLedger, nil
}

func (s fakeProjectStore) OperationsReadiness(context.Context, project.OperationsReadinessOptions) (project.OperationsReadiness, error) {
	if s.err != nil {
		return project.OperationsReadiness{}, s.err
	}
	return s.operations, nil
}

func (s fakeProjectStore) BackupManifest(_ context.Context, options project.BackupManifestOptions) (project.BackupManifest, error) {
	if s.backupHook != nil {
		s.backupHook(options)
	}
	if s.err != nil {
		return project.BackupManifest{}, s.err
	}
	return s.backup, nil
}

func (s fakeProjectStore) AcquireLease(_ context.Context, _ project.Record, options project.AcquireLeaseOptions) (project.LeaseRecord, error) {
	if s.leaseAcquireHook != nil {
		s.leaseAcquireHook(options)
	}
	if s.leaseErr != nil {
		return project.LeaseRecord{}, s.leaseErr
	}
	if s.err != nil {
		return project.LeaseRecord{}, s.err
	}
	return s.lease, nil
}

func (s fakeProjectStore) ReleaseLease(_ context.Context, _ project.Record, options project.ReleaseLeaseOptions) (project.LeaseRecord, error) {
	if s.leaseReleaseHook != nil {
		s.leaseReleaseHook(options)
	}
	if s.err != nil {
		return project.LeaseRecord{}, s.err
	}
	return s.lease, nil
}

func (s fakeProjectStore) RecoverExpiredLeases(_ context.Context, _ project.Record, options project.RecoverLeasesOptions) ([]project.LeaseRecord, error) {
	if s.leaseRecoverHook != nil {
		s.leaseRecoverHook(options)
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.leases, nil
}

func (s fakeProjectStore) RunWorkerOnce(_ context.Context, _ project.Record, options project.WorkerRunOnceOptions) (project.WorkerRunOnceResult, error) {
	if s.runOnceHook != nil {
		s.runOnceHook(options)
	}
	if s.runOnceErr != nil {
		return project.WorkerRunOnceResult{}, s.runOnceErr
	}
	if s.err != nil {
		return project.WorkerRunOnceResult{}, s.err
	}
	return s.runOnce, nil
}

func (s fakeProjectStore) ExecuteFixture(_ context.Context, _ project.Record, options project.FixtureExecutionOptions) (project.FixtureExecutionResult, error) {
	if s.fixtureExecuteHook != nil {
		s.fixtureExecuteHook(options)
	}
	if s.runOnceErr != nil {
		return project.FixtureExecutionResult{}, s.runOnceErr
	}
	if s.err != nil {
		return project.FixtureExecutionResult{}, s.err
	}
	return s.fixtureExecute, nil
}

func (s fakeProjectStore) VerifyReadOnly(_ context.Context, _ project.Record, options project.ReadOnlyVerifyOptions) (project.ReadOnlyVerifyResult, error) {
	if s.readOnlyVerifyHook != nil {
		s.readOnlyVerifyHook(options)
	}
	if s.runOnceErr != nil {
		return project.ReadOnlyVerifyResult{}, s.runOnceErr
	}
	if s.err != nil {
		return project.ReadOnlyVerifyResult{}, s.err
	}
	return s.readOnlyVerify, nil
}

func (s fakeProjectStore) WriteApprovedArtifact(_ context.Context, _ project.Record, options project.ApprovedArtifactWriteOptions) (project.ApprovedArtifactWriteResult, error) {
	if s.artifactWriteHook != nil {
		s.artifactWriteHook(options)
	}
	if s.runOnceErr != nil {
		return project.ApprovedArtifactWriteResult{}, s.runOnceErr
	}
	if s.err != nil {
		return project.ApprovedArtifactWriteResult{}, s.err
	}
	return s.artifactWrite, nil
}

func (s fakeProjectStore) WriteFixtureProject(_ context.Context, _ project.Record, options project.FixtureProjectWriteOptions) (project.FixtureProjectWriteResult, error) {
	if s.fixtureProjectWriteHook != nil {
		s.fixtureProjectWriteHook(options)
	}
	if s.runOnceErr != nil {
		return project.FixtureProjectWriteResult{}, s.runOnceErr
	}
	if s.err != nil {
		return project.FixtureProjectWriteResult{}, s.err
	}
	return s.fixtureProjectWrite, nil
}

func (s fakeProjectStore) WriteManagedGenerated(_ context.Context, _ project.Record, options project.ManagedGeneratedWriteOptions) (project.ManagedGeneratedWriteResult, error) {
	if s.managedGeneratedWriteHook != nil {
		s.managedGeneratedWriteHook(options)
	}
	if s.runOnceErr != nil {
		return project.ManagedGeneratedWriteResult{}, s.runOnceErr
	}
	if s.err != nil {
		return project.ManagedGeneratedWriteResult{}, s.err
	}
	return s.managedGeneratedWrite, nil
}

func (s fakeProjectStore) GetRun(context.Context, int64) (project.RunDetail, error) {
	if s.runErr != nil {
		return project.RunDetail{}, s.runErr
	}
	return s.runDetail, nil
}

func (s fakeProjectStore) ExecutionApprovalGate(context.Context, int64, project.ExecutionApprovalGateOptions) (project.ExecutionApprovalGate, error) {
	if s.runErr != nil {
		return project.ExecutionApprovalGate{}, s.runErr
	}
	return s.executionGate, nil
}

func (s fakeProjectStore) PreviewExecutionPlan(context.Context, int64) (project.ExecutionPlanPreview, error) {
	if s.runErr != nil {
		return project.ExecutionPlanPreview{}, s.runErr
	}
	return s.executionPlan, nil
}

func (s fakeProjectStore) PreviewProjectWriteDesignGate(context.Context, int64) (project.ProjectWriteDesignGate, error) {
	if s.runErr != nil {
		return project.ProjectWriteDesignGate{}, s.runErr
	}
	return s.projectWriteDesignGate, nil
}

func (s fakeProjectStore) PreviewManagedGeneratedWriteGate(context.Context, int64) (project.ManagedGeneratedWriteGate, error) {
	if s.runErr != nil {
		return project.ManagedGeneratedWriteGate{}, s.runErr
	}
	return s.managedGeneratedGate, nil
}

func (s fakeProjectStore) ControlRun(_ context.Context, _ int64, options project.RunControlOptions) (project.RunControlResult, error) {
	if s.runControlHook != nil {
		s.runControlHook(options)
	}
	if s.runErr != nil {
		return project.RunControlResult{}, s.runErr
	}
	if s.err != nil {
		return project.RunControlResult{}, s.err
	}
	return s.runControl, nil
}

func (s fakeProjectStore) ListWorkflowVersionRuns(_ context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.RunRecord, error) {
	if s.runsHook != nil {
		return s.runsHook(record, version, limit), nil
	}
	if s.runErr != nil {
		return nil, s.runErr
	}
	return s.runs, nil
}

func (s fakeProjectStore) ListRunEvents(context.Context, int64, int) ([]project.EventRecord, error) {
	if s.runErr != nil {
		return nil, s.runErr
	}
	return s.runEvents, nil
}

func (s fakeProjectStore) ListProjectArtifacts(_ context.Context, record project.Record, limit int) ([]project.ArtifactRecord, error) {
	if s.artifactsHook != nil {
		return s.artifactsHook(record, limit), nil
	}
	if s.artifactErr != nil {
		return nil, s.artifactErr
	}
	return s.artifacts, nil
}

func (s fakeProjectStore) ListWorkflowVersionArtifacts(_ context.Context, record project.Record, version project.WorkflowVersion, limit int) ([]project.ArtifactRecord, error) {
	if s.versionArtifactsHook != nil {
		return s.versionArtifactsHook(record, version, limit), nil
	}
	if s.artifactErr != nil {
		return nil, s.artifactErr
	}
	return s.artifacts, nil
}

func (s fakeProjectStore) ListProjectResiduals(context.Context, project.Record, int) ([]project.ResidualRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.residuals, nil
}

func (s fakeProjectStore) ListWorkflowVersionResiduals(context.Context, project.Record, project.WorkflowVersion, int) ([]project.ResidualRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.residuals, nil
}

func (s fakeProjectStore) GetArtifact(context.Context, int64) (project.ArtifactRecord, error) {
	if s.artifactErr != nil {
		return project.ArtifactRecord{}, s.artifactErr
	}
	return s.artifact, nil
}

func (s fakeProjectStore) GetArtifactContent(context.Context, int64) (project.ArtifactContent, error) {
	if s.artifactContentErr != nil {
		return project.ArtifactContent{}, s.artifactContentErr
	}
	return s.artifactContent, nil
}

func TestHealthEndpoint(t *testing.T) {
	addr, cleanup, err := ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	cleanup()

	host, port, ok := splitAddr(addr)
	if !ok {
		t.Fatalf("unexpected addr: %s", addr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeHandler(ctx, config.ServerConfig{Host: host, Port: port}, NewHandler(fakeProjectStore{}))
	}()

	var resp *http.Response
	for range 50 {
		resp, err = http.Get("http://" + addr + "/api/health")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	var body healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Status != "ok" || body.Service != "areaflow" {
		t.Fatalf("unexpected health response: %+v", body)
	}

	cancel()
	if err := <-errCh; err != context.Canceled {
		t.Fatalf("server error = %v, want context.Canceled", err)
	}
}

func TestReadinessEndpoint(t *testing.T) {
	for _, test := range []struct {
		name       string
		store      fakeProjectStore
		wantStatus int
		wantBody   string
	}{
		{name: "ready", wantStatus: http.StatusOK, wantBody: `"database":"ready"`},
		{name: "database unavailable", store: fakeProjectStore{pingErr: errors.New("database unavailable")}, wantStatus: http.StatusServiceUnavailable, wantBody: `"database":"unavailable"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			NewHandler(test.store).ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil))
			if resp.Code != test.wantStatus || !strings.Contains(resp.Body.String(), test.wantBody) {
				t.Fatalf("readiness status = %d body=%s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestReadinessEndpointFailsClosedForArtifactStore(t *testing.T) {
	server := Server{store: fakeProjectStore{}, doctorRunner: runProjectDoctor, artifactReadiness: fakeReadinessCheck{err: errors.New("S3 unavailable")}}
	resp := httptest.NewRecorder()
	server.handler().ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil))
	if resp.Code != http.StatusServiceUnavailable || !strings.Contains(resp.Body.String(), `"artifact_store":"unavailable"`) {
		t.Fatalf("readiness status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRequestTimeoutMiddlewareBoundsOrdinaryRequestsButNotSSE(t *testing.T) {
	server := Server{requestTimeout: 10 * time.Millisecond}
	ordinaryDeadline := make(chan bool, 1)
	ordinary := server.requestTimeoutMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		ordinaryDeadline <- ok
	}))
	ordinary.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if !<-ordinaryDeadline {
		t.Fatal("ordinary request must receive a deadline")
	}

	sseDeadline := make(chan bool, 1)
	sse := server.requestTimeoutMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		sseDeadline <- ok
	}))
	sse.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/events/stream", nil))
	if <-sseDeadline {
		t.Fatal("SSE request must not receive the ordinary request timeout")
	}
}

func TestValidateProductionProjectArtifacts(t *testing.T) {
	if err := validateProductionProjectArtifacts(context.Background(), fakeProjectStore{records: []project.Record{{Key: "area", ArtifactBackend: "s3"}}}); err != nil {
		t.Fatalf("S3 project rejected: %v", err)
	}
	if err := validateProductionProjectArtifacts(context.Background(), fakeProjectStore{records: []project.Record{{Key: "area", ArtifactBackend: "local"}}}); err == nil {
		t.Fatal("production local artifact backend must be rejected")
	}
}

func TestTokenAuthenticationMiddleware(t *testing.T) {
	store := fakeProjectStore{records: []project.Record{{ID: 1, Key: "area"}, {ID: 2, Key: "other"}}}
	principal := auth.Principal{
		TokenID: 1, TokenKey: "token-key", Actor: "operator", Projects: []string{"area"}, Capabilities: []string{"read"}, ScopeHash: "scope-hash",
	}
	authenticator := &fakeTokenAuthenticator{principal: principal}
	handler := NewHandlerWithAuth(store, nil, config.AuthConfig{Mode: "token"}, authenticator)

	statusResp := httptest.NewRecorder()
	handler.ServeHTTP(statusResp, httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil))
	if statusResp.Code != http.StatusOK || !strings.Contains(statusResp.Body.String(), `"requires_token":true`) {
		t.Fatalf("auth status = %d body=%s", statusResp.Code, statusResp.Body.String())
	}

	missingResp := httptest.NewRecorder()
	handler.ServeHTTP(missingResp, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if missingResp.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d body=%s", missingResp.Code, missingResp.Body.String())
	}

	projectsReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	projectsReq.Header.Set("Authorization", "Bearer raw-token")
	projectsResp := httptest.NewRecorder()
	handler.ServeHTTP(projectsResp, projectsReq)
	if projectsResp.Code != http.StatusOK || !strings.Contains(projectsResp.Body.String(), `"key":"area"`) || strings.Contains(projectsResp.Body.String(), `"key":"other"`) {
		t.Fatalf("scoped project list = %d body=%s", projectsResp.Code, projectsResp.Body.String())
	}
	if authenticator.rawToken != "raw-token" {
		t.Fatalf("raw token = %q", authenticator.rawToken)
	}

	crossProjectReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/other", nil)
	crossProjectReq.Header.Set("Authorization", "Bearer raw-token")
	crossProjectResp := httptest.NewRecorder()
	handler.ServeHTTP(crossProjectResp, crossProjectReq)
	if crossProjectResp.Code != http.StatusNotFound {
		t.Fatalf("cross-project status = %d body=%s", crossProjectResp.Code, crossProjectResp.Body.String())
	}

	approvalReq := httptest.NewRequest(http.MethodPost, "/api/v1/projects/area/workflow-versions/v1/approvals", strings.NewReader(`{}`))
	approvalReq.Header.Set("Authorization", "Bearer raw-token")
	approvalResp := httptest.NewRecorder()
	handler.ServeHTTP(approvalResp, approvalReq)
	if approvalResp.Code != http.StatusForbidden {
		t.Fatalf("approval capability status = %d body=%s", approvalResp.Code, approvalResp.Body.String())
	}
}

func TestTokenAuthenticationUsesPathProjectScope(t *testing.T) {
	store := fakeProjectStore{records: []project.Record{{ID: 1, Key: "area"}, {ID: 2, Key: "other"}}}
	authenticator := &fakeTokenAuthenticator{principal: auth.Principal{
		TokenKey: "token-key", Actor: "operator", Projects: []string{"area"}, Capabilities: []string{"read"},
	}}
	handler := NewHandlerWithAuth(store, nil, config.AuthConfig{Mode: "token"}, authenticator)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/other?project_key=area", nil)
	request.Header.Set("Authorization", "Bearer raw-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("path project override status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestTokenAuthenticationRequiresWriteCapability(t *testing.T) {
	store := fakeProjectStore{records: []project.Record{{ID: 1, Key: "area"}}}
	authenticator := &fakeTokenAuthenticator{principal: auth.Principal{
		TokenKey: "token-key", Actor: "approver", Projects: []string{"area"}, Capabilities: []string{"workflow.approval.record"},
	}}
	handler := NewHandlerWithAuth(store, nil, config.AuthConfig{Mode: "token"}, authenticator)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/area/import", strings.NewReader(`{"actor":"forged"}`))
	request.Header.Set("Authorization", "Bearer raw-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-approval write status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestAPIV1Alias(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		records: []project.Record{
			{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Kind: "desktop-app", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		},
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("api v1 alias status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode api v1 alias response: %v", err)
	}
	if len(body.Projects) != 1 || body.Projects[0].Key != "areamatrix" {
		t.Fatalf("unexpected api v1 alias projects: %+v", body.Projects)
	}
}

func TestProjectListEndpoint(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		records: []project.Record{
			{ID: 1, Key: "areaflow", Name: "AreaFlow", Kind: "platform", Adapter: "git-repo", WorkflowProfile: "areaflow-standard", RootPath: "/tmp/areaflow"},
			{ID: 2, Key: "areamatrix", Name: "AreaMatrix", Kind: "desktop-app", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project list status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode project list: %v", err)
	}
	if len(body.Projects) != 2 {
		t.Fatalf("project count = %d, want 2: %+v", len(body.Projects), body)
	}
	if body.Projects[0].Key != "areaflow" || body.Projects[1].Key != "areamatrix" {
		t.Fatalf("unexpected projects: %+v", body.Projects)
	}
}

func TestResourceCollectionEndpoints(t *testing.T) {
	created := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	projectA := project.Record{ID: 1, Key: "project-a", Name: "Project A", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	projectB := project.Record{ID: 2, Key: "project-b", Name: "Project B", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	store := fakeProjectStore{
		records:      []project.Record{projectA, projectB},
		recordsByKey: map[string]project.Record{projectA.Key: projectA, projectB.Key: projectB},
		versionsHook: func(record project.Record) []project.WorkflowVersion {
			return []project.WorkflowVersion{{ID: record.ID * 10, ProjectID: record.ID, DisplayLabel: "v1", LifecycleStatus: "active", StatusSummary: map[string]any{}, CreatedAt: created, UpdatedAt: created.Add(time.Duration(record.ID) * time.Minute)}}
		},
		runsHook: func(record project.Record, version project.WorkflowVersion, _ int) []project.RunRecord {
			return []project.RunRecord{{ID: record.ID * 100, ProjectID: record.ID, WorkflowVersionID: version.ID, RunType: "runner", RunKind: "execution", Status: "passed", Summary: map[string]any{}, Metadata: map[string]any{}, StartedAt: created.Add(time.Duration(record.ID) * time.Minute)}}
		},
		workersHook: func(record project.Record, _ int) []project.WorkerRecord {
			return []project.WorkerRecord{{ID: record.ID * 1000, ProjectID: record.ID, WorkerKey: record.Key + "-worker", WorkerType: "local", Status: "online", Capabilities: []string{"read_project"}, Metadata: map[string]any{}, RegisteredAt: created, UpdatedAt: created.Add(time.Duration(record.ID) * time.Minute)}}
		},
		artifactsHook: func(record project.Record, _ int) []project.ArtifactRecord {
			return []project.ArtifactRecord{{ID: record.ID * 10000, ProjectID: record.ID, WorkflowVersionID: record.ID * 10, ArtifactType: "evidence", StorageBackend: "local", URI: "artifact://" + record.Key, Metadata: map[string]any{}, CreatedAt: created.Add(time.Duration(record.ID) * time.Minute)}}
		},
	}
	handler := NewHandler(store)

	for _, endpoint := range []struct {
		path string
		key  string
	}{
		{path: "/api/v1/workflows", key: "workflows"},
		{path: "/api/v1/runs", key: "runs"},
		{path: "/api/v1/workers", key: "workers"},
		{path: "/api/v1/artifacts", key: "artifacts"},
	} {
		t.Run(endpoint.key, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, endpoint.path+"?project_key=project-a&limit=1", nil))
			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			items, ok := body[endpoint.key].([]any)
			if !ok || len(items) != 1 {
				t.Fatalf("%s items = %#v", endpoint.key, body[endpoint.key])
			}
			item := items[0].(map[string]any)
			projectBody := item["project"].(map[string]any)
			if projectBody["key"] != "project-a" {
				t.Fatalf("project key = %v, want project-a", projectBody["key"])
			}
		})
	}
}

func TestResourceCollectionEndpointRejectsInvalidLimit(t *testing.T) {
	handler := NewHandler(fakeProjectStore{})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/runs?limit=0", nil))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
}

type resourceCollectionTestStore struct {
	fakeProjectStore
	workflowPage project.WorkflowCollectionPage
	runPage      project.RunCollectionPage
	workerPage   project.WorkerCollectionPage
	artifactPage project.ArtifactCollectionPage
	workflowOpts project.WorkflowPageOptions
	runOpts      project.RunPageOptions
	workerOpts   project.ResourcePageOptions
	artifactOpts project.ArtifactPageOptions
	err          error
}

func (s *resourceCollectionTestStore) ListWorkflowCollection(_ context.Context, options project.WorkflowPageOptions) (project.WorkflowCollectionPage, error) {
	s.workflowOpts = options
	return s.workflowPage, s.err
}

func (s *resourceCollectionTestStore) ListRunCollection(_ context.Context, options project.RunPageOptions) (project.RunCollectionPage, error) {
	s.runOpts = options
	return s.runPage, s.err
}

func (s *resourceCollectionTestStore) ListWorkerCollection(_ context.Context, options project.ResourcePageOptions) (project.WorkerCollectionPage, error) {
	s.workerOpts = options
	return s.workerPage, s.err
}

func (s *resourceCollectionTestStore) ListArtifactCollection(_ context.Context, options project.ArtifactPageOptions) (project.ArtifactCollectionPage, error) {
	s.artifactOpts = options
	return s.artifactPage, s.err
}

func TestResourceCollectionsForwardFiltersCursorAndAssociations(t *testing.T) {
	created := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	record := project.Record{ID: 7, Key: "area", Name: "Area"}
	version := project.WorkflowVersion{ID: 8, ProjectID: 7, DisplayLabel: "v1", StatusSummary: map[string]any{}, CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 9, ProjectID: 7, WorkflowVersionID: 8, Status: "running", RunKind: "execution", Summary: map[string]any{}, Metadata: map[string]any{}, StartedAt: created}
	worker := project.WorkerRecord{ID: 10, ProjectID: 7, WorkerKey: "w", Status: "online", Capabilities: []string{}, Metadata: map[string]any{}, RegisteredAt: created, UpdatedAt: created}
	artifact := project.ArtifactRecord{ID: 11, ProjectID: 7, WorkflowVersionID: 8, RunID: 9, ArtifactType: "report", Metadata: map[string]any{}, CreatedAt: created}
	store := &resourceCollectionTestStore{
		fakeProjectStore: fakeProjectStore{record: record},
		workflowPage:     project.WorkflowCollectionPage{Items: []project.WorkflowCollectionItem{{Project: record, Workflow: version}}, NextCursor: "workflow-next"},
		runPage:          project.RunCollectionPage{Items: []project.RunCollectionItem{{Project: record, Workflow: version, Run: run}}, NextCursor: "run-next"},
		workerPage:       project.WorkerCollectionPage{Items: []project.WorkerCollectionItem{{Project: record, Worker: worker}}, NextCursor: "worker-next"},
		artifactPage:     project.ArtifactCollectionPage{Items: []project.ArtifactCollectionItem{{Project: record, Artifact: artifact}}, NextCursor: "artifact-next"},
	}
	handler := NewHandler(store)

	requestAPI[workflowCollectionResponse](t, handler, "/api/v1/workflows?project_key=area&status=active&kind=release&import_mode=authored&cursor=w&limit=25")
	if store.workflowOpts.ProjectKey != "area" || store.workflowOpts.Status != "active" || store.workflowOpts.Kind != "release" || store.workflowOpts.ImportMode != "authored" || store.workflowOpts.Cursor != "w" || store.workflowOpts.Limit != 25 {
		t.Fatalf("workflow options = %+v", store.workflowOpts)
	}
	dryRun := false
	runs := requestAPI[runCollectionResponse](t, handler, "/api/v1/runs?project_key=area&status=running&kind=execution&type=approved_artifact_write&dry_run=false&cursor=r&limit=25")
	if store.runOpts.DryRun == nil || *store.runOpts.DryRun != dryRun || len(runs.Runs) != 1 || runs.Runs[0].Run.ProjectID != 7 || runs.NextCursor != "run-next" {
		t.Fatalf("run response/options = %+v %+v", runs, store.runOpts)
	}
	requestAPI[workerCollectionResponse](t, handler, "/api/v1/workers?project_key=area&worker_key=local&status=online&type=local_host&capability=read_project&cursor=x")
	if store.workerOpts.Key != "local" || store.workerOpts.Kind != "read_project" || store.workerOpts.Type != "local_host" {
		t.Fatalf("worker options = %+v", store.workerOpts)
	}
	artifacts := requestAPI[artifactCollectionResponse](t, handler, "/api/v1/artifacts?project_key=area&type=report&storage_backend=local&sha256=abc&run_id=9&workflow_version_id=8&cursor=a")
	if store.artifactOpts.RunID != 9 || store.artifactOpts.WorkflowVersionID != 8 || len(artifacts.Artifacts) != 1 || artifacts.Artifacts[0].Artifact.ProjectID != 7 || artifacts.Artifacts[0].Artifact.RunID != 9 || artifacts.NextCursor != "artifact-next" {
		t.Fatalf("artifact response/options = %+v %+v", artifacts, store.artifactOpts)
	}
}

func TestResourceCollectionRejectsInvalidFiltersAndCursor(t *testing.T) {
	store := &resourceCollectionTestStore{fakeProjectStore: fakeProjectStore{}, err: project.ErrInvalidResourceCursor}
	handler := NewHandler(store)
	for _, path := range []string{"/api/v1/runs?dry_run=maybe", "/api/v1/artifacts?run_id=zero", "/api/v1/workflows?cursor=bad"} {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, path, nil))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d body=%s", path, resp.Code, resp.Body.String())
		}
	}
}

type auditEventCollectionTestStore struct {
	fakeProjectStore
	page    project.AuditEventPage
	options project.AuditEventPageOptions
	err     error
}

func (s *auditEventCollectionTestStore) ListAuditEventCollection(_ context.Context, options project.AuditEventPageOptions) (project.AuditEventPage, error) {
	s.options = options
	return s.page, s.err
}

func TestAuditEventCollectionForwardsFiltersAndCursor(t *testing.T) {
	created := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	store := &auditEventCollectionTestStore{
		fakeProjectStore: fakeProjectStore{record: project.Record{ID: 7, Key: "area"}},
		page:             project.AuditEventPage{Items: []project.AuditEventRecord{{ID: 9, ProjectID: 7, ActorID: 3, Action: "worker.register", Decision: "allowed", Metadata: map[string]any{}, CreatedAt: created}}, NextCursor: "next"},
	}
	response := requestAPI[auditEventsResponse](t, NewHandler(store), "/api/v1/audit-events?project_key=area&actor_id=3&action=worker.register&decision=allowed&resource_type=worker&resource=local&from=2026-07-01T00%3A00%3A00Z&to=2026-07-31T23%3A59%3A59Z&cursor=current&limit=25")
	if store.options.ProjectID != 7 || store.options.ActorID != 3 || store.options.Action != "worker.register" || store.options.Decision != "allowed" || store.options.ResourceType != "worker" || store.options.Resource != "local" || store.options.Cursor != "current" || store.options.Limit != 25 || store.options.From == nil || store.options.To == nil {
		t.Fatalf("audit options = %+v", store.options)
	}
	if response.Count != 1 || response.NextCursor != "next" || len(response.AuditEvents) != 1 {
		t.Fatalf("audit response = %+v", response)
	}
}

func TestAuditEventCollectionRejectsInvalidFilters(t *testing.T) {
	store := &auditEventCollectionTestStore{fakeProjectStore: fakeProjectStore{record: project.Record{ID: 7, Key: "area"}}, err: project.ErrInvalidResourceCursor}
	handler := NewHandler(store)
	for _, path := range []string{
		"/api/v1/audit-events?actor_id=none",
		"/api/v1/audit-events?from=yesterday",
		"/api/v1/audit-events?from=2026-07-14T00%3A00%3A00Z&to=2026-07-13T00%3A00%3A00Z",
		"/api/v1/audit-events?cursor=bad",
	} {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, path, nil))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d body=%s", path, resp.Code, resp.Body.String())
		}
	}
}

type resourceDetailTestStore struct {
	fakeProjectStore
	workerDetail project.WorkerDetail
	tasks        []project.RunTaskRecord
	attempts     []project.RunAttemptRecord
}

func (s resourceDetailTestStore) GetWorker(context.Context, int64, int64, int) (project.WorkerDetail, error) {
	return s.workerDetail, nil
}
func (s resourceDetailTestStore) ListRunTasks(context.Context, int64) ([]project.RunTaskRecord, error) {
	return s.tasks, nil
}
func (s resourceDetailTestStore) GetRunTask(_ context.Context, _ int64, taskID int64) (project.RunTaskRecord, error) {
	for _, item := range s.tasks {
		if item.ID == taskID {
			return item, nil
		}
	}
	return project.RunTaskRecord{}, project.ErrRunTaskNotFound
}
func (s resourceDetailTestStore) ListRunAttempts(context.Context, int64) ([]project.RunAttemptRecord, error) {
	return s.attempts, nil
}
func (s resourceDetailTestStore) GetRunAttempt(_ context.Context, _ int64, attemptID int64) (project.RunAttemptRecord, error) {
	for _, item := range s.attempts {
		if item.ID == attemptID {
			return item, nil
		}
	}
	return project.RunAttemptRecord{}, project.ErrRunAttemptNotFound
}

func TestWorkerDetailAndRunChildResourceEndpoints(t *testing.T) {
	created := time.Date(2026, 7, 13, 11, 0, 0, 0, time.UTC)
	record := project.Record{ID: 2, Key: "area"}
	run := project.RunRecord{ID: 3, ProjectID: 2, WorkflowVersionID: 4, Summary: map[string]any{}, Metadata: map[string]any{}, StartedAt: created}
	worker := project.WorkerRecord{ID: 5, ProjectID: 2, WorkerKey: "local", Capabilities: []string{}, Metadata: map[string]any{}, RegisteredAt: created, UpdatedAt: created}
	task := project.RunTaskRecord{ID: 6, ProjectID: 2, RunID: 3, Metadata: map[string]any{}, CreatedAt: created, UpdatedAt: created}
	attempt := project.RunAttemptRecord{ID: 7, ProjectID: 2, RunID: 3, RunTaskID: 6, Metadata: map[string]any{}, StartedAt: created}
	store := resourceDetailTestStore{
		fakeProjectStore: fakeProjectStore{record: record, runDetail: project.RunDetail{Run: run}},
		workerDetail:     project.WorkerDetail{Worker: worker, Heartbeats: []project.WorkerHeartbeatRecord{{ID: 8, ProjectID: 2, WorkerID: 5, Status: "online", ObservedAt: created, Metadata: map[string]any{}}}, Leases: []project.LeaseRecord{}},
		tasks:            []project.RunTaskRecord{task}, attempts: []project.RunAttemptRecord{attempt},
	}
	handler := NewHandler(store)
	workerBody := requestAPI[workerDetailResponse](t, handler, "/api/v1/workers/5?project_key=area")
	if workerBody.Worker.ID != 5 || len(workerBody.Heartbeats) != 1 || workerBody.Heartbeats[0].ProjectID != 2 {
		t.Fatalf("worker detail = %+v", workerBody)
	}
	taskList := requestAPI[struct {
		Tasks []runTaskResponse `json:"tasks"`
	}](t, handler, "/api/v1/runs/3/tasks?project_key=area")
	if len(taskList.Tasks) != 1 || taskList.Tasks[0].ProjectID != 2 {
		t.Fatalf("task list = %+v", taskList)
	}
	taskBody := requestAPI[runTaskResponse](t, handler, "/api/v1/runs/3/tasks/6?project_key=area")
	if taskBody.ID != 6 {
		t.Fatalf("task detail = %+v", taskBody)
	}
	attemptList := requestAPI[struct {
		Attempts []runAttemptResponse `json:"attempts"`
	}](t, handler, "/api/v1/runs/3/attempts?project_key=area")
	if len(attemptList.Attempts) != 1 || attemptList.Attempts[0].ProjectID != 2 {
		t.Fatalf("attempt list = %+v", attemptList)
	}
	attemptBody := requestAPI[runAttemptResponse](t, handler, "/api/v1/runs/3/attempts/7?project_key=area")
	if attemptBody.ID != 7 {
		t.Fatalf("attempt detail = %+v", attemptBody)
	}
}

func TestProjectDetailEndpoint(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/areamatrix",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project detail status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode project detail: %v", err)
	}
	if body.Key != "areamatrix" || body.Adapter != "areamatrix" || body.Root != "/tmp/areamatrix" {
		t.Fatalf("unexpected project detail: %+v", body)
	}
}

func TestProjectScopedAPIsUseRouteProjectKey(t *testing.T) {
	created := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	projectA := project.Record{ID: 101, Key: "scope-a", Name: "Scope A", Kind: "fixture", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	projectB := project.Record{ID: 202, Key: "scope-b", Name: "Scope B", Kind: "fixture", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	records := map[string]project.Record{
		projectA.Key: projectA,
		projectB.Key: projectB,
	}
	versions := map[string]project.WorkflowVersion{
		projectA.Key: {
			ID:              1001,
			ProjectID:       projectA.ID,
			DisplayLabel:    "shared-v1",
			VersionKind:     "workflow_version",
			LifecycleStatus: "authoring",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"project_key": projectA.Key},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		projectB.Key: {
			ID:              2001,
			ProjectID:       projectB.ID,
			DisplayLabel:    "shared-v1",
			VersionKind:     "workflow_version",
			LifecycleStatus: "authoring",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"project_key": projectB.Key},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
	}
	seenKeys := []string{}
	handler := NewHandler(fakeProjectStore{
		recordsByKey: records,
		getByKeyHook: func(key string) {
			seenKeys = append(seenKeys, key)
		},
		summaryHook: func(record project.Record) project.ProjectSummary {
			return project.ProjectSummary{
				Project: record,
				Inventory: project.ImportInventory{
					Versions: record.ID,
				},
			}
		},
		eventsHook: func(projectID int64, _ int) []project.EventRecord {
			return []project.EventRecord{{
				ID:        projectID + 10,
				ProjectID: projectID,
				Type:      "api.scope.event",
				Severity:  "info",
				Message:   fmt.Sprintf("event for %d", projectID),
				Metadata:  map[string]any{"project_id": projectID},
				CreatedAt: created,
			}}
		},
		auditEventsHook: func(projectID int64, _ int) []project.AuditEventRecord {
			return []project.AuditEventRecord{{
				ID:           projectID + 20,
				ProjectID:    projectID,
				Action:       "api.scope.audit",
				Capability:   "read_project",
				ResourceType: "project",
				Resource:     fmt.Sprintf("%d", projectID),
				Decision:     "allowed",
				Metadata:     map[string]any{"project_id": projectID},
				CreatedAt:    created,
			}}
		},
		versionsHook: func(record project.Record) []project.WorkflowVersion {
			return []project.WorkflowVersion{versions[record.Key]}
		},
		getWorkflowVersionHook: func(record project.Record, label string) (project.WorkflowVersion, error) {
			if label != "shared-v1" {
				return project.WorkflowVersion{}, project.ErrWorkflowVersionNotFound
			}
			return versions[record.Key], nil
		},
		workersHook: func(record project.Record, _ int) []project.WorkerRecord {
			return []project.WorkerRecord{{
				ID:                       record.ID + 30,
				ProjectID:                record.ID,
				WorkerKey:                "worker-" + record.Key,
				WorkerType:               "local_host",
				Status:                   "online",
				Capabilities:             []string{"read_project"},
				Metadata:                 map[string]any{"project_key": record.Key},
				RegisteredAt:             created,
				HeartbeatIntervalSeconds: 30,
				LeaseTimeoutSeconds:      300,
				UpdatedAt:                created,
			}}
		},
		runsHook: func(record project.Record, version project.WorkflowVersion, _ int) []project.RunRecord {
			if version.ProjectID != record.ID {
				t.Fatalf("workflow version project leak: version project %d for route project %d", version.ProjectID, record.ID)
			}
			return []project.RunRecord{{
				ID:                record.ID + 40,
				ProjectID:         record.ID,
				WorkflowVersionID: version.ID,
				RunType:           "execution",
				RunKind:           "api_scope",
				Status:            "queued",
				RiskLevel:         "low",
				RiskPolicy:        "pause",
				DryRun:            true,
				Summary:           map[string]any{"project_key": record.Key},
				Metadata:          map[string]any{"project_key": record.Key},
				StartedAt:         created,
			}}
		},
		artifactsHook: func(record project.Record, _ int) []project.ArtifactRecord {
			return []project.ArtifactRecord{scopedAPIArtifact(record, versions[record.Key], created, "project")}
		},
		versionArtifactsHook: func(record project.Record, version project.WorkflowVersion, _ int) []project.ArtifactRecord {
			if version.ProjectID != record.ID {
				t.Fatalf("workflow artifact version project leak: version project %d for route project %d", version.ProjectID, record.ID)
			}
			return []project.ArtifactRecord{scopedAPIArtifact(record, version, created, "version")}
		},
	})

	for _, record := range []project.Record{projectA, projectB} {
		summary := requestAPI[projectSummaryResponse](t, handler, "/api/v1/projects/"+record.Key+"/summary")
		if summary.Project.Key != record.Key || summary.Inventory.Versions != record.ID {
			t.Fatalf("summary leaked project scope for %s: %+v", record.Key, summary)
		}

		events := requestAPI[projectEventsResponse](t, handler, "/api/v1/projects/"+record.Key+"/events")
		if events.Project != record.Key || len(events.Events) != 1 || events.Events[0].ProjectID != record.ID {
			t.Fatalf("events leaked project scope for %s: %+v", record.Key, events)
		}

		audit := requestAPI[auditEventsResponse](t, handler, "/api/v1/audit-events?project_key="+record.Key)
		if audit.ProjectKey != record.Key || len(audit.AuditEvents) != 1 || audit.AuditEvents[0].ProjectID != record.ID {
			t.Fatalf("audit events leaked project scope for %s: %+v", record.Key, audit)
		}

		workers := requestAPI[workerListResponse](t, handler, "/api/v1/projects/"+record.Key+"/workers")
		if workers.Project.Key != record.Key || len(workers.Workers) != 1 || workers.Workers[0].ProjectID != record.ID {
			t.Fatalf("workers leaked project scope for %s: %+v", record.Key, workers)
		}

		versionList := requestAPI[workflowVersionListResponse](t, handler, "/api/v1/projects/"+record.Key+"/workflow-versions")
		if versionList.Project.Key != record.Key || len(versionList.WorkflowVersions) != 1 || versionList.WorkflowVersions[0].ID != versions[record.Key].ID {
			t.Fatalf("workflow versions leaked project scope for %s: %+v", record.Key, versionList)
		}

		runs := requestAPI[workflowVersionRunsResponse](t, handler, "/api/v1/projects/"+record.Key+"/workflow-versions/shared-v1/runs")
		if runs.Project.Key != record.Key || runs.WorkflowVersion.ID != versions[record.Key].ID || len(runs.Runs) != 1 || runs.Runs[0].ID != record.ID+40 {
			t.Fatalf("runs leaked project scope for %s: %+v", record.Key, runs)
		}

		projectArtifacts := requestAPI[projectArtifactsResponse](t, handler, "/api/v1/projects/"+record.Key+"/artifacts")
		if projectArtifacts.Project.Key != record.Key || len(projectArtifacts.Artifacts) != 1 || !strings.Contains(projectArtifacts.Artifacts[0].URI, record.Key) {
			t.Fatalf("project artifacts leaked project scope for %s: %+v", record.Key, projectArtifacts)
		}

		versionArtifacts := requestAPI[workflowVersionArtifactsResponse](t, handler, "/api/v1/projects/"+record.Key+"/workflow-versions/shared-v1/artifacts")
		if versionArtifacts.Project.Key != record.Key || versionArtifacts.WorkflowVersion.ID != versions[record.Key].ID || len(versionArtifacts.Artifacts) != 1 || !strings.Contains(versionArtifacts.Artifacts[0].URI, record.Key) {
			t.Fatalf("workflow version artifacts leaked project scope for %s: %+v", record.Key, versionArtifacts)
		}
	}

	for _, want := range []string{projectA.Key, projectB.Key} {
		if !stringSliceContains(seenKeys, want) {
			t.Fatalf("route project key %q was not resolved through GetByKey; seen=%v", want, seenKeys)
		}
	}
}

func scopedAPIArtifact(record project.Record, version project.WorkflowVersion, created time.Time, scope string) project.ArtifactRecord {
	return project.ArtifactRecord{
		ID:                record.ID + 50,
		ProjectID:         record.ID,
		WorkflowVersionID: version.ID,
		ArtifactType:      "api_scope_" + scope,
		StorageBackend:    "project_reference",
		URI:               "project://" + record.Key + "/" + scope + ".json",
		SourcePath:        "workflow/" + record.Key + "/" + scope + ".json",
		SHA256:            "sha-" + record.Key + "-" + scope,
		SizeBytes:         12,
		ContentType:       "application/json",
		Metadata:          map[string]any{"project_key": record.Key, "scope": scope},
		CreatedAt:         created,
	}
}

func requestAPI[T any](t *testing.T, handler http.Handler, path string) T {
	t.Helper()
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, path, nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d body=%s", path, resp.Code, resp.Body.String())
	}
	var body T
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode GET %s response: %v", path, err)
	}
	return body
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestProjectsEndpointMethodAndNotFound(t *testing.T) {
	handler := NewHandler(fakeProjectStore{err: errors.New("not found")})

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/projects", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("project list method status = %d body=%s", methodResp.Code, methodResp.Body.String())
	}

	notFoundResp := httptest.NewRecorder()
	handler.ServeHTTP(notFoundResp, httptest.NewRequest(http.MethodGet, "/api/projects/missing", nil))
	if notFoundResp.Code != http.StatusNotFound {
		t.Fatalf("project detail not found status = %d body=%s", notFoundResp.Code, notFoundResp.Body.String())
	}
}

func TestProjectWorkerEndpoints(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 0, 0, 0, time.UTC)
	heartbeat := created.Add(time.Second)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	worker := project.WorkerRecord{
		ID:                       7,
		ProjectID:                1,
		ActorID:                  8,
		WorkerKey:                "local-1",
		WorkerType:               "local_host",
		Status:                   "online",
		Hostname:                 "dev-host",
		PID:                      123,
		Capabilities:             []string{"read_project", "write_artifacts"},
		Metadata:                 map[string]any{"mode": "v0.6a"},
		RegisteredAt:             created,
		LastHeartbeatAt:          &heartbeat,
		HeartbeatIntervalSeconds: 30,
		LeaseTimeoutSeconds:      300,
		UpdatedAt:                heartbeat,
	}
	var registerOptions project.RegisterWorkerOptions
	var heartbeatWorkerKey string
	var heartbeatOptions project.WorkerHeartbeatOptions
	handler := NewHandler(fakeProjectStore{
		record:  record,
		worker:  worker,
		workers: []project.WorkerRecord{worker},
		workerRegisterHook: func(options project.RegisterWorkerOptions) {
			registerOptions = options
		},
		workerHeartbeatHook: func(workerKey string, options project.WorkerHeartbeatOptions) {
			heartbeatWorkerKey = workerKey
			heartbeatOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers", strings.NewReader(`{
		"worker_key": "local-1",
		"worker_type": "local_host",
		"capabilities": ["read_project", "write_artifacts"],
		"idempotency_key": "worker-register-local-1"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("register status = %d body=%s", resp.Code, resp.Body.String())
	}
	var registered workerResponse
	if err := json.NewDecoder(resp.Body).Decode(&registered); err != nil {
		t.Fatalf("decode worker register response: %v", err)
	}
	if registered.WorkerKey != "local-1" || registered.WorkerType != "local_host" || len(registered.Capabilities) != 2 {
		t.Fatalf("unexpected worker register response: %+v", registered)
	}
	if registerOptions.IdempotencyKey != "worker-register-local-1" {
		t.Fatalf("worker register idempotency key = %q", registerOptions.IdempotencyKey)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/local-1/heartbeat", strings.NewReader(`{
		"status": "online",
		"idempotency_key": "worker-heartbeat-local-1"
	}`))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("heartbeat status = %d body=%s", resp.Code, resp.Body.String())
	}
	var heartbeatBody workerResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatBody); err != nil {
		t.Fatalf("decode worker heartbeat response: %v", err)
	}
	if heartbeatBody.LastHeartbeatAt == "" || heartbeatBody.Status != "online" {
		t.Fatalf("unexpected worker heartbeat response: %+v", heartbeatBody)
	}
	if heartbeatWorkerKey != "local-1" || heartbeatOptions.IdempotencyKey != "worker-heartbeat-local-1" {
		t.Fatalf("worker heartbeat passthrough = key %q options %+v", heartbeatWorkerKey, heartbeatOptions)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workers?limit=1", nil)
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", resp.Code, resp.Body.String())
	}
	var list workerListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode worker list response: %v", err)
	}
	if list.Project.Key != "areamatrix" || len(list.Workers) != 1 {
		t.Fatalf("unexpected worker list response: %+v", list)
	}
}

func TestProjectWorkerLifecycleIdempotencyConflict(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		record:    project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		workerErr: project.ErrIdempotencyConflict,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workers", strings.NewReader(`{
		"worker_key": "local-1",
		"idempotency_key": "same-key"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("worker register status = %d body=%s, want 409", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workers/local-1/heartbeat", strings.NewReader(`{
		"status": "online",
		"idempotency_key": "same-key"
	}`))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("worker heartbeat status = %d body=%s, want 409", resp.Code, resp.Body.String())
	}
}

func TestLocalServiceStatusEndpoint(t *testing.T) {
	generated := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		service: project.LocalServiceStatus{
			Status: "ready",
			Mode:   "local_service",
			API: project.LocalServiceComponentStatus{
				Status:  "ready",
				Message: "AreaFlow API is available",
			},
			Database: project.LocalServiceComponentStatus{
				Status:  "ready",
				Message: "PostgreSQL connection is healthy",
			},
			WorkerPool: project.LocalServiceWorkerPoolStatus{
				Status:             "warn",
				Message:            "worker pool has recovery items",
				TotalProjects:      1,
				TotalWorkers:       2,
				TotalOnlineWorkers: 1,
				TotalActiveLeases:  0,
				TotalQueuedTasks:   3,
				TotalNeedsRecovery: 1,
			},
			Dashboard: project.LocalServiceDashboardStatus{
				URL:     "http://127.0.0.1:5174",
				APIURL:  "http://127.0.0.1:3847/api/v1",
				Status:  "ready",
				Message: "dashboard should use AreaFlow API as source of truth",
			},
			Capabilities:     []string{"observe_api", "open_web_dashboard"},
			ForbiddenActions: []string{"maintain_second_database", "run_workflow_directly"},
			GeneratedAt:      generated,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/service/status", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("service status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body localServiceStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode service status response: %v", err)
	}
	if body.Status != "ready" || body.Mode != "local_service" {
		t.Fatalf("unexpected service status: %+v", body)
	}
	if body.Database.Status != "ready" || body.WorkerPool.Status != "warn" {
		t.Fatalf("unexpected component status: %+v", body)
	}
	if body.WorkerPool.TotalQueuedTasks != 3 || body.WorkerPool.TotalNeedsRecovery != 1 {
		t.Fatalf("unexpected worker pool status: %+v", body.WorkerPool)
	}
	if body.Dashboard.URL != "http://127.0.0.1:5174" || body.Dashboard.APIURL != "http://127.0.0.1:3847/api/v1" {
		t.Fatalf("unexpected dashboard status: %+v", body.Dashboard)
	}
	if len(body.ForbiddenActions) != 2 || body.GeneratedAt == "" {
		t.Fatalf("unexpected service guardrails: %+v", body)
	}
}

func TestWebWriteActionGateEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		webGate: project.BuildWebWriteActionGate(project.WebWriteActionGateOptions{GeneratedAt: generated}),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/web/write-action-gate", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("web write action gate = %d body=%s", resp.Code, resp.Body.String())
	}
	var body webWriteActionGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode web write action gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_web_write_action_gate" {
		t.Fatalf("unexpected web write gate: %+v", body)
	}
	if len(body.Actions) < 6 {
		t.Fatalf("expected write action matrix, got %+v", body.Actions)
	}
	if body.DBWriteAttempted || body.ProjectWriteAttempted || body.CommandCreated || body.AuditEventWritten {
		t.Fatalf("web write gate should be read-only: %+v", body)
	}
	if !containsString(body.ForbiddenActions, "enable_write_buttons_by_default") {
		t.Fatalf("missing write button guardrail: %+v", body.ForbiddenActions)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/web/write-action-gate", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("web write gate POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestDesktopServiceControlGateEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		desktopGate: project.BuildDesktopServiceControlGate(project.DesktopServiceControlGateOptions{GeneratedAt: generated}),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/service-control-gate", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("desktop service control gate = %d body=%s", resp.Code, resp.Body.String())
	}
	var body desktopServiceControlGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode desktop service control gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_desktop_service_control_gate" {
		t.Fatalf("unexpected desktop control gate: %+v", body)
	}
	if len(body.Actions) < 5 {
		t.Fatalf("expected desktop service action matrix, got %+v", body.Actions)
	}
	if body.DBWriteAttempted || body.ProjectWriteAttempted || body.ProcessControlAttempted ||
		body.CommandCreated || body.ApprovalCreated || body.AuditEventWritten ||
		body.WorkerScheduled || body.WorkflowExecutionStarted || body.SecretsResolved || body.NetworkUsed {
		t.Fatalf("desktop service control gate should be read-only: %+v", body)
	}
	if !containsString(body.ForbiddenActions, "start_service_without_gate") ||
		!containsString(body.ForbiddenActions, "schedule_worker_from_desktop") {
		t.Fatalf("missing desktop guardrails: %+v", body.ForbiddenActions)
	}

	seenDashboard := false
	seenRestart := false
	for _, action := range body.Actions {
		switch action.Key {
		case "open_dashboard":
			seenDashboard = true
			if action.Status != "ready" || action.DefaultUIState != "enabled_link" {
				t.Fatalf("dashboard action should be launcher-only: %+v", action)
			}
		case "restart_service":
			seenRestart = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("restart action should stay blocked: %+v", action)
			}
			if !containsString(action.Blockers, "restart_recovery_contract_not_defined") {
				t.Fatalf("restart blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenDashboard || !seenRestart {
		t.Fatalf("missing dashboard or restart action: %+v", body.Actions)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/desktop/service-control-gate", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("desktop service control gate POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestDesktopNotificationGateEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		notificationGate: project.BuildDesktopNotificationGate(project.DesktopNotificationGateOptions{GeneratedAt: generated}),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/notification-gate", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("desktop notification gate = %d body=%s", resp.Code, resp.Body.String())
	}
	var body desktopNotificationGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode desktop notification gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_desktop_notification_gate" {
		t.Fatalf("unexpected desktop notification gate: %+v", body)
	}
	if len(body.Actions) < 5 {
		t.Fatalf("expected notification action matrix, got %+v", body.Actions)
	}
	if body.DBWriteAttempted || body.ProjectWriteAttempted || body.EventStreamOpened ||
		body.NotificationRequested || body.CommandCreated || body.ApprovalCreated ||
		body.AuditEventWritten || body.WorkerScheduled || body.WorkflowExecutionStarted ||
		body.SecretsResolved || body.NetworkUsed {
		t.Fatalf("desktop notification gate should be read-only: %+v", body)
	}
	if !containsString(body.ForbiddenActions, "request_os_notification_permission_without_gate") ||
		!containsString(body.ForbiddenActions, "schedule_worker_from_notification") {
		t.Fatalf("missing notification guardrails: %+v", body.ForbiddenActions)
	}

	seenStream := false
	seenSystemNotification := false
	for _, action := range body.Actions {
		switch action.Key {
		case "observe_event_stream":
			seenStream = true
			if action.Status != "ready" || action.DefaultUIState != "available_read_only" {
				t.Fatalf("event stream action should be read-only available: %+v", action)
			}
		case "enable_system_notifications":
			seenSystemNotification = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("system notification action should stay blocked: %+v", action)
			}
			if !containsString(action.Blockers, "notification_permission_flow_not_implemented") {
				t.Fatalf("notification blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenStream || !seenSystemNotification {
		t.Fatalf("missing event stream or notification action: %+v", body.Actions)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/desktop/notification-gate", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("desktop notification gate POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestDesktopTrayMenuGateEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		trayMenuGate: project.BuildDesktopTrayMenuGate(project.DesktopTrayMenuGateOptions{GeneratedAt: generated}),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/desktop/tray-menu-gate", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("desktop tray menu gate = %d body=%s", resp.Code, resp.Body.String())
	}
	var body desktopTrayMenuGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode desktop tray menu gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_desktop_tray_menu_gate" {
		t.Fatalf("unexpected desktop tray menu gate: %+v", body)
	}
	if len(body.Actions) < 7 {
		t.Fatalf("expected tray menu action matrix, got %+v", body.Actions)
	}
	if body.DBWriteAttempted || body.ProjectWriteAttempted || body.TrayMenuCreated ||
		body.OSIntegrationRequested || body.CommandCreated || body.ApprovalCreated ||
		body.AuditEventWritten || body.ServiceControlAttempted || body.NotificationRequested ||
		body.WorkerScheduled || body.WorkflowExecutionStarted || body.SecretsResolved || body.NetworkUsed {
		t.Fatalf("desktop tray menu gate should be read-only: %+v", body)
	}
	if !containsString(body.ForbiddenActions, "create_tray_menu_from_gate") ||
		!containsString(body.ForbiddenActions, "schedule_worker_from_tray") {
		t.Fatalf("missing tray menu guardrails: %+v", body.ForbiddenActions)
	}

	seenDashboard := false
	seenStopService := false
	for _, action := range body.Actions {
		switch action.Key {
		case "open_dashboard":
			seenDashboard = true
			if action.Status != "ready" || action.DefaultUIState != "enabled_link" {
				t.Fatalf("dashboard tray action should be launcher-only: %+v", action)
			}
		case "stop_service":
			seenStopService = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("stop service tray action should stay blocked: %+v", action)
			}
			if !containsString(action.Blockers, "service_control_gate_blocked") {
				t.Fatalf("stop service blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenDashboard || !seenStopService {
		t.Fatalf("missing dashboard or stop service tray action: %+v", body.Actions)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/desktop/tray-menu-gate", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("desktop tray menu gate POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestSecurityBoundaryReadinessEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		security: project.BuildSecurityBoundaryReadiness(project.SecurityBoundaryReadinessOptions{GeneratedAt: generated}),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/security/boundary-readiness", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("security boundary readiness = %d body=%s", resp.Code, resp.Body.String())
	}
	var body securityBoundaryReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode security boundary readiness: %v", err)
	}
	if body.Status != "ready" || body.Mode != "read_only_security_boundary_readiness" {
		t.Fatalf("unexpected security boundary readiness: %+v", body)
	}
	if body.SecretResolveOpen || body.RemoteWorkerCredentialsOpen || body.AuthorizationChanged ||
		body.APITokenIssuanceOpen || body.TeamPermissionEnforcementOpen || body.ExternalAPICallOpen {
		t.Fatalf("security boundary readiness opened forbidden capability: %+v", body)
	}
	if !containsString(body.ForbiddenActions, "resolve_secret_plaintext") ||
		!containsString(body.ForbiddenActions, "issue_remote_worker_credential") ||
		!containsString(body.ForbiddenActions, "change_api_authorization") {
		t.Fatalf("missing forbidden security actions: %+v", body.ForbiddenActions)
	}
	seenSecretResolve := false
	seenTokenLifecycle := false
	for _, item := range body.Items {
		switch item.Key {
		case "secret_resolve":
			seenSecretResolve = true
			if item.Status != "ready" {
				t.Fatalf("secret resolve item should stay readiness-only: %+v", item)
			}
		case "api_token_lifecycle":
			seenTokenLifecycle = true
			if item.Status != "ready" {
				t.Fatalf("api token item should stay readiness-only: %+v", item)
			}
		}
	}
	if !seenSecretResolve || !seenTokenLifecycle {
		t.Fatalf("missing security readiness items: %+v", body.Items)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/security/boundary-readiness", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("security boundary readiness POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func assertReal100GuardrailResponse(t *testing.T, status, scope string, blockers []string, wantScope string, wantBlockers []string) {
	t.Helper()
	if status != project.Real100StatusBlocked || scope != wantScope || !reflect.DeepEqual(blockers, wantBlockers) {
		t.Fatalf("unexpected real 100 guardrail: status=%q scope=%q blockers=%v", status, scope, blockers)
	}
}

func assertReal100DisambiguationResponse(t *testing.T, response any, wantScope string, wantReleaseCandidateDecision string) {
	t.Helper()
	value := reflect.Indirect(reflect.ValueOf(response))
	if !value.IsValid() {
		t.Fatal("real 100 disambiguation response is invalid")
	}
	claimScope := value.FieldByName("ClaimScope").String()
	notReal100 := value.FieldByName("NotReal100").Bool()
	evidenceOnly := value.FieldByName("EvidenceOnly").Bool()
	statusAlone := value.FieldByName("StatusAloneIsNotCompletion").Bool()
	releaseCandidateDecision := value.FieldByName("ReleaseCandidateDecision").String()
	if claimScope != wantScope || !notReal100 || !evidenceOnly || !statusAlone ||
		releaseCandidateDecision != wantReleaseCandidateDecision {
		t.Fatalf("unexpected real 100 disambiguation: claim_scope=%q not_real_100=%t evidence_only=%t status_alone=%t release_candidate_decision=%q",
			claimScope,
			notReal100,
			evidenceOnly,
			statusAlone,
			releaseCandidateDecision,
		)
	}
}

func assertReal100BreakdownHasKey(t *testing.T, items []project.Real100BreakdownItem, key string) {
	t.Helper()
	for _, item := range items {
		if item.Key == key {
			return
		}
	}
	t.Fatalf("missing real 100 breakdown key %q in %+v", key, items)
}

func TestCompletionAuditEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	audit := project.BuildCompletionAudit(project.CompletionAuditOptions{GeneratedAt: generated}, project.CompletionAuditParts{
		ReleaseFinalGate:          &project.ReleaseFinalGate{Status: "blocked", Mode: "read_only_release_final_gate"},
		SecurityBoundaryReadiness: ptrSecurityReadiness(project.BuildSecurityBoundaryReadiness(project.SecurityBoundaryReadinessOptions{GeneratedAt: generated})),
		LocalServiceStatus:        &project.LocalServiceStatus{Status: "ready", Mode: "local_service"},
	})
	handler := NewHandler(fakeProjectStore{
		completion: audit,
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/completion-audit", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("completion audit = %d body=%s", resp.Code, resp.Body.String())
	}
	var body completionAuditResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode completion audit: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_completion_audit" || body.Scope != "v1.0" {
		t.Fatalf("unexpected completion audit: %+v", body)
	}
	if body.ReleaseFinalGateStatus != "incomplete" || body.AreaMatrixDogfoodStatus != "incomplete" ||
		body.ProtectedPathProofStatus != "blocked" {
		t.Fatalf("unexpected completion aggregate statuses: %+v", body)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.CompletionAuditReadinessScope,
		[]string{
			"real_areamatrix_read_only_shim_not_landed",
			"real_areamatrix_execution_cutover_not_proven",
			"real_areamatrix_archive_not_proven",
			"real_areamatrix_shim_retirement_not_proven",
			"release_candidate_snapshot_not_ready",
			"package_a_status_projection_not_applied",
		},
	)
	assertReal100DisambiguationResponse(t, body, project.CompletionAuditReadinessScope, "requires_release_candidate_snapshot")
	assertReal100BreakdownHasKey(t, body.Real100Breakdown.NeedsExactAuthorization, "package_a_exact_authorization")
	assertReal100BreakdownHasKey(t, body.Real100Breakdown.NeedsRealAreaMatrixWrite, "package_a_status_projection_apply")
	assertReal100BreakdownHasKey(t, body.Real100Breakdown.AreaFlowOnlyCanContinue, "E1_design_source_alignment")
	if !body.SafetyFacts["read_only"] || body.SafetyFacts["release_package_created"] ||
		body.SafetyFacts["publish_attempted"] || body.SafetyFacts["restore_apply_attempted"] ||
		body.SafetyFacts["secret_resolved"] || body.SafetyFacts["remote_worker_credentials_issued"] ||
		body.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("unexpected completion safety facts: %+v", body.SafetyFacts)
	}
	if !containsString(body.ForbiddenActions, "run_smoke") ||
		!containsString(body.ForbiddenActions, "touch_areamatrix_protected_paths") {
		t.Fatalf("missing completion forbidden actions: %+v", body.ForbiddenActions)
	}
	seenDogfood := false
	seenProtectedPaths := false
	for _, item := range body.Items {
		switch item.Key {
		case "E4_areamatrix_dogfood_completion":
			seenDogfood = true
			if item.Status != "blocked" {
				t.Fatalf("dogfood item should be blocked: %+v", item)
			}
		case "E9_areamatrix_protected_path_proof":
			seenProtectedPaths = true
			if item.Status != "blocked" {
				t.Fatalf("protected path item should be blocked: %+v", item)
			}
		}
	}
	if !seenDogfood || !seenProtectedPaths {
		t.Fatalf("missing completion audit items: %+v", body.Items)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/completion-audit", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("completion audit POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestCompletionAuditSnapshotReadinessEndpoint(t *testing.T) {
	record := project.Record{ID: 7, Key: "areamatrix", Name: "AreaMatrix"}
	readiness := project.CompletionAuditSnapshotReadiness{
		Project:       record,
		Status:        "blocked",
		Message:       "release candidate completion audit snapshot is not ready",
		HasSnapshot:   true,
		RequiredClass: "release_candidate",
		BundleHash:    "bundle-hash-1",
		Latest: project.CompletionAuditSnapshot{
			Project:               record,
			Status:                "recorded",
			AuditStatus:           "complete",
			AuditScope:            "v1.0",
			AuditHash:             "fixture-hash",
			ReleaseCandidateLabel: "v1.0-fixture",
			EvidenceClass:         "fixture",
			EvidenceURI:           "scripts/smoke-completion-audit-full-proof.sh",
			EventID:               11,
			Metadata:              map[string]any{"fixture_snapshot": true, "release_candidate_snapshot": false},
		},
		Items: []project.ReadinessItem{
			{
				Key:     "completion_audit_snapshot_fixture_only",
				Status:  "blocked",
				Message: "Latest completion audit snapshot is fixture evidence, not release_candidate evidence",
				Metadata: map[string]any{
					"fixture_snapshot":           true,
					"release_candidate_snapshot": false,
				},
			},
		},
		SafetyFacts: map[string]bool{"read_only": true, "project_write_attempted": false},
	}
	handler := NewHandler(fakeProjectStore{
		record:             record,
		completionSnapshot: readiness,
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/completion-audit/snapshot-readiness", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("completion audit snapshot readiness = %d body=%s", resp.Code, resp.Body.String())
	}
	var body completionAuditSnapshotReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode completion audit snapshot readiness: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || !body.HasSnapshot ||
		body.RequiredClass != "release_candidate" || body.BundleHash != "bundle-hash-1" ||
		body.Latest.EvidenceClass != "fixture" {
		t.Fatalf("unexpected completion audit snapshot readiness: %+v", body)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.CompletionAuditReadinessScope,
		project.Real100CompletionAuditBlockers(),
	)
	assertReal100DisambiguationResponse(t, body, project.CompletionAuditReadinessScope, "requires_release_candidate_snapshot")
	assertReal100GuardrailResponse(
		t,
		body.Latest.Real100Status,
		body.Latest.ReadinessScope,
		body.Latest.Real100Blockers,
		project.CompletionAuditReadinessScope,
		project.Real100CompletionAuditBlockers(),
	)
	assertReal100DisambiguationResponse(t, body.Latest, project.CompletionAuditReadinessScope, "requires_release_candidate_snapshot")
	if len(body.Items) != 1 || body.Items[0].Key != "completion_audit_snapshot_fixture_only" ||
		body.Items[0].Metadata["release_candidate_snapshot"] != false {
		t.Fatalf("missing fixture-only blocker: %+v", body.Items)
	}
	if len(body.Gaps) != 1 || body.Gaps[0].Key != "completion_audit_snapshot_fixture_only" ||
		body.Gaps[0].Category != "snapshot" ||
		!containsString(body.Gaps[0].Blockers, "completion_audit_snapshot_fixture_only") {
		t.Fatalf("missing fixture-only gap: %+v", body.Gaps)
	}
	if body.Closure.Ready ||
		body.Closure.ReadyForReleaseCandidateClosure ||
		body.Closure.SnapshotStatus != "fixture_only" ||
		body.Closure.Snapshot.Status != "fixture_only" ||
		body.Closure.Snapshot.Ready ||
		body.Closure.RequiredClass != "release_candidate" ||
		body.Closure.RequiredEvidenceClass != "release_candidate" ||
		body.Closure.GapCount != 1 ||
		!containsString(body.Closure.Blockers, "completion_audit_snapshot_fixture_only") {
		t.Fatalf("missing fixture-only closure: %+v", body.Closure)
	}
	if !body.SafetyFacts["read_only"] || body.SafetyFacts["project_write_attempted"] {
		t.Fatalf("unexpected safety facts: %+v", body.SafetyFacts)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/completion-audit/snapshot-readiness", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("completion audit snapshot readiness POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestSupportBundlePreviewEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 3, 13, 0, 0, 0, time.UTC)
	preview := project.BuildSupportBundlePreview(project.BackupManifest{
		Status:       "ready",
		ManifestHash: "backup-hash",
		Projects: []project.BackupProjectManifest{
			{Project: project.Record{Key: "areamatrix", Name: "AreaMatrix", RootPath: "/tmp/areamatrix"}},
		},
	}, project.AuditCoverage{Status: "warn", TotalAuditEvents: 4}, project.SupportBundlePreviewOptions{GeneratedAt: generated})
	handler := NewHandler(fakeProjectStore{supportBundle: preview})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/ops/support-bundle-preview", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("support bundle preview = %d body=%s", resp.Code, resp.Body.String())
	}
	var body supportBundlePreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode support bundle preview: %v", err)
	}
	if body.Status != "ready" || body.Mode != "metadata_only_support_bundle_preview" || body.BundleID == "" {
		t.Fatalf("unexpected support bundle preview: %+v", body)
	}
	if len(body.Projects) != 1 || body.Projects[0].Key != "areamatrix" || len(body.PathReferences) != 1 {
		t.Fatalf("unexpected support bundle project references: %+v", body)
	}
	if !body.SafetyFacts["read_only"] || !body.SafetyFacts["metadata_only"] ||
		body.SafetyFacts["export_open"] || body.SafetyFacts["secret_values_included"] ||
		body.SafetyFacts["raw_artifact_contents_included"] || body.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("support bundle preview opened unsafe fact: %+v", body.SafetyFacts)
	}
	if !containsString(body.ForbiddenActions, "export_support_bundle") ||
		!containsString(body.ForbiddenActions, "read_secret_values") {
		t.Fatalf("missing support bundle forbidden actions: %+v", body.ForbiddenActions)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/ops/support-bundle-preview", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("support bundle preview POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestMigrationLedgerReadinessEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 3, 13, 30, 0, 0, time.UTC)
	readiness := project.MigrationLedgerReadiness{
		Status:                               "needs_attention",
		Mode:                                 "read_only_migration_ledger_readiness",
		Entries:                              []project.MigrationLedgerEntry{{Name: "000001_v0_1_core.sql", Applied: true, Status: "ready", RequiredEvidence: []string{"embedded migration exists"}}},
		AppliedCount:                         1,
		PendingCount:                         0,
		SchemaMigrationsTablePresent:         true,
		FullLedgerTablePresent:               false,
		PreflightApplyVerifyRemediationReady: false,
		Capabilities:                         []string{"read_embedded_migrations", "read_schema_migration_names"},
		ForbiddenActions:                     []string{"apply_migration", "write_migration_ledger", "rollback_database"},
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"migration_apply_attempted":           false,
			"database_write_attempted":            false,
			"area_matrix_protected_paths_touched": false,
		},
		GeneratedAt: generated,
	}
	handler := NewHandler(fakeProjectStore{migrationLedger: readiness})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/ops/migration-ledger-readiness", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("migration ledger readiness = %d body=%s", resp.Code, resp.Body.String())
	}
	var body migrationLedgerReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode migration ledger readiness: %v", err)
	}
	if body.Status != "needs_attention" || body.Mode != "read_only_migration_ledger_readiness" {
		t.Fatalf("unexpected migration ledger readiness: %+v", body)
	}
	if body.FullLedgerTablePresent || body.PreflightApplyVerifyRemediationReady || len(body.Entries) != 1 {
		t.Fatalf("unexpected migration ledger proof state: %+v", body)
	}
	if !body.SafetyFacts["read_only"] || body.SafetyFacts["migration_apply_attempted"] ||
		body.SafetyFacts["database_write_attempted"] || body.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("migration ledger readiness opened unsafe fact: %+v", body.SafetyFacts)
	}
	if !containsString(body.ForbiddenActions, "apply_migration") ||
		!containsString(body.ForbiddenActions, "rollback_database") {
		t.Fatalf("missing migration forbidden actions: %+v", body.ForbiddenActions)
	}

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/ops/migration-ledger-readiness", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("migration ledger readiness POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func TestOperationsReadinessEndpoint(t *testing.T) {
	generated := time.Date(2026, 7, 3, 14, 0, 0, 0, time.UTC)
	service := project.LocalServiceStatus{Status: "ready", Mode: "local_service", GeneratedAt: generated}
	support := project.BuildSupportBundlePreview(project.BackupManifest{Status: "ready", ManifestHash: "backup-hash"}, project.AuditCoverage{Status: "warn"}, project.SupportBundlePreviewOptions{GeneratedAt: generated})
	ledger := project.MigrationLedgerReadiness{
		Status:                               "needs_attention",
		Mode:                                 "read_only_migration_ledger_readiness",
		AppliedCount:                         1,
		SchemaMigrationsTablePresent:         true,
		FullLedgerTablePresent:               false,
		PreflightApplyVerifyRemediationReady: false,
		SafetyFacts:                          map[string]bool{"read_only": true, "database_write_attempted": false},
		GeneratedAt:                          generated,
	}
	readiness := project.BuildOperationsReadiness(service, support, ledger, project.OperationsReadinessOptions{GeneratedAt: generated})
	handler := NewHandler(fakeProjectStore{operations: readiness})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/ops/readiness", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("operations readiness = %d body=%s", resp.Code, resp.Body.String())
	}
	var body operationsReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode operations readiness: %v", err)
	}
	if body.Status != "needs_attention" || body.Mode != "read_only_operations_readiness" ||
		body.TelemetryDefault != "local_only" || body.ManagedOpsStatus != "deferred_v1x" {
		t.Fatalf("unexpected operations readiness: %+v", body)
	}
	if !body.SafetyFacts["read_only"] || body.SafetyFacts["support_bundle_exported"] ||
		body.SafetyFacts["remote_telemetry_enabled"] || body.SafetyFacts["managed_upgrade_attempted"] ||
		body.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("operations readiness opened unsafe fact: %+v", body.SafetyFacts)
	}
	assertOperationsAPIItem(t, body, "install_migrate_start_register_smoke", "needs_attention")
	assertOperationsAPIItem(t, body, "metadata_only_support_bundle_preview", "ready")
	assertOperationsAPIItem(t, body, "managed_ops_deferred", "deferred")

	methodResp := httptest.NewRecorder()
	handler.ServeHTTP(methodResp, httptest.NewRequest(http.MethodPost, "/api/v1/ops/readiness", nil))
	if methodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("operations readiness POST = %d body=%s", methodResp.Code, methodResp.Body.String())
	}
}

func assertOperationsAPIItem(t *testing.T, body operationsReadinessResponse, key string, status string) {
	t.Helper()
	for _, item := range body.Items {
		if item.Key == key {
			if item.Status != status {
				t.Fatalf("operations API item %s status = %q, want %q: %+v", key, item.Status, status, item)
			}
			if len(item.RequiredEvidence) == 0 {
				t.Fatalf("operations API item %s missing required evidence: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("operations API item %s not found: %+v", key, body.Items)
}

func TestBackupManifestEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		backup: project.BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			Scope:         "project",
			ProjectKey:    "areamatrix",
			SchemaVersion: 1,
			GeneratedAt:   created,
			ManifestHash:  "abc123",
			TableCounts: []project.BackupTableCount{
				{Table: "projects", Rows: 1},
				{Table: "artifacts", Rows: 2},
			},
			Projects: []project.BackupProjectManifest{
				{
					Project: project.Record{
						ID:              1,
						Key:             "areamatrix",
						Name:            "AreaMatrix",
						Kind:            "product-repo",
						Adapter:         "areamatrix",
						WorkflowProfile: "areamatrix",
					},
					Inventory:     project.ImportInventory{Versions: 2, Residuals: 10, Artifacts: 6},
					ArtifactCount: 1,
					Artifacts: []project.BackupArtifactSummary{
						{
							ID:                7,
							ProjectID:         1,
							WorkflowVersionID: 2,
							ArtifactType:      "runner_preview_report",
							StorageBackend:    "local",
							URI:               "/tmp/areaflow/artifacts/areamatrix/report.json",
							SourcePath:        "report.json",
							SHA256:            "def456",
							SizeBytes:         128,
							ContentType:       "application/json",
							CreatedAt:         created,
						},
					},
				},
			},
			Capabilities:     []string{"export_postgres_metadata", "export_artifact_metadata"},
			ForbiddenActions: []string{"restore_database", "read_artifact_contents"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/manifest", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("backup manifest status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body backupManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode backup manifest response: %v", err)
	}
	if body.Status != "ready" || body.Mode != "read_only_manifest" || body.Scope != "project" || body.ProjectKey != "areamatrix" || body.ManifestHash != "abc123" {
		t.Fatalf("unexpected backup manifest: %+v", body)
	}
	if len(body.TableCounts) != 2 || body.TableCounts[0].Table != "projects" {
		t.Fatalf("unexpected table counts: %+v", body.TableCounts)
	}
	if len(body.Projects) != 1 || body.Projects[0].Project.Key != "areamatrix" {
		t.Fatalf("unexpected backup projects: %+v", body.Projects)
	}
	if body.Projects[0].ArtifactCount != 1 || body.Projects[0].Artifacts[0].SHA256 != "def456" {
		t.Fatalf("unexpected backup artifacts: %+v", body.Projects[0].Artifacts)
	}
	if len(body.ForbiddenActions) != 2 || body.GeneratedAt == "" {
		t.Fatalf("unexpected backup guardrails: %+v", body)
	}
}

func TestBackupManifestEndpointForwardsProjectScope(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "project alias", path: "/api/v1/backup/manifest?project=areamatrix"},
		{name: "project key", path: "/api/v1/backup/manifest?project_key=areamatrix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got project.BackupManifestOptions
			handler := NewHandler(fakeProjectStore{
				backup: project.BackupManifest{
					Status:     "ready",
					Mode:       "read_only_manifest",
					Scope:      "project",
					ProjectKey: "areamatrix",
				},
				backupHook: func(options project.BackupManifestOptions) {
					got = options
				},
			})
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if resp.Code != http.StatusOK {
				t.Fatalf("backup manifest status = %d body=%s", resp.Code, resp.Body.String())
			}
			if got.ProjectKey != "areamatrix" {
				t.Fatalf("project scope not forwarded: %+v", got)
			}
			var body backupManifestResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode backup manifest: %v", err)
			}
			if body.Scope != "project" || body.ProjectKey != "areamatrix" {
				t.Fatalf("project scope not encoded: %+v", body)
			}
		})
	}
}

func TestRestorePlanEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		restore: project.RestorePlan{
			Status:        "needs_attention",
			Mode:          "read_only_restore_plan",
			Scope:         "project",
			ProjectKey:    "areamatrix",
			SchemaVersion: 1,
			ManifestHash:  "abc123",
			Projects: []project.Record{
				{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Kind: "product-repo", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
			},
			Items: []project.RestorePlanItem{
				{
					Key:      "manifest_shape",
					Category: "manifest",
					Status:   "ready",
					Message:  "backup manifest has schema version and stable hash",
					Metadata: map[string]any{"manifest_hash": "abc123"},
				},
				{
					Key:      "artifact_integrity:areamatrix",
					Category: "artifact",
					Status:   "needs_attention",
					Message:  "artifact integrity has warnings or skipped references",
					Metadata: map[string]any{"skipped_artifacts": float64(1)},
				},
			},
			Capabilities:     []string{"generate_restore_plan"},
			ForbiddenActions: []string{"restore_database", "apply_restore"},
			GeneratedAt:      created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/backup/restore-plan", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("restore plan status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body restorePlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode restore plan response: %v", err)
	}
	if body.Status != "needs_attention" || body.Mode != "read_only_restore_plan" || body.Scope != "project" || body.ProjectKey != "areamatrix" || body.ManifestHash != "abc123" {
		t.Fatalf("unexpected restore plan: %+v", body)
	}
	if len(body.Projects) != 1 || body.Projects[0].Key != "areamatrix" {
		t.Fatalf("unexpected restore projects: %+v", body.Projects)
	}
	if len(body.Items) != 2 || body.Items[1].Key != "artifact_integrity:areamatrix" {
		t.Fatalf("unexpected restore items: %+v", body.Items)
	}
	if len(body.ForbiddenActions) != 2 || body.GeneratedAt == "" {
		t.Fatalf("unexpected restore guardrails: %+v", body)
	}
}

func TestRestorePlanEndpointForwardsProjectScope(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "project alias", path: "/api/v1/backup/restore-plan?project=areamatrix"},
		{name: "project key", path: "/api/v1/backup/restore-plan?project_key=areamatrix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got project.RestorePlanOptions
			handler := NewHandler(fakeProjectStore{
				restore: project.RestorePlan{
					Status:     "needs_attention",
					Mode:       "read_only_restore_plan",
					Scope:      "project",
					ProjectKey: "areamatrix",
				},
				restoreHook: func(options project.RestorePlanOptions) {
					got = options
				},
			})
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if resp.Code != http.StatusOK {
				t.Fatalf("restore plan status = %d body=%s", resp.Code, resp.Body.String())
			}
			if got.ProjectKey != "areamatrix" {
				t.Fatalf("project scope not forwarded: %+v", got)
			}
			var body restorePlanResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode restore plan: %v", err)
			}
			if body.Scope != "project" || body.ProjectKey != "areamatrix" {
				t.Fatalf("project scope not encoded: %+v", body)
			}
		})
	}
}

func TestReleaseReadinessEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		release: project.ReleaseReadiness{
			Status: "needs_attention",
			Mode:   "read_only_release_readiness",
			Backup: project.BackupManifest{
				Status:        "ready",
				Mode:          "read_only_manifest",
				SchemaVersion: 1,
				ManifestHash:  "hash-a",
				Projects:      []project.BackupProjectManifest{{Project: record}},
				GeneratedAt:   created,
			},
			RestorePlan: project.RestorePlan{
				Status:        "needs_attention",
				Mode:          "read_only_restore_plan",
				SchemaVersion: 1,
				ManifestHash:  "hash-a",
				Projects:      []project.Record{record},
				Items:         []project.RestorePlanItem{{Key: "artifact_inventory", Category: "artifact", Status: "needs_attention"}},
				GeneratedAt:   created,
			},
			AuditCoverage: project.AuditCoverage{
				Status:              "warn",
				Mode:                "read_only_audit_coverage",
				Scope:               "platform",
				TotalAuditEvents:    10,
				CoveredRequirements: 8,
				GapRequirements:     3,
				GeneratedAt:         created,
			},
			Projects: []project.ReleaseReadinessProject{
				{
					Project:             record,
					Status:              "needs_attention",
					NeedsAttentionItems: 1,
					Permission:          project.PermissionPolicyDoctor{Status: "pass", Mode: "read_only_permission_policy_doctor", Project: record, GeneratedAt: created},
					ArtifactIntegrity:   project.ArtifactIntegrityReport{Status: "warn", Mode: "read_only_artifact_integrity", Project: record, CheckedArtifacts: 2, PassedArtifacts: 1, SkippedArtifacts: 1, GeneratedAt: created},
					Conformance:         project.ConformanceReport{Status: "pass", Mode: "read_only_adapter_profile_conformance", Project: record, ProfileID: "areamatrix", Adapter: "areamatrix", ProfileHash: "hash-profile", StageCount: 16, GateCount: 17, GeneratedAt: created},
				},
			},
			Items: []project.ReleaseReadinessItem{
				{Key: "backup_manifest", Category: "backup", Status: "ready", Message: "backup manifest is ready"},
				{Key: "restore_plan", Category: "restore", Status: "needs_attention", Message: "restore dry-run plan needs attention"},
				{Key: "audit_coverage", Category: "audit", Status: "needs_attention", Message: "audit coverage has gaps"},
				{Key: "artifact_integrity:areamatrix", Category: "artifact", Status: "needs_attention", Message: "artifact integrity has warnings"},
			},
			Capabilities:     []string{"generate_release_readiness"},
			ForbiddenActions: []string{"restore_database", "write_project_files", "start_worker"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/readiness", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release readiness status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release readiness: %v", err)
	}
	if body.Status != "needs_attention" || body.Mode != "read_only_release_readiness" {
		t.Fatalf("unexpected release readiness: %+v", body)
	}
	if body.RestorePlan.Status != "needs_attention" || body.AuditCoverage.Status != "warn" {
		t.Fatalf("unexpected nested readiness: %+v", body)
	}
	if len(body.Projects) != 1 || body.Projects[0].Project.Key != "areamatrix" || body.Projects[0].ArtifactIntegrity.Status != "warn" {
		t.Fatalf("unexpected project release readiness: %+v", body.Projects)
	}
	if len(body.Items) != 4 || body.Items[1].Key != "restore_plan" || body.Items[1].Status != "needs_attention" {
		t.Fatalf("unexpected release items: %+v", body.Items)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[0] != "restore_database" {
		t.Fatalf("unexpected forbidden actions: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
	assertReal100DisambiguationResponse(t, body, project.ReleasePreviewReadinessScope, "not_release_candidate_evidence")
}

func TestReleaseRemediationPlanEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		remediation: project.ReleaseRemediationPlan{
			Status: "needs_attention",
			Mode:   "read_only_release_remediation_plan",
			Readiness: project.ReleaseReadiness{
				Status: "needs_attention",
				Mode:   "read_only_release_readiness",
				Backup: project.BackupManifest{Status: "ready", Mode: "read_only_manifest", SchemaVersion: 1, ManifestHash: "hash-a", Projects: []project.BackupProjectManifest{{Project: record}}, GeneratedAt: created},
				RestorePlan: project.RestorePlan{
					Status:        "needs_attention",
					Mode:          "read_only_restore_plan",
					SchemaVersion: 1,
					ManifestHash:  "hash-a",
					Projects:      []project.Record{record},
					GeneratedAt:   created,
				},
				AuditCoverage: project.AuditCoverage{Status: "warn", Mode: "read_only_audit_coverage", Scope: "platform", GapRequirements: 3, GeneratedAt: created},
				GeneratedAt:   created,
			},
			Actions: []project.ReleaseRemediationAction{
				{
					Key:               "remediate:restore_plan",
					Category:          "restore",
					Status:            "needs_attention",
					SourceItem:        "restore_plan",
					RecommendedAction: "decide artifact archive policy",
					Rationale:         "project reference originals remain outside AreaFlow",
					Owner:             "release_owner",
					NextCommand:       "areaflow backup restore-plan --json",
					Acceptance:        "restore plan is ready or accepted",
					Metadata:          map[string]any{"restore_status": "needs_attention"},
				},
				{
					Key:               "remediate:audit_coverage",
					Category:          "audit",
					Status:            "needs_attention",
					SourceItem:        "audit_coverage",
					RecommendedAction: "close enabled audit gaps",
					Rationale:         "audit gaps must be explicit",
					Owner:             "platform_owner",
					NextCommand:       "areaflow audit coverage --json",
					Acceptance:        "audit coverage is pass or accepted",
					Metadata:          map[string]any{"gap_requirements": float64(3)},
				},
			},
			Capabilities:     []string{"generate_remediation_plan"},
			ForbiddenActions: []string{"write_project_files", "mark_gap_accepted"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/remediation-plan", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release remediation status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseRemediationPlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release remediation plan: %v", err)
	}
	if body.Status != "needs_attention" || body.Mode != "read_only_release_remediation_plan" {
		t.Fatalf("unexpected remediation plan: %+v", body)
	}
	if body.Readiness.Status != "needs_attention" || body.Readiness.RestorePlan.Status != "needs_attention" {
		t.Fatalf("unexpected readiness summary: %+v", body.Readiness)
	}
	if len(body.Actions) != 2 || body.Actions[0].Key != "remediate:restore_plan" || body.Actions[1].NextCommand != "areaflow audit coverage --json" {
		t.Fatalf("unexpected remediation actions: %+v", body.Actions)
	}
	if len(body.ForbiddenActions) != 2 || body.ForbiddenActions[1] != "mark_gap_accepted" {
		t.Fatalf("unexpected remediation guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseAcceptancePreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		acceptance: project.ReleaseAcceptancePreview{
			Status: "needs_decision",
			Mode:   "read_only_release_acceptance_preview",
			Remediation: project.ReleaseRemediationPlan{
				Status: "needs_attention",
				Mode:   "read_only_release_remediation_plan",
				Readiness: project.ReleaseReadiness{
					Status:      "needs_attention",
					Mode:        "read_only_release_readiness",
					GeneratedAt: created,
				},
				Actions: []project.ReleaseRemediationAction{
					{
						Key:        "remediate:restore_plan",
						Category:   "restore",
						Status:     "needs_attention",
						SourceItem: "restore_plan",
						Owner:      "release_owner",
						Acceptance: "restore plan is ready or accepted",
						Metadata:   map[string]any{"restore_status": "needs_attention"},
					},
				},
				GeneratedAt: created,
			},
			Decisions: []project.ReleaseAcceptanceDecision{
				{
					Key:              "accept:restore_plan",
					SourceAction:     "remediate:restore_plan",
					Category:         "restore",
					Status:           "needs_decision",
					AcceptanceType:   "metadata_only_history",
					Owner:            "release_owner",
					Reason:           "metadata-only history requires explicit acceptance",
					RequiredEvidence: []string{"release notes state metadata-only artifacts"},
					NextCommand:      "areaflow backup restore-plan --json",
					Metadata:         map[string]any{"restore_status": "needs_attention"},
				},
			},
			Capabilities:     []string{"generate_acceptance_preview"},
			ForbiddenActions: []string{"write_database", "mark_gap_accepted", "create_approval"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/acceptance-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release acceptance preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseAcceptancePreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release acceptance preview: %v", err)
	}
	if body.Status != "needs_decision" || body.Mode != "read_only_release_acceptance_preview" {
		t.Fatalf("unexpected acceptance preview: %+v", body)
	}
	if body.Remediation.Status != "needs_attention" || body.Remediation.Readiness.Status != "needs_attention" {
		t.Fatalf("unexpected nested remediation: %+v", body.Remediation)
	}
	if len(body.Decisions) != 1 || body.Decisions[0].AcceptanceType != "metadata_only_history" {
		t.Fatalf("unexpected decisions: %+v", body.Decisions)
	}
	if len(body.Decisions[0].RequiredEvidence) != 1 || body.Decisions[0].NextCommand != "areaflow backup restore-plan --json" {
		t.Fatalf("unexpected decision guidance: %+v", body.Decisions[0])
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[1] != "mark_gap_accepted" {
		t.Fatalf("unexpected acceptance guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseAcceptanceGateEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		gateRelease: project.ReleaseAcceptanceGate{
			Status: "blocked",
			Mode:   "read_only_release_acceptance_gate",
			Preview: project.ReleaseAcceptancePreview{
				Status:      "needs_decision",
				Mode:        "read_only_release_acceptance_preview",
				GeneratedAt: created,
			},
			Items: []project.ReleaseAcceptanceGateItem{
				{
					Key:              "gate:accept:restore_plan",
					Category:         "restore",
					Status:           "blocked",
					DecisionStatus:   "needs_decision",
					AcceptanceType:   "metadata_only_history",
					Message:          "explicit release acceptance evidence is required before this exception can pass",
					Owner:            "release_owner",
					RequiredEvidence: []string{"release notes state metadata-only artifacts"},
					NextCommand:      "areaflow backup restore-plan --json",
					Metadata:         map[string]any{"restore_status": "needs_attention"},
				},
			},
			Capabilities:     []string{"evaluate_release_acceptance_gate"},
			ForbiddenActions: []string{"write_database", "mark_gap_accepted", "create_approval"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/acceptance-gate", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release acceptance gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseAcceptanceGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release acceptance gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_acceptance_gate" {
		t.Fatalf("unexpected acceptance gate: %+v", body)
	}
	if body.Preview.Status != "needs_decision" {
		t.Fatalf("unexpected nested preview: %+v", body.Preview)
	}
	if len(body.Items) != 1 || body.Items[0].DecisionStatus != "needs_decision" || body.Items[0].AcceptanceType != "metadata_only_history" {
		t.Fatalf("unexpected gate items: %+v", body.Items)
	}
	if len(body.Items[0].RequiredEvidence) != 1 || body.Items[0].NextCommand != "areaflow backup restore-plan --json" {
		t.Fatalf("unexpected gate guidance: %+v", body.Items[0])
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[2] != "create_approval" {
		t.Fatalf("unexpected gate guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionDoctorEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		exception: project.ReleaseExceptionDoctor{
			Status: "warn",
			Mode:   "read_only_release_exception_doctor",
			Gate: project.ReleaseAcceptanceGate{
				Status:      "blocked",
				Mode:        "read_only_release_acceptance_gate",
				GeneratedAt: created,
			},
			Checks: []project.ReleaseExceptionDoctorCheck{
				{
					Key:      "exception_record_schema",
					Category: "schema",
					Status:   "warn",
					Message:  "release exception record schema is designed but not enabled for writes",
					Metadata: map[string]any{"writes_enabled": false},
				},
				{
					Key:      "exception:gate:accept:restore_plan",
					Category: "restore",
					Status:   "warn",
					Message:  "release exception record is required before this gate item can pass",
					Metadata: map[string]any{"exception_writable": false},
				},
			},
			Capabilities:     []string{"check_exception_record_requirements"},
			ForbiddenActions: []string{"write_database", "mark_gap_accepted", "create_approval"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/exception-doctor", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release exception doctor status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseExceptionDoctorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release exception doctor: %v", err)
	}
	if body.Status != "warn" || body.Mode != "read_only_release_exception_doctor" {
		t.Fatalf("unexpected exception doctor: %+v", body)
	}
	if body.Gate.Status != "blocked" {
		t.Fatalf("unexpected nested gate: %+v", body.Gate)
	}
	if len(body.Checks) != 2 || body.Checks[0].Key != "exception_record_schema" || body.Checks[1].Category != "restore" {
		t.Fatalf("unexpected exception checks: %+v", body.Checks)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[1] != "mark_gap_accepted" {
		t.Fatalf("unexpected exception guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionRecordPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		recordPrev: project.ReleaseExceptionRecordPreview{
			Status: "draft",
			Mode:   "read_only_release_exception_record_preview",
			Doctor: project.ReleaseExceptionDoctor{
				Status:      "warn",
				Mode:        "read_only_release_exception_doctor",
				GeneratedAt: created,
			},
			Drafts: []project.ReleaseExceptionRecordDraft{
				{
					Key:              "release_exception:restore_plan",
					SourceGateItem:   "gate:accept:restore_plan",
					SourceDecision:   "needs_decision",
					AcceptanceType:   "metadata_only_history",
					Status:           "draft",
					Owner:            "release_owner",
					Reason:           "explicit release acceptance evidence is required",
					RequiredEvidence: []string{"release notes state metadata-only artifacts"},
					AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
					RollbackPlan:     "revoke the exception record and rerun release acceptance gate before release apply",
					ReviewRequired:   true,
					Metadata:         map[string]any{"exception_writable": false},
				},
			},
			Capabilities:     []string{"preview_exception_records"},
			ForbiddenActions: []string{"write_database", "insert_exception_record", "insert_audit_event"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/exception-record-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release exception record preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseExceptionRecordPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release exception record preview: %v", err)
	}
	if body.Status != "draft" || body.Mode != "read_only_release_exception_record_preview" {
		t.Fatalf("unexpected record preview: %+v", body)
	}
	if body.Doctor.Status != "warn" {
		t.Fatalf("unexpected nested doctor: %+v", body.Doctor)
	}
	if len(body.Drafts) != 1 || body.Drafts[0].Key != "release_exception:restore_plan" || body.Drafts[0].Status != "draft" {
		t.Fatalf("unexpected drafts: %+v", body.Drafts)
	}
	if len(body.Drafts[0].AuditActions) != 3 || body.Drafts[0].RollbackPlan == "" || !body.Drafts[0].ReviewRequired {
		t.Fatalf("unexpected draft audit/rollback guidance: %+v", body.Drafts[0])
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[1] != "insert_exception_record" {
		t.Fatalf("unexpected record preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionSchemaPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		schemaPrev: project.ReleaseExceptionSchemaPreview{
			Status: "needs_approval",
			Mode:   "read_only_release_exception_schema_preview",
			RecordPreview: project.ReleaseExceptionRecordPreview{
				Status:      "draft",
				Mode:        "read_only_release_exception_record_preview",
				GeneratedAt: created,
			},
			Tables: []project.ReleaseExceptionSchemaTable{
				{
					Name:    "release_exceptions",
					Purpose: "stores explicit release exception records",
					Columns: []project.ReleaseExceptionSchemaColumn{
						{Name: "exception_key", Type: "TEXT", Nullable: false, Purpose: "stable exception identifier"},
						{Name: "rollback_plan", Type: "TEXT", Nullable: false, Purpose: "rollback plan"},
					},
					Indexes: []project.ReleaseExceptionSchemaIndex{
						{Name: "release_exceptions_key_idx", Columns: []string{"exception_key"}, Unique: true, Purpose: "lookup"},
					},
					ForeignKeys: []project.ReleaseExceptionSchemaForeignKey{
						{Column: "project_id", ReferencesTable: "projects", ReferencesColumn: "id", OnDelete: "CASCADE"},
					},
				},
			},
			ApplySteps: []project.ReleaseExceptionMigrationStep{
				{Order: 1, Action: "create_table", Description: "create release_exceptions table", SQLPreview: "CREATE TABLE IF NOT EXISTS release_exceptions (...)"},
			},
			RollbackSteps: []project.ReleaseExceptionMigrationStep{
				{Order: 1, Action: "drop_table", Description: "drop release_exceptions", SQLPreview: "DROP TABLE IF EXISTS release_exceptions"},
			},
			AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
			Capabilities:     []string{"preview_release_exception_schema"},
			ForbiddenActions: []string{"write_database", "create_migration_file", "run_migration"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/exception-schema-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release exception schema preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseExceptionSchemaPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release exception schema preview: %v", err)
	}
	if body.Status != "needs_approval" || body.Mode != "read_only_release_exception_schema_preview" {
		t.Fatalf("unexpected schema preview: %+v", body)
	}
	if body.RecordPreview.Status != "draft" {
		t.Fatalf("unexpected nested record preview: %+v", body.RecordPreview)
	}
	if len(body.Tables) != 1 || body.Tables[0].Name != "release_exceptions" || len(body.Tables[0].Columns) != 2 {
		t.Fatalf("unexpected tables: %+v", body.Tables)
	}
	if len(body.ApplySteps) != 1 || body.ApplySteps[0].Action != "create_table" {
		t.Fatalf("unexpected apply steps: %+v", body.ApplySteps)
	}
	if len(body.RollbackSteps) != 1 || body.RollbackSteps[0].Action != "drop_table" {
		t.Fatalf("unexpected rollback steps: %+v", body.RollbackSteps)
	}
	if len(body.AuditActions) != 3 || body.AuditActions[0] != "release.exception.request" {
		t.Fatalf("unexpected audit actions: %+v", body.AuditActions)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[1] != "create_migration_file" {
		t.Fatalf("unexpected schema preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionMigrationApprovalGateEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		migration: project.ReleaseExceptionMigrationApprovalGate{
			Status: "blocked",
			Mode:   "read_only_release_exception_migration_approval_gate",
			SchemaPreview: project.ReleaseExceptionSchemaPreview{
				Status:      "needs_approval",
				Mode:        "read_only_release_exception_schema_preview",
				GeneratedAt: created,
			},
			Items: []project.ReleaseExceptionMigrationApprovalGateItem{
				{
					Key:              "migration_approval:release_exception_schema",
					Category:         "migration",
					Status:           "blocked",
					ApprovalStatus:   "needs_approval",
					Message:          "explicit migration approval is required",
					Owner:            "release_owner",
					RequiredEvidence: []string{"approved migration approval record"},
					NextCommand:      "areaflow release exception-schema-preview --json",
					Metadata:         map[string]any{"risk_level": "R4 migration_security", "migration_writable": false},
				},
			},
			Capabilities:     []string{"evaluate_release_exception_migration_approval_gate"},
			ForbiddenActions: []string{"write_database", "create_migration_file", "run_migration", "approve_migration"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/exception-migration-approval-gate", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release exception migration approval gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseExceptionMigrationApprovalGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release exception migration approval gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_exception_migration_approval_gate" {
		t.Fatalf("unexpected migration approval gate: %+v", body)
	}
	if body.SchemaPreview.Status != "needs_approval" {
		t.Fatalf("unexpected nested schema preview: %+v", body.SchemaPreview)
	}
	if len(body.Items) != 1 || body.Items[0].ApprovalStatus != "needs_approval" || body.Items[0].Status != "blocked" {
		t.Fatalf("unexpected migration approval items: %+v", body.Items)
	}
	if body.Items[0].Metadata["migration_writable"] != false {
		t.Fatalf("unexpected migration approval metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.ForbiddenActions) != 4 || body.ForbiddenActions[1] != "create_migration_file" {
		t.Fatalf("unexpected migration approval guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionApplyPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		applyPrev: project.ReleaseExceptionApplyPreview{
			Status: "blocked",
			Mode:   "read_only_release_exception_apply_preview",
			MigrationGate: project.ReleaseExceptionMigrationApprovalGate{
				Status: "blocked",
				Mode:   "read_only_release_exception_migration_approval_gate",
			},
			Items: []project.ReleaseExceptionApplyPreviewItem{
				{
					Key:              "release_exception_apply:migration_approval",
					Category:         "migration",
					Status:           "blocked",
					Action:           "wait_for_migration_approval",
					Message:          "release exception apply is blocked",
					Owner:            "release_owner",
					RequiredEvidence: []string{"release exception migration approval gate returns pass"},
					NextCommand:      "areaflow release exception-migration-approval-gate --json",
					Metadata:         map[string]any{"risk_level": "R4 migration_security", "apply_writable": false},
				},
			},
			ApplySteps: []project.ReleaseExceptionApplyPreviewStep{
				{Order: 1, Action: "verify_migration_approval", Description: "confirm gate passes", BlockedBy: []string{"migration_approval:release_exception_schema"}},
			},
			RollbackSteps: []project.ReleaseExceptionApplyPreviewStep{
				{Order: 1, Action: "disable_exception_writes", Description: "disable writes"},
			},
			Capabilities:     []string{"preview_release_exception_apply_plan"},
			ForbiddenActions: []string{"write_database", "run_migration", "insert_exception_record", "apply_release"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/exception-apply-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release exception apply preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseExceptionApplyPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release exception apply preview: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_exception_apply_preview" {
		t.Fatalf("unexpected apply preview: %+v", body)
	}
	if body.MigrationGate.Status != "blocked" {
		t.Fatalf("unexpected nested migration gate: %+v", body.MigrationGate)
	}
	if len(body.Items) != 1 || body.Items[0].Action != "wait_for_migration_approval" || body.Items[0].Status != "blocked" {
		t.Fatalf("unexpected apply preview items: %+v", body.Items)
	}
	if body.Items[0].Metadata["apply_writable"] != false {
		t.Fatalf("unexpected apply preview metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.ApplySteps) != 1 || body.ApplySteps[0].Action != "verify_migration_approval" {
		t.Fatalf("unexpected apply steps: %+v", body.ApplySteps)
	}
	if len(body.RollbackSteps) != 1 || body.RollbackSteps[0].Action != "disable_exception_writes" {
		t.Fatalf("unexpected rollback steps: %+v", body.RollbackSteps)
	}
	if len(body.ForbiddenActions) != 4 || body.ForbiddenActions[1] != "run_migration" {
		t.Fatalf("unexpected apply preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseFinalGateEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		finalGate: project.ReleaseFinalGate{
			Status: "blocked",
			Mode:   "read_only_release_final_gate",
			Readiness: project.ReleaseReadiness{
				Status:      "needs_attention",
				Mode:        "read_only_release_readiness",
				GeneratedAt: created,
			},
			AcceptanceGate: project.ReleaseAcceptanceGate{
				Status:      "blocked",
				Mode:        "read_only_release_acceptance_gate",
				GeneratedAt: created,
			},
			ExceptionApply: project.ReleaseExceptionApplyPreview{
				Status:      "blocked",
				Mode:        "read_only_release_exception_apply_preview",
				GeneratedAt: created,
			},
			Items: []project.ReleaseFinalGateItem{
				{
					Key:              "final_gate:release_readiness",
					Category:         "readiness",
					Status:           "blocked",
					Message:          "release readiness is not ready",
					Owner:            "release_owner",
					RequiredEvidence: []string{"release readiness status ready"},
					NextCommand:      "areaflow release readiness --json",
					Metadata:         map[string]any{"readiness_status": "needs_attention"},
				},
			},
			Capabilities:     []string{"evaluate_release_final_gate"},
			ForbiddenActions: []string{"write_database", "create_release_package", "apply_release"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/final-gate", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release final gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseFinalGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release final gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_final_gate" {
		t.Fatalf("unexpected final gate: %+v", body)
	}
	if body.Readiness.Status != "needs_attention" || body.AcceptanceGate.Status != "blocked" || body.ExceptionApply.Status != "blocked" {
		t.Fatalf("unexpected nested final gate sources: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "final_gate:release_readiness" || body.Items[0].Status != "blocked" {
		t.Fatalf("unexpected final gate items: %+v", body.Items)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[1] != "create_release_package" {
		t.Fatalf("unexpected final gate guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseFinalGateEndpointForwardsProjectScope(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "project alias", path: "/api/v1/release/final-gate?project=areamatrix"},
		{name: "project key", path: "/api/v1/release/final-gate?project_key=areamatrix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got project.ReleaseFinalGateOptions
			handler := NewHandler(fakeProjectStore{
				finalGate: project.ReleaseFinalGate{
					Status:     "blocked",
					Mode:       "read_only_release_final_gate",
					Scope:      "project",
					ProjectKey: "areamatrix",
				},
				finalGateHook: func(options project.ReleaseFinalGateOptions) {
					got = options
				},
			})
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if resp.Code != http.StatusOK {
				t.Fatalf("release final gate status = %d body=%s", resp.Code, resp.Body.String())
			}
			if got.ProjectKey != "areamatrix" {
				t.Fatalf("project scope not forwarded: %+v", got)
			}
			var body releaseFinalGateResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode release final gate: %v", err)
			}
			if body.Scope != "project" || body.ProjectKey != "areamatrix" {
				t.Fatalf("project scope not encoded: %+v", body)
			}
		})
	}
}

func TestReleaseEndpointsForwardProjectScope(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		store    func(*string) fakeProjectStore
		response any
	}{
		{
			name: "readiness",
			path: "/api/v1/release/readiness?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					release: project.ReleaseReadiness{
						Status:     "ready",
						Mode:       "read_only_release_readiness",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					releaseHook: func(options project.ReleaseReadinessOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseReadinessResponse{},
		},
		{
			name: "remediation plan",
			path: "/api/v1/release/remediation-plan?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					remediation: project.ReleaseRemediationPlan{
						Status:     "ready",
						Mode:       "read_only_release_remediation_plan",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					remediationHook: func(options project.ReleaseRemediationOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseRemediationPlanResponse{},
		},
		{
			name: "acceptance preview",
			path: "/api/v1/release/acceptance-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					acceptance: project.ReleaseAcceptancePreview{
						Status:     "ready",
						Mode:       "read_only_release_acceptance_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					acceptanceHook: func(options project.ReleaseAcceptancePreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseAcceptancePreviewResponse{},
		},
		{
			name: "acceptance gate",
			path: "/api/v1/release/acceptance-gate?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					gateRelease: project.ReleaseAcceptanceGate{
						Status:     "pass",
						Mode:       "read_only_release_acceptance_gate",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					gateReleaseHook: func(options project.ReleaseAcceptanceGateOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseAcceptanceGateResponse{},
		},
		{
			name: "exception doctor",
			path: "/api/v1/release/exception-doctor?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					exception: project.ReleaseExceptionDoctor{
						Status:     "pass",
						Mode:       "read_only_release_exception_doctor",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					exceptionHook: func(options project.ReleaseExceptionDoctorOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseExceptionDoctorResponse{},
		},
		{
			name: "exception record preview",
			path: "/api/v1/release/exception-record-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					recordPrev: project.ReleaseExceptionRecordPreview{
						Status:     "ready",
						Mode:       "read_only_release_exception_record_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					recordPrevHook: func(options project.ReleaseExceptionRecordPreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseExceptionRecordPreviewResponse{},
		},
		{
			name: "exception schema preview",
			path: "/api/v1/release/exception-schema-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					schemaPrev: project.ReleaseExceptionSchemaPreview{
						Status:     "needs_approval",
						Mode:       "read_only_release_exception_schema_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					schemaPrevHook: func(options project.ReleaseExceptionSchemaPreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseExceptionSchemaPreviewResponse{},
		},
		{
			name: "exception migration approval gate",
			path: "/api/v1/release/exception-migration-approval-gate?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					migration: project.ReleaseExceptionMigrationApprovalGate{
						Status:     "pass",
						Mode:       "read_only_release_exception_migration_approval_gate",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					migrationHook: func(options project.ReleaseExceptionMigrationApprovalGateOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseExceptionMigrationApprovalGateResponse{},
		},
		{
			name: "exception apply preview",
			path: "/api/v1/release/exception-apply-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					applyPrev: project.ReleaseExceptionApplyPreview{
						Status:     "ready",
						Mode:       "read_only_release_exception_apply_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					applyPrevHook: func(options project.ReleaseExceptionApplyPreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseExceptionApplyPreviewResponse{},
		},
		{
			name: "evidence bundle",
			path: "/api/v1/release/evidence-bundle?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					evidence: project.ReleaseEvidenceBundle{
						Status:     "ready",
						Mode:       "read_only_release_evidence_bundle",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					evidenceHook: func(options project.ReleaseEvidenceBundleOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseEvidenceBundleResponse{},
		},
		{
			name: "package preview",
			path: "/api/v1/release/package-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					packagePrev: project.ReleasePackagePreview{
						Status:     "ready",
						Mode:       "read_only_release_package_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					packagePrevHook: func(options project.ReleasePackagePreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releasePackagePreviewResponse{},
		},
		{
			name: "distribution preview",
			path: "/api/v1/release/distribution-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					distPrev: project.ReleaseDistributionPreview{
						Status:     "ready",
						Mode:       "read_only_release_distribution_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					distPrevHook: func(options project.ReleaseDistributionPreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseDistributionPreviewResponse{},
		},
		{
			name: "publish gate",
			path: "/api/v1/release/publish-gate?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					publishGate: project.ReleasePublishGate{
						Status:     "pass",
						Mode:       "read_only_release_publish_gate",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					publishGateHook: func(options project.ReleasePublishGateOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releasePublishGateResponse{},
		},
		{
			name: "publish approval preview",
			path: "/api/v1/release/publish-approval-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					publishAppr: project.ReleasePublishApprovalPreview{
						Status:     "needs_approval",
						Mode:       "read_only_release_publish_approval_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					publishApprHook: func(options project.ReleasePublishApprovalPreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releasePublishApprovalPreviewResponse{},
		},
		{
			name: "rollout plan preview",
			path: "/api/v1/release/rollout-plan-preview?project_key=areamatrix",
			store: func(got *string) fakeProjectStore {
				return fakeProjectStore{
					rollout: project.ReleaseRolloutPlanPreview{
						Status:     "ready",
						Mode:       "read_only_release_rollout_plan_preview",
						Scope:      "project",
						ProjectKey: "areamatrix",
					},
					rolloutHook: func(options project.ReleaseRolloutPlanPreviewOptions) {
						*got = options.ProjectKey
					},
				}
			},
			response: &releaseRolloutPlanPreviewResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProjectKey := ""
			handler := NewHandler(tt.store(&gotProjectKey))
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if resp.Code != http.StatusOK {
				t.Fatalf("release endpoint status = %d body=%s", resp.Code, resp.Body.String())
			}
			if gotProjectKey != "areamatrix" {
				t.Fatalf("project scope not forwarded: %q", gotProjectKey)
			}
			if err := json.NewDecoder(resp.Body).Decode(tt.response); err != nil {
				t.Fatalf("decode release response: %v", err)
			}
			assertResponseProjectScope(t, tt.response, "areamatrix")
		})
	}
}

func assertResponseProjectScope(t *testing.T, response any, wantProjectKey string) {
	t.Helper()
	value := reflect.ValueOf(response)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		t.Fatalf("response must be a non-nil pointer: %T", response)
	}
	value = value.Elem()
	scope := value.FieldByName("Scope")
	projectKey := value.FieldByName("ProjectKey")
	if !scope.IsValid() || !projectKey.IsValid() {
		t.Fatalf("response missing scope fields: %T", response)
	}
	if scope.String() != "project" || projectKey.String() != wantProjectKey {
		t.Fatalf("project scope not encoded: scope=%q project_key=%q", scope.String(), projectKey.String())
	}
}

func TestReleaseEvidenceBundleEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		evidence: project.ReleaseEvidenceBundle{
			Status:     "blocked",
			Mode:       "read_only_release_evidence_bundle",
			BundleHash: "bundle-hash-1",
			FinalGate: project.ReleaseFinalGate{
				Status:      "blocked",
				Mode:        "read_only_release_final_gate",
				GeneratedAt: created,
			},
			Backup: project.BackupManifest{
				Status:        "ready",
				Mode:          "read_only_manifest",
				SchemaVersion: 1,
				ManifestHash:  "abc123",
				GeneratedAt:   created,
			},
			AuditCoverage: project.AuditCoverage{
				Status:      "warn",
				Mode:        "read_only_audit_coverage",
				GeneratedAt: created,
			},
			Items: []project.ReleaseEvidenceBundleItem{
				{
					Key:         "evidence:release_final_gate",
					Category:    "release_gate",
					Status:      "blocked",
					Source:      "release final-gate",
					Description: "release final go/no-go result",
					Metadata:    map[string]any{"final_gate_status": "blocked"},
				},
			},
			Capabilities:     []string{"assemble_release_evidence_index"},
			ForbiddenActions: []string{"create_release_package", "read_artifact_contents", "apply_release"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/evidence-bundle", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release evidence bundle status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseEvidenceBundleResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release evidence bundle: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_evidence_bundle" ||
		body.BundleHash != "bundle-hash-1" {
		t.Fatalf("unexpected evidence bundle: %+v", body)
	}
	if body.FinalGate.Status != "blocked" || body.Backup.Status != "ready" || body.AuditCoverage.Status != "warn" {
		t.Fatalf("unexpected nested evidence sources: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "evidence:release_final_gate" || body.Items[0].Status != "blocked" {
		t.Fatalf("unexpected evidence items: %+v", body.Items)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[1] != "read_artifact_contents" {
		t.Fatalf("unexpected evidence guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleasePackagePreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		packagePrev: project.ReleasePackagePreview{
			Status: "blocked",
			Mode:   "read_only_release_package_preview",
			EvidenceBundle: project.ReleaseEvidenceBundle{
				Status:      "blocked",
				Mode:        "read_only_release_evidence_bundle",
				BundleHash:  "bundle-hash-1",
				GeneratedAt: created,
			},
			PackageName: "areaflow-v1.0-release-evidence-preview",
			Items: []project.ReleasePackagePreviewItem{
				{
					Key:         "package:manifest",
					Category:    "manifest",
					Status:      "blocked",
					PackagePath: "release/manifest.json",
					Source:      "release evidence-bundle",
					Description: "release package manifest preview",
					Metadata:    map[string]any{"package_writable": false},
				},
			},
			Capabilities:     []string{"preview_release_package_manifest"},
			ForbiddenActions: []string{"create_release_package", "read_artifact_contents", "compress_artifacts"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/package-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release package preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releasePackagePreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release package preview: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_package_preview" {
		t.Fatalf("unexpected package preview: %+v", body)
	}
	if body.EvidenceBundle.Status != "blocked" || body.EvidenceBundle.BundleHash != "bundle-hash-1" ||
		body.PackageName == "" {
		t.Fatalf("unexpected nested package preview state: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "package:manifest" || body.Items[0].PackagePath != "release/manifest.json" {
		t.Fatalf("unexpected package preview items: %+v", body.Items)
	}
	if body.Items[0].Metadata["package_writable"] != false {
		t.Fatalf("unexpected package preview metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[0] != "create_release_package" {
		t.Fatalf("unexpected package preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseDistributionPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		distPrev: project.ReleaseDistributionPreview{
			Status: "blocked",
			Mode:   "read_only_release_distribution_preview",
			PackagePreview: project.ReleasePackagePreview{
				Status:      "blocked",
				Mode:        "read_only_release_package_preview",
				PackageName: "areaflow-v1.0-release-evidence-preview",
				GeneratedAt: created,
			},
			Items: []project.ReleaseDistributionPreviewItem{
				{
					Key:              "distribution:package_preview",
					Category:         "package",
					Status:           "blocked",
					Channel:          "release_package",
					Action:           "wait_for_package_preview",
					Message:          "release distribution is blocked until package preview is ready",
					Owner:            "release-owner",
					RequiredEvidence: []string{"release package preview ready"},
					NextCommand:      "areaflow release package-preview --json",
					Metadata:         map[string]any{"package_writable": false},
				},
			},
			Capabilities:     []string{"preview_release_distribution_channels"},
			ForbiddenActions: []string{"publish_release", "create_git_tag", "sign_release"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/distribution-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release distribution preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseDistributionPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release distribution preview: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_distribution_preview" {
		t.Fatalf("unexpected distribution preview: %+v", body)
	}
	if body.PackagePreview.Status != "blocked" || body.PackagePreview.PackageName == "" {
		t.Fatalf("unexpected nested distribution preview state: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "distribution:package_preview" || body.Items[0].Channel != "release_package" {
		t.Fatalf("unexpected distribution preview items: %+v", body.Items)
	}
	if body.Items[0].Metadata["package_writable"] != false {
		t.Fatalf("unexpected distribution preview metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[0] != "publish_release" {
		t.Fatalf("unexpected distribution preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleasePublishGateEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		publishGate: project.ReleasePublishGate{
			Status: "blocked",
			Mode:   "read_only_release_publish_gate",
			DistributionPreview: project.ReleaseDistributionPreview{
				Status:      "blocked",
				Mode:        "read_only_release_distribution_preview",
				GeneratedAt: created,
			},
			Items: []project.ReleasePublishGateItem{
				{
					Key:              "publish_gate:distribution_preview",
					Category:         "distribution_preview",
					Status:           "blocked",
					Channel:          "all",
					Message:          "release distribution preview blocks publish",
					Owner:            "release-owner",
					RequiredEvidence: []string{"release distribution preview status ready"},
					NextCommand:      "areaflow release distribution-preview --json",
					Metadata:         map[string]any{"publish_writable": false},
				},
			},
			Capabilities:     []string{"evaluate_release_publish_gate"},
			ForbiddenActions: []string{"publish_release", "create_git_tag", "push_git"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/publish-gate", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release publish gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releasePublishGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release publish gate: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_publish_gate" {
		t.Fatalf("unexpected publish gate: %+v", body)
	}
	if body.DistributionPreview.Status != "blocked" {
		t.Fatalf("unexpected nested publish gate state: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "publish_gate:distribution_preview" || body.Items[0].Channel != "all" {
		t.Fatalf("unexpected publish gate items: %+v", body.Items)
	}
	if body.Items[0].Metadata["publish_writable"] != false {
		t.Fatalf("unexpected publish gate metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[0] != "publish_release" {
		t.Fatalf("unexpected publish gate guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleasePublishApprovalPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		publishAppr: project.ReleasePublishApprovalPreview{
			Status: "blocked",
			Mode:   "read_only_release_publish_approval_preview",
			PublishGate: project.ReleasePublishGate{
				Status:      "blocked",
				Mode:        "read_only_release_publish_gate",
				GeneratedAt: created,
			},
			Items: []project.ReleasePublishApprovalPreviewItem{
				{
					Key:              "publish_approval:publish_gate",
					Category:         "publish_gate",
					Status:           "blocked",
					ApprovalStatus:   "blocked",
					Channel:          "all",
					Message:          "release publish approval cannot be requested until publish gate passes",
					Owner:            "release-owner",
					RequiredEvidence: []string{"publish gate pass"},
					NextCommand:      "areaflow release publish-gate --json",
					Metadata:         map[string]any{"approval_writable": false},
				},
			},
			Capabilities:     []string{"preview_release_publish_approval"},
			ForbiddenActions: []string{"create_approval", "approve_release", "publish_release"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/publish-approval-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release publish approval preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releasePublishApprovalPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release publish approval preview: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_publish_approval_preview" {
		t.Fatalf("unexpected publish approval preview: %+v", body)
	}
	if body.PublishGate.Status != "blocked" {
		t.Fatalf("unexpected nested publish approval preview state: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "publish_approval:publish_gate" || body.Items[0].ApprovalStatus != "blocked" {
		t.Fatalf("unexpected publish approval preview items: %+v", body.Items)
	}
	if body.Items[0].Metadata["approval_writable"] != false {
		t.Fatalf("unexpected publish approval preview metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[0] != "create_approval" {
		t.Fatalf("unexpected publish approval preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseRolloutPlanPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		rollout: project.ReleaseRolloutPlanPreview{
			Status: "blocked",
			Mode:   "read_only_release_rollout_plan_preview",
			PublishApprovalPreview: project.ReleasePublishApprovalPreview{
				Status:      "blocked",
				Mode:        "read_only_release_publish_approval_preview",
				GeneratedAt: created,
			},
			Items: []project.ReleaseRolloutPlanPreviewItem{
				{
					Key:              "rollout_plan:publish_approval",
					Category:         "publish_approval",
					Status:           "blocked",
					Stage:            "preflight",
					Action:           "wait_for_publish_approval_preview",
					Message:          "release rollout plan is blocked until publish approval preview is no longer blocked",
					Owner:            "release-owner",
					RequiredEvidence: []string{"publish approval preview ready"},
					NextCommand:      "areaflow release publish-approval-preview --json",
					Metadata:         map[string]any{"rollout_writable": false},
				},
			},
			RolloutSteps: []project.ReleaseRolloutPlanPreviewStep{
				{Order: 1, Stage: "preflight", Action: "verify_publish_approval", Description: "confirm approval"},
			},
			VerificationCheckpoints: []project.ReleaseRolloutPlanPreviewStep{
				{Order: 1, Stage: "approval", Action: "publish_approval_recorded", Description: "approval recorded"},
			},
			RollbackSteps: []project.ReleaseRolloutPlanPreviewStep{
				{Order: 1, Stage: "pause", Action: "pause_distribution", Description: "pause channels"},
			},
			Capabilities:     []string{"preview_release_rollout_plan"},
			ForbiddenActions: []string{"create_rollout", "write_release_state", "publish_release"},
			GeneratedAt:      created,
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/release/rollout-plan-preview", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("release rollout plan preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body releaseRolloutPlanPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode release rollout plan preview: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_release_rollout_plan_preview" {
		t.Fatalf("unexpected rollout plan preview: %+v", body)
	}
	if body.PublishApprovalPreview.Status != "blocked" {
		t.Fatalf("unexpected nested rollout plan preview state: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "rollout_plan:publish_approval" || body.Items[0].Action != "wait_for_publish_approval_preview" {
		t.Fatalf("unexpected rollout plan preview items: %+v", body.Items)
	}
	if body.Items[0].Metadata["rollout_writable"] != false {
		t.Fatalf("unexpected rollout plan preview metadata: %+v", body.Items[0].Metadata)
	}
	if len(body.RolloutSteps) != 1 || body.RolloutSteps[0].Action != "verify_publish_approval" {
		t.Fatalf("unexpected rollout steps: %+v", body.RolloutSteps)
	}
	if len(body.VerificationCheckpoints) != 1 || body.VerificationCheckpoints[0].Action != "publish_approval_recorded" {
		t.Fatalf("unexpected rollout verification checkpoints: %+v", body.VerificationCheckpoints)
	}
	if len(body.RollbackSteps) != 1 || body.RollbackSteps[0].Action != "pause_distribution" {
		t.Fatalf("unexpected rollout rollback steps: %+v", body.RollbackSteps)
	}
	if len(body.ForbiddenActions) != 3 || body.ForbiddenActions[0] != "create_rollout" {
		t.Fatalf("unexpected rollout plan preview guardrails: %+v", body.ForbiddenActions)
	}
	assertReal100GuardrailResponse(
		t,
		body.Real100Status,
		body.ReadinessScope,
		body.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestWorkerPoolSummaryEndpoint(t *testing.T) {
	heartbeat := time.Date(2026, 6, 29, 4, 5, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		workerPool: project.WorkerPoolSummary{
			Projects: []project.WorkerPoolProjectSummary{
				{
					Project:             project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
					Workers:             2,
					OnlineWorkers:       1,
					OfflineWorkers:      1,
					ActiveLeases:        1,
					NeedsRecoveryLeases: 1,
					QueuedTasks:         3,
					NeedsRecoveryTasks:  2,
					Capabilities:        []string{"read_project", "write_artifacts"},
					WorkerTypes:         []string{"local_host"},
					Scheduling: project.SchedulingPolicy{
						Priority:             100,
						MaxParallelTasks:     1,
						AgentRole:            "local_worker",
						RequiredCapabilities: []string{"read_project", "write_artifacts"},
						EngineProfile:        "codex-cli",
					},
					Role: project.RoleReadiness{
						RequiredRole: "local_worker",
						Matched:      true,
						MatchedTypes: []string{"local_host"},
						Status:       "ready",
					},
					Engine: project.EngineReadiness{
						ProfileID:      "codex-cli",
						Provider:       "codex-cli",
						SecretRef:      "none",
						SecretReady:    true,
						ResourceLimits: map[string]any{"max_active_leases": float64(1)},
						Status:         "blocked",
						BlockedReasons: []string{"engine_profile_disabled"},
					},
					Resources: project.ResourceReadiness{
						MaxActiveLeases: 1,
						Status:          "ready",
					},
					LastWorkerHeartbeat: &heartbeat,
				},
			},
			TotalProjects:      1,
			TotalWorkers:       2,
			TotalOnlineWorkers: 1,
			TotalActiveLeases:  1,
			TotalQueuedTasks:   3,
			TotalNeedsRecovery: 3,
			GeneratedAt:        heartbeat,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/worker-pool/summary", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("worker pool summary status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workerPoolSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode worker pool summary response: %v", err)
	}
	if body.TotalProjects != 1 || body.TotalWorkers != 2 || body.TotalQueuedTasks != 3 || body.TotalNeedsRecovery != 3 {
		t.Fatalf("unexpected worker pool totals: %+v", body)
	}
	if len(body.Projects) != 1 || body.Projects[0].Project.Key != "areamatrix" {
		t.Fatalf("unexpected worker pool projects: %+v", body.Projects)
	}
	if len(body.Projects[0].Capabilities) != 2 || body.Projects[0].LastWorkerHeartbeat == "" {
		t.Fatalf("unexpected worker pool project detail: %+v", body.Projects[0])
	}
	if body.Projects[0].Scheduling.Priority != 100 || body.Projects[0].Scheduling.MaxParallelTasks != 1 {
		t.Fatalf("unexpected worker pool scheduling policy: %+v", body.Projects[0].Scheduling)
	}
	if body.Projects[0].Scheduling.AgentRole != "local_worker" || body.Projects[0].Scheduling.EngineProfile != "codex-cli" {
		t.Fatalf("unexpected worker pool scheduling route: %+v", body.Projects[0].Scheduling)
	}
	if len(body.Projects[0].Scheduling.RequiredCapabilities) != 2 {
		t.Fatalf("unexpected worker pool scheduling capabilities: %+v", body.Projects[0].Scheduling)
	}
	if body.Projects[0].Engine.ProfileID != "codex-cli" || body.Projects[0].Engine.Status != "blocked" {
		t.Fatalf("unexpected worker pool engine readiness: %+v", body.Projects[0].Engine)
	}
	if body.Projects[0].Resources.MaxActiveLeases != 1 || body.Projects[0].Resources.Status != "ready" {
		t.Fatalf("unexpected worker pool resource readiness: %+v", body.Projects[0].Resources)
	}
	if len(body.Projects[0].WorkerTypes) != 1 || body.Projects[0].WorkerTypes[0] != "local_host" {
		t.Fatalf("unexpected worker pool worker types: %+v", body.Projects[0].WorkerTypes)
	}
	if body.Projects[0].Role.Status != "ready" || !body.Projects[0].Role.Matched {
		t.Fatalf("unexpected worker pool role readiness: %+v", body.Projects[0].Role)
	}
}

func TestWorkerPoolSchedulePreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 7, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		schedule: project.WorkerPoolSchedulePreview{
			Projects: []project.WorkerPoolProjectSchedule{
				{
					Project:     project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
					Priority:    100,
					MaxParallel: 1,
					AgentRole:   "local_worker",
					Role: project.RoleReadiness{
						RequiredRole: "local_worker",
						Matched:      true,
						MatchedTypes: []string{"local_host"},
						Status:       "ready",
					},
					EngineProfile: "codex-cli",
					Engine: project.EngineReadiness{
						ProfileID:      "codex-cli",
						Provider:       "codex-cli",
						SecretRef:      "none",
						SecretReady:    true,
						ResourceLimits: map[string]any{"max_active_leases": float64(1)},
						Status:         "ready",
					},
					Resources: project.ResourceReadiness{
						MaxActiveLeases: 1,
						Status:          "ready",
					},
					QueuedTasks:    2,
					OnlineWorkers:  1,
					AvailableSlots: 1,
					Capabilities:   []string{"read_project"},
					RequiredCaps:   []string{"read_project"},
					Recommended:    true,
					NextAction:     "worker_run_once_preview",
				},
				{
					Project:        project.Record{ID: 2, Key: "blocked"},
					Priority:       100,
					MaxParallel:    1,
					AgentRole:      "local_worker",
					QueuedTasks:    1,
					OnlineWorkers:  0,
					AvailableSlots: 0,
					RequiredCaps:   []string{"read_project"},
					BlockedReasons: []string{"no_online_workers"},
					NextAction:     "idle",
				},
			},
			Policy: project.WorkerPoolSchedulePolicy{
				Strategy:               "default_fifo",
				DefaultProjectPriority: 100,
				SlotStrategy:           "online_workers_minus_active_leases",
				DryRunOnly:             true,
			},
			GeneratedAt:   created,
			Recommended:   1,
			Blocked:       1,
			QueuedTasks:   3,
			AvailableSlot: 1,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/worker-pool/schedule-preview", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("worker pool schedule preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workerPoolSchedulePreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode worker pool schedule preview response: %v", err)
	}
	if body.Recommended != 1 || body.Blocked != 1 || body.QueuedTasks != 3 || !body.Policy.DryRunOnly {
		t.Fatalf("unexpected schedule preview totals: %+v", body)
	}
	if len(body.Projects) != 2 || !body.Projects[0].Recommended || body.Projects[0].NextAction != "worker_run_once_preview" {
		t.Fatalf("unexpected schedule preview projects: %+v", body.Projects)
	}
	if body.Projects[0].MaxParallel != 1 || body.Projects[0].AgentRole != "local_worker" || body.Projects[0].EngineProfile != "codex-cli" {
		t.Fatalf("unexpected schedule preview routing fields: %+v", body.Projects[0])
	}
	if body.Projects[0].Role.Status != "ready" || !body.Projects[0].Role.Matched {
		t.Fatalf("unexpected schedule preview role readiness: %+v", body.Projects[0].Role)
	}
	if len(body.Projects[0].RequiredCaps) != 1 || body.Projects[0].RequiredCaps[0] != "read_project" {
		t.Fatalf("unexpected schedule preview required caps: %+v", body.Projects[0])
	}
	if body.Projects[0].Engine.ProfileID != "codex-cli" || body.Projects[0].Engine.Status != "ready" {
		t.Fatalf("unexpected schedule preview engine readiness: %+v", body.Projects[0].Engine)
	}
	if body.Projects[0].Resources.MaxActiveLeases != 1 || body.Projects[0].Resources.Status != "ready" {
		t.Fatalf("unexpected schedule preview resource readiness: %+v", body.Projects[0].Resources)
	}
}

func TestCodexCLIAdapterPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 1, 23, 20, 0, 0, time.UTC)
	var captured project.CodexCLIAdapterPreviewOptions
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		codexPreviewHook: func(options project.CodexCLIAdapterPreviewOptions) {
			captured = options
		},
		codexPreview: project.CodexCLIAdapterPreview{
			Project: project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
			Status:  "blocked",
			Mode:    "read_only_codex_cli_adapter_preview",
			Engine: project.EngineReadiness{
				ProfileID:      "codex-cli",
				Provider:       "codex-cli",
				SecretRef:      "none",
				SecretReady:    true,
				ResourceLimits: map[string]any{},
				Status:         "blocked",
				BlockedReasons: []string{"engine_profile_disabled"},
			},
			Command: project.EngineCommandPreview{
				Command: "codex exec",
				Reason:  "run_commands capability not allowed",
			},
			Capabilities: []project.EngineCapabilityPreflight{{
				Capability: "execute_agents",
				Required:   true,
				Allowed:    false,
				Reason:     "capability not allowed",
			}},
			Paths: []project.EnginePathPreflight{{
				Path:       "workflow/versions/*/execution/**",
				Capability: "*",
				Effect:     "deny",
				Allowed:    true,
				Reason:     "forbidden path denied",
			}},
			ArtifactRedaction: project.ArtifactRedactionPlan{
				Status:         "ready",
				RetentionClass: "run_evidence",
				Rules:          []string{"redact stdout"},
				RedactedFields: []string{"stdout"},
			},
			ForbiddenActions:        []string{"execute_codex_cli"},
			Blockers:                []string{"engine_profile_disabled"},
			ProjectWriteAttempted:   false,
			ExecutionWriteAttempted: false,
			EngineCallAttempted:     false,
			CommandsRun:             false,
			SecretsResolved:         false,
			NetworkUsed:             false,
			GeneratedAt:             created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/engines/codex-cli/preview?command=codex%20exec", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("codex preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	if captured.Command != "codex exec" {
		t.Fatalf("captured command = %q", captured.Command)
	}
	var body codexCLIAdapterPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode codex preview response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Engine.ProfileID != "codex-cli" {
		t.Fatalf("unexpected codex preview body: %+v", body)
	}
	if body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || body.ExecutionAllowed {
		t.Fatalf("codex preview should be read-only and blocked: %+v", body)
	}
	if len(body.Capabilities) != 1 || len(body.Paths) != 1 || body.ArtifactRedaction.Status != "ready" {
		t.Fatalf("unexpected codex preview preflight detail: %+v", body)
	}
}

func TestWorkerPoolEndpointsUseAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 7, 0, 0, time.UTC)
	readyProject := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	blockedProject := project.Record{ID: 2, Key: "blocked", Name: "Blocked"}
	handler := NewHandler(fakeProjectStore{
		workerPool: project.WorkerPoolSummary{
			Projects: []project.WorkerPoolProjectSummary{{
				Project:             readyProject,
				Workers:             2,
				OnlineWorkers:       1,
				OfflineWorkers:      1,
				ActiveLeases:        1,
				NeedsRecoveryLeases: 1,
				QueuedTasks:         3,
				NeedsRecoveryTasks:  2,
				Capabilities:        []string{"read_project", "write_artifacts"},
				WorkerTypes:         []string{"local_host"},
				Scheduling: project.SchedulingPolicy{
					Priority:             100,
					MaxParallelTasks:     1,
					AgentRole:            "local_worker",
					RequiredCapabilities: []string{"read_project", "write_artifacts"},
					EngineProfile:        "codex-cli",
				},
				Role: project.RoleReadiness{
					RequiredRole: "local_worker",
					Matched:      true,
					MatchedTypes: []string{"local_host"},
					Status:       "ready",
				},
				Engine: project.EngineReadiness{
					ProfileID:      "codex-cli",
					Provider:       "codex-cli",
					SecretRef:      "none",
					SecretReady:    true,
					ResourceLimits: map[string]any{"max_active_leases": float64(1)},
					Status:         "ready",
				},
				Resources: project.ResourceReadiness{
					MaxActiveLeases: 1,
					Status:          "ready",
				},
				LastWorkerHeartbeat: &created,
			}},
			TotalProjects:      1,
			TotalWorkers:       2,
			TotalOnlineWorkers: 1,
			TotalActiveLeases:  1,
			TotalQueuedTasks:   3,
			TotalNeedsRecovery: 3,
			GeneratedAt:        created,
		},
		schedule: project.WorkerPoolSchedulePreview{
			Projects: []project.WorkerPoolProjectSchedule{{
				Project:     readyProject,
				Priority:    150,
				MaxParallel: 2,
				AgentRole:   "local_worker",
				Role: project.RoleReadiness{
					RequiredRole: "local_worker",
					Matched:      true,
					MatchedTypes: []string{"local_host"},
					Status:       "ready",
				},
				EngineProfile: "codex-cli",
				Engine: project.EngineReadiness{
					ProfileID:      "codex-cli",
					Provider:       "codex-cli",
					SecretRef:      "none",
					SecretReady:    true,
					ResourceLimits: map[string]any{"max_active_leases": float64(2)},
					Status:         "ready",
				},
				Resources: project.ResourceReadiness{
					MaxActiveLeases: 2,
					Status:          "ready",
				},
				QueuedTasks:    2,
				OnlineWorkers:  2,
				AvailableSlots: 1,
				Capabilities:   []string{"read_project", "write_artifacts"},
				RequiredCaps:   []string{"read_project"},
				Recommended:    true,
				NextAction:     "worker_run_once_preview",
			}, {
				Project:        blockedProject,
				Priority:       50,
				MaxParallel:    1,
				AgentRole:      "remote_worker",
				QueuedTasks:    1,
				OnlineWorkers:  1,
				AvailableSlots: 0,
				RequiredCaps:   []string{"read_project"},
				BlockedReasons: []string{"missing_agent_role:remote_worker"},
				NextAction:     "idle",
			}},
			Policy: project.WorkerPoolSchedulePolicy{
				Strategy:               "default_fifo",
				DefaultProjectPriority: 100,
				SlotStrategy:           "min_online_workers_and_project_parallelism_minus_active_leases",
				DryRunOnly:             true,
			},
			GeneratedAt:   created,
			Recommended:   1,
			Blocked:       1,
			QueuedTasks:   3,
			AvailableSlot: 1,
		},
	})

	summaryResp := httptest.NewRecorder()
	handler.ServeHTTP(summaryResp, httptest.NewRequest(http.MethodGet, "/api/v1/worker-pool/summary", nil))
	if summaryResp.Code != http.StatusOK {
		t.Fatalf("worker pool summary status = %d body=%s", summaryResp.Code, summaryResp.Body.String())
	}
	var summary workerPoolSummaryResponse
	if err := json.NewDecoder(summaryResp.Body).Decode(&summary); err != nil {
		t.Fatalf("decode worker pool summary response: %v", err)
	}
	if summary.TotalProjects != 1 || summary.TotalQueuedTasks != 3 || len(summary.Projects) != 1 {
		t.Fatalf("unexpected worker pool summary: %+v", summary)
	}
	if summary.Projects[0].Scheduling.EngineProfile != "codex-cli" || summary.Projects[0].Role.Status != "ready" {
		t.Fatalf("unexpected worker pool scheduling readiness: %+v", summary.Projects[0])
	}

	previewResp := httptest.NewRecorder()
	handler.ServeHTTP(previewResp, httptest.NewRequest(http.MethodGet, "/api/v1/worker-pool/schedule-preview", nil))
	if previewResp.Code != http.StatusOK {
		t.Fatalf("worker pool schedule preview status = %d body=%s", previewResp.Code, previewResp.Body.String())
	}
	var preview workerPoolSchedulePreviewResponse
	if err := json.NewDecoder(previewResp.Body).Decode(&preview); err != nil {
		t.Fatalf("decode worker pool schedule preview response: %v", err)
	}
	if preview.Recommended != 1 || preview.Blocked != 1 || !preview.Policy.DryRunOnly {
		t.Fatalf("unexpected schedule preview totals: %+v", preview)
	}
	if len(preview.Projects) != 2 || !preview.Projects[0].Recommended || preview.Projects[0].NextAction != "worker_run_once_preview" {
		t.Fatalf("unexpected schedule preview projects: %+v", preview.Projects)
	}
	if len(preview.Projects[1].BlockedReasons) != 1 || preview.Projects[1].BlockedReasons[0] != "missing_agent_role:remote_worker" {
		t.Fatalf("unexpected blocked project: %+v", preview.Projects[1])
	}
}

func TestWorkerPoolReadinessArraysEncodeAsEmptyArrays(t *testing.T) {
	generated := time.Date(2026, 6, 29, 4, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		workerPool: project.WorkerPoolSummary{
			GeneratedAt:   generated,
			TotalProjects: 1,
			Projects: []project.WorkerPoolProjectSummary{{
				Project: project.Record{
					ID:              1,
					Key:             "areamatrix",
					Name:            "AreaMatrix",
					Adapter:         "areamatrix",
					WorkflowProfile: "areamatrix",
				},
				Role:      project.RoleReadiness{Status: "ready"},
				Engine:    project.EngineReadiness{Status: "ready"},
				Resources: project.ResourceReadiness{Status: "ready"},
			}},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/worker-pool/summary", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("worker pool summary status = %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, expected := range []string{
		`"capabilities":[]`,
		`"worker_types":[]`,
		`"required_capabilities":[]`,
		`"matched_types":[]`,
		`"blocked_reasons":[]`,
		`"resource_limits":{}`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("summary response missing %s: %s", expected, body)
		}
	}
}

func TestProjectWorkerLeaseEndpoints(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 10, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	released := created.Add(time.Minute)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	lease := project.LeaseRecord{
		ID:                  9,
		ProjectID:           1,
		RunID:               3,
		RunTaskID:           4,
		WorkerID:            7,
		LeaseKind:           "run_task",
		Status:              "active",
		AcquiredAt:          created,
		ExpiresAt:           expires,
		HeartbeatAt:         &created,
		AllowedCapabilities: []string{"read_project"},
		Scope:               map[string]any{"run_task_id": float64(4)},
		Metadata:            map[string]any{"dry_run": true},
	}
	releasedLease := lease
	releasedLease.Status = "released"
	releasedLease.ReleasedAt = &released
	var acquireOptions project.AcquireLeaseOptions
	var releaseOptions project.ReleaseLeaseOptions
	var recoverOptions project.RecoverLeasesOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		lease:  lease,
		leases: []project.LeaseRecord{releasedLease},
		leaseAcquireHook: func(options project.AcquireLeaseOptions) {
			acquireOptions = options
		},
		leaseReleaseHook: func(options project.ReleaseLeaseOptions) {
			releaseOptions = options
		},
		leaseRecoverHook: func(options project.RecoverLeasesOptions) {
			recoverOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/local-1/lease-acquire", strings.NewReader(`{
		"run_task_id": 4,
		"allowed_capabilities": ["read_project"],
		"idempotency_key": "lease-acquire-4"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("lease acquire status = %d body=%s", resp.Code, resp.Body.String())
	}
	var acquired leaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&acquired); err != nil {
		t.Fatalf("decode lease acquire response: %v", err)
	}
	if acquired.ID != 9 || acquired.RunTaskID != 4 || acquired.Status != "active" {
		t.Fatalf("unexpected lease acquire response: %+v", acquired)
	}
	if acquireOptions.IdempotencyKey != "lease-acquire-4" {
		t.Fatalf("lease acquire idempotency key = %q", acquireOptions.IdempotencyKey)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/local-1/lease-release", strings.NewReader(`{
		"lease_id": 9,
		"status": "released",
		"idempotency_key": "lease-release-9"
	}`))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("lease release status = %d body=%s", resp.Code, resp.Body.String())
	}
	if releaseOptions.IdempotencyKey != "lease-release-9" {
		t.Fatalf("lease release idempotency key = %q", releaseOptions.IdempotencyKey)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/lease-recover", strings.NewReader(`{
		"limit": 3,
		"idempotency_key": "lease-recover-3"
	}`))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("lease recover status = %d body=%s", resp.Code, resp.Body.String())
	}
	var recovered leaseRecoverResponse
	if err := json.NewDecoder(resp.Body).Decode(&recovered); err != nil {
		t.Fatalf("decode lease recover response: %v", err)
	}
	if len(recovered.Leases) != 1 || recovered.Leases[0].Status != "released" {
		t.Fatalf("unexpected lease recover response: %+v", recovered)
	}
	if recoverOptions.IdempotencyKey != "lease-recover-3" {
		t.Fatalf("lease recover idempotency key = %q", recoverOptions.IdempotencyKey)
	}
}

func TestProjectWorkerLeaseAcquireCapabilityDenied(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		record:   project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		leaseErr: project.ErrWorkerCapabilityDenied,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/local-1/lease-acquire", strings.NewReader(`{
		"run_task_id": 4,
		"allowed_capabilities": ["write_artifacts"]
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s, want 403", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "worker capability denied") {
		t.Fatalf("unexpected error body: %s", resp.Body.String())
	}
}

func TestProjectWorkerRunOnceEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 20, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	var capturedOptions project.WorkerRunOnceOptions
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	worker := project.WorkerRecord{
		ID:                       7,
		ProjectID:                1,
		WorkerKey:                "local-1",
		WorkerType:               "local_host",
		Status:                   "online",
		Capabilities:             []string{"read_project"},
		Metadata:                 map[string]any{},
		RegisteredAt:             created,
		LastHeartbeatAt:          &created,
		HeartbeatIntervalSeconds: 30,
		LeaseTimeoutSeconds:      300,
		UpdatedAt:                created,
	}
	lease := project.LeaseRecord{
		ID:                  9,
		ProjectID:           1,
		RunID:               3,
		RunTaskID:           4,
		WorkerID:            7,
		LeaseKind:           "run_task",
		Status:              "completed",
		AcquiredAt:          created,
		ExpiresAt:           expires,
		AllowedCapabilities: []string{"read_project"},
		Scope:               map[string]any{"run_once": true},
		Metadata:            map[string]any{"dry_run": true},
	}
	task := project.RunTaskRecord{
		ID:                4,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunID:             3,
		TaskKey:           "v2:runner-preview",
		TaskKind:          "workflow_item_preview",
		Status:            "passed",
		RiskLevel:         "low",
		Sequence:          1,
		Metadata:          map[string]any{"dry_run": true},
		CreatedAt:         created,
		UpdatedAt:         created,
	}
	attempt := project.RunAttemptRecord{
		ID:                10,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunID:             3,
		RunTaskID:         4,
		AttemptKind:       "worker_run_once",
		Status:            "passed",
		DryRun:            true,
		Metadata:          map[string]any{"would_execute": false},
		StartedAt:         created,
	}
	artifact := project.ArtifactRecord{
		ID:                11,
		ProjectID:         1,
		WorkflowVersionID: 2,
		ArtifactType:      "worker_run_once_report",
		StorageBackend:    "local",
		URI:               "/tmp/areaflow/artifacts/areamatrix/workers/local-1/run-once/run-task-4-report.json",
		SourcePath:        "workers/local-1/run-once/run-task-4-report.json",
		SHA256:            "def456",
		SizeBytes:         256,
		ContentType:       "application/json",
		Metadata:          map[string]any{"dry_run": true},
		CreatedAt:         created,
	}
	handler := NewHandler(fakeProjectStore{
		record: record,
		runOnceHook: func(options project.WorkerRunOnceOptions) {
			capturedOptions = options
		},
		runOnce: project.WorkerRunOnceResult{
			Project:  record,
			Worker:   worker,
			Lease:    lease,
			Task:     task,
			Attempt:  attempt,
			Artifact: artifact,
			Claimed:  true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/local-1/run-once", strings.NewReader(`{
		"run_id": 3,
		"allowed_capabilities": ["read_project"]
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("run-once status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workerRunOnceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode run-once response: %v", err)
	}
	if !body.Claimed || body.Lease == nil || body.Task == nil || body.Attempt == nil || body.Artifact == nil {
		t.Fatalf("unexpected run-once response: %+v", body)
	}
	if body.Lease.Status != "completed" || body.Task.ID != 4 {
		t.Fatalf("unexpected run-once lease/task: %+v", body)
	}
	if body.Attempt.AttemptKind != "worker_run_once" || !body.Attempt.DryRun {
		t.Fatalf("unexpected run-once attempt: %+v", body.Attempt)
	}
	if body.Artifact.ArtifactType != "worker_run_once_report" || body.Artifact.SHA256 != "def456" {
		t.Fatalf("unexpected run-once artifact: %+v", body.Artifact)
	}
	if capturedOptions.RunID != 3 {
		t.Fatalf("run id = %d, want 3", capturedOptions.RunID)
	}
	if len(capturedOptions.AllowedCapabilities) != 1 || capturedOptions.AllowedCapabilities[0] != "read_project" {
		t.Fatalf("unexpected captured capabilities: %+v", capturedOptions.AllowedCapabilities)
	}
}

func TestProjectWorkerRunOnceCapabilityDenied(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		record:     project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		runOnceErr: project.ErrWorkerCapabilityDenied,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workers/local-1/run-once", strings.NewReader(`{
		"allowed_capabilities": ["write_artifacts"]
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s, want 403", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "worker capability denied") {
		t.Fatalf("unexpected error body: %s", resp.Body.String())
	}
}

func TestProjectWorkerFixtureExecuteEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 1, 0, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "fixture_execution", RunKind: "execution", Status: "passed", DryRun: false, StartedAt: created}
	worker := project.WorkerRecord{ID: 4, ProjectID: 1, WorkerKey: "local-1", WorkerType: "local_host", Status: "online", Capabilities: []string{"execute_agents", "read_project", "run_commands", "write_artifacts"}, RegisteredAt: created, UpdatedAt: created}
	lease := project.LeaseRecord{ID: 5, ProjectID: 1, RunID: 3, RunTaskID: 6, WorkerID: 4, LeaseKind: "fixture_execution", Status: "completed", AcquiredAt: created, ExpiresAt: expires, AllowedCapabilities: []string{"read_project"}, Scope: map[string]any{"fixture_only": true}}
	task := project.RunTaskRecord{ID: 6, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:fixture-execution", TaskKind: "fixture_execution_task", Status: "passed", RiskLevel: "low", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	attempt := project.RunAttemptRecord{ID: 7, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "fixture_execution", Status: "passed", DryRun: false, StartedAt: created}
	artifact := project.ArtifactRecord{ID: 8, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "fixture_execution_report", StorageBackend: "local", URI: "/tmp/artifacts/report.json", SourcePath: "versions/v2/fixture-execution/report.json", SHA256: "abc123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.FixtureExecutionOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		fixtureExecuteHook: func(options project.FixtureExecutionOptions) {
			capturedOptions = options
		},
		fixtureExecute: project.FixtureExecutionResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Worker:                        worker,
			Lease:                         lease,
			Task:                          task,
			Attempt:                       attempt,
			Artifact:                      artifact,
			Gate:                          project.ExecutionApprovalGate{Project: record, Version: version, Run: run, Status: "pass", Mode: "read_only_execution_approval_gate"},
			Status:                        "passed",
			Decision:                      "allowed",
			Message:                       "fixture execution applied in AreaFlow state only",
			Created:                       true,
			IdempotencyKey:                "fixture-exec-key",
			EventID:                       9,
			AuditEventID:                  10,
			AreaFlowExecutionStateWritten: true,
			TaskClaimed:                   true,
			LeaseCreated:                  true,
			AttemptCreated:                true,
			ArtifactCreated:               true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workers/local-1/fixture-execute", strings.NewReader(`{
		"run_id": 3,
		"allowed_capabilities": ["read_project"],
		"lease_timeout_seconds": 120,
		"idempotency_key": "fixture-exec-key",
		"actor": "local-user",
		"reason": "fixture execute"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("fixture execute status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.WorkerKey != "local-1" || capturedOptions.RunID != 3 || capturedOptions.IdempotencyKey != "fixture-exec-key" {
		t.Fatalf("unexpected fixture execution options: %+v", capturedOptions)
	}
	var body fixtureExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode fixture execution response: %v", err)
	}
	if body.Status != "passed" || body.Decision != "allowed" || body.Run.ID != 3 || body.Attempt.AttemptKind != "fixture_execution" || body.Artifact.ArtifactType != "fixture_execution_report" {
		t.Fatalf("unexpected fixture execution response: %+v", body)
	}
	if body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowExecutionStateWritten || !body.TaskClaimed || !body.LeaseCreated || !body.AttemptCreated || !body.ArtifactCreated {
		t.Fatalf("unexpected fixture execution safety facts: %+v", body)
	}
}

func TestProjectWorkerReadOnlyVerifyEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 2, 0, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "read_only_verify", RunKind: "execution", Status: "verified", DryRun: false, StartedAt: created}
	worker := project.WorkerRecord{ID: 4, ProjectID: 1, WorkerKey: "local-1", WorkerType: "local_host", Status: "online", Capabilities: []string{"read_project", "write_artifacts"}, RegisteredAt: created, UpdatedAt: created}
	lease := project.LeaseRecord{ID: 5, ProjectID: 1, RunID: 3, RunTaskID: 6, WorkerID: 4, LeaseKind: "read_only_verify", Status: "completed", AcquiredAt: created, ExpiresAt: expires, AllowedCapabilities: []string{"read_project", "write_artifacts"}, Scope: map[string]any{"read_only_verify": true}}
	task := project.RunTaskRecord{ID: 6, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:read-only-verify:docs/README.md", TaskKind: "read_only_verify_task", Status: "verified", RiskLevel: "low", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	attempt := project.RunAttemptRecord{ID: 7, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "read_only_verify", Status: "passed", DryRun: false, StartedAt: created}
	artifact := project.ArtifactRecord{ID: 8, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "read_only_verify_report", StorageBackend: "local", URI: "/tmp/artifacts/read-only-report.json", SourcePath: "versions/v2/read-only-verify/report.json", SHA256: "report123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.ReadOnlyVerifyOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		readOnlyVerifyHook: func(options project.ReadOnlyVerifyOptions) {
			capturedOptions = options
		},
		readOnlyVerify: project.ReadOnlyVerifyResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Worker:                        worker,
			Lease:                         lease,
			Task:                          task,
			Attempt:                       attempt,
			Artifact:                      artifact,
			Gate:                          project.ExecutionApprovalGate{Project: record, Version: version, Run: run, Status: "pass", Mode: "read_only_verify_gate"},
			TargetPath:                    "docs/README.md",
			TargetSHA256:                  "abc123",
			TargetSizeBytes:               64,
			Status:                        "verified",
			Decision:                      "allowed",
			Message:                       "read-only verify completed without managed project writes",
			Created:                       true,
			IdempotencyKey:                "read-only-exec-key",
			EventID:                       9,
			AuditEventID:                  10,
			ProjectReadAttempted:          true,
			ProjectReadAllowed:            true,
			AreaFlowExecutionStateWritten: true,
			TaskClaimed:                   true,
			LeaseCreated:                  true,
			AttemptCreated:                true,
			ArtifactCreated:               true,
			VerificationPassed:            true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workers/local-1/read-only-verify", strings.NewReader(`{
		"run_id": 3,
		"allowed_capabilities": ["read_project", "write_artifacts"],
		"lease_timeout_seconds": 120,
		"idempotency_key": "read-only-exec-key",
		"actor": "local-user",
		"reason": "verify docs"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("read-only verify status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.WorkerKey != "local-1" || capturedOptions.RunID != 3 || capturedOptions.IdempotencyKey != "read-only-exec-key" {
		t.Fatalf("unexpected read-only verify options: %+v", capturedOptions)
	}
	if capturedOptions.LeaseTimeoutSeconds != 120 || len(capturedOptions.AllowedCapabilities) != 2 {
		t.Fatalf("unexpected read-only verify lease/caps: %+v", capturedOptions)
	}
	var body readOnlyVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode read-only verify response: %v", err)
	}
	if body.Status != "verified" || body.Decision != "allowed" || body.Run.ID != 3 || body.Attempt.AttemptKind != "read_only_verify" || body.Artifact.ArtifactType != "read_only_verify_report" {
		t.Fatalf("unexpected read-only verify response: %+v", body)
	}
	if !body.ProjectReadAttempted || !body.ProjectReadAllowed || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowExecutionStateWritten || !body.TaskClaimed || !body.LeaseCreated || !body.AttemptCreated || !body.ArtifactCreated || !body.VerificationPassed {
		t.Fatalf("unexpected read-only verify safety facts: %+v", body)
	}
}

func TestProjectWorkerApprovedArtifactWriteEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 3, 0, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "approved_artifact_write", RunKind: "execution", Status: "artifact_written", DryRun: false, StartedAt: created}
	worker := project.WorkerRecord{ID: 4, ProjectID: 1, WorkerKey: "local-1", WorkerType: "local_host", Status: "online", Capabilities: []string{"write_artifacts"}, RegisteredAt: created, UpdatedAt: created}
	lease := project.LeaseRecord{ID: 5, ProjectID: 1, RunID: 3, RunTaskID: 6, WorkerID: 4, LeaseKind: "approved_artifact_write", Status: "completed", AcquiredAt: created, ExpiresAt: expires, AllowedCapabilities: []string{"write_artifacts"}, Scope: map[string]any{"approved_artifact_write": true}}
	task := project.RunTaskRecord{ID: 6, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:approved-artifact-write:approval-note", TaskKind: "approved_artifact_write_task", Status: "artifact_written", RiskLevel: "low", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	attempt := project.RunAttemptRecord{ID: 7, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "approved_artifact_write", Status: "passed", DryRun: false, StartedAt: created}
	artifact := project.ArtifactRecord{ID: 8, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "approved_artifact_write_report", StorageBackend: "local", URI: "/tmp/artifacts/approved-artifact.json", SourcePath: "versions/v2/approved-artifact-write/report.json", SHA256: "report123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.ApprovedArtifactWriteOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		artifactWriteHook: func(options project.ApprovedArtifactWriteOptions) {
			capturedOptions = options
		},
		artifactWrite: project.ApprovedArtifactWriteResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Worker:                        worker,
			Lease:                         lease,
			Task:                          task,
			Attempt:                       attempt,
			Artifact:                      artifact,
			Gate:                          project.ExecutionApprovalGate{Project: record, Version: version, Run: run, Status: "pass", Mode: "approved_artifact_write_gate"},
			ArtifactLabel:                 "approval-note",
			Status:                        "artifact_written",
			Decision:                      "allowed",
			Message:                       "approved artifact write completed in AreaFlow artifact store only",
			Created:                       true,
			IdempotencyKey:                "artifact-write-key",
			EventID:                       9,
			AuditEventID:                  10,
			AreaFlowArtifactWritten:       true,
			AreaFlowExecutionStateWritten: true,
			TaskClaimed:                   true,
			LeaseCreated:                  true,
			AttemptCreated:                true,
			ArtifactCreated:               true,
			ArtifactWritePassed:           true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workers/local-1/approved-artifact-write", strings.NewReader(`{
		"run_id": 3,
		"allowed_capabilities": ["write_artifacts"],
		"lease_timeout_seconds": 120,
		"idempotency_key": "artifact-write-key",
		"actor": "local-user",
		"reason": "write approved artifact"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("approved artifact write status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.WorkerKey != "local-1" || capturedOptions.RunID != 3 || capturedOptions.IdempotencyKey != "artifact-write-key" {
		t.Fatalf("unexpected approved artifact write options: %+v", capturedOptions)
	}
	var body approvedArtifactWriteResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode approved artifact write response: %v", err)
	}
	if body.Status != "artifact_written" || body.Decision != "allowed" || body.Run.ID != 3 || body.Attempt.AttemptKind != "approved_artifact_write" || body.Artifact.ArtifactType != "approved_artifact_write_report" {
		t.Fatalf("unexpected approved artifact write response: %+v", body)
	}
	if body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowArtifactWritten || !body.AreaFlowExecutionStateWritten || !body.TaskClaimed || !body.LeaseCreated || !body.AttemptCreated || !body.ArtifactCreated || !body.ArtifactWritePassed {
		t.Fatalf("unexpected approved artifact write safety facts: %+v", body)
	}
}

func TestProjectWorkerFixtureProjectWriteEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 4, 0, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	record := project.Record{ID: 1, Key: "areamatrix-fixture", Name: "AreaMatrix Fixture", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "fixture_project_write", RunKind: "execution", Status: "rollback_verified", DryRun: false, StartedAt: created}
	worker := project.WorkerRecord{ID: 4, ProjectID: 1, WorkerKey: "local-1", WorkerType: "local_host", Status: "online", Capabilities: []string{"read_project", "write_artifacts", "write_code"}, RegisteredAt: created, UpdatedAt: created}
	lease := project.LeaseRecord{ID: 5, ProjectID: 1, RunID: 3, RunTaskID: 6, WorkerID: 4, LeaseKind: "fixture_project_write", Status: "completed", AcquiredAt: created, ExpiresAt: expires, AllowedCapabilities: []string{"read_project", "write_artifacts", "write_code"}, Scope: map[string]any{"fixture_project_write": true}}
	task := project.RunTaskRecord{ID: 6, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:fixture-project-write:fixtures/input.txt", TaskKind: "fixture_project_write_task", Status: "rollback_verified", RiskLevel: "medium", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	copyAttempt := project.RunAttemptRecord{ID: 7, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "copy", Status: "passed", DryRun: false, StartedAt: created}
	verifyAttempt := project.RunAttemptRecord{ID: 8, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "verify", Status: "passed", DryRun: false, StartedAt: created}
	rollbackAttempt := project.RunAttemptRecord{ID: 9, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "rollback", Status: "passed", DryRun: false, StartedAt: created}
	writeSetArtifact := project.ArtifactRecord{ID: 10, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "fixture_project_write_set", StorageBackend: "local", URI: "/tmp/artifacts/write-set.json", SourcePath: "versions/v2/fixture-project-write/write-set.json", SHA256: "writeset123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	preimageArtifact := project.ArtifactRecord{ID: 11, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "fixture_project_write_preimage", StorageBackend: "local", URI: "/tmp/artifacts/preimage.bin", SourcePath: "versions/v2/fixture-project-write/preimage.bin", SHA256: "before123", SizeBytes: 12, ContentType: "application/octet-stream", CreatedAt: created}
	reportArtifact := project.ArtifactRecord{ID: 12, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "fixture_project_write_report", StorageBackend: "local", URI: "/tmp/artifacts/fixture-project-write.json", SourcePath: "versions/v2/fixture-project-write/report.json", SHA256: "report123", SizeBytes: 256, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.FixtureProjectWriteOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		fixtureProjectWriteHook: func(options project.FixtureProjectWriteOptions) {
			capturedOptions = options
		},
		fixtureProjectWrite: project.FixtureProjectWriteResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Worker:                        worker,
			Lease:                         lease,
			Task:                          task,
			CopyAttempt:                   copyAttempt,
			VerifyAttempt:                 verifyAttempt,
			RollbackAttempt:               rollbackAttempt,
			WriteSetArtifact:              writeSetArtifact,
			PreimageArtifact:              preimageArtifact,
			Artifact:                      reportArtifact,
			Gate:                          project.ExecutionApprovalGate{Project: record, Version: version, Run: run, Status: "pass", Mode: "fixture_project_write_gate"},
			TargetPath:                    "fixtures/input.txt",
			ExpectedBeforeSHA256:          "before123",
			ExpectedBeforeSize:            12,
			AfterSHA256:                   "after123",
			AfterSize:                     13,
			RestoredSHA256:                "before123",
			RestoredSize:                  12,
			Status:                        "rollback_verified",
			Decision:                      "allowed",
			Message:                       "fixture project write verified and rolled back",
			Created:                       true,
			IdempotencyKey:                "fixture-project-write-key",
			EventID:                       13,
			AuditEventID:                  14,
			ProjectReadAttempted:          true,
			ProjectReadAllowed:            true,
			ProjectWriteAttempted:         true,
			ProjectWriteAllowed:           true,
			AreaFlowArtifactWritten:       true,
			AreaFlowExecutionStateWritten: true,
			TaskClaimed:                   true,
			LeaseCreated:                  true,
			AttemptCreated:                true,
			ArtifactCreated:               true,
			WriteSetPassed:                true,
			VerificationPassed:            true,
			RollbackAttempted:             true,
			RollbackVerified:              true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix-fixture/workers/local-1/fixture-project-write", strings.NewReader(`{
		"run_id": 3,
		"allowed_capabilities": ["read_project", "write_artifacts", "write_code"],
		"lease_timeout_seconds": 120,
		"idempotency_key": "fixture-project-write-key",
		"actor": "local-user",
		"reason": "write fixture project file"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("fixture project write status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.WorkerKey != "local-1" || capturedOptions.RunID != 3 || capturedOptions.IdempotencyKey != "fixture-project-write-key" {
		t.Fatalf("unexpected fixture project write options: %+v", capturedOptions)
	}
	if capturedOptions.LeaseTimeoutSeconds != 120 || len(capturedOptions.AllowedCapabilities) != 3 {
		t.Fatalf("unexpected fixture project write lease/caps: %+v", capturedOptions)
	}
	var body fixtureProjectWriteResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode fixture project write response: %v", err)
	}
	if body.Status != "rollback_verified" || body.Decision != "allowed" || body.Run.ID != 3 || body.CopyAttempt.AttemptKind != "copy" || body.VerifyAttempt.AttemptKind != "verify" || body.RollbackAttempt.AttemptKind != "rollback" || body.Artifact.ArtifactType != "fixture_project_write_report" {
		t.Fatalf("unexpected fixture project write response: %+v", body)
	}
	if !body.ProjectReadAttempted || !body.ProjectReadAllowed || !body.ProjectWriteAttempted || !body.ProjectWriteAllowed || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowArtifactWritten || !body.AreaFlowExecutionStateWritten || !body.TaskClaimed || !body.LeaseCreated || !body.AttemptCreated || !body.ArtifactCreated || !body.WriteSetPassed || !body.VerificationPassed || !body.RollbackAttempted || !body.RollbackVerified {
		t.Fatalf("unexpected fixture project write safety facts: %+v", body)
	}
}

func TestProjectWorkerManagedGeneratedWriteEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	record := project.Record{ID: 1, Key: "areamatrix-fixture", Name: "AreaMatrix Fixture", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "managed_generated_write", RunKind: "execution", Status: "rollback_verified", DryRun: false, StartedAt: created}
	worker := project.WorkerRecord{ID: 4, ProjectID: 1, WorkerKey: "local-1", WorkerType: "local_host", Status: "online", Capabilities: []string{"read_project", "write_artifacts", "write_generated"}, RegisteredAt: created, UpdatedAt: created}
	lease := project.LeaseRecord{ID: 5, ProjectID: 1, RunID: 3, RunTaskID: 6, WorkerID: 4, LeaseKind: "managed_generated_write", Status: "completed", AcquiredAt: created, ExpiresAt: expires, AllowedCapabilities: []string{"read_project", "write_artifacts", "write_generated"}, Scope: map[string]any{"managed_generated_write": true}}
	task := project.RunTaskRecord{ID: 6, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:managed-generated-write:.areaflow/generated/status.json", TaskKind: "managed_generated_write_task", Status: "rollback_verified", RiskLevel: "medium", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	copyAttempt := project.RunAttemptRecord{ID: 7, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "copy", Status: "passed", DryRun: false, StartedAt: created}
	verifyAttempt := project.RunAttemptRecord{ID: 8, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "verify", Status: "passed", DryRun: false, StartedAt: created}
	rollbackAttempt := project.RunAttemptRecord{ID: 9, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 6, AttemptKind: "rollback", Status: "passed", DryRun: false, StartedAt: created}
	writeSetArtifact := project.ArtifactRecord{ID: 10, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "managed_generated_write_set", StorageBackend: "local", URI: "/tmp/artifacts/write-set.json", SourcePath: "versions/v2/managed-generated-write/write-set.json", SHA256: "writeset123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	preimageArtifact := project.ArtifactRecord{ID: 11, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "managed_generated_write_preimage", StorageBackend: "local", URI: "/tmp/artifacts/preimage.bin", SourcePath: "versions/v2/managed-generated-write/preimage.bin", SHA256: "before123", SizeBytes: 12, ContentType: "application/octet-stream", CreatedAt: created}
	reportArtifact := project.ArtifactRecord{ID: 12, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "managed_generated_write_report", StorageBackend: "local", URI: "/tmp/artifacts/managed-generated-write.json", SourcePath: "versions/v2/managed-generated-write/report.json", SHA256: "report123", SizeBytes: 256, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.ManagedGeneratedWriteOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		managedGeneratedWriteHook: func(options project.ManagedGeneratedWriteOptions) {
			capturedOptions = options
		},
		managedGeneratedWrite: project.ManagedGeneratedWriteResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Worker:                        worker,
			Lease:                         lease,
			Task:                          task,
			CopyAttempt:                   copyAttempt,
			VerifyAttempt:                 verifyAttempt,
			RollbackAttempt:               rollbackAttempt,
			WriteSetArtifact:              writeSetArtifact,
			PreimageArtifact:              preimageArtifact,
			Artifact:                      reportArtifact,
			Gate:                          project.ExecutionApprovalGate{Project: record, Version: version, Run: run, Status: "pass", Mode: "managed_generated_write_gate"},
			TargetPath:                    ".areaflow/generated/status.json",
			ExpectedBeforeSHA256:          "before123",
			ExpectedBeforeSize:            12,
			AfterSHA256:                   "after123",
			AfterSize:                     13,
			RestoredSHA256:                "before123",
			RestoredSize:                  12,
			Status:                        "rollback_verified",
			Decision:                      "allowed",
			Message:                       "managed generated write verified and rolled back in fixture/temp project",
			Created:                       true,
			IdempotencyKey:                "managed-generated-write-key",
			EventID:                       13,
			AuditEventID:                  14,
			GeneratedOnly:                 true,
			GeneratedOnlyApplyOpen:        true,
			ProjectReadAttempted:          true,
			ProjectReadAllowed:            true,
			ProjectWriteAttempted:         true,
			ProjectWriteAllowed:           true,
			AreaFlowArtifactWritten:       true,
			AreaFlowExecutionStateWritten: true,
			TaskClaimed:                   true,
			LeaseCreated:                  true,
			AttemptCreated:                true,
			ArtifactCreated:               true,
			WriteSetPassed:                true,
			VerificationPassed:            true,
			RollbackAttempted:             true,
			RollbackVerified:              true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix-fixture/workers/local-1/managed-generated-write", strings.NewReader(`{
		"run_id": 3,
		"allowed_capabilities": ["read_project", "write_artifacts", "write_generated"],
		"lease_timeout_seconds": 120,
		"idempotency_key": "managed-generated-write-key",
		"actor": "local-user",
		"reason": "write generated fixture file"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("managed generated write status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.WorkerKey != "local-1" || capturedOptions.RunID != 3 || capturedOptions.IdempotencyKey != "managed-generated-write-key" {
		t.Fatalf("unexpected managed generated write options: %+v", capturedOptions)
	}
	if capturedOptions.LeaseTimeoutSeconds != 120 || len(capturedOptions.AllowedCapabilities) != 3 || capturedOptions.AllowedCapabilities[2] != "write_generated" {
		t.Fatalf("unexpected managed generated write lease/caps: %+v", capturedOptions)
	}
	var body managedGeneratedWriteResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode managed generated write response: %v", err)
	}
	if body.Status != "rollback_verified" || body.Decision != "allowed" || body.Run.ID != 3 || body.CopyAttempt.AttemptKind != "copy" || body.VerifyAttempt.AttemptKind != "verify" || body.RollbackAttempt.AttemptKind != "rollback" || body.Artifact.ArtifactType != "managed_generated_write_report" {
		t.Fatalf("unexpected managed generated write response: %+v", body)
	}
	if !body.GeneratedOnly || !body.GeneratedOnlyApplyOpen || !body.ProjectReadAttempted || !body.ProjectReadAllowed || !body.ProjectWriteAttempted || !body.ProjectWriteAllowed || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowArtifactWritten || !body.AreaFlowExecutionStateWritten || !body.TaskClaimed || !body.LeaseCreated || !body.AttemptCreated || !body.ArtifactCreated || !body.WriteSetPassed || !body.VerificationPassed || !body.RollbackAttempted || !body.RollbackVerified {
		t.Fatalf("unexpected managed generated write safety facts: %+v", body)
	}
}

func TestProjectWorkerEndpointsUseAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 20, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	released := created.Add(time.Minute)
	var capturedOptions project.WorkerRunOnceOptions
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	worker := project.WorkerRecord{
		ID:                       7,
		ProjectID:                1,
		WorkerKey:                "local-1",
		WorkerType:               "local_host",
		Status:                   "online",
		Hostname:                 "host-a",
		Capabilities:             []string{"read_project"},
		Metadata:                 map[string]any{"dry_run": true},
		RegisteredAt:             created,
		LastHeartbeatAt:          &created,
		HeartbeatIntervalSeconds: 30,
		LeaseTimeoutSeconds:      300,
		UpdatedAt:                created,
	}
	lease := project.LeaseRecord{
		ID:                  9,
		ProjectID:           1,
		RunID:               3,
		RunTaskID:           4,
		WorkerID:            7,
		LeaseKind:           "run_task",
		Status:              "active",
		AcquiredAt:          created,
		ExpiresAt:           expires,
		HeartbeatAt:         &created,
		AllowedCapabilities: []string{"read_project"},
		Scope:               map[string]any{"run_task_id": float64(4)},
		Metadata:            map[string]any{"dry_run": true},
	}
	releasedLease := lease
	releasedLease.Status = "released"
	releasedLease.ReleasedAt = &released
	runOnceLease := lease
	runOnceLease.Status = "completed"
	task := project.RunTaskRecord{
		ID:                4,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunID:             3,
		TaskKey:           "v2:runner-preview",
		TaskKind:          "workflow_item_preview",
		Status:            "passed",
		RiskLevel:         "low",
		Sequence:          1,
		Metadata:          map[string]any{"dry_run": true},
		CreatedAt:         created,
		UpdatedAt:         created,
	}
	attempt := project.RunAttemptRecord{
		ID:                10,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunID:             3,
		RunTaskID:         4,
		AttemptKind:       "worker_run_once",
		Status:            "passed",
		DryRun:            true,
		Metadata:          map[string]any{"would_execute": false},
		StartedAt:         created,
	}
	artifact := project.ArtifactRecord{
		ID:                11,
		ProjectID:         1,
		WorkflowVersionID: 2,
		ArtifactType:      "worker_run_once_report",
		StorageBackend:    "local",
		URI:               "/tmp/areaflow/artifacts/areamatrix/workers/local-1/run-once/run-task-4-report.json",
		SourcePath:        "workers/local-1/run-once/run-task-4-report.json",
		SHA256:            "def456",
		SizeBytes:         256,
		ContentType:       "application/json",
		Metadata:          map[string]any{"dry_run": true},
		CreatedAt:         created,
	}
	handler := NewHandler(fakeProjectStore{
		record:  record,
		worker:  worker,
		workers: []project.WorkerRecord{worker},
		lease:   lease,
		leases:  []project.LeaseRecord{releasedLease},
		runOnceHook: func(options project.WorkerRunOnceOptions) {
			capturedOptions = options
		},
		runOnce: project.WorkerRunOnceResult{
			Project:  record,
			Worker:   worker,
			Lease:    runOnceLease,
			Task:     task,
			Attempt:  attempt,
			Artifact: artifact,
			Claimed:  true,
		},
	})

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantField  string
	}{
		{name: "register", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workers", body: `{"worker_key":"local-1","worker_type":"local_host","capabilities":["read_project"]}`, wantStatus: http.StatusCreated, wantField: "worker_key"},
		{name: "list", method: http.MethodGet, path: "/api/v1/projects/areamatrix/workers?limit=1", wantStatus: http.StatusOK, wantField: "workers"},
		{name: "heartbeat", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workers/local-1/heartbeat", body: `{"status":"online"}`, wantStatus: http.StatusOK, wantField: "worker_key"},
		{name: "lease acquire", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workers/local-1/lease-acquire", body: `{"run_task_id":4,"allowed_capabilities":["read_project"]}`, wantStatus: http.StatusCreated, wantField: "run_task_id"},
		{name: "lease release", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workers/local-1/lease-release", body: `{"lease_id":9,"status":"released"}`, wantStatus: http.StatusOK, wantField: "status"},
		{name: "lease recover", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workers/lease-recover", body: `{"limit":3}`, wantStatus: http.StatusOK, wantField: "leases"},
		{name: "run once", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workers/local-1/run-once", body: `{"run_id":3,"allowed_capabilities":["read_project"]}`, wantStatus: http.StatusOK, wantField: "claimed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if resp.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d body=%s", tt.method, tt.path, resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode %s response: %v", tt.path, err)
			}
			if _, ok := body[tt.wantField]; !ok {
				t.Fatalf("%s response missing %q: %+v", tt.path, tt.wantField, body)
			}
		})
	}
	if capturedOptions.RunID != 3 {
		t.Fatalf("run-once run_id = %d, want 3", capturedOptions.RunID)
	}

	denied := NewHandler(fakeProjectStore{
		record:     record,
		runOnceErr: project.ErrWorkerCapabilityDenied,
	})
	resp := httptest.NewRecorder()
	denied.ServeHTTP(resp, httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workers/local-1/run-once", strings.NewReader(`{
		"allowed_capabilities": ["write_artifacts"]
	}`)))
	if resp.Code != http.StatusForbidden || !strings.Contains(resp.Body.String(), "worker capability denied") {
		t.Fatalf("capability denied response = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRunDetailEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 0, 0, 0, time.UTC)
	finished := created.Add(time.Second)
	handler := NewHandler(fakeProjectStore{
		runDetail: project.RunDetail{
			Run: project.RunRecord{
				ID:                3,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunType:           "runner_preview",
				RunKind:           "execution",
				Status:            "passed",
				RiskLevel:         "low",
				RiskPolicy:        "pause",
				DryRun:            true,
				Summary:           map[string]any{"task_count": float64(1)},
				Metadata:          map[string]any{"dry_run": true},
				StartedAt:         created,
				FinishedAt:        &finished,
			},
			Tasks: []project.RunTaskRecord{{
				ID:                4,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunID:             3,
				TaskKey:           "v2:runner-preview",
				TaskKind:          "workflow_item_preview",
				Status:            "passed",
				RiskLevel:         "low",
				Sequence:          1,
				Metadata:          map[string]any{"dry_run": true},
				CreatedAt:         created,
				UpdatedAt:         finished,
			}},
			Attempts: []project.RunAttemptRecord{{
				ID:                5,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunID:             3,
				RunTaskID:         4,
				AttemptKind:       "worker_run_once",
				Status:            "passed",
				DryRun:            true,
				Metadata:          map[string]any{"would_execute": false},
				StartedAt:         created,
				FinishedAt:        &finished,
			}},
			Artifacts: []project.ArtifactRecord{{
				ID:                6,
				ProjectID:         1,
				WorkflowVersionID: 2,
				ArtifactType:      "worker_run_once_report",
				StorageBackend:    "local",
				URI:               "/tmp/areaflow/artifacts/areamatrix/workers/local-1/run-once/run-task-4-report.json",
				SourcePath:        "workers/local-1/run-once/run-task-4-report.json",
				SHA256:            "def456",
				SizeBytes:         256,
				ContentType:       "application/json",
				Metadata:          map[string]any{"dry_run": true},
				CreatedAt:         created,
			}},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/runs/3", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("run detail status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body runDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode run detail: %v", err)
	}
	if body.Run.ID != 3 || !body.Run.DryRun || len(body.Tasks) != 1 || len(body.Attempts) != 1 || len(body.Artifacts) != 1 {
		t.Fatalf("unexpected run detail response: %+v", body)
	}
	if body.Tasks[0].ID != 4 || body.Artifacts[0].ArtifactType != "worker_run_once_report" {
		t.Fatalf("unexpected run detail nested records: %+v", body)
	}
}

func TestExecutionApprovalGateEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 1, 16, 20, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "runner_preview",
		RunKind:           "execution",
		Status:            "passed",
		DryRun:            true,
		StartedAt:         created,
	}
	handler := NewHandler(fakeProjectStore{
		executionGate: project.ExecutionApprovalGate{
			Project:              record,
			Version:              version,
			Run:                  run,
			Status:               "blocked",
			Mode:                 "read_only_execution_approval_gate",
			RequiredCapabilities: []string{"read_project", "write_artifacts", "run_commands", "execute_agents"},
			Items: []project.ReadinessItem{{
				Key:      "dry_run_boundary",
				Status:   "blocked",
				Message:  "dry-run preview runs cannot enter real execution apply",
				Metadata: map[string]any{"dry_run": true},
			}},
			Blockers:         []string{"dry_run_boundary: dry-run preview runs cannot enter real execution apply"},
			ForbiddenActions: []string{"claim_task", "start_worker"},
			EnginePreview: project.CodexCLIAdapterPreview{
				Project: record,
				Status:  "blocked",
				Mode:    "read_only_codex_cli_adapter_preview",
				Engine:  project.EngineReadiness{Status: "blocked"},
			},
			Workers:     []project.WorkerRecord{},
			GeneratedAt: created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/3/execution-approval-gate", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("execution approval gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionApprovalGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution approval gate response: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_execution_approval_gate" || body.Run.ID != 3 || body.WorkflowVersion.DisplayLabel != "v2" {
		t.Fatalf("unexpected execution approval gate response: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "dry_run_boundary" || body.Items[0].Status != "blocked" {
		t.Fatalf("unexpected execution approval gate items: %+v", body.Items)
	}
	if body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.TaskClaimed || body.WorkerStarted || body.AttemptCreated || body.ArtifactCreated {
		t.Fatalf("execution approval gate should be read-only: %+v", body)
	}
}

func TestExecutionPlanEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 5, 20, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "approved_artifact_write",
		RunKind:           "execution",
		Status:            "queued",
		DryRun:            false,
		StartedAt:         created,
	}
	gate := project.ExecutionApprovalGate{
		Project:              record,
		Version:              version,
		Run:                  run,
		Status:               "pass",
		Mode:                 "read_only_execution_approval_gate",
		RequiredCapabilities: []string{"write_artifacts"},
		GeneratedAt:          created,
	}
	handler := NewHandler(fakeProjectStore{
		executionPlan: project.ExecutionPlanPreview{
			Project: record,
			Version: version,
			Run:     run,
			Gate:    gate,
			Status:  "blocked",
			Mode:    "read_only_execution_plan_preview",
			Steps: []project.ExecutionPlanStep{
				{
					Key:                  "copy",
					AttemptKind:          "copy",
					Status:               "blocked",
					Message:              "copy attempt remains closed",
					RequiredCapabilities: []string{"write_code"},
					Blockers:             []string{"copy_apply_not_implemented"},
					ReadsProject:         true,
					WritesProject:        true,
					WritesAreaFlow:       true,
					UsesEngine:           true,
					RunsCommands:         true,
					CreatesAttempt:       true,
					CreatesArtifact:      true,
				},
				{
					Key:                  "approved_artifact_write",
					AttemptKind:          "approved_artifact_write",
					Status:               "ready",
					Message:              "approved artifact write is open",
					RequiredCapabilities: []string{"write_artifacts"},
					WritesAreaFlow:       true,
					CreatesAttempt:       true,
					CreatesArtifact:      true,
				},
			},
			Blockers:         []string{"copy: copy_apply_not_implemented"},
			ForbiddenActions: []string{"write_managed_project", "execute_engine"},
			GeneratedAt:      created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/3/execution-plan", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("execution plan status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionPlanPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution plan response: %v", err)
	}
	if body.Status != "blocked" || body.Mode != "read_only_execution_plan_preview" || body.Run.ID != 3 || body.Gate.Status != "pass" {
		t.Fatalf("unexpected execution plan response: %+v", body)
	}
	if len(body.Steps) != 2 || body.Steps[0].Key != "copy" || body.Steps[0].Status != "blocked" || !body.Steps[0].WritesProject {
		t.Fatalf("unexpected execution plan copy step: %+v", body.Steps)
	}
	if body.Steps[1].Key != "approved_artifact_write" || body.Steps[1].Status != "ready" || body.Steps[1].WritesProject {
		t.Fatalf("unexpected execution plan artifact step: %+v", body.Steps[1])
	}
	if body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted ||
		body.CommandsRun || body.SecretsResolved || body.NetworkUsed || body.TaskClaimed || body.WorkerStarted ||
		body.AttemptCreated || body.ArtifactCreated {
		t.Fatalf("execution plan should be read-only: %+v", body)
	}
}

func TestProjectWriteDesignGateEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 8, 20, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "approved_artifact_write",
		RunKind:           "execution",
		Status:            "queued",
		DryRun:            false,
		StartedAt:         created,
	}
	executionGate := project.ExecutionApprovalGate{
		Project:              record,
		Version:              version,
		Run:                  run,
		Status:               "pass",
		Mode:                 "read_only_project_write_design_gate_execution_approval",
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_code"},
		GeneratedAt:          created,
	}
	handler := NewHandler(fakeProjectStore{
		projectWriteDesignGate: project.ProjectWriteDesignGate{
			Project:              record,
			Version:              version,
			Run:                  run,
			Gate:                 executionGate,
			Status:               "ready",
			Mode:                 "read_only_project_write_design_gate",
			RequiredCapabilities: []string{"read_project", "write_artifacts", "write_code"},
			Items: []project.ReadinessItem{{
				Key:      "write_set_contract",
				Status:   "pass",
				Message:  "future project write apply must start from an approved write-set artifact",
				Metadata: map[string]any{"required_fields": []string{"expected_before_sha256"}},
			}},
			WriteSetFields:                []string{"operation", "target_path", "expected_before_sha256", "rollback_plan_artifact_id"},
			UnsupportedOperations:         []string{"delete", "move", "project_root_escape"},
			ApplySequence:                 []string{"project_write_design_gate", "fixture_approved_project_write", "fixture_rollback_drill", "managed_project_generated_only_write"},
			ForbiddenActions:              []string{"write_managed_project", "execute_engine"},
			ProjectWriteApplyOpen:         false,
			ProjectReadAttempted:          false,
			ProjectWriteAttempted:         false,
			ExecutionWriteAttempted:       false,
			AreaFlowArtifactWritten:       false,
			AreaFlowExecutionStateWritten: false,
			EngineCallAttempted:           false,
			CommandsRun:                   false,
			SecretsResolved:               false,
			NetworkUsed:                   false,
			TaskClaimed:                   false,
			WorkerStarted:                 false,
			AttemptCreated:                false,
			ArtifactCreated:               false,
			GeneratedAt:                   created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/3/project-write-design-gate", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project write design gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectWriteDesignGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode project write design gate response: %v", err)
	}
	if body.Status != "ready" || body.Mode != "read_only_project_write_design_gate" || body.Run.ID != 3 || body.Gate.Status != "pass" {
		t.Fatalf("unexpected project write design gate response: %+v", body)
	}
	if body.ProjectWriteApplyOpen || body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted ||
		body.AreaFlowArtifactWritten || body.AreaFlowExecutionStateWritten || body.EngineCallAttempted ||
		body.CommandsRun || body.SecretsResolved || body.NetworkUsed || body.TaskClaimed || body.WorkerStarted ||
		body.AttemptCreated || body.ArtifactCreated {
		t.Fatalf("project write design gate should be read-only and apply-closed: %+v", body)
	}
	if !containsString(body.WriteSetFields, "expected_before_sha256") || !containsString(body.WriteSetFields, "rollback_plan_artifact_id") {
		t.Fatalf("missing write-set safety fields: %+v", body.WriteSetFields)
	}
	if !containsString(body.UnsupportedOperations, "delete") || !containsString(body.ApplySequence, "fixture_rollback_drill") {
		t.Fatalf("missing destructive-op denial or fixture rollback sequence: %+v", body)
	}
}

func TestManagedGeneratedWriteGateEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 9, 20, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "fixture_project_write",
		RunKind:           "execution",
		Status:            "rollback_verified",
		DryRun:            false,
		StartedAt:         created,
	}
	executionGate := project.ExecutionApprovalGate{
		Project:              record,
		Version:              version,
		Run:                  run,
		Status:               "pass",
		Mode:                 "read_only_managed_generated_write_gate_execution_approval",
		RequiredCapabilities: []string{"read_project", "write_artifacts", "write_generated"},
		GeneratedAt:          created,
	}
	handler := NewHandler(fakeProjectStore{
		managedGeneratedGate: project.ManagedGeneratedWriteGate{
			Project:                  record,
			Version:                  version,
			Run:                      run,
			Gate:                     executionGate,
			Status:                   "ready",
			Mode:                     "read_only_managed_generated_write_gate",
			RequiredCapabilities:     []string{"read_project", "write_artifacts", "write_generated"},
			AllowedGeneratedPrefixes: []string{".areaflow/generated/", ".areamatrix/generated/"},
			Items: []project.ReadinessItem{{
				Key:      "generated_prefix_policy",
				Status:   "pass",
				Message:  "future apply must stay inside generated-only prefixes",
				Metadata: map[string]any{"default_generated_prefixes": []string{".areaflow/generated/"}},
			}},
			RequiredWriteSetFields:        []string{"operation", "target_path", "generated_only", "rollback_plan_artifact_id"},
			UnsupportedOperations:         []string{"source_write", "workflow_execution_write", "delete"},
			ApplySequence:                 []string{"fixture_rollback_drill", "managed_generated_write_gate", "managed_project_generated_only_write"},
			ForbiddenActions:              []string{"write_managed_project", "execute_engine"},
			GeneratedOnlyWriteReady:       true,
			GeneratedOnlyApplyOpen:        false,
			ProjectReadAttempted:          false,
			ProjectWriteAttempted:         false,
			ExecutionWriteAttempted:       false,
			AreaFlowArtifactWritten:       false,
			AreaFlowExecutionStateWritten: false,
			EngineCallAttempted:           false,
			CommandsRun:                   false,
			SecretsResolved:               false,
			NetworkUsed:                   false,
			TaskClaimed:                   false,
			WorkerStarted:                 false,
			LeaseCreated:                  false,
			AttemptCreated:                false,
			ArtifactCreated:               false,
			GeneratedAt:                   created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/3/managed-generated-write-gate", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("managed generated write gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body managedGeneratedWriteGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode managed generated write gate response: %v", err)
	}
	if body.Status != "ready" || body.Mode != "read_only_managed_generated_write_gate" || body.Run.ID != 3 || body.Gate.Status != "pass" {
		t.Fatalf("unexpected managed generated write gate response: %+v", body)
	}
	if !body.GeneratedOnlyWriteReady || body.GeneratedOnlyApplyOpen {
		t.Fatalf("managed generated write gate should be ready but apply-closed: %+v", body)
	}
	if body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted ||
		body.AreaFlowArtifactWritten || body.AreaFlowExecutionStateWritten || body.EngineCallAttempted ||
		body.CommandsRun || body.SecretsResolved || body.NetworkUsed || body.TaskClaimed || body.WorkerStarted ||
		body.LeaseCreated || body.AttemptCreated || body.ArtifactCreated {
		t.Fatalf("managed generated write gate should be read-only and non-mutating: %+v", body)
	}
	if !containsString(body.AllowedGeneratedPrefixes, ".areaflow/generated/") ||
		!containsString(body.RequiredWriteSetFields, "generated_only") ||
		!containsString(body.UnsupportedOperations, "source_write") {
		t.Fatalf("missing generated-only safety contract: %+v", body)
	}
}

func TestWorkflowVersionRunsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 40, 0, 0, time.UTC)
	finished := created.Add(time.Second)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:  1,
			Key: "areamatrix",
		},
		version: project.WorkflowVersion{
			ID:           2,
			DisplayLabel: "v2",
			ImportMode:   "authored",
		},
		runs: []project.RunRecord{{
			ID:                3,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunType:           "runner_preview",
			RunKind:           "execution",
			Status:            "passed",
			RiskLevel:         "low",
			RiskPolicy:        "pause",
			DryRun:            true,
			Summary:           map[string]any{"task_count": float64(1)},
			Metadata:          map[string]any{"dry_run": true},
			StartedAt:         created,
			FinishedAt:        &finished,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions/v2/runs?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("workflow version runs status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow version runs: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.WorkflowVersion.DisplayLabel != "v2" || len(body.Runs) != 1 {
		t.Fatalf("unexpected workflow version runs response: %+v", body)
	}
	if body.Runs[0].ID != 3 || body.Runs[0].RunType != "runner_preview" || !body.Runs[0].DryRun {
		t.Fatalf("unexpected run metadata: %+v", body.Runs[0])
	}
}

func TestRunEventsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 10, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		runEvents: []project.EventRecord{{
			ID:        8,
			Type:      "worker.run_once.completed",
			Severity:  "info",
			Message:   "Worker run-once dry-run completed",
			Metadata:  map[string]any{"run_task_id": float64(4)},
			CreatedAt: created,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/runs/3/events?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("run events status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body runEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode run events: %v", err)
	}
	if body.RunID != 3 || len(body.Events) != 1 || body.Events[0].Type != "worker.run_once.completed" {
		t.Fatalf("unexpected run events response: %+v", body)
	}
}

func TestRunControlEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 12, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "runner_preview",
		RunKind:           "execution",
		Status:            "running",
		RiskLevel:         "low",
		RiskPolicy:        "pause",
		DryRun:            true,
		Summary:           map[string]any{"control_status": "running"},
		Metadata:          map[string]any{"protected_control": true},
		StartedAt:         created,
	}
	var got project.RunControlOptions
	handler := NewHandler(fakeProjectStore{
		runControlHook: func(options project.RunControlOptions) {
			got = options
		},
		runControl: project.RunControlResult{
			Project:                  record,
			Run:                      run,
			PreviousStatus:           "queued",
			Status:                   "running",
			Decision:                 "allowed",
			Message:                  "run marked running in protected mode",
			EventID:                  11,
			AuditEventID:             12,
			IdempotencyKey:           "run.start:test",
			Created:                  true,
			ProjectWriteAttempted:    false,
			ExecutionWriteAttempted:  false,
			AreaMatrixWriteAttempted: false,
			EngineCallAttempted:      false,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/3/start", strings.NewReader(`{"actor":"local-user","reason":"start fixture","idempotency_key":"run.start:test"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("run control status = %d body=%s", resp.Code, resp.Body.String())
	}
	if got.Action != "start" || got.Actor != "local-user" || got.Reason != "start fixture" || got.IdempotencyKey != "run.start:test" {
		t.Fatalf("unexpected run control options: %+v", got)
	}
	var body runControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode run control response: %v", err)
	}
	if body.Run.ID != 3 || body.Status != "running" || body.PreviousStatus != "queued" || body.Decision != "allowed" {
		t.Fatalf("unexpected run control response: %+v", body)
	}
	if body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.AreaMatrixWriteAttempted || body.EngineCallAttempted {
		t.Fatalf("run control attempted forbidden action: %+v", body)
	}
}

func TestRunControlEndpointBlocked(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		runErr: fmt.Errorf("%w: run status passed cannot be drained", project.ErrRunControlBlocked),
	})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodPost, "/api/runs/3/drain", strings.NewReader(`{"actor":"local-user"}`)))
	if resp.Code != http.StatusConflict {
		t.Fatalf("blocked run control status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestGlobalRunEndpointsHonorProjectKeyVisibility(t *testing.T) {
	created := time.Date(2026, 7, 2, 12, 10, 0, 0, time.UTC)
	scopeA := project.Record{ID: 1, Key: "scope-a", Name: "Scope A", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	scopeB := project.Record{ID: 2, Key: "scope-b", Name: "Scope B", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         scopeA.ID,
		WorkflowVersionID: 10,
		RunType:           "runner_preview",
		RunKind:           "execution",
		Status:            "queued",
		DryRun:            true,
		StartedAt:         created,
	}
	streamCalls := 0
	var gotStreamFilter project.EventStreamFilter
	handler := NewHandler(fakeProjectStore{
		recordsByKey: map[string]project.Record{
			scopeA.Key: scopeA,
			scopeB.Key: scopeB,
		},
		runDetail: project.RunDetail{Run: run},
		runEvents: []project.EventRecord{{
			ID:        8,
			ProjectID: scopeA.ID,
			RunID:     run.ID,
			Type:      "runner.preview.completed",
			Severity:  "info",
			Message:   "runner preview completed",
			CreatedAt: created,
		}},
		events: []project.EventRecord{{
			ID:        9,
			ProjectID: scopeA.ID,
			RunID:     run.ID,
			Type:      "runner.preview.completed",
			Severity:  "info",
			Message:   "runner preview completed",
			CreatedAt: created,
		}},
		streamHook: func(filter project.EventStreamFilter) {
			streamCalls++
			gotStreamFilter = filter
		},
		executionGate: project.ExecutionApprovalGate{
			Project:     scopeA,
			Run:         run,
			Status:      "blocked",
			Mode:        "read_only_execution_approval_gate",
			GeneratedAt: created,
		},
		runControl: project.RunControlResult{
			Project:               scopeA,
			Run:                   run,
			PreviousStatus:        "queued",
			Status:                "running",
			Decision:              "allowed",
			Message:               "run marked running in protected mode",
			IdempotencyKey:        "run.start:scope-a",
			Created:               true,
			ProjectWriteAttempted: false,
		},
	})

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "detail visible", method: http.MethodGet, path: "/api/v1/runs/3?project_key=scope-a", want: http.StatusOK},
		{name: "detail hidden", method: http.MethodGet, path: "/api/v1/runs/3?project_key=scope-b", want: http.StatusNotFound},
		{name: "events visible", method: http.MethodGet, path: "/api/v1/runs/3/events?limit=1&project_key=scope-a", want: http.StatusOK},
		{name: "events hidden", method: http.MethodGet, path: "/api/v1/runs/3/events?limit=1&project_key=scope-b", want: http.StatusNotFound},
		{name: "event stream visible", method: http.MethodGet, path: "/api/v1/runs/3/events/stream?once=true&project_key=scope-a", want: http.StatusOK},
		{name: "event stream hidden", method: http.MethodGet, path: "/api/v1/runs/3/events/stream?once=true&project_key=scope-b", want: http.StatusNotFound},
		{name: "gate visible", method: http.MethodGet, path: "/api/v1/runs/3/execution-approval-gate?project_key=scope-a", want: http.StatusOK},
		{name: "gate hidden", method: http.MethodGet, path: "/api/v1/runs/3/execution-approval-gate?project_key=scope-b", want: http.StatusNotFound},
		{name: "control visible", method: http.MethodPost, path: "/api/v1/runs/3/start?project_key=scope-a", body: `{"actor":"local-user","reason":"start scoped run","idempotency_key":"run.start:scope-a"}`, want: http.StatusCreated},
		{name: "control hidden", method: http.MethodPost, path: "/api/v1/runs/3/start?project_key=scope-b", body: `{"actor":"local-user","reason":"start scoped run","idempotency_key":"run.start:scope-b"}`, want: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if resp.Code != tt.want {
				t.Fatalf("%s %s status = %d, want %d body=%s", tt.method, tt.path, resp.Code, tt.want, resp.Body.String())
			}
		})
	}
	if streamCalls != 1 {
		t.Fatalf("run event stream calls = %d, want 1", streamCalls)
	}
	if gotStreamFilter.RunID != run.ID || gotStreamFilter.ProjectID != 0 {
		t.Fatalf("unexpected run stream filter: %+v", gotStreamFilter)
	}
}

func TestArtifactEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 20, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		artifact: project.ArtifactRecord{
			ID:                6,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "runner_preview_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/v2/runner-preview/run-3-report.json",
			SourcePath:        "v2/runner-preview/run-3-report.json",
			SHA256:            "abc123",
			SizeBytes:         128,
			ContentType:       "application/json",
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/artifacts/6", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("artifact status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body artifactResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	if body.ID != 6 || body.ArtifactType != "runner_preview_report" || body.SHA256 != "abc123" {
		t.Fatalf("unexpected artifact response: %+v", body)
	}
}

func TestArtifactContentEndpoint(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		artifactContent: project.ArtifactContent{
			Artifact: project.ArtifactRecord{
				ID:             6,
				ArtifactType:   "runner_preview_report",
				StorageBackend: "local",
				SHA256:         "abc123",
				ContentType:    "application/json",
			},
			Content:     []byte(`{"ok":true}`),
			ContentType: "application/json",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/6/content", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("artifact content status = %d body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	if got := resp.Header().Get("X-AreaFlow-Artifact-ID"); got != "6" {
		t.Fatalf("artifact id header = %q, want 6", got)
	}
	if got := resp.Header().Get("X-AreaFlow-Artifact-SHA256"); got != "abc123" {
		t.Fatalf("artifact sha header = %q, want abc123", got)
	}
	if got := resp.Body.String(); got != `{"ok":true}` {
		t.Fatalf("artifact content = %q", got)
	}
}

func TestArtifactContentEndpointUnavailable(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		artifactContentErr: fmt.Errorf("%w: storage backend project_reference is metadata-only for content API", project.ErrArtifactContentUnavailable),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/6/content", nil))
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("artifact content status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestArtifactContentEndpointMismatch(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		artifactContentErr: fmt.Errorf("%w: sha256 mismatch", project.ErrArtifactContentMismatch),
	})

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/6/content", nil))
	if resp.Code != http.StatusConflict {
		t.Fatalf("artifact content status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestGlobalArtifactEndpointsHonorProjectKeyVisibility(t *testing.T) {
	created := time.Date(2026, 7, 2, 12, 20, 0, 0, time.UTC)
	scopeA := project.Record{ID: 1, Key: "scope-a", Name: "Scope A", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	scopeB := project.Record{ID: 2, Key: "scope-b", Name: "Scope B", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	artifact := project.ArtifactRecord{
		ID:                6,
		ProjectID:         scopeA.ID,
		WorkflowVersionID: 10,
		ArtifactType:      "runner_preview_report",
		StorageBackend:    "local",
		URI:               "/tmp/areaflow/artifacts/scope-a/v2/runner-preview/report.json",
		SourcePath:        "v2/runner-preview/report.json",
		SHA256:            "abc123",
		SizeBytes:         11,
		ContentType:       "application/json",
		CreatedAt:         created,
	}
	handler := NewHandler(fakeProjectStore{
		recordsByKey: map[string]project.Record{
			scopeA.Key: scopeA,
			scopeB.Key: scopeB,
		},
		artifact: artifact,
		artifactContent: project.ArtifactContent{
			Artifact:    artifact,
			Content:     []byte(`{"ok":true}`),
			ContentType: "application/json",
		},
	})

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "metadata visible", path: "/api/v1/artifacts/6?project_key=scope-a", want: http.StatusOK},
		{name: "metadata hidden", path: "/api/v1/artifacts/6?project_key=scope-b", want: http.StatusNotFound},
		{name: "content visible", path: "/api/v1/artifacts/6/content?project_key=scope-a", want: http.StatusOK},
		{name: "content hidden", path: "/api/v1/artifacts/6/content?project_key=scope-b", want: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if resp.Code != tt.want {
				t.Fatalf("%s status = %d, want %d body=%s", tt.path, resp.Code, tt.want, resp.Body.String())
			}
			if tt.want == http.StatusOK && strings.Contains(tt.path, "/content") && resp.Body.String() != `{"ok":true}` {
				t.Fatalf("content response = %q", resp.Body.String())
			}
		})
	}
}

func TestProjectArtifactsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		artifacts: []project.ArtifactRecord{{
			ID:                7,
			ProjectID:         1,
			WorkflowVersionID: 2,
			WorkflowItemID:    5,
			ArtifactType:      "worker_run_once_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/v2/workers/local-1/report.json",
			SourcePath:        "v2/workers/local-1/report.json",
			SHA256:            "def456",
			SizeBytes:         256,
			ContentType:       "application/json",
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/artifacts?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project artifacts status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectArtifactsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode project artifacts: %v", err)
	}
	if body.Project.Key != "areamatrix" || len(body.Artifacts) != 1 {
		t.Fatalf("unexpected project artifacts response: %+v", body)
	}
	if body.Artifacts[0].ID != 7 || body.Artifacts[0].WorkflowItemID != 5 || body.Artifacts[0].ArtifactType != "worker_run_once_report" {
		t.Fatalf("unexpected artifact metadata: %+v", body.Artifacts[0])
	}
}

func TestWorkflowVersionArtifactsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 32, 0, 0, time.UTC)
	version := project.WorkflowVersion{
		ID:              2,
		ProjectID:       1,
		DisplayLabel:    "v2",
		VersionKind:     "workflow_version",
		LifecycleStatus: "draft",
		ImportMode:      "authored",
		StatusSummary:   map[string]any{},
		CreatedAt:       created,
		UpdatedAt:       created,
	}
	handler := NewHandler(fakeProjectStore{
		record:  project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		version: version,
		artifacts: []project.ArtifactRecord{{
			ID:                8,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "plan",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/v2/plan.md",
			SourcePath:        "v2/plan.md",
			SHA256:            "abc456",
			SizeBytes:         512,
			ContentType:       "text/markdown",
			Metadata:          map[string]any{"stage": "plans"},
			CreatedAt:         created,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions/v2/artifacts?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("workflow version artifacts status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionArtifactsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow version artifacts: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.WorkflowVersion.DisplayLabel != "v2" || len(body.Artifacts) != 1 {
		t.Fatalf("unexpected workflow version artifacts response: %+v", body)
	}
	if body.Artifacts[0].WorkflowVersionID != 2 || body.Artifacts[0].ArtifactType != "plan" {
		t.Fatalf("unexpected workflow version artifact: %+v", body.Artifacts[0])
	}
}

func TestProjectResidualsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 35, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	imported := updated.Add(time.Minute)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		residuals: []project.ResidualRecord{{
			ID:                9,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ResidualKey:       "distribution-notarization",
			Status:            "blocked-external",
			Type:              "release-blocker",
			Title:             "Distribution notarization evidence",
			SourcePath:        "workflow/versions/v1-mvp/residuals/residuals.yaml",
			CurrentImpact:     "Blocks formal distribution claims",
			ExecutableTask:    false,
			PromotionRequired: true,
			CloseCondition:    "Formal distribution decision recorded",
			Metadata:          map[string]any{"blocker": "external account"},
			Immutable:         true,
			CreatedAt:         created,
			UpdatedAt:         updated,
			ImportedAt:        &imported,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/residuals?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project residuals status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectResidualsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode project residuals: %v", err)
	}
	if body.Project.Key != "areamatrix" || len(body.Residuals) != 1 {
		t.Fatalf("unexpected project residuals response: %+v", body)
	}
	residual := body.Residuals[0]
	if residual.ResidualKey != "distribution-notarization" || residual.Status != "blocked-external" || !residual.PromotionRequired {
		t.Fatalf("unexpected residual metadata: %+v", residual)
	}
	if residual.ImportedAt == "" || residual.Metadata["blocker"] != "external account" {
		t.Fatalf("unexpected residual import/metadata fields: %+v", residual)
	}
}

func TestWorkflowVersionResidualsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 40, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	imported := updated.Add(time.Minute)
	version := project.WorkflowVersion{
		ID:              2,
		ProjectID:       1,
		DisplayLabel:    "v1-mvp",
		VersionKind:     "workflow_version",
		LifecycleStatus: "archived",
		ImportMode:      "metadata_only",
		Immutable:       true,
		StatusSummary:   map[string]any{},
		CreatedAt:       created,
		UpdatedAt:       updated,
		ImportedAt:      &imported,
	}
	handler := NewHandler(fakeProjectStore{
		record:  project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		version: version,
		residuals: []project.ResidualRecord{{
			ID:                10,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ResidualKey:       "v1-rl-006",
			Status:            "blocked-decision",
			Type:              "release-evidence",
			Title:             "Release evidence decision",
			SourcePath:        "workflow/versions/v1-mvp/residuals/residuals.yaml",
			CurrentImpact:     "Blocks formal distribution claims",
			Metadata:          map[string]any{"owner": "release"},
			Immutable:         true,
			CreatedAt:         created,
			UpdatedAt:         updated,
			ImportedAt:        &imported,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions/v1-mvp/residuals?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("workflow version residuals status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionResidualsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow version residuals: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.WorkflowVersion.DisplayLabel != "v1-mvp" || len(body.Residuals) != 1 {
		t.Fatalf("unexpected workflow version residuals response: %+v", body)
	}
	if body.Residuals[0].WorkflowVersionID != 2 || body.Residuals[0].ResidualKey != "v1-rl-006" {
		t.Fatalf("unexpected workflow version residual: %+v", body.Residuals[0])
	}
}

func TestRunAndArtifactNotFoundEndpoints(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		runErr:      project.ErrRunNotFound,
		artifactErr: project.ErrArtifactNotFound,
	})
	runResp := httptest.NewRecorder()
	handler.ServeHTTP(runResp, httptest.NewRequest(http.MethodGet, "/api/runs/999", nil))
	if runResp.Code != http.StatusNotFound {
		t.Fatalf("run not found status = %d body=%s", runResp.Code, runResp.Body.String())
	}
	artifactResp := httptest.NewRecorder()
	handler.ServeHTTP(artifactResp, httptest.NewRequest(http.MethodGet, "/api/artifacts/999", nil))
	if artifactResp.Code != http.StatusNotFound {
		t.Fatalf("artifact not found status = %d body=%s", artifactResp.Code, artifactResp.Body.String())
	}
}

func TestWebDashboardReadEndpointsUseAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 5, 30, 0, 0, time.UTC)
	finished := created.Add(time.Second)
	updated := created.Add(time.Minute)
	imported := updated.Add(time.Minute)
	record := project.Record{
		ID:              1,
		Key:             "areamatrix",
		Name:            "AreaMatrix",
		Kind:            "desktop-app",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix",
		DefaultBranch:   "main",
		RootPath:        "/tmp/areamatrix",
	}
	version := project.WorkflowVersion{
		ID:              2,
		ProjectID:       1,
		DisplayLabel:    "v2",
		VersionKind:     "workflow_version",
		LifecycleStatus: "draft",
		ImportMode:      "authored",
		StatusSummary:   map[string]any{},
		CreatedAt:       created,
		UpdatedAt:       created,
	}
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "runner_preview",
		RunKind:           "execution",
		Status:            "passed",
		RiskLevel:         "low",
		RiskPolicy:        "pause",
		DryRun:            true,
		Summary:           map[string]any{"task_count": float64(1)},
		Metadata:          map[string]any{"dry_run": true},
		StartedAt:         created,
		FinishedAt:        &finished,
	}
	task := project.RunTaskRecord{
		ID:                4,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunID:             3,
		TaskKey:           "v2:runner-preview",
		TaskKind:          "workflow_item_preview",
		Status:            "passed",
		RiskLevel:         "low",
		Sequence:          1,
		Metadata:          map[string]any{"dry_run": true},
		CreatedAt:         created,
		UpdatedAt:         finished,
	}
	attempt := project.RunAttemptRecord{
		ID:                5,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunID:             3,
		RunTaskID:         4,
		AttemptKind:       "worker_run_once",
		Status:            "passed",
		DryRun:            true,
		Metadata:          map[string]any{"would_execute": false},
		StartedAt:         created,
		FinishedAt:        &finished,
	}
	artifact := project.ArtifactRecord{
		ID:                6,
		ProjectID:         1,
		WorkflowVersionID: 2,
		WorkflowItemID:    5,
		ArtifactType:      "worker_run_once_report",
		StorageBackend:    "local",
		URI:               "/tmp/areaflow/artifacts/areamatrix/v2/workers/local-1/report.json",
		SourcePath:        "v2/workers/local-1/report.json",
		SHA256:            "def456",
		SizeBytes:         256,
		ContentType:       "application/json",
		Metadata:          map[string]any{"dry_run": true},
		CreatedAt:         created,
	}
	residual := project.ResidualRecord{
		ID:                9,
		ProjectID:         1,
		WorkflowVersionID: 2,
		ResidualKey:       "distribution-notarization",
		Status:            "blocked-external",
		Type:              "release-blocker",
		Title:             "Distribution notarization evidence",
		SourcePath:        "workflow/versions/v1-mvp/residuals/residuals.yaml",
		CurrentImpact:     "Blocks formal distribution claims",
		ExecutableTask:    false,
		PromotionRequired: true,
		CloseCondition:    "Formal distribution decision recorded",
		Metadata:          map[string]any{"blocker": "external account"},
		Immutable:         true,
		CreatedAt:         created,
		UpdatedAt:         updated,
		ImportedAt:        &imported,
	}
	approval := project.ApprovalRecord{
		ID:                  10,
		ProjectID:           1,
		WorkflowVersionID:   2,
		TransitionPreviewID: 4,
		ApprovalKind:        "workflow_transition",
		Decision:            "approved",
		ScopeType:           "workflow_version",
		ScopeID:             "v2",
		Actor:               "local-user",
		Reason:              "ready preview",
		RiskLevel:           "normal",
		Metadata:            map[string]any{"phase": "v0.7"},
		CreatedAt:           created,
	}
	event := project.EventRecord{
		ID:        11,
		Type:      "worker.run_once.completed",
		Severity:  "info",
		Message:   "Worker run-once dry-run completed",
		Metadata:  map[string]any{"run_task_id": float64(4)},
		CreatedAt: created,
	}
	handler := NewHandler(fakeProjectStore{
		record:    record,
		version:   version,
		artifact:  artifact,
		artifacts: []project.ArtifactRecord{artifact},
		residuals: []project.ResidualRecord{residual},
		approvals: []project.ApprovalRecord{approval},
		runs:      []project.RunRecord{run},
		runDetail: project.RunDetail{
			Run:       run,
			Tasks:     []project.RunTaskRecord{task},
			Attempts:  []project.RunAttemptRecord{attempt},
			Artifacts: []project.ArtifactRecord{artifact},
		},
		runEvents: []project.EventRecord{event},
	})

	tests := []struct {
		name      string
		path      string
		wantField string
	}{
		{name: "project detail", path: "/api/v1/projects/areamatrix", wantField: "key"},
		{name: "project artifacts", path: "/api/v1/projects/areamatrix/artifacts?limit=1", wantField: "artifacts"},
		{name: "project residuals", path: "/api/v1/projects/areamatrix/residuals?limit=1", wantField: "residuals"},
		{name: "version runs", path: "/api/v1/projects/areamatrix/workflow-versions/v2/runs?limit=1", wantField: "runs"},
		{name: "version artifacts", path: "/api/v1/projects/areamatrix/workflow-versions/v2/artifacts?limit=1", wantField: "artifacts"},
		{name: "version residuals", path: "/api/v1/projects/areamatrix/workflow-versions/v2/residuals?limit=1", wantField: "residuals"},
		{name: "version approvals", path: "/api/v1/projects/areamatrix/workflow-versions/v2/approvals?limit=1", wantField: "approval_records"},
		{name: "run detail", path: "/api/v1/runs/3", wantField: "run"},
		{name: "run events", path: "/api/v1/runs/3/events?limit=1", wantField: "events"},
		{name: "artifact detail", path: "/api/v1/artifacts/6", wantField: "artifact_type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if resp.Code != http.StatusOK {
				t.Fatalf("%s status = %d body=%s", tt.path, resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode %s response: %v", tt.path, err)
			}
			if _, ok := body[tt.wantField]; !ok {
				t.Fatalf("%s response missing %q: %+v", tt.path, tt.wantField, body)
			}
		})
	}
}

func TestWorkflowVersionRunnerPreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	finished := created.Add(time.Second)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", StatusSummary: map[string]any{}}
	runner := project.RunnerPreviewResult{
		Project: record,
		Version: version,
		Run: project.RunRecord{
			ID:                3,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunType:           "runner_preview",
			RunKind:           "execution",
			Status:            "passed",
			RiskLevel:         "low",
			RiskPolicy:        "pause",
			DryRun:            true,
			Summary:           map[string]any{"attempt_count": float64(2)},
			Metadata:          map[string]any{"dry_run": true},
			StartedAt:         created,
			FinishedAt:        &finished,
		},
		Tasks: []project.RunTaskRecord{{
			ID:                4,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			TaskKey:           "v2:runner-preview",
			TaskKind:          "workflow_item_preview",
			Status:            "queued",
			RiskLevel:         "low",
			Sequence:          1,
			Metadata:          map[string]any{"copy_ready_source": "not_materialized_in_v0.5_preview"},
			CreatedAt:         created,
			UpdatedAt:         created,
		}},
		Attempts: []project.RunAttemptRecord{{
			ID:                5,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "copy",
			Status:            "passed",
			DryRun:            true,
			Metadata:          map[string]any{"would_execute": false},
			StartedAt:         created,
			FinishedAt:        &finished,
		}, {
			ID:                6,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "verify",
			Status:            "passed",
			DryRun:            true,
			Metadata:          map[string]any{"would_execute": false},
			StartedAt:         created,
			FinishedAt:        &finished,
		}},
		Artifacts: []project.ArtifactRecord{{
			ID:                7,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "runner_preview_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/v2/runner-preview/run-3-report.json",
			SourcePath:        "v2/runner-preview/run-3-report.json",
			SHA256:            "abc123",
			SizeBytes:         128,
			ContentType:       "application/json",
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
		}},
		Preflight: project.RunnerPreflight{
			Status: "pass",
			Checks: []project.ReadinessItem{{
				Key:     "dry_run",
				Status:  "pass",
				Message: "runner preview is dry-run only",
			}},
		},
		Created:        true,
		IdempotencyKey: "runner.preview:areamatrix:v2",
	}
	handler := NewHandler(fakeProjectStore{
		record:  record,
		version: version,
		runner:  runner,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workflow-versions/v2/runner-preview", strings.NewReader(`{"actor":"local-user"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body runnerPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode runner preview response: %v", err)
	}
	if body.Run.ID != 3 || !body.Run.DryRun || body.Preflight.Status != "pass" {
		t.Fatalf("unexpected runner preview response: %+v", body)
	}
	if len(body.Tasks) != 1 || len(body.Attempts) != 2 || len(body.Artifacts) != 1 {
		t.Fatalf("unexpected runner preview tasks/attempts/artifacts: %+v", body)
	}
	if body.Tasks[0].Status != "queued" {
		t.Fatalf("runner preview task status = %q, want queued", body.Tasks[0].Status)
	}
	if body.Attempts[0].AttemptKind != "copy" || body.Attempts[1].AttemptKind != "verify" {
		t.Fatalf("unexpected runner preview attempts: %+v", body.Attempts)
	}
	if body.Artifacts[0].ArtifactType != "runner_preview_report" || body.Artifacts[0].SHA256 != "abc123" {
		t.Fatalf("unexpected runner preview artifact: %+v", body.Artifacts[0])
	}
}

func TestWorkflowVersionRunnerPreviewEndpointUsesAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	finished := created.Add(time.Second)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", StatusSummary: map[string]any{}}
	handler := NewHandler(fakeProjectStore{
		record:  record,
		version: version,
		runner: project.RunnerPreviewResult{
			Project: record,
			Version: version,
			Run: project.RunRecord{
				ID:                3,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunType:           "runner_preview",
				RunKind:           "execution",
				Status:            "passed",
				RiskLevel:         "low",
				RiskPolicy:        "pause",
				DryRun:            true,
				Summary:           map[string]any{"attempt_count": float64(2)},
				Metadata:          map[string]any{"dry_run": true},
				StartedAt:         created,
				FinishedAt:        &finished,
			},
			Tasks: []project.RunTaskRecord{{
				ID:                4,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunID:             3,
				TaskKey:           "v2:runner-preview",
				TaskKind:          "workflow_item_preview",
				Status:            "queued",
				RiskLevel:         "low",
				Sequence:          1,
				Metadata:          map[string]any{"copy_ready_source": "not_materialized_in_v0.5_preview"},
				CreatedAt:         created,
				UpdatedAt:         created,
			}},
			Attempts: []project.RunAttemptRecord{{
				ID:                5,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunID:             3,
				RunTaskID:         4,
				AttemptKind:       "copy",
				Status:            "passed",
				DryRun:            true,
				Metadata:          map[string]any{"would_execute": false},
				StartedAt:         created,
				FinishedAt:        &finished,
			}, {
				ID:                6,
				ProjectID:         1,
				WorkflowVersionID: 2,
				RunID:             3,
				RunTaskID:         4,
				AttemptKind:       "verify",
				Status:            "passed",
				DryRun:            true,
				Metadata:          map[string]any{"would_execute": false},
				StartedAt:         created,
				FinishedAt:        &finished,
			}},
			Artifacts: []project.ArtifactRecord{{
				ID:                7,
				ProjectID:         1,
				WorkflowVersionID: 2,
				ArtifactType:      "runner_preview_report",
				StorageBackend:    "local",
				URI:               "/tmp/areaflow/artifacts/areamatrix/v2/runner-preview/run-3-report.json",
				SourcePath:        "v2/runner-preview/run-3-report.json",
				SHA256:            "abc123",
				SizeBytes:         128,
				ContentType:       "application/json",
				Metadata:          map[string]any{"dry_run": true},
				CreatedAt:         created,
			}},
			Preflight: project.RunnerPreflight{
				Status: "pass",
				Checks: []project.ReadinessItem{{
					Key:     "dry_run",
					Status:  "pass",
					Message: "runner preview is dry-run only",
				}},
			},
			Created:        true,
			IdempotencyKey: "runner.preview:areamatrix:v2",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workflow-versions/v2/runner-preview", strings.NewReader(`{"actor":"local-user"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body runnerPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode runner preview response: %v", err)
	}
	if !body.Run.DryRun || len(body.Tasks) != 1 || len(body.Attempts) != 2 || len(body.Artifacts) != 1 {
		t.Fatalf("unexpected runner preview response: %+v", body)
	}
	if body.Attempts[0].AttemptKind != "copy" || body.Attempts[1].AttemptKind != "verify" {
		t.Fatalf("unexpected runner preview attempts: %+v", body.Attempts)
	}
	if body.Artifacts[0].ArtifactType != "runner_preview_report" || body.Preflight.Status != "pass" {
		t.Fatalf("unexpected runner preview evidence: %+v", body)
	}
}

func TestWorkflowVersionFixtureQueueEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 1, 10, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "fixture_execution", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created}
	task := project.RunTaskRecord{ID: 4, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:fixture-execution", TaskKind: "fixture_execution_task", Status: "queued", RiskLevel: "low", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	var capturedOptions project.FixtureExecutionQueueOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		fixtureQueueHook: func(options project.FixtureExecutionQueueOptions) {
			capturedOptions = options
		},
		fixtureQueue: project.FixtureExecutionQueueResult{
			Project:        record,
			Version:        version,
			Run:            run,
			Task:           task,
			Created:        true,
			IdempotencyKey: "fixture-queue-key",
			EventID:        5,
			AuditEventID:   6,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workflow-versions/v2/fixture-queue", strings.NewReader(`{
		"actor": "local-user",
		"reason": "queue fixture",
		"idempotency_key": "fixture-queue-key"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("fixture queue status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.Actor != "local-user" || capturedOptions.Reason != "queue fixture" || capturedOptions.IdempotencyKey != "fixture-queue-key" {
		t.Fatalf("unexpected fixture queue options: %+v", capturedOptions)
	}
	var body fixtureExecutionQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode fixture queue response: %v", err)
	}
	if body.Run.ID != 3 || body.Run.DryRun || body.Task.ID != 4 || body.Task.TaskKind != "fixture_execution_task" {
		t.Fatalf("unexpected fixture queue response: %+v", body)
	}
	if body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed {
		t.Fatalf("unexpected fixture queue safety facts: %+v", body)
	}
}

func TestWorkflowVersionReadOnlyVerifyQueueEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 2, 10, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "read_only_verify", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created}
	task := project.RunTaskRecord{ID: 4, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:read-only-verify:docs/README.md", TaskKind: "read_only_verify_task", Status: "queued", RiskLevel: "low", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	var capturedOptions project.ReadOnlyVerifyQueueOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		readOnlyQueueHook: func(options project.ReadOnlyVerifyQueueOptions) {
			capturedOptions = options
		},
		readOnlyQueue: project.ReadOnlyVerifyQueueResult{
			Project:        record,
			Version:        version,
			Run:            run,
			Task:           task,
			TargetPath:     "docs/README.md",
			Created:        true,
			IdempotencyKey: "read-only-queue-key",
			EventID:        5,
			AuditEventID:   6,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workflow-versions/v2/read-only-verify-queue", strings.NewReader(`{
		"target_path": "docs/README.md",
		"actor": "local-user",
		"reason": "queue read-only verify",
		"idempotency_key": "read-only-queue-key"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("read-only verify queue status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetPath != "docs/README.md" || capturedOptions.Actor != "local-user" || capturedOptions.Reason != "queue read-only verify" || capturedOptions.IdempotencyKey != "read-only-queue-key" {
		t.Fatalf("unexpected read-only verify queue options: %+v", capturedOptions)
	}
	var body readOnlyVerifyQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode read-only verify queue response: %v", err)
	}
	if body.Run.ID != 3 || body.Run.DryRun || body.Task.ID != 4 || body.Task.TaskKind != "read_only_verify_task" || body.TargetPath != "docs/README.md" {
		t.Fatalf("unexpected read-only verify queue response: %+v", body)
	}
	if body.ProjectReadAttempted || body.ProjectReadAllowed || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed {
		t.Fatalf("unexpected read-only verify queue safety facts: %+v", body)
	}
}

func TestWorkflowVersionApprovedArtifactWriteQueueEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 3, 10, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "approved_artifact_write", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created}
	task := project.RunTaskRecord{ID: 4, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:approved-artifact-write:approval-note", TaskKind: "approved_artifact_write_task", Status: "queued", RiskLevel: "low", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	var capturedOptions project.ApprovedArtifactWriteQueueOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		artifactWriteQueueHook: func(options project.ApprovedArtifactWriteQueueOptions) {
			capturedOptions = options
		},
		artifactWriteQueue: project.ApprovedArtifactWriteQueueResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Task:                          task,
			ArtifactLabel:                 "approval-note",
			Created:                       true,
			IdempotencyKey:                "artifact-write-queue-key",
			EventID:                       5,
			AuditEventID:                  6,
			AreaFlowExecutionStateWritten: true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/workflow-versions/v2/approved-artifact-write-queue", strings.NewReader(`{
		"artifact_label": "approval-note",
		"actor": "local-user",
		"reason": "queue artifact write",
		"idempotency_key": "artifact-write-queue-key"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("approved artifact write queue status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.ArtifactLabel != "approval-note" || capturedOptions.Actor != "local-user" || capturedOptions.Reason != "queue artifact write" || capturedOptions.IdempotencyKey != "artifact-write-queue-key" {
		t.Fatalf("unexpected approved artifact write queue options: %+v", capturedOptions)
	}
	var body approvedArtifactWriteQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode approved artifact write queue response: %v", err)
	}
	if body.Run.ID != 3 || body.Run.DryRun || body.Task.ID != 4 || body.Task.TaskKind != "approved_artifact_write_task" || body.ArtifactLabel != "approval-note" {
		t.Fatalf("unexpected approved artifact write queue response: %+v", body)
	}
	if body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.AreaFlowArtifactWritten || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowExecutionStateWritten {
		t.Fatalf("unexpected approved artifact write queue safety facts: %+v", body)
	}
}

func TestWorkflowVersionFixtureProjectWriteQueueEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 4, 10, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix-fixture", Name: "AreaMatrix Fixture", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "fixture_project_write", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created}
	task := project.RunTaskRecord{ID: 4, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:fixture-project-write:fixtures/input.txt", TaskKind: "fixture_project_write_task", Status: "queued", RiskLevel: "medium", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	writeSetArtifact := project.ArtifactRecord{ID: 5, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "fixture_project_write_set", StorageBackend: "local", URI: "/tmp/artifacts/write-set.json", SourcePath: "versions/v2/fixture-project-write/write-set.json", SHA256: "writeset123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.FixtureProjectWriteQueueOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		fixtureProjectQueueHook: func(options project.FixtureProjectWriteQueueOptions) {
			capturedOptions = options
		},
		fixtureProjectQueue: project.FixtureProjectWriteQueueResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Task:                          task,
			WriteSetArtifact:              writeSetArtifact,
			TargetPath:                    "fixtures/input.txt",
			ExpectedBeforeSHA256:          "before123",
			ExpectedBeforeSize:            12,
			AfterSHA256:                   "after123",
			AfterSize:                     13,
			Created:                       true,
			IdempotencyKey:                "fixture-project-write-queue-key",
			EventID:                       6,
			AuditEventID:                  7,
			AreaFlowArtifactWritten:       true,
			AreaFlowExecutionStateWritten: true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix-fixture/workflow-versions/v2/fixture-project-write-queue", strings.NewReader(`{
		"target_path": "fixtures/input.txt",
		"content": "after content",
		"expected_before_sha256": "before123",
		"expected_before_size": 12,
		"actor": "local-user",
		"reason": "queue fixture project write",
		"idempotency_key": "fixture-project-write-queue-key"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("fixture project write queue status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetPath != "fixtures/input.txt" ||
		capturedOptions.Content != "after content" ||
		capturedOptions.ExpectedBeforeSHA256 != "before123" ||
		capturedOptions.ExpectedBeforeSize != 12 ||
		capturedOptions.Actor != "local-user" ||
		capturedOptions.Reason != "queue fixture project write" ||
		capturedOptions.IdempotencyKey != "fixture-project-write-queue-key" {
		t.Fatalf("unexpected fixture project write queue options: %+v", capturedOptions)
	}
	var body fixtureProjectWriteQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode fixture project write queue response: %v", err)
	}
	if body.Run.ID != 3 || body.Run.DryRun || body.Task.ID != 4 || body.Task.TaskKind != "fixture_project_write_task" || body.WriteSetArtifact.ArtifactType != "fixture_project_write_set" || body.TargetPath != "fixtures/input.txt" {
		t.Fatalf("unexpected fixture project write queue response: %+v", body)
	}
	if body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowArtifactWritten || !body.AreaFlowExecutionStateWritten {
		t.Fatalf("unexpected fixture project write queue safety facts: %+v", body)
	}
}

func TestWorkflowVersionManagedGeneratedWriteQueueEndpoint(t *testing.T) {
	created := time.Date(2026, 7, 2, 10, 10, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix-fixture", Name: "AreaMatrix Fixture", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", CreatedAt: created, UpdatedAt: created}
	run := project.RunRecord{ID: 3, ProjectID: 1, WorkflowVersionID: 2, RunType: "managed_generated_write", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created}
	task := project.RunTaskRecord{ID: 4, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, TaskKey: "v2:managed-generated-write:.areaflow/generated/status.json", TaskKind: "managed_generated_write_task", Status: "queued", RiskLevel: "medium", Sequence: 1, CreatedAt: created, UpdatedAt: created}
	writeSetArtifact := project.ArtifactRecord{ID: 5, ProjectID: 1, WorkflowVersionID: 2, ArtifactType: "managed_generated_write_set", StorageBackend: "local", URI: "/tmp/artifacts/write-set.json", SourcePath: "versions/v2/managed-generated-write/write-set.json", SHA256: "writeset123", SizeBytes: 128, ContentType: "application/json", CreatedAt: created}
	var capturedOptions project.ManagedGeneratedWriteQueueOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		managedGeneratedQueueHook: func(options project.ManagedGeneratedWriteQueueOptions) {
			capturedOptions = options
		},
		managedGeneratedQueue: project.ManagedGeneratedWriteQueueResult{
			Project:                       record,
			Version:                       version,
			Run:                           run,
			Task:                          task,
			WriteSetArtifact:              writeSetArtifact,
			TargetPath:                    ".areaflow/generated/status.json",
			ExpectedBeforeSHA256:          "before123",
			ExpectedBeforeSize:            12,
			AfterSHA256:                   "after123",
			AfterSize:                     13,
			Created:                       true,
			IdempotencyKey:                "managed-generated-write-queue-key",
			EventID:                       6,
			AuditEventID:                  7,
			GeneratedOnly:                 true,
			GeneratedOnlyApplyOpen:        true,
			AreaFlowArtifactWritten:       true,
			AreaFlowExecutionStateWritten: true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix-fixture/workflow-versions/v2/managed-generated-write-queue", strings.NewReader(`{
		"target_path": ".areaflow/generated/status.json",
		"content": "after content",
		"expected_before_sha256": "before123",
		"expected_before_size": 12,
		"actor": "local-user",
		"reason": "queue managed generated write",
		"idempotency_key": "managed-generated-write-queue-key"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("managed generated write queue status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetPath != ".areaflow/generated/status.json" ||
		capturedOptions.Content != "after content" ||
		capturedOptions.ExpectedBeforeSHA256 != "before123" ||
		capturedOptions.ExpectedBeforeSize != 12 ||
		capturedOptions.Actor != "local-user" ||
		capturedOptions.Reason != "queue managed generated write" ||
		capturedOptions.IdempotencyKey != "managed-generated-write-queue-key" {
		t.Fatalf("unexpected managed generated write queue options: %+v", capturedOptions)
	}
	var body managedGeneratedWriteQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode managed generated write queue response: %v", err)
	}
	if body.Run.ID != 3 || body.Run.DryRun || body.Task.ID != 4 || body.Task.TaskKind != "managed_generated_write_task" || body.WriteSetArtifact.ArtifactType != "managed_generated_write_set" || body.TargetPath != ".areaflow/generated/status.json" || !body.GeneratedOnly || !body.GeneratedOnlyApplyOpen {
		t.Fatalf("unexpected managed generated write queue response: %+v", body)
	}
	if body.ProjectReadAttempted || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted || body.CommandsRun || body.SecretsResolved || body.NetworkUsed || !body.AreaFlowArtifactWritten || !body.AreaFlowExecutionStateWritten {
		t.Fatalf("unexpected managed generated write queue safety facts: %+v", body)
	}
}

func TestProjectCutoverReadinessEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	record := project.Record{
		ID:              1,
		Key:             "areamatrix",
		Name:            "AreaMatrix",
		Kind:            "desktop-app",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix",
		DefaultBranch:   "main",
		RootPath:        "/tmp/AreaMatrix",
	}
	summary := project.ProjectSummary{
		Project: record,
		Inventory: project.ImportInventory{
			MirrorExports: 1,
		},
		Import:              project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		HasImport:           true,
		HasPreviousImport:   true,
		PreviousImport:      project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "pass",
		LatestEventCount:    1,
	}
	readiness := project.ProjectReadinessFromSummary(summary)
	diff := project.ProjectImportDiffFromSummary(summary)
	bundle := project.ProjectVerificationBundleFromParts(summary, readiness, diff, []project.EventRecord{
		{ID: 1, Type: "project.doctor.completed", Severity: "info", Message: "doctor", CreatedAt: created},
	})
	compat := project.CompatibilityContractFromSummary(summary, map[string]project.CommandPermission{})
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", StatusSummary: map[string]any{}}
	gates := []project.GateResult{
		{ID: 4, GateName: "approval_gate", Status: "pass", CheckedAt: created},
		{ID: 5, GateName: "live_mapping_gate", Status: "pass", CheckedAt: created},
	}
	cutover := project.ProjectCutoverReadinessFromParts(bundle, compat, version, gates)
	handler := NewHandler(fakeProjectStore{
		record:  record,
		cutover: cutover,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/cutover-readiness?version=v2", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectCutoverReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode cutover readiness response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.WorkflowVersion.DisplayLabel != "v2" {
		t.Fatalf("unexpected cutover readiness project/version: %+v", body)
	}
	if body.PhaseGate.Name != "v0.4-cutover-readiness" {
		t.Fatalf("unexpected phase gate: %+v", body.PhaseGate)
	}
	if len(body.Items) == 0 || len(body.Gates) != 2 {
		t.Fatalf("unexpected cutover readiness body: %+v", body)
	}
}

func TestProjectCutoverReadinessEndpointRequiresVersion(t *testing.T) {
	handler := NewHandler(fakeProjectStore{})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/cutover-readiness", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want bad request", resp.Code, resp.Body.String())
	}
}

func TestProjectCutoverEndpointsUseAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	record := project.Record{
		ID:              1,
		Key:             "areamatrix",
		Name:            "AreaMatrix",
		Kind:            "desktop-app",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix",
		DefaultBranch:   "main",
		RootPath:        "/tmp/AreaMatrix",
	}
	summary := project.ProjectSummary{
		Project: record,
		Inventory: project.ImportInventory{
			MirrorExports: 1,
		},
		Import:              project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		HasImport:           true,
		HasPreviousImport:   true,
		PreviousImport:      project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "pass",
		LatestEventCount:    1,
	}
	readiness := project.ProjectReadinessFromSummary(summary)
	diff := project.ProjectImportDiffFromSummary(summary)
	bundle := project.ProjectVerificationBundleFromParts(summary, readiness, diff, []project.EventRecord{
		{ID: 1, Type: "project.doctor.completed", Severity: "info", Message: "doctor", CreatedAt: created},
	})
	compat := project.CompatibilityContractFromSummary(summary, map[string]project.CommandPermission{})
	version := project.WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", StatusSummary: map[string]any{}}
	cutover := project.ProjectCutoverReadinessFromParts(bundle, compat, version, []project.GateResult{
		{ID: 4, GateName: "approval_gate", Status: "pass", CheckedAt: created},
		{ID: 5, GateName: "live_mapping_gate", Status: "pass", CheckedAt: created},
	})
	handler := NewHandler(fakeProjectStore{
		record:  record,
		compat:  compat,
		cutover: cutover,
	})

	tests := []struct {
		name      string
		path      string
		wantField string
	}{
		{name: "compatibility", path: "/api/v1/projects/areamatrix/compatibility", wantField: "commands"},
		{name: "cutover readiness", path: "/api/v1/projects/areamatrix/cutover-readiness?version=v2", wantField: "phase_gate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if resp.Code != http.StatusOK {
				t.Fatalf("%s status = %d body=%s", tt.path, resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode %s response: %v", tt.path, err)
			}
			if _, ok := body[tt.wantField]; !ok {
				t.Fatalf("%s response missing %q: %+v", tt.path, tt.wantField, body)
			}
		})
	}
}

func TestProjectStatusProjectionsEndpoint(t *testing.T) {
	generatedAt := time.Date(2026, 7, 1, 8, 30, 0, 0, time.UTC)
	writtenAt := generatedAt.Add(time.Minute)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		statusProjections: []project.StatusProjectionRecord{
			{
				ID:           10,
				ProjectID:    1,
				TargetKind:   "project_status_json",
				TargetURI:    ".areaflow/status.json",
				SummaryState: "mirroring",
				Payload:      map[string]any{"version_count": float64(2)},
				SourceHash:   "hash-a",
				WriteState:   "written",
				GeneratedAt:  generatedAt,
				WrittenAt:    &writtenAt,
				Metadata:     map[string]any{"legacy_snapshot_kind": "mirror_export"},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/status-projections?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status projections status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body statusProjectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode status projections: %v", err)
	}
	if body.Project.Key != "areamatrix" || len(body.Projections) != 1 {
		t.Fatalf("unexpected status projections response: %+v", body)
	}
	projection := body.Projections[0]
	if projection.TargetKind != "project_status_json" || projection.TargetURI != ".areaflow/status.json" {
		t.Fatalf("unexpected projection target: %+v", projection)
	}
	if projection.WriteState != "written" || projection.WrittenAt == "" || projection.SourceHash != "hash-a" {
		t.Fatalf("unexpected projection write state: %+v", projection)
	}
	if projection.Payload["version_count"] != float64(2) {
		t.Fatalf("unexpected projection payload: %+v", projection.Payload)
	}
}

func TestProjectStatusProjectionAuthorizationEndpoint(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	var capturedOptions project.StatusProjectionAuthorizationPreviewOptions
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		statusAuthorization: project.StatusProjectionAuthorizationPreview{
			Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
			Status:                         "needs_approval",
			Mode:                           "status_projection_apply_authorization_preview_v1",
			ClaimScope:                     "package_a_status_projection_preflight_only",
			NotReal100:                     true,
			Decision:                       "needs_explicit_approval",
			Message:                        "status projection apply requires an explicit authorization packet before writing the managed project",
			TargetKind:                     "project_status_json",
			TargetURI:                      ".areaflow/status.json",
			TargetPath:                     "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			SchemaURI:                      "schemas/status-projection.schema.json",
			ValidatorPreflight:             "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			ProtectedPathFingerprintSHA256: "protected-hash",
			SourceHash:                     "source-a",
			SummaryState:                   "mirroring",
			RequiredAuthorizationPhrase:    project.StatusProjectionApplyRequiredApprovalReason,
			Permission: project.StatusProjectionAuthorizationPermission{
				Capability:        "write_status",
				ResourceType:      "path",
				TargetURI:         ".areaflow/status.json",
				CapabilityAllowed: true,
				PathAllowed:       true,
				Allowed:           true,
				Reason:            "allowed",
			},
			Preimage: project.StatusProjectionPreimage{
				TargetPath:            "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
				Exists:                true,
				Readable:              true,
				SizeBytes:             42,
				SHA256:                "before-hash",
				SchemaStatus:          "legacy",
				LegacyShape:           true,
				MissingRequiredFields: []string{"schema_version"},
				Message:               "target uses legacy status projection shape",
			},
			WriteSet: []project.StatusProjectionWriteSetEntry{
				{
					TargetURI:                ".areaflow/status.json",
					TargetPath:               "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
					Operation:                "replace_or_create",
					Capability:               "write_status",
					ExpectedBeforeExists:     true,
					ExpectedBeforeSHA256:     "before-hash",
					ExpectedBeforeSizeBytes:  42,
					RequiresPreimageMatch:    true,
					RequiresSchemaValidation: true,
					RollbackAction:           "restore the captured preimage bytes for .areaflow/status.json",
					ProtectedPath:            true,
				},
			},
			RequiredPreflight:                      []string{"areaflow project status-projections areamatrix --json"},
			RequiredPacketFields:                   []string{"expected_before_sha256", "rollback_plan"},
			RequiredCapabilities:                   []string{"write_status"},
			ProtectedPaths:                         []string{".areaflow/status.json"},
			RollbackPlan:                           []string{"restore the captured preimage bytes for .areaflow/status.json"},
			BlockedBy:                              []string{"explicit_status_projection_apply_approval_missing"},
			ForbiddenActions:                       []string{"write_execution"},
			SafetyFacts:                            map[string]bool{"project_write_attempted": false, "execution_write_attempted": false},
			ApprovalRequired:                       true,
			ApprovalStatus:                         "missing",
			WouldWriteProjectFileAfterApproval:     true,
			WouldCreateCommandRequestAfterApproval: true,
			GeneratedAt:                            generatedAt,
		},
		statusAuthorizationHook: func(options project.StatusProjectionAuthorizationPreviewOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/status-projections/authorization?target_uri=.areaflow/status.json", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status projection authorization status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetURI != ".areaflow/status.json" {
		t.Fatalf("unexpected status projection authorization options: %+v", capturedOptions)
	}
	var body statusProjectionAuthorizationPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode status projection authorization: %v", err)
	}
	if body.Status != "needs_approval" || body.ApplyOpen || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted {
		t.Fatalf("unexpected status projection authorization response: %+v", body)
	}
	if body.ClaimScope != "package_a_status_projection_preflight_only" || !body.NotReal100 {
		t.Fatalf("expected status projection authorization guardrail fields: %+v", body)
	}
	if body.TargetURI != ".areaflow/status.json" || body.SchemaURI != "schemas/status-projection.schema.json" {
		t.Fatalf("unexpected authorization target/schema: %+v", body)
	}
	if body.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("unexpected protected path fingerprint: %+v", body)
	}
	if body.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("unexpected required authorization phrase: %q", body.RequiredAuthorizationPhrase)
	}
	if body.Preimage.SchemaStatus != "legacy" || !body.Preimage.LegacyShape {
		t.Fatalf("expected legacy preimage: %+v", body.Preimage)
	}
	if len(body.WriteSet) != 1 || body.WriteSet[0].ExpectedBeforeSHA256 != "before-hash" || !body.WriteSet[0].RequiresPreimageMatch {
		t.Fatalf("unexpected write set: %+v", body.WriteSet)
	}
}

func TestProjectStatusProjectionApplyGateEndpoint(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	var capturedOptions project.StatusProjectionApplyGateOptions
	authorization := project.StatusProjectionAuthorizationPreview{
		Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                         "needs_approval",
		Mode:                           "status_projection_apply_authorization_preview_v1",
		ClaimScope:                     "package_a_status_projection_preflight_only",
		NotReal100:                     true,
		Decision:                       "needs_explicit_approval",
		Message:                        "status projection apply requires an explicit authorization packet before writing the managed project",
		TargetKind:                     "project_status_json",
		TargetURI:                      ".areaflow/status.json",
		TargetPath:                     "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		ProtectedPathFingerprintSHA256: "protected-hash",
		SourceHash:                     "source-a",
		SummaryState:                   "mirroring",
		Preimage: project.StatusProjectionPreimage{
			TargetPath:   "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			Exists:       true,
			Readable:     true,
			SizeBytes:    42,
			SHA256:       "before-hash",
			SchemaStatus: "legacy",
			LegacyShape:  true,
			Message:      "target uses legacy status projection shape",
		},
		RequiredPacketFields: []string{"expected_before_sha256", "explicit_approval"},
		RequiredCapabilities: []string{"write_status"},
		ProtectedPaths:       []string{".areaflow/status.json"},
		ForbiddenActions:     []string{"write_execution"},
		SafetyFacts:          map[string]bool{"project_write_attempted": false, "execution_write_attempted": false},
		ApprovalRequired:     true,
		ApprovalStatus:       "missing",
		GeneratedAt:          generatedAt,
	}
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		statusApplyGate: project.StatusProjectionApplyGate{
			Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
			Status:                         "blocked",
			Mode:                           "status_projection_apply_gate_v1",
			ClaimScope:                     "package_a_status_projection_preflight_only",
			NotReal100:                     true,
			Decision:                       "no_go",
			Message:                        "status projection apply packet is blocked",
			TargetURI:                      ".areaflow/status.json",
			TargetPath:                     "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			Authorization:                  authorization,
			Items:                          []project.StatusProjectionApplyGateItem{{Key: "explicit_approval", Category: "approval", Status: "blocked", Expected: "true", Actual: "false", BlockedBy: []string{"explicit_status_projection_apply_approval_missing"}}},
			RequiredPacketFields:           []string{"expected_before_sha256", "explicit_approval"},
			RequiredCapabilities:           []string{"write_status"},
			ProtectedPaths:                 []string{".areaflow/status.json"},
			ForbiddenActions:               []string{"write_execution"},
			SafetyFacts:                    map[string]bool{"command_request_created": false, "status_projection_written": false, "project_write_attempted": false},
			ApplyCommandEligibleIsNotApply: true,
			RequiresSeparateApplyCommand:   true,
			ApprovalRequired:               true,
			ApprovalStatus:                 "missing_or_incomplete",
			GeneratedAt:                    generatedAt,
		},
		statusApplyGateHook: func(options project.StatusProjectionApplyGateOptions) {
			capturedOptions = options
		},
	})

	query := url.Values{}
	query.Set("target_uri", ".areaflow/status.json")
	query.Set("expected_before_exists", "true")
	query.Set("expected_before_sha256", "before-hash")
	query.Set("expected_before_size", "42")
	query.Set("source_hash", "source-a")
	query.Set("schema_uri", "schemas/status-projection.schema.json")
	query.Set("validator_preflight", authorization.ValidatorPreflight)
	query.Set("protected_path_check", "git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json")
	query.Set("protected_path_fingerprint_sha256", "protected-hash")
	query.Set("rollback_action", "restore the captured preimage bytes for .areaflow/status.json")
	query.Set("accept_preimage_schema", "legacy")
	query.Set("explicit_approval", "true")
	query.Set("approval_actor", "as")
	query.Set("approval_reason", "approve real status projection schema migration")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/status-projections/apply-gate?"+query.Encode(), nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status projection apply gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetURI != ".areaflow/status.json" || capturedOptions.ExpectedBeforeExists == nil || !*capturedOptions.ExpectedBeforeExists {
		t.Fatalf("unexpected target/preimage exists options: %+v", capturedOptions)
	}
	if capturedOptions.ExpectedBeforeSizeBytes == nil || *capturedOptions.ExpectedBeforeSizeBytes != 42 || capturedOptions.ExpectedBeforeSHA256 != "before-hash" {
		t.Fatalf("unexpected expected-before options: %+v", capturedOptions)
	}
	if capturedOptions.SourceHash != "source-a" || capturedOptions.SchemaURI != "schemas/status-projection.schema.json" || capturedOptions.AcceptedPreimageSchemaStatus != "legacy" {
		t.Fatalf("unexpected packet options: %+v", capturedOptions)
	}
	if capturedOptions.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("unexpected protected path fingerprint option: %+v", capturedOptions)
	}
	if !capturedOptions.ExplicitApproval || capturedOptions.ApprovalActor != "as" || capturedOptions.ApprovalReason == "" {
		t.Fatalf("unexpected approval options: %+v", capturedOptions)
	}
	var body statusProjectionApplyGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode status projection apply gate: %v", err)
	}
	if body.Status != "blocked" || body.Decision != "no_go" || body.ApplyCommandEligible {
		t.Fatalf("unexpected apply gate response: %+v", body)
	}
	if body.ClaimScope != "package_a_status_projection_preflight_only" || !body.NotReal100 || !body.ApplyCommandEligibleIsNotApply || !body.RequiresSeparateApplyCommand {
		t.Fatalf("expected apply gate non-apply guardrail fields: %+v", body)
	}
	if body.CommandRequestCreated || body.StatusProjectionWritten || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted {
		t.Fatalf("apply gate must be read-only: %+v", body)
	}
	if body.Authorization.Preimage.SchemaStatus != "legacy" || len(body.Items) != 1 {
		t.Fatalf("unexpected nested apply gate response: %+v", body)
	}
}

func TestProjectStatusProjectionApplyPacketEndpoint(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	var capturedOptions project.StatusProjectionApplyPacketPreviewOptions
	authorization := project.StatusProjectionAuthorizationPreview{
		Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                         "needs_approval",
		Mode:                           "status_projection_apply_authorization_preview_v1",
		ClaimScope:                     "package_a_status_projection_preflight_only",
		NotReal100:                     true,
		Decision:                       "needs_explicit_approval",
		TargetKind:                     "project_status_json",
		TargetURI:                      ".areaflow/status.json",
		TargetPath:                     "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		ProtectedPathFingerprintSHA256: "protected-hash",
		SourceHash:                     "source-a",
		Preimage: project.StatusProjectionPreimage{
			Exists:       true,
			SizeBytes:    42,
			SHA256:       "before-hash",
			SchemaStatus: "legacy",
		},
		GeneratedAt: generatedAt,
	}
	gate := project.StatusProjectionApplyGate{
		Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                         "pass",
		Mode:                           "status_projection_apply_gate_v1",
		ClaimScope:                     "package_a_status_projection_preflight_only",
		NotReal100:                     true,
		Decision:                       "go",
		TargetURI:                      ".areaflow/status.json",
		Authorization:                  authorization,
		RequiredAuthorizationPhrase:    project.StatusProjectionApplyRequiredApprovalReason,
		ApplyCommandEligible:           true,
		ApplyCommandEligibleIsNotApply: true,
		RequiresSeparateApplyCommand:   true,
		ApprovalRequired:               true,
		ApprovalStatus:                 "approved",
		GeneratedAt:                    generatedAt,
	}
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		statusApplyPacket: project.StatusProjectionApplyPacketPreview{
			Project:                     project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
			Status:                      "ready",
			Mode:                        "status_projection_apply_packet_preview_v1",
			ClaimScope:                  "package_a_status_projection_preflight_only",
			NotReal100:                  true,
			Decision:                    "ready_for_apply_command",
			Blockers:                    []string{"explicit_status_projection_apply_approval_missing"},
			RequiredAuthorizationPhrase: project.StatusProjectionApplyRequiredApprovalReason,
			Authorization:               authorization,
			Gate:                        gate,
			Packet: project.StatusProjectionApplyPacket{
				TargetURI:                      ".areaflow/status.json",
				ExpectedBeforeExists:           true,
				ExpectedBeforeSHA256:           "before-hash",
				ExpectedBeforeSizeBytes:        42,
				SourceHash:                     "source-a",
				SchemaURI:                      "schemas/status-projection.schema.json",
				ValidatorPreflight:             authorization.ValidatorPreflight,
				ProtectedPathCheck:             "git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json",
				ProtectedPathFingerprintSHA256: "protected-hash",
				RollbackAction:                 "restore the captured preimage bytes for .areaflow/status.json",
				AcceptedPreimageSchemaStatus:   "legacy",
				ExplicitApproval:               true,
				ApprovalActor:                  "as",
				ApprovalReason:                 "approve status projection apply",
				RequiredAuthorizationPhrase:    project.StatusProjectionApplyRequiredApprovalReason,
			},
			ApplyCommand:        []string{"areaflow", "project", "status-projection-apply", "areamatrix", "--explicit-approval"},
			APIRequest:          project.StatusProjectionApplyAPIRequest{TargetURI: ".areaflow/status.json", SourceHash: "source-a", ProtectedPathFingerprintSHA256: "protected-hash", ExplicitApproval: true, ApprovalActor: "as", RequiredAuthorizationPhrase: project.StatusProjectionApplyRequiredApprovalReason},
			RequiredHumanReview: []string{"review target preimage schema status"},
			ForbiddenActions:    []string{"write_execution"},
			SafetyFacts:         map[string]bool{"project_write_attempted": false, "command_request_created": false},
			WouldCreateCommandRequestAfterApplyCommand: true,
			WouldWriteProjectFileAfterApplyCommand:     true,
			ApplyCommandEligibleIsNotApply:             true,
			RequiresSeparateApplyCommand:               true,
			GeneratedAt:                                generatedAt,
		},
		statusApplyPacketHook: func(options project.StatusProjectionApplyPacketPreviewOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/status-projections/apply-packet?target_uri=.areaflow/status.json&explicit_approval=true&approval_actor=as&approval_reason=approve", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status projection apply packet status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetURI != ".areaflow/status.json" || !capturedOptions.ExplicitApproval || capturedOptions.ApprovalActor != "as" {
		t.Fatalf("unexpected status projection apply packet options: %+v", capturedOptions)
	}
	var body statusProjectionApplyPacketPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode status projection apply packet: %v", err)
	}
	if body.Status != "ready" || body.Decision != "ready_for_apply_command" || !body.Packet.ExplicitApproval || body.Packet.SourceHash != "source-a" {
		t.Fatalf("unexpected apply packet response: %+v", body)
	}
	if body.ClaimScope != "package_a_status_projection_preflight_only" || !body.NotReal100 || !body.ApplyCommandEligibleIsNotApply || !body.RequiresSeparateApplyCommand {
		t.Fatalf("expected apply packet non-apply guardrail fields: %+v", body)
	}
	if len(body.Blockers) != 1 || body.Blockers[0] != "explicit_status_projection_apply_approval_missing" {
		t.Fatalf("expected top-level blockers in apply packet response: %+v", body.Blockers)
	}
	if body.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason ||
		body.Gate.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason ||
		body.Packet.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason ||
		body.APIRequest.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("expected required authorization phrase in apply packet response: %+v", body)
	}
	if body.Packet.ProtectedPathFingerprintSHA256 != "protected-hash" || body.APIRequest.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("unexpected apply packet fingerprint response: %+v", body)
	}
	if !body.Gate.ApplyCommandEligible || body.CommandRequestCreated || body.StatusProjectionWritten || body.ProjectWriteAttempted {
		t.Fatalf("apply packet preview must be read-only but command eligible: %+v", body)
	}
}

func TestProjectStatusProjectionApplyEndpoint(t *testing.T) {
	generatedAt := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	var capturedOptions project.ApplyStatusProjectionOptions
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		statusApply: project.ApplyStatusProjectionResult{
			Project:                   project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
			Status:                    "written",
			Decision:                  "allowed",
			Message:                   "status projection written",
			EventID:                   11,
			AuditEventID:              12,
			SnapshotID:                13,
			StatusProjectionID:        14,
			TargetKind:                "project_status_json",
			TargetURI:                 ".areaflow/status.json",
			WrittenTarget:             "/tmp/areamatrix/.areaflow/status.json",
			WriteHash:                 "hash-a",
			WriteSize:                 100,
			PreimageCaptured:          true,
			PreimageExists:            true,
			PreimageSHA256:            "before-hash",
			PreimageSize:              42,
			PostWriteVerified:         true,
			PostWriteSHA256:           "hash-a",
			PostWriteSize:             100,
			ProtectedPathsVerified:    true,
			ProtectedPathBeforeHash:   "protected-hash",
			ProtectedPathAfterHash:    "protected-hash",
			ExpectedProtectedPathHash: "protected-hash",
			RootContained:             true,
			StableProjectionValid:     true,
			AtomicReplaceUsed:         true,
			RollbackCompensation:      true,
			SourceHash:                "source-a",
			SummaryState:              "mirroring",
			ApplyGateStatus:           "pass",
			ApplyGateDecision:         "go",
			ApplyGateApprovalStatus:   "approved",
			ApplyCommandEligible:      true,
			IdempotencyKey:            "projection-key",
			Created:                   true,
			GeneratedAt:               generatedAt,
			ProjectWriteAttempted:     true,
			ExecutionWriteAttempted:   false,
			EngineCallAttempted:       false,
		},
		statusApplyHook: func(options project.ApplyStatusProjectionOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/status-projections/apply", strings.NewReader(`{"target_uri":".areaflow/status.json","actor":"local-user","reason":"fixture apply","idempotency_key":"projection-key","expected_before_exists":true,"expected_before_sha256":"before-hash","expected_before_size":42,"source_hash":"source-a","schema_uri":"schemas/status-projection.schema.json","validator_preflight":"python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json","protected_path_check":"git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json","protected_path_fingerprint_sha256":"protected-hash","rollback_action":"restore the captured preimage bytes for .areaflow/status.json","accept_preimage_schema":"legacy","explicit_approval":true,"approval_actor":"as","approval_reason":"approve status projection apply"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status projection apply status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.TargetURI != ".areaflow/status.json" || capturedOptions.Actor != "local-user" || capturedOptions.Reason != "fixture apply" || capturedOptions.IdempotencyKey != "projection-key" {
		t.Fatalf("unexpected status projection apply options: %+v", capturedOptions)
	}
	if capturedOptions.Gate.ExpectedBeforeExists == nil || !*capturedOptions.Gate.ExpectedBeforeExists || capturedOptions.Gate.ExpectedBeforeSHA256 != "before-hash" {
		t.Fatalf("unexpected apply gate preimage options: %+v", capturedOptions.Gate)
	}
	if capturedOptions.Gate.ExpectedBeforeSizeBytes == nil || *capturedOptions.Gate.ExpectedBeforeSizeBytes != 42 || capturedOptions.Gate.SourceHash != "source-a" {
		t.Fatalf("unexpected apply gate packet options: %+v", capturedOptions.Gate)
	}
	if capturedOptions.Gate.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("unexpected apply gate protected path fingerprint: %+v", capturedOptions.Gate)
	}
	if !capturedOptions.Gate.ExplicitApproval || capturedOptions.Gate.ApprovalActor != "as" || capturedOptions.Gate.ApprovalReason == "" {
		t.Fatalf("unexpected apply gate approval options: %+v", capturedOptions.Gate)
	}
	if capturedOptions.Writer == nil {
		t.Fatalf("expected status projection writer to be injected")
	}
	var body statusProjectionApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode status projection apply: %v", err)
	}
	if body.StatusProjectionID != 14 || !body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted {
		t.Fatalf("unexpected status projection apply response: %+v", body)
	}
	if !body.PreimageCaptured || !body.PreimageExists || body.PreimageSHA256 != "before-hash" || body.PreimageSize != 42 {
		t.Fatalf("unexpected status projection preimage response: %+v", body)
	}
	if !body.PostWriteVerified || body.PostWriteSHA256 != "hash-a" || body.PostWriteSize != 100 {
		t.Fatalf("unexpected status projection post-write response: %+v", body)
	}
	if !body.ProtectedPathsVerified || body.ProtectedPathBeforeHash != "protected-hash" || body.ProtectedPathAfterHash != "protected-hash" || body.ExpectedProtectedPathHash != "protected-hash" {
		t.Fatalf("unexpected status projection protected path response: %+v", body)
	}
	if !body.RootContained || !body.StableProjectionValid || !body.AtomicReplaceUsed || !body.RollbackCompensation {
		t.Fatalf("unexpected status projection write response: %+v", body)
	}
	if body.ApplyGateStatus != "pass" || body.ApplyGateDecision != "go" || !body.ApplyCommandEligible {
		t.Fatalf("unexpected status projection apply gate response: %+v", body)
	}
}

func TestProjectStatusProjectionApplyEndpointReturnsConflictWhenGateBlocksNonExactApprovalReason(t *testing.T) {
	generatedAt := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)
	var capturedOptions project.ApplyStatusProjectionOptions
	handler := NewHandler(fakeProjectStore{
		record: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		statusApply: project.ApplyStatusProjectionResult{
			Project:                 project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
			Status:                  "blocked",
			Decision:                "denied",
			Message:                 "status projection apply blocked by protected boundary",
			Blockers:                []string{"status_projection_apply_gate_blocked", "approval_reason_missing_or_mismatch"},
			EventID:                 11,
			AuditEventID:            12,
			SnapshotID:              13,
			StatusProjectionID:      14,
			TargetKind:              "project_status_json",
			TargetURI:               ".areaflow/status.json",
			SourceHash:              "source-a",
			SummaryState:            "mirroring",
			ApplyGateStatus:         "blocked",
			ApplyGateDecision:       "no_go",
			ApplyGateApprovalStatus: "missing_or_incomplete",
			ApplyCommandEligible:    false,
			IdempotencyKey:          "projection-key",
			Created:                 true,
			GeneratedAt:             generatedAt,
			ProjectWriteAttempted:   false,
			ExecutionWriteAttempted: false,
			EngineCallAttempted:     false,
		},
		statusApplyHook: func(options project.ApplyStatusProjectionOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/status-projections/apply", strings.NewReader(`{"target_uri":".areaflow/status.json","actor":"local-user","reason":"fixture apply","idempotency_key":"projection-key","expected_before_exists":true,"expected_before_sha256":"before-hash","expected_before_size":42,"source_hash":"source-a","schema_uri":"schemas/status-projection.schema.json","validator_preflight":"python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json","protected_path_check":"git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json","protected_path_fingerprint_sha256":"protected-hash","rollback_action":"restore the captured preimage bytes for .areaflow/status.json","accept_preimage_schema":"legacy","explicit_approval":true,"approval_actor":"as","approval_reason":"approve status projection apply"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("status projection blocked apply status = %d body=%s", resp.Code, resp.Body.String())
	}
	if !capturedOptions.Gate.ExplicitApproval || capturedOptions.Gate.ApprovalActor != "as" || capturedOptions.Gate.ApprovalReason != "approve status projection apply" {
		t.Fatalf("unexpected apply gate approval options: %+v", capturedOptions.Gate)
	}
	if capturedOptions.Writer == nil {
		t.Fatalf("expected status projection writer to be injected even though store gate blocks before calling it")
	}
	var body statusProjectionApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode blocked status projection apply: %v", err)
	}
	if body.Status != "blocked" || body.Decision != "denied" || body.ApplyCommandEligible {
		t.Fatalf("unexpected blocked status projection response: %+v", body)
	}
	if body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted {
		t.Fatalf("blocked status projection apply must not report write/execution/engine attempts: %+v", body)
	}
	if body.ApplyGateStatus != "blocked" || body.ApplyGateDecision != "no_go" || body.ApplyGateApprovalStatus != "missing_or_incomplete" {
		t.Fatalf("unexpected blocked apply gate facts: %+v", body)
	}
	if !containsString(body.Blockers, "approval_reason_missing_or_mismatch") {
		t.Fatalf("expected exact approval blocker in response: %+v", body.Blockers)
	}
}

func TestProjectEventsEndpoint(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		events: []project.EventRecord{
			{
				ID:       7,
				Type:     "project.doctor.completed",
				Severity: "info",
				Message:  "AreaFlow project doctor completed",
				Metadata: map[string]any{
					"overall_status": "pass",
				},
				CreatedAt: time.Date(2026, 6, 29, 2, 52, 5, 0, time.UTC),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/events?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode events response: %v", err)
	}
	if body.Project != "areamatrix" || len(body.Events) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body.Events[0].Type != "project.doctor.completed" {
		t.Fatalf("unexpected event: %+v", body.Events[0])
	}
}

func TestProjectDoctorEndpointRecordsCommand(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	var capturedOptions project.RecordDoctorReportOptions
	store := fakeProjectStore{
		record: record,
		doctorRecord: project.RecordDoctorReportResult{
			EventID:        7,
			Severity:       "warning",
			OverallStatus:  "warn",
			IdempotencyKey: "doctor-key",
			Created:        true,
		},
		doctorRecordHook: func(options project.RecordDoctorReportOptions) {
			capturedOptions = options
		},
	}
	server := Server{
		store: store,
		doctorRunner: func(_ context.Context, got project.Record, _ ProjectStore, allowNative bool) (doctor.Report, error) {
			if got.Key != "areamatrix" {
				t.Fatalf("project key = %q, want areamatrix", got.Key)
			}
			if !allowNative {
				t.Fatal("expected allow_native to be forwarded")
			}
			return doctor.Report{
				Project: "areamatrix",
				Profile: "areamatrix",
				Checks: []doctor.Check{{
					Name:    "hash_drift",
					Status:  doctor.StatusWarn,
					Message: "warn",
				}},
			}, nil
		},
	}
	handler := apiVersionAlias(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.projectHandler(w, r)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/doctor", strings.NewReader(`{
		"allow_native": true,
		"idempotency_key": "doctor-key",
		"actor": "local-user",
		"reason": "api doctor"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("doctor status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectDoctorRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode doctor response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.EventID != 7 || body.IdempotencyKey != "doctor-key" || !body.Created {
		t.Fatalf("unexpected doctor response: %+v", body)
	}
	if capturedOptions.IdempotencyKey != "doctor-key" || capturedOptions.Actor != "local-user" || capturedOptions.Reason != "api doctor" {
		t.Fatalf("unexpected doctor record options: %+v", capturedOptions)
	}
	if body.Report["overall_status"] != "warn" {
		t.Fatalf("unexpected report summary: %+v", body.Report)
	}
}

func TestProjectImportEndpointRunsImporter(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	var capturedOptions importer.Options
	server := Server{
		store: fakeProjectStore{record: record},
		importer: func(_ context.Context, got project.Record, options importer.Options) (importer.Result, error) {
			if got.Key != "areamatrix" {
				t.Fatalf("project key = %q, want areamatrix", got.Key)
			}
			capturedOptions = options
			return importer.Result{
				ProjectKey:     got.Key,
				Versions:       2,
				Residuals:      3,
				Artifacts:      4,
				ActiveTasks:    1,
				V1Done:         5,
				V1Total:        6,
				StatusSnapshot: "source-a",
				RunID:          9,
				IdempotencyKey: "import-key",
				Created:        true,
			}, nil
		},
	}
	handler := apiVersionAlias(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.projectHandler(w, r)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/import", strings.NewReader(`{
		"idempotency_key": "import-key",
		"actor": "local-user",
		"reason": "api import"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("import status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectImportRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Versions != 2 || body.RunID != 9 || body.IdempotencyKey != "import-key" || !body.Created {
		t.Fatalf("unexpected import response: %+v", body)
	}
	if capturedOptions.IdempotencyKey != "import-key" || capturedOptions.Actor != "local-user" || capturedOptions.Reason != "api import" {
		t.Fatalf("unexpected import options: %+v", capturedOptions)
	}
}

func TestProjectCutoverApplyEndpointRecordsCommand(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	version := project.WorkflowVersion{
		ID:              7,
		ProjectID:       1,
		DisplayLabel:    "v2",
		VersionKind:     "workflow_version",
		LifecycleStatus: "authoring_cutover",
		ImportMode:      "authored",
		StatusSummary:   map[string]any{"authoring_cutover": map[string]any{"applied": true}},
		CreatedAt:       time.Date(2026, 7, 1, 1, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 7, 1, 1, 5, 0, 0, time.UTC),
	}
	var capturedOptions project.ApplyCutoverOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		cutoverApply: project.ApplyCutoverResult{
			Project:                 record,
			Version:                 version,
			Status:                  "applied",
			Decision:                "allowed",
			Message:                 "authoring cutover applied in AreaFlow state",
			Warnings:                []string{"authoring cutover does not write AreaMatrix project files or execution state"},
			EventID:                 8,
			AuditEventID:            9,
			IdempotencyKey:          "cutover-key",
			Created:                 true,
			ProjectWriteAttempted:   false,
			ExecutionWriteAttempted: false,
			CutoverReadinessGateID:  6,
		},
		cutoverApplyHook: func(options project.ApplyCutoverOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/cutover-apply", strings.NewReader(`{
		"version": "v2",
		"idempotency_key": "cutover-key",
		"actor": "local-user",
		"reason": "api cutover",
		"mode": "authoring_cutover"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("cutover apply status = %d body=%s", resp.Code, resp.Body.String())
	}
	rawBody := resp.Body.String()
	var body projectCutoverApplyResponse
	if err := json.NewDecoder(strings.NewReader(rawBody)).Decode(&body); err != nil {
		t.Fatalf("decode cutover apply response: %v", err)
	}
	if !strings.Contains(rawBody, `"area_matrix_write_attempted":false`) {
		t.Fatalf("cutover apply response must expose AreaMatrix write proof: %s", rawBody)
	}
	if body.Project.Key != "areamatrix" || body.WorkflowVersion.DisplayLabel != "v2" || body.Decision != "allowed" || body.IdempotencyKey != "cutover-key" {
		t.Fatalf("unexpected cutover apply response: %+v", body)
	}
	if body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.AreaMatrixWriteAttempted {
		t.Fatalf("cutover apply response must not claim project/execution/AreaMatrix writes: %+v", body)
	}
	if capturedOptions.VersionLabel != "v2" || capturedOptions.IdempotencyKey != "cutover-key" || capturedOptions.Actor != "local-user" || capturedOptions.Reason != "api cutover" || capturedOptions.Mode != "authoring_cutover" {
		t.Fatalf("unexpected cutover apply options: %+v", capturedOptions)
	}
}

func TestAuditEventsEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		auditEvents: []project.AuditEventRecord{
			{
				ID:           9,
				ProjectID:    1,
				Action:       "project.upsert",
				Capability:   "register_project",
				ResourceType: "project",
				Resource:     "areamatrix",
				Decision:     "allowed",
				Reason:       "local CLI project registration",
				Metadata:     map[string]any{"source": "test"},
				CreatedAt:    created,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/audit-events?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body auditEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode audit events response: %v", err)
	}
	if body.ProjectKey != "" || len(body.AuditEvents) != 1 {
		t.Fatalf("unexpected audit events response: %+v", body)
	}
	event := body.AuditEvents[0]
	if event.Action != "project.upsert" || event.Decision != "allowed" || event.Resource != "areamatrix" {
		t.Fatalf("unexpected audit event: %+v", event)
	}
}

func TestAuditEventsEndpointSupportsProjectFilter(t *testing.T) {
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		auditEvents: []project.AuditEventRecord{
			{
				ID:        10,
				ProjectID: 1,
				Action:    "worker.register",
				Decision:  "allowed",
				Metadata:  map[string]any{},
				CreatedAt: time.Date(2026, 6, 29, 3, 5, 0, 0, time.UTC),
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/audit-events?project_key=areamatrix&limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body auditEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode audit events response: %v", err)
	}
	if body.ProjectKey != "areamatrix" || len(body.AuditEvents) != 1 {
		t.Fatalf("unexpected filtered audit events response: %+v", body)
	}
	if body.AuditEvents[0].Action != "worker.register" {
		t.Fatalf("unexpected filtered audit event: %+v", body.AuditEvents[0])
	}
}

func TestAuditCoverageEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 11, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		auditCover: project.AuditCoverage{
			Status:              "warn",
			Mode:                "read_only_audit_coverage",
			Scope:               "project",
			ProjectID:           1,
			ProjectKey:          "areamatrix",
			TotalAuditEvents:    2,
			CoveredRequirements: 1,
			GapRequirements:     1,
			Requirements: []project.AuditCoverageRequirement{
				{
					Key:           "project_registration",
					Category:      "write",
					Description:   "project writes are audited",
					Status:        "pass",
					EvidenceCount: 1,
					RequiredActions: []project.AuditCoverageActionEvidence{
						{Action: "project.upsert", Decision: "allowed", Count: 1, Status: "pass", LastAuditAt: &created},
					},
					LastAuditAt: &created,
				},
				{
					Key:           "secret_resolution",
					Category:      "secret",
					Description:   "secret resolution is audited",
					Status:        "gap",
					EvidenceCount: 0,
					RequiredActions: []project.AuditCoverageActionEvidence{
						{Action: "secret.resolve", Count: 0, Status: "gap"},
					},
					MissingActions: []string{"secret.resolve"},
				},
			},
			GeneratedAt: created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/coverage?project_key=areamatrix", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("audit coverage status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body auditCoverageResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode audit coverage response: %v", err)
	}
	if body.Status != "warn" || body.Scope != "project" || body.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected audit coverage: %+v", body)
	}
	if body.CoveredRequirements != 1 || body.GapRequirements != 1 || len(body.Requirements) != 2 {
		t.Fatalf("unexpected audit coverage counts: %+v", body)
	}
	if body.Requirements[0].RequiredActions[0].LastAuditAt == "" {
		t.Fatalf("expected action evidence timestamp: %+v", body.Requirements[0])
	}
	if len(body.Requirements[1].MissingActions) != 1 || body.Requirements[1].MissingActions[0] != "secret.resolve" {
		t.Fatalf("unexpected missing actions: %+v", body.Requirements[1])
	}
}

func TestPermissionPolicyDoctorEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		permission: project.PermissionPolicyDoctor{
			Status: "pass",
			Mode:   "read_only_permission_policy_doctor",
			Project: project.Record{
				ID:              1,
				Key:             "areamatrix",
				Name:            "AreaMatrix",
				Adapter:         "areamatrix",
				WorkflowProfile: "areamatrix",
			},
			Checks: []project.PermissionPolicyCheck{
				{
					Key:      "default_read_only",
					Category: "capability",
					Status:   "pass",
					Message:  "high-risk capabilities are disabled by default",
					Metadata: map[string]any{},
				},
				{
					Key:      "status_export_write",
					Category: "path",
					Status:   "pass",
					Message:  "status export path is explicitly allowed and not denied",
					Metadata: map[string]any{"path": ".areaflow/status.json"},
				},
			},
			GeneratedAt: created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/permissions/doctor?project_key=areamatrix", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("permission doctor status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body permissionPolicyDoctorResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode permission doctor response: %v", err)
	}
	if body.Status != "pass" || body.Mode != "read_only_permission_policy_doctor" || body.Project.Key != "areamatrix" {
		t.Fatalf("unexpected permission doctor: %+v", body)
	}
	if len(body.Checks) != 2 || body.Checks[1].Metadata["path"] != ".areaflow/status.json" {
		t.Fatalf("unexpected permission checks: %+v", body.Checks)
	}
	if body.GeneratedAt == "" {
		t.Fatalf("expected generated_at: %+v", body)
	}
}

func TestPermissionPolicyDoctorEndpointRequiresProjectKey(t *testing.T) {
	handler := NewHandler(fakeProjectStore{})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/permissions/doctor", nil))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("permission doctor status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestArtifactIntegrityEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		integrity: project.ArtifactIntegrityReport{
			Status:           "warn",
			Mode:             "read_only_artifact_integrity",
			Project:          project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
			CheckedArtifacts: 2,
			PassedArtifacts:  1,
			SkippedArtifacts: 1,
			GeneratedAt:      created,
			Checks: []project.ArtifactIntegrityCheck{
				{
					Artifact: project.ArtifactRecord{ID: 7, ProjectID: 1, ArtifactType: "runner_preview_report", StorageBackend: "local", URI: "/tmp/report.json", SHA256: "abc123", SizeBytes: 12},
					Status:   "pass",
					Message:  "local artifact hash and size match metadata",
					Metadata: map[string]any{"read_contents": true},
				},
				{
					Artifact: project.ArtifactRecord{ID: 8, ProjectID: 1, ArtifactType: "source_ref", StorageBackend: "external_project", URI: "workflow/file.md", SourcePath: "workflow/file.md", SHA256: "def456", SizeBytes: 20},
					Status:   "skipped",
					Message:  "referenced project artifact content remains in managed project",
					Metadata: map[string]any{"read_contents": false},
				},
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/integrity?project_key=areamatrix", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("artifact integrity status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body artifactIntegrityResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode artifact integrity: %v", err)
	}
	if body.Status != "warn" || body.Mode != "read_only_artifact_integrity" || body.Project.Key != "areamatrix" {
		t.Fatalf("unexpected artifact integrity response: %+v", body)
	}
	if body.CheckedArtifacts != 2 || body.PassedArtifacts != 1 || body.SkippedArtifacts != 1 {
		t.Fatalf("unexpected counters: %+v", body)
	}
	if len(body.Checks) != 2 || body.Checks[0].Artifact.ArtifactType != "runner_preview_report" || body.Checks[1].Status != "skipped" {
		t.Fatalf("unexpected checks: %+v", body.Checks)
	}
}

func TestArtifactIntegrityEndpointRequiresProjectKey(t *testing.T) {
	handler := NewHandler(fakeProjectStore{})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/artifacts/integrity", nil))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("artifact integrity status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestArtifactArchivePreviewEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 9, 30, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	var got project.ArtifactArchivePreviewOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		archivePreviewHook: func(options project.ArtifactArchivePreviewOptions) {
			got = options
		},
		archivePreview: project.ArtifactArchivePreviewResult{
			Project: record,
			Status:  "needs_attention",
			Mode:    "metadata_only_archive_preview",
			Summary: project.ArtifactArchivePreviewSummary{
				TotalArtifacts:    2,
				ArchiveCandidates: 1,
				ExternalRefs:      1,
			},
			Items: []project.ArtifactArchivePreviewItem{{
				ArtifactID:     7,
				ArtifactType:   "runner_preview_report",
				StorageBackend: "local",
				RetentionClass: "ephemeral",
				ArchiveState:   "archive_candidate",
				Action:         "eligible_for_future_gc_preview",
				Decision:       "preview_only",
				Reason:         "ephemeral artifacts may be cleaned only by a future explicit GC command",
			}, {
				ArtifactID:     8,
				ArtifactType:   "source_ref",
				StorageBackend: "project_reference",
				RetentionClass: "external_ref",
				ArchiveState:   "metadata_only_reference",
				Action:         "keep_metadata_only",
				Decision:       "requires_archive_ownership_decision",
				Reason:         "project reference originals remain in the managed project and are not copied or deleted",
			}},
			EventID:                 12,
			AuditEventID:            13,
			IdempotencyKey:          "artifact.archive.preview:test",
			Created:                 true,
			GeneratedAt:             created,
			ProjectWriteAttempted:   false,
			StorageWriteAttempted:   false,
			ArtifactDeleteAttempted: false,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/artifacts/archive-preview", strings.NewReader(`{"retention_class":"ephemeral","limit":20,"actor":"local-user","reason":"preview","idempotency_key":"artifact.archive.preview:test"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("artifact archive preview status = %d body=%s", resp.Code, resp.Body.String())
	}
	if got.RetentionClass != "ephemeral" || got.Limit != 20 || got.Actor != "local-user" || got.Reason != "preview" || got.IdempotencyKey != "artifact.archive.preview:test" {
		t.Fatalf("unexpected archive preview options: %+v", got)
	}
	var body artifactArchivePreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode artifact archive preview: %v", err)
	}
	if body.Status != "needs_attention" || body.Mode != "metadata_only_archive_preview" || body.Summary.ArchiveCandidates != 1 || len(body.Items) != 2 {
		t.Fatalf("unexpected archive preview response: %+v", body)
	}
	if body.ProjectWriteAttempted || body.StorageWriteAttempted || body.ArtifactDeleteAttempted {
		t.Fatalf("archive preview attempted forbidden action: %+v", body)
	}
}

func TestConformanceEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		conformance: project.ConformanceReport{
			Status:      "pass",
			Mode:        "read_only_adapter_profile_conformance",
			Project:     project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
			ProfileID:   "areamatrix",
			Adapter:     "areamatrix",
			ProfileHash: "abc123",
			StageCount:  16,
			GateCount:   17,
			GeneratedAt: created,
			Checks: []project.ConformanceCheck{
				{
					Key:      "project_adapter_profile",
					Category: "binding",
					Status:   "pass",
					Message:  "project adapter/profile binding matches loaded workflow profile defaults",
					Metadata: map[string]any{"profile_id": "areamatrix"},
				},
				{
					Key:      "adapter_snapshot",
					Category: "adapter",
					Status:   "pass",
					Message:  "AreaMatrix adapter can load a read-only project snapshot",
					Metadata: map[string]any{"versions": 2},
				},
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conformance?project_key=areamatrix", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("conformance status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body conformanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode conformance: %v", err)
	}
	if body.Status != "pass" || body.Mode != "read_only_adapter_profile_conformance" || body.Project.Key != "areamatrix" {
		t.Fatalf("unexpected conformance response: %+v", body)
	}
	if body.ProfileID != "areamatrix" || body.Adapter != "areamatrix" || body.StageCount != 16 || body.GateCount != 17 {
		t.Fatalf("unexpected conformance summary: %+v", body)
	}
	if len(body.Checks) != 2 || body.Checks[0].Key != "project_adapter_profile" || body.Checks[1].Metadata["versions"] != float64(2) {
		t.Fatalf("unexpected conformance checks: %+v", body.Checks)
	}
}

func TestConformanceEndpointRequiresProjectKey(t *testing.T) {
	handler := NewHandler(fakeProjectStore{})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/v1/conformance", nil))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("conformance status = %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestProjectEventsEndpointRejectsInvalidLimit(t *testing.T) {
	handler := NewHandler(fakeProjectStore{record: project.Record{ID: 1, Key: "areamatrix"}})
	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/events?limit=0", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestProjectEventsEndpointReportsMissingProject(t *testing.T) {
	handler := NewHandler(fakeProjectStore{err: errors.New("not found")})
	req := httptest.NewRequest(http.MethodGet, "/api/projects/missing/events", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}

func TestGlobalEventStreamEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 6, 0, 0, 0, time.UTC)
	var gotFilter project.EventStreamFilter
	handler := NewHandler(fakeProjectStore{
		streamHook: func(filter project.EventStreamFilter) {
			gotFilter = filter
		},
		events: []project.EventRecord{{
			ID:        10,
			ProjectID: 1,
			Type:      "project.import.completed",
			Severity:  "info",
			Message:   "AreaMatrix metadata import completed",
			Metadata:  map[string]any{"versions": float64(2)},
			CreatedAt: created,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/events/stream?once=true&after_id=9&limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("event stream status = %d body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", got)
	}
	if gotFilter.AfterID != 9 || gotFilter.Limit != 1 || gotFilter.ProjectID != 0 || gotFilter.RunID != 0 {
		t.Fatalf("unexpected stream filter: %+v", gotFilter)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "id: 10") || !strings.Contains(body, "event: project.import.completed") || !strings.Contains(body, `"project_id":1`) {
		t.Fatalf("unexpected SSE body: %s", body)
	}
}

func TestProjectEventStreamEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 6, 5, 0, 0, time.UTC)
	var gotFilter project.EventStreamFilter
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 3, Key: "areamatrix"},
		streamHook: func(filter project.EventStreamFilter) {
			gotFilter = filter
		},
		events: []project.EventRecord{{
			ID:        11,
			ProjectID: 3,
			Type:      "project.doctor.completed",
			Severity:  "info",
			Message:   "AreaFlow project doctor completed",
			Metadata:  map[string]any{"overall_status": "pass"},
			CreatedAt: created,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/events/stream?once=true", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project event stream status = %d body=%s", resp.Code, resp.Body.String())
	}
	if gotFilter.ProjectID != 3 || gotFilter.RunID != 0 {
		t.Fatalf("unexpected project stream filter: %+v", gotFilter)
	}
	if body := resp.Body.String(); !strings.Contains(body, "event: project.doctor.completed") {
		t.Fatalf("unexpected project SSE body: %s", body)
	}
}

func TestRunEventStreamEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 6, 10, 0, 0, time.UTC)
	var gotFilter project.EventStreamFilter
	handler := NewHandler(fakeProjectStore{
		streamHook: func(filter project.EventStreamFilter) {
			gotFilter = filter
		},
		events: []project.EventRecord{{
			ID:        12,
			ProjectID: 3,
			RunID:     4,
			Type:      "worker.run_once.completed",
			Severity:  "info",
			Message:   "Worker run-once dry-run completed",
			Metadata:  map[string]any{"run_task_id": float64(5)},
			CreatedAt: created,
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/runs/4/events/stream?once=true", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("run event stream status = %d body=%s", resp.Code, resp.Body.String())
	}
	if gotFilter.RunID != 4 || gotFilter.ProjectID != 0 {
		t.Fatalf("unexpected run stream filter: %+v", gotFilter)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "event: worker.run_once.completed") || !strings.Contains(body, `"run_id":4`) {
		t.Fatalf("unexpected run SSE body: %s", body)
	}
}

func TestWorkflowVersionsEndpointListsVersions(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		versions: []project.WorkflowVersion{
			{
				ID:              7,
				DisplayLabel:    "v2",
				VersionKind:     "workflow_version",
				LifecycleStatus: "draft",
				ImportMode:      "authored",
				StatusSummary: map[string]any{
					"phase": "v0.3a",
					"profile_binding": map[string]any{
						"profile_id":      "areamatrix",
						"profile_version": float64(0),
						"profile_hash":    "abc123",
					},
				},
				CreatedAt: created,
				UpdatedAt: created,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow versions response: %v", err)
	}
	if body.Project.Key != "areamatrix" || len(body.WorkflowVersions) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body.WorkflowVersions[0].ImportMode != "authored" {
		t.Fatalf("unexpected version: %+v", body.WorkflowVersions[0])
	}
}

func TestWorkflowVersionsEndpointCreatesVersion(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		record: record,
		create: project.CreateWorkflowVersionResult{
			Project: record,
			Version: project.WorkflowVersion{
				ID:              7,
				DisplayLabel:    "v2",
				VersionKind:     "workflow_version",
				LifecycleStatus: "draft",
				ImportMode:      "authored",
				StatusSummary: map[string]any{
					"phase": "v0.3a",
					"profile_binding": map[string]any{
						"profile_id":      "areamatrix",
						"profile_version": float64(0),
						"profile_hash":    "abc123",
					},
				},
				CreatedAt: created,
				UpdatedAt: created,
			},
			InitialItem: project.WorkflowItem{
				ID:                9,
				WorkflowVersionID: 7,
				Stage:             "version_init",
				ItemType:          "workflow_version_candidate",
				ExternalKey:       "v2:version_init",
				Status:            "draft",
				Metadata:          map[string]any{"phase": "v0.3a"},
				CreatedAt:         created,
				UpdatedAt:         created,
			},
			StageItems: []project.WorkflowItem{
				{
					ID:                10,
					WorkflowVersionID: 7,
					Stage:             "discussion",
					ItemType:          "discussion_package",
					ExternalKey:       "v2:discussion:discussion_package",
					Status:            "draft",
					Metadata:          map[string]any{"phase": "v0.3b"},
					CreatedAt:         created,
					UpdatedAt:         created,
				},
			},
			Created:        true,
			IdempotencyKey: "create-v2",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workflow-versions", strings.NewReader(`{
		"display_label": "v2",
		"idempotency_key": "create-v2",
		"actor": "local-user",
		"reason": "create v2"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow version create response: %v", err)
	}
	if body.WorkflowVersion.DisplayLabel != "v2" || body.InitialItem.Stage != "version_init" {
		t.Fatalf("unexpected response: %+v", body)
	}
	if len(body.StageItems) != 1 || body.StageItems[0].Stage != "discussion" {
		t.Fatalf("unexpected stage items: %+v", body.StageItems)
	}
	binding, ok := body.WorkflowVersion.StatusSummary["profile_binding"].(map[string]any)
	if !ok || binding["profile_id"] != "areamatrix" || binding["profile_hash"] != "abc123" {
		t.Fatalf("missing profile binding: %+v", body.WorkflowVersion.StatusSummary)
	}
}

func TestWorkflowVersionsEndpointShowsVersion(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		version: project.WorkflowVersion{
			ID:              7,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "draft",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"phase": "v0.3a"},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions/v2", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow version response: %v", err)
	}
	if body.DisplayLabel != "v2" || body.ImportMode != "authored" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestWorkflowVersionStagesEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		version: project.WorkflowVersion{
			ID:              7,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "draft",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"phase": "v0.3a"},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		items: []project.WorkflowItem{
			{
				ID:                9,
				WorkflowVersionID: 7,
				Stage:             "version_init",
				ItemType:          "workflow_version_candidate",
				ExternalKey:       "v2:version_init",
				Status:            "draft",
				Metadata:          map[string]any{"phase": "v0.3a"},
				CreatedAt:         created,
				UpdatedAt:         created,
			},
			{
				ID:                10,
				WorkflowVersionID: 7,
				Stage:             "drafts",
				ItemType:          "draft_verify",
				ExternalKey:       "v2:drafts:draft_verify",
				Status:            "blocked",
				Metadata:          map[string]any{"phase": "v0.3b"},
				CreatedAt:         created,
				UpdatedAt:         created,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions/v2/stages", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body workflowVersionStagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode workflow version stages response: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("item count = %d, want 2", len(body.Items))
	}
	if body.Items[1].ItemType != "draft_verify" {
		t.Fatalf("unexpected item: %+v", body.Items[1])
	}
}

func TestWorkflowVersionStagesEndpointEnsuresSkeleton(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix"}
	version := project.WorkflowVersion{
		ID:              7,
		DisplayLabel:    "v2",
		VersionKind:     "workflow_version",
		LifecycleStatus: "draft",
		ImportMode:      "authored",
		StatusSummary:   map[string]any{"phase": "v0.3a"},
		CreatedAt:       created,
		UpdatedAt:       created,
	}
	handler := NewHandler(fakeProjectStore{
		record:  record,
		version: version,
		ensure: project.EnsureStageSkeletonResult{
			Project: record,
			Version: version,
			Items: []project.WorkflowItem{
				{
					ID:                10,
					WorkflowVersionID: 7,
					Stage:             "discussion",
					ItemType:          "discussion_package",
					ExternalKey:       "v2:discussion:discussion_package",
					Status:            "draft",
					Metadata:          map[string]any{"phase": "v0.3b"},
					CreatedAt:         created,
					UpdatedAt:         created,
				},
			},
			Links: []project.WorkflowItemLink{
				{
					ID:                20,
					WorkflowVersionID: 7,
					FromItemID:        10,
					ToItemID:          11,
					RelationType:      "derives_from",
					Metadata:          map[string]any{"source": "stage_skeleton"},
					CreatedAt:         created,
				},
			},
			Created: 1,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workflow-versions/v2/stages", strings.NewReader(`{
		"actor": "local-user",
		"reason": "ensure skeleton"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body ensureStageSkeletonResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode ensure stage skeleton response: %v", err)
	}
	if body.Created != 1 || len(body.Items) != 1 || len(body.Links) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body.Links[0].RelationType != "derives_from" || body.Links[0].FromItemID != 10 || body.Links[0].ToItemID != 11 {
		t.Fatalf("unexpected skeleton link: %+v", body.Links[0])
	}
}

func TestWorkflowVersionGatesEndpointRunsWorkflowGate(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		version: project.WorkflowVersion{
			ID:              7,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "draft",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"phase": "v0.3a"},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		gate: project.GateResult{
			ID:                3,
			ProjectID:         1,
			WorkflowVersionID: 7,
			WorkflowItemID:    10,
			GateName:          "plan_doctor",
			ScopeType:         "workflow_version",
			ScopeID:           "v2",
			Status:            "fail",
			Inputs:            map[string]any{"item_count": float64(10)},
			SourceHashes:      map[string]any{},
			Failures:          []string{"plan artifact is placeholder-only"},
			Metadata:          map[string]any{"phase": "v0.3c"},
			CheckedAt:         created,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workflow-versions/v2/gates", strings.NewReader(`{
		"gate_name": "plan_doctor",
		"actor": "local-user",
		"reason": "api gate smoke"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body gateResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode gate result response: %v", err)
	}
	if body.GateName != "plan_doctor" || body.Status != "fail" {
		t.Fatalf("unexpected gate response: %+v", body)
	}
}

func TestWorkflowVersionGatesEndpointListsGateResults(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		version: project.WorkflowVersion{
			ID:              7,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "draft",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"phase": "v0.3a"},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		gates: []project.GateResult{
			{
				ID:                3,
				ProjectID:         1,
				WorkflowVersionID: 7,
				GateName:          "discussion_gate",
				ScopeType:         "workflow_version",
				ScopeID:           "v2",
				Status:            "warn",
				Inputs:            map[string]any{"item_count": float64(10)},
				SourceHashes:      map[string]any{},
				Warnings:          []string{"discussion artifact is placeholder-only"},
				Metadata:          map[string]any{"phase": "v0.3c"},
				CheckedAt:         created,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/workflow-versions/v2/gates?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body gateResultsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode gate results response: %v", err)
	}
	if len(body.GateResults) != 1 || body.GateResults[0].GateName != "discussion_gate" {
		t.Fatalf("unexpected gate results response: %+v", body)
	}
}

func TestWorkflowVersionTransitionPreviewsEndpointCreatesPreview(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 40, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		version: project.WorkflowVersion{
			ID:              7,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "draft",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"phase": "v0.3a"},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		preview: project.WorkflowTransitionPreview{
			ID:                4,
			ProjectID:         1,
			WorkflowVersionID: 7,
			FromStage:         "promotion_preview",
			ToStage:           "approval",
			Status:            "blocked",
			RequiredGateName:  "promotion_preview",
			Blockers:          []string{"latest promotion_preview gate status is fail"},
			Warnings:          []string{"transition preview is read-only"},
			Metadata:          map[string]any{"phase": "v0.3d"},
			CreatedAt:         created,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workflow-versions/v2/transition-previews", strings.NewReader(`{
		"actor": "local-user",
		"reason": "api preview smoke"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body transitionPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode transition preview response: %v", err)
	}
	if body.Status != "blocked" || body.RequiredGateName != "promotion_preview" {
		t.Fatalf("unexpected transition preview response: %+v", body)
	}
}

func TestWorkflowVersionApprovalsEndpointRecordsDecision(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 45, 0, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		version: project.WorkflowVersion{
			ID:              7,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "draft",
			ImportMode:      "authored",
			StatusSummary:   map[string]any{"phase": "v0.3a"},
			CreatedAt:       created,
			UpdatedAt:       created,
		},
		approval: project.ApprovalRecord{
			ID:                  5,
			ProjectID:           1,
			WorkflowVersionID:   7,
			TransitionPreviewID: 4,
			ApprovalKind:        "workflow_transition",
			Decision:            "rejected",
			ScopeType:           "workflow_version",
			ScopeID:             "v2",
			Actor:               "local-user",
			Reason:              "blocked preview",
			RiskLevel:           "normal",
			Metadata:            map[string]any{"phase": "v0.3d"},
			CreatedAt:           created,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects/areamatrix/workflow-versions/v2/approvals", strings.NewReader(`{
		"decision": "rejected",
		"transition_preview_id": 4,
		"actor": "local-user",
		"reason": "blocked preview"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body approvalRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode approval response: %v", err)
	}
	if body.Decision != "rejected" || body.TransitionPreviewID != 4 {
		t.Fatalf("unexpected approval response: %+v", body)
	}
}

func TestProjectCompatibilityEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		record: record,
		compat: project.CompatibilityContract{
			Project: record,
			Status:  "pass",
			Commands: []project.CompatibilityCommand{
				{
					Command:        "./dev workflow status",
					Mode:           "forward",
					Status:         "pass",
					AreaFlowTarget: "areaflow project summary",
					Fallback:       ".areaflow/status.json",
					Message:        "command can forward to AreaFlow",
					Metadata:       map[string]any{"read_only": true},
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/compatibility", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body compatibilityContractResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode compatibility response: %v", err)
	}
	if body.Status != "pass" || len(body.Commands) != 1 {
		t.Fatalf("unexpected compatibility response: %+v", body)
	}
	if body.Commands[0].Command != "./dev workflow status" {
		t.Fatalf("unexpected command response: %+v", body.Commands[0])
	}
}

func TestProjectShimPreviewEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	contract := project.CompatibilityContract{
		Project: record,
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:       "./task-loop run",
				Mode:          "blocked",
				Status:        "pass",
				BlockedReason: "execution and task-loop replacement are out of v0.4 scope",
				Message:       "command is intentionally blocked",
				Metadata:      map[string]any{"read_only": false},
			},
		},
	}
	handler := NewHandler(fakeProjectStore{
		record:      record,
		shimPreview: project.ShimPreviewFromCompatibility(contract),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/shim-preview", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body shimPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode shim preview response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Mode != "read_only_planning" {
		t.Fatalf("unexpected shim preview response: %+v", body)
	}
	if len(body.PlannedFiles) == 0 || body.PlannedFiles[0].Path != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected shim preview planned files: %+v", body.PlannedFiles)
	}
	if len(body.CommandMappings) != 1 || body.CommandMappings[0].Mode != "blocked" {
		t.Fatalf("unexpected shim preview mappings: %+v", body.CommandMappings)
	}
}

func TestProjectShimReadinessEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	contract := project.CompatibilityContract{
		Project: record,
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:       "./task-loop run",
				Mode:          "blocked",
				Status:        "pass",
				BlockedReason: "execution and task-loop replacement are out of v0.4 scope",
				Metadata:      map[string]any{"read_only": false},
			},
		},
	}
	preview := project.ShimPreviewFromCompatibility(contract)
	handler := NewHandler(fakeProjectStore{
		record:        record,
		shimReadiness: project.ShimReadinessFromPreview(preview),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/shim-readiness", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body shimReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode shim readiness response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" {
		t.Fatalf("unexpected shim readiness response: %+v", body)
	}
	if len(body.Items) == 0 || body.Preview.Mode != "read_only_planning" {
		t.Fatalf("unexpected shim readiness detail: %+v", body)
	}
}

func TestProjectShimAuthorizationEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	contract := project.CompatibilityContract{
		Project: record,
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:  "./task-loop run",
				Mode:     "blocked",
				Status:   "pass",
				Metadata: map[string]any{"read_only": false},
			},
		},
	}
	readiness := project.ShimReadinessFromPreview(project.ShimPreviewFromCompatibility(contract))
	handler := NewHandler(fakeProjectStore{
		record:            record,
		shimAuthorization: project.ShimAuthorizationPacketFromReadiness(readiness),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/shim-authorization", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body shimAuthorizationPacketResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode shim authorization response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_authorization_packet" {
		t.Fatalf("unexpected shim authorization response: %+v", body)
	}
	if len(body.AllowedFiles) == 0 || body.AllowedFiles[0].Path != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected allowed files: %+v", body.AllowedFiles)
	}
	if !containsString(body.RequiredPreflight, "areaflow project status-projections areamatrix --json") {
		t.Fatalf("expected status projections preflight: %+v", body.RequiredPreflight)
	}
	if !containsString(body.RequiredPreflight, "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json") {
		t.Fatalf("expected executable status projection schema preflight: %+v", body.RequiredPreflight)
	}
	if !containsString(body.RequiredPreflight, "verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash") {
		t.Fatalf("expected stable projection schema preflight: %+v", body.RequiredPreflight)
	}
	if body.SafetyFacts["project_write_attempted"] || body.SafetyFacts["execution_write_attempted"] || body.SafetyFacts["task_loop_run_forwarded"] {
		t.Fatalf("unexpected shim authorization safety facts: %+v", body.SafetyFacts)
	}
}

func TestProjectShimApplyPacketEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var capturedOptions project.ShimApplyPacketPreviewOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		shimApplyPacket: project.ShimApplyPacketPreview{
			Project:  record,
			Status:   "blocked",
			Mode:     "shim_apply_packet_preview_v1",
			Decision: "readiness_blocked",
			Message:  "shim apply packet is blocked until shim readiness proof, status projection schema and AreaMatrix review evidence pass",
			Authorization: project.ShimAuthorizationPacket{
				Project: record,
				Status:  "blocked",
				Mode:    "read_only_authorization_packet",
			},
			Gate: project.ShimApplyGate{
				Project: record,
				Status:  "blocked",
				Mode:    "shim_apply_gate_v1",
				Items: []project.ShimApplyGateItem{
					{Key: "readiness_blockers", Category: "readiness", Status: "blocked", BlockedBy: []string{"shim_readiness_still_blocked"}},
				},
				SafetyFacts: map[string]bool{"project_write_attempted": false, "task_loop_run_forwarded": false},
			},
			Packet: project.ShimApplyPacket{
				CommandType:                "project.shim.apply",
				ProjectKey:                 "areamatrix",
				AuthorizationSnapshotHash:  "authorization-hash",
				StatusProjectionGateID:     "status-gate-1",
				ProtectedPathFingerprintID: "fingerprint-1",
				ExplicitApproval:           true,
				ApprovalID:                 "approval-1",
			},
			ApplyGateCommand: []string{"areaflow", "project", "shim-apply-gate", "areamatrix"},
			SafetyFacts:      map[string]bool{"project_write_attempted": false, "area_matrix_files_modified": false},
		},
		shimApplyPacketHook: func(options project.ShimApplyPacketPreviewOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/shim-apply-packet?explicit_approval=true&approval_id=approval-1&approval_actor=as&approval_reason=approve&status_projection_packet_id=status-packet-1&status_projection_gate_id=status-gate-1&read_only_smoke_evidence_id=smoke-1&dirty_worktree_review_id=dirty-review-1&protected_path_fingerprint_id=fingerprint-1&rollback_plan_id=rollback-1&idempotency_key=shim-key&audit_correlation_id=audit-shim-key", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("shim apply packet status = %d body=%s", resp.Code, resp.Body.String())
	}
	if !capturedOptions.ExplicitApproval || capturedOptions.ApprovalID != "approval-1" || capturedOptions.StatusProjectionGateID != "status-gate-1" || capturedOptions.ProtectedPathFingerprintID != "fingerprint-1" {
		t.Fatalf("unexpected shim apply packet options: %+v", capturedOptions)
	}
	var body shimApplyPacketPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode shim apply packet: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Packet.CommandType != "project.shim.apply" || body.CommandRequestCreated || body.ProjectWriteAttempted || body.TaskLoopRunForwarded {
		t.Fatalf("unexpected shim apply packet response: %+v", body)
	}
	if body.SafetyFacts["area_matrix_files_modified"] {
		t.Fatalf("shim apply packet must not modify AreaMatrix: %+v", body.SafetyFacts)
	}
}

func TestProjectShimApplyGateEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var capturedOptions project.ShimApplyGateOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		shimApplyGate: project.ShimApplyGate{
			Project:              record,
			Status:               "blocked",
			Mode:                 "shim_apply_gate_v1",
			Decision:             "no_go",
			Message:              "shim apply packet is blocked",
			AllowedFiles:         []string{"scripts/areaflow_shim.py", ".areaflow/status.json"},
			RequiredPacketFields: []string{"allowed_files", "authorization_snapshot_hash", "explicit_approval"},
			RequiredProofFacts:   []string{"status_projection_apply_gate", "protected_path_fingerprint"},
			Items: []project.ShimApplyGateItem{
				{Key: "readiness_blockers", Category: "readiness", Status: "blocked", BlockedBy: []string{"shim_readiness_still_blocked"}},
			},
			SafetyFacts: map[string]bool{"project_write_attempted": false, "area_matrix_files_modified": false},
		},
		shimApplyGateHook: func(options project.ShimApplyGateOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/shim-apply-gate?allowed_files=scripts/areaflow_shim.py,.areaflow/status.json&authorization_snapshot_hash=authorization-hash&expected_authorization_mode=read_only_authorization_packet&approval_id=approval-1&approval_scope=areamatrix_compatibility_shim_files_only_no_execution_cutover&explicit_approval=true&approval_actor=as&approval_reason=approve&status_projection_packet_id=status-packet-1&status_projection_gate_id=status-gate-1&read_only_smoke_evidence_id=smoke-1&dirty_worktree_review_id=dirty-review-1&protected_path_fingerprint_id=fingerprint-1&rollback_plan_id=rollback-1&failure_mode=fail_closed&idempotency_key=shim-key&audit_correlation_id=audit-shim-key", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("shim apply gate status = %d body=%s", resp.Code, resp.Body.String())
	}
	if !capturedOptions.ExplicitApproval || capturedOptions.AuthorizationSnapshotHash != "authorization-hash" || capturedOptions.StatusProjectionPacketID != "status-packet-1" || capturedOptions.FailureMode != "fail_closed" {
		t.Fatalf("unexpected shim apply gate options: %+v", capturedOptions)
	}
	if len(capturedOptions.AllowedFiles) != 2 || capturedOptions.AllowedFiles[0] != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected allowed files options: %+v", capturedOptions.AllowedFiles)
	}
	var body shimApplyGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode shim apply gate: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.ApplyCommandEligible || body.CommandRequestCreated || body.ProjectWriteAttempted || body.TaskLoopRunForwarded {
		t.Fatalf("unexpected shim apply gate response: %+v", body)
	}
	if body.SafetyFacts["area_matrix_files_modified"] {
		t.Fatalf("shim apply gate must not modify AreaMatrix: %+v", body.SafetyFacts)
	}
}

func TestProjectShimApplyEndpointRecordsCommand(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var capturedOptions project.ApplyShimCommandOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		shimApplyCommand: project.ApplyShimCommandResult{
			Project:  record,
			Status:   "recorded",
			Mode:     "shim_apply_command_v1",
			Decision: "allowed",
			Message:  "shim apply command recorded",
			Gate: project.ShimApplyGate{
				Project:              record,
				Status:               "pass",
				Decision:             "go",
				ApplyCommandEligible: true,
				SafetyFacts:          map[string]bool{"project_write_attempted": false},
			},
			EventID:                11,
			AuditEventID:           12,
			IdempotencyKey:         "shim-key",
			Created:                true,
			ApplyOpen:              true,
			AreaFlowCommandCreated: true,
			CommandRequestCreated:  true,
			SafetyFacts: map[string]bool{
				"command_request_created":    true,
				"area_flow_command_created":  true,
				"project_write_attempted":    false,
				"execution_write_attempted":  false,
				"task_loop_run_forwarded":    false,
				"area_matrix_files_modified": false,
			},
		},
		shimApplyCommandHook: func(options project.ApplyShimCommandOptions) {
			capturedOptions = options
		},
	})
	body := `{"allowed_files":["scripts/areaflow_shim.py",".areaflow/status.json"],"authorization_snapshot_hash":"authorization-hash","expected_authorization_mode":"read_only_authorization_packet","approval_id":"approval-1","approval_scope":"areamatrix_compatibility_shim_files_only_no_execution_cutover","explicit_approval":true,"approval_actor":"as","approval_reason":"approve","status_projection_packet_id":"status-packet-1","status_projection_gate_id":"status-gate-1","read_only_smoke_evidence_id":"smoke-1","dirty_worktree_review_id":"dirty-review-1","protected_path_fingerprint_id":"fingerprint-1","rollback_plan_id":"rollback-1","failure_mode":"fail_closed","idempotency_key":"shim-key","audit_correlation_id":"audit-shim-key"}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/shim-apply", strings.NewReader(body))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("shim apply status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.IdempotencyKey != "shim-key" || capturedOptions.Actor != "as" || capturedOptions.Reason != "approve" {
		t.Fatalf("unexpected shim apply top-level options: %+v", capturedOptions)
	}
	if !capturedOptions.Gate.ExplicitApproval || capturedOptions.Gate.AuthorizationSnapshotHash != "authorization-hash" || capturedOptions.Gate.StatusProjectionGateID != "status-gate-1" || capturedOptions.Gate.FailureMode != "fail_closed" {
		t.Fatalf("unexpected shim apply options: %+v", capturedOptions)
	}
	if len(capturedOptions.Gate.AllowedFiles) != 2 || capturedOptions.Gate.AllowedFiles[0] != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected shim apply allowed files: %+v", capturedOptions.Gate.AllowedFiles)
	}
	var response shimApplyCommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode shim apply: %v", err)
	}
	if response.Project.Key != "areamatrix" || response.Status != "recorded" || response.Decision != "allowed" {
		t.Fatalf("unexpected shim apply response: %+v", response)
	}
	if !response.ApplyOpen || !response.CommandRequestCreated || !response.AreaFlowCommandCreated || response.ProjectWriteAttempted || response.ExecutionWriteAttempted || response.TaskLoopRunForwarded || response.AreaMatrixFilesModified {
		t.Fatalf("shim apply endpoint must record only AreaFlow command state: %+v", response)
	}
	if response.EventID != 11 || response.AuditEventID != 12 || response.IdempotencyKey != "shim-key" || !response.Created {
		t.Fatalf("missing shim apply command evidence: %+v", response)
	}
	if len(response.Blockers) != 0 || response.Gate.Status != "pass" {
		t.Fatalf("unexpected blocker/gate response: %+v", response)
	}
}

func TestProjectShimReadinessEvidenceEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var capturedOptions project.RecordShimReadinessEvidenceOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		shimEvidence: project.RecordShimReadinessEvidenceResult{
			Project:                 record,
			EvidenceKey:             "real_areamatrix_readonly_smoke",
			Status:                  "recorded",
			Decision:                "allowed",
			Message:                 "shim readiness evidence recorded",
			EventID:                 11,
			AuditEventID:            12,
			IdempotencyKey:          "shim-evidence-key",
			Created:                 true,
			ProjectWriteAttempted:   false,
			ExecutionWriteAttempted: false,
			EngineCallAttempted:     false,
			Metadata:                map[string]any{"summary": "readonly smoke passed"},
		},
		shimEvidenceHook: func(options project.RecordShimReadinessEvidenceOptions) {
			capturedOptions = options
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/shim-readiness/evidence", strings.NewReader(`{
		"evidence_key": "real_areamatrix_readonly_smoke",
		"summary": "readonly smoke passed",
		"evidence_uri": "scripts/smoke-areamatrix-readonly.sh",
		"idempotency_key": "shim-evidence-key",
		"actor": "tester",
		"reason": "record readonly smoke"
	}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if capturedOptions.EvidenceKey != "real_areamatrix_readonly_smoke" || capturedOptions.Actor != "tester" {
		t.Fatalf("options not forwarded: %+v", capturedOptions)
	}
	var body shimReadinessEvidenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode shim readiness evidence response: %v", err)
	}
	if body.EvidenceKey != "real_areamatrix_readonly_smoke" || body.ProjectWriteAttempted || body.ExecutionWriteAttempted || body.EngineCallAttempted {
		t.Fatalf("unexpected shim readiness evidence response: %+v", body)
	}
}

func TestProjectExecutionCutoverReadinessEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionCutover: project.AreaMatrixExecutionCutoverReadiness{
			Project: record,
			Status:  "blocked",
			Mode:    "read_only_areamatrix_execution_cutover_readiness",
			Items: []project.AreaMatrixExecutionCutoverReadinessItem{
				{
					Key:              "explicit_execution_cutover_approval",
					Category:         "approval",
					Status:           "blocked",
					Message:          "explicit execution cutover approval is required",
					RequiredEvidence: []string{"R3/R4 approval"},
					NextCommand:      "areaflow project execution-cutover-readiness areamatrix --json",
					Metadata:         map[string]any{"approval_required": true},
				},
			},
			MigrationPath:    []string{"Import", "Mirror", "Shadow", "Authoring Cutover", "Execution Beta", "Execution Cutover"},
			CommandEvidence:  map[string]int{"runner.preview": 1},
			Capabilities:     []string{"read_project", "write_artifacts", "run_commands"},
			ForbiddenActions: []string{"forward_task_loop_run", "apply_execution_cutover"},
			SafetyFacts: map[string]bool{
				"read_only":                    true,
				"project_write_attempted":      false,
				"execution_cutover_apply_open": false,
			},
			NextSteps: []project.AreaMatrixExecutionCutoverNextStep{
				{
					Key:         "land_areamatrix_shim",
					Owner:       "project_owner",
					Action:      "land AreaMatrix compatibility shim after explicit edit approval",
					RiskLevel:   "R2 managed_write",
					BlockedBy:   []string{"explicit_edit_approval"},
					NextCommand: "areaflow project shim-readiness areamatrix --json",
				},
			},
			GeneratedAt: time.Date(2026, 7, 2, 19, 40, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-cutover-readiness", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionCutoverReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution cutover readiness response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_areamatrix_execution_cutover_readiness" {
		t.Fatalf("unexpected execution cutover readiness response: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "explicit_execution_cutover_approval" {
		t.Fatalf("unexpected execution cutover items: %+v", body.Items)
	}
	if body.SafetyFacts["execution_cutover_apply_open"] || !body.SafetyFacts["read_only"] {
		t.Fatalf("unexpected safety facts: %+v", body.SafetyFacts)
	}
	if len(body.NextSteps) != 1 || body.NextSteps[0].RiskLevel != "R2 managed_write" {
		t.Fatalf("unexpected next steps: %+v", body.NextSteps)
	}
}

func TestProjectExecutionForwardingV1ReadinessEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwarding: project.ExecutionForwardingV1Readiness{
			Project: record,
			Status:  "blocked",
			Mode:    "read_only_execution_forwarding_v1_readiness",
			Items: []project.ExecutionForwardingV1ReadinessItem{
				{
					Key:              "forwarding_command_api",
					Category:         "command_api",
					Status:           "pass",
					Message:          "Execution Forwarding v1 apply command exists and stays protected by packet, gate, idempotency and audit",
					RequiredEvidence: []string{"Command API design", "idempotency key", "approval id", "audit response"},
					NextCommand:      "areaflow project execution-forwarding-v1-apply areamatrix --json",
					Metadata:         map[string]any{"apply_open": false},
				},
			},
			AllowedTaskTypes: []string{"read_only_verify", "artifact_evidence"},
			CommandEvidence:  map[string]int{"run.read_only_verify_queue": 1},
			Capabilities:     []string{"read_project", "write_artifacts"},
			ForbiddenActions: []string{"engine_execution", "restore_apply"},
			SafetyFacts: map[string]bool{
				"read_only":                true,
				"forwarding_v1_apply_open": false,
				"task_loop_run_forwarded":  false,
				"project_write_attempted":  false,
			},
			NextSteps: []project.ExecutionForwardingV1NextStep{
				{
					Key:         "define_forwarding_v1_command",
					Owner:       "execution_owner",
					Action:      "land AreaMatrix read-only shim and real forwarding smoke before enabling forwarding",
					RiskLevel:   "R3 execution",
					BlockedBy:   []string{"read_only_shim_missing", "real_forwarding_smoke_missing"},
					NextCommand: "areaflow project execution-forwarding-v1-apply areamatrix --json",
				},
			},
			GeneratedAt: time.Date(2026, 7, 3, 12, 30, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-readiness", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionForwardingV1ReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution forwarding v1 readiness response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_execution_forwarding_v1_readiness" {
		t.Fatalf("unexpected forwarding v1 readiness response: %+v", body)
	}
	if len(body.AllowedTaskTypes) != 2 || body.AllowedTaskTypes[0] != "read_only_verify" {
		t.Fatalf("unexpected allowed task types: %+v", body.AllowedTaskTypes)
	}
	if body.SafetyFacts["forwarding_v1_apply_open"] || body.SafetyFacts["task_loop_run_forwarded"] || !body.SafetyFacts["read_only"] {
		t.Fatalf("unexpected safety facts: %+v", body.SafetyFacts)
	}
	if len(body.NextSteps) != 1 || body.NextSteps[0].RiskLevel != "R3 execution" {
		t.Fatalf("unexpected next steps: %+v", body.NextSteps)
	}
}

func TestProjectExecutionForwardingV1ApplyPreviewEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	readiness := project.ExecutionForwardingV1Readiness{
		Project:          record,
		Status:           "blocked",
		Mode:             "read_only_execution_forwarding_v1_readiness",
		AllowedTaskTypes: []string{"read_only_verify", "artifact_evidence"},
		SafetyFacts: map[string]bool{
			"read_only":                true,
			"forwarding_v1_apply_open": false,
		},
	}
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwardingApply: project.ExecutionForwardingV1ApplyPreview{
			Project:          record,
			Status:           "blocked",
			Mode:             "read_only_execution_forwarding_v1_apply_preview",
			Readiness:        readiness,
			AllowedTaskTypes: []string{"read_only_verify", "artifact_evidence"},
			ForwardingTargets: []project.ExecutionForwardingV1ForwardingTarget{
				{
					TaskType:              "read_only_verify",
					TargetCommandType:     "run.read_only_verify_queue",
					TargetStatus:          "available_scoped",
					RequiredCapabilities:  []string{"read_project", "manage_workers"},
					RequiredPacketFields:  []string{"project_key", "forwarded_task_type"},
					CreatesCommandRequest: true,
					CreatesRun:            true,
					CreatesRunTask:        true,
					CreatesAuditEvent:     true,
					ProjectWriteAllowed:   false,
					ExecutionWriteAllowed: false,
					LegacyFallbackAllowed: false,
					FailureMode:           "fail_closed",
				},
			},
			BlockedTargets: []project.ExecutionForwardingV1BlockedTarget{
				{
					TaskType:        "engine_execution",
					ForbiddenAction: "engine_execution",
					Reason:          "engine execution stays closed",
					FailureMode:     "fail_closed",
					SafetyFacts: map[string]bool{
						"engine_call_attempted": false,
						"commands_run":          false,
					},
				},
			},
			RequiredCapabilities: []string{"read_project", "write_artifacts", "manage_workers"},
			ApplyPacketFields:    []string{"command_type", "approval_id", "readiness_snapshot_hash"},
			FailClosedFields:     []string{"status", "failure_mode", "audit_event_id"},
			RequiredProofFacts:   []string{"legacy_task_loop_runner_not_started", "rollback_to_read_only_shim_verified"},
			RequiredEvidence:     []string{"explicit R3 execution forwarding v1 approval"},
			ForbiddenActions:     []string{"engine_execution", "restore_apply"},
			ApprovalRequired:     true,
			ApprovalStatus:       "needs_approval",
			ApplyOpen:            false,
			RollbackTarget:       "read_only_shim",
			SafetyFacts: map[string]bool{
				"read_only_preview":         true,
				"apply_open":                false,
				"task_loop_run_forwarded":   false,
				"project_write_attempted":   false,
				"execution_write_attempted": false,
			},
			Items: []project.ExecutionForwardingV1ApplyPreviewItem{
				{
					Key:              "forwarding_v1:explicit_approval",
					Category:         "approval",
					Status:           "blocked",
					ApprovalStatus:   "needs_approval",
					Message:          "explicit approval is required",
					Owner:            "project_owner",
					RequiredEvidence: []string{"approval"},
					NextCommand:      "areaflow project execution-forwarding-v1-apply-preview areamatrix --json",
				},
			},
			GeneratedAt: time.Date(2026, 7, 3, 15, 45, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-apply-preview", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionForwardingV1ApplyPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution forwarding v1 apply preview response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_execution_forwarding_v1_apply_preview" {
		t.Fatalf("unexpected forwarding v1 apply preview response: %+v", body)
	}
	if !body.ApprovalRequired || body.ApprovalStatus != "needs_approval" || body.ApplyOpen {
		t.Fatalf("unexpected approval/apply fields: %+v", body)
	}
	if body.RollbackTarget != "read_only_shim" {
		t.Fatalf("rollback target = %q", body.RollbackTarget)
	}
	if len(body.ForwardingTargets) != 1 ||
		body.ForwardingTargets[0].TargetCommandType != "run.read_only_verify_queue" ||
		body.ForwardingTargets[0].ProjectWriteAllowed ||
		body.ForwardingTargets[0].LegacyFallbackAllowed {
		t.Fatalf("unexpected forwarding targets: %+v", body.ForwardingTargets)
	}
	if len(body.BlockedTargets) != 1 ||
		body.BlockedTargets[0].TaskType != "engine_execution" ||
		body.BlockedTargets[0].FailureMode != "fail_closed" {
		t.Fatalf("unexpected blocked targets: %+v", body.BlockedTargets)
	}
	if !containsString(body.FailClosedFields, "audit_event_id") {
		t.Fatalf("missing fail closed fields: %+v", body.FailClosedFields)
	}
	if body.SafetyFacts["apply_open"] || body.SafetyFacts["task_loop_run_forwarded"] || !body.SafetyFacts["read_only_preview"] {
		t.Fatalf("unexpected safety facts: %+v", body.SafetyFacts)
	}
	if len(body.Items) != 1 || body.Items[0].ApprovalStatus != "needs_approval" {
		t.Fatalf("unexpected items: %+v", body.Items)
	}
}

func TestProjectExecutionForwardingV1ApplyPacketEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var captured project.ExecutionForwardingV1ApplyPacketPreviewOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwardingPacketHook: func(options project.ExecutionForwardingV1ApplyPacketPreviewOptions) {
			captured = options
		},
		executionForwardingPacket: project.ExecutionForwardingV1ApplyPacketPreview{
			Project:  record,
			Status:   "blocked",
			Mode:     "execution_forwarding_v1_apply_packet_preview_v1",
			Decision: "readiness_blocked",
			Message:  "read-only shim is not ready",
			Packet: project.ExecutionForwardingV1ApplyPacket{
				CommandType:                "project.execution_forwarding_v1.apply",
				ProjectKey:                 "areamatrix",
				AllowedTaskTypes:           []string{"read_only_verify"},
				TargetCommandTypes:         []string{"run.read_only_verify_queue"},
				ApprovalID:                 "approval-1",
				ApprovalScope:              "execution_forwarding_v1_read_only_evidence_only",
				ReadinessSnapshotHash:      "snapshot-hash",
				ExpectedShimLifecycleState: "read_only_shim",
				LegacyNonWriteProofID:      "proof-1",
				RollbackPlanID:             "rollback-1",
				ProtectedPathFingerprintID: "fingerprint-1",
				IdempotencyKey:             "forwarding-key",
				AuditCorrelationID:         "audit-forwarding-key",
				FailureMode:                "fail_closed",
				ExplicitApproval:           true,
				ApprovalActor:              "as",
				ApprovalReason:             "approve forwarding v1",
			},
			Gate: project.ExecutionForwardingV1ApplyGate{
				Project:              record,
				Status:               "blocked",
				Mode:                 "execution_forwarding_v1_apply_gate_v1",
				Decision:             "no_go",
				ApprovalRequired:     true,
				ApprovalStatus:       "missing_or_incomplete",
				ApplyCommandEligible: false,
				SafetyFacts:          map[string]bool{"read_only_gate": true, "command_request_created": false},
				Items: []project.ExecutionForwardingV1ApplyGateItem{
					{Key: "read_only_shim", Category: "readiness", Status: "blocked", BlockedBy: []string{"read_only_shim_not_pass"}},
				},
			},
			ApplyGateCommand:   []string{"areaflow", "project", "execution-forwarding-v1-apply-gate", "areamatrix"},
			FutureApplyCommand: []string{"areaflow", "project", "execution-forwarding-v1-apply", "areamatrix"},
			SafetyFacts:        map[string]bool{"read_only_preview": true, "command_request_created": false},
			GeneratedAt:        time.Date(2026, 7, 4, 4, 55, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-apply-packet?explicit_approval=true&approval_id=approval-1&approval_actor=as&approval_reason=approve&legacy_non_write_proof_id=proof-1&rollback_plan_id=rollback-1&protected_path_fingerprint_id=fingerprint-1&idempotency_key=forwarding-key&audit_correlation_id=audit-forwarding-key", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if !captured.ExplicitApproval || captured.ApprovalID != "approval-1" || captured.LegacyNonWriteProofID != "proof-1" {
		t.Fatalf("query options not captured: %+v", captured)
	}
	var body executionForwardingV1ApplyPacketPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution forwarding v1 apply packet response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Packet.ReadinessSnapshotHash != "snapshot-hash" || body.Gate.Decision != "no_go" {
		t.Fatalf("unexpected apply packet response: %+v", body)
	}
	if body.CommandRequestCreated || body.TaskLoopRunForwarded || body.ProjectWriteAttempted {
		t.Fatalf("packet endpoint must be read-only: %+v", body)
	}
}

func TestProjectExecutionForwardingV1ApplyGateEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var captured project.ExecutionForwardingV1ApplyGateOptions
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwardingGateHook: func(options project.ExecutionForwardingV1ApplyGateOptions) {
			captured = options
		},
		executionForwardingGate: project.ExecutionForwardingV1ApplyGate{
			Project:              record,
			Status:               "blocked",
			Mode:                 "execution_forwarding_v1_apply_gate_v1",
			Decision:             "no_go",
			Message:              "execution forwarding v1 apply packet is blocked",
			AllowedTaskTypes:     []string{"read_only_verify"},
			TargetCommandTypes:   []string{"run.read_only_verify_queue"},
			ApprovalRequired:     true,
			ApprovalStatus:       "missing_or_incomplete",
			ApplyCommandEligible: false,
			ApplyOpen:            false,
			SafetyFacts:          map[string]bool{"read_only_gate": true, "command_request_created": false, "task_loop_run_forwarded": false},
			Items: []project.ExecutionForwardingV1ApplyGateItem{
				{Key: "readiness_snapshot_hash", Category: "packet", Status: "blocked", Expected: "current", Actual: "stale", BlockedBy: []string{"readiness_snapshot_hash_missing_or_mismatch"}},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-apply-gate?allowed_task_types=read_only_verify&approval_id=approval-1&approval_scope=execution_forwarding_v1_read_only_evidence_only&readiness_snapshot_hash=stale&expected_shim_lifecycle_state=read_only_shim&legacy_non_write_proof_id=proof-1&rollback_plan_id=rollback-1&protected_path_fingerprint_id=fingerprint-1&failure_mode=fail_closed&idempotency_key=forwarding-key&audit_correlation_id=audit-forwarding-key&explicit_approval=true&approval_actor=as&approval_reason=approve", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if len(captured.AllowedTaskTypes) != 1 || captured.AllowedTaskTypes[0] != "read_only_verify" || !captured.ExplicitApproval || captured.ReadinessSnapshotHash != "stale" {
		t.Fatalf("query options not captured: %+v", captured)
	}
	var body executionForwardingV1ApplyGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution forwarding v1 apply gate response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Decision != "no_go" || body.ApplyCommandEligible {
		t.Fatalf("unexpected apply gate response: %+v", body)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "readiness_snapshot_hash" {
		t.Fatalf("unexpected gate items: %+v", body.Items)
	}
	if body.SafetyFacts["command_request_created"] || body.SafetyFacts["task_loop_run_forwarded"] {
		t.Fatalf("gate endpoint must be read-only: %+v", body.SafetyFacts)
	}
}

func TestProjectExecutionForwardingV1ApplyEndpointBlocked(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	var captured project.ApplyExecutionForwardingV1Options
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwardingApplyHook: func(options project.ApplyExecutionForwardingV1Options) {
			captured = options
		},
		executionForwardingApplyCmd: project.ApplyExecutionForwardingV1Result{
			Project:  record,
			Status:   "blocked",
			Decision: "denied",
			Message:  "execution forwarding v1 apply blocked by gate requirements",
			Blockers: []string{"execution_forwarding_v1_apply_gate_blocked", "read_only_shim_not_pass"},
			Gate: project.ExecutionForwardingV1ApplyGate{
				Project:              record,
				Status:               "blocked",
				Mode:                 "execution_forwarding_v1_apply_gate_v1",
				Decision:             "no_go",
				ApprovalStatus:       "missing_or_incomplete",
				ApplyCommandEligible: false,
				AllowedTaskTypes:     []string{"read_only_verify"},
				TargetCommandTypes:   []string{"run.read_only_verify_queue"},
				SafetyFacts:          map[string]bool{"read_only_gate": true},
			},
			EventID:                   11,
			AuditEventID:              12,
			IdempotencyKey:            "forwarding-key",
			Created:                   true,
			CommandRequestCreated:     true,
			AreaFlowCommandCreated:    true,
			AreaFlowAuditEventCreated: true,
			SafetyFacts: map[string]bool{
				"apply_command_executed":    true,
				"command_request_created":   true,
				"area_flow_command_created": true,
				"area_flow_run_created":     false,
				"task_loop_run_forwarded":   false,
				"project_write_attempted":   false,
				"engine_call_attempted":     false,
			},
		},
	})
	body := `{"allowed_task_types":["read_only_verify"],"approval_id":"approval-1","approval_scope":"execution_forwarding_v1_read_only_evidence_only","readiness_snapshot_hash":"hash","expected_shim_lifecycle_state":"read_only_shim","legacy_non_write_proof_id":"proof-1","rollback_plan_id":"rollback-1","protected_path_fingerprint_id":"fingerprint-1","idempotency_key":"forwarding-key","audit_correlation_id":"audit-forwarding-key","failure_mode":"fail_closed","explicit_approval":true,"approval_actor":"as","approval_reason":"approve forwarding v1","actor":"as","reason":"apply forwarding v1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/areamatrix/execution-forwarding-v1-apply", strings.NewReader(body))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if captured.IdempotencyKey != "forwarding-key" ||
		captured.Actor != "as" ||
		captured.Gate.ReadinessSnapshotHash != "hash" ||
		len(captured.Gate.AllowedTaskTypes) != 1 ||
		captured.Gate.AllowedTaskTypes[0] != "read_only_verify" {
		t.Fatalf("request options not captured: %+v", captured)
	}
	var decoded executionForwardingV1ApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode execution forwarding v1 apply response: %v", err)
	}
	if decoded.Project.Key != "areamatrix" || decoded.Status != "blocked" || decoded.Decision != "denied" {
		t.Fatalf("unexpected apply response: %+v", decoded)
	}
	if !decoded.CommandRequestCreated || !decoded.AreaFlowCommandCreated || !decoded.AreaFlowAuditEventCreated {
		t.Fatalf("expected AreaFlow command/audit evidence: %+v", decoded)
	}
	if decoded.AreaFlowRunCreated || decoded.TaskLoopRunForwarded || decoded.ProjectWriteAttempted || decoded.EngineCallAttempted {
		t.Fatalf("unexpected runtime side effects: %+v", decoded)
	}
	if !decoded.SafetyFacts["apply_command_executed"] || decoded.SafetyFacts["task_loop_run_forwarded"] {
		t.Fatalf("unexpected safety facts: %+v", decoded.SafetyFacts)
	}
}

func TestProjectExecutionForwardingV1CommandPreviewEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwardingCmd: project.ExecutionForwardingV1CommandPreview{
			Project:                                record,
			Status:                                 "blocked",
			Mode:                                   "read_only_execution_forwarding_v1_command_preview",
			Decision:                               "would_forward_after_approval",
			Message:                                "task type is in the forwarding v1 target matrix, but apply remains closed",
			TaskType:                               "read_only_verify",
			TargetCommandType:                      "run.read_only_verify_queue",
			TargetStatus:                           "available_scoped",
			FailureMode:                            "fail_closed",
			AllowedTaskType:                        true,
			ApplyOpen:                              false,
			WouldCreateCommandRequestAfterApproval: true,
			WouldCreateRunAfterApproval:            true,
			WouldCreateRunTaskAfterApproval:        true,
			WouldCreateAuditEventAfterApproval:     true,
			ProjectWriteAllowed:                    false,
			ExecutionWriteAllowed:                  false,
			LegacyFallbackAllowed:                  false,
			RequiredPacketFields:                   []string{"project_key", "forwarded_task_type"},
			RequiredCapabilities:                   []string{"read_project"},
			FailClosedFields:                       []string{"legacy_task_loop_started", "audit_event_id"},
			BlockedBy:                              []string{"execution_forwarding_v1_apply_open=false"},
			AllowedTaskTypes:                       []string{"read_only_verify"},
			ForbiddenActions:                       []string{"engine_execution"},
			SafetyFacts: map[string]bool{
				"read_only_preview":         true,
				"command_preview":           true,
				"area_flow_command_created": false,
				"task_loop_run_forwarded":   false,
				"project_write_attempted":   false,
			},
			GeneratedAt: time.Date(2026, 7, 4, 1, 45, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-command-preview?task_type=read_only_verify", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionForwardingV1CommandPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution forwarding v1 command preview response: %v", err)
	}
	if body.Project.Key != "areamatrix" ||
		body.Decision != "would_forward_after_approval" ||
		body.TargetCommandType != "run.read_only_verify_queue" ||
		!body.AllowedTaskType ||
		body.ApplyOpen ||
		body.ProjectWriteAllowed ||
		body.LegacyFallbackAllowed {
		t.Fatalf("unexpected command preview response: %+v", body)
	}
	if body.SafetyFacts["area_flow_command_created"] || body.SafetyFacts["task_loop_run_forwarded"] || !body.SafetyFacts["command_preview"] {
		t.Fatalf("unexpected safety facts: %+v", body.SafetyFacts)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-command-preview", nil)
	missingResp := httptest.NewRecorder()
	handler.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusBadRequest {
		t.Fatalf("missing task_type status = %d body=%s", missingResp.Code, missingResp.Body.String())
	}
}

func TestProjectExecutionForwardingV1RollbackPreviewEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix"}
	applyPreview := project.ExecutionForwardingV1ApplyPreview{
		Project:        record,
		Status:         "blocked",
		Mode:           "read_only_execution_forwarding_v1_apply_preview",
		ApplyOpen:      false,
		RollbackTarget: "read_only_shim",
		SafetyFacts: map[string]bool{
			"read_only_preview": true,
			"apply_open":        false,
		},
		GeneratedAt: time.Date(2026, 7, 3, 15, 45, 0, 0, time.UTC),
	}
	handler := NewHandler(fakeProjectStore{
		record: record,
		executionForwardingRoll: project.ExecutionForwardingV1RollbackPreview{
			Project:            record,
			Status:             "blocked",
			Mode:               "read_only_execution_forwarding_v1_rollback_preview",
			ApplyPreview:       applyPreview,
			RollbackTarget:     "read_only_shim",
			FailClosedSteps:    []string{"keep ./task-loop run blocked"},
			ReopenConditions:   []string{"explicit R3 approval", "protected path proof clean"},
			RequiredProofFacts: []string{"task_loop_run_forwarding_disabled", "protected_path_proof_clean_after_rollback_recorded"},
			RequiredEvidence:   []string{"protected path proof after rollback"},
			ForbiddenActions:   []string{"create_rollback_command", "delete_forwarding_history", "restore_apply"},
			RollbackApplyOpen:  false,
			SafetyFacts: map[string]bool{
				"read_only_preview":         true,
				"rollback_apply_open":       false,
				"apply_open":                false,
				"task_loop_run_forwarded":   false,
				"project_write_attempted":   false,
				"execution_write_attempted": false,
			},
			Items: []project.ExecutionForwardingV1RollbackPreviewItem{
				{
					Key:              "rollback_v1:fail_closed",
					Category:         "rollback",
					Status:           "blocked",
					Message:          "rollback proof is required",
					Owner:            "execution_owner",
					RequiredEvidence: []string{"legacy non-write proof"},
					NextCommand:      "areaflow completion protected-path-proof record areamatrix --status clean --summary <text> --evidence-uri <uri> --json",
				},
			},
			GeneratedAt: time.Date(2026, 7, 3, 16, 0, 0, 0, time.UTC),
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/execution-forwarding-v1-rollback-preview", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body executionForwardingV1RollbackPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode execution forwarding v1 rollback preview response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_execution_forwarding_v1_rollback_preview" {
		t.Fatalf("unexpected rollback preview response: %+v", body)
	}
	if body.RollbackTarget != "read_only_shim" || body.RollbackApplyOpen {
		t.Fatalf("unexpected rollback fields: %+v", body)
	}
	if body.SafetyFacts["rollback_apply_open"] || body.SafetyFacts["task_loop_run_forwarded"] || !body.SafetyFacts["read_only_preview"] {
		t.Fatalf("unexpected safety facts: %+v", body.SafetyFacts)
	}
	if len(body.Items) != 1 || body.Items[0].Key != "rollback_v1:fail_closed" {
		t.Fatalf("unexpected items: %+v", body.Items)
	}
	if !containsString(body.ForbiddenActions, "delete_forwarding_history") {
		t.Fatalf("missing forbidden action: %+v", body.ForbiddenActions)
	}
}

func TestProjectSummaryEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 2, 52, 5, 0, time.UTC)
	handler := NewHandler(fakeProjectStore{
		record: project.Record{ID: 1, Key: "areamatrix"},
		summary: project.ProjectSummary{
			Project: project.Record{
				ID:              1,
				Key:             "areamatrix",
				Name:            "AreaMatrix",
				Kind:            "desktop-app",
				Adapter:         "areamatrix",
				WorkflowProfile: "areamatrix",
				DefaultBranch:   "main",
				RootPath:        "/tmp/AreaMatrix",
			},
			Inventory: project.ImportInventory{
				Versions:        2,
				Residuals:       10,
				Artifacts:       6,
				ImportSnapshots: 1,
				MirrorExports:   1,
			},
			Import: project.Snapshot{
				SourceHash: "hash-a",
				CreatedAt:  created,
				Summary:    map[string]any{"residual_count": float64(10)},
			},
			HasImport: true,
			LatestDoctor: project.EventRecord{
				Severity:  "info",
				CreatedAt: created,
				Metadata:  map[string]any{"overall_status": "pass"},
			},
			HasLatestDoctor:     true,
			DoctorStatus:        "pass",
			DriftStatus:         "pass",
			ConfigDriftStatus:   "pass",
			StageCoverageStatus: "pass",
			NativeDoctorStatus:  "warn",
			Config:              testProjectConfigRecord(created),
			HasConfig:           true,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/summary", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode summary response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Inventory.Residuals != 10 {
		t.Fatalf("unexpected summary: %+v", body)
	}
	if body.Import == nil || body.Import.SourceHash != "hash-a" {
		t.Fatalf("unexpected import summary: %+v", body.Import)
	}
	if body.Config == nil || body.Config.ConfigHash != "hash-config" {
		t.Fatalf("unexpected config summary: %+v", body.Config)
	}
	if body.Import.HistoryReadyForDiff {
		t.Fatalf("history should not be ready without previous import: %+v", body.Import)
	}
	if body.Doctor == nil || body.Doctor.Status != "pass" {
		t.Fatalf("unexpected doctor summary: %+v", body.Doctor)
	}
	if body.Doctor.DriftStatus != "pass" || body.Doctor.StageCoverageStatus != "pass" {
		t.Fatalf("unexpected stable doctor fields: %+v", body.Doctor)
	}
	if body.Doctor.ConfigDriftStatus != "pass" {
		t.Fatalf("unexpected config drift status: %+v", body.Doctor)
	}
	if body.Doctor.NativeDoctorStatus != "warn" {
		t.Fatalf("unexpected native doctor status: %+v", body.Doctor)
	}
}

func TestProjectReadinessEndpoint(t *testing.T) {
	summary := project.ProjectSummary{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Inventory: project.ImportInventory{
			Versions:        2,
			Residuals:       10,
			Artifacts:       6,
			ImportSnapshots: 1,
			MirrorExports:   1,
		},
		HasImport:           true,
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "warn",
		LatestEventCount:    2,
	}
	readiness := project.ProjectReadinessFromSummary(summary)
	handler := NewHandler(fakeProjectStore{
		record:    summary.Project,
		readiness: readiness,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/readiness", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "warn" {
		t.Fatalf("unexpected readiness response: %+v", body)
	}
	if len(body.Items) != 11 {
		t.Fatalf("readiness item count = %d, want 11", len(body.Items))
	}
	if body.Summary.Project.Key != "areamatrix" {
		t.Fatalf("unexpected readiness summary: %+v", body.Summary)
	}
}

func TestProjectGeneratedWriteReadinessEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"}
	readiness := project.GeneratedWriteReadiness{
		Project:                   record,
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		RequiredCapabilities:      []string{"read_project", "write_artifacts", "write_generated"},
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		RequiredWritePaths:        []string{".areaflow/generated/**", ".areamatrix/generated/**"},
		Blockers:                  []string{"real_areamatrix_apply_open: real AreaMatrix generated-only apply remains closed until explicit approval opens it"},
		ReviewBlockers:            []string{},
		ForbiddenActions:          []string{"queue_run", "write_project_file"},
		ReadyForReview:            true,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		ProjectConfigRead:         true,
		ProjectWriteAttempted:     false,
		EngineCallAttempted:       false,
		Items: []project.ReadinessItem{
			{Key: "required_capabilities", Status: "pass", Message: "capabilities ready"},
			{Key: "real_areamatrix_apply_open", Status: "blocked", Message: "apply closed"},
		},
		GeneratedAt: time.Date(2026, 7, 2, 10, 30, 0, 0, time.UTC),
	}
	handler := NewHandler(fakeProjectStore{
		record:                  record,
		generatedWriteReadiness: readiness,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/generated-write-readiness", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body generatedWriteReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode generated write readiness response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_generated_write_readiness" {
		t.Fatalf("unexpected generated write readiness response: %+v", body)
	}
	if !body.ReadyForReview || body.ApplyOpen || body.RealAreaMatrixWriteOpened || !body.GeneratedOnly {
		t.Fatalf("unexpected generated write readiness flags: %+v", body)
	}
	if body.ProjectWriteAttempted || body.EngineCallAttempted || len(body.Items) != 2 {
		t.Fatalf("generated write readiness should remain read-only: %+v", body)
	}
}

func TestProjectGeneratedWriteApplyBetaGateEndpoint(t *testing.T) {
	record := project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"}
	readiness := project.GeneratedWriteReadiness{
		Project:                   record,
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		ReadyForReview:            true,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
	}
	gate := project.GeneratedWriteApplyBetaGate{
		Project:                   record,
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_apply_beta_gate",
		Readiness:                 readiness,
		RequiredCapabilities:      []string{"read_project", "write_artifacts", "write_generated"},
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		RequiredEvidence:          []string{"explicit R3 approval for real AreaMatrix generated-only apply beta"},
		ForbiddenActions:          []string{"queue_run", "write_project_file"},
		ApprovalRequired:          true,
		ApprovalStatus:            "needs_approval",
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		Items: []project.GeneratedWriteApplyBetaGateItem{
			{Key: "generated_apply_beta:readiness", Category: "readiness", Status: "pass", Message: "ready"},
			{Key: "generated_apply_beta:explicit_approval", Category: "approval", Status: "blocked", ApprovalStatus: "needs_approval", Message: "approval required"},
		},
		GeneratedAt: time.Date(2026, 7, 2, 12, 30, 0, 0, time.UTC),
	}
	handler := NewHandler(fakeProjectStore{
		record:                 record,
		generatedApplyBetaGate: gate,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/areamatrix/generated-write-apply-beta-gate", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body generatedWriteApplyBetaGateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode generated write apply beta gate response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "blocked" || body.Mode != "read_only_generated_write_apply_beta_gate" {
		t.Fatalf("unexpected generated write apply beta gate response: %+v", body)
	}
	if !body.ApprovalRequired || body.ApprovalStatus != "needs_approval" || body.ApplyOpen || body.RealAreaMatrixWriteOpened {
		t.Fatalf("unexpected generated write apply beta gate flags: %+v", body)
	}
	if body.Readiness.Project.Key != "areamatrix" || !body.Readiness.ReadyForReview {
		t.Fatalf("nested readiness not encoded: %+v", body.Readiness)
	}
	if body.ProjectWriteAttempted || body.EngineCallAttempted || len(body.Items) != 2 {
		t.Fatalf("generated write apply beta gate should remain read-only: %+v", body)
	}
}

func testProjectConfigRecord(loadedAt time.Time) project.ProjectConfigRecord {
	return project.ProjectConfigRecord{
		ID:              1,
		ProjectID:       1,
		ProtocolVersion: 1,
		ConfigPath:      "examples/areamatrix/areaflow.yaml",
		ConfigHash:      "hash-config",
		Ownership:       map[string]any{"mode": "import"},
		StatusExport:    map[string]any{"path": ".areaflow/status.json"},
		Migration:       map[string]any{"phase": "import"},
		Active:          true,
		LoadedAt:        loadedAt,
	}
}

func TestProjectImportDiffEndpoint(t *testing.T) {
	summary := project.ProjectSummary{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Import: project.Snapshot{
			SourceHash: "hash-b",
			CreatedAt:  time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC),
			Summary:    map[string]any{"version_count": float64(2)},
		},
		HasImport: true,
		PreviousImport: project.Snapshot{
			SourceHash: "hash-a",
			CreatedAt:  time.Date(2026, 6, 29, 3, 0, 0, 0, time.UTC),
			Summary:    map[string]any{"version_count": float64(1)},
		},
		HasPreviousImport: true,
	}
	handler := NewHandler(fakeProjectStore{
		record: summary.Project,
		diff:   project.ProjectImportDiffFromSummary(summary),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/import-diff", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectImportDiffResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode import diff response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status != "changed" {
		t.Fatalf("unexpected import diff response: %+v", body)
	}
	if !body.HasPrevious || !body.SourceChanged {
		t.Fatalf("unexpected import diff flags: %+v", body)
	}
	if len(body.Changes) != 9 {
		t.Fatalf("change count = %d, want 9", len(body.Changes))
	}
}

func TestProjectVerificationBundleEndpoint(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	summary := project.ProjectSummary{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Import:    project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		HasImport: true,
	}
	readiness := project.ProjectReadinessFromSummary(summary)
	diff := project.ProjectImportDiffFromSummary(summary)
	bundle := project.ProjectVerificationBundleFromParts(summary, readiness, diff, []project.EventRecord{
		{
			ID:        7,
			Type:      "project.import.completed",
			Severity:  "info",
			Message:   "import completed",
			Metadata:  map[string]any{"overall_status": "pass"},
			CreatedAt: created,
		},
	})
	handler := NewHandler(fakeProjectStore{
		record: summary.Project,
		bundle: bundle,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects/areamatrix/verification-bundle?limit=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	var body projectVerificationBundleResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode verification bundle response: %v", err)
	}
	if body.Project.Key != "areamatrix" || body.Status == "" {
		t.Fatalf("unexpected bundle response: %+v", body)
	}
	if body.PhaseGate.Name != "v0.2-shadow-doctor" || body.PhaseGate.Status == "" {
		t.Fatalf("unexpected phase gate response: %+v", body.PhaseGate)
	}
	if len(body.Events) != 1 || body.Events[0].Type != "project.import.completed" {
		t.Fatalf("unexpected bundle events: %+v", body.Events)
	}
}

func TestProjectShadowDoctorEndpointsUseAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	summary := project.ProjectSummary{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "desktop-app",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Inventory: project.ImportInventory{
			Versions:        2,
			Residuals:       10,
			Artifacts:       6,
			ImportSnapshots: 2,
			MirrorExports:   1,
		},
		Import:              project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		PreviousImport:      project.Snapshot{SourceHash: "hash-a", CreatedAt: created.Add(-time.Minute)},
		HasImport:           true,
		HasPreviousImport:   true,
		HasLatestDoctor:     true,
		DoctorStatus:        "warn",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "warn",
		LatestEventCount:    1,
	}
	readiness := project.ProjectReadinessFromSummary(summary)
	diff := project.ProjectImportDiffFromSummary(summary)
	event := project.EventRecord{
		ID:        7,
		Type:      "project.doctor.completed",
		Severity:  "info",
		Message:   "doctor completed",
		Metadata:  map[string]any{"overall_status": "warn"},
		CreatedAt: created,
	}
	handler := NewHandler(fakeProjectStore{
		record:    summary.Project,
		summary:   summary,
		readiness: readiness,
		diff:      diff,
		bundle:    project.ProjectVerificationBundleFromParts(summary, readiness, diff, []project.EventRecord{event}),
		events:    []project.EventRecord{event},
	})

	tests := []struct {
		name string
		path string
	}{
		{name: "summary", path: "/api/v1/projects/areamatrix/summary"},
		{name: "readiness", path: "/api/v1/projects/areamatrix/readiness"},
		{name: "import diff", path: "/api/v1/projects/areamatrix/import-diff"},
		{name: "verification bundle", path: "/api/v1/projects/areamatrix/verification-bundle?limit=1"},
		{name: "events", path: "/api/v1/projects/areamatrix/events?limit=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if resp.Code != http.StatusOK {
				t.Fatalf("%s status = %d body=%s", tt.path, resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode %s response: %v", tt.path, err)
			}
			if len(body) == 0 {
				t.Fatalf("%s returned empty response", tt.path)
			}
		})
	}
}

func TestWorkflowVersionAuthoringEndpointsUseAPIV1(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 30, 0, 0, time.UTC)
	record := project.Record{ID: 1, Key: "areamatrix"}
	version := project.WorkflowVersion{
		ID:              7,
		ProjectID:       1,
		DisplayLabel:    "v2",
		VersionKind:     "workflow_version",
		LifecycleStatus: "draft",
		ImportMode:      "authored",
		StatusSummary: map[string]any{
			"phase": "v0.3a",
			"profile_binding": map[string]any{
				"profile_id":      "areamatrix",
				"profile_version": float64(0),
				"profile_hash":    "abc123",
			},
		},
		CreatedAt: created,
		UpdatedAt: created,
	}
	stageItem := project.WorkflowItem{
		ID:                10,
		WorkflowVersionID: 7,
		Stage:             "discussion",
		ItemType:          "discussion_package",
		ExternalKey:       "v2:discussion:discussion_package",
		Status:            "draft",
		Metadata:          map[string]any{"phase": "v0.3b"},
		CreatedAt:         created,
		UpdatedAt:         created,
	}
	gate := project.GateResult{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 7,
		WorkflowItemID:    10,
		GateName:          "discussion_gate",
		ScopeType:         "workflow_version",
		ScopeID:           "v2",
		Status:            "warn",
		Inputs:            map[string]any{"item_count": float64(1)},
		SourceHashes:      map[string]any{},
		Warnings:          []string{"discussion artifact is placeholder-only"},
		Metadata:          map[string]any{"phase": "v0.3c"},
		CheckedAt:         created,
	}
	preview := project.WorkflowTransitionPreview{
		ID:                4,
		ProjectID:         1,
		WorkflowVersionID: 7,
		FromStage:         "promotion_preview",
		ToStage:           "approval",
		Status:            "blocked",
		RequiredGateName:  "promotion_preview",
		Blockers:          []string{"latest promotion_preview gate status is fail"},
		Warnings:          []string{"transition preview is read-only"},
		Metadata:          map[string]any{"phase": "v0.3d"},
		CreatedAt:         created,
	}
	approval := project.ApprovalRecord{
		ID:                  5,
		ProjectID:           1,
		WorkflowVersionID:   7,
		TransitionPreviewID: 4,
		ApprovalKind:        "workflow_transition",
		Decision:            "rejected",
		ScopeType:           "workflow_version",
		ScopeID:             "v2",
		Actor:               "local-user",
		Reason:              "blocked preview",
		RiskLevel:           "normal",
		Metadata:            map[string]any{"phase": "v0.3d"},
		CreatedAt:           created,
	}
	handler := NewHandler(fakeProjectStore{
		record:   record,
		versions: []project.WorkflowVersion{version},
		version:  version,
		create: project.CreateWorkflowVersionResult{
			Project:        record,
			Version:        version,
			InitialItem:    project.WorkflowItem{ID: 9, WorkflowVersionID: 7, Stage: "version_init", ItemType: "workflow_version_candidate", ExternalKey: "v2:version_init", Status: "draft", CreatedAt: created, UpdatedAt: created},
			StageItems:     []project.WorkflowItem{stageItem},
			Created:        true,
			IdempotencyKey: "create-v2",
		},
		items: []project.WorkflowItem{stageItem},
		ensure: project.EnsureStageSkeletonResult{
			Project: record,
			Version: version,
			Items:   []project.WorkflowItem{stageItem},
			Links: []project.WorkflowItemLink{
				{ID: 20, WorkflowVersionID: 7, FromItemID: 10, ToItemID: 11, RelationType: "derives_from", Metadata: map[string]any{"source": "stage_skeleton"}, CreatedAt: created},
			},
			Created: 1,
		},
		gate:     gate,
		gates:    []project.GateResult{gate},
		preview:  preview,
		approval: approval,
	})

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantField  string
	}{
		{name: "list versions", method: http.MethodGet, path: "/api/v1/projects/areamatrix/workflow-versions", wantStatus: http.StatusOK, wantField: "workflow_versions"},
		{name: "create version", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workflow-versions", body: `{"display_label":"v2","idempotency_key":"create-v2","actor":"local-user","reason":"create v2"}`, wantStatus: http.StatusCreated, wantField: "workflow_version"},
		{name: "show version", method: http.MethodGet, path: "/api/v1/projects/areamatrix/workflow-versions/v2", wantStatus: http.StatusOK, wantField: "display_label"},
		{name: "list stages", method: http.MethodGet, path: "/api/v1/projects/areamatrix/workflow-versions/v2/stages", wantStatus: http.StatusOK, wantField: "items"},
		{name: "ensure stages", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workflow-versions/v2/stages", body: `{"actor":"local-user","reason":"ensure skeleton"}`, wantStatus: http.StatusOK, wantField: "links"},
		{name: "run gate", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workflow-versions/v2/gates", body: `{"gate_name":"discussion_gate","actor":"local-user","reason":"api gate smoke"}`, wantStatus: http.StatusOK, wantField: "gate_name"},
		{name: "list gates", method: http.MethodGet, path: "/api/v1/projects/areamatrix/workflow-versions/v2/gates?limit=1", wantStatus: http.StatusOK, wantField: "gate_results"},
		{name: "transition preview", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workflow-versions/v2/transition-previews", body: `{"actor":"local-user","reason":"api preview smoke"}`, wantStatus: http.StatusOK, wantField: "required_gate_name"},
		{name: "approval record", method: http.MethodPost, path: "/api/v1/projects/areamatrix/workflow-versions/v2/approvals", body: `{"decision":"rejected","transition_preview_id":4,"actor":"local-user","reason":"blocked preview"}`, wantStatus: http.StatusOK, wantField: "decision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if resp.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d body=%s", tt.method, tt.path, resp.Code, resp.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode %s response: %v", tt.path, err)
			}
			if _, ok := body[tt.wantField]; !ok {
				t.Fatalf("%s response missing %q: %+v", tt.path, tt.wantField, body)
			}
		})
	}
}

func splitAddr(addr string) (string, string, bool) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], true
		}
	}
	return "", "", false
}

func TestTrustedProxyMiddlewareRejectsSpoofedForwardedHeaders(t *testing.T) {
	server := Server{serverConfig: config.ServerConfig{TrustedProxyCIDRs: []string{"10.0.0.0/8"}}}
	handler := server.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.RemoteAddr = "192.0.2.4:4567"
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestTrustedProxyMiddlewareAllowsConfiguredProxy(t *testing.T) {
	server := Server{serverConfig: config.ServerConfig{TrustedProxyCIDRs: []string{"10.0.0.0/8"}}}
	handler := server.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.RemoteAddr = "10.2.3.4:4567"
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestAuthenticatedApprovalUsesPrincipalActor(t *testing.T) {
	record := project.Record{ID: 1, Key: "area"}
	version := project.WorkflowVersion{ID: 2, DisplayLabel: "v1", ImportMode: "authored"}
	var captured project.CreateApprovalOptions
	store := fakeProjectStore{
		record:       record,
		version:      version,
		approval:     project.ApprovalRecord{ID: 3, WorkflowVersionID: version.ID, Decision: "rejected"},
		approvalHook: func(options project.CreateApprovalOptions) { captured = options },
	}
	authenticator := &fakeTokenAuthenticator{principal: auth.Principal{
		Actor: "authenticated-approver", AuthMode: "token", TokenKey: "svc", Projects: []string{"area"},
		Capabilities: []string{"workflow.approval.record"},
	}}
	handler := NewHandlerWithAuth(store, nil, config.AuthConfig{Mode: "token"}, authenticator)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/area/workflow-versions/v1/approvals", strings.NewReader(`{"decision":"rejected","actor":"spoofed","reason":"test"}`))
	request.Header.Set("Authorization", "Bearer af_test")
	request.Header.Set("Idempotency-Key", "approval-test")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if captured.Actor != "authenticated-approver" {
		t.Fatalf("actor = %q, want authenticated principal", captured.Actor)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func ptrSecurityReadiness(readiness project.SecurityBoundaryReadiness) *project.SecurityBoundaryReadiness {
	return &readiness
}
