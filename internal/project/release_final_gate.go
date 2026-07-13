package project

import (
	"context"
	"time"
)

type ReleaseFinalGateOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseFinalGateItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseFinalGate struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Readiness        ReleaseReadiness
	AcceptanceGate   ReleaseAcceptanceGate
	ExceptionApply   ReleaseExceptionApplyPreview
	Items            []ReleaseFinalGateItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseFinalGate(ctx context.Context, options ReleaseFinalGateOptions) (ReleaseFinalGate, error) {
	options = normalizeReleaseFinalGateOptions(options)
	readiness, err := s.ReleaseReadiness(ctx, ReleaseReadinessOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseFinalGate{}, err
	}
	acceptanceGate, err := s.ReleaseAcceptanceGate(ctx, ReleaseAcceptanceGateOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseFinalGate{}, err
	}
	exceptionApply, err := s.ReleaseExceptionApplyPreview(ctx, ReleaseExceptionApplyPreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseFinalGate{}, err
	}
	return BuildReleaseFinalGate(readiness, acceptanceGate, exceptionApply, options), nil
}

func normalizeReleaseFinalGateOptions(options ReleaseFinalGateOptions) ReleaseFinalGateOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseFinalGate(readiness ReleaseReadiness, acceptanceGate ReleaseAcceptanceGate, exceptionApply ReleaseExceptionApplyPreview, options ReleaseFinalGateOptions) ReleaseFinalGate {
	options = normalizeReleaseFinalGateOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, readiness.ProjectKey)
	gate := ReleaseFinalGate{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "pass",
		Mode:             "read_only_release_final_gate",
		Scope:            scope,
		ProjectKey:       projectKey,
		Readiness:        readiness,
		AcceptanceGate:   acceptanceGate,
		ExceptionApply:   exceptionApply,
		Items:            []ReleaseFinalGateItem{},
		Capabilities: []string{
			"read_release_readiness",
			"read_release_acceptance_gate",
			"read_release_exception_apply_preview",
			"evaluate_release_final_gate",
			"report_release_final_blockers",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"create_release_package",
			"create_approval",
			"mark_gap_accepted",
			"run_migration",
			"insert_exception_record",
			"insert_audit_event",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	gate.addItem(releaseFinalReadinessItem(readiness))
	gate.addItem(releaseFinalAcceptanceItem(acceptanceGate))
	gate.addItem(releaseFinalExceptionApplyItem(acceptanceGate, exceptionApply))
	return gate
}

func (g *ReleaseFinalGate) addItem(item ReleaseFinalGateItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	g.Items = append(g.Items, item)
	if item.Status == "blocked" {
		g.Status = "blocked"
	}
}

func releaseFinalReadinessItem(readiness ReleaseReadiness) ReleaseFinalGateItem {
	status := "pass"
	message := "release readiness is ready"
	requiredEvidence := []string{"release readiness status remains ready"}
	if readiness.Status != "ready" {
		status = "blocked"
		message = "release readiness is not ready"
		requiredEvidence = []string{"areaflow release readiness --json returns status ready"}
	}
	return ReleaseFinalGateItem{
		Key:              "final_gate:release_readiness",
		Category:         "readiness",
		Status:           status,
		Message:          message,
		Owner:            "release_owner",
		RequiredEvidence: requiredEvidence,
		NextCommand:      "areaflow release readiness --json",
		Metadata: map[string]any{
			"readiness_status": readiness.Status,
			"item_count":       len(readiness.Items),
			"project_count":    len(readiness.Projects),
		},
	}
}

func releaseFinalAcceptanceItem(acceptanceGate ReleaseAcceptanceGate) ReleaseFinalGateItem {
	status := "pass"
	message := "release acceptance gate passes"
	requiredEvidence := []string{"release acceptance gate remains pass"}
	if acceptanceGate.Status != "pass" {
		status = "blocked"
		message = "release acceptance gate blocks release"
		requiredEvidence = []string{"areaflow release acceptance-gate --json returns status pass"}
	}
	return ReleaseFinalGateItem{
		Key:              "final_gate:release_acceptance",
		Category:         "acceptance",
		Status:           status,
		Message:          message,
		Owner:            "release_owner",
		RequiredEvidence: requiredEvidence,
		NextCommand:      "areaflow release acceptance-gate --json",
		Metadata: map[string]any{
			"acceptance_gate_status": acceptanceGate.Status,
			"item_count":             len(acceptanceGate.Items),
		},
	}
}

func releaseFinalExceptionApplyItem(acceptanceGate ReleaseAcceptanceGate, exceptionApply ReleaseExceptionApplyPreview) ReleaseFinalGateItem {
	status := "pass"
	message := "release exception apply preview is ready"
	requiredEvidence := []string{"release exception apply preview remains ready or no release exception apply is required"}
	exceptionApplyRequired := releaseAcceptanceGateRequiresExceptionApply(acceptanceGate)
	if exceptionApply.Status == "blocked" && exceptionApplyRequired {
		status = "blocked"
		message = "release exception apply preview blocks release"
		requiredEvidence = []string{"areaflow release exception-apply-preview --json returns status ready, or release acceptance gate passes without exceptions"}
	} else if exceptionApply.Status == "blocked" {
		message = "release acceptance gate passes without exception apply requirements"
	}
	return ReleaseFinalGateItem{
		Key:              "final_gate:release_exception_apply",
		Category:         "release_exception",
		Status:           status,
		Message:          message,
		Owner:            "release_owner",
		RequiredEvidence: requiredEvidence,
		NextCommand:      "areaflow release exception-apply-preview --json",
		Metadata: map[string]any{
			"acceptance_gate_status":   acceptanceGate.Status,
			"exception_apply_required": exceptionApplyRequired,
			"exception_apply_status":   exceptionApply.Status,
			"item_count":               len(exceptionApply.Items),
			"risk_level":               "R4 migration_security",
		},
	}
}

func releaseAcceptanceGateRequiresExceptionApply(acceptanceGate ReleaseAcceptanceGate) bool {
	if acceptanceGate.Status != "pass" {
		return true
	}
	for _, item := range acceptanceGate.Items {
		if item.DecisionStatus != "" && item.DecisionStatus != "ready" {
			return true
		}
	}
	return false
}
