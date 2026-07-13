package project

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AuditCoverageOptions struct {
	ProjectID   int64
	ProjectKey  string
	GeneratedAt time.Time
}

type AuditCoverageActionSpec struct {
	Action   string
	Decision string
}

type AuditCoverageActionEvidence struct {
	Action      string
	Decision    string
	Count       int64
	Status      string
	LastAuditAt *time.Time
}

type AuditCoverageRequirement struct {
	Key             string
	Category        string
	Description     string
	Status          string
	EvidenceCount   int64
	RequiredActions []AuditCoverageActionEvidence
	MissingActions  []string
	LastAuditAt     *time.Time
}

type AuditCoverage struct {
	Status              string
	Mode                string
	Scope               string
	ProjectID           int64
	ProjectKey          string
	TotalAuditEvents    int64
	CoveredRequirements int
	GapRequirements     int
	Requirements        []AuditCoverageRequirement
	GeneratedAt         time.Time
}

type auditActionCount struct {
	Action      string
	Decision    string
	Count       int64
	LastAuditAt time.Time
}

type auditCoverageRequirementSpec struct {
	Key         string
	Category    string
	Description string
	Actions     []AuditCoverageActionSpec
}

var auditCoverageRequirementSpecs = []auditCoverageRequirementSpec{
	{
		Key:         "project_registration",
		Category:    "write",
		Description: "project registration and config upsert writes are audited",
		Actions:     []AuditCoverageActionSpec{{Action: "project.upsert", Decision: "allowed"}},
	},
	{
		Key:         "status_mirror_write",
		Category:    "write",
		Description: "managed project status mirror writes are audited",
		Actions:     []AuditCoverageActionSpec{{Action: "status.export", Decision: "allowed"}},
	},
	{
		Key:         "workflow_authoring",
		Category:    "write",
		Description: "AreaFlow-owned workflow version and stage authoring writes are audited",
		Actions: []AuditCoverageActionSpec{
			{Action: "workflow.version.create", Decision: "allowed"},
			{Action: "workflow.stage_skeleton.create", Decision: "allowed"},
			{Action: "workflow.item.mark_ready", Decision: "allowed"},
		},
	},
	{
		Key:         "approval_decision",
		Category:    "approval",
		Description: "explicit workflow approval decisions are audited",
		Actions:     []AuditCoverageActionSpec{{Action: "workflow.approval.record"}},
	},
	{
		Key:         "runner_preview",
		Category:    "execution",
		Description: "dry-run execution preview is audited before real worker execution",
		Actions:     []AuditCoverageActionSpec{{Action: "runner.preview", Decision: "allowed"}},
	},
	{
		Key:         "worker_registration",
		Category:    "worker",
		Description: "worker registration is audited",
		Actions:     []AuditCoverageActionSpec{{Action: "worker.register", Decision: "allowed"}},
	},
	{
		Key:         "worker_capability_denial",
		Category:    "permission",
		Description: "worker capability denial paths are audited",
		Actions:     []AuditCoverageActionSpec{{Action: "worker.run_once", Decision: "denied"}},
	},
	{
		Key:         "worker_lease_lifecycle",
		Category:    "worker",
		Description: "explicit worker lease acquire, release and recovery actions are audited",
		Actions: []AuditCoverageActionSpec{
			{Action: "lease.acquire", Decision: "allowed"},
			{Action: "lease.release", Decision: "allowed"},
			{Action: "lease.recover", Decision: "allowed"},
		},
	},
	{
		Key:         "command_execution",
		Category:    "command",
		Description: "real command execution is audited when enabled",
		Actions:     []AuditCoverageActionSpec{{Action: "command.execute"}},
	},
	{
		Key:         "secret_resolution",
		Category:    "secret",
		Description: "secret reference resolution is audited when enabled",
		Actions:     []AuditCoverageActionSpec{{Action: "secret.resolve"}},
	},
	{
		Key:         "permission_change",
		Category:    "permission",
		Description: "permission policy changes are audited when enabled",
		Actions:     []AuditCoverageActionSpec{{Action: "permission.change"}},
	},
}

func (s Store) AuditCoverage(ctx context.Context, options AuditCoverageOptions) (AuditCoverage, error) {
	options = normalizeAuditCoverageOptions(options)
	total, err := s.auditEventTotal(ctx, options.ProjectID)
	if err != nil {
		return AuditCoverage{}, err
	}
	counts, err := s.auditActionCounts(ctx, options.ProjectID)
	if err != nil {
		return AuditCoverage{}, err
	}
	coverage := BuildAuditCoverage(options, total, counts)
	return coverage, nil
}

