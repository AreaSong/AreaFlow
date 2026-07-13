package project

import (
	"context"
	"time"
)

type GeneratedWriteApplyBetaGateOptions struct {
	GeneratedAt time.Time
}

type GeneratedWriteApplyBetaGateItem struct {
	Key              string
	Category         string
	Status           string
	ApprovalStatus   string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type GeneratedWriteApplyBetaGate struct {
	Project                       Record
	Status                        string
	Mode                          string
	Readiness                     GeneratedWriteReadiness
	Items                         []GeneratedWriteApplyBetaGateItem
	RequiredCapabilities          []string
	AllowedGeneratedPrefixes      []string
	RequiredEvidence              []string
	ForbiddenActions              []string
	ApprovalRequired              bool
	ApprovalStatus                string
	ApplyOpen                     bool
	RealAreaMatrixWriteOpened     bool
	GeneratedOnly                 bool
	ProjectReadAttempted          bool
	ProjectWriteAttempted         bool
	ExecutionWriteAttempted       bool
	AreaFlowArtifactWritten       bool
	AreaFlowExecutionStateWritten bool
	EngineCallAttempted           bool
	CommandsRun                   bool
	SecretsResolved               bool
	NetworkUsed                   bool
	TaskClaimed                   bool
	WorkerStarted                 bool
	LeaseCreated                  bool
	AttemptCreated                bool
	ArtifactCreated               bool
	GeneratedAt                   time.Time
}

func (s Store) GeneratedWriteApplyBetaGate(ctx context.Context, record Record, options GeneratedWriteApplyBetaGateOptions) (GeneratedWriteApplyBetaGate, error) {
	options = normalizeGeneratedWriteApplyBetaGateOptions(options)
	readiness, err := s.GeneratedWriteReadiness(ctx, record, GeneratedWriteReadinessOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return GeneratedWriteApplyBetaGate{}, err
	}
	return BuildGeneratedWriteApplyBetaGate(readiness, options), nil
}

func normalizeGeneratedWriteApplyBetaGateOptions(options GeneratedWriteApplyBetaGateOptions) GeneratedWriteApplyBetaGateOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildGeneratedWriteApplyBetaGate(readiness GeneratedWriteReadiness, options GeneratedWriteApplyBetaGateOptions) GeneratedWriteApplyBetaGate {
	options = normalizeGeneratedWriteApplyBetaGateOptions(options)
	requiredEvidence := []string{
		"generated-write-readiness ready_for_review=true",
		"make smoke-docker-managed-generated-write passes",
		"make smoke-docker-areamatrix-readonly passes after config review",
		"explicit R3 approval for real AreaMatrix generated-only apply beta",
		"single existing regular generated target path selected",
		"expected-before sha256 and size captured for the target path",
		"preimage artifact and rollback verification plan reviewed",
		"non-target AreaMatrix fingerprints remain unchanged",
	}
	gate := GeneratedWriteApplyBetaGate{
		Project:                  readiness.Project,
		Status:                   "pass",
		Mode:                     "read_only_generated_write_apply_beta_gate",
		Readiness:                readiness,
		Items:                    []GeneratedWriteApplyBetaGateItem{},
		RequiredCapabilities:     []string{"read_project", "write_artifacts", "write_generated"},
		AllowedGeneratedPrefixes: readiness.AllowedGeneratedPrefixes,
		RequiredEvidence:         requiredEvidence,
		ForbiddenActions: []string{
			"queue_run",
			"claim_task",
			"start_worker",
			"create_lease",
			"create_attempt",
			"create_artifact",
			"write_project_file",
			"write_source_file",
			"write_workflow_execution",
			"write_progress_json",
			"git_checkpoint",
			"execute_engine",
			"run_commands",
			"resolve_secrets",
			"use_network",
		},
		ApprovalRequired:          true,
		ApprovalStatus:            "needs_approval",
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		ProjectReadAttempted:      false,
		ProjectWriteAttempted:     false,
		ExecutionWriteAttempted:   false,
		EngineCallAttempted:       false,
		CommandsRun:               false,
		SecretsResolved:           false,
		NetworkUsed:               false,
		TaskClaimed:               false,
		WorkerStarted:             false,
		LeaseCreated:              false,
		AttemptCreated:            false,
		ArtifactCreated:           false,
		GeneratedAt:               options.GeneratedAt,
	}
	gate.addItem(generatedApplyBetaReadinessItem(readiness))
	gate.addItem(generatedApplyBetaApprovalItem(readiness, requiredEvidence))
	gate.addItem(generatedApplyBetaScopeItem(readiness))
	gate.addItem(generatedApplyBetaReadOnlyItem())
	return gate
}

func (g *GeneratedWriteApplyBetaGate) addItem(item GeneratedWriteApplyBetaGateItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	g.Items = append(g.Items, item)
	if item.Status == "blocked" || item.Status == "fail" || item.Status == "needs_approval" {
		g.Status = "blocked"
	}
}

func generatedApplyBetaReadinessItem(readiness GeneratedWriteReadiness) GeneratedWriteApplyBetaGateItem {
	status := "pass"
	message := "generated write readiness is ready for beta apply review"
	if !readiness.ReadyForReview {
		status = "blocked"
		message = "generated write readiness is not ready for beta apply review"
	}
	return GeneratedWriteApplyBetaGateItem{
		Key:              "generated_apply_beta:readiness",
		Category:         "readiness",
		Status:           status,
		Message:          message,
		Owner:            "platform_owner",
		RequiredEvidence: []string{"areaflow project generated-write-readiness " + readiness.Project.Key + " --json"},
		NextCommand:      "areaflow project generated-write-readiness " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"readiness_status":             readiness.Status,
			"ready_for_review":             readiness.ReadyForReview,
			"review_blocker_count":         len(readiness.ReviewBlockers),
			"apply_open":                   readiness.ApplyOpen,
			"real_areamatrix_write_opened": readiness.RealAreaMatrixWriteOpened,
		},
	}
}

