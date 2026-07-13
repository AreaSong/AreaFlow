package project

import (
	"context"
	"time"
)

type ReleaseExceptionMigrationApprovalGateOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseExceptionMigrationApprovalGateItem struct {
	Key              string
	Category         string
	Status           string
	ApprovalStatus   string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseExceptionMigrationApprovalGate struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	SchemaPreview    ReleaseExceptionSchemaPreview
	Items            []ReleaseExceptionMigrationApprovalGateItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseExceptionMigrationApprovalGate(ctx context.Context, options ReleaseExceptionMigrationApprovalGateOptions) (ReleaseExceptionMigrationApprovalGate, error) {
	options = normalizeReleaseExceptionMigrationApprovalGateOptions(options)
	schemaPreview, err := s.ReleaseExceptionSchemaPreview(ctx, ReleaseExceptionSchemaPreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseExceptionMigrationApprovalGate{}, err
	}
	return BuildReleaseExceptionMigrationApprovalGate(schemaPreview, options), nil
}

func normalizeReleaseExceptionMigrationApprovalGateOptions(options ReleaseExceptionMigrationApprovalGateOptions) ReleaseExceptionMigrationApprovalGateOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseExceptionMigrationApprovalGate(schemaPreview ReleaseExceptionSchemaPreview, options ReleaseExceptionMigrationApprovalGateOptions) ReleaseExceptionMigrationApprovalGate {
	options = normalizeReleaseExceptionMigrationApprovalGateOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, schemaPreview.ProjectKey, schemaPreview.RecordPreview.ProjectKey)
	gate := ReleaseExceptionMigrationApprovalGate{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "pass",
		Mode:             "read_only_release_exception_migration_approval_gate",
		Scope:            scope,
		ProjectKey:       projectKey,
		SchemaPreview:    schemaPreview,
		Items:            []ReleaseExceptionMigrationApprovalGateItem{},
		Capabilities: []string{
			"read_release_exception_schema_preview",
			"evaluate_release_exception_migration_approval_gate",
			"report_migration_approval_blockers",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"create_migration_file",
			"run_migration",
			"insert_exception_record",
			"insert_audit_event",
			"create_approval",
			"approve_migration",
			"mark_gap_accepted",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	gate.addItem(releaseExceptionMigrationApprovalItem(schemaPreview))
	return gate
}

func (g *ReleaseExceptionMigrationApprovalGate) addItem(item ReleaseExceptionMigrationApprovalGateItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	g.Items = append(g.Items, item)
	if item.Status == "blocked" {
		g.Status = "blocked"
	}
}

func releaseExceptionMigrationApprovalItem(schemaPreview ReleaseExceptionSchemaPreview) ReleaseExceptionMigrationApprovalGateItem {
	metadata := map[string]any{
		"schema_preview_status": schemaPreview.Status,
		"table_count":           len(schemaPreview.Tables),
		"apply_step_count":      len(schemaPreview.ApplySteps),
		"rollback_step_count":   len(schemaPreview.RollbackSteps),
		"audit_action_count":    len(schemaPreview.AuditActions),
		"risk_level":            "R4 migration_security",
		"migration_writable":    false,
	}
	switch schemaPreview.Status {
	case "blocked":
		return ReleaseExceptionMigrationApprovalGateItem{
			Key:              "migration_approval:release_exception_schema",
			Category:         "migration",
			Status:           "blocked",
			ApprovalStatus:   "blocked",
			Message:          "release exception schema preview is blocked; migration approval cannot be requested",
			Owner:            "release_owner",
			RequiredEvidence: []string{"release exception schema preview returns needs_approval with a reviewed apply and rollback plan"},
			NextCommand:      "areaflow release exception-schema-preview --json",
			Metadata:         metadata,
		}
	case "needs_approval":
		return ReleaseExceptionMigrationApprovalGateItem{
			Key:            "migration_approval:release_exception_schema",
			Category:       "migration",
			Status:         "blocked",
			ApprovalStatus: "needs_approval",
			Message:        "explicit migration approval is required before creating or running the release exception migration",
			Owner:          "release_owner",
			RequiredEvidence: []string{
				"approved migration approval record for release exception schema",
				"reviewed release_exceptions table, indexes, and foreign keys",
				"reviewed apply steps and rollback steps",
				"release exception schema preview remains unchanged since approval review",
			},
			NextCommand: "areaflow release exception-schema-preview --json",
			Metadata:    metadata,
		}
	default:
		return ReleaseExceptionMigrationApprovalGateItem{
			Key:              "migration_approval:release_exception_schema",
			Category:         "migration",
			Status:           "blocked",
			ApprovalStatus:   "unknown",
			Message:          "release exception schema preview returned an unsupported status",
			Owner:            "release_owner",
			RequiredEvidence: []string{"release exception schema preview returns needs_approval before migration approval review"},
			NextCommand:      "areaflow release exception-schema-preview --json",
			Metadata:         metadata,
		}
	}
}
