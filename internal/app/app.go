package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/api"
	"github.com/areasong/areaflow/internal/config"
	"github.com/areasong/areaflow/internal/db"
	"github.com/areasong/areaflow/internal/doctor"
	"github.com/areasong/areaflow/internal/importer"
	"github.com/areasong/areaflow/internal/migrate"
	"github.com/areasong/areaflow/internal/project"
	statusmirror "github.com/areasong/areaflow/internal/status"
	workflowprofile "github.com/areasong/areaflow/internal/workflow"
)

type command struct {
	stdout io.Writer
	stderr io.Writer
}

type projectSummaryJSON struct {
	Project   projectRecordJSON    `json:"project"`
	Config    *projectConfigJSON   `json:"config,omitempty"`
	Inventory projectInventoryJSON `json:"inventory"`
	Import    *projectImportJSON   `json:"import,omitempty"`
	Doctor    *projectDoctorJSON   `json:"doctor,omitempty"`
}

type localServiceStatusJSON struct {
	Status           string                     `json:"status"`
	Mode             string                     `json:"mode"`
	API              localServiceComponentJSON  `json:"api"`
	Database         localServiceComponentJSON  `json:"database"`
	WorkerPool       localServiceWorkerPoolJSON `json:"worker_pool"`
	Dashboard        localServiceDashboardJSON  `json:"dashboard"`
	Capabilities     []string                   `json:"capabilities"`
	ForbiddenActions []string                   `json:"forbidden_actions"`
	GeneratedAt      string                     `json:"generated_at"`
}

type localServiceComponentJSON struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type localServiceWorkerPoolJSON struct {
	Status             string `json:"status"`
	Message            string `json:"message"`
	TotalProjects      int64  `json:"total_projects"`
	TotalWorkers       int64  `json:"total_workers"`
	TotalOnlineWorkers int64  `json:"total_online_workers"`
	TotalActiveLeases  int64  `json:"total_active_leases"`
	TotalQueuedTasks   int64  `json:"total_queued_tasks"`
	TotalNeedsRecovery int64  `json:"total_needs_recovery"`
}

type localServiceDashboardJSON struct {
	URL     string `json:"url"`
	APIURL  string `json:"api_url"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type desktopServiceControlGateJSON struct {
	Status                   string                            `json:"status"`
	Mode                     string                            `json:"mode"`
	Actions                  []desktopServiceControlActionJSON `json:"actions"`
	Capabilities             []string                          `json:"capabilities"`
	ForbiddenActions         []string                          `json:"forbidden_actions"`
	GeneratedAt              string                            `json:"generated_at"`
	DBWriteAttempted         bool                              `json:"db_write_attempted"`
	ProjectWriteAttempted    bool                              `json:"project_write_attempted"`
	ProcessControlAttempted  bool                              `json:"process_control_attempted"`
	CommandCreated           bool                              `json:"command_created"`
	ApprovalCreated          bool                              `json:"approval_created"`
	AuditEventWritten        bool                              `json:"audit_event_written"`
	WorkerScheduled          bool                              `json:"worker_scheduled"`
	WorkflowExecutionStarted bool                              `json:"workflow_execution_started"`
	SecretsResolved          bool                              `json:"secrets_resolved"`
	NetworkUsed              bool                              `json:"network_used"`
}

type desktopServiceControlActionJSON struct {
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

type desktopNotificationGateJSON struct {
	Status                   string                          `json:"status"`
	Mode                     string                          `json:"mode"`
	Actions                  []desktopNotificationActionJSON `json:"actions"`
	Capabilities             []string                        `json:"capabilities"`
	ForbiddenActions         []string                        `json:"forbidden_actions"`
	GeneratedAt              string                          `json:"generated_at"`
	DBWriteAttempted         bool                            `json:"db_write_attempted"`
	ProjectWriteAttempted    bool                            `json:"project_write_attempted"`
	EventStreamOpened        bool                            `json:"event_stream_opened"`
	NotificationRequested    bool                            `json:"notification_requested"`
	CommandCreated           bool                            `json:"command_created"`
	ApprovalCreated          bool                            `json:"approval_created"`
	AuditEventWritten        bool                            `json:"audit_event_written"`
	WorkerScheduled          bool                            `json:"worker_scheduled"`
	WorkflowExecutionStarted bool                            `json:"workflow_execution_started"`
	SecretsResolved          bool                            `json:"secrets_resolved"`
	NetworkUsed              bool                            `json:"network_used"`
}

type desktopNotificationActionJSON struct {
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

type desktopTrayMenuGateJSON struct {
	Status                   string                      `json:"status"`
	Mode                     string                      `json:"mode"`
	Actions                  []desktopTrayMenuActionJSON `json:"actions"`
	Capabilities             []string                    `json:"capabilities"`
	ForbiddenActions         []string                    `json:"forbidden_actions"`
	GeneratedAt              string                      `json:"generated_at"`
	DBWriteAttempted         bool                        `json:"db_write_attempted"`
	ProjectWriteAttempted    bool                        `json:"project_write_attempted"`
	TrayMenuCreated          bool                        `json:"tray_menu_created"`
	OSIntegrationRequested   bool                        `json:"os_integration_requested"`
	CommandCreated           bool                        `json:"command_created"`
	ApprovalCreated          bool                        `json:"approval_created"`
	AuditEventWritten        bool                        `json:"audit_event_written"`
	ServiceControlAttempted  bool                        `json:"service_control_attempted"`
	NotificationRequested    bool                        `json:"notification_requested"`
	WorkerScheduled          bool                        `json:"worker_scheduled"`
	WorkflowExecutionStarted bool                        `json:"workflow_execution_started"`
	SecretsResolved          bool                        `json:"secrets_resolved"`
	NetworkUsed              bool                        `json:"network_used"`
}

type desktopTrayMenuActionJSON struct {
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

type securityBoundaryReadinessJSON struct {
	Status                        string                              `json:"status"`
	Mode                          string                              `json:"mode"`
	Items                         []securityBoundaryReadinessItemJSON `json:"items"`
	Capabilities                  []string                            `json:"capabilities"`
	ForbiddenActions              []string                            `json:"forbidden_actions"`
	GeneratedAt                   string                              `json:"generated_at"`
	AuthEnforcementOpen           bool                                `json:"auth_enforcement_open"`
	TeamPermissionEnforcementOpen bool                                `json:"team_permission_enforcement_open"`
	APITokenIssuanceOpen          bool                                `json:"api_token_issuance_open"`
	APITokenEnforcementOpen       bool                                `json:"api_token_enforcement_open"`
	SecretResolveOpen             bool                                `json:"secret_resolve_open"`
	RemoteWorkerCredentialsOpen   bool                                `json:"remote_worker_credentials_open"`
	BudgetEnforcementOpen         bool                                `json:"budget_enforcement_open"`
	QuotaDecrementOpen            bool                                `json:"quota_decrement_open"`
	UsageChargeWritten            bool                                `json:"usage_charge_written"`
	WebhookDeliveryOpen           bool                                `json:"webhook_delivery_open"`
	InboundCallbackOpen           bool                                `json:"inbound_callback_open"`
	ExternalAPICallOpen           bool                                `json:"external_api_call_open"`
	AuthorizationChanged          bool                                `json:"authorization_changed"`
	SecretPlaintextRead           bool                                `json:"secret_plaintext_read"`
	RemoteWorkerDirectPGAllowed   bool                                `json:"remote_worker_direct_pg_allowed"`
	TeamConsoleCommandOpen        bool                                `json:"team_console_command_open"`
	RemoteOpsControlOpen          bool                                `json:"remote_ops_control_open"`
	ManagedUpgradeOpen            bool                                `json:"managed_upgrade_open"`
	SupportBundleExportOpen       bool                                `json:"support_bundle_export_open"`
	DefaultRemoteTelemetryOpen    bool                                `json:"default_remote_telemetry_open"`
}

type securityBoundaryReadinessItemJSON struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	RequiredEvidence []string       `json:"required_evidence"`
	BlockedBy        []string       `json:"blocked_by"`
	Metadata         map[string]any `json:"metadata"`
}

type completionAuditJSON struct {
	Status                     string                    `json:"status"`
	Mode                       string                    `json:"mode"`
	Scope                      string                    `json:"scope"`
	ReadinessScope             string                    `json:"readiness_scope"`
	ClaimScope                 string                    `json:"claim_scope"`
	NotReal100                 bool                      `json:"not_real_100"`
	EvidenceOnly               bool                      `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                      `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                    `json:"release_candidate_decision"`
	Real100Status              string                    `json:"real_100_status"`
	Real100Blockers            []string                  `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown  `json:"real_100_breakdown"`
	Items                      []completionAuditItemJSON `json:"items"`
	DeferredV1x                []string                  `json:"deferred_v1x"`
	Capabilities               []string                  `json:"capabilities"`
	ForbiddenActions           []string                  `json:"forbidden_actions"`
	SafetyFacts                map[string]bool           `json:"safety_facts"`
	ReleaseFinalGateStatus     string                    `json:"release_final_gate_status"`
	AreaMatrixDogfoodStatus    string                    `json:"area_matrix_dogfood_status"`
	TaskMatrixStatus           string                    `json:"task_matrix_status"`
	ImplementationGapStatus    string                    `json:"implementation_gap_status"`
	ProtectedPathProofStatus   string                    `json:"protected_path_proof_status"`
	GeneratedAt                string                    `json:"generated_at"`
}

type completionAuditItemJSON struct {
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

type completionAuditSnapshotJSON struct {
	Project                         projectRecordJSON        `json:"project"`
	Status                          string                   `json:"status"`
	Decision                        string                   `json:"decision"`
	Message                         string                   `json:"message"`
	AuditStatus                     string                   `json:"audit_status"`
	AuditScope                      string                   `json:"audit_scope"`
	ReadinessScope                  string                   `json:"readiness_scope"`
	ClaimScope                      string                   `json:"claim_scope"`
	NotReal100                      bool                     `json:"not_real_100"`
	EvidenceOnly                    bool                     `json:"evidence_only"`
	StatusAloneIsNotCompletion      bool                     `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision        string                   `json:"release_candidate_decision"`
	Real100Status                   string                   `json:"real_100_status"`
	Real100Blockers                 []string                 `json:"real_100_blockers"`
	Real100Breakdown                project.Real100Breakdown `json:"real_100_breakdown"`
	AuditHash                       string                   `json:"audit_hash"`
	ReleaseCandidateLabel           string                   `json:"release_candidate_label"`
	EvidenceClass                   string                   `json:"evidence_class"`
	EvidenceURI                     string                   `json:"evidence_uri"`
	ProofEventIDs                   map[string]int64         `json:"proof_event_ids"`
	EventID                         int64                    `json:"event_id,omitempty"`
	AuditEventID                    int64                    `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string                   `json:"idempotency_key"`
	Created                         bool                     `json:"created"`
	ProjectWriteAttempted           bool                     `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool                     `json:"execution_write_attempted"`
	ReleasePackageCreated           bool                     `json:"release_package_created"`
	PublishAttempted                bool                     `json:"publish_attempted"`
	RestoreApplyAttempted           bool                     `json:"restore_apply_attempted"`
	SecretResolved                  bool                     `json:"secret_resolved"`
	RemoteWorkerCredentialsIssued   bool                     `json:"remote_worker_credentials_issued"`
	AreaMatrixProtectedPathsTouched bool                     `json:"area_matrix_protected_paths_touched"`
	CommandsRun                     bool                     `json:"commands_run"`
	SmokeRunAttempted               bool                     `json:"smoke_run_attempted"`
	WorkerStarted                   bool                     `json:"worker_started"`
	Metadata                        map[string]any           `json:"metadata"`
}

type completionAuditSnapshotReadinessJSON struct {
	Project                    projectRecordJSON                      `json:"project"`
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
	Latest                     completionAuditSnapshotJSON            `json:"latest"`
	Items                      []projectReadinessItemJSON             `json:"items"`
	Gaps                       []project.CompletionAuditSnapshotGap   `json:"gaps"`
	Closure                    project.CompletionAuditSnapshotClosure `json:"closure"`
	SafetyFacts                map[string]bool                        `json:"safety_facts"`
}

type supportBundlePreviewJSON struct {
	Status                   string                           `json:"status"`
	Mode                     string                           `json:"mode"`
	BundleID                 string                           `json:"bundle_id"`
	Scope                    string                           `json:"scope"`
	Projects                 []projectRecordJSON              `json:"projects"`
	IncludedMetadata         []string                         `json:"included_metadata"`
	ExcludedSensitiveContent []string                         `json:"excluded_sensitive_content"`
	PathReferences           []supportBundlePathReferenceJSON `json:"path_references"`
	Hashes                   []supportBundleHashReferenceJSON `json:"hashes"`
	Capabilities             []string                         `json:"capabilities"`
	ForbiddenActions         []string                         `json:"forbidden_actions"`
	SafetyFacts              map[string]bool                  `json:"safety_facts"`
	GeneratedAt              string                           `json:"generated_at"`
}

type supportBundlePathReferenceJSON struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"`
	URI         string `json:"uri"`
	ProjectKey  string `json:"project_key,omitempty"`
	Description string `json:"description"`
}

type supportBundleHashReferenceJSON struct {
	Key         string `json:"key"`
	Hash        string `json:"hash"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type migrationLedgerReadinessJSON struct {
	Status                               string                     `json:"status"`
	Mode                                 string                     `json:"mode"`
	Entries                              []migrationLedgerEntryJSON `json:"entries"`
	AppliedCount                         int                        `json:"applied_count"`
	PendingCount                         int                        `json:"pending_count"`
	SchemaMigrationsTablePresent         bool                       `json:"schema_migrations_table_present"`
	FullLedgerTablePresent               bool                       `json:"full_ledger_table_present"`
	PreflightApplyVerifyRemediationReady bool                       `json:"preflight_apply_verify_remediation_ready"`
	Capabilities                         []string                   `json:"capabilities"`
	ForbiddenActions                     []string                   `json:"forbidden_actions"`
	SafetyFacts                          map[string]bool            `json:"safety_facts"`
	GeneratedAt                          string                     `json:"generated_at"`
}

type migrationLedgerEntryJSON struct {
	Name             string                     `json:"name"`
	Applied          bool                       `json:"applied"`
	Status           string                     `json:"status"`
	RequiredEvidence []string                   `json:"required_evidence"`
	Phases           []migrationLedgerPhaseJSON `json:"phases"`
	Metadata         map[string]any             `json:"metadata"`
}

type migrationLedgerPhaseJSON struct {
	Phase       string         `json:"phase"`
	Status      string         `json:"status"`
	Message     string         `json:"message"`
	Remediation string         `json:"remediation"`
	Metadata    map[string]any `json:"metadata"`
}

type operationsReadinessJSON struct {
	Status              string                        `json:"status"`
	Mode                string                        `json:"mode"`
	Items               []operationsReadinessItemJSON `json:"items"`
	ServiceStatus       localServiceStatusJSON        `json:"service_status"`
	SupportBundle       supportBundlePreviewJSON      `json:"support_bundle"`
	MigrationLedger     migrationLedgerReadinessJSON  `json:"migration_ledger"`
	Capabilities        []string                      `json:"capabilities"`
	ForbiddenActions    []string                      `json:"forbidden_actions"`
	SafetyFacts         map[string]bool               `json:"safety_facts"`
	TelemetryDefault    string                        `json:"telemetry_default"`
	ManagedOpsStatus    string                        `json:"managed_ops_status"`
	SupportExportStatus string                        `json:"support_export_status"`
	GeneratedAt         string                        `json:"generated_at"`
}

type operationsSmokeProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	ProofKey                        string            `json:"proof_key"`
	Status                          string            `json:"status"`
	EvidenceStatus                  string            `json:"evidence_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	EngineCallAttempted             bool              `json:"engine_call_attempted"`
	ServiceProcessControlAttempted  bool              `json:"service_process_control_attempted"`
	SupportBundleExported           bool              `json:"support_bundle_exported"`
	MigrationApplyAttempted         bool              `json:"migration_apply_attempted"`
	RemoteTelemetryEnabled          bool              `json:"remote_telemetry_enabled"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	RecordCommandRunsSmoke          bool              `json:"record_command_runs_smoke"`
	Metadata                        map[string]any    `json:"metadata"`
}

type protectedPathProofJSON struct {
	Project                           projectRecordJSON `json:"project"`
	Status                            string            `json:"status"`
	ProofStatus                       string            `json:"proof_status"`
	Decision                          string            `json:"decision"`
	Message                           string            `json:"message"`
	EventID                           int64             `json:"event_id,omitempty"`
	AuditEventID                      int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                    string            `json:"idempotency_key"`
	Created                           bool              `json:"created"`
	ProjectWriteAttempted             bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted           bool              `json:"execution_write_attempted"`
	EngineCallAttempted               bool              `json:"engine_call_attempted"`
	CommandsRun                       bool              `json:"commands_run"`
	GitStatusRunByCommand             bool              `json:"git_status_run_by_command"`
	AreaMatrixProtectedPathsTouched   bool              `json:"area_matrix_protected_paths_touched"`
	GitStatusOutputHash               string            `json:"git_status_output_hash"`
	GitStatusOutputLines              int               `json:"git_status_output_lines"`
	GitStatusOutputEmpty              bool              `json:"git_status_output_empty"`
	ProtectedPathSetHash              string            `json:"protected_path_set_hash"`
	ProtectedPathSetCount             int64             `json:"protected_path_set_count"`
	ProtectedPathProofBindingStatus   string            `json:"protected_path_proof_binding_status"`
	ProtectedPathProofBindingBlockers []string          `json:"protected_path_proof_binding_blockers"`
	Summary                           string            `json:"summary,omitempty"`
	EvidenceURI                       string            `json:"evidence_uri,omitempty"`
	AuthorizedApprovalID              string            `json:"authorized_approval_id,omitempty"`
	AuthorizedAllowedPaths            []string          `json:"authorized_allowed_paths,omitempty"`
	AuthorizedDirtyOutputHash         string            `json:"authorized_dirty_output_hash,omitempty"`
	AuthorizedReviewer                string            `json:"authorized_reviewer,omitempty"`
	AuthorizedRollbackEvidenceURI     string            `json:"authorized_rollback_evidence_uri,omitempty"`
	AuthorizedTouchedPaths            []string          `json:"authorized_touched_paths,omitempty"`
	AuthorizedProofComplete           *bool             `json:"authorized_proof_complete,omitempty"`
	Metadata                          map[string]any    `json:"metadata"`
}

type archiveProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	Status                          string            `json:"status"`
	ProofStatus                     string            `json:"proof_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	Facts                           []string          `json:"facts"`
	MissingFacts                    []string          `json:"missing_facts"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	ArtifactBytesCopied             bool              `json:"artifact_bytes_copied"`
	ArtifactBytesDeleted            bool              `json:"artifact_bytes_deleted"`
	HistoricalFilesDeleted          bool              `json:"historical_files_deleted"`
	HistoricalFilesMoved            bool              `json:"historical_files_moved"`
	ProgressJSONRewritten           bool              `json:"progress_json_rewritten"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	CommandsRun                     bool              `json:"commands_run"`
	ArchiveScopeBindingStatus       string            `json:"archive_scope_binding_status"`
	ArchiveScopeBindingBlockers     []string          `json:"archive_scope_binding_blockers"`
	ArchiveBindingContract          string            `json:"archive_binding_contract"`
	ArchiveSourcePathsHash          string            `json:"archive_source_paths_hash"`
	ArchiveForbiddenActionsHash     string            `json:"archive_forbidden_actions_hash"`
	ArchiveScopeBindingHash         string            `json:"archive_scope_binding_hash"`
	ArchiveScope                    string            `json:"archive_scope"`
	ArchiveReferenceMode            string            `json:"archive_reference_mode"`
	ArchiveSourcePaths              []string          `json:"archive_source_paths"`
	ArchiveForbiddenActions         []string          `json:"archive_forbidden_actions"`
	ArchiveRollbackTarget           string            `json:"archive_rollback_target"`
	ArchiveFailClosed               bool              `json:"archive_fail_closed"`
	ReviewDecision                  string            `json:"review_decision,omitempty"`
	ReviewedBy                      string            `json:"reviewed_by,omitempty"`
	ReviewedAt                      string            `json:"reviewed_at,omitempty"`
	ReviewMetadataStatus            string            `json:"review_metadata_status"`
	ReviewMetadataBlockers          []string          `json:"review_metadata_blockers"`
	Metadata                        map[string]any    `json:"metadata"`
}

type shimRetirementProofJSON struct {
	Project                            projectRecordJSON `json:"project"`
	Status                             string            `json:"status"`
	ProofStatus                        string            `json:"proof_status"`
	Decision                           string            `json:"decision"`
	Message                            string            `json:"message"`
	Facts                              []string          `json:"facts"`
	MissingFacts                       []string          `json:"missing_facts"`
	EventID                            int64             `json:"event_id,omitempty"`
	AuditEventID                       int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                     string            `json:"idempotency_key"`
	Created                            bool              `json:"created"`
	ProjectWriteAttempted              bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted            bool              `json:"execution_write_attempted"`
	CommandsRun                        bool              `json:"commands_run"`
	LegacyRunnerStarted                bool              `json:"legacy_runner_started"`
	LegacyProgressWritten              bool              `json:"legacy_progress_written"`
	LegacyLogsWritten                  bool              `json:"legacy_logs_written"`
	LegacyCheckpointWritten            bool              `json:"legacy_checkpoint_written"`
	HistoricalFilesDeleted             bool              `json:"historical_files_deleted"`
	ProgressJSONRewritten              bool              `json:"progress_json_rewritten"`
	AreaMatrixProtectedPathsTouched    bool              `json:"area_matrix_protected_paths_touched"`
	ShimRetirementScopeBindingStatus   string            `json:"shim_retirement_scope_binding_status"`
	ShimRetirementScopeBindingBlockers []string          `json:"shim_retirement_scope_binding_blockers"`
	ShimRetirementBindingContract      string            `json:"shim_retirement_binding_contract"`
	ShimRetirementPrerequisitesHash    string            `json:"shim_retirement_prerequisites_hash"`
	ShimRetiredSurfacesHash            string            `json:"shim_retired_surfaces_hash"`
	ShimRetirementScopeBindingHash     string            `json:"shim_retirement_scope_binding_hash"`
	ShimRetirementScope                string            `json:"shim_retirement_scope"`
	ShimRetirementPrerequisites        []string          `json:"shim_retirement_prerequisites"`
	ShimRetiredSurfaces                []string          `json:"shim_retired_surfaces"`
	ShimRollbackTarget                 string            `json:"shim_rollback_target"`
	ShimFailClosed                     bool              `json:"shim_fail_closed"`
	ShimReopenRequiresApproval         bool              `json:"shim_reopen_requires_approval"`
	ReviewDecision                     string            `json:"review_decision,omitempty"`
	ReviewedBy                         string            `json:"reviewed_by,omitempty"`
	ReviewedAt                         string            `json:"reviewed_at,omitempty"`
	ReviewMetadataStatus               string            `json:"review_metadata_status"`
	ReviewMetadataBlockers             []string          `json:"review_metadata_blockers"`
	Metadata                           map[string]any    `json:"metadata"`
}

type executionCutoverProofJSON struct {
	Project                              projectRecordJSON `json:"project"`
	Status                               string            `json:"status"`
	ProofStatus                          string            `json:"proof_status"`
	Decision                             string            `json:"decision"`
	Message                              string            `json:"message"`
	Facts                                []string          `json:"facts"`
	MissingFacts                         []string          `json:"missing_facts"`
	EventID                              int64             `json:"event_id,omitempty"`
	AuditEventID                         int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                       string            `json:"idempotency_key"`
	Created                              bool              `json:"created"`
	ExecutionCutoverScope                string            `json:"execution_cutover_scope"`
	AllowedTaskTypes                     []string          `json:"allowed_task_types"`
	ForbiddenActions                     []string          `json:"forbidden_actions"`
	RollbackTarget                       string            `json:"rollback_target"`
	RollbackMode                         string            `json:"rollback_mode"`
	FailClosed                           bool              `json:"fail_closed"`
	ReopenRequiresApproval               bool              `json:"reopen_requires_approval"`
	SourceWriteOpen                      bool              `json:"source_write_open"`
	GeneratedRetainedWriteOpen           bool              `json:"generated_retained_write_open"`
	RepairApplyOpen                      bool              `json:"repair_apply_open"`
	CheckpointApplyOpen                  bool              `json:"checkpoint_apply_open"`
	EngineExecutionOpen                  bool              `json:"engine_execution_open"`
	SecretResolveOpen                    bool              `json:"secret_resolve_open"`
	NetworkAPIIntegrationOpen            bool              `json:"network_api_integration_open"`
	PublishApplyOpen                     bool              `json:"publish_apply_open"`
	RestoreApplyOpen                     bool              `json:"restore_apply_open"`
	ProjectWriteAttempted                bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted              bool              `json:"execution_write_attempted"`
	TaskLoopRunForwardedByCommand        bool              `json:"task_loop_run_forwarded_by_command"`
	EngineCallAttempted                  bool              `json:"engine_call_attempted"`
	CommandsRun                          bool              `json:"commands_run"`
	LegacyProgressWritten                bool              `json:"legacy_progress_written"`
	LegacyLogsWritten                    bool              `json:"legacy_logs_written"`
	LegacyCheckpointWritten              bool              `json:"legacy_checkpoint_written"`
	AreaMatrixProtectedPathsTouched      bool              `json:"area_matrix_protected_paths_touched"`
	ExecutionCutoverScopeBindingStatus   string            `json:"execution_cutover_scope_binding_status"`
	ExecutionCutoverScopeBindingBlockers []string          `json:"execution_cutover_scope_binding_blockers"`
	ExecutionCutoverBindingContract      string            `json:"execution_cutover_binding_contract"`
	AllowedTaskTypesHash                 string            `json:"allowed_task_types_hash"`
	ForbiddenActionsHash                 string            `json:"forbidden_actions_hash"`
	ExecutionCutoverBindingHash          string            `json:"execution_cutover_binding_hash"`
	ExecutionCutoverScopeBindingHash     string            `json:"execution_cutover_scope_binding_hash"`
	ReviewDecision                       string            `json:"review_decision,omitempty"`
	ReviewedBy                           string            `json:"reviewed_by,omitempty"`
	ReviewedAt                           string            `json:"reviewed_at,omitempty"`
	ReviewMetadataStatus                 string            `json:"review_metadata_status"`
	ReviewMetadataBlockers               []string          `json:"review_metadata_blockers"`
	Metadata                             map[string]any    `json:"metadata"`
}

type validationProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	Status                          string            `json:"status"`
	ProofStatus                     string            `json:"proof_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	Facts                           []string          `json:"facts"`
	MissingFacts                    []string          `json:"missing_facts"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	EngineCallAttempted             bool              `json:"engine_call_attempted"`
	CommandsRun                     bool              `json:"commands_run"`
	SmokeRunAttempted               bool              `json:"smoke_run_attempted"`
	WebBuildRunByCommand            bool              `json:"web_build_run_by_command"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	Metadata                        map[string]any    `json:"metadata"`
}

type sourceAlignmentProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	Status                          string            `json:"status"`
	ProofStatus                     string            `json:"proof_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	Facts                           []string          `json:"facts"`
	MissingFacts                    []string          `json:"missing_facts"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	CommandsRun                     bool              `json:"commands_run"`
	DocsWritten                     bool              `json:"docs_written"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	SourceAlignmentBindingStatus    string            `json:"source_alignment_binding_status"`
	SourceAlignmentBindingBlockers  []string          `json:"source_alignment_binding_blockers"`
	SourceAlignmentSourcePaths      []string          `json:"source_alignment_source_paths"`
	SourceAlignmentSourceHashes     map[string]string `json:"source_alignment_source_hashes"`
	SourceAlignmentSourceSetHash    string            `json:"source_alignment_source_set_hash"`
	SourceAlignmentSourceFileCount  int64             `json:"source_alignment_source_file_count"`
	MissingSourceCount              int64             `json:"source_alignment_missing_source_count"`
	UnreadableSourceCount           int64             `json:"source_alignment_unreadable_source_count"`
	Metadata                        map[string]any    `json:"metadata"`
}

type taskMatrixProofJSON struct {
	Project                            projectRecordJSON `json:"project"`
	Status                             string            `json:"status"`
	ProofStatus                        string            `json:"proof_status"`
	Decision                           string            `json:"decision"`
	Message                            string            `json:"message"`
	Facts                              []string          `json:"facts"`
	MissingFacts                       []string          `json:"missing_facts"`
	EventID                            int64             `json:"event_id,omitempty"`
	AuditEventID                       int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                     string            `json:"idempotency_key"`
	Created                            bool              `json:"created"`
	ProjectWriteAttempted              bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted            bool              `json:"execution_write_attempted"`
	CommandsRun                        bool              `json:"commands_run"`
	DocsWritten                        bool              `json:"docs_written"`
	TasksWritten                       bool              `json:"tasks_written"`
	AreaMatrixProtectedPathsTouched    bool              `json:"area_matrix_protected_paths_touched"`
	TaskMatrixBindingStatus            string            `json:"task_matrix_binding_status"`
	TaskMatrixBindingBlockers          []string          `json:"task_matrix_binding_blockers"`
	TaskMatrixSourcePaths              []string          `json:"task_matrix_source_paths"`
	TaskMatrixSourceSetHash            string            `json:"task_matrix_source_set_hash"`
	TaskBacklogHash                    string            `json:"task_backlog_hash"`
	TaskStatusAuditHash                string            `json:"task_status_audit_hash"`
	PlannedV1RequiredTaskCount         int64             `json:"planned_v1_required_task_count"`
	MissingEvidenceV1RequiredTaskCount int64             `json:"missing_evidence_v1_required_task_count"`
	BlockedV1RequiredTaskCount         int64             `json:"blocked_v1_required_task_count"`
	Metadata                           map[string]any    `json:"metadata"`
}

type securityClosureProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	Status                          string            `json:"status"`
	ProofStatus                     string            `json:"proof_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	Facts                           []string          `json:"facts"`
	MissingFacts                    []string          `json:"missing_facts"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	AuthorizationChanged            bool              `json:"authorization_changed"`
	SecretPlaintextRead             bool              `json:"secret_plaintext_read"`
	RemoteWorkerCredentialsIssued   bool              `json:"remote_worker_credentials_issued"`
	CommandsRun                     bool              `json:"commands_run"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	SecurityClosureBindingStatus    string            `json:"security_closure_binding_status"`
	SecurityClosureBindingBlockers  []string          `json:"security_closure_binding_blockers"`
	SecurityClosureBindingHash      string            `json:"security_closure_binding_hash"`
	SecurityBoundaryStatus          string            `json:"security_boundary_status"`
	PermissionDoctorStatus          string            `json:"permission_doctor_status"`
	AuditCoverageStatus             string            `json:"audit_coverage_status"`
	Metadata                        map[string]any    `json:"metadata"`
}

type backupRestoreProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	Status                          string            `json:"status"`
	ProofStatus                     string            `json:"proof_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	Facts                           []string          `json:"facts"`
	MissingFacts                    []string          `json:"missing_facts"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	DatabaseRestoreAttempted        bool              `json:"database_restore_attempted"`
	ArtifactBytesCopied             bool              `json:"artifact_bytes_copied"`
	ArtifactBytesDeleted            bool              `json:"artifact_bytes_deleted"`
	ArtifactBytesUploaded           bool              `json:"artifact_bytes_uploaded"`
	ArtifactGCAttempted             bool              `json:"artifact_gc_attempted"`
	CommandsRun                     bool              `json:"commands_run"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	RestorePlanScope                string            `json:"restore_plan_scope"`
	RestorePlanProjectKey           string            `json:"restore_plan_project_key"`
	Metadata                        map[string]any    `json:"metadata"`
}

type releasePackagingProofJSON struct {
	Project                         projectRecordJSON `json:"project"`
	Status                          string            `json:"status"`
	ProofStatus                     string            `json:"proof_status"`
	Decision                        string            `json:"decision"`
	Message                         string            `json:"message"`
	Facts                           []string          `json:"facts"`
	MissingFacts                    []string          `json:"missing_facts"`
	EventID                         int64             `json:"event_id,omitempty"`
	AuditEventID                    int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string            `json:"idempotency_key"`
	Created                         bool              `json:"created"`
	ProjectWriteAttempted           bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool              `json:"execution_write_attempted"`
	ReleasePackageCreated           bool              `json:"release_package_created"`
	ReleaseStateWritten             bool              `json:"release_state_written"`
	ReleaseApprovalCreated          bool              `json:"release_approval_created"`
	RolloutStateCreated             bool              `json:"rollout_state_created"`
	MigrationApplyAttempted         bool              `json:"migration_apply_attempted"`
	TagCreated                      bool              `json:"tag_created"`
	PackageSigned                   bool              `json:"package_signed"`
	ArtifactUploaded                bool              `json:"artifact_uploaded"`
	GitPushAttempted                bool              `json:"git_push_attempted"`
	PublishAttempted                bool              `json:"publish_attempted"`
	CommandsRun                     bool              `json:"commands_run"`
	AreaMatrixProtectedPathsTouched bool              `json:"area_matrix_protected_paths_touched"`
	Metadata                        map[string]any    `json:"metadata"`
}

type operationsReadinessItemJSON struct {
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

type backupManifestJSON struct {
	Status           string                      `json:"status"`
	Mode             string                      `json:"mode"`
	Scope            string                      `json:"scope"`
	ProjectKey       string                      `json:"project_key,omitempty"`
	SchemaVersion    int                         `json:"schema_version"`
	GeneratedAt      string                      `json:"generated_at"`
	ManifestHash     string                      `json:"manifest_hash"`
	TableCounts      []backupTableCountJSON      `json:"table_counts"`
	Projects         []backupProjectManifestJSON `json:"projects"`
	Capabilities     []string                    `json:"capabilities"`
	ForbiddenActions []string                    `json:"forbidden_actions"`
}

type backupTableCountJSON struct {
	Table string `json:"table"`
	Rows  int64  `json:"rows"`
}

type backupProjectManifestJSON struct {
	Project       projectRecordJSON           `json:"project"`
	Inventory     projectInventoryJSON        `json:"inventory"`
	ArtifactCount int64                       `json:"artifact_count"`
	Artifacts     []backupArtifactSummaryJSON `json:"artifacts"`
}

type backupArtifactSummaryJSON struct {
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

type restorePlanJSON struct {
	Status           string                `json:"status"`
	Mode             string                `json:"mode"`
	Scope            string                `json:"scope"`
	ProjectKey       string                `json:"project_key,omitempty"`
	SchemaVersion    int                   `json:"schema_version"`
	ManifestHash     string                `json:"manifest_hash"`
	Projects         []projectRecordJSON   `json:"projects"`
	Items            []restorePlanItemJSON `json:"items"`
	Capabilities     []string              `json:"capabilities"`
	ForbiddenActions []string              `json:"forbidden_actions"`
	GeneratedAt      string                `json:"generated_at"`
}

type backupScopeFlags struct {
	json       bool
	projectKey string
}

type releaseExceptionMigrationFlags struct {
	json   bool
	actor  string
	reason string
}

type releaseExceptionCommandFlags struct {
	json           bool
	projectKey     string
	exceptionKey   string
	actor          string
	reason         string
	owner          string
	reviewAt       *time.Time
	expiresAt      *time.Time
	idempotencyKey string
}

type releaseExceptionRecordJSON struct {
	ID               int64          `json:"id"`
	ProjectID        int64          `json:"project_id"`
	ProjectKey       string         `json:"project_key"`
	ExceptionKey     string         `json:"exception_key"`
	SourceGateItem   string         `json:"source_gate_item"`
	SourceDecision   string         `json:"source_decision"`
	AcceptanceType   string         `json:"acceptance_type"`
	Status           string         `json:"status"`
	Owner            string         `json:"owner"`
	Reason           string         `json:"reason"`
	RequiredEvidence []string       `json:"required_evidence"`
	RollbackPlan     string         `json:"rollback_plan"`
	ReviewRequired   bool           `json:"review_required"`
	ReviewAt         string         `json:"review_at,omitempty"`
	ExpiresAt        string         `json:"expires_at,omitempty"`
	RequestedBy      string         `json:"requested_by"`
	ApprovedBy       string         `json:"approved_by,omitempty"`
	RevokedBy        string         `json:"revoked_by,omitempty"`
	DecisionReason   string         `json:"decision_reason,omitempty"`
	AuditEventID     int64          `json:"audit_event_id"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
	ApprovedAt       string         `json:"approved_at,omitempty"`
	RevokedAt        string         `json:"revoked_at,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key"`
	Created          bool           `json:"created"`
}

type restorePlanItemJSON struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type releaseReadinessJSON struct {
	Status                     string                        `json:"status"`
	Mode                       string                        `json:"mode"`
	Scope                      string                        `json:"scope"`
	ProjectKey                 string                        `json:"project_key,omitempty"`
	ReadinessScope             string                        `json:"readiness_scope"`
	ClaimScope                 string                        `json:"claim_scope"`
	NotReal100                 bool                          `json:"not_real_100"`
	EvidenceOnly               bool                          `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                          `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                        `json:"release_candidate_decision"`
	Real100Status              string                        `json:"real_100_status"`
	Real100Blockers            []string                      `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown      `json:"real_100_breakdown"`
	Backup                     backupManifestJSON            `json:"backup"`
	RestorePlan                restorePlanJSON               `json:"restore_plan"`
	AuditCoverage              auditCoverageJSON             `json:"audit_coverage"`
	Projects                   []releaseReadinessProjectJSON `json:"projects"`
	Items                      []releaseReadinessItemJSON    `json:"items"`
	Capabilities               []string                      `json:"capabilities"`
	ForbiddenActions           []string                      `json:"forbidden_actions"`
	GeneratedAt                string                        `json:"generated_at"`
}

type releaseReadinessProjectJSON struct {
	Project             projectRecordJSON          `json:"project"`
	Permission          permissionPolicyDoctorJSON `json:"permission"`
	ArtifactIntegrity   artifactIntegrityJSON      `json:"artifact_integrity"`
	Conformance         conformanceJSON            `json:"conformance"`
	Status              string                     `json:"status"`
	NeedsAttentionItems int                        `json:"needs_attention_items"`
	BlockedItems        int                        `json:"blocked_items"`
}

type releaseReadinessItemJSON struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type releaseRemediationPlanJSON struct {
	Status                     string                         `json:"status"`
	Mode                       string                         `json:"mode"`
	Scope                      string                         `json:"scope"`
	ProjectKey                 string                         `json:"project_key,omitempty"`
	ReadinessScope             string                         `json:"readiness_scope"`
	ClaimScope                 string                         `json:"claim_scope"`
	NotReal100                 bool                           `json:"not_real_100"`
	EvidenceOnly               bool                           `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                           `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                         `json:"release_candidate_decision"`
	Real100Status              string                         `json:"real_100_status"`
	Real100Blockers            []string                       `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown       `json:"real_100_breakdown"`
	Readiness                  releaseReadinessJSON           `json:"readiness"`
	Actions                    []releaseRemediationActionJSON `json:"actions"`
	Capabilities               []string                       `json:"capabilities"`
	ForbiddenActions           []string                       `json:"forbidden_actions"`
	GeneratedAt                string                         `json:"generated_at"`
}

type releaseRemediationActionJSON struct {
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

type releaseAcceptancePreviewJSON struct {
	Status                     string                          `json:"status"`
	Mode                       string                          `json:"mode"`
	Scope                      string                          `json:"scope"`
	ProjectKey                 string                          `json:"project_key,omitempty"`
	ReadinessScope             string                          `json:"readiness_scope"`
	ClaimScope                 string                          `json:"claim_scope"`
	NotReal100                 bool                            `json:"not_real_100"`
	EvidenceOnly               bool                            `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                            `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                          `json:"release_candidate_decision"`
	Real100Status              string                          `json:"real_100_status"`
	Real100Blockers            []string                        `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown        `json:"real_100_breakdown"`
	Remediation                releaseRemediationPlanJSON      `json:"remediation"`
	Decisions                  []releaseAcceptanceDecisionJSON `json:"decisions"`
	Capabilities               []string                        `json:"capabilities"`
	ForbiddenActions           []string                        `json:"forbidden_actions"`
	GeneratedAt                string                          `json:"generated_at"`
}

type releaseAcceptanceDecisionJSON struct {
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

type releaseAcceptanceGateJSON struct {
	Status                     string                          `json:"status"`
	Mode                       string                          `json:"mode"`
	Scope                      string                          `json:"scope"`
	ProjectKey                 string                          `json:"project_key,omitempty"`
	ReadinessScope             string                          `json:"readiness_scope"`
	ClaimScope                 string                          `json:"claim_scope"`
	NotReal100                 bool                            `json:"not_real_100"`
	EvidenceOnly               bool                            `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                            `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                          `json:"release_candidate_decision"`
	Real100Status              string                          `json:"real_100_status"`
	Real100Blockers            []string                        `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown        `json:"real_100_breakdown"`
	Preview                    releaseAcceptancePreviewJSON    `json:"preview"`
	Items                      []releaseAcceptanceGateItemJSON `json:"items"`
	Capabilities               []string                        `json:"capabilities"`
	ForbiddenActions           []string                        `json:"forbidden_actions"`
	GeneratedAt                string                          `json:"generated_at"`
}

type releaseAcceptanceGateItemJSON struct {
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

type releaseExceptionDoctorJSON struct {
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
	Gate                       releaseAcceptanceGateJSON         `json:"gate"`
	Checks                     []releaseExceptionDoctorCheckJSON `json:"checks"`
	Capabilities               []string                          `json:"capabilities"`
	ForbiddenActions           []string                          `json:"forbidden_actions"`
	GeneratedAt                string                            `json:"generated_at"`
}

type releaseExceptionDoctorCheckJSON struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type releaseExceptionRecordPreviewJSON struct {
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
	Doctor                     releaseExceptionDoctorJSON        `json:"doctor"`
	Drafts                     []releaseExceptionRecordDraftJSON `json:"drafts"`
	Capabilities               []string                          `json:"capabilities"`
	ForbiddenActions           []string                          `json:"forbidden_actions"`
	GeneratedAt                string                            `json:"generated_at"`
}

type releaseExceptionRecordDraftJSON struct {
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

type releaseExceptionSchemaPreviewJSON struct {
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
	RecordPreview              releaseExceptionRecordPreviewJSON   `json:"record_preview"`
	Tables                     []releaseExceptionSchemaTableJSON   `json:"tables"`
	ApplySteps                 []releaseExceptionMigrationStepJSON `json:"apply_steps"`
	RollbackSteps              []releaseExceptionMigrationStepJSON `json:"rollback_steps"`
	AuditActions               []string                            `json:"audit_actions"`
	Capabilities               []string                            `json:"capabilities"`
	ForbiddenActions           []string                            `json:"forbidden_actions"`
	GeneratedAt                string                              `json:"generated_at"`
}

type releaseExceptionSchemaTableJSON struct {
	Name        string                                 `json:"name"`
	Purpose     string                                 `json:"purpose"`
	Columns     []releaseExceptionSchemaColumnJSON     `json:"columns"`
	Indexes     []releaseExceptionSchemaIndexJSON      `json:"indexes"`
	ForeignKeys []releaseExceptionSchemaForeignKeyJSON `json:"foreign_keys"`
}

type releaseExceptionSchemaColumnJSON struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Purpose  string `json:"purpose"`
}

type releaseExceptionSchemaIndexJSON struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Purpose string   `json:"purpose"`
}

type releaseExceptionSchemaForeignKeyJSON struct {
	Column           string `json:"column"`
	ReferencesTable  string `json:"references_table"`
	ReferencesColumn string `json:"references_column"`
	OnDelete         string `json:"on_delete"`
}

type releaseExceptionMigrationStepJSON struct {
	Order       int    `json:"order"`
	Action      string `json:"action"`
	Description string `json:"description"`
	SQLPreview  string `json:"sql_preview"`
}

type releaseExceptionMigrationApprovalGateJSON struct {
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
	SchemaPreview              releaseExceptionSchemaPreviewJSON           `json:"schema_preview"`
	Items                      []releaseExceptionMigrationApprovalItemJSON `json:"items"`
	Capabilities               []string                                    `json:"capabilities"`
	ForbiddenActions           []string                                    `json:"forbidden_actions"`
	GeneratedAt                string                                      `json:"generated_at"`
}

type releaseExceptionMigrationApprovalItemJSON struct {
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

type releaseExceptionApplyPreviewJSON struct {
	Status                     string                                    `json:"status"`
	Mode                       string                                    `json:"mode"`
	Scope                      string                                    `json:"scope"`
	ProjectKey                 string                                    `json:"project_key,omitempty"`
	ReadinessScope             string                                    `json:"readiness_scope"`
	ClaimScope                 string                                    `json:"claim_scope"`
	NotReal100                 bool                                      `json:"not_real_100"`
	EvidenceOnly               bool                                      `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                                      `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                                    `json:"release_candidate_decision"`
	Real100Status              string                                    `json:"real_100_status"`
	Real100Blockers            []string                                  `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown                  `json:"real_100_breakdown"`
	MigrationGate              releaseExceptionMigrationApprovalGateJSON `json:"migration_gate"`
	Items                      []releaseExceptionApplyPreviewItemJSON    `json:"items"`
	ApplySteps                 []releaseExceptionApplyPreviewStepJSON    `json:"apply_steps"`
	RollbackSteps              []releaseExceptionApplyPreviewStepJSON    `json:"rollback_steps"`
	Capabilities               []string                                  `json:"capabilities"`
	ForbiddenActions           []string                                  `json:"forbidden_actions"`
	GeneratedAt                string                                    `json:"generated_at"`
}

type releaseExceptionApplyPreviewItemJSON struct {
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

type releaseExceptionApplyPreviewStepJSON struct {
	Order       int      `json:"order"`
	Action      string   `json:"action"`
	Description string   `json:"description"`
	BlockedBy   []string `json:"blocked_by"`
}

type releaseFinalGateJSON struct {
	Status                     string                           `json:"status"`
	Mode                       string                           `json:"mode"`
	Scope                      string                           `json:"scope"`
	ProjectKey                 string                           `json:"project_key,omitempty"`
	ReadinessScope             string                           `json:"readiness_scope"`
	ClaimScope                 string                           `json:"claim_scope"`
	NotReal100                 bool                             `json:"not_real_100"`
	EvidenceOnly               bool                             `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                             `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                           `json:"release_candidate_decision"`
	Real100Status              string                           `json:"real_100_status"`
	Real100Blockers            []string                         `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown         `json:"real_100_breakdown"`
	Readiness                  releaseReadinessJSON             `json:"readiness"`
	AcceptanceGate             releaseAcceptanceGateJSON        `json:"acceptance_gate"`
	ExceptionApply             releaseExceptionApplyPreviewJSON `json:"exception_apply"`
	Items                      []releaseFinalGateItemJSON       `json:"items"`
	Capabilities               []string                         `json:"capabilities"`
	ForbiddenActions           []string                         `json:"forbidden_actions"`
	GeneratedAt                string                           `json:"generated_at"`
}

type releaseFinalGateItemJSON struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type releaseEvidenceBundleJSON struct {
	Status                     string                          `json:"status"`
	Mode                       string                          `json:"mode"`
	Scope                      string                          `json:"scope"`
	ProjectKey                 string                          `json:"project_key,omitempty"`
	ReadinessScope             string                          `json:"readiness_scope"`
	ClaimScope                 string                          `json:"claim_scope"`
	NotReal100                 bool                            `json:"not_real_100"`
	EvidenceOnly               bool                            `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                            `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                          `json:"release_candidate_decision"`
	Real100Status              string                          `json:"real_100_status"`
	Real100Blockers            []string                        `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown        `json:"real_100_breakdown"`
	BundleHash                 string                          `json:"bundle_hash"`
	FinalGate                  releaseFinalGateJSON            `json:"final_gate"`
	Backup                     backupManifestJSON              `json:"backup"`
	AuditCoverage              auditCoverageJSON               `json:"audit_coverage"`
	Items                      []releaseEvidenceBundleItemJSON `json:"items"`
	Capabilities               []string                        `json:"capabilities"`
	ForbiddenActions           []string                        `json:"forbidden_actions"`
	GeneratedAt                string                          `json:"generated_at"`
}

type releaseEvidenceBundleItemJSON struct {
	Key         string         `json:"key"`
	Category    string         `json:"category"`
	Status      string         `json:"status"`
	Source      string         `json:"source"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

type releasePackagePreviewJSON struct {
	Status                     string                          `json:"status"`
	Mode                       string                          `json:"mode"`
	Scope                      string                          `json:"scope"`
	ProjectKey                 string                          `json:"project_key,omitempty"`
	ReadinessScope             string                          `json:"readiness_scope"`
	ClaimScope                 string                          `json:"claim_scope"`
	NotReal100                 bool                            `json:"not_real_100"`
	EvidenceOnly               bool                            `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                            `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                          `json:"release_candidate_decision"`
	Real100Status              string                          `json:"real_100_status"`
	Real100Blockers            []string                        `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown        `json:"real_100_breakdown"`
	EvidenceBundle             releaseEvidenceBundleJSON       `json:"evidence_bundle"`
	PackageName                string                          `json:"package_name"`
	Items                      []releasePackagePreviewItemJSON `json:"items"`
	Capabilities               []string                        `json:"capabilities"`
	ForbiddenActions           []string                        `json:"forbidden_actions"`
	GeneratedAt                string                          `json:"generated_at"`
}

type releasePackagePreviewItemJSON struct {
	Key         string         `json:"key"`
	Category    string         `json:"category"`
	Status      string         `json:"status"`
	PackagePath string         `json:"package_path"`
	Source      string         `json:"source"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}

type releaseDistributionPreviewJSON struct {
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
	PackagePreview             releasePackagePreviewJSON            `json:"package_preview"`
	Items                      []releaseDistributionPreviewItemJSON `json:"items"`
	Capabilities               []string                             `json:"capabilities"`
	ForbiddenActions           []string                             `json:"forbidden_actions"`
	GeneratedAt                string                               `json:"generated_at"`
}

type releaseDistributionPreviewItemJSON struct {
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

type releasePublishGateJSON struct {
	Status                     string                         `json:"status"`
	Mode                       string                         `json:"mode"`
	Scope                      string                         `json:"scope"`
	ProjectKey                 string                         `json:"project_key,omitempty"`
	ReadinessScope             string                         `json:"readiness_scope"`
	ClaimScope                 string                         `json:"claim_scope"`
	NotReal100                 bool                           `json:"not_real_100"`
	EvidenceOnly               bool                           `json:"evidence_only"`
	StatusAloneIsNotCompletion bool                           `json:"status_alone_is_not_completion"`
	ReleaseCandidateDecision   string                         `json:"release_candidate_decision"`
	Real100Status              string                         `json:"real_100_status"`
	Real100Blockers            []string                       `json:"real_100_blockers"`
	Real100Breakdown           project.Real100Breakdown       `json:"real_100_breakdown"`
	DistributionPreview        releaseDistributionPreviewJSON `json:"distribution_preview"`
	Items                      []releasePublishGateItemJSON   `json:"items"`
	Capabilities               []string                       `json:"capabilities"`
	ForbiddenActions           []string                       `json:"forbidden_actions"`
	GeneratedAt                string                         `json:"generated_at"`
}

type releasePublishGateItemJSON struct {
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

type releasePublishApprovalPreviewJSON struct {
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
	PublishGate                releasePublishGateJSON                  `json:"publish_gate"`
	Items                      []releasePublishApprovalPreviewItemJSON `json:"items"`
	Capabilities               []string                                `json:"capabilities"`
	ForbiddenActions           []string                                `json:"forbidden_actions"`
	GeneratedAt                string                                  `json:"generated_at"`
}

type releasePublishApprovalPreviewItemJSON struct {
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

type releaseRolloutPlanPreviewJSON struct {
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
	PublishApprovalPreview     releasePublishApprovalPreviewJSON   `json:"publish_approval_preview"`
	Items                      []releaseRolloutPlanPreviewItemJSON `json:"items"`
	RolloutSteps               []releaseRolloutPlanPreviewStepJSON `json:"rollout_steps"`
	VerificationCheckpoints    []releaseRolloutPlanPreviewStepJSON `json:"verification_checkpoints"`
	RollbackSteps              []releaseRolloutPlanPreviewStepJSON `json:"rollback_steps"`
	Capabilities               []string                            `json:"capabilities"`
	ForbiddenActions           []string                            `json:"forbidden_actions"`
	GeneratedAt                string                              `json:"generated_at"`
}

type releaseRolloutPlanPreviewItemJSON struct {
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

type releaseRolloutPlanPreviewStepJSON struct {
	Order       int      `json:"order"`
	Stage       string   `json:"stage"`
	Action      string   `json:"action"`
	Description string   `json:"description"`
	BlockedBy   []string `json:"blocked_by"`
}

type auditCoverageJSON struct {
	Status              string                         `json:"status"`
	Mode                string                         `json:"mode"`
	Scope               string                         `json:"scope"`
	ProjectID           int64                          `json:"project_id,omitempty"`
	ProjectKey          string                         `json:"project_key,omitempty"`
	TotalAuditEvents    int64                          `json:"total_audit_events"`
	CoveredRequirements int                            `json:"covered_requirements"`
	GapRequirements     int                            `json:"gap_requirements"`
	Requirements        []auditCoverageRequirementJSON `json:"requirements"`
	GeneratedAt         string                         `json:"generated_at"`
}

type auditCoverageRequirementJSON struct {
	Key             string                            `json:"key"`
	Category        string                            `json:"category"`
	Description     string                            `json:"description"`
	Status          string                            `json:"status"`
	EvidenceCount   int64                             `json:"evidence_count"`
	RequiredActions []auditCoverageActionEvidenceJSON `json:"required_actions"`
	MissingActions  []string                          `json:"missing_actions"`
	LastAuditAt     string                            `json:"last_audit_at,omitempty"`
}

type auditCoverageActionEvidenceJSON struct {
	Action      string `json:"action"`
	Decision    string `json:"decision,omitempty"`
	Count       int64  `json:"count"`
	Status      string `json:"status"`
	LastAuditAt string `json:"last_audit_at,omitempty"`
}

type permissionPolicyDoctorJSON struct {
	Status      string                      `json:"status"`
	Mode        string                      `json:"mode"`
	Project     projectRecordJSON           `json:"project"`
	Checks      []permissionPolicyCheckJSON `json:"checks"`
	GeneratedAt string                      `json:"generated_at"`
}

type permissionPolicyCheckJSON struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type artifactIntegrityJSON struct {
	Status           string                       `json:"status"`
	Mode             string                       `json:"mode"`
	Project          projectRecordJSON            `json:"project"`
	CheckedArtifacts int                          `json:"checked_artifacts"`
	PassedArtifacts  int                          `json:"passed_artifacts"`
	WarnArtifacts    int                          `json:"warn_artifacts"`
	FailedArtifacts  int                          `json:"failed_artifacts"`
	SkippedArtifacts int                          `json:"skipped_artifacts"`
	Checks           []artifactIntegrityCheckJSON `json:"checks"`
	GeneratedAt      string                       `json:"generated_at"`
}

type artifactArchivePreviewJSON struct {
	Project                 projectRecordJSON                 `json:"project"`
	Status                  string                            `json:"status"`
	Mode                    string                            `json:"mode"`
	Summary                 artifactArchivePreviewSummaryJSON `json:"summary"`
	Items                   []artifactArchivePreviewItemJSON  `json:"items"`
	EventID                 int64                             `json:"event_id,omitempty"`
	AuditEventID            int64                             `json:"audit_event_id,omitempty"`
	IdempotencyKey          string                            `json:"idempotency_key"`
	Created                 bool                              `json:"created"`
	GeneratedAt             string                            `json:"generated_at"`
	ProjectWriteAttempted   bool                              `json:"project_write_attempted"`
	StorageWriteAttempted   bool                              `json:"storage_write_attempted"`
	ArtifactDeleteAttempted bool                              `json:"artifact_delete_attempted"`
}

type artifactArchivePreviewSummaryJSON struct {
	TotalArtifacts    int `json:"total_artifacts"`
	ArchiveCandidates int `json:"archive_candidates"`
	RetainedArtifacts int `json:"retained_artifacts"`
	ExternalRefs      int `json:"external_refs"`
	NeedsPolicy       int `json:"needs_policy"`
}

type artifactArchivePreviewItemJSON struct {
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

type artifactIntegrityCheckJSON struct {
	Artifact artifactJSON   `json:"artifact"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type conformanceJSON struct {
	Status      string                 `json:"status"`
	Mode        string                 `json:"mode"`
	Project     projectRecordJSON      `json:"project"`
	ProfileID   string                 `json:"profile_id"`
	Adapter     string                 `json:"adapter"`
	ProfileHash string                 `json:"profile_hash"`
	StageCount  int                    `json:"stage_count"`
	GateCount   int                    `json:"gate_count"`
	Checks      []conformanceCheckJSON `json:"checks"`
	GeneratedAt string                 `json:"generated_at"`
}

type conformanceCheckJSON struct {
	Key      string         `json:"key"`
	Category string         `json:"category"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type projectConfigJSON struct {
	ProtocolVersion int            `json:"protocol_version"`
	ConfigPath      string         `json:"config_path"`
	ConfigHash      string         `json:"config_hash"`
	Ownership       map[string]any `json:"ownership"`
	StatusExport    map[string]any `json:"status_export"`
	Migration       map[string]any `json:"migration"`
	LoadedAt        string         `json:"loaded_at"`
}

type projectRecordJSON struct {
	Key             string `json:"key"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Adapter         string `json:"adapter"`
	WorkflowProfile string `json:"workflow_profile"`
	DefaultBranch   string `json:"default_branch"`
	Root            string `json:"root"`
}

type projectInventoryJSON struct {
	Versions        int64 `json:"versions"`
	Residuals       int64 `json:"residuals"`
	Artifacts       int64 `json:"artifacts"`
	ImportSnapshots int64 `json:"import_snapshots"`
	MirrorExports   int64 `json:"mirror_exports"`
}

type projectImportJSON struct {
	SourceHash             string         `json:"source_hash"`
	CreatedAt              string         `json:"created_at"`
	Summary                map[string]any `json:"summary"`
	HasPrevious            bool           `json:"has_previous"`
	PreviousSourceHash     string         `json:"previous_source_hash,omitempty"`
	PreviousCreatedAt      string         `json:"previous_created_at,omitempty"`
	HistoryReadyForDiff    bool           `json:"history_ready_for_diff"`
	SourceHashChangedSince bool           `json:"source_hash_changed_since_previous"`
}

type projectImportRunJSON struct {
	Project        string `json:"project"`
	Versions       int    `json:"versions"`
	Residuals      int    `json:"residuals"`
	Artifacts      int    `json:"artifacts"`
	ActiveTasks    int    `json:"active_tasks"`
	V1Done         int    `json:"v1_done"`
	V1Total        int    `json:"v1_total"`
	StatusSnapshot string `json:"status_snapshot"`
	RunID          int64  `json:"run_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Created        bool   `json:"created"`
}

type projectDoctorJSON struct {
	Status              string         `json:"status"`
	DriftStatus         string         `json:"drift_status"`
	ConfigDriftStatus   string         `json:"config_drift_status"`
	StageCoverageStatus string         `json:"stage_coverage_status"`
	NativeDoctorStatus  string         `json:"native_doctor_status"`
	Severity            string         `json:"severity"`
	CreatedAt           string         `json:"created_at"`
	Metadata            map[string]any `json:"metadata"`
}

type projectReadinessJSON struct {
	Project projectRecordJSON          `json:"project"`
	Status  string                     `json:"status"`
	Items   []projectReadinessItemJSON `json:"items"`
	Summary projectSummaryJSON         `json:"summary"`
}

type generatedWriteReadinessJSON struct {
	Project                       projectRecordJSON          `json:"project"`
	Status                        string                     `json:"status"`
	Mode                          string                     `json:"mode"`
	Items                         []projectReadinessItemJSON `json:"items"`
	RequiredCapabilities          []string                   `json:"required_capabilities"`
	AllowedGeneratedPrefixes      []string                   `json:"allowed_generated_prefixes"`
	RequiredWritePaths            []string                   `json:"required_write_paths"`
	ConfiguredWritePaths          []string                   `json:"configured_write_paths"`
	ConfiguredForbiddenPaths      []string                   `json:"configured_forbidden_paths"`
	Blockers                      []string                   `json:"blockers"`
	ReviewBlockers                []string                   `json:"review_blockers"`
	ForbiddenActions              []string                   `json:"forbidden_actions"`
	ReadyForReview                bool                       `json:"ready_for_review"`
	ApplyOpen                     bool                       `json:"apply_open"`
	RealAreaMatrixWriteOpened     bool                       `json:"real_areamatrix_write_opened"`
	GeneratedOnly                 bool                       `json:"generated_only"`
	ProjectConfigRead             bool                       `json:"project_config_read"`
	ProjectReadAttempted          bool                       `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                       `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                       `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                       `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                       `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                       `json:"engine_call_attempted"`
	CommandsRun                   bool                       `json:"commands_run"`
	SecretsResolved               bool                       `json:"secrets_resolved"`
	NetworkUsed                   bool                       `json:"network_used"`
	TaskClaimed                   bool                       `json:"task_claimed"`
	WorkerStarted                 bool                       `json:"worker_started"`
	LeaseCreated                  bool                       `json:"lease_created"`
	AttemptCreated                bool                       `json:"attempt_created"`
	ArtifactCreated               bool                       `json:"artifact_created"`
	GeneratedAt                   string                     `json:"generated_at"`
}

type generatedWriteApplyBetaGateJSON struct {
	Project                       projectRecordJSON                     `json:"project"`
	Status                        string                                `json:"status"`
	Mode                          string                                `json:"mode"`
	Readiness                     generatedWriteReadinessJSON           `json:"readiness"`
	Items                         []generatedWriteApplyBetaGateItemJSON `json:"items"`
	RequiredCapabilities          []string                              `json:"required_capabilities"`
	AllowedGeneratedPrefixes      []string                              `json:"allowed_generated_prefixes"`
	RequiredEvidence              []string                              `json:"required_evidence"`
	ForbiddenActions              []string                              `json:"forbidden_actions"`
	ApprovalRequired              bool                                  `json:"approval_required"`
	ApprovalStatus                string                                `json:"approval_status"`
	ApplyOpen                     bool                                  `json:"apply_open"`
	RealAreaMatrixWriteOpened     bool                                  `json:"real_areamatrix_write_opened"`
	GeneratedOnly                 bool                                  `json:"generated_only"`
	ProjectReadAttempted          bool                                  `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                                  `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                                  `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                                  `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                                  `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                                  `json:"engine_call_attempted"`
	CommandsRun                   bool                                  `json:"commands_run"`
	SecretsResolved               bool                                  `json:"secrets_resolved"`
	NetworkUsed                   bool                                  `json:"network_used"`
	TaskClaimed                   bool                                  `json:"task_claimed"`
	WorkerStarted                 bool                                  `json:"worker_started"`
	LeaseCreated                  bool                                  `json:"lease_created"`
	AttemptCreated                bool                                  `json:"attempt_created"`
	ArtifactCreated               bool                                  `json:"artifact_created"`
	GeneratedAt                   string                                `json:"generated_at"`
}

type generatedWriteApplyBetaGateItemJSON struct {
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

type projectReadinessItemJSON struct {
	Key      string         `json:"key"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type projectImportDiffJSON struct {
	Project       projectRecordJSON       `json:"project"`
	Status        string                  `json:"status"`
	HasPrevious   bool                    `json:"has_previous"`
	SourceChanged bool                    `json:"source_changed"`
	Latest        projectDiffSnapshotJSON `json:"latest"`
	Previous      projectDiffSnapshotJSON `json:"previous,omitempty"`
	Changes       []projectDiffChangeJSON `json:"changes"`
}

type projectDiffSnapshotJSON struct {
	SourceHash string `json:"source_hash"`
	CreatedAt  string `json:"created_at"`
}

type projectDiffChangeJSON struct {
	Key      string `json:"key"`
	Status   string `json:"status"`
	Previous string `json:"previous"`
	Latest   string `json:"latest"`
}

type projectVerificationBundleJSON struct {
	Project    projectRecordJSON     `json:"project"`
	Status     string                `json:"status"`
	PhaseGate  projectPhaseGateJSON  `json:"phase_gate"`
	Summary    projectSummaryJSON    `json:"summary"`
	Readiness  projectReadinessJSON  `json:"readiness"`
	ImportDiff projectImportDiffJSON `json:"import_diff"`
	Events     []projectEventJSON    `json:"events"`
}

type projectCutoverReadinessJSON struct {
	Project         projectRecordJSON             `json:"project"`
	WorkflowVersion workflowVersionJSON           `json:"workflow_version"`
	Status          string                        `json:"status"`
	PhaseGate       projectPhaseGateJSON          `json:"phase_gate"`
	Items           []projectReadinessItemJSON    `json:"items"`
	Verification    projectVerificationBundleJSON `json:"verification"`
	Compatibility   compatibilityContractJSON     `json:"compatibility"`
	Gates           []gateResultJSON              `json:"gates"`
}

type projectCutoverApplyJSON struct {
	Project                 projectRecordJSON   `json:"project"`
	WorkflowVersion         workflowVersionJSON `json:"workflow_version"`
	Status                  string              `json:"status"`
	Decision                string              `json:"decision"`
	Message                 string              `json:"message"`
	Blockers                []string            `json:"blockers"`
	Warnings                []string            `json:"warnings"`
	EventID                 int64               `json:"event_id,omitempty"`
	AuditEventID            int64               `json:"audit_event_id,omitempty"`
	IdempotencyKey          string              `json:"idempotency_key"`
	Created                 bool                `json:"created"`
	ProjectWriteAttempted   bool                `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                `json:"execution_write_attempted"`
	CutoverReadinessGateID  int64               `json:"cutover_readiness_gate_id,omitempty"`
}

type statusProjectionApplyJSON struct {
	Project                   projectRecordJSON `json:"project"`
	Status                    string            `json:"status"`
	Decision                  string            `json:"decision"`
	Message                   string            `json:"message"`
	Blockers                  []string          `json:"blockers"`
	EventID                   int64             `json:"event_id,omitempty"`
	AuditEventID              int64             `json:"audit_event_id,omitempty"`
	SnapshotID                int64             `json:"snapshot_id,omitempty"`
	StatusProjectionID        int64             `json:"status_projection_id,omitempty"`
	TargetKind                string            `json:"target_kind"`
	TargetURI                 string            `json:"target_uri"`
	WrittenTarget             string            `json:"written_target,omitempty"`
	WriteHash                 string            `json:"write_hash,omitempty"`
	WriteSize                 int64             `json:"write_size,omitempty"`
	PreimageCaptured          bool              `json:"preimage_captured"`
	PreimageExists            bool              `json:"preimage_exists"`
	PreimageSHA256            string            `json:"preimage_sha256,omitempty"`
	PreimageSize              int64             `json:"preimage_size,omitempty"`
	PostWriteVerified         bool              `json:"post_write_verified"`
	PostWriteSHA256           string            `json:"post_write_sha256,omitempty"`
	PostWriteSize             int64             `json:"post_write_size,omitempty"`
	ProtectedPathsVerified    bool              `json:"protected_paths_verified"`
	ProtectedPathBeforeHash   string            `json:"protected_path_before_hash,omitempty"`
	ProtectedPathAfterHash    string            `json:"protected_path_after_hash,omitempty"`
	ExpectedProtectedPathHash string            `json:"expected_protected_path_hash,omitempty"`
	RootContained             bool              `json:"root_contained"`
	StableProjectionValid     bool              `json:"stable_projection_validated"`
	AtomicReplaceUsed         bool              `json:"atomic_replace_used"`
	RollbackCompensation      bool              `json:"rollback_compensation_enabled"`
	SourceHash                string            `json:"source_hash,omitempty"`
	SummaryState              string            `json:"summary_state"`
	ApplyGateStatus           string            `json:"apply_gate_status"`
	ApplyGateDecision         string            `json:"apply_gate_decision"`
	ApplyGateApprovalStatus   string            `json:"apply_gate_approval_status"`
	ApplyCommandEligible      bool              `json:"apply_command_eligible"`
	IdempotencyKey            string            `json:"idempotency_key"`
	Created                   bool              `json:"created"`
	GeneratedAt               string            `json:"generated_at"`
	ProjectWriteAttempted     bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted   bool              `json:"execution_write_attempted"`
	EngineCallAttempted       bool              `json:"engine_call_attempted"`
}

type statusProjectionAuthorizationPreviewJSON struct {
	Project                                       projectRecordJSON                           `json:"project"`
	Status                                        string                                      `json:"status"`
	Mode                                          string                                      `json:"mode"`
	ClaimScope                                    string                                      `json:"claim_scope"`
	NotReal100                                    bool                                        `json:"not_real_100"`
	Decision                                      string                                      `json:"decision"`
	Message                                       string                                      `json:"message"`
	TargetKind                                    string                                      `json:"target_kind"`
	TargetURI                                     string                                      `json:"target_uri"`
	TargetPath                                    string                                      `json:"target_path"`
	SchemaURI                                     string                                      `json:"schema_uri"`
	ValidatorPreflight                            string                                      `json:"validator_preflight"`
	ProtectedPathFingerprintSHA256                string                                      `json:"protected_path_fingerprint_sha256,omitempty"`
	SourceHash                                    string                                      `json:"source_hash,omitempty"`
	SummaryState                                  string                                      `json:"summary_state"`
	RequiredAuthorizationPhrase                   string                                      `json:"required_authorization_phrase,omitempty"`
	Permission                                    statusProjectionAuthorizationPermissionJSON `json:"permission"`
	Preimage                                      statusProjectionPreimageJSON                `json:"preimage"`
	WriteSet                                      []statusProjectionWriteSetEntryJSON         `json:"write_set"`
	RequiredPreflight                             []string                                    `json:"required_preflight"`
	RequiredPacketFields                          []string                                    `json:"required_packet_fields"`
	RequiredCapabilities                          []string                                    `json:"required_capabilities"`
	ProtectedPaths                                []string                                    `json:"protected_paths"`
	RollbackPlan                                  []string                                    `json:"rollback_plan"`
	BlockedBy                                     []string                                    `json:"blocked_by"`
	Warnings                                      []string                                    `json:"warnings"`
	ForbiddenActions                              []string                                    `json:"forbidden_actions"`
	SafetyFacts                                   map[string]bool                             `json:"safety_facts"`
	ApplyOpen                                     bool                                        `json:"apply_open"`
	ApprovalRequired                              bool                                        `json:"approval_required"`
	ApprovalStatus                                string                                      `json:"approval_status"`
	WouldCreateCommandRequestAfterApproval        bool                                        `json:"would_create_command_request_after_approval"`
	WouldCreateProjectStatusSnapshotAfterApproval bool                                        `json:"would_create_project_status_snapshot_after_approval"`
	WouldCreateStatusProjectionAfterApproval      bool                                        `json:"would_create_status_projection_after_approval"`
	WouldCreateEventAfterApproval                 bool                                        `json:"would_create_event_after_approval"`
	WouldCreateAuditEventAfterApproval            bool                                        `json:"would_create_audit_event_after_approval"`
	WouldWriteProjectFileAfterApproval            bool                                        `json:"would_write_project_file_after_approval"`
	WouldWriteExecutionAfterApproval              bool                                        `json:"would_write_execution_after_approval"`
	WouldRunEngineAfterApproval                   bool                                        `json:"would_run_engine_after_approval"`
	ProjectWriteAttempted                         bool                                        `json:"project_write_attempted"`
	ExecutionWriteAttempted                       bool                                        `json:"execution_write_attempted"`
	EngineCallAttempted                           bool                                        `json:"engine_call_attempted"`
	GeneratedAt                                   string                                      `json:"generated_at"`
}

type statusProjectionAuthorizationPermissionJSON struct {
	Capability        string `json:"capability"`
	ResourceType      string `json:"resource_type"`
	TargetURI         string `json:"target_uri"`
	CapabilityAllowed bool   `json:"capability_allowed"`
	PathAllowed       bool   `json:"path_allowed"`
	Allowed           bool   `json:"allowed"`
	Reason            string `json:"reason"`
}

type statusProjectionPreimageJSON struct {
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

type statusProjectionWriteSetEntryJSON struct {
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

type statusProjectionApplyGateJSON struct {
	Project                        projectRecordJSON                        `json:"project"`
	Status                         string                                   `json:"status"`
	Mode                           string                                   `json:"mode"`
	ClaimScope                     string                                   `json:"claim_scope"`
	NotReal100                     bool                                     `json:"not_real_100"`
	Decision                       string                                   `json:"decision"`
	Message                        string                                   `json:"message"`
	TargetURI                      string                                   `json:"target_uri"`
	TargetPath                     string                                   `json:"target_path"`
	Authorization                  statusProjectionAuthorizationPreviewJSON `json:"authorization"`
	Items                          []statusProjectionApplyGateItemJSON      `json:"items"`
	RequiredPacketFields           []string                                 `json:"required_packet_fields"`
	RequiredCapabilities           []string                                 `json:"required_capabilities"`
	RequiredAuthorizationPhrase    string                                   `json:"required_authorization_phrase,omitempty"`
	ProtectedPaths                 []string                                 `json:"protected_paths"`
	ForbiddenActions               []string                                 `json:"forbidden_actions"`
	SafetyFacts                    map[string]bool                          `json:"safety_facts"`
	ApplyCommandEligible           bool                                     `json:"apply_command_eligible"`
	ApplyCommandEligibleIsNotApply bool                                     `json:"apply_command_eligible_is_not_apply"`
	RequiresSeparateApplyCommand   bool                                     `json:"requires_separate_apply_command"`
	ApprovalRequired               bool                                     `json:"approval_required"`
	ApprovalStatus                 string                                   `json:"approval_status"`
	ProjectWriteAttempted          bool                                     `json:"project_write_attempted"`
	ExecutionWriteAttempted        bool                                     `json:"execution_write_attempted"`
	EngineCallAttempted            bool                                     `json:"engine_call_attempted"`
	CommandRequestCreated          bool                                     `json:"command_request_created"`
	StatusProjectionWritten        bool                                     `json:"status_projection_written"`
	GeneratedAt                    string                                   `json:"generated_at"`
}

type statusProjectionApplyPacketPreviewJSON struct {
	Project                                      projectRecordJSON                        `json:"project"`
	Status                                       string                                   `json:"status"`
	Mode                                         string                                   `json:"mode"`
	ClaimScope                                   string                                   `json:"claim_scope"`
	NotReal100                                   bool                                     `json:"not_real_100"`
	Decision                                     string                                   `json:"decision"`
	Message                                      string                                   `json:"message"`
	Blockers                                     []string                                 `json:"blockers"`
	RequiredAuthorizationPhrase                  string                                   `json:"required_authorization_phrase,omitempty"`
	Authorization                                statusProjectionAuthorizationPreviewJSON `json:"authorization"`
	Gate                                         statusProjectionApplyGateJSON            `json:"gate"`
	Packet                                       statusProjectionApplyPacketJSON          `json:"packet"`
	ApplyCommand                                 []string                                 `json:"apply_command"`
	APIRequest                                   statusProjectionApplyAPIRequestJSON      `json:"api_request"`
	RequiredHumanReview                          []string                                 `json:"required_human_review"`
	ForbiddenActions                             []string                                 `json:"forbidden_actions"`
	SafetyFacts                                  map[string]bool                          `json:"safety_facts"`
	WouldCreateCommandRequestAfterApplyCommand   bool                                     `json:"would_create_command_request_after_apply_command"`
	WouldCreateStatusProjectionAfterApplyCommand bool                                     `json:"would_create_status_projection_after_apply_command"`
	WouldWriteProjectFileAfterApplyCommand       bool                                     `json:"would_write_project_file_after_apply_command"`
	ApplyCommandEligibleIsNotApply               bool                                     `json:"apply_command_eligible_is_not_apply"`
	RequiresSeparateApplyCommand                 bool                                     `json:"requires_separate_apply_command"`
	ProjectWriteAttempted                        bool                                     `json:"project_write_attempted"`
	ExecutionWriteAttempted                      bool                                     `json:"execution_write_attempted"`
	EngineCallAttempted                          bool                                     `json:"engine_call_attempted"`
	CommandRequestCreated                        bool                                     `json:"command_request_created"`
	StatusProjectionWritten                      bool                                     `json:"status_projection_written"`
	GeneratedAt                                  string                                   `json:"generated_at"`
}

type statusProjectionApplyPacketJSON struct {
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

type statusProjectionApplyAPIRequestJSON statusProjectionApplyPacketJSON

type statusProjectionApplyGateItemJSON struct {
	Key              string   `json:"key"`
	Category         string   `json:"category"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Expected         string   `json:"expected"`
	Actual           string   `json:"actual"`
	RequiredEvidence []string `json:"required_evidence"`
	BlockedBy        []string `json:"blocked_by"`
}

type statusProjectionJSON struct {
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

type statusProjectionsJSON struct {
	Project     projectRecordJSON      `json:"project"`
	Projections []statusProjectionJSON `json:"projections"`
}

type compatibilityContractJSON struct {
	Project  projectRecordJSON          `json:"project"`
	Status   string                     `json:"status"`
	Commands []compatibilityCommandJSON `json:"commands"`
}

type shimPreviewJSON struct {
	Project              projectRecordJSON         `json:"project"`
	Status               string                    `json:"status"`
	Mode                 string                    `json:"mode"`
	Contract             compatibilityContractJSON `json:"contract"`
	PlannedFiles         []shimFilePlanJSON        `json:"planned_files"`
	CommandMappings      []shimCommandMappingJSON  `json:"command_mappings"`
	DiscoveryOrder       []string                  `json:"discovery_order"`
	ForbiddenPaths       []string                  `json:"forbidden_paths"`
	ForbiddenCommands    []string                  `json:"forbidden_commands"`
	VerificationCommands []string                  `json:"verification_commands"`
	RollbackSteps        []string                  `json:"rollback_steps"`
	Notes                []string                  `json:"notes"`
}

type shimFilePlanJSON struct {
	Path     string `json:"path"`
	Action   string `json:"action"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
	Boundary string `json:"boundary"`
}

type shimCommandMappingJSON struct {
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

type shimReadinessJSON struct {
	Project projectRecordJSON       `json:"project"`
	Status  string                  `json:"status"`
	Preview shimPreviewJSON         `json:"preview"`
	Items   []shimReadinessItemJSON `json:"items"`
}

type shimAuthorizationPacketJSON struct {
	Project              projectRecordJSON  `json:"project"`
	Status               string             `json:"status"`
	Mode                 string             `json:"mode"`
	Intent               string             `json:"intent"`
	ReadinessStatus      string             `json:"readiness_status"`
	AllowedFiles         []shimFilePlanJSON `json:"allowed_files"`
	ForbiddenPaths       []string           `json:"forbidden_paths"`
	ForbiddenActions     []string           `json:"forbidden_actions"`
	RequiredPreflight    []string           `json:"required_preflight"`
	PostEditVerification []string           `json:"post_edit_verification"`
	RollbackScope        []string           `json:"rollback_scope"`
	SafetyFacts          map[string]bool    `json:"safety_facts"`
	NextRequiredApproval string             `json:"next_required_approval"`
}

type shimApplyPacketPreviewJSON struct {
	Project                                        projectRecordJSON           `json:"project"`
	Status                                         string                      `json:"status"`
	Mode                                           string                      `json:"mode"`
	Decision                                       string                      `json:"decision"`
	Message                                        string                      `json:"message"`
	Authorization                                  shimAuthorizationPacketJSON `json:"authorization"`
	Gate                                           shimApplyGateJSON           `json:"gate"`
	Packet                                         shimApplyPacketJSON         `json:"packet"`
	ApplyGateCommand                               []string                    `json:"apply_gate_command"`
	FutureApplyCommand                             []string                    `json:"future_apply_command"`
	RequiredHumanReview                            []string                    `json:"required_human_review"`
	ForbiddenActions                               []string                    `json:"forbidden_actions"`
	SafetyFacts                                    map[string]bool             `json:"safety_facts"`
	WouldCreateCommandRequestAfterApplyCommand     bool                        `json:"would_create_command_request_after_apply_command"`
	WouldWriteAreaMatrixShimFilesAfterApplyCommand bool                        `json:"would_write_area_matrix_shim_files_after_apply_command"`
	WouldWriteStatusProjectionAfterApplyCommand    bool                        `json:"would_write_status_projection_after_apply_command"`
	CommandRequestCreated                          bool                        `json:"command_request_created"`
	ProjectWriteAttempted                          bool                        `json:"project_write_attempted"`
	ExecutionWriteAttempted                        bool                        `json:"execution_write_attempted"`
	EngineCallAttempted                            bool                        `json:"engine_call_attempted"`
	TaskLoopRunForwarded                           bool                        `json:"task_loop_run_forwarded"`
	StatusProjectionWritten                        bool                        `json:"status_projection_written"`
	AreaMatrixFilesModified                        bool                        `json:"area_matrix_files_modified"`
	GeneratedAt                                    string                      `json:"generated_at"`
}

type shimApplyPacketJSON struct {
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

type shimApplyGateJSON struct {
	Project                 projectRecordJSON       `json:"project"`
	Status                  string                  `json:"status"`
	Mode                    string                  `json:"mode"`
	Decision                string                  `json:"decision"`
	Message                 string                  `json:"message"`
	Items                   []shimApplyGateItemJSON `json:"items"`
	RequiredPacketFields    []string                `json:"required_packet_fields"`
	RequiredCapabilities    []string                `json:"required_capabilities"`
	AllowedFiles            []string                `json:"allowed_files"`
	ForbiddenPaths          []string                `json:"forbidden_paths"`
	ForbiddenActions        []string                `json:"forbidden_actions"`
	RequiredPreflight       []string                `json:"required_preflight"`
	PostEditVerification    []string                `json:"post_edit_verification"`
	RollbackScope           []string                `json:"rollback_scope"`
	RequiredProofFacts      []string                `json:"required_proof_facts"`
	SafetyFacts             map[string]bool         `json:"safety_facts"`
	ApprovalRequired        bool                    `json:"approval_required"`
	ApprovalStatus          string                  `json:"approval_status"`
	ApplyCommandEligible    bool                    `json:"apply_command_eligible"`
	ApplyOpen               bool                    `json:"apply_open"`
	CommandRequestCreated   bool                    `json:"command_request_created"`
	ProjectWriteAttempted   bool                    `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                    `json:"execution_write_attempted"`
	EngineCallAttempted     bool                    `json:"engine_call_attempted"`
	TaskLoopRunForwarded    bool                    `json:"task_loop_run_forwarded"`
	StatusProjectionWritten bool                    `json:"status_projection_written"`
	AreaMatrixFilesModified bool                    `json:"area_matrix_files_modified"`
	GeneratedAt             string                  `json:"generated_at"`
}

type shimApplyCommandJSON struct {
	Project                 projectRecordJSON `json:"project"`
	Status                  string            `json:"status"`
	Mode                    string            `json:"mode"`
	Decision                string            `json:"decision"`
	Message                 string            `json:"message"`
	Gate                    shimApplyGateJSON `json:"gate"`
	Blockers                []string          `json:"blockers"`
	EventID                 int64             `json:"event_id,omitempty"`
	AuditEventID            int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey          string            `json:"idempotency_key"`
	Created                 bool              `json:"created"`
	RequiredPreflight       []string          `json:"required_preflight"`
	ForbiddenActions        []string          `json:"forbidden_actions"`
	SafetyFacts             map[string]bool   `json:"safety_facts"`
	ApplyOpen               bool              `json:"apply_open"`
	AreaFlowCommandCreated  bool              `json:"area_flow_command_created"`
	CommandRequestCreated   bool              `json:"command_request_created"`
	ProjectWriteAttempted   bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted bool              `json:"execution_write_attempted"`
	EngineCallAttempted     bool              `json:"engine_call_attempted"`
	TaskLoopRunForwarded    bool              `json:"task_loop_run_forwarded"`
	StatusProjectionWritten bool              `json:"status_projection_written"`
	AreaMatrixFilesModified bool              `json:"area_matrix_files_modified"`
	GeneratedAt             string            `json:"generated_at"`
}

type shimApplyGateItemJSON struct {
	Key              string   `json:"key"`
	Category         string   `json:"category"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Expected         string   `json:"expected"`
	Actual           string   `json:"actual"`
	RequiredEvidence []string `json:"required_evidence"`
	BlockedBy        []string `json:"blocked_by"`
}

type shimReadinessEvidenceJSON struct {
	Project                 projectRecordJSON `json:"project"`
	EvidenceKey             string            `json:"evidence_key"`
	Status                  string            `json:"status"`
	Decision                string            `json:"decision"`
	Message                 string            `json:"message"`
	EventID                 int64             `json:"event_id,omitempty"`
	AuditEventID            int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey          string            `json:"idempotency_key"`
	Created                 bool              `json:"created"`
	ProjectWriteAttempted   bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted bool              `json:"execution_write_attempted"`
	EngineCallAttempted     bool              `json:"engine_call_attempted"`
	Metadata                map[string]any    `json:"metadata"`
}

type shimReadinessItemJSON struct {
	Key      string         `json:"key"`
	Status   string         `json:"status"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type executionCutoverReadinessJSON struct {
	Project          projectRecordJSON                   `json:"project"`
	Status           string                              `json:"status"`
	Mode             string                              `json:"mode"`
	Items            []executionCutoverReadinessItemJSON `json:"items"`
	MigrationPath    []string                            `json:"migration_path"`
	CommandEvidence  map[string]int                      `json:"command_evidence"`
	Capabilities     []string                            `json:"capabilities"`
	ForbiddenActions []string                            `json:"forbidden_actions"`
	SafetyFacts      map[string]bool                     `json:"safety_facts"`
	NextSteps        []executionCutoverNextStepJSON      `json:"next_steps"`
	GeneratedAt      string                              `json:"generated_at"`
}

type executionForwardingV1ReadinessJSON struct {
	Project          projectRecordJSON                   `json:"project"`
	Status           string                              `json:"status"`
	Mode             string                              `json:"mode"`
	Items            []executionCutoverReadinessItemJSON `json:"items"`
	AllowedTaskTypes []string                            `json:"allowed_task_types"`
	CommandEvidence  map[string]int                      `json:"command_evidence"`
	Capabilities     []string                            `json:"capabilities"`
	ForbiddenActions []string                            `json:"forbidden_actions"`
	SafetyFacts      map[string]bool                     `json:"safety_facts"`
	NextSteps        []executionCutoverNextStepJSON      `json:"next_steps"`
	GeneratedAt      string                              `json:"generated_at"`
}

type executionForwardingV1ApplyPreviewJSON struct {
	Project              projectRecordJSON                           `json:"project"`
	Status               string                                      `json:"status"`
	Mode                 string                                      `json:"mode"`
	Readiness            executionForwardingV1ReadinessJSON          `json:"readiness"`
	Items                []executionForwardingV1ApplyPreviewItemJSON `json:"items"`
	AllowedTaskTypes     []string                                    `json:"allowed_task_types"`
	ForwardingTargets    []executionForwardingV1ForwardingTargetJSON `json:"forwarding_targets"`
	BlockedTargets       []executionForwardingV1BlockedTargetJSON    `json:"blocked_targets"`
	RequiredCapabilities []string                                    `json:"required_capabilities"`
	ApplyPacketFields    []string                                    `json:"apply_packet_fields"`
	FailClosedFields     []string                                    `json:"fail_closed_fields"`
	RequiredProofFacts   []string                                    `json:"required_proof_facts"`
	RequiredEvidence     []string                                    `json:"required_evidence"`
	ForbiddenActions     []string                                    `json:"forbidden_actions"`
	ApprovalRequired     bool                                        `json:"approval_required"`
	ApprovalStatus       string                                      `json:"approval_status"`
	ApplyOpen            bool                                        `json:"apply_open"`
	RollbackTarget       string                                      `json:"rollback_target"`
	SafetyFacts          map[string]bool                             `json:"safety_facts"`
	GeneratedAt          string                                      `json:"generated_at"`
}

type executionForwardingV1ApplyPacketPreviewJSON struct {
	Project                                    projectRecordJSON                     `json:"project"`
	Status                                     string                                `json:"status"`
	Mode                                       string                                `json:"mode"`
	Decision                                   string                                `json:"decision"`
	Message                                    string                                `json:"message"`
	ApplyPreview                               executionForwardingV1ApplyPreviewJSON `json:"apply_preview"`
	Gate                                       executionForwardingV1ApplyGateJSON    `json:"gate"`
	Packet                                     executionForwardingV1ApplyPacketJSON  `json:"packet"`
	ApplyGateCommand                           []string                              `json:"apply_gate_command"`
	FutureApplyCommand                         []string                              `json:"future_apply_command"`
	RequiredHumanReview                        []string                              `json:"required_human_review"`
	ForbiddenActions                           []string                              `json:"forbidden_actions"`
	SafetyFacts                                map[string]bool                       `json:"safety_facts"`
	WouldCreateCommandRequestAfterApplyCommand bool                                  `json:"would_create_command_request_after_apply_command"`
	WouldCreateRunAfterApplyCommand            bool                                  `json:"would_create_run_after_apply_command"`
	WouldCreateRunTaskAfterApplyCommand        bool                                  `json:"would_create_run_task_after_apply_command"`
	WouldCreateAuditEventAfterApplyCommand     bool                                  `json:"would_create_audit_event_after_apply_command"`
	CommandRequestCreated                      bool                                  `json:"command_request_created"`
	AreaFlowRunCreated                         bool                                  `json:"area_flow_run_created"`
	TaskLoopRunForwarded                       bool                                  `json:"task_loop_run_forwarded"`
	ProjectWriteAttempted                      bool                                  `json:"project_write_attempted"`
	ExecutionWriteAttempted                    bool                                  `json:"execution_write_attempted"`
	EngineCallAttempted                        bool                                  `json:"engine_call_attempted"`
	GeneratedAt                                string                                `json:"generated_at"`
}

type executionForwardingV1ApplyPacketJSON struct {
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

type executionForwardingV1ApplyGateJSON struct {
	Project                 projectRecordJSON                        `json:"project"`
	Status                  string                                   `json:"status"`
	Mode                    string                                   `json:"mode"`
	Decision                string                                   `json:"decision"`
	Message                 string                                   `json:"message"`
	Items                   []executionForwardingV1ApplyGateItemJSON `json:"items"`
	RequiredPacketFields    []string                                 `json:"required_packet_fields"`
	RequiredCapabilities    []string                                 `json:"required_capabilities"`
	AllowedTaskTypes        []string                                 `json:"allowed_task_types"`
	TargetCommandTypes      []string                                 `json:"target_command_types"`
	BlockedTaskTypes        []string                                 `json:"blocked_task_types"`
	ForbiddenActions        []string                                 `json:"forbidden_actions"`
	FailClosedFields        []string                                 `json:"fail_closed_fields"`
	RequiredProofFacts      []string                                 `json:"required_proof_facts"`
	SafetyFacts             map[string]bool                          `json:"safety_facts"`
	ApprovalRequired        bool                                     `json:"approval_required"`
	ApprovalStatus          string                                   `json:"approval_status"`
	ApplyCommandEligible    bool                                     `json:"apply_command_eligible"`
	ApplyOpen               bool                                     `json:"apply_open"`
	CommandRequestCreated   bool                                     `json:"command_request_created"`
	AreaFlowRunCreated      bool                                     `json:"area_flow_run_created"`
	TaskLoopRunForwarded    bool                                     `json:"task_loop_run_forwarded"`
	ProjectWriteAttempted   bool                                     `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                                     `json:"execution_write_attempted"`
	EngineCallAttempted     bool                                     `json:"engine_call_attempted"`
	GeneratedAt             string                                   `json:"generated_at"`
}

type executionForwardingV1ApplyGateItemJSON struct {
	Key              string   `json:"key"`
	Category         string   `json:"category"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	Expected         string   `json:"expected"`
	Actual           string   `json:"actual"`
	RequiredEvidence []string `json:"required_evidence"`
	BlockedBy        []string `json:"blocked_by"`
}

type executionForwardingV1ApplyJSON struct {
	Project                         projectRecordJSON                  `json:"project"`
	Status                          string                             `json:"status"`
	Decision                        string                             `json:"decision"`
	Message                         string                             `json:"message"`
	Blockers                        []string                           `json:"blockers"`
	Gate                            executionForwardingV1ApplyGateJSON `json:"gate"`
	EventID                         int64                              `json:"event_id,omitempty"`
	AuditEventID                    int64                              `json:"audit_event_id,omitempty"`
	IdempotencyKey                  string                             `json:"idempotency_key"`
	Created                         bool                               `json:"created"`
	SafetyFacts                     map[string]bool                    `json:"safety_facts"`
	CommandRequestCreated           bool                               `json:"command_request_created"`
	AreaFlowCommandCreated          bool                               `json:"area_flow_command_created"`
	AreaFlowRunCreated              bool                               `json:"area_flow_run_created"`
	AreaFlowRunTaskCreated          bool                               `json:"area_flow_run_task_created"`
	AreaFlowRunAttemptCreated       bool                               `json:"area_flow_run_attempt_created"`
	AreaFlowArtifactCreated         bool                               `json:"area_flow_artifact_created"`
	AreaFlowAuditEventCreated       bool                               `json:"area_flow_audit_event_created"`
	TaskLoopRunForwarded            bool                               `json:"task_loop_run_forwarded"`
	LegacyTaskLoopStarted           bool                               `json:"legacy_task_loop_started"`
	LegacyProgressWritten           bool                               `json:"legacy_progress_written"`
	LegacyLogsWritten               bool                               `json:"legacy_logs_written"`
	LegacyCheckpointWritten         bool                               `json:"legacy_checkpoint_written"`
	ProjectWriteAttempted           bool                               `json:"project_write_attempted"`
	ExecutionWriteAttempted         bool                               `json:"execution_write_attempted"`
	EngineCallAttempted             bool                               `json:"engine_call_attempted"`
	CommandsRun                     bool                               `json:"commands_run"`
	SecretsResolved                 bool                               `json:"secrets_resolved"`
	NetworkUsed                     bool                               `json:"network_used"`
	AreaMatrixProtectedPathsTouched bool                               `json:"areamatrix_protected_paths_touched"`
	GeneratedAt                     string                             `json:"generated_at"`
}

type executionForwardingV1ForwardingTargetJSON struct {
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

type executionForwardingV1BlockedTargetJSON struct {
	TaskType        string          `json:"task_type"`
	ForbiddenAction string          `json:"forbidden_action"`
	Reason          string          `json:"reason"`
	FailureMode     string          `json:"failure_mode"`
	SafetyFacts     map[string]bool `json:"safety_facts"`
}

type executionForwardingV1ApplyPreviewItemJSON struct {
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

type executionForwardingV1CommandPreviewJSON struct {
	Project                                projectRecordJSON `json:"project"`
	Status                                 string            `json:"status"`
	Mode                                   string            `json:"mode"`
	Decision                               string            `json:"decision"`
	Message                                string            `json:"message"`
	TaskType                               string            `json:"task_type"`
	TargetCommandType                      string            `json:"target_command_type"`
	TargetStatus                           string            `json:"target_status"`
	FailureMode                            string            `json:"failure_mode"`
	AllowedTaskType                        bool              `json:"allowed_task_type"`
	BlockedTaskType                        bool              `json:"blocked_task_type"`
	ApplyOpen                              bool              `json:"apply_open"`
	WouldCreateCommandRequestAfterApproval bool              `json:"would_create_command_request_after_approval"`
	WouldCreateRunAfterApproval            bool              `json:"would_create_run_after_approval"`
	WouldCreateRunTaskAfterApproval        bool              `json:"would_create_run_task_after_approval"`
	WouldCreateRunAttemptAfterApproval     bool              `json:"would_create_run_attempt_after_approval"`
	WouldCreateArtifactAfterApproval       bool              `json:"would_create_artifact_after_approval"`
	WouldCreateAuditEventAfterApproval     bool              `json:"would_create_audit_event_after_approval"`
	ProjectWriteAllowed                    bool              `json:"project_write_allowed"`
	ExecutionWriteAllowed                  bool              `json:"execution_write_allowed"`
	LegacyFallbackAllowed                  bool              `json:"legacy_fallback_allowed"`
	RequiredPacketFields                   []string          `json:"required_packet_fields"`
	RequiredCapabilities                   []string          `json:"required_capabilities"`
	FailClosedFields                       []string          `json:"fail_closed_fields"`
	BlockedBy                              []string          `json:"blocked_by"`
	AllowedTaskTypes                       []string          `json:"allowed_task_types"`
	ForbiddenActions                       []string          `json:"forbidden_actions"`
	SafetyFacts                            map[string]bool   `json:"safety_facts"`
	GeneratedAt                            string            `json:"generated_at"`
}

type executionForwardingV1RollbackPreviewJSON struct {
	Project            projectRecordJSON                              `json:"project"`
	Status             string                                         `json:"status"`
	Mode               string                                         `json:"mode"`
	ApplyPreview       executionForwardingV1ApplyPreviewJSON          `json:"apply_preview"`
	Items              []executionForwardingV1RollbackPreviewItemJSON `json:"items"`
	RollbackTarget     string                                         `json:"rollback_target"`
	FailClosedSteps    []string                                       `json:"fail_closed_steps"`
	ReopenConditions   []string                                       `json:"reopen_conditions"`
	RequiredProofFacts []string                                       `json:"required_proof_facts"`
	RequiredEvidence   []string                                       `json:"required_evidence"`
	ForbiddenActions   []string                                       `json:"forbidden_actions"`
	RollbackApplyOpen  bool                                           `json:"rollback_apply_open"`
	SafetyFacts        map[string]bool                                `json:"safety_facts"`
	GeneratedAt        string                                         `json:"generated_at"`
}

type executionForwardingV1RollbackPreviewItemJSON struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	Owner            string         `json:"owner"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type executionCutoverReadinessItemJSON struct {
	Key              string         `json:"key"`
	Category         string         `json:"category"`
	Status           string         `json:"status"`
	Message          string         `json:"message"`
	RequiredEvidence []string       `json:"required_evidence"`
	NextCommand      string         `json:"next_command"`
	Metadata         map[string]any `json:"metadata"`
}

type executionCutoverNextStepJSON struct {
	Key         string         `json:"key"`
	Owner       string         `json:"owner"`
	Action      string         `json:"action"`
	RiskLevel   string         `json:"risk_level"`
	BlockedBy   []string       `json:"blocked_by"`
	NextCommand string         `json:"next_command"`
	Metadata    map[string]any `json:"metadata"`
}

type compatibilityCommandJSON struct {
	Command        string         `json:"command"`
	Mode           string         `json:"mode"`
	Status         string         `json:"status"`
	Message        string         `json:"message"`
	AreaFlowTarget string         `json:"areaflow_target,omitempty"`
	Fallback       string         `json:"fallback,omitempty"`
	BlockedReason  string         `json:"blocked_reason,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

type projectPhaseGateJSON struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	AcceptedWarnings []string `json:"accepted_warnings"`
	Blockers         []string `json:"blockers"`
}

type projectEventJSON struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Severity  string         `json:"severity"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
}

type workflowVersionJSON struct {
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

type workflowItemJSON struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	Stage             string         `json:"stage"`
	ItemType          string         `json:"item_type"`
	ExternalKey       string         `json:"external_key"`
	Title             string         `json:"title"`
	Status            string         `json:"status"`
	SourcePath        string         `json:"source_path,omitempty"`
	SourceHash        string         `json:"source_hash,omitempty"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	ImportedAt        string         `json:"imported_at,omitempty"`
}

type workflowItemLinkJSON struct {
	ID                int64          `json:"id"`
	WorkflowVersionID int64          `json:"workflow_version_id"`
	FromItemID        int64          `json:"from_item_id"`
	ToItemID          int64          `json:"to_item_id"`
	RelationType      string         `json:"relation_type"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         string         `json:"created_at"`
}

type workflowVersionCreateJSON struct {
	Project         projectRecordJSON   `json:"project"`
	WorkflowVersion workflowVersionJSON `json:"workflow_version"`
	InitialItem     workflowItemJSON    `json:"initial_item"`
	StageItems      []workflowItemJSON  `json:"stage_items"`
	Created         bool                `json:"created"`
	IdempotencyKey  string              `json:"idempotency_key"`
}

type workflowVersionListJSON struct {
	Project          projectRecordJSON     `json:"project"`
	WorkflowVersions []workflowVersionJSON `json:"workflow_versions"`
}

type workflowVersionStagesJSON struct {
	Project         projectRecordJSON      `json:"project"`
	WorkflowVersion workflowVersionJSON    `json:"workflow_version"`
	Items           []workflowItemJSON     `json:"items"`
	Links           []workflowItemLinkJSON `json:"links"`
}

type workflowProfileJSON struct {
	ProfileID       string                  `json:"profile_id"`
	ProfileVersion  int                     `json:"profile_version"`
	DisplayName     string                  `json:"display_name"`
	Description     string                  `json:"description"`
	Path            string                  `json:"path"`
	SHA256          string                  `json:"sha256"`
	StageCount      int                     `json:"stage_count"`
	GateCount       int                     `json:"gate_count"`
	TransitionCount int                     `json:"transition_count"`
	Warnings        []string                `json:"warnings"`
	Profile         workflowprofile.Profile `json:"profile"`
}

type workflowProfileCheckJSON struct {
	ProfileID       string   `json:"profile_id"`
	ProfileVersion  int      `json:"profile_version"`
	Path            string   `json:"path"`
	SHA256          string   `json:"sha256"`
	Status          string   `json:"status"`
	StageCount      int      `json:"stage_count"`
	GateCount       int      `json:"gate_count"`
	TransitionCount int      `json:"transition_count"`
	Warnings        []string `json:"warnings"`
}

type workflowProfileSummaryJSON struct {
	ProfileID       string   `json:"profile_id"`
	ProfileVersion  int      `json:"profile_version"`
	DisplayName     string   `json:"display_name"`
	Description     string   `json:"description"`
	Path            string   `json:"path"`
	SHA256          string   `json:"sha256"`
	StageCount      int      `json:"stage_count"`
	GateCount       int      `json:"gate_count"`
	TransitionCount int      `json:"transition_count"`
	Warnings        []string `json:"warnings"`
}

type workflowProfileListJSON struct {
	Profiles []workflowProfileSummaryJSON `json:"profiles"`
}

type ensureStageSkeletonJSON struct {
	Project         projectRecordJSON      `json:"project"`
	WorkflowVersion workflowVersionJSON    `json:"workflow_version"`
	Items           []workflowItemJSON     `json:"items"`
	Links           []workflowItemLinkJSON `json:"links"`
	Created         int                    `json:"created"`
}

type markWorkflowItemReadyJSON struct {
	Project         projectRecordJSON   `json:"project"`
	WorkflowVersion workflowVersionJSON `json:"workflow_version"`
	Item            workflowItemJSON    `json:"item"`
	Artifact        artifactJSON        `json:"artifact"`
}

type gateResultJSON struct {
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

type gateResultsJSON struct {
	Project         projectRecordJSON   `json:"project"`
	WorkflowVersion workflowVersionJSON `json:"workflow_version"`
	GateResults     []gateResultJSON    `json:"gate_results"`
}

type transitionPreviewJSON struct {
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

type transitionPreviewsJSON struct {
	Project            projectRecordJSON       `json:"project"`
	WorkflowVersion    workflowVersionJSON     `json:"workflow_version"`
	TransitionPreviews []transitionPreviewJSON `json:"transition_previews"`
}

type approvalRecordJSON struct {
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

type approvalRecordsJSON struct {
	Project         projectRecordJSON    `json:"project"`
	WorkflowVersion workflowVersionJSON  `json:"workflow_version"`
	ApprovalRecords []approvalRecordJSON `json:"approval_records"`
}

type runnerPreviewJSON struct {
	Project         projectRecordJSON   `json:"project"`
	WorkflowVersion workflowVersionJSON `json:"workflow_version"`
	Run             runJSON             `json:"run"`
	Tasks           []runTaskJSON       `json:"tasks"`
	Attempts        []runAttemptJSON    `json:"attempts"`
	Artifacts       []artifactJSON      `json:"artifacts"`
	Preflight       runnerPreflightJSON `json:"preflight"`
	Created         bool                `json:"created"`
	IdempotencyKey  string              `json:"idempotency_key"`
}

type fixtureExecutionQueueJSON struct {
	Project                 projectRecordJSON   `json:"project"`
	WorkflowVersion         workflowVersionJSON `json:"workflow_version"`
	Run                     runJSON             `json:"run"`
	Task                    runTaskJSON         `json:"task"`
	Created                 bool                `json:"created"`
	IdempotencyKey          string              `json:"idempotency_key"`
	EventID                 int64               `json:"event_id,omitempty"`
	AuditEventID            int64               `json:"audit_event_id,omitempty"`
	ProjectWriteAttempted   bool                `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                `json:"execution_write_attempted"`
	EngineCallAttempted     bool                `json:"engine_call_attempted"`
	CommandsRun             bool                `json:"commands_run"`
	SecretsResolved         bool                `json:"secrets_resolved"`
	NetworkUsed             bool                `json:"network_used"`
}

type readOnlyVerifyQueueJSON struct {
	Project                 projectRecordJSON   `json:"project"`
	WorkflowVersion         workflowVersionJSON `json:"workflow_version"`
	Run                     runJSON             `json:"run"`
	Task                    runTaskJSON         `json:"task"`
	TargetPath              string              `json:"target_path"`
	Created                 bool                `json:"created"`
	IdempotencyKey          string              `json:"idempotency_key"`
	EventID                 int64               `json:"event_id,omitempty"`
	AuditEventID            int64               `json:"audit_event_id,omitempty"`
	ProjectReadAttempted    bool                `json:"project_read_attempted"`
	ProjectReadAllowed      bool                `json:"project_read_allowed"`
	ProjectWriteAttempted   bool                `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                `json:"execution_write_attempted"`
	EngineCallAttempted     bool                `json:"engine_call_attempted"`
	CommandsRun             bool                `json:"commands_run"`
	SecretsResolved         bool                `json:"secrets_resolved"`
	NetworkUsed             bool                `json:"network_used"`
}

type approvedArtifactWriteQueueJSON struct {
	Project                       projectRecordJSON   `json:"project"`
	WorkflowVersion               workflowVersionJSON `json:"workflow_version"`
	Run                           runJSON             `json:"run"`
	Task                          runTaskJSON         `json:"task"`
	ArtifactLabel                 string              `json:"artifact_label"`
	Created                       bool                `json:"created"`
	IdempotencyKey                string              `json:"idempotency_key"`
	EventID                       int64               `json:"event_id,omitempty"`
	AuditEventID                  int64               `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                `json:"engine_call_attempted"`
	CommandsRun                   bool                `json:"commands_run"`
	SecretsResolved               bool                `json:"secrets_resolved"`
	NetworkUsed                   bool                `json:"network_used"`
}

type fixtureProjectWriteQueueJSON struct {
	Project                       projectRecordJSON   `json:"project"`
	WorkflowVersion               workflowVersionJSON `json:"workflow_version"`
	Run                           runJSON             `json:"run"`
	Task                          runTaskJSON         `json:"task"`
	WriteSetArtifact              artifactJSON        `json:"write_set_artifact"`
	TargetPath                    string              `json:"target_path"`
	ExpectedBeforeSHA256          string              `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64               `json:"expected_before_size"`
	AfterSHA256                   string              `json:"after_sha256"`
	AfterSize                     int64               `json:"after_size"`
	Created                       bool                `json:"created"`
	IdempotencyKey                string              `json:"idempotency_key"`
	EventID                       int64               `json:"event_id,omitempty"`
	AuditEventID                  int64               `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                `json:"engine_call_attempted"`
	CommandsRun                   bool                `json:"commands_run"`
	SecretsResolved               bool                `json:"secrets_resolved"`
	NetworkUsed                   bool                `json:"network_used"`
}

type managedGeneratedWriteQueueJSON struct {
	Project                       projectRecordJSON   `json:"project"`
	WorkflowVersion               workflowVersionJSON `json:"workflow_version"`
	Run                           runJSON             `json:"run"`
	Task                          runTaskJSON         `json:"task"`
	WriteSetArtifact              artifactJSON        `json:"write_set_artifact"`
	TargetPath                    string              `json:"target_path"`
	ExpectedBeforeSHA256          string              `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64               `json:"expected_before_size"`
	AfterSHA256                   string              `json:"after_sha256"`
	AfterSize                     int64               `json:"after_size"`
	Created                       bool                `json:"created"`
	IdempotencyKey                string              `json:"idempotency_key"`
	EventID                       int64               `json:"event_id,omitempty"`
	AuditEventID                  int64               `json:"audit_event_id,omitempty"`
	GeneratedOnly                 bool                `json:"generated_only"`
	GeneratedOnlyApplyOpen        bool                `json:"generated_only_apply_open"`
	ProjectReadAttempted          bool                `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                `json:"engine_call_attempted"`
	CommandsRun                   bool                `json:"commands_run"`
	SecretsResolved               bool                `json:"secrets_resolved"`
	NetworkUsed                   bool                `json:"network_used"`
}

type runControlJSON struct {
	Project                  projectRecordJSON `json:"project"`
	Run                      runJSON           `json:"run"`
	PreviousStatus           string            `json:"previous_status"`
	Status                   string            `json:"status"`
	Decision                 string            `json:"decision"`
	Message                  string            `json:"message"`
	Blockers                 []string          `json:"blockers"`
	EventID                  int64             `json:"event_id,omitempty"`
	AuditEventID             int64             `json:"audit_event_id,omitempty"`
	IdempotencyKey           string            `json:"idempotency_key"`
	Created                  bool              `json:"created"`
	ProjectWriteAttempted    bool              `json:"project_write_attempted"`
	ExecutionWriteAttempted  bool              `json:"execution_write_attempted"`
	AreaMatrixWriteAttempted bool              `json:"area_matrix_write_attempted"`
	EngineCallAttempted      bool              `json:"engine_call_attempted"`
}

type executionApprovalGateJSON struct {
	Project                 projectRecordJSON          `json:"project"`
	WorkflowVersion         workflowVersionJSON        `json:"workflow_version"`
	Run                     runJSON                    `json:"run"`
	Status                  string                     `json:"status"`
	Mode                    string                     `json:"mode"`
	Items                   []projectReadinessItemJSON `json:"items"`
	Blockers                []string                   `json:"blockers"`
	Warnings                []string                   `json:"warnings"`
	RequiredCapabilities    []string                   `json:"required_capabilities"`
	ApprovalFound           bool                       `json:"approval_found"`
	Approval                approvalRecordJSON         `json:"approval,omitempty"`
	ApprovalGateFound       bool                       `json:"approval_gate_found"`
	ApprovalGate            gateResultJSON             `json:"approval_gate,omitempty"`
	LiveMappingGateFound    bool                       `json:"live_mapping_gate_found"`
	LiveMappingGate         gateResultJSON             `json:"live_mapping_gate,omitempty"`
	EnginePreview           codexCLIAdapterPreviewJSON `json:"engine_preview"`
	Workers                 []workerJSON               `json:"workers"`
	ForbiddenActions        []string                   `json:"forbidden_actions"`
	ProjectWriteAttempted   bool                       `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                       `json:"execution_write_attempted"`
	EngineCallAttempted     bool                       `json:"engine_call_attempted"`
	CommandsRun             bool                       `json:"commands_run"`
	SecretsResolved         bool                       `json:"secrets_resolved"`
	NetworkUsed             bool                       `json:"network_used"`
	TaskClaimed             bool                       `json:"task_claimed"`
	WorkerStarted           bool                       `json:"worker_started"`
	AttemptCreated          bool                       `json:"attempt_created"`
	ArtifactCreated         bool                       `json:"artifact_created"`
	GeneratedAt             string                     `json:"generated_at"`
}

type executionPlanPreviewJSON struct {
	Project                       projectRecordJSON         `json:"project"`
	WorkflowVersion               workflowVersionJSON       `json:"workflow_version"`
	Run                           runJSON                   `json:"run"`
	Gate                          executionApprovalGateJSON `json:"gate"`
	Status                        string                    `json:"status"`
	Mode                          string                    `json:"mode"`
	Steps                         []executionPlanStepJSON   `json:"steps"`
	Blockers                      []string                  `json:"blockers"`
	ForbiddenActions              []string                  `json:"forbidden_actions"`
	ProjectReadAttempted          bool                      `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                      `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                      `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                      `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                      `json:"engine_call_attempted"`
	CommandsRun                   bool                      `json:"commands_run"`
	SecretsResolved               bool                      `json:"secrets_resolved"`
	NetworkUsed                   bool                      `json:"network_used"`
	TaskClaimed                   bool                      `json:"task_claimed"`
	WorkerStarted                 bool                      `json:"worker_started"`
	AttemptCreated                bool                      `json:"attempt_created"`
	ArtifactCreated               bool                      `json:"artifact_created"`
	GeneratedAt                   string                    `json:"generated_at"`
}

type executionPlanStepJSON struct {
	Key                  string         `json:"key"`
	AttemptKind          string         `json:"attempt_kind"`
	Status               string         `json:"status"`
	Message              string         `json:"message"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	Prerequisites        []string       `json:"prerequisites"`
	Blockers             []string       `json:"blockers"`
	ReadsProject         bool           `json:"reads_project"`
	WritesProject        bool           `json:"writes_project"`
	WritesAreaFlow       bool           `json:"writes_areaflow"`
	UsesEngine           bool           `json:"uses_engine"`
	RunsCommands         bool           `json:"runs_commands"`
	UsesSecrets          bool           `json:"uses_secrets"`
	UsesNetwork          bool           `json:"uses_network"`
	CreatesAttempt       bool           `json:"creates_attempt"`
	CreatesArtifact      bool           `json:"creates_artifact"`
	Metadata             map[string]any `json:"metadata"`
}

type projectWriteDesignGateJSON struct {
	Project                       projectRecordJSON          `json:"project"`
	WorkflowVersion               workflowVersionJSON        `json:"workflow_version"`
	Run                           runJSON                    `json:"run"`
	Gate                          executionApprovalGateJSON  `json:"gate"`
	Status                        string                     `json:"status"`
	Mode                          string                     `json:"mode"`
	Items                         []projectReadinessItemJSON `json:"items"`
	RequiredCapabilities          []string                   `json:"required_capabilities"`
	WriteSetFields                []string                   `json:"write_set_fields"`
	UnsupportedOperations         []string                   `json:"unsupported_operations"`
	ApplySequence                 []string                   `json:"apply_sequence"`
	Blockers                      []string                   `json:"blockers"`
	ForbiddenActions              []string                   `json:"forbidden_actions"`
	ProjectWriteApplyOpen         bool                       `json:"project_write_apply_open"`
	ProjectReadAttempted          bool                       `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                       `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                       `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                       `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                       `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                       `json:"engine_call_attempted"`
	CommandsRun                   bool                       `json:"commands_run"`
	SecretsResolved               bool                       `json:"secrets_resolved"`
	NetworkUsed                   bool                       `json:"network_used"`
	TaskClaimed                   bool                       `json:"task_claimed"`
	WorkerStarted                 bool                       `json:"worker_started"`
	AttemptCreated                bool                       `json:"attempt_created"`
	ArtifactCreated               bool                       `json:"artifact_created"`
	GeneratedAt                   string                     `json:"generated_at"`
}

type managedGeneratedWriteGateJSON struct {
	Project                       projectRecordJSON          `json:"project"`
	WorkflowVersion               workflowVersionJSON        `json:"workflow_version"`
	Run                           runJSON                    `json:"run"`
	Gate                          executionApprovalGateJSON  `json:"gate"`
	Status                        string                     `json:"status"`
	Mode                          string                     `json:"mode"`
	Items                         []projectReadinessItemJSON `json:"items"`
	RequiredCapabilities          []string                   `json:"required_capabilities"`
	AllowedGeneratedPrefixes      []string                   `json:"allowed_generated_prefixes"`
	RequiredWriteSetFields        []string                   `json:"required_write_set_fields"`
	UnsupportedOperations         []string                   `json:"unsupported_operations"`
	ApplySequence                 []string                   `json:"apply_sequence"`
	Blockers                      []string                   `json:"blockers"`
	ForbiddenActions              []string                   `json:"forbidden_actions"`
	GeneratedOnlyWriteReady       bool                       `json:"generated_only_write_ready"`
	GeneratedOnlyApplyOpen        bool                       `json:"generated_only_apply_open"`
	ProjectReadAttempted          bool                       `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                       `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                       `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                       `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                       `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                       `json:"engine_call_attempted"`
	CommandsRun                   bool                       `json:"commands_run"`
	SecretsResolved               bool                       `json:"secrets_resolved"`
	NetworkUsed                   bool                       `json:"network_used"`
	TaskClaimed                   bool                       `json:"task_claimed"`
	WorkerStarted                 bool                       `json:"worker_started"`
	LeaseCreated                  bool                       `json:"lease_created"`
	AttemptCreated                bool                       `json:"attempt_created"`
	ArtifactCreated               bool                       `json:"artifact_created"`
	GeneratedAt                   string                     `json:"generated_at"`
}

type workerJSON struct {
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

type workerListJSON struct {
	Project projectRecordJSON `json:"project"`
	Workers []workerJSON      `json:"workers"`
}

type workerPoolSummaryJSON struct {
	Projects           []workerPoolProjectSummaryJSON `json:"projects"`
	TotalProjects      int64                          `json:"total_projects"`
	TotalWorkers       int64                          `json:"total_workers"`
	TotalOnlineWorkers int64                          `json:"total_online_workers"`
	TotalActiveLeases  int64                          `json:"total_active_leases"`
	TotalQueuedTasks   int64                          `json:"total_queued_tasks"`
	TotalNeedsRecovery int64                          `json:"total_needs_recovery"`
	GeneratedAt        string                         `json:"generated_at"`
}

type workerPoolProjectSummaryJSON struct {
	Project             projectRecordJSON     `json:"project"`
	Workers             int64                 `json:"workers"`
	OnlineWorkers       int64                 `json:"online_workers"`
	OfflineWorkers      int64                 `json:"offline_workers"`
	ActiveLeases        int64                 `json:"active_leases"`
	NeedsRecoveryLeases int64                 `json:"needs_recovery_leases"`
	QueuedTasks         int64                 `json:"queued_tasks"`
	NeedsRecoveryTasks  int64                 `json:"needs_recovery_tasks"`
	Capabilities        []string              `json:"capabilities"`
	WorkerTypes         []string              `json:"worker_types"`
	Scheduling          schedulingPolicyJSON  `json:"scheduling"`
	Role                roleReadinessJSON     `json:"role"`
	Engine              engineReadinessJSON   `json:"engine"`
	Resources           resourceReadinessJSON `json:"resources"`
	LastWorkerHeartbeat string                `json:"last_worker_heartbeat,omitempty"`
}

type schedulingPolicyJSON struct {
	Priority             int      `json:"priority"`
	MaxParallelTasks     int      `json:"max_parallel_tasks"`
	AgentRole            string   `json:"agent_role"`
	RequiredCapabilities []string `json:"required_capabilities"`
	EngineProfile        string   `json:"engine_profile,omitempty"`
}

type roleReadinessJSON struct {
	RequiredRole   string   `json:"required_role"`
	Matched        bool     `json:"matched"`
	MatchedTypes   []string `json:"matched_types"`
	Status         string   `json:"status"`
	BlockedReasons []string `json:"blocked_reasons"`
}

type engineReadinessJSON struct {
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

type resourceReadinessJSON struct {
	MaxActiveLeases int64    `json:"max_active_leases"`
	MaxQueuedTasks  int64    `json:"max_queued_tasks"`
	Status          string   `json:"status"`
	BlockedReasons  []string `json:"blocked_reasons"`
}

type workerPoolSchedulePreviewJSON struct {
	Projects      []workerPoolProjectScheduleJSON `json:"projects"`
	Policy        workerPoolSchedulePolicyJSON    `json:"policy"`
	GeneratedAt   string                          `json:"generated_at"`
	Recommended   int64                           `json:"recommended"`
	Blocked       int64                           `json:"blocked"`
	QueuedTasks   int64                           `json:"queued_tasks"`
	AvailableSlot int64                           `json:"available_slots"`
}

type workerPoolSchedulePolicyJSON struct {
	Strategy               string `json:"strategy"`
	DefaultProjectPriority int    `json:"default_project_priority"`
	SlotStrategy           string `json:"slot_strategy"`
	DryRunOnly             bool   `json:"dry_run_only"`
}

type workerPoolProjectScheduleJSON struct {
	Project        projectRecordJSON     `json:"project"`
	Priority       int                   `json:"priority"`
	MaxParallel    int                   `json:"max_parallel"`
	AgentRole      string                `json:"agent_role"`
	Role           roleReadinessJSON     `json:"role"`
	EngineProfile  string                `json:"engine_profile,omitempty"`
	Engine         engineReadinessJSON   `json:"engine"`
	Resources      resourceReadinessJSON `json:"resources"`
	QueuedTasks    int64                 `json:"queued_tasks"`
	ActiveLeases   int64                 `json:"active_leases"`
	OnlineWorkers  int64                 `json:"online_workers"`
	AvailableSlots int64                 `json:"available_slots"`
	NeedsRecovery  int64                 `json:"needs_recovery"`
	Capabilities   []string              `json:"capabilities"`
	RequiredCaps   []string              `json:"required_capabilities"`
	Recommended    bool                  `json:"recommended"`
	BlockedReasons []string              `json:"blocked_reasons"`
	NextAction     string                `json:"next_action"`
}

type codexCLIAdapterPreviewJSON struct {
	Project                 projectRecordJSON               `json:"project"`
	Status                  string                          `json:"status"`
	Mode                    string                          `json:"mode"`
	Engine                  engineReadinessJSON             `json:"engine"`
	Command                 engineCommandPreviewJSON        `json:"command"`
	Capabilities            []engineCapabilityPreflightJSON `json:"capabilities"`
	Paths                   []enginePathPreflightJSON       `json:"paths"`
	ArtifactRedaction       artifactRedactionPlanJSON       `json:"artifact_redaction"`
	ForbiddenActions        []string                        `json:"forbidden_actions"`
	Blockers                []string                        `json:"blockers"`
	ExecutionAllowed        bool                            `json:"execution_allowed"`
	ProjectWriteAttempted   bool                            `json:"project_write_attempted"`
	ExecutionWriteAttempted bool                            `json:"execution_write_attempted"`
	EngineCallAttempted     bool                            `json:"engine_call_attempted"`
	CommandsRun             bool                            `json:"commands_run"`
	SecretsResolved         bool                            `json:"secrets_resolved"`
	NetworkUsed             bool                            `json:"network_used"`
	GeneratedAt             string                          `json:"generated_at"`
}

type engineCommandPreviewJSON struct {
	Command           string `json:"command"`
	Allowed           bool   `json:"allowed"`
	Reason            string `json:"reason"`
	CapabilityAllowed bool   `json:"capability_allowed"`
	CommandAllowed    bool   `json:"command_allowed"`
	Denied            bool   `json:"denied"`
}

type engineCapabilityPreflightJSON struct {
	Capability string `json:"capability"`
	Required   bool   `json:"required"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
}

type enginePathPreflightJSON struct {
	Path       string `json:"path"`
	Capability string `json:"capability"`
	Effect     string `json:"effect"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
}

type artifactRedactionPlanJSON struct {
	Status         string   `json:"status"`
	RetentionClass string   `json:"retention_class"`
	Rules          []string `json:"rules"`
	RedactedFields []string `json:"redacted_fields"`
}

type leaseJSON struct {
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

type leaseRecoverJSON struct {
	Project projectRecordJSON `json:"project"`
	Leases  []leaseJSON       `json:"leases"`
}

type workerRunOnceJSON struct {
	Project  projectRecordJSON `json:"project"`
	Worker   workerJSON        `json:"worker"`
	Lease    *leaseJSON        `json:"lease,omitempty"`
	Task     *runTaskJSON      `json:"task,omitempty"`
	Attempt  *runAttemptJSON   `json:"attempt,omitempty"`
	Artifact *artifactJSON     `json:"artifact,omitempty"`
	Claimed  bool              `json:"claimed"`
}

type fixtureExecutionJSON struct {
	Project                       projectRecordJSON         `json:"project"`
	WorkflowVersion               workflowVersionJSON       `json:"workflow_version"`
	Run                           runJSON                   `json:"run"`
	Worker                        workerJSON                `json:"worker"`
	Lease                         leaseJSON                 `json:"lease"`
	Task                          runTaskJSON               `json:"task"`
	Attempt                       runAttemptJSON            `json:"attempt"`
	Artifact                      artifactJSON              `json:"artifact"`
	Gate                          executionApprovalGateJSON `json:"gate"`
	Status                        string                    `json:"status"`
	Decision                      string                    `json:"decision"`
	Message                       string                    `json:"message"`
	Blockers                      []string                  `json:"blockers"`
	Created                       bool                      `json:"created"`
	IdempotencyKey                string                    `json:"idempotency_key"`
	EventID                       int64                     `json:"event_id,omitempty"`
	AuditEventID                  int64                     `json:"audit_event_id,omitempty"`
	ProjectWriteAttempted         bool                      `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                      `json:"execution_write_attempted"`
	AreaFlowExecutionStateWritten bool                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                      `json:"engine_call_attempted"`
	CommandsRun                   bool                      `json:"commands_run"`
	SecretsResolved               bool                      `json:"secrets_resolved"`
	NetworkUsed                   bool                      `json:"network_used"`
	TaskClaimed                   bool                      `json:"task_claimed"`
	WorkerStarted                 bool                      `json:"worker_started"`
	LeaseCreated                  bool                      `json:"lease_created"`
	AttemptCreated                bool                      `json:"attempt_created"`
	ArtifactCreated               bool                      `json:"artifact_created"`
}

type readOnlyVerifyJSON struct {
	Project                       projectRecordJSON         `json:"project"`
	WorkflowVersion               workflowVersionJSON       `json:"workflow_version"`
	Run                           runJSON                   `json:"run"`
	Worker                        workerJSON                `json:"worker"`
	Lease                         leaseJSON                 `json:"lease"`
	Task                          runTaskJSON               `json:"task"`
	Attempt                       runAttemptJSON            `json:"attempt"`
	Artifact                      artifactJSON              `json:"artifact"`
	Gate                          executionApprovalGateJSON `json:"gate"`
	TargetPath                    string                    `json:"target_path"`
	TargetSHA256                  string                    `json:"target_sha256"`
	TargetSizeBytes               int64                     `json:"target_size_bytes"`
	Status                        string                    `json:"status"`
	Decision                      string                    `json:"decision"`
	Message                       string                    `json:"message"`
	Blockers                      []string                  `json:"blockers"`
	Created                       bool                      `json:"created"`
	IdempotencyKey                string                    `json:"idempotency_key"`
	EventID                       int64                     `json:"event_id,omitempty"`
	AuditEventID                  int64                     `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                      `json:"project_read_attempted"`
	ProjectReadAllowed            bool                      `json:"project_read_allowed"`
	ProjectWriteAttempted         bool                      `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                      `json:"execution_write_attempted"`
	AreaFlowExecutionStateWritten bool                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                      `json:"engine_call_attempted"`
	CommandsRun                   bool                      `json:"commands_run"`
	SecretsResolved               bool                      `json:"secrets_resolved"`
	NetworkUsed                   bool                      `json:"network_used"`
	TaskClaimed                   bool                      `json:"task_claimed"`
	WorkerStarted                 bool                      `json:"worker_started"`
	LeaseCreated                  bool                      `json:"lease_created"`
	AttemptCreated                bool                      `json:"attempt_created"`
	ArtifactCreated               bool                      `json:"artifact_created"`
	VerificationPassed            bool                      `json:"verification_passed"`
}

type approvedArtifactWriteJSON struct {
	Project                       projectRecordJSON         `json:"project"`
	WorkflowVersion               workflowVersionJSON       `json:"workflow_version"`
	Run                           runJSON                   `json:"run"`
	Worker                        workerJSON                `json:"worker"`
	Lease                         leaseJSON                 `json:"lease"`
	Task                          runTaskJSON               `json:"task"`
	Attempt                       runAttemptJSON            `json:"attempt"`
	Artifact                      artifactJSON              `json:"artifact"`
	Gate                          executionApprovalGateJSON `json:"gate"`
	ArtifactLabel                 string                    `json:"artifact_label"`
	Status                        string                    `json:"status"`
	Decision                      string                    `json:"decision"`
	Message                       string                    `json:"message"`
	Blockers                      []string                  `json:"blockers"`
	Created                       bool                      `json:"created"`
	IdempotencyKey                string                    `json:"idempotency_key"`
	EventID                       int64                     `json:"event_id,omitempty"`
	AuditEventID                  int64                     `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                      `json:"project_read_attempted"`
	ProjectWriteAttempted         bool                      `json:"project_write_attempted"`
	ExecutionWriteAttempted       bool                      `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                      `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                      `json:"engine_call_attempted"`
	CommandsRun                   bool                      `json:"commands_run"`
	SecretsResolved               bool                      `json:"secrets_resolved"`
	NetworkUsed                   bool                      `json:"network_used"`
	TaskClaimed                   bool                      `json:"task_claimed"`
	WorkerStarted                 bool                      `json:"worker_started"`
	LeaseCreated                  bool                      `json:"lease_created"`
	AttemptCreated                bool                      `json:"attempt_created"`
	ArtifactCreated               bool                      `json:"artifact_created"`
	ArtifactWritePassed           bool                      `json:"artifact_write_passed"`
}

type fixtureProjectWriteJSON struct {
	Project                       projectRecordJSON         `json:"project"`
	WorkflowVersion               workflowVersionJSON       `json:"workflow_version"`
	Run                           runJSON                   `json:"run"`
	Worker                        workerJSON                `json:"worker"`
	Lease                         leaseJSON                 `json:"lease"`
	Task                          runTaskJSON               `json:"task"`
	CopyAttempt                   runAttemptJSON            `json:"copy_attempt"`
	VerifyAttempt                 runAttemptJSON            `json:"verify_attempt"`
	RollbackAttempt               runAttemptJSON            `json:"rollback_attempt"`
	WriteSetArtifact              artifactJSON              `json:"write_set_artifact"`
	PreimageArtifact              artifactJSON              `json:"preimage_artifact"`
	Artifact                      artifactJSON              `json:"artifact"`
	Gate                          executionApprovalGateJSON `json:"gate"`
	TargetPath                    string                    `json:"target_path"`
	ExpectedBeforeSHA256          string                    `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64                     `json:"expected_before_size"`
	AfterSHA256                   string                    `json:"after_sha256"`
	AfterSize                     int64                     `json:"after_size"`
	RestoredSHA256                string                    `json:"restored_sha256"`
	RestoredSize                  int64                     `json:"restored_size"`
	Status                        string                    `json:"status"`
	Decision                      string                    `json:"decision"`
	Message                       string                    `json:"message"`
	Blockers                      []string                  `json:"blockers"`
	Created                       bool                      `json:"created"`
	IdempotencyKey                string                    `json:"idempotency_key"`
	EventID                       int64                     `json:"event_id,omitempty"`
	AuditEventID                  int64                     `json:"audit_event_id,omitempty"`
	ProjectReadAttempted          bool                      `json:"project_read_attempted"`
	ProjectReadAllowed            bool                      `json:"project_read_allowed"`
	ProjectWriteAttempted         bool                      `json:"project_write_attempted"`
	ProjectWriteAllowed           bool                      `json:"project_write_allowed"`
	ExecutionWriteAttempted       bool                      `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                      `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                      `json:"engine_call_attempted"`
	CommandsRun                   bool                      `json:"commands_run"`
	SecretsResolved               bool                      `json:"secrets_resolved"`
	NetworkUsed                   bool                      `json:"network_used"`
	TaskClaimed                   bool                      `json:"task_claimed"`
	WorkerStarted                 bool                      `json:"worker_started"`
	LeaseCreated                  bool                      `json:"lease_created"`
	AttemptCreated                bool                      `json:"attempt_created"`
	ArtifactCreated               bool                      `json:"artifact_created"`
	WriteSetPassed                bool                      `json:"write_set_passed"`
	VerificationPassed            bool                      `json:"verification_passed"`
	RollbackAttempted             bool                      `json:"rollback_attempted"`
	RollbackVerified              bool                      `json:"rollback_verified"`
}

type managedGeneratedWriteJSON struct {
	Project                       projectRecordJSON         `json:"project"`
	WorkflowVersion               workflowVersionJSON       `json:"workflow_version"`
	Run                           runJSON                   `json:"run"`
	Worker                        workerJSON                `json:"worker"`
	Lease                         leaseJSON                 `json:"lease"`
	Task                          runTaskJSON               `json:"task"`
	CopyAttempt                   runAttemptJSON            `json:"copy_attempt"`
	VerifyAttempt                 runAttemptJSON            `json:"verify_attempt"`
	RollbackAttempt               runAttemptJSON            `json:"rollback_attempt"`
	WriteSetArtifact              artifactJSON              `json:"write_set_artifact"`
	PreimageArtifact              artifactJSON              `json:"preimage_artifact"`
	Artifact                      artifactJSON              `json:"artifact"`
	Gate                          executionApprovalGateJSON `json:"gate"`
	TargetPath                    string                    `json:"target_path"`
	ExpectedBeforeSHA256          string                    `json:"expected_before_sha256"`
	ExpectedBeforeSize            int64                     `json:"expected_before_size"`
	AfterSHA256                   string                    `json:"after_sha256"`
	AfterSize                     int64                     `json:"after_size"`
	RestoredSHA256                string                    `json:"restored_sha256"`
	RestoredSize                  int64                     `json:"restored_size"`
	Status                        string                    `json:"status"`
	Decision                      string                    `json:"decision"`
	Message                       string                    `json:"message"`
	Blockers                      []string                  `json:"blockers"`
	Created                       bool                      `json:"created"`
	IdempotencyKey                string                    `json:"idempotency_key"`
	EventID                       int64                     `json:"event_id,omitempty"`
	AuditEventID                  int64                     `json:"audit_event_id,omitempty"`
	GeneratedOnly                 bool                      `json:"generated_only"`
	GeneratedOnlyApplyOpen        bool                      `json:"generated_only_apply_open"`
	ProjectReadAttempted          bool                      `json:"project_read_attempted"`
	ProjectReadAllowed            bool                      `json:"project_read_allowed"`
	ProjectWriteAttempted         bool                      `json:"project_write_attempted"`
	ProjectWriteAllowed           bool                      `json:"project_write_allowed"`
	ExecutionWriteAttempted       bool                      `json:"execution_write_attempted"`
	AreaFlowArtifactWritten       bool                      `json:"area_flow_artifact_written"`
	AreaFlowExecutionStateWritten bool                      `json:"area_flow_execution_state_written"`
	EngineCallAttempted           bool                      `json:"engine_call_attempted"`
	CommandsRun                   bool                      `json:"commands_run"`
	SecretsResolved               bool                      `json:"secrets_resolved"`
	NetworkUsed                   bool                      `json:"network_used"`
	TaskClaimed                   bool                      `json:"task_claimed"`
	WorkerStarted                 bool                      `json:"worker_started"`
	LeaseCreated                  bool                      `json:"lease_created"`
	AttemptCreated                bool                      `json:"attempt_created"`
	ArtifactCreated               bool                      `json:"artifact_created"`
	WriteSetPassed                bool                      `json:"write_set_passed"`
	VerificationPassed            bool                      `json:"verification_passed"`
	RollbackAttempted             bool                      `json:"rollback_attempted"`
	RollbackVerified              bool                      `json:"rollback_verified"`
}

type artifactJSON struct {
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

type runnerPreflightJSON struct {
	Status   string                     `json:"status"`
	Checks   []projectReadinessItemJSON `json:"checks"`
	Blockers []string                   `json:"blockers"`
}

type runJSON struct {
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

type runTaskJSON struct {
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

type runAttemptJSON struct {
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

// Run dispatches the areaflow CLI. The command surface is intentionally small
// until v0.1 import/mirror logic lands.
func Run(ctx context.Context, args []string) error {
	cmd := command{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	return cmd.run(ctx, args)
}

func (c command) run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		c.printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		c.printHelp()
		return nil
	case "version":
		fmt.Fprintln(c.stdout, "areaflow phase-0.1")
		return nil
	case "server":
		cfg := config.FromEnv()
		return api.Serve(ctx, cfg.Server)
	case "migrate":
		return c.runMigrate(ctx, args[1:])
	case "project":
		return c.runProject(ctx, args[1:])
	case "workflow":
		return c.runWorkflow(ctx, args[1:])
	case "run":
		return c.runRun(ctx, args[1:])
	case "worker":
		return c.runWorker(ctx, args[1:])
	case "engine":
		return c.runEngine(ctx, args[1:])
	case "service":
		return c.runService(ctx, args[1:])
	case "desktop":
		return c.runDesktop(ctx, args[1:])
	case "security":
		return c.runSecurity(ctx, args[1:])
	case "completion":
		return c.runCompletion(ctx, args[1:])
	case "ops":
		return c.runOps(ctx, args[1:])
	case "support":
		return c.runSupport(ctx, args[1:])
	case "backup":
		return c.runBackup(ctx, args[1:])
	case "release":
		return c.runRelease(ctx, args[1:])
	case "audit":
		return c.runAudit(ctx, args[1:])
	case "permissions":
		return c.runPermissions(ctx, args[1:])
	case "artifact":
		return c.runArtifact(ctx, args[1:])
	case "conformance":
		return c.runConformance(ctx, args[1:])
	case "health":
		fmt.Fprintln(c.stdout, "ok")
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun `areaflow help` for available commands.", args[0])
	}
}

func (c command) printHelp() {
	fmt.Fprintln(c.stdout, `AreaFlow

Usage:
  areaflow help
  areaflow version
  areaflow health
  areaflow server
  areaflow service status
  areaflow service status --json
  areaflow desktop service-control-gate
  areaflow desktop service-control-gate --json
  areaflow desktop notification-gate
  areaflow desktop notification-gate --json
  areaflow desktop tray-menu-gate
  areaflow desktop tray-menu-gate --json
  areaflow security boundary-readiness
  areaflow security boundary-readiness --json
  areaflow completion audit
  areaflow completion audit --json
  areaflow completion audit-snapshot readiness areamatrix --json
  areaflow completion audit-snapshot record areamatrix --release-candidate v1.0-rc1 --evidence-class fixture --evidence-uri local:completion-audit --json
  areaflow completion audit-snapshot record areamatrix --release-candidate v1.0-rc1 --evidence-class release_candidate --evidence-uri docs/development/real-release-candidate-evidence.md --summary "real release candidate evidence reviewed" --review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z --json
  areaflow completion archive-proof record areamatrix --status incomplete --fact historical_workflow_versions_marked_immutable
  areaflow completion archive-proof record areamatrix --status complete --fact historical_workflow_versions_marked_immutable --fact historical_execution_metadata_indexed_in_areaflow --fact historical_artifact_refs_have_hash_path_type_project_version_run --fact project_reference_restore_limitations_recorded --fact old_progress_logs_checkpoints_are_reference_only --fact new_run_attempt_artifact_audit_state_owned_by_areaflow --fact areamatrix_workflow_readme_summary_contract_reviewed --fact areamatrix_status_json_rough_projection_contract_reviewed --fact archive_does_not_delete_or_move_historical_files --fact archive_does_not_rewrite_progress_json --fact rollback_to_execution_forwarding_documented --summary "real release candidate archive evidence reviewed" --evidence-uri docs/development/real-release-candidate-evidence.md#archive-gate --review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z --archive-scope areamatrix_historical_execution_reference_only --archive-reference-mode metadata_indexed_reference_only --archive-source-path .areaflow/status.json --archive-source-path workflow/README.md --archive-source-path 'workflow/versions/**/execution/**' --archive-source-path 'workflow/versions/**/execution/_shared/progress.json' --archive-forbidden-action copy_artifact_bytes --archive-forbidden-action delete_artifact_bytes --archive-forbidden-action delete_historical_files --archive-forbidden-action move_historical_files --archive-forbidden-action rewrite_progress_json --archive-forbidden-action run_commands --archive-forbidden-action write_areamatrix_protected_paths --archive-rollback-target execution_forwarding_read_only_shim --archive-fail-closed --json
  areaflow completion shim-retirement-proof record areamatrix --status incomplete --fact archive_gate_passed
  areaflow completion shim-retirement-proof record areamatrix --status complete --fact archive_gate_passed --fact execution_forwarding_stable_for_declared_window --fact no_legacy_task_loop_run_usage_in_active_workflow_versions --fact areaflow_run_attempt_artifact_audit_coverage_pass --fact compat_commands_mapped_or_deliberately_blocked --fact legacy_progress_log_checkpoint_archive_reference_policy_accepted --fact rollback_to_read_only_shim_documented --fact user_facing_retirement_notice_present --fact protected_path_proof_reference_recorded --summary "real release candidate shim retirement evidence reviewed" --evidence-uri docs/development/real-release-candidate-evidence.md#shim-retirement-gate --review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z --shim-retirement-scope read_only_shim_retirement_after_execution_forwarding_v1 --shim-prerequisite archive_gate_passed --shim-prerequisite execution_cutover_gate_passed --shim-prerequisite protected_path_proof_recorded --shim-retired-surface legacy_task_loop_runner --shim-retired-surface legacy_progress_json_writes --shim-retired-surface legacy_logs_writes --shim-retired-surface legacy_checkpoint_writes --shim-rollback-target read_only_shim --shim-fail-closed --shim-reopen-requires-approval --json
  areaflow completion execution-cutover-proof record areamatrix --status complete --fact explicit_execution_cutover_approval_recorded --fact execution_cutover_command_response_recorded --fact execution_cutover_event_and_audit_recorded --fact task_loop_run_forwarding_window_proven --fact rollback_plan_and_compatibility_window_proven --fact no_unapproved_project_or_execution_write_attempted --summary "real release candidate execution cutover evidence reviewed" --evidence-uri docs/development/real-release-candidate-evidence.md#execution-cutover-gate --review-decision approved --reviewed-by release-owner --reviewed-at 2026-07-04T12:00:00Z --execution-cutover-scope execution_forwarding_v1_read_only_evidence_only --allowed-task-types read_only_verify,doctor_readiness,artifact_evidence,status_projection_validation,release_readiness_check --forbidden-actions start_legacy_task_loop_runner,write_legacy_progress_json,write_legacy_logs,write_legacy_checkpoint,write_areamatrix_source,write_areamatrix_execution_directory,generated_retained_write,repair_apply,checkpoint_apply,engine_execution,secret_resolve,network_api_integration,publish_apply,restore_apply --rollback-target read_only_shim --rollback-mode fail_closed_to_read_only_shim --fail-closed --reopen-requires-approval --json
  areaflow completion validation-proof record areamatrix --status complete --fact go_test_passed --json
  areaflow completion source-alignment-proof record areamatrix --status complete --fact zero_to_hundred_phases_aligned --json
  areaflow completion task-matrix-proof record areamatrix --status complete --fact all_v0_v1_tasks_have_status_evidence_and_boundary --source-set-hash <sha256> --backlog-hash <sha256> --task-status-audit-hash <sha256> --planned-v1-required-task-count 0 --missing-evidence-v1-required-task-count 0 --blocked-v1-required-task-count 0 --json
  areaflow completion security-closure-proof record areamatrix --status complete --fact project_key_isolation_covers_workflow_run_lease_artifact_secret_audit --json
  areaflow completion backup-restore-proof record areamatrix --status complete --fact backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata --backup-manifest-hash <sha256> --restore-plan-scope project --restore-plan-project-key areamatrix --restore-plan-manifest-hash <sha256> --json
  areaflow completion release-packaging-proof record areamatrix --status complete --fact release_final_gate_passed --json
  areaflow completion protected-path-proof record areamatrix --status clean --summary "AreaMatrix protected paths clean" --evidence-uri local:areamatrix-protected-path-git-status
  areaflow completion protected-path-proof record areamatrix --status clean --summary "AreaMatrix protected paths clean" --evidence-uri local:areamatrix-protected-path-git-status --json
  areaflow ops readiness
  areaflow ops readiness --json
  areaflow ops migration-ledger-readiness
  areaflow ops migration-ledger-readiness --json
  areaflow ops smoke-proof record <project> --key local_ops_smoke
  areaflow ops smoke-proof record <project> --key local_ops_smoke --json
  areaflow support bundle-preview
  areaflow support bundle-preview --json
  areaflow backup manifest
  areaflow backup manifest --json
  areaflow backup restore-plan
  areaflow backup restore-plan --json
  areaflow release readiness
  areaflow release readiness --project areamatrix
  areaflow release readiness --json
  areaflow release remediation-plan
  areaflow release remediation-plan --project areamatrix
  areaflow release remediation-plan --json
  areaflow release acceptance-preview
  areaflow release acceptance-preview --project areamatrix
  areaflow release acceptance-preview --json
  areaflow release acceptance-gate
  areaflow release acceptance-gate --project areamatrix
  areaflow release acceptance-gate --json
  areaflow release exception-doctor
  areaflow release exception-doctor --project areamatrix
  areaflow release exception-doctor --json
  areaflow release exception-record-preview
  areaflow release exception-record-preview --project areamatrix
  areaflow release exception-record-preview --json
  areaflow release exception-schema-preview
  areaflow release exception-schema-preview --project areamatrix
  areaflow release exception-schema-preview --json
  areaflow release exception-migration-approval-gate
  areaflow release exception-migration-approval-gate --project areamatrix
  areaflow release exception-migration-approval-gate --json
  areaflow release exception-migration-approve --actor release-owner --reason "approve reviewed 000012 migration"
  areaflow release exception-migration-apply
  areaflow release exception-migration-revoke --actor release-owner --reason "disable release exception writes"
  areaflow release exception-request --project areamatrix --exception-key release_exception:restore_plan --actor release-owner --reason "accept metadata-only history"
  areaflow release exception-approve --project areamatrix --exception-key release_exception:restore_plan --actor release-owner --reason "evidence reviewed"
  areaflow release exception-revoke --project areamatrix --exception-key release_exception:restore_plan --actor release-owner --reason "risk acceptance withdrawn"
  areaflow release exception-apply-preview
  areaflow release exception-apply-preview --project areamatrix
  areaflow release exception-apply-preview --json
  areaflow release final-gate
  areaflow release final-gate --project areamatrix
  areaflow release final-gate --json
  areaflow release evidence-bundle
  areaflow release evidence-bundle --project areamatrix
  areaflow release evidence-bundle --json
  areaflow release package-preview
  areaflow release package-preview --project areamatrix
  areaflow release package-preview --json
  areaflow release distribution-preview
  areaflow release distribution-preview --project areamatrix
  areaflow release distribution-preview --json
  areaflow release publish-gate
  areaflow release publish-gate --project areamatrix
  areaflow release publish-gate --json
  areaflow release publish-approval-preview
  areaflow release publish-approval-preview --project areamatrix
  areaflow release publish-approval-preview --json
  areaflow release rollout-plan-preview
  areaflow release rollout-plan-preview --project areamatrix
  areaflow release rollout-plan-preview --json
  areaflow audit coverage
  areaflow audit coverage --project areamatrix --json
  areaflow permissions doctor areamatrix
  areaflow permissions doctor areamatrix --json
  areaflow artifact integrity areamatrix
  areaflow artifact integrity areamatrix --json
  areaflow artifact archive-preview areamatrix
  areaflow artifact archive-preview areamatrix --json
  areaflow conformance check areamatrix
  areaflow conformance check areamatrix --json
  areaflow engine codex-preview areamatrix
  areaflow engine codex-preview areamatrix --json
  areaflow migrate up
  areaflow migrate status
  areaflow project add --config examples/areamatrix/areaflow.yaml
  areaflow project status areamatrix
  areaflow project summary areamatrix
  areaflow project summary areamatrix --json
  areaflow project readiness areamatrix
  areaflow project readiness areamatrix --json
  areaflow project generated-write-readiness areamatrix
  areaflow project generated-write-readiness areamatrix --json
  areaflow project generated-write-apply-beta-gate areamatrix
  areaflow project generated-write-apply-beta-gate areamatrix --json
  areaflow project import-diff areamatrix
  areaflow project import-diff areamatrix --json
  areaflow project verify-bundle areamatrix
  areaflow project verify-bundle areamatrix --json
  areaflow project compatibility areamatrix
  areaflow project compatibility areamatrix --json
  areaflow project shim-preview areamatrix
  areaflow project shim-preview areamatrix --json
  areaflow project shim-readiness areamatrix
  areaflow project shim-readiness areamatrix --json
  areaflow project shim-authorization areamatrix
  areaflow project shim-authorization areamatrix --json
  areaflow project shim-apply-packet areamatrix
  areaflow project shim-apply-packet areamatrix --json
  areaflow project shim-apply-gate areamatrix
  areaflow project shim-apply-gate areamatrix --json
  areaflow project shim-apply areamatrix
  areaflow project shim-apply areamatrix --json
  areaflow project shim-readiness-evidence areamatrix --key real_areamatrix_readonly_smoke
  areaflow project shim-readiness-evidence areamatrix --key areamatrix_dirty_worktree_review --json
  areaflow project execution-cutover-readiness areamatrix
  areaflow project execution-cutover-readiness areamatrix --json
  areaflow project execution-forwarding-v1-readiness areamatrix
  areaflow project execution-forwarding-v1-readiness areamatrix --json
  areaflow project execution-forwarding-v1-apply-preview areamatrix
  areaflow project execution-forwarding-v1-apply-preview areamatrix --json
  areaflow project execution-forwarding-v1-apply-packet areamatrix
  areaflow project execution-forwarding-v1-apply-packet areamatrix --json
  areaflow project execution-forwarding-v1-apply-gate areamatrix
  areaflow project execution-forwarding-v1-apply-gate areamatrix --json
  areaflow project execution-forwarding-v1-command-preview areamatrix --task-type read_only_verify
  areaflow project execution-forwarding-v1-command-preview areamatrix --task-type engine_execution --json
  areaflow project execution-forwarding-v1-rollback-preview areamatrix
  areaflow project execution-forwarding-v1-rollback-preview areamatrix --json
  areaflow project cutover-readiness areamatrix --version v2
  areaflow project cutover-readiness areamatrix --version v2 --json
  areaflow project cutover-apply areamatrix --version v2
  areaflow project cutover-apply areamatrix --version v2 --json
  areaflow project status-projections areamatrix
  areaflow project status-projections areamatrix --json
  areaflow project status-projection-authorization areamatrix
  areaflow project status-projection-authorization areamatrix --json
  areaflow project status-projection-apply-packet areamatrix
  areaflow project status-projection-apply-packet areamatrix --json
  areaflow project status-projection-apply-gate areamatrix
  areaflow project status-projection-apply-gate areamatrix --json
  areaflow project status-projection-apply areamatrix
  areaflow project status-projection-apply areamatrix --json
  areaflow project list
  areaflow project import areamatrix
  areaflow project export-status areamatrix
  areaflow project doctor areamatrix
  areaflow project doctor areamatrix --json
  areaflow project doctor areamatrix --allow-native --json
  areaflow project events areamatrix
  areaflow workflow version create areamatrix v2
  areaflow workflow version create areamatrix v2 --json
  areaflow workflow version list areamatrix
  areaflow workflow version show areamatrix v2
  areaflow workflow version stages areamatrix v2
  areaflow workflow version ensure-skeleton areamatrix v2
  areaflow workflow version mark-ready areamatrix v2 --stage queue --item-type queue_candidate
  areaflow workflow profile list
  areaflow workflow profile list --json
  areaflow workflow profile show areamatrix
  areaflow workflow profile check areamatrix
  areaflow workflow gate run areamatrix v2 discussion_gate
  areaflow workflow gate run areamatrix v2 plan_doctor
  areaflow workflow gate list areamatrix v2
  areaflow workflow transition preview areamatrix v2
  areaflow workflow transition list areamatrix v2
  areaflow workflow approval record areamatrix v2 --decision rejected
  areaflow workflow approval list areamatrix v2
  areaflow run preview areamatrix v2
  areaflow run preview areamatrix v2 --json
  areaflow run fixture-queue areamatrix v2
  areaflow run fixture-queue areamatrix v2 --json
  areaflow run read-only-verify-queue areamatrix v2 --target-path docs/README.md
  areaflow run read-only-verify-queue areamatrix v2 --target-path docs/README.md --json
  areaflow run approved-artifact-write-queue areamatrix v2 --artifact-label approval-note
  areaflow run approved-artifact-write-queue areamatrix v2 --artifact-label approval-note --json
  areaflow run fixture-project-write-queue areamatrix-fixture v2 --target-path fixtures/input.txt --content "after" --expected-before-sha256 HASH --expected-before-size 6
  areaflow run fixture-project-write-queue areamatrix-fixture v2 --target-path fixtures/input.txt --content "after" --expected-before-sha256 HASH --expected-before-size 6 --json
  areaflow run managed-generated-write-queue areamatrix-fixture v2 --target-path .areaflow/generated/status.json --content "after" --expected-before-sha256 HASH --expected-before-size 6
  areaflow run managed-generated-write-queue areamatrix-fixture v2 --target-path .areaflow/generated/status.json --content "after" --expected-before-sha256 HASH --expected-before-size 6 --json
  areaflow run execution-gate 3
  areaflow run execution-gate 3 --json
  areaflow run execution-plan 3
  areaflow run execution-plan 3 --json
  areaflow run project-write-design-gate 3
  areaflow run project-write-design-gate 3 --json
  areaflow run managed-generated-write-gate 3
  areaflow run managed-generated-write-gate 3 --json
  areaflow run start 3
  areaflow run drain 3
  areaflow run cancel 3
  areaflow run start 3 --json
  areaflow worker register areamatrix --worker-key local-1
  areaflow worker heartbeat areamatrix local-1
  areaflow worker list areamatrix
  areaflow worker pool-summary
  areaflow worker schedule-preview
  areaflow worker lease-acquire areamatrix local-1 --run-task-id 1
  areaflow worker lease-release areamatrix local-1 --lease-id 1
  areaflow worker lease-recover areamatrix
  areaflow worker run-once areamatrix local-1
  areaflow worker run-once areamatrix local-1 --run-id 3
  areaflow worker fixture-execute areamatrix local-1 --run-id 3
  areaflow worker read-only-verify areamatrix local-1 --run-id 3
  areaflow worker approved-artifact-write areamatrix local-1 --run-id 3
  areaflow worker fixture-project-write areamatrix-fixture local-1 --run-id 3
  areaflow worker managed-generated-write areamatrix-fixture local-1 --run-id 3

Phase 0.1 / v0.3c scope:
  - Go binary skeleton
  - local REST health endpoint
  - PostgreSQL migration baseline
  - AreaMatrix metadata import, status mirror, and read-only doctor
  - DB-only workflow version candidate, stage skeleton, workflow gate results, transition previews, and approval records
  - no task execution yet`)
}

func (c command) runMigrate(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing migrate command: use `areaflow migrate up` or `areaflow migrate status`")
	}

	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch args[0] {
	case "up":
		applied, err := migrate.Up(ctx, pool)
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Fprintln(c.stdout, "migrations already up to date")
			return nil
		}
		for _, name := range applied {
			fmt.Fprintf(c.stdout, "applied %s\n", name)
		}
		return nil
	case "status":
		statuses, err := migrate.Statuses(ctx, pool)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			state := "pending"
			if status.Applied {
				state = "applied"
			}
			fmt.Fprintf(c.stdout, "%s %s\n", state, status.Name)
		}
		return nil
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func (c command) runProject(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing project command: use `add`, `status`, or `list`")
	}
	if len(args) >= 2 && isHelpFlag(args[1]) {
		if usage, ok := projectSubcommandUsage(args[0]); ok {
			fmt.Fprintln(c.stdout, usage)
			return nil
		}
	}

	if args[0] == "add" {
		if _, err := projectConfigPath(args[1:]); err != nil {
			return err
		}
	}
	if args[0] == "status" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project status <id>`")
	}
	if args[0] == "summary" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project summary <id>`")
	}
	if args[0] == "readiness" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project readiness <id>`")
	}
	if args[0] == "generated-write-readiness" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project generated-write-readiness <id>`")
	}
	if args[0] == "generated-write-apply-beta-gate" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project generated-write-apply-beta-gate <id>`")
	}
	if args[0] == "import-diff" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project import-diff <id>`")
	}
	if args[0] == "verify-bundle" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project verify-bundle <id>`")
	}
	if args[0] == "compatibility" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project compatibility <id>`")
	}
	if args[0] == "shim-preview" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-preview <id>`")
	}
	if args[0] == "shim-readiness" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-readiness <id>`")
	}
	if args[0] == "shim-authorization" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-authorization <id>`")
	}
	if args[0] == "shim-apply-packet" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-apply-packet <id>`")
	}
	if args[0] == "shim-apply-gate" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-apply-gate <id>`")
	}
	if args[0] == "shim-apply" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-apply <id>`")
	}
	if args[0] == "shim-readiness-evidence" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project shim-readiness-evidence <id> --key <evidence-key>`")
	}
	if args[0] == "execution-cutover-readiness" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-cutover-readiness <id>`")
	}
	if args[0] == "execution-forwarding-v1-readiness" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-forwarding-v1-readiness <id>`")
	}
	if args[0] == "execution-forwarding-v1-apply-preview" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-forwarding-v1-apply-preview <id>`")
	}
	if args[0] == "execution-forwarding-v1-apply-packet" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-forwarding-v1-apply-packet <id>`")
	}
	if args[0] == "execution-forwarding-v1-apply-gate" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-forwarding-v1-apply-gate <id>`")
	}
	if args[0] == "execution-forwarding-v1-command-preview" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-forwarding-v1-command-preview <id> --task-type <type>`")
	}
	if args[0] == "execution-forwarding-v1-rollback-preview" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project execution-forwarding-v1-rollback-preview <id>`")
	}
	if args[0] == "cutover-readiness" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project cutover-readiness <id> --version <label>`")
	}
	if args[0] == "cutover-apply" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project cutover-apply <id> --version <label>`")
	}
	if args[0] == "status-projections" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project status-projections <id>`")
	}
	if args[0] == "status-projection-authorization" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project status-projection-authorization <id>`")
	}
	if args[0] == "status-projection-apply-packet" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project status-projection-apply-packet <id>`")
	}
	if args[0] == "status-projection-apply-gate" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project status-projection-apply-gate <id>`")
	}
	if args[0] == "status-projection-apply" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project status-projection-apply <id>`")
	}
	if args[0] == "import" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project import <id>`")
	}
	if args[0] == "export-status" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project export-status <id>`")
	}
	if args[0] == "doctor" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project doctor <id>`")
	}
	if args[0] == "events" && len(args) < 2 {
		return fmt.Errorf("missing project id: use `areaflow project events <id>`")
	}

	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := project.NewStore(pool)
	switch args[0] {
	case "add":
		configPath, _ := projectConfigPath(args[1:])
		projectConfig, err := project.LoadConfig(configPath)
		if err != nil {
			return err
		}
		record, err := store.UpsertFromConfig(ctx, projectConfig)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "registered %s %s\n", record.Key, record.RootPath)
		return nil
	case "status":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		c.printProject(record)
		return nil
	case "summary":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		summary, err := store.ProjectSummary(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(summaryToJSON(summary))
		}
		c.printSummary(summary)
		return nil
	case "readiness":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		readiness, err := store.ProjectReadiness(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(readinessToJSON(readiness))
		}
		c.printReadiness(readiness)
		return nil
	case "generated-write-readiness":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		readiness, err := store.GeneratedWriteReadiness(ctx, record, project.GeneratedWriteReadinessOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(generatedWriteReadinessToJSON(readiness))
		}
		c.printGeneratedWriteReadiness(readiness)
		return nil
	case "generated-write-apply-beta-gate":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		gate, err := store.GeneratedWriteApplyBetaGate(ctx, record, project.GeneratedWriteApplyBetaGateOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(generatedWriteApplyBetaGateToJSON(gate))
		}
		c.printGeneratedWriteApplyBetaGate(gate)
		return nil
	case "import-diff":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		diff, err := store.ProjectImportDiff(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(importDiffToJSON(diff))
		}
		c.printImportDiff(diff)
		return nil
	case "verify-bundle":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := bundleFlags(args[2:])
		if err != nil {
			return err
		}
		bundle, err := store.ProjectVerificationBundle(ctx, record, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(verificationBundleToJSON(bundle))
		}
		c.printVerificationBundle(bundle)
		return nil
	case "compatibility":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		contract, err := store.CompatibilityContract(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(compatibilityContractToJSON(contract))
		}
		c.printCompatibilityContract(contract)
		return nil
	case "shim-preview":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.ShimPreview(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(shimPreviewToJSON(preview))
		}
		c.printShimPreview(preview)
		return nil
	case "shim-readiness":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		readiness, err := store.ShimReadiness(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(shimReadinessToJSON(readiness))
		}
		c.printShimReadiness(readiness)
		return nil
	case "shim-authorization":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		packet, err := store.ShimAuthorizationPacket(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(shimAuthorizationPacketToJSON(packet))
		}
		c.printShimAuthorizationPacket(packet)
		return nil
	case "shim-apply-packet":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := shimApplyPacketFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.ShimApplyPacketPreview(ctx, record, project.ShimApplyPacketPreviewOptions{
			ExplicitApproval:           flags.explicitApproval,
			ApprovalID:                 flags.approvalID,
			ApprovalActor:              flags.approvalActor,
			ApprovalReason:             flags.approvalReason,
			StatusProjectionPacketID:   flags.statusProjectionPacketID,
			StatusProjectionGateID:     flags.statusProjectionGateID,
			ReadOnlySmokeEvidenceID:    flags.readOnlySmokeEvidenceID,
			DirtyWorktreeReviewID:      flags.dirtyWorktreeReviewID,
			ProtectedPathFingerprintID: flags.protectedPathFingerprintID,
			RollbackPlanID:             flags.rollbackPlanID,
			IdempotencyKey:             flags.idempotencyKey,
			AuditCorrelationID:         flags.auditCorrelationID,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(shimApplyPacketPreviewToJSON(preview))
		}
		c.printShimApplyPacketPreview(preview)
		return nil
	case "shim-apply-gate":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := shimApplyGateFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		gate, err := store.ShimApplyGate(ctx, record, project.ShimApplyGateOptions{
			AllowedFiles:               flags.allowedFiles,
			ApprovalID:                 flags.approvalID,
			ApprovalScope:              flags.approvalScope,
			AuthorizationSnapshotHash:  flags.authorizationSnapshotHash,
			ExpectedAuthorizationMode:  flags.expectedAuthorizationMode,
			StatusProjectionPacketID:   flags.statusProjectionPacketID,
			StatusProjectionGateID:     flags.statusProjectionGateID,
			ReadOnlySmokeEvidenceID:    flags.readOnlySmokeEvidenceID,
			DirtyWorktreeReviewID:      flags.dirtyWorktreeReviewID,
			ProtectedPathFingerprintID: flags.protectedPathFingerprintID,
			RollbackPlanID:             flags.rollbackPlanID,
			IdempotencyKey:             flags.idempotencyKey,
			AuditCorrelationID:         flags.auditCorrelationID,
			FailureMode:                flags.failureMode,
			ExplicitApproval:           flags.explicitApproval,
			ApprovalActor:              flags.approvalActor,
			ApprovalReason:             flags.approvalReason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(shimApplyGateToJSON(gate))
		}
		c.printShimApplyGate(gate)
		return nil
	case "shim-apply":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := shimApplyFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		result, err := store.ApplyShimCommand(ctx, record, project.ApplyShimCommandOptions{
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.approvalActor,
			Reason:         flags.approvalReason,
			Gate: project.ShimApplyGateOptions{
				AllowedFiles:               flags.allowedFiles,
				ApprovalID:                 flags.approvalID,
				ApprovalScope:              flags.approvalScope,
				AuthorizationSnapshotHash:  flags.authorizationSnapshotHash,
				ExpectedAuthorizationMode:  flags.expectedAuthorizationMode,
				StatusProjectionPacketID:   flags.statusProjectionPacketID,
				StatusProjectionGateID:     flags.statusProjectionGateID,
				ReadOnlySmokeEvidenceID:    flags.readOnlySmokeEvidenceID,
				DirtyWorktreeReviewID:      flags.dirtyWorktreeReviewID,
				ProtectedPathFingerprintID: flags.protectedPathFingerprintID,
				RollbackPlanID:             flags.rollbackPlanID,
				IdempotencyKey:             flags.idempotencyKey,
				AuditCorrelationID:         flags.auditCorrelationID,
				FailureMode:                flags.failureMode,
				ExplicitApproval:           flags.explicitApproval,
				ApprovalActor:              flags.approvalActor,
				ApprovalReason:             flags.approvalReason,
			},
		})
		if err != nil {
			return err
		}
		if flags.json {
			if err := c.printJSON(shimApplyCommandToJSON(result)); err != nil {
				return err
			}
		} else {
			c.printShimApplyCommand(result)
		}
		if result.Decision == "denied" {
			return fmt.Errorf("shim apply blocked for %s: %s", record.Key, strings.Join(result.Blockers, "; "))
		}
		return nil
	case "shim-readiness-evidence":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := shimReadinessEvidenceFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		result, err := store.RecordShimReadinessEvidence(ctx, record, project.RecordShimReadinessEvidenceOptions{
			EvidenceKey:    flags.evidenceKey,
			Status:         flags.status,
			Summary:        flags.summary,
			EvidenceURI:    flags.evidenceURI,
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(shimReadinessEvidenceToJSON(result))
		}
		c.printShimReadinessEvidence(result)
		return nil
	case "execution-cutover-readiness":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		readiness, err := store.AreaMatrixExecutionCutoverReadiness(ctx, record, project.AreaMatrixExecutionCutoverReadinessOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(executionCutoverReadinessToJSON(readiness))
		}
		c.printExecutionCutoverReadiness(readiness)
		return nil
	case "execution-forwarding-v1-readiness":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		readiness, err := store.ExecutionForwardingV1Readiness(ctx, record, project.ExecutionForwardingV1ReadinessOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(executionForwardingV1ReadinessToJSON(readiness))
		}
		c.printExecutionForwardingV1Readiness(readiness)
		return nil
	case "execution-forwarding-v1-apply-preview":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.ExecutionForwardingV1ApplyPreview(ctx, record, project.ExecutionForwardingV1ApplyPreviewOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(executionForwardingV1ApplyPreviewToJSON(preview))
		}
		c.printExecutionForwardingV1ApplyPreview(preview)
		return nil
	case "execution-forwarding-v1-apply-packet":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := executionForwardingV1ApplyPacketFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.ExecutionForwardingV1ApplyPacketPreview(ctx, record, project.ExecutionForwardingV1ApplyPacketPreviewOptions{
			ExplicitApproval:           flags.explicitApproval,
			ApprovalID:                 flags.approvalID,
			ApprovalActor:              flags.approvalActor,
			ApprovalReason:             flags.approvalReason,
			LegacyNonWriteProofID:      flags.legacyNonWriteProofID,
			RollbackPlanID:             flags.rollbackPlanID,
			ProtectedPathFingerprintID: flags.protectedPathFingerprintID,
			IdempotencyKey:             flags.idempotencyKey,
			AuditCorrelationID:         flags.auditCorrelationID,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionForwardingV1ApplyPacketPreviewToJSON(preview))
		}
		c.printExecutionForwardingV1ApplyPacketPreview(preview)
		return nil
	case "execution-forwarding-v1-apply-gate":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := executionForwardingV1ApplyGateFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		gate, err := store.ExecutionForwardingV1ApplyGate(ctx, record, project.ExecutionForwardingV1ApplyGateOptions{
			AllowedTaskTypes:           flags.allowedTaskTypes,
			ApprovalID:                 flags.approvalID,
			ApprovalScope:              flags.approvalScope,
			ReadinessSnapshotHash:      flags.readinessSnapshotHash,
			ExpectedShimLifecycleState: flags.expectedShimLifecycleState,
			LegacyNonWriteProofID:      flags.legacyNonWriteProofID,
			RollbackPlanID:             flags.rollbackPlanID,
			ProtectedPathFingerprintID: flags.protectedPathFingerprintID,
			IdempotencyKey:             flags.idempotencyKey,
			AuditCorrelationID:         flags.auditCorrelationID,
			FailureMode:                flags.failureMode,
			ExplicitApproval:           flags.explicitApproval,
			ApprovalActor:              flags.approvalActor,
			ApprovalReason:             flags.approvalReason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionForwardingV1ApplyGateToJSON(gate))
		}
		c.printExecutionForwardingV1ApplyGate(gate)
		return nil
	case "execution-forwarding-v1-apply":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := executionForwardingV1ApplyFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		result, err := store.ApplyExecutionForwardingV1(ctx, record, project.ApplyExecutionForwardingV1Options{
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.approvalActor,
			Reason:         flags.approvalReason,
			Gate: project.ExecutionForwardingV1ApplyGateOptions{
				AllowedTaskTypes:           flags.allowedTaskTypes,
				ApprovalID:                 flags.approvalID,
				ApprovalScope:              flags.approvalScope,
				ReadinessSnapshotHash:      flags.readinessSnapshotHash,
				ExpectedShimLifecycleState: flags.expectedShimLifecycleState,
				LegacyNonWriteProofID:      flags.legacyNonWriteProofID,
				RollbackPlanID:             flags.rollbackPlanID,
				ProtectedPathFingerprintID: flags.protectedPathFingerprintID,
				IdempotencyKey:             flags.idempotencyKey,
				AuditCorrelationID:         flags.auditCorrelationID,
				FailureMode:                flags.failureMode,
				ExplicitApproval:           flags.explicitApproval,
				ApprovalActor:              flags.approvalActor,
				ApprovalReason:             flags.approvalReason,
			},
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionForwardingV1ApplyToJSON(result))
		}
		c.printExecutionForwardingV1Apply(result)
		if result.Decision == "denied" {
			return fmt.Errorf("execution forwarding v1 apply blocked for %s: %s", record.Key, strings.Join(result.Blockers, "; "))
		}
		return nil
	case "execution-forwarding-v1-command-preview":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := executionForwardingV1CommandPreviewFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.ExecutionForwardingV1CommandPreview(ctx, record, project.ExecutionForwardingV1CommandPreviewOptions{
			TaskType: flags.taskType,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionForwardingV1CommandPreviewToJSON(preview))
		}
		c.printExecutionForwardingV1CommandPreview(preview)
		return nil
	case "execution-forwarding-v1-rollback-preview":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.ExecutionForwardingV1RollbackPreview(ctx, record, project.ExecutionForwardingV1RollbackPreviewOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(executionForwardingV1RollbackPreviewToJSON(preview))
		}
		c.printExecutionForwardingV1RollbackPreview(preview)
		return nil
	case "cutover-readiness":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := cutoverReadinessFlags(args[2:])
		if err != nil {
			return err
		}
		readiness, err := store.ProjectCutoverReadiness(ctx, record, flags.version, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(cutoverReadinessToJSON(readiness))
		}
		c.printCutoverReadiness(readiness)
		return nil
	case "cutover-apply":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := cutoverApplyFlags(args[2:])
		if err != nil {
			return err
		}
		result, err := store.ApplyCutover(ctx, record, project.ApplyCutoverOptions{
			VersionLabel:   flags.version,
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
			Mode:           flags.mode,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(cutoverApplyToJSON(result))
		}
		c.printCutoverApply(result)
		if result.Decision == "denied" {
			return fmt.Errorf("cutover apply blocked for %s/%s", record.Key, flags.version)
		}
		return nil
	case "status-projections":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := statusProjectionFlags(args[2:])
		if err != nil {
			return err
		}
		projections, err := store.ListStatusProjections(ctx, record, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(statusProjectionsToJSON(record, projections))
		}
		c.printStatusProjections(record, projections)
		return nil
	case "status-projection-authorization":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := statusProjectionAuthorizationFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.StatusProjectionAuthorizationPreview(ctx, record, project.StatusProjectionAuthorizationPreviewOptions{
			TargetURI: flags.targetURI,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(statusProjectionAuthorizationPreviewToJSON(preview))
		}
		c.printStatusProjectionAuthorizationPreview(preview)
		return nil
	case "status-projection-apply-packet":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := statusProjectionApplyPacketFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		preview, err := store.StatusProjectionApplyPacketPreview(ctx, record, project.StatusProjectionApplyPacketPreviewOptions{
			TargetURI:        flags.targetURI,
			ExplicitApproval: flags.explicitApproval,
			ApprovalActor:    flags.approvalActor,
			ApprovalReason:   flags.approvalReason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(statusProjectionApplyPacketPreviewToJSON(preview))
		}
		c.printStatusProjectionApplyPacketPreview(preview)
		return nil
	case "status-projection-apply-gate":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := statusProjectionApplyGateFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		gate, err := store.StatusProjectionApplyGate(ctx, record, project.StatusProjectionApplyGateOptions{
			TargetURI:                      flags.targetURI,
			ExpectedBeforeExists:           flags.expectedBeforeExists,
			ExpectedBeforeSHA256:           flags.expectedBeforeSHA256,
			ExpectedBeforeSizeBytes:        flags.expectedBeforeSizeBytes,
			SourceHash:                     flags.sourceHash,
			SchemaURI:                      flags.schemaURI,
			ValidatorPreflight:             flags.validatorPreflight,
			ProtectedPathCheck:             flags.protectedPathCheck,
			ProtectedPathFingerprintSHA256: flags.protectedPathFingerprintSHA256,
			RollbackAction:                 flags.rollbackAction,
			AcceptedPreimageSchemaStatus:   flags.acceptedPreimageSchemaStatus,
			ExplicitApproval:               flags.explicitApproval,
			ApprovalActor:                  flags.approvalActor,
			ApprovalReason:                 flags.approvalReason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(statusProjectionApplyGateToJSON(gate))
		}
		c.printStatusProjectionApplyGate(gate)
		return nil
	case "status-projection-apply":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := statusProjectionApplyFlagsFromArgs(args[2:])
		if err != nil {
			return err
		}
		result, err := store.ApplyStatusProjection(ctx, record, project.ApplyStatusProjectionOptions{
			TargetURI:      flags.targetURI,
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
			Writer:         appStatusProjectionWriter,
			Gate: project.StatusProjectionApplyGateOptions{
				TargetURI:                      flags.targetURI,
				ExpectedBeforeExists:           flags.expectedBeforeExists,
				ExpectedBeforeSHA256:           flags.expectedBeforeSHA256,
				ExpectedBeforeSizeBytes:        flags.expectedBeforeSizeBytes,
				SourceHash:                     flags.sourceHash,
				SchemaURI:                      flags.schemaURI,
				ValidatorPreflight:             flags.validatorPreflight,
				ProtectedPathCheck:             flags.protectedPathCheck,
				ProtectedPathFingerprintSHA256: flags.protectedPathFingerprintSHA256,
				RollbackAction:                 flags.rollbackAction,
				AcceptedPreimageSchemaStatus:   flags.acceptedPreimageSchemaStatus,
				ExplicitApproval:               flags.explicitApproval,
				ApprovalActor:                  flags.approvalActor,
				ApprovalReason:                 flags.approvalReason,
			},
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(statusProjectionApplyToJSON(result))
		}
		c.printStatusProjectionApply(result)
		if result.Decision == "denied" {
			return fmt.Errorf("status projection apply blocked for %s: %s", record.Key, strings.Join(result.Blockers, "; "))
		}
		return nil
	case "import":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := importFlags(args[2:])
		if err != nil {
			return err
		}
		result, err := importer.ImportProject(ctx, pool, record, importer.Options{
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(importResultToJSON(result))
		}
		fmt.Fprintf(c.stdout, "imported %s versions=%d residuals=%d artifacts=%d active_tasks=%d v1=%d/%d snapshot=%s run_id=%d idempotency_key=%s created=%t\n",
			result.ProjectKey,
			result.Versions,
			result.Residuals,
			result.Artifacts,
			result.ActiveTasks,
			result.V1Done,
			result.V1Total,
			result.StatusSnapshot,
			result.RunID,
			result.IdempotencyKey,
			result.Created,
		)
		return nil
	case "export-status":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.ApplyStatusProjection(ctx, record, project.ApplyStatusProjectionOptions{
			TargetURI: ".areaflow/status.json",
			Actor:     "local-user",
			Reason:    "legacy export-status compatibility command",
			Writer:    appStatusProjectionWriter,
		})
		if err != nil {
			return err
		}
		if result.Decision == "denied" {
			return fmt.Errorf("status export denied for %s: %s", result.TargetURI, strings.Join(result.Blockers, "; "))
		}
		fmt.Fprintf(c.stdout, "exported %s\n", result.WrittenTarget)
		return nil
	case "doctor":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		flags, err := doctorFlags(args[2:])
		if err != nil {
			return err
		}
		var report doctor.Report
		if flags.allowNative {
			report, err = doctor.AreaMatrixWithNative(ctx, record, store)
		} else {
			report, err = doctor.AreaMatrix(ctx, record, store)
		}
		if err != nil {
			return err
		}
		if flags.json {
			if err := c.printJSON(report.Summary()); err != nil {
				return err
			}
		} else {
			c.printDoctorReport(report)
		}
		if _, err := store.RecordDoctorReport(ctx, record.ID, report.Summary(), project.RecordDoctorReportOptions{
			IdempotencyKey: flags.idempotencyKey,
			Reason:         "project doctor CLI run",
		}); err != nil {
			return err
		}
		if report.HasFailures() {
			return fmt.Errorf("doctor failed for %s", record.Key)
		}
		return nil
	case "events":
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		limit, err := eventLimit(args[2:])
		if err != nil {
			return err
		}
		events, err := store.ListEvents(ctx, record.ID, limit)
		if err != nil {
			return err
		}
		c.printEvents(record, events)
		return nil
	case "list":
		records, err := store.List(ctx)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			fmt.Fprintln(c.stdout, "no projects registered")
			return nil
		}
		for _, record := range records {
			fmt.Fprintf(c.stdout, "%s\t%s\t%s\n", record.Key, record.Adapter, record.RootPath)
		}
		return nil
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func (c command) runWorkflow(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing workflow command: use `areaflow workflow version ...`, `profile ...`, `gate ...`, `transition ...`, or `approval ...`")
	}
	switch args[0] {
	case "version":
		return c.runWorkflowVersion(ctx, args[1:])
	case "profile":
		return c.runWorkflowProfile(ctx, args[1:])
	case "gate":
		return c.runWorkflowGate(ctx, args[1:])
	case "transition":
		return c.runWorkflowTransition(ctx, args[1:])
	case "approval":
		return c.runWorkflowApproval(ctx, args[1:])
	default:
		return fmt.Errorf("unknown workflow command %q", args[0])
	}
}

func (c command) runWorkflowProfile(ctx context.Context, args []string) error {
	_ = ctx
	if len(args) == 0 {
		return fmt.Errorf("missing workflow profile command: use `list`, `show <profile>`, or `check <profile>`")
	}
	root, err := workflowProfileRoot()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		flags, err := workflowProfileFlags(args[1:])
		if err != nil {
			return err
		}
		profiles, err := workflowprofile.ListBuiltInProfiles(root)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workflowProfileListToJSON(profiles))
		}
		c.printWorkflowProfileList(profiles)
		return nil
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow workflow profile show <profile> [--json]")
		}
		flags, err := workflowProfileFlags(args[2:])
		if err != nil {
			return err
		}
		loaded, err := workflowprofile.LoadBuiltInProfile(root, args[1])
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workflowProfileToJSON(loaded))
		}
		c.printWorkflowProfile(loaded)
		return nil
	case "check":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow workflow profile check <profile> [--json]")
		}
		flags, err := workflowProfileFlags(args[2:])
		if err != nil {
			return err
		}
		loaded, err := workflowprofile.LoadBuiltInProfile(root, args[1])
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workflowProfileCheckToJSON(loaded))
		}
		c.printWorkflowProfileCheck(loaded)
		return nil
	default:
		return fmt.Errorf("unknown workflow profile command %q", args[0])
	}
}

func (c command) runRun(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing run command: use `areaflow run preview <project> <version>`, `areaflow run fixture-queue <project> <version>`, `areaflow run read-only-verify-queue <project> <version> --target-path PATH`, `areaflow run approved-artifact-write-queue <project> <version>`, `areaflow run fixture-project-write-queue <project> <version> --target-path PATH --content TEXT --expected-before-sha256 HASH --expected-before-size N`, `areaflow run managed-generated-write-queue <project> <version> --target-path PATH --content TEXT --expected-before-sha256 HASH --expected-before-size N`, `areaflow run execution-gate <run-id>`, `areaflow run execution-plan <run-id>`, `areaflow run project-write-design-gate <run-id>`, `areaflow run managed-generated-write-gate <run-id>`, or `areaflow run start|drain|cancel <run-id>`")
	}
	switch args[0] {
	case "preview":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
		}
		flags, err := runnerPreviewFlags(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.PreviewRunner(ctx, record, args[2], project.RunnerPreviewOptions{
			Actor:          flags.actor,
			Reason:         flags.reason,
			RiskLevel:      flags.riskLevel,
			RiskPolicy:     flags.riskPolicy,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(runnerPreviewToJSON(result))
		}
		c.printRunnerPreview(result)
		return nil
	case "fixture-queue":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow run fixture-queue <project> <version> [--json] [--actor A] [--reason R] [--idempotency-key K]")
		}
		flags, err := fixtureExecutionQueueFlags(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.QueueFixtureExecution(ctx, record, args[2], project.FixtureExecutionQueueOptions{
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(fixtureExecutionQueueToJSON(result))
		}
		c.printFixtureExecutionQueue(result)
		return nil
	case "read-only-verify-queue":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow run read-only-verify-queue <project> <version> --target-path PATH [--json] [--actor A] [--reason R] [--idempotency-key K]")
		}
		flags, err := readOnlyVerifyQueueFlags(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.QueueReadOnlyVerify(ctx, record, args[2], project.ReadOnlyVerifyQueueOptions{
			TargetPath:     flags.targetPath,
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(readOnlyVerifyQueueToJSON(result))
		}
		c.printReadOnlyVerifyQueue(result)
		return nil
	case "approved-artifact-write-queue":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow run approved-artifact-write-queue <project> <version> [--artifact-label LABEL] [--json] [--actor A] [--reason R] [--idempotency-key K]")
		}
		flags, err := approvedArtifactWriteQueueFlags(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.QueueApprovedArtifactWrite(ctx, record, args[2], project.ApprovedArtifactWriteQueueOptions{
			ArtifactLabel:  flags.artifactLabel,
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(approvedArtifactWriteQueueToJSON(result))
		}
		c.printApprovedArtifactWriteQueue(result)
		return nil
	case "fixture-project-write-queue":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow run fixture-project-write-queue <project> <version> --target-path PATH --content TEXT --expected-before-sha256 HASH --expected-before-size N [--json] [--actor A] [--reason R] [--idempotency-key K]")
		}
		flags, err := fixtureProjectWriteQueueFlags(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.QueueFixtureProjectWrite(ctx, record, args[2], project.FixtureProjectWriteQueueOptions{
			TargetPath:           flags.targetPath,
			Content:              flags.content,
			ExpectedBeforeSHA256: flags.expectedBeforeSHA256,
			ExpectedBeforeSize:   flags.expectedBeforeSize,
			Actor:                flags.actor,
			Reason:               flags.reason,
			IdempotencyKey:       flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(fixtureProjectWriteQueueToJSON(result))
		}
		c.printFixtureProjectWriteQueue(result)
		return nil
	case "managed-generated-write-queue":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow run managed-generated-write-queue <project> <version> --target-path PATH --content TEXT --expected-before-sha256 HASH --expected-before-size N [--json] [--actor A] [--reason R] [--idempotency-key K]")
		}
		flags, err := managedGeneratedWriteQueueFlags(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.QueueManagedGeneratedWrite(ctx, record, args[2], project.ManagedGeneratedWriteQueueOptions{
			TargetPath:           flags.targetPath,
			Content:              flags.content,
			ExpectedBeforeSHA256: flags.expectedBeforeSHA256,
			ExpectedBeforeSize:   flags.expectedBeforeSize,
			Actor:                flags.actor,
			Reason:               flags.reason,
			IdempotencyKey:       flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(managedGeneratedWriteQueueToJSON(result))
		}
		c.printManagedGeneratedWriteQueue(result)
		return nil
	case "execution-gate":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow run execution-gate <run-id> [--json] [--capability CAP]")
		}
		runID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("run id must be a positive integer")
		}
		flags, err := executionGateFlags(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.ExecutionApprovalGate(ctx, runID, project.ExecutionApprovalGateOptions{
			RequiredCapabilities: flags.capabilities,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionApprovalGateToJSON(gate))
		}
		c.printExecutionApprovalGate(gate)
		return nil
	case "execution-plan":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow run execution-plan <run-id> [--json]")
		}
		runID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("run id must be a positive integer")
		}
		flags, err := executionPlanFlags(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.PreviewExecutionPlan(ctx, runID)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionPlanPreviewToJSON(preview))
		}
		c.printExecutionPlanPreview(preview)
		return nil
	case "project-write-design-gate":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow run project-write-design-gate <run-id> [--json]")
		}
		runID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("run id must be a positive integer")
		}
		flags, err := projectWriteDesignGateFlags(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.PreviewProjectWriteDesignGate(ctx, runID)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(projectWriteDesignGateToJSON(gate))
		}
		c.printProjectWriteDesignGate(gate)
		return nil
	case "managed-generated-write-gate":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow run managed-generated-write-gate <run-id> [--json]")
		}
		runID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("run id must be a positive integer")
		}
		flags, err := managedGeneratedWriteGateFlags(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.PreviewManagedGeneratedWriteGate(ctx, runID)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(managedGeneratedWriteGateToJSON(gate))
		}
		c.printManagedGeneratedWriteGate(gate)
		return nil
	case "start", "drain", "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow run %s <run-id> [--json] [--actor ACTOR] [--reason TEXT] [--idempotency-key KEY]", args[0])
		}
		runID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("run id must be a positive integer")
		}
		flags, err := runControlFlags(args[2:], args[0])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		result, err := store.ControlRun(ctx, runID, project.RunControlOptions{
			Action:         args[0],
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(runControlToJSON(result))
		}
		c.printRunControl(result)
		return nil
	default:
		return fmt.Errorf("unknown run command %q", args[0])
	}
}

func (c command) runService(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing service command: use `areaflow service status`")
	}
	switch args[0] {
	case "status":
		flags, err := serviceStatusFlags(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		status, err := store.LocalServiceStatus(ctx, project.LocalServiceStatusOptions{
			APIBaseURL:      "http://" + cfg.Server.Addr() + "/api/v1",
			WebDashboardURL: flags.webURL,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(localServiceStatusToJSON(status))
		}
		c.printLocalServiceStatus(status)
		return nil
	default:
		return fmt.Errorf("unknown service command %q", args[0])
	}
}

func (c command) runDesktop(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing desktop command: use `areaflow desktop service-control-gate`")
	}
	jsonOutput, err := outputJSON(args[1:])
	if err != nil {
		return err
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := project.NewStore(pool)
	switch args[0] {
	case "service-control-gate":
		gate, err := store.DesktopServiceControlGate(ctx, project.DesktopServiceControlGateOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(desktopServiceControlGateToJSON(gate))
		}
		c.printDesktopServiceControlGate(gate)
		return nil
	case "notification-gate":
		gate, err := store.DesktopNotificationGate(ctx, project.DesktopNotificationGateOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(desktopNotificationGateToJSON(gate))
		}
		c.printDesktopNotificationGate(gate)
		return nil
	case "tray-menu-gate":
		gate, err := store.DesktopTrayMenuGate(ctx, project.DesktopTrayMenuGateOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(desktopTrayMenuGateToJSON(gate))
		}
		c.printDesktopTrayMenuGate(gate)
		return nil
	default:
		return fmt.Errorf("unknown desktop command %q", args[0])
	}
}

func (c command) runSecurity(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing security command: use `areaflow security boundary-readiness`")
	}
	switch args[0] {
	case "boundary-readiness":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		readiness, err := store.SecurityBoundaryReadiness(ctx, project.SecurityBoundaryReadinessOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(securityBoundaryReadinessToJSON(readiness))
		}
		c.printSecurityBoundaryReadiness(readiness)
		return nil
	default:
		return fmt.Errorf("unknown security command %q", args[0])
	}
}

func (c command) runCompletion(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing completion command: use `areaflow completion audit`")
	}
	switch args[0] {
	case "audit":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		audit, err := store.CompletionAudit(ctx, project.CompletionAuditOptions{
			APIBaseURL: "http://" + cfg.Server.Addr() + "/api/v1",
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(completionAuditToJSON(audit))
		}
		c.printCompletionAudit(audit)
		return nil
	case "audit-snapshot":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow completion audit-snapshot record|readiness <project>")
		}
		projectKey := args[2]
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		switch args[1] {
		case "record":
			flags, err := completionAuditSnapshotFlagsFromArgs(args[3:])
			if err != nil {
				return err
			}
			result, err := store.RecordCompletionAuditSnapshot(ctx, record, project.RecordCompletionAuditSnapshotOptions{
				ReleaseCandidateLabel: flags.releaseCandidateLabel,
				EvidenceClass:         flags.evidenceClass,
				EvidenceURI:           flags.evidenceURI,
				Summary:               flags.summary,
				ReviewDecision:        flags.reviewDecision,
				ReviewedBy:            flags.reviewedBy,
				ReviewedAt:            flags.reviewedAt,
				IdempotencyKey:        flags.idempotencyKey,
				Actor:                 flags.actor,
				Reason:                flags.reason,
			})
			if err != nil {
				return err
			}
			if flags.json {
				return c.printJSON(completionAuditSnapshotToJSON(result))
			}
			c.printCompletionAuditSnapshot(result)
			return nil
		case "readiness":
			flags, err := completionAuditSnapshotReadinessFlagsFromArgs(args[3:])
			if err != nil {
				return err
			}
			readiness, err := store.CompletionAuditSnapshotReadiness(ctx, record)
			if err != nil {
				return err
			}
			if flags.json {
				return c.printJSON(completionAuditSnapshotReadinessToJSON(readiness))
			}
			c.printCompletionAuditSnapshotReadiness(readiness)
			return nil
		default:
			return fmt.Errorf("usage: areaflow completion audit-snapshot record|readiness <project>")
		}
	case "protected-path-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion protected-path-proof record <project> --status clean|authorized|dirty|blocked [--summary TEXT] [--evidence-uri URI] [--git-status-output TEXT] [--approval-id ID] [--allowed-path PATH...] [--dirty-output-hash SHA256] [--reviewer TEXT] [--rollback-evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary and evidence-uri required for clean|authorized; approval-id, allowed-path, git-status-output, dirty-output-hash, reviewer and rollback-evidence-uri required for authorized)")
		}
		projectKey := args[2]
		flags, err := protectedPathProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordProtectedPathProof(ctx, record, project.RecordProtectedPathProofOptions{
			ProofStatus:                flags.status,
			Summary:                    flags.summary,
			EvidenceURI:                flags.evidenceURI,
			GitStatusOutput:            flags.gitStatusOutput,
			AuthorizedApprovalID:       flags.approvalID,
			AuthorizedAllowedPaths:     flags.allowedPaths,
			AuthorizedDirtyOutputHash:  flags.dirtyOutputHash,
			AuthorizedReviewer:         flags.reviewer,
			AuthorizedRollbackEvidence: flags.rollbackEvidenceURI,
			IdempotencyKey:             flags.idempotencyKey,
			Actor:                      flags.actor,
			Reason:                     flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(protectedPathProofToJSON(result))
		}
		c.printProtectedPathProof(result)
		return nil
	case "archive-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion archive-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--archive-scope SCOPE] [--archive-reference-mode MODE] [--archive-source-path PATH...] [--archive-forbidden-action ACTION...] [--archive-rollback-target TARGET] [--archive-fail-closed] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, release-candidate evidence-uri, approved review metadata and archive scope binding required for complete)")
		}
		projectKey := args[2]
		flags, err := archiveProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordArchiveProof(ctx, record, project.RecordArchiveProofOptions{
			ProofStatus:             flags.status,
			Facts:                   flags.facts,
			Summary:                 flags.summary,
			EvidenceURI:             flags.evidenceURI,
			ReviewDecision:          flags.reviewDecision,
			ReviewedBy:              flags.reviewedBy,
			ReviewedAt:              flags.reviewedAt,
			ArchiveScope:            flags.archiveScope,
			ArchiveReferenceMode:    flags.archiveReferenceMode,
			ArchiveSourcePaths:      flags.archiveSourcePaths,
			ArchiveForbiddenActions: flags.archiveForbiddenActions,
			ArchiveRollbackTarget:   flags.archiveRollbackTarget,
			ArchiveFailClosed:       flags.archiveFailClosed,
			IdempotencyKey:          flags.idempotencyKey,
			Actor:                   flags.actor,
			Reason:                  flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(archiveProofToJSON(result))
		}
		c.printArchiveProof(result)
		return nil
	case "shim-retirement-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion shim-retirement-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--shim-retirement-scope SCOPE] [--shim-prerequisite KEY...] [--shim-retired-surface SURFACE...] [--shim-rollback-target TARGET] [--shim-fail-closed] [--shim-reopen-requires-approval] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, release-candidate evidence-uri, approved review metadata and shim retirement scope binding required for complete)")
		}
		projectKey := args[2]
		flags, err := shimRetirementProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordShimRetirementProof(ctx, record, project.RecordShimRetirementProofOptions{
			ProofStatus:                 flags.status,
			Facts:                       flags.facts,
			Summary:                     flags.summary,
			EvidenceURI:                 flags.evidenceURI,
			ReviewDecision:              flags.reviewDecision,
			ReviewedBy:                  flags.reviewedBy,
			ReviewedAt:                  flags.reviewedAt,
			ShimRetirementScope:         flags.shimRetirementScope,
			ShimRetirementPrerequisites: flags.shimRetirementPrerequisites,
			ShimRetiredSurfaces:         flags.shimRetiredSurfaces,
			ShimRollbackTarget:          flags.shimRollbackTarget,
			ShimFailClosed:              flags.shimFailClosed,
			ShimReopenRequiresApproval:  flags.shimReopenRequiresApproval,
			IdempotencyKey:              flags.idempotencyKey,
			Actor:                       flags.actor,
			Reason:                      flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(shimRetirementProofToJSON(result))
		}
		c.printShimRetirementProof(result)
		return nil
	case "execution-cutover-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion execution-cutover-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--execution-cutover-scope SCOPE] [--allowed-task-type TYPE...] [--allowed-task-types a,b] [--forbidden-action ACTION...] [--forbidden-actions a,b] [--rollback-target TARGET] [--rollback-mode MODE] [--fail-closed] [--reopen-requires-approval] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, release-candidate evidence-uri, approved review metadata and scope binding required for complete)")
		}
		projectKey := args[2]
		flags, err := executionCutoverProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordExecutionCutoverProof(ctx, record, project.RecordExecutionCutoverProofOptions{
			ProofStatus:                flags.status,
			Facts:                      flags.facts,
			Summary:                    flags.summary,
			EvidenceURI:                flags.evidenceURI,
			ReviewDecision:             flags.reviewDecision,
			ReviewedBy:                 flags.reviewedBy,
			ReviewedAt:                 flags.reviewedAt,
			ExecutionCutoverScope:      flags.executionCutoverScope,
			AllowedTaskTypes:           flags.allowedTaskTypes,
			ForbiddenActions:           flags.forbiddenActions,
			RollbackTarget:             flags.rollbackTarget,
			RollbackMode:               flags.rollbackMode,
			FailClosed:                 flags.failClosed,
			ReopenRequiresApproval:     flags.reopenRequiresApproval,
			SourceWriteOpen:            flags.sourceWriteOpen,
			GeneratedRetainedWriteOpen: flags.generatedRetainedWriteOpen,
			RepairApplyOpen:            flags.repairApplyOpen,
			CheckpointApplyOpen:        flags.checkpointApplyOpen,
			EngineExecutionOpen:        flags.engineExecutionOpen,
			SecretResolveOpen:          flags.secretResolveOpen,
			NetworkAPIIntegrationOpen:  flags.networkAPIIntegrationOpen,
			PublishApplyOpen:           flags.publishApplyOpen,
			RestoreApplyOpen:           flags.restoreApplyOpen,
			IdempotencyKey:             flags.idempotencyKey,
			Actor:                      flags.actor,
			Reason:                     flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(executionCutoverProofToJSON(result))
		}
		c.printExecutionCutoverProof(result)
		return nil
	case "validation-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion validation-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--validation-command CMD...] [--validation-result-hash SHA256] [--validation-started-at RFC3339] [--validation-finished-at RFC3339] [--validation-scope SCOPE] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and validation binding fields required for complete)")
		}
		projectKey := args[2]
		flags, err := validationProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordValidationProof(ctx, record, project.RecordValidationProofOptions{
			ProofStatus:          flags.status,
			Facts:                flags.facts,
			Summary:              flags.summary,
			EvidenceURI:          flags.evidenceURI,
			ValidationCommands:   flags.validationCommands,
			ValidationResultHash: flags.validationResultHash,
			ValidationStartedAt:  flags.validationStartedAt,
			ValidationFinishedAt: flags.validationFinishedAt,
			ValidationScope:      flags.validationScope,
			IdempotencyKey:       flags.idempotencyKey,
			Actor:                flags.actor,
			Reason:               flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(validationProofToJSON(result))
		}
		c.printValidationProof(result)
		return nil
	case "source-alignment-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion source-alignment-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and current source binding required for complete; binding is collected automatically)")
		}
		projectKey := args[2]
		flags, err := sourceAlignmentProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		var sourceAlignmentBinding map[string]any
		if flags.status == "complete" {
			sourceAlignmentBinding, err = project.SourceAlignmentCurrentBinding()
			if err != nil {
				return err
			}
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordSourceAlignmentProof(ctx, record, project.RecordSourceAlignmentProofOptions{
			ProofStatus:            flags.status,
			Facts:                  flags.facts,
			Summary:                flags.summary,
			EvidenceURI:            flags.evidenceURI,
			SourceAlignmentBinding: sourceAlignmentBinding,
			IdempotencyKey:         flags.idempotencyKey,
			Actor:                  flags.actor,
			Reason:                 flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(sourceAlignmentProofToJSON(result))
		}
		c.printSourceAlignmentProof(result)
		return nil
	case "task-matrix-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion task-matrix-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--source-set-hash SHA256] [--backlog-hash SHA256] [--task-status-audit-hash SHA256] [--planned-v1-required-task-count N] [--missing-evidence-v1-required-task-count N] [--blocked-v1-required-task-count N] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and binding required for complete)")
		}
		projectKey := args[2]
		flags, err := taskMatrixProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordTaskMatrixProof(ctx, record, project.RecordTaskMatrixProofOptions{
			ProofStatus:                           flags.status,
			Facts:                                 flags.facts,
			Summary:                               flags.summary,
			EvidenceURI:                           flags.evidenceURI,
			TaskMatrixSourceSetHash:               flags.taskMatrixSourceSetHash,
			TaskBacklogHash:                       flags.taskBacklogHash,
			TaskStatusAuditHash:                   flags.taskStatusAuditHash,
			PlannedV1RequiredTaskCount:            flags.plannedV1RequiredTaskCount,
			PlannedV1RequiredTaskCountSet:         flags.plannedV1RequiredTaskCountSet,
			MissingEvidenceV1RequiredTaskCount:    flags.missingEvidenceV1RequiredTaskCount,
			MissingEvidenceV1RequiredTaskCountSet: flags.missingEvidenceV1RequiredTaskCountSet,
			BlockedV1RequiredTaskCount:            flags.blockedV1RequiredTaskCount,
			BlockedV1RequiredTaskCountSet:         flags.blockedV1RequiredTaskCountSet,
			IdempotencyKey:                        flags.idempotencyKey,
			Actor:                                 flags.actor,
			Reason:                                flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(taskMatrixProofToJSON(result))
		}
		c.printTaskMatrixProof(result)
		return nil
	case "security-closure-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion security-closure-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and current binding required for complete)")
		}
		projectKey := args[2]
		flags, err := securityClosureProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		securityClosureBinding := map[string]any{}
		if flags.status == "complete" {
			binding, err := store.SecurityClosureCurrentBinding(ctx, record, project.SecurityClosureCurrentBindingOptions{})
			if err != nil {
				return err
			}
			securityClosureBinding = binding.Metadata
		}
		result, err := store.RecordSecurityClosureProof(ctx, record, project.RecordSecurityClosureProofOptions{
			ProofStatus:            flags.status,
			Facts:                  flags.facts,
			Summary:                flags.summary,
			EvidenceURI:            flags.evidenceURI,
			SecurityClosureBinding: securityClosureBinding,
			IdempotencyKey:         flags.idempotencyKey,
			Actor:                  flags.actor,
			Reason:                 flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(securityClosureProofToJSON(result))
		}
		c.printSecurityClosureProof(result)
		return nil
	case "backup-restore-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion backup-restore-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] --backup-manifest-hash HASH --backup-manifest-status ready --backup-manifest-project-count N --backup-manifest-table-count N --restore-plan-status ready|needs_attention --restore-plan-scope project --restore-plan-project-key KEY --restore-plan-manifest-hash HASH --restore-plan-item-count N --artifact-integrity-status pass|warn --artifact-integrity-checked-count N --artifact-integrity-failed-count 0 --artifact-archive-preview-status ready|needs_attention --artifact-archive-preview-total-artifacts N --artifact-archive-preview-external-refs N --artifact-archive-preview-needs-policy 0 --artifact-archive-preview-project-write-attempted false --artifact-archive-preview-storage-write-attempted false --artifact-archive-preview-delete-attempted false [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and output binding required for complete)")
		}
		projectKey := args[2]
		flags, err := backupRestoreProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordBackupRestoreProof(ctx, record, project.RecordBackupRestoreProofOptions{
			ProofStatus:                                 flags.status,
			Facts:                                       flags.facts,
			Summary:                                     flags.summary,
			EvidenceURI:                                 flags.evidenceURI,
			BackupManifestHash:                          flags.backupManifestHash,
			BackupManifestStatus:                        flags.backupManifestStatus,
			BackupManifestProjectCount:                  flags.backupManifestProjectCount,
			BackupManifestTableCount:                    flags.backupManifestTableCount,
			RestorePlanStatus:                           flags.restorePlanStatus,
			RestorePlanScope:                            flags.restorePlanScope,
			RestorePlanProjectKey:                       flags.restorePlanProjectKey,
			RestorePlanManifestHash:                     flags.restorePlanManifestHash,
			RestorePlanItemCount:                        flags.restorePlanItemCount,
			ArtifactIntegrityStatus:                     flags.artifactIntegrityStatus,
			ArtifactIntegrityCheckedCount:               flags.artifactIntegrityCheckedCount,
			ArtifactIntegrityFailedCount:                flags.artifactIntegrityFailedCount,
			ArtifactArchivePreviewStatus:                flags.artifactArchivePreviewStatus,
			ArtifactArchivePreviewTotalArtifacts:        flags.artifactArchivePreviewTotalArtifacts,
			ArtifactArchivePreviewExternalRefs:          flags.artifactArchivePreviewExternalRefs,
			ArtifactArchivePreviewNeedsPolicy:           flags.artifactArchivePreviewNeedsPolicy,
			ArtifactArchivePreviewProjectWriteAttempted: flags.artifactArchivePreviewProjectWriteAttempted,
			ArtifactArchivePreviewStorageWriteAttempted: flags.artifactArchivePreviewStorageWriteAttempted,
			ArtifactArchivePreviewDeleteAttempted:       flags.artifactArchivePreviewDeleteAttempted,
			IdempotencyKey:                              flags.idempotencyKey,
			Actor:                                       flags.actor,
			Reason:                                      flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(backupRestoreProofToJSON(result))
		}
		c.printBackupRestoreProof(result)
		return nil
	case "release-packaging-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow completion release-packaging-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary and evidence-uri required for complete)")
		}
		projectKey := args[2]
		flags, err := releasePackagingProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		var metadata map[string]any
		if flags.status == "complete" {
			bundle, err := store.ReleaseEvidenceBundle(ctx, project.ReleaseEvidenceBundleOptions{ProjectID: record.ID, ProjectKey: record.Key})
			if err != nil {
				return err
			}
			metadata = project.ReleaseEvidenceBundleBindingMetadata(bundle)
		}
		result, err := store.RecordReleasePackagingProof(ctx, record, project.RecordReleasePackagingProofOptions{
			ProofStatus:    flags.status,
			Facts:          flags.facts,
			Summary:        flags.summary,
			EvidenceURI:    flags.evidenceURI,
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
			Metadata:       metadata,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releasePackagingProofToJSON(result))
		}
		c.printReleasePackagingProof(result)
		return nil
	default:
		return fmt.Errorf("unknown completion command %q", args[0])
	}
}

func (c command) runOps(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing ops command: use `areaflow ops readiness`, `areaflow ops migration-ledger-readiness`, or `areaflow ops smoke-proof record <project>`")
	}
	switch args[0] {
	case "readiness":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		readiness, err := store.OperationsReadiness(ctx, project.OperationsReadinessOptions{
			APIBaseURL: "http://" + cfg.Server.Addr() + "/api/v1",
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(operationsReadinessToJSON(readiness))
		}
		c.printOperationsReadiness(readiness)
		return nil
	case "migration-ledger-readiness":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		readiness, err := store.MigrationLedgerReadiness(ctx, project.MigrationLedgerReadinessOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(migrationLedgerReadinessToJSON(readiness))
		}
		c.printMigrationLedgerReadiness(readiness)
		return nil
	case "smoke-proof":
		if len(args) < 3 || args[1] != "record" {
			return fmt.Errorf("usage: areaflow ops smoke-proof record <project> --key <proof-key> [--status pass|blocked] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary and evidence-uri required for pass)")
		}
		projectKey := args[2]
		flags, err := operationsSmokeProofFlagsFromArgs(args[3:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, projectKey)
		if err != nil {
			return err
		}
		result, err := store.RecordOperationsSmokeProof(ctx, record, project.RecordOperationsSmokeProofOptions{
			ProofKey:       flags.proofKey,
			EvidenceStatus: flags.status,
			Summary:        flags.summary,
			EvidenceURI:    flags.evidenceURI,
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(operationsSmokeProofToJSON(result))
		}
		c.printOperationsSmokeProof(result)
		return nil
	default:
		return fmt.Errorf("unknown ops command %q", args[0])
	}
}

func (c command) runSupport(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing support command: use `areaflow support bundle-preview`")
	}
	switch args[0] {
	case "bundle-preview":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.SupportBundlePreview(ctx, project.SupportBundlePreviewOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(supportBundlePreviewToJSON(preview))
		}
		c.printSupportBundlePreview(preview)
		return nil
	default:
		return fmt.Errorf("unknown support command %q", args[0])
	}
}

func (c command) runBackup(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing backup command: use `areaflow backup manifest` or `areaflow backup restore-plan`")
	}
	switch args[0] {
	case "manifest":
		flags, err := backupScopeFlagsFromArgs(args[1:], "areaflow backup manifest")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		manifest, err := store.BackupManifest(ctx, project.BackupManifestOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(backupManifestToJSON(manifest))
		}
		c.printBackupManifest(manifest)
		return nil
	case "restore-plan":
		jsonOutput, projectKey, err := restorePlanFlagsFromArgs(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		plan, err := store.RestorePlan(ctx, project.RestorePlanOptions{ProjectKey: projectKey})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(restorePlanToJSON(plan))
		}
		c.printRestorePlan(plan)
		return nil
	default:
		return fmt.Errorf("unknown backup command %q", args[0])
	}
}

func (c command) runRelease(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing release command: use `areaflow release readiness`, `areaflow release remediation-plan`, `areaflow release acceptance-preview`, `areaflow release acceptance-gate`, `areaflow release exception-doctor`, `areaflow release exception-record-preview`, `areaflow release exception-schema-preview`, `areaflow release exception-migration-approval-gate`, `areaflow release exception-apply-preview`, `areaflow release final-gate`, `areaflow release evidence-bundle`, `areaflow release package-preview`, `areaflow release distribution-preview`, `areaflow release publish-gate`, `areaflow release publish-approval-preview`, or `areaflow release rollout-plan-preview`")
	}
	switch args[0] {
	case "readiness":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release readiness")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		readiness, err := store.ReleaseReadiness(ctx, project.ReleaseReadinessOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseReadinessToJSON(readiness))
		}
		c.printReleaseReadiness(readiness)
		return nil
	case "remediation-plan":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release remediation-plan")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		plan, err := store.ReleaseRemediationPlan(ctx, project.ReleaseRemediationOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseRemediationPlanToJSON(plan))
		}
		c.printReleaseRemediationPlan(plan)
		return nil
	case "acceptance-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release acceptance-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleaseAcceptancePreview(ctx, project.ReleaseAcceptancePreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseAcceptancePreviewToJSON(preview))
		}
		c.printReleaseAcceptancePreview(preview)
		return nil
	case "acceptance-gate":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release acceptance-gate")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.ReleaseAcceptanceGate(ctx, project.ReleaseAcceptanceGateOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseAcceptanceGateToJSON(gate))
		}
		c.printReleaseAcceptanceGate(gate)
		return nil
	case "exception-doctor":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release exception-doctor")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		doctor, err := store.ReleaseExceptionDoctor(ctx, project.ReleaseExceptionDoctorOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseExceptionDoctorToJSON(doctor))
		}
		c.printReleaseExceptionDoctor(doctor)
		return nil
	case "exception-record-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release exception-record-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleaseExceptionRecordPreview(ctx, project.ReleaseExceptionRecordPreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseExceptionRecordPreviewToJSON(preview))
		}
		c.printReleaseExceptionRecordPreview(preview)
		return nil
	case "exception-schema-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release exception-schema-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleaseExceptionSchemaPreview(ctx, project.ReleaseExceptionSchemaPreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseExceptionSchemaPreviewToJSON(preview))
		}
		c.printReleaseExceptionSchemaPreview(preview)
		return nil
	case "exception-migration-approval-gate":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release exception-migration-approval-gate")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.ReleaseExceptionMigrationApprovalGate(ctx, project.ReleaseExceptionMigrationApprovalGateOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseExceptionMigrationApprovalGateToJSON(gate))
		}
		c.printReleaseExceptionMigrationApprovalGate(gate)
		return nil
	case "exception-migration-approve", "exception-migration-revoke":
		flags, err := releaseExceptionMigrationFlagsFromArgs(args[1:], "areaflow release "+args[0])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		var state migrate.ApprovalState
		if args[0] == "exception-migration-approve" {
			state, err = migrate.Approve(ctx, pool, migrate.ReleaseExceptionMigrationName, flags.actor, flags.reason)
		} else {
			state, err = migrate.Revoke(ctx, pool, migrate.ReleaseExceptionMigrationName, flags.actor, flags.reason)
		}
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(map[string]any{"migration": migrate.ReleaseExceptionMigrationName, "status": state.Status, "migration_hash": state.MigrationHash, "actor": state.Actor, "reason": state.Reason, "applied": state.Applied})
		}
		fmt.Fprintf(c.stdout, "release exception migration approval: migration=%s status=%s applied=%t actor=%s\n", migrate.ReleaseExceptionMigrationName, state.Status, state.Applied, state.Actor)
		return nil
	case "exception-migration-apply":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		applied, err := migrate.ApplyApproved(ctx, pool, migrate.ReleaseExceptionMigrationName)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(map[string]any{"migration": migrate.ReleaseExceptionMigrationName, "applied": applied})
		}
		fmt.Fprintf(c.stdout, "release exception migration: migration=%s applied=%t\n", migrate.ReleaseExceptionMigrationName, applied)
		return nil
	case "exception-request", "exception-approve", "exception-revoke":
		flags, err := releaseExceptionCommandFlagsFromArgs(args[1:], "areaflow release "+args[0], args[0] == "exception-request")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, flags.projectKey)
		if err != nil {
			return err
		}
		var exception project.ReleaseExceptionRecord
		switch args[0] {
		case "exception-request":
			exception, err = store.RequestReleaseException(ctx, record, project.RequestReleaseExceptionOptions{
				ExceptionKey: flags.exceptionKey, Actor: flags.actor, Reason: flags.reason, Owner: flags.owner,
				ReviewAt: flags.reviewAt, ExpiresAt: flags.expiresAt, IdempotencyKey: flags.idempotencyKey,
			})
		case "exception-approve":
			exception, err = store.ApproveReleaseException(ctx, record, project.DecideReleaseExceptionOptions{ExceptionKey: flags.exceptionKey, Actor: flags.actor, Reason: flags.reason, IdempotencyKey: flags.idempotencyKey})
		case "exception-revoke":
			exception, err = store.RevokeReleaseException(ctx, record, project.DecideReleaseExceptionOptions{ExceptionKey: flags.exceptionKey, Actor: flags.actor, Reason: flags.reason, IdempotencyKey: flags.idempotencyKey})
		}
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseExceptionRecordToJSON(exception))
		}
		fmt.Fprintf(c.stdout, "release exception: project=%s key=%s status=%s audit_event_id=%d created=%t\n", exception.ProjectKey, exception.ExceptionKey, exception.Status, exception.AuditEventID, exception.Created)
		return nil
	case "exception-apply-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release exception-apply-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleaseExceptionApplyPreview(ctx, project.ReleaseExceptionApplyPreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseExceptionApplyPreviewToJSON(preview))
		}
		c.printReleaseExceptionApplyPreview(preview)
		return nil
	case "final-gate":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release final-gate")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.ReleaseFinalGate(ctx, project.ReleaseFinalGateOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseFinalGateToJSON(gate))
		}
		c.printReleaseFinalGate(gate)
		return nil
	case "evidence-bundle":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release evidence-bundle")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		bundle, err := store.ReleaseEvidenceBundle(ctx, project.ReleaseEvidenceBundleOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseEvidenceBundleToJSON(bundle))
		}
		c.printReleaseEvidenceBundle(bundle)
		return nil
	case "package-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release package-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleasePackagePreview(ctx, project.ReleasePackagePreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releasePackagePreviewToJSON(preview))
		}
		c.printReleasePackagePreview(preview)
		return nil
	case "distribution-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release distribution-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleaseDistributionPreview(ctx, project.ReleaseDistributionPreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseDistributionPreviewToJSON(preview))
		}
		c.printReleaseDistributionPreview(preview)
		return nil
	case "publish-gate":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release publish-gate")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		gate, err := store.ReleasePublishGate(ctx, project.ReleasePublishGateOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releasePublishGateToJSON(gate))
		}
		c.printReleasePublishGate(gate)
		return nil
	case "publish-approval-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release publish-approval-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleasePublishApprovalPreview(ctx, project.ReleasePublishApprovalPreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releasePublishApprovalPreviewToJSON(preview))
		}
		c.printReleasePublishApprovalPreview(preview)
		return nil
	case "rollout-plan-preview":
		flags, err := releaseScopeFlagsFromArgs(args[1:], "areaflow release rollout-plan-preview")
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		preview, err := store.ReleaseRolloutPlanPreview(ctx, project.ReleaseRolloutPlanPreviewOptions{ProjectKey: flags.projectKey})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(releaseRolloutPlanPreviewToJSON(preview))
		}
		c.printReleaseRolloutPlanPreview(preview)
		return nil
	default:
		return fmt.Errorf("unknown release command %q", args[0])
	}
}

func (c command) runAudit(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing audit command: use `areaflow audit coverage`")
	}
	switch args[0] {
	case "coverage":
		flags, err := auditCoverageFlags(args[1:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		var projectID int64
		projectKey := flags.projectKey
		if projectKey != "" {
			record, err := store.GetByKey(ctx, projectKey)
			if err != nil {
				return err
			}
			projectID = record.ID
			projectKey = record.Key
		}
		coverage, err := store.AuditCoverage(ctx, project.AuditCoverageOptions{
			ProjectID:  projectID,
			ProjectKey: projectKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(auditCoverageToJSON(coverage))
		}
		c.printAuditCoverage(coverage)
		return nil
	default:
		return fmt.Errorf("unknown audit command %q", args[0])
	}
}

func (c command) runPermissions(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing permissions command: use `areaflow permissions doctor <project>`")
	}
	switch args[0] {
	case "doctor":
		if len(args) < 2 {
			return fmt.Errorf("missing project id: use `areaflow permissions doctor <project>`")
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		doctor, err := store.PermissionPolicyDoctor(ctx, record, project.PermissionPolicyDoctorOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(permissionPolicyDoctorToJSON(doctor))
		}
		c.printPermissionPolicyDoctor(doctor)
		return nil
	default:
		return fmt.Errorf("unknown permissions command %q", args[0])
	}
}

func (c command) runArtifact(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing artifact command: use `areaflow artifact integrity <project>` or `areaflow artifact archive-preview <project>`")
	}
	switch args[0] {
	case "integrity":
		if len(args) < 2 {
			return fmt.Errorf("missing project id: use `areaflow artifact integrity <project>`")
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		report, err := store.ArtifactIntegrity(ctx, record, project.ArtifactIntegrityOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(artifactIntegrityToJSON(report))
		}
		c.printArtifactIntegrity(report)
		return nil
	case "archive-preview":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow artifact archive-preview <project> [--json] [--retention-class CLASS] [--limit N] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := artifactArchivePreviewFlags(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		preview, err := store.ArtifactArchivePreview(ctx, record, project.ArtifactArchivePreviewOptions{
			RetentionClass: flags.retentionClass,
			Limit:          flags.limit,
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(artifactArchivePreviewToJSON(preview))
		}
		c.printArtifactArchivePreview(preview)
		return nil
	default:
		return fmt.Errorf("unknown artifact command %q", args[0])
	}
}

func (c command) runConformance(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing conformance command: use `areaflow conformance check <project>`")
	}
	switch args[0] {
	case "check":
		if len(args) < 2 {
			return fmt.Errorf("missing project id: use `areaflow conformance check <project>`")
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		cfg := config.FromEnv()
		pool, err := db.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		store := project.NewStore(pool)
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		report, err := store.ConformanceCheck(ctx, record, project.ConformanceOptions{})
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(conformanceToJSON(report))
		}
		c.printConformance(report)
		return nil
	default:
		return fmt.Errorf("unknown conformance command %q", args[0])
	}
}

func (c command) runWorker(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing worker command: use `register`, `heartbeat`, or `list`")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := project.NewStore(pool)

	switch args[0] {
	case "register":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow worker register <project> [--json] [--worker-key KEY] [--worker-type TYPE] [--hostname HOST] [--pid PID] [--capability CAP] [--heartbeat-interval SECONDS] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := workerRegisterFlags(args[2:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		worker, err := store.RegisterWorker(ctx, record, project.RegisterWorkerOptions{
			WorkerKey:                flags.workerKey,
			WorkerType:               flags.workerType,
			Hostname:                 flags.hostname,
			PID:                      flags.pid,
			Capabilities:             flags.capabilities,
			HeartbeatIntervalSeconds: flags.heartbeatIntervalSeconds,
			LeaseTimeoutSeconds:      flags.leaseTimeoutSeconds,
			Actor:                    flags.actor,
			Reason:                   flags.reason,
			IdempotencyKey:           flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workerToJSON(worker))
		}
		c.printWorker(worker)
		return nil
	case "heartbeat":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker heartbeat <project> <worker-key> [--json] [--status STATUS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := workerHeartbeatFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		worker, err := store.RecordWorkerHeartbeat(ctx, record, args[2], project.WorkerHeartbeatOptions{
			Status:         flags.status,
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workerToJSON(worker))
		}
		c.printWorker(worker)
		return nil
	case "list":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow worker list <project> [--json] [--limit N]")
		}
		flags, err := workerListFlags(args[2:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		workers, err := store.ListWorkers(ctx, record, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workerListToJSON(record, workers))
		}
		c.printWorkers(record, workers)
		return nil
	case "pool-summary":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return fmt.Errorf("usage: areaflow worker pool-summary [--json]")
		}
		summary, err := store.WorkerPoolSummary(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(workerPoolSummaryToJSON(summary))
		}
		c.printWorkerPoolSummary(summary)
		return nil
	case "schedule-preview":
		jsonOutput, err := outputJSON(args[1:])
		if err != nil {
			return fmt.Errorf("usage: areaflow worker schedule-preview [--json]")
		}
		preview, err := store.WorkerPoolSchedulePreview(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(workerPoolSchedulePreviewToJSON(preview))
		}
		c.printWorkerPoolSchedulePreview(preview)
		return nil
	case "lease-acquire":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker lease-acquire <project> <worker-key> --run-task-id ID [--json] [--lease-kind KIND] [--capability CAP] [--lease-timeout SECONDS] [--recover-expired] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := leaseAcquireFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		lease, err := store.AcquireLease(ctx, record, project.AcquireLeaseOptions{
			WorkerKey:            args[2],
			RunTaskID:            flags.runTaskID,
			LeaseKind:            flags.leaseKind,
			AllowedCapabilities:  flags.capabilities,
			LeaseTimeoutSeconds:  flags.leaseTimeoutSeconds,
			RecoverExpiredBefore: flags.recoverExpired,
			Actor:                flags.actor,
			Reason:               flags.reason,
			IdempotencyKey:       flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(leaseToJSON(lease))
		}
		c.printLease(lease)
		return nil
	case "lease-release":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker lease-release <project> <worker-key> --lease-id ID [--json] [--status STATUS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := leaseReleaseFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		lease, err := store.ReleaseLease(ctx, record, project.ReleaseLeaseOptions{
			WorkerKey:      args[2],
			LeaseID:        flags.leaseID,
			Status:         flags.status,
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(leaseToJSON(lease))
		}
		c.printLease(lease)
		return nil
	case "lease-recover":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow worker lease-recover <project> [--json] [--limit N] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := leaseRecoverFlags(args[2:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		leases, err := store.RecoverExpiredLeases(ctx, record, project.RecoverLeasesOptions{
			Limit:          flags.limit,
			Actor:          flags.actor,
			Reason:         flags.reason,
			IdempotencyKey: flags.idempotencyKey,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(leaseRecoverToJSON(record, leases))
		}
		c.printLeases(record, leases)
		return nil
	case "run-once":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker run-once <project> <worker-key> [--json] [--run-id ID] [--capability CAP] [--lease-timeout SECONDS] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := workerRunOnceFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.RunWorkerOnce(ctx, record, project.WorkerRunOnceOptions{
			WorkerKey:           args[2],
			RunID:               flags.runID,
			AllowedCapabilities: flags.capabilities,
			LeaseTimeoutSeconds: flags.leaseTimeoutSeconds,
			Actor:               flags.actor,
			Reason:              flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workerRunOnceToJSON(result))
		}
		c.printWorkerRunOnce(result)
		return nil
	case "fixture-execute":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker fixture-execute <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := fixtureExecutionFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.ExecuteFixture(ctx, record, project.FixtureExecutionOptions{
			WorkerKey:           args[2],
			RunID:               flags.runID,
			AllowedCapabilities: flags.capabilities,
			LeaseTimeoutSeconds: flags.leaseTimeoutSeconds,
			IdempotencyKey:      flags.idempotencyKey,
			Actor:               flags.actor,
			Reason:              flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(fixtureExecutionToJSON(result))
		}
		c.printFixtureExecution(result)
		return nil
	case "read-only-verify":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker read-only-verify <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := readOnlyVerifyFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.VerifyReadOnly(ctx, record, project.ReadOnlyVerifyOptions{
			WorkerKey:           args[2],
			RunID:               flags.runID,
			AllowedCapabilities: flags.capabilities,
			LeaseTimeoutSeconds: flags.leaseTimeoutSeconds,
			IdempotencyKey:      flags.idempotencyKey,
			Actor:               flags.actor,
			Reason:              flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(readOnlyVerifyToJSON(result))
		}
		c.printReadOnlyVerify(result)
		return nil
	case "approved-artifact-write":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker approved-artifact-write <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := approvedArtifactWriteFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.WriteApprovedArtifact(ctx, record, project.ApprovedArtifactWriteOptions{
			WorkerKey:           args[2],
			RunID:               flags.runID,
			AllowedCapabilities: flags.capabilities,
			LeaseTimeoutSeconds: flags.leaseTimeoutSeconds,
			IdempotencyKey:      flags.idempotencyKey,
			Actor:               flags.actor,
			Reason:              flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(approvedArtifactWriteToJSON(result))
		}
		c.printApprovedArtifactWrite(result)
		return nil
	case "fixture-project-write":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker fixture-project-write <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := fixtureProjectWriteFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.WriteFixtureProject(ctx, record, project.FixtureProjectWriteOptions{
			WorkerKey:           args[2],
			RunID:               flags.runID,
			AllowedCapabilities: flags.capabilities,
			LeaseTimeoutSeconds: flags.leaseTimeoutSeconds,
			IdempotencyKey:      flags.idempotencyKey,
			Actor:               flags.actor,
			Reason:              flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(fixtureProjectWriteToJSON(result))
		}
		c.printFixtureProjectWrite(result)
		return nil
	case "managed-generated-write":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow worker managed-generated-write <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := managedGeneratedWriteFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.WriteManagedGenerated(ctx, record, project.ManagedGeneratedWriteOptions{
			WorkerKey:           args[2],
			RunID:               flags.runID,
			AllowedCapabilities: flags.capabilities,
			LeaseTimeoutSeconds: flags.leaseTimeoutSeconds,
			IdempotencyKey:      flags.idempotencyKey,
			Actor:               flags.actor,
			Reason:              flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(managedGeneratedWriteToJSON(result))
		}
		c.printManagedGeneratedWrite(result)
		return nil
	default:
		return fmt.Errorf("unknown worker command %q", args[0])
	}
}

func (c command) runEngine(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing engine command: use `areaflow engine codex-preview <project>`")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := project.NewStore(pool)

	switch args[0] {
	case "codex-preview":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow engine codex-preview <project> [--json] [--command COMMAND]")
		}
		flags, err := codexPreviewFlags(args[2:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		preview, err := store.CodexCLIAdapterPreview(ctx, record, project.CodexCLIAdapterPreviewOptions{
			Command: flags.command,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(codexCLIAdapterPreviewToJSON(preview))
		}
		c.printCodexCLIAdapterPreview(preview)
		return nil
	default:
		return fmt.Errorf("unknown engine command %q", args[0])
	}
}

func (c command) runWorkflowVersion(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("missing workflow version command: use `create`, `list`, `show`, `stages`, `ensure-skeleton`, or `mark-ready`")
	}

	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := project.NewStore(pool)
	switch args[0] {
	case "create":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow version create <project> <label> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := workflowVersionCreateFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.CreateWorkflowVersion(ctx, record, project.CreateWorkflowVersionOptions{
			DisplayLabel:   args[2],
			IdempotencyKey: flags.idempotencyKey,
			Actor:          flags.actor,
			Reason:         flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(workflowVersionCreateToJSON(result))
		}
		c.printWorkflowVersionCreate(result)
		return nil
	case "list":
		if len(args) < 2 {
			return fmt.Errorf("usage: areaflow workflow version list <project> [--json]")
		}
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		versions, err := store.ListWorkflowVersions(ctx, record)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(workflowVersionListToJSON(record, versions))
		}
		c.printWorkflowVersionList(record, versions)
		return nil
	case "show":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow version show <project> <label> [--json]")
		}
		jsonOutput, err := outputJSON(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		version, err := store.GetWorkflowVersion(ctx, record, args[2])
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(workflowVersionToJSON(version))
		}
		c.printWorkflowVersion(record, version)
		return nil
	case "stages":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow version stages <project> <label> [--json]")
		}
		jsonOutput, err := outputJSON(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		version, err := store.GetWorkflowVersion(ctx, record, args[2])
		if err != nil {
			return err
		}
		items, err := store.ListWorkflowItems(ctx, record, version)
		if err != nil {
			return err
		}
		links, err := store.ListWorkflowItemLinks(ctx, record, version, 100)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(workflowVersionStagesToJSON(record, version, items, links))
		}
		c.printWorkflowVersionStages(record, version, items)
		return nil
	case "ensure-skeleton":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow version ensure-skeleton <project> <label> [--json] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := stageSkeletonFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.EnsureStageSkeleton(ctx, record, args[2], project.EnsureStageSkeletonOptions{
			Actor:  flags.actor,
			Reason: flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(ensureStageSkeletonToJSON(result))
		}
		c.printEnsureStageSkeleton(result)
		return nil
	case "mark-ready":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow version mark-ready <project> <label> --stage STAGE --item-type TYPE [--json] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := workflowItemReadyFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.MarkWorkflowItemReady(ctx, record, args[2], project.MarkWorkflowItemReadyOptions{
			Stage:    flags.stage,
			ItemType: flags.itemType,
			Actor:    flags.actor,
			Reason:   flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(markWorkflowItemReadyToJSON(result))
		}
		c.printMarkWorkflowItemReady(result)
		return nil
	default:
		return fmt.Errorf("unknown workflow version command %q", args[0])
	}
}

func (c command) runWorkflowGate(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing workflow gate command: use `run` or `list`")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := project.NewStore(pool)
	switch args[0] {
	case "run":
		if len(args) < 4 {
			return fmt.Errorf("usage: areaflow workflow gate run <project> <label> <gate_name> [--json] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := gateFlags(args[4:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		result, err := store.RunWorkflowGate(ctx, record, args[2], project.RunGateOptions{
			GateName: args[3],
			Actor:    flags.actor,
			Reason:   flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(gateResultToJSON(result))
		}
		c.printGateResult(result)
		return nil
	case "list":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow gate list <project> <label> [--json] [--limit N]")
		}
		flags, err := gateListFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		version, err := store.GetWorkflowVersion(ctx, record, args[2])
		if err != nil {
			return err
		}
		results, err := store.ListGateResults(ctx, record, version, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(gateResultsToJSON(record, version, results))
		}
		c.printGateResults(record, version, results)
		return nil
	default:
		return fmt.Errorf("unknown workflow gate command %q", args[0])
	}
}

func (c command) runWorkflowTransition(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing workflow transition command: use `preview` or `list`")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := project.NewStore(pool)
	switch args[0] {
	case "preview":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow transition preview <project> <label> [--json] [--from STAGE] [--to STAGE] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := transitionPreviewFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		preview, err := store.PreviewWorkflowTransition(ctx, record, args[2], project.PreviewTransitionOptions{
			FromStage: flags.fromStage,
			ToStage:   flags.toStage,
			Actor:     flags.actor,
			Reason:    flags.reason,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(transitionPreviewToJSON(preview))
		}
		c.printTransitionPreview(preview)
		return nil
	case "list":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow transition list <project> <label> [--json] [--limit N]")
		}
		flags, err := gateListFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		version, err := store.GetWorkflowVersion(ctx, record, args[2])
		if err != nil {
			return err
		}
		previews, err := store.ListWorkflowTransitionPreviews(ctx, record, version, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(transitionPreviewsToJSON(record, version, previews))
		}
		c.printTransitionPreviews(record, version, previews)
		return nil
	default:
		return fmt.Errorf("unknown workflow transition command %q", args[0])
	}
}

func (c command) runWorkflowApproval(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing workflow approval command: use `record` or `list`")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := project.NewStore(pool)
	switch args[0] {
	case "record":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow approval record <project> <label> [--json] [--decision approved|rejected] [--transition-preview-id ID] [--kind KIND] [--risk-level LEVEL] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
		flags, err := approvalRecordFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		approval, err := store.CreateApprovalRecord(ctx, record, args[2], project.CreateApprovalOptions{
			Decision:            flags.decision,
			ApprovalKind:        flags.kind,
			Actor:               flags.actor,
			Reason:              flags.reason,
			RiskLevel:           flags.riskLevel,
			IdempotencyKey:      flags.idempotencyKey,
			TransitionPreviewID: flags.transitionPreviewID,
		})
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(approvalRecordToJSON(approval))
		}
		c.printApprovalRecord(approval)
		return nil
	case "list":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow workflow approval list <project> <label> [--json] [--limit N]")
		}
		flags, err := gateListFlags(args[3:])
		if err != nil {
			return err
		}
		record, err := store.GetByKey(ctx, args[1])
		if err != nil {
			return err
		}
		version, err := store.GetWorkflowVersion(ctx, record, args[2])
		if err != nil {
			return err
		}
		approvals, err := store.ListApprovalRecords(ctx, record, version, flags.limit)
		if err != nil {
			return err
		}
		if flags.json {
			return c.printJSON(approvalRecordsToJSON(record, version, approvals))
		}
		c.printApprovalRecords(record, version, approvals)
		return nil
	default:
		return fmt.Errorf("unknown workflow approval command %q", args[0])
	}
}

func projectConfigPath(args []string) (string, error) {
	if len(args) != 2 || args[0] != "--config" {
		return "", fmt.Errorf("usage: areaflow project add --config <path>")
	}
	return args[1], nil
}

type workflowVersionCreateFlagSet struct {
	json           bool
	idempotencyKey string
	actor          string
	reason         string
}

func workflowVersionCreateFlags(args []string) (workflowVersionCreateFlagSet, error) {
	flags := workflowVersionCreateFlagSet{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--idempotency-key":
			if i+1 >= len(args) {
				return workflowVersionCreateFlagSet{}, fmt.Errorf("usage: areaflow workflow version create <project> <label> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return workflowVersionCreateFlagSet{}, fmt.Errorf("usage: areaflow workflow version create <project> <label> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return workflowVersionCreateFlagSet{}, fmt.Errorf("usage: areaflow workflow version create <project> <label> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
			}
			flags.reason = args[i+1]
			i++
		default:
			return workflowVersionCreateFlagSet{}, fmt.Errorf("usage: areaflow workflow version create <project> <label> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]")
		}
	}
	return flags, nil
}

type stageSkeletonFlagSet struct {
	json   bool
	actor  string
	reason string
}

func stageSkeletonFlags(args []string) (stageSkeletonFlagSet, error) {
	flags := stageSkeletonFlagSet{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--actor":
			if i+1 >= len(args) {
				return stageSkeletonFlagSet{}, fmt.Errorf("usage: areaflow workflow version ensure-skeleton <project> <label> [--json] [--actor ACTOR] [--reason TEXT]")
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return stageSkeletonFlagSet{}, fmt.Errorf("usage: areaflow workflow version ensure-skeleton <project> <label> [--json] [--actor ACTOR] [--reason TEXT]")
			}
			flags.reason = args[i+1]
			i++
		default:
			return stageSkeletonFlagSet{}, fmt.Errorf("usage: areaflow workflow version ensure-skeleton <project> <label> [--json] [--actor ACTOR] [--reason TEXT]")
		}
	}
	return flags, nil
}

type workflowItemReadyFlagSet struct {
	json     bool
	stage    string
	itemType string
	actor    string
	reason   string
}

func workflowItemReadyFlags(args []string) (workflowItemReadyFlagSet, error) {
	flags := workflowItemReadyFlagSet{}
	usage := "usage: areaflow workflow version mark-ready <project> <label> --stage STAGE --item-type TYPE [--json] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--stage":
			if i+1 >= len(args) {
				return workflowItemReadyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.stage = args[i+1]
			i++
		case "--item-type":
			if i+1 >= len(args) {
				return workflowItemReadyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.itemType = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return workflowItemReadyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return workflowItemReadyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return workflowItemReadyFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if strings.TrimSpace(flags.stage) == "" || strings.TrimSpace(flags.itemType) == "" {
		return workflowItemReadyFlagSet{}, fmt.Errorf("stage and item type are required")
	}
	return flags, nil
}

type workflowProfileFlagSet struct {
	json bool
}

func workflowProfileFlags(args []string) (workflowProfileFlagSet, error) {
	flags := workflowProfileFlagSet{}
	for _, arg := range args {
		switch arg {
		case "--json":
			flags.json = true
		default:
			return workflowProfileFlagSet{}, fmt.Errorf("usage: areaflow workflow profile list [--json] | show|check <profile> [--json]")
		}
	}
	return flags, nil
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

type gateFlagSet struct {
	json   bool
	actor  string
	reason string
}

func gateFlags(args []string) (gateFlagSet, error) {
	flags := gateFlagSet{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--actor":
			if i+1 >= len(args) {
				return gateFlagSet{}, fmt.Errorf("usage: areaflow workflow gate run <project> <label> <gate_name> [--json] [--actor ACTOR] [--reason TEXT]")
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return gateFlagSet{}, fmt.Errorf("usage: areaflow workflow gate run <project> <label> <gate_name> [--json] [--actor ACTOR] [--reason TEXT]")
			}
			flags.reason = args[i+1]
			i++
		default:
			return gateFlagSet{}, fmt.Errorf("usage: areaflow workflow gate run <project> <label> <gate_name> [--json] [--actor ACTOR] [--reason TEXT]")
		}
	}
	return flags, nil
}

type gateListFlagSet struct {
	json  bool
	limit int
}

type transitionPreviewFlagSet struct {
	json      bool
	fromStage string
	toStage   string
	actor     string
	reason    string
}

func transitionPreviewFlags(args []string) (transitionPreviewFlagSet, error) {
	flags := transitionPreviewFlagSet{}
	usage := "usage: areaflow workflow transition preview <project> <label> [--json] [--from STAGE] [--to STAGE] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--from":
			if i+1 >= len(args) {
				return transitionPreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.fromStage = args[i+1]
			i++
		case "--to":
			if i+1 >= len(args) {
				return transitionPreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.toStage = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return transitionPreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return transitionPreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return transitionPreviewFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

type approvalRecordFlagSet struct {
	json                bool
	decision            string
	kind                string
	riskLevel           string
	idempotencyKey      string
	actor               string
	reason              string
	transitionPreviewID int64
}

type runnerPreviewFlagSet struct {
	json           bool
	actor          string
	reason         string
	riskLevel      string
	riskPolicy     string
	idempotencyKey string
}

type fixtureExecutionQueueFlagSet struct {
	json           bool
	actor          string
	reason         string
	idempotencyKey string
}

type readOnlyVerifyQueueFlagSet struct {
	json           bool
	targetPath     string
	actor          string
	reason         string
	idempotencyKey string
}

type approvedArtifactWriteQueueFlagSet struct {
	json           bool
	artifactLabel  string
	actor          string
	reason         string
	idempotencyKey string
}

type fixtureProjectWriteQueueFlagSet struct {
	json                 bool
	targetPath           string
	content              string
	expectedBeforeSHA256 string
	expectedBeforeSize   int64
	actor                string
	reason               string
	idempotencyKey       string
}

type managedGeneratedWriteQueueFlagSet struct {
	json                 bool
	targetPath           string
	content              string
	expectedBeforeSHA256 string
	expectedBeforeSize   int64
	actor                string
	reason               string
	idempotencyKey       string
}

type runControlFlagSet struct {
	json           bool
	actor          string
	reason         string
	idempotencyKey string
}

type executionGateFlagSet struct {
	json         bool
	capabilities []string
}

type executionPlanFlagSet struct {
	json bool
}

type projectWriteDesignGateFlagSet struct {
	json bool
}

type managedGeneratedWriteGateFlagSet struct {
	json bool
}

type workerRegisterFlagSet struct {
	json                     bool
	workerKey                string
	workerType               string
	hostname                 string
	pid                      int
	capabilities             []string
	heartbeatIntervalSeconds int
	leaseTimeoutSeconds      int
	actor                    string
	reason                   string
	idempotencyKey           string
}

type workerHeartbeatFlagSet struct {
	json           bool
	status         string
	actor          string
	reason         string
	idempotencyKey string
}

type workerListFlagSet struct {
	json  bool
	limit int
}

type leaseAcquireFlagSet struct {
	json                bool
	runTaskID           int64
	leaseKind           string
	capabilities        []string
	leaseTimeoutSeconds int
	recoverExpired      bool
	idempotencyKey      string
	actor               string
	reason              string
}

type leaseReleaseFlagSet struct {
	json           bool
	leaseID        int64
	status         string
	idempotencyKey string
	actor          string
	reason         string
}

type leaseRecoverFlagSet struct {
	json           bool
	limit          int
	idempotencyKey string
	actor          string
	reason         string
}

type workerRunOnceFlagSet struct {
	json                bool
	runID               int64
	capabilities        []string
	leaseTimeoutSeconds int
	actor               string
	reason              string
}

type fixtureExecutionFlagSet struct {
	json                bool
	runID               int64
	capabilities        []string
	leaseTimeoutSeconds int
	idempotencyKey      string
	actor               string
	reason              string
}

type readOnlyVerifyFlagSet struct {
	json                bool
	runID               int64
	capabilities        []string
	leaseTimeoutSeconds int
	idempotencyKey      string
	actor               string
	reason              string
}

type approvedArtifactWriteFlagSet struct {
	json                bool
	runID               int64
	capabilities        []string
	leaseTimeoutSeconds int
	idempotencyKey      string
	actor               string
	reason              string
}

type fixtureProjectWriteFlagSet struct {
	json                bool
	runID               int64
	capabilities        []string
	leaseTimeoutSeconds int
	idempotencyKey      string
	actor               string
	reason              string
}

type managedGeneratedWriteFlagSet struct {
	json                bool
	runID               int64
	capabilities        []string
	leaseTimeoutSeconds int
	idempotencyKey      string
	actor               string
	reason              string
}

type artifactArchivePreviewFlagSet struct {
	json           bool
	retentionClass string
	limit          int
	idempotencyKey string
	actor          string
	reason         string
}

type serviceStatusFlagSet struct {
	json   bool
	webURL string
}

type auditCoverageFlagSet struct {
	json       bool
	projectKey string
}

func approvalRecordFlags(args []string) (approvalRecordFlagSet, error) {
	flags := approvalRecordFlagSet{}
	usage := "usage: areaflow workflow approval record <project> <label> [--json] [--decision approved|rejected] [--transition-preview-id ID] [--kind KIND] [--risk-level LEVEL] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--decision":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.decision = args[i+1]
			i++
		case "--kind":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.kind = args[i+1]
			i++
		case "--risk-level":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.riskLevel = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--transition-preview-id":
			if i+1 >= len(args) {
				return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
			}
			id, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || id <= 0 {
				return approvalRecordFlagSet{}, fmt.Errorf("transition preview id must be a positive integer")
			}
			flags.transitionPreviewID = id
			i++
		default:
			return approvalRecordFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func workerRunOnceFlags(args []string) (workerRunOnceFlagSet, error) {
	flags := workerRunOnceFlagSet{}
	usage := "usage: areaflow worker run-once <project> <worker-key> [--json] [--run-id ID] [--capability CAP] [--lease-timeout SECONDS] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-id":
			if i+1 >= len(args) {
				return workerRunOnceFlagSet{}, fmt.Errorf("%s", usage)
			}
			runID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || runID <= 0 {
				return workerRunOnceFlagSet{}, fmt.Errorf("run id must be a positive integer")
			}
			flags.runID = runID
			i++
		case "--capability":
			if i+1 >= len(args) {
				return workerRunOnceFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return workerRunOnceFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return workerRunOnceFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--actor":
			if i+1 >= len(args) {
				return workerRunOnceFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return workerRunOnceFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return workerRunOnceFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func fixtureExecutionFlags(args []string) (fixtureExecutionFlagSet, error) {
	flags := fixtureExecutionFlagSet{}
	usage := "usage: areaflow worker fixture-execute <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-id":
			if i+1 >= len(args) {
				return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
			}
			runID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || runID <= 0 {
				return fixtureExecutionFlagSet{}, fmt.Errorf("run id must be a positive integer")
			}
			flags.runID = runID
			i++
		case "--capability":
			if i+1 >= len(args) {
				return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return fixtureExecutionFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.runID == 0 {
		return fixtureExecutionFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func readOnlyVerifyFlags(args []string) (readOnlyVerifyFlagSet, error) {
	flags := readOnlyVerifyFlagSet{}
	usage := "usage: areaflow worker read-only-verify <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-id":
			if i+1 >= len(args) {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
			}
			runID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || runID <= 0 {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("run id must be a positive integer")
			}
			flags.runID = runID
			i++
		case "--capability":
			if i+1 >= len(args) {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.runID == 0 {
		return readOnlyVerifyFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func approvedArtifactWriteFlags(args []string) (approvedArtifactWriteFlagSet, error) {
	flags := approvedArtifactWriteFlagSet{}
	usage := "usage: areaflow worker approved-artifact-write <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-id":
			if i+1 >= len(args) {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			runID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || runID <= 0 {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("run id must be a positive integer")
			}
			flags.runID = runID
			i++
		case "--capability":
			if i+1 >= len(args) {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.runID == 0 {
		return approvedArtifactWriteFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func fixtureProjectWriteFlags(args []string) (fixtureProjectWriteFlagSet, error) {
	flags := fixtureProjectWriteFlagSet{}
	usage := "usage: areaflow worker fixture-project-write <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-id":
			if i+1 >= len(args) {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			runID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || runID <= 0 {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("run id must be a positive integer")
			}
			flags.runID = runID
			i++
		case "--capability":
			if i+1 >= len(args) {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.runID == 0 {
		return fixtureProjectWriteFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func managedGeneratedWriteFlags(args []string) (managedGeneratedWriteFlagSet, error) {
	flags := managedGeneratedWriteFlagSet{}
	usage := "usage: areaflow worker managed-generated-write <project> <worker-key> --run-id ID [--json] [--capability CAP] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-id":
			if i+1 >= len(args) {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			runID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || runID <= 0 {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("run id must be a positive integer")
			}
			flags.runID = runID
			i++
		case "--capability":
			if i+1 >= len(args) {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.runID == 0 {
		return managedGeneratedWriteFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func serviceStatusFlags(args []string) (serviceStatusFlagSet, error) {
	flags := serviceStatusFlagSet{}
	usage := "usage: areaflow service status [--json] [--web-url URL]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--web-url":
			if i+1 >= len(args) {
				return serviceStatusFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.webURL = args[i+1]
			i++
		default:
			return serviceStatusFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func auditCoverageFlags(args []string) (auditCoverageFlagSet, error) {
	flags := auditCoverageFlagSet{}
	usage := "usage: areaflow audit coverage [--json] [--project KEY]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--project":
			if i+1 >= len(args) {
				return auditCoverageFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.projectKey = strings.TrimSpace(args[i+1])
			if flags.projectKey == "" {
				return auditCoverageFlagSet{}, fmt.Errorf("project key is required")
			}
			i++
		default:
			return auditCoverageFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func leaseAcquireFlags(args []string) (leaseAcquireFlagSet, error) {
	flags := leaseAcquireFlagSet{}
	usage := "usage: areaflow worker lease-acquire <project> <worker-key> --run-task-id ID [--json] [--lease-kind KIND] [--capability CAP] [--lease-timeout SECONDS] [--recover-expired] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--run-task-id":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			id, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || id <= 0 {
				return leaseAcquireFlagSet{}, fmt.Errorf("run task id must be a positive integer")
			}
			flags.runTaskID = id
			i++
		case "--lease-kind":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.leaseKind = args[i+1]
			i++
		case "--capability":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return leaseAcquireFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--recover-expired":
			flags.recoverExpired = true
		case "--idempotency-key":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return leaseAcquireFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.runTaskID == 0 {
		return leaseAcquireFlagSet{}, fmt.Errorf("run task id is required")
	}
	return flags, nil
}

func leaseReleaseFlags(args []string) (leaseReleaseFlagSet, error) {
	flags := leaseReleaseFlagSet{}
	usage := "usage: areaflow worker lease-release <project> <worker-key> --lease-id ID [--json] [--status STATUS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--lease-id":
			if i+1 >= len(args) {
				return leaseReleaseFlagSet{}, fmt.Errorf("%s", usage)
			}
			id, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || id <= 0 {
				return leaseReleaseFlagSet{}, fmt.Errorf("lease id must be a positive integer")
			}
			flags.leaseID = id
			i++
		case "--status":
			if i+1 >= len(args) {
				return leaseReleaseFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return leaseReleaseFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return leaseReleaseFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return leaseReleaseFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return leaseReleaseFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.leaseID == 0 {
		return leaseReleaseFlagSet{}, fmt.Errorf("lease id is required")
	}
	return flags, nil
}

func leaseRecoverFlags(args []string) (leaseRecoverFlagSet, error) {
	flags := leaseRecoverFlagSet{limit: 20}
	usage := "usage: areaflow worker lease-recover <project> [--json] [--limit N] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--limit":
			if i+1 >= len(args) {
				return leaseRecoverFlagSet{}, fmt.Errorf("%s", usage)
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return leaseRecoverFlagSet{}, fmt.Errorf("lease recover limit must be a positive integer")
			}
			flags.limit = limit
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return leaseRecoverFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return leaseRecoverFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return leaseRecoverFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return leaseRecoverFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func workerRegisterFlags(args []string) (workerRegisterFlagSet, error) {
	flags := workerRegisterFlagSet{}
	usage := "usage: areaflow worker register <project> [--json] [--worker-key KEY] [--worker-type TYPE] [--hostname HOST] [--pid PID] [--capability CAP] [--heartbeat-interval SECONDS] [--lease-timeout SECONDS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--worker-key":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.workerKey = args[i+1]
			i++
		case "--worker-type":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.workerType = args[i+1]
			i++
		case "--hostname":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.hostname = args[i+1]
			i++
		case "--pid":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			pid, err := strconv.Atoi(args[i+1])
			if err != nil || pid <= 0 {
				return workerRegisterFlagSet{}, fmt.Errorf("worker pid must be a positive integer")
			}
			flags.pid = pid
			i++
		case "--capability":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		case "--heartbeat-interval":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			interval, err := strconv.Atoi(args[i+1])
			if err != nil || interval <= 0 {
				return workerRegisterFlagSet{}, fmt.Errorf("heartbeat interval must be a positive integer")
			}
			flags.heartbeatIntervalSeconds = interval
			i++
		case "--lease-timeout":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			timeout, err := strconv.Atoi(args[i+1])
			if err != nil || timeout <= 0 {
				return workerRegisterFlagSet{}, fmt.Errorf("lease timeout must be a positive integer")
			}
			flags.leaseTimeoutSeconds = timeout
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return workerRegisterFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func workerHeartbeatFlags(args []string) (workerHeartbeatFlagSet, error) {
	flags := workerHeartbeatFlagSet{}
	usage := "usage: areaflow worker heartbeat <project> <worker-key> [--json] [--status STATUS] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return workerHeartbeatFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return workerHeartbeatFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return workerHeartbeatFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return workerHeartbeatFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return workerHeartbeatFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func workerListFlags(args []string) (workerListFlagSet, error) {
	flags := workerListFlagSet{limit: 20}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--limit":
			if i+1 >= len(args) {
				return workerListFlagSet{}, fmt.Errorf("usage: areaflow worker list <project> [--json] [--limit N]")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return workerListFlagSet{}, fmt.Errorf("worker limit must be a positive integer")
			}
			flags.limit = limit
			i++
		default:
			return workerListFlagSet{}, fmt.Errorf("usage: areaflow worker list <project> [--json] [--limit N]")
		}
	}
	return flags, nil
}

func runnerPreviewFlags(args []string) (runnerPreviewFlagSet, error) {
	flags := runnerPreviewFlagSet{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--actor":
			if i+1 >= len(args) {
				return runnerPreviewFlagSet{}, fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return runnerPreviewFlagSet{}, fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
			}
			flags.reason = args[i+1]
			i++
		case "--risk-level":
			if i+1 >= len(args) {
				return runnerPreviewFlagSet{}, fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
			}
			flags.riskLevel = args[i+1]
			i++
		case "--risk-policy":
			if i+1 >= len(args) {
				return runnerPreviewFlagSet{}, fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
			}
			flags.riskPolicy = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return runnerPreviewFlagSet{}, fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return runnerPreviewFlagSet{}, fmt.Errorf("usage: areaflow run preview <project> <version> [--json] [--actor A] [--reason R] [--risk-level L] [--risk-policy P] [--idempotency-key K]")
		}
	}
	return flags, nil
}

func fixtureExecutionQueueFlags(args []string) (fixtureExecutionQueueFlagSet, error) {
	flags := fixtureExecutionQueueFlagSet{}
	usage := "usage: areaflow run fixture-queue <project> <version> [--json] [--actor A] [--reason R] [--idempotency-key K]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--actor":
			if i+1 >= len(args) {
				return fixtureExecutionQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return fixtureExecutionQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return fixtureExecutionQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return fixtureExecutionQueueFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func readOnlyVerifyQueueFlags(args []string) (readOnlyVerifyQueueFlagSet, error) {
	flags := readOnlyVerifyQueueFlagSet{}
	usage := "usage: areaflow run read-only-verify-queue <project> <version> --target-path PATH [--json] [--actor A] [--reason R] [--idempotency-key K]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target-path":
			if i+1 >= len(args) {
				return readOnlyVerifyQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.targetPath = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return readOnlyVerifyQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return readOnlyVerifyQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return readOnlyVerifyQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return readOnlyVerifyQueueFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if strings.TrimSpace(flags.targetPath) == "" {
		return readOnlyVerifyQueueFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func approvedArtifactWriteQueueFlags(args []string) (approvedArtifactWriteQueueFlagSet, error) {
	flags := approvedArtifactWriteQueueFlagSet{}
	usage := "usage: areaflow run approved-artifact-write-queue <project> <version> [--artifact-label LABEL] [--json] [--actor A] [--reason R] [--idempotency-key K]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--artifact-label":
			if i+1 >= len(args) {
				return approvedArtifactWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.artifactLabel = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return approvedArtifactWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return approvedArtifactWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return approvedArtifactWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return approvedArtifactWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func fixtureProjectWriteQueueFlags(args []string) (fixtureProjectWriteQueueFlagSet, error) {
	flags := fixtureProjectWriteQueueFlagSet{}
	usage := "usage: areaflow run fixture-project-write-queue <project> <version> --target-path PATH --content TEXT --expected-before-sha256 HASH --expected-before-size N [--json] [--actor A] [--reason R] [--idempotency-key K]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target-path":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.targetPath = args[i+1]
			i++
		case "--content":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.content = args[i+1]
			i++
		case "--expected-before-sha256":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.expectedBeforeSHA256 = args[i+1]
			i++
		case "--expected-before-size":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			size, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || size < 0 {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("expected before size must be a non-negative integer")
			}
			flags.expectedBeforeSize = size
			i++
		case "--actor":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if strings.TrimSpace(flags.targetPath) == "" || strings.TrimSpace(flags.expectedBeforeSHA256) == "" {
		return fixtureProjectWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func managedGeneratedWriteQueueFlags(args []string) (managedGeneratedWriteQueueFlagSet, error) {
	flags := managedGeneratedWriteQueueFlagSet{}
	usage := "usage: areaflow run managed-generated-write-queue <project> <version> --target-path PATH --content TEXT --expected-before-sha256 HASH --expected-before-size N [--json] [--actor A] [--reason R] [--idempotency-key K]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target-path":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.targetPath = args[i+1]
			i++
		case "--content":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.content = args[i+1]
			i++
		case "--expected-before-sha256":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.expectedBeforeSHA256 = args[i+1]
			i++
		case "--expected-before-size":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			size, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || size < 0 {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("expected before size must be a non-negative integer")
			}
			flags.expectedBeforeSize = size
			i++
		case "--actor":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	if strings.TrimSpace(flags.targetPath) == "" || strings.TrimSpace(flags.expectedBeforeSHA256) == "" {
		return managedGeneratedWriteQueueFlagSet{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func runControlFlags(args []string, action string) (runControlFlagSet, error) {
	flags := runControlFlagSet{}
	usage := fmt.Sprintf("usage: areaflow run %s <run-id> [--json] [--actor ACTOR] [--reason TEXT] [--idempotency-key KEY]", action)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--actor":
			if i+1 >= len(args) {
				return runControlFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return runControlFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return runControlFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return runControlFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func executionGateFlags(args []string) (executionGateFlagSet, error) {
	flags := executionGateFlagSet{}
	usage := "usage: areaflow run execution-gate <run-id> [--json] [--capability CAP]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--capability":
			if i+1 >= len(args) {
				return executionGateFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.capabilities = append(flags.capabilities, args[i+1])
			i++
		default:
			return executionGateFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func executionPlanFlags(args []string) (executionPlanFlagSet, error) {
	flags := executionPlanFlagSet{}
	usage := "usage: areaflow run execution-plan <run-id> [--json]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		default:
			return executionPlanFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func projectWriteDesignGateFlags(args []string) (projectWriteDesignGateFlagSet, error) {
	flags := projectWriteDesignGateFlagSet{}
	usage := "usage: areaflow run project-write-design-gate <run-id> [--json]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		default:
			return projectWriteDesignGateFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func managedGeneratedWriteGateFlags(args []string) (managedGeneratedWriteGateFlagSet, error) {
	flags := managedGeneratedWriteGateFlagSet{}
	usage := "usage: areaflow run managed-generated-write-gate <run-id> [--json]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		default:
			return managedGeneratedWriteGateFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func artifactArchivePreviewFlags(args []string) (artifactArchivePreviewFlagSet, error) {
	flags := artifactArchivePreviewFlagSet{limit: 100}
	usage := "usage: areaflow artifact archive-preview <project> [--json] [--retention-class CLASS] [--limit N] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--retention-class":
			if i+1 >= len(args) {
				return artifactArchivePreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.retentionClass = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return artifactArchivePreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.Atoi(args[i+1])
			if err != nil || value <= 0 {
				return artifactArchivePreviewFlagSet{}, fmt.Errorf("limit must be a positive integer")
			}
			flags.limit = value
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return artifactArchivePreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return artifactArchivePreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return artifactArchivePreviewFlagSet{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return artifactArchivePreviewFlagSet{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func gateListFlags(args []string) (gateListFlagSet, error) {
	flags := gateListFlagSet{limit: 10}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--limit":
			if i+1 >= len(args) {
				return gateListFlagSet{}, fmt.Errorf("usage: areaflow workflow gate list <project> <label> [--json] [--limit N]")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return gateListFlagSet{}, fmt.Errorf("gate result limit must be a positive integer")
			}
			flags.limit = limit
			i++
		default:
			return gateListFlagSet{}, fmt.Errorf("usage: areaflow workflow gate list <project> <label> [--json] [--limit N]")
		}
	}
	return flags, nil
}

func eventLimit(args []string) (int, error) {
	if len(args) == 0 {
		return 10, nil
	}
	if len(args) != 2 || args[0] != "--limit" {
		return 0, fmt.Errorf("usage: areaflow project events <id> [--limit N]")
	}
	limit, err := strconv.Atoi(args[1])
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("event limit must be a positive integer")
	}
	return limit, nil
}

type statusProjectionFlagSet struct {
	json  bool
	limit int
}

func statusProjectionFlags(args []string) (statusProjectionFlagSet, error) {
	flags := statusProjectionFlagSet{limit: 20}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--limit":
			if i+1 >= len(args) {
				return statusProjectionFlagSet{}, fmt.Errorf("usage: areaflow project status-projections <id> [--json] [--limit N]")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return statusProjectionFlagSet{}, fmt.Errorf("status projection limit must be a positive integer")
			}
			flags.limit = limit
			i++
		default:
			return statusProjectionFlagSet{}, fmt.Errorf("usage: areaflow project status-projections <id> [--json] [--limit N]")
		}
	}
	return flags, nil
}

func outputJSON(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && args[0] == "--json" {
		return true, nil
	}
	return false, fmt.Errorf("usage: --json is the only supported extra flag")
}

func backupScopeFlagsFromArgs(args []string, commandName string) (backupScopeFlags, error) {
	flags := backupScopeFlags{}
	usage := fmt.Sprintf("usage: %s [--project KEY] [--json]", commandName)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--project":
			if i+1 >= len(args) {
				return backupScopeFlags{}, fmt.Errorf("%s", usage)
			}
			flags.projectKey = args[i+1]
			i++
		default:
			return backupScopeFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func restorePlanFlagsFromArgs(args []string) (bool, string, error) {
	flags, err := backupScopeFlagsFromArgs(args, "areaflow backup restore-plan")
	return flags.json, flags.projectKey, err
}

func releaseScopeFlagsFromArgs(args []string, commandName string) (backupScopeFlags, error) {
	return backupScopeFlagsFromArgs(args, commandName)
}

func releaseExceptionMigrationFlagsFromArgs(args []string, commandName string) (releaseExceptionMigrationFlags, error) {
	flags := releaseExceptionMigrationFlags{}
	usage := fmt.Sprintf("usage: %s --actor ACTOR --reason TEXT [--json]", commandName)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--actor", "--reason":
			if i+1 >= len(args) {
				return releaseExceptionMigrationFlags{}, fmt.Errorf("%s", usage)
			}
			if args[i] == "--actor" {
				flags.actor = strings.TrimSpace(args[i+1])
			} else {
				flags.reason = strings.TrimSpace(args[i+1])
			}
			i++
		default:
			return releaseExceptionMigrationFlags{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.actor == "" || flags.reason == "" {
		return releaseExceptionMigrationFlags{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func releaseExceptionCommandFlagsFromArgs(args []string, commandName string, request bool) (releaseExceptionCommandFlags, error) {
	flags := releaseExceptionCommandFlags{}
	usage := fmt.Sprintf("usage: %s --project KEY --exception-key KEY --actor ACTOR --reason TEXT [--owner OWNER] [--review-at RFC3339] [--expires-at RFC3339] [--idempotency-key KEY] [--json]", commandName)
	for i := 0; i < len(args); i++ {
		name := args[i]
		if name == "--json" {
			flags.json = true
			continue
		}
		if i+1 >= len(args) {
			return releaseExceptionCommandFlags{}, fmt.Errorf("%s", usage)
		}
		value := strings.TrimSpace(args[i+1])
		i++
		switch name {
		case "--project":
			flags.projectKey = value
		case "--exception-key":
			flags.exceptionKey = value
		case "--actor":
			flags.actor = value
		case "--reason":
			flags.reason = value
		case "--owner":
			flags.owner = value
		case "--idempotency-key":
			flags.idempotencyKey = value
		case "--review-at", "--expires-at":
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return releaseExceptionCommandFlags{}, fmt.Errorf("%s: invalid %s: %w", usage, name, err)
			}
			if name == "--review-at" {
				flags.reviewAt = &parsed
			} else {
				flags.expiresAt = &parsed
			}
		default:
			return releaseExceptionCommandFlags{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.projectKey == "" || flags.exceptionKey == "" || flags.actor == "" || flags.reason == "" {
		return releaseExceptionCommandFlags{}, fmt.Errorf("%s", usage)
	}
	if !request && (flags.owner != "" || flags.reviewAt != nil || flags.expiresAt != nil) {
		return releaseExceptionCommandFlags{}, fmt.Errorf("%s", usage)
	}
	return flags, nil
}

func executionForwardingV1CommandPreviewFlagsFromArgs(args []string) (executionForwardingV1CommandPreviewFlags, error) {
	flags := executionForwardingV1CommandPreviewFlags{}
	usage := "usage: areaflow project execution-forwarding-v1-command-preview <id> --task-type <type> [--json]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--task-type":
			if i+1 >= len(args) {
				return executionForwardingV1CommandPreviewFlags{}, fmt.Errorf(usage)
			}
			flags.taskType = strings.TrimSpace(args[i+1])
			i++
		default:
			return executionForwardingV1CommandPreviewFlags{}, fmt.Errorf(usage)
		}
	}
	if flags.taskType == "" {
		return executionForwardingV1CommandPreviewFlags{}, fmt.Errorf(usage)
	}
	return flags, nil
}

func executionForwardingV1ApplyPacketFlagsFromArgs(args []string) (executionForwardingV1ApplyPacketFlags, error) {
	flags := executionForwardingV1ApplyPacketFlags{}
	usage := "usage: areaflow project execution-forwarding-v1-apply-packet <id> [--json] [--explicit-approval] [--approval-id ID] [--approval-actor ACTOR] [--approval-reason TEXT] [--legacy-non-write-proof-id ID] [--rollback-plan-id ID] [--protected-path-fingerprint-id ID] [--idempotency-key KEY] [--audit-correlation-id ID]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalID = args[i+1]
			i++
		case "--approval-actor":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		case "--legacy-non-write-proof-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.legacyNonWriteProofID = args[i+1]
			i++
		case "--rollback-plan-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackPlanID = args[i+1]
			i++
		case "--protected-path-fingerprint-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathFingerprintID = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--audit-correlation-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.auditCorrelationID = args[i+1]
			i++
		default:
			return executionForwardingV1ApplyPacketFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func executionForwardingV1ApplyGateFlagsFromArgs(args []string) (executionForwardingV1ApplyGateFlags, error) {
	flags := executionForwardingV1ApplyGateFlags{}
	usage := "usage: areaflow project execution-forwarding-v1-apply-gate <id> [--json] [--allowed-task-types a,b] [--approval-id ID] [--approval-scope SCOPE] [--readiness-snapshot-hash HASH] [--expected-shim-lifecycle-state STATE] [--legacy-non-write-proof-id ID] [--rollback-plan-id ID] [--protected-path-fingerprint-id ID] [--failure-mode fail_closed] [--idempotency-key KEY] [--audit-correlation-id ID] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--allowed-task-types":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.allowedTaskTypes = commaSeparatedList(args[i+1])
			i++
		case "--approval-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalID = args[i+1]
			i++
		case "--approval-scope":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalScope = args[i+1]
			i++
		case "--readiness-snapshot-hash":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.readinessSnapshotHash = args[i+1]
			i++
		case "--expected-shim-lifecycle-state":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.expectedShimLifecycleState = args[i+1]
			i++
		case "--legacy-non-write-proof-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.legacyNonWriteProofID = args[i+1]
			i++
		case "--rollback-plan-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackPlanID = args[i+1]
			i++
		case "--protected-path-fingerprint-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathFingerprintID = args[i+1]
			i++
		case "--failure-mode":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.failureMode = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--audit-correlation-id":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.auditCorrelationID = args[i+1]
			i++
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-actor":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		default:
			return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func shimApplyPacketFlagsFromArgs(args []string) (shimApplyPacketFlags, error) {
	flags := shimApplyPacketFlags{}
	usage := "usage: areaflow project shim-apply-packet <id> [--json] [--explicit-approval] [--approval-id ID] [--approval-actor ACTOR] [--approval-reason TEXT] [--status-projection-packet-id ID] [--status-projection-gate-id ID] [--read-only-smoke-evidence-id ID] [--dirty-worktree-review-id ID] [--protected-path-fingerprint-id ID] [--rollback-plan-id ID] [--idempotency-key KEY] [--audit-correlation-id ID]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalID = args[i+1]
			i++
		case "--approval-actor":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		case "--status-projection-packet-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.statusProjectionPacketID = args[i+1]
			i++
		case "--status-projection-gate-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.statusProjectionGateID = args[i+1]
			i++
		case "--read-only-smoke-evidence-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.readOnlySmokeEvidenceID = args[i+1]
			i++
		case "--dirty-worktree-review-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.dirtyWorktreeReviewID = args[i+1]
			i++
		case "--protected-path-fingerprint-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathFingerprintID = args[i+1]
			i++
		case "--rollback-plan-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackPlanID = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--audit-correlation-id":
			if i+1 >= len(args) {
				return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.auditCorrelationID = args[i+1]
			i++
		default:
			return shimApplyPacketFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func shimApplyGateFlagsFromArgs(args []string) (shimApplyGateFlags, error) {
	flags := shimApplyGateFlags{}
	usage := "usage: areaflow project shim-apply-gate <id> [--json] [--allowed-files a,b] [--approval-id ID] [--approval-scope SCOPE] [--authorization-snapshot-hash HASH] [--expected-authorization-mode MODE] [--status-projection-packet-id ID] [--status-projection-gate-id ID] [--read-only-smoke-evidence-id ID] [--dirty-worktree-review-id ID] [--protected-path-fingerprint-id ID] [--rollback-plan-id ID] [--failure-mode fail_closed] [--idempotency-key KEY] [--audit-correlation-id ID] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--allowed-files":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.allowedFiles = commaSeparatedList(args[i+1])
			i++
		case "--approval-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalID = args[i+1]
			i++
		case "--approval-scope":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalScope = args[i+1]
			i++
		case "--authorization-snapshot-hash":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.authorizationSnapshotHash = args[i+1]
			i++
		case "--expected-authorization-mode":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.expectedAuthorizationMode = args[i+1]
			i++
		case "--status-projection-packet-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.statusProjectionPacketID = args[i+1]
			i++
		case "--status-projection-gate-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.statusProjectionGateID = args[i+1]
			i++
		case "--read-only-smoke-evidence-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.readOnlySmokeEvidenceID = args[i+1]
			i++
		case "--dirty-worktree-review-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.dirtyWorktreeReviewID = args[i+1]
			i++
		case "--protected-path-fingerprint-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathFingerprintID = args[i+1]
			i++
		case "--rollback-plan-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackPlanID = args[i+1]
			i++
		case "--failure-mode":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.failureMode = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--audit-correlation-id":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.auditCorrelationID = args[i+1]
			i++
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-actor":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		default:
			return shimApplyGateFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func shimApplyFlagsFromArgs(args []string) (shimApplyGateFlags, error) {
	flags, err := shimApplyGateFlagsFromArgs(args)
	if err != nil {
		return shimApplyGateFlags{}, fmt.Errorf("usage: areaflow project shim-apply <id> [--json] [--allowed-files a,b] [--approval-id ID] [--approval-scope SCOPE] [--authorization-snapshot-hash HASH] [--expected-authorization-mode MODE] [--status-projection-packet-id ID] [--status-projection-gate-id ID] [--read-only-smoke-evidence-id ID] [--dirty-worktree-review-id ID] [--protected-path-fingerprint-id ID] [--rollback-plan-id ID] [--failure-mode fail_closed] [--idempotency-key KEY] [--audit-correlation-id ID] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]")
	}
	return flags, nil
}

func executionForwardingV1ApplyFlagsFromArgs(args []string) (executionForwardingV1ApplyGateFlags, error) {
	flags, err := executionForwardingV1ApplyGateFlagsFromArgs(args)
	if err != nil {
		return executionForwardingV1ApplyGateFlags{}, fmt.Errorf("usage: areaflow project execution-forwarding-v1-apply <id> [--json] [--allowed-task-types a,b] [--approval-id ID] [--approval-scope SCOPE] [--readiness-snapshot-hash HASH] [--expected-shim-lifecycle-state STATE] [--legacy-non-write-proof-id ID] [--rollback-plan-id ID] [--protected-path-fingerprint-id ID] [--failure-mode fail_closed] [--idempotency-key KEY] [--audit-correlation-id ID] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]")
	}
	return flags, nil
}

func commaSeparatedList(value string) []string {
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

func codexPreviewFlags(args []string) (codexPreviewFlagSet, error) {
	flags := codexPreviewFlagSet{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--command":
			if i+1 >= len(args) {
				return codexPreviewFlagSet{}, fmt.Errorf("usage: areaflow engine codex-preview <project> [--json] [--command COMMAND]")
			}
			flags.command = args[i+1]
			i++
		default:
			return codexPreviewFlagSet{}, fmt.Errorf("usage: areaflow engine codex-preview <project> [--json] [--command COMMAND]")
		}
	}
	return flags, nil
}

type codexPreviewFlagSet struct {
	json    bool
	command string
}

type projectDoctorFlags struct {
	json           bool
	allowNative    bool
	idempotencyKey string
}

type projectImportFlags struct {
	json           bool
	idempotencyKey string
	actor          string
	reason         string
}

type projectBundleFlags struct {
	json  bool
	limit int
}

type projectCutoverReadinessFlags struct {
	json    bool
	limit   int
	version string
}

type projectCutoverApplyFlags struct {
	json           bool
	version        string
	idempotencyKey string
	actor          string
	reason         string
	mode           string
}

type executionForwardingV1CommandPreviewFlags struct {
	json     bool
	taskType string
}

type executionForwardingV1ApplyPacketFlags struct {
	json                       bool
	explicitApproval           bool
	approvalID                 string
	approvalActor              string
	approvalReason             string
	legacyNonWriteProofID      string
	rollbackPlanID             string
	protectedPathFingerprintID string
	idempotencyKey             string
	auditCorrelationID         string
}

type executionForwardingV1ApplyGateFlags struct {
	json                       bool
	allowedTaskTypes           []string
	approvalID                 string
	approvalScope              string
	readinessSnapshotHash      string
	expectedShimLifecycleState string
	legacyNonWriteProofID      string
	rollbackPlanID             string
	protectedPathFingerprintID string
	idempotencyKey             string
	auditCorrelationID         string
	failureMode                string
	explicitApproval           bool
	approvalActor              string
	approvalReason             string
}

type shimApplyPacketFlags struct {
	json                       bool
	explicitApproval           bool
	approvalID                 string
	approvalActor              string
	approvalReason             string
	statusProjectionPacketID   string
	statusProjectionGateID     string
	readOnlySmokeEvidenceID    string
	dirtyWorktreeReviewID      string
	protectedPathFingerprintID string
	rollbackPlanID             string
	idempotencyKey             string
	auditCorrelationID         string
}

type shimApplyGateFlags struct {
	json                       bool
	allowedFiles               []string
	approvalID                 string
	approvalScope              string
	authorizationSnapshotHash  string
	expectedAuthorizationMode  string
	statusProjectionPacketID   string
	statusProjectionGateID     string
	readOnlySmokeEvidenceID    string
	dirtyWorktreeReviewID      string
	protectedPathFingerprintID string
	rollbackPlanID             string
	idempotencyKey             string
	auditCorrelationID         string
	failureMode                string
	explicitApproval           bool
	approvalActor              string
	approvalReason             string
}

type shimReadinessEvidenceFlags struct {
	json           bool
	evidenceKey    string
	status         string
	summary        string
	evidenceURI    string
	idempotencyKey string
	actor          string
	reason         string
}

type operationsSmokeProofFlags struct {
	json           bool
	proofKey       string
	status         string
	summary        string
	evidenceURI    string
	idempotencyKey string
	actor          string
	reason         string
}

type protectedPathProofFlags struct {
	json                bool
	status              string
	summary             string
	evidenceURI         string
	gitStatusOutput     string
	approvalID          string
	allowedPaths        []string
	dirtyOutputHash     string
	reviewer            string
	rollbackEvidenceURI string
	idempotencyKey      string
	actor               string
	reason              string
}

type completionAuditSnapshotFlags struct {
	json                  bool
	releaseCandidateLabel string
	evidenceClass         string
	summary               string
	evidenceURI           string
	reviewDecision        string
	reviewedBy            string
	reviewedAt            time.Time
	idempotencyKey        string
	actor                 string
	reason                string
}

type completionAuditSnapshotReadinessFlags struct {
	json bool
}

type archiveProofFlags struct {
	json                    bool
	status                  string
	facts                   []string
	summary                 string
	evidenceURI             string
	reviewDecision          string
	reviewedBy              string
	reviewedAt              time.Time
	archiveScope            string
	archiveReferenceMode    string
	archiveSourcePaths      []string
	archiveForbiddenActions []string
	archiveRollbackTarget   string
	archiveFailClosed       bool
	idempotencyKey          string
	actor                   string
	reason                  string
}

type shimRetirementProofFlags struct {
	json                        bool
	status                      string
	facts                       []string
	summary                     string
	evidenceURI                 string
	reviewDecision              string
	reviewedBy                  string
	reviewedAt                  time.Time
	shimRetirementScope         string
	shimRetirementPrerequisites []string
	shimRetiredSurfaces         []string
	shimRollbackTarget          string
	shimFailClosed              bool
	shimReopenRequiresApproval  bool
	idempotencyKey              string
	actor                       string
	reason                      string
}

type executionCutoverProofFlags struct {
	json                       bool
	status                     string
	facts                      []string
	summary                    string
	evidenceURI                string
	reviewDecision             string
	reviewedBy                 string
	reviewedAt                 time.Time
	executionCutoverScope      string
	allowedTaskTypes           []string
	forbiddenActions           []string
	rollbackTarget             string
	rollbackMode               string
	failClosed                 bool
	reopenRequiresApproval     bool
	sourceWriteOpen            bool
	generatedRetainedWriteOpen bool
	repairApplyOpen            bool
	checkpointApplyOpen        bool
	engineExecutionOpen        bool
	secretResolveOpen          bool
	networkAPIIntegrationOpen  bool
	publishApplyOpen           bool
	restoreApplyOpen           bool
	idempotencyKey             string
	actor                      string
	reason                     string
}

type validationProofFlags struct {
	json                 bool
	status               string
	facts                []string
	summary              string
	evidenceURI          string
	validationCommands   []string
	validationResultHash string
	validationStartedAt  string
	validationFinishedAt string
	validationScope      string
	idempotencyKey       string
	actor                string
	reason               string
}

type sourceAlignmentProofFlags struct {
	json           bool
	status         string
	facts          []string
	summary        string
	evidenceURI    string
	idempotencyKey string
	actor          string
	reason         string
}

type taskMatrixProofFlags struct {
	json                                  bool
	status                                string
	facts                                 []string
	summary                               string
	evidenceURI                           string
	taskMatrixSourceSetHash               string
	taskBacklogHash                       string
	taskStatusAuditHash                   string
	plannedV1RequiredTaskCount            int64
	plannedV1RequiredTaskCountSet         bool
	missingEvidenceV1RequiredTaskCount    int64
	missingEvidenceV1RequiredTaskCountSet bool
	blockedV1RequiredTaskCount            int64
	blockedV1RequiredTaskCountSet         bool
	idempotencyKey                        string
	actor                                 string
	reason                                string
}

type securityClosureProofFlags struct {
	json           bool
	status         string
	facts          []string
	summary        string
	evidenceURI    string
	idempotencyKey string
	actor          string
	reason         string
}

type backupRestoreProofFlags struct {
	json                                        bool
	status                                      string
	facts                                       []string
	summary                                     string
	evidenceURI                                 string
	backupManifestHash                          string
	backupManifestStatus                        string
	backupManifestProjectCount                  *int64
	backupManifestTableCount                    *int64
	restorePlanStatus                           string
	restorePlanScope                            string
	restorePlanProjectKey                       string
	restorePlanManifestHash                     string
	restorePlanItemCount                        *int64
	artifactIntegrityStatus                     string
	artifactIntegrityCheckedCount               *int64
	artifactIntegrityFailedCount                *int64
	artifactArchivePreviewStatus                string
	artifactArchivePreviewTotalArtifacts        *int64
	artifactArchivePreviewExternalRefs          *int64
	artifactArchivePreviewNeedsPolicy           *int64
	artifactArchivePreviewProjectWriteAttempted *bool
	artifactArchivePreviewStorageWriteAttempted *bool
	artifactArchivePreviewDeleteAttempted       *bool
	idempotencyKey                              string
	actor                                       string
	reason                                      string
}

type releasePackagingProofFlags struct {
	json           bool
	status         string
	facts          []string
	summary        string
	evidenceURI    string
	idempotencyKey string
	actor          string
	reason         string
}

type statusProjectionApplyFlags struct {
	json                           bool
	targetURI                      string
	idempotencyKey                 string
	actor                          string
	reason                         string
	expectedBeforeExists           *bool
	expectedBeforeSHA256           string
	expectedBeforeSizeBytes        *int64
	sourceHash                     string
	schemaURI                      string
	validatorPreflight             string
	protectedPathCheck             string
	protectedPathFingerprintSHA256 string
	rollbackAction                 string
	acceptedPreimageSchemaStatus   string
	explicitApproval               bool
	approvalActor                  string
	approvalReason                 string
}

type statusProjectionAuthorizationFlags struct {
	json      bool
	targetURI string
}

type statusProjectionApplyPacketFlags struct {
	json             bool
	targetURI        string
	explicitApproval bool
	approvalActor    string
	approvalReason   string
}

type statusProjectionApplyGateFlags struct {
	json                           bool
	targetURI                      string
	expectedBeforeExists           *bool
	expectedBeforeSHA256           string
	expectedBeforeSizeBytes        *int64
	sourceHash                     string
	schemaURI                      string
	validatorPreflight             string
	protectedPathCheck             string
	protectedPathFingerprintSHA256 string
	rollbackAction                 string
	acceptedPreimageSchemaStatus   string
	explicitApproval               bool
	approvalActor                  string
	approvalReason                 string
}

func bundleFlags(args []string) (projectBundleFlags, error) {
	flags := projectBundleFlags{limit: 10}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--limit":
			if i+1 >= len(args) {
				return projectBundleFlags{}, fmt.Errorf("usage: areaflow project verify-bundle <id> [--json] [--limit N]")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return projectBundleFlags{}, fmt.Errorf("bundle event limit must be a positive integer")
			}
			flags.limit = limit
			i++
		default:
			return projectBundleFlags{}, fmt.Errorf("usage: areaflow project verify-bundle <id> [--json] [--limit N]")
		}
	}
	return flags, nil
}

func cutoverReadinessFlags(args []string) (projectCutoverReadinessFlags, error) {
	flags := projectCutoverReadinessFlags{limit: 10}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--limit":
			if i+1 >= len(args) {
				return projectCutoverReadinessFlags{}, fmt.Errorf("usage: areaflow project cutover-readiness <id> --version <label> [--json] [--limit N]")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return projectCutoverReadinessFlags{}, fmt.Errorf("cutover readiness event limit must be a positive integer")
			}
			flags.limit = limit
			i++
		case "--version":
			if i+1 >= len(args) {
				return projectCutoverReadinessFlags{}, fmt.Errorf("usage: areaflow project cutover-readiness <id> --version <label> [--json] [--limit N]")
			}
			flags.version = args[i+1]
			i++
		default:
			return projectCutoverReadinessFlags{}, fmt.Errorf("usage: areaflow project cutover-readiness <id> --version <label> [--json] [--limit N]")
		}
	}
	if flags.version == "" {
		return projectCutoverReadinessFlags{}, fmt.Errorf("missing workflow version: use --version <label>")
	}
	return flags, nil
}

func cutoverApplyFlags(args []string) (projectCutoverApplyFlags, error) {
	flags := projectCutoverApplyFlags{}
	usage := "usage: areaflow project cutover-apply <id> --version <label> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] [--mode authoring_cutover]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--version":
			if i+1 >= len(args) {
				return projectCutoverApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.version = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return projectCutoverApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return projectCutoverApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return projectCutoverApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--mode":
			if i+1 >= len(args) {
				return projectCutoverApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.mode = args[i+1]
			i++
		default:
			return projectCutoverApplyFlags{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.version == "" {
		return projectCutoverApplyFlags{}, fmt.Errorf("missing workflow version: use --version <label>")
	}
	return flags, nil
}

func shimReadinessEvidenceFlagsFromArgs(args []string) (shimReadinessEvidenceFlags, error) {
	flags := shimReadinessEvidenceFlags{status: "pass"}
	usage := "usage: areaflow project shim-readiness-evidence <id> --key <evidence-key> [--status pass|blocked] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--key":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceKey = args[i+1]
			i++
		case "--status":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--summary":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return shimReadinessEvidenceFlags{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.evidenceKey == "" {
		return shimReadinessEvidenceFlags{}, fmt.Errorf("missing evidence key: use --key <evidence-key>")
	}
	return flags, nil
}

func operationsSmokeProofFlagsFromArgs(args []string) (operationsSmokeProofFlags, error) {
	flags := operationsSmokeProofFlags{status: "pass"}
	usage := "usage: areaflow ops smoke-proof record <project> --key <proof-key> [--status pass|blocked] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary and evidence-uri required for pass)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--key":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.proofKey = args[i+1]
			i++
		case "--status":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--summary":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return operationsSmokeProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	if flags.proofKey == "" {
		return operationsSmokeProofFlags{}, fmt.Errorf("missing proof key: use --key <proof-key>")
	}
	return flags, nil
}

func protectedPathProofFlagsFromArgs(args []string) (protectedPathProofFlags, error) {
	flags := protectedPathProofFlags{status: "clean"}
	usage := "usage: areaflow completion protected-path-proof record <project> --status clean|authorized|dirty|blocked [--summary TEXT] [--evidence-uri URI] [--git-status-output TEXT] [--approval-id ID] [--allowed-path PATH...] [--dirty-output-hash SHA256] [--reviewer TEXT] [--rollback-evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary and evidence-uri required for clean|authorized; approval-id, allowed-path, git-status-output, dirty-output-hash, reviewer and rollback-evidence-uri required for authorized)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--summary":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--git-status-output":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.gitStatusOutput = args[i+1]
			i++
		case "--approval-id":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalID = args[i+1]
			i++
		case "--allowed-path":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.allowedPaths = append(flags.allowedPaths, args[i+1])
			i++
		case "--dirty-output-hash":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.dirtyOutputHash = args[i+1]
			i++
		case "--reviewer":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewer = args[i+1]
			i++
		case "--rollback-evidence-uri":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackEvidenceURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return protectedPathProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func completionAuditSnapshotFlagsFromArgs(args []string) (completionAuditSnapshotFlags, error) {
	flags := completionAuditSnapshotFlags{releaseCandidateLabel: "v1.0-candidate", evidenceClass: "fixture"}
	usage := "usage: areaflow completion audit-snapshot record <project> --release-candidate LABEL [--evidence-class fixture|release_candidate] [--evidence-uri URI] [--summary TEXT] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--release-candidate":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.releaseCandidateLabel = args[i+1]
			i++
		case "--evidence-class":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceClass = args[i+1]
			i++
		case "--summary":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--review-decision":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewDecision = args[i+1]
			i++
		case "--reviewed-by":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewedBy = args[i+1]
			i++
		case "--reviewed-at":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			parsed, err := time.Parse(time.RFC3339, args[i+1])
			if err != nil {
				return completionAuditSnapshotFlags{}, fmt.Errorf("reviewed-at must be RFC3339: %w", err)
			}
			flags.reviewedAt = parsed
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return completionAuditSnapshotFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func completionAuditSnapshotReadinessFlagsFromArgs(args []string) (completionAuditSnapshotReadinessFlags, error) {
	flags := completionAuditSnapshotReadinessFlags{}
	usage := "usage: areaflow completion audit-snapshot readiness <project> [--json]"
	for _, arg := range args {
		switch arg {
		case "--json":
			flags.json = true
		default:
			return completionAuditSnapshotReadinessFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func archiveProofFlagsFromArgs(args []string) (archiveProofFlags, error) {
	flags := archiveProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion archive-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--archive-scope SCOPE] [--archive-reference-mode MODE] [--archive-source-path PATH...] [--archive-source-paths a,b] [--archive-forbidden-action ACTION...] [--archive-forbidden-actions a,b] [--archive-rollback-target TARGET] [--archive-fail-closed] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, release-candidate evidence-uri, approved review metadata and archive scope binding required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--archive-fail-closed":
			flags.archiveFailClosed = true
		case "--status":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--review-decision":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewDecision = args[i+1]
			i++
		case "--reviewed-by":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewedBy = args[i+1]
			i++
		case "--reviewed-at":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			parsed, err := time.Parse(time.RFC3339, args[i+1])
			if err != nil {
				return archiveProofFlags{}, fmt.Errorf("reviewed-at must be RFC3339: %w", err)
			}
			flags.reviewedAt = parsed
			i++
		case "--archive-scope":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveScope = args[i+1]
			i++
		case "--archive-reference-mode":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveReferenceMode = args[i+1]
			i++
		case "--archive-source-path":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveSourcePaths = append(flags.archiveSourcePaths, args[i+1])
			i++
		case "--archive-source-paths":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveSourcePaths = append(flags.archiveSourcePaths, commaSeparatedList(args[i+1])...)
			i++
		case "--archive-forbidden-action":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveForbiddenActions = append(flags.archiveForbiddenActions, args[i+1])
			i++
		case "--archive-forbidden-actions":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveForbiddenActions = append(flags.archiveForbiddenActions, commaSeparatedList(args[i+1])...)
			i++
		case "--archive-rollback-target":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.archiveRollbackTarget = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return archiveProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return archiveProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func shimRetirementProofFlagsFromArgs(args []string) (shimRetirementProofFlags, error) {
	flags := shimRetirementProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion shim-retirement-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--shim-retirement-scope SCOPE] [--shim-prerequisite KEY...] [--shim-prerequisites a,b] [--shim-retired-surface SURFACE...] [--shim-retired-surfaces a,b] [--shim-rollback-target TARGET] [--shim-fail-closed] [--shim-reopen-requires-approval] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, release-candidate evidence-uri, approved review metadata and shim retirement scope binding required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--shim-fail-closed":
			flags.shimFailClosed = true
		case "--shim-reopen-requires-approval":
			flags.shimReopenRequiresApproval = true
		case "--status":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--review-decision":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewDecision = args[i+1]
			i++
		case "--reviewed-by":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewedBy = args[i+1]
			i++
		case "--reviewed-at":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			parsed, err := time.Parse(time.RFC3339, args[i+1])
			if err != nil {
				return shimRetirementProofFlags{}, fmt.Errorf("reviewed-at must be RFC3339: %w", err)
			}
			flags.reviewedAt = parsed
			i++
		case "--shim-retirement-scope":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.shimRetirementScope = args[i+1]
			i++
		case "--shim-prerequisite":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.shimRetirementPrerequisites = append(flags.shimRetirementPrerequisites, args[i+1])
			i++
		case "--shim-prerequisites":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.shimRetirementPrerequisites = append(flags.shimRetirementPrerequisites, commaSeparatedList(args[i+1])...)
			i++
		case "--shim-retired-surface":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.shimRetiredSurfaces = append(flags.shimRetiredSurfaces, args[i+1])
			i++
		case "--shim-retired-surfaces":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.shimRetiredSurfaces = append(flags.shimRetiredSurfaces, commaSeparatedList(args[i+1])...)
			i++
		case "--shim-rollback-target":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.shimRollbackTarget = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return shimRetirementProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func executionCutoverProofFlagsFromArgs(args []string) (executionCutoverProofFlags, error) {
	flags := executionCutoverProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion execution-cutover-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--review-decision approved] [--reviewed-by ACTOR] [--reviewed-at RFC3339] [--execution-cutover-scope SCOPE] [--allowed-task-type TYPE...] [--allowed-task-types a,b] [--forbidden-action ACTION...] [--forbidden-actions a,b] [--rollback-target TARGET] [--rollback-mode MODE] [--fail-closed] [--reopen-requires-approval] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, release-candidate evidence-uri, approved review metadata and scope binding required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--fail-closed":
			flags.failClosed = true
		case "--reopen-requires-approval":
			flags.reopenRequiresApproval = true
		case "--source-write-open":
			flags.sourceWriteOpen = true
		case "--generated-retained-write-open":
			flags.generatedRetainedWriteOpen = true
		case "--repair-apply-open":
			flags.repairApplyOpen = true
		case "--checkpoint-apply-open":
			flags.checkpointApplyOpen = true
		case "--engine-execution-open":
			flags.engineExecutionOpen = true
		case "--secret-resolve-open":
			flags.secretResolveOpen = true
		case "--network-api-integration-open":
			flags.networkAPIIntegrationOpen = true
		case "--publish-apply-open":
			flags.publishApplyOpen = true
		case "--restore-apply-open":
			flags.restoreApplyOpen = true
		case "--status":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--review-decision":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewDecision = args[i+1]
			i++
		case "--reviewed-by":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reviewedBy = args[i+1]
			i++
		case "--reviewed-at":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			parsed, err := time.Parse(time.RFC3339, args[i+1])
			if err != nil {
				return executionCutoverProofFlags{}, fmt.Errorf("reviewed-at must be RFC3339: %w", err)
			}
			flags.reviewedAt = parsed
			i++
		case "--execution-cutover-scope":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.executionCutoverScope = args[i+1]
			i++
		case "--allowed-task-type":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.allowedTaskTypes = append(flags.allowedTaskTypes, args[i+1])
			i++
		case "--allowed-task-types":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.allowedTaskTypes = append(flags.allowedTaskTypes, commaSeparatedList(args[i+1])...)
			i++
		case "--forbidden-action":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.forbiddenActions = append(flags.forbiddenActions, args[i+1])
			i++
		case "--forbidden-actions":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.forbiddenActions = append(flags.forbiddenActions, commaSeparatedList(args[i+1])...)
			i++
		case "--rollback-target":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackTarget = args[i+1]
			i++
		case "--rollback-mode":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackMode = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return executionCutoverProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func validationProofFlagsFromArgs(args []string) (validationProofFlags, error) {
	flags := validationProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion validation-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--validation-command CMD...] [--validation-result-hash SHA256] [--validation-started-at RFC3339] [--validation-finished-at RFC3339] [--validation-scope SCOPE] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and validation binding fields required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--validation-command":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validationCommands = append(flags.validationCommands, args[i+1])
			i++
		case "--validation-result-hash":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validationResultHash = args[i+1]
			i++
		case "--validation-started-at":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validationStartedAt = args[i+1]
			i++
		case "--validation-finished-at":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validationFinishedAt = args[i+1]
			i++
		case "--validation-scope":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validationScope = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return validationProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return validationProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func sourceAlignmentProofFlagsFromArgs(args []string) (sourceAlignmentProofFlags, error) {
	flags := sourceAlignmentProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion source-alignment-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and current source binding required for complete; binding is collected automatically)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return sourceAlignmentProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func taskMatrixProofFlagsFromArgs(args []string) (taskMatrixProofFlags, error) {
	flags := taskMatrixProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion task-matrix-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--source-set-hash SHA256] [--backlog-hash SHA256] [--task-status-audit-hash SHA256] [--planned-v1-required-task-count N] [--missing-evidence-v1-required-task-count N] [--blocked-v1-required-task-count N] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and binding required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--source-set-hash":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.taskMatrixSourceSetHash = args[i+1]
			i++
		case "--task-backlog-hash", "--backlog-hash":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.taskBacklogHash = args[i+1]
			i++
		case "--task-status-audit-hash":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.taskStatusAuditHash = args[i+1]
			i++
		case "--planned-v1-required-task-count":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.plannedV1RequiredTaskCount = value
			flags.plannedV1RequiredTaskCountSet = true
			i++
		case "--missing-evidence-v1-required-task-count":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.missingEvidenceV1RequiredTaskCount = value
			flags.missingEvidenceV1RequiredTaskCountSet = true
			i++
		case "--blocked-v1-required-task-count":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.blockedV1RequiredTaskCount = value
			flags.blockedV1RequiredTaskCountSet = true
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return taskMatrixProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func securityClosureProofFlagsFromArgs(args []string) (securityClosureProofFlags, error) {
	flags := securityClosureProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion security-closure-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and current binding required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return securityClosureProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func backupRestoreProofFlagsFromArgs(args []string) (backupRestoreProofFlags, error) {
	flags := backupRestoreProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion backup-restore-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] --backup-manifest-hash HASH --backup-manifest-status ready --backup-manifest-project-count N --backup-manifest-table-count N --restore-plan-status ready|needs_attention --restore-plan-scope project --restore-plan-project-key KEY --restore-plan-manifest-hash HASH --restore-plan-item-count N --artifact-integrity-status pass|warn --artifact-integrity-checked-count N --artifact-integrity-failed-count 0 --artifact-archive-preview-status ready|needs_attention --artifact-archive-preview-total-artifacts N --artifact-archive-preview-external-refs N --artifact-archive-preview-needs-policy 0 --artifact-archive-preview-project-write-attempted false --artifact-archive-preview-storage-write-attempted false --artifact-archive-preview-delete-attempted false [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary, evidence-uri and output binding required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--backup-manifest-hash":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.backupManifestHash = args[i+1]
			i++
		case "--backup-manifest-status":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.backupManifestStatus = args[i+1]
			i++
		case "--backup-manifest-project-count":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "backup-manifest-project-count")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.backupManifestProjectCount = &value
			i++
		case "--backup-manifest-table-count":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "backup-manifest-table-count")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.backupManifestTableCount = &value
			i++
		case "--restore-plan-status":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.restorePlanStatus = args[i+1]
			i++
		case "--restore-plan-scope":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.restorePlanScope = args[i+1]
			i++
		case "--restore-plan-project-key":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.restorePlanProjectKey = args[i+1]
			i++
		case "--restore-plan-manifest-hash":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.restorePlanManifestHash = args[i+1]
			i++
		case "--restore-plan-item-count":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "restore-plan-item-count")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.restorePlanItemCount = &value
			i++
		case "--artifact-integrity-status":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.artifactIntegrityStatus = args[i+1]
			i++
		case "--artifact-integrity-checked-count":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "artifact-integrity-checked-count")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactIntegrityCheckedCount = &value
			i++
		case "--artifact-integrity-failed-count":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "artifact-integrity-failed-count")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactIntegrityFailedCount = &value
			i++
		case "--artifact-archive-preview-status":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.artifactArchivePreviewStatus = args[i+1]
			i++
		case "--artifact-archive-preview-total-artifacts":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "artifact-archive-preview-total-artifacts")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactArchivePreviewTotalArtifacts = &value
			i++
		case "--artifact-archive-preview-external-refs":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "artifact-archive-preview-external-refs")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactArchivePreviewExternalRefs = &value
			i++
		case "--artifact-archive-preview-needs-policy":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofInt64Flag(args[i+1], "artifact-archive-preview-needs-policy")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactArchivePreviewNeedsPolicy = &value
			i++
		case "--artifact-archive-preview-project-write-attempted":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofBoolFlag(args[i+1], "artifact-archive-preview-project-write-attempted")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactArchivePreviewProjectWriteAttempted = &value
			i++
		case "--artifact-archive-preview-storage-write-attempted":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofBoolFlag(args[i+1], "artifact-archive-preview-storage-write-attempted")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactArchivePreviewStorageWriteAttempted = &value
			i++
		case "--artifact-archive-preview-delete-attempted":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := backupRestoreProofBoolFlag(args[i+1], "artifact-archive-preview-delete-attempted")
			if err != nil {
				return backupRestoreProofFlags{}, err
			}
			flags.artifactArchivePreviewDeleteAttempted = &value
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return backupRestoreProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func backupRestoreProofInt64Flag(raw string, name string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func backupRestoreProofBoolFlag(raw string, name string) (bool, error) {
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return value, nil
}

func releasePackagingProofFlagsFromArgs(args []string) (releasePackagingProofFlags, error) {
	flags := releasePackagingProofFlags{status: "incomplete"}
	usage := "usage: areaflow completion release-packaging-proof record <project> --status complete|incomplete|blocked --fact KEY [--fact KEY...] [--summary TEXT] [--evidence-uri URI] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] (summary and evidence-uri required for complete)"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--status":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.status = args[i+1]
			i++
		case "--fact":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.facts = append(flags.facts, args[i+1])
			i++
		case "--summary":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.summary = args[i+1]
			i++
		case "--evidence-uri":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.evidenceURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return releasePackagingProofFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func isHelpFlag(value string) bool {
	return value == "-h" || value == "--help" || value == "help"
}

func projectSubcommandUsage(command string) (string, bool) {
	switch command {
	case "status-projection-authorization":
		return "usage: areaflow project status-projection-authorization <id> [--target .areaflow/status.json] [--json]", true
	case "status-projection-apply-packet":
		return "usage: areaflow project status-projection-apply-packet <id> [--target .areaflow/status.json] [--json] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]", true
	case "status-projection-apply-gate":
		return "usage: areaflow project status-projection-apply-gate <id> [--target .areaflow/status.json] [--json] [--expected-before-exists true|false] [--expected-before-sha256 HASH] [--expected-before-size N] [--source-hash HASH] [--schema-uri URI] [--validator-preflight CMD] [--protected-path-check CMD] [--protected-path-fingerprint-sha256 HASH] [--rollback-action TEXT] [--accept-preimage-schema STATUS] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]", true
	case "status-projection-apply":
		return "usage: areaflow project status-projection-apply <id> [--target .areaflow/status.json] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] [--expected-before-exists true|false] [--expected-before-sha256 HASH] [--expected-before-size N] [--source-hash HASH] [--schema-uri URI] [--validator-preflight CMD] [--protected-path-check CMD] [--protected-path-fingerprint-sha256 HASH] [--rollback-action TEXT] [--accept-preimage-schema STATUS] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]", true
	case "shim-apply-packet":
		return "usage: areaflow project shim-apply-packet <id> [--json] [--explicit-approval] [--approval-id ID] [--approval-actor ACTOR] [--approval-reason TEXT] [--status-projection-packet-id ID] [--status-projection-gate-id ID] [--read-only-smoke-evidence-id ID] [--dirty-worktree-review-id ID] [--protected-path-fingerprint-id ID] [--rollback-plan-id ID] [--idempotency-key KEY] [--audit-correlation-id ID]", true
	case "shim-apply-gate":
		return "usage: areaflow project shim-apply-gate <id> [--json] [--allowed-files a,b] [--approval-id ID] [--approval-scope SCOPE] [--authorization-snapshot-hash HASH] [--expected-authorization-mode MODE] [--status-projection-packet-id ID] [--status-projection-gate-id ID] [--read-only-smoke-evidence-id ID] [--dirty-worktree-review-id ID] [--protected-path-fingerprint-id ID] [--rollback-plan-id ID] [--failure-mode fail_closed] [--idempotency-key KEY] [--audit-correlation-id ID] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]", true
	case "shim-apply":
		return "usage: areaflow project shim-apply <id> [--json] [--allowed-files a,b] [--approval-id ID] [--approval-scope SCOPE] [--authorization-snapshot-hash HASH] [--expected-authorization-mode MODE] [--status-projection-packet-id ID] [--status-projection-gate-id ID] [--read-only-smoke-evidence-id ID] [--dirty-worktree-review-id ID] [--protected-path-fingerprint-id ID] [--rollback-plan-id ID] [--failure-mode fail_closed] [--idempotency-key KEY] [--audit-correlation-id ID] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]", true
	default:
		return "", false
	}
}

func statusProjectionApplyFlagsFromArgs(args []string) (statusProjectionApplyFlags, error) {
	flags := statusProjectionApplyFlags{targetURI: ".areaflow/status.json"}
	usage := "usage: areaflow project status-projection-apply <id> [--target .areaflow/status.json] [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT] [--expected-before-exists true|false] [--expected-before-sha256 HASH] [--expected-before-size N] [--source-hash HASH] [--schema-uri URI] [--validator-preflight CMD] [--protected-path-check CMD] [--protected-path-fingerprint-sha256 HASH] [--rollback-action TEXT] [--accept-preimage-schema STATUS] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.targetURI = args[i+1]
			i++
		case "--idempotency-key":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		case "--expected-before-exists":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseBool(args[i+1])
			if err != nil {
				return statusProjectionApplyFlags{}, fmt.Errorf("expected-before-exists must be true or false")
			}
			flags.expectedBeforeExists = &value
			i++
		case "--expected-before-sha256":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.expectedBeforeSHA256 = args[i+1]
			i++
		case "--expected-before-size":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || value < 0 {
				return statusProjectionApplyFlags{}, fmt.Errorf("expected-before-size must be a non-negative integer")
			}
			flags.expectedBeforeSizeBytes = &value
			i++
		case "--source-hash":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.sourceHash = args[i+1]
			i++
		case "--schema-uri":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.schemaURI = args[i+1]
			i++
		case "--validator-preflight":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validatorPreflight = args[i+1]
			i++
		case "--protected-path-check":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathCheck = args[i+1]
			i++
		case "--protected-path-fingerprint-sha256":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathFingerprintSHA256 = args[i+1]
			i++
		case "--rollback-action":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackAction = args[i+1]
			i++
		case "--accept-preimage-schema":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.acceptedPreimageSchemaStatus = args[i+1]
			i++
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-actor":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		default:
			return statusProjectionApplyFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func statusProjectionAuthorizationFlagsFromArgs(args []string) (statusProjectionAuthorizationFlags, error) {
	flags := statusProjectionAuthorizationFlags{targetURI: ".areaflow/status.json"}
	usage := "usage: areaflow project status-projection-authorization <id> [--target .areaflow/status.json] [--json]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target":
			if i+1 >= len(args) {
				return statusProjectionAuthorizationFlags{}, fmt.Errorf("%s", usage)
			}
			flags.targetURI = args[i+1]
			i++
		default:
			return statusProjectionAuthorizationFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func statusProjectionApplyPacketFlagsFromArgs(args []string) (statusProjectionApplyPacketFlags, error) {
	flags := statusProjectionApplyPacketFlags{targetURI: ".areaflow/status.json"}
	usage := "usage: areaflow project status-projection-apply-packet <id> [--target .areaflow/status.json] [--json] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target":
			if i+1 >= len(args) {
				return statusProjectionApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.targetURI = args[i+1]
			i++
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-actor":
			if i+1 >= len(args) {
				return statusProjectionApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return statusProjectionApplyPacketFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		default:
			return statusProjectionApplyPacketFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func statusProjectionApplyGateFlagsFromArgs(args []string) (statusProjectionApplyGateFlags, error) {
	flags := statusProjectionApplyGateFlags{targetURI: ".areaflow/status.json"}
	usage := "usage: areaflow project status-projection-apply-gate <id> [--target .areaflow/status.json] [--json] [--expected-before-exists true|false] [--expected-before-sha256 HASH] [--expected-before-size N] [--source-hash HASH] [--schema-uri URI] [--validator-preflight CMD] [--protected-path-check CMD] [--protected-path-fingerprint-sha256 HASH] [--rollback-action TEXT] [--accept-preimage-schema STATUS] [--explicit-approval] [--approval-actor ACTOR] [--approval-reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--target":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.targetURI = args[i+1]
			i++
		case "--expected-before-exists":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseBool(args[i+1])
			if err != nil {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("expected-before-exists must be true or false")
			}
			flags.expectedBeforeExists = &value
			i++
		case "--expected-before-sha256":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.expectedBeforeSHA256 = args[i+1]
			i++
		case "--expected-before-size":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			value, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || value < 0 {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("expected-before-size must be a non-negative integer")
			}
			flags.expectedBeforeSizeBytes = &value
			i++
		case "--source-hash":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.sourceHash = args[i+1]
			i++
		case "--schema-uri":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.schemaURI = args[i+1]
			i++
		case "--validator-preflight":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.validatorPreflight = args[i+1]
			i++
		case "--protected-path-check":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathCheck = args[i+1]
			i++
		case "--protected-path-fingerprint-sha256":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.protectedPathFingerprintSHA256 = args[i+1]
			i++
		case "--rollback-action":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.rollbackAction = args[i+1]
			i++
		case "--accept-preimage-schema":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.acceptedPreimageSchemaStatus = args[i+1]
			i++
		case "--explicit-approval":
			flags.explicitApproval = true
		case "--approval-actor":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalActor = args[i+1]
			i++
		case "--approval-reason":
			if i+1 >= len(args) {
				return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
			}
			flags.approvalReason = args[i+1]
			i++
		default:
			return statusProjectionApplyGateFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func doctorFlags(args []string) (projectDoctorFlags, error) {
	flags := projectDoctorFlags{}
	usage := "usage: areaflow project doctor <id> [--json] [--allow-native] [--idempotency-key KEY]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--allow-native":
			flags.allowNative = true
		case "--idempotency-key":
			if i+1 >= len(args) {
				return projectDoctorFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		default:
			return projectDoctorFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func importFlags(args []string) (projectImportFlags, error) {
	flags := projectImportFlags{}
	usage := "usage: areaflow project import <id> [--json] [--idempotency-key KEY] [--actor ACTOR] [--reason TEXT]"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.json = true
		case "--idempotency-key":
			if i+1 >= len(args) {
				return projectImportFlags{}, fmt.Errorf("%s", usage)
			}
			flags.idempotencyKey = args[i+1]
			i++
		case "--actor":
			if i+1 >= len(args) {
				return projectImportFlags{}, fmt.Errorf("%s", usage)
			}
			flags.actor = args[i+1]
			i++
		case "--reason":
			if i+1 >= len(args) {
				return projectImportFlags{}, fmt.Errorf("%s", usage)
			}
			flags.reason = args[i+1]
			i++
		default:
			return projectImportFlags{}, fmt.Errorf("%s", usage)
		}
	}
	return flags, nil
}

func (c command) printJSON(value any) error {
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (c command) printProject(record project.Record) {
	fmt.Fprintf(c.stdout, "id: %s\n", record.Key)
	fmt.Fprintf(c.stdout, "name: %s\n", record.Name)
	fmt.Fprintf(c.stdout, "kind: %s\n", record.Kind)
	fmt.Fprintf(c.stdout, "adapter: %s\n", record.Adapter)
	fmt.Fprintf(c.stdout, "workflow_profile: %s\n", record.WorkflowProfile)
	fmt.Fprintf(c.stdout, "root: %s\n", record.RootPath)
	fmt.Fprintf(c.stdout, "default_branch: %s\n", record.DefaultBranch)
}

func (c command) printSummary(summary project.ProjectSummary) {
	fmt.Fprintf(c.stdout, "project summary: %s\n", summary.Project.Key)
	fmt.Fprintf(c.stdout, "name: %s\n", summary.Project.Name)
	fmt.Fprintf(c.stdout, "adapter: %s\n", summary.Project.Adapter)
	fmt.Fprintf(c.stdout, "workflow_profile: %s\n", summary.Project.WorkflowProfile)
	fmt.Fprintf(c.stdout, "root: %s\n", summary.Project.RootPath)
	if summary.HasConfig {
		fmt.Fprintf(c.stdout, "config.protocol_version: %d\n", summary.Config.ProtocolVersion)
		fmt.Fprintf(c.stdout, "config.path: %s\n", summary.Config.ConfigPath)
		fmt.Fprintf(c.stdout, "config.hash: %s\n", summary.Config.ConfigHash)
		fmt.Fprintf(c.stdout, "config.ownership_mode: %s\n", metadataValue(summary.Config.Ownership, "mode"))
		fmt.Fprintf(c.stdout, "config.migration_phase: %s\n", metadataValue(summary.Config.Migration, "phase"))
		fmt.Fprintf(c.stdout, "config.status_export_path: %s\n", metadataValue(summary.Config.StatusExport, "path"))
	}
	fmt.Fprintf(c.stdout, "inventory.versions: %d\n", summary.Inventory.Versions)
	fmt.Fprintf(c.stdout, "inventory.residuals: %d\n", summary.Inventory.Residuals)
	fmt.Fprintf(c.stdout, "inventory.artifacts: %d\n", summary.Inventory.Artifacts)
	fmt.Fprintf(c.stdout, "inventory.import_snapshots: %d\n", summary.Inventory.ImportSnapshots)
	fmt.Fprintf(c.stdout, "inventory.mirror_exports: %d\n", summary.Inventory.MirrorExports)
	if summary.HasImport {
		fmt.Fprintf(c.stdout, "import.source_hash: %s\n", summary.Import.SourceHash)
		fmt.Fprintf(c.stdout, "import.created_at: %s\n", summary.Import.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
		fmt.Fprintf(c.stdout, "import.history_ready_for_diff: %t\n", summary.HasPreviousImport)
		if summary.HasPreviousImport {
			fmt.Fprintf(c.stdout, "import.previous_source_hash: %s\n", summary.PreviousImport.SourceHash)
			fmt.Fprintf(c.stdout, "import.source_hash_changed_since_previous: %t\n", summary.Import.SourceHash != summary.PreviousImport.SourceHash)
		}
		fmt.Fprintf(c.stdout, "import.residual_count: %s\n", metadataValue(summary.Import.Summary, "residual_count"))
		fmt.Fprintf(c.stdout, "import.version_count: %s\n", metadataValue(summary.Import.Summary, "version_count"))
		fmt.Fprintf(c.stdout, "import.v1_execution: %s\n", nestedSummaryValue(summary.Import.Summary, "v1_execution", "done")+"/"+nestedSummaryValue(summary.Import.Summary, "v1_execution", "total"))
	}
	if summary.HasLatestDoctor {
		fmt.Fprintf(c.stdout, "doctor.status: %s\n", metadataString(summary.LatestDoctor.Metadata, "overall_status"))
		fmt.Fprintf(c.stdout, "doctor.stable_status: %s\n", summary.DoctorStatus)
		fmt.Fprintf(c.stdout, "doctor.drift_status: %s\n", summary.DriftStatus)
		fmt.Fprintf(c.stdout, "doctor.config_drift_status: %s\n", summary.ConfigDriftStatus)
		fmt.Fprintf(c.stdout, "doctor.stage_coverage_status: %s\n", summary.StageCoverageStatus)
		fmt.Fprintf(c.stdout, "doctor.native_doctor_status: %s\n", summary.NativeDoctorStatus)
		fmt.Fprintf(c.stdout, "doctor.severity: %s\n", summary.LatestDoctor.Severity)
		fmt.Fprintf(c.stdout, "doctor.created_at: %s\n", summary.LatestDoctor.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
		fmt.Fprintf(c.stdout, "doctor.counts: %s\n", metadataValue(summary.LatestDoctor.Metadata, "counts"))
	}
}

func (c command) printReadiness(readiness project.ProjectReadiness) {
	fmt.Fprintf(c.stdout, "project readiness: %s status=%s\n", readiness.Project.Key, readiness.Status)
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
}

func (c command) printGeneratedWriteReadiness(readiness project.GeneratedWriteReadiness) {
	fmt.Fprintf(c.stdout, "generated write readiness: %s status=%s mode=%s\n", readiness.Project.Key, readiness.Status, readiness.Mode)
	fmt.Fprintf(c.stdout, "ready_for_review: %t\n", readiness.ReadyForReview)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", readiness.ApplyOpen)
	fmt.Fprintf(c.stdout, "real_areamatrix_write_opened: %t\n", readiness.RealAreaMatrixWriteOpened)
	fmt.Fprintf(c.stdout, "generated_only: %t\n", readiness.GeneratedOnly)
	fmt.Fprintf(c.stdout, "required_capabilities: %s\n", strings.Join(readiness.RequiredCapabilities, ","))
	fmt.Fprintf(c.stdout, "allowed_generated_prefixes: %s\n", strings.Join(readiness.AllowedGeneratedPrefixes, ","))
	fmt.Fprintf(c.stdout, "required_write_paths: %s\n", strings.Join(readiness.RequiredWritePaths, ","))
	fmt.Fprintf(c.stdout, "project_config_read: %t\n", readiness.ProjectConfigRead)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", readiness.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", readiness.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", readiness.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", readiness.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", readiness.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", readiness.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", readiness.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", readiness.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", readiness.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", readiness.TaskClaimed)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", readiness.WorkerStarted)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", readiness.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", readiness.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", readiness.ArtifactCreated)
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
	for _, blocker := range readiness.ReviewBlockers {
		fmt.Fprintf(c.stdout, "review_blocker: %s\n", blocker)
	}
	for _, blocker := range readiness.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printGeneratedWriteApplyBetaGate(gate project.GeneratedWriteApplyBetaGate) {
	fmt.Fprintf(c.stdout, "generated write apply beta gate: %s status=%s mode=%s\n", gate.Project.Key, gate.Status, gate.Mode)
	fmt.Fprintf(c.stdout, "readiness.ready_for_review: %t\n", gate.Readiness.ReadyForReview)
	fmt.Fprintf(c.stdout, "approval_required: %t\n", gate.ApprovalRequired)
	fmt.Fprintf(c.stdout, "approval_status: %s\n", gate.ApprovalStatus)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", gate.ApplyOpen)
	fmt.Fprintf(c.stdout, "real_areamatrix_write_opened: %t\n", gate.RealAreaMatrixWriteOpened)
	fmt.Fprintf(c.stdout, "generated_only: %t\n", gate.GeneratedOnly)
	fmt.Fprintf(c.stdout, "required_capabilities: %s\n", strings.Join(gate.RequiredCapabilities, ","))
	fmt.Fprintf(c.stdout, "allowed_generated_prefixes: %s\n", strings.Join(gate.AllowedGeneratedPrefixes, ","))
	fmt.Fprintf(c.stdout, "required_evidence: %s\n", strings.Join(gate.RequiredEvidence, " | "))
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", gate.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", gate.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", gate.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", gate.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", gate.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", gate.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", gate.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", gate.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", gate.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", gate.TaskClaimed)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", gate.WorkerStarted)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", gate.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", gate.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", gate.ArtifactCreated)
	for _, item := range gate.Items {
		if item.ApprovalStatus != "" {
			fmt.Fprintf(c.stdout, "[%s] %s approval=%s: %s\n", item.Status, item.Key, item.ApprovalStatus, item.Message)
			continue
		}
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
}

func (c command) printImportDiff(diff project.ProjectImportDiff) {
	fmt.Fprintf(c.stdout, "project import-diff: %s status=%s\n", diff.Project.Key, diff.Status)
	if diff.Latest.SourceHash != "" {
		fmt.Fprintf(c.stdout, "latest.source_hash: %s\n", diff.Latest.SourceHash)
	}
	if !diff.HasPrevious {
		fmt.Fprintln(c.stdout, "previous: none")
		return
	}
	fmt.Fprintf(c.stdout, "previous.source_hash: %s\n", diff.Previous.SourceHash)
	fmt.Fprintf(c.stdout, "source_changed: %t\n", diff.SourceChanged)
	for _, change := range diff.Changes {
		fmt.Fprintf(c.stdout, "[%s] %s: %s -> %s\n", change.Status, change.Key, change.Previous, change.Latest)
	}
}

func (c command) printVerificationBundle(bundle project.ProjectVerificationBundle) {
	fmt.Fprintf(c.stdout, "project verification bundle: %s status=%s\n", bundle.Project.Key, bundle.Status)
	fmt.Fprintf(c.stdout, "phase_gate.%s: %s\n", bundle.PhaseGate.Name, bundle.PhaseGate.Status)
	if len(bundle.PhaseGate.AcceptedWarnings) > 0 {
		fmt.Fprintf(c.stdout, "phase_gate.accepted_warnings: %s\n", bundle.PhaseGate.AcceptedWarnings)
	}
	if len(bundle.PhaseGate.Blockers) > 0 {
		fmt.Fprintf(c.stdout, "phase_gate.blockers: %s\n", bundle.PhaseGate.Blockers)
	}
	fmt.Fprintf(c.stdout, "readiness.status: %s\n", bundle.Readiness.Status)
	fmt.Fprintf(c.stdout, "import_diff.status: %s\n", bundle.ImportDiff.Status)
	fmt.Fprintf(c.stdout, "events.count: %d\n", len(bundle.Events))
	for _, item := range bundle.Readiness.Items {
		fmt.Fprintf(c.stdout, "readiness.%s: %s\n", item.Key, item.Status)
	}
}

func (c command) printCompatibilityContract(contract project.CompatibilityContract) {
	fmt.Fprintf(c.stdout, "project compatibility: %s status=%s\n", contract.Project.Key, contract.Status)
	for _, command := range contract.Commands {
		fmt.Fprintf(c.stdout, "[%s] %s mode=%s target=%s fallback=%s\n",
			command.Status,
			command.Command,
			command.Mode,
			command.AreaFlowTarget,
			command.Fallback,
		)
		if command.BlockedReason != "" {
			fmt.Fprintf(c.stdout, "  blocked_reason: %s\n", command.BlockedReason)
		}
		if command.Message != "" {
			fmt.Fprintf(c.stdout, "  message: %s\n", command.Message)
		}
	}
}

func (c command) printShimPreview(preview project.ShimPreview) {
	fmt.Fprintf(c.stdout, "shim preview: %s status=%s mode=%s\n", preview.Project.Key, preview.Status, preview.Mode)
	fmt.Fprintf(c.stdout, "planned_files.count: %d\n", len(preview.PlannedFiles))
	for _, file := range preview.PlannedFiles {
		required := "optional"
		if file.Required {
			required = "required"
		}
		fmt.Fprintf(c.stdout, "[%s] %s action=%s boundary=%s\n", required, file.Path, file.Action, file.Boundary)
	}
	fmt.Fprintf(c.stdout, "command_mappings.count: %d\n", len(preview.CommandMappings))
	for _, mapping := range preview.CommandMappings {
		fmt.Fprintf(c.stdout, "[%s] %s mode=%s target=%s fallback=%s\n",
			mapping.Status,
			mapping.Command,
			mapping.Mode,
			mapping.AreaFlowTarget,
			mapping.Fallback,
		)
		if mapping.BlockedReason != "" {
			fmt.Fprintf(c.stdout, "  blocked_reason: %s\n", mapping.BlockedReason)
		}
	}
	fmt.Fprintf(c.stdout, "forbidden_commands: %s\n", strings.Join(preview.ForbiddenCommands, ", "))
}

func (c command) printShimReadiness(readiness project.ShimReadiness) {
	fmt.Fprintf(c.stdout, "shim readiness: %s status=%s\n", readiness.Project.Key, readiness.Status)
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
}

func (c command) printShimAuthorizationPacket(packet project.ShimAuthorizationPacket) {
	fmt.Fprintf(c.stdout, "shim authorization: %s status=%s mode=%s\n", packet.Project.Key, packet.Status, packet.Mode)
	fmt.Fprintf(c.stdout, "readiness_status: %s\n", packet.ReadinessStatus)
	fmt.Fprintf(c.stdout, "intent: %s\n", packet.Intent)
	fmt.Fprintf(c.stdout, "allowed_files.count: %d\n", len(packet.AllowedFiles))
	for _, file := range packet.AllowedFiles {
		required := "optional"
		if file.Required {
			required = "required"
		}
		fmt.Fprintf(c.stdout, "[%s] %s action=%s boundary=%s\n", required, file.Path, file.Action, file.Boundary)
	}
	fmt.Fprintf(c.stdout, "forbidden_paths: %s\n", strings.Join(packet.ForbiddenPaths, ", "))
	fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(packet.ForbiddenActions, ", "))
	fmt.Fprintf(c.stdout, "required_preflight.count: %d\n", len(packet.RequiredPreflight))
	for _, item := range packet.RequiredPreflight {
		fmt.Fprintf(c.stdout, "required_preflight: %s\n", item)
	}
	fmt.Fprintf(c.stdout, "post_edit_verification.count: %d\n", len(packet.PostEditVerification))
	for _, item := range packet.PostEditVerification {
		fmt.Fprintf(c.stdout, "post_edit_verification: %s\n", item)
	}
	fmt.Fprintf(c.stdout, "rollback_scope.count: %d\n", len(packet.RollbackScope))
	for _, item := range packet.RollbackScope {
		fmt.Fprintf(c.stdout, "rollback_scope: %s\n", item)
	}
	fmt.Fprintf(c.stdout, "next_required_approval: %s\n", packet.NextRequiredApproval)
	for key, value := range packet.SafetyFacts {
		fmt.Fprintf(c.stdout, "safety.%s: %t\n", key, value)
	}
}

func (c command) printShimApplyPacketPreview(preview project.ShimApplyPacketPreview) {
	fmt.Fprintf(c.stdout, "shim apply packet: %s status=%s decision=%s mode=%s\n", preview.Project.Key, preview.Status, preview.Decision, preview.Mode)
	fmt.Fprintf(c.stdout, "message: %s\n", preview.Message)
	fmt.Fprintf(c.stdout, "authorization_snapshot_hash: %s\n", preview.Packet.AuthorizationSnapshotHash)
	fmt.Fprintf(c.stdout, "allowed_files: %s\n", strings.Join(preview.Packet.AllowedFiles, ", "))
	fmt.Fprintf(c.stdout, "apply_gate_command: %s\n", strings.Join(preview.ApplyGateCommand, " "))
	fmt.Fprintf(c.stdout, "future_apply_command: %s\n", strings.Join(preview.FutureApplyCommand, " "))
	fmt.Fprintf(c.stdout, "gate_status: %s decision=%s eligible=%t\n", preview.Gate.Status, preview.Gate.Decision, preview.Gate.ApplyCommandEligible)
	for _, item := range preview.Gate.Items {
		if item.Status != "pass" {
			fmt.Fprintf(c.stdout, "[%s] %s: %s blockers=%s\n", item.Status, item.Key, item.Message, strings.Join(item.BlockedBy, ", "))
		}
	}
	fmt.Fprintf(c.stdout, "would_create_command_request_after_apply_command: %t\n", preview.WouldCreateCommandRequestAfterApplyCommand)
	fmt.Fprintf(c.stdout, "would_write_area_matrix_shim_files_after_apply_command: %t\n", preview.WouldWriteAreaMatrixShimFilesAfterApplyCommand)
	fmt.Fprintf(c.stdout, "would_write_status_projection_after_apply_command: %t\n", preview.WouldWriteStatusProjectionAfterApplyCommand)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", preview.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", preview.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "task_loop_run_forwarded: %t\n", preview.TaskLoopRunForwarded)
	fmt.Fprintf(c.stdout, "area_matrix_files_modified: %t\n", preview.AreaMatrixFilesModified)
}

func (c command) printShimApplyGate(gate project.ShimApplyGate) {
	fmt.Fprintf(c.stdout, "shim apply gate: %s status=%s decision=%s mode=%s\n", gate.Project.Key, gate.Status, gate.Decision, gate.Mode)
	fmt.Fprintf(c.stdout, "message: %s\n", gate.Message)
	fmt.Fprintf(c.stdout, "apply_command_eligible: %t\n", gate.ApplyCommandEligible)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", gate.ApplyOpen)
	fmt.Fprintf(c.stdout, "allowed_files: %s\n", strings.Join(gate.AllowedFiles, ", "))
	fmt.Fprintf(c.stdout, "required_packet_fields: %s\n", strings.Join(gate.RequiredPacketFields, ", "))
	fmt.Fprintf(c.stdout, "required_proof_facts: %s\n", strings.Join(gate.RequiredProofFacts, ", "))
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s expected=%s actual=%s\n", item.Status, item.Category, item.Key, item.Expected, item.Actual)
		if len(item.BlockedBy) > 0 {
			fmt.Fprintf(c.stdout, "  blocked_by: %s\n", strings.Join(item.BlockedBy, ", "))
		}
	}
	fmt.Fprintf(c.stdout, "command_request_created: %t\n", gate.CommandRequestCreated)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", gate.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", gate.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "task_loop_run_forwarded: %t\n", gate.TaskLoopRunForwarded)
	fmt.Fprintf(c.stdout, "status_projection_written: %t\n", gate.StatusProjectionWritten)
	fmt.Fprintf(c.stdout, "area_matrix_files_modified: %t\n", gate.AreaMatrixFilesModified)
}

func (c command) printShimApplyCommand(result project.ApplyShimCommandResult) {
	fmt.Fprintf(c.stdout, "shim apply: %s status=%s decision=%s mode=%s\n", result.Project.Key, result.Status, result.Decision, result.Mode)
	fmt.Fprintf(c.stdout, "message: %s\n", result.Message)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", result.ApplyOpen)
	fmt.Fprintf(c.stdout, "blockers: %s\n", strings.Join(result.Blockers, ", "))
	fmt.Fprintf(c.stdout, "gate_status: %s decision=%s eligible=%t\n", result.Gate.Status, result.Gate.Decision, result.Gate.ApplyCommandEligible)
	fmt.Fprintf(c.stdout, "area_flow_command_created: %t\n", result.AreaFlowCommandCreated)
	fmt.Fprintf(c.stdout, "command_request_created: %t\n", result.CommandRequestCreated)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "task_loop_run_forwarded: %t\n", result.TaskLoopRunForwarded)
	fmt.Fprintf(c.stdout, "status_projection_written: %t\n", result.StatusProjectionWritten)
	fmt.Fprintf(c.stdout, "area_matrix_files_modified: %t\n", result.AreaMatrixFilesModified)
}

func (c command) printShimReadinessEvidence(result project.RecordShimReadinessEvidenceResult) {
	fmt.Fprintf(c.stdout, "shim readiness evidence: %s key=%s status=%s decision=%s\n",
		result.Project.Key,
		result.EvidenceKey,
		result.Status,
		result.Decision,
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printOperationsSmokeProof(result project.OperationsSmokeProof) {
	fmt.Fprintf(c.stdout, "operations smoke proof: %s key=%s status=%s evidence=%s decision=%s\n",
		result.Project.Key,
		result.ProofKey,
		result.Status,
		result.EvidenceStatus,
		result.Decision,
	)
	fmt.Fprintf(c.stdout, "record_command_runs_smoke: %t\n", result.RecordCommandRunsSmoke)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "service_process_control_attempted: %t\n", result.ServiceProcessControlAttempted)
	fmt.Fprintf(c.stdout, "support_bundle_exported: %t\n", result.SupportBundleExported)
	fmt.Fprintf(c.stdout, "migration_apply_attempted: %t\n", result.MigrationApplyAttempted)
	fmt.Fprintf(c.stdout, "remote_telemetry_enabled: %t\n", result.RemoteTelemetryEnabled)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printProtectedPathProof(result project.ProtectedPathProof) {
	fmt.Fprintf(c.stdout, "protected path proof: %s status=%s proof=%s decision=%s\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "git_status_run_by_command: %t\n", result.GitStatusRunByCommand)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "git_status_output_lines: %d\n", result.GitStatusOutputLines)
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printArchiveProof(result project.ArchiveProof) {
	fmt.Fprintf(c.stdout, "archive proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "artifact_bytes_copied: %t\n", result.ArtifactBytesCopied)
	fmt.Fprintf(c.stdout, "artifact_bytes_deleted: %t\n", result.ArtifactBytesDeleted)
	fmt.Fprintf(c.stdout, "historical_files_deleted: %t\n", result.HistoricalFilesDeleted)
	fmt.Fprintf(c.stdout, "historical_files_moved: %t\n", result.HistoricalFilesMoved)
	fmt.Fprintf(c.stdout, "progress_json_rewritten: %t\n", result.ProgressJSONRewritten)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "archive_scope_binding_status: %s\n", metadataString(result.Metadata, "archive_scope_binding_status"))
	fmt.Fprintf(c.stdout, "archive_scope_binding_hash: %s\n", metadataString(result.Metadata, "archive_scope_binding_hash"))
	fmt.Fprintf(c.stdout, "archive_scope: %s\n", result.ArchiveScope)
	fmt.Fprintf(c.stdout, "archive_reference_mode: %s\n", result.ArchiveReferenceMode)
	fmt.Fprintf(c.stdout, "archive_rollback_target: %s\n", result.ArchiveRollbackTarget)
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printShimRetirementProof(result project.ShimRetirementProof) {
	fmt.Fprintf(c.stdout, "shim retirement proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "legacy_runner_started: %t\n", result.LegacyRunnerStarted)
	fmt.Fprintf(c.stdout, "legacy_progress_written: %t\n", result.LegacyProgressWritten)
	fmt.Fprintf(c.stdout, "legacy_logs_written: %t\n", result.LegacyLogsWritten)
	fmt.Fprintf(c.stdout, "legacy_checkpoint_written: %t\n", result.LegacyCheckpointWritten)
	fmt.Fprintf(c.stdout, "historical_files_deleted: %t\n", result.HistoricalFilesDeleted)
	fmt.Fprintf(c.stdout, "progress_json_rewritten: %t\n", result.ProgressJSONRewritten)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "shim_retirement_scope_binding_status: %s\n", metadataString(result.Metadata, "shim_retirement_scope_binding_status"))
	fmt.Fprintf(c.stdout, "shim_retirement_scope_binding_hash: %s\n", metadataString(result.Metadata, "shim_retirement_scope_binding_hash"))
	fmt.Fprintf(c.stdout, "shim_retirement_scope: %s\n", result.ShimRetirementScope)
	fmt.Fprintf(c.stdout, "shim_rollback_target: %s\n", result.ShimRollbackTarget)
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printExecutionCutoverProof(result project.ExecutionCutoverProof) {
	fmt.Fprintf(c.stdout, "execution cutover proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "task_loop_run_forwarded_by_command: %t\n", result.TaskLoopRunForwardedByCommand)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "execution_cutover_scope: %s\n", result.ExecutionCutoverScope)
	fmt.Fprintf(c.stdout, "allowed_task_types: %s\n", strings.Join(result.AllowedTaskTypes, ","))
	fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(result.ForbiddenActions, ","))
	fmt.Fprintf(c.stdout, "rollback_target: %s\n", result.RollbackTarget)
	fmt.Fprintf(c.stdout, "rollback_mode: %s\n", result.RollbackMode)
	fmt.Fprintf(c.stdout, "fail_closed: %t\n", result.FailClosed)
	fmt.Fprintf(c.stdout, "reopen_requires_approval: %t\n", result.ReopenRequiresApproval)
	fmt.Fprintf(c.stdout, "source_write_open: %t\n", result.SourceWriteOpen)
	fmt.Fprintf(c.stdout, "generated_retained_write_open: %t\n", result.GeneratedRetainedWriteOpen)
	fmt.Fprintf(c.stdout, "repair_apply_open: %t\n", result.RepairApplyOpen)
	fmt.Fprintf(c.stdout, "checkpoint_apply_open: %t\n", result.CheckpointApplyOpen)
	fmt.Fprintf(c.stdout, "engine_execution_open: %t\n", result.EngineExecutionOpen)
	fmt.Fprintf(c.stdout, "secret_resolve_open: %t\n", result.SecretResolveOpen)
	fmt.Fprintf(c.stdout, "network_api_integration_open: %t\n", result.NetworkAPIIntegrationOpen)
	fmt.Fprintf(c.stdout, "publish_apply_open: %t\n", result.PublishApplyOpen)
	fmt.Fprintf(c.stdout, "restore_apply_open: %t\n", result.RestoreApplyOpen)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "legacy_progress_written: %t\n", result.LegacyProgressWritten)
	fmt.Fprintf(c.stdout, "legacy_logs_written: %t\n", result.LegacyLogsWritten)
	fmt.Fprintf(c.stdout, "legacy_checkpoint_written: %t\n", result.LegacyCheckpointWritten)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "execution_cutover_scope_binding_status: %s\n", metadataString(result.Metadata, "execution_cutover_scope_binding_status"))
	fmt.Fprintf(c.stdout, "execution_cutover_scope_binding_hash: %s\n", metadataString(result.Metadata, "execution_cutover_scope_binding_hash"))
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printValidationProof(result project.ValidationProof) {
	fmt.Fprintf(c.stdout, "validation proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "smoke_run_attempted: %t\n", result.SmokeRunAttempted)
	fmt.Fprintf(c.stdout, "web_build_run_by_command: %t\n", result.WebBuildRunByCommand)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printSourceAlignmentProof(result project.SourceAlignmentProof) {
	fmt.Fprintf(c.stdout, "source alignment proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "docs_written: %t\n", result.DocsWritten)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "source_alignment_binding_status: %s\n", metadataString(result.Metadata, "source_alignment_binding_status"))
	fmt.Fprintf(c.stdout, "source_alignment_source_set_hash: %s\n", metadataString(result.Metadata, "source_alignment_source_set_hash"))
	fmt.Fprintf(c.stdout, "source_alignment_source_file_count: %d\n", metadataInt64Value(result.Metadata, "source_alignment_source_file_count"))
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printTaskMatrixProof(result project.TaskMatrixProof) {
	fmt.Fprintf(c.stdout, "task matrix proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "docs_written: %t\n", result.DocsWritten)
	fmt.Fprintf(c.stdout, "tasks_written: %t\n", result.TasksWritten)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "task_matrix_binding_status: %s\n", metadataString(result.Metadata, "task_matrix_binding_status"))
	fmt.Fprintf(c.stdout, "task_matrix_source_set_hash: %s\n", metadataString(result.Metadata, "task_matrix_source_set_hash"))
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printSecurityClosureProof(result project.SecurityClosureProof) {
	fmt.Fprintf(c.stdout, "security closure proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "authorization_changed: %t\n", result.AuthorizationChanged)
	fmt.Fprintf(c.stdout, "secret_plaintext_read: %t\n", result.SecretPlaintextRead)
	fmt.Fprintf(c.stdout, "remote_worker_credentials_issued: %t\n", result.RemoteWorkerCredentialsIssued)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	if status := metadataString(result.Metadata, "security_closure_binding_status"); status != "" {
		fmt.Fprintf(c.stdout, "security_closure_binding_status: %s\n", status)
	}
	if hash := metadataString(result.Metadata, "security_closure_binding_hash"); hash != "" {
		fmt.Fprintf(c.stdout, "security_closure_binding_hash: %s\n", hash)
	}
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printBackupRestoreProof(result project.BackupRestoreProof) {
	fmt.Fprintf(c.stdout, "backup restore proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "database_restore_attempted: %t\n", result.DatabaseRestoreAttempted)
	fmt.Fprintf(c.stdout, "artifact_bytes_copied: %t\n", result.ArtifactBytesCopied)
	fmt.Fprintf(c.stdout, "artifact_bytes_deleted: %t\n", result.ArtifactBytesDeleted)
	fmt.Fprintf(c.stdout, "artifact_bytes_uploaded: %t\n", result.ArtifactBytesUploaded)
	fmt.Fprintf(c.stdout, "artifact_gc_attempted: %t\n", result.ArtifactGCAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printReleasePackagingProof(result project.ReleasePackagingProof) {
	fmt.Fprintf(c.stdout, "release packaging proof: %s status=%s proof=%s decision=%s facts=%d missing=%d\n",
		result.Project.Key,
		result.Status,
		result.ProofStatus,
		result.Decision,
		len(result.Facts),
		len(result.MissingFacts),
	)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "release_package_created: %t\n", result.ReleasePackageCreated)
	fmt.Fprintf(c.stdout, "release_state_written: %t\n", result.ReleaseStateWritten)
	fmt.Fprintf(c.stdout, "release_approval_created: %t\n", result.ReleaseApprovalCreated)
	fmt.Fprintf(c.stdout, "rollout_state_created: %t\n", result.RolloutStateCreated)
	fmt.Fprintf(c.stdout, "migration_apply_attempted: %t\n", result.MigrationApplyAttempted)
	fmt.Fprintf(c.stdout, "tag_created: %t\n", result.TagCreated)
	fmt.Fprintf(c.stdout, "package_signed: %t\n", result.PackageSigned)
	fmt.Fprintf(c.stdout, "artifact_uploaded: %t\n", result.ArtifactUploaded)
	fmt.Fprintf(c.stdout, "git_push_attempted: %t\n", result.GitPushAttempted)
	fmt.Fprintf(c.stdout, "publish_attempted: %t\n", result.PublishAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	if len(result.MissingFacts) > 0 {
		fmt.Fprintf(c.stdout, "missing_facts: %s\n", strings.Join(result.MissingFacts, ","))
	}
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printCompletionAuditSnapshot(result project.CompletionAuditSnapshot) {
	fmt.Fprintf(c.stdout, "completion audit snapshot: %s status=%s decision=%s audit=%s scope=%s rc=%s evidence_class=%s\n",
		result.Project.Key,
		result.Status,
		result.Decision,
		result.AuditStatus,
		result.AuditScope,
		result.ReleaseCandidateLabel,
		result.EvidenceClass,
	)
	c.printCompletionReal100Guardrail(result.Real100Guardrail)
	fmt.Fprintf(c.stdout, "audit_hash: %s\n", result.AuditHash)
	fmt.Fprintf(c.stdout, "proof_event_ids: %d\n", len(result.ProofEventIDs))
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "release_package_created: %t\n", result.ReleasePackageCreated)
	fmt.Fprintf(c.stdout, "publish_attempted: %t\n", result.PublishAttempted)
	fmt.Fprintf(c.stdout, "restore_apply_attempted: %t\n", result.RestoreApplyAttempted)
	fmt.Fprintf(c.stdout, "secret_resolved: %t\n", result.SecretResolved)
	fmt.Fprintf(c.stdout, "remote_worker_credentials_issued: %t\n", result.RemoteWorkerCredentialsIssued)
	fmt.Fprintf(c.stdout, "area_matrix_protected_paths_touched: %t\n", result.AreaMatrixProtectedPathsTouched)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "smoke_run_attempted: %t\n", result.SmokeRunAttempted)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", result.WorkerStarted)
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
}

func (c command) printExecutionCutoverReadiness(readiness project.AreaMatrixExecutionCutoverReadiness) {
	fmt.Fprintf(c.stdout, "execution cutover readiness: %s status=%s mode=%s\n", readiness.Project.Key, readiness.Status, readiness.Mode)
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n", item.Status, item.Category, item.Key, item.Message)
		if item.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", item.NextCommand)
		}
	}
	fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(readiness.ForbiddenActions, ", "))
}

func (c command) printExecutionForwardingV1Readiness(readiness project.ExecutionForwardingV1Readiness) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 readiness: %s status=%s mode=%s\n", readiness.Project.Key, readiness.Status, readiness.Mode)
	fmt.Fprintf(c.stdout, "allowed_task_types: %s\n", strings.Join(readiness.AllowedTaskTypes, ", "))
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n", item.Status, item.Category, item.Key, item.Message)
		if item.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", item.NextCommand)
		}
	}
	fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(readiness.ForbiddenActions, ", "))
}

func (c command) printExecutionForwardingV1ApplyPreview(preview project.ExecutionForwardingV1ApplyPreview) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 apply preview: %s status=%s mode=%s\n", preview.Project.Key, preview.Status, preview.Mode)
	fmt.Fprintf(c.stdout, "readiness.status: %s\n", preview.Readiness.Status)
	fmt.Fprintf(c.stdout, "approval_required: %t\n", preview.ApprovalRequired)
	fmt.Fprintf(c.stdout, "approval_status: %s\n", preview.ApprovalStatus)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", preview.ApplyOpen)
	fmt.Fprintf(c.stdout, "rollback_target: %s\n", preview.RollbackTarget)
	fmt.Fprintf(c.stdout, "allowed_task_types: %s\n", strings.Join(preview.AllowedTaskTypes, ", "))
	fmt.Fprintf(c.stdout, "forwarding_targets: %d\n", len(preview.ForwardingTargets))
	fmt.Fprintf(c.stdout, "blocked_targets: %d\n", len(preview.BlockedTargets))
	fmt.Fprintf(c.stdout, "required_capabilities: %s\n", strings.Join(preview.RequiredCapabilities, ", "))
	fmt.Fprintf(c.stdout, "apply_packet_fields: %s\n", strings.Join(preview.ApplyPacketFields, ", "))
	fmt.Fprintf(c.stdout, "fail_closed_fields: %s\n", strings.Join(preview.FailClosedFields, ", "))
	fmt.Fprintf(c.stdout, "required_proof_facts: %s\n", strings.Join(preview.RequiredProofFacts, ", "))
	fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ", "))
	for _, item := range preview.Items {
		if item.ApprovalStatus != "" {
			fmt.Fprintf(c.stdout, "[%s] %s/%s approval=%s: %s\n", item.Status, item.Category, item.Key, item.ApprovalStatus, item.Message)
			continue
		}
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n", item.Status, item.Category, item.Key, item.Message)
	}
}

func (c command) printExecutionForwardingV1ApplyPacketPreview(preview project.ExecutionForwardingV1ApplyPacketPreview) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 apply packet: %s status=%s decision=%s apply_command_eligible=%t approval_status=%s\n",
		preview.Project.Key,
		preview.Status,
		preview.Decision,
		preview.Gate.ApplyCommandEligible,
		preview.Gate.ApprovalStatus,
	)
	fmt.Fprintf(c.stdout, "readiness_snapshot_hash: %s\n", preview.Packet.ReadinessSnapshotHash)
	fmt.Fprintf(c.stdout, "approval_id: %s\n", preview.Packet.ApprovalID)
	fmt.Fprintf(c.stdout, "approval_scope: %s\n", preview.Packet.ApprovalScope)
	fmt.Fprintf(c.stdout, "legacy_non_write_proof_id: %s\n", preview.Packet.LegacyNonWriteProofID)
	fmt.Fprintf(c.stdout, "rollback_plan_id: %s\n", preview.Packet.RollbackPlanID)
	fmt.Fprintf(c.stdout, "protected_path_fingerprint_id: %s\n", preview.Packet.ProtectedPathFingerprintID)
	fmt.Fprintf(c.stdout, "failure_mode: %s\n", preview.Packet.FailureMode)
	fmt.Fprintf(c.stdout, "apply_gate_command: %s\n", strings.Join(preview.ApplyGateCommand, " "))
	fmt.Fprintf(c.stdout, "future_apply_command: %s\n", strings.Join(preview.FutureApplyCommand, " "))
	for _, item := range preview.Gate.Items {
		if item.Status == "pass" {
			continue
		}
		fmt.Fprintf(c.stdout, "blocked_item: %s status=%s expected=%s actual=%s\n", item.Key, item.Status, item.Expected, item.Actual)
		for _, blocker := range item.BlockedBy {
			fmt.Fprintf(c.stdout, "blocked_by: %s\n", blocker)
		}
	}
}

func (c command) printExecutionForwardingV1ApplyGate(gate project.ExecutionForwardingV1ApplyGate) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 apply gate: %s status=%s decision=%s apply_command_eligible=%t approval_status=%s\n",
		gate.Project.Key,
		gate.Status,
		gate.Decision,
		gate.ApplyCommandEligible,
		gate.ApprovalStatus,
	)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", gate.ApplyOpen)
	fmt.Fprintf(c.stdout, "allowed_task_types: %s\n", strings.Join(gate.AllowedTaskTypes, ", "))
	fmt.Fprintf(c.stdout, "target_command_types: %s\n", strings.Join(gate.TargetCommandTypes, ", "))
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "item: %s status=%s expected=%s actual=%s\n", item.Key, item.Status, item.Expected, item.Actual)
		for _, blocker := range item.BlockedBy {
			fmt.Fprintf(c.stdout, "blocked_by: %s\n", blocker)
		}
	}
}

func (c command) printExecutionForwardingV1Apply(result project.ApplyExecutionForwardingV1Result) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 apply: %s status=%s decision=%s command_request_created=%t\n",
		result.Project.Key,
		result.Status,
		result.Decision,
		result.CommandRequestCreated,
	)
	fmt.Fprintf(c.stdout, "gate.status: %s\n", result.Gate.Status)
	fmt.Fprintf(c.stdout, "gate.decision: %s\n", result.Gate.Decision)
	fmt.Fprintf(c.stdout, "apply_command_eligible: %t\n", result.Gate.ApplyCommandEligible)
	fmt.Fprintf(c.stdout, "area_flow_run_created: %t\n", result.AreaFlowRunCreated)
	fmt.Fprintf(c.stdout, "task_loop_run_forwarded: %t\n", result.TaskLoopRunForwarded)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocked_by: %s\n", blocker)
	}
}

func (c command) printExecutionForwardingV1CommandPreview(preview project.ExecutionForwardingV1CommandPreview) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 command preview: %s task_type=%s status=%s decision=%s\n", preview.Project.Key, preview.TaskType, preview.Status, preview.Decision)
	fmt.Fprintf(c.stdout, "mode: %s\n", preview.Mode)
	fmt.Fprintf(c.stdout, "message: %s\n", preview.Message)
	fmt.Fprintf(c.stdout, "target_command_type: %s\n", preview.TargetCommandType)
	fmt.Fprintf(c.stdout, "target_status: %s\n", preview.TargetStatus)
	fmt.Fprintf(c.stdout, "failure_mode: %s\n", preview.FailureMode)
	fmt.Fprintf(c.stdout, "apply_open: %t\n", preview.ApplyOpen)
	fmt.Fprintf(c.stdout, "allowed_task_type: %t\n", preview.AllowedTaskType)
	fmt.Fprintf(c.stdout, "blocked_task_type: %t\n", preview.BlockedTaskType)
	fmt.Fprintf(c.stdout, "would_create_command_request_after_approval: %t\n", preview.WouldCreateCommandRequestAfterApproval)
	fmt.Fprintf(c.stdout, "would_create_run_after_approval: %t\n", preview.WouldCreateRunAfterApproval)
	fmt.Fprintf(c.stdout, "project_write_allowed: %t\n", preview.ProjectWriteAllowed)
	fmt.Fprintf(c.stdout, "execution_write_allowed: %t\n", preview.ExecutionWriteAllowed)
	fmt.Fprintf(c.stdout, "legacy_fallback_allowed: %t\n", preview.LegacyFallbackAllowed)
	fmt.Fprintf(c.stdout, "blocked_by: %s\n", strings.Join(preview.BlockedBy, ", "))
	fmt.Fprintf(c.stdout, "fail_closed_fields: %s\n", strings.Join(preview.FailClosedFields, ", "))
}

func (c command) printExecutionForwardingV1RollbackPreview(preview project.ExecutionForwardingV1RollbackPreview) {
	fmt.Fprintf(c.stdout, "execution forwarding v1 rollback preview: %s status=%s mode=%s\n", preview.Project.Key, preview.Status, preview.Mode)
	fmt.Fprintf(c.stdout, "apply_preview.status: %s\n", preview.ApplyPreview.Status)
	fmt.Fprintf(c.stdout, "rollback_target: %s\n", preview.RollbackTarget)
	fmt.Fprintf(c.stdout, "rollback_apply_open: %t\n", preview.RollbackApplyOpen)
	fmt.Fprintf(c.stdout, "fail_closed_steps: %s\n", strings.Join(preview.FailClosedSteps, ", "))
	fmt.Fprintf(c.stdout, "reopen_conditions: %s\n", strings.Join(preview.ReopenConditions, ", "))
	fmt.Fprintf(c.stdout, "required_proof_facts: %s\n", strings.Join(preview.RequiredProofFacts, ", "))
	fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ", "))
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n", item.Status, item.Category, item.Key, item.Message)
	}
}

func (c command) printCutoverReadiness(readiness project.ProjectCutoverReadiness) {
	fmt.Fprintf(c.stdout, "cutover readiness: %s/%s status=%s phase_gate=%s\n",
		readiness.Project.Key,
		readiness.Version.DisplayLabel,
		readiness.Status,
		readiness.PhaseGate.Status,
	)
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
	if len(readiness.PhaseGate.Blockers) > 0 {
		fmt.Fprintf(c.stdout, "phase_gate.blockers: %s\n", readiness.PhaseGate.Blockers)
	}
	if len(readiness.PhaseGate.AcceptedWarnings) > 0 {
		fmt.Fprintf(c.stdout, "phase_gate.accepted_warnings: %s\n", readiness.PhaseGate.AcceptedWarnings)
	}
}

func (c command) printCutoverApply(result project.ApplyCutoverResult) {
	fmt.Fprintf(c.stdout, "cutover apply: %s/%s status=%s decision=%s\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Status,
		result.Decision,
	)
	fmt.Fprintf(c.stdout, "message: %s\n", result.Message)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	if result.EventID != 0 {
		fmt.Fprintf(c.stdout, "event_id: %d\n", result.EventID)
	}
	if result.AuditEventID != 0 {
		fmt.Fprintf(c.stdout, "audit_event_id: %d\n", result.AuditEventID)
	}
	if result.IdempotencyKey != "" {
		fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	}
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(c.stdout, "warning: %s\n", warning)
	}
}

func summaryToJSON(summary project.ProjectSummary) projectSummaryJSON {
	out := projectSummaryJSON{
		Project: projectRecordJSON{
			Key:             summary.Project.Key,
			Name:            summary.Project.Name,
			Kind:            summary.Project.Kind,
			Adapter:         summary.Project.Adapter,
			WorkflowProfile: summary.Project.WorkflowProfile,
			DefaultBranch:   summary.Project.DefaultBranch,
			Root:            summary.Project.RootPath,
		},
		Inventory: projectInventoryJSON{
			Versions:        summary.Inventory.Versions,
			Residuals:       summary.Inventory.Residuals,
			Artifacts:       summary.Inventory.Artifacts,
			ImportSnapshots: summary.Inventory.ImportSnapshots,
			MirrorExports:   summary.Inventory.MirrorExports,
		},
	}
	if summary.HasConfig {
		out.Config = &projectConfigJSON{
			ProtocolVersion: summary.Config.ProtocolVersion,
			ConfigPath:      summary.Config.ConfigPath,
			ConfigHash:      summary.Config.ConfigHash,
			Ownership:       summary.Config.Ownership,
			StatusExport:    summary.Config.StatusExport,
			Migration:       summary.Config.Migration,
			LoadedAt:        formatTime(summary.Config.LoadedAt),
		}
	}
	if summary.HasImport {
		out.Import = &projectImportJSON{
			SourceHash:             summary.Import.SourceHash,
			CreatedAt:              formatTime(summary.Import.CreatedAt),
			Summary:                summary.Import.Summary,
			HasPrevious:            summary.HasPreviousImport,
			HistoryReadyForDiff:    summary.HasPreviousImport,
			SourceHashChangedSince: summary.HasPreviousImport && summary.Import.SourceHash != summary.PreviousImport.SourceHash,
		}
		if summary.HasPreviousImport {
			out.Import.PreviousSourceHash = summary.PreviousImport.SourceHash
			out.Import.PreviousCreatedAt = formatTime(summary.PreviousImport.CreatedAt)
		}
	}
	if summary.HasLatestDoctor {
		out.Doctor = &projectDoctorJSON{
			Status:              summary.DoctorStatus,
			DriftStatus:         summary.DriftStatus,
			ConfigDriftStatus:   summary.ConfigDriftStatus,
			StageCoverageStatus: summary.StageCoverageStatus,
			NativeDoctorStatus:  summary.NativeDoctorStatus,
			Severity:            summary.LatestDoctor.Severity,
			CreatedAt:           formatTime(summary.LatestDoctor.CreatedAt),
			Metadata:            summary.LatestDoctor.Metadata,
		}
	}
	return out
}

func importResultToJSON(result importer.Result) projectImportRunJSON {
	return projectImportRunJSON{
		Project:        result.ProjectKey,
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

func verificationBundleToJSON(bundle project.ProjectVerificationBundle) projectVerificationBundleJSON {
	out := projectVerificationBundleJSON{
		Project: projectRecordJSON{
			Key:             bundle.Project.Key,
			Name:            bundle.Project.Name,
			Kind:            bundle.Project.Kind,
			Adapter:         bundle.Project.Adapter,
			WorkflowProfile: bundle.Project.WorkflowProfile,
			DefaultBranch:   bundle.Project.DefaultBranch,
			Root:            bundle.Project.RootPath,
		},
		Status: bundle.Status,
		PhaseGate: projectPhaseGateJSON{
			Name:             bundle.PhaseGate.Name,
			Status:           bundle.PhaseGate.Status,
			AcceptedWarnings: bundle.PhaseGate.AcceptedWarnings,
			Blockers:         bundle.PhaseGate.Blockers,
		},
		Summary:    summaryToJSON(bundle.Summary),
		Readiness:  readinessToJSON(bundle.Readiness),
		ImportDiff: importDiffToJSON(bundle.ImportDiff),
		Events:     make([]projectEventJSON, 0, len(bundle.Events)),
	}
	for _, event := range bundle.Events {
		out.Events = append(out.Events, eventToJSON(event))
	}
	return out
}

func importDiffToJSON(diff project.ProjectImportDiff) projectImportDiffJSON {
	out := projectImportDiffJSON{
		Project: projectRecordJSON{
			Key:             diff.Project.Key,
			Name:            diff.Project.Name,
			Kind:            diff.Project.Kind,
			Adapter:         diff.Project.Adapter,
			WorkflowProfile: diff.Project.WorkflowProfile,
			DefaultBranch:   diff.Project.DefaultBranch,
			Root:            diff.Project.RootPath,
		},
		Status:        diff.Status,
		HasPrevious:   diff.HasPrevious,
		SourceChanged: diff.SourceChanged,
		Latest: projectDiffSnapshotJSON{
			SourceHash: diff.Latest.SourceHash,
			CreatedAt:  formatTime(diff.Latest.CreatedAt),
		},
		Changes: make([]projectDiffChangeJSON, 0, len(diff.Changes)),
	}
	if diff.HasPrevious {
		out.Previous = projectDiffSnapshotJSON{
			SourceHash: diff.Previous.SourceHash,
			CreatedAt:  formatTime(diff.Previous.CreatedAt),
		}
	}
	for _, change := range diff.Changes {
		out.Changes = append(out.Changes, projectDiffChangeJSON{
			Key:      change.Key,
			Status:   change.Status,
			Previous: change.Previous,
			Latest:   change.Latest,
		})
	}
	return out
}

func compatibilityContractToJSON(contract project.CompatibilityContract) compatibilityContractJSON {
	out := compatibilityContractJSON{
		Project:  recordToJSON(contract.Project),
		Status:   contract.Status,
		Commands: make([]compatibilityCommandJSON, 0, len(contract.Commands)),
	}
	for _, command := range contract.Commands {
		out.Commands = append(out.Commands, compatibilityCommandJSON{
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
	return out
}

func shimPreviewToJSON(preview project.ShimPreview) shimPreviewJSON {
	out := shimPreviewJSON{
		Project:              recordToJSON(preview.Project),
		Status:               preview.Status,
		Mode:                 preview.Mode,
		Contract:             compatibilityContractToJSON(preview.Contract),
		PlannedFiles:         make([]shimFilePlanJSON, 0, len(preview.PlannedFiles)),
		CommandMappings:      make([]shimCommandMappingJSON, 0, len(preview.CommandMappings)),
		DiscoveryOrder:       append([]string{}, preview.DiscoveryOrder...),
		ForbiddenPaths:       append([]string{}, preview.ForbiddenPaths...),
		ForbiddenCommands:    append([]string{}, preview.ForbiddenCommands...),
		VerificationCommands: append([]string{}, preview.VerificationCommands...),
		RollbackSteps:        append([]string{}, preview.RollbackSteps...),
		Notes:                append([]string{}, preview.Notes...),
	}
	for _, file := range preview.PlannedFiles {
		out.PlannedFiles = append(out.PlannedFiles, shimFilePlanJSON{
			Path:     file.Path,
			Action:   file.Action,
			Required: file.Required,
			Reason:   file.Reason,
			Boundary: file.Boundary,
		})
	}
	for _, mapping := range preview.CommandMappings {
		out.CommandMappings = append(out.CommandMappings, shimCommandMappingJSON{
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
	return out
}

func shimReadinessToJSON(readiness project.ShimReadiness) shimReadinessJSON {
	out := shimReadinessJSON{
		Project: recordToJSON(readiness.Project),
		Status:  readiness.Status,
		Preview: shimPreviewToJSON(readiness.Preview),
		Items:   make([]shimReadinessItemJSON, 0, len(readiness.Items)),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, shimReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func shimAuthorizationPacketToJSON(packet project.ShimAuthorizationPacket) shimAuthorizationPacketJSON {
	out := shimAuthorizationPacketJSON{
		Project:              recordToJSON(packet.Project),
		Status:               packet.Status,
		Mode:                 packet.Mode,
		Intent:               packet.Intent,
		ReadinessStatus:      packet.ReadinessStatus,
		AllowedFiles:         make([]shimFilePlanJSON, 0, len(packet.AllowedFiles)),
		ForbiddenPaths:       append([]string{}, packet.ForbiddenPaths...),
		ForbiddenActions:     append([]string{}, packet.ForbiddenActions...),
		RequiredPreflight:    append([]string{}, packet.RequiredPreflight...),
		PostEditVerification: append([]string{}, packet.PostEditVerification...),
		RollbackScope:        append([]string{}, packet.RollbackScope...),
		SafetyFacts:          map[string]bool{},
		NextRequiredApproval: packet.NextRequiredApproval,
	}
	for _, file := range packet.AllowedFiles {
		out.AllowedFiles = append(out.AllowedFiles, shimFilePlanJSON{
			Path:     file.Path,
			Action:   file.Action,
			Required: file.Required,
			Reason:   file.Reason,
			Boundary: file.Boundary,
		})
	}
	for key, value := range packet.SafetyFacts {
		out.SafetyFacts[key] = value
	}
	return out
}

func shimApplyPacketPreviewToJSON(preview project.ShimApplyPacketPreview) shimApplyPacketPreviewJSON {
	return shimApplyPacketPreviewJSON{
		Project:             recordToJSON(preview.Project),
		Status:              preview.Status,
		Mode:                preview.Mode,
		Decision:            preview.Decision,
		Message:             preview.Message,
		Authorization:       shimAuthorizationPacketToJSON(preview.Authorization),
		Gate:                shimApplyGateToJSON(preview.Gate),
		Packet:              shimApplyPacketToJSON(preview.Packet),
		ApplyGateCommand:    append([]string{}, preview.ApplyGateCommand...),
		FutureApplyCommand:  append([]string{}, preview.FutureApplyCommand...),
		RequiredHumanReview: append([]string{}, preview.RequiredHumanReview...),
		ForbiddenActions:    append([]string{}, preview.ForbiddenActions...),
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

func shimApplyPacketToJSON(packet project.ShimApplyPacket) shimApplyPacketJSON {
	return shimApplyPacketJSON{
		CommandType:                packet.CommandType,
		ProjectKey:                 packet.ProjectKey,
		AllowedFiles:               append([]string{}, packet.AllowedFiles...),
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

func shimApplyGateToJSON(gate project.ShimApplyGate) shimApplyGateJSON {
	out := shimApplyGateJSON{
		Project:                 recordToJSON(gate.Project),
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Decision:                gate.Decision,
		Message:                 gate.Message,
		Items:                   make([]shimApplyGateItemJSON, 0, len(gate.Items)),
		RequiredPacketFields:    append([]string{}, gate.RequiredPacketFields...),
		RequiredCapabilities:    append([]string{}, gate.RequiredCapabilities...),
		AllowedFiles:            append([]string{}, gate.AllowedFiles...),
		ForbiddenPaths:          append([]string{}, gate.ForbiddenPaths...),
		ForbiddenActions:        append([]string{}, gate.ForbiddenActions...),
		RequiredPreflight:       append([]string{}, gate.RequiredPreflight...),
		PostEditVerification:    append([]string{}, gate.PostEditVerification...),
		RollbackScope:           append([]string{}, gate.RollbackScope...),
		RequiredProofFacts:      append([]string{}, gate.RequiredProofFacts...),
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
		out.Items = append(out.Items, shimApplyGateItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Expected:         item.Expected,
			Actual:           item.Actual,
			RequiredEvidence: append([]string{}, item.RequiredEvidence...),
			BlockedBy:        append([]string{}, item.BlockedBy...),
		})
	}
	return out
}

func shimApplyCommandToJSON(result project.ApplyShimCommandResult) shimApplyCommandJSON {
	return shimApplyCommandJSON{
		Project:                 recordToJSON(result.Project),
		Status:                  result.Status,
		Mode:                    result.Mode,
		Decision:                result.Decision,
		Message:                 result.Message,
		Gate:                    shimApplyGateToJSON(result.Gate),
		Blockers:                append([]string{}, result.Blockers...),
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		IdempotencyKey:          result.IdempotencyKey,
		Created:                 result.Created,
		RequiredPreflight:       append([]string{}, result.RequiredPreflight...),
		ForbiddenActions:        append([]string{}, result.ForbiddenActions...),
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

func shimReadinessEvidenceToJSON(result project.RecordShimReadinessEvidenceResult) shimReadinessEvidenceJSON {
	return shimReadinessEvidenceJSON{
		Project:                 recordToJSON(result.Project),
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

func operationsSmokeProofToJSON(result project.OperationsSmokeProof) operationsSmokeProofJSON {
	return operationsSmokeProofJSON{
		Project:                         recordToJSON(result.Project),
		ProofKey:                        result.ProofKey,
		Status:                          result.Status,
		EvidenceStatus:                  result.EvidenceStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		EngineCallAttempted:             result.EngineCallAttempted,
		ServiceProcessControlAttempted:  result.ServiceProcessControlAttempted,
		SupportBundleExported:           result.SupportBundleExported,
		MigrationApplyAttempted:         result.MigrationApplyAttempted,
		RemoteTelemetryEnabled:          result.RemoteTelemetryEnabled,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		RecordCommandRunsSmoke:          result.RecordCommandRunsSmoke,
		Metadata:                        jsonObject(result.Metadata),
	}
}

func protectedPathProofToJSON(result project.ProtectedPathProof) protectedPathProofJSON {
	out := protectedPathProofJSON{
		Project:                           recordToJSON(result.Project),
		Status:                            result.Status,
		ProofStatus:                       result.ProofStatus,
		Decision:                          result.Decision,
		Message:                           result.Message,
		EventID:                           result.EventID,
		AuditEventID:                      result.AuditEventID,
		IdempotencyKey:                    result.IdempotencyKey,
		Created:                           result.Created,
		ProjectWriteAttempted:             result.ProjectWriteAttempted,
		ExecutionWriteAttempted:           result.ExecutionWriteAttempted,
		EngineCallAttempted:               result.EngineCallAttempted,
		CommandsRun:                       result.CommandsRun,
		GitStatusRunByCommand:             result.GitStatusRunByCommand,
		AreaMatrixProtectedPathsTouched:   result.AreaMatrixProtectedPathsTouched,
		GitStatusOutputHash:               result.GitStatusOutputHash,
		GitStatusOutputLines:              result.GitStatusOutputLines,
		GitStatusOutputEmpty:              metadataBoolValue(result.Metadata, "git_status_output_empty"),
		ProtectedPathSetHash:              metadataString(result.Metadata, "protected_path_set_hash"),
		ProtectedPathSetCount:             metadataInt64Value(result.Metadata, "protected_path_set_count"),
		ProtectedPathProofBindingStatus:   metadataString(result.Metadata, "protected_path_proof_binding_status"),
		ProtectedPathProofBindingBlockers: metadataStringSlice(result.Metadata, "protected_path_proof_binding_blockers"),
		Summary:                           metadataString(result.Metadata, "summary"),
		EvidenceURI:                       metadataString(result.Metadata, "evidence_uri"),
		Metadata:                          jsonObject(result.Metadata),
	}
	if result.ProofStatus == "authorized" {
		complete := metadataBoolValue(result.Metadata, "authorized_proof_complete")
		if _, ok := result.Metadata["authorized_proof_complete"]; !ok {
			complete = protectedPathProofAuthorizedJSONComplete(result.Metadata)
		}
		out.AuthorizedApprovalID = metadataString(result.Metadata, "authorized_approval_id")
		out.AuthorizedAllowedPaths = metadataStringSlice(result.Metadata, "authorized_allowed_paths")
		out.AuthorizedDirtyOutputHash = metadataString(result.Metadata, "authorized_dirty_output_hash")
		out.AuthorizedReviewer = metadataString(result.Metadata, "authorized_reviewer")
		out.AuthorizedRollbackEvidenceURI = metadataString(result.Metadata, "authorized_rollback_evidence_uri")
		out.AuthorizedTouchedPaths = metadataStringSlice(result.Metadata, "authorized_touched_paths")
		out.AuthorizedProofComplete = &complete
	}
	return out
}

func archiveProofToJSON(result project.ArchiveProof) archiveProofJSON {
	return archiveProofJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		ProofStatus:                     result.ProofStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Facts:                           append([]string{}, result.Facts...),
		MissingFacts:                    append([]string{}, result.MissingFacts...),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		ArtifactBytesCopied:             result.ArtifactBytesCopied,
		ArtifactBytesDeleted:            result.ArtifactBytesDeleted,
		HistoricalFilesDeleted:          result.HistoricalFilesDeleted,
		HistoricalFilesMoved:            result.HistoricalFilesMoved,
		ProgressJSONRewritten:           result.ProgressJSONRewritten,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		CommandsRun:                     result.CommandsRun,
		ArchiveScopeBindingStatus:       metadataString(result.Metadata, "archive_scope_binding_status"),
		ArchiveScopeBindingBlockers:     metadataStringSlice(result.Metadata, "archive_scope_binding_blockers"),
		ArchiveBindingContract:          metadataString(result.Metadata, "archive_binding_contract"),
		ArchiveSourcePathsHash:          metadataString(result.Metadata, "archive_source_paths_hash"),
		ArchiveForbiddenActionsHash:     metadataString(result.Metadata, "archive_forbidden_actions_hash"),
		ArchiveScopeBindingHash:         metadataString(result.Metadata, "archive_scope_binding_hash"),
		ArchiveScope:                    result.ArchiveScope,
		ArchiveReferenceMode:            result.ArchiveReferenceMode,
		ArchiveSourcePaths:              append([]string{}, result.ArchiveSourcePaths...),
		ArchiveForbiddenActions:         append([]string{}, result.ArchiveForbiddenActions...),
		ArchiveRollbackTarget:           result.ArchiveRollbackTarget,
		ArchiveFailClosed:               result.ArchiveFailClosed,
		ReviewDecision:                  metadataString(result.Metadata, "review_decision"),
		ReviewedBy:                      metadataString(result.Metadata, "reviewed_by"),
		ReviewedAt:                      metadataString(result.Metadata, "reviewed_at"),
		ReviewMetadataStatus:            metadataString(result.Metadata, "review_metadata_status"),
		ReviewMetadataBlockers:          metadataStringSlice(result.Metadata, "review_metadata_blockers"),
		Metadata:                        jsonObject(result.Metadata),
	}
}

func shimRetirementProofToJSON(result project.ShimRetirementProof) shimRetirementProofJSON {
	return shimRetirementProofJSON{
		Project:                            recordToJSON(result.Project),
		Status:                             result.Status,
		ProofStatus:                        result.ProofStatus,
		Decision:                           result.Decision,
		Message:                            result.Message,
		Facts:                              append([]string{}, result.Facts...),
		MissingFacts:                       append([]string{}, result.MissingFacts...),
		EventID:                            result.EventID,
		AuditEventID:                       result.AuditEventID,
		IdempotencyKey:                     result.IdempotencyKey,
		Created:                            result.Created,
		ProjectWriteAttempted:              result.ProjectWriteAttempted,
		ExecutionWriteAttempted:            result.ExecutionWriteAttempted,
		CommandsRun:                        result.CommandsRun,
		LegacyRunnerStarted:                result.LegacyRunnerStarted,
		LegacyProgressWritten:              result.LegacyProgressWritten,
		LegacyLogsWritten:                  result.LegacyLogsWritten,
		LegacyCheckpointWritten:            result.LegacyCheckpointWritten,
		HistoricalFilesDeleted:             result.HistoricalFilesDeleted,
		ProgressJSONRewritten:              result.ProgressJSONRewritten,
		AreaMatrixProtectedPathsTouched:    result.AreaMatrixProtectedPathsTouched,
		ShimRetirementScopeBindingStatus:   metadataString(result.Metadata, "shim_retirement_scope_binding_status"),
		ShimRetirementScopeBindingBlockers: metadataStringSlice(result.Metadata, "shim_retirement_scope_binding_blockers"),
		ShimRetirementBindingContract:      metadataString(result.Metadata, "shim_retirement_binding_contract"),
		ShimRetirementPrerequisitesHash:    metadataString(result.Metadata, "shim_retirement_prerequisites_hash"),
		ShimRetiredSurfacesHash:            metadataString(result.Metadata, "shim_retired_surfaces_hash"),
		ShimRetirementScopeBindingHash:     metadataString(result.Metadata, "shim_retirement_scope_binding_hash"),
		ShimRetirementScope:                result.ShimRetirementScope,
		ShimRetirementPrerequisites:        append([]string{}, result.ShimRetirementPrerequisites...),
		ShimRetiredSurfaces:                append([]string{}, result.ShimRetiredSurfaces...),
		ShimRollbackTarget:                 result.ShimRollbackTarget,
		ShimFailClosed:                     result.ShimFailClosed,
		ShimReopenRequiresApproval:         result.ShimReopenRequiresApproval,
		ReviewDecision:                     metadataString(result.Metadata, "review_decision"),
		ReviewedBy:                         metadataString(result.Metadata, "reviewed_by"),
		ReviewedAt:                         metadataString(result.Metadata, "reviewed_at"),
		ReviewMetadataStatus:               metadataString(result.Metadata, "review_metadata_status"),
		ReviewMetadataBlockers:             metadataStringSlice(result.Metadata, "review_metadata_blockers"),
		Metadata:                           jsonObject(result.Metadata),
	}
}

func executionCutoverProofToJSON(result project.ExecutionCutoverProof) executionCutoverProofJSON {
	return executionCutoverProofJSON{
		Project:                              recordToJSON(result.Project),
		Status:                               result.Status,
		ProofStatus:                          result.ProofStatus,
		Decision:                             result.Decision,
		Message:                              result.Message,
		Facts:                                append([]string{}, result.Facts...),
		MissingFacts:                         append([]string{}, result.MissingFacts...),
		EventID:                              result.EventID,
		AuditEventID:                         result.AuditEventID,
		IdempotencyKey:                       result.IdempotencyKey,
		Created:                              result.Created,
		ExecutionCutoverScope:                result.ExecutionCutoverScope,
		AllowedTaskTypes:                     append([]string{}, result.AllowedTaskTypes...),
		ForbiddenActions:                     append([]string{}, result.ForbiddenActions...),
		RollbackTarget:                       result.RollbackTarget,
		RollbackMode:                         result.RollbackMode,
		FailClosed:                           result.FailClosed,
		ReopenRequiresApproval:               result.ReopenRequiresApproval,
		SourceWriteOpen:                      result.SourceWriteOpen,
		GeneratedRetainedWriteOpen:           result.GeneratedRetainedWriteOpen,
		RepairApplyOpen:                      result.RepairApplyOpen,
		CheckpointApplyOpen:                  result.CheckpointApplyOpen,
		EngineExecutionOpen:                  result.EngineExecutionOpen,
		SecretResolveOpen:                    result.SecretResolveOpen,
		NetworkAPIIntegrationOpen:            result.NetworkAPIIntegrationOpen,
		PublishApplyOpen:                     result.PublishApplyOpen,
		RestoreApplyOpen:                     result.RestoreApplyOpen,
		ProjectWriteAttempted:                result.ProjectWriteAttempted,
		ExecutionWriteAttempted:              result.ExecutionWriteAttempted,
		TaskLoopRunForwardedByCommand:        result.TaskLoopRunForwardedByCommand,
		EngineCallAttempted:                  result.EngineCallAttempted,
		CommandsRun:                          result.CommandsRun,
		LegacyProgressWritten:                result.LegacyProgressWritten,
		LegacyLogsWritten:                    result.LegacyLogsWritten,
		LegacyCheckpointWritten:              result.LegacyCheckpointWritten,
		AreaMatrixProtectedPathsTouched:      result.AreaMatrixProtectedPathsTouched,
		ExecutionCutoverScopeBindingStatus:   metadataString(result.Metadata, "execution_cutover_scope_binding_status"),
		ExecutionCutoverScopeBindingBlockers: metadataStringSlice(result.Metadata, "execution_cutover_scope_binding_blockers"),
		ExecutionCutoverBindingContract:      metadataString(result.Metadata, "execution_cutover_binding_contract"),
		AllowedTaskTypesHash:                 metadataString(result.Metadata, "allowed_task_types_hash"),
		ForbiddenActionsHash:                 metadataString(result.Metadata, "forbidden_actions_hash"),
		ExecutionCutoverBindingHash:          metadataString(result.Metadata, "execution_cutover_binding_hash"),
		ExecutionCutoverScopeBindingHash:     metadataString(result.Metadata, "execution_cutover_scope_binding_hash"),
		ReviewDecision:                       metadataString(result.Metadata, "review_decision"),
		ReviewedBy:                           metadataString(result.Metadata, "reviewed_by"),
		ReviewedAt:                           metadataString(result.Metadata, "reviewed_at"),
		ReviewMetadataStatus:                 metadataString(result.Metadata, "review_metadata_status"),
		ReviewMetadataBlockers:               metadataStringSlice(result.Metadata, "review_metadata_blockers"),
		Metadata:                             jsonObject(result.Metadata),
	}
}

func validationProofToJSON(result project.ValidationProof) validationProofJSON {
	return validationProofJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		ProofStatus:                     result.ProofStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Facts:                           append([]string{}, result.Facts...),
		MissingFacts:                    append([]string{}, result.MissingFacts...),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		EngineCallAttempted:             result.EngineCallAttempted,
		CommandsRun:                     result.CommandsRun,
		SmokeRunAttempted:               result.SmokeRunAttempted,
		WebBuildRunByCommand:            result.WebBuildRunByCommand,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		Metadata:                        jsonObject(result.Metadata),
	}
}

func sourceAlignmentProofToJSON(result project.SourceAlignmentProof) sourceAlignmentProofJSON {
	return sourceAlignmentProofJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		ProofStatus:                     result.ProofStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Facts:                           append([]string{}, result.Facts...),
		MissingFacts:                    append([]string{}, result.MissingFacts...),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		CommandsRun:                     result.CommandsRun,
		DocsWritten:                     result.DocsWritten,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		SourceAlignmentBindingStatus:    metadataString(result.Metadata, "source_alignment_binding_status"),
		SourceAlignmentBindingBlockers:  metadataStringSlice(result.Metadata, "source_alignment_binding_blockers"),
		SourceAlignmentSourcePaths:      metadataStringSlice(result.Metadata, "source_alignment_source_paths"),
		SourceAlignmentSourceHashes:     metadataStringMapValue(result.Metadata, "source_alignment_source_hashes"),
		SourceAlignmentSourceSetHash:    metadataString(result.Metadata, "source_alignment_source_set_hash"),
		SourceAlignmentSourceFileCount:  metadataInt64Value(result.Metadata, "source_alignment_source_file_count"),
		MissingSourceCount:              metadataInt64Value(result.Metadata, "source_alignment_missing_source_count"),
		UnreadableSourceCount:           metadataInt64Value(result.Metadata, "source_alignment_unreadable_source_count"),
		Metadata:                        jsonObject(result.Metadata),
	}
}

func taskMatrixProofToJSON(result project.TaskMatrixProof) taskMatrixProofJSON {
	return taskMatrixProofJSON{
		Project:                            recordToJSON(result.Project),
		Status:                             result.Status,
		ProofStatus:                        result.ProofStatus,
		Decision:                           result.Decision,
		Message:                            result.Message,
		Facts:                              append([]string{}, result.Facts...),
		MissingFacts:                       append([]string{}, result.MissingFacts...),
		EventID:                            result.EventID,
		AuditEventID:                       result.AuditEventID,
		IdempotencyKey:                     result.IdempotencyKey,
		Created:                            result.Created,
		ProjectWriteAttempted:              result.ProjectWriteAttempted,
		ExecutionWriteAttempted:            result.ExecutionWriteAttempted,
		CommandsRun:                        result.CommandsRun,
		DocsWritten:                        result.DocsWritten,
		TasksWritten:                       result.TasksWritten,
		AreaMatrixProtectedPathsTouched:    result.AreaMatrixProtectedPathsTouched,
		TaskMatrixBindingStatus:            metadataString(result.Metadata, "task_matrix_binding_status"),
		TaskMatrixBindingBlockers:          metadataStringSlice(result.Metadata, "task_matrix_binding_blockers"),
		TaskMatrixSourcePaths:              metadataStringSlice(result.Metadata, "task_matrix_source_paths"),
		TaskMatrixSourceSetHash:            metadataString(result.Metadata, "task_matrix_source_set_hash"),
		TaskBacklogHash:                    metadataString(result.Metadata, "task_backlog_hash"),
		TaskStatusAuditHash:                metadataString(result.Metadata, "task_status_audit_hash"),
		PlannedV1RequiredTaskCount:         metadataInt64Value(result.Metadata, "planned_v1_required_task_count"),
		MissingEvidenceV1RequiredTaskCount: metadataInt64Value(result.Metadata, "missing_evidence_v1_required_task_count"),
		BlockedV1RequiredTaskCount:         metadataInt64Value(result.Metadata, "blocked_v1_required_task_count"),
		Metadata:                           jsonObject(result.Metadata),
	}
}

func securityClosureProofToJSON(result project.SecurityClosureProof) securityClosureProofJSON {
	return securityClosureProofJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		ProofStatus:                     result.ProofStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Facts:                           append([]string{}, result.Facts...),
		MissingFacts:                    append([]string{}, result.MissingFacts...),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		AuthorizationChanged:            result.AuthorizationChanged,
		SecretPlaintextRead:             result.SecretPlaintextRead,
		RemoteWorkerCredentialsIssued:   result.RemoteWorkerCredentialsIssued,
		CommandsRun:                     result.CommandsRun,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		SecurityClosureBindingStatus:    metadataString(result.Metadata, "security_closure_binding_status"),
		SecurityClosureBindingBlockers:  metadataStringSlice(result.Metadata, "security_closure_binding_blockers"),
		SecurityClosureBindingHash:      metadataString(result.Metadata, "security_closure_binding_hash"),
		SecurityBoundaryStatus:          metadataString(result.Metadata, "security_boundary_status"),
		PermissionDoctorStatus:          metadataString(result.Metadata, "permission_doctor_status"),
		AuditCoverageStatus:             metadataString(result.Metadata, "audit_coverage_status"),
		Metadata:                        jsonObject(result.Metadata),
	}
}

func backupRestoreProofToJSON(result project.BackupRestoreProof) backupRestoreProofJSON {
	return backupRestoreProofJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		ProofStatus:                     result.ProofStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Facts:                           append([]string{}, result.Facts...),
		MissingFacts:                    append([]string{}, result.MissingFacts...),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		DatabaseRestoreAttempted:        result.DatabaseRestoreAttempted,
		ArtifactBytesCopied:             result.ArtifactBytesCopied,
		ArtifactBytesDeleted:            result.ArtifactBytesDeleted,
		ArtifactBytesUploaded:           result.ArtifactBytesUploaded,
		ArtifactGCAttempted:             result.ArtifactGCAttempted,
		CommandsRun:                     result.CommandsRun,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		RestorePlanScope:                metadataString(result.Metadata, "restore_plan_scope"),
		RestorePlanProjectKey:           metadataString(result.Metadata, "restore_plan_project_key"),
		Metadata:                        jsonObject(result.Metadata),
	}
}

func releasePackagingProofToJSON(result project.ReleasePackagingProof) releasePackagingProofJSON {
	return releasePackagingProofJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		ProofStatus:                     result.ProofStatus,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Facts:                           append([]string{}, result.Facts...),
		MissingFacts:                    append([]string{}, result.MissingFacts...),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		ReleasePackageCreated:           result.ReleasePackageCreated,
		ReleaseStateWritten:             result.ReleaseStateWritten,
		ReleaseApprovalCreated:          result.ReleaseApprovalCreated,
		RolloutStateCreated:             result.RolloutStateCreated,
		MigrationApplyAttempted:         result.MigrationApplyAttempted,
		TagCreated:                      result.TagCreated,
		PackageSigned:                   result.PackageSigned,
		ArtifactUploaded:                result.ArtifactUploaded,
		GitPushAttempted:                result.GitPushAttempted,
		PublishAttempted:                result.PublishAttempted,
		CommandsRun:                     result.CommandsRun,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		Metadata:                        jsonObject(result.Metadata),
	}
}

func completionAuditSnapshotToJSON(result project.CompletionAuditSnapshot) completionAuditSnapshotJSON {
	guardrail := completionAuditReal100Guardrail(result.Real100Guardrail)
	return completionAuditSnapshotJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		Decision:                        result.Decision,
		Message:                         result.Message,
		AuditStatus:                     result.AuditStatus,
		AuditScope:                      result.AuditScope,
		ReadinessScope:                  guardrail.ReadinessScope,
		ClaimScope:                      guardrail.ClaimScope,
		NotReal100:                      guardrail.NotReal100,
		EvidenceOnly:                    guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion:      guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:        guardrail.ReleaseCandidateDecision,
		Real100Status:                   guardrail.Real100Status,
		Real100Blockers:                 jsonStringSlice(guardrail.Real100Blockers),
		Real100Breakdown:                guardrail.Real100Breakdown,
		AuditHash:                       result.AuditHash,
		ReleaseCandidateLabel:           result.ReleaseCandidateLabel,
		EvidenceClass:                   result.EvidenceClass,
		EvidenceURI:                     result.EvidenceURI,
		ProofEventIDs:                   copyInt64Map(result.ProofEventIDs),
		EventID:                         result.EventID,
		AuditEventID:                    result.AuditEventID,
		IdempotencyKey:                  result.IdempotencyKey,
		Created:                         result.Created,
		ProjectWriteAttempted:           result.ProjectWriteAttempted,
		ExecutionWriteAttempted:         result.ExecutionWriteAttempted,
		ReleasePackageCreated:           result.ReleasePackageCreated,
		PublishAttempted:                result.PublishAttempted,
		RestoreApplyAttempted:           result.RestoreApplyAttempted,
		SecretResolved:                  result.SecretResolved,
		RemoteWorkerCredentialsIssued:   result.RemoteWorkerCredentialsIssued,
		AreaMatrixProtectedPathsTouched: result.AreaMatrixProtectedPathsTouched,
		CommandsRun:                     result.CommandsRun,
		SmokeRunAttempted:               result.SmokeRunAttempted,
		WorkerStarted:                   result.WorkerStarted,
		Metadata:                        jsonObject(result.Metadata),
	}
}

func completionAuditSnapshotReadinessToJSON(readiness project.CompletionAuditSnapshotReadiness) completionAuditSnapshotReadinessJSON {
	guardrail := completionAuditReal100Guardrail(readiness.Real100Guardrail)
	out := completionAuditSnapshotReadinessJSON{
		Project:                    recordToJSON(readiness.Project),
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
		Latest:                     completionAuditSnapshotToJSON(readiness.Latest),
		Items:                      make([]projectReadinessItemJSON, 0, len(readiness.Items)),
		Gaps:                       project.CompletionAuditSnapshotReadinessGaps(readiness),
		Closure:                    project.CompletionAuditSnapshotReadinessClosure(readiness),
		SafetyFacts:                copyBoolMap(readiness.SafetyFacts),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: jsonObject(item.Metadata),
		})
	}
	return out
}

func copyInt64Map(in map[string]int64) map[string]int64 {
	out := map[string]int64{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func executionCutoverReadinessToJSON(readiness project.AreaMatrixExecutionCutoverReadiness) executionCutoverReadinessJSON {
	out := executionCutoverReadinessJSON{
		Project:          recordToJSON(readiness.Project),
		Status:           readiness.Status,
		Mode:             readiness.Mode,
		Items:            make([]executionCutoverReadinessItemJSON, 0, len(readiness.Items)),
		MigrationPath:    append([]string{}, readiness.MigrationPath...),
		CommandEvidence:  readiness.CommandEvidence,
		Capabilities:     append([]string{}, readiness.Capabilities...),
		ForbiddenActions: append([]string{}, readiness.ForbiddenActions...),
		SafetyFacts:      readiness.SafetyFacts,
		NextSteps:        make([]executionCutoverNextStepJSON, 0, len(readiness.NextSteps)),
		GeneratedAt:      formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, executionCutoverReadinessItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			RequiredEvidence: append([]string{}, item.RequiredEvidence...),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	for _, step := range readiness.NextSteps {
		out.NextSteps = append(out.NextSteps, executionCutoverNextStepJSON{
			Key:         step.Key,
			Owner:       step.Owner,
			Action:      step.Action,
			RiskLevel:   step.RiskLevel,
			BlockedBy:   append([]string{}, step.BlockedBy...),
			NextCommand: step.NextCommand,
			Metadata:    step.Metadata,
		})
	}
	return out
}

func executionForwardingV1ReadinessToJSON(readiness project.ExecutionForwardingV1Readiness) executionForwardingV1ReadinessJSON {
	out := executionForwardingV1ReadinessJSON{
		Project:          recordToJSON(readiness.Project),
		Status:           readiness.Status,
		Mode:             readiness.Mode,
		Items:            make([]executionCutoverReadinessItemJSON, 0, len(readiness.Items)),
		AllowedTaskTypes: append([]string{}, readiness.AllowedTaskTypes...),
		CommandEvidence:  readiness.CommandEvidence,
		Capabilities:     append([]string{}, readiness.Capabilities...),
		ForbiddenActions: append([]string{}, readiness.ForbiddenActions...),
		SafetyFacts:      readiness.SafetyFacts,
		NextSteps:        make([]executionCutoverNextStepJSON, 0, len(readiness.NextSteps)),
		GeneratedAt:      formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, executionCutoverReadinessItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			RequiredEvidence: append([]string{}, item.RequiredEvidence...),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	for _, step := range readiness.NextSteps {
		out.NextSteps = append(out.NextSteps, executionCutoverNextStepJSON{
			Key:         step.Key,
			Owner:       step.Owner,
			Action:      step.Action,
			RiskLevel:   step.RiskLevel,
			BlockedBy:   append([]string{}, step.BlockedBy...),
			NextCommand: step.NextCommand,
			Metadata:    step.Metadata,
		})
	}
	return out
}

func executionForwardingV1ApplyPreviewToJSON(preview project.ExecutionForwardingV1ApplyPreview) executionForwardingV1ApplyPreviewJSON {
	out := executionForwardingV1ApplyPreviewJSON{
		Project:              recordToJSON(preview.Project),
		Status:               preview.Status,
		Mode:                 preview.Mode,
		Readiness:            executionForwardingV1ReadinessToJSON(preview.Readiness),
		Items:                make([]executionForwardingV1ApplyPreviewItemJSON, 0, len(preview.Items)),
		AllowedTaskTypes:     append([]string{}, preview.AllowedTaskTypes...),
		ForwardingTargets:    make([]executionForwardingV1ForwardingTargetJSON, 0, len(preview.ForwardingTargets)),
		BlockedTargets:       make([]executionForwardingV1BlockedTargetJSON, 0, len(preview.BlockedTargets)),
		RequiredCapabilities: append([]string{}, preview.RequiredCapabilities...),
		ApplyPacketFields:    append([]string{}, preview.ApplyPacketFields...),
		FailClosedFields:     append([]string{}, preview.FailClosedFields...),
		RequiredProofFacts:   append([]string{}, preview.RequiredProofFacts...),
		RequiredEvidence:     append([]string{}, preview.RequiredEvidence...),
		ForbiddenActions:     append([]string{}, preview.ForbiddenActions...),
		ApprovalRequired:     preview.ApprovalRequired,
		ApprovalStatus:       preview.ApprovalStatus,
		ApplyOpen:            preview.ApplyOpen,
		RollbackTarget:       preview.RollbackTarget,
		SafetyFacts:          preview.SafetyFacts,
		GeneratedAt:          formatTime(preview.GeneratedAt),
	}
	for _, target := range preview.ForwardingTargets {
		out.ForwardingTargets = append(out.ForwardingTargets, executionForwardingV1ForwardingTargetJSON{
			TaskType:              target.TaskType,
			TargetCommandType:     target.TargetCommandType,
			TargetStatus:          target.TargetStatus,
			RequiredCapabilities:  append([]string{}, target.RequiredCapabilities...),
			RequiredPacketFields:  append([]string{}, target.RequiredPacketFields...),
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
		out.BlockedTargets = append(out.BlockedTargets, executionForwardingV1BlockedTargetJSON{
			TaskType:        target.TaskType,
			ForbiddenAction: target.ForbiddenAction,
			Reason:          target.Reason,
			FailureMode:     target.FailureMode,
			SafetyFacts:     target.SafetyFacts,
		})
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, executionForwardingV1ApplyPreviewItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			ApprovalStatus:   item.ApprovalStatus,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: append([]string{}, item.RequiredEvidence...),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return out
}

func executionForwardingV1ApplyPacketPreviewToJSON(preview project.ExecutionForwardingV1ApplyPacketPreview) executionForwardingV1ApplyPacketPreviewJSON {
	return executionForwardingV1ApplyPacketPreviewJSON{
		Project:             recordToJSON(preview.Project),
		Status:              preview.Status,
		Mode:                preview.Mode,
		Decision:            preview.Decision,
		Message:             preview.Message,
		ApplyPreview:        executionForwardingV1ApplyPreviewToJSON(preview.ApplyPreview),
		Gate:                executionForwardingV1ApplyGateToJSON(preview.Gate),
		Packet:              executionForwardingV1ApplyPacketToJSON(preview.Packet),
		ApplyGateCommand:    append([]string{}, preview.ApplyGateCommand...),
		FutureApplyCommand:  append([]string{}, preview.FutureApplyCommand...),
		RequiredHumanReview: append([]string{}, preview.RequiredHumanReview...),
		ForbiddenActions:    append([]string{}, preview.ForbiddenActions...),
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

func executionForwardingV1ApplyPacketToJSON(packet project.ExecutionForwardingV1ApplyPacket) executionForwardingV1ApplyPacketJSON {
	return executionForwardingV1ApplyPacketJSON{
		CommandType:                packet.CommandType,
		ProjectKey:                 packet.ProjectKey,
		AllowedTaskTypes:           append([]string{}, packet.AllowedTaskTypes...),
		TargetCommandTypes:         append([]string{}, packet.TargetCommandTypes...),
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

func executionForwardingV1ApplyGateToJSON(gate project.ExecutionForwardingV1ApplyGate) executionForwardingV1ApplyGateJSON {
	out := executionForwardingV1ApplyGateJSON{
		Project:                 recordToJSON(gate.Project),
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Decision:                gate.Decision,
		Message:                 gate.Message,
		Items:                   make([]executionForwardingV1ApplyGateItemJSON, 0, len(gate.Items)),
		RequiredPacketFields:    append([]string{}, gate.RequiredPacketFields...),
		RequiredCapabilities:    append([]string{}, gate.RequiredCapabilities...),
		AllowedTaskTypes:        append([]string{}, gate.AllowedTaskTypes...),
		TargetCommandTypes:      append([]string{}, gate.TargetCommandTypes...),
		BlockedTaskTypes:        append([]string{}, gate.BlockedTaskTypes...),
		ForbiddenActions:        append([]string{}, gate.ForbiddenActions...),
		FailClosedFields:        append([]string{}, gate.FailClosedFields...),
		RequiredProofFacts:      append([]string{}, gate.RequiredProofFacts...),
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
		out.Items = append(out.Items, executionForwardingV1ApplyGateItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Expected:         item.Expected,
			Actual:           item.Actual,
			RequiredEvidence: append([]string{}, item.RequiredEvidence...),
			BlockedBy:        append([]string{}, item.BlockedBy...),
		})
	}
	return out
}

func executionForwardingV1ApplyToJSON(result project.ApplyExecutionForwardingV1Result) executionForwardingV1ApplyJSON {
	return executionForwardingV1ApplyJSON{
		Project:                         recordToJSON(result.Project),
		Status:                          result.Status,
		Decision:                        result.Decision,
		Message:                         result.Message,
		Blockers:                        append([]string{}, result.Blockers...),
		Gate:                            executionForwardingV1ApplyGateToJSON(result.Gate),
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

func executionForwardingV1CommandPreviewToJSON(preview project.ExecutionForwardingV1CommandPreview) executionForwardingV1CommandPreviewJSON {
	return executionForwardingV1CommandPreviewJSON{
		Project:                                recordToJSON(preview.Project),
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
		RequiredPacketFields:                   append([]string{}, preview.RequiredPacketFields...),
		RequiredCapabilities:                   append([]string{}, preview.RequiredCapabilities...),
		FailClosedFields:                       append([]string{}, preview.FailClosedFields...),
		BlockedBy:                              append([]string{}, preview.BlockedBy...),
		AllowedTaskTypes:                       append([]string{}, preview.AllowedTaskTypes...),
		ForbiddenActions:                       append([]string{}, preview.ForbiddenActions...),
		SafetyFacts:                            preview.SafetyFacts,
		GeneratedAt:                            formatTime(preview.GeneratedAt),
	}
}

func executionForwardingV1RollbackPreviewToJSON(preview project.ExecutionForwardingV1RollbackPreview) executionForwardingV1RollbackPreviewJSON {
	out := executionForwardingV1RollbackPreviewJSON{
		Project:            recordToJSON(preview.Project),
		Status:             preview.Status,
		Mode:               preview.Mode,
		ApplyPreview:       executionForwardingV1ApplyPreviewToJSON(preview.ApplyPreview),
		Items:              make([]executionForwardingV1RollbackPreviewItemJSON, 0, len(preview.Items)),
		RollbackTarget:     preview.RollbackTarget,
		FailClosedSteps:    append([]string{}, preview.FailClosedSteps...),
		ReopenConditions:   append([]string{}, preview.ReopenConditions...),
		RequiredProofFacts: append([]string{}, preview.RequiredProofFacts...),
		RequiredEvidence:   append([]string{}, preview.RequiredEvidence...),
		ForbiddenActions:   append([]string{}, preview.ForbiddenActions...),
		RollbackApplyOpen:  preview.RollbackApplyOpen,
		SafetyFacts:        preview.SafetyFacts,
		GeneratedAt:        formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, executionForwardingV1RollbackPreviewItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			Owner:            item.Owner,
			RequiredEvidence: append([]string{}, item.RequiredEvidence...),
			NextCommand:      item.NextCommand,
			Metadata:         item.Metadata,
		})
	}
	return out
}

func cutoverReadinessToJSON(readiness project.ProjectCutoverReadiness) projectCutoverReadinessJSON {
	out := projectCutoverReadinessJSON{
		Project:         recordToJSON(readiness.Project),
		WorkflowVersion: workflowVersionToJSON(readiness.Version),
		Status:          readiness.Status,
		PhaseGate: projectPhaseGateJSON{
			Name:             readiness.PhaseGate.Name,
			Status:           readiness.PhaseGate.Status,
			AcceptedWarnings: readiness.PhaseGate.AcceptedWarnings,
			Blockers:         readiness.PhaseGate.Blockers,
		},
		Items:         make([]projectReadinessItemJSON, 0, len(readiness.Items)),
		Verification:  verificationBundleToJSON(readiness.Verification),
		Compatibility: compatibilityContractToJSON(readiness.Compatibility),
		Gates:         make([]gateResultJSON, 0, len(readiness.Gates)),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	for _, gate := range readiness.Gates {
		out.Gates = append(out.Gates, gateResultToJSON(gate))
	}
	return out
}

func cutoverApplyToJSON(result project.ApplyCutoverResult) projectCutoverApplyJSON {
	return projectCutoverApplyJSON{
		Project:                 recordToJSON(result.Project),
		WorkflowVersion:         workflowVersionToJSON(result.Version),
		Status:                  result.Status,
		Decision:                result.Decision,
		Message:                 result.Message,
		Blockers:                jsonStringSlice(result.Blockers),
		Warnings:                jsonStringSlice(result.Warnings),
		EventID:                 result.EventID,
		AuditEventID:            result.AuditEventID,
		IdempotencyKey:          result.IdempotencyKey,
		Created:                 result.Created,
		ProjectWriteAttempted:   result.ProjectWriteAttempted,
		ExecutionWriteAttempted: result.ExecutionWriteAttempted,
		CutoverReadinessGateID:  result.CutoverReadinessGateID,
	}
}

func statusProjectionApplyToJSON(result project.ApplyStatusProjectionResult) statusProjectionApplyJSON {
	return statusProjectionApplyJSON{
		Project:                   recordToJSON(result.Project),
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

func statusProjectionAuthorizationPreviewToJSON(preview project.StatusProjectionAuthorizationPreview) statusProjectionAuthorizationPreviewJSON {
	out := statusProjectionAuthorizationPreviewJSON{
		Project:                                recordToJSON(preview.Project),
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
		Permission:                             statusProjectionAuthorizationPermissionToJSON(preview.Permission),
		Preimage:                               statusProjectionPreimageToJSON(preview.Preimage),
		WriteSet:                               make([]statusProjectionWriteSetEntryJSON, 0, len(preview.WriteSet)),
		RequiredPreflight:                      jsonStringSlice(preview.RequiredPreflight),
		RequiredPacketFields:                   jsonStringSlice(preview.RequiredPacketFields),
		RequiredCapabilities:                   jsonStringSlice(preview.RequiredCapabilities),
		ProtectedPaths:                         jsonStringSlice(preview.ProtectedPaths),
		RollbackPlan:                           jsonStringSlice(preview.RollbackPlan),
		BlockedBy:                              jsonStringSlice(preview.BlockedBy),
		Warnings:                               jsonStringSlice(preview.Warnings),
		ForbiddenActions:                       jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:                            copyBoolMap(preview.SafetyFacts),
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
		out.WriteSet = append(out.WriteSet, statusProjectionWriteSetEntryToJSON(entry))
	}
	return out
}

func statusProjectionAuthorizationPermissionToJSON(permission project.StatusProjectionAuthorizationPermission) statusProjectionAuthorizationPermissionJSON {
	return statusProjectionAuthorizationPermissionJSON{
		Capability:        permission.Capability,
		ResourceType:      permission.ResourceType,
		TargetURI:         permission.TargetURI,
		CapabilityAllowed: permission.CapabilityAllowed,
		PathAllowed:       permission.PathAllowed,
		Allowed:           permission.Allowed,
		Reason:            permission.Reason,
	}
}

func statusProjectionPreimageToJSON(preimage project.StatusProjectionPreimage) statusProjectionPreimageJSON {
	return statusProjectionPreimageJSON{
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

func statusProjectionWriteSetEntryToJSON(entry project.StatusProjectionWriteSetEntry) statusProjectionWriteSetEntryJSON {
	return statusProjectionWriteSetEntryJSON{
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

func statusProjectionApplyGateToJSON(gate project.StatusProjectionApplyGate) statusProjectionApplyGateJSON {
	out := statusProjectionApplyGateJSON{
		Project:                        recordToJSON(gate.Project),
		Status:                         gate.Status,
		Mode:                           gate.Mode,
		ClaimScope:                     gate.ClaimScope,
		NotReal100:                     gate.NotReal100,
		Decision:                       gate.Decision,
		Message:                        gate.Message,
		TargetURI:                      gate.TargetURI,
		TargetPath:                     gate.TargetPath,
		Authorization:                  statusProjectionAuthorizationPreviewToJSON(gate.Authorization),
		Items:                          make([]statusProjectionApplyGateItemJSON, 0, len(gate.Items)),
		RequiredPacketFields:           jsonStringSlice(gate.RequiredPacketFields),
		RequiredCapabilities:           jsonStringSlice(gate.RequiredCapabilities),
		RequiredAuthorizationPhrase:    gate.RequiredAuthorizationPhrase,
		ProtectedPaths:                 jsonStringSlice(gate.ProtectedPaths),
		ForbiddenActions:               jsonStringSlice(gate.ForbiddenActions),
		SafetyFacts:                    copyBoolMap(gate.SafetyFacts),
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
		out.Items = append(out.Items, statusProjectionApplyGateItemToJSON(item))
	}
	return out
}

func statusProjectionApplyPacketPreviewToJSON(preview project.StatusProjectionApplyPacketPreview) statusProjectionApplyPacketPreviewJSON {
	return statusProjectionApplyPacketPreviewJSON{
		Project:                     recordToJSON(preview.Project),
		Status:                      preview.Status,
		Mode:                        preview.Mode,
		ClaimScope:                  preview.ClaimScope,
		NotReal100:                  preview.NotReal100,
		Decision:                    preview.Decision,
		Message:                     preview.Message,
		Blockers:                    jsonStringSlice(preview.Blockers),
		RequiredAuthorizationPhrase: preview.RequiredAuthorizationPhrase,
		Authorization:               statusProjectionAuthorizationPreviewToJSON(preview.Authorization),
		Gate:                        statusProjectionApplyGateToJSON(preview.Gate),
		Packet:                      statusProjectionApplyPacketToJSON(preview.Packet),
		ApplyCommand:                jsonStringSlice(preview.ApplyCommand),
		APIRequest:                  statusProjectionApplyAPIRequestToJSON(preview.APIRequest),
		RequiredHumanReview:         jsonStringSlice(preview.RequiredHumanReview),
		ForbiddenActions:            jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:                 copyBoolMap(preview.SafetyFacts),
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

func statusProjectionApplyPacketToJSON(packet project.StatusProjectionApplyPacket) statusProjectionApplyPacketJSON {
	return statusProjectionApplyPacketJSON{
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

func statusProjectionApplyAPIRequestToJSON(request project.StatusProjectionApplyAPIRequest) statusProjectionApplyAPIRequestJSON {
	return statusProjectionApplyAPIRequestJSON(statusProjectionApplyPacketJSON{
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
	})
}

func statusProjectionApplyGateItemToJSON(item project.StatusProjectionApplyGateItem) statusProjectionApplyGateItemJSON {
	return statusProjectionApplyGateItemJSON{
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

func statusProjectionsToJSON(record project.Record, projections []project.StatusProjectionRecord) statusProjectionsJSON {
	out := statusProjectionsJSON{
		Project:     recordToJSON(record),
		Projections: make([]statusProjectionJSON, 0, len(projections)),
	}
	for _, projection := range projections {
		out.Projections = append(out.Projections, statusProjectionToJSON(projection))
	}
	return out
}

func statusProjectionToJSON(projection project.StatusProjectionRecord) statusProjectionJSON {
	out := statusProjectionJSON{
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
		out.WrittenAt = formatTime(*projection.WrittenAt)
	}
	return out
}

func eventToJSON(event project.EventRecord) projectEventJSON {
	return projectEventJSON{
		ID:        event.ID,
		Type:      event.Type,
		Severity:  event.Severity,
		Message:   event.Message,
		Metadata:  event.Metadata,
		CreatedAt: formatTime(event.CreatedAt),
	}
}

func workflowVersionCreateToJSON(result project.CreateWorkflowVersionResult) workflowVersionCreateJSON {
	out := workflowVersionCreateJSON{
		Project:         recordToJSON(result.Project),
		WorkflowVersion: workflowVersionToJSON(result.Version),
		InitialItem:     workflowItemToJSON(result.InitialItem),
		StageItems:      make([]workflowItemJSON, 0, len(result.StageItems)),
		Created:         result.Created,
		IdempotencyKey:  result.IdempotencyKey,
	}
	for _, item := range result.StageItems {
		out.StageItems = append(out.StageItems, workflowItemToJSON(item))
	}
	return out
}

func workflowVersionListToJSON(record project.Record, versions []project.WorkflowVersion) workflowVersionListJSON {
	out := workflowVersionListJSON{
		Project:          recordToJSON(record),
		WorkflowVersions: make([]workflowVersionJSON, 0, len(versions)),
	}
	for _, version := range versions {
		out.WorkflowVersions = append(out.WorkflowVersions, workflowVersionToJSON(version))
	}
	return out
}

func workflowVersionStagesToJSON(record project.Record, version project.WorkflowVersion, items []project.WorkflowItem, links []project.WorkflowItemLink) workflowVersionStagesJSON {
	out := workflowVersionStagesJSON{
		Project:         recordToJSON(record),
		WorkflowVersion: workflowVersionToJSON(version),
		Items:           make([]workflowItemJSON, 0, len(items)),
		Links:           make([]workflowItemLinkJSON, 0, len(links)),
	}
	for _, item := range items {
		out.Items = append(out.Items, workflowItemToJSON(item))
	}
	for _, link := range links {
		out.Links = append(out.Links, workflowItemLinkToJSON(link))
	}
	return out
}

func ensureStageSkeletonToJSON(result project.EnsureStageSkeletonResult) ensureStageSkeletonJSON {
	out := ensureStageSkeletonJSON{
		Project:         recordToJSON(result.Project),
		WorkflowVersion: workflowVersionToJSON(result.Version),
		Items:           make([]workflowItemJSON, 0, len(result.Items)),
		Links:           make([]workflowItemLinkJSON, 0, len(result.Links)),
		Created:         result.Created,
	}
	for _, item := range result.Items {
		out.Items = append(out.Items, workflowItemToJSON(item))
	}
	for _, link := range result.Links {
		out.Links = append(out.Links, workflowItemLinkToJSON(link))
	}
	return out
}

func markWorkflowItemReadyToJSON(result project.MarkWorkflowItemReadyResult) markWorkflowItemReadyJSON {
	return markWorkflowItemReadyJSON{
		Project:         recordToJSON(result.Project),
		WorkflowVersion: workflowVersionToJSON(result.Version),
		Item:            workflowItemToJSON(result.Item),
		Artifact:        artifactToJSON(result.Artifact),
	}
}

func gateResultToJSON(result project.GateResult) gateResultJSON {
	return gateResultJSON{
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

func gateResultsToJSON(record project.Record, version project.WorkflowVersion, results []project.GateResult) gateResultsJSON {
	out := gateResultsJSON{
		Project:         recordToJSON(record),
		WorkflowVersion: workflowVersionToJSON(version),
		GateResults:     make([]gateResultJSON, 0, len(results)),
	}
	for _, result := range results {
		out.GateResults = append(out.GateResults, gateResultToJSON(result))
	}
	return out
}

func transitionPreviewToJSON(preview project.WorkflowTransitionPreview) transitionPreviewJSON {
	return transitionPreviewJSON{
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

func transitionPreviewsToJSON(record project.Record, version project.WorkflowVersion, previews []project.WorkflowTransitionPreview) transitionPreviewsJSON {
	out := transitionPreviewsJSON{
		Project:            recordToJSON(record),
		WorkflowVersion:    workflowVersionToJSON(version),
		TransitionPreviews: make([]transitionPreviewJSON, 0, len(previews)),
	}
	for _, preview := range previews {
		out.TransitionPreviews = append(out.TransitionPreviews, transitionPreviewToJSON(preview))
	}
	return out
}

func approvalRecordToJSON(approval project.ApprovalRecord) approvalRecordJSON {
	return approvalRecordJSON{
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

func approvalRecordsToJSON(record project.Record, version project.WorkflowVersion, approvals []project.ApprovalRecord) approvalRecordsJSON {
	out := approvalRecordsJSON{
		Project:         recordToJSON(record),
		WorkflowVersion: workflowVersionToJSON(version),
		ApprovalRecords: make([]approvalRecordJSON, 0, len(approvals)),
	}
	for _, approval := range approvals {
		out.ApprovalRecords = append(out.ApprovalRecords, approvalRecordToJSON(approval))
	}
	return out
}

func runnerPreviewToJSON(result project.RunnerPreviewResult) runnerPreviewJSON {
	out := runnerPreviewJSON{
		Project:         recordToJSON(result.Project),
		WorkflowVersion: workflowVersionToJSON(result.Version),
		Run:             runToJSON(result.Run),
		Tasks:           make([]runTaskJSON, 0, len(result.Tasks)),
		Attempts:        make([]runAttemptJSON, 0, len(result.Attempts)),
		Artifacts:       make([]artifactJSON, 0, len(result.Artifacts)),
		Preflight:       runnerPreflightToJSON(result.Preflight),
		Created:         result.Created,
		IdempotencyKey:  result.IdempotencyKey,
	}
	for _, task := range result.Tasks {
		out.Tasks = append(out.Tasks, runTaskToJSON(task))
	}
	for _, attempt := range result.Attempts {
		out.Attempts = append(out.Attempts, runAttemptToJSON(attempt))
	}
	for _, artifact := range result.Artifacts {
		out.Artifacts = append(out.Artifacts, artifactToJSON(artifact))
	}
	return out
}

func fixtureExecutionQueueToJSON(result project.FixtureExecutionQueueResult) fixtureExecutionQueueJSON {
	return fixtureExecutionQueueJSON{
		Project:                 recordToJSON(result.Project),
		WorkflowVersion:         workflowVersionToJSON(result.Version),
		Run:                     runToJSON(result.Run),
		Task:                    runTaskToJSON(result.Task),
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

func readOnlyVerifyQueueToJSON(result project.ReadOnlyVerifyQueueResult) readOnlyVerifyQueueJSON {
	return readOnlyVerifyQueueJSON{
		Project:                 recordToJSON(result.Project),
		WorkflowVersion:         workflowVersionToJSON(result.Version),
		Run:                     runToJSON(result.Run),
		Task:                    runTaskToJSON(result.Task),
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

func approvedArtifactWriteQueueToJSON(result project.ApprovedArtifactWriteQueueResult) approvedArtifactWriteQueueJSON {
	return approvedArtifactWriteQueueJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Task:                          runTaskToJSON(result.Task),
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

func fixtureProjectWriteQueueToJSON(result project.FixtureProjectWriteQueueResult) fixtureProjectWriteQueueJSON {
	return fixtureProjectWriteQueueJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Task:                          runTaskToJSON(result.Task),
		WriteSetArtifact:              artifactToJSON(result.WriteSetArtifact),
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

func managedGeneratedWriteQueueToJSON(result project.ManagedGeneratedWriteQueueResult) managedGeneratedWriteQueueJSON {
	return managedGeneratedWriteQueueJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Task:                          runTaskToJSON(result.Task),
		WriteSetArtifact:              artifactToJSON(result.WriteSetArtifact),
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

func runControlToJSON(result project.RunControlResult) runControlJSON {
	return runControlJSON{
		Project:                  recordToJSON(result.Project),
		Run:                      runToJSON(result.Run),
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

func executionApprovalGateToJSON(gate project.ExecutionApprovalGate) executionApprovalGateJSON {
	out := executionApprovalGateJSON{
		Project:                 recordToJSON(gate.Project),
		WorkflowVersion:         workflowVersionToJSON(gate.Version),
		Run:                     runToJSON(gate.Run),
		Status:                  gate.Status,
		Mode:                    gate.Mode,
		Items:                   make([]projectReadinessItemJSON, 0, len(gate.Items)),
		Blockers:                gate.Blockers,
		Warnings:                gate.Warnings,
		RequiredCapabilities:    gate.RequiredCapabilities,
		ApprovalFound:           gate.ApprovalFound,
		ApprovalGateFound:       gate.ApprovalGateFound,
		LiveMappingGateFound:    gate.LiveMappingGateFound,
		EnginePreview:           codexCLIAdapterPreviewToJSON(gate.EnginePreview),
		Workers:                 make([]workerJSON, 0, len(gate.Workers)),
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
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	if gate.ApprovalFound {
		out.Approval = approvalRecordToJSON(gate.Approval)
	}
	if gate.ApprovalGateFound {
		out.ApprovalGate = gateResultToJSON(gate.ApprovalGate)
	}
	if gate.LiveMappingGateFound {
		out.LiveMappingGate = gateResultToJSON(gate.LiveMappingGate)
	}
	for _, worker := range gate.Workers {
		out.Workers = append(out.Workers, workerToJSON(worker))
	}
	return out
}

func executionPlanPreviewToJSON(preview project.ExecutionPlanPreview) executionPlanPreviewJSON {
	out := executionPlanPreviewJSON{
		Project:                       recordToJSON(preview.Project),
		WorkflowVersion:               workflowVersionToJSON(preview.Version),
		Run:                           runToJSON(preview.Run),
		Gate:                          executionApprovalGateToJSON(preview.Gate),
		Status:                        preview.Status,
		Mode:                          preview.Mode,
		Steps:                         make([]executionPlanStepJSON, 0, len(preview.Steps)),
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
		out.Steps = append(out.Steps, executionPlanStepToJSON(step))
	}
	return out
}

func executionPlanStepToJSON(step project.ExecutionPlanStep) executionPlanStepJSON {
	return executionPlanStepJSON{
		Key:                  step.Key,
		AttemptKind:          step.AttemptKind,
		Status:               step.Status,
		Message:              step.Message,
		RequiredCapabilities: step.RequiredCapabilities,
		Prerequisites:        step.Prerequisites,
		Blockers:             step.Blockers,
		ReadsProject:         step.ReadsProject,
		WritesProject:        step.WritesProject,
		WritesAreaFlow:       step.WritesAreaFlow,
		UsesEngine:           step.UsesEngine,
		RunsCommands:         step.RunsCommands,
		UsesSecrets:          step.UsesSecrets,
		UsesNetwork:          step.UsesNetwork,
		CreatesAttempt:       step.CreatesAttempt,
		CreatesArtifact:      step.CreatesArtifact,
		Metadata:             step.Metadata,
	}
}

func projectWriteDesignGateToJSON(gate project.ProjectWriteDesignGate) projectWriteDesignGateJSON {
	out := projectWriteDesignGateJSON{
		Project:                       recordToJSON(gate.Project),
		WorkflowVersion:               workflowVersionToJSON(gate.Version),
		Run:                           runToJSON(gate.Run),
		Gate:                          executionApprovalGateToJSON(gate.Gate),
		Status:                        gate.Status,
		Mode:                          gate.Mode,
		Items:                         make([]projectReadinessItemJSON, 0, len(gate.Items)),
		RequiredCapabilities:          gate.RequiredCapabilities,
		WriteSetFields:                gate.WriteSetFields,
		UnsupportedOperations:         gate.UnsupportedOperations,
		ApplySequence:                 gate.ApplySequence,
		Blockers:                      gate.Blockers,
		ForbiddenActions:              gate.ForbiddenActions,
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
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func managedGeneratedWriteGateToJSON(gate project.ManagedGeneratedWriteGate) managedGeneratedWriteGateJSON {
	out := managedGeneratedWriteGateJSON{
		Project:                       recordToJSON(gate.Project),
		WorkflowVersion:               workflowVersionToJSON(gate.Version),
		Run:                           runToJSON(gate.Run),
		Gate:                          executionApprovalGateToJSON(gate.Gate),
		Status:                        gate.Status,
		Mode:                          gate.Mode,
		Items:                         make([]projectReadinessItemJSON, 0, len(gate.Items)),
		RequiredCapabilities:          gate.RequiredCapabilities,
		AllowedGeneratedPrefixes:      gate.AllowedGeneratedPrefixes,
		RequiredWriteSetFields:        gate.RequiredWriteSetFields,
		UnsupportedOperations:         gate.UnsupportedOperations,
		ApplySequence:                 gate.ApplySequence,
		Blockers:                      gate.Blockers,
		ForbiddenActions:              gate.ForbiddenActions,
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
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func workerToJSON(worker project.WorkerRecord) workerJSON {
	out := workerJSON{
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
		out.LastHeartbeatAt = formatTime(*worker.LastHeartbeatAt)
	}
	return out
}

func workerListToJSON(record project.Record, workers []project.WorkerRecord) workerListJSON {
	out := workerListJSON{
		Project: recordToJSON(record),
		Workers: make([]workerJSON, 0, len(workers)),
	}
	for _, worker := range workers {
		out.Workers = append(out.Workers, workerToJSON(worker))
	}
	return out
}

func workerPoolSummaryToJSON(summary project.WorkerPoolSummary) workerPoolSummaryJSON {
	out := workerPoolSummaryJSON{
		Projects:           make([]workerPoolProjectSummaryJSON, 0, len(summary.Projects)),
		TotalProjects:      summary.TotalProjects,
		TotalWorkers:       summary.TotalWorkers,
		TotalOnlineWorkers: summary.TotalOnlineWorkers,
		TotalActiveLeases:  summary.TotalActiveLeases,
		TotalQueuedTasks:   summary.TotalQueuedTasks,
		TotalNeedsRecovery: summary.TotalNeedsRecovery,
		GeneratedAt:        formatTime(summary.GeneratedAt),
	}
	for _, projectSummary := range summary.Projects {
		item := workerPoolProjectSummaryJSON{
			Project:             recordToJSON(projectSummary.Project),
			Workers:             projectSummary.Workers,
			OnlineWorkers:       projectSummary.OnlineWorkers,
			OfflineWorkers:      projectSummary.OfflineWorkers,
			ActiveLeases:        projectSummary.ActiveLeases,
			NeedsRecoveryLeases: projectSummary.NeedsRecoveryLeases,
			QueuedTasks:         projectSummary.QueuedTasks,
			NeedsRecoveryTasks:  projectSummary.NeedsRecoveryTasks,
			Capabilities:        jsonStringSlice(projectSummary.Capabilities),
			WorkerTypes:         jsonStringSlice(projectSummary.WorkerTypes),
			Scheduling:          schedulingPolicyToJSON(projectSummary.Scheduling),
			Role:                roleReadinessToJSON(projectSummary.Role),
			Engine:              engineReadinessToJSON(projectSummary.Engine),
			Resources:           resourceReadinessToJSON(projectSummary.Resources),
		}
		if projectSummary.LastWorkerHeartbeat != nil {
			item.LastWorkerHeartbeat = formatTime(*projectSummary.LastWorkerHeartbeat)
		}
		out.Projects = append(out.Projects, item)
	}
	return out
}

func localServiceStatusToJSON(status project.LocalServiceStatus) localServiceStatusJSON {
	return localServiceStatusJSON{
		Status: status.Status,
		Mode:   status.Mode,
		API: localServiceComponentJSON{
			Status:  status.API.Status,
			Message: status.API.Message,
		},
		Database: localServiceComponentJSON{
			Status:  status.Database.Status,
			Message: status.Database.Message,
		},
		WorkerPool: localServiceWorkerPoolJSON{
			Status:             status.WorkerPool.Status,
			Message:            status.WorkerPool.Message,
			TotalProjects:      status.WorkerPool.TotalProjects,
			TotalWorkers:       status.WorkerPool.TotalWorkers,
			TotalOnlineWorkers: status.WorkerPool.TotalOnlineWorkers,
			TotalActiveLeases:  status.WorkerPool.TotalActiveLeases,
			TotalQueuedTasks:   status.WorkerPool.TotalQueuedTasks,
			TotalNeedsRecovery: status.WorkerPool.TotalNeedsRecovery,
		},
		Dashboard: localServiceDashboardJSON{
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

func desktopServiceControlGateToJSON(gate project.DesktopServiceControlGate) desktopServiceControlGateJSON {
	out := desktopServiceControlGateJSON{
		Status:                   gate.Status,
		Mode:                     gate.Mode,
		Actions:                  make([]desktopServiceControlActionJSON, 0, len(gate.Actions)),
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
		out.Actions = append(out.Actions, desktopServiceControlActionJSON{
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
	return out
}

func desktopNotificationGateToJSON(gate project.DesktopNotificationGate) desktopNotificationGateJSON {
	out := desktopNotificationGateJSON{
		Status:                   gate.Status,
		Mode:                     gate.Mode,
		Actions:                  make([]desktopNotificationActionJSON, 0, len(gate.Actions)),
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
		out.Actions = append(out.Actions, desktopNotificationActionJSON{
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
	return out
}

func desktopTrayMenuGateToJSON(gate project.DesktopTrayMenuGate) desktopTrayMenuGateJSON {
	out := desktopTrayMenuGateJSON{
		Status:                   gate.Status,
		Mode:                     gate.Mode,
		Actions:                  make([]desktopTrayMenuActionJSON, 0, len(gate.Actions)),
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
		out.Actions = append(out.Actions, desktopTrayMenuActionJSON{
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
	return out
}

func securityBoundaryReadinessToJSON(readiness project.SecurityBoundaryReadiness) securityBoundaryReadinessJSON {
	out := securityBoundaryReadinessJSON{
		Status:                        readiness.Status,
		Mode:                          readiness.Mode,
		Items:                         make([]securityBoundaryReadinessItemJSON, 0, len(readiness.Items)),
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
		out.Items = append(out.Items, securityBoundaryReadinessItemJSON{
			Key:              item.Key,
			Category:         item.Category,
			Status:           item.Status,
			Message:          item.Message,
			RequiredEvidence: jsonStringSlice(item.RequiredEvidence),
			BlockedBy:        jsonStringSlice(item.BlockedBy),
			Metadata:         item.Metadata,
		})
	}
	return out
}

func completionAuditToJSON(audit project.CompletionAudit) completionAuditJSON {
	guardrail := completionAuditReal100Guardrail(audit.Real100Guardrail)
	out := completionAuditJSON{
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
		Items:                      make([]completionAuditItemJSON, 0, len(audit.Items)),
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
		out.Items = append(out.Items, completionAuditItemJSON{
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
	return out
}

func supportBundlePreviewToJSON(preview project.SupportBundlePreview) supportBundlePreviewJSON {
	out := supportBundlePreviewJSON{
		Status:                   preview.Status,
		Mode:                     preview.Mode,
		BundleID:                 preview.BundleID,
		Scope:                    preview.Scope,
		Projects:                 make([]projectRecordJSON, 0, len(preview.Projects)),
		IncludedMetadata:         jsonStringSlice(preview.IncludedMetadata),
		ExcludedSensitiveContent: jsonStringSlice(preview.ExcludedSensitiveContent),
		PathReferences:           make([]supportBundlePathReferenceJSON, 0, len(preview.PathReferences)),
		Hashes:                   make([]supportBundleHashReferenceJSON, 0, len(preview.Hashes)),
		Capabilities:             jsonStringSlice(preview.Capabilities),
		ForbiddenActions:         jsonStringSlice(preview.ForbiddenActions),
		SafetyFacts:              preview.SafetyFacts,
		GeneratedAt:              formatTime(preview.GeneratedAt),
	}
	for _, record := range preview.Projects {
		out.Projects = append(out.Projects, recordToJSON(record))
	}
	for _, ref := range preview.PathReferences {
		out.PathReferences = append(out.PathReferences, supportBundlePathReferenceJSON{
			Key:         ref.Key,
			Kind:        ref.Kind,
			URI:         ref.URI,
			ProjectKey:  ref.ProjectKey,
			Description: ref.Description,
		})
	}
	for _, hash := range preview.Hashes {
		out.Hashes = append(out.Hashes, supportBundleHashReferenceJSON{
			Key:         hash.Key,
			Hash:        hash.Hash,
			Source:      hash.Source,
			Description: hash.Description,
		})
	}
	return out
}

func migrationLedgerReadinessToJSON(readiness project.MigrationLedgerReadiness) migrationLedgerReadinessJSON {
	out := migrationLedgerReadinessJSON{
		Status:                               readiness.Status,
		Mode:                                 readiness.Mode,
		Entries:                              make([]migrationLedgerEntryJSON, 0, len(readiness.Entries)),
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
		entryJSON := migrationLedgerEntryJSON{
			Name:             entry.Name,
			Applied:          entry.Applied,
			Status:           entry.Status,
			RequiredEvidence: jsonStringSlice(entry.RequiredEvidence),
			Phases:           make([]migrationLedgerPhaseJSON, 0, len(entry.Phases)),
			Metadata:         jsonObject(entry.Metadata),
		}
		for _, phase := range entry.Phases {
			entryJSON.Phases = append(entryJSON.Phases, migrationLedgerPhaseJSON{
				Phase:       phase.Phase,
				Status:      phase.Status,
				Message:     phase.Message,
				Remediation: phase.Remediation,
				Metadata:    jsonObject(phase.Metadata),
			})
		}
		out.Entries = append(out.Entries, entryJSON)
	}
	return out
}

func operationsReadinessToJSON(readiness project.OperationsReadiness) operationsReadinessJSON {
	out := operationsReadinessJSON{
		Status:              readiness.Status,
		Mode:                readiness.Mode,
		Items:               make([]operationsReadinessItemJSON, 0, len(readiness.Items)),
		ServiceStatus:       localServiceStatusToJSON(readiness.ServiceStatus),
		SupportBundle:       supportBundlePreviewToJSON(readiness.SupportBundle),
		MigrationLedger:     migrationLedgerReadinessToJSON(readiness.MigrationLedger),
		Capabilities:        jsonStringSlice(readiness.Capabilities),
		ForbiddenActions:    jsonStringSlice(readiness.ForbiddenActions),
		SafetyFacts:         readiness.SafetyFacts,
		TelemetryDefault:    readiness.TelemetryDefault,
		ManagedOpsStatus:    readiness.ManagedOpsStatus,
		SupportExportStatus: readiness.SupportExportStatus,
		GeneratedAt:         formatTime(readiness.GeneratedAt),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, operationsReadinessItemJSON{
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
	return out
}

func backupManifestToJSON(manifest project.BackupManifest) backupManifestJSON {
	out := backupManifestJSON{
		Status:           manifest.Status,
		Mode:             manifest.Mode,
		Scope:            manifest.Scope,
		ProjectKey:       manifest.ProjectKey,
		SchemaVersion:    manifest.SchemaVersion,
		GeneratedAt:      formatTime(manifest.GeneratedAt),
		ManifestHash:     manifest.ManifestHash,
		TableCounts:      make([]backupTableCountJSON, 0, len(manifest.TableCounts)),
		Projects:         make([]backupProjectManifestJSON, 0, len(manifest.Projects)),
		Capabilities:     jsonStringSlice(manifest.Capabilities),
		ForbiddenActions: jsonStringSlice(manifest.ForbiddenActions),
	}
	for _, table := range manifest.TableCounts {
		out.TableCounts = append(out.TableCounts, backupTableCountJSON{
			Table: table.Table,
			Rows:  table.Rows,
		})
	}
	for _, projectManifest := range manifest.Projects {
		item := backupProjectManifestJSON{
			Project: recordToJSON(projectManifest.Project),
			Inventory: projectInventoryJSON{
				Versions:        projectManifest.Inventory.Versions,
				Residuals:       projectManifest.Inventory.Residuals,
				Artifacts:       projectManifest.Inventory.Artifacts,
				ImportSnapshots: projectManifest.Inventory.ImportSnapshots,
				MirrorExports:   projectManifest.Inventory.MirrorExports,
			},
			ArtifactCount: projectManifest.ArtifactCount,
			Artifacts:     make([]backupArtifactSummaryJSON, 0, len(projectManifest.Artifacts)),
		}
		for _, artifact := range projectManifest.Artifacts {
			item.Artifacts = append(item.Artifacts, backupArtifactSummaryJSON{
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
		out.Projects = append(out.Projects, item)
	}
	return out
}

func restorePlanToJSON(plan project.RestorePlan) restorePlanJSON {
	out := restorePlanJSON{
		Status:           plan.Status,
		Mode:             plan.Mode,
		Scope:            plan.Scope,
		ProjectKey:       plan.ProjectKey,
		SchemaVersion:    plan.SchemaVersion,
		ManifestHash:     plan.ManifestHash,
		Projects:         make([]projectRecordJSON, 0, len(plan.Projects)),
		Items:            make([]restorePlanItemJSON, 0, len(plan.Items)),
		Capabilities:     jsonStringSlice(plan.Capabilities),
		ForbiddenActions: jsonStringSlice(plan.ForbiddenActions),
		GeneratedAt:      formatTime(plan.GeneratedAt),
	}
	for _, record := range plan.Projects {
		out.Projects = append(out.Projects, recordToJSON(record))
	}
	for _, item := range plan.Items {
		out.Items = append(out.Items, restorePlanItemJSON{
			Key:      item.Key,
			Category: item.Category,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func releaseReadinessToJSON(readiness project.ReleaseReadiness) releaseReadinessJSON {
	guardrail := releasePreviewReal100Guardrail(readiness.Real100Guardrail)
	out := releaseReadinessJSON{
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
		Backup:                     backupManifestToJSON(readiness.Backup),
		RestorePlan:                restorePlanToJSON(readiness.RestorePlan),
		AuditCoverage:              auditCoverageToJSON(readiness.AuditCoverage),
		Projects:                   make([]releaseReadinessProjectJSON, 0, len(readiness.Projects)),
		Items:                      make([]releaseReadinessItemJSON, 0, len(readiness.Items)),
		Capabilities:               jsonStringSlice(readiness.Capabilities),
		ForbiddenActions:           jsonStringSlice(readiness.ForbiddenActions),
		GeneratedAt:                formatTime(readiness.GeneratedAt),
	}
	for _, projectReadiness := range readiness.Projects {
		out.Projects = append(out.Projects, releaseReadinessProjectJSON{
			Project:             recordToJSON(projectReadiness.Project),
			Permission:          permissionPolicyDoctorToJSON(projectReadiness.Permission),
			ArtifactIntegrity:   artifactIntegrityToJSON(projectReadiness.ArtifactIntegrity),
			Conformance:         conformanceToJSON(projectReadiness.Conformance),
			Status:              projectReadiness.Status,
			NeedsAttentionItems: projectReadiness.NeedsAttentionItems,
			BlockedItems:        projectReadiness.BlockedItems,
		})
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, releaseReadinessItemJSON{
			Key:      item.Key,
			Category: item.Category,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func releaseRemediationPlanToJSON(plan project.ReleaseRemediationPlan) releaseRemediationPlanJSON {
	guardrail := releasePreviewReal100Guardrail(plan.Real100Guardrail)
	out := releaseRemediationPlanJSON{
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
		Readiness:                  releaseReadinessToJSON(plan.Readiness),
		Actions:                    make([]releaseRemediationActionJSON, 0, len(plan.Actions)),
		Capabilities:               jsonStringSlice(plan.Capabilities),
		ForbiddenActions:           jsonStringSlice(plan.ForbiddenActions),
		GeneratedAt:                formatTime(plan.GeneratedAt),
	}
	for _, action := range plan.Actions {
		out.Actions = append(out.Actions, releaseRemediationActionJSON{
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
	return out
}

func releaseAcceptancePreviewToJSON(preview project.ReleaseAcceptancePreview) releaseAcceptancePreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releaseAcceptancePreviewJSON{
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
		Remediation:                releaseRemediationPlanToJSON(preview.Remediation),
		Decisions:                  make([]releaseAcceptanceDecisionJSON, 0, len(preview.Decisions)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, decision := range preview.Decisions {
		out.Decisions = append(out.Decisions, releaseAcceptanceDecisionJSON{
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
	return out
}

func releaseAcceptanceGateToJSON(gate project.ReleaseAcceptanceGate) releaseAcceptanceGateJSON {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	out := releaseAcceptanceGateJSON{
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
		Preview:                    releaseAcceptancePreviewToJSON(gate.Preview),
		Items:                      make([]releaseAcceptanceGateItemJSON, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		out.Items = append(out.Items, releaseAcceptanceGateItemJSON{
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
	return out
}

func releaseExceptionDoctorToJSON(doctor project.ReleaseExceptionDoctor) releaseExceptionDoctorJSON {
	guardrail := releasePreviewReal100Guardrail(doctor.Real100Guardrail)
	out := releaseExceptionDoctorJSON{
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
		Gate:                       releaseAcceptanceGateToJSON(doctor.Gate),
		Checks:                     make([]releaseExceptionDoctorCheckJSON, 0, len(doctor.Checks)),
		Capabilities:               jsonStringSlice(doctor.Capabilities),
		ForbiddenActions:           jsonStringSlice(doctor.ForbiddenActions),
		GeneratedAt:                formatTime(doctor.GeneratedAt),
	}
	for _, check := range doctor.Checks {
		out.Checks = append(out.Checks, releaseExceptionDoctorCheckJSON{
			Key:      check.Key,
			Category: check.Category,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return out
}

func releaseExceptionRecordPreviewToJSON(preview project.ReleaseExceptionRecordPreview) releaseExceptionRecordPreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releaseExceptionRecordPreviewJSON{
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
		Doctor:                     releaseExceptionDoctorToJSON(preview.Doctor),
		Drafts:                     make([]releaseExceptionRecordDraftJSON, 0, len(preview.Drafts)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, draft := range preview.Drafts {
		out.Drafts = append(out.Drafts, releaseExceptionRecordDraftJSON{
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
	return out
}

func releaseExceptionRecordToJSON(record project.ReleaseExceptionRecord) releaseExceptionRecordJSON {
	out := releaseExceptionRecordJSON{
		ID:               record.ID,
		ProjectID:        record.ProjectID,
		ProjectKey:       record.ProjectKey,
		ExceptionKey:     record.ExceptionKey,
		SourceGateItem:   record.SourceGateItem,
		SourceDecision:   record.SourceDecision,
		AcceptanceType:   record.AcceptanceType,
		Status:           record.Status,
		Owner:            record.Owner,
		Reason:           record.Reason,
		RequiredEvidence: record.RequiredEvidence,
		RollbackPlan:     record.RollbackPlan,
		ReviewRequired:   record.ReviewRequired,
		RequestedBy:      record.RequestedBy,
		ApprovedBy:       record.ApprovedBy,
		RevokedBy:        record.RevokedBy,
		DecisionReason:   record.DecisionReason,
		AuditEventID:     record.AuditEventID,
		Metadata:         record.Metadata,
		CreatedAt:        record.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        record.UpdatedAt.UTC().Format(time.RFC3339),
		IdempotencyKey:   record.IdempotencyKey,
		Created:          record.Created,
	}
	if record.ReviewAt != nil {
		out.ReviewAt = record.ReviewAt.UTC().Format(time.RFC3339)
	}
	if record.ExpiresAt != nil {
		out.ExpiresAt = record.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if record.ApprovedAt != nil {
		out.ApprovedAt = record.ApprovedAt.UTC().Format(time.RFC3339)
	}
	if record.RevokedAt != nil {
		out.RevokedAt = record.RevokedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func releaseExceptionSchemaPreviewToJSON(preview project.ReleaseExceptionSchemaPreview) releaseExceptionSchemaPreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releaseExceptionSchemaPreviewJSON{
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
		RecordPreview:              releaseExceptionRecordPreviewToJSON(preview.RecordPreview),
		Tables:                     make([]releaseExceptionSchemaTableJSON, 0, len(preview.Tables)),
		ApplySteps:                 make([]releaseExceptionMigrationStepJSON, 0, len(preview.ApplySteps)),
		RollbackSteps:              make([]releaseExceptionMigrationStepJSON, 0, len(preview.RollbackSteps)),
		AuditActions:               jsonStringSlice(preview.AuditActions),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, table := range preview.Tables {
		out.Tables = append(out.Tables, releaseExceptionSchemaTableToJSON(table))
	}
	for _, step := range preview.ApplySteps {
		out.ApplySteps = append(out.ApplySteps, releaseExceptionMigrationStepToJSON(step))
	}
	for _, step := range preview.RollbackSteps {
		out.RollbackSteps = append(out.RollbackSteps, releaseExceptionMigrationStepToJSON(step))
	}
	return out
}

func releaseExceptionSchemaTableToJSON(table project.ReleaseExceptionSchemaTable) releaseExceptionSchemaTableJSON {
	out := releaseExceptionSchemaTableJSON{
		Name:        table.Name,
		Purpose:     table.Purpose,
		Columns:     make([]releaseExceptionSchemaColumnJSON, 0, len(table.Columns)),
		Indexes:     make([]releaseExceptionSchemaIndexJSON, 0, len(table.Indexes)),
		ForeignKeys: make([]releaseExceptionSchemaForeignKeyJSON, 0, len(table.ForeignKeys)),
	}
	for _, column := range table.Columns {
		out.Columns = append(out.Columns, releaseExceptionSchemaColumnJSON{
			Name:     column.Name,
			Type:     column.Type,
			Nullable: column.Nullable,
			Purpose:  column.Purpose,
		})
	}
	for _, index := range table.Indexes {
		out.Indexes = append(out.Indexes, releaseExceptionSchemaIndexJSON{
			Name:    index.Name,
			Columns: jsonStringSlice(index.Columns),
			Unique:  index.Unique,
			Purpose: index.Purpose,
		})
	}
	for _, foreignKey := range table.ForeignKeys {
		out.ForeignKeys = append(out.ForeignKeys, releaseExceptionSchemaForeignKeyJSON{
			Column:           foreignKey.Column,
			ReferencesTable:  foreignKey.ReferencesTable,
			ReferencesColumn: foreignKey.ReferencesColumn,
			OnDelete:         foreignKey.OnDelete,
		})
	}
	return out
}

func releaseExceptionMigrationStepToJSON(step project.ReleaseExceptionMigrationStep) releaseExceptionMigrationStepJSON {
	return releaseExceptionMigrationStepJSON{
		Order:       step.Order,
		Action:      step.Action,
		Description: step.Description,
		SQLPreview:  step.SQLPreview,
	}
}

func releaseExceptionMigrationApprovalGateToJSON(gate project.ReleaseExceptionMigrationApprovalGate) releaseExceptionMigrationApprovalGateJSON {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	out := releaseExceptionMigrationApprovalGateJSON{
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
		SchemaPreview:              releaseExceptionSchemaPreviewToJSON(gate.SchemaPreview),
		Items:                      make([]releaseExceptionMigrationApprovalItemJSON, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		out.Items = append(out.Items, releaseExceptionMigrationApprovalItemJSON{
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
	return out
}

func releaseExceptionApplyPreviewToJSON(preview project.ReleaseExceptionApplyPreview) releaseExceptionApplyPreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releaseExceptionApplyPreviewJSON{
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
		MigrationGate:              releaseExceptionMigrationApprovalGateToJSON(preview.MigrationGate),
		Items:                      make([]releaseExceptionApplyPreviewItemJSON, 0, len(preview.Items)),
		ApplySteps:                 make([]releaseExceptionApplyPreviewStepJSON, 0, len(preview.ApplySteps)),
		RollbackSteps:              make([]releaseExceptionApplyPreviewStepJSON, 0, len(preview.RollbackSteps)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, releaseExceptionApplyPreviewItemJSON{
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
		out.ApplySteps = append(out.ApplySteps, releaseExceptionApplyPreviewStepJSON{
			Order:       step.Order,
			Action:      step.Action,
			Description: step.Description,
			BlockedBy:   jsonStringSlice(step.BlockedBy),
		})
	}
	for _, step := range preview.RollbackSteps {
		out.RollbackSteps = append(out.RollbackSteps, releaseExceptionApplyPreviewStepJSON{
			Order:       step.Order,
			Action:      step.Action,
			Description: step.Description,
			BlockedBy:   jsonStringSlice(step.BlockedBy),
		})
	}
	return out
}

func releaseFinalGateToJSON(gate project.ReleaseFinalGate) releaseFinalGateJSON {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	out := releaseFinalGateJSON{
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
		Readiness:                  releaseReadinessToJSON(gate.Readiness),
		AcceptanceGate:             releaseAcceptanceGateToJSON(gate.AcceptanceGate),
		ExceptionApply:             releaseExceptionApplyPreviewToJSON(gate.ExceptionApply),
		Items:                      make([]releaseFinalGateItemJSON, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		out.Items = append(out.Items, releaseFinalGateItemJSON{
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
	return out
}

func releaseEvidenceBundleToJSON(bundle project.ReleaseEvidenceBundle) releaseEvidenceBundleJSON {
	guardrail := releasePreviewReal100Guardrail(bundle.Real100Guardrail)
	out := releaseEvidenceBundleJSON{
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
		FinalGate:                  releaseFinalGateToJSON(bundle.FinalGate),
		Backup:                     backupManifestToJSON(bundle.Backup),
		AuditCoverage:              auditCoverageToJSON(bundle.AuditCoverage),
		Items:                      make([]releaseEvidenceBundleItemJSON, 0, len(bundle.Items)),
		Capabilities:               jsonStringSlice(bundle.Capabilities),
		ForbiddenActions:           jsonStringSlice(bundle.ForbiddenActions),
		GeneratedAt:                formatTime(bundle.GeneratedAt),
	}
	for _, item := range bundle.Items {
		out.Items = append(out.Items, releaseEvidenceBundleItemJSON{
			Key:         item.Key,
			Category:    item.Category,
			Status:      item.Status,
			Source:      item.Source,
			Description: item.Description,
			Metadata:    item.Metadata,
		})
	}
	return out
}

func releasePackagePreviewToJSON(preview project.ReleasePackagePreview) releasePackagePreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releasePackagePreviewJSON{
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
		EvidenceBundle:             releaseEvidenceBundleToJSON(preview.EvidenceBundle),
		PackageName:                preview.PackageName,
		Items:                      make([]releasePackagePreviewItemJSON, 0, len(preview.Items)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, releasePackagePreviewItemJSON{
			Key:         item.Key,
			Category:    item.Category,
			Status:      item.Status,
			PackagePath: item.PackagePath,
			Source:      item.Source,
			Description: item.Description,
			Metadata:    item.Metadata,
		})
	}
	return out
}

func releaseDistributionPreviewToJSON(preview project.ReleaseDistributionPreview) releaseDistributionPreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releaseDistributionPreviewJSON{
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
		PackagePreview:             releasePackagePreviewToJSON(preview.PackagePreview),
		Items:                      make([]releaseDistributionPreviewItemJSON, 0, len(preview.Items)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, releaseDistributionPreviewItemJSON{
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
	return out
}

func releasePublishGateToJSON(gate project.ReleasePublishGate) releasePublishGateJSON {
	guardrail := releasePreviewReal100Guardrail(gate.Real100Guardrail)
	out := releasePublishGateJSON{
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
		DistributionPreview:        releaseDistributionPreviewToJSON(gate.DistributionPreview),
		Items:                      make([]releasePublishGateItemJSON, 0, len(gate.Items)),
		Capabilities:               jsonStringSlice(gate.Capabilities),
		ForbiddenActions:           jsonStringSlice(gate.ForbiddenActions),
		GeneratedAt:                formatTime(gate.GeneratedAt),
	}
	for _, item := range gate.Items {
		out.Items = append(out.Items, releasePublishGateItemJSON{
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
	return out
}

func releasePublishApprovalPreviewToJSON(preview project.ReleasePublishApprovalPreview) releasePublishApprovalPreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releasePublishApprovalPreviewJSON{
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
		PublishGate:                releasePublishGateToJSON(preview.PublishGate),
		Items:                      make([]releasePublishApprovalPreviewItemJSON, 0, len(preview.Items)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, releasePublishApprovalPreviewItemJSON{
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
	return out
}

func releaseRolloutPlanPreviewToJSON(preview project.ReleaseRolloutPlanPreview) releaseRolloutPlanPreviewJSON {
	guardrail := releasePreviewReal100Guardrail(preview.Real100Guardrail)
	out := releaseRolloutPlanPreviewJSON{
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
		PublishApprovalPreview:     releasePublishApprovalPreviewToJSON(preview.PublishApprovalPreview),
		Items:                      make([]releaseRolloutPlanPreviewItemJSON, 0, len(preview.Items)),
		RolloutSteps:               make([]releaseRolloutPlanPreviewStepJSON, 0, len(preview.RolloutSteps)),
		VerificationCheckpoints:    make([]releaseRolloutPlanPreviewStepJSON, 0, len(preview.VerificationCheckpoints)),
		RollbackSteps:              make([]releaseRolloutPlanPreviewStepJSON, 0, len(preview.RollbackSteps)),
		Capabilities:               jsonStringSlice(preview.Capabilities),
		ForbiddenActions:           jsonStringSlice(preview.ForbiddenActions),
		GeneratedAt:                formatTime(preview.GeneratedAt),
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, releaseRolloutPlanPreviewItemJSON{
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
		out.RolloutSteps = append(out.RolloutSteps, releaseRolloutPlanPreviewStepToJSON(step))
	}
	for _, checkpoint := range preview.VerificationCheckpoints {
		out.VerificationCheckpoints = append(out.VerificationCheckpoints, releaseRolloutPlanPreviewStepToJSON(checkpoint))
	}
	for _, step := range preview.RollbackSteps {
		out.RollbackSteps = append(out.RollbackSteps, releaseRolloutPlanPreviewStepToJSON(step))
	}
	return out
}

func releaseRolloutPlanPreviewStepToJSON(step project.ReleaseRolloutPlanPreviewStep) releaseRolloutPlanPreviewStepJSON {
	return releaseRolloutPlanPreviewStepJSON{
		Order:       step.Order,
		Stage:       step.Stage,
		Action:      step.Action,
		Description: step.Description,
		BlockedBy:   jsonStringSlice(step.BlockedBy),
	}
}

func auditCoverageToJSON(coverage project.AuditCoverage) auditCoverageJSON {
	out := auditCoverageJSON{
		Status:              coverage.Status,
		Mode:                coverage.Mode,
		Scope:               coverage.Scope,
		ProjectID:           coverage.ProjectID,
		ProjectKey:          coverage.ProjectKey,
		TotalAuditEvents:    coverage.TotalAuditEvents,
		CoveredRequirements: coverage.CoveredRequirements,
		GapRequirements:     coverage.GapRequirements,
		Requirements:        make([]auditCoverageRequirementJSON, 0, len(coverage.Requirements)),
		GeneratedAt:         formatTime(coverage.GeneratedAt),
	}
	for _, requirement := range coverage.Requirements {
		item := auditCoverageRequirementJSON{
			Key:             requirement.Key,
			Category:        requirement.Category,
			Description:     requirement.Description,
			Status:          requirement.Status,
			EvidenceCount:   requirement.EvidenceCount,
			RequiredActions: make([]auditCoverageActionEvidenceJSON, 0, len(requirement.RequiredActions)),
			MissingActions:  jsonStringSlice(requirement.MissingActions),
		}
		if requirement.LastAuditAt != nil {
			item.LastAuditAt = formatTime(*requirement.LastAuditAt)
		}
		for _, action := range requirement.RequiredActions {
			actionItem := auditCoverageActionEvidenceJSON{
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
		out.Requirements = append(out.Requirements, item)
	}
	return out
}

func permissionPolicyDoctorToJSON(doctor project.PermissionPolicyDoctor) permissionPolicyDoctorJSON {
	out := permissionPolicyDoctorJSON{
		Status:      doctor.Status,
		Mode:        doctor.Mode,
		Project:     recordToJSON(doctor.Project),
		Checks:      make([]permissionPolicyCheckJSON, 0, len(doctor.Checks)),
		GeneratedAt: formatTime(doctor.GeneratedAt),
	}
	for _, check := range doctor.Checks {
		out.Checks = append(out.Checks, permissionPolicyCheckJSON{
			Key:      check.Key,
			Category: check.Category,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return out
}

func artifactIntegrityToJSON(report project.ArtifactIntegrityReport) artifactIntegrityJSON {
	out := artifactIntegrityJSON{
		Status:           report.Status,
		Mode:             report.Mode,
		Project:          recordToJSON(report.Project),
		CheckedArtifacts: report.CheckedArtifacts,
		PassedArtifacts:  report.PassedArtifacts,
		WarnArtifacts:    report.WarnArtifacts,
		FailedArtifacts:  report.FailedArtifacts,
		SkippedArtifacts: report.SkippedArtifacts,
		Checks:           make([]artifactIntegrityCheckJSON, 0, len(report.Checks)),
		GeneratedAt:      formatTime(report.GeneratedAt),
	}
	for _, check := range report.Checks {
		out.Checks = append(out.Checks, artifactIntegrityCheckJSON{
			Artifact: artifactToJSON(check.Artifact),
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return out
}

func artifactArchivePreviewToJSON(preview project.ArtifactArchivePreviewResult) artifactArchivePreviewJSON {
	out := artifactArchivePreviewJSON{
		Project: recordToJSON(preview.Project),
		Status:  preview.Status,
		Mode:    preview.Mode,
		Summary: artifactArchivePreviewSummaryJSON{
			TotalArtifacts:    preview.Summary.TotalArtifacts,
			ArchiveCandidates: preview.Summary.ArchiveCandidates,
			RetainedArtifacts: preview.Summary.RetainedArtifacts,
			ExternalRefs:      preview.Summary.ExternalRefs,
			NeedsPolicy:       preview.Summary.NeedsPolicy,
		},
		Items:                   make([]artifactArchivePreviewItemJSON, 0, len(preview.Items)),
		EventID:                 preview.EventID,
		AuditEventID:            preview.AuditEventID,
		IdempotencyKey:          preview.IdempotencyKey,
		Created:                 preview.Created,
		GeneratedAt:             formatTime(preview.GeneratedAt),
		ProjectWriteAttempted:   preview.ProjectWriteAttempted,
		StorageWriteAttempted:   preview.StorageWriteAttempted,
		ArtifactDeleteAttempted: preview.ArtifactDeleteAttempted,
	}
	for _, item := range preview.Items {
		out.Items = append(out.Items, artifactArchivePreviewItemJSON{
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
	return out
}

func conformanceToJSON(report project.ConformanceReport) conformanceJSON {
	out := conformanceJSON{
		Status:      report.Status,
		Mode:        report.Mode,
		Project:     recordToJSON(report.Project),
		ProfileID:   report.ProfileID,
		Adapter:     report.Adapter,
		ProfileHash: report.ProfileHash,
		StageCount:  report.StageCount,
		GateCount:   report.GateCount,
		Checks:      make([]conformanceCheckJSON, 0, len(report.Checks)),
		GeneratedAt: formatTime(report.GeneratedAt),
	}
	for _, check := range report.Checks {
		out.Checks = append(out.Checks, conformanceCheckJSON{
			Key:      check.Key,
			Category: check.Category,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return out
}

func schedulingPolicyToJSON(policy project.SchedulingPolicy) schedulingPolicyJSON {
	return schedulingPolicyJSON{
		Priority:             policy.Priority,
		MaxParallelTasks:     policy.MaxParallelTasks,
		AgentRole:            policy.AgentRole,
		RequiredCapabilities: jsonStringSlice(policy.RequiredCapabilities),
		EngineProfile:        policy.EngineProfile,
	}
}

func roleReadinessToJSON(readiness project.RoleReadiness) roleReadinessJSON {
	return roleReadinessJSON{
		RequiredRole:   readiness.RequiredRole,
		Matched:        readiness.Matched,
		MatchedTypes:   jsonStringSlice(readiness.MatchedTypes),
		Status:         readiness.Status,
		BlockedReasons: jsonStringSlice(readiness.BlockedReasons),
	}
}

func engineReadinessToJSON(readiness project.EngineReadiness) engineReadinessJSON {
	return engineReadinessJSON{
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

func resourceReadinessToJSON(readiness project.ResourceReadiness) resourceReadinessJSON {
	return resourceReadinessJSON{
		MaxActiveLeases: readiness.MaxActiveLeases,
		MaxQueuedTasks:  readiness.MaxQueuedTasks,
		Status:          readiness.Status,
		BlockedReasons:  jsonStringSlice(readiness.BlockedReasons),
	}
}

func workerPoolSchedulePreviewToJSON(preview project.WorkerPoolSchedulePreview) workerPoolSchedulePreviewJSON {
	out := workerPoolSchedulePreviewJSON{
		Projects:      make([]workerPoolProjectScheduleJSON, 0, len(preview.Projects)),
		Policy:        workerPoolSchedulePolicyToJSON(preview.Policy),
		GeneratedAt:   formatTime(preview.GeneratedAt),
		Recommended:   preview.Recommended,
		Blocked:       preview.Blocked,
		QueuedTasks:   preview.QueuedTasks,
		AvailableSlot: preview.AvailableSlot,
	}
	for _, schedule := range preview.Projects {
		out.Projects = append(out.Projects, workerPoolProjectScheduleToJSON(schedule))
	}
	return out
}

func workerPoolSchedulePolicyToJSON(policy project.WorkerPoolSchedulePolicy) workerPoolSchedulePolicyJSON {
	return workerPoolSchedulePolicyJSON{
		Strategy:               policy.Strategy,
		DefaultProjectPriority: policy.DefaultProjectPriority,
		SlotStrategy:           policy.SlotStrategy,
		DryRunOnly:             policy.DryRunOnly,
	}
}

func workerPoolProjectScheduleToJSON(schedule project.WorkerPoolProjectSchedule) workerPoolProjectScheduleJSON {
	return workerPoolProjectScheduleJSON{
		Project:        recordToJSON(schedule.Project),
		Priority:       schedule.Priority,
		MaxParallel:    schedule.MaxParallel,
		AgentRole:      schedule.AgentRole,
		Role:           roleReadinessToJSON(schedule.Role),
		EngineProfile:  schedule.EngineProfile,
		Engine:         engineReadinessToJSON(schedule.Engine),
		Resources:      resourceReadinessToJSON(schedule.Resources),
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

func codexCLIAdapterPreviewToJSON(preview project.CodexCLIAdapterPreview) codexCLIAdapterPreviewJSON {
	out := codexCLIAdapterPreviewJSON{
		Project:                 recordToJSON(preview.Project),
		Status:                  preview.Status,
		Mode:                    preview.Mode,
		Engine:                  engineReadinessToJSON(preview.Engine),
		Command:                 engineCommandPreviewToJSON(preview.Command),
		Capabilities:            make([]engineCapabilityPreflightJSON, 0, len(preview.Capabilities)),
		Paths:                   make([]enginePathPreflightJSON, 0, len(preview.Paths)),
		ArtifactRedaction:       artifactRedactionPlanToJSON(preview.ArtifactRedaction),
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
		out.Capabilities = append(out.Capabilities, engineCapabilityPreflightJSON{
			Capability: capability.Capability,
			Required:   capability.Required,
			Allowed:    capability.Allowed,
			Reason:     capability.Reason,
		})
	}
	for _, path := range preview.Paths {
		out.Paths = append(out.Paths, enginePathPreflightJSON{
			Path:       path.Path,
			Capability: path.Capability,
			Effect:     path.Effect,
			Allowed:    path.Allowed,
			Reason:     path.Reason,
		})
	}
	return out
}

func engineCommandPreviewToJSON(command project.EngineCommandPreview) engineCommandPreviewJSON {
	return engineCommandPreviewJSON{
		Command:           command.Command,
		Allowed:           command.Allowed,
		Reason:            command.Reason,
		CapabilityAllowed: command.CapabilityAllowed,
		CommandAllowed:    command.CommandAllowed,
		Denied:            command.Denied,
	}
}

func artifactRedactionPlanToJSON(plan project.ArtifactRedactionPlan) artifactRedactionPlanJSON {
	return artifactRedactionPlanJSON{
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

func leaseToJSON(lease project.LeaseRecord) leaseJSON {
	out := leaseJSON{
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
		out.HeartbeatAt = formatTime(*lease.HeartbeatAt)
	}
	if lease.ReleasedAt != nil {
		out.ReleasedAt = formatTime(*lease.ReleasedAt)
	}
	return out
}

func leaseRecoverToJSON(record project.Record, leases []project.LeaseRecord) leaseRecoverJSON {
	out := leaseRecoverJSON{
		Project: recordToJSON(record),
		Leases:  make([]leaseJSON, 0, len(leases)),
	}
	for _, lease := range leases {
		out.Leases = append(out.Leases, leaseToJSON(lease))
	}
	return out
}

func workerRunOnceToJSON(result project.WorkerRunOnceResult) workerRunOnceJSON {
	out := workerRunOnceJSON{
		Project: recordToJSON(result.Project),
		Worker:  workerToJSON(result.Worker),
		Claimed: result.Claimed,
	}
	if result.Claimed {
		lease := leaseToJSON(result.Lease)
		task := runTaskToJSON(result.Task)
		attempt := runAttemptToJSON(result.Attempt)
		artifact := artifactToJSON(result.Artifact)
		out.Lease = &lease
		out.Task = &task
		out.Attempt = &attempt
		out.Artifact = &artifact
	}
	return out
}

func fixtureExecutionToJSON(result project.FixtureExecutionResult) fixtureExecutionJSON {
	return fixtureExecutionJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Worker:                        workerToJSON(result.Worker),
		Lease:                         leaseToJSON(result.Lease),
		Task:                          runTaskToJSON(result.Task),
		Attempt:                       runAttemptToJSON(result.Attempt),
		Artifact:                      artifactToJSON(result.Artifact),
		Gate:                          executionApprovalGateToJSON(result.Gate),
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

func readOnlyVerifyToJSON(result project.ReadOnlyVerifyResult) readOnlyVerifyJSON {
	return readOnlyVerifyJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Worker:                        workerToJSON(result.Worker),
		Lease:                         leaseToJSON(result.Lease),
		Task:                          runTaskToJSON(result.Task),
		Attempt:                       runAttemptToJSON(result.Attempt),
		Artifact:                      artifactToJSON(result.Artifact),
		Gate:                          executionApprovalGateToJSON(result.Gate),
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

func approvedArtifactWriteToJSON(result project.ApprovedArtifactWriteResult) approvedArtifactWriteJSON {
	return approvedArtifactWriteJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Worker:                        workerToJSON(result.Worker),
		Lease:                         leaseToJSON(result.Lease),
		Task:                          runTaskToJSON(result.Task),
		Attempt:                       runAttemptToJSON(result.Attempt),
		Artifact:                      artifactToJSON(result.Artifact),
		Gate:                          executionApprovalGateToJSON(result.Gate),
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

func fixtureProjectWriteToJSON(result project.FixtureProjectWriteResult) fixtureProjectWriteJSON {
	return fixtureProjectWriteJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Worker:                        workerToJSON(result.Worker),
		Lease:                         leaseToJSON(result.Lease),
		Task:                          runTaskToJSON(result.Task),
		CopyAttempt:                   runAttemptToJSON(result.CopyAttempt),
		VerifyAttempt:                 runAttemptToJSON(result.VerifyAttempt),
		RollbackAttempt:               runAttemptToJSON(result.RollbackAttempt),
		WriteSetArtifact:              artifactToJSON(result.WriteSetArtifact),
		PreimageArtifact:              artifactToJSON(result.PreimageArtifact),
		Artifact:                      artifactToJSON(result.Artifact),
		Gate:                          executionApprovalGateToJSON(result.Gate),
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

func managedGeneratedWriteToJSON(result project.ManagedGeneratedWriteResult) managedGeneratedWriteJSON {
	return managedGeneratedWriteJSON{
		Project:                       recordToJSON(result.Project),
		WorkflowVersion:               workflowVersionToJSON(result.Version),
		Run:                           runToJSON(result.Run),
		Worker:                        workerToJSON(result.Worker),
		Lease:                         leaseToJSON(result.Lease),
		Task:                          runTaskToJSON(result.Task),
		CopyAttempt:                   runAttemptToJSON(result.CopyAttempt),
		VerifyAttempt:                 runAttemptToJSON(result.VerifyAttempt),
		RollbackAttempt:               runAttemptToJSON(result.RollbackAttempt),
		WriteSetArtifact:              artifactToJSON(result.WriteSetArtifact),
		PreimageArtifact:              artifactToJSON(result.PreimageArtifact),
		Artifact:                      artifactToJSON(result.Artifact),
		Gate:                          executionApprovalGateToJSON(result.Gate),
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

func artifactToJSON(artifact project.ArtifactRecord) artifactJSON {
	return artifactJSON{
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

func runnerPreflightToJSON(preflight project.RunnerPreflight) runnerPreflightJSON {
	out := runnerPreflightJSON{
		Status:   preflight.Status,
		Checks:   make([]projectReadinessItemJSON, 0, len(preflight.Checks)),
		Blockers: preflight.Blockers,
	}
	for _, check := range preflight.Checks {
		out.Checks = append(out.Checks, projectReadinessItemJSON{
			Key:      check.Key,
			Status:   check.Status,
			Message:  check.Message,
			Metadata: check.Metadata,
		})
	}
	return out
}

func runToJSON(run project.RunRecord) runJSON {
	out := runJSON{
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
		out.FinishedAt = formatTime(*run.FinishedAt)
	}
	return out
}

func runTaskToJSON(task project.RunTaskRecord) runTaskJSON {
	return runTaskJSON{
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

func runAttemptToJSON(attempt project.RunAttemptRecord) runAttemptJSON {
	out := runAttemptJSON{
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
		out.FinishedAt = formatTime(*attempt.FinishedAt)
	}
	return out
}

func workflowVersionToJSON(version project.WorkflowVersion) workflowVersionJSON {
	out := workflowVersionJSON{
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
		out.ImportedAt = formatTime(*version.ImportedAt)
	}
	return out
}

func workflowItemToJSON(item project.WorkflowItem) workflowItemJSON {
	out := workflowItemJSON{
		ID:                item.ID,
		WorkflowVersionID: item.WorkflowVersionID,
		Stage:             item.Stage,
		ItemType:          item.ItemType,
		ExternalKey:       item.ExternalKey,
		Title:             item.Title,
		Status:            item.Status,
		SourcePath:        item.SourcePath,
		SourceHash:        item.SourceHash,
		Metadata:          item.Metadata,
		CreatedAt:         formatTime(item.CreatedAt),
		UpdatedAt:         formatTime(item.UpdatedAt),
	}
	if item.ImportedAt != nil {
		out.ImportedAt = formatTime(*item.ImportedAt)
	}
	return out
}

func workflowItemLinkToJSON(link project.WorkflowItemLink) workflowItemLinkJSON {
	return workflowItemLinkJSON{
		ID:                link.ID,
		WorkflowVersionID: link.WorkflowVersionID,
		FromItemID:        link.FromItemID,
		ToItemID:          link.ToItemID,
		RelationType:      link.RelationType,
		Metadata:          link.Metadata,
		CreatedAt:         formatTime(link.CreatedAt),
	}
}

func workflowProfileToJSON(loaded workflowprofile.LoadedProfile) workflowProfileJSON {
	return workflowProfileJSON{
		ProfileID:       loaded.Profile.ProfileID,
		ProfileVersion:  loaded.Profile.ProfileVersion,
		DisplayName:     loaded.Profile.DisplayName,
		Description:     loaded.Profile.Description,
		Path:            loaded.Path,
		SHA256:          loaded.SHA256,
		StageCount:      len(loaded.Profile.Stages),
		GateCount:       len(loaded.Profile.Gates),
		TransitionCount: len(loaded.Profile.Transitions),
		Warnings:        jsonStringSlice(loaded.Warnings),
		Profile:         loaded.Profile,
	}
}

func workflowProfileCheckToJSON(loaded workflowprofile.LoadedProfile) workflowProfileCheckJSON {
	return workflowProfileCheckJSON{
		ProfileID:       loaded.Profile.ProfileID,
		ProfileVersion:  loaded.Profile.ProfileVersion,
		Path:            loaded.Path,
		SHA256:          loaded.SHA256,
		Status:          "pass",
		StageCount:      len(loaded.Profile.Stages),
		GateCount:       len(loaded.Profile.Gates),
		TransitionCount: len(loaded.Profile.Transitions),
		Warnings:        jsonStringSlice(loaded.Warnings),
	}
}

func workflowProfileSummaryToJSON(loaded workflowprofile.LoadedProfile) workflowProfileSummaryJSON {
	return workflowProfileSummaryJSON{
		ProfileID:       loaded.Profile.ProfileID,
		ProfileVersion:  loaded.Profile.ProfileVersion,
		DisplayName:     loaded.Profile.DisplayName,
		Description:     loaded.Profile.Description,
		Path:            loaded.Path,
		SHA256:          loaded.SHA256,
		StageCount:      len(loaded.Profile.Stages),
		GateCount:       len(loaded.Profile.Gates),
		TransitionCount: len(loaded.Profile.Transitions),
		Warnings:        jsonStringSlice(loaded.Warnings),
	}
}

func workflowProfileListToJSON(profiles []workflowprofile.LoadedProfile) workflowProfileListJSON {
	out := workflowProfileListJSON{
		Profiles: make([]workflowProfileSummaryJSON, 0, len(profiles)),
	}
	for _, loaded := range profiles {
		out.Profiles = append(out.Profiles, workflowProfileSummaryToJSON(loaded))
	}
	return out
}

func recordToJSON(record project.Record) projectRecordJSON {
	return projectRecordJSON{
		Key:             record.Key,
		Name:            record.Name,
		Kind:            record.Kind,
		Adapter:         record.Adapter,
		WorkflowProfile: record.WorkflowProfile,
		DefaultBranch:   record.DefaultBranch,
		Root:            record.RootPath,
	}
}

func readinessToJSON(readiness project.ProjectReadiness) projectReadinessJSON {
	out := projectReadinessJSON{
		Project: recordToJSON(readiness.Project),
		Status:  readiness.Status,
		Items:   make([]projectReadinessItemJSON, 0, len(readiness.Items)),
		Summary: summaryToJSON(readiness.Summary),
	}
	for _, item := range readiness.Items {
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func generatedWriteReadinessToJSON(readiness project.GeneratedWriteReadiness) generatedWriteReadinessJSON {
	out := generatedWriteReadinessJSON{
		Project:                       recordToJSON(readiness.Project),
		Status:                        readiness.Status,
		Mode:                          readiness.Mode,
		Items:                         make([]projectReadinessItemJSON, 0, len(readiness.Items)),
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
		out.Items = append(out.Items, projectReadinessItemJSON{
			Key:      item.Key,
			Status:   item.Status,
			Message:  item.Message,
			Metadata: item.Metadata,
		})
	}
	return out
}

func generatedWriteApplyBetaGateToJSON(gate project.GeneratedWriteApplyBetaGate) generatedWriteApplyBetaGateJSON {
	out := generatedWriteApplyBetaGateJSON{
		Project:                       recordToJSON(gate.Project),
		Status:                        gate.Status,
		Mode:                          gate.Mode,
		Readiness:                     generatedWriteReadinessToJSON(gate.Readiness),
		Items:                         make([]generatedWriteApplyBetaGateItemJSON, 0, len(gate.Items)),
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
		out.Items = append(out.Items, generatedWriteApplyBetaGateItemJSON{
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
	return out
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (c command) printDoctorReport(report doctor.Report) {
	fmt.Fprintf(c.stdout, "project doctor: %s profile=%s status=%s\n", report.Project, report.Profile, report.OverallStatus())
	for _, check := range report.Checks {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", check.Status, check.Name, check.Message)
		for _, detail := range check.Details {
			fmt.Fprintf(c.stdout, "  %s=%s\n", detail.Key, detail.Value)
		}
	}
}

func (c command) printWorkflowVersionCreate(result project.CreateWorkflowVersionResult) {
	state := "existing"
	if result.Created {
		state = "created"
	}
	fmt.Fprintf(c.stdout, "%s workflow version %s/%s status=%s import_mode=%s initial_stage=%s idempotency_key=%s\n",
		state,
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Version.LifecycleStatus,
		result.Version.ImportMode,
		result.InitialItem.Stage,
		result.IdempotencyKey,
	)
	if len(result.StageItems) > 0 {
		fmt.Fprintf(c.stdout, "stage_skeleton.created: %d\n", len(result.StageItems))
	}
}

func (c command) printWorkflowVersionList(record project.Record, versions []project.WorkflowVersion) {
	fmt.Fprintf(c.stdout, "workflow versions: %s count=%d\n", record.Key, len(versions))
	for _, version := range versions {
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\timmutable=%t\n",
			version.DisplayLabel,
			version.LifecycleStatus,
			version.ImportMode,
			version.Immutable,
		)
	}
}

func (c command) printWorkflowVersion(record project.Record, version project.WorkflowVersion) {
	fmt.Fprintf(c.stdout, "workflow version: %s/%s\n", record.Key, version.DisplayLabel)
	fmt.Fprintf(c.stdout, "id: %d\n", version.ID)
	fmt.Fprintf(c.stdout, "version_kind: %s\n", version.VersionKind)
	fmt.Fprintf(c.stdout, "lifecycle_status: %s\n", version.LifecycleStatus)
	fmt.Fprintf(c.stdout, "import_mode: %s\n", version.ImportMode)
	fmt.Fprintf(c.stdout, "immutable: %t\n", version.Immutable)
	fmt.Fprintf(c.stdout, "created_at: %s\n", formatTime(version.CreatedAt))
	fmt.Fprintf(c.stdout, "updated_at: %s\n", formatTime(version.UpdatedAt))
}

func (c command) printWorkflowProfile(loaded workflowprofile.LoadedProfile) {
	fmt.Fprintf(c.stdout, "workflow profile: %s version=%d\n", loaded.Profile.ProfileID, loaded.Profile.ProfileVersion)
	fmt.Fprintf(c.stdout, "path: %s\n", loaded.Path)
	fmt.Fprintf(c.stdout, "sha256: %s\n", loaded.SHA256)
	fmt.Fprintf(c.stdout, "stages: %d\n", len(loaded.Profile.Stages))
	fmt.Fprintf(c.stdout, "gates: %d\n", len(loaded.Profile.Gates))
	fmt.Fprintf(c.stdout, "transitions: %d\n", len(loaded.Profile.Transitions))
	if len(loaded.Warnings) > 0 {
		fmt.Fprintf(c.stdout, "warnings: %s\n", strings.Join(loaded.Warnings, "; "))
	}
	for _, stage := range loaded.Profile.Stages {
		fmt.Fprintf(c.stdout, "stage: %s gates=%s\n", stage.Name, strings.Join(stage.GateChecks, ","))
	}
}

func (c command) printWorkflowProfileCheck(loaded workflowprofile.LoadedProfile) {
	fmt.Fprintf(c.stdout, "workflow profile check: %s pass\n", loaded.Profile.ProfileID)
	fmt.Fprintf(c.stdout, "sha256: %s\n", loaded.SHA256)
	fmt.Fprintf(c.stdout, "stages: %d gates=%d transitions=%d\n", len(loaded.Profile.Stages), len(loaded.Profile.Gates), len(loaded.Profile.Transitions))
	if len(loaded.Warnings) > 0 {
		fmt.Fprintf(c.stdout, "warnings: %s\n", strings.Join(loaded.Warnings, "; "))
	}
}

func (c command) printWorkflowProfileList(profiles []workflowprofile.LoadedProfile) {
	fmt.Fprintf(c.stdout, "workflow profiles: count=%d\n", len(profiles))
	for _, loaded := range profiles {
		fmt.Fprintf(c.stdout, "%s\tversion=%d\tstages=%d\tgates=%d\ttransitions=%d\tsha256=%s\n",
			loaded.Profile.ProfileID,
			loaded.Profile.ProfileVersion,
			len(loaded.Profile.Stages),
			len(loaded.Profile.Gates),
			len(loaded.Profile.Transitions),
			loaded.SHA256,
		)
		if len(loaded.Warnings) > 0 {
			fmt.Fprintf(c.stdout, "  warnings: %s\n", strings.Join(loaded.Warnings, "; "))
		}
	}
}

func (c command) printWorkflowVersionStages(record project.Record, version project.WorkflowVersion, items []project.WorkflowItem) {
	fmt.Fprintf(c.stdout, "workflow version stages: %s/%s count=%d\n", record.Key, version.DisplayLabel, len(items))
	for _, item := range items {
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n",
			item.Stage,
			item.ItemType,
			item.Status,
			item.ExternalKey,
		)
	}
}

func (c command) printEnsureStageSkeleton(result project.EnsureStageSkeletonResult) {
	fmt.Fprintf(c.stdout, "stage skeleton: %s/%s created=%d\n", result.Project.Key, result.Version.DisplayLabel, result.Created)
	if len(result.Links) > 0 {
		fmt.Fprintf(c.stdout, "stage_skeleton.links: %d\n", len(result.Links))
	}
	for _, item := range result.Items {
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\n", item.Stage, item.ItemType, item.Status)
	}
}

func (c command) printMarkWorkflowItemReady(result project.MarkWorkflowItemReadyResult) {
	fmt.Fprintf(c.stdout, "workflow item ready: %s/%s %s/%s status=%s artifact=%d\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Item.Stage,
		result.Item.ItemType,
		result.Item.Status,
		result.Artifact.ID,
	)
	fmt.Fprintf(c.stdout, "source_hash: %s\n", result.Item.SourceHash)
}

func (c command) printGateResult(result project.GateResult) {
	fmt.Fprintf(c.stdout, "workflow gate: %s scope=%s/%s status=%s\n", result.GateName, result.ScopeType, result.ScopeID, result.Status)
	if len(result.Failures) > 0 {
		fmt.Fprintf(c.stdout, "failures: %s\n", result.Failures)
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintf(c.stdout, "warnings: %s\n", result.Warnings)
	}
}

func (c command) printGateResults(record project.Record, version project.WorkflowVersion, results []project.GateResult) {
	fmt.Fprintf(c.stdout, "workflow gate results: %s/%s count=%d\n", record.Key, version.DisplayLabel, len(results))
	for _, result := range results {
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n",
			formatTime(result.CheckedAt),
			result.GateName,
			result.Status,
			result.ScopeID,
		)
	}
}

func (c command) printTransitionPreview(preview project.WorkflowTransitionPreview) {
	fmt.Fprintf(c.stdout, "workflow transition preview: %s -> %s status=%s required_gate=%s\n",
		preview.FromStage,
		preview.ToStage,
		preview.Status,
		preview.RequiredGateName,
	)
	if preview.GateResultID != 0 {
		fmt.Fprintf(c.stdout, "gate_result_id: %d\n", preview.GateResultID)
	}
	if len(preview.Blockers) > 0 {
		fmt.Fprintf(c.stdout, "blockers: %s\n", preview.Blockers)
	}
	if len(preview.Warnings) > 0 {
		fmt.Fprintf(c.stdout, "warnings: %s\n", preview.Warnings)
	}
}

func (c command) printTransitionPreviews(record project.Record, version project.WorkflowVersion, previews []project.WorkflowTransitionPreview) {
	fmt.Fprintf(c.stdout, "workflow transition previews: %s/%s count=%d\n", record.Key, version.DisplayLabel, len(previews))
	for _, preview := range previews {
		fmt.Fprintf(c.stdout, "%s\t%s->%s\t%s\tgate=%s\n",
			formatTime(preview.CreatedAt),
			preview.FromStage,
			preview.ToStage,
			preview.Status,
			preview.RequiredGateName,
		)
	}
}

func (c command) printApprovalRecord(approval project.ApprovalRecord) {
	fmt.Fprintf(c.stdout, "workflow approval: %s scope=%s/%s actor=%s risk=%s\n",
		approval.Decision,
		approval.ScopeType,
		approval.ScopeID,
		approval.Actor,
		approval.RiskLevel,
	)
	if approval.TransitionPreviewID != 0 {
		fmt.Fprintf(c.stdout, "transition_preview_id: %d\n", approval.TransitionPreviewID)
	}
	fmt.Fprintf(c.stdout, "reason: %s\n", approval.Reason)
}

func (c command) printApprovalRecords(record project.Record, version project.WorkflowVersion, approvals []project.ApprovalRecord) {
	fmt.Fprintf(c.stdout, "workflow approvals: %s/%s count=%d\n", record.Key, version.DisplayLabel, len(approvals))
	for _, approval := range approvals {
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n",
			formatTime(approval.CreatedAt),
			approval.Decision,
			approval.Actor,
			approval.RiskLevel,
		)
	}
}

func (c command) printRunnerPreview(result project.RunnerPreviewResult) {
	fmt.Fprintf(c.stdout, "runner preview: %s/%s run=%d status=%s dry_run=%t created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Run.Status,
		result.Run.DryRun,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "preflight.status: %s\n", result.Preflight.Status)
	for _, check := range result.Preflight.Checks {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", check.Status, check.Key, check.Message)
	}
	fmt.Fprintf(c.stdout, "tasks.count: %d\n", len(result.Tasks))
	fmt.Fprintf(c.stdout, "attempts.count: %d\n", len(result.Attempts))
	fmt.Fprintf(c.stdout, "artifacts.count: %d\n", len(result.Artifacts))
	for _, attempt := range result.Attempts {
		fmt.Fprintf(c.stdout, "attempt.%s: %s dry_run=%t\n", attempt.AttemptKind, attempt.Status, attempt.DryRun)
	}
	for _, artifact := range result.Artifacts {
		fmt.Fprintf(c.stdout, "artifact.%s: %s\n", artifact.ArtifactType, artifact.URI)
	}
}

func (c command) printFixtureExecutionQueue(result project.FixtureExecutionQueueResult) {
	fmt.Fprintf(c.stdout, "fixture execution queued: %s/%s run=%d task=%d created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
}

func (c command) printReadOnlyVerifyQueue(result project.ReadOnlyVerifyQueueResult) {
	fmt.Fprintf(c.stdout, "read-only verify queued: %s/%s run=%d task=%d target=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.TargetPath,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_read_allowed: %t\n", result.ProjectReadAllowed)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
}

func (c command) printApprovedArtifactWriteQueue(result project.ApprovedArtifactWriteQueueResult) {
	fmt.Fprintf(c.stdout, "approved artifact write queued: %s/%s run=%d task=%d artifact_label=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.ArtifactLabel,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", result.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
}

func (c command) printFixtureProjectWriteQueue(result project.FixtureProjectWriteQueueResult) {
	fmt.Fprintf(c.stdout, "fixture project write queued: %s/%s run=%d task=%d target=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.TargetPath,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "write_set_artifact=%d expected_before_sha256=%s expected_before_size=%d after_sha256=%s after_size=%d\n",
		result.WriteSetArtifact.ID,
		result.ExpectedBeforeSHA256,
		result.ExpectedBeforeSize,
		result.AfterSHA256,
		result.AfterSize,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", result.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
}

func (c command) printManagedGeneratedWriteQueue(result project.ManagedGeneratedWriteQueueResult) {
	fmt.Fprintf(c.stdout, "managed generated write queued: %s/%s run=%d task=%d target=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.TargetPath,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "write_set_artifact=%d expected_before_sha256=%s expected_before_size=%d after_sha256=%s after_size=%d\n",
		result.WriteSetArtifact.ID,
		result.ExpectedBeforeSHA256,
		result.ExpectedBeforeSize,
		result.AfterSHA256,
		result.AfterSize,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "generated_only: %t\n", result.GeneratedOnly)
	fmt.Fprintf(c.stdout, "generated_only_apply_open: %t\n", result.GeneratedOnlyApplyOpen)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", result.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", result.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", result.NetworkUsed)
}

func (c command) printRunControl(result project.RunControlResult) {
	fmt.Fprintf(c.stdout, "run control: %s run=%d %s->%s decision=%s created=%t\n",
		result.Project.Key,
		result.Run.ID,
		result.PreviousStatus,
		result.Status,
		result.Decision,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "message: %s\n", result.Message)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_matrix_write_attempted: %t\n", result.AreaMatrixWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printExecutionApprovalGate(gate project.ExecutionApprovalGate) {
	fmt.Fprintf(c.stdout, "execution approval gate: %s/%s run=%d status=%s mode=%s\n",
		gate.Project.Key,
		gate.Version.DisplayLabel,
		gate.Run.ID,
		gate.Status,
		gate.Mode,
	)
	fmt.Fprintf(c.stdout, "required_capabilities: %s\n", strings.Join(gate.RequiredCapabilities, ","))
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", gate.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", gate.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", gate.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", gate.CommandsRun)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", gate.TaskClaimed)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", gate.WorkerStarted)
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
	for _, blocker := range gate.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
	for _, warning := range gate.Warnings {
		fmt.Fprintf(c.stdout, "warning: %s\n", warning)
	}
}

func (c command) printExecutionPlanPreview(preview project.ExecutionPlanPreview) {
	fmt.Fprintf(c.stdout, "execution plan preview: %s/%s run=%d status=%s mode=%s\n",
		preview.Project.Key,
		preview.Version.DisplayLabel,
		preview.Run.ID,
		preview.Status,
		preview.Mode,
	)
	fmt.Fprintf(c.stdout, "gate_status: %s\n", preview.Gate.Status)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", preview.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", preview.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", preview.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", preview.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", preview.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", preview.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", preview.CommandsRun)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", preview.TaskClaimed)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", preview.WorkerStarted)
	for _, step := range preview.Steps {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", step.Status, step.Key, step.Message)
		if len(step.RequiredCapabilities) > 0 {
			fmt.Fprintf(c.stdout, "  required_capabilities: %s\n", strings.Join(step.RequiredCapabilities, ","))
		}
		for _, blocker := range step.Blockers {
			fmt.Fprintf(c.stdout, "  blocker: %s\n", blocker)
		}
	}
	for _, blocker := range preview.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printProjectWriteDesignGate(gate project.ProjectWriteDesignGate) {
	fmt.Fprintf(c.stdout, "project write design gate: %s/%s run=%d status=%s mode=%s\n",
		gate.Project.Key,
		gate.Version.DisplayLabel,
		gate.Run.ID,
		gate.Status,
		gate.Mode,
	)
	fmt.Fprintf(c.stdout, "gate_status: %s\n", gate.Gate.Status)
	fmt.Fprintf(c.stdout, "project_write_apply_open: %t\n", gate.ProjectWriteApplyOpen)
	fmt.Fprintf(c.stdout, "required_capabilities: %s\n", strings.Join(gate.RequiredCapabilities, ","))
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", gate.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", gate.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", gate.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", gate.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", gate.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", gate.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", gate.CommandsRun)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", gate.TaskClaimed)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", gate.WorkerStarted)
	fmt.Fprintf(c.stdout, "write_set_fields: %s\n", strings.Join(gate.WriteSetFields, ","))
	fmt.Fprintf(c.stdout, "unsupported_operations: %s\n", strings.Join(gate.UnsupportedOperations, ","))
	fmt.Fprintf(c.stdout, "apply_sequence: %s\n", strings.Join(gate.ApplySequence, " -> "))
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
	for _, blocker := range gate.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printManagedGeneratedWriteGate(gate project.ManagedGeneratedWriteGate) {
	fmt.Fprintf(c.stdout, "managed generated write gate: %s/%s run=%d status=%s mode=%s\n",
		gate.Project.Key,
		gate.Version.DisplayLabel,
		gate.Run.ID,
		gate.Status,
		gate.Mode,
	)
	fmt.Fprintf(c.stdout, "gate_status: %s\n", gate.Gate.Status)
	fmt.Fprintf(c.stdout, "generated_only_write_ready: %t\n", gate.GeneratedOnlyWriteReady)
	fmt.Fprintf(c.stdout, "generated_only_apply_open: %t\n", gate.GeneratedOnlyApplyOpen)
	fmt.Fprintf(c.stdout, "required_capabilities: %s\n", strings.Join(gate.RequiredCapabilities, ","))
	fmt.Fprintf(c.stdout, "allowed_generated_prefixes: %s\n", strings.Join(gate.AllowedGeneratedPrefixes, ","))
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", gate.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", gate.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", gate.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", gate.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", gate.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", gate.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", gate.CommandsRun)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", gate.TaskClaimed)
	fmt.Fprintf(c.stdout, "worker_started: %t\n", gate.WorkerStarted)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", gate.LeaseCreated)
	fmt.Fprintf(c.stdout, "required_write_set_fields: %s\n", strings.Join(gate.RequiredWriteSetFields, ","))
	fmt.Fprintf(c.stdout, "unsupported_operations: %s\n", strings.Join(gate.UnsupportedOperations, ","))
	fmt.Fprintf(c.stdout, "apply_sequence: %s\n", strings.Join(gate.ApplySequence, " -> "))
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n", item.Status, item.Key, item.Message)
	}
	for _, blocker := range gate.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printWorker(worker project.WorkerRecord) {
	fmt.Fprintf(c.stdout, "worker: %s id=%d type=%s status=%s project_id=%d\n",
		worker.WorkerKey,
		worker.ID,
		worker.WorkerType,
		worker.Status,
		worker.ProjectID,
	)
	fmt.Fprintf(c.stdout, "heartbeat_interval_seconds: %d\n", worker.HeartbeatIntervalSeconds)
	fmt.Fprintf(c.stdout, "lease_timeout_seconds: %d\n", worker.LeaseTimeoutSeconds)
	if worker.LastHeartbeatAt != nil {
		fmt.Fprintf(c.stdout, "last_heartbeat_at: %s\n", formatTime(*worker.LastHeartbeatAt))
	}
	if len(worker.Capabilities) > 0 {
		fmt.Fprintf(c.stdout, "capabilities: %s\n", strings.Join(worker.Capabilities, ","))
	}
}

func (c command) printWorkers(record project.Record, workers []project.WorkerRecord) {
	fmt.Fprintf(c.stdout, "workers: %s count=%d\n", record.Key, len(workers))
	for _, worker := range workers {
		heartbeat := ""
		if worker.LastHeartbeatAt != nil {
			heartbeat = formatTime(*worker.LastHeartbeatAt)
		}
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n",
			worker.WorkerKey,
			worker.WorkerType,
			worker.Status,
			heartbeat,
		)
	}
}

func (c command) printLocalServiceStatus(status project.LocalServiceStatus) {
	fmt.Fprintf(c.stdout, "service status: %s mode=%s\n", status.Status, status.Mode)
	fmt.Fprintf(c.stdout, "api: %s %s\n", status.API.Status, status.API.Message)
	fmt.Fprintf(c.stdout, "database: %s %s\n", status.Database.Status, status.Database.Message)
	fmt.Fprintf(c.stdout, "worker_pool: %s projects=%d workers=%d online=%d queued=%d recovery=%d\n",
		status.WorkerPool.Status,
		status.WorkerPool.TotalProjects,
		status.WorkerPool.TotalWorkers,
		status.WorkerPool.TotalOnlineWorkers,
		status.WorkerPool.TotalQueuedTasks,
		status.WorkerPool.TotalNeedsRecovery,
	)
	fmt.Fprintf(c.stdout, "dashboard: %s api=%s url=%s\n",
		status.Dashboard.Status,
		status.Dashboard.APIURL,
		status.Dashboard.URL,
	)
	if len(status.Capabilities) > 0 {
		fmt.Fprintf(c.stdout, "capabilities: %s\n", strings.Join(status.Capabilities, ","))
	}
	if len(status.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(status.ForbiddenActions, ","))
	}
}

func (c command) printDesktopServiceControlGate(gate project.DesktopServiceControlGate) {
	fmt.Fprintf(c.stdout, "desktop service-control gate: status=%s mode=%s actions=%d\n",
		gate.Status,
		gate.Mode,
		len(gate.Actions),
	)
	fmt.Fprintf(c.stdout, "safety: process_control=%t command_created=%t audit_written=%t worker_scheduled=%t workflow_execution=%t\n",
		gate.ProcessControlAttempted,
		gate.CommandCreated,
		gate.AuditEventWritten,
		gate.WorkerScheduled,
		gate.WorkflowExecutionStarted,
	)
	for _, action := range gate.Actions {
		fmt.Fprintf(c.stdout, "%s: %s ui=%s risk=%s\n",
			action.Key,
			action.Status,
			action.DefaultUIState,
			action.RiskLevel,
		)
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printDesktopNotificationGate(gate project.DesktopNotificationGate) {
	fmt.Fprintf(c.stdout, "desktop notification gate: status=%s mode=%s actions=%d\n",
		gate.Status,
		gate.Mode,
		len(gate.Actions),
	)
	fmt.Fprintf(c.stdout, "safety: event_stream_opened=%t notification_requested=%t command_created=%t audit_written=%t worker_scheduled=%t workflow_execution=%t\n",
		gate.EventStreamOpened,
		gate.NotificationRequested,
		gate.CommandCreated,
		gate.AuditEventWritten,
		gate.WorkerScheduled,
		gate.WorkflowExecutionStarted,
	)
	for _, action := range gate.Actions {
		fmt.Fprintf(c.stdout, "%s: %s ui=%s risk=%s\n",
			action.Key,
			action.Status,
			action.DefaultUIState,
			action.RiskLevel,
		)
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printDesktopTrayMenuGate(gate project.DesktopTrayMenuGate) {
	fmt.Fprintf(c.stdout, "desktop tray-menu gate: status=%s mode=%s actions=%d\n",
		gate.Status,
		gate.Mode,
		len(gate.Actions),
	)
	fmt.Fprintf(c.stdout, "safety: tray_menu_created=%t os_integration=%t service_control=%t notification_requested=%t command_created=%t audit_written=%t worker_scheduled=%t workflow_execution=%t\n",
		gate.TrayMenuCreated,
		gate.OSIntegrationRequested,
		gate.ServiceControlAttempted,
		gate.NotificationRequested,
		gate.CommandCreated,
		gate.AuditEventWritten,
		gate.WorkerScheduled,
		gate.WorkflowExecutionStarted,
	)
	for _, action := range gate.Actions {
		fmt.Fprintf(c.stdout, "%s: %s ui=%s risk=%s\n",
			action.Key,
			action.Status,
			action.DefaultUIState,
			action.RiskLevel,
		)
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printSecurityBoundaryReadiness(readiness project.SecurityBoundaryReadiness) {
	fmt.Fprintf(c.stdout, "security boundary readiness: %s mode=%s items=%d\n",
		readiness.Status,
		readiness.Mode,
		len(readiness.Items),
	)
	if len(readiness.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(readiness.ForbiddenActions, ","))
	}
	if len(readiness.Capabilities) > 0 {
		fmt.Fprintf(c.stdout, "capabilities: %s\n", strings.Join(readiness.Capabilities, ","))
	}
}

func (c command) printCompletionAuditSnapshotReadiness(readiness project.CompletionAuditSnapshotReadiness) {
	fmt.Fprintf(c.stdout, "completion audit snapshot readiness: %s status=%s required_class=%s has_snapshot=%t\n",
		readiness.Project.Key,
		readiness.Status,
		readiness.RequiredClass,
		readiness.HasSnapshot,
	)
	c.printCompletionReal100Guardrail(readiness.Real100Guardrail)
	if readiness.HasSnapshot {
		fmt.Fprintf(c.stdout, "latest: rc=%s evidence_class=%s evidence_uri=%s event_id=%d\n",
			readiness.Latest.ReleaseCandidateLabel,
			readiness.Latest.EvidenceClass,
			readiness.Latest.EvidenceURI,
			readiness.Latest.EventID,
		)
	}
	for _, item := range readiness.Items {
		if item.Status != "ready" {
			fmt.Fprintf(c.stdout, "%s: %s\n", item.Key, item.Message)
		}
	}
}

func (c command) printCompletionAudit(audit project.CompletionAudit) {
	fmt.Fprintf(c.stdout, "completion audit: %s scope=%s mode=%s items=%d\n",
		audit.Status,
		audit.Scope,
		audit.Mode,
		len(audit.Items),
	)
	c.printCompletionReal100Guardrail(audit.Real100Guardrail)
	fmt.Fprintf(c.stdout, "release_final_gate: %s\n", audit.ReleaseFinalGateStatus)
	fmt.Fprintf(c.stdout, "areamatrix_dogfood: %s\n", audit.AreaMatrixDogfoodStatus)
	fmt.Fprintf(c.stdout, "task_matrix: %s\n", audit.TaskMatrixStatus)
	fmt.Fprintf(c.stdout, "implementation_gap: %s\n", audit.ImplementationGapStatus)
	fmt.Fprintf(c.stdout, "protected_path_proof: %s\n", audit.ProtectedPathProofStatus)
	for _, item := range audit.Items {
		if item.Status == "blocked" || item.Status == "incomplete" {
			fmt.Fprintf(c.stdout, "%s: %s next=%s\n", item.Key, item.Status, item.NextCommand)
		}
	}
}

func (c command) printCompletionReal100Guardrail(guardrail project.Real100Guardrail) {
	guardrail = completionAuditReal100Guardrail(guardrail)
	fmt.Fprintf(c.stdout, "real_100: status=%s claim_scope=%s not_real_100=%t evidence_only=%t status_alone_is_not_completion=%t release_candidate_decision=%s scope=%s blockers=%s\n",
		guardrail.Real100Status,
		guardrail.ClaimScope,
		guardrail.NotReal100,
		guardrail.EvidenceOnly,
		guardrail.StatusAloneIsNotCompletion,
		guardrail.ReleaseCandidateDecision,
		guardrail.ReadinessScope,
		strings.Join(guardrail.Real100Blockers, ","),
	)
	c.printReal100Breakdown(guardrail.Real100Breakdown)
}

func (c command) printReleaseReal100Guardrail(guardrail project.Real100Guardrail) {
	guardrail = releasePreviewReal100Guardrail(guardrail)
	fmt.Fprintf(c.stdout, "real_100: status=%s claim_scope=%s not_real_100=%t evidence_only=%t status_alone_is_not_completion=%t release_candidate_decision=%s scope=%s blockers=%s\n",
		guardrail.Real100Status,
		guardrail.ClaimScope,
		guardrail.NotReal100,
		guardrail.EvidenceOnly,
		guardrail.StatusAloneIsNotCompletion,
		guardrail.ReleaseCandidateDecision,
		guardrail.ReadinessScope,
		strings.Join(guardrail.Real100Blockers, ","),
	)
	c.printReal100Breakdown(guardrail.Real100Breakdown)
}

func (c command) printReal100Breakdown(breakdown project.Real100Breakdown) {
	fmt.Fprintf(c.stdout, "real_100_breakdown: exact_authorization=%d real_areamatrix_write=%d areaflow_only=%d completed=%d\n",
		len(breakdown.NeedsExactAuthorization),
		len(breakdown.NeedsRealAreaMatrixWrite),
		len(breakdown.AreaFlowOnlyCanContinue),
		len(breakdown.CompletedEvidence),
	)
}

func (c command) printSupportBundlePreview(preview project.SupportBundlePreview) {
	fmt.Fprintf(c.stdout, "support bundle preview: status=%s mode=%s scope=%s projects=%d\n",
		preview.Status,
		preview.Mode,
		preview.Scope,
		len(preview.Projects),
	)
	fmt.Fprintf(c.stdout, "included_metadata: %d excluded_sensitive_content: %d path_references: %d hashes: %d\n",
		len(preview.IncludedMetadata),
		len(preview.ExcludedSensitiveContent),
		len(preview.PathReferences),
		len(preview.Hashes),
	)
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printMigrationLedgerReadiness(readiness project.MigrationLedgerReadiness) {
	fmt.Fprintf(c.stdout, "migration ledger readiness: status=%s mode=%s applied=%d pending=%d entries=%d\n",
		readiness.Status,
		readiness.Mode,
		readiness.AppliedCount,
		readiness.PendingCount,
		len(readiness.Entries),
	)
	fmt.Fprintf(c.stdout, "schema_migrations_table_present: %t\n", readiness.SchemaMigrationsTablePresent)
	fmt.Fprintf(c.stdout, "full_ledger_table_present: %t\n", readiness.FullLedgerTablePresent)
	fmt.Fprintf(c.stdout, "preflight_apply_verify_remediation_ready: %t\n", readiness.PreflightApplyVerifyRemediationReady)
	if len(readiness.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(readiness.ForbiddenActions, ","))
	}
}

func (c command) printOperationsReadiness(readiness project.OperationsReadiness) {
	fmt.Fprintf(c.stdout, "operations readiness: status=%s mode=%s items=%d\n",
		readiness.Status,
		readiness.Mode,
		len(readiness.Items),
	)
	fmt.Fprintf(c.stdout, "service_status: %s\n", readiness.ServiceStatus.Status)
	fmt.Fprintf(c.stdout, "support_bundle: %s export=%s\n", readiness.SupportBundle.Status, readiness.SupportExportStatus)
	fmt.Fprintf(c.stdout, "migration_ledger: %s applied=%d pending=%d\n",
		readiness.MigrationLedger.Status,
		readiness.MigrationLedger.AppliedCount,
		readiness.MigrationLedger.PendingCount,
	)
	fmt.Fprintf(c.stdout, "telemetry_default: %s managed_ops: %s\n", readiness.TelemetryDefault, readiness.ManagedOpsStatus)
	for _, item := range readiness.Items {
		if item.Status == "blocked" || item.Status == "needs_attention" || item.Status == "deferred" {
			fmt.Fprintf(c.stdout, "%s: %s next=%s\n", item.Key, item.Status, item.NextCommand)
		}
	}
	if len(readiness.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(readiness.ForbiddenActions, ","))
	}
}

func (c command) printBackupManifest(manifest project.BackupManifest) {
	fmt.Fprintf(c.stdout, "backup manifest: status=%s mode=%s scope=%s project=%s schema=%d hash=%s\n",
		manifest.Status,
		manifest.Mode,
		manifest.Scope,
		manifest.ProjectKey,
		manifest.SchemaVersion,
		manifest.ManifestHash,
	)
	fmt.Fprintf(c.stdout, "tables.count: %d\n", len(manifest.TableCounts))
	fmt.Fprintf(c.stdout, "projects.count: %d\n", len(manifest.Projects))
	for _, projectManifest := range manifest.Projects {
		fmt.Fprintf(c.stdout, "project.%s: artifacts=%d versions=%d residuals=%d\n",
			projectManifest.Project.Key,
			projectManifest.ArtifactCount,
			projectManifest.Inventory.Versions,
			projectManifest.Inventory.Residuals,
		)
	}
	if len(manifest.Capabilities) > 0 {
		fmt.Fprintf(c.stdout, "capabilities: %s\n", strings.Join(manifest.Capabilities, ","))
	}
	if len(manifest.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(manifest.ForbiddenActions, ","))
	}
}

func (c command) printRestorePlan(plan project.RestorePlan) {
	fmt.Fprintf(c.stdout, "restore plan: status=%s mode=%s scope=%s project=%s schema=%d hash=%s projects=%d\n",
		plan.Status,
		plan.Mode,
		plan.Scope,
		plan.ProjectKey,
		plan.SchemaVersion,
		plan.ManifestHash,
		len(plan.Projects),
	)
	for _, item := range plan.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Message,
		)
	}
	if len(plan.Capabilities) > 0 {
		fmt.Fprintf(c.stdout, "capabilities: %s\n", strings.Join(plan.Capabilities, ","))
	}
	if len(plan.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(plan.ForbiddenActions, ","))
	}
}

func (c command) printReleaseReadiness(readiness project.ReleaseReadiness) {
	fmt.Fprintf(c.stdout, "release readiness: status=%s mode=%s projects=%d items=%d\n",
		readiness.Status,
		readiness.Mode,
		len(readiness.Projects),
		len(readiness.Items),
	)
	c.printReleaseReal100Guardrail(readiness.Real100Guardrail)
	fmt.Fprintf(c.stdout, "backup: %s manifest=%s\n", readiness.Backup.Status, readiness.Backup.ManifestHash)
	fmt.Fprintf(c.stdout, "restore_plan: %s\n", readiness.RestorePlan.Status)
	fmt.Fprintf(c.stdout, "audit_coverage: %s gaps=%d\n", readiness.AuditCoverage.Status, readiness.AuditCoverage.GapRequirements)
	for _, projectReadiness := range readiness.Projects {
		fmt.Fprintf(c.stdout, "project.%s: status=%s needs_attention=%d blocked=%d\n",
			projectReadiness.Project.Key,
			projectReadiness.Status,
			projectReadiness.NeedsAttentionItems,
			projectReadiness.BlockedItems,
		)
	}
	for _, item := range readiness.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Message,
		)
	}
	if len(readiness.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(readiness.ForbiddenActions, ","))
	}
}

func (c command) printReleaseRemediationPlan(plan project.ReleaseRemediationPlan) {
	fmt.Fprintf(c.stdout, "release remediation plan: status=%s mode=%s scope=%s project=%s actions=%d\n",
		plan.Status,
		plan.Mode,
		plan.Scope,
		plan.ProjectKey,
		len(plan.Actions),
	)
	c.printReleaseReal100Guardrail(plan.Real100Guardrail)
	fmt.Fprintf(c.stdout, "readiness: %s\n", plan.Readiness.Status)
	for _, action := range plan.Actions {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			action.Status,
			action.Category,
			action.Key,
			action.RecommendedAction,
		)
		if action.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", action.NextCommand)
		}
		if action.Acceptance != "" {
			fmt.Fprintf(c.stdout, "  acceptance: %s\n", action.Acceptance)
		}
	}
	if len(plan.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(plan.ForbiddenActions, ","))
	}
}

func (c command) printReleaseAcceptancePreview(preview project.ReleaseAcceptancePreview) {
	fmt.Fprintf(c.stdout, "release acceptance preview: status=%s mode=%s scope=%s project=%s decisions=%d\n",
		preview.Status,
		preview.Mode,
		preview.Scope,
		preview.ProjectKey,
		len(preview.Decisions),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "remediation: %s\n", preview.Remediation.Status)
	for _, decision := range preview.Decisions {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			decision.Status,
			decision.Category,
			decision.Key,
			decision.Reason,
		)
		if decision.AcceptanceType != "" {
			fmt.Fprintf(c.stdout, "  acceptance_type: %s\n", decision.AcceptanceType)
		}
		if decision.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", decision.NextCommand)
		}
		if len(decision.RequiredEvidence) > 0 {
			fmt.Fprintf(c.stdout, "  evidence: %s\n", strings.Join(decision.RequiredEvidence, "; "))
		}
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleaseAcceptanceGate(gate project.ReleaseAcceptanceGate) {
	fmt.Fprintf(c.stdout, "release acceptance gate: status=%s mode=%s scope=%s project=%s items=%d\n",
		gate.Status,
		gate.Mode,
		gate.Scope,
		gate.ProjectKey,
		len(gate.Items),
	)
	c.printReleaseReal100Guardrail(gate.Real100Guardrail)
	fmt.Fprintf(c.stdout, "preview: %s\n", gate.Preview.Status)
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Message,
		)
		if item.DecisionStatus != "" {
			fmt.Fprintf(c.stdout, "  decision: %s\n", item.DecisionStatus)
		}
		if item.AcceptanceType != "" {
			fmt.Fprintf(c.stdout, "  acceptance_type: %s\n", item.AcceptanceType)
		}
		if item.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", item.NextCommand)
		}
		if len(item.RequiredEvidence) > 0 {
			fmt.Fprintf(c.stdout, "  evidence: %s\n", strings.Join(item.RequiredEvidence, "; "))
		}
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printReleaseExceptionDoctor(doctor project.ReleaseExceptionDoctor) {
	fmt.Fprintf(c.stdout, "release exception doctor: status=%s mode=%s scope=%s project=%s checks=%d\n",
		doctor.Status,
		doctor.Mode,
		doctor.Scope,
		doctor.ProjectKey,
		len(doctor.Checks),
	)
	c.printReleaseReal100Guardrail(doctor.Real100Guardrail)
	fmt.Fprintf(c.stdout, "gate: %s\n", doctor.Gate.Status)
	for _, check := range doctor.Checks {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			check.Status,
			check.Category,
			check.Key,
			check.Message,
		)
	}
	if len(doctor.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(doctor.ForbiddenActions, ","))
	}
}

func (c command) printReleaseExceptionRecordPreview(preview project.ReleaseExceptionRecordPreview) {
	fmt.Fprintf(c.stdout, "release exception record preview: status=%s mode=%s scope=%s project=%s drafts=%d\n",
		preview.Status,
		preview.Mode,
		preview.Scope,
		preview.ProjectKey,
		len(preview.Drafts),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "doctor: %s\n", preview.Doctor.Status)
	for _, draft := range preview.Drafts {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n",
			draft.Status,
			draft.Key,
			draft.Reason,
		)
		if draft.AcceptanceType != "" {
			fmt.Fprintf(c.stdout, "  acceptance_type: %s\n", draft.AcceptanceType)
		}
		if len(draft.AuditActions) > 0 {
			fmt.Fprintf(c.stdout, "  audit: %s\n", strings.Join(draft.AuditActions, ","))
		}
		if draft.RollbackPlan != "" {
			fmt.Fprintf(c.stdout, "  rollback: %s\n", draft.RollbackPlan)
		}
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleaseExceptionSchemaPreview(preview project.ReleaseExceptionSchemaPreview) {
	fmt.Fprintf(c.stdout, "release exception schema preview: status=%s mode=%s scope=%s project=%s tables=%d\n",
		preview.Status,
		preview.Mode,
		preview.Scope,
		preview.ProjectKey,
		len(preview.Tables),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "record_preview: %s\n", preview.RecordPreview.Status)
	for _, table := range preview.Tables {
		fmt.Fprintf(c.stdout, "table.%s: columns=%d indexes=%d foreign_keys=%d\n",
			table.Name,
			len(table.Columns),
			len(table.Indexes),
			len(table.ForeignKeys),
		)
	}
	for _, step := range preview.ApplySteps {
		fmt.Fprintf(c.stdout, "apply.%d: %s\n", step.Order, step.Action)
	}
	for _, step := range preview.RollbackSteps {
		fmt.Fprintf(c.stdout, "rollback.%d: %s\n", step.Order, step.Action)
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleaseExceptionMigrationApprovalGate(gate project.ReleaseExceptionMigrationApprovalGate) {
	fmt.Fprintf(c.stdout, "release exception migration approval gate: status=%s mode=%s scope=%s project=%s items=%d\n",
		gate.Status,
		gate.Mode,
		gate.Scope,
		gate.ProjectKey,
		len(gate.Items),
	)
	c.printReleaseReal100Guardrail(gate.Real100Guardrail)
	fmt.Fprintf(c.stdout, "schema_preview: %s\n", gate.SchemaPreview.Status)
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n",
			item.Status,
			item.Key,
			item.Message,
		)
		if item.ApprovalStatus != "" {
			fmt.Fprintf(c.stdout, "  approval: %s\n", item.ApprovalStatus)
		}
		if item.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", item.NextCommand)
		}
		if len(item.RequiredEvidence) > 0 {
			fmt.Fprintf(c.stdout, "  evidence: %s\n", strings.Join(item.RequiredEvidence, "; "))
		}
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printReleaseExceptionApplyPreview(preview project.ReleaseExceptionApplyPreview) {
	fmt.Fprintf(c.stdout, "release exception apply preview: status=%s mode=%s scope=%s project=%s items=%d\n",
		preview.Status,
		preview.Mode,
		preview.Scope,
		preview.ProjectKey,
		len(preview.Items),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "migration_gate: %s\n", preview.MigrationGate.Status)
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] %s: %s\n",
			item.Status,
			item.Key,
			item.Message,
		)
		if item.Action != "" {
			fmt.Fprintf(c.stdout, "  action: %s\n", item.Action)
		}
		if item.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", item.NextCommand)
		}
		if len(item.RequiredEvidence) > 0 {
			fmt.Fprintf(c.stdout, "  evidence: %s\n", strings.Join(item.RequiredEvidence, "; "))
		}
	}
	for _, step := range preview.ApplySteps {
		fmt.Fprintf(c.stdout, "apply.%d: %s\n", step.Order, step.Action)
	}
	for _, step := range preview.RollbackSteps {
		fmt.Fprintf(c.stdout, "rollback.%d: %s\n", step.Order, step.Action)
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleaseFinalGate(gate project.ReleaseFinalGate) {
	fmt.Fprintf(c.stdout, "release final gate: status=%s mode=%s items=%d\n",
		gate.Status,
		gate.Mode,
		len(gate.Items),
	)
	c.printReleaseReal100Guardrail(gate.Real100Guardrail)
	fmt.Fprintf(c.stdout, "readiness: %s\n", gate.Readiness.Status)
	fmt.Fprintf(c.stdout, "acceptance_gate: %s\n", gate.AcceptanceGate.Status)
	fmt.Fprintf(c.stdout, "exception_apply: %s\n", gate.ExceptionApply.Status)
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Message,
		)
		if item.NextCommand != "" {
			fmt.Fprintf(c.stdout, "  next: %s\n", item.NextCommand)
		}
		if len(item.RequiredEvidence) > 0 {
			fmt.Fprintf(c.stdout, "  evidence: %s\n", strings.Join(item.RequiredEvidence, "; "))
		}
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printReleaseEvidenceBundle(bundle project.ReleaseEvidenceBundle) {
	fmt.Fprintf(c.stdout, "release evidence bundle: status=%s mode=%s items=%d\n",
		bundle.Status,
		bundle.Mode,
		len(bundle.Items),
	)
	c.printReleaseReal100Guardrail(bundle.Real100Guardrail)
	fmt.Fprintf(c.stdout, "final_gate: %s\n", bundle.FinalGate.Status)
	fmt.Fprintf(c.stdout, "backup: %s hash=%s\n", bundle.Backup.Status, bundle.Backup.ManifestHash)
	fmt.Fprintf(c.stdout, "audit_coverage: %s gaps=%d\n", bundle.AuditCoverage.Status, bundle.AuditCoverage.GapRequirements)
	for _, item := range bundle.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Description,
		)
	}
	if len(bundle.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(bundle.ForbiddenActions, ","))
	}
}

func (c command) printReleasePackagePreview(preview project.ReleasePackagePreview) {
	fmt.Fprintf(c.stdout, "release package preview: status=%s mode=%s package=%s items=%d\n",
		preview.Status,
		preview.Mode,
		preview.PackageName,
		len(preview.Items),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "evidence_bundle: %s\n", preview.EvidenceBundle.Status)
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] %s -> %s\n",
			item.Status,
			item.Key,
			item.PackagePath,
		)
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleaseDistributionPreview(preview project.ReleaseDistributionPreview) {
	fmt.Fprintf(c.stdout, "release distribution preview: status=%s mode=%s items=%d\n",
		preview.Status,
		preview.Mode,
		len(preview.Items),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "package_preview: %s package=%s\n", preview.PackagePreview.Status, preview.PackagePreview.PackageName)
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s action=%s next=%s\n",
			item.Status,
			item.Channel,
			item.Key,
			item.Action,
			item.NextCommand,
		)
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleasePublishGate(gate project.ReleasePublishGate) {
	fmt.Fprintf(c.stdout, "release publish gate: status=%s mode=%s items=%d\n",
		gate.Status,
		gate.Mode,
		len(gate.Items),
	)
	c.printReleaseReal100Guardrail(gate.Real100Guardrail)
	fmt.Fprintf(c.stdout, "distribution_preview: %s\n", gate.DistributionPreview.Status)
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s channel=%s next=%s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Channel,
			item.NextCommand,
		)
	}
	if len(gate.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(gate.ForbiddenActions, ","))
	}
}

func (c command) printReleasePublishApprovalPreview(preview project.ReleasePublishApprovalPreview) {
	fmt.Fprintf(c.stdout, "release publish approval preview: status=%s mode=%s items=%d\n",
		preview.Status,
		preview.Mode,
		len(preview.Items),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "publish_gate: %s\n", preview.PublishGate.Status)
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s approval=%s channel=%s next=%s\n",
			item.Status,
			item.Category,
			item.Key,
			item.ApprovalStatus,
			item.Channel,
			item.NextCommand,
		)
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printReleaseRolloutPlanPreview(preview project.ReleaseRolloutPlanPreview) {
	fmt.Fprintf(c.stdout, "release rollout plan preview: status=%s mode=%s items=%d\n",
		preview.Status,
		preview.Mode,
		len(preview.Items),
	)
	c.printReleaseReal100Guardrail(preview.Real100Guardrail)
	fmt.Fprintf(c.stdout, "publish_approval_preview: %s\n", preview.PublishApprovalPreview.Status)
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] %s/%s stage=%s action=%s next=%s\n",
			item.Status,
			item.Category,
			item.Key,
			item.Stage,
			item.Action,
			item.NextCommand,
		)
	}
	if len(preview.RolloutSteps) > 0 {
		fmt.Fprintf(c.stdout, "rollout_steps: %d\n", len(preview.RolloutSteps))
	}
	if len(preview.VerificationCheckpoints) > 0 {
		fmt.Fprintf(c.stdout, "verification_checkpoints: %d\n", len(preview.VerificationCheckpoints))
	}
	if len(preview.RollbackSteps) > 0 {
		fmt.Fprintf(c.stdout, "rollback_steps: %d\n", len(preview.RollbackSteps))
	}
	if len(preview.ForbiddenActions) > 0 {
		fmt.Fprintf(c.stdout, "forbidden_actions: %s\n", strings.Join(preview.ForbiddenActions, ","))
	}
}

func (c command) printAuditCoverage(coverage project.AuditCoverage) {
	fmt.Fprintf(c.stdout, "audit coverage: status=%s scope=%s events=%d covered=%d gaps=%d\n",
		coverage.Status,
		coverage.Scope,
		coverage.TotalAuditEvents,
		coverage.CoveredRequirements,
		coverage.GapRequirements,
	)
	for _, requirement := range coverage.Requirements {
		fmt.Fprintf(c.stdout, "[%s] %s/%s evidence=%d\n",
			requirement.Status,
			requirement.Category,
			requirement.Key,
			requirement.EvidenceCount,
		)
		if len(requirement.MissingActions) > 0 {
			fmt.Fprintf(c.stdout, "  missing: %s\n", strings.Join(requirement.MissingActions, ","))
		}
	}
}

func (c command) printPermissionPolicyDoctor(doctor project.PermissionPolicyDoctor) {
	fmt.Fprintf(c.stdout, "permission policy doctor: %s status=%s\n", doctor.Project.Key, doctor.Status)
	for _, check := range doctor.Checks {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			check.Status,
			check.Category,
			check.Key,
			check.Message,
		)
	}
}

func (c command) printArtifactIntegrity(report project.ArtifactIntegrityReport) {
	fmt.Fprintf(c.stdout, "artifact integrity: %s status=%s checked=%d passed=%d warn=%d failed=%d skipped=%d\n",
		report.Project.Key,
		report.Status,
		report.CheckedArtifacts,
		report.PassedArtifacts,
		report.WarnArtifacts,
		report.FailedArtifacts,
		report.SkippedArtifacts,
	)
	for _, check := range report.Checks {
		fmt.Fprintf(c.stdout, "[%s] artifact=%d backend=%s type=%s: %s\n",
			check.Status,
			check.Artifact.ID,
			check.Artifact.StorageBackend,
			check.Artifact.ArtifactType,
			check.Message,
		)
	}
}

func (c command) printArtifactArchivePreview(preview project.ArtifactArchivePreviewResult) {
	fmt.Fprintf(c.stdout, "artifact archive preview: %s status=%s mode=%s created=%t\n",
		preview.Project.Key,
		preview.Status,
		preview.Mode,
		preview.Created,
	)
	fmt.Fprintf(c.stdout, "summary: total=%d candidates=%d retained=%d external_refs=%d needs_policy=%d\n",
		preview.Summary.TotalArtifacts,
		preview.Summary.ArchiveCandidates,
		preview.Summary.RetainedArtifacts,
		preview.Summary.ExternalRefs,
		preview.Summary.NeedsPolicy,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", preview.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", preview.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "storage_write_attempted: %t\n", preview.StorageWriteAttempted)
	fmt.Fprintf(c.stdout, "artifact_delete_attempted: %t\n", preview.ArtifactDeleteAttempted)
	for _, item := range preview.Items {
		fmt.Fprintf(c.stdout, "[%s] artifact=%d backend=%s retention=%s action=%s decision=%s\n",
			item.ArchiveState,
			item.ArtifactID,
			item.StorageBackend,
			item.RetentionClass,
			item.Action,
			item.Decision,
		)
	}
}

func (c command) printConformance(report project.ConformanceReport) {
	fmt.Fprintf(c.stdout, "conformance: %s status=%s profile=%s adapter=%s stages=%d gates=%d\n",
		report.Project.Key,
		report.Status,
		report.ProfileID,
		report.Adapter,
		report.StageCount,
		report.GateCount,
	)
	fmt.Fprintf(c.stdout, "profile_hash: %s\n", report.ProfileHash)
	for _, check := range report.Checks {
		fmt.Fprintf(c.stdout, "[%s] %s/%s: %s\n",
			check.Status,
			check.Category,
			check.Key,
			check.Message,
		)
	}
}

func (c command) printWorkerPoolSummary(summary project.WorkerPoolSummary) {
	fmt.Fprintf(c.stdout, "worker pool: projects=%d workers=%d online=%d active_leases=%d queued_tasks=%d needs_recovery=%d\n",
		summary.TotalProjects,
		summary.TotalWorkers,
		summary.TotalOnlineWorkers,
		summary.TotalActiveLeases,
		summary.TotalQueuedTasks,
		summary.TotalNeedsRecovery,
	)
	for _, projectSummary := range summary.Projects {
		heartbeat := "never"
		if projectSummary.LastWorkerHeartbeat != nil {
			heartbeat = formatTime(*projectSummary.LastWorkerHeartbeat)
		}
		fmt.Fprintf(c.stdout, "%s\tworkers=%d/%d\tleases=%d\ttasks=%d\trecovery=%d\tpriority=%d\tmax_parallel=%d\trole=%s\trole_status=%s\tcaps=%s\ttypes=%s\trequired=%s\tengine=%s\tengine_status=%s\tresource_status=%s\tlast_heartbeat=%s\n",
			projectSummary.Project.Key,
			projectSummary.OnlineWorkers,
			projectSummary.Workers,
			projectSummary.ActiveLeases,
			projectSummary.QueuedTasks,
			projectSummary.NeedsRecoveryLeases+projectSummary.NeedsRecoveryTasks,
			projectSummary.Scheduling.Priority,
			projectSummary.Scheduling.MaxParallelTasks,
			projectSummary.Scheduling.AgentRole,
			projectSummary.Role.Status,
			strings.Join(projectSummary.Capabilities, ","),
			strings.Join(projectSummary.WorkerTypes, ","),
			strings.Join(projectSummary.Scheduling.RequiredCapabilities, ","),
			projectSummary.Scheduling.EngineProfile,
			projectSummary.Engine.Status,
			projectSummary.Resources.Status,
			heartbeat,
		)
	}
}

func (c command) printWorkerPoolSchedulePreview(preview project.WorkerPoolSchedulePreview) {
	fmt.Fprintf(c.stdout, "worker schedule preview: recommended=%d blocked=%d queued_tasks=%d available_slots=%d policy=%s dry_run_only=%t\n",
		preview.Recommended,
		preview.Blocked,
		preview.QueuedTasks,
		preview.AvailableSlot,
		preview.Policy.Strategy,
		preview.Policy.DryRunOnly,
	)
	for _, schedule := range preview.Projects {
		reasons := "none"
		if len(schedule.BlockedReasons) > 0 {
			reasons = strings.Join(schedule.BlockedReasons, ",")
		}
		fmt.Fprintf(c.stdout, "%s\trecommended=%t\tpriority=%d\tmax_parallel=%d\trole=%s\trole_status=%s\tqueued=%d\tslots=%d\trequired=%s\tengine=%s\tengine_status=%s\tresource_status=%s\tnext=%s\tblocked=%s\n",
			schedule.Project.Key,
			schedule.Recommended,
			schedule.Priority,
			schedule.MaxParallel,
			schedule.AgentRole,
			schedule.Role.Status,
			schedule.QueuedTasks,
			schedule.AvailableSlots,
			strings.Join(schedule.RequiredCaps, ","),
			schedule.EngineProfile,
			schedule.Engine.Status,
			schedule.Resources.Status,
			schedule.NextAction,
			reasons,
		)
	}
}

func (c command) printCodexCLIAdapterPreview(preview project.CodexCLIAdapterPreview) {
	fmt.Fprintf(c.stdout, "codex cli adapter preview: %s status=%s mode=%s\n",
		preview.Project.Key,
		preview.Status,
		preview.Mode,
	)
	fmt.Fprintf(c.stdout, "engine: profile=%s provider=%s status=%s enabled=%t secret_required=%t secret_ready=%t\n",
		preview.Engine.ProfileID,
		preview.Engine.Provider,
		preview.Engine.Status,
		preview.Engine.Enabled,
		preview.Engine.SecretRequired,
		preview.Engine.SecretReady,
	)
	fmt.Fprintf(c.stdout, "command: %s allowed=%t reason=%s\n",
		preview.Command.Command,
		preview.Command.Allowed,
		preview.Command.Reason,
	)
	fmt.Fprintf(c.stdout, "execution_allowed: %t\n", preview.ExecutionAllowed)
	if len(preview.Blockers) > 0 {
		fmt.Fprintf(c.stdout, "blockers: %s\n", strings.Join(preview.Blockers, ","))
	}
	fmt.Fprintf(c.stdout, "side_effects: project_write=%t execution_write=%t engine_call=%t commands_run=%t secrets_resolved=%t network=%t\n",
		preview.ProjectWriteAttempted,
		preview.ExecutionWriteAttempted,
		preview.EngineCallAttempted,
		preview.CommandsRun,
		preview.SecretsResolved,
		preview.NetworkUsed,
	)
	fmt.Fprintf(c.stdout, "artifact_redaction: %s retention=%s fields=%s\n",
		preview.ArtifactRedaction.Status,
		preview.ArtifactRedaction.RetentionClass,
		strings.Join(preview.ArtifactRedaction.RedactedFields, ","),
	)
}

func (c command) printLease(lease project.LeaseRecord) {
	fmt.Fprintf(c.stdout, "lease: id=%d status=%s kind=%s run_task=%d worker=%d\n",
		lease.ID,
		lease.Status,
		lease.LeaseKind,
		lease.RunTaskID,
		lease.WorkerID,
	)
	fmt.Fprintf(c.stdout, "expires_at: %s\n", formatTime(lease.ExpiresAt))
	if lease.ReleasedAt != nil {
		fmt.Fprintf(c.stdout, "released_at: %s\n", formatTime(*lease.ReleasedAt))
	}
	if len(lease.AllowedCapabilities) > 0 {
		fmt.Fprintf(c.stdout, "allowed_capabilities: %s\n", strings.Join(lease.AllowedCapabilities, ","))
	}
}

func (c command) printLeases(record project.Record, leases []project.LeaseRecord) {
	fmt.Fprintf(c.stdout, "leases: %s count=%d\n", record.Key, len(leases))
	for _, lease := range leases {
		fmt.Fprintf(c.stdout, "%d\t%s\t%s\trun_task=%d\n",
			lease.ID,
			lease.LeaseKind,
			lease.Status,
			lease.RunTaskID,
		)
	}
}

func (c command) printWorkerRunOnce(result project.WorkerRunOnceResult) {
	if !result.Claimed {
		fmt.Fprintf(c.stdout, "worker run-once: %s idle\n", result.Worker.WorkerKey)
		return
	}
	fmt.Fprintf(c.stdout, "worker run-once: %s claimed run_task=%d lease=%d status=%s\n",
		result.Worker.WorkerKey,
		result.Task.ID,
		result.Lease.ID,
		result.Lease.Status,
	)
	fmt.Fprintf(c.stdout, "attempt=%d artifact=%d artifact_type=%s\n",
		result.Attempt.ID,
		result.Artifact.ID,
		result.Artifact.ArtifactType,
	)
}

func (c command) printFixtureExecution(result project.FixtureExecutionResult) {
	fmt.Fprintf(c.stdout, "fixture execution: %s/%s run=%d task=%d status=%s decision=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.Status,
		result.Decision,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "worker=%s lease=%d attempt=%d artifact=%d artifact_type=%s\n",
		result.Worker.WorkerKey,
		result.Lease.ID,
		result.Attempt.ID,
		result.Artifact.ID,
		result.Artifact.ArtifactType,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", result.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", result.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", result.TaskClaimed)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", result.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", result.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", result.ArtifactCreated)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printReadOnlyVerify(result project.ReadOnlyVerifyResult) {
	fmt.Fprintf(c.stdout, "read-only verify: %s/%s run=%d task=%d status=%s decision=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.Status,
		result.Decision,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "target=%s sha256=%s size=%d\n",
		result.TargetPath,
		result.TargetSHA256,
		result.TargetSizeBytes,
	)
	fmt.Fprintf(c.stdout, "worker=%s lease=%d attempt=%d artifact=%d artifact_type=%s\n",
		result.Worker.WorkerKey,
		result.Lease.ID,
		result.Attempt.ID,
		result.Artifact.ID,
		result.Artifact.ArtifactType,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_read_allowed: %t\n", result.ProjectReadAllowed)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", result.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", result.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", result.TaskClaimed)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", result.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", result.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", result.ArtifactCreated)
	fmt.Fprintf(c.stdout, "verification_passed: %t\n", result.VerificationPassed)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printApprovedArtifactWrite(result project.ApprovedArtifactWriteResult) {
	fmt.Fprintf(c.stdout, "approved artifact write: %s/%s run=%d task=%d status=%s decision=%s artifact_label=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.Status,
		result.Decision,
		result.ArtifactLabel,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "worker=%s lease=%d attempt=%d artifact=%d artifact_type=%s\n",
		result.Worker.WorkerKey,
		result.Lease.ID,
		result.Attempt.ID,
		result.Artifact.ID,
		result.Artifact.ArtifactType,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", result.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", result.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", result.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", result.TaskClaimed)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", result.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", result.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", result.ArtifactCreated)
	fmt.Fprintf(c.stdout, "artifact_write_passed: %t\n", result.ArtifactWritePassed)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printFixtureProjectWrite(result project.FixtureProjectWriteResult) {
	fmt.Fprintf(c.stdout, "fixture project write: %s/%s run=%d task=%d status=%s decision=%s target=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.Status,
		result.Decision,
		result.TargetPath,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "worker=%s lease=%d copy_attempt=%d verify_attempt=%d rollback_attempt=%d report_artifact=%d artifact_type=%s\n",
		result.Worker.WorkerKey,
		result.Lease.ID,
		result.CopyAttempt.ID,
		result.VerifyAttempt.ID,
		result.RollbackAttempt.ID,
		result.Artifact.ID,
		result.Artifact.ArtifactType,
	)
	fmt.Fprintf(c.stdout, "write_set_artifact=%d preimage_artifact=%d expected_before_sha256=%s expected_before_size=%d after_sha256=%s after_size=%d restored_sha256=%s restored_size=%d\n",
		result.WriteSetArtifact.ID,
		result.PreimageArtifact.ID,
		result.ExpectedBeforeSHA256,
		result.ExpectedBeforeSize,
		result.AfterSHA256,
		result.AfterSize,
		result.RestoredSHA256,
		result.RestoredSize,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_read_allowed: %t\n", result.ProjectReadAllowed)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "project_write_allowed: %t\n", result.ProjectWriteAllowed)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", result.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", result.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", result.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", result.TaskClaimed)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", result.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", result.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", result.ArtifactCreated)
	fmt.Fprintf(c.stdout, "write_set_passed: %t\n", result.WriteSetPassed)
	fmt.Fprintf(c.stdout, "verification_passed: %t\n", result.VerificationPassed)
	fmt.Fprintf(c.stdout, "rollback_attempted: %t\n", result.RollbackAttempted)
	fmt.Fprintf(c.stdout, "rollback_verified: %t\n", result.RollbackVerified)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printManagedGeneratedWrite(result project.ManagedGeneratedWriteResult) {
	fmt.Fprintf(c.stdout, "managed generated write: %s/%s run=%d task=%d status=%s decision=%s target=%s created=%t\n",
		result.Project.Key,
		result.Version.DisplayLabel,
		result.Run.ID,
		result.Task.ID,
		result.Status,
		result.Decision,
		result.TargetPath,
		result.Created,
	)
	fmt.Fprintf(c.stdout, "worker=%s lease=%d copy_attempt=%d verify_attempt=%d rollback_attempt=%d report_artifact=%d artifact_type=%s\n",
		result.Worker.WorkerKey,
		result.Lease.ID,
		result.CopyAttempt.ID,
		result.VerifyAttempt.ID,
		result.RollbackAttempt.ID,
		result.Artifact.ID,
		result.Artifact.ArtifactType,
	)
	fmt.Fprintf(c.stdout, "write_set_artifact=%d preimage_artifact=%d expected_before_sha256=%s expected_before_size=%d after_sha256=%s after_size=%d restored_sha256=%s restored_size=%d\n",
		result.WriteSetArtifact.ID,
		result.PreimageArtifact.ID,
		result.ExpectedBeforeSHA256,
		result.ExpectedBeforeSize,
		result.AfterSHA256,
		result.AfterSize,
		result.RestoredSHA256,
		result.RestoredSize,
	)
	fmt.Fprintf(c.stdout, "idempotency_key: %s\n", result.IdempotencyKey)
	fmt.Fprintf(c.stdout, "generated_only: %t\n", result.GeneratedOnly)
	fmt.Fprintf(c.stdout, "generated_only_apply_open: %t\n", result.GeneratedOnlyApplyOpen)
	fmt.Fprintf(c.stdout, "project_read_attempted: %t\n", result.ProjectReadAttempted)
	fmt.Fprintf(c.stdout, "project_read_allowed: %t\n", result.ProjectReadAllowed)
	fmt.Fprintf(c.stdout, "project_write_attempted: %t\n", result.ProjectWriteAttempted)
	fmt.Fprintf(c.stdout, "project_write_allowed: %t\n", result.ProjectWriteAllowed)
	fmt.Fprintf(c.stdout, "execution_write_attempted: %t\n", result.ExecutionWriteAttempted)
	fmt.Fprintf(c.stdout, "area_flow_artifact_written: %t\n", result.AreaFlowArtifactWritten)
	fmt.Fprintf(c.stdout, "area_flow_execution_state_written: %t\n", result.AreaFlowExecutionStateWritten)
	fmt.Fprintf(c.stdout, "engine_call_attempted: %t\n", result.EngineCallAttempted)
	fmt.Fprintf(c.stdout, "commands_run: %t\n", result.CommandsRun)
	fmt.Fprintf(c.stdout, "secrets_resolved: %t\n", result.SecretsResolved)
	fmt.Fprintf(c.stdout, "network_used: %t\n", result.NetworkUsed)
	fmt.Fprintf(c.stdout, "task_claimed: %t\n", result.TaskClaimed)
	fmt.Fprintf(c.stdout, "lease_created: %t\n", result.LeaseCreated)
	fmt.Fprintf(c.stdout, "attempt_created: %t\n", result.AttemptCreated)
	fmt.Fprintf(c.stdout, "artifact_created: %t\n", result.ArtifactCreated)
	fmt.Fprintf(c.stdout, "write_set_passed: %t\n", result.WriteSetPassed)
	fmt.Fprintf(c.stdout, "verification_passed: %t\n", result.VerificationPassed)
	fmt.Fprintf(c.stdout, "rollback_attempted: %t\n", result.RollbackAttempted)
	fmt.Fprintf(c.stdout, "rollback_verified: %t\n", result.RollbackVerified)
	for _, blocker := range result.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
}

func (c command) printEvents(record project.Record, events []project.EventRecord) {
	fmt.Fprintf(c.stdout, "project events: %s count=%d\n", record.Key, len(events))
	for _, event := range events {
		status := metadataString(event.Metadata, "overall_status")
		counts := metadataValue(event.Metadata, "counts")
		if status != "" {
			fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\tstatus=%s\tcounts=%s\n",
				event.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				event.Severity,
				event.Type,
				event.Message,
				status,
				counts,
			)
			continue
		}
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n",
			event.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			event.Severity,
			event.Type,
			event.Message,
		)
	}
}

func (c command) printStatusProjections(record project.Record, projections []project.StatusProjectionRecord) {
	fmt.Fprintf(c.stdout, "status projections: %s count=%d\n", record.Key, len(projections))
	for _, projection := range projections {
		writtenAt := ""
		if projection.WrittenAt != nil {
			writtenAt = projection.WrittenAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\t%s\twritten_at=%s\n",
			projection.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z"),
			projection.TargetKind,
			projection.TargetURI,
			projection.SummaryState,
			projection.WriteState,
			writtenAt,
		)
	}
}

func (c command) printStatusProjectionApply(result project.ApplyStatusProjectionResult) {
	fmt.Fprintf(c.stdout, "status projection apply: project=%s status=%s decision=%s target=%s projection_id=%d snapshot_id=%d idempotency_key=%s\n",
		result.Project.Key,
		result.Status,
		result.Decision,
		result.TargetURI,
		result.StatusProjectionID,
		result.SnapshotID,
		result.IdempotencyKey,
	)
	if result.WrittenTarget != "" {
		fmt.Fprintf(c.stdout, "written_target=%s write_hash=%s write_size=%d\n",
			result.WrittenTarget,
			result.WriteHash,
			result.WriteSize,
		)
	}
	if result.PreimageCaptured || result.PostWriteVerified || result.ProtectedPathsVerified || result.RootContained || result.StableProjectionValid || result.AtomicReplaceUsed || result.RollbackCompensation {
		fmt.Fprintf(c.stdout, "write_safety: preimage_captured=%t preimage_exists=%t preimage_sha256=%s preimage_size=%d post_write_verified=%t post_write_sha256=%s post_write_size=%d protected_paths_verified=%t protected_path_before_hash=%s protected_path_after_hash=%s expected_protected_path_hash=%s root_contained=%t stable_projection_validated=%t atomic_replace_used=%t rollback_compensation_enabled=%t\n",
			result.PreimageCaptured,
			result.PreimageExists,
			result.PreimageSHA256,
			result.PreimageSize,
			result.PostWriteVerified,
			result.PostWriteSHA256,
			result.PostWriteSize,
			result.ProtectedPathsVerified,
			result.ProtectedPathBeforeHash,
			result.ProtectedPathAfterHash,
			result.ExpectedProtectedPathHash,
			result.RootContained,
			result.StableProjectionValid,
			result.AtomicReplaceUsed,
			result.RollbackCompensation,
		)
	}
	fmt.Fprintf(c.stdout, "apply_gate: status=%s decision=%s approval_status=%s apply_command_eligible=%t\n",
		result.ApplyGateStatus,
		result.ApplyGateDecision,
		result.ApplyGateApprovalStatus,
		result.ApplyCommandEligible,
	)
	if len(result.Blockers) > 0 {
		fmt.Fprintf(c.stdout, "blockers=%s\n", strings.Join(result.Blockers, "; "))
	}
}

func (c command) printStatusProjectionAuthorizationPreview(preview project.StatusProjectionAuthorizationPreview) {
	fmt.Fprintf(c.stdout, "status projection authorization: project=%s status=%s decision=%s target=%s apply_open=%t approval_required=%t\n",
		preview.Project.Key,
		preview.Status,
		preview.Decision,
		preview.TargetURI,
		preview.ApplyOpen,
		preview.ApprovalRequired,
	)
	fmt.Fprintf(c.stdout, "claim_scope=%s not_real_100=%t\n", preview.ClaimScope, preview.NotReal100)
	fmt.Fprintf(c.stdout, "command_request_created=false status_projection_written=false project_write_attempted=%t execution_write_attempted=%t engine_call_attempted=%t\n",
		preview.ProjectWriteAttempted,
		preview.ExecutionWriteAttempted,
		preview.EngineCallAttempted,
	)
	fmt.Fprintf(c.stdout, "schema_uri=%s validator_preflight=%s\n", preview.SchemaURI, preview.ValidatorPreflight)
	if preview.RequiredAuthorizationPhrase != "" {
		fmt.Fprintf(c.stdout, "required_authorization_phrase=%s\n", preview.RequiredAuthorizationPhrase)
	}
	fmt.Fprintf(c.stdout, "preimage: exists=%t schema_status=%s sha256=%s size=%d\n",
		preview.Preimage.Exists,
		preview.Preimage.SchemaStatus,
		preview.Preimage.SHA256,
		preview.Preimage.SizeBytes,
	)
	for _, preflight := range preview.RequiredPreflight {
		fmt.Fprintf(c.stdout, "required_preflight: %s\n", preflight)
	}
	for _, path := range preview.ProtectedPaths {
		fmt.Fprintf(c.stdout, "protected_path: %s\n", path)
	}
	for _, blocker := range preview.BlockedBy {
		fmt.Fprintf(c.stdout, "blocked_by: %s\n", blocker)
	}
	for _, warning := range preview.Warnings {
		fmt.Fprintf(c.stdout, "warning: %s\n", warning)
	}
}

func (c command) printStatusProjectionApplyGate(gate project.StatusProjectionApplyGate) {
	fmt.Fprintf(c.stdout, "status projection apply gate: project=%s status=%s decision=%s target=%s apply_command_eligible=%t approval_status=%s\n",
		gate.Project.Key,
		gate.Status,
		gate.Decision,
		gate.TargetURI,
		gate.ApplyCommandEligible,
		gate.ApprovalStatus,
	)
	fmt.Fprintf(c.stdout, "claim_scope=%s not_real_100=%t apply_open=false apply_command_eligible_is_not_apply=%t requires_separate_apply_command=%t\n",
		gate.ClaimScope,
		gate.NotReal100,
		gate.ApplyCommandEligibleIsNotApply,
		gate.RequiresSeparateApplyCommand,
	)
	fmt.Fprintf(c.stdout, "command_request_created=%t status_projection_written=%t project_write_attempted=%t execution_write_attempted=%t engine_call_attempted=%t\n",
		gate.CommandRequestCreated,
		gate.StatusProjectionWritten,
		gate.ProjectWriteAttempted,
		gate.ExecutionWriteAttempted,
		gate.EngineCallAttempted,
	)
	fmt.Fprintf(c.stdout, "target_path=%s\n", gate.TargetPath)
	if gate.RequiredAuthorizationPhrase != "" {
		fmt.Fprintf(c.stdout, "required_authorization_phrase=%s\n", gate.RequiredAuthorizationPhrase)
	}
	for _, item := range gate.Items {
		fmt.Fprintf(c.stdout, "item: %s status=%s expected=%s actual=%s\n",
			item.Key,
			item.Status,
			item.Expected,
			item.Actual,
		)
		for _, blocker := range item.BlockedBy {
			fmt.Fprintf(c.stdout, "blocked_by: %s\n", blocker)
		}
	}
}

func (c command) printStatusProjectionApplyPacketPreview(preview project.StatusProjectionApplyPacketPreview) {
	fmt.Fprintf(c.stdout, "status projection apply packet: project=%s status=%s decision=%s target=%s apply_command_eligible=%t approval_status=%s\n",
		preview.Project.Key,
		preview.Status,
		preview.Decision,
		preview.Packet.TargetURI,
		preview.Gate.ApplyCommandEligible,
		preview.Gate.ApprovalStatus,
	)
	fmt.Fprintf(c.stdout, "claim_scope=%s not_real_100=%t apply_command_eligible_is_not_apply=%t requires_separate_apply_command=%t\n",
		preview.ClaimScope,
		preview.NotReal100,
		preview.ApplyCommandEligibleIsNotApply,
		preview.RequiresSeparateApplyCommand,
	)
	fmt.Fprintf(c.stdout, "command_request_created=%t status_projection_written=%t project_write_attempted=%t execution_write_attempted=%t engine_call_attempted=%t\n",
		preview.CommandRequestCreated,
		preview.StatusProjectionWritten,
		preview.ProjectWriteAttempted,
		preview.ExecutionWriteAttempted,
		preview.EngineCallAttempted,
	)
	fmt.Fprintf(c.stdout, "packet: source_hash=%s expected_before_exists=%t expected_before_sha256=%s expected_before_size=%d schema=%s preimage_schema=%s\n",
		preview.Packet.SourceHash,
		preview.Packet.ExpectedBeforeExists,
		preview.Packet.ExpectedBeforeSHA256,
		preview.Packet.ExpectedBeforeSizeBytes,
		preview.Packet.SchemaURI,
		preview.Packet.AcceptedPreimageSchemaStatus,
	)
	fmt.Fprintf(c.stdout, "validator_preflight=%s\n", preview.Packet.ValidatorPreflight)
	fmt.Fprintf(c.stdout, "protected_path_check=%s\n", preview.Packet.ProtectedPathCheck)
	fmt.Fprintf(c.stdout, "rollback_action=%s\n", preview.Packet.RollbackAction)
	if preview.RequiredAuthorizationPhrase != "" {
		fmt.Fprintf(c.stdout, "required_authorization_phrase=%s\n", preview.RequiredAuthorizationPhrase)
	}
	for _, blocker := range preview.Blockers {
		fmt.Fprintf(c.stdout, "blocker: %s\n", blocker)
	}
	fmt.Fprintf(c.stdout, "apply_command: %s\n", strings.Join(preview.ApplyCommand, " "))
	for _, review := range preview.RequiredHumanReview {
		fmt.Fprintf(c.stdout, "required_human_review: %s\n", review)
	}
	for _, item := range preview.Gate.Items {
		if item.Status == "pass" {
			continue
		}
		fmt.Fprintf(c.stdout, "blocked_item: %s status=%s expected=%s actual=%s\n",
			item.Key,
			item.Status,
			item.Expected,
			item.Actual,
		)
		for _, blocker := range item.BlockedBy {
			fmt.Fprintf(c.stdout, "blocked_by: %s\n", blocker)
		}
	}
}

func appStatusProjectionWriter(ctx context.Context, record project.Record, snapshot project.Snapshot, targetURI string) (project.StatusProjectionWriteResult, error) {
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

func metadataString(metadata map[string]any, key string) string {
	value, ok := metadata[key].(string)
	if !ok {
		return ""
	}
	return value
}

func metadataBoolValue(metadata map[string]any, key string) bool {
	value, ok := metadata[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true"
	default:
		return false
	}
}

func metadataInt64Value(metadata map[string]any, key string) int64 {
	value, ok := metadata[key]
	if !ok {
		return 0
	}
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
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func metadataStringSlice(metadata map[string]any, key string) []string {
	value, ok := metadata[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func metadataStringMapValue(metadata map[string]any, key string) map[string]string {
	value, ok := metadata[key]
	if !ok || value == nil {
		return nil
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
		return nil
	}
}

func protectedPathProofAuthorizedJSONComplete(metadata map[string]any) bool {
	return strings.TrimSpace(metadataString(metadata, "authorized_approval_id")) != "" &&
		len(metadataStringSlice(metadata, "authorized_allowed_paths")) > 0 &&
		len(strings.TrimSpace(metadataString(metadata, "authorized_dirty_output_hash"))) == 64 &&
		strings.TrimSpace(metadataString(metadata, "authorized_reviewer")) != "" &&
		strings.TrimSpace(metadataString(metadata, "authorized_rollback_evidence_uri")) != "" &&
		len(metadataStringSlice(metadata, "authorized_touched_paths")) > 0
}

func metadataValue(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func nestedSummaryValue(metadata map[string]any, key string, nestedKey string) string {
	value, ok := metadata[key].(map[string]any)
	if !ok {
		return ""
	}
	return metadataValue(value, nestedKey)
}
