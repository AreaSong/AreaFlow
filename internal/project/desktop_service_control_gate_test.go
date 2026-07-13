package project

import (
	"testing"
	"time"
)

func TestBuildDesktopServiceControlGateKeepsControlsDisabled(t *testing.T) {
	generated := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	gate := BuildDesktopServiceControlGate(DesktopServiceControlGateOptions{GeneratedAt: generated})

	if gate.Status != "blocked" || gate.Mode != "read_only_desktop_service_control_gate" {
		t.Fatalf("unexpected desktop service control gate: %+v", gate)
	}
	if len(gate.Actions) < 5 {
		t.Fatalf("expected desktop control action matrix, got %d actions", len(gate.Actions))
	}
	if gate.DBWriteAttempted || gate.ProjectWriteAttempted || gate.ProcessControlAttempted || gate.CommandCreated {
		t.Fatalf("gate should not attempt side effects: %+v", gate)
	}
	if !containsString(gate.ForbiddenActions, "run_workflow_directly") {
		t.Fatalf("missing workflow guardrail: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, generated)
	}

	seenRestart := false
	seenDashboard := false
	for _, action := range gate.Actions {
		if !containsString(action.ForbiddenDirectActions, "bypass_areaflow_api") &&
			action.Key != "enable_notifications" {
			t.Fatalf("action missing API guardrail: %+v", action)
		}
		switch action.Key {
		case "open_dashboard":
			seenDashboard = true
			if action.Status != "ready" || action.DefaultUIState != "enabled_link" {
				t.Fatalf("dashboard action should remain a read-only launcher: %+v", action)
			}
		case "restart_service":
			seenRestart = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("restart action should stay disabled: %+v", action)
			}
			if !containsString(action.Blockers, "restart_recovery_contract_not_defined") {
				t.Fatalf("restart blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenDashboard || !seenRestart {
		t.Fatalf("missing dashboard or restart action: %+v", gate.Actions)
	}
}
