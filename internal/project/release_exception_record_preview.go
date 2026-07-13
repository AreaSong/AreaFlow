package project

import (
	"context"
	"strings"
	"time"
)

type ReleaseExceptionRecordPreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseExceptionRecordDraft struct {
	Key              string
	SourceGateItem   string
	SourceDecision   string
	AcceptanceType   string
	Status           string
	Owner            string
	Reason           string
	RequiredEvidence []string
	AuditActions     []string
	RollbackPlan     string
	ReviewRequired   bool
	Metadata         map[string]any
}

type ReleaseExceptionRecordPreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Doctor           ReleaseExceptionDoctor
	Drafts           []ReleaseExceptionRecordDraft
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseExceptionRecordPreview(ctx context.Context, options ReleaseExceptionRecordPreviewOptions) (ReleaseExceptionRecordPreview, error) {
	options = normalizeReleaseExceptionRecordPreviewOptions(options)
	doctor, err := s.ReleaseExceptionDoctor(ctx, ReleaseExceptionDoctorOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseExceptionRecordPreview{}, err
	}
	return BuildReleaseExceptionRecordPreview(doctor, options), nil
}

func normalizeReleaseExceptionRecordPreviewOptions(options ReleaseExceptionRecordPreviewOptions) ReleaseExceptionRecordPreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseExceptionRecordPreview(doctor ReleaseExceptionDoctor, options ReleaseExceptionRecordPreviewOptions) ReleaseExceptionRecordPreview {
	options = normalizeReleaseExceptionRecordPreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, doctor.ProjectKey, doctor.Gate.ProjectKey)
	preview := ReleaseExceptionRecordPreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_exception_record_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		Doctor:           doctor,
		Drafts:           []ReleaseExceptionRecordDraft{},
		Capabilities: []string{
			"read_release_exception_doctor",
			"preview_exception_records",
			"preview_exception_audit_plan",
			"preview_exception_rollback_plan",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"mark_gap_accepted",
			"create_approval",
			"insert_exception_record",
			"insert_audit_event",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	for _, item := range doctor.Gate.Items {
		draft := releaseExceptionDraftForGateItem(item)
		preview.addDraft(draft)
	}
	if len(preview.Drafts) == 0 {
		preview.addDraft(ReleaseExceptionRecordDraft{
			Key:            "release_exception:none",
			SourceDecision: "missing",
			Status:         "blocked",
			Owner:          "release_owner",
			Reason:         "release exception record preview has no gate items to draft",
			RollbackPlan:   "rerun release acceptance gate after restoring release readiness inputs",
			Metadata:       map[string]any{"doctor_status": doctor.Status},
		})
	}
	return preview
}

func (p *ReleaseExceptionRecordPreview) addDraft(draft ReleaseExceptionRecordDraft) {
	if draft.Metadata == nil {
		draft.Metadata = map[string]any{}
	}
	if len(draft.AuditActions) == 0 {
		draft.AuditActions = []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"}
	}
	p.Drafts = append(p.Drafts, draft)
	if worseReleaseExceptionRecordPreviewStatus(draft.Status, p.Status) {
		p.Status = draft.Status
	}
}

func releaseExceptionDraftForGateItem(item ReleaseAcceptanceGateItem) ReleaseExceptionRecordDraft {
	metadata := copyReleaseMetadata(item.Metadata)
	metadata["gate_item_status"] = item.Status
	metadata["decision_status"] = item.DecisionStatus
	metadata["exception_writable"] = false
	key := "release_exception:" + strings.TrimPrefix(item.Key, "gate:accept:")
	switch item.DecisionStatus {
	case "needs_decision":
		return ReleaseExceptionRecordDraft{
			Key:              key,
			SourceGateItem:   item.Key,
			SourceDecision:   item.DecisionStatus,
			AcceptanceType:   item.AcceptanceType,
			Status:           "draft",
			Owner:            item.Owner,
			Reason:           item.Message,
			RequiredEvidence: item.RequiredEvidence,
			AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
			RollbackPlan:     "revoke the exception record and rerun release acceptance gate before release apply",
			ReviewRequired:   true,
			Metadata:         metadata,
		}
	case "ready":
		return ReleaseExceptionRecordDraft{
			Key:              key,
			SourceGateItem:   item.Key,
			SourceDecision:   item.DecisionStatus,
			AcceptanceType:   item.AcceptanceType,
			Status:           "not_required",
			Owner:            item.Owner,
			Reason:           "release acceptance gate item already passes",
			RequiredEvidence: item.RequiredEvidence,
			AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
			RollbackPlan:     "no exception rollback required",
			ReviewRequired:   false,
			Metadata:         metadata,
		}
	default:
		return ReleaseExceptionRecordDraft{
			Key:              key,
			SourceGateItem:   item.Key,
			SourceDecision:   item.DecisionStatus,
			AcceptanceType:   item.AcceptanceType,
			Status:           "blocked",
			Owner:            item.Owner,
			Reason:           "release gate item cannot produce an exception record until the blocker is remediated",
			RequiredEvidence: item.RequiredEvidence,
			AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
			RollbackPlan:     "repair the source blocker and regenerate the exception record preview",
			ReviewRequired:   true,
			Metadata:         metadata,
		}
	}
}

func worseReleaseExceptionRecordPreviewStatus(candidate string, current string) bool {
	return releaseExceptionRecordPreviewStatusRank(candidate) > releaseExceptionRecordPreviewStatusRank(current)
}

func releaseExceptionRecordPreviewStatusRank(status string) int {
	switch status {
	case "blocked", "fail":
		return 3
	case "draft", "needs_attention":
		return 2
	case "ready", "not_required", "pass":
		return 1
	default:
		return 0
	}
}
