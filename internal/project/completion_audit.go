package project

import (
	"context"
	"fmt"
	"time"
)

type CompletionAuditOptions struct {
	GeneratedAt     time.Time
	APIBaseURL      string
	WebDashboardURL string
}

const completionAuditTargetProjectKey = "areamatrix"

type CompletionAuditItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	EvidenceRefs     []string
	RequiredEvidence []string
	BlockedBy        []string
	NextCommand      string
	Metadata         map[string]any
}

type CompletionAudit struct {
	Real100Guardrail
	Status                   string
	Mode                     string
	Scope                    string
	Items                    []CompletionAuditItem
	DeferredV1x              []string
	Capabilities             []string
	ForbiddenActions         []string
	SafetyFacts              map[string]bool
	ReleaseFinalGateStatus   string
	AreaMatrixDogfoodStatus  string
	TaskMatrixStatus         string
	ImplementationGapStatus  string
	ProtectedPathProofStatus string
	GeneratedAt              time.Time
}

type CompletionAuditParts struct {
	ReleaseFinalGate                   *ReleaseFinalGate
	ReleaseFinalGateError              string
	ReleaseEvidenceBundle              *ReleaseEvidenceBundle
	ReleaseEvidenceBundleError         string
	AreaMatrixDogfood                  *AreaMatrixExecutionCutoverReadiness
	AreaMatrixDogfoodError             string
	TargetProject                      *Record
	SecurityBoundaryReadiness          *SecurityBoundaryReadiness
	SecurityBoundaryReadinessError     string
	OperationsReadiness                *OperationsReadiness
	OperationsReadinessError           string
	LocalServiceStatus                 *LocalServiceStatus
	LocalServiceStatusError            string
	ArchiveProof                       *ArchiveProof
	ArchiveProofError                  string
	ShimRetirementProof                *ShimRetirementProof
	ShimRetirementProofError           string
	ExecutionCutoverProof              *ExecutionCutoverProof
	ExecutionCutoverProofError         string
	ProtectedPathProof                 *ProtectedPathProof
	ProtectedPathProofError            string
	ValidationProof                    *ValidationProof
	ValidationProofError               string
	SourceAlignmentProof               *SourceAlignmentProof
	SourceAlignmentProofError          string
	SourceAlignmentCurrentBinding      map[string]any
	SourceAlignmentCurrentBindingError string
	TaskMatrixProof                    *TaskMatrixProof
	TaskMatrixProofError               string
	TaskMatrixCurrentBinding           map[string]any
	TaskMatrixCurrentBindingError      string
	SecurityClosureProof               *SecurityClosureProof
	SecurityClosureProofError          string
	SecurityClosureCurrentBinding      map[string]any
	SecurityClosureCurrentBindingError string
	BackupRestoreProof                 *BackupRestoreProof
	BackupRestoreProofError            string
	BackupRestoreCurrentBinding        map[string]any
	BackupRestoreCurrentBindingError   string
	ReleasePackagingProof              *ReleasePackagingProof
	ReleasePackagingProofError         string
	PackageAStatusProjection           completionAuditSnapshotPackageAStatusProjectionBinding
}

func (s Store) CompletionAudit(ctx context.Context, options CompletionAuditOptions) (CompletionAudit, error) {
	options = normalizeCompletionAuditOptions(options)
	parts := CompletionAuditParts{}
	var targetRecord Record

	if record, err := s.GetByKey(ctx, completionAuditTargetProjectKey); err != nil {
		parts.AreaMatrixDogfoodError = err.Error()
	} else {
		targetRecord = record
		recordCopy := record
		parts.TargetProject = &recordCopy
		if gate, err := s.ReleaseFinalGate(ctx, ReleaseFinalGateOptions{GeneratedAt: options.GeneratedAt, ProjectID: record.ID, ProjectKey: record.Key}); err != nil {
			parts.ReleaseFinalGateError = err.Error()
		} else {
			parts.ReleaseFinalGate = &gate
		}
		if bundle, err := s.ReleaseEvidenceBundle(ctx, ReleaseEvidenceBundleOptions{GeneratedAt: options.GeneratedAt, ProjectID: record.ID, ProjectKey: record.Key}); err != nil {
			parts.ReleaseEvidenceBundleError = err.Error()
		} else {
			parts.ReleaseEvidenceBundle = &bundle
		}
		if readiness, err := s.AreaMatrixExecutionCutoverReadiness(ctx, record, AreaMatrixExecutionCutoverReadinessOptions{}); err != nil {
			parts.AreaMatrixDogfoodError = err.Error()
		} else {
			parts.AreaMatrixDogfood = &readiness
		}
	}

	if readiness, err := s.SecurityBoundaryReadiness(ctx, SecurityBoundaryReadinessOptions{GeneratedAt: options.GeneratedAt}); err != nil {
		parts.SecurityBoundaryReadinessError = err.Error()
	} else {
		parts.SecurityBoundaryReadiness = &readiness
	}

	if readiness, err := s.OperationsReadiness(ctx, OperationsReadinessOptions{
		APIBaseURL:              options.APIBaseURL,
		WebDashboardURL:         options.WebDashboardURL,
		GeneratedAt:             options.GeneratedAt,
		SmokeProofProject:       targetRecord,
		SmokeProofProjectScoped: true,
	}); err != nil {
		parts.OperationsReadinessError = err.Error()
	} else {
		parts.OperationsReadiness = &readiness
	}

	if targetRecord.ID == 0 {
		completionAuditSetTargetProjectProofErrors(&parts)
	} else if proof, err := s.LatestProtectedPathProofForProject(ctx, targetRecord); err != nil {
		parts.ProtectedPathProofError = err.Error()
	} else if proof.Status != "" {
		parts.ProtectedPathProof = &proof
	}

	if targetRecord.ID != 0 {
		parts.PackageAStatusProjection = s.completionAuditSnapshotPackageAStatusProjectionBinding(ctx, targetRecord)

		if proof, err := s.LatestArchiveProofForProject(ctx, targetRecord); err != nil {
			parts.ArchiveProofError = err.Error()
		} else if proof.Status != "" {
			parts.ArchiveProof = &proof
		}

		if proof, err := s.LatestShimRetirementProofForProject(ctx, targetRecord); err != nil {
			parts.ShimRetirementProofError = err.Error()
		} else if proof.Status != "" {
			parts.ShimRetirementProof = &proof
		}

		if proof, err := s.LatestExecutionCutoverProofForProject(ctx, targetRecord); err != nil {
			parts.ExecutionCutoverProofError = err.Error()
		} else if proof.Status != "" {
			parts.ExecutionCutoverProof = &proof
		}

		if proof, err := s.LatestValidationProofForProject(ctx, targetRecord); err != nil {
			parts.ValidationProofError = err.Error()
		} else if proof.Status != "" {
			parts.ValidationProof = &proof
		}

		if proof, err := s.LatestSourceAlignmentProofForProject(ctx, targetRecord); err != nil {
			parts.SourceAlignmentProofError = err.Error()
		} else if proof.Status != "" {
			parts.SourceAlignmentProof = &proof
			if binding, err := SourceAlignmentCurrentBinding(); err != nil {
				parts.SourceAlignmentCurrentBindingError = err.Error()
			} else {
				parts.SourceAlignmentCurrentBinding = binding
			}
		}

		if proof, err := s.LatestTaskMatrixProofForProject(ctx, targetRecord); err != nil {
			parts.TaskMatrixProofError = err.Error()
		} else if proof.Status != "" {
			parts.TaskMatrixProof = &proof
			if binding, err := TaskMatrixCurrentBinding(); err != nil {
				parts.TaskMatrixCurrentBindingError = err.Error()
			} else {
				parts.TaskMatrixCurrentBinding = binding
			}
		}

		if proof, err := s.LatestSecurityClosureProofForProject(ctx, targetRecord); err != nil {
			parts.SecurityClosureProofError = err.Error()
		} else if proof.Status != "" {
			parts.SecurityClosureProof = &proof
			binding, err := s.SecurityClosureCurrentBinding(ctx, targetRecord, SecurityClosureCurrentBindingOptions{GeneratedAt: options.GeneratedAt})
			if err != nil {
				parts.SecurityClosureCurrentBindingError = err.Error()
			} else {
				parts.SecurityClosureCurrentBinding = binding.Metadata
			}
		}

		if proof, err := s.LatestBackupRestoreProofForProject(ctx, targetRecord); err != nil {
			parts.BackupRestoreProofError = err.Error()
		} else if proof.Status != "" {
			parts.BackupRestoreProof = &proof
			binding, err := s.BackupRestoreCurrentBinding(ctx, targetRecord, BackupRestoreCurrentBindingOptions{GeneratedAt: options.GeneratedAt})
			if err != nil {
				parts.BackupRestoreCurrentBindingError = err.Error()
			} else {
				parts.BackupRestoreCurrentBinding = binding.Metadata
			}
		}

		if proof, err := s.LatestReleasePackagingProofForProject(ctx, targetRecord); err != nil {
			parts.ReleasePackagingProofError = err.Error()
		} else if proof.Status != "" {
			parts.ReleasePackagingProof = &proof
		}
	}

	return BuildCompletionAudit(options, parts), nil
}

func completionAuditSetTargetProjectProofErrors(parts *CompletionAuditParts) {
	message := parts.AreaMatrixDogfoodError
	if message == "" {
		message = fmt.Sprintf("target project %q is unavailable", completionAuditTargetProjectKey)
	}
	parts.ReleaseFinalGateError = message
	parts.ReleaseEvidenceBundleError = message
	parts.ArchiveProofError = message
	parts.ShimRetirementProofError = message
	parts.ExecutionCutoverProofError = message
	parts.ProtectedPathProofError = message
	parts.ValidationProofError = message
	parts.SourceAlignmentProofError = message
	parts.TaskMatrixProofError = message
	parts.SecurityClosureProofError = message
	parts.BackupRestoreProofError = message
	parts.ReleasePackagingProofError = message
}

