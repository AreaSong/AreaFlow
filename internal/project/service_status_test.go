package project

import (
	"testing"
	"time"
)

func TestBaseLocalServiceStatus(t *testing.T) {
	generated := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	status := baseLocalServiceStatus(LocalServiceStatusOptions{
		APIBaseURL:      "http://127.0.0.1:3847/api/v1",
		WebDashboardURL: "http://127.0.0.1:5174",
		GeneratedAt:     generated,
	})

	if status.Status != "ready" || status.Mode != "local_service" {
		t.Fatalf("unexpected local service status: %+v", status)
	}
	if status.API.Status != "ready" || status.Database.Status != "ready" || status.Dashboard.Status != "ready" {
		t.Fatalf("unexpected component status: %+v", status)
	}
	if status.Dashboard.APIURL != "http://127.0.0.1:3847/api/v1" || status.Dashboard.URL != "http://127.0.0.1:5174" {
		t.Fatalf("unexpected dashboard urls: %+v", status.Dashboard)
	}
	if !containsString(status.Capabilities, "open_web_dashboard") {
		t.Fatalf("missing desktop capability: %+v", status.Capabilities)
	}
	if !containsString(status.ForbiddenActions, "maintain_second_database") {
		t.Fatalf("missing desktop forbidden action: %+v", status.ForbiddenActions)
	}
	if !status.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", status.GeneratedAt, generated)
	}
}

func TestLocalServiceWorkerPoolStatusWarnsOnRecovery(t *testing.T) {
	status := localServiceWorkerPoolStatus(WorkerPoolSummary{
		TotalProjects:      2,
		TotalWorkers:       3,
		TotalOnlineWorkers: 2,
		TotalQueuedTasks:   5,
		TotalNeedsRecovery: 1,
	})

	if status.Status != "warn" || status.Message != "worker pool has recovery items" {
		t.Fatalf("unexpected recovery status: %+v", status)
	}
	if status.TotalProjects != 2 || status.TotalQueuedTasks != 5 || status.TotalNeedsRecovery != 1 {
		t.Fatalf("unexpected worker pool totals: %+v", status)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
