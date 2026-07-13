package project

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/areasong/areaflow/internal/migrate"
)

type SupportBundlePreviewOptions struct {
	GeneratedAt time.Time
}

type SupportBundlePathReference struct {
	Key         string
	Kind        string
	URI         string
	ProjectKey  string
	Description string
}

type SupportBundleHashReference struct {
	Key         string
	Hash        string
	Source      string
	Description string
}

type SupportBundlePreview struct {
	Status                   string
	Mode                     string
	BundleID                 string
	Scope                    string
	Projects                 []Record
	IncludedMetadata         []string
	ExcludedSensitiveContent []string
	PathReferences           []SupportBundlePathReference
	Hashes                   []SupportBundleHashReference
	Capabilities             []string
	ForbiddenActions         []string
	SafetyFacts              map[string]bool
	GeneratedAt              time.Time
}

type MigrationLedgerReadinessOptions struct {
	GeneratedAt time.Time
}

type MigrationLedgerEntry struct {
	Name             string
	Applied          bool
	Status           string
	RequiredEvidence []string
	Phases           []MigrationLedgerPhase
	Metadata         map[string]any
}

type MigrationLedgerPhase struct {
	Phase       string
	Status      string
	Message     string
	Remediation string
	Metadata    map[string]any
}

type MigrationLedgerReadiness struct {
	Status                               string
	Mode                                 string
	Entries                              []MigrationLedgerEntry
	AppliedCount                         int
	PendingCount                         int
	SchemaMigrationsTablePresent         bool
	FullLedgerTablePresent               bool
	PreflightApplyVerifyRemediationReady bool
	Capabilities                         []string
	ForbiddenActions                     []string
	SafetyFacts                          map[string]bool
	GeneratedAt                          time.Time
}

type OperationsReadinessOptions struct {
	APIBaseURL              string
	WebDashboardURL         string
	GeneratedAt             time.Time
	SmokeProof              OperationsSmokeProof
	SmokeProofProject       Record
	SmokeProofProjectScoped bool
}

type OperationsReadinessItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	EvidenceRefs     []string
	RequiredEvidence []string
	BlockedBy        []string
	NextCommand      string
	Metadata         map[string]any
}

type OperationsReadiness struct {
	Status              string
	Mode                string
	Items               []OperationsReadinessItem
	ServiceStatus       LocalServiceStatus
	SupportBundle       SupportBundlePreview
	MigrationLedger     MigrationLedgerReadiness
	Capabilities        []string
	ForbiddenActions    []string
	SafetyFacts         map[string]bool
	TelemetryDefault    string
	ManagedOpsStatus    string
	SupportExportStatus string
	GeneratedAt         time.Time
}

const operationsSmokeProofFreshnessWindow = 24 * time.Hour

func (s Store) SupportBundlePreview(ctx context.Context, options SupportBundlePreviewOptions) (SupportBundlePreview, error) {
	options = normalizeSupportBundlePreviewOptions(options)
	backup, err := s.BackupManifest(ctx, BackupManifestOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return SupportBundlePreview{}, err
	}
	auditCoverage, err := s.AuditCoverage(ctx, AuditCoverageOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return SupportBundlePreview{}, err
	}
	return BuildSupportBundlePreview(backup, auditCoverage, options), nil
}

func (s Store) MigrationLedgerReadiness(ctx context.Context, options MigrationLedgerReadinessOptions) (MigrationLedgerReadiness, error) {
	options = normalizeMigrationLedgerReadinessOptions(options)
	migrations, err := migrate.List()
	if err != nil {
		return MigrationLedgerReadiness{}, err
	}
	schemaTable, err := s.tableExists(ctx, "schema_migrations")
	if err != nil {
		return MigrationLedgerReadiness{}, err
	}
	fullLedgerTable, err := s.tableExists(ctx, "migration_ledger")
	if err != nil {
		return MigrationLedgerReadiness{}, err
	}
	applied := map[string]bool{}
	if schemaTable {
		rows, err := s.pool.Query(ctx, `SELECT name FROM schema_migrations`)
		if err != nil {
			return MigrationLedgerReadiness{}, fmt.Errorf("list schema migrations: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return MigrationLedgerReadiness{}, fmt.Errorf("scan schema migration: %w", err)
			}
			applied[name] = true
		}
		if err := rows.Err(); err != nil {
			return MigrationLedgerReadiness{}, fmt.Errorf("list schema migrations: %w", err)
		}
	}
	ledgerPhases, err := s.migrationLedgerPhases(ctx, fullLedgerTable)
	if err != nil {
		return MigrationLedgerReadiness{}, err
	}
	return BuildMigrationLedgerReadiness(migrations, applied, ledgerPhases, schemaTable, fullLedgerTable, options), nil
}

