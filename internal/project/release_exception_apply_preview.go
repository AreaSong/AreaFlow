package project

import (
	"context"
	"time"
)

type ReleaseExceptionApplyPreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseExceptionApplyPreviewItem struct {
	Key              string
	Category         string
	Status           string
	Action           string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseExceptionApplyPreviewStep struct {
	Order       int
	Action      string
	Description string
	BlockedBy   []string
}

type ReleaseExceptionApplyPreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	MigrationGate    ReleaseExceptionMigrationApprovalGate
	Items            []ReleaseExceptionApplyPreviewItem
	ApplySteps       []ReleaseExceptionApplyPreviewStep
	RollbackSteps    []ReleaseExceptionApplyPreviewStep
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseExceptionApplyPreview(ctx context.Context, options ReleaseExceptionApplyPreviewOptions) (ReleaseExceptionApplyPreview, error) {
	options = normalizeReleaseExceptionApplyPreviewOptions(options)
	migrationGate, err := s.ReleaseExceptionMigrationApprovalGate(ctx, ReleaseExceptionMigrationApprovalGateOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseExceptionApplyPreview{}, err
	}
	return BuildReleaseExceptionApplyPreview(migrationGate, options), nil
}

func normalizeReleaseExceptionApplyPreviewOptions(options ReleaseExceptionApplyPreviewOptions) ReleaseExceptionApplyPreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseExceptionApplyPreview(migrationGate ReleaseExceptionMigrationApprovalGate, options ReleaseExceptionApplyPreviewOptions) ReleaseExceptionApplyPreview {
	options = normalizeReleaseExceptionApplyPreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, migrationGate.ProjectKey, migrationGate.SchemaPreview.ProjectKey)
	preview := ReleaseExceptionApplyPreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_exception_apply_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		MigrationGate:    migrationGate,
		Items:            []ReleaseExceptionApplyPreviewItem{},
		ApplySteps: []ReleaseExceptionApplyPreviewStep{
			{Order: 1, Action: "verify_migration_approval", Description: "confirm release exception migration approval gate passes", BlockedBy: []string{"migration_approval:release_exception_schema"}},
			{Order: 2, Action: "apply_release_exception_migration", Description: "create and run the approved release_exceptions migration", BlockedBy: []string{"migration_approval:release_exception_schema"}},
			{Order: 3, Action: "write_exception_records", Description: "write approved release exception records and related audit events", BlockedBy: []string{"release_exception_write_approval"}},
			{Order: 4, Action: "rerun_release_acceptance_gate", Description: "rerun release acceptance gate after approved exception records exist", BlockedBy: []string{"release_exception_write_approval"}},
		},
		RollbackSteps: []ReleaseExceptionApplyPreviewStep{
			{Order: 1, Action: "disable_exception_writes", Description: "disable exception write endpoints before rollback", BlockedBy: []string{}},
			{Order: 2, Action: "revoke_exception_records", Description: "revoke applied exception records with audit events", BlockedBy: []string{}},
			{Order: 3, Action: "restore_release_gate_block", Description: "rerun release acceptance gate and restore blocked status if exceptions are revoked", BlockedBy: []string{}},
		},
		Capabilities: []string{
			"read_release_exception_migration_approval_gate",
			"preview_release_exception_apply_plan",
			"report_release_exception_apply_blockers",
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
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	if migrationGate.Status == "blocked" {
		preview.addItem(releaseExceptionApplyBlockedByMigrationGateItem(migrationGate))
	} else {
		preview.addItem(releaseExceptionApplyReadyItem(migrationGate))
	}
	return preview
}

func (p *ReleaseExceptionApplyPreview) addItem(item ReleaseExceptionApplyPreviewItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	p.Items = append(p.Items, item)
	if item.Status == "blocked" {
		p.Status = "blocked"
	}
}

func releaseExceptionApplyBlockedByMigrationGateItem(migrationGate ReleaseExceptionMigrationApprovalGate) ReleaseExceptionApplyPreviewItem {
	requiredEvidence := []string{"release exception migration approval gate returns pass"}
	if len(migrationGate.Items) > 0 {
		requiredEvidence = migrationGate.Items[0].RequiredEvidence
	}
	metadata := map[string]any{
		"migration_gate_status": migrationGate.Status,
		"risk_level":            "R4 migration_security",
		"apply_writable":        false,
	}
	if len(migrationGate.Items) > 0 {
		metadata["approval_status"] = migrationGate.Items[0].ApprovalStatus
		metadata["blocked_by"] = migrationGate.Items[0].Key
	}
	return ReleaseExceptionApplyPreviewItem{
		Key:              "release_exception_apply:migration_approval",
		Category:         "migration",
		Status:           "blocked",
		Action:           "wait_for_migration_approval",
		Message:          "release exception apply is blocked until migration approval gate passes",
		Owner:            "release_owner",
		RequiredEvidence: requiredEvidence,
		NextCommand:      "areaflow release exception-migration-approval-gate --json",
		Metadata:         metadata,
	}
}

func releaseExceptionApplyReadyItem(migrationGate ReleaseExceptionMigrationApprovalGate) ReleaseExceptionApplyPreviewItem {
	return ReleaseExceptionApplyPreviewItem{
		Key:              "release_exception_apply:records",
		Category:         "release_exception",
		Status:           "ready",
		Action:           "preview_apply_records",
		Message:          "release exception apply prerequisites are ready for explicit apply approval",
		Owner:            "release_owner",
		RequiredEvidence: []string{"explicit release exception apply approval", "release exception schema preview unchanged since migration approval"},
		NextCommand:      "areaflow release exception-apply-preview --json",
		Metadata: map[string]any{
			"migration_gate_status": migrationGate.Status,
			"risk_level":            "R4 migration_security",
			"apply_writable":        false,
		},
	}
}
