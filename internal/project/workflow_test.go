package project

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateWorkflowVersionLabel(t *testing.T) {
	valid := []string{"v2", "v1-mvp", "v_template", "2026.06"}
	for _, label := range valid {
		t.Run(label, func(t *testing.T) {
			if err := ValidateWorkflowVersionLabel(label); err != nil {
				t.Fatalf("label should be valid: %v", err)
			}
		})
	}

	invalid := []string{"", "-v2", "v/2", "v 2", "v2!", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm"}
	for _, label := range invalid {
		t.Run(label, func(t *testing.T) {
			err := ValidateWorkflowVersionLabel(label)
			if !errors.Is(err, ErrInvalidWorkflowVersionLabel) {
				t.Fatalf("error = %v, want ErrInvalidWorkflowVersionLabel", err)
			}
		})
	}
}

func TestNormalizeCreateWorkflowVersionOptions(t *testing.T) {
	record := Record{Key: "areamatrix"}
	options := normalizeCreateWorkflowVersionOptions(record, CreateWorkflowVersionOptions{
		DisplayLabel: " v2 ",
	})

	if options.DisplayLabel != "v2" {
		t.Fatalf("label = %q, want v2", options.DisplayLabel)
	}
	if options.Actor != "local-user" {
		t.Fatalf("actor = %q, want local-user", options.Actor)
	}
	if options.Reason == "" {
		t.Fatal("reason should default")
	}
	if options.IdempotencyKey != "workflow.version.create:areamatrix:v2" {
		t.Fatalf("idempotency key = %q", options.IdempotencyKey)
	}
}

func TestWorkflowVersionRequestHashIgnoresReason(t *testing.T) {
	record := Record{Key: "areamatrix"}
	first, err := workflowVersionRequestHash(record, CreateWorkflowVersionOptions{
		DisplayLabel: "v2",
		Reason:       "first",
		ProfileBinding: ProfileBinding{
			ProfileID:      "areamatrix",
			ProfileVersion: 0,
			ProfileHash:    "hash-a",
		},
	})
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := workflowVersionRequestHash(record, CreateWorkflowVersionOptions{
		DisplayLabel: "v2",
		Reason:       "second",
		ProfileBinding: ProfileBinding{
			ProfileID:      "areamatrix",
			ProfileVersion: 0,
			ProfileHash:    "hash-a",
		},
	})
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("hash should ignore reason: %s != %s", first, second)
	}
}

func TestWorkflowVersionRequestHashIncludesProfileBinding(t *testing.T) {
	record := Record{Key: "areamatrix"}
	first, err := workflowVersionRequestHash(record, CreateWorkflowVersionOptions{
		DisplayLabel: "v2",
		ProfileBinding: ProfileBinding{
			ProfileID:      "areamatrix",
			ProfileVersion: 0,
			ProfileHash:    "hash-a",
		},
	})
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := workflowVersionRequestHash(record, CreateWorkflowVersionOptions{
		DisplayLabel: "v2",
		ProfileBinding: ProfileBinding{
			ProfileID:      "areamatrix",
			ProfileVersion: 0,
			ProfileHash:    "hash-b",
		},
	})
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first == second {
		t.Fatalf("hash should include profile binding: %s", first)
	}
}

func TestProfileBindingMetadata(t *testing.T) {
	metadata := profileBindingMetadata(ProfileBinding{
		ProfileID:      "areamatrix",
		ProfileVersion: 2,
		ProfileHash:    "abc123",
		ProfilePath:    "workflow/profiles/areamatrix/profile.yaml",
	})
	if metadata["profile_id"] != "areamatrix" || metadata["profile_version"] != 2 || metadata["profile_hash"] != "abc123" {
		t.Fatalf("unexpected profile binding metadata: %+v", metadata)
	}
}

func TestEvaluateProfileBindingDriftGatePassesWhenBindingMatches(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	binding := ProfileBinding{
		ProfileID:      "areamatrix",
		ProfileVersion: 0,
		ProfileHash:    "hash-a",
		ProfilePath:    "workflow/profiles/areamatrix/profile.yaml",
	}

	result := evaluateProfileBindingDriftGate(record, version, binding, true, binding, true, "", workflowGateSpecs["profile_binding_drift"], RunGateOptions{
		Actor:  "local-user",
		Reason: "test",
	})

	if result.Status != "pass" {
		t.Fatalf("status = %q, want pass: %+v", result.Status, result)
	}
	if result.Inputs["profile_migration_done"] != false {
		t.Fatalf("gate should not migrate profiles: %+v", result.Inputs)
	}
}

