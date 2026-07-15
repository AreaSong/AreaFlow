package areamatrix

import "testing"

func TestLoadRealAreaMatrixSnapshot(t *testing.T) {
	root := "/Users/as/Ai-Project/project/AreaMatrix"
	snapshot, err := Load(root)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}

	if snapshot.TaskSummary.ActiveCount != 0 {
		t.Fatalf("active count = %d, want 0", snapshot.TaskSummary.ActiveCount)
	}
	if snapshot.TaskSummary.BacklogPackages != 5 {
		t.Fatalf("backlog packages = %d, want 5", snapshot.TaskSummary.BacklogPackages)
	}
	if snapshot.TaskSummary.BacklogClosed != 5 {
		t.Fatalf("backlog closed = %d, want 5", snapshot.TaskSummary.BacklogClosed)
	}
	if snapshot.TaskSummary.BacklogOpen != 0 {
		t.Fatalf("backlog open = %d, want 0", snapshot.TaskSummary.BacklogOpen)
	}
	if snapshot.TaskSummary.V1ExecutionTotal != 637 {
		t.Fatalf("v1 execution total = %d, want 637", snapshot.TaskSummary.V1ExecutionTotal)
	}
	if snapshot.TaskSummary.V1ExecutionDone != 637 {
		t.Fatalf("v1 execution done = %d, want 637", snapshot.TaskSummary.V1ExecutionDone)
	}
	if len(snapshot.Residuals) < 10 {
		t.Fatalf("residual count = %d, want at least the established 10 records", len(snapshot.Residuals))
	}
	if len(snapshot.Versions) < 2 {
		t.Fatalf("version count = %d, want at least 2", len(snapshot.Versions))
	}

	v1 := findVersion(snapshot.Versions, "v1-mvp")
	if v1.Label == "" {
		t.Fatal("missing v1-mvp version")
	}
	if v1.Lifecycle != "mixed-blocked" {
		t.Fatalf("v1 lifecycle = %q, want mixed-blocked", v1.Lifecycle)
	}
	if got := v1.StatusSummary["technical_queue"]; got != "complete" {
		t.Fatalf("v1 technical_queue = %v, want complete", got)
	}
	if got := v1.StatusSummary["formal_alpha"]; got != "blocked" {
		t.Fatalf("v1 formal_alpha = %v, want blocked", got)
	}
}

func findVersion(versions []Version, label string) Version {
	for _, version := range versions {
		if version.Label == label {
			return version
		}
	}
	return Version{}
}
