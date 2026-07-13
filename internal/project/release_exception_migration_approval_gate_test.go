package project

import (
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/migrate"
)

func TestBuildReleaseExceptionMigrationApprovalGateBlocksWithoutApproval(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	schemaPreview := ReleaseExceptionSchemaPreview{
		Status: "needs_approval",
		Mode:   "read_only_release_exception_schema_preview",
		Tables: []ReleaseExceptionSchemaTable{
			{Name: "release_exceptions"},
		},
		ApplySteps: []ReleaseExceptionMigrationStep{
			{Order: 1, Action: "create_table"},
		},
		RollbackSteps: []ReleaseExceptionMigrationStep{
			{Order: 1, Action: "drop_table"},
		},
		AuditActions: []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
	}

	gate := BuildReleaseExceptionMigrationApprovalGate(schemaPreview, ReleaseExceptionMigrationApprovalGateOptions{GeneratedAt: created})

	if gate.Status != "blocked" || gate.Mode != "read_only_release_exception_migration_approval_gate" {
		t.Fatalf("unexpected migration approval gate: %+v", gate)
	}
	if gate.SchemaPreview.Status != "needs_approval" {
		t.Fatalf("unexpected schema preview: %+v", gate.SchemaPreview)
	}
	if len(gate.Items) != 1 {
		t.Fatalf("gate items = %d, want 1: %+v", len(gate.Items), gate.Items)
	}
	item := gate.Items[0]
	if item.Key != "migration_approval:release_exception_schema" || item.Status != "blocked" || item.ApprovalStatus != "needs_approval" {
		t.Fatalf("unexpected gate item: %+v", item)
	}
	if item.Owner == "" || item.Message == "" || len(item.RequiredEvidence) != 4 || item.NextCommand == "" {
		t.Fatalf("gate item missing guidance: %+v", item)
	}
	if item.Metadata["risk_level"] != "R4 migration_security" || item.Metadata["migration_writable"] != false {
		t.Fatalf("unexpected gate item metadata: %+v", item.Metadata)
	}
	if len(gate.ForbiddenActions) == 0 || gate.ForbiddenActions[3] != "create_migration_file" || gate.ForbiddenActions[4] != "run_migration" {
		t.Fatalf("unexpected forbidden actions: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, created)
	}
}

func TestBuildReleaseExceptionMigrationApprovalGatePassesEffectiveApproval(t *testing.T) {
	hash, err := migrate.ExpectedHash(migrate.ReleaseExceptionMigrationName)
	if err != nil {
		t.Fatal(err)
	}
	gate := BuildReleaseExceptionMigrationApprovalGateWithState(
		ReleaseExceptionSchemaPreview{Status: "needs_approval"},
		ReleaseExceptionMigrationApprovalGateOptions{},
		migrate.ApprovalState{Status: "approved", MigrationHash: hash, Actor: "release-owner", Applied: true},
	)

	if gate.Status != "pass" || len(gate.Items) != 1 || gate.Items[0].ApprovalStatus != "approved" {
		t.Fatalf("effective approval should pass migration gate: %+v", gate)
	}
	if gate.Items[0].Metadata["migration_applied"] != true {
		t.Fatalf("migration applied state missing: %+v", gate.Items[0].Metadata)
	}
}

func TestBuildReleaseExceptionMigrationApprovalGateBlocksRevokedApproval(t *testing.T) {
	gate := BuildReleaseExceptionMigrationApprovalGateWithState(
		ReleaseExceptionSchemaPreview{Status: "needs_approval"},
		ReleaseExceptionMigrationApprovalGateOptions{},
		migrate.ApprovalState{Status: "revoked", MigrationHash: "hash"},
	)

	if gate.Status != "blocked" || gate.Items[0].ApprovalStatus != "revoked" {
		t.Fatalf("revoked approval should block migration gate: %+v", gate)
	}
}

func TestBuildReleaseExceptionMigrationApprovalGateBlocksBlockedSchemaPreview(t *testing.T) {
	gate := BuildReleaseExceptionMigrationApprovalGate(
		ReleaseExceptionSchemaPreview{Status: "blocked"},
		ReleaseExceptionMigrationApprovalGateOptions{},
	)

	if gate.Status != "blocked" {
		t.Fatalf("expected blocked status: %+v", gate)
	}
	if len(gate.Items) != 1 || gate.Items[0].ApprovalStatus != "blocked" {
		t.Fatalf("expected blocked approval item: %+v", gate.Items)
	}
}

func TestBuildReleaseExceptionMigrationApprovalGatePropagatesProjectScope(t *testing.T) {
	gate := BuildReleaseExceptionMigrationApprovalGate(
		ReleaseExceptionSchemaPreview{
			Status:     "needs_approval",
			Scope:      "project",
			ProjectKey: "areamatrix",
			RecordPreview: ReleaseExceptionRecordPreview{
				Scope:      "project",
				ProjectKey: "areamatrix",
			},
		},
		ReleaseExceptionMigrationApprovalGateOptions{ProjectKey: "areamatrix"},
	)

	if gate.Scope != "project" || gate.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped migration approval gate: %+v", gate)
	}
	if gate.SchemaPreview.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested schema preview to keep project key: %+v", gate.SchemaPreview)
	}
}
