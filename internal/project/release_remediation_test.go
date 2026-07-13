package project

import (
	"testing"
	"time"
)

func TestBuildReleaseRemediationPlanClassifiesNeedsAttentionItems(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	readiness := ReleaseReadiness{
		Status: "needs_attention",
		Mode:   "read_only_release_readiness",
		Items: []ReleaseReadinessItem{
			{
				Key:      "backup_manifest",
				Category: "backup",
				Status:   "ready",
				Message:  "backup manifest is ready",
			},
			{
				Key:      "restore_plan",
				Category: "restore",
				Status:   "needs_attention",
				Message:  "restore dry-run plan needs attention",
				Metadata: map[string]any{"restore_status": "needs_attention"},
			},
			{
				Key:      "audit_coverage",
				Category: "audit",
				Status:   "needs_attention",
				Message:  "audit coverage has gaps",
				Metadata: map[string]any{"gap_requirements": 3},
			},
			{
				Key:      "artifact_integrity:areamatrix",
				Category: "artifact",
				Status:   "needs_attention",
				Message:  "artifact integrity has warnings",
				Metadata: map[string]any{"project_key": "areamatrix", "skipped_artifacts": 1},
			},
		},
	}

	plan := BuildReleaseRemediationPlan(readiness, ReleaseRemediationOptions{GeneratedAt: created})

	if plan.Status != "needs_attention" || plan.Mode != "read_only_release_remediation_plan" {
		t.Fatalf("unexpected remediation plan: %+v", plan)
	}
	if len(plan.Actions) != 3 {
		t.Fatalf("action count = %d, want 3: %+v", len(plan.Actions), plan.Actions)
	}
	assertRemediationAction(t, plan, "remediate:restore_plan", "restore", "needs_attention")
	assertRemediationAction(t, plan, "remediate:audit_coverage", "audit", "needs_attention")
	assertRemediationAction(t, plan, "remediate:artifact_integrity:areamatrix", "artifact", "needs_attention")
	if plan.Actions[2].NextCommand != "areaflow artifact integrity areamatrix --json" {
		t.Fatalf("unexpected artifact next command: %+v", plan.Actions[2])
	}
	if !plan.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", plan.GeneratedAt, created)
	}
}

func TestBuildReleaseRemediationPlanReadyWhenNoActionsRequired(t *testing.T) {
	plan := BuildReleaseRemediationPlan(
		ReleaseReadiness{Status: "ready", Mode: "read_only_release_readiness"},
		ReleaseRemediationOptions{},
	)

	if plan.Status != "ready" {
		t.Fatalf("expected ready status: %+v", plan)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Key != "release_ready" || plan.Actions[0].Status != "ready" {
		t.Fatalf("unexpected ready action: %+v", plan.Actions)
	}
}

func TestBuildReleaseRemediationPlanBlocksOnBlockedItems(t *testing.T) {
	plan := BuildReleaseRemediationPlan(
		ReleaseReadiness{
			Status: "blocked",
			Items: []ReleaseReadinessItem{
				{Key: "permission_policy:areamatrix", Category: "permission", Status: "blocked", Metadata: map[string]any{"project_key": "areamatrix"}},
			},
		},
		ReleaseRemediationOptions{},
	)

	if plan.Status != "blocked" {
		t.Fatalf("expected blocked status: %+v", plan)
	}
	assertRemediationAction(t, plan, "remediate:permission_policy:areamatrix", "permission", "blocked")
}

func TestBuildReleaseRemediationPlanPropagatesProjectScope(t *testing.T) {
	plan := BuildReleaseRemediationPlan(
		ReleaseReadiness{
			Status:     "ready",
			Scope:      "project",
			ProjectKey: "areamatrix",
		},
		ReleaseRemediationOptions{ProjectKey: "areamatrix"},
	)

	if plan.Scope != "project" || plan.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped remediation plan: %+v", plan)
	}
	if plan.Readiness.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested readiness to keep project key: %+v", plan.Readiness)
	}
}

func assertRemediationAction(t *testing.T, plan ReleaseRemediationPlan, key string, category string, status string) {
	t.Helper()
	for _, action := range plan.Actions {
		if action.Key == key {
			if action.Category != category || action.Status != status {
				t.Fatalf("action %s = %+v, want category=%s status=%s", key, action, category, status)
			}
			if action.RecommendedAction == "" || action.Acceptance == "" {
				t.Fatalf("action %s missing guidance: %+v", key, action)
			}
			return
		}
	}
	t.Fatalf("action %s not found: %+v", key, plan.Actions)
}
