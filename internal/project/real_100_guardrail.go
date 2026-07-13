package project

import "strings"

const (
	Real100StatusBlocked          = "blocked"
	ReleasePreviewReadinessScope  = "areaflow_release_preview_only"
	CompletionAuditReadinessScope = "completion_audit_evidence_only"
)

type Real100Guardrail struct {
	ClaimScope                 string           `json:"claim_scope,omitempty"`
	NotReal100                 bool             `json:"not_real_100,omitempty"`
	EvidenceOnly               bool             `json:"evidence_only,omitempty"`
	StatusAloneIsNotCompletion bool             `json:"status_alone_is_not_completion,omitempty"`
	ReleaseCandidateDecision   string           `json:"release_candidate_decision,omitempty"`
	ReadinessScope             string           `json:"readiness_scope,omitempty"`
	Real100Status              string           `json:"real_100_status,omitempty"`
	Real100Blockers            []string         `json:"real_100_blockers,omitempty"`
	Real100Breakdown           Real100Breakdown `json:"real_100_breakdown,omitempty"`
}

type Real100Breakdown struct {
	NeedsExactAuthorization  []Real100BreakdownItem `json:"needs_exact_authorization,omitempty"`
	NeedsRealAreaMatrixWrite []Real100BreakdownItem `json:"needs_real_areamatrix_write,omitempty"`
	AreaFlowOnlyCanContinue  []Real100BreakdownItem `json:"areaflow_only_can_continue,omitempty"`
	CompletedEvidence        []Real100BreakdownItem `json:"completed_evidence,omitempty"`
}

type Real100BreakdownItem struct {
	Key                         string   `json:"key"`
	Status                      string   `json:"status,omitempty"`
	Message                     string   `json:"message,omitempty"`
	RequiredAuthorizationPhrase string   `json:"required_authorization_phrase,omitempty"`
	Blockers                    []string `json:"blockers,omitempty"`
	EvidenceRefs                []string `json:"evidence_refs,omitempty"`
	NextCommand                 string   `json:"next_command,omitempty"`
}

func ReleasePreviewReal100Guardrail() Real100Guardrail {
	return Real100Guardrail{
		ClaimScope:                 ReleasePreviewReadinessScope,
		NotReal100:                 true,
		EvidenceOnly:               true,
		StatusAloneIsNotCompletion: true,
		ReleaseCandidateDecision:   "not_release_candidate_evidence",
		ReadinessScope:             ReleasePreviewReadinessScope,
		Real100Status:              Real100StatusBlocked,
		Real100Blockers:            Real100ReleasePreviewBlockers(),
		Real100Breakdown:           Real100ReleasePreviewBreakdown(),
	}
}

func CompletionAuditReal100Guardrail() Real100Guardrail {
	return Real100Guardrail{
		ClaimScope:                 CompletionAuditReadinessScope,
		NotReal100:                 true,
		EvidenceOnly:               true,
		StatusAloneIsNotCompletion: true,
		ReleaseCandidateDecision:   "requires_release_candidate_snapshot",
		ReadinessScope:             CompletionAuditReadinessScope,
		Real100Status:              Real100StatusBlocked,
		Real100Blockers:            Real100CompletionAuditBlockers(),
		Real100Breakdown:           Real100CompletionAuditBreakdown(nil),
	}
}

func CompletionAuditReal100GuardrailForItems(items []CompletionAuditItem) Real100Guardrail {
	guardrail := CompletionAuditReal100Guardrail()
	guardrail.Real100Breakdown = Real100CompletionAuditBreakdown(items)
	guardrail.Real100Blockers = Real100CompletionAuditBlockersForBreakdown(guardrail.Real100Breakdown)
	return guardrail
}