func TestEvaluateProfileBindingDriftGateBlocksWhenBindingMissing(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}

	result := evaluateProfileBindingDriftGate(record, version, ProfileBinding{}, false, ProfileBinding{}, false, "", workflowGateSpecs["profile_binding_drift"], RunGateOptions{})

	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if len(result.Failures) == 0 {
		t.Fatal("expected missing binding failure")
	}
}

func TestEvaluateProfileBindingDriftGateWarnsWhenHashDiffers(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	frozen := ProfileBinding{ProfileID: "areamatrix", ProfileVersion: 0, ProfileHash: "hash-a"}
	current := ProfileBinding{ProfileID: "areamatrix", ProfileVersion: 0, ProfileHash: "hash-b"}

	result := evaluateProfileBindingDriftGate(record, version, frozen, true, current, true, "", workflowGateSpecs["profile_binding_drift"], RunGateOptions{})

	if result.Status != "warn" {
		t.Fatalf("status = %q, want warn: %+v", result.Status, result)
	}
	if len(result.Warnings) < 2 {
		t.Fatalf("warnings = %+v, want drift warning", result.Warnings)
	}
}

func TestWorkflowVersionProfileBindingReadsStatusSummary(t *testing.T) {
	version := WorkflowVersion{
		StatusSummary: map[string]any{
			"profile_binding": map[string]any{
				"profile_id":      "areamatrix",
				"profile_version": float64(2),
				"profile_hash":    "abc",
				"profile_path":    "profile.yaml",
			},
		},
	}

	binding, found := workflowVersionProfileBinding(version)
	if !found {
		t.Fatal("expected profile binding")
	}
	if binding.ProfileID != "areamatrix" || binding.ProfileVersion != 2 || binding.ProfileHash != "abc" {
		t.Fatalf("unexpected binding: %+v", binding)
	}
}

func TestEvaluateWorkflowGateBlocksWhenRequiredItemMissing(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}

	result := evaluateWorkflowGate(record, version, nil, workflowGateSpecs["plan_doctor"], RunGateOptions{})

	if result.GateName != "plan_doctor" {
		t.Fatalf("gate name = %q, want plan_doctor", result.GateName)
	}
	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if len(result.Failures) == 0 {
		t.Fatal("expected missing item failures")
	}
}

func TestEvaluateWorkflowGateFailsForSkeletonPlaceholders(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	items := []WorkflowItem{
		skeletonWorkflowItem(10, version.ID, "plans", "plan"),
		skeletonWorkflowItem(11, version.ID, "drafts", "draft_manifest"),
		skeletonWorkflowItem(12, version.ID, "drafts", "draft_copy"),
		skeletonWorkflowItem(13, version.ID, "drafts", "draft_verify"),
	}

	result := evaluateWorkflowGate(record, version, items, workflowGateSpecs["draft_doctor"], RunGateOptions{})

	if result.Status != "fail" {
		t.Fatalf("status = %q, want fail", result.Status)
	}
	if result.WorkflowItemID != 11 {
		t.Fatalf("workflow item id = %d, want draft manifest id", result.WorkflowItemID)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures = %+v, want placeholder failure", result.Failures)
	}
}

func TestEvaluateWorkflowGatePassesForNonPlaceholderItems(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	items := []WorkflowItem{
		readyWorkflowItem(10, version.ID, "queue", "queue_candidate", "hash-a"),
		readyWorkflowItem(11, version.ID, "promotion_preview", "promotion_preview", "hash-b"),
	}

	result := evaluateWorkflowGate(record, version, items, workflowGateSpecs["promotion_preview"], RunGateOptions{
		Actor:  "local-user",
		Reason: "test",
	})

	if result.Status != "pass" {
		t.Fatalf("status = %q, want pass: %+v", result.Status, result)
	}
	if len(result.Failures) != 0 {
		t.Fatalf("failures = %+v, want none", result.Failures)
	}
	if len(result.SourceHashes) != 2 {
		t.Fatalf("source hashes = %+v, want two entries", result.SourceHashes)
	}
}

