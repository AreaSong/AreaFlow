package project

import (
	"testing"
	"time"
)

func TestBuildDesktopNotificationGateKeepsNotificationsDisabled(t *testing.T) {
	generated := time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC)
	gate := BuildDesktopNotificationGate(DesktopNotificationGateOptions{GeneratedAt: generated})

	if gate.Status != "blocked" || gate.Mode != "read_only_desktop_notification_gate" {
		t.Fatalf("unexpected desktop notification gate: %+v", gate)
	}
	if len(gate.Actions) < 5 {
		t.Fatalf("expected notification action matrix, got %d actions", len(gate.Actions))
	}
	if gate.DBWriteAttempted || gate.ProjectWriteAttempted || gate.EventStreamOpened ||
		gate.NotificationRequested || gate.CommandCreated || gate.WorkerScheduled ||
		gate.WorkflowExecutionStarted || gate.SecretsResolved || gate.NetworkUsed {
		t.Fatalf("gate should not attempt side effects: %+v", gate)
	}
	if !containsString(gate.ForbiddenActions, "request_os_notification_permission_without_gate") ||
		!containsString(gate.ForbiddenActions, "schedule_worker_from_notification") {
		t.Fatalf("missing notification guardrails: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, generated)
	}

	seenStream := false
	seenSystem := false
	for _, action := range gate.Actions {
		if !containsString(action.ForbiddenDirectActions, "bypass_areaflow_api") {
			t.Fatalf("action missing API guardrail: %+v", action)
		}
		switch action.Key {
		case "observe_event_stream":
			seenStream = true
			if action.Status != "ready" || action.DefaultUIState != "available_read_only" {
				t.Fatalf("event stream action should remain read-only available: %+v", action)
			}
		case "enable_system_notifications":
			seenSystem = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("system notification action should stay disabled: %+v", action)
			}
			if !containsString(action.Blockers, "notification_permission_flow_not_implemented") {
				t.Fatalf("system notification blockers = %+v", action.Blockers)
			}
		}
	}
	if !seenStream || !seenSystem {
		t.Fatalf("missing event stream or system notification action: %+v", gate.Actions)
	}
}
