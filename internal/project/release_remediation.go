package project

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ReleaseRemediationOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseRemediationAction struct {
	Key               string
	Category          string
	Status            string
	SourceItem        string
	RecommendedAction string
	Rationale         string
	Owner             string
	NextCommand       string
	Acceptance        string
	Metadata          map[string]any
}

type ReleaseRemediationPlan struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Readiness        ReleaseReadiness
	Actions          []ReleaseRemediationAction
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseRemediationPlan(ctx context.Context, options ReleaseRemediationOptions) (ReleaseRemediationPlan, error) {
	options = normalizeReleaseRemediationOptions(options)
	readiness, err := s.ReleaseReadiness(ctx, ReleaseReadinessOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseRemediationPlan{}, err
	}
	return BuildReleaseRemediationPlan(readiness, options), nil
}

func normalizeReleaseRemediationOptions(options ReleaseRemediationOptions) ReleaseRemediationOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseRemediationPlan(readiness ReleaseReadiness, options ReleaseRemediationOptions) ReleaseRemediationPlan {
	options = normalizeReleaseRemediationOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, readiness.ProjectKey)
	plan := ReleaseRemediationPlan{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_remediation_plan",
		Scope:            scope,
		ProjectKey:       projectKey,
		Readiness:        readiness,
		Actions:          []ReleaseRemediationAction{},
		Capabilities: []string{
			"read_release_readiness",
			"classify_release_gaps",
			"generate_remediation_plan",
		},
		ForbiddenActions: []string{
			"write_project_files",
			"write_database",
			"write_artifact_store",
			"resolve_secrets",
			"execute_commands",
			"apply_restore",
			"apply_cutover",
			"start_worker",
			"mark_gap_accepted",
		},
		GeneratedAt: options.GeneratedAt,
	}
	for _, item := range readiness.Items {
		if item.Status == "ready" {
			continue
		}
		action := remediationActionForItem(item)
		plan.addAction(action)
	}
	if len(plan.Actions) == 0 && readiness.Status == "ready" {
		plan.addAction(ReleaseRemediationAction{
			Key:               "release_ready",
			Category:          "release",
			Status:            "ready",
			SourceItem:        "release_readiness",
			RecommendedAction: "no remediation required",
			Rationale:         "release readiness has no blocked or needs_attention items",
			Owner:             "release_owner",
			Acceptance:        "release readiness remains ready",
			Metadata:          map[string]any{"readiness_status": readiness.Status},
		})
	}
	return plan
}

func (p *ReleaseRemediationPlan) addAction(action ReleaseRemediationAction) {
	if action.Metadata == nil {
		action.Metadata = map[string]any{}
	}
	p.Actions = append(p.Actions, action)
	if worseRemediationStatus(action.Status, p.Status) {
		p.Status = action.Status
	}
}

func remediationActionForItem(item ReleaseReadinessItem) ReleaseRemediationAction {
	switch item.Category {
	case "restore":
		return remediationActionForRestore(item)
	case "audit":
		return remediationActionForAudit(item)
	case "artifact":
		return remediationActionForArtifact(item)
	case "permission":
		return remediationActionForPermission(item)
	case "conformance":
		return remediationActionForConformance(item)
	case "backup":
		return remediationActionForBackup(item)
	default:
		return remediationActionGeneric(item)
	}
}