func TestAuthoredStageSkeletonLinksCoverTracePath(t *testing.T) {
	if len(authoredStageSkeletonLinks) == 0 {
		t.Fatal("expected skeleton trace links")
	}
	knownItems := map[string]bool{}
	for _, spec := range authoredStageSkeleton {
		knownItems[skeletonItemKey(spec.Stage, spec.ItemType)] = true
	}
	seen := map[string]bool{}
	for _, spec := range authoredStageSkeletonLinks {
		if spec.RelationType != "derives_from" {
			t.Fatalf("relation type = %q, want derives_from", spec.RelationType)
		}
		from := skeletonItemKey(spec.FromStage, spec.FromItemType)
		to := skeletonItemKey(spec.ToStage, spec.ToItemType)
		if !knownItems[from] {
			t.Fatalf("unknown link source: %s", from)
		}
		if !knownItems[to] {
			t.Fatalf("unknown link target: %s", to)
		}
		id := skeletonLinkID(spec)
		if seen[id] {
			t.Fatalf("duplicate skeleton link: %s", id)
		}
		seen[id] = true
	}
	if !seen["queue:queue_candidate->promotion_preview:promotion_preview"] {
		t.Fatalf("missing queue to promotion preview trace: %+v", authoredStageSkeletonLinks)
	}
}

func TestEvaluateWorkflowTransitionPreviewBlocksWithoutGate(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}

	preview := evaluateWorkflowTransitionPreview(record, version, GateResult{}, false, PreviewTransitionOptions{})

	if preview.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", preview.Status)
	}
	if len(preview.Blockers) == 0 {
		t.Fatal("expected missing gate blocker")
	}
}

func TestEvaluateWorkflowTransitionPreviewReadyWhenPromotionPreviewPassed(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	gate := GateResult{ID: 9, GateName: "promotion_preview", Status: "pass"}

	preview := evaluateWorkflowTransitionPreview(record, version, gate, true, PreviewTransitionOptions{
		FromStage: "promotion_preview",
		ToStage:   "approval",
		Actor:     "local-user",
		Reason:    "test",
	})

	if preview.Status != "ready" {
		t.Fatalf("status = %q, want ready", preview.Status)
	}
	if preview.GateResultID != 9 {
		t.Fatalf("gate result id = %d, want 9", preview.GateResultID)
	}
}

func TestBuildApprovalRecordCarriesPreviewMetadata(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	preview := WorkflowTransitionPreview{ID: 9, Status: "ready", FromStage: "promotion_preview", ToStage: "approval"}

	approval := buildApprovalRecord(record, version, preview, true, CreateApprovalOptions{
		Decision:     "approved",
		ApprovalKind: "workflow_transition",
		Actor:        "local-user",
		Reason:       "explicit approval",
		RiskLevel:    "normal",
		Metadata:     map[string]any{"note": "test"},
	})

	if approval.TransitionPreviewID != 9 || approval.Decision != "approved" {
		t.Fatalf("unexpected approval: %+v", approval)
	}
	if approval.Metadata["approval_is_execution"] != false {
		t.Fatalf("approval should not be execution metadata: %+v", approval.Metadata)
	}
	if approval.Metadata["note"] != "test" {
		t.Fatalf("metadata not carried: %+v", approval.Metadata)
	}
}

func TestApprovalRecordRequestHashAndIdempotencyKey(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	approval := ApprovalRecord{
		ProjectID:           record.ID,
		WorkflowVersionID:   version.ID,
		TransitionPreviewID: 4,
		ApprovalKind:        "workflow_transition",
		Decision:            "approved",
		ScopeType:           "workflow_version",
		ScopeID:             "v2",
		Actor:               "local-user",
		Reason:              "approve",
		RiskLevel:           "normal",
		Metadata:            map[string]any{"phase": "v0.3d"},
	}

	first, err := approvalRecordRequestHash(record, version, approval)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := approvalRecordRequestHash(record, version, approval)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("approval request hash differed: %s != %s", first, second)
	}

	key := approvalRecordIdempotencyKey(record, version, approval, first)
	if !strings.HasPrefix(key, "workflow.approval.record:areamatrix:v2:approved:4:") {
		t.Fatalf("unexpected approval idempotency key: %s", key)
	}

	approval.Decision = "rejected"
	changed, err := approvalRecordRequestHash(record, version, approval)
	if err != nil {
		t.Fatalf("changed hash failed: %v", err)
	}
	if first == changed {
		t.Fatalf("hash should change with approval decision: %s", first)
	}
}

func TestEvaluateApprovalGateBlocksWithoutApproval(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}

	result := evaluateApprovalGate(record, version, ApprovalRecord{}, false, WorkflowTransitionPreview{}, false, RunGateOptions{})

	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if len(result.Failures) == 0 {
		t.Fatal("expected approval blocker")
	}
}

