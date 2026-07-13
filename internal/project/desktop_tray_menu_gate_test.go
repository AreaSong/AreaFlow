package project

import (
	"testing"
	"time"
)

func TestBuildDesktopTrayMenuGateKeepsControlActionsDisabled(t *testing.T) {
	generated := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	gate := BuildDesktopTrayMenuGate(DesktopTrayMenuGateOptions{GeneratedAt: generated})

	if gate.Status != "blocked" || gate.Mode != "read_only_desktop_tray_menu_gate" {
		t.Fatalf("unexpected desktop tray menu gate: %+v", gate)
	}
	if len(gate.Actions) < 7 {
		t.Fatalf("expected tray menu action matrix, got %d actions", len(gate.Actions))
	}
	if gate.DBWriteAttempted || gate.ProjectWriteAttempted || gate.TrayMenuCreated ||
		gate.OSIntegrationRequested || gate.CommandCreated || gate.ServiceControlAttempted ||
		gate.NotificationRequested || gate.WorkerScheduled || gate.WorkflowExecutionStarted ||
		gate.SecretsResolved || gate.NetworkUsed {
		t.Fatalf("gate should not attempt side effects: %+v", gate)
	}
	if !containsString(gate.ForbiddenActions, "create_tray_menu_from_gate") ||
		!containsString(gate.ForbiddenActions, "schedule_worker_from_tray") {
		t.Fatalf("missing tray guardrails: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, generated)
	}

	seenDashboard := false
	seenStart := false
	seenSettings := false
	for _, action := range gate.Actions {
		if !containsString(action.ForbiddenDirectActions, "bypass_areaflow_api") {
			t.Fatalf("action missing API guardrail: %+v", action)
		}
		switch action.Key {
		case "open_dashboard":
			seenDashboard = true
			if action.Status != "ready" || action.DefaultUIState != "enabled_link" {
				t.Fatalf("dashboard action should remain launcher-only: %+v", action)
			}
		case "start_service":
			seenStart = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("start service action should stay disabled: %+v", action)
			}
			if !containsString(action.Blockers, "service_control_gate_blocked") {
				t.Fatalf("start service blockers = %+v", action.Blockers)
			}
		case "open_settings":
			seenSettings = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("settings action should stay disabled: %+v", action)
			}
			if !containsString(action.Blockers, "secret_ui_contract_not_defined") {
				t.Fatalf("settings blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenDashboard || !seenStart || !seenSettings {
		t.Fatalf("missing dashboard, start, or settings action: %+v", gate.Actions)
	}
}