func (s Store) OperationsReadiness(ctx context.Context, options OperationsReadinessOptions) (OperationsReadiness, error) {
	options = normalizeOperationsReadinessOptions(options)
	service, err := s.LocalServiceStatus(ctx, LocalServiceStatusOptions{
		APIBaseURL:      options.APIBaseURL,
		WebDashboardURL: options.WebDashboardURL,
		GeneratedAt:     options.GeneratedAt,
	})
	if err != nil {
		return OperationsReadiness{}, err
	}
	support, err := s.SupportBundlePreview(ctx, SupportBundlePreviewOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return OperationsReadiness{}, err
	}
	ledger, err := s.MigrationLedgerReadiness(ctx, MigrationLedgerReadinessOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return OperationsReadiness{}, err
	}
	if options.SmokeProofProjectScoped {
		if options.SmokeProofProject.ID != 0 {
			proof, err := s.LatestOperationsSmokeProofForProject(ctx, options.SmokeProofProject)
			if err != nil {
				return OperationsReadiness{}, err
			}
			options.SmokeProof = proof
		}
	} else {
		proof, err := s.LatestOperationsSmokeProof(ctx)
		if err != nil {
			return OperationsReadiness{}, err
		}
		options.SmokeProof = proof
	}
	return BuildOperationsReadiness(service, support, ledger, options), nil
}

func normalizeSupportBundlePreviewOptions(options SupportBundlePreviewOptions) SupportBundlePreviewOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func normalizeMigrationLedgerReadinessOptions(options MigrationLedgerReadinessOptions) MigrationLedgerReadinessOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func normalizeOperationsReadinessOptions(options OperationsReadinessOptions) OperationsReadinessOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildSupportBundlePreview(backup BackupManifest, auditCoverage AuditCoverage, options SupportBundlePreviewOptions) SupportBundlePreview {
	options = normalizeSupportBundlePreviewOptions(options)
	preview := SupportBundlePreview{
		Status:   "ready",
		Mode:     "metadata_only_support_bundle_preview",
		BundleID: "support-bundle-preview-v1",
		Scope:    "local_v1_metadata_only",
		Projects: supportBundleProjects(backup),
		IncludedMetadata: []string{
			"areaflow_version",
			"schema_migration_names",
			"postgres_table_counts",
			"project_keys",
			"workflow_version_labels",
			"run_task_attempt_ids",
			"artifact_metadata_hash_size_backend_relation",
			"health_readiness_doctor_summary",
			"release_readiness_final_gate_summary",
			"audit_coverage_summary",
			"redacted_log_index",
		},
		ExcludedSensitiveContent: []string{
			"secret_values",
			"api_token_values",
			"private_environment_values",
			"prompt_text",
			"user_file_contents",
			"raw_artifact_contents",
			"unredacted_stdout_stderr",
			"managed_project_file_copies",
			"areamatrix_execution_logs_progress_evidence_originals",
		},
		PathReferences: []SupportBundlePathReference{},
		Hashes: []SupportBundleHashReference{
			{
				Key:         "backup_manifest",
				Hash:        backup.ManifestHash,
				Source:      "backup manifest",
				Description: "stable metadata manifest hash",
			},
		},
		Capabilities: []string{
			"preview_support_bundle_metadata",
			"read_backup_manifest_metadata",
			"read_audit_coverage_summary",
			"report_redaction_policy",
		},
		ForbiddenActions: []string{
			"export_support_bundle",
			"read_secret_values",
			"read_prompt_text",
			"read_user_file_contents",
			"read_raw_artifact_contents",
			"read_unredacted_logs",
			"copy_project_files",
			"upload_bundle",
			"write_database",
			"write_project_files",
		},
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"metadata_only":                       true,
			"export_open":                         false,
			"approval_required":                   true,
			"audit_plan_required":                 true,
			"secret_values_included":              false,
			"api_token_values_included":           false,
			"prompt_text_included":                false,
			"user_file_contents_included":         false,
			"raw_artifact_contents_included":      false,
			"unredacted_logs_included":            false,
			"managed_project_files_copied":        false,
			"area_matrix_protected_paths_touched": false,
			"remote_upload_attempted":             false,
			"database_write_attempted":            false,
		},
		GeneratedAt: options.GeneratedAt,
	}
	for _, project := range backup.Projects {
		preview.PathReferences = append(preview.PathReferences, SupportBundlePathReference{
			Key:         "project:" + project.Project.Key,
			Kind:        "project_root_reference",
			URI:         project.Project.RootPath,
			ProjectKey:  project.Project.Key,
			Description: "metadata-only project root reference; content is not copied",
		})
	}
	preview.Hashes = append(preview.Hashes, SupportBundleHashReference{
		Key:         "audit_coverage",
		Hash:        "",
		Source:      "audit coverage",
		Description: fmt.Sprintf("audit coverage status=%s total_audit_events=%d", auditCoverage.Status, auditCoverage.TotalAuditEvents),
	})
	return preview
}

