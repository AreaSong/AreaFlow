package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/areasong/areaflow/internal/config"
)

type QueryObserver interface {
	ObserveDatabaseQuery(operation string, err error, duration time.Duration, auditWrite bool)
}

func Open(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	return open(ctx, cfg, nil)
}

func OpenObserved(ctx context.Context, cfg config.DatabaseConfig, observer QueryObserver) (*pgxpool.Pool, error) {
	return open(ctx, cfg, observer)
}

func open(ctx context.Context, cfg config.DatabaseConfig, observer QueryObserver) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres pool configuration: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnIdleTime = cfg.MaxConnectionIdle
	poolConfig.MaxConnLifetime = cfg.MaxConnectionLifetime
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(cfg.QueryTimeout.Milliseconds(), 10)
	if observer != nil {
		poolConfig.ConnConfig.Tracer = queryTracer{observer: observer}
	}

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	pingCtx, pingCancel := context.WithTimeout(ctx, cfg.AcquireTimeout)
	defer pingCancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

type queryTracer struct {
	observer QueryObserver
}

type queryTrace struct {
	started    time.Time
	operation  string
	auditWrite bool
}

type queryTraceKey struct{}

func (t queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	normalized := strings.ToLower(strings.Join(strings.Fields(data.SQL), " "))
	operation := "unknown"
	if fields := strings.Fields(normalized); len(fields) > 0 {
		operation = fields[0]
	}
	return context.WithValue(ctx, queryTraceKey{}, queryTrace{
		started: time.Now(), operation: operation, auditWrite: strings.Contains(normalized, "insert into audit_events"),
	})
}

func (t queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	trace, ok := ctx.Value(queryTraceKey{}).(queryTrace)
	if !ok || t.observer == nil {
		return
	}
	t.observer.ObserveDatabaseQuery(trace.operation, data.Err, time.Since(trace.started), trace.auditWrite && data.Err == nil)
}