func NormalizeReal100Guardrail(guardrail Real100Guardrail, fallback Real100Guardrail) Real100Guardrail {
	if guardrail.ClaimScope == "" {
		guardrail.ClaimScope = fallback.ClaimScope
	}
	if !guardrail.NotReal100 {
		guardrail.NotReal100 = fallback.NotReal100
	}
	if !guardrail.EvidenceOnly {
		guardrail.EvidenceOnly = fallback.EvidenceOnly
	}
	if !guardrail.StatusAloneIsNotCompletion {
		guardrail.StatusAloneIsNotCompletion = fallback.StatusAloneIsNotCompletion
	}
	if guardrail.ReleaseCandidateDecision == "" {
		guardrail.ReleaseCandidateDecision = fallback.ReleaseCandidateDecision
	}
	if guardrail.ReadinessScope == "" {
		guardrail.ReadinessScope = fallback.ReadinessScope
	}
	if guardrail.Real100Status == "" {
		guardrail.Real100Status = fallback.Real100Status
	}
	if len(guardrail.Real100Blockers) == 0 {
		guardrail.Real100Blockers = fallback.Real100Blockers
	}
	if real100BreakdownEmpty(guardrail.Real100Breakdown) {
		guardrail.Real100Breakdown = fallback.Real100Breakdown
	}
	return Real100Guardrail{
		ClaimScope:                 guardrail.ClaimScope,
		NotReal100:                 guardrail.NotReal100,
		EvidenceOnly:               guardrail.EvidenceOnly,
		StatusAloneIsNotCompletion: guardrail.StatusAloneIsNotCompletion,
		ReleaseCandidateDecision:   guardrail.ReleaseCandidateDecision,
		ReadinessScope:             guardrail.ReadinessScope,
		Real100Status:              guardrail.Real100Status,
		Real100Blockers:            append([]string{}, guardrail.Real100Blockers...),
		Real100Breakdown:           copyReal100Breakdown(guardrail.Real100Breakdown),
	}
}

func Real100ReleasePreviewBlockers() []string {
	return []string{
		"package_a_status_projection_apply_provenance_missing",
		"real_areamatrix_read_only_shim_not_landed",
		"real_areamatrix_execution_cutover_not_proven",
		"real_areamatrix_archive_not_proven",
		"real_areamatrix_shim_retirement_not_proven",
	}
}

func Real100CompletionAuditBlockers() []string {
	return append(Real100ReleasePreviewBlockers(), "release_candidate_snapshot_not_ready")
}

func Real100CompletionAuditBlockersForBreakdown(breakdown Real100Breakdown) []string {
	if real100BreakdownEmpty(breakdown) {
		return Real100CompletionAuditBlockers()
	}

	blockers := []string{}
	needsPackageA := false
	for _, item := range breakdown.NeedsExactAuthorization {
		itemBlockers := real100CanonicalBlockers(item.Blockers)
		if real100HasAnyBlocker(itemBlockers, "package_a_status_projection_not_applied", "package_a_status_projection_apply_provenance_missing") {
			needsPackageA = true
		}
		blockers = append(blockers, itemBlockers...)
	}
	for _, item := range breakdown.NeedsRealAreaMatrixWrite {
		itemBlockers := real100CanonicalBlockers(item.Blockers)
		if real100HasAnyBlocker(itemBlockers, "package_a_status_projection_not_applied", "package_a_status_projection_apply_provenance_missing") {
			needsPackageA = true
		}
		blockers = append(blockers, itemBlockers...)
	}
	if needsPackageA {
		blockers = append(blockers, "real_areamatrix_read_only_shim_not_landed")
	}
	for _, item := range breakdown.AreaFlowOnlyCanContinue {
		blockers = append(blockers, real100CanonicalBlockers(item.Blockers)...)
	}
	return real100OrderCompletionBlockers(uniqueStrings(blockers))
}

