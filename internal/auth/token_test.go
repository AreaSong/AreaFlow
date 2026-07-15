package auth

import "testing"

func TestTokenFormatAndPrincipalScope(t *testing.T) {
	key, secret, err := generateTokenParts()
	if err != nil {
		t.Fatal(err)
	}
	raw := "af_" + key + "_" + secret
	parsed, err := tokenKeyFromRaw(raw)
	if err != nil || parsed != key {
		t.Fatalf("parse token key = %q, %v", parsed, err)
	}
	if len(tokenHash(raw)) != 64 {
		t.Fatal("token hash must be SHA-256 hex")
	}
	principal := Principal{Projects: []string{"area"}, Capabilities: []string{"read"}}
	if !principal.AllowsProject("area") || principal.AllowsProject("other") {
		t.Fatal("project scope evaluation failed")
	}
	if !principal.AllowsCapability("read") || principal.AllowsCapability("admin") {
		t.Fatal("capability evaluation failed")
	}
}

func TestInvalidTokenFormat(t *testing.T) {
	for _, raw := range []string{
		"",
		"secret",
		"af_short_value",
		"af_zzzzzzzzzzzzzzzzzzzzzzzz_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"af_012345678901234567890123_not-base64!",
		"other_012345678901234567890123_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		if _, err := tokenKeyFromRaw(raw); err == nil {
			t.Fatalf("expected invalid token: %q", raw)
		}
	}
}