func BuildMigrationLedgerReadiness(migrations []migrate.Migration, applied map[string]bool, ledgerPhases map[string]map[string]MigrationLedgerPhase, schemaTable bool, fullLedgerTable bool, options MigrationLedgerReadinessOptions) MigrationLedgerReadiness {
	options = normalizeMigrationLedgerReadinessOptions(options)
	readiness := MigrationLedgerReadiness{
		Status:                       "ready",
		Mode:                         "read_only_migration_ledger_readiness",
		Entries:                      []MigrationLedgerEntry{},
		SchemaMigrationsTablePresent: schemaTable,
		FullLedgerTablePresent:       fullLedgerTable,
		Capabilities: []string{
			"read_embedded_migrations",
			"read_schema_migration_names",
			"report_migration_ledger_gap",
		},
		ForbiddenActions: []string{
			"apply_migration",
			"create_migration",
			"write_schema_migrations",
			"write_migration_ledger",
			"rollback_database",
			"write_project_files",
			"touch_areamatrix_protected_paths",
		},
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"migration_apply_attempted":           false,
			"database_write_attempted":            false,
			"destructive_rollback_attempted":      false,
			"project_write_attempted":             false,
			"area_matrix_protected_paths_touched": false,
		},
		GeneratedAt: options.GeneratedAt,
	}
	allLedgerPhasesReady := fullLedgerTable
	for _, migration := range migrations {
		phaseMap := ledgerPhases[migration.Name]
		entry := MigrationLedgerEntry{
			Name:    migration.Name,
			Applied: applied[migration.Name],
			Status:  "ready",
			Phases:  migrationLedgerPhasesForEntry(phaseMap),
			RequiredEvidence: []string{
				"embedded migration exists",
				"schema_migrations records applied migration",
				"migration_ledger records preflight/apply/verify/remediation phases",
			},
			Metadata: map[string]any{
				"embedded":           true,
				"ledger_phase_count": len(phaseMap),
			},
		}
		if !schemaTable {
			entry.Status = "blocked"
			entry.RequiredEvidence = append(entry.RequiredEvidence, "schema_migrations table exists")
			allLedgerPhasesReady = false
		} else if !entry.Applied {
			entry.Status = "blocked"
			entry.RequiredEvidence = append(entry.RequiredEvidence, "migration applied")
			readiness.PendingCount++
			allLedgerPhasesReady = false
		} else {
			readiness.AppliedCount++
			if !fullLedgerTable {
				allLedgerPhasesReady = false
			} else {
				missing := missingMigrationLedgerPhases(phaseMap)
				notReady := notReadyMigrationLedgerPhases(phaseMap)
				if len(missing) > 0 || len(notReady) > 0 {
					entry.Status = "needs_attention"
					entry.Metadata["missing_ledger_phases"] = missing
					entry.Metadata["not_ready_ledger_phases"] = notReady
					entry.RequiredEvidence = append(entry.RequiredEvidence, "all migration_ledger phases are ready")
					allLedgerPhasesReady = false
				}
			}
		}
		readiness.Entries = append(readiness.Entries, entry)
		readiness.Status = combineOperationsStatus(readiness.Status, entry.Status)
	}
	if !fullLedgerTable && readiness.Status == "ready" {
		readiness.Status = "needs_attention"
	}
	readiness.PreflightApplyVerifyRemediationReady = allLedgerPhasesReady && readiness.Status == "ready"
	return readiness
}

