package project

import (
	"testing"
	"time"
)

func TestBuildWebWriteActionGateIsReadOnlyAndBlocked(t *testing.T) {
	generated := time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)
	gate := BuildWebWriteActionGate(WebWriteActionGateOptions{GeneratedAt: generated})

	if gate.Status != "blocked" || gate.Mode != "read_only_web_write_action_gate" {
		t.Fatalf("unexpected gate status: %+v", gate)
	}
	if len(gate.Actions) < 6 {
		t.Fatalf("expected write action matrix, got %d actions", len(gate.Actions))
	}
	if !gate.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, generated)
	}
	if gate.DBWriteAttempted || gate.ProjectWriteAttempted || gate.CommandCreated || gate.AuditEventWritten {
		t.Fatalf("gate should be read-only, got %+v", gate)
	}
	if !containsString(gate.ForbiddenActions, "enable_write_buttons_by_default") {
		t.Fatalf("missing web write forbidden action: %+v", gate.ForbiddenActions)
	}

	seenGeneratedWrite := false
	for _, action := range gate.Actions {
		if action.Status != "blocked" || action.DefaultUIState != "disabled" {
			t.Fatalf("action should remain disabled: %+v", action)
		}
		if !containsString(action.ForbiddenDirectActions, "bypass_areaflow_api") {
			t.Fatalf("action missing direct-write guardrail: %+v", action)
		}
		if action.Key == "generated_write_apply_beta" {
			seenGeneratedWrite = true
			if action.RiskLevel != "R3 execution" {
				t.Fatalf("generated write risk = %s, want R3 execution", action.RiskLevel)
			}
			if !containsString(action.Blockers, "real_areamatrix_generated_apply_closed") {
				t.Fatalf("generated write blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenGeneratedWrite {
		t.Fatalf("missing generated write beta action: %+v", gate.Actions)
	}
}
