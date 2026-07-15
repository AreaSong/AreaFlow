package auth

import "testing"

func TestOpaqueSessionFormat(t *testing.T) {
	key, secret, err := generateOpaqueParts()
	if err != nil {
		t.Fatal(err)
	}
	raw := "afs_" + key + "_" + secret
	parsed, err := opaqueKey(raw, "afs")
	if err != nil || parsed != key {
		t.Fatalf("opaqueKey = %q, %v", parsed, err)
	}
	if len(opaqueHash(raw)) != 64 {
		t.Fatal("session hash must be SHA-256 hex")
	}
}

func TestInvalidOpaqueSession(t *testing.T) {
	for _, raw := range []string{"", "afs_short_value", "token", "afs_zzzzzzzzzzzzzzzzzzzzzzzz_secret"} {
		if _, err := opaqueKey(raw, "afs"); err == nil {
			t.Fatalf("expected invalid session %q", raw)
		}
	}
}
