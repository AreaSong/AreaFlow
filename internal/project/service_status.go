package project

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type LocalServiceStatusOptions struct {
	APIBaseURL       string
	WebDashboardURL  string
	GeneratedAt      time.Time
	DatabaseStatus   string
	DatabaseMessage  string
	WorkerPoolStatus string
}

type LocalServiceComponentStatus struct {
	Status  string
	Message string
}

type LocalServiceWorkerPoolStatus struct {
	Status             string
	Message            string
	TotalProjects      int64
	TotalWorkers       int64
	TotalOnlineWorkers int64
	TotalActiveLeases  int64
	TotalQueuedTasks   int64
	TotalNeedsRecovery int64
}

type LocalServiceDashboardStatus struct {
	URL     string
	APIURL  string
	Status  string
	Message string
}

type LocalServiceStatus struct {
	Status           string
	Mode             string
	API              LocalServiceComponentStatus
	Database         LocalServiceComponentStatus
	WorkerPool       LocalServiceWorkerPoolStatus
	Dashboard        LocalServiceDashboardStatus
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) LocalServiceStatus(ctx context.Context, options LocalServiceStatusOptions) (LocalServiceStatus, error) {
	options = normalizeLocalServiceStatusOptions(options)
	status := baseLocalServiceStatus(options)
	if err := s.pool.Ping(ctx); err != nil {
		status.Status = "blocked"
		status.Database = LocalServiceComponentStatus{
			Status:  "blocked",
			Message: fmt.Sprintf("postgres ping failed: %v", err),
		}
		return status, nil
	}

	summary, err := s.WorkerPoolSummary(ctx)
	if err != nil {
		status.Status = "blocked"
		status.WorkerPool = LocalServiceWorkerPoolStatus{
			Status:  "blocked",
			Message: fmt.Sprintf("worker pool summary failed: %v", err),
		}
		return status, nil
	}

	status.WorkerPool = localServiceWorkerPoolStatus(summary)
	if status.WorkerPool.Status == "warn" {
		status.Status = "warn"
	}
	return status, nil
}

func baseLocalServiceStatus(options LocalServiceStatusOptions) LocalServiceStatus {
	return LocalServiceStatus{
		Status: "ready",
		Mode:   "local_service",
		API: LocalServiceComponentStatus{
			Status:  "ready",
			Message: "AreaFlow API is available",
		},
		Database: LocalServiceComponentStatus{
			Status:  "ready",
			Message: "PostgreSQL connection is healthy",
		},
		WorkerPool: LocalServiceWorkerPoolStatus{
			Status:  "ready",
			Message: "worker pool summary is available",
		},
		Dashboard: LocalServiceDashboardStatus{
			URL:     options.WebDashboardURL,
			APIURL:  options.APIBaseURL,
			Status:  "ready",
			Message: "dashboard should use AreaFlow API as source of truth",
		},
		Capabilities: []string{
			"observe_api",
			"observe_database",
			"observe_worker_pool",
			"open_web_dashboard",
			"stream_events",
		},
		ForbiddenActions: []string{
			"write_project_files",
			"run_workflow_directly",
			"maintain_second_database",
			"bypass_areaflow_api",
		},
		GeneratedAt: options.GeneratedAt,
	}
}

func localServiceWorkerPoolStatus(summary WorkerPoolSummary) LocalServiceWorkerPoolStatus {
	status := LocalServiceWorkerPoolStatus{
		Status:             "ready",
		Message:            "worker pool summary is available",
		TotalProjects:      summary.TotalProjects,
		TotalWorkers:       summary.TotalWorkers,
		TotalOnlineWorkers: summary.TotalOnlineWorkers,
		TotalActiveLeases:  summary.TotalActiveLeases,
		TotalQueuedTasks:   summary.TotalQueuedTasks,
		TotalNeedsRecovery: summary.TotalNeedsRecovery,
	}
	if summary.TotalNeedsRecovery > 0 {
		status.Status = "warn"
		status.Message = "worker pool has recovery items"
	}
	return status
}

func normalizeLocalServiceStatusOptions(options LocalServiceStatusOptions) LocalServiceStatusOptions {
	options.APIBaseURL = strings.TrimSpace(options.APIBaseURL)
	options.WebDashboardURL = strings.TrimSpace(options.WebDashboardURL)
	if options.APIBaseURL == "" {
		options.APIBaseURL = "/api/v1"
	}
	if options.WebDashboardURL == "" {
		options.WebDashboardURL = "http://127.0.0.1:5174"
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}