func Real100ReleasePreviewBreakdown() Real100Breakdown {
	return Real100Breakdown{
		NeedsExactAuthorization: []Real100BreakdownItem{
			{
				Key:                         "package_a_exact_authorization",
				Status:                      Real100StatusBlocked,
				Message:                     "Package A status projection still needs sealed apply provenance before real 100%",
				RequiredAuthorizationPhrase: "",
				Blockers:                    []string{"package_a_status_projection_apply_provenance_missing"},
			},
		},
		NeedsRealAreaMatrixWrite: []Real100BreakdownItem{
			{
				Key:      "package_a_status_projection_apply",
				Status:   Real100StatusBlocked,
				Message:  "real AreaMatrix .areaflow/status.json must remain stable and Package A apply provenance must be sealed",
				Blockers: []string{"package_a_status_projection_apply_provenance_missing"},
			},
			{
				Key:      "real_areamatrix_read_only_shim",
				Status:   Real100StatusBlocked,
				Message:  "real AreaMatrix compatibility shim files are not landed",
				Blockers: []string{"real_areamatrix_read_only_shim_not_landed"},
			},
			{
				Key:      "real_areamatrix_execution_cutover",
				Status:   Real100StatusBlocked,
				Message:  "real AreaMatrix execution cutover has not been proven",
				Blockers: []string{"real_areamatrix_execution_cutover_not_proven"},
			},
			{
				Key:      "real_areamatrix_archive",
				Status:   Real100StatusBlocked,
				Message:  "real AreaMatrix archive proof has not been accepted",
				Blockers: []string{"real_areamatrix_archive_not_proven"},
			},
			{
				Key:      "real_areamatrix_shim_retirement",
				Status:   Real100StatusBlocked,
				Message:  "real AreaMatrix shim retirement proof has not been accepted",
				Blockers: []string{"real_areamatrix_shim_retirement_not_proven"},
			},
		},
		AreaFlowOnlyCanContinue: []Real100BreakdownItem{
			{
				Key:     "areaflow_release_preview_evidence",
				Status:  "available",
				Message: "AreaFlow-only release preview, guardrail and evidence work can continue without writing AreaMatrix",
			},
			{
				Key:     "execution_forwarding_v1_readiness",
				Status:  "available",
				Message: "readiness, packet, gate and rollback proof hardening can continue while real forwarding remains closed",
			},
		},
	}
}

func Real100CompletionAuditBreakdown(items []CompletionAuditItem) Real100Breakdown {
	if len(items) == 0 {
		breakdown := Real100ReleasePreviewBreakdown()
		breakdown.AreaFlowOnlyCanContinue = append(breakdown.AreaFlowOnlyCanContinue, Real100BreakdownItem{
			Key:      "release_candidate_snapshot_readiness",
			Status:   Real100StatusBlocked,
			Message:  "real release-candidate snapshot readiness still needs evidence-only closure",
			Blockers: []string{"release_candidate_snapshot_not_ready"},
		})
		return breakdown
	}

	breakdown := Real100Breakdown{}
	for _, item := range items {
		if item.Status == "complete" {
			breakdown.CompletedEvidence = append(breakdown.CompletedEvidence, real100BreakdownItemFromAuditItem(item))
			addCompletionAuditGranularCompletedEvidence(&breakdown, item)
			continue
		}
		if item.Key == "E4_areamatrix_dogfood_completion" {
			addCompletionAuditDogfoodNeeds(&breakdown, item)
			addCompletionAuditGranularCompletedEvidence(&breakdown, item)
			continue
		}
		breakdown.AreaFlowOnlyCanContinue = append(breakdown.AreaFlowOnlyCanContinue, real100BreakdownItemFromAuditItem(item))
	}
	breakdown.AreaFlowOnlyCanContinue = append(breakdown.AreaFlowOnlyCanContinue, Real100BreakdownItem{
		Key:      "release_candidate_snapshot_readiness",
		Status:   Real100StatusBlocked,
		Message:  "real release-candidate snapshot readiness still needs evidence-only closure",
		Blockers: []string{"release_candidate_snapshot_not_ready"},
	})
	return normalizeReal100Breakdown(breakdown)
}

