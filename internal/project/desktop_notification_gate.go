package project

import (
	"context"
	"time"
)

type DesktopNotificationGateOptions struct {
	GeneratedAt time.Time
}

type DesktopNotificationGate struct {
	Status                   string
	Mode                     string
	Actions                  []DesktopNotificationAction
	Capabilities             []string
	ForbiddenActions         []string
	GeneratedAt              time.Time
	DBWriteAttempted         bool
	ProjectWriteAttempted    bool
	EventStreamOpened        bool
	NotificationRequested    bool
	CommandCreated           bool
	ApprovalCreated          bool
	AuditEventWritten        bool
	WorkerScheduled          bool
	WorkflowExecutionStarted bool
	SecretsResolved          bool
	NetworkUsed              bool
}

type DesktopNotificationAction struct {
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

func (s Store) DesktopNotificationGate(_ context.Context, options DesktopNotificationGateOptions) (DesktopNotificationGate, error) {
	return BuildDesktopNotificationGate(options), nil
}

func BuildDesktopNotificationGate(options DesktopNotificationGateOptions) DesktopNotificationGate {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return DesktopNotificationGate{
		Status: "blocked",
		Mode:   "read_only_desktop_notification_gate",
		Actions: []DesktopNotificationAction{
			{
				Key:            "observe_event_stream",
				Label:          "Observe AreaFlow events",
				Category:       "event_stream",
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
					"/api/v1/projects/{project_key}/events/stream",
				},
				ForbiddenDirectActions: desktopNotificationForbiddenActions(),
			},
			{
				Key:            "enable_system_notifications",
				Label:          "Enable system notifications",
				Category:       "os_notification",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"notification permission preview",
					"event filter preview",
					"privacy redaction preview",
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
					"redaction_policy",
				},
				Blockers: []string{
					"notification_permission_flow_not_implemented",
					"notification_redaction_contract_not_defined",
				},
				ForbiddenDirectActions: desktopNotificationForbiddenActions(),
			},
			{
				Key:            "approval_needed_notifications",
				Label:          "Notify approval needed",
				Category:       "workflow_signal",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"approval event filter preview",
					"notification dedupe preview",
				},
				RequiredEvidence: []string{
					"approval_event_filter",
					"notification_dedupe_key",
				},
				Blockers: []string{
					"system_notifications_not_open",
					"approval_event_filter_not_defined",
				},
				ForbiddenDirectActions: desktopNotificationForbiddenActions(),
			},
			{
				Key:            "run_failure_notifications",
				Label:          "Notify run failures",
				Category:       "workflow_signal",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"run failure event filter preview",
					"notification rate limit preview",
				},
				RequiredEvidence: []string{
					"run_failure_event_filter",
					"rate_limit_policy",
				},
				Blockers: []string{
					"system_notifications_not_open",
					"rate_limit_policy_not_defined",
				},
				ForbiddenDirectActions: desktopNotificationForbiddenActions(),
			},
			{
				Key:            "worker_recovery_notifications",
				Label:          "Notify worker recovery",
				Category:       "worker_signal",
				Status:         "blocked",
				DefaultUIState: "disabled",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"observe_api",
					"stream_events",
				},
				RequiredPreviews: []string{
					"worker recovery event filter preview",
				},
				RequiredEvidence: []string{
					"worker_recovery_event_filter",
					"project_scope_filter",
				},
				Blockers: []string{
					"system_notifications_not_open",
					"worker_event_filter_not_defined",
				},
				ForbiddenDirectActions: desktopNotificationForbiddenActions(),
			},
		},
		Capabilities: []string{
			"observe_desktop_notification_requirements",
			"observe_api",
			"stream_events",
		},
		ForbiddenActions: []string{
			"request_os_notification_permission_without_gate",
			"open_event_stream_from_gate",
			"subscribe_without_filter",
			"send_remote_notifications",
			"include_secret_values",
			"schedule_worker_from_notification",
			"run_workflow_from_notification",
			"maintain_second_notification_state",
			"bypass_areaflow_api",
		},
		GeneratedAt: generatedAt,
	}
}

func desktopNotificationForbiddenActions() []string {
	return []string{
		"request_os_notification_permission_without_gate",
		"open_event_stream_from_gate",
		"subscribe_without_filter",
		"send_remote_notifications",
		"include_secret_values",
		"schedule_worker_from_notification",
		"run_workflow_from_notification",
		"maintain_second_notification_state",
		"bypass_areaflow_api",
	}
}
