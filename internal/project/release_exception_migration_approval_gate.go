package project

import (
	"context"
	"time"

	"github.com/areasong/areaflow/internal/migrate"
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
	approval, err := migrate.Approval(ctx, s.pool, migrate.ReleaseExceptionMigrationName)
	if err != nil {
		return ReleaseExceptionMigrationApprovalGate{}, err
	}
	effective, err := migrate.ApprovalEffective(migrate.ReleaseExceptionMigrationName, approval)
	if err != nil {
		return ReleaseExceptionMigrationApprovalGate{}, err
	}
	return buildReleaseExceptionMigrationApprovalGate(schemaPreview, options, approval, effective), nil
}

func normalizeReleaseExceptionMigrationApprovalGateOptions(options ReleaseExceptionMigrationApprovalGateOptions) ReleaseExceptionMigrationApprovalGateOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseExceptionMigrationApprovalGate(schemaPreview ReleaseExceptionSchemaPreview, options ReleaseExceptionMigrationApprovalGateOptions) ReleaseExceptionMigrationApprovalGate {
	return BuildReleaseExceptionMigrationApprovalGateWithState(schemaPreview, options, migrate.ApprovalState{})
}

func BuildReleaseExceptionMigrationApprovalGateWithState(schemaPreview ReleaseExceptionSchemaPreview, options ReleaseExceptionMigrationApprovalGateOptions, approval migrate.ApprovalState) ReleaseExceptionMigrationApprovalGate {
	effective, _ := migrate.ApprovalEffective(migrate.ReleaseExceptionMigrationName, approval)
	return buildReleaseExceptionMigrationApprovalGate(schemaPreview, options, approval, effective)
}

func buildReleaseExceptionMigrationApprovalGate(schemaPreview ReleaseExceptionSchemaPreview, options ReleaseExceptionMigrationApprovalGateOptions, approval migrate.ApprovalState, approvalEffective bool) ReleaseExceptionMigrationApprovalGate {
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
	gate.addItem(releaseExceptionMigrationApprovalItem(schemaPreview, approval, approvalEffective))
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

func releaseExceptionMigrationApprovalItem(schemaPreview ReleaseExceptionSchemaPreview, approval migrate.ApprovalState, approvalEffective bool) ReleaseExceptionMigrationApprovalGateItem {
	metadata := map[string]any{
		"schema_preview_status": schemaPreview.Status,
		"table_count":           len(schemaPreview.Tables),
		"apply_step_count":      len(schemaPreview.ApplySteps),
		"rollback_step_count":   len(schemaPreview.RollbackSteps),
		"audit_action_count":    len(schemaPreview.AuditActions),
		"risk_level":            "R4 migration_security",
		"migration_writable":    false,
		"migration_name":        migrate.ReleaseExceptionMigrationName,
		"migration_applied":     approval.Applied,
		"approval_actor":        approval.Actor,
		"approval_effective":    approvalEffective,
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
		if approval.Status == "approved" && approvalEffective {
			metadata["approval_status"] = approval.Status
			metadata["migration_hash"] = approval.MigrationHash
			return ReleaseExceptionMigrationApprovalGateItem{
				Key:              "migration_approval:release_exception_schema",
				Category:         "migration",
				Status:           "pass",
				ApprovalStatus:   "approved",
				Message:          "explicit R4 migration approval is effective for the release exception schema",
				Owner:            "release_owner",
				RequiredEvidence: []string{"migration approval remains approved and bound to the embedded 000012 migration hash"},
				NextCommand:      "areaflow release exception-migration-apply",
				Metadata:         metadata,
			}
		}
		if approval.Status == "approved" {
			metadata["approval_status"] = "stale"
			return ReleaseExceptionMigrationApprovalGateItem{
				Key:              "migration_approval:release_exception_schema",
				Category:         "migration",
				Status:           "blocked",
				ApprovalStatus:   "stale",
				Message:          "release exception migration approval does not match the embedded migration hash",
				Owner:            "release_owner",
				RequiredEvidence: []string{"review the current 000012 migration and record a new explicit approval"},
				NextCommand:      "areaflow release exception-migration-approve --actor ACTOR --reason TEXT",
				Metadata:         metadata,
			}
		}
		if approval.Status == "revoked" {
			metadata["approval_status"] = approval.Status
			return ReleaseExceptionMigrationApprovalGateItem{
				Key:              "migration_approval:release_exception_schema",
				Category:         "migration",
				Status:           "blocked",
				ApprovalStatus:   "revoked",
				Message:          "release exception migration approval has been revoked",
				Owner:            "release_owner",
				RequiredEvidence: []string{"record a new explicit R4 migration approval before apply or exception writes"},
				NextCommand:      "areaflow release exception-migration-approve --actor ACTOR --reason TEXT",
				Metadata:         metadata,
			}
		}
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