func addCompletionAuditDogfoodNeeds(breakdown *Real100Breakdown, item CompletionAuditItem) {
	blockers := uniqueStrings(item.BlockedBy)
	if real100HasPackageABlocker(blockers) {
		breakdown.NeedsExactAuthorization = append(breakdown.NeedsExactAuthorization, Real100BreakdownItem{
			Key:                         "package_a_exact_authorization",
			Status:                      item.Status,
			Message:                     "Package A status projection still has unresolved apply or provenance blockers",
			RequiredAuthorizationPhrase: "",
			Blockers:                    real100BlockersWithPrefix(blockers, "package_a_", "completion_audit_snapshot_package_a_"),
			NextCommand:                 real100PackageANextCommand,
		})
		breakdown.NeedsRealAreaMatrixWrite = append(breakdown.NeedsRealAreaMatrixWrite, Real100BreakdownItem{
			Key:          "package_a_status_projection_apply",
			Status:       item.Status,
			Message:      "real AreaMatrix .areaflow/status.json must remain stable and Package A apply provenance must be sealed",
			Blockers:     real100BlockersWithPrefix(blockers, "package_a_", "completion_audit_snapshot_package_a_"),
			EvidenceRefs: append([]string{}, item.EvidenceRefs...),
			NextCommand:  real100PackageANextCommand,
		})
	}
	if real100HasAnyBlocker(blockers, "real_areamatrix_read_only_shim_not_landed") {
		breakdown.NeedsRealAreaMatrixWrite = append(breakdown.NeedsRealAreaMatrixWrite, Real100BreakdownItem{
			Key:          "real_areamatrix_read_only_shim",
			Status:       item.Status,
			Message:      "real AreaMatrix compatibility shim files are not landed",
			Blockers:     []string{"real_areamatrix_read_only_shim_not_landed"},
			EvidenceRefs: append([]string{}, item.EvidenceRefs...),
			NextCommand:  real100ReadOnlyShimNextCommand,
		})
	}
	if real100HasAnyBlocker(blockers, "execution_cutover_not_complete") || real100HasBlockerPrefix(blockers, "execution_cutover_") {
		breakdown.NeedsRealAreaMatrixWrite = append(breakdown.NeedsRealAreaMatrixWrite, Real100BreakdownItem{
			Key:          "real_areamatrix_execution_cutover",
			Status:       item.Status,
			Message:      "real AreaMatrix execution cutover has not been proven",
			Blockers:     real100BlockersWithPrefix(blockers, "execution_cutover_"),
			EvidenceRefs: append([]string{}, item.EvidenceRefs...),
			NextCommand:  real100ExecutionCutoverNextCommand,
		})
	}
	if real100HasAnyBlocker(blockers, "real_areamatrix_archive_not_proven") || real100HasBlockerPrefix(blockers, "archive_") {
		breakdown.NeedsRealAreaMatrixWrite = append(breakdown.NeedsRealAreaMatrixWrite, Real100BreakdownItem{
			Key:          "real_areamatrix_archive",
			Status:       item.Status,
			Message:      "real AreaMatrix archive proof has not been accepted",
			Blockers:     real100BlockersWithPrefix(blockers, "real_areamatrix_archive", "archive_"),
			EvidenceRefs: append([]string{}, item.EvidenceRefs...),
			NextCommand:  real100ArchiveNextCommand,
		})
	}
	if real100HasAnyBlocker(blockers, "real_areamatrix_shim_retirement_not_proven") || real100HasBlockerPrefix(blockers, "shim_retirement_") {
		breakdown.NeedsRealAreaMatrixWrite = append(breakdown.NeedsRealAreaMatrixWrite, Real100BreakdownItem{
			Key:          "real_areamatrix_shim_retirement",
			Status:       item.Status,
			Message:      "real AreaMatrix shim retirement proof has not been accepted",
			Blockers:     real100BlockersWithPrefix(blockers, "real_areamatrix_shim_retirement", "shim_retirement_"),
			EvidenceRefs: append([]string{}, item.EvidenceRefs...),
			NextCommand:  real100ShimRetirementNextCommand,
		})
	}
}

const (
	real100PackageANextCommand         = "areaflow project status-projection-apply-packet areamatrix --json && areaflow project status-projection-apply-gate areamatrix --json"
	real100ReadOnlyShimNextCommand     = "make smoke-package-b-readiness"
	real100ExecutionCutoverNextCommand = "areaflow project execution-cutover-readiness areamatrix --json"
	real100ArchiveNextCommand          = "areaflow completion archive-proof record areamatrix --status complete --fact <required_fact> --json"
	real100ShimRetirementNextCommand   = "areaflow completion shim-retirement-proof record areamatrix --status complete --fact <required_fact> --json"
)

