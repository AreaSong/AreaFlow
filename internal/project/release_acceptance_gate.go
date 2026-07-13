package project

import (
	"context"
	"time"
)

type ReleaseAcceptanceGateOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseAcceptanceGateItem struct {
	Key              string
	Category         string
	Status           string
	DecisionStatus   string
	AcceptanceType   string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseAcceptanceGate struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Preview          ReleaseAcceptancePreview
	Items            []ReleaseAcceptanceGateItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseAcceptanceGate(ctx context.Context, options ReleaseAcceptanceGateOptions) (ReleaseAcceptanceGate, error) {
	options = normalizeReleaseAcceptanceGateOptions(options)
	preview, err := s.ReleaseAcceptancePreview(ctx, ReleaseAcceptancePreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseAcceptanceGate{}, err
	}
	return BuildReleaseAcceptanceGate(preview, options), nil
}

func normalizeReleaseAcceptanceGateOptions(options ReleaseAcceptanceGateOptions) ReleaseAcceptanceGateOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseAcceptanceGate(preview ReleaseAcceptancePreview, options ReleaseAcceptanceGateOptions) ReleaseAcceptanceGate {
	options = normalizeReleaseAcceptanceGateOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, preview.ProjectKey, preview.Remediation.ProjectKey)
	gate := ReleaseAcceptanceGate{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "pass",
		Mode:             "read_only_release_acceptance_gate",
		Scope:            scope,
		ProjectKey:       projectKey,
		Preview:          preview,
		Items:            []ReleaseAcceptanceGateItem{},
		Capabilities: []string{
			"read_release_acceptance_preview",
			"evaluate_release_acceptance_gate",
			"report_acceptance_blockers",
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
	for _, decision := range preview.Decisions {
		gate.addItem(acceptanceGateItemForDecision(decision))
	}
	if len(gate.Items) == 0 {
		gate.addItem(ReleaseAcceptanceGateItem{
			Key:              "release_acceptance",
			Category:         "release",
			Status:           "blocked",
			DecisionStatus:   "missing",
			AcceptanceType:   "none",
			Message:          "release acceptance preview produced no decisions",
			Owner:            "release_owner",
			RequiredEvidence: []string{"release acceptance preview contains at least one decision"},
			Metadata:         map[string]any{"preview_status": preview.Status},
		})
	}
	return gate
}

func (g *ReleaseAcceptanceGate) addItem(item ReleaseAcceptanceGateItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	if item.AcceptanceType == "" {
		item.AcceptanceType = "none"
	}
	g.Items = append(g.Items, item)
	if item.Status == "blocked" {
		g.Status = "blocked"
	}
}

func acceptanceGateItemForDecision(decision ReleaseAcceptanceDecision) ReleaseAcceptanceGateItem {
	metadata := copyReleaseMetadata(decision.Metadata)
	metadata["decision_status"] = decision.Status
	metadata["source_action"] = decision.SourceAction
	switch decision.Status {
	case "ready":
		return ReleaseAcceptanceGateItem{
			Key:              "gate:" + decision.Key,
			Category:         decision.Category,
			Status:           "pass",
			DecisionStatus:   decision.Status,
			AcceptanceType:   decision.AcceptanceType,
			Message:          "release acceptance decision is ready",
			Owner:            decision.Owner,
			RequiredEvidence: decision.RequiredEvidence,
			NextCommand:      decision.NextCommand,
			Metadata:         metadata,
		}
	case "needs_decision":
		return ReleaseAcceptanceGateItem{
			Key:              "gate:" + decision.Key,
			Category:         decision.Category,
			Status:           "blocked",
			DecisionStatus:   decision.Status,
			AcceptanceType:   decision.AcceptanceType,
			Message:          "explicit release acceptance evidence is required before this exception can pass",
			Owner:            decision.Owner,
			RequiredEvidence: decision.RequiredEvidence,
			NextCommand:      decision.NextCommand,
			Metadata:         metadata,
		}
	case "not_acceptable":
		return ReleaseAcceptanceGateItem{
			Key:              "gate:" + decision.Key,
			Category:         decision.Category,
			Status:           "blocked",
			DecisionStatus:   decision.Status,
			AcceptanceType:   decision.AcceptanceType,
			Message:          "release blocker cannot be accepted and must be remediated",
			Owner:            decision.Owner,
			RequiredEvidence: decision.RequiredEvidence,
			NextCommand:      decision.NextCommand,
			Metadata:         metadata,
		}
	default:
		return ReleaseAcceptanceGateItem{
			Key:              "gate:" + decision.Key,
			Category:         decision.Category,
			Status:           "blocked",
			DecisionStatus:   decision.Status,
			AcceptanceType:   decision.AcceptanceType,
			Message:          "release acceptance decision has an unknown status",
			Owner:            decision.Owner,
			RequiredEvidence: decision.RequiredEvidence,
			NextCommand:      decision.NextCommand,
			Metadata:         metadata,
		}
	}
}
