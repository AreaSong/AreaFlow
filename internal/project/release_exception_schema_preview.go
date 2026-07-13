package project

import (
	"context"
	"time"
)

type ReleaseExceptionSchemaPreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseExceptionSchemaColumn struct {
	Name     string
	Type     string
	Nullable bool
	Purpose  string
}

type ReleaseExceptionSchemaIndex struct {
	Name    string
	Columns []string
	Unique  bool
	Purpose string
}

type ReleaseExceptionSchemaForeignKey struct {
	Column           string
	ReferencesTable  string
	ReferencesColumn string
	OnDelete         string
}

type ReleaseExceptionSchemaTable struct {
	Name        string
	Purpose     string
	Columns     []ReleaseExceptionSchemaColumn
	Indexes     []ReleaseExceptionSchemaIndex
	ForeignKeys []ReleaseExceptionSchemaForeignKey
}

type ReleaseExceptionMigrationStep struct {
	Order       int
	Action      string
	Description string
	SQLPreview  string
}

type ReleaseExceptionSchemaPreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	RecordPreview    ReleaseExceptionRecordPreview
	Tables           []ReleaseExceptionSchemaTable
	ApplySteps       []ReleaseExceptionMigrationStep
	RollbackSteps    []ReleaseExceptionMigrationStep
	AuditActions     []string
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseExceptionSchemaPreview(ctx context.Context, options ReleaseExceptionSchemaPreviewOptions) (ReleaseExceptionSchemaPreview, error) {
	options = normalizeReleaseExceptionSchemaPreviewOptions(options)
	recordPreview, err := s.ReleaseExceptionRecordPreview(ctx, ReleaseExceptionRecordPreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseExceptionSchemaPreview{}, err
	}
	return BuildReleaseExceptionSchemaPreview(recordPreview, options), nil
}

func normalizeReleaseExceptionSchemaPreviewOptions(options ReleaseExceptionSchemaPreviewOptions) ReleaseExceptionSchemaPreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseExceptionSchemaPreview(recordPreview ReleaseExceptionRecordPreview, options ReleaseExceptionSchemaPreviewOptions) ReleaseExceptionSchemaPreview {
	options = normalizeReleaseExceptionSchemaPreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, recordPreview.ProjectKey, recordPreview.Doctor.ProjectKey)
	preview := ReleaseExceptionSchemaPreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "needs_approval",
		Mode:             "read_only_release_exception_schema_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		RecordPreview:    recordPreview,
		Tables: []ReleaseExceptionSchemaTable{
			releaseExceptionsTablePreview(),
		},
		ApplySteps: []ReleaseExceptionMigrationStep{
			{Order: 1, Action: "create_table", Description: "create release_exceptions table", SQLPreview: "CREATE TABLE IF NOT EXISTS release_exceptions (...)"},
			{Order: 2, Action: "create_index", Description: "create lookup indexes for active release exceptions", SQLPreview: "CREATE INDEX IF NOT EXISTS release_exceptions_project_status_idx ON release_exceptions (project_id, status, created_at DESC)"},
			{Order: 3, Action: "audit_contract", Description: "reserve release.exception request/approve/revoke audit actions", SQLPreview: "audit_events.action IN ('release.exception.request','release.exception.approve','release.exception.revoke')"},
		},
		RollbackSteps: []ReleaseExceptionMigrationStep{
			{Order: 1, Action: "disable_writes", Description: "disable release exception write endpoints before rollback", SQLPreview: "set release exception write capability to disabled"},
			{Order: 2, Action: "export_records", Description: "export release exception records and related audit event ids", SQLPreview: "SELECT * FROM release_exceptions ORDER BY created_at, id"},
			{Order: 3, Action: "drop_table", Description: "drop release_exceptions only after exported records are archived", SQLPreview: "DROP TABLE IF EXISTS release_exceptions"},
		},
		AuditActions: []string{
			"release.exception.request",
			"release.exception.approve",
			"release.exception.revoke",
		},
		Capabilities: []string{
			"read_release_exception_record_preview",
			"preview_release_exception_schema",
			"preview_release_exception_migration_plan",
			"preview_release_exception_rollback_plan",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"create_migration_file",
			"run_migration",
			"insert_exception_record",
			"insert_audit_event",
			"mark_gap_accepted",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	if recordPreview.Status == "blocked" {
		preview.Status = "blocked"
	}
	return preview
}

