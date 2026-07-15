package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type queryObservation struct {
	operation  string
	err        error
	duration   time.Duration
	auditWrite bool
}

type queryObserverRecorder struct{ observations []queryObservation }

func (r *queryObserverRecorder) ObserveDatabaseQuery(operation string, err error, duration time.Duration, auditWrite bool) {
	r.observations = append(r.observations, queryObservation{operation: operation, err: err, duration: duration, auditWrite: auditWrite})
}

func TestQueryTracerRecognizesAuditInsert(t *testing.T) {
	recorder := &queryObserverRecorder{}
	tracer := queryTracer{observer: recorder}
	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "INSERT INTO audit_events (action) VALUES ($1)"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
	if len(recorder.observations) != 1 {
		t.Fatalf("observations = %d", len(recorder.observations))
	}
	observation := recorder.observations[0]
	if observation.operation != "insert" || !observation.auditWrite || observation.duration < 0 {
		t.Fatalf("unexpected observation: %+v", observation)
	}
}