func generatedApplyBetaApprovalItem(readiness GeneratedWriteReadiness, requiredEvidence []string) GeneratedWriteApplyBetaGateItem {
	return GeneratedWriteApplyBetaGateItem{
		Key:              "generated_apply_beta:explicit_approval",
		Category:         "approval",
		Status:           "blocked",
		ApprovalStatus:   "needs_approval",
		Message:          "explicit R3 approval is required before real AreaMatrix generated-only apply beta can open",
		Owner:            "platform_owner",
		RequiredEvidence: append([]string{}, requiredEvidence...),
		NextCommand:      "areaflow project generated-write-apply-beta-gate " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"risk_level":                    "R3 execution",
			"approval_scope":                "real_areamatrix_generated_only_apply_beta",
			"apply_open":                    false,
			"real_areamatrix_write_opened":  false,
			"generated_only":                true,
			"source_write_allowed":          false,
			"workflow_execution_write_open": false,
		},
	}
}

func generatedApplyBetaScopeItem(readiness GeneratedWriteReadiness) GeneratedWriteApplyBetaGateItem {
	return GeneratedWriteApplyBetaGateItem{
		Key:      "generated_apply_beta:scope",
		Category: "scope",
		Status:   "pass",
		Message:  "future beta scope is limited to one existing generated file with expected-before and rollback verification",
		Owner:    "platform_owner",
		RequiredEvidence: []string{
			"target path is inside .areaflow/generated/** or .areamatrix/generated/**",
			"target path is an existing regular file",
			"expected-before sha256 and size match before apply",
			"rollback verification restores preimage sha256 and size",
		},
		NextCommand: "areaflow project generated-write-readiness " + readiness.Project.Key + " --json",
		Metadata: map[string]any{
			"allowed_generated_prefixes": readiness.AllowedGeneratedPrefixes,
			"required_capabilities":      []string{"read_project", "write_artifacts", "write_generated"},
			"unsupported_operations": []string{
				"create",
				"delete",
				"move",
				"chmod",
				"binary_rewrite",
				"source_write",
				"workflow_execution_write",
				"progress_json_write",
				"checkpoint",
				"repair",
			},
		},
	}
}

func generatedApplyBetaReadOnlyItem() GeneratedWriteApplyBetaGateItem {
	return GeneratedWriteApplyBetaGateItem{
		Key:      "generated_apply_beta:read_only_gate",
		Category: "safety",
		Status:   "pass",
		Message:  "gate did not queue runs, claim tasks, create leases, create attempts, create artifacts or write project files",
		Owner:    "platform_owner",
		Metadata: map[string]any{
			"project_read_attempted":            false,
			"project_write_attempted":           false,
			"execution_write_attempted":         false,
			"area_flow_artifact_written":        false,
			"area_flow_execution_state_written": false,
			"engine_call_attempted":             false,
			"commands_run":                      false,
			"secrets_resolved":                  false,
			"network_used":                      false,
			"task_claimed":                      false,
			"worker_started":                    false,
			"lease_created":                     false,
			"attempt_created":                   false,
			"artifact_created":                  false,
		},
	}
}
