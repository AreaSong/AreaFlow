package project

import (
	"testing"
	"time"
)

func TestBuildReleaseAcceptanceGateBlocksPendingDecisions(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	preview := ReleaseAcceptancePreview{
		Status: "needs_decision",
		Mode:   "read_only_release_acceptance_preview",
		Decisions: []ReleaseAcceptanceDecision{
			{
				Key:              "accept:restore_plan",
				SourceAction:     "remediate:restore_plan",
				Category:         "restore",
				Status:           "needs_decision",
				AcceptanceType:   "metadata_only_history",
				Owner:            "release_owner",
				RequiredEvidence: []string{"release notes state metadata-only artifacts"},
				NextCommand:      "areaflow backup restore-plan --json",
				Metadata:         map[string]any{"restore_status": "needs_attention"},
			},
			{
				Key:              "accept:audit_coverage",
				SourceAction:     "remediate:audit_coverage",
				Category:         "audit",
				Status:           "needs_decision",
				AcceptanceType:   "future_only_gap",
				Owner:            "platform_owner",
				RequiredEvidence: []string{"audit coverage lists missing actions"},
				NextCommand:      "areaflow audit coverage --json",
				Metadata:         map[string]any{"gap_requirements": 3},
			},
		},
	}

	gate := BuildReleaseAcceptanceGate(preview, ReleaseAcceptanceGateOptions{GeneratedAt: created})

	if gate.Status != "blocked" || gate.Mode != "read_only_release_acceptance_gate" ||
		gate.Scope != "platform" || gate.ProjectKey != "" {
		t.Fatalf("unexpected release acceptance gate: %+v", gate)
	}
	if len(gate.Items) != 2 {
		t.Fatalf("gate items = %d, want 2: %+v", len(gate.Items), gate.Items)
	}
	assertAcceptanceGateItem(t, gate, "gate:accept:restore_plan", "restore", "blocked", "needs_decision", "metadata_only_history")
	assertAcceptanceGateItem(t, gate, "gate:accept:audit_coverage", "audit", "blocked", "needs_decision", "future_only_gap")
	if gate.Items[0].NextCommand != "areaflow backup restore-plan --json" {
		t.Fatalf("unexpected next command: %+v", gate.Items[0])
	}
	if len(gate.ForbiddenActions) == 0 || gate.ForbiddenActions[0] != "write_database" {
		t.Fatalf("unexpected forbidden actions: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, created)
	}
}

func TestBuildReleaseAcceptanceGatePassesReadyDecisions(t *testing.T) {
	gate := BuildReleaseAcceptanceGate(
		ReleaseAcceptancePreview{
			Status: "ready",
			Decisions: []ReleaseAcceptanceDecision{
				{
					Key:              "accept:release_readiness",
					Category:         "release",
					Status:           "ready",
					AcceptanceType:   "none",
					Owner:            "release_owner",
					RequiredEvidence: []string{"release readiness remains ready"},
				},
			},
		},
		ReleaseAcceptanceGateOptions{},
	)

	if gate.Status != "pass" {
		t.Fatalf("expected pass status: %+v", gate)
	}
	assertAcceptanceGateItem(t, gate, "gate:accept:release_readiness", "release", "pass", "ready", "none")
}

func TestBuildReleaseAcceptanceGateBlocksNotAcceptableDecisions(t *testing.T) {
	gate := BuildReleaseAcceptanceGate(
		ReleaseAcceptancePreview{
			Status: "not_acceptable",
			Decisions: []ReleaseAcceptanceDecision{
				{
					Key:              "accept:permission_policy:areamatrix",
					Category:         "permission",
					Status:           "not_acceptable",
					AcceptanceType:   "none",
					Owner:            "security_owner",
					RequiredEvidence: []string{"permission policy doctor returns pass"},
				},
			},
		},
		ReleaseAcceptanceGateOptions{},
	)

	if gate.Status != "blocked" {
		t.Fatalf("expected blocked status: %+v", gate)
	}
	assertAcceptanceGateItem(t, gate, "gate:accept:permission_policy:areamatrix", "permission", "blocked", "not_acceptable", "none")
}

func TestBuildReleaseAcceptanceGatePropagatesProjectScope(t *testing.T) {
	gate := BuildReleaseAcceptanceGate(
		ReleaseAcceptancePreview{
			Status:     "ready",
			Scope:      "project",
			ProjectKey: "areamatrix",
		},
		ReleaseAcceptanceGateOptions{ProjectKey: "areamatrix"},
	)

	if gate.Scope != "project" || gate.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped gate: %+v", gate)
	}
	if gate.Preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested preview to keep project key: %+v", gate.Preview)
	}
}

func assertAcceptanceGateItem(t *testing.T, gate ReleaseAcceptanceGate, key string, category string, status string, decisionStatus string, acceptanceType string) {
	t.Helper()
	for _, item := range gate.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status || item.DecisionStatus != decisionStatus || item.AcceptanceType != acceptanceType {
				t.Fatalf("item %s = %+v, want category=%s status=%s decision=%s acceptance_type=%s", key, item, category, status, decisionStatus, acceptanceType)
			}
			if item.Owner == "" || item.Message == "" || len(item.RequiredEvidence) == 0 {
				t.Fatalf("item %s missing guidance: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, gate.Items)
}
