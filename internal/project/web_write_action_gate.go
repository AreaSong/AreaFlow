package project

import (
	"context"
	"time"
)

type WebWriteActionGateOptions struct {
	GeneratedAt time.Time
}

type WebWriteActionGate struct {
	Status                  string
	Mode                    string
	Actions                 []WebWriteActionGateAction
	Capabilities            []string
	ForbiddenActions        []string
	GeneratedAt             time.Time
	DBWriteAttempted        bool
	ProjectWriteAttempted   bool
	ArtifactWriteAttempted  bool
	ExecutionWriteAttempted bool
	CommandCreated          bool
	ApprovalCreated         bool
	AuditEventWritten       bool
	WorkerScheduled         bool
	EngineCallAttempted     bool
	CommandsRun             bool
	SecretsResolved         bool
	NetworkUsed             bool
}

type WebWriteActionGateAction struct {
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

func (s Store) WebWriteActionGate(_ context.Context, options WebWriteActionGateOptions) (WebWriteActionGate, error) {
	return BuildWebWriteActionGate(options), nil
}

func BuildWebWriteActionGate(options WebWriteActionGateOptions) WebWriteActionGate {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return WebWriteActionGate{
		Status: "blocked",
		Mode:   "read_only_web_write_action_gate",
		Actions: []WebWriteActionGateAction{
			{
				Key:            "approval_record",
				Label:          "Record approval decision",
				Category:       "workflow_governance",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "workflow.approval.record",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"write_workflow",
				},
				RequiredPreviews: []string{
					"approval impact preview",
					"latest transition preview",
				},
				RequiredApprovals: []string{
					"explicit actor confirmation",
				},
				RequiredAuditEvents: []string{
					"approval decision",
				},
				RequiredEvidence: []string{
					"idempotency_key",
					"request_hash",
					"actor",
					"reason",
				},
				Blockers: []string{
					"web_write_actions_not_open",
					"command_api_confirmation_contract_not_bound_to_web",
				},
				ForbiddenDirectActions: commonWebWriteForbiddenActions(),
			},
			{
				Key:            "run_drain",
				Label:          "Drain run",
				Category:       "run_control",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "run.drain",
				RiskLevel:      "R2 managed_write",
				RequiredCapabilities: []string{
					"manage_workers",
				},
				RequiredPreviews: []string{
					"run detail",
					"active lease summary",
				},
				RequiredApprovals: []string{
					"operator confirmation",
				},
				RequiredAuditEvents: []string{
					"run drain requested",
				},
				RequiredEvidence: []string{
					"idempotency_key",
					"request_hash",
					"expected_run_status",
				},
				Blockers: []string{
					"web_write_actions_not_open",
					"web_run_control_requires_command_api_binding",
				},
				ForbiddenDirectActions: commonWebWriteForbiddenActions(),
			},
			{
				Key:            "run_cancel",
				Label:          "Cancel run",
				Category:       "run_control",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "run.cancel",
				RiskLevel:      "R3 execution",
				RequiredCapabilities: []string{
					"manage_workers",
				},
				RequiredPreviews: []string{
					"run detail",
					"active lease summary",
					"artifact retention preview",
				},
				RequiredApprovals: []string{
					"operator confirmation",
					"high risk approval when execution has started",
				},
				RequiredAuditEvents: []string{
					"run cancel requested",
				},
				RequiredEvidence: []string{
					"idempotency_key",
					"request_hash",
					"cancel_reason",
				},
				Blockers: []string{
					"web_write_actions_not_open",
					"web_run_control_requires_command_api_binding",
				},
				ForbiddenDirectActions: commonWebWriteForbiddenActions(),
			},
			{
				Key:            "artifact_archive_preview",
				Label:          "Preview artifact archive",
				Category:       "artifact",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "artifact.archive.preview",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"write_artifacts",
				},
				RequiredPreviews: []string{
					"artifact integrity",
					"retention policy preview",
				},
				RequiredApprovals: []string{
					"operator confirmation for command creation",
				},
				RequiredAuditEvents: []string{
					"artifact archive preview requested",
				},
				RequiredEvidence: []string{
					"idempotency_key",
					"request_hash",
					"retention_policy",
				},
				Blockers: []string{
					"web_write_actions_not_open",
					"archive_apply_not_open",
				},
				ForbiddenDirectActions: append(commonWebWriteForbiddenActions(),
					"delete_artifact_content",
					"rewrite_artifact_metadata",
				),
			},
			{
				Key:            "status_projection_apply",
				Label:          "Write status projection",
				Category:       "projection",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "project.status_projection.apply",
				RiskLevel:      "R1 projection",
				RequiredCapabilities: []string{
					"write_status",
				},
				RequiredPreviews: []string{
					"status projection preview",
					"path allowlist preflight",
				},
				RequiredApprovals: []string{
					"operator confirmation",
				},
				RequiredAuditEvents: []string{
					"status projection apply requested",
				},
				RequiredEvidence: []string{
					"idempotency_key",
					"request_hash",
					"target_path",
					"source_hash",
				},
				Blockers: []string{
					"web_write_actions_not_open",
					"project_path_write_requires_command_api_binding",
				},
				ForbiddenDirectActions: append(commonWebWriteForbiddenActions(),
					"write_workflow_readme",
					"write_workflow_versions",
				),
			},
			{
				Key:            "generated_write_apply_beta",
				Label:          "Approve generated write beta",
				Category:       "project_write",
				Status:         "blocked",
				DefaultUIState: "disabled",
				CommandAPI:     "future.generated_write.apply_beta",
				RiskLevel:      "R3 execution",
				RequiredCapabilities: []string{
					"read_project",
					"write_artifacts",
					"write_generated",
				},
				RequiredPreviews: []string{
					"generated write readiness",
					"generated write apply beta gate",
					"rollback verification plan",
				},
				RequiredApprovals: []string{
					"explicit R3 approval",
				},
				RequiredAuditEvents: []string{
					"generated write beta approval requested",
					"project write permission decision",
				},
				RequiredEvidence: []string{
					"single existing generated target",
					"expected_before_hash",
					"preimage_artifact",
					"non_target_fingerprint_plan",
				},
				Blockers: []string{
					"real_areamatrix_generated_apply_closed",
					"web_write_actions_not_open",
				},
				ForbiddenDirectActions: append(commonWebWriteForbiddenActions(),
					"write_source_code",
					"write_execution",
					"write_progress_json",
					"write_checkpoint",
				),
			},
		},
		Capabilities: []string{
			"observe_web_write_actions",
			"view_risk_preview",
			"view_permission_preflight",
			"view_required_approval",
			"view_audit_requirements",
		},
		ForbiddenActions: []string{
			"enable_write_buttons_by_default",
			"write_database_directly",
			"write_project_files",
			"call_command_api_without_idempotency_key",
			"bypass_permission_preflight",
			"bypass_approval_gate",
			"schedule_worker_from_web",
			"treat_sse_as_state_source",
		},
		GeneratedAt: generatedAt,
	}
}

func commonWebWriteForbiddenActions() []string {
	return []string{
		"write_database_directly",
		"write_project_files_directly",
		"bypass_areaflow_api",
		"bypass_permission_preflight",
		"bypass_approval_gate",
		"skip_audit_event",
	}
}
