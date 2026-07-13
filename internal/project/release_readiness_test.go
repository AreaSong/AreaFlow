package project

import (
	"testing"
	"time"
)

func TestBuildReleaseReadinessNeedsAttentionForCurrentBaselineGaps(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	readiness := BuildReleaseReadiness(
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "hash-a",
			Projects:      []BackupProjectManifest{{Project: record}},
			TableCounts:   []BackupTableCount{{Table: "projects", Rows: 1}},
			GeneratedAt:   created,
		},
		RestorePlan{
			Status:        "needs_attention",
			Mode:          "read_only_restore_plan",
			SchemaVersion: 1,
			ManifestHash:  "hash-a",
			Projects:      []Record{record},
			Items:         []RestorePlanItem{{Key: "artifact_inventory", Status: "needs_attention"}},
			GeneratedAt:   created,
		},
		AuditCoverage{
			Status:              "warn",
			Mode:                "read_only_audit_coverage",
			Scope:               "platform",
			TotalAuditEvents:    10,
			CoveredRequirements: 8,
			GapRequirements:     3,
			GeneratedAt:         created,
		},
		[]ReleaseReadinessProject{
			{
				Project: record,
				Permission: PermissionPolicyDoctor{
					Status:  "pass",
					Mode:    "read_only_permission_policy_doctor",
					Project: record,
					Checks:  []PermissionPolicyCheck{{Key: "project_config", Status: "pass"}},
				},
				ArtifactIntegrity: ArtifactIntegrityReport{
					Status:           "warn",
					Mode:             "read_only_artifact_integrity",
					Project:          record,
					CheckedArtifacts: 2,
					PassedArtifacts:  1,
					SkippedArtifacts: 1,
				},
				Conformance: ConformanceReport{
					Status:      "pass",
					Mode:        "read_only_adapter_profile_conformance",
					Project:     record,
					ProfileID:   "areamatrix",
					Adapter:     "areamatrix",
					ProfileHash: "hash-profile",
					StageCount:  16,
					GateCount:   17,
					Checks:      []ConformanceCheck{{Key: "project_adapter_profile", Status: "pass"}},
				},
			},
		},
		ReleaseReadinessOptions{GeneratedAt: created, ProjectID: record.ID, ProjectKey: record.Key},
	)

	if readiness.Status != "needs_attention" || readiness.Mode != "read_only_release_readiness" ||
		readiness.Scope != "project" || readiness.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release readiness: %+v", readiness)
	}
	assertReleaseItem(t, readiness, "backup_manifest", "ready")
	assertReleaseItem(t, readiness, "restore_plan", "needs_attention")
	assertReleaseItem(t, readiness, "audit_coverage", "needs_attention")
	assertReleaseItem(t, readiness, "permission_policy:areamatrix", "ready")
	assertReleaseItem(t, readiness, "artifact_integrity:areamatrix", "needs_attention")
	assertReleaseItem(t, readiness, "adapter_profile_conformance:areamatrix", "ready")
	if len(readiness.Projects) != 1 || readiness.Projects[0].Status != "needs_attention" {
		t.Fatalf("unexpected project readiness: %+v", readiness.Projects)
	}
	if readiness.Projects[0].NeedsAttentionItems != 1 || readiness.Projects[0].BlockedItems != 0 {
		t.Fatalf("unexpected project item counters: %+v", readiness.Projects[0])
	}
	if !readiness.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", readiness.GeneratedAt, created)
	}
}

func TestBuildReleaseReadinessBlocksOnPermissionFailure(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildReleaseReadiness(
		BackupManifest{Status: "ready", Mode: "read_only_manifest", SchemaVersion: 1, ManifestHash: "hash-a", Projects: []BackupProjectManifest{{Project: record}}},
		RestorePlan{Status: "ready", Mode: "read_only_restore_plan", SchemaVersion: 1, ManifestHash: "hash-a", Projects: []Record{record}},
		AuditCoverage{Status: "pass", Mode: "read_only_audit_coverage", Scope: "platform"},
		[]ReleaseReadinessProject{
			{
				Project:           record,
				Permission:        PermissionPolicyDoctor{Status: "fail", Project: record},
				ArtifactIntegrity: ArtifactIntegrityReport{Status: "pass", Project: record},
				Conformance:       ConformanceReport{Status: "pass", Project: record},
			},
		},
		ReleaseReadinessOptions{},
	)

	if readiness.Status != "blocked" {
		t.Fatalf("expected blocked status: %+v", readiness)
	}
	assertReleaseItem(t, readiness, "permission_policy:areamatrix", "blocked")
}

func assertReleaseItem(t *testing.T, readiness ReleaseReadiness, key string, status string) {
	t.Helper()
	for _, item := range readiness.Items {
		if item.Key == key {
			if item.Status != status {
				t.Fatalf("release item %s status = %q, want %q: %+v", key, item.Status, status, item)
			}
			return
		}
	}
	t.Fatalf("release item %s not found: %+v", key, readiness.Items)
}
