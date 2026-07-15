package auth

import "testing"

func TestRoleCapabilities(t *testing.T) {
	capabilities := CapabilitiesForRoles([]string{RoleViewer, RoleApprover})
	for _, expected := range []string{"read", "workflow.approval.record"} {
		if !contains(capabilities, expected) {
			t.Fatalf("missing capability %s: %v", expected, capabilities)
		}
	}
	if !ValidRole(RolePlatformAdmin) || ValidRole("owner") {
		t.Fatal("role validation mismatch")
	}
}

func TestSafeReturnTo(t *testing.T) {
	for input, expected := range map[string]string{
		"/projects":            "/projects",
		"https://evil.example": "/",
		"//evil.example":       "/",
		"":                     "/",
	} {
		if got := safeReturnTo(input); got != expected {
			t.Fatalf("safeReturnTo(%q) = %q, want %q", input, got, expected)
		}
	}
}