func releaseExceptionsTablePreview() ReleaseExceptionSchemaTable {
	return ReleaseExceptionSchemaTable{
		Name:    "release_exceptions",
		Purpose: "stores explicit release exception records after approval is enabled",
		Columns: []ReleaseExceptionSchemaColumn{
			{Name: "id", Type: "BIGSERIAL", Nullable: false, Purpose: "primary key"},
			{Name: "project_id", Type: "BIGINT", Nullable: true, Purpose: "optional project scope"},
			{Name: "exception_key", Type: "TEXT", Nullable: false, Purpose: "stable exception identifier"},
			{Name: "source_gate_item", Type: "TEXT", Nullable: false, Purpose: "release acceptance gate item"},
			{Name: "source_decision", Type: "TEXT", Nullable: false, Purpose: "source decision status"},
			{Name: "acceptance_type", Type: "TEXT", Nullable: false, Purpose: "metadata_only_history, future_only_gap, archive_exception, or none"},
			{Name: "status", Type: "TEXT", Nullable: false, Purpose: "requested, approved, rejected, revoked, expired"},
			{Name: "owner", Type: "TEXT", Nullable: false, Purpose: "accountable owner"},
			{Name: "reason", Type: "TEXT", Nullable: false, Purpose: "why the exception is needed"},
			{Name: "required_evidence", Type: "JSONB", Nullable: false, Purpose: "evidence required before approval"},
			{Name: "audit_actions", Type: "JSONB", Nullable: false, Purpose: "audit actions associated with lifecycle"},
			{Name: "rollback_plan", Type: "TEXT", Nullable: false, Purpose: "how to revoke or recover from this exception"},
			{Name: "review_required", Type: "BOOLEAN", Nullable: false, Purpose: "whether explicit review is required"},
			{Name: "review_at", Type: "TIMESTAMPTZ", Nullable: true, Purpose: "scheduled review time"},
			{Name: "expires_at", Type: "TIMESTAMPTZ", Nullable: true, Purpose: "optional expiry time"},
			{Name: "requested_by", Type: "TEXT", Nullable: false, Purpose: "actor that requested the record"},
			{Name: "approved_by", Type: "TEXT", Nullable: true, Purpose: "actor that approved the record"},
			{Name: "revoked_by", Type: "TEXT", Nullable: true, Purpose: "actor that revoked the record"},
			{Name: "created_by_actor_id", Type: "BIGINT", Nullable: true, Purpose: "optional normalized requesting actor"},
			{Name: "approved_by_actor_id", Type: "BIGINT", Nullable: true, Purpose: "optional normalized approving actor"},
			{Name: "decision_reason", Type: "TEXT", Nullable: true, Purpose: "approval or revocation reason"},
			{Name: "audit_event_id", Type: "BIGINT", Nullable: true, Purpose: "audit event for latest lifecycle transition"},
			{Name: "metadata", Type: "JSONB", Nullable: false, Purpose: "structured extension metadata"},
			{Name: "created_at", Type: "TIMESTAMPTZ", Nullable: false, Purpose: "creation time"},
			{Name: "updated_at", Type: "TIMESTAMPTZ", Nullable: false, Purpose: "last update time"},
			{Name: "approved_at", Type: "TIMESTAMPTZ", Nullable: true, Purpose: "approval time"},
			{Name: "revoked_at", Type: "TIMESTAMPTZ", Nullable: true, Purpose: "revocation time"},
		},
		Indexes: []ReleaseExceptionSchemaIndex{
			{Name: "release_exceptions_key_idx", Columns: []string{"exception_key"}, Unique: true, Purpose: "stable idempotency and lookup"},
			{Name: "release_exceptions_project_status_idx", Columns: []string{"project_id", "status", "created_at"}, Unique: false, Purpose: "project-scoped active exception queries"},
			{Name: "release_exceptions_acceptance_type_idx", Columns: []string{"acceptance_type", "status"}, Unique: false, Purpose: "release governance reporting"},
		},
		ForeignKeys: []ReleaseExceptionSchemaForeignKey{
			{Column: "project_id", ReferencesTable: "projects", ReferencesColumn: "id", OnDelete: "CASCADE"},
			{Column: "created_by_actor_id", ReferencesTable: "actors", ReferencesColumn: "id", OnDelete: "SET NULL"},
			{Column: "approved_by_actor_id", ReferencesTable: "actors", ReferencesColumn: "id", OnDelete: "SET NULL"},
			{Column: "audit_event_id", ReferencesTable: "audit_events", ReferencesColumn: "id", OnDelete: "SET NULL"},
		},
	}
}
