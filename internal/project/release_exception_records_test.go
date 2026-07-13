package project

import (
	"context"
	"testing"
	"time"
)

func TestValidateReleaseExceptionTransition(t *testing.T) {
	for _, test := range []struct {
		current string
		target  string
		valid   bool
	}{
		{current: "requested", target: "approved", valid: true},
		{current: "requested", target: "revoked", valid: true},
		{current: "approved", target: "revoked", valid: true},
		{current: "revoked", target: "approved", valid: false},
		{current: "approved", target: "approved", valid: false},
	} {
		err := validateReleaseExceptionTransition(test.current, test.target)
		if (err == nil) != test.valid {
			t.Fatalf("transition %s -> %s valid=%t err=%v", test.current, test.target, test.valid, err)
		}
	}
}

func TestRequestReleaseExceptionRejectsPastExpiryBeforeDatabaseAccess(t *testing.T) {
	past := time.Now().UTC().Add(-time.Minute)
	_, err := (Store{}).RequestReleaseException(context.Background(), Record{}, RequestReleaseExceptionOptions{
		ExceptionKey: "release_exception:restore_plan", Actor: "owner", Reason: "reviewed", ExpiresAt: &past,
	})
	if err == nil || err.Error() != "release exception expiry must be in the future" {
		t.Fatalf("unexpected past expiry result: %v", err)
	}
}

func TestMatchingEffectiveReleaseExceptionRequiresExactGateAndType(t *testing.T) {
	decision := ReleaseAcceptanceDecision{Key: "accept:restore_plan", AcceptanceType: "metadata_only_history"}
	exceptions := []ReleaseExceptionRecord{
		{ID: 1, Status: "approved", SourceGateItem: "gate:accept:restore_plan", AcceptanceType: "future_only_gap", Metadata: map[string]any{"source_fingerprint": "mismatch"}},
		{ID: 2, Status: "approved", SourceGateItem: "gate:accept:restore_plan", AcceptanceType: "metadata_only_history", Metadata: map[string]any{
			"source_fingerprint": releaseExceptionSourceFingerprint("gate:accept:restore_plan", "", "metadata_only_history", "", nil),
		}},
	}
	match := matchingEffectiveReleaseException(decision, exceptions)
	if match == nil || match.ID != 2 {
		t.Fatalf("unexpected effective exception match: %+v", match)
	}
}

func TestMatchingEffectiveReleaseExceptionRejectsStaleFingerprint(t *testing.T) {
	decision := ReleaseAcceptanceDecision{
		Key: "accept:restore_plan", Category: "restore", AcceptanceType: "metadata_only_history",
		Owner: "release_owner", RequiredEvidence: []string{"current evidence"},
	}
	exceptions := []ReleaseExceptionRecord{{
		Status: "approved", SourceGateItem: "gate:accept:restore_plan", AcceptanceType: "metadata_only_history",
		Metadata: map[string]any{"source_fingerprint": releaseExceptionSourceFingerprint(
			"gate:accept:restore_plan", "restore", "metadata_only_history", "release_owner", []string{"old evidence"},
		)},
	}}
	if match := matchingEffectiveReleaseException(decision, exceptions); match != nil {
		t.Fatalf("stale exception fingerprint must not match: %+v", match)
	}
}
