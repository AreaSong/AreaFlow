package workflow

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAreaMatrixProfile(t *testing.T) {
	loaded, err := LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile failed: %v", err)
	}
	if loaded.Profile.ProfileID != "areamatrix" || loaded.Profile.ProfileVersion != 0 {
		t.Fatalf("unexpected profile identity: %+v", loaded.Profile)
	}
	if loaded.SHA256 == "" || len(loaded.SHA256) != 64 {
		t.Fatalf("unexpected profile hash: %q", loaded.SHA256)
	}
	if len(loaded.Profile.Stages) != 16 {
		t.Fatalf("stage count = %d, want 16", len(loaded.Profile.Stages))
	}
	if len(loaded.Profile.Gates) != 17 {
		t.Fatalf("gate count = %d, want 17", len(loaded.Profile.Gates))
	}
	if !loaded.Profile.VersionBinding.FreezeProfileHash {
		t.Fatal("expected profile hash freeze policy")
	}
}

func TestListBuiltInProfiles(t *testing.T) {
	profiles, err := ListBuiltInProfiles(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("list profiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profile count = %d, want 1", len(profiles))
	}
	loaded := profiles[0]
	if loaded.Profile.ProfileID != "areamatrix" {
		t.Fatalf("profile id = %q, want areamatrix", loaded.Profile.ProfileID)
	}
	if len(loaded.Profile.Stages) != 16 || len(loaded.Profile.Gates) != 17 {
		t.Fatalf("unexpected profile shape: stages=%d gates=%d", len(loaded.Profile.Stages), len(loaded.Profile.Gates))
	}
}

func TestValidateProfileRejectsUnknownTransitionGate(t *testing.T) {
	profile := Profile{
		ProfileID:      "test",
		ProfileVersion: 1,
		ItemStates:     []string{"draft"},
		Stages: []Stage{
			{Name: "one", RequiredArtifacts: []string{"a"}},
			{Name: "two", RequiredArtifacts: []string{"b"}},
		},
		Gates: []Gate{{Name: "known", StatusSource: "gate_results"}},
		Transitions: []Transition{{
			From:         "one",
			To:           "two",
			RequiredGate: "missing",
		}},
		Permissions: safePermissionPolicy(),
	}
	_, err := ValidateProfile(profile)
	if err == nil {
		t.Fatal("expected invalid profile")
	}
	if !errors.Is(err, ErrInvalidProfile) || !strings.Contains(err.Error(), "unknown gate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProfileRejectsUnsafePermissionPolicy(t *testing.T) {
	profile := Profile{
		ProfileID:      "test",
		ProfileVersion: 1,
		ItemStates:     []string{"draft"},
		Stages: []Stage{
			{Name: "one", RequiredArtifacts: []string{"a"}},
		},
		Gates:       []Gate{{Name: "known", StatusSource: "gate_results"}},
		Permissions: PermissionPolicy{DefaultMode: "write"},
	}
	_, err := ValidateProfile(profile)
	if err == nil {
		t.Fatal("expected invalid profile")
	}
	for _, want := range []string{
		"permissions.default_mode must be readonly",
		"permissions.write_requires missing required guard: capability",
		"permissions.write_requires missing required guard: audit_event",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want containing %q", err, want)
		}
	}
}

func safePermissionPolicy() PermissionPolicy {
	return PermissionPolicy{
		DefaultMode: "readonly",
		WriteRequires: []string{
			"capability",
			"path_allowlist",
			"gate_result",
			"approval_record",
			"audit_event",
		},
	}
}
