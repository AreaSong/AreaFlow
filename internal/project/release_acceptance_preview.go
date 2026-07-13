package project

import (
	"context"
	"fmt"
	"time"
)

type ReleaseAcceptancePreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseAcceptanceDecision struct {
	Key              string
	SourceAction     string
	Category         string
	Status           string
	AcceptanceType   string
	Owner            string
	Reason           string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseAcceptancePreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Remediation      ReleaseRemediationPlan
	Decisions        []ReleaseAcceptanceDecision
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseAcceptancePreview(ctx context.Context, options ReleaseAcceptancePreviewOptions) (ReleaseAcceptancePreview, error) {
	options = normalizeReleaseAcceptancePreviewOptions(options)
	plan, err := s.ReleaseRemediationPlan(ctx, ReleaseRemediationOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseAcceptancePreview{}, err
	}
	return BuildReleaseAcceptancePreview(plan, options), nil
}

func normalizeReleaseAcceptancePreviewOptions(options ReleaseAcceptancePreviewOptions) ReleaseAcceptancePreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseAcceptancePreview(plan ReleaseRemediationPlan, options ReleaseAcceptancePreviewOptions) ReleaseAcceptancePreview {
	options = normalizeReleaseAcceptancePreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, plan.ProjectKey, plan.Readiness.ProjectKey)
	preview := ReleaseAcceptancePreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_acceptance_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		Remediation:      plan,
		Decisions:        []ReleaseAcceptanceDecision{},
		Capabilities: []string{
			"read_release_remediation_plan",
			"classify_acceptance_candidates",
			"generate_acceptance_preview",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"mark_gap_accepted",
			"create_approval",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	for _, action := range plan.Actions {
		decision := acceptanceDecisionForAction(action)
		preview.addDecision(decision)
	}
	if len(preview.Decisions) == 0 && plan.Status == "ready" {
		preview.addDecision(ReleaseAcceptanceDecision{
			Key:              "release_ready",
			SourceAction:     "release_ready",
			Category:         "release",
			Status:           "ready",
			AcceptanceType:   "none",
			Owner:            "release_owner",
			Reason:           "release remediation plan has no acceptance candidates",
			RequiredEvidence: []string{"release readiness remains ready"},
			Metadata:         map[string]any{"remediation_status": plan.Status},
		})
	}
	return preview
}

func (p *ReleaseAcceptancePreview) addDecision(decision ReleaseAcceptanceDecision) {
	if decision.Metadata == nil {
		decision.Metadata = map[string]any{}
	}
	if decision.AcceptanceType == "" {
		decision.AcceptanceType = "none"
	}
	p.Decisions = append(p.Decisions, decision)
	if worseAcceptancePreviewStatus(decision.Status, p.Status) {
		p.Status = decision.Status
	}
}

func acceptanceDecisionForAction(action ReleaseRemediationAction) ReleaseAcceptanceDecision {
	if action.Status == "ready" {
		return ReleaseAcceptanceDecision{
			Key:              "accept:" + action.SourceItem,
			SourceAction:     action.Key,
			Category:         action.Category,
			Status:           "ready",
			AcceptanceType:   "none",
			Owner:            action.Owner,
			Reason:           "source remediation action is already ready",
			RequiredEvidence: []string{action.Acceptance},
			NextCommand:      action.NextCommand,
			Metadata:         acceptanceMetadata(action),
		}
	}
	if action.Status == "blocked" {
		return notAcceptableDecision(action, "blocked release remediation actions must be repaired before release acceptance")
	}
	switch action.Category {
	case "restore":
		return needsDecision(action, "metadata_only_history", "historical project reference artifacts can be accepted only as an explicit metadata-only release exception", []string{
			"release notes state which historical artifact originals remain outside AreaFlow-owned storage",
			"archive owner accepts restore limitation",
			"restore dry-run plan remains reproducible",
		})
	case "audit":
		return needsDecision(action, "future_only_gap", "future-only audit gaps can be accepted only when they are disabled capabilities with named owners", []string{
			"audit coverage lists missing actions",
			"each future-only gap has an owner and enablement milestone",
			"enabled capabilities have no missing audit evidence",
		})
	case "artifact":
		return needsDecision(action, "archive_exception", "skipped project-reference artifacts can be accepted only with an explicit archive ownership decision", []string{
			"artifact integrity lists skipped references",
			"archive owner accepts metadata-only historical references",
			"local AreaFlow-owned artifacts still pass hash and size checks",
		})
	case "permission":
		return notAcceptableDecision(action, "permission policy gaps are release blockers and cannot be accepted by preview")
	case "conformance":
		return notAcceptableDecision(action, "adapter/profile conformance gaps are release blockers and cannot be accepted by preview")
	case "backup":
		return notAcceptableDecision(action, "backup manifest gaps are release blockers and cannot be accepted by preview")
	default:
		return notAcceptableDecision(action, fmt.Sprintf("%s remediation has no release acceptance rule", action.Category))
	}
}

func needsDecision(action ReleaseRemediationAction, acceptanceType string, reason string, evidence []string) ReleaseAcceptanceDecision {
	return ReleaseAcceptanceDecision{
		Key:              "accept:" + action.SourceItem,
		SourceAction:     action.Key,
		Category:         action.Category,
		Status:           "needs_decision",
		AcceptanceType:   acceptanceType,
		Owner:            action.Owner,
		Reason:           reason,
		RequiredEvidence: evidence,
		NextCommand:      action.NextCommand,
		Metadata:         acceptanceMetadata(action),
	}
}

func notAcceptableDecision(action ReleaseRemediationAction, reason string) ReleaseAcceptanceDecision {
	return ReleaseAcceptanceDecision{
		Key:              "accept:" + action.SourceItem,
		SourceAction:     action.Key,
		Category:         action.Category,
		Status:           "not_acceptable",
		AcceptanceType:   "none",
		Owner:            action.Owner,
		Reason:           reason,
		RequiredEvidence: []string{action.Acceptance},
		NextCommand:      action.NextCommand,
		Metadata:         acceptanceMetadata(action),
	}
}

func acceptanceMetadata(action ReleaseRemediationAction) map[string]any {
	metadata := copyReleaseMetadata(action.Metadata)
	metadata["remediation_status"] = action.Status
	metadata["remediation_acceptance"] = action.Acceptance
	return metadata
}

func worseAcceptancePreviewStatus(candidate string, current string) bool {
	return acceptancePreviewStatusRank(candidate) > acceptancePreviewStatusRank(current)
}

func acceptancePreviewStatusRank(status string) int {
	switch status {
	case "not_acceptable", "blocked":
		return 3
	case "needs_decision", "needs_attention":
		return 2
	case "ready", "acceptable":
		return 1
	default:
		return 0
	}
}
