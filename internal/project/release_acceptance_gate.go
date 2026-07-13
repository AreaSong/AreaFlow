package project

import (
	"context"
	"encoding/json"
	"time"

	"github.com/areasong/areaflow/internal/migrate"
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
	scope, err := s.resolveReleaseProjectScope(ctx, options.ProjectID, options.ProjectKey)
	if err != nil {
		return ReleaseAcceptanceGate{}, err
	}
	exceptions := []ReleaseExceptionRecord{}
	if scope.ProjectID > 0 {
		approval, approvalErr := migrate.Approval(ctx, s.pool, migrate.ReleaseExceptionMigrationName)
		if approvalErr != nil {
			return ReleaseAcceptanceGate{}, approvalErr
		}
		effective, effectiveErr := migrate.ApprovalEffective(migrate.ReleaseExceptionMigrationName, approval)
		if effectiveErr != nil {
			return ReleaseAcceptanceGate{}, effectiveErr
		}
		if effective && approval.Applied {
			exceptions, err = s.EffectiveReleaseExceptions(ctx, scope.ProjectID, options.GeneratedAt)
			if err != nil {
				return ReleaseAcceptanceGate{}, err
			}
		}
	}
	return BuildReleaseAcceptanceGateWithExceptions(preview, options, exceptions), nil
}

func normalizeReleaseAcceptanceGateOptions(options ReleaseAcceptanceGateOptions) ReleaseAcceptanceGateOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseAcceptanceGate(preview ReleaseAcceptancePreview, options ReleaseAcceptanceGateOptions) ReleaseAcceptanceGate {
	return BuildReleaseAcceptanceGateWithExceptions(preview, options, nil)
}

func BuildReleaseAcceptanceGateWithExceptions(preview ReleaseAcceptancePreview, options ReleaseAcceptanceGateOptions, exceptions []ReleaseExceptionRecord) ReleaseAcceptanceGate {
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
		gate.addItem(acceptanceGateItemForDecision(decision, matchingEffectiveReleaseException(decision, exceptions)))
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

func acceptanceGateItemForDecision(decision ReleaseAcceptanceDecision, exception *ReleaseExceptionRecord) ReleaseAcceptanceGateItem {
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
		if exception != nil {
			metadata["approved_exception_key"] = exception.ExceptionKey
			metadata["approved_exception_id"] = exception.ID
			metadata["approved_exception_expires_at"] = exception.ExpiresAt
			metadata["source_decision_status"] = decision.Status
			return ReleaseAcceptanceGateItem{
				Key:              "gate:" + decision.Key,
				Category:         decision.Category,
				Status:           "pass",
				DecisionStatus:   "ready",
				AcceptanceType:   decision.AcceptanceType,
				Message:          "release acceptance decision is covered by an effective approved exception",
				Owner:            exception.Owner,
				RequiredEvidence: []string{"approved release exception remains effective and matches the current gate item"},
				NextCommand:      "areaflow release acceptance-gate --json",
				Metadata:         metadata,
			}
		}
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

func matchingEffectiveReleaseException(decision ReleaseAcceptanceDecision, exceptions []ReleaseExceptionRecord) *ReleaseExceptionRecord {
	gateItem := "gate:" + decision.Key
	expectedFingerprint := releaseExceptionSourceFingerprint(gateItem, decision.Category, decision.AcceptanceType, decision.Owner, decision.RequiredEvidence)
	for i := range exceptions {
		exception := &exceptions[i]
		fingerprint, _ := exception.Metadata["source_fingerprint"].(string)
		if exception.Status == "approved" && exception.SourceGateItem == gateItem &&
			exception.AcceptanceType == decision.AcceptanceType && fingerprint == expectedFingerprint {
			return exception
		}
	}
	return nil
}

func releaseExceptionSourceFingerprint(gateItem string, category string, acceptanceType string, owner string, requiredEvidence []string) string {
	payload, _ := json.Marshal(map[string]any{
		"gate_item": gateItem, "category": category, "acceptance_type": acceptanceType,
		"owner": owner, "required_evidence": requiredEvidence,
	})
	return releaseExceptionSHA256Hex(payload)
}