func BuildCompletionAudit(options CompletionAuditOptions, parts CompletionAuditParts) CompletionAudit {
	options = normalizeCompletionAuditOptions(options)
	audit := CompletionAudit{
		Real100Guardrail:         CompletionAuditReal100Guardrail(),
		Status:                   "complete",
		Mode:                     "read_only_completion_audit",
		Scope:                    "v1.0",
		Items:                    []CompletionAuditItem{},
		DeferredV1x:              defaultCompletionAuditDeferredV1x(),
		ReleaseFinalGateStatus:   completionAuditReleaseStatus(parts),
		AreaMatrixDogfoodStatus:  completionAuditDogfoodStatus(parts),
		TaskMatrixStatus:         "incomplete",
		ImplementationGapStatus:  "incomplete",
		ProtectedPathProofStatus: completionAuditProtectedPathStatus(parts),
		Capabilities: []string{
			"read_completion_audit",
			"read_release_final_gate",
			"read_areamatrix_dogfood_status",
			"read_task_matrix_status",
			"read_implementation_gap_status",
			"read_security_boundary",
			"read_operations_readiness",
			"report_v1x_deferred_capabilities",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"create_audit_event",
			"run_smoke",
			"execute_commands",
			"start_worker",
			"create_release_package",
			"publish_release",
			"apply_restore",
			"resolve_secret_plaintext",
			"issue_remote_worker_credential",
			"touch_areamatrix_protected_paths",
		},
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"release_package_created":             false,
			"publish_attempted":                   false,
			"restore_apply_attempted":             false,
			"secret_resolved":                     false,
			"remote_worker_credentials_issued":    false,
			"area_matrix_protected_paths_touched": false,
			"database_write_attempted":            false,
			"project_write_attempted":             false,
			"smoke_run_attempted":                 false,
			"worker_started":                      false,
		},
		GeneratedAt: options.GeneratedAt,
	}

	audit.addItem(completionDesignSourceAlignmentItem(parts))
	audit.addItem(completionTaskMatrixItem(parts))
	audit.addItem(completionCommandAPISmokeItem(parts))
	audit.addItem(completionAreaMatrixDogfoodItem(parts))
	audit.addItem(completionReleasePackagingItem(parts))
	audit.addItem(completionBackupRestoreArtifactItem(parts))
	audit.addItem(completionOperationsReadinessItem(parts))
	audit.addItem(completionSecurityPermissionIsolationItem(parts))
	audit.addItem(completionProtectedPathProofItem(parts))
	audit.finalizeAggregateStatuses()
	audit.Real100Guardrail = CompletionAuditReal100GuardrailForItems(audit.Items)
	return audit
}

func normalizeCompletionAuditOptions(options CompletionAuditOptions) CompletionAuditOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func (a *CompletionAudit) addItem(item CompletionAuditItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	a.Items = append(a.Items, item)
	a.Status = combineCompletionAuditStatus(a.Status, item.Status)
}

func (a *CompletionAudit) finalizeAggregateStatuses() {
	a.TaskMatrixStatus = completionAuditItemStatus(a.Items, "E2_phase_task_matrix")
	a.ImplementationGapStatus = completionAuditCombinedItemStatus(a.Items,
		"E1_design_source_alignment",
		"E2_phase_task_matrix",
	)
}

func completionAuditItemStatus(items []CompletionAuditItem, key string) string {
	for _, item := range items {
		if item.Key == key {
			return item.Status
		}
	}
	return "incomplete"
}

func completionAuditCombinedItemStatus(items []CompletionAuditItem, keys ...string) string {
	status := "complete"
	for _, key := range keys {
		status = combineCompletionAuditStatus(status, completionAuditItemStatus(items, key))
	}
	return status
}

