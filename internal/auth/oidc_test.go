package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"testing"
	"time"
)

func TestOIDCStateSealRoundTrip(t *testing.T) {
	manager := &OIDCManager{cipher: testOIDCCipher(t)}
	want := oidcState{
		State: "state", Nonce: "nonce", Verifier: "verifier", ReturnTo: "/projects?project=area",
		ExpiresAt: time.Now().UTC().Add(time.Minute).Truncate(time.Second),
	}
	sealed, err := manager.seal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got oidcState
	if err := manager.open(sealed, &got); err != nil {
		t.Fatal(err)
	}
	if got.State != want.State || got.Nonce != want.Nonce || got.Verifier != want.Verifier || got.ReturnTo != want.ReturnTo || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("OIDC state round trip mismatch: %+v", got)
	}
	if err := manager.open(sealed+"tampered", &got); err == nil {
		t.Fatal("tampered OIDC state must fail closed")
	}
}

func TestSafeReturnToRejectsExternalRedirect(t *testing.T) {
	for _, value := range []string{"", "https://evil.example", "//evil.example", "projects", `/\\evil.example`, "/access\r\nLocation:https://evil.example"} {
		if got := safeReturnTo(value); got != "/" {
			t.Fatalf("safeReturnTo(%q) = %q, want /", value, got)
		}
	}
	if got := safeReturnTo("/access?project=area"); got != "/access?project=area" {
		t.Fatalf("safe relative return_to rejected: %q", got)
	}
}

func TestStringSliceClaimNormalizesGroups(t *testing.T) {
	got := stringSliceClaim([]any{"operators", " operators ", 42, "auditors"})
	if len(got) != 2 || got[0] != "auditors" || got[1] != "operators" {
		t.Fatalf("groups = %#v", got)
	}
}

func testOIDCCipher(t *testing.T) cipher.AEAD {
	t.Helper()
	key := sha256.Sum256([]byte("test-only-session-secret-at-least-32-bytes"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return aead
}
