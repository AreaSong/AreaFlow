package project

import (
	"fmt"
	"strings"
	"time"
)

func requireProofEvidenceForStatus(proofName string, status string, summary string, evidenceURI string, statuses ...string) error {
	requiresEvidence := false
	for _, candidate := range statuses {
		if status == candidate {
			requiresEvidence = true
			break
		}
	}
	if !requiresEvidence {
		return nil
	}

	missing := []string{}
	if strings.TrimSpace(summary) == "" {
		missing = append(missing, "summary")
	}
	if strings.TrimSpace(evidenceURI) == "" {
		missing = append(missing, "evidence_uri")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s proof status %q missing required evidence fields: %s", proofName, status, strings.Join(missing, ","))
	}
	return nil
}

func proofMetadataHasTraceableEvidence(metadata map[string]any) bool {
	return strings.TrimSpace(metadataString(metadata, "summary")) != "" &&
		strings.TrimSpace(metadataString(metadata, "evidence_uri")) != ""
}

func proofReleaseCandidateEvidenceURIBlockers(prefix string, evidenceURI string) []string {
	evidenceURI = strings.TrimSpace(evidenceURI)
	blockers := []string{}
	if evidenceURI == "" {
		blockers = append(blockers, prefix+"_evidence_uri_missing")
		return blockers
	}
	if completionAuditSnapshotContainsFixtureMarker(evidenceURI) {
		blockers = append(blockers, prefix+"_evidence_uri_non_release_marker")
	}
	if completionAuditSnapshotUsesMechanismEvidenceURI(evidenceURI) {
		blockers = append(blockers, prefix+"_evidence_uri_mechanism")
	}
	if !completionAuditSnapshotUsesReleaseCandidateEvidenceURI(evidenceURI) {
		blockers = append(blockers, prefix+"_evidence_uri_not_release_candidate")
	}
	return uniqueStrings(blockers)
}

func proofReviewMetadataFromFields(reviewDecision string, reviewedBy string, reviewedAt time.Time, metadata map[string]any) map[string]any {
	reviewedAtValue := strings.TrimSpace(metadataString(metadata, "reviewed_at"))
	if !reviewedAt.IsZero() {
		reviewedAtValue = reviewedAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"review_decision": strings.ToLower(strings.TrimSpace(firstNonEmptyString(reviewDecision, metadataString(metadata, "review_decision")))),
		"reviewed_by":     strings.TrimSpace(firstNonEmptyString(reviewedBy, metadataString(metadata, "reviewed_by"))),
		"reviewed_at":     reviewedAtValue,
	}
}

func proofReviewMetadataFieldBlockers(prefix string, metadata map[string]any) []string {
	blockers := []string{}
	decision := strings.ToLower(strings.TrimSpace(metadataString(metadata, "review_decision")))
	if decision == "" {
		blockers = append(blockers, prefix+"_review_decision_missing")
	} else if decision != "approved" {
		blockers = append(blockers, prefix+"_review_decision_not_approved")
	}
	if strings.TrimSpace(metadataString(metadata, "reviewed_by")) == "" {
		blockers = append(blockers, prefix+"_reviewed_by_missing")
	}
	reviewedAt := strings.TrimSpace(metadataString(metadata, "reviewed_at"))
	if reviewedAt == "" {
		blockers = append(blockers, prefix+"_reviewed_at_missing")
	} else if parsed, err := time.Parse(time.RFC3339, reviewedAt); err != nil || parsed.IsZero() {
		blockers = append(blockers, prefix+"_reviewed_at_invalid")
	}
	return uniqueStrings(blockers)
}

func proofReviewMetadataBlockers(prefix string, metadata map[string]any) []string {
	blockers := proofReviewMetadataFieldBlockers(prefix, metadata)
	if metadataString(metadata, "review_metadata_status") != "approved" {
		blockers = append(blockers, prefix+"_review_metadata_status_not_approved")
	}
	return uniqueStrings(blockers)
}

func proofCompleteReviewEvidenceBlockers(prefix string, metadata map[string]any) []string {
	blockers := proofReleaseCandidateEvidenceURIBlockers(prefix, metadataString(metadata, "evidence_uri"))
	blockers = append(blockers, proofReviewMetadataBlockers(prefix, metadata)...)
	return uniqueStrings(blockers)
}

func proofMetadataHasApprovedReviewEvidence(prefix string, metadata map[string]any) bool {
	return len(proofCompleteReviewEvidenceBlockers(prefix, metadata)) == 0
}

func requireCompleteProofReviewEvidence(proofName string, prefix string, status string, evidenceURI string, reviewMetadata map[string]any) error {
	if status != "complete" {
		return nil
	}
	blockers := proofReleaseCandidateEvidenceURIBlockers(prefix, evidenceURI)
	blockers = append(blockers, proofReviewMetadataFieldBlockers(prefix, reviewMetadata)...)
	if len(blockers) > 0 {
		return fmt.Errorf("complete %s proof requires approved release-candidate review evidence: %s", proofName, strings.Join(uniqueStrings(blockers), ","))
	}
	return nil
}

func addProofReviewMetadata(metadata map[string]any, proofStatus string, prefix string, reviewMetadata map[string]any) {
	for _, key := range []string{"review_decision", "reviewed_by", "reviewed_at"} {
		value := strings.TrimSpace(metadataString(reviewMetadata, key))
		if value != "" || proofStatus == "complete" {
			metadata[key] = value
		}
	}
	if proofStatus != "complete" {
		metadata["review_metadata_status"] = "not_required"
		metadata["review_metadata_blockers"] = []string{}
		return
	}
	blockers := proofReviewMetadataFieldBlockers(prefix, metadata)
	metadata["review_metadata_blockers"] = blockers
	if len(blockers) == 0 {
		metadata["review_metadata_status"] = "approved"
	} else {
		metadata["review_metadata_status"] = "fail"
	}
}

func proofMetadataFromEventMetadata(eventMetadata map[string]any) map[string]any {
	metadata := map[string]any{}
	if raw, ok := eventMetadata["metadata"].(map[string]any); ok {
		for key, value := range raw {
			metadata[key] = value
		}
	}
	for key, value := range eventMetadata {
		metadata[key] = value
	}
	return metadata
}
