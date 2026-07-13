package project

import (
	"context"
	"time"
)

type DesktopServiceControlGateOptions struct {
	GeneratedAt time.Time
}

type DesktopServiceControlGate struct {
	Status                   string
	Mode                     string
	Actions                  []DesktopServiceControlAction
	Capabilities             []string
	ForbiddenActions         []string
	GeneratedAt              time.Time
	DBWriteAttempted         bool
	ProjectWriteAttempted    bool
	ProcessControlAttempted  bool
	CommandCreated           bool
	ApprovalCreated          bool
	AuditEventWritten        bool
	WorkerScheduled          bool
	WorkflowExecutionStarted bool
	SecretsResolved          bool
	NetworkUsed              bool
}

type DesktopServiceControlAction struct {
	Key                    string
	Label                  string
	Category               string
	Status                 string
	DefaultUIState         string
	CommandAPI             string
	RiskLevel              string
	RequiredCapabilities   []string
	RequiredPreviews       []string
	RequiredApprovals      []string
	RequiredAuditEvents    []string
	RequiredEvidence       []string
	Blockers               []string
	ForbiddenDirectActions []string
}

func (s Store) DesktopServiceControlGate(_ context.Context, options DesktopServiceControlGateOptions) (DesktopServiceControlGate, error) {
	return BuildDesktopServiceControlGate(options), nil
}

func BuildDesktopServiceControlGate(options DesktopServiceControlGateOptions) DesktopServiceControlGate {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return DesktopServiceControlGate{
		Status: "blocked",
		Mode:   "read_only_desktop_service_control_gate",
		Actions: []DesktopServiceControlAction{
			{
				Key:            "open_dashboard",
				Label:          "Open Web dashboard",
				Category:       "launcher",
				Status:         "ready",
				DefaultUIState: "enabled_link",
				CommandAPI:     "none",
				RiskLevel:      "R0 read_only",
				RequiredCapabilities: []string{
					"open_web_dashboard",
				},
				RequiredPreviews: []string{
					"service status",
				},
				RequiredAuditEvents: []string{},
				RequiredEvidence: []string{
					"dashboard_url",
					"api_url",
				},
				ForbiddenDirectActions: []string{
					"write_project_files",
					"run_workflow_directly",
					"maintain_second_database",
					"bypass_areaflow_api",
				},
			},
			{
				Key:            "start_service",
				Label:          "Start local service",
				Category:       "local_service_manager",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "future.desktop.service.start",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"manage_local_service",
				},
				RequiredPreviews: []string{
					"service status",
					"local process preflight",
					"port availability check",
				},
				RequiredApprovals: []string{
					"local operator confirmation",
				},
				RequiredAuditEvents: []string{
					"desktop service start requested",
				},
				RequiredEvidence: []string{
					"api_bind_addr",
					"dashboard_url",
					"postgres_readiness",
				},
				Blockers: []string{
					"desktop_service_control_not_open",
					"process_supervision_contract_not_defined",
				},
				ForbiddenDirectActions: desktopControlForbiddenActions(),
			},
			{
				Key:            "stop_service",
				Label:          "Stop local service",
				Category:       "local_service_manager",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "future.desktop.service.stop",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"manage_local_service",
				},
				RequiredPreviews: []string{
					"active request summary",
					"worker lease summary",
				},
				RequiredApprovals: []string{
					"local operator confirmation",
				},
				RequiredAuditEvents: []string{
					"desktop service stop requested",
				},
				RequiredEvidence: []string{
					"drain_or_cancel_policy",
					"active_request_count",
				},
				Blockers: []string{
					"desktop_service_control_not_open",
					"service_stop_requires_drain_policy",
				},
				ForbiddenDirectActions: desktopControlForbiddenActions(),
			},
			{
				Key:            "restart_service",
				Label:          "Restart local service",
				Category:       "local_service_manager",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "future.desktop.service.restart",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"manage_local_service",
				},
				RequiredPreviews: []string{
					"stop service gate",
					"start service gate",
					"state recovery check",
				},
				RequiredApprovals: []string{
					"local operator confirmation",
				},
				RequiredAuditEvents: []string{
					"desktop service restart requested",
				},
				RequiredEvidence: []string{
					"pre_restart_service_status",
					"post_restart_smoke_plan",
				},
				Blockers: []string{
					"desktop_service_control_not_open",
					"restart_recovery_contract_not_defined",
				},
				ForbiddenDirectActions: desktopControlForbiddenActions(),
			},
			{
				Key:            "enable_notifications",
				Label:          "Enable system notifications",
				Category:       "notification",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "future.desktop.notifications.enable",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"notification permission preview",
					"event subscription filter preview",
				},
				RequiredApprovals: []string{
					"local OS notification permission",
				},
				RequiredAuditEvents: []string{
					"desktop notifications enabled",
				},
				RequiredEvidence: []string{
					"os_permission_status",
					"event_filter",
				},
				Blockers: []string{
					"notification_permission_flow_not_implemented",
					"sse_notification_bridge_not_implemented",
				},
				ForbiddenDirectActions: []string{
					"read_secret_values",
					"send_remote_notifications",
					"subscribe_without_filter",
				},
			},
			{
				Key:            "tray_menu",
				Label:          "Enable tray/menu controls",
				Category:       "shell",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "future.desktop.tray.enable",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"open_web_dashboard",
				},
				RequiredPreviews: []string{
					"menu action gate",
				},
				RequiredApprovals: []string{
					"local operator confirmation for service controls",
				},
				RequiredAuditEvents: []string{
					"desktop tray action requested",
				},
				RequiredEvidence: []string{
					"menu_action_matrix",
				},
				Blockers: []string{
					"tray_menu_not_implemented",
					"service_control_actions_disabled",
				},
				ForbiddenDirectActions: desktopControlForbiddenActions(),
			},
		},
		Capabilities: []string{
			"observe_desktop_service_controls",
			"open_web_dashboard",
			"view_service_control_requirements",
			"view_notification_requirements",
		},
		ForbiddenActions: []string{
			"start_service_without_gate",
			"stop_service_without_drain_policy",
			"restart_service_without_recovery_check",
			"run_workflow_directly",
			"schedule_worker_from_desktop",
			"maintain_second_database",
			"write_project_files",
			"resolve_secrets",
			"bypass_areaflow_api",
		},
		GeneratedAt: generatedAt,
	}
}

func desktopControlForbiddenActions() []string {
	return []string{
		"run_workflow_directly",
		"schedule_worker_from_desktop",
		"write_project_files",
		"maintain_second_database",
		"bypass_areaflow_api",
		"resolve_secrets",
		"ignore_active_leases",
	}
}