func remediationActionForRestore(item ReleaseReadinessItem) ReleaseRemediationAction {
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          "restore",
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "decide whether historical project reference artifacts stay metadata-only or are copied into the AreaFlow artifact store",
		Rationale:         "restore planning cannot claim full recovery while artifact originals remain outside AreaFlow-owned storage",
		Owner:             "release_owner",
		NextCommand:       "areaflow backup restore-plan --json",
		Acceptance:        "restore plan is ready, or release notes explicitly accept metadata-only historical artifacts",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationActionForAudit(item ReleaseReadinessItem) ReleaseRemediationAction {
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          "audit",
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "close enabled-capability audit gaps and document future-only gaps before release",
		Rationale:         "v1.0 release readiness must distinguish missing evidence from intentionally disabled long-term capabilities",
		Owner:             "platform_owner",
		NextCommand:       "areaflow audit coverage --json",
		Acceptance:        "audit coverage is pass, or release readiness records accepted future-only audit gaps with owners",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationActionForArtifact(item ReleaseReadinessItem) ReleaseRemediationAction {
	projectKey := stringFromAny(item.Metadata["project_key"])
	nextCommand := "areaflow artifact integrity <project> --json"
	if projectKey != "" {
		nextCommand = fmt.Sprintf("areaflow artifact integrity %s --json", projectKey)
	}
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          "artifact",
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "verify local artifacts and choose an archive policy for skipped project-reference originals",
		Rationale:         "release evidence must not treat unverified external project references as fully restorable artifact content",
		Owner:             "artifact_owner",
		NextCommand:       nextCommand,
		Acceptance:        "artifact integrity is pass, or skipped references are explicitly accepted with archive ownership",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationActionForPermission(item ReleaseReadinessItem) ReleaseRemediationAction {
	projectKey := stringFromAny(item.Metadata["project_key"])
	nextCommand := "areaflow permissions doctor <project> --json"
	if projectKey != "" {
		nextCommand = fmt.Sprintf("areaflow permissions doctor %s --json", projectKey)
	}
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          "permission",
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "repair permission policy failures or record explicit release-blocking security decisions",
		Rationale:         "release cannot proceed if write, command, secret, network, git or worker permissions are ambiguous",
		Owner:             "security_owner",
		NextCommand:       nextCommand,
		Acceptance:        "permission policy doctor returns pass for every release project",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationActionForConformance(item ReleaseReadinessItem) ReleaseRemediationAction {
	projectKey := stringFromAny(item.Metadata["project_key"])
	nextCommand := "areaflow conformance check <project> --json"
	if projectKey != "" {
		nextCommand = fmt.Sprintf("areaflow conformance check %s --json", projectKey)
	}
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          "conformance",
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "repair adapter/profile binding, profile contract, or read-only snapshot boundary",
		Rationale:         "AreaFlow core must not ship with ambiguous adapter/profile ownership boundaries",
		Owner:             "platform_owner",
		NextCommand:       nextCommand,
		Acceptance:        "adapter/profile conformance returns pass for every release project",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationActionForBackup(item ReleaseReadinessItem) ReleaseRemediationAction {
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          "backup",
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "repair backup metadata enumeration before release packaging",
		Rationale:         "release cannot proceed if PostgreSQL metadata or artifact metadata cannot be enumerated",
		Owner:             "release_owner",
		NextCommand:       "areaflow backup manifest --json",
		Acceptance:        "backup manifest returns ready with stable manifest_hash",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationActionGeneric(item ReleaseReadinessItem) ReleaseRemediationAction {
	return ReleaseRemediationAction{
		Key:               "remediate:" + item.Key,
		Category:          item.Category,
		Status:            remediationStatusFromReadiness(item.Status),
		SourceItem:        item.Key,
		RecommendedAction: "review and close release readiness item",
		Rationale:         item.Message,
		Owner:             "release_owner",
		Acceptance:        "source release readiness item becomes ready or is explicitly accepted",
		Metadata:          copyReleaseMetadata(item.Metadata),
	}
}

func remediationStatusFromReadiness(status string) string {
	switch status {
	case "blocked":
		return "blocked"
	case "needs_attention", "warn", "skipped":
		return "needs_attention"
	case "ready", "pass":
		return "ready"
	default:
		return "blocked"
	}
}

func worseRemediationStatus(candidate string, current string) bool {
	return releaseReadinessStatusRank(candidate) > releaseReadinessStatusRank(current)
}

func copyReleaseMetadata(metadata map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