func BuildOperationsReadiness(service LocalServiceStatus, support SupportBundlePreview, ledger MigrationLedgerReadiness, options OperationsReadinessOptions) OperationsReadiness {
	options = normalizeOperationsReadinessOptions(options)
	readiness := OperationsReadiness{
		Status:              "ready",
		Mode:                "read_only_operations_readiness",
		Items:               []OperationsReadinessItem{},
		ServiceStatus:       service,
		SupportBundle:       support,
		MigrationLedger:     ledger,
		TelemetryDefault:    "local_only",
		ManagedOpsStatus:    "deferred_v1x",
		SupportExportStatus: "deferred_v1x",
		Capabilities: []string{
			"read_service_status",
			"preview_support_bundle_metadata",
			"read_migration_ledger_readiness",
			"report_local_only_telemetry",
			"report_managed_ops_deferred",
		},
		ForbiddenActions: []string{
			"start_service_process",
			"stop_service_process",
			"export_support_bundle",
			"upload_telemetry",
			"run_managed_upgrade",
			"rollback_database",
			"write_project_files",
			"touch_areamatrix_protected_paths",
		},
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"support_bundle_exported":             false,
			"support_bundle_metadata_only":        true,
			"remote_telemetry_enabled":            false,
			"managed_upgrade_attempted":           false,
			"destructive_rollback_attempted":      false,
			"service_process_control_attempted":   false,
			"database_write_attempted":            false,
			"project_write_attempted":             false,
			"area_matrix_protected_paths_touched": false,
		},
		GeneratedAt: options.GeneratedAt,
	}
	readiness.addItem(operationsLocalBootstrapSmokeItem(options.SmokeProof, options.GeneratedAt))
	readiness.addItem(operationsServiceStatusItem(service))
	readiness.addItem(operationsSupportBundleItem(support))
	readiness.addItem(operationsMigrationLedgerItem(ledger))
	readiness.addItem(operationsTelemetryItem())
	readiness.addItem(operationsManagedOpsDeferredItem())
	return readiness
}

func (r *OperationsReadiness) addItem(item OperationsReadinessItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	r.Items = append(r.Items, item)
	r.Status = combineOperationsStatus(r.Status, item.Status)
}