func addCompletionAuditGranularCompletedEvidence(breakdown *Real100Breakdown, item CompletionAuditItem) {
	if item.Metadata == nil {
		return
	}
	completed := []struct {
		key      string
		flag     string
		message  string
		eventKey string
		uriKey   string
	}{
		{"package_a_status_projection", "package_a_status_projection_ready", "Package A status projection evidence is stable", "", ""},
		{"real_areamatrix_execution_cutover_proof", "execution_cutover_gate_passed", "execution cutover proof has been accepted", "latest_execution_cutover_proof_event_id", "latest_execution_cutover_proof_evidence_uri"},
		{"real_areamatrix_archive_proof", "archive_gate_passed", "archive proof has been accepted", "latest_archive_proof_event_id", "latest_archive_proof_evidence_uri"},
		{"real_areamatrix_shim_retirement_proof", "shim_retirement_gate_passed", "shim retirement proof has been accepted", "latest_shim_retirement_proof_event_id", "latest_shim_retirement_proof_evidence_uri"},
	}
	for _, entry := range completed {
		if !metadataBool(item.Metadata, entry.flag) {
			continue
		}
		completedItem := Real100BreakdownItem{
			Key:     entry.key,
			Status:  "complete",
			Message: entry.message,
		}
		if entry.uriKey != "" {
			completedItem.EvidenceRefs = []string{metadataString(item.Metadata, entry.uriKey)}
		}
		if entry.eventKey != "" {
			completedItem.Blockers = []string{}
		}
		breakdown.CompletedEvidence = append(breakdown.CompletedEvidence, completedItem)
	}
}

func real100BreakdownItemFromAuditItem(item CompletionAuditItem) Real100BreakdownItem {
	return Real100BreakdownItem{
		Key:          item.Key,
		Status:       item.Status,
		Message:      item.Message,
		Blockers:     append([]string{}, item.BlockedBy...),
		EvidenceRefs: append([]string{}, item.EvidenceRefs...),
		NextCommand:  item.NextCommand,
	}
}

func real100HasPackageABlocker(blockers []string) bool {
	return real100HasBlockerPrefix(blockers, "package_a_") ||
		real100HasBlockerPrefix(blockers, "completion_audit_snapshot_package_a_")
}

func real100HasBlockerPrefix(blockers []string, prefixes ...string) bool {
	for _, blocker := range blockers {
		for _, prefix := range prefixes {
			if strings.HasPrefix(blocker, prefix) {
				return true
			}
		}
	}
	return false
}

func real100HasAnyBlocker(blockers []string, values ...string) bool {
	for _, blocker := range blockers {
		for _, value := range values {
			if blocker == value {
				return true
			}
		}
	}
	return false
}

func real100BlockersWithPrefix(blockers []string, prefixes ...string) []string {
	filtered := []string{}
	for _, blocker := range blockers {
		for _, prefix := range prefixes {
			if strings.HasPrefix(blocker, prefix) {
				filtered = append(filtered, blocker)
				break
			}
		}
	}
	if len(filtered) == 0 {
		return append([]string{}, blockers...)
	}
	return uniqueStrings(filtered)
}