func TestEvaluateApprovalGatePassesWithApprovedReadyPreview(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	approval := ApprovalRecord{ID: 3, Decision: "approved"}
	preview := WorkflowTransitionPreview{ID: 4, Status: "ready"}

	result := evaluateApprovalGate(record, version, approval, true, preview, true, RunGateOptions{
		Actor:  "local-user",
		Reason: "test",
	})

	if result.Status != "pass" {
		t.Fatalf("status = %q, want pass: %+v", result.Status, result)
	}
	if result.Inputs["approval_record_id"] != int64(3) {
		t.Fatalf("approval id not recorded: %+v", result.Inputs)
	}
}

func TestEvaluateLiveMappingGateRequiresApprovalPreviewAndPromotionGate(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}

	result := evaluateLiveMappingGate(record, version, ApprovalRecord{}, false, WorkflowTransitionPreview{}, false, GateResult{}, false, RunGateOptions{})

	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if len(result.Failures) != 3 {
		t.Fatalf("failures = %+v, want three blockers", result.Failures)
	}
}

func TestEvaluateLiveMappingGatePassesReadOnlyWhenChainIsReady(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	approval := ApprovalRecord{ID: 3, Decision: "approved"}
	preview := WorkflowTransitionPreview{ID: 4, Status: "ready"}
	promotionGate := GateResult{ID: 5, GateName: "promotion_preview", Status: "pass"}

	result := evaluateLiveMappingGate(record, version, approval, true, preview, true, promotionGate, true, RunGateOptions{
		Actor:  "local-user",
		Reason: "test",
	})

	if result.Status != "pass" {
		t.Fatalf("status = %q, want pass: %+v", result.Status, result)
	}
	if result.Inputs["execution_write_attempted"] != false {
		t.Fatalf("live mapping should be read-only: %+v", result.Inputs)
	}
}

func TestEvaluateCutoverReadinessGateIsReadOnlyAndBlocked(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", WorkflowProfile: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	readiness := ProjectCutoverReadiness{
		Project: record,
		Version: version,
		Status:  "blocked",
		PhaseGate: PhaseGate{
			Name:     "v0.4-cutover-readiness",
			Status:   "blocked",
			Blockers: []string{"approval_gate is blocked"},
		},
	}

	result := evaluateCutoverReadinessGate(record, version, readiness, workflowGateSpecs["cutover_readiness_gate"], RunGateOptions{
		Actor:  "local-user",
		Reason: "test",
	})

	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if result.Inputs["cutover_apply_attempted"] != false || result.Inputs["execution_write_attempted"] != false {
		t.Fatalf("cutover readiness gate should be read-only: %+v", result.Inputs)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures = %+v, want blocker", result.Failures)
	}
}

func TestEvaluateRunnerPreflightBlocksHighRiskWithoutAllow(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"}
	preflight := EvaluateRunnerPreflight(record, version, RunnerPreviewOptions{
		RiskLevel:  "high",
		RiskPolicy: "pause",
	})

	if preflight.Status != "blocked" {
		t.Fatalf("preflight status = %q, want blocked", preflight.Status)
	}
	if len(preflight.Blockers) == 0 {
		t.Fatal("expected risk blocker")
	}
}

func TestEvaluateRunnerPreflightPassesDryRunLowRisk(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"}
	preflight := EvaluateRunnerPreflight(record, version, RunnerPreviewOptions{
		RiskLevel:  "low",
		RiskPolicy: "pause",
	})

	if preflight.Status != "pass" {
		t.Fatalf("preflight status = %q, want pass: %+v", preflight.Status, preflight)
	}
	if len(preflight.Checks) < 6 {
		t.Fatalf("preflight check count = %d, want dry-run permission checks", len(preflight.Checks))
	}
}

func skeletonWorkflowItem(id int64, versionID int64, stage string, itemType string) WorkflowItem {
	return WorkflowItem{
		ID:                id,
		ProjectID:         1,
		WorkflowVersionID: versionID,
		Stage:             stage,
		ItemType:          itemType,
		ExternalKey:       "v2:" + stage + ":" + itemType,
		Status:            "blocked",
		Metadata: map[string]any{
			"owned_by": "areaflow",
			"phase":    "v0.3b",
		},
	}
}

func readyWorkflowItem(id int64, versionID int64, stage string, itemType string, hash string) WorkflowItem {
	return WorkflowItem{
		ID:                id,
		ProjectID:         1,
		WorkflowVersionID: versionID,
		Stage:             stage,
		ItemType:          itemType,
		ExternalKey:       "v2:" + stage + ":" + itemType,
		Status:            "ready",
		SourceHash:        hash,
	}
}
