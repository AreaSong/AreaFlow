package project

import (
	"testing"
	"time"
)

func TestBuildReleaseAcceptancePreviewClassifiesExplicitDecisions(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	plan := ReleaseRemediationPlan{
		Status: "needs_attention",
		Mode:   "read_only_release_remediation_plan",
		Actions: []ReleaseRemediationAction{
			{
				Key:         "remediate:restore_plan",
				Category:    "restore",
				Status:      "needs_attention",
				SourceItem:  "restore_plan",
				Owner:       "release_owner",
				NextCommand: "areaflow backup restore-plan --json",
				Acceptance:  "restore plan is ready, or release notes explicitly accept metadata-only historical artifacts",
				Metadata:    map[string]any{"restore_status": "needs_attention"},
			},
			{
				Key:         "remediate:audit_coverage",
				Category:    "audit",
				Status:      "needs_attention",
				SourceItem:  "audit_coverage",
				Owner:       "platform_owner",
				NextCommand: "areaflow audit coverage --json",
				Acceptance:  "audit coverage is pass, or release readiness records accepted future-only audit gaps with owners",
				Metadata:    map[string]any{"gap_requirements": 3},
			},
			{
				Key:         "remediate:artifact_integrity:areamatrix",
				Category:    "artifact",
				Status:      "needs_attention",
				SourceItem:  "artifact_integrity:areamatrix",
				Owner:       "artifact_owner",
				NextCommand: "areaflow artifact integrity areamatrix --json",
				Acceptance:  "artifact integrity is pass, or skipped references are explicitly accepted with archive ownership",
				Metadata:    map[string]any{"project_key": "areamatrix", "skipped_artifacts": 1},
			},
		},
	}

	preview := BuildReleaseAcceptancePreview(plan, ReleaseAcceptancePreviewOptions{GeneratedAt: created})

	if preview.Status != "needs_decision" || preview.Mode != "read_only_release_acceptance_preview" ||
		preview.Scope != "platform" || preview.ProjectKey != "" {
		t.Fatalf("unexpected acceptance preview: %+v", preview)
	}
	if len(preview.Decisions) != 3 {
		t.Fatalf("decision count = %d, want 3: %+v", len(preview.Decisions), preview.Decisions)
	}
	assertAcceptanceDecision(t, preview, "accept:restore_plan", "restore", "needs_decision", "metadata_only_history")
	assertAcceptanceDecision(t, preview, "accept:audit_coverage", "audit", "needs_decision", "future_only_gap")
	assertAcceptanceDecision(t, preview, "accept:artifact_integrity:areamatrix", "artifact", "needs_decision", "archive_exception")
	if preview.Decisions[2].NextCommand != "areaflow artifact integrity areamatrix --json" {
		t.Fatalf("unexpected artifact next command: %+v", preview.Decisions[2])
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "write_database" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleaseAcceptancePreviewBlocksUnacceptableCategories(t *testing.T) {
	preview := BuildReleaseAcceptancePreview(
		ReleaseRemediationPlan{
			Status: "blocked",
			Actions: []ReleaseRemediationAction{
				{
					Key:        "remediate:permission_policy:areamatrix",
					Category:   "permission",
					Status:     "blocked",
					SourceItem: "permission_policy:areamatrix",
					Owner:      "security_owner",
					Acceptance: "permission policy doctor returns pass for every release project",
					Metadata:   map[string]any{"project_key": "areamatrix"},
				},
			},
		},
		ReleaseAcceptancePreviewOptions{},
	)

	if preview.Status != "not_acceptable" {
		t.Fatalf("expected not_acceptable status: %+v", preview)
	}
	assertAcceptanceDecision(t, preview, "accept:permission_policy:areamatrix", "permission", "not_acceptable", "none")
}

func TestBuildReleaseAcceptancePreviewReadyWhenNoAcceptanceRequired(t *testing.T) {
	preview := BuildReleaseAcceptancePreview(
		ReleaseRemediationPlan{
			Status: "ready",
			Actions: []ReleaseRemediationAction{
				{
					Key:        "release_ready",
					Category:   "release",
					Status:     "ready",
					SourceItem: "release_readiness",
					Owner:      "release_owner",
					Acceptance: "release readiness remains ready",
				},
			},
		},
		ReleaseAcceptancePreviewOptions{},
	)

	if preview.Status != "ready" {
		t.Fatalf("expected ready status: %+v", preview)
	}
	assertAcceptanceDecision(t, preview, "accept:release_readiness", "release", "ready", "none")
}

func TestBuildReleaseAcceptancePreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleaseAcceptancePreview(
		ReleaseRemediationPlan{
			Status:     "ready",
			Scope:      "project",
			ProjectKey: "areamatrix",
			Readiness: ReleaseReadiness{
				Scope:      "project",
				ProjectKey: "areamatrix",
			},
		},
		ReleaseAcceptancePreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped preview: %+v", preview)
	}
}

func assertAcceptanceDecision(t *testing.T, preview ReleaseAcceptancePreview, key string, category string, status string, acceptanceType string) {
	t.Helper()
	for _, decision := range preview.Decisions {
		if decision.Key == key {
			if decision.Category != category || decision.Status != status || decision.AcceptanceType != acceptanceType {
				t.Fatalf("decision %s = %+v, want category=%s status=%s acceptance_type=%s", key, decision, category, status, acceptanceType)
			}
			if decision.Owner == "" || decision.Reason == "" || len(decision.RequiredEvidence) == 0 {
				t.Fatalf("decision %s missing guidance: %+v", key, decision)
			}
			return
		}
	}
	t.Fatalf("decision %s not found: %+v", key, preview.Decisions)
}