func real100CanonicalBlockers(blockers []string) []string {
	canonical := []string{}
	for _, blocker := range blockers {
		switch {
		case blocker == "package_a_status_projection_apply_provenance_missing" ||
			blocker == "completion_audit_snapshot_package_a_apply_provenance_missing":
			canonical = append(canonical, "package_a_status_projection_apply_provenance_missing")
		case blocker == "package_a_status_projection_not_applied" ||
			blocker == "completion_audit_snapshot_package_a_not_applied" ||
			strings.HasPrefix(blocker, "package_a_") ||
			strings.HasPrefix(blocker, "completion_audit_snapshot_package_a_"):
			canonical = append(canonical, "package_a_status_projection_not_applied")
		case blocker == "real_areamatrix_read_only_shim_not_landed":
			canonical = append(canonical, "real_areamatrix_read_only_shim_not_landed")
		case blocker == "real_areamatrix_execution_cutover_not_proven" ||
			blocker == "execution_cutover_not_complete" ||
			strings.HasPrefix(blocker, "execution_cutover_"):
			canonical = append(canonical, "real_areamatrix_execution_cutover_not_proven")
		case blocker == "real_areamatrix_archive_not_proven" ||
			strings.HasPrefix(blocker, "archive_"):
			canonical = append(canonical, "real_areamatrix_archive_not_proven")
		case blocker == "real_areamatrix_shim_retirement_not_proven" ||
			strings.HasPrefix(blocker, "shim_retirement_"):
			canonical = append(canonical, "real_areamatrix_shim_retirement_not_proven")
		case blocker == "release_candidate_snapshot_not_ready":
			canonical = append(canonical, "release_candidate_snapshot_not_ready")
		}
	}
	return uniqueStrings(canonical)
}

func real100OrderCompletionBlockers(blockers []string) []string {
	seen := map[string]bool{}
	for _, blocker := range blockers {
		seen[blocker] = true
	}
	ordered := []string{}
	for _, blocker := range Real100CompletionAuditBlockers() {
		if seen[blocker] {
			ordered = append(ordered, blocker)
			delete(seen, blocker)
		}
	}
	for _, blocker := range blockers {
		if seen[blocker] {
			ordered = append(ordered, blocker)
			delete(seen, blocker)
		}
	}
	return ordered
}

func normalizeReal100Breakdown(breakdown Real100Breakdown) Real100Breakdown {
	return Real100Breakdown{
		NeedsExactAuthorization:  uniqueReal100BreakdownItems(breakdown.NeedsExactAuthorization),
		NeedsRealAreaMatrixWrite: uniqueReal100BreakdownItems(breakdown.NeedsRealAreaMatrixWrite),
		AreaFlowOnlyCanContinue:  uniqueReal100BreakdownItems(breakdown.AreaFlowOnlyCanContinue),
		CompletedEvidence:        uniqueReal100BreakdownItems(breakdown.CompletedEvidence),
	}
}

func uniqueReal100BreakdownItems(items []Real100BreakdownItem) []Real100BreakdownItem {
	seen := map[string]bool{}
	out := []Real100BreakdownItem{}
	for _, item := range items {
		if item.Key == "" || seen[item.Key] {
			continue
		}
		seen[item.Key] = true
		item.Blockers = uniqueStrings(item.Blockers)
		item.EvidenceRefs = uniqueStrings(item.EvidenceRefs)
		out = append(out, item)
	}
	return out
}

func real100BreakdownEmpty(breakdown Real100Breakdown) bool {
	return len(breakdown.NeedsExactAuthorization) == 0 &&
		len(breakdown.NeedsRealAreaMatrixWrite) == 0 &&
		len(breakdown.AreaFlowOnlyCanContinue) == 0 &&
		len(breakdown.CompletedEvidence) == 0
}

func copyReal100Breakdown(breakdown Real100Breakdown) Real100Breakdown {
	breakdown = normalizeReal100Breakdown(breakdown)
	return Real100Breakdown{
		NeedsExactAuthorization:  copyReal100BreakdownItems(breakdown.NeedsExactAuthorization),
		NeedsRealAreaMatrixWrite: copyReal100BreakdownItems(breakdown.NeedsRealAreaMatrixWrite),
		AreaFlowOnlyCanContinue:  copyReal100BreakdownItems(breakdown.AreaFlowOnlyCanContinue),
		CompletedEvidence:        copyReal100BreakdownItems(breakdown.CompletedEvidence),
	}
}

func copyReal100BreakdownItems(items []Real100BreakdownItem) []Real100BreakdownItem {
	out := make([]Real100BreakdownItem, 0, len(items))
	for _, item := range items {
		item.Blockers = append([]string{}, item.Blockers...)
		item.EvidenceRefs = append([]string{}, item.EvidenceRefs...)
		out = append(out, item)
	}
	return out
}
