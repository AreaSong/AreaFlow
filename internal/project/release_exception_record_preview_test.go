package project

import (
	"testing"
	"time"
)

func TestBuildReleaseExceptionRecordPreviewDraftsPendingExceptions(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	doctor := ReleaseExceptionDoctor{
		Status: "warn",
		Mode:   "read_only_release_exception_doctor",
		Gate: ReleaseAcceptanceGate{
			Status: "blocked",
			Items: []ReleaseAcceptanceGateItem{
				{
					Key:              "gate:accept:restore_plan",
					Category:         "restore",
					Status:           "blocked",
					DecisionStatus:   "needs_decision",
					AcceptanceType:   "metadata_only_history",
					Message:          "explicit release acceptance evidence is required before this exception can pass",
					Owner:            "release_owner",
					RequiredEvidence: []string{"release notes state metadata-only artifacts"},
					NextCommand:      "areaflow backup restore-plan --json",
					Metadata:         map[string]any{"restore_status": "needs_attention"},
				},
				{
					Key:              "gate:accept:audit_coverage",
					Category:         "audit",
					Status:           "blocked",
					DecisionStatus:   "needs_decision",
					AcceptanceType:   "future_only_gap",
					Message:          "explicit release acceptance evidence is required before this exception can pass",
					Owner:            "platform_owner",
					RequiredEvidence: []string{"audit coverage lists missing actions"},
					NextCommand:      "areaflow audit coverage --json",
					Metadata:         map[string]any{"gap_requirements": 3},
				},
			},
		},
	}

	preview := BuildReleaseExceptionRecordPreview(doctor, ReleaseExceptionRecordPreviewOptions{GeneratedAt: created})

	if preview.Status != "draft" || preview.Mode != "read_only_release_exception_record_preview" {
		t.Fatalf("unexpected record preview: %+v", preview)
	}
	if len(preview.Drafts) != 2 {
		t.Fatalf("draft count = %d, want 2: %+v", len(preview.Drafts), preview.Drafts)
	}
	assertReleaseExceptionDraft(t, preview, "release_exception:restore_plan", "draft", "metadata_only_history", true)
	assertReleaseExceptionDraft(t, preview, "release_exception:audit_coverage", "draft", "future_only_gap", true)
	if preview.Drafts[0].RollbackPlan == "" || len(preview.Drafts[0].AuditActions) != 3 {
		t.Fatalf("draft missing audit or rollback plan: %+v", preview.Drafts[0])
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "write_database" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleaseExceptionRecordPreviewMarksReadyItemsNotRequired(t *testing.T) {
	preview := BuildReleaseExceptionRecordPreview(
		ReleaseExceptionDoctor{
			Status: "warn",
			Gate: ReleaseAcceptanceGate{
				Status: "pass",
				Items: []ReleaseAcceptanceGateItem{
					{
						Key:              "gate:accept:release_readiness",
						Category:         "release",
						Status:           "pass",
						DecisionStatus:   "ready",
						AcceptanceType:   "none",
						Owner:            "release_owner",
						RequiredEvidence: []string{"release readiness remains ready"},
					},
				},
			},
		},
		ReleaseExceptionRecordPreviewOptions{},
	)

	if preview.Status != "ready" {
		t.Fatalf("expected ready status: %+v", preview)
	}
	assertReleaseExceptionDraft(t, preview, "release_exception:release_readiness", "not_required", "none", false)
}

func TestBuildReleaseExceptionRecordPreviewBlocksNotAcceptableItems(t *testing.T) {
	preview := BuildReleaseExceptionRecordPreview(
		ReleaseExceptionDoctor{
			Status: "fail",
			Gate: ReleaseAcceptanceGate{
				Status: "blocked",
				Items: []ReleaseAcceptanceGateItem{
					{
						Key:              "gate:accept:permission_policy:areamatrix",
						Category:         "permission",
						Status:           "blocked",
						DecisionStatus:   "not_acceptable",
						AcceptanceType:   "none",
						Owner:            "security_owner",
						RequiredEvidence: []string{"permission policy doctor returns pass"},
					},
				},
			},
		},
		ReleaseExceptionRecordPreviewOptions{},
	)

	if preview.Status != "blocked" {
		t.Fatalf("expected blocked status: %+v", preview)
	}
	assertReleaseExceptionDraft(t, preview, "release_exception:permission_policy:areamatrix", "blocked", "none", true)
}

func TestBuildReleaseExceptionRecordPreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleaseExceptionRecordPreview(
		ReleaseExceptionDoctor{
			Status:     "warn",
			Scope:      "project",
			ProjectKey: "areamatrix",
			Gate: ReleaseAcceptanceGate{
				Scope:      "project",
				ProjectKey: "areamatrix",
			},
		},
		ReleaseExceptionRecordPreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped record preview: %+v", preview)
	}
	if preview.Doctor.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested doctor to keep project key: %+v", preview.Doctor)
	}
}

func assertReleaseExceptionDraft(t *testing.T, preview ReleaseExceptionRecordPreview, key string, status string, acceptanceType string, reviewRequired bool) {
	t.Helper()
	for _, draft := range preview.Drafts {
		if draft.Key == key {
			if draft.Status != status || draft.AcceptanceType != acceptanceType || draft.ReviewRequired != reviewRequired {
				t.Fatalf("draft %s = %+v, want status=%s acceptance_type=%s review=%t", key, draft, status, acceptanceType, reviewRequired)
			}
			if draft.Owner == "" || draft.Reason == "" || draft.RollbackPlan == "" || len(draft.AuditActions) == 0 {
				t.Fatalf("draft %s missing guidance: %+v", key, draft)
			}
			return
		}
	}
	t.Fatalf("draft %s not found: %+v", key, preview.Drafts)
}