func combineOperationsStatus(current string, next string) string {
	rank := map[string]int{
		"ready":           0,
		"deferred":        0,
		"needs_attention": 1,
		"blocked":         2,
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func operationsLocalBootstrapSmokeItem(proof OperationsSmokeProof, generatedAt time.Time) OperationsReadinessItem {
	metadata := map[string]any{
		"completion_audit_runs_smoke": false,
		"evidence_recorded":           false,
		"smoke_proof_max_age_seconds": int64(operationsSmokeProofFreshnessWindow.Seconds()),
	}
	freshnessBlockers := []string{}
	if proof.EventID != 0 {
		freshnessBlockers = operationsSmokeProofFreshnessBlockers(proof, generatedAt)
		metadata["evidence_recorded"] = true
		metadata["latest_smoke_proof_key"] = proof.ProofKey
		metadata["latest_smoke_proof_event_id"] = proof.EventID
		metadata["latest_smoke_proof_project_key"] = proof.Project.Key
		metadata["latest_smoke_proof_status"] = proof.EvidenceStatus
		metadata["latest_smoke_proof_summary"] = metadataString(proof.Metadata, "summary")
		metadata["latest_smoke_proof_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["latest_smoke_proof_traceable_evidence"] = proofMetadataHasTraceableEvidence(proof.Metadata)
		metadata["record_command_runs_smoke"] = proof.RecordCommandRunsSmoke
		metadata["recorded_at"] = proof.CreatedAt
		metadata["latest_smoke_proof_fresh"] = len(freshnessBlockers) == 0
		metadata["latest_smoke_proof_freshness_status"] = "fresh"
		if len(freshnessBlockers) > 0 {
			metadata["latest_smoke_proof_freshness_status"] = "stale"
			metadata["latest_smoke_proof_freshness_blockers"] = freshnessBlockers
		}
		if !proof.CreatedAt.IsZero() {
			age := generatedAt.Sub(proof.CreatedAt)
			if age < 0 {
				age = 0
			}
			metadata["latest_smoke_proof_age_seconds"] = int64(age.Seconds())
		}
	}
	if proof.EventID != 0 && proof.EvidenceStatus == "pass" &&
		proofMetadataHasTraceableEvidence(proof.Metadata) &&
		len(freshnessBlockers) == 0 &&
		!proof.RecordCommandRunsSmoke &&
		!proof.ProjectWriteAttempted &&
		!proof.ExecutionWriteAttempted &&
		!proof.EngineCallAttempted &&
		!proof.ServiceProcessControlAttempted &&
		!proof.SupportBundleExported &&
		!proof.MigrationApplyAttempted &&
		!proof.RemoteTelemetryEnabled &&
		!proof.AreaMatrixProtectedPathsTouched {
		evidenceRefs := []string{
			"docs/operations/deployment.md",
			"docs/history/v1.0/evidence/bootstrap-smoke-evidence.md",
		}
		if uri := metadataString(proof.Metadata, "evidence_uri"); uri != "" {
			evidenceRefs = append(evidenceRefs, uri)
		}
		return OperationsReadinessItem{
			Key:          "install_migrate_start_register_smoke",
			Category:     "bootstrap",
			Status:       "ready",
			Message:      "fresh operations smoke proof has been recorded; completion audit can consume this proof without running smoke",
			EvidenceRefs: evidenceRefs,
			RequiredEvidence: []string{
				"empty PostgreSQL database migrated",
				"server starts",
				"project add/import succeeds",
				"service status and doctor are queryable",
				"smoke cleanup result recorded",
			},
			NextCommand: "areaflow ops readiness --json",
			Metadata:    metadata,
		}
	}
	if proof.EventID != 0 && proof.EvidenceStatus == "pass" && !proofMetadataHasTraceableEvidence(proof.Metadata) {
		return OperationsReadinessItem{
			Key:      "install_migrate_start_register_smoke",
			Category: "bootstrap",
			Status:   "needs_attention",
			Message:  "latest operations smoke proof lacks traceable evidence",
			EvidenceRefs: []string{
				"docs/operations/deployment.md",
				"docs/history/v1.0/evidence/bootstrap-smoke-evidence.md",
			},
			RequiredEvidence: []string{
				"empty PostgreSQL database migrated",
				"server starts",
				"project add/import succeeds",
				"service status and doctor are queryable",
				"smoke cleanup result recorded",
			},
			BlockedBy:   []string{"operations_smoke_proof_evidence_missing"},
			NextCommand: "areaflow ops smoke-proof record <project> --key local_ops_smoke --status pass --summary <text> --evidence-uri <uri> --json",
			Metadata:    metadata,
		}
	}
	if proof.EventID != 0 && proof.EvidenceStatus == "pass" && len(freshnessBlockers) > 0 {
		return OperationsReadinessItem{
			Key:      "install_migrate_start_register_smoke",
			Category: "bootstrap",
			Status:   "needs_attention",
			Message:  "latest operations smoke proof is stale and must be refreshed before completion audit can pass",
			EvidenceRefs: []string{
				"docs/operations/deployment.md",
				"docs/history/v1.0/evidence/bootstrap-smoke-evidence.md",
			},
			RequiredEvidence: []string{
				"empty PostgreSQL database migrated",
				"server starts",
				"project add/import succeeds",
				"service status and doctor are queryable",
				"smoke cleanup result recorded within freshness window",
			},
			BlockedBy:   freshnessBlockers,
			NextCommand: "areaflow ops smoke-proof record <project> --key local_ops_smoke --status pass --summary <text> --evidence-uri <uri> --json",
			Metadata:    metadata,
		}
	}
	if proof.EventID != 0 && proof.EvidenceStatus != "pass" {
		metadata["blocked_proof_event_id"] = proof.EventID
		return OperationsReadinessItem{
			Key:      "install_migrate_start_register_smoke",
			Category: "bootstrap",
			Status:   "blocked",
			Message:  "latest operations smoke proof is not pass",
			EvidenceRefs: []string{
				"docs/operations/deployment.md",
				"docs/history/v1.0/evidence/bootstrap-smoke-evidence.md",
			},
			RequiredEvidence: []string{
				"empty PostgreSQL database migrated",
				"server starts",
				"project add/import succeeds",
				"service status and doctor are queryable",
				"smoke cleanup result recorded",
			},
			BlockedBy:   []string{"operations_smoke_proof_not_pass"},
			NextCommand: "areaflow ops smoke-proof record <project> --key local_ops_smoke --status pass --json",
			Metadata:    metadata,
		}
	}
	return OperationsReadinessItem{
		Key:      "install_migrate_start_register_smoke",
		Category: "bootstrap",
		Status:   "needs_attention",
		Message:  "fresh install/migrate/start/register smoke evidence must be supplied before completion audit can pass",
		EvidenceRefs: []string{
			"docs/operations/deployment.md",
			"docs/history/v1.0/evidence/bootstrap-smoke-evidence.md",
		},
		RequiredEvidence: []string{
			"empty PostgreSQL database migrated",
			"server starts",
			"project add/import succeeds",
			"service status and doctor are queryable",
			"smoke cleanup result recorded",
		},
		BlockedBy:   []string{"fresh_local_ops_smoke_missing"},
		NextCommand: "AREAFLOW_DATABASE_URL=... ./scripts/smoke-local.sh",
		Metadata:    metadata,
	}
}

func operationsSmokeProofFreshnessBlockers(proof OperationsSmokeProof, generatedAt time.Time) []string {
	if proof.CreatedAt.IsZero() {
		return []string{"operations_smoke_proof_recorded_at_missing"}
	}
	age := generatedAt.Sub(proof.CreatedAt)
	if age < 0 {
		age = 0
	}
	if age > operationsSmokeProofFreshnessWindow {
		return []string{"operations_smoke_proof_stale"}
	}
	return nil
}

func operationsServiceStatusItem(status LocalServiceStatus) OperationsReadinessItem {
	itemStatus := "ready"
	if status.Status == "warn" {
		itemStatus = "needs_attention"
	}
	if status.Status == "blocked" {
		itemStatus = "blocked"
	}
	return OperationsReadinessItem{
		Key:      "local_service_status",
		Category: "service",
		Status:   itemStatus,
		Message:  "local service status is queryable and remains observation-only",
		EvidenceRefs: []string{
			"GET /api/v1/service/status",
			"areaflow service status --json",
		},
		RequiredEvidence: []string{
			"service status does not control processes",
			"service status does not maintain second state source",
		},
		NextCommand: "areaflow service status --json",
		Metadata: map[string]any{
			"service_status":     status.Status,
			"database_status":    status.Database.Status,
			"worker_pool_status": status.WorkerPool.Status,
		},
	}
}

func operationsSupportBundleItem(preview SupportBundlePreview) OperationsReadinessItem {
	itemStatus := "ready"
	if preview.Status != "ready" {
		itemStatus = preview.Status
	}
	blockedBy := operationsSupportBundleSafetyBlockers(preview)
	if len(blockedBy) > 0 {
		itemStatus = "blocked"
	}
	message := "support bundle preview is metadata-only and excludes sensitive content"
	if len(blockedBy) > 0 {
		message = "support bundle preview is unsafe or missing required redaction guardrails"
	}
	return OperationsReadinessItem{
		Key:      "metadata_only_support_bundle_preview",
		Category: "support",
		Status:   itemStatus,
		Message:  message,
		EvidenceRefs: []string{
			"GET /api/v1/ops/support-bundle-preview",
			"areaflow support bundle-preview --json",
		},
		RequiredEvidence: []string{
			"metadata-only included list",
			"sensitive content excluded list",
			"export_open=false",
			"redaction policy summary",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow support bundle-preview --json",
		Metadata: map[string]any{
			"support_bundle_status":                   preview.Status,
			"support_bundle_mode":                     preview.Mode,
			"support_bundle_scope":                    preview.Scope,
			"support_bundle_included_metadata_count":  len(preview.IncludedMetadata),
			"support_bundle_excluded_sensitive_count": len(preview.ExcludedSensitiveContent),
			"support_bundle_path_reference_count":     len(preview.PathReferences),
			"support_bundle_hash_count":               len(preview.Hashes),
			"support_bundle_forbidden_action_count":   len(preview.ForbiddenActions),
			"export_open":                             preview.SafetyFacts["export_open"],
			"metadata_only":                           preview.SafetyFacts["metadata_only"],
			"read_only":                               preview.SafetyFacts["read_only"],
			"secret_values_included":                  preview.SafetyFacts["secret_values_included"],
			"api_token_values_included":               preview.SafetyFacts["api_token_values_included"],
			"prompt_text_included":                    preview.SafetyFacts["prompt_text_included"],
			"user_file_contents_included":             preview.SafetyFacts["user_file_contents_included"],
			"raw_artifact_contents_included":          preview.SafetyFacts["raw_artifact_contents_included"],
			"unredacted_logs_included":                preview.SafetyFacts["unredacted_logs_included"],
			"managed_project_files_copied":            preview.SafetyFacts["managed_project_files_copied"],
			"remote_upload_attempted":                 preview.SafetyFacts["remote_upload_attempted"],
			"database_write_attempted":                preview.SafetyFacts["database_write_attempted"],
			"area_matrix_protected_paths_touched":     preview.SafetyFacts["area_matrix_protected_paths_touched"],
		},
	}
}

func operationsSupportBundleSafetyBlockers(preview SupportBundlePreview) []string {
	blockers := []string{}
	requiredTrueSafetyFacts := map[string]string{
		"read_only":     "support_bundle_not_read_only",
		"metadata_only": "support_bundle_not_metadata_only",
	}
	for fact, blocker := range requiredTrueSafetyFacts {
		if !preview.SafetyFacts[fact] {
			blockers = append(blockers, blocker)
		}
	}
	requiredFalseSafetyFacts := map[string]string{
		"export_open":                         "support_bundle_export_open",
		"secret_values_included":              "support_bundle_secret_values_included",
		"api_token_values_included":           "support_bundle_api_token_values_included",
		"prompt_text_included":                "support_bundle_prompt_text_included",
		"user_file_contents_included":         "support_bundle_user_file_contents_included",
		"raw_artifact_contents_included":      "support_bundle_raw_artifact_contents_included",
		"unredacted_logs_included":            "support_bundle_unredacted_logs_included",
		"managed_project_files_copied":        "support_bundle_managed_project_files_copied",
		"area_matrix_protected_paths_touched": "support_bundle_areamatrix_protected_paths_touched",
		"remote_upload_attempted":             "support_bundle_remote_upload_attempted",
		"database_write_attempted":            "support_bundle_database_write_attempted",
	}
	for fact, blocker := range requiredFalseSafetyFacts {
		if preview.SafetyFacts[fact] {
			blockers = append(blockers, blocker)
		}
	}
	for _, excluded := range []string{
		"secret_values",
		"api_token_values",
		"private_environment_values",
		"prompt_text",
		"user_file_contents",
		"raw_artifact_contents",
		"unredacted_stdout_stderr",
		"managed_project_file_copies",
		"areamatrix_execution_logs_progress_evidence_originals",
	} {
		if !containsOperationsString(preview.ExcludedSensitiveContent, excluded) {
			blockers = append(blockers, "support_bundle_exclusion_missing:"+excluded)
		}
	}
	for _, forbidden := range []string{
		"export_support_bundle",
		"read_secret_values",
		"read_prompt_text",
		"read_user_file_contents",
		"read_raw_artifact_contents",
		"read_unredacted_logs",
		"copy_project_files",
		"upload_bundle",
		"write_database",
		"write_project_files",
	} {
		if !containsOperationsString(preview.ForbiddenActions, forbidden) {
			blockers = append(blockers, "support_bundle_forbidden_action_missing:"+forbidden)
		}
	}
	for _, included := range preview.IncludedMetadata {
		switch included {
		case "secret_values", "api_token_values", "private_environment_values", "prompt_text", "user_file_contents", "raw_artifact_contents", "unredacted_stdout_stderr", "managed_project_file_copies":
			blockers = append(blockers, "support_bundle_sensitive_metadata_included:"+included)
		}
	}
	return uniqueStrings(blockers)
}

func containsOperationsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func operationsMigrationLedgerItem(ledger MigrationLedgerReadiness) OperationsReadinessItem {
	status := ledger.Status
	blockedBy := []string{}
	if !ledger.FullLedgerTablePresent {
		blockedBy = append(blockedBy, "full_migration_ledger_missing")
	}
	if ledger.PendingCount > 0 {
		blockedBy = append(blockedBy, "pending_schema_migrations")
	}
	return OperationsReadinessItem{
		Key:      "migration_ledger_readiness",
		Category: "migration",
		Status:   status,
		Message:  "migration ledger records applied migrations with preflight/apply/verify/remediation proof",
		EvidenceRefs: []string{
			"GET /api/v1/ops/migration-ledger-readiness",
			"areaflow ops migration-ledger-readiness --json",
		},
		RequiredEvidence: []string{
			"all embedded migrations applied",
			"preflight/apply/verify/remediation ledger exists",
			"rollback/remediation wording recorded",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow ops migration-ledger-readiness --json",
		Metadata: map[string]any{
			"applied_count":                   ledger.AppliedCount,
			"pending_count":                   ledger.PendingCount,
			"schema_migrations_table_present": ledger.SchemaMigrationsTablePresent,
			"full_ledger_table_present":       ledger.FullLedgerTablePresent,
		},
	}
}

func operationsTelemetryItem() OperationsReadinessItem {
	return OperationsReadinessItem{
		Key:      "local_only_telemetry_default",
		Category: "telemetry",
		Status:   "ready",
		Message:  "telemetry remains local-only by default",
		EvidenceRefs: []string{
			"docs/operations/deployment.md",
			"docs/history/v1.0/contracts/v1.0-stable-platform-contract.md",
		},
		RequiredEvidence: []string{
			"remote telemetry disabled by default",
			"any future remote telemetry requires opt-in and audit",
		},
		Metadata: map[string]any{
			"remote_telemetry_enabled": false,
		},
	}
}

func operationsManagedOpsDeferredItem() OperationsReadinessItem {
	return OperationsReadinessItem{
		Key:      "managed_ops_deferred",
		Category: "managed_ops",
		Status:   "deferred",
		Message:  "remote ops, managed upgrade, destructive rollback and full support export remain v1.x",
		EvidenceRefs: []string{
			"docs/operations/deployment.md",
			"proposals/high-risk-apply.md",
		},
		RequiredEvidence: []string{
			"managed ops R4 apply packet before opening",
			"auth/team scope",
			"backup/preimage",
			"approval/audit",
		},
		Metadata: map[string]any{
			"remote_ops_open":          false,
			"managed_upgrade_open":     false,
			"full_support_export_open": false,
		},
	}
}

func supportBundleProjects(backup BackupManifest) []Record {
	projects := make([]Record, 0, len(backup.Projects))
	for _, project := range backup.Projects {
		projects = append(projects, project.Project)
	}
	return projects
}

func (s Store) migrationLedgerPhases(ctx context.Context, fullLedgerTable bool) (map[string]map[string]MigrationLedgerPhase, error) {
	phases := map[string]map[string]MigrationLedgerPhase{}
	if !fullLedgerTable {
		return phases, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT migration_name, phase, status, message, evidence_json, COALESCE(remediation, '')
FROM migration_ledger
ORDER BY migration_name, phase`)
	if err != nil {
		return nil, fmt.Errorf("list migration ledger phases: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var migrationName string
		var phase MigrationLedgerPhase
		var metadataRaw []byte
		if err := rows.Scan(&migrationName, &phase.Phase, &phase.Status, &phase.Message, &metadataRaw, &phase.Remediation); err != nil {
			return nil, fmt.Errorf("scan migration ledger phase: %w", err)
		}
		phase.Metadata = map[string]any{}
		if len(metadataRaw) > 0 {
			if err := json.Unmarshal(metadataRaw, &phase.Metadata); err != nil {
				return nil, fmt.Errorf("parse migration ledger metadata for %s/%s: %w", migrationName, phase.Phase, err)
			}
		}
		if phases[migrationName] == nil {
			phases[migrationName] = map[string]MigrationLedgerPhase{}
		}
		phases[migrationName][phase.Phase] = phase
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list migration ledger phases: %w", err)
	}
	return phases, nil
}

func migrationLedgerPhasesForEntry(phaseMap map[string]MigrationLedgerPhase) []MigrationLedgerPhase {
	ordered := []MigrationLedgerPhase{}
	for _, phase := range requiredMigrationLedgerPhases() {
		if entry, ok := phaseMap[phase]; ok {
			ordered = append(ordered, entry)
		}
	}
	return ordered
}

func missingMigrationLedgerPhases(phaseMap map[string]MigrationLedgerPhase) []string {
	missing := []string{}
	for _, phase := range requiredMigrationLedgerPhases() {
		if _, ok := phaseMap[phase]; !ok {
			missing = append(missing, phase)
		}
	}
	return missing
}

func notReadyMigrationLedgerPhases(phaseMap map[string]MigrationLedgerPhase) []string {
	notReady := []string{}
	for _, phase := range requiredMigrationLedgerPhases() {
		entry, ok := phaseMap[phase]
		if !ok {
			continue
		}
		if entry.Status != "ready" && entry.Status != "pass" {
			notReady = append(notReady, phase)
		}
	}
	return notReady
}

func requiredMigrationLedgerPhases() []string {
	return []string{"preflight", "apply", "verify", "remediation"}
}

func (s Store) tableExists(ctx context.Context, table string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)`, table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", table, err)
	}
	return exists, nil
}
