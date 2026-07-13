package project

import (
	"testing"
	"time"
)

func TestBuildReleaseExceptionSchemaPreviewDescribesMigrationPlan(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	recordPreview := ReleaseExceptionRecordPreview{
		Status: "draft",
		Mode:   "read_only_release_exception_record_preview",
		Drafts: []ReleaseExceptionRecordDraft{
			{Key: "release_exception:restore_plan", Status: "draft", AcceptanceType: "metadata_only_history"},
		},
	}

	preview := BuildReleaseExceptionSchemaPreview(recordPreview, ReleaseExceptionSchemaPreviewOptions{GeneratedAt: created})

	if preview.Status != "needs_approval" || preview.Mode != "read_only_release_exception_schema_preview" {
		t.Fatalf("unexpected schema preview: %+v", preview)
	}
	if len(preview.Tables) != 1 || preview.Tables[0].Name != "release_exceptions" {
		t.Fatalf("unexpected tables: %+v", preview.Tables)
	}
	assertSchemaColumn(t, preview.Tables[0], "exception_key", "TEXT", false)
	assertSchemaColumn(t, preview.Tables[0], "required_evidence", "JSONB", false)
	assertSchemaColumn(t, preview.Tables[0], "rollback_plan", "TEXT", false)
	if len(preview.Tables[0].Indexes) != 3 || preview.Tables[0].Indexes[0].Name != "release_exceptions_key_idx" {
		t.Fatalf("unexpected indexes: %+v", preview.Tables[0].Indexes)
	}
	if len(preview.Tables[0].ForeignKeys) != 4 {
		t.Fatalf("unexpected foreign keys: %+v", preview.Tables[0].ForeignKeys)
	}
	if len(preview.ApplySteps) != 3 || preview.ApplySteps[0].Action != "create_table" {
		t.Fatalf("unexpected apply steps: %+v", preview.ApplySteps)
	}
	if len(preview.RollbackSteps) != 3 || preview.RollbackSteps[2].Action != "drop_table" {
		t.Fatalf("unexpected rollback steps: %+v", preview.RollbackSteps)
	}
	if len(preview.AuditActions) != 3 || preview.AuditActions[0] != "release.exception.request" {
		t.Fatalf("unexpected audit actions: %+v", preview.AuditActions)
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "write_database" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleaseExceptionSchemaPreviewBlocksWhenRecordPreviewBlocked(t *testing.T) {
	preview := BuildReleaseExceptionSchemaPreview(
		ReleaseExceptionRecordPreview{Status: "blocked"},
		ReleaseExceptionSchemaPreviewOptions{},
	)

	if preview.Status != "blocked" {
		t.Fatalf("expected blocked status: %+v", preview)
	}
}

func TestBuildReleaseExceptionSchemaPreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleaseExceptionSchemaPreview(
		ReleaseExceptionRecordPreview{
			Status:     "draft",
			Scope:      "project",
			ProjectKey: "areamatrix",
			Doctor: ReleaseExceptionDoctor{
				Scope:      "project",
				ProjectKey: "areamatrix",
			},
		},
		ReleaseExceptionSchemaPreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped schema preview: %+v", preview)
	}
	if preview.RecordPreview.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested record preview to keep project key: %+v", preview.RecordPreview)
	}
}

func assertSchemaColumn(t *testing.T, table ReleaseExceptionSchemaTable, name string, typ string, nullable bool) {
	t.Helper()
	for _, column := range table.Columns {
		if column.Name == name {
			if column.Type != typ || column.Nullable != nullable {
				t.Fatalf("column %s = %+v, want type=%s nullable=%t", name, column, typ, nullable)
			}
			if column.Purpose == "" {
				t.Fatalf("column %s missing purpose", name)
			}
			return
		}
	}
	t.Fatalf("column %s not found: %+v", name, table.Columns)
}