func normalizeAuditCoverageOptions(options AuditCoverageOptions) AuditCoverageOptions {
	options.ProjectKey = strings.TrimSpace(options.ProjectKey)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func (s Store) auditEventTotal(ctx context.Context, projectID int64) (int64, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM audit_events
WHERE ($1::bigint = 0 OR project_id = $1)`,
		projectID,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("count audit events: %w", err)
	}
	return total, nil
}

func (s Store) auditActionCounts(ctx context.Context, projectID int64) ([]auditActionCount, error) {
	rows, err := s.pool.Query(ctx, `
SELECT action, decision, COUNT(*), MAX(created_at)
FROM audit_events
WHERE ($1::bigint = 0 OR project_id = $1)
GROUP BY action, decision
ORDER BY action, decision`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit action counts: %w", err)
	}
	defer rows.Close()

	counts := []auditActionCount{}
	for rows.Next() {
		var count auditActionCount
		if err := rows.Scan(&count.Action, &count.Decision, &count.Count, &count.LastAuditAt); err != nil {
			return nil, fmt.Errorf("scan audit action count: %w", err)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit action counts: %w", err)
	}
	return counts, nil
}

func BuildAuditCoverage(options AuditCoverageOptions, total int64, counts []auditActionCount) AuditCoverage {
	options = normalizeAuditCoverageOptions(options)
	scope := "platform"
	if options.ProjectID > 0 || options.ProjectKey != "" {
		scope = "project"
	}
	coverage := AuditCoverage{
		Status:           "pass",
		Mode:             "read_only_audit_coverage",
		Scope:            scope,
		ProjectID:        options.ProjectID,
		ProjectKey:       options.ProjectKey,
		TotalAuditEvents: total,
		Requirements:     make([]AuditCoverageRequirement, 0, len(auditCoverageRequirementSpecs)),
		GeneratedAt:      options.GeneratedAt,
	}
	index := auditActionCountIndex(counts)
	for _, spec := range auditCoverageRequirementSpecs {
		requirement := buildAuditCoverageRequirement(spec, index)
		if requirement.Status == "pass" {
			coverage.CoveredRequirements++
		} else {
			coverage.GapRequirements++
		}
		coverage.Requirements = append(coverage.Requirements, requirement)
	}
	if coverage.GapRequirements > 0 {
		coverage.Status = "warn"
	}
	return coverage
}

func buildAuditCoverageRequirement(spec auditCoverageRequirementSpec, index map[string]auditActionCount) AuditCoverageRequirement {
	requirement := AuditCoverageRequirement{
		Key:             spec.Key,
		Category:        spec.Category,
		Description:     spec.Description,
		Status:          "pass",
		RequiredActions: make([]AuditCoverageActionEvidence, 0, len(spec.Actions)),
	}
	for _, actionSpec := range spec.Actions {
		count := lookupAuditActionCount(index, actionSpec)
		status := "pass"
		var lastAuditAt *time.Time
		if count.Count == 0 {
			status = "gap"
			requirement.Status = "gap"
			requirement.MissingActions = append(requirement.MissingActions, auditActionSpecLabel(actionSpec))
		} else {
			last := count.LastAuditAt
			lastAuditAt = &last
			if requirement.LastAuditAt == nil || last.After(*requirement.LastAuditAt) {
				requirement.LastAuditAt = &last
			}
		}
		requirement.EvidenceCount += count.Count
		requirement.RequiredActions = append(requirement.RequiredActions, AuditCoverageActionEvidence{
			Action:      actionSpec.Action,
			Decision:    actionSpec.Decision,
			Count:       count.Count,
			Status:      status,
			LastAuditAt: lastAuditAt,
		})
	}
	return requirement
}

func auditActionCountIndex(counts []auditActionCount) map[string]auditActionCount {
	index := map[string]auditActionCount{}
	for _, count := range counts {
		index[auditActionKey(count.Action, count.Decision)] = count
		if count.Decision != "" {
			anyKey := auditActionKey(count.Action, "")
			existing := index[anyKey]
			existing.Action = count.Action
			existing.Count += count.Count
			if existing.LastAuditAt.IsZero() || count.LastAuditAt.After(existing.LastAuditAt) {
				existing.LastAuditAt = count.LastAuditAt
			}
			index[anyKey] = existing
		}
	}
	return index
}

func lookupAuditActionCount(index map[string]auditActionCount, spec AuditCoverageActionSpec) auditActionCount {
	count := index[auditActionKey(spec.Action, spec.Decision)]
	count.Action = spec.Action
	count.Decision = spec.Decision
	return count
}

func auditActionKey(action string, decision string) string {
	return strings.TrimSpace(action) + "\x00" + strings.TrimSpace(decision)
}

func auditActionSpecLabel(spec AuditCoverageActionSpec) string {
	if strings.TrimSpace(spec.Decision) == "" {
		return strings.TrimSpace(spec.Action)
	}
	return strings.TrimSpace(spec.Action) + ":" + strings.TrimSpace(spec.Decision)
}
