package project

import (
	"testing"
	"time"
)

func TestBuildReleaseFinalGateBlocksWhenAnyFinalInputBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	gate := BuildReleaseFinalGate(
		ReleaseReadiness{Status: "needs_attention", Scope: "project", ProjectKey: "areamatrix", Items: []ReleaseReadinessItem{{Key: "restore_plan"}}},
		ReleaseAcceptanceGate{Status: "blocked", Items: []ReleaseAcceptanceGateItem{{Key: "gate:accept:restore_plan"}}},
		ReleaseExceptionApplyPreview{Status: "blocked", Items: []ReleaseExceptionApplyPreviewItem{{Key: "release_exception_apply:migration_approval"}}},
		ReleaseFinalGateOptions{GeneratedAt: created, ProjectKey: "areamatrix"},
	)

	if gate.Status != "blocked" || gate.Mode != "read_only_release_final_gate" ||
		gate.Scope != "project" || gate.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected final gate: %+v", gate)
	}
	if len(gate.Items) != 3 {
		t.Fatalf("items = %d, want 3: %+v", len(gate.Items), gate.Items)
	}
	assertReleaseFinalGateItem(t, gate, "final_gate:release_readiness", "readiness", "blocked")
	assertReleaseFinalGateItem(t, gate, "final_gate:release_acceptance", "acceptance", "blocked")
	assertReleaseFinalGateItem(t, gate, "final_gate:release_exception_apply", "release_exception", "blocked")
	if len(gate.ForbiddenActions) == 0 || gate.ForbiddenActions[3] != "create_release_package" || gate.ForbiddenActions[11] != "apply_release" {
		t.Fatalf("unexpected forbidden actions: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, created)
	}
}

func TestBuildReleaseFinalGatePassesReadyInputs(t *testing.T) {
	gate := BuildReleaseFinalGate(
		ReleaseReadiness{Status: "ready"},
		ReleaseAcceptanceGate{Status: "pass"},
		ReleaseExceptionApplyPreview{Status: "ready"},
		ReleaseFinalGateOptions{},
	)

	if gate.Status != "pass" {
		t.Fatalf("expected pass status: %+v", gate)
	}
	assertReleaseFinalGateItem(t, gate, "final_gate:release_readiness", "readiness", "pass")
	assertReleaseFinalGateItem(t, gate, "final_gate:release_acceptance", "acceptance", "pass")
	assertReleaseFinalGateItem(t, gate, "final_gate:release_exception_apply", "release_exception", "pass")
}

func TestBuildReleaseFinalGateSkipsExceptionApplyWhenAcceptanceHasNoExceptions(t *testing.T) {
	gate := BuildReleaseFinalGate(
		ReleaseReadiness{Status: "ready"},
		ReleaseAcceptanceGate{
			Status: "pass",
			Items: []ReleaseAcceptanceGateItem{{
				Key:            "gate:accept:release_ready",
				Status:         "pass",
				DecisionStatus: "ready",
			}},
		},
		ReleaseExceptionApplyPreview{Status: "blocked", Items: []ReleaseExceptionApplyPreviewItem{{Key: "release_exception_apply:migration_approval"}}},
		ReleaseFinalGateOptions{},
	)

	if gate.Status != "pass" {
		t.Fatalf("expected pass when acceptance gate has no exception requirements: %+v", gate)
	}
	for _, item := range gate.Items {
		if item.Key != "final_gate:release_exception_apply" {
			continue
		}
		if item.Status != "pass" {
			t.Fatalf("exception apply item should pass when no exception is required: %+v", item)
		}
		if item.Metadata["exception_apply_required"] != false {
			t.Fatalf("exception apply requirement metadata = %+v", item.Metadata)
		}
		return
	}
	t.Fatalf("exception apply item not found: %+v", gate.Items)
}

func assertReleaseFinalGateItem(t *testing.T, gate ReleaseFinalGate, key string, category string, status string) {
	t.Helper()
	for _, item := range gate.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status {
				t.Fatalf("item %s = %+v, want category=%s status=%s", key, item, category, status)
			}
			if item.Owner == "" || item.Message == "" || len(item.RequiredEvidence) == 0 || item.NextCommand == "" {
				t.Fatalf("item %s missing guidance: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, gate.Items)
}
