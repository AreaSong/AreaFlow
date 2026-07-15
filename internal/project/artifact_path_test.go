package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteProjectArtifactUsesProjectNamespace(t *testing.T) {
	root := t.TempDir()
	stored, err := writeProjectArtifact(
		Record{Key: "areamatrix", ArtifactRoot: root},
		filepath.Join("v2", "runs", "report.json"),
		[]byte(`{"ok":true}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("write project artifact failed: %v", err)
	}

	want := filepath.Join(root, "areamatrix", "v2", "runs", "report.json")
	if stored.URI != want {
		t.Fatalf("uri = %q, want %q", stored.URI, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("artifact was not written under project namespace: %v", err)
	}
}

func TestWriteProjectArtifactRejectsEscapingPath(t *testing.T) {
	_, err := writeProjectArtifact(
		Record{Key: "areamatrix", ArtifactRoot: t.TempDir()},
		filepath.Join("..", "other", "report.json"),
		[]byte("{}"),
		"application/json",
	)
	if err == nil {
		t.Fatal("expected escaping artifact path to fail")
	}
}
