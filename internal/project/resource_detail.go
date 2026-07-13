package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrRunTaskNotFound    = errors.New("run task not found")
	ErrRunAttemptNotFound = errors.New("run attempt not found")
)

type WorkerHeartbeatRecord struct {
	ID         int64
	ProjectID  int64
	WorkerID   int64
	Status     string
	ObservedAt time.Time
	Metadata   map[string]any
}

type WorkerDetail struct {
	Worker     WorkerRecord
	Heartbeats []WorkerHeartbeatRecord
	Leases     []LeaseRecord
}

func (s Store) GetWorker(ctx context.Context, workerID int64, projectID int64, historyLimit int) (WorkerDetail, error) {
	if workerID <= 0 {
		return WorkerDetail{}, fmt.Errorf("worker id is required")
	}
	if historyLimit <= 0 {
		historyLimit = 50
	}
	query := `SELECT id, project_id, COALESCE(actor_id, 0), worker_key, worker_type, status,
       COALESCE(hostname, ''), COALESCE(pid, 0), capabilities, metadata,
       registered_at, last_heartbeat_at, heartbeat_interval_seconds, lease_timeout_seconds, updated_at
FROM workers WHERE id = $1`
	args := []any{workerID}
	if projectID > 0 {
		query += " AND project_id = $2"
		args = append(args, projectID)
	}
	worker, err := scanWorker(s.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkerDetail{}, ErrWorkerNotFound
		}
		return WorkerDetail{}, fmt.Errorf("load worker: %w", err)
	}
	heartbeats, err := s.ListWorkerHeartbeats(ctx, worker.ID, worker.ProjectID, historyLimit)
	if err != nil {
		return WorkerDetail{}, err
	}
	leases, err := s.ListWorkerLeases(ctx, worker.ID, worker.ProjectID, historyLimit)
	if err != nil {
		return WorkerDetail{}, err
	}
	return WorkerDetail{Worker: worker, Heartbeats: heartbeats, Leases: leases}, nil
}

func (s Store) ListWorkerHeartbeats(ctx context.Context, workerID int64, projectID int64, limit int) ([]WorkerHeartbeatRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, worker_id, status, observed_at, metadata
FROM worker_heartbeats WHERE worker_id = $1 AND ($2 = 0 OR project_id = $2)
ORDER BY observed_at DESC, id DESC LIMIT $3`, workerID, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list worker heartbeats: %w", err)
	}
	defer rows.Close()
	result := []WorkerHeartbeatRecord{}
	for rows.Next() {
		var item WorkerHeartbeatRecord
		var raw []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.WorkerID, &item.Status, &item.ObservedAt, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Metadata); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Store) ListWorkerLeases(ctx context.Context, workerID int64, projectID int64, limit int) ([]LeaseRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, COALESCE(run_id, 0), COALESCE(run_task_id, 0),
       COALESCE(workflow_item_id, 0), COALESCE(worker_id, 0), lease_kind, status,
       acquired_at, expires_at, heartbeat_at, released_at, allowed_capabilities, scope, metadata
FROM leases WHERE worker_id = $1 AND ($2 = 0 OR project_id = $2)
ORDER BY acquired_at DESC, id DESC LIMIT $3`, workerID, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list worker leases: %w", err)
	}
	defer rows.Close()
	result := []LeaseRecord{}
	for rows.Next() {
		lease, err := scanLease(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, lease)
	}
	return result, rows.Err()
}

func (s Store) ListRunTasks(ctx context.Context, runID int64) ([]RunTaskRecord, error) {
	if _, err := s.GetRun(ctx, runID); err != nil {
		return nil, err
	}
	return s.listRunTasks(ctx, runID)
}

func (s Store) GetRunTask(ctx context.Context, runID int64, taskID int64) (RunTaskRecord, error) {
	if runID <= 0 || taskID <= 0 {
		return RunTaskRecord{}, fmt.Errorf("run id and task id are required")
	}
	task, err := scanRunTask(s.pool.QueryRow(ctx, `SELECT id, project_id, COALESCE(workflow_version_id, 0),
       COALESCE(workflow_item_id, 0), run_id, task_key, task_kind, status, risk_level,
       sequence, metadata, created_at, updated_at FROM run_tasks WHERE run_id = $1 AND id = $2`, runID, taskID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunTaskRecord{}, ErrRunTaskNotFound
		}
		return RunTaskRecord{}, fmt.Errorf("load run task: %w", err)
	}
	return task, nil
}

func (s Store) ListRunAttempts(ctx context.Context, runID int64) ([]RunAttemptRecord, error) {
	if _, err := s.GetRun(ctx, runID); err != nil {
		return nil, err
	}
	return s.listRunAttempts(ctx, runID)
}

func (s Store) GetRunAttempt(ctx context.Context, runID int64, attemptID int64) (RunAttemptRecord, error) {
	if runID <= 0 || attemptID <= 0 {
		return RunAttemptRecord{}, fmt.Errorf("run id and attempt id are required")
	}
	attempt, err := scanRunAttempt(s.pool.QueryRow(ctx, `SELECT id, project_id, COALESCE(workflow_version_id, 0),
       COALESCE(workflow_item_id, 0), run_id, COALESCE(run_task_id, 0), attempt_kind,
       status, dry_run, metadata, started_at, finished_at FROM run_attempts WHERE run_id = $1 AND id = $2`, runID, attemptID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunAttemptRecord{}, ErrRunAttemptNotFound
		}
		return RunAttemptRecord{}, fmt.Errorf("load run attempt: %w", err)
	}
	return attempt, nil
}
