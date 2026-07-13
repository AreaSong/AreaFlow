package project

import (
	"context"
	"time"
)

type DesktopTrayMenuGateOptions struct {
	GeneratedAt time.Time
}

type DesktopTrayMenuGate struct {
	Status                   string
	Mode                     string
	Actions                  []DesktopTrayMenuAction
	Capabilities             []string
	ForbiddenActions         []string
	GeneratedAt              time.Time
	DBWriteAttempted         bool
	ProjectWriteAttempted    bool
	TrayMenuCreated          bool
	OSIntegrationRequested   bool
	CommandCreated           bool
	ApprovalCreated          bool
	AuditEventWritten        bool
	ServiceControlAttempted  bool
	NotificationRequested    bool
	WorkerScheduled          bool
	WorkflowExecutionStarted bool
	SecretsResolved          bool
	NetworkUsed              bool
}

type DesktopTrayMenuAction struct {
	Key                    string
	Label                  string
	Category               string
	Status                 string
	DefaultUIState         string
	RiskLevel              string
	RequiredCapabilities   []string
	RequiredPreviews       []string
	RequiredApprovals      []string
	RequiredAuditEvents    []string
	RequiredEvidence       []string
	Blockers               []string
	ForbiddenDirectActions []string
}

func (s Store) DesktopTrayMenuGate(_ context.Context, options DesktopTrayMenuGateOptions) (DesktopTrayMenuGate, error) {
	return BuildDesktopTrayMenuGate(options), nil
}

func BuildDesktopTrayMenuGate(options DesktopTrayMenuGateOptions) DesktopTrayMenuGate {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return DesktopTrayMenuGate{
		Status: "blocked",
		Mode:   "read_only_desktop_tray_menu_gate",
		Actions: []DesktopTrayMenuAction{
			{
				Key:            "open_dashboard",
				Label:          "Open dashboard",
				Category:       "launcher",
				Status:         "ready",
				DefaultUIState: "enabled_link",
				RiskLevel:      "R0 read_only",
				RequiredCapabilities: []string{
					"open_web_dashboard",
				},
				RequiredPreviews: []string{
					"service status",
				},
				RequiredEvidence: []string{
					"dashboard_url",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
			{
				Key:            "show_service_status",
				Label:          "Show service status",
				Category:       "status",
				Status:         "ready",
				DefaultUIState: "available_read_only",
				RiskLevel:      "R0 read_only",
				RequiredCapabilities: []string{
					"observe_api",
					"observe_worker_pool",
				},
				RequiredPreviews: []string{
					"service status",
				},
				RequiredEvidence: []string{
					"/api/v1/service/status",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
			{
				Key:            "show_recent_events",
				Label:          "Show recent events",
				Category:       "status",
				Status:         "ready",
				DefaultUIState: "available_read_only",
				RiskLevel:      "R0 read_only",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"event filter preview",
				},
				RequiredEvidence: []string{
					"/api/v1/events/stream",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
			{
				Key:            "start_service",
				Label:          "Start service",
				Category:       "service_control",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"manage_local_service",
				},
				RequiredPreviews: []string{
					"desktop service control gate",
					"local process preflight",
				},
				RequiredApprovals: []string{
					"local operator confirmation",
				},
				RequiredAuditEvents: []string{
					"desktop tray service start requested",
				},
				RequiredEvidence: []string{
					"process_supervision_contract",
				},
				Blockers: []string{
					"service_control_gate_blocked",
					"tray_service_control_not_open",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
			{
				Key:            "stop_service",
				Label:          "Stop service",
				Category:       "service_control",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"manage_local_service",
				},
				RequiredPreviews: []string{
					"desktop service control gate",
					"active lease summary",
				},
				RequiredApprovals: []string{
					"local operator confirmation",
				},
				RequiredAuditEvents: []string{
					"desktop tray service stop requested",
				},
				RequiredEvidence: []string{
					"drain_or_cancel_policy",
				},
				Blockers: []string{
					"service_control_gate_blocked",
					"tray_service_control_not_open",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
			{
				Key:            "enable_notifications",
				Label:          "Enable notifications",
				Category:       "notification",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"desktop notification gate",
				},
				RequiredApprovals: []string{
					"local OS notification permission",
				},
				RequiredAuditEvents: []string{
					"desktop tray notifications requested",
				},
				RequiredEvidence: []string{
					"notification_gate_pass",
					"os_permission_status",
				},
				Blockers: []string{
					"notification_gate_blocked",
					"tray_notification_action_not_open",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
			{
				Key:            "open_settings",
				Label:          "Open settings",
				Category:       "settings",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
				},
				RequiredPreviews: []string{
					"settings surface preview",
					"secret readiness preview",
				},
				RequiredEvidence: []string{
					"settings_contract",
					"secret_values_hidden",
				},
				Blockers: []string{
					"settings_surface_not_implemented",
					"secret_ui_contract_not_defined",
				},
				ForbiddenDirectActions: desktopTrayMenuForbiddenActions(),
			},
		},
		Capabilities: []string{
			"observe_desktop_tray_requirements",
			"open_web_dashboard",
			"observe_api",
		},
		ForbiddenActions: []string{
			"create_tray_menu_from_gate",
			"request_os_integration_without_gate",
			"start_service_from_tray_without_gate",
			"stop_service_from_tray_without_gate",
			"request_notifications_from_tray_without_gate",
			"open_secret_settings_with_values",
			"schedule_worker_from_tray",
			"run_workflow_from_tray",
			"maintain_second_tray_state",
			"bypass_areaflow_api",
		},
		GeneratedAt: generatedAt,
	}
}

func desktopTrayMenuForbiddenActions() []string {
	return []string{
		"create_tray_menu_from_gate",
		"request_os_integration_without_gate",
		"start_service_from_tray_without_gate",
		"stop_service_from_tray_without_gate",
		"request_notifications_from_tray_without_gate",
		"open_secret_settings_with_values",
		"schedule_worker_from_tray",
		"run_workflow_from_tray",
		"maintain_second_tray_state",
		"bypass_areaflow_api",
	}
}