func combineCompletionAuditStatus(current string, next string) string {
	rank := map[string]int{
		"complete":       0,
		"not_applicable": 0,
		"deferred":       0,
		"incomplete":     1,
		"blocked":        2,
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func completionAuditReleaseStatus(parts CompletionAuditParts) string {
	if parts.ReleaseFinalGateError != "" {
		return "blocked"
	}
	if parts.ReleaseFinalGate == nil || parts.ReleaseFinalGate.Status != "pass" {
		return "incomplete"
	}
	if parts.ReleasePackagingProof != nil &&
		completionAuditProofProjectMatches(parts.ReleasePackagingProof.Project) &&
		releasePackagingProofCompletesAudit(*parts.ReleasePackagingProof) &&
		parts.ReleaseEvidenceBundle != nil &&
		len(releasePackagingProofBundleBindingBlockers(*parts.ReleasePackagingProof, *parts.ReleaseEvidenceBundle)) == 0 {
		return "complete"
	}
	return "incomplete"
}

func completionAuditDogfoodStatus(parts CompletionAuditParts) string {
	if parts.AreaMatrixDogfoodError != "" {
		return "blocked"
	}
	if len(completionAuditSnapshotPackageAStatusProjectionBlockers(parts.PackageAStatusProjection)) > 0 {
		return "incomplete"
	}
	if len(completionAuditDogfoodRealProjectIdentityBlockers(parts)) > 0 {
		return "blocked"
	}
	if parts.ArchiveProof == nil || parts.ShimRetirementProof == nil || parts.ExecutionCutoverProof == nil {
		return "incomplete"
	}
	if completionAuditProofProjectMatches(parts.ExecutionCutoverProof.Project) &&
		completionAuditProofProjectMatches(parts.ArchiveProof.Project) &&
		completionAuditProofProjectMatches(parts.ShimRetirementProof.Project) &&
		executionCutoverProofCompletesAudit(*parts.ExecutionCutoverProof) &&
		archiveProofCompletesAudit(*parts.ArchiveProof) &&
		shimRetirementProofCompletesAudit(*parts.ShimRetirementProof) &&
		len(executionCutoverProofCurrentBindingBlockers(parts.ExecutionCutoverProof.Metadata, executionCutoverProofCurrentBinding())) == 0 &&
		len(archiveProofCurrentBindingBlockers(parts.ArchiveProof.Metadata, archiveProofCurrentBinding())) == 0 &&
		len(shimRetirementProofCurrentBindingBlockers(parts.ShimRetirementProof.Metadata, shimRetirementProofCurrentBinding())) == 0 {
		return "complete"
	}
	return "incomplete"
}

func completionAuditProtectedPathStatus(parts CompletionAuditParts) string {
	if parts.ProtectedPathProof == nil {
		return "blocked"
	}
	if completionAuditProofProjectMatches(parts.ProtectedPathProof.Project) &&
		protectedPathProofCompletesAudit(*parts.ProtectedPathProof) {
		return "complete"
	}
	return "blocked"
}

func completionAuditProofProjectMatches(record Record) bool {
	return record.Key == completionAuditTargetProjectKey
}

func completionAuditDogfoodRealProjectIdentityBlockers(parts CompletionAuditParts) []string {
	if parts.TargetProject == nil {
		return []string{"areamatrix_project_identity_missing"}
	}
	return completionAuditSnapshotRealProjectIdentityBlockers(*parts.TargetProject)
}

func completionDesignSourceAlignmentItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "design sources exist, but completion audit still needs a dedicated source-alignment proof"
	blockedBy := []string{"source_alignment_proof_missing"}
	metadata := map[string]any{
		"manual_source_alignment_required": true,
	}
	if parts.SourceAlignmentProofError != "" {
		status = "blocked"
		message = "source alignment proof could not be queried"
		blockedBy = append(blockedBy, "source_alignment_proof_query_failed")
		metadata["source_alignment_proof_query_error"] = parts.SourceAlignmentProofError
	} else if parts.SourceAlignmentProof != nil {
		proof := *parts.SourceAlignmentProof
		metadata["source_alignment_proof_status"] = proof.ProofStatus
		metadata["source_alignment_proof_decision"] = proof.Decision
		metadata["source_alignment_proof_missing_facts"] = proof.MissingFacts
		metadata["latest_source_alignment_proof_project_key"] = proof.Project.Key
		metadata["latest_source_alignment_proof_event_id"] = proof.EventID
		metadata["latest_source_alignment_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["source_alignment_binding_status"] = metadataString(proof.Metadata, "source_alignment_binding_status")
		metadata["source_alignment_source_set_hash"] = metadataString(proof.Metadata, "source_alignment_source_set_hash")
		metadata["source_alignment_source_file_count"] = metadataInt64(proof.Metadata, "source_alignment_source_file_count")
		metadata["source_alignment_missing_source_count"] = metadataInt64(proof.Metadata, "source_alignment_missing_source_count")
		metadata["source_alignment_unreadable_source_count"] = metadataInt64(proof.Metadata, "source_alignment_unreadable_source_count")
		bindingBlockers := sourceAlignmentProofMetadataBindingBlockers(proof.Metadata)
		if len(bindingBlockers) > 0 {
			metadata["source_alignment_binding_blockers"] = bindingBlockers
		}
		currentBindingBlockers := []string{}
		if parts.SourceAlignmentCurrentBindingError != "" {
			currentBindingBlockers = []string{"source_alignment_current_binding_query_failed"}
			metadata["source_alignment_current_binding_query_error"] = parts.SourceAlignmentCurrentBindingError
			metadata["source_alignment_current_binding_bound"] = false
		} else if parts.SourceAlignmentCurrentBinding == nil {
			currentBindingBlockers = []string{"source_alignment_current_binding_missing"}
			metadata["source_alignment_current_binding_bound"] = false
		} else {
			addSourceAlignmentBindingMetadataWithPrefix(metadata, "current_", parts.SourceAlignmentCurrentBinding)
			currentBindingBlockers = sourceAlignmentProofCurrentBindingBlockers(proof.Metadata, parts.SourceAlignmentCurrentBinding)
			metadata["source_alignment_current_binding_bound"] = len(currentBindingBlockers) == 0
			if len(currentBindingBlockers) > 0 {
				metadata["source_alignment_current_binding_blockers"] = currentBindingBlockers
			}
		}
		if !completionAuditProofProjectMatches(proof.Project) {
			blockedBy = []string{"source_alignment_proof_project_mismatch"}
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if sourceAlignmentProofCompletesAudit(proof) && len(currentBindingBlockers) == 0 {
			status = "complete"
			message = "design source alignment proof has been recorded and current source binding still matches"
			blockedBy = []string{}
			metadata["source_alignment_gate_passed"] = true
		} else if sourceAlignmentProofCompletesAudit(proof) {
			status = "blocked"
			message = "source alignment proof has been recorded, but current source binding is stale or unavailable"
			blockedBy = uniqueStrings(append([]string{}, currentBindingBlockers...))
			metadata["source_alignment_proof_recorded"] = true
		} else if proof.ProofStatus == "blocked" {
			status = "blocked"
			message = "source alignment proof is blocked"
			blockedBy = []string{"source_alignment_proof_blocked"}
		} else if containsCompletionAuditString(currentBindingBlockers, "source_alignment_current_binding_query_failed") {
			status = "blocked"
			message = "source alignment proof is incomplete, and current binding could not be queried"
			blockedBy = uniqueStrings(append([]string{"source_alignment_proof_incomplete"}, currentBindingBlockers...))
		} else {
			blockedBy = uniqueStrings(append([]string{"source_alignment_proof_incomplete"}, bindingBlockers...))
		}
	}
	return CompletionAuditItem{
		Key:      "E1_design_source_alignment",
		Category: "design",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"docs/history/v1.0/plans/master-plan.md",
			"docs/history/v1.0/plans/platform-blueprint.md",
			"docs/history/v1.0/plans/phase-backlog.md",
			"docs/roadmap.md",
			"docs/history/v1.0/contracts/completion-audit-contract.md",
		},
		RequiredEvidence: []string{
			"source alignment proof covers 0-100% phases",
			"v1.0 and v1.x boundaries are consistent",
			"preview_only and implemented_scoped states are not described as real apply",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow completion source-alignment-proof record areamatrix --status complete --fact <required_fact> --json",
		Metadata:    metadata,
	}
}

func addSourceAlignmentBindingMetadataWithPrefix(metadata map[string]any, prefix string, binding map[string]any) {
	for key, value := range binding {
		metadata[prefix+key] = value
	}
}

func completionTaskMatrixItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "v0-v1.0 task matrix still contains preview/scoped work and must be closed by evidence before 100%"
	blockedBy := []string{"task_matrix_proof_missing"}
	metadata := map[string]any{
		"task_matrix_status": "incomplete",
	}
	if parts.TaskMatrixProofError != "" {
		status = "blocked"
		message = "task matrix proof could not be queried"
		blockedBy = append(blockedBy, "task_matrix_proof_query_failed")
		metadata["task_matrix_proof_query_error"] = parts.TaskMatrixProofError
	} else if parts.TaskMatrixProof != nil {
		proof := *parts.TaskMatrixProof
		bindingBlockers := taskMatrixProofMetadataBindingBlockers(proof.Metadata)
		currentBindingBlockers := []string{}
		metadata["task_matrix_proof_status"] = proof.ProofStatus
		metadata["task_matrix_proof_decision"] = proof.Decision
		metadata["task_matrix_proof_missing_facts"] = proof.MissingFacts
		metadata["latest_task_matrix_proof_project_key"] = proof.Project.Key
		metadata["latest_task_matrix_proof_event_id"] = proof.EventID
		metadata["latest_task_matrix_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["task_matrix_binding_status"] = metadataString(proof.Metadata, "task_matrix_binding_status")
		metadata["task_matrix_binding_blockers"] = bindingBlockers
		metadata["task_matrix_source_set_hash"] = metadataString(proof.Metadata, "task_matrix_source_set_hash")
		metadata["task_backlog_hash"] = metadataString(proof.Metadata, "task_backlog_hash")
		metadata["task_status_audit_hash"] = metadataString(proof.Metadata, "task_status_audit_hash")
		metadata["planned_v1_required_task_count"] = metadataInt64(proof.Metadata, "planned_v1_required_task_count")
		metadata["missing_evidence_v1_required_task_count"] = metadataInt64(proof.Metadata, "missing_evidence_v1_required_task_count")
		metadata["blocked_v1_required_task_count"] = metadataInt64(proof.Metadata, "blocked_v1_required_task_count")
		if parts.TaskMatrixCurrentBindingError != "" {
			currentBindingBlockers = []string{"task_matrix_current_binding_query_failed"}
			metadata["task_matrix_current_binding_query_error"] = parts.TaskMatrixCurrentBindingError
			metadata["task_matrix_current_binding_bound"] = false
		} else if parts.TaskMatrixCurrentBinding == nil {
			currentBindingBlockers = []string{"task_matrix_current_binding_missing"}
			metadata["task_matrix_current_binding_bound"] = false
		} else {
			metadata["current_task_matrix_source_set_hash"] = metadataString(parts.TaskMatrixCurrentBinding, "task_matrix_source_set_hash")
			metadata["current_task_backlog_hash"] = metadataString(parts.TaskMatrixCurrentBinding, "task_backlog_hash")
			metadata["current_task_status_audit_hash"] = metadataString(parts.TaskMatrixCurrentBinding, "task_status_audit_hash")
			metadata["current_planned_v1_required_task_count"] = metadataInt64(parts.TaskMatrixCurrentBinding, "planned_v1_required_task_count")
			metadata["current_missing_evidence_v1_required_task_count"] = metadataInt64(parts.TaskMatrixCurrentBinding, "missing_evidence_v1_required_task_count")
			metadata["current_blocked_v1_required_task_count"] = metadataInt64(parts.TaskMatrixCurrentBinding, "blocked_v1_required_task_count")
			currentBindingBlockers = taskMatrixProofCurrentBindingBlockers(proof.Metadata, parts.TaskMatrixCurrentBinding)
			metadata["task_matrix_current_binding_bound"] = len(currentBindingBlockers) == 0
			metadata["task_matrix_current_binding_blockers"] = currentBindingBlockers
		}
		if !completionAuditProofProjectMatches(proof.Project) {
			blockedBy = []string{"task_matrix_proof_project_mismatch"}
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if proof.ProofStatus == "complete" && len(bindingBlockers) > 0 {
			blockedBy = []string{"task_matrix_binding_incomplete"}
		} else if proof.ProofStatus == "complete" && len(currentBindingBlockers) > 0 {
			status = "blocked"
			message = "task matrix proof is stale or current task matrix binding is unavailable"
			blockedBy = uniqueStrings(currentBindingBlockers)
		} else if taskMatrixProofCompletesAudit(proof) {
			status = "complete"
			message = "phase and task matrix proof has been recorded"
			blockedBy = []string{}
			metadata["task_matrix_gate_passed"] = true
			metadata["task_matrix_status"] = "complete"
		} else if proof.ProofStatus == "blocked" {
			status = "blocked"
			message = "task matrix proof is blocked"
			blockedBy = []string{"task_matrix_proof_blocked"}
		} else {
			blockedBy = []string{"task_matrix_proof_incomplete"}
		}
	}
	return CompletionAuditItem{
		Key:      "E2_phase_task_matrix",
		Category: "task_matrix",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"docs/history/v1.0/plans/task-backlog.md",
			"docs/history/v1.0/evidence/task-backlog-status-audit.md",
		},
		RequiredEvidence: []string{
			"no v0-v1.0 task remains planned",
			"preview_only and implemented_scoped items are either closed by evidence or explicitly deferred",
			"closest open task lists next command and required evidence",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow completion task-matrix-proof record areamatrix --status complete --fact <required_fact> --source-set-hash <sha256> --backlog-hash <sha256> --task-status-audit-hash <sha256> --planned-v1-required-task-count 0 --missing-evidence-v1-required-task-count 0 --blocked-v1-required-task-count 0 --json",
		Metadata:    metadata,
	}
}

func completionCommandAPISmokeItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "completion audit does not run validation; fresh command, build and smoke evidence must be provided"
	blockedBy := []string{"fresh_validation_proof_missing"}
	metadata := map[string]any{
		"completion_audit_runs_smoke": false,
	}
	if parts.ValidationProofError != "" {
		status = "blocked"
		message = "validation proof could not be queried"
		blockedBy = append(blockedBy, "validation_proof_query_failed")
		metadata["validation_proof_query_error"] = parts.ValidationProofError
	} else if parts.ValidationProof != nil {
		proof := *parts.ValidationProof
		metadata["validation_proof_status"] = proof.ProofStatus
		metadata["validation_proof_decision"] = proof.Decision
		metadata["validation_proof_missing_facts"] = proof.MissingFacts
		metadata["latest_validation_proof_project_key"] = proof.Project.Key
		metadata["latest_validation_proof_event_id"] = proof.EventID
		metadata["latest_validation_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["validation_evidence_binding_status"] = metadataString(proof.Metadata, "validation_evidence_binding_status")
		metadata["validation_scope"] = metadataString(proof.Metadata, "validation_scope")
		metadata["validation_result_hash"] = metadataString(proof.Metadata, "validation_result_hash")
		metadata["validation_command_count"] = metadataInt64(proof.Metadata, "validation_command_count")
		metadata["validation_started_at"] = metadataString(proof.Metadata, "validation_started_at")
		metadata["validation_finished_at"] = metadataString(proof.Metadata, "validation_finished_at")
		if blockers := validationProofMetadataBindingBlockers(proof.Metadata); len(blockers) > 0 {
			metadata["validation_evidence_binding_blockers"] = blockers
		}
		if !completionAuditProofProjectMatches(proof.Project) {
			blockedBy = []string{"validation_proof_project_mismatch"}
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if validationProofCompletesAudit(proof) {
			status = "complete"
			message = "fresh command, build and smoke validation proof has been recorded"
			blockedBy = []string{}
			metadata["validation_gate_passed"] = true
		} else if proof.ProofStatus == "blocked" {
			status = "blocked"
			message = "validation proof is blocked"
			blockedBy = []string{"validation_proof_blocked"}
		} else {
			blockedBy = []string{"validation_proof_incomplete"}
		}
	}
	return CompletionAuditItem{
		Key:      "E3_command_api_smoke_evidence",
		Category: "validation",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"docs/history/v1.0/evidence/v1-stable-fixture-evidence.md",
			"docs/history/v1.0/evidence/multi-project-isolation-evidence.md",
			"docs/history/v1.0/evidence/completion-audit-evidence.md",
		},
		RequiredEvidence: []string{
			"go test ./...",
			"go build ./cmd/areaflow",
			"cd web && npm run build",
			"git diff --check -- .",
			"AREAFLOW_DATABASE_URL=... ./scripts/smoke-v1-stable-fixture.sh",
			"AREAFLOW_DATABASE_URL=... ./scripts/smoke-web.sh",
			"AREAFLOW_DATABASE_URL=... ./scripts/smoke-project-isolation.sh",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow completion validation-proof record areamatrix --status complete --fact <required_fact> --json",
		Metadata:    metadata,
	}
}

func completionAreaMatrixDogfoodItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "AreaMatrix dogfood has not proven execution cutover, archive and shim retirement"
	blockedBy := []string{"execution_cutover_not_complete", "real_areamatrix_archive_not_proven", "real_areamatrix_shim_retirement_not_proven"}
	metadata := map[string]any{
		"required_path": []string{"Import", "Mirror", "Shadow", "Authoring Cutover", "Execution Beta", "Execution Cutover", "Archive", "Shim Retirement"},
	}
	packageABlockers := completionAuditSnapshotPackageAStatusProjectionBlockers(parts.PackageAStatusProjection)
	addCompletionAuditSnapshotPackageAStatusProjectionMetadata(metadata, parts.PackageAStatusProjection)
	if len(packageABlockers) > 0 {
		blockedBy = append(blockedBy, packageABlockers...)
	}
	projectIdentityBlockers := completionAuditDogfoodRealProjectIdentityBlockers(parts)
	if len(projectIdentityBlockers) > 0 {
		blockedBy = append(blockedBy, projectIdentityBlockers...)
		metadata["real_project_identity_status"] = "blocked"
		metadata["real_project_identity_blockers"] = projectIdentityBlockers
		metadata["expected_project_key"] = completionAuditTargetProjectKey
		metadata["expected_project_root"] = completionAuditTargetProjectRoot
		if parts.TargetProject != nil {
			metadata["actual_project_key"] = parts.TargetProject.Key
			metadata["actual_project_root"] = parts.TargetProject.RootPath
			metadata["actual_project_adapter"] = parts.TargetProject.Adapter
			metadata["actual_workflow_profile"] = parts.TargetProject.WorkflowProfile
			metadata["actual_default_branch"] = parts.TargetProject.DefaultBranch
			metadata["actual_project_kind"] = parts.TargetProject.Kind
		}
	} else {
		metadata["real_project_identity_status"] = "pass"
		metadata["real_project_identity_blockers"] = []string{}
	}
	currentArchiveBinding := archiveProofCurrentBinding()
	metadata["current_archive_binding_contract"] = metadataString(currentArchiveBinding, "archive_binding_contract")
	metadata["current_archive_source_paths_hash"] = metadataString(currentArchiveBinding, "archive_source_paths_hash")
	metadata["current_archive_forbidden_actions_hash"] = metadataString(currentArchiveBinding, "archive_forbidden_actions_hash")
	metadata["current_archive_binding_hash"] = metadataString(currentArchiveBinding, "archive_binding_hash")
	metadata["current_archive_scope_binding_hash"] = metadataString(currentArchiveBinding, "archive_scope_binding_hash")
	currentShimRetirementBinding := shimRetirementProofCurrentBinding()
	metadata["current_shim_retirement_binding_contract"] = metadataString(currentShimRetirementBinding, "shim_retirement_binding_contract")
	metadata["current_shim_retirement_prerequisites_hash"] = metadataString(currentShimRetirementBinding, "shim_retirement_prerequisites_hash")
	metadata["current_shim_retired_surfaces_hash"] = metadataString(currentShimRetirementBinding, "shim_retired_surfaces_hash")
	metadata["current_shim_retirement_binding_hash"] = metadataString(currentShimRetirementBinding, "shim_retirement_binding_hash")
	metadata["current_shim_retirement_scope_binding_hash"] = metadataString(currentShimRetirementBinding, "shim_retirement_scope_binding_hash")
	currentExecutionCutoverBinding := executionCutoverProofCurrentBinding()
	metadata["current_execution_cutover_binding_contract"] = metadataString(currentExecutionCutoverBinding, "execution_cutover_binding_contract")
	metadata["current_allowed_task_types_hash"] = metadataString(currentExecutionCutoverBinding, "allowed_task_types_hash")
	metadata["current_forbidden_actions_hash"] = metadataString(currentExecutionCutoverBinding, "forbidden_actions_hash")
	metadata["current_execution_cutover_binding_hash"] = metadataString(currentExecutionCutoverBinding, "execution_cutover_binding_hash")
	metadata["current_execution_cutover_scope_binding_hash"] = metadataString(currentExecutionCutoverBinding, "execution_cutover_scope_binding_hash")
	if parts.AreaMatrixDogfoodError != "" {
		metadata["query_error"] = parts.AreaMatrixDogfoodError
		blockedBy = append(blockedBy, "areamatrix_dogfood_query_failed")
	}
	archiveComplete := false
	if parts.ArchiveProofError != "" {
		metadata["archive_proof_query_error"] = parts.ArchiveProofError
		blockedBy = append(blockedBy, "archive_proof_query_failed")
	}
	if parts.ArchiveProof != nil {
		proof := *parts.ArchiveProof
		archiveBindingBlockers := archiveProofMetadataBindingBlockers(proof.Metadata)
		archiveCurrentBindingBlockers := archiveProofCurrentBindingBlockers(proof.Metadata, currentArchiveBinding)
		archiveReviewEvidenceBlockers := proofCompleteReviewEvidenceBlockers("archive_proof", proof.Metadata)
		archiveComplete = archiveProofCompletesAudit(proof) && len(archiveCurrentBindingBlockers) == 0
		metadata["archive_proof_status"] = proof.ProofStatus
		metadata["archive_proof_decision"] = proof.Decision
		metadata["archive_proof_missing_facts"] = proof.MissingFacts
		metadata["archive_proof_review_evidence_blockers"] = archiveReviewEvidenceBlockers
		metadata["archive_proof_review_metadata_status"] = metadataString(proof.Metadata, "review_metadata_status")
		metadata["archive_proof_review_metadata_blockers"] = metadataStringSlice(proof.Metadata, "review_metadata_blockers")
		metadata["latest_archive_proof_project_key"] = proof.Project.Key
		metadata["latest_archive_proof_event_id"] = proof.EventID
		metadata["latest_archive_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["archive_scope_binding_status"] = metadataString(proof.Metadata, "archive_scope_binding_status")
		metadata["archive_scope_binding_blockers"] = archiveBindingBlockers
		metadata["archive_binding_contract"] = metadataString(proof.Metadata, "archive_binding_contract")
		metadata["archive_source_paths_hash"] = metadataString(proof.Metadata, "archive_source_paths_hash")
		metadata["archive_forbidden_actions_hash"] = metadataString(proof.Metadata, "archive_forbidden_actions_hash")
		metadata["archive_binding_hash"] = metadataString(proof.Metadata, "archive_binding_hash")
		metadata["archive_scope_binding_hash"] = metadataString(proof.Metadata, "archive_scope_binding_hash")
		metadata["archive_current_binding_bound"] = len(archiveCurrentBindingBlockers) == 0
		metadata["archive_scope_current_binding_bound"] = len(archiveCurrentBindingBlockers) == 0
		metadata["archive_current_binding_blockers"] = archiveCurrentBindingBlockers
		metadata["archive_scope_current_binding_blockers"] = archiveCurrentBindingBlockers
		metadata["archive_scope"] = metadataString(proof.Metadata, "archive_scope")
		metadata["archive_reference_mode"] = metadataString(proof.Metadata, "archive_reference_mode")
		metadata["archive_source_paths"] = metadataStringSlice(proof.Metadata, "archive_source_paths")
		metadata["archive_forbidden_actions"] = metadataStringSlice(proof.Metadata, "archive_forbidden_actions")
		metadata["archive_rollback_target"] = metadataString(proof.Metadata, "archive_rollback_target")
		metadata["archive_fail_closed"] = metadataBool(proof.Metadata, "archive_fail_closed")
		if !completionAuditProofProjectMatches(proof.Project) {
			archiveComplete = false
			blockedBy = append(blockedBy, "archive_proof_project_mismatch")
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if archiveComplete && len(projectIdentityBlockers) == 0 {
			blockedBy = removeString(blockedBy, "real_areamatrix_archive_not_proven")
		} else if len(archiveBindingBlockers) > 0 {
			blockedBy = append(blockedBy, "archive_scope_binding_incomplete")
		} else if proof.ProofStatus == "complete" && proof.EventID <= 0 {
			blockedBy = append(blockedBy, "archive_proof_event_id_missing")
		} else if proof.ProofStatus == "complete" && len(archiveCurrentBindingBlockers) > 0 {
			blockedBy = append(blockedBy, archiveCurrentBindingBlockers...)
		} else if len(archiveReviewEvidenceBlockers) > 0 {
			blockedBy = append(blockedBy, archiveReviewEvidenceBlockers...)
		} else if proof.ProofStatus == "blocked" || proof.Status == "blocked" {
			blockedBy = append(blockedBy, "archive_proof_blocked")
		} else {
			blockedBy = append(blockedBy, "archive_proof_incomplete")
		}
	}
	shimRetirementComplete := false
	if parts.ShimRetirementProofError != "" {
		metadata["shim_retirement_proof_query_error"] = parts.ShimRetirementProofError
		blockedBy = append(blockedBy, "shim_retirement_proof_query_failed")
	}
	if parts.ShimRetirementProof != nil {
		proof := *parts.ShimRetirementProof
		shimBindingBlockers := shimRetirementProofMetadataBindingBlockers(proof.Metadata)
		shimCurrentBindingBlockers := shimRetirementProofCurrentBindingBlockers(proof.Metadata, currentShimRetirementBinding)
		shimReviewEvidenceBlockers := proofCompleteReviewEvidenceBlockers("shim_retirement_proof", proof.Metadata)
		shimRetirementComplete = shimRetirementProofCompletesAudit(proof) && len(shimCurrentBindingBlockers) == 0
		metadata["shim_retirement_proof_status"] = proof.ProofStatus
		metadata["shim_retirement_proof_decision"] = proof.Decision
		metadata["shim_retirement_proof_missing_facts"] = proof.MissingFacts
		metadata["shim_retirement_proof_review_evidence_blockers"] = shimReviewEvidenceBlockers
		metadata["shim_retirement_proof_review_metadata_status"] = metadataString(proof.Metadata, "review_metadata_status")
		metadata["shim_retirement_proof_review_metadata_blockers"] = metadataStringSlice(proof.Metadata, "review_metadata_blockers")
		metadata["latest_shim_retirement_proof_project_key"] = proof.Project.Key
		metadata["latest_shim_retirement_proof_event_id"] = proof.EventID
		metadata["latest_shim_retirement_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["shim_retirement_scope_binding_status"] = metadataString(proof.Metadata, "shim_retirement_scope_binding_status")
		metadata["shim_retirement_scope_binding_blockers"] = shimBindingBlockers
		metadata["shim_retirement_binding_contract"] = metadataString(proof.Metadata, "shim_retirement_binding_contract")
		metadata["shim_retirement_prerequisites_hash"] = metadataString(proof.Metadata, "shim_retirement_prerequisites_hash")
		metadata["shim_retired_surfaces_hash"] = metadataString(proof.Metadata, "shim_retired_surfaces_hash")
		metadata["shim_retirement_binding_hash"] = metadataString(proof.Metadata, "shim_retirement_binding_hash")
		metadata["shim_retirement_scope_binding_hash"] = metadataString(proof.Metadata, "shim_retirement_scope_binding_hash")
		metadata["shim_retirement_current_binding_bound"] = len(shimCurrentBindingBlockers) == 0
		metadata["shim_retirement_scope_current_binding_bound"] = len(shimCurrentBindingBlockers) == 0
		metadata["shim_retirement_current_binding_blockers"] = shimCurrentBindingBlockers
		metadata["shim_retirement_scope_current_binding_blockers"] = shimCurrentBindingBlockers
		metadata["shim_retirement_scope"] = metadataString(proof.Metadata, "shim_retirement_scope")
		metadata["shim_retirement_prerequisites"] = metadataStringSlice(proof.Metadata, "shim_retirement_prerequisites")
		metadata["shim_retired_surfaces"] = metadataStringSlice(proof.Metadata, "shim_retired_surfaces")
		metadata["shim_rollback_target"] = metadataString(proof.Metadata, "shim_rollback_target")
		metadata["shim_fail_closed"] = metadataBool(proof.Metadata, "shim_fail_closed")
		metadata["shim_reopen_requires_approval"] = metadataBool(proof.Metadata, "shim_reopen_requires_approval")
		if !completionAuditProofProjectMatches(proof.Project) {
			shimRetirementComplete = false
			blockedBy = append(blockedBy, "shim_retirement_proof_project_mismatch")
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if shimRetirementComplete && len(projectIdentityBlockers) == 0 {
			blockedBy = removeString(blockedBy, "real_areamatrix_shim_retirement_not_proven")
		} else if len(shimBindingBlockers) > 0 {
			blockedBy = append(blockedBy, "shim_retirement_scope_binding_incomplete")
		} else if proof.ProofStatus == "complete" && proof.EventID <= 0 {
			blockedBy = append(blockedBy, "shim_retirement_proof_event_id_missing")
		} else if proof.ProofStatus == "complete" && len(shimCurrentBindingBlockers) > 0 {
			blockedBy = append(blockedBy, shimCurrentBindingBlockers...)
		} else if len(shimReviewEvidenceBlockers) > 0 {
			blockedBy = append(blockedBy, shimReviewEvidenceBlockers...)
		} else if proof.ProofStatus == "blocked" || proof.Status == "blocked" {
			blockedBy = append(blockedBy, "shim_retirement_proof_blocked")
		} else {
			blockedBy = append(blockedBy, "shim_retirement_proof_incomplete")
		}
	}
	executionCutoverComplete := false
	if parts.ExecutionCutoverProofError != "" {
		metadata["execution_cutover_proof_query_error"] = parts.ExecutionCutoverProofError
		blockedBy = append(blockedBy, "execution_cutover_proof_query_failed")
	}
	if parts.ExecutionCutoverProof != nil {
		proof := *parts.ExecutionCutoverProof
		executionCutoverBindingBlockers := executionCutoverProofMetadataBindingBlockers(proof.Metadata)
		executionCutoverCurrentBindingBlockers := executionCutoverProofCurrentBindingBlockers(proof.Metadata, currentExecutionCutoverBinding)
		executionCutoverReviewEvidenceBlockers := proofCompleteReviewEvidenceBlockers("execution_cutover_proof", proof.Metadata)
		executionCutoverComplete = executionCutoverProofCompletesAudit(proof) && len(executionCutoverCurrentBindingBlockers) == 0
		metadata["execution_cutover_proof_status"] = proof.ProofStatus
		metadata["execution_cutover_proof_decision"] = proof.Decision
		metadata["execution_cutover_proof_missing_facts"] = proof.MissingFacts
		metadata["execution_cutover_proof_review_evidence_blockers"] = executionCutoverReviewEvidenceBlockers
		metadata["execution_cutover_proof_review_metadata_status"] = metadataString(proof.Metadata, "review_metadata_status")
		metadata["execution_cutover_proof_review_metadata_blockers"] = metadataStringSlice(proof.Metadata, "review_metadata_blockers")
		metadata["latest_execution_cutover_proof_project_key"] = proof.Project.Key
		metadata["latest_execution_cutover_proof_event_id"] = proof.EventID
		metadata["latest_execution_cutover_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["execution_cutover_scope_binding_status"] = metadataString(proof.Metadata, "execution_cutover_scope_binding_status")
		metadata["execution_cutover_scope_binding_blockers"] = executionCutoverBindingBlockers
		metadata["execution_cutover_binding_contract"] = metadataString(proof.Metadata, "execution_cutover_binding_contract")
		metadata["allowed_task_types_hash"] = metadataString(proof.Metadata, "allowed_task_types_hash")
		metadata["forbidden_actions_hash"] = metadataString(proof.Metadata, "forbidden_actions_hash")
		metadata["execution_cutover_binding_hash"] = metadataString(proof.Metadata, "execution_cutover_binding_hash")
		metadata["execution_cutover_scope_binding_hash"] = metadataString(proof.Metadata, "execution_cutover_scope_binding_hash")
		metadata["execution_cutover_current_binding_bound"] = len(executionCutoverCurrentBindingBlockers) == 0
		metadata["execution_cutover_scope_current_binding_bound"] = len(executionCutoverCurrentBindingBlockers) == 0
		metadata["execution_cutover_current_binding_blockers"] = executionCutoverCurrentBindingBlockers
		metadata["execution_cutover_scope_current_binding_blockers"] = executionCutoverCurrentBindingBlockers
		metadata["execution_cutover_scope"] = metadataString(proof.Metadata, "execution_cutover_scope")
		metadata["execution_cutover_allowed_task_types"] = metadataStringSlice(proof.Metadata, "allowed_task_types")
		metadata["execution_cutover_forbidden_actions"] = metadataStringSlice(proof.Metadata, "forbidden_actions")
		metadata["execution_cutover_rollback_target"] = metadataString(proof.Metadata, "rollback_target")
		metadata["execution_cutover_rollback_mode"] = metadataString(proof.Metadata, "rollback_mode")
		metadata["execution_cutover_fail_closed"] = metadataBool(proof.Metadata, "fail_closed")
		metadata["execution_cutover_reopen_requires_approval"] = metadataBool(proof.Metadata, "reopen_requires_approval")
		metadata["source_write_open"] = metadataBool(proof.Metadata, "source_write_open")
		metadata["generated_retained_write_open"] = metadataBool(proof.Metadata, "generated_retained_write_open")
		metadata["repair_apply_open"] = metadataBool(proof.Metadata, "repair_apply_open")
		metadata["checkpoint_apply_open"] = metadataBool(proof.Metadata, "checkpoint_apply_open")
		metadata["engine_execution_open"] = metadataBool(proof.Metadata, "engine_execution_open")
		metadata["secret_resolve_open"] = metadataBool(proof.Metadata, "secret_resolve_open")
		metadata["network_api_integration_open"] = metadataBool(proof.Metadata, "network_api_integration_open")
		metadata["publish_apply_open"] = metadataBool(proof.Metadata, "publish_apply_open")
		metadata["restore_apply_open"] = metadataBool(proof.Metadata, "restore_apply_open")
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["task_loop_run_forwarded_by_command"] = proof.TaskLoopRunForwardedByCommand
		metadata["engine_call_attempted"] = proof.EngineCallAttempted
		metadata["commands_run"] = proof.CommandsRun
		metadata["legacy_progress_written"] = proof.LegacyProgressWritten
		metadata["legacy_logs_written"] = proof.LegacyLogsWritten
		metadata["legacy_checkpoint_written"] = proof.LegacyCheckpointWritten
		metadata["area_matrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		if !completionAuditProofProjectMatches(proof.Project) {
			executionCutoverComplete = false
			blockedBy = append(blockedBy, "execution_cutover_proof_project_mismatch")
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if executionCutoverComplete && len(projectIdentityBlockers) == 0 {
			blockedBy = removeString(blockedBy, "execution_cutover_not_complete")
		} else if len(executionCutoverBindingBlockers) > 0 {
			blockedBy = append(blockedBy, "execution_cutover_scope_binding_incomplete")
		} else if proof.ProofStatus == "complete" && proof.EventID <= 0 {
			blockedBy = append(blockedBy, "execution_cutover_proof_event_id_missing")
		} else if proof.ProofStatus == "complete" && len(executionCutoverCurrentBindingBlockers) > 0 {
			blockedBy = append(blockedBy, executionCutoverCurrentBindingBlockers...)
		} else if len(executionCutoverReviewEvidenceBlockers) > 0 {
			blockedBy = append(blockedBy, executionCutoverReviewEvidenceBlockers...)
		} else if proof.ProofStatus == "blocked" || proof.Status == "blocked" {
			blockedBy = append(blockedBy, "execution_cutover_proof_blocked")
		} else {
			blockedBy = append(blockedBy, "execution_cutover_proof_incomplete")
		}
	}
	if parts.AreaMatrixDogfood != nil {
		metadata["execution_cutover_status"] = parts.AreaMatrixDogfood.Status
		metadata["execution_cutover_apply_open"] = parts.AreaMatrixDogfood.SafetyFacts["execution_cutover_apply_open"]
		metadata["task_loop_run_forwarded"] = parts.AreaMatrixDogfood.SafetyFacts["task_loop_run_forwarded"]
	}
	if executionCutoverComplete && len(projectIdentityBlockers) == 0 {
		metadata["execution_cutover_gate_passed"] = true
	}
	if archiveComplete && len(projectIdentityBlockers) == 0 {
		metadata["archive_gate_passed"] = true
	}
	if shimRetirementComplete && len(projectIdentityBlockers) == 0 {
		metadata["shim_retirement_gate_passed"] = true
	}
	if len(blockedBy) == 0 {
		status = "complete"
		message = "AreaMatrix dogfood execution cutover, archive and shim retirement are proven"
	} else if len(projectIdentityBlockers) > 0 {
		status = "blocked"
		message = "AreaMatrix dogfood proof is blocked by real AreaMatrix project identity"
		blockedBy = uniqueStrings(blockedBy)
	} else {
		blockedBy = uniqueStrings(blockedBy)
	}
	return CompletionAuditItem{
		Key:      "E4_areamatrix_dogfood_completion",
		Category: "dogfood",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"docs/history/v1.0/migrations/areamatrix-workflow-migration.md",
			"docs/history/v1.0/migrations/areamatrix-execution-cutover-boundary.md",
			"GET /api/v1/projects/areamatrix/execution-cutover-readiness",
		},
		RequiredEvidence: []string{
			"AreaMatrix Import -> Mirror -> Shadow -> Authoring Cutover -> Execution Beta -> Execution Cutover -> Archive -> Shim Retirement complete",
			"execution cutover approval, command response, event and audit proof",
			"archive proof records immutable historical index, metadata-only reference limits, no historical deletion and rollback path",
			"shim retirement proof records stable forwarding window, old runner retirement notice, command mapping and rollback path",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow completion shim-retirement-proof record areamatrix --status complete --fact <required_fact> --json",
		Metadata:    metadata,
	}
}

func completionReleasePackagingItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "release final gate has not proven pass, and release/package/publish/rollout remain preview-only"
	blockedBy := []string{"release_final_gate_not_passed", "release_packaging_proof_missing"}
	metadata := map[string]any{}
	releaseFinalGatePassed := false
	if parts.ReleaseFinalGateError != "" {
		status = "blocked"
		message = "release final gate could not be queried"
		blockedBy = append(blockedBy, "release_final_gate_query_failed")
		metadata["query_error"] = parts.ReleaseFinalGateError
	}
	if parts.ReleaseFinalGate != nil {
		metadata["release_final_gate_status"] = parts.ReleaseFinalGate.Status
		metadata["release_final_gate_mode"] = parts.ReleaseFinalGate.Mode
		if parts.ReleaseFinalGate.Status == "pass" {
			releaseFinalGatePassed = true
			message = "release final gate passes, but release packaging proof is still required"
			blockedBy = removeString(blockedBy, "release_final_gate_not_passed")
		}
	}
	if parts.ReleaseEvidenceBundleError != "" {
		status = "blocked"
		message = "release evidence bundle could not be queried"
		blockedBy = append(blockedBy, "release_evidence_bundle_query_failed")
		metadata["release_evidence_bundle_query_error"] = parts.ReleaseEvidenceBundleError
	} else if parts.ReleaseEvidenceBundle != nil {
		addReleaseEvidenceBundleBindingMetadata(metadata, "current_", *parts.ReleaseEvidenceBundle)
	}
	if parts.ReleasePackagingProofError != "" {
		status = "blocked"
		message = "release packaging proof could not be queried"
		blockedBy = append(blockedBy, "release_packaging_proof_query_failed")
		metadata["release_packaging_proof_query_error"] = parts.ReleasePackagingProofError
	} else if parts.ReleasePackagingProof != nil {
		proof := *parts.ReleasePackagingProof
		metadata["release_packaging_proof_status"] = proof.ProofStatus
		metadata["release_packaging_proof_decision"] = proof.Decision
		metadata["release_packaging_proof_missing_facts"] = proof.MissingFacts
		metadata["latest_release_packaging_proof_project_key"] = proof.Project.Key
		metadata["latest_release_packaging_proof_event_id"] = proof.EventID
		metadata["latest_release_packaging_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		addReleasePackagingProofBundleBindingMetadata(metadata, proof)
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["release_package_created"] = proof.ReleasePackageCreated
		metadata["release_state_written"] = proof.ReleaseStateWritten
		metadata["release_approval_created"] = proof.ReleaseApprovalCreated
		metadata["rollout_state_created"] = proof.RolloutStateCreated
		metadata["migration_apply_attempted"] = proof.MigrationApplyAttempted
		metadata["tag_created"] = proof.TagCreated
		metadata["package_signed"] = proof.PackageSigned
		metadata["artifact_uploaded"] = proof.ArtifactUploaded
		metadata["git_push_attempted"] = proof.GitPushAttempted
		metadata["publish_attempted"] = proof.PublishAttempted
		metadata["commands_run"] = proof.CommandsRun
		metadata["area_matrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		if !completionAuditProofProjectMatches(proof.Project) {
			blockedBy = []string{"release_packaging_proof_project_mismatch"}
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if releasePackagingProofCompletesAudit(proof) && releaseFinalGatePassed && parts.ReleaseEvidenceBundle != nil && len(releasePackagingProofBundleBindingBlockers(proof, *parts.ReleaseEvidenceBundle)) == 0 {
			status = "complete"
			message = "release final gate and packaging preview proof are complete"
			blockedBy = []string{}
			metadata["release_packaging_gate_passed"] = true
			metadata["release_packaging_proof_bundle_bound"] = true
		} else if releasePackagingProofCompletesAudit(proof) {
			blockedBy = removeString(blockedBy, "release_packaging_proof_missing")
			metadata["release_packaging_proof_recorded"] = true
			if parts.ReleaseEvidenceBundle == nil {
				message = "release packaging proof has been recorded, but the current release evidence bundle is unavailable"
				blockedBy = append(blockedBy, "release_evidence_bundle_missing")
			} else if bundleBlockers := releasePackagingProofBundleBindingBlockers(proof, *parts.ReleaseEvidenceBundle); len(bundleBlockers) > 0 {
				message = "release packaging proof has been recorded, but the current release evidence bundle binding is stale or incomplete"
				blockedBy = append(blockedBy, bundleBlockers...)
				metadata["release_packaging_proof_bundle_bound"] = false
			} else {
				message = "release packaging proof has been recorded, but the current release final gate has not passed"
				metadata["release_packaging_proof_bundle_bound"] = true
			}
		} else if proof.ProofStatus == "blocked" || proof.Status == "blocked" {
			status = "blocked"
			message = "release packaging proof is blocked"
			blockedBy = append(removeString(blockedBy, "release_packaging_proof_missing"), "release_packaging_proof_blocked")
		} else {
			blockedBy = append(removeString(blockedBy, "release_packaging_proof_missing"), "release_packaging_proof_incomplete")
			blockedBy = append(blockedBy, releasePackagingProofRequiredBundleMetadataBlockers(proof.Metadata)...)
		}
	}
	return CompletionAuditItem{
		Key:      "E5_release_packaging_preview",
		Category: "release",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"GET /api/v1/release/final-gate",
			"GET /api/v1/release/evidence-bundle",
			"GET /api/v1/release/package-preview",
			"GET /api/v1/release/distribution-preview",
			"GET /api/v1/release/publish-gate",
			"GET /api/v1/release/publish-approval-preview",
			"GET /api/v1/release/rollout-plan-preview",
		},
		RequiredEvidence: []string{
			"release final gate pass",
			"release package preview remains preview-only",
			"publish and rollout surfaces do not create release state",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow completion release-packaging-proof record areamatrix --status complete --fact <required_fact> --json",
		Metadata:    metadata,
	}
}

func addReleaseEvidenceBundleBindingMetadata(metadata map[string]any, prefix string, bundle ReleaseEvidenceBundle) {
	binding := ReleaseEvidenceBundleBindingMetadata(bundle)
	for key, value := range binding {
		metadata[prefix+key] = value
	}
}

func addReleasePackagingProofBundleBindingMetadata(metadata map[string]any, proof ReleasePackagingProof) {
	metadata["proof_release_evidence_bundle_hash"] = metadataString(proof.Metadata, "release_evidence_bundle_hash")
	metadata["proof_release_evidence_bundle_status"] = metadataString(proof.Metadata, "release_evidence_bundle_status")
	metadata["proof_release_evidence_bundle_mode"] = metadataString(proof.Metadata, "release_evidence_bundle_mode")
	metadata["proof_release_evidence_bundle_scope"] = metadataString(proof.Metadata, "release_evidence_bundle_scope")
	metadata["proof_release_evidence_bundle_project_key"] = metadataString(proof.Metadata, "release_evidence_bundle_project_key")
	metadata["proof_release_evidence_bundle_item_count"] = metadataInt64(proof.Metadata, "release_evidence_bundle_item_count")
	metadata["proof_release_evidence_bundle_project_inventory_key"] = metadataString(proof.Metadata, "release_evidence_bundle_project_inventory_key")
	metadata["proof_release_evidence_bundle_project_inventory_present"] = metadataBool(proof.Metadata, "release_evidence_bundle_project_inventory_present")
	metadata["proof_release_evidence_bundle_project_inventory_ready"] = metadataBool(proof.Metadata, "release_evidence_bundle_project_inventory_ready")
	metadata["proof_release_evidence_bundle_ready"] = metadataBool(proof.Metadata, "release_evidence_bundle_ready")
}

func completionBackupRestoreArtifactItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "backup, restore and artifact readiness must be closed by dedicated proof"
	blockedBy := []string{"backup_restore_proof_missing"}
	metadata := map[string]any{}
	if parts.ReleaseFinalGate != nil {
		metadata["release_readiness_status"] = parts.ReleaseFinalGate.Readiness.Status
		metadata["backup_status"] = parts.ReleaseFinalGate.Readiness.Backup.Status
		metadata["restore_plan_status"] = parts.ReleaseFinalGate.Readiness.RestorePlan.Status
		metadata["release_readiness_is_sufficient_for_e6"] = false
	}
	if parts.BackupRestoreProofError != "" {
		status = "blocked"
		message = "backup restore proof could not be queried"
		blockedBy = append(blockedBy, "backup_restore_proof_query_failed")
		metadata["backup_restore_proof_query_error"] = parts.BackupRestoreProofError
	} else if parts.BackupRestoreProof != nil {
		proof := *parts.BackupRestoreProof
		metadata["backup_restore_proof_status"] = proof.ProofStatus
		metadata["backup_restore_proof_decision"] = proof.Decision
		metadata["backup_restore_proof_missing_facts"] = proof.MissingFacts
		metadata["latest_backup_restore_proof_project_key"] = proof.Project.Key
		metadata["latest_backup_restore_proof_event_id"] = proof.EventID
		metadata["latest_backup_restore_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["database_restore_attempted"] = proof.DatabaseRestoreAttempted
		metadata["artifact_bytes_copied"] = proof.ArtifactBytesCopied
		metadata["artifact_bytes_deleted"] = proof.ArtifactBytesDeleted
		metadata["artifact_bytes_uploaded"] = proof.ArtifactBytesUploaded
		metadata["artifact_gc_attempted"] = proof.ArtifactGCAttempted
		metadata["commands_run"] = proof.CommandsRun
		metadata["area_matrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		metadata["backup_restore_evidence_binding_status"] = metadataString(proof.Metadata, "backup_restore_evidence_binding_status")
		metadata["backup_manifest_hash"] = metadataString(proof.Metadata, "backup_manifest_hash")
		metadata["backup_manifest_status"] = metadataString(proof.Metadata, "backup_manifest_status")
		metadata["backup_manifest_project_count"] = metadataInt64(proof.Metadata, "backup_manifest_project_count")
		metadata["backup_manifest_table_count"] = metadataInt64(proof.Metadata, "backup_manifest_table_count")
		metadata["restore_plan_status"] = metadataString(proof.Metadata, "restore_plan_status")
		metadata["restore_plan_manifest_hash"] = metadataString(proof.Metadata, "restore_plan_manifest_hash")
		metadata["restore_plan_item_count"] = metadataInt64(proof.Metadata, "restore_plan_item_count")
		metadata["artifact_integrity_status"] = metadataString(proof.Metadata, "artifact_integrity_status")
		metadata["artifact_integrity_checked_count"] = metadataInt64(proof.Metadata, "artifact_integrity_checked_count")
		metadata["artifact_integrity_failed_count"] = metadataInt64(proof.Metadata, "artifact_integrity_failed_count")
		metadata["artifact_archive_preview_status"] = metadataString(proof.Metadata, "artifact_archive_preview_status")
		metadata["artifact_archive_preview_total_artifacts"] = metadataInt64(proof.Metadata, "artifact_archive_preview_total_artifacts")
		metadata["artifact_archive_preview_external_refs"] = metadataInt64(proof.Metadata, "artifact_archive_preview_external_refs")
		metadata["artifact_archive_preview_needs_policy"] = metadataInt64(proof.Metadata, "artifact_archive_preview_needs_policy")
		metadata["artifact_archive_preview_project_write_attempted"] = metadataBool(proof.Metadata, "artifact_archive_preview_project_write_attempted")
		metadata["artifact_archive_preview_storage_write_attempted"] = metadataBool(proof.Metadata, "artifact_archive_preview_storage_write_attempted")
		metadata["artifact_archive_preview_delete_attempted"] = metadataBool(proof.Metadata, "artifact_archive_preview_delete_attempted")
		bindingBlockers := backupRestoreProofMetadataBindingBlockers(proof.Metadata)
		if len(bindingBlockers) > 0 {
			metadata["backup_restore_evidence_binding_blockers"] = bindingBlockers
		}
		currentBindingBlockers := []string{}
		metadata["backup_restore_current_binding_comparison_mode"] = "stable_safety_fields_excluding_append_only_manifest_hash"
		metadata["backup_restore_current_binding_excluded_fields"] = backupRestoreProofCurrentBindingHashFields
		if parts.BackupRestoreCurrentBindingError != "" {
			currentBindingBlockers = []string{"backup_restore_current_binding_query_failed"}
			metadata["backup_restore_current_binding_query_error"] = parts.BackupRestoreCurrentBindingError
			metadata["backup_restore_current_binding_bound"] = false
		} else if parts.BackupRestoreCurrentBinding == nil {
			currentBindingBlockers = []string{"backup_restore_current_binding_missing"}
			metadata["backup_restore_current_binding_bound"] = false
		} else {
			addBackupRestoreBindingMetadataWithPrefix(metadata, "current_", parts.BackupRestoreCurrentBinding)
			currentBindingBlockers = backupRestoreProofCurrentBindingBlockers(proof.Metadata, parts.BackupRestoreCurrentBinding)
			metadata["backup_restore_current_binding_bound"] = len(currentBindingBlockers) == 0
			if len(currentBindingBlockers) > 0 {
				metadata["backup_restore_current_binding_blockers"] = currentBindingBlockers
			}
		}
		if !completionAuditProofProjectMatches(proof.Project) {
			blockedBy = []string{"backup_restore_proof_project_mismatch"}
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if backupRestoreProofCompletesAudit(proof) && len(currentBindingBlockers) == 0 {
			status = "complete"
			message = "backup, restore and artifact retention proof has been recorded and current binding still matches"
			blockedBy = []string{}
			metadata["backup_restore_gate_passed"] = true
		} else if backupRestoreProofCompletesAudit(proof) {
			status = "blocked"
			message = "backup restore proof has been recorded, but current backup/restore/artifact binding is stale or unavailable"
			blockedBy = uniqueStrings(append([]string{}, currentBindingBlockers...))
			metadata["backup_restore_proof_recorded"] = true
		} else if proof.ProofStatus == "blocked" || proof.Status == "blocked" {
			status = "blocked"
			message = "backup restore proof is blocked"
			blockedBy = []string{"backup_restore_proof_blocked"}
		} else if containsCompletionAuditString(currentBindingBlockers, "backup_restore_current_binding_query_failed") {
			status = "blocked"
			message = "backup restore proof is incomplete, and current binding could not be queried"
			blockedBy = uniqueStrings(append([]string{"backup_restore_proof_incomplete"}, currentBindingBlockers...))
		} else {
			blockedBy = uniqueStrings(append([]string{"backup_restore_proof_incomplete"}, bindingBlockers...))
		}
	}
	return CompletionAuditItem{
		Key:      "E6_backup_restore_artifact_retention",
		Category: "backup_restore",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"GET /api/v1/backup/manifest",
			"GET /api/v1/backup/restore-plan",
			"GET /api/v1/artifacts/integrity",
			"docs/history/v1.0/contracts/artifact-backup-restore-contract.md",
			"docs/history/v1.0/contracts/object-artifact-retention-contract.md",
		},
		RequiredEvidence: []string{
			"backup manifest covers PostgreSQL metadata and AreaFlow-owned artifact metadata",
			"restore dry-run identifies metadata-only history and object verifier limits",
			"archive preview does not copy, upload, delete or GC artifact bytes",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow completion backup-restore-proof record areamatrix --status complete --fact <required_fact> --json",
		Metadata:    metadata,
	}
}

func completionOperationsReadinessItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "operations readiness still needs install/migrate/start/register smoke and scoped ops proof"
	blockedBy := []string{"install_migrate_start_register_smoke_missing", "support_bundle_preview_missing", "migration_ledger_readiness_missing"}
	metadata := map[string]any{}
	if parts.OperationsReadinessError != "" {
		status = "blocked"
		message = "operations readiness could not be queried"
		blockedBy = append(blockedBy, "operations_readiness_query_failed")
		metadata["query_error"] = parts.OperationsReadinessError
	} else if parts.OperationsReadiness != nil {
		readiness := parts.OperationsReadiness
		metadata["operations_status"] = readiness.Status
		metadata["service_status"] = readiness.ServiceStatus.Status
		metadata["support_bundle_status"] = readiness.SupportBundle.Status
		metadata["support_bundle_mode"] = readiness.SupportBundle.Mode
		metadata["support_bundle_metadata_only"] = readiness.SupportBundle.SafetyFacts["metadata_only"]
		metadata["support_bundle_export_open"] = readiness.SupportBundle.SafetyFacts["export_open"]
		metadata["support_bundle_secret_values_included"] = readiness.SupportBundle.SafetyFacts["secret_values_included"]
		metadata["support_bundle_prompt_text_included"] = readiness.SupportBundle.SafetyFacts["prompt_text_included"]
		metadata["support_bundle_user_file_contents_included"] = readiness.SupportBundle.SafetyFacts["user_file_contents_included"]
		metadata["support_bundle_raw_artifact_contents_included"] = readiness.SupportBundle.SafetyFacts["raw_artifact_contents_included"]
		metadata["support_bundle_unredacted_logs_included"] = readiness.SupportBundle.SafetyFacts["unredacted_logs_included"]
		metadata["support_bundle_sensitive_exclusion_count"] = len(readiness.SupportBundle.ExcludedSensitiveContent)
		metadata["migration_ledger_status"] = readiness.MigrationLedger.Status
		metadata["telemetry_default"] = readiness.TelemetryDefault
		metadata["managed_ops_status"] = readiness.ManagedOpsStatus
		metadata["support_export_status"] = readiness.SupportExportStatus
		for _, item := range readiness.Items {
			if item.Key == "install_migrate_start_register_smoke" {
				metadata["latest_operations_smoke_proof_key"] = metadataString(item.Metadata, "latest_smoke_proof_key")
				metadata["latest_operations_smoke_proof_evidence_uri"] = metadataString(item.Metadata, "latest_smoke_proof_uri")
				metadata["latest_operations_smoke_proof_event_id"] = metadataInt64(item.Metadata, "latest_smoke_proof_event_id")
				metadata["latest_operations_smoke_proof_fresh"] = metadataBool(item.Metadata, "latest_smoke_proof_fresh")
				metadata["latest_operations_smoke_proof_freshness_status"] = metadataString(item.Metadata, "latest_smoke_proof_freshness_status")
				metadata["latest_operations_smoke_proof_age_seconds"] = metadataInt64(item.Metadata, "latest_smoke_proof_age_seconds")
				metadata["operations_smoke_proof_max_age_seconds"] = metadataInt64(item.Metadata, "smoke_proof_max_age_seconds")
			}
			if item.Key == "metadata_only_support_bundle_preview" {
				metadata["support_bundle_blockers"] = item.BlockedBy
			}
		}
		blockedBy = operationsReadinessBlockers(*readiness)
		switch readiness.Status {
		case "ready":
			status = "complete"
			message = "operations readiness evidence is complete for v1.0 scope"
		case "blocked":
			status = "blocked"
			message = "operations readiness has blocked items"
		default:
			status = "incomplete"
			message = "operations readiness is scoped, but still needs attention before 100%"
		}
		if len(blockedBy) == 0 && status != "complete" {
			blockedBy = []string{"operations_readiness_not_ready"}
		}
		return CompletionAuditItem{
			Key:      "E7_operations_readiness",
			Category: "operations",
			Status:   status,
			Message:  message,
			EvidenceRefs: []string{
				"GET /api/v1/ops/readiness",
				"GET /api/v1/ops/support-bundle-preview",
				"GET /api/v1/ops/migration-ledger-readiness",
				"areaflow ops readiness --json",
				"areaflow support bundle-preview --json",
				"docs/operations/deployment.md",
				"docs/history/v1.0/milestones/v0.9-desktop-shell.md",
			},
			RequiredEvidence: []string{
				"install / migrate / start / project register smoke",
				"metadata-only support bundle preview with redaction proof",
				"telemetry default local-only proof",
				"migration ledger preflight/apply/verify/remediation proof",
			},
			BlockedBy:   blockedBy,
			NextCommand: "areaflow ops readiness --json",
			Metadata:    metadata,
		}
	}
	if parts.LocalServiceStatusError != "" {
		status = "blocked"
		message = "local service status could not be queried"
		blockedBy = append(blockedBy, "local_service_status_query_failed")
		metadata["query_error"] = parts.LocalServiceStatusError
	}
	if parts.LocalServiceStatus != nil {
		metadata["service_status"] = parts.LocalServiceStatus.Status
		metadata["service_mode"] = parts.LocalServiceStatus.Mode
	}
	return CompletionAuditItem{
		Key:      "E7_operations_readiness",
		Category: "operations",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"GET /api/v1/service/status",
			"GET /api/v1/ops/readiness",
			"docs/operations/deployment.md",
			"docs/history/v1.0/milestones/v0.9-desktop-shell.md",
		},
		RequiredEvidence: []string{
			"install / migrate / start / project register smoke",
			"metadata-only support bundle preview with redaction proof",
			"telemetry default local-only proof",
			"migration ledger preflight/apply/verify/remediation proof",
		},
		BlockedBy:   blockedBy,
		NextCommand: "areaflow ops readiness --json",
		Metadata:    metadata,
	}
}

func operationsReadinessBlockers(readiness OperationsReadiness) []string {
	blockers := []string{}
	for _, item := range readiness.Items {
		if item.Status != "blocked" && item.Status != "needs_attention" {
			continue
		}
		blockers = append(blockers, item.BlockedBy...)
		if len(item.BlockedBy) == 0 {
			blockers = append(blockers, item.Key)
		}
	}
	return uniqueStrings(blockers)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func removeString(values []string, target string) []string {
	out := []string{}
	for _, value := range values {
		if value == target {
			continue
		}
		out = append(out, value)
	}
	return out
}

func containsCompletionAuditString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func completionSecurityPermissionIsolationItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "incomplete"
	message := "security boundary readiness exists, but project isolation and permission evidence must be complete"
	blockedBy := []string{"project_isolation_smoke_missing", "audit_gap_closure_missing"}
	metadata := map[string]any{}
	forbiddenSecurityOpen := false
	if parts.SecurityBoundaryReadinessError != "" {
		status = "blocked"
		message = "security boundary readiness could not be queried"
		blockedBy = append(blockedBy, "security_boundary_query_failed")
		metadata["query_error"] = parts.SecurityBoundaryReadinessError
	}
	if parts.SecurityBoundaryReadiness != nil {
		metadata["security_boundary_status"] = parts.SecurityBoundaryReadiness.Status
		metadata["secret_resolve_open"] = parts.SecurityBoundaryReadiness.SecretResolveOpen
		metadata["remote_worker_credentials_open"] = parts.SecurityBoundaryReadiness.RemoteWorkerCredentialsOpen
		if parts.SecurityBoundaryReadiness.SecretResolveOpen || parts.SecurityBoundaryReadiness.RemoteWorkerCredentialsOpen ||
			parts.SecurityBoundaryReadiness.AuthorizationChanged {
			forbiddenSecurityOpen = true
			status = "blocked"
			message = "security boundary readiness opened a forbidden v1.0 capability"
			blockedBy = append(blockedBy, "security_boundary_opened_forbidden_capability")
		}
	}
	if parts.SecurityClosureProofError != "" {
		status = "blocked"
		message = "security closure proof could not be queried"
		blockedBy = append(blockedBy, "security_closure_proof_query_failed")
		metadata["security_closure_proof_query_error"] = parts.SecurityClosureProofError
	} else if parts.SecurityClosureProof != nil {
		proof := *parts.SecurityClosureProof
		metadata["security_closure_proof_status"] = proof.ProofStatus
		metadata["security_closure_proof_decision"] = proof.Decision
		metadata["security_closure_proof_missing_facts"] = proof.MissingFacts
		metadata["latest_security_closure_proof_project_key"] = proof.Project.Key
		metadata["latest_security_closure_proof_event_id"] = proof.EventID
		metadata["latest_security_closure_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["project_write_attempted"] = proof.ProjectWriteAttempted
		metadata["execution_write_attempted"] = proof.ExecutionWriteAttempted
		metadata["authorization_changed"] = proof.AuthorizationChanged
		metadata["secret_plaintext_read"] = proof.SecretPlaintextRead
		metadata["remote_worker_credentials_issued"] = proof.RemoteWorkerCredentialsIssued
		metadata["commands_run"] = proof.CommandsRun
		metadata["area_matrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		metadata["security_closure_binding_status"] = metadataString(proof.Metadata, "security_closure_binding_status")
		metadata["security_closure_binding_hash"] = metadataString(proof.Metadata, "security_closure_binding_hash")
		metadata["security_boundary_mode"] = metadataString(proof.Metadata, "security_boundary_mode")
		metadata["security_boundary_capabilities_hash"] = metadataString(proof.Metadata, "security_boundary_capabilities_hash")
		metadata["security_boundary_forbidden_actions_hash"] = metadataString(proof.Metadata, "security_boundary_forbidden_actions_hash")
		metadata["permission_doctor_status"] = metadataString(proof.Metadata, "permission_doctor_status")
		metadata["permission_doctor_fail_count"] = metadataInt64(proof.Metadata, "permission_doctor_fail_count")
		metadata["permission_doctor_warn_count"] = metadataInt64(proof.Metadata, "permission_doctor_warn_count")
		metadata["audit_coverage_status"] = metadataString(proof.Metadata, "audit_coverage_status")
		metadata["audit_coverage_scope"] = metadataString(proof.Metadata, "audit_coverage_scope")
		metadata["audit_coverage_gap_requirements"] = metadataInt64(proof.Metadata, "audit_coverage_gap_requirements")
		metadata["audit_coverage_missing_action_count"] = metadataInt64(proof.Metadata, "audit_coverage_missing_action_count")
		bindingBlockers := securityClosureProofMetadataBindingBlockers(proof.Metadata)
		if len(bindingBlockers) > 0 {
			metadata["security_closure_binding_blockers"] = bindingBlockers
		}
		currentBindingBlockers := []string{}
		if parts.SecurityClosureCurrentBindingError != "" {
			currentBindingBlockers = []string{"security_closure_current_binding_query_failed"}
			metadata["security_closure_current_binding_query_error"] = parts.SecurityClosureCurrentBindingError
			metadata["security_closure_current_binding_bound"] = false
		} else if parts.SecurityClosureCurrentBinding == nil {
			currentBindingBlockers = []string{"security_closure_current_binding_missing"}
			metadata["security_closure_current_binding_bound"] = false
		} else {
			addSecurityClosureBindingMetadataWithPrefix(metadata, "current_", parts.SecurityClosureCurrentBinding)
			currentBindingBlockers = securityClosureProofCurrentBindingBlockers(proof.Metadata, parts.SecurityClosureCurrentBinding)
			metadata["security_closure_current_binding_bound"] = len(currentBindingBlockers) == 0
			if len(currentBindingBlockers) > 0 {
				metadata["security_closure_current_binding_blockers"] = currentBindingBlockers
			}
		}
		if !completionAuditProofProjectMatches(proof.Project) {
			blockedBy = []string{"security_closure_proof_project_mismatch"}
			metadata["expected_project_key"] = completionAuditTargetProjectKey
		} else if securityClosureProofCompletesAudit(proof) && len(currentBindingBlockers) == 0 {
			blockedBy = removeString(blockedBy, "project_isolation_smoke_missing")
			blockedBy = removeString(blockedBy, "audit_gap_closure_missing")
			metadata["security_closure_gate_passed"] = true
			if !forbiddenSecurityOpen && parts.SecurityBoundaryReadinessError == "" {
				status = "complete"
				message = "security, permission and isolation closure proof has been recorded and current binding still matches"
			}
		} else if securityClosureProofCompletesAudit(proof) {
			status = "blocked"
			message = "security closure proof has been recorded, but current security/permission/audit binding is stale or unavailable"
			blockedBy = uniqueStrings(append([]string{}, currentBindingBlockers...))
			metadata["security_closure_proof_recorded"] = true
		} else if proof.ProofStatus == "blocked" {
			status = "blocked"
			message = "security closure proof is blocked"
			blockedBy = []string{"security_closure_proof_blocked"}
		} else if containsCompletionAuditString(currentBindingBlockers, "security_closure_current_binding_query_failed") {
			status = "blocked"
			message = "security closure proof is incomplete, and current binding could not be queried"
			blockedBy = uniqueStrings(append([]string{"security_closure_proof_incomplete"}, currentBindingBlockers...))
		} else {
			blockedBy = uniqueStrings(append([]string{"security_closure_proof_incomplete"}, bindingBlockers...))
		}
	}
	return CompletionAuditItem{
		Key:      "E8_security_permission_isolation",
		Category: "security",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"GET /api/v1/security/boundary-readiness",
			"GET /api/v1/permissions/doctor",
			"GET /api/v1/audit/coverage",
			"docs/history/v1.0/evidence/multi-project-isolation-evidence.md",
		},
		RequiredEvidence: []string{
			"project_key isolation covers workflow, run, lease, artifact, secret and audit",
			"permission doctor proves default read-only and deny-first policy",
			"audit coverage covers enabled capabilities",
			"auth/team/token/secret/remote worker remain readiness-only",
		},
		BlockedBy:   blockedBy,
		NextCommand: "AREAFLOW_DATABASE_URL=... ./scripts/smoke-project-isolation.sh",
		Metadata:    metadata,
	}
}

func addSecurityClosureBindingMetadataWithPrefix(metadata map[string]any, prefix string, binding map[string]any) {
	for key, value := range binding {
		metadata[prefix+key] = value
	}
}

func completionProtectedPathProofItem(parts CompletionAuditParts) CompletionAuditItem {
	status := "blocked"
	message := "AreaMatrix protected path proof is missing from the report input"
	blockedBy := []string{"protected_path_proof_missing"}
	metadata := map[string]any{
		"protected_path_proof_status": completionAuditProtectedPathStatus(parts),
		"api_runs_git_status":         false,
	}
	if parts.ProtectedPathProofError != "" {
		message = "AreaMatrix protected path proof could not be queried"
		blockedBy = []string{"protected_path_proof_query_failed"}
		metadata["query_error"] = parts.ProtectedPathProofError
	}
	if parts.ProtectedPathProof != nil {
		proof := parts.ProtectedPathProof
		bindingBlockers := protectedPathProofMetadataBindingBlockers(proof.Metadata)
		metadata["latest_proof_status"] = proof.Status
		metadata["latest_proof_decision"] = proof.Decision
		metadata["latest_proof_project_key"] = proof.Project.Key
		metadata["expected_project_key"] = completionAuditTargetProjectKey
		metadata["latest_proof_event_id"] = proof.EventID
		metadata["latest_proof_audit_event_id"] = proof.AuditEventID
		metadata["latest_proof_summary"] = metadataString(proof.Metadata, "summary")
		metadata["latest_proof_evidence_uri"] = metadataString(proof.Metadata, "evidence_uri")
		metadata["latest_proof_traceable_evidence"] = proofMetadataHasTraceableEvidence(proof.Metadata)
		metadata["authorized_approval_id"] = metadataString(proof.Metadata, "authorized_approval_id")
		metadata["authorized_allowed_paths"] = metadataStringSlice(proof.Metadata, "authorized_allowed_paths")
		metadata["authorized_dirty_output_hash"] = metadataString(proof.Metadata, "authorized_dirty_output_hash")
		metadata["authorized_reviewer"] = metadataString(proof.Metadata, "authorized_reviewer")
		metadata["authorized_rollback_evidence_uri"] = metadataString(proof.Metadata, "authorized_rollback_evidence_uri")
		metadata["authorized_touched_paths"] = metadataStringSlice(proof.Metadata, "authorized_touched_paths")
		metadata["authorized_proof_complete"] = proof.ProofStatus != "authorized" || protectedPathProofAuthorizedMetadataComplete(proof.Metadata)
		metadata["protected_path_proof_binding_status"] = metadataString(proof.Metadata, "protected_path_proof_binding_status")
		metadata["protected_path_proof_binding_blockers"] = bindingBlockers
		metadata["protected_path_set_hash"] = metadataString(proof.Metadata, "protected_path_set_hash")
		metadata["protected_path_set_count"] = metadataInt64(proof.Metadata, "protected_path_set_count")
		metadata["git_status_output_empty"] = metadataBool(proof.Metadata, "git_status_output_empty")
		metadata["git_status_output_hash"] = proof.GitStatusOutputHash
		metadata["git_status_output_lines"] = proof.GitStatusOutputLines
		metadata["git_status_run_by_command"] = proof.GitStatusRunByCommand
		metadata["area_matrix_protected_paths_touched"] = proof.AreaMatrixProtectedPathsTouched
		if !completionAuditProofProjectMatches(proof.Project) {
			message = "AreaMatrix protected path proof belongs to a different project"
			blockedBy = []string{"protected_path_proof_project_mismatch"}
		} else if (proof.ProofStatus == "clean" || proof.ProofStatus == "authorized") && !proofMetadataHasTraceableEvidence(proof.Metadata) {
			message = "AreaMatrix protected path proof lacks traceable evidence"
			blockedBy = []string{"protected_path_proof_evidence_missing"}
		} else if proof.ProofStatus == "authorized" && !protectedPathProofAuthorizedMetadataComplete(proof.Metadata) {
			message = "AreaMatrix protected path proof authorization metadata is incomplete"
			blockedBy = []string{"protected_path_proof_authorization_incomplete"}
		} else if (proof.ProofStatus == "clean" || proof.ProofStatus == "authorized") && len(bindingBlockers) > 0 {
			message = "AreaMatrix protected path proof binding metadata is incomplete"
			blockedBy = []string{"protected_path_proof_binding_incomplete"}
		} else if !protectedPathProofCompletesAudit(*proof) {
			message = "AreaMatrix protected path proof is present but not clean or authorized"
			blockedBy = []string{"protected_path_proof_not_clean"}
		}
	}
	if completionAuditProtectedPathStatus(parts) == "complete" {
		status = "complete"
		message = "AreaMatrix protected path proof is clean"
		blockedBy = nil
	}
	return CompletionAuditItem{
		Key:      "E9_areamatrix_protected_path_proof",
		Category: "protected_paths",
		Status:   status,
		Message:  message,
		EvidenceRefs: []string{
			"docs/history/v1.0/contracts/completion-audit-contract.md",
			"docs/history/v1.0/contracts/v1.0-stable-platform-contract.md",
		},
		RequiredEvidence: []string{
			"AreaMatrix protected path git status command has no output, or changes are authorized by completed cross-repo task",
		},
		BlockedBy:   blockedBy,
		NextCommand: completionAuditProtectedPathProofNextCommand,
		Metadata:    metadata,
	}
}

const completionAuditProtectedPathProofNextCommand = "git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json; if output is empty record: areaflow completion protected-path-proof record areamatrix --status clean --summary \"AreaMatrix protected path git status returned no output\" --evidence-uri local:areamatrix-protected-path-git-status --json; if output is reviewed and explicitly authorized record: areaflow completion protected-path-proof record areamatrix --status authorized --git-status-output <status-output> --dirty-output-hash <sha256> --approval-id <approval-id> --allowed-path <path> --reviewer <reviewer> --rollback-evidence-uri <uri> --summary <summary> --evidence-uri <uri> --json"

func protectedPathProofCompletesAudit(proof ProtectedPathProof) bool {
	if proof.Status != "recorded" || proof.Decision != "allowed" {
		return false
	}
	if proof.ProofStatus != "clean" && proof.ProofStatus != "authorized" {
		return false
	}
	if !proofMetadataHasTraceableEvidence(proof.Metadata) {
		return false
	}
	if len(protectedPathProofMetadataBindingBlockers(proof.Metadata)) > 0 {
		return false
	}
	if proof.ProofStatus == "authorized" && !protectedPathProofAuthorizedMetadataComplete(proof.Metadata) {
		return false
	}
	if proof.AreaMatrixProtectedPathsTouched && proof.ProofStatus != "authorized" {
		return false
	}
	return true
}

func defaultCompletionAuditDeferredV1x() []string {
	return []string{
		"restore_apply",
		"release_publish_apply",
		"secret_resolve",
		"remote_worker",
		"team_console",
		"plugin_execution",
		"external_integrations_webhooks",
		"object_artifact_store_gc_delete",
		"budget_quota_enforcement",
		"managed_ops_upgrade_support_export",
	}
}

func (a CompletionAudit) SummaryLine() string {
	return fmt.Sprintf("completion audit: %s scope=%s items=%d", a.Status, a.Scope, len(a.Items))
}
