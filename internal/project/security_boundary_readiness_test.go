package project

import (
	"testing"
	"time"
)

func TestBuildSecurityBoundaryReadiness(t *testing.T) {
	generated := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})

	if readiness.Status != "ready" || readiness.Mode != "read_only_security_boundary_readiness" {
		t.Fatalf("unexpected security boundary readiness: %+v", readiness)
	}
	if !readiness.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", readiness.GeneratedAt, generated)
	}
	if len(readiness.Items) < 8 {
		t.Fatalf("expected security readiness items, got %+v", readiness.Items)
	}
	assertSecurityBoundaryItem(t, readiness, "auth_enforcement", "ready")
	assertSecurityBoundaryItem(t, readiness, "team_permission_enforcement", "ready")
	assertSecurityBoundaryItem(t, readiness, "api_token_lifecycle", "ready")
	assertSecurityBoundaryItem(t, readiness, "secret_resolve", "ready")
	assertSecurityBoundaryItem(t, readiness, "remote_worker_credentials", "ready")
	assertSecurityBoundaryItem(t, readiness, "managed_ops_boundary", "ready")

	if readiness.AuthEnforcementOpen ||
		readiness.TeamPermissionEnforcementOpen ||
		readiness.APITokenIssuanceOpen ||
		readiness.APITokenEnforcementOpen ||
		readiness.SecretResolveOpen ||
		readiness.RemoteWorkerCredentialsOpen ||
		readiness.BudgetEnforcementOpen ||
		readiness.QuotaDecrementOpen ||
		readiness.UsageChargeWritten ||
		readiness.WebhookDeliveryOpen ||
		readiness.InboundCallbackOpen ||
		readiness.ExternalAPICallOpen ||
		readiness.AuthorizationChanged ||
		readiness.SecretPlaintextRead ||
		readiness.RemoteWorkerDirectPGAllowed ||
		readiness.TeamConsoleCommandOpen ||
		readiness.RemoteOpsControlOpen ||
		readiness.ManagedUpgradeOpen ||
		readiness.SupportBundleExportOpen ||
		readiness.DefaultRemoteTelemetryOpen {
		t.Fatalf("security readiness opened a forbidden capability: %+v", readiness)
	}
	if !containsString(readiness.Capabilities, "read_security_boundary") {
		t.Fatalf("missing read capability: %+v", readiness.Capabilities)
	}
	if !containsString(readiness.ForbiddenActions, "resolve_secret_plaintext") ||
		!containsString(readiness.ForbiddenActions, "issue_remote_worker_credential") ||
		!containsString(readiness.ForbiddenActions, "change_api_authorization") {
		t.Fatalf("missing forbidden security actions: %+v", readiness.ForbiddenActions)
	}
}

func assertSecurityBoundaryItem(t *testing.T, readiness SecurityBoundaryReadiness, key string, status string) {
	t.Helper()
	for _, item := range readiness.Items {
		if item.Key == key {
			if item.Status != status {
				t.Fatalf("item %s status = %q, want %q: %+v", key, item.Status, status, item)
			}
			if len(item.RequiredEvidence) == 0 {
				t.Fatalf("item %s missing required evidence: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("security boundary item %s not found: %+v", key, readiness.Items)
}
