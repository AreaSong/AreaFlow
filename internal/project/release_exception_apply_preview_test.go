package project

import (
	"testing"
	"time"
)

func TestBuildReleaseExceptionApplyPreviewBlocksWhenMigrationGateBlocked(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	migrationGate := ReleaseExceptionMigrationApprovalGate{
		Status: "blocked",
		Mode:   "read_only_release_exception_migration_approval_gate",
		Items: []ReleaseExceptionMigrationApprovalGateItem{
			{
				Key:              "migration_approval:release_exception_schema",
				Status:           "blocked",
				ApprovalStatus:   "needs_approval",
				RequiredEvidence: []string{"approved migration approval record for release exception schema"},
			},
		},
	}

	preview := BuildReleaseExceptionApplyPreview(migrationGate, ReleaseExceptionApplyPreviewOptions{GeneratedAt: created})

	if preview.Status != "blocked" || preview.Mode != "read_only_release_exception_apply_preview" {
		t.Fatalf("unexpected apply preview: %+v", preview)
	}
	if preview.MigrationGate.Status != "blocked" {
		t.Fatalf("unexpected nested migration gate: %+v", preview.MigrationGate)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("items = %d, want 1: %+v", len(preview.Items), preview.Items)
	}
	item := preview.Items[0]
	if item.Key != "release_exception_apply:migration_approval" || item.Status != "blocked" || item.Action != "wait_for_migration_approval" {
		t.Fatalf("unexpected apply preview item: %+v", item)
	}
	if item.Metadata["blocked_by"] != "migration_approval:release_exception_schema" || item.Metadata["apply_writable"] != false {
		t.Fatalf("unexpected apply preview metadata: %+v", item.Metadata)
	}
	if len(preview.ApplySteps) != 4 || preview.ApplySteps[0].Action != "verify_migration_approval" {
		t.Fatalf("unexpected apply steps: %+v", preview.ApplySteps)
	}
	if len(preview.RollbackSteps) != 3 || preview.RollbackSteps[0].Action != "disable_exception_writes" {
		t.Fatalf("unexpected rollback steps: %+v", preview.RollbackSteps)
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[4] != "run_migration" || preview.ForbiddenActions[5] != "insert_exception_record" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleaseExceptionApplyPreviewReadyWhenMigrationGatePasses(t *testing.T) {
	preview := BuildReleaseExceptionApplyPreview(
		ReleaseExceptionMigrationApprovalGate{Status: "pass"},
		ReleaseExceptionApplyPreviewOptions{},
	)

	if preview.Status != "ready" {
		t.Fatalf("expected ready status: %+v", preview)
	}
	if len(preview.Items) != 1 || preview.Items[0].Status != "ready" || preview.Items[0].Action != "preview_apply_records" {
		t.Fatalf("unexpected ready item: %+v", preview.Items)
	}
	if preview.Items[0].Metadata["apply_writable"] != false {
		t.Fatalf("apply preview must remain read-only: %+v", preview.Items[0].Metadata)
	}
}

func TestBuildReleaseExceptionApplyPreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleaseExceptionApplyPreview(
		ReleaseExceptionMigrationApprovalGate{
			Status:     "pass",
			Scope:      "project",
			ProjectKey: "areamatrix",
			SchemaPreview: ReleaseExceptionSchemaPreview{
				Scope:      "project",
				ProjectKey: "areamatrix",
			},
		},
		ReleaseExceptionApplyPreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped apply preview: %+v", preview)
	}
	if preview.MigrationGate.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested migration gate to keep project key: %+v", preview.MigrationGate)
	}
}
